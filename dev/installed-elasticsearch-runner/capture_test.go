package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestFixtureChildProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--eic-child" {
			os.Exit(fixtureChild(os.Args[i+1:]))
		}
	}
}

func fixtureCapture(t *testing.T) (string, string) {
	t.Helper()
	parent := t.TempDir()
	pkg := filepath.Join(parent, "package.tar.gz")
	if err := os.WriteFile(pkg, []byte("local fixture package identity, not a distribution"), 0600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "capture")
	pin, err := runFixture(context.Background(), root, exe, []string{"-test.run=^TestFixtureChildProcess$", "--", "--eic-child"}, pkg, 120*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return root, pin
}

func TestCaptureAndIndependentReconstruction(t *testing.T) {
	root, pin := fixtureCapture(t)
	s, err := checkCapture(root, pin)
	if err != nil {
		t.Fatal(err)
	}
	if s.CompletedRows != 300 || s.Starts != 312 || s.ExpectedFailures != 4 || s.UnexpectedFailures != 0 || !s.Complete || s.PerformanceAuthority {
		t.Fatalf("%+v", s)
	}
	before, err := inventory(filepath.Join(root, "work", "N0"), []string{"build"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = checkCapture(root, pin); err != nil {
		t.Fatal(err)
	}
	after, err := inventory(filepath.Join(root, "work", "N0"), []string{"build"})
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("checker changed outputs", err)
	}
	for _, mutate := range []struct {
		name   string
		change func(string)
	}{
		{"lost-failure", func(r string) {
			var p processRecord
			path := filepath.Join(r, "attempts", "C009", "process.json")
			readJSON(path, &p)
			p.ExitCode = 0
			p.Outcome = "SUCCESS"
			replaceJSON(t, path, p)
		}},
		{"missing-nested", func(r string) {
			path := filepath.Join(r, "attempts", "T003", "nested.json")
			var p []processRecord
			readJSON(path, &p)
			replaceJSON(t, path, p[:5])
		}},
		{"hidden-nested", func(r string) {
			path := filepath.Join(r, "attempts", "T003", "nested.json")
			var p []processRecord
			readJSON(path, &p)
			replaceJSON(t, path, append(p, p[0]))
		}},
		{"reordered-row", func(r string) {
			path := filepath.Join(r, "attempts", "P001", "request.json")
			var q row
			readJSON(path, &q)
			q.ID = "P002"
			replaceJSON(t, path, q)
		}},
		{"input-drift", func(r string) {
			os.WriteFile(filepath.Join(r, "attempts", "P001", "input.bin"), []byte("changed"), 0600)
		}},
		{"forged-summary", func(r string) {
			path := filepath.Join(r, "summary.json")
			var s captureSummary
			readJSON(path, &s)
			s.Starts--
			replaceJSON(t, path, s)
		}},
		{"lost-output", func(r string) {
			os.Remove(filepath.Join(r, "attempts", "P001", "outputs", "build", "reports", "report.txt"))
		}},
		{"lost-row", func(r string) { os.RemoveAll(filepath.Join(r, "attempts", "P002")) }},
		{"undeclared-owner-output", func(r string) {
			os.WriteFile(filepath.Join(r, "attempts", "P001", "outputs", "build", "extra.txt"), []byte("extra"), 0600)
		}},
		{"overlapping-boundaries", func(r string) {
			path := filepath.Join(r, "attempts", "P002", "process.json")
			var p processRecord
			readJSON(path, &p)
			p.StartNS = 0
			replaceJSON(t, path, p)
		}},
		{"extra-raw-file", func(r string) { os.WriteFile(filepath.Join(r, "attempts", "unknown"), []byte("hidden"), 0600) }},
		{"nested-log-failure", func(r string) {
			os.WriteFile(filepath.Join(r, "attempts", "T003", "nested", "01", "stderr.log"), []byte("failure was hidden"), 0600)
		}},
		{"top-level-extra", func(r string) { os.WriteFile(filepath.Join(r, "unaccounted-build.json"), []byte("{}"), 0600) }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			copyRoot := filepath.Join(t.TempDir(), "capture")
			if err := copyTree(root, copyRoot); err != nil {
				t.Fatal(err)
			}
			mutate.change(copyRoot)
			if _, err := checkCapture(copyRoot, pin); err == nil {
				t.Fatal("tampering bypassed pinned receipt")
			}
			// Also exercise semantic reconstruction, without relying on a stale seal.
			os.Remove(filepath.Join(copyRoot, "receipt.json"))
			newPin, err := sealCapture(copyRoot)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = checkCapture(copyRoot, newPin); err == nil {
				t.Fatal("forged raw capture accepted after resealing")
			}
		})
	}
}

func replaceJSON(t *testing.T, path string, value any) {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryTypesAbsenceAndEscapes(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "out"), 0750)
	os.WriteFile(filepath.Join(root, "out", "a"), []byte("a"), 0640)
	got, err := inventory(root, []string{"out", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Type != "absent" || got[1].Type != "directory" || got[2].Mode != 0640 {
		t.Fatalf("%+v", got)
	}
	for _, selectors := range [][]string{{"../escape"}, {"out", "out/a"}, {"out", "out"}, {"/tmp"}, {"."}} {
		if _, err := inventory(root, selectors); err == nil {
			t.Fatalf("unsafe selectors %v", selectors)
		}
	}
	os.Symlink("/tmp", filepath.Join(root, "out", "link"))
	if _, err = inventory(root, []string{"out"}); err == nil {
		t.Fatal("symlink followed")
	}
}

func TestTimeoutAndCancellationRetainActualProcessFailure(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, cancelNow := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if cancelNow {
			cancel()
		}
		defer cancel()
		root := t.TempDir()
		p, err := runProcess(ctx, root, exe, []string{"-test.run=^TestFixtureChildProcess$", "--", "--eic-child", "sleep"}, 100*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		if cancelNow {
			if p.Started {
				t.Fatal("started after cancellation")
			}
		} else if !p.Started || p.Outcome != "TIMEOUT" || p.Signal == 0 || p.EndNS <= p.StartNS {
			t.Fatalf("%+v", p)
		}
	}
}

func TestTimeoutTerminatesFixtureDescendant(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p, err := runProcess(context.Background(), dir, exe, []string{"-test.run=^TestFixtureChildProcess$", "--", "--eic-child", "tree"}, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if p.Outcome != "TIMEOUT" {
		t.Fatalf("%+v", p)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "stdout.log"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		t.Fatalf("descendant PID %q: %v", raw, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, readErr := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if os.IsNotExist(readErr) || (readErr == nil && strings.Contains(string(data), ") Z ")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatal("owned fixture descendant survived parent timeout")
}

func TestStrictJSON(t *testing.T) {
	for _, input := range []string{`{"chargedSeconds":4440,"chargedSeconds":0}`, `{"chargedSeconds":4440} {}`, `{"chargedSeconds":4440,"unknown":true}`} {
		path := filepath.Join(t.TempDir(), "in.json")
		os.WriteFile(path, []byte(input), 0600)
		var b budgetInput
		if err := readJSON(path, &b); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	if _, err := checkCapture(t.TempDir(), strings.Repeat("0", 64)); err == nil {
		t.Fatal("missing receipt accepted")
	}
}

func TestFixtureBudgetStopsWithoutResetOrOverwrite(t *testing.T) {
	parent := t.TempDir()
	pkg := filepath.Join(parent, "package")
	os.WriteFile(pkg, []byte("fixture"), 0600)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "capture")
	pin, err := runFixture(context.Background(), root, exe, []string{"-test.run=^TestFixtureChildProcess$", "--", "--eic-child"}, pkg, time.Nanosecond)
	if err == nil || pin == "" {
		t.Fatal("exhausted capture did not retain a sealed partial result", err)
	}
	var s captureSummary
	if err = readJSON(filepath.Join(root, "summary.json"), &s); err != nil {
		t.Fatal(err)
	}
	if s.Starts != 0 || s.Complete {
		t.Fatalf("%+v", s)
	}
	if _, err = checkCapture(root, pin); err == nil {
		t.Fatal("partial capture reported complete")
	}
	if _, err = runFixture(context.Background(), root, exe, nil, pkg, time.Second); err == nil {
		t.Fatal("occupied campaign restarted")
	}
	after, err := hashFile(filepath.Join(root, "receipt.json"))
	if err != nil || after != pin {
		t.Fatal("restart altered previous evidence", err)
	}
}
