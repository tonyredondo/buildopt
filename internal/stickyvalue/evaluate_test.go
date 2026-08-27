package stickyvalue

import (
	"math"
	"reflect"
	"testing"
)

func zeroCosts() Costs {
	zero := int64(0)
	return Costs{&zero, &zero, &zero, &zero, &zero, &zero, &zero, &zero, &zero}
}

func TestEvaluateQualifiesExactProfitablePairs(t *testing.T) {
	pairs := []Pair{
		{PairID: "p1", Order: "CANDIDATE_FIRST", NativeWallNs: 200, CandidateWallNs: 100, OutputsEquivalent: true},
		{PairID: "p2", Order: "NATIVE_FIRST", NativeWallNs: 220, CandidateWallNs: 110, OutputsEquivalent: true},
		{PairID: "p3", Order: "CANDIDATE_FIRST", NativeWallNs: 240, CandidateWallNs: 120, OutputsEquivalent: true},
		{PairID: "p4", Order: "NATIVE_FIRST", NativeWallNs: 260, CandidateWallNs: 130, OutputsEquivalent: true},
	}
	got, err := Evaluate(pairs, zeroCosts())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Qualified || got.MeanEffectNs != 115 || got.GrossSavingNs != 460 || got.NetSavingNs != 460 || got.PositivePairCount != 4 || got.LowerBoundNs <= 0 || got.CandidateP95Ns > got.NativeP95Ns {
		t.Fatalf("unexpected evaluation: %+v", got)
	}
	wantEffects := []int64{100, 110, 120, 130}
	if !reflect.DeepEqual(got.PairEffectsNs, wantEffects) {
		t.Fatalf("effects = %v, want %v", got.PairEffectsNs, wantEffects)
	}
}

func TestEvaluateRejectsUnavailableCostAndInexactPair(t *testing.T) {
	costs := zeroCosts()
	costs.StateNs = nil
	got, err := Evaluate([]Pair{{PairID: "p1", Order: "NATIVE_FIRST", NativeWallNs: 200, CandidateWallNs: 100}}, costs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Qualified || !reflect.DeepEqual(got.Reasons, []string{"PAIR_NOT_COMPARABLE", "COST_UNAVAILABLE"}) {
		t.Fatalf("unexpected rejection: %+v", got)
	}
}

func TestEvaluateRejectsOverflowAndDuplicates(t *testing.T) {
	if _, err := Evaluate([]Pair{{PairID: "p", Order: "NATIVE_FIRST", NativeWallNs: math.MaxInt64, CandidateWallNs: 0, OutputsEquivalent: true}, {PairID: "q", Order: "CANDIDATE_FIRST", NativeWallNs: math.MaxInt64, CandidateWallNs: 0, OutputsEquivalent: true}}, zeroCosts()); err == nil {
		t.Fatal("overflow accepted")
	}
	if _, err := Evaluate([]Pair{{PairID: "p", Order: "NATIVE_FIRST", OutputsEquivalent: true}, {PairID: "p", Order: "CANDIDATE_FIRST", OutputsEquivalent: true}}, zeroCosts()); err == nil {
		t.Fatal("duplicate pair accepted")
	}
}
