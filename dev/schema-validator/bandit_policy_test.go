package schemavalidator

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

type banditPolicySpec struct {
	SchemaVersion            string   `json:"schemaVersion"`
	PolicyVersion            string   `json:"policyVersion"`
	CatalogVersion           string   `json:"catalogVersion"`
	Arms                     []string `json:"arms"`
	PreOutcomeFeatures       []string `json:"preOutcomeFeatures"`
	ForbiddenOutcomeFeatures []string `json:"forbiddenOutcomeFeatures"`
	Reward                   struct {
		Primary                  string   `json:"primary"`
		Penalties                []string `json:"penalties"`
		NonCompensableGuardrails []string `json:"nonCompensableGuardrails"`
	} `json:"reward"`
	MinimumValidOutcomesPerCandidate   int                `json:"minimumValidOutcomesPerCandidate"`
	MinimumStableControlPercent        int                `json:"minimumStableControlPercent"`
	MinimumEpsilonPercent              int                `json:"minimumEpsilonPercent"`
	MaximumEpsilonPercent              int                `json:"maximumEpsilonPercent"`
	MaximumOutcomeDelayHours           int                `json:"maximumOutcomeDelayHours"`
	TrimPercent                        int                `json:"trimPercent"`
	ShrinkageControlPseudoObservations int                `json:"shrinkageControlPseudoObservations"`
	Cases                              []banditReplayCase `json:"cases"`
}

type banditReplayCase struct {
	ID                    string               `json:"id"`
	Trigger               string               `json:"trigger"`
	AAValid               bool                 `json:"aaValid"`
	CandidateSamplesReady bool                 `json:"candidateSamplesReady"`
	PropensityPresent     bool                 `json:"propensityPresent"`
	OutcomeDelayHours     int                  `json:"outcomeDelayHours"`
	DuplicateOutcome      bool                 `json:"duplicateOutcome"`
	ResetReason           string               `json:"resetReason"`
	Guardrail             string               `json:"guardrail"`
	EpsilonPercent        int                  `json:"epsilonPercent"`
	RandomPercent         int                  `json:"randomPercent"`
	EligibleArms          []string             `json:"eligibleArms"`
	Rewards               map[string][]float64 `json:"rewards"`
	ExpectedMode          string               `json:"expectedMode"`
	ExpectedArm           string               `json:"expectedArm"`
	ExpectedUpdate        bool                 `json:"expectedUpdate"`
	ExpectedOutcome       string               `json:"expectedOutcome"`
}

type banditReplayResult struct {
	Mode    string
	Arm     string
	Update  bool
	Outcome string
}

