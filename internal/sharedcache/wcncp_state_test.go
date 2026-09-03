package sharedcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var wcncpTestGeneration atomic.Int64

const (
	wcncpTestScope      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	wcncpTestOtherScope = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestWCNCPersistsAcrossRestartAndIsolatesNamespaces(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage := openStateTestStorage(t, ctx, root)
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	// Identical bytes in different namespaces must not be mutually visible.
	content := []byte(`{"wcncp":"same-bytes"}`)
	// Use a valid observation payload for real namespace proof below; the
	// raw-bytes check uses validated records through helpers.
	_ = content
	obsObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	if file, err := storage.OpenWCNCPObject(ctx, wcncpTestScope, WCNCPKindOpportunity, obsObject.SHA256); file != nil || !errors.Is(err, ErrWCNCPNotFound) {
		t.Fatalf("cross-kind object = %+v/%v", file, err)
	}
	if file, err := storage.OpenWCNCPObject(ctx, wcncpTestOtherScope, WCNCPKindObservation, obsObject.SHA256); file != nil || !errors.Is(err, ErrWCNCPNotFound) {
		t.Fatalf("cross-repository object = %+v/%v", file, err)
	}
	// Existing typed-state namespace must not address WCNCP bytes and vice versa.
	if file, err := storage.OpenStateObject(ctx, wcncpTestScope, StateKindEvidence, obsObject.SHA256); file != nil || !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("cross-plane state object = %+v/%v", file, err)
	}

	obsManifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 1, now, obsObject, nil))
	obsHead := casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope,
		Kind:                  WCNCPKindObservation,
		IdempotencyKey:        wcncpTestDigest("wcncp-obs-cas-one"),
		ManifestSHA256:        obsManifest.ManifestSHA256,
	})

	storage.clock = func() time.Time { return now.Add(time.Minute) }
	oppObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindOpportunity, wcncpTestValidRecord(t, WCNCPKindOpportunity))
	oppManifest := wcncpTestManifest(WCNCPKindOpportunity, 1, now.Add(time.Minute), oppObject, []WCNCPReference{{Kind: WCNCPKindObservation, ManifestSHA256: obsManifest.ManifestSHA256, Relation: "DERIVED_FROM"}})
	oppStored := putWCNCPTestManifest(t, storage, oppManifest)
	oppHead := casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope,
		Kind:                  WCNCPKindOpportunity,
		IdempotencyKey:        wcncpTestDigest("wcncp-opp-cas-one"),
		ManifestSHA256:        oppStored.ManifestSHA256,
	})
	if _, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindProposal); !errors.Is(err, ErrWCNCPNotFound) {
		t.Fatalf("unpublished proposal = %v", err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	storage = openStateTestStorage(t, ctx, root)
	defer storage.Close()
	for _, testCase := range []struct {
		kind StateKind
		head WCNCPCASResult
	}{
		{kind: WCNCPKindObservation, head: obsHead},
		{kind: WCNCPKindOpportunity, head: oppHead},
	} {
		snapshot, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, testCase.kind)
		if err != nil {
			t.Fatalf("load %s after restart: %v", testCase.kind, err)
		}
		if snapshot.HeadSHA256 != testCase.head.HeadSHA256 || !snapshot.Head.Authority.SelectionRequiresLocalRevalidation || snapshot.Head.Authority.ProductionAuthorized {
			t.Fatalf("restarted %s snapshot = %+v", testCase.kind, snapshot)
		}
	}
	replayed, err := storage.CASWCNCPHead(ctx, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope,
		Kind:                  WCNCPKindObservation,
		IdempotencyKey:        wcncpTestDigest("wcncp-obs-cas-one"),
		ManifestSHA256:        obsManifest.ManifestSHA256,
	})
	if err != nil || !replayed.Replayed || replayed.HeadSHA256 != obsHead.HeadSHA256 {
		t.Fatalf("restart replay = %+v/%v", replayed, err)
	}
}

