//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveDiskBytesCountFilesAndDoNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "inside")
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inside, "file"), []byte("five!"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if size, err := treeBytes(root); err != nil || size != 5 {
		t.Fatalf("live size follows a symlink or loses bytes: %d, %v", size, err)
	}
}

func TestLiveDiskObservationCancelsWithinLargeDirectory(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < 8192; index++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%06d", index)), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	stop := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := treeBytesUntil(root, stop)
		result <- err
	}()
	// The measured native owner exposed a multi-second completion delay.
	// This component proof checks cancellation inside one large directory;
	// actual owner process/driver timing is qualified separately.
	time.Sleep(time.Millisecond)
	close(stop)
	select {
	case err := <-result:
		if !errors.Is(err, errDiskObservationStopped) {
			t.Fatalf("live scan ignored cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled resource sample did not return")
	}
	if size, err := treeBytes(root); err != nil || size != 8192 {
		t.Fatalf("complete postflight lost files: %d, %v", size, err)
	}
}

func TestLiveDiskLogicalBytesAndExactCeiling(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("five!"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(file, filepath.Join(root, "hardlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(file, filepath.Join(root, "symlink")); err != nil {
		t.Fatal(err)
	}
	sparse := filepath.Join(root, "sparse")
	if err := os.WriteFile(sparse, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(sparse, 1<<20); err != nil {
		t.Fatal(err)
	}
	want := int64(1<<20) + 10
	if got, err := treeBytes(root); err != nil || got != want {
		t.Fatalf("logical bytes: %d %v", got, err)
	}
	if err := diskGuard(root, Limits{MaxBytes: want}); err != nil {
		t.Fatal(err)
	}
	if err := diskGuard(root, Limits{MaxBytes: want - 1}); err == nil {
		t.Fatal("size ceiling bypassed")
	}
	if _, err := treeBytes(filepath.Join(root, "absent")); !os.IsNotExist(err) {
		t.Fatalf("root loss hidden: %v", err)
	}
}
