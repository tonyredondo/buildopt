package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	optimizeDiscoveryReady       = "READY"
	optimizeDiscoveryComplete    = "COMPLETE"
	optimizeDiscoveryRetained    = "NATIVE_RETAINED"
	optimizeDiscoverySkipped     = "SKIPPED"
	optimizeDiscoveryReasonFound = "STRUCTURAL_CANDIDATE_DISCOVERED"
	optimizeFamilyDependency     = "DEPENDENCY_SOURCE"
	optimizeFamilyResource       = "RESOURCE"
	optimizeFamilyLeaf           = "LEAF_SOURCE"
	optimizeFamilyMixed          = "MIXED_SOURCE"
)

var optimizeGlobalChangePaths = append([]string(nil), defaultProposalGlobalChanges...)

type optimizeDiscoveryContext struct {
	Ready            bool     `json:"-"`
	Source           string   `json:"source"`
	Reason           string   `json:"reason"`
	RepositoryID     string   `json:"repositoryId"`
	BaseRevision     string   `json:"baseRevision"`
	TargetRevision   string   `json:"targetRevision"`
	ChangeSHA256     string   `json:"changeSha256"`
	ChangedPathCount int      `json:"changedPathCount"`
	Entrypoints      []string `json:"entrypoints"`
	changedPaths     []string
	gradleOptions    []string
}

type optimizeDiscoveryGraph struct {
	TotalProjects    int `json:"totalProjects"`
	SelectedProjects int `json:"selectedProjects"`
	OmittedProjects  int `json:"omittedProjects"`
}

const optimizeAggregatePartitionSchema = "buildopt.poc/aggregate-workflow-partition/v1"

type optimizeAggregateTaskGroup struct {
	Selector       string   `json:"selector"`
	Variant        string   `json:"variant"`
	TaskContract   string   `json:"taskContract"`
	Entrypoints    []string `json:"entrypoints"`
	OutputPatterns []string `json:"outputPatterns"`
}

type optimizeAggregatePartition struct {
	SchemaVersion                  string                       `json:"schemaVersion"`
	Status                         string                       `json:"status"`
	Reason                         string                       `json:"reason"`
	ABIPolicy                      string                       `json:"abiPolicy"`
	RebuildProjects                []string                     `json:"rebuildProjects"`
	AffectedProjects               []string                     `json:"affectedProjects"`
	MaterializedProjects           []string                     `json:"materializedProjects"`
	LegacyCandidateEntrypointCount int                          `json:"legacyCandidateEntrypointCount"`
	CandidateEntrypointCount       int                          `json:"candidateEntrypointCount"`
	CandidateOutputCount           int                          `json:"candidateOutputCount"`
	MaterializedOutputCount        int                          `json:"materializedOutputCount"`
	TaskGroups                     []optimizeAggregateTaskGroup `json:"taskGroups"`
}

type optimizeDiscoveryResult struct {
	Status               string                        `json:"status"`
	Reason               string                        `json:"reason"`
	Source               string                        `json:"source"`
	RepositoryID         string                        `json:"repositoryId"`
	BaseRevision         string                        `json:"baseRevision"`
	TargetRevision       string                        `json:"targetRevision"`
	ChangeSHA256         string                        `json:"changeSha256"`
	ChangedPathCount     int                           `json:"changedPathCount"`
	Entrypoints          []string                      `json:"entrypoints"`
	RequiredOutputs      []string                      `json:"requiredOutputs"`
	CandidateOutputs     []string                      `json:"candidateOutputs"`
	CandidateEntrypoints []string                      `json:"candidateEntrypoints"`
	AggregatePartition   *optimizeAggregatePartition   `json:"aggregatePartition,omitempty"`
	Materialization      optimizeOutputMaterialization `json:"materialization"`
	ChangeFamily         string                        `json:"changeFamily"`
	ChangedProjects      []string                      `json:"changedProjects"`
	WorkflowIgnoredPaths []string                      `json:"workflowIgnoredPaths"`
	Graph                optimizeDiscoveryGraph        `json:"graph"`
	GeneratedFiles       []string                      `json:"generatedFiles"`
	ReviewRequired       bool                          `json:"reviewRequired"`
	ProductionAuthorized bool                          `json:"productionAuthorized"`
	TestOptimization     string                        `json:"testOptimization"`
	outputCandidates     []outputContractCandidate
}

type optimizeDiscoveryDocuments struct {
	values map[string][]byte
}

func retainedOptimizeDiscovery(invocation optimizeInvocation, reason string) optimizeDiscoveryResult {
	return optimizeDiscoveryResult{
		Status: optimizeDiscoveryRetained, Reason: reason,
		Source:           invocation.discovery.Source,
		RepositoryID:     invocation.discovery.RepositoryID,
		BaseRevision:     invocation.discovery.BaseRevision,
		TargetRevision:   invocation.discovery.TargetRevision,
		ChangeSHA256:     invocation.discovery.ChangeSHA256,
		ChangedPathCount: invocation.discovery.ChangedPathCount,
		Entrypoints:      append([]string(nil), invocation.discovery.Entrypoints...),
		RequiredOutputs:  []string{}, CandidateOutputs: []string{}, CandidateEntrypoints: []string{},
		ChangedProjects: []string{}, WorkflowIgnoredPaths: []string{}, GeneratedFiles: []string{},
		ReviewRequired: true, ProductionAuthorized: false,
		TestOptimization: "OUT_OF_SCOPE",
	}
}

