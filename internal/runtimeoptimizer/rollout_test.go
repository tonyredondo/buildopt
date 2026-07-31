package runtimeoptimizer

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestRolloutBudgetEnforcesRollingWindowsConcurrencyAndActualCharge(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	controller, _, _ := testRolloutController(t, now)
	if err := controller.ConfigureRepository("repository-1", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
		t.Fatal(err)
	}
	if err := controller.RecordNaturalUsage("repository-1", RunnerUsage{EventID: "natural-1", At: now, RunnerMS: 100_000}); err != nil {
		t.Fatal(err)
	}
	reservation, err := controller.ReserveValidation("repository-1", "reservation-1", 5_000)
	if err != nil || reservation.State != "RESERVED" {
		t.Fatalf("reservation = %+v/%v", reservation, err)
	}
	if _, err := controller.ReserveValidation("repository-1", "reservation-concurrent", 1); !errors.Is(err, ErrRepositoryBusy) {
		t.Fatalf("concurrent = %v", err)
	}
	finished, err := controller.FinishValidation("repository-1", "reservation-1", 4_000, false)
	if err != nil || finished.State != "COMPLETED" || finished.ActualRunnerMS != 4_000 {
		t.Fatalf("finish = %+v/%v", finished, err)
	}
	if _, err := controller.ReserveValidation("repository-1", "reservation-over-week", 1_001); err == nil {
		t.Fatal("weekly budget was exceeded")
	}
	if reservation, err := controller.ReserveValidation("repository-1", "reservation-boundary", 1_000); err != nil || reservation.ReservedRunnerMS != 1_000 {
		t.Fatalf("boundary = %+v/%v", reservation, err)
	}
	if cancelled, err := controller.FinishValidation("repository-1", "reservation-boundary", 0, true); err != nil || cancelled.State != "CANCELLED" {
		t.Fatalf("cancel = %+v/%v", cancelled, err)
	}
	if repeated, err := controller.FinishValidation("repository-1", "reservation-1", 4_000, false); err != nil || repeated != finished {
		t.Fatalf("repeat = %+v/%v", repeated, err)
	}
}

func TestRolloutBudgetDailyBurstAndZeroBudget(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	controller, _, _ := testRolloutController(t, now)
	if err := controller.ConfigureRepository("daily", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
		t.Fatal(err)
	}
	if err := controller.RecordNaturalUsage("daily", RunnerUsage{EventID: "old-natural", At: now.Add(-48 * time.Hour), RunnerMS: 1_000_000}); err != nil {
		t.Fatal(err)
	}
	if err := controller.RecordNaturalUsage("daily", RunnerUsage{EventID: "daily-natural", At: now, RunnerMS: 10_000}); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.ReserveValidation("daily", "daily-over", 1_001); err == nil {
		t.Fatal("daily burst budget was exceeded")
	}
	if err := controller.ConfigureRepository("zero", LearningBudget{}); err != nil {
		t.Fatal(err)
	}
	if err := controller.RecordNaturalUsage("zero", RunnerUsage{EventID: "zero-natural", At: now, RunnerMS: 1_000_000}); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.ReserveValidation("zero", "zero-reservation", 1); err == nil {
		t.Fatal("zero budget admitted additional compute")
	}
}

func TestRolloutStagesNeverSkipGatesAndRetainFivePercentControl(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	controller, _, _ := testRolloutController(t, now)
	if err := controller.ConfigureRepository("repository-1", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
		t.Fatal(err)
	}
	action, err := controller.StartAction("repository-1", "action-1", DirectRolloutClass, true)
	if err != nil || action.Stage != "CANARY_5" {
		t.Fatalf("start = %+v/%v", action, err)
	}
	blocked, err := controller.AdvanceAction("repository-1", "action-1", RolloutEvidence{})
	if err != nil || blocked.Stage != "CANARY_5" {
		t.Fatalf("blocked = %+v/%v", blocked, err)
	}
	evidence := completeRolloutEvidence()
	for _, expected := range []string{"CANARY_25", "CANARY_50", "ACTIVE_95"} {
		action, err = controller.AdvanceAction("repository-1", "action-1", evidence)
		if err != nil || action.Stage != expected {
			t.Fatalf("advance = %+v/%v, want %s", action, err, expected)
		}
	}
	counts := map[string]int{}
	for index := 0; index < BasisPointTotal; index++ {
		selection := selectRolloutPoint(action, index, "W3_H4G")
		counts[selection.Arm]++
		if selection.Arm == "CANDIDATE" && selection.PropensityBasisPoints != 9500 {
			t.Fatalf("candidate propensity = %d", selection.PropensityBasisPoints)
		}
		if selection.Arm == "CONTROL" && selection.PropensityBasisPoints != 500 {
			t.Fatalf("control propensity = %d", selection.PropensityBasisPoints)
		}
	}
	if counts["CANDIDATE"] != 9500 || counts["CONTROL"] != 500 {
		t.Fatalf("95/5 counts = %v", counts)
	}
}

