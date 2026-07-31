package runtimeoptimizer

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestCohortLedgerPersistsAssignmentBeforeReturn(t *testing.T) {
	now := time.Date(2026, 7, 31, 14, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "cohorts")
	ledger, err := OpenCohortLedger(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	request := testCohortRequest("assignment-1", testFixedPolicy())
	assignment, created, err := ledger.Assign(request)
	if err != nil || !created {
		t.Fatalf("assign = %+v/%v/%v", assignment, created, err)
	}
	if assignment.PropensityBasisPoints <= 0 || assignment.AssignedAt != now || assignment.RandomBasisPoint < 0 || assignment.RandomBasisPoint >= BasisPointTotal {
		t.Fatalf("assignment = %+v", assignment)
	}
	stateInfo, err := os.Stat(filepath.Join(root, "cohort-assignments.json"))
	if err != nil || stateInfo.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %v/%v", stateInfo, err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil || rootInfo.Mode().Perm() != 0o700 {
		t.Fatalf("root mode = %v/%v", rootInfo, err)
	}

	reopened, err := OpenCohortLedger(root, func() time.Time { return now.Add(time.Hour) })
	if err != nil {
		t.Fatal(err)
	}
	repeat, created, err := reopened.Assign(request)
	if err != nil || created || !reflect.DeepEqual(repeat, assignment) {
		t.Fatalf("repeat = %+v/%v/%v", repeat, created, err)
	}
	conflict := request
	conflict.ContextDigest = testDigest("9")
	if _, _, err := reopened.Assign(conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict = %v", err)
	}
}

