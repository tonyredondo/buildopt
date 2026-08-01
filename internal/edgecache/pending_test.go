package edgecache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type pendingPanicReader struct{}

func (pendingPanicReader) Read([]byte) (int, error) {
	panic("pending body was read before capacity admission")
}

func testWriteAuthority() WriteAuthority {
	read := testReadAuthority()
	return WriteAuthority{
		tenant:               read.tenant,
		repository:           read.repository,
		trustDomain:          read.trustDomain,
		namespace:            read.namespace,
		namespaceGeneration:  read.namespaceGeneration,
		authorityDigest:      read.authorityDigest,
		revocationEpoch:      read.revocationEpoch,
		revocationDigest:     read.revocationDigest,
		l1SecurityGeneration: read.l1SecurityGeneration,
		attemptID:            "attempt-edge-pending",
		expiresAt:            read.expiresAt,
	}
}

func TestPendingIsAttemptPrivateIdempotentAndTTLBound(t *testing.T) {
	store, readAuthority := reducedCapacityStore(t, 1000)
	store.pendingTTL = time.Minute
	writeAuthority := testWriteAuthority()
	payload := []byte("attempt-private bytes")

	first, err := store.PutPendingDurable(
		context.Background(),
		writeAuthority,
		"pending-key",
		int64(len(payload)),
		bytes.NewReader(payload),
		edgeTestNow,
	)
	if err != nil || !first.Added || !validDigest(first.Digest) {
		t.Fatalf("first pending write = %+v/%v", first, err)
	}
	replay, err := store.PutPendingDurable(
		context.Background(),
		writeAuthority,
		"pending-key",
		int64(len(payload)),
		bytes.NewReader(payload),
		edgeTestNow,
	)
	if err != nil || replay.Added || replay.Digest != first.Digest {
		t.Fatalf("pending replay = %+v/%v", replay, err)
	}
	conflict := []byte("different attempt bytes")
	if _, err := store.PutPendingDurable(
		context.Background(),
		writeAuthority,
		"pending-key",
		int64(len(conflict)),
		bytes.NewReader(conflict),
		edgeTestNow,
	); !errors.Is(err, ErrPendingConflict) {
		t.Fatalf("pending collision = %v", err)
	}

	file, err := store.OpenPending(
		context.Background(),
		writeAuthority,
		"pending-key",
		edgeTestNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertFileBytes(t, file, payload)
	otherAttempt := writeAuthority
	otherAttempt.attemptID = "attempt-edge-other"
	if file, err := store.OpenPending(
		context.Background(),
		otherAttempt,
		"pending-key",
		edgeTestNow,
	); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("cross-attempt pending read = %v/%v", file, err)
	}
	if file, err := store.OpenCommitted(
		context.Background(),
		readAuthority,
		"pending-key",
		edgeTestNow,
	); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("pending became committed = %v/%v", file, err)
	}
	snapshot, err := store.CapacitySnapshot(context.Background())
	if err != nil || snapshot.PendingBytes != int64(len(payload)) ||
		snapshot.TotalLogicalBytes != int64(len(payload)) {
		t.Fatalf("pending capacity = %+v/%v", snapshot, err)
	}
	rejectedSize := store.capacityBytes - int64(len(payload)) + 1
	if _, err := store.PutPendingDurable(
		context.Background(),
		writeAuthority,
		"over-capacity",
		rejectedSize,
		pendingPanicReader{},
		edgeTestNow,
	); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("pending hard-cap rejection = %v", err)
	}
	report, err := store.Maintain(
		context.Background(),
		edgeTestNow.Add(time.Minute),
	)
	if err != nil || report.ExpiredPending != 1 {
		t.Fatalf("pending TTL maintenance = %+v/%v", report, err)
	}
	if file, err := store.OpenPending(
		context.Background(),
		writeAuthority,
		"pending-key",
		edgeTestNow.Add(time.Minute),
	); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expired pending read = %v/%v", file, err)
	}
}

func TestPendingReplicationRetriesRecoversAndRunsAsynchronously(t *testing.T) {
	ctx := context.Background()
	now := edgeTestNow
	writeAuthority := testWriteAuthority()
	root := filepath.Join(t.TempDir(), "edge")
	config := testStoreConfig(root, "http://127.0.0.1:8042")
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("durable replication bytes")
	written, err := store.PutPendingDurable(
		ctx,
		writeAuthority,
		"retry-key",
		int64(len(payload)),
		bytes.NewReader(payload),
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	var unavailable atomic.Bool
	unavailable.Store(true)
	replicated := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		if unavailable.Load() {
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil || !bytes.Equal(body, payload) ||
			request.Header.Get("Authorization") != "Bearer edge-token" ||
			request.Header.Get(sharedAuthorityDigestHeader) !=
				writeAuthority.authorityDigest {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.Header().Set("X-BuildOpt-Blob-Digest", written.Digest)
		response.WriteHeader(http.StatusCreated)
		replicated <- request.URL.Path
	}))
	defer server.Close()
	config.Shared.BaseURL = server.URL
	client, err := NewSharedClient(config.Shared, []byte("edge-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	first, err := store.ReplicatePendingOnce(ctx, writeAuthority, client, now)
	if err != nil || first.Deferred != 1 || first.Replicated != 0 {
		t.Fatalf("first replication = %+v/%v", first, err)
	}
	beforeBackoff, err := store.ReplicatePendingOnce(
		ctx,
		writeAuthority,
		client,
		now.Add(500*time.Millisecond),
	)
	if err != nil || beforeBackoff.Claimed != 0 {
		t.Fatalf("replication before durable backoff = %+v/%v", beforeBackoff, err)
	}
	if _, err := store.database.Exec(`UPDATE edge_pending_objects
SET state = 'REPLICATING' WHERE cache_key = 'retry-key'`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var state, lastError string
	if err := reopened.database.QueryRow(`SELECT state, last_error_class
FROM edge_pending_objects WHERE cache_key = 'retry-key'`).Scan(
		&state,
		&lastError,
	); err != nil || state != pendingQueued || lastError != "INTERRUPTED" {
		t.Fatalf("recovered claim = %s/%s/%v", state, lastError, err)
	}
	unavailable.Store(false)
	second, err := reopened.ReplicatePendingOnce(
		ctx,
		writeAuthority,
		client,
		now.Add(2*time.Second),
	)
	if err != nil || second.Replicated != 1 || second.Deferred != 0 {
		t.Fatalf("recovered replication = %+v/%v", second, err)
	}
	if path := <-replicated; path != "/cache/retry-key" {
		t.Fatalf("replicated path = %q", path)
	}
	if file, err := reopened.OpenCommitted(
		ctx,
		testReadAuthority(),
		"retry-key",
		now.Add(2*time.Second),
	); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("replication promoted locally = %v/%v", file, err)
	}

	secondPayload := []byte("async bytes")
	payload = secondPayload
	secondWrite, err := reopened.PutPendingDurable(
		ctx,
		writeAuthority,
		"async-key",
		int64(len(secondPayload)),
		bytes.NewReader(secondPayload),
		now.Add(3*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	written = secondWrite
	workerContext, cancel := context.WithCancel(ctx)
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- reopened.RunReplicator(
			workerContext,
			writeAuthority,
			client,
			5*time.Millisecond,
			func() time.Time { return now.Add(3 * time.Second) },
		)
	}()
	select {
	case path := <-replicated:
		if path != "/cache/async-key" {
			t.Fatalf("async replicated path = %q", path)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("asynchronous replicator did not publish")
	}
	cancel()
	if err := <-workerDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("replicator shutdown = %v", err)
	}
}
