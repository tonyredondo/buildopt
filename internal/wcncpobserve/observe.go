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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	// Outbox bounds from the frozen contract: 100 MiB, 1,000 items, 30 days.
	OutboxMaxBytes = 100 << 20
	OutboxMaxItems = 1000
	OutboxMaxAge   = 30 * 24 * time.Hour
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
	revisionPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
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
	Outcome  string `json:"outcome"`
	ExitCode *int   `json:"exitCode,omitempty"`
	Signal   *int   `json:"signal,omitempty"`
}

// ObservationBindings identify the exact source, toolchain, workflow, and
// output contract observed by one native invocation. Missing facts use a
// typed sentinel digest and force Completeness to INCOMPLETE; they are never
// silently treated as prospective evidence.
type ObservationBindings struct {
	RepositoryRevision    string `json:"repositoryRevision"`
	SourceTreeSHA256      string `json:"sourceTreeSha256"`
	WrapperSHA256         string `json:"wrapperSha256"`
	GradleVersion         string `json:"gradleVersion"`
	JDKSHA256             string `json:"jdkSha256"`
	BuildOptPackageSHA256 string `json:"buildoptPackageSha256"`
	WorkflowSHA256        string `json:"workflowSha256"`
	EnvironmentSHA256     string `json:"environmentSha256"`
	OutputContractSHA256  string `json:"outputContractSha256"`
}

// TypedDuration prevents local or hosted wall time from becoming controlled
// value input merely because a numeric duration exists.
type TypedDuration struct {
	State          string  `json:"state"`
	ValueMs        *int64  `json:"valueMs,omitempty"`
	Classification string  `json:"classification"`
	Reason         *string `json:"reason,omitempty"`
}

// OutputManifest records either an exact digest or why the output contract
// was unavailable for this observation.
type OutputManifest struct {
	State  string  `json:"state"`
	SHA256 *string `json:"sha256,omitempty"`
	Reason *string `json:"reason,omitempty"`
}

// ObservationAuthority is data, not ambient permission. A wrapper may mark a
// row as prospective only when its complete controlled bindings were supplied
// by the frozen experiment package.
type ObservationAuthority struct {
	ProspectiveGateInput bool `json:"prospectiveGateInput"`
	ProductionAuthorized bool `json:"productionAuthorized"`
}

// ObservationFacts exactly matches wcncp-observation.v1.schema.json. Keeping
// this wire owner beside the wrapper prevents a second, incompatible ad-hoc
// JSON shape from reaching the central batch endpoint.
type ObservationFacts struct {
	SchemaVersion      string               `json:"schemaVersion"`
	RecordType         string               `json:"recordType"`
	ObservationID      string               `json:"observationId"`
	RepositoryScope    string               `json:"repositoryScope"`
	RunnerID           string               `json:"runnerId"`
	IdempotencyKey     string               `json:"idempotencyKey"`
	InvocationOrdinal  int64                `json:"invocationOrdinal"`
	EnvironmentClass   string               `json:"environmentClass"`
	Bindings           ObservationBindings  `json:"bindings"`
	Arguments          []string             `json:"arguments"`
	Duration           TypedDuration        `json:"duration"`
	ConfigurationCache string               `json:"configurationCache"`
	BuildCacheMode     string               `json:"buildCacheMode"`
	OutputManifest     OutputManifest       `json:"outputManifest"`
	Child              ChildResult          `json:"child"`
	Completeness       string               `json:"completeness"`
	Authority          ObservationAuthority `json:"authority"`
}

