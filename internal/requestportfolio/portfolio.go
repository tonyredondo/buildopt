// Package requestportfolio persists the exact Gradle request identities seen
// by the committed wrapper. It is an observation-only POC boundary: entries
// may inform later experiments, but they cannot authorize an action.
package requestportfolio

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
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/filelock"
)

const (
	SchemaVersion         = "buildopt.poc/observed-request-portfolio/v1"
	EvidenceSchemaVersion = "buildopt.poc/observed-request-evidence/v1"
	MaximumEntries        = 128
	maximumRecentIDs      = 256
	maximumFileBytes      = 4 << 20
	maximumLockWait       = 5 * time.Second
)

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var taskPattern = regexp.MustCompile(`^(:[A-Za-z0-9_.-]+)+$`)

// Evidence is the finalized request-graph evidence optionally emitted by the
// same Gradle invocation. The arguments digest prevents evidence from one
// request being attached to another.
type Evidence struct {
	SchemaVersion               string   `json:"schemaVersion"`
	ObservationID               string   `json:"observationId"`
	ArgumentsSHA256             string   `json:"argumentsSha256"`
	CompatibilityIdentitySHA256 string   `json:"compatibilityIdentitySha256"`
	RequestedTasks              []string `json:"requestedTasks"`
	RequestGraphIdentitySHA256  string   `json:"requestGraphIdentitySha256"`
}

// Observation contains only bounded portable facts derived from one wrapper
// invocation. Exact argv bytes remain private to the process; their canonical
// length-prefixed SHA-256 is the durable identity.
type Observation struct {
	ObservationID               string   `json:"observationId"`
	ObservedAt                  string   `json:"observedAt"`
	RepositoryScopeSHA256       string   `json:"repositoryScopeSha256"`
	ArgumentsSHA256             string   `json:"argumentsSha256"`
	WorkingDirectorySHA256      string   `json:"workingDirectorySha256"`
	WorkingDirectoryEvidence    string   `json:"workingDirectoryEvidence"`
	CompatibilityIdentitySHA256 string   `json:"compatibilityIdentitySha256"`
	CompatibilityEvidence       string   `json:"compatibilityEvidence"`
	RequestedTasks              []string `json:"requestedTasks"`
	RequestGraphIdentitySHA256  string   `json:"requestGraphIdentitySha256"`
	RequestGraphEvidence        string   `json:"requestGraphEvidence"`
	Outcome                     string   `json:"outcome"`
	ExitCode                    int      `json:"exitCode"`
	Bypassed                    bool     `json:"bypassed"`
}

// OutcomeCounts preserves negative observations rather than dropping them
// from a seemingly favorable request frequency.
type OutcomeCounts struct {
	Success      int `json:"success"`
	BuildFailure int `json:"buildFailure"`
	InfraFailure int `json:"infraFailure"`
	Cancelled    int `json:"cancelled"`
	Bypassed     int `json:"bypassed"`
}

// Entry is one exact argv/compatibility/request-graph identity over time.
type Entry struct {
	IdentitySHA256              string        `json:"identitySha256"`
	RepositoryScopeSHA256       string        `json:"repositoryScopeSha256"`
	ArgumentsSHA256             string        `json:"argumentsSha256"`
	WorkingDirectorySHA256      string        `json:"workingDirectorySha256"`
	WorkingDirectoryEvidence    string        `json:"workingDirectoryEvidence"`
	CompatibilityIdentitySHA256 string        `json:"compatibilityIdentitySha256"`
	CompatibilityEvidence       string        `json:"compatibilityEvidence"`
	RequestedTasks              []string      `json:"requestedTasks"`
	RequestGraphIdentitySHA256  string        `json:"requestGraphIdentitySha256"`
	RequestGraphEvidence        string        `json:"requestGraphEvidence"`
	FirstObservedAt             string        `json:"firstObservedAt"`
	LastObservedAt              string        `json:"lastObservedAt"`
	ObservationCount            int           `json:"observationCount"`
	EligibleObservationCount    int           `json:"eligibleObservationCount"`
	Outcomes                    OutcomeCounts `json:"outcomes"`
	Lifecycle                   string        `json:"lifecycle"`
	CandidateEligible           bool          `json:"candidateEligible"`
}

