package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	optimizeIncrementalSchema            = "buildopt.poc/incremental-learning/v2"
	optimizeIncrementalCollecting        = "COLLECTING"
	optimizeIncrementalComplete          = "COMPLETE"
	optimizeIncrementalRetained          = "NATIVE_RETAINED"
	optimizeIncrementalReasonPending     = "ORDINARY_OBSERVATIONS_PENDING"
	optimizeIncrementalReasonComplete    = "ORDINARY_OBSERVATIONS_COMPLETE"
	optimizeIncrementalReasonOutputDrift = "REQUIRED_OUTPUT_DRIFT"
	optimizeIncrementalReasonCandidate   = "CANDIDATE_EXECUTION_FAILED"
	optimizeIncrementalReasonCancelled   = "CALIBRATION_CANCELLED"
	optimizeIncrementalReasonPreparation = "CANDIDATE_PREPARATION_FAILED"
	optimizeIncrementalReasonState       = "INCREMENTAL_STATE_INVALID"
	optimizeIncrementalArmDiscovery      = "DISCOVERY_CONTROL"
	optimizeIncrementalArmControl        = "CONTROL"
	optimizeIncrementalArmCandidate      = "CANDIDATE"
	optimizeIncrementalArmComplete       = "COMPLETE"
	optimizeIncrementalCheckpointFile    = "incremental.json"
)

type optimizeIncrementalObservation struct {
	Sequence                   int                          `json:"sequence"`
	Pair                       int                          `json:"pair"`
	Arm                        string                       `json:"arm"`
	Order                      string                       `json:"order"`
	DurationMS                 int64                        `json:"durationMs"`
	RequiredOutputSHA256       string                       `json:"requiredOutputSha256"`
	RequiredOutputCount        int                          `json:"requiredOutputCount"`
	ExitCode                   int                          `json:"exitCode"`
	IncrementalOverheadMS      int64                        `json:"incrementalOverheadMs"`
	Economics                  optimizeIncrementalEconomics `json:"economics"`
	CapturedAt                 string                       `json:"capturedAt"`
	ProductAttributableFailure bool                         `json:"productAttributableFailure"`
}

type optimizeIncrementalEconomics struct {
	GradleMS             int64 `json:"gradleMs"`
	PreExecutionMS       int64 `json:"preExecutionMs"`
	PostExecutionMS      int64 `json:"postExecutionMs"`
	MaterializationMS    int64 `json:"materializationMs"`
	OutputVerificationMS int64 `json:"outputVerificationMs"`
	DiscoveryMS          int64 `json:"discoveryMs"`
	OtherWrapperMS       int64 `json:"otherWrapperMs"`
}

type optimizeIncrementalLearning struct {
	Status               string                           `json:"status"`
	Reason               string                           `json:"reason"`
	Performed            bool                             `json:"performed"`
	Reused               bool                             `json:"reused"`
	TargetPairs          int                              `json:"targetPairs"`
	PairsCompleted       int                              `json:"pairsCompleted"`
	NextArm              string                           `json:"nextArm"`
	ExpectedOutputSHA256 string                           `json:"expectedOutputSha256"`
	ExpectedOutputCount  int                              `json:"expectedOutputCount"`
	DiscoverySHA256      string                           `json:"discoverySha256"`
	IncrementalCostMS    int64                            `json:"incrementalCostMs"`
	Baseline             *optimizeIncrementalObservation  `json:"baseline,omitempty"`
	Observations         []optimizeIncrementalObservation `json:"observations"`
	CheckpointSHA256     string                           `json:"checkpointSha256"`
	GeneratedFiles       []string                         `json:"generatedFiles"`
	FallbackSuccessful   bool                             `json:"fallbackSuccessful"`
	ProductionAuthorized bool                             `json:"productionAuthorized"`
	TestOptimization     string                           `json:"testOptimization"`
}

type optimizeIncrementalDocument struct {
	SchemaVersion        string                           `json:"schemaVersion"`
	BindingSHA256        string                           `json:"bindingSha256"`
	Status               string                           `json:"status"`
	Reason               string                           `json:"reason"`
	TargetPairs          int                              `json:"targetPairs"`
	PairsCompleted       int                              `json:"pairsCompleted"`
	NextArm              string                           `json:"nextArm"`
	ExpectedOutputSHA256 string                           `json:"expectedOutputSha256"`
	ExpectedOutputCount  int                              `json:"expectedOutputCount"`
	DiscoverySHA256      string                           `json:"discoverySha256"`
	IncrementalCostMS    int64                            `json:"incrementalCostMs"`
	Baseline             *optimizeIncrementalObservation  `json:"baseline,omitempty"`
	Observations         []optimizeIncrementalObservation `json:"observations"`
	FallbackSuccessful   bool                             `json:"fallbackSuccessful"`
	ProductionAuthorized bool                             `json:"productionAuthorized"`
	TestOptimization     string                           `json:"testOptimization"`
}

