package buildsession

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

func TestExporterPublishesPrivateImmutableJSON(t *testing.T) {
	exporter, err := NewExporter(filepath.Join(t.TempDir(), "exports"))
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	record := validExportRecord()

	path, created, err := exporter.Export(record)
	if err != nil {
		t.Fatalf("export BUILD_SESSION: %v", err)
	}
	if !created || filepath.Base(path) != exportFilename(record.SessionID) {
		t.Fatalf("first export = %s/%v, want created target", path, created)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat export: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("export mode = %o, want 600", info.Mode().Perm())
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	var document Document
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if document.Build.ID != record.SessionID {
		t.Fatalf("export build ID = %q", document.Build.ID)
	}

	replayedPath, replayedCreated, err := exporter.Export(record)
	if err != nil {
		t.Fatalf("replay identical export: %v", err)
	}
	if replayedCreated || replayedPath != path {
		t.Fatalf(
			"replayed export = %s/%v, want %s/false",
			replayedPath,
			replayedCreated,
			path,
		)
	}

	conflicting := validExportRecord()
	conflicting.ExportContext.Revision = "different-revision"
	if _, _, err := exporter.Export(conflicting); !errors.Is(
		err,
		ErrExportConflict,
	) {
		t.Fatalf("conflicting export error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("list export directory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected export directory entries: %+v", entries)
	}
	streamPath := filepath.Join(filepath.Dir(path), "buildopt-events.jsonl")
	streamInfo, err := os.Stat(streamPath)
	if err != nil {
		t.Fatalf("stat JSONL stream: %v", err)
	}
	if streamInfo.Mode().Perm() != 0o600 {
		t.Fatalf("JSONL stream mode = %o, want 600", streamInfo.Mode().Perm())
	}
	var stream bytes.Buffer
	if err := exporter.WriteJSONL(&stream); err != nil {
		t.Fatalf("copy JSONL stream: %v", err)
	}
	lines := bytes.Split(bytes.TrimSuffix(stream.Bytes(), []byte{'\n'}), []byte{'\n'})
	if len(lines) != 4 {
		t.Fatalf("JSONL line count = %d, want 4", len(lines))
	}
	if !bytes.Equal(lines[0], lines[2]) ||
		!bytes.Equal(lines[1], lines[3]) {
		t.Fatal("identical replay did not preserve byte-identical events")
	}
	for index, line := range lines {
		event, err := decodeExportEvent(line)
		if err != nil {
			t.Fatalf("decode JSONL line %d: %v", index, err)
		}
		wantSequence := index%2 + 1
		if event.BuildID != record.SessionID ||
			event.Sequence != wantSequence {
			t.Fatalf("JSONL event %d = %+v", index, event)
		}
	}
}

func TestNewExporterRejectsFilePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("create occupied path: %v", err)
	}
	if _, err := NewExporter(path); err == nil {
		t.Fatal("accepted a file as the export directory")
	}
}

func TestExporterRecoversPartialSessionFromMissingFinalEvent(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "exports")
	recoveredAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	exporter, err := newExporter(directory, func() time.Time {
		return recoveredAt
	})
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	record := validExportRecord()
	document, content, events := exportFixture(t, exporter, record)
	if err := exporter.stream.append(events[0]); err != nil {
		t.Fatalf("append observed event: %v", err)
	}

	exporter, err = newExporter(directory, func() time.Time {
		return recoveredAt
	})
	if err != nil {
		t.Fatalf("recover exporter: %v", err)
	}
	partialPath := filepath.Join(
		directory,
		partialExportFilename(record.SessionID, events[0].EventID),
	)
	partialContent, err := os.ReadFile(partialPath)
	if err != nil {
		t.Fatalf("read partial recovery: %v", err)
	}
	var partial Document
	if err := json.Unmarshal(partialContent, &partial); err != nil {
		t.Fatalf("decode partial recovery: %v", err)
	}
	if partial.Complete ||
		partial.MeasurementMetadata.Status != "PARTIAL" ||
		partial.Performance.CustomerVisibleBuildMs.State != "PARTIAL" ||
		partial.Recovery == nil ||
		partial.Recovery.Source != "EVENT_REPLAY" ||
		partial.Recovery.RecoveredAt != recoveredAt.Format(time.RFC3339Nano) ||
		len(partial.Recovery.MissingSequenceRanges) != 1 ||
		partial.Recovery.MissingSequenceRanges[0] !=
			(MissingSequenceRange{First: 2, Last: 2}) {
		t.Fatalf("unexpected partial recovery: %+v", partial)
	}
	if document.Complete != true || len(content) == 0 {
		t.Fatal("fixture lost its complete source document")
	}

	completePath, created, err := exporter.Export(record)
	if err != nil {
		t.Fatalf("complete recovered export: %v", err)
	}
	if !created || filepath.Base(completePath) != exportFilename(record.SessionID) {
		t.Fatalf("completed export = %s/%v", completePath, created)
	}
	if _, err := NewExporter(directory); err != nil {
		t.Fatalf("reopen completed exporter: %v", err)
	}
}

