package launcher

import (
	"strings"
	"testing"
)

func TestOptimizeValueStateCountsOnlySuccessfulExactReplays(t *testing.T) {
	calibration := optimizeCalibrationResult{
		Status: optimizeCalibrationComplete, MeanSavedMS: 1250, CalibrationCostMS: 5000,
	}
	selection := optimizeSelectionResult{Selected: true, DurationNS: 250000}

	first := nextOptimizeValueState(optimizeValueState{}, calibration, selection, 0)
	if first.MatchingReplayCount != 1 || first.ProjectedGrossSavedMS != 1250 ||
		first.CumulativeSelectionOverheadMS != .25 ||
		first.ProjectedCumulativeNetSavedMS != -3750.25 {
		t.Fatalf("first replay value = %+v", first)
	}
	failed := nextOptimizeValueState(first, calibration, selection, 37)
	if failed.MatchingReplayCount != 1 || failed.ProjectedGrossSavedMS != 1250 ||
		failed.CumulativeSelectionOverheadMS != .5 ||
		failed.ProjectedCumulativeNetSavedMS != -3750.5 {
		t.Fatalf("failed replay value = %+v", failed)
	}
	second := nextOptimizeValueState(failed, calibration, selection, 0)
	if second.MatchingReplayCount != 2 || second.ProjectedGrossSavedMS != 2500 ||
		second.CumulativeSelectionOverheadMS != .75 ||
		second.ProjectedCumulativeNetSavedMS != -2500.75 {
		t.Fatalf("second replay value = %+v", second)
	}
}

func TestOptimizeValueReportExplainsObservedValueWithoutInventingLifetime(t *testing.T) {
	result := optimizeResult{
		Outcome: "QUALIFIED_AND_USED", Reason: optimizeSelectionReasonSelected,
		Phase: "ACTIVE", CompletedAt: "2026-08-16T12:00:00Z",
		Native: optimizeNativeResult{Authoritative: false},
		Discovery: optimizeDiscoveryResult{Graph: optimizeDiscoveryGraph{
			TotalProjects: 20, SelectedProjects: 5, OmittedProjects: 15,
		}},
		Calibration: optimizeCalibrationResult{
			Status: optimizeCalibrationComplete, PairsMeasured: 8, PositivePairs: 8,
			ControlMeanMS: 10000, CandidateMeanMS: 7000, MeanSavedMS: 3000,
			ReductionRatio: .3, Interval95SavedMS: []float64{2500, 3500},
			ControlP95MS: 11000, CandidateP95MS: 7600,
			CalibrationCostMS: 12000, BreakEvenBuilds: 4, MaximumBreakEvenBuilds: 30,
			FallbackSuccessful: true, EvidenceSHA256: strings.Repeat("a", 64),
			DiscoverySHA256: strings.Repeat("b", 64),
			GeneratedFiles:  []string{".buildopt/optimize/v1/calibration/evidence.json"},
		},
		Selection: optimizeSelectionResult{Selected: true, DurationNS: 500000},
		Value: optimizeValueState{
			MatchingReplayCount: 2, ProjectedGrossSavedMS: 6000,
			CumulativeSelectionOverheadMS: 1, ProjectedCumulativeNetSavedMS: -6001,
		},
		GeneratedFiles: optimizeGeneratedFiles{Result: ".buildopt/optimize/v1/result.json"},
	}

	report := newOptimizeValueReport(result)
	if report.SchemaVersion != optimizeValueReportSchemaVersion || report.Graph.ReductionRatio != .75 ||
		report.Performance.ObservedNetMeanSavedMS != 3000 || report.Performance.P95DeltaMS != -3400 ||
		report.Economics.ProjectedCumulativeNetSavedMS != -6001 ||
		report.Economics.ProjectedPaybackRemainingMS != 6001 ||
		report.Economics.ExpectedUsefulLifetime.Status != "UNAVAILABLE" ||
		report.Economics.ExpectedUsefulLifetime.EstimatedMatchingBuilds != nil ||
		report.Fallback.CurrentBuildUsedFallback || report.ProductionAuthorized {
		t.Fatalf("value report = %+v", report)
	}
	markdown := string(renderOptimizeValueMarkdown(report))
	for _, fragment := range []string{
		"20-project", "15 projects omitted (75.0%)", "3000 ms/build (30.0%)",
		"95% interval", "break-even is **4 matching builds**", "Expected useful lifetime: **unavailable**",
		"Mechanism percentages are never added together",
	} {
		if !strings.Contains(markdown, fragment) {
			t.Fatalf("value markdown does not contain %q:\n%s", fragment, markdown)
		}
	}
}

func TestOptimizeValueReportMakesNoTimingClaimWithoutCalibration(t *testing.T) {
	result := optimizeResult{
		Outcome: optimizeOutcomeNative, Reason: "AMBIGUOUS_COMPARISON_BASE",
		Phase: "NATIVE_RETAINED", CompletedAt: "2026-08-16T12:00:00Z",
		Native:         optimizeNativeResult{Authoritative: true, Started: true, ExitCode: 0},
		GeneratedFiles: optimizeGeneratedFiles{Result: ".buildopt/optimize/v1/result.json"},
	}
	report := newOptimizeValueReport(result)
	if report.Graph.Status != "UNAVAILABLE" || report.Performance.Status != "UNAVAILABLE" ||
		report.Economics.Status != "UNAVAILABLE" || !report.Fallback.CurrentBuildUsedFallback ||
		!report.Fallback.Authoritative || !report.Fallback.Successful ||
		report.Fallback.Reason != "AMBIGUOUS_COMPARISON_BASE" {
		t.Fatalf("native value report = %+v", report)
	}
	markdown := string(renderOptimizeValueMarkdown(report))
	if !strings.Contains(markdown, "no wall-time claim is made") ||
		!strings.Contains(markdown, "Exact reason: `AMBIGUOUS_COMPARISON_BASE`") {
		t.Fatalf("native markdown = %s", markdown)
	}
}

func TestOptimizeValueReportUsesExactSelectionFallbackReason(t *testing.T) {
	result := optimizeResult{
		Outcome: optimizeOutcomeLearning, Reason: optimizePortfolioReasonReused,
		Phase: "QUALIFIED", CompletedAt: "2026-08-16T12:00:00Z",
		Native: optimizeNativeResult{Authoritative: true, Started: true, ExitCode: 0},
		Selection: optimizeSelectionResult{
			Status: optimizeSelectionRetained, Reason: optimizeSelectionReasonBindings,
		},
		GeneratedFiles: optimizeGeneratedFiles{Result: ".buildopt/optimize/v1/result.json"},
	}
	report := newOptimizeValueReport(result)
	if report.Reason != optimizePortfolioReasonReused ||
		report.Fallback.Reason != optimizeSelectionReasonBindings {
		t.Fatalf("fallback reason = %+v", report)
	}
}
