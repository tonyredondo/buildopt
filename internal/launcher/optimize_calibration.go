package launcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	optimizeBindingCalibration              = "CALIBRATION_COMPLETE"
	optimizeCalibrationComplete             = "COMPLETE"
	optimizeCalibrationRetained             = "NATIVE_RETAINED"
	optimizeCalibrationSkipped              = "SKIPPED"
	optimizeCalibrationReasonQualified      = "CANDIDATE_CALIBRATION_QUALIFIED"
	optimizeCalibrationReasonNoValue        = "CALIBRATION_VALUE_NOT_PROVEN"
	optimizeCalibrationReasonBreakEven      = "CALIBRATION_BREAK_EVEN_EXCEEDED"
	optimizeCalibrationReasonPairs          = "CALIBRATION_PAIR_BUDGET_INSUFFICIENT"
	optimizeCalibrationReasonUnavailable    = "CALIBRATION_EXECUTION_FAILED"
	optimizeCalibrationReasonCancelled      = "CALIBRATION_CANCELLED"
	optimizeCalibrationEvidenceFile         = "evidence.json"
	optimizeRequiredCalibrationPairs        = 8
	optimizeMinimumPositiveCalibrationPairs = 6
)

type optimizeCalibrationResult struct {
	Status                 string    `json:"status"`
	Reason                 string    `json:"reason"`
	Performed              bool      `json:"performed"`
	Reused                 bool      `json:"reused"`
	PairsRequested         int       `json:"pairsRequested"`
	PairsMeasured          int       `json:"pairsMeasured"`
	ControlMeanMS          float64   `json:"controlMeanMs"`
	CandidateMeanMS        float64   `json:"candidateMeanMs"`
	MeanSavedMS            float64   `json:"meanSavedMs"`
	ReductionRatio         float64   `json:"reductionRatio"`
	Interval95SavedMS      []float64 `json:"interval95SavedMs"`
	PositivePairs          int       `json:"positivePairs"`
	ControlP95MS           float64   `json:"controlP95Ms"`
	CandidateP95MS         float64   `json:"candidateP95Ms"`
	QualificationPolicy    string    `json:"qualificationPolicy,omitempty"`
	CalibrationCostMS      int64     `json:"calibrationCostMs"`
	BreakEvenBuilds        int       `json:"breakEvenBuilds"`
	MaximumBreakEvenBuilds int       `json:"maximumBreakEvenBuilds"`
	ValueGatePassed        bool      `json:"valueGatePassed"`
	Qualified              bool      `json:"qualified"`
	FallbackSuccessful     bool      `json:"fallbackSuccessful"`
	EvidenceSHA256         string    `json:"evidenceSha256"`
	DiscoverySHA256        string    `json:"discoverySha256"`
	GeneratedFiles         []string  `json:"generatedFiles"`
	ProductionAuthorized   bool      `json:"productionAuthorized"`
	TestOptimization       string    `json:"testOptimization"`
}

