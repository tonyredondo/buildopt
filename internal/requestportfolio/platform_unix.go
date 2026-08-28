//go:build !windows

package requestportfolio

import "os"

func privateDirectoryInfo(info os.FileInfo) bool {
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o077 == 0
}

func privateFileInfo(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o077 == 0
}

func replaceFile(source, destination string) error { return os.Rename(source, destination) }

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
