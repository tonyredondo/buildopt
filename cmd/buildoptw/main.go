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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

const (
	wrapperScriptPOSIX   = "gradlew"
	wrapperScriptWindows = "gradlew.bat"
)

func main() {
	code := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, os.Getwd)
	if code < 0 {
		exitWithSignal(-code)
	}
	os.Exit(code)
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
	result := wcncpobserve.RunNativePassthrough(context.Background(), wrapperPath, args, cwd, stdin, stdout, stderr)
	code := childExitCode(result)
	if !observe {
		return code
	}
	// Observation starts only when configured and must avoid pre-child
	// network. Everything below runs after the child exits under a strict
	// post-child deadline and cannot replace the child result.
	observeAfterChild(repositoryScope, outboxDir, getenv, cwd, args, result)
	return code
}

func childExitCode(result wcncpobserve.PassthroughResult) int {
	if result.Child.ExitCode != nil {
		return *result.Child.ExitCode
	}
	if result.Child.Signal != nil {
		return -*result.Child.Signal
	}
	return 1
}

func runStatus(flag string, getenv func(string) string, stdout *os.File) int {
	outboxDir := getenv("WCNCP_OUTBOX_DIR")
	queued := 0
	if outboxDir != "" {
		items, err := (wcncpobserve.Outbox{Dir: outboxDir}).Pending(wcncpobserve.OutboxMaxItems, wcncpobserve.OutboxMaxBytes)
		if err == nil {
			queued = len(items)
		}
	}
	remoteState, remoteVerified := "", false
	if backend, scope := getenv("WCNCP_BACKEND_URL"), getenv("WCNCP_REPOSITORY_SCOPE"); backend != "" && scope != "" {
		if endpoint, err := wcncpobserve.Endpoint(backend, scope, "status"); err == nil {
			remoteState, remoteVerified = wcncpobserve.FetchStatus(context.Background(), http.DefaultClient, endpoint, getenv("WCNCP_BACKEND_TOKEN"), 100*time.Millisecond)
		}
	}
	status := wcncpobserve.DeriveStatus(queued, remoteState, remoteVerified)
	if flag == "--wcncp-explain" {
		_, _ = fmt.Fprintf(stdout, "buildoptw %s: %s\n", status.State, status.Detail)
	} else {
		_, _ = fmt.Fprintf(stdout, "%s\n", status.State)
	}
	return 0
}

func observeAfterChild(repositoryScope, outboxDir string, getenv func(string) string, cwd string, args []string, result wcncpobserve.PassthroughResult) {
	redacted, err := wcncpobserve.RedactArguments(repositoryScope, args)
	if err != nil {
		return
	}
	runnerID := loadOrCreateRunnerID(outboxDir)
	if runnerID == "" {
		return
	}
	facts := buildObservation(repositoryScope, runnerID, getenv, cwd, args, redacted, result)
	if err := facts.Validate(); err != nil {
		return
	}
	raw, err := json.Marshal(facts)
	if err != nil {
		return
	}
	outbox := wcncpobserve.Outbox{Dir: outboxDir}
	filename := fmt.Sprintf("obs-%d-%s.json", time.Now().UTC().UnixNano(), runnerID[:8])
	if err := outbox.Enqueue(filename, raw, time.Now().UTC()); err != nil {
		return
	}
	// Bounded upload attempt; outage leaves the item queued for later.
	if backend := getenv("WCNCP_BACKEND_URL"); backend != "" {
		endpoint, err := wcncpobserve.Endpoint(backend, repositoryScope, "WCNCP_OBSERVATION/batch")
		if err != nil {
			return
		}
		pending, err := outbox.Pending(32, 1<<20)
		if err != nil || len(pending) == 0 {
			return
		}
		batch := make([]json.RawMessage, 0, len(pending))
		for _, item := range pending {
			batch = append(batch, json.RawMessage(item.Raw))
		}
		outcome := wcncpobserve.UploadBatch(context.Background(), http.DefaultClient, endpoint, getenv("WCNCP_BACKEND_TOKEN"), batch, 100*time.Millisecond)
		if outcome.Uploaded == len(pending) {
			_ = outbox.Acknowledge(pending)
		}
	}
}