type optimizeOutputObservation struct {
	directory       string
	initPath        string
	snapshotPath    string
	entrypointsJSON string
	impact          buildimpact.InlineObservation
}

func emptyOptimizeIncrementalLearning() optimizeIncrementalLearning {
	return optimizeIncrementalLearning{
		Observations:         []optimizeIncrementalObservation{},
		GeneratedFiles:       []string{},
		ProductionAuthorized: false,
		TestOptimization:     "OUT_OF_SCOPE",
	}
}

func (run *optimizeRun) executionMode(selection optimizeSelectionResult) string {
	if run.incrementalFallback.started {
		return "FULL_GRAPH_RECOVERY"
	}
	if run.incrementalCandidate {
		return "INCREMENTAL_CANDIDATE"
	}
	return optimizeExecutionMode(selection)
}

func (run *optimizeRun) executionDescription(selection optimizeSelectionResult) string {
	if run.incrementalFallback.started {
		return "optimized native Gradle full-graph recovery"
	}
	if run.incrementalCandidate {
		return "incremental structural candidate observation"
	}
	return optimizeSelectionDescription(selection)
}

func (run *optimizeRun) prepareOutputObservation() error {
	if !run.invocation.discovery.Ready ||
		(run.previousState != nil && run.state.Resume.Accepted &&
			run.previousState.Discovery.Status == optimizeDiscoveryComplete) {
		return nil
	}
	directory := filepath.Join(run.invocation.stateDirectory, "observation")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create ordinary output observation: %w", err)
	}
	initPath := filepath.Join(directory, "output-contract.init.gradle")
	snapshotPath := filepath.Join(directory, "snapshot.json")
	if err := os.WriteFile(initPath, profileOutputContractInit, 0o600); err != nil {
		return fmt.Errorf("write ordinary output observation init script: %w", err)
	}
	_ = os.Remove(snapshotPath)
	entrypoints, err := json.Marshal(run.invocation.discovery.Entrypoints)
	if err != nil {
		return fmt.Errorf("encode ordinary output observation entrypoints: %w", err)
	}
	impact, err := buildimpact.PrepareInlineObservation(directory, run.invocation.discovery.Entrypoints)
	if err != nil {
		return err
	}
	run.outputObservation = &optimizeOutputObservation{
		directory: directory, initPath: initPath, snapshotPath: snapshotPath,
		entrypointsJSON: string(entrypoints), impact: impact,
	}
	return nil
}

func (run *optimizeRun) augmentGradleOutputObservation(invocation *gradleInvocation) {
	if run == nil || run.outputObservation == nil || invocation == nil {
		return
	}
	childArgs := make([]string, 0, len(invocation.childArgs)+6)
	childArgs = append(childArgs, invocation.childArgs[0],
		"--init-script", run.outputObservation.initPath,
		"--init-script", run.outputObservation.impact.InitPath)
	childArgs = append(childArgs, invocation.childArgs[1:]...)
	invocation.childArgs = append(childArgs, "buildoptOutputContract", "buildoptImpactDiscovery")
	if invocation.environment == nil {
		invocation.environment = make(map[string]string)
	}
	invocation.environment["BUILDOPT_OUTPUT_CONTRACT_SNAPSHOT"] = run.outputObservation.snapshotPath
	invocation.environment["BUILDOPT_OUTPUT_CONTRACT_ENTRYPOINTS"] = run.outputObservation.entrypointsJSON
	invocation.environment["BUILDOPT_IMPACT_DISCOVERY_OUTPUT"] = run.outputObservation.impact.OutputPath
	invocation.environment["BUILDOPT_IMPACT_ENTRYPOINTS"] = run.outputObservation.impact.EntrypointsJSON
	invocation.environment["BUILDOPT_IMPACT_DISCOVERY_INLINE"] = "1"
}

