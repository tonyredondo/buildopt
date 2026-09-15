//go:build linux && amd64 && replay_integration

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveDiskChild(t *testing.T) {
	if os.Getenv("BUILDOPT_FIXTURE_CHILD") != "1" {
		return
	}
	repo, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	mode := os.Args[len(os.Args)-1]
	file := filepath.Join(repo, "out", "active")
	mustWrite(t, file, []byte("x"))
	time.Sleep(250 * time.Millisecond)
	if mode == "ceiling" {
		check(os.Truncate(file, 80<<20))
		time.Sleep(30 * time.Second)
		t.Fatal("size guard did not cancel native child")
	}
	tmp := filepath.Join(repo, "out", "temporary")
	check(os.Mkdir(tmp, 0700))
	mustWrite(t, filepath.Join(tmp, "file"), []byte("temporary"))
	time.Sleep(200 * time.Millisecond)
	if mode == "fallback" {
		check(os.Rename(tmp, tmp+"-renamed"))
		tmp += "-renamed"
	}
	check(os.RemoveAll(tmp))
	check(os.Truncate(file, 1024))
	time.Sleep(250 * time.Millisecond)
	source, err := os.ReadFile(filepath.Join(repo, "source.txt"))
	check(err)
	mustWrite(t, filepath.Join(repo, "out", "result.txt"), source)
	mustWrite(t, filepath.Join(repo, "out", "diagnostics.txt"), []byte("success\n"))
	evidence := filepath.Dir(os.Getenv("BUILDOPT_REPLAY_CAPTURE"))
	mustWrite(t, filepath.Join(evidence, "fixture-tasks.json"), jsonBytes([]TaskOutcome{{":fixture", "EXECUTED", true}}))
}

func TestLiveDiskRunnerIntegration(t *testing.T) {
	for _, mode := range []string{"mutations", "fallback", "ceiling"} {
		t.Run(mode, func(t *testing.T) {
			m := runnableFixture(t, 1, mode)
			m.Command[1] = "-test.run=^TestLiveDiskChild$"
			m.Limits.RetryPairs = 0
			r := runFixture(t, m)
			err := r.execute(1, 0, -1)
			if mode == "ceiling" {
				if err == nil {
					t.Fatal("size breach did not stop the runner")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				result, err := writeResult(m.RunRoot)
				if err != nil {
					t.Fatal(err)
				}
				if result.Decision != "FIXTURE_VERIFIED" || result.WorkflowStarts != 2 {
					t.Fatalf("incomplete fixture: %+v", result)
				}
				if err := checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
					t.Fatal(err)
				}
			}
			paths, err := filepath.Glob(filepath.Join(m.RunRoot, "attempts", "*", "disk-observer.json"))
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if mode == "ceiling" {
				want = 1
			}
			if len(paths) != want {
				t.Fatalf("observer receipts=%d want=%d", len(paths), want)
			}
			for _, path := range paths {
				var receipt DiskObserverReceipt
				if err := readJSON(path, &receipt); err != nil {
					t.Fatal(err)
				}
				if receipt.Status["checks"].(float64) < 2 {
					t.Fatal("fixture did not exercise live polling")
				}
				fallback := receipt.Status["fallbackReason"].(string)
				if (fallback != "") != (mode == "fallback") {
					t.Fatalf("unexpected fallback: %q", fallback)
				}
				var native ProcessReceipt
				if err := readJSON(filepath.Join(filepath.Dir(path), "native-finish.json"), &native); err != nil {
					t.Fatal(err)
				}
				if mode == "ceiling" && native.Outcome != "OWNERSHIP_OR_LIMIT_FAILURE" {
					t.Fatalf("size cancellation lost: %+v", native)
				}
				if mode == "ceiling" && native.End.NS-native.Start.NS > int64(3*time.Second) {
					t.Fatal("live size cancellation exceeded three seconds")
				}
				if err := checkDiskReceipt(m, filepath.Dir(path), native); err != nil {
					t.Fatal(err)
				}
				ps, err := cgroupProcesses(native.Supervisor.Cgroup)
				if !os.IsNotExist(err) && (err != nil || len(ps) != 0) {
					t.Fatalf("owned processes remain: %v %v", ps, err)
				}
				if mode == "mutations" {
					raw, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")) == nil {
						t.Fatal("missing disk receipt qualified")
					}
					mustWrite(t, path, raw)
				}
				t.Logf("%s: %+v", native.Attempt, receipt.Status)
			}
		})
	}
}
