package sharedcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestTypedStatePersistsAcrossRestartAndIsolatesNamespaces(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage := openStateTestStorage(t, ctx, root)
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	evidenceObject := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("evidence-one"))
	if file, err := storage.OpenStateObject(ctx, stateTestScope, StateKindPortfolio, evidenceObject.SHA256); file != nil || !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("cross-kind object = %+v/%v", file, err)
	}
	if file, err := storage.OpenStateObject(ctx, stateTestOtherScope, StateKindEvidence, evidenceObject.SHA256); file != nil || !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("cross-repository object = %+v/%v", file, err)
	}

	evidence := stateTestManifest(StateKindEvidence, 1, now, evidenceObject, "")
	evidenceStored := putStateTestManifest(t, storage, evidence)
	evidenceHead := casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope,
		Kind:                  StateKindEvidence,
		IdempotencyKey:        stateTestDigest("evidence-cas-one"),
		ManifestSHA256:        evidenceStored.ManifestSHA256,
	})

	storage.clock = func() time.Time { return now.Add(time.Minute) }
	portfolioObject := putStateTestObject(t, storage, stateTestScope, StateKindPortfolio, []byte("portfolio-one"))
	portfolio := stateTestManifest(
		StateKindPortfolio,
		1,
		now.Add(time.Minute),
		portfolioObject,
		evidenceStored.ManifestSHA256,
	)
	portfolioStored := putStateTestManifest(t, storage, portfolio)
	portfolioHead := casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope,
		Kind:                  StateKindPortfolio,
		IdempotencyKey:        stateTestDigest("portfolio-cas-one"),
		ManifestSHA256:        portfolioStored.ManifestSHA256,
	})
	if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindCheckpoint); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("unpublished checkpoint = %v", err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	storage = openStateTestStorage(t, ctx, root)
	defer storage.Close()
	for _, testCase := range []struct {
		kind StateKind
		head StateCASResult
	}{
		{kind: StateKindEvidence, head: evidenceHead},
		{kind: StateKindPortfolio, head: portfolioHead},
	} {
		snapshot, err := storage.LoadCurrentState(ctx, stateTestScope, testCase.kind)
		if err != nil {
			t.Fatalf("load %s after restart: %v", testCase.kind, err)
		}
		if snapshot.HeadSHA256 != testCase.head.HeadSHA256 ||
			!snapshot.Head.Authority.SelectionRequiresLocalRevalidation ||
			snapshot.Head.Authority.ProductionAuthorized {
			t.Fatalf("restarted %s snapshot = %+v", testCase.kind, snapshot)
		}
	}
	replayed, err := storage.CASStateHead(ctx, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope,
		Kind:                  StateKindEvidence,
		IdempotencyKey:        stateTestDigest("evidence-cas-one"),
		ManifestSHA256:        evidenceStored.ManifestSHA256,
	})
	if err != nil || !replayed.Replayed || replayed.HeadSHA256 != evidenceHead.HeadSHA256 {
		t.Fatalf("restart replay = %+v/%v", replayed, err)
	}
}

func TestTypedStateCASIsAtomicConcurrentAndIdempotent(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	object := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("concurrent-evidence"))
	manifest := putStateTestManifest(t, storage, stateTestManifest(StateKindEvidence, 1, now, object, ""))
	requests := []StateCASRequest{
		{
			RepositoryScopeSHA256: stateTestScope,
			Kind:                  StateKindEvidence,
			IdempotencyKey:        stateTestDigest("concurrent-one"),
			ManifestSHA256:        manifest.ManifestSHA256,
		},
		{
			RepositoryScopeSHA256: stateTestScope,
			Kind:                  StateKindEvidence,
			IdempotencyKey:        stateTestDigest("concurrent-two"),
			ManifestSHA256:        manifest.ManifestSHA256,
		},
	}
	type outcome struct {
		result StateCASResult
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
			result, err := storage.CASStateHead(ctx, request)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(outcomes)
	winners := 0
	preconditions := 0
	var winner StateCASResult
	for outcome := range outcomes {
		switch {
		case outcome.err == nil:
			winners++
			winner = outcome.result
		case errors.Is(outcome.err, ErrStateHeadPrecondition):
			preconditions++
		default:
			t.Fatalf("concurrent CAS = %+v/%v", outcome.result, outcome.err)
		}
	}
	if winners != 1 || preconditions != 1 {
		t.Fatalf("concurrent outcomes winners=%d preconditions=%d", winners, preconditions)
	}
	snapshot, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindEvidence)
	if err != nil || snapshot.HeadSHA256 != winner.HeadSHA256 {
		t.Fatalf("winner visibility = %+v/%v", snapshot, err)
	}

	winningRequest := requests[0]
	if winner.HeadSHA256 == "" {
		t.Fatal("winning head digest is empty")
	}
	for _, request := range requests {
		replayed, err := storage.CASStateHead(ctx, request)
		if err == nil && replayed.Replayed {
			winningRequest = request
			break
		}
	}
	conflict := winningRequest
	conflict.RepositoryScopeSHA256 = stateTestOtherScope
	if _, err := storage.CASStateHead(ctx, conflict); !errors.Is(err, ErrStateIdempotency) {
		t.Fatalf("changed idempotency payload = %v", err)
	}
}

