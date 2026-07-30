package localauthority

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
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	// AuthorityContractVersion is the local private-beta authority envelope.
	AuthorityContractVersion = "buildopt-local-cache-authority/v1"
	// TrustRootContractVersion is the pinned local public-key document.
	TrustRootContractVersion = "buildopt-local-trust-root/v1"
	// SigningKeyContractVersion is the private local deployment-key document.
	SigningKeyContractVersion = "buildopt-local-signing-key/v1"
	// CredentialBytes is the exact local cache credential entropy.
	CredentialBytes = 32

	maximumAuthorityBytes  = 4 << 20
	maximumTrustRootBytes  = 64 << 10
	maximumSigningKeyBytes = 4 << 10
	maximumPolicyTasks     = 4096
	maximumPolicyLifetime  = 30 * 24 * time.Hour
	maximumAttemptLifetime = 24 * time.Hour
	digestPrefix           = "sha256:"
)

var (
	identifierPattern = regexp.MustCompile(
		`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`,
	)
	revisionPattern = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	digestPattern   = regexp.MustCompile(
		`^(?:sha256|hmac-sha256):[0-9a-f]{64}$`,
	)
	namespacePattern = regexp.MustCompile(
		`^[A-Za-z0-9._/-]{1,512}$`,
	)
	versionRangePattern = regexp.MustCompile(
		`^(?:=[0-9]+\.[0-9]+\.[0-9]+|>=[0-9]+\.[0-9]+\.[0-9]+ <[0-9]+\.[0-9]+\.[0-9]+)$`,
	)
)

// ErrRejected means local policy or revocation authority was not proven.
var ErrRejected = errors.New("local cache authority was rejected")

// ErrRollback means validly signed state would move a persisted monotonic
// policy, namespace, or revocation generation backwards.
var ErrRollback = errors.New("local cache authority rollback was rejected")

// RepositoryIdentity is the exact deployment-isolated authority scope.
type RepositoryIdentity struct {
	Tenant      string `json:"tenant"`
	Repository  string `json:"repository"`
	TrustDomain string `json:"trustDomain"`
}

// AuthorityAttempt binds one invocation and its local data-plane credential.
type AuthorityAttempt struct {
	AttemptID        string `json:"attemptId"`
	OwnerID          string `json:"ownerId"`
	LeaseID          string `json:"leaseId"`
	LeaseExpiresAt   string `json:"leaseExpiresAt"`
	AllowRead        bool   `json:"allowRead"`
	AllowWrite       bool   `json:"allowWrite"`
	CredentialDigest string `json:"credentialDigest"`
}

// RemoteCachePolicy is the exact OPTIMIZATION_POLICY remote-cache grant.
type RemoteCachePolicy struct {
	Read                bool   `json:"read"`
	Write               string `json:"write"`
	Namespace           string `json:"namespace"`
	NamespaceGeneration int64  `json:"namespaceGeneration"`
}

// ConfigurationCachePolicy is the tracked Gradle configuration decision.
type ConfigurationCachePolicy struct {
	Enabled         bool   `json:"enabled"`
	ContractVersion string `json:"contractVersion"`
}

// GrantReference is the optional Test Optimization authority reference.
type GrantReference struct {
	Digest    string `json:"digest"`
	ExpiresAt string `json:"expiresAt"`
}

// ResourceProfileReference selects one already authenticated resource profile.
type ResourceProfileReference struct {
	ProfileID      string `json:"profileId"`
	ProfileDigest  string `json:"profileDigest"`
	CatalogVersion string `json:"catalogVersion"`
}

// PolicyBudgets bound synchronous and validation overhead.
type PolicyBudgets struct {
	MaxSynchronousOverheadMs    int64   `json:"maxSynchronousOverheadMs"`
	MaxSynchronousOverheadRatio float64 `json:"maxSynchronousOverheadRatio"`
	MaxValidationRunnerMsPerDay int64   `json:"maxValidationRunnerMsPerDay"`
}