func validOptimizeCalibrationCheckpoint(state optimizeState) bool {
	calibration := state.Calibration
	if calibration.ProductionAuthorized {
		return false
	}
	switch calibration.Status {
	case "":
		return state.Phase == optimizePhaseUnseen && !state.BuildStarted &&
			calibration.Reason == "" && calibration.TestOptimization == ""
	case optimizeCalibrationSkipped:
		return !calibration.Performed && calibration.Reason != "" &&
			len(calibration.GeneratedFiles) == 0 && len(calibration.Interval95SavedMS) == 0 &&
			calibration.ControlP95MS == 0 && calibration.CandidateP95MS == 0 &&
			calibration.TestOptimization == "OUT_OF_SCOPE" &&
			optimizeStringIn(state.Phase, "DISCOVERED", "CALIBRATING", "NATIVE_RETAINED")
	case optimizeCalibrationRetained:
		return state.Phase == "NATIVE_RETAINED" && !calibration.Performed &&
			calibration.Reason != "" && len(calibration.GeneratedFiles) == 0 &&
			len(calibration.Interval95SavedMS) == 0 && calibration.ControlP95MS == 0 &&
			calibration.CandidateP95MS == 0 && calibration.TestOptimization == "OUT_OF_SCOPE"
	case optimizeCalibrationComplete:
		if !calibration.Performed || calibration.PairsMeasured != optimizeRequiredCalibrationPairs ||
			calibration.PairsRequested < optimizeRequiredCalibrationPairs || calibration.CalibrationCostMS < 1 ||
			calibration.MaximumBreakEvenBuilds < 1 || calibration.MaximumBreakEvenBuilds > 1000 ||
			len(calibration.Interval95SavedMS) != 2 || len(calibration.GeneratedFiles) != 1 ||
			!validOptimizeGeneratedPath(calibration.GeneratedFiles[0]) ||
			!validOptimizeSHA(calibration.EvidenceSHA256) || !validOptimizeSHA(calibration.DiscoverySHA256) ||
			calibration.ControlP95MS <= 0 || calibration.CandidateP95MS <= 0 ||
			!validOptimizeCalibrationQualificationPolicy(calibration.QualificationPolicy) ||
			!calibration.FallbackSuccessful || calibration.TestOptimization != "OUT_OF_SCOPE" ||
			calibration.ValueGatePassed != validOptimizeCalibrationValueGate(calibration) {
			return false
		}
		if calibration.Qualified {
			qualifiedState := state.Phase == "QUALIFIED" && state.LastOutcome == optimizeOutcomeLearning
			replayState := optimizeStringIn(state.Phase, "ACTIVE", "STALE") && state.LastOutcome == "QUALIFIED_AND_USED"
			return (qualifiedState || replayState) &&
				calibration.Reason == optimizeCalibrationReasonQualified && calibration.ValueGatePassed &&
				calibration.BreakEvenBuilds > 0 && calibration.BreakEvenBuilds <= calibration.MaximumBreakEvenBuilds
		}
		return state.Phase == "NATIVE_RETAINED" && state.LastOutcome == optimizeOutcomeNative &&
			optimizeStringIn(calibration.Reason, optimizeCalibrationReasonNoValue, optimizeCalibrationReasonBreakEven)
	case optimizeCalibrationRemoteQualified:
		selectedReplay := optimizeStringIn(state.Phase, "ACTIVE", "STALE") &&
			state.LastOutcome == "QUALIFIED_AND_USED" && state.Selection.Selected
		refreshedNative := state.Phase == "QUALIFIED" && state.LastOutcome == optimizeOutcomeLearning &&
			!state.Selection.Selected && state.Portfolio.Reason == optimizePortfolioReasonRefreshed
		return (selectedReplay || refreshedNative) &&
			!calibration.Performed && calibration.Reused &&
			calibration.Reason == optimizeCalibrationReasonQualified &&
			calibration.PairsRequested == optimizeRequiredCalibrationPairs &&
			calibration.PairsMeasured == optimizeRequiredCalibrationPairs &&
			validOptimizeCalibrationQualificationPolicy(calibration.QualificationPolicy) &&
			validOptimizeCalibrationValueGate(calibration) &&
			calibration.CalibrationCostMS == 0 && calibration.BreakEvenBuilds == 0 &&
			calibration.MaximumBreakEvenBuilds >= 1 && calibration.MaximumBreakEvenBuilds <= 1000 &&
			calibration.ValueGatePassed && calibration.Qualified && calibration.FallbackSuccessful &&
			validOptimizeSHA(calibration.EvidenceSHA256) && validOptimizeSHA(calibration.DiscoverySHA256) &&
			len(calibration.GeneratedFiles) == 1 && validOptimizeGeneratedPath(calibration.GeneratedFiles[0]) &&
			calibration.TestOptimization == "OUT_OF_SCOPE"
	default:
		return false
	}
}

func validOptimizeCalibrationValueGate(calibration optimizeCalibrationResult) bool {
	minimumPositivePairs := optimizeRequiredCalibrationPairs
	p95NonRegressive := true
	if calibration.QualificationPolicy == profilediscovery.StructuralQualificationRobust7Of8P95 {
		minimumPositivePairs = 7
		p95NonRegressive = calibration.CandidateP95MS <= calibration.ControlP95MS
	} else if calibration.QualificationPolicy == profilediscovery.StructuralQualificationRobust6Of8AlternatingP95 {
		minimumPositivePairs = optimizeMinimumPositiveCalibrationPairs
		p95NonRegressive = calibration.CandidateP95MS <= calibration.ControlP95MS
	}
	return calibration.ControlMeanMS > 0 && calibration.CandidateMeanMS > 0 &&
		calibration.MeanSavedMS >= 500 && calibration.ReductionRatio >= 0.02 &&
		len(calibration.Interval95SavedMS) == 2 && calibration.Interval95SavedMS[0] > 0 &&
		calibration.PositivePairs >= minimumPositivePairs &&
		calibration.PositivePairs <= optimizeRequiredCalibrationPairs &&
		calibration.ControlP95MS > 0 && calibration.CandidateP95MS > 0 &&
		p95NonRegressive
}

func validOptimizeCalibrationQualificationPolicy(policy string) bool {
	return policy == "" ||
		policy == profilediscovery.StructuralQualificationRobust7Of8P95 ||
		policy == profilediscovery.StructuralQualificationRobust6Of8AlternatingP95
}

