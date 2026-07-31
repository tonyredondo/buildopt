// Package taskintelligence qualifies reviewed custom-task contracts and keeps
// every diagnostic or publication path fail closed.
package taskintelligence

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
)

type State string

const (
	Unknown             State = "UNKNOWN"
	Observing           State = "OBSERVING"
	ContractQualified   State = "CONTRACT_QUALIFIED"
	QuarantineValidated State = "QUARANTINE_VALIDATED"
	Active              State = "ACTIVE"
	Suspended           State = "SUSPENDED"
)

type CoverageStatus string

const (
	CoverageExact       CoverageStatus = "EXACT"
	CoverageIncomplete  CoverageStatus = "INCOMPLETE"
	CoverageUnavailable CoverageStatus = "UNAVAILABLE"
)

var requiredDimensions = []string{"FILESYSTEM", "ENVIRONMENT", "PROCESS_TREE", "NETWORK", "CLOCK", "RANDOMNESS", "NATIVE_EXECUTABLE"}
var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type TraceCoverage struct {
	Dimensions map[string]CoverageStatus
	Dropped    uint64
	Fault      string
}

type TraceDecision struct {
	TraceComplete bool
	Qualification string
	Reason        string
}

// EvaluateTrace rejects missing dimensions, dropped events, and instrumentation faults.
func EvaluateTrace(coverage TraceCoverage) TraceDecision {
	if coverage.Dropped != 0 || coverage.Fault != "" {
		return TraceDecision{Qualification: "INCONCLUSIVE", Reason: "TRACE_FAULT"}
	}
	for _, dimension := range requiredDimensions {
		if coverage.Dimensions[dimension] != CoverageExact {
			return TraceDecision{Qualification: "INCONCLUSIVE", Reason: "INCOMPLETE_" + dimension}
		}
	}
	return TraceDecision{TraceComplete: true, Qualification: "ELIGIBLE", Reason: "COMPLETE"}
}

type RegisteredPath struct {
	Path   string
	Digest string
}

type ReviewedContract struct {
	TaskType              string
	ImplementationDigest  string
	ContractDigest        string
	Route                 string
	Inputs                []RegisteredPath
	Outputs               []string
	IsTest                bool
	ProducerProfileDigest string
}

type Qualification struct {
	State    State
	Contract ReviewedContract
	Reason   string
}

func NewQualification() Qualification { return Qualification{State: Unknown} }

func (q Qualification) Observe() (Qualification, error) {
	if q.State != Unknown {
		return q, errors.New("observe requires UNKNOWN")
	}
	q.State = Observing
	q.Reason = "EVIDENCE_ONLY"
	return q, nil
}

// QualifyReviewed accepts only an official or reviewed source contract. Trace
// history, the diagnostic agent, and an unavailable helper are never routes.
func (q Qualification) QualifyReviewed(contract ReviewedContract) (Qualification, error) {
	if q.State != Observing {
		return q, errors.New("qualify requires OBSERVING")
	}
	if contract.IsTest {
		return q, errors.New("Test tasks are excluded")
	}
	if contract.Route != "OFFICIAL_CONTRACT" && contract.Route != "REVIEWED_SOURCE_PATCH" && contract.Route != "HERMETIC_PRODUCER_PROFILE" {
		return q, errors.New("unsupported qualification route")
	}
	if contract.Route == "HERMETIC_PRODUCER_PROFILE" && !digestPattern.MatchString(contract.ProducerProfileDigest) {
		return q, errors.New("hermetic route requires one continuous producer profile")
	}
	if contract.TaskType == "" || !digestPattern.MatchString(contract.ImplementationDigest) || !digestPattern.MatchString(contract.ContractDigest) || len(contract.Inputs) == 0 || len(contract.Outputs) == 0 {
		return q, errors.New("incomplete reviewed contract")
	}
	seen := map[string]bool{}
	for _, input := range contract.Inputs {
		if input.Path == "" || !digestPattern.MatchString(input.Digest) || seen[input.Path] {
			return q, errors.New("invalid registered input")
		}
		seen[input.Path] = true
	}
	for _, output := range contract.Outputs {
		if output == "" || seen[output] {
			return q, errors.New("invalid or overlapping output")
		}
		seen[output] = true
	}
	q.State = ContractQualified
	q.Contract = defensiveContract(contract)
	q.Reason = "REVIEWED_CONTRACT"
	return q, nil
}

