package adaptivefragment

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// FragmentPlanSchemaVersion identifies the in-memory AF-009 plan contract.
	FragmentPlanSchemaVersion = "buildopt.adaptive/fragment-plan/v1"
	maxPlanningCandidates     = 32
	maxPlanningPredictions    = 1024
)

// PlanDisposition identifies whether BuildOpt may present an exact fragment
// composition or must retain the caller's native Gradle workflow.
type PlanDisposition string

const (
	PlanComposed       PlanDisposition = "COMPOSED"
	PlanNativeRetained PlanDisposition = "NATIVE_GRADLE"
)

// PlanningCandidate is one exact compatible fragment generation supplied by
// the cheap lookup and lifecycle gates. The planner revalidates its state,
// repository scope, expiry, dependency graph and correctness authority.
type PlanningCandidate struct {
	Fragment PersistedFragment
}

// CompositionPrediction is a signed economic prediction for one exact set of
// fragment families. It is not derived by adding isolated percentages. The
// evidence digest binds the external assessment that produced the prediction.
type CompositionPrediction struct {
	Families       []string
	EvidenceSHA256 string
	HorizonBuilds  uint64
	PredictedNetMs int64
}

// PlanRequest contains only facts available before Gradle starts. The native
// workflow remains authoritative when any identity or prediction is ambiguous.
type PlanRequest struct {
	RepositoryScopeSHA256 string
	NativeWorkflowSHA256  string
	DecisionAt            string
	MinimumPredictedNetMs uint64
	Candidates            []PlanningCandidate
	Predictions           []CompositionPrediction
}

// PlannedFragment retains the exact authority of every selected constituent;
// composition never replaces those authorities with a weaker aggregate.
type PlannedFragment struct {
	FamilyID           string    `json:"familyId"`
	RevisionID         string    `json:"revisionId"`
	FragmentGeneration uint64    `json:"fragmentGeneration"`
	Kind               Kind      `json:"kind"`
	Authority          Authority `json:"authority"`
	AuthoritySHA256    string    `json:"authoritySha256"`
}

// FragmentPlan is the deterministic AF-009 result. A native result contains
// no selected fragments, no predicted value and no activation authority.
type FragmentPlan struct {
	SchemaVersion         string            `json:"schemaVersion"`
	PlanID                string            `json:"planId"`
	Disposition           PlanDisposition   `json:"disposition"`
	Reason                string            `json:"reason"`
	RepositoryScopeSHA256 string            `json:"repositoryScopeSha256"`
	NativeWorkflowSHA256  string            `json:"nativeWorkflowSha256"`
	DecisionAt            string            `json:"decisionAt"`
	MinimumPredictedNetMs uint64            `json:"minimumPredictedNetMs"`
	PredictionEvidence    string            `json:"predictionEvidenceSha256,omitempty"`
	PredictionHorizon     uint64            `json:"predictionHorizonBuilds,omitempty"`
	PredictedNetMs        int64             `json:"predictedNetMs"`
	CorrectnessMode       string            `json:"correctnessMode"`
	Selected              []PlannedFragment `json:"selected"`
	RejectedAlternatives  []RejectedPlan    `json:"rejectedAlternatives"`
}

// RejectedPlan makes conflict, dependency and economic fallback visible
// without granting any authority to the rejected families.
type RejectedPlan struct {
	Families []string `json:"families"`
	Reason   string   `json:"reason"`
}

