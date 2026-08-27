//go:build !windows

package stickyobservation

import "os"

func privateObservationDirectoryInfo(info os.FileInfo) bool {
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o077 == 0
}

func privateObservationFileInfo(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o077 == 0
}
