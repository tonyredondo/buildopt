package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFrozenSourceAndMutationBytes(t *testing.T) {
	inputs := sourceInputs()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	task, err := os.ReadFile(filepath.Join(repo, "jvm/patcher/src/spike/resources/reviewed-native/ForbiddenPatternsTask.java.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if digest(task) != inputs.TaskPreimageSHA256 {
		t.Fatal("approved task preimage drift")
	}
	input, err := os.ReadFile(filepath.Join(repo, "dev/installed-elasticsearch-runner/testdata/RecoveryState.java.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if digest(input) != inputs.OwnerInputSHA256 {
		t.Fatal("owner fixture input drift")
	}
	if digest(append(append([]byte{}, input...), []byte("// buildopt input invalidation fixture\n")...)) != inputs.BenignSHA256 {
		t.Fatal("benign mutation drift")
	}
	if digest(append(append([]byte{}, input...), []byte("//\tbuildopt input invalidation fixture\n")...)) != inputs.NegativeSHA256 {
		t.Fatal("tab mutation drift")
	}
}

func TestFrozenAllocationAndOrder(t *testing.T) {
	p := makeProtocol()
	if len(p.Rows) != 300 || p.RequiredStarts != 312 || p.ReserveStarts != 2 || p.MaximumStarts != 314 {
		t.Fatalf("allocation changed: rows=%d starts=%d+%d/%d", len(p.Rows), p.RequiredStarts, p.ReserveStarts, p.MaximumStarts)
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	full := 0
	var chronology []string
	v := map[int][]string{}
	for _, r := range p.Rows {
		if seen[r.ID] {
			t.Fatalf("duplicate %s", r.ID)
		}
		seen[r.ID] = true
		counts[r.Block] += 1 + r.NestedStarts
		if r.Workload == "workflow" {
			full++
		}
		if r.Block == "H" {
			chronology = append(chronology, r.Revision+":"+r.Arm)
		}
		if r.Block == "V" && r.Pair > 0 {
			v[r.Pair] = append(v[r.Pair], r.Arm)
		}
		if len(r.Arguments) == 0 {
			t.Fatalf("missing argv: %s", r.ID)
		}
	}
	want := map[string]int{"P": 3, "D": 2, "C": 12, "M": 24, "T": 16, "V": 27, "L": 40, "H": 20, "O-setup": 8, "O": 160}
	if !reflect.DeepEqual(counts, want) || full != 102 {
		t.Fatalf("counts=%v full=%d", counts, full)
	}
	orders := [][]string{{"N0", "N1", "W1"}, {"W1", "N1", "N0"}, {"N1", "W1", "N0"}, {"N0", "W1", "N1"}, {"W1", "N0", "N1"}, {"N1", "N0", "W1"}, {"N0", "N1", "W1"}, {"W1", "N1", "N0"}}
	for i, order := range orders {
		if !reflect.DeepEqual(v[i+1], order) {
			t.Fatalf("V%d=%v", i+1, v[i+1])
		}
	}
	var wantHistory []string
	for _, rev := range p.Revisions[1:] {
		for _, arm := range []string{"N0", "W1", "W1", "N0"} {
			wantHistory = append(wantHistory, rev+":"+arm)
		}
	}
	if !reflect.DeepEqual(chronology, wantHistory) {
		t.Fatal("chronological first/repeat order drift")
	}
}

func TestProtocolDriftRefuses(t *testing.T) {
	for _, mutate := range []func(*protocol){
		func(p *protocol) { p.Rows[0], p.Rows[1] = p.Rows[1], p.Rows[0] },
		func(p *protocol) { p.Rows[0].Arguments = append(p.Rows[0].Arguments, "--rerun-tasks") },
		func(p *protocol) { p.MaximumStarts++ },
		func(p *protocol) { p.Revisions[1] = p.Revisions[2] },
		func(p *protocol) { p.Rows = p.Rows[:len(p.Rows)-1] },
	} {
		p := makeProtocol()
		mutate(&p)
		if validateProtocol(p) == nil {
			t.Fatal("accepted changed frozen protocol")
		}
	}
}

func TestBudgetNeverPromotesMissingOrHistoricalEvidence(t *testing.T) {
	p := makeProtocol()
	b := budgetInput{ChargedSeconds: 4440, PreparationSeconds: 0, ManagementSeconds: 0, Estimates: []costEstimate{}}
	r, err := admit(p, b)
	if err != nil || r.Decision != "UNPROVED" || r.RemainingSeconds != 2760 || r.PublicExecutionAuthorized || len(r.UnsizedProfiles) == 0 {
		t.Fatalf("report=%+v err=%v", r, err)
	}
	for _, profile := range costProfiles(p) {
		b.Estimates = append(b.Estimates, costEstimate{Profile: profile, UpperBoundMs: 1, Provenance: "HISTORICAL"})
	}
	r, err = admit(p, b)
	if err != nil || r.Decision != "UNPROVED" {
		t.Fatalf("historical timing admitted: %+v %v", r, err)
	}
	for i := range b.Estimates {
		b.Estimates[i].Provenance = "OPERATOR_ESTIMATE"
	}
	r, err = admit(p, b)
	if err != nil || r.Decision != "FITS_ESTIMATE_ONLY" || r.PublicExecutionAuthorized {
		t.Fatalf("estimate did not remain separate from authority: %+v %v", r, err)
	}
	for i := range b.Estimates {
		b.Estimates[i].UpperBoundMs = 600000
	}
	r, err = admit(p, b)
	if err != nil || r.Decision != "DOES_NOT_FIT" {
		t.Fatalf("overspend admitted: %+v %v", r, err)
	}
	b.ChargedSeconds = 7200
	r, err = admit(p, b)
	if err != nil || r.Decision != "BUDGET_EXHAUSTED" {
		t.Fatalf("budget reset: %+v %v", r, err)
	}
}

func TestBudgetInvalidInputs(t *testing.T) {
	for _, b := range []budgetInput{
		{ChargedSeconds: 4439}, {ChargedSeconds: 4440, PreparationSeconds: -1},
		{ChargedSeconds: 4440, Estimates: []costEstimate{{Profile: "unknown", UpperBoundMs: 10, Provenance: "OPERATOR_ESTIMATE"}}},
		{ChargedSeconds: 4440, Estimates: []costEstimate{{Profile: "workflow:N0", UpperBoundMs: -1, Provenance: "OPERATOR_ESTIMATE"}}},
	} {
		if _, err := admit(makeProtocol(), b); err == nil {
			t.Fatalf("accepted invalid budget %+v", b)
		}
	}
}

func TestOwnerExtendedBudgetKeepsAccruedCostsAndRuntimeGates(t *testing.T) {
	b := budgetInput{ChargedSeconds: 6600, CeilingSeconds: 86400}
	r, err := admit(makeProtocol(), b)
	if err != nil || r.CeilingSeconds != 86400 || r.RemainingSeconds != 79800 || r.Decision != "UNPROVED" || r.PublicExecutionAuthorized {
		t.Fatalf("owner extension reset costs or bypassed runtime gates: %+v %v", r, err)
	}
	b.CeilingSeconds = 7200
	r, err = admit(makeProtocol(), b)
	if err != nil || r.Decision != "DOES_NOT_FIT" {
		t.Fatalf("historical envelope changed: %+v %v", r, err)
	}
}
