package stickydecision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const signatureDomain = "buildopt-sticky-decision/v1"

// SignDecision creates the canonical immutable decision document. The
// decision digest is over the unsigned canonical payload, which avoids a
// recursive self-hash while still binding every action field to the signature.
func SignDecision(decision Decision, privateKey ed25519.PrivateKey) ([]byte, string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, "", errors.New("sticky decision signing key has an invalid size")
	}
	if decision.Authentication.Algorithm == "" {
		decision.Authentication.Algorithm = "Ed25519"
	}
	if decision.Authentication.Algorithm != "Ed25519" || decision.Authentication.KeyID == "" {
		return nil, "", errors.New("sticky decision signing identity is incomplete")
	}
	decision.DecisionDigest = ""
	decision.Authentication.Signature = ""
	unsigned, _, err := CanonicalDocument(decision)
	if err != nil {
		return nil, "", fmt.Errorf("canonicalize unsigned decision: %w", err)
	}
	digest := digestBytes(unsigned)
	payload := []byte(signatureDomain + "\x00" + decision.Authentication.KeyID + "\x00" + digest)
	signature := ed25519.Sign(privateKey, payload)
	decision.DecisionDigest = digest
	decision.Authentication.Signature = base64.RawURLEncoding.EncodeToString(signature)
	canonicalRaw, finalDigest, err := CanonicalDocument(decision)
	if err != nil {
		return nil, "", fmt.Errorf("canonicalize signed decision: %w", err)
	}
	if _, err := DecodeDocument(canonicalRaw, time.Time{}); err != nil {
		return nil, "", err
	}
	return canonicalRaw, finalDigest, nil
}

// VerifiedDecision is the only representation accepted by an action selector.
// It carries copies of the canonical bytes and semantic decision.
type VerifiedDecision struct {
	decision Decision
	raw      []byte
	digest   string
}

// Decision returns a defensive copy of the verified semantic document.
func (verified VerifiedDecision) Decision() Decision {
	decision := verified.decision
	decision.EvidenceRefs = append([]string(nil), decision.EvidenceRefs...)
	return decision
}

// CanonicalDocument returns a defensive copy of the signed bytes.
func (verified VerifiedDecision) CanonicalDocument() []byte {
	return append([]byte(nil), verified.raw...)
}

// Digest returns the content digest of the complete signed document.
func (verified VerifiedDecision) Digest() string { return verified.digest }

// VerifyDecision validates canonical bytes, expiry, revocation and the exact
// Ed25519 key named by the decision. A cache key or object digest is never used
// as an alternative authorization path.
func VerifyDecision(
	ctx context.Context,
	raw []byte,
	publicKeys map[string]ed25519.PublicKey,
	currentRevocationEpoch int64,
	now time.Time,
) (VerifiedDecision, error) {
	if ctx == nil {
		return VerifiedDecision{}, fmt.Errorf("%w: nil context", ErrInvalidDocument)
	}
	if err := ctx.Err(); err != nil {
		return VerifiedDecision{}, err
	}
	document, err := DecodeDocument(raw, now)
	if err != nil {
		return VerifiedDecision{}, err
	}
	if document.Decision == nil {
		return VerifiedDecision{}, fmt.Errorf("%w: record is not a decision", ErrCrossPlane)
	}
	decision := *document.Decision
	if decision.Binding.RevocationEpoch != currentRevocationEpoch {
		return VerifiedDecision{}, ErrRevoked
	}
	publicKey, ok := publicKeys[decision.Authentication.KeyID]
	if !ok || len(publicKey) != ed25519.PublicKeySize {
		return VerifiedDecision{}, fmt.Errorf("%w: decision key is unknown", ErrInvalidDocument)
	}
	unsignedDecision := decision
	unsignedDecision.DecisionDigest = ""
	unsignedDecision.Authentication.Signature = ""
	unsigned, _, err := CanonicalDocument(unsignedDecision)
	if err != nil {
		return VerifiedDecision{}, fmt.Errorf("%w: canonicalize decision payload: %v", ErrInvalidDocument, err)
	}
	digest := digestBytes(unsigned)
	if decision.DecisionDigest != digest {
		return VerifiedDecision{}, fmt.Errorf("%w: decision digest mismatch", ErrInvalidDocument)
	}
	signature, err := base64.RawURLEncoding.DecodeString(decision.Authentication.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(
		publicKey,
		[]byte(signatureDomain+"\x00"+decision.Authentication.KeyID+"\x00"+digest),
		signature,
	) {
		return VerifiedDecision{}, fmt.Errorf("%w: decision signature is invalid", ErrInvalidDocument)
	}
	if !bytes.Equal(document.Raw, raw) {
		return VerifiedDecision{}, fmt.Errorf("%w: signed bytes are not canonical", ErrInvalidDocument)
	}
	return VerifiedDecision{decision: decision, raw: append([]byte(nil), raw...), digest: document.Digest}, nil
}

// UnsignedDecisionDigest computes the payload identity used by SignDecision.
// It is useful to prepare a decision preview without creating a signature.
func UnsignedDecisionDigest(decision Decision) (string, error) {
	decision.DecisionDigest = ""
	decision.Authentication.Signature = ""
	raw, _, err := CanonicalDocument(decision)
	if err != nil {
		return "", err
	}
	return digestBytes(raw), nil
}

// MarshalCanonical is a small convenience for callers creating any control
// record. It rejects values that cannot be represented as JSON.
func MarshalCanonical(value any) ([]byte, error) {
	if value == nil {
		return nil, errors.New("cannot canonicalize nil sticky decision value")
	}
	if _, err := json.Marshal(value); err != nil {
		return nil, err
	}
	raw, _, err := CanonicalDocument(value)
	return raw, err
}
