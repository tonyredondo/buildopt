// Package stickydecision owns the canonical control-plane records used by the
// sticky-wrapper learning POC. These records are deliberately separate from
// Gradle cache objects: a cache key or hit can never authorize an action.
package stickydecision

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	ActionSchemaVersion      = "buildopt.sticky/action/v1"
	ObservationSchemaVersion = "buildopt.sticky/observation/v1"
	TrialSchemaVersion       = "buildopt.sticky/trial/v1"
	DecisionSchemaVersion    = "buildopt.sticky/decision/v1"
	LedgerSchemaVersion      = "buildopt.sticky/economic-ledger/v1"
	StateHeadSchemaVersion   = "buildopt.sticky/state-head/v1"
	RevocationSchemaVersion  = "buildopt.sticky/revocation/v1"
	ActionRecordType         = "STICKY_ACTION"
	ObservationRecordType    = "STICKY_OBSERVATION"
	TrialRecordType          = "STICKY_TRIAL"
	DecisionRecordType       = "STICKY_DECISION"
	LedgerRecordType         = "STICKY_ECONOMIC_LEDGER"
	StateHeadRecordType      = "STICKY_STATE_HEAD"
	RevocationRecordType     = "STICKY_REVOCATION"
	ControlPlane             = "BUILDOPT_STATE"
	ExecutionNativeNoop      = "NATIVE_NOOP"
	ExecutionObserve         = "OBSERVE"
	ExecutionShadow          = "SHADOW"
	ExecutionTrial           = "TRIAL"
	ExecutionActiveRuntime   = "ACTIVE_RUNTIME_PROFILE"
	ExecutionActivePatch     = "ACTIVE_DURABLE_PATCH"
	ExecutionSuspended       = "SUSPENDED"
	ExecutionRetired         = "RETIRED"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
	digestPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	revisionPattern   = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	gradlePattern     = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`)
)

var (
	ErrInvalidDocument     = errors.New("sticky decision document is invalid")
	ErrExpired             = errors.New("sticky decision document is expired")
	ErrRevoked             = errors.New("sticky decision document is revoked")
	ErrNotFound            = errors.New("sticky decision state is absent")
	ErrGenerationConflict  = errors.New("sticky decision generation conflicts")
	ErrIdempotencyConflict = errors.New("sticky decision idempotency key conflicts")
	ErrCorrupt             = errors.New("sticky decision state is corrupt")
	ErrCrossScope          = errors.New("sticky decision scope does not match the store")
	ErrCrossPlane          = errors.New("sticky decision document is not a control-plane record")
	ErrBusy                = errors.New("sticky decision store is busy")
)

// Binding is the compatibility boundary shared by every sticky control
// document. A change to any digest or revision requires a new decision
// generation; Gradle cache keys are intentionally absent from this type.
type Binding struct {
	RepositoryScopeSHA256    string `json:"repositoryScopeSha256"`
	Workflow                 string `json:"workflow"`
	SourceRevision           string `json:"sourceRevision"`
	GradleVersion            string `json:"gradleVersion"`
	WrapperSHA256            string `json:"wrapperSha256"`
	OptionsSHA256            string `json:"optionsSha256"`
	OutputContractSHA256     string `json:"outputContractSha256"`
	BuildOptExecutableSHA256 string `json:"buildoptExecutableSha256"`
	RevocationEpoch          int64  `json:"revocationEpoch"`
}

// ActionRecord records one append-only qualification or rollout transition.
// It audits a state change; it is never an executable instruction by itself.
type ActionRecord struct {
	SchemaVersion          string   `json:"schemaVersion"`
	RecordType             string   `json:"recordType"`
	ActionID               string   `json:"actionId"`
	StoreGeneration        uint64   `json:"storeGeneration"`
	IdempotencyKey         string   `json:"idempotencyKey"`
	Sequence               uint64   `json:"sequence"`
	Transition             string   `json:"transition"`
	FromQualificationState string   `json:"fromQualificationState"`
	ToQualificationState   string   `json:"toQualificationState"`
	FromRolloutState       string   `json:"fromRolloutState"`
	ToRolloutState         string   `json:"toRolloutState"`
	Binding                Binding  `json:"binding"`
	DecisionDigest         string   `json:"decisionDigest,omitempty"`
	EvidenceRefs           []string `json:"evidenceRefs,omitempty"`
	OccurredAt             string   `json:"occurredAt"`
}

// Observation records one requested ordinary build. It may describe a
// native fallback, but it never promotes an action without later evidence.
type Observation struct {
	SchemaVersion   string  `json:"schemaVersion"`
	RecordType      string  `json:"recordType"`
	ObservationID   string  `json:"observationId"`
	StoreGeneration uint64  `json:"storeGeneration"`
	IdempotencyKey  string  `json:"idempotencyKey"`
	Binding         Binding `json:"binding"`
	ObservationKind string  `json:"observationKind"`
	Outcome         string  `json:"outcome"`
	WallTimeMs      uint64  `json:"wallTimeMs"`
	OutputDigest    string  `json:"outputDigest,omitempty"`
	EvidenceQuality string  `json:"evidenceQuality"`
	RecordedAt      string  `json:"recordedAt"`
}

// Trial records the isolated candidate/control comparison that can support a
// later decision. INCONCLUSIVE trials are explicitly non-promoting.
type Trial struct {
	SchemaVersion         string  `json:"schemaVersion"`
	RecordType            string  `json:"recordType"`
	TrialID               string  `json:"trialId"`
	StoreGeneration       uint64  `json:"storeGeneration"`
	IdempotencyKey        string  `json:"idempotencyKey"`
	ActionID              string  `json:"actionId"`
	Binding               Binding `json:"binding"`
	IsolationDigest       string  `json:"isolationDigest"`
	CandidateOutcome      string  `json:"candidateOutcome"`
	ControlOutcome        string  `json:"controlOutcome"`
	CandidateOutputDigest string  `json:"candidateOutputDigest,omitempty"`
	ControlOutputDigest   string  `json:"controlOutputDigest,omitempty"`
	Equivalence           string  `json:"equivalence"`
	Result                string  `json:"result"`
	CandidateWallTimeMs   uint64  `json:"candidateWallTimeMs"`
	ControlWallTimeMs     uint64  `json:"controlWallTimeMs"`
	LearningCostMs        uint64  `json:"learningCostMs"`
	RecordedAt            string  `json:"recordedAt"`
}

// Decision is the canonical signed authorization consulted by a later fast
// path. It binds every compatibility input and has an explicit expiry and
// revocation epoch. The signature covers the unsigned canonical payload.
type Decision struct {
	SchemaVersion       string         `json:"schemaVersion"`
	RecordType          string         `json:"recordType"`
	DecisionID          string         `json:"decisionId"`
	StoreGeneration     uint64         `json:"storeGeneration"`
	IdempotencyKey      string         `json:"idempotencyKey"`
	Binding             Binding        `json:"binding"`
	ActionID            string         `json:"actionId,omitempty"`
	ActionGeneration    uint64         `json:"actionGeneration"`
	QualificationState  string         `json:"qualificationState"`
	RolloutState        string         `json:"rolloutState"`
	ExecutionDecision   string         `json:"executionDecision"`
	PolicyDigest        string         `json:"policyDigest"`
	CacheContractDigest string         `json:"cacheContractDigest"`
	EvidenceRefs        []string       `json:"evidenceRefs"`
	DecisionDigest      string         `json:"decisionDigest"`
	IssuedAt            string         `json:"issuedAt"`
	ExpiresAt           string         `json:"expiresAt"`
	Authentication      Authentication `json:"authentication"`
}

// Authentication identifies the owner key that signed a decision.
type Authentication struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

// LedgerEntry is one signed wall-time/cost contribution. Savings remain
// signed: a negative entry is evidence of a regression, not a zero.
type LedgerEntry struct {
	ActionID       string `json:"actionId"`
	ObservationRef string `json:"observationRef"`
	GrossSavedMs   int64  `json:"grossSavedMs"`
	BuildOptCostMs uint64 `json:"buildoptCostMs"`
	NetSavedMs     int64  `json:"netSavedMs"`
	Outcome        string `json:"outcome"`
	ObservedAt     string `json:"observedAt"`
}

// EconomicLedger is an immutable aggregate over prior observations and
// trials. Its aggregate values must equal the sum of its entries.
type EconomicLedger struct {
	SchemaVersion    string        `json:"schemaVersion"`
	RecordType       string        `json:"recordType"`
	LedgerID         string        `json:"ledgerId"`
	StoreGeneration  uint64        `json:"storeGeneration"`
	IdempotencyKey   string        `json:"idempotencyKey"`
	Binding          Binding       `json:"binding"`
	SupersedesDigest string        `json:"supersedesDigest,omitempty"`
	Entries          []LedgerEntry `json:"entries"`
	GrossSavedMs     int64         `json:"grossSavedMs"`
	BuildOptCostMs   uint64        `json:"buildoptCostMs"`
	NetSavedMs       int64         `json:"netSavedMs"`
	AsOf             string        `json:"asOf"`
}

// StateHead is the only mutable pointer in either store. All documents it
// references remain immutable and content addressed.
type StateHead struct {
	SchemaVersion         string `json:"schemaVersion"`
	RecordType            string `json:"recordType"`
	Plane                 string `json:"plane"`
	RepositoryScopeSHA256 string `json:"repositoryScopeSha256"`
	Generation            uint64 `json:"generation"`
	RecordTypeAtHead      string `json:"recordTypeAtHead"`
	RecordDigest          string `json:"recordDigest"`
	RevocationEpoch       int64  `json:"revocationEpoch"`
	UpdatedAt             string `json:"updatedAt"`
}

// Revocation is a small mutable authority input kept outside the record
// chain. It is supplied by the owner-operated POC service, not by Gradle.
type Revocation struct {
	SchemaVersion         string `json:"schemaVersion"`
	RecordType            string `json:"recordType"`
	RepositoryScopeSHA256 string `json:"repositoryScopeSha256"`
	Epoch                 int64  `json:"epoch"`
	UpdatedAt             string `json:"updatedAt"`
}

// Document is the decoded union returned by DecodeDocument and stores.
type Document struct {
	Raw             []byte
	Digest          string
	RecordType      string
	StoreGeneration uint64
	IdempotencyKey  string
	Binding         Binding
	ExpiresAt       time.Time
	Action          *ActionRecord
	Observation     *Observation
	Trial           *Trial
	Decision        *Decision
	Ledger          *EconomicLedger
}

// HeadSnapshot combines the verified current head and its immutable record.
type HeadSnapshot struct {
	Head         StateHead
	HeadDigest   string
	Document     Document
	RecordDigest string
}

func canonical(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return contractcrypto.CanonicalizeJCS(raw)
}

func digestBytes(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// CanonicalDocument returns the exact RFC 8785 bytes and their bare SHA-256.
func CanonicalDocument(value any) ([]byte, string, error) {
	raw, err := canonical(value)
	if err != nil {
		return nil, "", err
	}
	return raw, digestBytes(raw), nil
}

func equalCanonical(raw []byte) ([]byte, error) {
	canonicalRaw, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonicalRaw) {
		return nil, fmt.Errorf("%w: document is not canonical JCS", ErrInvalidDocument)
	}
	return canonicalRaw, nil
}
