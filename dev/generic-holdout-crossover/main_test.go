package main

import (
	"strings"
	"testing"
)

func TestCalculateResultRequiresEveryReciprocalBlock(t *testing.T) {
	spec := specification{}
	spec.Qualification.MinimumMeanSavedMS = 500
	spec.Qualification.MinimumReductionRatio = 0.02
	spec.Qualification.MinimumPositiveBlocks = requiredBlocks
	blocks := make([]block, requiredBlocks)
	for index := range blocks {
		blocks[index] = block{ControlMeanMS: 10_000, CandidateMeanMS: 9_000, SavedMS: 1_000}
	}
	got := calculateResult(blocks, spec, true, true, true)
	if !got.Qualified || got.Decision != "REVIEW_STRUCTURAL_PROFILE" || got.PositiveBlocks != requiredBlocks || got.Interval95SavedMS[0] != 1_000 {
		t.Fatalf("qualified crossover = %+v", got)
	}
	blocks[7].CandidateMeanMS = 10_100
	blocks[7].SavedMS = -100
	got = calculateResult(blocks, spec, true, true, true)
	if got.Qualified || got.Decision != "RETAIN_NATIVE_GRADLE" || got.PositiveBlocks != requiredBlocks-1 {
		t.Fatalf("negative reciprocal block qualified = %+v", got)
	}
}

func TestCompareTasksExplainsPathAndOutcomeDrift(t *testing.T) {
	reference := taskOutcomes{Tasks: []task{
		{Path: ":alpha", Outcome: "EXECUTED"},
		{Path: ":beta", Outcome: "FROM_CACHE"},
	}}
	current := taskOutcomes{Tasks: []task{
		{Path: ":alpha", Outcome: "UP_TO_DATE"},
		{Path: ":gamma", Outcome: "EXECUTED"},
	}}
	difference := compareTasks(reference, current)
	if strings.Join(difference.AddedPaths, ",") != ":gamma" ||
		strings.Join(difference.RemovedPaths, ",") != ":beta" ||
		strings.Join(difference.ChangedOutcomes, ",") != ":alpha:EXECUTED->UP_TO_DATE" {
		t.Fatalf("task difference = %+v", difference)
	}
}
