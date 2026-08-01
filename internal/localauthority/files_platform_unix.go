//go:build !windows

package localauthority

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func openPrivateDataFile(path string, flags int, mode os.FileMode) (*os.File, error) {
	descriptor, err := unix.Open(path, flags|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func privateDirectoryInfo(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.IsDir() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o700
}

func privateFileInfo(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.Mode().IsRegular() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o600
}

func syncPrivateDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
