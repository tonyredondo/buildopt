//go:build windows

package stickyobservation

import "os"

// Windows does not expose POSIX permission bits through os.FileInfo. The
// private boundary here therefore checks native object type and rejects
// symlink/reparse-shaped entries; the parent lives under the user's private
// cache root and Windows ACLs remain the platform access-control boundary.
func privateObservationDirectoryInfo(info os.FileInfo) bool {
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func privateObservationFileInfo(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}
