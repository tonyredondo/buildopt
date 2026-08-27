package durablecatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	PublicScreenSchemaVersion = "buildopt.poc/sticky-wrapper-opportunity-gate/v1"

	DetectorTaskContract = "TASK_CONTRACT_JAVA_V1"
	DetectorGraphBreadth = "DECLARED_GRAPH_SCOPE_V1"

	DetectorStatusTestableActions  = "TESTABLE_ACTIONS"
	DetectorStatusNoOpportunity    = "NO_OPPORTUNITY"
	DetectorStatusInputUnavailable = "INPUT_UNAVAILABLE"
)

// PublicScreenInput contains only detector evidence and fixed economic costs.
// It does not grant authority to apply or activate an opportunity.
type PublicScreenInput struct {
	GeneratedAt          string
	CohortManifestSHA256 string
	DetectorOrder        []string
	MinimumPassing       uint64
	MaximumBuildsToRepay uint64
	Families             []PublicFamilyInput
}

// PublicFamilyInput binds one public family to its frozen revision window.
type PublicFamilyInput struct {
	FamilyKey               string
	RevisionWindowSHA256    string
	RepositoryScopeSHA256   string
	WorkflowArgumentsSHA256 string
	OutputContract          []string
	Detectors               []PublicDetectorInput
}

// PublicDetectorInput is either explicit unavailable evidence or observations
// that the named generic detector can evaluate.
type PublicDetectorInput struct {
	DetectorID          string
	InputEvidenceSHA256 string
	UnavailableReason   string
	GraphObservations   []PublicGraphObservation
	TrialCostNs         uint64
	ValidationCostNs    uint64
	PublicationCostNs   uint64
}

// PublicGraphObservation is one exact recurrence candidate. CandidatePlanSHA256
// identifies the full selected project/task plan; project counts alone are not
// a valid substitute. ProjectedSavingNs must come from omitted critical-path
// analysis rather than from proportional project-count estimates.
type PublicGraphObservation struct {
	Ordinal               uint64
	ObservationID         string
	CandidatePlanSHA256   string
	BindingDigest         string
	FullProjectCount      uint64
	CandidateProjectCount uint64
	FullOutputSHA256      string
	CandidateOutputSHA256 string
	ProjectedSavingNs     uint64
	ProductFailure        bool
}

type PublicAction struct {
	ActionID                string   `json:"actionId"`
	DetectorID              string   `json:"detectorId"`
	ActionKind              string   `json:"actionKind"`
	BindingDigest           string   `json:"bindingDigest"`
	RecurringObservationIDs []string `json:"recurringObservationIds"`
	RecurrenceCount         uint64   `json:"recurrenceCount"`
	ProjectedSavingNs       uint64   `json:"projectedSavingNs"`
	TrialCostNs             uint64   `json:"trialCostNs"`
	ValidationCostNs        uint64   `json:"validationCostNs"`
	PublicationCostNs       uint64   `json:"publicationCostNs"`
	CompatibleBuildsToRepay uint64   `json:"compatibleBuildsToRepay"`
	OutputContract          []string `json:"outputContract"`
	OwnerReviewRequired     bool     `json:"ownerReviewRequired"`
}

type PublicDetectorResult struct {
	DetectorID          string   `json:"detectorId"`
	Status              string   `json:"status"`
	Reason              string   `json:"reason"`
	InputEvidenceSHA256 string   `json:"inputEvidenceSha256"`
	ActionIDs           []string `json:"actionIds"`
}

type PublicFamilyResult struct {
	FamilyKey            string                 `json:"familyKey"`
	RevisionWindowSHA256 string                 `json:"revisionWindowSha256"`
	DetectorResults      []PublicDetectorResult `json:"detectorResults"`
	TestableActions      []PublicAction         `json:"testableActions"`
	Passed               bool                   `json:"passed"`
	Reason               string                 `json:"reason"`
}

type PublicScreenReport struct {
	SchemaVersion          string               `json:"schemaVersion"`
	WorkItem               string               `json:"workItem"`
	GeneratedAt            string               `json:"generatedAt"`
	CohortManifestSHA256   string               `json:"cohortManifestSha256"`
	DetectorOrder          []string             `json:"detectorOrder"`
	Families               []PublicFamilyResult `json:"families"`
	PassingFamilies        uint64               `json:"passingFamilies"`
	MinimumPassingFamilies uint64               `json:"minimumPassingFamilies"`
	GatePassed             bool                 `json:"gatePassed"`
	Outcome                string               `json:"outcome"`
}

