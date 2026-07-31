package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/runtimeoptimizer"
)

func TestRunAcceptsValidOwnerEvidence(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	digest := strings.Repeat("a", 64)
	request := input{
		SchemaVersion: inputSchema, BuildOptRevision: strings.Repeat("b", 40),
		WorkflowRunID: 42, WorkflowRunURL: "https://github.com/owner/repo/actions/runs/42",
		Runner: runnerInput{
			Class: runtimeoptimizer.GoldenRunnerClass, CPUCount: 4, MemoryBytes: 16 << 30,
			OOMBefore: 2, OOMAfter: 2, QueueMeasurement: "COMMON_WORKFLOW_JOB_EXACT_ZERO_INCREMENTAL",
		},
		AA: pairedInput{
			ControlMS: []int64{4000, 4010, 3990, 4005}, CandidateMS: []int64{3995, 4005, 4000, 3995},
			ControlSHA256: []string{digest, digest, digest, digest}, CandidateSHA256: []string{digest, digest, digest, digest},
		},
		Autotuning: pairedInput{
			ControlMS: []int64{8000, 8200, 8100, 8300}, CandidateMS: []int64{5000, 5200, 5100, 5300},
			ControlSHA256: []string{digest, digest, digest, digest}, CandidateSHA256: []string{digest, digest, digest, digest},
		},
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(root, "input.json")
	outputPath := filepath.Join(root, "output.json")
	if err := os.WriteFile(inputPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, outputPath); err != nil {
		t.Fatal(err)
	}
	resultRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var result evidence
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Gate.State != "PASSED" || result.AA.SampleRatio.Status != runtimeoptimizer.SampleRatioValid ||
		result.AA.DelayedOutcomesUpdated != 200 || result.AA.DuplicateOutcomeUpdated ||
		result.Autotuning.Metrics.Interval95MS[0] <= 0 || result.Autotuning.Metrics.P95DeltaMS > 0 {
		t.Fatalf("unexpected evidence: %+v", result)
	}
}

func TestMetricsRejectsArtifactDivergenceAtGate(t *testing.T) {
	t.Parallel()
	control := strings.Repeat("a", 64)
	candidate := strings.Repeat("b", 64)
	result, err := metrics(pairedInput{
		ControlMS: []int64{10, 10, 10, 10}, CandidateMS: []int64{5, 5, 5, 5},
		ControlSHA256:   []string{control, control, control, control},
		CandidateSHA256: []string{candidate, candidate, candidate, candidate},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ArtifactDivergence {
		t.Fatal("artifact divergence was not retained")
	}
}
