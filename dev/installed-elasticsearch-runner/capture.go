package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const fixtureClass = "LOCAL_PROTOCOL_FIXTURE"

type fileBinding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type captureFreeze struct {
	Schema           string      `json:"schemaVersion"`
	EnvironmentClass string      `json:"environmentClass"`
	Protocol         protocol    `json:"protocol"`
	Executable       fileBinding `json:"executable"`
	Prefix           []string    `json:"prefix"`
	Package          fileBinding `json:"package"`
	BootID           string      `json:"bootId"`
	StartedNS        int64       `json:"startedBootNanoseconds"`
	DeadlineNS       int64       `json:"deadlineBootNanoseconds"`
	OutputSelectors  []string    `json:"outputSelectors"`
}

type captureSummary struct {
	Schema               string `json:"schemaVersion"`
	EnvironmentClass     string `json:"environmentClass"`
	CompletedRows        int    `json:"completedRows"`
	Starts               int    `json:"starts"`
	ExpectedFailures     int    `json:"expectedFailures"`
	UnexpectedFailures   int    `json:"unexpectedFailures"`
	Complete             bool   `json:"complete"`
	PerformanceAuthority bool   `json:"performanceAuthority"`
}

type captureReceipt struct {
	Schema string  `json:"schemaVersion"`
	Files  []entry `json:"files"`
}

func fixtureInput(r row) []byte {
	variant := "original"
	if r.Block == "M" || r.State == "benign-comment" || r.State == "tab-violation" {
		variant = r.State
	}
	return []byte(fixtureClass + "\n" + r.Revision + "\n" + r.Workload + "\n" + variant + "\n")
}

func checkBinding(b fileBinding) error {
	if !filepath.IsAbs(b.Path) {
		return errors.New("binding path must be absolute")
	}
	canonical, err := filepath.EvalSymlinks(b.Path)
	if err != nil {
		return err
	}
	if canonical != b.Path {
		return errors.New("symlink in frozen binding")
	}
	h, err := hashFile(b.Path)
	if err != nil {
		return err
	}
	if h != b.SHA256 {
		return errors.New("frozen file bytes changed")
	}
	return nil
}

