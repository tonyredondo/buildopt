//go:build linux && amd64

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestAcquisitionUsesContentsNotCheckoutName(t *testing.T) {
	m := fixtureManifest(t, 2)
	layer := filepath.Join(filepath.Dir(m.RunRoot), "buildopt", "modules-2")
	mustWrite(t, filepath.Join(layer, "files-2.1", "example", "artifact.jar"), []byte("downloaded dependency"))
	m.Acquisition = []Layer{{Kind: "dependencies", Source: bound(t, layer), Destination: "gradle/caches/modules-2"}}
	if err := validateManifest(m); err != nil {
		t.Fatalf("legitimate acquisition inside the BuildOpt checkout: %v", err)
	}
	mustWrite(t, filepath.Join(layer, "executionHistory", "state.bin"), []byte("task state"))
	m.Acquisition[0].Source = bound(t, layer)
	if err := validateManifest(m); err == nil {
		t.Fatal("accepted task history hidden inside acquisition layer")
	}
}

func TestLiveDiskMeasurementToleratesNativeTempDeletion(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "stable"), []byte("retained"))
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		for round := 0; ; round++ {
			select {
			case <-done:
				return
			default:
			}
			dir := filepath.Join(root, "native-tmp")
			_ = os.Mkdir(dir, 0700)
			for index := 0; index < 30; index++ {
				_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("proj-%d-%d.tmp", round, index)), []byte("temporary"), 0600)
			}
			_ = os.RemoveAll(dir)
		}
	}()
	defer func() { close(done); <-stopped }()
	for iteration := 0; iteration < 200; iteration++ {
		bytes, err := treeBytes(root)
		if err != nil || bytes < int64(len("retained")) {
			t.Fatalf("live native temporary files cancel measurement: %d %v", bytes, err)
		}
	}
	if _, err := treeBytes(filepath.Join(root, "missing-root")); err == nil {
		t.Fatal("missing owned root was silently accepted")
	}
}

func TestExactCopyPreservesModesAndIndependentBytes(t *testing.T) {
	root := t.TempDir()
	source, destination := filepath.Join(root, "source"), filepath.Join(root, "copy")
	mustWrite(t, filepath.Join(source, "readonly", "license"), []byte("retained bytes"))
	if err := os.Chmod(filepath.Join(source, "readonly", "license"), 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(source, "readonly"), 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(source, "readonly"), 0755) })
	previousMask := unix.Umask(0027)
	defer unix.Umask(previousMask)
	if err := copyTree(source, destination); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(destination, "readonly"), 0755) })
	want, err := inventory(source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := inventory(destination)
	if err != nil || !equalJSON(want, got) {
		t.Fatalf("copy changes content/type/mode: %v\nwant %+v\ngot %+v", err, want, got)
	}
	if err := os.WriteFile(filepath.Join(destination, "readonly", "license"), []byte("private change"), 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(source, "readonly", "license"))
	if err != nil || string(raw) != "retained bytes" {
		t.Fatal("copy shares a mutable inode with its input")
	}
}
