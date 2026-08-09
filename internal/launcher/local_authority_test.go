package launcher

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

var launcherAuthorityNow = time.Date(
	2026,
	time.July,
	30,
	12,
	0,
	0,
	0,
	time.UTC,
)

func TestLauncherInstallsAuthenticatedCacheContext(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()
	environment, document := writeLauncherAuthorityFixture(
		t,
		upstream.URL,
	)
	remoteToken := bytes.Repeat([]byte{0x6b}, localauthority.CredentialBytes)
	remoteTokenPath := filepath.Join(
		filepath.Dir(environment[localCredentialPathEnvironment]),
		"remote-token",
	)
	if err := os.WriteFile(
		remoteTokenPath,
		[]byte(base64.RawURLEncoding.EncodeToString(remoteToken)),
		0o600,
	); err != nil {
		t.Fatalf("write remote token: %v", err)
	}
	environment[sharedCacheTokenPathEnvironment] = remoteTokenPath

	actual, configured, err := localAuthorityContextFromEnvironment(
		context.Background(),
		func(key string) string { return environment[key] },
		launcherAuthorityNow,
	)
	if err != nil || !configured || actual == nil {
		t.Fatalf(
			"load launcher authority = %+v/%t/%v",
			actual,
			configured,
			err,
		)
	}
	if actual.attemptID != document.Attempt.AttemptID ||
		actual.cacheBinding == nil ||
		actual.cacheBinding.authorityDigest == "" ||
		!actual.cacheBinding.allowWrite ||
		actual.managedL1Config.securityGeneration != 9 ||
		!actual.managedL1Config.l2WriteAuthorized ||
		actual.childEnvironment[managedSharedModeEnvironment] !=
			managedSharedReadWriteMode ||
		actual.childEnvironment[managedAuthorityContractEnvironment] !=
			localauthority.AuthorityContractVersion {
		t.Fatalf("unexpected launcher authority context: %+v", actual)
	}
	if actual.cacheBinding.credential != base64.RawURLEncoding.EncodeToString(remoteToken) {
		t.Fatal("gateway did not use the separate remote beta token")
	}
	if strings.Contains(
		strings.Join(mapValues(actual.childEnvironment), "\n"),
		actual.cacheBinding.credential,
	) {
		t.Fatal("remote cache credential escaped into child environment")
	}

	replayed, replayConfigured, replayErr :=
		localAuthorityContextFromEnvironment(
			context.Background(),
			func(key string) string { return environment[key] },
			launcherAuthorityNow,
		)
	if replayErr != nil || !replayConfigured ||
		replayed.cacheBinding.authorityDigest !=
			actual.cacheBinding.authorityDigest {
		t.Fatalf(
			"exact authority replay = %+v/%t/%v",
			replayed,
			replayConfigured,
			replayErr,
		)
	}
}

func TestLauncherAuthorityFailsClosed(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	environment, _ := writeLauncherAuthorityFixture(t, upstream.URL)

	tests := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{
			name: "incomplete files",
			mutate: func(environment map[string]string) {
				delete(environment, localTrustRootPathEnvironment)
			},
		},
		{
			name: "repository mismatch",
			mutate: func(environment map[string]string) {
				environment[managedL1RepositoryEnvironment] = "other/repository"
			},
		},
		{
			name: "write on local build",
			mutate: func(environment map[string]string) {
				environment["CI"] = ""
			},
		},
		{
			name: "insecure remote Shared endpoint",
			mutate: func(environment map[string]string) {
				environment[sharedCacheURLEnvironment] = "http://cache.example"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := make(map[string]string, len(environment))
			for key, value := range environment {
				candidate[key] = value
			}
			test.mutate(candidate)
			actual, configured, err :=
				localAuthorityContextFromEnvironment(
					context.Background(),
					func(key string) string { return candidate[key] },
					launcherAuthorityNow,
				)
			if err == nil || !configured || actual != nil {
				t.Fatalf(
					"invalid authority = %+v/%t/%v",
					actual,
					configured,
					err,
				)
			}
		})
	}

	actual, configured, err := localAuthorityContextFromEnvironment(
		context.Background(),
		func(string) string { return "" },
		launcherAuthorityNow,
	)
	if err != nil || configured || actual != nil {
		t.Fatalf("absent authority = %+v/%t/%v", actual, configured, err)
	}
}