// runFixture qualifies the capture plumbing with owned executable fixtures.
// It has no Elasticsearch launcher or public execution mode. Every outer and
// nested fixture is an actual process; none of these rows can enter value gates.
func runFixture(ctx context.Context, root, executable string, prefix []string, pkg string, limit time.Duration) (pin string, retErr error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || limit <= 0 || limit > 120*time.Second {
		return "", errors.New("new absolute root and at most 120 seconds required")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(root))
	if err != nil || parent != filepath.Dir(root) {
		return "", errors.New("noncanonical capture parent")
	}
	f := captureFreeze{Schema: "buildopt.eic/fixture-freeze/v1", EnvironmentClass: fixtureClass, Protocol: makeProtocol(), Prefix: prefix, OutputSelectors: []string{"build/absent", "build/reports"}}
	f.Executable.Path = executable
	f.Executable.SHA256, err = hashFile(executable)
	if err != nil {
		return "", err
	}
	f.Package.Path = pkg
	f.Package.SHA256, err = hashFile(pkg)
	if err != nil {
		return "", err
	}
	if err = checkBinding(f.Executable); err != nil {
		return "", err
	}
	if err = checkBinding(f.Package); err != nil {
		return "", err
	}
	f.BootID, err = bootID()
	if err != nil {
		return "", err
	}
	f.StartedNS, err = bootNow()
	if err != nil {
		return "", err
	}
	f.DeadlineNS = f.StartedNS + int64(limit)
	if err = os.Mkdir(root, 0700); err != nil {
		return "", err
	}
	if err = writeJSON(filepath.Join(root, "freeze.json"), f); err != nil {
		return "", err
	}
	if err = os.Mkdir(filepath.Join(root, "attempts"), 0700); err != nil {
		return "", err
	}
	for _, arm := range []string{"N0", "N1", "W1"} {
		if err = os.MkdirAll(filepath.Join(root, "work", arm), 0700); err != nil {
			return "", err
		}
	}
	s := captureSummary{Schema: "buildopt.eic/fixture-summary/v1", EnvironmentClass: fixtureClass}
	defer func() {
		if err := writeJSON(filepath.Join(root, "summary.json"), s); err != nil {
			retErr = errors.Join(retErr, err)
			return
		}
		var err error
		pin, err = sealCapture(root)
		retErr = errors.Join(retErr, err)
	}()
	now, err := bootNow()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(f.DeadlineNS-now))
	defer cancel()
	for _, r := range f.Protocol.Rows {
		now, err := bootNow()
		if err != nil {
			return "", err
		}
		if ctx.Err() != nil || now >= f.DeadlineNS {
			return "", errors.New("fixture budget exhausted; retained partial rows")
		}
		if err = fixtureDiskGuard(root); err != nil {
			return "", err
		}
		if err = checkBinding(f.Executable); err != nil {
			return "", err
		}
		dir := filepath.Join(root, "attempts", r.ID)
		if err = os.Mkdir(dir, 0700); err != nil {
			return "", err
		}
		if err = writeJSON(filepath.Join(dir, "request.json"), r); err != nil {
			return "", err
		}
		if err = writeNew(filepath.Join(dir, "input.bin"), fixtureInput(r)); err != nil {
			return "", err
		}
		if err = os.WriteFile(filepath.Join(root, "work", r.Arm, "source.bin"), fixtureInput(r), 0600); err != nil {
			return "", err
		}
		args := append(append([]string{}, prefix...), root, r.ID)
		p, err := runProcess(ctx, dir, executable, args, min(2*time.Second, time.Duration(f.DeadlineNS-now)))
		if p.Started {
			s.Starts++
		}
		if err != nil {
			return "", err
		}
		if p.Outcome != "SUCCESS" && !(p.Outcome == "FAILURE" && p.ExitCode == 19 && r.ExpectedFailure) {
			s.UnexpectedFailures++
			return "", fmt.Errorf("fixture failure at %s: %s/%d; raw stderr retained", r.ID, p.Outcome, p.ExitCode)
		}
		var nested []processRecord
		if err = readJSON(filepath.Join(dir, "nested.json"), &nested); err != nil {
			return "", err
		}
		for _, n := range nested {
			if n.Started {
				s.Starts++
			}
		}
		if len(nested) != r.NestedStarts {
			return "", errors.New("fixture nested start count mismatch")
		}
		if p.Outcome == "SUCCESS" && !r.ExpectedFailure {
		} else if p.Outcome == "FAILURE" && p.ExitCode == 19 && r.ExpectedFailure {
			s.ExpectedFailures++
		} else {
			s.UnexpectedFailures++
			return "", fmt.Errorf("fixture failure at %s: %s/%d", r.ID, p.Outcome, p.ExitCode)
		}
		if err = os.Mkdir(filepath.Join(dir, "outputs"), 0700); err != nil {
			return "", err
		}
		if err = copyTree(filepath.Join(root, "work", r.Arm, "build"), filepath.Join(dir, "outputs", "build")); err != nil {
			return "", err
		}
		outputs, err := inventory(filepath.Join(dir, "outputs"), f.OutputSelectors)
		if err != nil {
			return "", err
		}
		if err = writeJSON(filepath.Join(dir, "outputs.json"), outputs); err != nil {
			return "", err
		}
		s.CompletedRows++
	}
	if err = checkBinding(f.Package); err != nil {
		return "", err
	}
	s.Complete = true
	return "", nil
}

func fixtureDiskGuard(root string) error {
	var stats unix.Statfs_t
	if err := unix.Statfs(root, &stats); err != nil {
		return err
	}
	if stats.Bavail*uint64(stats.Bsize) < 1<<30 {
		return errors.New("fixture requires 1 GiB free")
	}
	var bytes int64
	return filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		bytes += info.Size()
		if bytes > 64<<20 {
			return errors.New("fixture exceeded 64 MiB")
		}
		return nil
	})
}