// Portfolio is one bounded repository-scoped snapshot. It deliberately
// carries explicit non-authority flags so no consumer can confuse observation
// frequency with a qualified optimization decision.
type Portfolio struct {
	SchemaVersion         string   `json:"schemaVersion"`
	RepositoryScopeSHA256 string   `json:"repositoryScopeSha256"`
	Generation            int64    `json:"generation"`
	Entries               []Entry  `json:"entries"`
	RecentObservationIDs  []string `json:"recentObservationIds"`
	SelectionAuthorized   bool     `json:"selectionAuthorized"`
	ActivationAuthorized  bool     `json:"activationAuthorized"`
	PerformanceMeasured   bool     `json:"performanceMeasured"`
}

// Store atomically updates one private portfolio snapshot under a sibling
// advisory lock. Busy or unavailable storage is a soft launcher failure.
type Store struct{ path string }

func NewStore(path string) (*Store, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("request portfolio path must be one clean absolute path")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return nil, fmt.Errorf("create request portfolio directory: %w", err)
	}
	info, err := os.Lstat(parent)
	if err != nil || !privateDirectoryInfo(info) {
		return nil, errors.New("request portfolio directory is not private")
	}
	return &Store{path: path}, nil
}

func (store *Store) Path() string {
	if store == nil {
		return ""
	}
	return store.path
}

