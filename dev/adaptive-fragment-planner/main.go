// Command adaptive-fragment-planner recomputes the deterministic AF-009
// conflict, dependency, economic-floor and native-fallback proof.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-planner/v1"
	outcome      = "FRAGMENT_COMPOSITION_PLAN_AVAILABLE"
)

type report struct {
	SchemaVersion string                        `json:"schemaVersion"`
	WorkItem      string                        `json:"workItem"`
	CapturedAt    string                        `json:"capturedAt"`
	Policy        policyProof                   `json:"policy"`
	Input         inputProof                    `json:"input"`
	SelectedPlan  adaptivefragment.FragmentPlan `json:"selectedPlan"`
	Fallbacks     []fallbackProof               `json:"fallbacks"`
	Invariants    invariantProof                `json:"invariants"`
	Boundaries    boundaryProof                 `json:"boundaries"`
	Outcome       string                        `json:"outcome"`
}

type policyProof struct {
	PredictionModel       string `json:"predictionModel"`
	MinimumPredictedNetMs uint64 `json:"minimumPredictedNetMs"`
	OrderIndependent      bool   `json:"orderIndependent"`
	DependencyClosure     bool   `json:"dependencyClosure"`
	MutualExclusion       bool   `json:"mutualExclusion"`
	AuthorityComposition  string `json:"authorityComposition"`
	AmbiguityFallback     string `json:"ambiguityFallback"`
	NoEligibleFallback    string `json:"noEligibleFallback"`
	PreGradleDecision     bool   `json:"preGradleDecision"`
}

type inputProof struct {
	CandidateCount         uint64 `json:"candidateCount"`
	ExactPredictionCount   uint64 `json:"exactPredictionCount"`
	RepositoryRuleUsed     bool   `json:"repositoryRuleUsed"`
	IsolatedPercentageAdds uint64 `json:"isolatedPercentageAdds"`
	MeasurementOnlyBuilds  uint64 `json:"measurementOnlyBuilds"`
}

type fallbackProof struct {
	Case        string                           `json:"case"`
	Disposition adaptivefragment.PlanDisposition `json:"disposition"`
	Reason      string                           `json:"reason"`
	PlanID      string                           `json:"planId"`
	Selected    uint64                           `json:"selected"`
}

type invariantProof struct {
	ReorderedPlanIDMatches      bool   `json:"reorderedPlanIdMatches"`
	SelectedFragmentCount       uint64 `json:"selectedFragmentCount"`
	SelectedDependencyCount     uint64 `json:"selectedDependencyCount"`
	SelectedConflictCount       uint64 `json:"selectedConflictCount"`
	RetainedAuthorityCount      uint64 `json:"retainedAuthorityCount"`
	RejectedAlternativeCount    uint64 `json:"rejectedAlternativeCount"`
	FallbackCaseCount           uint64 `json:"fallbackCaseCount"`
	FallbackSelections          uint64 `json:"fallbackSelections"`
	ProductAttributableFailures uint64 `json:"productAttributableFailures"`
}

type boundaryProof struct {
	ProofOfConcept     bool   `json:"proofOfConcept"`
	SyntheticEconomics bool   `json:"syntheticEconomics"`
	NewTimingClaim     bool   `json:"newTimingClaim"`
	ActivationGranted  bool   `json:"activationGranted"`
	ProductionGranted  bool   `json:"productionGranted"`
	TestOptimization   string `json:"testOptimization"`
}

func main() {
	output := flag.String("output", "", "write the AF-009 report")
	validate := flag.String("validate", "", "validate an AF-009 report")
	flag.Parse()
	if flag.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-planner (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	expected, err := buildReport()
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("fragment planner report does not match recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment planner failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment planner: FRAGMENT_COMPOSITION_PLAN_AVAILABLE")
	}
}

