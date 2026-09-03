// Package wcncpobserve owns the WCNCP-003 wrapper observation path. It runs
// after the existing native fast path without changing Gradle task selection,
// collects bounded finalized-build facts, writes one private atomic outbox
// item after the child exits, and attempts a bounded batch upload. Backend
// loss preserves the Gradle result; observation or upload failure never
// replaces the child exit code or signal.
package wcncpobserve

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// Outbox bounds from the frozen contract: 100 MiB, 1,000 items, 30 days.
	OutboxMaxBytes = 100 << 20
	OutboxMaxItems = 1000
	OutboxMaxAge  = 30 * 24 * time.Hour
	// PostChildUploadDeadlineMs bounds wrapper work after the child exits.
	PostChildUploadDeadlineMs = 100
	// ObservationSchemaVersion is the artifact payload version stored in the
	// control plane for WCNCP observations built by this package.
	ObservationSchemaVersion = "buildopt.wcncp/observation/v1"
)

var (
	// ErrObservationInvalid means caller inputs violate the privacy or
	// shape contract and must not be uploaded.
	ErrObservationInvalid = errors.New("BuildOpt WCNCP observation is invalid")
	// ErrOutboxFull means the bounded outbox evicted the oldest item.
	ErrOutboxFull = errors.New("BuildOpt WCNCP outbox is full")
)

