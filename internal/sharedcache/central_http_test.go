package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestCentralHTTPSStateLifecycleCapabilitiesAndRevocation(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, t.TempDir()+"/shared")
	defer storage.Close()
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	storage.clock = func() time.Time { return now }
	handler, err := NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	writeToken := issueCentralTestToken(t, storage, now, CentralStateWrite)
	readToken := issueCentralTestToken(t, storage, now, CentralStateRead)
	cacheToken := issueCentralTestToken(t, storage, now, CentralCacheRead)

	plain := centralTestRequest(handler, http.MethodGet, "/cache/absent", readToken.Token, nil, false, map[string]string{
		CentralNamespaceHeader: centralTestScope().Namespace,
	})
	if plain.Code != http.StatusUpgradeRequired || plain.Header().Get("Upgrade") != "TLS/1.3" {
		t.Fatalf("plain request = %d %+v", plain.Code, plain.Header())
	}
	unauthorized := centralTestRequest(handler, http.MethodGet, "/cache/absent", "", nil, true, map[string]string{
		CentralNamespaceHeader: centralTestScope().Namespace,
	})
	if unauthorized.Code != http.StatusUnauthorized || unauthorized.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("unauthorized request = %d %+v", unauthorized.Code, unauthorized.Header())
	}
	wrongCapability := centralTestRequest(handler, http.MethodGet, centralStateHeadPath(StateKindEvidence), cacheToken.Token, nil, true, nil)
	if wrongCapability.Code != http.StatusForbidden {
		t.Fatalf("wrong capability = %d", wrongCapability.Code)
	}
	wrongScopePath := "/api/v1/repositories/" + stateTestOtherScope + "/state/evidence/head"
	wrongScope := centralTestRequest(handler, http.MethodGet, wrongScopePath, readToken.Token, nil, true, nil)
	if wrongScope.Code != http.StatusNotFound {
		t.Fatalf("wrong scope = %d", wrongScope.Code)
	}

	content := []byte("central-evidence")
	objectDigest := stateTestBytesDigest(content)
	objectPath := centralStateObjectPath(StateKindEvidence, objectDigest)
	missingObjectPrecondition := centralTestRequest(handler, http.MethodPut, objectPath, writeToken.Token, bytes.NewReader(content), true, nil)
	if missingObjectPrecondition.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing object precondition = %d", missingObjectPrecondition.Code)
	}
	createdObject := centralTestRequest(handler, http.MethodPut, objectPath, writeToken.Token, bytes.NewReader(content), true, map[string]string{"If-None-Match": "*"})
	if createdObject.Code != http.StatusCreated || createdObject.Header().Get("ETag") != `"`+objectDigest+`"` {
		t.Fatalf("object create = %d %+v", createdObject.Code, createdObject.Header())
	}
	object := StateObject{
		RepositoryScopeSHA256: stateTestScope,
		Kind:                  StateKindEvidence, SHA256: objectDigest, SizeBytes: int64(len(content)),
	}
	manifest := stateTestManifest(StateKindEvidence, 1, now, object, "")
	manifestRaw, manifestDigest, err := canonicalStateValue(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := centralStateManifestPath(StateKindEvidence, manifestDigest)
	createdManifest := centralTestRequest(handler, http.MethodPut, manifestPath, writeToken.Token, bytes.NewReader(manifestRaw), true, map[string]string{"If-None-Match": "*"})
	if createdManifest.Code != http.StatusCreated {
		t.Fatalf("manifest create = %d %s", createdManifest.Code, createdManifest.Body.String())
	}
	head := StateHead{
		SchemaVersion: "buildopt.central/state-head/v1", RecordType: "CENTRAL_STATE_HEAD",
		Kind: StateKindEvidence, RepositoryScopeSHA256: stateTestScope,
		Generation: 1, ManifestSHA256: manifestDigest,
		CompatibilitySHA256: manifest.CompatibilitySHA256,
		UpdatedAt:           now.Format(time.RFC3339Nano), Authority: manifest.Authority,
	}
	casDocument := centralStateCASDocument{
		SchemaVersion: "buildopt.central/state-cas/v1", RecordType: "CENTRAL_STATE_CAS",
		Operation: "CREATE_OR_ADVANCE", IdempotencyKey: stateTestDigest("central-cas-one"),
		ExpectedGeneration: 0, Next: head,
	}
	casRaw, err := json.Marshal(casDocument)
	if err != nil {
		t.Fatal(err)
	}
	createdHead := centralTestRequest(handler, http.MethodPost, centralStateCASPath(StateKindEvidence), writeToken.Token, bytes.NewReader(casRaw), true, map[string]string{"If-None-Match": "*"})
	if createdHead.Code != http.StatusCreated || createdHead.Header().Get("ETag") == "" {
		t.Fatalf("head create = %d %s", createdHead.Code, createdHead.Body.String())
	}
	replayedHead := centralTestRequest(handler, http.MethodPost, centralStateCASPath(StateKindEvidence), writeToken.Token, bytes.NewReader(casRaw), true, map[string]string{"If-None-Match": "*"})
	if replayedHead.Code != http.StatusOK || replayedHead.Header().Get("X-BuildOpt-Idempotent-Replay") != "true" {
		t.Fatalf("head replay = %d %+v", replayedHead.Code, replayedHead.Header())
	}

	loadedHead := centralTestRequest(handler, http.MethodGet, centralStateHeadPath(StateKindEvidence), readToken.Token, nil, true, nil)
	if loadedHead.Code != http.StatusOK || loadedHead.Body.String() != createdHead.Body.String() {
		t.Fatalf("head get = %d %s", loadedHead.Code, loadedHead.Body.String())
	}
	loadedManifest := centralTestRequest(handler, http.MethodGet, manifestPath, readToken.Token, nil, true, nil)
	if loadedManifest.Code != http.StatusOK || !bytes.Equal(loadedManifest.Body.Bytes(), manifestRaw) {
		t.Fatalf("manifest get = %d %s", loadedManifest.Code, loadedManifest.Body.String())
	}
	loadedObject := centralTestRequest(handler, http.MethodGet, objectPath, readToken.Token, nil, true, nil)
	if loadedObject.Code != http.StatusOK || !bytes.Equal(loadedObject.Body.Bytes(), content) {
		t.Fatalf("object get = %d %q", loadedObject.Code, loadedObject.Body.Bytes())
	}
	if loadedHead.Header().Get("Cache-Control") != "no-store" ||
		loadedHead.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("security headers = %+v", loadedHead.Header())
	}

	revoked, err := storage.RevokeCentralToken(ctx, readToken.TokenID, now.Add(time.Minute))
	if err != nil || !revoked {
		t.Fatalf("revoke read token = %t/%v", revoked, err)
	}
	storage.clock = func() time.Time { return now.Add(2 * time.Minute) }
	afterRevoke := centralTestRequest(handler, http.MethodGet, centralStateHeadPath(StateKindEvidence), readToken.Token, nil, true, nil)
	if afterRevoke.Code != http.StatusUnauthorized {
		t.Fatalf("revoked request = %d", afterRevoke.Code)
	}
}

