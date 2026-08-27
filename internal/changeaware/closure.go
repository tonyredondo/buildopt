// Package changeaware derives conservative partial-work evidence from one
// completed Gradle workflow and an exact adjacent-revision change set.
package changeaware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

const (
	CaptureSchemaVersion = "buildopt.poc/change-aware-producer-capture/v1"
	ReportSchemaVersion  = "buildopt.poc/change-aware-producer-report/v1"

	CaptureComplete    = "COMPLETE"
	CaptureUnavailable = "UNAVAILABLE"
	CaptureFailed      = "FAILED"

	StatusTestableActions  = "TESTABLE_ACTIONS"
	StatusNoSafeAction     = "NO_SAFE_ACTION"
	StatusNotApplicable    = "NOT_APPLICABLE"
	StatusInputUnavailable = "INPUT_UNAVAILABLE"
	StatusProducerFailed   = "PRODUCER_FAILED"
)

// Capture is observation-only evidence from one useful Gradle invocation.
type Capture struct {
	SchemaVersion   string         `json:"schemaVersion"`
	GeneratedAt     string         `json:"generatedAt"`
	Family          string         `json:"family"`
	DSL             string         `json:"dsl"`
	BaseRevision    string         `json:"baseRevision"`
	TargetRevision  string         `json:"targetRevision"`
	Status          string         `json:"status"`
	Reason          string         `json:"reason"`
	ChangedPaths    []string       `json:"changedPaths"`
	RequestedTasks  []string       `json:"requestedTasks"`
	RequiredOutputs []string       `json:"requiredOutputs"`
	Tasks           []TaskEvidence `json:"tasks"`
}

// TaskEvidence binds one configured task to its exact graph position and
// finalized repository inputs and outputs.
type TaskEvidence struct {
	Path      string           `json:"path"`
	DependsOn []string         `json:"dependsOn"`
	Inputs    []PathEvidence   `json:"inputs"`
	Outputs   []OutputEvidence `json:"outputs"`
}

// PathEvidence identifies a repository-relative file or directory consumed by
// a task after Gradle has finalized its inputs.
type PathEvidence struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// OutputEvidence identifies a repository-relative task output and binds any
// existing content to its deterministic digest.
type OutputEvidence struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Exists bool   `json:"exists"`
}

// Report is a typed, non-authorizing detector outcome.
type Report struct {
	SchemaVersion        string        `json:"schemaVersion"`
	GeneratedAt          string        `json:"generatedAt"`
	Family               string        `json:"family"`
	DSL                  string        `json:"dsl"`
	BaseRevision         string        `json:"baseRevision"`
	TargetRevision       string        `json:"targetRevision"`
	Status               string        `json:"status"`
	Reason               string        `json:"reason"`
	ChangedPaths         []string      `json:"changedPaths"`
	RequestedTasks       []string      `json:"requestedTasks"`
	RequiredOutputs      []string      `json:"requiredOutputs"`
	AffectedInputTasks   []string      `json:"affectedInputTasks"`
	RequiredProducers    []string      `json:"requiredProducers"`
	CandidateTasks       []string      `json:"candidateTasks"`
	OmittedTasks         []string      `json:"omittedTasks"`
	OmittedOutputs       []BoundOutput `json:"omittedOutputs"`
	ActionBindingSHA256  string        `json:"actionBindingSha256,omitempty"`
	InputComplete        bool          `json:"inputComplete"`
	TestableActions      int           `json:"testableActions"`
	PerformanceMeasured  bool          `json:"performanceMeasured"`
	ActivationAuthorized bool          `json:"activationAuthorized"`
}

// BoundOutput identifies one verified output that a candidate would omit and
// therefore must preserve exactly.
type BoundOutput struct {
	ProducerTask string `json:"producerTask"`
	Path         string `json:"path"`
	Kind         string `json:"kind"`
	SHA256       string `json:"sha256"`
}

