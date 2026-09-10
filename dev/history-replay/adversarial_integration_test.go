//go:build linux && amd64 && replay_integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTypedFailuresAndNativeFallback(t *testing.T) {
	for _, test := range []struct {
		mode, class string
		starts      int
	}{{"mismatch", candidateFailure, 2}, {"native-failure", nativeFailure, 6}, {"unavailable", dependencyUnavailable, 6}, {"no-action", comparable, 6}, {"fallback", nativeRetained, 6}} {
		t.Run(test.mode, func(t *testing.T) {
			m := runnableFixture(t, 3, test.mode)
			if test.mode == "fallback" {
				m.Candidate.Prerequisites["source.txt"] = digest([]byte("revision 0\n"))
			}
			r := runFixture(t, m)
			err := r.execute(1, 0, -1)
			if test.class == candidateFailure {
				if err == nil || !strings.Contains(err.Error(), candidateFailure) {
					t.Fatalf("mismatch did not stop: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			result, err := writeResult(m.RunRoot)
			if err != nil {
				t.Fatal(err)
			}
			ordinal := 1
			if test.class == candidateFailure {
				ordinal = 0
			}
			if result.WorkflowStarts != test.starts || result.Slots[0][ordinal].Class != test.class {
				t.Fatalf("wrong failure/fallback outcome: %+v", result)
			}
			if test.class == candidateFailure {
				if result.Slots[0][1].Class != notRunDependency || result.Decision != "INCOMPLETE_EVIDENCE" {
					t.Fatal("unsafe continuation or invented positive")
				}
				if _, _, _, err = resumeRunner(m.RunRoot); err == nil {
					t.Fatal("safety failure admitted as infrastructure retry")
				}
			}
			if test.mode == "no-action" && (result.Replications[0].NoActionSlots != 2 || result.Replications[0].NoActionAttributedSavingNS != 0) {
				t.Fatal("no-action timing noise attributed to mechanism")
			}
			if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDriftStopsBeforeNextChild(t *testing.T) {
	for _, kind := range []string{"source", "patch", "runtime", "package", "future-state", "shared-state"} {
		t.Run(kind, func(t *testing.T) {
			m := runnableFixture(t, 2, "success")
			private := filepath.Join(filepath.Dir(m.RunRoot), kind+"-input")
			mustWrite(t, private, []byte("frozen\n"))
			if kind == "runtime" {
				m.Runtime = append(m.Runtime, bound(t, private))
			}
			if kind == "package" {
				m.Package = append(m.Package, bound(t, private))
			}
			r := runFixture(t, m)
			r.fault = func(point, id string) {
				if point != "pair-checkpoint" || id != "r1-000" {
					return
				}
				switch kind {
				case "source":
					mustWrite(t, filepath.Join(m.RunRoot, "r1/N/repo/source.txt"), []byte("unrecorded source\n"))
				case "patch":
					mustWrite(t, m.Candidate.Files[0].After.Path, []byte("unfrozen patch\n"))
				case "runtime", "package":
					mustWrite(t, private, []byte("changed\n"))
				case "future-state":
					mustWrite(t, filepath.Join(m.RunRoot, "r1/I/gradle/caches/build-cache-1/future"), []byte("future cache\n"))
				case "shared-state":
					if err := os.Symlink(filepath.Join(m.RunRoot, "r1/N/gradle"), filepath.Join(m.RunRoot, "r1/I/borrowed")); err != nil {
						t.Fatal(err)
					}
				}
			}
			err := r.execute(1, 0, -1)
			if err == nil {
				t.Fatal("drift accepted")
			}
			t.Logf("expected pre-child refusal: %s", err)
			files, err := filepath.Glob(filepath.Join(m.RunRoot, "attempts", "*", "native-pid.json"))
			if err != nil || len(files) != 2 {
				t.Fatalf("started child after drift: %v %v", files, err)
			}
		})
	}
}

func TestAllocationStopsAndDetachedChild(t *testing.T) {
	t.Run("workflow-limit", func(t *testing.T) {
		m := runnableFixture(t, 3, "success")
		m.Limits.MaxWorkflowStarts = 2
		r := runFixture(t, m)
		if err := r.execute(1, 0, -1); err == nil || !strings.Contains(err.Error(), notRunLimit) {
			t.Fatalf("limit not enforced: %v", err)
		}
		result, err := writeResult(m.RunRoot)
		if err != nil {
			t.Fatal(err)
		}
		if result.WorkflowStarts != 2 || result.Slots[0][1].Class != notRunLimit {
			t.Fatal("bad limit outcome")
		}
	})
	t.Run("disk-before-start", func(t *testing.T) {
		m := runnableFixture(t, 2, "success")
		m.Limits.MinimumFreeBytes = ^uint64(0)
		r := runFixture(t, m)
		if err := r.execute(1, 0, -1); err == nil {
			t.Fatal("free-space guard bypassed")
		}
		files, _ := filepath.Glob(filepath.Join(m.RunRoot, "attempts", "*", "native-pid.json"))
		if len(files) != 0 {
			t.Fatal("child started without disk reserve")
		}
	})
	t.Run("detached-and-timeout", func(t *testing.T) {
		m := runnableFixture(t, 2, "detached")
		m.Limits.MaxRequestNS = int64(600 * time.Millisecond)
		r := runFixture(t, m)
		if err := r.execute(1, 0, -1); err == nil {
			t.Fatal("detached timed-out fixture accepted")
		}
		var native ProcessReceipt
		if err := readJSON(filepath.Join(m.RunRoot, "attempts/r1-000-g0-N/native-finish.json"), &native); err != nil {
			t.Fatal(err)
		}
		var child ProcessIdentity
		if err := readJSON(filepath.Join(m.RunRoot, "attempts/r1-000-g0-N/detached.json"), &child); err != nil {
			t.Fatal(err)
		}
		if native.Outcome != "CANCELLED" || !strings.HasSuffix(child.Cgroup, "/"+native.Unit) {
			t.Fatalf("missing real detached containment: %+v", native)
		}
		if sameProcess(child) {
			t.Fatal("owned detached process survived timeout")
		}
		if len(native.Observed) < 2 {
			t.Fatal("process tree was not actually observed")
		}
		t.Logf("detached PID %d and service %s terminated", child.PID, native.Unit)
	})
}

func TestRawEvidenceTampering(t *testing.T) {
	m := runnableFixture(t, 2, "success")
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	if _, err := writeResult(m.RunRoot); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(m.RunRoot, "attempts/r1-000-g0-N")
	for _, test := range []struct {
		name, path string
		mutate     func([]byte) []byte
	}{
		{"native-duration", filepath.Join(dir, "native-finish.json"), func(raw []byte) []byte {
			var x ProcessReceipt
			if err := decodeStrict(raw, &x); err != nil {
				t.Fatal(err)
			}
			x.End.NS = x.Start.NS - 1
			return jsonBytes(x)
		}},
		{"output-bytes", filepath.Join(dir, "outputs/000000"), func(raw []byte) []byte { return append(raw, '!') }},
		{"hidden-preparation", filepath.Join(m.RunRoot, "costs/r1-000-g0-I-request.json"), func(raw []byte) []byte {
			var x Cost
			if err := decodeStrict(raw, &x); err != nil {
				t.Fatal(err)
			}
			x.Class = "research"
			return jsonBytes(x)
		}},
		{"normalization", filepath.Join(dir, "capture.json"), func(raw []byte) []byte {
			var x Capture
			if err := decodeStrict(raw, &x); err != nil {
				t.Fatal(err)
			}
			x.Outputs[0].Transform = "drop-differences"
			return jsonBytes(x)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			mustWrite(t, test.path, test.mutate(raw))
			err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json"))
			mustWrite(t, test.path, raw)
			if err == nil {
				t.Fatal("raw forgery accepted")
			}
			t.Logf("expected raw refusal: %s", err)
		})
	}
	t.Run("missing-maintenance", func(t *testing.T) {
		names := []string{".start.json", ".work-start.json", ".end.json", ".json"}
		base := filepath.Join(m.RunRoot, "costs/r1-001-I-inverse")
		for _, suffix := range names {
			if err := os.Rename(base+suffix, base+suffix+".quarantined"); err != nil {
				t.Fatal(err)
			}
		}
		err := checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json"))
		for _, suffix := range names {
			if e := os.Rename(base+suffix+".quarantined", base+suffix); e != nil {
				t.Fatal(e)
			}
		}
		if err == nil || !strings.Contains(err.Error(), "required phase") {
			t.Fatalf("hidden inverse did not fail its semantic contract: %v", err)
		}
		t.Logf("expected semantic accounting refusal: %s", err)
	})
	t.Run("rebound-duration-forgery", func(t *testing.T) {
		endPath := filepath.Join(dir, "end.json")
		pairPath := filepath.Join(m.RunRoot, "pairs/r1-000.json")
		originalEnd, err := os.ReadFile(endPath)
		if err != nil {
			t.Fatal(err)
		}
		originalPair, err := os.ReadFile(pairPath)
		if err != nil {
			t.Fatal(err)
		}
		var end AttemptEnd
		if err = decodeStrict(originalEnd, &end); err != nil {
			t.Fatal(err)
		}
		end.DurationNS++
		mustWrite(t, endPath, jsonBytes(end))
		var pair PairRecord
		if err = decodeStrict(originalPair, &pair); err != nil {
			t.Fatal(err)
		}
		pair.Attempts[0] = bound(t, endPath)
		mustWrite(t, pairPath, jsonBytes(pair))
		err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json"))
		mustWrite(t, endPath, originalEnd)
		mustWrite(t, pairPath, originalPair)
		if err == nil || !strings.Contains(err.Error(), "forged request duration") {
			t.Fatalf("rebound records bypassed independent clock arithmetic: %v", err)
		}
		t.Logf("expected semantic duration refusal: %s", err)
	})
	t.Run("hidden-retry", func(t *testing.T) {
		var a AttemptStart
		if err := readJSON(filepath.Join(dir, "start.json"), &a); err != nil {
			t.Fatal(err)
		}
		a.ID = "r1-001-g9-N"
		a.Ordinal = 1
		a.Generation = 9
		a.At.NS = stamp().NS
		path := filepath.Join(m.RunRoot, "attempts", a.ID, "start.json")
		mustWrite(t, path, jsonBytes(a))
		if err := checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err == nil {
			t.Fatal("uncharged retry accepted")
		} else {
			t.Logf("expected hidden retry refusal: %s", err)
		}
	})
}

func TestFullHorizonDoesNotStopAtNegativeValue(t *testing.T) {
	m := runnableFixture(t, 5, "early-negative")
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 10 || len(result.Slots[0]) != 5 {
		t.Fatal("value shortened the fixed horizon")
	}
	early, later := result.Slots[0][1], result.Slots[0][4]
	if early.NativeNS >= early.CandidateNS || later.NativeNS <= later.CandidateNS {
		t.Fatalf("fixture did not exercise both real timing signs: early %+v later %+v", early, later)
	}
	t.Logf("all ordinals retained: early delta %s, final delta %s", time.Duration(early.NativeNS-early.CandidateNS), time.Duration(later.NativeNS-later.CandidateNS))
}
