package launcher

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/outputequivalence"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	profileMeasureUsage           = "usage: buildopt profile measure --manifest PATH --graph PATH --generated-manifest PATH --changes-file PATH --fallback-changes-file PATH --base-revision REVISION --buildopt-revision REVISION --evidence-output PATH [--output-equivalence PATH] [--gradle-option VALUE ...] [--target-stability-confirmations 1|2|3] [--adaptive-candidate-stability] [--calibration-only] [--timeout DURATION]\n"
	measurementPairs              = 8
	maximumMeasurementInterArmGap = 5 * time.Second
)

var defaultStructuralGradleOptions = []string{
	"--daemon",
	"--build-cache",
	"--parallel",
	"--no-configuration-cache",
	"--console=plain",
	"--no-scan",
}

type structuralMeasurementConfig struct {
	repositoryRoot               string
	manifestPath                 string
	graphPath                    string
	generatedPath                string
	changesPath                  string
	fallbackChangesPath          string
	baseRevision                 string
	targetRevision               string
	buildOptRevision             string
	evidenceOutput               string
	gradleOptions                []string
	timeout                      time.Duration
	analysis                     profilediscovery.AnalysisReport
	executable                   string
	executableSHA256             string
	inputDocuments               map[string][]byte
	outputEquivalencePath        string
	outputEquivalenceSHA256      string
	outputEquivalence            *outputequivalence.Contract
	targetStabilityConfirmations int
	adaptiveCandidateStability   bool
	pairedTargetStability        bool
	gradleDistributionSeed       string
	gradleDependencySeed         string
	gradleReadOnlyDependencyRoot string
	gradleDependencySHA256       string
	gradleDependencyFileCount    int
	gradleDependencyByteCount    int64
	gradleNativeBuildCacheSeed   string
	gradleSharedBuildCacheSeed   string
	gradleMeasurementHome        string
	deadline                     time.Time
	parentContext                context.Context
}

type structuralMeasurementArm struct {
	name                   string
	workspace              string
	gradleHome             string
	buildCacheRoot         string
	cacheSeed              string
	readOnlyDependencyRoot string
	warmups                []profilediscovery.StructuralWarmupObservation
}

type structuralArmResult struct {
	durationMS   int64
	outputSHA    string
	outputCount  int
	log          string
	logSHA256    string
	taskOutcomes profilediscovery.StructuralTaskOutcomes
	hostPressure profilediscovery.StructuralHostPressure
	startedAt    time.Time
	finishedAt   time.Time
}

type measurementDependencyBinding struct {
	sha256    string
	fileCount int
	byteCount int64
}

type structuralCalibrationEvidence struct {
	SchemaVersion              string                                         `json:"schemaVersion"`
	CapturedAt                 string                                         `json:"capturedAt"`
	RepositoryRevision         string                                         `json:"repositoryRevision"`
	BaseRevision               string                                         `json:"baseRevision"`
	BuildOptRevision           string                                         `json:"buildoptRevision"`
	ExecutableSHA256           string                                         `json:"executableSha256"`
	SourceEvidenceSHA256       string                                         `json:"sourceEvidenceSha256"`
	OutputEquivalenceSHA256    string                                         `json:"outputEquivalenceSha256,omitempty"`
	ResolvedDependenciesSHA256 string                                         `json:"resolvedDependenciesSha256,omitempty"`
	Analysis                   profilediscovery.AnalysisReport                `json:"analysis"`
	GradleOptions              []string                                       `json:"gradleOptions"`
	CandidateWarmups           []profilediscovery.StructuralWarmupObservation `json:"candidateWarmups"`
	CandidateStabilization     string                                         `json:"candidateStabilization"`
	RequiredOutputSHA256       string                                         `json:"requiredOutputSha256"`
	RequiredOutputCount        int                                            `json:"requiredOutputCount"`
	Boundaries                 structuralCalibrationBoundaries                `json:"boundaries"`
}

type structuralCalibrationBoundaries struct {
	TimingClaim          bool `json:"timingClaim"`
	AutomaticActivation  bool `json:"automaticActivation"`
	ProductionAuthorized bool `json:"productionAuthorized"`
}

func runStructuralProfileMeasurement(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileMeasureUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile measure", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "checked Build Impact manifest")
	graph := flags.String("graph", "", "checked Build Impact graph")
	generated := flags.String("generated-manifest", "", "checked generated-state binding")
	changes := flags.String("changes-file", "", "exact base-to-target changed paths")
	fallbackChanges := flags.String("fallback-changes-file", "", "global-change fallback paths")
	baseRevision := flags.String("base-revision", "", "immutable baseline Git revision")
	buildOptRevision := flags.String("buildopt-revision", "", "immutable BuildOpt revision")
	evidenceOutput := flags.String("evidence-output", "", "repository-relative evidence output")
	outputEquivalence := flags.String("output-equivalence", "", "owner-reviewed semantic output-equivalence contract")
	timeout := flags.Duration("timeout", 20*time.Minute, "per-build timeout")
	targetStabilityConfirmations := flags.Int("target-stability-confirmations", 1, "target-workload warm-ups required before measured pairs")
	adaptiveCandidateStability := flags.Bool("adaptive-candidate-stability", false, "stop the candidate after two exact matching target fingerprints, bounded by three")
	calibrationOnly := flags.Bool("calibration-only", false, "capture candidate calibration phases without measured pairs or a qualification decision")
	var gradleOptions repeatedStringFlag
	flags.Var(&gradleOptions, "gradle-option", "Gradle option shared by both arms; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 ||
		*manifest == "" || *graph == "" || *generated == "" || *changes == "" ||
		*fallbackChanges == "" || *baseRevision == "" || *buildOptRevision == "" ||
		*evidenceOutput == "" || *timeout <= 0 ||
		(*targetStabilityConfirmations < 1 || *targetStabilityConfirmations > 3) ||
		(*adaptiveCandidateStability && *targetStabilityConfirmations != 3) {
		_, _ = io.WriteString(stderr, profileMeasureUsage)
		return exitUsage
	}
	if len(gradleOptions) == 0 {
		gradleOptions = append(gradleOptions, defaultStructuralGradleOptions...)
	}
	config, err := prepareStructuralMeasurementConfig(
		*manifest, *graph, *generated, *changes, *fallbackChanges,
		*baseRevision, *buildOptRevision, *evidenceOutput,
		append([]string(nil), gradleOptions...), *timeout,
		*targetStabilityConfirmations, *adaptiveCandidateStability, *outputEquivalence,
	)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile measurement unavailable: %v\n", err)
		return exitConfiguration
	}
	if *calibrationOnly {
		raw, calibrationErr := calibrateStructuralProfile(config, stderr)
		if calibrationErr != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: structural profile calibration unavailable: %v\n", calibrationErr)
			return exitConfiguration
		}
		if calibrationErr = writeRepositoryDocument(config.repositoryRoot, config.evidenceOutput, raw, 0o644); calibrationErr != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: structural profile calibration unavailable: %v\n", calibrationErr)
			return exitConfiguration
		}
		_, _ = fmt.Fprintf(stderr, "buildopt: structural candidate calibration written to %s; no qualification decision was made\n", config.evidenceOutput)
		if _, calibrationErr = stdout.Write(raw); calibrationErr != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: write structural profile calibration: %v\n", calibrationErr)
			return exitConfiguration
		}
		return 0
	}
	raw, qualified, err := measureStructuralProfile(config, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile measurement unavailable: %v\n", err)
		return exitConfiguration
	}
	if err := writeRepositoryDocument(config.repositoryRoot, config.evidenceOutput, raw, 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile measurement unavailable: %v\n", err)
		return exitConfiguration
	}
	decision := "RETAIN_NATIVE_GRADLE"
	if qualified {
		decision = "QUALIFIED_EVIDENCE_WRITTEN"
	}
	_, _ = fmt.Fprintf(stderr, "buildopt: structural profile measurement decision %s; review %s before evaluation\n", decision, config.evidenceOutput)
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write structural profile measurement: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func calibrateStructuralProfile(config structuralMeasurementConfig, progress io.Writer) ([]byte, error) {
	root, err := os.MkdirTemp("", "buildopt-structural-calibration-*")
	if err != nil {
		return nil, fmt.Errorf("create isolated calibration root: %w", err)
	}
	defer cleanupStructuralMeasurementRoot(root)
	config, err = prepareStructuralDependencySnapshot(config, root, progress)
	if err != nil {
		return nil, err
	}
	candidate, err := prepareStructuralMeasurementArm(config, root, "candidate", true, "", progress)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stopStructuralMeasurementArm(candidate) }()
	outputSHA, outputCount, err := hashStructuralMeasurementOutputs(config, candidate.workspace)
	if err != nil {
		return nil, fmt.Errorf("verify candidate calibration outputs: %w", err)
	}
	changesSHA := sha256.Sum256(config.inputDocuments[config.changesPath])
	executableSHA, err := hashMeasurementFile(config.executable)
	if err != nil {
		return nil, fmt.Errorf("hash installed BuildOpt executable: %w", err)
	}
	if executableSHA != config.executableSHA256 {
		return nil, errors.New("installed BuildOpt executable changed during calibration")
	}
	if err := verifyStructuralDependencyBinding(config); err != nil {
		return nil, err
	}
	evidence := structuralCalibrationEvidence{
		SchemaVersion:              "buildopt.evidence/poc-structural-calibration/v1",
		CapturedAt:                 time.Now().UTC().Format(time.RFC3339),
		RepositoryRevision:         config.targetRevision,
		BaseRevision:               config.baseRevision,
		BuildOptRevision:           config.buildOptRevision,
		ExecutableSHA256:           config.executableSHA256,
		SourceEvidenceSHA256:       hex.EncodeToString(changesSHA[:]),
		OutputEquivalenceSHA256:    config.outputEquivalenceSHA256,
		ResolvedDependenciesSHA256: config.gradleDependencySHA256,
		Analysis:                   config.analysis,
		GradleOptions:              append([]string(nil), config.gradleOptions...),
		CandidateWarmups:           append([]profilediscovery.StructuralWarmupObservation(nil), candidate.warmups...),
		CandidateStabilization:     structuralCandidateStabilizationPolicy(config),
		RequiredOutputSHA256:       outputSHA,
		RequiredOutputCount:        outputCount,
		Boundaries:                 structuralCalibrationBoundaries{},
	}
	raw, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render structural calibration evidence: %w", err)
	}
	return append(raw, '\n'), nil
}

