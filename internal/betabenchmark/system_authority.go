package betabenchmark

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const systemAuthorityKeyID = "beta-system-fault-v1"

type systemAuthorityFixture struct {
	verified       localauthority.Verified
	document       localauthority.Document
	credential     []byte
	privateKey     ed25519.PrivateKey
	authorityPath  string
	trustRootPath  string
	credentialPath string
}

func newSystemAuthorityFixture(
	ctx context.Context,
	root string,
	manifestDigest string,
	now time.Time,
	ordinal int,
	allowWrite bool,
	withGrant bool,
) (systemAuthorityFixture, error) {
	privateKey := benchmarkSigningKey(manifestDigest)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	credentialSeed := sha256.Sum256([]byte(fmt.Sprintf(
		"buildopt-beta-system-fault-credential/v1\x00%s\x00%d",
		manifestDigest,
		ordinal,
	)))
	credential := credentialSeed[:]
	credentialDigest := sha256.Sum256(credential)
	policyExpiresAt := now.UTC().Add(time.Hour)
	document := localauthority.Document{
		Repository: localauthority.RepositoryIdentity{
			Tenant:      "beta-smoke",
			Repository:  "buildopt",
			TrustDomain: "local-benchmark",
		},
		SourceRevision: strings.TrimPrefix(
			manifestDigest,
			"sha256:",
		),
		SourceStateDigest:   "hmac-sha256:" + strings.Repeat("1", 64),
		CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
		Attempt: localauthority.AuthorityAttempt{
			AttemptID: fmt.Sprintf(
				"11111111-1111-4111-8111-%012d",
				ordinal,
			),
			OwnerID: "protected-main",
			LeaseID: fmt.Sprintf("beta-system-lease-%d", ordinal),
			LeaseExpiresAt: now.UTC().Add(30 * time.Minute).
				Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       allowWrite,
			CredentialDigest: "sha256:" + hex.EncodeToString(credentialDigest[:]),
		},
		Policy: localauthority.OptimizationPolicy{
			SchemaVersion:               "1.0",
			RecordType:                  "OPTIMIZATION_POLICY",
			PolicyID:                    "beta-system-policy",
			PolicyVersion:               7,
			ConfigurationPolicyDigest:   "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:             7,
			L1SecurityGeneration:        9,
			GatewayConnectionGeneration: 3,
			IssuedAt: now.UTC().Add(-time.Minute).
				Format(time.RFC3339Nano),
			LauncherVersionRange: ">=0.1.0 <0.2.0",
			PluginVersionRange:   ">=0.1.0 <0.2.0",
			Mode:                 "VERIFIED",
			AllowedActions:       []string{"REMOTE_CACHE_ALLOWLISTED"},
			RemoteCache: localauthority.RemoteCachePolicy{
				Read:                true,
				Write:               "DISABLED",
				Namespace:           "stable",
				NamespaceGeneration: 12,
			},
			ConfigurationCache: localauthority.ConfigurationCachePolicy{
				Enabled:         true,
				ContractVersion: "configuration-cache-v1",
			},
			ResourceProfile: localauthority.ResourceProfileReference{
				ProfileID:      "W4_H6G",
				ProfileDigest:  "sha256:" + strings.Repeat("4", 64),
				CatalogVersion: "resource-catalog-v1",
			},
			Budgets: localauthority.PolicyBudgets{
				MaxSynchronousOverheadMs:    500,
				MaxSynchronousOverheadRatio: 0.02,
				MaxValidationRunnerMsPerDay: 60000,
			},
			ExportProfile: "SUMMARY",
			QualifiedTasks: []localauthority.QualifiedTask{{
				ImplementationHash:  "sha256:" + strings.Repeat("5", 64),
				QualificationSource: "OFFICIAL",
				ContractRef:         "java-compile-v1",
				CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
				QualificationState:  "CONTRACT_QUALIFIED",
				RepeatabilityGate:   "PASSED",
				RelocatabilityGate:  "PASSED",
			}},
			AffectedBuild: localauthority.AffectedBuild{
				EnabledInCI:    allowWrite,
				EnabledLocally: !allowWrite,
			},
			ExpiresAt: policyExpiresAt.Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            fmt.Sprintf("beta-revocation-%d", ordinal),
			TrustDomain:          "local-benchmark",
			RevocationEpoch:      7,
			L1SecurityGeneration: 9,
			ValidUntil: now.UTC().Add(2 * time.Hour).
				Format(time.RFC3339Nano),
		},
	}
	if allowWrite {
		document.Policy.RemoteCache.Write = "TRUSTED_CI_ONLY"
	}
	if withGrant {
		document.Policy.TestOptimizationGrant = &localauthority.GrantReference{
			Digest:    "sha256:" + strings.Repeat("6", 64),
			ExpiresAt: now.UTC().Add(2 * time.Hour).Format(time.RFC3339Nano),
		}
	}
	canonical, err := localauthority.Sign(
		document,
		systemAuthorityKeyID,
		privateKey,
	)
	if err != nil {
		return systemAuthorityFixture{}, err
	}
	keys := map[string]ed25519.PublicKey{systemAuthorityKeyID: publicKey}
	verified, err := localauthority.Verify(
		ctx,
		canonical,
		keys,
		credential,
		now.UTC(),
	)
	if err != nil {
		return systemAuthorityFixture{}, err
	}
	trustRoot, err := localauthority.EncodeTrustRoot(
		localauthority.TrustRoot{
			Keys: []localauthority.PublicKey{{
				KeyID: systemAuthorityKeyID,
				PublicKey: base64.RawURLEncoding.EncodeToString(
					publicKey,
				),
			}},
		},
	)
	if err != nil {
		return systemAuthorityFixture{}, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return systemAuthorityFixture{}, err
	}
	authorityPath := filepath.Join(root, "authority.json")
	trustRootPath := filepath.Join(root, "trust-root.json")
	credentialPath := filepath.Join(root, "credential")
	for _, file := range []struct {
		path    string
		content []byte
	}{
		{path: authorityPath, content: canonical},
		{path: trustRootPath, content: trustRoot},
		{
			path: credentialPath,
			content: []byte(
				base64.RawURLEncoding.EncodeToString(credential),
			),
		},
	} {
		if err := os.WriteFile(file.path, file.content, 0o600); err != nil {
			return systemAuthorityFixture{}, err
		}
	}
	return systemAuthorityFixture{
		verified:       verified,
		document:       verified.Document(),
		credential:     credential,
		privateKey:     privateKey,
		authorityPath:  authorityPath,
		trustRootPath:  trustRootPath,
		credentialPath: credentialPath,
	}, nil
}

