package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/edgecache"
	"github.com/tonyredondo/buildopt/internal/localauthority"
)

type runningEdgeProxy struct {
	baseURL string
	cancel  context.CancelFunc
	done    <-chan error
}

func TestTwoEdgeProxiesKeepPendingPrivateAndSharedResolvesCollision(t *testing.T) {
	ctx := context.Background()
	now := sharedAuthorityNow
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return now }
	verifiedAuthority, credential, privateKey, _ := sharedAuthorityFixture(
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
	sharedHandler, err := NewBetaTokenHTTPHandler(
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
	var sharedAvailable atomic.Bool
	sharedAvailable.Store(true)
	sharedServer := httptest.NewServer(http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		if !sharedAvailable.Load() {
			response.Header().Set("Content-Length", "0")
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		sharedHandler.ServeHTTP(response, request)
	}))
	defer sharedServer.Close()

	readAuthority, err := edgecache.NewReadAuthority(verifiedAuthority, now)
	if err != nil {
		t.Fatal(err)
	}
	writeAuthority, err := edgecache.NewWriteAuthority(verifiedAuthority, now)
	if err != nil {
		t.Fatal(err)
	}
	stores := make([]*edgecache.Store, 0, 2)
	clients := make([]*edgecache.SharedClient, 0, 2)
	proxies := make([]runningEdgeProxy, 0, 2)
	for index := range 2 {
		config := edgecache.Config{
			Shared: edgecache.Shared{
				BaseURL:               sharedServer.URL,
				AllowInsecureLoopback: true,
			},
			Storage: edgecache.Storage{
				StateDirectory: filepath.Join(
					t.TempDir(),
					"edge-"+strconv.Itoa(index+1),
				),
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
		store, err := edgecache.OpenStore(config)
		if err != nil {
			t.Fatal(err)
		}
		stores = append(stores, store)
		client, err := edgecache.NewSharedClient(
			config.Shared,
			[]byte(token.Token),
			sharedServer.Client(),
		)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, client)
		proxy, err := edgecache.NewProxy(
			store,
			client,
			readAuthority,
			&writeAuthority,
			func() time.Time { return now },
		)
		if err != nil {
			t.Fatal(err)
		}
		proxies = append(proxies, startRealEdgeProxy(t, proxy))
	}
	defer func() {
		for _, proxy := range proxies {
			proxy.cancel()
			if err := <-proxy.done; err != nil {
				t.Errorf("stop Edge proxy: %v", err)
			}
		}
		for _, client := range clients {
			client.Close()
		}
		for _, store := range stores {
			if err := store.Close(); err != nil {
				t.Errorf("close Edge store: %v", err)
			}
		}
	}()

	firstPayload := []byte("first Edge candidate")
	secondPayload := []byte("second Edge collision")
	assertEdgePUT(t, proxies[0].baseURL, "two-node-key", firstPayload, http.StatusCreated)
	assertEdgePUT(t, proxies[1].baseURL, "two-node-key", secondPayload, http.StatusCreated)
	status, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil || status.PendingObjectCount != 0 {
		t.Fatalf("Shared before asynchronous replication = %+v/%v", status, err)
	}

	sharedAvailable.Store(false)
	assertEdgeGET(t, proxies[0].baseURL, "two-node-key", firstPayload, "PENDING_ATTEMPT")
	assertEdgeGET(t, proxies[1].baseURL, "two-node-key", secondPayload, "PENDING_ATTEMPT")
	sharedAvailable.Store(true)

	firstReport, err := stores[0].ReplicatePendingOnce(
		ctx,
		writeAuthority,
		clients[0],
		now,
	)
	if err != nil || firstReport.Replicated != 1 {
		t.Fatalf("first Edge replication = %+v/%v", firstReport, err)
	}
	secondReport, err := stores[1].ReplicatePendingOnce(
		ctx,
		writeAuthority,
		clients[1],
		now,
	)
	if err != nil || secondReport.Rejected != 1 {
		t.Fatalf("colliding Edge replication = %+v/%v", secondReport, err)
	}
	status, err = storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil || status.State != AttemptPending ||
		status.PendingObjectCount != 1 {
		t.Fatalf("Shared collision state = %+v/%v", status, err)
	}

	digestBytes := sha256.Sum256(firstPayload)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
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
		"two-edge-decision",
		[]CommitObject{{
			NamespaceGeneration: status.NamespaceGeneration,
			Key:                 "two-node-key",
			Checksum:            digest,
			SizeBytes:           int64(len(firstPayload)),
		}},
		testRevocationEpoch,
		now,
	)
	verifiedDecision, err := VerifyCommitDecision(
		ctx,
		canonical,
		map[string]ed25519.PublicKey{
			testDecisionKeyID: privateKey.Public().(ed25519.PublicKey),
		},
		testRevocationEpoch,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CommitAttempt(
		ctx,
		status.StateVersion,
		testRevocationEpoch,
		verifiedDecision,
	); err != nil {
		t.Fatal(err)
	}

	assertEdgeGET(t, proxies[0].baseURL, "two-node-key", firstPayload, "COMMITTED")
	assertEdgeGET(t, proxies[1].baseURL, "two-node-key", firstPayload, "COMMITTED")
	sharedAvailable.Store(false)
	assertEdgeGET(t, proxies[0].baseURL, "two-node-key", firstPayload, "COMMITTED")
	assertEdgeGET(t, proxies[1].baseURL, "two-node-key", firstPayload, "COMMITTED")
}

func startRealEdgeProxy(t *testing.T, proxy *edgecache.Proxy) runningEdgeProxy {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- proxy.Serve(ctx, listener)
	}()
	return runningEdgeProxy{
		baseURL: "http://" + listener.Addr().String(),
		cancel:  cancel,
		done:    done,
	}
}

func assertEdgePUT(
	t *testing.T,
	baseURL string,
	key string,
	payload []byte,
	wantStatus int,
) {
	t.Helper()
	request, err := http.NewRequest(
		http.MethodPut,
		baseURL+"/cache/"+key,
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != wantStatus {
		t.Fatalf("Edge PUT %s = %d, want %d", key, response.StatusCode, wantStatus)
	}
}

func assertEdgeGET(
	t *testing.T,
	baseURL string,
	key string,
	want []byte,
	wantState string,
) {
	t.Helper()
	response, err := http.Get(baseURL + "/cache/" + key)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	actual, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK ||
		!bytes.Equal(actual, want) ||
		response.Header.Get("X-BuildOpt-Edge-State") != wantState {
		t.Fatalf(
			"Edge GET %s = %d/%q/%q/%v, want %q/%q",
			key,
			response.StatusCode,
			actual,
			response.Header.Get("X-BuildOpt-Edge-State"),
			err,
			want,
			wantState,
		)
	}
}
