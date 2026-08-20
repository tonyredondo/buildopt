package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOptimizeOutputMaterializationRestoresOnlyUnaffectedOutputs(t *testing.T) {
	repository, invocation, discovery := prepareOptimizeMaterializationFixture(t)
	materialization, err := captureOptimizeOutputMaterialization(invocation, discovery)
	if err != nil {
		t.Fatal(err)
	}
	discovery.Materialization = materialization
	if materialization.Status != optimizeMaterializationCaptured || materialization.FileCount != 1 {
		t.Fatalf("materialization = %+v", materialization)
	}
	if err := os.RemoveAll(filepath.Join(repository, "changed", "build")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repository, "unchanged", "build")); err != nil {
		t.Fatal(err)
	}
	run := &optimizeRun{invocation: invocation, previousState: &optimizeState{Discovery: discovery}}
	if err := run.materializeCandidateOutputs(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repository, "changed", "build", "changed.jar")); !os.IsNotExist(err) {
		t.Fatal("changed output was unexpectedly materialized")
	}
	raw, err := os.ReadFile(filepath.Join(repository, "unchanged", "build", "unchanged.jar"))
	if err != nil || string(raw) != "unchanged-output\n" {
		t.Fatalf("unaffected output was not restored exactly: %q, %v", raw, err)
	}
}

func TestOptimizeOutputMaterializationFailsBeforeWritingOnCorruption(t *testing.T) {
	repository, invocation, discovery := prepareOptimizeMaterializationFixture(t)
	materialization, err := captureOptimizeOutputMaterialization(invocation, discovery)
	if err != nil {
		t.Fatal(err)
	}
	discovery.Materialization = materialization
	manifestRaw, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(materialization.ManifestFile)))
	if err != nil {
		t.Fatal(err)
	}
	var manifest optimizeOutputMaterializationManifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, filepath.FromSlash(manifest.Entries[0].Blob)), []byte("corrupt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repository, "unchanged", "build")); err != nil {
		t.Fatal(err)
	}
	run := &optimizeRun{invocation: invocation, previousState: &optimizeState{Discovery: discovery}}
	if err := run.materializeCandidateOutputs(); err == nil {
		t.Fatal("corrupt materialization blob was accepted")
	}
	if _, err := os.Stat(filepath.Join(repository, "unchanged", "build", "unchanged.jar")); !os.IsNotExist(err) {
		t.Fatal("corrupt materialization wrote output bytes before failing")
	}
}

func TestOptimizeOutputMaterializationRejectsStaleWorkspaceBytes(t *testing.T) {
	repository, invocation, discovery := prepareOptimizeMaterializationFixture(t)
	materialization, err := captureOptimizeOutputMaterialization(invocation, discovery)
	if err != nil {
		t.Fatal(err)
	}
	discovery.Materialization = materialization
	stale := filepath.Join(repository, "unchanged", "build", "unchanged.jar")
	if err := os.WriteFile(stale, []byte("stale-output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run := &optimizeRun{invocation: invocation, previousState: &optimizeState{Discovery: discovery}}
	if err := run.materializeCandidateOutputs(); err == nil {
		t.Fatal("stale workspace output was overwritten")
	}
	raw, err := os.ReadFile(stale)
	if err != nil || string(raw) != "stale-output\n" {
		t.Fatalf("stale output changed during rejection: %q, %v", raw, err)
	}
}

func prepareOptimizeMaterializationFixture(t *testing.T) (string, optimizeInvocation, optimizeDiscoveryResult) {
	t.Helper()
	repository := t.TempDir()
	stateRelative := ".buildopt/optimize/v1"
	stateDirectory := filepath.Join(repository, filepath.FromSlash(stateRelative))
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"changed/build/changed.jar":     "changed-output\n",
		"unchanged/build/unchanged.jar": "unchanged-output\n",
	}
	for relative, contents := range files {
		path := filepath.Join(repository, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	discovery := optimizeDiscoveryResult{
		Status: optimizeDiscoveryComplete, RepositoryID: "example/materialization",
		TargetRevision:   "0123456789abcdef0123456789abcdef01234567",
		RequiredOutputs:  []string{"changed/build/*.jar", "unchanged/build/*.jar"},
		CandidateOutputs: []string{"changed/build/*.jar"},
	}
	return repository, optimizeInvocation{
		repositoryRoot: repository, stateDirectory: stateDirectory, stateRelative: stateRelative,
	}, discovery
}