// fixtureChild is deliberately a tiny owner-output producer, not a simulation
// of Gradle caching. The unit tests and CLI invoke this same executable path.
func fixtureChild(args []string) int {
	if len(args) == 1 && args[0] == "tree" {
		self, err := ownExecutable()
		if err != nil {
			return 65
		}
		prefix := append([]string{}, os.Args[1:len(os.Args)-1]...)
		child := exec.Command(self, append(prefix, "sleep")...)
		child.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
		child.Env = []string{"GORACE=atexit_sleep_ms=0"}
		if err = child.Start(); err != nil {
			return 65
		}
		fmt.Println(child.Process.Pid)
		if err = child.Wait(); err != nil {
			return 65
		}
		return 0
	}
	if len(args) == 1 && args[0] == "sleep" {
		time.Sleep(10 * time.Second)
		return 0
	}
	if len(args) == 1 && args[0] == "leaf" {
		fmt.Println("EIC_NESTED_FIXTURE")
		return 0
	}
	if len(args) != 2 {
		return 64
	}
	root, id := args[0], args[1]
	var f captureFreeze
	if err := readJSON(filepath.Join(root, "freeze.json"), &f); err != nil {
		return 65
	}
	self, err := ownExecutable()
	if err != nil || self != f.Executable.Path || checkBinding(f.Executable) != nil || validateProtocol(f.Protocol) != nil || f.EnvironmentClass != fixtureClass || len(os.Args) != len(f.Prefix)+3 || !reflect.DeepEqual(os.Args[1:len(f.Prefix)+1], f.Prefix) {
		return 65
	}
	var request row
	found := false
	for _, r := range f.Protocol.Rows {
		if r.ID == id {
			request = r
			found = true
			break
		}
	}
	if !found {
		return 65
	}
	dir := filepath.Join(root, "attempts", id)
	input, err := os.ReadFile(filepath.Join(root, "work", request.Arm, "source.bin"))
	if err != nil || string(input) != string(fixtureInput(request)) {
		return 65
	}
	nested := []processRecord{}
	if request.NestedStarts > 0 {
		if err = os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
			return 65
		}
		for i := 0; i < request.NestedStarts; i++ {
			childDir := filepath.Join(dir, "nested", fmt.Sprintf("%02d", i+1))
			if err = os.Mkdir(childDir, 0700); err != nil {
				return 65
			}
			p, e := runProcess(context.Background(), childDir, f.Executable.Path, append(append([]string{}, f.Prefix...), "leaf"), time.Second)
			nested = append(nested, p)
			if e != nil || p.Outcome != "SUCCESS" {
				return 65
			}
		}
	}
	if err = writeJSON(filepath.Join(dir, "nested.json"), nested); err != nil {
		return 65
	}
	output := filepath.Join(root, "work", request.Arm, "build", "reports")
	if err = os.MkdirAll(output, 0755); err != nil {
		return 65
	}
	path := filepath.Join(output, "report.txt")
	fmt.Println("EIC_OUTER_FIXTURE:" + id)
	if request.ExpectedFailure {
		if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
			return 65
		}
		fmt.Fprintln(os.Stderr, "EIC_EXPECTED_FAILURE:"+id)
		return 19
	}
	if err = os.WriteFile(path, []byte("input-sha256="+digest(input)+"\n"), 0644); err != nil {
		return 65
	}
	return 0
}

func sealCapture(root string) (string, error) {
	files, err := inventory(root, []string{"freeze.json", "attempts", "summary.json"})
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "receipt.json")
	if err = writeJSON(path, captureReceipt{Schema: "buildopt.eic/fixture-receipt/v1", Files: files}); err != nil {
		return "", err
	}
	return hashFile(path)
}

