// Package sharedcache owns the private-beta single-node Shared storage.
//
// WCNCP-001 extends the existing typed-state seam with the
// Wrapper-Coordinated Native Corrections control plane. The five WCNCP record
// kinds share physical CAS bytes with the Gradle data plane and the existing
// typed state, but use independent wcncp_* metadata tables so no Gradle cache
// key, existing state kind, or WCNCP kind can address another namespace.
package sharedcache

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	// WCNCPKindObservation is one completed wrapper invocation and its facts.
	WCNCPKindObservation StateKind = "WCNCP_OBSERVATION"
	// WCNCPKindOpportunity is detector output derived from observations.
	WCNCPKindOpportunity StateKind = "WCNCP_OPPORTUNITY"
	// WCNCPKindProposal is an exact patch recipe with preimage and inverse.
	WCNCPKindProposal StateKind = "WCNCP_PROPOSAL"
	// WCNCPKindValidation is correctness, paired value, and cost rows.
	WCNCPKindValidation StateKind = "WCNCP_VALIDATION"
	// WCNCPKindDecision is an owner accept/reject/defer over a proposal.
	WCNCPKindDecision StateKind = "WCNCP_DECISION"

	maximumWCNCPManifestBytes = 1 << 20
	maximumWCNCPArtifactBytes = 16 << 20
	maximumWCNCPGeneration    = 9007199254740991

	wcncpStagedTTL      = 24 * time.Hour
	wcncpObservationTTL = 30 * 24 * time.Hour
	wcncpOpportunityTTL = 30 * 24 * time.Hour
	wcncpProposalTTL    = 90 * 24 * time.Hour
	wcncpValidationTTL  = 90 * 24 * time.Hour
)

var (
	// ErrWCNCPNotFound means no visible WCNCP state owns the identity.
	ErrWCNCPNotFound = errors.New("BuildOpt WCNCP state was not found")
	// ErrWCNCPInvalid means a WCNCP request violates the frozen contract.
	ErrWCNCPInvalid = errors.New("BuildOpt WCNCP state is invalid")
	// ErrWCNCPDigestMismatch means uploaded bytes do not match their address.
	ErrWCNCPDigestMismatch = errors.New("BuildOpt WCNCP digest mismatch")
	// ErrWCNCPManifestIncomplete means a manifest references unavailable state.
	ErrWCNCPManifestIncomplete = errors.New("BuildOpt WCNCP manifest is incomplete")
	// ErrWCNCPGenerationConflict means a manifest skips or reuses a generation.
	ErrWCNCPGenerationConflict = errors.New("BuildOpt WCNCP generation conflicts")
	// ErrWCNCPHeadPrecondition means another writer owns the expected head.
	ErrWCNCPHeadPrecondition = errors.New("BuildOpt WCNCP head precondition failed")
	// ErrWCNCPIdempotency means one key was reused for a changed request.
	ErrWCNCPIdempotency = errors.New("BuildOpt WCNCP idempotency conflict")
	// ErrWCNPCorrupt means durable metadata or bytes failed revalidation.
	ErrWCNPCorrupt = errors.New("BuildOpt WCNCP state is corrupt")
)

var (
	wcncpScopePattern = regexp.MustCompile(`^.+$`)
)

// WCNCP record schema versions frozen by WCNCP-000.
const (
	WCNCPObservationSchemaVersion = "buildopt.wcncp/observation/v1"
	WCNCPOpportunitySchemaVersion = "buildopt.wcncp/opportunity/v1"
	WCNCPProposalSchemaVersion    = "buildopt.wcncp/proposal/v1"
	WCNCPValidationSchemaVersion  = "buildopt.wcncp/validation/v1"
	WCNCPDecisionSchemaVersion    = "buildopt.wcncp/decision/v1"
	WCNCPManifestSchemaVersion    = "buildopt.wcncp/manifest/v1"
	WCNCPHeadSchemaVersion        = "buildopt.wcncp/head/v1"
)

// WCNCPObservation is the Go binding for WCNCP_OBSERVATION v1. Field names
// match the frozen JSON schema so canonical JCS bytes validate unchanged.
type WCNCPObservation struct {
	SchemaVersion     string `json:"schemaVersion"`
	RecordType        string `json:"recordType"`
	ObservationID     string `json:"observationId"`
	RepositoryScope   string `json:"repositoryScope"`
	RunnerID          string `json:"runnerId"`
	IdempotencyKey    string `json:"idempotencyKey"`
	InvocationOrdinal int64  `json:"invocationOrdinal"`
	EnvironmentClass  string `json:"environmentClass"`
	Bindings          struct {
		RepositoryRevision    string `json:"repositoryRevision"`
		SourceTreeSHA256      string `json:"sourceTreeSha256"`
		WrapperSHA256         string `json:"wrapperSha256"`
		GradleVersion         string `json:"gradleVersion"`
		JDKSHA256             string `json:"jdkSha256"`
		BuildOptPackageSHA256 string `json:"buildoptPackageSha256"`
		WorkflowSHA256        string `json:"workflowSha256"`
		EnvironmentSHA256     string `json:"environmentSha256"`
		OutputContractSHA256  string `json:"outputContractSha256"`
	} `json:"bindings"`
	Arguments []string `json:"arguments"`
	Duration  struct {
		State          string  `json:"state"`
		ValueMs        *int64  `json:"valueMs,omitempty"`
		Classification string  `json:"classification"`
		Reason         *string `json:"reason,omitempty"`
	} `json:"duration"`
	ConfigurationCache string `json:"configurationCache"`
	BuildCacheMode     string `json:"buildCacheMode"`
	OutputManifest     struct {
		State  string  `json:"state"`
		SHA256 *string `json:"sha256,omitempty"`
		Reason *string `json:"reason,omitempty"`
	} `json:"outputManifest"`
	Child struct {
		Outcome  string `json:"outcome"`
		ExitCode *int   `json:"exitCode,omitempty"`
		Signal   *int   `json:"signal,omitempty"`
	} `json:"child"`
	Completeness string `json:"completeness"`
	Authority    struct {
		ProspectiveGateInput bool `json:"prospectiveGateInput"`
		ProductionAuthorized bool `json:"productionAuthorized"`
	} `json:"authority"`
}

// WCNCPOpportunity is the Go binding for WCNCP_OPPORTUNITY v1.
type WCNCPOpportunity struct {
	SchemaVersion   string `json:"schemaVersion"`
	RecordType      string `json:"recordType"`
	OpportunityID   string `json:"opportunityId"`
	RepositoryScope string `json:"repositoryScope"`
	Detector        struct {
		ID                   string `json:"id"`
		Version              string `json:"version"`
		ImplementationSHA256 string `json:"implementationSha256"`
	} `json:"detector"`
	ObservationIDs []string `json:"observationIds"`
	SourceBinding  struct {
		State        string  `json:"state"`
		Path         *string `json:"path,omitempty"`
		SourceSHA256 *string `json:"sourceSha256,omitempty"`
		StartByte    *int64  `json:"startByte,omitempty"`
		EndByte      *int64  `json:"endByte,omitempty"`
		Reason       *string `json:"reason,omitempty"`
	} `json:"sourceBinding"`
	Materiality struct {
		State                string  `json:"state"`
		EnvironmentClass     string  `json:"environmentClass"`
		CriticalPathMs       *int64  `json:"criticalPathMs,omitempty"`
		WorkflowPercentMilli *int64  `json:"workflowPercentMilli,omitempty"`
		Reason               *string `json:"reason,omitempty"`
	} `json:"materiality"`
	Decision            string   `json:"decision"`
	RequiredDiagnostics []string `json:"requiredDiagnostics"`
	Authority           struct {
		CandidateBuildAuthorized bool `json:"candidateBuildAuthorized"`
		TimingAuthorized         bool `json:"timingAuthorized"`
		ProductionAuthorized     bool `json:"productionAuthorized"`
	} `json:"authority"`
}

// WCNCPProposal is the Go binding for WCNCP_PROPOSAL v1.
type WCNCPProposal struct {
	SchemaVersion     string `json:"schemaVersion"`
	RecordType        string `json:"recordType"`
	ProposalID        string `json:"proposalId"`
	RepositoryScope   string `json:"repositoryScope"`
	OpportunitySHA256 string `json:"opportunitySha256"`
	RecipeID          string `json:"recipeId"`
	Source            struct {
		Path            string `json:"path"`
		StartByte       int64  `json:"startByte"`
		EndByte         int64  `json:"endByte"`
		PreimageSHA256  string `json:"preimageSha256"`
		PostimageSHA256 string `json:"postimageSha256"`
	} `json:"source"`
	PatchBundleSHA256        string `json:"patchBundleSha256"`
	InversePatchBundleSHA256 string `json:"inversePatchBundleSha256"`
	Rationale                string `json:"rationale"`
	VerificationProtocol     string `json:"verificationProtocol"`
	Authority                struct {
		AutomaticApplyAuthorized       bool `json:"automaticApplyAuthorized"`
		AutomaticCommitAuthorized      bool `json:"automaticCommitAuthorized"`
		AutomaticPushAuthorized        bool `json:"automaticPushAuthorized"`
		AutomaticPullRequestAuthorized bool `json:"automaticPullRequestAuthorized"`
		AutomaticMergeAuthorized       bool `json:"automaticMergeAuthorized"`
		ProductionAuthorized           bool `json:"productionAuthorized"`
	} `json:"authority"`
}