func TestTypedStateConcurrentObjectPublicationsRemainAvailable(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()

	type outcome struct {
		digest string
		err    error
	}
	const writers = 16
	start := make(chan struct{})
	outcomes := make(chan outcome, writers)
	var group sync.WaitGroup
	for index := range writers {
		content := []byte("concurrent-object-" + strconv.Itoa(index))
		digest := sha256.Sum256(content)
		expected := hex.EncodeToString(digest[:])
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			object, _, err := storage.PutStateObject(
				ctx,
				stateTestScope,
				StateKindEvidence,
				expected,
				bytes.NewReader(content),
			)
			outcomes <- outcome{digest: object.SHA256, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(outcomes)

	for item := range outcomes {
		if item.err != nil || item.digest == "" {
			t.Fatalf("concurrent object publication = %q/%v", item.digest, item.err)
		}
		file, err := storage.OpenStateObject(
			ctx,
			stateTestScope,
			StateKindEvidence,
			item.digest,
		)
		if err != nil {
			t.Fatalf("open concurrent object %s: %v", item.digest, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close concurrent object %s: %v", item.digest, err)
		}
	}
}

func TestTypedStatePartialPublicationNeverBecomesVisible(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage := openStateTestStorage(t, ctx, root)
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	object := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("partial-evidence"))
	manifest := stateTestManifest(StateKindEvidence, 1, now, object, "")
	putStateTestManifest(t, storage, manifest)
	if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindEvidence); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("staged state became visible = %v", err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	storage = openStateTestStorage(t, ctx, root)
	defer storage.Close()
	if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindEvidence); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("staged state became visible after restart = %v", err)
	}
	storage.clock = func() time.Time { return now.Add(25 * time.Hour) }
	report, err := storage.MaintainState(ctx)
	if err != nil || report.ExpiredStagedManifests != 1 ||
		report.ExpiredStagedObjects != 1 || report.DeletedUnreferencedBlob != 1 {
		t.Fatalf("staged cleanup = %+v/%v", report, err)
	}
}

func TestTypedStateRejectsBlobAndManifestCorruptionAfterRestart(t *testing.T) {
	t.Run("blob", func(t *testing.T) {
		ctx := context.Background()
		root := filepath.Join(t.TempDir(), "shared")
		storage := openStateTestStorage(t, ctx, root)
		now := time.Now().UTC().Truncate(time.Second)
		object := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("durable-evidence"))
		manifest := putStateTestManifest(t, storage, stateTestManifest(StateKindEvidence, 1, now, object, ""))
		casStateTestHead(t, storage, StateCASRequest{
			RepositoryScopeSHA256: stateTestScope,
			Kind:                  StateKindEvidence,
			IdempotencyKey:        stateTestDigest("corrupt-blob"),
			ManifestSHA256:        manifest.ManifestSHA256,
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
		if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindEvidence); !errors.Is(err, ErrStateCorrupt) {
			t.Fatalf("corrupt blob load = %v", err)
		}
	})

	t.Run("manifest", func(t *testing.T) {
		ctx := context.Background()
		root := filepath.Join(t.TempDir(), "shared")
		storage := openStateTestStorage(t, ctx, root)
		now := time.Now().UTC().Truncate(time.Second)
		object := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("manifest-evidence"))
		manifest := putStateTestManifest(t, storage, stateTestManifest(StateKindEvidence, 1, now, object, ""))
		casStateTestHead(t, storage, StateCASRequest{
			RepositoryScopeSHA256: stateTestScope,
			Kind:                  StateKindEvidence,
			IdempotencyKey:        stateTestDigest("corrupt-manifest"),
			ManifestSHA256:        manifest.ManifestSHA256,
		})
		path := storage.Layout().StateDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		database, err := sql.Open("sqlite", sqliteDataSource(path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(
			`UPDATE state_manifests SET canonical_document = ?`,
			[]byte(`{"corrupt":true}`),
		); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		storage = openStateTestStorage(t, ctx, root)
		defer storage.Close()
		if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindEvidence); !errors.Is(err, ErrStateCorrupt) {
			t.Fatalf("corrupt manifest load = %v", err)
		}
	})
}