func checkCapture(root, pin string) (captureSummary, error) {
	s := captureSummary{Schema: "buildopt.eic/fixture-summary/v1", EnvironmentClass: fixtureClass}
	top, err := os.ReadDir(root)
	if err != nil {
		return s, err
	}
	allowedTop := map[string]bool{"freeze.json": true, "attempts": true, "summary.json": true, "receipt.json": true, "work": true}
	if len(top) != len(allowedTop) {
		return s, errors.New("extra or missing top-level capture entry")
	}
	for _, e := range top {
		if !allowedTop[e.Name()] {
			return s, errors.New("undeclared capture entry")
		}
	}
	actual, err := hashFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		return s, err
	}
	if actual != pin {
		return s, errors.New("receipt differs from external SHA-256 pin")
	}
	var receipt captureReceipt
	if err = readJSON(filepath.Join(root, "receipt.json"), &receipt); err != nil {
		return s, err
	}
	files, err := inventory(root, []string{"freeze.json", "attempts", "summary.json"})
	if err != nil {
		return s, err
	}
	if receipt.Schema != "buildopt.eic/fixture-receipt/v1" || !reflect.DeepEqual(files, receipt.Files) {
		return s, errors.New("retained evidence inventory drift")
	}
	var f captureFreeze
	if err = readJSON(filepath.Join(root, "freeze.json"), &f); err != nil {
		return s, err
	}
	if f.Schema != "buildopt.eic/fixture-freeze/v1" || f.EnvironmentClass != fixtureClass || f.BootID == "" || f.StartedNS < 0 || f.DeadlineNS <= f.StartedNS || f.DeadlineNS > f.StartedNS+int64(120*time.Second) || !reflect.DeepEqual(f.OutputSelectors, []string{"build/absent", "build/reports"}) {
		return s, errors.New("fixture identity or deadline drift")
	}
	if err = validateProtocol(f.Protocol); err != nil {
		return s, err
	}
	if err = checkBinding(f.Executable); err != nil {
		return s, err
	}
	if err = checkBinding(f.Package); err != nil {
		return s, err
	}
	dirs, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return s, err
	}
	if len(dirs) != len(f.Protocol.Rows) {
		return s, errors.New("missing or extra attempt; incomplete capture")
	}
	previous := f.StartedNS
	for _, r := range f.Protocol.Rows {
		dir := filepath.Join(root, "attempts", r.ID)
		var request row
		if err = readJSON(filepath.Join(dir, "request.json"), &request); err != nil {
			return s, err
		}
		if !reflect.DeepEqual(request, r) {
			return s, errors.New("request identity, order or arguments changed")
		}
		input, err := os.ReadFile(filepath.Join(dir, "input.bin"))
		if err != nil {
			return s, err
		}
		if string(input) != string(fixtureInput(r)) {
			return s, errors.New("altered fixture input history")
		}
		var p processRecord
		if err = readJSON(filepath.Join(dir, "process.json"), &p); err != nil {
			return s, err
		}
		if err = checkProcess(dir, p, previous, f.DeadlineNS); err != nil {
			return s, err
		}
		previous = p.EndNS
		s.Starts++
		if p.Outcome == "FAILURE" && p.ExitCode == 19 && r.ExpectedFailure {
			s.ExpectedFailures++
		} else if p.Outcome != "SUCCESS" || r.ExpectedFailure {
			s.UnexpectedFailures++
			return s, errors.New("lost or unexpected fixture failure")
		}
		stdout, err := os.ReadFile(filepath.Join(dir, "stdout.log"))
		if err != nil {
			return s, err
		}
		stderr, err := os.ReadFile(filepath.Join(dir, "stderr.log"))
		if err != nil {
			return s, err
		}
		wantError := ""
		if r.ExpectedFailure {
			wantError = "EIC_EXPECTED_FAILURE:" + r.ID + "\n"
		}
		if string(stdout) != "EIC_OUTER_FIXTURE:"+r.ID+"\n" || string(stderr) != wantError {
			return s, errors.New("fixture raw log/exit disagreement")
		}
		var nested []processRecord
		if err = readJSON(filepath.Join(dir, "nested.json"), &nested); err != nil {
			return s, err
		}
		if len(nested) != r.NestedStarts {
			return s, errors.New("missing or hidden nested starts")
		}
		last := p.StartNS
		for i, n := range nested {
			nestedDir := filepath.Join(dir, "nested", fmt.Sprintf("%02d", i+1))
			if err = checkProcess(nestedDir, n, last, p.EndNS); err != nil {
				return s, err
			}
			last = n.EndNS
			var raw processRecord
			if err = readJSON(filepath.Join(nestedDir, "process.json"), &raw); err != nil {
				return s, err
			}
			if raw != n || n.Outcome != "SUCCESS" {
				return s, errors.New("nested process lost a failure")
			}
			stdout, e := os.ReadFile(filepath.Join(nestedDir, "stdout.log"))
			if e != nil {
				return s, e
			}
			stderr, e := os.ReadFile(filepath.Join(nestedDir, "stderr.log"))
			if e != nil {
				return s, e
			}
			if string(stdout) != "EIC_NESTED_FIXTURE\n" || len(stderr) != 0 {
				return s, errors.New("nested raw log/exit disagreement")
			}
			s.Starts++
		}
		var outputs []entry
		if err = readJSON(filepath.Join(dir, "outputs.json"), &outputs); err != nil {
			return s, err
		}
		actual, err := inventory(filepath.Join(dir, "outputs"), f.OutputSelectors)
		if err != nil {
			return s, err
		}
		if !reflect.DeepEqual(outputs, actual) {
			return s, errors.New("output type, mode, size, bytes or absence changed")
		}
		want := []entry{{Path: "build/absent", Type: "absent"}, {Path: "build/reports", Type: "directory", Mode: 0755}}
		if !r.ExpectedFailure {
			data := []byte("input-sha256=" + digest(input) + "\n")
			want = append(want, entry{Path: "build/reports/report.txt", Type: "file", Mode: 0644, Size: int64(len(data)), SHA256: digest(data)})
		}
		if !reflect.DeepEqual(actual, want) {
			return s, errors.New("owner fixture output contract mismatch")
		}
		completeOutputs, err := inventory(filepath.Join(dir, "outputs"), []string{"build"})
		if err != nil {
			return s, err
		}
		physical := append([]entry{{Path: "build", Type: "directory", Mode: 0755}}, want[1:]...)
		if !reflect.DeepEqual(completeOutputs, physical) {
			return s, errors.New("undeclared owner output")
		}
		outputRoots, err := os.ReadDir(filepath.Join(dir, "outputs"))
		if err != nil {
			return s, err
		}
		if len(outputRoots) != 1 || outputRoots[0].Name() != "build" {
			return s, errors.New("undeclared output root")
		}
		if err = checkAttemptLayout(dir, r); err != nil {
			return s, err
		}
		s.CompletedRows++
	}
	s.Complete = s.CompletedRows == 300 && s.Starts == 312 && s.UnexpectedFailures == 0
	var reported captureSummary
	if err = readJSON(filepath.Join(root, "summary.json"), &reported); err != nil {
		return s, err
	}
	if !reflect.DeepEqual(s, reported) {
		return s, errors.New("summary disagrees with reconstructed raw rows")
	}
	return s, nil
}

