package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
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
