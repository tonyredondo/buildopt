package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCapacityPolicyCannotExpandPrivateBetaBounds(t *testing.T) {
	valid := CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              30 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	}
	testCases := map[string]CapacityPolicy{}
	expandedDeployment := valid
	expandedDeployment.DeploymentBytes = privateBetaDeploymentLimit + 1
	testCases["deployment"] = expandedDeployment
	expandedRepository := valid
	expandedRepository.DeploymentBytes = privateBetaDeploymentLimit
	expandedRepository.RepositoryBytes = privateBetaRepositoryLimit + 1
	testCases["repository"] = expandedRepository
	expandedPending := valid
	expandedPending.PendingQuarantineBytes = 101
	testCases["pending"] = expandedPending
	expandedTTL := valid
	expandedTTL.StableTTL = privateBetaStableTTL + time.Second
	testCases["stable ttl"] = expandedTTL
	expandedQuarantineTTL := valid
	expandedQuarantineTTL.QuarantineTTL =
		privateBetaQuarantineTTL + time.Second
	testCases["quarantine ttl"] = expandedQuarantineTTL

	if err := validateCapacityPolicy(valid, 100); err != nil {
		t.Fatalf("valid reduced policy: %v", err)
	}
	for name, policy := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := validateCapacityPolicy(policy, 100); err == nil {
				t.Fatal("expanded private-beta policy was accepted")
			}
		})
	}
}

func TestPendingAdmissionReservesBeforeReading(t *testing.T) {
	ctx := context.Background()
	storage := openCapacityTestStorage(t, 100, CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              30 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	})
	first := lifecycleAttemptRequest(
		"capacity-first",
		"start-capacity-first",
		"owner-capacity-first",
		1,
	)
	if _, _, err := storage.StartAttempt(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.PutPendingSized(
		ctx,
		first.AttemptID,
		"first-key",
		100,
		bytes.NewReader(bytes.Repeat([]byte{1}, 100)),
	); err != nil {
		t.Fatal(err)
	}

	second := lifecycleAttemptRequest(
		"capacity-second",
		"start-capacity-second",
		"owner-capacity-second",
		2,
	)
	if _, _, err := storage.StartAttempt(ctx, second); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.PutPendingSized(
		ctx,
		second.AttemptID,
		"second-key",
		21,
		&panicReader{},
	); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("pending pool admission = %v, want capacity rejection", err)
	}
	handler, err := NewHTTPHandler(storage, HTTPBinding{
		Tenant:              second.Repository.Tenant,
		NamespaceGeneration: second.NamespaceGeneration,
		PendingAttemptID:    second.AttemptID,
		AllowWrite:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPut,
		"/cache/http-capacity-key",
		&panicReader{},
	)
	request.ContentLength = 21
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("HTTP capacity admission = %d, want 413", response.Code)
	}
	if _, err := storage.PutPending(
		ctx,
		second.AttemptID,
		"unknown-key",
		&panicReader{},
	); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("unknown-length reservation = %v, want capacity rejection", err)
	}
	if _, err := storage.PutPendingSized(
		ctx,
		second.AttemptID,
		"second-key",
		101,
		&panicReader{},
	); !errors.Is(err, ErrBlobTooLarge) {
		t.Fatalf("object admission = %v, want object limit", err)
	}

	snapshot, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PendingBytes != 100 ||
		snapshot.ReservedBytes != 0 ||
		!snapshot.AdmissionBlocked {
		t.Fatalf("capacity snapshot = %+v", snapshot)
	}
}

