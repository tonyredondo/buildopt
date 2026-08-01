//go:build windows

// Package platformfs implements filesystem guarantees whose native mechanism
// differs between Unix and Windows.
package platformfs

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// ValidateNoLinks rejects any reparse point in path without comparing its
// spelling with the canonical Windows path. That comparison is invalid when a
// runner or user profile uses an 8.3 component such as RUNNER~1.
func ValidateNoLinks(path string) error {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		pointer, err := windows.UTF16PtrFromString(current)
		if err != nil {
			return err
		}
		attributes, err := windows.GetFileAttributes(pointer)
		if err != nil {
			return err
		}
		if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return fmt.Errorf("%s is a reparse point", current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}
