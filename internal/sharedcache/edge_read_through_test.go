package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/edgecache"
	"github.com/tonyredondo/buildopt/internal/localauthority"
)

func TestEdgeReadThroughUsesRealCommittedSharedRouteAndSurvivesOfflineRestart(t *testing.T) {
	ctx := context.Background()
	now := sharedAuthorityNow
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return now }
	verifiedAuthority, credential, privateKey, _ := sharedAuthorityFixture(t, func(document *localauthority.Document) {
		document.Attempt.AllowRead = true
		document.Attempt.AllowWrite = true
	})
	binding, _, err := storage.InstallLocalAuthority(ctx, verifiedAuthority, credential, now)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("real Shared committed bytes")
	pending, err := storage.PutPending(ctx, binding.AttemptID, "edge-real-key", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	status, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	request := StartAttemptRequest{
		AttemptID:                 status.AttemptID,
		AuthorityDigest:           status.AuthorityDigest,
		Repository:                status.Repository,
		NamespaceGeneration:       status.NamespaceGeneration,
		SourceRevision:            status.SourceRevision,
		SourceStateDigest:         status.SourceStateDigest,
		PolicyDigest:              status.PolicyDigest,
		ConfigurationPolicyDigest: status.ConfigurationPolicyDigest,
		CacheContractDigest:       status.CacheContractDigest,
		OwnerID:                   status.OwnerID,
		LeaseID:                   status.LeaseID,
		LeaseExpiresAt:            status.LeaseExpiresAt,
	}
	canonical := signLifecycleDecision(
		t,
		privateKey,
		request,
		"edge-real-decision",
		[]CommitObject{pending.Object},
		testRevocationEpoch,
		now,
	)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	verifiedDecision, err := VerifyCommitDecision(
		ctx,
		canonical,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		testRevocationEpoch,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CommitAttempt(ctx, status.StateVersion, testRevocationEpoch, verifiedDecision); err != nil {
		t.Fatal(err)
	}
	handler, err := NewBetaTokenHTTPHandler(storage, binding, BetaTokenPlaneStable)
	if err != nil {
		t.Fatal(err)
	}
	token := issueBetaTokenForTest(
		t,
		storage,
		betaScopeForBinding(binding, BetaTokenPlaneStable),
		BetaTokenRead,
		now,
		now.Add(30*time.Minute),
	)
	server := httptest.NewServer(handler)
	authority, err := edgecache.NewReadAuthority(verifiedAuthority, now)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "edge")
	config := edgecache.Config{
		Shared: edgecache.Shared{
			BaseURL:               server.URL,
			AllowInsecureLoopback: true,
		},
		Storage: edgecache.Storage{
			StateDirectory:       root,
			FilesystemPolicy:     edgecache.FilesystemPolicy,
			CapacityBytes:        edgecache.MinimumCapacityBytes,
			MaximumObjectBytes:   edgecache.MaximumObjectBytes,
			StableTTLSeconds:     int64(edgecache.MaximumStableTTL / time.Second),
			PendingTTLSeconds:    int64(edgecache.MaximumPendingTTL / time.Second),
			HighWatermarkPercent: edgecache.HighWatermarkPercent,
			LowWatermarkPercent:  edgecache.LowWatermarkPercent,
			ProtectedPercent:     edgecache.ProtectedPercent,
		},
	}
	edge, err := edgecache.OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	client, err := edgecache.NewSharedClient(config.Shared, []byte(token.Token), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	file, err := edge.ReadThrough(ctx, authority, client, "edge-real-key", now)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("online Edge read = %q/%v", actual, err)
	}
	client.Close()
	server.Close()
	if err := edge.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := edgecache.OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	file, err = reopened.OpenCommitted(ctx, authority, "edge-real-key", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	actual, err = io.ReadAll(file)
	_ = file.Close()
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("offline Edge read = %q/%v", actual, err)
	}
}

