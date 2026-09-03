package wcncpdetect

import (
	"strings"
	"testing"
)

func actionableInput() DetectorInput {
	return DetectorInput{
		DetectorID: "CONFIGURATION_CACHE_READINESS_PATCH", DetectorVersion: "v1",
		ProblemClass: "CONFIGURATION_CACHE_BLOCKER", ProblemReproducible: true,
		SourceOwned: true, SourcePath: "buildSrc/src/main/kotlin/Example.kt",
		SourceReversible: true, ConfigCacheEnabled: true,
		WorkflowReachable: true, Repetition: 3,
		CriticalPathMs: 900, WorkflowMs: 12000,
		EnvironmentClass: "CONTROLLED_PERFORMANCE", HasGradleProblemData: true,
	}
}

func TestKotlinAndGroovyPositivesAreActionable(t *testing.T) {
	t.Parallel()
	kotlin := actionableInput()
	kotlin.SourcePath = "buildSrc/src/main/kotlin/Example.kt"
	if row := ConfigurationCacheReadinessV1(kotlin); row.Decision != "ACTIONABLE_MATERIAL_CORRECTION" || row.RecipeClass == "" {
		t.Fatalf("kotlin = %+v", row)
	}
	groovy := actionableInput()
	groovy.SourcePath = "buildSrc/src/main/groovy/Example.groovy"
	if row := ConfigurationCacheReadinessV1(groovy); row.Decision != "ACTIONABLE_MATERIAL_CORRECTION" {
		t.Fatalf("groovy = %+v", row)
	}
}

func TestTypedNegativesReconstructIndependently(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		mutate func(*DetectorInput)
		want string
	}{
		{"missing problem data", func(input *DetectorInput) { input.HasGradleProblemData = false }, "INCOMPLETE_OBSERVATION"},
		{"ambiguous binding", func(input *DetectorInput) { input.BindingAmbiguous = true }, "SOURCE_OR_BINDING_AMBIGUOUS"},
		{"unreproducible", func(input *DetectorInput) { input.ProblemReproducible = false }, "NO_REPRODUCIBLE_BLOCKER"},
		{"drifted", func(input *DetectorInput) { input.SourceDrifted = true }, "SOURCE_DRIFTED"},
		{"external plugin", func(input *DetectorInput) { input.ExternalPluginOwned = true }, "UNSUPPORTED_PROBLEM_CLASS"},
		{"generated source", func(input *DetectorInput) { input.GeneratedOrVendor = true }, "UNSUPPORTED_PROBLEM_CLASS"},
		{"absolute path", func(input *DetectorInput) { input.AbsolutePathDependent = true }, "UNSAFE_OR_NON_REVERSIBLE"},
		{"non reversible", func(input *DetectorInput) { input.SourceReversible = false }, "UNSAFE_OR_NON_REVERSIBLE"},
		{"owner semantics", func(input *DetectorInput) { input.RequiresOwnerSemantics = true }, "OWNER_SEMANTICS_REQUIRED"},
		{"suppression", func(input *DetectorInput) { input.SuppressionStyle = true }, "UNSUPPORTED_PROBLEM_CLASS"},
		{"disable cache", func(input *DetectorInput) { input.DisablesConfigCache = true }, "UNSUPPORTED_PROBLEM_CLASS"},
		{"cache disabled", func(input *DetectorInput) { input.ConfigCacheEnabled = false }, "UNSUPPORTED_PROBLEM_CLASS"},
		{"unreachable", func(input *DetectorInput) { input.WorkflowReachable = false }, "NO_REPRODUCIBLE_BLOCKER"},
		{"non material", func(input *DetectorInput) { input.CriticalPathMs = 80; input.WorkflowMs = 471064 }, "NON_MATERIAL_BLOCKER"},
		{"hosted materiality", func(input *DetectorInput) { input.EnvironmentClass = "STANDARD_HOSTED_CI" }, "INCOMPLETE_OBSERVATION"},
	}
	for _, testCase := range testCases {
		input := actionableInput()
		testCase.mutate(&input)
		if row := ConfigurationCacheReadinessV1(input); row.Decision != testCase.want {
			t.Fatalf("%s = %s, want %s", testCase.name, row.Decision, testCase.want)
		}
	}
}

