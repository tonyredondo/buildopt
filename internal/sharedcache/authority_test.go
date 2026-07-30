package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

var sharedAuthorityNow = time.Date(
	2026,
	time.July,
	30,
	12,
	0,
	0,
	0,
	time.UTC,
)

func TestInstallLocalAuthorityAuthenticatesPendingDataPlane(t *testing.T) {
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return sharedAuthorityNow }
	verified, credential, _, _ := sharedAuthorityFixture(t, func(
		*localauthority.Document,
	) {
	})

	binding, changed, err := storage.InstallLocalAuthority(
		context.Background(),
		verified,
		credential,
		sharedAuthorityNow,
	)
	if err != nil || !changed {
		t.Fatalf("install authority = %+v/%t/%v", binding, changed, err)
	}
	if binding.AuthorityDigest == "" ||
		binding.AttemptID != verified.Document().Attempt.AttemptID ||
		!binding.AllowRead ||
		!binding.AllowWrite {
		t.Fatalf("unexpected binding: %+v", binding)
	}
	status, err := storage.AttemptStatus(
		context.Background(),
		binding.AttemptID,
	)
	if err != nil ||
		status.AuthorityDigest != binding.AuthorityDigest ||
		status.State != AttemptPending {
		t.Fatalf("authority attempt = %+v/%v", status, err)
	}
	assertRowCount(t, storage.control.database, "local_authority_state", 1)
	assertRowCount(t, storage.control.database, "local_authority_documents", 1)
	assertRowCount(t, storage.cache.database, "attempt_authorities", 1)

	handler, err := NewLocalAuthorityHTTPHandler(
		storage,
		binding,
		credential,
	)
	if err != nil {
		t.Fatalf("create authenticated handler: %v", err)
	}
	payload := []byte("authenticated pending payload")
	unauthorized := authorityHTTPRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/authority-key",
		payload,
		"",
		binding.AuthorityDigest,
	)
	if unauthorized.Code != http.StatusUnauthorized ||
		unauthorized.Body.Len() != 0 {
		t.Fatalf("unauthorized PUT = %d/%q", unauthorized.Code, unauthorized.Body.Bytes())
	}
	wrongDigest := authorityHTTPRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/authority-key",
		payload,
		base64.RawURLEncoding.EncodeToString(credential),
		"sha256:"+strings.Repeat("f", 64),
	)
	if wrongDigest.Code != http.StatusUnauthorized {
		t.Fatalf("wrong authority digest PUT = %d", wrongDigest.Code)
	}
	created := authorityHTTPRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/authority-key",
		payload,
		base64.RawURLEncoding.EncodeToString(credential),
		binding.AuthorityDigest,
	)
	if created.Code != http.StatusCreated ||
		created.Header().Get("X-BuildOpt-Blob-Digest") == "" {
		t.Fatalf("authenticated PUT = %d/%v", created.Code, created.Header())
	}
	status, err = storage.AttemptStatus(context.Background(), binding.AttemptID)
	if err != nil || status.PendingObjectCount != 1 {
		t.Fatalf("pending status = %+v/%v", status, err)
	}

	replay, changed, err := storage.InstallLocalAuthority(
		context.Background(),
		verified,
		credential,
		sharedAuthorityNow.Add(time.Minute),
	)
	if err != nil || changed ||
		replay.AuthorityDigest != binding.AuthorityDigest {
		t.Fatalf("authority replay = %+v/%t/%v", replay, changed, err)
	}

	var storedCredential string
	if err := storage.control.database.QueryRow(
		`SELECT credential_digest
FROM local_authority_documents
WHERE authority_digest = ?`,
		binding.AuthorityDigest,
	).Scan(&storedCredential); err != nil {
		t.Fatal(err)
	}
	if storedCredential == base64.RawURLEncoding.EncodeToString(credential) ||
		storedCredential != localCredentialDigest(credential) {
		t.Fatalf("persisted credential value = %q", storedCredential)
	}
}