func prepareStructuralMeasurementConfig(
	manifest, graph, generated, changes, fallbackChanges, baseRevision,
	buildOptRevision, evidenceOutput string,
	gradleOptions []string,
	timeout time.Duration,
	targetStabilityConfirmations int,
	adaptiveCandidateStability bool,
	outputEquivalencePath string,
) (structuralMeasurementConfig, error) {
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	inputPaths := map[string]string{
		"manifest": manifest, "graph": graph, "generated manifest": generated,
		"changes file": changes, "fallback changes file": fallbackChanges,
	}
	if outputEquivalencePath != "" {
		inputPaths["output equivalence"] = outputEquivalencePath
	}
	inputDocuments := make(map[string][]byte, len(inputPaths))
	seenInputPaths := make(map[string]string, len(inputPaths))
	for label, candidate := range inputPaths {
		if previous, exists := seenInputPaths[candidate]; exists {
			return structuralMeasurementConfig{}, fmt.Errorf("%s and %s must use distinct input paths", previous, label)
		}
		seenInputPaths[candidate] = label
		if err := validateMeasurementInputFile(repositoryRoot, candidate); err != nil {
			return structuralMeasurementConfig{}, fmt.Errorf("%s: %w", label, err)
		}
		raw, err := os.ReadFile(filepath.Join(repositoryRoot, candidate))
		if err != nil {
			return structuralMeasurementConfig{}, fmt.Errorf("read %s: %w", label, err)
		}
		inputDocuments[candidate] = raw
	}
	if evidenceOutput == "" || filepath.IsAbs(evidenceOutput) || filepath.Clean(evidenceOutput) != evidenceOutput || evidenceOutput == "." || evidenceOutput == ".." {
		return structuralMeasurementConfig{}, errors.New("evidence output must be clean and repository relative")
	}
	if label, exists := seenInputPaths[evidenceOutput]; exists {
		return structuralMeasurementConfig{}, fmt.Errorf("evidence output must not replace %s", label)
	}
	var equivalence *outputequivalence.Contract
	equivalenceSHA256 := ""
	if outputEquivalencePath != "" {
		parsed, err := outputequivalence.Parse(inputDocuments[outputEquivalencePath])
		if err != nil {
			return structuralMeasurementConfig{}, err
		}
		equivalence = &parsed
		equivalenceSHA256 = outputequivalence.SHA256(inputDocuments[outputEquivalencePath])
	}
	if !validMeasurementRevision(baseRevision) || !validMeasurementRevision(buildOptRevision) {
		return structuralMeasurementConfig{}, errors.New("base and BuildOpt revisions must be lowercase 40-character Git revisions")
	}
	if len(gradleOptions) == 0 {
		return structuralMeasurementConfig{}, errors.New("at least one Gradle option is required")
	}
	if targetStabilityConfirmations < 1 || targetStabilityConfirmations > 3 {
		return structuralMeasurementConfig{}, errors.New("target stability confirmations must be between one and three")
	}
	if adaptiveCandidateStability && targetStabilityConfirmations != 3 {
		return structuralMeasurementConfig{}, errors.New("adaptive candidate stabilization requires three bounded confirmations")
	}
	for _, option := range gradleOptions {
		if !validImpactGradleOption(option) {
			return structuralMeasurementConfig{}, fmt.Errorf("Gradle option is not allowed: %s", option)
		}
	}
	if len(gradleOptions) > 32 || !uniqueMeasurementStrings(gradleOptions) {
		return structuralMeasurementConfig{}, errors.New("Gradle options must be unique and bounded")
	}
	if dirty, gitErr := gitOutput(repositoryRoot, "status", "--porcelain", "--untracked-files=no"); gitErr != nil {
		return structuralMeasurementConfig{}, gitErr
	} else if strings.TrimSpace(dirty) != "" {
		return structuralMeasurementConfig{}, errors.New("tracked repository state must be clean before measurement")
	}
	targetRevision, err := gitOutput(repositoryRoot, "rev-parse", "HEAD")
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	targetRevision = strings.TrimSpace(targetRevision)
	if !validMeasurementRevision(targetRevision) || targetRevision == baseRevision {
		return structuralMeasurementConfig{}, errors.New("measurement requires distinct immutable base and target revisions")
	}
	resolvedBase, err := gitOutput(repositoryRoot, "rev-parse", baseRevision+"^{commit}")
	if err != nil || strings.TrimSpace(resolvedBase) != baseRevision {
		return structuralMeasurementConfig{}, errors.New("base revision is unavailable or ambiguous")
	}
	if err := gitRun(repositoryRoot, "merge-base", "--is-ancestor", baseRevision, targetRevision); err != nil {
		return structuralMeasurementConfig{}, errors.New("base revision must be an ancestor of the target revision")
	}
	declaredChanges, err := readImpactChangedPaths(repositoryRoot, changes)
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	if err := validateMeasurementChangeSet(repositoryRoot, baseRevision, targetRevision, declaredChanges); err != nil {
		return structuralMeasurementConfig{}, err
	}
	if _, err := readImpactChangedPaths(repositoryRoot, fallbackChanges); err != nil {
		return structuralMeasurementConfig{}, fmt.Errorf("fallback changes file: %w", err)
	}
	analysis, err := profilediscovery.AnalyzeOpportunity(profilediscovery.AnalysisOptions{
		RepositoryRoot: repositoryRoot,
		ManifestPath:   manifest,
		GraphPath:      graph,
		GeneratedPath:  generated,
	})
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	if analysis.Decision != profilediscovery.DecisionMeasure || analysis.Plan == nil {
		return structuralMeasurementConfig{}, fmt.Errorf("structural analysis retained native Gradle: %s", analysis.Reason)
	}
	for label, candidate := range inputPaths {
		current, err := os.ReadFile(filepath.Join(repositoryRoot, candidate))
		if err != nil || !bytes.Equal(current, inputDocuments[candidate]) {
			return structuralMeasurementConfig{}, fmt.Errorf("%s changed during measurement preparation", label)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return structuralMeasurementConfig{}, fmt.Errorf("resolve BuildOpt executable: %w", err)
	}
	executableSHA256, err := hashMeasurementFile(executable)
	if err != nil {
		return structuralMeasurementConfig{}, fmt.Errorf("hash installed BuildOpt executable: %w", err)
	}
	gradleDistributionSeed, err := measurementGradleDistributionSeed()
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	gradleDependencySeed, err := measurementGradleDependencySeed()
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	gradleNativeBuildCacheSeed, err := measurementGradleNativeBuildCacheSeed()
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	return structuralMeasurementConfig{
		repositoryRoot: repositoryRoot, manifestPath: manifest, graphPath: graph,
		generatedPath: generated, changesPath: changes,
		fallbackChangesPath: fallbackChanges, baseRevision: baseRevision,
		targetRevision: targetRevision, buildOptRevision: buildOptRevision,
		evidenceOutput: evidenceOutput, gradleOptions: gradleOptions,
		timeout: timeout, analysis: analysis, executable: executable,
		executableSHA256:             executableSHA256,
		inputDocuments:               inputDocuments,
		outputEquivalencePath:        outputEquivalencePath,
		outputEquivalenceSHA256:      equivalenceSHA256,
		outputEquivalence:            equivalence,
		targetStabilityConfirmations: targetStabilityConfirmations,
		adaptiveCandidateStability:   adaptiveCandidateStability,
		gradleDistributionSeed:       gradleDistributionSeed,
		gradleDependencySeed:         gradleDependencySeed,
		gradleNativeBuildCacheSeed:   gradleNativeBuildCacheSeed,
	}, nil
}

func measureStructuralProfile(config structuralMeasurementConfig, progress io.Writer) ([]byte, bool, error) {
	if err := checkStructuralMeasurementBudget(config); err != nil {
		return nil, false, err
	}
	root, err := os.MkdirTemp("", "buildopt-structural-measurement-*")
	if err != nil {
		return nil, false, fmt.Errorf("create isolated measurement root: %w", err)
	}
	defer cleanupStructuralMeasurementRoot(root)
	config, err = prepareStructuralDependencySnapshot(config, root, progress)
	if err != nil {
		return nil, false, err
	}
	if config.pairedTargetStability {
		config.gradleMeasurementHome, err = prepareStructuralSharedGradleHome(config, root)
		if err != nil {
			return nil, false, err
		}
		sharedDaemonArm := newStructuralMeasurementArm(config, root, "control")
		defer func() { _ = stopStructuralMeasurementArm(sharedDaemonArm) }()
	}
	control, err := prepareStructuralMeasurementArm(config, root, "control", false, "", progress)
	if err != nil {
		return nil, false, err
	}
	if !config.pairedTargetStability {
		defer func() { _ = stopStructuralMeasurementArm(control) }()
	}
	candidate, err := prepareStructuralMeasurementArm(config, root, "candidate", true, control.cacheSeed, progress)
	if err != nil {
		return nil, false, err
	}
	if !config.pairedTargetStability {
		defer func() { _ = stopStructuralMeasurementArm(candidate) }()
	}
	observations := make([]profilediscovery.StructuralMeasurementObservation, 0, measurementPairs)
	stableOutputSHA := ""
	stableOutputCount := 0
	stableControlFingerprint := ""
	stableCandidateFingerprint := ""
	if len(control.warmups) > 1 && len(candidate.warmups) > 0 {
		stableControlFingerprint = control.warmups[len(control.warmups)-1].TaskOutcomes.FingerprintSHA256
		stableCandidateFingerprint = candidate.warmups[len(candidate.warmups)-1].TaskOutcomes.FingerprintSHA256
	}
	for pair := 1; pair <= measurementPairs; pair++ {
		if err := checkStructuralMeasurementBudget(config); err != nil {
			return nil, false, err
		}
		order := "CANDIDATE_FIRST"
		if pair%2 == 1 {
			order = "CONTROL_FIRST"
		}
		// Prepare both isolated arms before starting either measured process. A
		// large repository can spend seconds resetting its checkout and restoring
		// the native cache; charging that work to the inter-arm gap would measure
		// fixture preparation rather than temporal comparability.
		if err := resetStructuralArm(config, control, config.targetRevision, true); err != nil {
			return nil, false, fmt.Errorf("pair %d prepare control arm: %w", pair, err)
		}
		if err := resetStructuralArm(config, candidate, config.targetRevision, true); err != nil {
			return nil, false, fmt.Errorf("pair %d prepare candidate arm: %w", pair, err)
		}
		var controlResult, candidateResult structuralArmResult
		if order == "CONTROL_FIRST" {
			controlResult, err = runStructuralArm(config, control, false, false, config.changesPath)
			if err == nil {
				candidateResult, err = runStructuralArm(config, candidate, true, true, config.changesPath)
			}
		} else {
			candidateResult, err = runStructuralArm(config, candidate, true, true, config.changesPath)
			if err == nil {
				controlResult, err = runStructuralArm(config, control, false, false, config.changesPath)
			}
		}
		if err != nil {
			return nil, false, fmt.Errorf("pair %d: %w", pair, err)
		}
		if config.pairedTargetStability {
			if err := validateStructuralPairedTargetShape(control, candidate, controlResult, candidateResult,
				stableControlFingerprint, stableCandidateFingerprint); err != nil {
				return nil, false, fmt.Errorf("pair %d: %w", pair, err)
			}
			if stableControlFingerprint == "" {
				stableControlFingerprint = controlResult.taskOutcomes.FingerprintSHA256
				stableCandidateFingerprint = candidateResult.taskOutcomes.FingerprintSHA256
			}
		}
		firstResult, secondResult := controlResult, candidateResult
		if order == "CANDIDATE_FIRST" {
			firstResult, secondResult = candidateResult, controlResult
		}
		interArmGap := secondResult.startedAt.Sub(firstResult.finishedAt)
		if interArmGap < 0 || interArmGap > maximumMeasurementInterArmGap {
			return nil, false, fmt.Errorf("pair %d inter-arm gap %s exceeds %s", pair, interArmGap, maximumMeasurementInterArmGap)
		}
		if !strings.Contains(candidateResult.log, "explicit Build Impact POC candidate "+config.analysis.Plan.AlternativeID+" selected") {
			return nil, false, errors.New("installed candidate did not select the analyzed structural alternative")
		}
		// Verify outputs only after both measured processes have finished. Output
		// traversal can be material on large graphs and must not inflate the gap
		// between the two Gradle starts.
		controlResult.outputSHA, controlResult.outputCount, err = hashStructuralMeasurementOutputs(config, control.workspace)
		if err != nil {
			return nil, false, fmt.Errorf("pair %d verify control outputs: %w", pair, err)
		}
		candidateResult.outputSHA, candidateResult.outputCount, err = hashStructuralMeasurementOutputs(config, candidate.workspace)
		if err != nil {
			return nil, false, fmt.Errorf("pair %d verify candidate outputs: %w", pair, err)
		}
		if controlResult.outputSHA != candidateResult.outputSHA || controlResult.outputCount != candidateResult.outputCount {
			return nil, false, fmt.Errorf("pair %d required outputs differ between optimized native Gradle and BuildOpt", pair)
		}
		if stableOutputSHA == "" {
			stableOutputSHA = controlResult.outputSHA
			stableOutputCount = controlResult.outputCount
		} else if stableOutputSHA != controlResult.outputSHA || stableOutputCount != controlResult.outputCount {
			return nil, false, fmt.Errorf("pair %d required outputs drifted across measurements", pair)
		}
		observations = append(observations, profilediscovery.StructuralMeasurementObservation{
			Pair: pair, Order: order,
			ControlDurationMS:     controlResult.durationMS,
			CandidateDurationMS:   candidateResult.durationMS,
			RequiredOutputSHA256:  controlResult.outputSHA,
			RequiredOutputCount:   controlResult.outputCount,
			ControlLogSHA256:      controlResult.logSHA256,
			CandidateLogSHA256:    candidateResult.logSHA256,
			ControlTaskOutcomes:   controlResult.taskOutcomes,
			CandidateTaskOutcomes: candidateResult.taskOutcomes,
			ControlHostPressure:   structuralHostPressurePointer(controlResult.hostPressure),
			CandidateHostPressure: structuralHostPressurePointer(candidateResult.hostPressure),
		})
		_, _ = fmt.Fprintf(progress, "buildopt: structural pair %d/%d control=%dms candidate=%dms saved=%dms gap=%dms order=%s\n",
			pair, measurementPairs, controlResult.durationMS, candidateResult.durationMS,
			controlResult.durationMS-candidateResult.durationMS, interArmGap.Milliseconds(), order)
		_, _ = fmt.Fprintf(progress, "buildopt: structural pair %d task outcomes control=%s candidate=%s\n",
			pair, formatStructuralTaskOutcomes(controlResult.taskOutcomes), formatStructuralTaskOutcomes(candidateResult.taskOutcomes))
	}
	// The fallback proves correctness; it is not part of the measured effect.
	// Standalone measurements use one daemon per arm, so release the control
	// daemon before fallback. Automatic paired measurements deliberately share
	// one daemon while their writable build caches remain isolated; preserve
	// that daemon for fallback instead of charging another cold JVM startup.
	if control.gradleHome != candidate.gradleHome {
		if err := stopStructuralMeasurementArm(control); err != nil {
			return nil, false, fmt.Errorf("stop control daemon before fallback: %w", err)
		}
	}
	fallbackResult, fallbackReason, err := measureStructuralFallback(config, candidate, progress)
	if err != nil {
		return nil, false, err
	}
	if fallbackResult.outputSHA != stableOutputSHA || fallbackResult.outputCount != stableOutputCount {
		difference, differenceErr := describeMeasurementOutputDifference(
			control.workspace,
			candidate.workspace,
			config.analysis.Plan.RequiredOutputs,
		)
		if differenceErr != nil {
			return nil, false, fmt.Errorf("full-graph fallback changed required outputs; inspect difference: %w", differenceErr)
		}
		return nil, false, fmt.Errorf("full-graph fallback changed required outputs: %s", difference)
	}
	changesRaw := config.inputDocuments[config.changesPath]
	changesSHA := sha256.Sum256(changesRaw)
	executableSHA, err := hashMeasurementFile(config.executable)
	if err != nil {
		return nil, false, fmt.Errorf("hash installed BuildOpt executable: %w", err)
	}
	if executableSHA != config.executableSHA256 {
		return nil, false, errors.New("installed BuildOpt executable changed during measurement")
	}
	if err := verifyStructuralDependencyBinding(config); err != nil {
		return nil, false, err
	}
	return profilediscovery.RenderStructuralMeasurementEvidence(profilediscovery.StructuralMeasurementOptions{
		CapturedAt: time.Now(), Analysis: config.analysis,
		RepositoryRevision:         config.targetRevision,
		BuildOptRevision:           config.buildOptRevision,
		ExecutableSHA256:           config.executableSHA256,
		SourceEvidenceSHA256:       hex.EncodeToString(changesSHA[:]),
		OutputEquivalenceSHA256:    config.outputEquivalenceSHA256,
		ResolvedDependenciesSHA256: config.gradleDependencySHA256,
		GradleOptions:              config.gradleOptions, Observations: observations,
		ControlWarmups: control.warmups, CandidateWarmups: candidate.warmups,
		CandidateStabilization: structuralCandidateStabilizationPolicy(config),
		FallbackReason:         fallbackReason, FallbackSuccessful: true,
	})
}

func prepareStructuralMeasurementArm(config structuralMeasurementConfig, root, name string, candidate bool, sharedCacheSeed string, progress io.Writer) (structuralMeasurementArm, error) {
	if err := checkStructuralMeasurementBudget(config); err != nil {
		return structuralMeasurementArm{}, err
	}
	totalWarmups := 2 + config.targetStabilityConfirmations
	if config.pairedTargetStability {
		totalWarmups = 2
	}
	arm := newStructuralMeasurementArm(config, root, name)
	arm.readOnlyDependencyRoot = config.gradleReadOnlyDependencyRoot
	if err := cloneStructuralMeasurementRepository(config.repositoryRoot, arm.workspace); err != nil {
		return arm, fmt.Errorf("clone %s arm: %w", name, err)
	}
	if err := os.MkdirAll(arm.gradleHome, 0o700); err != nil {
		return arm, fmt.Errorf("create %s Gradle home: %w", name, err)
	}
	if config.gradleMeasurementHome == "" && config.gradleDistributionSeed != "" {
		destination := filepath.Join(arm.gradleHome, "wrapper", "dists")
		if err := copyMeasurementDistributionTree(config.gradleDistributionSeed, destination); err != nil {
			return arm, fmt.Errorf("seed %s Gradle distribution: %w", name, err)
		}
	}
	if config.gradleSharedBuildCacheSeed != "" {
		destination := arm.buildCacheRoot
		if err := linkMeasurementCacheSeed(config.gradleSharedBuildCacheSeed, destination); err != nil {
			return arm, fmt.Errorf("seed %s native build cache: %w", name, err)
		}
	}
	if config.pairedTargetStability && sharedCacheSeed == "" && config.gradleSharedBuildCacheSeed != "" {
		// The authoritative native cache was snapshotted once before either arm
		// existed and is immutable. Reuse that identical seed directly instead
		// of rebuilding the base revision only to recreate an already-bound
		// cache. The target warm-up still proves the exact measured task shape.
		if err := linkMeasurementCacheSeed(config.gradleSharedBuildCacheSeed, arm.cacheSeed); err != nil {
			return arm, fmt.Errorf("snapshot authoritative %s native build cache: %w", name, err)
		}
		return prepareStructuralBoundCacheDaemonWarmup(config, arm, progress)
	}
	if config.pairedTargetStability && sharedCacheSeed != "" {
		cache := arm.buildCacheRoot
		if err := os.RemoveAll(cache); err != nil {
			return arm, fmt.Errorf("replace %s native build cache: %w", name, err)
		}
		restoreSeed := copyMeasurementCacheSeed
		if config.gradleSharedBuildCacheSeed != "" {
			restoreSeed = linkMeasurementCacheSeed
		}
		if err := restoreSeed(sharedCacheSeed, cache); err != nil {
			return arm, fmt.Errorf("share %s native build-cache seed: %w", name, err)
		}
		if err := linkMeasurementCacheSeed(sharedCacheSeed, arm.cacheSeed); err != nil {
			return arm, fmt.Errorf("snapshot shared %s native build cache: %w", name, err)
		}
		if config.gradleSharedBuildCacheSeed != "" {
			return arm, nil
		}
		return prepareStructuralPairedTargetWarmup(config, arm, candidate, 1, progress)
	}
	if err := resetStructuralArm(config, arm, config.baseRevision, false); err != nil {
		return arm, fmt.Errorf("prepare %s baseline: %w", name, err)
	}
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s arm at %s (cache seed 1/%d)\n", name, config.baseRevision, totalWarmups)
	seedWarmup, err := runStructuralArm(config, arm, candidate, candidate, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("warm %s arm: %w", name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("CACHE_SEED", seedWarmup))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s arm cache seed in %dms with %s\n",
		name, seedWarmup.durationMS, formatStructuralTaskOutcomes(seedWarmup.taskOutcomes))
	cache := arm.buildCacheRoot
	if err := snapshotMeasurementCacheSeed(cache, arm.cacheSeed); err != nil {
		return arm, fmt.Errorf("snapshot %s native build cache: %w", name, err)
	}
	if config.pairedTargetStability {
		return prepareStructuralPairedTargetWarmup(config, arm, candidate, 2, progress)
	}
	if err := resetStructuralArm(config, arm, config.baseRevision, true); err != nil {
		return arm, fmt.Errorf("prepare %s daemon stabilization: %w", name, err)
	}
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s arm at %s (base daemon stabilization 2/%d)\n", name, config.baseRevision, totalWarmups)
	stabilizationWarmup, err := runStructuralArm(config, arm, candidate, candidate, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("stabilize %s arm: %w", name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("BASE_DAEMON_STABILIZATION", stabilizationWarmup))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s arm daemon stabilization in %dms with %s\n",
		name, stabilizationWarmup.durationMS, formatStructuralTaskOutcomes(stabilizationWarmup.taskOutcomes))
	if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
		return arm, fmt.Errorf("prepare %s target-workload stabilization: %w", name, err)
	}
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s arm at %s (target-workload stabilization 3/%d)\n", name, config.targetRevision, totalWarmups)
	targetWarmup, err := runStructuralArm(config, arm, candidate, candidate, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("stabilize %s target workload: %w", name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("TARGET_WORKLOAD_STABILIZATION", targetWarmup))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s arm target workload in %dms with %s\n",
		name, targetWarmup.durationMS, formatStructuralTaskOutcomes(targetWarmup.taskOutcomes))
	for confirmationIndex := 2; confirmationIndex <= config.targetStabilityConfirmations; confirmationIndex++ {
		if err := checkStructuralMeasurementBudget(config); err != nil {
			return arm, err
		}
		if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
			return arm, fmt.Errorf("prepare %s target-workload stability confirmation: %w", name, err)
		}
		phase := "TARGET_WORKLOAD_STABILITY_CONFIRMATION"
		if confirmationIndex == 3 {
			phase = "TARGET_WORKLOAD_STABILITY_RECONFIRMATION"
		}
		_, _ = fmt.Fprintf(progress, "buildopt: confirming isolated %s target-workload shape at %s (%d/%d)\n",
			name, config.targetRevision, 2+confirmationIndex, totalWarmups)
		confirmation, err := runStructuralArm(config, arm, candidate, candidate, config.changesPath)
		if err != nil {
			return arm, fmt.Errorf("confirm %s target workload: %w", name, err)
		}
		arm.warmups = append(arm.warmups, structuralWarmupDiagnostic(phase, confirmation))
		_, _ = fmt.Fprintf(progress, "buildopt: confirmed isolated %s target workload in %dms with %s\n",
			name, confirmation.durationMS, formatStructuralTaskOutcomes(confirmation.taskOutcomes))
		if shouldStopAdaptiveCandidateStabilization(candidate, config.adaptiveCandidateStability, confirmationIndex, arm.warmups) {
			_, _ = fmt.Fprintf(progress, "buildopt: adaptive candidate stabilization converged after two exact target fingerprints; third warm-up omitted\n")
			break
		}
	}
	if config.targetStabilityConfirmations == 3 && !structuralTargetWarmupsConverged(arm.warmups) {
		return arm, fmt.Errorf("%s target workload did not converge in three bounded warm-ups: %s",
			name, describeStructuralTaskOutcomeDifference(
				arm.warmups[len(arm.warmups)-2].TaskOutcomes,
				arm.warmups[len(arm.warmups)-1].TaskOutcomes,
			))
	}
	return arm, nil
}

