package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	maximumCommitDecisionBytes = 4 << 20
	maximumCommitObjects       = 4096
)

var (
	identifierPattern = regexp.MustCompile(
		`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`,
	)
	cacheKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,256}$`)
	revisionPattern = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	digestPattern   = regexp.MustCompile(
		`^(?:sha256|hmac-sha256):[0-9a-f]{64}$`,
	)
)

// RepositoryIdentity is the immutable tenant/repository/trust binding shared
// by an attempt and its authorization.
type RepositoryIdentity struct {
	Tenant      string `json:"tenant"`
	Repository  string `json:"repository"`
	TrustDomain string `json:"trustDomain"`
}

// CommitObject is one exactly covered pending object.
type CommitObject struct {
	NamespaceGeneration int64  `json:"namespaceGeneration"`
	Key                 string `json:"key"`
	Checksum            string `json:"checksum"`
	SizeBytes           int64  `json:"sizeBytes"`
}

// TestOptimizationGrant binds the optional test-cache authority.
type TestOptimizationGrant struct {
	State      string `json:"state"`
	GrantID    string `json:"grantId,omitempty"`
	GrantEpoch *int64 `json:"grantEpoch,omitempty"`
	Digest     string `json:"digest,omitempty"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// CommitValidation is the positive validation bound into a decision.
