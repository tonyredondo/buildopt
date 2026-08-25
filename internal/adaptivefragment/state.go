package adaptivefragment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	FragmentStateSchemaVersion = "buildopt.adaptive/fragment/v1"
	ObservationSchemaVersion   = "buildopt.adaptive/observation/v1"
	PortfolioSchemaVersion     = "buildopt.adaptive/portfolio/v1"
	LedgerSchemaVersion        = "buildopt.adaptive/economic-ledger/v1"
)

// PersistedFragment is one immutable generation of an AF-001 fragment. A later
// lifecycle state creates a new document; it never rewrites this generation.
type PersistedFragment struct {
	SchemaVersion         string                `json:"schemaVersion"`
	RecordType            string                `json:"recordType"`
	FamilyID              string                `json:"familyId"`
	RevisionID            string                `json:"revisionId"`
	RepositoryScopeSHA256 string                `json:"repositoryScopeSha256"`
	Kind                  Kind                  `json:"kind"`
	SelectorSHA256        string                `json:"selectorSha256"`
	Authority             Authority             `json:"authority"`
	AuthoritySHA256       string                `json:"authoritySha256"`
	Bindings              map[BindingKey]string `json:"bindings"`
	Requires              []string              `json:"requires"`
	ConflictsWith         []string              `json:"conflictsWith"`
	State                 State                 `json:"state"`
	Generation            uint64                `json:"generation"`
	CreatedAt             string                `json:"createdAt"`
	UpdatedAt             string                `json:"updatedAt"`
	EvidenceExpiresAt     string                `json:"evidenceExpiresAt"`
}

// Observation is one append-only lifecycle transition for one exact fragment
// revision. ABSENT is valid only as the source of generation one.
type Observation struct {
	SchemaVersion          string `json:"schemaVersion"`
	RecordType             string `json:"recordType"`
	ObservationID          string `json:"observationId"`
	RepositoryScopeSHA256  string `json:"repositoryScopeSha256"`
	FamilyID               string `json:"familyId"`
	RevisionID             string `json:"revisionId"`
	Generation             uint64 `json:"generation"`
	FromState              string `json:"fromState"`
	ToState                State  `json:"toState"`
	Source                 string `json:"source"`
	ContextBindingsSHA256  string `json:"contextBindingsSha256"`
	EvidenceDocumentSHA256 string `json:"evidenceDocumentSha256"`
	ObservedAt             string `json:"observedAt"`
	NativeRetentionReason  string `json:"nativeRetentionReason,omitempty"`
}

// PortfolioEntry links a portfolio snapshot to one exact immutable fragment
// generation. It does not grant activation beyond the fragment's state.
type PortfolioEntry struct {
	FamilyID           string `json:"familyId"`
	RevisionID         string `json:"revisionId"`
	FragmentGeneration uint64 `json:"fragmentGeneration"`
	State              State  `json:"state"`
}

// Portfolio is one immutable repository-scoped set of current fragment
// generations. Superseded snapshots are linked by their external JCS digest.
type Portfolio struct {
	SchemaVersion         string           `json:"schemaVersion"`
	RecordType            string           `json:"recordType"`
	RepositoryScopeSHA256 string           `json:"repositoryScopeSha256"`
	Generation            uint64           `json:"generation"`
	SupersedesSHA256      string           `json:"supersedesSha256,omitempty"`
	Fragments             []PortfolioEntry `json:"fragments"`
	CreatedAt             string           `json:"createdAt"`
}

// LedgerEntry is the typed AF-002 container for signed observations. AF-005
// owns recurrence, decay, payback and decision formulas.
type LedgerEntry struct {
	FamilyID                string `json:"familyId"`
	RevisionID              string `json:"revisionId"`
	FragmentGeneration      uint64 `json:"fragmentGeneration"`
	RequestedBuildCount     uint64 `json:"requestedBuildCount"`
	GrossSavedMs            int64  `json:"grossSavedMs"`
	SynchronousOverheadMs   uint64 `json:"synchronousOverheadMs"`
	OutOfBandLearningCostMs uint64 `json:"outOfBandLearningCostMs"`
	CumulativeNetMs         int64  `json:"cumulativeNetMs"`
	LastObservedAt          string `json:"lastObservedAt"`
	EvidenceExpiresAt       string `json:"evidenceExpiresAt"`
}

