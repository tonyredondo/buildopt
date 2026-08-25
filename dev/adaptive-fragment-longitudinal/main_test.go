package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSummaryPreservesNegativeAndInconclusiveRows(t *testing.T) {
	rows := []resultRow{
		{Outcome: "NET_POSITIVE", MeasuredBuilds: 1, SelectedBuilds: 1, Observations: []observation{{ControlWallMS: 20, BuildOptWallMS: 10, SignedDeltaMS: 10, ExactRequiredOutputs: true}}},
		{Outcome: "NET_NEGATIVE", MeasuredBuilds: 1, Observations: []observation{{ControlWallMS: 10, BuildOptWallMS: 11, SignedDeltaMS: -1, ExactRequiredOutputs: true}}},
		{Outcome: "INCONCLUSIVE", Exclusions: []exclusion{{Reason: "NO_PAIR"}}},
	}
	summary := summarize(rows, 3)
	if summary.NetPositiveRows != 1 || summary.NetNegativeRows != 1 || summary.InconclusiveRows != 1 ||
		summary.MeasuredBuilds != 2 || summary.SelectedBuilds != 1 || summary.ExactComparableBuilds != 2 ||
		!summary.CompleteSignedMeasuredDeltas || !summary.AggregateDecisionDeferred {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestSummaryRejectsManipulatedDelta(t *testing.T) {
	rows := []resultRow{{Outcome: "NET_POSITIVE", MeasuredBuilds: 1, Observations: []observation{{ControlWallMS: 20, BuildOptWallMS: 10, SignedDeltaMS: 11, ExactRequiredOutputs: true}}}}
	if summarize(rows, 3).CompleteSignedMeasuredDeltas {
		t.Fatal("manipulated signed delta was accepted")
	}
}

func TestAssembleRowRejectsChangedFrozenSource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := assembleRow(root, repositorySpec{
		Key: "example", RepositoryID: "example/repository", SourceType: "QUALIFIED_LIFETIME_SUBJECT_V2",
		SourcePath: "source.json", SourceSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		Workflow: []string{"build"},
	})
	if err == nil || err.Error() != "declared source digest does not match" {
		t.Fatalf("changed source error = %v", err)
	}
}
