//go:build windows

package launcher

import "io"

func nativeGradleProcessReplacementSupported(
	stdin io.Reader,
	stdout, stderr io.Writer,
) bool {
	return false
}

func replaceWithNativeGradleProcess(
	childArgs []string,
	environmentOverrides map[string]string,
) error {
	panic("native Gradle process replacement is unavailable on Windows")
}
