package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildhistory"
	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/selfhosted"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const serverTestToken = "server-test-ingest-token-0123456789abcdef"
const serverHistoryTestToken = "server-test-history-token-0123456789abcdef"

func TestBuildoptServerUsageAndConfiguration(t *testing.T) {
	testCases := []struct {
		name         string
		args         []string
		token        string
		historyToken string
		wantExit     int
		wantOutput   string
	}{
		{
			name:       "help",
			args:       []string{"--help"},
			wantExit:   0,
			wantOutput: serverUsage,
		},
		{
			name:       "missing command",
			wantExit:   exitUsage,
			wantOutput: serverUsage,
		},
		{
			name:       "unknown command",
			args:       []string{"unknown"},
			wantExit:   exitUsage,
			wantOutput: serverUsage,
		},
		{
			name:       "export help",
			args:       []string{"export", "--help"},
			wantExit:   0,
			wantOutput: serverUsage,
		},
		{
			name: "unsupported export format",
			args: []string{
				"export",
				"--export-dir",
				"/tmp/buildopt-test-export",
				"--format",
				"json",
			},
			wantExit:   exitConfiguration,
			wantOutput: "format jsonl",
		},
		{
			name:       "unknown argument",
			args:       []string{"serve", "extra"},
			wantExit:   exitUsage,
			wantOutput: serverUsage,
		},
		{
			name:       "non-loopback listener",
			args:       []string{"serve", "--listen", "0.0.0.0:8042"},
			token:      serverTestToken,
			wantExit:   exitConfiguration,
			wantOutput: "invalid listen address",
		},
		{
			name:       "missing token",
			args:       []string{"serve", "--listen", "127.0.0.1:0"},
			wantExit:   exitConfiguration,
			wantOutput: "invalid session ingest configuration",
		},
		{
			name:         "history without exports",
			args:         []string{"serve", "--listen", "127.0.0.1:0"},
			token:        serverTestToken,
			historyToken: serverHistoryTestToken,
			wantExit:     exitConfiguration,
			wantOutput:   "history API requires an export directory",
		},
		{
			name: "relative Shared state",
			args: []string{
				"serve",
				"--listen",
				"127.0.0.1:0",
				"--state-dir",
				"relative/shared",
			},
			token:      serverTestToken,
			wantExit:   exitConfiguration,
			wantOutput: "state directory must be absolute",
		},
		{
			name: "incomplete cache authority",
			args: []string{
				"serve",
				"--listen",
				"127.0.0.1:0",
				"--cache-authority",
				"/tmp/authority.json",
			},
			token:      serverTestToken,
			wantExit:   exitConfiguration,
			wantOutput: "authenticated Shared cache requires",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return testCase.token
				}
				if key == buildhistory.TokenEnvironment {
					return testCase.historyToken
				}
				return ""
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(
				context.Background(),
				testCase.args,
				getenv,
				&stdout,
				&stderr,
			)
			if exitCode != testCase.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, testCase.wantExit)
			}
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, testCase.wantOutput) {
				t.Fatalf(
					"output = %q, want %q",
					output,
					testCase.wantOutput,
				)
			}
		})
	}
}

