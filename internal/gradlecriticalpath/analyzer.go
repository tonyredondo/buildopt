// Package gradlecriticalpath combines Gradle task-operation timings with the
// resolved task DAG to identify the dependency chain with the greatest
// cumulative task duration in each build.
package gradlecriticalpath

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	GraphSchema  = "buildopt.diagnostics/gradle-task-graph/v1"
	ReportSchema = "buildopt.diagnostics/gradle-task-critical-path/v1"
	maximumTasks = 100000
)

type GraphDocument struct {
	SchemaVersion string      `json:"schemaVersion"`
	BuildPath     string      `json:"buildPath"`
	Tasks         []GraphTask `json:"tasks"`
}

type GraphTask struct {
	Identity     string   `json:"identity"`
	Path         string   `json:"path"`
	TaskClass    string   `json:"taskClass"`
	Dependencies []string `json:"dependencies"`
}

type Task struct {
	Identity     string   `json:"identity"`
	BuildPath    string   `json:"buildPath"`
	Path         string   `json:"path"`
	TaskClass    string   `json:"taskClass"`
	StartTimeMs  int64    `json:"startTimeMs"`
	EndTimeMs    int64    `json:"endTimeMs"`
	DurationMs   int64    `json:"durationMs"`
	Outcome      string   `json:"outcome"`
	Dependencies []string `json:"dependencies"`
	CriticalPath bool     `json:"criticalPath"`
}

type BuildSummary struct {
	BuildPath              string   `json:"buildPath"`
	TaskCount              int      `json:"taskCount"`
	TaskExecutionSpanMs    int64    `json:"taskExecutionSpanMs"`
	CumulativeDurationMs   int64    `json:"cumulativeDurationMs"`
	CriticalPathDurationMs int64    `json:"criticalPathDurationMs"`
	CriticalPathTasks      []string `json:"criticalPathTasks"`
}

type Summary struct {
	TaskCount                  int            `json:"taskCount"`
	BuildCount                 int            `json:"buildCount"`
	CumulativeTaskDurationMs   int64          `json:"cumulativeTaskDurationMs"`
	MainBuildTaskSpanMs        int64          `json:"mainBuildTaskSpanMs"`
	MainBuildCriticalPathMs    int64          `json:"mainBuildCriticalPathMs"`
	MainBuildCriticalPathTasks []string       `json:"mainBuildCriticalPathTasks"`
	Outcomes                   map[string]int `json:"outcomes"`
}

type Boundaries struct {
	DiagnosticOnly       bool   `json:"diagnosticOnly"`
	ProductionAuthorized bool   `json:"productionAuthorized"`
	CriticalPathMethod   string `json:"criticalPathMethod"`
	TestOptimization     string `json:"testOptimization"`
}

type Report struct {
	SchemaVersion        string         `json:"schemaVersion"`
	Arm                  string         `json:"arm"`
	OperationTraceSHA256 string         `json:"operationTraceSha256"`
	TaskGraphSHA256      string         `json:"taskGraphSha256"`
	Builds               []BuildSummary `json:"builds"`
	Tasks                []Task         `json:"tasks"`
	Summary              Summary        `json:"summary"`
	Boundaries           Boundaries     `json:"boundaries"`
}

type operationRecord struct {
	ID               int64                  `json:"id"`
	StartTime        *int64                 `json:"startTime"`
	EndTime          *int64                 `json:"endTime"`
	DetailsClassName string                 `json:"detailsClassName"`
	ResultClassName  string                 `json:"resultClassName"`
	Details          operationDetails       `json:"details"`
	Result           map[string]interface{} `json:"result"`
	Failure          interface{}            `json:"failure"`
}

type operationDetails struct {
	BuildPath string `json:"buildPath"`
	TaskPath  string `json:"taskPath"`
	TaskClass string `json:"taskClass"`
}

type taskStart struct {
	BuildPath string
	Path      string
	TaskClass string
	Started   int64
}

type taskEnd struct {
	Ended   int64
	Result  map[string]interface{}
	Failure interface{}
}