// Observe merges one idempotent observation and publishes the next snapshot.
func (store *Store) Observe(observation Observation) (Portfolio, error) {
	if store == nil || store.path == "" {
		return Portfolio{}, errors.New("request portfolio store is unavailable")
	}
	if err := observation.Validate(); err != nil {
		return Portfolio{}, err
	}
	lock, err := os.OpenFile(store.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return Portfolio{}, fmt.Errorf("open request portfolio lock: %w", err)
	}
	defer lock.Close()
	lockDeadline := time.Now().Add(maximumLockWait)
	for {
		err = filelock.Try(lock, filelock.Exclusive)
		if err == nil {
			break
		}
		if !errors.Is(err, filelock.ErrBusy) || time.Now().After(lockDeadline) {
			return Portfolio{}, err
		}
		time.Sleep(2 * time.Millisecond)
	}
	defer filelock.Unlock(lock)

	portfolio := Portfolio{SchemaVersion: SchemaVersion, RepositoryScopeSHA256: observation.RepositoryScopeSHA256}
	if _, err := os.Lstat(store.path); err == nil {
		portfolio, err = Load(store.path)
		if err != nil {
			return Portfolio{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Portfolio{}, fmt.Errorf("inspect request portfolio: %w", err)
	}
	if portfolio.RepositoryScopeSHA256 != observation.RepositoryScopeSHA256 {
		return Portfolio{}, errors.New("request portfolio repository scope differs")
	}
	for _, id := range portfolio.RecentObservationIDs {
		if id == observation.ObservationID {
			return portfolio, nil
		}
	}
	portfolio.Generation++
	portfolio.RecentObservationIDs = append(portfolio.RecentObservationIDs, observation.ObservationID)
	if len(portfolio.RecentObservationIDs) > maximumRecentIDs {
		portfolio.RecentObservationIDs = append([]string(nil), portfolio.RecentObservationIDs[len(portfolio.RecentObservationIDs)-maximumRecentIDs:]...)
	}
	identity := observationIdentity(observation)
	index := -1
	for current := range portfolio.Entries {
		if portfolio.Entries[current].IdentitySHA256 == identity {
			index = current
			break
		}
	}
	if index < 0 {
		portfolio.Entries = append(portfolio.Entries, Entry{
			IdentitySHA256: identity, RepositoryScopeSHA256: observation.RepositoryScopeSHA256,
			ArgumentsSHA256:             observation.ArgumentsSHA256,
			WorkingDirectorySHA256:      observation.WorkingDirectorySHA256,
			WorkingDirectoryEvidence:    observation.WorkingDirectoryEvidence,
			CompatibilityIdentitySHA256: observation.CompatibilityIdentitySHA256,
			CompatibilityEvidence:       observation.CompatibilityEvidence,
			RequestedTasks:              append([]string(nil), observation.RequestedTasks...),
			RequestGraphIdentitySHA256:  observation.RequestGraphIdentitySHA256,
			RequestGraphEvidence:        observation.RequestGraphEvidence,
			FirstObservedAt:             observation.ObservedAt,
		})
		index = len(portfolio.Entries) - 1
	}
	entry := &portfolio.Entries[index]
	observedAt, _ := canonicalTimestamp(observation.ObservedAt)
	firstObservedAt, _ := canonicalTimestamp(entry.FirstObservedAt)
	if observedAt.Before(firstObservedAt) {
		entry.FirstObservedAt = observation.ObservedAt
	}
	if entry.LastObservedAt == "" {
		entry.LastObservedAt = observation.ObservedAt
	} else {
		lastObservedAt, _ := canonicalTimestamp(entry.LastObservedAt)
		if observedAt.After(lastObservedAt) {
			entry.LastObservedAt = observation.ObservedAt
		}
	}
	entry.ObservationCount++
	switch {
	case observation.Bypassed:
		entry.Outcomes.Bypassed++
	case observation.Outcome == "SUCCESS":
		entry.Outcomes.Success++
	case observation.Outcome == "BUILD_FAILURE":
		entry.Outcomes.BuildFailure++
	case observation.Outcome == "INFRA_FAILURE":
		entry.Outcomes.InfraFailure++
	case observation.Outcome == "CANCELLED":
		entry.Outcomes.Cancelled++
	}
	complete := observation.WorkingDirectoryEvidence == "EXACT" && observation.CompatibilityEvidence == "EXACT" && observation.RequestGraphEvidence == "EXACT"
	if complete && observation.Outcome == "SUCCESS" && !observation.Bypassed {
		entry.EligibleObservationCount++
	}
	entry.CandidateEligible = entry.EligibleObservationCount > 0
	switch {
	case entry.CandidateEligible:
		entry.Lifecycle = "EVIDENCE_COMPLETE"
	case complete:
		entry.Lifecycle = "INELIGIBLE_OUTCOME"
	default:
		entry.Lifecycle = "OBSERVED_INCOMPLETE"
	}
	trimEntries(&portfolio)
	if err := portfolio.Validate(); err != nil {
		return Portfolio{}, err
	}
	if err := writeCanonicalAtomic(store.path, portfolio); err != nil {
		return Portfolio{}, err
	}
	return portfolio, nil
}

func Load(path string) (Portfolio, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return Portfolio{}, errors.New("request portfolio path must be one clean absolute path")
	}
	pathInfo, err := os.Lstat(path)
	if err != nil || !privateFileInfo(pathInfo) || pathInfo.Size() < 1 || pathInfo.Size() > maximumFileBytes {
		return Portfolio{}, errors.New("request portfolio file is unsafe or too large")
	}
	file, err := os.Open(path)
	if err != nil {
		return Portfolio{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !os.SameFile(pathInfo, info) || !privateFileInfo(info) || info.Size() < 1 || info.Size() > maximumFileBytes {
		return Portfolio{}, errors.New("request portfolio file is unsafe or too large")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maximumFileBytes+1))
	if err != nil || len(raw) > maximumFileBytes {
		return Portfolio{}, errors.New("read request portfolio")
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Portfolio{}, errors.New("request portfolio is not canonical")
	}
	var portfolio Portfolio
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&portfolio); err != nil {
		return Portfolio{}, fmt.Errorf("decode request portfolio: %w", err)
	}
	if err := portfolio.Validate(); err != nil {
		return Portfolio{}, err
	}
	return portfolio, nil
}

func (evidence Evidence) Validate(observationID, argumentsSHA256 string) error {
	if evidence.SchemaVersion != EvidenceSchemaVersion || evidence.ObservationID != observationID ||
		!digestPattern.MatchString(evidence.ObservationID) || evidence.ArgumentsSHA256 != argumentsSHA256 ||
		!digestPattern.MatchString(evidence.ArgumentsSHA256) || !digestPattern.MatchString(evidence.CompatibilityIdentitySHA256) ||
		!digestPattern.MatchString(evidence.RequestGraphIdentitySHA256) ||
		len(evidence.RequestedTasks) == 0 || !canonicalTasks(evidence.RequestedTasks) {
		return errors.New("observed request evidence is invalid")
	}
	return nil
}

func LoadEvidence(path, observationID, argumentsSHA256 string) (Evidence, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return Evidence{}, errors.New("observed request evidence path must be one clean absolute path")
	}
	pathInfo, err := os.Lstat(path)
	if err != nil || !privateFileInfo(pathInfo) || pathInfo.Size() < 1 || pathInfo.Size() > 1<<20 {
		return Evidence{}, errors.New("observed request evidence file is unsafe")
	}
	file, err := os.Open(path)
	if err != nil {
		return Evidence{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !os.SameFile(pathInfo, info) || !privateFileInfo(info) {
		return Evidence{}, errors.New("observed request evidence file changed")
	}
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return Evidence{}, errors.New("read observed request evidence")
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Evidence{}, errors.New("observed request evidence is not canonical")
	}
	var evidence Evidence
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err != nil {
		return Evidence{}, err
	}
	return evidence, evidence.Validate(observationID, argumentsSHA256)
}

// WriteEvidence publishes finalized evidence from the same Gradle invocation
// as one canonical private file. Callers must bind the evidence to the exact
// observation and argument digest before publishing it.
func WriteEvidence(path string, evidence Evidence) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("observed request evidence path must be one clean absolute path")
	}
	parentInfo, err := os.Lstat(filepath.Dir(path))
	if err != nil || !privateDirectoryInfo(parentInfo) {
		return errors.New("observed request evidence directory is not private")
	}
	if err := evidence.Validate(evidence.ObservationID, evidence.ArgumentsSHA256); err != nil {
		return err
	}
	return writeCanonicalAtomic(path, evidence)
}

