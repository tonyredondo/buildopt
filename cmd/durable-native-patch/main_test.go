package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/durablenative"
)

func TestRunAppliesAndRevertsFamilyPlan(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "buildSrc", "Generate.java")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	source := []byte("public abstract class Generate extends DefaultTask {\n @Input public String getValue() { return \"x\"; }\n @OutputFile public File getOutput() { return null; }\n @TaskAction public void run() {}\n}\n")
	if err := os.WriteFile(sourcePath, source, 0o640); err != nil {
		t.Fatal(err)
	}
	candidate, ok := durablenative.ScanSource("buildSrc/Generate.java", source)
	if !ok {
		t.Fatal("fixture was not detected")
	}
	patch, err := durablenative.CompilePatch(source, candidate)
	if err != nil {
		t.Fatal(err)
	}
	planPath := writePlan(t, root, plan{Patches: []plannedPatch{{Family: "fixture", Patch: patch}}})

	if err := run(planPath, root, "fixture", "apply"); err != nil {
		t.Fatal(err)
	}
	if err := run(planPath, root, "fixture", "apply"); err != nil {
		t.Fatalf("second apply was not idempotent: %v", err)
	}
	patched, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patched), "@org.gradle.api.tasks.CacheableTask") {
		t.Fatal("cacheability marker was not applied")
	}
	if info, err := os.Stat(sourcePath); err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("source mode changed: %v %v", info, err)
	}
	if err := run(planPath, root, "fixture", "revert"); err != nil {
		t.Fatal(err)
	}
	reverted, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(reverted) != string(source) {
		t.Fatal("revert did not restore exact source bytes")
	}
}

func TestRunRejectsUnknownFamilyAndEscapingPath(t *testing.T) {
	root := t.TempDir()
	if err := run(writePlan(t, root, plan{}), root, "missing", "apply"); err == nil {
		t.Fatal("missing family was accepted")
	}
	escaping := durablenative.Patch{
		SchemaVersion: "buildopt.patch/add-cacheable-task-marker/v1",
		Path:          "../outside.java", ClassName: "Outside",
		ExpectedSourceSHA256: strings.Repeat("a", 64), PatchedSourceSHA256: strings.Repeat("b", 64),
		InsertedText: "@org.gradle.api.tasks.CacheableTask\n",
	}
	err := run(writePlan(t, root, plan{Patches: []plannedPatch{{Family: "fixture", Patch: escaping}}}), root, "fixture", "apply")
	if err == nil || !strings.Contains(err.Error(), "escapes source root") {
		t.Fatalf("escaping path result = %v", err)
	}
}

func writePlan(t *testing.T, root string, value plan) string {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, strings.ReplaceAll(t.Name(), "/", "-")+".json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
