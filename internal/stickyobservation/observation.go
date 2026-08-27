// Package stickyobservation records bounded facts from ordinary sticky-wrapper
// builds. It is deliberately a data-only plane: an observation can inform a
// later trial, but it cannot authorize a decision or alter Gradle execution.
package stickyobservation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/filelock"
)

const (
	SchemaVersion  = "buildopt.sticky/ordinary-observation/v1"
	RecordType     = "STICKY_ORDINARY_BUILD_OBSERVATION"
	maxRecordBytes = 1 << 20
)

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

// Phase is one mutually exclusive contribution to an observation. An
// unavailable phase has no duration by design; its time remains in the
// explicitly measured Unattributed phase instead of being reported as zero.
type Phase struct {
	DurationNs int64  `json:"durationNs,omitempty"`
	Evidence   string `json:"evidence"`
}

// Timing reconciles the measured process wall time. Wrapper and bootstrap are
// often outside the Go process and can therefore be UNAVAILABLE. All phases
// with evidence other than UNAVAILABLE plus UnattributedNs sum to TotalNs.
type Timing struct {
	TotalNs        int64 `json:"totalNs"`
	Decision       Phase `json:"decision"`
	Network        Phase `json:"network"`
	Cache          Phase `json:"cache"`
	Gradle         Phase `json:"gradle"`
	Observation    Phase `json:"observation"`
	Wrapper        Phase `json:"wrapper"`
	Bootstrap      Phase `json:"bootstrap"`
	UnattributedNs int64 `json:"unattributedNs"`
}

// ConfigurationCache records only what the observer can establish without
// parsing Gradle internals. PRESENT means the standard state directory exists
// after this build; reuse is proven by comparing consecutive records/builds.
type ConfigurationCache struct {
	Requested bool   `json:"requested"`
	State     string `json:"state"`
}

// Provenance binds a record to the repository and executable inputs without
// storing a checkout path or credentials.
type Provenance struct {
	RepositoryScopeSHA256  string `json:"repositoryScopeSha256"`
	SourceRevision         string `json:"sourceRevision,omitempty"`
	SourceRevisionEvidence string `json:"sourceRevisionEvidence"`
	GradleVersion          string `json:"gradleVersion"`
	WrapperSHA256          string `json:"wrapperSha256"`
	BuildOptSHA256         string `json:"buildoptSha256"`
	ArgumentsSHA256        string `json:"argumentsSha256"`
}

// Record is one ordinary requested build. It is never interpreted as a
// qualification, trial or activation record.
type Record struct {
	SchemaVersion      string             `json:"schemaVersion"`
	RecordType         string             `json:"recordType"`
	ObservationID      string             `json:"observationId"`
	IdempotencyKey     string             `json:"idempotencyKey"`
	Provenance         Provenance         `json:"provenance"`
	Outcome            string             `json:"outcome"`
	ExitCode           int                `json:"exitCode"`
	StartedAt          string             `json:"startedAt"`
	CompletedAt        string             `json:"completedAt"`
	Timing             Timing             `json:"timing"`
	ConfigurationCache ConfigurationCache `json:"configurationCache"`
}

// Recorder appends canonical records to one private JSONL file. A busy
// recorder is a soft failure for the build; callers can retain native Gradle.
type Recorder struct {
	path string
}

// NewRecorder creates the parent cache directories with private permissions.
func NewRecorder(path string) (*Recorder, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("ordinary observation path must be one clean absolute path")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return nil, fmt.Errorf("create ordinary observation directory: %w", err)
	}
	info, err := os.Stat(parent)
	if err != nil || !privateObservationDirectoryInfo(info) {
		return nil, errors.New("ordinary observation directory is not private")
	}
	return &Recorder{path: path}, nil
}

// Path returns the append-only record path.
func (recorder *Recorder) Path() string {
	if recorder == nil {
		return ""
	}
	return recorder.path
}

// Append validates and appends one canonical JSON line. It uses a sibling
// lock file so concurrent Gradle invocations never interleave records.
func (recorder *Recorder) Append(record Record) error {
	if recorder == nil || recorder.path == "" {
		return errors.New("ordinary observation recorder is unavailable")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode ordinary observation: %w", err)
	}
	raw, err = contractcrypto.CanonicalizeJCS(raw)
	if err != nil || len(raw) > maxRecordBytes {
		return errors.New("ordinary observation encoding is invalid")
	}
	raw = append(raw, '\n')
	lock, err := os.OpenFile(recorder.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open ordinary observation lock: %w", err)
	}
	defer lock.Close()
	if err := filelock.Try(lock, filelock.Exclusive); err != nil {
		return err
	}
	defer filelock.Unlock(lock)
	file, err := os.OpenFile(recorder.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open ordinary observation log: %w", err)
	}
	defer file.Close()
	if info, statErr := file.Stat(); statErr != nil || !privateObservationFileInfo(info) {
		return errors.New("ordinary observation log is not private")
	}
	if _, err := file.Write(raw); err != nil {
		return fmt.Errorf("append ordinary observation: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync ordinary observation: %w", err)
	}
	return nil
}

