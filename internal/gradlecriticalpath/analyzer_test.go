package gradlecriticalpath

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeFindsLongestDependencyChain(t *testing.T) {
	root := t.TempDir()
	graph := filepath.Join(root, "graph.jsonl")
	operations := filepath.Join(root, "operations-log.txt")
	writeLines(t, graph, GraphDocument{
		SchemaVersion: GraphSchema, BuildPath: ":",
		Tasks: []GraphTask{
			{Identity: ":a", Path: ":a", TaskClass: "A"},
			{Identity: ":b", Path: ":b", TaskClass: "B", Dependencies: []string{":a"}},
			{Identity: ":c", Path: ":c", TaskClass: "C"},
			{Identity: ":d", Path: ":d", TaskClass: "D", Dependencies: []string{":b", ":c"}},
		},
	})
	writeOperation(t, operations, 1, 100, 110, ":", ":a", "A", nil)
	writeOperation(t, operations, 2, 110, 130, ":", ":b", "B", nil)
	writeOperation(t, operations, 3, 100, 125, ":", ":c", "C", map[string]interface{}{"skipMessage": "FROM-CACHE"})
	writeOperation(t, operations, 4, 130, 135, ":", ":d", "D", nil)

	report, err := Analyze(operations, graph, "candidate")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if report.Summary.MainBuildCriticalPathMs != 35 ||
		!equal(report.Summary.MainBuildCriticalPathTasks, []string{":a", ":b", ":d"}) ||
		report.Summary.Outcomes["FROM-CACHE"] != 1 || report.Summary.TaskCount != 4 {
		t.Fatalf("unexpected report: %+v", report)
	}
	critical := make(map[string]bool)
	for _, task := range report.Tasks {
		critical[task.Identity] = task.CriticalPath
	}
	if !critical[":a"] || !critical[":b"] || !critical[":d"] || critical[":c"] {
		t.Fatalf("critical membership: %+v", critical)
	}
}

func TestAnalyzeRejectsCycleAndIncompleteOperations(t *testing.T) {
	root := t.TempDir()
	graph := filepath.Join(root, "graph.jsonl")
	operations := filepath.Join(root, "operations-log.txt")
	writeLines(t, graph, GraphDocument{
		SchemaVersion: GraphSchema, BuildPath: ":",
		Tasks: []GraphTask{
			{Identity: ":a", Path: ":a", TaskClass: "A", Dependencies: []string{":b"}},
			{Identity: ":b", Path: ":b", TaskClass: "B", Dependencies: []string{":a"}},
		},
	})
	writeOperation(t, operations, 1, 1, 2, ":", ":a", "A", nil)
	writeOperation(t, operations, 2, 2, 3, ":", ":b", "B", nil)
	if _, err := Analyze(operations, graph, "control"); err == nil {
		t.Fatal("cyclic graph was accepted")
	}

	incomplete := filepath.Join(root, "incomplete-log.txt")
	writeJSONLine(t, incomplete, operationRecord{
		ID: 1, StartTime: pointer(int64(1)),
		DetailsClassName: "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails",
		Details:          operationDetails{BuildPath: ":", TaskPath: ":a", TaskClass: "A"},
	})
	if _, err := Analyze(incomplete, graph, "control"); err == nil {
		t.Fatal("incomplete trace was accepted")
	}
}

func writeOperation(t *testing.T, path string, id, start, end int64, buildPath, taskPath, taskClass string, result map[string]interface{}) {
	t.Helper()
	writeJSONLine(t, path, operationRecord{
		ID: id, StartTime: &start,
		DetailsClassName: "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails",
		Details:          operationDetails{BuildPath: buildPath, TaskPath: taskPath, TaskClass: taskClass},
	})
	if result == nil {
		result = map[string]interface{}{"actionable": true}
	}
	writeJSONLine(t, path, operationRecord{
		ID: id, EndTime: &end,
		ResultClassName: "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationResult",
		Result:          result,
	})
}

func writeLines(t *testing.T, path string, values ...interface{}) {
	t.Helper()
	for _, value := range values {
		writeJSONLine(t, path, value)
	}
}

func writeJSONLine(t *testing.T, path string, value interface{}) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func pointer[T any](value T) *T { return &value }

func equal(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
