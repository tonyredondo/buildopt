package runtimeoptimizer

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

// WorkspaceLifecycleKind describes the runner workspace before the invocation.
type WorkspaceLifecycleKind string

const (
	WorkspaceLifecycleNew        WorkspaceLifecycleKind = "NEW"
	WorkspaceLifecyclePersistent WorkspaceLifecycleKind = "PERSISTENT"
)

// WorkspaceLifecycleContract contains the proof needed to omit physical cleanup.
type WorkspaceLifecycleContract struct {
	Kind                      WorkspaceLifecycleKind
	EmptyVerified             bool
	PersistentLifecycleProven bool
	PreventsStaleOutputs      bool
	CleanIsPipelineBarrier    bool
}

// CleanTaskContract is the exact model evidence for one CLI task position.
type CleanTaskContract struct {
	ArgumentIndex              int
	InvocationToken            string
	TaskPath                   string
	ImplementationType         string
	CoreTask                   bool
	DeletesDeclaredOutputsOnly bool
	AddedActions               int
	Dependencies               []string
	Finalizers                 []string
	SideEffects                []string
	Customized                 bool
	FailureIsObserved          bool
}

// CleanRemovalRequest binds immutable argv to policy, model, and lifecycle proof.
type CleanRemovalRequest struct {
	Arguments                 []string
	TaskArgumentIndexes       []int
	TaskContracts             []CleanTaskContract
	Authorized                bool
	ModelAvailable            bool
	ReleaseContract           bool
	ReproducibilityValidation bool
	Workspace                 WorkspaceLifecycleContract
}

// CleanRemovalDecision preserves original argv and records the exact safe rewrite.
type CleanRemovalDecision struct {
	OriginalArguments  []string
	EffectiveArguments []string
	RemovedTaskPaths   []string
	Applied            bool
	SkipInvocation     bool
	Reason             string
}

// EvaluateCleanRemoval removes every modeled clean task or preserves the whole command.
func EvaluateCleanRemoval(request CleanRemovalRequest) (CleanRemovalDecision, error) {
	decision := preserveCleanArguments(request.Arguments, "NOT_ELIGIBLE")
	if err := validateCleanRemovalShape(request); err != nil {
		return CleanRemovalDecision{}, err
	}
	if !request.Authorized {
		decision.Reason = "NOT_AUTHORIZED"
		return decision, nil
	}
	if !request.ModelAvailable {
		decision.Reason = "MODEL_UNAVAILABLE"
		return decision, nil
	}
	if request.ReleaseContract {
		decision.Reason = "RELEASE_CONTRACT"
		return decision, nil
	}
	if request.ReproducibilityValidation {
		decision.Reason = "REPRODUCIBILITY_VALIDATION"
		return decision, nil
	}
	if reason := workspaceCleanRemovalReason(request.Workspace); reason != "" {
		decision.Reason = reason
		return decision, nil
	}

	contracts := make(map[int]CleanTaskContract, len(request.TaskContracts))
	for _, contract := range request.TaskContracts {
		contracts[contract.ArgumentIndex] = contract
	}
	cleanIndexes := make(map[int]struct{})
	for _, index := range request.TaskArgumentIndexes {
		if !cleanInvocationToken(request.Arguments[index]) {
			continue
		}
		contract, ok := contracts[index]
		if !ok {
			decision.Reason = "CLEAN_MODEL_INCOMPLETE"
			return decision, nil
		}
		if reason := cleanTaskContractReason(request.Arguments[index], contract); reason != "" {
			decision.Reason = reason
			return decision, nil
		}
		cleanIndexes[index] = struct{}{}
	}
	if len(cleanIndexes) == 0 {
		decision.Reason = "NO_CLEAN_TASK"
		return decision, nil
	}
	for _, contract := range request.TaskContracts {
		if _, ok := cleanIndexes[contract.ArgumentIndex]; !ok {
			return CleanRemovalDecision{}, errors.New("evaluate clean removal: contract does not identify a clean task")
		}
	}

	effective := make([]string, 0, len(request.Arguments)-len(cleanIndexes))
	removed := make([]string, 0, len(cleanIndexes))
	for index, argument := range request.Arguments {
		if _, remove := cleanIndexes[index]; remove {
			removed = append(removed, contracts[index].TaskPath)
			continue
		}
		effective = append(effective, argument)
	}
	remainingTasks := 0
	for _, index := range request.TaskArgumentIndexes {
		if _, removed := cleanIndexes[index]; !removed {
			remainingTasks++
		}
	}
	decision.EffectiveArguments = effective
	decision.RemovedTaskPaths = removed
	decision.Applied = true
	decision.SkipInvocation = remainingTasks == 0
	decision.Reason = "APPLIED"
	return decision, nil
}

