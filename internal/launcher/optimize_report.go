package launcher

import (
	"fmt"
	"math"
	"strings"
)

const optimizeValueReportSchemaVersion = "buildopt.poc/magic-wow-report/v1"

type optimizeValueState struct {
	MatchingReplayCount           int     `json:"matchingReplayCount"`
	ProjectedGrossSavedMS         float64 `json:"projectedGrossSavedMs"`
	CumulativeSelectionOverheadMS float64 `json:"cumulativeSelectionOverheadMs"`
	ProjectedCumulativeNetSavedMS float64 `json:"projectedCumulativeNetSavedMs"`
}

type optimizeValueReport struct {
	SchemaVersion        string                   `json:"schemaVersion"`
	GeneratedAt          string                   `json:"generatedAt"`
	Outcome              string                   `json:"outcome"`
	Reason               string                   `json:"reason"`
	Phase                string                   `json:"phase"`
	Graph                optimizeValueGraph       `json:"graph"`
	Performance          optimizeValuePerformance `json:"performance"`
	Economics            optimizeValueEconomics   `json:"economics"`
	Fallback             optimizeValueFallback    `json:"fallback"`
	Sources              optimizeValueSources     `json:"sources"`
	ProofOfConcept       bool                     `json:"proofOfConcept"`
	ProductionAuthorized bool                     `json:"productionAuthorized"`
	TestOptimization     string                   `json:"testOptimization"`
}

type optimizeValueGraph struct {
	Status           string  `json:"status"`
	FullProjects     int     `json:"fullProjects"`
	SelectedProjects int     `json:"selectedProjects"`
	OmittedProjects  int     `json:"omittedProjects"`
	ReductionRatio   float64 `json:"reductionRatio"`
}

type optimizeValuePerformance struct {
	Status                     string    `json:"status"`
	Pairs                      int       `json:"pairs"`
	PositivePairs              int       `json:"positivePairs"`
	OptimizedNativeMeanMS      float64   `json:"optimizedNativeMeanMs"`
	BuildOptMeanMS             float64   `json:"buildOptMeanMs"`
	ObservedNetMeanSavedMS     float64   `json:"observedNetMeanSavedMs"`
	ObservedReductionRatio     float64   `json:"observedReductionRatio"`
	Interval95SavedMS          []float64 `json:"interval95SavedMs"`
	OptimizedNativeP95MS       float64   `json:"optimizedNativeP95Ms"`
	BuildOptP95MS              float64   `json:"buildOptP95Ms"`
	P95DeltaMS                 float64   `json:"p95DeltaMs"`
	CurrentSelectionOverheadMS float64   `json:"currentSelectionOverheadMs"`
	OutputsEquivalent          bool      `json:"outputsEquivalent"`
	LauncherOverheadIncluded   bool      `json:"launcherOverheadIncluded"`
	UnavailableReason          string    `json:"unavailableReason"`
}

type optimizeValueEconomics struct {
	Status                        string                   `json:"status"`
	CalibrationCostMS             int64                    `json:"calibrationCostMs"`
	BreakEvenBuilds               int                      `json:"breakEvenBuilds"`
	MaximumBreakEvenBuilds        int                      `json:"maximumBreakEvenBuilds"`
	ExpectedUsefulLifetime        optimizeExpectedLifetime `json:"expectedUsefulLifetime"`
	MatchingReplayCount           int                      `json:"matchingReplayCount"`
	ProjectedGrossSavedMS         float64                  `json:"projectedGrossSavedMs"`
	CumulativeSelectionOverheadMS float64                  `json:"cumulativeSelectionOverheadMs"`
	ProjectedCumulativeNetSavedMS float64                  `json:"projectedCumulativeNetSavedMs"`
	ProjectedPaybackRemainingMS   float64                  `json:"projectedPaybackRemainingMs"`
	ProjectionBasis               string                   `json:"projectionBasis"`
	UnavailableReason             string                   `json:"unavailableReason"`
}

type optimizeExpectedLifetime struct {
	Status                  string `json:"status"`
	EstimatedMatchingBuilds *int   `json:"estimatedMatchingBuilds"`
	Reason                  string `json:"reason"`
}