func TestFixedAAUsesIdenticalControlArmsAndDeclaredPropensities(t *testing.T) {
	ledger, err := OpenCohortLedger(filepath.Join(t.TempDir(), "cohorts"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	policy := testAAPolicy()
	seen := map[string]bool{}
	for index := 0; index < 200; index++ {
		request := testCohortRequest(identifierForIndex("aa", index), policy)
		request.SeedDigest = testDigest(string(rune('a' + index%6)))
		assignment, _, err := ledger.Assign(request)
		if err != nil {
			t.Fatal(err)
		}
		if assignment.ResourceProfileID != "STABLE_CONTROL" {
			t.Fatalf("A/A selected %s", assignment.ResourceProfileID)
		}
		want := map[string]int{"A": 7000, "B": 3000}[assignment.Cohort]
		if assignment.PropensityBasisPoints != want {
			t.Fatalf("%s propensity = %d, want %d", assignment.Cohort, assignment.PropensityBasisPoints, want)
		}
		seen[assignment.Cohort] = true
	}
	if !seen["A"] || !seen["B"] {
		t.Fatalf("observed cohorts = %v", seen)
	}
}

func TestCohortPolicyRejectsUnsafeOrUndeclaredArms(t *testing.T) {
	ledger, err := OpenCohortLedger(filepath.Join(t.TempDir(), "cohorts"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	cases := []FixedCohortPolicy{
		{PolicyVersion: "policy-v1", CatalogVersion: GoldenResourceCatalogVersion, Mode: FixedCohortMode, MinimumAssignments: 100, MaximumChiSquare: 16.266, Allocations: []CohortAllocation{{Cohort: "control", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 400}, {Cohort: "candidate", ResourceProfileID: "W2_H3G", PropensityBasisPoints: 9600}}},
		{PolicyVersion: "policy-v1", CatalogVersion: GoldenResourceCatalogVersion, Mode: FixedCohortMode, MinimumAssignments: 100, MaximumChiSquare: 16.266, Allocations: []CohortAllocation{{Cohort: "control", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 5000}, {Cohort: "candidate", ResourceProfileID: "UNBOUNDED", PropensityBasisPoints: 5000}}},
		{PolicyVersion: "policy-v1", CatalogVersion: GoldenResourceCatalogVersion, Mode: FixedAAMode, MinimumAssignments: 100, MaximumChiSquare: 10.828, Allocations: []CohortAllocation{{Cohort: "A", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 5000}, {Cohort: "B", ResourceProfileID: "W2_H3G", PropensityBasisPoints: 5000}}},
	}
	for index, policy := range cases {
		if _, _, err := ledger.Assign(testCohortRequest(identifierForIndex("invalid", index), policy)); err == nil {
			t.Fatalf("invalid policy %d was accepted", index)
		}
	}
}

func TestAnalyzeSampleRatioUsesDeclaredNonFiftyFiftyAllocation(t *testing.T) {
	policy := testAAPolicy()
	assignments := syntheticAssignments(policy, map[string]int{"A": 70, "B": 30})
	report := AnalyzeSampleRatio(assignments, policy)
	if report.Status != SampleRatioValid || report.ChiSquare != 0 {
		t.Fatalf("valid ratio = %+v", report)
	}

	mismatch := syntheticAssignments(policy, map[string]int{"A": 95, "B": 5})
	report = AnalyzeSampleRatio(mismatch, policy)
	if report.Status != SampleRatioMismatch || report.ChiSquare <= policy.MaximumChiSquare {
		t.Fatalf("mismatch ratio = %+v", report)
	}

	incomplete := assignments[:20]
	if report := AnalyzeSampleRatio(incomplete, policy); report.Status != SampleRatioInconclusive {
		t.Fatalf("small sample = %+v", report)
	}
	tampered := append([]CohortAssignment(nil), assignments...)
	tampered[0].PropensityBasisPoints = 0
	if report := AnalyzeSampleRatio(tampered, policy); report.Status != SampleRatioInconclusive {
		t.Fatalf("missing propensity = %+v", report)
	}
}

func testAAPolicy() FixedCohortPolicy {
	return FixedCohortPolicy{
		PolicyVersion: "beta-aa-v1", CatalogVersion: GoldenResourceCatalogVersion, Mode: FixedAAMode,
		MinimumAssignments: 100, MaximumChiSquare: 10.828,
		Allocations: []CohortAllocation{
			{Cohort: "A", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 7000},
			{Cohort: "B", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 3000},
		},
	}
}

func testFixedPolicy() FixedCohortPolicy {
	return FixedCohortPolicy{
		PolicyVersion: "beta-fixed-v1", CatalogVersion: GoldenResourceCatalogVersion, Mode: FixedCohortMode,
		MinimumAssignments: 100, MaximumChiSquare: 16.266,
		Allocations: []CohortAllocation{
			{Cohort: "control", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 2500},
			{Cohort: "w2", ResourceProfileID: "W2_H3G", PropensityBasisPoints: 2500},
			{Cohort: "w3", ResourceProfileID: "W3_H4G", PropensityBasisPoints: 2500},
			{Cohort: "w4", ResourceProfileID: "W4_H6G", PropensityBasisPoints: 2500},
		},
	}
}

func testCohortRequest(id string, policy FixedCohortPolicy) CohortAssignmentRequest {
	return CohortAssignmentRequest{
		AssignmentID: id, RepositoryID: "repository-1", MeasurementEpoch: "epoch-1",
		BucketDigest: testDigest("1"), ContextDigest: testDigest("2"), SeedDigest: testDigest("3"), Policy: policy,
	}
}

func testDigest(character string) string {
	if character == "" {
		character = "0"
	}
	return "sha256:" + repeat(character[:1], 64)
}

func identifierForIndex(prefix string, index int) string {
	const digits = "0123456789"
	if index < 10 {
		return prefix + "-" + string(digits[index])
	}
	return identifierForIndex(prefix, index/10) + string(digits[index%10])
}

func syntheticAssignments(policy FixedCohortPolicy, counts map[string]int) []CohortAssignment {
	assignments := make([]CohortAssignment, 0)
	index := 0
	for _, allocation := range policy.Allocations {
		for count := 0; count < counts[allocation.Cohort]; count++ {
			request := testCohortRequest(identifierForIndex("synthetic", index), policy)
			assignments = append(assignments, CohortAssignment{Request: request, Cohort: allocation.Cohort, ResourceProfileID: allocation.ResourceProfileID, PropensityBasisPoints: allocation.PropensityBasisPoints, AssignedAt: time.Now()})
			index++
		}
	}
	return assignments
}
