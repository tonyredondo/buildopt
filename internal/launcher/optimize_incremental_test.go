package launcher

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

func TestOptimizeIncrementalWallTimeIncludesWrapperWork(t *testing.T) {
	started := time.Unix(100, 0)
	run := &optimizeRun{childExecution: childExecution{
		startedAt:   started,
		completedAt: started.Add(1250 * time.Millisecond),
	}}
	if got := optimizeIncrementalWallTimeMS(run, 375); got != 1625 {
		t.Fatalf("incremental wall time = %d, want 1625", got)
	}
	observation := optimizeIncrementalObservation{
		DurationMS: 1625, IncrementalOverheadMS: 375,
		Economics: optimizeIncrementalEconomics{
			GradleMS: 1250, PreExecutionMS: 125, PostExecutionMS: 250,
			MaterializationMS: 100, OutputVerificationMS: 200, OtherWrapperMS: 75,
		},
	}
	if !validOptimizeIncrementalEconomics(observation) {
		t.Fatal("end-to-end wall-time economics were rejected")
	}
}

func TestExpectedOptimizeIncrementalArmAlternatesBalancedPairs(t *testing.T) {
	want := []struct {
		arm   string
		pair  int
		order string
	}{
		{optimizeIncrementalArmControl, 1, "CONTROL_FIRST"},
		{optimizeIncrementalArmCandidate, 1, "CONTROL_FIRST"},
		{optimizeIncrementalArmCandidate, 2, "CANDIDATE_FIRST"},
		{optimizeIncrementalArmControl, 2, "CANDIDATE_FIRST"},
		{optimizeIncrementalArmControl, 3, "CONTROL_FIRST"},
		{optimizeIncrementalArmCandidate, 3, "CONTROL_FIRST"},
	}
	for index, expected := range want {
		arm, pair, order := expectedOptimizeIncrementalArm(index)
		if arm != expected.arm || pair != expected.pair || order != expected.order {
			t.Fatalf("observation %d = %s/%d/%s, want %s/%d/%s", index, arm, pair, order, expected.arm, expected.pair, expected.order)
		}
	}
}

func TestOptimizeOutputObservationPrecedesGradleArgumentSeparator(t *testing.T) {
	run := optimizeRun{outputObservation: &optimizeOutputObservation{
		initPath:        "/private/output-contract.init.gradle",
		snapshotPath:    "/private/snapshot.json",
		entrypointsJSON: `["assemble"]`,
		impact: buildimpact.InlineObservation{
			InitPath: "/private/impact.init.gradle", OutputPath: "/private/impact.json",
			EntrypointsJSON: `["assemble"]`,
		},
	}}
	invocation := gradleInvocation{
		childArgs:   []string{"./gradlew", "--", "assemble"},
		environment: map[string]string{},
	}

	run.augmentGradleOutputObservation(&invocation)

	want := []string{
		"./gradlew", "--init-script", "/private/output-contract.init.gradle",
		"--init-script", "/private/impact.init.gradle",
		"--", "assemble", "buildoptOutputContract", "buildoptImpactDiscovery",
	}
	if !reflect.DeepEqual(invocation.childArgs, want) {
		t.Fatalf("augmented args = %q, want %q", invocation.childArgs, want)
	}
	if invocation.environment["BUILDOPT_OUTPUT_CONTRACT_SNAPSHOT"] != "/private/snapshot.json" {
		t.Fatalf("snapshot environment = %q", invocation.environment["BUILDOPT_OUTPUT_CONTRACT_SNAPSHOT"])
	}
	if invocation.environment["BUILDOPT_OUTPUT_CONTRACT_ENTRYPOINTS"] != `["assemble"]` {
		t.Fatalf("entrypoint environment = %q", invocation.environment["BUILDOPT_OUTPUT_CONTRACT_ENTRYPOINTS"])
	}
	if invocation.environment["BUILDOPT_IMPACT_DISCOVERY_OUTPUT"] != "/private/impact.json" ||
		invocation.environment["BUILDOPT_IMPACT_DISCOVERY_INLINE"] != "1" {
		t.Fatalf("impact environment = %#v", invocation.environment)
	}
}