func TestCentralHTTPSCacheCapabilitiesAndAttemptBinding(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, t.TempDir()+"/shared")
	defer storage.Close()
	now := lifecycleTestNow
	storage.clock = func() time.Time { return now }
	handler, err := NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	readToken := issueCentralTestToken(t, storage, now, CentralCacheRead)
	writeToken := issueCentralTestToken(t, storage, now, CentralCacheWrite)
	stateToken := issueCentralTestToken(t, storage, now, CentralStateRead)

	cacheHeaders := map[string]string{CentralNamespaceHeader: centralTestScope().Namespace}
	missingNamespace := centralTestRequest(handler, http.MethodGet, "/cache/key-one", readToken.Token, nil, true, nil)
	if missingNamespace.Code != http.StatusNotFound {
		t.Fatalf("missing namespace = %d", missingNamespace.Code)
	}
	miss := centralTestRequest(handler, http.MethodGet, "/cache/key-one", readToken.Token, nil, true, cacheHeaders)
	if miss.Code != http.StatusNotFound {
		t.Fatalf("cache miss = %d", miss.Code)
	}
	wrongNamespace := centralTestRequest(handler, http.MethodGet, "/cache/key-one", readToken.Token, nil, true, map[string]string{
		CentralNamespaceHeader: "gradle/other",
	})
	if wrongNamespace.Code != http.StatusNotFound {
		t.Fatalf("wrong namespace = %d", wrongNamespace.Code)
	}
	forbidden := centralTestRequest(handler, http.MethodGet, "/cache/key-one", stateToken.Token, nil, true, cacheHeaders)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("cache read capability = %d", forbidden.Code)
	}
	missingAttempt := centralTestRequest(handler, http.MethodPut, "/cache/key-one", writeToken.Token, bytes.NewReader([]byte("cache-value")), true, cacheHeaders)
	if missingAttempt.Code != http.StatusBadRequest {
		t.Fatalf("missing attempt = %d", missingAttempt.Code)
	}

	attempt := lifecycleAttemptRequest("central-attempt", "central-start", "central-owner", 1)
	started, _, err := storage.StartAttempt(ctx, attempt)
	if err != nil || started.AttemptID != attempt.AttemptID {
		t.Fatalf("start attempt = %+v/%v", started, err)
	}
	value := []byte("cache-value")
	put := centralTestRequest(handler, http.MethodPut, "/cache/key-one", writeToken.Token, bytes.NewReader(value), true, map[string]string{
		CentralAttemptHeader:   attempt.AttemptID,
		CentralNamespaceHeader: centralTestScope().Namespace,
		"Content-Length":       strconv.Itoa(len(value)),
	})
	if put.Code != http.StatusCreated {
		t.Fatalf("cache put = %d %s", put.Code, put.Body.String())
	}
	if status, err := storage.AttemptStatus(ctx, attempt.AttemptID); err != nil || status.PendingObjectCount != 1 {
		t.Fatalf("pending object = %+v/%v", status, err)
	}
	digest := put.Header().Get("X-BuildOpt-Blob-Digest")
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifyLifecycleDecision(t, map[string]ed25519.PublicKey{
		testDecisionKeyID: publicKey,
	}, signLifecycleDecision(
		t, privateKey, attempt, "central-decision",
		[]CommitObject{{
			NamespaceGeneration: 1, Key: "key-one", Checksum: digest,
			SizeBytes: int64(len(value)),
		}}, testRevocationEpoch, now,
	))
	if _, err := storage.CommitAttempt(ctx, 2, testRevocationEpoch, verified); err != nil {
		t.Fatal(err)
	}
	hit := centralTestRequest(handler, http.MethodGet, "/cache/key-one", readToken.Token, nil, true, cacheHeaders)
	if hit.Code != http.StatusOK || !bytes.Equal(hit.Body.Bytes(), value) {
		t.Fatalf("cache hit = %d %q", hit.Code, hit.Body.Bytes())
	}
	otherScope := centralTestScope()
	otherScope.Repository = "other-repository"
	otherToken, err := storage.IssueCentralToken(ctx, CentralTokenIssueRequest{
		Scope: otherScope, Capabilities: []CentralCapability{CentralCacheRead},
		ExpiresAt: now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	isolation := centralTestRequest(handler, http.MethodGet, "/cache/key-one", otherToken.Token, nil, true, cacheHeaders)
	if isolation.Code != http.StatusNotFound {
		t.Fatalf("cross-repository cache read = %d", isolation.Code)
	}
}

func issueCentralTestToken(
	t *testing.T,
	storage *Storage,
	now time.Time,
	capabilities ...CentralCapability,
) IssuedCentralToken {
	t.Helper()
	issued, err := storage.IssueCentralToken(context.Background(), CentralTokenIssueRequest{
		Scope: centralTestScope(), Capabilities: capabilities, ExpiresAt: now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return issued
}

func centralTestRequest(
	handler http.Handler,
	method string,
	path string,
	token string,
	body io.Reader,
	secure bool,
	headers map[string]string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "https://central.example"+path, body)
	if !secure {
		request.TLS = nil
	} else if request.TLS == nil {
		request.TLS = &tls.ConnectionState{Version: tls.VersionTLS13}
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func centralStateObjectPath(kind StateKind, digest string) string {
	return centralStateBasePath(kind) + "/objects/" + digest
}

func centralStateManifestPath(kind StateKind, digest string) string {
	return centralStateBasePath(kind) + "/manifests/" + digest
}

func centralStateHeadPath(kind StateKind) string {
	return centralStateBasePath(kind) + "/head"
}

func centralStateCASPath(kind StateKind) string {
	return centralStateBasePath(kind) + "/head:cas"
}

func centralStateBasePath(kind StateKind) string {
	segment := map[StateKind]string{
		StateKindPortfolio: "portfolios", StateKindEvidence: "evidence",
		StateKindCheckpoint: "checkpoints",
	}[kind]
	return "/api/v1/repositories/" + stateTestScope + "/state/" + segment
}
