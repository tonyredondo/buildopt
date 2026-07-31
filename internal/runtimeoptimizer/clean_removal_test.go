package runtimeoptimizer

import (
	"reflect"
	"testing"
)

func TestCleanRemovalAppliesToVerifiedNewWorkspace(t *testing.T) {
	request := testCleanRemovalRequest()
	decision, err := EvaluateCleanRemoval(request)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Applied || decision.SkipInvocation || decision.Reason != "APPLIED" ||
		!reflect.DeepEqual(decision.EffectiveArguments, []string{"./gradlew", "assemble", "--stacktrace"}) ||
		!reflect.DeepEqual(decision.RemovedTaskPaths, []string{":clean"}) {
		t.Fatalf("decision = %+v", decision)
	}
	request.Arguments[1] = "mutated"
	if decision.OriginalArguments[1] != "clean" {
		t.Fatal("decision did not retain immutable original arguments")
	}
}

func TestCleanRemovalSkipsCleanOnlyInvocation(t *testing.T) {
	request := testCleanRemovalRequest()
	request.Arguments = []string{"./gradlew", "clean"}
	request.TaskArgumentIndexes = []int{1}
	decision, err := EvaluateCleanRemoval(request)
	if err != nil || !decision.Applied || !decision.SkipInvocation || !reflect.DeepEqual(decision.EffectiveArguments, []string{"./gradlew"}) {
		t.Fatalf("decision = %+v/%v", decision, err)
	}
}

func TestCleanRemovalAcceptsProvenPersistentLifecycle(t *testing.T) {
	request := testCleanRemovalRequest()
	request.Workspace = WorkspaceLifecycleContract{Kind: WorkspaceLifecyclePersistent, PersistentLifecycleProven: true, PreventsStaleOutputs: true}
	decision, err := EvaluateCleanRemoval(request)
	if err != nil || !decision.Applied {
		t.Fatalf("decision = %+v/%v", decision, err)
	}
}

func TestCleanRemovalPreservesWholeInvocationForUnsafeContracts(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		mutate func(*CleanRemovalRequest)
	}{
		{name: "unauthorized", reason: "NOT_AUTHORIZED", mutate: func(request *CleanRemovalRequest) { request.Authorized = false }},
		{name: "model unavailable", reason: "MODEL_UNAVAILABLE", mutate: func(request *CleanRemovalRequest) { request.ModelAvailable = false; request.TaskContracts = nil }},
		{name: "failure semantics", reason: "FAILURE_SEMANTICS", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].FailureIsObserved = true }},
		{name: "added actions", reason: "ADDED_ACTIONS", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].AddedActions = 1 }},
		{name: "dependency", reason: "DEPENDENCIES", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].Dependencies = []string{":prepare"} }},
		{name: "finalizer", reason: "FINALIZERS", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].Finalizers = []string{":report"} }},
		{name: "side effect", reason: "SIDE_EFFECTS", mutate: func(request *CleanRemovalRequest) {
			request.TaskContracts[0].SideEffects = []string{"delete-external-directory"}
		}},
		{name: "customized", reason: "CUSTOMIZED_CLEAN", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].Customized = true }},
		{name: "undeclared deletion", reason: "UNDECLARED_DELETION", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].DeletesDeclaredOutputsOnly = false }},
		{name: "non-core", reason: "TASK_NOT_ALLOWLISTED_CORE_CLEAN", mutate: func(request *CleanRemovalRequest) { request.TaskContracts[0].CoreTask = false }},
		{name: "CI barrier", reason: "CI_BARRIER", mutate: func(request *CleanRemovalRequest) { request.Workspace.CleanIsPipelineBarrier = true }},
		{name: "new workspace not empty", reason: "WORKSPACE_NOT_VERIFIED_EMPTY", mutate: func(request *CleanRemovalRequest) { request.Workspace.EmptyVerified = false }},
		{name: "unknown workspace", reason: "WORKSPACE_LIFECYCLE_UNKNOWN", mutate: func(request *CleanRemovalRequest) { request.Workspace = WorkspaceLifecycleContract{} }},
		{name: "persistent stale risk", reason: "PERSISTENT_LIFECYCLE_UNPROVEN", mutate: func(request *CleanRemovalRequest) {
			request.Workspace = WorkspaceLifecycleContract{Kind: WorkspaceLifecyclePersistent, PersistentLifecycleProven: true}
		}},
		{name: "release", reason: "RELEASE_CONTRACT", mutate: func(request *CleanRemovalRequest) { request.ReleaseContract = true }},
		{name: "reproducibility", reason: "REPRODUCIBILITY_VALIDATION", mutate: func(request *CleanRemovalRequest) { request.ReproducibilityValidation = true }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := testCleanRemovalRequest()
			testCase.mutate(&request)
			decision, err := EvaluateCleanRemoval(request)
			if err != nil || decision.Applied || decision.SkipInvocation || decision.Reason != testCase.reason || !reflect.DeepEqual(decision.OriginalArguments, decision.EffectiveArguments) {
				t.Fatalf("decision = %+v/%v", decision, err)
			}
		})
	}
}