func cloneStructuralMeasurementRepository(source, destination string) error {
	originURL, originErr := gitOutput(source, "config", "--get", "remote.origin.url")
	if err := gitRun("", "clone", "--quiet", "--no-checkout", "--shared", "--", source, destination); err != nil {
		return err
	}
	if originErr != nil || strings.TrimSpace(originURL) == "" {
		return nil
	}
	// A local clone rewrites origin to the source checkout. Preserve the
	// repository's public identity because some Gradle builds derive metadata
	// from that URL even though calibration never fetches from the network.
	return gitRun(destination, "remote", "set-url", "origin", strings.TrimSpace(originURL))
}

func newStructuralMeasurementArm(config structuralMeasurementConfig, root, name string) structuralMeasurementArm {
	gradleHome := filepath.Join(root, name+"-gradle-home")
	buildCacheRoot := filepath.Join(gradleHome, "caches", "build-cache-1")
	if config.gradleMeasurementHome != "" {
		gradleHome = config.gradleMeasurementHome
		buildCacheRoot = filepath.Join(root, name+"-build-cache")
	}
	return structuralMeasurementArm{
		name:           name,
		workspace:      filepath.Join(root, name+"-repository"),
		gradleHome:     gradleHome,
		buildCacheRoot: buildCacheRoot,
		cacheSeed:      filepath.Join(root, name+"-build-cache-seed"),
	}
}

