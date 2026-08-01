package buildimpact

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func selectionFixture(t *testing.T) (LoadedManifest, LoadedGraph, PromotionInput) {
	t.Helper()
	manifest, graphValue := validGraph(t)
	graph, err := ParseDeclaredGraph(encodeGraph(t, graphValue), manifest)
	if err != nil {
		t.Fatal(err)
	}
	promotion := qualifyingPromotionInput()
	promotion.RepositoryID = manifest.Manifest.RepositoryID
	promotion.PipelineClass = manifest.Manifest.PipelineClass
	promotion.ManifestDigest = manifest.Digest
	promotion.GraphDigest = graph.Digest
	promotion.AdapterVersion = graph.Graph.AdapterVersion
	for index := range promotion.Results {
		promotion.Results[index].RepositoryID = promotion.RepositoryID
		promotion.Results[index].PipelineClass = promotion.PipelineClass
		promotion.Results[index].ManifestDigest = promotion.ManifestDigest
		promotion.Results[index].GraphDigest = promotion.GraphDigest
		promotion.Results[index].AdapterVersion = promotion.AdapterVersion
		promotion.Results[index].AlternativeID = "service-a"
	}
	return manifest, graph, promotion
}

func TestQualifiedPromotionSelectsOnlyCustomerAlternative(t *testing.T) {
	manifest, graph, promotion := selectionFixture(t)
	plan := PlanSelection(
		manifest,
		graph,
		[]string{"library-c/src/main/java/Library.java"},
		promotion,
		SelectionControls{Enabled: true},
	)
	if plan.SchemaVersion != SelectionPlanSchemaVersion || plan.Mode != SelectionCustomerAlternative || plan.Reason != "PROMOTED_CUSTOMER_ALTERNATIVE" || !plan.SelectionAuthorized || plan.PromotionState != PromotionQualified {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.AlternativeID != "service-a" || !reflect.DeepEqual(plan.Entrypoints, []string{":service-a:assemble", ":library-c:assemble"}) || !reflect.DeepEqual(plan.AffectedProjects, []string{":library-c", ":service-a"}) || !reflect.DeepEqual(plan.OmittedProjects, []string{":service-b"}) || !reflect.DeepEqual(plan.PreservedTestCheckIDs, []string{"jvm-tests"}) {
		t.Fatalf("selected plan = %+v", plan)
	}
}

func TestSelectionControlsAlwaysRestoreOriginalEntrypoints(t *testing.T) {
	for _, test := range []struct {
		name     string
		controls SelectionControls
		reason   string
	}{
		{name: "disabled", controls: SelectionControls{}, reason: "SELECTION_DISABLED"},
		{name: "kill switch", controls: SelectionControls{Enabled: true, KillSwitchActive: true}, reason: "KILL_SWITCH_ACTIVE"},
		{name: "local bypass", controls: SelectionControls{Enabled: true, LocalBypass: true}, reason: "LOCAL_BYPASS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest, graph, promotion := selectionFixture(t)
			plan := PlanSelection(manifest, graph, []string{"library-c/src/main/java/Library.java"}, promotion, test.controls)
			assertFullGraphPlan(t, manifest, plan, test.reason)
		})
	}
}

func TestUnqualifiedOrDriftedPromotionCannotSelect(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*PromotionInput)
		reason string
	}{
		{
			name: "insufficient",
			mutate: func(input *PromotionInput) {
				input.Results = input.Results[:2]
			},
			reason: "PROMOTION_SHADOW_WINDOW_INSUFFICIENT",
		},
		{
			name: "suspended",
			mutate: func(input *PromotionInput) {
				input.Results[len(input.Results)-1].Outcome = ValidationFalseNegative
				input.Results[len(input.Results)-1].Reason = "REQUIRED_ARTIFACT_DIVERGENCE"
				input.Results[len(input.Results)-1].FalseNegative = true
			},
			reason: "PROMOTION_FALSE_NEGATIVE_OBSERVED",
		},
		{
			name: "manifest drift",
			mutate: func(input *PromotionInput) {
				input.ManifestDigest = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
			},
			reason: "PROMOTION_BINDING_MISMATCH",
		},
		{
			name: "graph drift",
			mutate: func(input *PromotionInput) {
				input.GraphDigest = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
			},
			reason: "PROMOTION_BINDING_MISMATCH",
		},
		{
			name: "adapter drift",
			mutate: func(input *PromotionInput) {
				input.AdapterVersion = "gradle-declared-v2"
			},
			reason: "PROMOTION_BINDING_MISMATCH",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest, graph, promotion := selectionFixture(t)
			test.mutate(&promotion)
			plan := PlanSelection(manifest, graph, []string{"library-c/src/main/java/Library.java"}, promotion, SelectionControls{Enabled: true})
			assertFullGraphPlan(t, manifest, plan, test.reason)
		})
	}
}

