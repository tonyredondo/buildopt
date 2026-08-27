package durablecatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	FreshEvidenceSchemaVersion = "buildopt.poc/sticky-public-evidence/v1"

	CaptureComplete    = "COMPLETE"
	CaptureUnavailable = "UNAVAILABLE"
	CaptureFailed      = "FAILED"

	DetectorStatusNotApplicable  = "NOT_APPLICABLE"
	DetectorStatusProducerFailed = "PRODUCER_FAILED"
)

// FreshEvidenceInput is the generic detector boundary used before public
// repository screening. DSL is provenance only; no detector may branch on it.
type FreshEvidenceInput struct {
	SchemaVersion           string             `json:"schemaVersion"`
	GeneratedAt             string             `json:"generatedAt"`
	FamilyKey               string             `json:"familyKey"`
	DSL                     string             `json:"dsl"`
	RepositoryScopeSHA256   string             `json:"repositoryScopeSha256"`
	WorkflowArgumentsSHA256 string             `json:"workflowArgumentsSha256"`
	OutputContract          []string           `json:"outputContract"`
	TaskContract            TaskProducerInput  `json:"taskContract"`
	DeclaredGraph           GraphProducerInput `json:"declaredGraph"`
}

// TaskProducerInput carries a complete generic scan or an explicit capture
// failure. Complete scans with no Java custom tasks are NOT_APPLICABLE.
type TaskProducerInput struct {
	CaptureStatus string                  `json:"captureStatus"`
	Reason        string                  `json:"reason"`
	Candidates    []TaskCandidateEvidence `json:"candidates"`
}

type TaskCandidateEvidence struct {
	Input             adaptivefragment.PatchOpportunityInput `json:"input"`
	SourcePreimage    string                                 `json:"sourcePreimage"`
	SourcePostimage   string                                 `json:"sourcePostimage"`
	ValidationCommand []string                               `json:"validationCommand"`
}

// GraphProducerInput has the same completeness boundary. A complete graph
// scan with no safe reduction is NO_OPPORTUNITY rather than unavailable.
type GraphProducerInput struct {
	CaptureStatus string                   `json:"captureStatus"`
	Reason        string                   `json:"reason"`
	Candidates    []GraphCandidateEvidence `json:"candidates"`
}

type GraphCandidateEvidence struct {
	Input                     GraphBreadthInput `json:"input"`
	CandidatePlanSHA256       string            `json:"candidatePlanSha256"`
	BindingDigest             string            `json:"bindingDigest"`
	OmittedCriticalPathSHA256 string            `json:"omittedCriticalPathSha256"`
	ExactOutputClosureSHA256  string            `json:"exactOutputClosureSha256"`
	ProjectedSavingNs         uint64            `json:"projectedSavingNs"`
	SourcePreimage            string            `json:"sourcePreimage"`
	SourcePostimage           string            `json:"sourcePostimage"`
	ValidationCommand         []string          `json:"validationCommand"`
}

type UnavailableCost struct {
	Status string `json:"status"`
}

type FreshPublicAction struct {
	ActionID                   string           `json:"actionId"`
	DetectorID                 string           `json:"detectorId"`
	ActionKind                 string           `json:"actionKind"`
	BindingDigest              string           `json:"bindingDigest"`
	RelativeSourcePath         string           `json:"relativeSourcePath,omitempty"`
	CandidatePlanSHA256        string           `json:"candidatePlanSha256,omitempty"`
	EvidenceSHA256             string           `json:"evidenceSha256"`
	TaskImplementationSHA256   string           `json:"taskImplementationSha256,omitempty"`
	InputSnapshotSHA256        string           `json:"inputSnapshotSha256,omitempty"`
	OutputSnapshotSHA256       string           `json:"outputSnapshotSha256,omitempty"`
	FullGraphSHA256            string           `json:"fullGraphSha256,omitempty"`
	OmittedCriticalPathSHA256  string           `json:"omittedCriticalPathSha256,omitempty"`
	ExactOutputClosureSHA256   string           `json:"exactOutputClosureSha256,omitempty"`
	RequestedBuilds            uint64           `json:"requestedBuilds"`
	DeclaredInputCount         uint64           `json:"declaredInputCount,omitempty"`
	DeclaredOutputCount        uint64           `json:"declaredOutputCount,omitempty"`
	CacheableBefore            bool             `json:"cacheableBefore"`
	RequestedWorkflowReachable bool             `json:"requestedWorkflowReachable"`
	OutputContract             []string         `json:"outputContract"`
	ValidationCommand          []string         `json:"validationCommand"`
	Transaction                PatchTransaction `json:"transaction"`
	ProjectedSavingNs          uint64           `json:"projectedSavingNs,omitempty"`
	TrialCost                  UnavailableCost  `json:"trialCost"`
	ValidationCost             UnavailableCost  `json:"validationCost"`
	PublicationCost            UnavailableCost  `json:"publicationCost"`
	OwnerReviewRequired        bool             `json:"ownerReviewRequired"`
	PatchAuthorized            bool             `json:"patchAuthorized"`
	ActivationAuthorized       bool             `json:"activationAuthorized"`
}