func (run *optimizeRun) observedImpactSnapshot() (*buildimpact.DiscoverySnapshot, error) {
	if run.outputObservation == nil {
		return nil, errors.New("inline impact observation was not prepared")
	}
	snapshot, err := buildimpact.ReadInlineObservation(
		run.outputObservation.impact,
		run.invocation.discovery.Entrypoints,
	)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (run *optimizeRun) observedOutputSnapshot() (*outputContractSnapshot, error) {
	if run.outputObservation == nil {
		return nil, errors.New("ordinary output observation was not prepared")
	}
	raw, err := os.ReadFile(run.outputObservation.snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("read ordinary output observation: %w", err)
	}
	snapshot, err := parseOutputContractSnapshot(raw, run.invocation.discovery.Entrypoints)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (run *optimizeRun) cleanupOutputObservation() {
	if run.outputObservation == nil {
		return
	}
	_ = os.Remove(run.outputObservation.snapshotPath)
	_ = os.Remove(run.outputObservation.initPath)
	_ = os.Remove(run.outputObservation.impact.OutputPath)
	_ = os.Remove(run.outputObservation.impact.InitPath)
	_ = os.Remove(run.outputObservation.directory)
}

func expectedOptimizeIncrementalArm(observations int) (string, int, string) {
	pair := observations/2 + 1
	controlFirst := pair%2 == 1
	arm := optimizeIncrementalArmCandidate
	order := "CANDIDATE_FIRST"
	if controlFirst {
		order = "CONTROL_FIRST"
	}
	if (observations%2 == 0 && controlFirst) || (observations%2 == 1 && !controlFirst) {
		arm = optimizeIncrementalArmControl
	}
	return arm, pair, order
}

func (run *optimizeRun) prepareIncrementalLearningArm() *impactInvocation {
	run.incrementalArm = optimizeIncrementalArmDiscovery
	if !run.state.Resume.Accepted || run.previousState == nil ||
		run.previousState.Discovery.Status != optimizeDiscoveryComplete ||
		run.previousState.Calibration.Status == optimizeCalibrationComplete {
		return nil
	}
	discovery := run.previousState.Discovery
	learning := run.previousState.IncrementalLearning
	if learning.Status == optimizeIncrementalRetained || learning.Status == optimizeIncrementalComplete {
		return nil
	}
	if learning.Baseline == nil {
		run.incrementalArm = optimizeIncrementalArmDiscovery
		return nil
	}
	arm, _, _ := expectedOptimizeIncrementalArm(len(learning.Observations))
	run.incrementalArm = arm
	if arm != optimizeIncrementalArmCandidate {
		return nil
	}

	directory := filepath.ToSlash(filepath.Join(run.invocation.stateRelative, "discovery"))
	arguments := []string{
		"--repository-id", discovery.RepositoryID,
		"--pipeline-class", optimizePipelineClass(discovery.Entrypoints, discovery.ChangeSHA256),
		"--changes-file", filepath.ToSlash(filepath.Join(directory, "changes.txt")),
		"--manifest", filepath.ToSlash(filepath.Join(directory, "manifest.json")),
		"--graph", filepath.ToSlash(filepath.Join(directory, "graph.json")),
		"--generated-manifest", filepath.ToSlash(filepath.Join(directory, "generated-manifest.json")),
	}
	options, reason := optimizeCalibrationGradleOptions(run.invocation.discovery.gradleOptions)
	if reason != "" {
		run.incrementalFailure = optimizeIncrementalReasonPreparation
		return nil
	}
	for _, option := range options {
		arguments = append(arguments, "--gradle-option="+option)
	}
	impact, err := prepareImpactInvocation(arguments, false)
	if err != nil || !impact.plan.CandidateSelected ||
		!equalOptimizeStrings(impact.plan.Entrypoints, discovery.CandidateEntrypoints) {
		run.incrementalFailure = optimizeIncrementalReasonPreparation
		return nil
	}
	run.incrementalCandidate = true
	return &impact
}

func (run *optimizeRun) captureIncrementalOutput(exitCode int) bool {
	if run == nil || run.incrementalArm == "" || run.incrementalArm == optimizeIncrementalArmDiscovery {
		return false
	}
	discovery := run.previousState.Discovery
	if exitCode != 0 {
		if run.incrementalCandidate {
			run.incrementalFailure = optimizeIncrementalReasonCandidate
			return true
		}
		return false
	}
	digest, count, err := run.hashIncrementalOutputs(discovery.RequiredOutputs)
	if err != nil {
		run.incrementalFailure = optimizeIncrementalReasonOutputDrift
		return run.incrementalCandidate
	}
	run.incrementalOutputSHA = digest
	run.incrementalOutputCount = count
	learning := run.previousState.IncrementalLearning
	if learning.ExpectedOutputSHA256 != "" &&
		(digest != learning.ExpectedOutputSHA256 || count != learning.ExpectedOutputCount) {
		run.incrementalFailure = optimizeIncrementalReasonOutputDrift
		return run.incrementalCandidate
	}
	return false
}

func (run *optimizeRun) captureIncrementalCancellation() bool {
	if run == nil || run.incrementalArm == "" || !run.childExecution.cancelled {
		return false
	}
	run.incrementalFailure = optimizeIncrementalReasonCancelled
	return true
}

func (run *optimizeRun) recordIncrementalFallback(execution childExecution, exitCode int) {
	run.incrementalFallback = execution
	if !execution.started || exitCode != 0 {
		run.incrementalFailure = optimizeIncrementalReasonCandidate
	}
}

func (run *optimizeRun) collectIncrementalLearning(
	discovery optimizeDiscoveryResult,
	exitCode int,
) (optimizeIncrementalLearning, optimizeCalibrationResult) {
	if discovery.Status != optimizeDiscoveryComplete || !run.childExecution.started {
		return emptyOptimizeIncrementalLearning(), emptyOptimizeCalibration(run.invocation, discovery.Reason)
	}
	if run.invocation.calibrationPairs < optimizeRequiredCalibrationPairs {
		return emptyOptimizeIncrementalLearning(), emptyOptimizeCalibration(run.invocation, optimizeCalibrationReasonPairs)
	}
	calibration := emptyOptimizeCalibration(run.invocation, optimizeIncrementalReasonPending)
	learning := emptyOptimizeIncrementalLearning()
	if run.state.Resume.Accepted && run.previousState != nil &&
		run.previousState.Discovery.Status == optimizeDiscoveryComplete {
		learning = cloneOptimizeIncrementalLearning(run.previousState.IncrementalLearning)
	}
	if learning.Status == "" {
		learning.Status = optimizeIncrementalCollecting
		learning.Reason = optimizeIncrementalReasonPending
		learning.Performed = true
		learning.TargetPairs = optimizeRequiredCalibrationPairs
		learning.FallbackSuccessful = true
		learning.TestOptimization = "OUT_OF_SCOPE"
	}
	if run.incrementalFailure == optimizeIncrementalReasonCancelled {
		return retainedOptimizeIncrementalLearning(run.invocation, learning, run.incrementalFailure), retainedIncrementalCalibration(run.invocation, run.incrementalFailure)
	}
	if exitCode != 0 {
		return emptyOptimizeIncrementalLearning(), emptyOptimizeCalibration(run.invocation, discovery.Reason)
	}
	discoverySHA, err := optimizeGeneratedDocumentsSHA(run.invocation.repositoryRoot, discovery.GeneratedFiles)
	if err != nil {
		return retainedOptimizeIncrementalLearning(run.invocation, learning, optimizeIncrementalReasonState), retainedIncrementalCalibration(run.invocation, optimizeIncrementalReasonState)
	}
	if learning.DiscoverySHA256 == "" {
		learning.DiscoverySHA256 = discoverySHA
	} else if learning.DiscoverySHA256 != discoverySHA {
		return retainedOptimizeIncrementalLearning(run.invocation, learning, optimizeIncrementalReasonState), retainedIncrementalCalibration(run.invocation, optimizeIncrementalReasonState)
	}
	if run.incrementalFailure != "" {
		fallbackOK := run.incrementalFallback.started && run.incrementalFallback.err == nil
		if !run.incrementalCandidate {
			fallbackOK = run.childExecution.started && run.childExecution.err == nil
		}
		learning.FallbackSuccessful = learning.FallbackSuccessful && fallbackOK
		return retainedOptimizeIncrementalLearning(run.invocation, learning, run.incrementalFailure), retainedIncrementalCalibration(run.invocation, run.incrementalFailure)
	}

	digest := run.incrementalOutputSHA
	count := run.incrementalOutputCount
	if digest == "" {
		digest, count, err = run.hashIncrementalOutputs(discovery.RequiredOutputs)
		if err != nil {
			return retainedOptimizeIncrementalLearning(run.invocation, learning, optimizeIncrementalReasonOutputDrift), retainedIncrementalCalibration(run.invocation, optimizeIncrementalReasonOutputDrift)
		}
	}
	overhead := optimizeIncrementalOverheadMS(run)
	economics := optimizeIncrementalEconomicsForRun(run, overhead)
	observation := optimizeIncrementalObservation{
		DurationMS:            optimizeIncrementalWallTimeMS(run, overhead),
		RequiredOutputSHA256:  digest,
		RequiredOutputCount:   count,
		ExitCode:              exitCode,
		IncrementalOverheadMS: overhead,
		Economics:             economics,
		CapturedAt:            time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
	}
	if learning.Baseline == nil {
		run.incrementalObserved = true
		observation.Sequence = 0
		observation.Pair = 0
		observation.Arm = optimizeIncrementalArmDiscovery
		observation.Order = "BASELINE"
		learning.Baseline = &observation
		learning.ExpectedOutputSHA256 = digest
		learning.ExpectedOutputCount = count
		learning.NextArm = optimizeIncrementalArmControl
		// Ordinary paired builds are useful customer work and their recurring
		// wrapper cost is already included in DurationMS. Only the initial
		// discovery/capture overhead is a one-time cost that must be repaid.
		learning.IncrementalCostMS += overhead
	} else {
		arm, pair, order := expectedOptimizeIncrementalArm(len(learning.Observations))
		if run.incrementalArm != arm || digest != learning.ExpectedOutputSHA256 || count != learning.ExpectedOutputCount {
			return retainedOptimizeIncrementalLearning(run.invocation, learning, optimizeIncrementalReasonOutputDrift), retainedIncrementalCalibration(run.invocation, optimizeIncrementalReasonOutputDrift)
		}
		observation.Sequence = len(learning.Observations) + 1
		observation.Pair = pair
		observation.Arm = arm
		observation.Order = order
		learning.Observations = append(learning.Observations, observation)
		run.incrementalObserved = true
		learning.PairsCompleted = len(learning.Observations) / 2
		if len(learning.Observations) < optimizeRequiredCalibrationPairs*2 {
			learning.NextArm, _, _ = expectedOptimizeIncrementalArm(len(learning.Observations))
		} else {
			learning.NextArm = optimizeIncrementalArmComplete
		}
	}
	if err := writeOptimizeIncrementalCheckpoint(run.invocation, &learning); err != nil {
		return retainedOptimizeIncrementalLearning(run.invocation, learning, optimizeIncrementalReasonState), retainedIncrementalCalibration(run.invocation, optimizeIncrementalReasonState)
	}
	if len(learning.Observations) != optimizeRequiredCalibrationPairs*2 {
		return learning, calibration
	}
	completed, completedCalibration := completeOptimizeIncrementalLearning(run.invocation, discovery, learning)
	return completed, completedCalibration
}

func optimizeIncrementalWallTimeMS(run *optimizeRun, overhead int64) int64 {
	return maximumOptimizeDurationMS(run.childExecution.completedAt.Sub(run.childExecution.startedAt)) + overhead
}

func (run *optimizeRun) hashIncrementalOutputs(patterns []string) (string, int, error) {
	started := time.Now()
	digest, count, err := hashMeasurementOutputs(run.invocation.repositoryRoot, patterns)
	run.outputVerificationTime += time.Since(started)
	return digest, count, err
}

func optimizeIncrementalEconomicsForRun(run *optimizeRun, overhead int64) optimizeIncrementalEconomics {
	pre := run.childExecution.startedAt.Sub(run.startedAt).Milliseconds()
	if pre < 0 {
		pre = 0
	}
	if pre > overhead {
		pre = overhead
	}
	post := overhead - pre
	materialization := run.materializationTime.Milliseconds()
	verification := run.outputVerificationTime.Milliseconds()
	discovery := run.discoveryTime.Milliseconds()
	other := overhead - materialization - verification - discovery
	if other < 0 {
		other = 0
	}
	return optimizeIncrementalEconomics{
		GradleMS:       maximumOptimizeDurationMS(run.childExecution.completedAt.Sub(run.childExecution.startedAt)),
		PreExecutionMS: pre, PostExecutionMS: post, MaterializationMS: materialization,
		OutputVerificationMS: verification, DiscoveryMS: discovery, OtherWrapperMS: other,
	}
}

func completeOptimizeIncrementalLearning(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
	learning optimizeIncrementalLearning,
) (optimizeIncrementalLearning, optimizeCalibrationResult) {
	calibration := retainedIncrementalCalibration(invocation, optimizeCalibrationReasonUnavailable)
	analysis, err := profilediscovery.AnalyzeOpportunity(profilediscovery.AnalysisOptions{
		RepositoryRoot: invocation.repositoryRoot,
		ManifestPath:   filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery", "manifest.json"),
		GraphPath:      filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery", "graph.json"),
		GeneratedPath:  filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery", "generated-manifest.json"),
	})
	if err != nil {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonState), calibration
	}
	observations := make([]profilediscovery.StructuralMeasurementObservation, 0, optimizeRequiredCalibrationPairs)
	for index := 0; index < len(learning.Observations); index += 2 {
		first := learning.Observations[index]
		second := learning.Observations[index+1]
		control, candidate := first, second
		if first.Arm == optimizeIncrementalArmCandidate {
			control, candidate = second, first
		}
		if control.Arm != optimizeIncrementalArmControl || candidate.Arm != optimizeIncrementalArmCandidate ||
			control.Pair != candidate.Pair || control.Order != candidate.Order ||
			control.RequiredOutputSHA256 != candidate.RequiredOutputSHA256 ||
			control.RequiredOutputCount != candidate.RequiredOutputCount {
			return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonOutputDrift), calibration
		}
		observations = append(observations, profilediscovery.StructuralMeasurementObservation{
			Pair: control.Pair, Order: control.Order,
			ControlDurationMS: control.DurationMS, CandidateDurationMS: candidate.DurationMS,
			RequiredOutputSHA256: control.RequiredOutputSHA256,
			RequiredOutputCount:  control.RequiredOutputCount,
		})
	}
	changesRaw, err := os.ReadFile(filepath.Join(invocation.repositoryRoot, filepath.FromSlash(invocation.stateRelative), "discovery", "changes.txt"))
	if err != nil {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonState), calibration
	}
	changesSHA := sha256.Sum256(changesRaw)
	options, reason := optimizeCalibrationGradleOptions(invocation.discovery.gradleOptions)
	if reason != "" {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonPreparation), calibration
	}
	raw, _, err := profilediscovery.RenderStructuralMeasurementEvidence(profilediscovery.StructuralMeasurementOptions{
		CapturedAt: time.Now(), Analysis: analysis,
		RepositoryRevision:   discovery.TargetRevision,
		BuildOptRevision:     optimizeExecutableRevision(invocation.executableSHA256),
		ExecutableSHA256:     invocation.executableSHA256,
		SourceEvidenceSHA256: hex.EncodeToString(changesSHA[:]),
		GradleOptions:        options, Observations: observations,
		FallbackReason:     "ORDINARY_FULL_GRAPH_CONTROLS_SUCCEEDED",
		FallbackSuccessful: learning.FallbackSuccessful,
	})
	if err != nil {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonState), calibration
	}
	evidencePath := filepath.Join(filepath.FromSlash(invocation.stateRelative), "calibration", optimizeCalibrationEvidenceFile)
	absoluteEvidence := filepath.Join(invocation.repositoryRoot, evidencePath)
	if err := os.MkdirAll(filepath.Dir(absoluteEvidence), 0o700); err != nil || writePrivateAtomicFile(absoluteEvidence, raw) != nil {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonState), calibration
	}
	summary, err := profilediscovery.InspectStructuralMeasurementEvidence(raw, analysis)
	if err != nil || summary.ProductionAuthorized || !summary.FallbackSuccessful {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonState), calibration
	}
	evidenceDigest := sha256.Sum256(raw)
	costMS := learning.IncrementalCostMS
	if costMS < 1 {
		costMS = 1
	}
	breakEven := 0
	if summary.MeanSavedMS > 0 {
		breakEven = int(math.Ceil(float64(costMS) / summary.MeanSavedMS))
	}
	qualified := summary.Qualified && breakEven > 0 && breakEven <= invocation.maxBreakEvenBuilds
	calibrationReason := optimizeCalibrationReasonNoValue
	if summary.Qualified && !qualified {
		calibrationReason = optimizeCalibrationReasonBreakEven
	}
	if qualified {
		calibrationReason = optimizeCalibrationReasonQualified
	}
	calibration = optimizeCalibrationResult{
		Status: optimizeCalibrationComplete, Reason: calibrationReason, Performed: true,
		PairsRequested: invocation.calibrationPairs, PairsMeasured: summary.Pairs,
		ControlMeanMS: summary.ControlMeanMS, CandidateMeanMS: summary.CandidateMeanMS,
		MeanSavedMS: summary.MeanSavedMS, ReductionRatio: summary.ReductionRatio,
		Interval95SavedMS: append([]float64(nil), summary.Interval95SavedMS...),
		PositivePairs:     summary.PositivePairs, ControlP95MS: summary.ControlP95MS,
		CandidateP95MS: summary.CandidateP95MS, CalibrationCostMS: costMS,
		BreakEvenBuilds: breakEven, MaximumBreakEvenBuilds: invocation.maxBreakEvenBuilds,
		ValueGatePassed: summary.Qualified, Qualified: qualified,
		FallbackSuccessful: summary.FallbackSuccessful,
		EvidenceSHA256:     hex.EncodeToString(evidenceDigest[:]), DiscoverySHA256: learning.DiscoverySHA256,
		GeneratedFiles:       []string{filepath.ToSlash(evidencePath)},
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	learning.Status = optimizeIncrementalComplete
	learning.Reason = optimizeIncrementalReasonComplete
	learning.NextArm = optimizeIncrementalArmComplete
	if err := writeOptimizeIncrementalCheckpoint(invocation, &learning); err != nil {
		return retainedOptimizeIncrementalLearning(invocation, learning, optimizeIncrementalReasonState), retainedIncrementalCalibration(invocation, optimizeIncrementalReasonState)
	}
	return learning, calibration
}

