package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/strictdiagnostic"
)

type request struct {
	Slot              string        `json:"slot"`
	Directory         string        `json:"directory"`
	Arguments         []string      `json:"arguments"`
	Environment       []string      `json:"environment"`
	ReportRoot        string        `json:"reportRoot"`
	RequiredArtifacts []string      `json:"requiredArtifacts"`
	TimeoutSeconds    float64       `json:"timeoutSeconds"`
	Ownership         *ownership    `json:"ownership,omitempty"`
	Outputs           *outputPolicy `json:"outputs,omitempty"`
	Native            *nativePaths  `json:"nativePaths,omitempty"`
}
type artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type result struct {
	Schema       string                     `json:"schemaVersion"`
	Slot         string                     `json:"slot"`
	Started      bool                       `json:"started"`
	ExitCode     int                        `json:"exitCode"`
	Signal       int                        `json:"signal"`
	StdoutSHA256 string                     `json:"stdoutSha256"`
	StderrSHA256 string                     `json:"stderrSha256"`
	Outcome      string                     `json:"outcome"`
	Detail       string                     `json:"detail"`
	Start        float64                    `json:"startBootSeconds"`
	End          float64                    `json:"endBootSeconds"`
	LogSHA256    string                     `json:"logSha256"`
	Selection    strictdiagnostic.Selection `json:"reportSelection"`
	Artifacts    []artifact                 `json:"artifacts"`
}

type processEvidence struct {
	Started            bool    `json:"started"`
	ExitCode           int     `json:"exitCode"`
	Signal             int     `json:"signal"`
	Outcome            string  `json:"outcome"`
	Detail             string  `json:"detail"`
	Start              float64 `json:"startBootSeconds"`
	End                float64 `json:"endBootSeconds"`
	ElapsedNanoseconds int64   `json:"elapsedNanoseconds"`
}

type startedEvidence struct {
	PID  int    `json:"pid"`
	Slot string `json:"slot"`
}

func artifactFailureCanClassify(outcome string) bool {
	return outcome == "CHILD_SUCCESS" || outcome == "CHILD_FAILURE" || outcome == "ROOT_REPORT_CAPTURED"
}

// The OS writer serializes stdout/stderr and exposes persistence errors to the
// capture owner. A child exit zero cannot hide a full disk or failed writer.
type logWriter struct {
	mu    sync.Mutex
	file  *os.File
	err   error
	bytes int64
}

type streamWriter struct {
	combined *logWriter
	stream   *os.File
}

func (w streamWriter) Write(p []byte) (int, error) {
	// One lock also preserves correspondence between the merged and split logs.
	w.combined.mu.Lock()
	defer w.combined.mu.Unlock()
	n, err := w.combined.write(p)
	if err != nil {
		return n, err
	}
	n, err = w.stream.Write(p)
	if err != nil {
		w.combined.err = err
	}
	return n, err
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.write(p)
}
func (w *logWriter) write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.bytes+int64(len(p)) > 100<<20 {
		w.err = errors.New("100 MiB log limit exceeded")
		return 0, w.err
	}
	n, err := w.file.Write(p)
	w.bytes += int64(n)
	w.err = err
	return n, err
}

