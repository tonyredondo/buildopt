package requestaligned

import (
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/changeaware"
)

const (
	TransitionSchemaVersionV1     = "buildopt.poc/request-aligned-transition/v1"
	ClassificationSchemaVersionV1 = "buildopt.poc/request-aligned-classification/v1"
	TransitionSchemaVersion       = "buildopt.poc/request-aligned-transition/v2"
	ClassificationSchemaVersion   = "buildopt.poc/request-aligned-classification/v2"

	ClassificationRelevantComplete  = "RELEVANT_COMPLETE"
	ClassificationIrrelevant        = "IRRELEVANT_TO_REQUEST"
	ClassificationGlobalOrAmbiguous = "GLOBAL_OR_AMBIGUOUS"
	ClassificationInputUnavailable  = "INPUT_UNAVAILABLE"
	ClassificationProducerFailed    = "PRODUCER_FAILED"
)

// Transition binds two adjacent revisions to captures of the exact same
// ordinary Gradle request. It carries no candidate timing or activation.
type Transition struct {
	SchemaVersion  string   `json:"schemaVersion"`
	GeneratedAt    string   `json:"generatedAt"`
	BaseRevision   string   `json:"baseRevision"`
	TargetRevision string   `json:"targetRevision"`
	ChangedPaths   []string `json:"changedPaths"`
	BaseCapture    Capture  `json:"baseCapture"`
	TargetCapture  Capture  `json:"targetCapture"`
}

// Classification is one typed adjacent-request decision. Only a complete
// relevant row with one TestableAction may be considered by a later block.
type Classification struct {
	SchemaVersion         string                    `json:"schemaVersion"`
	GeneratedAt           string                    `json:"generatedAt"`
	BaseRevision          string                    `json:"baseRevision"`
	TargetRevision        string                    `json:"targetRevision"`
	Status                string                    `json:"status"`
	Reason                string                    `json:"reason"`
	ChangedPaths          []string                  `json:"changedPaths"`
	GradleArguments       []string                  `json:"gradleArguments"`
	RequestedTasks        []string                  `json:"requestedTasks"`
	RequestIdentitySHA256 string                    `json:"requestIdentitySha256,omitempty"`
	AffectedInputTasks    []string                  `json:"affectedInputTasks"`
	CandidateTasks        []string                  `json:"candidateTasks"`
	OmittedTasks          []string                  `json:"omittedTasks"`
	OmittedOutputs        []changeaware.BoundOutput `json:"omittedOutputs"`
	OmittedOutputStates   []OutputState             `json:"omittedOutputStates,omitempty"`
	ActionBindingSHA256   string                    `json:"actionBindingSha256,omitempty"`
	InputComplete         bool                      `json:"inputComplete"`
	TestableActions       int                       `json:"testableActions"`
	PerformanceMeasured   bool                      `json:"performanceMeasured"`
	ActivationAuthorized  bool                      `json:"activationAuthorized"`
}

