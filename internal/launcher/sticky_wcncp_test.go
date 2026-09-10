package launcher

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

func TestStickyWCNCPNativeObservation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX child fixture")
	}
	root := writeStickyConnectionRepository(t, "https://127.0.0.1:1", "example/installed", "BUILDOPT_TEAM_TOKEN")
	outbox := t.TempDir()
	t.Setenv(stickyWrapperRootEnvironment, root)
	t.Setenv("BUILDOPT_STICKY_OBSERVATION", "wcncp")
	t.Setenv("WCNCP_OUTBOX_DIR", outbox)
	t.Setenv("WCNCP_REPOSITORY_SCOPE", "foreign/project")
	t.Setenv("WCNCP_BACKEND_TOKEN", "private-secret")
	t.Setenv("BUILDOPT_TEAM_TOKEN", "malformed-private-token")
	t.Setenv("BUILDOPT_STICKY_OBSERVATION_OUTPUT", filepath.Join(t.TempDir(), "old.json"))
	script := "#!/bin/sh\nif env | sed -n '/^WCNCP_/p;/^BUILDOPT_/p' | read -r private; then exit 77; fi\nprintf '%s\\n' \"$@\"\ncat\nprintf 'native-error\\n' >&2\nexit 23\n"
	if err := os.WriteFile(stickyConnectionGradleCommand(root), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"run", "--", stickyConnectionGradleCommand(root), "build", "argument with spaces", ""}
	code := Run(args, strings.NewReader("native-input\n"), &stdout, &stderr)
	if code != 23 || stdout.String() != "build\nargument with spaces\n\nnative-input\n" || stderr.String() != "native-error\n" {
		t.Fatalf("native contract: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	files, err := filepath.Glob(filepath.Join(outbox, "*", "obs-*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("one scoped typed observation: %v %v", files, err)
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var facts wcncpobserve.ObservationFacts
	if err := json.Unmarshal(raw, &facts); err != nil {
		t.Fatal(err)
	}
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	if facts.RepositoryScope != "example/installed" || facts.Child.Outcome != "FAILED" || *facts.Child.ExitCode != 23 || facts.Completeness != "INCOMPLETE" || facts.Authority.ProductionAuthorized || facts.Duration.State != "UNAVAILABLE" {
		t.Fatalf("incorrect typed observation: %+v", facts)
	}
	for _, private := range []string{"private-secret", "malformed-private-token", root, "argument with spaces"} {
		if bytes.Contains(raw, []byte(private)) {
			t.Fatalf("observation disclosed private value")
		}
	}
	if _, err := os.Stat(os.Getenv("BUILDOPT_STICKY_OBSERVATION_OUTPUT")); !os.IsNotExist(err) {
		t.Fatalf("duplicate old observation: %v", err)
	}
}

func TestStickyWCNCPStatusUsesTypedQueue(t *testing.T) {
	root := t.TempDir()
	config := stickywrapper.Config{Mode: "observe", ServerURL: "https://127.0.0.1:1", ProjectScope: "example/status", CredentialEnv: "BUILDOPT_TOKEN", TrialBudgetPercent: 5}
	if _, err := (stickywrapper.Generator{Root: root, Resolver: wcncpFixtureResolver{}}).Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	outbox := filepath.Join(t.TempDir(), "not-created")
	t.Setenv("BUILDOPT_STICKY_OBSERVATION", "wcncp")
	t.Setenv("WCNCP_OUTBOX_DIR", outbox)
	t.Setenv("BUILDOPT_TOKEN", "")
	// Typed status must not consume unrelated ordinary/lifecycle state.
	old := filepath.Join(t.TempDir(), "old.json")
	if err := os.WriteFile(old, []byte("invalid old log"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUILDOPT_STICKY_OBSERVATION_OUTPUT", old)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"wrapper", "status", "--root", root, "--json"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status exit %d: %s", code, stderr.String())
	}
	var report map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	var status struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(report["nativeCorrection"], &status); err != nil || status.State != "UNAVAILABLE" {
		t.Fatalf("typed status absent: %s %v", stdout.String(), err)
	}
	if _, err := os.Stat(outbox); !os.IsNotExist(err) {
		t.Fatalf("status created outbox: %v", err)
	}
}

type wcncpFixtureResolver struct{}

func (wcncpFixtureResolver) Latest(context.Context) (stickywrapper.Release, error) {
	release := stickywrapper.Release{Version: "0.0.1", Distributions: map[string]stickywrapper.Distribution{}}
	for _, platform := range []string{"linux-amd64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		release.Distributions[platform] = stickywrapper.Distribution{URL: "https://127.0.0.1:1/0.0.1/" + platform, SHA256: strings.Repeat("a", 64)}
	}
	return release, nil
}
func (r wcncpFixtureResolver) Version(ctx context.Context, _ string) (stickywrapper.Release, error) {
	return r.Latest(ctx)
}

func TestStickyWCNCPAuthenticatedDelivery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX child fixture")
	}
	for _, scenario := range []string{"trusted", "missing", "malformed", "expired", "wrong-scope", "read-only", "untrusted", "revoked", "outage", "redirect", "partial-ack", "timeout"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			storage, err := sharedcache.Open(ctx, filepath.Join(t.TempDir(), "server"))
			if err != nil {
				t.Fatal(err)
			}
			defer storage.Close()
			now := time.Now().UTC().Truncate(time.Second)
			capabilities := []sharedcache.CentralCapability{sharedcache.CentralStateRead, sharedcache.CentralStateWrite}
			if scenario == "read-only" {
				capabilities = capabilities[:1]
			}
			issued := issueStickyConnectionToken(t, storage, now, "example/delivery", "gradle/project", capabilities)
			if scenario != "untrusted" && scenario != "read-only" {
				if _, err := storage.GrantWCNCPActor(ctx, issued.TokenID, sharedcache.WCNCPActorTrustedObserver, now); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "revoked" {
				if _, err := storage.RevokeCentralToken(ctx, issued.TokenID, now); err != nil {
					t.Fatal(err)
				}
			}
			handler, err := sharedcache.NewCentralHTTPSHandler(storage)
			if err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(t.TempDir(), "child-finished")
			var requests, early, responseCode atomic.Int64
			var recovered atomic.Bool
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if _, err := os.Stat(marker); err != nil {
					early.Add(1)
				}
				switch scenario {
				case "outage":
					if !recovered.Load() {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
				case "redirect":
					http.Redirect(w, r, "https://127.0.0.1:1/foreign", http.StatusTemporaryRedirect)
					return
				case "partial-ack":
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"published":0}`)
					return
				case "timeout":
					_, _ = io.Copy(io.Discard, r.Body)
					select {
					case <-r.Context().Done():
					case <-time.After(500 * time.Millisecond):
					}
					return
				}
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, r)
				responseCode.Store(int64(recorder.Code))
				for key, values := range recorder.Header() {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.WriteHeader(recorder.Code)
				_, _ = w.Write(recorder.Body.Bytes())
			}))
			server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
			server.StartTLS()
			defer server.Close()
			ca := filepath.Join(t.TempDir(), "ca.pem")
			if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
				t.Fatal(err)
			}
			root := writeStickyConnectionRepository(t, server.URL, "example/delivery", "BUILDOPT_TEAM_TOKEN")
			if err := os.WriteFile(stickyConnectionGradleCommand(root), []byte("#!/bin/sh\nprintf done > \"$NATIVE_MARKER\"\nprintf native\\n\n"), 0755); err != nil {
				t.Fatal(err)
			}
			tokenJSON := stickyConnectionTokenJSON(t, issued)
			switch scenario {
			case "missing":
				tokenJSON = ""
			case "malformed":
				tokenJSON = "private-invalid-document"
			case "expired":
				var document centralIssuedTokenDocument
				if err := json.Unmarshal([]byte(tokenJSON), &document); err != nil {
					t.Fatal(err)
				}
				document.IssuedAt = now.Add(-2 * time.Hour).Format(time.RFC3339Nano)
				document.ExpiresAt = now.Add(-time.Hour).Format(time.RFC3339Nano)
				raw, _ := json.Marshal(document)
				tokenJSON = string(raw)
			case "wrong-scope":
				tokenJSON = strings.ReplaceAll(tokenJSON, "example/delivery", "example/other")
			}
			t.Setenv(stickyWrapperRootEnvironment, root)
			t.Setenv(stickyWrapperCAEnvironment, ca)
			t.Setenv(stickyObservationModeEnvironment, "wcncp")
			t.Setenv("WCNCP_OUTBOX_DIR", t.TempDir())
			t.Setenv("BUILDOPT_TEAM_TOKEN", tokenJSON)
			t.Setenv("NATIVE_MARKER", marker)
			var stdout, stderr bytes.Buffer
			start := time.Now()
			if code := Run([]string{"run", "--", stickyConnectionGradleCommand(root), "build"}, nil, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("native code=%d stderr=%q", code, stderr.String())
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("optional observation exceeded local integration bound")
			}
			files, _ := filepath.Glob(filepath.Join(os.Getenv("WCNCP_OUTBOX_DIR"), "*", "obs-*.json"))
			if early.Load() != 0 {
				t.Fatal("pre-child backend request")
			}
			wantRequests := int64(1)
			if scenario == "missing" || scenario == "malformed" || scenario == "expired" || scenario == "wrong-scope" || scenario == "read-only" {
				wantRequests = 0
			}
			if requests.Load() != wantRequests {
				t.Fatalf("requests=%d want=%d", requests.Load(), wantRequests)
			}
			if scenario == "trusted" {
				if len(files) != 0 || responseCode.Load() != http.StatusCreated {
					t.Fatalf("trusted upload queue=%d HTTP=%d", len(files), responseCode.Load())
				}
				observer := installedWCNCPObserver(root, false)
				defer observer.Client.CloseIdleConnections()
				if status := observer.Status(); status.State != "OBSERVING" {
					t.Fatalf("authenticated status: %+v", status)
				}
			} else if len(files) != 1 {
				t.Fatalf("failed upload lost local observation: %v", files)
			}
			if scenario == "outage" {
				recovered.Store(true)
				if err := os.Remove(marker); err != nil {
					t.Fatal(err)
				}
				stdout.Reset()
				stderr.Reset()
				if code := Run([]string{"run", "--", stickyConnectionGradleCommand(root), "build"}, nil, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
					t.Fatalf("recovered native build failed: %d %s", code, stderr.String())
				}
				remaining, _ := filepath.Glob(filepath.Join(os.Getenv("WCNCP_OUTBOX_DIR"), "*", "obs-*.json"))
				if len(remaining) != 0 || requests.Load() != 2 || responseCode.Load() != http.StatusCreated || early.Load() != 0 {
					t.Fatalf("recovery did not acknowledge the queued batch: remaining=%d requests=%d status=%d", len(remaining), requests.Load(), responseCode.Load())
				}
			}
		})
	}
}

func TestStickyWCNCPDirectFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX child fixture")
	}
	for _, bypass := range []bool{false, true} {
		t.Run(fmt.Sprint(bypass), func(t *testing.T) {
			root := t.TempDir()
			if _, err := (stickywrapper.Generator{Root: root, Resolver: wcncpFixtureResolver{}}).Init(context.Background(), stickywrapper.DefaultConfig()); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".buildopt", "wrapper.properties"), []byte("invalid\n"), 0644); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\nif env | sed -n '/^WCNCP_/p;/^BUILDOPT_/p' | read -r private; then exit 77; fi\nprintf '<%s>\\n' \"$@\"\nexit 23\n"
			if err := os.WriteFile(filepath.Join(root, "gradlew"), []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("BUILDOPT_STICKY_OBSERVATION", "wcncp")
			t.Setenv("WCNCP_BACKEND_TOKEN", "must-not-reach-child")
			t.Setenv("WCNCP_OUTBOX_DIR", filepath.Join(root, "must-not-exist"))
			if bypass {
				t.Setenv("BUILDOPT_BYPASS", "1")
			}
			command := exec.Command(filepath.Join(root, "buildoptw"), "build", "", "two words", "$(exit 99)", "*")
			command.Dir = root
			raw, err := command.CombinedOutput()
			if err == nil || command.ProcessState.ExitCode() != 23 {
				t.Fatalf("direct native exit=%v stdout=%s", err, raw)
			}
			if !bytes.HasSuffix(raw, []byte("<build>\n<>\n<two words>\n<$(exit 99)>\n<*>\n")) {
				t.Fatalf("fallback changed arguments: %q", raw)
			}
			if _, err := os.Stat(os.Getenv("WCNCP_OUTBOX_DIR")); !os.IsNotExist(err) {
				t.Fatal("fallback created observation state")
			}
		})
	}
}

func TestStickyWCNCPDisabledAndUnavailableStateRetainsNative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX child fixture")
	}
	for _, scenario := range []string{"off", "bypass", "missing-config", "blocked-outbox", "full-outbox", "unhandled-signal"} {
		t.Run(scenario, func(t *testing.T) {
			root := writeStickyConnectionRepository(t, "https://127.0.0.1:1", "example/local-state", "BUILDOPT_TOKEN")
			outbox := t.TempDir()
			t.Setenv("BUILDOPT_STICKY_WRAPPER_ROOT", root)
			t.Setenv("BUILDOPT_STICKY_OBSERVATION", "wcncp")
			t.Setenv("BUILDOPT_TOKEN", "")
			t.Setenv("WCNCP_OUTBOX_DIR", outbox)
			script := "#!/bin/sh\nprintf native\\n\nexit 23\n"
			if scenario == "unhandled-signal" {
				script = "#!/bin/sh\nkill -TERM $$\n"
			}
			if err := os.WriteFile(stickyConnectionGradleCommand(root), []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(root, ".buildopt", "config.toml")
			switch scenario {
			case "off":
				raw, _ := os.ReadFile(configPath)
				if err := os.WriteFile(configPath, bytes.ReplaceAll(raw, []byte(`mode = "auto"`), []byte(`mode = "off"`)), 0644); err != nil {
					t.Fatal(err)
				}
			case "bypass":
				t.Setenv("BUILDOPT_BYPASS", "1")
			case "missing-config":
				if err := os.Remove(configPath); err != nil {
					t.Fatal(err)
				}
			case "blocked-outbox":
				blocked := filepath.Join(outbox, "file")
				if err := os.WriteFile(blocked, []byte("occupied"), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("WCNCP_OUTBOX_DIR", blocked)
			case "full-outbox":
				scoped := filepath.Join(outbox, optimizePortfolioRepositoryScope("example/local-state"))
				if err := os.MkdirAll(scoped, 0700); err != nil {
					t.Fatal(err)
				}
				exit := 0
				facts := wcncpobserve.BuildObservation("example/local-state", strings.Repeat("a", 64), func(string) string { return "" }, root, nil, []string{}, wcncpobserve.PassthroughResult{Child: wcncpobserve.ChildResult{Outcome: "SUCCESS", ExitCode: &exit}})
				raw, err := json.Marshal(facts)
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < wcncpobserve.OutboxMaxItems; i++ {
					path := filepath.Join(scoped, fmt.Sprintf("obs-%04d.json", i))
					if err := os.WriteFile(path, raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			var stdout, stderr bytes.Buffer
			code := Run([]string{"run", "--", stickyConnectionGradleCommand(root), "build"}, nil, &stdout, &stderr)
			expected := 23
			if scenario == "unhandled-signal" {
				expected = 143
			}
			if code != expected {
				t.Fatalf("native exit=%d want=%d stderr=%s", code, expected, stderr.String())
			}
			files, _ := filepath.Glob(filepath.Join(outbox, "*", "obs-*.json"))
			switch scenario {
			case "unhandled-signal":
				if len(files) != 1 {
					t.Fatalf("signal observation count=%d", len(files))
				}
				raw, _ := os.ReadFile(files[0])
				var facts wcncpobserve.ObservationFacts
				if err := json.Unmarshal(raw, &facts); err != nil || facts.Child.Signal == nil || *facts.Child.Signal != 15 || facts.Child.ExitCode != nil {
					t.Fatalf("unhandled signal observation incorrect: %s %v", raw, err)
				}
			case "full-outbox":
				if len(files) != wcncpobserve.OutboxMaxItems {
					t.Fatalf("outbox bound=%d", len(files))
				}
				if _, err := os.Stat(filepath.Join(outbox, optimizePortfolioRepositoryScope("example/local-state"), "obs-0000.json")); !os.IsNotExist(err) {
					t.Fatal("full outbox did not evict oldest item")
				}
			default:
				if len(files) != 0 {
					t.Fatalf("disabled/unavailable state recorded %d observations", len(files))
				}
			}
		})
	}
}