// Analyze validates the complete graph before deriving any action. Missing or
// ambiguous ownership never becomes an optimistic partial build.
func Analyze(capture Capture) (Report, error) {
	report := baseReport(capture)
	if capture.SchemaVersion != CaptureSchemaVersion || capture.GeneratedAt == "" ||
		!safeLabel(capture.Family) || (capture.DSL != "KOTLIN" && capture.DSL != "GROOVY") ||
		!validRevision(capture.BaseRevision) || !validRevision(capture.TargetRevision) ||
		capture.BaseRevision == capture.TargetRevision {
		return Report{}, errors.New("change-aware capture identity is invalid")
	}
	switch capture.Status {
	case CaptureUnavailable:
		if capture.Reason == "" || len(capture.Tasks) != 0 {
			return Report{}, errors.New("unavailable capture boundary is invalid")
		}
		report.Status, report.Reason = StatusInputUnavailable, capture.Reason
		return report, nil
	case CaptureFailed:
		if capture.Reason == "" || len(capture.Tasks) != 0 {
			return Report{}, errors.New("failed capture boundary is invalid")
		}
		report.Status, report.Reason = StatusProducerFailed, capture.Reason
		return report, nil
	case CaptureComplete:
		if capture.Reason != "" {
			return Report{}, errors.New("complete capture has a failure reason")
		}
	default:
		return Report{}, errors.New("unknown capture status")
	}
	if err := validateStringSet(capture.ChangedPaths, safeRepositoryPath); err != nil ||
		validateStringSet(capture.RequestedTasks, safeTaskPath) != nil ||
		validateStringSet(capture.RequiredOutputs, safeRepositoryPath) != nil ||
		len(capture.Tasks) == 0 {
		return Report{}, errors.New("change-aware capture scope is invalid")
	}
	tasks, reverse, err := validateTasks(capture.Tasks)
	if err != nil {
		return noSafe(report, "TASK_GRAPH_INCOMPLETE_OR_AMBIGUOUS"), nil
	}
	if cyclic(tasks) {
		return noSafe(report, "TASK_GRAPH_CYCLIC"), nil
	}
	for _, requested := range capture.RequestedTasks {
		if _, ok := tasks[requested]; !ok {
			return noSafe(report, "REQUESTED_TASK_UNAVAILABLE"), nil
		}
	}

	affectedSeeds := map[string]bool{}
	for _, changed := range capture.ChangedPaths {
		owners := []string{}
		for taskPath, task := range tasks {
			if taskConsumes(task, changed) {
				owners = append(owners, taskPath)
			}
		}
		if len(owners) == 0 {
			return noSafe(report, "CHANGE_INPUT_OWNERSHIP_UNPROVEN"), nil
		}
		for _, owner := range owners {
			affectedSeeds[owner] = true
		}
	}
	report.AffectedInputTasks = sortedKeys(affectedSeeds)

	requiredProducers := map[string]bool{}
	for _, required := range capture.RequiredOutputs {
		producers := []string{}
		for taskPath, task := range tasks {
			if taskProduces(task, required) {
				producers = append(producers, taskPath)
			}
		}
		if len(producers) == 0 {
			return noSafe(report, "REQUIRED_OUTPUT_PRODUCER_MISSING"), nil
		}
		if len(producers) != 1 {
			return noSafe(report, "REQUIRED_OUTPUT_PRODUCER_AMBIGUOUS"), nil
		}
		requiredProducers[producers[0]] = true
	}
	report.RequiredProducers = sortedKeys(requiredProducers)

	requiredGraph := ancestors(tasks, capture.RequestedTasks)
	for producer := range requiredProducers {
		if !requiredGraph[producer] {
			return noSafe(report, "REQUIRED_OUTPUT_OUTSIDE_REQUESTED_GRAPH"), nil
		}
	}
	affectedGraph := descendants(reverse, report.AffectedInputTasks)
	affectedRequiredOutput := false
	for producer := range requiredProducers {
		if affectedGraph[producer] {
			affectedRequiredOutput = true
			break
		}
	}
	if !affectedRequiredOutput {
		report.Status, report.Reason = StatusNotApplicable, "CHANGE_DOES_NOT_REACH_REQUIRED_OUTPUT"
		report.InputComplete = true
		return report, nil
	}
	candidateSet := map[string]bool{}
	for taskPath := range requiredGraph {
		if affectedGraph[taskPath] {
			candidateSet[taskPath] = true
		}
	}
	if len(candidateSet) == 0 {
		report.Status, report.Reason = StatusNotApplicable, "CHANGE_DOES_NOT_REACH_REQUIRED_OUTPUT"
		report.InputComplete = true
		return report, nil
	}
	omittedSet := map[string]bool{}
	for taskPath := range requiredGraph {
		if !candidateSet[taskPath] {
			omittedSet[taskPath] = true
		}
	}
	if len(omittedSet) == 0 {
		return noSafe(report, "FULL_PRODUCER_GRAPH_REQUIRED"), nil
	}
	omittedOutputs := []BoundOutput{}
	for _, taskPath := range sortedKeys(omittedSet) {
		for _, output := range tasks[taskPath].Outputs {
			if !output.Exists || !validSHA(output.SHA256) {
				return noSafe(report, "OMITTED_OUTPUT_EVIDENCE_INCOMPLETE"), nil
			}
			omittedOutputs = append(omittedOutputs, BoundOutput{
				ProducerTask: taskPath, Path: output.Path, Kind: output.Kind, SHA256: output.SHA256,
			})
		}
	}
	if len(omittedOutputs) == 0 {
		return noSafe(report, "OMITTED_OUTPUT_CLOSURE_EMPTY"), nil
	}
	sort.Slice(omittedOutputs, func(i, j int) bool {
		if omittedOutputs[i].ProducerTask == omittedOutputs[j].ProducerTask {
			return omittedOutputs[i].Path < omittedOutputs[j].Path
		}
		return omittedOutputs[i].ProducerTask < omittedOutputs[j].ProducerTask
	})
	report.Status, report.Reason = StatusTestableActions, "EXACT_CHANGE_AWARE_PRODUCER_CLOSURE"
	report.CandidateTasks = sortedKeys(candidateSet)
	report.OmittedTasks = sortedKeys(omittedSet)
	report.OmittedOutputs = omittedOutputs
	report.ActionBindingSHA256 = actionBinding(report)
	report.InputComplete, report.TestableActions = true, 1
	return report, nil
}

