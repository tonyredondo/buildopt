//go:build linux && amd64

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceCopiesPreserveCompleteIndependentFiles(t *testing.T) {
	root := t.TempDir()
	jobs := []fileCopy{}
	for index := 0; index < 96; index++ {
		source := filepath.Join(root, fmt.Sprintf("source-%03d", index))
		content := bytes.Repeat([]byte{byte(index)}, 4096+index)
		if err := os.WriteFile(source, content, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(source, 0444); err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, fileCopy{source, source + "-copy"})
	}
	if err := copyEvidenceFiles(jobs); err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		left, err := os.Stat(job.Source)
		if err != nil {
			t.Fatal(err)
		}
		right, err := os.Stat(job.Destination)
		if err != nil {
			t.Fatal(err)
		}
		a, err := fileDigest(job.Source)
		if err != nil {
			t.Fatal(err)
		}
		b, err := fileDigest(job.Destination)
		if err != nil || a != b || os.SameFile(left, right) || left.Mode() != right.Mode() || left.Size() != right.Size() {
			t.Fatalf("retained file differs or aliases native state: %s: %v", job.Source, err)
		}
	}
	// A later mutation of retained bytes cannot change the native workspace.
	first := jobs[0]
	before, _ := fileDigest(first.Source)
	if err := os.Chmod(first.Destination, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first.Destination, []byte("tampered evidence"), 0644); err != nil {
		t.Fatal(err)
	}
	if after, err := fileDigest(first.Source); err != nil || before != after {
		t.Fatalf("native source changed with evidence: %v", err)
	}
}

func TestEvidenceCopyFailureNeverOverwritesExistingBytes(t *testing.T) {
	root := t.TempDir()
	source, destination := filepath.Join(root, "source"), filepath.Join(root, "existing")
	for _, path := range []string{source, destination} {
		if err := os.WriteFile(path, []byte(path), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyEvidenceFiles([]fileCopy{{source, destination}, {filepath.Join(root, "missing"), filepath.Join(root, "absent")}}); err == nil {
		t.Fatal("incomplete evidence copy succeeded")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != destination {
		t.Fatalf("existing evidence was replaced: %v", err)
	}
}
