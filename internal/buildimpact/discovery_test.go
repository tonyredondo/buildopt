package buildimpact

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
  "includedBuildPaths":["conventions/**"],
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
	if !reflect.DeepEqual(generated.Graph.Graph.GlobalChangePaths, []string{"conventions/**"}) {
		t.Fatalf("included-build global paths = %v", generated.Graph.Graph.GlobalChangePaths)
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

func TestResolveProjectOwnersUsesMostSpecificSourceBoundary(t *testing.T) {
	snapshot := DiscoverySnapshot{Projects: []DiscoveredProject{
		{Path: ":parent", SourcePaths: []string{"modules/**"}},
		{Path: ":child", SourcePaths: []string{"modules/child/**"}},
	}}
	owners, err := ResolveProjectOwners(snapshot, []string{"modules/child/src/main/java/Example.java"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(owners, []string{":child"}) {
		t.Fatalf("owners = %v", owners)
	}
	if _, err := ResolveProjectOwners(DiscoverySnapshot{Projects: []DiscoveredProject{
		{Path: ":left", SourcePaths: []string{"shared/**"}},
		{Path: ":right", SourcePaths: []string{"shared/**"}},
	}}, []string{"shared/Example.java"}); err == nil {
		t.Fatal("ambiguous source ownership was accepted")
	}
	if _, err := ResolveProjectOwners(snapshot, []string{"outside/Example.java"}); err == nil {
		t.Fatal("unowned source path was accepted")
	}
}

func TestResolveProjectOwnersPrefersDirectOwnershipOverTransitiveInputs(t *testing.T) {
	snapshot := DiscoverySnapshot{Projects: []DiscoveredProject{
		{
			Path:             ":consumer",
			SourcePaths:      []string{"modules/library/**", "modules/consumer/**"},
			OwnedSourcePaths: []string{"modules/consumer/**"},
		},
		{
			Path:             ":library",
			SourcePaths:      []string{"modules/library/**"},
			OwnedSourcePaths: []string{"modules/library/**"},
		},
	}}
	owners, err := ResolveProjectOwners(snapshot, []string{"modules/library/src/Library.java"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(owners, []string{":library"}) {
		t.Fatalf("owners = %v", owners)
	}
	if _, err := ResolveProjectOwners(DiscoverySnapshot{Projects: []DiscoveredProject{
		{Path: ":left", SourcePaths: []string{"shared/**"}, OwnedSourcePaths: []string{"shared/**"}},
		{Path: ":right", SourcePaths: []string{"shared/**"}, OwnedSourcePaths: []string{"shared/**"}},
	}}, []string{"shared/Example.java"}); err == nil {
		t.Fatal("ambiguous direct ownership was accepted")
	}
}

func TestGenerateImpactAcceptsRootProjectDependency(t *testing.T) {
	manifest := automaticDiscoveryManifest(t)
	raw := []byte(`{
  "schemaVersion":"buildopt.build-impact/gradle-discovery/v1",
  "gradleVersion":"8.14.4",
  "complete":true,
  "fallbackReasons":[],
  "projects":[
    {"path":":","sourcePaths":["src/**"],"dependsOn":[],"unknownRelationships":false},
    {"path":":service-a","sourcePaths":["service-a/**"],"dependsOn":[":"],"unknownRelationships":false},
    {"path":":service-b","sourcePaths":["service-b/**"],"dependsOn":[],"unknownRelationships":false}
  ],
  "entrypoints":[
    {"name":"assemble","reachesProjects":[":",":service-a",":service-b"],"containsTestTasks":false,"unknownRelationships":false},
    {"name":":service-a:assemble","reachesProjects":[":",":service-a"],"containsTestTasks":false,"unknownRelationships":false}
  ]
}`)
	generated, err := GenerateImpact(manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	projects := generated.Graph.Graph.Projects
	if len(projects) != 3 || projects[0].Path != ":" || !reflect.DeepEqual(projects[1].DependsOn, []string{":"}) {
		t.Fatalf("generated root dependency graph = %+v", projects)
	}
	decision := EvaluateImpact(manifest, generated.Graph.Graph, []string{"src/main/java/example/Core.java"})
	if decision.Mode != DecisionShadowAlternative || !reflect.DeepEqual(decision.AffectedProjects, []string{":", ":service-a"}) {
		t.Fatalf("root dependency impact decision = %+v", decision)
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

func TestGenerateImpactConservativelyNormalizesProjectDependencyCycles(t *testing.T) {
	manifest := automaticDiscoveryManifest(t)
	raw := []byte(`{
  "schemaVersion":"buildopt.build-impact/gradle-discovery/v1",
  "gradleVersion":"9.6.1",
  "complete":true,
  "fallbackReasons":[],
  "projects":[
    {"path":":library-c","sourcePaths":["library-c/**"],"dependsOn":[":service-a"],"unknownRelationships":false},
    {"path":":service-a","sourcePaths":["service-a/**"],"dependsOn":[":library-c"],"unknownRelationships":false},
    {"path":":service-b","sourcePaths":["service-b/**"],"dependsOn":[],"unknownRelationships":false}
  ],
  "entrypoints":[
    {"name":"assemble","reachesProjects":[":library-c",":service-a",":service-b"],"containsTestTasks":false,"unknownRelationships":false},
    {"name":":service-a:assemble","reachesProjects":[":library-c",":service-a"],"containsTestTasks":false,"unknownRelationships":false}
  ]
}`)
	generated, err := GenerateImpact(manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, project := range generated.Graph.Graph.Projects[:2] {
		expectedOwned := []string{strings.TrimPrefix(project.Path, ":") + "/**"}
		if !reflect.DeepEqual(project.SourcePaths, []string{"library-c/**", "service-a/**"}) ||
			!reflect.DeepEqual(project.OwnedSourcePaths, expectedOwned) || len(project.DependsOn) != 0 {
			t.Fatalf("normalized cyclic project = %+v", project)
		}
	}
	decision := EvaluateImpact(manifest, generated.Graph.Graph, []string{"library-c/src/main/java/Library.java"})
	if decision.Mode != DecisionShadowAlternative || !reflect.DeepEqual(decision.AffectedProjects, []string{":library-c", ":service-a"}) {
		t.Fatalf("cycle-safe impact decision = %+v", decision)
	}
}

func TestDeriveProjectEntrypointsUsesTransitiveProjectDependencies(t *testing.T) {
	snapshot := DiscoverySnapshot{
		SchemaVersion: DiscoverySchemaVersion, GradleVersion: "9.6.1", Complete: true,
		FallbackReasons: []string{},
		Projects: []DiscoveredProject{
			{Path: ":app", SourcePaths: []string{"app/**"}, DependsOn: []string{":service"}},
			{Path: ":service", SourcePaths: []string{"service/**"}, DependsOn: []string{":library"}},
			{Path: ":library", SourcePaths: []string{"library/**"}, DependsOn: []string{}},
		},
		Entrypoints: []DiscoveredEntrypoint{{Name: "assemble", ReachesProjects: []string{":app", ":library", ":service"}}},
	}
	derived, err := DeriveProjectEntrypoints(snapshot, []string{":app:assemble"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{":app", ":library", ":service"}
	if len(derived.Entrypoints) != 2 || derived.Entrypoints[0].Name != "assemble" ||
		!reflect.DeepEqual(derived.Entrypoints[1].ReachesProjects, want) ||
		derived.Entrypoints[1].UnknownRelationships || derived.Entrypoints[1].ContainsTestTasks {
		t.Fatalf("derived entrypoint = %+v, want reaches %v", derived.Entrypoints, want)
	}
	if _, err := DeriveProjectEntrypoints(snapshot, []string{":missing:assemble"}); err == nil {
		t.Fatal("missing candidate project was accepted")
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

func TestDiscoveryGradleArgumentsOwnObservationOnlyOptionsWithoutDroppingOwnerProperties(t *testing.T) {
	got := discoveryGradleArguments([]string{
		"--daemon", "--build-cache", "--configure-on-demand", "--console=rich", "-Powner.mode=jvm",
	}, "/private/discovery.init.gradle")
	want := []string{
		"--build-cache", "-Powner.mode=jvm", "--no-daemon", "--no-configure-on-demand", "--console=plain",
		"--init-script", "/private/discovery.init.gradle", "buildoptImpactDiscovery",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discovery Gradle arguments = %#v, want %#v", got, want)
	}
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
	for _, project := range generated.Graph.Graph.Projects {
		for _, dependency := range project.DependsOn {
			if dependency == project.Path {
				t.Fatalf("generated self dependency for %s", project.Path)
			}
		}
	}
}

func TestTestPreparationDiscoveryDistinguishesBuildWorkFromTestExecution(t *testing.T) {
	if os.Getenv("BUILDOPT_RUN_BUILD_IMPACT_DISCOVERY_PROOF") != "1" {
		t.Skip("real Gradle discovery proof is run by check-build-impact-automatic")
	}
	repositoryRoot := buildImpactRepositoryRoot(t)
	fixtureRoot := filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/build-impact/test-preparation"))
	discover := func(t *testing.T) GeneratedImpact {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		generated, err := Discover(ctx, DiscoveryOptions{
			RepositoryRoot: fixtureRoot,
			ManifestPath:   "buildopt-impact-manifest.json",
			RepositoryID:   "tonyredondo/buildopt-impact-test-preparation",
			PipelineClass:  "pull-request",
			GradleCommand:  filepath.Join(repositoryRoot, "gradlew"),
			GradleArgs:     []string{"--offline"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return generated
	}

	generated := discover(t)
	assertFixtureBytes(t, filepath.Join(fixtureRoot, "buildopt-impact-graph.generated.json"), generated.GraphJSON)
	assertFixtureBytes(t, filepath.Join(fixtureRoot, "buildopt-impact.generated.json"), generated.GeneratedJSON)
	if !generated.Generated.Complete {
		t.Fatalf("safe test preparation was not complete: %+v", generated.Generated)
	}
	for _, entrypoint := range generated.Graph.Graph.Entrypoints {
		if entrypoint.ContainsTestTasks || entrypoint.UnknownRelationships {
			t.Fatalf("safe test preparation entrypoint = %+v", entrypoint)
		}
	}

	t.Setenv("BUILDOPT_TEST_PREPARATION_UNSAFE", "1")
	unsafe := discover(t)
	if unsafe.Generated.Complete || !reflect.DeepEqual(unsafe.Generated.FallbackReasons, []string{"UNSUPPORTED_OR_TEST_ENTRYPOINT"}) {
		t.Fatalf("Test dependency did not fail closed: %+v", unsafe.Generated)
	}
	foundTest := false
	for _, entrypoint := range unsafe.Graph.Graph.Entrypoints {
		foundTest = foundTest || entrypoint.ContainsTestTasks
	}
	if !foundTest {
		t.Fatal("unsafe test preparation did not expose its Test dependency")
	}
}

func TestDiscoveryAcceptsTypedBuildOwnedProducersAndRejectsArbitraryTasks(t *testing.T) {
	if os.Getenv("BUILDOPT_RUN_BUILD_IMPACT_DISCOVERY_PROOF") != "1" {
		t.Skip("real Gradle discovery proof is run by check-build-impact-automatic")
	}
	repositoryRoot := buildImpactRepositoryRoot(t)
	workspace := t.TempDir()
	write := func(name, contents string) {
		t.Helper()
		path := filepath.Join(workspace, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("settings.gradle.kts", "rootProject.name = \"typed-compile-producer\"\n")
	write("build.gradle.kts", `plugins { java; checkstyle }
sourceSets.create("javaSpring3")
tasks.register<Jar>("sourceBundle")
tasks.register("aggregateProducer") { dependsOn("sourceBundle") }
tasks.register("arbitraryProducer")
`)
	write("src/javaSpring3/java/example/Spring3.java", "package example; public final class Spring3 {}\n")

	discover := func(entrypoint string) GeneratedImpact {
		t.Helper()
		write("buildopt-impact-manifest.json", fmt.Sprintf(`{
  "schemaVersion":"buildopt.build-impact/manifest/v1",
  "manifestVersion":1,
  "repositoryId":"tonyredondo/buildopt-typed-compile",
  "pipelineClass":"pull-request",
  "ownership":"REPOSITORY_COMMITTED",
  "originalEntrypoints":["assemble"],
  "allowedAlternatives":[{"id":"candidate","entrypoints":[%q]}],
  "requiredArtifacts":[{"id":"classes","path":"build/classes/**","owner":"BUILD_OPTIMIZATION"}],
  "requiredChecks":[],
  "globalChangePaths":["gradle/**"],
  "unknownChangePolicy":"FULL_GRAPH"
}`, entrypoint))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		generated, err := Discover(ctx, DiscoveryOptions{
			RepositoryRoot: workspace,
			ManifestPath:   "buildopt-impact-manifest.json",
			RepositoryID:   "tonyredondo/buildopt-typed-compile",
			PipelineClass:  "pull-request",
			GradleCommand:  filepath.Join(repositoryRoot, "gradlew"),
			GradleArgs:     []string{"--offline"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return generated
	}

	typed := discover(":compileJavaSpring3Java")
	if !typed.Generated.Complete || len(typed.Generated.FallbackReasons) != 0 {
		t.Fatalf("typed compile producer was not accepted: %+v", typed.Generated)
	}
	verification := discover(":checkstyleMain")
	if !verification.Generated.Complete || len(verification.Generated.FallbackReasons) != 0 {
		t.Fatalf("typed verification producer was not accepted: %+v", verification.Generated)
	}
	archive := discover(":sourceBundle")
	if !archive.Generated.Complete || len(archive.Generated.FallbackReasons) != 0 {
		t.Fatalf("typed archive producer was not accepted: %+v", archive.Generated)
	}
	aggregate := discover(":aggregateProducer")
	if !aggregate.Generated.Complete || len(aggregate.Generated.FallbackReasons) != 0 {
		t.Fatalf("no-action aggregate producer was not accepted: %+v", aggregate.Generated)
	}
	arbitrary := discover(":arbitraryProducer")
	if arbitrary.Generated.Complete || !reflect.DeepEqual(arbitrary.Generated.FallbackReasons, []string{"UNSUPPORTED_OR_TEST_ENTRYPOINT"}) {
		t.Fatalf("arbitrary producer did not fail closed: %+v", arbitrary.Generated)
	}
}

func TestDiscoveryIncludesRootWhenSubprojectDependsOnIt(t *testing.T) {
	if os.Getenv("BUILDOPT_RUN_BUILD_IMPACT_DISCOVERY_PROOF") != "1" {
		t.Skip("real Gradle discovery proof is run by check-build-impact-automatic")
	}
	repositoryRoot := buildImpactRepositoryRoot(t)
	workspace := t.TempDir()
	write := func(name, contents string) {
		t.Helper()
		path := filepath.Join(workspace, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("settings.gradle.kts", "rootProject.name = \"root-producer\"\ninclude(\"consumer\")\n")
	write("build.gradle.kts", "plugins { java }\n")
	write("consumer/build.gradle.kts", "plugins { java }\ndependencies { implementation(project(\":\")) }\n")
	write("src/main/java/example/Core.java", "package example; public final class Core {}\n")
	write("consumer/src/main/java/example/Consumer.java", "package example; public final class Consumer {}\n")
	write("buildopt-impact-manifest.json", `{
  "schemaVersion":"buildopt.build-impact/manifest/v1",
  "manifestVersion":1,
  "repositoryId":"tonyredondo/buildopt-root-producer",
  "pipelineClass":"pull-request",
  "ownership":"REPOSITORY_COMMITTED",
  "originalEntrypoints":["assemble"],
  "allowedAlternatives":[{"id":"consumer","entrypoints":[":consumer:assemble"]}],
  "requiredArtifacts":[{"id":"consumer-jar","path":"consumer/build/libs/*.jar","owner":"BUILD_OPTIMIZATION"}],
  "requiredChecks":[],
  "globalChangePaths":["gradle/**"],
  "unknownChangePolicy":"FULL_GRAPH"
}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	generated, err := Discover(ctx, DiscoveryOptions{
		RepositoryRoot: workspace,
		ManifestPath:   "buildopt-impact-manifest.json",
		RepositoryID:   "tonyredondo/buildopt-root-producer",
		PipelineClass:  "pull-request",
		GradleCommand:  filepath.Join(repositoryRoot, "gradlew"),
		GradleArgs:     []string{"--offline"},
	})
	if err != nil {
		t.Fatal(err)
	}
	projects := generated.Graph.Graph.Projects
	if len(projects) != 2 || projects[0].Path != ":" || projects[1].Path != ":consumer" || !reflect.DeepEqual(projects[1].DependsOn, []string{":"}) {
		t.Fatalf("discovered root producer graph = %+v", projects)
	}
	for _, entrypoint := range generated.Graph.Graph.Entrypoints {
		if entrypoint.Name == ":consumer:assemble" && !reflect.DeepEqual(entrypoint.ReachesProjects, []string{":", ":consumer"}) {
			t.Fatalf("consumer reach = %+v", entrypoint)
		}
	}
}

func TestPocCombinedDiscoveryIgnoresInheritedBuildSrcInit(t *testing.T) {
	if os.Getenv("BUILDOPT_RUN_BUILD_IMPACT_DISCOVERY_PROOF") != "1" {
		t.Skip("real Gradle discovery proof is run by check-build-impact-automatic")
	}
	repositoryRoot := buildImpactRepositoryRoot(t)
	baseRoot := filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/poc-value/build-impact"))
	overlayRoot := filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/poc-value/combined-impact"))
	for _, dsl := range []string{"kotlin", "groovy"} {
		t.Run(dsl, func(t *testing.T) {
			workspace := filepath.Join(t.TempDir(), "repository")
			copyFixtureTree(t, baseRoot, workspace)
			for _, name := range []string{
				"build.gradle",
				"build.gradle.kts",
				"buildopt-impact-manifest.json",
				"buildopt-impact-graph.generated.json",
				"buildopt-impact.generated.json",
			} {
				raw, err := os.ReadFile(filepath.Join(overlayRoot, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workspace, name), raw, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if dsl == "kotlin" {
				if err := os.Remove(filepath.Join(workspace, "build.gradle")); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Join(workspace, "settings.gradle")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Remove(filepath.Join(workspace, "build.gradle.kts")); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Join(workspace, "settings.gradle.kts")); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			generated, err := Discover(ctx, DiscoveryOptions{
				RepositoryRoot: workspace,
				ManifestPath:   "buildopt-impact-manifest.json",
				RepositoryID:   "tonyredondo/buildopt-poc-combined-impact",
				PipelineClass:  "poc-value",
				GradleCommand:  filepath.Join(repositoryRoot, "gradlew"),
				GradleArgs:     []string{"--offline"},
			})
			if err != nil {
				t.Fatal(err)
			}
			assertFixtureBytes(t, filepath.Join(overlayRoot, "buildopt-impact-graph.generated.json"), generated.GraphJSON)
			assertFixtureBytes(t, filepath.Join(overlayRoot, "buildopt-impact.generated.json"), generated.GeneratedJSON)
		})
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