func baseReport(capture Capture) Report {
	return Report{
		SchemaVersion: ReportSchemaVersion, GeneratedAt: capture.GeneratedAt,
		Family: capture.Family, DSL: capture.DSL,
		BaseRevision: capture.BaseRevision, TargetRevision: capture.TargetRevision,
		ChangedPaths:       append([]string(nil), capture.ChangedPaths...),
		RequestedTasks:     append([]string(nil), capture.RequestedTasks...),
		RequiredOutputs:    append([]string(nil), capture.RequiredOutputs...),
		AffectedInputTasks: []string{}, RequiredProducers: []string{},
		CandidateTasks: []string{}, OmittedTasks: []string{}, OmittedOutputs: []BoundOutput{},
		PerformanceMeasured: false, ActivationAuthorized: false,
	}
}

func noSafe(report Report, reason string) Report {
	report.Status, report.Reason = StatusNoSafeAction, reason
	report.InputComplete = true
	return report
}

func validateTasks(values []TaskEvidence) (map[string]TaskEvidence, map[string][]string, error) {
	tasks := make(map[string]TaskEvidence, len(values))
	reverse := map[string][]string{}
	for _, task := range values {
		if !safeTaskPath(task.Path) || tasks[task.Path].Path != "" ||
			(len(task.DependsOn) != 0 && validateStringSet(task.DependsOn, safeTaskPath) != nil) {
			return nil, nil, errors.New("invalid task identity")
		}
		if err := validatePaths(task.Inputs); err != nil || validateOutputs(task.Outputs) != nil {
			return nil, nil, errors.New("invalid task paths")
		}
		tasks[task.Path] = task
	}
	for taskPath, task := range tasks {
		for _, dependency := range task.DependsOn {
			if dependency == taskPath || tasks[dependency].Path == "" {
				return nil, nil, errors.New("unknown task dependency")
			}
			reverse[dependency] = append(reverse[dependency], taskPath)
		}
	}
	return tasks, reverse, nil
}

