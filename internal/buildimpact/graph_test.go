package buildimpact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func loadedGraphManifest(t *testing.T) LoadedManifest {
	t.Helper()
	manifest := validManifest()
	manifest.OriginalEntrypoints = []string{"assemble"}
	manifest.RequiredArtifacts = []Artifact{{ID: "distribution", Path: "distribution/build/libs/*.zip", Owner: BuildOptimization}}
	loaded, err := ParseManifest(encodeManifest(t, manifest), "acme/monorepo", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func validGraph(t *testing.T) (LoadedManifest, DeclaredGraph) {
	t.Helper()
	manifest := loadedGraphManifest(t)
	graph := DeclaredGraph{
		SchemaVersion:  DeclaredGraphSchemaVersion,
		RepositoryID:   "acme/monorepo",
		PipelineClass:  "pull-request",
		ManifestDigest: manifest.Digest,
		AdapterVersion: "gradle-declared-v1",
		Complete:       true,
		Projects: []Project{
			{Path: ":library-c", SourcePaths: []string{"library-c/**"}, DependsOn: []string{}},
			{Path: ":service-a", SourcePaths: []string{"service-a/**"}, DependsOn: []string{":library-c"}},
			{Path: ":service-b", SourcePaths: []string{"service-b/**"}, DependsOn: []string{}},
		},
		Entrypoints: []DeclaredEntrypoint{
			{Name: "assemble", Owner: BuildOptimization, ReachesProjects: []string{":library-c", ":service-a", ":service-b"}, ArtifactIDs: []string{"distribution"}, CheckIDs: []string{"compile-check"}},
			{Name: ":library-c:assemble", Owner: BuildOptimization, ReachesProjects: []string{":library-c"}, ArtifactIDs: []string{}, CheckIDs: []string{}},
			{Name: ":service-a:assemble", Owner: BuildOptimization, ReachesProjects: []string{":service-a", ":library-c"}, ArtifactIDs: []string{"distribution"}, CheckIDs: []string{"compile-check"}},
			{Name: ":service-b:assemble", Owner: BuildOptimization, ReachesProjects: []string{":service-b"}, ArtifactIDs: []string{"distribution"}, CheckIDs: []string{"compile-check"}},
		},
		GlobalChangePaths: []string{"conventions/**"},
	}
	return manifest, graph
}

func encodeGraph(t *testing.T, graph DeclaredGraph) []byte {
	t.Helper()
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestParseDeclaredGraphBindsManifestAndCanonicalDigest(t *testing.T) {
	manifest, graph := validGraph(t)
	first, err := ParseDeclaredGraph(encodeGraph(t, graph), manifest)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseDeclaredGraph(append([]byte("\n"), encodeGraph(t, graph)...), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || len(first.Digest) != 71 {
		t.Fatalf("digests = %q, %q", first.Digest, second.Digest)
	}
	graph.Projects[0].SourcePaths[0] = "mutated/**"
	if first.Graph.Projects[0].SourcePaths[0] != "library-c/**" {
		t.Fatal("loaded graph aliases caller state")
	}
}

func TestDeclaredGraphRejectsUnknownTrailingAndMismatchedBindings(t *testing.T) {
	manifest, graph := validGraph(t)
	raw := encodeGraph(t, graph)
	unknown := append(raw[:len(raw)-1], []byte(`,"historyAuthorizes":true}`)...)
	if _, err := ParseDeclaredGraph(unknown, manifest); err == nil {
		t.Fatal("unknown graph field accepted")
	}
	if _, err := ParseDeclaredGraph(append(raw, raw...), manifest); err == nil {
		t.Fatal("trailing graph accepted")
	}
	graph.ManifestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := ParseDeclaredGraph(encodeGraph(t, graph), manifest); err == nil {
		t.Fatal("mismatched manifest digest accepted")
	}
}

func TestLocalizedChangeProducesShadowPredictionOnly(t *testing.T) {
	manifest, graph := validGraph(t)
	decision := EvaluateImpact(manifest, graph, []string{"library-c/src/main/java/Library.java"})
	if decision.Mode != DecisionShadowAlternative || decision.Reason != "CUSTOMER_ALTERNATIVE_AND_DECLARED_GRAPH" || decision.PredictedAlternativeID != "service-a" {
		t.Fatalf("decision = %+v", decision)
	}
	if decision.SelectionAuthorized || !reflect.DeepEqual(decision.ExecutableEntrypoints, []string{"assemble"}) {
		t.Fatalf("shadow execution changed = %+v", decision)
	}
	if !reflect.DeepEqual(decision.AffectedProjects, []string{":library-c", ":service-a"}) || !reflect.DeepEqual(decision.OmittedProjects, []string{":service-b"}) {
		t.Fatalf("project sets = %+v", decision)
	}
	if !reflect.DeepEqual(decision.PreservedTestCheckIDs, []string{"jvm-tests"}) {
		t.Fatalf("Test Optimization checks = %+v", decision.PreservedTestCheckIDs)
	}
}

func TestGlobalAndUnknownChangesAlwaysUseFullGraph(t *testing.T) {
	manifest, graph := validGraph(t)
	for _, test := range []struct {
		path   string
		reason string
	}{
		{path: "settings.gradle.kts", reason: "GLOBAL_CHANGE"},
		{path: "gradle/libs.versions.toml", reason: "GLOBAL_CHANGE"},
		{path: "gradle.properties", reason: "GLOBAL_CHANGE"},
		{path: "conventions/java/compiler.conf", reason: "GLOBAL_CHANGE"},
		{path: "unowned/file.txt", reason: "UNKNOWN_CHANGE_PATH"},
		{path: "../outside", reason: "INVALID_CHANGE_PATH"},
	} {
		t.Run(test.path, func(t *testing.T) {
			decision := EvaluateImpact(manifest, graph, []string{test.path})
			if decision.Mode != DecisionFullGraph || decision.Reason != test.reason || decision.SelectionAuthorized {
				t.Fatalf("decision = %+v", decision)
			}
		})
	}
}

func TestIncompleteOrUnknownGraphUsesFullGraph(t *testing.T) {
	manifest, graph := validGraph(t)
	graph.Complete = false
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "GRAPH_INCOMPLETE" {
		t.Fatalf("incomplete decision = %+v", decision)
	}
	_, graph = validGraph(t)
	graph.Projects[0].UnknownRelationships = true
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "UNKNOWN_RELATIONSHIP" {
		t.Fatalf("unknown decision = %+v", decision)
	}
	_, graph = validGraph(t)
	graph.Entrypoints[2].UnknownRelationships = true
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "UNKNOWN_RELATIONSHIP" {
		t.Fatalf("unknown entrypoint decision = %+v", decision)
	}
}

func TestAlternativeMustCoverAffectedProjectsArtifactsAndBuildChecks(t *testing.T) {
	manifest, graph := validGraph(t)
	for _, mutate := range []func(*DeclaredGraph){
		func(value *DeclaredGraph) {
			value.Entrypoints[1].ReachesProjects = []string{":service-a"}
			value.Entrypoints[2].ReachesProjects = []string{":service-a"}
		},
		func(value *DeclaredGraph) { value.Entrypoints[2].ArtifactIDs = nil },
		func(value *DeclaredGraph) { value.Entrypoints[2].CheckIDs = nil },
		func(value *DeclaredGraph) { value.Entrypoints[2].ContainsTestTasks = true },
	} {
		candidate := graph
		candidate.Entrypoints = append([]DeclaredEntrypoint(nil), graph.Entrypoints...)
		mutate(&candidate)
		decision := EvaluateImpact(manifest, candidate, []string{"library-c/src/Library.java"})
		if decision.Mode != DecisionFullGraph || decision.Reason != "NO_AUTHORIZED_ALTERNATIVE" {
			t.Fatalf("decision = %+v", decision)
		}
	}
}

func TestPOCOutputScopeDoesNotWeakenProductionAffectedClosure(t *testing.T) {
	manifest, graph := validGraph(t)
	graph.Projects = append(graph.Projects, Project{
		Path: ":consumer", SourcePaths: []string{"consumer/**"},
		DependsOn: []string{":service-b"},
	})
	graph.Entrypoints[0].ReachesProjects = append(
		graph.Entrypoints[0].ReachesProjects,
		":consumer",
	)

	changed := []string{"service-b/src/main/java/Service.java"}
	production := EvaluateImpact(manifest, graph, changed)
	if production.Mode != DecisionFullGraph || production.Reason != "NO_AUTHORIZED_ALTERNATIVE" {
		t.Fatalf("production decision = %+v", production)
	}
	poc := evaluatePOCImpact(manifest, graph, changed)
	if poc.Mode != DecisionShadowAlternative ||
		poc.Reason != "CUSTOMER_ALTERNATIVE_AND_DECLARED_OUTPUT_SCOPE" ||
		poc.PredictedAlternativeID != "service-b" ||
		!reflect.DeepEqual(poc.AffectedProjects, []string{":consumer", ":service-b"}) {
		t.Fatalf("POC decision = %+v", poc)
	}
}

func TestPOCOutputScopeUsesMostSpecificNestedProjectOwner(t *testing.T) {
	manifest, graph := validGraph(t)
	graph.Projects[1].SourcePaths = []string{"services/service-a/**"}
	graph.Projects = append(graph.Projects, Project{
		Path:        ":services",
		SourcePaths: []string{"services/**"},
		DependsOn:   []string{},
	})
	graph.Entrypoints[0].ReachesProjects = append(graph.Entrypoints[0].ReachesProjects, ":services")

	changed := []string{"services/service-a/src/main/java/Service.java"}
	production := EvaluateImpact(manifest, graph, changed)
	if production.Mode != DecisionFullGraph || production.Reason != "NO_AUTHORIZED_ALTERNATIVE" {
		t.Fatalf("production decision = %+v", production)
	}
	poc := evaluatePOCImpact(manifest, graph, changed)
	if poc.Mode != DecisionShadowAlternative ||
		poc.Reason != "CUSTOMER_ALTERNATIVE_AND_DECLARED_OUTPUT_SCOPE" ||
		poc.PredictedAlternativeID != "service-a" {
		t.Fatalf("POC decision = %+v", poc)
	}
}

func TestInvalidGraphFailsClosed(t *testing.T) {
	manifest, graph := validGraph(t)
	graph.Projects[0].DependsOn = []string{":service-a"}
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "GRAPH_INVALID" {
		t.Fatalf("cycle decision = %+v", decision)
	}
	_, graph = validGraph(t)
	graph.Entrypoints = graph.Entrypoints[:len(graph.Entrypoints)-1]
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "GRAPH_INVALID" {
		t.Fatalf("missing entrypoint decision = %+v", decision)
	}
	_, graph = validGraph(t)
	graph.Entrypoints[0].CheckIDs = []string{"jvm-tests"}
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "GRAPH_INVALID" {
		t.Fatalf("Test-owned check decision = %+v", decision)
	}
}

func TestDoubleStarMatchesNestedAndZeroSegments(t *testing.T) {
	for _, candidate := range []string{"gradle/file.txt", "gradle/nested/file.txt"} {
		if !matchRepositoryGlob("gradle/**", candidate) {
			t.Fatalf("double-star did not match %q", candidate)
		}
	}
	if !matchRepositoryGlob("**/build.gradle.kts", "service-a/build.gradle.kts") || matchRepositoryGlob("service-a/*", "service-a/nested/file") {
		t.Fatal("segment glob semantics are incorrect")
	}
}

func TestNoChangesAndNoCandidateRemainFullGraph(t *testing.T) {
	manifest, graph := validGraph(t)
	if decision := EvaluateImpact(manifest, graph, nil); decision.Reason != "NO_DECLARED_CHANGES" {
		t.Fatalf("no-change decision = %+v", decision)
	}
	graph.Entrypoints[3].ArtifactIDs = nil
	if decision := EvaluateImpact(manifest, graph, []string{"service-b/src/App.java"}); decision.Reason != "NO_AUTHORIZED_ALTERNATIVE" {
		t.Fatalf("no-candidate decision = %+v", decision)
	}
}

func TestCheckedInManifestAndGraphProduceConservativeShadow(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	manifest, err := LoadRepositoryManifest(repositoryRoot, filepath.FromSlash("fixtures/build-impact/manifest.v1.json"), "tonyredondo/buildopt", "pull-request")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash("fixtures/build-impact/declared-graph.v1.json")))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ParseDeclaredGraph(raw, manifest)
	if err != nil {
		t.Fatal(err)
	}
	decision := EvaluateImpact(manifest, graph.Graph, []string{"jvm/patcher/src/main/java/dev/buildopt/patcher/PatchBundle.java"})
	if decision.Mode != DecisionShadowAlternative || decision.PredictedAlternativeID != "jvm-components" || decision.SelectionAuthorized {
		t.Fatalf("decision = %+v", decision)
	}
	if !reflect.DeepEqual(decision.ExecutableEntrypoints, []string{"assemble"}) || !reflect.DeepEqual(decision.OmittedProjects, []string{":fixtures:golden-lane"}) || !reflect.DeepEqual(decision.PreservedTestCheckIDs, []string{"jvm-tests"}) {
		t.Fatalf("conservative boundary = %+v", decision)
	}
}

func TestDeclaredGraphRequiresExplicitSecurityFields(t *testing.T) {
	manifest, graph := validGraph(t)
	raw := string(encodeGraph(t, graph))
	for _, replacement := range []struct {
		name string
		old  string
		new  string
	}{
		{name: "complete", old: `"complete":true,`, new: ""},
		{name: "project unknown", old: `,"unknownRelationships":false}`, new: `}`},
		{name: "project dependencies", old: `"dependsOn":[],`, new: ""},
		{name: "entrypoint tests", old: `"containsTestTasks":false,`, new: ""},
		{name: "entrypoint artifacts", old: `"artifactIds":["distribution"],`, new: ""},
		{name: "global paths", old: `,"globalChangePaths":["conventions/**"]`, new: ""},
	} {
		t.Run(replacement.name, func(t *testing.T) {
			candidate := strings.Replace(raw, replacement.old, replacement.new, 1)
			if candidate == raw {
				t.Fatal("test did not remove its field")
			}
			if _, err := ParseDeclaredGraph([]byte(candidate), manifest); err == nil {
				t.Fatal("graph with missing security field accepted")
			}
		})
	}
}