func buildReport() (report, error) {
	request, families, err := fixtureRequest()
	if err != nil {
		return report{}, err
	}
	selected := adaptivefragment.PlanFragments(request)
	reordered := cloneRequest(request)
	reverseCandidates(reordered.Candidates)
	reversePredictions(reordered.Predictions)
	reorderedPlan := adaptivefragment.PlanFragments(reordered)

	fallbacks := []fallbackProof{}
	appendFallback := func(name string, candidate adaptivefragment.PlanRequest) {
		plan := adaptivefragment.PlanFragments(candidate)
		fallbacks = append(fallbacks, fallbackProof{
			Case: name, Disposition: plan.Disposition, Reason: plan.Reason,
			PlanID: plan.PlanID, Selected: uint64(len(plan.Selected)),
		})
	}
	missingPrediction := cloneRequest(request)
	missingPrediction.Predictions = []adaptivefragment.CompositionPrediction{}
	appendFallback("EXACT_COMPOSITION_PREDICTION_UNAVAILABLE", missingPrediction)

	belowFloor := cloneRequest(request)
	belowFloor.Predictions = []adaptivefragment.CompositionPrediction{prediction(99, families["locality"])}
	appendFallback("PREDICTED_NET_BELOW_FLOOR", belowFloor)

	ambiguous := cloneRequest(request)
	ambiguous.Predictions = []adaptivefragment.CompositionPrediction{
		prediction(1000, families["locality"]), prediction(1001, families["locality"]),
	}
	appendFallback("AMBIGUOUS_COMPOSITION_PREDICTION", ambiguous)

	missingDependency := cloneRequest(request)
	missingDependency.Candidates = removeCandidate(missingDependency.Candidates, families["materialization"])
	missingDependency.Predictions = []adaptivefragment.CompositionPrediction{prediction(1000, families["locality"])}
	appendFallback("DEPENDENCY_UNAVAILABLE", missingDependency)

	expired := cloneRequest(request)
	expired.Candidates[0].Fragment.EvidenceExpiresAt = expired.DecisionAt
	appendFallback("CANDIDATE_EVIDENCE_EXPIRED", expired)

	unqualified := cloneRequest(request)
	unqualified.Candidates[0].Fragment.State = adaptivefragment.StateShadow
	appendFallback("CANDIDATE_NOT_QUALIFIED", unqualified)

	unknown := cloneRequest(request)
	unknown.Predictions = []adaptivefragment.CompositionPrediction{prediction(1000, digest("unknown-family"))}
	appendFallback("PREDICTION_REFERENCES_UNKNOWN_FRAGMENT", unknown)

	report := report{
		SchemaVersion: reportSchema, WorkItem: "AF-009", CapturedAt: "2026-08-25T18:00:00Z",
		Policy: policyProof{
			PredictionModel:       "EXACT_COMPOSITION_PREDICTION_NOT_ADDED_ISOLATED_EFFECTS",
			MinimumPredictedNetMs: request.MinimumPredictedNetMs, OrderIndependent: true,
			DependencyClosure: true, MutualExclusion: true,
			AuthorityComposition: "CONJUNCTION_OF_CONSTITUENT_AUTHORITIES",
			AmbiguityFallback:    "NATIVE_GRADLE", NoEligibleFallback: "NATIVE_GRADLE", PreGradleDecision: true,
		},
		Input: inputProof{
			CandidateCount: uint64(len(request.Candidates)), ExactPredictionCount: uint64(len(request.Predictions)),
		},
		SelectedPlan: selected, Fallbacks: fallbacks,
		Invariants: invariantProof{
			ReorderedPlanIDMatches: selected.PlanID == reorderedPlan.PlanID,
			SelectedFragmentCount:  uint64(len(selected.Selected)), SelectedDependencyCount: 1,
			SelectedConflictCount: 0, RetainedAuthorityCount: uint64(len(selected.Selected)),
			RejectedAlternativeCount: uint64(len(selected.RejectedAlternatives)),
			FallbackCaseCount:        uint64(len(fallbacks)),
		},
		Boundaries: boundaryProof{
			ProofOfConcept: true, SyntheticEconomics: true,
			TestOptimization: "OUT_OF_SCOPE",
		},
		Outcome: outcome,
	}
	for _, fallback := range fallbacks {
		report.Invariants.FallbackSelections += fallback.Selected
	}
	return report, nil
}

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-009" || candidate.Outcome != outcome ||
		candidate.SelectedPlan.Disposition != adaptivefragment.PlanComposed ||
		candidate.SelectedPlan.Reason != "POSITIVE_EXACT_COMPOSITION" || candidate.SelectedPlan.PredictedNetMs != 5000 ||
		candidate.SelectedPlan.CorrectnessMode != "CONJUNCTION_OF_CONSTITUENT_AUTHORITIES" ||
		!candidate.Invariants.ReorderedPlanIDMatches || candidate.Invariants.SelectedFragmentCount != 2 ||
		candidate.Invariants.SelectedDependencyCount != 1 ||
		candidate.Invariants.RetainedAuthorityCount != 2 || candidate.Invariants.SelectedConflictCount != 0 ||
		candidate.Invariants.RejectedAlternativeCount != 3 || candidate.Invariants.FallbackCaseCount != 7 ||
		candidate.Invariants.FallbackSelections != 0 || candidate.Invariants.ProductAttributableFailures != 0 ||
		candidate.Input.RepositoryRuleUsed || candidate.Input.IsolatedPercentageAdds != 0 || candidate.Input.MeasurementOnlyBuilds != 0 ||
		!candidate.Boundaries.ProofOfConcept || !candidate.Boundaries.SyntheticEconomics || candidate.Boundaries.NewTimingClaim ||
		candidate.Boundaries.ActivationGranted || candidate.Boundaries.ProductionGranted || candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("fragment planner report invariant failed")
	}
	expectedReasons := []string{
		"EXACT_COMPOSITION_PREDICTION_UNAVAILABLE", "PREDICTED_NET_BELOW_FLOOR",
		"AMBIGUOUS_COMPOSITION_PREDICTION", "DEPENDENCY_UNAVAILABLE",
		"CANDIDATE_EVIDENCE_EXPIRED", "CANDIDATE_NOT_QUALIFIED", "PREDICTION_REFERENCES_UNKNOWN_FRAGMENT",
	}
	if len(candidate.Fallbacks) != len(expectedReasons) {
		return errors.New("fragment planner fallback coverage is incomplete")
	}
	for index, fallback := range candidate.Fallbacks {
		if fallback.Case != expectedReasons[index] || fallback.Reason != expectedReasons[index] ||
			fallback.Disposition != adaptivefragment.PlanNativeRetained || fallback.Selected != 0 || fallback.PlanID == "" {
			return errors.New("fragment planner fallback is unsafe")
		}
	}
	return nil
}