func TestCleanRemovalIsAllOrNothingForMultipleCleanTasks(t *testing.T) {
	request := testCleanRemovalRequest()
	request.Arguments = []string{"./gradlew", "clean", ":library:clean", "assemble"}
	request.TaskArgumentIndexes = []int{1, 2, 3}
	decision, err := EvaluateCleanRemoval(request)
	if err != nil || decision.Applied || decision.Reason != "CLEAN_MODEL_INCOMPLETE" || !reflect.DeepEqual(decision.OriginalArguments, decision.EffectiveArguments) {
		t.Fatalf("incomplete = %+v/%v", decision, err)
	}
	second := testCleanTaskContract()
	second.ArgumentIndex = 2
	second.InvocationToken = ":library:clean"
	second.TaskPath = ":library:clean"
	request.TaskContracts = append(request.TaskContracts, second)
	decision, err = EvaluateCleanRemoval(request)
	if err != nil || !decision.Applied || !reflect.DeepEqual(decision.EffectiveArguments, []string{"./gradlew", "assemble"}) {
		t.Fatalf("complete = %+v/%v", decision, err)
	}
}

func TestCleanRemovalUsesModeledTaskPositions(t *testing.T) {
	request := testCleanRemovalRequest()
	request.Arguments = []string{"./gradlew", "--project-dir", "clean", "assemble"}
	request.TaskArgumentIndexes = []int{3}
	request.TaskContracts = nil
	decision, err := EvaluateCleanRemoval(request)
	if err != nil || decision.Applied || decision.Reason != "NO_CLEAN_TASK" || !reflect.DeepEqual(decision.OriginalArguments, decision.EffectiveArguments) {
		t.Fatalf("decision = %+v/%v", decision, err)
	}
}

func TestCleanRemovalRejectsAmbiguousArgumentModel(t *testing.T) {
	request := testCleanRemovalRequest()
	request.TaskArgumentIndexes = []int{2, 1}
	if _, err := EvaluateCleanRemoval(request); err == nil {
		t.Fatal("unsorted task indexes were accepted")
	}
	request = testCleanRemovalRequest()
	request.TaskContracts[0].ArgumentIndex = 2
	decision, err := EvaluateCleanRemoval(request)
	if err != nil || decision.Applied || decision.Reason != "CLEAN_MODEL_INCOMPLETE" {
		t.Fatalf("mismatched contract = %+v/%v", decision, err)
	}
}

func testCleanRemovalRequest() CleanRemovalRequest {
	return CleanRemovalRequest{
		Arguments:           []string{"./gradlew", "clean", "assemble", "--stacktrace"},
		TaskArgumentIndexes: []int{1, 2},
		TaskContracts:       []CleanTaskContract{testCleanTaskContract()},
		Authorized:          true,
		ModelAvailable:      true,
		Workspace:           WorkspaceLifecycleContract{Kind: WorkspaceLifecycleNew, EmptyVerified: true},
	}
}

func testCleanTaskContract() CleanTaskContract {
	return CleanTaskContract{
		ArgumentIndex: 1, InvocationToken: "clean", TaskPath: ":clean", ImplementationType: "org.gradle.api.tasks.Delete", CoreTask: true, DeletesDeclaredOutputsOnly: true,
	}
}
