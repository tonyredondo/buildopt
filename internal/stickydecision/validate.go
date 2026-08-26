package stickydecision

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

var validExecutionDecisions = map[string]bool{
	ExecutionNativeNoop:    true,
	ExecutionObserve:       true,
	ExecutionShadow:        true,
	ExecutionTrial:         true,
	ExecutionActiveRuntime: true,
	ExecutionActivePatch:   true,
	ExecutionSuspended:     true,
	ExecutionRetired:       true,
}

var validQualificationStates = map[string]bool{
	"UNKNOWN":              true,
	"OBSERVING":            true,
	"CONTRACT_QUALIFIED":   true,
	"QUARANTINE_VALIDATED": true,
	"REJECTED":             true,
	"SUSPENDED":            true,
}

var validRolloutStates = map[string]bool{
	"PROPOSED":       true,
	"SHADOW":         true,
	"CI_CANARY":      true,
	"ACTIVE_IN_CI":   true,
	"ACTIVE_LOCALLY": true,
	"SUSPENDED":      true,
	"ROLLED_BACK":    true,
}

var validOutcomes = map[string]bool{
	"SUCCESS":       true,
	"BUILD_FAILURE": true,
	"INFRA_FAILURE": true,
	"CANCELLED":     true,
}

var validEvidenceQuality = map[string]bool{
	"EXACT":        true,
	"APPROXIMATED": true,
	"UNAVAILABLE":  true,
}

const (
	maxInt64Value = int64(^uint64(0) >> 1)
	minInt64Value = -maxInt64Value - 1
)

// DecodeDocument verifies canonical bytes, the closed record shape and the
// local semantic invariants. A zero now skips expiry checks for inspection;
// stores always pass their current UTC clock.
func DecodeDocument(raw []byte, now time.Time) (Document, error) {
	canonicalRaw, err := equalCanonical(raw)
	if err != nil {
		return Document{}, err
	}
	var identity struct {
		RecordType string `json:"recordType"`
	}
	if err := json.Unmarshal(canonicalRaw, &identity); err != nil {
		return Document{}, fmt.Errorf("%w: %v", ErrInvalidDocument, err)
	}
	var document Document
	document.Raw = append([]byte(nil), canonicalRaw...)
	document.Digest = digestBytes(canonicalRaw)
	switch identity.RecordType {
	case ActionRecordType:
		var value ActionRecord
		if err := decodeStrict(canonicalRaw, &value); err != nil {
			return Document{}, fmt.Errorf("%w: action: %v", ErrInvalidDocument, err)
		}
		if err := validateAction(value, now); err != nil {
			return Document{}, err
		}
		document.RecordType, document.StoreGeneration, document.IdempotencyKey = value.RecordType, value.StoreGeneration, value.IdempotencyKey
		document.Binding, document.Action = value.Binding, &value
	case ObservationRecordType:
		var value Observation
		if err := decodeStrict(canonicalRaw, &value); err != nil {
			return Document{}, fmt.Errorf("%w: observation: %v", ErrInvalidDocument, err)
		}
		if err := validateObservation(value, now); err != nil {
			return Document{}, err
		}
		document.RecordType, document.StoreGeneration, document.IdempotencyKey = value.RecordType, value.StoreGeneration, value.IdempotencyKey
		document.Binding, document.Observation = value.Binding, &value
	case TrialRecordType:
		var value Trial
		if err := decodeStrict(canonicalRaw, &value); err != nil {
			return Document{}, fmt.Errorf("%w: trial: %v", ErrInvalidDocument, err)
		}
		if err := validateTrial(value, now); err != nil {
			return Document{}, err
		}
		document.RecordType, document.StoreGeneration, document.IdempotencyKey = value.RecordType, value.StoreGeneration, value.IdempotencyKey
		document.Binding, document.Trial = value.Binding, &value
	case DecisionRecordType:
		var value Decision
		if err := decodeStrict(canonicalRaw, &value); err != nil {
			return Document{}, fmt.Errorf("%w: decision: %v", ErrInvalidDocument, err)
		}
		if err := validateDecision(value, now); err != nil {
			return Document{}, err
		}
		document.RecordType, document.StoreGeneration, document.IdempotencyKey = value.RecordType, value.StoreGeneration, value.IdempotencyKey
		document.Binding, document.Decision, document.ExpiresAt = value.Binding, &value, mustParseTime(value.ExpiresAt)
	case LedgerRecordType:
		var value EconomicLedger
		if err := decodeStrict(canonicalRaw, &value); err != nil {
			return Document{}, fmt.Errorf("%w: ledger: %v", ErrInvalidDocument, err)
		}
		if err := validateLedger(value, now); err != nil {
			return Document{}, err
		}
		document.RecordType, document.StoreGeneration, document.IdempotencyKey = value.RecordType, value.StoreGeneration, value.IdempotencyKey
		document.Binding, document.Ledger = value.Binding, &value
	default:
		return Document{}, fmt.Errorf("%w: unsupported record type %q", ErrInvalidDocument, identity.RecordType)
	}
	return document, nil
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}

