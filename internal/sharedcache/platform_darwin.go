//go:build darwin

package sharedcache

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/tonyredondo/buildopt/internal/filelock"
	"golang.org/x/sys/unix"
)

func preparePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a private directory", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("%s is not owned by the current user", path)
	}
	if info.Mode().Perm() != 0o700 {
		return fmt.Errorf("%s must have mode 0700", path)
	}
	return nil
}

func openPrivateLock(path string) (*os.File, error) {
	return openPrivateFile(path, unix.O_CREAT|unix.O_RDWR, 0o600)
}

func preparePrivateDatabase(path string) error {
	file, err := openPrivateFile(path, unix.O_CREAT|unix.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

func openPrivateBlob(path string) (*os.File, error) {
	return openPrivateFile(path, unix.O_RDONLY, 0)
}

func openPrivateFile(path string, flags int, mode uint32) (*os.File, error) {
	descriptor, err := unix.Open(path, flags|unix.O_CLOEXEC|unix.O_NOFOLLOW, mode)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file == nil {
		_ = unix.Close(descriptor)
		return nil, errors.New("create private file handle")
	}
	if err := validatePrivateRegularFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return file, nil
}

func validatePrivateRegularFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 || stat.Nlink != 1 {
		return errors.New("file must be private, regular, and singly linked")
	}
	if info.Mode().Perm() != 0o600 {
		return errors.New("file must have mode 0600")
	}
	return nil
}

func validatePrivateSidecar(path string) error {
	file, err := openPrivateFile(path, unix.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return file.Close()
}

func acquireExclusiveLock(file *os.File) error { return filelock.Try(file, filelock.Exclusive) }
func releaseExclusiveLock(file *os.File) error { return filelock.Unlock(file) }
func isLockBusy(err error) bool                { return errors.Is(err, filelock.ErrBusy) }

func validateLocalStorageFilesystem(paths ...string) error {
	if len(paths) == 0 {
		return errors.New("no storage paths")
	}
	var device int32
	for index, path := range paths {
		var filesystem unix.Statfs_t
		if err := unix.Statfs(path, &filesystem); err != nil {
			return err
		}
		if filesystem.Flags&unix.MNT_LOCAL == 0 {
			return fmt.Errorf("%s is not on a local filesystem", path)
		}
		var stat unix.Stat_t
		if err := unix.Stat(path, &stat); err != nil {
			return err
		}
		if index == 0 {
			device = stat.Dev
		} else if stat.Dev != device {
			return fmt.Errorf("%s is on a different filesystem", path)
		}
	}
	return nil
}

func storageDiskCapacity(path string) (uint64, uint64, error) {
	var filesystem unix.Statfs_t
	if err := unix.Statfs(path, &filesystem); err != nil {
		return 0, 0, err
	}
	blockSize := uint64(filesystem.Bsize)
	return filesystem.Blocks * blockSize, filesystem.Bavail * blockSize, nil
}

func isSupportedLocalFilesystemType(int64) bool { return true }

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
