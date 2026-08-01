// Package buildhistory exposes a bounded read model over immutable,
// already-redacted BUILD_SESSION exports.
package buildhistory

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildsession"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	// ListPath is the authenticated build-history collection endpoint.
	ListPath = "/api/v1/build-sessions"
	// DetailPath is the authenticated exact-session lookup endpoint.
	DetailPath = "/api/v1/build-session"
	// TokenEnvironment configures the independent history read credential.
	TokenEnvironment = "BUILDOPT_HISTORY_API_TOKEN"

	defaultLimit     = 25
	maximumLimit     = 100
	maximumDocuments = 10000
	maximumFileBytes = 1 << 20
	filePrefix       = "build-session-"
	fileSuffix       = ".json"
)

var (
	// ErrInvalidQuery reports a malformed filter, limit, cursor, or identity.
	ErrInvalidQuery = errors.New("invalid build history query")
	// ErrNotFound reports an absent immutable BUILD_SESSION identity.
	ErrNotFound = errors.New("build session not found")
)

// Store reads immutable BUILD_SESSION exports from one exporter-owned root.
type Store struct {
	directory string
}

// Filter is the bounded collection query accepted by List.
type Filter struct {
	RepositoryID string
	Outcome      string
	Limit        int
	Cursor       string
}

// Summary is the stable UI-facing subset of one BUILD_SESSION.
type Summary struct {
	ID                string `json:"id"`
	RepositoryID      string `json:"repositoryId"`
	Revision          string `json:"revision"`
	StartedAt         string `json:"startedAt"`
	CompletedAt       string `json:"completedAt"`
	Outcome           string `json:"outcome"`
	ExitCode          int    `json:"exitCode"`
	DurationMs        int64  `json:"durationMs"`
	Complete          bool   `json:"complete"`
	PluginVersion     string `json:"pluginVersion"`
	Environment       string `json:"environment"`
	PipelineClass     string `json:"pipelineClass"`
	CacheState        string `json:"cacheState"`
	ExperimentArm     string `json:"experimentArm"`
	MeasurementStatus string `json:"measurementStatus"`
}

// ListResponse is one stable cursor page of build summaries.
type ListResponse struct {
	SchemaVersion string    `json:"schemaVersion"`
	Items         []Summary `json:"items"`
	NextCursor    string    `json:"nextCursor,omitempty"`
	MatchedCount  int       `json:"matchedCount"`
}

// DetailResponse returns the exact already-redacted normative document.
type DetailResponse struct {
	SchemaVersion string                `json:"schemaVersion"`
	Session       buildsession.Document `json:"session"`
}

type loadedDocument struct {
	document    buildsession.Document
	completedAt time.Time
}

type cursorValue struct {
	CompletedAt string `json:"completedAt"`
	ID          string `json:"id"`
}

