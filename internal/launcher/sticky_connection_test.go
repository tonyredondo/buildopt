package launcher

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestStickyConnectionIsPortableScopedAndRevocable(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	storage, err := sharedcache.Open(ctx, filepath.Join(t.TempDir(), "server"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	issued := issueStickyConnectionToken(
		t,
		storage,
		now,
		"example/portable-project",
		"gradle-9.6.1/linux-amd64/jdk-21/project",
		[]sharedcache.CentralCapability{
			sharedcache.CentralCacheRead,
			sharedcache.CentralStateRead,
		},
	)
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		handler.ServeHTTP(response, request)
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return fmt.Errorf("redirect rejected")
	}
	tokenJSON := stickyConnectionTokenJSON(t, issued)

	firstRoot := writeStickyConnectionRepository(t, server.URL, "example/portable-project", "BUILDOPT_TEAM_TOKEN")
	secondRoot := writeStickyConnectionRepository(t, server.URL, "example/portable-project", "BUILDOPT_TEAM_TOKEN")
	getenv := func(name string) string {
		if name == "BUILDOPT_TEAM_TOKEN" {
			return tokenJSON
		}
		return ""
	}
	first, credentialEnvironment, err := prepareStickyWrapperConnection(
		firstRoot,
		[]string{filepath.Join(firstRoot, "gradlew"), "build"},
		getenv,
		now,
		client,
	)
	if err != nil || first == nil || credentialEnvironment != "BUILDOPT_TEAM_TOKEN" {
		t.Fatalf("first connection = %+v/%q/%v", first, credentialEnvironment, err)
	}
	defer first.close()
	second, _, err := prepareStickyWrapperConnection(
		secondRoot,
		[]string{filepath.Join(secondRoot, "gradlew"), "build"},
		getenv,
		now,
		client,
	)
	if err != nil || second == nil {
		t.Fatalf("second connection = %+v/%v", second, err)
	}
	defer second.close()
	if first.projectScopeSHA256 != second.projectScopeSHA256 ||
		first.connectionScopeSHA256 != second.connectionScopeSHA256 {
		t.Fatalf("clean checkouts drifted: %+v / %+v", first, second)
	}
	trailingSlashRoot := writeStickyConnectionRepository(
		t,
		server.URL+"/",
		"example/portable-project",
		"BUILDOPT_TEAM_TOKEN",
	)
	trailingSlash, _, err := prepareStickyWrapperConnection(
		trailingSlashRoot,
		[]string{filepath.Join(trailingSlashRoot, "gradlew"), "build"},
		getenv,
		now,
		client,
	)
	if err != nil || trailingSlash == nil {
		t.Fatalf("trailing-slash connection = %+v/%v", trailingSlash, err)
	}
	defer trailingSlash.close()
	if trailingSlash.serverURL != server.URL ||
		trailingSlash.connectionScopeSHA256 != first.connectionScopeSHA256 {
		t.Fatalf("canonical origin drifted: %+v / %+v", first, trailingSlash)
	}

	beforeRejected := requests.Load()
	otherRoot := writeStickyConnectionRepository(t, server.URL, "example/other-project", "BUILDOPT_TEAM_TOKEN")
	if connection, _, err := prepareStickyWrapperConnection(
		otherRoot,
		[]string{filepath.Join(otherRoot, "gradlew"), "build"},
		getenv,
		now,
		client,
	); err == nil || connection != nil {
		t.Fatalf("cross-repository token accepted: %+v/%v", connection, err)
	}
	if requests.Load() != beforeRejected {
		t.Fatal("cross-repository mismatch reached the server")
	}

	missingRoot := writeStickyConnectionRepository(t, server.URL, "example/portable-project", "BUILDOPT_MISSING_TOKEN")
	if connection, _, err := prepareStickyWrapperConnection(
		missingRoot,
		[]string{filepath.Join(missingRoot, "gradlew")},
		func(string) string { return "" },
		now,
		client,
	); err != nil || connection != nil {
		t.Fatalf("missing credential did not retain native: %+v/%v", connection, err)
	}
	if requests.Load() != beforeRejected {
		t.Fatal("missing credential reached the server")
	}

	stateOnly := issueStickyConnectionToken(
		t,
		storage,
		now,
		"example/portable-project",
		"gradle-9.6.1/linux-amd64/jdk-21/project",
		[]sharedcache.CentralCapability{sharedcache.CentralStateRead},
	)
	if connection, _, err := prepareStickyWrapperConnection(
		firstRoot,
		[]string{filepath.Join(firstRoot, "gradlew")},
		func(string) string { return stickyConnectionTokenJSON(t, stateOnly) },
		now,
		client,
	); err == nil || connection != nil || !strings.Contains(err.Error(), "CACHE_READ and STATE_READ") {
		t.Fatalf("wrong capabilities accepted: %+v/%v", connection, err)
	}
	if requests.Load() != beforeRejected {
		t.Fatal("wrong capability reached the server")
	}

	var futureDocument centralIssuedTokenDocument
	if err := json.Unmarshal([]byte(tokenJSON), &futureDocument); err != nil {
		t.Fatal(err)
	}
	futureDocument.IssuedAt = now.Add(time.Minute).Format(time.RFC3339Nano)
	futureJSON, err := json.Marshal(futureDocument)
	if err != nil {
		t.Fatal(err)
	}
	if connection, _, err := prepareStickyWrapperConnection(
		firstRoot,
		[]string{filepath.Join(firstRoot, "gradlew")},
		func(string) string { return string(futureJSON) },
		now,
		client,
	); err == nil || connection != nil || !strings.Contains(err.Error(), "time binding") {
		t.Fatalf("future-issued credential accepted: %+v/%v", connection, err)
	}
	if requests.Load() != beforeRejected {
		t.Fatal("future-issued credential reached the server")
	}

	otherNamespace := issueStickyConnectionToken(
		t,
		storage,
		now,
		"example/portable-project",
		"gradle-9.6.1/linux-amd64/jdk-21/other",
		[]sharedcache.CentralCapability{
			sharedcache.CentralCacheRead,
			sharedcache.CentralStateRead,
		},
	)
	other, _, err := prepareStickyWrapperConnection(
		firstRoot,
		[]string{filepath.Join(firstRoot, "gradlew")},
		func(string) string { return stickyConnectionTokenJSON(t, otherNamespace) },
		now,
		client,
	)
	if err != nil || other == nil {
		t.Fatalf("second namespace connection = %+v/%v", other, err)
	}
	defer other.close()
	if other.connectionScopeSHA256 == first.connectionScopeSHA256 {
		t.Fatal("different server namespaces shared a connection identity")
	}

	if revoked, err := storage.RevokeCentralToken(ctx, issued.TokenID, time.Now().UTC()); err != nil || !revoked {
		t.Fatalf("revoke token = %t/%v", revoked, err)
	}
	if connection, _, err := prepareStickyWrapperConnection(
		firstRoot,
		[]string{filepath.Join(firstRoot, "gradlew")},
		getenv,
		now.Add(2*time.Minute),
		client,
	); err == nil || connection != nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("revoked credential accepted: %+v/%v", connection, err)
	}
	assertStickyRevokedCannotRead(t, client, server.URL, issued)
}