func (observation Observation) Validate() error {
	if !digestPattern.MatchString(observation.ObservationID) || !digestPattern.MatchString(observation.RepositoryScopeSHA256) ||
		!digestPattern.MatchString(observation.ArgumentsSHA256) || !digestPattern.MatchString(observation.WorkingDirectorySHA256) ||
		(observation.WorkingDirectoryEvidence != "EXACT" && observation.WorkingDirectoryEvidence != "UNAVAILABLE") ||
		!digestPattern.MatchString(observation.CompatibilityIdentitySHA256) ||
		(observation.CompatibilityEvidence != "EXACT" && observation.CompatibilityEvidence != "UNAVAILABLE") ||
		(observation.RequestGraphEvidence != "EXACT" && observation.RequestGraphEvidence != "UNAVAILABLE") ||
		(observation.Outcome != "SUCCESS" && observation.Outcome != "BUILD_FAILURE" && observation.Outcome != "INFRA_FAILURE" && observation.Outcome != "CANCELLED") ||
		observation.ExitCode < 0 || observation.ExitCode > 255 {
		return errors.New("request portfolio observation is invalid")
	}
	if _, err := canonicalTimestamp(observation.ObservedAt); err != nil {
		return err
	}
	if observation.RequestGraphEvidence == "EXACT" {
		if !digestPattern.MatchString(observation.RequestGraphIdentitySHA256) || !canonicalTasks(observation.RequestedTasks) || len(observation.RequestedTasks) == 0 {
			return errors.New("exact request graph evidence is invalid")
		}
	} else if observation.RequestGraphIdentitySHA256 != "" || len(observation.RequestedTasks) != 0 {
		return errors.New("unavailable request graph carries values")
	}
	return nil
}

func (portfolio Portfolio) Validate() error {
	if portfolio.SchemaVersion != SchemaVersion || !digestPattern.MatchString(portfolio.RepositoryScopeSHA256) ||
		portfolio.Generation < 1 || len(portfolio.Entries) == 0 || len(portfolio.Entries) > MaximumEntries ||
		len(portfolio.RecentObservationIDs) == 0 || len(portfolio.RecentObservationIDs) > maximumRecentIDs ||
		portfolio.SelectionAuthorized || portfolio.ActivationAuthorized || portfolio.PerformanceMeasured {
		return errors.New("request portfolio contract is invalid")
	}
	previous := ""
	for _, entry := range portfolio.Entries {
		if !digestPattern.MatchString(entry.IdentitySHA256) || entry.IdentitySHA256 <= previous ||
			entry.RepositoryScopeSHA256 != portfolio.RepositoryScopeSHA256 || !digestPattern.MatchString(entry.ArgumentsSHA256) ||
			!digestPattern.MatchString(entry.WorkingDirectorySHA256) ||
			(entry.WorkingDirectoryEvidence != "EXACT" && entry.WorkingDirectoryEvidence != "UNAVAILABLE") ||
			!digestPattern.MatchString(entry.CompatibilityIdentitySHA256) || entry.ObservationCount < 1 ||
			entry.EligibleObservationCount < 0 || entry.EligibleObservationCount > entry.ObservationCount ||
			entry.ObservationCount != entry.Outcomes.Success+entry.Outcomes.BuildFailure+entry.Outcomes.InfraFailure+entry.Outcomes.Cancelled+entry.Outcomes.Bypassed ||
			entry.CandidateEligible != (entry.EligibleObservationCount > 0) {
			return errors.New("request portfolio entry is invalid")
		}
		first, firstErr := canonicalTimestamp(entry.FirstObservedAt)
		last, lastErr := canonicalTimestamp(entry.LastObservedAt)
		if firstErr != nil || lastErr != nil || last.Before(first) {
			return errors.New("request portfolio entry timestamps are invalid")
		}
		if entry.RequestGraphEvidence == "EXACT" {
			if !digestPattern.MatchString(entry.RequestGraphIdentitySHA256) || len(entry.RequestedTasks) == 0 || !canonicalTasks(entry.RequestedTasks) {
				return errors.New("request portfolio graph is invalid")
			}
		} else if entry.RequestGraphEvidence != "UNAVAILABLE" || entry.RequestGraphIdentitySHA256 != "" || len(entry.RequestedTasks) != 0 {
			return errors.New("request portfolio unavailable graph is invalid")
		}
		if entry.CompatibilityEvidence != "EXACT" && entry.CompatibilityEvidence != "UNAVAILABLE" {
			return errors.New("request portfolio compatibility is invalid")
		}
		expected := "OBSERVED_INCOMPLETE"
		if entry.WorkingDirectoryEvidence == "EXACT" && entry.CompatibilityEvidence == "EXACT" && entry.RequestGraphEvidence == "EXACT" {
			expected = "INELIGIBLE_OUTCOME"
		}
		if entry.CandidateEligible {
			expected = "EVIDENCE_COMPLETE"
		}
		if entry.Lifecycle != expected {
			return errors.New("request portfolio lifecycle is invalid")
		}
		previous = entry.IdentitySHA256
	}
	ids := map[string]bool{}
	for _, id := range portfolio.RecentObservationIDs {
		if id == "" || ids[id] {
			return errors.New("request portfolio recent ids are invalid")
		}
		ids[id] = true
	}
	return nil
}

