//go:build windows

package wcncpobserve

import "os/exec"

func exitErrorSignal(_ *exec.ExitError) (int, bool) { return 0, false }
