package stickydecision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestStickyLifecycleAndCanonicalSignature(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	scope := strings.Repeat("a", 64)
	binding := testBinding(scope, 0)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"owner": publicKey}

	observation := Observation{
		SchemaVersion: ObservationSchemaVersion, RecordType: ObservationRecordType,
		ObservationID: "observation-1", StoreGeneration: 1,
		IdempotencyKey: digestFor("observation-1"), Binding: binding,
		ObservationKind: "REQUESTED_BUILD", Outcome: "SUCCESS", WallTimeMs: 1200,
		OutputDigest: digestFor("output"), EvidenceQuality: "EXACT",
		RecordedAt: now.Format(time.RFC3339Nano),
	}
	action := ActionRecord{
		SchemaVersion: ActionSchemaVersion, RecordType: ActionRecordType,
		ActionID: "action-1", StoreGeneration: 2, IdempotencyKey: digestFor("action-1"),
		Sequence: 1, Transition: "PROPOSE",
		FromQualificationState: "UNKNOWN", ToQualificationState: "OBSERVING",
		FromRolloutState: "PROPOSED", ToRolloutState: "PROPOSED", Binding: binding,
		OccurredAt: now.Add(time.Second).Format(time.RFC3339Nano),
	}
	trial := Trial{
		SchemaVersion: TrialSchemaVersion, RecordType: TrialRecordType,
		TrialID: "trial-1", StoreGeneration: 3, IdempotencyKey: digestFor("trial-1"),
		ActionID: "action-1", Binding: binding, IsolationDigest: digestFor("isolation"),
		CandidateOutcome: "SUCCESS", ControlOutcome: "SUCCESS",
		CandidateOutputDigest: digestFor("same-output"), ControlOutputDigest: digestFor("same-output"),
		Equivalence: "EXACT", Result: "QUALIFIED", CandidateWallTimeMs: 800,
		ControlWallTimeMs: 1200, LearningCostMs: 100,
		RecordedAt: now.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
	observationRaw, observationDigest, err := CanonicalDocument(observation)
	if err != nil {
		t.Fatal(err)
	}
	actionRaw, _, err := CanonicalDocument(action)
	if err != nil {
		t.Fatal(err)
	}
	trialRaw, trialDigest, err := CanonicalDocument(trial)
	if err != nil {
		t.Fatal(err)
	}
	decision := Decision{
		SchemaVersion: DecisionSchemaVersion, RecordType: DecisionRecordType,
		DecisionID: "decision-1", StoreGeneration: 4, IdempotencyKey: digestFor("decision-1"),
		Binding: binding, ActionID: "action-1", ActionGeneration: 1,
		QualificationState: "QUARANTINE_VALIDATED", RolloutState: "ACTIVE_IN_CI",
		ExecutionDecision: ExecutionActiveRuntime, PolicyDigest: digestFor("policy"),
		CacheContractDigest: digestFor("cache-contract"), EvidenceRefs: []string{trialDigest},
		IssuedAt:       now.Add(3 * time.Second).Format(time.RFC3339Nano),
		ExpiresAt:      now.Add(time.Hour).Format(time.RFC3339Nano),
		Authentication: Authentication{Algorithm: "Ed25519", KeyID: "owner"},
	}
	decisionRaw, decisionDigest, err := SignDecision(decision, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyDecision(context.Background(), decisionRaw, keys, 0, now)
	if err != nil || verified.Digest() != decisionDigest {
		t.Fatalf("verify signed decision = %s/%v", verified.Digest(), err)
	}
	if _, err := VerifyDecision(context.Background(), bytes.Replace(decisionRaw, []byte("ACTIVE_IN_CI"), []byte("ACTIVE_LOCALLY"), 1), keys, 0, now); err == nil {
		t.Fatal("tampered signed decision was accepted")
	}
	ledger := EconomicLedger{
		SchemaVersion: LedgerSchemaVersion, RecordType: LedgerRecordType,
		LedgerID: "ledger-1", StoreGeneration: 5, IdempotencyKey: digestFor("ledger-1"), Binding: binding,
		Entries:      []LedgerEntry{{ActionID: "action-1", ObservationRef: observationDigest, GrossSavedMs: 400, BuildOptCostMs: 100, NetSavedMs: 300, Outcome: "SUCCESS", ObservedAt: now.Add(4 * time.Second).Format(time.RFC3339Nano)}},
		GrossSavedMs: 400, BuildOptCostMs: 100, NetSavedMs: 300, AsOf: now.Add(4 * time.Second).Format(time.RFC3339Nano),
	}
	ledgerRaw, _, err := CanonicalDocument(ledger)
	if err != nil {
		t.Fatal(err)
	}
	documents := make([]Document, 0, 5)
	for _, raw := range [][]byte{observationRaw, actionRaw, trialRaw, decisionRaw, ledgerRaw} {
		document, decodeErr := DecodeDocument(raw, now)
		if decodeErr != nil {
			t.Fatalf("decode lifecycle document: %v", decodeErr)
		}
		documents = append(documents, document)
	}
	if err := ValidateChain(documents); err != nil {
		t.Fatalf("validate lifecycle chain: %v", err)
	}
	unknownEvidence := decision
	unknownEvidence.EvidenceRefs = []string{digestFor("missing-evidence")}
	unknownEvidenceRaw, _, err := SignDecision(unknownEvidence, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	unknownEvidenceDocument, err := DecodeDocument(unknownEvidenceRaw, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateChain(append(append([]Document(nil), documents[:3]...), unknownEvidenceDocument)); err == nil {
		t.Fatal("active decision with unknown evidence was accepted")
	}
	unknownLedger := ledger
	unknownLedger.Entries = []LedgerEntry{{ActionID: "action-1", ObservationRef: digestFor("missing-observation"), GrossSavedMs: 1, BuildOptCostMs: 0, NetSavedMs: 1, Outcome: "SUCCESS", ObservedAt: now.Add(4 * time.Second).Format(time.RFC3339Nano)}}
	unknownLedger.GrossSavedMs, unknownLedger.BuildOptCostMs, unknownLedger.NetSavedMs = 1, 0, 1
	unknownLedgerRaw, _, err := CanonicalDocument(unknownLedger)
	if err != nil {
		t.Fatal(err)
	}
	unknownLedgerDocument, err := DecodeDocument(unknownLedgerRaw, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateChain(append(append([]Document(nil), documents[:4]...), unknownLedgerDocument)); err == nil {
		t.Fatal("ledger with unknown observation was accepted")
	}
	overflowLedger := ledger
	overflowLedger.Entries = []LedgerEntry{{
		ActionID: "action-1", ObservationRef: observationDigest,
		GrossSavedMs: minInt64Value, BuildOptCostMs: uint64(maxInt64Value), NetSavedMs: 0,
		Outcome: "SUCCESS", ObservedAt: now.Add(4 * time.Second).Format(time.RFC3339Nano),
	}}
	overflowLedger.GrossSavedMs, overflowLedger.BuildOptCostMs, overflowLedger.NetSavedMs = minInt64Value, uint64(maxInt64Value), 0
	overflowRaw, _, err := CanonicalDocument(overflowLedger)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDocument(overflowRaw, now); err == nil {
		t.Fatal("overflowing ledger arithmetic was accepted")
	}
	if _, err := DecodeDocument([]byte(`{"recordType":"STICKY_DECISION","unknown":true}`), now); err == nil {
		t.Fatal("unknown fields were accepted")
	}
	if err := ValidateActionTransition(ActionRecord{
		FromQualificationState: "UNKNOWN", ToQualificationState: "QUARANTINE_VALIDATED",
		FromRolloutState: "PROPOSED", ToRolloutState: "ACTIVE_IN_CI", Transition: "ACTIVATE_IN_CI",
	}); err == nil {
		t.Fatal("direct activation transition was accepted")
	}
}

func TestAllStickyActionTransitions(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	scope := strings.Repeat("e", 64)
	binding := testBinding(scope, 0)
	cases := []struct {
		name, transition, fromQualification, toQualification, fromRollout, toRollout string
	}{
		{name: "propose", transition: "PROPOSE", fromQualification: "UNKNOWN", toQualification: "OBSERVING", fromRollout: "PROPOSED", toRollout: "PROPOSED"},
		{name: "begin-shadow", transition: "BEGIN_SHADOW", fromQualification: "OBSERVING", toQualification: "CONTRACT_QUALIFIED", fromRollout: "PROPOSED", toRollout: "SHADOW"},
		{name: "begin-ci-canary", transition: "BEGIN_CI_CANARY", fromQualification: "CONTRACT_QUALIFIED", toQualification: "QUARANTINE_VALIDATED", fromRollout: "SHADOW", toRollout: "CI_CANARY"},
		{name: "activate-in-ci", transition: "ACTIVATE_IN_CI", fromQualification: "QUARANTINE_VALIDATED", toQualification: "QUARANTINE_VALIDATED", fromRollout: "CI_CANARY", toRollout: "ACTIVE_IN_CI"},
		{name: "activate-locally", transition: "ACTIVATE_LOCALLY", fromQualification: "QUARANTINE_VALIDATED", toQualification: "QUARANTINE_VALIDATED", fromRollout: "ACTIVE_IN_CI", toRollout: "ACTIVE_LOCALLY"},
		{name: "suspend", transition: "SUSPEND", fromQualification: "QUARANTINE_VALIDATED", toQualification: "SUSPENDED", fromRollout: "ACTIVE_LOCALLY", toRollout: "SUSPENDED"},
		{name: "rollback-from-canary", transition: "ROLLBACK", fromQualification: "QUARANTINE_VALIDATED", toQualification: "SUSPENDED", fromRollout: "CI_CANARY", toRollout: "ROLLED_BACK"},
		{name: "rollback-from-ci", transition: "ROLLBACK", fromQualification: "QUARANTINE_VALIDATED", toQualification: "SUSPENDED", fromRollout: "ACTIVE_IN_CI", toRollout: "ROLLED_BACK"},
		{name: "rollback-from-local", transition: "ROLLBACK", fromQualification: "QUARANTINE_VALIDATED", toQualification: "SUSPENDED", fromRollout: "ACTIVE_LOCALLY", toRollout: "ROLLED_BACK"},
		{name: "retire-suspended", transition: "RETIRE", fromQualification: "SUSPENDED", toQualification: "REJECTED", fromRollout: "SUSPENDED", toRollout: "ROLLED_BACK"},
		{name: "retire-rolled-back", transition: "RETIRE", fromQualification: "SUSPENDED", toQualification: "REJECTED", fromRollout: "ROLLED_BACK", toRollout: "ROLLED_BACK"},
	}
	for index, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			sequence := uint64(2)
			if testCase.transition == "PROPOSE" {
				sequence = 1
			}
			action := ActionRecord{
				SchemaVersion: ActionSchemaVersion, RecordType: ActionRecordType,
				ActionID: "transition-" + testCase.name, StoreGeneration: uint64(index + 1),
				IdempotencyKey: digestFor(testCase.name), Sequence: sequence, Transition: testCase.transition,
				FromQualificationState: testCase.fromQualification, ToQualificationState: testCase.toQualification,
				FromRolloutState: testCase.fromRollout, ToRolloutState: testCase.toRollout,
				Binding: binding, OccurredAt: now.Add(time.Duration(index) * time.Second).Format(time.RFC3339Nano),
			}
			if testCase.transition != "PROPOSE" {
				action.EvidenceRefs = []string{digestFor("evidence-" + testCase.name)}
			}
			raw, _, err := CanonicalDocument(action)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeDocument(raw, now); err != nil {
				t.Fatalf("valid transition rejected: %v", err)
			}
		})
	}

	invalid := []ActionRecord{
		{Transition: "ROLLBACK", Sequence: 2, FromQualificationState: "OBSERVING", ToQualificationState: "SUSPENDED", FromRolloutState: "CI_CANARY", ToRolloutState: "ROLLED_BACK"},
		{Transition: "SUSPEND", Sequence: 2, FromQualificationState: "SUSPENDED", ToQualificationState: "SUSPENDED", FromRolloutState: "SUSPENDED", ToRolloutState: "SUSPENDED"},
		{Transition: "RETIRE", Sequence: 2, FromQualificationState: "SUSPENDED", ToQualificationState: "REJECTED", FromRolloutState: "ACTIVE_IN_CI", ToRolloutState: "ROLLED_BACK"},
	}
	for _, action := range invalid {
		if err := ValidateActionTransition(action); err == nil {
			t.Fatalf("invalid transition accepted: %+v", action)
		}
	}
	sequenceMismatch := ActionRecord{
		SchemaVersion: ActionSchemaVersion, RecordType: ActionRecordType,
		ActionID: "sequence-mismatch", StoreGeneration: 1, IdempotencyKey: digestFor("sequence-mismatch"),
		Sequence: 2, Transition: "PROPOSE",
		FromQualificationState: "UNKNOWN", ToQualificationState: "OBSERVING",
		FromRolloutState: "PROPOSED", ToRolloutState: "PROPOSED", Binding: binding,
		OccurredAt: now.Format(time.RFC3339Nano),
	}
	if err := validateAction(sequenceMismatch, now); err == nil {
		t.Fatal("PROPOSE with a non-initial sequence was accepted")
	}
}

func TestLocalStoreCASReplayExpiryRevocationCorruptionAndPlaneSeparation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 27, 11, 0, 0, 0, time.UTC)
	clock := now
	scope := strings.Repeat("b", 64)
	root := filepath.Join(t.TempDir(), "local")
	store, err := OpenLocalWithOptions(root, scope, StoreOptions{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	observation := testObservation(scope, 1, digestFor("local-observation"), now)
	raw, digest, err := CanonicalDocument(observation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutCacheObject("gradle-key", raw); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Current(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cache object became control state: %v", err)
	}
	result, err := store.Append(ctx, raw, 0, "", observation.IdempotencyKey)
	if err != nil || result.RecordDigest != digest || result.Replayed {
		t.Fatalf("append = %+v/%v", result, err)
	}
	replay, err := store.Append(ctx, raw, 0, "", observation.IdempotencyKey)
	if err != nil || !replay.Replayed || replay.HeadDigest != result.HeadDigest {
		t.Fatalf("replay = %+v/%v", replay, err)
	}
	changed := observation
	changed.WallTimeMs++
	changedRaw, _, _ := CanonicalDocument(changed)
	if _, err := store.Append(ctx, changedRaw, 0, "", observation.IdempotencyKey); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed replay = %v", err)
	}
	if _, err := store.Append(ctx, raw, 0, "", digestFor("other-key")); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("record/key mismatch = %v", err)
	}
	if err := store.Revoke(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Current(ctx); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked local state = %v", err)
	}

	corruptRoot := filepath.Join(t.TempDir(), "corrupt")
	corruptStore, err := OpenLocalWithOptions(corruptRoot, scope, StoreOptions{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	corruptResult, err := corruptStore.Append(ctx, raw, 0, "", observation.IdempotencyKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptRoot, localRecordsDirectory, recordFilename(corruptResult.Generation, digest)), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := corruptStore.Current(ctx); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("corrupt local record = %v", err)
	}

	expiringRoot := filepath.Join(t.TempDir(), "expiring")
	expiringStore, err := OpenLocalWithOptions(expiringRoot, scope, StoreOptions{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)
	decision := testDecision(scope, 1, digestFor("expiring-decision"), now, now.Add(time.Minute), "owner")
	decisionRaw, _, err := SignDecision(decision, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	// The store has no key registry here; expiry is still enforced before any
	// action could be selected.
	if _, err := expiringStore.Append(ctx, decisionRaw, 0, "", decision.IdempotencyKey); err != nil {
		t.Fatal(err)
	}
	clock = now.Add(2 * time.Minute)
	if _, err := expiringStore.Current(ctx); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired local decision = %v (key %x)", err, publicKey[:2])
	}
}

func TestCentralStoreUsesTypedStateCASAndRejectsCrossPlane(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	scope := strings.Repeat("c", 64)
	storage, err := sharedcache.Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	central, err := NewCentralStore(storage, scope, StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	observation := testObservation(scope, 1, digestFor("central-observation"), now)
	raw, digest, err := CanonicalDocument(observation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := central.PutCacheObject(ctx, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := central.Current(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("central cache blob became state: %v", err)
	}
	result, err := central.Append(ctx, raw, 0, "", observation.IdempotencyKey)
	if err != nil || result.RecordDigest != digest {
		t.Fatalf("central append = %+v/%v", result, err)
	}
	current, err := central.Current(ctx)
	if err != nil || current.RecordDigest != digest || current.Head.Generation != 1 {
		t.Fatalf("central current = %+v/%v", current, err)
	}
	replay, err := central.Append(ctx, raw, 0, "", observation.IdempotencyKey)
	if err != nil || !replay.Replayed {
		t.Fatalf("central replay = %+v/%v", replay, err)
	}
	reopened, err := NewCentralStore(storage, scope, StoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	restartedReplay, err := reopened.Append(ctx, raw, 0, "", observation.IdempotencyKey)
	if err != nil || !restartedReplay.Replayed || restartedReplay.HeadDigest != result.HeadDigest {
		t.Fatalf("central replay after adapter restart = %+v/%v", restartedReplay, err)
	}
	if _, err := central.Append(ctx, raw, 0, "", digestFor("changed-idempotency")); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("central idempotency mismatch = %v", err)
	}
	if err := central.Revoke(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := central.Current(ctx); !errors.Is(err, ErrRevoked) {
		t.Fatalf("central revocation = %v", err)
	}
}

func testBinding(scope string, epoch int64) Binding {
	return Binding{
		RepositoryScopeSHA256: scope, Workflow: "pull-request/build",
		SourceRevision: strings.Repeat("d", 40), GradleVersion: "9.6.1",
		WrapperSHA256: digestFor("wrapper"), OptionsSHA256: digestFor("options"),
		OutputContractSHA256: digestFor("outputs"), BuildOptExecutableSHA256: digestFor("buildopt"),
		RevocationEpoch: epoch,
	}
}

func testObservation(scope string, generation uint64, key string, now time.Time) Observation {
	return Observation{
		SchemaVersion: ObservationSchemaVersion, RecordType: ObservationRecordType,
		ObservationID: "observation-local", StoreGeneration: generation, IdempotencyKey: key,
		Binding: testBinding(scope, 0), ObservationKind: "REQUESTED_BUILD", Outcome: "SUCCESS",
		WallTimeMs: 100, OutputDigest: digestFor("output"), EvidenceQuality: "EXACT",
		RecordedAt: now.Format(time.RFC3339Nano),
	}
}

func testDecision(scope string, generation uint64, key string, issued, expires time.Time, keyID string) Decision {
	return Decision{
		SchemaVersion: DecisionSchemaVersion, RecordType: DecisionRecordType,
		DecisionID: "decision-expiring", StoreGeneration: generation, IdempotencyKey: key,
		Binding: testBinding(scope, 0), QualificationState: "OBSERVING", RolloutState: "PROPOSED",
		ExecutionDecision: ExecutionObserve, PolicyDigest: digestFor("policy"), CacheContractDigest: digestFor("cache"),
		EvidenceRefs: []string{digestFor("evidence")}, IssuedAt: issued.Format(time.RFC3339Nano), ExpiresAt: expires.Format(time.RFC3339Nano),
		Authentication: Authentication{Algorithm: "Ed25519", KeyID: keyID},
	}
}

func digestFor(value string) string {
	return digestBytes([]byte(value + "-sticky-test"))
}