func retainedIncrementalCalibration(invocation optimizeInvocation, reason string) optimizeCalibrationResult {
	result := emptyOptimizeCalibration(invocation, reason)
	result.Status = optimizeCalibrationRetained
	return result
}

func retainedOptimizeIncrementalLearning(invocation optimizeInvocation, learning optimizeIncrementalLearning, reason string) optimizeIncrementalLearning {
	learning.Status = optimizeIncrementalRetained
	learning.Reason = reason
	learning.NextArm = optimizeIncrementalArmComplete
	if learning.Baseline != nil {
		_ = writeOptimizeIncrementalCheckpoint(invocation, &learning)
	}
	return learning
}

func optimizeIncrementalOverheadMS(run *optimizeRun) int64 {
	if run.childExecution.startedAt.IsZero() || run.childExecution.completedAt.IsZero() {
		return 1
	}
	pre := run.childExecution.startedAt.Sub(run.startedAt).Milliseconds()
	post := time.Since(run.childExecution.completedAt).Milliseconds()
	if pre < 0 {
		pre = 0
	}
	if post < 0 {
		post = 0
	}
	if pre+post < 1 {
		return 1
	}
	return pre + post
}

func maximumOptimizeDurationMS(duration time.Duration) int64 {
	value := duration.Milliseconds()
	if value < 1 {
		return 1
	}
	return value
}