// QualifiedTask is one explicit policy-owned cache contract.
type QualifiedTask struct {
	ImplementationHash  string `json:"implementationHash"`
	QualificationSource string `json:"qualificationSource"`
	ContractRef         string `json:"contractRef"`
	CacheContractDigest string `json:"cacheContractDigest"`
	QualificationState  string `json:"qualificationState"`
	RepeatabilityGate   string `json:"repeatabilityGate"`
	RelocatabilityGate  string `json:"relocatabilityGate"`
}

// AffectedBuild declares where a policy may act.
type AffectedBuild struct {
	EnabledInCI    bool `json:"enabledInCi"`
	EnabledLocally bool `json:"enabledLocally"`
}

// OptimizationPolicy is the complete strict OPTIMIZATION_POLICY v1 document.
// Its existing schema carries signatureKeyId while the local authority
// envelope supplies the actual deployment-key signature.
type OptimizationPolicy struct {
	SchemaVersion               string                   `json:"schemaVersion"`
	RecordType                  string                   `json:"recordType"`
	PolicyID                    string                   `json:"policyId"`
	PolicyVersion               int64                    `json:"policyVersion"`
	PolicyDigest                string                   `json:"policyDigest"`
	ConfigurationPolicyDigest   string                   `json:"configurationPolicyDigest"`
	RevocationEpoch             int64                    `json:"revocationEpoch"`
	L1SecurityGeneration        int64                    `json:"l1SecurityGeneration"`
	GatewayConnectionGeneration int64                    `json:"gatewayConnectionGeneration"`
	SignatureKeyID              string                   `json:"signatureKeyId"`
	IssuedAt                    string                   `json:"issuedAt"`
	LauncherVersionRange        string                   `json:"launcherVersionRange"`
	PluginVersionRange          string                   `json:"pluginVersionRange"`
	Mode                        string                   `json:"mode"`
	AllowedActions              []string                 `json:"allowedActions"`
	RemoteCache                 RemoteCachePolicy        `json:"remoteCache"`
	ConfigurationCache          ConfigurationCachePolicy `json:"configurationCache"`
	TestOptimizationGrant       *GrantReference          `json:"testOptimizationGrant,omitempty"`
	ResourceProfile             ResourceProfileReference `json:"resourceProfile"`
	Budgets                     PolicyBudgets            `json:"budgets"`
	ExportProfile               string                   `json:"exportProfile"`
	QualifiedTasks              []QualifiedTask          `json:"qualifiedTasks"`
	AffectedBuild               AffectedBuild            `json:"affectedBuild"`
	ExpiresAt                   string                   `json:"expiresAt"`
}

// Authentication is the canonical Ed25519 envelope shared by local records.
type Authentication struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

// RevocationState is the strict cumulative cache-control revocation record.
type RevocationState struct {
	ContractVersion       string         `json:"contractVersion"`
	RequestID             string         `json:"requestId"`
	TrustDomain           string         `json:"trustDomain"`
	RevocationEpoch       int64          `json:"revocationEpoch"`
	L1SecurityGeneration  int64          `json:"l1SecurityGeneration"`
	CumulativeStateDigest string         `json:"cumulativeStateDigest"`
	ValidUntil            string         `json:"validUntil"`
	Authentication        Authentication `json:"authentication"`
}

// Document is the canonical, signed local authority consumed before Gradle
// and independently registered with Shared before any cache route is active.
type Document struct {
	SchemaVersion       string             `json:"schemaVersion"`
	AuthorityDigest     string             `json:"authorityDigest"`
	Repository          RepositoryIdentity `json:"repository"`
	SourceRevision      string             `json:"sourceRevision"`
	SourceStateDigest   string             `json:"sourceStateDigest"`
	CacheContractDigest string             `json:"cacheContractDigest"`
	Attempt             AuthorityAttempt   `json:"attempt"`
	Policy              OptimizationPolicy `json:"policy"`
	Revocation          RevocationState    `json:"revocation"`
	Authentication      Authentication     `json:"authentication"`
}

// PublicKey binds one pinned deployment-key identity to raw Ed25519 bytes.
type PublicKey struct {
	KeyID     string `json:"keyId"`
	PublicKey string `json:"publicKey"`
}