func TestGlobalUnknownAndInvalidLoadedInputsFailClosed(t *testing.T) {
	manifest, graph, promotion := selectionFixture(t)
	plan := PlanSelection(manifest, graph, []string{"settings.gradle.kts"}, promotion, SelectionControls{Enabled: true})
	assertFullGraphPlan(t, manifest, plan, "IMPACT_GLOBAL_CHANGE")

	plan = PlanSelection(manifest, graph, []string{"unknown/file.txt"}, promotion, SelectionControls{Enabled: true})
	assertFullGraphPlan(t, manifest, plan, "IMPACT_UNKNOWN_CHANGE_PATH")

	tamperedManifest := manifest
	tamperedManifest.Manifest.ManifestVersion++
	plan = PlanSelection(tamperedManifest, graph, []string{"library-c/src/main/java/Library.java"}, promotion, SelectionControls{Enabled: true})
	if plan.Reason != "MANIFEST_INVALID" || plan.SelectionAuthorized || len(plan.Entrypoints) != 0 {
		t.Fatalf("tampered manifest plan = %+v", plan)
	}

	tamperedGraph := graph
	tamperedGraph.Graph.Complete = false
	plan = PlanSelection(manifest, tamperedGraph, []string{"library-c/src/main/java/Library.java"}, promotion, SelectionControls{Enabled: true})
	assertFullGraphPlan(t, manifest, plan, "GRAPH_INVALID")
}

func TestSelectionPlanReturnsDefensiveCollections(t *testing.T) {
	manifest, graph, promotion := selectionFixture(t)
	plan := PlanSelection(manifest, graph, []string{"library-c/src/main/java/Library.java"}, promotion, SelectionControls{Enabled: true})
	plan.Entrypoints[0] = "tampered"
	plan.PreservedTestCheckIDs[0] = "tampered"
	second := PlanSelection(manifest, graph, []string{"library-c/src/main/java/Library.java"}, promotion, SelectionControls{Enabled: true})
	if second.Entrypoints[0] == "tampered" || second.PreservedTestCheckIDs[0] == "tampered" {
		t.Fatalf("selection plan leaked mutable collections: %+v", second)
	}
}

func assertFullGraphPlan(t *testing.T, manifest LoadedManifest, plan SelectionPlan, reason string) {
	t.Helper()
	if plan.Mode != SelectionFullGraph || plan.Reason != reason || plan.SelectionAuthorized || !reflect.DeepEqual(plan.Entrypoints, manifest.Manifest.OriginalEntrypoints) || plan.AlternativeID != "" {
		t.Fatalf("full graph plan = %+v", plan)
	}
}

