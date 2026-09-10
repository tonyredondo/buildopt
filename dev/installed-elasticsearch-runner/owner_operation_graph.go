package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

const ownerTaskPlanResult = "org.gradle.internal.build.BuildOperationFiringBuildWorkPreparer$PopulateWorkGraph$CalculateTaskGraphResult"
const ownerTaskStartDetails = "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails"

type ownerPlanIdentity struct {
	Type  string `json:"nodeType"`
	Build string `json:"buildPath"`
	ID    int64  `json:"taskId"`
	Path  string `json:"taskPath"`
}

func (n ownerPlanIdentity) identity() (string, error) {
	if n.Type != "TASK" || n.ID <= 0 || !strings.HasPrefix(n.Build, ":") || !strings.HasPrefix(n.Path, ":") || n.Path == ":" || strings.ContainsAny(n.Build+n.Path, "/\\\n\r") {
		return "", errors.New("invalid task-plan identity")
	}
	if n.Build == ":" {
		return n.Path, nil
	}
	return n.Build + n.Path, nil
}

// Included plugin builds can add work after their one whenReady notification.
// The pinned native operation trace retains each later task plan as well as
// every execution. Join those identities instead of inventing missing tasks.
func ownerOperationGraphs(path string) ([]gradlecriticalpath.GraphDocument, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	type node struct {
		Task         ownerPlanIdentity   `json:"task"`
		Identity     ownerPlanIdentity   `json:"nodeIdentity"`
		Dependencies []ownerPlanIdentity `json:"nodeDependencies"`
	}
	type start struct {
		Build string `json:"buildPath"`
		Path  string `json:"taskPath"`
		Class string `json:"taskClass"`
		ID    int64  `json:"taskId"`
	}
	nodes := map[string]node{}
	starts := map[string]start{}
	planIDs := map[int64]bool{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)
	for scanner.Scan() {
		var record struct {
			ID           int64  `json:"id"`
			DetailsClass string `json:"detailsClassName"`
			ResultClass  string `json:"resultClassName"`
			Details      start  `json:"details"`
			Result       struct {
				Plan []node `json:"taskPlan"`
			} `json:"result"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		if record.DetailsClass == ownerTaskStartDetails {
			d := record.Details
			key, err := (ownerPlanIdentity{Type: "TASK", Build: d.Build, Path: d.Path, ID: d.ID}).identity()
			if err != nil || d.Class == "" || starts[key].Class != "" {
				return nil, errors.New("missing or duplicate task execution identity")
			}
			starts[key] = d
		}
		if record.ResultClass == ownerTaskPlanResult {
			if record.ID <= 0 || planIDs[record.ID] {
				return nil, errors.New("duplicate task-plan operation")
			}
			planIDs[record.ID] = true
			for _, n := range record.Result.Plan {
				key, err := n.Identity.identity()
				if err != nil || n.Task != n.Identity {
					return nil, errors.New("task-plan node identity mismatch")
				}
				if _, ok := nodes[key]; ok {
					return nil, errors.New("task occurs in multiple execution plans")
				}
				nodes[key] = n
			}
		}
		if len(nodes) > 100000 || len(starts) > 100000 {
			return nil, errors.New("owner task bound exceeded")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(nodes) == 0 || len(nodes) != len(starts) {
		return nil, errors.New("task plans and executions differ")
	}
	builds := map[string][]gradlecriticalpath.GraphTask{}
	for key, n := range nodes {
		s, ok := starts[key]
		if !ok || s.ID != n.Identity.ID {
			return nil, errors.New("executed task does not match planned task id")
		}
		deps := []string{}
		seen := map[string]bool{}
		for _, dep := range n.Dependencies {
			d, err := dep.identity()
			if err != nil || seen[d] || nodes[d].Identity != dep {
				return nil, errors.New("missing, duplicate or mismatched task dependency")
			}
			seen[d] = true
			deps = append(deps, d)
		}
		sort.Strings(deps)
		builds[s.Build] = append(builds[s.Build], gradlecriticalpath.GraphTask{Identity: key, Path: s.Path, TaskClass: s.Class, Dependencies: deps})
	}
	result := []gradlecriticalpath.GraphDocument{}
	for build, tasks := range builds {
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].Identity < tasks[j].Identity })
		result = append(result, gradlecriticalpath.GraphDocument{SchemaVersion: gradlecriticalpath.GraphSchema, BuildPath: build, Tasks: tasks})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].BuildPath < result[j].BuildPath })
	return result, nil
}

func analyzeOwnerOperations(path string) (gradlecriticalpath.Report, error) {
	graphs, err := ownerOperationGraphs(path)
	if err != nil {
		return gradlecriticalpath.Report{}, err
	}
	// The derived DAG is temporary. Raw operation bytes remain sealed evidence;
	// every check reconstructs the same graph before using the existing analyzer.
	dir, err := os.MkdirTemp("", "buildopt-owner-graph-")
	if err != nil {
		return gradlecriticalpath.Report{}, err
	}
	defer os.RemoveAll(dir)
	graphPath := filepath.Join(dir, "graph.jsonl")
	file, err := os.OpenFile(graphPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return gradlecriticalpath.Report{}, err
	}
	encoder := json.NewEncoder(file)
	for _, g := range graphs {
		if err := encoder.Encode(g); err != nil {
			file.Close()
			return gradlecriticalpath.Report{}, err
		}
	}
	if err := file.Close(); err != nil {
		return gradlecriticalpath.Report{}, err
	}
	// The analyzer requires one completed native operation per graph task.
	return gradlecriticalpath.Analyze(path, graphPath, "control")
}