func TestStickyConnectionRejectsRedirectsAndForeignWrapperRoots(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	storage, err := sharedcache.Open(context.Background(), filepath.Join(t.TempDir(), "server"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	issued := issueStickyConnectionToken(
		t,
		storage,
		now,
		"example/redirect-project",
		"gradle/project",
		[]sharedcache.CentralCapability{sharedcache.CentralCacheRead, sharedcache.CentralStateRead},
	)
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, "https://elsewhere.invalid/", http.StatusFound)
	}))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect rejected") }
	root := writeStickyConnectionRepository(t, server.URL, "example/redirect-project", "BUILDOPT_TOKEN")
	getenv := func(string) string { return stickyConnectionTokenJSON(t, issued) }
	if connection, _, err := prepareStickyWrapperConnection(
		root,
		[]string{filepath.Join(root, "gradlew")},
		getenv,
		now,
		client,
	); err == nil || connection != nil || !strings.Contains(err.Error(), "redirect rejected") {
		t.Fatalf("redirect accepted: %+v/%v", connection, err)
	}
	foreign := filepath.Join(t.TempDir(), "gradlew")
	if connection, _, err := prepareStickyWrapperConnection(
		root,
		[]string{foreign},
		getenv,
		now,
		client,
	); err == nil || connection != nil {
		t.Fatalf("foreign Gradle Wrapper accepted: %+v/%v", connection, err)
	}
}