func validateBinding(binding Binding) error {
	if !digestPattern.MatchString(binding.RepositoryScopeSHA256) ||
		!identifierPattern.MatchString(binding.Workflow) ||
		!revisionPattern.MatchString(binding.SourceRevision) ||
		!gradlePattern.MatchString(binding.GradleVersion) ||
		!digestPattern.MatchString(binding.WrapperSHA256) ||
		!digestPattern.MatchString(binding.OptionsSHA256) ||
		!digestPattern.MatchString(binding.OutputContractSHA256) ||
		!digestPattern.MatchString(binding.BuildOptExecutableSHA256) ||
		binding.RevocationEpoch < 0 {
		return fmt.Errorf("%w: binding is invalid", ErrInvalidDocument)
	}
	return nil
}

func validateEnvelope(schema, recordType, id string, generation uint64, key string, binding Binding, at string, now time.Time) error {
	if schema == "" || recordType == "" || id == "" || !identifierPattern.MatchString(id) ||
		generation == 0 || !digestPattern.MatchString(key) {
		return fmt.Errorf("%w: envelope identity is invalid", ErrInvalidDocument)
	}
	if err := validateBinding(binding); err != nil {
		return err
	}
	if !contractcrypto.ValidUTCTimestamp(at) {
		return fmt.Errorf("%w: timestamp is invalid", ErrInvalidDocument)
	}
	if !now.IsZero() {
		observed := mustParseTime(at)
		if observed.After(now.UTC().Add(5 * time.Minute)) {
			return fmt.Errorf("%w: timestamp is in the future", ErrInvalidDocument)
		}
	}
	return nil
}

func validateAction(value ActionRecord, now time.Time) error {
	if value.SchemaVersion != ActionSchemaVersion || value.RecordType != ActionRecordType ||
		value.Sequence == 0 || value.Transition == "" ||
		!validQualificationStates[value.FromQualificationState] ||
		!validQualificationStates[value.ToQualificationState] ||
		!validRolloutStates[value.FromRolloutState] ||
		!validRolloutStates[value.ToRolloutState] {
		return fmt.Errorf("%w: action shape is invalid", ErrInvalidDocument)
	}
	if err := validateEnvelope(value.SchemaVersion, value.RecordType, value.ActionID, value.StoreGeneration, value.IdempotencyKey, value.Binding, value.OccurredAt, now); err != nil {
		return err
	}
	if err := ValidateActionTransition(value); err != nil {
		return err
	}
	if (value.Transition == "PROPOSE" && value.Sequence != 1) ||
		(value.Transition != "PROPOSE" && value.Sequence < 2) {
		return fmt.Errorf("%w: action sequence does not match transition", ErrInvalidDocument)
	}
	if value.Transition != "PROPOSE" && len(value.EvidenceRefs) == 0 {
		return fmt.Errorf("%w: action transition needs evidence", ErrInvalidDocument)
	}
	for _, ref := range value.EvidenceRefs {
		if !digestPattern.MatchString(ref) {
			return fmt.Errorf("%w: action evidence reference is invalid", ErrInvalidDocument)
		}
	}
	if value.DecisionDigest != "" && !digestPattern.MatchString(value.DecisionDigest) {
		return fmt.Errorf("%w: action decision reference is invalid", ErrInvalidDocument)
	}
	return nil
}