// Load reads all records from a private JSONL log for checkers and status
// commands. It rejects malformed, non-canonical or arithmetic-inconsistent
// lines rather than silently skipping evidence.
func Load(path string) ([]Record, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("ordinary observation path must be one clean absolute path")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !privateObservationFileInfo(info) || info.Size() > 64<<20 {
		return nil, errors.New("ordinary observation log is unsafe or too large")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 64<<20+1))
	var records []Record
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode ordinary observation log: %w", err)
		}
		if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, errors.New("ordinary observation log contains an empty record")
		}
		canonical, err := contractcrypto.CanonicalizeJCS(raw)
		if err != nil || !bytes.Equal(canonical, raw) {
			return nil, errors.New("ordinary observation record is not canonical")
		}
		var record Record
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("decode ordinary observation record: %w", err)
		}
		if err := record.Validate(); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// Validate enforces the ordinary-observation data contract.
func (record Record) Validate() error {
	if record.SchemaVersion != SchemaVersion || record.RecordType != RecordType ||
		record.ObservationID == "" || record.IdempotencyKey == "" ||
		!digestPattern.MatchString(record.IdempotencyKey) ||
		!digestPattern.MatchString(record.Provenance.RepositoryScopeSHA256) ||
		!digestPattern.MatchString(record.Provenance.WrapperSHA256) ||
		!digestPattern.MatchString(record.Provenance.BuildOptSHA256) ||
		!digestPattern.MatchString(record.Provenance.ArgumentsSHA256) ||
		record.Provenance.GradleVersion == "" ||
		(record.Provenance.SourceRevisionEvidence != "EXACT" && record.Provenance.SourceRevisionEvidence != "UNAVAILABLE") {
		return errors.New("ordinary observation provenance is invalid")
	}
	if record.Provenance.SourceRevisionEvidence == "EXACT" && !revisionPattern.MatchString(record.Provenance.SourceRevision) {
		return errors.New("ordinary observation source revision is invalid")
	}
	if record.Provenance.SourceRevisionEvidence == "UNAVAILABLE" && record.Provenance.SourceRevision != "" {
		return errors.New("unavailable source revision cannot carry a value")
	}
	if record.Outcome != "SUCCESS" && record.Outcome != "BUILD_FAILURE" && record.Outcome != "INFRA_FAILURE" && record.Outcome != "CANCELLED" {
		return errors.New("ordinary observation outcome is invalid")
	}
	if record.ExitCode < 0 || record.ExitCode > 255 {
		return errors.New("ordinary observation exit code is invalid")
	}
	started, err := parseTimestamp(record.StartedAt)
	if err != nil {
		return err
	}
	completed, err := parseTimestamp(record.CompletedAt)
	if err != nil || completed.Before(started) {
		return errors.New("ordinary observation timestamps are invalid")
	}
	if record.Timing.TotalNs <= 0 || record.Timing.UnattributedNs < 0 {
		return errors.New("ordinary observation timing is invalid")
	}
	phases := []Phase{record.Timing.Decision, record.Timing.Network, record.Timing.Cache, record.Timing.Gradle, record.Timing.Observation, record.Timing.Wrapper, record.Timing.Bootstrap}
	accounted := record.Timing.UnattributedNs
	for _, phase := range phases {
		if phase.Evidence != "EXACT" && phase.Evidence != "APPROXIMATED" && phase.Evidence != "UNAVAILABLE" {
			return errors.New("ordinary observation phase evidence is invalid")
		}
		if phase.DurationNs < 0 || (phase.Evidence == "UNAVAILABLE" && phase.DurationNs != 0) {
			return errors.New("ordinary observation phase duration is invalid")
		}
		if phase.Evidence != "UNAVAILABLE" {
			accounted += phase.DurationNs
		}
	}
	if accounted != record.Timing.TotalNs {
		return fmt.Errorf("ordinary observation timing does not reconcile: accounted %d total %d", accounted, record.Timing.TotalNs)
	}
	if record.ConfigurationCache.State != "NOT_REQUESTED" && record.ConfigurationCache.State != "PRESENT" && record.ConfigurationCache.State != "ABSENT" && record.ConfigurationCache.State != "UNAVAILABLE" {
		return errors.New("ordinary observation Configuration Cache state is invalid")
	}
	if !record.ConfigurationCache.Requested && record.ConfigurationCache.State != "NOT_REQUESTED" {
		return errors.New("ordinary observation reports Configuration Cache without requesting it")
	}
	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New("ordinary observation timestamp must be canonical UTC")
	}
	return parsed, nil
}

// Digest returns the lowercase SHA-256 digest used for record identities.
func Digest(value ...string) string {
	hash := sha256.New()
	for _, item := range value {
		_, _ = io.WriteString(hash, item)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// ScopeForRoot derives a path-independent-enough local scope for repositories
// without a configured central project identity. Callers may provide a stable
// project identifier as the first argument instead of a checkout path.
func ScopeForRoot(identity string) string {
	return Digest("buildopt-sticky-observation-scope-v1", strings.TrimSpace(identity))
}