func TestTypedStateRejectsIncompleteInvalidAndSkippedPromotion(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return now }

	missing := stateTestManifest(
		StateKindEvidence,
		1,
		now,
		StateObject{
			RepositoryScopeSHA256: stateTestScope,
			Kind:                  StateKindEvidence,
			SHA256:                stateTestDigest("missing"),
			SizeBytes:             7,
		},
		"",
	)
	missingRaw, _, err := canonicalStateValue(missing)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.PutStateManifest(ctx, missingRaw); !errors.Is(err, ErrStateManifestIncomplete) {
		t.Fatalf("missing artifact manifest = %v", err)
	}

	invalidRaw := append(bytes.TrimSuffix(missingRaw, []byte("}")), []byte(`,"unknown":true}`)...)
	if _, _, err := storage.PutStateManifest(ctx, invalidRaw); !errors.Is(err, ErrStateInvalid) {
		t.Fatalf("unknown manifest property = %v", err)
	}

	checkpointObjectOne := putStateTestObject(t, storage, stateTestScope, StateKindCheckpoint, []byte("checkpoint-one"))
	checkpointObjectTwo := putStateTestObject(t, storage, stateTestScope, StateKindCheckpoint, []byte("checkpoint-two"))
	duplicateCheckpoint := stateTestManifest(StateKindCheckpoint, 1, now, checkpointObjectOne, "")
	duplicateCheckpoint.Artifacts = append(
		duplicateCheckpoint.Artifacts,
		StateArtifact{
			Role: "OPTIMIZE_STATE", SHA256: checkpointObjectTwo.SHA256,
			SizeBytes:            checkpointObjectTwo.SizeBytes,
			PayloadSchemaVersion: "buildopt.poc/optimize-state/v1",
		},
	)
	duplicateCheckpointRaw, _, err := canonicalStateValue(duplicateCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.PutStateManifest(ctx, duplicateCheckpointRaw); !errors.Is(err, ErrStateInvalid) {
		t.Fatalf("duplicate checkpoint state artifact = %v", err)
	}

	object := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("staged-corruption"))
	skipped := putStateTestManifest(t, storage, stateTestManifest(StateKindEvidence, 3, now, object, ""))
	if _, err := storage.CASStateHead(ctx, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope,
		Kind:                  StateKindEvidence,
		IdempotencyKey:        stateTestDigest("skipped-generation"),
		ManifestSHA256:        skipped.ManifestSHA256,
	}); !errors.Is(err, ErrStateGenerationConflict) {
		t.Fatalf("skipped generation = %v", err)
	}
	path := filepath.Join(storage.Layout().Blobs, object.SHA256[:2], object.SHA256[2:])
	if err := os.WriteFile(path, bytes.Repeat([]byte{'z'}, int(object.SizeBytes)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CASStateHead(ctx, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope,
		Kind:                  StateKindEvidence,
		IdempotencyKey:        stateTestDigest("corrupt-staged-generation"),
		ExpectedGeneration:    2,
		ExpectedHeadSHA256:    stateTestDigest("absent-head"),
		ManifestSHA256:        skipped.ManifestSHA256,
	}); !errors.Is(err, ErrStateCorrupt) {
		t.Fatalf("corrupt staged artifact = %v", err)
	}
	if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindEvidence); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("rejected generation became visible = %v", err)
	}
}