// ScreenPublicOpportunities runs the declared detectors in order and applies
// the preregistered recurrence and payback gate. Unsupported evidence remains
// visible and never becomes a synthetic opportunity.
func ScreenPublicOpportunities(input PublicScreenInput) (PublicScreenReport, error) {
	if input.GeneratedAt == "" || !validSHA(input.CohortManifestSHA256) || input.MinimumPassing == 0 ||
		input.MaximumBuildsToRepay == 0 || len(input.Families) != 5 ||
		!sameDetectorOrder(input.DetectorOrder) {
		return PublicScreenReport{}, errors.New("public opportunity screen identity is invalid")
	}
	families := append([]PublicFamilyInput(nil), input.Families...)
	sort.Slice(families, func(i, j int) bool { return families[i].FamilyKey < families[j].FamilyKey })
	results := make([]PublicFamilyResult, 0, len(families))
	seen := map[string]bool{}
	var passing uint64
	for _, family := range families {
		if family.FamilyKey == "" || seen[family.FamilyKey] || !validSHA(family.RevisionWindowSHA256) ||
			!validSHA(family.RepositoryScopeSHA256) || !validSHA(family.WorkflowArgumentsSHA256) ||
			len(family.OutputContract) == 0 || len(family.Detectors) != len(input.DetectorOrder) {
			return PublicScreenReport{}, fmt.Errorf("family %q identity is invalid", family.FamilyKey)
		}
		seen[family.FamilyKey] = true
		result, err := screenFamily(family, input.DetectorOrder, input.MaximumBuildsToRepay)
		if err != nil {
			return PublicScreenReport{}, fmt.Errorf("family %s: %w", family.FamilyKey, err)
		}
		if result.Passed {
			passing++
		}
		results = append(results, result)
	}
	passed := passing >= input.MinimumPassing
	outcome := "SWL_014C_WITH_STOP_EVIDENCE"
	if passed {
		outcome = "READY_FOR_SWL_014D"
	}
	return PublicScreenReport{
		SchemaVersion: PublicScreenSchemaVersion, WorkItem: "SWL-014C",
		GeneratedAt: input.GeneratedAt, CohortManifestSHA256: input.CohortManifestSHA256,
		DetectorOrder: append([]string(nil), input.DetectorOrder...), Families: results,
		PassingFamilies: passing, MinimumPassingFamilies: input.MinimumPassing,
		GatePassed: passed, Outcome: outcome,
	}, nil
}

func screenFamily(family PublicFamilyInput, order []string, maximumBuilds uint64) (PublicFamilyResult, error) {
	detectors := make(map[string]PublicDetectorInput, len(family.Detectors))
	for _, detector := range family.Detectors {
		if detector.DetectorID == "" || detectors[detector.DetectorID].DetectorID != "" || !validSHA(detector.InputEvidenceSHA256) {
			return PublicFamilyResult{}, errors.New("detector identity is invalid")
		}
		detectors[detector.DetectorID] = detector
	}
	result := PublicFamilyResult{FamilyKey: family.FamilyKey, RevisionWindowSHA256: family.RevisionWindowSHA256}
	result.TestableActions = []PublicAction{}
	for _, detectorID := range order {
		detector, ok := detectors[detectorID]
		if !ok {
			return PublicFamilyResult{}, fmt.Errorf("detector %s is missing", detectorID)
		}
		detectorResult, actions, err := screenDetector(family, detector, maximumBuilds)
		if err != nil {
			return PublicFamilyResult{}, err
		}
		result.DetectorResults = append(result.DetectorResults, detectorResult)
		result.TestableActions = append(result.TestableActions, actions...)
	}
	sort.Slice(result.TestableActions, func(i, j int) bool { return result.TestableActions[i].ActionID < result.TestableActions[j].ActionID })
	result.Passed = len(result.TestableActions) > 0
	result.Reason = "NO_TESTABLE_ACTION"
	if result.Passed {
		result.Reason = "TESTABLE_ACTION_REPAYS_BOUNDED_TRIAL"
	}
	return result, nil
}

func screenDetector(family PublicFamilyInput, detector PublicDetectorInput, maximumBuilds uint64) (PublicDetectorResult, []PublicAction, error) {
	base := PublicDetectorResult{DetectorID: detector.DetectorID, InputEvidenceSHA256: detector.InputEvidenceSHA256, ActionIDs: []string{}}
	if detector.UnavailableReason != "" {
		if len(detector.GraphObservations) != 0 {
			return PublicDetectorResult{}, nil, errors.New("unavailable detector must not carry observations")
		}
		base.Status, base.Reason = DetectorStatusInputUnavailable, detector.UnavailableReason
		return base, nil, nil
	}
	if detector.DetectorID != DetectorGraphBreadth {
		return PublicDetectorResult{}, nil, fmt.Errorf("detector %s has no generic public input producer", detector.DetectorID)
	}
	actions, reason, err := graphActions(family, detector, maximumBuilds)
	if err != nil {
		return PublicDetectorResult{}, nil, err
	}
	if len(actions) == 0 {
		base.Status, base.Reason = DetectorStatusNoOpportunity, reason
		return base, nil, nil
	}
	base.Status, base.Reason = DetectorStatusTestableActions, "RECURRING_EXACT_ACTION_REPAYS_BOUNDED_TRIAL"
	for _, action := range actions {
		base.ActionIDs = append(base.ActionIDs, action.ActionID)
	}
	sort.Strings(base.ActionIDs)
	return base, actions, nil
}

