//go:build windows

package edgecache

import (
	"errors"
	"os"

	"github.com/tonyredondo/buildopt/internal/filelock"
)

func acquireStoreLock(file *os.File) error {
	return filelock.Try(file, filelock.Exclusive)
}

func releaseStoreLock(file *os.File) error { return filelock.Unlock(file) }

func openNoFollow(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("Edge blob path is a reparse point or not regular")
	}
	return os.Open(path)
}

// Windows FlushFileBuffers does not accept directory handles.
func syncDirectory(string) error { return nil }