var (
	safeTaskPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{0,127}$`)
	safeFlagPattern   = regexp.MustCompile(`^--[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	secretHintPattern = regexp.MustCompile(`(?i)(token|secret|password|passwd|pwd|key|credential|auth|bearer|cookie|session)`)
	hexDigestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Known Gradle task names and safe structural flags may be stored literally.
// Everything else is redacted or hashed with a repository-scoped
// representation. Secret-looking tokens are never uploaded.
var safeStructuralFlags = map[string]bool{
	"--configuration-cache": true, "--build-cache": true, "--parallel": true,
	"--offline": true, "--refresh-dependencies": true, "--continue": true,
	"--rerun-tasks": true, "--info": true, "--stacktrace": true,
	"--console": true, "--warning-mode": true, "--max-workers": true,
	"--profile": true, "--scan": true,
}

// RedactArguments returns a privacy-safe representation of ordered Gradle
// arguments. Known tasks and safe flags pass through; values are hashed with
// the repository scope; secret-looking tokens become REDACTED_SECRET and are
// never hashed or uploaded in any reversible form.
func RedactArguments(repositoryScope string, args []string) ([]string, error) {
	if len(repositoryScope) == 0 || len(repositoryScope) > 256 {
		return nil, ErrObservationInvalid
	}
	if len(args) > 128 {
		return nil, ErrObservationInvalid
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if len(arg) == 0 || len(arg) > 256 {
			return nil, ErrObservationInvalid
		}
		if secretHintPattern.MatchString(arg) {
			out = append(out, "REDACTED_SECRET")
			continue
		}
		if safeStructuralFlags[arg] {
			out = append(out, arg)
			continue
		}
		if strings.HasPrefix(arg, "--") {
			// Split --flag=value; flag allowlisted, value hashed.
			if index := strings.Index(arg, "="); index > 0 {
				flag, value := arg[:index], arg[index+1:]
				if safeFlagPattern.MatchString(flag) && safeStructuralFlags[flag] {
					out = append(out, flag+"="+hashArgument(repositoryScope, value))
					continue
				}
			}
			if safeFlagPattern.MatchString(arg) {
				// Unknown flag shape: hash to avoid leaking values in flags.
				out = append(out, hashArgument(repositoryScope, arg))
				continue
			}
			out = append(out, hashArgument(repositoryScope, arg))
			continue
		}
		if safeTaskPattern.MatchString(arg) && !strings.Contains(arg, "/") && !strings.Contains(arg, "\\") {
			// Task-looking tokens without path separators pass through when
			// they do not resemble secrets. This preserves debuggability
			// without allowing names to become classification inputs.
			out = append(out, arg)
			continue
		}
		out = append(out, hashArgument(repositoryScope, arg))
	}
	return out, nil
}

func hashArgument(repositoryScope, value string) string {
	digest := sha256.Sum256([]byte(repositoryScope + "\x00" + value))
	return "hash:" + hex.EncodeToString(digest[:16])
}

// RedactPath returns a repository-relative path when the absolute path sits
// under the repository root; otherwise it classifies and omits. Usernames,
// home directories, and machine roots never enter observations.
func RedactPath(repositoryRoot, path string) string {
	if repositoryRoot == "" || path == "" {
		return "OMITTED"
	}
	cleanRoot := filepath.Clean(repositoryRoot)
	cleanPath := filepath.Clean(path)
	relative, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "OMITTED"
	}
	// Normalize to forward slashes for portable evidence.
	return filepath.ToSlash(relative)
}

// ChildResult is the authoritative native Gradle outcome. Exit code or signal
// remains authoritative after the child starts.
type ChildResult struct {
	Outcome  string
	ExitCode *int
	Signal   *int
}

// ObservationFacts are the bounded finalized-build facts recorded after the
// child exits. No pre-child network occurs; upload happens only after the
// child completes under a strict deadline.
type ObservationFacts struct {
	RepositoryScope    string
	RunnerID           string
	IdempotencyKey     string
	InvocationOrdinal  int64
	EnvironmentClass   string
	RepositoryRevision string
	SourceTreeSHA256   string
	WrapperSHA256      string
	GradleVersion      string
	JDKSHA256          string
	PackageSHA256      string
	WorkflowSHA256     string
	EnvironmentSHA256  string
	OutputContractSHA256 string
	Arguments          []string
	DurationMs         *int64
	DurationReason     *string
	ConfigurationCache string
	BuildCacheMode     string
	OutputManifestSHA256 *string
	OutputUnavailableReason *string
	Child              ChildResult
	Completeness       string
}

// Validate enforces shape and privacy before any persistence or upload.
func (facts ObservationFacts) Validate() error {
	if len(facts.RepositoryScope) == 0 || len(facts.RepositoryScope) > 256 ||
		!hexDigestPattern.MatchString(facts.RunnerID) ||
		len(facts.IdempotencyKey) < 16 || len(facts.IdempotencyKey) > 128 ||
		facts.InvocationOrdinal < 1 {
		return ErrObservationInvalid
	}
	if facts.EnvironmentClass != "CONTROLLED_PERFORMANCE" && facts.EnvironmentClass != "STANDARD_HOSTED_CI" && facts.EnvironmentClass != "LOCAL_FUNCTIONAL" {
		return ErrObservationInvalid
	}
	if len(facts.Arguments) > 128 {
		return ErrObservationInvalid
	}
	for _, arg := range facts.Arguments {
		if len(arg) == 0 || len(arg) > 256 || secretHintPattern.MatchString(arg) {
			return ErrObservationInvalid
		}
		// Raw absolute paths must never reach the backend.
		if filepath.IsAbs(arg) || strings.Contains(arg, string(filepath.Separator)+"home"+string(filepath.Separator)) {
			return ErrObservationInvalid
		}
	}
	if facts.Child.Outcome != "SUCCESS" && facts.Child.Outcome != "FAILED" && facts.Child.Outcome != "SIGNALED" && facts.Child.Outcome != "NOT_STARTED" {
		return ErrObservationInvalid
	}
	return nil
}

// Status is the verified local/remote projection for status and explain. It
// distinguishes UNAVAILABLE, QUEUED, OBSERVING, and proposal states without
// becoming an authorization path.
type Status struct {
	State  string
	Detail string
}

// DeriveStatus maps verified local outbox depth and remote projection to a
// customer-visible state.
func DeriveStatus(outboxQueued int, remoteState string, remoteVerified bool) Status {
	if !remoteVerified {
		if outboxQueued > 0 {
			return Status{State: "QUEUED", Detail: "observations queued locally; backend unavailable or unverified"}
		}
		return Status{State: "UNAVAILABLE", Detail: "no verified projection available"}
	}
	switch remoteState {
	case "", "OBSERVING":
		if outboxQueued > 0 {
			return Status{State: "QUEUED", Detail: "local items queued behind verified observing state"}
		}
		return Status{State: "OBSERVING", Detail: "observing native builds"}
	case "OPPORTUNITY_DETECTED", "VALIDATION_QUEUED", "VALIDATING", "REVIEW_READY", "OWNER_ACCEPTED":
		return Status{State: remoteState, Detail: "derived from verified remote projection"}
	default:
		return Status{State: "OBSERVING", Detail: "unknown remote state retains observing"}
	}
}

// Outbox is a private local durable queue using atomic replacement. Upload
// happens only after the Gradle child completes. Eviction is oldest-first and
// emits a local diagnostic; it never fabricates a successful upload.
type Outbox struct {
	Dir string
}

// Enqueue writes one item atomically and enforces count, byte, and age
// bounds. Duplicate retry uses the original idempotency key; the caller owns
// deduplication by filename.
func (outbox Outbox) Enqueue(filename string, content []byte, now time.Time) error {
	if outbox.Dir == "" || len(filename) == 0 || len(content) == 0 {
		return ErrObservationInvalid
	}
	if err := os.MkdirAll(outbox.Dir, 0o700); err != nil {
		return err
	}
	if err := outbox.enforceBounds(now); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(outbox.Dir, ".tmp-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporaryName)
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryName)
		return err
	}
	if err := os.Chmod(temporaryName, 0o600); err != nil {
		_ = os.Remove(temporaryName)
		return err
	}
	return os.Rename(temporaryName, filepath.Join(outbox.Dir, filename))
}

func (outbox Outbox) enforceBounds(now time.Time) error {
	entries, err := os.ReadDir(outbox.Dir)
	if err != nil {
		return err
	}
	var totalBytes int64
	var oldestName string
	var oldestTime time.Time
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".tmp-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		count++
		totalBytes += info.Size()
		modTime := info.ModTime()
		if oldestName == "" || modTime.Before(oldestTime) {
			oldestName, oldestTime = entry.Name(), modTime
		}
		// Age eviction inline: remove items older than 30 days.
		if now.Sub(modTime) > OutboxMaxAge {
			_ = os.Remove(filepath.Join(outbox.Dir, entry.Name()))
			count--
			totalBytes -= info.Size()
		}
	}
	if count >= OutboxMaxItems || totalBytes >= OutboxMaxBytes {
		if oldestName != "" {
			_ = os.Remove(filepath.Join(outbox.Dir, oldestName))
		}
		return nil
	}
	return nil
}