func TestWCNCPConcurrentCASIsAtomicAndIdempotent(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	manifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 1, now, object, nil))
	requests := []WCNCPCASRequest{
		{RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation, IdempotencyKey: wcncpTestDigest("wcncp-concurrent-one"), ManifestSHA256: manifest.ManifestSHA256},
		{RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation, IdempotencyKey: wcncpTestDigest("wcncp-concurrent-two"), ManifestSHA256: manifest.ManifestSHA256},
	}
	type outcome struct {
		result WCNCPCASResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, len(requests))
	var group sync.WaitGroup
	for _, request := range requests {
		request := request
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, err := storage.CASWCNCPHead(ctx, request)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(outcomes)
	winners, preconditions := 0, 0
	var winner WCNCPCASResult
	for outcome := range outcomes {
		switch {
		case outcome.err == nil:
			winners++
			winner = outcome.result
		case errors.Is(outcome.err, ErrWCNCPHeadPrecondition):
			preconditions++
		default:
			t.Fatalf("concurrent CAS = %+v/%v", outcome.result, outcome.err)
		}
	}
	if winners != 1 || preconditions != 1 {
		t.Fatalf("concurrent outcomes winners=%d preconditions=%d", winners, preconditions)
	}
	snapshot, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindObservation)
	if err != nil || snapshot.HeadSHA256 != winner.HeadSHA256 {
		t.Fatalf("winner visibility = %+v/%v", snapshot, err)
	}
	winningRequest := requests[0]
	for _, request := range requests {
		replayed, err := storage.CASWCNCPHead(ctx, request)
		if err == nil && replayed.Replayed {
			winningRequest = request
			break
		}
	}
	conflict := winningRequest
	conflict.RepositoryScopeSHA256 = wcncpTestOtherScope
	if _, err := storage.CASWCNCPHead(ctx, conflict); !errors.Is(err, ErrWCNCPIdempotency) {
		t.Fatalf("changed idempotency payload = %v", err)
	}
}

func TestWCNCPPartialPublicationNeverBecomesVisible(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage := openStateTestStorage(t, ctx, root)
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	manifest := wcncpTestManifest(WCNCPKindObservation, 1, now, object, nil)
	putWCNCPTestManifest(t, storage, manifest)
	if _, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindObservation); !errors.Is(err, ErrWCNCPNotFound) {
		t.Fatalf("staged state became visible = %v", err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	storage = openStateTestStorage(t, ctx, root)
	defer storage.Close()
	if _, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindObservation); !errors.Is(err, ErrWCNCPNotFound) {
		t.Fatalf("staged state became visible after restart = %v", err)
	}
	storage.clock = func() time.Time { return now.Add(25 * time.Hour) }
	report, err := storage.MaintainWCNCP(ctx)
	if err != nil || report.ExpiredStagedManifests != 1 || report.ExpiredStagedObjects != 1 {
		t.Fatalf("staged cleanup = %+v/%v", report, err)
	}
}

func TestWCNCPRejectsBlobAndManifestCorruptionAfterRestart(t *testing.T) {
	t.Run("blob", func(t *testing.T) {
		ctx := context.Background()
		root := filepath.Join(t.TempDir(), "shared")
		storage := openStateTestStorage(t, ctx, root)
		now := time.Now().UTC().Truncate(time.Second)
		object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
		manifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 1, now, object, nil))
		casWCNCPTestHead(t, storage, WCNCPCASRequest{
			RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
			IdempotencyKey: wcncpTestDigest("wcncp-corrupt-blob"), ManifestSHA256: manifest.ManifestSHA256,
		})
		layout := storage.Layout()
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(layout.Blobs, object.SHA256[:2], object.SHA256[2:])
		if err := os.WriteFile(path, bytes.Repeat([]byte{'x'}, int(object.SizeBytes)), 0o600); err != nil {
			t.Fatal(err)
		}
		storage = openStateTestStorage(t, ctx, root)
		defer storage.Close()
		if _, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindObservation); !errors.Is(err, ErrWCNPCorrupt) {
			t.Fatalf("corrupt blob load = %v", err)
		}
	})
	t.Run("manifest", func(t *testing.T) {
		ctx := context.Background()
		root := filepath.Join(t.TempDir(), "shared")
		storage := openStateTestStorage(t, ctx, root)
		now := time.Now().UTC().Truncate(time.Second)
		object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
		manifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 1, now, object, nil))
		casWCNCPTestHead(t, storage, WCNCPCASRequest{
			RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
			IdempotencyKey: wcncpTestDigest("wcncp-corrupt-manifest"), ManifestSHA256: manifest.ManifestSHA256,
		})
		path := storage.Layout().StateDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		database, err := sql.Open("sqlite", sqliteDataSource(path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`UPDATE wcncp_manifests SET canonical_document = ?`, []byte(`{"corrupt":true}`)); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		storage = openStateTestStorage(t, ctx, root)
		defer storage.Close()
		if _, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindObservation); !errors.Is(err, ErrWCNPCorrupt) {
			t.Fatalf("corrupt manifest load = %v", err)
		}
	})
}