func TestProofRolloutStartsShadowAndUnsafeContextsStayControl(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	controller, _, _ := testRolloutController(t, now)
	if err := controller.ConfigureRepository("repository-1", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
		t.Fatal(err)
	}
	action, err := controller.StartAction("repository-1", "proof-1", ProofRolloutClass, false)
	if err != nil || action.Stage != "SHADOW" {
		t.Fatalf("proof start = %+v/%v", action, err)
	}
	evidence := completeRolloutEvidence()
	evidence.ContractComplete = false
	if action, err = controller.AdvanceAction("repository-1", "proof-1", evidence); err != nil || action.Stage != "SHADOW" {
		t.Fatalf("contract block = %+v/%v", action, err)
	}
	evidence.ContractComplete = true
	if action, err = controller.AdvanceAction("repository-1", "proof-1", evidence); err != nil || action.Stage != "CANARY_1" {
		t.Fatalf("proof canary = %+v/%v", action, err)
	}
	for _, context := range []SelectionContext{
		{AssignmentID: "local", SeedDigest: testDigest("7"), CandidateProfileID: "W2_H3G"},
		{AssignmentID: "release", SeedDigest: testDigest("7"), CI: true, Release: true, CandidateProfileID: "W2_H3G"},
		{AssignmentID: "effects", SeedDigest: testDigest("7"), CI: true, ExternalEffects: true, CandidateProfileID: "W2_H3G"},
		{AssignmentID: "unsafe-arm", SeedDigest: testDigest("7"), CI: true, CandidateProfileID: "ARBITRARY"},
	} {
		selection, err := controller.SelectAction("repository-1", "proof-1", context)
		if err != nil || selection.ResourceProfileID != "STABLE_CONTROL" {
			t.Fatalf("unsafe context = %+v/%v", selection, err)
		}
	}
}

func TestRolloutSuspendsOnDriftRegressionOrIncompleteTelemetry(t *testing.T) {
	reasons := []RolloutObservation{
		{ObservationID: "incomplete"},
		{ObservationID: "drift", TelemetryComplete: true, DriftDetected: true},
		{ObservationID: "regression", TelemetryComplete: true, P95Regression: true},
		{ObservationID: "queue", TelemetryComplete: true, QueueRegression: true},
		{ObservationID: "oom", TelemetryComplete: true, OOM: true},
		{ObservationID: "swapping", TelemetryComplete: true, SustainedSwapping: true},
		{ObservationID: "divergence", TelemetryComplete: true, ArtifactDivergence: true},
		{ObservationID: "failure", TelemetryComplete: true, AttributableFailure: true},
	}
	for _, observation := range reasons {
		t.Run(observation.ObservationID, func(t *testing.T) {
			now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
			controller, _, _ := testRolloutController(t, now)
			if err := controller.ConfigureRepository("repository-1", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
				t.Fatal(err)
			}
			if _, err := controller.StartAction("repository-1", "action-1", DirectRolloutClass, true); err != nil {
				t.Fatal(err)
			}
			action, err := controller.ObserveAction("repository-1", "action-1", observation)
			if err != nil || action.State != RolloutSuspended || action.CandidateBasisPoints != 0 {
				t.Fatalf("suspend = %+v/%v", action, err)
			}
			selection, err := controller.SelectAction("repository-1", "action-1", SelectionContext{AssignmentID: "after", SeedDigest: testDigest("7"), CI: true, CandidateProfileID: "W2_H3G"})
			if err != nil || selection.ResourceProfileID != "STABLE_CONTROL" {
				t.Fatalf("fallback = %+v/%v", selection, err)
			}
		})
	}
}

