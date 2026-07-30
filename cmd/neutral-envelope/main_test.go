package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/neutralenvelope"
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

func TestPilotObserveClassifiesCommandStartFailureAsInfrastructure(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatalf("chmod private directory: %v", err)
	}
	assignmentPath := filepath.Join(directory, "assignment.json")
	observationPath := filepath.Join(directory, "observation.json")
	assignment, err := neutralenvelope.NewPilotAssignment(
		neutralenvelope.PilotDefinition{
			ExperimentID:             "pilot-start-failure",
			MeasurementEpoch:         1,
			ActionID:                 "managed-l1-cache-hit-v1",
			BaselineDefinitionDigest: "sha256:" + strings.Repeat("a", 64),
			ControlDefinitionDigest:  "sha256:" + strings.Repeat("b", 64),
			CohortID:                 "internal-linux-amd64",
			Environment:              "LOCAL",
			PipelineClass:            "unit-test",
			RunnerClass:              "unit-test",
			WorkUnitsFingerprint:     "hmac-sha256:" + strings.Repeat("c", 64),
			RequiredDeliverable:      "libs/pilot.jar",
		},
		1,
		"CONTROL",
		time.Now().Add(-time.Second),
	)
	if err != nil {
		t.Fatalf("NewPilotAssignment: %v", err)
	}
	if err := neutralenvelope.WritePilotAssignment(
		assignmentPath,
		assignment,
	); err != nil {
		t.Fatalf("WritePilotAssignment: %v", err)
	}

	exitCode := run([]string{
		"pilot-observe",
		"--assignment", assignmentPath,
		"--deliverable", filepath.Join(directory, "missing.jar"),
		"--output", observationPath,
		"--",
		filepath.Join(directory, "missing-command"),
	})
	if exitCode != 1 {
		t.Fatalf("pilot-observe exit = %d, want 1", exitCode)
	}
	observation, err := neutralenvelope.LoadPilotObservation(observationPath)
	if err != nil {
		t.Fatalf("LoadPilotObservation: %v", err)
	}
	if observation.Outcome != "INFRA_FAILURE" {
		t.Fatalf("outcome = %q, want INFRA_FAILURE", observation.Outcome)
	}
}