func ArgumentsSHA256(arguments []string) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, "buildopt-observed-request-argv-v1")
	for _, argument := range arguments {
		var length [8]byte
		for index := 0; index < len(length); index++ {
			length[7-index] = byte(uint64(len(argument)) >> (index * 8))
		}
		_, _ = hash.Write(length[:])
		_, _ = io.WriteString(hash, argument)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func CompatibilitySHA256(parts ...string) string {
	return digest(append([]string{"buildopt-observed-request-compatibility-v1"}, parts...)...)
}

func observationIdentity(observation Observation) string {
	graph := observation.RequestGraphIdentitySHA256
	if graph == "" {
		graph = "UNAVAILABLE"
	}
	return digest("buildopt-observed-request-identity-v1", observation.RepositoryScopeSHA256, observation.ArgumentsSHA256,
		observation.WorkingDirectoryEvidence, observation.WorkingDirectorySHA256,
		observation.CompatibilityEvidence, observation.CompatibilityIdentitySHA256, observation.RequestGraphEvidence, graph)
}

func trimEntries(portfolio *Portfolio) {
	sort.Slice(portfolio.Entries, func(left, right int) bool {
		return portfolio.Entries[left].IdentitySHA256 < portfolio.Entries[right].IdentitySHA256
	})
	if len(portfolio.Entries) <= MaximumEntries {
		return
	}
	sort.Slice(portfolio.Entries, func(left, right int) bool {
		if portfolio.Entries[left].LastObservedAt != portfolio.Entries[right].LastObservedAt {
			return portfolio.Entries[left].LastObservedAt > portfolio.Entries[right].LastObservedAt
		}
		if portfolio.Entries[left].ObservationCount != portfolio.Entries[right].ObservationCount {
			return portfolio.Entries[left].ObservationCount > portfolio.Entries[right].ObservationCount
		}
		return portfolio.Entries[left].IdentitySHA256 < portfolio.Entries[right].IdentitySHA256
	})
	portfolio.Entries = append([]Entry(nil), portfolio.Entries[:MaximumEntries]...)
	sort.Slice(portfolio.Entries, func(left, right int) bool {
		return portfolio.Entries[left].IdentitySHA256 < portfolio.Entries[right].IdentitySHA256
	})
}

func canonicalTasks(tasks []string) bool {
	previous := ""
	for _, task := range tasks {
		if !taskPattern.MatchString(task) || task <= previous {
			return false
		}
		previous = task
	}
	return true
}

func canonicalTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC || parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New("request portfolio timestamp is not canonical UTC")
	}
	return parsed, nil
}

func digest(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(hash, value)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeCanonicalAtomic(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	raw, err = contractcrypto.CanonicalizeJCS(raw)
	if err != nil || len(raw) > maximumFileBytes {
		return errors.New("request portfolio encoding is invalid")
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".request-portfolio-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil && !strings.Contains(strings.ToLower(err.Error()), "invalid") {
		return err
	}
	return nil
}
