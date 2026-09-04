package wcncpmateriality

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationMateriality(t *testing.T) {
	directory := t.TempDir()
	operations := filepath.Join(directory, "operations.jsonl")
	graph := filepath.Join(directory, "graph.jsonl")
	write(t, operations, `{"displayName":"Run build","detailsClassName":"x.RunAsBuildOperationBuildActionExecutor","id":1,"startTime":1000}
{"displayName":"Load build","id":2,"parentId":1,"startTime":1100}
{"id":2,"endTime":1300}
{"displayName":"Configure build","id":3,"parentId":1,"startTime":1350}
{"id":3,"endTime":1850}
{"displayName":"Calculate build tree task graph","id":4,"parentId":1,"startTime":1900}
{"id":4,"endTime":2200}
{"id":1,"endTime":11000}
`)
	write(t, graph, `{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[{"identity":":ok","path":":ok","taskClass":"Ok","dependencies":[]}]}
`)
	report, err := Analyze(operations, graph, "example", "CONFIGURATION_CACHE_UNLOCK", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.WorkflowMs != 10000 || report.CriticalPathContributionMs != 1000 || report.MaterialPercent != 10 || !report.MinimumMillisecondsPassed || !report.MinimumPercentPassed {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestConfigurationIntervalsAreNotDoubleCounted(t *testing.T) {
	directory := t.TempDir()
	operations := filepath.Join(directory, "operations.jsonl")
	graph := filepath.Join(directory, "graph.jsonl")
	write(t, operations, `{"displayName":"Run build","detailsClassName":"x.RunAsBuildOperationBuildActionExecutor","id":1,"startTime":0}
{"displayName":"Load build","id":2,"parentId":1,"startTime":100}
{"id":2,"endTime":500}
{"displayName":"Configure build","id":3,"parentId":1,"startTime":400}
{"id":3,"endTime":800}
{"displayName":"Calculate build tree task graph","id":4,"parentId":1,"startTime":900}
{"id":4,"endTime":1000}
{"id":1,"endTime":10000}
`)
	write(t, graph, "unused\n")
	report, err := Analyze(operations, graph, "example", "CONFIGURATION_CACHE_UNLOCK", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.CriticalPathContributionMs != 800 {
		t.Fatalf("contribution = %d", report.CriticalPathContributionMs)
	}
}

func TestRejectsIncompleteTrace(t *testing.T) {
	directory := t.TempDir()
	operations := filepath.Join(directory, "operations.jsonl")
	graph := filepath.Join(directory, "graph.jsonl")
	write(t, operations, `{"displayName":"Run build","detailsClassName":"x.RunAsBuildOperationBuildActionExecutor","id":1,"startTime":0}
{"id":1,"endTime":1000}
`)
	write(t, graph, "unused\n")
	if _, err := Analyze(operations, graph, "example", "CONFIGURATION_CACHE_UNLOCK", ""); err == nil {
		t.Fatal("incomplete trace was accepted")
	}
}

func TestCriticalTaskClassMateriality(t *testing.T) {
	directory := t.TempDir()
	operations := filepath.Join(directory, "operations.jsonl")
	graph := filepath.Join(directory, "graph.jsonl")
	write(t, operations, `{"displayName":"Run build","detailsClassName":"x.RunAsBuildOperationBuildActionExecutor","id":1,"startTime":0}
{"displayName":"Load build","id":2,"parentId":1,"startTime":100}
{"id":2,"endTime":200}
{"displayName":"Configure build","id":3,"parentId":1,"startTime":250}
{"id":3,"endTime":350}
{"displayName":"Calculate build tree task graph","id":4,"parentId":1,"startTime":400}
{"id":4,"endTime":500}
{"id":5,"startTime":1000,"detailsClassName":"org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails","details":{"buildPath":":","taskPath":":generate","taskClass":"example.Generator"}}
{"id":5,"endTime":1800,"resultClassName":"org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationResult","result":{"skipMessage":""}}
{"id":1,"endTime":10000}
`)
	write(t, graph, `{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[{"identity":":generate","path":":generate","taskClass":"example.Generator","dependencies":[]}]}
`)
	report, err := Analyze(operations, graph, "original-name", "CRITICAL_TASK_CLASS", "example.Generator")
	if err != nil {
		t.Fatal(err)
	}
	if report.CriticalPathContributionMs != 800 || report.MaterialPercent != 8 || len(report.MatchedTasks) != 1 || !report.MatchedTasks[0].CriticalPath {
		t.Fatalf("unexpected task report: %+v", report)
	}
	renamed, err := Analyze(operations, graph, "renamed-family", "CRITICAL_TASK_CLASS", "example.Generator")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.CriticalPathContributionMs != report.CriticalPathContributionMs || renamed.MaterialPercent != report.MaterialPercent {
		t.Fatal("family label changed materiality")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
