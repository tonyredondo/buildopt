package stickyobservation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validRecord(now time.Time) Record {
	return Record{
		SchemaVersion:  SchemaVersion,
		RecordType:     RecordType,
		ObservationID:  "ordinary-1",
		IdempotencyKey: Digest("ordinary-1"),
		Provenance: Provenance{
			RepositoryScopeSHA256:  Digest("scope"),
			SourceRevision:         strings.Repeat("a", 40),
			SourceRevisionEvidence: "EXACT",
			GradleVersion:          "9.6.1",
			WrapperSHA256:          Digest("wrapper"),
			BuildOptSHA256:         Digest("buildopt"),
			ArgumentsSHA256:        Digest("args"),
		},
		Outcome:     "SUCCESS",
		ExitCode:    0,
		StartedAt:   now.Format(time.RFC3339Nano),
		CompletedAt: now.Add(120 * time.Millisecond).Format(time.RFC3339Nano),
		Timing: Timing{
			TotalNs:        120_000_000,
			Decision:       Phase{DurationNs: 2_000_000, Evidence: "EXACT"},
			Network:        Phase{Evidence: "UNAVAILABLE"},
			Cache:          Phase{DurationNs: 8_000_000, Evidence: "APPROXIMATED"},
			Gradle:         Phase{DurationNs: 100_000_000, Evidence: "EXACT"},
			Observation:    Phase{DurationNs: 1_000_000, Evidence: "EXACT"},
			Wrapper:        Phase{Evidence: "UNAVAILABLE"},
			Bootstrap:      Phase{Evidence: "UNAVAILABLE"},
			UnattributedNs: 9_000_000,
		},
		ConfigurationCache: ConfigurationCache{Requested: true, State: "PRESENT"},
	}
}

func TestRecordValidationReconcilesExclusivePhases(t *testing.T) {
	now := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	record := validRecord(now)
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	record.Timing.UnattributedNs++
	if err := record.Validate(); err == nil {
		t.Fatal("non-reconciled timing was accepted")
	}
	record = validRecord(now)
	record.Timing.Bootstrap = Phase{DurationNs: 1, Evidence: "UNAVAILABLE"}
	if err := record.Validate(); err == nil {
		t.Fatal("unavailable phase with a fabricated duration was accepted")
	}
	record = validRecord(now)
	record.Provenance.SourceRevisionEvidence = "UNAVAILABLE"
	record.Provenance.SourceRevision = strings.Repeat("b", 40)
	if err := record.Validate(); err == nil {
		t.Fatal("unavailable source revision with a value was accepted")
	}
}

func TestRecorderAppendsCanonicalPrivateJSONL(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "private", "observations.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	record := validRecord(now)
	if err := recorder.Append(record); err != nil {
		t.Fatal(err)
	}
	record.ObservationID = "ordinary-2"
	record.IdempotencyKey = Digest(record.ObservationID)
	record.StartedAt = now.Add(time.Second).Format(time.RFC3339Nano)
	record.CompletedAt = now.Add(time.Second + 120*time.Millisecond).Format(time.RFC3339Nano)
	if err := recorder.Append(record); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || len(loaded) != 2 {
		t.Fatalf("loaded = %d/%v", len(loaded), err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("observation log mode = %v/%v", info, err)
	}
	if _, err := NewRecorder(root + "/../unsafe/observations.jsonl"); err == nil {
		t.Fatal("unclean observation path was accepted")
	}
}

func TestLoadRejectsTamperedRecord(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "private", "observations.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Append(validRecord(time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC))); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"totalNs":120000000`, `"totalNs":120000001`, 1))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("tampered observation was accepted")
	} else if errors.Is(err, os.ErrNotExist) {
		t.Fatal("unexpected missing observation")
	}
}
