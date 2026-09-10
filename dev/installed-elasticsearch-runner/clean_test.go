package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCleanSnapshotDetectsChangedInputs(t *testing.T) {
	for _, change := range []string{"bytes", "mode", "mtime", "extra", "symlink"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "input")
			if err := os.WriteFile(path, []byte("seed"), 0600); err != nil {
				t.Fatal(err)
			}
			before, err := cleanSnapshotDigest(root)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "bytes":
				err = os.WriteFile(path, []byte("drift"), 0600)
			case "mode":
				err = os.Chmod(path, 0644)
			case "mtime":
				err = os.Chtimes(path, info.ModTime().Add(1), info.ModTime().Add(1))
			case "extra":
				err = os.WriteFile(filepath.Join(root, "output"), []byte("old"), 0600)
			case "symlink":
				err = os.Symlink("/outside", filepath.Join(root, "link"))
			}
			if err != nil {
				t.Fatal(err)
			}
			after, err := cleanSnapshotDigest(root)
			if err == nil && before == after {
				t.Fatal("changed starting state accepted")
			}
		})
	}
}

func TestCleanDiagnosticKeepsScenarioAndSlotsSeparate(t *testing.T) {
	root := t.TempDir()
	rows := cleanDiagnosticRows()
	if len(rows) != 2 {
		t.Fatal(rows)
	}
	for i, slot := range []string{"CD001", "CD002"} {
		r, err := resolvedNativeRequest(root, "/runner", rows[i])
		if err != nil {
			t.Fatal(err)
		}
		if r.Row.ID != slot || r.Row.State != "clean-workspace-populated-cache" || r.Seed == nil || r.Seed.Path != filepath.Join(root, "clean-seed.json") {
			t.Fatalf("%+v", r)
		}
		prefix := append([]string{"/usr/bin/taskset", "--cpu-list", "0-7"}, makeProtocol().Rows[0].Arguments...)
		if !reflect.DeepEqual(r.Arguments[:len(prefix)], prefix) {
			t.Fatal(r.Arguments)
		}
		if err := validateNativeSequence(root, slot); err == nil {
			t.Fatal("warm sequence admits clean slot")
		}
	}
	if err := os.Mkdir(filepath.Join(root, "attempts"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := validateCleanSequence(root, "CD001"); err != nil {
		t.Fatal(err)
	}
	if err := validateCleanSequence(root, "CD002"); err == nil {
		t.Fatal("missing predecessor accepted")
	}
	dir := filepath.Join(root, "attempts", "CD001")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := validateCleanSequence(root, "CD001"); err == nil {
		t.Fatal("replay accepted")
	}
	if err := writeJSON(filepath.Join(dir, "native-result.json"), nativeResult{RowID: "CD001", Process: processRecord{Started: true, Outcome: "SUCCESS"}, InputState: "VERIFIED"}); err != nil {
		t.Fatal(err)
	}
	if err := validateCleanSequence(root, "CD002"); err == nil {
		t.Fatal("incomplete capture accepted")
	}
}

func TestCleanSeedRequiresExactRestorationAndExternalPin(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"arms/N0", "state/N0"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "input"), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	seed, err := currentCleanSeed(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "clean-seed.json")
	if err := writeJSON(path, seed); err != nil {
		t.Fatal(err)
	}
	pin, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	binding := fileBinding{Path: path, SHA256: pin}
	if err := verifyCleanSeed(root, binding); err != nil {
		t.Fatal(err)
	}
	if err := verifyCleanSeed(root, fileBinding{Path: path}); err == nil {
		t.Fatal("missing external pin accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "state/N0/new-cache-entry"), []byte("candidate"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyCleanSeed(root, binding); err == nil {
		t.Fatal("new cache entry accepted")
	}
}