// NewStore binds a read model to the canonical export directory.
func NewStore(directory string) (*Store, error) {
	if directory == "" {
		return nil, errors.New("build history export directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, errors.New("resolve build history export directory")
	}
	info, err := os.Lstat(absolute)
	if err != nil || !privateHistoryDirectoryInfo(info) {
		return nil, errors.New("build history export directory is not private")
	}
	return &Store{directory: absolute}, nil
}

// List returns newest-first summaries with exact filters and a stable cursor.
func (store *Store) List(filter Filter) (ListResponse, error) {
	if filter.Limit == 0 {
		filter.Limit = defaultLimit
	}
	if filter.Limit < 1 || filter.Limit > maximumLimit ||
		!validOutcomeFilter(filter.Outcome) ||
		len(filter.RepositoryID) > 256 {
		return ListResponse{}, ErrInvalidQuery
	}

	documents, err := store.load()
	if err != nil {
		return ListResponse{}, err
	}
	filtered := make([]loadedDocument, 0, len(documents))
	for _, loaded := range documents {
		if filter.RepositoryID != "" &&
			loaded.document.Build.RepositoryID != filter.RepositoryID {
			continue
		}
		if filter.Outcome != "" &&
			loaded.document.Build.Outcome != filter.Outcome {
			continue
		}
		filtered = append(filtered, loaded)
	}

	start := 0
	if filter.Cursor != "" {
		cursor, decodeErr := decodeCursor(filter.Cursor)
		if decodeErr != nil {
			return ListResponse{}, ErrInvalidQuery
		}
		found := false
		for index, loaded := range filtered {
			if loaded.document.Build.ID == cursor.ID &&
				loaded.document.Build.CompletedAt == cursor.CompletedAt {
				start = index + 1
				found = true
				break
			}
		}
		if !found {
			return ListResponse{}, ErrInvalidQuery
		}
	}

	end := min(start+filter.Limit, len(filtered))
	items := make([]Summary, 0, end-start)
	for _, loaded := range filtered[start:end] {
		items = append(items, summarize(loaded.document))
	}
	response := ListResponse{
		SchemaVersion: "buildopt.api/build-session-history/v1",
		Items:         items,
		MatchedCount:  len(filtered),
	}
	if end < len(filtered) {
		last := filtered[end-1].document.Build
		response.NextCursor = encodeCursor(cursorValue{
			CompletedAt: last.CompletedAt,
			ID:          last.ID,
		})
	}
	return response, nil
}

// Get returns one exact already-redacted BUILD_SESSION document.
func (store *Store) Get(sessionID string) (DetailResponse, error) {
	if sessionID == "" || len(sessionID) > 256 {
		return DetailResponse{}, ErrInvalidQuery
	}
	documents, err := store.load()
	if err != nil {
		return DetailResponse{}, err
	}
	for _, loaded := range documents {
		if loaded.document.Build.ID == sessionID {
			return DetailResponse{
				SchemaVersion: "buildopt.api/build-session-detail/v1",
				Session:       loaded.document,
			}, nil
		}
	}
	return DetailResponse{}, ErrNotFound
}

func (store *Store) load() ([]loadedDocument, error) {
	entries, err := os.ReadDir(store.directory)
	if err != nil {
		return nil, errors.New("read build history export directory")
	}
	documents := make([]loadedDocument, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, filePrefix) ||
			!strings.HasSuffix(name, fileSuffix) {
			continue
		}
		if len(documents) == maximumDocuments {
			return nil, errors.New("build history document limit exceeded")
		}
		loaded, loadErr := store.loadFile(name)
		if loadErr != nil {
			return nil, loadErr
		}
		documents = append(documents, loaded)
	}
	sort.Slice(documents, func(left, right int) bool {
		if !documents[left].completedAt.Equal(documents[right].completedAt) {
			return documents[left].completedAt.After(documents[right].completedAt)
		}
		return documents[left].document.Build.ID < documents[right].document.Build.ID
	})
	return documents, nil
}

func (store *Store) loadFile(name string) (loadedDocument, error) {
	path := filepath.Join(store.directory, name)
	info, err := os.Lstat(path)
	if err != nil || !privateHistoryFileInfo(info) ||
		info.Size() < 1 || info.Size() > maximumFileBytes {
		return loadedDocument{}, errors.New("build history contains an unsafe document")
	}
	file, err := os.Open(path)
	if err != nil {
		return loadedDocument{}, errors.New("open build history document")
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maximumFileBytes+1))
	decoder.DisallowUnknownFields()
	var document buildsession.Document
	if err := decoder.Decode(&document); err != nil {
		return loadedDocument{}, errors.New("decode build history document")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return loadedDocument{}, errors.New("build history document contains trailing data")
	}
	completedAt, err := validateDocument(document)
	if err != nil {
		return loadedDocument{}, err
	}
	digest := sha256.Sum256([]byte(document.Build.ID))
	expectedName := filePrefix + hex.EncodeToString(digest[:]) + fileSuffix
	if name != expectedName {
		return loadedDocument{}, errors.New("build history filename does not bind session identity")
	}
	return loadedDocument{document: document, completedAt: completedAt}, nil
}

