package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func validInput() input {
	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return input{
		SchemaVersion:    inputSchema,
		BuildOptRevision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Workflow:         workflow{Name: "Task Intelligence Pilot Evaluation", RunID: 1, URL: "https://example.invalid/run/1"},
		Runner:           runner{Class: "linux-amd64-4c-16g-v1", CPUCount: 4, MemoryBytes: 16 << 30},
		Pilot:            pilot{Repository: "tonyredondo/buildopt-pilot", BaseRevision: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", AcceptedRevision: "cccccccccccccccccccccccccccccccccccccccc", PullRequest: 1, PullRequestURL: "https://example.invalid/pr/1"},
		ControlMS:        []int64{1000, 1010, 1020, 1030}, CandidateMS: []int64{100, 110, 120, 130},
		ControlSHA256: []string{sha, sha, sha, sha}, CandidateSHA256: []string{sha, sha, sha, sha},
		ControlOutcomes: []string{"EXECUTED", "EXECUTED", "EXECUTED", "EXECUTED"}, CandidateOutcomes: []string{"FROM_CACHE", "FROM_CACHE", "FROM_CACHE", "FROM_CACHE"},
	}
}

func writeRequest(t *testing.T, request input) (string, string) {
	t.Helper()
	root := t.TempDir()
	inputPath, outputPath := filepath.Join(root, "input.json"), filepath.Join(root, "output.json")
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return inputPath, outputPath
}

func TestRunProducesPassingImmutableEvidence(t *testing.T) {
	inputPath, outputPath := writeRequest(t, validInput())
	if err := run(inputPath, outputPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var result evidence
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Gate.State != "PASSED" || result.Measurement.Interval95MS[0] <= 0 || result.Measurement.ArtifactDivergence {
		t.Fatalf("result = %+v", result)
	}
}

func TestRunRejectsNonCacheHitAndArtifactDivergence(t *testing.T) {
	request := validInput()
	request.CandidateOutcomes[2] = "EXECUTED"
	inputPath, outputPath := writeRequest(t, request)
	if err := run(inputPath, outputPath); err == nil {
		t.Fatal("non-cache candidate passed")
	}
	request = validInput()
	request.CandidateSHA256[1] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	inputPath, outputPath = writeRequest(t, request)
	if err := run(inputPath, outputPath); err == nil {
		t.Fatal("divergent artifact passed")
	}
}