// ValidateActionTransition enforces the RFC's separate qualification and
// rollout state machines. It intentionally rejects direct UNKNOWN-to-ACTIVE
// transitions and any promotion from a suspended or rolled-back action.
func ValidateActionTransition(value ActionRecord) error {
	valid := false
	switch value.Transition {
	case "PROPOSE":
		valid = value.FromQualificationState == "UNKNOWN" && value.ToQualificationState == "OBSERVING" &&
			value.FromRolloutState == "PROPOSED" && value.ToRolloutState == "PROPOSED"
	case "BEGIN_SHADOW":
		valid = value.FromQualificationState == "OBSERVING" && value.ToQualificationState == "CONTRACT_QUALIFIED" &&
			value.FromRolloutState == "PROPOSED" && value.ToRolloutState == "SHADOW"
	case "BEGIN_CI_CANARY":
		valid = value.FromQualificationState == "CONTRACT_QUALIFIED" && value.ToQualificationState == "QUARANTINE_VALIDATED" &&
			value.FromRolloutState == "SHADOW" && value.ToRolloutState == "CI_CANARY"
	case "ACTIVATE_IN_CI":
		valid = value.FromQualificationState == "QUARANTINE_VALIDATED" && value.ToQualificationState == "QUARANTINE_VALIDATED" &&
			value.FromRolloutState == "CI_CANARY" && value.ToRolloutState == "ACTIVE_IN_CI"
	case "ACTIVATE_LOCALLY":
		valid = value.FromQualificationState == "QUARANTINE_VALIDATED" && value.ToQualificationState == "QUARANTINE_VALIDATED" &&
			value.FromRolloutState == "ACTIVE_IN_CI" && value.ToRolloutState == "ACTIVE_LOCALLY"
	case "SUSPEND":
		valid = value.FromQualificationState != "REJECTED" && value.FromQualificationState != "SUSPENDED" &&
			value.FromRolloutState != "SUSPENDED" && value.FromRolloutState != "ROLLED_BACK" &&
			value.ToQualificationState == "SUSPENDED" && value.ToRolloutState == "SUSPENDED"
	case "ROLLBACK":
		valid = value.FromQualificationState == "QUARANTINE_VALIDATED" &&
			(value.FromRolloutState == "CI_CANARY" || value.FromRolloutState == "ACTIVE_IN_CI" || value.FromRolloutState == "ACTIVE_LOCALLY") &&
			value.ToQualificationState == "SUSPENDED" && value.ToRolloutState == "ROLLED_BACK"
	case "RETIRE":
		valid = value.FromQualificationState == "SUSPENDED" && value.ToQualificationState == "REJECTED" &&
			(value.FromRolloutState == "SUSPENDED" || value.FromRolloutState == "ROLLED_BACK") && value.ToRolloutState == "ROLLED_BACK"
	}
	if !valid {
		return fmt.Errorf("%w: impossible action transition %s", ErrInvalidDocument, value.Transition)
	}
	return nil
}

func validateObservation(value Observation, now time.Time) error {
	if value.SchemaVersion != ObservationSchemaVersion || value.RecordType != ObservationRecordType ||
		value.ObservationKind != "REQUESTED_BUILD" || !validOutcomes[value.Outcome] ||
		!validEvidenceQuality[value.EvidenceQuality] {
		return fmt.Errorf("%w: observation shape is invalid", ErrInvalidDocument)
	}
	if err := validateEnvelope(value.SchemaVersion, value.RecordType, value.ObservationID, value.StoreGeneration, value.IdempotencyKey, value.Binding, value.RecordedAt, now); err != nil {
		return err
	}
	if value.EvidenceQuality == "UNAVAILABLE" && value.OutputDigest != "" {
		return fmt.Errorf("%w: unavailable observation cannot carry an output digest", ErrInvalidDocument)
	}
	if value.OutputDigest != "" && !digestPattern.MatchString(value.OutputDigest) {
		return fmt.Errorf("%w: observation output digest is invalid", ErrInvalidDocument)
	}
	return nil
}