func TestPendingAdmissionStopsBeforeDiskExhaustion(t *testing.T) {
	ctx := context.Background()
	storage := openCapacityTestStorage(t, 100, CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              30 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	})
	request := lifecycleAttemptRequest(
		"disk-capacity",
		"start-disk-capacity",
		"owner-disk-capacity",
		1,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	storage.testHooks.diskCapacity = func(string) (uint64, uint64, error) {
		return 1_000, 50, nil
	}
	if _, err := storage.PutPendingSized(
		ctx,
		request.AttemptID,
		"disk-key",
		60,
		&panicReader{},
	); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("disk admission = %v, want capacity rejection", err)
	}
	if _, err := storage.PutPendingSized(
		ctx,
		request.AttemptID,
		"lying-length-key",
		10,
		bytes.NewReader(bytes.Repeat([]byte{9}, 60)),
	); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("dynamic disk reservation = %v, want capacity rejection", err)
	}
	assertRowCount(t, storage.cache.database, "pending_objects", 0)
	assertEmptyDirectory(t, storage.Layout().Spool)
}

func TestConcurrentReservationsCannotOversubscribePendingPool(t *testing.T) {
	ctx := context.Background()
	storage := openCapacityTestStorage(t, 100, CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              30 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	})
	first := lifecycleAttemptRequest(
		"reservation-first",
		"start-reservation-first",
		"owner-reservation-first",
		1,
	)
	second := lifecycleAttemptRequest(
		"reservation-second",
		"start-reservation-second",
		"owner-reservation-second",
		2,
	)
	for _, request := range []StartAttemptRequest{first, second} {
		if _, _, err := storage.StartAttempt(ctx, request); err != nil {
			t.Fatal(err)
		}
	}

	reader := newGatedReader(bytes.Repeat([]byte{7}, 60))
	putDone := make(chan error, 1)
	go func() {
		_, err := storage.PutPendingSized(
			ctx,
			first.AttemptID,
			"first-key",
			60,
			reader,
		)
		putDone <- err
	}()
	<-reader.started
	if _, err := storage.PutPendingSized(
		ctx,
		second.AttemptID,
		"second-key",
		60,
		&panicReader{},
	); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("overlapping reservation = %v, want capacity rejection", err)
	}
	close(reader.release)
	if err := <-putDone; err != nil {
		t.Fatalf("first reserved upload: %v", err)
	}
	snapshot, err := storage.CapacitySnapshot(ctx)
	if err != nil ||
		snapshot.PendingBytes != 60 ||
		snapshot.ReservedBytes != 0 {
		t.Fatalf("released reservation snapshot = %+v/%v", snapshot, err)
	}
}

func TestAbortedPendingBlobIsCollectedByCapacityMaintenance(t *testing.T) {
	ctx := context.Background()
	storage := openCapacityTestStorage(t, 100, CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              30 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	})
	request := lifecycleAttemptRequest(
		"capacity-abort",
		"start-capacity-abort",
		"owner-capacity-abort",
		1,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.PutPendingSized(
		ctx,
		request.AttemptID,
		"abort-key",
		80,
		bytes.NewReader(bytes.Repeat([]byte{8}, 80)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AbortAttempt(ctx, AbortAttemptRequest{
		RequestID:            "abort-capacity-abort",
		AttemptID:            request.AttemptID,
		ExpectedStateVersion: pending.StateVersion,
		Reason:               "CANCELLED",
	}); err != nil {
		t.Fatal(err)
	}
	report, err := storage.MaintainCapacity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.DeletedUnreferencedBlob != 1 {
		t.Fatalf("abort maintenance = %+v", report)
	}
	assertBlobAbsent(t, storage, pending.Object.Checksum)
}

func TestByteSLRUWatermarksAndStableTTL(t *testing.T) {
	ctx := context.Background()
	storage := openCapacityTestStorage(t, 100, CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              30 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	})
	now := lifecycleTestNow
	storage.clock = func() time.Time { return now }
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 8; index++ {
		commitCapacityObject(
			t,
			storage,
			publicKey,
			privateKey,
			index,
			now,
		)
	}
	for index := 0; index < 7; index++ {
		now = now.Add(2 * time.Minute)
		file, _, err := storage.OpenCommitted(
			ctx,
			"tenant-test",
			1,
			fmt.Sprintf("key-%04d", index),
		)
		if err != nil {
			t.Fatalf("promote %d: %v", index, err)
		}
		_ = file.Close()
	}
	snapshot, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.StableBytes != 800 ||
		snapshot.ProtectedBytes != 600 ||
		snapshot.ProbationBytes != 200 {
		t.Fatalf("post-promotion SLRU = %+v", snapshot)
	}

	commitCapacityObject(
		t,
		storage,
		publicKey,
		privateKey,
		8,
		now,
	)
	snapshot, err = storage.CapacitySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.StableBytes != 700 ||
		snapshot.ProtectedBytes != 600 ||
		snapshot.ProbationBytes != 100 ||
		snapshot.HighWatermarkReached {
		t.Fatalf("post-watermark SLRU = %+v", snapshot)
	}
	file, _, err := storage.OpenCommitted(
		ctx,
		"tenant-test",
		1,
		"key-0006",
	)
	if err != nil {
		t.Fatalf("protected entry was evicted: %v", err)
	}
	_ = file.Close()
	if file, _, err := storage.OpenCommitted(
		ctx,
		"tenant-test",
		1,
		"key-0007",
	); !errors.Is(err, ErrCacheMiss) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("cold probation entry = %v, want eviction", err)
	}

	now = now.Add(31 * 24 * time.Hour)
	report, err := storage.MaintainCapacity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.ExpiredObjects != 7 ||
		report.StableBytesAfter != 0 ||
		report.DeletedUnreferencedBlob != 9 {
		t.Fatalf("TTL maintenance = %+v", report)
	}
	snapshot, err = storage.CapacitySnapshot(ctx)
	if err != nil || snapshot.StableBytes != 0 || snapshot.StableObjects != 0 {
		t.Fatalf("post-TTL capacity = %+v/%v", snapshot, err)
	}
}

