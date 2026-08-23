package launcher

import (
	"errors"
	"fmt"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/nativevolatility"
)

func TestOptimizeTaskLineageReturnsSortedTransitiveAncestors(t *testing.T) {
	lineage, err := newOptimizeTaskLineage([]buildimpact.DiscoveredTask{
		{Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{":app:classes", ":lib:jar"}},
		{Path: ":app:classes", ProjectPath: ":app", DependsOn: []string{":app:compileJava"}},
		{Path: ":app:compileJava", ProjectPath: ":app", DependsOn: []string{}},
		{Path: ":lib:jar", ProjectPath: ":lib", DependsOn: []string{":lib:compileJava"}},
		{Path: ":lib:compileJava", ProjectPath: ":lib", DependsOn: []string{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ancestors, err := lineage.ancestors([]string{":app:jar"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{":app:classes", ":app:compileJava", ":lib:compileJava", ":lib:jar"}
	if !equalOptimizeStrings(ancestors, want) {
		t.Fatalf("transitive ancestors = %v, want %v", ancestors, want)
	}
}

func TestOptimizeTaskLineageRejectsIncompleteAmbiguousAndCyclicGraphs(t *testing.T) {
	tests := []struct {
		name  string
		tasks []buildimpact.DiscoveredTask
		want  error
	}{
		{
			name: "missing dependency",
			tasks: []buildimpact.DiscoveredTask{{
				Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{":app:classes"},
			}},
			want: errOptimizeTaskLineageIncomplete,
		},
		{
			name: "duplicate task",
			tasks: []buildimpact.DiscoveredTask{
				{Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{}},
				{Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{}},
			},
			want: errOptimizeTaskLineageAmbiguous,
		},
		{
			name: "cycle",
			tasks: []buildimpact.DiscoveredTask{
				{Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{":app:classes"}},
				{Path: ":app:classes", ProjectPath: ":app", DependsOn: []string{":app:jar"}},
			},
			want: errOptimizeTaskLineageCyclic,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newOptimizeTaskLineage(test.tasks)
			if !errors.Is(err, test.want) {
				t.Fatalf("lineage error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestOptimizeTaskLineageRejectsUnknownDirectProducer(t *testing.T) {
	lineage, err := newOptimizeTaskLineage([]buildimpact.DiscoveredTask{{
		Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lineage.ancestors([]string{":app:missing"}); !errors.Is(err, errOptimizeTaskLineageIncomplete) {
		t.Fatalf("unknown producer error = %v", err)
	}
}

func TestOptimizeTaskLineageBuildsMinimalDirectProducerFrontier(t *testing.T) {
	lineage, err := newOptimizeTaskLineage([]buildimpact.DiscoveredTask{
		{Path: ":app:assemble", ProjectPath: ":app", DependsOn: []string{":app:jar", ":app:javadocJar"}},
		{Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{":app:classes"}},
		{Path: ":app:javadocJar", ProjectPath: ":app", DependsOn: []string{":app:javadoc"}},
		{Path: ":app:classes", ProjectPath: ":app", DependsOn: []string{}},
		{Path: ":app:javadoc", ProjectPath: ":app", DependsOn: []string{}},
		{Path: ":custom:archive", ProjectPath: ":custom", DependsOn: []string{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := lineage.rebuildEntrypoints([]nativevolatility.Entry{
		{Path: "app/build/libs/app.jar", ProducerTasks: []string{":app:jar"}},
		{Path: "app/build/libs/app-javadoc.jar", ProducerTasks: []string{":app:javadocJar"}},
		{Path: "custom/build/archive.bin", ProducerTasks: []string{":custom:archive"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{":app:jar", ":app:javadocJar", ":custom:archive"}
	if !equalOptimizeStrings(frontier, want) {
		t.Fatalf("rebuild frontier = %v, want %v", frontier, want)
	}
}

func TestOptimizeTaskLineageRemovesAlreadyCoveredAndDominatedProducers(t *testing.T) {
	lineage, err := newOptimizeTaskLineage([]buildimpact.DiscoveredTask{
		{Path: ":app:assemble", ProjectPath: ":app", DependsOn: []string{":app:jar"}},
		{Path: ":app:jar", ProjectPath: ":app", DependsOn: []string{":app:classes"}},
		{Path: ":app:classes", ProjectPath: ":app", DependsOn: []string{":app:compileJava"}},
		{Path: ":app:compileJava", ProjectPath: ":app", DependsOn: []string{}},
		{Path: ":other:jar", ProjectPath: ":other", DependsOn: []string{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := lineage.rebuildEntrypoints([]nativevolatility.Entry{
		{Path: "app/build/classes/App.class", ProducerTasks: []string{":app:compileJava"}},
		{Path: "app/build/libs/app.jar", ProducerTasks: []string{":app:jar"}},
		{Path: "other/build/libs/other.jar", ProducerTasks: []string{":other:jar"}},
	}, []string{":app:assemble"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{":other:jar"}
	if !equalOptimizeStrings(frontier, want) {
		t.Fatalf("minimal rebuild frontier = %v, want %v", frontier, want)
	}
}

func TestOptimizeQuarantineCandidateEntrypointBound(t *testing.T) {
	entrypoints := make([]string, maximumStructuralAlternativeEntrypoints)
	for index := range entrypoints {
		entrypoints[index] = fmt.Sprintf(":project-%d:jar", index)
	}
	if err := validateOptimizeQuarantineCandidateEntrypoints(entrypoints); err != nil {
		t.Fatalf("bounded quarantine frontier rejected: %v", err)
	}
	entrypoints = append(entrypoints, ":overflow:jar")
	if err := validateOptimizeQuarantineCandidateEntrypoints(entrypoints); err == nil {
		t.Fatal("oversized quarantine frontier accepted")
	}
}
