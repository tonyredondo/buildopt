//go:build windows

package sharedcache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tonyredondo/buildopt/internal/filelock"
	"github.com/tonyredondo/buildopt/internal/platformfs"
	"golang.org/x/sys/windows"
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
	return nil
}

func openPrivateLock(path string) (*os.File, error) {
	return openPrivateFile(path, os.O_CREATE|os.O_RDWR, 0o600)
}

func preparePrivateDatabase(path string) error {
	file, err := openPrivateFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

func openPrivateBlob(path string) (*os.File, error) {
	return openPrivateFile(path, os.O_RDONLY, 0)
}

func openPrivateFile(path string, flags int, mode os.FileMode) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("private file path is a reparse point or not regular")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(path, flags, mode)
	if err != nil {
		return nil, err
	}
	if err := validatePrivateRegularFile(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func validatePrivateRegularFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("file must be a private regular file")
	}
	return nil
}

func validatePrivateSidecar(path string) error {
	file, err := openPrivateFile(path, os.O_RDONLY, 0)
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
	volume := strings.ToUpper(filepath.VolumeName(paths[0]))
	if volume == "" || strings.HasPrefix(volume, `\\`) {
		return errors.New("storage must use a local Windows volume")
	}
	for _, path := range paths {
		if strings.ToUpper(filepath.VolumeName(path)) != volume {
			return fmt.Errorf("%s is on a different volume", path)
		}
		if err := platformfs.ValidateNoLinks(path); err != nil {
			return fmt.Errorf("%s contains a reparse point", path)
		}
	}
	return nil
}

func storageDiskCapacity(path string) (uint64, uint64, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var available, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &available, &total, &free); err != nil {
		return 0, 0, err
	}
	return total, available, nil
}

func isSupportedLocalFilesystemType(int64) bool { return true }

// Windows FlushFileBuffers does not accept directory handles. Atomic file
// publication and SQLite FULL synchronous mode remain the durability boundary.
func syncDirectory(string) error { return nil }
