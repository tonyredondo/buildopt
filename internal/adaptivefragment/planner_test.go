package adaptivefragment

import (
	"reflect"
	"sort"
	"testing"
)

func TestFragmentPlannerSelectsHighestExactCompatibleComposition(t *testing.T) {
	request, families := plannerFixture(t)
	before := clonePlanRequest(request)
	plan := PlanFragments(request)
	if plan.Disposition != PlanComposed || plan.Reason != "POSITIVE_EXACT_COMPOSITION" || plan.PredictedNetMs != 5000 {
		t.Fatalf("plan = %+v", plan)
	}
	wantFamilies := []string{families["locality"], families["patch"]}
	sort.Strings(wantFamilies)
	if selectedFamilies(plan) == nil || !reflect.DeepEqual(selectedFamilies(plan), wantFamilies) {
		t.Fatalf("selected families = %v, want %v", selectedFamilies(plan), wantFamilies)
	}
	if plan.CorrectnessMode != "CONJUNCTION_OF_CONSTITUENT_AUTHORITIES" || len(plan.Selected) != 2 {
		t.Fatalf("correctness authority = %+v", plan)
	}
	for _, selected := range plan.Selected {
		if selected.Authority == "" || !validSHA(selected.AuthoritySHA256) {
			t.Fatalf("selected authority was weakened: %+v", selected)
		}
	}
	if len(plan.RejectedAlternatives) != 3 {
		t.Fatalf("rejected alternatives = %+v", plan.RejectedAlternatives)
	}
	wantReasons := map[string]bool{
		"DEPENDENCY_NOT_SELECTED":   true,
		"MUTUAL_EXCLUSION":          true,
		"PREDICTED_NET_BELOW_FLOOR": true,
	}
	for _, rejected := range plan.RejectedAlternatives {
		delete(wantReasons, rejected.Reason)
	}
	if len(wantReasons) != 0 {
		t.Fatalf("missing rejected reasons: %v", wantReasons)
	}
	if !reflect.DeepEqual(request, before) {
		t.Fatal("planner mutated its request")
	}
}

func TestFragmentPlannerIsOrderIndependent(t *testing.T) {
	request, _ := plannerFixture(t)
	want := PlanFragments(request)
	for left, right := 0, len(request.Candidates)-1; left < right; left, right = left+1, right-1 {
		request.Candidates[left], request.Candidates[right] = request.Candidates[right], request.Candidates[left]
	}
	for left, right := 0, len(request.Predictions)-1; left < right; left, right = left+1, right-1 {
		request.Predictions[left], request.Predictions[right] = request.Predictions[right], request.Predictions[left]
	}
	got := PlanFragments(request)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reordered plan = %+v, want %+v", got, want)
	}
}

func TestFragmentPlannerUsesCanonicalTieBreak(t *testing.T) {
	request, families := plannerFixture(t)
	request.Predictions = []CompositionPrediction{
		plannerPrediction(2500, families["task"]),
		plannerPrediction(2500, families["locality"]),
	}
	plan := PlanFragments(request)
	want := families["locality"]
	if families["task"] < want {
		want = families["task"]
	}
	if plan.Disposition != PlanComposed || !reflect.DeepEqual(selectedFamilies(plan), []string{want}) {
		t.Fatalf("tie plan = %+v, want %s", plan, want)
	}
}

