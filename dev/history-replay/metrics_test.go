//go:build linux && amd64

package main

import (
	"testing"
)

func metricHistory() (Manifest, []Slot) {
	m := Manifest{Phase: "CONFIRMATION", PrefixEnd: 20, ExecutionEnd: 100}
	slots := []Slot{}
	for i := 0; i <= 100; i++ {
		slots = append(slots, Slot{Ordinal: i, Class: comparable, NativeNS: 10e9, CandidateNS: 8e9, NativeActions: 1, CandidateActions: 1, Attempts: []string{}})
	}
	return m, slots
}

func TestEconomicsGoldenAndForgeryInputs(t *testing.T) {
	m, s := metricHistory()
	s[0].CandidateNS = 60e9 // +50 minus the other 20 prefix savings = 10s adoption.
	c := []Cost{{ID: "adopt", Class: "customer-machine", Purpose: "adoption", Replication: 1, Ordinal: 0, StartNS: 100, EndNS: 5e9 + 100, DurationNS: 5e9}, {ID: "maintain", Class: "customer-machine", Purpose: "inverse", Replication: 1, Ordinal: 30, StartNS: 200, EndNS: 3e9 + 200, DurationNS: 3e9}}
	e, err := calculate(m, 1, s, c)
	if err != nil {
		t.Fatal(err)
	}
	if e.AdoptionNS != 15e9 || e.MaintenanceNS != 3e9 || e.NetSavedNS != 142e9 || e.NativeNS != 800e9 || e.NetPerScheduledNS != 1.775e9 || e.NetFraction != 0.1775 || e.PaybackOrdinal != 28 || e.Gate != "G3_PASS" {
		t.Fatalf("wrong economics: %+v", e)
	}
	if e.NativeP95NS != 10e9 || e.CandidateP95NS != 8e9 || len(e.Intervals) != 2 || e.Intervals[0].LowerMeanNS <= 0 {
		t.Fatalf("wrong distribution: %+v", e)
	}
	for _, mutation := range []func([]Slot) []Slot{func(s []Slot) []Slot { return s[:100] }, func(s []Slot) []Slot { s[4].Ordinal = 3; return s }, func(s []Slot) []Slot { s[40].CandidateNS = -1; return s }} {
		x := append([]Slot{}, s...)
		if _, err := calculate(m, 1, mutation(x), c); err == nil {
			t.Fatal("accepted invalid raw metric input")
		}
	}
	if _, err := calculate(m, 1, s, append(c, c[0])); err == nil {
		t.Fatal("double charge accepted")
	}
}

func TestNoActionIncompleteAndNativeFailure(t *testing.T) {
	m, s := metricHistory()
	for i := range s {
		s[i].NativeActions = 0
		s[i].CandidateActions = 0
		s[i].CandidateNS = s[i].NativeNS + 1e6
	}
	e, err := calculate(m, 1, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.NoActionSlots != 80 || e.NoActionAttributedSavingNS != 0 || e.NoActionDeltaNS != -80e6 || e.Gate == "G3_PASS" {
		t.Fatalf("no action invented mechanism value: %+v", e)
	}
	m, s = metricHistory()
	s[21].Class = nativeFailure
	e, err = calculate(m, 1, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.Comparable != 79 || e.UnpairedCandidateNS != 8e9 || e.NetSavedNS != 150e9 {
		t.Fatalf("failure earned saving: %+v", e)
	}
	s[100] = Slot{Ordinal: 100, Class: notRunLimit, Reason: "allocation", Attempts: []string{}}
	e, err = calculate(m, 1, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.Complete || e.Gate == "G3_PASS" || e.PaybackOrdinal != -1 {
		t.Fatal("partial horizon passed")
	}
}

func TestPaybackMustSurviveAndNegativeDoesNotStop(t *testing.T) {
	m, s := metricHistory()
	for i := 21; i <= 40; i++ {
		s[i].CandidateNS = 12e9
	}
	e, err := calculate(m, 1, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.Scheduled != 80 || len(e.CurveNS) != 80 || e.PaybackOrdinal != 60 || e.NetSavedNS != 80e9 {
		t.Fatalf("early-negative horizon shortened: %+v", e)
	}
	m, s = metricHistory()
	s[22].CandidateNS = 20e9
	e, err = calculate(m, 1, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.TemporaryCrossings) != 1 || e.TemporaryCrossings[0] != 21 || e.PaybackOrdinal != 26 {
		t.Fatalf("temporary crossing credited: %+v", e)
	}
}

func TestQuantileAndBootstrapDeterministic(t *testing.T) {
	if nearestRank([]int64{5, 1, 4, 2, 3}, 50) != 3 || nearestRank([]int64{5, 1, 4, 2, 3}, 95) != 5 {
		t.Fatal("not nearest rank")
	}
	s := make([]int64, 80)
	for i := range s {
		s[i] = 10
	}
	for _, block := range []int{5, 10} {
		b, err := bootstrap(s, 80, block)
		if err != nil {
			t.Fatal(err)
		}
		if b.LowerMeanNS != 9 || b.UpperMeanNS != 9 {
			t.Fatalf("adoption not subtracted exactly once: %+v", b)
		}
	}
	for i := range s {
		s[i] = int64(i - 35)
	}
	a, _ := bootstrap(s, 100, 5)
	b, _ := bootstrap(s, 100, 5)
	if a != b {
		t.Fatal("unfrozen random stream")
	}
}
