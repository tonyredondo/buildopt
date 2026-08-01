//go:build windows

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func try(file *os.File, mode Mode) error {
	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if mode == Exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	err := windows.LockFileEx(
		windows.Handle(file.Fd()), flags, 0, 1, 0, new(windows.Overlapped),
	)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return ErrBusy
	}
	return err
}

func unlock(file *os.File) error {
	return windows.UnlockFileEx(
		windows.Handle(file.Fd()), 0, 1, 0, new(windows.Overlapped),
	)
}