func cloneOptimizeIncrementalLearning(source optimizeIncrementalLearning) optimizeIncrementalLearning {
	clone := source
	if source.Baseline != nil {
		baseline := *source.Baseline
		clone.Baseline = &baseline
	}
	clone.Observations = append([]optimizeIncrementalObservation(nil), source.Observations...)
	clone.GeneratedFiles = append([]string(nil), source.GeneratedFiles...)
	clone.Reused = true
	return clone
}

func writeOptimizeIncrementalCheckpoint(invocation optimizeInvocation, learning *optimizeIncrementalLearning) error {
	path := filepath.Join(filepath.FromSlash(invocation.stateRelative), "calibration", optimizeIncrementalCheckpointFile)
	document := optimizeIncrementalDocument{
		SchemaVersion: optimizeIncrementalSchema, BindingSHA256: invocation.bindingSHA256,
		Status: learning.Status, Reason: learning.Reason, TargetPairs: learning.TargetPairs,
		PairsCompleted: learning.PairsCompleted, NextArm: learning.NextArm,
		ExpectedOutputSHA256: learning.ExpectedOutputSHA256, ExpectedOutputCount: learning.ExpectedOutputCount,
		DiscoverySHA256: learning.DiscoverySHA256, IncrementalCostMS: learning.IncrementalCostMS,
		Baseline:             learning.Baseline,
		Observations:         append([]optimizeIncrementalObservation(nil), learning.Observations...),
		FallbackSuccessful:   learning.FallbackSuccessful,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	absolute := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return err
	}
	if err := writePrivateAtomicFile(absolute, raw); err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	learning.CheckpointSHA256 = hex.EncodeToString(digest[:])
	learning.GeneratedFiles = []string{filepath.ToSlash(path)}
	return nil
}