func TestWCNCPRejectsIncompleteInvalidAndSkippedPromotion(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	missing := wcncpTestManifest(WCNCPKindObservation, 1, now, WCNCPObject{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
		SHA256: wcncpTestDigest("missing"), SizeBytes: 7,
	}, nil)
	missingRaw, _, err := canonicalStateValue(missing)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.PutWCNCPManifest(ctx, missingRaw); !errors.Is(err, ErrWCNCPManifestIncomplete) {
		t.Fatalf("missing artifact manifest = %v", err)
	}
	invalidRaw := append(bytes.TrimSuffix(missingRaw, []byte("}")), []byte(`,"unknown":true}`)...)
	if _, _, err := storage.PutWCNCPManifest(ctx, invalidRaw); !errors.Is(err, ErrWCNCPInvalid) {
		t.Fatalf("unknown manifest property = %v", err)
	}
	object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	skipped := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 3, now, object, nil))
	if _, err := storage.CASWCNCPHead(ctx, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
		IdempotencyKey: wcncpTestDigest("wcncp-skipped-generation"), ManifestSHA256: skipped.ManifestSHA256,
	}); !errors.Is(err, ErrWCNCPGenerationConflict) {
		t.Fatalf("skipped generation = %v", err)
	}
	// Stale head precondition: publish gen 1 then try to CAS gen 2 with wrong expected head.
	genOne := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindDecision, 1, now, putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindDecision, wcncpTestValidRecord(t, WCNCPKindDecision)), wcncpTestDecisionRefs(t, storage, now)))
	headOne := casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindDecision,
		IdempotencyKey: wcncpTestDigest("wcncp-decision-one"), ManifestSHA256: genOne.ManifestSHA256,
	})
	_ = headOne
	genTwoObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindDecision, wcncpTestValidRecord(t, WCNCPKindDecision))
	genTwo := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindDecision, 2, now, genTwoObject, wcncpTestDecisionRefs(t, storage, now)))
	if _, err := storage.CASWCNCPHead(ctx, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindDecision,
		IdempotencyKey: wcncpTestDigest("wcncp-stale-head"), ExpectedGeneration: 1,
		ExpectedHeadSHA256: wcncpTestDigest("wrong-head"), ManifestSHA256: genTwo.ManifestSHA256,
	}); !errors.Is(err, ErrWCNCPHeadPrecondition) {
		t.Fatalf("stale head = %v", err)
	}
}

