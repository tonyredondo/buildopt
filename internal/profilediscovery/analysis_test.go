package profilediscovery

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpportunityAnalysisFindsStructuralCandidatesAcrossRepositories(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name           string
		manifest       string
		graph          string
		generated      string
		minimumOmitted int
	}{
		{
			name:           "opentelemetry",
			manifest:       "fixtures/poc-qualified-profile-adoption/opentelemetry/buildopt-impact-manifest.json",
			graph:          "fixtures/poc-qualified-profile-adoption/opentelemetry/buildopt-impact-graph.generated.json",
			generated:      "fixtures/poc-qualified-profile-adoption/opentelemetry/buildopt-impact.generated.json",
			minimumOmitted: 1,
		},
		{
			name:           "kafka",
			manifest:       "fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
			graph:          "fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
			generated:      "fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
			minimumOmitted: 50,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := AnalyzeOpportunity(AnalysisOptions{
				RepositoryRoot: root,
				ManifestPath:   test.manifest,
				GraphPath:      test.graph,
				GeneratedPath:  test.generated,
			})
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != DecisionMeasure || report.Reason != "COMPLETE_STRUCTURAL_REDUCTION" || report.Plan == nil {
				t.Fatalf("decision = %s/%s, plan = %+v", report.Decision, report.Reason, report.Plan)
			}
			if report.Plan.OmittedProjectCount < test.minimumOmitted || report.Plan.OmittedProjectRatio <= 0 || report.Plan.OmittedProjectRatio >= 1 {
				t.Fatalf("invalid reduction = %+v", report.Plan)
			}
			if !report.MeasurementRequired || report.ActivationAutomatic || report.ProductionAuthorized || !report.ReviewRequired {
				t.Fatalf("unsafe analysis authority = %+v", report)
			}
			if len(report.Mechanisms) == 0 || report.Mechanisms[0].Status != "MEASURE_CANDIDATE" || report.Mechanisms[0].Name != "BUILD_IMPACT" {
				t.Fatalf("mechanisms = %+v", report.Mechanisms)
			}
			for _, mechanism := range report.Mechanisms[1:] {
				if mechanism.Status != "NOT_AUTHORIZED" {
					t.Fatalf("unmeasured mechanism enabled: %+v", mechanism)
				}
			}
			first, err := RenderAnalysis(report)
			if err != nil {
				t.Fatal(err)
			}
			second, err := RenderAnalysis(report)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, second) {
				t.Fatal("analysis rendering is not deterministic")
			}
		})
	}
}

func TestOpportunityAnalysisFailsClosedOnUncertainStructure(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
		reason string
	}{
		{"incomplete", func(value map[string]any) { value["complete"] = false }, "GRAPH_INCOMPLETE"},
		{"unknown", func(value map[string]any) {
			value["projects"].([]any)[0].(map[string]any)["unknownRelationships"] = true
		}, "GRAPH_UNKNOWN_RELATIONSHIP"},
		{"test-task", func(value map[string]any) {
			value["entrypoints"].([]any)[0].(map[string]any)["containsTestTasks"] = true
		}, "UNSUPPORTED_TEST_TASK"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			temporary := t.TempDir()
			manifest := "manifest.json"
			graph := "graph.json"
			generated := "generated.json"
			for source, target := range map[string]string{
				"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json":        manifest,
				"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json": graph,
				"fixtures/poc-kafka-packaging/buildopt-impact.generated.json":       generated,
			} {
				raw, err := os.ReadFile(filepath.Join(root, source))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(temporary, target), raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			mutateJSONFile(t, filepath.Join(temporary, graph), test.mutate)
			report, err := AnalyzeOpportunity(AnalysisOptions{
				RepositoryRoot: temporary,
				ManifestPath:   manifest,
				GraphPath:      graph,
				GeneratedPath:  generated,
			})
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != DecisionNative || report.Reason != test.reason || report.Plan != nil {
				t.Fatalf("decision = %s/%s, plan = %+v", report.Decision, report.Reason, report.Plan)
			}
		})
	}
}
