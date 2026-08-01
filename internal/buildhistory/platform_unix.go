//go:build !windows

package buildhistory

import "os"

func privateHistoryDirectoryInfo(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm() == 0o700
}

func privateHistoryFileInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm() == 0o600
}