func buildObservation(repositoryScope, runnerID string, getenv func(string) string, cwd string, args, redacted []string, result wcncpobserve.PassthroughResult) wcncpobserve.ObservationFacts {
	now := time.Now().UTC()
	idempotencyKey := fmt.Sprintf("wcncp:%s:%d", runnerID[:16], now.UnixNano())
	class := environmentClass(getenv)
	bindings, complete := observationBindings(getenv, cwd, redacted)
	duration := wcncpobserve.TypedDuration{State: "UNAVAILABLE", Classification: "NOT_EVALUATED"}
	durationReason := "local functional duration is not evidence"
	duration.Reason = &durationReason
	if class == "STANDARD_HOSTED_CI" {
		value := result.Duration.Milliseconds()
		duration = wcncpobserve.TypedDuration{State: "COMPLETE", ValueMs: &value, Classification: "DIAGNOSTIC_ONLY"}
	} else if class == "CONTROLLED_PERFORMANCE" && complete && getenv("WCNCP_PROSPECTIVE_GATE_INPUT") == "1" {
		value := result.Duration.Milliseconds()
		duration = wcncpobserve.TypedDuration{State: "COMPLETE", ValueMs: &value, Classification: "CONTROLLED_VALUE_INPUT"}
	}
	outputManifest := wcncpobserve.OutputManifest{State: "UNAVAILABLE"}
	outputReason := "output manifest was not supplied by the frozen workflow package"
	outputManifest.Reason = &outputReason
	if value := getenv("WCNCP_OUTPUT_MANIFEST_SHA256"); isSHA256(value) {
		outputManifest = wcncpobserve.OutputManifest{State: "COMPLETE", SHA256: &value}
	} else {
		complete = false
	}
	facts := wcncpobserve.ObservationFacts{
		SchemaVersion: wcncpobserve.ObservationSchemaVersion, RecordType: "WCNCP_OBSERVATION",
		ObservationID: missingDigest("observation-id"), RepositoryScope: repositoryScope,
		RunnerID: runnerID, IdempotencyKey: idempotencyKey, InvocationOrdinal: now.UnixMilli(),
		EnvironmentClass: class, Bindings: bindings, Arguments: redacted, Duration: duration,
		ConfigurationCache: configCacheHint(args), BuildCacheMode: buildCacheHint(args),
		OutputManifest: outputManifest, Child: result.Child, Completeness: "INCOMPLETE",
		Authority: wcncpobserve.ObservationAuthority{ProductionAuthorized: false},
	}
	if complete {
		facts.Completeness = "COMPLETE"
		facts.Authority.ProspectiveGateInput = class == "CONTROLLED_PERFORMANCE" && getenv("WCNCP_PROSPECTIVE_GATE_INPUT") == "1"
	}
	identity, _ := json.Marshal(facts)
	facts.ObservationID = digestBytes(identity)
	return facts
}

func observationBindings(getenv func(string) string, cwd string, redacted []string) (wcncpobserve.ObservationBindings, bool) {
	complete := true
	revision := getenv("WCNCP_REPOSITORY_REVISION")
	if len(revision) != 40 {
		revision = gitOutput(cwd, "rev-parse", "HEAD")
		complete = false
	}
	if len(revision) != 40 {
		revision = strings.Repeat("0", 40)
	}
	sourceTree := configuredDigest(getenv, "WCNCP_SOURCE_TREE_SHA256", &complete)
	if sourceTree == "" {
		tree := gitOutput(cwd, "rev-parse", "HEAD^{tree}")
		status := gitOutput(cwd, "status", "--porcelain=v1", "--untracked-files=no")
		sourceTree = digestBytes([]byte(tree + "\x00" + status))
	}
	wrapperDigest := getenv("WCNCP_WRAPPER_SHA256")
	if !isSHA256(wrapperDigest) {
		wrapperDigest = digestFile(filepath.Join(cwd, wrapperScriptForPlatform()))
		complete = false
	}
	if !isSHA256(wrapperDigest) {
		wrapperDigest = missingDigest("wrapper")
	}
	gradleVersion := getenv("WCNCP_GRADLE_VERSION")
	if gradleVersion == "" || len(gradleVersion) > 32 {
		gradleVersion = "UNAVAILABLE"
		complete = false
	}
	packageDigest := getenv("WCNCP_PACKAGE_SHA256")
	if !isSHA256(packageDigest) {
		if executable, err := os.Executable(); err == nil {
			packageDigest = digestFile(executable)
		}
		complete = false
	}
	if !isSHA256(packageDigest) {
		packageDigest = missingDigest("buildopt-package")
	}
	workflowRaw, _ := json.Marshal(struct {
		Arguments []string `json:"arguments"`
	}{Arguments: redacted})
	workflowDigest := configuredDigest(getenv, "WCNCP_WORKFLOW_SHA256", &complete)
	if workflowDigest == "" {
		workflowDigest = digestBytes(workflowRaw)
	}
	environmentDigest := configuredDigest(getenv, "WCNCP_ENVIRONMENT_SHA256", &complete)
	if environmentDigest == "" {
		environmentDigest = digestBytes([]byte(runtime.GOOS + "\x00" + runtime.GOARCH + "\x00" + getenv("JAVA_HOME")))
	}
	return wcncpobserve.ObservationBindings{
		RepositoryRevision: revision, SourceTreeSHA256: sourceTree, WrapperSHA256: wrapperDigest,
		GradleVersion: gradleVersion, JDKSHA256: configuredDigestOrMissing(getenv, "WCNCP_JDK_SHA256", "jdk", &complete),
		BuildOptPackageSHA256: packageDigest, WorkflowSHA256: workflowDigest, EnvironmentSHA256: environmentDigest,
		OutputContractSHA256: configuredDigestOrMissing(getenv, "WCNCP_OUTPUT_CONTRACT_SHA256", "output-contract", &complete),
	}, complete
}

func configuredDigest(getenv func(string) string, key string, complete *bool) string {
	value := getenv(key)
	if isSHA256(value) {
		return value
	}
	*complete = false
	return ""
}

func configuredDigestOrMissing(getenv func(string) string, key, marker string, complete *bool) string {
	if value := configuredDigest(getenv, key, complete); value != "" {
		return value
	}
	return missingDigest(marker)
}

func missingDigest(marker string) string {
	return digestBytes([]byte("WCNCP_UNAVAILABLE\x00" + marker))
}

func digestBytes(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func digestFile(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func gitOutput(cwd string, args ...string) string {
	command := exec.Command("git", args...)
	command.Dir = cwd
	raw, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func wrapperScriptForPlatform() string {
	if os.PathSeparator == '\\' {
		return wrapperScriptWindows
	}
	return wrapperScriptPOSIX
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
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			if existing, readErr := os.ReadFile(path); readErr == nil && len(existing) == 64 {
				return string(existing)
			}
		}
		return ""
	}
	if _, err := file.WriteString(id); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return ""
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
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
