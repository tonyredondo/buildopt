package buildimpact

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const (
	promotionRepository = "tonyredondo/buildopt"
	promotionPipeline   = "pull-request"
	promotionManifest   = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	promotionGraph      = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	promotionAdapter    = "gradle-declared-v1"
)

func qualifyingPromotionInput() PromotionInput {
	return PromotionInput{
		RepositoryID:           promotionRepository,
		PipelineClass:          promotionPipeline,
		ManifestDigest:         promotionManifest,
		GraphDigest:            promotionGraph,
		AdapterVersion:         promotionAdapter,
		MandatoryChangeClasses: []string{"production-source", "resource-change"},
		Results:                promotionCorpus(MinimumEligibleDecisions, MinimumShadowDays),
	}
}

func promotionCorpus(count, days int) []ValidationResult {
	results := make([]ValidationResult, count)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	spanSeconds := int64(days * 24 * 60 * 60)
	for index := range results {
		seconds := int64(0)
		if count > 1 {
			seconds = int64(index) * spanSeconds / int64(count-1)
		}
		changeClass := "production-source"
		mode := ValidationShadow
		outcome := ValidationShadowPassed
		reason := "FULL_EXECUTION_VALIDATES_SHADOW_MODEL"
		fullControl := false
		if index >= count-2*MinimumControlsPerChangeClass {
			mode = ValidationPairedControl
			outcome = ValidationControlPassed
			reason = "PAIRED_CONTROL_MATCHED"
			fullControl = true
			if index >= count-MinimumControlsPerChangeClass {
				changeClass = "resource-change"
			}
		}
		results[index] = ValidationResult{
			SchemaVersion:      ValidationResultSchemaVersion,
			ObservationID:      fmt.Sprintf("bia-synthetic-%04d", index),
			RepositoryID:       promotionRepository,
			PipelineClass:      promotionPipeline,
			Revision:           fmt.Sprintf("%040x", index+1),
			ManifestDigest:     promotionManifest,
			GraphDigest:        promotionGraph,
			AdapterVersion:     promotionAdapter,
			ChangeClass:        changeClass,
			ObservedAt:         start.Add(time.Duration(seconds) * time.Second).Format(time.RFC3339),
			Mode:               mode,
			AlternativeID:      "jvm-components",
			Outcome:            outcome,
			Reason:             reason,
			EligibleDecision:   true,
			ValidationComplete: true,
			FullControl:        fullControl,
		}
	}
	return results
}

func clonePromotionInput(input PromotionInput) PromotionInput {
	input.MandatoryChangeClasses = append([]string(nil), input.MandatoryChangeClasses...)
	input.Results = append([]ValidationResult(nil), input.Results...)
	return input
}

func TestPromotionQualifiesOnlyAtUnchangedBIA002Threshold(t *testing.T) {
	report := EvaluatePromotion(qualifyingPromotionInput())
	if report.State != PromotionQualified || report.Reason != "BIA_002_SATISFIED" || report.SelectionAuthorized {
		t.Fatalf("report = %+v", report)
	}
	if report.ShadowDays != MinimumShadowDays || report.EligibleDecisions != MinimumEligibleDecisions || report.ValidatedDecisions != MinimumEligibleDecisions || report.ValidationCoverage != 1 || report.FullControls != 2*MinimumControlsPerChangeClass || report.FalseNegatives != 0 {
		t.Fatalf("counts = %+v", report)
	}
	if report.AggregateUpper95 > MaximumAggregateFalseNegative || len(report.Strata) != 2 {
		t.Fatalf("bounds = %+v", report)
	}
	for _, stratum := range report.Strata {
		if stratum.FullControls != MinimumControlsPerChangeClass || stratum.FalseNegatives != 0 || stratum.Upper95 > MaximumStratumFalseNegative {
			t.Fatalf("stratum = %+v", stratum)
		}
	}
}