func validateTrial(value Trial, now time.Time) error {
	if value.SchemaVersion != TrialSchemaVersion || value.RecordType != TrialRecordType ||
		value.ActionID == "" || !identifierPattern.MatchString(value.ActionID) ||
		!digestPattern.MatchString(value.IsolationDigest) ||
		!validOutcomes[value.CandidateOutcome] || !validOutcomes[value.ControlOutcome] ||
		(value.Equivalence != "EXACT" && value.Equivalence != "REVIEWED" && value.Equivalence != "NONE") ||
		(value.Result != "QUALIFIED" && value.Result != "RETAIN_NATIVE" && value.Result != "INCONCLUSIVE") {
		return fmt.Errorf("%w: trial shape is invalid", ErrInvalidDocument)
	}
	if err := validateEnvelope(value.SchemaVersion, value.RecordType, value.TrialID, value.StoreGeneration, value.IdempotencyKey, value.Binding, value.RecordedAt, now); err != nil {
		return err
	}
	if value.Result == "QUALIFIED" && (value.Equivalence == "NONE" || value.CandidateOutcome != "SUCCESS" || value.ControlOutcome != "SUCCESS") {
		return fmt.Errorf("%w: qualified trial lacks successful equivalent arms", ErrInvalidDocument)
	}
	if value.CandidateOutputDigest != "" && !digestPattern.MatchString(value.CandidateOutputDigest) {
		return fmt.Errorf("%w: candidate output digest is invalid", ErrInvalidDocument)
	}
	if value.ControlOutputDigest != "" && !digestPattern.MatchString(value.ControlOutputDigest) {
		return fmt.Errorf("%w: control output digest is invalid", ErrInvalidDocument)
	}
	if value.Equivalence == "EXACT" && (value.CandidateOutputDigest == "" || value.ControlOutputDigest == "" || value.CandidateOutputDigest != value.ControlOutputDigest) {
		return fmt.Errorf("%w: exact trial outputs differ", ErrInvalidDocument)
	}
	return nil
}

func validateDecision(value Decision, now time.Time) error {
	if value.SchemaVersion != DecisionSchemaVersion || value.RecordType != DecisionRecordType ||
		!validQualificationStates[value.QualificationState] || !validRolloutStates[value.RolloutState] ||
		!validExecutionDecisions[value.ExecutionDecision] ||
		!digestPattern.MatchString(value.PolicyDigest) || !digestPattern.MatchString(value.CacheContractDigest) ||
		value.Authentication.Algorithm != "Ed25519" ||
		!identifierPattern.MatchString(value.Authentication.KeyID) || value.Authentication.Signature == "" ||
		!contractcrypto.ValidUTCTimestamp(value.IssuedAt) || !contractcrypto.ValidUTCTimestamp(value.ExpiresAt) {
		return fmt.Errorf("%w: decision shape is invalid", ErrInvalidDocument)
	}
	if err := validateEnvelope(value.SchemaVersion, value.RecordType, value.DecisionID, value.StoreGeneration, value.IdempotencyKey, value.Binding, value.IssuedAt, now); err != nil {
		return err
	}
	issuedAt := mustParseTime(value.IssuedAt)
	expiresAt := mustParseTime(value.ExpiresAt)
	if !expiresAt.After(issuedAt) {
		return fmt.Errorf("%w: decision expiry is not after issuance", ErrInvalidDocument)
	}
	if !now.IsZero() && !now.UTC().Before(expiresAt) {
		return ErrExpired
	}
	if value.DecisionDigest != "" && !digestPattern.MatchString(value.DecisionDigest) {
		return fmt.Errorf("%w: decision digest is invalid", ErrInvalidDocument)
	}
	for _, ref := range value.EvidenceRefs {
		if !digestPattern.MatchString(ref) {
			return fmt.Errorf("%w: decision evidence reference is invalid", ErrInvalidDocument)
		}
	}
	active := value.ExecutionDecision == ExecutionActiveRuntime || value.ExecutionDecision == ExecutionActivePatch
	if active && (value.ActionID == "" || value.ActionGeneration == 0 || value.QualificationState != "QUARANTINE_VALIDATED" || (value.RolloutState != "ACTIVE_IN_CI" && value.RolloutState != "ACTIVE_LOCALLY") || len(value.EvidenceRefs) == 0) {
		return fmt.Errorf("%w: active decision is not fully qualified", ErrInvalidDocument)
	}
	if !active && (value.ActionID != "" && value.ActionGeneration == 0) {
		return fmt.Errorf("%w: action generation is missing", ErrInvalidDocument)
	}
	if value.ExecutionDecision == ExecutionNativeNoop && (value.ActionID != "" || value.ActionGeneration != 0) {
		return fmt.Errorf("%w: native no-op carries an action", ErrInvalidDocument)
	}
	if value.ExecutionDecision == ExecutionActiveRuntime && value.RolloutState == "ACTIVE_LOCALLY" && value.ActionID == "" {
		return fmt.Errorf("%w: local active decision lacks action", ErrInvalidDocument)
	}
	return nil
}