// Analyze reads one Gradle operation trace and its matching resolved task DAG.
func Analyze(operationTracePath, taskGraphPath, arm string) (Report, error) {
	if arm != "control" && arm != "candidate" {
		return Report{}, errors.New("arm must be control or candidate")
	}
	traceDigest, err := fileSHA256(operationTracePath)
	if err != nil {
		return Report{}, fmt.Errorf("hash operation trace: %w", err)
	}
	graphDigest, err := fileSHA256(taskGraphPath)
	if err != nil {
		return Report{}, fmt.Errorf("hash task graph: %w", err)
	}
	graphs, err := readGraphs(taskGraphPath)
	if err != nil {
		return Report{}, err
	}
	starts, ends, err := readOperations(operationTracePath)
	if err != nil {
		return Report{}, err
	}
	tasks, err := joinTasks(graphs, starts, ends)
	if err != nil {
		return Report{}, err
	}
	builds, err := criticalPaths(tasks)
	if err != nil {
		return Report{}, err
	}
	critical := make(map[string]bool)
	for _, build := range builds {
		for _, identity := range build.CriticalPathTasks {
			critical[identity] = true
		}
	}
	outcomes := make(map[string]int)
	var cumulative int64
	for index := range tasks {
		tasks[index].CriticalPath = critical[tasks[index].Identity]
		outcomes[tasks[index].Outcome]++
		cumulative += tasks[index].DurationMs
	}
	main := BuildSummary{BuildPath: ":"}
	for _, build := range builds {
		if build.BuildPath == ":" {
			main = build
			break
		}
	}
	return Report{
		SchemaVersion: ReportSchema, Arm: arm,
		OperationTraceSHA256: traceDigest, TaskGraphSHA256: graphDigest,
		Builds: builds, Tasks: tasks,
		Summary: Summary{
			TaskCount: len(tasks), BuildCount: len(builds),
			CumulativeTaskDurationMs:   cumulative,
			MainBuildTaskSpanMs:        main.TaskExecutionSpanMs,
			MainBuildCriticalPathMs:    main.CriticalPathDurationMs,
			MainBuildCriticalPathTasks: main.CriticalPathTasks,
			Outcomes:                   outcomes,
		},
		Boundaries: Boundaries{
			DiagnosticOnly: true, ProductionAuthorized: false,
			CriticalPathMethod: "LONGEST_HARD_DEPENDENCY_CHAIN_BY_TASK_DURATION",
			TestOptimization:   "OUT_OF_SCOPE",
		},
	}, nil
}

