package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/outputequivalence"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const profileProposalUsage = "usage: buildopt profile propose (--owner-input PATH [--changes-file PATH] | --repository-id OWNER/REPO --pipeline-class CLASS --entrypoint TASK [--entrypoint TASK ...] --required-output GLOB [--required-output GLOB ...] --changes-file PATH [--output-equivalence PATH]) --base-revision REVISION [--global-change GLOB ...] [--gradle-command PATH] [--gradle-option VALUE ...] [--discovery-cache-dir ABSOLUTE_PATH] [--output-contract-output PATH] [--manifest-output PATH] [--graph-output PATH] [--generated-manifest-output PATH] [--fallback-changes-output PATH] [--proposal-output PATH] [--buildopt-revision REVISION] [--timeout DURATION]\n"

const maximumStructuralAlternativeEntrypoints = 64

var defaultProposalGlobalChanges = []string{
	"build-logic/**",
	"build.gradle",
	"build.gradle.kts",
	"buildSrc/**",
	"gradle.properties",
	"gradle/**",
	"settings.gradle",
	"settings.gradle.kts",
}

type profileProposalDocuments struct {
	OutputContract  string `json:"outputContract"`
	Manifest        string `json:"manifest,omitempty"`
	Graph           string `json:"graph,omitempty"`
	Generated       string `json:"generatedManifest,omitempty"`
	FallbackChanges string `json:"fallbackChanges,omitempty"`
	Proposal        string `json:"proposal"`
}

type profileProposalReport struct {
	repositoryRoot          string                           `json:"-"`
	cacheHit                bool                             `json:"-"`
	cacheKey                string                           `json:"-"`
	SchemaVersion           string                           `json:"schemaVersion"`
	Decision                string                           `json:"decision"`
	Reason                  string                           `json:"reason"`
	RepositoryID            string                           `json:"repositoryId"`
	PipelineClass           string                           `json:"pipelineClass"`
	BaseRevision            string                           `json:"baseRevision"`
	TargetRevision          string                           `json:"targetRevision"`
	OriginalEntrypoint      string                           `json:"originalEntrypoint,omitempty"`
	OriginalEntrypoints     []string                         `json:"originalEntrypoints"`
	ChangedPaths            []string                         `json:"changedPaths"`
	RequiredOutputs         []string                         `json:"requiredOutputs"`
	GlobalChangePaths       []string                         `json:"globalChangePaths"`
	CandidateEntrypoints    []string                         `json:"candidateEntrypoints,omitempty"`
	OmittedProjects         []string                         `json:"omittedProjects,omitempty"`
	UnknownRelationships    bool                             `json:"unknownRelationships"`
	Analysis                *profilediscovery.AnalysisReport `json:"analysis,omitempty"`
	Documents               profileProposalDocuments         `json:"documents"`
	MeasureCommand          []string                         `json:"measureCommand,omitempty"`
	BuildOptRevisionNeeded  bool                             `json:"buildOptRevisionNeeded"`
	ReviewRequired          bool                             `json:"reviewRequired"`
	ActivationAutomatic     bool                             `json:"activationAutomatic"`
	ProductionAuthorized    bool                             `json:"productionAuthorized"`
	TestOptimization        string                           `json:"testOptimization"`
	OwnerInput              string                           `json:"ownerInput,omitempty"`
	OwnerInputSHA256        string                           `json:"ownerInputSha256,omitempty"`
	OutputEquivalence       string                           `json:"outputEquivalence,omitempty"`
	OutputEquivalenceSHA256 string                           `json:"outputEquivalenceSha256,omitempty"`
	ChangeSource            string                           `json:"changeSource,omitempty"`
}

type proposalStringFlag []string