// Classify validates both request observations and derives the conservative
// affected/omitted producer closure for the target revision.
func Classify(transition Transition) (Classification, error) {
	report := baseClassification(transition)
	if (transition.SchemaVersion != TransitionSchemaVersionV1 && transition.SchemaVersion != TransitionSchemaVersion) ||
		transition.GeneratedAt == "" ||
		!validRevision(transition.BaseRevision) || !validRevision(transition.TargetRevision) ||
		transition.BaseRevision == transition.TargetRevision ||
		validateStringSetAllowEmpty(transition.ChangedPaths, safeRepositoryPath) != nil {
		return Classification{}, errors.New("request-aligned transition identity is invalid")
	}
	expectedCaptureSchema := CaptureSchemaVersion
	if transition.SchemaVersion == TransitionSchemaVersionV1 {
		expectedCaptureSchema = CaptureSchemaVersionV1
	}
	if transition.BaseCapture.SchemaVersion != expectedCaptureSchema ||
		transition.TargetCapture.SchemaVersion != expectedCaptureSchema {
		return Classification{}, errors.New("request-aligned transition and capture versions differ")
	}

	base, err := Produce(transition.BaseCapture)
	if err != nil {
		return Classification{}, err
	}
	target, err := Produce(transition.TargetCapture)
	if err != nil {
		return Classification{}, err
	}
	report.GradleArguments = append([]string(nil), target.GradleArguments...)
	report.RequestedTasks = append([]string(nil), target.RequestedTasks...)

	if base.Status == StatusFailed || target.Status == StatusFailed {
		return classified(report, ClassificationProducerFailed, "REQUEST_PRODUCER_FAILED", false), nil
	}
	if base.Status == StatusUnavailable || target.Status == StatusUnavailable {
		reason := firstUnavailableReason(base, target)
		if strings.Contains(reason, "AMBIGUOUS") || reason == "REQUESTED_TASK_UNAVAILABLE" {
			return classified(report, ClassificationGlobalOrAmbiguous, reason, false), nil
		}
		return classified(report, ClassificationInputUnavailable, reason, false), nil
	}
	if transition.SchemaVersion == TransitionSchemaVersion {
		return classifyV2(report, transition, base, target)
	}
	if base.RequestIdentitySHA256 != target.RequestIdentitySHA256 {
		return classified(report, ClassificationGlobalOrAmbiguous, "REQUEST_IDENTITY_CHANGED", true), nil
	}
	report.RequestIdentitySHA256 = target.RequestIdentitySHA256

	tasks := make(map[string]changeaware.TaskEvidence, len(target.Tasks))
	reverse := map[string][]string{}
	for _, task := range target.Tasks {
		tasks[task.Path] = task
	}
	for _, task := range target.Tasks {
		for _, dependency := range task.DependsOn {
			reverse[dependency] = append(reverse[dependency], task.Path)
		}
	}
	if taskGraphCyclic(tasks) {
		return classified(report, ClassificationGlobalOrAmbiguous, "TASK_GRAPH_CYCLIC", true), nil
	}

	affectedSeeds := map[string]bool{}
	for _, changed := range transition.ChangedPaths {
		for taskPath, task := range tasks {
			if taskConsumesPath(task, changed) {
				affectedSeeds[taskPath] = true
			}
		}
	}
	if len(affectedSeeds) == 0 {
		return classified(report, ClassificationIrrelevant, "NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS", true), nil
	}
	report.AffectedInputTasks = sortedSet(affectedSeeds)

	requiredGraph := taskAncestors(tasks, target.RequestedTasks)
	affectedGraph := taskDescendants(reverse, report.AffectedInputTasks)
	candidate := map[string]bool{}
	omitted := map[string]bool{}
	for taskPath := range requiredGraph {
		if affectedGraph[taskPath] {
			candidate[taskPath] = true
		} else {
			omitted[taskPath] = true
		}
	}
	if len(candidate) == 0 {
		return classified(report, ClassificationGlobalOrAmbiguous, "AFFECTED_TASK_OUTSIDE_REQUESTED_GRAPH", true), nil
	}
	report.CandidateTasks = sortedSet(candidate)
	report.OmittedTasks = sortedSet(omitted)

	currentOutputs := map[string]ProducerOutput{}
	for _, output := range target.CurrentOutputs {
		currentOutputs[output.ProducerTask+"\x00"+output.Kind+"\x00"+output.Path] = output
	}
	for _, taskPath := range report.OmittedTasks {
		for _, output := range tasks[taskPath].Outputs {
			if !output.Exists || !validSHA(output.SHA256) {
				return classified(report, ClassificationInputUnavailable, "OMITTED_OUTPUT_EVIDENCE_INCOMPLETE", false), nil
			}
			key := taskPath + "\x00" + output.Kind + "\x00" + output.Path
			current, exists := currentOutputs[key]
			if !exists || current.SHA256 != output.SHA256 {
				return classified(report, ClassificationInputUnavailable, "CURRENT_OUTPUT_BINDING_MISSING", false), nil
			}
			report.OmittedOutputs = append(report.OmittedOutputs, changeaware.BoundOutput{
				ProducerTask: taskPath, Path: output.Path, Kind: output.Kind, SHA256: output.SHA256,
			})
		}
	}
	sort.Slice(report.OmittedOutputs, func(i, j int) bool {
		if report.OmittedOutputs[i].ProducerTask != report.OmittedOutputs[j].ProducerTask {
			return report.OmittedOutputs[i].ProducerTask < report.OmittedOutputs[j].ProducerTask
		}
		if report.OmittedOutputs[i].Path != report.OmittedOutputs[j].Path {
			return report.OmittedOutputs[i].Path < report.OmittedOutputs[j].Path
		}
		return report.OmittedOutputs[i].Kind < report.OmittedOutputs[j].Kind
	})

	report = classified(report, ClassificationRelevantComplete, "RELEVANT_REQUEST_INPUT_COMPLETE", true)
	if len(report.OmittedTasks) > 0 && len(report.OmittedOutputs) > 0 {
		report.Reason = "EXACT_RELEVANT_PRODUCER_CLOSURE"
		report.TestableActions = 1
		report.ActionBindingSHA256 = classificationBinding(report)
	} else {
		report.Reason = "FULL_REQUEST_GRAPH_REQUIRED"
	}
	return report, nil
}

