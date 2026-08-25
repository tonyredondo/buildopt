package adaptivefragment

import (
	"reflect"
	"strings"
	"testing"
)

func TestTaskContractOpportunityIsGenericAndNonAuthorizing(t *testing.T) {
	input := validPatchOpportunityInput()
	got, err := DetectTaskContractOpportunity(input)
	if err != nil {
		t.Fatal(err)
	}
	renamed := input
	renamed.RepositoryScopeSHA256 = sha("another-repository")
	renamed.RelativeSourcePath = "plugin/src/main/java/example/OtherTask.java"
	renamedGot, err := DetectTaskContractOpportunity(renamed)
	if err != nil {
		t.Fatal(err)
	}
	got.RelativeSourcePath = renamedGot.RelativeSourcePath
	if !reflect.DeepEqual(got, renamedGot) {
		t.Fatalf("repository/path rename changed generic decision:\n%+v\n%+v", got, renamedGot)
	}
	if got.Status != PatchOpportunityStatusProposed || got.Kind != PatchOpportunityKindTaskContract ||
		!got.OwnerReviewRequired || !got.TransactionalValidationRequired || !got.ExactRevertRequired ||
		got.PatchAuthorized || got.ActivationAuthorized {
		t.Fatalf("detector transferred unsafe authority: %+v", got)
	}
}

func TestTaskContractOpportunityRejectsUnsafeEvidence(t *testing.T) {
	cases := map[string]func(*PatchOpportunityInput){
		"too few builds": func(value *PatchOpportunityInput) { value.Observations = value.Observations[:2] },
		"cacheable": func(value *PatchOpportunityInput) { value.Observations[0].Cacheable = true },
		"up to date": func(value *PatchOpportunityInput) { value.Observations[0].UpToDate = true },
		"product failure": func(value *PatchOpportunityInput) { value.Observations[0].ProductFailure = true },
		"unstable input": func(value *PatchOpportunityInput) { value.Observations[0].InputSnapshotSHA256 = strings.Repeat("a", 64) },
		"unstable output": func(value *PatchOpportunityInput) { value.Observations[0].OutputSnapshotSHA256 = strings.Repeat("b", 64) },
		"unknown side effect": func(value *PatchOpportunityInput) { value.Facts.UnknownSideEffects = true },
		"ambiguous task action": func(value *PatchOpportunityInput) { value.Facts.TaskActionCount = 2 },
		"unsafe path": func(value *PatchOpportunityInput) { value.RelativeSourcePath = "../Task.java" },
		"cheap task": func(value *PatchOpportunityInput) {
			for index := range value.Observations { value.Observations[index].DurationMs = 100 }
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := validPatchOpportunityInput()
			mutate(&input)
			if _, err := DetectTaskContractOpportunity(input); err == nil {
				t.Fatal("unsafe opportunity evidence was accepted")
			}
		})
	}
}

func validPatchOpportunityInput() PatchOpportunityInput {
	return PatchOpportunityInput{
		EvidenceSHA256: sha("opportunity-evidence"), RepositoryScopeSHA256: sha("repository"),
		TaskImplementationSHA256: sha("task-implementation"),
		RelativeSourcePath: "buildSrc/src/main/java/example/GenerateManifest.java",
		SourcePreimageSHA256: strings.Repeat("3", 64),
		Facts: JavaTaskContractFacts{ExtendsDefaultTask: true, InternalInputCount: 1, InternalOutputCount: 1, TaskActionCount: 1},
		Observations: []TaskContractObservation{
			patchObservation(1, 1100), patchObservation(2, 900), patchObservation(3, 1300),
		},
	}
}

func patchObservation(ordinal, duration uint64) TaskContractObservation {
	return TaskContractObservation{RequestedBuildOrdinal: ordinal, DurationMs: duration, Executed: true,
		InputSnapshotSHA256: strings.Repeat("1", 64), OutputSnapshotSHA256: strings.Repeat("2", 64)}
}