func readGraphs(path string) (map[string]GraphTask, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open task graph: %w", err)
	}
	defer file.Close()
	graphs := make(map[string]GraphTask)
	builds := make(map[string]bool)
	err = scanJSONLines(file, func(raw []byte) error {
		var document GraphDocument
		if decodeErr := json.Unmarshal(raw, &document); decodeErr != nil {
			return fmt.Errorf("decode task graph: %w", decodeErr)
		}
		if document.SchemaVersion != GraphSchema || !validBuildPath(document.BuildPath) ||
			len(document.Tasks) == 0 || builds[document.BuildPath] {
			return errors.New("invalid or duplicate task graph document")
		}
		builds[document.BuildPath] = true
		for _, task := range document.Tasks {
			if task.Identity == "" || task.Path == "" || task.TaskClass == "" ||
				graphs[task.Identity].Identity != "" {
				return errors.New("invalid or duplicate task graph task")
			}
			graphs[task.Identity] = task
			if len(graphs) > maximumTasks {
				return errors.New("task graph exceeds task bound")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(graphs) == 0 {
		return nil, errors.New("task graph is empty")
	}
	return graphs, nil
}

func readOperations(path string) (map[int64]taskStart, map[int64]taskEnd, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open operation trace: %w", err)
	}
	defer file.Close()
	starts := make(map[int64]taskStart)
	ends := make(map[int64]taskEnd)
	err = scanJSONLines(file, func(raw []byte) error {
		var record operationRecord
		if decodeErr := json.Unmarshal(raw, &record); decodeErr != nil {
			return fmt.Errorf("decode operation trace: %w", decodeErr)
		}
		if record.StartTime != nil && record.DetailsClassName ==
			"org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails" {
			if starts[record.ID].Path != "" || record.Details.TaskPath == "" ||
				!validBuildPath(record.Details.BuildPath) {
				return errors.New("invalid or duplicate task start operation")
			}
			starts[record.ID] = taskStart{
				BuildPath: record.Details.BuildPath, Path: record.Details.TaskPath,
				TaskClass: record.Details.TaskClass, Started: *record.StartTime,
			}
		}
		if record.EndTime != nil && record.ResultClassName ==
			"org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationResult" {
			if _, exists := ends[record.ID]; exists {
				return errors.New("duplicate task end operation")
			}
			ends[record.ID] = taskEnd{Ended: *record.EndTime, Result: record.Result, Failure: record.Failure}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(starts) == 0 || len(starts) > maximumTasks || len(ends) != len(starts) {
		return nil, nil, errors.New("operation trace has incomplete task operations")
	}
	return starts, ends, nil
}

func joinTasks(graphs map[string]GraphTask, starts map[int64]taskStart, ends map[int64]taskEnd) ([]Task, error) {
	tasks := make([]Task, 0, len(starts))
	seen := make(map[string]bool)
	for id, start := range starts {
		end, ok := ends[id]
		if !ok || end.Ended < start.Started {
			return nil, errors.New("task operation end is missing or precedes its start")
		}
		identity := taskIdentity(start.BuildPath, start.Path)
		graph, ok := graphs[identity]
		if !ok || seen[identity] {
			return nil, fmt.Errorf("executed task is absent or duplicated in graph: %s", identity)
		}
		seen[identity] = true
		tasks = append(tasks, Task{
			Identity: identity, BuildPath: start.BuildPath, Path: start.Path,
			TaskClass: start.TaskClass, StartTimeMs: start.Started, EndTimeMs: end.Ended,
			DurationMs: end.Ended - start.Started, Outcome: taskOutcome(end),
			Dependencies: append([]string{}, graph.Dependencies...),
		})
	}
	if len(seen) != len(graphs) {
		return nil, errors.New("task graph contains tasks without completed operations")
	}
	sort.Slice(tasks, func(left, right int) bool { return tasks[left].Identity < tasks[right].Identity })
	return tasks, nil
}

func criticalPaths(tasks []Task) ([]BuildSummary, error) {
	byIdentity := make(map[string]Task, len(tasks))
	byBuild := make(map[string][]string)
	for _, task := range tasks {
		byIdentity[task.Identity] = task
		byBuild[task.BuildPath] = append(byBuild[task.BuildPath], task.Identity)
	}
	buildPaths := make([]string, 0, len(byBuild))
	for buildPath := range byBuild {
		buildPaths = append(buildPaths, buildPath)
	}
	sort.Strings(buildPaths)
	builds := make([]BuildSummary, 0, len(buildPaths))
	for _, buildPath := range buildPaths {
		memo := make(map[string]pathScore)
		visiting := make(map[string]bool)
		var best pathScore
		var first, last int64
		var cumulative int64
		for index, identity := range byBuild[buildPath] {
			task := byIdentity[identity]
			if index == 0 || task.StartTimeMs < first {
				first = task.StartTimeMs
			}
			if task.EndTimeMs > last {
				last = task.EndTimeMs
			}
			cumulative += task.DurationMs
			score, err := longestPath(identity, buildPath, byIdentity, memo, visiting)
			if err != nil {
				return nil, err
			}
			if betterPath(score, best) {
				best = score
			}
		}
		builds = append(builds, BuildSummary{
			BuildPath: buildPath, TaskCount: len(byBuild[buildPath]),
			TaskExecutionSpanMs: last - first, CumulativeDurationMs: cumulative,
			CriticalPathDurationMs: best.Duration, CriticalPathTasks: best.Tasks,
		})
	}
	return builds, nil
}

type pathScore struct {
	Duration int64
	Tasks    []string
}

func longestPath(identity, buildPath string, tasks map[string]Task, memo map[string]pathScore, visiting map[string]bool) (pathScore, error) {
	if score, ok := memo[identity]; ok {
		return score, nil
	}
	if visiting[identity] {
		return pathScore{}, errors.New("task dependency graph contains a cycle")
	}
	visiting[identity] = true
	task := tasks[identity]
	var best pathScore
	for _, dependency := range task.Dependencies {
		candidate, ok := tasks[dependency]
		if !ok {
			return pathScore{}, fmt.Errorf("task dependency is absent from operations: %s", dependency)
		}
		if candidate.BuildPath != buildPath {
			continue
		}
		score, err := longestPath(dependency, buildPath, tasks, memo, visiting)
		if err != nil {
			return pathScore{}, err
		}
		if betterPath(score, best) {
			best = score
		}
	}
	delete(visiting, identity)
	best.Duration += task.DurationMs
	best.Tasks = append(append([]string(nil), best.Tasks...), identity)
	memo[identity] = best
	return best, nil
}

// betterPath keeps the structural chain meaningful when Gradle's millisecond
// trace resolution reports multiple task paths with the same duration.
func betterPath(candidate, current pathScore) bool {
	if len(candidate.Tasks) == 0 {
		return false
	}
	if len(current.Tasks) == 0 {
		return true
	}
	if candidate.Duration != current.Duration {
		return candidate.Duration > current.Duration
	}
	if len(candidate.Tasks) != len(current.Tasks) {
		return len(candidate.Tasks) > len(current.Tasks)
	}
	return strings.Join(candidate.Tasks, "\x00") < strings.Join(current.Tasks, "\x00")
}

func taskOutcome(end taskEnd) string {
	if end.Failure != nil {
		return "FAILED"
	}
	if message, ok := end.Result["skipMessage"].(string); ok && message != "" {
		return message
	}
	if end.Result["originBuildCacheKeyBytes"] != nil {
		return "FROM-CACHE"
	}
	if actionable, ok := end.Result["actionable"].(bool); ok && !actionable {
		return "NO-ACTIONS"
	}
	return "EXECUTED"
}

func taskIdentity(buildPath, taskPath string) string {
	if buildPath == ":" {
		return taskPath
	}
	return buildPath + taskPath
}

func validBuildPath(value string) bool {
	return strings.HasPrefix(value, ":") && !strings.ContainsAny(value, "\r\n\x00")
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func scanJSONLines(reader io.Reader, visit func([]byte) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		if len(strings.TrimSpace(scanner.Text())) == 0 {
			continue
		}
		if err := visit(scanner.Bytes()); err != nil {
			return err
		}
	}
	return scanner.Err()
}
