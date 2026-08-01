//go:build windows

package localauthority

import (
	"errors"
	"os"
)

func openPrivateDataFile(path string, flags int, mode os.FileMode) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("local authority path is a reparse point or not regular")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return os.OpenFile(path, flags, mode)
}

func privateDirectoryInfo(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func privateFileInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func syncPrivateDirectory(string) error { return nil }