// TrustRoot is provisioned out of band and never accepted from an authority
// document itself.
type TrustRoot struct {
	SchemaVersion string      `json:"schemaVersion"`
	Keys          []PublicKey `json:"keys"`
}

// SigningKey is the private deployment key held only by the beta server.
type SigningKey struct {
	SchemaVersion string `json:"schemaVersion"`
	KeyID         string `json:"keyId"`
	PrivateKey    string `json:"privateKey"`
}

// Verified is immutable authority proven against an out-of-band trust root.
type Verified struct {
	document       Document
	canonical      []byte
	issuedAt       time.Time
	expiresAt      time.Time
	revValidUntil  time.Time
	leaseExpiresAt time.Time
}

// Document returns a defensive semantic copy.
func (verified Verified) Document() Document {
	document := verified.document
	document.Policy.AllowedActions = slices.Clone(
		document.Policy.AllowedActions,
	)
	document.Policy.QualifiedTasks = slices.Clone(
		document.Policy.QualifiedTasks,
	)
	if document.Policy.TestOptimizationGrant != nil {
		grant := *document.Policy.TestOptimizationGrant
		document.Policy.TestOptimizationGrant = &grant
	}
	return document
}

// CanonicalDocument returns the exact authenticated durable bytes.
func (verified Verified) CanonicalDocument() []byte {
	return bytes.Clone(verified.canonical)
}

// ExpiresAt is the earliest policy, revocation, attempt, or envelope horizon.
func (verified Verified) ExpiresAt() time.Time {
	return minTime(
		verified.expiresAt,
		verified.revValidUntil,
		verified.leaseExpiresAt,
	)
}

// IssuedAt returns the authenticated policy issue time.
func (verified Verified) IssuedAt() time.Time {
	return verified.issuedAt
}

// SupportsComponents reports whether both exact component versions satisfy the
// authenticated policy ranges.
func (verified Verified) SupportsComponents(
	launcherVersion string,
	pluginVersion string,
) bool {
	return versionSatisfies(
		launcherVersion,
		verified.document.Policy.LauncherVersionRange,
	) && versionSatisfies(
		pluginVersion,
		verified.document.Policy.PluginVersionRange,
	)
}

// Verify authenticates one canonical local authority and its cumulative
// revocation record. Credential is the decoded 32-byte data-plane secret.
func Verify(
	ctx context.Context,
	raw []byte,
	keys map[string]ed25519.PublicKey,
	credential []byte,
	now time.Time,
) (Verified, error) {
	if ctx == nil {
		return Verified{}, reject("nil context")
	}
	if err := ctx.Err(); err != nil {
		return Verified{}, err
	}
	if len(raw) == 0 || len(raw) > maximumAuthorityBytes {
		return Verified{}, reject("invalid document size")
	}
	if len(credential) != CredentialBytes {
		return Verified{}, reject("invalid local cache credential")
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return Verified{}, reject("document is not canonical JCS")
	}

	var document Document
	if err := decodeStrict(canonical, &document); err != nil {
		return Verified{}, reject("decode: " + err.Error())
	}
	issuedAt, expiresAt, revValidUntil, leaseExpiresAt, err :=
		validateSemantics(document, credential, now.UTC())
	if err != nil {
		return Verified{}, err
	}

	root, err := decodeJSONObject(canonical)
	if err != nil {
		return Verified{}, reject("decode canonical object")
	}
	policyObject, ok := root["policy"].(map[string]any)
	if !ok {
		return Verified{}, reject("missing policy object")
	}
	policyDigest, err := digestObject(policyObject, "policyDigest", "")
	if err != nil || policyDigest != document.Policy.PolicyDigest {
		return Verified{}, reject("policy digest mismatch")
	}
	revocationObject, ok := root["revocation"].(map[string]any)
	if !ok {
		return Verified{}, reject("missing revocation object")
	}
	revocationDigest, err := digestRevocationObject(revocationObject)
	if err != nil ||
		revocationDigest != document.Revocation.CumulativeStateDigest {
		return Verified{}, reject("revocation digest mismatch")
	}
	authorityDigest, err := digestObject(
		root,
		"authorityDigest",
		"authentication",
	)
	if err != nil || authorityDigest != document.AuthorityDigest {
		return Verified{}, reject("authority digest mismatch")
	}

	publicKey, ok := keys[document.Authentication.KeyID]
	if !ok || len(publicKey) != ed25519.PublicKeySize {
		return Verified{}, reject("unknown deployment key")
	}
	if !verifySignature(
		publicKey,
		document.Revocation.Authentication.Signature,
		signaturePayload(
			"buildopt-cache-revocation/v1",
			document.Revocation.Authentication.KeyID,
			revocationDigest,
		),
	) {
		return Verified{}, reject("invalid revocation signature")
	}
	if !verifySignature(
		publicKey,
		document.Authentication.Signature,
		signaturePayload(
			AuthorityContractVersion,
			document.Authentication.KeyID,
			authorityDigest,
		),
	) {
		return Verified{}, reject("invalid authority signature")
	}
	return Verified{
		document:       document,
		canonical:      bytes.Clone(canonical),
		issuedAt:       issuedAt,
		expiresAt:      expiresAt,
		revValidUntil:  revValidUntil,
		leaseExpiresAt: leaseExpiresAt,
	}, nil
}