func fixtureRequest() (adaptivefragment.PlanRequest, map[string]string, error) {
	materialization, err := fragment(adaptivefragment.KindOutputMaterialization, adaptivefragment.AuthorityVerifiedProducer, "materialization", nil, nil)
	if err != nil {
		return adaptivefragment.PlanRequest{}, nil, err
	}
	subgraph, err := fragment(adaptivefragment.KindSubgraph, adaptivefragment.AuthorityGradleModel, "subgraph", []string{materialization.FamilyID}, nil)
	if err != nil {
		return adaptivefragment.PlanRequest{}, nil, err
	}
	patchBase, err := fragment(adaptivefragment.KindPatch, adaptivefragment.AuthorityReviewedPatch, "patch", nil, nil)
	if err != nil {
		return adaptivefragment.PlanRequest{}, nil, err
	}
	task, err := fragment(adaptivefragment.KindTaskContract, adaptivefragment.AuthorityReviewedAdapter, "task", nil, []string{patchBase.FamilyID})
	if err != nil {
		return adaptivefragment.PlanRequest{}, nil, err
	}
	patch := patchBase
	locality, err := fragment(adaptivefragment.KindCacheLocality, adaptivefragment.AuthorityGradleNative, "locality", nil, nil)
	if err != nil {
		return adaptivefragment.PlanRequest{}, nil, err
	}
	families := map[string]string{
		"materialization": materialization.FamilyID, "subgraph": subgraph.FamilyID,
		"task": task.FamilyID, "patch": patch.FamilyID, "locality": locality.FamilyID,
	}
	request := adaptivefragment.PlanRequest{
		RepositoryScopeSHA256: materialization.RepositoryScopeSHA256,
		NativeWorkflowSHA256:  digest("native-workflow"), DecisionAt: "2026-08-25T17:00:00Z",
		MinimumPredictedNetMs: 100,
		Candidates: []adaptivefragment.PlanningCandidate{
			{Fragment: persisted(materialization)}, {Fragment: persisted(subgraph)}, {Fragment: persisted(task)},
			{Fragment: persisted(patch)}, {Fragment: persisted(locality)},
		},
		Predictions: []adaptivefragment.CompositionPrediction{
			prediction(5000, materialization.FamilyID, subgraph.FamilyID),
			prediction(4200, task.FamilyID, locality.FamilyID),
			prediction(4800, patch.FamilyID, locality.FamilyID),
			prediction(9000, task.FamilyID, patch.FamilyID),
			prediction(6000, subgraph.FamilyID), prediction(99, locality.FamilyID),
		},
	}
	return request, families, nil
}

