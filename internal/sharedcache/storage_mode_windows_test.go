//go:build windows

package sharedcache

import "os"

func testPrivateMode(info os.FileInfo, directory bool, _ os.FileMode) bool {
	// Go reports synthesized 0666/0777 modes on Windows. Production validates
	// the native file type and rejects reparse traversal instead.
	return info.IsDir() == directory && info.Mode()&os.ModeSymlink == 0
}
