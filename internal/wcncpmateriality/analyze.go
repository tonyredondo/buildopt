// Package wcncpmateriality reconstructs controlled critical-path materiality
// from Gradle operation traces and their matching resolved task graphs.
package wcncpmateriality

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

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

const ReportSchema = "buildopt.wcncp/controlled-materiality-run/v1"

type operation struct {
	ID               int64  `json:"id"`
	ParentID         *int64 `json:"parentId"`
	StartTime        *int64 `json:"startTime"`
	EndTime          *int64 `json:"endTime"`
	DisplayName      string `json:"displayName"`
	DetailsClassName string `json:"detailsClassName"`
}

type interval struct {
	start int64
	end   int64
}

type MatchedTask struct {
	Identity     string `json:"identity"`
	DurationMs   int64  `json:"durationMs"`
	CriticalPath bool   `json:"criticalPath"`
	Outcome      string `json:"outcome"`
}

type Report struct {
	SchemaVersion              string        `json:"schemaVersion"`
	Family                     string        `json:"family"`
	Method                     string        `json:"method"`
	TaskClass                  string        `json:"taskClass,omitempty"`
	OperationTraceSHA256       string        `json:"operationTraceSha256"`
	TaskGraphSHA256            string        `json:"taskGraphSha256"`
	WorkflowMs                 int64         `json:"workflowMs"`
	CriticalPathContributionMs int64         `json:"criticalPathContributionMs"`
	MaterialPercent            float64       `json:"materialPercent"`
	MatchedTasks               []MatchedTask `json:"matchedTasks,omitempty"`
	MinimumMillisecondsPassed  bool          `json:"minimumMillisecondsPassed"`
	MinimumPercentPassed       bool          `json:"minimumPercentPassed"`
	EnvironmentClass           string        `json:"environmentClass"`
	PerformanceGateAuthority   bool          `json:"performanceGateAuthority"`
}

// Analyze produces one independently reconstructable controlled materiality row.
func Analyze(operationPath, graphPath, family, method, taskClass string) (Report, error) {
	if family == "" {
		return Report{}, errors.New("family is required")
	}
	if method != "CONFIGURATION_CACHE_UNLOCK" && method != "CRITICAL_TASK_CLASS" {
		return Report{}, errors.New("unsupported materiality method")
	}
	if method == "CRITICAL_TASK_CLASS" && taskClass == "" {
		return Report{}, errors.New("task class is required")
	}
	operationSHA, err := fileSHA256(operationPath)
	if err != nil {
		return Report{}, fmt.Errorf("hash operation trace: %w", err)
	}
	graphSHA, err := fileSHA256(graphPath)
	if err != nil {
		return Report{}, fmt.Errorf("hash task graph: %w", err)
	}
	workflowMs, configurationMs, err := operationDurations(operationPath)
	if err != nil {
		return Report{}, err
	}
	contribution := configurationMs
	var matched []MatchedTask
	if method == "CRITICAL_TASK_CLASS" {
		critical, analyzeErr := gradlecriticalpath.Analyze(operationPath, graphPath, "control")
		if analyzeErr != nil {
			return Report{}, fmt.Errorf("analyze task critical path: %w", analyzeErr)
		}
		contribution = 0
		for _, task := range critical.Tasks {
			if task.TaskClass != taskClass {
				continue
			}
			matched = append(matched, MatchedTask{
				Identity: task.Identity, DurationMs: task.DurationMs,
				CriticalPath: task.CriticalPath, Outcome: task.Outcome,
			})
			if task.CriticalPath {
				contribution += task.DurationMs
			}
		}
		if len(matched) == 0 {
			return Report{}, errors.New("task class did not match an executed task")
		}
		sort.Slice(matched, func(left, right int) bool { return matched[left].Identity < matched[right].Identity })
	}
	percent := float64(contribution) * 100 / float64(workflowMs)
	return Report{
		SchemaVersion: ReportSchema, Family: family, Method: method, TaskClass: taskClass,
		OperationTraceSHA256: operationSHA, TaskGraphSHA256: graphSHA,
		WorkflowMs: workflowMs, CriticalPathContributionMs: contribution,
		MaterialPercent: percent, MatchedTasks: matched,
		MinimumMillisecondsPassed: contribution >= 500,
		MinimumPercentPassed:      percent >= 2,
		EnvironmentClass:          "CONTROLLED_PERFORMANCE", PerformanceGateAuthority: true,
	}, nil
}

func operationDurations(path string) (int64, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open operation trace: %w", err)
	}
	defer file.Close()
	starts := make(map[int64]operation)
	ends := make(map[int64]int64)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var row operation
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return 0, 0, fmt.Errorf("decode operation trace: %w", err)
		}
		if row.StartTime != nil {
			if _, exists := starts[row.ID]; exists {
				return 0, 0, errors.New("duplicate operation start")
			}
			starts[row.ID] = row
		}
		if row.EndTime != nil {
			if _, exists := ends[row.ID]; exists {
				return 0, 0, errors.New("duplicate operation end")
			}
			ends[row.ID] = *row.EndTime
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("scan operation trace: %w", err)
	}
	var root operation
	for _, row := range starts {
		if row.ParentID == nil && row.DisplayName == "Run build" {
			if root.StartTime != nil {
				return 0, 0, errors.New("multiple root build operations")
			}
			root = row
		}
	}
	rootEnd, ok := ends[root.ID]
	if root.StartTime == nil || !ok || rootEnd <= *root.StartTime {
		return 0, 0, errors.New("complete root build operation is required")
	}
	var configuration []interval
	for id, row := range starts {
		if row.ParentID == nil || *row.ParentID != root.ID || row.StartTime == nil {
			continue
		}
		if row.DisplayName != "Load build" && row.DisplayName != "Configure build" && row.DisplayName != "Calculate build tree task graph" {
			continue
		}
		end, exists := ends[id]
		if !exists || end < *row.StartTime {
			return 0, 0, errors.New("configuration operation is incomplete")
		}
		configuration = append(configuration, interval{start: *row.StartTime, end: end})
	}
	if len(configuration) < 3 {
		return 0, 0, errors.New("root load, configure, and task-graph operations are required")
	}
	sort.Slice(configuration, func(left, right int) bool { return configuration[left].start < configuration[right].start })
	merged := configuration[0]
	var configurationMs int64
	for _, current := range configuration[1:] {
		if current.start <= merged.end {
			if current.end > merged.end {
				merged.end = current.end
			}
			continue
		}
		configurationMs += merged.end - merged.start
		merged = current
	}
	configurationMs += merged.end - merged.start
	if configurationMs <= 0 {
		return 0, 0, errors.New("configuration critical path is empty")
	}
	return rootEnd - *root.StartTime, configurationMs, nil
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
