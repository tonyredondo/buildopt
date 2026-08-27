package stickywrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickyobservation"
)

func TestBuildStatusAggregatesValidatedObservationsAndRendersJSON(t *testing.T) {
	root := t.TempDir()
	if _, err := (Generator{Root: root, Resolver: &fakeResolver{latest: fixtureRelease("1.2.3", 'a')}}).Init(context.Background(), configuredFixture()); err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	identity := "example/pilot"
	scope := stickyobservation.ScopeForRoot(identity)
	observationPath := filepath.Join(cacheRoot, "buildopt", "sticky", "observations", scope, "builds.jsonl")
	recorder, err := stickyobservation.NewRecorder(observationPath)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	record := stickyobservation.Record{
		SchemaVersion:  stickyobservation.SchemaVersion,
		RecordType:     stickyobservation.RecordType,
		ObservationID:  "build-status-1",
		IdempotencyKey: stickyobservation.Digest("build-status-1"),
		Provenance: stickyobservation.Provenance{
			RepositoryScopeSHA256:  scope,
			SourceRevision:         strings.Repeat("a", 40),
			SourceRevisionEvidence: "EXACT",
			GradleVersion:          "9.6.1",
			WrapperSHA256:          strings.Repeat("b", 64),
			BuildOptSHA256:         strings.Repeat("c", 64),
			ArgumentsSHA256:        strings.Repeat("d", 64),
		},
		Outcome:     "SUCCESS",
		ExitCode:    0,
		StartedAt:   started.Format(time.RFC3339Nano),
		CompletedAt: started.Add(2 * time.Second).Format(time.RFC3339Nano),
		Timing: stickyobservation.Timing{
			TotalNs:        2 * int64(time.Second),
			Decision:       stickyobservation.Phase{DurationNs: 100 * int64(time.Millisecond), Evidence: "EXACT"},
			Cache:          stickyobservation.Phase{DurationNs: 100 * int64(time.Millisecond), Evidence: "APPROXIMATED"},
			Gradle:         stickyobservation.Phase{DurationNs: 1500 * int64(time.Millisecond), Evidence: "EXACT"},
			Observation:    stickyobservation.Phase{DurationNs: 100 * int64(time.Millisecond), Evidence: "EXACT"},
			Network:        stickyobservation.Phase{Evidence: "UNAVAILABLE"},
			Wrapper:        stickyobservation.Phase{Evidence: "UNAVAILABLE"},
			Bootstrap:      stickyobservation.Phase{Evidence: "UNAVAILABLE"},
			UnattributedNs: 200 * int64(time.Millisecond),
		},
		ConfigurationCache: stickyobservation.ConfigurationCache{Requested: false, State: "NOT_REQUESTED"},
	}
	if err := recorder.Append(record); err != nil {
		t.Fatal(err)
	}
	report, err := BuildStatus(root, "STATUS")
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision.State != "NATIVE" || report.Observations.Count != 1 || report.Observations.Successful != 1 || report.Observations.WallTime.Value == nil || *report.Observations.WallTime.Value != 2000 || report.Observations.GradleTime.Value == nil || *report.Observations.GradleTime.Value != 1500 {
		t.Fatalf("unexpected status report: %+v", report)
	}
	var machine bytes.Buffer
	if err := WriteReport(report, true, &machine); err != nil {
		t.Fatal(err)
	}
	var decoded StatusReport
	if err := json.Unmarshal(machine.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ReportType != "STATUS" || decoded.Observations.WallTime.Value == nil || *decoded.Observations.WallTime.Value != 2000 {
		t.Fatalf("JSON report lost values: %+v", decoded)
	}
	var human bytes.Buffer
	if err := WriteReport(report, false, &human); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(human.String(), "Wall time: 2000 milliseconds.") || strings.Contains(human.String(), "BUILDOPT_") || strings.Contains(human.String(), root) {
		t.Fatalf("unsafe or incomplete human report: %q", human.String())
	}
	explain, err := BuildStatus(root, "EXPLAIN")
	if err != nil || explain.ReportType != "EXPLAIN" || len(explain.Explanation) < 5 {
		t.Fatalf("explain report = %+v/%v", explain, err)
	}
}

func TestBuildStatusIsReadOnlyAndRejectsTamperedObservationLog(t *testing.T) {
	root := t.TempDir()
	if _, err := (Generator{Root: root, Resolver: &fakeResolver{latest: fixtureRelease("1.2.3", 'a')}}).Init(context.Background(), configuredFixture()); err != nil {
		t.Fatal(err)
	}
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	before := captureTree(t, root)
	report, err := BuildStatus(root, "STATUS")
	if err != nil {
		t.Fatal(err)
	}
	if report.Observations.Count != 0 || report.Observations.WallTime.State != "UNAVAILABLE" || report.Decision.State != "NATIVE" {
		t.Fatalf("empty status report = %+v", report)
	}
	if after := captureTree(t, root); len(after) != len(before) {
		t.Fatalf("status changed repository tree: before=%#v after=%#v", before, after)
	}

	identity := "example/pilot"
	scope := stickyobservation.ScopeForRoot(identity)
	path := filepath.Join(cacheRoot, "buildopt", "sticky", "observations", scope, "builds.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"tampered\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildStatus(root, "STATUS"); err == nil {
		t.Fatal("tampered observation log was accepted")
	}
}

func TestStatusMeasurementRejectsUnavailableZero(t *testing.T) {
	value := int64(0)
	report := StatusReport{
		SchemaVersion: StatusSchemaVersion,
		ReportType:    "STATUS",
		Repository:    strings.Repeat("a", 64),
		Observations: ObservationStatus{
			WallTime: Measurement{State: "UNAVAILABLE", Value: &value},
		},
	}
	if err := report.Validate(); err == nil {
		t.Fatal("unavailable numeric zero was accepted")
	}
}

func TestBuildStatusLoadsVerifiedLifecycleEconomics(t *testing.T) {
	root := t.TempDir()
	if _, err := (Generator{Root: root, Resolver: &fakeResolver{latest: fixtureRelease("1.2.3", 'a')}}).Init(context.Background(), configuredFixture()); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join("..", "..", "benchmarks", "results", "sticky-wrapper-learning-lifecycle-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(source, &document); err != nil {
		t.Fatal(err)
	}
	scope := stickyobservation.ScopeForRoot("example/pilot")
	ledger := document["ledger"].(map[string]any)
	binding := ledger["binding"].(map[string]any)
	binding["repositoryScopeSha256"] = scope
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "lifecycle.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(learningLifecycleOutputEnv, path)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	report, err := BuildStatus(root, "STATUS")
	if err != nil {
		t.Fatal(err)
	}
	if report.Trials.Count.Value == nil || *report.Trials.Count.Value != 4 || report.Economics.NetSavedMs.Value == nil || *report.Economics.NetSavedMs.Value != 2 || report.Decision.StoredDecision != stickydecision.ExecutionRetired || !report.Fallback.Applied {
		t.Fatalf("lifecycle status = %+v", report)
	}
}