func issueStickyConnectionToken(
	t *testing.T,
	storage *sharedcache.Storage,
	now time.Time,
	repository string,
	namespace string,
	capabilities []sharedcache.CentralCapability,
) sharedcache.IssuedCentralToken {
	t.Helper()
	issued, err := storage.IssueCentralToken(
		context.Background(),
		sharedcache.CentralTokenIssueRequest{
			Scope: sharedcache.CentralTokenScope{
				RepositoryScopeSHA256: optimizePortfolioRepositoryScope(repository),
				Tenant:                "owner-poc", Repository: repository,
				TrustDomain: "owner-poc", Namespace: namespace, NamespaceGeneration: 1,
			},
			Capabilities: capabilities,
			ExpiresAt:    now.Add(time.Hour),
		},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	return issued
}

func stickyConnectionTokenJSON(t *testing.T, issued sharedcache.IssuedCentralToken) string {
	t.Helper()
	raw, err := json.Marshal(centralIssuedTokenDocument{
		SchemaVersion: "buildopt.central/access-token/v1",
		TokenID:       issued.TokenID, Token: issued.Token,
		RepositoryScopeSHA256: issued.Scope.RepositoryScopeSHA256,
		Tenant:                issued.Scope.Tenant, Repository: issued.Scope.Repository,
		TrustDomain: issued.Scope.TrustDomain, Namespace: issued.Scope.Namespace,
		NamespaceGeneration: issued.Scope.NamespaceGeneration,
		Capabilities:        issued.Capabilities,
		IssuedAt:            issued.IssuedAt.Format(time.RFC3339Nano),
		ExpiresAt:           issued.ExpiresAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func writeStickyConnectionRepository(t *testing.T, serverURL, projectScope, credentialEnvironment string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".buildopt"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(
		"schema_version = \"buildopt.config/v1\"\nmode = \"auto\"\nserver_url = %q\nproject_scope = %q\ncredential_env = %q\ntrial_budget_percent = 5\n",
		serverURL,
		projectScope,
		credentialEnvironment,
	)
	if err := os.WriteFile(filepath.Join(root, ".buildopt", "config.toml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "gradlew"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertStickyRevokedCannotRead(
	t *testing.T,
	client *http.Client,
	serverURL string,
	issued sharedcache.IssuedCentralToken,
) {
	t.Helper()
	paths := []struct {
		path    string
		headers map[string]string
	}{
		{path: centralStatePath(issued.Scope.RepositoryScopeSHA256, sharedcache.StateKindEvidence, "head", "")},
		{path: "/cache/" + strings.Repeat("a", 64), headers: map[string]string{sharedcache.CentralNamespaceHeader: issued.Scope.Namespace}},
	}
	for _, item := range paths {
		request, err := http.NewRequest(http.MethodGet, serverURL+item.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+issued.Token)
		for key, value := range item.headers {
			request.Header.Set(key, value)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("revoked read %s = HTTP %d", item.path, response.StatusCode)
		}
	}
}