// PlanFragments returns the highest directly predicted compatible composition.
// It never adds isolated effects and never starts Gradle, performs I/O or
// mutates fragment state.
func PlanFragments(request PlanRequest) FragmentPlan {
	base, decisionAt, reason := validatePlanRequest(request)
	if reason != "" {
		return nativeFragmentPlan(base, reason)
	}

	candidates, reason := canonicalPlanningCandidates(request.Candidates, request.RepositoryScopeSHA256, decisionAt)
	if reason != "" {
		return nativeFragmentPlan(base, reason)
	}
	if reason = validateDependencyGraph(candidates); reason != "" {
		return nativeFragmentPlan(base, reason)
	}

	predictions, reason := canonicalPredictions(request.Predictions, candidates)
	if reason != "" {
		return nativeFragmentPlan(base, reason)
	}

	type eligiblePlan struct {
		prediction CompositionPrediction
		key        string
	}
	eligible := make([]eligiblePlan, 0, len(predictions))
	rejected := make([]RejectedPlan, 0, len(predictions))
	for _, prediction := range predictions {
		alternativeReason := validateAlternative(prediction.Families, candidates)
		if alternativeReason == "" && prediction.PredictedNetMs < int64(request.MinimumPredictedNetMs) {
			alternativeReason = "PREDICTED_NET_BELOW_FLOOR"
		}
		if alternativeReason != "" {
			rejected = append(rejected, RejectedPlan{Families: append([]string{}, prediction.Families...), Reason: alternativeReason})
			continue
		}
		eligible = append(eligible, eligiblePlan{prediction: prediction, key: strings.Join(prediction.Families, ",")})
	}
	sortRejectedPlans(rejected)
	if len(eligible) == 0 {
		base.RejectedAlternatives = rejected
		if len(predictions) > 0 && allRejectedForFloor(rejected) {
			return nativeFragmentPlanWithRejected(base, "PREDICTED_NET_BELOW_FLOOR", rejected)
		}
		return nativeFragmentPlanWithRejected(base, "NO_ELIGIBLE_COMPOSITION", rejected)
	}

	sort.Slice(eligible, func(left, right int) bool {
		if eligible[left].prediction.PredictedNetMs != eligible[right].prediction.PredictedNetMs {
			return eligible[left].prediction.PredictedNetMs > eligible[right].prediction.PredictedNetMs
		}
		return eligible[left].key < eligible[right].key
	})
	selectedPrediction := eligible[0].prediction
	selected := make([]PlannedFragment, 0, len(selectedPrediction.Families))
	for _, family := range selectedPrediction.Families {
		fragment := candidates[family].Fragment
		selected = append(selected, PlannedFragment{
			FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
			FragmentGeneration: fragment.Generation, Kind: fragment.Kind,
			Authority: fragment.Authority, AuthoritySHA256: fragment.AuthoritySHA256,
		})
	}
	base.Disposition = PlanComposed
	base.Reason = "POSITIVE_EXACT_COMPOSITION"
	base.PredictionEvidence = selectedPrediction.EvidenceSHA256
	base.PredictionHorizon = selectedPrediction.HorizonBuilds
	base.PredictedNetMs = selectedPrediction.PredictedNetMs
	base.CorrectnessMode = "CONJUNCTION_OF_CONSTITUENT_AUTHORITIES"
	base.Selected = selected
	base.RejectedAlternatives = rejected
	base.PlanID = fragmentPlanID(base)
	return base
}

func validatePlanRequest(request PlanRequest) (FragmentPlan, time.Time, string) {
	base := FragmentPlan{
		SchemaVersion:         FragmentPlanSchemaVersion,
		RepositoryScopeSHA256: request.RepositoryScopeSHA256,
		NativeWorkflowSHA256:  request.NativeWorkflowSHA256,
		DecisionAt:            request.DecisionAt,
		MinimumPredictedNetMs: request.MinimumPredictedNetMs,
		CorrectnessMode:       "NATIVE_GRADLE",
		Selected:              []PlannedFragment{}, RejectedAlternatives: []RejectedPlan{},
	}
	decisionAt, err := parseUTC(request.DecisionAt)
	if !validSHA(request.RepositoryScopeSHA256) || !validSHA(request.NativeWorkflowSHA256) || err != nil ||
		request.MinimumPredictedNetMs == 0 || request.MinimumPredictedNetMs > maxEconomicComponentMs ||
		len(request.Candidates) == 0 || len(request.Candidates) > maxPlanningCandidates ||
		len(request.Predictions) > maxPlanningPredictions {
		return base, time.Time{}, "INVALID_PLANNING_INPUT"
	}
	if len(request.Predictions) == 0 {
		return base, time.Time{}, "EXACT_COMPOSITION_PREDICTION_UNAVAILABLE"
	}
	return base, decisionAt, ""
}

func canonicalPlanningCandidates(input []PlanningCandidate, repositoryScope string, decisionAt time.Time) (map[string]PlanningCandidate, string) {
	result := make(map[string]PlanningCandidate, len(input))
	for _, candidate := range input {
		fragment := candidate.Fragment
		if err := validatePersistedFragment(fragment); err != nil || fragment.RepositoryScopeSHA256 != repositoryScope ||
			(fragment.State != StateQualified && fragment.State != StateActive) {
			return nil, "CANDIDATE_NOT_QUALIFIED"
		}
		expiresAt, _ := parseUTC(fragment.EvidenceExpiresAt)
		if !expiresAt.After(decisionAt) {
			return nil, "CANDIDATE_EVIDENCE_EXPIRED"
		}
		if _, exists := result[fragment.FamilyID]; exists {
			return nil, "AMBIGUOUS_CANDIDATE_REVISION"
		}
		result[fragment.FamilyID] = PlanningCandidate{Fragment: clonePersistedFragments([]PersistedFragment{fragment})[0]}
	}
	return result, ""
}