func validOptimizeSHA(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func emptyOptimizeCalibration(invocation optimizeInvocation, reason string) optimizeCalibrationResult {
	return optimizeCalibrationResult{
		Status: optimizeCalibrationSkipped, Reason: reason,
		PairsRequested:         invocation.calibrationPairs,
		MaximumBreakEvenBuilds: invocation.maxBreakEvenBuilds,
		Interval95SavedMS:      []float64{}, GeneratedFiles: []string{},
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
}

func (run *optimizeRun) resumeCalibration() (optimizeDiscoveryResult, optimizeCalibrationResult, bool) {
	if !run.state.Resume.Accepted || run.previousState == nil ||
		run.previousState.Calibration.Status != optimizeCalibrationComplete ||
		!run.previousState.Calibration.Performed {
		return optimizeDiscoveryResult{}, optimizeCalibrationResult{}, false
	}
	discovery := run.previousState.Discovery
	calibration := run.previousState.Calibration
	if err := validateOptimizeCalibrationEvidence(run.invocation, discovery, calibration); err != nil {
		return optimizeDiscoveryResult{}, optimizeCalibrationResult{}, false
	}
	calibration.Reused = true
	return discovery, calibration, true
}

func (run *optimizeRun) calibrate(
	ctx context.Context,
	learningStarted time.Time,
	discovery optimizeDiscoveryResult,
	progress interface{ Write([]byte) (int, error) },
) optimizeCalibrationResult {
	result := emptyOptimizeCalibration(run.invocation, discovery.Reason)
	if discovery.Status != optimizeDiscoveryComplete {
		return result
	}
	if run.invocation.calibrationPairs < optimizeRequiredCalibrationPairs {
		result.Reason = optimizeCalibrationReasonPairs
		return result
	}
	options, reason := optimizeCalibrationGradleOptions(run.invocation.discovery.gradleOptions)
	if reason != "" {
		result.Status = optimizeCalibrationRetained
		result.Reason = reason
		return result
	}
	if err := ctx.Err(); err != nil {
		result.Status = optimizeCalibrationRetained
		result.Reason = "CALIBRATION_BUDGET_EXHAUSTED"
		return result
	}
	directory := filepath.Join(filepath.FromSlash(run.invocation.stateRelative), "discovery")
	evidencePath := filepath.Join(filepath.FromSlash(run.invocation.stateRelative), "calibration", optimizeCalibrationEvidenceFile)
	config, err := prepareStructuralMeasurementConfig(
		filepath.Join(directory, "manifest.json"),
		filepath.Join(directory, "graph.json"),
		filepath.Join(directory, "generated-manifest.json"),
		filepath.Join(directory, "changes.txt"),
		filepath.Join(directory, "fallback-changes.txt"),
		discovery.BaseRevision,
		optimizeExecutableRevision(run.invocation.executableSHA256),
		evidencePath,
		options,
		run.invocation.calibrationBudget,
		1,
		false,
		"",
		run.gradleBuildCacheSeed,
	)
	if err != nil {
		result.Status = optimizeCalibrationRetained
		result.Reason = optimizeCalibrationReasonUnavailable
		return result
	}
	config.pairedTargetStability = true
	config.qualificationPolicy = profilediscovery.StructuralQualificationRobust6Of8AlternatingP95
	config.parentContext = ctx
	if deadline, ok := ctx.Deadline(); ok {
		config.deadline = deadline
	}
	discoverySHA, err := optimizeGeneratedDocumentsSHA(run.invocation.repositoryRoot, discovery.GeneratedFiles)
	if err != nil {
		result.Status = optimizeCalibrationRetained
		result.Reason = "CALIBRATION_DISCOVERY_DRIFT"
		return result
	}
	if err := stopOptimizeDiscoveryGradleDaemon(ctx, run.invocation.repositoryRoot); err != nil {
		_, _ = fmt.Fprintf(progress, "buildopt: discovery Gradle daemon cleanup unavailable: %v\n", err)
		result.Status = optimizeCalibrationRetained
		result.Reason = optimizeCalibrationReasonUnavailable
		return result
	}
	_, _ = fmt.Fprintln(progress, "buildopt: stopped discovery Gradle daemon before isolated calibration")
	raw, _, err := measureStructuralProfile(config, progress)
	if err != nil {
		_, _ = fmt.Fprintf(progress, "buildopt: calibration evidence unavailable: %v\n", err)
		result.Status = optimizeCalibrationRetained
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			result.Reason = optimizeCalibrationReasonCancelled
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Reason = "CALIBRATION_BUDGET_EXHAUSTED"
		} else {
			result.Reason = optimizeCalibrationReasonUnavailable
		}
		return result
	}
	if current, digestErr := optimizeGeneratedDocumentsSHA(run.invocation.repositoryRoot, discovery.GeneratedFiles); digestErr != nil || current != discoverySHA {
		result.Status = optimizeCalibrationRetained
		result.Reason = "CALIBRATION_DISCOVERY_DRIFT"
		return result
	}
	summary, err := profilediscovery.InspectStructuralMeasurementEvidence(raw, config.analysis)
	if err != nil || summary.ProductionAuthorized || !summary.FallbackSuccessful {
		result.Status = optimizeCalibrationRetained
		result.Reason = optimizeCalibrationReasonUnavailable
		return result
	}
	absoluteEvidence := filepath.Join(run.invocation.repositoryRoot, evidencePath)
	if err := os.MkdirAll(filepath.Dir(absoluteEvidence), 0o700); err != nil || writePrivateAtomicFile(absoluteEvidence, raw) != nil {
		result.Status = optimizeCalibrationRetained
		result.Reason = "CALIBRATION_STATE_WRITE_FAILED"
		return result
	}
	evidenceDigest := sha256.Sum256(raw)
	costMS := time.Since(learningStarted).Milliseconds()
	if costMS < 1 {
		costMS = 1
	}
	breakEven := 0
	if summary.MeanSavedMS > 0 {
		breakEven = int(math.Ceil(float64(costMS) / summary.MeanSavedMS))
	}
	qualified := summary.Qualified && breakEven > 0 && breakEven <= run.invocation.maxBreakEvenBuilds
	reason = optimizeCalibrationReasonNoValue
	if summary.Qualified && !qualified {
		reason = optimizeCalibrationReasonBreakEven
	}
	if qualified {
		reason = optimizeCalibrationReasonQualified
	}
	return optimizeCalibrationResult{
		Status: optimizeCalibrationComplete, Reason: reason, Performed: true,
		PairsRequested: run.invocation.calibrationPairs, PairsMeasured: summary.Pairs,
		ControlMeanMS: summary.ControlMeanMS, CandidateMeanMS: summary.CandidateMeanMS,
		MeanSavedMS: summary.MeanSavedMS, ReductionRatio: summary.ReductionRatio,
		Interval95SavedMS: append([]float64(nil), summary.Interval95SavedMS...),
		PositivePairs:     summary.PositivePairs, ControlP95MS: summary.ControlP95MS,
		CandidateP95MS: summary.CandidateP95MS, CalibrationCostMS: costMS,
		QualificationPolicy: summary.QualificationPolicy,
		BreakEvenBuilds:     breakEven, MaximumBreakEvenBuilds: run.invocation.maxBreakEvenBuilds,
		ValueGatePassed: summary.Qualified, Qualified: qualified,
		FallbackSuccessful: summary.FallbackSuccessful,
		EvidenceSHA256:     hex.EncodeToString(evidenceDigest[:]), DiscoverySHA256: discoverySHA,
		GeneratedFiles:       []string{filepath.ToSlash(evidencePath)},
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
}

// stopOptimizeDiscoveryGradleDaemon releases the native build's resident JVM
// before calibration starts its isolated control and candidate daemon. This is
// outside the measured pairs and prevents large repositories from exhausting a
// bounded host merely because discovery and calibration have distinct homes.
func stopOptimizeDiscoveryGradleDaemon(ctx context.Context, repositoryRoot string) error {
	stopContext, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	command := measurementGradleCommand(repositoryRoot, []string{"--stop"})
	cmd := exec.CommandContext(stopContext, command[0], command[1:]...)
	cmd.Dir = repositoryRoot
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop discovery Gradle daemon: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func optimizeCalibrationGradleOptions(options []string) ([]string, string) {
	normalized := make([]string, 0, len(options)+3)
	for index := 0; index < len(options); index++ {
		option := options[index]
		if option == "--max-workers" || option == "--console" {
			if index+1 >= len(options) {
				return nil, "CALIBRATION_OPTIONS_UNSUPPORTED"
			}
			option += "=" + options[index+1]
			index++
		}
		if !validImpactGradleOption(option) {
			return nil, "CALIBRATION_OPTIONS_UNSUPPORTED"
		}
		normalized = append(normalized, option)
	}
	if !optimizeHasGradleOption(normalized, "--build-cache", "--no-build-cache") {
		normalized = append(normalized, "--build-cache")
	}
	if !optimizeHasGradleOptionPrefix(normalized, "--console=") {
		normalized = append(normalized, "--console=plain")
	}
	if !optimizeHasGradleOption(normalized, "--scan", "--no-scan") {
		normalized = append(normalized, "--no-scan")
	}
	if len(normalized) > 32 || !uniqueMeasurementStrings(normalized) {
		return nil, "CALIBRATION_OPTIONS_UNSUPPORTED"
	}
	return normalized, ""
}

func optimizeHasGradleOption(options []string, names ...string) bool {
	for _, option := range options {
		for _, name := range names {
			if option == name {
				return true
			}
		}
	}
	return false
}

func optimizeHasGradleOptionPrefix(options []string, prefix string) bool {
	for _, option := range options {
		if strings.HasPrefix(option, prefix) {
			return true
		}
	}
	return false
}

func optimizeExecutableRevision(executableSHA string) string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		revision := ""
		modified := false
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = strings.ToLower(strings.TrimSpace(setting.Value))
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}
		if !modified && validMeasurementRevision(revision) {
			return revision
		}
	}
	return executableSHA[:40]
}