func validateSemantics(
	document Document,
	credential []byte,
	now time.Time,
) (time.Time, time.Time, time.Time, time.Time, error) {
	fail := func(reason string) (
		time.Time,
		time.Time,
		time.Time,
		time.Time,
		error,
	) {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{},
			reject(reason)
	}
	if document.SchemaVersion != AuthorityContractVersion {
		return fail("unsupported authority contract")
	}
	for name, value := range map[string]string{
		"repository.tenant":      document.Repository.Tenant,
		"repository.repository":  document.Repository.Repository,
		"repository.trustDomain": document.Repository.TrustDomain,
		"attempt.attemptId":      document.Attempt.AttemptID,
		"attempt.ownerId":        document.Attempt.OwnerID,
		"attempt.leaseId":        document.Attempt.LeaseID,
		"policy.policyId":        document.Policy.PolicyID,
		"policy.signatureKeyId":  document.Policy.SignatureKeyID,
		"revocation.requestId":   document.Revocation.RequestID,
		"authentication.keyId":   document.Authentication.KeyID,
	} {
		if !identifierPattern.MatchString(value) {
			return fail("invalid " + name)
		}
	}
	if !revisionPattern.MatchString(document.SourceRevision) ||
		!digestPattern.MatchString(document.SourceStateDigest) {
		return fail("invalid source binding")
	}
	for name, value := range map[string]string{
		"authorityDigest":           document.AuthorityDigest,
		"cacheContractDigest":       document.CacheContractDigest,
		"credentialDigest":          document.Attempt.CredentialDigest,
		"policyDigest":              document.Policy.PolicyDigest,
		"configurationPolicyDigest": document.Policy.ConfigurationPolicyDigest,
		"revocationDigest":          document.Revocation.CumulativeStateDigest,
	} {
		if !validSHA256(value) {
			return fail("invalid " + name)
		}
	}
	if document.Attempt.CredentialDigest != credentialDigest(credential) {
		return fail("local cache credential digest mismatch")
	}
	if !document.Attempt.AllowRead && !document.Attempt.AllowWrite {
		return fail("authority grants no cache operation")
	}
	if document.Policy.SchemaVersion != "1.0" ||
		document.Policy.RecordType != "OPTIMIZATION_POLICY" ||
		document.Policy.PolicyVersion < 1 ||
		document.Policy.Mode != "VERIFIED" {
		return fail("unsupported optimization policy")
	}
	if document.Policy.SignatureKeyID != document.Authentication.KeyID ||
		document.Revocation.Authentication.KeyID !=
			document.Authentication.KeyID ||
		document.Revocation.Authentication.Algorithm != "Ed25519" ||
		document.Authentication.Algorithm != "Ed25519" {
		return fail("deployment key binding mismatch")
	}
	if document.Revocation.ContractVersion !=
		"buildopt-cache-control/v1" ||
		document.Revocation.TrustDomain != document.Repository.TrustDomain ||
		document.Revocation.RevocationEpoch < 1 ||
		document.Revocation.L1SecurityGeneration < 1 {
		return fail("invalid cumulative revocation state")
	}
	if document.Policy.RevocationEpoch !=
		document.Revocation.RevocationEpoch ||
		document.Policy.L1SecurityGeneration !=
			document.Revocation.L1SecurityGeneration ||
		document.Policy.GatewayConnectionGeneration < 1 {
		return fail("policy generation is not current")
	}
	if document.Policy.RemoteCache.NamespaceGeneration < 1 ||
		!namespacePattern.MatchString(document.Policy.RemoteCache.Namespace) {
		return fail("invalid remote cache namespace")
	}
	if !document.Policy.AffectedBuild.EnabledInCI &&
		!document.Policy.AffectedBuild.EnabledLocally {
		return fail("policy enables no build class")
	}
	if document.Attempt.AllowRead && !document.Policy.RemoteCache.Read {
		return fail("read authority exceeds policy")
	}
	switch document.Policy.RemoteCache.Write {
	case "DISABLED":
		if document.Attempt.AllowWrite {
			return fail("write authority exceeds policy")
		}
	case "TRUSTED_CI_ONLY":
		if !document.Policy.AffectedBuild.EnabledInCI {
			return fail("trusted CI write lacks CI scope")
		}
	default:
		return fail("invalid remote cache write policy")
	}
	if document.Attempt.AllowWrite &&
		document.Policy.RemoteCache.Write != "TRUSTED_CI_ONLY" {
		return fail("write authority exceeds policy")
	}
	if !containsUniqueAllowedAction(
		document.Policy.AllowedActions,
		"REMOTE_CACHE_ALLOWLISTED",
	) {
		return fail("remote cache action is not explicitly allowed")
	}
	if !validVersionRange(
		document.Policy.LauncherVersionRange,
	) || !validVersionRange(
		document.Policy.PluginVersionRange,
	) {
		return fail("invalid component version range")
	}
	if !validConfigurationPolicy(document.Policy.ConfigurationCache) ||
		!validResourceProfile(document.Policy.ResourceProfile) ||
		!validBudgets(document.Policy.Budgets) ||
		!validExportProfile(document.Policy.ExportProfile) {
		return fail("invalid optimization policy selection")
	}
	if len(document.Policy.QualifiedTasks) == 0 ||
		len(document.Policy.QualifiedTasks) > maximumPolicyTasks ||
		!containsQualifiedContract(
			document.Policy.QualifiedTasks,
			document.CacheContractDigest,
		) {
		return fail("cache contract is not positively qualified")
	}
	if document.Policy.TestOptimizationGrant != nil {
		if !validSHA256(document.Policy.TestOptimizationGrant.Digest) {
			return fail("invalid Test Optimization grant digest")
		}
		if _, err := parseTimestamp(
			document.Policy.TestOptimizationGrant.ExpiresAt,
		); err != nil {
			return fail("invalid Test Optimization grant expiry")
		}
	}

	issuedAt, err := parseTimestamp(document.Policy.IssuedAt)
	if err != nil {
		return fail("invalid policy issuedAt")
	}
	expiresAt, err := parseTimestamp(document.Policy.ExpiresAt)
	if err != nil ||
		issuedAt.After(now) ||
		!expiresAt.After(now) ||
		!expiresAt.After(issuedAt) ||
		expiresAt.Sub(issuedAt) > maximumPolicyLifetime {
		return fail("policy is not currently valid")
	}
	revValidUntil, err := parseTimestamp(document.Revocation.ValidUntil)
	if err != nil || revValidUntil.Before(expiresAt) {
		return fail("revocation state does not cover policy lifetime")
	}
	leaseExpiresAt, err := parseTimestamp(document.Attempt.LeaseExpiresAt)
	if err != nil ||
		!leaseExpiresAt.After(now) ||
		leaseExpiresAt.After(expiresAt) ||
		leaseExpiresAt.Sub(now) > maximumAttemptLifetime {
		return fail("attempt lease is outside policy lifetime")
	}
	if document.Policy.TestOptimizationGrant != nil {
		grantExpiry, _ := parseTimestamp(
			document.Policy.TestOptimizationGrant.ExpiresAt,
		)
		if grantExpiry.Before(expiresAt) {
			return fail("Test Optimization grant does not cover policy")
		}
	}
	return issuedAt, expiresAt, revValidUntil, leaseExpiresAt, nil
}

