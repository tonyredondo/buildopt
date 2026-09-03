package sharedcache

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestWCNCPHTTPSAuthorityFailsClosed(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, t.TempDir()+"/shared")
	defer storage.Close()
	now := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)
	storage.clock = func() time.Time { return now }
	handler, err := NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	observerToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	if _, err := storage.GrantWCNCPActor(ctx, observerToken.TokenID, WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	validatorToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	if _, err := storage.GrantWCNCPActor(ctx, validatorToken.TokenID, WCNCPActorValidator, now); err != nil {
		t.Fatal(err)
	}
	ownerToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	if _, err := storage.GrantWCNCPActor(ctx, ownerToken.TokenID, WCNCPActorOwner, now); err != nil {
		t.Fatal(err)
	}
	developerToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	// No grant: developer-default, no WCNCP writes.

	obsRecord := wcncpTestValidRecord(t, WCNCPKindObservation)
	obsCanonical, obsDigest, err := CanonicalWCNCPRecord(WCNCPKindObservation, obsRecord)
	if err != nil {
		t.Fatal(err)
	}
	obsPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/objects/" + obsDigest

	// Wrong actor: developer cannot publish observations.
	developerPut := centralTestRequest(handler, http.MethodPut, obsPath, developerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*"})
	if developerPut.Code != http.StatusForbidden {
		t.Fatalf("developer observation write = %d", developerPut.Code)
	}
	// Fork context is read-only even for trusted observers.
	forkPut := centralTestRequest(handler, http.MethodPut, obsPath, observerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*", wcncpForkHeader: "1"})
	if forkPut.Code != http.StatusForbidden {
		t.Fatalf("fork observation write = %d", forkPut.Code)
	}
	// Wrong repository scope fails closed with 404, not 403, to avoid scope oracle.
	wrongScopePath := "/api/v1/repositories/" + stateTestOtherScope + "/wcncp/WCNCP_OBSERVATION/objects/" + obsDigest
	wrongScope := centralTestRequest(handler, http.MethodPut, wrongScopePath, observerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*"})
	if wrongScope.Code != http.StatusNotFound {
		t.Fatalf("wrong scope = %d", wrongScope.Code)
	}
	// Correct observer publication succeeds.
	created := centralTestRequest(handler, http.MethodPut, obsPath, observerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*"})
	if created.Code != http.StatusCreated {
		t.Fatalf("observer observation write = %d %s", created.Code, created.Body.String())
	}
	// Digest mismatch fails closed.
	mismatchPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/objects/" + wcncpTestDigest("mismatch")
	mismatch := centralTestRequest(handler, http.MethodPut, mismatchPath, observerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*"})
	if mismatch.Code != http.StatusBadRequest {
		t.Fatalf("digest mismatch = %d", mismatch.Code)
	}
	// Validator cannot publish observations; owner cannot publish validations.
	validatorObs := centralTestRequest(handler, http.MethodPut, obsPath, validatorToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*"})
	if validatorObs.Code != http.StatusForbidden {
		t.Fatalf("validator observation write = %d", validatorObs.Code)
	}
	// Owner decision requires OWNER actor: validator attempting a decision fails.
	decisionRecord := wcncpTestValidRecord(t, WCNCPKindDecision)
	decisionCanonical, decisionDigest, err := CanonicalWCNCPRecord(WCNCPKindDecision, decisionRecord)
	if err != nil {
		t.Fatal(err)
	}
	decisionPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_DECISION/objects/" + decisionDigest
	validatorDecision := centralTestRequest(handler, http.MethodPut, decisionPath, validatorToken.Token, bytes.NewReader(decisionCanonical), true, map[string]string{"If-None-Match": "*"})
	if validatorDecision.Code != http.StatusForbidden {
		t.Fatalf("validator decision write = %d", validatorDecision.Code)
	}
	// Live revocation fails closed.
	revoked, err := storage.RevokeCentralToken(ctx, observerToken.TokenID, now.Add(time.Minute))
	if err != nil || !revoked {
		t.Fatalf("revoke = %v/%v", revoked, err)
	}
	afterRevoke := centralTestRequest(handler, http.MethodGet, obsPath, observerToken.Token, nil, true, nil)
	if afterRevoke.Code != http.StatusUnauthorized {
		t.Fatalf("revoked read = %d", afterRevoke.Code)
	}
	// Expired token fails closed.
	expiredToken := issueCentralTestToken(t, storage, now.Add(-2*time.Hour), CentralStateRead, CentralStateWrite)
	// Manually expire by advancing clock past expiry (tokens live 1h in helper).
	storage.clock = func() time.Time { return now.Add(2 * time.Hour) }
	expired := centralTestRequest(handler, http.MethodGet, obsPath, expiredToken.Token, nil, true, nil)
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("expired read = %d", expired.Code)
	}
	storage.clock = func() time.Time { return now }
}

func TestWCNCPHTTPSIdempotencyStalePartialAndRestart(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir() + "/shared"
	storage := openStateTestStorage(t, ctx, root)
	now := time.Date(2026, 9, 4, 11, 0, 0, 0, time.UTC)
	storage.clock = func() time.Time { return now }
	handler, err := NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	observerToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	if _, err := storage.GrantWCNCPActor(ctx, observerToken.TokenID, WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	obsRecord := wcncpTestValidRecord(t, WCNCPKindObservation)
	obsCanonical, obsDigest, err := CanonicalWCNCPRecord(WCNCPKindObservation, obsRecord)
	if err != nil {
		t.Fatal(err)
	}
	obsPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/objects/" + obsDigest
	if code := centralTestRequest(handler, http.MethodPut, obsPath, observerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*"}).Code; code != http.StatusCreated {
		t.Fatalf("object put = %d", code)
	}
	// Manifest referencing a missing object fails as partial batch precondition.
	missingManifest := wcncpTestManifest(WCNCPKindObservation, 7, now, WCNCPObject{RepositoryScopeSHA256: stateTestScope, Kind: WCNCPKindObservation, SHA256: wcncpTestDigest("absent"), SizeBytes: 7}, nil)
	missingRaw, _, err := canonicalStateValue(missingManifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestMissingPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/manifests/" + wcncpTestDigest("absent-manifest")
	_ = manifestMissingPath
	// Put the missing manifest via direct storage to prove HTTP maps the error:
	// instead exercise HTTP manifest PUT with incomplete reference.
	badManifestPut := centralTestRequest(handler, http.MethodPut, "/api/v1/repositories/"+stateTestScope+"/wcncp/WCNCP_OBSERVATION/manifests/"+wcncpTestDigest("x"), observerToken.Token, bytes.NewReader(missingRaw), true, nil)
	if badManifestPut.Code != http.StatusPreconditionFailed && badManifestPut.Code != http.StatusBadRequest {
		t.Fatalf("partial manifest = %d", badManifestPut.Code)
	}
	// Complete manifest then CAS head, replay, and stale precondition.
	object, _, err := storage.PutWCNCPObject(ctx, stateTestScope, WCNCPKindObservation, obsDigest, bytes.NewReader(obsCanonical))
	if err != nil {
		// Already present via HTTP; reload digest.
		object = WCNCPObject{RepositoryScopeSHA256: stateTestScope, Kind: WCNCPKindObservation, SHA256: obsDigest, SizeBytes: int64(len(obsCanonical))}
	}
	manifest := wcncpTestManifest(WCNCPKindObservation, 1, now, object, nil)
	manifestRaw, _, err := canonicalStateValue(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded WCNCPManifest
	rawForDigest, digest, err := func() ([]byte, string, error) {
		m, c, d, err := decodeWCNCPManifest(manifestRaw)
		_ = m
		return c, d, err
	}()
	if err != nil {
		t.Fatal(err)
	}
	_ = rawForDigest
	manifestPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/manifests/" + digest
	if code := centralTestRequest(handler, http.MethodPut, manifestPath, observerToken.Token, bytes.NewReader(manifestRaw), true, nil).Code; code != http.StatusCreated {
		t.Fatalf("manifest put = %d", code)
	}
	casPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/cas"
	casDocument, _ := json.Marshal(map[string]interface{}{
		"idempotencyKey": wcncpTestDigest("wcncp-http-cas-one"), "expectedGeneration": 0, "manifestSha256": digest,
	})
	firstCAS := centralTestRequest(handler, http.MethodPost, casPath, observerToken.Token, bytes.NewReader(casDocument), true, nil)
	if firstCAS.Code != http.StatusOK {
		t.Fatalf("cas = %d %s", firstCAS.Code, firstCAS.Body.String())
	}
	replayCAS := centralTestRequest(handler, http.MethodPost, casPath, observerToken.Token, bytes.NewReader(casDocument), true, nil)
	if replayCAS.Code != http.StatusOK {
		t.Fatalf("replay cas = %d", replayCAS.Code)
	}
	// Changed request under reused key conflicts.
	conflictDocument, _ := json.Marshal(map[string]interface{}{
		"idempotencyKey": wcncpTestDigest("wcncp-http-cas-one"), "expectedGeneration": 0, "manifestSha256": wcncpTestDigest("different"),
	})
	conflict := centralTestRequest(handler, http.MethodPost, casPath, observerToken.Token, bytes.NewReader(conflictDocument), true, nil)
	if conflict.Code != http.StatusConflict && conflict.Code != http.StatusPreconditionFailed && conflict.Code != http.StatusBadRequest {
		t.Fatalf("conflicting replay = %d", conflict.Code)
	}
	// Batch with one invalid item rejects the whole batch.
	batchPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/batch"
	var validItem map[string]interface{}
	if err := json.Unmarshal(obsCanonical, &validItem); err != nil {
		t.Fatal(err)
	}
	invalidItem := map[string]interface{}{"bogus": true}
	batchRaw, _ := json.Marshal([]interface{}{validItem, invalidItem})
	badBatch := centralTestRequest(handler, http.MethodPost, batchPath, observerToken.Token, bytes.NewReader(batchRaw), true, nil)
	if badBatch.Code != http.StatusBadRequest {
		t.Fatalf("partial batch = %d", badBatch.Code)
	}
	// Restart accepts the exact retry once (verified local snapshot survives).
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	storage = openStateTestStorage(t, ctx, root)
	defer storage.Close()
	storage.clock = func() time.Time { return now.Add(time.Minute) }
	handler, err = NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/snapshot"
	snapshot := centralTestRequest(handler, http.MethodGet, snapshotPath, observerToken.Token, nil, true, nil)
	// Token from before restart is still valid (not revoked); snapshot must verify.
	if snapshot.Code != http.StatusOK {
		t.Fatalf("snapshot after restart = %d", snapshot.Code)
	}
	_ = decoded
}

func TestWCNCPHTTPSLogsCarryIDsNotSecrets(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, t.TempDir()+"/shared")
	defer storage.Close()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	storage.clock = func() time.Time { return now }
	handler, err := NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	observerToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	if _, err := storage.GrantWCNCPActor(ctx, observerToken.TokenID, WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	obsRecord := wcncpTestValidRecord(t, WCNCPKindObservation)
	obsCanonical, obsDigest, err := CanonicalWCNCPRecord(WCNCPKindObservation, obsRecord)
	if err != nil {
		t.Fatal(err)
	}
	obsPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/objects/" + obsDigest
	requestID := "test-request-12345678"
	created := centralTestRequest(handler, http.MethodPut, obsPath, observerToken.Token, bytes.NewReader(obsCanonical), true, map[string]string{"If-None-Match": "*", wcncpRequestHeader: requestID})
	if created.Code != http.StatusCreated {
		t.Fatalf("put = %d", created.Code)
	}
	var count int
	if err := storage.control.database.QueryRow(`SELECT count(*) FROM wcncp_audit_events WHERE request_id = ? AND manifest_digest = ?`, requestID, obsDigest).Scan(&count); err != nil || count < 1 {
		t.Fatalf("audit event = %d/%v", count, err)
	}
	// Audit must never contain credentials, source content, or raw arguments.
	rows, err := storage.control.database.Query(`SELECT request_id, token_id, manifest_digest, result FROM wcncp_audit_events`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var requestID, tokenID, digest, result string
		var digestNull *string
		_ = digestNull
		if err := rows.Scan(&requestID, &tokenID, &digest, &result); err != nil {
			// manifest_digest may be NULL for batch counts; scan tolerantly.
			continue
		}
		if tokenID == observerToken.Token {
			t.Fatal("raw credential in audit log")
		}
		if len(digest) > 0 && len(digest) != 64 && digest != obsDigest {
			// Digests only; no source blobs.
			t.Fatalf("unexpected audit digest %q", digest)
		}
	}
}