func TestClassificationIgnoresRepositoryAndTaskNames(t *testing.T) {
	t.Parallel()
	// Renamed repository, task, project, and checkout-root labels must not
	// change the decision: names are report labels, never classifier inputs.
	first := actionableInput()
	first.SourcePath = "buildSrc/src/main/kotlin/Alpha.kt"
	second := actionableInput()
	second.SourcePath = "buildSrc/src/main/kotlin/Alpha.kt"
	firstRow, secondRow := ConfigurationCacheReadinessV1(first), ConfigurationCacheReadinessV1(second)
	if firstRow.Decision != secondRow.Decision || firstRow.Decision != "ACTIONABLE_MATERIAL_CORRECTION" {
		t.Fatalf("renamed labels = %+v/%+v", firstRow, secondRow)
	}
	if strings.Contains(firstRow.RecipeClass, "Alpha") || strings.Contains(firstRow.Decision, "Alpha") {
		t.Fatal("name leaked into classification")
	}
}

func TestMaterialityGatesOnlyOnControlledRows(t *testing.T) {
	t.Parallel()
	if state, _, _ := Materiality("CONTROLLED_PERFORMANCE", 900, 12000); state != "PASSED" {
		t.Fatalf("controlled material = %s", state)
	}
	if state, _, _ := Materiality("CONTROLLED_PERFORMANCE", 80, 471064); state != "FAILED" {
		t.Fatalf("non-material = %s", state)
	}
	if state, _, _ := Materiality("STANDARD_HOSTED_CI", 90000, 12000); state != "REQUIRES_CONTROLLED_MEASUREMENT" {
		t.Fatalf("hosted must never decide = %s", state)
	}
}

func TestGroupingDeduplicatesAndNeverAveragesAcrossKeys(t *testing.T) {
	t.Parallel()
	base := ObservationSummary{
		ObservationID: strings.Repeat("a", 64), RepositoryScope: "example/repository",
		SourceTreeSHA256: strings.Repeat("1", 64), WrapperSHA256: strings.Repeat("2", 64),
		GradleVersion: "9.6.1", JDKSHA256: strings.Repeat("3", 64), PackageSHA256: strings.Repeat("4", 64),
		WorkflowSHA256: strings.Repeat("5", 64), BuildCacheMode: "ENABLED", ConfigurationCache: "PROBLEM",
		EnvironmentSHA256: strings.Repeat("6", 64), EnvironmentClass: "CONTROLLED_PERFORMANCE",
		OutputContractSHA256: strings.Repeat("7", 64),
	}
	duplicate := base
	otherWorkflow := base
	otherWorkflow.ObservationID = strings.Repeat("b", 64)
	otherWorkflow.WorkflowSHA256 = strings.Repeat("8", 64)
	groups := GroupCompatible([]ObservationSummary{base, duplicate, otherWorkflow})
	if len(groups) != 2 {
		t.Fatalf("groups = %d", len(groups))
	}
	for _, ids := range groups {
		for _, id := range ids {
			if id == "" {
				t.Fatal("empty identity grouped")
			}
		}
	}
	firstDigest := GroupDigest(CompatibilityOf(base), []string{base.ObservationID})
	if secondDigest := GroupDigest(CompatibilityOf(base), []string{base.ObservationID}); firstDigest != secondDigest {
		t.Fatal("group digest not deterministic")
	}
}

func TestCatalogAdaptersClassifyFreshEvidence(t *testing.T) {
	t.Parallel()
	kotlinNorm := NormalizationInput{TaskOwned: true, FileInputsDeclared: true, NormalizationDeclared: true, NormalizationSound: true, PortableCacheable: true, CriticalPathMs: 800, WorkflowMs: 10000, EnvironmentClass: "CONTROLLED_PERFORMANCE", SourcePath: "buildSrc/src/main/kotlin/Cacheable.kt"}
	if row := NormalizationAwareCacheabilityV2(kotlinNorm); row.Decision != "ACTIONABLE_MATERIAL_CORRECTION" {
		t.Fatalf("kotlin normalization = %+v", row)
	}
	groovyContract := TaskContractInput{TaskOwned: true, DeclaresInputsOutputs: true, IncrementalSafe: true, ContractStable: true, CriticalPathMs: 750, WorkflowMs: 9000, EnvironmentClass: "CONTROLLED_PERFORMANCE", SourcePath: "buildSrc/src/main/groovy/Stable.groovy"}
	if row := DurableTaskContractCurrent(groovyContract); row.Decision != "ACTIONABLE_MATERIAL_CORRECTION" {
		t.Fatalf("groovy contract = %+v", row)
	}
	unsafe := kotlinNorm
	unsafe.NormalizationSound = false
	if row := NormalizationAwareCacheabilityV2(unsafe); row.Decision != "UNSAFE_OR_NON_REVERSIBLE" {
		t.Fatalf("unsound normalization = %+v", row)
	}
	hosted := kotlinNorm
	hosted.EnvironmentClass = "STANDARD_HOSTED_CI"
	if row := NormalizationAwareCacheabilityV2(hosted); row.Decision != "INCOMPLETE_OBSERVATION" {
		t.Fatalf("hosted adapter must not decide = %+v", row)
	}
}