// WCNCPValidation is the Go binding for WCNCP_VALIDATION v1.
type WCNCPValidation struct {
	SchemaVersion    string `json:"schemaVersion"`
	RecordType       string `json:"recordType"`
	ValidationID     string `json:"validationId"`
	RepositoryScope  string `json:"repositoryScope"`
	ProposalSHA256   string `json:"proposalSha256"`
	LeaseSHA256      string `json:"leaseSha256"`
	EnvironmentClass string `json:"environmentClass"`
	BindingsSHA256   string `json:"bindingsSha256"`
	Correctness      struct {
		Starts          int64 `json:"starts"`
		ExactOutputs    bool  `json:"exactOutputs"`
		Invalidation    bool  `json:"invalidation"`
		ExactRevert     bool  `json:"exactRevert"`
		ProductFailures int64 `json:"productFailures"`
	} `json:"correctness"`
	Timing struct {
		State                  string  `json:"state"`
		Pairs                  *int64  `json:"pairs,omitempty"`
		PositivePairs          *int64  `json:"positivePairs,omitempty"`
		MeanSavingMs           *int64  `json:"meanSavingMs,omitempty"`
		MeanSavingPercentMilli *int64  `json:"meanSavingPercentMilli,omitempty"`
		CandidateP95DeltaMs    *int64  `json:"candidateP95DeltaMs,omitempty"`
		Paired95LowerMs        *int64  `json:"paired95LowerMs,omitempty"`
		Paired95UpperMs        *int64  `json:"paired95UpperMs,omitempty"`
		Reason                 *string `json:"reason,omitempty"`
	} `json:"timing"`
	Cost struct {
		MachineMs     int64  `json:"machineMs"`
		OperationalMs int64  `json:"operationalMs"`
		PaybackBuilds *int64 `json:"paybackBuilds,omitempty"`
	} `json:"cost"`
	Decision  string `json:"decision"`
	Authority struct {
		SourceApplyAuthorized bool `json:"sourceApplyAuthorized"`
		OwnerDecisionRequired bool `json:"ownerDecisionRequired"`
		ProductionAuthorized  bool `json:"productionAuthorized"`
	} `json:"authority"`
}

// WCNCPDecision is the Go binding for WCNCP_DECISION v1.
type WCNCPDecision struct {
	SchemaVersion      string   `json:"schemaVersion"`
	RecordType         string   `json:"recordType"`
	DecisionID         string   `json:"decisionId"`
	RepositoryScope    string   `json:"repositoryScope"`
	ProposalSHA256     string   `json:"proposalSha256"`
	ValidationSHA256   string   `json:"validationSha256"`
	PrincipalSHA256    string   `json:"principalSha256"`
	Decision           string   `json:"decision"`
	ActiveReviewMs     int64    `json:"activeReviewMs"`
	ClarificationCount int64    `json:"clarificationCount"`
	Concerns           []string `json:"concerns,omitempty"`
	DecidedAt          string   `json:"decidedAt"`
	Authority          struct {
		OwnerAuthenticated   bool `json:"ownerAuthenticated"`
		SourceApplied        bool `json:"sourceApplied"`
		CommitCreated        bool `json:"commitCreated"`
		PushCreated          bool `json:"pushCreated"`
		PullRequestCreated   bool `json:"pullRequestCreated"`
		Merged               bool `json:"merged"`
		ProductionAuthorized bool `json:"productionAuthorized"`
	} `json:"authority"`
}

// WCNCPManifest is the immutable publication envelope for one WCNCP record.
type WCNCPManifest struct {
	SchemaVersion         string           `json:"schemaVersion"`
	RecordType            string           `json:"recordType"`
	Kind                  StateKind        `json:"kind"`
	RepositoryScopeSHA256 string           `json:"repositoryScopeSha256"`
	Generation            int64            `json:"generation"`
	CompatibilitySHA256   string           `json:"compatibilitySha256"`
	BindingsSHA256        string           `json:"bindingsSha256"`
	Origin                StateOrigin      `json:"origin"`
	Artifacts             []StateArtifact  `json:"artifacts"`
	References            []WCNCPReference `json:"references"`
	Status                string           `json:"status"`
	RetentionClass        string           `json:"retentionClass"`
	CreatedAt             string           `json:"createdAt"`
	ExpiresAt             string           `json:"expiresAt,omitempty"`
	Authority             StateAuthority   `json:"authority"`
}

// WCNCPReference links one manifest to a parent WCNCP manifest.
type WCNCPReference struct {
	Kind           StateKind `json:"kind"`
	ManifestSHA256 string    `json:"manifestSha256"`
	Relation       string    `json:"relation"`
}

// WCNCPHead is the sole mutable pointer for one repository and WCNCP kind.
type WCNCPHead struct {
	SchemaVersion          string         `json:"schemaVersion"`
	RecordType             string         `json:"recordType"`
	Kind                   StateKind      `json:"kind"`
	RepositoryScopeSHA256  string         `json:"repositoryScopeSha256"`
	Generation             int64          `json:"generation"`
	ManifestSHA256         string         `json:"manifestSha256"`
	PreviousManifestSHA256 string         `json:"previousManifestSha256,omitempty"`
	CompatibilitySHA256    string         `json:"compatibilitySha256"`
	UpdatedAt              string         `json:"updatedAt"`
	Authority              StateAuthority `json:"authority"`
}

// WCNCPObject identifies a namespaced CAS object after verification.
type WCNCPObject struct {
	RepositoryScopeSHA256 string
	Kind                  StateKind
	SHA256                string
	SizeBytes             int64
}

// WCNCPObjectInput is one immutable record staged for an atomic metadata
// publication. Physical CAS bytes may be written before the transaction, but
// no repository/kind visibility is granted unless every input validates and
// the complete metadata transaction commits.
type WCNCPObjectInput struct {
	ExpectedSHA256 string
	Raw            []byte
}

// WCNCP CAS request/response mirror the generic typed-state protocol.
type WCNCPCASRequest struct {
	RepositoryScopeSHA256 string
	Kind                  StateKind
	IdempotencyKey        string
	ExpectedGeneration    int64
	ExpectedHeadSHA256    string
	ManifestSHA256        string
	ProposedHead          *WCNCPHead
}

type WCNCPCASResult struct {
	Head       WCNCPHead
	HeadSHA256 string
	Replayed   bool
}

type WCNCPSnapshot struct {
	Head           WCNCPHead
	HeadSHA256     string
	Manifest       WCNCPManifest
	ManifestSHA256 string
}

type WCNCPMaintenanceReport struct {
	ExpiredStagedManifests  int
	ExpiredStagedObjects    int
	ExpiredSuperseded       int
	DeletedUnreferencedBlob int
}

func validWCNCPKind(kind StateKind) bool {
	return kind == WCNCPKindObservation || kind == WCNCPKindOpportunity ||
		kind == WCNCPKindProposal || kind == WCNCPKindValidation ||
		kind == WCNCPKindDecision
}

func wcncpRetentionClass(kind StateKind) string {
	switch kind {
	case WCNCPKindObservation:
		return "RAW_30_DAYS"
	case WCNCPKindOpportunity:
		return "CURRENT_PLUS_30_DAYS_AFTER_SUPERSEDED"
	case WCNCPKindProposal:
		return "WHILE_ACTIVE_PLUS_90_DAYS"
	case WCNCPKindValidation:
		return "WHILE_PROPOSAL_RETAINED_PLUS_90_DAYS"
	case WCNCPKindDecision:
		return "DURABLE"
	default:
		return ""
	}
}

func wcncpArtifactRole(kind StateKind) string {
	return string(kind)
}

func wcncpPayloadSchemaVersion(kind StateKind) string {
	switch kind {
	case WCNCPKindObservation:
		return WCNCPObservationSchemaVersion
	case WCNCPKindOpportunity:
		return WCNCPOpportunitySchemaVersion
	case WCNCPKindProposal:
		return WCNCPProposalSchemaVersion
	case WCNCPKindValidation:
		return WCNCPValidationSchemaVersion
	case WCNCPKindDecision:
		return WCNCPDecisionSchemaVersion
	default:
		return ""
	}
}

// ValidateWCNCPRecord validates one canonical WCNCP JSON document against the
// frozen WCNCP-000 contract. It mirrors the JSON Schema rules so Go rejects
// exactly what the schema rejects, including environment-gated fields and
// fail-closed authority.
func ValidateWCNCPRecord(kind StateKind, raw []byte) error {
	if !validWCNCPKind(kind) {
		return ErrWCNCPInvalid
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		return ErrWCNCPInvalid
	}
	// Strict unknown-field rejection matches additionalProperties:false.
	switch kind {
	case WCNCPKindObservation:
		var record WCNCPObservation
		if err := decodeStrictWCNCPJSON(canonical, &record); err != nil {
			return ErrWCNCPInvalid
		}
		return validateWCNCPObservation(record)
	case WCNCPKindOpportunity:
		var record WCNCPOpportunity
		if err := decodeStrictWCNCPJSON(canonical, &record); err != nil {
			return ErrWCNCPInvalid
		}
		return validateWCNCPOpportunity(record)
	case WCNCPKindProposal:
		var record WCNCPProposal
		if err := decodeStrictWCNCPJSON(canonical, &record); err != nil {
			return ErrWCNCPInvalid
		}
		return validateWCNCPProposal(record)
	case WCNCPKindValidation:
		var record WCNCPValidation
		if err := decodeStrictWCNCPJSON(canonical, &record); err != nil {
			return ErrWCNCPInvalid
		}
		return validateWCNCPValidation(record)
	case WCNCPKindDecision:
		var record WCNCPDecision
		if err := decodeStrictWCNCPJSON(canonical, &record); err != nil {
			return ErrWCNCPInvalid
		}
		return validateWCNCPDecision(record)
	default:
		return ErrWCNCPInvalid
	}
}

