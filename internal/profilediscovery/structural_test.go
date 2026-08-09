package profilediscovery

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuralProfileQualificationIsRepositoryIndependentAndDeterministic(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name      string
		manifest  string
		graph     string
		generated string
	}{
		{
			name:      "kafka",
			manifest:  "fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
			graph:     "fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
			generated: "fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
		},
		{
			name:      "opentelemetry",
			manifest:  "fixtures/poc-qualified-profile-adoption/opentelemetry/buildopt-impact-manifest.json",
			graph:     "fixtures/poc-qualified-profile-adoption/opentelemetry/buildopt-impact-graph.generated.json",
			generated: "fixtures/poc-qualified-profile-adoption/opentelemetry/buildopt-impact.generated.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := structuralTestRepository(t, root, test.manifest, test.graph, test.generated)
			evidence := qualifiedStructuralTestEvidence(t, repository)
			writeStructuralTestEvidence(t, repository, evidence)
			options := StructuralOptions{
				RepositoryRoot: repository,
				ManifestPath:   "buildopt-impact-manifest.json",
				GraphPath:      "buildopt-impact-graph.generated.json",
				GeneratedPath:  "buildopt-impact.generated.json",
				EvidencePath:   "buildopt-structural-qualification.json",
			}
			first, err := QualifyStructuralProfile(options)
			if err != nil {
				t.Fatal(err)
			}
			second, err := QualifyStructuralProfile(options)
			if err != nil {
				t.Fatal(err)
			}
			firstRaw, err := RenderStructuralProfile(first)
			if err != nil {
				t.Fatal(err)
			}
			secondRaw, err := RenderStructuralProfile(second)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(firstRaw, secondRaw) {
				t.Fatal("structural profile rendering is not deterministic")
			}
			if first.SchemaVersion != StructuralProfileSchema || first.ProfileID != StructuralProfileID ||
				!first.Mechanisms.BuildImpact || first.Mechanisms.StandardJarAdapter ||
				first.Mechanisms.SafeCache || first.Mechanisms.RuntimeTuning || first.Mechanisms.HotState ||
				first.Mechanisms.StandardCopyAdapter || first.Mechanisms.SharedEdgeCache ||
				len(first.Preconditions) != 3 || first.Qualification.Pairs != 8 ||
				first.Qualification.MeanSavedMS != 3000 || first.Qualification.Interval95SavedMS[0] <= 0 {
				t.Fatalf("structural profile = %+v", first)
			}
		})
	}
}

func TestStructuralProfileQualificationFailsClosed(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		name   string
		mutate func(*structuralEvidence)
	}{
		{"source-drift", func(e *structuralEvidence) { e.SourceBindings.GraphSHA256 = strings.Repeat("0", 64) }},
		{"output-mismatch", func(e *structuralEvidence) {
			e.Observations[0].CandidateRequiredOutputSHA256 = strings.Repeat("b", 64)
		}},
		{"product-failure", func(e *structuralEvidence) { e.Observations[0].ProductAttributableFailure = true }},
		{"missing-fallback", func(e *structuralEvidence) { e.Fallback.BuildSuccessful = false }},
		{"extra-mechanism", func(e *structuralEvidence) { e.Execution.Mechanisms = append(e.Execution.Mechanisms, "SAFE_CACHE") }},
		{"below-value-gate", func(e *structuralEvidence) {
			for index := range e.Observations {
				e.Observations[index].CandidateDurationMS = 9950
				e.Observations[index].SavedMS = 50
			}
			result, err := calculateStructuralResult(e.Observations)
			if err != nil {
				panic(err)
			}
			e.Result = result
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := structuralTestRepository(
				t,
				root,
				"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
				"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
				"fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
			)
			evidence := qualifiedStructuralTestEvidence(t, repository)
			test.mutate(&evidence)
			writeStructuralTestEvidence(t, repository, evidence)
			_, err := QualifyStructuralProfile(StructuralOptions{
				RepositoryRoot: repository,
				ManifestPath:   "buildopt-impact-manifest.json",
				GraphPath:      "buildopt-impact-graph.generated.json",
				GeneratedPath:  "buildopt-impact.generated.json",
				EvidencePath:   "buildopt-structural-qualification.json",
			})
			if err == nil {
				t.Fatal("unqualified structural evidence produced a profile")
			}
		})
	}
}

func structuralTestRepository(t *testing.T, root, manifest, graph, generated string) string {
	t.Helper()
	repository := t.TempDir()
	for source, target := range map[string]string{
		manifest:  "buildopt-impact-manifest.json",
		graph:     "buildopt-impact-graph.generated.json",
		generated: "buildopt-impact.generated.json",
	} {
		raw, err := os.ReadFile(filepath.Join(root, source))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repository, target), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return repository
}

func qualifiedStructuralTestEvidence(t *testing.T, repository string) structuralEvidence {
	t.Helper()
	analysis, err := AnalyzeOpportunity(AnalysisOptions{
		RepositoryRoot: repository,
		ManifestPath:   "buildopt-impact-manifest.json",
		GraphPath:      "buildopt-impact-graph.generated.json",
		GeneratedPath:  "buildopt-impact.generated.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Plan == nil {
		t.Fatalf("analysis = %+v", analysis)
	}
	inputs := map[string]string{}
	for _, input := range analysis.Inputs {
		inputs[input.Role] = trimSHA(input.SHA256)
	}
	observations := make([]structuralObservation, structuralPairCount)
	for index := range observations {
		order := "CANDIDATE_FIRST"
		if index%2 == 0 {
			order = "CONTROL_FIRST"
		}
		observations[index] = structuralObservation{
			Pair: index + 1, Order: order,
			ControlDurationMS: 10000, CandidateDurationMS: 7000, SavedMS: 3000,
			ControlRequiredOutputSHA256:   strings.Repeat("a", 64),
			CandidateRequiredOutputSHA256: strings.Repeat("a", 64),
			RequiredOutputCount:           1,
		}
	}
	result, err := calculateStructuralResult(observations)
	if err != nil {
		t.Fatal(err)
	}
	return structuralEvidence{
		SchemaVersion: StructuralEvidenceSchema,
		EvidenceState: "QUALIFIED",
		Subject: structuralSubject{
			RepositoryID: analysis.Subject.RepositoryID, RepositoryRevision: strings.Repeat("a", 40),
			PipelineClass: analysis.Subject.PipelineClass,
		},
		SourceBindings: structuralSourceBindings{
			ManifestSHA256: inputs["BUILD_IMPACT_MANIFEST"], GraphSHA256: inputs["BUILD_IMPACT_GRAPH"],
			GeneratedSHA256: inputs["GENERATED_MANIFEST"],
		},
		Plan: *analysis.Plan,
		Execution: structuralExecution{
			CandidateSurface:         "INSTALLED_BUILDOPT_QUALIFIED_PROFILE",
			Mechanisms:               []string{"BUILD_IMPACT"},
			GradleOptions:            []string{"--daemon", "--offline", "--build-cache", "--parallel", "--no-configuration-cache", "--console=plain", "--max-workers=4", "--no-scan"},
			LauncherOverheadIncluded: true,
		},
		Observations: observations,
		Fallback:     structuralFallback{Mode: "FULL_GRAPH", Reason: "IMPACT_GLOBAL_CHANGE", BuildSuccessful: true},
		Result:       result,
		Boundaries:   structuralBoundaries{ProofOfConcept: true},
	}
}

func writeStructuralTestEvidence(t *testing.T, repository string, evidence structuralEvidence) {
	t.Helper()
	raw, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "buildopt-structural-qualification.json"), append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