func baseClassification(transition Transition) Classification {
	changed := append([]string(nil), transition.ChangedPaths...)
	sort.Strings(changed)
	schemaVersion := ClassificationSchemaVersion
	if transition.SchemaVersion == TransitionSchemaVersionV1 {
		schemaVersion = ClassificationSchemaVersionV1
	}
	return Classification{
		SchemaVersion: schemaVersion, GeneratedAt: transition.GeneratedAt,
		BaseRevision: transition.BaseRevision, TargetRevision: transition.TargetRevision,
		ChangedPaths: changed, AffectedInputTasks: []string{}, CandidateTasks: []string{},
		OmittedTasks: []string{}, OmittedOutputs: []changeaware.BoundOutput{},
		PerformanceMeasured: false, ActivationAuthorized: false,
	}
}

func classifyV2(
	report Classification,
	transition Transition,
	base Observation,
	target Observation,
) (Classification, error) {
	if base.CompatibilityIdentitySHA256 != target.CompatibilityIdentitySHA256 {
		return classified(report, ClassificationGlobalOrAmbiguous, "REQUEST_COMPATIBILITY_CHANGED", true), nil
	}
	if base.RequestGraphIdentitySHA256 != target.RequestGraphIdentitySHA256 {
		return classified(report, ClassificationGlobalOrAmbiguous, "REQUEST_GRAPH_CHANGED", true), nil
	}
	report.RequestIdentitySHA256 = target.RequestIdentitySHA256

	tasks := make(map[string]changeaware.TaskEvidence, len(target.Tasks))
	reverse := map[string][]string{}
	for _, task := range target.Tasks {
		tasks[task.Path] = task
	}
	for _, task := range target.Tasks {
		for _, dependency := range task.DependsOn {
			reverse[dependency] = append(reverse[dependency], task.Path)
		}
	}
	if taskGraphCyclic(tasks) {
		return classified(report, ClassificationGlobalOrAmbiguous, "TASK_GRAPH_CYCLIC", true), nil
	}

	affectedSeeds := map[string]bool{}
	for _, changed := range transition.ChangedPaths {
		for taskPath, task := range tasks {
			if taskConsumesPath(task, changed) {
				affectedSeeds[taskPath] = true
			}
		}
	}
	buildLogicChanged := base.BuildLogicSHA256 != target.BuildLogicSHA256
	if len(affectedSeeds) == 0 {
		reason := "NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS"
		if buildLogicChanged {
			reason = "BUILD_LOGIC_CHANGED_IRRELEVANT_TO_REQUEST"
		}
		return classified(report, ClassificationIrrelevant, reason, true), nil
	}
	if buildLogicChanged {
		return classified(report, ClassificationGlobalOrAmbiguous, "BUILD_LOGIC_CHANGED_WITH_RELEVANT_REQUEST_INPUT", true), nil
	}
	report.AffectedInputTasks = sortedSet(affectedSeeds)

	requiredGraph := taskAncestors(tasks, target.RequestedTasks)
	affectedGraph := taskDescendants(reverse, report.AffectedInputTasks)
	candidate := map[string]bool{}
	omitted := map[string]bool{}
	for taskPath := range requiredGraph {
		if affectedGraph[taskPath] {
			candidate[taskPath] = true
		} else {
			omitted[taskPath] = true
		}
	}
	if len(candidate) == 0 {
		return classified(report, ClassificationGlobalOrAmbiguous, "AFFECTED_TASK_OUTSIDE_REQUESTED_GRAPH", true), nil
	}
	report.CandidateTasks = sortedSet(candidate)
	report.OmittedTasks = sortedSet(omitted)

	statesByTask := map[string][]OutputState{}
	for _, state := range target.CurrentOutputStates {
		for _, producer := range state.ProducerTasks {
			statesByTask[producer] = append(statesByTask[producer], state)
		}
	}
	selected := map[string]OutputState{}
	for _, taskPath := range report.OmittedTasks {
		for _, output := range tasks[taskPath].Outputs {
			var matched *OutputState
			for _, state := range statesByTask[taskPath] {
				if state.Path == output.Path && state.Kind == output.Kind {
					copy := state
					matched = &copy
					break
				}
			}
			if matched == nil || matched.Exists != output.Exists || matched.SHA256 != output.SHA256 {
				return classified(report, ClassificationInputUnavailable, "CURRENT_OUTPUT_STATE_BINDING_MISSING", false), nil
			}
			for _, producer := range matched.ProducerTasks {
				if !omitted[producer] {
					return classified(report, ClassificationInputUnavailable, "OUTPUT_STATE_CROSSES_CANDIDATE_BOUNDARY", false), nil
				}
			}
			selected[outputStateIdentity(*matched)] = *matched
		}
	}
	for _, state := range selected {
		report.OmittedOutputStates = append(report.OmittedOutputStates, state)
	}
	sortOutputStates(report.OmittedOutputStates)

	report = classified(report, ClassificationRelevantComplete, "RELEVANT_REQUEST_INPUT_COMPLETE", true)
	if len(report.OmittedTasks) > 0 && len(report.OmittedOutputStates) > 0 {
		report.Reason = "EXACT_RELEVANT_PRODUCER_CLOSURE"
		report.TestableActions = 1
		report.ActionBindingSHA256 = classificationBindingV2(report)
	} else {
		report.Reason = "FULL_REQUEST_GRAPH_REQUIRED"
	}
	return report, nil
}

