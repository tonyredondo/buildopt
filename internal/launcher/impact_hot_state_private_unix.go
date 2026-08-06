//go:build !windows

package launcher

import "os"

func impactHotStateDirectoryPrivate(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm()&0o077 == 0
}
