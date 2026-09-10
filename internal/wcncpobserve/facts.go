package wcncpobserve

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildObservation constructs private facts; missing frozen bindings never grant authority.
func BuildObservation(repositoryScope, runnerID string, getenv func(string) string, cwd string, args, redacted []string, result PassthroughResult) ObservationFacts {
	now := time.Now().UTC()
	idempotencyKey := fmt.Sprintf("wcncp:%s:%d", runnerID[:16], now.UnixNano())
	class := environmentClass(getenv)
	bindings, complete := observationBindings(getenv, cwd, redacted)
	outputManifest := OutputManifest{State: "UNAVAILABLE"}
	outputReason := "output manifest was not supplied by the frozen workflow package"
	outputManifest.Reason = &outputReason
	if value := getenv("WCNCP_OUTPUT_MANIFEST_SHA256"); isSHA256(value) {
		outputManifest = OutputManifest{State: "COMPLETE", SHA256: &value}
	} else {
		complete = false
	}
	duration := TypedDuration{State: "UNAVAILABLE", Classification: "NOT_EVALUATED"}
	durationReason := "local functional duration is not evidence"
	duration.Reason = &durationReason
	if class == "STANDARD_HOSTED_CI" {
		value := result.Duration.Milliseconds()
		duration = TypedDuration{State: "COMPLETE", ValueMs: &value, Classification: "DIAGNOSTIC_ONLY"}
	} else if class == "CONTROLLED_PERFORMANCE" && complete && getenv("WCNCP_PROSPECTIVE_GATE_INPUT") == "1" {
		value := result.Duration.Milliseconds()
		duration = TypedDuration{State: "COMPLETE", ValueMs: &value, Classification: "CONTROLLED_VALUE_INPUT"}
	}
	facts := ObservationFacts{
		SchemaVersion: ObservationSchemaVersion, RecordType: "WCNCP_OBSERVATION",
		ObservationID: missingDigest("observation-id"), RepositoryScope: repositoryScope,
		RunnerID: runnerID, IdempotencyKey: idempotencyKey, InvocationOrdinal: now.UnixMilli(),
		EnvironmentClass: class, Bindings: bindings, Arguments: redacted, Duration: duration,
		ConfigurationCache: configCacheHint(args), BuildCacheMode: buildCacheHint(args),
		OutputManifest: outputManifest, Child: result.Child, Completeness: "INCOMPLETE",
		Authority: ObservationAuthority{ProductionAuthorized: false},
	}
	if complete {
		facts.Completeness = "COMPLETE"
		facts.Authority.ProspectiveGateInput = class == "CONTROLLED_PERFORMANCE" && getenv("WCNCP_PROSPECTIVE_GATE_INPUT") == "1"
	}
	identity, _ := json.Marshal(facts)
	facts.ObservationID = digestBytes(identity)
	return facts
}

func observationBindings(getenv func(string) string, cwd string, redacted []string) (ObservationBindings, bool) {
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
	return ObservationBindings{
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
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.WaitDelay = 50 * time.Millisecond
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
		return "gradlew.bat"
	}
	return "gradlew"
}

func loadOrCreateRunnerID(outboxDir string) string {
	path := filepath.Join(outboxDir, "runner.id")
	if raw, err := os.ReadFile(path); err == nil && isSHA256(string(raw)) {
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
			if existing, readErr := os.ReadFile(path); readErr == nil && isSHA256(string(existing)) {
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

// Flags express intent, not whether Gradle stored or reused a cache entry.
// Without an explicit flag, gradle.properties can still enable either cache.
func configCacheHint(args []string) string {
	state := "UNAVAILABLE"
	for _, arg := range args {
		switch arg {
		case "--configuration-cache":
			state = "UNAVAILABLE"
		case "--no-configuration-cache":
			state = "NOT_REQUESTED"
		}
	}
	return state
}

func buildCacheHint(args []string) string {
	state := "UNAVAILABLE"
	for _, arg := range args {
		switch arg {
		case "--build-cache":
			state = "ENABLED"
		case "--no-build-cache":
			state = "DISABLED"
		}
	}
	return state
}
