package main

import (
	"os"
	"path/filepath"
	"testing"
)

func mutationTestSources(t *testing.T) ([]byte, []byte) {
	t.Helper()
	task, e := os.ReadFile("../../jvm/patcher/src/spike/resources/reviewed-native/ForbiddenPatternsTask.java.txt")
	if e != nil {
		t.Fatal(e)
	}
	problems, e := os.ReadFile("testdata/ElasticsearchBuildProblems.java.txt")
	if e != nil {
		t.Fatal(e)
	}
	return task, problems
}

func TestActualMutationProjectsPreserveFrozenSourceAndInputs(t *testing.T) {
	task, problems := mutationTestSources(t)
	rows := mutationRows()
	if len(rows) != 24 {
		t.Fatalf("mutation allocation: %d", len(rows))
	}
	root := t.TempDir()
	for _, r := range rows {
		dir, err := prepareMutationProject(root, r, task, problems)
		if err != nil {
			t.Fatal(r.ID, err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, "buildSrc/src/main/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTask.java"))
		if err != nil {
			t.Fatal(err)
		}
		expected := sourceInputs().TaskPreimageSHA256
		if r.Arm == "N1" {
			expected = sourceInputs().TaskPostimageSHA256
		}
		if digest(raw) != expected {
			t.Fatal("upstream source or annotation-only candidate changed")
		}
	}
	if _, err := prepareMutationProject(root, rows[0], task, problems); err == nil {
		t.Fatal("fixture replay accepted")
	}
	if _, err := prepareMutationProject(t.TempDir(), row{ID: "escape", Arm: "../escape", State: "rules:before"}, task, problems); err == nil {
		t.Fatal("unallocated row accepted")
	}
	bad := append([]byte{}, task...)
	bad[0] ^= 1
	if _, err := prepareMutationProject(t.TempDir(), rows[0], bad, problems); err == nil {
		t.Fatal("upstream drift accepted")
	}
}

func TestMutationPreparationHasRealInvalidationInputs(t *testing.T) {
	task, problems := mutationTestSources(t)
	root := t.TempDir()
	for _, r := range mutationRows() {
		dir, err := prepareMutationProject(root, r, task, problems)
		if err != nil {
			t.Fatal(err)
		}
		if r.State == "relative-rename:after" {
			if _, err = os.Stat(filepath.Join(dir, "src/renamed.txt")); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Stat(filepath.Join(dir, "src/input.txt")); !os.IsNotExist(err) {
				t.Fatal("rename retained old input")
			}
		}
		if r.State == "all-source-removal:after" {
			es, e := os.ReadDir(filepath.Join(dir, "src"))
			if e != nil || len(es) != 0 {
				t.Fatal("source removal did not empty inputs")
			}
		}
		if r.State == "malformed-utf8:after" {
			b, e := os.ReadFile(filepath.Join(dir, "src/input.txt"))
			if e != nil || string(b) != string([]byte{0xc3, 0x28}) {
				t.Fatal("invalid UTF-8 fixture missing")
			}
		}
	}
}