func validateLedger(value EconomicLedger, now time.Time) error {
	if value.SchemaVersion != LedgerSchemaVersion || value.RecordType != LedgerRecordType || len(value.Entries) == 0 {
		return fmt.Errorf("%w: ledger shape is invalid", ErrInvalidDocument)
	}
	if err := validateEnvelope(value.SchemaVersion, value.RecordType, value.LedgerID, value.StoreGeneration, value.IdempotencyKey, value.Binding, value.AsOf, now); err != nil {
		return err
	}
	if value.SupersedesDigest != "" && !digestPattern.MatchString(value.SupersedesDigest) {
		return fmt.Errorf("%w: ledger predecessor is invalid", ErrInvalidDocument)
	}
	var gross int64
	var cost uint64
	for _, entry := range value.Entries {
		if !identifierPattern.MatchString(entry.ActionID) || !digestPattern.MatchString(entry.ObservationRef) ||
			!validOutcomes[entry.Outcome] || !contractcrypto.ValidUTCTimestamp(entry.ObservedAt) {
			return fmt.Errorf("%w: ledger entry is invalid", ErrInvalidDocument)
		}
		if entry.BuildOptCostMs > uint64(maxInt64Value) {
			return fmt.Errorf("%w: ledger cost exceeds signed range", ErrInvalidDocument)
		}
		costMs := int64(entry.BuildOptCostMs)
		if entry.GrossSavedMs < minInt64Value+costMs || entry.NetSavedMs != entry.GrossSavedMs-costMs {
			return fmt.Errorf("%w: ledger entry arithmetic overflows or does not reconcile", ErrInvalidDocument)
		}
		if (gross > 0 && entry.GrossSavedMs > maxInt64Value-gross) ||
			(gross < 0 && entry.GrossSavedMs < minInt64Value-gross) {
			return fmt.Errorf("%w: ledger gross total overflows", ErrInvalidDocument)
		}
		if cost > ^uint64(0)-entry.BuildOptCostMs {
			return fmt.Errorf("%w: ledger cost total overflows", ErrInvalidDocument)
		}
		gross += entry.GrossSavedMs
		cost += entry.BuildOptCostMs
	}
	if cost > uint64(maxInt64Value) || value.GrossSavedMs != gross || value.BuildOptCostMs != cost || value.NetSavedMs != gross-int64(cost) {
		return fmt.Errorf("%w: ledger aggregate does not equal entries", ErrInvalidDocument)
	}
	return nil
}

func parseTime(value string) (time.Time, error) {
	if !contractcrypto.ValidUTCTimestamp(value) {
		return time.Time{}, errors.New("invalid UTC timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC {
		return time.Time{}, errors.New("invalid UTC timestamp")
	}
	return parsed, nil
}

func mustParseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed.UTC()
}