func TestExporterReplaysPublicationAfterCompleteDocumentSurvives(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "exports")
	exporter, err := NewExporter(directory)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	record := validExportRecord()
	_, content, events := exportFixture(t, exporter, record)
	if err := exporter.stream.append(events[0]); err != nil {
		t.Fatalf("append observed event: %v", err)
	}
	completePath := filepath.Join(directory, exportFilename(record.SessionID))
	if err := publishPrivateFile(
		directory,
		completePath,
		content,
		"test BUILD_SESSION",
	); err != nil {
		t.Fatalf("publish complete fixture: %v", err)
	}

	reopened, err := NewExporter(directory)
	if err != nil {
		t.Fatalf("reopen exporter: %v", err)
	}
	var stream bytes.Buffer
	if err := reopened.WriteJSONL(&stream); err != nil {
		t.Fatalf("copy healed stream: %v", err)
	}
	lines := bytes.Split(
		bytes.TrimSuffix(stream.Bytes(), []byte{'\n'}),
		[]byte{'\n'},
	)
	if len(lines) != 2 {
		t.Fatalf("healed JSONL line count = %d, want 2", len(lines))
	}
	event, err := decodeExportEvent(lines[1])
	if err != nil {
		t.Fatalf("decode healed publication: %v", err)
	}
	if event.Sequence != publishedEventSequence {
		t.Fatalf("healed event sequence = %d, want 2", event.Sequence)
	}
	partials, err := filepath.Glob(
		filepath.Join(directory, "build-session-*-partial.json"),
	)
	if err != nil || len(partials) != 0 {
		t.Fatalf("unexpected partial recovery = %v/%v", partials, err)
	}
}

func TestExporterRepairsOnlyTruncatedJSONLTail(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "exports")
	exporter, err := NewExporter(directory)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	record := validExportRecord()
	_, _, events := exportFixture(t, exporter, record)
	if err := exporter.stream.append(events[0]); err != nil {
		t.Fatalf("append observed event: %v", err)
	}
	file, err := os.OpenFile(
		exporter.stream.path,
		os.O_APPEND|os.O_WRONLY,
		0,
	)
	if err != nil {
		t.Fatalf("open JSONL tail: %v", err)
	}
	if _, err := file.WriteString(`{"eventId":"truncated"`); err != nil {
		_ = file.Close()
		t.Fatalf("append truncated JSONL tail: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close JSONL tail: %v", err)
	}

	if _, err := NewExporter(directory); err != nil {
		t.Fatalf("repair exporter: %v", err)
	}
	content, err := os.ReadFile(exporter.stream.path)
	if err != nil {
		t.Fatalf("read repaired stream: %v", err)
	}
	if !bytes.HasSuffix(content, []byte{'\n'}) ||
		bytes.Contains(content, []byte("truncated")) {
		t.Fatalf("JSONL tail was not repaired: %q", content)
	}
	if _, err := os.Stat(filepath.Join(
		directory,
		partialExportFilename(record.SessionID, events[0].EventID),
	)); err != nil {
		t.Fatalf("partial recovery after repair: %v", err)
	}
}

func TestExporterRejectsMalformedCompleteJSONLLine(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "exports")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatalf("create export directory: %v", err)
	}
	streamPath := filepath.Join(directory, "buildopt-events.jsonl")
	if err := os.WriteFile(
		streamPath,
		[]byte("{not-json}\n"),
		0o600,
	); err != nil {
		t.Fatalf("write malformed JSONL stream: %v", err)
	}
	if _, err := NewExporter(directory); err == nil {
		t.Fatal("accepted a malformed complete JSONL line")
	}
	content, err := os.ReadFile(streamPath)
	if err != nil {
		t.Fatalf("read rejected JSONL stream: %v", err)
	}
	if string(content) != "{not-json}\n" {
		t.Fatalf("malformed complete line was modified: %q", content)
	}
}

func TestExporterRejectsConflictingJSONLSequence(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "exports")
	exporter, err := NewExporter(directory)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	record := validExportRecord()
	_, _, events := exportFixture(t, exporter, record)
	if err := exporter.stream.append(events[0]); err != nil {
		t.Fatalf("append observed event: %v", err)
	}
	var payload observedEventPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("decode observed payload: %v", err)
	}
	payload.Document.Build.Revision = "different-revision"
	changedPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode conflicting payload: %v", err)
	}
	conflicting := newExportEvent(
		record.SessionID,
		observedEventSequence,
		events[0].OccurredAt,
		changedPayload,
	)
	if err := exporter.stream.append(conflicting); err != nil {
		t.Fatalf("append individually valid conflicting event: %v", err)
	}

	if _, err := NewExporter(directory); !errors.Is(
		err,
		errJSONLEventConflict,
	) {
		t.Fatalf("conflicting sequence error = %v", err)
	}
}

func TestExporterBoundsJSONLBeforePublishingDocument(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "exports")
	exporter, err := NewExporter(directory)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	record := validExportRecord()
	_, _, events := exportFixture(t, exporter, record)
	firstLine, err := encodeExportEventLine(events[0])
	if err != nil {
		t.Fatalf("encode first event: %v", err)
	}
	exporter.stream.maximumBytes = int64(len(firstLine) + 1)
	if _, _, err := exporter.Export(record); !errors.Is(
		err,
		errJSONLFull,
	) {
		t.Fatalf("bounded export error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(
		directory,
		exportFilename(record.SessionID),
	)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("complete JSON published despite full JSONL stream: %v", err)
	}
	if info, err := os.Stat(exporter.stream.path); err == nil ||
		!errors.Is(err, os.ErrNotExist) {
		t.Fatalf("JSONL stream was modified despite failed preflight: %v/%v", info, err)
	}
}

func exportFixture(
	t *testing.T,
	exporter *Exporter,
	record sessioningest.Record,
) (Document, []byte, [2]exportEvent) {
	t.Helper()
	document, err := NewDocument(record)
	if err != nil {
		t.Fatalf("create fixture document: %v", err)
	}
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatalf("encode fixture document: %v", err)
	}
	content = append(content, '\n')
	events, err := newBuildSessionEvents(
		document,
		filepath.Join(exporter.directory, exportFilename(record.SessionID)),
		content,
	)
	if err != nil {
		t.Fatalf("create fixture events: %v", err)
	}
	return document, content, events
}
