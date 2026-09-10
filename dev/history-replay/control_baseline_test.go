//go:build linux && amd64 && replay_integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControlBaselineChronologicalReplay(t *testing.T) {
	m := runnableFixture(t, 2, "success")
	current := filepath.Join(filepath.Dir(m.RunRoot), "current.txt")
	mustWrite(t, current, []byte("installed control implementation\n"))
	m.ControlBaseline = &Patch{Identity: "installed-control", Prerequisites: map[string]string{}, Files: []FilePatch{
		{Path: "candidate.txt", After: bound(t, current), AfterMode: 0644},
	}}
	m.Replications = 2
	r := runFixture(t, m)
	// Exercise checkpoint recovery while each arm owns a different postimage
	// of the same source path, then independent fresh reverse-order replication.
	if err := r.execute(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	next, rep, ordinal, err := resumeRunner(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err = next.execute(rep, ordinal, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil || result.WorkflowStarts != 8 || result.Decision != "FIXTURE_VERIFIED" {
		t.Fatalf("incomplete current/revised replay: %+v %v", result, err)
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
	for _, state := range next.states {
		want := m.Candidate.Files[0].After.SHA256
		if state.Arm == "N" {
			want = m.ControlBaseline.Files[0].After.SHA256
		}
		found := false
		for _, entry := range state.Entries {
			if entry.Path == "repo/candidate.txt" {
				found = entry.SHA256 == want
			}
		}
		if !found {
			t.Fatalf("wrong installed source in %s", state.Arm)
		}
	}
	// The offline checker must reject a changed control definition, even when
	// the candidate and claimed timing totals are otherwise untouched.
	changed := m
	bad := *m.ControlBaseline
	bad.Files = append([]FilePatch{}, bad.Files...)
	bad.Files[0].After.SHA256 = strings.Repeat("a", 64)
	changed.ControlBaseline = &bad
	for _, state := range next.states {
		if state.Arm == "N" && checkRecordedSource(changed, state, map[string]string{}) == nil {
			t.Fatal("accepted forged control source")
		}
	}
	mustWrite(t, current, []byte("drifted installed source\n"))
	if checkInputs(m) == nil {
		t.Fatal("accepted control postimage drift")
	}
}

func TestControlBaselineContract(t *testing.T) {
	m := fixtureManifest(t, 2)
	raw := jsonBytes(m)
	if strings.Contains(string(raw), "controlBaseline") {
		t.Fatal("omission changed historical manifest serialization")
	}
	var decoded Manifest
	if err := decodeStrict(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"null", "{}"} {
		invalid := strings.Replace(string(raw), `"baseline":`, `"controlBaseline": `+value+`, "baseline":`, 1)
		if decodeStrict([]byte(invalid), &decoded) == nil {
			t.Fatal("accepted incomplete control baseline", value)
		}
	}
	m.ControlBaseline = &m.Baseline
	m.Phase = "ENGINEERING"
	if err := validateManifest(m); err == nil || !strings.Contains(err.Error(), "qualification-only") {
		t.Fatalf("control override escaped qualification scope: %v", err)
	}
	if _, err := os.Stat(m.RunRoot); !os.IsNotExist(err) {
		t.Fatal("contract validation created run state")
	}
}
