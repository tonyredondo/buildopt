package buildimpact

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanPOCCandidateAcceptsGraphWithinGraphBoundAboveManifestBound(t *testing.T) {
	fixtureRoot := filepath.Join(buildImpactRepositoryRoot(t), filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	temporaryRoot := t.TempDir()
	for _, name := range []string{"buildopt-impact-manifest.json", "buildopt-impact-graph.generated.json", "buildopt-impact.generated.json"} {
		raw, err := os.ReadFile(filepath.Join(fixtureRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "buildopt-impact-graph.generated.json" {
			raw = append(bytes.Repeat([]byte(" "), maximumManifestBytes), raw...)
		}
		if err := os.WriteFile(filepath.Join(temporaryRoot, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := PlanPOCCandidate(POCCandidateOptions{
		RepositoryRoot:        temporaryRoot,
		ManifestPath:          "buildopt-impact-manifest.json",
		GraphPath:             "buildopt-impact-graph.generated.json",
		GeneratedManifestPath: "buildopt-impact.generated.json",
		RepositoryID:          "tonyredondo/buildopt-impact-synthetic",
		PipelineClass:         "pull-request",
		ChangedPaths:          []string{"library-c/src/main/java/synthetic/LibraryC.java"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CandidateSelected {
		t.Fatalf("candidate plan = %+v", plan)
	}
}

func TestPlanPOCCandidateUsesReviewedAlternativeWithoutProductionAuthority(t *testing.T) {
	fixtureRoot := filepath.Join(buildImpactRepositoryRoot(t), filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	plan, err := PlanPOCCandidate(POCCandidateOptions{
		RepositoryRoot:        fixtureRoot,
		ManifestPath:          "buildopt-impact-manifest.json",
		GraphPath:             "buildopt-impact-graph.generated.json",
		GeneratedManifestPath: "buildopt-impact.generated.json",
		RepositoryID:          "tonyredondo/buildopt-impact-synthetic",
		PipelineClass:         "pull-request",
		ChangedPaths:          []string{"library-c/src/main/java/synthetic/LibraryC.java"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CandidateSelected || plan.ProductionAuthorized || plan.Mode != POCCandidateMode || plan.AlternativeID != "affected-service-a" || len(plan.Entrypoints) != 1 || plan.Entrypoints[0] != ":service-a:assemble" {
		t.Fatalf("candidate plan = %+v", plan)
	}
	if len(plan.PreservedTestCheckIDs) != 1 || plan.PreservedTestCheckIDs[0] != "synthetic-tests" {
		t.Fatalf("preserved Test Optimization checks = %#v", plan.PreservedTestCheckIDs)
	}
	phaseSum := plan.PhaseTimings.ManifestLoadAndValidationNs +
		plan.PhaseTimings.GraphLoadAndValidationNs +
		plan.PhaseTimings.GeneratedStateLoadAndValidationNs +
		plan.PhaseTimings.ImpactEvaluationNs
	if phaseSum <= 0 || plan.PhaseTimings.TotalNs < phaseSum {
		t.Fatalf("planner phase timings do not reconcile: %+v", plan.PhaseTimings)
	}
}

func TestPlanPOCCandidateRetainsFullGraphForUnknownChange(t *testing.T) {
	fixtureRoot := filepath.Join(buildImpactRepositoryRoot(t), filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	plan, err := PlanPOCCandidate(POCCandidateOptions{
		RepositoryRoot:        fixtureRoot,
		ManifestPath:          "buildopt-impact-manifest.json",
		GraphPath:             "buildopt-impact-graph.generated.json",
		GeneratedManifestPath: "buildopt-impact.generated.json",
		RepositoryID:          "tonyredondo/buildopt-impact-synthetic",
		PipelineClass:         "pull-request",
		ChangedPaths:          []string{"unowned/file.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.CandidateSelected || plan.ProductionAuthorized || plan.Mode != DecisionFullGraph || plan.Reason != "IMPACT_UNKNOWN_CHANGE_PATH" || len(plan.Entrypoints) != 1 || plan.Entrypoints[0] != "assemble" {
		t.Fatalf("fallback plan = %+v", plan)
	}
}

func TestPlanPOCCandidateBypassDoesNotRequireGeneratedState(t *testing.T) {
	fixtureRoot := filepath.Join(buildImpactRepositoryRoot(t), filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	plan, err := PlanPOCCandidate(POCCandidateOptions{
		RepositoryRoot:        fixtureRoot,
		ManifestPath:          "buildopt-impact-manifest.json",
		GraphPath:             "missing-graph.json",
		GeneratedManifestPath: "missing-generated.json",
		RepositoryID:          "tonyredondo/buildopt-impact-synthetic",
		PipelineClass:         "pull-request",
		ChangedPaths:          []string{"library-c/src/main/java/synthetic/LibraryC.java"},
		LocalBypass:           true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.CandidateSelected || plan.Reason != "LOCAL_BYPASS" || len(plan.Entrypoints) != 1 || plan.Entrypoints[0] != "assemble" {
		t.Fatalf("bypass plan = %+v", plan)
	}
}

func TestPlanPOCCandidateRejectsGeneratedBindingDrift(t *testing.T) {
	fixtureRoot := filepath.Join(buildImpactRepositoryRoot(t), filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	temporaryRoot := t.TempDir()
	for _, name := range []string{"buildopt-impact-manifest.json", "buildopt-impact-graph.generated.json", "buildopt-impact.generated.json"} {
		raw, err := os.ReadFile(filepath.Join(fixtureRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "buildopt-impact.generated.json" {
			raw = []byte(`{"schemaVersion":"buildopt.build-impact/generated-manifest/v1"}`)
		}
		if err := os.WriteFile(filepath.Join(temporaryRoot, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := PlanPOCCandidate(POCCandidateOptions{
		RepositoryRoot:        temporaryRoot,
		ManifestPath:          "buildopt-impact-manifest.json",
		GraphPath:             "buildopt-impact-graph.generated.json",
		GeneratedManifestPath: "buildopt-impact.generated.json",
		RepositoryID:          "tonyredondo/buildopt-impact-synthetic",
		PipelineClass:         "pull-request",
		ChangedPaths:          []string{"library-c/src/main/java/synthetic/LibraryC.java"},
	})
	if err == nil {
		t.Fatal("generated binding drift was accepted")
	}
}