type optimizeValueFallback struct {
	Mode                     string `json:"mode"`
	CurrentBuildUsedFallback bool   `json:"currentBuildUsedFallback"`
	Authoritative            bool   `json:"authoritative"`
	Successful               bool   `json:"successful"`
	Reason                   string `json:"reason"`
}

type optimizeValueSources struct {
	ResultFile      string `json:"resultFile"`
	CalibrationFile string `json:"calibrationFile"`
	EvidenceSHA256  string `json:"evidenceSha256"`
	DiscoverySHA256 string `json:"discoverySha256"`
}

func nextOptimizeValueState(
	previous optimizeValueState,
	calibration optimizeCalibrationResult,
	selection optimizeSelectionResult,
	exitCode int,
) optimizeValueState {
	if calibration.Status != optimizeCalibrationComplete {
		return optimizeValueState{}
	}
	result := previous
	if selection.Selected {
		result.CumulativeSelectionOverheadMS += float64(selection.DurationNS) / 1_000_000
		if exitCode == 0 {
			result.MatchingReplayCount++
		}
	}
	result.ProjectedGrossSavedMS = float64(result.MatchingReplayCount) * calibration.MeanSavedMS
	result.ProjectedCumulativeNetSavedMS = result.ProjectedGrossSavedMS -
		float64(calibration.CalibrationCostMS) - result.CumulativeSelectionOverheadMS
	return result
}

func validOptimizeValueState(state optimizeState) bool {
	value := state.Value
	if value.MatchingReplayCount < 0 || value.ProjectedGrossSavedMS < 0 ||
		value.CumulativeSelectionOverheadMS < 0 {
		return false
	}
	if state.Calibration.Status != optimizeCalibrationComplete {
		return value == (optimizeValueState{})
	}
	expectedGross := float64(value.MatchingReplayCount) * state.Calibration.MeanSavedMS
	expectedNet := expectedGross - float64(state.Calibration.CalibrationCostMS) -
		value.CumulativeSelectionOverheadMS
	return value.ProjectedGrossSavedMS == expectedGross &&
		value.ProjectedCumulativeNetSavedMS == expectedNet
}

