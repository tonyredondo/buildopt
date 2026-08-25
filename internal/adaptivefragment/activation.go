package adaptivefragment

import (
	"path"
	"sort"
	"strconv"
	"strings"
)

const BuildImpactActivationSchemaVersion = "buildopt.adaptive/build-impact-activation/v1"

// BuildImpactChangeScope is the pre-Gradle classification of the current
// change. Unknown or build-wide state never reaches partial execution.
type BuildImpactChangeScope string

const (
	BuildImpactChangeUnrelated BuildImpactChangeScope = "UNRELATED"
	BuildImpactChangeLocalized BuildImpactChangeScope = "LOCALIZED"
	BuildImpactChangeGlobal    BuildImpactChangeScope = "GLOBAL"
	BuildImpactChangeAmbiguous BuildImpactChangeScope = "AMBIGUOUS"
)

// StoredBuildImpactOutput binds one repository-relative output to the exact
// bytes that the verified producer authority previously materialized.
type StoredBuildImpactOutput struct {
	RelativePath  string `json:"relativePath"`
	ContentSHA256 string `json:"contentSha256"`
}

// BuildImpactProducerPair links the omission and exact-output fragments for
// one producer. Context is producer-specific so one producer can invalidate
// without suspending unrelated families.
type BuildImpactProducerPair struct {
	ProducerID                  string
	Subgraph                    PersistedFragment
	Materialization             PersistedFragment
	SubgraphContext             Context
	MaterializationContext      Context
	CurrentOutputRevisionSHA256 string
	StoredOutputRevisionSHA256  string
	StoredOutputs               []StoredBuildImpactOutput
	RebuildEntrypoints          []string
}

// BuildImpactActivationRequest contains only facts available before Gradle.
// Exact composition predictions retain the AF-009 non-additive semantics.
type BuildImpactActivationRequest struct {
	RepositoryScopeSHA256   string
	NativeWorkflowSHA256    string
	DecisionAt              string
	ChangeScope             BuildImpactChangeScope
	MinimumPredictedNetMs   uint64
	NativeEntrypoints       []string
	FinalizationEntrypoints []string
	Producers               []BuildImpactProducerPair
	Predictions             []CompositionPrediction
}

// BuildImpactProducerDecision makes every independent restore/rebuild choice
// visible without turning the decision into correctness authority.
type BuildImpactProducerDecision struct {
	ProducerID              string `json:"producerId"`
	Disposition             string `json:"disposition"`
	Reason                  string `json:"reason"`
	SubgraphFamilyID        string `json:"subgraphFamilyId"`
	MaterializationFamilyID string `json:"materializationFamilyId"`
}

// BuildImpactActivationPlan is either a partial producer execution with exact
// restorations or the complete original Gradle workflow.
type BuildImpactActivationPlan struct {
	SchemaVersion         string                        `json:"schemaVersion"`
	PlanID                string                        `json:"planId"`
	Disposition           string                        `json:"disposition"`
	Reason                string                        `json:"reason"`
	RepositoryScopeSHA256 string                        `json:"repositoryScopeSha256"`
	NativeWorkflowSHA256  string                        `json:"nativeWorkflowSha256"`
	DecisionAt            string                        `json:"decisionAt"`
	ChangeScope           BuildImpactChangeScope        `json:"changeScope"`
	Entrypoints           []string                      `json:"entrypoints"`
	PlannerPlanID         string                        `json:"plannerPlanId,omitempty"`
	SelectedFragments     []PlannedFragment             `json:"selectedFragments"`
	Restorations          []StoredBuildImpactOutput     `json:"restorations"`
	Producers             []BuildImpactProducerDecision `json:"producers"`
}

