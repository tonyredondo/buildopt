package buildimpact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const artifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func validationFixture(t *testing.T, mode string) (LoadedManifest, LoadedGraph, ValidationObservation) {
	t.Helper()
	manifest, graphValue := validGraph(t)
	graph, err := ParseDeclaredGraph(encodeGraph(t, graphValue), manifest)
	if err != nil {
		t.Fatal(err)
	}
	observation := ValidationObservation{
		SchemaVersion:  ValidationObservationSchemaVersion,
		ObservationID:  "bia-observation-1",
		RepositoryID:   manifest.Manifest.RepositoryID,
		PipelineClass:  manifest.Manifest.PipelineClass,
		Revision:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ManifestDigest: manifest.Digest,
		GraphDigest:    graph.Digest,
		AdapterVersion: graph.Graph.AdapterVersion,
		ChangeClass:    "production-source",
		ObservedAt:     "2026-08-01T08:00:00Z",
		Mode:           mode,
		ChangedPaths:   []string{"library-c/src/main/java/Library.java"},
		Baseline: RunObservation{
			Outcome:     RunSuccess,
			Entrypoints: []string{"assemble"},
			Projects:    []string{":library-c", ":service-a", ":service-b"},
			Artifacts:   []ObservedArtifact{{ID: "distribution", Path: "distribution/build/libs/app.zip", Digest: artifactDigest, SizeBytes: 128}},
			Checks:      []ObservedCheck{{ID: "compile-check", Owner: BuildOptimization, Outcome: "PASSED"}, {ID: "jvm-tests", Owner: TestOptimization, Outcome: "PASSED"}},
		},
	}
	if mode == ValidationPairedControl {
		observation.Candidate = &RunObservation{
			Outcome:     RunSuccess,
			Entrypoints: []string{":service-a:assemble", ":library-c:assemble"},
			Projects:    []string{":library-c", ":service-a"},
			Artifacts:   append([]ObservedArtifact(nil), observation.Baseline.Artifacts...),
			Checks:      append([]ObservedCheck(nil), observation.Baseline.Checks...),
		}
	}
	return manifest, graph, observation
}