func TestExplicitRollbackRestoresControl(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	controller, _, _ := testRolloutController(t, now)
	if err := controller.ConfigureRepository("repository-1", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.StartAction("repository-1", "action-1", DirectRolloutClass, true); err != nil {
		t.Fatal(err)
	}
	action, err := controller.RollbackAction("repository-1", "action-1", "operator-rollback")
	if err != nil || action.State != RolloutRolledBack || action.CandidateBasisPoints != 0 {
		t.Fatalf("rollback = %+v/%v", action, err)
	}
	selection, err := controller.SelectAction("repository-1", "action-1", SelectionContext{AssignmentID: "rolled-back", SeedDigest: testDigest("7"), CI: true, CandidateProfileID: "W4_H6G"})
	if err != nil || selection.ResourceProfileID != "STABLE_CONTROL" {
		t.Fatalf("selection = %+v/%v", selection, err)
	}
}

func TestSignedKillSwitchIsImmediateMonotonicAndDurable(t *testing.T) {
	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	controller, privateKey, root := testRolloutController(t, now)
	if err := controller.ConfigureRepository("repository-1", LearningBudget{WeeklyPercent: 5, DailyPercent: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.StartAction("repository-1", "action-1", DirectRolloutClass, true); err != nil {
		t.Fatal(err)
	}
	directive := signKillSwitch(t, privateKey, SignedKillSwitch{SchemaVersion: KillSwitchSchema, RepositoryID: "repository-1", Generation: 1, Enabled: true, Reason: "operator-stop", IssuedAt: now, ExpiresAt: now.Add(time.Hour), KeyID: "rollout-key"})
	if err := controller.ApplyKillSwitch(directive); err != nil {
		t.Fatal(err)
	}
	selection, err := controller.SelectAction("repository-1", "action-1", SelectionContext{AssignmentID: "killed", SeedDigest: testDigest("7"), CI: true, CandidateProfileID: "W2_H3G"})
	if err != nil || selection.Reason != "KILL_SWITCH" || selection.ResourceProfileID != "STABLE_CONTROL" {
		t.Fatalf("killed = %+v/%v", selection, err)
	}
	if err := controller.ApplyKillSwitch(directive); err != nil {
		t.Fatalf("idempotent directive = %v", err)
	}
	staleConflict := directive
	staleConflict.Reason = "different-reason"
	if err := controller.ApplyKillSwitch(staleConflict); err == nil {
		t.Fatal("conflicting stale directive was accepted")
	}
	tampered := directive
	tampered.Generation = 2
	if err := controller.ApplyKillSwitch(tampered); err == nil {
		t.Fatal("tampered directive was accepted")
	}
	disable := signKillSwitch(t, privateKey, SignedKillSwitch{SchemaVersion: KillSwitchSchema, RepositoryID: "repository-1", Generation: 2, Enabled: false, Reason: "operator-clear", IssuedAt: now, ExpiresAt: now.Add(time.Hour), KeyID: "rollout-key"})
	if err := controller.ApplyKillSwitch(disable); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenRolloutController(root, func() time.Time { return now }, map[string]ed25519.PublicKey{"rollout-key": privateKey.Public().(ed25519.PublicKey)})
	if err != nil {
		t.Fatal(err)
	}
	selection, err = reopened.SelectAction("repository-1", "action-1", SelectionContext{AssignmentID: "cleared", SeedDigest: testDigest("7"), CI: true, CandidateProfileID: "W2_H3G"})
	if err != nil || selection.Reason != RolloutSuspended {
		t.Fatalf("cleared without revalidation = %+v/%v", selection, err)
	}
}

func TestFallbackBoundaryAndContaminationRejection(t *testing.T) {
	cases := []struct {
		context FallbackContext
		action  string
		kill    bool
	}{
		{FallbackContext{CandidateFailed: true}, "RETRY_ORIGINAL_ONCE", false},
		{FallbackContext{CandidateFailed: true, TaskActionsStarted: true, IsolatedBaselineReady: true, BaselinePassed: true}, "RETURN_ISOLATED_BASELINE", false},
		{FallbackContext{CandidateFailed: true, TaskActionsStarted: true, ManifestHasNoEffects: true}, "RUN_ISOLATED_BASELINE", false},
		{FallbackContext{CandidateFailed: true, TaskActionsStarted: true}, "PRESERVE_FAILURE", true},
	}
	for _, testCase := range cases {
		decision := ResolveFallback(testCase.context)
		if decision.Action != testCase.action || decision.EnableKillSwitch != testCase.kill {
			t.Fatalf("fallback = %+v, want %s/%v", decision, testCase.action, testCase.kill)
		}
	}

	now := time.Date(2026, 7, 31, 16, 0, 0, 0, time.UTC)
	scheduler, err := OpenScheduler(filepath.Join(t.TempDir(), "scheduler"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	entry, _, err := scheduler.Schedule(testRequest("contamination"))
	if err != nil {
		t.Fatal(err)
	}
	candidateContamination := entry
	candidateContamination.Plan.Candidate.WriteNamespace = entry.Plan.Stable.WriteNamespace
	if err := candidateContamination.ValidateIsolation(); err == nil {
		t.Fatal("candidate-to-stable contamination was accepted")
	}
	controlContamination := entry
	controlContamination.Plan.Control.WriteNamespace = entry.Plan.Stable.WriteNamespace
	if err := controlContamination.ValidateIsolation(); err == nil {
		t.Fatal("control-to-stable contamination was accepted")
	}
}

func testRolloutController(t *testing.T, now time.Time) (*RolloutController, ed25519.PrivateKey, string) {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	root := filepath.Join(t.TempDir(), "rollout")
	controller, err := OpenRolloutController(root, func() time.Time { return now }, map[string]ed25519.PublicKey{"rollout-key": privateKey.Public().(ed25519.PublicKey)})
	if err != nil {
		t.Fatal(err)
	}
	return controller, privateKey, root
}

func signKillSwitch(t *testing.T, privateKey ed25519.PrivateKey, directive SignedKillSwitch) SignedKillSwitch {
	t.Helper()
	payload, err := KillSwitchSigningPayload(directive)
	if err != nil {
		t.Fatal(err)
	}
	directive.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return directive
}

func completeRolloutEvidence() RolloutEvidence {
	return RolloutEvidence{CorrectnessPassed: true, SampleReady: true, BudgetAvailable: true, TelemetryComplete: true, ContractComplete: true, QuarantineComplete: true, RevalidationPassed: true}
}