// ValidateChain verifies a complete chronological vector independently of
// either persistence implementation. It is used by the local and central
// conformance tests to ensure the same lifecycle semantics are observed.
func ValidateChain(documents []Document) error {
	if len(documents) == 0 {
		return fmt.Errorf("%w: empty lifecycle", ErrInvalidDocument)
	}
	var scope string
	var previousGeneration uint64
	lastAction := map[string]ActionRecord{}
	trialsByDigest := map[string]Trial{}
	observationsByDigest := map[string]Observation{}
	actions := map[string]ActionRecord{}
	for _, document := range documents {
		if document.RecordType == "" || document.StoreGeneration != previousGeneration+1 {
			return fmt.Errorf("%w: store generations are not contiguous", ErrGenerationConflict)
		}
		previousGeneration = document.StoreGeneration
		if scope == "" {
			scope = document.Binding.RepositoryScopeSHA256
		} else if document.Binding.RepositoryScopeSHA256 != scope {
			return ErrCrossScope
		}
		if document.Action != nil {
			prior, exists := lastAction[document.Action.ActionID]
			if document.Action.Sequence == 1 {
				if exists {
					return fmt.Errorf("%w: action sequence restarted", ErrInvalidDocument)
				}
			} else if !exists || document.Action.Sequence != prior.Sequence+1 ||
				document.Action.FromQualificationState != prior.ToQualificationState ||
				document.Action.FromRolloutState != prior.ToRolloutState {
				return fmt.Errorf("%w: action sequence is discontinuous", ErrInvalidDocument)
			}
			lastAction[document.Action.ActionID] = *document.Action
			actions[document.Action.ActionID] = *document.Action
		}
		if document.Trial != nil {
			trialsByDigest[document.Digest] = *document.Trial
		}
		if document.Observation != nil {
			observationsByDigest[document.Digest] = *document.Observation
		}
		if document.Decision != nil {
			decision := *document.Decision
			if decision.ExecutionDecision == ExecutionActiveRuntime || decision.ExecutionDecision == ExecutionActivePatch {
				if decision.QualificationState != "QUARANTINE_VALIDATED" || len(decision.EvidenceRefs) == 0 {
					return fmt.Errorf("%w: active decision lacks qualification evidence", ErrInvalidDocument)
				}
				for _, ref := range decision.EvidenceRefs {
					if trial, ok := trialsByDigest[ref]; ok {
						if trial.Result == "INCONCLUSIVE" {
							return fmt.Errorf("%w: inconclusive trial cannot authorize activation", ErrInvalidDocument)
						}
						continue
					}
					if _, ok := observationsByDigest[ref]; !ok {
						return fmt.Errorf("%w: active decision references unknown evidence", ErrInvalidDocument)
					}
				}
			}
		}
		if document.Ledger != nil {
			for _, entry := range document.Ledger.Entries {
				if _, ok := observationsByDigest[entry.ObservationRef]; !ok {
					return fmt.Errorf("%w: ledger references unknown observation", ErrInvalidDocument)
				}
				if _, ok := actions[entry.ActionID]; !ok && entry.ActionID != "native" {
					return fmt.Errorf("%w: ledger action reference is unknown", ErrInvalidDocument)
				}
			}
		}
	}
	return nil
}

// IsExpired reports whether a decoded decision has crossed its UTC expiry.
func (document Document) IsExpired(now time.Time) bool {
	return document.Decision != nil && !now.UTC().Before(document.ExpiresAt)
}

// HasControlPlaneIdentity makes the plane boundary explicit for callers that
// receive arbitrary JSON from a cache or network.
func (document Document) HasControlPlaneIdentity() bool {
	return document.RecordType == ActionRecordType || document.RecordType == ObservationRecordType ||
		document.RecordType == TrialRecordType || document.RecordType == DecisionRecordType || document.RecordType == LedgerRecordType
}

func invalidDocument(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidDocument, strings.TrimSpace(fmt.Sprintf(format, args...)))
}