func (values *proposalStringFlag) String() string { return strings.Join(*values, ",") }
func (values *proposalStringFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func runStructuralProfileProposal(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileProposalUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile propose", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repositoryID := flags.String("repository-id", "", "owner/repository identity")
	pipelineClass := flags.String("pipeline-class", "", "pipeline class")
	ownerInputPath := flags.String("owner-input", "", "checked versioned owner input")
	outputEquivalence := flags.String("output-equivalence", "", "owner-reviewed semantic output-equivalence contract")
	var entrypoints proposalStringFlag
	flags.Var(&entrypoints, "entrypoint", "original Gradle task selector; repeat for multi-entrypoint workflows")
	changesFile := flags.String("changes-file", "", "exact base-to-target changed paths")
	baseRevision := flags.String("base-revision", "", "immutable baseline Git revision")
	gradleCommand := flags.String("gradle-command", "", "repository-relative or absolute Gradle command")
	manifestOutput := flags.String("manifest-output", "buildopt-impact-manifest.json", "reviewable manifest output")
	outputContractOutput := flags.String("output-contract-output", "buildopt-output-contract.json", "validated output-contract artifact")
	graphOutput := flags.String("graph-output", "buildopt-impact-graph.generated.json", "reviewable graph output")
	generatedOutput := flags.String("generated-manifest-output", "buildopt-impact.generated.json", "generated-state binding output")
	fallbackOutput := flags.String("fallback-changes-output", "buildopt-fallback-changes.txt", "full-graph fallback input")
	proposalOutput := flags.String("proposal-output", "buildopt-profile-proposal.json", "reviewable proposal output")
	buildOptRevision := flags.String("buildopt-revision", "", "immutable BuildOpt revision for the next measure command")
	discoveryCacheDir := flags.String("discovery-cache-dir", "", "absolute private directory for exact proposal replay")
	timeout := flags.Duration("timeout", 0, "Gradle discovery timeout")
	var requiredOutputs proposalStringFlag
	var globalChanges proposalStringFlag
	var gradleOptions repeatedStringFlag
	flags.Var(&requiredOutputs, "required-output", "repository-owned output glob; repeat for multiple outputs")
	flags.Var(&globalChanges, "global-change", "full-graph fallback glob; repeat to replace defaults")
	flags.Var(&gradleOptions, "gradle-option", "Gradle discovery option; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *baseRevision == "" {
		_, _ = io.WriteString(stderr, profileProposalUsage)
		return exitUsage
	}
	ownerInputDigest := ""
	ownerChangeSource := ""
	if *ownerInputPath != "" {
		if *repositoryID != "" || *pipelineClass != "" || len(entrypoints) != 0 || len(requiredOutputs) != 0 || *outputEquivalence != "" || len(globalChanges) != 0 || *gradleCommand != "" || len(gradleOptions) != 0 || *timeout != 0 {
			_, _ = io.WriteString(stderr, profileProposalUsage)
			return exitUsage
		}
		root, err := canonicalWorkingDirectory()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
			return exitConfiguration
		}
		input, digest, err := readProfileOwnerInput(root, *ownerInputPath)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
			return exitConfiguration
		}
		*repositoryID, *pipelineClass = input.RepositoryID, input.PipelineClass
		entrypoints = append(entrypoints, input.Entrypoints...)
		requiredOutputs = append(requiredOutputs, input.RequiredOutputs...)
		globalChanges = append(globalChanges, input.GlobalChanges...)
		*gradleCommand = input.GradleCommand
		gradleOptions = append(gradleOptions, input.GradleOptions...)
		*timeout = time.Duration(input.TimeoutMinutes) * time.Minute
		ownerInputDigest = digest
		ownerChangeSource = input.ChangeSource
		if input.OutputEquivalence != nil {
			*outputEquivalence = input.OutputEquivalence.Path
		}
	} else {
		if *repositoryID == "" || *pipelineClass == "" || len(entrypoints) == 0 || len(requiredOutputs) == 0 || *changesFile == "" {
			_, _ = io.WriteString(stderr, profileProposalUsage)
			return exitUsage
		}
		if *timeout == 0 {
			*timeout = 5 * time.Minute
		}
	}
	if *timeout <= 0 {
		_, _ = io.WriteString(stderr, profileProposalUsage)
		return exitUsage
	}
	if len(globalChanges) == 0 {
		globalChanges = append(globalChanges, defaultProposalGlobalChanges...)
	}
	report, documents, err := prepareStructuralProfileProposal(context.Background(), structuralProposalConfig{
		repositoryID: *repositoryID, pipelineClass: *pipelineClass,
		entrypoints: append([]string(nil), entrypoints...), changesFile: *changesFile, baseRevision: *baseRevision,
		requiredOutputs: append([]string(nil), requiredOutputs...),
		globalChanges:   append([]string(nil), globalChanges...),
		gradleCommand:   *gradleCommand, gradleOptions: append([]string(nil), gradleOptions...),
		outputContractOutput: *outputContractOutput,
		manifestOutput:       *manifestOutput, graphOutput: *graphOutput,
		generatedOutput: *generatedOutput, fallbackOutput: *fallbackOutput,
		proposalOutput: *proposalOutput, buildOptRevision: *buildOptRevision,
		ownerInput: *ownerInputPath, ownerInputSHA256: ownerInputDigest,
		outputEquivalence: *outputEquivalence,
		changeSource:      ownerChangeSource,
		discoveryCacheDir: *discoveryCacheDir,
		timeout:           *timeout,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
		return exitConfiguration
	}
	if report.cacheHit {
		_, _ = fmt.Fprintf(stderr, "buildopt: exact structural proposal replay %s\n", report.cacheKey)
	}
	for _, output := range []string{report.Documents.OutputContract, report.Documents.Manifest, report.Documents.Graph, report.Documents.Generated, report.Documents.FallbackChanges} {
		raw, ok := documents[output]
		if output == "" || !ok {
			continue
		}
		if err := writeRepositoryDocument(report.repositoryRoot, output, raw, 0o644); err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
			return exitConfiguration
		}
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
		return exitConfiguration
	}
	raw = append(raw, '\n')
	if err := writeRepositoryDocument(report.repositoryRoot, report.Documents.Proposal, raw, 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write structural profile proposal: %v\n", err)
		return exitConfiguration
	}
	return 0
}