type FreshDetectorResult struct {
	DetectorID string              `json:"detectorId"`
	Status     string              `json:"status"`
	Reason     string              `json:"reason"`
	Actions    []FreshPublicAction `json:"actions"`
}

type FreshEvidenceReport struct {
	SchemaVersion           string                `json:"schemaVersion"`
	GeneratedAt             string                `json:"generatedAt"`
	FamilyKey               string                `json:"familyKey"`
	DSL                     string                `json:"dsl"`
	RepositoryScopeSHA256   string                `json:"repositoryScopeSha256"`
	WorkflowArgumentsSHA256 string                `json:"workflowArgumentsSha256"`
	OutputContract          []string              `json:"outputContract"`
	Detectors               []FreshDetectorResult `json:"detectors"`
	InputComplete           bool                  `json:"inputComplete"`
	TestableActions         uint64                `json:"testableActions"`
}

// ProduceFreshEvidence evaluates both detector families without conflating
// unavailable input with a conclusive absence of opportunity.
func ProduceFreshEvidence(input FreshEvidenceInput) (FreshEvidenceReport, error) {
	if input.SchemaVersion != FreshEvidenceSchemaVersion || input.GeneratedAt == "" ||
		!safeLabel(input.FamilyKey) || (input.DSL != "KOTLIN" && input.DSL != "GROOVY") ||
		!validSHA(input.RepositoryScopeSHA256) || !validSHA(input.WorkflowArgumentsSHA256) ||
		!validOutputContract(input.OutputContract) {
		return FreshEvidenceReport{}, errors.New("fresh public evidence identity is invalid")
	}
	task, err := produceTaskEvidence(input)
	if err != nil {
		return FreshEvidenceReport{}, fmt.Errorf("task-contract producer: %w", err)
	}
	graph, err := produceGraphEvidence(input)
	if err != nil {
		return FreshEvidenceReport{}, fmt.Errorf("declared-graph producer: %w", err)
	}
	actions := uint64(len(task.Actions) + len(graph.Actions))
	complete := conclusiveDetectorStatus(task.Status) && conclusiveDetectorStatus(graph.Status)
	return FreshEvidenceReport{
		SchemaVersion: FreshEvidenceSchemaVersion, GeneratedAt: input.GeneratedAt,
		FamilyKey: input.FamilyKey, DSL: input.DSL,
		RepositoryScopeSHA256:   input.RepositoryScopeSHA256,
		WorkflowArgumentsSHA256: input.WorkflowArgumentsSHA256,
		OutputContract:          append([]string(nil), input.OutputContract...),
		Detectors:               []FreshDetectorResult{task, graph}, InputComplete: complete,
		TestableActions: actions,
	}, nil
}

