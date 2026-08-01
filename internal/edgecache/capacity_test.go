package edgecache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func reducedCapacityStore(t *testing.T, capacity int64) (*Store, ReadAuthority) {
	t.Helper()
	config := testStoreConfig(filepath.Join(t.TempDir(), "edge"), "http://127.0.0.1:8042")
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	store.capacityBytes = capacity
	store.highWatermarkBytes = percentage(capacity, HighWatermarkPercent)
	store.lowWatermarkBytes = percentage(capacity, LowWatermarkPercent)
	store.protectedBytes = percentage(capacity, ProtectedPercent)
	store.maximumObjectBytes = capacity
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close reduced Edge store: %v", err)
		}
	})
	return store, testReadAuthority()
}

func persistSizedEntry(t *testing.T, store *Store, authority ReadAuthority, key string, size int, now time.Time) string {
	t.Helper()
	payload := bytes.Repeat([]byte(key[:1]), size)
	digestBytes := sha256.Sum256(payload)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	err := store.persistFetched(
		context.Background(),
		authority,
		key,
		fetchedObject{
			body:           io.NopCloser(bytes.NewReader(payload)),
			size:           int64(size),
			digest:         digest,
			decisionDigest: "sha256:" + strings.Repeat("c", 64),
		},
		now,
	)
	if err != nil {
		t.Fatalf("persist %s: %v", key, err)
	}
	return digest
}

func TestByteSLRUEvictsProbationBeforeProtectedAtHighWatermark(t *testing.T) {
	store, authority := reducedCapacityStore(t, 1000)
	firstDigest := persistSizedEntry(t, store, authority, "alpha", 300, edgeTestNow)
	persistSizedEntry(t, store, authority, "bravo", 300, edgeTestNow.Add(time.Second))
	persistSizedEntry(t, store, authority, "charlie", 300, edgeTestNow.Add(2*time.Second))

	snapshot, err := store.CapacitySnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.StableBytes != 600 || snapshot.ProbationBytes != 600 ||
		snapshot.ProtectedUsedBytes != 0 || snapshot.Objects != 2 {
		t.Fatalf("post-pressure snapshot = %+v", snapshot)
	}
	if file, err := store.OpenCommitted(context.Background(), authority, "alpha", edgeTestNow.Add(3*time.Second)); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("oldest probation read = %v/%v", file, err)
	}
	firstHex := strings.TrimPrefix(firstDigest, "sha256:")
	if _, err := os.Lstat(filepath.Join(store.blobs, firstHex[:2], firstHex)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("evicted probation blob still exists: %v", err)
	}

	file, err := store.OpenCommitted(context.Background(), authority, "bravo", edgeTestNow.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	persistSizedEntry(t, store, authority, "delta", 300, edgeTestNow.Add(5*time.Second))
	if file, err := store.OpenCommitted(context.Background(), authority, "charlie", edgeTestNow.Add(6*time.Second)); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("probation-before-protected read = %v/%v", file, err)
	}
	if file, err := store.OpenCommitted(context.Background(), authority, "bravo", edgeTestNow.Add(6*time.Second)); err != nil {
		t.Fatalf("protected survivor: %v", err)
	} else {
		_ = file.Close()
	}
}