func checkProcess(dir string, p processRecord, start, end int64) error {
	if !p.Started || p.PID <= 0 || p.StartNS < start || p.EndNS <= p.StartNS || p.EndNS > end || p.Signal != 0 || (p.Outcome != "SUCCESS" && p.Outcome != "FAILURE") || (p.Outcome == "SUCCESS") != (p.ExitCode == 0) {
		return errors.New("invalid, overlapping or unsuccessful process boundaries")
	}
	var started processRecord
	if err := readJSON(filepath.Join(dir, "started.json"), &started); err != nil {
		return err
	}
	if !started.Started || started.PID != p.PID || started.StartNS != p.StartNS || started.EndNS != 0 || started.ExitCode != -1 || started.Signal != 0 || started.Outcome != "NOT_STARTED" {
		return errors.New("start reservation drift")
	}
	return nil
}

func checkAttemptLayout(dir string, r row) error {
	allowed := map[string]bool{"request.json": true, "input.bin": true, "stdout.log": true, "stderr.log": true, "started.json": true, "process.json": true, "nested.json": true, "outputs.json": true, "outputs": true}
	if r.NestedStarts > 0 {
		allowed["nested"] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) != len(allowed) {
		return errors.New("extra or missing raw attempt file")
	}
	for _, e := range entries {
		if !allowed[e.Name()] {
			return errors.New("undeclared raw attempt file")
		}
	}
	if r.NestedStarts > 0 {
		dirs, err := os.ReadDir(filepath.Join(dir, "nested"))
		if err != nil {
			return err
		}
		if len(dirs) != r.NestedStarts {
			return errors.New("undeclared nested process")
		}
		for i, d := range dirs {
			if d.Name() != fmt.Sprintf("%02d", i+1) || !d.IsDir() {
				return errors.New("nested order drift")
			}
			files, err := os.ReadDir(filepath.Join(dir, "nested", d.Name()))
			if err != nil {
				return err
			}
			if len(files) != 4 {
				return errors.New("nested raw files drift")
			}
			for _, file := range files {
				if file.Name() != "stdout.log" && file.Name() != "stderr.log" && file.Name() != "started.json" && file.Name() != "process.json" {
					return errors.New("unknown nested evidence")
				}
			}
		}
	}
	return nil
}

// Ensure the fixture entrypoint is the same executable as the capture parent.
// The CLI never accepts an arbitrary child command or an Elasticsearch root.
func ownExecutable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}
