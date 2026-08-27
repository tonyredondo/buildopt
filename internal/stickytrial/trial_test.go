package stickytrial

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("STICKY_TRIAL_HELPER") != "1" {
		return
	}
	if delay, _ := strconv.Atoi(os.Getenv("STICKY_TRIAL_SLEEP_MS")); delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	output := os.Getenv("STICKY_TRIAL_OUT")
	if output == "" {
		os.Exit(2)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		os.Exit(3)
	}
	if err := os.WriteFile(output, []byte(os.Getenv("STICKY_TRIAL_VALUE")), 0o600); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}

func TestSchedulerRequiresTrustedCIAndBalancesOrder(t *testing.T) {
	if _, err := NewScheduler(Budget{NaturalRunnerNs: 1_000, MaxExtraPermille: 50, MaxConcurrent: 1}); err == nil {
		t.Fatal("untrusted scheduler was accepted")
	}
	scheduler, err := NewScheduler(Budget{NaturalRunnerNs: 1_000_000, MaxExtraPermille: 50, MaxConcurrent: 1, TrustedCI: true})
	if err != nil {
		t.Fatal(err)
	}
	first, err := scheduler.Assign("trial-1", 20_000)
	if err != nil || first.Order != CandidateFirst || first.Pair != 1 {
		t.Fatalf("first assignment = %+v / %v", first, err)
	}
	if _, err := scheduler.Assign("trial-2", 20_000); err == nil {
		t.Fatal("concurrent assignment was accepted")
	}
	if _, err := scheduler.Complete(first, 10_000); err != nil {
		t.Fatal(err)
	}
	second, err := scheduler.Assign("trial-2", 20_000)
	if err != nil || second.Order != NativeFirst || second.Pair != 2 {
		t.Fatalf("second assignment = %+v / %v", second, err)
	}
	if _, err := scheduler.Complete(second, 20_000); err != nil {
		t.Fatal(err)
	}
	if snapshot := scheduler.Snapshot(); snapshot.UsedNs != 30_000 || snapshot.ActiveTrials != 0 || snapshot.ReservedNs != 0 {
		t.Fatalf("budget snapshot = %+v", snapshot)
	}
}

