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
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const profileProposalUsage = "usage: buildopt profile propose --repository-id OWNER/REPO --pipeline-class CLASS --entrypoint TASK --changes-file PATH --base-revision REVISION --required-output GLOB [--required-output GLOB ...] [--global-change GLOB ...] [--gradle-command PATH] [--gradle-option VALUE ...] [--manifest-output PATH] [--graph-output PATH] [--generated-manifest-output PATH] [--fallback-changes-output PATH] [--proposal-output PATH] [--buildopt-revision REVISION] [--timeout DURATION]\n"

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
	Manifest         string `json:"manifest,omitempty"`
	Graph            string `json:"graph,omitempty"`
	Generated        string `json:"generatedManifest,omitempty"`
	FallbackChanges  string `json:"fallbackChanges,omitempty"`
	Proposal         string `json:"proposal"`
}

type profileProposalReport struct {
	repositoryRoot          string                               `json:"-"`
	SchemaVersion          string                               `json:"schemaVersion"`
	Decision               string                               `json:"decision"`
	Reason                 string                               `json:"reason"`
	RepositoryID           string                               `json:"repositoryId"`
	PipelineClass          string                               `json:"pipelineClass"`
	BaseRevision           string                               `json:"baseRevision"`
	TargetRevision         string                               `json:"targetRevision"`
	OriginalEntrypoint     string                               `json:"originalEntrypoint"`
	ChangedPaths           []string                             `json:"changedPaths"`
	RequiredOutputs        []string                             `json:"requiredOutputs"`
	GlobalChangePaths      []string                             `json:"globalChangePaths"`
	CandidateEntrypoints   []string                             `json:"candidateEntrypoints,omitempty"`
	OmittedProjects        []string                             `json:"omittedProjects,omitempty"`
	UnknownRelationships   bool                                 `json:"unknownRelationships"`
	Analysis               *profilediscovery.AnalysisReport     `json:"analysis,omitempty"`
	Documents              profileProposalDocuments             `json:"documents"`
	MeasureCommand         []string                             `json:"measureCommand,omitempty"`
	BuildOptRevisionNeeded bool                                 `json:"buildOptRevisionNeeded"`
	ReviewRequired         bool                                 `json:"reviewRequired"`
	ActivationAutomatic    bool                                 `json:"activationAutomatic"`
	ProductionAuthorized   bool                                 `json:"productionAuthorized"`
	TestOptimization       string                               `json:"testOptimization"`
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
	entrypoint := flags.String("entrypoint", "", "one original Gradle task selector")
	changesFile := flags.String("changes-file", "", "exact base-to-target changed paths")
	baseRevision := flags.String("base-revision", "", "immutable baseline Git revision")
	gradleCommand := flags.String("gradle-command", "", "repository-relative or absolute Gradle command")
	manifestOutput := flags.String("manifest-output", "buildopt-impact-manifest.json", "reviewable manifest output")
	graphOutput := flags.String("graph-output", "buildopt-impact-graph.generated.json", "reviewable graph output")
	generatedOutput := flags.String("generated-manifest-output", "buildopt-impact.generated.json", "generated-state binding output")
	fallbackOutput := flags.String("fallback-changes-output", "buildopt-fallback-changes.txt", "full-graph fallback input")
	proposalOutput := flags.String("proposal-output", "buildopt-profile-proposal.json", "reviewable proposal output")
	buildOptRevision := flags.String("buildopt-revision", "", "immutable BuildOpt revision for the next measure command")
	timeout := flags.Duration("timeout", 5*time.Minute, "Gradle discovery timeout")
	var requiredOutputs proposalStringFlag
	var globalChanges proposalStringFlag
	var gradleOptions repeatedStringFlag
	flags.Var(&requiredOutputs, "required-output", "repository-owned output glob; repeat for multiple outputs")
	flags.Var(&globalChanges, "global-change", "full-graph fallback glob; repeat to replace defaults")
	flags.Var(&gradleOptions, "gradle-option", "Gradle discovery option; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *repositoryID == "" || *pipelineClass == "" || *entrypoint == "" || *changesFile == "" || *baseRevision == "" || len(requiredOutputs) == 0 || *timeout <= 0 {
		_, _ = io.WriteString(stderr, profileProposalUsage)
		return exitUsage
	}
	if len(globalChanges) == 0 {
		globalChanges = append(globalChanges, defaultProposalGlobalChanges...)
	}
	report, documents, err := prepareStructuralProfileProposal(context.Background(), structuralProposalConfig{
		repositoryID: *repositoryID, pipelineClass: *pipelineClass,
		entrypoint: *entrypoint, changesFile: *changesFile, baseRevision: *baseRevision,
		requiredOutputs: append([]string(nil), requiredOutputs...),
		globalChanges: append([]string(nil), globalChanges...),
		gradleCommand: *gradleCommand, gradleOptions: append([]string(nil), gradleOptions...),
		manifestOutput: *manifestOutput, graphOutput: *graphOutput,
		generatedOutput: *generatedOutput, fallbackOutput: *fallbackOutput,
		proposalOutput: *proposalOutput, buildOptRevision: *buildOptRevision,
		timeout: *timeout,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile proposal unavailable: %v\n", err)
		return exitConfiguration
	}
	for _, output := range []string{report.Documents.Manifest, report.Documents.Graph, report.Documents.Generated, report.Documents.FallbackChanges} {
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
	repositoryID, pipelineClass, entrypoint, changesFile, baseRevision string
	requiredOutputs, globalChanges, gradleOptions                     []string
	gradleCommand, manifestOutput, graphOutput, generatedOutput       string
	fallbackOutput, proposalOutput, buildOptRevision                   string
	timeout                                                              time.Duration
}

func prepareStructuralProfileProposal(ctx context.Context, config structuralProposalConfig) (profileProposalReport, map[string][]byte, error) {
	root, err := canonicalWorkingDirectory()
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	if strings.Contains(config.entrypoint, ":") {
		return profileProposalReport{}, nil, errors.New("proposal entrypoint must be one unqualified Gradle task selector")
	}
	if len(config.requiredOutputs) > 256 || len(config.globalChanges) > 256 || !uniqueMeasurementStrings(config.requiredOutputs) || !uniqueMeasurementStrings(config.globalChanges) {
		return profileProposalReport{}, nil, errors.New("proposal outputs and global changes must be unique and bounded")
	}
	if len(config.gradleOptions) > 32 || !uniqueMeasurementStrings(config.gradleOptions) {
		return profileProposalReport{}, nil, errors.New("Gradle discovery options must be unique and bounded")
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
	changedPaths, err := readImpactChangedPaths(root, config.changesFile)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	if err := validateMeasurementChangeSet(root, config.baseRevision, targetRevision, changedPaths); err != nil {
		return profileProposalReport{}, nil, err
	}
	sort.Strings(changedPaths)
	sort.Strings(config.requiredOutputs)
	sort.Strings(config.globalChanges)
	report := nativeProfileProposal(config, targetRevision, changedPaths)
	report.repositoryRoot = root

	observationContext, cancel := context.WithTimeout(ctx, config.timeout)
	defer cancel()
	snapshot, err := buildimpact.ObserveGradle(observationContext, buildimpact.ObservationOptions{
		RepositoryRoot: root, Entrypoints: []string{config.entrypoint},
		GradleCommand: config.gradleCommand, GradleArgs: config.gradleOptions,
	})
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	report.UnknownRelationships = !snapshot.Complete || snapshot.Entrypoints[0].UnknownRelationships
	if !snapshot.Complete {
		report.Reason = "ORIGINAL_WORKFLOW_UNSUPPORTED"
		return report, nil, nil
	}
	for _, changedPath := range changedPaths {
		if matchesAnyProposalGlob(config.globalChanges, changedPath) {
			report.Reason = "GLOBAL_CHANGE_REQUIRES_FULL_GRAPH"
			return report, nil, nil
		}
	}
	owners, err := buildimpact.ResolveProjectOwners(snapshot, changedPaths)
	if err != nil {
		report.Reason = "SOURCE_OWNERSHIP_AMBIGUOUS"
		return report, nil, nil
	}
	candidateEntrypoints := make([]string, 0, len(owners))
	for _, owner := range owners {
		if owner == ":" {
			candidateEntrypoints = append(candidateEntrypoints, ":"+config.entrypoint)
		} else {
			candidateEntrypoints = append(candidateEntrypoints, owner+":"+config.entrypoint)
		}
	}
	report.CandidateEntrypoints = candidateEntrypoints

	manifest := buildimpact.Manifest{
		SchemaVersion: buildimpact.ManifestSchemaVersion, ManifestVersion: 1,
		RepositoryID: config.repositoryID, PipelineClass: config.pipelineClass,
		Ownership: buildimpact.RepositoryOwnership,
		OriginalEntrypoints: []string{config.entrypoint},
		AllowedAlternatives: []buildimpact.EntrypointSet{{ID: "changed-projects", Entrypoints: candidateEntrypoints}},
		RequiredChecks: []buildimpact.Check{}, GlobalChangePaths: config.globalChanges,
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
	generated, err := buildimpact.DiscoverWithManifest(finalContext, buildimpact.DiscoveryOptions{
		RepositoryRoot: root, GradleCommand: config.gradleCommand, GradleArgs: config.gradleOptions,
	}, loadedManifest)
	if err != nil {
		report.Reason = "CANDIDATE_WORKFLOW_UNSUPPORTED"
		return report, nil, nil
	}
	if !generated.Generated.Complete {
		report.Reason = "CANDIDATE_GRAPH_INCOMPLETE"
		report.UnknownRelationships = true
		return report, nil, nil
	}
	fallbackChange, err := proposalFallbackChange(config.globalChanges)
	if err != nil {
		return profileProposalReport{}, nil, err
	}
	fallbackRaw := []byte(fallbackChange + "\n")
	documents := map[string][]byte{
		config.manifestOutput: manifestRaw, config.graphOutput: generated.GraphJSON,
		config.generatedOutput: generated.GeneratedJSON, config.fallbackOutput: fallbackRaw,
	}
	analysis := profilediscovery.AnalyzeGeneratedOpportunity(loadedManifest, generated.Graph, generated.Generated)
	report.Analysis = &analysis
	report.Decision = analysis.Decision
	report.Reason = analysis.Reason
	report.UnknownRelationships = false
	if analysis.Decision != profilediscovery.DecisionMeasure {
		return report, nil, nil
	}
	report.Documents.Manifest = config.manifestOutput
	report.Documents.Graph = config.graphOutput
	report.Documents.Generated = config.generatedOutput
	report.Documents.FallbackChanges = config.fallbackOutput
	report.OmittedProjects = proposalOmittedProjects(generated.Graph.Graph, candidateEntrypoints)
	report.MeasureCommand = proposalMeasureCommand(config)
	report.BuildOptRevisionNeeded = config.buildOptRevision == ""
	return report, documents, nil
}

func nativeProfileProposal(config structuralProposalConfig, targetRevision string, changedPaths []string) profileProposalReport {
	return profileProposalReport{
		SchemaVersion: "buildopt.poc/profile-proposal/v1", Decision: "NATIVE_FULL_GRAPH",
		Reason: "NO_SAFE_STRUCTURAL_CANDIDATE", RepositoryID: config.repositoryID,
		PipelineClass: config.pipelineClass, BaseRevision: config.baseRevision,
		TargetRevision: targetRevision, OriginalEntrypoint: config.entrypoint,
		ChangedPaths: changedPaths, RequiredOutputs: config.requiredOutputs,
		GlobalChangePaths: config.globalChanges,
		Documents: profileProposalDocuments{Proposal: config.proposalOutput},
		ReviewRequired: true, ActivationAutomatic: false, ProductionAuthorized: false,
		TestOptimization: "OUT_OF_SCOPE",
	}
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
	paths := []string{config.manifestOutput, config.graphOutput, config.generatedOutput, config.fallbackOutput, config.proposalOutput}
	seen := map[string]bool{config.changesFile: true}
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
	return []string{
		"buildopt", "profile", "measure",
		"--manifest", config.manifestOutput,
		"--graph", config.graphOutput,
		"--generated-manifest", config.generatedOutput,
		"--changes-file", config.changesFile,
		"--fallback-changes-file", config.fallbackOutput,
		"--base-revision", config.baseRevision,
		"--buildopt-revision", revision,
		"--evidence-output", "buildopt-profile-evidence.json",
	}
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
