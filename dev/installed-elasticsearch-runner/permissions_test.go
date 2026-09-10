package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCopyTreePreservesPermissionsAcrossUmask(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--eic-copy-permissions" {
			syscall.Umask(0077)
			if err := copyTree(os.Args[i+1], os.Args[i+2]); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	root := t.TempDir()
	source, target := filepath.Join(root, "source"), filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(source, "readonly"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"group-writable", "readonly/member"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("retained bytes"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(source, name), 0664); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(source, 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(source, "readonly"), 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(filepath.Join(source, "readonly"), 0700)
		os.Chmod(filepath.Join(target, "readonly"), 0700)
	})
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := exec.Command(exe, "-test.run=^TestCopyTreePreservesPermissionsAcrossUmask$", "--", "--eic-copy-permissions", source, target).CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	for path, mode := range map[string]os.FileMode{".": 0775, "readonly": 0555, "group-writable": 0664, "readonly/member": 0664} {
		info, err := os.Stat(filepath.Join(target, path))
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("%s mode: %v %v", path, info, err)
		}
	}
}