func validateDependencyGraph(candidates map[string]PlanningCandidate) string {
	for _, candidate := range candidates {
		for _, required := range candidate.Fragment.Requires {
			if _, exists := candidates[required]; !exists {
				return "DEPENDENCY_UNAVAILABLE"
			}
		}
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(family string) bool {
		if visiting[family] {
			return false
		}
		if visited[family] {
			return true
		}
		visiting[family] = true
		for _, required := range candidates[family].Fragment.Requires {
			if !visit(required) {
				return false
			}
		}
		delete(visiting, family)
		visited[family] = true
		return true
	}
	for family := range candidates {
		if !visit(family) {
			return "DEPENDENCY_CYCLE"
		}
	}
	return ""
}

func canonicalPredictions(input []CompositionPrediction, candidates map[string]PlanningCandidate) ([]CompositionPrediction, string) {
	result := make([]CompositionPrediction, 0, len(input))
	seen := map[string]CompositionPrediction{}
	for _, prediction := range input {
		families, err := normalizedDigests(prediction.Families)
		if err != nil || len(families) == 0 || !validSHA(prediction.EvidenceSHA256) || prediction.HorizonBuilds == 0 ||
			prediction.HorizonBuilds > maxEconomicHorizon || prediction.PredictedNetMs < -maxEconomicComponentMs ||
			prediction.PredictedNetMs > maxEconomicComponentMs {
			return nil, "INVALID_COMPOSITION_PREDICTION"
		}
		for _, family := range families {
			if _, exists := candidates[family]; !exists {
				return nil, "PREDICTION_REFERENCES_UNKNOWN_FRAGMENT"
			}
		}
		prediction.Families = families
		key := strings.Join(families, ",")
		if previous, exists := seen[key]; exists {
			if previous.EvidenceSHA256 != prediction.EvidenceSHA256 || previous.HorizonBuilds != prediction.HorizonBuilds ||
				previous.PredictedNetMs != prediction.PredictedNetMs {
				return nil, "AMBIGUOUS_COMPOSITION_PREDICTION"
			}
			continue
		}
		seen[key] = prediction
		result = append(result, prediction)
	}
	sort.Slice(result, func(left, right int) bool {
		return strings.Join(result[left].Families, ",") < strings.Join(result[right].Families, ",")
	})
	return result, ""
}

func validateAlternative(families []string, candidates map[string]PlanningCandidate) string {
	selected := make(map[string]bool, len(families))
	for _, family := range families {
		selected[family] = true
	}
	for _, family := range families {
		fragment := candidates[family].Fragment
		for _, required := range fragment.Requires {
			if !selected[required] {
				return "DEPENDENCY_NOT_SELECTED"
			}
		}
		for _, conflict := range fragment.ConflictsWith {
			if selected[conflict] {
				return "MUTUAL_EXCLUSION"
			}
		}
		for otherFamily := range selected {
			if contains(candidates[otherFamily].Fragment.ConflictsWith, family) {
				return "MUTUAL_EXCLUSION"
			}
		}
	}
	return ""
}

func nativeFragmentPlan(base FragmentPlan, reason string) FragmentPlan {
	return nativeFragmentPlanWithRejected(base, reason, []RejectedPlan{})
}

func nativeFragmentPlanWithRejected(base FragmentPlan, reason string, rejected []RejectedPlan) FragmentPlan {
	base.Disposition = PlanNativeRetained
	base.Reason = reason
	base.PredictionEvidence = ""
	base.PredictionHorizon = 0
	base.PredictedNetMs = 0
	base.CorrectnessMode = "NATIVE_GRADLE"
	base.Selected = []PlannedFragment{}
	base.RejectedAlternatives = rejected
	base.PlanID = fragmentPlanID(base)
	return base
}

func sortRejectedPlans(rejected []RejectedPlan) {
	sort.Slice(rejected, func(left, right int) bool {
		leftKey := strings.Join(rejected[left].Families, ",") + ":" + rejected[left].Reason
		rightKey := strings.Join(rejected[right].Families, ",") + ":" + rejected[right].Reason
		return leftKey < rightKey
	})
}

func allRejectedForFloor(rejected []RejectedPlan) bool {
	if len(rejected) == 0 {
		return false
	}
	for _, alternative := range rejected {
		if alternative.Reason != "PREDICTED_NET_BELOW_FLOOR" {
			return false
		}
	}
	return true
}

func fragmentPlanID(plan FragmentPlan) string {
	rows := []string{
		string(plan.Disposition), plan.Reason, plan.RepositoryScopeSHA256,
		plan.NativeWorkflowSHA256, plan.DecisionAt,
		plan.PredictionEvidence, string(plan.CorrectnessMode),
	}
	for _, selected := range plan.Selected {
		rows = append(rows, selected.FamilyID, selected.RevisionID,
			strconv.FormatUint(selected.FragmentGeneration, 10), selected.AuthoritySHA256,
			string(selected.Kind), string(selected.Authority))
	}
	for _, rejected := range plan.RejectedAlternatives {
		rows = append(rows, strings.Join(rejected.Families, ","), rejected.Reason)
	}
	rows = append(rows,
		strconv.FormatUint(plan.MinimumPredictedNetMs, 10),
		strconv.FormatUint(plan.PredictionHorizon, 10),
		strconv.FormatInt(plan.PredictedNetMs, 10),
	)
	return digest("buildopt-adaptive-fragment-plan-v1", rows...)
}