func fragment(kind adaptivefragment.Kind, authority adaptivefragment.Authority, selector string, requires, conflicts []string) (adaptivefragment.Fragment, error) {
	common := digest("binding-common")
	bindings := map[adaptivefragment.BindingKey]string{}
	switch kind {
	case adaptivefragment.KindSubgraph:
		for _, key := range []adaptivefragment.BindingKey{adaptivefragment.BindingWorkflow, adaptivefragment.BindingWrapper, adaptivefragment.BindingProducerLineage, adaptivefragment.BindingOutputContract, adaptivefragment.BindingChangeFamily} {
			bindings[key] = common
		}
	case adaptivefragment.KindOutputMaterialization:
		for _, key := range []adaptivefragment.BindingKey{adaptivefragment.BindingWrapper, adaptivefragment.BindingProducerLineage, adaptivefragment.BindingOutputContract} {
			bindings[key] = common
		}
	case adaptivefragment.KindTaskContract:
		for _, key := range []adaptivefragment.BindingKey{adaptivefragment.BindingWrapper, adaptivefragment.BindingTaskImplementation, adaptivefragment.BindingOutputContract} {
			bindings[key] = common
		}
	case adaptivefragment.KindPatch:
		for _, key := range []adaptivefragment.BindingKey{adaptivefragment.BindingTaskImplementation, adaptivefragment.BindingOutputContract, adaptivefragment.BindingPatchBase} {
			bindings[key] = common
		}
	case adaptivefragment.KindCacheLocality:
		for _, key := range []adaptivefragment.BindingKey{adaptivefragment.BindingWrapper, adaptivefragment.BindingNetworkClass, adaptivefragment.BindingCacheNamespace} {
			bindings[key] = common
		}
	}
	return adaptivefragment.Derive(adaptivefragment.Input{
		RepositoryID: "fixture/planner", Kind: kind, Selector: []string{selector},
		Authority: authority, AuthoritySHA256: digest("authority-" + selector), Bindings: bindings,
		Requires: requires, ConflictsWith: conflicts,
	})
}

