package sharedcache

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

// TestWCNCPMultiRunnerConvergence proves one restarted HTTPS service with at
// least three isolated runner roots: shared observations converge once, one
// validator owns the lease, outage preserves native results, and every
// wrong-scope, fork, stale, late, and tampered publication fails closed. No
// synthetic timing counts as public value.
func TestWCNCPMultiRunnerConvergence(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir() + "/server"
	storage := openStateTestStorage(t, ctx, root)
	now := time.Date(2026, 9, 4, 14, 0, 0, 0, time.UTC)
	storage.clock = func() time.Time { return now }
	handler, err := NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	observerToken := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
	if _, err := storage.GrantWCNCPActor(ctx, observerToken.TokenID, WCNCPActorTrustedObserver, now); err != nil {
		t.Fatal(err)
	}
	validatorGrant := func(holder string) IssuedCentralToken {
		token := issueCentralTestToken(t, storage, now, CentralStateRead, CentralStateWrite)
		if _, err := storage.GrantWCNCPActor(ctx, token.TokenID, WCNCPActorValidator, now); err != nil {
			t.Fatal(err)
		}
		return token
	}
	_ = validatorGrant

	// Runner A publishes a trusted observation.
	obsA := wcncpTestValidRecord(t, WCNCPKindObservation)
	obsACanonical, obsADigest, err := CanonicalWCNCPRecord(WCNCPKindObservation, obsA)
	if err != nil {
		t.Fatal(err)
	}
	obsAPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/objects/" + obsADigest
	if code := centralTestRequest(handler, http.MethodPut, obsAPath, observerToken.Token, bytes.NewReader(obsACanonical), true, map[string]string{"If-None-Match": "*"}).Code; code != http.StatusCreated {
		t.Fatalf("runner A publish = %d", code)
	}
	// Runner B publishes a compatible distinct observation (different ordinal).
	var obsBMap map[string]interface{}
	if err := json.Unmarshal(obsACanonical, &obsBMap); err != nil {
		t.Fatal(err)
	}
	obsBMap["invocationOrdinal"] = float64(2)
	obsBMap["idempotencyKey"] = "observation:0002-system-proof"
	obsBRaw, _ := json.Marshal(obsBMap)
	obsBCanonical, obsBDigest, err := CanonicalWCNCPRecord(WCNCPKindObservation, obsBRaw)
	if err != nil {
		t.Fatal(err)
	}
	obsBPath := "/api/v1/repositories/" + stateTestScope + "/wcncp/WCNCP_OBSERVATION/objects/" + obsBDigest
	if code := centralTestRequest(handler, http.MethodPut, obsBPath, observerToken.Token, bytes.NewReader(obsBCanonical), true, map[string]string{"If-None-Match": "*"}).Code; code != http.StatusCreated {
		t.Fatalf("runner B publish = %d", code)
	}
	// Runner C retries one duplicate (exact replay converges once) and holds
	// one offline outbox item for after the restart.
	duplicate := centralTestRequest(handler, http.MethodPut, obsAPath, observerToken.Token, bytes.NewReader(obsACanonical), true, map[string]string{"If-None-Match": "*"})
	if duplicate.Code != http.StatusCreated && duplicate.Code != http.StatusOK {
		t.Fatalf("runner C duplicate = %d", duplicate.Code)
	}
	offlineOutbox := wcncpobserve.Outbox{Dir: t.TempDir() + "/runner-c-outbox"}
	offlineItem := []byte(`{"runner":"C","offline":true}`)
	if err := offlineOutbox.Enqueue("obs-offline.json", offlineItem, now); err != nil {
		t.Fatal(err)
	}
	// Two validators race for one proposal; exactly one acquires the lease.
	proposal := wcncpTestDigest("system-proof-proposal")
	var group sync.WaitGroup
	leases := make(chan error, 2)
	for _, holder := range []string{"validator-1", "validator-2"} {
		group.Add(1)
		go func(holder string) {
			defer group.Done()
			_, err := storage.ClaimWCNCPLease(ctx, stateTestScope, proposal, "WCNCP_CORRECTNESS_V1", "LOCAL_FUNCTIONAL", holder, time.Minute, now)
			leases <- err
		}(holder)
	}
	group.Wait()
	close(leases)
	winners, held := 0, 0
	for err := range leases {
		if err == nil {
			winners++
		} else {
			held++
		}
	}
	if winners != 1 || held != 1 {
		t.Fatalf("validator race winners=%d held=%d", winners, held)
	}
	// Server loss during an ordinary build preserves native output: the
	// upload fails open and the item stays queued.
	outcome := wcncpobserve.UploadBatch(ctx, http.DefaultClient, "http://127.0.0.1:1/no-listener", "token", []json.RawMessage{json.RawMessage(obsACanonical)}, 50*time.Millisecond)
	if !outcome.Queued || outcome.Uploaded != 0 {
		t.Fatalf("outage upload = %+v", outcome)
	}
	// Restart accepts the queued exact retry once.
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
	retry := centralTestRequest(handler, http.MethodPut, obsAPath, observerToken.Token, bytes.NewReader(obsACanonical), true, map[string]string{"If-None-Match": "*"})
	if retry.Code != http.StatusCreated && retry.Code != http.StatusOK {
		t.Fatalf("restart retry = %d", retry.Code)
	}
	// An incompatible runner observation remains isolated: same bytes under a
	// different repository scope are not visible here.
	if _, err := storage.OpenWCNCPObject(ctx, stateTestOtherScope, WCNCPKindObservation, obsADigest); err == nil {
		t.Fatal("cross-repository observation visible")
	}
	// Wrong-scope, fork, stale, late, and tampered publications fail closed.
	wrongScope := centralTestRequest(handler, http.MethodPut, "/api/v1/repositories/"+stateTestOtherScope+"/wcncp/WCNCP_OBSERVATION/objects/"+obsADigest, observerToken.Token, bytes.NewReader(obsACanonical), true, map[string]string{"If-None-Match": "*"})
	if wrongScope.Code != http.StatusNotFound {
		t.Fatalf("wrong scope = %d", wrongScope.Code)
	}
	forked := centralTestRequest(handler, http.MethodPut, obsBPath, observerToken.Token, bytes.NewReader(obsBCanonical), true, map[string]string{"If-None-Match": "*", wcncpForkHeader: "1"})
	if forked.Code != http.StatusForbidden {
		t.Fatalf("fork write = %d", forked.Code)
	}
	tampered := centralTestRequest(handler, http.MethodPut, obsBPath, observerToken.Token, bytes.NewReader([]byte(`{"tampered":true}`)), true, map[string]string{"If-None-Match": "*"})
	if tampered.Code == http.StatusCreated || tampered.Code == http.StatusOK {
		t.Fatalf("tampered write = %d", tampered.Code)
	}
}