func TestFragmentPlannerRetainsNativeOnUnsafeOrAmbiguousInput(t *testing.T) {
	base, families := plannerFixture(t)
	tests := []struct {
		name   string
		mutate func(PlanRequest) PlanRequest
		reason string
	}{
		{
			name: "missing exact composition prediction",
			mutate: func(request PlanRequest) PlanRequest {
				request.Predictions = []CompositionPrediction{}
				return request
			},
			reason: "EXACT_COMPOSITION_PREDICTION_UNAVAILABLE",
		},
		{
			name: "all predictions below floor",
			mutate: func(request PlanRequest) PlanRequest {
				request.Predictions = []CompositionPrediction{plannerPrediction(99, families["locality"])}
				return request
			},
			reason: "PREDICTED_NET_BELOW_FLOOR",
		},
		{
			name: "ambiguous exact prediction",
			mutate: func(request PlanRequest) PlanRequest {
				first := plannerPrediction(1000, families["locality"])
				second := plannerPrediction(1001, families["locality"])
				request.Predictions = []CompositionPrediction{first, second}
				return request
			},
			reason: "AMBIGUOUS_COMPOSITION_PREDICTION",
		},
		{
			name: "duplicate candidate revision",
			mutate: func(request PlanRequest) PlanRequest {
				request.Candidates = append(request.Candidates, request.Candidates[0])
				return request
			},
			reason: "AMBIGUOUS_CANDIDATE_REVISION",
		},
		{
			name: "dependency unavailable",
			mutate: func(request PlanRequest) PlanRequest {
				filtered := []PlanningCandidate{}
				for _, candidate := range request.Candidates {
					if candidate.Fragment.FamilyID != families["materialization"] {
						filtered = append(filtered, candidate)
					}
				}
				request.Candidates = filtered
				request.Predictions = []CompositionPrediction{plannerPrediction(1000, families["locality"])}
				return request
			},
			reason: "DEPENDENCY_UNAVAILABLE",
		},
		{
			name: "expired evidence",
			mutate: func(request PlanRequest) PlanRequest {
				request.Candidates[0].Fragment.EvidenceExpiresAt = request.DecisionAt
				return request
			},
			reason: "CANDIDATE_EVIDENCE_EXPIRED",
		},
		{
			name: "unqualified lifecycle",
			mutate: func(request PlanRequest) PlanRequest {
				request.Candidates[0].Fragment.State = StateShadow
				return request
			},
			reason: "CANDIDATE_NOT_QUALIFIED",
		},
		{
			name: "unknown predicted fragment",
			mutate: func(request PlanRequest) PlanRequest {
				request.Predictions = []CompositionPrediction{plannerPrediction(1000, digest("unknown", "family"))}
				return request
			},
			reason: "PREDICTION_REFERENCES_UNKNOWN_FRAGMENT",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := PlanFragments(test.mutate(clonePlanRequest(base)))
			if plan.Disposition != PlanNativeRetained || plan.Reason != test.reason || len(plan.Selected) != 0 ||
				plan.PredictedNetMs != 0 || plan.CorrectnessMode != "NATIVE_GRADLE" {
				t.Fatalf("native plan = %+v, want reason %s", plan, test.reason)
			}
		})
	}
}

func TestFragmentPlannerRetainsNativeOnDependencyCycle(t *testing.T) {
	request, _ := plannerFixture(t)
	first := plannerContract(t, KindTaskContract, AuthorityReviewedAdapter, "cycle-first", nil, nil)
	second := plannerContract(t, KindTaskContract, AuthorityReviewedAdapter, "cycle-second", nil, nil)
	first = plannerContract(t, KindTaskContract, AuthorityReviewedAdapter, "cycle-first", []string{second.FamilyID}, nil)
	second = plannerContract(t, KindTaskContract, AuthorityReviewedAdapter, "cycle-second", []string{first.FamilyID}, nil)
	request.Candidates = []PlanningCandidate{{Fragment: plannerPersisted(first)}, {Fragment: plannerPersisted(second)}}
	request.Predictions = []CompositionPrediction{plannerPrediction(3000, first.FamilyID, second.FamilyID)}
	plan := PlanFragments(request)
	if plan.Disposition != PlanNativeRetained || plan.Reason != "DEPENDENCY_CYCLE" {
		t.Fatalf("cycle plan = %+v", plan)
	}
}

