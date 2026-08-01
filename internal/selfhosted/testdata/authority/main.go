package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

func main() {
	root := flag.String("root", "", "existing private output directory")
	namespaceGeneration := flag.Int64("namespace-generation", 12, "stable namespace generation")
	revocationEpoch := flag.Int64("revocation-epoch", 7, "revocation epoch")
	l1SecurityGeneration := flag.Int64("l1-security-generation", 9, "L1 security generation")
	attemptID := flag.String(
		"attempt-id",
		"22222222-2222-4222-8222-222222222222",
		"authority attempt UUID",
	)
	flag.Parse()
	if flag.NArg() != 0 || *root == "" || !filepath.IsAbs(*root) ||
		*namespaceGeneration < 1 || *revocationEpoch < 1 || *l1SecurityGeneration < 1 {
		fmt.Fprintln(os.Stderr, "invalid synthetic authority request")
		os.Exit(64)
	}
	info, err := os.Lstat(*root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintln(os.Stderr, "synthetic authority root must be a private real directory")
		os.Exit(1)
	}

	now := time.Now().UTC()
	credential := bytes.Repeat([]byte{0x5a}, localauthority.CredentialBytes)
	credentialHash := sha256.Sum256(credential)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	document := localauthority.Document{
		Repository: localauthority.RepositoryIdentity{
			Tenant:      "tenant-internal",
			Repository:  "tonyredondo/buildopt",
			TrustDomain: "private-beta",
		},
		SourceRevision:      strings.Repeat("a", 40),
		SourceStateDigest:   "hmac-sha256:" + strings.Repeat("1", 64),
		CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
		Attempt: localauthority.AuthorityAttempt{
			AttemptID:        *attemptID,
			OwnerID:          "protected-main",
			LeaseID:          "lease-self-hosted-authority-1",
			LeaseExpiresAt:   now.Add(45 * time.Minute).Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       true,
			CredentialDigest: fmt.Sprintf("sha256:%x", credentialHash),
		},
		Policy: localauthority.OptimizationPolicy{
			SchemaVersion:               "1.0",
			RecordType:                  "OPTIMIZATION_POLICY",
			PolicyID:                    "self-hosted-policy",
			PolicyVersion:               *revocationEpoch,
			ConfigurationPolicyDigest:   "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:             *revocationEpoch,
			L1SecurityGeneration:        *l1SecurityGeneration,
			GatewayConnectionGeneration: 3,
			IssuedAt:                    now.Add(-time.Minute).Format(time.RFC3339Nano),
			LauncherVersionRange:        ">=0.1.0 <0.3.0",
			PluginVersionRange:          ">=0.1.0 <0.3.0",
			Mode:                        "VERIFIED",
			AllowedActions:              []string{"REMOTE_CACHE_ALLOWLISTED"},
			RemoteCache: localauthority.RemoteCachePolicy{
				Read:                true,
				Write:               "TRUSTED_CI_ONLY",
				Namespace:           "stable",
				NamespaceGeneration: *namespaceGeneration,
			},
			ConfigurationCache: localauthority.ConfigurationCachePolicy{
				Enabled:         true,
				ContractVersion: "configuration-cache-v1",
			},
			ResourceProfile: localauthority.ResourceProfileReference{
				ProfileID:      "STABLE_CONTROL",
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
			AffectedBuild: localauthority.AffectedBuild{EnabledInCI: true},
			ExpiresAt:     now.Add(time.Hour).Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            fmt.Sprintf("self-hosted-revocation-%d", *revocationEpoch),
			TrustDomain:          "private-beta",
			RevocationEpoch:      *revocationEpoch,
			L1SecurityGeneration: *l1SecurityGeneration,
			ValidUntil:           now.Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	authority, err := localauthority.Sign(document, "deployment-key-1", privateKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	trustRoot, err := localauthority.EncodeTrustRoot(localauthority.TrustRoot{
		Keys: []localauthority.PublicKey{{
			KeyID:     "deployment-key-1",
			PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		}},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := localauthority.Verify(
		context.Background(),
		authority,
		map[string]ed25519.PublicKey{"deployment-key-1": publicKey},
		credential,
		now,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for path, content := range map[string][]byte{
		filepath.Join(*root, "authority.json"):  authority,
		filepath.Join(*root, "trust-root.json"): trustRoot,
		filepath.Join(*root, "credential"): []byte(
			base64.RawURLEncoding.EncodeToString(credential),
		),
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
