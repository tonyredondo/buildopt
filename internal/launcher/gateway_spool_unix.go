//go:build !windows

package launcher

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func isGatewayDiskPressure(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT)
}

func validateGatewaySpoolFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("%w: inspect file: %v", errGatewaySpoolUnavailable, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || stat.Nlink != 1 {
		return errors.New("local gateway spool file is not private")
	}
	return nil
}

func openGatewaySpoolFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}

func detachGatewaySpoolFile(path string) (string, error) {
	return "", os.Remove(path)
}

func syncGatewaySpoolDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