type CommitValidation struct {
	Status      string `json:"status"`
	EvidenceRef string `json:"evidenceRef,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// CommitAuthentication identifies the Ed25519 authority and signature.
type CommitAuthentication struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

// CommitDecision is the strict COMMIT_DECISION v1 wire document.
type CommitDecision struct {
	SchemaVersion             string                `json:"schemaVersion"`
	RecordType                string                `json:"recordType"`
	ContractVersion           string                `json:"contractVersion"`
	DecisionID                string                `json:"decisionId"`
	DecisionDigest            string                `json:"decisionDigest"`
	AttemptID                 string                `json:"attemptId"`
	Repository                RepositoryIdentity    `json:"repository"`
	SourceRevision            string                `json:"sourceRevision"`
	SourceStateDigest         string                `json:"sourceStateDigest"`
	Objects                   []CommitObject        `json:"objects"`
	PolicyDigest              string                `json:"policyDigest"`
	ConfigurationPolicyDigest string                `json:"configurationPolicyDigest"`
	CacheContractDigest       string                `json:"cacheContractDigest"`
	TestOptimizationGrant     TestOptimizationGrant `json:"testOptimizationGrant"`
	RevocationEpoch           int64                 `json:"revocationEpoch"`
	Validation                CommitValidation      `json:"validation"`
	IssuedAt                  string                `json:"issuedAt"`
	ExpiresAt                 string                `json:"expiresAt"`
	Authentication            CommitAuthentication  `json:"authentication"`
}

// VerifiedCommitDecision can only be produced by complete canonical,
// temporal, epoch, and Ed25519 verification in this package.
type VerifiedCommitDecision struct {
	decision  CommitDecision
	canonical []byte
	issuedAt  time.Time
	expiresAt time.Time
}

// Decision returns a copy of the verified semantic document.
func (verified VerifiedCommitDecision) Decision() CommitDecision {
	decision := verified.decision
	decision.Objects = slices.Clone(decision.Objects)
	if decision.TestOptimizationGrant.GrantEpoch != nil {
		grantEpoch := *decision.TestOptimizationGrant.GrantEpoch
		decision.TestOptimizationGrant.GrantEpoch = &grantEpoch
	}
	return decision
}

// CanonicalDocument returns a copy of the exact verified durable bytes.
func (verified VerifiedCommitDecision) CanonicalDocument() []byte {
	return bytes.Clone(verified.canonical)
}

// VerifyCommitDecision validates the strict canonical document against the
// current revocation epoch and the exact named Ed25519 public key.
func VerifyCommitDecision(
	ctx context.Context,
	raw []byte,
	publicKeys map[string]ed25519.PublicKey,
	currentRevocationEpoch int64,
	now time.Time,
) (VerifiedCommitDecision, error) {
	if ctx == nil {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: nil context",
			ErrCommitRejected,
		)
	}
	if err := ctx.Err(); err != nil {
		return VerifiedCommitDecision{}, err
	}
	if len(raw) == 0 || len(raw) > maximumCommitDecisionBytes {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: invalid document size",
			ErrCommitRejected,
		)
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: document is not canonical JCS",
			ErrCommitRejected,
		)
	}

	var decision CommitDecision
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decision); err != nil {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: decode: %v",
			ErrCommitRejected,
			err,
		)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: %v",
			ErrCommitRejected,
			err,
		)
	}
	issuedAt, expiresAt, err := validateCommitDecisionSemantics(
		decision,
		currentRevocationEpoch,
		now.UTC(),
	)
	if err != nil {
		return VerifiedCommitDecision{}, err
	}

	digest, err := commitDecisionDigest(canonical)
	if err != nil || digest != decision.DecisionDigest {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: decision digest mismatch",
			ErrCommitRejected,
		)
	}
	publicKey, ok := publicKeys[decision.Authentication.KeyID]
	if !ok || len(publicKey) != ed25519.PublicKeySize {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: unknown decision key",
			ErrCommitRejected,
		)
	}
	signature, err := base64.RawURLEncoding.DecodeString(
		decision.Authentication.Signature,
	)
	if err != nil ||
		len(signature) != ed25519.SignatureSize ||
		!ed25519.Verify(
			publicKey,
			commitDecisionSignaturePayload(
				decision.Authentication.KeyID,
				digest,
			),
			signature,
		) {
		return VerifiedCommitDecision{}, fmt.Errorf(
			"%w: invalid Ed25519 signature",
			ErrCommitRejected,
		)
	}
	return VerifiedCommitDecision{
		decision:  decision,
		canonical: bytes.Clone(canonical),
		issuedAt:  issuedAt,
		expiresAt: expiresAt,
	}, nil
}

func validateCommitDecisionSemantics(
	decision CommitDecision,
	currentRevocationEpoch int64,
	now time.Time,
) (time.Time, time.Time, error) {
	reject := func(reason string) (time.Time, time.Time, error) {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"%w: %s",
			ErrCommitRejected,
			reason,
		)
	}
	if decision.SchemaVersion != "1.0" ||
		decision.RecordType != "COMMIT_DECISION" ||
		decision.ContractVersion != "buildopt-cache-commit/v1" {
		return reject("unsupported decision contract")
	}
	for name, value := range map[string]string{
		"decisionId":             decision.DecisionID,
		"attemptId":              decision.AttemptID,
		"repository.tenant":      decision.Repository.Tenant,
		"repository.repository":  decision.Repository.Repository,
		"repository.trustDomain": decision.Repository.TrustDomain,
		"authentication.keyId":   decision.Authentication.KeyID,
	} {
		if !identifierPattern.MatchString(value) {
			return reject("invalid " + name)
		}
	}
	if !revisionPattern.MatchString(decision.SourceRevision) ||
		!digestPattern.MatchString(decision.SourceStateDigest) {
		return reject("invalid source binding")
	}
	for name, digest := range map[string]string{
		"decisionDigest":            decision.DecisionDigest,
		"policyDigest":              decision.PolicyDigest,
		"configurationPolicyDigest": decision.ConfigurationPolicyDigest,
		"cacheContractDigest":       decision.CacheContractDigest,
	} {
		if _, err := parseDigest(digest); err != nil {
			return reject("invalid " + name)
		}
	}
	if len(decision.Objects) < 1 ||
		len(decision.Objects) > maximumCommitObjects {
		return reject("invalid object count")
	}
	var previous CommitObject
	for index, object := range decision.Objects {
		if object.NamespaceGeneration < 1 ||
			!cacheKeyPattern.MatchString(object.Key) ||
			object.SizeBytes < 0 ||
			object.SizeBytes > MaximumBlobBytes {
			return reject("invalid object")
		}
		if _, err := parseDigest(object.Checksum); err != nil {
			return reject("invalid object checksum")
		}
		if index > 0 {
			if object.NamespaceGeneration < previous.NamespaceGeneration ||
				(object.NamespaceGeneration == previous.NamespaceGeneration &&
					object.Key <= previous.Key) {
				return reject("objects are not strictly sorted")
			}
		}
		previous = object
	}
	if decision.RevocationEpoch < 0 ||
		decision.RevocationEpoch != currentRevocationEpoch {
		return reject("revocation epoch is stale or future")
	}

	issuedAt, err := parseDecisionTimestamp(decision.IssuedAt)
	if err != nil {
		return reject("invalid issuedAt")
	}
	expiresAt, err := parseDecisionTimestamp(decision.ExpiresAt)
	if err != nil ||
		issuedAt.After(now) ||
		!expiresAt.After(now) ||
		!expiresAt.After(issuedAt) {
		return reject("decision is not currently valid")
	}
	switch decision.Validation.Status {
	case "PASSED":
		if decision.Validation.EvidenceRef == "" ||
			!identifierPattern.MatchString(
				decision.Validation.EvidenceRef,
			) ||
			decision.Validation.Reason != "" {
			return reject("invalid passed validation")
		}
		completedAt, completedErr := parseDecisionTimestamp(
			decision.Validation.CompletedAt,
		)
		validationExpiresAt, expiresErr := parseDecisionTimestamp(
			decision.Validation.ExpiresAt,
		)
		if completedErr != nil ||
			expiresErr != nil ||
			completedAt.After(issuedAt) ||
			validationExpiresAt.Before(expiresAt) {
			return reject("invalid validation window")
		}
	case "NOT_REQUIRED":
		if decision.Validation.Reason != "ALLOWLISTED_DIRECT_ACTION" ||
			decision.Validation.EvidenceRef != "" ||
			decision.Validation.CompletedAt != "" ||
			decision.Validation.ExpiresAt != "" {
			return reject("invalid direct-action validation")
		}
	default:
		return reject("invalid validation status")
	}
	switch decision.TestOptimizationGrant.State {
	case "PRESENT":
		if decision.TestOptimizationGrant.GrantID == "" ||
			!identifierPattern.MatchString(
				decision.TestOptimizationGrant.GrantID,
			) ||
			decision.TestOptimizationGrant.GrantEpoch == nil ||
			*decision.TestOptimizationGrant.GrantEpoch < 0 ||
			decision.TestOptimizationGrant.Reason != "" {
			return reject("invalid test optimization grant")
		}
		if _, err := parseDigest(
			decision.TestOptimizationGrant.Digest,
		); err != nil {
			return reject("invalid test optimization grant digest")
		}
		grantExpiresAt, err := parseDecisionTimestamp(
			decision.TestOptimizationGrant.ExpiresAt,
		)
		if err != nil || grantExpiresAt.Before(expiresAt) {
			return reject("invalid test optimization grant window")
		}
	case "NOT_REQUIRED":
		if decision.TestOptimizationGrant.Reason != "NO_TEST_OUTPUTS" ||
			decision.TestOptimizationGrant.GrantID != "" ||
			decision.TestOptimizationGrant.GrantEpoch != nil ||
			decision.TestOptimizationGrant.Digest != "" ||
			decision.TestOptimizationGrant.ExpiresAt != "" {
			return reject("invalid no-test grant")
		}
	default:
		return reject("invalid test optimization grant state")
	}
	if decision.Authentication.Algorithm != "Ed25519" ||
		len(decision.Authentication.Signature) != 86 {
		return reject("invalid authentication envelope")
	}
	return issuedAt, expiresAt, nil
}

func parseDecisionTimestamp(value string) (time.Time, error) {
	if !contractcrypto.ValidUTCTimestamp(value) {
		return time.Time{}, errors.New("timestamp is not canonical UTC")
	}
	return time.Parse(time.RFC3339Nano, value)
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func commitDecisionDigest(canonical []byte) (string, error) {
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return "", err
	}
	delete(document, "decisionDigest")
	authentication, ok := document["authentication"].(map[string]any)
	if !ok {
		return "", errors.New("missing authentication object")
	}
	delete(authentication, "signature")
	payload, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	payload, err = contractcrypto.CanonicalizeJCS(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return digestPrefix + hex.EncodeToString(digest[:]), nil
}

func commitDecisionSignaturePayload(keyID string, digest string) []byte {
	return []byte(
		"buildopt-cache-commit/v1\x00" +
			keyID +
			"\x00" +
			digest,
	)
}

func canonicalDocumentFingerprint(canonical []byte) string {
	digest := sha256.Sum256(canonical)
	return digestPrefix + hex.EncodeToString(digest[:])
}

func fingerprintValue(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return "", err
	}
	return canonicalDocumentFingerprint(canonical), nil
}

func validIdentifier(value string) bool {
	return identifierPattern.MatchString(value)
}

func validCacheKey(value string) bool {
	return cacheKeyPattern.MatchString(value)
}

func validSHA256Digest(value string) bool {
	_, err := parseDigest(value)
	return err == nil
}

func validSourceDigest(value string) bool {
	return digestPattern.MatchString(value)
}

func sameCommitObjects(left []CommitObject, right []CommitObject) bool {
	return slices.Equal(left, right)
}