func decodeStrictWCNCPJSON(raw []byte, target any) error {
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

func validateWCNCPObservation(record WCNCPObservation) error {
	if record.SchemaVersion != WCNCPObservationSchemaVersion ||
		record.RecordType != "WCNCP_OBSERVATION" ||
		!validSHA256(record.ObservationID) ||
		len(record.RepositoryScope) < 1 || len(record.RepositoryScope) > 256 ||
		!validSHA256(record.RunnerID) ||
		len(record.IdempotencyKey) < 16 || len(record.IdempotencyKey) > 128 ||
		record.InvocationOrdinal < 1 || record.InvocationOrdinal > maximumWCNCPGeneration {
		return ErrWCNCPInvalid
	}
	for _, c := range record.IdempotencyKey {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return ErrWCNCPInvalid
		}
	}
	if record.EnvironmentClass != "CONTROLLED_PERFORMANCE" && record.EnvironmentClass != "STANDARD_HOSTED_CI" && record.EnvironmentClass != "LOCAL_FUNCTIONAL" {
		return ErrWCNCPInvalid
	}
	if !stateRevisionPattern.MatchString(record.Bindings.RepositoryRevision) ||
		!validSHA256(record.Bindings.SourceTreeSHA256) ||
		!validSHA256(record.Bindings.WrapperSHA256) ||
		len(record.Bindings.GradleVersion) < 1 || len(record.Bindings.GradleVersion) > 32 ||
		!validSHA256(record.Bindings.JDKSHA256) ||
		!validSHA256(record.Bindings.BuildOptPackageSHA256) ||
		!validSHA256(record.Bindings.WorkflowSHA256) ||
		!validSHA256(record.Bindings.EnvironmentSHA256) ||
		!validSHA256(record.Bindings.OutputContractSHA256) {
		return ErrWCNCPInvalid
	}
	if len(record.Arguments) > 128 {
		return ErrWCNCPInvalid
	}
	for _, arg := range record.Arguments {
		if len(arg) < 1 || len(arg) > 256 {
			return ErrWCNCPInvalid
		}
	}
	// Typed duration: COMPLETE requires valueMs without reason; UNAVAILABLE
	// requires reason without valueMs. Classification is environment-gated.
	if record.Duration.State != "COMPLETE" && record.Duration.State != "UNAVAILABLE" {
		return ErrWCNCPInvalid
	}
	if record.Duration.State == "COMPLETE" {
		if record.Duration.ValueMs == nil || *record.Duration.ValueMs < 0 || record.Duration.Reason != nil {
			return ErrWCNCPInvalid
		}
	} else {
		if record.Duration.Reason == nil || len(*record.Duration.Reason) < 1 || record.Duration.ValueMs != nil {
			return ErrWCNCPInvalid
		}
	}
	switch record.EnvironmentClass {
	case "CONTROLLED_PERFORMANCE":
		// Any classification is permitted only when consistent with state?
		// Frozen vectors use CONTROLLED_VALUE_INPUT for controlled rows.
		if record.Duration.Classification != "CONTROLLED_VALUE_INPUT" && record.Duration.Classification != "DIAGNOSTIC_ONLY" && record.Duration.Classification != "NOT_EVALUATED" {
			return ErrWCNCPInvalid
		}
	case "STANDARD_HOSTED_CI":
		if record.Duration.Classification != "DIAGNOSTIC_ONLY" {
			return ErrWCNCPInvalid
		}
	case "LOCAL_FUNCTIONAL":
		if record.Duration.Classification != "NOT_EVALUATED" {
			return ErrWCNCPInvalid
		}
	}
	if record.ConfigurationCache != "NOT_REQUESTED" && record.ConfigurationCache != "STORE" && record.ConfigurationCache != "REUSE" && record.ConfigurationCache != "PROBLEM" && record.ConfigurationCache != "UNAVAILABLE" {
		return ErrWCNCPInvalid
	}
	if record.BuildCacheMode != "ENABLED" && record.BuildCacheMode != "DISABLED" && record.BuildCacheMode != "UNAVAILABLE" {
		return ErrWCNCPInvalid
	}
	if record.OutputManifest.State != "COMPLETE" && record.OutputManifest.State != "UNAVAILABLE" {
		return ErrWCNCPInvalid
	}
	if record.OutputManifest.State == "COMPLETE" {
		if record.OutputManifest.SHA256 == nil || !validSHA256(*record.OutputManifest.SHA256) || record.OutputManifest.Reason != nil {
			return ErrWCNCPInvalid
		}
	} else {
		if record.OutputManifest.Reason == nil || len(*record.OutputManifest.Reason) < 1 || record.OutputManifest.SHA256 != nil {
			return ErrWCNCPInvalid
		}
	}
	if record.Child.Outcome != "SUCCESS" && record.Child.Outcome != "FAILED" && record.Child.Outcome != "SIGNALED" && record.Child.Outcome != "NOT_STARTED" {
		return ErrWCNCPInvalid
	}
	if record.Completeness != "COMPLETE" && record.Completeness != "INCOMPLETE" {
		return ErrWCNCPInvalid
	}
	if record.Authority.ProductionAuthorized {
		return ErrWCNCPInvalid
	}
	return nil
}

func validateWCNCPOpportunity(record WCNCPOpportunity) error {
	if record.SchemaVersion != WCNCPOpportunitySchemaVersion ||
		record.RecordType != "WCNCP_OPPORTUNITY" ||
		!validSHA256(record.OpportunityID) ||
		len(record.RepositoryScope) < 1 || len(record.RepositoryScope) > 256 {
		return ErrWCNCPInvalid
	}
	if record.Detector.ID != "CONFIGURATION_CACHE_READINESS_PATCH" && record.Detector.ID != "NORMALIZATION_AWARE_CACHEABILITY" && record.Detector.ID != "DURABLE_TASK_CONTRACT" {
		return ErrWCNCPInvalid
	}
	if len(record.Detector.Version) < 1 || len(record.Detector.Version) > 32 || !validSHA256(record.Detector.ImplementationSHA256) {
		return ErrWCNCPInvalid
	}
	if len(record.ObservationIDs) < 1 || len(record.ObservationIDs) > 32 {
		return ErrWCNCPInvalid
	}
	seen := map[string]bool{}
	for _, id := range record.ObservationIDs {
		if !validSHA256(id) || seen[id] {
			return ErrWCNCPInvalid
		}
		seen[id] = true
	}
	if record.SourceBinding.State != "COMPLETE" && record.SourceBinding.State != "AMBIGUOUS" && record.SourceBinding.State != "UNAVAILABLE" && record.SourceBinding.State != "DRIFTED" {
		return ErrWCNCPInvalid
	}
	if record.SourceBinding.State == "COMPLETE" {
		if record.SourceBinding.Path == nil || len(*record.SourceBinding.Path) < 1 || record.SourceBinding.SourceSHA256 == nil || !validSHA256(*record.SourceBinding.SourceSHA256) || record.SourceBinding.StartByte == nil || record.SourceBinding.EndByte == nil || *record.SourceBinding.EndByte < 1 || *record.SourceBinding.StartByte < 0 {
			return ErrWCNCPInvalid
		}
	}
	if record.Materiality.State != "PASSED" && record.Materiality.State != "FAILED" && record.Materiality.State != "REQUIRES_CONTROLLED_MEASUREMENT" {
		return ErrWCNCPInvalid
	}
	if record.Materiality.EnvironmentClass != "CONTROLLED_PERFORMANCE" && record.Materiality.EnvironmentClass != "STANDARD_HOSTED_CI" && record.Materiality.EnvironmentClass != "LOCAL_FUNCTIONAL" {
		return ErrWCNCPInvalid
	}
	// Hosted and local rows can never admit or reject on timing.
	if (record.Materiality.EnvironmentClass == "STANDARD_HOSTED_CI" || record.Materiality.EnvironmentClass == "LOCAL_FUNCTIONAL") && record.Materiality.State != "REQUIRES_CONTROLLED_MEASUREMENT" {
		return ErrWCNCPInvalid
	}
	validDecisions := map[string]bool{
		"ACTIONABLE_MATERIAL_CORRECTION": true, "NO_REPRODUCIBLE_BLOCKER": true, "NON_MATERIAL_BLOCKER": true, "OWNER_SEMANTICS_REQUIRED": true, "UNSAFE_OR_NON_REVERSIBLE": true, "UNSUPPORTED_PROBLEM_CLASS": true, "INCOMPLETE_OBSERVATION": true, "SOURCE_OR_BINDING_AMBIGUOUS": true, "SOURCE_DRIFTED": true,
	}
	if !validDecisions[record.Decision] {
		return ErrWCNCPInvalid
	}
	if len(record.RequiredDiagnostics) > 8 {
		return ErrWCNCPInvalid
	}
	if record.Authority.CandidateBuildAuthorized || record.Authority.TimingAuthorized || record.Authority.ProductionAuthorized {
		return ErrWCNCPInvalid
	}
	return nil
}