func containsUniqueAllowedAction(actions []string, required string) bool {
	allowed := map[string]struct{}{
		"REMOTE_CACHE_ALLOWLISTED": {},
		"CONFIGURATION_CACHE":      {},
		"RESOURCE_PROFILE":         {},
		"BUILD_IMPACT":             {},
		"PATCH_CANDIDATE":          {},
	}
	if len(actions) > 16 {
		return false
	}
	seen := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		if _, ok := allowed[action]; !ok {
			return false
		}
		if _, duplicate := seen[action]; duplicate {
			return false
		}
		seen[action] = struct{}{}
	}
	_, ok := seen[required]
	return ok
}

func containsQualifiedContract(
	tasks []QualifiedTask,
	requiredDigest string,
) bool {
	matched := false
	for _, task := range tasks {
		if !validSHA256(task.ImplementationHash) ||
			!identifierPattern.MatchString(task.ContractRef) ||
			!validSHA256(task.CacheContractDigest) {
			return false
		}
		switch task.QualificationSource {
		case "OFFICIAL", "REVIEWED_ADAPTER",
			"HERMETIC_PRODUCER_PROFILE", "SOURCE_PATCH":
		default:
			return false
		}
		switch task.QualificationState {
		case "CONTRACT_QUALIFIED":
		case "QUARANTINE_VALIDATED":
			if task.RepeatabilityGate != "PASSED" ||
				task.RelocatabilityGate != "PASSED" {
				return false
			}
		case "UNKNOWN", "OBSERVING", "REJECTED", "SUSPENDED":
		default:
			return false
		}
		if !validGate(task.RepeatabilityGate) ||
			!validGate(task.RelocatabilityGate) {
			return false
		}
		if task.CacheContractDigest == requiredDigest &&
			(task.QualificationState == "CONTRACT_QUALIFIED" ||
				task.QualificationState == "QUARANTINE_VALIDATED") {
			matched = true
		}
	}
	return matched
}