type structuralProposalConfig struct {
	repositoryID, pipelineClass, changesFile, baseRevision           string
	entrypoints                                                      []string
	requiredOutputs, globalChanges, gradleOptions                    []string
	gradleCommand, outputContractOutput, manifestOutput, graphOutput string
	generatedOutput                                                  string
	fallbackOutput, proposalOutput, buildOptRevision                 string
	ownerInput, ownerInputSHA256                                     string
	outputEquivalence                                                string
	changeSource                                                     string
	discoveryCacheDir                                                string
	timeout                                                          time.Duration
	observedOutputSnapshot                                           *outputContractSnapshot
	observedImpactSnapshot                                           *buildimpact.DiscoverySnapshot
	candidateOwnerProjects                                           map[string]bool
}

func prepareStructuralProfileProposal(ctx context.Context, config structuralProposalConfig) (profileProposalReport, map[string][]byte, error) {
	root, err := canonicalWorkingDirectory()
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	if len(config.entrypoints) > 256 || !uniqueMeasurementStrings(config.entrypoints) {
		return profileProposalReport{}, nil, errors.New("proposal entrypoints must be unique and bounded")
	}
	selectors, err := proposalTerminalSelectors(config.entrypoints)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	if len(config.requiredOutputs) > 256 || len(config.globalChanges) > 256 || !uniqueMeasurementStrings(config.requiredOutputs) || !uniqueMeasurementStrings(config.globalChanges) {
		return profileProposalReport{}, nil, errors.New("proposal outputs and global changes must be unique and bounded")
	}
	if len(config.gradleOptions) > 32 || !uniqueMeasurementStrings(config.gradleOptions) {
		return profileProposalReport{}, nil, errors.New("Gradle discovery options must be unique and bounded")
	}
	outputEquivalenceSHA256 := ""
	if config.outputEquivalence != "" {
		raw, err := readRepositoryRegularDocument(root, config.outputEquivalence, maximumOwnerInputBytes)
		if err != nil {
			return profileProposalReport{}, nil, err
		}
		if _, err := outputequivalence.Parse(raw); err != nil {
			return profileProposalReport{}, nil, err
		}
		outputEquivalenceSHA256 = outputequivalence.SHA256(raw)
	}
	if err := validateProposalOutputs(config); err != nil {
		return profileProposalReport{}, nil, err
	}
	if !validMeasurementRevision(config.baseRevision) || (config.buildOptRevision != "" && !validMeasurementRevision(config.buildOptRevision)) {
		return profileProposalReport{}, nil, errors.New("base and optional BuildOpt revisions must be lowercase 40-character Git revisions")
	}
	targetRevision, err := proposalGitTarget(root, config.baseRevision)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	var changedPaths []string
	if config.changesFile != "" {
		changedPaths, err = readImpactChangedPaths(root, config.changesFile)
		if err == nil {
			err = validateMeasurementChangeSet(root, config.baseRevision, targetRevision, changedPaths)
		}
	} else if config.changeSource == "GIT_DIFF_BASE_TO_HEAD" {
		changedPaths, err = proposalGitChangedPaths(root, config.baseRevision, targetRevision)
	} else {
		err = errors.New("proposal requires an exact change source")
	}
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	sort.Strings(changedPaths)
	sort.Strings(config.requiredOutputs)
	sort.Strings(config.globalChanges)
	report := nativeProfileProposal(config, targetRevision, changedPaths)
	report.OutputEquivalence = config.outputEquivalence
	report.OutputEquivalenceSHA256 = outputEquivalenceSHA256
	report.repositoryRoot = root
	var cacheBinding profileProposalCacheBinding
	if config.discoveryCacheDir != "" {
		cacheBinding, err = prepareProfileProposalCacheBinding(root, targetRevision, changedPaths, config, outputEquivalenceSHA256)
		if err != nil {
			return profileProposalReport{}, nil, err
		}
		cachedReport, cachedDocuments, hit, cacheErr := loadProfileProposalCache(config.discoveryCacheDir, cacheBinding, config)
		if cacheErr != nil {
			return profileProposalReport{}, nil, cacheErr
		}
		if hit {
			cachedReport.repositoryRoot = root
			cachedReport.cacheHit = true
			cachedReport.cacheKey = cacheBinding.Digest
			return cachedReport, cachedDocuments, nil
		}
	}
	outputConfig := outputContractConfig{
		repositoryID: config.repositoryID, pipelineClass: config.pipelineClass,
		repositoryRevision: targetRevision,
		entrypoints:        append([]string(nil), config.entrypoints...),
		requiredOutputs:    append([]string(nil), config.requiredOutputs...),
		gradleCommand:      config.gradleCommand,
		gradleOptions:      append([]string(nil), config.gradleOptions...),
		timeout:            config.timeout,
	}
	var outputReport outputContractReport
	if config.observedOutputSnapshot != nil {
		outputReport, err = prepareOutputContractReportFromSnapshot(root, outputConfig, *config.observedOutputSnapshot)
	} else {
		outputReport, err = prepareOutputContract(ctx, root, outputConfig)
	}
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	outputRaw, err := renderOutputContract(outputReport)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	documents := map[string][]byte{config.outputContractOutput: outputRaw}
	report.Documents.OutputContract = config.outputContractOutput
	if outputReport.Decision != "VALIDATED_REQUIRED_OUTPUTS" {
		report.Reason = outputReport.Reason
		return report, documents, nil
	}

	for _, changedPath := range changedPaths {
		if matchesAnyProposalGlob(config.globalChanges, changedPath) {
			report.Reason = "GLOBAL_CHANGE_REQUIRES_FULL_GRAPH"
			return report, documents, nil
		}
	}
	candidateEntrypoints := proposalOutputOwnerEntrypointsForProjects(
		outputReport,
		selectors,
		config.candidateOwnerProjects,
	)
	report.CandidateEntrypoints = candidateEntrypoints
	if len(candidateEntrypoints) > maximumStructuralAlternativeEntrypoints {
		report.Reason = "CANDIDATE_TASK_SET_TOO_LARGE"
		return report, documents, nil
	}

	manifest := buildimpact.Manifest{
		SchemaVersion: buildimpact.ManifestSchemaVersion, ManifestVersion: 1,
		RepositoryID: config.repositoryID, PipelineClass: config.pipelineClass,
		Ownership:           buildimpact.RepositoryOwnership,
		OriginalEntrypoints: append([]string(nil), config.entrypoints...),
		AllowedAlternatives: []buildimpact.EntrypointSet{{ID: "changed-projects", Entrypoints: candidateEntrypoints}},
		RequiredChecks:      []buildimpact.Check{}, GlobalChangePaths: config.globalChanges,
		UnknownChangePolicy: buildimpact.FullGraphPolicy,
	}
	for index, output := range config.requiredOutputs {
		manifest.RequiredArtifacts = append(manifest.RequiredArtifacts, buildimpact.Artifact{
			ID: fmt.Sprintf("required-output-%d", index+1), Path: output, Owner: buildimpact.BuildOptimization,
		})
	}
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	manifestRaw = append(manifestRaw, '\n')
	loadedManifest, err := buildimpact.ParseManifest(manifestRaw, config.repositoryID, config.pipelineClass)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	finalContext, finalCancel := context.WithTimeout(ctx, config.timeout)
	defer finalCancel()
	var generated buildimpact.GeneratedImpact
	if config.observedImpactSnapshot != nil {
		candidateSnapshot, deriveErr := buildimpact.DeriveProjectEntrypoints(
			*config.observedImpactSnapshot,
			candidateEntrypoints,
		)
		if deriveErr != nil {
			err = deriveErr
		} else {
			var snapshotRaw []byte
			snapshotRaw, err = json.Marshal(candidateSnapshot)
			if err == nil {
				generated, err = buildimpact.GenerateImpact(loadedManifest, snapshotRaw)
			}
		}
	} else {
		generated, err = buildimpact.DiscoverWithManifest(finalContext, buildimpact.DiscoveryOptions{
			RepositoryRoot: root, GradleCommand: config.gradleCommand, GradleArgs: config.gradleOptions,
		}, loadedManifest)
	}
	if err != nil {
		report.Reason = "CANDIDATE_WORKFLOW_UNSUPPORTED"
		return report, documents, nil
	}
	snapshot := generated.Snapshot
	report.UnknownRelationships = !snapshot.Complete
	originalUnknown := false
	originalSet := make(map[string]bool, len(config.entrypoints))
	for _, entrypoint := range config.entrypoints {
		originalSet[entrypoint] = true
	}
	for _, entrypoint := range snapshot.Entrypoints {
		if entrypoint.UnknownRelationships {
			report.UnknownRelationships = true
			originalUnknown = originalUnknown || originalSet[entrypoint.Name]
		}
	}
	if !snapshot.Complete || report.UnknownRelationships {
		if originalUnknown {
			report.Reason = "ORIGINAL_WORKFLOW_UNSUPPORTED"
		} else {
			report.Reason = "CANDIDATE_GRAPH_INCOMPLETE"
		}
		return report, documents, nil
	}
	if _, err := buildimpact.ResolveProjectOwners(snapshot, changedPaths); err != nil {
		report.Reason = "SOURCE_OWNERSHIP_AMBIGUOUS"
		return report, documents, nil
	}
	if !generated.Generated.Complete {
		report.Reason = "CANDIDATE_GRAPH_INCOMPLETE"
		report.UnknownRelationships = true
		return report, documents, nil
	}
	fallbackChange, err := proposalFallbackChange(config.globalChanges)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	fallbackRaw := []byte(fallbackChange + "\n")
	documents[config.manifestOutput] = manifestRaw
	documents[config.graphOutput] = generated.GraphJSON
	documents[config.generatedOutput] = generated.GeneratedJSON
	documents[config.fallbackOutput] = fallbackRaw
	analysis := profilediscovery.AnalyzeGeneratedOpportunity(loadedManifest, generated.Graph, generated.Generated)
	report.Analysis = &analysis
	report.Decision = analysis.Decision
	report.Reason = analysis.Reason
	report.UnknownRelationships = false
	if analysis.Decision != profilediscovery.DecisionMeasure {
		return report, documents, nil
	}
	report.Documents.Manifest = config.manifestOutput
	report.Documents.Graph = config.graphOutput
	report.Documents.Generated = config.generatedOutput
	report.Documents.FallbackChanges = config.fallbackOutput
	report.OmittedProjects = proposalOmittedProjects(generated.Graph.Graph, candidateEntrypoints)
	report.MeasureCommand = proposalMeasureCommand(config)
	report.BuildOptRevisionNeeded = config.buildOptRevision == ""
	if config.discoveryCacheDir != "" {
		if err := storeProfileProposalCache(config.discoveryCacheDir, cacheBinding, config, report, documents, generated.Snapshot); err != nil {
			return profileProposalReport{}, nil, err
		}
	}
	return report, documents, nil
}