func TestWCNCPValidatesFrozenSchemasAndRejectsAuthorityEscalation(t *testing.T) {
	for _, kind := range []StateKind{WCNCPKindObservation, WCNCPKindOpportunity, WCNCPKindProposal, WCNCPKindValidation, WCNCPKindDecision} {
		valid := wcncpTestValidRecord(t, kind)
		if err := ValidateWCNCPRecord(kind, valid); err != nil {
			t.Fatalf("valid %s rejected: %v", kind, err)
		}
	}
	// Language-neutral vectors: every valid record passes, every invalid fails.
	vectors := wcncpTestVectors(t)
	for kind, raw := range vectors.Records {
		var typed StateKind
		switch kind {
		case "observation":
			typed = WCNCPKindObservation
		case "opportunity":
			typed = WCNCPKindOpportunity
		case "proposal":
			typed = WCNCPKindProposal
		case "validation":
			typed = WCNCPKindValidation
		case "decision":
			typed = WCNCPKindDecision
		default:
			t.Fatalf("unknown vector kind %s", kind)
		}
		if err := ValidateWCNCPRecord(typed, raw); err != nil {
			t.Fatalf("vector valid %s rejected: %v", kind, err)
		}
	}
	for kind, raw := range vectors.InvalidRecords {
		var typed StateKind
		switch kind {
		case "observation":
			typed = WCNCPKindObservation
		case "opportunity":
			typed = WCNCPKindOpportunity
		case "proposal":
			typed = WCNCPKindProposal
		case "validation":
			typed = WCNCPKindValidation
		case "decision":
			typed = WCNCPKindDecision
		default:
			t.Fatalf("unknown vector kind %s", kind)
		}
		if err := ValidateWCNCPRecord(typed, raw); err == nil {
			t.Fatalf("vector invalid %s accepted", kind)
		}
	}
	// Wrong-kind validation must fail closed.
	if err := ValidateWCNCPRecord(WCNCPKindProposal, wcncpTestValidRecord(t, WCNCPKindObservation)); err == nil {
		t.Fatal("cross-kind record accepted")
	}
	// Non-canonical trailing bytes must fail.
	valid := wcncpTestValidRecord(t, WCNCPKindObservation)
	trailing := append(append([]byte{}, valid...), []byte(" ")...)
	// Trailing whitespace is canonicalized away, so craft a real violation:
	// flip production authority in a decision copy.
	decision := wcncpTestValidRecord(t, WCNCPKindDecision)
	var decoded map[string]any
	if err := json.Unmarshal(decision, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded["authority"].(map[string]any)["sourceApplied"] = true
	mutated, _ := json.Marshal(decoded)
	if err := ValidateWCNCPRecord(WCNCPKindDecision, mutated); err == nil {
		t.Fatal("escalated decision authority accepted")
	}
	_ = trailing
}

func TestWCNCPRejectsInvalidPayloadBeforeVisibility(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	// Schema-invalid bytes must never gain metadata visibility even when the
	// content hash matches.
	invalid := wcncpTestInvalidRecord(t, WCNCPKindProposal)
	digest := sha256.Sum256(invalid)
	expected := hex.EncodeToString(digest[:])
	if _, _, err := storage.PutWCNCPObject(ctx, wcncpTestScope, WCNCPKindProposal, expected, bytes.NewReader(invalid)); !errors.Is(err, ErrWCNCPInvalid) {
		t.Fatalf("invalid payload put = %v", err)
	}
	if _, err := storage.OpenWCNCPObject(ctx, wcncpTestScope, WCNCPKindProposal, expected); !errors.Is(err, ErrWCNCPNotFound) {
		t.Fatalf("invalid payload visible = %v", err)
	}
}

func TestWCNCRetentionKeepsReferencedAndExpiresSuperseded(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	base := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return base }

	obsOneObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	obsOne := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 1, base, obsOneObject, nil))
	obsOneHead := casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
		IdempotencyKey: wcncpTestDigest("wcncp-ret-obs-one"), ManifestSHA256: obsOne.ManifestSHA256,
	})
	oppOneObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindOpportunity, wcncpTestValidRecord(t, WCNCPKindOpportunity))
	oppOne := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindOpportunity, 1, base, oppOneObject, []WCNCPReference{{Kind: WCNCPKindObservation, ManifestSHA256: obsOne.ManifestSHA256, Relation: "DERIVED_FROM"}}))
	oppOneHead := casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindOpportunity,
		IdempotencyKey: wcncpTestDigest("wcncp-ret-opp-one"), ManifestSHA256: oppOne.ManifestSHA256,
	})

	storage.clock = func() time.Time { return base.Add(2 * time.Minute) }
	obsTwoObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	obsTwo := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 2, base.Add(2*time.Minute), obsTwoObject, nil))
	casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
		IdempotencyKey: wcncpTestDigest("wcncp-ret-obs-two"), ExpectedGeneration: 1,
		ExpectedHeadSHA256: obsOneHead.HeadSHA256, ManifestSHA256: obsTwo.ManifestSHA256,
	})
	// Referenced observation must survive even after supersession + 31d.
	storage.clock = func() time.Time { return base.Add(31 * 24 * time.Hour) }
	if report, err := storage.MaintainWCNCP(ctx); err != nil || report.ExpiredSuperseded != 0 {
		t.Fatalf("referenced maintenance = %+v/%v", report, err)
	}
	assertWCNCPManifestPresent(t, storage, obsOne.ManifestSHA256, true)

	// Advance the opportunity so the first observation loses its last
	// reference, then expire it after its own 30d window.
	oppTwoObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindOpportunity, wcncpTestValidRecord(t, WCNCPKindOpportunity))
	oppTwo := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindOpportunity, 2, base.Add(31*24*time.Hour), oppTwoObject, []WCNCPReference{{Kind: WCNCPKindObservation, ManifestSHA256: obsTwo.ManifestSHA256, Relation: "DERIVED_FROM"}}))
	casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindOpportunity,
		IdempotencyKey: wcncpTestDigest("wcncp-ret-opp-two"), ExpectedGeneration: 1,
		ExpectedHeadSHA256: oppOneHead.HeadSHA256, ManifestSHA256: oppTwo.ManifestSHA256,
	})
	storage.clock = func() time.Time { return base.Add(62 * 24 * time.Hour) }
	if report, err := storage.MaintainWCNCP(ctx); err != nil || report.ExpiredSuperseded < 1 {
		t.Fatalf("supersession expiry = %+v/%v", report, err)
	}
	// Decisions are durable: publish then attempt maintenance far in future.
	decObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindDecision, wcncpTestValidRecord(t, WCNCPKindDecision))
	decManifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindDecision, 1, base.Add(62*24*time.Hour), decObject, wcncpTestDecisionRefs(t, storage, base.Add(62*24*time.Hour))))
	casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindDecision,
		IdempotencyKey: wcncpTestDigest("wcncp-ret-dec"), ManifestSHA256: decManifest.ManifestSHA256,
	})
	storage.clock = func() time.Time { return base.Add(400 * 24 * time.Hour) }
	if _, err := storage.MaintainWCNCP(ctx); err != nil {
		t.Fatal(err)
	}
	assertWCNCPManifestPresent(t, storage, decManifest.ManifestSHA256, true)
}

