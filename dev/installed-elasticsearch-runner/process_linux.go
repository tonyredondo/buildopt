package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type processRecord struct {
	Started  bool   `json:"started"`
	PID      int    `json:"pid"`
	StartNS  int64  `json:"startBootNanoseconds"`
	EndNS    int64  `json:"endBootNanoseconds"`
	ExitCode int    `json:"exitCode"`
	Signal   int    `json:"signal"`
	Outcome  string `json:"outcome"`
}

func bootNow() (int64, error) {
	var ts unix.Timespec
	err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &ts)
	return ts.Nano(), err
}
func bootID() (string, error) {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	return strings.TrimSpace(string(b)), err
}

type boundedWriter struct {
	writer io.Writer
	left   int
}

func (w *boundedWriter) Write(b []byte) (int, error) {
	if len(b) > w.left {
		return 0, errors.New("fixture log exceeds 1 MiB")
	}
	n, err := w.writer.Write(b)
	w.left -= n
	return n, err
}

// runProcess owns one Linux process group. Fixture descendants also use this
// supervisor, including parent-death signals. This is not a claim of containment
// for Gradle/TestKit daemons; the public adapter must prove its own ownership.
func runProcess(ctx context.Context, dir, executable string, args []string, limit time.Duration) (processRecord, error) {
	return runConfiguredProcess(ctx, dir, executable, args, dir, []string{"LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TZ=UTC", "GORACE=atexit_sleep_ms=0"}, limit)
}

func runConfiguredProcess(ctx context.Context, dir, executable string, args []string, workingDirectory string, environment []string, limit time.Duration) (processRecord, error) {
	p := processRecord{ExitCode: -1, Outcome: "NOT_STARTED"}
	if limit <= 0 || limit > 600*time.Second {
		return p, errors.New("invalid process deadline")
	}
	var err error
	p.StartNS, err = bootNow()
	if err != nil {
		return p, err
	}
	if ctx.Err() != nil {
		p.EndNS = p.StartNS
		return p, nil
	}
	stdout, err := os.OpenFile(filepath.Join(dir, "stdout.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return p, err
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(filepath.Join(dir, "stderr.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return p, err
	}
	defer stderr.Close()
	childCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	cmd := exec.CommandContext(childCtx, executable, args...)
	cmd.Dir = workingDirectory
	cmd.Env = environment
	cmd.Stdout = &boundedWriter{stdout, 1 << 20}
	cmd.Stderr = &boundedWriter{stderr, 1 << 20}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	err = cmd.Start()
	if err == nil {
		p.Started = true
		p.PID = cmd.Process.Pid
		if err = writeJSON(filepath.Join(dir, "started.json"), p); err != nil {
			cmd.Cancel()
			cmd.Wait()
			return p, err
		}
		err = cmd.Wait()
		// Any process still in this owned group is a leaked descendant. Kill it
		// even when the leader succeeded; never target unrelated process groups.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		p.ExitCode = cmd.ProcessState.ExitCode()
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			p.Signal = int(status.Signal())
		}
	}
	var clockErr error
	p.EndNS, clockErr = bootNow()
	if clockErr != nil {
		return p, clockErr
	}
	switch {
	case childCtx.Err() != nil:
		p.Outcome = "TIMEOUT"
	case !p.Started:
		p.Outcome = "START_FAILURE"
	case p.Signal != 0:
		p.Outcome = "SIGNALED"
	case p.ExitCode != 0:
		p.Outcome = "FAILURE"
	case err != nil:
		p.Outcome = "CAPTURE_FAILURE"
	default:
		p.Outcome = "SUCCESS"
	}
	if err := stdout.Sync(); err != nil {
		return p, err
	}
	if err := stderr.Sync(); err != nil {
		return p, err
	}
	return p, writeJSON(filepath.Join(dir, "process.json"), p)
}