func TestSchedulerCancellationReleasesUnusedReservation(t *testing.T) {
	scheduler, err := NewScheduler(Budget{NaturalRunnerNs: 1_000_000, MaxExtraPermille: 50, MaxConcurrent: 1, TrustedCI: true})
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := scheduler.Assign("cancelled", 40_000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scheduler.Cancel(assignment, 5_000); err != nil {
		t.Fatal(err)
	}
	if snapshot := scheduler.Snapshot(); snapshot.UsedNs != 5_000 || snapshot.ReservedNs != 0 || snapshot.ActiveTrials != 0 {
		t.Fatalf("cancelled snapshot = %+v", snapshot)
	}
	if _, err := scheduler.Assign("next", 40_000); err != nil {
		t.Fatalf("unused reservation was not released: %v", err)
	}
}

func TestRunPairedRequiresExactOutputsAndTwoInvocations(t *testing.T) {
	root := t.TempDir()
	isolation := testIsolation(root)
	for _, path := range isolation.paths() {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	scheduler, err := NewScheduler(Budget{NaturalRunnerNs: int64(60 * time.Second), MaxExtraPermille: 500, MaxConcurrent: 1, TrustedCI: true})
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := scheduler.Assign("exact", int64(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	candidate := helperCommand(executable, isolation.CandidateDir, "same")
	native := helperCommand(executable, isolation.NativeDir, "same")
	trial, err := RunPaired(context.Background(), assignment, isolation, candidate, native, []string{"build/out.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if trial.Equivalence != EquivalenceExact || trial.Result == ResultInconclusive || trial.InvocationCount != 2 || trial.ActualExtraNs <= 0 {
		t.Fatalf("exact trial = %+v", trial)
	}
	if snapshot := scheduler.Snapshot(); snapshot.ActiveTrials != 0 || snapshot.UsedNs != trial.ActualExtraNs {
		t.Fatalf("completed snapshot = %+v", snapshot)
	}
}

func TestRunPairedWithExecutorUsesCallerOwnedProcessBoundary(t *testing.T) {
	root := t.TempDir()
	isolation := testIsolation(root)
	for _, path := range isolation.paths() {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	scheduler, err := NewScheduler(Budget{NaturalRunnerNs: 1_000_000, MaxExtraPermille: 500, MaxConcurrent: 1, TrustedCI: true})
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := scheduler.Assign("adapter", 100_000)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	execute := func(context.Context, Command, Isolation, []string) ArmResult {
		calls++
		return ArmResult{Outcome: OutcomeSuccess, DurationNs: 10_000, OutputSHA256: Digest("same"), OutputBytes: 4}
	}
	trial, err := RunPairedWithExecutor(context.Background(), assignment, isolation,
		Command{Program: "/bin/true", Dir: isolation.CandidateDir, Env: []string{"PATH=/usr/bin:/bin"}},
		Command{Program: "/bin/true", Dir: isolation.NativeDir, Env: []string{"PATH=/usr/bin:/bin"}},
		[]string{"build/out.txt"}, execute)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || trial.InvocationCount != 2 || trial.Equivalence != EquivalenceExact {
		t.Fatalf("adapter calls=%d trial=%+v", calls, trial)
	}
}

func TestRunPairedRetainsInconclusiveAndCancellation(t *testing.T) {
	root := t.TempDir()
	isolation := testIsolation(root)
	for _, path := range isolation.paths() {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	scheduler, err := NewScheduler(Budget{NaturalRunnerNs: int64(60 * time.Second), MaxExtraPermille: 500, MaxConcurrent: 1, TrustedCI: true})
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := scheduler.Assign("different", int64(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	candidate := helperCommand(executable, isolation.CandidateDir, "candidate")
	native := helperCommand(executable, isolation.NativeDir, "native")
	trial, err := RunPaired(context.Background(), assignment, isolation, candidate, native, []string{"build/out.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if trial.Equivalence != EquivalenceNone || trial.Result != ResultInconclusive {
		t.Fatalf("different output trial = %+v", trial)
	}

	assignment, err = scheduler.Assign("cancelled", int64(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	cancel := context.Background()
	cancelContext, cancelFunc := context.WithTimeout(cancel, 20*time.Millisecond)
	defer cancelFunc()
	sleeping := helperCommand(executable, isolation.CandidateDir, "cancelled")
	sleeping.Env = append(sleeping.Env, "STICKY_TRIAL_SLEEP_MS=1000")
	_, err = RunPaired(cancelContext, assignment, isolation, sleeping, native, []string{"build/out.txt"})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("cancellation error = %v", err)
	}
	if snapshot := scheduler.Snapshot(); snapshot.ActiveTrials != 0 || snapshot.ReservedNs != 0 {
		t.Fatalf("cancelled trial retained reservation = %+v", snapshot)
	}
}

func helperCommand(executable, dir, value string) Command {
	output := filepath.Join(dir, "build", "out.txt")
	env := append([]string(nil), os.Environ()...)
	env = append(env, "STICKY_TRIAL_HELPER=1", "STICKY_TRIAL_OUT="+output, "STICKY_TRIAL_VALUE="+value)
	return Command{Program: executable, Args: []string{"-test.run=TestHelperProcess"}, Dir: dir, Env: env}
}

func testIsolation(root string) Isolation {
	return Isolation{
		CandidateDir: filepath.Join(root, "candidate"), NativeDir: filepath.Join(root, "native"),
		CandidateGradleHome: filepath.Join(root, "candidate-gradle"), NativeGradleHome: filepath.Join(root, "native-gradle"),
		CandidateCache: filepath.Join(root, "candidate-cache"), NativeCache: filepath.Join(root, "native-cache"),
		CandidateState: filepath.Join(root, "candidate-state"), NativeState: filepath.Join(root, "native-state"),
	}
}