func validGate(value string) bool {
	switch value {
	case "PASSED", "FAILED", "INCONCLUSIVE", "NOT_EVALUATED":
		return true
	default:
		return false
	}
}

func validConfigurationPolicy(policy ConfigurationCachePolicy) bool {
	return identifierPattern.MatchString(policy.ContractVersion)
}

func validResourceProfile(profile ResourceProfileReference) bool {
	return identifierPattern.MatchString(profile.ProfileID) &&
		validSHA256(profile.ProfileDigest) &&
		identifierPattern.MatchString(profile.CatalogVersion)
}

func validBudgets(budgets PolicyBudgets) bool {
	return budgets.MaxSynchronousOverheadMs >= 0 &&
		budgets.MaxSynchronousOverheadMs <= 60000 &&
		budgets.MaxSynchronousOverheadRatio >= 0 &&
		budgets.MaxSynchronousOverheadRatio <= 0.02 &&
		budgets.MaxValidationRunnerMsPerDay >= 0 &&
		budgets.MaxValidationRunnerMsPerDay <= 86400000
}

func validExportProfile(value string) bool {
	switch value {
	case "NONE", "SUMMARY", "TASKS", "DIAGNOSTIC":
		return true
	default:
		return false
	}
}

func versionSatisfies(version string, expression string) bool {
	current, ok := parseVersion(version)
	if !ok {
		return false
	}
	if exact, found := strings.CutPrefix(expression, "="); found {
		expected, valid := parseVersion(exact)
		return valid && current == expected
	}
	lowerText, upperText, found := strings.Cut(expression, " <")
	if !found {
		return false
	}
	lowerText, found = strings.CutPrefix(lowerText, ">=")
	if !found {
		return false
	}
	lower, lowerOK := parseVersion(lowerText)
	upper, upperOK := parseVersion(upperText)
	return lowerOK && upperOK && compareVersion(current, lower) >= 0 &&
		compareVersion(current, upper) < 0
}