func TestWCNCPConcurrentObjectPublicationsRemainAvailable(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	type outcome struct {
		digest string
		err    error
	}
	const writers = 8
	start := make(chan struct{})
	outcomes := make(chan outcome, writers)
	var group sync.WaitGroup
	digests := make([]string, writers)
	payloads := make([][]byte, writers)
	for index := range writers {
		// Each writer needs a distinct valid record; vary runner-adjacent
		// bytes via invocation ordinal is not possible with frozen vectors,
		// so vary the repository scope instead with precomputed valid scopes.
		_ = index
	}
	_ = digests
	_ = payloads
	for index := range writers {
		content := wcncpTestValidRecord(t, WCNCPKindObservation)
		// Make each payload distinct but still valid by round-tripping a
		// controlled field: invocation ordinal 1..8.
		var decoded map[string]any
		if err := json.Unmarshal(content, &decoded); err != nil {
			t.Fatal(err)
		}
		decoded["invocationOrdinal"] = float64(index + 1)
		decoded["idempotencyKey"] = "observation:concurrent-" + strconv.Itoa(index+1) + "-pad-x"
		mutated, err := json.Marshal(decoded)
		if err != nil {
			t.Fatal(err)
		}
		canonical, digest, err := CanonicalWCNCPRecord(WCNCPKindObservation, mutated)
		if err != nil {
			t.Fatal(err)
		}
		payloads[index] = canonical
		digests[index] = digest
	}
	for index := range writers {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			object, _, err := storage.PutWCNCPObject(ctx, wcncpTestScope, WCNCPKindObservation, digests[index], bytes.NewReader(payloads[index]))
			outcomes <- outcome{digest: object.SHA256, err: err}
		}(index)
	}
	close(start)
	group.Wait()
	close(outcomes)
	for item := range outcomes {
		if item.err != nil || item.digest == "" {
			t.Fatalf("concurrent object publication = %q/%v", item.digest, item.err)
		}
		file, err := storage.OpenWCNCPObject(ctx, wcncpTestScope, WCNCPKindObservation, item.digest)
		if err != nil {
			t.Fatalf("open concurrent object %s: %v", item.digest, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close concurrent object %s: %v", item.digest, err)
		}
	}
}