// Validate enforces shape and privacy before any persistence or upload.
func (facts ObservationFacts) Validate() error {
	if facts.SchemaVersion != ObservationSchemaVersion || facts.RecordType != "WCNCP_OBSERVATION" ||
		!hexDigestPattern.MatchString(facts.ObservationID) ||
		len(facts.RepositoryScope) == 0 || len(facts.RepositoryScope) > 256 ||
		!hexDigestPattern.MatchString(facts.RunnerID) ||
		len(facts.IdempotencyKey) < 16 || len(facts.IdempotencyKey) > 128 ||
		facts.InvocationOrdinal < 1 {
		return ErrObservationInvalid
	}
	if facts.EnvironmentClass != "CONTROLLED_PERFORMANCE" && facts.EnvironmentClass != "STANDARD_HOSTED_CI" && facts.EnvironmentClass != "LOCAL_FUNCTIONAL" {
		return ErrObservationInvalid
	}
	bindings := facts.Bindings
	if !revisionPattern.MatchString(bindings.RepositoryRevision) ||
		!hexDigestPattern.MatchString(bindings.SourceTreeSHA256) ||
		!hexDigestPattern.MatchString(bindings.WrapperSHA256) ||
		len(bindings.GradleVersion) == 0 || len(bindings.GradleVersion) > 32 ||
		!hexDigestPattern.MatchString(bindings.JDKSHA256) ||
		!hexDigestPattern.MatchString(bindings.BuildOptPackageSHA256) ||
		!hexDigestPattern.MatchString(bindings.WorkflowSHA256) ||
		!hexDigestPattern.MatchString(bindings.EnvironmentSHA256) ||
		!hexDigestPattern.MatchString(bindings.OutputContractSHA256) {
		return ErrObservationInvalid
	}
	if len(facts.Arguments) > 128 {
		return ErrObservationInvalid
	}
	for _, arg := range facts.Arguments {
		if len(arg) == 0 || len(arg) > 256 {
			return ErrObservationInvalid
		}
		// Redaction markers are the only allowed secret-derived outputs: they
		// prove a secret-looking input was seen without carrying its value.
		if arg == "REDACTED_SECRET" || arg == "OMITTED" || strings.HasPrefix(arg, "hash:") {
			continue
		}
		if secretHintPattern.MatchString(arg) {
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
	if facts.Duration.State == "COMPLETE" {
		if facts.Duration.ValueMs == nil || *facts.Duration.ValueMs < 0 || facts.Duration.Reason != nil {
			return ErrObservationInvalid
		}
	} else if facts.Duration.State == "UNAVAILABLE" {
		if facts.Duration.ValueMs != nil || facts.Duration.Reason == nil || *facts.Duration.Reason == "" {
			return ErrObservationInvalid
		}
	} else {
		return ErrObservationInvalid
	}
	switch facts.EnvironmentClass {
	case "CONTROLLED_PERFORMANCE":
		if facts.Duration.Classification != "CONTROLLED_VALUE_INPUT" && facts.Duration.Classification != "NOT_EVALUATED" {
			return ErrObservationInvalid
		}
	case "STANDARD_HOSTED_CI":
		if facts.Duration.Classification != "DIAGNOSTIC_ONLY" {
			return ErrObservationInvalid
		}
	case "LOCAL_FUNCTIONAL":
		if facts.Duration.Classification != "NOT_EVALUATED" {
			return ErrObservationInvalid
		}
	}
	if facts.ConfigurationCache != "NOT_REQUESTED" && facts.ConfigurationCache != "STORE" && facts.ConfigurationCache != "REUSE" && facts.ConfigurationCache != "PROBLEM" && facts.ConfigurationCache != "UNAVAILABLE" {
		return ErrObservationInvalid
	}
	if facts.BuildCacheMode != "ENABLED" && facts.BuildCacheMode != "DISABLED" && facts.BuildCacheMode != "UNAVAILABLE" {
		return ErrObservationInvalid
	}
	if facts.OutputManifest.State == "COMPLETE" {
		if facts.OutputManifest.SHA256 == nil || !hexDigestPattern.MatchString(*facts.OutputManifest.SHA256) || facts.OutputManifest.Reason != nil {
			return ErrObservationInvalid
		}
	} else if facts.OutputManifest.State == "UNAVAILABLE" {
		if facts.OutputManifest.SHA256 != nil || facts.OutputManifest.Reason == nil || *facts.OutputManifest.Reason == "" {
			return ErrObservationInvalid
		}
	} else {
		return ErrObservationInvalid
	}
	if facts.Completeness != "COMPLETE" && facts.Completeness != "INCOMPLETE" {
		return ErrObservationInvalid
	}
	if facts.Authority.ProductionAuthorized || (facts.Authority.ProspectiveGateInput && (facts.Completeness != "COMPLETE" || facts.EnvironmentClass != "CONTROLLED_PERFORMANCE")) {
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
	if !strings.HasPrefix(filename, "obs-") || !strings.HasSuffix(filename, ".json") || strings.ContainsAny(filename, `/\\`) || len(content) > OutboxMaxBytes {
		return ErrObservationInvalid
	}
	if err := os.MkdirAll(outbox.Dir, 0o700); err != nil {
		return err
	}
	if err := outbox.enforceBounds(now, int64(len(content))); err != nil {
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

type queuedItem struct {
	name    string
	size    int64
	modTime time.Time
}

func (outbox Outbox) enforceBounds(now time.Time, incomingBytes int64) error {
	entries, err := os.ReadDir(outbox.Dir)
	if err != nil {
		return err
	}
	var totalBytes int64
	items := make([]queuedItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "obs-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime()
		if now.Sub(modTime) > OutboxMaxAge {
			_ = os.Remove(filepath.Join(outbox.Dir, entry.Name()))
			continue
		}
		items = append(items, queuedItem{name: entry.Name(), size: info.Size(), modTime: modTime})
		totalBytes += info.Size()
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].modTime.Equal(items[j].modTime) {
			return items[i].name < items[j].name
		}
		return items[i].modTime.Before(items[j].modTime)
	})
	for len(items) >= OutboxMaxItems || totalBytes+incomingBytes > OutboxMaxBytes {
		if len(items) == 0 {
			return ErrOutboxFull
		}
		oldest := items[0]
		if err := os.Remove(filepath.Join(outbox.Dir, oldest.name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("evict WCNCP observation: %w", err)
		}
		totalBytes -= oldest.size
		items = items[1:]
	}
	return nil
}

// QueuedObservation is one durable outbox item selected for a bounded upload.
type QueuedObservation struct {
	Name string
	Raw  []byte
}

// Pending returns oldest-first observation items without treating runner.id
// or temporary files as queue entries.
func (outbox Outbox) Pending(maxItems int, maxBytes int64) ([]QueuedObservation, error) {
	if maxItems <= 0 || maxBytes <= 0 {
		return nil, ErrObservationInvalid
	}
	entries, err := os.ReadDir(outbox.Dir)
	if os.IsNotExist(err) {
		return []QueuedObservation{}, nil
	}
	if err != nil {
		return nil, err
	}
	type candidate struct {
		name string
		mod  time.Time
	}
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "obs-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err == nil {
			candidates = append(candidates, candidate{name: entry.Name(), mod: info.ModTime()})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].mod.Equal(candidates[j].mod) {
			return candidates[i].name < candidates[j].name
		}
		return candidates[i].mod.Before(candidates[j].mod)
	})
	selected := make([]QueuedObservation, 0, min(maxItems, len(candidates)))
	var selectedBytes int64
	for _, candidate := range candidates {
		if len(selected) == maxItems {
			break
		}
		raw, err := os.ReadFile(filepath.Join(outbox.Dir, candidate.name))
		if err != nil {
			return nil, err
		}
		if selectedBytes+int64(len(raw)) > maxBytes {
			break
		}
		selected = append(selected, QueuedObservation{Name: candidate.name, Raw: raw})
		selectedBytes += int64(len(raw))
	}
	return selected, nil
}

// Acknowledge removes only the exact successfully published queue entries.
func (outbox Outbox) Acknowledge(items []QueuedObservation) error {
	for _, item := range items {
		if !strings.HasPrefix(item.Name, "obs-") || !strings.HasSuffix(item.Name, ".json") || strings.ContainsAny(item.Name, `/\\`) {
			return ErrObservationInvalid
		}
		if err := os.Remove(filepath.Join(outbox.Dir, item.Name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