func TestProtectedTargetDemotesOldestByBytes(t *testing.T) {
	store, authority := reducedCapacityStore(t, 1000)
	store.protectedBytes = 400
	persistSizedEntry(t, store, authority, "alpha", 300, edgeTestNow)
	persistSizedEntry(t, store, authority, "bravo", 300, edgeTestNow.Add(time.Second))
	for index, key := range []string{"alpha", "bravo"} {
		file, err := store.OpenCommitted(context.Background(), authority, key, edgeTestNow.Add(time.Duration(index+2)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		_ = file.Close()
	}
	var alpha, bravo string
	if err := store.database.QueryRow("SELECT segment FROM edge_entries WHERE cache_key = 'alpha'").Scan(&alpha); err != nil {
		t.Fatal(err)
	}
	if err := store.database.QueryRow("SELECT segment FROM edge_entries WHERE cache_key = 'bravo'").Scan(&bravo); err != nil {
		t.Fatal(err)
	}
	if alpha != segmentProbation || bravo != segmentProtected {
		t.Fatalf("segments = alpha:%s bravo:%s", alpha, bravo)
	}
	snapshot, err := store.CapacitySnapshot(context.Background())
	if err != nil || snapshot.ProbationBytes != 300 || snapshot.ProtectedUsedBytes != 300 {
		t.Fatalf("protected snapshot = %+v/%v", snapshot, err)
	}
}

func TestTTLAndMaintenanceDeleteAuthorityBeforePhysicalBlob(t *testing.T) {
	store, authority := reducedCapacityStore(t, 1000)
	store.stableTTL = time.Minute
	digest := persistSizedEntry(t, store, authority, "alpha", 200, edgeTestNow)
	report, err := store.Maintain(context.Background(), edgeTestNow.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if report.ExpiredObjects != 1 || report.StableBytesBefore != 200 ||
		report.StableBytesAfter != 0 || report.DeletedBlobs != 1 {
		t.Fatalf("maintenance report = %+v", report)
	}
	if file, err := store.OpenCommitted(context.Background(), authority, "alpha", edgeTestNow.Add(time.Minute)); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expired read = %v/%v", file, err)
	}
	hexDigest := strings.TrimPrefix(digest, "sha256:")
	if _, err := os.Lstat(filepath.Join(store.blobs, hexDigest[:2], hexDigest)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired blob still exists: %v", err)
	}
}

func TestReservationsAndHardQuotaNeverOversubscribe(t *testing.T) {
	store, authority := reducedCapacityStore(t, 1000)
	start := make(chan struct{})
	type reservationResult struct {
		value *reservation
		err   error
	}
	results := make(chan reservationResult, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			value, err := store.reserve(context.Background(), 600)
			results <- reservationResult{value: value, err: err}
		}()
	}
	ready.Wait()
	close(start)
	var accepted []*reservation
	rejected := 0
	for range 2 {
		result := <-results
		if result.err == nil {
			accepted = append(accepted, result.value)
		} else if errors.Is(result.err, ErrCapacityExceeded) {
			rejected++
		} else {
			t.Fatal(result.err)
		}
	}
	if len(accepted) != 1 || rejected != 1 {
		t.Fatalf("reservations accepted/rejected = %d/%d", len(accepted), rejected)
	}
	snapshot, err := store.CapacitySnapshot(context.Background())
	if err != nil || snapshot.ReservedBytes != 600 {
		t.Fatalf("reserved snapshot = %+v/%v", snapshot, err)
	}
	store.release(accepted[0])
	persistSizedEntry(t, store, authority, "alpha", 800, edgeTestNow)
	if value, err := store.reserve(context.Background(), 201); value != nil || !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("hard-quota reservation = %v/%v", value, err)
	}
}

func TestVersionOneMetadataMigratesToByteSLRU(t *testing.T) {
	root := filepath.Join(t.TempDir(), "edge")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", filepath.Join(root, "edge.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE edge_entries (
    tenant_id TEXT NOT NULL, repository_id TEXT NOT NULL,
    trust_domain TEXT NOT NULL, namespace TEXT NOT NULL,
    namespace_generation INTEGER NOT NULL, cache_key TEXT NOT NULL,
    schema_version TEXT NOT NULL, blob_digest TEXT NOT NULL,
    size_bytes INTEGER NOT NULL, decision_digest TEXT NOT NULL,
    authority_digest TEXT NOT NULL, revocation_epoch INTEGER NOT NULL,
    revocation_digest TEXT NOT NULL, l1_security_generation INTEGER NOT NULL,
    cached_at_unix_ms INTEGER NOT NULL, expires_at_unix_ms INTEGER NOT NULL,
    PRIMARY KEY (tenant_id, repository_id, trust_domain, namespace,
                 namespace_generation, cache_key)) STRICT`,
		"PRAGMA user_version=1",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	config := testStoreConfig(root, "http://127.0.0.1:8042")
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var version int
	if err := store.database.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentEdgeSchemaVersion {
		t.Fatalf("migrated version = %d/%v", version, err)
	}
	if _, err := store.database.Exec("SELECT segment, last_access_unix_ms FROM edge_entries"); err != nil {
		t.Fatalf("migrated SLRU columns: %v", err)
	}
}
