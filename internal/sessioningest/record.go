package sessioningest

import (
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
)

// Record is the provisional internal handoff from the local gateway to
// buildopt-server. WS-006 owns conversion to the normative BUILD_SESSION v1
// export.
type Record struct {
	ProtocolVersion             string `json:"protocolVersion"`
	RecordType                  string `json:"recordType"`
	SessionID                   string `json:"sessionId"`
	GatewayConnectionGeneration string `json:"gatewayConnectionGeneration"`
	StartedAt                   string `json:"startedAt"`
	CompletedAt                 string `json:"completedAt"`
	DurationMs                  int64  `json:"durationMs"`
	Outcome                     string `json:"outcome"`
	ExitCode                    int    `json:"exitCode"`
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
	default:
		return fmt.Errorf("unsupported session outcome %q", record.Outcome)
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
