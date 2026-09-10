//go:build linux && amd64 && replay_integration

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == "observer" && os.Getenv("BUILDOPT_OBSERVER_TEST_FAULT") != "" {
		if err := runObserverFaultHelper(os.Args[2], os.Args[3], os.Getenv("BUILDOPT_OBSERVER_TEST_FAULT")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) >= 4 && os.Args[1] == "fixture-projector" {
		raw, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		lines := strings.Split(string(raw), "\n")
		if len(lines) != 3 || lines[0] != "root="+os.Args[3] || !strings.HasPrefix(lines[1], "value=") || lines[2] != "" {
			fmt.Fprintln(os.Stderr, "invalid fixture projection input")
			os.Exit(1)
		}
		value := lines[1] + "\n"
		if len(os.Args) == 5 && os.Args[4] == "constant" {
			value = "value=expected\n"
		}
		fmt.Println(digest([]byte(value)))
		os.Exit(0)
	}
	if len(os.Args) > 1 && (os.Args[1] == "worker" || strings.HasPrefix(os.Args[1], "observer") || os.Args[1] == "validate" || os.Args[1] == "run" || os.Args[1] == "check" || os.Args[1] == "resume") {
		if err := command(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) == 5 && os.Args[1] == "fixture-driver" {
		r, err := newRunner(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		r.fault = func(point, id string) {
			if point == os.Args[3] && id == os.Args[4] {
				if point == "before-receipt" {
					_ = os.WriteFile(filepath.Join(r.manifest.RunRoot, "attempts", id, "end.json"), []byte(`{"schema":`), 0600)
				}
				os.Exit(86)
			}
		}
		if err = r.execute(1, 0, -1); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestCheckpointResumeAndCrashRecovery(t *testing.T) {
	t.Run("verified-pair", func(t *testing.T) {
		m := runnableFixture(t, 3, "success")
		r := runFixture(t, m)
		if err := r.execute(1, 0, 0); err != nil {
			t.Fatal(err)
		}
		next, rep, ordinal, err := resumeRunner(m.RunRoot)
		if err != nil {
			t.Fatal(err)
		}
		if ordinal != 1 || rep != 1 {
			t.Fatal("wrong continuation")
		}
		if err = next.execute(rep, ordinal, -1); err != nil {
			t.Fatal(err)
		}
		result, err := writeResult(m.RunRoot)
		if err != nil {
			t.Fatal(err)
		}
		if result.WorkflowStarts != 6 || result.Decision != "FIXTURE_VERIFIED" {
			t.Fatal("resume duplicated or lost work")
		}
	})
	for _, point := range []string{"after-receipt", "before-receipt"} {
		t.Run(point, func(t *testing.T) {
			m := runnableFixture(t, 3, "success")
			path := filepath.Join(filepath.Dir(m.RunRoot), "fixture-manifest.json")
			mustWrite(t, path, jsonBytes(m))
			c := exec.Command(m.Executable.Path, "fixture-driver", path, point, "r1-001-g0-I")
			b, err := c.CombinedOutput()
			if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 86 {
				t.Fatalf("expected actual driver interruption: %v %s", err, b)
			}
			next, rep, ordinal, err := resumeRunner(m.RunRoot)
			if err != nil {
				t.Fatal(err)
			}
			if ordinal != 1 || next.retries != 1 {
				t.Fatal("lost recovery identity")
			}
			if err = next.execute(rep, ordinal, -1); err != nil {
				t.Fatal(err)
			}
			result, err := writeResult(m.RunRoot)
			if err != nil {
				t.Fatal(err)
			}
			if result.WorkflowStarts != 7 || result.Decision != "FIXTURE_VERIFIED" || result.Slots[0][1].ExtraCandidateNS <= 0 || len(result.Slots[0][1].Attempts) != 2 {
				t.Fatalf("interrupted candidate received duplicate/free credit: %+v", result)
			}
		})
	}
	t.Run("sealed-pair-without-checkpoint", func(t *testing.T) {
		m := runnableFixture(t, 2, "success")
		path := filepath.Join(filepath.Dir(m.RunRoot), "fixture-manifest.json")
		mustWrite(t, path, jsonBytes(m))
		c := exec.Command(m.Executable.Path, "fixture-driver", path, "pair-sealed", "r1-001")
		b, err := c.CombinedOutput()
		if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 86 {
			t.Fatalf("expected actual interruption: %v %s", err, b)
		}
		if _, _, _, err = resumeRunner(m.RunRoot); err == nil || !strings.Contains(err.Error(), "sealed pair") {
			t.Fatalf("sealed pair was replayed: %v", err)
		}
		result, err := writeResult(m.RunRoot)
		if err != nil {
			t.Fatal(err)
		}
		if result.WorkflowStarts != 4 || result.Decision != "INCOMPLETE_EVIDENCE" {
			t.Fatal("uncheckpointed pair silently resumed")
		}
	})
	t.Run("warm-session-loss", func(t *testing.T) {
		m := runnableFixture(t, 2, "success")
		m.DaemonPolicy = "REPLICATION"
		m.Limits.RetryPairs = 0
		r := runFixture(t, m)
		if err := r.execute(1, 0, 0); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := resumeRunner(m.RunRoot); err == nil || !strings.Contains(err.Error(), "warm daemon") {
			t.Fatalf("warm memory fabricated on resume: %v", err)
		}
	})
}

func TestFixtureChild(t *testing.T) {
	if os.Getenv("BUILDOPT_FIXTURE_CHILD") != "1" {
		return
	}
	repo, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(repo, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Dir(os.Getenv("BUILDOPT_REPLAY_CAPTURE"))
	mode := os.Args[len(os.Args)-1]
	if mode == "observed" {
		time.Sleep(600 * time.Millisecond)
	}
	output := source
	exit := 0
	diagnostic := "success\n"
	_, candidate := os.Stat(filepath.Join(repo, "candidate.txt"))
	if mode == "early-negative" {
		if candidate == nil && strings.Contains(string(source), "revision 1") || candidate != nil && !strings.Contains(string(source), "revision 0") && !strings.Contains(string(source), "revision 1") {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if mode == "detached" || mode == "detached-after-anchor" && strings.Contains(string(source), "revision 1") {
		child := exec.Command("/usr/bin/sleep", "30")
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err = child.Start(); err != nil {
			t.Fatal(err)
		}
		identity, err := processIdentity(child.Process.Pid)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(evidence, "detached.json"), jsonBytes(identity))
		time.Sleep(30 * time.Second)
	}
	if mode == "mismatch" && candidate == nil {
		output = []byte("incorrect candidate output\n")
	}
	if mode == "native-failure" && strings.Contains(string(source), "revision 1") {
		exit = 1
		diagnostic = "fixture-rule: source.txt:1: rejected\n"
	}
	if mode == "unavailable" && strings.Contains(string(source), "revision 1") {
		exit = 69
		diagnostic = "artifact: fixed-fixture.jar: unavailable\n"
	}
	if mode == "no-action" {
		output = []byte("unchanged native output\n")
	}
	if mode == "projector" {
		output = []byte("root=" + repo + "\nvalue=" + strings.TrimSpace(string(source)) + "\n")
	}
	mustWrite(t, filepath.Join(repo, "out/result.txt"), output)
	mustWrite(t, filepath.Join(repo, "out/diagnostics.txt"), []byte(diagnostic))
	outcome := "EXECUTED"
	action := true
	if mode == "no-action" {
		outcome = "UP-TO-DATE"
		action = false
	}
	if exit != 0 {
		outcome = "FAILED"
	}
	mustWrite(t, filepath.Join(evidence, "fixture-tasks.json"), jsonBytes([]TaskOutcome{{":fixture", outcome, action}}))
	if exit != 0 {
		os.Exit(exit)
	}
}

func runnableFixture(t *testing.T, count int, mode string) Manifest {
	m := fixtureManifest(t, count)
	m.Environment["BUILDOPT_FIXTURE_CHILD"] = "1"
	m.Command[len(m.Command)-1] = mode
	m.Outputs.Diagnostics = []OutputRule{{"out/diagnostics.txt", ":fixture", "exact"}}
	// Qualifying this process path on the real cgroup-v2 user manager is
	// mandatory. No fake systemd, skipped test or process-group substitute.
	return m
}

func runFixture(t *testing.T, m Manifest) *runner {
	t.Helper()
	path := filepath.Join(filepath.Dir(m.RunRoot), "fixture-manifest.json")
	mustWrite(t, path, jsonBytes(m))
	r, err := newRunner(path)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestOwnedReplayAndIndependentChecker(t *testing.T) {
	m := runnableFixture(t, 3, "success")
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != "FIXTURE_VERIFIED" || result.WorkflowStarts != 6 || len(result.Slots[0]) != 3 {
		t.Fatalf("wrong run: %+v", result)
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*RunResult){"positive-summary": func(r *RunResult) { r.Decision = "G3_PASS" }, "payback": func(r *RunResult) { r.Replications[0].PaybackOrdinal = 1 }, "duration": func(r *RunResult) { r.Slots[0][1].NativeNS++ }} {
		t.Run(name, func(t *testing.T) {
			changed := result
			changed.Replications = append([]Economics{}, result.Replications...)
			changed.Slots = [][]Slot{append([]Slot{}, result.Slots[0]...)}
			mutate(&changed)
			path := filepath.Join(m.RunRoot, "forged-"+name+".json")
			mustWrite(t, path, jsonBytes(changed))
			if checkResult(m.RunRoot, path) == nil {
				t.Fatal("checker accepted forged claim")
			}
		})
	}
}

func TestTwoFreshReplicationsReverseOrder(t *testing.T) {
	m := runnableFixture(t, 2, "success")
	m.Replications = 2
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 8 || len(result.Replications) != 2 || result.Decision != "FIXTURE_VERIFIED" {
		t.Fatal("incomplete independent replications")
	}
	for rep := 1; rep <= 2; rep++ {
		var pair PairRecord
		if err = readJSON(filepath.Join(m.RunRoot, "pairs", fmt.Sprintf("r%d-000.json", rep)), &pair); err != nil {
			t.Fatal(err)
		}
		var first AttemptStart
		if err = readJSON(filepath.Join(filepath.Dir(pair.Attempts[0].Path), "start.json"), &first); err != nil {
			t.Fatal(err)
		}
		want := "N"
		if rep == 2 {
			want = "I"
		}
		if first.Arm != want {
			t.Fatal("replication order was not reversed")
		}
		var before StatePin
		if err = readJSON(first.Before.Path, &before); err != nil {
			t.Fatal(err)
		}
		if before.Previous != "" || before.CandidateApplied {
			t.Fatal("replication borrowed prior mutable state")
		}
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}