func TestEdgePendingReplicationUsesRealSharedAttemptWithoutLocalPromotion(t *testing.T) {
	ctx := context.Background()
	now := sharedAuthorityNow
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return now }
	verifiedAuthority, credential, _, _ := sharedAuthorityFixture(
		t,
		func(document *localauthority.Document) {
			document.Attempt.AllowRead = true
			document.Attempt.AllowWrite = true
		},
	)
	binding, _, err := storage.InstallLocalAuthority(
		ctx,
		verifiedAuthority,
		credential,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewBetaTokenHTTPHandler(
		storage,
		binding,
		BetaTokenPlaneStable,
	)
	if err != nil {
		t.Fatal(err)
	}
	token := issueBetaTokenForTest(
		t,
		storage,
		betaScopeForBinding(binding, BetaTokenPlaneStable),
		BetaTokenReadWrite,
		now,
		now.Add(30*time.Minute),
	)
	server := httptest.NewServer(handler)
	defer server.Close()

	root := filepath.Join(t.TempDir(), "edge-pending")
	config := edgecache.Config{
		Shared: edgecache.Shared{
			BaseURL:               server.URL,
			AllowInsecureLoopback: true,
		},
		Storage: edgecache.Storage{
			StateDirectory:       root,
			FilesystemPolicy:     edgecache.FilesystemPolicy,
			CapacityBytes:        edgecache.MinimumCapacityBytes,
			MaximumObjectBytes:   edgecache.MaximumObjectBytes,
			StableTTLSeconds:     int64(edgecache.MaximumStableTTL / time.Second),
			PendingTTLSeconds:    int64(edgecache.MaximumPendingTTL / time.Second),
			HighWatermarkPercent: edgecache.HighWatermarkPercent,
			LowWatermarkPercent:  edgecache.LowWatermarkPercent,
			ProtectedPercent:     edgecache.ProtectedPercent,
		},
	}
	edge, err := edgecache.OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	writeAuthority, err := edgecache.NewWriteAuthority(verifiedAuthority, now)
	if err != nil {
		t.Fatal(err)
	}
	readAuthority, err := edgecache.NewReadAuthority(verifiedAuthority, now)
	if err != nil {
		t.Fatal(err)
	}
	client, err := edgecache.NewSharedClient(
		config.Shared,
		[]byte(token.Token),
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	payload := []byte("real Edge pending replication")
	written, err := edge.PutPendingDurable(
		ctx,
		writeAuthority,
		"edge-pending-key",
		int64(len(payload)),
		bytes.NewReader(payload),
		now,
	)
	if err != nil || !written.Added {
		t.Fatalf("local pending write = %+v/%v", written, err)
	}
	status, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil || status.PendingObjectCount != 0 {
		t.Fatalf("Shared before replication = %+v/%v", status, err)
	}
	file, err := edge.OpenPending(
		ctx,
		writeAuthority,
		"edge-pending-key",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("attempt-private Edge read = %q/%v", actual, err)
	}
	if file, err := edge.OpenCommitted(
		ctx,
		readAuthority,
		"edge-pending-key",
		now,
	); file != nil || !errors.Is(err, edgecache.ErrCacheMiss) {
		t.Fatalf("pre-replication local promotion = %v/%v", file, err)
	}

	report, err := edge.ReplicatePendingOnce(
		ctx,
		writeAuthority,
		client,
		now,
	)
	if err != nil || report.Replicated != 1 || report.Claimed != 1 {
		t.Fatalf("real Shared replication = %+v/%v", report, err)
	}
	status, err = storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil || status.State != AttemptPending ||
		status.PendingObjectCount != 1 {
		t.Fatalf("Shared pending state = %+v/%v", status, err)
	}
	if file, _, err := storage.OpenCommitted(
		ctx,
		binding.httpBinding.Tenant,
		binding.httpBinding.NamespaceGeneration,
		"edge-pending-key",
	); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Shared pending became committed = %v/%v", file, err)
	}
	if file, err := edge.OpenCommitted(
		ctx,
		readAuthority,
		"edge-pending-key",
		now,
	); file != nil || !errors.Is(err, edgecache.ErrCacheMiss) {
		t.Fatalf("post-replication local promotion = %v/%v", file, err)
	}
	if err := edge.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := edgecache.OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	snapshot, err := reopened.PendingSnapshot(ctx)
	if err != nil || snapshot.Replicated != 1 || snapshot.Objects != 1 {
		t.Fatalf("restart pending snapshot = %+v/%v", snapshot, err)
	}
	file, err = reopened.OpenPending(
		ctx,
		writeAuthority,
		"edge-pending-key",
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	actual, err = io.ReadAll(file)
	_ = file.Close()
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("restart pending read = %q/%v", actual, err)
	}
}
