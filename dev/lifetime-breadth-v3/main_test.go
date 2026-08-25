package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEligibleObservation(t *testing.T) {
	for _, test := range []struct {
		wrapper        bool
		reason         string
		selected, want bool
	}{
		{true, "QUALIFIED_PROFILE_SELECTED", true, true},
		{true, "ORDINARY_OBSERVATIONS_PENDING", false, true},
		{true, "STRUCTURAL_PROFILE_REBOUND", false, true},
		{true, "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", false, true},
		{true, "ECONOMIC_PREQUALIFICATION_REJECTED", false, true},
		{false, "ORDINARY_OBSERVATIONS_PENDING", false, false},
		{true, "PORTFOLIO_CONTEXT_DRIFT", false, false},
	} {
		if got := eligibleObservation(test.wrapper, test.reason, test.selected); got != test.want {
			t.Fatalf("eligibleObservation(%v, %q, %v) = %v, want %v", test.wrapper, test.reason, test.selected, got, test.want)
		}
	}
}

func TestValidateOrdinaryStopPoints(t *testing.T) {
	c := contract{}
	c.Learning.MaximumRequestedBuildsPerQualification = 17
	c.Learning.MaximumProjectedPaybackMatches = 5
	base := ordinaryEconomics{SchemaVersion: "buildopt.poc/ordinary-learning-economics/v1", Decision: "NATIVE_RETAINED", RequestedBuilds: 3, CompatibleBuilds: 3, SuccessfulBuilds: 3, ExactOutputBuilds: 3, StructurallyPortableBuilds: 3, LearningCostMS: 1, MaximumPaybackMatches: 5, TestOptimization: "OUT_OF_SCOPE"}
	if err := validateOrdinary(base, c); err != nil {
		t.Fatal(err)
	}
	base.RequestedBuilds, base.CompatibleBuilds, base.SuccessfulBuilds, base.ExactOutputBuilds, base.StructurallyPortableBuilds = 2, 2, 2, 2, 2
	if err := validateOrdinary(base, c); err == nil {
		t.Fatal("two-build economic retention was accepted")
	}
}

func TestBuildSummaryFromEarlyRetentions(t *testing.T) {
	root := t.TempDir()
	specs := filepath.Join(root, "specs")
	output := filepath.Join(root, "evidence")
	if err := os.MkdirAll(specs, 0o700); err != nil {
		t.Fatal(err)
	}
	subjects := subjectSpec{}
	repositories := []string{
		"spring-projects/spring-framework",
		"open-telemetry/opentelemetry-java-instrumentation",
		"apache/kafka",
		"micronaut-projects/micronaut-core",
		"apache/groovy",
	}
	for index := 0; index < 5; index++ {
		subjects.Repositories = append(subjects.Repositories, repositorySpec{
			Key: "subject-" + string(rune('a'+index)), RepositoryID: repositories[index],
			PublicRevision: repeat("1", 40), TargetRevision: repeat(string(rune('2'+index)), 40),
		})
	}
	subjectRaw := mustJSON(t, subjects)
	if err := os.WriteFile(filepath.Join(specs, "subjects.json"), subjectRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	c := contract{SchemaVersion: contractSchema, WorkItem: "POC-LIFETIME-BREADTH-V3-001"}
	c.FrozenSubjects.Path, c.FrozenSubjects.SHA256 = "specs/subjects.json", digest(subjectRaw)
	c.FrozenSubjects.RepositoryCount = 5
	for _, subject := range subjects.Repositories {
		c.FrozenSubjects.Repositories = append(c.FrozenSubjects.Repositories, subject.RepositoryID)
	}
	c.Learning.HistoricalCompatibleMatchesRequired, c.Learning.MaximumProjectedPaybackMatches = 5, 5
	c.Learning.MaximumRequestedBuildsPerQualification, c.Learning.EarlyRetentionBuildCounts = 17, []int{1, 3}
	c.Learning.RobustPairs, c.Learning.MinimumPositivePairs = 8, 6
	c.Learning.PositiveIntervalRequired, c.Learning.CandidateP95NonRegressive = true, true
	c.Learning.PositiveFirstPairOnlyContinuesLearning, c.Learning.PositiveEconomicsDoesNotQualifyAlone = true, true
	c.Acceptance.MinimumNetPositiveRepositoryFamilies, c.Acceptance.MinimumEligibleDescendantSelectionRatio = 3, 0.5
	c.Acceptance.AllFiveSubjectsObserved, c.Acceptance.SameExecutableSHA256 = true, true
	c.Acceptance.ExactOutputsForEveryRequestedBuild, c.Acceptance.ZeroProductAttributableFailures = true, true
	c.Acceptance.TerminalPassDecision, c.Acceptance.TerminalFailDecision = "FUNCTIONAL_COVERAGE_PROVEN", "FUNCTIONAL_COVERAGE_NOT_PROVEN"
	c.Boundaries = boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"}
	contractPath := filepath.Join(specs, "contract.json")
	if err := os.WriteFile(contractPath, mustJSON(t, c), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, expected := range subjects.Repositories {
		directory := filepath.Join(output, expected.Key)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		economics := ordinaryEconomics{SchemaVersion: "buildopt.poc/ordinary-learning-economics/v1", Decision: "NATIVE_RETAINED", Reason: "COMPATIBLE_LIFETIME_INSUFFICIENT", RequestedBuilds: 1, CompatibleBuilds: 1, SuccessfulBuilds: 1, ExactOutputBuilds: 1, StructurallyPortableBuilds: 1, LearningCostMS: 10, HistoricalCompatibleMatches: 4, MaximumPaybackMatches: 5, TestOptimization: "OUT_OF_SCOPE"}
		result := rawResult{SchemaVersion: earlySchema, Key: expected.Key}
		result.BuildOpt.Revision, result.BuildOpt.ExecutableSHA256 = repeat("a", 40), repeat("b", 64)
		result.Repository.ID = expected.RepositoryID
		result.Qualification.ParentRevision, result.Qualification.TargetRevision = expected.PublicRevision, expected.TargetRevision
		result.Qualification.OrdinaryEconomics = economics
		capture := capture{ProductFailure: false}
		capture.BuildOpt = result.BuildOpt
		capture.IncrementalLearning.OrdinaryEconomics = economics
		if err := os.WriteFile(filepath.Join(directory, "result.json"), mustJSON(t, result), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "qualification-capture.json"), mustJSON(t, capture), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := buildSummary(contractPath, output, "2026-08-25T00:00:00Z", 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision != "FUNCTIONAL_COVERAGE_NOT_PROVEN" || got.Aggregation.SubjectCount != 5 || got.Aggregation.RequestedQualificationBuilds != 5 || got.Aggregation.SignedNetSavedMS != -50 {
		t.Fatalf("summary = %+v", got)
	}
	c.FrozenSubjects.Repositories[0] = "example/drifted"
	if err := os.WriteFile(contractPath, mustJSON(t, c), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := buildSummary(contractPath, output, "2026-08-25T00:00:00Z", 12); err == nil {
		t.Fatal("drifted repository list was accepted")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func repeat(value string, count int) string {
	result := ""
	for len(result) < count {
		result += value
	}
	return result[:count]
}
