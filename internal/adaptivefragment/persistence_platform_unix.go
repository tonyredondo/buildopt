//go:build !windows

package adaptivefragment

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func openPrivateLocalLock(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	info, err := file.Stat()
	if err != nil || !privateLocalFileInfo(info) {
		file.Close()
		return nil, errors.New("adaptive local state lock is not private")
	}
	return file, nil
}

func privateLocalDirectoryInfo(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.IsDir() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o700
}

func privateLocalFileInfo(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.Mode().IsRegular() &&
		info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o600
}

func replaceLocalStateFile(source, destination string) error {
	return os.Rename(source, destination)
}

func syncLocalStateDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
