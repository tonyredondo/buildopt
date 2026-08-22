package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/nativevolatility"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

func TestApplyOptimizeNativeQuarantineFiltersPackAndResetsCalibration(t *testing.T) {
	repository := t.TempDir()
	stateRelative := ".buildopt/optimize/quarantine-test"
	stateDirectory := filepath.Join(repository, filepath.FromSlash(stateRelative))
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"changed/build/changed.jar":              "changed\n",
		"stable/build/stable.jar":                "stable\n",
		"volatile/build/classes/changed.class":   "native-first\n",
		"volatile/build/classes/unchanged.class": "same\n",
	}
	for relative, contents := range files {
		absolute := filepath.Join(repository, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binding := optimizeDigest("quarantine-test-binding")
	invocation := optimizeInvocation{
		repositoryRoot: repository, stateDirectory: stateDirectory, stateRelative: stateRelative,
		bindingSHA256: binding, calibrationPairs: 8, maxBreakEvenBuilds: 30,
		calibrationBudget: 30 * time.Minute,
	}
	discovery := optimizeDiscoveryResult{
		Status: optimizeDiscoveryComplete, Reason: optimizeDiscoveryReasonFound,
		Source: "TEST", RepositoryID: "example/quarantine", BaseRevision: strings.Repeat("a", 40),
		TargetRevision: strings.Repeat("b", 40), ChangeSHA256: optimizeDigest("change"),
		ChangedPathCount: 1, Entrypoints: []string{"classes"},
		RequiredOutputs: []string{
			"changed/build/*.jar", "stable/build/*.jar", "volatile/build/classes/**",
		},
		CandidateOutputs:     []string{"changed/build/*.jar"},
		CandidateEntrypoints: []string{":changed:classes"},
		AggregatePartition: &optimizeAggregatePartition{
			SchemaVersion: optimizeAggregatePartitionSchema, Status: optimizeDiscoveryComplete,
			Reason: "REVISION_BOUND_OUTPUT_PARTITION", ABIPolicy: "EXACT_REVISION_OUTPUTS_NO_CROSS_REVISION_ABI_INFERENCE",
			RebuildProjects: []string{":changed"}, AffectedProjects: []string{":changed"},
			MaterializedProjects:     []string{":stable", ":volatile"},
			CandidateEntrypointCount: 1, CandidateOutputCount: 1, MaterializedOutputCount: 2,
			TaskGroups: []optimizeAggregateTaskGroup{},
		},
		ChangeFamily: optimizeFamilyLeaf, ChangedProjects: []string{":changed"},
		WorkflowIgnoredPaths: []string{}, Graph: optimizeDiscoveryGraph{TotalProjects: 3, SelectedProjects: 1, OmittedProjects: 2},
		GeneratedFiles: []string{
			stateRelative + "/discovery/changes.txt", stateRelative + "/discovery/fallback-changes.txt",
			stateRelative + "/discovery/generated-manifest.json", stateRelative + "/discovery/graph.json",
			stateRelative + "/discovery/manifest.json", stateRelative + "/discovery/output-contract.json",
			stateRelative + "/discovery/proposal.json", stateRelative + "/discovery/snapshot.json",
		},
		ReviewRequired: true, TestOptimization: "OUT_OF_SCOPE",
		outputCandidates: []outputContractCandidate{
			{Pattern: "changed/build/*.jar", ProducerTasks: []string{":changed:classes"}},
			{Pattern: "stable/build/*.jar", ProducerTasks: []string{":stable:jar"}},
			{Pattern: "volatile/build/classes/**", ProducerTasks: []string{":volatile:compileJava"}},
		},
	}
	materialization, err := captureOptimizeOutputMaterialization(invocation, discovery)
	if err != nil {
		t.Fatal(err)
	}
	discovery.Materialization = materialization
	writeOptimizeQuarantineTestDiscoveryDocuments(t, invocation, discovery)
	manifest, _, err := loadOptimizeOutputMaterialization(invocation, discovery)
	if err != nil {
		t.Fatal(err)
	}
	second := nativevolatility.Observation{
		SchemaVersion: nativevolatility.ObservationSchema, BindingSHA256: binding,
		Entries: make([]nativevolatility.Entry, 0, len(manifest.Entries)),
	}
	for _, entry := range manifest.Entries {
		second.Entries = append(second.Entries, nativevolatility.Entry{
			Path: entry.Path, SHA256: entry.SHA256,
			ProducerTasks: append([]string(nil), entry.ProducerTasks...),
		})
	}
	secondPath := filepath.Join(repository, "second.json")
	secondRaw, err := json.MarshalIndent(second, "", "  ")
	if err != nil || os.WriteFile(secondPath, append(secondRaw, '\n'), 0o600) != nil {
		t.Fatal("write second native observation")
	}
	exactState := optimizeQuarantineTestState(invocation, discovery)
	exactResult, err := applyOptimizeNativeQuarantine(invocation, &exactState, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if exactResult.Reason != nativevolatility.ReasonExact ||
		len(exactResult.QuarantinedOutputs) != 0 ||
		len(exactResult.TransportedOutputs) != len(manifest.Entries) ||
		exactState.Phase != "NATIVE_RETAINED" ||
		exactState.LastReason != "CALIBRATION_VALUE_NOT_PROVEN" ||
		exactState.Discovery.Materialization.QuarantineFile == "" ||
		exactState.Discovery.Materialization.QuarantineSHA256 == "" {
		t.Fatalf("exact state = %+v, result = %+v", exactState, exactResult)
	}
	for index := range second.Entries {
		if second.Entries[index].Path == "volatile/build/classes/changed.class" {
			second.Entries[index].SHA256 = strings.Repeat("9", 64)
		}
	}
	secondRaw, err = json.MarshalIndent(second, "", "  ")
	if err != nil || os.WriteFile(secondPath, append(secondRaw, '\n'), 0o600) != nil {
		t.Fatal("write changed second native observation")
	}
	state := optimizeQuarantineTestState(invocation, discovery)
	if !validOptimizeState(state) {
		t.Fatal("pre-quarantine state is invalid")
	}
	result, err := applyOptimizeNativeQuarantine(invocation, &state, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != nativevolatility.ReasonQuarantined ||
		len(result.QuarantinedOutputs) != 2 || len(result.TransportedOutputs) != 1 ||
		!equalOptimizeStrings(state.Discovery.CandidateEntrypoints,
			[]string{":changed:classes", ":volatile:compileJava"}) ||
		state.Phase != "DISCOVERED" || state.IncrementalLearning.Status != "" ||
		state.Calibration.Status != optimizeCalibrationSkipped {
		t.Fatalf("quarantined state = %+v, result = %+v", state, result)
	}
	assertOptimizeQuarantineTestDiscoveryDocuments(t, invocation, state.Discovery)
	filtered, _, err := loadOptimizeOutputMaterialization(invocation, state.Discovery)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Entries) != 1 || filtered.Entries[0].Path != "stable/build/stable.jar" {
		t.Fatalf("filtered manifest = %+v", filtered.Entries)
	}
	for _, relative := range []string{"stable/build", "volatile/build"} {
		if err := os.RemoveAll(filepath.Join(repository, filepath.FromSlash(relative))); err != nil {
			t.Fatal(err)
		}
	}
	run := &optimizeRun{invocation: invocation, previousState: &state}
	if err := run.materializeCandidateOutputs(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repository, "stable/build/stable.jar")); err != nil {
		t.Fatal("stable output was not transported")
	}
	if _, err := os.Stat(filepath.Join(repository, "volatile/build/classes/changed.class")); !os.IsNotExist(err) {
		t.Fatal("quarantined output was transported")
	}
	if _, _, err := run.hashIncrementalOutputs(state.Discovery); err == nil {
		t.Fatal("missing locally rebuilt quarantine was accepted")
	}
	for _, relative := range []string{
		"volatile/build/classes/changed.class",
		"volatile/build/classes/unchanged.class",
	} {
		absolute := filepath.Join(repository, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte("second-native\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if digest, count, err := run.hashIncrementalOutputs(state.Discovery); err != nil ||
		!validOptimizeSHA(digest) || count != 2 {
		t.Fatalf("stable verification = %q, %d, %v", digest, count, err)
	}
}

func assertOptimizeQuarantineTestDiscoveryDocuments(
	t *testing.T,
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) {
	t.Helper()
	directory := filepath.Join(invocation.stateDirectory, "discovery")
	manifestRaw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := buildimpact.ParseManifest(
		manifestRaw, discovery.RepositoryID, "quarantine-test",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Manifest.AllowedAlternatives) != 1 ||
		!equalOptimizeStrings(
			manifest.Manifest.AllowedAlternatives[0].Entrypoints,
			discovery.CandidateEntrypoints,
		) {
		t.Fatalf("rewritten manifest = %+v", manifest.Manifest.AllowedAlternatives)
	}
	graphRaw, err := os.ReadFile(filepath.Join(directory, "graph.json"))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		t.Fatal(err)
	}
	generatedRaw, err := os.ReadFile(filepath.Join(directory, "generated-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var generated buildimpact.GeneratedManifest
	if err := json.Unmarshal(generatedRaw, &generated); err != nil {
		t.Fatal(err)
	}
	if generated.ManifestDigest != manifest.Digest || generated.GraphDigest != graph.Digest {
		t.Fatalf("rewritten generated binding = %+v", generated)
	}
	entrypoints := make([]string, 0, len(graph.Graph.Entrypoints))
	for _, entrypoint := range graph.Graph.Entrypoints {
		entrypoints = append(entrypoints, entrypoint.Name)
	}
	for _, candidate := range discovery.CandidateEntrypoints {
		if !optimizeStringIn(candidate, entrypoints...) {
			t.Fatalf("rewritten graph entrypoints = %v, missing %s", entrypoints, candidate)
		}
	}
	proposalRaw, err := os.ReadFile(filepath.Join(directory, "proposal.json"))
	if err != nil {
		t.Fatal(err)
	}
	var proposal profileProposalReport
	if err := json.Unmarshal(proposalRaw, &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.Analysis == nil || proposal.Analysis.Plan == nil ||
		!equalOptimizeStrings(proposal.CandidateEntrypoints, discovery.CandidateEntrypoints) ||
		!equalOptimizeStrings(proposal.Analysis.Plan.Entrypoints, discovery.CandidateEntrypoints) {
		t.Fatalf("rewritten proposal = %+v", proposal)
	}
}

func writeOptimizeQuarantineTestDiscoveryDocuments(
	t *testing.T,
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
) {
	t.Helper()
	manifest := buildimpact.Manifest{
		SchemaVersion: buildimpact.ManifestSchemaVersion, ManifestVersion: 1,
		RepositoryID: discovery.RepositoryID, PipelineClass: "quarantine-test",
		Ownership: buildimpact.RepositoryOwnership, OriginalEntrypoints: []string{"classes"},
		AllowedAlternatives: []buildimpact.EntrypointSet{{
			ID: "changed-projects", Entrypoints: append([]string(nil), discovery.CandidateEntrypoints...),
		}},
		RequiredArtifacts: []buildimpact.Artifact{
			{ID: "changed-output", Path: "changed/build/*.jar", Owner: buildimpact.BuildOptimization},
			{ID: "stable-output", Path: "stable/build/*.jar", Owner: buildimpact.BuildOptimization},
			{ID: "volatile-output", Path: "volatile/build/classes/**", Owner: buildimpact.BuildOptimization},
		},
		RequiredChecks: []buildimpact.Check{}, GlobalChangePaths: []string{"settings.gradle"},
		UnknownChangePolicy: buildimpact.FullGraphPolicy,
	}
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw = append(manifestRaw, '\n')
	loadedManifest, err := buildimpact.ParseManifest(
		manifestRaw, discovery.RepositoryID, manifest.PipelineClass,
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := buildimpact.DiscoverySnapshot{
		SchemaVersion: buildimpact.DiscoverySchemaVersion, GradleVersion: "9.6.1",
		Complete: true, FallbackReasons: []string{}, IncludedBuildPaths: []string{},
		Projects: []buildimpact.DiscoveredProject{
			{Path: ":changed", SourcePaths: []string{"changed/**"}, DependsOn: []string{}},
			{Path: ":stable", SourcePaths: []string{"stable/**"}, DependsOn: []string{}},
			{Path: ":volatile", SourcePaths: []string{"volatile/**"}, DependsOn: []string{}},
		},
		Tasks: []buildimpact.DiscoveredTask{
			{Path: ":changed:classes", ProjectPath: ":changed", DependsOn: []string{}},
			{Path: ":stable:jar", ProjectPath: ":stable", DependsOn: []string{}},
			{Path: ":volatile:compileJava", ProjectPath: ":volatile", DependsOn: []string{}},
		},
		Entrypoints: []buildimpact.DiscoveredEntrypoint{{
			Name: "classes", ReachesProjects: []string{":changed", ":stable", ":volatile"},
		}},
	}
	derived, err := buildimpact.DeriveProjectEntrypoints(snapshot, discovery.CandidateEntrypoints)
	if err != nil {
		t.Fatal(err)
	}
	derivedRaw, err := json.MarshalIndent(derived, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := buildimpact.GenerateImpact(loadedManifest, derivedRaw)
	if err != nil {
		t.Fatal(err)
	}
	analysis := profilediscovery.AnalyzeGeneratedOpportunity(
		generated.Manifest, generated.Graph, generated.Generated,
	)
	if analysis.Decision != profilediscovery.DecisionMeasure {
		t.Fatalf("test discovery analysis = %+v", analysis)
	}
	proposal := profileProposalReport{
		SchemaVersion: "buildopt.poc/profile-proposal/v1",
		Decision:      analysis.Decision, Reason: analysis.Reason,
		RepositoryID: discovery.RepositoryID, PipelineClass: manifest.PipelineClass,
		OriginalEntrypoints:  []string{"classes"},
		CandidateEntrypoints: append([]string(nil), discovery.CandidateEntrypoints...),
		OmittedProjects: proposalOmittedProjects(
			generated.Graph.Graph, discovery.CandidateEntrypoints,
		),
		UnknownRelationships: false, Analysis: &analysis, ReviewRequired: true,
		TestOptimization: "OUT_OF_SCOPE",
	}
	proposalRaw, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.ToSlash(filepath.Join(invocation.stateRelative, "discovery"))
	if err := writeOptimizeDiscoveryDocuments(invocation.repositoryRoot, optimizeDiscoveryDocuments{
		values: map[string][]byte{
			directory + "/manifest.json":           manifestRaw,
			directory + "/graph.json":              generated.GraphJSON,
			directory + "/generated-manifest.json": generated.GeneratedJSON,
			directory + "/snapshot.json":           append(derivedRaw, '\n'),
			directory + "/proposal.json":           append(proposalRaw, '\n'),
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func optimizeQuarantineTestState(invocation optimizeInvocation, discovery optimizeDiscoveryResult) optimizeState {
	digest := func(label string) string { return optimizeDigest("quarantine-state", label) }
	state := optimizeState{
		SchemaVersion: optimizeStateSchemaVersion, Generation: 1, Attempt: 17,
		Phase: "NATIVE_RETAINED", LastOutcome: optimizeOutcomeNative,
		LastReason: "CALIBRATION_VALUE_NOT_PROVEN", BuildStarted: true,
		Bindings: optimizeBindings{
			SHA256: invocation.bindingSHA256, Completeness: optimizeBindingDiscovery,
			ExecutableSHA256: digest("executable"), WrapperSHA256: digest("wrapper"),
			InvocationSHA256: digest("invocation"), RepositoryScopeSHA256: digest("repository"),
			DiscoveryContextSHA256: digest("discovery"),
		},
		Budget:    optimizeInvocationBudget(invocation),
		Resume:    optimizeResume{Mode: optimizeResumeAuto, Reason: optimizeResumeNone},
		Discovery: discovery, IncrementalLearning: emptyOptimizeIncrementalLearning(),
		Calibration: emptyOptimizeCalibration(invocation, "CALIBRATION_VALUE_NOT_PROVEN"),
		Portfolio:   emptyOptimizePortfolio("CALIBRATION_VALUE_NOT_PROVEN"),
		Selection:   emptyOptimizeSelection(optimizeSelectionSkipped, optimizeSelectionReasonNone, false),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	state.Selection.DurationNS = 1
	return state
}
