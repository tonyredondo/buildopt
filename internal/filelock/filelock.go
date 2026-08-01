// Package filelock provides process-scoped advisory locks with the same
// non-blocking contract on Unix and Windows.
package filelock

import (
	"errors"
	"os"
)

// ErrBusy reports that another process owns an incompatible lock.
var ErrBusy = errors.New("file lock is busy")

// Mode selects shared or exclusive ownership.
type Mode uint8

const (
	Shared Mode = iota
	Exclusive
)

// Try acquires a non-blocking advisory lock over file.
func Try(file *os.File, mode Mode) error {
	if file == nil {
		return errors.New("file lock is nil")
	}
	return try(file, mode)
}

// Unlock releases a lock previously acquired with Try.
func Unlock(file *os.File) error {
	if file == nil {
		return nil
	}
	return unlock(file)
}
