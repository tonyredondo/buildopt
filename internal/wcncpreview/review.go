// Package wcncpreview owns WCNCP-006 review delivery: digest-bound human and
// machine artifacts over exact proposal and validation digests, with a
// lifecycle derived from immutable owner decisions. Acceptance changes only
// control-plane state; it never applies source, commits, pushes, opens a PR,
// merges, or contacts upstream. Historical recipes traverse this lane only as
// HISTORICAL_RECIPE_SYSTEM_FIXTURE on isolated copies with zero prospective
// value counting.
package wcncpreview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

const (
	// SystemFixtureLabel marks diagnostic-only historical recipe replays.
	SystemFixtureLabel = "HISTORICAL_RECIPE_SYSTEM_FIXTURE"
	// NoAutoMutationStatement is embedded in every artifact so acceptance can
	// never be mistaken for deployment.
	NoAutoMutationStatement = "No automatic mutation, commit, push, pull request, merge, or upstream contact occurred or is authorized by this artifact."
)

var (
	// ErrStale means the proposal drifted or was superseded before review.
	ErrStale = errors.New("BuildOpt WCNCP proposal is stale")
	// ErrTampered means patch bytes do not match the proposal binding.
	ErrTampered = errors.New("BuildOpt WCNCP patch is tampered")
	// ErrDecision means the owner decision is malformed or unauthorized.
	ErrDecision = errors.New("BuildOpt WCNCP owner decision is invalid")
)

// PatchBundleVerifier is the mandatory boundary to the canonical signed
// PatchBundle verifier. Draft never accepts an envelope based on shape alone.
type PatchBundleVerifier func(signedBundle []byte) error

// Artifact is the first-exposure review bundle. Timing fields stay pending
// until value qualification; only value-qualified proposals reach real owner
// review in WCNCP-012.
type Artifact struct {
	SchemaVersion       string `json:"schemaVersion"`
	ProposalSHA256      string `json:"proposalSha256"`
	ValidationSHA256    string `json:"validationSha256"`
	PatchSHA256         string `json:"patchSha256"`
	InversePatchSHA256  string `json:"inversePatchSha256"`
	Lane                string `json:"lane"`
	Problem             string `json:"problem"`
	ExpectedImprovement string `json:"expectedImprovement"`
	Diff                string `json:"diff"`
	DeclarationLocation string `json:"declarationLocation"`
	SafetyRationale     string `json:"safetyRationale"`
	CorrectnessEvidence string `json:"correctnessEvidence"`
	WallTimeTable       string `json:"wallTimeTable"`
	IntervalP95Payback  string `json:"intervalP95Payback"`
	Limitations         string `json:"limitations"`
	ApplyCommand        string `json:"applyCommand"`
	RevertCommand       string `json:"revertCommand"`
	AuthorityStatement  string `json:"authorityStatement"`
	PatchBundleContract string `json:"patchBundleContract"`
	SignatureBoundary   string `json:"signatureBoundary"`
}

