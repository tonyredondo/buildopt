package sessioningest

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// ProtocolVersion identifies the provisional WS-005 ingest protocol.
	ProtocolVersion = "1.0"
	// RecordType distinguishes the ingest handoff from the WS-006 export.
	RecordType = "BUILD_SESSION_INGEST"

	// OutcomeSuccess records a child process that exited successfully.
	OutcomeSuccess = "SUCCESS"
	// OutcomeBuildFailure records a child process with a nonzero exit status.
	OutcomeBuildFailure = "BUILD_FAILURE"
	// OutcomeCancelled records a child process stopped after cancellation.
	OutcomeCancelled = "CANCELLED"

	// ExportContextEnvironment carries predeclared non-secret BUILD_SESSION
	// identity and workload context.
	ExportContextEnvironment = "BUILDOPT_BUILD_SESSION_CONTEXT"

	maxExportContextBytes = 32 << 10
)

// Record is the provisional internal handoff from the local gateway to
// buildopt-server. Its optional WS-006 fields support conversion without
// replacing the normative BUILD_SESSION v1 schema.
type Record struct {
	ProtocolVersion             string            `json:"protocolVersion"`
	RecordType                  string            `json:"recordType"`
	SessionID                   string            `json:"sessionId"`
	GatewayConnectionGeneration string            `json:"gatewayConnectionGeneration"`
	StartedAt                   string            `json:"startedAt"`
	CompletedAt                 string            `json:"completedAt"`
	DurationMs                  int64             `json:"durationMs"`
	Outcome                     string            `json:"outcome"`
	ExitCode                    int               `json:"exitCode"`
	ExportContext               *ExportContext    `json:"exportContext,omitempty"`
	GradleInvocation            *GradleInvocation `json:"gradleInvocation,omitempty"`
}

// ExportContext contains pre-outcome facts required by BUILD_SESSION v1 that
// cannot be inferred safely by the launcher.
type ExportContext struct {
	RepositoryID         string   `json:"repositoryId"`
	Revision             string   `json:"revision"`
	RequestedTasks       []string `json:"requestedTasks"`
	SourceStateDigest    string   `json:"sourceStateDigest"`
	WorkUnitsFingerprint string   `json:"workUnitsFingerprint"`
	TokenKeyVersion      string   `json:"tokenKeyVersion"`
	TrustDomain          string   `json:"trustDomain"`
}

// GradleInvocation records the authenticated Gradle process observed by the
// launcher for one session.
type GradleInvocation struct {
	ID            string `json:"id"`
	StartedAt     string `json:"startedAt"`
	CompletedAt   string `json:"completedAt"`
	DurationMs    int64  `json:"durationMs"`
	PluginVersion string `json:"pluginVersion"`
}

// NewRecord creates a provisional ingest record from one launcher invocation.
func NewRecord(
	sessionID string,
	gatewayGeneration string,
	startedAt time.Time,
	completedAt time.Time,
	outcome string,
	exitCode int,
) Record {
	return Record{
		ProtocolVersion:             ProtocolVersion,
		RecordType:                  RecordType,
		SessionID:                   sessionID,
		GatewayConnectionGeneration: gatewayGeneration,
		StartedAt:                   startedAt.UTC().Format(time.RFC3339Nano),
		CompletedAt:                 completedAt.UTC().Format(time.RFC3339Nano),
		DurationMs:                  completedAt.Sub(startedAt).Milliseconds(),
		Outcome:                     outcome,
		ExitCode:                    exitCode,
	}
}

// Validate enforces the WS-005 ingest boundary before storage or delivery.
func (record Record) Validate() error {
	if record.ProtocolVersion != ProtocolVersion {
		return errors.New("unsupported session ingest protocol version")
	}
	if record.RecordType != RecordType {
		return errors.New("unsupported session ingest record type")
	}
	if err := validateIdentifier("session ID", record.SessionID); err != nil {
		return err
	}
	if err := validateIdentifier(
		"gateway connection generation",
		record.GatewayConnectionGeneration,
	); err != nil {
		return err
	}

	startedAt, err := parseUTCTimestamp("startedAt", record.StartedAt)
	if err != nil {
		return err
	}
	completedAt, err := parseUTCTimestamp("completedAt", record.CompletedAt)
	if err != nil {
		return err
	}
	if completedAt.Before(startedAt) {
		return errors.New("completedAt precedes startedAt")
	}
	if record.DurationMs < 0 {
		return errors.New("durationMs must be non-negative")
	}

	switch record.Outcome {
	case OutcomeSuccess:
		if record.ExitCode != 0 {
			return errors.New("successful session must have exitCode 0")
		}
	case OutcomeBuildFailure:
		if record.ExitCode < 1 || record.ExitCode > 255 {
			return errors.New(
				"failed session exitCode must be between 1 and 255",
			)
		}
	case OutcomeCancelled:
		if record.ExitCode < 1 || record.ExitCode > 255 {
			return errors.New(
				"cancelled session exitCode must be between 1 and 255",
			)
		}
	default:
		return fmt.Errorf("unsupported session outcome %q", record.Outcome)
	}

	if (record.ExportContext == nil) != (record.GradleInvocation == nil) {
		return errors.New(
			"BUILD_SESSION export context and Gradle invocation must appear together",
		)
	}
	if record.ExportContext != nil {
		if err := record.ExportContext.Validate(); err != nil {
			return err
		}
		if err := record.GradleInvocation.Validate(); err != nil {
			return err
		}
		invocationStartedAt, _ := parseUTCTimestamp(
			"gradleInvocation.startedAt",
			record.GradleInvocation.StartedAt,
		)
		invocationCompletedAt, _ := parseUTCTimestamp(
			"gradleInvocation.completedAt",
			record.GradleInvocation.CompletedAt,
		)
		if invocationStartedAt.Before(startedAt) ||
			invocationCompletedAt.After(completedAt) {
			return errors.New(
				"Gradle invocation must remain inside the session envelope",
			)
		}
		if record.GradleInvocation.DurationMs > record.DurationMs {
			return errors.New(
				"Gradle invocation duration exceeds the session duration",
			)
		}
	}
	return nil
}