func produceTaskEvidence(input FreshEvidenceInput) (FreshDetectorResult, error) {
	base, terminal, err := captureBoundary(DetectorTaskContract, input.TaskContract.CaptureStatus,
		input.TaskContract.Reason, len(input.TaskContract.Candidates))
	if err != nil || terminal {
		return base, err
	}
	if len(input.TaskContract.Candidates) == 0 {
		base.Status, base.Reason = DetectorStatusNotApplicable, "NO_JAVA_CUSTOM_TASK_IN_REQUESTED_WORKFLOW"
		return base, nil
	}
	for _, candidate := range input.TaskContract.Candidates {
		if DigestBytes([]byte(candidate.SourcePreimage)) != candidate.Input.SourcePreimageSHA256 ||
			len(candidate.ValidationCommand) == 0 || !safeCommand(candidate.ValidationCommand) {
			return FreshDetectorResult{}, errors.New("task candidate source or validation binding is invalid")
		}
		proposal, proposalErr := TaskProposal(candidate.Input)
		if proposalErr != nil {
			continue
		}
		transaction, transactionErr := ProvePatchTransaction(
			[]byte(candidate.SourcePreimage), []byte(candidate.SourcePostimage))
		if transactionErr != nil {
			return FreshDetectorResult{}, transactionErr
		}
		binding := freshActionBinding(DetectorTaskContract, input.RepositoryScopeSHA256,
			input.WorkflowArgumentsSHA256, proposal.RelativeSourcePath,
			candidate.Input.EvidenceSHA256, candidate.Input.TaskImplementationSHA256,
			candidate.Input.Observations[0].InputSnapshotSHA256,
			candidate.Input.Observations[0].OutputSnapshotSHA256,
			transaction.PreimageSHA256, transaction.PostimageSHA256)
		action := newFreshAction(input, DetectorTaskContract,
			KindTaskContract, binding, proposal.RelativeSourcePath, "", 0,
			candidate.ValidationCommand, transaction)
		action.EvidenceSHA256 = candidate.Input.EvidenceSHA256
		action.TaskImplementationSHA256 = candidate.Input.TaskImplementationSHA256
		action.InputSnapshotSHA256 = candidate.Input.Observations[0].InputSnapshotSHA256
		action.OutputSnapshotSHA256 = candidate.Input.Observations[0].OutputSnapshotSHA256
		action.RequestedBuilds = proposal.RequestedBuilds
		action.DeclaredInputCount = candidate.Input.Facts.InternalInputCount
		action.DeclaredOutputCount = candidate.Input.Facts.InternalOutputCount
		action.CacheableBefore = false
		action.RequestedWorkflowReachable = true
		base.Actions = append(base.Actions, action)
	}
	return finishFreshDetector(base), nil
}

func produceGraphEvidence(input FreshEvidenceInput) (FreshDetectorResult, error) {
	base, terminal, err := captureBoundary(DetectorGraphBreadth, input.DeclaredGraph.CaptureStatus,
		input.DeclaredGraph.Reason, len(input.DeclaredGraph.Candidates))
	if err != nil || terminal {
		return base, err
	}
	for _, candidate := range input.DeclaredGraph.Candidates {
		if !validSHA(candidate.CandidatePlanSHA256) || !validSHA(candidate.BindingDigest) ||
			!validSHA(candidate.OmittedCriticalPathSHA256) ||
			!validSHA(candidate.ExactOutputClosureSHA256) ||
			candidate.ProjectedSavingNs == 0 || len(candidate.ValidationCommand) == 0 ||
			!safeCommand(candidate.ValidationCommand) {
			return FreshDetectorResult{}, errors.New("graph candidate binding is invalid")
		}
		_, proposalErr := DetectGraphBreadthOpportunity(candidate.Input)
		if proposalErr != nil {
			continue
		}
		transaction, transactionErr := ProvePatchTransaction(
			[]byte(candidate.SourcePreimage), []byte(candidate.SourcePostimage))
		if transactionErr != nil {
			return FreshDetectorResult{}, transactionErr
		}
		binding := freshActionBinding(DetectorGraphBreadth, input.RepositoryScopeSHA256,
			input.WorkflowArgumentsSHA256, candidate.CandidatePlanSHA256,
			candidate.BindingDigest, candidate.Input.GraphSHA256,
			candidate.OmittedCriticalPathSHA256, candidate.ExactOutputClosureSHA256,
			transaction.PostimageSHA256)
		action := newFreshAction(input, DetectorGraphBreadth,
			KindGraphBreadth, binding, "", candidate.CandidatePlanSHA256,
			candidate.ProjectedSavingNs, candidate.ValidationCommand, transaction)
		action.EvidenceSHA256 = candidate.Input.EvidenceSHA256
		action.FullGraphSHA256 = candidate.Input.GraphSHA256
		action.OmittedCriticalPathSHA256 = candidate.OmittedCriticalPathSHA256
		action.ExactOutputClosureSHA256 = candidate.ExactOutputClosureSHA256
		action.RequestedBuilds = uint64(len(candidate.Input.Observations))
		action.RequestedWorkflowReachable = true
		base.Actions = append(base.Actions, action)
	}
	return finishFreshDetector(base), nil
}