func validateWCNCPProposal(record WCNCPProposal) error {
	if record.SchemaVersion != WCNCPProposalSchemaVersion ||
		record.RecordType != "WCNCP_PROPOSAL" ||
		!validSHA256(record.ProposalID) ||
		len(record.RepositoryScope) < 1 || len(record.RepositoryScope) > 256 ||
		!validSHA256(record.OpportunitySHA256) {
		return ErrWCNCPInvalid
	}
	matched, _ := regexp.MatchString(`^[A-Z0-9][A-Z0-9._-]{2,127}$`, record.RecipeID)
	if !matched {
		return ErrWCNCPInvalid
	}
	if len(record.Source.Path) < 1 || len(record.Source.Path) > 1024 || record.Source.StartByte < 0 || record.Source.EndByte < 1 || !validSHA256(record.Source.PreimageSHA256) || !validSHA256(record.Source.PostimageSHA256) {
		return ErrWCNCPInvalid
	}
	if !validSHA256(record.PatchBundleSHA256) || !validSHA256(record.InversePatchBundleSHA256) {
		return ErrWCNCPInvalid
	}
	if len(record.Rationale) < 1 || len(record.Rationale) > 4096 || len(record.VerificationProtocol) < 1 || len(record.VerificationProtocol) > 128 {
		return ErrWCNCPInvalid
	}
	if record.Authority.AutomaticApplyAuthorized || record.Authority.AutomaticCommitAuthorized || record.Authority.AutomaticPushAuthorized || record.Authority.AutomaticPullRequestAuthorized || record.Authority.AutomaticMergeAuthorized || record.Authority.ProductionAuthorized {
		return ErrWCNCPInvalid
	}
	return nil
}

func validateWCNCPValidation(record WCNCPValidation) error {
	if record.SchemaVersion != WCNCPValidationSchemaVersion ||
		record.RecordType != "WCNCP_VALIDATION" ||
		!validSHA256(record.ValidationID) ||
		len(record.RepositoryScope) < 1 || len(record.RepositoryScope) > 256 ||
		!validSHA256(record.ProposalSHA256) ||
		!validSHA256(record.LeaseSHA256) ||
		!validSHA256(record.BindingsSHA256) {
		return ErrWCNCPInvalid
	}
	if record.EnvironmentClass != "CONTROLLED_PERFORMANCE" && record.EnvironmentClass != "STANDARD_HOSTED_CI" && record.EnvironmentClass != "LOCAL_FUNCTIONAL" {
		return ErrWCNCPInvalid
	}
	if record.Correctness.Starts < 0 || record.Correctness.ProductFailures < 0 {
		return ErrWCNCPInvalid
	}
	if record.Timing.State != "CONTROLLED_COMPLETE" && record.Timing.State != "NOT_EVALUATED_STANDARD_CI" && record.Timing.State != "NOT_RUN" {
		return ErrWCNCPInvalid
	}
	if record.Timing.State == "CONTROLLED_COMPLETE" {
		if record.Timing.Pairs == nil || record.Timing.PositivePairs == nil || record.Timing.MeanSavingMs == nil || record.Timing.MeanSavingPercentMilli == nil || record.Timing.CandidateP95DeltaMs == nil || record.Timing.Paired95LowerMs == nil || record.Timing.Paired95UpperMs == nil {
			return ErrWCNCPInvalid
		}
		if *record.Timing.Pairs < 0 || *record.Timing.Pairs > 8 || *record.Timing.PositivePairs < 0 || *record.Timing.PositivePairs > 8 {
			return ErrWCNCPInvalid
		}
	} else {
		if record.Timing.Reason == nil || len(*record.Timing.Reason) < 1 {
			return ErrWCNCPInvalid
		}
	}
	if record.Cost.MachineMs < 0 || record.Cost.OperationalMs < 0 {
		return ErrWCNCPInvalid
	}
	if record.Cost.PaybackBuilds != nil && *record.Cost.PaybackBuilds < 1 {
		return ErrWCNCPInvalid
	}
	validDecisions := map[string]bool{"QUALIFIED": true, "REJECTED_CORRECTNESS": true, "REJECTED_VALUE": true, "INCOMPLETE": true, "NOT_EVALUATED_STANDARD_CI": true}
	if !validDecisions[record.Decision] {
		return ErrWCNCPInvalid
	}
	// Hosted rows never carry controlled timing or a value decision.
	if record.EnvironmentClass == "STANDARD_HOSTED_CI" && (record.Timing.State != "NOT_EVALUATED_STANDARD_CI" || record.Decision != "NOT_EVALUATED_STANDARD_CI") {
		return ErrWCNCPInvalid
	}
	if record.Authority.SourceApplyAuthorized || !record.Authority.OwnerDecisionRequired || record.Authority.ProductionAuthorized {
		return ErrWCNCPInvalid
	}
	return nil
}

func validateWCNCPDecision(record WCNCPDecision) error {
	if record.SchemaVersion != WCNCPDecisionSchemaVersion ||
		record.RecordType != "WCNCP_DECISION" ||
		!validSHA256(record.DecisionID) ||
		len(record.RepositoryScope) < 1 || len(record.RepositoryScope) > 256 ||
		!validSHA256(record.ProposalSHA256) ||
		!validSHA256(record.ValidationSHA256) ||
		!validSHA256(record.PrincipalSHA256) {
		return ErrWCNCPInvalid
	}
	if record.Decision != "ACCEPT" && record.Decision != "REJECT" && record.Decision != "DEFER" {
		return ErrWCNCPInvalid
	}
	if record.ActiveReviewMs < 0 || record.ActiveReviewMs > 86400000 || record.ClarificationCount < 0 || record.ClarificationCount > 100 {
		return ErrWCNCPInvalid
	}
	if len(record.Concerns) > 32 {
		return ErrWCNCPInvalid
	}
	if !contractcrypto.ValidUTCTimestamp(record.DecidedAt) {
		return ErrWCNCPInvalid
	}
	// Owner acceptance never implies source application, commit, push, PR,
	// merge, or production authority.
	if !record.Authority.OwnerAuthenticated || record.Authority.SourceApplied || record.Authority.CommitCreated || record.Authority.PushCreated || record.Authority.PullRequestCreated || record.Authority.Merged || record.Authority.ProductionAuthorized {
		return ErrWCNCPInvalid
	}
	return nil
}

// CanonicalWCNCPRecord renders one validated record with the same JCS
// identity used for blob addressing and independent reconstruction.
func CanonicalWCNCPRecord(kind StateKind, raw []byte) ([]byte, string, error) {
	if err := ValidateWCNCPRecord(kind, raw); err != nil {
		return nil, "", err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		return nil, "", ErrWCNCPInvalid
	}
	return canonical, digestBytes(canonical), nil
}

// PutWCNCPObject verifies expectedSHA256 before granting visibility in
// exactly one repository/kind namespace. Physical blob bytes may
// deduplicate with Gradle cache objects; metadata never shares.
func (storage *Storage) PutWCNCPObject(ctx context.Context, repositoryScopeSHA256 string, kind StateKind, expectedSHA256 string, reader io.Reader) (WCNCPObject, bool, error) {
	if ctx == nil || reader == nil || !validSHA256(repositoryScopeSHA256) || !validWCNCPKind(kind) || !validSHA256(expectedSHA256) {
		return WCNCPObject{}, false, ErrWCNCPInvalid
	}
	raw, err := io.ReadAll(io.LimitReader(reader, maximumWCNCPArtifactBytes+1))
	if err != nil || len(raw) > maximumWCNCPArtifactBytes {
		return WCNCPObject{}, false, ErrWCNCPInvalid
	}
	objects, inserted, err := storage.PutWCNCPObjectBatch(ctx, repositoryScopeSHA256, kind, []WCNCPObjectInput{{ExpectedSHA256: expectedSHA256, Raw: raw}})
	if err != nil {
		return WCNCPObject{}, false, err
	}
	return objects[0], inserted[0], nil
}

// PutWCNCPObjectBatch validates every record and publishes all metadata in one
// transaction. A corrupt or unavailable item leaves no partial logical batch.
func (storage *Storage) PutWCNCPObjectBatch(ctx context.Context, repositoryScopeSHA256 string, kind StateKind, inputs []WCNCPObjectInput) ([]WCNCPObject, []bool, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) || !validWCNCPKind(kind) || len(inputs) == 0 || len(inputs) > wcncpBatchMaxItems {
		return nil, nil, ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return nil, nil, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	objects := make([]WCNCPObject, 0, len(inputs))
	for _, input := range inputs {
		if !validSHA256(input.ExpectedSHA256) || len(input.Raw) == 0 || len(input.Raw) > maximumWCNCPArtifactBytes {
			return nil, nil, ErrWCNCPInvalid
		}
		if err := ValidateWCNCPRecord(kind, input.Raw); err != nil {
			return nil, nil, err
		}
		blob, _, err := storage.blobs.putLocked(ctx, bytes.NewReader(input.Raw))
		if err != nil {
			return nil, nil, err
		}
		if blob.Size < 1 || blob.Size > maximumWCNCPArtifactBytes || strings.TrimPrefix(blob.Digest, digestPrefix) != input.ExpectedSHA256 {
			return nil, nil, ErrWCNCPDigestMismatch
		}
		verified, err := storage.blobs.openVerified(ctx, blob)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrWCNPCorrupt, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(verified, maximumWCNCPArtifactBytes+1))
		closeErr := verified.Close()
		if readErr != nil || closeErr != nil || !bytes.Equal(content, input.Raw) {
			return nil, nil, ErrWCNPCorrupt
		}
		objects = append(objects, WCNCPObject{RepositoryScopeSHA256: repositoryScopeSHA256, Kind: kind, SHA256: input.ExpectedSHA256, SizeBytes: blob.Size})
	}
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	transaction, err := storage.state.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, nil, err
	}
	defer transaction.Rollback()
	inserted := make([]bool, len(objects))
	for index, object := range objects {
		result, err := transaction.ExecContext(ctx, `INSERT INTO wcncp_objects (
    repository_scope_sha256, kind, blob_digest, size_bytes, created_at_unix_ms
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(repository_scope_sha256, kind, blob_digest) DO NOTHING`,
			repositoryScopeSHA256, kind, digestPrefix+object.SHA256, object.SizeBytes, storage.now().UnixMilli())
		if err != nil {
			return nil, nil, err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return nil, nil, err
		}
		inserted[index] = rows == 1
	}
	if err := transaction.Commit(); err != nil {
		return nil, nil, err
	}
	return objects, inserted, nil
}

