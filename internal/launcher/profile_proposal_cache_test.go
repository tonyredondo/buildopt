package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

func TestProfileProposalCacheReplaysOnlyExactValidatedArtifacts(t *testing.T) {
	base := strings.Repeat("a", 40)
	target := strings.Repeat("b", 40)
	config := structuralProposalConfig{
		repositoryID: "example/repository", pipelineClass: "build",
		baseRevision: base, entrypoints: []string{"build"},
		requiredOutputs: []string{"app/build/libs/app.jar"},
		globalChanges:   []string{"build.gradle"}, timeout: 5 * time.Minute,
		outputContractOutput: "buildopt-output-contract.json",
		manifestOutput:       "buildopt-impact-manifest.json",
		graphOutput:          "buildopt-impact-graph.generated.json",
		generatedOutput:      "buildopt-impact.generated.json",
		fallbackOutput:       "buildopt-fallback-changes.txt",
		proposalOutput:       "buildopt-profile-proposal.json",
	}
	manifest := buildimpact.Manifest{
		SchemaVersion: buildimpact.ManifestSchemaVersion, ManifestVersion: 1,
		RepositoryID: config.repositoryID, PipelineClass: config.pipelineClass,
		Ownership:           buildimpact.RepositoryOwnership,
		OriginalEntrypoints: []string{"build"},
		AllowedAlternatives: []buildimpact.EntrypointSet{{ID: "changed-projects", Entrypoints: []string{":app:build"}}},
		RequiredArtifacts:   []buildimpact.Artifact{{ID: "required-output-1", Path: config.requiredOutputs[0], Owner: buildimpact.BuildOptimization}},
		RequiredChecks:      []buildimpact.Check{}, GlobalChangePaths: config.globalChanges,
		UnknownChangePolicy: buildimpact.FullGraphPolicy,
	}
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw = append(manifestRaw, '\n')
	loadedManifest, err := buildimpact.ParseManifest(manifestRaw, config.repositoryID, config.pipelineClass)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := buildimpact.DiscoverySnapshot{
		SchemaVersion: buildimpact.DiscoverySchemaVersion, GradleVersion: "9.6.1", Complete: true,
		Projects: []buildimpact.DiscoveredProject{
			{Path: ":", SourcePaths: []string{"src/**"}, DependsOn: []string{}},
			{Path: ":app", SourcePaths: []string{"app/src/**"}, DependsOn: []string{}},
		},
		Entrypoints: []buildimpact.DiscoveredEntrypoint{
			{Name: "build", ReachesProjects: []string{":", ":app"}},
			{Name: ":app:build", ReachesProjects: []string{":app"}},
		},
	}
	snapshotRaw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := buildimpact.GenerateImpact(loadedManifest, snapshotRaw)
	if err != nil {
		t.Fatal(err)
	}
	analysis := profilediscovery.AnalyzeGeneratedOpportunity(loadedManifest, generated.Graph, generated.Generated)
	if analysis.Decision != profilediscovery.DecisionMeasure {
		t.Fatalf("analysis = %+v", analysis)
	}
	outputContract := outputContractReport{
		SchemaVersion: outputContractSchema, Decision: "VALIDATED_REQUIRED_OUTPUTS",
		Reason:       "DECLARED_OUTPUTS_MATCH_EXECUTED_WORKFLOW",
		RepositoryID: config.repositoryID, PipelineClass: config.pipelineClass,
		RepositoryRevision: target, GradleVersion: "9.6.1",
		OriginalEntrypoints: config.entrypoints, DeclaredOutputs: config.requiredOutputs,
		Validations: []outputContractValidation{{
			Pattern: config.requiredOutputs[0], Status: "VALIDATED", MatchedFiles: 1,
			OwnerProjects: []string{":app"}, ProducerTasks: []string{":app:build"},
		}},
		ReviewRequired: true, TestOptimization: "OUT_OF_SCOPE",
	}
	outputRaw, err := renderOutputContract(outputContract)
	if err != nil {
		t.Fatal(err)
	}
	report := nativeProfileProposal(config, target, []string{"app/src/Main.java"})
	report.Decision = analysis.Decision
	report.Reason = analysis.Reason
	report.CandidateEntrypoints = []string{":app:build"}
	report.Analysis = &analysis
	report.Documents = profileProposalDocuments{
		OutputContract: config.outputContractOutput, Manifest: config.manifestOutput,
		Graph: config.graphOutput, Generated: config.generatedOutput,
		FallbackChanges: config.fallbackOutput, Proposal: config.proposalOutput,
	}
	documents := map[string][]byte{
		config.outputContractOutput: outputRaw,
		config.manifestOutput:       manifestRaw,
		config.graphOutput:          generated.GraphJSON,
		config.generatedOutput:      generated.GeneratedJSON,
		config.fallbackOutput:       []byte("build.gradle\n"),
	}
	binding := profileProposalCacheBinding{
		SchemaVersion: profileProposalCacheSchema, Digest: strings.Repeat("c", 64),
		RepositoryID: config.repositoryID, PipelineClass: config.pipelineClass,
		BaseRevision: base, TargetRevision: target,
		ChangedPaths: []string{"app/src/Main.java"}, Entrypoints: config.entrypoints,
		RequiredOutputs: config.requiredOutputs, GlobalChanges: config.globalChanges,
		TimeoutNanoseconds: config.timeout.Nanoseconds(),
		OutputPaths:        []string{config.outputContractOutput, config.manifestOutput, config.graphOutput, config.generatedOutput, config.fallbackOutput, config.proposalOutput},
		WrapperFiles:       map[string]string{"gradlew": strings.Repeat("d", 64)},
		ExecutableSHA256:   strings.Repeat("e", 64),
	}
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	if err := storeProfileProposalCache(cacheRoot, binding, config, report, documents, snapshot); err != nil {
		t.Fatal(err)
	}
	loadedReport, loadedDocuments, hit, err := loadProfileProposalCache(cacheRoot, binding, config)
	if err != nil || !hit || loadedReport.Decision != report.Decision || len(loadedDocuments) != len(documents) {
		t.Fatalf("exact replay = %v/%v/%+v", hit, err, loadedReport)
	}

	drifted := binding
	drifted.GradleOptions = []string{"--offline"}
	if _, _, _, err := loadProfileProposalCache(cacheRoot, drifted, config); err == nil {
		t.Fatal("proposal replay accepted option drift under the same lookup key")
	}

	cachePath := filepath.Join(cacheRoot, binding.Digest+".json")
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 1
	if err := os.WriteFile(cachePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := loadProfileProposalCache(cacheRoot, binding, config); err == nil {
		t.Fatal("proposal replay accepted tampered state")
	}
}