func TestPromotionInsufficientEvidenceRemainsInconclusive(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*PromotionInput)
		reason string
	}{
		{
			name: "window",
			mutate: func(input *PromotionInput) {
				input.Results = promotionCorpus(MinimumEligibleDecisions, MinimumShadowDays-1)
			},
			reason: "SHADOW_WINDOW_INSUFFICIENT",
		},
		{
			name: "eligible decisions",
			mutate: func(input *PromotionInput) {
				input.Results = input.Results[:MinimumEligibleDecisions-1]
				input.Results[len(input.Results)-1].ObservedAt = "2026-01-31T00:00:00Z"
			},
			reason: "ELIGIBLE_DECISIONS_INSUFFICIENT",
		},
		{
			name: "coverage",
			mutate: func(input *PromotionInput) {
				for index := 0; index < 31; index++ {
					input.Results[index].Outcome = ValidationInconclusive
					input.Results[index].Reason = "BASELINE_NOT_SUCCESSFUL"
					input.Results[index].ValidationComplete = false
				}
			},
			reason: "VALIDATION_COVERAGE_INSUFFICIENT",
		},
		{
			name: "mandatory controls",
			mutate: func(input *PromotionInput) {
				last := len(input.Results) - 1
				input.Results[last].Mode = ValidationShadow
				input.Results[last].Outcome = ValidationShadowPassed
				input.Results[last].Reason = "FULL_EXECUTION_VALIDATES_SHADOW_MODEL"
				input.Results[last].FullControl = false
			},
			reason: "STRATUM_CONTROLS_INSUFFICIENT",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := clonePromotionInput(qualifyingPromotionInput())
			test.mutate(&input)
			report := EvaluatePromotion(input)
			if report.State != PromotionInconclusive || report.Reason != test.reason || report.SelectionAuthorized {
				t.Fatalf("report = %+v", report)
			}
		})
	}
}

func TestAnyFalseNegativeSuspendsImmediately(t *testing.T) {
	input := clonePromotionInput(qualifyingPromotionInput())
	last := len(input.Results) - 1
	input.Results[last].Outcome = ValidationFalseNegative
	input.Results[last].Reason = "REQUIRED_ARTIFACT_DIVERGENCE"
	input.Results[last].FalseNegative = true
	report := EvaluatePromotion(input)
	if report.State != PromotionSuspended || report.Reason != "FALSE_NEGATIVE_OBSERVED" || report.FalseNegatives != 1 || report.SelectionAuthorized {
		t.Fatalf("report = %+v", report)
	}
}

func TestBindingChangeResetsEvidence(t *testing.T) {
	input := clonePromotionInput(qualifyingPromotionInput())
	for index := range input.Results {
		input.Results[index].ManifestDigest = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	}
	current := promotionCorpus(1, 0)[0]
	current.ObservationID = "bia-current-binding"
	input.Results = append(input.Results, current)
	report := EvaluatePromotion(input)
	if report.State != PromotionInconclusive || report.Reason != "SHADOW_WINDOW_INSUFFICIENT" || !report.ResetApplied || report.ExcludedPreReset != MinimumEligibleDecisions || report.EligibleDecisions != 1 || report.SelectionAuthorized {
		t.Fatalf("report = %+v", report)
	}
}

func TestPromotionRejectsDuplicateOrInvalidResults(t *testing.T) {
	input := clonePromotionInput(qualifyingPromotionInput())
	input.Results[1].ObservationID = input.Results[0].ObservationID
	if report := EvaluatePromotion(input); report.Reason != "DUPLICATE_OBSERVATION" || report.State != PromotionInconclusive {
		t.Fatalf("duplicate report = %+v", report)
	}

	input = clonePromotionInput(qualifyingPromotionInput())
	input.Results[0].SelectionAuthorized = true
	if report := EvaluatePromotion(input); report.Reason != "RESULT_INVALID" || report.State != PromotionInconclusive {
		t.Fatalf("invalid report = %+v", report)
	}
}