// ActivateBuildImpactFragments independently revalidates producer pairs,
// invokes the AF-009 exact-composition planner and returns the minimum Gradle
// entrypoints needed after verified unaffected outputs are restored.
func ActivateBuildImpactFragments(request BuildImpactActivationRequest) BuildImpactActivationPlan {
	base, reason := validateBuildImpactActivationRequest(request)
	if reason != "" {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, reason)
	}
	if request.ChangeScope == BuildImpactChangeGlobal {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, "GLOBAL_CHANGE")
	}
	if request.ChangeScope == BuildImpactChangeAmbiguous {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, "AMBIGUOUS_CHANGE")
	}

	producers, knownFamilies, reason := canonicalBuildImpactProducers(request)
	if reason != "" {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, reason)
	}
	if !predictionsReferenceKnownFamilies(request.Predictions, knownFamilies) {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, "INVALID_ACTIVATION_PREDICTION")
	}

	candidates := []PlanningCandidate{}
	eligibleFamilies := map[string]bool{}
	predecisions := map[string]BuildImpactProducerDecision{}
	for _, producer := range producers {
		decision, eligible, unsafe := evaluateBuildImpactProducer(producer, request.DecisionAt)
		if unsafe {
			return nativeBuildImpactActivation(base, request.NativeEntrypoints, decision.Reason)
		}
		predecisions[producer.ProducerID] = decision
		if eligible {
			candidates = append(candidates,
				PlanningCandidate{Fragment: producer.Subgraph},
				PlanningCandidate{Fragment: producer.Materialization},
			)
			eligibleFamilies[producer.Subgraph.FamilyID] = true
			eligibleFamilies[producer.Materialization.FamilyID] = true
		}
	}
	if len(candidates) == 0 {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, "NO_ACTIVE_BUILD_IMPACT_FRAGMENT")
	}

	predictions := filterBuildImpactPredictions(request.Predictions, eligibleFamilies)
	planner := PlanFragments(PlanRequest{
		RepositoryScopeSHA256: request.RepositoryScopeSHA256,
		NativeWorkflowSHA256:  request.NativeWorkflowSHA256,
		DecisionAt:            request.DecisionAt,
		MinimumPredictedNetMs: request.MinimumPredictedNetMs,
		Candidates:            candidates,
		Predictions:           predictions,
	})
	if planner.Disposition != PlanComposed {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, "PLANNER_"+planner.Reason)
	}

	selected := map[string]bool{}
	for _, fragment := range planner.Selected {
		selected[fragment.FamilyID] = true
	}
	rebuildEntrypoints := []string{}
	for _, producer := range producers {
		subgraphSelected := selected[producer.Subgraph.FamilyID]
		materializationSelected := selected[producer.Materialization.FamilyID]
		if subgraphSelected != materializationSelected {
			return nativeBuildImpactActivation(base, request.NativeEntrypoints, "INCOMPLETE_PRODUCER_FRAGMENT_PAIR")
		}
		decision := predecisions[producer.ProducerID]
		if subgraphSelected {
			decision.Disposition = "RESTORE"
			decision.Reason = "EXACT_UNAFFECTED_OUTPUT"
			base.Restorations = append(base.Restorations, cloneStoredOutputs(producer.StoredOutputs)...)
		} else {
			decision.Disposition = "REBUILD"
			if decision.Reason == "ELIGIBLE" {
				decision.Reason = "PLANNER_NOT_SELECTED"
			}
			rebuildEntrypoints = append(rebuildEntrypoints, producer.RebuildEntrypoints...)
		}
		base.Producers = append(base.Producers, decision)
	}
	if len(base.Restorations) == 0 {
		return nativeBuildImpactActivation(base, request.NativeEntrypoints, "NO_OUTPUT_RESTORATION_SELECTED")
	}
	sort.Slice(base.Restorations, func(left, right int) bool {
		return base.Restorations[left].RelativePath < base.Restorations[right].RelativePath
	})
	entrypoints := append(rebuildEntrypoints, request.FinalizationEntrypoints...)
	entrypoints, _ = normalizedStrings(entrypoints, false)
	base.Disposition = "PARTIAL_GRAPH"
	base.Reason = "COMPOSABLE_BUILD_IMPACT"
	base.Entrypoints = entrypoints
	base.PlannerPlanID = planner.PlanID
	base.SelectedFragments = append([]PlannedFragment{}, planner.Selected...)
	base.PlanID = buildImpactActivationPlanID(base)
	return base
}

func validateBuildImpactActivationRequest(request BuildImpactActivationRequest) (BuildImpactActivationPlan, string) {
	base := BuildImpactActivationPlan{
		SchemaVersion:         BuildImpactActivationSchemaVersion,
		RepositoryScopeSHA256: request.RepositoryScopeSHA256,
		NativeWorkflowSHA256:  request.NativeWorkflowSHA256,
		DecisionAt:            request.DecisionAt,
		ChangeScope:           request.ChangeScope,
		Entrypoints:           []string{}, SelectedFragments: []PlannedFragment{},
		Restorations: []StoredBuildImpactOutput{}, Producers: []BuildImpactProducerDecision{},
	}
	_, timeErr := parseUTC(request.DecisionAt)
	native, nativeErr := normalizedGradleEntrypoints(request.NativeEntrypoints)
	finalization, finalizationErr := normalizedGradleEntrypoints(request.FinalizationEntrypoints)
	if !validSHA(request.RepositoryScopeSHA256) || !validSHA(request.NativeWorkflowSHA256) || timeErr != nil ||
		request.MinimumPredictedNetMs == 0 || request.MinimumPredictedNetMs > maxEconomicComponentMs ||
		len(request.Producers) == 0 || len(request.Producers) > 16 || len(request.Predictions) > maxPlanningPredictions ||
		nativeErr != nil || finalizationErr != nil {
		return base, "INVALID_ACTIVATION_INPUT"
	}
	request.NativeEntrypoints = native
	request.FinalizationEntrypoints = finalization
	switch request.ChangeScope {
	case BuildImpactChangeUnrelated, BuildImpactChangeLocalized, BuildImpactChangeGlobal, BuildImpactChangeAmbiguous:
	default:
		return base, "INVALID_CHANGE_SCOPE"
	}
	return base, ""
}

