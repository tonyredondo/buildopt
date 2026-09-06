package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestChild(t *testing.T) {
	mode := os.Getenv("CNC_TEST_CHILD")
	if mode == "" {
		return
	}
	root := os.Getenv("CNC_TEST_REPORT_ROOT")
	if mode == "report" || mode == "duplicate" || mode == "ambiguous" {
		path := filepath.Join(root, "one", "configuration-cache-report.html")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			os.Exit(21)
		}
		if err := os.WriteFile(path, []byte("<html>fixture-owned report</html>"), 0600); err != nil {
			os.Exit(22)
		}
		fmt.Println("See the complete report at file://" + path)
		if mode == "duplicate" {
			fmt.Println("See the complete report at file://" + path)
		}
		if mode == "ambiguous" {
			fmt.Println("See the complete report at file://" + filepath.Join(root, "two", "configuration-cache-report.html"))
		}
		os.Exit(1)
	}
	if mode == "wait" {
		if path := os.Getenv("CNC_TEST_HANDSHAKE"); path != "" {
			if err := os.WriteFile(path, []byte("started"), 0600); err != nil {
				os.Exit(23)
			}
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if mode == "detached" {
		child := exec.Command("/bin/sleep", "30")
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := child.Start(); err != nil {
			os.Exit(24)
		}
		if err := os.WriteFile(os.Getenv("CNC_TEST_HANDSHAKE"), []byte(fmt.Sprint(child.Process.Pid)), 0600); err != nil {
			_ = child.Process.Kill()
			os.Exit(25)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if mode == "failure" {
		fmt.Fprintln(os.Stderr, "intentional task failure")
		os.Exit(7)
	}
	fmt.Println("native child completed")
	os.Exit(0)
}

func setup(t *testing.T) (string, campaign, func() (clockReading, error)) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	now := clockReading{"test-boot", 100}
	if err := initialize(root, strings.Repeat("a", 64), now); err != nil {
		t.Fatal(err)
	}
	var state campaign
	if err := readJSON(filepath.Join(root, "state.json"), &state); err != nil {
		t.Fatal(err)
	}
	return root, state, func() (clockReading, error) { return now, nil }
}

func childRequest(t *testing.T, root, slot, mode string) request {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	reportRoot := filepath.Join(root, "source", "build", "reports", "configuration-cache")
	return request{Slot: slot, Directory: root, Arguments: []string{binary, "-test.run=^TestChild$"}, Environment: []string{"CNC_TEST_CHILD=" + mode, "CNC_TEST_REPORT_ROOT=" + reportRoot}, ReportRoot: reportRoot, RequiredArtifacts: []string{}, TimeoutSeconds: 5}
}

func TestCaptureReportsAndFailures(t *testing.T) {
	for _, tc := range []struct{ mode, slot, want string }{
		{"success", "P01", "CHILD_SUCCESS"}, {"failure", "P01", "CHILD_FAILURE"},
		{"report", "D01", "ROOT_REPORT_CAPTURED"}, {"duplicate", "D01", "ROOT_REPORT_CAPTURED"},
		{"ambiguous", "D01", "ROOT_REPORT_REFERENCE_AMBIGUOUS"}, {"success", "D01", "ROOT_REPORT_REFERENCE_MISSING"},
	} {
		t.Run(tc.mode+tc.slot, func(t *testing.T) {
			root, state, clock := setup(t)
			r := childRequest(t, root, tc.slot, tc.mode)
			got, err := capture(context.Background(), root, state, r, clock, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Outcome != tc.want || !got.Started {
				t.Fatalf("unexpected result: %+v", got)
			}
			results, err := inspect(root, state)
			if err != nil || len(results) != 1 {
				t.Fatalf("inspect: %v %v", results, err)
			}
			if _, err = capture(context.Background(), root, state, r, clock, nil); err == nil {
				t.Fatal("duplicate attempt started")
			}
		})
	}
}

func TestTimeoutAndCancellation(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		root, state, clock := setup(t)
		r := childRequest(t, root, "P01", "wait")
		r.TimeoutSeconds = .05
		got, err := capture(context.Background(), root, state, r, clock, nil)
		if err != nil || got.Outcome != "TIME_LIMIT" {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("handshake cancellation", func(t *testing.T) {
		root, state, clock := setup(t)
		r := childRequest(t, root, "P01", "wait")
		handshake := filepath.Join(root, "handshake")
		r.Environment = append(r.Environment, "CNC_TEST_HANDSHAKE="+handshake)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			limit := time.Now().Add(3 * time.Second)
			for time.Now().Before(limit) {
				if _, err := os.Stat(handshake); err == nil {
					cancel()
					return
				}
				time.Sleep(time.Millisecond)
			}
		}()
		got, err := capture(ctx, root, state, r, clock, nil)
		<-done
		if err != nil || got.Outcome != "CANCELLED" {
			t.Fatalf("%+v %v", got, err)
		}
	})
}

func TestPreStartAndPostStartFailures(t *testing.T) {
	root, state, clock := setup(t)
	r := childRequest(t, root, "P01", "success")
	got, err := capture(context.Background(), root, state, r, clock, func() error { return fmt.Errorf("source drift") })
	if err != nil || got.Started || got.Outcome != "PRE_START_FAILURE" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = inspect(root, state); err != nil {
		t.Fatal(err)
	}
	root, state, clock = setup(t)
	r = childRequest(t, root, "P01", "success")
	r.Arguments[0] = filepath.Join(root, "absent")
	got, err = capture(context.Background(), root, state, r, clock, nil)
	if err != nil || got.Started || got.Outcome != "PRE_START_FAILURE" {
		t.Fatalf("%+v %v", got, err)
	}
	root, state, clock = setup(t)
	r = childRequest(t, root, "M01", "success")
	r.RequiredArtifacts = []string{"operations-log.txt"}
	got, err = capture(context.Background(), root, state, r, clock, nil)
	if err != nil || !got.Started || got.Outcome != "HARNESS_FAILURE" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestBudgetLockAndAmbiguousRecovery(t *testing.T) {
	root, state, clock := setup(t)
	if _, err := remaining(state, clockReading{"other-boot", 101}); err == nil {
		t.Fatal("reboot reset allowed")
	}
	if _, err := remaining(state, clockReading{"test-boot", 99}); err == nil {
		t.Fatal("backwards clock allowed")
	}
	if _, err := remaining(state, clockReading{"test-boot", 7300}); err == nil {
		t.Fatal("deadline extended")
	}
	state.MaximumStarts = 61
	if _, err := remaining(state, clockReading{"test-boot", 101}); err == nil {
		t.Fatal("start limit changed")
	}
	state.MaximumStarts = 60
	first, err := lockCampaign(root)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := lockCampaign(root); err == nil {
		second.Close()
		t.Fatal("concurrent owner allowed")
	}
	first.Close()
	directory := filepath.Join(root, "attempts", "P01")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeNewJSON(filepath.Join(directory, "reservation.json"), reservation{Slot: "P01", PackageSHA256: state.PackageSHA256, BootSeconds: state.Started}); err != nil {
		t.Fatal(err)
	}
	if _, err := inspect(root, state); err == nil {
		t.Fatal("ambiguous reservation silently recovered")
	}
	if _, err := capture(context.Background(), root, state, childRequest(t, root, "P01", "success"), clock, nil); err == nil {
		t.Fatal("reserved attempt repeated")
	}
}

func TestReadOnlyInspectionAndTampering(t *testing.T) {
	root, state, clock := setup(t)
	r := childRequest(t, root, "P01", "success")
	if _, err := capture(context.Background(), root, state, r, clock, nil); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	if _, err := inspect(root, state); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, root); before != after {
		t.Fatal("check changed state")
	}
	if err := os.WriteFile(filepath.Join(root, "attempts", "P01", "child.log"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := inspect(root, state); err == nil {
		t.Fatal("log tampering accepted")
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Fprint(hash, path)
			hash.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func TestGroupCancellation(t *testing.T) {
	root, state, clock := setup(t)
	handshake := filepath.Join(root, "grandchild.pid")
	// This subprocess is the actual command-group consumer, not a mocked wait.
	r := request{Slot: "P01", Directory: root, Arguments: []string{"/bin/sh", "-c", "sleep 30 & echo $! > \"$1\"; wait", "fixture", handshake}, Environment: []string{"PATH=/usr/bin:/bin"}, ReportRoot: filepath.Join(root, "reports"), TimeoutSeconds: .1}
	got, err := capture(context.Background(), root, state, r, clock, nil)
	if err != nil || got.Outcome != "TIME_LIMIT" {
		t.Fatalf("%+v %v", got, err)
	}
	data, err := os.ReadFile(handshake)
	if err != nil {
		t.Fatal(err)
	}
	var pid int
	if _, err = fmt.Sscanf(string(data), "%d", &pid); err != nil {
		t.Fatal(err)
	}
	if err = syscall.Kill(pid, 0); err == nil {
		// A reparented zombie is terminated, not a live child. The runner cannot
		// reap a process it did not parent.
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err == nil && !strings.Contains(string(stat), ") Z ") {
			t.Fatal("grandchild survived cancellation")
		}
	}
}

func TestMissingExecutableDoesNotLaunchShell(t *testing.T) {
	if _, err := exec.LookPath("cnc-nonexistent-executable"); err == nil {
		t.Fatal("unexpected fixture command")
	}
}

func TestDetachedProcessIsNotCoveredByGroupProof(t *testing.T) {
	root, state, clock := setup(t)
	handshake := filepath.Join(root, "detached.pid")
	r := childRequest(t, root, "P01", "detached")
	r.Environment = append(r.Environment, "CNC_TEST_HANDSHAKE="+handshake)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Wait for actual ownership evidence, not a scheduler-dependent short sleep.
	ready := make(chan int, 1)
	go func() {
		deadline := time.NewTimer(3 * time.Second)
		defer deadline.Stop()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				data, err := os.ReadFile(handshake)
				var pid int
				if err == nil {
					if _, err = fmt.Sscanf(string(data), "%d", &pid); err == nil && pid > 0 {
						ready <- pid
						cancel()
						return
					}
				}
			case <-deadline.C:
				ready <- 0
				cancel()
				return
			}
		}
	}()
	got, err := capture(ctx, root, state, r, clock, nil)
	pid := <-ready
	if pid > 0 {
		defer syscall.Kill(pid, syscall.SIGKILL) // Only this fixture's recorded child.
	}
	if err != nil || got.Outcome != "CANCELLED" || pid <= 0 {
		t.Fatalf("detached fixture: %+v pid=%d %v", got, pid, err)
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("fixture no longer demonstrates detached lifetime: %v", err)
	}
	// This reproduces why a process group alone is insufficient. The separate
	// host-only TestSystemdOwnership proves the production cgroup consuming path.
}

func TestIndependentProcessMarkerAndSignal(t *testing.T) {
	for _, mode := range []string{"marker-slot", "marker-pid", "signal"} {
		t.Run(mode, func(t *testing.T) {
			root, state, clock := setup(t)
			r := childRequest(t, root, "P01", "success")
			if _, err := capture(context.Background(), root, state, r, clock, nil); err != nil {
				t.Fatal(err)
			}
			mutate := func(name string, value any) {
				t.Helper()
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "attempts/P01", name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "marker-slot":
				mutate("started.json", startedEvidence{1, "P02"})
			case "marker-pid":
				mutate("started.json", startedEvidence{0, "P01"})
			case "signal":
				var raw processEvidence
				var summary result
				if err := readJSON(filepath.Join(root, "attempts/P01/process.json"), &raw); err != nil {
					t.Fatal(err)
				}
				if err := readJSON(filepath.Join(root, "attempts/P01/result.json"), &summary); err != nil {
					t.Fatal(err)
				}
				raw.Signal, summary.Signal = 9, 9
				mutate("process.json", raw)
				mutate("result.json", summary)
			}
			if _, err := inspect(root, state); err == nil {
				t.Fatal("forged process evidence accepted")
			}
		})
	}
}

func TestTimeoutRetainsClassificationWhenArtifactsAreMissing(t *testing.T) {
	root, state, clock := setup(t)
	r := childRequest(t, root, "M01", "wait")
	r.TimeoutSeconds = 0.1
	r.RequiredArtifacts = []string{"missing-trace.txt"}
	r.Outputs = &outputPolicy{Selectors: []string{"build/*.jar"}, Producers: []string{":jar"}}
	got, err := capture(context.Background(), root, state, r, clock, nil)
	if err != nil || !got.Started || got.Outcome != "TIME_LIMIT" {
		t.Fatalf("timeout masked by missing artifact: %+v %v", got, err)
	}
	if _, err := inspect(root, state); err != nil {
		t.Fatal(err)
	}
}
