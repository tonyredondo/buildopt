package buildsession

import (
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

func TestNewDocumentDeclaresObservedAndUnavailableValues(t *testing.T) {
	record := validExportRecord()
	document, err := NewDocument(record)
	if err != nil {
		t.Fatalf("create BUILD_SESSION document: %v", err)
	}

	if document.SchemaVersion != "1.0" ||
		document.RecordType != "BUILD_SESSION" ||
		!document.Complete {
		t.Fatalf("unexpected envelope: %+v", document)
	}
	if document.Build.ID != record.SessionID ||
		document.Build.Outcome != sessioningest.OutcomeSuccess ||
		document.Build.ExitCode != 0 ||
		document.Build.RequiredDeliverablesStatus != "NOT_REQUIRED" {
		t.Fatalf("unexpected build: %+v", document.Build)
	}
	if len(document.GradleInvocations) != 1 ||
		document.GradleInvocations[0].ID != record.GradleInvocation.ID ||
		document.GradleInvocations[0].ProcessMs.Method != "APPROXIMATED" ||
		document.GradleInvocations[0].ProcessMs.MethodDescription == "" ||
		document.GradleInvocations[0].ProcessMs.ValueMs == nil ||
		*document.GradleInvocations[0].ProcessMs.ValueMs !=
			record.GradleInvocation.DurationMs {
		t.Fatalf(
			"unexpected Gradle invocation: %+v",
			document.GradleInvocations,
		)
	}
	if document.Performance.CustomerVisibleBuildMs.ValueMs == nil ||
		*document.Performance.CustomerVisibleBuildMs.ValueMs !=
			record.DurationMs {
		t.Fatalf(
			"unexpected customer-visible duration: %+v",
			document.Performance.CustomerVisibleBuildMs,
		)
	}
	if document.Performance.CustomerVisibleFeedbackMs.Method !=
		"UNAVAILABLE" ||
		document.Performance.CustomerVisibleFeedbackMs.ValueMs != nil ||
		document.Capabilities.TaskOutcomes.Method != "UNAVAILABLE" {
		t.Fatalf("unavailable metrics invented a value: %+v", document)
	}
	if !strings.HasPrefix(
		document.ExperimentAssignment.BaselineDefinitionDigest,
		"sha256:",
	) ||
		!strings.HasPrefix(
			document.Workload.RequestedWorkManifestDigest,
			"sha256:",
		) {
		t.Fatalf("missing derived digests: %+v", document)
	}

	record.ExportContext.RequestedTasks[0] = "mutated-after-export"
	if document.GradleInvocations[0].RequestedTasks[0] != "neutralProbe" {
		t.Fatal("document retained mutable ingest context")
	}
}

func TestNewDocumentRepresentsBuildFailureAsPartialFailureTiming(
	t *testing.T,
) {
	record := validExportRecord()
	record.Outcome = sessioningest.OutcomeBuildFailure
	record.ExitCode = 37

	document, err := NewDocument(record)
	if err != nil {
		t.Fatalf("create failed BUILD_SESSION document: %v", err)
	}
	measurement := document.Performance.TimeToFirstBuildFailureMs
	if document.Build.Outcome != sessioningest.OutcomeBuildFailure ||
		document.Build.ExitCode != 37 ||
		measurement.State != "PARTIAL" ||
		measurement.Method != "APPROXIMATED" ||
		measurement.MethodDescription == "" ||
		measurement.ValueMs == nil ||
		*measurement.ValueMs != record.DurationMs ||
		measurement.Reason == "" {
		t.Fatalf("unexpected failure document: %+v", document)
	}
}

func TestNewDocumentRejectsIncompleteExportInput(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(*sessioningest.Record)
		want   string
	}{
		{
			name: "missing context and invocation",
			mutate: func(record *sessioningest.Record) {
				record.ExportContext = nil
				record.GradleInvocation = nil
			},
			want: "required",
		},
		{
			name: "non-schema plugin version",
			mutate: func(record *sessioningest.Record) {
				record.GradleInvocation.PluginVersion = "development"
			},
			want: "schema-compatible",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			record := validExportRecord()
			testCase.mutate(&record)
			if _, err := NewDocument(record); err == nil ||
				!strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("document error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func validExportRecord() sessioningest.Record {
	startedAt := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	record := sessioningest.NewRecord(
		"session-export-test",
		"gateway-export-test",
		startedAt,
		startedAt.Add(2*time.Second),
		sessioningest.OutcomeSuccess,
		0,
	)
	record.ExportContext = &sessioningest.ExportContext{
		RepositoryID:         "repository-export-test",
		Revision:             "revision-export-test",
		RequestedTasks:       []string{"neutralProbe"},
		SourceStateDigest:    "hmac-sha256:" + strings.Repeat("a", 64),
		WorkUnitsFingerprint: "hmac-sha256:" + strings.Repeat("b", 64),
		TokenKeyVersion:      "fixture-token-v1",
		TrustDomain:          "fixture-local",
	}
	record.GradleInvocation = &sessioningest.GradleInvocation{
		ID:            "gradle-invocation-export-test",
		StartedAt:     startedAt.Add(100 * time.Millisecond).Format(time.RFC3339Nano),
		CompletedAt:   startedAt.Add(1900 * time.Millisecond).Format(time.RFC3339Nano),
		DurationMs:    1800,
		PluginVersion: "0.1.0-SNAPSHOT",
	}
	return record
}