func graphActions(family PublicFamilyInput, detector PublicDetectorInput, maximumBuilds uint64) ([]PublicAction, string, error) {
	groups := map[string][]PublicGraphObservation{}
	seenOrdinals := map[uint64]bool{}
	seenObservationIDs := map[string]bool{}
	for _, observation := range detector.GraphObservations {
		if observation.Ordinal == 0 || observation.ObservationID == "" || !validSHA(observation.CandidatePlanSHA256) ||
			!validSHA(observation.BindingDigest) || !validSHA(observation.FullOutputSHA256) ||
			!validSHA(observation.CandidateOutputSHA256) {
			return nil, "", errors.New("graph observation identity is invalid")
		}
		if seenOrdinals[observation.Ordinal] || seenObservationIDs[observation.ObservationID] {
			return nil, "", errors.New("graph observation recurrence is duplicated")
		}
		seenOrdinals[observation.Ordinal], seenObservationIDs[observation.ObservationID] = true, true
		key := observation.CandidatePlanSHA256 + "\x00" + observation.BindingDigest
		groups[key] = append(groups[key], observation)
	}
	var actions []PublicAction
	for _, observations := range groups {
		sort.Slice(observations, func(i, j int) bool { return observations[i].Ordinal < observations[j].Ordinal })
		if len(observations) < 3 {
			continue
		}
		last := observations[len(observations)-3:]
		graphInput := GraphBreadthInput{
			EvidenceSHA256:        detector.InputEvidenceSHA256,
			RepositoryScopeSHA256: family.RepositoryScopeSHA256,
			ManifestSHA256:        last[0].BindingDigest,
			GraphSHA256:           last[0].CandidatePlanSHA256,
			Workflow:              strings.Join(family.OutputContract, "\x00"),
		}
		projected := uint64(0)
		ids := make([]string, 0, 3)
		for index, observation := range last {
			if observation.CandidatePlanSHA256 != last[0].CandidatePlanSHA256 || observation.BindingDigest != last[0].BindingDigest {
				return nil, "", errors.New("grouped graph observation binding drifted")
			}
			if index == 0 || observation.ProjectedSavingNs < projected {
				projected = observation.ProjectedSavingNs
			}
			ids = append(ids, observation.ObservationID)
			graphInput.Observations = append(graphInput.Observations, GraphBreadthObservation{
				RequestedBuildOrdinal: observation.Ordinal,
				FullProjectCount:      observation.FullProjectCount,
				CandidateProjectCount: observation.CandidateProjectCount,
				FullOutputSHA256:      observation.FullOutputSHA256,
				CandidateOutputSHA256: observation.CandidateOutputSHA256,
				ProductFailure:        observation.ProductFailure,
			})
		}
		proposal, err := DetectGraphBreadthOpportunity(graphInput)
		if err != nil || proposal.Status != StatusProposed || projected == 0 {
			continue
		}
		totalCost, ok := checkedCost(detector.TrialCostNs, detector.ValidationCostNs, detector.PublicationCostNs)
		if !ok {
			return nil, "", errors.New("detector costs overflow")
		}
		if totalCost == 0 {
			continue
		}
		builds := totalCost / projected
		if totalCost%projected != 0 {
			builds++
		}
		if builds == 0 {
			builds = 1
		}
		if builds > maximumBuilds {
			continue
		}
		actionID := publicActionID(detector.DetectorID, family.RepositoryScopeSHA256, family.WorkflowArgumentsSHA256, last[0].CandidatePlanSHA256)
		actions = append(actions, PublicAction{
			ActionID: actionID, DetectorID: detector.DetectorID, ActionKind: KindGraphBreadth,
			BindingDigest: last[0].BindingDigest, RecurringObservationIDs: ids,
			RecurrenceCount: uint64(len(last)), ProjectedSavingNs: projected,
			TrialCostNs: detector.TrialCostNs, ValidationCostNs: detector.ValidationCostNs,
			PublicationCostNs: detector.PublicationCostNs, CompatibleBuildsToRepay: builds,
			OutputContract: append([]string(nil), family.OutputContract...), OwnerReviewRequired: true,
		})
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].ActionID < actions[j].ActionID })
	if len(actions) == 0 {
		return nil, "NO_RECURRING_ACTION_WITH_POSITIVE_PROJECTED_VALUE", nil
	}
	return actions, "", nil
}

func publicActionID(detectorID, repositoryScope, workflowArguments, candidatePlan string) string {
	preimage := strings.Join([]string{"buildopt-sticky-public-action-v1", detectorID, repositoryScope, workflowArguments, candidatePlan}, "\x00")
	digest := sha256.Sum256([]byte(preimage))
	return hex.EncodeToString(digest[:])
}

func sameDetectorOrder(order []string) bool {
	return len(order) == 2 && order[0] == DetectorTaskContract && order[1] == DetectorGraphBreadth
}

func checkedCost(values ...uint64) (uint64, bool) {
	var result uint64
	for _, value := range values {
		next := result + value
		if next < result {
			return 0, false
		}
		result = next
	}
	return result, true
}
