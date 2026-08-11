package launcher

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	profileMeasureUsage           = "usage: buildopt profile measure --manifest PATH --graph PATH --generated-manifest PATH --changes-file PATH --fallback-changes-file PATH --base-revision REVISION --buildopt-revision REVISION --evidence-output PATH [--gradle-option VALUE ...] [--target-stability-confirmations 1|2] [--timeout DURATION]\n"
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
	targetStabilityConfirmations int
}

type structuralMeasurementArm struct {
	name       string
	workspace  string
	gradleHome string
	cacheSeed  string
	warmups    []profilediscovery.StructuralWarmupObservation
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
	timeout := flags.Duration("timeout", 20*time.Minute, "per-build timeout")
	targetStabilityConfirmations := flags.Int("target-stability-confirmations", 1, "target-workload warm-ups required before measured pairs")
	var gradleOptions repeatedStringFlag
	flags.Var(&gradleOptions, "gradle-option", "Gradle option shared by both arms; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 ||
		*manifest == "" || *graph == "" || *generated == "" || *changes == "" ||
		*fallbackChanges == "" || *baseRevision == "" || *buildOptRevision == "" ||
		*evidenceOutput == "" || *timeout <= 0 ||
		(*targetStabilityConfirmations != 1 && *targetStabilityConfirmations != 2) {
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
		*targetStabilityConfirmations,
	)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile measurement unavailable: %v\n", err)
		return exitConfiguration
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

func prepareStructuralMeasurementConfig(
	manifest, graph, generated, changes, fallbackChanges, baseRevision,
	buildOptRevision, evidenceOutput string,
	gradleOptions []string,
	timeout time.Duration,
	targetStabilityConfirmations int,
) (structuralMeasurementConfig, error) {
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		return structuralMeasurementConfig{}, err
	}
	inputPaths := map[string]string{
		"manifest": manifest, "graph": graph, "generated manifest": generated,
		"changes file": changes, "fallback changes file": fallbackChanges,
	}
	inputDocuments := make(map[string][]byte, len(inputPaths))
	for label, candidate := range inputPaths {
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
	if !validMeasurementRevision(baseRevision) || !validMeasurementRevision(buildOptRevision) {
		return structuralMeasurementConfig{}, errors.New("base and BuildOpt revisions must be lowercase 40-character Git revisions")
	}
	if len(gradleOptions) == 0 {
		return structuralMeasurementConfig{}, errors.New("at least one Gradle option is required")
	}
	if targetStabilityConfirmations != 1 && targetStabilityConfirmations != 2 {
		return structuralMeasurementConfig{}, errors.New("target stability confirmations must be one or two")
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
	return structuralMeasurementConfig{
		repositoryRoot: repositoryRoot, manifestPath: manifest, graphPath: graph,
		generatedPath: generated, changesPath: changes,
		fallbackChangesPath: fallbackChanges, baseRevision: baseRevision,
		targetRevision: targetRevision, buildOptRevision: buildOptRevision,
		evidenceOutput: evidenceOutput, gradleOptions: gradleOptions,
		timeout: timeout, analysis: analysis, executable: executable,
		executableSHA256:             executableSHA256,
		inputDocuments:               inputDocuments,
		targetStabilityConfirmations: targetStabilityConfirmations,
	}, nil
}

func measureStructuralProfile(config structuralMeasurementConfig, progress io.Writer) ([]byte, bool, error) {
	root, err := os.MkdirTemp("", "buildopt-structural-measurement-*")
	if err != nil {
		return nil, false, fmt.Errorf("create isolated measurement root: %w", err)
	}
	defer os.RemoveAll(root)
	control, err := prepareStructuralMeasurementArm(config, root, "control", false, progress)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = stopStructuralMeasurementArm(control) }()
	candidate, err := prepareStructuralMeasurementArm(config, root, "candidate", true, progress)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = stopStructuralMeasurementArm(candidate) }()
	observations := make([]profilediscovery.StructuralMeasurementObservation, 0, measurementPairs)
	stableOutputSHA := ""
	stableOutputCount := 0
	for pair := 1; pair <= measurementPairs; pair++ {
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
			controlResult, err = runStructuralArm(config, control, false, config.changesPath)
			if err == nil {
				candidateResult, err = runStructuralArm(config, candidate, true, config.changesPath)
			}
		} else {
			candidateResult, err = runStructuralArm(config, candidate, true, config.changesPath)
			if err == nil {
				controlResult, err = runStructuralArm(config, control, false, config.changesPath)
			}
		}
		if err != nil {
			return nil, false, fmt.Errorf("pair %d: %w", pair, err)
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
		controlResult.outputSHA, controlResult.outputCount, err = hashMeasurementOutputs(control.workspace, config.analysis.Plan.RequiredOutputs)
		if err != nil {
			return nil, false, fmt.Errorf("pair %d verify control outputs: %w", pair, err)
		}
		candidateResult.outputSHA, candidateResult.outputCount, err = hashMeasurementOutputs(candidate.workspace, config.analysis.Plan.RequiredOutputs)
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
	// Release both hot measurement daemons before running the full graph so a
	// large repository cannot combine three JVM heaps and exhaust the host.
	if err := stopStructuralMeasurementArm(control); err != nil {
		return nil, false, fmt.Errorf("stop control daemon before fallback: %w", err)
	}
	if err := stopStructuralMeasurementArm(candidate); err != nil {
		return nil, false, fmt.Errorf("stop candidate daemon before fallback: %w", err)
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
	return profilediscovery.RenderStructuralMeasurementEvidence(profilediscovery.StructuralMeasurementOptions{
		CapturedAt: time.Now(), Analysis: config.analysis,
		RepositoryRevision:   config.targetRevision,
		BuildOptRevision:     config.buildOptRevision,
		ExecutableSHA256:     config.executableSHA256,
		SourceEvidenceSHA256: hex.EncodeToString(changesSHA[:]),
		GradleOptions:        config.gradleOptions, Observations: observations,
		ControlWarmups: control.warmups, CandidateWarmups: candidate.warmups,
		FallbackReason: fallbackReason, FallbackSuccessful: true,
	})
}

func prepareStructuralMeasurementArm(config structuralMeasurementConfig, root, name string, candidate bool, progress io.Writer) (structuralMeasurementArm, error) {
	totalWarmups := 2 + config.targetStabilityConfirmations
	arm := structuralMeasurementArm{
		name:       name,
		workspace:  filepath.Join(root, name+"-repository"),
		gradleHome: filepath.Join(root, name+"-gradle-home"),
		cacheSeed:  filepath.Join(root, name+"-build-cache-seed"),
	}
	if err := gitRun("", "clone", "--quiet", "--no-checkout", "--shared", "--", config.repositoryRoot, arm.workspace); err != nil {
		return arm, fmt.Errorf("clone %s arm: %w", name, err)
	}
	if err := os.MkdirAll(arm.gradleHome, 0o700); err != nil {
		return arm, fmt.Errorf("create %s Gradle home: %w", name, err)
	}
	if err := resetStructuralArm(config, arm, config.baseRevision, false); err != nil {
		return arm, fmt.Errorf("prepare %s baseline: %w", name, err)
	}
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s arm at %s (cache seed 1/%d)\n", name, config.baseRevision, totalWarmups)
	seedWarmup, err := runStructuralArm(config, arm, candidate, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("warm %s arm: %w", name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("CACHE_SEED", seedWarmup))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s arm cache seed in %dms with %s\n",
		name, seedWarmup.durationMS, formatStructuralTaskOutcomes(seedWarmup.taskOutcomes))
	cache := filepath.Join(arm.gradleHome, "caches", "build-cache-1")
	if err := copyMeasurementTree(cache, arm.cacheSeed); err != nil {
		return arm, fmt.Errorf("snapshot %s native build cache: %w", name, err)
	}
	if err := resetStructuralArm(config, arm, config.baseRevision, true); err != nil {
		return arm, fmt.Errorf("prepare %s daemon stabilization: %w", name, err)
	}
	_, _ = fmt.Fprintf(progress, "buildopt: warming isolated %s arm at %s (base daemon stabilization 2/%d)\n", name, config.baseRevision, totalWarmups)
	stabilizationWarmup, err := runStructuralArm(config, arm, candidate, config.changesPath)
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
	targetWarmup, err := runStructuralArm(config, arm, candidate, config.changesPath)
	if err != nil {
		return arm, fmt.Errorf("stabilize %s target workload: %w", name, err)
	}
	arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("TARGET_WORKLOAD_STABILIZATION", targetWarmup))
	_, _ = fmt.Fprintf(progress, "buildopt: warmed isolated %s arm target workload in %dms with %s\n",
		name, targetWarmup.durationMS, formatStructuralTaskOutcomes(targetWarmup.taskOutcomes))
	if config.targetStabilityConfirmations == 2 {
		if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
			return arm, fmt.Errorf("prepare %s target-workload stability confirmation: %w", name, err)
		}
		_, _ = fmt.Fprintf(progress, "buildopt: confirming isolated %s target-workload shape at %s (4/%d)\n", name, config.targetRevision, totalWarmups)
		confirmation, err := runStructuralArm(config, arm, candidate, config.changesPath)
		if err != nil {
			return arm, fmt.Errorf("confirm %s target workload: %w", name, err)
		}
		arm.warmups = append(arm.warmups, structuralWarmupDiagnostic("TARGET_WORKLOAD_STABILITY_CONFIRMATION", confirmation))
		_, _ = fmt.Fprintf(progress, "buildopt: confirmed isolated %s target workload in %dms with %s\n",
			name, confirmation.durationMS, formatStructuralTaskOutcomes(confirmation.taskOutcomes))
	}
	return arm, nil
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
	if err := resetStructuralArm(config, arm, config.targetRevision, true); err != nil {
		return structuralArmResult{}, "", err
	}
	fallbackConfig := config
	fallbackConfig.gradleOptions = structuralFallbackGradleOptions(config.gradleOptions)
	_, _ = fmt.Fprintln(progress, "buildopt: validating full-graph fallback with --no-daemon and measured scheduling")
	result, err := runStructuralArm(fallbackConfig, arm, true, config.fallbackChangesPath)
	if err != nil {
		return result, "", fmt.Errorf("full-graph fallback: %w", err)
	}
	reason := measurementFallbackReason(result.log)
	if reason == "" {
		return result, "", errors.New("installed fallback did not report full-graph execution")
	}
	result.outputSHA, result.outputCount, err = hashMeasurementOutputs(arm.workspace, config.analysis.Plan.RequiredOutputs)
	return result, reason, err
}

func structuralFallbackGradleOptions(measured []string) []string {
	options := make([]string, 0, len(measured)+1)
	for _, option := range measured {
		if option == "--daemon" || option == "--no-daemon" {
			continue
		}
		options = append(options, option)
	}
	return append(options, "--no-daemon")
}

func resetStructuralArm(config structuralMeasurementConfig, arm structuralMeasurementArm, revision string, restoreCache bool) error {
	if err := gitRun(arm.workspace, "reset", "--hard", "--quiet", revision); err != nil {
		return err
	}
	if err := gitRun(arm.workspace, "clean", "-ffdx", "--quiet"); err != nil {
		return err
	}
	if restoreCache {
		cache := filepath.Join(arm.gradleHome, "caches", "build-cache-1")
		if err := os.RemoveAll(cache); err != nil {
			return fmt.Errorf("clear isolated native build cache: %w", err)
		}
		if err := copyMeasurementTree(arm.cacheSeed, cache); err != nil {
			return fmt.Errorf("restore isolated native build cache: %w", err)
		}
	}
	for _, input := range []string{config.manifestPath, config.graphPath, config.generatedPath, config.changesPath, config.fallbackChangesPath} {
		if err := copyMeasurementInputDocument(arm.workspace, input, config.inputDocuments[input]); err != nil {
			return err
		}
	}
	return nil
}

func runStructuralArm(config structuralMeasurementConfig, arm structuralMeasurementArm, candidate bool, changesPath string) (structuralArmResult, error) {
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
	ctx, cancel := context.WithTimeout(context.Background(), config.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = arm.workspace
	cmd.Env = measurementEnvironment(arm.gradleHome)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	pressureBefore := readStructuralPressureSnapshot()
	started := time.Now()
	err := cmd.Run()
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
		return result, fmt.Errorf("%s arm produced invalid exact task evidence: %w", arm.name, validationErr)
	}
	if validationErr := profilediscovery.ValidateStructuralHostPressure(structuralHostPressurePointer(result.hostPressure)); validationErr != nil {
		return result, fmt.Errorf("%s arm produced invalid host-pressure evidence: %w", arm.name, validationErr)
	}
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("%s arm exceeded %s", arm.name, config.timeout)
	}
	if err != nil {
		return result, fmt.Errorf("%s arm failed: %w\n%s", arm.name, err, tailMeasurementLog(result.log, 80))
	}
	return result, nil
}