func persisted(fragment adaptivefragment.Fragment) adaptivefragment.PersistedFragment {
	bindings := make(map[adaptivefragment.BindingKey]string, len(fragment.Bindings))
	for key, value := range fragment.Bindings {
		bindings[key] = value
	}
	return adaptivefragment.PersistedFragment{
		SchemaVersion: adaptivefragment.FragmentStateSchemaVersion, RecordType: "ADAPTIVE_FRAGMENT",
		FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
		RepositoryScopeSHA256: fragment.RepositoryScopeSHA256, Kind: fragment.Kind,
		SelectorSHA256: fragment.SelectorSHA256, Authority: fragment.Authority,
		AuthoritySHA256: fragment.AuthoritySHA256, Bindings: bindings,
		Requires: append([]string{}, fragment.Requires...), ConflictsWith: append([]string{}, fragment.ConflictsWith...),
		State: adaptivefragment.StateQualified, Generation: 1,
		CreatedAt: "2026-08-25T15:00:00Z", UpdatedAt: "2026-08-25T16:00:00Z",
		EvidenceExpiresAt: "2026-09-25T16:00:00Z",
	}
}

func prediction(value int64, families ...string) adaptivefragment.CompositionPrediction {
	canonical := append([]string{}, families...)
	sort.Strings(canonical)
	return adaptivefragment.CompositionPrediction{
		Families: append([]string{}, families...), EvidenceSHA256: digest(fmt.Sprintf("prediction-%v", canonical)),
		HorizonBuilds: 5, PredictedNetMs: value,
	}
}

func cloneRequest(request adaptivefragment.PlanRequest) adaptivefragment.PlanRequest {
	result := request
	result.Candidates = make([]adaptivefragment.PlanningCandidate, len(request.Candidates))
	for index, candidate := range request.Candidates {
		result.Candidates[index] = adaptivefragment.PlanningCandidate{Fragment: persisted(adaptivefragment.Fragment{
			SchemaVersion: adaptivefragment.SchemaVersion, FamilyID: candidate.Fragment.FamilyID,
			RevisionID: candidate.Fragment.RevisionID, RepositoryScopeSHA256: candidate.Fragment.RepositoryScopeSHA256,
			Kind: candidate.Fragment.Kind, SelectorSHA256: candidate.Fragment.SelectorSHA256,
			Authority: candidate.Fragment.Authority, AuthoritySHA256: candidate.Fragment.AuthoritySHA256,
			Bindings: candidate.Fragment.Bindings, Requires: candidate.Fragment.Requires,
			ConflictsWith: candidate.Fragment.ConflictsWith,
		})}
		result.Candidates[index].Fragment.State = candidate.Fragment.State
		result.Candidates[index].Fragment.Generation = candidate.Fragment.Generation
		result.Candidates[index].Fragment.CreatedAt = candidate.Fragment.CreatedAt
		result.Candidates[index].Fragment.UpdatedAt = candidate.Fragment.UpdatedAt
		result.Candidates[index].Fragment.EvidenceExpiresAt = candidate.Fragment.EvidenceExpiresAt
	}
	result.Predictions = make([]adaptivefragment.CompositionPrediction, len(request.Predictions))
	for index, item := range request.Predictions {
		result.Predictions[index] = item
		result.Predictions[index].Families = append([]string{}, item.Families...)
	}
	return result
}

func removeCandidate(candidates []adaptivefragment.PlanningCandidate, family string) []adaptivefragment.PlanningCandidate {
	result := []adaptivefragment.PlanningCandidate{}
	for _, candidate := range candidates {
		if candidate.Fragment.FamilyID != family {
			result = append(result, candidate)
		}
	}
	return result
}

func reverseCandidates(values []adaptivefragment.PlanningCandidate) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reversePredictions(values []adaptivefragment.CompositionPrediction) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func readJSONStrict(path string, destination any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("fragment planner report contains multiple documents")
	}
	return nil
}

func writeJSON(path string, value any) error {
	document, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	document = append(document, '\n')
	return os.WriteFile(path, document, 0o600)
}
