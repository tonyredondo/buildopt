package profilediscovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

func TestKafkaDiscoveryReproducesReviewedProfileDeterministically(t *testing.T) {
	root := repositoryRoot(t)
	report, err := Discover(kafkaOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionProfile || report.Reason != "QUALIFIED_EVIDENCE_BOUND" || report.Profile == nil {
		t.Fatalf("decision = %s/%s, profile = %v", report.Decision, report.Reason, report.Profile)
	}
	expectedRaw, err := os.ReadFile(filepath.Join(root, "fixtures/poc-kafka-packaging/buildopt-qualified-edge-profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected Profile
	if err := json.Unmarshal(expectedRaw, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*report.Profile, expected) {
		t.Fatalf("generated profile differs from reviewed profile\ngot:  %+v\nwant: %+v", *report.Profile, expected)
	}
	first, err := Render(report)
	if err != nil {
		t.Fatal(err)
	}
	secondReport, err := Discover(kafkaOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(secondReport)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("repeated discovery was not byte-identical")
	}
	if report.Plan == nil || report.Plan.OmittedProjectCount != 61 || report.Plan.AlternativeID != "kafka-clients-jar" {
		t.Fatalf("plan = %+v", report.Plan)
	}
	if report.ProductionAuthorized || report.ActivationAutomatic || !report.ReviewRequired {
		t.Fatalf("unsafe authority flags = %+v", report)
	}
}

func TestUnqualifiedMatrixCellsRetainNativeGradle(t *testing.T) {
	root := repositoryRoot(t)
	for _, fixture := range []string{"spring.json", "opentelemetry.json"} {
		t.Run(fixture, func(t *testing.T) {
			options := kafkaOptions(root)
			options.CellEvidencePath = filepath.ToSlash(filepath.Join("benchmarks/results/poc-qualified-profile-matrix-v1", fixture))
			report, err := Discover(options)
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != DecisionNative || report.Reason != "EVIDENCE_NOT_QUALIFIED" || report.Profile != nil {
				t.Fatalf("decision = %s/%s, profile = %v", report.Decision, report.Reason, report.Profile)
			}
		})
	}
}

func TestQualificationDoesNotDependOnRepositoryName(t *testing.T) {
	root := repositoryRoot(t)
	temporary := t.TempDir()
	options := copyKafkaInputs(t, root, temporary)
	repositoryID := "example/independent-build"

	mutateJSONFile(t, filepath.Join(temporary, options.ManifestPath), func(value map[string]any) {
		value["repositoryId"] = repositoryID
	})
	manifestRaw, err := os.ReadFile(filepath.Join(temporary, options.ManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := buildimpact.ParseManifest(manifestRaw, repositoryID, "poc-kafka-packaging-v1")
	if err != nil {
		t.Fatal(err)
	}
	mutateJSONFile(t, filepath.Join(temporary, options.GraphPath), func(value map[string]any) {
		value["repositoryId"] = repositoryID
		value["manifestDigest"] = manifest.Digest
	})
	graphRaw, err := os.ReadFile(filepath.Join(temporary, options.GraphPath))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		t.Fatal(err)
	}
	mutateJSONFile(t, filepath.Join(temporary, options.GeneratedPath), func(value map[string]any) {
		value["repositoryId"] = repositoryID
		value["manifestDigest"] = manifest.Digest
		value["graphDigest"] = graph.Digest
	})
	mutateJSONFile(t, filepath.Join(temporary, options.CellEvidencePath), func(value map[string]any) {
		value["repository"].(map[string]any)["nameWithOwner"] = repositoryID
	})
	evidenceRaw, err := os.ReadFile(filepath.Join(temporary, options.CellEvidencePath))
	if err != nil {
		t.Fatal(err)
	}
	evidenceDigest := sha256.Sum256(evidenceRaw)
	mutateJSONFile(t, filepath.Join(temporary, options.MatrixSummaryPath), func(value map[string]any) {
		for _, rawCell := range value["cells"].([]any) {
			cell := rawCell.(map[string]any)
			if cell["id"] == "kafka-impact-read-only-edge" {
				cell["evidenceSha256"] = hex.EncodeToString(evidenceDigest[:])
			}
		}
	})

	report, err := Discover(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionProfile || report.Profile == nil || report.Profile.RepositoryID != repositoryID {
		t.Fatalf("repository-independent discovery failed: %+v", report)
	}
}

func TestDiscoveryFailsClosedOnDriftAndUncertainGraphs(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name   string
		file   string
		mutate func(map[string]any)
		reason string
	}{
		{"evidence-drift", "evidence", func(value map[string]any) { value["capturedAt"] = "drift" }, "EVIDENCE_DRIFT"},
		{"invalid-evidence-revision", "summary", func(value map[string]any) { value["evidenceRevision"] = "main" }, "EVIDENCE_REVISION_INVALID"},
		{"incomplete-graph", "graph", func(value map[string]any) { value["complete"] = false }, "GRAPH_INCOMPLETE"},
		{"unknown-relationship", "graph", func(value map[string]any) {
			value["projects"].([]any)[0].(map[string]any)["unknownRelationships"] = true
		}, "GRAPH_UNKNOWN_RELATIONSHIP"},
		{"test-task", "graph", func(value map[string]any) {
			value["entrypoints"].([]any)[0].(map[string]any)["containsTestTasks"] = true
		}, "UNSUPPORTED_TEST_TASK"},
		{"incomplete-generated-state", "generated", func(value map[string]any) { value["complete"] = false }, "GENERATED_STATE_INCOMPLETE"},
		{"precondition-drift", "contract", func(value map[string]any) {
			value["preconditions"].([]any)[0].(map[string]any)["sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}, "PROFILE_PRECONDITION_DRIFT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			temporary := t.TempDir()
			options := copyKafkaInputs(t, root, temporary)
			target := map[string]string{
				"evidence":  options.CellEvidencePath,
				"summary":   options.MatrixSummaryPath,
				"graph":     options.GraphPath,
				"generated": options.GeneratedPath,
				"contract":  options.ProfileContractPath,
			}[test.file]
			mutateJSONFile(t, filepath.Join(temporary, target), test.mutate)
			report, err := Discover(options)
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != DecisionNative || report.Reason != test.reason || report.Profile != nil {
				t.Fatalf("decision = %s/%s, profile = %v", report.Decision, report.Reason, report.Profile)
			}
		})
	}
}

func TestInputPathCannotEscapeThroughParentSymlink(t *testing.T) {
	repository := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "summary.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(repository, "linked")); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	_, err := Discover(Options{RepositoryRoot: repository, MatrixSummaryPath: "linked/summary.json"})
	if err == nil || !strings.Contains(err.Error(), "resolves outside the repository") {
		t.Fatalf("error = %v, want repository-containment rejection", err)
	}
}

func kafkaOptions(root string) Options {
	return Options{
		RepositoryRoot:      root,
		ManifestPath:        "fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
		GraphPath:           "fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
		GeneratedPath:       "fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
		MatrixSummaryPath:   "benchmarks/results/poc-qualified-profile-matrix-v1/summary.json",
		CellEvidencePath:    "benchmarks/results/poc-qualified-profile-matrix-v1/kafka.json",
		ProfileContractPath: "specs/poc-kafka-composition-usability-v1.json",
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func copyKafkaInputs(t *testing.T, root, targetRoot string) Options {
	t.Helper()
	options := kafkaOptions(targetRoot)
	source := kafkaOptions(root)
	pairs := [][2]string{
		{source.ManifestPath, options.ManifestPath}, {source.GraphPath, options.GraphPath},
		{source.GeneratedPath, options.GeneratedPath}, {source.MatrixSummaryPath, options.MatrixSummaryPath},
		{source.CellEvidencePath, options.CellEvidencePath}, {source.ProfileContractPath, options.ProfileContractPath},
	}
	for _, pair := range pairs {
		raw, err := os.ReadFile(filepath.Join(root, pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(targetRoot, pair[1])
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return options
}

func mutateJSONFile(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	mutate(value)
	raw, err = json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
