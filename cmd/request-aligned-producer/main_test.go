package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadStrictRejectsUnknownFields(t *testing.T) {
	file := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(file, []byte(`{"known":true,"unknown":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var target struct {
		Known bool `json:"known"`
	}
	if err := readStrict(file, &target); err == nil {
		t.Fatal("unknown field was accepted")
	}
}

func TestWriteJSONCreatesCanonicalDocument(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "output.json")
	if err := writeJSON(file, map[string]bool{"ok": true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{\n  \"ok\": true\n}\n" {
		t.Fatalf("unexpected JSON: %q", raw)
	}
}
