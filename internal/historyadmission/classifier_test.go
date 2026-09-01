package historyadmission

import (
	"reflect"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

func TestClassifyUsesExactOwnersAndDependencyClosure(t *testing.T) {
	snapshot := buildimpact.DiscoverySnapshot{Complete: true, Projects: []buildimpact.DiscoveredProject{
		{Path: ":library", SourcePaths: []string{"library/**"}},
		{Path: ":consumer", SourcePaths: []string{"consumer/**"}, DependsOn: []string{":library"}},
		{Path: ":leaf", SourcePaths: []string{"leaf/**"}},
	}}
	result, err := Classify(snapshot, []string{"library/src/main/java/Api.java"})
	if err != nil || result.Family != FamilyDependency || !reflect.DeepEqual(result.Owners, []string{":library"}) || !reflect.DeepEqual(result.AffectedProjects, []string{":consumer", ":library"}) {
		t.Fatalf("classification = %+v, %v", result, err)
	}
	resource, err := Classify(snapshot, []string{"leaf/src/main/resources/a.txt"})
	if err != nil || resource.Family != FamilyResource {
		t.Fatalf("resource = %+v, %v", resource, err)
	}
}

func TestClassifyRejectsIncompleteAndAmbiguousGraphs(t *testing.T) {
	if _, err := Classify(buildimpact.DiscoverySnapshot{}, []string{"a.java"}); err == nil {
		t.Fatal("incomplete graph accepted")
	}
	snapshot := buildimpact.DiscoverySnapshot{Complete: true, Projects: []buildimpact.DiscoveredProject{{Path: ":a", SourcePaths: []string{"shared/**"}}, {Path: ":b", SourcePaths: []string{"shared/**"}}}}
	if _, err := Classify(snapshot, []string{"shared/A.java"}); err == nil {
		t.Fatal("ambiguous owner accepted")
	}
	snapshot = buildimpact.DiscoverySnapshot{Complete: true, Entrypoints: []buildimpact.DiscoveredEntrypoint{{Name: "assemble", UnknownRelationships: true}}}
	if _, err := Classify(snapshot, []string{"shared/A.java"}); err == nil {
		t.Fatal("unknown entrypoint relationships accepted")
	}
}

func TestUnsafeStructuralChangeDoesNotUseRepositoryOrTaskNames(t *testing.T) {
	if !UnsafeStructuralChange([]string{"module/build.gradle.kts"}) ||
		!UnsafeStructuralChange([]string{"gradle/libs.versions.toml"}) {
		t.Fatal("build-logic change accepted")
	}
	if UnsafeStructuralChange([]string{"spring/src/main/java/Gradle.java", "kafka/src/main/resources/task.txt"}) {
		t.Fatal("source labels affected structural classification")
	}
}

func TestSnapshotFromDeclaredGraphPreservesClassificationFacts(t *testing.T) {
	graph := buildimpact.DeclaredGraph{Complete: true,
		Projects: []buildimpact.Project{
			{Path: ":library", SourcePaths: []string{"library/**"}},
			{Path: ":consumer", SourcePaths: []string{"consumer/**"}, DependsOn: []string{":library"}},
		},
		Entrypoints: []buildimpact.DeclaredEntrypoint{{Name: "classes", ReachesProjects: []string{":consumer", ":library"}}},
	}
	classification, err := Classify(SnapshotFromDeclaredGraph(graph), []string{"library/src/main/java/A.java"})
	if err != nil || classification.Family != FamilyDependency || !reflect.DeepEqual(classification.Owners, []string{":library"}) {
		t.Fatalf("classification = %+v, %v", classification, err)
	}
}