// EconomicLedger is an immutable repository-scoped economic snapshot linked
// to exactly one portfolio generation.
type EconomicLedger struct {
	SchemaVersion         string        `json:"schemaVersion"`
	RecordType            string        `json:"recordType"`
	RepositoryScopeSHA256 string        `json:"repositoryScopeSha256"`
	Generation            uint64        `json:"generation"`
	SupersedesSHA256      string        `json:"supersedesSha256,omitempty"`
	PortfolioGeneration   uint64        `json:"portfolioGeneration"`
	Entries               []LedgerEntry `json:"entries"`
	AsOf                  string        `json:"asOf"`
}

// StateBundle is a conformance-only linked vector. Runtime persistence stores
// the four record types independently.
type StateBundle struct {
	Fragment     PersistedFragment `json:"fragment"`
	Observations []Observation     `json:"observations"`
	Portfolio    Portfolio         `json:"portfolio"`
	Ledger       EconomicLedger    `json:"ledger"`
}

// CanonicalDocumentSHA256 returns the lowercase SHA-256 of one RFC 8785
// canonical JSON document. The digest is stored by the containing manifest or
// reference, never inside the document being hashed.
func CanonicalDocumentSHA256(document []byte) (string, error) {
	canonical, err := contractcrypto.CanonicalizeJCS(document)
	if err != nil {
		return "", fmt.Errorf("canonicalize adaptive fragment document: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// ValidateStateBundle enforces cross-document AF-002 invariants that JSON
// Schema cannot express. Schema validation remains the first validation layer.
func ValidateStateBundle(bundle StateBundle) error {
	fragment := bundle.Fragment
	if err := validatePersistedFragment(fragment); err != nil {
		return err
	}
	if len(bundle.Observations) == 0 {
		return errors.New("adaptive fragment observations are empty")
	}
	if err := validateObservations(fragment, bundle.Observations); err != nil {
		return err
	}
	if err := validatePortfolio(fragment, bundle.Portfolio); err != nil {
		return err
	}
	if err := validateLedger(fragment, bundle.Portfolio, bundle.Ledger); err != nil {
		return err
	}
	return nil
}

func validatePersistedFragment(fragment PersistedFragment) error {
	if fragment.SchemaVersion != FragmentStateSchemaVersion || fragment.RecordType != "ADAPTIVE_FRAGMENT" {
		return errors.New("adaptive fragment record identity is invalid")
	}
	if fragment.Generation == 0 {
		return errors.New("adaptive fragment generation is invalid")
	}
	contract := Fragment{
		SchemaVersion:         SchemaVersion,
		FamilyID:              fragment.FamilyID,
		RevisionID:            fragment.RevisionID,
		RepositoryScopeSHA256: fragment.RepositoryScopeSHA256,
		Kind:                  fragment.Kind,
		SelectorSHA256:        fragment.SelectorSHA256,
		Authority:             fragment.Authority,
		AuthoritySHA256:       fragment.AuthoritySHA256,
		Bindings:              fragment.Bindings,
		Requires:              fragment.Requires,
		ConflictsWith:         fragment.ConflictsWith,
	}
	if !Valid(contract) {
		return errors.New("adaptive fragment semantic identity is invalid")
	}
	createdAt, err := parseUTC(fragment.CreatedAt)
	if err != nil {
		return errors.New("adaptive fragment createdAt is invalid")
	}
	updatedAt, err := parseUTC(fragment.UpdatedAt)
	if err != nil || updatedAt.Before(createdAt) {
		return errors.New("adaptive fragment updatedAt is invalid")
	}
	expiresAt, err := parseUTC(fragment.EvidenceExpiresAt)
	if err != nil || !expiresAt.After(updatedAt) {
		return errors.New("adaptive fragment evidence expiration is invalid")
	}
	return nil
}

func validateObservations(fragment PersistedFragment, observations []Observation) error {
	previousState := "ABSENT"
	var previousAt time.Time
	seenObservations := map[string]bool{}
	for index, observation := range observations {
		expectedGeneration := uint64(index + 1)
		if observation.SchemaVersion != ObservationSchemaVersion || observation.RecordType != "ADAPTIVE_FRAGMENT_OBSERVATION" ||
			observation.Generation != expectedGeneration {
			return errors.New("adaptive fragment observation generation is incompatible")
		}
		if observation.RepositoryScopeSHA256 != fragment.RepositoryScopeSHA256 ||
			observation.FamilyID != fragment.FamilyID || observation.RevisionID != fragment.RevisionID {
			return errors.New("adaptive fragment observation scope is incompatible")
		}
		if !validSHA(observation.ObservationID) || !validSHA(observation.ContextBindingsSHA256) ||
			!validSHA(observation.EvidenceDocumentSHA256) {
			return errors.New("adaptive fragment observation digest is invalid")
		}
		if seenObservations[observation.ObservationID] {
			return errors.New("adaptive fragment observation ID is duplicated")
		}
		seenObservations[observation.ObservationID] = true
		if observation.FromState != previousState {
			return errors.New("adaptive fragment observation lifecycle is discontinuous")
		}
		if index == 0 {
			if observation.ToState != StateObserved {
				return errors.New("adaptive fragment initial transition is impossible")
			}
		} else if !CanTransition(State(observation.FromState), observation.ToState) {
			return errors.New("adaptive fragment lifecycle transition is impossible")
		}
		observedAt, err := parseUTC(observation.ObservedAt)
		if err != nil || (!previousAt.IsZero() && observedAt.Before(previousAt)) {
			return errors.New("adaptive fragment observation time is invalid")
		}
		previousAt = observedAt
		previousState = string(observation.ToState)
	}
	if uint64(len(observations)) != fragment.Generation || State(previousState) != fragment.State {
		return errors.New("adaptive fragment current generation or state is incompatible")
	}
	updatedAt, _ := parseUTC(fragment.UpdatedAt)
	if !updatedAt.Equal(previousAt) {
		return errors.New("adaptive fragment update does not match its latest observation")
	}
	return nil
}

func validatePortfolio(fragment PersistedFragment, portfolio Portfolio) error {
	if portfolio.SchemaVersion != PortfolioSchemaVersion || portfolio.RecordType != "ADAPTIVE_FRAGMENT_PORTFOLIO" ||
		portfolio.Generation == 0 {
		return errors.New("adaptive fragment portfolio identity is invalid")
	}
	if portfolio.RepositoryScopeSHA256 != fragment.RepositoryScopeSHA256 {
		return errors.New("adaptive fragment portfolio repository scope is incompatible")
	}
	createdAt, err := parseUTC(portfolio.CreatedAt)
	if err != nil {
		return errors.New("adaptive fragment portfolio time is invalid")
	}
	fragmentUpdatedAt, _ := parseUTC(fragment.UpdatedAt)
	if createdAt.Before(fragmentUpdatedAt) {
		return errors.New("adaptive fragment portfolio predates its fragment")
	}
	if (portfolio.Generation == 1 && portfolio.SupersedesSHA256 != "") ||
		(portfolio.Generation > 1 && !validSHA(portfolio.SupersedesSHA256)) {
		return errors.New("adaptive fragment portfolio ancestry is invalid")
	}
	if len(portfolio.Fragments) == 0 {
		return errors.New("adaptive fragment portfolio is empty")
	}
	if !sort.SliceIsSorted(portfolio.Fragments, func(left, right int) bool {
		return portfolio.Fragments[left].FamilyID < portfolio.Fragments[right].FamilyID
	}) {
		return errors.New("adaptive fragment portfolio is not canonical")
	}
	found := false
	seen := map[string]bool{}
	for _, entry := range portfolio.Fragments {
		if seen[entry.FamilyID] {
			return errors.New("adaptive fragment portfolio contains duplicate families")
		}
		seen[entry.FamilyID] = true
		if entry.FamilyID == fragment.FamilyID {
			found = true
			if entry.RevisionID != fragment.RevisionID || entry.FragmentGeneration != fragment.Generation || entry.State != fragment.State {
				return errors.New("adaptive fragment portfolio generation is incompatible")
			}
		}
	}
	if !found {
		return errors.New("adaptive fragment portfolio does not reference the fragment")
	}
	return nil
}

func validateLedger(fragment PersistedFragment, portfolio Portfolio, ledger EconomicLedger) error {
	if ledger.SchemaVersion != LedgerSchemaVersion || ledger.RecordType != "ADAPTIVE_FRAGMENT_ECONOMIC_LEDGER" ||
		ledger.Generation == 0 {
		return errors.New("adaptive fragment ledger identity is invalid")
	}
	if ledger.RepositoryScopeSHA256 != fragment.RepositoryScopeSHA256 {
		return errors.New("adaptive fragment ledger repository scope is incompatible")
	}
	if ledger.PortfolioGeneration != portfolio.Generation {
		return errors.New("adaptive fragment ledger portfolio generation is incompatible")
	}
	if (ledger.Generation == 1 && ledger.SupersedesSHA256 != "") ||
		(ledger.Generation > 1 && !validSHA(ledger.SupersedesSHA256)) {
		return errors.New("adaptive fragment ledger ancestry is invalid")
	}
	asOf, err := parseUTC(ledger.AsOf)
	if err != nil {
		return errors.New("adaptive fragment ledger asOf is invalid")
	}
	if len(ledger.Entries) == 0 {
		return errors.New("adaptive fragment ledger is empty")
	}
	if !sort.SliceIsSorted(ledger.Entries, func(left, right int) bool {
		return ledger.Entries[left].FamilyID < ledger.Entries[right].FamilyID
	}) {
		return errors.New("adaptive fragment ledger is not canonical")
	}
	found := false
	seen := map[string]bool{}
	for _, entry := range ledger.Entries {
		if seen[entry.FamilyID] {
			return errors.New("adaptive fragment ledger contains duplicate families")
		}
		seen[entry.FamilyID] = true
		lastObservedAt, timeErr := parseUTC(entry.LastObservedAt)
		expiresAt, expiryErr := parseUTC(entry.EvidenceExpiresAt)
		if timeErr != nil || expiryErr != nil || !expiresAt.After(lastObservedAt) || asOf.Before(lastObservedAt) {
			return errors.New("adaptive fragment ledger time is invalid")
		}
		if entry.FamilyID == fragment.FamilyID {
			found = true
			if entry.RevisionID != fragment.RevisionID || entry.FragmentGeneration != fragment.Generation {
				return errors.New("adaptive fragment ledger generation is incompatible")
			}
		}
	}
	if !found {
		return errors.New("adaptive fragment ledger does not reference the fragment")
	}
	return nil
}

func parseUTC(value string) (time.Time, error) {
	if !strings.HasSuffix(value, "Z") {
		return time.Time{}, errors.New("timestamp is not UTC")
	}
	return time.Parse(time.RFC3339Nano, value)
}

// MarshalCanonicalDocument returns RFC 8785 bytes for a typed state document.
// It is used by conformance tests and later storage envelopes.
func MarshalCanonicalDocument(value any) ([]byte, error) {
	document, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal adaptive fragment document: %w", err)
	}
	canonical, err := contractcrypto.CanonicalizeJCS(document)
	if err != nil {
		return nil, fmt.Errorf("canonicalize adaptive fragment document: %w", err)
	}
	return canonical, nil
}