func plannerFixture(t *testing.T) (PlanRequest, map[string]string) {
	t.Helper()
	materialization := plannerContract(t, KindOutputMaterialization, AuthorityVerifiedProducer, "materialization", nil, nil)
	subgraph := plannerContract(t, KindSubgraph, AuthorityGradleModel, "subgraph", []string{materialization.FamilyID}, nil)
	patchBase := plannerContract(t, KindPatch, AuthorityReviewedPatch, "patch", nil, nil)
	task := plannerContract(t, KindTaskContract, AuthorityReviewedAdapter, "task", nil, []string{patchBase.FamilyID})
	patch := patchBase
	locality := plannerContract(t, KindCacheLocality, AuthorityGradleNative, "locality", nil, nil)
	families := map[string]string{
		"materialization": materialization.FamilyID, "subgraph": subgraph.FamilyID,
		"task": task.FamilyID, "patch": patch.FamilyID, "locality": locality.FamilyID,
	}
	request := PlanRequest{
		RepositoryScopeSHA256: materialization.RepositoryScopeSHA256,
		NativeWorkflowSHA256:  digest("planner-native-workflow", "assemble"),
		DecisionAt:            "2026-08-25T17:00:00Z", MinimumPredictedNetMs: 100,
		Candidates: []PlanningCandidate{
			{Fragment: plannerPersisted(materialization)}, {Fragment: plannerPersisted(subgraph)},
			{Fragment: plannerPersisted(task)}, {Fragment: plannerPersisted(patch)},
			{Fragment: plannerPersisted(locality)},
		},
		Predictions: []CompositionPrediction{
			plannerPrediction(3000, materialization.FamilyID, subgraph.FamilyID),
			plannerPrediction(4200, task.FamilyID, locality.FamilyID),
			plannerPrediction(5000, patch.FamilyID, locality.FamilyID),
			plannerPrediction(9000, task.FamilyID, patch.FamilyID),
			plannerPrediction(6000, subgraph.FamilyID),
			plannerPrediction(99, locality.FamilyID),
		},
	}
	return request, families
}

func plannerContract(t *testing.T, kind Kind, authority Authority, selector string, requires, conflicts []string) Fragment {
	t.Helper()
	common := digest("planner-binding", "common")
	bindings := map[BindingKey]string{}
	for _, key := range requiredBindings[kind] {
		bindings[key] = common
	}
	fragment, err := Derive(Input{
		RepositoryID: "fixture/planner", Kind: kind, Selector: []string{selector},
		Authority: authority, AuthoritySHA256: digest("planner-authority", selector),
		Bindings: bindings, Requires: requires, ConflictsWith: conflicts,
	})
	if err != nil {
		t.Fatal(err)
	}
	return fragment
}

func plannerPersisted(fragment Fragment) PersistedFragment {
	return PersistedFragment{
		SchemaVersion: FragmentStateSchemaVersion, RecordType: "ADAPTIVE_FRAGMENT",
		FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
		RepositoryScopeSHA256: fragment.RepositoryScopeSHA256, Kind: fragment.Kind,
		SelectorSHA256: fragment.SelectorSHA256, Authority: fragment.Authority,
		AuthoritySHA256: fragment.AuthoritySHA256, Bindings: fragment.Bindings,
		Requires: fragment.Requires, ConflictsWith: fragment.ConflictsWith,
		State: StateQualified, Generation: 1,
		CreatedAt: "2026-08-25T15:00:00Z", UpdatedAt: "2026-08-25T16:00:00Z",
		EvidenceExpiresAt: "2026-09-25T16:00:00Z",
	}
}

func plannerPrediction(value int64, families ...string) CompositionPrediction {
	return CompositionPrediction{
		Families: append([]string{}, families...), EvidenceSHA256: digest("planner-evidence", stringsForDigest(families)),
		HorizonBuilds: 5, PredictedNetMs: value,
	}
}

func stringsForDigest(values []string) string {
	copyValues := append([]string{}, values...)
	sort.Strings(copyValues)
	return stringsJoin(copyValues)
}

func stringsJoin(values []string) string {
	result := ""
	for index, value := range values {
		if index > 0 {
			result += ","
		}
		result += value
	}
	return result
}

func selectedFamilies(plan FragmentPlan) []string {
	result := make([]string, len(plan.Selected))
	for index, selected := range plan.Selected {
		result[index] = selected.FamilyID
	}
	return result
}

func clonePlanRequest(request PlanRequest) PlanRequest {
	result := request
	result.Candidates = make([]PlanningCandidate, len(request.Candidates))
	for index, candidate := range request.Candidates {
		result.Candidates[index] = PlanningCandidate{Fragment: clonePersistedFragments([]PersistedFragment{candidate.Fragment})[0]}
	}
	result.Predictions = make([]CompositionPrediction, len(request.Predictions))
	for index, prediction := range request.Predictions {
		result.Predictions[index] = prediction
		result.Predictions[index].Families = append([]string{}, prediction.Families...)
	}
	return result
}