// Draft builds a review artifact bound to exact proposal, validation, patch,
// and inverse digests. Empty safety, evidence, or command fields fail closed;
// historical replays must pass the system-fixture lane explicitly.
func Draft(proposalSHA256, validationSHA256 string, patch, inverse []byte, lane string, fields map[string]string, verify PatchBundleVerifier) (Artifact, string, error) {
	if !validDigest(proposalSHA256) || !validDigest(validationSHA256) || len(patch) == 0 || len(inverse) == 0 {
		return Artifact{}, "", ErrTampered
	}
	if verify == nil || !signedPatchBundleShape(patch) || !signedPatchBundleShape(inverse) || verify(patch) != nil || verify(inverse) != nil {
		return Artifact{}, "", ErrTampered
	}
	if lane != SystemFixtureLabel && lane != "PROSPECTIVE" {
		return Artifact{}, "", ErrDecision
	}
	required := []string{"problem", "expectedImprovement", "diff", "declarationLocation", "safetyRationale", "correctnessEvidence", "limitations", "applyCommand", "revertCommand"}
	for _, key := range required {
		if strings.TrimSpace(fields[key]) == "" {
			return Artifact{}, "", ErrDecision
		}
	}
	patchDigest := sha256.Sum256(patch)
	inverseDigest := sha256.Sum256(inverse)
	artifact := Artifact{
		SchemaVersion: "buildopt.wcncp/review-artifact/v1", ProposalSHA256: proposalSHA256,
		ValidationSHA256: validationSHA256, PatchSHA256: hex.EncodeToString(patchDigest[:]),
		InversePatchSHA256: hex.EncodeToString(inverseDigest[:]), Lane: lane,
		Problem: fields["problem"], ExpectedImprovement: fields["expectedImprovement"],
		Diff: fields["diff"], DeclarationLocation: fields["declarationLocation"],
		SafetyRationale: fields["safetyRationale"], CorrectnessEvidence: fields["correctnessEvidence"],
		WallTimeTable: "NOT_RUN_FIXTURE", IntervalP95Payback: "NOT_EVALUATED_STANDARD_CI",
		Limitations: fields["limitations"], ApplyCommand: fields["applyCommand"],
		RevertCommand: fields["revertCommand"], AuthorityStatement: NoAutoMutationStatement,
		PatchBundleContract: "buildopt-patch-bundle/v1",
		SignatureBoundary:   "verified through the mandatory PatchBundleVerifier boundary before review binding",
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		return Artifact{}, "", err
	}
	digest := sha256.Sum256(canonicalJSON(raw))
	return artifact, hex.EncodeToString(digest[:]), nil
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func signedPatchBundleShape(raw []byte) bool {
	var envelope struct {
		ContractVersion string `json:"contractVersion"`
		BundleDigest    string `json:"bundleDigest"`
		Signature       struct {
			Algorithm          string `json:"algorithm"`
			Canonicalization   string `json:"canonicalization"`
			KeyID              string `json:"keyId"`
			SignedBundleDigest string `json:"signedBundleDigest"`
			Value              string `json:"value"`
		} `json:"signature"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false
	}
	return envelope.ContractVersion == "buildopt-patch-bundle/v1" &&
		strings.HasPrefix(envelope.BundleDigest, "sha256:") && validDigest(strings.TrimPrefix(envelope.BundleDigest, "sha256:")) &&
		envelope.Signature.Algorithm == "Ed25519" && envelope.Signature.Canonicalization == "JCS" &&
		envelope.Signature.KeyID != "" && envelope.Signature.SignedBundleDigest == envelope.BundleDigest && len(envelope.Signature.Value) == 86
}

func canonicalJSON(raw []byte) []byte {
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	// Deterministic key order is sufficient for fixture identity; durable
	// control-plane identity still uses JCS via sharedcache.
	normalized, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return normalized
}

// VerifyPatchBinding rejects tampered patch bytes before review.
func VerifyPatchBinding(expectedPatchSHA256 string, patch []byte) error {
	digest := sha256.Sum256(patch)
	if hex.EncodeToString(digest[:]) != expectedPatchSHA256 {
		return ErrTampered
	}
	return nil
}

// Lifecycle derives review status from immutable decisions and drift. Owner
// acceptance never implies SOURCE_APPLIED or merged.
func Lifecycle(decisions []string, drifted bool) string {
	state, err := ResolveLifecycle(decisions, drifted)
	if err != nil {
		return "INVALID_DECISION"
	}
	return state
}

// ResolveLifecycle rejects contradictory or repeated owner decisions instead
// of selecting whichever happened to appear first.
func ResolveLifecycle(decisions []string, drifted bool) (string, error) {
	if drifted {
		return "STALE", nil
	}
	if len(decisions) > 1 {
		return "", ErrDecision
	}
	for _, decision := range decisions {
		switch decision {
		case "ACCEPT":
			return "OWNER_ACCEPTED", nil
		case "REJECT":
			return "OWNER_REJECTED", nil
		case "DEFER":
			return "OWNER_DEFERRED", nil
		default:
			return "", ErrDecision
		}
	}
	return "REVIEW_READY", nil
}