func classified(report Classification, status, reason string, complete bool) Classification {
	report.Status, report.Reason, report.InputComplete = status, reason, complete
	return report
}

func firstUnavailableReason(values ...Observation) string {
	for _, value := range values {
		if value.Status == StatusUnavailable && value.Reason != "" {
			return value.Reason
		}
	}
	return "REQUEST_INPUT_UNAVAILABLE"
}

func taskConsumesPath(task changeaware.TaskEvidence, changed string) bool {
	for _, input := range task.Inputs {
		if input.Path == changed || (input.Kind == "DIRECTORY" && pathUnder(input.Path, changed)) {
			return true
		}
	}
	return false
}

func pathUnder(parent, child string) bool {
	return strings.HasPrefix(child, strings.TrimSuffix(parent, "/")+"/")
}

func taskAncestors(tasks map[string]changeaware.TaskEvidence, roots []string) map[string]bool {
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

func taskDescendants(reverse map[string][]string, roots []string) map[string]bool {
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

func taskGraphCyclic(tasks map[string]changeaware.TaskEvidence) bool {
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

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func classificationBinding(report Classification) string {
	parts := []string{"buildopt-request-aligned-action-v1", report.RequestIdentitySHA256,
		report.BaseRevision, report.TargetRevision}
	parts = append(parts, report.ChangedPaths...)
	parts = append(parts, report.CandidateTasks...)
	for _, output := range report.OmittedOutputs {
		parts = append(parts, output.ProducerTask, output.Path, output.Kind, output.SHA256)
	}
	return digest("buildopt-request-aligned-action-binding-v1", parts...)
}

func classificationBindingV2(report Classification) string {
	parts := []string{"buildopt-request-aligned-action-v2", report.RequestIdentitySHA256,
		report.BaseRevision, report.TargetRevision}
	parts = append(parts, report.ChangedPaths...)
	parts = append(parts, report.CandidateTasks...)
	for _, state := range report.OmittedOutputStates {
		parts = append(parts, state.ProducerTasks...)
		parts = append(parts, state.Path, state.Kind, state.SHA256)
		if state.Exists {
			parts = append(parts, "PRESENT")
		} else {
			parts = append(parts, "ABSENT")
		}
	}
	return digest("buildopt-request-aligned-action-binding-v2", parts...)
}

func validRevision(value string) bool {
	if len(value) != 40 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