func prepareStructuralSharedGradleHome(config structuralMeasurementConfig, root string) (string, error) {
	gradleHome := filepath.Join(root, "shared-gradle-home")
	if err := os.MkdirAll(gradleHome, 0o700); err != nil {
		return "", fmt.Errorf("create shared measurement Gradle home: %w", err)
	}
	if config.gradleDistributionSeed != "" {
		destination := filepath.Join(gradleHome, "wrapper", "dists")
		if err := copyMeasurementDistributionTree(config.gradleDistributionSeed, destination); err != nil {
			return "", fmt.Errorf("seed shared measurement Gradle distribution: %w", err)
		}
	}
	initScript := filepath.Join(gradleHome, "init.d", "buildopt-measurement-cache.gradle")
	if err := writeStructuralBuildCacheInitScript(initScript, root); err != nil {
		return "", fmt.Errorf("configure isolated measurement build caches: %w", err)
	}
	return gradleHome, nil
}

func writeStructuralBuildCacheInitScript(scriptPath, measurementRoot string) error {
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o700); err != nil {
		return err
	}
	escapedRoot := strings.ReplaceAll(measurementRoot, `\`, `\\`)
	escapedRoot = strings.ReplaceAll(escapedRoot, `'`, `\'`)
	contents := "def buildoptMeasurementRoot = new File('" + escapedRoot + "').canonicalFile\n" +
		"settingsEvaluated { settings ->\n" +
		"    def settingsPath = settings.settingsDir.canonicalFile.toPath()\n" +
		"    def controlPath = new File(buildoptMeasurementRoot, 'control-repository').toPath()\n" +
		"    def candidatePath = new File(buildoptMeasurementRoot, 'candidate-repository').toPath()\n" +
		"    def cacheRoot\n" +
		"    if (settingsPath.startsWith(controlPath)) {\n" +
		"        cacheRoot = new File(buildoptMeasurementRoot, 'control-build-cache')\n" +
		"    } else if (settingsPath.startsWith(candidatePath)) {\n" +
		"        cacheRoot = new File(buildoptMeasurementRoot, 'candidate-build-cache')\n" +
		"    } else {\n" +
		"        throw new GradleException('measurement build escaped the isolated control and candidate roots')\n" +
		"    }\n" +
		"    settings.buildCache {\n" +
		"        local { directory = cacheRoot }\n" +
		"    }\n" +
		"}\n"
	return os.WriteFile(scriptPath, []byte(contents), 0o600)
}