func validateCleanRemovalShape(request CleanRemovalRequest) error {
	if len(request.Arguments) == 0 || request.Arguments[0] == "" || filepath.Base(request.Arguments[0]) != "gradlew" {
		return errors.New("evaluate clean removal: expected Gradle Wrapper argv")
	}
	if !sortedUniqueIndexes(request.TaskArgumentIndexes, len(request.Arguments)) {
		return errors.New("evaluate clean removal: invalid task argument indexes")
	}
	seenContracts := make(map[int]struct{}, len(request.TaskContracts))
	for _, contract := range request.TaskContracts {
		if contract.ArgumentIndex < 1 || contract.ArgumentIndex >= len(request.Arguments) ||
			!slices.Contains(request.TaskArgumentIndexes, contract.ArgumentIndex) || contract.InvocationToken == "" {
			return errors.New("evaluate clean removal: invalid clean task contract position")
		}
		if _, duplicate := seenContracts[contract.ArgumentIndex]; duplicate {
			return errors.New("evaluate clean removal: duplicate clean task contract")
		}
		seenContracts[contract.ArgumentIndex] = struct{}{}
	}
	return nil
}

func sortedUniqueIndexes(indexes []int, argumentCount int) bool {
	previous := 0
	for position, index := range indexes {
		if index < 1 || index >= argumentCount || position > 0 && index <= previous {
			return false
		}
		previous = index
	}
	return true
}

func workspaceCleanRemovalReason(contract WorkspaceLifecycleContract) string {
	if contract.CleanIsPipelineBarrier {
		return "CI_BARRIER"
	}
	switch contract.Kind {
	case WorkspaceLifecycleNew:
		if !contract.EmptyVerified {
			return "WORKSPACE_NOT_VERIFIED_EMPTY"
		}
	case WorkspaceLifecyclePersistent:
		if !contract.PersistentLifecycleProven || !contract.PreventsStaleOutputs {
			return "PERSISTENT_LIFECYCLE_UNPROVEN"
		}
	default:
		return "WORKSPACE_LIFECYCLE_UNKNOWN"
	}
	return ""
}

func cleanTaskContractReason(argument string, contract CleanTaskContract) string {
	if contract.InvocationToken != argument || !allowlistedCleanTaskPath(contract.TaskPath) ||
		contract.ImplementationType != "org.gradle.api.tasks.Delete" || !contract.CoreTask {
		return "TASK_NOT_ALLOWLISTED_CORE_CLEAN"
	}
	if !contract.DeletesDeclaredOutputsOnly {
		return "UNDECLARED_DELETION"
	}
	if contract.Customized {
		return "CUSTOMIZED_CLEAN"
	}
	if contract.AddedActions != 0 {
		return "ADDED_ACTIONS"
	}
	if len(contract.Dependencies) != 0 {
		return "DEPENDENCIES"
	}
	if len(contract.Finalizers) != 0 {
		return "FINALIZERS"
	}
	if len(contract.SideEffects) != 0 {
		return "SIDE_EFFECTS"
	}
	if contract.FailureIsObserved {
		return "FAILURE_SEMANTICS"
	}
	return ""
}

func cleanInvocationToken(value string) bool {
	return value == "clean" || allowlistedCleanTaskPath(value)
}

func allowlistedCleanTaskPath(value string) bool {
	if value == ":clean" {
		return true
	}
	if !strings.HasPrefix(value, ":") || !strings.HasSuffix(value, ":clean") {
		return false
	}
	parts := strings.Split(value[1:], ":")
	if len(parts) < 2 || parts[len(parts)-1] != "clean" {
		return false
	}
	for _, part := range parts[:len(parts)-1] {
		if !identifierPattern.MatchString(part) {
			return false
		}
	}
	return true
}

func preserveCleanArguments(arguments []string, reason string) CleanRemovalDecision {
	original := slices.Clone(arguments)
	return CleanRemovalDecision{OriginalArguments: original, EffectiveArguments: slices.Clone(original), Reason: reason}
}