func summarizeStructuralTaskOutcomes(log string) profilediscovery.StructuralTaskOutcomes {
	var outcomes profilediscovery.StructuralTaskOutcomes
	var taskLines []string
	var tasks []profilediscovery.StructuralTaskObservation
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
		canonical := strings.Join(fields, " ")
		taskLines = append(taskLines, canonical)
		outcomes.Total++
		outcome := "EXECUTED"
		switch {
		case strings.HasSuffix(line, " FROM-CACHE"):
			outcomes.FromCache++
			outcome = "FROM_CACHE"
		case strings.HasSuffix(line, " UP-TO-DATE"):
			outcomes.UpToDate++
			outcome = "UP_TO_DATE"
		case strings.HasSuffix(line, " NO-SOURCE"):
			outcomes.NoSource++
			outcome = "NO_SOURCE"
		case strings.HasSuffix(line, " SKIPPED"):
			outcomes.Skipped++
			outcome = "SKIPPED"
		default:
			outcomes.Executed++
		}
		tasks = append(tasks, profilediscovery.StructuralTaskObservation{Path: fields[2], Outcome: outcome})
	}
	if len(taskLines) > 0 {
		sort.Strings(taskLines)
		sort.Slice(tasks, func(left, right int) bool { return tasks[left].Path < tasks[right].Path })
		digest := sha256.Sum256([]byte(strings.Join(taskLines, "\n") + "\n"))
		outcomes.FingerprintSHA256 = hex.EncodeToString(digest[:])
		outcomes.Tasks = tasks
	}
	return outcomes
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
	cmd.Env = measurementEnvironment(arm.gradleHome)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stop %s Gradle daemon: %w", arm.name, err)
	}
	return nil
}

func measurementEnvironment(gradleHome string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "BUILDOPT_") || name == "GRADLE_USER_HOME" {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "GRADLE_USER_HOME="+gradleHome)
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

func copyMeasurementTree(source, target string) error {
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
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
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("native build cache seed contains a non-regular entry")
		}
		return copyMeasurementRegularFile(candidate, destination)
	})
}

func copyMeasurementRegularFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
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
