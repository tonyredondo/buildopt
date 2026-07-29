package buildsession

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("unexpected export directory entries: %+v", entries)
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
