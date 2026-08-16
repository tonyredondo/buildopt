package launcher

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOptimizeCalibrationGradleOptionsAreSafeAndDeterministic(t *testing.T) {
	got, reason := optimizeCalibrationGradleOptions([]string{
		"--offline", "--max-workers", "4", "--console", "plain",
	})
	want := []string{
		"--offline", "--max-workers=4", "--console=plain", "--build-cache", "--no-scan",
	}
	if reason != "" || !reflect.DeepEqual(got, want) {
		t.Fatalf("calibration options = %q/%q, want %q/empty", got, reason, want)
	}

	got, reason = optimizeCalibrationGradleOptions([]string{
		"--no-build-cache", "--no-scan", "--console=rich",
	})
	if reason != "" || !reflect.DeepEqual(got, []string{"--no-build-cache", "--no-scan", "--console=rich"}) {
		t.Fatalf("explicit calibration options = %q/%q", got, reason)
	}

	for _, options := range [][]string{
		{"--max-workers"},
		{"--unknown"},
		{"--offline", "--offline"},
	} {
		if got, reason := optimizeCalibrationGradleOptions(options); got != nil || reason != "CALIBRATION_OPTIONS_UNSUPPORTED" {
			t.Fatalf("unsafe calibration options %q = %q/%q", options, got, reason)
		}
	}
}

func TestOptimizeCalibrationRequiresEightBalancedPairs(t *testing.T) {
	run := optimizeRun{invocation: optimizeInvocation{
		calibrationPairs: 6, maxBreakEvenBuilds: 30,
	}}
	discovery := optimizeDiscoveryResult{
		Status: optimizeDiscoveryComplete, Reason: "STRUCTURAL_CANDIDATE_DISCOVERED",
	}
	result := run.calibrate(context.Background(), time.Now(), discovery, &strings.Builder{})
	if result.Status != optimizeCalibrationSkipped || result.Reason != optimizeCalibrationReasonPairs ||
		result.Performed || result.Qualified || result.PairsRequested != 6 ||
		result.MaximumBreakEvenBuilds != 30 || result.ProductionAuthorized ||
		result.TestOptimization != "OUT_OF_SCOPE" {
		t.Fatalf("six-pair calibration = %+v", result)
	}
}

func TestOptimizeCalibrationCheckpointRejectsAuthorityAndMetricTampering(t *testing.T) {
	state := optimizeState{
		Phase: "QUALIFIED", LastOutcome: optimizeOutcomeLearning,
		Calibration: optimizeCalibrationResult{
			Status: optimizeCalibrationComplete, Reason: optimizeCalibrationReasonQualified,
			Performed: true, PairsRequested: 8, PairsMeasured: 8,
			ControlMeanMS: 10000, CandidateMeanMS: 7000, MeanSavedMS: 3000,
			ReductionRatio: 0.3, Interval95SavedMS: []float64{2500, 3500}, PositivePairs: 8,
			ControlP95MS: 11000, CandidateP95MS: 7600,
			CalibrationCostMS: 24000, BreakEvenBuilds: 8, MaximumBreakEvenBuilds: 30,
			ValueGatePassed: true, Qualified: true, FallbackSuccessful: true,
			EvidenceSHA256: strings.Repeat("a", 64), DiscoverySHA256: strings.Repeat("b", 64),
			GeneratedFiles:   []string{".buildopt/optimize/v1/calibration/evidence.json"},
			TestOptimization: "OUT_OF_SCOPE",
		},
	}
	if !validOptimizeCalibrationCheckpoint(state) {
		t.Fatalf("valid calibration checkpoint rejected: %+v", state.Calibration)
	}

	mutated := state
	mutated.Calibration.ProductionAuthorized = true
	if validOptimizeCalibrationCheckpoint(mutated) {
		t.Fatal("production-authorized calibration checkpoint was accepted")
	}
	mutated = state
	mutated.Calibration.PairsMeasured = 7
	if validOptimizeCalibrationCheckpoint(mutated) {
		t.Fatal("seven-pair calibration checkpoint was accepted")
	}
	mutated = state
	mutated.Calibration.BreakEvenBuilds = 31
	if validOptimizeCalibrationCheckpoint(mutated) {
		t.Fatal("checkpoint beyond its break-even guardrail was accepted")
	}
}
