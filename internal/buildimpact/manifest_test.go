package buildimpact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validManifest() Manifest {
	return Manifest{
		SchemaVersion:       ManifestSchemaVersion,
		ManifestVersion:     1,
		RepositoryID:        "acme/monorepo",
		PipelineClass:       "pull-request",
		Ownership:           RepositoryOwnership,
		OriginalEntrypoints: []string{"assemble", "check"},
		AllowedAlternatives: []EntrypointSet{
			{ID: "service-a", Entrypoints: []string{":service-a:assemble", ":library-c:assemble"}},
			{ID: "service-b", Entrypoints: []string{":service-b:assemble"}},
		},
		RequiredArtifacts: []Artifact{
			{ID: "service-a-jar", Path: "service-a/build/libs/*.jar", Owner: BuildOptimization},
		},
		RequiredChecks: []Check{
			{ID: "compile-check", Owner: BuildOptimization},
			{ID: "jvm-tests", Owner: TestOptimization},
		},
		GlobalChangePaths:   []string{"settings.gradle.kts", "build-logic/**", "gradle/**"},
		UnknownChangePolicy: FullGraphPolicy,
	}
}

func encodeManifest(t *testing.T, manifest Manifest) []byte {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestParseManifestBindsRepositoryPipelineAndStableDigest(t *testing.T) {
	raw := encodeManifest(t, validManifest())
	first, err := ParseManifest(raw, "acme/monorepo", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseManifest(append([]byte("\n "), raw...), "acme/monorepo", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || !strings.HasPrefix(first.Digest, "sha256:") || len(first.Digest) != 71 {
		t.Fatalf("digests = %q, %q", first.Digest, second.Digest)
	}
}

func TestParseManifestRejectsUnknownFieldsAndTrailingDocuments(t *testing.T) {
	raw := encodeManifest(t, validManifest())
	withUnknown := append(raw[:len(raw)-1], []byte(`,"command":"./gradlew build"}`)...)
	if _, err := ParseManifest(withUnknown, "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("unknown command field accepted")
	}
	if _, err := ParseManifest(append(raw, raw...), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("trailing manifest accepted")
	}
}

func TestParseManifestRejectsBindingAndFailOpenPolicy(t *testing.T) {
	manifest := validManifest()
	for _, test := range []struct {
		name       string
		mutate     func(*Manifest)
		repository string
		pipeline   string
	}{
		{name: "schema", mutate: func(value *Manifest) { value.SchemaVersion = "1.0" }, repository: "acme/monorepo", pipeline: "pull-request"},
		{name: "version", mutate: func(value *Manifest) { value.ManifestVersion = 0 }, repository: "acme/monorepo", pipeline: "pull-request"},
		{name: "repository", mutate: func(*Manifest) {}, repository: "other/repo", pipeline: "pull-request"},
		{name: "pipeline", mutate: func(*Manifest) {}, repository: "acme/monorepo", pipeline: "main"},
		{name: "ownership", mutate: func(value *Manifest) { value.Ownership = "CONTROL_PLANE_INFERRED" }, repository: "acme/monorepo", pipeline: "pull-request"},
		{name: "unknown policy", mutate: func(value *Manifest) { value.UnknownChangePolicy = "OMIT" }, repository: "acme/monorepo", pipeline: "pull-request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := manifest
			test.mutate(&candidate)
			if _, err := ParseManifest(encodeManifest(t, candidate), test.repository, test.pipeline); err == nil {
				t.Fatal("invalid binding or policy accepted")
			}
		})
	}
}

func TestParseManifestRejectsUnsafeEntrypoints(t *testing.T) {
	for _, entrypoint := range []string{"", "--scan", ":service::assemble", ":..:assemble", ":service:assemble ", ":service:*"} {
		t.Run(entrypoint, func(t *testing.T) {
			manifest := validManifest()
			manifest.AllowedAlternatives[0].Entrypoints = []string{entrypoint}
			if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
				t.Fatalf("unsafe entrypoint %q accepted", entrypoint)
			}
		})
	}
	manifest := validManifest()
	manifest.AllowedAlternatives[0].Entrypoints = append([]string(nil), manifest.OriginalEntrypoints...)
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("original entrypoint set accepted as an alternative")
	}
}

