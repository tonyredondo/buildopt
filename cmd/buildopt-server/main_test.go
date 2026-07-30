package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const serverTestToken = "server-test-ingest-token-0123456789abcdef"

func TestBuildoptServerUsageAndConfiguration(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		token      string
		wantExit   int
		wantOutput string
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
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return testCase.token
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
			!info.Mode().IsRegular() ||
			info.Mode().Perm() != 0o600 {
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
	case <-time.After(3 * time.Second):
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
				return ""
			},
			output,
			&stderr,
		)
	}()

	select {
	case <-output.written:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not report readiness")
	}
	line := output.String()
	const prefix = "buildopt-server: listening on "
	start := strings.Index(line, prefix)
	if start < 0 {
		t.Fatalf("missing listen line: %q", line)
	}
	endpoint := strings.TrimSpace(line[start+len(prefix):])

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
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after cancellation")
	}
	if !strings.Contains(
		output.String(),
		"accepted session server-main-test outcome=SUCCESS exit=0",
	) {
		t.Fatalf("missing acceptance log: %q", output.String())
	}
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
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(writer.String(), fragment) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server output never contained %q: %q", fragment, writer.String())
}
