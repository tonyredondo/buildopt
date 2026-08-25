package adaptivefragment

import (
	"reflect"
	"sort"
	"testing"
)

func TestBuildImpactActivationRestoresEveryUnaffectedProducer(t *testing.T) {
	request, _ := buildImpactActivationFixture(t)
	before := cloneBuildImpactActivationRequest(request)
	plan := ActivateBuildImpactFragments(request)
	if plan.Disposition != "PARTIAL_GRAPH" || plan.Reason != "COMPOSABLE_BUILD_IMPACT" ||
		!reflect.DeepEqual(plan.Entrypoints, []string{"packageAll"}) || len(plan.Restorations) != 2 ||
		len(plan.SelectedFragments) != 4 || plan.PlannerPlanID == "" {
		t.Fatalf("activation plan = %+v", plan)
	}
	for _, producer := range plan.Producers {
		if producer.Disposition != "RESTORE" || producer.Reason != "EXACT_UNAFFECTED_OUTPUT" {
			t.Fatalf("producer decision = %+v", producer)
		}
	}
	if !reflect.DeepEqual(request, before) {
		t.Fatal("activation mutated its request")
	}
}

func TestBuildImpactActivationRebuildsOnlyChangedProducer(t *testing.T) {
	request, producerIDs := buildImpactActivationFixture(t)
	for index := range request.Producers {
		if request.Producers[index].ProducerID != producerIDs["a"] {
			continue
		}
		request.Producers[index].SubgraphContext.Bindings[BindingChangeFamily] = digest("changed-family", "producer-a")
		request.Producers[index].CurrentOutputRevisionSHA256 = digest("output-revision", "producer-a-changed")
	}
	request.ChangeScope = BuildImpactChangeLocalized
	plan := ActivateBuildImpactFragments(request)
	if plan.Disposition != "PARTIAL_GRAPH" ||
		!reflect.DeepEqual(plan.Entrypoints, []string{":producer-a:produce", "packageAll"}) ||
		len(plan.Restorations) != 1 || len(plan.SelectedFragments) != 2 {
		t.Fatalf("localized activation = %+v", plan)
	}
	decisions := producerDecisionMap(plan.Producers)
	if decisions[producerIDs["a"]].Disposition != "REBUILD" || decisions[producerIDs["a"]].Reason != "SUBGRAPH_BINDING_DRIFT" ||
		decisions[producerIDs["b"]].Disposition != "RESTORE" {
		t.Fatalf("localized decisions = %+v", decisions)
	}
}

func TestBuildImpactActivationRebuildsStaleOutputProducer(t *testing.T) {
	request, producerIDs := buildImpactActivationFixture(t)
	for index := range request.Producers {
		if request.Producers[index].ProducerID == producerIDs["a"] {
			request.Producers[index].CurrentOutputRevisionSHA256 = digest("output-revision", "producer-a-stale")
		}
	}
	request.ChangeScope = BuildImpactChangeLocalized
	plan := ActivateBuildImpactFragments(request)
	decisions := producerDecisionMap(plan.Producers)
	if plan.Disposition != "PARTIAL_GRAPH" || decisions[producerIDs["a"]].Reason != "OUTPUT_REVISION_STALE" ||
		decisions[producerIDs["b"]].Disposition != "RESTORE" {
		t.Fatalf("stale output activation = %+v", plan)
	}
}

func TestBuildImpactActivationRebuildsOnlyExpiredProducer(t *testing.T) {
	request, producerIDs := buildImpactActivationFixture(t)
	for index := range request.Producers {
		if request.Producers[index].ProducerID == producerIDs["a"] {
			request.Producers[index].Subgraph.EvidenceExpiresAt = request.DecisionAt
		}
	}
	plan := ActivateBuildImpactFragments(request)
	decisions := producerDecisionMap(plan.Producers)
	if plan.Disposition != "PARTIAL_GRAPH" ||
		decisions[producerIDs["a"]].Disposition != "REBUILD" ||
		decisions[producerIDs["a"]].Reason != "PRODUCER_FRAGMENT_EXPIRED" ||
		decisions[producerIDs["b"]].Disposition != "RESTORE" {
		t.Fatalf("expired producer activation = %+v", plan)
	}
}

