//go:build !windows

// Package platformfs implements filesystem guarantees whose native mechanism
// differs between Unix and Windows.
package platformfs

import (
	"errors"
	"path/filepath"
)

// ValidateNoLinks rejects a path whose resolution traverses a symbolic link.
func ValidateNoLinks(path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if resolved != path {
		return errors.New("path traverses a symbolic link")
	}
	return nil
}
