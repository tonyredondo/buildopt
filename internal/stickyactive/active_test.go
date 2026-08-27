package stickyactive

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickytrial"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("STICKY_ACTIVE_HELPER") != "1" {
		return
	}
	output := os.Getenv("STICKY_ACTIVE_OUT")
	if output == "" {
		os.Exit(2)
	}
	if delay, _ := strconv.Atoi(os.Getenv("STICKY_ACTIVE_DELAY_MS")); delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if code, _ := strconv.Atoi(os.Getenv("STICKY_ACTIVE_EXIT")); code != 0 {
		os.Exit(code)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		os.Exit(3)
	}
	if err := os.WriteFile(output, []byte(os.Getenv("STICKY_ACTIVE_VALUE")), 0o600); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}

func TestRunnerActivatesOnlyWithExactDecisionAndSuspendsRegression(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	profile, privateKey := testProfile(t, root, now, 0, 0, 25, "same", "same")
	runner, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	runner.execute = deterministicExecutor(t, map[string]time.Duration{
		"candidate": 2 * time.Millisecond,
		"native":    5 * time.Millisecond,
	})
	first, err := runner.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusActiveExecuted || first.Reason != ReasonActive || !first.CandidateExecuted || !first.Counterfactual || !first.ExactOutputs || first.Selected != SelectedCandidate || first.SavingNs != int64(3*time.Millisecond) {
		t.Fatalf("active execution = %+v", first)
	}
	if first.Candidate == nil || first.Native == nil {
		t.Fatal("active execution omitted one arm")
	}

	// A new signed decision generation can change the candidate timing. The
	// same runner is intentionally sticky: a clear regression suspends it and
	// the next invocation cannot execute the candidate again.
	profile.Candidate = helperCommand(t, root, "candidate-regression", "same", 60, 0)
	profile.Native = helperCommand(t, root, "native-regression", "same", 1, 0)
	profile.DecisionRaw = signedActiveDecision(t, privateKey, profile.ExpectedBinding, profile.ActionID, now)
	regressedRunner, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	regressedRunner.execute = deterministicExecutor(t, map[string]time.Duration{
		"candidate-regression": 60 * time.Millisecond,
		"native-regression":    1 * time.Millisecond,
	})
	regressed, err := regressedRunner.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if regressed.Status != StatusSuspended || regressed.Reason != ReasonRegression || !regressed.Suspended || regressed.Selected != SelectedNative || regressed.Candidate == nil || regressed.Native == nil {
		t.Fatalf("regression suspension = %+v", regressed)
	}
	second, err := regressedRunner.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != StatusSuspended || second.Reason != ReasonPreviouslySuspended || second.CandidateExecuted || second.Native == nil {
		t.Fatalf("post-suspension execution = %+v", second)
	}
}

func TestRunWithExecutorUsesCallerOwnedProcessBoundary(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	profile, _ := testProfile(t, root, now, 0, 0, 25, "same", "same")
	runner, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	execution, err := runner.RunWithExecutor(context.Background(), false, func(context.Context, Command, []string) ArmResult {
		calls++
		return ArmResult{Outcome: OutcomeSuccess, DurationNs: int64(time.Millisecond), OutputSHA256: digest("same"), OutputBytes: 4}
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || execution.Status != StatusActiveExecuted || !execution.ExactOutputs {
		t.Fatalf("adapter calls=%d execution=%+v", calls, execution)
	}
	if _, err := runner.RunWithExecutor(context.Background(), false, nil); err == nil {
		t.Fatal("nil executor accepted")
	}
}

func deterministicExecutor(t *testing.T, durations map[string]time.Duration) commandExecutor {
	t.Helper()
	return func(ctx context.Context, command Command, outputs []string) ArmResult {
		t.Helper()
		if err := ctx.Err(); err != nil {
			return ArmResult{Outcome: OutcomeCancelled, ExitCode: 130}
		}
		duration, ok := durations[filepath.Base(command.Dir)]
		if !ok {
			t.Fatalf("no deterministic duration for %s", command.Dir)
		}
		if len(outputs) != 1 || outputs[0] != "out.txt" {
			t.Fatalf("unexpected deterministic outputs: %v", outputs)
		}
		return ArmResult{
			Outcome:      OutcomeSuccess,
			DurationNs:   duration.Nanoseconds(),
			OutputSHA256: digest("deterministic-output"),
			OutputBytes:  4,
		}
	}
}

func TestRunnerFailClosedDecisionAndOutputCases(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 27, 21, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		mutate func(*Profile)
		reason string
	}{
		{name: "bypass", mutate: func(profile *Profile) {}, reason: ReasonBypass},
		{name: "binding drift", mutate: func(profile *Profile) { profile.ExpectedBinding.Workflow = "other/build" }, reason: ReasonBindingMismatch},
		{name: "expiry", mutate: func(profile *Profile) { profile.Now = func() time.Time { return now.Add(2 * time.Hour) } }, reason: ReasonExpiredDecision},
		{name: "revocation", mutate: func(profile *Profile) { profile.RevocationEpoch = 1 }, reason: ReasonRevokedDecision},
		{name: "candidate failure", mutate: func(profile *Profile) { profile.Candidate = helperCommand(t, root, "candidate-failure", "same", 1, 37) }, reason: ReasonCandidateFailure},
		{name: "output mismatch", mutate: func(profile *Profile) {
			profile.Candidate = helperCommand(t, root, "candidate-mismatch", "candidate", 1, 0)
		}, reason: ReasonOutputMismatch},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			profile, _ := testProfile(t, root, now, 1, 1, 1, "same", "same")
			testCase.mutate(&profile)
			runner, err := New(profile)
			if err != nil {
				t.Fatal(err)
			}
			execution, err := runner.Run(context.Background(), testCase.name == "bypass")
			if err != nil {
				t.Fatal(err)
			}
			expectedCandidateExecuted := testCase.name == "candidate failure" || testCase.name == "output mismatch"
			if execution.Reason != testCase.reason || execution.Selected != SelectedNative || execution.Native == nil || execution.CandidateExecuted != expectedCandidateExecuted {
				t.Fatalf("fail-closed execution = %+v", execution)
			}
			if execution.Native.Outcome != OutcomeSuccess {
				t.Fatalf("native fallback = %+v", execution.Native)
			}
		})
	}
}