func TestParseManifestRejectsUnsafePathsAndAmbiguousIDs(t *testing.T) {
	for _, unsafePath := range []string{"/tmp/output.jar", "../output.jar", "build\\output.jar", "build/[.jar", "build//output.jar"} {
		t.Run(unsafePath, func(t *testing.T) {
			manifest := validManifest()
			manifest.RequiredArtifacts[0].Path = unsafePath
			if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
				t.Fatalf("unsafe artifact path %q accepted", unsafePath)
			}
		})
	}
	manifest := validManifest()
	manifest.RequiredChecks[0].ID = manifest.RequiredArtifacts[0].ID
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("artifact/check ID collision accepted")
	}
	manifest = validManifest()
	manifest.RequiredArtifacts = append(manifest.RequiredArtifacts, Artifact{ID: "same-output", Path: manifest.RequiredArtifacts[0].Path, Owner: BuildOptimization})
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("duplicate artifact path accepted")
	}
	manifest = validManifest()
	manifest.RequiredChecks = nil
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("implicit required check list accepted")
	}
	manifest = validManifest()
	manifest.GlobalChangePaths[0] = "settings.gradle.kts\ncommand"
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("control character in repository path accepted")
	}
}

func TestParseManifestRequiresExplicitOwners(t *testing.T) {
	manifest := validManifest()
	manifest.RequiredArtifacts[0].Owner = TestOptimization
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("Test Optimization artifact ownership accepted")
	}
	manifest = validManifest()
	manifest.RequiredChecks[0].Owner = "SHARED"
	if _, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("ambiguous check owner accepted")
	}
}

func TestLoadRepositoryManifestRejectsTraversalAndSymlinks(t *testing.T) {
	root := t.TempDir()
	raw := encodeManifest(t, validManifest())
	if err := os.WriteFile(filepath.Join(root, "build-impact.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRepositoryManifest(root, "build-impact.json", "acme/monorepo", "pull-request")
	if err != nil || loaded.Manifest.ManifestVersion != 1 {
		t.Fatalf("load = %+v, %v", loaded, err)
	}
	if _, err := LoadRepositoryManifest(root, "../build-impact.json", "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("traversal path accepted")
	}
	if err := os.Symlink(filepath.Join(root, "build-impact.json"), filepath.Join(root, "linked.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRepositoryManifest(root, "linked.json", "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("symlink manifest accepted")
	}
	parent := t.TempDir()
	linkedRoot := filepath.Join(parent, "linked-root")
	if err := os.Symlink(root, linkedRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRepositoryManifest(linkedRoot, "build-impact.json", "acme/monorepo", "pull-request"); err == nil {
		t.Fatal("symlink repository root accepted")
	}
}

func TestParseManifestDefensivelyCopiesNestedSlices(t *testing.T) {
	manifest := validManifest()
	loaded, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	manifest.AllowedAlternatives[0].Entrypoints[0] = "mutated"
	if loaded.Manifest.AllowedAlternatives[0].Entrypoints[0] != ":service-a:assemble" {
		t.Fatal("loaded manifest aliases caller-owned entrypoints")
	}
}

func TestCheckedInManifestLoadsThroughProductionBoundary(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	loaded, err := LoadRepositoryManifest(repositoryRoot, filepath.FromSlash("fixtures/build-impact/manifest.v1.json"), "tonyredondo/buildopt", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Manifest.Ownership != RepositoryOwnership || len(loaded.Manifest.AllowedAlternatives) != 1 {
		t.Fatalf("loaded manifest = %+v", loaded.Manifest)
	}
}
