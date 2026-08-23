package launcher

import (
	"errors"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
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
