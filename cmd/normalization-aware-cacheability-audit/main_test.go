package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRejectsSourceDrift(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "one")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "subjects.json")
	content := `{"schemaVersion":"buildopt.specs/poc-normalization-aware-cacheability-subjects/v2","reusePolicy":"SOURCE_REVISIONS_ONLY_NO_DNO_EVIDENCE_ROWS","families":[` +
		`{"Key":"one","Repository":"https://example.invalid/one.git","Revision":"0000000000000000000000000000000000000000"},` +
		`{"Key":"two","Repository":"https://example.invalid/two.git","Revision":"0000000000000000000000000000000000000000"},` +
		`{"Key":"three","Repository":"https://example.invalid/three.git","Revision":"0000000000000000000000000000000000000000"},` +
		`{"Key":"four","Repository":"https://example.invalid/four.git","Revision":"0000000000000000000000000000000000000000"},` +
		`{"Key":"five","Repository":"https://example.invalid/five.git","Revision":"0000000000000000000000000000000000000000"}]}`
	if err := os.WriteFile(manifest, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(manifest, root, "", "contract"); err == nil {
		t.Fatal("source drift was accepted")
	}
}