type optimizeGitHubEvent struct {
	Before      string `json:"before"`
	PullRequest *struct {
		Base struct {
			SHA string `json:"sha"`
		} `json:"base"`
	} `json:"pull_request"`
}

func inspectOptimizeDiscoveryContext(repositoryRoot string, gradleArgs []string, getenv func(string) string) optimizeDiscoveryContext {
	context := optimizeDiscoveryContext{Reason: "REPOSITORY_CONTEXT_UNAVAILABLE"}
	entrypoints, options, reason := splitOptimizeGradleWorkflow(gradleArgs)
	context.Entrypoints = entrypoints
	context.gradleOptions = options
	if reason != "" {
		context.Reason = reason
		return context
	}
	target, err := gitOutput(repositoryRoot, "rev-parse", "HEAD")
	if err != nil || !validMeasurementRevision(strings.TrimSpace(target)) {
		context.Reason = "TARGET_REVISION_UNAVAILABLE"
		return context
	}
	context.TargetRevision = strings.TrimSpace(target)
	if !optimizeRepositoryClean(repositoryRoot) {
		context.Reason = "WORKTREE_DIRTY"
		return context
	}
	context.RepositoryID = optimizeRepositoryID(repositoryRoot, getenv)
	if !outputContractRepositoryPattern.MatchString(context.RepositoryID) {
		context.Reason = "REPOSITORY_IDENTITY_UNAVAILABLE"
		return context
	}
	base, source, reason := optimizeComparisonBase(repositoryRoot, context.TargetRevision, getenv)
	context.Source = source
	if reason != "" {
		context.Reason = reason
		return context
	}
	context.BaseRevision = base
	target, err = proposalGitTarget(repositoryRoot, base)
	if err != nil || target != context.TargetRevision {
		context.Reason = "BASE_REVISION_AMBIGUOUS"
		return context
	}
	changes, err := proposalGitChangedPaths(repositoryRoot, base, context.TargetRevision)
	if err != nil {
		context.Reason = "CHANGE_SET_UNAVAILABLE"
		return context
	}
	context.changedPaths = changes
	context.ChangedPathCount = len(changes)
	context.ChangeSHA256 = optimizeDigest("buildopt-optimize-change-set-v1", changes...)
	context.Ready = true
	context.Reason = optimizeDiscoveryReady
	return context
}

func optimizeRepositoryClean(repositoryRoot string) bool {
	tracked, err := gitOutput(repositoryRoot, "status", "--porcelain=v1", "-z", "--untracked-files=no")
	if err != nil || tracked != "" {
		return false
	}
	untracked, err := gitOutput(
		repositoryRoot,
		"status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".",
		":(exclude).buildopt", ":(exclude).buildopt/**",
		":(exclude).gradle", ":(exclude).gradle/**",
	)
	return err == nil && untracked == ""
}

func splitOptimizeGradleWorkflow(arguments []string) ([]string, []string, string) {
	if len(arguments) == 0 || len(arguments) > 128 {
		return nil, nil, "WORKFLOW_ARGUMENTS_AMBIGUOUS"
	}
	valueOptions := map[string]bool{
		"--configuration-cache-problems": true,
		"--console":                      true,
		"--max-workers":                  true,
		"--priority":                     true,
		"--warning-mode":                 true,
	}
	rejectedOptions := map[string]bool{
		"--build-file": true, "-b": true,
		"--exclude-task": true, "-x": true,
		"--gradle-user-home": true, "-g": true,
		"--include-build": true,
		"--init-script":   true, "-I": true,
		"--project-dir": true, "-p": true,
		"--settings-file": true, "-c": true,
		"--tests": true,
	}
	flagOptions := map[string]bool{
		"--build-cache": true, "--continue": true, "--configuration-cache": true,
		"--daemon": true, "--debug": true, "--dry-run": true,
		"--full-stacktrace": true, "--info": true, "--no-build-cache": true,
		"--no-configuration-cache": true, "--no-daemon": true, "--no-parallel": true,
		"--no-scan": true, "--offline": true, "--parallel": true, "--profile": true,
		"--quiet": true, "-q": true, "--refresh-dependencies": true,
		"--rerun-tasks": true, "--scan": true, "--stacktrace": true, "-s": true,
		"--warn": true, "-w": true, "--write-locks": true,
	}
	entrypoints := make([]string, 0, 4)
	options := make([]string, 0, len(arguments))
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if argument == "--" {
			continue
		}
		optionName := argument
		if equals := strings.IndexByte(argument, '='); equals >= 0 {
			optionName = argument[:equals]
		}
		if rejectedOptions[optionName] {
			return entrypoints, options, "WORKFLOW_BOUNDARY_UNSUPPORTED"
		}
		if strings.HasPrefix(argument, "-P") || strings.HasPrefix(argument, "-D") {
			if argument == "-P" || argument == "-D" {
				return entrypoints, options, "WORKFLOW_ARGUMENTS_AMBIGUOUS"
			}
			options = append(options, argument)
			continue
		}
		if valueOptions[optionName] {
			options = append(options, argument)
			if optionName == argument {
				if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "-") {
					return entrypoints, options, "WORKFLOW_ARGUMENTS_AMBIGUOUS"
				}
				index++
				options = append(options, arguments[index])
			}
			continue
		}
		if flagOptions[argument] {
			options = append(options, argument)
			continue
		}
		if strings.HasPrefix(argument, "-") {
			return entrypoints, options, "WORKFLOW_OPTION_UNSUPPORTED"
		}
		entrypoints = append(entrypoints, argument)
	}
	if len(entrypoints) == 0 || len(entrypoints) > maximumStructuralAlternativeEntrypoints {
		return entrypoints, options, "WORKFLOW_ARGUMENTS_AMBIGUOUS"
	}
	if _, err := proposalTerminalSelectors(entrypoints); err != nil || !uniqueMeasurementStrings(entrypoints) {
		return entrypoints, options, "WORKFLOW_ARGUMENTS_AMBIGUOUS"
	}
	return entrypoints, options, ""
}