func TestTypedStateRetentionKeepsReferencedEvidenceAndExpiresCheckpoint(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, filepath.Join(t.TempDir(), "shared"))
	defer storage.Close()
	base := time.Now().UTC().Truncate(time.Second)
	storage.clock = func() time.Time { return base }

	evidenceOneObject := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("evidence-retained"))
	evidenceOne := putStateTestManifest(t, storage, stateTestManifest(StateKindEvidence, 1, base, evidenceOneObject, ""))
	evidenceOneHead := casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope, Kind: StateKindEvidence,
		IdempotencyKey: stateTestDigest("retention-evidence-one"), ManifestSHA256: evidenceOne.ManifestSHA256,
	})
	storage.clock = func() time.Time { return base.Add(time.Minute) }
	portfolioOneObject := putStateTestObject(t, storage, stateTestScope, StateKindPortfolio, []byte("portfolio-retained"))
	portfolioOne := putStateTestManifest(t, storage, stateTestManifest(StateKindPortfolio, 1, base.Add(time.Minute), portfolioOneObject, evidenceOne.ManifestSHA256))
	portfolioOneHead := casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope, Kind: StateKindPortfolio,
		IdempotencyKey: stateTestDigest("retention-portfolio-one"), ManifestSHA256: portfolioOne.ManifestSHA256,
	})

	storage.clock = func() time.Time { return base.Add(2 * time.Minute) }
	evidenceTwoObject := putStateTestObject(t, storage, stateTestScope, StateKindEvidence, []byte("evidence-current"))
	evidenceTwo := putStateTestManifest(t, storage, stateTestManifest(StateKindEvidence, 2, base.Add(2*time.Minute), evidenceTwoObject, ""))
	casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope, Kind: StateKindEvidence,
		IdempotencyKey: stateTestDigest("retention-evidence-two"), ExpectedGeneration: 1,
		ExpectedHeadSHA256: evidenceOneHead.HeadSHA256, ManifestSHA256: evidenceTwo.ManifestSHA256,
	})
	storage.clock = func() time.Time { return base.Add(31 * 24 * time.Hour) }
	if report, err := storage.MaintainState(ctx); err != nil || report.ExpiredSuperseded != 0 {
		t.Fatalf("referenced evidence maintenance = %+v/%v", report, err)
	}
	assertStateManifestPresent(t, storage, evidenceOne.ManifestSHA256, true)

	portfolioTwoObject := putStateTestObject(t, storage, stateTestScope, StateKindPortfolio, []byte("portfolio-current"))
	portfolioTwo := putStateTestManifest(t, storage, stateTestManifest(StateKindPortfolio, 2, base.Add(31*24*time.Hour), portfolioTwoObject, evidenceTwo.ManifestSHA256))
	casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope, Kind: StateKindPortfolio,
		IdempotencyKey: stateTestDigest("retention-portfolio-two"), ExpectedGeneration: 1,
		ExpectedHeadSHA256: portfolioOneHead.HeadSHA256, ManifestSHA256: portfolioTwo.ManifestSHA256,
	})
	storage.clock = func() time.Time { return base.Add(62 * 24 * time.Hour) }
	if report, err := storage.MaintainState(ctx); err != nil || report.ExpiredSuperseded != 1 {
		t.Fatalf("portfolio supersession = %+v/%v", report, err)
	}
	assertStateManifestPresent(t, storage, evidenceOne.ManifestSHA256, true)
	storage.clock = func() time.Time { return base.Add(93 * 24 * time.Hour) }
	if report, err := storage.MaintainState(ctx); err != nil || report.ExpiredSuperseded != 1 {
		t.Fatalf("evidence retention expiry = %+v/%v", report, err)
	}
	assertStateManifestPresent(t, storage, evidenceOne.ManifestSHA256, false)

	checkpointNow := base.Add(94 * 24 * time.Hour)
	storage.clock = func() time.Time { return checkpointNow }
	checkpointObject := putStateTestObject(t, storage, stateTestScope, StateKindCheckpoint, []byte("checkpoint-expiring"))
	checkpoint := putStateTestManifest(t, storage, stateTestManifest(StateKindCheckpoint, 1, checkpointNow, checkpointObject, ""))
	casStateTestHead(t, storage, StateCASRequest{
		RepositoryScopeSHA256: stateTestScope, Kind: StateKindCheckpoint,
		IdempotencyKey: stateTestDigest("retention-checkpoint"), ManifestSHA256: checkpoint.ManifestSHA256,
	})
	storage.clock = func() time.Time { return checkpointNow.Add(25 * time.Hour) }
	report, err := storage.MaintainState(ctx)
	if err != nil || report.ExpiredCheckpoints != 1 {
		t.Fatalf("checkpoint expiry = %+v/%v", report, err)
	}
	if _, err := storage.LoadCurrentState(ctx, stateTestScope, StateKindCheckpoint); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("expired checkpoint load = %v", err)
	}
}