type QuarantineEvidence struct {
	EveryInputMutationChangedKey bool
	Repeatable                   bool
	Relocatable                  bool
	ArtifactsExact               bool
	RelevantValidationPassed     bool
	Discrepancy                  bool
}

func (q Qualification) ValidateQuarantine(evidence QuarantineEvidence) (Qualification, error) {
	if q.State != ContractQualified {
		return q, errors.New("quarantine validation requires CONTRACT_QUALIFIED")
	}
	if evidence.Discrepancy {
		q.State, q.Reason = Suspended, "QUARANTINE_DISCREPANCY"
		return q, nil
	}
	if !evidence.EveryInputMutationChangedKey || !evidence.Repeatable || !evidence.Relocatable || !evidence.ArtifactsExact || !evidence.RelevantValidationPassed {
		return q, errors.New("incomplete quarantine evidence")
	}
	q.State, q.Reason = QuarantineValidated, "QUARANTINE_PASSED"
	return q, nil
}

func (q Qualification) Activate() (Qualification, error) {
	if q.State != QuarantineValidated {
		return q, errors.New("activate requires QUARANTINE_VALIDATED")
	}
	q.State, q.Reason = Active, "ACTIVE_REVIEWED_CONTRACT"
	return q, nil
}

func (q Qualification) Suspend(reason string) Qualification {
	q.State = Suspended
	q.Reason = reason
	return q
}

type CorrelationEvent struct {
	TaskExecutionID string
	CacheKey        string
	TaskOutcome     string
	PutCompleted    bool
	Attributed      bool
}

type PublicationDecision struct {
	Capability     string
	AttemptAborted bool
	Keys           []string
	Reason         string
}

// Correlate permits selective publication only when every PUT is exactly
// bound to one successful task and the task contract is active.
func Correlate(state State, events []CorrelationEvent) PublicationDecision {
	if state != Active {
		return PublicationDecision{Capability: "UNAVAILABLE", AttemptAborted: true, Reason: "TASK_NOT_ACTIVE"}
	}
	if len(events) == 0 {
		return PublicationDecision{Capability: "UNAVAILABLE", AttemptAborted: true, Reason: "NO_EVENTS"}
	}
	keys := make([]string, 0, len(events))
	seen := map[string]bool{}
	for _, event := range events {
		if !event.Attributed || event.TaskExecutionID == "" || event.CacheKey == "" || !event.PutCompleted || event.TaskOutcome != "SUCCESS" {
			return PublicationDecision{Capability: "UNAVAILABLE", AttemptAborted: true, Reason: "UNATTRIBUTED"}
		}
		if seen[event.CacheKey] {
			return PublicationDecision{Capability: "UNAVAILABLE", AttemptAborted: true, Reason: "AMBIGUOUS_CACHE_KEY"}
		}
		seen[event.CacheKey] = true
		keys = append(keys, event.CacheKey)
	}
	sort.Strings(keys)
	return PublicationDecision{Capability: "EXACT", Keys: keys, Reason: "EXACT_TASK_KEY_PUT_OUTCOME"}
}

func defensiveContract(contract ReviewedContract) ReviewedContract {
	contract.Inputs = append([]RegisteredPath(nil), contract.Inputs...)
	contract.Outputs = append([]string(nil), contract.Outputs...)
	return contract
}

func (q Qualification) String() string { return fmt.Sprintf("%s:%s", q.State, q.Reason) }