func newOptimizeValueReport(result optimizeResult) optimizeValueReport {
	fallbackReason := result.Reason
	if result.Selection.Status == optimizeSelectionRetained {
		fallbackReason = result.Selection.Reason
	}
	report := optimizeValueReport{
		SchemaVersion: optimizeValueReportSchemaVersion,
		GeneratedAt:   result.CompletedAt, Outcome: result.Outcome,
		Reason: result.Reason, Phase: result.Phase,
		Graph: optimizeValueGraph{Status: "UNAVAILABLE"},
		Performance: optimizeValuePerformance{
			Status: "UNAVAILABLE", Interval95SavedMS: []float64{},
			UnavailableReason: "CALIBRATION_EVIDENCE_UNAVAILABLE",
		},
		Economics: optimizeValueEconomics{
			Status: "UNAVAILABLE",
			ExpectedUsefulLifetime: optimizeExpectedLifetime{
				Status: "UNAVAILABLE", EstimatedMatchingBuilds: nil,
				Reason: "EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT",
			},
			ProjectionBasis: "NONE", UnavailableReason: "CALIBRATION_EVIDENCE_UNAVAILABLE",
		},
		Fallback: optimizeValueFallback{
			Mode: "OPTIMIZED_NATIVE", CurrentBuildUsedFallback: result.Native.Authoritative,
			Authoritative: result.Native.Authoritative, Successful: result.Native.ExitCode == 0,
			Reason: fallbackReason,
		},
		Sources: optimizeValueSources{
			ResultFile:      result.GeneratedFiles.Result,
			EvidenceSHA256:  result.Calibration.EvidenceSHA256,
			DiscoverySHA256: result.Calibration.DiscoverySHA256,
		},
		ProofOfConcept: true, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	if len(result.Calibration.GeneratedFiles) == 1 {
		report.Sources.CalibrationFile = result.Calibration.GeneratedFiles[0]
	}
	if result.Discovery.Graph.TotalProjects > 0 {
		report.Graph = optimizeValueGraph{
			Status: "OBSERVED", FullProjects: result.Discovery.Graph.TotalProjects,
			SelectedProjects: result.Discovery.Graph.SelectedProjects,
			OmittedProjects:  result.Discovery.Graph.OmittedProjects,
			ReductionRatio: float64(result.Discovery.Graph.OmittedProjects) /
				float64(result.Discovery.Graph.TotalProjects),
		}
	}
	if result.Calibration.Status == optimizeCalibrationComplete ||
		result.Calibration.Status == optimizeCalibrationRemoteQualified {
		calibration := result.Calibration
		performanceStatus := "OBSERVED_CALIBRATION"
		if calibration.Status == optimizeCalibrationRemoteQualified {
			performanceStatus = "REMOTE_EVIDENCE_REVALIDATED"
		}
		report.Performance = optimizeValuePerformance{
			Status: performanceStatus, Pairs: calibration.PairsMeasured,
			PositivePairs:              calibration.PositivePairs,
			OptimizedNativeMeanMS:      calibration.ControlMeanMS,
			BuildOptMeanMS:             calibration.CandidateMeanMS,
			ObservedNetMeanSavedMS:     calibration.MeanSavedMS,
			ObservedReductionRatio:     calibration.ReductionRatio,
			Interval95SavedMS:          append([]float64(nil), calibration.Interval95SavedMS...),
			OptimizedNativeP95MS:       calibration.ControlP95MS,
			BuildOptP95MS:              calibration.CandidateP95MS,
			P95DeltaMS:                 calibration.CandidateP95MS - calibration.ControlP95MS,
			CurrentSelectionOverheadMS: float64(result.Selection.DurationNS) / 1_000_000,
			OutputsEquivalent:          true, LauncherOverheadIncluded: true,
		}
		if calibration.Status == optimizeCalibrationRemoteQualified {
			report.Economics.UnavailableReason = "REMOTE_CALIBRATION_COST_NOT_REPUBLISHED_WITH_PORTFOLIO"
			report.Economics.ExpectedUsefulLifetime.Reason = "CROSS_COMMIT_MATCH_COUNT_NOT_YET_OBSERVED"
		} else {
			paybackRemaining := float64(calibration.CalibrationCostMS) - result.Value.ProjectedGrossSavedMS +
				result.Value.CumulativeSelectionOverheadMS
			if paybackRemaining < 0 {
				paybackRemaining = 0
			}
			report.Economics = optimizeValueEconomics{
				Status: "CALIBRATION_PROJECTION", CalibrationCostMS: calibration.CalibrationCostMS,
				BreakEvenBuilds:               optimizeBreakEven(calibration.CalibrationCostMS, calibration.MeanSavedMS),
				MaximumBreakEvenBuilds:        calibration.MaximumBreakEvenBuilds,
				ExpectedUsefulLifetime:        report.Economics.ExpectedUsefulLifetime,
				MatchingReplayCount:           result.Value.MatchingReplayCount,
				ProjectedGrossSavedMS:         result.Value.ProjectedGrossSavedMS,
				CumulativeSelectionOverheadMS: result.Value.CumulativeSelectionOverheadMS,
				ProjectedCumulativeNetSavedMS: result.Value.ProjectedCumulativeNetSavedMS,
				ProjectedPaybackRemainingMS:   paybackRemaining,
				ProjectionBasis:               "OBSERVED_CALIBRATION_MEAN_TIMES_SUCCESSFUL_EXACT_REPLAYS_MINUS_CALIBRATION_AND_SELECTION_COST",
			}
		}
	}
	if result.Selection.Selected {
		report.Fallback = optimizeValueFallback{
			Mode: "OPTIMIZED_NATIVE", CurrentBuildUsedFallback: false,
			Authoritative: false, Successful: result.Calibration.FallbackSuccessful,
			Reason: "AVAILABLE_ON_ANY_BINDING_DRIFT_OR_UNCERTAINTY",
		}
	}
	return report
}

func renderOptimizeValueMarkdown(report optimizeValueReport) []byte {
	var builder strings.Builder
	fmt.Fprintln(&builder, "# BuildOpt value report")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "**Outcome:** `%s` — `%s`\n\n", report.Outcome, report.Reason)
	fmt.Fprintln(&builder, "This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.")
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Work reduction")
	fmt.Fprintln(&builder)
	if report.Graph.Status == "OBSERVED" {
		fmt.Fprintf(&builder, "BuildOpt compared the full **%d-project** graph with a **%d-project** selected graph: **%d projects omitted (%.1f%%)**. Graph reduction alone is not counted as value.\n",
			report.Graph.FullProjects, report.Graph.SelectedProjects, report.Graph.OmittedProjects,
			report.Graph.ReductionRatio*100)
	} else {
		fmt.Fprintln(&builder, "No complete selective graph was available, so no graph-reduction claim is made.")
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Measured wall-time value")
	fmt.Fprintln(&builder)
	if report.Performance.Status == "OBSERVED_CALIBRATION" {
		fmt.Fprintf(&builder, "Across **%d balanced pairs**, optimized native Gradle averaged **%.0f ms** and BuildOpt averaged **%.0f ms**. The observed installed-path saving was **%.0f ms/build (%.1f%%)**, with a paired 95%% interval of **%.0f to %.0f ms** and **%d/%d** positive pairs.\n",
			report.Performance.Pairs, report.Performance.OptimizedNativeMeanMS,
			report.Performance.BuildOptMeanMS, report.Performance.ObservedNetMeanSavedMS,
			report.Performance.ObservedReductionRatio*100,
			report.Performance.Interval95SavedMS[0], report.Performance.Interval95SavedMS[1],
			report.Performance.PositivePairs, report.Performance.Pairs)
		fmt.Fprintf(&builder, "Tail check: native p95 **%.0f ms**, BuildOpt p95 **%.0f ms** (delta **%+.0f ms**). Required outputs were equivalent and launcher overhead was included.\n",
			report.Performance.OptimizedNativeP95MS, report.Performance.BuildOptP95MS,
			report.Performance.P95DeltaMS)
	} else {
		fmt.Fprintf(&builder, "No comparable calibration evidence is available (`%s`), so no wall-time claim is made.\n", report.Performance.UnavailableReason)
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Learning cost and payback")
	fmt.Fprintln(&builder)
	if report.Economics.Status == "CALIBRATION_PROJECTION" {
		fmt.Fprintf(&builder, "Calibration cost **%d ms**. At the observed mean saving, break-even is **%d matching builds** (owner limit: **%d**). Successful exact replays counted: **%d**. Projected cumulative net value so far: **%+.0f ms**, after calibration and **%.3f ms** of selection overhead; projected payback remaining: **%.0f ms**.\n",
			report.Economics.CalibrationCostMS, report.Economics.BreakEvenBuilds,
			report.Economics.MaximumBreakEvenBuilds, report.Economics.MatchingReplayCount,
			report.Economics.ProjectedCumulativeNetSavedMS,
			report.Economics.CumulativeSelectionOverheadMS,
			report.Economics.ProjectedPaybackRemainingMS)
		fmt.Fprintf(&builder, "Expected useful lifetime: **unavailable** — `%s`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.\n",
			report.Economics.ExpectedUsefulLifetime.Reason)
	} else {
		fmt.Fprintln(&builder, "Calibration economics are unavailable because no comparable timing evidence exists.")
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Fallback")
	fmt.Fprintln(&builder)
	if report.Fallback.CurrentBuildUsedFallback {
		fmt.Fprintf(&builder, "The current build used **optimized native Gradle**. Exact reason: `%s`. Native result authoritative: **%t**; build successful: **%t**.\n",
			report.Fallback.Reason, report.Fallback.Authoritative, report.Fallback.Successful)
	} else {
		fmt.Fprintf(&builder, "The qualified selective profile ran. Optimized native Gradle remains the automatic fallback: `%s`.\n", report.Fallback.Reason)
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.")
	return []byte(builder.String())
}

func optimizeBreakEven(cost int64, meanSaved float64) int {
	if cost < 1 || meanSaved <= 0 {
		return 0
	}
	return int(math.Ceil(float64(cost) / meanSaved))
}
