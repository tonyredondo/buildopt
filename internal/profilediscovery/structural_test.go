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
	"time"
)

func TestRenderStructuralMeasurementEvidenceQualifiesOnlyPositiveExactPairs(t *testing.T) {
	root := repositoryRoot(t)
	repository := structuralTestRepository(
		t,
		root,
		"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
		"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
		"fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
	)
	analysis, err := AnalyzeOpportunity(AnalysisOptions{
		RepositoryRoot: repository,
		ManifestPath:   "buildopt-impact-manifest.json",
		GraphPath:      "buildopt-impact-graph.generated.json",
		GeneratedPath:  "buildopt-impact.generated.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	observations := make([]StructuralMeasurementObservation, structuralPairCount)
	for index := range observations {
		order := "CANDIDATE_FIRST"
		if index%2 == 0 {
			order = "CONTROL_FIRST"
		}
		observations[index] = StructuralMeasurementObservation{
			Pair: index + 1, Order: order,
			ControlDurationMS: 10000, CandidateDurationMS: 7000,
			RequiredOutputSHA256: strings.Repeat("a", 64), RequiredOutputCount: 1,
		}
	}
	raw, qualified, err := RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256:        strings.Repeat("e", 64),
		SourceEvidenceSHA256:    strings.Repeat("d", 64),
		OutputEquivalenceSHA256: strings.Repeat("f", 64),
		GradleOptions:           []string{"--daemon", "--build-cache"}, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || !qualified {
		t.Fatalf("render qualified evidence = %v/%v", qualified, err)
	}
	var evidence structuralEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.EvidenceState != "QUALIFIED" || !evidence.Result.Qualified ||
		evidence.Result.MeanSavedMS != 3000 || evidence.Result.PositivePairs != 8 ||
		evidence.SourceBindings.OutputEquivalenceSHA256 != strings.Repeat("f", 64) ||
		evidence.Execution.OutputEquivalenceMode != "OWNER_REVIEWED_SEMANTIC_V1" ||
		evidence.Boundaries.ProductionAuthorized || evidence.Boundaries.TestOptimizationModified {
		t.Fatalf("rendered evidence = %+v", evidence)
	}

	for index := range observations {
		observations[index].CandidateDurationMS = 9950
	}
	raw, qualified, err = RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256:     strings.Repeat("e", 64),
		SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions:        []string{"--daemon", "--build-cache"}, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || qualified {
		t.Fatalf("render inconclusive evidence = %v/%v", qualified, err)
	}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.EvidenceState != "INCONCLUSIVE" || evidence.Result.Qualified ||
		evidence.Result.Decision != "RETAIN_NATIVE_GRADLE" {
		t.Fatalf("inconclusive evidence = %+v", evidence)
	}
}

func TestRenderStructuralMeasurementEvidencePreservesWarmupAndPairDiagnostics(t *testing.T) {
	root := repositoryRoot(t)
	repository := structuralTestRepository(
		t,
		root,
		"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
		"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
		"fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
	)
	analysis, err := AnalyzeOpportunity(AnalysisOptions{
		RepositoryRoot: repository,
		ManifestPath:   "buildopt-impact-manifest.json",
		GraphPath:      "buildopt-impact-graph.generated.json",
		GeneratedPath:  "buildopt-impact.generated.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	taskOutcomes := StructuralTaskOutcomes{Total: 3, Executed: 1, FromCache: 1, UpToDate: 1}
	warmups := []StructuralWarmupObservation{
		{Phase: "CACHE_SEED", DurationMS: 2000, LogSHA256: strings.Repeat("1", 64), TaskOutcomes: taskOutcomes},
		{Phase: "DAEMON_STABILIZATION", DurationMS: 1000, LogSHA256: strings.Repeat("2", 64), TaskOutcomes: taskOutcomes},
	}
	observations := make([]StructuralMeasurementObservation, structuralPairCount)
	for index := range observations {
		order := "CANDIDATE_FIRST"
		if index%2 == 0 {
			order = "CONTROL_FIRST"
		}
		observations[index] = StructuralMeasurementObservation{
			Pair: index + 1, Order: order,
			ControlDurationMS: 10000, CandidateDurationMS: 7000,
			RequiredOutputSHA256: strings.Repeat("a", 64), RequiredOutputCount: 1,
			ControlLogSHA256: strings.Repeat("3", 64), CandidateLogSHA256: strings.Repeat("4", 64),
			ControlTaskOutcomes: taskOutcomes, CandidateTaskOutcomes: taskOutcomes,
		}
	}
	raw, qualified, err := RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256: strings.Repeat("e", 64), SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions: []string{"--daemon", "--build-cache"}, ControlWarmups: warmups,
		CandidateWarmups: warmups, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || !qualified {
		t.Fatalf("render diagnostic evidence = %v/%v", qualified, err)
	}
	var evidence structuralEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Execution.WarmupsPerArm != 2 || len(evidence.Execution.ControlWarmups) != 2 ||
		!reflect.DeepEqual(evidence.Observations[0].ControlTaskOutcomes, taskOutcomes) ||
		evidence.Observations[0].CandidateLogSHA256 != strings.Repeat("4", 64) {
		t.Fatalf("diagnostic evidence = %+v", evidence)
	}

	pressure := &StructuralHostPressure{Available: true, CPUSomeTotalUS: 1, IOSomeTotalUS: 2}
	controlOutcomes := taskOutcomes
	controlOutcomes.FingerprintSHA256 = strings.Repeat("5", 64)
	candidateOutcomes := taskOutcomes
	candidateOutcomes.FingerprintSHA256 = strings.Repeat("6", 64)
	warmups = []StructuralWarmupObservation{
		{Phase: "CACHE_SEED", DurationMS: 3000, LogSHA256: strings.Repeat("1", 64), TaskOutcomes: controlOutcomes, HostPressure: pressure},
		{Phase: "BASE_DAEMON_STABILIZATION", DurationMS: 2000, LogSHA256: strings.Repeat("2", 64), TaskOutcomes: controlOutcomes, HostPressure: pressure},
		{Phase: "TARGET_WORKLOAD_STABILIZATION", DurationMS: 1000, LogSHA256: strings.Repeat("3", 64), TaskOutcomes: controlOutcomes, HostPressure: pressure},
	}
	candidateWarmups := append([]StructuralWarmupObservation(nil), warmups...)
	for index := range candidateWarmups {
		candidateWarmups[index].TaskOutcomes = candidateOutcomes
	}
	for index := range observations {
		observations[index].ControlTaskOutcomes = controlOutcomes
		observations[index].CandidateTaskOutcomes = candidateOutcomes
		observations[index].ControlHostPressure = pressure
		observations[index].CandidateHostPressure = pressure
	}
	raw, qualified, err = RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256: strings.Repeat("e", 64), SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions: []string{"--daemon", "--build-cache"}, ControlWarmups: warmups,
		CandidateWarmups: candidateWarmups, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || !qualified {
		t.Fatalf("render three-phase evidence = %v/%v", qualified, err)
	}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Execution.WarmupsPerArm != 3 || !evidence.Result.ExecutionShapeObserved ||
		!evidence.Result.ExecutionShapeStable || evidence.Observations[0].ControlHostPressure == nil {
		t.Fatalf("three-phase diagnostic evidence = %+v", evidence)
	}

	controlFourPhase := append(append([]StructuralWarmupObservation(nil), warmups...), StructuralWarmupObservation{
		Phase: "TARGET_WORKLOAD_STABILITY_CONFIRMATION", DurationMS: 900,
		LogSHA256: strings.Repeat("8", 64), TaskOutcomes: controlOutcomes, HostPressure: pressure,
	})
	candidateFourPhase := append(append([]StructuralWarmupObservation(nil), candidateWarmups...), StructuralWarmupObservation{
		Phase: "TARGET_WORKLOAD_STABILITY_CONFIRMATION", DurationMS: 900,
		LogSHA256: strings.Repeat("9", 64), TaskOutcomes: candidateOutcomes, HostPressure: pressure,
	})
	raw, qualified, err = RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 11, 13, 30, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256: strings.Repeat("e", 64), SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions: []string{"--daemon", "--build-cache"}, ControlWarmups: controlFourPhase,
		CandidateWarmups: candidateFourPhase, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || !qualified {
		t.Fatalf("render four-phase evidence = %v/%v", qualified, err)
	}

	controlFivePhase := append(append([]StructuralWarmupObservation(nil), controlFourPhase...), StructuralWarmupObservation{
		Phase: "TARGET_WORKLOAD_STABILITY_RECONFIRMATION", DurationMS: 850,
		LogSHA256: strings.Repeat("a", 64), TaskOutcomes: controlOutcomes, HostPressure: pressure,
	})
	candidateFivePhase := append(append([]StructuralWarmupObservation(nil), candidateFourPhase...), StructuralWarmupObservation{
		Phase: "TARGET_WORKLOAD_STABILITY_RECONFIRMATION", DurationMS: 850,
		LogSHA256: strings.Repeat("b", 64), TaskOutcomes: candidateOutcomes, HostPressure: pressure,
	})
	controlFivePhase[2].TaskOutcomes.FingerprintSHA256 = strings.Repeat("7", 64)
	candidateFivePhase[2].TaskOutcomes.FingerprintSHA256 = strings.Repeat("8", 64)
	raw, qualified, err = RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 11, 13, 45, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256: strings.Repeat("e", 64), SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions: []string{"--daemon", "--build-cache"}, ControlWarmups: controlFivePhase,
		CandidateWarmups: candidateFivePhase, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || !qualified {
		t.Fatalf("render converged five-phase evidence = %v/%v", qualified, err)
	}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Execution.WarmupsPerArm != 5 || !evidence.Result.TargetWarmupShapeStable {
		t.Fatalf("converged five-phase evidence = %+v", evidence)
	}

	observations[7].ControlTaskOutcomes.FingerprintSHA256 = strings.Repeat("7", 64)
	raw, qualified, err = RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256: strings.Repeat("e", 64), SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions: []string{"--daemon", "--build-cache"}, ControlWarmups: warmups,
		CandidateWarmups: candidateWarmups, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || qualified {
		t.Fatalf("render drifting execution shape = %v/%v", qualified, err)
	}
	evidence = structuralEvidence{}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if !evidence.Result.ExecutionShapeObserved || evidence.Result.ExecutionShapeStable || evidence.Result.Qualified {
		t.Fatalf("drifting execution shape qualified = %+v", evidence.Result)
	}

	observations[7].ControlTaskOutcomes.FingerprintSHA256 = controlOutcomes.FingerprintSHA256
	candidateWarmups[2].TaskOutcomes.FingerprintSHA256 = strings.Repeat("7", 64)
	raw, qualified, err = RenderStructuralMeasurementEvidence(StructuralMeasurementOptions{
		CapturedAt: time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC), Analysis: analysis,
		RepositoryRevision: strings.Repeat("b", 40), BuildOptRevision: strings.Repeat("c", 40),
		ExecutableSHA256: strings.Repeat("e", 64), SourceEvidenceSHA256: strings.Repeat("d", 64),
		GradleOptions: []string{"--daemon", "--build-cache"}, ControlWarmups: warmups,
		CandidateWarmups: candidateWarmups, Observations: observations,
		FallbackReason: "IMPACT_GLOBAL_CHANGE", FallbackSuccessful: true,
	})
	if err != nil || qualified {
		t.Fatalf("render mismatched target warm-up = %v/%v", qualified, err)
	}
	evidence = structuralEvidence{}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if !evidence.Result.ExecutionShapeStable || !evidence.Result.TargetWarmupShapeObserved ||
		evidence.Result.TargetWarmupShapeStable || evidence.Result.Qualified {
		t.Fatalf("mismatched target warm-up qualified = %+v", evidence.Result)
	}
}

func TestValidateStructuralTaskOutcomesExplainsMalformedExactEvidence(t *testing.T) {
	valid := StructuralTaskOutcomes{
		Total: 2, Executed: 1, FromCache: 1,
		Tasks: []StructuralTaskObservation{
			{Path: ":alpha", Outcome: "EXECUTED"},
			{Path: ":beta", Outcome: "FROM_CACHE"},
		},
	}
	digest := sha256.Sum256([]byte("> Task :alpha\n> Task :beta FROM-CACHE\n"))
	valid.FingerprintSHA256 = hex.EncodeToString(digest[:])
	if err := ValidateStructuralTaskOutcomes(valid); err != nil {
		t.Fatalf("valid exact task evidence: %v", err)
	}

	invalidConsoleOutcomes := valid
	invalidConsoleOutcomes.Tasks = append([]StructuralTaskObservation(nil), valid.Tasks...)
	invalidConsoleOutcomes.Tasks[0].ConsoleOutcomeTransitions = []string{"EXECUTED", "FROM_CACHE"}
	if err := ValidateStructuralTaskOutcomes(invalidConsoleOutcomes); err == nil ||
		!strings.Contains(err.Error(), "terminal console outcome") {
		t.Fatalf("non-terminal console outcome error = %v", err)
	}

	mismatch := valid
	mismatch.FingerprintSHA256 = strings.Repeat("f", 64)
	if err := ValidateStructuralTaskOutcomes(mismatch); err == nil || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("fingerprint mismatch error = %v", err)
	}

	if err := ValidateStructuralHostPressure(&StructuralHostPressure{Available: true, CPUSomeTotalUS: -1}); err == nil {
		t.Fatal("negative host-pressure counter was accepted")
	}
}

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

func TestStructuralProfileQualificationBindsReviewedOutputEquivalence(t *testing.T) {
	root := repositoryRoot(t)
	repository := structuralTestRepository(
		t,
		root,
		"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
		"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
		"fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
	)
	contract := []byte(`{"schemaVersion":"buildopt.poc/output-equivalence/v1","rules":[{"pattern":"build/*.jar","mode":"CANONICAL_ZIP"}],"reviewRequired":true,"activationAutomatic":false,"productionAuthorized":false}`)
	digest := sha256.Sum256(contract)
	if err := os.WriteFile(filepath.Join(repository, "output-equivalence.json"), contract, 0o600); err != nil {
		t.Fatal(err)
	}
	evidence := qualifiedStructuralTestEvidence(t, repository)
	evidence.SourceBindings.OutputEquivalenceSHA256 = hex.EncodeToString(digest[:])
	evidence.Execution.OutputEquivalenceMode = "OWNER_REVIEWED_SEMANTIC_V1"
	writeStructuralTestEvidence(t, repository, evidence)
	options := StructuralOptions{
		RepositoryRoot: repository, ManifestPath: "buildopt-impact-manifest.json",
		GraphPath: "buildopt-impact-graph.generated.json", GeneratedPath: "buildopt-impact.generated.json",
		EvidencePath: "buildopt-structural-qualification.json", OutputEquivalencePath: "output-equivalence.json",
	}
	profile, err := QualifyStructuralProfile(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Preconditions) != 4 || profile.Preconditions[3].Path != "output-equivalence.json" ||
		profile.Preconditions[3].SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("output-equivalence precondition = %+v", profile.Preconditions)
	}
	if err := os.WriteFile(filepath.Join(repository, "output-equivalence.json"), append(contract, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := QualifyStructuralProfile(options); err == nil {
		t.Fatal("output-equivalence contract drift was accepted")
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
		CapturedAt:    "2026-08-09T00:00:00Z",
		Subject: structuralSubject{
			RepositoryID: analysis.Subject.RepositoryID, RepositoryRevision: strings.Repeat("a", 40),
			PipelineClass: analysis.Subject.PipelineClass,
		},
		SourceBindings: structuralSourceBindings{
			ManifestSHA256: inputs["BUILD_IMPACT_MANIFEST"], GraphSHA256: inputs["BUILD_IMPACT_GRAPH"],
			GeneratedSHA256: inputs["GENERATED_MANIFEST"], SourceEvidenceSHA256: strings.Repeat("c", 64),
		},
		Plan: *analysis.Plan,
		Execution: structuralExecution{
			CandidateSurface:         "INSTALLED_BUILDOPT_STRUCTURAL_IMPACT_ONLY",
			BuildOptRevision:         strings.Repeat("d", 40),
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