// proposalOutputOwnerProjects returns the reviewed owners of the declared
// outputs. Candidate lifecycle tasks are rooted at these projects rather than
// at the changed source owner: a shared dependency change must still rebuild
// the project that produces the requested output. Graph discovery subsequently
// proves that every candidate covers both the change and the declared output.
func proposalOutputOwnerProjects(report outputContractReport) []string {
	owners := map[string]bool{}
	for _, validation := range report.Validations {
		for _, owner := range validation.OwnerProjects {
			owners[owner] = true
		}
	}
	result := make([]string, 0, len(owners))
	for owner := range owners {
		result = append(result, owner)
	}
	sort.Strings(result)
	return result
}

func proposalOutputOwnerEntrypoints(report outputContractReport, selectors []string) []string {
	return proposalOutputOwnerEntrypointsForProjects(report, selectors, nil)
}

func proposalOutputOwnerEntrypointsForProjects(
	report outputContractReport,
	selectors []string,
	included map[string]bool,
) []string {
	owners := proposalOutputOwnerProjects(report)
	entrypoints := make([]string, 0, len(owners)*len(selectors))
	for _, owner := range owners {
		if included != nil && !included[owner] {
			continue
		}
		for _, selector := range selectors {
			if !proposalOwnerHasSelectorOutput(report, owner, selector) {
				continue
			}
			if owner == ":" {
				entrypoints = append(entrypoints, ":"+selector)
			} else {
				entrypoints = append(entrypoints, owner+":"+selector)
			}
		}
	}
	sort.Strings(entrypoints)
	return entrypoints
}