func advanceSystemAuthorityFixture(
	ctx context.Context,
	current systemAuthorityFixture,
	now time.Time,
	revokeGrant bool,
) (systemAuthorityFixture, error) {
	document := current.verified.Document()
	document.AuthorityDigest = ""
	document.Authentication = localauthority.Authentication{}
	document.Attempt.AttemptID = "22222222-2222-4222-8222-" +
		document.Attempt.AttemptID[len(document.Attempt.AttemptID)-12:]
	document.Attempt.LeaseID += "-next"
	document.Attempt.LeaseExpiresAt = now.UTC().Add(30 * time.Minute).
		Format(time.RFC3339Nano)
	document.Attempt.AllowWrite = false
	document.Policy.PolicyVersion++
	document.Policy.ConfigurationPolicyDigest =
		"sha256:" + strings.Repeat("7", 64)
	document.Policy.RevocationEpoch++
	document.Policy.L1SecurityGeneration++
	document.Policy.GatewayConnectionGeneration++
	document.Policy.IssuedAt = now.UTC().Format(time.RFC3339Nano)
	document.Policy.RemoteCache.Write = "DISABLED"
	document.Policy.RemoteCache.Namespace = "stable-revoked"
	document.Policy.RemoteCache.NamespaceGeneration++
	document.Policy.AffectedBuild.EnabledInCI = false
	document.Policy.AffectedBuild.EnabledLocally = true
	document.Revocation.RequestID += "-next"
	document.Revocation.RevocationEpoch++
	document.Revocation.L1SecurityGeneration++
	document.Revocation.CumulativeStateDigest = ""
	document.Revocation.Authentication = localauthority.Authentication{}
	if revokeGrant {
		if document.Policy.TestOptimizationGrant == nil {
			return systemAuthorityFixture{}, fmt.Errorf(
				"advance system grant authority: current grant is absent",
			)
		}
		document.Policy.TestOptimizationGrant.Digest =
			"sha256:" + strings.Repeat("8", 64)
	}
	canonical, err := localauthority.Sign(
		document,
		systemAuthorityKeyID,
		current.privateKey,
	)
	if err != nil {
		return systemAuthorityFixture{}, err
	}
	publicKey := current.privateKey.Public().(ed25519.PublicKey)
	verified, err := localauthority.Verify(
		ctx,
		canonical,
		map[string]ed25519.PublicKey{systemAuthorityKeyID: publicKey},
		current.credential,
		now.UTC(),
	)
	if err != nil {
		return systemAuthorityFixture{}, err
	}
	result := current
	result.verified = verified
	result.document = verified.Document()
	return result, nil
}
