package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func correctnessCaptureFixture(t *testing.T) (correctnessRuntime, correctnessStep) {
	t.Helper()
	binding, _ := compareFixture(t, firstManifest, "EXECUTED")
	root := filepath.Dir(filepath.Dir(filepath.Dir(binding.Path)))
	dir := filepath.Join(root, "attempts/C001")
	if err := os.Rename(filepath.Dir(binding.Path), dir); err != nil {
		t.Fatal(err)
	}
	f := correctnessRuntime{Root: root, Runner: fileBinding{Path: "/qualified-runner", SHA256: strings.Repeat("a", 64)}, BootID: "boot", StartNS: 1, DeadlineNS: 30000000000}
	step := correctnessStepForTest(t, "C001")
	request, err := correctnessRequestFor(f, step)
	if err != nil {
		t.Fatal(err)
	}
	request.BootID = f.BootID
	request.ReservedNS = 1
	p := processRecord{Started: true, PID: 10, StartNS: 2, EndNS: 10000000002, Outcome: "SUCCESS"}
	ownership := map[string]string{"unit": "owned.service", "cgroup": "0::/user/owned.service", "invocationId": "owned-invocation"}
	inputs := []entry{{Path: sourceInputs().TaskPath, Type: "file", Mode: 0644, SHA256: sourceInputs().TaskPreimageSHA256}, {Path: sourceInputs().OwnerInputPath, Type: "file", Mode: 0644, SHA256: step.InputAfter}, {Path: "gradlew", Type: "file", Mode: 0755, SHA256: "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"}}
	for name, value := range map[string]any{"native-request.json": request, "native-result.json": nativeResult{RowID: step.Row.ID, Process: p, InputState: "VERIFIED", Cgroup: ownership["cgroup"], InvocationID: ownership["invocationId"]}, "process.json": p, "ownership.json": ownership, "closed.json": correctnessClosed{ownership["unit"], ownership["cgroup"], true, p.EndNS + 1}, "source-before.json": inputs, "source-after.json": inputs} {
		raw, _ := json.Marshal(value)
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	transitions := filepath.Join(root, "transitions/C001")
	if err := os.MkdirAll(transitions, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(transitions, "after.json"), correctnessTransition{RowID: step.Row.ID, Before: inputs[1:2], After: inputs[1:2]}); err != nil {
		t.Fatal(err)
	}
	resealCorrectnessDiagnosticFixture(t, dir)
	return f, step
}

func resealCorrectnessDiagnosticFixture(t *testing.T, dir string) {
	t.Helper()
	bindings := map[string]string{}
	for _, name := range nativeDiagnosticArtifacts {
		sha, err := hashFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		bindings[name] = sha
	}
	raw, _ := json.Marshal(bindings)
	if err := os.WriteFile(filepath.Join(dir, "diagnostic-complete.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCorrectnessCaptureReconstructsOutcomeAndRejectsTampering(t *testing.T) {
	f, step := correctnessCaptureFixture(t)
	if result, err := inspectCorrectnessRow(f, step); err != nil || result.TaskOutcome != "EXECUTED" {
		t.Fatalf("valid capture: %+v %v", result, err)
	}
	for _, kind := range []string{"native-failure", "request", "source", "transition", "closure", "raw-output"} {
		t.Run(kind, func(t *testing.T) {
			f, step := correctnessCaptureFixture(t)
			dir := filepath.Join(f.Root, "attempts/C001")
			var path string
			var value any
			switch kind {
			case "native-failure":
				var r nativeResult
				path = filepath.Join(dir, "native-result.json")
				readJSON(path, &r)
				r.Process.ExitCode = 1
				r.Process.Outcome = "FAILURE"
				value = r
			case "request":
				var r nativeRequest
				path = filepath.Join(dir, "native-request.json")
				readJSON(path, &r)
				r.Arguments = append(r.Arguments, "--rerun-tasks")
				value = r
			case "source":
				var r []entry
				path = filepath.Join(dir, "source-before.json")
				readJSON(path, &r)
				r[0].SHA256 = strings.Repeat("b", 64)
				value = r
			case "transition":
				var r correctnessTransition
				path = filepath.Join(f.Root, "transitions/C001/after.json")
				readJSON(path, &r)
				r.After[0].SHA256 = strings.Repeat("b", 64)
				value = r
			case "closure":
				path = filepath.Join(dir, "closed.json")
				value = correctnessClosed{"owned.service", "0::/user/owned.service", false, 10000000003}
			case "raw-output":
				path = filepath.Join(dir, "operations-log.txt")
				value = map[string]any{}
			}
			raw, _ := json.Marshal(value)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			// Even a newly hashed summary cannot hide altered raw semantics.
			resealCorrectnessDiagnosticFixture(t, dir)
			if _, err := inspectCorrectnessRow(f, step); err == nil {
				t.Fatal("tampered capture accepted")
			}
		})
	}
}

func TestCorrectnessSequenceStopsOnIncompleteOrReorderedCapture(t *testing.T) {
	plan := makeCorrectnessPlan()
	good := []correctnessRowReceipt{}
	for _, step := range plan.Steps[:12] {
		if err := validateCorrectnessSequence(good, step.Row.ID); err != nil {
			t.Fatal(err)
		}
		good = append(good, correctnessRowReceipt{RowID: step.Row.ID, Complete: fileBinding{Path: "/campaign/attempts/" + step.Row.ID + "/row-complete.json", SHA256: strings.Repeat("a", 64)}})
	}
	for _, bad := range [][]correctnessRowReceipt{good[:1], append(append([]correctnessRowReceipt{}, good[:2]...), good[1]), {{RowID: "C001"}}, {{RowID: "C002", Complete: good[0].Complete}}} {
		if err := validateCorrectnessSequence(bad, "C003"); err == nil {
			t.Fatal("invalid prior capture accepted", bad)
		}
	}
	if err := validateCorrectnessSequence(good, "V001"); err == nil {
		t.Fatal("value request accepted")
	}
}

func TestCorrectnessFailureSignatureRequiresExactOwnerAndLine(t *testing.T) {
	expected := "Found invalid patterns:\n  - tab on line 1178 of server/src/main/java/org/elasticsearch/indices/recovery/RecoveryState.java\n"
	if _, err := correctnessFailureSignature(expected); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"BUILD FAILED", strings.ReplaceAll(expected, "RecoveryState.java", "Other.java"), strings.ReplaceAll(expected, "1178", "1"), expected + expected} {
		if _, err := correctnessFailureSignature(s); err == nil {
			t.Fatal("invalid failure accepted", s)
		}
	}
}
