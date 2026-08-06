//go:build windows

package launcher

import "os"

// Windows privacy is enforced by the path's absence of reparse points and the
// current user's access token, not by POSIX permission bits.
func impactHotStateDirectoryPrivate(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}
