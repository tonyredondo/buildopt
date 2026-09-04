package configurationinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const reportMarker = "function configurationCacheProblems() { return ("

const (
	ProblemExternalProcess = "EXTERNAL_PROCESS"
	ProblemFileRead        = "FILE_READ"
	ProblemPropertyRead    = "PROPERTY_READ"
	ProblemOther           = "OTHER"
)

// ErrInvalidReport means the Gradle report has no unique, complete embedded
// diagnostics payload. It is never converted into a no-action row.
var ErrInvalidReport = errors.New("Gradle Configuration Cache report is invalid")

// Problem is one normalized strict-report diagnostic.
type Problem struct {
	Index      int    `json:"index"`
	Kind       string `json:"kind"`
	Message    string `json:"message"`
	TraceOwner string `json:"traceOwner"`
}

type fragment struct {
	Text string `json:"text"`
	Name string `json:"name"`
}

type traceEntry struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

type diagnostic struct {
	Trace   []traceEntry `json:"trace"`
	Problem []fragment   `json:"problem"`
}

type envelope struct {
	Diagnostics       []diagnostic `json:"diagnostics"`
	TotalProblemCount int          `json:"totalProblemCount"`
}

// ParseReport extracts the single JSON value embedded by Gradle's report. The
// surrounding JavaScript is not executed.
func ParseReport(content []byte) ([]Problem, error) {
	markerIndex := bytes.Index(content, []byte(reportMarker))
	if markerIndex < 0 || bytes.Index(content[markerIndex+len(reportMarker):], []byte(reportMarker)) >= 0 {
		return nil, ErrInvalidReport
	}
	decoder := json.NewDecoder(bytes.NewReader(content[markerIndex+len(reportMarker):]))
	var payload envelope
	if err := decoder.Decode(&payload); err != nil {
		return nil, ErrInvalidReport
	}
	if payload.TotalProblemCount != len(payload.Diagnostics) || payload.TotalProblemCount == 0 {
		return nil, ErrInvalidReport
	}
	problems := make([]Problem, 0, len(payload.Diagnostics))
	for index, raw := range payload.Diagnostics {
		message := fragments(raw.Problem)
		if message == "" || len(raw.Trace) == 0 {
			return nil, ErrInvalidReport
		}
		problems = append(problems, Problem{Index: index, Kind: problemKind(message), Message: message, TraceOwner: owner(raw.Trace)})
	}
	return problems, nil
}

// DecodeFacts rejects unknown fields and trailing JSON.
func DecodeFacts(content []byte) ([]SourceFacts, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var facts []SourceFacts
	if err := decoder.Decode(&facts); err != nil || len(facts) == 0 {
		return nil, ErrInvalidFacts
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidFacts
	}
	return facts, nil
}

func fragments(parts []fragment) string {
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(part.Text)
		builder.WriteString(part.Name)
	}
	return strings.TrimSpace(builder.String())
}

func owner(trace []traceEntry) string {
	entry := trace[len(trace)-1]
	return strings.Join([]string{entry.Kind, entry.Path, entry.Location, entry.Type}, "|")
}

func problemKind(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "external process started"):
		return ProblemExternalProcess
	case strings.Contains(lower, "file") && (strings.Contains(lower, "unsupported") || strings.Contains(lower, "untracked")):
		return ProblemFileRead
	case (strings.Contains(lower, "environment variable") || strings.Contains(lower, "system property") || strings.Contains(lower, "gradle property")) &&
		(strings.Contains(lower, "unsupported") || strings.Contains(lower, "untracked")):
		return ProblemPropertyRead
	default:
		return ProblemOther
	}
}