func TestBuildoptServerConsumesSelfHostedConfigurationBeforeListening(t *testing.T) {
	root := t.TempDir()
	exportDirectory := filepath.Join(root, "exports")
	if err := os.Mkdir(exportDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	configuration := selfhosted.Config{
		SchemaVersion: selfhosted.SchemaVersion,
		Profile:       selfhosted.Profile,
		Server:        selfhosted.Server{Listen: "127.0.0.1:0"},
		Storage: selfhosted.Storage{
			StateDirectory:         filepath.Join(root, "state"),
			FilesystemPolicy:       selfhosted.FilesystemPolicy,
			MinimumDeploymentBytes: selfhosted.MinimumDeploymentBytes,
			MaximumDeploymentBytes: selfhosted.MaximumDeploymentBytes,
			UsableVolumePercent:    selfhosted.UsableVolumePercent,
		},
		Export: selfhosted.Export{Directory: exportDirectory, Profile: "summary"},
		Cache: selfhosted.Cache{
			AuthorityPath:           filepath.Join(root, "secrets", "authority.json"),
			TrustRootPath:           filepath.Join(root, "secrets", "trust.json"),
			CredentialPath:          filepath.Join(root, "secrets", "credential"),
			BetaTokenAuthentication: true,
		},
	}
	raw, err := json.Marshal(configuration)
	if err != nil {
		t.Fatal(err)
	}
	configurationPath := filepath.Join(root, "self-hosted.json")
	if err := os.WriteFile(configurationPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	previousOpen := openSelfHostedStorage
	t.Cleanup(func() { openSelfHostedStorage = previousOpen })
	preflightFailure := errors.New("test preflight failure")
	openSelfHostedStorage = func(_ context.Context, stateDirectory string) (*sharedcache.Storage, error) {
		if stateDirectory != configuration.Storage.StateDirectory {
			t.Fatalf("state directory = %q", stateDirectory)
		}
		return nil, preflightFailure
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run(
		context.Background(),
		[]string{"serve", "--self-hosted-config", configurationPath},
		func(key string) string {
			if key == sessioningest.ServerTokenEnvironment {
				return serverTestToken
			}
			return ""
		},
		&stdout,
		&stderr,
	)
	if exitCode != exitConfiguration || !strings.Contains(stderr.String(), preflightFailure.Error()) {
		t.Fatalf("run = %d, stderr = %q", exitCode, stderr.String())
	}
	if strings.Contains(stdout.String(), "listening on") {
		t.Fatalf("listener opened before storage preflight: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = run(
		context.Background(),
		[]string{"serve", "--self-hosted-config", configurationPath, "--listen", "127.0.0.1:0"},
		func(string) string { return "" },
		&stdout,
		&stderr,
	)
	if exitCode != exitConfiguration || !strings.Contains(stderr.String(), "cannot be combined") {
		t.Fatalf("mixed configuration = %d, stderr = %q", exitCode, stderr.String())
	}
}

func TestBuildoptServerOwnsSingleNodeSharedStorageLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stateDirectory := filepath.Join(t.TempDir(), "shared")
	output := newNotifyingWriter()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() {
		exited <- run(
			ctx,
			[]string{
				"serve",
				"--listen",
				"127.0.0.1:0",
				"--state-dir",
				stateDirectory,
			},
			func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return serverTestToken
				}
				return ""
			},
			output,
			&stderr,
		)
	}()

	waitForServerOutput(
		t,
		output,
		"single-node Shared storage initialized",
	)
	line := output.String()
	const prefix = "buildopt-server: listening on "
	start := strings.Index(line, prefix)
	if start < 0 {
		t.Fatalf("missing listen line: %q", line)
	}
	end := strings.IndexByte(line[start+len(prefix):], '\n')
	if end < 0 {
		t.Fatalf("unterminated listen line: %q", line)
	}
	endpoint := line[start+len(prefix) : start+len(prefix)+end]

	layout := sharedcache.Layout{
		Root:            stateDirectory,
		Blobs:           filepath.Join(stateDirectory, "blobs", "sha256"),
		Spool:           filepath.Join(stateDirectory, "spool"),
		Quarantine:      filepath.Join(stateDirectory, "quarantine"),
		CacheDatabase:   filepath.Join(stateDirectory, "cache.sqlite"),
		ControlDatabase: filepath.Join(stateDirectory, "control.sqlite"),
		WriterLock:      filepath.Join(stateDirectory, "writer.lock"),
	}
	for _, path := range []string{
		layout.CacheDatabase,
		layout.ControlDatabase,
		layout.WriterLock,
	} {
		if info, err := os.Lstat(path); err != nil ||
			!privateLifecycleTokenKeyInfo(info) {
			t.Fatalf("server storage file %s = %v/%v", path, info, err)
		}
	}
	if second, err := sharedcache.Open(
		context.Background(),
		stateDirectory,
	); second != nil || !errors.Is(err, sharedcache.ErrWriterBusy) {
		if second != nil {
			_ = second.Close()
		}
		t.Fatalf("concurrent server storage = %+v/%v", second, err)
	}

	request, err := http.NewRequest(
		http.MethodGet,
		endpoint+"/cache/test",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+serverTestToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request absent cache data plane: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("cache data plane status = %d, want 404", response.StatusCode)
	}

	cancel()
	select {
	case exitCode := <-exited:
		if exitCode != 0 {
			t.Fatalf(
				"server exit = %d, stderr = %q",
				exitCode,
				stderr.String(),
			)
		}
	case <-time.After(nativeTestTimeout(3 * time.Second)):
		t.Fatal("server did not release Shared storage")
	}
	reopened, err := sharedcache.Open(context.Background(), stateDirectory)
	if err != nil {
		t.Fatalf("reopen server storage: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("close reopened server storage: %v", err)
	}
}

func TestBuildoptServerRoutesOnlyAuthenticatedCurrentCacheAuthority(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root := t.TempDir()
	stateDirectory := filepath.Join(root, "shared")
	authorityPath, trustRootPath, credentialPath, credential, authorityDigest :=
		writeServerAuthorityFixture(t, root, time.Now().UTC())
	output := newNotifyingWriter()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() {
		exited <- run(
			ctx,
			[]string{
				"serve",
				"--listen",
				"127.0.0.1:0",
				"--state-dir",
				stateDirectory,
				"--cache-authority",
				authorityPath,
				"--cache-trust-root",
				trustRootPath,
				"--cache-credential",
				credentialPath,
			},
			func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return serverTestToken
				}
				return ""
			},
			output,
			&stderr,
		)
	}()
	deadline := time.Now().Add(nativeTestTimeout(3 * time.Second))
	for !strings.Contains(
		output.String(),
		"authenticated cache routing enabled",
	) {
		select {
		case exitCode := <-exited:
			t.Fatalf(
				"authenticated server exited early with %d: %q",
				exitCode,
				stderr.String(),
			)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"authenticated server did not start: %q/%q",
				output.String(),
				stderr.String(),
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
	endpoint := serverEndpointFromOutput(t, output.String())

	requestCache := func(
		method string,
		digest string,
		authorized bool,
	) int {
		t.Helper()
		request, err := http.NewRequest(
			method,
			endpoint+"/cache/authority-key",
			strings.NewReader("candidate"),
		)
		if err != nil {
			t.Fatal(err)
		}
		if authorized {
			request.Header.Set(
				"Authorization",
				"Bearer "+
					base64.RawURLEncoding.EncodeToString(credential),
			)
			request.Header.Set(
				sharedcache.AuthorityDigestHeader,
				digest,
			)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("request authenticated cache route: %v", err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if status := requestCache(
		http.MethodPut,
		authorityDigest,
		false,
	); status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated cache status = %d, want 401", status)
	}
	if status := requestCache(
		http.MethodPut,
		authorityDigest,
		true,
	); status != http.StatusCreated {
		t.Fatalf("authenticated cache status = %d, want 201", status)
	}

	reloadStarted := time.Now()
	nextAuthorityDigest := advanceServerAuthorityFixture(
		t,
		authorityPath,
		reloadStarted,
	)
	reloadMessage := "local cache authority reloaded: " +
		nextAuthorityDigest
	deadline = time.Now().Add(5 * time.Second)
	for !strings.Contains(output.String(), reloadMessage) {
		if time.Now().After(deadline) {
			t.Fatalf(
				"authority did not reload: %q/%q",
				output.String(),
				stderr.String(),
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if elapsed := time.Since(reloadStarted); elapsed >= 60*time.Second {
		t.Fatalf("authority reload took %s, want <60s", elapsed)
	}
	requireServerStatus(t, endpoint+readinessPath, http.StatusOK)
	if status := requestCache(
		http.MethodPut,
		authorityDigest,
		true,
	); status != http.StatusUnauthorized {
		t.Fatalf("revoked authority status = %d, want 401", status)
	}
	if status := requestCache(
		http.MethodGet,
		nextAuthorityDigest,
		true,
	); status != http.StatusNotFound {
		t.Fatalf("current authority miss status = %d, want 404", status)
	}

	cancel()
	select {
	case exitCode := <-exited:
		if exitCode != 0 {
			t.Fatalf(
				"server exit = %d, stderr = %q",
				exitCode,
				stderr.String(),
			)
		}
	case <-time.After(nativeTestTimeout(3 * time.Second)):
		t.Fatal("authenticated server did not stop")
	}
}

func advanceServerAuthorityFixture(
	t *testing.T,
	authorityPath string,
	now time.Time,
) string {
	t.Helper()
	content, err := os.ReadFile(authorityPath)
	if err != nil {
		t.Fatalf("read authority for reload: %v", err)
	}
	var document localauthority.Document
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatalf("decode authority for reload: %v", err)
	}
	document.Attempt = localauthority.AuthorityAttempt{
		AttemptID: "33333333-3333-4333-8333-333333333333",
		OwnerID:   "protected-main",
		LeaseID:   "lease-server-authority-2",
		LeaseExpiresAt: now.Add(45 * time.Minute).
			UTC().
			Format(time.RFC3339Nano),
		AllowRead:        true,
		AllowWrite:       false,
		CredentialDigest: document.Attempt.CredentialDigest,
	}
	document.Policy.PolicyVersion = 8
	document.Policy.ConfigurationPolicyDigest =
		"sha256:" + strings.Repeat("6", 64)
	document.Policy.RevocationEpoch = 8
	document.Policy.L1SecurityGeneration = 10
	document.Policy.GatewayConnectionGeneration = 4
	document.Policy.IssuedAt = now.Add(-time.Second).
		UTC().
		Format(time.RFC3339Nano)
	document.Policy.RemoteCache.NamespaceGeneration = 13
	document.Policy.ExpiresAt = now.Add(time.Hour).
		UTC().
		Format(time.RFC3339Nano)
	document.Revocation.RequestID = "revocation-request-8"
	document.Revocation.RevocationEpoch = 8
	document.Revocation.L1SecurityGeneration = 10
	document.Revocation.ValidUntil = now.Add(2 * time.Hour).
		UTC().
		Format(time.RFC3339Nano)

	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, 32))
	authority, err := localauthority.Sign(
		document,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatalf("sign reloaded authority: %v", err)
	}
	var signed localauthority.Document
	if err := json.Unmarshal(authority, &signed); err != nil {
		t.Fatalf("decode signed reloaded authority: %v", err)
	}
	temporary := authorityPath + ".next"
	if err := os.WriteFile(temporary, authority, 0o600); err != nil {
		t.Fatalf("write reloaded authority: %v", err)
	}
	if err := os.Rename(temporary, authorityPath); err != nil {
		t.Fatalf("publish reloaded authority: %v", err)
	}
	return signed.AuthorityDigest
}

func TestBuildoptServerReceivesAndStopsGracefully(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exportDirectory := filepath.Join(t.TempDir(), "exports")
	output := newNotifyingWriter()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() {
		exited <- run(
			ctx,
			[]string{
				"serve",
				"--listen",
				"127.0.0.1:0",
				"--export-dir",
				exportDirectory,
			},
			func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return serverTestToken
				}
				if key == buildhistory.TokenEnvironment {
					return serverHistoryTestToken
				}
				return ""
			},
			output,
			&stderr,
		)
	}()

	select {
	case <-output.written:
	case <-time.After(nativeTestTimeout(2 * time.Second)):
		t.Fatal("server did not report readiness")
	}
	line := output.String()
	const prefix = "buildopt-server: listening on "
	start := strings.Index(line, prefix)
	if start < 0 {
		t.Fatalf("missing listen line: %q", line)
	}
	endpoint := strings.TrimSpace(line[start+len(prefix):])
	waitForServerReady(t, endpoint)

	client, err := sessioningest.NewClient(endpoint, serverTestToken)
	if err != nil {
		t.Fatalf("create ingest client: %v", err)
	}
	startedAt := time.Now()
	record := sessioningest.NewRecord(
		"server-main-test",
		"gateway-main-test",
		startedAt,
		startedAt.Add(time.Second),
		sessioningest.OutcomeSuccess,
		0,
	)
	record.ExportContext = &sessioningest.ExportContext{
		RepositoryID:         "repository-main-test",
		Revision:             "revision-main-test",
		RequestedTasks:       []string{"neutralProbe"},
		SourceStateDigest:    "hmac-sha256:" + strings.Repeat("a", 64),
		WorkUnitsFingerprint: "hmac-sha256:" + strings.Repeat("b", 64),
		TokenKeyVersion:      "fixture-token-v1",
		TrustDomain:          "fixture-local",
	}
	record.GradleInvocation = &sessioningest.GradleInvocation{
		ID: "gradle-invocation-main-test",
		StartedAt: startedAt.Add(100 * time.Millisecond).
			UTC().
			Format(time.RFC3339Nano),
		CompletedAt: startedAt.Add(900 * time.Millisecond).
			UTC().
			Format(time.RFC3339Nano),
		DurationMs:    800,
		PluginVersion: "0.1.0-SNAPSHOT",
	}
	if result, err := client.Deliver(context.Background(), record); err != nil ||
		result != sessioningest.PutCreated {
		t.Fatalf("deliver session = %d/%v", result, err)
	}
	exports, err := filepath.Glob(
		filepath.Join(exportDirectory, "build-session-*.json"),
	)
	if err != nil || len(exports) != 1 {
		t.Fatalf("exported files = %v/%v, want one", exports, err)
	}
	content, err := os.ReadFile(exports[0])
	if err != nil {
		t.Fatalf("read BUILD_SESSION export: %v", err)
	}
	var exported struct {
		SchemaVersion string `json:"schemaVersion"`
		RecordType    string `json:"recordType"`
		Build         struct {
			ID string `json:"id"`
		} `json:"build"`
	}
	if err := json.Unmarshal(content, &exported); err != nil {
		t.Fatalf("decode BUILD_SESSION export: %v", err)
	}
	if exported.SchemaVersion != "1.0" ||
		exported.RecordType != "BUILD_SESSION" ||
		exported.Build.ID != record.SessionID {
		t.Fatalf("unexpected BUILD_SESSION export: %+v", exported)
	}

	historyRequest, err := http.NewRequest(
		http.MethodGet,
		endpoint+buildhistory.ListPath+"?outcome=SUCCESS&limit=1",
		nil,
	)
	if err != nil {
		t.Fatalf("create history list request: %v", err)
	}
	historyRequest.Header.Set("Authorization", "Bearer "+serverHistoryTestToken)
	historyResponse, err := http.DefaultClient.Do(historyRequest)
	if err != nil {
		t.Fatalf("request build history: %v", err)
	}
	defer historyResponse.Body.Close()
	var historyPage buildhistory.ListResponse
	if historyResponse.StatusCode != http.StatusOK ||
		json.NewDecoder(historyResponse.Body).Decode(&historyPage) != nil ||
		historyPage.MatchedCount != 1 || len(historyPage.Items) != 1 ||
		historyPage.Items[0].ID != record.SessionID ||
		!strings.HasPrefix(historyPage.Items[0].RepositoryID, "hmac-sha256:") {
		t.Fatalf("unexpected build history response: %d/%+v", historyResponse.StatusCode, historyPage)
	}

	detailRequest, err := http.NewRequest(
		http.MethodGet,
		endpoint+buildhistory.DetailPath+"?id="+url.QueryEscape(record.SessionID),
		nil,
	)
	if err != nil {
		t.Fatalf("create history detail request: %v", err)
	}
	detailRequest.Header.Set("Authorization", "Bearer "+serverHistoryTestToken)
	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request build history detail: %v", err)
	}
	defer detailResponse.Body.Close()
	var historyDetail buildhistory.DetailResponse
	if detailResponse.StatusCode != http.StatusOK ||
		json.NewDecoder(detailResponse.Body).Decode(&historyDetail) != nil ||
		historyDetail.Session.Build.ID != record.SessionID {
		t.Fatalf("unexpected build history detail: %d/%+v", detailResponse.StatusCode, historyDetail)
	}

	wrongCredential, err := http.NewRequest(
		http.MethodGet,
		endpoint+buildhistory.ListPath,
		nil,
	)
	if err != nil {
		t.Fatalf("create wrong-credential request: %v", err)
	}
	wrongCredential.Header.Set("Authorization", "Bearer "+serverTestToken)
	wrongResponse, err := http.DefaultClient.Do(wrongCredential)
	if err != nil {
		t.Fatalf("request history with ingest credential: %v", err)
	}
	defer wrongResponse.Body.Close()
	if wrongResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ingest credential history status = %d", wrongResponse.StatusCode)
	}

	dashboardResponse, err := http.Get(endpoint + buildhistory.DashboardPath)
	if err != nil {
		t.Fatalf("request build history dashboard: %v", err)
	}
	defer dashboardResponse.Body.Close()
	dashboardBody, err := io.ReadAll(dashboardResponse.Body)
	if err != nil || dashboardResponse.StatusCode != http.StatusOK ||
		!bytes.Contains(dashboardBody, []byte("<title>Build history · BuildOpt</title>")) ||
		bytes.Contains(dashboardBody, []byte(serverHistoryTestToken)) ||
		bytes.Contains(dashboardBody, []byte(record.SessionID)) {
		t.Fatalf("unexpected build history dashboard: %d/%v", dashboardResponse.StatusCode, err)
	}

	cancel()
	select {
	case exitCode := <-exited:
		if exitCode != 0 {
			t.Fatalf(
				"server exit = %d, stderr = %q",
				exitCode,
				stderr.String(),
			)
		}
	case <-time.After(nativeTestTimeout(2 * time.Second)):
		t.Fatal("server did not stop after cancellation")
	}
	if !strings.Contains(
		output.String(),
		"accepted session server-main-test outcome=SUCCESS exit=0",
	) {
		t.Fatalf("missing acceptance log: %q", output.String())
	}

	var jsonl bytes.Buffer
	stderr.Reset()
	exitCode := run(
		context.Background(),
		[]string{
			"export",
			"--export-dir",
			exportDirectory,
			"--format",
			"jsonl",
		},
		func(string) string { return "" },
		&jsonl,
		&stderr,
	)
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf(
			"stdout JSONL export = %d/%q",
			exitCode,
			stderr.String(),
		)
	}
	lines := bytes.Split(
		bytes.TrimSuffix(jsonl.Bytes(), []byte{'\n'}),
		[]byte{'\n'},
	)
	if len(lines) != 2 {
		t.Fatalf("stdout JSONL lines = %d, want 2", len(lines))
	}
	for index, line := range lines {
		var event struct {
			BuildID  string `json:"buildId"`
			Sequence int    `json:"sequence"`
		}
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatalf("decode stdout JSONL line %d: %v", index, err)
		}
		if event.BuildID != record.SessionID ||
			event.Sequence != index+1 {
			t.Fatalf("stdout JSONL event %d = %+v", index, event)
		}
	}
}

