//go:build windows

package launcher

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func isGatewayDiskPressure(err error) bool {
	return errors.Is(err, windows.ERROR_DISK_FULL) ||
		errors.Is(err, windows.ERROR_HANDLE_DISK_FULL) ||
		errors.Is(err, syscall.ENOSPC)
}

func validateGatewaySpoolFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("%w: inspect file: %v", errGatewaySpoolUnavailable, err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("local gateway spool file is not a regular file")
	}
	return nil
}

func openGatewaySpoolFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("local gateway spool path is not a regular file")
	}
	return os.Open(path)
}

// Windows does not allow deleting a file while it is open with the sharing
// flags used by os.CreateTemp. Retain the invocation-private name until the
// verified payload handle closes, then remove it immediately.
func detachGatewaySpoolFile(path string) (string, error) { return path, nil }

func syncGatewaySpoolDirectory(string) error {
	// Windows does not expose directory fsync through os.File. Every payload is
	// flushed before use and remains invocation-scoped rather than durable state.
	return nil
}