func TestNewAuthoritySupersedesOldRouteAndRejectsRollback(t *testing.T) {
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return sharedAuthorityNow }
	first, credential, privateKey, keys := sharedAuthorityFixture(
		t,
		func(*localauthority.Document) {},
	)
	firstBinding, _, err := storage.InstallLocalAuthority(
		context.Background(),
		first,
		credential,
		sharedAuthorityNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	firstHandler, err := NewLocalAuthorityHTTPHandler(
		storage,
		firstBinding,
		credential,
	)
	if err != nil {
		t.Fatal(err)
	}

	advanced, _, _, _ := sharedAuthorityFixture(t, func(
		document *localauthority.Document,
	) {
		document.Attempt.AttemptID = "22222222-2222-4222-8222-222222222222"
		document.Attempt.LeaseID = "lease-authority-2"
		document.Policy.PolicyVersion++
		document.Policy.ConfigurationPolicyDigest = "sha256:" +
			strings.Repeat("6", 64)
		document.Policy.RevocationEpoch++
		document.Policy.L1SecurityGeneration++
		document.Policy.GatewayConnectionGeneration++
		document.Policy.RemoteCache.Namespace = "stable-v2"
		document.Policy.RemoteCache.NamespaceGeneration++
		document.Revocation.RequestID = "revocation-request-8"
		document.Revocation.RevocationEpoch++
		document.Revocation.L1SecurityGeneration++
	})
	secondBinding, changed, err := storage.InstallLocalAuthority(
		context.Background(),
		advanced,
		credential,
		sharedAuthorityNow.Add(time.Minute),
	)
	if err != nil || !changed {
		t.Fatalf("install advanced authority = %+v/%t/%v", secondBinding, changed, err)
	}
	oldResponse := authorityHTTPRequest(
		t,
		firstHandler,
		http.MethodGet,
		"/cache/old",
		nil,
		base64.RawURLEncoding.EncodeToString(credential),
		firstBinding.AuthorityDigest,
	)
	if oldResponse.Code != http.StatusUnauthorized {
		t.Fatalf("superseded authority GET = %d, want 401", oldResponse.Code)
	}
	secondHandler, err := NewLocalAuthorityHTTPHandler(
		storage,
		secondBinding,
		credential,
	)
	if err != nil {
		t.Fatal(err)
	}
	newResponse := authorityHTTPRequest(
		t,
		secondHandler,
		http.MethodPut,
		"/cache/new",
		[]byte("new generation"),
		base64.RawURLEncoding.EncodeToString(credential),
		secondBinding.AuthorityDigest,
	)
	if newResponse.Code != http.StatusCreated {
		t.Fatalf("advanced authority PUT = %d", newResponse.Code)
	}

	rollbackDocument := first.Document()
	rollbackDocument.AuthorityDigest = ""
	rollbackDocument.Policy.PolicyDigest = ""
	rollbackDocument.Revocation.CumulativeStateDigest = ""
	rollbackDocument.Revocation.Authentication.Signature = ""
	rollbackDocument.Authentication.Signature = ""
	rollbackCanonical, err := localauthority.Sign(
		rollbackDocument,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	rollbackVerified, err := localauthority.Verify(
		context.Background(),
		rollbackCanonical,
		keys,
		credential,
		sharedAuthorityNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.InstallLocalAuthority(
		context.Background(),
		rollbackVerified,
		credential,
		sharedAuthorityNow.Add(2*time.Minute),
	); !errors.Is(err, localauthority.ErrRollback) {
		t.Fatalf("rollback install = %v, want ErrRollback", err)
	}
}

func TestReadOnlyLocalAuthorityCreatesNoPendingAttempt(t *testing.T) {
	storage := openLifecycleTestStorage(t)
	storage.clock = func() time.Time { return sharedAuthorityNow }
	verified, credential, _, _ := sharedAuthorityFixture(t, func(
		document *localauthority.Document,
	) {
		document.Attempt.AllowWrite = false
		document.Policy.RemoteCache.Write = "DISABLED"
		document.Policy.AffectedBuild.EnabledInCI = false
		document.Policy.AffectedBuild.EnabledLocally = true
	})
	binding, _, err := storage.InstallLocalAuthority(
		context.Background(),
		verified,
		credential,
		sharedAuthorityNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AttemptStatus(
		context.Background(),
		binding.AttemptID,
	); err != ErrAttemptNotFound {
		t.Fatalf("read-only attempt status = %v, want not found", err)
	}
	handler, err := NewLocalAuthorityHTTPHandler(
		storage,
		binding,
		credential,
	)
	if err != nil {
		t.Fatal(err)
	}
	response := authorityHTTPRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/missing",
		nil,
		base64.RawURLEncoding.EncodeToString(credential),
		binding.AuthorityDigest,
	)
	if response.Code != http.StatusNotFound || response.Body.Len() != 0 {
		t.Fatalf("authorized miss = %d/%q", response.Code, response.Body.Bytes())
	}
}

func sharedAuthorityFixture(
	t *testing.T,
	mutate func(*localauthority.Document),
) (
	localauthority.Verified,
	[]byte,
	ed25519.PrivateKey,
	map[string]ed25519.PublicKey,
) {
	t.Helper()
	credential := bytes.Repeat([]byte{0x5a}, localauthority.CredentialBytes)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, 32))
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
			AttemptID:        "11111111-1111-4111-8111-111111111111",
			OwnerID:          "protected-main",
			LeaseID:          "lease-authority-1",
			LeaseExpiresAt:   sharedAuthorityNow.Add(45 * time.Minute).Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       true,
			CredentialDigest: localCredentialDigest(credential),
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
			IssuedAt:                    sharedAuthorityNow.Add(-time.Minute).Format(time.RFC3339Nano),
			LauncherVersionRange:        ">=0.1.0 <0.2.0",
			PluginVersionRange:          ">=0.1.0 <0.2.0",
			Mode:                        "VERIFIED",
			AllowedActions:              []string{"REMOTE_CACHE_ALLOWLISTED"},
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
				EnabledInCI: true,
			},
			ExpiresAt: sharedAuthorityNow.Add(time.Hour).Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            "revocation-request-7",
			TrustDomain:          "private-beta",
			RevocationEpoch:      7,
			L1SecurityGeneration: 9,
			ValidUntil:           sharedAuthorityNow.Add(2 * time.Hour).Format(time.RFC3339Nano),
		},
	}
	mutate(&document)
	canonical, err := localauthority.Sign(
		document,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"deployment-key-1": publicKey}
	verified, err := localauthority.Verify(
		context.Background(),
		canonical,
		keys,
		credential,
		sharedAuthorityNow,
	)
	if err != nil {
		t.Fatalf("verify fixture authority: %v", err)
	}
	return verified, credential, privateKey, keys
}

func authorityHTTPRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body []byte,
	credential string,
	authorityDigest string,
) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.ContentLength = int64(len(body))
	}
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	if authorityDigest != "" {
		request.Header.Set(AuthorityDigestHeader, authorityDigest)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