func validateOptimizeIncrementalEvidence(invocation optimizeInvocation, learning optimizeIncrementalLearning) error {
	if learning.Status == "" {
		return nil
	}
	if len(learning.GeneratedFiles) != 1 || !validOptimizeGeneratedPath(learning.GeneratedFiles[0]) ||
		!validOptimizeSHA(learning.CheckpointSHA256) {
		return errors.New("incremental checkpoint binding is incomplete")
	}
	raw, err := os.ReadFile(filepath.Join(invocation.repositoryRoot, filepath.FromSlash(learning.GeneratedFiles[0])))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != learning.CheckpointSHA256 {
		return errors.New("incremental checkpoint digest drift")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document optimizeIncrementalDocument
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("incremental checkpoint has trailing content")
	}
	expected := optimizeIncrementalDocument{
		SchemaVersion: optimizeIncrementalSchema, BindingSHA256: invocation.bindingSHA256,
		Status: learning.Status, Reason: learning.Reason, TargetPairs: learning.TargetPairs,
		PairsCompleted: learning.PairsCompleted, NextArm: learning.NextArm,
		ExpectedOutputSHA256: learning.ExpectedOutputSHA256, ExpectedOutputCount: learning.ExpectedOutputCount,
		DiscoverySHA256: learning.DiscoverySHA256, IncrementalCostMS: learning.IncrementalCostMS,
		Baseline:             learning.Baseline,
		Observations:         append([]optimizeIncrementalObservation(nil), learning.Observations...),
		FallbackSuccessful:   learning.FallbackSuccessful,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	if !reflect.DeepEqual(document, expected) {
		return errors.New("incremental checkpoint no longer matches state")
	}
	return nil
}

func validOptimizeIncrementalCheckpoint(state optimizeState) bool {
	learning := state.IncrementalLearning
	if learning.Status == "" {
		return reflect.DeepEqual(learning, optimizeIncrementalLearning{}) ||
			reflect.DeepEqual(learning, emptyOptimizeIncrementalLearning())
	}
	if learning.ProductionAuthorized || learning.TestOptimization != "OUT_OF_SCOPE" ||
		!learning.Performed || learning.TargetPairs != optimizeRequiredCalibrationPairs ||
		learning.ExpectedOutputCount < 1 || !validOptimizeSHA(learning.ExpectedOutputSHA256) ||
		!validOptimizeSHA(learning.DiscoverySHA256) || !validOptimizeSHA(learning.CheckpointSHA256) ||
		len(learning.GeneratedFiles) != 1 || !validOptimizeGeneratedPath(learning.GeneratedFiles[0]) ||
		learning.Baseline == nil || !validOptimizeIncrementalObservation(*learning.Baseline, true) ||
		learning.PairsCompleted != len(learning.Observations)/2 ||
		len(learning.Observations) > optimizeRequiredCalibrationPairs*2 {
		return false
	}
	for index, observation := range learning.Observations {
		arm, pair, order := expectedOptimizeIncrementalArm(index)
		if observation.Sequence != index+1 || observation.Arm != arm || observation.Pair != pair ||
			observation.Order != order || !validOptimizeIncrementalObservation(observation, false) ||
			observation.RequiredOutputSHA256 != learning.ExpectedOutputSHA256 ||
			observation.RequiredOutputCount != learning.ExpectedOutputCount {
			return false
		}
	}
	switch learning.Status {
	case optimizeIncrementalCollecting:
		expected, _, _ := expectedOptimizeIncrementalArm(len(learning.Observations))
		return state.Phase == "CALIBRATING" && learning.Reason == optimizeIncrementalReasonPending &&
			len(learning.Observations) < optimizeRequiredCalibrationPairs*2 && learning.NextArm == expected
	case optimizeIncrementalComplete:
		return optimizeStringIn(state.Phase, "QUALIFIED", "NATIVE_RETAINED", "ACTIVE", "STALE") &&
			learning.Reason == optimizeIncrementalReasonComplete &&
			len(learning.Observations) == optimizeRequiredCalibrationPairs*2 &&
			learning.NextArm == optimizeIncrementalArmComplete
	case optimizeIncrementalRetained:
		return state.Phase == "NATIVE_RETAINED" && learning.Reason != "" &&
			learning.NextArm == optimizeIncrementalArmComplete
	default:
		return false
	}
}

func validOptimizeIncrementalObservation(observation optimizeIncrementalObservation, baseline bool) bool {
	if observation.DurationMS < 1 || observation.RequiredOutputCount < 1 ||
		!validOptimizeSHA(observation.RequiredOutputSHA256) || observation.ExitCode != 0 ||
		observation.IncrementalOverheadMS < 1 || observation.ProductAttributableFailure {
		return false
	}
	if _, err := time.Parse(time.RFC3339, observation.CapturedAt); err != nil {
		return false
	}
	if !validOptimizeIncrementalEconomics(observation) {
		return false
	}
	if baseline {
		return observation.Sequence == 0 && observation.Pair == 0 &&
			observation.Arm == optimizeIncrementalArmDiscovery && observation.Order == "BASELINE"
	}
	return true
}

func validOptimizeIncrementalEconomics(observation optimizeIncrementalObservation) bool {
	economics := observation.Economics
	if economics == (optimizeIncrementalEconomics{}) {
		// Existing checkpoints remain safe to resume; newly captured evidence
		// is required to carry complete attribution by its experiment checker.
		return true
	}
	if economics.GradleMS+observation.IncrementalOverheadMS != observation.DurationMS || economics.PreExecutionMS < 0 ||
		economics.PostExecutionMS < 0 || economics.MaterializationMS < 0 ||
		economics.OutputVerificationMS < 0 || economics.DiscoveryMS < 0 || economics.OtherWrapperMS < 0 {
		return false
	}
	return economics.PreExecutionMS+economics.PostExecutionMS == observation.IncrementalOverheadMS &&
		economics.MaterializationMS+economics.OutputVerificationMS+economics.DiscoveryMS+
			economics.OtherWrapperMS <= observation.IncrementalOverheadMS
}