func TestSyntheticRepositorySelectedBuildMatchesFullBuild(t *testing.T) {
	if os.Getenv("BUILDOPT_RUN_BUILD_IMPACT_GRADLE_PROOF") != "1" {
		t.Skip("real synthetic Gradle proof is run by check-build-impact-selection")
	}
	repositoryRoot := buildImpactRepositoryRoot(t)
	fixtureRoot := filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/build-impact/synthetic-repository"))
	manifest, err := LoadRepositoryManifest(fixtureRoot, "buildopt-impact-manifest.json", "tonyredondo/buildopt-impact-synthetic", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	graphRaw, err := os.ReadFile(filepath.Join(fixtureRoot, "buildopt-impact-graph.json"))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		t.Fatal(err)
	}
	promotion := qualifyingPromotionInput()
	promotion.RepositoryID = manifest.Manifest.RepositoryID
	promotion.PipelineClass = manifest.Manifest.PipelineClass
	promotion.ManifestDigest = manifest.Digest
	promotion.GraphDigest = graph.Digest
	promotion.AdapterVersion = graph.Graph.AdapterVersion
	for index := range promotion.Results {
		promotion.Results[index].RepositoryID = promotion.RepositoryID
		promotion.Results[index].PipelineClass = promotion.PipelineClass
		promotion.Results[index].ManifestDigest = promotion.ManifestDigest
		promotion.Results[index].GraphDigest = promotion.GraphDigest
		promotion.Results[index].AdapterVersion = promotion.AdapterVersion
		promotion.Results[index].AlternativeID = "affected-service-a"
	}
	plan := PlanSelection(
		manifest,
		graph,
		[]string{"library-c/src/main/java/synthetic/LibraryC.java"},
		promotion,
		SelectionControls{Enabled: true},
	)
	if !plan.SelectionAuthorized || !reflect.DeepEqual(plan.Entrypoints, []string{":service-a:assemble"}) || plan.AlternativeID != "affected-service-a" || !reflect.DeepEqual(plan.PreservedTestCheckIDs, []string{"synthetic-tests"}) {
		t.Fatalf("synthetic selection plan = %+v", plan)
	}

	baseline := filepath.Join(t.TempDir(), "baseline")
	candidate := filepath.Join(t.TempDir(), "candidate")
	copyFixtureTree(t, fixtureRoot, baseline)
	copyFixtureTree(t, fixtureRoot, candidate)
	runSyntheticGradle(t, repositoryRoot, baseline, "assemble", "testOwnedCheck")
	candidateTasks := append(append([]string(nil), plan.Entrypoints...), "testOwnedCheck")
	runSyntheticGradle(t, repositoryRoot, candidate, candidateTasks...)

	baselineJar := filepath.Join(baseline, "service-a", "build", "libs", "service-a-1.0.jar")
	candidateJar := filepath.Join(candidate, "service-a", "build", "libs", "service-a-1.0.jar")
	if fileDigest(t, baselineJar) != fileDigest(t, candidateJar) {
		t.Fatal("selected service-a artifact differs from full build")
	}
	if _, err := os.Stat(filepath.Join(baseline, "service-b", "build", "libs", "service-b-1.0.jar")); err != nil {
		t.Fatalf("full build omitted service-b: %v", err)
	}
	if _, err := os.Stat(filepath.Join(candidate, "service-b", "build", "libs", "service-b-1.0.jar")); !os.IsNotExist(err) {
		t.Fatalf("selected build unexpectedly materialized service-b: %v", err)
	}
	if fileDigest(t, filepath.Join(baseline, "build", "test-owned", "check.txt")) != fileDigest(t, filepath.Join(candidate, "build", "test-owned", "check.txt")) {
		t.Fatal("Test Optimization-owned check output diverged")
	}
}

func copyFixtureTree(t *testing.T, sourceRoot, destinationRoot string) {
	t.Helper()
	err := filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		source, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		defer source.Close()
		target, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(target, source); err != nil {
			target.Close()
			return err
		}
		return target.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runSyntheticGradle(t *testing.T, repositoryRoot, projectRoot string, tasks ...string) {
	t.Helper()
	args := []string{"--offline", "--no-daemon", "--console=plain", "-p", projectRoot}
	args = append(args, tasks...)
	command := exec.Command(filepath.Join(repositoryRoot, "gradlew"), args...)
	environment := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GRADLE_USER_HOME=") {
			environment = append(environment, value)
		}
	}
	command.Env = append(environment, "GRADLE_USER_HOME="+filepath.Join(repositoryRoot, ".tools", "gradle-user-home", "local"))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Gradle %v failed: %v\n%s", tasks, err, output)
	}
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