func TestRunnerCancellationRetainsNativeAndQualificationRejectsCurrentTrial(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 27, 22, 0, 0, 0, time.UTC)
	profile, _ := testProfile(t, root, now, 1, 10, 10, "same", "same")
	runner, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	execution, err := runner.Run(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Reason != ReasonCancelled || execution.CandidateExecuted || execution.Native == nil || execution.Native.Outcome != OutcomeCancelled {
		t.Fatalf("cancelled execution = %+v", execution)
	}

	report := stickytrial.Report{
		Trials: make([]stickytrial.PairedTrial, 4), PositivePairs: 0, ExactOutputPairs: 4,
		CancelledPairs: 0, CandidateMeanMs: 7.534, NativeMeanMs: 6.979,
		MeanSavedMs: -0.555, TotalInvocations: 8,
	}
	qualification := QualifyTrial(report, 4)
	if qualification.Authorized || qualification.Reason != QualificationNegative {
		t.Fatalf("negative trial qualification = %+v", qualification)
	}

	report.Trials = make([]stickytrial.PairedTrial, 4)
	report.PositivePairs, report.ExactOutputPairs, report.CandidateMeanMs, report.NativeMeanMs, report.MeanSavedMs = 4, 4, 5, 7, 2
	qualification = QualifyTrial(report, 4)
	if !qualification.Authorized || qualification.Reason != QualificationAuthorized {
		t.Fatalf("positive trial qualification = %+v", qualification)
	}
}

func testProfile(t *testing.T, root string, now time.Time, candidateDelay, nativeDelay, tolerance int, candidateValue, nativeValue string) (Profile, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	scope := digest("active-scope")
	binding := testBinding(scope)
	actionID := "action/active-test"
	profile := Profile{
		ActionID: actionID, ExpectedBinding: binding,
		PublicKeys:      map[string]ed25519.PublicKey{"owner": publicKey},
		Candidate:       helperCommand(t, root, "candidate", candidateValue, candidateDelay, 0),
		Native:          helperCommand(t, root, "native", nativeValue, nativeDelay, 0),
		RequiredOutputs: []string{"out.txt"}, CounterfactualEvery: 1,
		RegressionTolerancePermille: uint64(tolerance), Now: func() time.Time { return now },
	}
	profile.DecisionRaw = signedActiveDecision(t, privateKey, binding, actionID, now)
	return profile, privateKey
}

func helperCommand(t *testing.T, root, name, value string, delay, exitCode int) Command {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Command{
		Program: executable, Args: []string{"-test.run=TestHelperProcess", "--"}, Dir: dir,
		Env: append(os.Environ(), "STICKY_ACTIVE_HELPER=1", "STICKY_ACTIVE_OUT="+filepath.Join(dir, "out.txt"), "STICKY_ACTIVE_VALUE="+value, "STICKY_ACTIVE_DELAY_MS="+strconv.Itoa(delay), "STICKY_ACTIVE_EXIT="+strconv.Itoa(exitCode)),
	}
}

func signedActiveDecision(t *testing.T, privateKey ed25519.PrivateKey, binding stickydecision.Binding, actionID string, now time.Time) []byte {
	t.Helper()
	decision := stickydecision.Decision{
		SchemaVersion: stickydecision.DecisionSchemaVersion, RecordType: stickydecision.DecisionRecordType,
		DecisionID: "active-decision", StoreGeneration: 1, IdempotencyKey: digest("active-decision"),
		Binding: binding, ActionID: actionID, ActionGeneration: 1,
		QualificationState: "QUARANTINE_VALIDATED", RolloutState: "ACTIVE_IN_CI",
		ExecutionDecision: stickydecision.ExecutionActiveRuntime, PolicyDigest: digest("policy"),
		CacheContractDigest: digest("cache"), EvidenceRefs: []string{digest("trial")},
		IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		Authentication: stickydecision.Authentication{Algorithm: "Ed25519", KeyID: "owner"},
	}
	raw, _, err := stickydecision.SignDecision(decision, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testBinding(scope string) stickydecision.Binding {
	return stickydecision.Binding{
		RepositoryScopeSHA256: scope, Workflow: "benchmark/build", SourceRevision: strings.Repeat("a", 40),
		GradleVersion: "9.6.1", WrapperSHA256: digest("wrapper"), OptionsSHA256: digest("options"),
		OutputContractSHA256: digest("outputs"), BuildOptExecutableSHA256: digest("buildopt"),
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