func TestBanditPolicyV1(t *testing.T) {
	t.Parallel()

	spec := loadBanditPolicySpec(t)
	wantArms := []string{"STABLE_CONTROL", "W2_H3G", "W3_H4G", "W4_H6G"}
	if spec.SchemaVersion != "buildopt.specs/bandit-policy/v1" ||
		spec.PolicyVersion != "beta-bandit-v1" ||
		spec.CatalogVersion != "golden-linux-amd64-4c-16g-v1" ||
		!slices.Equal(spec.Arms, wantArms) {
		t.Errorf("invalid policy identity/catalog")
	}
	if spec.MinimumValidOutcomesPerCandidate != 20 ||
		spec.MinimumStableControlPercent != 5 ||
		spec.MinimumEpsilonPercent != 2 ||
		spec.MaximumEpsilonPercent != 10 ||
		spec.MaximumOutcomeDelayHours != 24 ||
		spec.TrimPercent != 10 ||
		spec.ShrinkageControlPseudoObservations != 5 {
		t.Errorf("invalid policy bounds: %+v", spec)
	}
	requiredFeatures := []string{
		"runnerCpuCount",
		"taskGraphClass",
		"changeClass",
		"historicalHitRateClass",
		"daemonJitState",
		"contentionClass",
	}
	for _, feature := range requiredFeatures {
		if !slices.Contains(spec.PreOutcomeFeatures, feature) {
			t.Errorf("required feature %q is absent", feature)
		}
	}
	for _, future := range []string{"actualHits", "finalDuration", "finalResult"} {
		if slices.Contains(spec.PreOutcomeFeatures, future) ||
			!slices.Contains(spec.ForbiddenOutcomeFeatures, future) {
			t.Errorf("future feature %q is not forbidden", future)
		}
	}
	if spec.Reward.Primary != "NEGATIVE_CUSTOMER_VISIBLE_BUILD_MS" ||
		len(spec.Reward.Penalties) != 5 ||
		len(spec.Reward.NonCompensableGuardrails) != 5 {
		t.Errorf("invalid reward definition: %+v", spec.Reward)
	}
	assertBanditResourceProfiles(t, spec)
	if len(spec.Cases) != 15 {
		t.Fatalf("case count = %d, want 15", len(spec.Cases))
	}
	seen := make(map[string]struct{}, len(spec.Cases))
	for _, testCase := range spec.Cases {
		if _, duplicate := seen[testCase.ID]; testCase.ID == "" || duplicate {
			t.Fatalf("empty or duplicate case ID %q", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
		actual := replayBanditCase(spec, testCase)
		expected := banditReplayResult{
			Mode:    testCase.ExpectedMode,
			Arm:     testCase.ExpectedArm,
			Update:  testCase.ExpectedUpdate,
			Outcome: testCase.ExpectedOutcome,
		}
		if actual != expected {
			t.Errorf("%s result = %+v, want %+v", testCase.ID, actual, expected)
		}
	}
}

func loadBanditPolicySpec(t *testing.T) banditPolicySpec {
	t.Helper()
	path := filepath.Join(findRepositoryRoot(t), "specs", "bandit-policy-v1.json")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var spec banditPolicySpec
	if err := decoder.Decode(&spec); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing data: %v", path, err)
	}
	return spec
}

func assertBanditResourceProfiles(t *testing.T, spec banditPolicySpec) {
	t.Helper()
	root := findRepositoryRoot(t)
	for _, arm := range spec.Arms {
		name := map[string]string{
			"STABLE_CONTROL": "stable-control.json",
			"W2_H3G":         "w2-h3g.json",
			"W3_H4G":         "w3-h4g.json",
			"W4_H6G":         "w4-h6g.json",
		}[arm]
		content, err := os.ReadFile(filepath.Join(
			root,
			"contracts",
			"jsonschema",
			"testdata",
			"foundation-contracts.v1",
			"resource-profile",
			"valid",
			name,
		))
		if err != nil {
			t.Fatalf("read profile %s: %v", arm, err)
		}
		var profile struct {
			CatalogVersion string `json:"catalogVersion"`
			ProfileID      string `json:"profileId"`
			Eligibility    struct {
				Eligible bool `json:"eligible"`
			} `json:"eligibility"`
		}
		if err := json.Unmarshal(content, &profile); err != nil {
			t.Fatalf("decode profile %s: %v", arm, err)
		}
		if profile.CatalogVersion != spec.CatalogVersion ||
			profile.ProfileID != arm ||
			!profile.Eligibility.Eligible {
			t.Errorf("invalid profile binding for %s", arm)
		}
	}
}

func replayBanditCase(
	spec banditPolicySpec,
	testCase banditReplayCase,
) banditReplayResult {
	control := banditReplayResult{
		Mode:    "FIXED_COHORT",
		Arm:     "STABLE_CONTROL",
		Outcome: "INCONCLUSIVE",
	}
	if testCase.Guardrail != "NONE" {
		control.Outcome = "SUSPENDED_ROLLBACK"
		return control
	}
	if testCase.ResetReason != "NONE" {
		control.Mode = "FIXED_AA"
		control.Outcome = "RESET"
		return control
	}
	if !testCase.AAValid {
		control.Mode = "FIXED_AA"
		return control
	}
	if !testCase.PropensityPresent {
		return control
	}
	if testCase.Trigger == "OUTCOME" {
		result := banditReplayResult{
			Mode:    "BANDIT",
			Arm:     firstCandidate(testCase.EligibleArms),
			Outcome: "INCONCLUSIVE",
		}
		if testCase.DuplicateOutcome ||
			testCase.OutcomeDelayHours > spec.MaximumOutcomeDelayHours {
			return result
		}
		result.Update = true
		result.Outcome = "UPDATED"
		return result
	}
	if !testCase.CandidateSamplesReady {
		control.Outcome = "PENDING_SAMPLE"
		return control
	}
	result := banditReplayResult{Mode: "BANDIT", Outcome: "ASSIGNED"}
	if testCase.EpsilonPercent < spec.MinimumEpsilonPercent ||
		testCase.EpsilonPercent > spec.MaximumEpsilonPercent ||
		testCase.RandomPercent < 0 ||
		testCase.RandomPercent >= 100 {
		return control
	}
	if testCase.RandomPercent < spec.MinimumStableControlPercent {
		result.Arm = "STABLE_CONTROL"
		result.Outcome = "ASSIGNED_CONTROL"
		return result
	}
	if testCase.RandomPercent <
		spec.MinimumStableControlPercent+testCase.EpsilonPercent {
		result.Arm = firstCandidate(testCase.EligibleArms)
		result.Outcome = "ASSIGNED_EXPLORATION"
		return result
	}
	result.Arm = greedyBanditArm(spec, testCase)
	return result
}

func firstCandidate(eligible []string) string {
	for _, arm := range eligible {
		if arm != "STABLE_CONTROL" {
			return arm
		}
	}
	return "STABLE_CONTROL"
}

func greedyBanditArm(
	spec banditPolicySpec,
	testCase banditReplayCase,
) string {
	controlMean := mean(testCase.Rewards["STABLE_CONTROL"])
	bestArm := "STABLE_CONTROL"
	bestPrediction := controlMean
	for _, arm := range spec.Arms {
		if arm == "STABLE_CONTROL" || !slices.Contains(testCase.EligibleArms, arm) {
			continue
		}
		rewards := append([]float64(nil), testCase.Rewards[arm]...)
		sort.Float64s(rewards)
		trim := len(rewards) * spec.TrimPercent / 100
		if trim > 0 && trim*2 < len(rewards) {
			rewards = rewards[trim : len(rewards)-trim]
		}
		numerator := sum(rewards) +
			float64(spec.ShrinkageControlPseudoObservations)*controlMean
		prediction := numerator /
			float64(len(rewards)+spec.ShrinkageControlPseudoObservations)
		if prediction > bestPrediction {
			bestArm = arm
			bestPrediction = prediction
		}
	}
	return bestArm
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sum(values) / float64(len(values))
}

func sum(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}