func prepareStructuralPairedTargetWarmup(config structuralMeasurementConfig, arm structuralMeasurementArm, candidate bool, phaseIndex int, progress io.Writer) (structuralMeasurementArm, error) {
	if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
		return arm, fmt.Errorf("prepare %s paired target stabilization: %w", arm.name, err)
	}
	totalWarmups := 2
	if candidate {
		totalWarmups = 1
	}
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s arm at %s (paired target stabilization %d/%d)\n",
		arm.name, config.targetRevision, phaseIndex, totalWarmups)
	result, err := runStructuralArm(config, arm, candidate, candidate, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("stabilize %s paired target workload: %w", arm.name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("TARGET_WORKLOAD_STABILIZATION", result))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s paired target workload in %dms with %s\n",
		arm.name, result.durationMS, formatStructuralTaskOutcomes(result.taskOutcomes))
	return arm, nil
}

func validateStructuralPairedTargetShape(
	control, candidate structuralMeasurementArm,
	controlResult, candidateResult structuralArmResult,
	controlBaseline, candidateBaseline string,
) error {
	legacyWarmups := len(control.warmups) == 2 && len(candidate.warmups) == 1
	boundCacheWarmups := len(control.warmups) == 1 && len(candidate.warmups) == 0
	if !legacyWarmups && !boundCacheWarmups {
		return errors.New("paired target-shape warm-up evidence is incomplete")
	}
	if legacyWarmups {
		controlBaseline = control.warmups[len(control.warmups)-1].TaskOutcomes.FingerprintSHA256
		candidateBaseline = candidate.warmups[len(candidate.warmups)-1].TaskOutcomes.FingerprintSHA256
	}
	if controlBaseline == "" && candidateBaseline == "" {
		return nil
	}
	if controlBaseline == "" || candidateBaseline == "" ||
		controlResult.taskOutcomes.FingerprintSHA256 != controlBaseline ||
		candidateResult.taskOutcomes.FingerprintSHA256 != candidateBaseline {
		return errors.New("measured task shape drifted from paired target stabilization")
	}
	return nil
}

func prepareStructuralBoundCacheDaemonWarmup(config structuralMeasurementConfig, arm structuralMeasurementArm, progress io.Writer) (structuralMeasurementArm, error) {
	if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
		return arm, fmt.Errorf("prepare %s daemon stabilization: %w", arm.name, err)
	}
	warmupConfig := config
	plan := *config.analysis.Plan
	plan.FallbackEntrypoints = []string{"help"}
	warmupConfig.analysis.Plan = &plan
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s daemon and repository configuration with Gradle help (1/1)\n", arm.name)
	result, err := runStructuralArm(warmupConfig, arm, false, false, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("stabilize %s daemon: %w", arm.name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("DAEMON_STABILIZATION", result))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s daemon in %dms with %s\n",
		arm.name, result.durationMS, formatStructuralTaskOutcomes(result.taskOutcomes))
	return arm, nil
}

func structuralTargetWarmupsConverged(warmups []profilediscovery.StructuralWarmupObservation) bool {
	if len(warmups) < 4 {
		return false
	}
	previous := warmups[len(warmups)-2].TaskOutcomes.FingerprintSHA256
	current := warmups[len(warmups)-1].TaskOutcomes.FingerprintSHA256
	return previous != "" && current == previous
}

func shouldStopAdaptiveCandidateStabilization(candidate, adaptive bool, confirmationIndex int, warmups []profilediscovery.StructuralWarmupObservation) bool {
	return candidate && adaptive && confirmationIndex == 2 && structuralTargetWarmupsConverged(warmups)
}

func structuralCandidateStabilizationPolicy(config structuralMeasurementConfig) string {
	if config.pairedTargetStability {
		if config.gradleSharedBuildCacheSeed != "" {
			return profilediscovery.CandidateStabilizationPairedBoundCacheShape
		}
		return profilediscovery.CandidateStabilizationPairedTargetShape
	}
	if config.adaptiveCandidateStability {
		return profilediscovery.CandidateStabilizationAdaptiveExactTwoOfThree
	}
	return ""
}

func describeStructuralTaskOutcomeDifference(previous, current profilediscovery.StructuralTaskOutcomes) string {
	const maximumDifferences = 16
	previousByPath := make(map[string]string, len(previous.Tasks))
	currentByPath := make(map[string]string, len(current.Tasks))
	paths := make(map[string]struct{}, len(previous.Tasks)+len(current.Tasks))
	for _, task := range previous.Tasks {
		previousByPath[task.Path] = task.Outcome
		paths[task.Path] = struct{}{}
	}
	for _, task := range current.Tasks {
		currentByPath[task.Path] = task.Outcome
		paths[task.Path] = struct{}{}
	}
	orderedPaths := make([]string, 0, len(paths))
	for taskPath := range paths {
		orderedPaths = append(orderedPaths, taskPath)
	}
	sort.Strings(orderedPaths)
	differences := make([]string, 0, maximumDifferences)
	totalDifferences := 0
	for _, taskPath := range orderedPaths {
		before, beforePresent := previousByPath[taskPath]
		after, afterPresent := currentByPath[taskPath]
		if beforePresent && afterPresent && before == after {
			continue
		}
		totalDifferences++
		if len(differences) == maximumDifferences {
			continue
		}
		if !beforePresent {
			before = "ABSENT"
		}
		if !afterPresent {
			after = "ABSENT"
		}
		differences = append(differences, fmt.Sprintf("%s %s -> %s", taskPath, before, after))
	}
	if totalDifferences == 0 {
		return "fingerprints differ without a task-level difference"
	}
	description := strings.Join(differences, "; ")
	if totalDifferences > len(differences) {
		description += fmt.Sprintf("; and %d more", totalDifferences-len(differences))
	}
	return description
}

func structuralWarmupDiagnostic(phase string, result structuralArmResult) profilediscovery.StructuralWarmupObservation {
	return profilediscovery.StructuralWarmupObservation{
		Phase: phase, DurationMS: result.durationMS, LogSHA256: result.logSHA256,
		TaskOutcomes: result.taskOutcomes,
		HostPressure: structuralHostPressurePointer(result.hostPressure),
	}
}

func structuralHostPressurePointer(pressure profilediscovery.StructuralHostPressure) *profilediscovery.StructuralHostPressure {
	if !pressure.Available {
		return nil
	}
	copy := pressure
	return &copy
}

func measureStructuralFallback(config structuralMeasurementConfig, arm structuralMeasurementArm, progress io.Writer) (structuralArmResult, string, error) {
	if err := checkStructuralMeasurementBudget(config); err != nil {
		return structuralArmResult{}, "", err
	}
	if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
		return structuralArmResult{}, "", err
	}
	fallbackConfig := config
	fallbackConfig.gradleOptions = structuralFallbackGradleOptions(config.gradleOptions)
	_, _ = fmt.Fprintln(progress, "buildopt: validating full-graph fallback with the candidate arm scheduling")
	// The fallback still enters through BuildOpt, but it must reject the
	// analyzed alternative and run the complete graph.
	result, err := runStructuralArm(fallbackConfig, arm, true, false, config.fallbackChangesPath)
	if err != nil {
		return result, "", fmt.Errorf("full-graph fallback: %w", err)
	}
	reason := measurementFallbackReason(result.log)
	if reason == "" {
		return result, "", errors.New("installed fallback did not report full-graph execution")
	}
	result.outputSHA, result.outputCount, err = hashStructuralMeasurementOutputs(config, arm.workspace)
	return result, reason, err
}

func structuralFallbackGradleOptions(measured []string) []string {
	return append([]string(nil), measured...)
}

func resetStructuralArm(config structuralMeasurementConfig, arm structuralMeasurementArm, revision string, restoreCache bool) error {
	if err := gitRun(arm.workspace, "reset", "--hard", "--quiet", revision); err != nil {
		return err
	}
	if err := gitRun(arm.workspace, "clean", "-ffdx", "--quiet"); err != nil {
		return err
	}
	if restoreCache {
		cache := arm.buildCacheRoot
		if err := os.RemoveAll(cache); err != nil {
			return fmt.Errorf("clear isolated native build cache: %w", err)
		}
		restoreSeed := copyMeasurementCacheSeed
		if config.gradleSharedBuildCacheSeed != "" {
			restoreSeed = linkMeasurementCacheSeed
		}
		if err := restoreSeed(arm.cacheSeed, cache); err != nil {
			return fmt.Errorf("restore isolated native build cache: %w", err)
		}
	}
	inputs := []string{config.manifestPath, config.graphPath, config.generatedPath, config.changesPath, config.fallbackChangesPath}
	if config.outputEquivalencePath != "" {
		inputs = append(inputs, config.outputEquivalencePath)
	}
	for _, input := range inputs {
		if err := copyMeasurementInputDocument(arm.workspace, input, config.inputDocuments[input]); err != nil {
			return err
		}
	}
	return nil
}