func encodeObservation(t *testing.T, observation ValidationObservation) []byte {
	t.Helper()
	raw, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestShadowValidationUsesFullExecutionWithoutAuthorizingSelection(t *testing.T) {
	manifest, graph, observation := validationFixture(t, ValidationShadow)
	parsed, err := ParseValidationObservation(encodeObservation(t, observation), manifest, graph)
	if err != nil {
		t.Fatal(err)
	}
	result := EvaluateValidation(manifest, graph, parsed)
	if result.Outcome != ValidationShadowPassed || !result.EligibleDecision || !result.ValidationComplete || result.FullControl || result.FalseNegative || result.SelectionAuthorized {
		t.Fatalf("result = %+v", result)
	}
	if result.AlternativeID != "service-a" || result.Reason != "FULL_EXECUTION_VALIDATES_SHADOW_MODEL" {
		t.Fatalf("shadow result = %+v", result)
	}
}

func TestPairedControlRequiresExactCandidateEquivalence(t *testing.T) {
	manifest, graph, observation := validationFixture(t, ValidationPairedControl)
	result := EvaluateValidation(manifest, graph, observation)
	if result.Outcome != ValidationControlPassed || !result.FullControl || !result.ValidationComplete || result.FalseNegative || result.SelectionAuthorized {
		t.Fatalf("result = %+v", result)
	}
}

func TestCandidateDivergenceIsFalseNegative(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*RunObservation)
		reason string
	}{
		{name: "artifact", mutate: func(run *RunObservation) {
			run.Artifacts[0].Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}, reason: "REQUIRED_ARTIFACT_DIVERGENCE"},
		{name: "check", mutate: func(run *RunObservation) { run.Checks[1].Outcome = "FAILED" }, reason: "CANDIDATE_CHECK_SET_DIVERGENCE"},
		{name: "projects", mutate: func(run *RunObservation) { run.Projects = []string{":service-a"} }, reason: "CANDIDATE_PROJECT_DIVERGENCE"},
		{name: "entrypoints", mutate: func(run *RunObservation) { run.Entrypoints = []string{":service-b:assemble"} }, reason: "CANDIDATE_ENTRYPOINT_DIVERGENCE"},
		{name: "build failure", mutate: func(run *RunObservation) { run.Outcome = RunBuildFailure }, reason: "CANDIDATE_BUILD_FAILURE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest, graph, observation := validationFixture(t, ValidationPairedControl)
			test.mutate(observation.Candidate)
			result := EvaluateValidation(manifest, graph, observation)
			if result.Outcome != ValidationFalseNegative || !result.FalseNegative || !result.ValidationComplete || result.Reason != test.reason || result.SelectionAuthorized {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestInfrastructureAndInvalidBaselineAreInconclusive(t *testing.T) {
	manifest, graph, observation := validationFixture(t, ValidationPairedControl)
	observation.Candidate.Outcome = RunInfrastructureFailure
	result := EvaluateValidation(manifest, graph, observation)
	if result.Outcome != ValidationInconclusive || result.ValidationComplete || result.FalseNegative || result.Reason != "CANDIDATE_INFRASTRUCTURE_INCONCLUSIVE" {
		t.Fatalf("infrastructure result = %+v", result)
	}
	manifest, graph, observation = validationFixture(t, ValidationShadow)
	observation.Baseline.Outcome = RunBuildFailure
	result = EvaluateValidation(manifest, graph, observation)
	if result.Outcome != ValidationInconclusive || result.EligibleDecision != true || result.Reason != "BASELINE_NOT_SUCCESSFUL" {
		t.Fatalf("baseline result = %+v", result)
	}
}

func TestFullGraphFallbackIsNotEligibleEvidence(t *testing.T) {
	manifest, graph, observation := validationFixture(t, ValidationShadow)
	observation.ChangedPaths = []string{"settings.gradle.kts"}
	result := EvaluateValidation(manifest, graph, observation)
	if result.Outcome != ValidationInconclusive || result.EligibleDecision || result.ValidationComplete || result.Reason != "DECISION_GLOBAL_CHANGE" {
		t.Fatalf("result = %+v", result)
	}
}

func TestObservationParserRejectsUnknownTrailingAndBindingDrift(t *testing.T) {
	manifest, graph, observation := validationFixture(t, ValidationShadow)
	raw := encodeObservation(t, observation)
	unknown := append(raw[:len(raw)-1], []byte(`,"historyAuthorized":true}`)...)
	if _, err := ParseValidationObservation(unknown, manifest, graph); err == nil {
		t.Fatal("unknown observation field accepted")
	}
	if _, err := ParseValidationObservation(append(raw, raw...), manifest, graph); err == nil {
		t.Fatal("trailing observation accepted")
	}
	observation.GraphDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := ParseValidationObservation(encodeObservation(t, observation), manifest, graph); err == nil {
		t.Fatal("graph binding drift accepted")
	}
}

func TestObservationParserRequiresExplicitCollectionsAndModeCandidateMatch(t *testing.T) {
	manifest, graph, observation := validationFixture(t, ValidationShadow)
	raw := string(encodeObservation(t, observation))
	for _, field := range []string{`"changedPaths":["library-c/src/main/java/Library.java"],`, `"projects":[":library-c",":service-a",":service-b"],`, `"candidate":null`} {
		candidate := strings.Replace(raw, field, "", 1)
		if candidate == raw {
			t.Fatal("test did not remove its field")
		}
		if _, err := ParseValidationObservation([]byte(candidate), manifest, graph); err == nil {
			t.Fatalf("missing field %s accepted", field)
		}
	}
	observation.Candidate = &RunObservation{Outcome: RunSuccess, Entrypoints: []string{"assemble"}, Projects: []string{}, Artifacts: []ObservedArtifact{}, Checks: []ObservedCheck{}}
	if _, err := ParseValidationObservation(encodeObservation(t, observation), manifest, graph); err == nil {
		t.Fatal("shadow candidate accepted")
	}
}

func TestObservationParserRejectsUnsafeIdentityTimeAndPath(t *testing.T) {
	for _, mutate := range []func(*ValidationObservation){
		func(value *ValidationObservation) { value.ObservationID = "bad" },
		func(value *ValidationObservation) { value.Revision = "main" },
		func(value *ValidationObservation) { value.ChangeClass = "Production Source" },
		func(value *ValidationObservation) { value.ObservedAt = "2026-08-01T10:00:00+02:00" },
		func(value *ValidationObservation) { value.ChangedPaths = []string{"../outside"} },
	} {
		manifest, graph, observation := validationFixture(t, ValidationShadow)
		mutate(&observation)
		if _, err := ParseValidationObservation(encodeObservation(t, observation), manifest, graph); err == nil {
			t.Fatal("unsafe observation accepted")
		}
	}
}

func TestCheckedInShadowAndControlObservations(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	manifest, err := LoadRepositoryManifest(repositoryRoot, filepath.FromSlash("fixtures/build-impact/manifest.v1.json"), "tonyredondo/buildopt", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	graphRaw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/build-impact/declared-graph.v1.json")))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		file    string
		outcome string
		control bool
	}{
		{file: "shadow-observation.v1.json", outcome: ValidationShadowPassed},
		{file: "paired-control-observation.v1.json", outcome: ValidationControlPassed, control: true},
	} {
		t.Run(test.file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(repositoryRoot, "fixtures", "build-impact", test.file))
			if err != nil {
				t.Fatal(err)
			}
			observation, err := ParseValidationObservation(raw, manifest, graph)
			if err != nil {
				t.Fatal(err)
			}
			result := EvaluateValidation(manifest, graph, observation)
			if result.Outcome != test.outcome || result.FullControl != test.control || !result.ValidationComplete || result.FalseNegative || result.SelectionAuthorized || result.AlternativeID != "jvm-components" {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}
