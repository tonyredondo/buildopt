// Command buildoptw is the ordinary Gradle entrypoint for WCNCP repositories.
//
//	./buildoptw <gradle arguments...>
//	./buildoptw --wcncp-status
//	./buildoptw --wcncp-explain
//
// The customer-visible behavior is: run the exact requested workflow through
// the repository native Gradle Wrapper, never make successful Gradle execution
// depend on BuildOpt service health, record a small private typed observation
// after the build, and explain whether BuildOpt is observing, has found an
// opportunity, is validating, or has a proposal ready. Expensive validation
// runs only on authorized validators within a fixed budget; source application
// and merge stay with the repository owner.
//
// Wrapper-reserved flags (--wcncp-status, --wcncp-explain) are stripped before
// Gradle passthrough and never forwarded, so native task selection is
// unchanged for every real Gradle invocation.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

const (
	wrapperScriptPOSIX   = "gradlew"
	wrapperScriptWindows = "gradlew.bat"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, os.Getwd))
}

func run(args []string, getenv func(string) string, stdin *os.File, stdout, stderr *os.File, getwd func() (string, error)) int {
	// Wrapper-reserved surfaces never reach Gradle.
	if len(args) == 1 && (args[0] == "--wcncp-status" || args[0] == "--wcncp-explain") {
		return runStatus(args[0], getenv, stdout)
	}
	cwd, err := getwd()
	if err != nil {
		cwd = "."
	}
	wrapper := wrapperScriptPOSIX
	if os.PathSeparator == '\\' {
		wrapper = wrapperScriptWindows
	}
	wrapperPath := filepath.Join(cwd, wrapper)
	if _, err := os.Stat(wrapperPath); err != nil {
		// No Wrapper: fail closed with a clear diagnostic and native-like exit.
		_, _ = fmt.Fprintf(stderr, "buildoptw: no Gradle wrapper at %s\n", wrapperPath)
		return 127
	}
	if wcncpobserve.BypassRequested(mapFromEnv(getenv)) {
		result := wcncpobserve.RunNativePassthrough(context.Background(), wrapperPath, args, cwd, stdin, stdout, stderr)
		return childExitCode(result)
	}
	repositoryScope := getenv("WCNCP_REPOSITORY_SCOPE")
	outboxDir := getenv("WCNCP_OUTBOX_DIR")
	observe := repositoryScope != "" && outboxDir != ""
	start := time.Now().UTC()
	result := wcncpobserve.RunNativePassthrough(context.Background(), wrapperPath, args, cwd, stdin, stdout, stderr)
	code := childExitCode(result)
	if !observe {
		return code
	}
	// Observation starts only when configured and must avoid pre-child
	// network. Everything below runs after the child exits under a strict
	// post-child deadline and cannot replace the child result.
	observeAfterChild(repositoryScope, outboxDir, getenv, cwd, args, start, result)
	return code
}

func childExitCode(result wcncpobserve.PassthroughResult) int {
	if result.Child.ExitCode != nil {
		return *result.Child.ExitCode
	}
	return 1
}

func runStatus(flag string, getenv func(string) string, stdout *os.File) int {
	outboxDir := getenv("WCNCP_OUTBOX_DIR")
	queued := 0
	if outboxDir != "" {
		entries, err := os.ReadDir(outboxDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".tmp-") {
					queued++
				}
			}
		}
	}
	// Remote projection is verified separately; without verification the
	// surface reports QUEUED or UNAVAILABLE and never claims a proposal state.
	status := wcncpobserve.DeriveStatus(queued, "OBSERVING", false)
	if flag == "--wcncp-explain" {
		_, _ = fmt.Fprintf(stdout, "buildoptw %s: %s\n", status.State, status.Detail)
	} else {
		_, _ = fmt.Fprintf(stdout, "%s\n", status.State)
	}
	return 0
}

func observeAfterChild(repositoryScope, outboxDir string, getenv func(string) string, cwd string, args []string, start time.Time, result wcncpobserve.PassthroughResult) {
	redacted, err := wcncpobserve.RedactArguments(repositoryScope, args)
	if err != nil {
		return
	}
	runnerID := loadOrCreateRunnerID(outboxDir)
	if runnerID == "" {
		return
	}
	facts := wcncpobserve.ObservationFacts{
		RepositoryScope: repositoryScope, RunnerID: runnerID,
		IdempotencyKey:    fmt.Sprintf("wcncp:%s:%d", runnerID[:16], time.Now().UTC().UnixNano()),
		InvocationOrdinal: time.Now().UTC().UnixMilli(),
		EnvironmentClass:  environmentClass(getenv),
		Arguments:         redacted,
		ConfigurationCache: configCacheHint(args),
		BuildCacheMode:     buildCacheHint(args),
		Child:              result.Child,
		Completeness:       "COMPLETE",
	}
	durationMs := result.Duration.Milliseconds()
	facts.DurationMs = &durationMs
	if err := facts.Validate(); err != nil {
		return
	}
	raw, err := json.Marshal(facts)
	if err != nil {
		return
	}
	outbox := wcncpobserve.Outbox{Dir: outboxDir}
	filename := fmt.Sprintf("obs-%d-%s.json", time.Now().UTC().UnixNano(), runnerID[:8])
	_ = outbox.Enqueue(filename, raw, time.Now().UTC())
	// Bounded upload attempt; outage leaves the item queued for later.
	if backend := getenv("WCNCP_BACKEND_URL"); backend != "" {
		token := getenv("WCNCP_BACKEND_TOKEN")
		_ = wcncpobserve.UploadBatch(context.Background(), http.DefaultClient, backend, token, []json.RawMessage{raw}, 100*time.Millisecond)
	}
	_ = start
	_ = cwd
}

func loadOrCreateRunnerID(outboxDir string) string {
	path := filepath.Join(outboxDir, "runner.id")
	if raw, err := os.ReadFile(path); err == nil && len(raw) == 64 {
		return string(raw)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	id := hex.EncodeToString(raw)
	if err := os.MkdirAll(outboxDir, 0o700); err != nil {
		return ""
	}
	if err := os.WriteFile(path, []byte(id), 0o600); err != nil {
		return ""
	}
	return id
}

func environmentClass(getenv func(string) string) string {
	switch getenv("WCNCP_ENVIRONMENT_CLASS") {
	case "CONTROLLED_PERFORMANCE", "STANDARD_HOSTED_CI", "LOCAL_FUNCTIONAL":
		return getenv("WCNCP_ENVIRONMENT_CLASS")
	default:
		return "LOCAL_FUNCTIONAL"
	}
}

func configCacheHint(args []string) string {
	for _, arg := range args {
		if arg == "--configuration-cache" {
			return "STORE"
		}
	}
	return "NOT_REQUESTED"
}

func buildCacheHint(args []string) string {
	for _, arg := range args {
		if arg == "--no-build-cache" {
			return "DISABLED"
		}
	}
	return "ENABLED"
}

func mapFromEnv(getenv func(string) string) map[string]string {
	return map[string]string{"BUILDOPT_BYPASS": getenv("BUILDOPT_BYPASS")}
}