func TestSchemaThreeCommittedEntriesUpgradeIntoProbation(t *testing.T) {
	ctx := context.Background()
	testNow := time.Now().UTC()
	root := filepath.Join(t.TempDir(), "shared")
	policy := CapacityPolicy{
		DeploymentBytes:        1_000,
		RepositoryBytes:        1_000,
		PendingQuarantineBytes: 100,
		StableTTL:              7 * 24 * time.Hour,
		QuarantineTTL:          7 * 24 * time.Hour,
		HighWatermarkPercent:   85,
		LowWatermarkPercent:    75,
		ProtectedPercent:       80,
		AccessUpdateInterval:   time.Minute,
	}
	storage, err := OpenWithCapacity(ctx, root, 100, policy)
	if err != nil {
		t.Fatal(err)
	}
	storage.clock = func() time.Time { return testNow }
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	commitCapacityObject(
		t,
		storage,
		publicKey,
		privateKey,
		1,
		testNow,
	)
	layout := storage.Layout()
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	downgradeCapacitySchemaForTest(t, layout.CacheDatabase, []string{
		"DROP INDEX storage_entries_expires",
		"DROP INDEX storage_entries_scope_segment",
		"DROP TABLE storage_entries",
	})
	downgradeCapacitySchemaForTest(t, layout.ControlDatabase, []string{
		"DROP INDEX central_access_tokens_expires",
		"DROP INDEX central_access_tokens_scope",
		"DROP TABLE central_access_tokens",
		"DROP INDEX github_webhook_deliveries_job",
		"DROP INDEX github_workflow_jobs_run",
		"DROP TABLE github_webhook_deliveries",
		"DROP TABLE github_workflow_jobs",
		"DROP INDEX beta_cache_tokens_expires",
		"DROP INDEX beta_cache_tokens_scope",
		"DROP TABLE beta_cache_tokens",
		"DROP INDEX storage_maintenance_runs_completed_at",
		"DROP TABLE storage_maintenance_runs",
	})

	reopened, err := OpenWithCapacity(ctx, root, 100, policy)
	if err != nil {
		t.Fatalf("upgrade schema three storage: %v", err)
	}
	defer reopened.Close()
	var (
		repository string
		segment    string
		expiresAt  int64
	)
	if err := reopened.cache.database.QueryRowContext(
		ctx,
		`SELECT repository_id, segment, expires_at_unix_ms
FROM storage_entries
WHERE tenant_id = 'tenant-test'
  AND namespace_generation = 1
  AND cache_key = 'key-0001'`,
	).Scan(&repository, &segment, &expiresAt); err != nil {
		t.Fatal(err)
	}
	if repository != "repository-test" ||
		segment != segmentProbation ||
		expiresAt != testNow.Add(7*24*time.Hour).UnixMilli() {
		t.Fatalf(
			"migrated capacity entry = %q/%q/%d",
			repository,
			segment,
			expiresAt,
		)
	}
	file, _, err := reopened.OpenCommitted(
		ctx,
		"tenant-test",
		1,
		"key-0001",
	)
	if err != nil {
		t.Fatalf("migrated committed object: %v", err)
	}
	_ = file.Close()
}