func TestBuildImpactActivationRetainsNativeForGlobalAmbiguousOrUnsafeState(t *testing.T) {
	base, producerIDs := buildImpactActivationFixture(t)
	tests := []struct {
		name   string
		mutate func(BuildImpactActivationRequest) BuildImpactActivationRequest
		reason string
	}{
		{
			name: "global change",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				request.ChangeScope = BuildImpactChangeGlobal
				return request
			},
			reason: "GLOBAL_CHANGE",
		},
		{
			name: "ambiguous change",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				request.ChangeScope = BuildImpactChangeAmbiguous
				return request
			},
			reason: "AMBIGUOUS_CHANGE",
		},
		{
			name: "ambiguous producer binding",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				for index := range request.Producers {
					if request.Producers[index].ProducerID == producerIDs["a"] {
						request.Producers[index].SubgraphContext.Ambiguous = []BindingKey{BindingChangeFamily}
					}
				}
				return request
			},
			reason: "PRODUCER_CONTEXT_UNSAFE",
		},
		{
			name: "duplicate producer family",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				request.Producers[1].Subgraph = request.Producers[0].Subgraph
				request.Producers[1].SubgraphContext = request.Producers[0].SubgraphContext
				request.Producers[1].Materialization = request.Producers[0].Materialization
				request.Producers[1].MaterializationContext = request.Producers[0].MaterializationContext
				return request
			},
			reason: "AMBIGUOUS_PRODUCER_FRAGMENT_FAMILY",
		},
		{
			name: "prediction unavailable",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				request.Predictions = []CompositionPrediction{}
				return request
			},
			reason: "PLANNER_EXACT_COMPOSITION_PREDICTION_UNAVAILABLE",
		},
		{
			name: "incomplete selected pair",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				request.Predictions = []CompositionPrediction{plannerPrediction(5000, request.Producers[0].Materialization.FamilyID)}
				return request
			},
			reason: "INCOMPLETE_PRODUCER_FRAGMENT_PAIR",
		},
		{
			name: "unsafe output path",
			mutate: func(request BuildImpactActivationRequest) BuildImpactActivationRequest {
				request.Producers[0].StoredOutputs[0].RelativePath = "../outside"
				return request
			},
			reason: "INVALID_STORED_OUTPUT",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := ActivateBuildImpactFragments(test.mutate(cloneBuildImpactActivationRequest(base)))
			if plan.Disposition != "NATIVE_GRADLE" || plan.Reason != test.reason ||
				!reflect.DeepEqual(plan.Entrypoints, []string{"fullBuild"}) || len(plan.Restorations) != 0 ||
				len(plan.SelectedFragments) != 0 {
				t.Fatalf("native plan = %+v, want %s", plan, test.reason)
			}
		})
	}
}

func TestBuildImpactActivationIsInputOrderIndependent(t *testing.T) {
	request, _ := buildImpactActivationFixture(t)
	want := ActivateBuildImpactFragments(request)
	for left, right := 0, len(request.Producers)-1; left < right; left, right = left+1, right-1 {
		request.Producers[left], request.Producers[right] = request.Producers[right], request.Producers[left]
	}
	for left, right := 0, len(request.Predictions)-1; left < right; left, right = left+1, right-1 {
		request.Predictions[left], request.Predictions[right] = request.Predictions[right], request.Predictions[left]
	}
	got := ActivateBuildImpactFragments(request)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reordered activation = %+v, want %+v", got, want)
	}
}

func buildImpactActivationFixture(t *testing.T) (BuildImpactActivationRequest, map[string]string) {
	t.Helper()
	producerA := buildImpactActivationProducer(t, "producer-a")
	producerB := buildImpactActivationProducer(t, "producer-b")
	request := BuildImpactActivationRequest{
		RepositoryScopeSHA256:   producerA.Subgraph.RepositoryScopeSHA256,
		NativeWorkflowSHA256:    digest("activation-native", "fullBuild"),
		DecisionAt:              "2026-08-25T19:00:00Z",
		ChangeScope:             BuildImpactChangeUnrelated,
		MinimumPredictedNetMs:   100,
		NativeEntrypoints:       []string{"fullBuild"},
		FinalizationEntrypoints: []string{"packageAll"},
		Producers:               []BuildImpactProducerPair{producerA, producerB},
		Predictions: []CompositionPrediction{
			plannerPrediction(1200, producerA.Subgraph.FamilyID, producerA.Materialization.FamilyID),
			plannerPrediction(1400, producerB.Subgraph.FamilyID, producerB.Materialization.FamilyID),
			plannerPrediction(3200,
				producerA.Subgraph.FamilyID, producerA.Materialization.FamilyID,
				producerB.Subgraph.FamilyID, producerB.Materialization.FamilyID,
			),
		},
	}
	return request, map[string]string{"a": producerA.ProducerID, "b": producerB.ProducerID}
}