func runStructuralArm(config structuralMeasurementConfig, arm structuralMeasurementArm, candidate, requireAlternative bool, changesPath string) (structuralArmResult, error) {
	var command []string
	if candidate {
		command = []string{config.executable, "impact",
			"--repository-id", config.analysis.Subject.RepositoryID,
			"--pipeline-class", config.analysis.Subject.PipelineClass,
			"--changes-file", changesPath,
			"--manifest", config.manifestPath,
			"--graph", config.graphPath,
			"--generated-manifest", config.generatedPath,
		}
		for _, option := range config.gradleOptions {
			command = append(command, "--gradle-option="+option)
		}
	} else {
		command = measurementGradleCommand(arm.workspace, append(append([]string(nil), config.gradleOptions...), config.analysis.Plan.FallbackEntrypoints...))
	}
	timeout, err := structuralMeasurementTimeout(config)
	if err != nil {
		return structuralArmResult{}, err
	}
	ctx, cancel := structuralMeasurementChildContext(config, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = arm.workspace
	cmd.Env = measurementEnvironment(arm.gradleHome, arm.readOnlyDependencyRoot)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	pressureBefore := readStructuralPressureSnapshot()
	started := time.Now()
	err = cmd.Run()
	finished := time.Now()
	pressureAfter := readStructuralPressureSnapshot()
	duration := finished.Sub(started).Milliseconds()
	if duration < 1 {
		duration = 1
	}
	logText := log.String()
	logSHA := sha256.Sum256([]byte(logText))
	result := structuralArmResult{
		durationMS: duration, log: logText, logSHA256: hex.EncodeToString(logSHA[:]),
		taskOutcomes: summarizeStructuralTaskOutcomes(logText),
		hostPressure: structuralPressureDelta(pressureBefore, pressureAfter),
		startedAt:    started, finishedAt: finished,
	}
	if validationErr := profilediscovery.ValidateStructuralTaskOutcomes(result.taskOutcomes); validationErr != nil {
		return result, structuralTaskEvidenceError(arm.name, validationErr, result.log)
	}
	if validationErr := profilediscovery.ValidateStructuralHostPressure(structuralHostPressurePointer(result.hostPressure)); validationErr != nil {
		return result, fmt.Errorf("%s arm produced invalid host-pressure evidence: %w", arm.name, validationErr)
	}
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("%s arm exceeded %s: %w", arm.name, timeout, context.DeadlineExceeded)
	}
	if err != nil {
		return result, fmt.Errorf("%s arm failed: %w\n%s", arm.name, err, tailMeasurementLog(result.log, 80))
	}
	if requireAlternative && !strings.Contains(result.log,
		"explicit Build Impact POC candidate "+config.analysis.Plan.AlternativeID+" selected") {
		return result, errors.New("installed candidate did not select the analyzed structural alternative")
	}
	return result, nil
}

func structuralMeasurementChildContext(config structuralMeasurementConfig, timeout time.Duration) (context.Context, context.CancelFunc) {
	parent := config.parentContext
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}

func checkStructuralMeasurementBudget(config structuralMeasurementConfig) error {
	_, err := structuralMeasurementTimeout(config)
	return err
}

func structuralMeasurementTimeout(config structuralMeasurementConfig) (time.Duration, error) {
	timeout := config.timeout
	if !config.deadline.IsZero() {
		remaining := time.Until(config.deadline)
		if remaining <= 0 {
			return 0, context.DeadlineExceeded
		}
		if remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		return 0, context.DeadlineExceeded
	}
	return timeout, nil
}

func structuralTaskEvidenceError(armName string, validationErr error, log string) error {
	return fmt.Errorf("%s arm produced invalid exact task evidence: %w\n%s",
		armName, validationErr, tailMeasurementLog(log, 80))
}

func summarizeStructuralTaskOutcomes(log string) profilediscovery.StructuralTaskOutcomes {
	var outcomes profilediscovery.StructuralTaskOutcomes
	type observedTask struct {
		outcome     string
		transitions []string
	}
	observed := map[string]*observedTask{}
	scanner := bufio.NewScanner(strings.NewReader(log))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "> Task ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		outcome := "EXECUTED"
		switch {
		case strings.HasSuffix(line, " FROM-CACHE"):
			outcome = "FROM_CACHE"
		case strings.HasSuffix(line, " UP-TO-DATE"):
			outcome = "UP_TO_DATE"
		case strings.HasSuffix(line, " NO-SOURCE"):
			outcome = "NO_SOURCE"
		case strings.HasSuffix(line, " SKIPPED"):
			outcome = "SKIPPED"
		}
		path := fields[2]
		task := observed[path]
		if task == nil {
			task = &observedTask{}
			observed[path] = task
		}
		if len(task.transitions) == 0 || task.transitions[len(task.transitions)-1] != outcome {
			task.transitions = append(task.transitions, outcome)
		}
		task.outcome = outcome
	}
	if len(observed) > 0 {
		tasks := make([]profilediscovery.StructuralTaskObservation, 0, len(observed))
		for path, observedTask := range observed {
			outcome := observedTask.outcome
			task := profilediscovery.StructuralTaskObservation{Path: path, Outcome: outcome}
			if len(observedTask.transitions) > 1 {
				task.ConsoleOutcomeTransitions = append([]string(nil), observedTask.transitions...)
			}
			tasks = append(tasks, task)
			switch outcome {
			case "FROM_CACHE":
				outcomes.FromCache++
			case "UP_TO_DATE":
				outcomes.UpToDate++
			case "NO_SOURCE":
				outcomes.NoSource++
			case "SKIPPED":
				outcomes.Skipped++
			default:
				outcomes.Executed++
			}
		}
		sort.Slice(tasks, func(left, right int) bool { return tasks[left].Path < tasks[right].Path })
		canonical := make([]string, len(tasks))
		for index, task := range tasks {
			canonical[index] = canonicalStructuralTaskLine(task)
		}
		digest := sha256.Sum256([]byte(strings.Join(canonical, "\n") + "\n"))
		outcomes.Total = len(tasks)
		outcomes.FingerprintSHA256 = hex.EncodeToString(digest[:])
		outcomes.Tasks = tasks
	}
	return outcomes
}

func canonicalStructuralTaskLine(task profilediscovery.StructuralTaskObservation) string {
	line := "> Task " + task.Path
	switch task.Outcome {
	case "FROM_CACHE":
		line += " FROM-CACHE"
	case "UP_TO_DATE":
		line += " UP-TO-DATE"
	case "NO_SOURCE":
		line += " NO-SOURCE"
	case "SKIPPED":
		line += " SKIPPED"
	default:
	}
	return line
}

func formatStructuralTaskOutcomes(outcomes profilediscovery.StructuralTaskOutcomes) string {
	return fmt.Sprintf("total:%d/executed:%d/from-cache:%d/up-to-date:%d/no-source:%d/skipped:%d/fingerprint:%s",
		outcomes.Total, outcomes.Executed, outcomes.FromCache, outcomes.UpToDate, outcomes.NoSource,
		outcomes.Skipped, outcomes.FingerprintSHA256)
}

type structuralPressureSnapshot struct {
	available         bool
	cpuSomeTotalUS    int64
	memorySomeTotalUS int64
	memoryFullTotalUS int64
	ioSomeTotalUS     int64
	ioFullTotalUS     int64
}

func readStructuralPressureSnapshot() structuralPressureSnapshot {
	if runtime.GOOS != "linux" {
		return structuralPressureSnapshot{}
	}
	cpuRaw, cpuErr := os.ReadFile("/proc/pressure/cpu")
	memoryRaw, memoryErr := os.ReadFile("/proc/pressure/memory")
	ioRaw, ioErr := os.ReadFile("/proc/pressure/io")
	if cpuErr != nil || memoryErr != nil || ioErr != nil {
		return structuralPressureSnapshot{}
	}
	cpuSome, _, err := parseStructuralPressure(cpuRaw, false)
	if err != nil {
		return structuralPressureSnapshot{}
	}
	memorySome, memoryFull, err := parseStructuralPressure(memoryRaw, true)
	if err != nil {
		return structuralPressureSnapshot{}
	}
	ioSome, ioFull, err := parseStructuralPressure(ioRaw, true)
	if err != nil {
		return structuralPressureSnapshot{}
	}
	return structuralPressureSnapshot{
		available: true, cpuSomeTotalUS: cpuSome,
		memorySomeTotalUS: memorySome, memoryFullTotalUS: memoryFull,
		ioSomeTotalUS: ioSome, ioFullTotalUS: ioFull,
	}
}

func parseStructuralPressure(raw []byte, requireFull bool) (int64, int64, error) {
	values := map[string]int64{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || (fields[0] != "some" && fields[0] != "full") {
			continue
		}
		for _, field := range fields[1:] {
			if !strings.HasPrefix(field, "total=") {
				continue
			}
			value, err := strconv.ParseInt(strings.TrimPrefix(field, "total="), 10, 64)
			if err != nil || value < 0 {
				return 0, 0, errors.New("Linux PSI total is invalid")
			}
			values[fields[0]] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	some, someOK := values["some"]
	full, fullOK := values["full"]
	if !someOK || (requireFull && !fullOK) {
		return 0, 0, errors.New("Linux PSI totals are incomplete")
	}
	return some, full, nil
}

func structuralPressureDelta(before, after structuralPressureSnapshot) profilediscovery.StructuralHostPressure {
	if !before.available || !after.available ||
		after.cpuSomeTotalUS < before.cpuSomeTotalUS ||
		after.memorySomeTotalUS < before.memorySomeTotalUS ||
		after.memoryFullTotalUS < before.memoryFullTotalUS ||
		after.ioSomeTotalUS < before.ioSomeTotalUS ||
		after.ioFullTotalUS < before.ioFullTotalUS {
		return profilediscovery.StructuralHostPressure{}
	}
	return profilediscovery.StructuralHostPressure{
		Available:         true,
		CPUSomeTotalUS:    after.cpuSomeTotalUS - before.cpuSomeTotalUS,
		MemorySomeTotalUS: after.memorySomeTotalUS - before.memorySomeTotalUS,
		MemoryFullTotalUS: after.memoryFullTotalUS - before.memoryFullTotalUS,
		IOSomeTotalUS:     after.ioSomeTotalUS - before.ioSomeTotalUS,
		IOFullTotalUS:     after.ioFullTotalUS - before.ioFullTotalUS,
	}
}

func measurementGradleCommand(repositoryRoot string, args []string) []string {
	wrapper := filepath.Join(repositoryRoot, gradleWrapperName(runtime.GOOS))
	if runtime.GOOS == "windows" {
		return append([]string{"cmd.exe", "/d", "/s", "/c", wrapper}, args...)
	}
	return append([]string{wrapper}, args...)
}

func stopStructuralMeasurementArm(arm structuralMeasurementArm) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := measurementGradleCommand(arm.workspace, []string{"--stop"})
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = arm.workspace
	cmd.Env = measurementEnvironment(arm.gradleHome, arm.readOnlyDependencyRoot)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stop %s Gradle daemon: %w", arm.name, err)
	}
	return nil
}