func openCapacityTestStorage(
	t *testing.T,
	maximumObjectBytes int64,
	policy CapacityPolicy,
) *Storage {
	t.Helper()
	storage, err := OpenWithCapacity(
		context.Background(),
		filepath.Join(t.TempDir(), "shared"),
		maximumObjectBytes,
		policy,
	)
	if err != nil {
		t.Fatal(err)
	}
	storage.clock = func() time.Time { return lifecycleTestNow }
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Errorf("close capacity test storage: %v", err)
		}
	})
	return storage
}

func commitCapacityObject(
	t *testing.T,
	storage *Storage,
	publicKey ed25519.PublicKey,
	privateKey ed25519.PrivateKey,
	index int,
	now time.Time,
) {
	t.Helper()
	ctx := context.Background()
	attemptID := fmt.Sprintf("capacity-attempt-%04d", index)
	request := lifecycleAttemptRequest(
		attemptID,
		fmt.Sprintf("capacity-start-%04d", index),
		fmt.Sprintf("capacity-owner-%04d", index),
		1,
	)
	request.LeaseExpiresAt = now.Add(time.Hour)
	status, created, err := storage.StartAttempt(ctx, request)
	if err != nil || !created || status.StateVersion != 1 {
		t.Fatalf("start capacity object %d = %+v/%t/%v", index, status, created, err)
	}
	content := bytes.Repeat([]byte{byte(index + 1)}, 100)
	pending, err := storage.PutPendingSized(
		ctx,
		attemptID,
		fmt.Sprintf("key-%04d", index),
		int64(len(content)),
		bytes.NewReader(content),
	)
	if err != nil {
		t.Fatalf("put capacity object %d: %v", index, err)
	}
	canonical := signLifecycleDecision(
		t,
		privateKey,
		request,
		fmt.Sprintf("capacity-decision-%04d", index),
		[]CommitObject{pending.Object},
		testRevocationEpoch,
		now,
	)
	verified, err := VerifyCommitDecision(
		ctx,
		canonical,
		map[string]ed25519.PublicKey{
			testDecisionKeyID: publicKey,
		},
		testRevocationEpoch,
		now,
	)
	if err != nil {
		t.Fatalf("verify capacity decision %d: %v", index, err)
	}
	if _, err := storage.CommitAttempt(
		ctx,
		pending.StateVersion,
		testRevocationEpoch,
		verified,
	); err != nil {
		t.Fatalf("commit capacity object %d: %v", index, err)
	}
}

type gatedReader struct {
	reader  *bytes.Reader
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newGatedReader(content []byte) *gatedReader {
	return &gatedReader{
		reader:  bytes.NewReader(content),
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (reader *gatedReader) Read(destination []byte) (int, error) {
	reader.once.Do(func() {
		close(reader.started)
		<-reader.release
	})
	return reader.reader.Read(destination)
}

var _ io.Reader = (*gatedReader)(nil)

func downgradeCapacitySchemaForTest(
	t *testing.T,
	path string,
	statements []string,
) {
	t.Helper()
	database := openTestSQLite(t, path)
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			_ = database.Close()
			t.Fatalf("downgrade %s: %v", filepath.Base(path), err)
		}
	}
	if _, err := database.Exec(
		"DELETE FROM schema_migrations WHERE version >= 4",
	); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	if _, err := database.Exec("PRAGMA user_version = 3"); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
}
