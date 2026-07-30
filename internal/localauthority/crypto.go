package localauthority

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

// ParseTrustRoot validates one canonical out-of-band public-key document.
func ParseTrustRoot(raw []byte) (map[string]ed25519.PublicKey, error) {
	if len(raw) == 0 || len(raw) > maximumTrustRootBytes {
		return nil, errors.New("invalid local trust-root size")
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return nil, errors.New("local trust root is not canonical JCS")
	}
	var root TrustRoot
	if err := decodeStrict(canonical, &root); err != nil {
		return nil, fmt.Errorf("decode local trust root: %w", err)
	}
	if root.SchemaVersion != TrustRootContractVersion ||
		len(root.Keys) == 0 ||
		len(root.Keys) > 16 {
		return nil, errors.New("unsupported local trust root")
	}
	keys := make(map[string]ed25519.PublicKey, len(root.Keys))
	previous := ""
	for _, entry := range root.Keys {
		if !identifierPattern.MatchString(entry.KeyID) ||
			(previous != "" && entry.KeyID <= previous) {
			return nil, errors.New(
				"local trust-root keys are not strictly sorted",
			)
		}
		publicKey, err := base64.RawURLEncoding.DecodeString(
			entry.PublicKey,
		)
		if err != nil || len(publicKey) != ed25519.PublicKeySize {
			return nil, errors.New(
				"local trust root contains an invalid Ed25519 key",
			)
		}
		keys[entry.KeyID] = slices.Clone(publicKey)
		previous = entry.KeyID
	}
	return keys, nil
}

// ParseSigningKey validates one canonical private deployment-key document.
func ParseSigningKey(raw []byte) (string, ed25519.PrivateKey, error) {
	if len(raw) == 0 || len(raw) > maximumSigningKeyBytes {
		return "", nil, errors.New("invalid local signing-key size")
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return "", nil, errors.New(
			"local signing key is not canonical JCS",
		)
	}
	var key SigningKey
	if err := decodeStrict(canonical, &key); err != nil {
		return "", nil, fmt.Errorf("decode local signing key: %w", err)
	}
	if key.SchemaVersion != SigningKeyContractVersion ||
		!identifierPattern.MatchString(key.KeyID) {
		return "", nil, errors.New("unsupported local signing key")
	}
	privateKey, err := base64.RawURLEncoding.DecodeString(key.PrivateKey)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return "", nil, errors.New(
			"local signing key contains invalid Ed25519 bytes",
		)
	}
	return key.KeyID, slices.Clone(privateKey), nil
}

// ParseCredential decodes the exact one-line, URL-safe local cache secret.
func ParseCredential(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 128 {
		return nil, errors.New("invalid local cache credential size")
	}
	if raw[len(raw)-1] == '\n' {
		raw = raw[:len(raw)-1]
	}
	if len(raw) == 0 || bytes.ContainsAny(raw, "\r\n\t ") {
		return nil, errors.New("invalid local cache credential encoding")
	}
	credential, err := base64.RawURLEncoding.DecodeString(string(raw))
	if err != nil || len(credential) != CredentialBytes {
		return nil, errors.New("invalid local cache credential")
	}
	return credential, nil
}

// Sign creates canonical authority bytes for a pre-provisioned deployment
// key. It is used by the local policy producer and conformance fixtures, never
// by the launcher.
func Sign(
	document Document,
	keyID string,
	privateKey ed25519.PrivateKey,
) ([]byte, error) {
	if !identifierPattern.MatchString(keyID) ||
		len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid authority signing key")
	}
	document.SchemaVersion = AuthorityContractVersion
	document.Policy.SignatureKeyID = keyID
	document.Revocation.Authentication = Authentication{
		Algorithm: "Ed25519",
		KeyID:     keyID,
	}
	document.Authentication = Authentication{
		Algorithm: "Ed25519",
		KeyID:     keyID,
	}

	policyDigest, err := digestValue(
		document.Policy,
		"policyDigest",
		"",
	)
	if err != nil {
		return nil, err
	}
	document.Policy.PolicyDigest = policyDigest
	revocationEncoded, err := json.Marshal(document.Revocation)
	if err != nil {
		return nil, err
	}
	revocationObject, err := decodeJSONObject(revocationEncoded)
	if err != nil {
		return nil, err
	}
	revocationDigest, err := digestRevocationObject(revocationObject)
	if err != nil {
		return nil, err
	}
	document.Revocation.CumulativeStateDigest = revocationDigest
	document.Revocation.Authentication.Signature =
		base64.RawURLEncoding.EncodeToString(ed25519.Sign(
			privateKey,
			signaturePayload(
				"buildopt-cache-revocation/v1",
				keyID,
				revocationDigest,
			),
		))
	authorityDigest, err := digestValue(
		document,
		"authorityDigest",
		"authentication",
	)
	if err != nil {
		return nil, err
	}
	document.AuthorityDigest = authorityDigest
	document.Authentication.Signature =
		base64.RawURLEncoding.EncodeToString(ed25519.Sign(
			privateKey,
			signaturePayload(
				AuthorityContractVersion,
				keyID,
				authorityDigest,
			),
		))
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	return contractcrypto.CanonicalizeJCS(encoded)
}

// EncodeTrustRoot returns canonical public configuration for one or more
// strictly sorted deployment keys.
func EncodeTrustRoot(root TrustRoot) ([]byte, error) {
	root.SchemaVersion = TrustRootContractVersion
	encoded, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return nil, err
	}
	if _, err := ParseTrustRoot(canonical); err != nil {
		return nil, err
	}
	return canonical, nil
}

// EncodeSigningKey returns the canonical private configuration document.
func EncodeSigningKey(
	keyID string,
	privateKey ed25519.PrivateKey,
) ([]byte, error) {
	document := SigningKey{
		SchemaVersion: SigningKeyContractVersion,
		KeyID:         keyID,
		PrivateKey: base64.RawURLEncoding.EncodeToString(
			privateKey,
		),
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return nil, err
	}
	if _, _, err := ParseSigningKey(canonical); err != nil {
		return nil, err
	}
	return canonical, nil
}

func digestValue(
	value any,
	digestField string,
	authenticationField string,
) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	object, err := decodeJSONObject(encoded)
	if err != nil {
		return "", err
	}
	return digestObject(object, digestField, authenticationField)
}