func capture(ctx context.Context, root string, state campaign, r request, clock func() (clockReading, error), beforeStart func() error, guards ...func() error) (result, error) {
	terminal := result{Schema: "buildopt.cnc/attempt/v1", Slot: r.Slot, ExitCode: -1, Artifacts: []artifact{}}
	now, err := clock()
	if err != nil {
		return terminal, err
	}
	left, err := remaining(state, now)
	if err != nil {
		return terminal, err
	}
	if r.TimeoutSeconds <= 0 || r.TimeoutSeconds > 1200 || len(r.Arguments) == 0 {
		return terminal, errors.New("invalid child timeout or argv")
	}
	directory := filepath.Join(root, "attempts", r.Slot)
	if !validSlot(r.Slot) {
		return terminal, errors.New("unknown slot")
	}
	if err = os.Mkdir(directory, 0700); err != nil {
		return terminal, err
	}
	if err = writeNewJSON(filepath.Join(directory, "request.json"), r); err != nil {
		return terminal, err
	}
	requestHash, err := fileDigest(filepath.Join(directory, "request.json"))
	if err != nil {
		return terminal, err
	}
	if err = writeNewJSON(filepath.Join(directory, "reservation.json"), reservation{r.Slot, state.PackageSHA256, now.Seconds, requestHash}); err != nil {
		return terminal, err
	}
	terminal.Start = now.Seconds
	logPath := filepath.Join(directory, "child.log")
	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return terminal, err
	}
	writer := &logWriter{file: file}
	stdout, err := os.OpenFile(filepath.Join(directory, "stdout.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		file.Close()
		return terminal, err
	}
	stderr, err := os.OpenFile(filepath.Join(directory, "stderr.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		file.Close()
		stdout.Close()
		return terminal, err
	}
	timeout := time.Duration(min(left, r.TimeoutSeconds) * float64(time.Second))
	childContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := exec.Command(r.Arguments[0], r.Arguments[1:]...)
	command.Dir = r.Directory
	command.Env = r.Environment
	command.Stdout = streamWriter{writer, stdout}
	command.Stderr = streamWriter{writer, stderr}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var ownedGroup *os.File
	if r.Ownership != nil {
		owned, ownershipErr := openOwnership(childContext, root, state, *r.Ownership)
		if ownershipErr != nil {
			err = ownershipErr
		} else {
			ownedGroup = owned
			defer owned.Close()
			applyOwnership(command, owned)
			start, leaseErr := processStartTicks(os.Getpid())
			if leaseErr == nil {
				leaseErr = writeNewJSON(filepath.Join(directory, "lease.json"), ownershipLease{os.Getpid(), start, min(state.Deadline, now.Seconds+r.TimeoutSeconds)})
			}
			if leaseErr != nil {
				err = leaseErr
			}
		}
	}
	command.WaitDelay = time.Second
	var processStart, processEnd float64
	var processElapsed int64
	var childStartedAt time.Time
	if err == nil && beforeStart != nil {
		err = beforeStart()
	}
	if err == nil {
		err = childContext.Err()
	}
	if err == nil {
		latest, clockErr := clock()
		if clockErr != nil {
			err = clockErr
		} else {
			_, err = remaining(state, latest)
			if err == nil && latest.Seconds >= now.Seconds+r.TimeoutSeconds {
				err = errors.New("attempt time limit reached before spawn")
			}
			processStart = latest.Seconds
		}
	}
	if err == nil {
		childStartedAt = time.Now()
		err = command.Start()
	}
	if err != nil {
		terminal.Outcome = "PRE_START_FAILURE"
		terminal.Detail = err.Error()
	} else {
		terminal.Started = true
		// Failure here still leaves reservation + terminal evidence; the process is
		// owned and stopped before returning. A crash leaves an ambiguous reservation.
		markerErr := writeNewJSON(filepath.Join(directory, "started.json"), startedEvidence{command.Process.Pid, r.Slot})
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		if markerErr != nil {
			cancel()
		}
		reviewTimer := time.NewTimer(time.Duration(max(0, state.ReviewAt-now.Seconds) * float64(time.Second)))
		defer reviewTimer.Stop()
		reviewChannel := reviewTimer.C
		resourceTimer := time.NewTicker(time.Second)
		defer resourceTimer.Stop()
		var resourceErr error
		finished := false
		for !finished {
			select {
			case err = <-done:
				finished = true
			case <-reviewChannel:
				fmt.Fprintln(os.Stderr, "CNC: 30-minute review due; no next start until review is recorded")
				reviewChannel = nil
			case <-resourceTimer.C:
				for _, guard := range guards {
					if guardErr := guard(); guardErr != nil {
						resourceErr = guardErr
						cancel()
						break
					}
				}
			case <-childContext.Done():
				if r.Ownership != nil {
					_ = command.Process.Signal(syscall.SIGTERM)
				} else {
					_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
				}
				timer := time.NewTimer(200 * time.Millisecond)
				waited := false
				select {
				case err = <-done:
					waited = true
					timer.Stop()
				case <-timer.C:
				}
				// Kill the entire owned group, including descendants whose leader exited.
				if r.Ownership == nil {
					_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
				}
				if r.Ownership != nil {
					if killErr := killWorkerFD(int(ownedGroup.Fd())); killErr != nil {
						resourceErr = killErr
					}
				}
				if !waited {
					err = <-done
				}
				finished = true
			}
		}
		processElapsed = time.Since(childStartedAt).Nanoseconds()
		if reading, clockErr := clock(); clockErr == nil {
			processEnd = reading.Seconds
		} else {
			resourceErr = clockErr
		}
		terminal.ExitCode = command.ProcessState.ExitCode()
		if status, ok := command.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			terminal.Signal = int(status.Signal())
		}
		switch {
		case processEnd >= state.Deadline || processEnd >= terminal.Start+r.TimeoutSeconds:
			terminal.Outcome = "TIME_LIMIT"
		case resourceErr != nil:
			terminal.Outcome = "HARNESS_FAILURE"
			terminal.Detail = resourceErr.Error()
		case markerErr != nil:
			terminal.Outcome = "HARNESS_FAILURE"
			terminal.Detail = markerErr.Error()
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			terminal.Outcome = "TIME_LIMIT"
		case ctx.Err() != nil:
			terminal.Outcome = "CANCELLED"
		case childContext.Err() != nil:
			terminal.Outcome = "TIME_LIMIT"
		case writer.err != nil:
			terminal.Outcome = "HARNESS_FAILURE"
			terminal.Detail = writer.err.Error()
		case err != nil:
			terminal.Outcome = "CHILD_FAILURE"
		default:
			terminal.Outcome = "CHILD_SUCCESS"
		}
		if beforeStart != nil && childContext.Err() == nil && artifactFailureCanClassify(terminal.Outcome) {
			if inputErr := beforeStart(); inputErr != nil {
				terminal.Outcome = "HARNESS_FAILURE"
				terminal.Detail = "post-child input verification: " + inputErr.Error()
			}
		}
	}
	for _, stream := range []*os.File{file, stdout, stderr} {
		if err = stream.Sync(); err != nil {
			terminal.Outcome = "HARNESS_FAILURE"
			terminal.Detail = err.Error()
		}
		if err = stream.Close(); err != nil {
			terminal.Outcome = "HARNESS_FAILURE"
			terminal.Detail = err.Error()
		}
	}
	terminal.StdoutSHA256, err = fileDigest(filepath.Join(directory, "stdout.log"))
	if err != nil {
		return terminal, err
	}
	terminal.StderrSHA256, err = fileDigest(filepath.Join(directory, "stderr.log"))
	if err != nil {
		return terminal, err
	}
	if err = writeNewJSON(filepath.Join(directory, "process.json"), processEvidence{Started: terminal.Started, ExitCode: terminal.ExitCode, Signal: terminal.Signal, Outcome: terminal.Outcome, Detail: terminal.Detail, Start: processStart, End: processEnd, ElapsedNanoseconds: processElapsed}); err != nil {
		return terminal, err
	}
	terminal.LogSHA256, err = fileDigest(logPath)
	if err != nil {
		return terminal, err
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		return terminal, err
	}
	terminal.Selection = strictdiagnostic.SelectRootReportV2(log, r.ReportRoot)
	if r.Outputs != nil && terminal.Outcome == "CHILD_SUCCESS" {
		if err := retainOutputs(r.Directory, directory, *r.Outputs); err != nil {
			terminal.Outcome = "HARNESS_FAILURE"
			terminal.Detail = err.Error()
		}
	}
	if terminal.Selection.Outcome == strictdiagnostic.OutcomeCaptured {
		selected := terminal.Selection.SelectedAbsolute
		bytes, readErr := os.ReadFile(selected)
		if readErr == nil {
			readErr = writeNew(filepath.Join(directory, "configuration-cache-report.html"), bytes)
		}
		if readErr != nil {
			if artifactFailureCanClassify(terminal.Outcome) {
				terminal.Outcome = "HARNESS_FAILURE"
				terminal.Detail = readErr.Error()
			}
		} else {
			digest, hashErr := fileDigest(filepath.Join(directory, "configuration-cache-report.html"))
			if hashErr != nil {
				return terminal, hashErr
			}
			terminal.Artifacts = append(terminal.Artifacts, artifact{"configuration-cache-report.html", digest})
		}
	}
	// Report selection never overrides an earlier timeout/cancel/harness error.
	if strings.HasPrefix(r.Slot, "D") && (terminal.Outcome == "CHILD_SUCCESS" || terminal.Outcome == "CHILD_FAILURE") {
		terminal.Outcome = string(terminal.Selection.Outcome)
	}
	for _, name := range r.RequiredArtifacts {
		if !filepath.IsLocal(name) {
			return terminal, errors.New("unsafe artifact name")
		}
		path := filepath.Join(directory, name)
		digest, hashErr := fileDigest(path)
		if hashErr != nil {
			if artifactFailureCanClassify(terminal.Outcome) {
				terminal.Outcome = "HARNESS_FAILURE"
				terminal.Detail = "required artifact: " + name
			}
			continue
		}
		terminal.Artifacts = append(terminal.Artifacts, artifact{name, digest})
	}
	for _, name := range []string{"inputs-before.json", "inputs-after.json"} {
		path := filepath.Join(directory, name)
		if _, statErr := os.Lstat(path); os.IsNotExist(statErr) {
			continue
		}
		digest, hashErr := fileDigest(path)
		if hashErr != nil {
			return terminal, hashErr
		}
		terminal.Artifacts = append(terminal.Artifacts, artifact{name, digest})
	}
	now, err = clock()
	if err != nil {
		return terminal, err
	}
	terminal.End = now.Seconds
	if terminal.Started && terminal.End >= state.Deadline {
		terminal.Outcome = "TIME_LIMIT"
	}
	// Keep the original absolute URI in the immutable child log/request; the
	// portable selection summary itself must not retain an absolute host path.
	terminal.Selection.SelectedAbsolute = ""
	terminal.Selection.References = nil
	if err = writeNewJSON(filepath.Join(directory, "result.json"), terminal); err != nil {
		return terminal, fmt.Errorf("terminal publication failed; reservation/log retained: %w", err)
	}
	return terminal, nil
}

