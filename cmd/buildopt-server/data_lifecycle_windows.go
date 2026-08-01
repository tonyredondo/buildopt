//go:build windows

package main

import (
	"errors"
	"os"
)

func openLifecycleTokenKey(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("token key path is a reparse point or not regular")
	}
	return os.Open(path)
}

func privateLifecycleTokenKeyInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func privateAuthoritySourceInfo(info os.FileInfo) bool {
	return privateLifecycleTokenKeyInfo(info)
}