func optimizeComparisonBase(repositoryRoot, target string, getenv func(string) string) (string, string, string) {
	if eventPath := getenv("GITHUB_EVENT_PATH"); eventPath != "" {
		base, err := optimizeGitHubBase(eventPath)
		if err != nil || !validMeasurementRevision(base) || optimizeZeroRevision(base) {
			return "", "GITHUB", "CI_BASE_INVALID"
		}
		return base, "GITHUB", ""
	}
	for _, candidate := range []struct {
		name, source string
	}{
		{"CI_MERGE_REQUEST_DIFF_BASE_SHA", "GITLAB_MERGE_REQUEST"},
		{"CI_COMMIT_BEFORE_SHA", "GITLAB_PUSH"},
	} {
		if base := strings.ToLower(strings.TrimSpace(getenv(candidate.name))); base != "" {
			if !validMeasurementRevision(base) || optimizeZeroRevision(base) {
				return "", candidate.source, "CI_BASE_INVALID"
			}
			return base, candidate.source, ""
		}
	}
	upstream, err := gitOutput(repositoryRoot, "rev-parse", "@{upstream}^{commit}")
	if err != nil {
		return "", "LOCAL", "BASE_REVISION_AMBIGUOUS"
	}
	upstream = strings.TrimSpace(upstream)
	mergeBase, err := gitOutput(repositoryRoot, "merge-base", upstream, target)
	if err != nil {
		return "", "LOCAL_UPSTREAM", "BASE_REVISION_AMBIGUOUS"
	}
	mergeBase = strings.TrimSpace(mergeBase)
	if !validMeasurementRevision(mergeBase) || mergeBase == target {
		return "", "LOCAL_UPSTREAM", "BASE_REVISION_AMBIGUOUS"
	}
	return mergeBase, "LOCAL_UPSTREAM", ""
}

