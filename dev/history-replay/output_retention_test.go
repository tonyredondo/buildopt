//go:build linux && amd64

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputRetentionReuseAndNativeIndependence(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "native")
	if err := os.WriteFile(source, []byte("native bytes"), 0640); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(root, "objects")
	files := []fileCopy{}
	for i := 0; i < 64; i++ {
		files = append(files, fileCopy{source, filepath.Join(root, fmt.Sprintf("retained-%d", i))})
	}
	if err := retainOutputFiles(files, store); err != nil {
		t.Fatal(err)
	}
	native, _ := os.Stat(source)
	first, _ := os.Stat(files[0].Destination)
	if os.SameFile(native, first) {
		t.Fatal("native shares evidence inode")
	}
	for _, f := range files {
		info, err := os.Stat(f.Destination)
		if err != nil || !os.SameFile(first, info) || info.Mode().Perm() != 0640 {
			t.Fatalf("reuse/mode mismatch: %v", err)
		}
		data, err := os.ReadFile(f.Destination)
		if err != nil || string(data) != "native bytes" {
			t.Fatalf("bytes mismatch: %v", err)
		}
	}
	if err := os.WriteFile(source, []byte("new native!!"), 0640); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(files[0].Destination)
	if string(data) != "native bytes" {
		t.Fatal("native mutation damaged evidence")
	}
	if err := retainOutputFiles([]fileCopy{{source, filepath.Join(root, "next")}}, store); err != nil {
		t.Fatal(err)
	}
	// Mode is part of the object key, even for identical bytes.
	if err := os.Chmod(source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := retainOutputFiles([]fileCopy{{source, filepath.Join(root, "other-mode")}}, store); err != nil {
		t.Fatal(err)
	}
	x, _ := os.Stat(filepath.Join(root, "next"))
	y, _ := os.Stat(filepath.Join(root, "other-mode"))
	if os.SameFile(x, y) || y.Mode().Perm() != 0600 {
		t.Fatal("mixed modes share an object")
	}
}

func TestOutputRetentionRejectsTamperingAndOverwrite(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "native")
	store := filepath.Join(root, "objects")
	retained := filepath.Join(root, "retained")
	if err := os.WriteFile(source, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := retainOutputFiles([]fileCopy{{source, retained}}, store); err != nil {
		t.Fatal(err)
	}
	if err := retainOutputFiles([]fileCopy{{source, retained}}, store); err == nil {
		t.Fatal("overwrote existing destination")
	}
	if err := os.WriteFile(retained, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	next := filepath.Join(root, "next")
	if err := retainOutputFiles([]fileCopy{{source, next}}, store); err == nil {
		t.Fatal("accepted corrupt retained object")
	}
	if _, err := os.Lstat(next); !os.IsNotExist(err) {
		t.Fatal("published rejected evidence")
	}
	data, _ := os.ReadFile(source)
	if string(data) != "original" {
		t.Fatal("evidence tampering changed native")
	}
	if err := retainOutputFiles([]fileCopy{{filepath.Join(root, "missing"), next}}, store); err == nil {
		t.Fatal("accepted missing source")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	if err := retainOutputFiles([]fileCopy{{link, next}}, store); err == nil {
		t.Fatal("accepted symlink source")
	}
}
