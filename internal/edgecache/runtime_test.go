package edgecache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

type runtimeAuthorityFixture struct {
	authority      []byte
	trustRoot      []byte
	credentialFile []byte
	credential     []byte
}

func TestRuntimeServesReplicatesReloadsFailsClosedAndStops(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	fixture := newRuntimeAuthorityFixture(t, now, 1)
	sharedCredential := string(fixture.credential)
	shared := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+sharedCredential {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch request.Method {
		case http.MethodGet:
			response.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			body, err := io.ReadAll(request.Body)
			if err != nil {
				response.WriteHeader(http.StatusInternalServerError)
				return
			}
			digest := sha256.Sum256(body)
			response.Header().Set(
				"X-BuildOpt-Blob-Digest",
				"sha256:"+hex.EncodeToString(digest[:]),
			)
			response.WriteHeader(http.StatusCreated)
		default:
			response.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer shared.Close()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	config, configPath := writeRuntimeConfiguration(
		t, root, listener.Addr().String(), shared.URL, fixture,
	)
	runtime, err := OpenRuntime(context.Background(), config)
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer runtime.Close()
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- runtime.Serve(ctx, listener) }()

	baseURL := "http://" + listener.Addr().String()
	waitForRuntimeStatus(t, config, func(status RuntimeStatus) bool {
		return StatusReady(status, time.Now().UTC()) && status.WriteEnabled
	})
	payload := []byte("edge-runtime-pending-object")
	request, err := http.NewRequest(
		http.MethodPut, baseURL+"/cache/runtime-key", bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("PUT status = %d", response.StatusCode)
	}
	waitForRuntimeStatus(t, config, func(status RuntimeStatus) bool {
		return status.Pending.Replicated == 1 && status.Pending.Queued == 0
	})

	writePrivateAtomic(t, config.Authority.SnapshotPath, []byte("invalid"))
	waitForHTTPStatus(t, baseURL+"/cache/runtime-key", http.StatusServiceUnavailable)
	waitForRuntimeStatus(t, config, func(status RuntimeStatus) bool {
		return status.State == "NOT_READY" &&
			status.AuthorityError == "AUTHORITY_UNAVAILABLE"
	})

	next := newRuntimeAuthorityFixture(t, time.Now().UTC().Truncate(time.Second), 2)
	writePrivateAtomic(t, config.Authority.SnapshotPath, next.authority)
	waitForHTTPStatus(t, baseURL+"/cache/runtime-key", http.StatusNotFound)
	waitForRuntimeStatus(t, config, func(status RuntimeStatus) bool {
		return status.State == "READY" && status.WriteEnabled
	})
	writePrivateAtomic(t, config.Authority.SnapshotPath, fixture.authority)
	waitForHTTPStatus(t, baseURL+"/cache/runtime-key", http.StatusServiceUnavailable)
	waitForRuntimeStatus(t, config, func(status RuntimeStatus) bool {
		return status.State == "NOT_READY" &&
			status.AuthorityError == "AUTHORITY_UNAVAILABLE"
	})
	writePrivateAtomic(t, config.Authority.SnapshotPath, next.authority)
	waitForHTTPStatus(t, baseURL+"/cache/runtime-key", http.StatusNotFound)
	waitForRuntimeStatus(t, config, func(status RuntimeStatus) bool {
		return status.State == "READY"
	})

	status, err := LoadRuntimeStatus(config)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		string(fixture.credential), root, configPath, "tonyredondo/buildopt",
	} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("status exposed private value %q", forbidden)
		}
	}

	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("serve shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Edge runtime did not stop")
	}
	status, err = LoadRuntimeStatus(config)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "STOPPED" || status.WriteEnabled ||
		StatusReady(status, time.Now().UTC()) {
		t.Fatalf("shutdown status = %+v", status)
	}
}

