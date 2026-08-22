package buildimpact

import (
	"reflect"
	"testing"
)

func TestWorkflowInputRelevanceAllowsOnlyCompleteUnconsumedUnownedPaths(t *testing.T) {
	snapshot := DiscoverySnapshot{Projects: []DiscoveredProject{
		{Path: ":module", SourcePaths: []string{"module/**"}},
	}}
	changed := []string{"CHANGELOG.md", "module/src/main/java/Example.java"}
	observation := WorkflowInputRelevance{
		SchemaVersion:   WorkflowInputRelevanceSchemaVersion,
		Complete:        true,
		FallbackReasons: []string{},
		Paths: []WorkflowInputPath{
			{Path: "CHANGELOG.md", ConsumingTasks: []string{}},
			{Path: "module/src/main/java/Example.java", ConsumingTasks: []string{":module:compileJava"}},
		},
	}
	owners, ignored, err := ResolveWorkflowProjectOwners(snapshot, observation, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(owners, []string{":module"}) ||
		!reflect.DeepEqual(ignored, []string{"CHANGELOG.md"}) {
		t.Fatalf("owners = %v, ignored = %v", owners, ignored)
	}

	consumed := observation
	consumed.Paths = append([]WorkflowInputPath(nil), observation.Paths...)
	consumed.Paths[0] = WorkflowInputPath{Path: "CHANGELOG.md", ConsumingTasks: []string{":module:processResources"}}
	if _, _, err := ResolveWorkflowProjectOwners(snapshot, consumed, changed); err == nil {
		t.Fatal("consumed unowned path was ignored")
	}

	incomplete := observation
	incomplete.Complete = false
	incomplete.FallbackReasons = []string{"TASK_INPUTS_UNAVAILABLE"}
	if _, _, err := ResolveWorkflowProjectOwners(snapshot, incomplete, changed); err == nil {
		t.Fatal("incomplete workflow-input evidence was accepted")
	}
}

func TestWorkflowInputRelevanceNeverSuppressesAmbiguousOrAllUnownedChanges(t *testing.T) {
	ambiguous := DiscoverySnapshot{Projects: []DiscoveredProject{
		{Path: ":left", SourcePaths: []string{"shared/**"}},
		{Path: ":right", SourcePaths: []string{"shared/**"}},
	}}
	observation := WorkflowInputRelevance{
		SchemaVersion:   WorkflowInputRelevanceSchemaVersion,
		Complete:        true,
		FallbackReasons: []string{},
		Paths: []WorkflowInputPath{
			{Path: "CHANGELOG.md", ConsumingTasks: []string{}},
			{Path: "shared/Example.java", ConsumingTasks: []string{}},
		},
	}
	if _, _, err := ResolveWorkflowProjectOwners(
		ambiguous, observation, []string{"CHANGELOG.md", "shared/Example.java"},
	); err == nil {
		t.Fatal("ambiguous project ownership was suppressed")
	}

	allUnowned := observation
	allUnowned.Paths = []WorkflowInputPath{{Path: "CHANGELOG.md", ConsumingTasks: []string{}}}
	if _, _, err := ResolveWorkflowProjectOwners(
		DiscoverySnapshot{Projects: []DiscoveredProject{{Path: ":module", SourcePaths: []string{"module/**"}}}},
		allUnowned,
		[]string{"CHANGELOG.md"},
	); err == nil {
		t.Fatal("change set without an owned project was accepted")
	}
}

func TestParseWorkflowInputRelevanceBindsExactChangedPaths(t *testing.T) {
	raw := []byte(`{
	  "schemaVersion":"buildopt.build-impact/workflow-input-relevance/v1",
	  "complete":true,
	  "fallbackReasons":[],
	  "paths":[
	    {"path":"module/src/main/java/Example.java","consumingTasks":[":module:compileJava"]},
	    {"path":"CHANGELOG.md","consumingTasks":[]}
	  ]
	}`)
	observation, err := ParseWorkflowInputRelevance(
		raw,
		[]string{"CHANGELOG.md", "module/src/main/java/Example.java"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Paths[0].Path != "CHANGELOG.md" {
		t.Fatalf("canonical paths = %+v", observation.Paths)
	}
	for _, invalid := range [][]string{
		{"CHANGELOG.md"},
		{"CHANGELOG.md", "other.txt"},
		{"../CHANGELOG.md", "module/src/main/java/Example.java"},
	} {
		if _, err := ParseWorkflowInputRelevance(raw, invalid); err == nil {
			t.Fatalf("changed paths %v were accepted", invalid)
		}
	}

	unknownFallback := []byte(`{
          "schemaVersion":"buildopt.build-impact/workflow-input-relevance/v1",
          "complete":false,
          "fallbackReasons":["UNKNOWN_REASON"],
          "paths":[
            {"path":"CHANGELOG.md","consumingTasks":[]},
            {"path":"module/src/main/java/Example.java","consumingTasks":[]}
          ]
        }`)
	if _, err := ParseWorkflowInputRelevance(unknownFallback, []string{"CHANGELOG.md", "module/src/main/java/Example.java"}); err == nil {
		t.Fatal("unknown workflow-input fallback reason was accepted")
	}
}
