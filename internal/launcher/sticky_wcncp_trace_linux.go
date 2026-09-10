//go:build linux

package launcher

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// This opt-in is diagnostic evidence for the owned EIC correctness campaign.
// No file is opened before native execution. The disabled path reads only the
// environment; trace failures never change native streams, signals or exit.
func beginStickyWCNCPTrace() func(childExecution) {
	path := os.Getenv("BUILDOPT_EIC_NATIVE_TRACE")
	if path == "" {
		return discardStickyWCNCPTrace
	}
	var begin unix.Timespec
	if unix.ClockGettime(unix.CLOCK_BOOTTIME, &begin) != nil {
		return discardStickyWCNCPTrace
	}
	return func(execution childExecution) {
		var end unix.Timespec
		if unix.ClockGettime(unix.CLOCK_BOOTTIME, &end) != nil {
			return
		}
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(path))
		if err != nil || parent != filepath.Dir(path) {
			return
		}
		duration := int64(0)
		code := 0
		signal := 0
		if execution.started {
			duration = execution.completedAt.Sub(execution.startedAt).Nanoseconds()
		} else {
			code = -1
		}
		var exit *exec.ExitError
		if errors.As(execution.err, &exit) {
			code = exit.ExitCode()
			if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				signal = int(status.Signal())
			}
		} else if execution.err != nil {
			code = -1
		}
		group, err := os.ReadFile("/proc/self/cgroup")
		if err != nil {
			return
		}
		record := struct {
			Schema     string `json:"schemaVersion"`
			Started    bool   `json:"started"`
			PID        int    `json:"pid"`
			Start      int64  `json:"startBootNanoseconds"`
			End        int64  `json:"endBootNanoseconds"`
			Duration   int64  `json:"nativeDurationNanoseconds"`
			Exit       int    `json:"exitCode"`
			Signal     int    `json:"signal"`
			Cgroup     string `json:"cgroup"`
			Diagnostic bool   `json:"diagnosticOnly"`
		}{"buildopt.eic/native-supervision/v1", execution.started, execution.pid, begin.Nano(), end.Nano(), duration, code, signal, strings.TrimSpace(string(group)), true}
		data, err := json.Marshal(record)
		if err != nil {
			return
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			return
		}
		defer file.Close()
		if _, err = file.Write(append(data, '\n')); err == nil {
			_ = file.Sync()
		}
	}
}

func discardStickyWCNCPTrace(childExecution) {}
