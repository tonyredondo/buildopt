package buildimpact

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGenerateImpactCanonicalizesConservativeSnapshot(t *testing.T) {
	manifest := automaticDiscoveryManifest(t)
	raw := []byte(`{
  "schemaVersion":"buildopt.build-impact/gradle-discovery/v1",
  "gradleVersion":"9.6.1",
  "complete":true,
  "fallbackReasons":[],
  "projects":[
    {"path":":service-b","sourcePaths":["service-b/**"],"dependsOn":[],"unknownRelationships":false},
    {"path":":library-c","sourcePaths":["library-c/**"],"dependsOn":[],"unknownRelationships":false},
    {"path":":service-a","sourcePaths":["service-a/**"],"dependsOn":[":library-c"],"unknownRelationships":false}
  ],
  "entrypoints":[
    {"name":"assemble","reachesProjects":[":service-b",":service-a",":library-c"],"containsTestTasks":false,"unknownRelationships":false},
    {"name":":service-a:assemble","reachesProjects":[":service-a",":library-c"],"containsTestTasks":false,"unknownRelationships":false}
  ]
}`)
	generated, err := GenerateImpact(manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !generated.Generated.Complete || generated.Generated.AdapterVersion != GradleDiscoveryAdapterVersion || generated.Generated.GraphDigest != generated.Graph.Digest {
		t.Fatalf("generated manifest = %+v", generated.Generated)
	}
	if got := generated.Graph.Graph.Projects[0].Path; got != ":library-c" {
		t.Fatalf("first canonical project = %q", got)
	}
	if got := generated.Graph.Graph.Entrypoints[0]; got.Name != ":service-a:assemble" || !reflect.DeepEqual(got.ReachesProjects, []string{":library-c", ":service-a"}) || !reflect.DeepEqual(got.ArtifactIDs, []string{"service-a-jar"}) || !reflect.DeepEqual(got.CheckIDs, []string{"compile-check"}) {
		t.Fatalf("generated alternative = %+v", got)
	}
	second, err := GenerateImpact(manifest, raw)
	if err != nil || !bytes.Equal(generated.GraphJSON, second.GraphJSON) || !bytes.Equal(generated.GeneratedJSON, second.GeneratedJSON) {
		t.Fatal("identical discovery was not deterministic")
	}
}

func TestGenerateImpactRejectsAmbiguousCompleteness(t *testing.T) {
	manifest := automaticDiscoveryManifest(t)
	raw := []byte(`{
  "schemaVersion":"buildopt.build-impact/gradle-discovery/v1",
  "gradleVersion":"9.6.1",
  "complete":true,
  "fallbackReasons":["INCLUDED_BUILDS_PRESENT"],
  "projects":[{"path":":library-c","sourcePaths":["library-c/**"],"dependsOn":[],"unknownRelationships":false}],
  "entrypoints":[]
}`)
	if _, err := GenerateImpact(manifest, raw); err == nil {
		t.Fatal("ambiguous complete discovery was accepted")
	}
}

func automaticDiscoveryManifest(t *testing.T) LoadedManifest {
	t.Helper()
	manifest := validManifest()
	manifest.OriginalEntrypoints = []string{"assemble"}
	manifest.AllowedAlternatives = []EntrypointSet{{ID: "service-a", Entrypoints: []string{":service-a:assemble"}}}
	loaded, err := ParseManifest(encodeManifest(t, manifest), manifest.RepositoryID, manifest.PipelineClass)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestSyntheticGradleDiscoveryGeneratedStateIsCurrent(t *testing.T) {
	if os.Getenv("BUILDOPT_RUN_BUILD_IMPACT_DISCOVERY_PROOF") != "1" {
		t.Skip("real Gradle discovery proof is run by check-build-impact-automatic")
	}
	repositoryRoot := buildImpactRepositoryRoot(t)
	fixtureRoot := filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	generated, err := Discover(ctx, DiscoveryOptions{
		RepositoryRoot: fixtureRoot,
		ManifestPath:   "buildopt-impact-manifest.json",
		RepositoryID:   "tonyredondo/buildopt-impact-synthetic",
		PipelineClass:  "pull-request",
		GradleCommand:  filepath.Join(repositoryRoot, "gradlew"),
		GradleArgs:     []string{"--offline"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFixtureBytes(t, filepath.Join(fixtureRoot, "buildopt-impact-graph.generated.json"), generated.GraphJSON)
	assertFixtureBytes(t, filepath.Join(fixtureRoot, "buildopt-impact.generated.json"), generated.GeneratedJSON)
	if !generated.Graph.Graph.Complete || !reflect.DeepEqual(generated.Graph.Graph.Projects[1].DependsOn, []string{":library-c"}) {
		t.Fatalf("generated fixture graph = %+v", generated.Graph.Graph)
	}
}

func assertFixtureBytes(t *testing.T, path string, expected []byte) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, expected) {
		t.Fatalf("generated fixture drift at %s", path)
	}
}