func TestCheckedInEvidenceIsHonestlyInconclusive(t *testing.T) {
	repositoryRoot := buildImpactRepositoryRoot(t)
	manifest, err := LoadRepositoryManifest(repositoryRoot, filepath.FromSlash("fixtures/build-impact/manifest.v1.json"), promotionRepository, promotionPipeline)
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
	var results []ValidationResult
	for _, name := range []string{"shadow-observation.v1.json", "paired-control-observation.v1.json"} {
		raw, err := os.ReadFile(filepath.Join(repositoryRoot, "fixtures", "build-impact", name))
		if err != nil {
			t.Fatal(err)
		}
		observation, err := ParseValidationObservation(raw, manifest, graph)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, EvaluateValidation(manifest, graph, observation))
	}
	report := EvaluatePromotion(PromotionInput{
		RepositoryID:           promotionRepository,
		PipelineClass:          promotionPipeline,
		ManifestDigest:         manifest.Digest,
		GraphDigest:            graph.Digest,
		AdapterVersion:         graph.Graph.AdapterVersion,
		MandatoryChangeClasses: []string{"production-source"},
		Results:                results,
	})
	if report.State != PromotionInconclusive || report.Reason != "SHADOW_WINDOW_INSUFFICIENT" || report.EligibleDecisions != 2 || report.FullControls != 1 || report.SelectionAuthorized {
		t.Fatalf("report = %+v", report)
	}
}

func TestCheckedInPromotionPolicyMatchesImplementation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(buildImpactRepositoryRoot(t), filepath.FromSlash("fixtures/build-impact/promotion-policy.v1.json")))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion                 string  `json:"schemaVersion"`
		MinimumShadowDays             int     `json:"minimumShadowDays"`
		MinimumEligibleDecisions      int     `json:"minimumEligibleDecisions"`
		MinimumControlsPerChangeClass int     `json:"minimumControlsPerChangeClass"`
		MinimumValidationCoverage     float64 `json:"minimumValidationCoverage"`
		MaximumAggregateFalseNegative float64 `json:"maximumAggregateFalseNegative"`
		MaximumStratumFalseNegative   float64 `json:"maximumStratumFalseNegative"`
		OneSidedConfidence            float64 `json:"oneSidedConfidence"`
		KnownFalseNegativeLimit       int     `json:"knownFalseNegativeLimit"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != "buildopt.build-impact/promotion-policy/v1" || fixture.MinimumShadowDays != MinimumShadowDays || fixture.MinimumEligibleDecisions != MinimumEligibleDecisions || fixture.MinimumControlsPerChangeClass != MinimumControlsPerChangeClass || fixture.MinimumValidationCoverage != MinimumValidationCoverage || fixture.MaximumAggregateFalseNegative != MaximumAggregateFalseNegative || fixture.MaximumStratumFalseNegative != MaximumStratumFalseNegative || fixture.OneSidedConfidence != 1-oneSidedAlpha || fixture.KnownFalseNegativeLimit != 0 {
		t.Fatalf("fixture does not match implementation: %+v", fixture)
	}
}

func TestExactZeroFailureUpperBounds(t *testing.T) {
	if bound := conservativeFalseNegativeUpper(0, MinimumEligibleDecisions); bound > MaximumAggregateFalseNegative {
		t.Fatalf("aggregate bound = %g", bound)
	}
	if bound := conservativeFalseNegativeUpper(0, MinimumControlsPerChangeClass); bound > MaximumStratumFalseNegative {
		t.Fatalf("stratum bound = %g", bound)
	}
	if bound := conservativeFalseNegativeUpper(1, MinimumEligibleDecisions); bound != 1 {
		t.Fatalf("failure bound = %g", bound)
	}
	want := 1 - math.Pow(0.05, 1.0/3000.0)
	if got := conservativeFalseNegativeUpper(0, 3000); math.Abs(got-want) > 1e-15 {
		t.Fatalf("upper bound = %.16g, want %.16g", got, want)
	}
}

func buildImpactRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
}
