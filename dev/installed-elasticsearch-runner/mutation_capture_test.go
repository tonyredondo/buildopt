package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Optional retained-data regression: performs no Gradle starts or evidence edits.
func TestReplayRetainedMutationCapture(t *testing.T) {
	path, pin := os.Getenv("EIC_MUTATION_RECEIPT"), os.Getenv("EIC_MUTATION_RECEIPT_SHA256")
	if path == "" {
		t.Skip("retained Gradle capture not selected")
	}
	if err := checkBinding(fileBinding{path, pin}); err != nil {
		t.Fatal(err)
	}
	var receipt mutationReceipt
	if err := readJSONLimit(path, &receipt, 16<<20); err != nil {
		t.Fatal(err)
	}
	f, err := loadMutationFreeze(receipt.Freeze, false)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := inventory(f.Root, mutationEvidenceSelectors)
	if err != nil || !reflect.DeepEqual(actual, receipt.Artifacts) {
		t.Fatal("retained evidence changed", err)
	}
	for _, r := range f.Rows {
		if _, err := os.Stat(filepath.Join(f.Root, "attempts", r.ID, "outputs")); os.IsNotExist(err) {
			continue
		}
		if _, err = checkMutationRow(f, r, receipt.Freeze.SHA256); err != nil {
			t.Fatal(r.ID, err)
		}
	}
}

func TestMutationRequestAndContainmentAreDerived(t *testing.T) {
	f := mutationFreeze{Root: t.TempDir(), JDK: "/jdk", Gradle: "/gradle", Runner: fileBinding{Path: "/runner", SHA256: "runner"}}
	f.Rows = mutationRows()
	r, err := mutationRequestFor(f, f.Rows[0], "freeze")
	if err != nil {
		t.Fatal(err)
	}
	if r.Directory != filepath.Join(f.Root, "work", "rules", "N0") || r.TimeoutSeconds != 120 {
		t.Fatalf("%+v", r)
	}
	for _, arg := range []string{"--offline", "--no-daemon", "--build-cache", "--no-configuration-cache", "forbiddenPatterns"} {
		if !containsNativeEnv(r.Arguments, arg) {
			t.Fatal("missing", arg)
		}
	}
	if !containsNativeEnv(r.Environment, "GRADLE_USER_HOME="+filepath.Join(f.Root, "state", "N0", "gradle")) {
		t.Fatal("shared home")
	}
	bad := f.Rows[0]
	bad.ID = "QM025"
	if _, err = mutationRequestFor(f, bad, "freeze"); err == nil {
		t.Fatal("unallocated request")
	}
	args := mutationServiceArguments("unit.service", r.Runner.Path, "/request", "pin")
	if !containsNativeEnv(args, "--property=RuntimeMaxSec=130s") || !containsNativeEnv(args, "--property=KillMode=control-group") || !reflect.DeepEqual(args[len(args)-5:], []string{"/runner", "mutation-child", "/request", "pin", "unit.service"}) {
		t.Fatal(args)
	}
}

func TestMutationInputContractRejectsDrift(t *testing.T) {
	task, problems := mutationTestSources(t)
	root := t.TempDir()
	for _, r := range mutationRows() {
		dir, err := prepareMutationProject(root, r, task, problems)
		if err != nil {
			t.Fatal(err)
		}
		entries, err := inventory(dir, mutationInputSelectors)
		if err != nil {
			t.Fatal(err)
		}
		if err = checkMutationInputs(r, entries); err != nil {
			t.Fatal(r.ID, err)
		}
		for i, e := range entries {
			if e.Type == "file" {
				bad := append([]entry{}, entries...)
				bad[i].SHA256 = "drift"
				if checkMutationInputs(r, bad) == nil {
					t.Fatal("input drift accepted", e.Path)
				}
			}
		}
	}
}