func canonicalBuildImpactProducers(request BuildImpactActivationRequest) ([]BuildImpactProducerPair, map[string]bool, string) {
	result := make([]BuildImpactProducerPair, len(request.Producers))
	knownFamilies := map[string]bool{}
	producerIDs := map[string]bool{}
	outputPaths := map[string]bool{}
	for index, producer := range request.Producers {
		if !validSHA(producer.ProducerID) || producerIDs[producer.ProducerID] ||
			validatePersistedFragment(producer.Subgraph) != nil || validatePersistedFragment(producer.Materialization) != nil ||
			producer.Subgraph.Kind != KindSubgraph || producer.Materialization.Kind != KindOutputMaterialization ||
			producer.Subgraph.RepositoryScopeSHA256 != request.RepositoryScopeSHA256 ||
			producer.Materialization.RepositoryScopeSHA256 != request.RepositoryScopeSHA256 ||
			!contains(producer.Subgraph.Requires, producer.Materialization.FamilyID) ||
			!validSHA(producer.CurrentOutputRevisionSHA256) || !validSHA(producer.StoredOutputRevisionSHA256) {
			return nil, nil, "INVALID_PRODUCER_FRAGMENT_PAIR"
		}
		rebuild, err := normalizedGradleEntrypoints(producer.RebuildEntrypoints)
		if err != nil {
			return nil, nil, "INVALID_PRODUCER_FRAGMENT_PAIR"
		}
		outputs, err := normalizedStoredOutputs(producer.StoredOutputs)
		if err != nil {
			return nil, nil, "INVALID_STORED_OUTPUT"
		}
		for _, output := range outputs {
			if outputPaths[output.RelativePath] {
				return nil, nil, "AMBIGUOUS_STORED_OUTPUT"
			}
			outputPaths[output.RelativePath] = true
		}
		if knownFamilies[producer.Subgraph.FamilyID] || knownFamilies[producer.Materialization.FamilyID] {
			return nil, nil, "AMBIGUOUS_PRODUCER_FRAGMENT_FAMILY"
		}
		producerIDs[producer.ProducerID] = true
		knownFamilies[producer.Subgraph.FamilyID] = true
		knownFamilies[producer.Materialization.FamilyID] = true
		producer.RebuildEntrypoints = rebuild
		producer.StoredOutputs = outputs
		producer.Subgraph = clonePersistedFragments([]PersistedFragment{producer.Subgraph})[0]
		producer.Materialization = clonePersistedFragments([]PersistedFragment{producer.Materialization})[0]
		producer.SubgraphContext.Bindings = cloneBindings(producer.SubgraphContext.Bindings)
		producer.SubgraphContext.Ambiguous = append([]BindingKey{}, producer.SubgraphContext.Ambiguous...)
		producer.MaterializationContext.Bindings = cloneBindings(producer.MaterializationContext.Bindings)
		producer.MaterializationContext.Ambiguous = append([]BindingKey{}, producer.MaterializationContext.Ambiguous...)
		result[index] = producer
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ProducerID < result[right].ProducerID })
	return result, knownFamilies, ""
}

func evaluateBuildImpactProducer(producer BuildImpactProducerPair, decisionAt string) (BuildImpactProducerDecision, bool, bool) {
	decision := BuildImpactProducerDecision{
		ProducerID:              producer.ProducerID,
		SubgraphFamilyID:        producer.Subgraph.FamilyID,
		MaterializationFamilyID: producer.Materialization.FamilyID,
	}
	fragments := []struct {
		fragment PersistedFragment
		context  Context
	}{
		{fragment: producer.Subgraph, context: producer.SubgraphContext},
		{fragment: producer.Materialization, context: producer.MaterializationContext},
	}
	for _, current := range fragments {
		fragment := current.fragment
		if fragment.State != StateQualified && fragment.State != StateActive {
			decision.Disposition = "REBUILD"
			decision.Reason = "PRODUCER_FRAGMENT_NOT_QUALIFIED"
			return decision, false, false
		}
		expiresAt, _ := parseUTC(fragment.EvidenceExpiresAt)
		observedAt, _ := parseUTC(decisionAt)
		if !expiresAt.After(observedAt) {
			decision.Disposition = "REBUILD"
			decision.Reason = "PRODUCER_FRAGMENT_EXPIRED"
			return decision, false, false
		}
		compatibility, err := Evaluate(persistedFragmentContract(fragment), current.context)
		if err != nil || compatibility.Reason == "AMBIGUOUS_BINDING" || compatibility.Reason == "MISSING_BINDING" || compatibility.Reason == "REPOSITORY_SCOPE_MISMATCH" {
			decision.Reason = "PRODUCER_CONTEXT_UNSAFE"
			return decision, false, true
		}
		if !compatibility.Compatible {
			decision.Disposition = "REBUILD"
			decision.Reason = string(fragment.Kind) + "_" + compatibility.Reason
			return decision, false, false
		}
	}
	if producer.CurrentOutputRevisionSHA256 != producer.StoredOutputRevisionSHA256 {
		decision.Disposition = "REBUILD"
		decision.Reason = "OUTPUT_REVISION_STALE"
		return decision, false, false
	}
	decision.Disposition = "CANDIDATE"
	decision.Reason = "ELIGIBLE"
	return decision, true, false
}

