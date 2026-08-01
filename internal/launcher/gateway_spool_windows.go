//go:build windows

package launcher

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func isGatewayDiskPressure(err error) bool {
	return errors.Is(err, windows.ERROR_DISK_FULL) || errors.Is(err, windows.ERROR_HANDLE_DISK_FULL)
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

func syncGatewaySpoolDirectory(string) error {
	// Windows does not expose directory fsync through os.File. Every payload is
	// flushed before use and remains invocation-scoped rather than durable state.
	return nil
}
