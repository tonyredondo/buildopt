package stickydecision

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSelectLocalAcceptsVerifiedNoopAndReportsActiveWithoutExecuting(t *testing.T) {
	now := time.Date(2026, 8, 27, 15, 0, 0, 0, time.UTC)
	scope := strings.Repeat("1", 64)
	binding := testBinding(scope, 0)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"owner": publicKey}

	root := filepath.Join(t.TempDir(), "noop")
	store, err := OpenLocalWithOptions(root, scope, StoreOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	decision := Decision{
		SchemaVersion: DecisionSchemaVersion, RecordType: DecisionRecordType,
		DecisionID: "noop-decision", StoreGeneration: 1, IdempotencyKey: digestFor("noop-decision"),
		Binding: binding, QualificationState: "OBSERVING", RolloutState: "PROPOSED",
		ExecutionDecision: ExecutionNativeNoop, PolicyDigest: digestFor("policy"),
		CacheContractDigest: digestFor("cache"), EvidenceRefs: []string{},
		IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano),
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		Authentication: Authentication{Algorithm: "Ed25519", KeyID: "owner"},
	}
	raw, _, err := SignDecision(decision, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(context.Background(), raw, 0, "", decision.IdempotencyKey); err != nil {
		t.Fatal(err)
	}

	selection := SelectLocal(context.Background(), root, scope, binding, SelectorOptions{
		PublicKeys: keys, Now: func() time.Time { return now },
	})
	if selection.Status != SelectionNative || selection.Decision != ExecutionNativeNoop || selection.Reason != SelectionReasonVerifiedNoop {
		t.Fatalf("verified no-op selection = %+v", selection)
	}
	if selection.Generation != 1 || selection.RecordDigest == "" || selection.ReadDurationNs <= 0 {
		t.Fatalf("verified no-op metadata = %+v", selection)
	}

	activeRoot := filepath.Join(t.TempDir(), "active")
	activeStore, err := OpenLocalWithOptions(activeRoot, scope, StoreOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	active := decision
	active.DecisionID = "active-decision"
	active.IdempotencyKey = digestFor("active-decision")
	active.ActionID = "qualified-action"
	active.ActionGeneration = 1
	active.QualificationState = "QUARANTINE_VALIDATED"
	active.RolloutState = "ACTIVE_IN_CI"
	active.ExecutionDecision = ExecutionActiveRuntime
	active.EvidenceRefs = []string{digestFor("trial")}
	activeRaw, _, err := SignDecision(active, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := activeStore.Append(context.Background(), activeRaw, 0, "", active.IdempotencyKey); err != nil {
		t.Fatal(err)
	}
	activeSelection := SelectLocal(context.Background(), activeRoot, scope, binding, SelectorOptions{
		PublicKeys: keys, Now: func() time.Time { return now },
	})
	if activeSelection.Status != SelectionActive || activeSelection.Decision != ExecutionActiveRuntime || activeSelection.Reason != SelectionReasonVerifiedActive {
		t.Fatalf("active selection = %+v", activeSelection)
	}
}

func TestSelectLocalFailsClosedAndDoesNotCreateMissingState(t *testing.T) {
	now := time.Date(2026, 8, 27, 16, 0, 0, 0, time.UTC)
	scope := strings.Repeat("2", 64)
	binding := testBinding(scope, 0)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"owner": publicKey}

	missingRoot := filepath.Join(t.TempDir(), "missing")
	var refreshCalls atomic.Int32
	refreshDone := make(chan struct{})
	refreshRelease := make(chan struct{})
	scheduler := &RefreshScheduler{}
	refresh := func(context.Context) error {
		refreshCalls.Add(1)
		close(refreshDone)
		<-refreshRelease
		return nil
	}
	options := SelectorOptions{PublicKeys: keys, Now: func() time.Time { return now }, Scheduler: scheduler, Refresh: refresh}
	missing := SelectLocal(context.Background(), missingRoot, scope, binding, options)
	if missing.Status != SelectionNative || missing.Reason != SelectionReasonNoLocalSnapshot || !missing.RefreshScheduled {
		t.Fatalf("missing selection = %+v", missing)
	}
	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("asynchronous refresh did not start")
	}
	second := SelectLocal(context.Background(), missingRoot, scope, binding, options)
	if second.RefreshScheduled {
		t.Fatalf("refresh was not coalesced: %+v", second)
	}
	if _, err := os.Stat(missingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing selector changed filesystem: %v", err)
	}
	close(refreshRelease)

	root := filepath.Join(t.TempDir(), "states")
	store, err := OpenLocalWithOptions(root, scope, StoreOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	base := testDecision(scope, 1, digestFor("selector-expired"), now.Add(-time.Minute), now.Add(time.Minute), "owner")
	base.ExecutionDecision = ExecutionNativeNoop
	base.EvidenceRefs = []string{}
	raw, _, err := SignDecision(base, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(context.Background(), raw, 0, "", base.IdempotencyKey); err != nil {
		t.Fatal(err)
	}
	clock := now.Add(2 * time.Minute)
	expired := SelectLocal(context.Background(), root, scope, binding, SelectorOptions{
		PublicKeys: keys, Now: func() time.Time { return clock },
	})
	if expired.Status != SelectionNative || expired.Reason != SelectionReasonExpired {
		t.Fatalf("expired selection = %+v", expired)
	}

	corruptRoot := filepath.Join(t.TempDir(), "corrupt")
	corruptStore, err := OpenLocalWithOptions(corruptRoot, scope, StoreOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	corruptDecision := base
	corruptDecision.DecisionID = "corrupt-decision"
	corruptDecision.IdempotencyKey = digestFor("corrupt-decision")
	corruptDecision.IssuedAt = now.Add(-time.Minute).Format(time.RFC3339Nano)
	corruptDecision.ExpiresAt = now.Add(time.Hour).Format(time.RFC3339Nano)
	corruptRaw, _, err := SignDecision(corruptDecision, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	result, err := corruptStore.Append(context.Background(), corruptRaw, 0, "", corruptDecision.IdempotencyKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptRoot, localHeadFile), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt := SelectLocal(context.Background(), corruptRoot, scope, binding, SelectorOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	if corrupt.Status != SelectionNative || corrupt.Reason != SelectionReasonCorrupt {
		t.Fatalf("corrupt selection = %+v (record %s)", corrupt, result.RecordDigest)
	}

	noKeys := SelectLocal(context.Background(), root, scope, binding, SelectorOptions{Now: func() time.Time { return now }})
	if noKeys.Status != SelectionNative || noKeys.Reason != SelectionReasonInvalidRequest {
		t.Fatalf("missing key registry selection = %+v", noKeys)
	}
	wrongBinding := binding
	wrongBinding.OptionsSHA256 = digestFor("different-options")
	wrong := SelectLocal(context.Background(), root, scope, wrongBinding, SelectorOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	if wrong.Status != SelectionNative || wrong.Reason != SelectionReasonIncompatible {
		t.Fatalf("binding mismatch selection = %+v", wrong)
	}
}

func TestLocalDecisionRootIsPortableAndBoundToScope(t *testing.T) {
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	scope := strings.Repeat("3", 64)
	root, err := LocalDecisionRoot(cacheRoot, scope)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cacheRoot, "buildopt", "sticky", "state", scope)
	if root != want {
		t.Fatalf("decision root = %q, want %q", root, want)
	}
	if _, err := LocalDecisionRoot("relative", scope); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("relative cache root error = %v", err)
	}
	if _, err := LocalDecisionRoot(cacheRoot, "not-a-scope"); !errors.Is(err, ErrCrossScope) {
		t.Fatalf("invalid scope error = %v", err)
	}
}