func proposalOwnerHasSelectorOutput(report outputContractReport, owner, selector string) bool {
	for _, candidate := range report.CandidateOutputs {
		if len(candidate.OwnerProjects) != 1 || candidate.OwnerProjects[0] != owner || candidate.FileCount < 1 {
			continue
		}
		for _, task := range candidate.ProducerTasks {
			name := task
			if index := strings.LastIndex(task, ":"); index >= 0 {
				name = task[index+1:]
			}
			if name == selector || optimizeLifecycleOutput(selector, candidate.Path) {
				return true
			}
		}
	}
	return false
}

func nativeProfileProposal(config structuralProposalConfig, targetRevision string, changedPaths []string) profileProposalReport {
	legacyEntrypoint := ""
	if len(config.entrypoints) == 1 {
		legacyEntrypoint = config.entrypoints[0]
	}
	return profileProposalReport{
		SchemaVersion: "buildopt.poc/profile-proposal/v1", Decision: "NATIVE_FULL_GRAPH",
		Reason: "NO_SAFE_STRUCTURAL_CANDIDATE", RepositoryID: config.repositoryID,
		PipelineClass: config.pipelineClass, BaseRevision: config.baseRevision,
		TargetRevision: targetRevision, OriginalEntrypoint: legacyEntrypoint,
		OriginalEntrypoints: append([]string(nil), config.entrypoints...),
		ChangedPaths:        changedPaths, RequiredOutputs: config.requiredOutputs,
		GlobalChangePaths: config.globalChanges,
		Documents:         profileProposalDocuments{OutputContract: config.outputContractOutput, Proposal: config.proposalOutput},
		ReviewRequired:    true, ActivationAutomatic: false, ProductionAuthorized: false,
		TestOptimization: "OUT_OF_SCOPE",
		OwnerInput:       config.ownerInput, OwnerInputSHA256: config.ownerInputSHA256,
		ChangeSource: config.changeSource,
	}
}

