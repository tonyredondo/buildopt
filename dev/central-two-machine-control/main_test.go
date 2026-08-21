package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestCommitCanAbortAnEmptyProducerAttempt(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "state")
	control := filepath.Join(root, "control")
	if err := prepare([]string{
		"--state-dir", state,
		"--output-dir", control,
		"--repository-id", "example/empty-cache-producer",
		"--source-revision", strings.Repeat("a", 40),
		"--namespace", "qualified-lifetime-v2/test",
	}); err != nil {
		t.Fatalf("prepare empty producer attempt: %v", err)
	}

	arguments := []string{"--state-dir", state, "--control-dir", control}
	if err := commit(arguments); err == nil ||
		!strings.Contains(err.Error(), "pending producer objects are empty") {
		t.Fatalf("strict empty commit error = %v", err)
	}
	if err := commit(append(arguments, "--abort-empty")); err != nil {
		t.Fatalf("abort empty producer attempt: %v", err)
	}

	storage, err := sharedcache.Open(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	status, err := storage.AttemptStatus(
		context.Background(),
		"22222222-2222-4222-8222-222222222222",
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != sharedcache.AttemptAborted ||
		status.PendingObjectCount != 0 ||
		status.AbortReason != "INCOMPLETE_COMMIT_DECISION" {
		t.Fatalf("empty producer status = %+v", status)
	}
}
