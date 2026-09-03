//go:build !windows

package wcncpobserve

import (
	"os/exec"
	"syscall"
)

func exitErrorSignal(exitError *exec.ExitError) (int, bool) {
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0, false
	}
	return int(status.Signal()), true
}
