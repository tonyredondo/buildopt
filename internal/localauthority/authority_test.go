package localauthority

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

var authorityTestNow = time.Date(
	2026,
	time.July,
	30,
	12,
	0,
	0,
	0,
	time.UTC,
)

func TestAuthoritySignVerifyAndComponentCompatibility(t *testing.T) {
	document, credential, privateKey, publicKey := authorityTestFixture()
	canonical, err := Sign(document, "deployment-key-1", privateKey)
	if err != nil {
		t.Fatalf("sign authority: %v", err)
	}
	verified, err := Verify(
		context.Background(),
		canonical,
		map[string]ed25519.PublicKey{"deployment-key-1": publicKey},
		credential,
		authorityTestNow,
	)
	if err != nil {
		t.Fatalf("verify authority: %v", err)
	}
	actual := verified.Document()
	if actual.AuthorityDigest == "" ||
		actual.Policy.PolicyDigest == "" ||
		actual.Revocation.CumulativeStateDigest == "" ||
		actual.Attempt.CredentialDigest != credentialDigest(credential) {
		t.Fatalf("verified digests are incomplete: %+v", actual)
	}
	if !bytes.Equal(canonical, verified.CanonicalDocument()) {
		t.Fatal("verified canonical bytes changed")
	}
	if !verified.ExpiresAt().Equal(
		authorityTestNow.Add(45 * time.Minute),
	) {
		t.Fatalf("effective expiry = %s", verified.ExpiresAt())
	}
	if !verified.IssuedAt().Equal(authorityTestNow.Add(-time.Minute)) {
		t.Fatalf("issuedAt = %s", verified.IssuedAt())
	}
	if !verified.SupportsComponents("0.1.0", "0.1.99") ||
		verified.SupportsComponents("0.2.0", "0.1.0") ||
		verified.SupportsComponents("0.1.0-SNAPSHOT", "0.1.0") {
		t.Fatal("component range evaluation was not exact")
	}

	actual.Policy.AllowedActions[0] = "MUTATED"
	actual.Policy.QualifiedTasks[0].ContractRef = "mutated"
	if verified.Document().Policy.AllowedActions[0] == "MUTATED" ||
		verified.Document().Policy.QualifiedTasks[0].ContractRef == "mutated" {
		t.Fatal("verified authority exposed mutable slices")
	}
	mutatedBytes := verified.CanonicalDocument()
	mutatedBytes[0] ^= 1
	if bytes.Equal(mutatedBytes, verified.CanonicalDocument()) {
		t.Fatal("verified authority exposed mutable canonical bytes")
	}
}

func TestAuthorityRejectsUntrustedOrInvalidDocuments(t *testing.T) {
	document, credential, privateKey, publicKey := authorityTestFixture()
	canonical, err := Sign(document, "deployment-key-1", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"deployment-key-1": publicKey}

	unknownObject := decodeAuthorityTestObject(t, canonical)
	unknownObject["unexpected"] = true
	unknownCanonical := canonicalAuthorityTestObject(t, unknownObject)

	tamperedObject := decodeAuthorityTestObject(t, canonical)
	tamperedObject["sourceRevision"] = strings.Repeat("b", 40)
	tamperedCanonical := canonicalAuthorityTestObject(t, tamperedObject)

	expiredDocument := document
	expiredDocument.Policy.IssuedAt = authorityTestNow.
		Add(-2 * time.Hour).
		Format(time.RFC3339Nano)
	expiredDocument.Policy.ExpiresAt = authorityTestNow.
		Add(-time.Hour).
		Format(time.RFC3339Nano)
	expiredDocument.Attempt.LeaseExpiresAt = authorityTestNow.
		Add(-90 * time.Minute).
		Format(time.RFC3339Nano)
	expiredDocument.Revocation.ValidUntil = authorityTestNow.
		Add(time.Hour).
		Format(time.RFC3339Nano)
	expiredCanonical, err := Sign(
		expiredDocument,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	longPolicy := document
	longPolicy.Policy.ExpiresAt = authorityTestNow.
		Add(31 * 24 * time.Hour).
		Format(time.RFC3339Nano)
	longPolicy.Attempt.LeaseExpiresAt = authorityTestNow.
		Add(time.Hour).
		Format(time.RFC3339Nano)
	longPolicy.Revocation.ValidUntil = authorityTestNow.
		Add(32 * 24 * time.Hour).
		Format(time.RFC3339Nano)
	longCanonical, err := Sign(longPolicy, "deployment-key-1", privateKey)
	if err != nil {
		t.Fatal(err)
	}

	otherPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x77}, 32))
	otherPublic := otherPrivate.Public().(ed25519.PublicKey)
	testCases := []struct {
		name       string
		raw        []byte
		keys       map[string]ed25519.PublicKey
		credential []byte
	}{
		{
			name:       "noncanonical",
			raw:        append(bytes.Clone(canonical), '\n'),
			keys:       keys,
			credential: credential,
		},
		{
			name:       "unknown field",
			raw:        unknownCanonical,
			keys:       keys,
			credential: credential,
		},
		{
			name:       "tampered signed content",
			raw:        tamperedCanonical,
			keys:       keys,
			credential: credential,
		},
		{
			name:       "wrong credential",
			raw:        canonical,
			keys:       keys,
			credential: bytes.Repeat([]byte{0x44}, CredentialBytes),
		},
		{
			name:       "wrong trust root",
			raw:        canonical,
			keys:       map[string]ed25519.PublicKey{"deployment-key-1": otherPublic},
			credential: credential,
		},
		{
			name:       "expired",
			raw:        expiredCanonical,
			keys:       keys,
			credential: credential,
		},
		{
			name:       "policy exceeds beta lifetime",
			raw:        longCanonical,
			keys:       keys,
			credential: credential,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := Verify(
				context.Background(),
				testCase.raw,
				testCase.keys,
				testCase.credential,
				authorityTestNow,
			); !errors.Is(err, ErrRejected) {
				t.Fatalf("rejection = %v, want ErrRejected", err)
			}
		})
	}
}