func TestOptimizeIncrementalCheckpointRejectsTampering(t *testing.T) {
	repository := t.TempDir()
	invocation := optimizeInvocation{
		repositoryRoot: repository,
		stateRelative:  ".buildopt/optimize/v1",
		bindingSHA256:  optimizeDigest("incremental-binding"),
	}
	digest := optimizeDigest("required-output")
	discovery := optimizeDigest("discovery")
	learning := optimizeIncrementalLearning{
		Status: optimizeIncrementalCollecting, Reason: optimizeIncrementalReasonPending,
		Performed: true, TargetPairs: optimizeRequiredCalibrationPairs,
		PairsCompleted: 1, NextArm: optimizeIncrementalArmCandidate,
		ExpectedOutputSHA256: digest, ExpectedOutputCount: 2,
		DiscoverySHA256: discovery, IncrementalCostMS: 19,
		Baseline: &optimizeIncrementalObservation{
			Sequence: 0, Pair: 0, Arm: optimizeIncrementalArmDiscovery, Order: "BASELINE",
			DurationMS: 100, RequiredOutputSHA256: digest, RequiredOutputCount: 2,
			IncrementalOverheadMS: 7, CapturedAt: "2026-08-20T10:00:00Z",
		},
		Observations: []optimizeIncrementalObservation{
			{Sequence: 1, Pair: 1, Arm: optimizeIncrementalArmControl, Order: "CONTROL_FIRST", DurationMS: 90, RequiredOutputSHA256: digest, RequiredOutputCount: 2, IncrementalOverheadMS: 6, CapturedAt: "2026-08-20T10:01:00Z"},
			{Sequence: 2, Pair: 1, Arm: optimizeIncrementalArmCandidate, Order: "CONTROL_FIRST", DurationMS: 60, RequiredOutputSHA256: digest, RequiredOutputCount: 2, IncrementalOverheadMS: 6, CapturedAt: "2026-08-20T10:02:00Z"},
		},
		FallbackSuccessful: true, TestOptimization: "OUT_OF_SCOPE",
	}
	if err := writeOptimizeIncrementalCheckpoint(invocation, &learning); err != nil {
		t.Fatal(err)
	}
	if err := validateOptimizeIncrementalEvidence(invocation, learning); err != nil {
		t.Fatalf("valid checkpoint rejected: %v", err)
	}
	path := filepath.Join(repository, filepath.FromSlash(learning.GeneratedFiles[0]))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 1
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateOptimizeIncrementalEvidence(invocation, learning); err == nil {
		t.Fatal("tampered incremental checkpoint was accepted")
	}
}

func TestOptimizeIncrementalCandidateRequiresExactObservedOutputs(t *testing.T) {
	repository := t.TempDir()
	output := filepath.Join(repository, "build", "artifact.jar")
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("exact-output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, count, err := hashMeasurementOutputs(repository, []string{"build/*.jar"})
	if err != nil {
		t.Fatal(err)
	}
	newRun := func(expected string) *optimizeRun {
		return &optimizeRun{
			invocation: optimizeInvocation{repositoryRoot: repository},
			previousState: &optimizeState{
				Discovery: optimizeDiscoveryResult{RequiredOutputs: []string{"build/*.jar"}},
				IncrementalLearning: optimizeIncrementalLearning{
					ExpectedOutputSHA256: expected,
					ExpectedOutputCount:  count,
				},
			},
			incrementalArm:       optimizeIncrementalArmCandidate,
			incrementalCandidate: true,
		}
	}
	matching := newRun(digest)
	if matching.captureIncrementalOutput(0) || matching.incrementalFailure != "" {
		t.Fatalf("matching candidate rejected: %+v", matching)
	}
	drifted := newRun(optimizeDigest("different-output"))
	if !drifted.captureIncrementalOutput(0) || drifted.incrementalFailure != optimizeIncrementalReasonOutputDrift {
		t.Fatalf("drifted candidate accepted: %+v", drifted)
	}
	failed := newRun(digest)
	if !failed.captureIncrementalOutput(37) || failed.incrementalFailure != optimizeIncrementalReasonCandidate {
		t.Fatalf("failed candidate did not request fallback: %+v", failed)
	}
}

func TestOptimizeIncrementalCancellationRetainsNativeWithoutRecovery(t *testing.T) {
	run := &optimizeRun{
		incrementalArm:       optimizeIncrementalArmCandidate,
		incrementalCandidate: true,
		childExecution:       childExecution{started: true, cancelled: true},
	}
	if !run.captureIncrementalCancellation() {
		t.Fatal("cancelled incremental observation was not captured")
	}
	if run.incrementalFailure != optimizeIncrementalReasonCancelled {
		t.Fatalf("incremental failure = %q, want %q", run.incrementalFailure, optimizeIncrementalReasonCancelled)
	}
	if run.incrementalFallback.started {
		t.Fatal("cancelled incremental observation unexpectedly started recovery")
	}
}