func serverEndpointFromOutput(t *testing.T, output string) string {
	t.Helper()
	const prefix = "buildopt-server: listening on "
	start := strings.Index(output, prefix)
	if start < 0 {
		t.Fatalf("missing listen line: %q", output)
	}
	remaining := output[start+len(prefix):]
	end := strings.IndexByte(remaining, '\n')
	if end < 0 {
		return strings.TrimSpace(remaining)
	}
	return remaining[:end]
}

func writeServerAuthorityFixture(
	t *testing.T,
	root string,
	now time.Time,
) (
	string,
	string,
	string,
	[]byte,
	string,
) {
	t.Helper()
	credential := bytes.Repeat([]byte{0x5a}, localauthority.CredentialBytes)
	credentialHash := sha256.Sum256(credential)
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
			AttemptID: "22222222-2222-4222-8222-222222222222",
			OwnerID:   "protected-main",
			LeaseID:   "lease-server-authority-1",
			LeaseExpiresAt: now.Add(45 * time.Minute).
				Format(time.RFC3339Nano),
			AllowRead:        true,
			AllowWrite:       true,
			CredentialDigest: fmt.Sprintf("sha256:%x", credentialHash),
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
			IssuedAt:                    now.Add(-time.Minute).Format(time.RFC3339Nano),
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
			ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		},
		Revocation: localauthority.RevocationState{
			ContractVersion:      "buildopt-cache-control/v1",
			RequestID:            "revocation-request-7",
			TrustDomain:          "private-beta",
			RevocationEpoch:      7,
			L1SecurityGeneration: 9,
			ValidUntil: now.Add(2 * time.Hour).
				Format(time.RFC3339Nano),
		},
	}
	authority, err := localauthority.Sign(
		document,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatalf("sign server authority: %v", err)
	}
	verified, err := localauthority.Verify(
		context.Background(),
		authority,
		map[string]ed25519.PublicKey{"deployment-key-1": publicKey},
		credential,
		now,
	)
	if err != nil {
		t.Fatalf("verify server authority fixture: %v", err)
	}
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
		t.Fatalf("encode server trust root: %v", err)
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
			t.Fatalf("write server authority fixture: %v", err)
		}
	}
	return authorityPath,
		trustRootPath,
		credentialPath,
		credential,
		verified.Document().AuthorityDigest
}

type notifyingWriter struct {
	mutex   sync.Mutex
	buffer  bytes.Buffer
	written chan struct{}
	once    sync.Once
}

func newNotifyingWriter() *notifyingWriter {
	return &notifyingWriter{written: make(chan struct{})}
}

func (writer *notifyingWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	count, err := writer.buffer.Write(data)
	writer.once.Do(func() {
		close(writer.written)
	})
	return count, err
}

func (writer *notifyingWriter) String() string {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.buffer.String()
}

func waitForServerOutput(
	t *testing.T,
	writer *notifyingWriter,
	fragment string,
) {
	t.Helper()
	deadline := time.Now().Add(nativeTestTimeout(3 * time.Second))
	for time.Now().Before(deadline) {
		if strings.Contains(writer.String(), fragment) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server output never contained %q: %q", fragment, writer.String())
}

func waitForServerReady(t *testing.T, endpoint string) {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(nativeTestTimeout(3 * time.Second))
	for time.Now().Before(deadline) {
		response, err := client.Get(endpoint + readinessPath)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s never became ready", endpoint)
}
