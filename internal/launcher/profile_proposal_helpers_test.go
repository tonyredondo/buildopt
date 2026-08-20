package launcher

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

func TestProposalTerminalSelectorsAcceptsSingleAndQualifiedWorkflows(t *testing.T) {
	selectors, err := proposalTerminalSelectors([]string{
		":instrumentation:spring:one:testClasses",
		":instrumentation:spring:two:testClasses",
		":instrumentation:spring:three:classes",
	})
	if err != nil {
		t.Fatalf("proposalTerminalSelectors: %v", err)
	}
	want := []string{"classes", "testClasses"}
	if !reflect.DeepEqual(selectors, want) {
		t.Fatalf("selectors = %v, want %v", selectors, want)
	}
}

func TestProposalAlternativeEntrypointBoundMatchesBuildImpactManifest(t *testing.T) {
	entrypoints := make([]string, maximumStructuralAlternativeEntrypoints+1)
	for index := range entrypoints {
		entrypoints[index] = fmt.Sprintf(":project-%d:jar", index)
	}
	manifest := buildimpact.Manifest{
		SchemaVersion: buildimpact.ManifestSchemaVersion, ManifestVersion: 1,
		RepositoryID: "owner/repository", PipelineClass: "jar",
		Ownership:           buildimpact.RepositoryOwnership,
		OriginalEntrypoints: []string{"jar"},
		AllowedAlternatives: []buildimpact.EntrypointSet{{ID: "changed-projects", Entrypoints: entrypoints}},
		RequiredArtifacts:   []buildimpact.Artifact{}, RequiredChecks: []buildimpact.Check{},
		GlobalChangePaths: []string{"build.gradle"}, UnknownChangePolicy: buildimpact.FullGraphPolicy,
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildimpact.ParseManifest(raw, manifest.RepositoryID, manifest.PipelineClass); err == nil {
		t.Fatal("Build Impact manifest accepted more alternative entrypoints than profile proposal permits")
	}
}

func TestProposalTerminalSelectorsRejectsMalformedEntrypoints(t *testing.T) {
	for _, entrypoint := range []string{"", ":", "classes task", "module/classes"} {
		if _, err := proposalTerminalSelectors([]string{entrypoint}); err == nil {
			t.Fatalf("expected %q to be rejected", entrypoint)
		}
	}
}

func TestProposalOutputOwnerProjectsUsesReviewedOutputOwners(t *testing.T) {
	report := outputContractReport{
		CandidateOutputs: []outputContractCandidate{
			{Pattern: "service-a/build/libs/a.jar", Path: "service-a/build/libs/a.jar", FileCount: 1, OwnerProjects: []string{":service-a"}, ProducerTasks: []string{":service-a:jar"}},
			{Pattern: "service-b/build/libs/b.jar", Path: "service-b/build/libs/b.jar", FileCount: 1, OwnerProjects: []string{":service-b"}, ProducerTasks: []string{":service-b:jar"}},
		},
		Validations: []outputContractValidation{
			{Pattern: "service-a/build/libs/*.jar", Status: "VALIDATED", OwnerProjects: []string{":service-a"}},
			{Pattern: "service-b/build/libs/*.jar", Status: "VALIDATED", OwnerProjects: []string{":service-b", ":service-a"}},
		},
	}
	want := []string{":service-a", ":service-b"}
	if got := proposalOutputOwnerProjects(report); !reflect.DeepEqual(got, want) {
		t.Fatalf("proposalOutputOwnerProjects = %v, want %v", got, want)
	}
	wantEntrypoints := []string{":service-a:jar", ":service-b:jar"}
	if got := proposalOutputOwnerEntrypoints(report, []string{"jar"}); !reflect.DeepEqual(got, wantEntrypoints) {
		t.Fatalf("proposalOutputOwnerEntrypoints = %v, want %v", got, wantEntrypoints)
	}
}