func validateDocument(document buildsession.Document) (time.Time, error) {
	if document.SchemaVersion != "1.0" || document.RecordType != "BUILD_SESSION" ||
		document.Build.ID == "" || len(document.Build.ID) > 256 ||
		document.Build.RepositoryID == "" || len(document.Build.RepositoryID) > 256 ||
		document.Build.Revision == "" || len(document.Build.Revision) > 256 ||
		len(document.GradleInvocations) != 1 ||
		document.Performance.CustomerVisibleBuildMs.ValueMs == nil ||
		*document.Performance.CustomerVisibleBuildMs.ValueMs < 0 ||
		(document.Complete && document.Recovery != nil) ||
		(!document.Complete && document.Recovery == nil) {
		return time.Time{}, errors.New("build history document violates the BUILD_SESSION summary contract")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, document.Build.StartedAt)
	if err != nil || startedAt.Location() != time.UTC {
		return time.Time{}, errors.New("build history document has invalid startedAt")
	}
	completedAt, err := time.Parse(time.RFC3339Nano, document.Build.CompletedAt)
	if err != nil || completedAt.Location() != time.UTC || completedAt.Before(startedAt) {
		return time.Time{}, errors.New("build history document has invalid completedAt")
	}
	switch document.Build.Outcome {
	case sessioningest.OutcomeSuccess:
		if document.Build.ExitCode != 0 {
			return time.Time{}, errors.New("successful build history document has nonzero exit code")
		}
	case sessioningest.OutcomeBuildFailure, sessioningest.OutcomeCancelled:
		if document.Build.ExitCode < 1 || document.Build.ExitCode > 255 {
			return time.Time{}, errors.New("failed build history document has invalid exit code")
		}
	default:
		return time.Time{}, errors.New("build history document has invalid outcome")
	}
	return completedAt, nil
}

func summarize(document buildsession.Document) Summary {
	return Summary{
		ID:                document.Build.ID,
		RepositoryID:      document.Build.RepositoryID,
		Revision:          document.Build.Revision,
		StartedAt:         document.Build.StartedAt,
		CompletedAt:       document.Build.CompletedAt,
		Outcome:           document.Build.Outcome,
		ExitCode:          document.Build.ExitCode,
		DurationMs:        *document.Performance.CustomerVisibleBuildMs.ValueMs,
		Complete:          document.Complete,
		PluginVersion:     document.Build.PluginVersion,
		Environment:       document.Workload.Environment,
		PipelineClass:     document.Workload.PipelineClass,
		CacheState:        document.Workload.CacheState,
		ExperimentArm:     document.ExperimentAssignment.Arm,
		MeasurementStatus: document.MeasurementMetadata.Status,
	}
}

func validOutcomeFilter(outcome string) bool {
	switch outcome {
	case "", sessioningest.OutcomeSuccess,
		sessioningest.OutcomeBuildFailure, sessioningest.OutcomeCancelled:
		return true
	default:
		return false
	}
}

func encodeCursor(cursor cursorValue) string {
	content, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(content)
}

func decodeCursor(encoded string) (cursorValue, error) {
	if encoded == "" || len(encoded) > 1024 {
		return cursorValue{}, ErrInvalidQuery
	}
	content, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return cursorValue{}, ErrInvalidQuery
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var cursor cursorValue
	if err := decoder.Decode(&cursor); err != nil || cursor.ID == "" ||
		cursor.CompletedAt == "" {
		return cursorValue{}, ErrInvalidQuery
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return cursorValue{}, ErrInvalidQuery
	}
	return cursor, nil
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return defaultLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maximumLimit {
		return 0, ErrInvalidQuery
	}
	return value, nil
}

func invalidQueryParameter(values map[string][]string, allowed ...string) bool {
	accepted := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		accepted[name] = struct{}{}
	}
	for name, entries := range values {
		if _, found := accepted[name]; !found || len(entries) != 1 {
			return true
		}
	}
	return false
}

func queryValue(values map[string][]string, name string) string {
	entries := values[name]
	if len(entries) == 0 {
		return ""
	}
	return entries[0]
}