func TestTrustRootSigningKeyAndCredentialDocuments(t *testing.T) {
	_, credential, privateKey, publicKey := authorityTestFixture()
	root, err := EncodeTrustRoot(TrustRoot{
		Keys: []PublicKey{{
			KeyID: "deployment-key-1",
			PublicKey: base64.RawURLEncoding.EncodeToString(
				publicKey,
			),
		}},
	})
	if err != nil {
		t.Fatalf("encode trust root: %v", err)
	}
	keys, err := ParseTrustRoot(root)
	if err != nil || !bytes.Equal(keys["deployment-key-1"], publicKey) {
		t.Fatalf("parse trust root = %v/%v", keys, err)
	}
	signing, err := EncodeSigningKey("deployment-key-1", privateKey)
	if err != nil {
		t.Fatalf("encode signing key: %v", err)
	}
	keyID, parsedPrivate, err := ParseSigningKey(signing)
	if err != nil ||
		keyID != "deployment-key-1" ||
		!bytes.Equal(parsedPrivate, privateKey) {
		t.Fatalf("parse signing key = %q/%v/%v", keyID, parsedPrivate, err)
	}
	encodedCredential := []byte(
		base64.RawURLEncoding.EncodeToString(credential) + "\n",
	)
	parsedCredential, err := ParseCredential(encodedCredential)
	if err != nil || !bytes.Equal(parsedCredential, credential) {
		t.Fatalf("parse credential = %v/%v", parsedCredential, err)
	}

	unsorted, err := json.Marshal(TrustRoot{
		SchemaVersion: TrustRootContractVersion,
		Keys: []PublicKey{
			{
				KeyID:     "z-key",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
			{
				KeyID:     "a-key",
				PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	unsorted, err = contractcrypto.CanonicalizeJCS(unsorted)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTrustRoot(unsorted); err == nil {
		t.Fatal("unsorted trust root was accepted")
	}
}

func TestStateAdvanceRejectsRollbackAndGenerationRebinding(t *testing.T) {
	document, credential, privateKey, publicKey := authorityTestFixture()
	canonical, err := Sign(document, "deployment-key-1", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(
		context.Background(),
		canonical,
		map[string]ed25519.PublicKey{"deployment-key-1": publicKey},
		credential,
		authorityTestNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	current := StateFromVerified(verified, authorityTestNow)
	if err := Advance(State{}, current); err != nil {
		t.Fatalf("initial state: %v", err)
	}
	if err := Advance(current, current); err != nil {
		t.Fatalf("exact replay: %v", err)
	}

	testCases := []struct {
		name   string
		mutate func(*State)
	}{
		{
			name: "policy rollback",
			mutate: func(state *State) {
				state.PolicyVersion--
			},
		},
		{
			name: "policy version reuse",
			mutate: func(state *State) {
				state.PolicyDigest = "sha256:" + strings.Repeat("f", 64)
			},
		},
		{
			name: "revocation rollback",
			mutate: func(state *State) {
				state.RevocationEpoch--
			},
		},
		{
			name: "revocation reuse",
			mutate: func(state *State) {
				state.RevocationDigest = "sha256:" + strings.Repeat("e", 64)
			},
		},
		{
			name: "revocation without L1 rotation",
			mutate: func(state *State) {
				state.RevocationEpoch++
				state.RevocationDigest = "sha256:" + strings.Repeat("d", 64)
			},
		},
		{
			name: "gateway rollback",
			mutate: func(state *State) {
				state.GatewayConnectionGeneration--
			},
		},
		{
			name: "namespace rollback",
			mutate: func(state *State) {
				state.NamespaceGeneration--
			},
		},
		{
			name: "namespace generation reuse",
			mutate: func(state *State) {
				state.Namespace = "stable-rebound"
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			next := current
			testCase.mutate(&next)
			if err := Advance(current, next); !errors.Is(err, ErrRollback) {
				t.Fatalf("advance = %v, want ErrRollback", err)
			}
		})
	}

	advanced := current
	advanced.PolicyVersion++
	advanced.PolicyDigest = "sha256:" + strings.Repeat("c", 64)
	advanced.RevocationEpoch++
	advanced.RevocationDigest = "sha256:" + strings.Repeat("b", 64)
	advanced.L1SecurityGeneration++
	advanced.GatewayConnectionGeneration++
	advanced.NamespaceGeneration++
	advanced.Namespace = "stable-v2"
	if err := Advance(current, advanced); err != nil {
		t.Fatalf("valid monotonic advance: %v", err)
	}
}

func authorityTestFixture() (
	Document,
	[]byte,
	ed25519.PrivateKey,
	ed25519.PublicKey,
) {
	credential := bytes.Repeat([]byte{0x5a}, CredentialBytes)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, 32))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	document := Document{
		Repository: RepositoryIdentity{
			Tenant:      "tenant-internal",
			Repository:  "tonyredondo/buildopt",
			TrustDomain: "private-beta",
		},
		SourceRevision:      strings.Repeat("a", 40),
		SourceStateDigest:   "hmac-sha256:" + strings.Repeat("1", 64),
		CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
		Attempt: AuthorityAttempt{
			AttemptID:        "11111111-1111-4111-8111-111111111111",
			OwnerID:          "protected-main",
			LeaseID:          "lease-authority-1",
			LeaseExpiresAt:   authorityTestNow.Add(45 * time.Minute).Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       true,
			CredentialDigest: credentialDigest(credential),
		},
		Policy: OptimizationPolicy{
			SchemaVersion:               "1.0",
			RecordType:                  "OPTIMIZATION_POLICY",
			PolicyID:                    "internal-policy",
			PolicyVersion:               7,
			ConfigurationPolicyDigest:   "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:             7,
			L1SecurityGeneration:        9,
			GatewayConnectionGeneration: 3,
			IssuedAt:                    authorityTestNow.Add(-time.Minute).Format(time.RFC3339Nano),
			LauncherVersionRange:        ">=0.1.0 <0.2.0",
			PluginVersionRange:          ">=0.1.0 <0.2.0",
			Mode:                        "VERIFIED",
			AllowedActions:              []string{"REMOTE_CACHE_ALLOWLISTED"},
			RemoteCache: RemoteCachePolicy{
				Read:                true,
				Write:               "TRUSTED_CI_ONLY",
				Namespace:           "stable",
				NamespaceGeneration: 12,
			},
			ConfigurationCache: ConfigurationCachePolicy{
				Enabled:         true,
				ContractVersion: "configuration-cache-v1",
			},
			ResourceProfile: ResourceProfileReference{
				ProfileID:      "W4_H6G",
				ProfileDigest:  "sha256:" + strings.Repeat("4", 64),
				CatalogVersion: "resource-catalog-v1",
			},
			Budgets: PolicyBudgets{
				MaxSynchronousOverheadMs:    500,
				MaxSynchronousOverheadRatio: 0.02,
				MaxValidationRunnerMsPerDay: 60000,
			},
			ExportProfile: "SUMMARY",
			QualifiedTasks: []QualifiedTask{{
				ImplementationHash:  "sha256:" + strings.Repeat("5", 64),
				QualificationSource: "OFFICIAL",
				ContractRef:         "java-compile-v1",
				CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
				QualificationState:  "CONTRACT_QUALIFIED",
				RepeatabilityGate:   "PASSED",
				RelocatabilityGate:  "PASSED",
			}},
			AffectedBuild: AffectedBuild{
				EnabledInCI:    true,
				EnabledLocally: false,
			},
			ExpiresAt: authorityTestNow.Add(time.Hour).Format(time.RFC3339Nano),
		},
		Revocation: RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            "revocation-request-7",
			TrustDomain:          "private-beta",
			RevocationEpoch:      7,
			L1SecurityGeneration: 9,
			ValidUntil:           authorityTestNow.Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	return document, credential, privateKey, publicKey
}

func decodeAuthorityTestObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		t.Fatal(err)
	}
	return object
}

func canonicalAuthorityTestObject(t *testing.T, object map[string]any) []byte {
	t.Helper()
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}