const (
	stateTestScope      = "1111111111111111111111111111111111111111111111111111111111111111"
	stateTestOtherScope = "2222222222222222222222222222222222222222222222222222222222222222"
)

func openStateTestStorage(t *testing.T, ctx context.Context, root string) *Storage {
	t.Helper()
	storage, err := Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	return storage
}

func putStateTestObject(t *testing.T, storage *Storage, scope string, kind StateKind, content []byte) StateObject {
	t.Helper()
	digest := stateTestBytesDigest(content)
	object, _, err := storage.PutStateObject(context.Background(), scope, kind, digest, bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func putStateTestManifest(t *testing.T, storage *Storage, manifest StateManifest) StateSnapshot {
	t.Helper()
	canonical, _, err := canonicalStateValue(manifest)
	if err != nil {
		t.Fatal(err)
	}
	stored, _, err := storage.PutStateManifest(context.Background(), canonical)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func casStateTestHead(t *testing.T, storage *Storage, request StateCASRequest) StateCASResult {
	t.Helper()
	result, err := storage.CASStateHead(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func stateTestManifest(kind StateKind, generation int64, createdAt time.Time, object StateObject, evidenceManifest string) StateManifest {
	manifest := StateManifest{
		SchemaVersion:         "buildopt.central/state-manifest/v1",
		RecordType:            "CENTRAL_STATE_MANIFEST",
		Kind:                  kind,
		RepositoryScopeSHA256: object.RepositoryScopeSHA256,
		Generation:            generation,
		CompatibilitySHA256:   stateTestDigest("compatibility"),
		BindingsSHA256:        stateTestDigest("bindings-" + string(kind) + strconv.FormatInt(generation, 10)),
		Origin: StateOrigin{
			BaseRevision:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			TargetRevision:           "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			BuildOptExecutableSHA256: stateTestDigest("buildopt"),
			WrapperSHA256:            stateTestDigest("wrapper"),
			GradleVersion:            "9.6.1",
		},
		Artifacts: []StateArtifact{{
			SHA256: object.SHA256, SizeBytes: object.SizeBytes,
		}},
		References: []StateReference{},
		CreatedAt:  createdAt.UTC().Format(time.RFC3339Nano),
		Authority: StateAuthority{
			SelectionRequiresLocalRevalidation: true,
			ProductionAuthorized:               false,
			TestOptimization:                   "OUT_OF_SCOPE",
		},
	}
	switch kind {
	case StateKindPortfolio:
		manifest.Artifacts[0].Role = "PORTFOLIO_INDEX"
		manifest.Artifacts[0].PayloadSchemaVersion = "buildopt.poc/optimize-profile-portfolio/v1"
		manifest.References = []StateReference{{
			Kind: StateKindEvidence, ManifestSHA256: evidenceManifest, Relation: "QUALIFICATION",
		}}
		manifest.Status = "COMPLETE"
		manifest.RetentionClass = "CURRENT_PLUS_30_DAYS_AFTER_SUPERSEDED"
	case StateKindEvidence:
		manifest.Artifacts[0].Role = "CALIBRATION_EVIDENCE"
		manifest.Artifacts[0].PayloadSchemaVersion = "buildopt.poc/structural-measurement/v1"
		manifest.Status = "COMPLETE"
		manifest.RetentionClass = "WHILE_REFERENCED_PLUS_30_DAYS"
	case StateKindCheckpoint:
		manifest.Artifacts[0].Role = "OPTIMIZE_STATE"
		manifest.Artifacts[0].PayloadSchemaVersion = "buildopt.poc/optimize-state/v1"
		manifest.Status = "RESUMABLE"
		manifest.RetentionClass = "24_HOURS_FROM_CREATED_AT"
		manifest.ExpiresAt = createdAt.Add(24 * time.Hour).UTC().Format(time.RFC3339Nano)
	}
	return manifest
}

func stateTestDigest(value string) string {
	return stateTestBytesDigest([]byte(value))
}

func stateTestBytesDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func assertStateManifestPresent(t *testing.T, storage *Storage, digest string, want bool) {
	t.Helper()
	var count int
	if err := storage.state.database.QueryRow(
		`SELECT count(*) FROM state_manifests WHERE manifest_digest = ?`,
		digestPrefix+digest,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if (count == 1) != want {
		t.Fatalf("manifest %s count = %d, want present=%t", digest, count, want)
	}
}
