//go:build windows

package adaptivefragment

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func openPrivateLocalLock(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("adaptive local state lock is unsafe")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
}

func privateLocalDirectoryInfo(info os.FileInfo) bool {
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func privateLocalFileInfo(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func replaceLocalStateFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(
		from,
		to,
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
	)
}

// Windows cannot flush directory handles through os.File. File contents are
// flushed before atomic replacement, which is the native durability boundary.
func syncLocalStateDirectory(string) error { return nil }