func predictionsReferenceKnownFamilies(predictions []CompositionPrediction, known map[string]bool) bool {
	for _, prediction := range predictions {
		for _, family := range prediction.Families {
			if !known[family] {
				return false
			}
		}
	}
	return true
}

func filterBuildImpactPredictions(predictions []CompositionPrediction, eligible map[string]bool) []CompositionPrediction {
	result := []CompositionPrediction{}
	for _, prediction := range predictions {
		include := true
		for _, family := range prediction.Families {
			if !eligible[family] {
				include = false
				break
			}
		}
		if include {
			copyPrediction := prediction
			copyPrediction.Families = append([]string{}, prediction.Families...)
			result = append(result, copyPrediction)
		}
	}
	return result
}

func nativeBuildImpactActivation(base BuildImpactActivationPlan, nativeEntrypoints []string, reason string) BuildImpactActivationPlan {
	entrypoints, _ := normalizedGradleEntrypoints(nativeEntrypoints)
	base.Disposition = "NATIVE_GRADLE"
	base.Reason = reason
	base.Entrypoints = entrypoints
	base.PlannerPlanID = ""
	base.SelectedFragments = []PlannedFragment{}
	base.Restorations = []StoredBuildImpactOutput{}
	base.Producers = []BuildImpactProducerDecision{}
	base.PlanID = buildImpactActivationPlanID(base)
	return base
}

func normalizedGradleEntrypoints(values []string) ([]string, error) {
	result, err := normalizedStrings(values, false)
	if err != nil {
		return nil, err
	}
	for _, value := range result {
		if strings.HasPrefix(value, "-") || strings.ContainsAny(value, " \t\r\n") {
			return nil, path.ErrBadPattern
		}
	}
	return result, nil
}

func normalizedStoredOutputs(values []StoredBuildImpactOutput) ([]StoredBuildImpactOutput, error) {
	if len(values) == 0 || len(values) > 64 {
		return nil, path.ErrBadPattern
	}
	result := append([]StoredBuildImpactOutput{}, values...)
	for _, output := range result {
		clean := path.Clean(output.RelativePath)
		if clean == "." || clean != output.RelativePath || strings.HasPrefix(clean, "../") || path.IsAbs(clean) ||
			strings.ContainsRune(clean, '\\') || !validSHA(output.ContentSHA256) {
			return nil, path.ErrBadPattern
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left].RelativePath < result[right].RelativePath })
	for index := 1; index < len(result); index++ {
		if result[index-1].RelativePath == result[index].RelativePath {
			return nil, path.ErrBadPattern
		}
	}
	return result, nil
}

func cloneStoredOutputs(values []StoredBuildImpactOutput) []StoredBuildImpactOutput {
	return append([]StoredBuildImpactOutput{}, values...)
}

func cloneBindings(values map[BindingKey]string) map[BindingKey]string {
	result := make(map[BindingKey]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func buildImpactActivationPlanID(plan BuildImpactActivationPlan) string {
	rows := []string{
		string(plan.Disposition), plan.Reason, plan.RepositoryScopeSHA256,
		plan.NativeWorkflowSHA256, plan.DecisionAt, string(plan.ChangeScope), plan.PlannerPlanID,
	}
	rows = append(rows, plan.Entrypoints...)
	for _, fragment := range plan.SelectedFragments {
		rows = append(rows, fragment.FamilyID, fragment.RevisionID, strconv.FormatUint(fragment.FragmentGeneration, 10), fragment.AuthoritySHA256)
	}
	for _, output := range plan.Restorations {
		rows = append(rows, output.RelativePath, output.ContentSHA256)
	}
	for _, producer := range plan.Producers {
		rows = append(rows, producer.ProducerID, producer.Disposition, producer.Reason, producer.SubgraphFamilyID, producer.MaterializationFamilyID)
	}
	return digest("buildopt-adaptive-build-impact-activation-v1", rows...)
}