func measurementEnvironment(gradleHome, readOnlyDependencyRoot string) []string {
	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "BUILDOPT_") || name == "GRADLE_USER_HOME" || name == gradleReadOnlyDependencyEnvironment {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment, "GRADLE_USER_HOME="+gradleHome)
	if readOnlyDependencyRoot != "" {
		environment = append(environment, gradleReadOnlyDependencyEnvironment+"="+readOnlyDependencyRoot)
	}
	return environment
}

func measurementGradleDistributionSeed() (string, error) {
	gradleHome := os.Getenv("GRADLE_USER_HOME")
	if gradleHome == "" {
		return "", nil
	}
	candidate := filepath.Join(gradleHome, "wrapper", "dists")
	info, err := os.Lstat(candidate)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect Gradle distribution seed: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Gradle distribution seed must be a real directory")
	}
	return candidate, nil
}

func measurementGradleDependencySeed() (string, error) {
	gradleHome := os.Getenv("GRADLE_USER_HOME")
	if gradleHome == "" {
		return "", nil
	}
	candidate := filepath.Join(gradleHome, "caches", "modules-2")
	info, err := os.Lstat(candidate)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect Gradle dependency seed: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Gradle dependency seed must be a real directory")
	}
	return candidate, nil
}

func measurementGradleNativeBuildCacheSeed() (string, error) {
	gradleHome := os.Getenv("GRADLE_USER_HOME")
	if gradleHome == "" {
		return "", nil
	}
	candidate := filepath.Join(gradleHome, "caches", "build-cache-1")
	info, err := os.Lstat(candidate)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect native Gradle build-cache seed: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("native Gradle build-cache seed must be a real directory")
	}
	return candidate, nil
}

func validateMeasurementInputFile(repositoryRoot, relativePath string) error {
	if relativePath == "" || filepath.IsAbs(relativePath) || filepath.Clean(relativePath) != relativePath || relativePath == "." || relativePath == ".." {
		return errors.New("input must be clean and repository relative")
	}
	path := filepath.Join(repositoryRoot, relativePath)
	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return err
	}
	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return errors.New("input escapes the repository")
	}
	info, err := os.Stat(canonicalPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return errors.New("input must be a bounded regular file")
	}
	return nil
}

func validateMeasurementChangeSet(repositoryRoot, baseRevision, targetRevision string, declared []string) error {
	raw, err := gitOutput(repositoryRoot, "diff", "--name-only", "--no-renames", "-z", baseRevision, targetRevision, "--")
	if err != nil {
		return err
	}
	actual := nullDelimitedPaths(raw)
	expected := append([]string(nil), declared...)
	sort.Strings(actual)
	sort.Strings(expected)
	if !sameMeasurementStrings(actual, expected) {
		return errors.New("changes file must exactly match the base-to-target Git diff")
	}
	return nil
}

func hashMeasurementOutputs(repositoryRoot string, patterns []string) (string, int, error) {
	outputs, err := measurementOutputDigests(repositoryRoot, patterns)
	if err != nil {
		return "", 0, err
	}
	files := make([]string, 0, len(outputs))
	for relative := range outputs {
		files = append(files, relative)
	}
	sort.Strings(files)
	manifest := sha256.New()
	for _, relative := range files {
		_, _ = fmt.Fprintf(manifest, "%s  %s\n", outputs[relative], relative)
	}
	return hex.EncodeToString(manifest.Sum(nil)), len(files), nil
}

func hashStructuralMeasurementOutputs(config structuralMeasurementConfig, repositoryRoot string) (string, int, error) {
	return outputequivalence.HashOutputs(repositoryRoot, config.analysis.Plan.RequiredOutputs, config.outputEquivalence)
}

func measurementOutputDigests(repositoryRoot string, patterns []string) (map[string]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(repositoryRoot, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repositoryRoot, candidate)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesMeasurementOutput(patterns, relative) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("required output is not a regular file: %s", relative)
		}
		files = append(files, relative)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect required outputs: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, errors.New("measurement produced no required outputs")
	}
	outputs := make(map[string]string, len(files))
	for _, relative := range files {
		file, err := os.Open(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		if err != nil {
			return nil, err
		}
		digest := sha256.New()
		_, copyErr := io.Copy(digest, file)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		outputs[relative] = hex.EncodeToString(digest.Sum(nil))
	}
	return outputs, nil
}

func describeMeasurementOutputDifference(expectedRoot, actualRoot string, patterns []string) (string, error) {
	expected, err := measurementOutputDigests(expectedRoot, patterns)
	if err != nil {
		return "", fmt.Errorf("expected outputs: %w", err)
	}
	actual, err := measurementOutputDigests(actualRoot, patterns)
	if err != nil {
		return "", fmt.Errorf("actual outputs: %w", err)
	}
	differences := make([]string, 0)
	for path, expectedDigest := range expected {
		actualDigest, ok := actual[path]
		switch {
		case !ok:
			differences = append(differences, "missing "+path)
		case actualDigest != expectedDigest:
			differences = append(differences, "changed "+path)
		}
	}
	for path := range actual {
		if _, ok := expected[path]; !ok {
			differences = append(differences, "unexpected "+path)
		}
	}
	sort.Strings(differences)
	if len(differences) == 0 {
		return "aggregate digest differs without a path-level difference", nil
	}
	const maximumReportedDifferences = 10
	reported := differences
	if len(reported) > maximumReportedDifferences {
		reported = reported[:maximumReportedDifferences]
	}
	description := strings.Join(reported, "; ")
	if len(reported) != len(differences) {
		description += fmt.Sprintf("; and %d more", len(differences)-len(reported))
	}
	return description, nil
}

func matchesMeasurementOutput(patterns []string, candidate string) bool {
	for _, pattern := range patterns {
		if matchMeasurementGlob(pattern, candidate) {
			return true
		}
	}
	return false
}

func matchMeasurementGlob(pattern, candidate string) bool {
	patternParts := strings.Split(filepath.ToSlash(pattern), "/")
	candidateParts := strings.Split(filepath.ToSlash(candidate), "/")
	var match func(int, int) bool
	match = func(patternIndex, candidateIndex int) bool {
		if patternIndex == len(patternParts) {
			return candidateIndex == len(candidateParts)
		}
		if patternParts[patternIndex] == "**" {
			for next := candidateIndex; next <= len(candidateParts); next++ {
				if match(patternIndex+1, next) {
					return true
				}
			}
			return false
		}
		if candidateIndex == len(candidateParts) {
			return false
		}
		matched, err := path.Match(patternParts[patternIndex], candidateParts[candidateIndex])
		return err == nil && matched && match(patternIndex+1, candidateIndex+1)
	}
	return match(0, 0)
}

func copyMeasurementInput(sourceRoot, targetRoot, relativePath string) error {
	raw, err := os.ReadFile(filepath.Join(sourceRoot, relativePath))
	if err != nil {
		return fmt.Errorf("read measurement input %s: %w", relativePath, err)
	}
	return copyMeasurementInputDocument(targetRoot, relativePath, raw)
}

func copyMeasurementInputDocument(targetRoot, relativePath string, raw []byte) error {
	target := filepath.Join(targetRoot, relativePath)
	if err := ensureMeasurementDirectory(targetRoot, filepath.Dir(target)); err != nil {
		return err
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("measurement input target is a symlink: %s", relativePath)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		return fmt.Errorf("copy measurement input %s: %w", relativePath, err)
	}
	return nil
}

func ensureMeasurementDirectory(root, directory string) error {
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return errors.New("measurement input directory escapes the isolated repository")
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, segment := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, inspectErr := os.Lstat(current)
		if os.IsNotExist(inspectErr) {
			if err := os.Mkdir(current, 0o755); err != nil {
				return err
			}
			continue
		}
		if inspectErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("measurement input directory contains a symlink or non-directory component")
		}
	}
	return nil
}

func snapshotMeasurementCacheSeed(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("native build cache seed is unavailable")
	}
	return filepath.WalkDir(source, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, candidate)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("native build cache seed contains a non-regular entry")
		}
		if entry.Name() == "gc.properties" || strings.HasSuffix(entry.Name(), ".lock") {
			return nil
		}
		return copyMeasurementRegularFileWithMode(candidate, destination, 0o400)
	})
}

func linkMeasurementCacheSeed(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("native build cache seed is unavailable")
	}
	return filepath.WalkDir(source, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, candidate)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		entryInfo, err := entry.Info()
		if err != nil || !entryInfo.Mode().IsRegular() || entryInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("native build cache seed contains a non-regular entry")
		}
		if entryInfo.Mode().Perm()&0o222 != 0 {
			return errors.New("native build cache seed contains a writable entry")
		}
		return os.Link(candidate, destination)
	})
}

