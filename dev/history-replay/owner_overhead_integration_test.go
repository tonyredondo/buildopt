//go:build linux && amd64 && replay_integration && replay_gradle && replay_owner

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGradleOwnerSymmetricCaptureOverhead(t *testing.T) {
	path := os.Getenv("BUILDOPT_REPLAY_OWNER_OVERHEAD_MANIFEST")
	if !filepath.IsAbs(path) {
		t.Fatal("owner overhead requires an explicitly frozen fresh manifest")
	}
	var m Manifest
	if err := readJSON(path, &m); err != nil {
		t.Fatal(err)
	}
	if m.Subject != "elasticsearch-owner-overhead" || m.Phase != "QUALIFICATION" || m.Replications != 1 || len(m.History) != 1 || m.ExecutionEnd != 0 || m.Limits.MaxWorkflowStarts != 20 || m.Limits.MaxRequestNS != int64(900*time.Second) || len(m.Candidate.Files) != 5 || absentBinding(m.Outputs.Owner) {
		t.Fatal("owner overhead scope/allocation differs")
	}
	measureCaptureOverhead(t, m, true)
}

func TestGradleOwnerCapturePrecision(t *testing.T) {
	path := os.Getenv("BUILDOPT_REPLAY_OWNER_OVERHEAD_MANIFEST")
	if !filepath.IsAbs(path) {
		t.Fatal("owner precision requires an explicitly frozen fresh manifest")
	}
	var m Manifest
	if err := readJSON(path, &m); err != nil {
		t.Fatal(err)
	}
	if m.Phase != "QUALIFICATION" || m.Replications != 1 || len(m.History) != 1 || m.ExecutionEnd != 0 || m.Limits.MaxRequestNS != int64(900*time.Second) || len(m.Candidate.Files) != 5 || absentBinding(m.Outputs.Owner) || absentBinding(m.Overhead) {
		t.Fatal("owner precision scope differs")
	}
	mode, pairs := "precision-pilot", 16
	switch m.Subject {
	case "elasticsearch-owner-precision-pilot":
		if m.Limits.MaxWorkflowStarts != 80 {
			t.Fatal("precision pilot requires exactly80 starts")
		}
	case "elasticsearch-owner-precision-confirmation":
		mode = "precision-confirmation"
		pairs = (m.Limits.MaxWorkflowStarts - 16) / 4
		if pairs < 64 || pairs > 512 || pairs%8 != 0 || m.Limits.MaxWorkflowStarts != 16+4*pairs {
			t.Fatal("confirmation sample count differs from the bounded design")
		}
	default:
		t.Fatal("unknown owner precision stage")
	}
	if m.Limits.MaxGradleStarts != m.Limits.MaxWorkflowStarts {
		t.Fatal("precision workflow and Gradle budgets differ")
	}
	measureCaptureOverheadDesign(t, m, true, 4, pairs, mode)
}
