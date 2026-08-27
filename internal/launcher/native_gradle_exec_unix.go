//go:build !windows

package launcher

import (
	"io"
	"os"
	"os/exec"
	"syscall"
)

var nativeGradleExec = syscall.Exec

func nativeGradleProcessReplacementSupported(
	stdin io.Reader,
	stdout, stderr io.Writer,
) bool {
	return stdin == os.Stdin && stdout == os.Stdout && stderr == os.Stderr
}

// replaceWithNativeGradleProcess removes the otherwise idle BuildOpt parent
// from the uninstrumented Gradle path. The Wrapper inherits the same standard
// streams and process identity, so shell cancellation continues to reach the
// build directly while instrumented invocations retain BuildOpt's lifecycle.
func replaceWithNativeGradleProcess(
	childArgs []string,
	environmentOverrides map[string]string,
) error {
	return replaceWithNativeGradleProcessWithReserved(childArgs, environmentOverrides, nil)
}

// replaceWithNativeGradleProcessWithReserved removes the otherwise idle
// BuildOpt parent while also scrubbing invocation-specific credentials. The
// reserved names are removal-only and never become Gradle inputs.
func replaceWithNativeGradleProcessWithReserved(
	childArgs []string,
	environmentOverrides map[string]string,
	additionalReserved []string,
) error {
	path, err := exec.LookPath(childArgs[0])
	if err != nil {
		return err
	}
	return nativeGradleExec(
		path,
		childArgs,
		replaceEnvironmentWithReserved(os.Environ(), environmentOverrides, additionalReserved),
	)
}