// OpenWCNCPObject returns verified bytes only when the exact repository and
// kind own metadata for the digest.
func (storage *Storage) OpenWCNCPObject(ctx context.Context, repositoryScopeSHA256 string, kind StateKind, digest string) (*os.File, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) || !validWCNCPKind(kind) || !validSHA256(digest) {
		return nil, ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return nil, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	var size int64
	err = storage.state.database.QueryRowContext(ctx, `SELECT size_bytes FROM wcncp_objects
WHERE repository_scope_sha256 = ? AND kind = ? AND blob_digest = ?`,
		repositoryScopeSHA256, kind, digestPrefix+digest).Scan(&size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWCNCPNotFound
	}
	if err != nil {
		return nil, err
	}
	file, err := storage.blobs.openVerified(ctx, Blob{Digest: digestPrefix + digest, Size: size})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWCNPCorrupt, err)
	}
	return file, nil
}

// PutWCNCPManifest validates a complete immutable generation before making it
// addressable. It does not change the current head, so partial publication
// stays invisible.
func (storage *Storage) PutWCNCPManifest(ctx context.Context, raw []byte) (WCNCPSnapshot, bool, error) {
	if ctx == nil || len(raw) == 0 || len(raw) > maximumWCNCPManifestBytes {
		return WCNCPSnapshot{}, false, ErrWCNCPInvalid
	}
	manifest, canonical, digest, err := decodeWCNCPManifest(raw)
	if err != nil {
		return WCNCPSnapshot{}, false, err
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPSnapshot{}, false, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	transaction, err := storage.state.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return WCNCPSnapshot{}, false, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	var existing []byte
	err = transaction.QueryRowContext(ctx, `SELECT canonical_document FROM wcncp_manifests
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
		manifest.RepositoryScopeSHA256, manifest.Kind, digestPrefix+digest).Scan(&existing)
	if err == nil {
		if !bytes.Equal(existing, canonical) {
			return WCNCPSnapshot{}, false, ErrWCNPCorrupt
		}
		if err := transaction.Commit(); err != nil {
			return WCNCPSnapshot{}, false, err
		}
		rollback = false
		return WCNCPSnapshot{Manifest: manifest, ManifestSHA256: digest}, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return WCNCPSnapshot{}, false, err
	}
	for _, artifact := range manifest.Artifacts {
		var size int64
		err := transaction.QueryRowContext(ctx, `SELECT size_bytes FROM wcncp_objects
WHERE repository_scope_sha256 = ? AND kind = ? AND blob_digest = ?`,
			manifest.RepositoryScopeSHA256, manifest.Kind, digestPrefix+artifact.SHA256).Scan(&size)
		if errors.Is(err, sql.ErrNoRows) || size != artifact.SizeBytes {
			return WCNCPSnapshot{}, false, ErrWCNCPManifestIncomplete
		}
		if err != nil {
			return WCNCPSnapshot{}, false, err
		}
	}
	for _, reference := range manifest.References {
		var present int
		err := transaction.QueryRowContext(ctx, `SELECT 1 FROM wcncp_manifests
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
			manifest.RepositoryScopeSHA256, reference.Kind, digestPrefix+reference.ManifestSHA256).Scan(&present)
		if errors.Is(err, sql.ErrNoRows) {
			return WCNCPSnapshot{}, false, ErrWCNCPManifestIncomplete
		}
		if err != nil {
			return WCNCPSnapshot{}, false, err
		}
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	var expiresAt any
	if manifest.ExpiresAt != "" {
		parsed, _ := time.Parse(time.RFC3339Nano, manifest.ExpiresAt)
		expiresAt = parsed.UnixMilli()
	}
	_, err = transaction.ExecContext(ctx, `INSERT INTO wcncp_manifests (
    repository_scope_sha256, kind, generation, manifest_digest,
    canonical_document, compatibility_sha256, bindings_sha256, status,
    created_at_unix_ms, expires_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		manifest.RepositoryScopeSHA256, manifest.Kind, manifest.Generation, digestPrefix+digest,
		canonical, manifest.CompatibilitySHA256, manifest.BindingsSHA256, manifest.Status,
		createdAt.UnixMilli(), expiresAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return WCNCPSnapshot{}, false, ErrWCNCPGenerationConflict
		}
		return WCNCPSnapshot{}, false, err
	}
	for _, artifact := range manifest.Artifacts {
		if _, err := transaction.ExecContext(ctx, `INSERT INTO wcncp_manifest_artifacts (
    repository_scope_sha256, kind, manifest_digest, role, blob_digest, size_bytes
) VALUES (?, ?, ?, ?, ?, ?)`,
			manifest.RepositoryScopeSHA256, manifest.Kind, digestPrefix+digest,
			artifact.Role, digestPrefix+artifact.SHA256, artifact.SizeBytes); err != nil {
			return WCNCPSnapshot{}, false, err
		}
	}
	for _, reference := range manifest.References {
		if _, err := transaction.ExecContext(ctx, `INSERT INTO wcncp_manifest_references (
    repository_scope_sha256, source_kind, source_manifest_digest,
    target_kind, target_manifest_digest, relation
) VALUES (?, ?, ?, ?, ?, ?)`,
			manifest.RepositoryScopeSHA256, manifest.Kind, digestPrefix+digest,
			reference.Kind, digestPrefix+reference.ManifestSHA256, reference.Relation); err != nil {
			return WCNCPSnapshot{}, false, err
		}
	}
	if err := transaction.Commit(); err != nil {
		return WCNCPSnapshot{}, false, err
	}
	rollback = false
	return WCNCPSnapshot{Manifest: manifest, ManifestSHA256: digest}, true, nil
}

// LoadWCNCPManifest returns one fully verified immutable manifest.
func (storage *Storage) LoadWCNCPManifest(ctx context.Context, repositoryScopeSHA256 string, kind StateKind, manifestSHA256 string) (WCNCPSnapshot, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) || !validWCNCPKind(kind) || !validSHA256(manifestSHA256) {
		return WCNCPSnapshot{}, ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPSnapshot{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	transaction, err := storage.state.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return WCNCPSnapshot{}, err
	}
	defer transaction.Rollback()
	manifest, err := loadWCNCPManifestTx(ctx, transaction, repositoryScopeSHA256, kind, manifestSHA256)
	if err != nil {
		return WCNCPSnapshot{}, err
	}
	if err := storage.verifyWCNCPManifestForPublication(ctx, transaction, manifest, manifestSHA256); err != nil {
		return WCNCPSnapshot{}, err
	}
	if err := transaction.Commit(); err != nil {
		return WCNCPSnapshot{}, err
	}
	return WCNCPSnapshot{Manifest: manifest, ManifestSHA256: manifestSHA256}, nil
}

// CASWCNCPHead publishes exactly the next complete generation. Idempotency
// and the head update commit in one SQLite transaction.
func (storage *Storage) CASWCNCPHead(ctx context.Context, request WCNCPCASRequest) (WCNCPCASResult, error) {
	if ctx == nil || !validSHA256(request.RepositoryScopeSHA256) || !validWCNCPKind(request.Kind) || !validSHA256(request.IdempotencyKey) || !validSHA256(request.ManifestSHA256) || request.ExpectedGeneration < 0 || request.ExpectedGeneration >= maximumWCNCPGeneration || (request.ExpectedGeneration == 0 && request.ExpectedHeadSHA256 != "") || (request.ExpectedGeneration > 0 && !validSHA256(request.ExpectedHeadSHA256)) {
		return WCNCPCASResult{}, ErrWCNCPInvalid
	}
	if request.ProposedHead != nil {
		canonical, _, err := canonicalStateValue(*request.ProposedHead)
		if err != nil {
			return WCNCPCASResult{}, ErrWCNCPInvalid
		}
		decoded, err := decodeWCNCPHead(canonical)
		if err != nil || decoded != *request.ProposedHead {
			return WCNCPCASResult{}, ErrWCNCPInvalid
		}
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPCASResult{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	fingerprint, err := wcncpCASFingerprint(request)
	if err != nil {
		return WCNCPCASResult{}, err
	}
	transaction, err := storage.state.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return WCNCPCASResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	var priorFingerprint, priorHeadDigest string
	var priorHead []byte
	err = transaction.QueryRowContext(ctx, `SELECT request_digest, head_digest, head_canonical_document
FROM wcncp_cas_requests WHERE idempotency_key = ?`, request.IdempotencyKey).Scan(&priorFingerprint, &priorHeadDigest, &priorHead)
	if err == nil {
		if priorFingerprint != digestPrefix+fingerprint {
			return WCNCPCASResult{}, ErrWCNCPIdempotency
		}
		head, err := decodeWCNCPHead(priorHead)
		if err != nil || digestBytes(priorHead) != strings.TrimPrefix(priorHeadDigest, digestPrefix) {
			return WCNCPCASResult{}, ErrWCNPCorrupt
		}
		if err := transaction.Commit(); err != nil {
			return WCNCPCASResult{}, err
		}
		rollback = false
		return WCNCPCASResult{Head: head, HeadSHA256: strings.TrimPrefix(priorHeadDigest, digestPrefix), Replayed: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return WCNCPCASResult{}, err
	}
	manifest, err := loadWCNCPManifestTx(ctx, transaction, request.RepositoryScopeSHA256, request.Kind, request.ManifestSHA256)
	if errors.Is(err, ErrWCNCPNotFound) {
		return WCNCPCASResult{}, ErrWCNCPManifestIncomplete
	}
	if err != nil {
		return WCNCPCASResult{}, err
	}
	if manifest.Generation != request.ExpectedGeneration+1 {
		return WCNCPCASResult{}, ErrWCNCPGenerationConflict
	}
	if err := storage.verifyWCNCPManifestForPublication(ctx, transaction, manifest, request.ManifestSHA256); err != nil {
		return WCNCPCASResult{}, err
	}
	current, currentDigest, exists, err := currentWCNCPHeadTx(ctx, transaction, request.RepositoryScopeSHA256, request.Kind)
	if err != nil {
		return WCNCPCASResult{}, err
	}
	if !exists {
		if request.ExpectedGeneration != 0 {
			return WCNCPCASResult{}, ErrWCNCPHeadPrecondition
		}
	} else if current.Generation != request.ExpectedGeneration || currentDigest != request.ExpectedHeadSHA256 {
		return WCNCPCASResult{}, ErrWCNCPHeadPrecondition
	}
	now := storage.now()
	createdAt, _ := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	if now.Before(createdAt) {
		return WCNCPCASResult{}, ErrWCNCPInvalid
	}
	head := WCNCPHead{}
	if request.ProposedHead != nil {
		head = *request.ProposedHead
		updatedAt, _ := time.Parse(time.RFC3339Nano, head.UpdatedAt)
		if head.Kind != request.Kind || head.RepositoryScopeSHA256 != request.RepositoryScopeSHA256 || head.Generation != manifest.Generation || head.ManifestSHA256 != request.ManifestSHA256 || head.CompatibilitySHA256 != manifest.CompatibilitySHA256 || updatedAt.Before(createdAt) || updatedAt.After(now.Add(5*time.Minute)) || (!exists && head.PreviousManifestSHA256 != "") || (exists && head.PreviousManifestSHA256 != current.ManifestSHA256) {
			return WCNCPCASResult{}, ErrWCNCPInvalid
		}
	} else {
		head = WCNCPHead{
			SchemaVersion: WCNCPHeadSchemaVersion, RecordType: "WCNCP_HEAD",
			Kind: request.Kind, RepositoryScopeSHA256: request.RepositoryScopeSHA256,
			Generation: manifest.Generation, ManifestSHA256: request.ManifestSHA256,
			CompatibilitySHA256: manifest.CompatibilitySHA256, UpdatedAt: now.Format(time.RFC3339Nano),
			Authority: StateAuthority{SelectionRequiresLocalRevalidation: true, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE"},
		}
		if exists {
			head.PreviousManifestSHA256 = current.ManifestSHA256
		}
	}
	headCanonical, headDigest, err := canonicalStateValue(head)
	if err != nil {
		return WCNCPCASResult{}, err
	}
	if exists {
		result, err := transaction.ExecContext(ctx, `UPDATE wcncp_heads SET
    generation = ?, head_digest = ?, canonical_document = ?,
    manifest_digest = ?, compatibility_sha256 = ?, updated_at_unix_ms = ?
WHERE repository_scope_sha256 = ? AND kind = ?
  AND generation = ? AND head_digest = ?`,
			head.Generation, digestPrefix+headDigest, headCanonical, digestPrefix+head.ManifestSHA256,
			head.CompatibilitySHA256, now.UnixMilli(), request.RepositoryScopeSHA256, request.Kind,
			request.ExpectedGeneration, digestPrefix+request.ExpectedHeadSHA256)
		if err != nil {
			return WCNCPCASResult{}, err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return WCNCPCASResult{}, ErrWCNCPHeadPrecondition
		}
		if _, err := transaction.ExecContext(ctx, `UPDATE wcncp_manifests SET retention_started_at_unix_ms = ?
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
			now.UnixMilli(), request.RepositoryScopeSHA256, request.Kind, digestPrefix+current.ManifestSHA256); err != nil {
			return WCNCPCASResult{}, err
		}
	} else if _, err := transaction.ExecContext(ctx, `INSERT INTO wcncp_heads (
    repository_scope_sha256, kind, generation, head_digest,
    canonical_document, manifest_digest, compatibility_sha256, updated_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		request.RepositoryScopeSHA256, request.Kind, head.Generation, digestPrefix+headDigest,
		headCanonical, digestPrefix+head.ManifestSHA256, head.CompatibilitySHA256, now.UnixMilli()); err != nil {
		return WCNCPCASResult{}, err
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE wcncp_manifests SET
    published_at_unix_ms = coalesce(published_at_unix_ms, ?),
    retention_started_at_unix_ms = NULL
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
		now.UnixMilli(), request.RepositoryScopeSHA256, request.Kind, digestPrefix+request.ManifestSHA256); err != nil {
		return WCNCPCASResult{}, err
	}
	if _, err := transaction.ExecContext(ctx, `INSERT INTO wcncp_cas_requests (
    idempotency_key, request_digest, repository_scope_sha256, kind,
    generation, head_digest, head_canonical_document, created_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		request.IdempotencyKey, digestPrefix+fingerprint, request.RepositoryScopeSHA256, request.Kind,
		head.Generation, digestPrefix+headDigest, headCanonical, now.UnixMilli()); err != nil {
		return WCNCPCASResult{}, err
	}
	if err := transaction.Commit(); err != nil {
		return WCNCPCASResult{}, err
	}
	rollback = false
	return WCNCPCASResult{Head: head, HeadSHA256: headDigest}, nil
}

// LoadCurrentWCNCP verifies the current head, manifest, references and every
// artifact byte before returning a snapshot. It is the reconstructable
// projection for fast reads.
func (storage *Storage) LoadCurrentWCNCP(ctx context.Context, repositoryScopeSHA256 string, kind StateKind) (WCNCPSnapshot, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) || !validWCNCPKind(kind) {
		return WCNCPSnapshot{}, ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPSnapshot{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	var headRaw, manifestRaw []byte
	var headDigest, manifestDigest string
	err = storage.state.database.QueryRowContext(ctx, `SELECT head.canonical_document, head.head_digest,
       manifest.canonical_document, manifest.manifest_digest
FROM wcncp_heads AS head
JOIN wcncp_manifests AS manifest
  ON manifest.repository_scope_sha256 = head.repository_scope_sha256
 AND manifest.kind = head.kind
 AND manifest.manifest_digest = head.manifest_digest
WHERE head.repository_scope_sha256 = ? AND head.kind = ?`,
		repositoryScopeSHA256, kind).Scan(&headRaw, &headDigest, &manifestRaw, &manifestDigest)
	if errors.Is(err, sql.ErrNoRows) {
		return WCNCPSnapshot{}, ErrWCNCPNotFound
	}
	if err != nil {
		return WCNCPSnapshot{}, err
	}
	head, err := decodeWCNCPHead(headRaw)
	if err != nil || digestBytes(headRaw) != strings.TrimPrefix(headDigest, digestPrefix) {
		return WCNCPSnapshot{}, ErrWCNPCorrupt
	}
	manifest, canonical, decodedDigest, err := decodeWCNCPManifest(manifestRaw)
	if err != nil || !bytes.Equal(canonical, manifestRaw) || decodedDigest != strings.TrimPrefix(manifestDigest, digestPrefix) || head.ManifestSHA256 != decodedDigest || head.Generation != manifest.Generation || head.RepositoryScopeSHA256 != manifest.RepositoryScopeSHA256 || head.Kind != manifest.Kind || head.CompatibilitySHA256 != manifest.CompatibilitySHA256 {
		return WCNCPSnapshot{}, ErrWCNPCorrupt
	}
	for _, artifact := range manifest.Artifacts {
		var size int64
		err := storage.state.database.QueryRowContext(ctx, `SELECT object.size_bytes FROM wcncp_manifest_artifacts AS artifact
JOIN wcncp_objects AS object
  ON object.repository_scope_sha256 = artifact.repository_scope_sha256
 AND object.kind = artifact.kind
 AND object.blob_digest = artifact.blob_digest
WHERE artifact.repository_scope_sha256 = ? AND artifact.kind = ?
  AND artifact.manifest_digest = ? AND artifact.role = ?
  AND artifact.blob_digest = ? AND artifact.size_bytes = ?`,
			repositoryScopeSHA256, kind, manifestDigest, artifact.Role, digestPrefix+artifact.SHA256, artifact.SizeBytes).Scan(&size)
		if err != nil || size != artifact.SizeBytes {
			return WCNCPSnapshot{}, ErrWCNPCorrupt
		}
		file, err := storage.blobs.openVerified(ctx, Blob{Digest: digestPrefix + artifact.SHA256, Size: artifact.SizeBytes})
		if file != nil {
			_ = file.Close()
		}
		if err != nil {
			return WCNCPSnapshot{}, fmt.Errorf("%w: %v", ErrWCNPCorrupt, err)
		}
	}
	for _, reference := range manifest.References {
		var present int
		err := storage.state.database.QueryRowContext(ctx, `SELECT 1 FROM wcncp_manifest_references AS reference
JOIN wcncp_manifests AS target
  ON target.repository_scope_sha256 = reference.repository_scope_sha256
 AND target.kind = reference.target_kind
 AND target.manifest_digest = reference.target_manifest_digest
WHERE reference.repository_scope_sha256 = ?
  AND reference.source_kind = ? AND reference.source_manifest_digest = ?
  AND reference.target_kind = ? AND reference.target_manifest_digest = ?
  AND reference.relation = ?`,
			repositoryScopeSHA256, kind, manifestDigest, reference.Kind, digestPrefix+reference.ManifestSHA256, reference.Relation).Scan(&present)
		if err != nil {
			return WCNCPSnapshot{}, ErrWCNPCorrupt
		}
	}
	return WCNCPSnapshot{Head: head, HeadSHA256: strings.TrimPrefix(headDigest, digestPrefix), Manifest: manifest, ManifestSHA256: strings.TrimPrefix(manifestDigest, digestPrefix)}, nil
}

// MaintainWCNCP applies only WCNCP retention. Gradle cache SLRU, existing
// typed-state retention, and capacity policy remain independent.
func (storage *Storage) MaintainWCNCP(ctx context.Context) (WCNCPMaintenanceReport, error) {
	if ctx == nil {
		return WCNCPMaintenanceReport{}, ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPMaintenanceReport{}, err
	}
	defer finish()
	storage.reconcileMutex.Lock()
	defer storage.reconcileMutex.Unlock()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	report, err := storage.maintainWCNCPMetadata(ctx, storage.now())
	if err != nil {
		return report, err
	}
	deleted, err := storage.deleteOrphanBlobs(ctx)
	if err != nil {
		return report, err
	}
	report.DeletedUnreferencedBlob = deleted
	return report, nil
}

func (storage *Storage) maintainWCNCPMetadata(ctx context.Context, now time.Time) (WCNCPMaintenanceReport, error) {
	var report WCNCPMaintenanceReport
	transaction, err := storage.state.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return report, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	// Staged manifests that never published expire after 24h unless a live
	// reference still needs them. Referenced staged parents (for example a
	// proposal staged for a decision test) must survive so the FK from the
	// child reference cannot fail.
	result, err := transaction.ExecContext(ctx, `DELETE FROM wcncp_manifests
WHERE published_at_unix_ms IS NULL AND created_at_unix_ms <= ?
  AND NOT EXISTS (
      SELECT 1 FROM wcncp_manifest_references AS reference
      WHERE reference.repository_scope_sha256 = wcncp_manifests.repository_scope_sha256
        AND reference.target_kind = wcncp_manifests.kind
        AND reference.target_manifest_digest = wcncp_manifests.manifest_digest
  )`,
		now.Add(-wcncpStagedTTL).UnixMilli())
	if err != nil {
		return report, err
	}
	report.ExpiredStagedManifests, err = affectedRows(result)
	if err != nil {
		return report, err
	}
	// Superseded observations and opportunities expire 30d after supersession.
	// Decisions are durable and never expire by retention.
	for _, testCase := range []struct {
		kind StateKind
		ttl  time.Duration
	}{
		{kind: WCNCPKindObservation, ttl: wcncpObservationTTL},
		{kind: WCNCPKindOpportunity, ttl: wcncpOpportunityTTL},
	} {
		result, err = transaction.ExecContext(ctx, `DELETE FROM wcncp_manifests
WHERE kind = ?
  AND retention_started_at_unix_ms IS NOT NULL
  AND retention_started_at_unix_ms <= ?
  AND NOT EXISTS (
      SELECT 1 FROM wcncp_heads AS head
      WHERE head.repository_scope_sha256 = wcncp_manifests.repository_scope_sha256
        AND head.kind = wcncp_manifests.kind
        AND head.manifest_digest = wcncp_manifests.manifest_digest
  )
  AND NOT EXISTS (
      SELECT 1 FROM wcncp_manifest_references AS reference
      WHERE reference.repository_scope_sha256 = wcncp_manifests.repository_scope_sha256
        AND reference.target_kind = wcncp_manifests.kind
        AND reference.target_manifest_digest = wcncp_manifests.manifest_digest
  )`,
			testCase.kind, now.Add(-testCase.ttl).UnixMilli())
		if err != nil {
			return report, err
		}
		expired, err := affectedRows(result)
		if err != nil {
			return report, err
		}
		report.ExpiredSuperseded += expired
	}
	// Proposals and validations expire 90d after supersession while no live
	// reference keeps them. A validation that references a proposal keeps the
	// proposal; a decision keeps both its proposal and validation.
	for _, testCase := range []struct {
		kind StateKind
		ttl  time.Duration
	}{
		{kind: WCNCPKindProposal, ttl: wcncpProposalTTL},
		{kind: WCNCPKindValidation, ttl: wcncpValidationTTL},
	} {
		result, err = transaction.ExecContext(ctx, `DELETE FROM wcncp_manifests
WHERE kind = ?
  AND retention_started_at_unix_ms IS NOT NULL
  AND retention_started_at_unix_ms <= ?
  AND NOT EXISTS (
      SELECT 1 FROM wcncp_heads AS head
      WHERE head.repository_scope_sha256 = wcncp_manifests.repository_scope_sha256
        AND head.kind = wcncp_manifests.kind
        AND head.manifest_digest = wcncp_manifests.manifest_digest
  )
  AND NOT EXISTS (
      SELECT 1 FROM wcncp_manifest_references AS reference
      WHERE reference.repository_scope_sha256 = wcncp_manifests.repository_scope_sha256
        AND reference.target_kind = wcncp_manifests.kind
        AND reference.target_manifest_digest = wcncp_manifests.manifest_digest
  )`,
			testCase.kind, now.Add(-testCase.ttl).UnixMilli())
		if err != nil {
			return report, err
		}
		expired, err := affectedRows(result)
		if err != nil {
			return report, err
		}
		report.ExpiredSuperseded += expired
	}
	// Staged objects without a manifest reference expire after 24h. Referenced
	// objects are preserved even when their manifest is superseded but still
	// retained.
	result, err = transaction.ExecContext(ctx, `DELETE FROM wcncp_objects
WHERE created_at_unix_ms <= ? AND NOT EXISTS (
    SELECT 1 FROM wcncp_manifest_artifacts AS artifact
    WHERE artifact.repository_scope_sha256 = wcncp_objects.repository_scope_sha256
      AND artifact.kind = wcncp_objects.kind
      AND artifact.blob_digest = wcncp_objects.blob_digest
)`, now.Add(-wcncpStagedTTL).UnixMilli())
	if err != nil {
		return report, err
	}
	report.ExpiredStagedObjects, err = affectedRows(result)
	if err != nil {
		return report, err
	}
	if err := transaction.Commit(); err != nil {
		return report, err
	}
	rollback = false
	return report, nil
}

func loadWCNCPManifestTx(ctx context.Context, transaction *sql.Tx, scope string, kind StateKind, digest string) (WCNCPManifest, error) {
	var raw []byte
	err := transaction.QueryRowContext(ctx, `SELECT canonical_document FROM wcncp_manifests
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
		scope, kind, digestPrefix+digest).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return WCNCPManifest{}, ErrWCNCPNotFound
	}
	if err != nil {
		return WCNCPManifest{}, err
	}
	manifest, canonical, actual, err := decodeWCNCPManifest(raw)
	if err != nil || !bytes.Equal(raw, canonical) || actual != digest {
		return WCNCPManifest{}, ErrWCNPCorrupt
	}
	return manifest, nil
}

func (storage *Storage) verifyWCNCPManifestForPublication(ctx context.Context, transaction *sql.Tx, manifest WCNCPManifest, manifestDigest string) error {
	for _, artifact := range manifest.Artifacts {
		var size int64
		err := transaction.QueryRowContext(ctx, `SELECT object.size_bytes FROM wcncp_manifest_artifacts AS artifact
JOIN wcncp_objects AS object
  ON object.repository_scope_sha256 = artifact.repository_scope_sha256
 AND object.kind = artifact.kind
 AND object.blob_digest = artifact.blob_digest
WHERE artifact.repository_scope_sha256 = ? AND artifact.kind = ?
  AND artifact.manifest_digest = ? AND artifact.role = ?
  AND artifact.blob_digest = ? AND artifact.size_bytes = ?`,
			manifest.RepositoryScopeSHA256, manifest.Kind, digestPrefix+manifestDigest,
			artifact.Role, digestPrefix+artifact.SHA256, artifact.SizeBytes).Scan(&size)
		if err != nil || size != artifact.SizeBytes {
			return ErrWCNCPManifestIncomplete
		}
		file, err := storage.blobs.openVerified(ctx, Blob{Digest: digestPrefix + artifact.SHA256, Size: artifact.SizeBytes})
		if file != nil {
			_ = file.Close()
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrWCNPCorrupt, err)
		}
	}
	for _, reference := range manifest.References {
		var present int
		err := transaction.QueryRowContext(ctx, `SELECT 1 FROM wcncp_manifest_references AS reference
JOIN wcncp_manifests AS target
  ON target.repository_scope_sha256 = reference.repository_scope_sha256
 AND target.kind = reference.target_kind
 AND target.manifest_digest = reference.target_manifest_digest
WHERE reference.repository_scope_sha256 = ?
  AND reference.source_kind = ? AND reference.source_manifest_digest = ?
  AND reference.target_kind = ? AND reference.target_manifest_digest = ?
  AND reference.relation = ?`,
			manifest.RepositoryScopeSHA256, manifest.Kind, digestPrefix+manifestDigest,
			reference.Kind, digestPrefix+reference.ManifestSHA256, reference.Relation).Scan(&present)
		if err != nil {
			return ErrWCNCPManifestIncomplete
		}
	}
	return nil
}

func currentWCNCPHeadTx(ctx context.Context, transaction *sql.Tx, scope string, kind StateKind) (WCNCPHead, string, bool, error) {
	var raw []byte
	var digest string
	err := transaction.QueryRowContext(ctx, `SELECT canonical_document, head_digest FROM wcncp_heads
WHERE repository_scope_sha256 = ? AND kind = ?`, scope, kind).Scan(&raw, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return WCNCPHead{}, "", false, nil
	}
	if err != nil {
		return WCNCPHead{}, "", false, err
	}
	head, err := decodeWCNCPHead(raw)
	digest = strings.TrimPrefix(digest, digestPrefix)
	if err != nil || digestBytes(raw) != digest {
		return WCNCPHead{}, "", false, ErrWCNPCorrupt
	}
	return head, digest, true, nil
}

func decodeWCNCPManifest(raw []byte) (WCNCPManifest, []byte, string, error) {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		return WCNCPManifest{}, nil, "", ErrWCNCPInvalid
	}
	var manifest WCNCPManifest
	if err := decodeStrictWCNCPJSON(canonical, &manifest); err != nil || validateWCNCPManifest(manifest) != nil {
		return WCNCPManifest{}, nil, "", ErrWCNCPInvalid
	}
	return manifest, canonical, digestBytes(canonical), nil
}

// DecodeWCNCPManifest verifies one canonical manifest received for offline
// inspection. Non-canonical but semantically equivalent documents are
// rejected so they cannot acquire a second identity.
func DecodeWCNCPManifest(raw []byte) (WCNCPManifest, string, error) {
	manifest, canonical, digest, err := decodeWCNCPManifest(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return WCNCPManifest{}, "", ErrWCNCPInvalid
	}
	return manifest, digest, nil
}

func decodeWCNCPHead(raw []byte) (WCNCPHead, error) {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return WCNCPHead{}, ErrWCNCPInvalid
	}
	var head WCNCPHead
	if err := decodeStrictWCNCPJSON(canonical, &head); err != nil ||
		head.SchemaVersion != WCNCPHeadSchemaVersion || head.RecordType != "WCNCP_HEAD" ||
		!validWCNCPKind(head.Kind) || !validSHA256(head.RepositoryScopeSHA256) ||
		head.Generation < 1 || head.Generation > maximumWCNCPGeneration ||
		!validSHA256(head.ManifestSHA256) || !validSHA256(head.CompatibilitySHA256) ||
		!contractcrypto.ValidUTCTimestamp(head.UpdatedAt) || !validStateAuthority(head.Authority) ||
		(head.Generation == 1 && head.PreviousManifestSHA256 != "") ||
		(head.Generation > 1 && !validSHA256(head.PreviousManifestSHA256)) {
		return WCNCPHead{}, ErrWCNCPInvalid
	}
	return head, nil
}

// DecodeWCNCPHead verifies one canonical head received for offline inspection.
func DecodeWCNCPHead(raw []byte) (WCNCPHead, string, error) {
	head, err := decodeWCNCPHead(raw)
	if err != nil {
		return WCNCPHead{}, "", err
	}
	return head, digestBytes(raw), nil
}

func validateWCNCPManifest(manifest WCNCPManifest) error {
	if manifest.SchemaVersion != WCNCPManifestSchemaVersion || manifest.RecordType != "WCNCP_MANIFEST" ||
		!validWCNCPKind(manifest.Kind) || !validSHA256(manifest.RepositoryScopeSHA256) ||
		manifest.Generation < 1 || manifest.Generation > maximumWCNCPGeneration ||
		!validSHA256(manifest.CompatibilitySHA256) || !validSHA256(manifest.BindingsSHA256) ||
		!stateRevisionPattern.MatchString(manifest.Origin.BaseRevision) ||
		!stateRevisionPattern.MatchString(manifest.Origin.TargetRevision) ||
		!validSHA256(manifest.Origin.BuildOptExecutableSHA256) ||
		!validSHA256(manifest.Origin.WrapperSHA256) ||
		!stateGradlePattern.MatchString(manifest.Origin.GradleVersion) ||
		len(manifest.Artifacts) < 1 || len(manifest.Artifacts) > 64 ||
		len(manifest.References) > 64 ||
		!contractcrypto.ValidUTCTimestamp(manifest.CreatedAt) ||
		!validStateAuthority(manifest.Authority) {
		return ErrWCNCPInvalid
	}
	if manifest.RetentionClass != wcncpRetentionClass(manifest.Kind) {
		return ErrWCNCPInvalid
	}
	if manifest.Status != "COMPLETE" {
		return ErrWCNCPInvalid
	}
	if manifest.ExpiresAt != "" {
		return ErrWCNCPInvalid
	}
	expectedRole := wcncpArtifactRole(manifest.Kind)
	expectedSchema := wcncpPayloadSchemaVersion(manifest.Kind)
	seenArtifacts := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		identity := artifact.Role + "\x00" + artifact.SHA256 + "\x00" + fmt.Sprintf("%d", artifact.SizeBytes) + "\x00" + artifact.PayloadSchemaVersion
		if artifact.Role != expectedRole || artifact.PayloadSchemaVersion != expectedSchema ||
			!validSHA256(artifact.SHA256) || artifact.SizeBytes < 1 || artifact.SizeBytes > maximumWCNCPArtifactBytes ||
			seenArtifacts[identity] {
			return ErrWCNCPInvalid
		}
		seenArtifacts[identity] = true
	}
	seenReferences := map[string]bool{}
	for _, reference := range manifest.References {
		identity := string(reference.Kind) + "\x00" + reference.ManifestSHA256 + "\x00" + reference.Relation
		if !validWCNCPKind(reference.Kind) || !validSHA256(reference.ManifestSHA256) || seenReferences[identity] {
			return ErrWCNCPInvalid
		}
		// Relations are constrained by source kind so a decision cannot be
		// mistaken for evidence and acceptance cannot become application.
		switch manifest.Kind {
		case WCNCPKindObservation:
			if len(manifest.References) != 0 {
				return ErrWCNCPInvalid
			}
		case WCNCPKindOpportunity:
			if reference.Relation != "DERIVED_FROM" || reference.Kind != WCNCPKindObservation {
				return ErrWCNCPInvalid
			}
		case WCNCPKindProposal:
			if reference.Relation != "DERIVED_FROM" || reference.Kind != WCNCPKindOpportunity {
				return ErrWCNCPInvalid
			}
		case WCNCPKindValidation:
			if reference.Relation != "VALIDATES" || reference.Kind != WCNCPKindProposal {
				return ErrWCNCPInvalid
			}
		case WCNCPKindDecision:
			if reference.Relation != "DECIDES" || (reference.Kind != WCNCPKindProposal && reference.Kind != WCNCPKindValidation) {
				return ErrWCNCPInvalid
			}
		}
		if reference.Relation != "DERIVED_FROM" && reference.Relation != "QUALIFICATION" && reference.Relation != "VALIDATES" && reference.Relation != "DECIDES" {
			return ErrWCNCPInvalid
		}
		seenReferences[identity] = true
	}
	// Decisions require both proposal and validation bindings so acceptance
	// cannot be reported without validated evidence.
	if manifest.Kind == WCNCPKindDecision && len(manifest.References) < 2 {
		return ErrWCNCPInvalid
	}
	_ = wcncpScopePattern
	return nil
}

func wcncpCASFingerprint(request WCNCPCASRequest) (string, error) {
	fields := map[string]any{
		"expectedGeneration":    request.ExpectedGeneration,
		"expectedHeadSha256":    nullableStateDigest(request.ExpectedHeadSHA256),
		"idempotencyKey":        request.IdempotencyKey,
		"kind":                  request.Kind,
		"manifestSha256":        request.ManifestSHA256,
		"repositoryScopeSha256": request.RepositoryScopeSHA256,
	}
	if request.ProposedHead != nil {
		fields["proposedHead"] = *request.ProposedHead
	}
	_, digest, err := canonicalStateValue(fields)
	return digest, err
}

// CanonicalWCNCPValue renders one client-authored WCNCP document with the
// same JCS identity as durable storage.
func CanonicalWCNCPValue(value any) ([]byte, string, error) {
	return canonicalStateValue(value)
}