func validSlot(slot string) bool {
	if len(slot) != 3 {
		return false
	}
	var n int
	if _, err := fmt.Sscanf(slot[1:], "%02d", &n); err != nil {
		return false
	}
	maximum := map[byte]int{'P': 2, 'D': 2, 'M': 2, 'F': 24, 'C': 10, 'V': 18, 'R': 2}[slot[0]]
	return n >= 1 && n <= maximum && fmt.Sprintf("%c%02d", slot[0], n) == slot
}

func inspect(root string, state campaign) ([]result, error) {
	if err := validateState(state); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return nil, err
	}
	if len(entries) > state.MaximumStarts {
		return nil, errors.New("attempt budget exceeded")
	}
	results := []result{}
	for _, entry := range entries {
		if !entry.IsDir() || !validSlot(entry.Name()) {
			return nil, errors.New("unknown attempt entry")
		}
		directory := filepath.Join(root, "attempts", entry.Name())
		var reserved reservation
		if err = readJSON(filepath.Join(directory, "reservation.json"), &reserved); err != nil {
			return nil, fmt.Errorf("ambiguous reservation %s: %w", entry.Name(), err)
		}
		if reserved.Slot != entry.Name() || reserved.PackageSHA256 != state.PackageSHA256 || reserved.BootSeconds < state.Started || reserved.BootSeconds >= state.Deadline {
			return nil, errors.New("reservation binding drift")
		}
		var terminal result
		if err = readJSON(filepath.Join(directory, "result.json"), &terminal); err != nil {
			return nil, fmt.Errorf("unreconciled attempt %s: %w", entry.Name(), err)
		}
		if terminal.Schema != "buildopt.cnc/attempt/v1" || terminal.Slot != reserved.Slot || terminal.Start != reserved.BootSeconds || terminal.End < terminal.Start {
			return nil, errors.New("terminal identity drift")
		}
		requestHash, err := fileDigest(filepath.Join(directory, "request.json"))
		if err != nil || requestHash != reserved.RequestSHA256 {
			return nil, errors.New("request drift")
		}
		var requested request
		if err = readJSON(filepath.Join(directory, "request.json"), &requested); err != nil {
			return nil, err
		}
		var observed processEvidence
		if err = readJSON(filepath.Join(directory, "process.json"), &observed); err != nil {
			return nil, err
		}
		if requested.Slot != terminal.Slot || observed.Started != terminal.Started || observed.ExitCode != terminal.ExitCode || observed.Signal != terminal.Signal {
			return nil, errors.New("process identity drift")
		}
		if observed.Outcome != "CHILD_SUCCESS" && observed.Outcome != "CHILD_FAILURE" && observed.Outcome != "PRE_START_FAILURE" && observed.Outcome != "HARNESS_FAILURE" && observed.Outcome != "CANCELLED" && observed.Outcome != "TIME_LIMIT" {
			return nil, errors.New("unknown process outcome")
		}
		if observed.Outcome == "CHILD_SUCCESS" && (!observed.Started || observed.ExitCode != 0) {
			return nil, errors.New("impossible child success")
		}
		if observed.Signal < 0 || observed.Signal > 64 || (observed.Signal != 0 && (!observed.Started || observed.ExitCode != -1)) {
			return nil, errors.New("impossible process signal")
		}
		if observed.Started && (observed.Start < reserved.BootSeconds || observed.End < observed.Start || observed.End > terminal.End || observed.ElapsedNanoseconds <= 0) {
			return nil, errors.New("child timing identity drift")
		}
		digest, err := fileDigest(filepath.Join(directory, "child.log"))
		if err != nil || digest != terminal.LogSHA256 {
			return nil, errors.New("log drift")
		}
		for name, expected := range map[string]string{"stdout.log": terminal.StdoutSHA256, "stderr.log": terminal.StderrSHA256} {
			got, err := fileDigest(filepath.Join(directory, name))
			if err != nil || got != expected {
				return nil, errors.New("stream log drift")
			}
		}
		log, err := os.ReadFile(filepath.Join(directory, "child.log"))
		if err != nil {
			return nil, err
		}
		selected := strictdiagnostic.SelectRootReportV2(log, requested.ReportRoot)
		// CNC deliberately omits the selector's absolute-path convenience field.
		selected.SelectedAbsolute = ""
		selected.References = nil
		if !reflect.DeepEqual(selected, terminal.Selection) {
			return nil, errors.New("report selection drift")
		}
		expectedOutcome := observed.Outcome
		if requested.Outputs != nil {
			if err := checkOutputs(directory, *requested.Outputs); err != nil && artifactFailureCanClassify(expectedOutcome) {
				expectedOutcome = "HARNESS_FAILURE"
			}
		}
		if strings.HasPrefix(terminal.Slot, "D") && (expectedOutcome == "CHILD_SUCCESS" || expectedOutcome == "CHILD_FAILURE") {
			expectedOutcome = string(selected.Outcome)
		}
		expectedArtifacts := []artifact{}
		if selected.Outcome == strictdiagnostic.OutcomeCaptured {
			path := filepath.Join(directory, "configuration-cache-report.html")
			hash, err := fileDigest(path)
			if err != nil {
				if artifactFailureCanClassify(expectedOutcome) {
					expectedOutcome = "HARNESS_FAILURE"
				}
			} else {
				original, err := fileDigest(filepath.Join(requested.ReportRoot, selected.SelectedRelative))
				if err != nil || original != hash {
					return nil, errors.New("retained report differs from selected root report")
				}
				expectedArtifacts = append(expectedArtifacts, artifact{"configuration-cache-report.html", hash})
			}
		}
		for _, name := range requested.RequiredArtifacts {
			path := filepath.Join(directory, name)
			if err = contained(directory, path); err != nil {
				return nil, err
			}
			hash, err := fileDigest(path)
			if err != nil {
				if artifactFailureCanClassify(expectedOutcome) {
					expectedOutcome = "HARNESS_FAILURE"
				}
				continue
			}
			expectedArtifacts = append(expectedArtifacts, artifact{name, hash})
		}
		for _, name := range []string{"inputs-before.json", "inputs-after.json"} {
			path := filepath.Join(directory, name)
			if _, statErr := os.Lstat(path); os.IsNotExist(statErr) {
				continue
			}
			hash, err := fileDigest(path)
			if err != nil {
				return nil, err
			}
			expectedArtifacts = append(expectedArtifacts, artifact{name, hash})
		}
		if terminal.Started && terminal.End >= state.Deadline {
			expectedOutcome = "TIME_LIMIT"
		}
		if expectedOutcome != terminal.Outcome || !reflect.DeepEqual(expectedArtifacts, terminal.Artifacts) {
			return nil, errors.New("terminal summary disagrees with raw evidence")
		}
		for _, file := range terminal.Artifacts {
			path := filepath.Join(directory, file.Path)
			if err = contained(directory, path); err != nil {
				return nil, err
			}
			digest, err = fileDigest(path)
			if err != nil || digest != file.SHA256 {
				return nil, errors.New("artifact drift")
			}
		}
		if terminal.Started {
			var marker startedEvidence
			if err = readJSON(filepath.Join(directory, "started.json"), &marker); err != nil || marker.PID <= 0 || marker.Slot != terminal.Slot {
				return nil, errors.New("started child has no valid marker")
			}
		}
		results = append(results, terminal)
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i].Slot, results[j].Slot
		order := "PDMFCVR"
		if a[0] != b[0] {
			return strings.IndexByte(order, a[0]) < strings.IndexByte(order, b[0])
		}
		return a < b
	})
	return results, nil
}