// ExportContextFromEnvironment parses the optional strict BUILD_SESSION
// context supplied before the build outcome is known.
func ExportContextFromEnvironment(
	getenv func(string) string,
) (*ExportContext, bool, error) {
	raw := getenv(ExportContextEnvironment)
	if raw == "" {
		return nil, false, nil
	}
	if len(raw) > maxExportContextBytes {
		return nil, false, errors.New(
			"BUILD_SESSION export context exceeds 32 KiB",
		)
	}

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var context ExportContext
	if err := decoder.Decode(&context); err != nil {
		return nil, false, errors.New(
			"decode BUILD_SESSION export context",
		)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, false, errors.New(
			"BUILD_SESSION export context must contain one JSON value",
		)
	}
	if err := context.Validate(); err != nil {
		return nil, false, err
	}
	return &context, true, nil
}

// Validate enforces the internal BUILD_SESSION context contract.
func (context ExportContext) Validate() error {
	for _, identifier := range []struct {
		name  string
		value string
	}{
		{name: "repository ID", value: context.RepositoryID},
		{name: "revision", value: context.Revision},
		{name: "token key version", value: context.TokenKeyVersion},
		{name: "trust domain", value: context.TrustDomain},
	} {
		if err := validateIdentifier(identifier.name, identifier.value); err != nil {
			return err
		}
	}
	if err := validateRequestedTasks(context.RequestedTasks); err != nil {
		return err
	}
	if err := validateDigest(
		"source state digest",
		context.SourceStateDigest,
		"hmac-sha256:",
	); err != nil {
		return err
	}
	if err := validateDigest(
		"work units fingerprint",
		context.WorkUnitsFingerprint,
		"hmac-sha256:",
	); err != nil {
		return err
	}
	return nil
}

// Validate enforces the authenticated Gradle invocation handoff contract.
func (invocation GradleInvocation) Validate() error {
	if err := validateIdentifier("Gradle invocation ID", invocation.ID); err != nil {
		return err
	}
	startedAt, err := parseUTCTimestamp(
		"gradleInvocation.startedAt",
		invocation.StartedAt,
	)
	if err != nil {
		return err
	}
	completedAt, err := parseUTCTimestamp(
		"gradleInvocation.completedAt",
		invocation.CompletedAt,
	)
	if err != nil {
		return err
	}
	if completedAt.Before(startedAt) {
		return errors.New(
			"gradleInvocation.completedAt precedes gradleInvocation.startedAt",
		)
	}
	if invocation.DurationMs < 0 {
		return errors.New(
			"gradleInvocation.durationMs must be non-negative",
		)
	}
	if invocation.PluginVersion == "" || len(invocation.PluginVersion) > 128 {
		return errors.New(
			"Gradle plugin version must contain 1 to 128 bytes",
		)
	}
	return nil
}

func validateIdentifier(name string, value string) error {
	if value == "" || len(value) > 256 {
		return fmt.Errorf("%s must contain 1 to 256 bytes", name)
	}
	for index := range len(value) {
		character := value[index]
		if isASCIIAlphaNumeric(character) {
			continue
		}
		if index > 0 && strings.ContainsRune("._:/@+-", rune(character)) {
			continue
		}
		return fmt.Errorf("%s contains an unsupported character", name)
	}
	return nil
}

func validateRequestedTasks(tasks []string) error {
	if len(tasks) < 1 || len(tasks) > 1024 {
		return errors.New("requested tasks must contain 1 to 1024 entries")
	}
	seen := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		if task == "" || len(task) > 512 {
			return errors.New(
				"requested task must contain 1 to 512 bytes",
			)
		}
		if _, found := seen[task]; found {
			return errors.New("requested tasks must be unique")
		}
		seen[task] = struct{}{}
	}
	return nil
}

func validateDigest(name string, value string, prefix string) error {
	if !strings.HasPrefix(value, prefix) ||
		len(value) != len(prefix)+64 {
		return fmt.Errorf("%s must use %s followed by 64 lowercase hex digits", name, prefix)
	}
	for _, character := range value[len(prefix):] {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return fmt.Errorf(
					"%s must use %s followed by 64 lowercase hex digits",
					name,
					prefix,
				)
			}
		}
	}
	return nil
}

func isASCIIAlphaNumeric(character byte) bool {
	return character >= 'A' && character <= 'Z' ||
		character >= 'a' && character <= 'z' ||
		character >= '0' && character <= '9'
}

func parseUTCTimestamp(name string, value string) (time.Time, error) {
	if !strings.HasSuffix(value, "Z") {
		return time.Time{}, fmt.Errorf("%s must use UTC", name)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must use RFC 3339: %w", name, err)
	}
	return parsed, nil
}