func newRuntimeAuthorityFixture(
	t *testing.T,
	now time.Time,
	generation int64,
) runtimeAuthorityFixture {
	t.Helper()
	credential := bytes.Repeat([]byte("Z"), localauthority.CredentialBytes)
	seed := bytes.Repeat([]byte{0x11}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	credentialDigest := sha256.Sum256(credential)
	document := localauthority.Document{
		Repository: localauthority.RepositoryIdentity{
			Tenant:      "tenant-internal",
			Repository:  "tonyredondo/buildopt",
			TrustDomain: "owner-poc",
		},
		SourceRevision:      strings.Repeat("a", 40),
		SourceStateDigest:   "hmac-sha256:" + strings.Repeat("1", 64),
		CacheContractDigest: "sha256:" + strings.Repeat("2", 64),
		Attempt: localauthority.AuthorityAttempt{
			AttemptID:        "attempt-runtime-" + strconv.FormatInt(generation, 10),
			OwnerID:          "protected-main",
			LeaseID:          "lease-runtime-" + strconv.FormatInt(generation, 10),
			LeaseExpiresAt:   now.Add(10 * time.Minute).Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       true,
			CredentialDigest: "sha256:" + hex.EncodeToString(credentialDigest[:]),
		},
		Policy: localauthority.OptimizationPolicy{
			SchemaVersion:               "1.0",
			RecordType:                  "OPTIMIZATION_POLICY",
			PolicyID:                    "edge-runtime-policy",
			PolicyVersion:               generation,
			ConfigurationPolicyDigest:   "sha256:" + strings.Repeat("3", 64),
			RevocationEpoch:             generation,
			L1SecurityGeneration:        generation,
			GatewayConnectionGeneration: generation,
			IssuedAt:                    now.Add(-time.Minute).Format(time.RFC3339Nano),
			LauncherVersionRange:        ">=0.1.0 <0.2.0",
			PluginVersionRange:          ">=0.1.0 <0.2.0",
			Mode:                        "VERIFIED",
			AllowedActions:              []string{"REMOTE_CACHE_ALLOWLISTED"},
			RemoteCache: localauthority.RemoteCachePolicy{
				Read:                true,
				Write:               "TRUSTED_CI_ONLY",
				Namespace:           "edge-runtime",
				NamespaceGeneration: generation,
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
			AffectedBuild: localauthority.AffectedBuild{EnabledInCI: true},
			ExpiresAt:     now.Add(10 * time.Minute).Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            "revocation-runtime-" + strconv.FormatInt(generation, 10),
			TrustDomain:          "owner-poc",
			RevocationEpoch:      generation,
			L1SecurityGeneration: generation,
			ValidUntil:           now.Add(10 * time.Minute).Format(time.RFC3339Nano),
		},
	}
	authority, err := localauthority.Sign(document, "edge-key-1", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	trustRoot, err := localauthority.EncodeTrustRoot(localauthority.TrustRoot{
		Keys: []localauthority.PublicKey{{
			KeyID:     "edge-key-1",
			PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtimeAuthorityFixture{
		authority:      authority,
		trustRoot:      trustRoot,
		credentialFile: []byte(base64.RawURLEncoding.EncodeToString(credential) + "\n"),
		credential:     credential,
	}
}

func writeRuntimeConfiguration(
	t *testing.T,
	root string,
	listen string,
	sharedURL string,
	fixture runtimeAuthorityFixture,
) (Config, string) {
	t.Helper()
	config := validConfig(root)
	config.EdgeID = "edge-runtime-test"
	config.Server.Listen = listen
	config.Shared.BaseURL = sharedURL
	config.Shared.AllowInsecureLoopback = true
	config.Shared.CredentialPath = filepath.Join(root, "credential")
	config.Authority.TrustRootPath = filepath.Join(root, "trust-root.json")
	config.Authority.SnapshotPath = filepath.Join(root, "authority.json")
	writePrivateAtomic(t, config.Shared.CredentialPath, fixture.credentialFile)
	writePrivateAtomic(t, config.Authority.TrustRootPath, fixture.trustRoot)
	writePrivateAtomic(t, config.Authority.SnapshotPath, fixture.authority)
	configPath := filepath.Join(root, "edge.json")
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	writePrivateAtomic(t, configPath, encoded)
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	return loaded, configPath
}

func writePrivateAtomic(t *testing.T, path string, content []byte) {
	t.Helper()
	temporary, err := os.CreateTemp(filepath.Dir(path), ".private-*")
	if err != nil {
		t.Fatal(err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := temporary.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := temporary.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		t.Fatal(err)
	}
}

func waitForRuntimeStatus(
	t *testing.T,
	config Config,
	accept func(RuntimeStatus) bool,
) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		status, err := LoadRuntimeStatus(config)
		if err == nil && accept(status) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	status, err := LoadRuntimeStatus(config)
	t.Fatalf("runtime status condition not reached: status=%+v err=%v", status, err)
}

func waitForHTTPStatus(t *testing.T, url string, expected int) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == expected {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("HTTP status not reached: %v", err)
	}
	_ = response.Body.Close()
	t.Fatalf("HTTP status = %d, want %d", response.StatusCode, expected)
}
