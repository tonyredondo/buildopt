package runtimeoptimizer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestReplayBoundedBanditPolicyCases(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "specs", "bandit-policy-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Cases []struct {
			ID string `json:"id"`
			BanditReplayCase
			ExpectedMode    string `json:"expectedMode"`
			ExpectedArm     string `json:"expectedArm"`
			ExpectedUpdate  bool   `json:"expectedUpdate"`
			ExpectedOutcome string `json:"expectedOutcome"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(content, &spec); err != nil {
		t.Fatal(err)
	}
	if len(spec.Cases) != 15 {
		t.Fatalf("cases = %d", len(spec.Cases))
	}
	for _, testCase := range spec.Cases {
		actual := ReplayBoundedBanditCase(testCase.BanditReplayCase)
		expected := BanditReplayResult{Mode: testCase.ExpectedMode, Arm: testCase.ExpectedArm, Update: testCase.ExpectedUpdate, Outcome: testCase.ExpectedOutcome}
		if actual != expected {
			t.Errorf("%s = %+v, want %+v", testCase.ID, actual, expected)
		}
	}
}

func TestBanditSelectionPropensitiesMatchAllTenThousandPoints(t *testing.T) {
	rewards := map[string][]float64{
		"STABLE_CONTROL": repeatedRewards(0, 20),
		"W2_H3G":         repeatedRewards(100, 20),
		"W3_H4G":         repeatedRewards(900, 20),
		"W4_H6G":         repeatedRewards(-100, 20),
	}
	eligible := []string{"STABLE_CONTROL", "W2_H3G", "W3_H4G", "W4_H6G"}
	counts := map[string]int{}
	declared := map[string]int{}
	for point := 0; point < BasisPointTotal; point++ {
		arm, propensity, _ := selectBanditArm(rewards, eligible, 1000, point)
		counts[arm]++
		if previous, ok := declared[arm]; ok && previous != propensity {
			t.Fatalf("%s propensity changed from %d to %d", arm, previous, propensity)
		}
		declared[arm] = propensity
	}
	if !reflect.DeepEqual(counts, declared) {
		t.Fatalf("observed = %v, declared = %v", counts, declared)
	}
	if counts["STABLE_CONTROL"] != MinimumControlBasisPoints || counts["W3_H4G"] <= counts["W2_H3G"] {
		t.Fatalf("unexpected bounded allocation %v", counts)
	}
}

func TestBanditEnginePersistsReadyAssignmentAndExactlyOnceOutcome(t *testing.T) {
	now := time.Date(2026, 7, 31, 15, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "bandit")
	engine, err := OpenBanditEngine(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	request := testBanditRequest("bandit-assignment-1", "epoch-1")
	state := engine.bucket(request.Bucket)
	state.Rewards["STABLE_CONTROL"] = repeatedRewards(0, 20)
	state.Rewards["W2_H3G"] = repeatedRewards(100, 20)
	state.Rewards["W3_H4G"] = repeatedRewards(900, 20)
	engine.state.Buckets[bucketKey(request.Bucket)] = state
	if err := engine.persist(); err != nil {
		t.Fatal(err)
	}

	assignment, created, err := engine.Assign(request)
	if err != nil || !created || assignment.Mode != BanditMode || assignment.PropensityBasisPoints <= 0 {
		t.Fatalf("assignment = %+v/%v/%v", assignment, created, err)
	}
	if !isGoldenArm(assignment.ResourceProfileID) {
		t.Fatalf("unsafe arm %q", assignment.ResourceProfileID)
	}
	info, err := os.Stat(filepath.Join(root, "bandit.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("state = %v/%v", info, err)
	}
	reopened, err := OpenBanditEngine(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	repeat, created, err := reopened.Assign(request)
	if err != nil || created || !reflect.DeepEqual(repeat, assignment) {
		t.Fatalf("repeat = %+v/%v/%v", repeat, created, err)
	}

	outcome := BanditOutcome{
		OutcomeID: "outcome-1", AssignmentID: request.AssignmentID, Bucket: request.Bucket,
		PropensityBasisPoints: assignment.PropensityBasisPoints, CompletedAt: now.Add(BanditMaximumOutcomeDelay), Guardrail: "NONE",
		Reward: RewardComponents{Complete: true, BaselineCustomerVisibleBuildMS: 1000, CustomerVisibleBuildMS: 800, CIQueuePenaltyMS: 20},
	}
	disposition, updated, err := reopened.RecordOutcome(outcome)
	if err != nil || !updated || disposition.Status != "UPDATED" || disposition.Reward != 180 {
		t.Fatalf("outcome = %+v/%v/%v", disposition, updated, err)
	}
	duplicate := outcome
	duplicate.OutcomeID = "outcome-duplicate"
	if disposition, updated, err := reopened.RecordOutcome(duplicate); err != nil || updated || disposition.OutcomeID != "outcome-1" {
		t.Fatalf("duplicate = %+v/%v/%v", disposition, updated, err)
	}
}

func TestBanditEngineRejectsMissingPropensityLatePartialAndUnsafeArm(t *testing.T) {
	now := time.Date(2026, 7, 31, 15, 0, 0, 0, time.UTC)
	engine, err := OpenBanditEngine(filepath.Join(t.TempDir(), "bandit"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	unsafe := testBanditRequest("unsafe", "epoch-1")
	unsafe.EligibleArms = append(unsafe.EligibleArms, "ARBITRARY")
	if _, _, err := engine.Assign(unsafe); err == nil {
		t.Fatal("unsafe arm was accepted")
	}
	ready := engine.bucket(unsafe.Bucket)
	ready.Rewards["W2_H3G"] = repeatedRewards(100, 20)
	ready.Rewards["W3_H4G"] = repeatedRewards(900, 20)
	engine.state.Buckets[bucketKey(unsafe.Bucket)] = ready

	for index, mutate := range []func(*BanditOutcome){
		func(outcome *BanditOutcome) { outcome.PropensityBasisPoints = 0 },
		func(outcome *BanditOutcome) { outcome.CompletedAt = now.Add(25 * time.Hour) },
		func(outcome *BanditOutcome) { outcome.Reward.Complete = false },
	} {
		request := testBanditRequest(identifierForIndex("invalid-outcome", index), "epoch-1")
		assignment, _, err := engine.Assign(request)
		if err != nil {
			t.Fatal(err)
		}
		outcome := BanditOutcome{OutcomeID: identifierForIndex("outcome", index), AssignmentID: request.AssignmentID, Bucket: request.Bucket, PropensityBasisPoints: assignment.PropensityBasisPoints, CompletedAt: now.Add(time.Hour), Guardrail: "NONE", Reward: RewardComponents{Complete: true, BaselineCustomerVisibleBuildMS: 1000, CustomerVisibleBuildMS: 900}}
		mutate(&outcome)
		if disposition, updated, err := engine.RecordOutcome(outcome); err != nil || updated || disposition.Status != "INCONCLUSIVE" {
			t.Fatalf("case %d = %+v/%v/%v", index, disposition, updated, err)
		}
	}
}

func TestBanditBucketsResetWithoutMixingEras(t *testing.T) {
	engine, err := OpenBanditEngine(filepath.Join(t.TempDir(), "bandit"), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	oldRequest := testBanditRequest("old", "epoch-1")
	state := engine.bucket(oldRequest.Bucket)
	state.Rewards["W2_H3G"] = repeatedRewards(100, 20)
	state.Rewards["W3_H4G"] = repeatedRewards(900, 20)
	engine.state.Buckets[bucketKey(oldRequest.Bucket)] = state
	oldAssignment, _, err := engine.Assign(oldRequest)
	if err != nil || oldAssignment.Mode != BanditMode {
		t.Fatalf("old = %+v/%v", oldAssignment, err)
	}
	newRequest := testBanditRequest("new", "epoch-2")
	newAssignment, _, err := engine.Assign(newRequest)
	if err != nil || newAssignment.Mode != FixedCohortMode || newAssignment.Disposition != "PENDING_SAMPLE" {
		t.Fatalf("new = %+v/%v", newAssignment, err)
	}
	reset := testBanditRequest("reset", "epoch-1")
	reset.ResetReason = "DRIFT_THRESHOLD_EXCEEDED"
	resetAssignment, _, err := engine.Assign(reset)
	if err != nil || resetAssignment.Mode != FixedAAMode || resetAssignment.Disposition != "RESET" {
		t.Fatalf("reset = %+v/%v", resetAssignment, err)
	}
}

func TestRecordFixedOutcomeValidatesAssignmentBinding(t *testing.T) {
	now := time.Date(2026, 7, 31, 15, 0, 0, 0, time.UTC)
	cohortLedger, err := OpenCohortLedger(filepath.Join(t.TempDir(), "cohort"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	assignment, _, err := cohortLedger.Assign(testCohortRequest("fixed-assignment", testFixedPolicy()))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := OpenBanditEngine(filepath.Join(t.TempDir(), "bandit"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	outcome := FixedCohortOutcome{OutcomeID: "fixed-outcome", CompletedAt: now.Add(time.Hour), Guardrail: "NONE", Reward: RewardComponents{Complete: true, BaselineCustomerVisibleBuildMS: 1000, CustomerVisibleBuildMS: 900}}
	if disposition, updated, err := engine.RecordFixedOutcome(assignment, outcome); err != nil || !updated || disposition.Status != "UPDATED" {
		t.Fatalf("fixed = %+v/%v/%v", disposition, updated, err)
	}
	tampered := assignment
	tampered.ResourceProfileID = "W4_H6G"
	tampered.Request.AssignmentID = "fixed-tampered"
	if _, _, err := engine.RecordFixedOutcome(tampered, outcome); err == nil {
		t.Fatal("tampered fixed assignment was accepted")
	}
}

func testBanditRequest(id, epoch string) BanditAssignmentRequest {
	return BanditAssignmentRequest{
		AssignmentID: id, ContextDigest: testDigest("4"), SeedDigest: testDigest("5"),
		Bucket:       BanditBucket{RepositoryID: "repository-1", MeasurementEpoch: epoch, PolicyVersion: BanditPolicyVersion, CatalogVersion: GoldenResourceCatalogVersion, BucketDigest: testDigest("6")},
		EligibleArms: []string{"STABLE_CONTROL", "W2_H3G", "W3_H4G"}, EpsilonBasisPoints: 500, AARatioStatus: SampleRatioValid, ResetReason: "NONE",
	}
}

func repeatedRewards(value float64, count int) []float64 {
	result := make([]float64, count)
	for index := range result {
		result[index] = value
	}
	return result
}

func isGoldenArm(arm string) bool {
	for _, profile := range GoldenResourceProfiles() {
		if arm == profile.ProfileID {
			return true
		}
	}
	return false
}