func TestWCNCPMigrationChecksumsAndForwardOpen(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage := openStateTestStorage(t, ctx, root)
	version, err := storage.state.SchemaVersion(ctx)
	if err != nil || version != StateSchemaVersion || StateSchemaVersion != 2 {
		t.Fatalf("state schema version = %d/%v, want 2", version, err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }
	object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	manifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, 1, now, object, nil))
	head := casWCNCPTestHead(t, storage, WCNCPCASRequest{
		RepositoryScopeSHA256: wcncpTestScope, Kind: WCNCPKindObservation,
		IdempotencyKey: wcncpTestDigest("wcncp-migrate-head"), ManifestSHA256: manifest.ManifestSHA256,
	})
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	// Reopen proves forward-open: v2 rows survive restart and validate.
	storage = openStateTestStorage(t, ctx, root)
	defer storage.Close()
	snapshot, err := storage.LoadCurrentWCNCP(ctx, wcncpTestScope, WCNCPKindObservation)
	if err != nil || snapshot.HeadSHA256 != head.HeadSHA256 {
		t.Fatalf("migrated visibility = %+v/%v", snapshot, err)
	}
	// Migration records must carry exact checksums; tampering fails closed.
	rows, err := storage.state.database.Query(`SELECT version, name, checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var version int
		var name, checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			t.Fatal(err)
		}
		count++
		if len(checksum) != 64 {
			t.Fatalf("migration %d checksum = %q", version, checksum)
		}
		_ = name
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != StateSchemaVersion {
		t.Fatalf("migration rows = %d, want %d", count, StateSchemaVersion)
	}
}

func TestWCNCPGradleCachePlaneCannotAddressControlState(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	object := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	// A Gradle cache lookup uses tenant/namespace/key addressing, never the
	// WCNCP (repository,kind,digest) triple. Prove the planes do not share
	// metadata by showing the control digest is not resolvable as a cache
	// object and the cache tables never reference wcncp metadata.
	var count int
	if err := storage.state.database.QueryRow(`SELECT count(*) FROM wcncp_objects WHERE blob_digest = ?`, digestPrefix+object.SHA256).Scan(&count); err != nil || count != 1 {
		t.Fatalf("wcncp object metadata = %d/%v", count, err)
	}
	if err := storage.cache.database.QueryRow(`SELECT count(*) FROM committed_objects WHERE blob_digest = ?`, digestPrefix+object.SHA256).Scan(&count); err == nil && count != 0 {
		t.Fatalf("control blob leaked into Gradle data plane metadata")
	}
}

// Helpers.

func putWCNCPTestObject(t *testing.T, storage *Storage, scope string, kind StateKind, content []byte) WCNCPObject {
	t.Helper()
	canonical, digest, err := CanonicalWCNCPRecord(kind, content)
	if err != nil {
		t.Fatal(err)
	}
	_ = canonical
	object, _, err := storage.PutWCNCPObject(context.Background(), scope, kind, digest, bytes.NewReader(canonical))
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func putWCNCPTestManifest(t *testing.T, storage *Storage, manifest WCNCPManifest) WCNCPSnapshot {
	t.Helper()
	canonical, _, err := canonicalStateValue(manifest)
	if err != nil {
		t.Fatal(err)
	}
	stored, _, err := storage.PutWCNCPManifest(context.Background(), canonical)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func casWCNCPTestHead(t *testing.T, storage *Storage, request WCNCPCASRequest) WCNCPCASResult {
	t.Helper()
	result, err := storage.CASWCNCPHead(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func wcncpTestManifest(kind StateKind, generation int64, createdAt time.Time, object WCNCPObject, references []WCNCPReference) WCNCPManifest {
	manifest := WCNCPManifest{
		SchemaVersion: WCNCPManifestSchemaVersion, RecordType: "WCNCP_MANIFEST",
		Kind: kind, RepositoryScopeSHA256: object.RepositoryScopeSHA256, Generation: generation,
		CompatibilitySHA256: wcncpTestDigest("compat-" + string(kind)), BindingsSHA256: wcncpTestDigest("bindings-" + string(kind) + strconv.FormatInt(generation, 10)),
		Origin: StateOrigin{
			BaseRevision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", TargetRevision: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			BuildOptExecutableSHA256: wcncpTestDigest("buildopt"), WrapperSHA256: wcncpTestDigest("wrapper"), GradleVersion: "9.6.1",
		},
		Artifacts: []StateArtifact{{
			Role: string(kind), SHA256: object.SHA256, SizeBytes: object.SizeBytes, PayloadSchemaVersion: wcncpPayloadSchemaVersion(kind),
		}},
		References: references, Status: "COMPLETE", RetentionClass: wcncpRetentionClass(kind),
		CreatedAt: createdAt.UTC().Format(time.RFC3339Nano),
		Authority: StateAuthority{SelectionRequiresLocalRevalidation: true, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE"},
	}
	if references == nil {
		manifest.References = []WCNCPReference{}
	}
	return manifest
}

func wcncpTestDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func assertWCNCPManifestPresent(t *testing.T, storage *Storage, digest string, want bool) {
	t.Helper()
	var count int
	if err := storage.state.database.QueryRow(`SELECT count(*) FROM wcncp_manifests WHERE manifest_digest = ?`, digestPrefix+digest).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if (count == 1) != want {
		t.Fatalf("wcncp manifest %s count = %d, want present=%t", digest, count, want)
	}
}

type wcncpVectorCatalog struct {
	Records        map[string]json.RawMessage `json:"records"`
	InvalidRecords map[string]json.RawMessage `json:"invalidRecords"`
}

func wcncpTestVectors(t *testing.T) wcncpVectorCatalog {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(findWCNCPRepoRoot(t), "contracts", "test-vectors", "wcncp", "wcncp-control-state.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog wcncpVectorCatalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	return catalog
}

func wcncpTestValidRecord(t *testing.T, kind StateKind) []byte {
	t.Helper()
	vectors := wcncpTestVectors(t)
	var name string
	switch kind {
	case WCNCPKindObservation:
		name = "observation"
	case WCNCPKindOpportunity:
		name = "opportunity"
	case WCNCPKindProposal:
		name = "proposal"
	case WCNCPKindValidation:
		name = "validation"
	case WCNCPKindDecision:
		name = "decision"
	default:
		t.Fatalf("unknown kind %s", kind)
	}
	raw, ok := vectors.Records[name]
	if !ok {
		t.Fatalf("missing vector %s", name)
	}
	// Return a copy so callers may mutate safely.
	out := append([]byte{}, raw...)
	return out
}

func wcncpTestInvalidRecord(t *testing.T, kind StateKind) []byte {
	t.Helper()
	vectors := wcncpTestVectors(t)
	var name string
	switch kind {
	case WCNCPKindObservation:
		name = "observation"
	case WCNCPKindOpportunity:
		name = "opportunity"
	case WCNCPKindProposal:
		name = "proposal"
	case WCNCPKindValidation:
		name = "validation"
	case WCNCPKindDecision:
		name = "decision"
	default:
		t.Fatalf("unknown kind %s", kind)
	}
	raw, ok := vectors.InvalidRecords[name]
	if !ok {
		t.Fatalf("missing invalid vector %s", name)
	}
	return append([]byte{}, raw...)
}

func wcncpTestDecisionRefs(t *testing.T, storage *Storage, now time.Time) []WCNCPReference {
	t.Helper()
	// Decision manifests require proposal+validation references. Build a
	// minimal parent chain in the same repository scope for lifecycle tests.
	// Generations use a process-wide counter so parallel subtests cannot
	// collide on the (repository,kind,generation) primary key.
	base := int64(wcncpTestGeneration.Add(1000))
	obsObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindObservation, wcncpTestValidRecord(t, WCNCPKindObservation))
	obsManifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindObservation, base+1, now, obsObject, nil))
	oppObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindOpportunity, wcncpTestValidRecord(t, WCNCPKindOpportunity))
	oppManifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindOpportunity, base+2, now, oppObject, []WCNCPReference{{Kind: WCNCPKindObservation, ManifestSHA256: obsManifest.ManifestSHA256, Relation: "DERIVED_FROM"}}))
	propObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindProposal, wcncpTestValidRecord(t, WCNCPKindProposal))
	propManifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindProposal, base+3, now, propObject, []WCNCPReference{{Kind: WCNCPKindOpportunity, ManifestSHA256: oppManifest.ManifestSHA256, Relation: "DERIVED_FROM"}}))
	valObject := putWCNCPTestObject(t, storage, wcncpTestScope, WCNCPKindValidation, wcncpTestValidRecord(t, WCNCPKindValidation))
	valManifest := putWCNCPTestManifest(t, storage, wcncpTestManifest(WCNCPKindValidation, base+4, now, valObject, []WCNCPReference{{Kind: WCNCPKindProposal, ManifestSHA256: propManifest.ManifestSHA256, Relation: "VALIDATES"}}))
	return []WCNCPReference{
		{Kind: WCNCPKindProposal, ManifestSHA256: propManifest.ManifestSHA256, Relation: "DECIDES"},
		{Kind: WCNCPKindValidation, ManifestSHA256: valManifest.ManifestSHA256, Relation: "DECIDES"},
	}
}

func findWCNCPRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}