func optimizeGitHubBase(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 1 || info.Size() > 1<<20 {
		return "", errors.New("GitHub event must be one bounded regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	var event optimizeGitHubEvent
	if err := decoder.Decode(&event); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", errors.New("GitHub event contains trailing JSON")
	}
	base := event.Before
	if event.PullRequest != nil {
		base = event.PullRequest.Base.SHA
	}
	return strings.ToLower(strings.TrimSpace(base)), nil
}

func optimizeZeroRevision(revision string) bool {
	return strings.Trim(revision, "0") == ""
}

func optimizeRepositoryID(repositoryRoot string, getenv func(string) string) string {
	if value := strings.TrimSpace(getenv("GITHUB_REPOSITORY")); outputContractRepositoryPattern.MatchString(value) {
		return value
	}
	if value := optimizeGitLabRepositoryID(strings.TrimSpace(getenv("CI_PROJECT_PATH"))); value != "" {
		return value
	}
	if remote, err := gitOutput(repositoryRoot, "config", "--get", "remote.origin.url"); err == nil {
		if repositoryID := optimizeRepositoryIDFromRemote(strings.TrimSpace(remote)); repositoryID != "" {
			return repositoryID
		}
	}
	digest := optimizeDigest("buildopt-optimize-local-repository-v1", repositoryRoot)
	return "local/" + digest[:20]
}

func optimizeGitLabRepositoryID(path string) string {
	segments := strings.Split(path, "/")
	if len(segments) < 2 || len(path) > 512 {
		return ""
	}
	for _, segment := range segments {
		if !outputContractRepositoryPattern.MatchString("x/" + segment) {
			return ""
		}
	}
	if len(segments) == 2 {
		return path
	}
	// The public result schema uses owner/repository. Preserve nested GitLab
	// identity without truncating to the last subgroup by storing an opaque ID;
	// the CI scope separately binds the provider's immutable numeric project ID.
	digest := optimizeDigest("buildopt-optimize-gitlab-repository-v1", path)
	return "gitlab/" + digest[:20]
}

func optimizeRepositoryIDFromRemote(remote string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	trimmed = strings.TrimRight(trimmed, "/")
	if colon := strings.LastIndex(trimmed, ":"); colon >= 0 && !strings.Contains(trimmed[colon+1:], "//") {
		trimmed = trimmed[colon+1:]
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) < 2 {
		return ""
	}
	candidate := parts[len(parts)-2] + "/" + parts[len(parts)-1]
	if outputContractRepositoryPattern.MatchString(candidate) {
		return candidate
	}
	return ""
}

func optimizeDiscoveryContextSHA(context optimizeDiscoveryContext) string {
	values := []string{
		context.Source, context.Reason, context.RepositoryID,
		context.BaseRevision, context.TargetRevision, context.ChangeSHA256,
		optimizeDigest("buildopt-optimize-entrypoints-v1", context.Entrypoints...),
	}
	return optimizeDigest("buildopt-optimize-discovery-context-v1", values...)
}

func (run *optimizeRun) discover(discoveryContext context.Context, exitCode int, diagnostics io.Writer) optimizeDiscoveryResult {
	result := retainedOptimizeDiscovery(run.invocation, "NATIVE_BUILD_FAILED")
	result.Status = optimizeDiscoverySkipped
	if exitCode != 0 || !run.childStarted {
		if !run.childStarted {
			result.Reason = "NATIVE_BUILD_NOT_STARTED"
		}
		return result
	}
	if !run.invocation.discovery.Ready {
		result.Status = optimizeDiscoveryRetained
		result.Reason = run.invocation.discovery.Reason
		return result
	}
	observed, observationErr := run.observedOutputSnapshot()
	if observationErr != nil {
		result.Status = optimizeDiscoveryRetained
		result.Reason = "ORDINARY_OUTPUT_OBSERVATION_UNAVAILABLE"
		_, _ = fmt.Fprintf(diagnostics, "buildopt: automatic discovery unavailable: %v\n", observationErr)
		return result
	}
	observedImpact, impactErr := run.observedImpactSnapshot()
	if impactErr != nil {
		result.Status = optimizeDiscoveryRetained
		result.Reason = "ORDINARY_IMPACT_OBSERVATION_UNAVAILABLE"
		_, _ = fmt.Fprintf(diagnostics, "buildopt: inline impact discovery unavailable: %v\n", impactErr)
		return result
	}
	observedInputs, inputErr := run.observedWorkflowInputRelevance()
	if inputErr != nil {
		result.Status = optimizeDiscoveryRetained
		result.Reason = "ORDINARY_WORKFLOW_INPUT_OBSERVATION_UNAVAILABLE"
		_, _ = fmt.Fprintf(diagnostics, "buildopt: workflow-input discovery unavailable: %v\n", inputErr)
		return result
	}
	discovered, documents, err := runAutomaticOptimizeDiscovery(
		discoveryContext, run.invocation, observed, observedImpact, observedInputs,
	)
	if err != nil {
		result.Status = optimizeDiscoveryRetained
		result.Reason = optimizeDiscoveryErrorReason(err)
		_, _ = fmt.Fprintf(diagnostics, "buildopt: automatic discovery unavailable: %v\n", err)
		return result
	}
	result = discovered
	if result.Status == optimizeDiscoveryComplete {
		materialization, captureErr := captureOptimizeOutputMaterialization(run.invocation, result)
		if captureErr != nil {
			result.Status = optimizeDiscoveryRetained
			result.Reason = "OUTPUT_MATERIALIZATION_CAPTURE_FAILED"
			_, _ = fmt.Fprintf(diagnostics, "buildopt: verified output capture unavailable: %v\n", captureErr)
		} else {
			result.Materialization = materialization
		}
	}
	if err := writeOptimizeDiscoveryDocuments(run.invocation.repositoryRoot, documents); err != nil {
		result.Status = optimizeDiscoveryRetained
		result.Reason = "DISCOVERY_STATE_WRITE_FAILED"
		result.GeneratedFiles = []string{}
	}
	return result
}

func runAutomaticOptimizeDiscovery(ctx context.Context, invocation optimizeInvocation, observed *outputContractSnapshot, observedImpact *buildimpact.DiscoverySnapshot, observedInputs *buildimpact.WorkflowInputRelevance) (optimizeDiscoveryResult, optimizeDiscoveryDocuments, error) {
	discovery := invocation.discovery
	result := optimizeDiscoveryResult{
		Status: optimizeDiscoveryRetained, Reason: "NO_SAFE_STRUCTURAL_CANDIDATE",
		Source: discovery.Source, RepositoryID: discovery.RepositoryID,
		BaseRevision: discovery.BaseRevision, TargetRevision: discovery.TargetRevision,
		ChangeSHA256: discovery.ChangeSHA256, ChangedPathCount: discovery.ChangedPathCount,
		Entrypoints:     append([]string(nil), discovery.Entrypoints...),
		RequiredOutputs: []string{}, CandidateOutputs: []string{}, CandidateEntrypoints: []string{},
		ChangedProjects: []string{}, WorkflowIgnoredPaths: []string{}, GeneratedFiles: []string{},
		ReviewRequired: true, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	if head, err := gitOutput(invocation.repositoryRoot, "rev-parse", "HEAD"); err != nil || strings.TrimSpace(head) != discovery.TargetRevision {
		result.Reason = "TARGET_REVISION_DRIFT"
		return result, optimizeDiscoveryDocuments{}, nil
	}
	for _, changedPath := range discovery.changedPaths {
		if matchesAnyProposalGlob(optimizeGlobalChangePaths, changedPath) {
			result.Reason = "GLOBAL_CHANGE_REQUIRES_FULL_GRAPH"
			return result, optimizeDiscoveryDocuments{}, nil
		}
	}
	pipelineClass := optimizePipelineClass(discovery.Entrypoints, discovery.ChangeSHA256)
	outputConfig := outputContractConfig{
		repositoryID: discovery.RepositoryID, pipelineClass: pipelineClass,
		repositoryRevision: discovery.TargetRevision,
		entrypoints:        append([]string(nil), discovery.Entrypoints...),
		gradleOptions:      append([]string(nil), discovery.gradleOptions...),
		timeout:            invocation.calibrationBudget,
	}
	if observed == nil {
		return result, optimizeDiscoveryDocuments{}, errors.New("ordinary output observation is required")
	}
	outputReport, err := prepareOutputContractReportFromSnapshot(invocation.repositoryRoot, outputConfig, *observed)
	if err != nil {
		return result, optimizeDiscoveryDocuments{}, fmt.Errorf("output preflight: %w", err)
	}
	if len(outputReport.CandidateOutputs) == 0 {
		result.Reason = "OUTPUTS_MISSING"
		return result, optimizeDiscoveryDocuments{}, nil
	}
	if observedImpact == nil {
		return result, optimizeDiscoveryDocuments{}, errors.New("inline graph preflight is required")
	}
	snapshot := *observedImpact
	if !snapshot.Complete {
		result.Reason = "ORIGINAL_WORKFLOW_UNSUPPORTED"
		return result, optimizeDiscoveryDocuments{}, nil
	}
	for _, entrypoint := range snapshot.Entrypoints {
		if entrypoint.ContainsTestTasks {
			result.Reason = "TEST_EXECUTION_OUT_OF_SCOPE"
			return result, optimizeDiscoveryDocuments{}, nil
		}
		if entrypoint.UnknownRelationships {
			result.Reason = "ORIGINAL_WORKFLOW_UNSUPPORTED"
			return result, optimizeDiscoveryDocuments{}, nil
		}
	}
	changedOwners, ownerErr := buildimpact.ResolveProjectOwners(snapshot, discovery.changedPaths)
	ignoredPaths := []string{}
	if ownerErr != nil {
		if observedInputs == nil {
			result.Reason = "SOURCE_OWNERSHIP_AMBIGUOUS"
			return result, optimizeDiscoveryDocuments{}, nil
		}
		changedOwners, ignoredPaths, ownerErr = buildimpact.ResolveWorkflowProjectOwners(
			snapshot, *observedInputs, discovery.changedPaths,
		)
		if ownerErr != nil {
			result.Reason = "SOURCE_OWNERSHIP_AMBIGUOUS"
			return result, optimizeDiscoveryDocuments{}, nil
		}
	}
	relevantPaths := optimizeWorkflowRelevantPaths(discovery.changedPaths, ignoredPaths)
	affected := optimizeAffectedProjects(snapshot, changedOwners)
	result.ChangeFamily = optimizeChangeFamily(snapshot, relevantPaths, changedOwners)
	result.ChangedProjects = append([]string(nil), changedOwners...)
	result.WorkflowIgnoredPaths = append([]string(nil), ignoredPaths...)
	workflowPatterns := optimizeRequiredOutputPatterns(outputReport.CandidateOutputs, discovery.Entrypoints, nil)
	partition, candidatePatterns, candidateOwners, partitionReason := optimizeAggregateWorkflowPartition(
		outputReport,
		discovery.Entrypoints,
		workflowPatterns,
		changedOwners,
		affected,
	)
	result.AggregatePartition = partition
	if partitionReason != "" || len(candidatePatterns) == 0 || len(workflowPatterns) == 0 {
		result.Reason = "OUTPUT_SEMANTICS_AMBIGUOUS"
		if partitionReason != "" {
			result.Reason = partitionReason
		}
		return result, optimizeDiscoveryDocuments{}, nil
	}

	directory := filepath.Join(filepath.FromSlash(invocation.stateRelative), "discovery")
	config := structuralProposalConfig{
		repositoryID: discovery.RepositoryID, pipelineClass: pipelineClass,
		entrypoints:  append([]string(nil), discovery.Entrypoints...),
		baseRevision: discovery.BaseRevision, requiredOutputs: workflowPatterns,
		globalChanges:        append([]string(nil), optimizeGlobalChangePaths...),
		gradleOptions:        append([]string(nil), discovery.gradleOptions...),
		outputContractOutput: filepath.Join(directory, "output-contract.json"),
		manifestOutput:       filepath.Join(directory, "manifest.json"),
		graphOutput:          filepath.Join(directory, "graph.json"),
		generatedOutput:      filepath.Join(directory, "generated-manifest.json"),
		fallbackOutput:       filepath.Join(directory, "fallback-changes.txt"),
		proposalOutput:       filepath.Join(directory, "proposal.json"),
		changeSource:         "GIT_DIFF_BASE_TO_HEAD", timeout: invocation.calibrationBudget,
		observedOutputSnapshot: &outputReport.snapshot,
		observedImpactSnapshot: observedImpact,
		candidateOwnerProjects: candidateOwners,
		workflowIgnoredPaths:   append([]string(nil), ignoredPaths...),
	}
	report, documents, err := prepareStructuralProfileProposal(ctx, config)
	if err != nil {
		return result, optimizeDiscoveryDocuments{}, fmt.Errorf("proposal: %w", err)
	}
	proposalRaw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return result, optimizeDiscoveryDocuments{}, err
	}
	documents[config.proposalOutput] = append(proposalRaw, '\n')
	changesPath := filepath.Join(directory, "changes.txt")
	// Replay must consume the same effective change set that discovery used.
	// Reintroducing paths proven unconsumed by the requested workflow would make
	// the generic impact planner reject the already verified candidate as an
	// unknown change.
	documents[changesPath] = optimizeReplayChangedPaths(discovery.changedPaths, ignoredPaths)
	paths := make([]string, 0, len(documents))
	for path := range documents {
		paths = append(paths, filepath.ToSlash(path))
	}
	sort.Strings(paths)
	result.RequiredOutputs = append([]string(nil), report.RequiredOutputs...)
	result.CandidateOutputs = append([]string(nil), candidatePatterns...)
	result.CandidateEntrypoints = append([]string(nil), report.CandidateEntrypoints...)
	result.outputCandidates = append([]outputContractCandidate(nil), outputReport.CandidateOutputs...)
	result.GeneratedFiles = paths
	result.Reason = report.Reason
	if report.Analysis != nil && report.Analysis.Plan != nil {
		result.Graph = optimizeDiscoveryGraph{
			TotalProjects:    report.Analysis.Plan.TotalProjectCount,
			SelectedProjects: report.Analysis.Plan.SelectedProjectCount,
			OmittedProjects:  report.Analysis.Plan.OmittedProjectCount,
		}
	}
	if report.Decision == profilediscovery.DecisionMeasure && report.Reason == "COMPLETE_STRUCTURAL_REDUCTION" {
		result.Status = optimizeDiscoveryComplete
		result.Reason = optimizeDiscoveryReasonFound
	}
	return result, optimizeDiscoveryDocuments{values: documents}, nil
}

func optimizeWorkflowRelevantPaths(changedPaths, ignoredPaths []string) []string {
	ignored := make(map[string]bool, len(ignoredPaths))
	for _, path := range ignoredPaths {
		ignored[path] = true
	}
	relevant := make([]string, 0, len(changedPaths))
	for _, path := range changedPaths {
		if !ignored[path] {
			relevant = append(relevant, path)
		}
	}
	return relevant
}

func optimizeReplayChangedPaths(changedPaths, ignoredPaths []string) []byte {
	return []byte(strings.Join(optimizeWorkflowRelevantPaths(changedPaths, ignoredPaths), "\n") + "\n")
}

func optimizeChangeFamily(snapshot buildimpact.DiscoverySnapshot, changedPaths, owners []string) string {
	resourceCount := 0
	for _, path := range changedPaths {
		normalized := "/" + strings.ToLower(filepath.ToSlash(path)) + "/"
		if strings.Contains(normalized, "/resources/") {
			resourceCount++
		}
	}
	if len(changedPaths) > 0 && resourceCount == len(changedPaths) {
		return optimizeFamilyResource
	}
	if resourceCount > 0 || len(owners) != 1 {
		return optimizeFamilyMixed
	}
	affected := optimizeAffectedProjects(snapshot, owners)
	if len(affected) > 1 {
		return optimizeFamilyDependency
	}
	return optimizeFamilyLeaf
}

func optimizeAffectedProjects(snapshot buildimpact.DiscoverySnapshot, owners []string) map[string]bool {
	affected := make(map[string]bool, len(owners))
	for _, owner := range owners {
		affected[owner] = true
	}
	changed := true
	for changed {
		changed = false
		for _, project := range snapshot.Projects {
			if affected[project.Path] {
				continue
			}
			for _, dependency := range project.DependsOn {
				if affected[dependency] {
					affected[project.Path] = true
					changed = true
					break
				}
			}
		}
	}
	return affected
}

// optimizeAggregateWorkflowPartition separates outputs that must be rebuilt
// from outputs that can be restored from the exact full-graph observation of
// the same repository revision. It never assumes ABI compatibility across
// commits: direct change owners are rebuilt and every omitted output remains
// bound to the revision-specific materialization manifest.
func optimizeAggregateWorkflowPartition(
	report outputContractReport,
	entrypoints, workflowPatterns, changedOwners []string,
	affected map[string]bool,
) (*optimizeAggregatePartition, []string, map[string]bool, string) {
	partition := &optimizeAggregatePartition{
		SchemaVersion: optimizeAggregatePartitionSchema,
		Status:        optimizeDiscoveryRetained,
		Reason:        "AGGREGATE_PARTITION_UNAVAILABLE",
		ABIPolicy:     "EXACT_REVISION_OUTPUTS_NO_CROSS_REVISION_ABI_INFERENCE",
		TaskGroups:    []optimizeAggregateTaskGroup{},
	}
	if len(changedOwners) == 0 || len(workflowPatterns) == 0 || !uniqueMeasurementStrings(changedOwners) {
		return partition, nil, nil, partition.Reason
	}
	candidateOwners := make(map[string]bool, len(changedOwners))
	for _, owner := range changedOwners {
		candidateOwners[owner] = true
	}
	partition.RebuildProjects = append([]string(nil), changedOwners...)
	sort.Strings(partition.RebuildProjects)
	for project := range affected {
		partition.AffectedProjects = append(partition.AffectedProjects, project)
	}
	sort.Strings(partition.AffectedProjects)

	selectors, err := proposalTerminalSelectors(entrypoints)
	if err != nil {
		partition.Reason = "AGGREGATE_ENTRYPOINTS_AMBIGUOUS"
		return partition, nil, candidateOwners, partition.Reason
	}
	candidatePatterns := optimizeRequiredOutputPatterns(report.CandidateOutputs, entrypoints, candidateOwners)
	candidateEntrypoints := optimizeObservedOutputEntrypoints(report, selectors, candidateOwners)
	partition.LegacyCandidateEntrypointCount = len(optimizeObservedOutputEntrypoints(report, selectors, affected))
	partition.CandidateEntrypointCount = len(candidateEntrypoints)
	partition.CandidateOutputCount = len(candidatePatterns)
	if len(candidatePatterns) == 0 || len(candidateEntrypoints) == 0 {
		partition.Reason = "AGGREGATE_DIRECT_OUTPUTS_MISSING"
		return partition, nil, candidateOwners, partition.Reason
	}
	if len(candidateEntrypoints) > maximumStructuralAlternativeEntrypoints {
		partition.Reason = "CANDIDATE_TASK_SET_TOO_LARGE"
		return partition, nil, candidateOwners, partition.Reason
	}

	workflowSet := make(map[string]bool, len(workflowPatterns))
	for _, pattern := range workflowPatterns {
		workflowSet[pattern] = true
	}
	candidateSet := make(map[string]bool, len(candidatePatterns))
	for _, pattern := range candidatePatterns {
		if !workflowSet[pattern] {
			partition.Reason = "AGGREGATE_OUTPUT_PARTITION_INCOMPLETE"
			return partition, nil, candidateOwners, partition.Reason
		}
		candidateSet[pattern] = true
	}
	materializedPatterns := make(map[string]bool, len(workflowPatterns)-len(candidatePatterns))
	for _, pattern := range workflowPatterns {
		if !candidateSet[pattern] {
			materializedPatterns[pattern] = true
		}
	}
	partition.MaterializedOutputCount = len(materializedPatterns)
	materializedProjects := map[string]bool{}
	for _, candidate := range report.CandidateOutputs {
		if !materializedPatterns[candidate.Pattern] || len(candidate.OwnerProjects) != 1 {
			continue
		}
		materializedProjects[candidate.OwnerProjects[0]] = true
	}
	for project := range materializedProjects {
		partition.MaterializedProjects = append(partition.MaterializedProjects, project)
	}
	sort.Strings(partition.MaterializedProjects)

	assignedPatterns := map[string]bool{}
	for _, selector := range selectors {
		group := optimizeAggregateTaskGroup{
			Selector: selector, Variant: optimizeAggregateVariant(selector),
			TaskContract: "GRADLE_OUTPUT_PRODUCER_LIFECYCLE_V1",
			Entrypoints:  []string{}, OutputPatterns: []string{},
		}
		for _, candidate := range candidateEntrypoints {
			if optimizeEntrypointSelector(candidate) == selector {
				group.Entrypoints = append(group.Entrypoints, candidate)
			}
		}
		for _, output := range report.CandidateOutputs {
			if !candidateSet[output.Pattern] || !optimizeOutputMatchesSelector(output, selector) {
				continue
			}
			group.OutputPatterns = append(group.OutputPatterns, output.Pattern)
			assignedPatterns[output.Pattern] = true
		}
		sort.Strings(group.Entrypoints)
		sort.Strings(group.OutputPatterns)
		if len(group.Entrypoints) > 0 && len(group.OutputPatterns) > 0 {
			partition.TaskGroups = append(partition.TaskGroups, group)
		}
	}
	if len(partition.TaskGroups) == 0 || len(assignedPatterns) != len(candidatePatterns) {
		partition.Reason = "AGGREGATE_TASK_GROUP_INCOMPLETE"
		return partition, nil, candidateOwners, partition.Reason
	}
	partition.Status = optimizeDiscoveryComplete
	partition.Reason = "REVISION_BOUND_OUTPUT_PARTITION"
	return partition, candidatePatterns, candidateOwners, ""
}

func optimizeEntrypointSelector(entrypoint string) string {
	if index := strings.LastIndex(entrypoint, ":"); index >= 0 {
		return entrypoint[index+1:]
	}
	return entrypoint
}

func optimizeObservedOutputEntrypoints(report outputContractReport, selectors []string, included map[string]bool) []string {
	owners := make([]string, 0, len(included))
	for owner := range included {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	entrypoints := make([]string, 0, len(owners)*len(selectors))
	for _, owner := range owners {
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
	return entrypoints
}

func optimizeOutputMatchesSelector(candidate outputContractCandidate, selector string) bool {
	for _, task := range candidate.ProducerTasks {
		if optimizeEntrypointSelector(task) == selector || optimizeLifecycleOutput(selector, candidate.Path) {
			return true
		}
	}
	return false
}

func optimizeAggregateVariant(selector string) string {
	switch strings.ToLower(selector) {
	case "assemble":
		return "MAIN_ARTIFACTS"
	case "classes":
		return "MAIN_CLASSES"
	case "testclasses":
		return "TEST_CLASSES"
	default:
		return "EXACT_TASK_OUTPUT"
	}
}

func optimizeRequiredOutputPatterns(candidates []outputContractCandidate, entrypoints []string, included map[string]bool) []string {
	selectors, err := proposalTerminalSelectors(entrypoints)
	if err != nil {
		return nil
	}
	direct := map[string]bool{}
	lifecycle := map[string]bool{}
	for _, candidate := range candidates {
		if len(candidate.OwnerProjects) != 1 ||
			(included != nil && !included[candidate.OwnerProjects[0]]) ||
			candidate.FileCount < 1 {
			continue
		}
		for _, task := range candidate.ProducerTasks {
			name := task
			if index := strings.LastIndex(task, ":"); index >= 0 {
				name = task[index+1:]
			}
			for _, selector := range selectors {
				if name == selector {
					direct[candidate.Pattern] = true
				}
				if optimizeLifecycleOutput(selector, candidate.Path) {
					lifecycle[candidate.Pattern] = true
				}
			}
		}
	}
	for pattern := range lifecycle {
		direct[pattern] = true
	}
	patterns := make([]string, 0, len(direct))
	for pattern := range direct {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)
	if len(patterns) > 256 {
		return nil
	}
	return patterns
}

func optimizeLifecycleOutput(selector, path string) bool {
	normalized := "/" + strings.ToLower(filepath.ToSlash(path)) + "/"
	switch strings.ToLower(selector) {
	case "assemble":
		// Assemble is a lifecycle task without outputs of its own. Its customer
		// contract is the terminal archives, distributions, and publication
		// metadata produced by the affected projects, not compiler caches,
		// incremental state, classes, documentation directories, or other
		// intermediates that happen to exist after the task graph executes.
		return strings.Contains(normalized, "/build/libs/") ||
			strings.Contains(normalized, "/build/distributions/") ||
			strings.Contains(normalized, "/build/publications/")
	case "classes":
		return strings.Contains(normalized, "/build/classes/") || strings.Contains(normalized, "/build/resources/main/")
	case "testclasses":
		return (strings.Contains(normalized, "/build/classes/") && strings.Contains(normalized, "/test/")) ||
			strings.Contains(normalized, "/build/resources/test/")
	default:
		return false
	}
}

func optimizePipelineClass(entrypoints []string, changeSHA string) string {
	name := strings.ToLower(strings.Join(entrypoints, "-"))
	var builder strings.Builder
	for _, character := range name {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('-')
		}
	}
	prefix := strings.Trim(builder.String(), "-._")
	if prefix == "" {
		prefix = "workflow"
	}
	if len(prefix) > 40 {
		prefix = prefix[:40]
	}
	return fmt.Sprintf("auto-%s-%s", prefix, changeSHA[:8])
}

func optimizeDiscoveryErrorReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "DISCOVERY_BUDGET_EXHAUSTED"
	}
	switch {
	case strings.HasPrefix(err.Error(), "output preflight:"):
		return "OUTPUT_PREFLIGHT_FAILED"
	case strings.HasPrefix(err.Error(), "graph preflight:"):
		return "GRAPH_PREFLIGHT_FAILED"
	case strings.HasPrefix(err.Error(), "proposal:"):
		return "PROPOSAL_PREFLIGHT_FAILED"
	}
	return "DISCOVERY_EXECUTION_FAILED"
}

func writeOptimizeDiscoveryDocuments(repositoryRoot string, documents optimizeDiscoveryDocuments) error {
	if len(documents.values) == 0 {
		return nil
	}
	for relative, raw := range documents.values {
		clean := filepath.Clean(relative)
		if relative == "" || filepath.IsAbs(relative) || clean != relative || clean == "." || clean == ".." ||
			(!strings.HasPrefix(clean, ".buildopt"+string(filepath.Separator)) && clean != ".buildopt") {
			return errors.New("optimize discovery output escaped .buildopt")
		}
		absolute := filepath.Join(repositoryRoot, clean)
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			return err
		}
		if err := writePrivateAtomicFile(absolute, raw); err != nil {
			return err
		}
	}
	return nil
}