func validVersionRange(expression string) bool {
	if !versionRangePattern.MatchString(expression) {
		return false
	}
	if _, exact := strings.CutPrefix(expression, "="); exact {
		return true
	}
	lowerText, upperText, found := strings.Cut(expression, " <")
	if !found {
		return false
	}
	lowerText, found = strings.CutPrefix(lowerText, ">=")
	if !found {
		return false
	}
	lower, lowerOK := parseVersion(lowerText)
	upper, upperOK := parseVersion(upperText)
	return lowerOK && upperOK && compareVersion(lower, upper) < 0
}

type semanticVersion [3]uint64

func parseVersion(value string) (semanticVersion, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return semanticVersion{}, false
	}
	var parsed semanticVersion
	for index, part := range parts {
		number, err := strconv.ParseUint(part, 10, 31)
		if err != nil || strconv.FormatUint(number, 10) != part {
			return semanticVersion{}, false
		}
		parsed[index] = number
	}
	return parsed, true
}

func compareVersion(left semanticVersion, right semanticVersion) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

func validSHA256(value string) bool {
	return len(value) == 71 &&
		value[:7] == digestPrefix &&
		isLowerHex(value[7:])
}

func isLowerHex(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func parseTimestamp(value string) (time.Time, error) {
	if !contractcrypto.ValidUTCTimestamp(value) {
		return time.Time{}, errors.New("timestamp is not canonical UTC")
	}
	return time.Parse(time.RFC3339Nano, value)
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func decodeJSONObject(raw []byte) (map[string]any, error) {
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	return object, nil
}

func digestObject(
	object map[string]any,
	digestField string,
	authenticationField string,
) (string, error) {
	clone, err := cloneObject(object)
	if err != nil {
		return "", err
	}
	delete(clone, digestField)
	if authenticationField != "" {
		authentication, ok := clone[authenticationField].(map[string]any)
		if !ok {
			return "", errors.New("missing authentication object")
		}
		delete(authentication, "signature")
	}
	encoded, err := json.Marshal(clone)
	if err != nil {
		return "", err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return digestPrefix + hex.EncodeToString(digest[:]), nil
}

func digestRevocationObject(object map[string]any) (string, error) {
	clone, err := cloneObject(object)
	if err != nil {
		return "", err
	}
	delete(clone, "requestId")
	delete(clone, "cumulativeStateDigest")
	authentication, ok := clone["authentication"].(map[string]any)
	if !ok {
		return "", errors.New("missing authentication object")
	}
	delete(authentication, "signature")
	encoded, err := json.Marshal(clone)
	if err != nil {
		return "", err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return digestPrefix + hex.EncodeToString(digest[:]), nil
}

func cloneObject(object map[string]any) (map[string]any, error) {
	encoded, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return decodeJSONObject(encoded)
}

func credentialDigest(credential []byte) string {
	digest := sha256.Sum256(credential)
	return digestPrefix + hex.EncodeToString(digest[:])
}

func signaturePayload(contract string, keyID string, digest string) []byte {
	return []byte(contract + "\x00" + keyID + "\x00" + digest)
}

func verifySignature(
	publicKey ed25519.PublicKey,
	encoded string,
	payload []byte,
) bool {
	signature, err := base64.RawURLEncoding.DecodeString(encoded)
	return err == nil &&
		len(signature) == ed25519.SignatureSize &&
		ed25519.Verify(publicKey, payload, signature)
}

func reject(reason string) error {
	return fmt.Errorf("%w: %s", ErrRejected, reason)
}

func minTime(values ...time.Time) time.Time {
	minimum := values[0]
	for _, value := range values[1:] {
		if value.Before(minimum) {
			minimum = value
		}
	}
	return minimum
}
