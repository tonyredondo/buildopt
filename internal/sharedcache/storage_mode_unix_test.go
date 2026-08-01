//go:build !windows

package sharedcache

import "os"

func testPrivateMode(info os.FileInfo, directory bool, mode os.FileMode) bool {
	return info.IsDir() == directory && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm() == mode
}