func copyMeasurementCacheSeed(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("native build cache seed is unavailable")
	}
	return filepath.WalkDir(source, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, candidate)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		entryInfo, err := entry.Info()
		if err != nil || !entryInfo.Mode().IsRegular() || entryInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("native build cache seed contains a non-regular entry")
		}
		return copyMeasurementRegularFileWithMode(candidate, destination, 0o600)
	})
}

// prepareStructuralDependencySnapshot makes the dependency artifacts and
// native build-cache entries produced by the authoritative build available to
// both isolated measurement arms. Resolved dependencies are bound in place
// through Gradle's read-only cache contract and verified before and after the
// measurement. Writable build caches, Configuration Cache state, and
// repository outputs remain private per arm. Automatic paired measurements
// may share the Gradle daemon process, but never a writable build-cache
// directory.
func prepareStructuralDependencySnapshot(config structuralMeasurementConfig, root string, progress io.Writer) (structuralMeasurementConfig, error) {
	if config.gradleDependencySeed == "" && config.gradleNativeBuildCacheSeed == "" {
		return config, nil
	}
	if err := checkStructuralMeasurementBudget(config); err != nil {
		return config, err
	}
	if config.gradleDependencySeed != "" {
		_, _ = fmt.Fprintln(progress, "buildopt: binding resolved Gradle dependencies in place for both isolated arms")
		binding, err := hashMeasurementDependencyTree(config.gradleDependencySeed)
		if err != nil {
			return config, fmt.Errorf("bind resolved Gradle dependencies: %w", err)
		}
		config.gradleReadOnlyDependencyRoot = filepath.Dir(config.gradleDependencySeed)
		config.gradleDependencySHA256 = binding.sha256
		config.gradleDependencyFileCount = binding.fileCount
		config.gradleDependencyByteCount = binding.byteCount
		_, _ = fmt.Fprintf(progress, "buildopt: shared read-only dependency binding ready (%d files, %d bytes, sha256 %s)\n",
			binding.fileCount, binding.byteCount, binding.sha256)
	}
	if config.gradleNativeBuildCacheSeed != "" {
		sharedBuildCache := filepath.Join(root, "native-build-cache-seed")
		_, _ = fmt.Fprintln(progress, "buildopt: snapshotting the native Gradle build cache once for both isolated arms")
		fileCount, byteCount, err := copyMeasurementDependencyTree(config.gradleNativeBuildCacheSeed, sharedBuildCache)
		if err != nil {
			return config, fmt.Errorf("snapshot native Gradle build cache: %w", err)
		}
		config.gradleSharedBuildCacheSeed = sharedBuildCache
		_, _ = fmt.Fprintf(progress, "buildopt: shared immutable native build-cache seed ready (%d files, %d bytes)\n", fileCount, byteCount)
	}
	return config, nil
}

func hashMeasurementDependencyTree(source string) (measurementDependencyBinding, error) {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return measurementDependencyBinding{}, errors.New("Gradle dependency cache is unavailable")
	}
	aggregate := sha256.New()
	binding := measurementDependencyBinding{}
	err = filepath.WalkDir(source, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("Gradle dependency cache contains a symlink")
		}
		if entry.IsDir() {
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return errors.New("Gradle dependency cache contains a non-regular entry")
		}
		if entryInfo.Name() == "gc.properties" || strings.HasSuffix(entryInfo.Name(), ".lock") {
			return nil
		}
		relative, err := filepath.Rel(source, candidate)
		if err != nil {
			return err
		}
		fileDigest, err := hashMeasurementFile(candidate)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(aggregate, "%s\x00%d\x00%s\n", filepath.ToSlash(relative), entryInfo.Size(), fileDigest)
		binding.fileCount++
		binding.byteCount += entryInfo.Size()
		return nil
	})
	if err != nil {
		return measurementDependencyBinding{}, err
	}
	binding.sha256 = hex.EncodeToString(aggregate.Sum(nil))
	return binding, nil
}

func verifyStructuralDependencyBinding(config structuralMeasurementConfig) error {
	if config.gradleDependencySeed == "" {
		return nil
	}
	binding, err := hashMeasurementDependencyTree(config.gradleDependencySeed)
	if err != nil {
		return fmt.Errorf("verify resolved Gradle dependencies: %w", err)
	}
	if binding.sha256 != config.gradleDependencySHA256 ||
		binding.fileCount != config.gradleDependencyFileCount ||
		binding.byteCount != config.gradleDependencyByteCount {
		return errors.New("resolved Gradle dependency cache changed during measurement")
	}
	return nil
}

func copyMeasurementDependencyTree(source, target string) (int, int64, error) {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return 0, 0, errors.New("Gradle cache seed is unavailable")
	}
	directories := make([]string, 0, 64)
	fileCount := 0
	var byteCount int64
	err = filepath.WalkDir(source, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, candidate)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("Gradle cache seed contains a symlink")
		}
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o700); err != nil {
				return err
			}
			directories = append(directories, destination)
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return errors.New("Gradle cache seed contains a non-regular entry")
		}
		if entryInfo.Name() == "gc.properties" || strings.HasSuffix(entryInfo.Name(), ".lock") {
			return nil
		}
		if err := copyMeasurementRegularFileWithMode(candidate, destination, 0o400); err != nil {
			return err
		}
		fileCount++
		byteCount += entryInfo.Size()
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Chmod(directories[index], 0o500); err != nil {
			return 0, 0, err
		}
	}
	return fileCount, byteCount, nil
}

func cleanupStructuralMeasurementRoot(root string) {
	for _, immutableRoot := range []string{
		filepath.Join(root, "readonly-dependencies"),
		filepath.Join(root, "native-build-cache-seed"),
	} {
		_ = filepath.WalkDir(immutableRoot, func(candidate string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.IsDir() {
				_ = os.Chmod(candidate, 0o700)
			} else if info, err := entry.Info(); err == nil && info.Mode().IsRegular() {
				_ = os.Chmod(candidate, 0o600)
			}
			return nil
		})
	}
	for attempt := 0; attempt < 3; attempt++ {
		if err := os.RemoveAll(root); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// copyMeasurementDistributionTree keeps each arm private while moving the
// already verified Wrapper distribution out of the network-dependent preflight.
// Only the executable bit is preserved; broader source permissions are not.
func copyMeasurementDistributionTree(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Gradle distribution seed is unavailable")
	}
	return filepath.WalkDir(source, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, candidate)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		entryInfo, err := entry.Info()
		if err != nil || !entryInfo.Mode().IsRegular() {
			return errors.New("Gradle distribution seed contains a non-regular entry")
		}
		mode := fs.FileMode(0o600)
		if entryInfo.Mode().Perm()&0o111 != 0 {
			mode = 0o700
		}
		return copyMeasurementRegularFileWithMode(candidate, destination, mode)
	})
}

func copyMeasurementRegularFile(source, target string) error {
	return copyMeasurementRegularFileWithMode(source, target, 0o600)
}

func copyMeasurementRegularFileWithMode(source, target string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = os.Remove(target)
		return err
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(target)
		return err
	}
	return nil
}

func hashMeasurementFile(candidate string) (string, error) {
	file, err := os.Open(candidate)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func gitOutput(directory string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if directory != "" {
		cmd.Dir = directory
	}
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(raw)))
	}
	return string(raw), nil
}

func gitRun(directory string, args ...string) error {
	_, err := gitOutput(directory, args...)
	return err
}

func writeRepositoryDocument(repositoryRoot, relativePath string, raw []byte, mode fs.FileMode) error {
	if relativePath == "" || filepath.IsAbs(relativePath) || filepath.Clean(relativePath) != relativePath || relativePath == "." || relativePath == ".." {
		return errors.New("output must be clean and repository relative")
	}
	path := filepath.Join(repositoryRoot, relativePath)
	parent := filepath.Dir(path)
	parentRelative, err := filepath.Rel(repositoryRoot, parent)
	if err != nil || parentRelative == ".." || strings.HasPrefix(parentRelative, ".."+string(filepath.Separator)) || filepath.IsAbs(parentRelative) {
		return errors.New("output directory must be an existing repository directory without symlinks")
	}
	current := repositoryRoot
	if parentRelative != "." {
		for _, segment := range strings.Split(parentRelative, string(filepath.Separator)) {
			current = filepath.Join(current, segment)
			info, inspectErr := os.Lstat(current)
			if inspectErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("output directory must be an existing repository directory without symlinks")
			}
		}
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("output must not be a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect output: %w", err)
	}
	temporary, err := os.CreateTemp(parent, ".buildopt-measurement-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}

func measurementFallbackReason(log string) string {
	const marker = "Build Impact POC retained the full graph ("
	start := strings.Index(log, marker)
	if start < 0 {
		return ""
	}
	remainder := log[start+len(marker):]
	end := strings.IndexByte(remainder, ')')
	if end <= 0 {
		return ""
	}
	return remainder[:end]
}

func tailMeasurementLog(log string, maximumLines int) string {
	lines := nonEmptyLines(log)
	if len(lines) > maximumLines {
		lines = lines[len(lines)-maximumLines:]
	}
	return strings.Join(lines, "\n")
}

func nonEmptyLines(raw string) []string {
	lines := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func nullDelimitedPaths(raw string) []string {
	if raw == "" {
		return nil
	}
	raw = strings.TrimSuffix(raw, "\x00")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\x00")
}

func sameMeasurementStrings(left, right []string) bool {
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

func uniqueMeasurementStrings(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func validMeasurementRevision(value string) bool {
	if len(value) != 40 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