func buildImpactActivationProducer(t *testing.T, name string) BuildImpactProducerPair {
	t.Helper()
	materialization := plannerContract(t, KindOutputMaterialization, AuthorityVerifiedProducer, name+"-materialization", nil, nil)
	subgraph := plannerContract(t, KindSubgraph, AuthorityGradleModel, name+"-subgraph", []string{materialization.FamilyID}, nil)
	outputRevision := digest("output-revision", name)
	return BuildImpactProducerPair{
		ProducerID: digest("producer", name),
		Subgraph:   plannerPersisted(subgraph), Materialization: plannerPersisted(materialization),
		SubgraphContext: Context{
			RepositoryID: "fixture/planner", Bindings: cloneBindings(subgraph.Bindings), Ambiguous: []BindingKey{},
		},
		MaterializationContext: Context{
			RepositoryID: "fixture/planner", Bindings: cloneBindings(materialization.Bindings), Ambiguous: []BindingKey{},
		},
		CurrentOutputRevisionSHA256: outputRevision, StoredOutputRevisionSHA256: outputRevision,
		StoredOutputs: []StoredBuildImpactOutput{{
			RelativePath: name + "/build/fragment/value.txt", ContentSHA256: digest("content", name),
		}},
		RebuildEntrypoints: []string{":" + name + ":produce"},
	}
}

func cloneBuildImpactActivationRequest(request BuildImpactActivationRequest) BuildImpactActivationRequest {
	result := request
	result.NativeEntrypoints = append([]string{}, request.NativeEntrypoints...)
	result.FinalizationEntrypoints = append([]string{}, request.FinalizationEntrypoints...)
	result.Producers = make([]BuildImpactProducerPair, len(request.Producers))
	for index, producer := range request.Producers {
		result.Producers[index] = producer
		result.Producers[index].Subgraph = clonePersistedFragments([]PersistedFragment{producer.Subgraph})[0]
		result.Producers[index].Materialization = clonePersistedFragments([]PersistedFragment{producer.Materialization})[0]
		result.Producers[index].SubgraphContext.Bindings = cloneBindings(producer.SubgraphContext.Bindings)
		result.Producers[index].SubgraphContext.Ambiguous = append([]BindingKey{}, producer.SubgraphContext.Ambiguous...)
		result.Producers[index].MaterializationContext.Bindings = cloneBindings(producer.MaterializationContext.Bindings)
		result.Producers[index].MaterializationContext.Ambiguous = append([]BindingKey{}, producer.MaterializationContext.Ambiguous...)
		result.Producers[index].StoredOutputs = cloneStoredOutputs(producer.StoredOutputs)
		result.Producers[index].RebuildEntrypoints = append([]string{}, producer.RebuildEntrypoints...)
	}
	result.Predictions = make([]CompositionPrediction, len(request.Predictions))
	for index, prediction := range request.Predictions {
		result.Predictions[index] = prediction
		result.Predictions[index].Families = append([]string{}, prediction.Families...)
	}
	return result
}

func producerDecisionMap(decisions []BuildImpactProducerDecision) map[string]BuildImpactProducerDecision {
	result := map[string]BuildImpactProducerDecision{}
	for _, decision := range decisions {
		result[decision.ProducerID] = decision
	}
	return result
}

func sortedSelectedActivationFamilies(plan BuildImpactActivationPlan) []string {
	result := make([]string, 0, len(plan.SelectedFragments))
	for _, fragment := range plan.SelectedFragments {
		result = append(result, fragment.FamilyID)
	}
	sort.Strings(result)
	return result
}