func proposalGitChangedPaths(root, baseRevision, targetRevision string) ([]string, error) {
	raw, err := gitOutput(root, "diff", "--name-only", "--no-renames", "-z", baseRevision, targetRevision, "--")
	if err != nil {
		return nil, err
	}
	paths := nullDelimitedPaths(raw)
	if len(paths) == 0 || len(paths) > maximumImpactChangedPaths || !uniqueMeasurementStrings(paths) {
		return nil, errors.New("Git change source must contain unique bounded paths")
	}
	for _, candidate := range paths {
		if !validObservedOutputPath(candidate) || strings.ContainsAny(candidate, "\r\n\x00") {
			return nil, errors.New("Git change source contains an unsafe path")
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func proposalTerminalSelectors(entrypoints []string) ([]string, error) {
	selectors := make([]string, 0, len(entrypoints))
	seen := make(map[string]bool, len(entrypoints))
	for _, entrypoint := range entrypoints {
		if entrypoint == "" || strings.ContainsAny(entrypoint, " /\\") {
			return nil, errors.New("proposal entrypoints must be Gradle task selectors")
		}
		selector := entrypoint
		if index := strings.LastIndex(entrypoint, ":"); index >= 0 {
			selector = entrypoint[index+1:]
		}
		if selector == "" || strings.Contains(selector, ":") {
			return nil, errors.New("proposal entrypoints must end in a task name")
		}
		if !seen[selector] {
			seen[selector] = true
			selectors = append(selectors, selector)
		}
	}
	sort.Strings(selectors)
	return selectors, nil
}

func proposalGitTarget(root, baseRevision string) (string, error) {
	if dirty, err := gitOutput(root, "status", "--porcelain", "--untracked-files=no"); err != nil {
		return "", err
	} else if strings.TrimSpace(dirty) != "" {
		return "", errors.New("tracked repository state must be clean before proposal discovery")
	}
	target, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	target = strings.TrimSpace(target)
	if !validMeasurementRevision(target) || target == baseRevision {
		return "", errors.New("proposal requires distinct immutable base and target revisions")
	}
	if resolved, err := gitOutput(root, "rev-parse", baseRevision+"^{commit}"); err != nil || strings.TrimSpace(resolved) != baseRevision {
		return "", errors.New("base revision is unavailable or ambiguous")
	}
	if err := gitRun(root, "merge-base", "--is-ancestor", baseRevision, target); err != nil {
		return "", errors.New("base revision must be an ancestor of target revision")
	}
	return target, nil
}

func validateProposalOutputs(config structuralProposalConfig) error {
	paths := []string{config.outputContractOutput, config.manifestOutput, config.graphOutput, config.generatedOutput, config.fallbackOutput, config.proposalOutput}
	seen := map[string]bool{}
	if config.changesFile != "" {
		seen[config.changesFile] = true
	}
	if config.ownerInput != "" {
		seen[config.ownerInput] = true
	}
	if config.outputEquivalence != "" {
		seen[config.outputEquivalence] = true
	}
	for _, candidate := range paths {
		if candidate == "" || filepath.IsAbs(candidate) || filepath.Clean(candidate) != candidate || candidate == "." || candidate == ".." || seen[candidate] {
			return errors.New("proposal outputs must be distinct clean repository-relative paths")
		}
		seen[candidate] = true
	}
	return nil
}

func matchesAnyProposalGlob(patterns []string, candidate string) bool {
	for _, pattern := range patterns {
		if matchProposalGlob(pattern, candidate) {
			return true
		}
	}
	return false
}

func matchProposalGlob(pattern, candidate string) bool {
	patternParts := strings.Split(pattern, "/")
	candidateParts := strings.Split(candidate, "/")
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

func proposalFallbackChange(patterns []string) (string, error) {
	for _, pattern := range patterns {
		if !strings.ContainsAny(pattern, "*?[") {
			return pattern, nil
		}
	}
	candidates := []string{
		"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts",
		"gradle.properties", "gradle/buildopt-global-change",
		"buildSrc/buildopt-global-change", "build-logic/buildopt-global-change",
	}
	for _, candidate := range candidates {
		if matchesAnyProposalGlob(patterns, candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("global-change policy needs at least one concrete fallback example")
}

func proposalMeasureCommand(config structuralProposalConfig) []string {
	revision := config.buildOptRevision
	if revision == "" {
		revision = "<BUILDOPT_REVISION>"
	}
	command := []string{
		"buildopt", "profile", "measure",
		"--manifest", config.manifestOutput,
		"--graph", config.graphOutput,
		"--generated-manifest", config.generatedOutput,
		"--changes-file", config.changesFile,
		"--fallback-changes-file", config.fallbackOutput,
		"--base-revision", config.baseRevision,
		"--buildopt-revision", revision,
		"--evidence-output", "buildopt-profile-evidence.json",
		"--target-stability-confirmations", "3",
		"--adaptive-candidate-stability",
	}
	if config.outputEquivalence != "" {
		command = append(command, "--output-equivalence", config.outputEquivalence)
	}
	return command
}

func proposalOmittedProjects(graph buildimpact.DeclaredGraph, candidates []string) []string {
	candidateSet := map[string]bool{}
	for _, candidate := range candidates {
		candidateSet[candidate] = true
	}
	reached := map[string]bool{}
	for _, entrypoint := range graph.Entrypoints {
		if !candidateSet[entrypoint.Name] {
			continue
		}
		for _, project := range entrypoint.ReachesProjects {
			reached[project] = true
		}
	}
	omitted := make([]string, 0)
	for _, project := range graph.Projects {
		if !reached[project.Path] {
			omitted = append(omitted, project.Path)
		}
	}
	sort.Strings(omitted)
	return omitted
}