func optimizeGeneratedDocumentsSHA(repositoryRoot string, paths []string) (string, error) {
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	digest := sha256.New()
	for _, relative := range ordered {
		if !validOptimizeGeneratedPath(relative) {
			return "", errors.New("generated discovery path is invalid")
		}
		raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		optimizeWriteDigestValue(digest, relative)
		optimizeWriteDigestValue(digest, string(raw))
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func validateOptimizeCalibrationEvidence(invocation optimizeInvocation, discovery optimizeDiscoveryResult, calibration optimizeCalibrationResult) error {
	if !calibration.Performed || len(calibration.GeneratedFiles) != 1 ||
		!validOptimizeGeneratedPath(calibration.GeneratedFiles[0]) {
		return errors.New("calibration checkpoint is incomplete")
	}
	discoverySHA, err := optimizeGeneratedDocumentsSHA(invocation.repositoryRoot, discovery.GeneratedFiles)
	if err != nil || discoverySHA != calibration.DiscoverySHA256 {
		return errors.New("calibration discovery binding drift")
	}
	evidencePath := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(calibration.GeneratedFiles[0]))
	raw, err := os.ReadFile(evidencePath)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != calibration.EvidenceSHA256 {
		return errors.New("calibration evidence binding drift")
	}
	analysis, err := profilediscovery.AnalyzeOpportunity(profilediscovery.AnalysisOptions{
		RepositoryRoot: invocation.repositoryRoot,
		ManifestPath:   filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery", "manifest.json"),
		GraphPath:      filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery", "graph.json"),
		GeneratedPath:  filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery", "generated-manifest.json"),
	})
	if err != nil {
		return err
	}
	summary, err := profilediscovery.InspectStructuralMeasurementEvidence(raw, analysis)
	if err != nil || !sameOptimizeCalibrationSummary(calibration, summary) {
		return errors.New("calibration evidence no longer matches its checkpoint")
	}
	return nil
}

func sameOptimizeCalibrationSummary(calibration optimizeCalibrationResult, summary profilediscovery.StructuralMeasurementSummary) bool {
	breakEven := 0
	if summary.MeanSavedMS > 0 {
		breakEven = int(math.Ceil(float64(calibration.CalibrationCostMS) / summary.MeanSavedMS))
	}
	qualified := summary.Qualified && breakEven > 0 && breakEven <= calibration.MaximumBreakEvenBuilds
	return calibration.PairsMeasured == summary.Pairs &&
		calibration.ControlMeanMS == summary.ControlMeanMS &&
		calibration.CandidateMeanMS == summary.CandidateMeanMS &&
		calibration.MeanSavedMS == summary.MeanSavedMS &&
		calibration.ReductionRatio == summary.ReductionRatio &&
		calibration.PositivePairs == summary.PositivePairs &&
		calibration.ControlP95MS == summary.ControlP95MS &&
		calibration.CandidateP95MS == summary.CandidateP95MS &&
		calibration.QualificationPolicy == summary.QualificationPolicy &&
		calibration.BreakEvenBuilds == breakEven &&
		calibration.ValueGatePassed == summary.Qualified &&
		calibration.Qualified == qualified &&
		calibration.FallbackSuccessful == summary.FallbackSuccessful &&
		sameFloatSlice(calibration.Interval95SavedMS, summary.Interval95SavedMS)
}

func sameFloatSlice(left, right []float64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validOptimizeGeneratedPath(path string) bool {
	native := filepath.FromSlash(path)
	return path != "" && !filepath.IsAbs(native) && filepath.Clean(native) == native &&
		strings.HasPrefix(path, ".buildopt/")
}
