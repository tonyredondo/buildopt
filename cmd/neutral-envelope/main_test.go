package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUsage(t *testing.T) {
	if exitCode := run(nil); exitCode != 64 {
		t.Fatalf("empty usage exit = %d, want 64", exitCode)
	}
	if exitCode := run([]string{"unknown"}); exitCode != 64 {
		t.Fatalf("unknown command exit = %d, want 64", exitCode)
	}
}

func TestObserveCommand(t *testing.T) {
	directory := t.TempDir()
	deliverable := filepath.Join(directory, "deliverable.txt")
	observation := filepath.Join(directory, "observation.json")
	exitCode := run([]string{
		"observe",
		"--arm", "NATIVE",
		"--pair", "1",
		"--order", "1",
		"--command-class", "test-command",
		"--deliverable", deliverable,
		"--output", observation,
		"--",
		"/bin/sh",
		"-c",
		`printf 'deterministic\n' >"$1"`,
		"neutral-envelope-test",
		deliverable,
	})
	if exitCode != 0 {
		t.Fatalf("observe exit = %d, want 0", exitCode)
	}
	if info, err := os.Stat(observation); err != nil {
		t.Fatalf("stat observation: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("observation mode = %o, want 600", info.Mode().Perm())
	}
}