func validatePaths(values []PathEvidence) error {
	seen := map[string]bool{}
	for _, value := range values {
		if !safeRepositoryPath(value.Path) || (value.Kind != "FILE" && value.Kind != "DIRECTORY") ||
			seen[value.Kind+"\x00"+value.Path] {
			return errors.New("invalid path evidence")
		}
		seen[value.Kind+"\x00"+value.Path] = true
	}
	return nil
}

func validateOutputs(values []OutputEvidence) error {
	seen := map[string]bool{}
	for _, value := range values {
		if !safeRepositoryPath(value.Path) || (value.Kind != "FILE" && value.Kind != "DIRECTORY") ||
			(value.Exists && !validSHA(value.SHA256)) || (!value.Exists && value.SHA256 != "") ||
			seen[value.Kind+"\x00"+value.Path] {
			return errors.New("invalid output evidence")
		}
		seen[value.Kind+"\x00"+value.Path] = true
	}
	return nil
}

func taskConsumes(task TaskEvidence, changed string) bool {
	for _, input := range task.Inputs {
		if input.Path == changed || (input.Kind == "DIRECTORY" && under(input.Path, changed)) {
			return true
		}
	}
	return false
}

func taskProduces(task TaskEvidence, required string) bool {
	for _, output := range task.Outputs {
		if output.Path == required || (output.Kind == "DIRECTORY" && under(output.Path, required)) {
			return true
		}
	}
	return false
}

func ancestors(tasks map[string]TaskEvidence, roots []string) map[string]bool {
	result := map[string]bool{}
	pending := append([]string(nil), roots...)
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if result[current] {
			continue
		}
		result[current] = true
		pending = append(pending, tasks[current].DependsOn...)
	}
	return result
}

func descendants(reverse map[string][]string, roots []string) map[string]bool {
	result := map[string]bool{}
	pending := append([]string(nil), roots...)
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if result[current] {
			continue
		}
		result[current] = true
		pending = append(pending, reverse[current]...)
	}
	return result
}

func cyclic(tasks map[string]TaskEvidence) bool {
	state := map[string]uint8{}
	var visit func(string) bool
	visit = func(taskPath string) bool {
		if state[taskPath] == 1 {
			return true
		}
		if state[taskPath] == 2 {
			return false
		}
		state[taskPath] = 1
		for _, dependency := range tasks[taskPath].DependsOn {
			if visit(dependency) {
				return true
			}
		}
		state[taskPath] = 2
		return false
	}
	for taskPath := range tasks {
		if visit(taskPath) {
			return true
		}
	}
	return false
}

func actionBinding(report Report) string {
	parts := []string{"buildopt-change-aware-action-v1", report.Family, report.BaseRevision, report.TargetRevision}
	parts = append(parts, report.ChangedPaths...)
	parts = append(parts, report.CandidateTasks...)
	for _, output := range report.OmittedOutputs {
		parts = append(parts, output.ProducerTask, output.Path, output.Kind, output.SHA256)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func validateStringSet(values []string, validate func(string) bool) error {
	if len(values) == 0 {
		return errors.New("empty set")
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !validate(value) || seen[value] {
			return fmt.Errorf("invalid or duplicate value %q", value)
		}
		seen[value] = true
	}
	return nil
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func under(root, candidate string) bool {
	return candidate != root && strings.HasPrefix(candidate, root+"/")
}

func safeRepositoryPath(value string) bool {
	return value != "" && value == path.Clean(value) && value != "." && !strings.HasPrefix(value, "/") &&
		!strings.HasPrefix(value, "../") && !strings.Contains(value, "\\") && !strings.ContainsRune(value, '\x00')
}

func safeTaskPath(value string) bool {
	return strings.HasPrefix(value, ":") && !strings.ContainsAny(value, "\x00\r\n ")
}

func safeLabel(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if !((character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') ||
			character == '-' || character == '_') {
			return false
		}
	}
	return true
}

func validSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validRevision(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