func writeLauncherAuthorityFixture(
	t *testing.T,
	upstream string,
) (map[string]string, localauthority.Document) {
	return writeLauncherAuthorityFixtureAt(
		t,
		upstream,
		launcherAuthorityNow,
	)
}

func writeLauncherAuthorityFixtureAt(
	t *testing.T,
	upstream string,
	now time.Time,
) (map[string]string, localauthority.Document) {
	t.Helper()
	root := t.TempDir()
	stateRoot := filepath.Join(root, "state")
	credential := bytes.Repeat([]byte{0x5a}, localauthority.CredentialBytes)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, 32))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	credentialHash := sha256.Sum256(credential)
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
			AttemptID: "11111111-1111-4111-8111-111111111111",
			OwnerID:   "protected-main",
			LeaseID:   "lease-authority-1",
			LeaseExpiresAt: now.
				Add(45 * time.Minute).
				Format(time.RFC3339Nano),
			AllowRead:  true,
			AllowWrite: true,
			CredentialDigest: "sha256:" +
				hex.EncodeToString(credentialHash[:]),
		},
		Policy: localauthority.OptimizationPolicy{
			SchemaVersion:               "1.0",
			RecordType:                  "OPTIMIZATION_POLICY",
			PolicyID:                    "internal-policy",
			PolicyVersion:               7,
			ConfigurationPolicyDigest:   "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:             7,
			L1SecurityGeneration:        9,
			GatewayConnectionGeneration: 3,
			IssuedAt: now.
				Add(-time.Minute).
				Format(time.RFC3339Nano),
			LauncherVersionRange: ">=0.1.0 <0.2.0",
			PluginVersionRange:   ">=0.1.0 <0.2.0",
			Mode:                 "VERIFIED",
			AllowedActions: []string{
				"DEPENDENCY_CACHE",
				"REMOTE_CACHE_ALLOWLISTED",
			},
			RemoteCache: localauthority.RemoteCachePolicy{
				Read:                true,
				Write:               "TRUSTED_CI_ONLY",
				Namespace:           "stable",
				NamespaceGeneration: 12,
			},
			ConfigurationCache: localauthority.ConfigurationCachePolicy{
				Enabled:         true,
				ContractVersion: "configuration-cache-v1",
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
				EnabledInCI: true,
			},
			ExpiresAt: now.
				Add(time.Hour).
				Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            "revocation-request-7",
			TrustDomain:          "private-beta",
			RevocationEpoch:      7,
			L1SecurityGeneration: 9,
			ValidUntil: now.
				Add(2 * time.Hour).
				Format(time.RFC3339Nano),
		},
	}
	authority, err := localauthority.Sign(
		document,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatalf("sign launcher authority: %v", err)
	}
	var signed localauthority.Document
	verified, err := localauthority.Verify(
		context.Background(),
		authority,
		map[string]ed25519.PublicKey{"deployment-key-1": publicKey},
		credential,
		now,
	)
	if err != nil {
		t.Fatalf("verify launcher fixture: %v", err)
	}
	signed = verified.Document()
	trustRoot, err := localauthority.EncodeTrustRoot(
		localauthority.TrustRoot{
			Keys: []localauthority.PublicKey{{
				KeyID: "deployment-key-1",
				PublicKey: base64.RawURLEncoding.EncodeToString(
					publicKey,
				),
			}},
		},
	)
	if err != nil {
		t.Fatalf("encode launcher trust root: %v", err)
	}
	authorityPath := filepath.Join(root, "authority.json")
	trustRootPath := filepath.Join(root, "trust-root.json")
	credentialPath := filepath.Join(root, "credential")
	for path, content := range map[string][]byte{
		authorityPath: authority,
		trustRootPath: trustRoot,
		credentialPath: []byte(
			base64.RawURLEncoding.EncodeToString(credential),
		),
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("write launcher authority fixture: %v", err)
		}
	}
	return map[string]string{
		localAuthorityPathEnvironment:     authorityPath,
		localTrustRootPathEnvironment:     trustRootPath,
		localCredentialPathEnvironment:    credentialPath,
		sharedCacheURLEnvironment:         upstream,
		managedL1StateRootEnvironment:     stateRoot,
		managedL1TenantEnvironment:        "tenant-internal",
		managedL1RepositoryEnvironment:    "tonyredondo/buildopt",
		managedL1TrustDomainEnvironment:   "private-beta",
		managedL1CompatibilityEnvironment: "gradle-9.6-java-17-linux-amd64",
		"CI":                              "true",
	}, signed
}

func mapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
