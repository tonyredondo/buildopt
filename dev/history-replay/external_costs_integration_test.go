//go:build linux && amd64 && replay_integration

package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestExternalPreparationIsChargedExactlyOnce(t *testing.T) {
	m := runnableFixture(t, 2, "success")
	root := filepath.Dir(m.RunRoot)
	evidence := filepath.Join(root, "external-workflow.txt")
	mustWrite(t, evidence, []byte("qualification-only measured customer setup\n"))
	start := stamp()
	time.Sleep(2 * time.Millisecond)
	end := stamp()
	record := ExternalCostRecord{"buildopt.history-replay/external-cost/v1", "setup", "customer-machine", "adoption", 1, 0, start, end, []Binding{bound(t, evidence)}, "Controlled fixture setup, measured outside replay; no public product cost inference"}
	path := filepath.Join(root, "external-adoption.json")
	mustWrite(t, path, jsonBytes(record))
	m.ExternalCosts = []Binding{bound(t, path)}
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	prefix := result.Slots[0][0]
	expected := end.NS - start.NS + max(int64(0), prefix.CandidateNS-prefix.NativeNS)
	if result.Replications[0].AdoptionNS != expected {
		t.Fatalf("external setup not charged exactly once: got %d want %d", result.Replications[0].AdoptionNS, expected)
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
	bad := m
	bad.ExternalCosts = append(bad.ExternalCosts, bad.ExternalCosts[0])
	if _, err = externalCosts(bad); err == nil {
		t.Fatal("duplicate external setup accepted")
	}
}