func captureBoundary(detectorID, status, reason string, candidates int) (FreshDetectorResult, bool, error) {
	base := FreshDetectorResult{DetectorID: detectorID, Actions: []FreshPublicAction{}}
	switch status {
	case CaptureComplete:
		if reason != "" {
			return FreshDetectorResult{}, false, errors.New("complete capture must not have a failure reason")
		}
		return base, false, nil
	case CaptureUnavailable:
		if reason == "" || candidates != 0 {
			return FreshDetectorResult{}, false, errors.New("unavailable capture boundary is invalid")
		}
		base.Status, base.Reason = DetectorStatusInputUnavailable, reason
		return base, true, nil
	case CaptureFailed:
		if reason == "" || candidates != 0 {
			return FreshDetectorResult{}, false, errors.New("failed capture boundary is invalid")
		}
		base.Status, base.Reason = DetectorStatusProducerFailed, reason
		return base, true, nil
	default:
		return FreshDetectorResult{}, false, errors.New("unknown capture status")
	}
}

func finishFreshDetector(result FreshDetectorResult) FreshDetectorResult {
	sort.Slice(result.Actions, func(i, j int) bool { return result.Actions[i].ActionID < result.Actions[j].ActionID })
	if len(result.Actions) == 0 {
		result.Status, result.Reason = DetectorStatusNoOpportunity, "COMPLETE_EVIDENCE_HAS_NO_SAFE_ACTION"
		return result
	}
	result.Status, result.Reason = DetectorStatusTestableActions, "COMPLETE_INDEPENDENTLY_TESTABLE_ACTIONS"
	return result
}

func newFreshAction(input FreshEvidenceInput, detectorID, kind, binding, sourcePath,
	plan string, saving uint64, command []string, transaction PatchTransaction) FreshPublicAction {
	return FreshPublicAction{
		ActionID: freshActionID(detectorID, binding), DetectorID: detectorID,
		ActionKind: kind, BindingDigest: binding, RelativeSourcePath: sourcePath,
		CandidatePlanSHA256: plan, OutputContract: append([]string(nil), input.OutputContract...),
		ValidationCommand: append([]string(nil), command...), Transaction: transaction,
		ProjectedSavingNs:   saving,
		TrialCost:           UnavailableCost{Status: "UNAVAILABLE_NOT_MEASURED"},
		ValidationCost:      UnavailableCost{Status: "UNAVAILABLE_NOT_MEASURED"},
		PublicationCost:     UnavailableCost{Status: "UNAVAILABLE_NOT_MEASURED"},
		OwnerReviewRequired: true, PatchAuthorized: false, ActivationAuthorized: false,
	}
}

func conclusiveDetectorStatus(status string) bool {
	return status == DetectorStatusTestableActions || status == DetectorStatusNoOpportunity ||
		status == DetectorStatusNotApplicable
}

func validOutputContract(values []string) bool {
	if len(values) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !safeLabel(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func safeCommand(command []string) bool {
	for _, value := range command {
		if !safeLabel(value) {
			return false
		}
	}
	return len(command) > 0
}

func freshActionBinding(parts ...string) string {
	preimage := strings.Join(append([]string{"buildopt-fresh-public-action-binding-v1"}, parts...), "\x00")
	sum := sha256.Sum256([]byte(preimage))
	return hex.EncodeToString(sum[:])
}

func freshActionID(detectorID, binding string) string {
	return freshActionBinding(detectorID, binding)
}
