package durablecatalog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

func TestGraphBreadthDetectorIsScopeIndependentAndNonAuthorizing(t *testing.T) {
	firstInput := validGraphInput("first-scope")
	first, err := DetectGraphBreadthOpportunity(firstInput)
	if err != nil {
		t.Fatal(err)
	}
	secondInput := validGraphInput("renamed-scope")
	secondInput.Workflow = "renamed-workflow"
	second, err := DetectGraphBreadthOpportunity(secondInput)
	if err != nil {
		t.Fatal(err)
	}
	first.Workflow = second.Workflow
	if first != second {
		t.Fatalf("scope or workflow changed structural fields: %+v / %+v", first, second)
	}
	if first.Status != StatusProposed || first.Kind != KindGraphBreadth ||
		first.OmittedProjectCount != 2 || !first.OwnerReviewRequired ||
		!first.TransactionalValidationRequired || !first.ExactRevertRequired ||
		first.PatchAuthorized || first.ActivationAuthorized {
		t.Fatalf("graph detector transferred authority: %+v", first)
	}
}

func TestGraphBreadthDetectorRejectsUnsafeEvidence(t *testing.T) {
	cases := map[string]func(*GraphBreadthInput){
		"too few builds": func(input *GraphBreadthInput) { input.Observations = input.Observations[:2] },
		"candidate is full graph": func(input *GraphBreadthInput) {
			input.Observations[0].CandidateProjectCount = input.Observations[0].FullProjectCount
		},
		"output mismatch": func(input *GraphBreadthInput) {
			input.Observations[0].CandidateOutputSHA256 = strings.Repeat("f", 64)
		},
		"product failure": func(input *GraphBreadthInput) { input.Observations[0].ProductFailure = true },
		"duplicate ordinal": func(input *GraphBreadthInput) {
			input.Observations[1].RequestedBuildOrdinal = input.Observations[0].RequestedBuildOrdinal
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := validGraphInput("unsafe")
			mutate(&input)
			if _, err := DetectGraphBreadthOpportunity(input); err == nil {
				t.Fatal("unsafe graph evidence was accepted")
			}
		})
	}
}

func TestPatchTransactionUsesExactBytesAndNoCheckout(t *testing.T) {
	preimage := []byte("before\n")
	postimage := []byte("after\n")
	proof, err := ProvePatchTransaction(preimage, postimage)
	if err != nil {
		t.Fatal(err)
	}
	if proof.PreimageSHA256 != DigestBytes(preimage) || proof.PostimageSHA256 != DigestBytes(postimage) ||
		!proof.AppliedOutsideCheckout || !proof.CheckoutUnchanged || !proof.ExactRevertRestoredPreimage ||
		proof.RejectedProposalMutations != 0 {
		t.Fatalf("unexpected transaction proof: %+v", proof)
	}
	if bytes.Equal(preimage, postimage) {
		t.Fatal("test images unexpectedly equal")
	}
}

func TestMeasurementRequiresPositiveMajorityAndNativeExecution(t *testing.T) {
	base := Measurement{
		Pairs: 8, ControlMeanMs: 2000, CandidateMeanMs: 1000, MeanSavedMs: 1000,
		MeanReductionRatio: 0.5, Interval95SavedMs: []float64{500, 1500},
		PositivePairs: 5, RequiredOutputsIdentical: true,
	}
	if !base.Qualifies() {
		t.Fatal("positive native measurement did not qualify")
	}
	base.PositivePairs = 4
	if base.Qualifies() {
		t.Fatal("non-majority measurement qualified")
	}
	base.PositivePairs = 5
	base.BuildOptRequiredAfterAcceptance = true
	if base.Qualifies() {
		t.Fatal("runtime-dependent durable patch qualified")
	}
}

func validGraphInput(scope string) GraphBreadthInput {
	observations := make([]GraphBreadthObservation, 0, 3)
	for ordinal := uint64(1); ordinal <= 3; ordinal++ {
		observations = append(observations, GraphBreadthObservation{
			RequestedBuildOrdinal: ordinal, FullProjectCount: 3, CandidateProjectCount: 1,
			FullOutputSHA256: strings.Repeat("1", 64), CandidateOutputSHA256: strings.Repeat("1", 64),
		})
	}
	return GraphBreadthInput{
		EvidenceSHA256: strings.Repeat("2", 64), RepositoryScopeSHA256: DigestBytes([]byte(scope)),
		ManifestSHA256: strings.Repeat("3", 64), GraphSHA256: strings.Repeat("4", 64),
		Workflow: "assemble", Observations: observations,
	}
}

func validTaskInput() adaptivefragment.PatchOpportunityInput {
	observations := []adaptivefragment.TaskContractObservation{
		{RequestedBuildOrdinal: 1, DurationMs: 1200, Executed: true, InputSnapshotSHA256: strings.Repeat("1", 64), OutputSnapshotSHA256: strings.Repeat("2", 64)},
		{RequestedBuildOrdinal: 2, DurationMs: 1000, Executed: true, InputSnapshotSHA256: strings.Repeat("1", 64), OutputSnapshotSHA256: strings.Repeat("2", 64)},
		{RequestedBuildOrdinal: 3, DurationMs: 1400, Executed: true, InputSnapshotSHA256: strings.Repeat("1", 64), OutputSnapshotSHA256: strings.Repeat("2", 64)},
	}
	return adaptivefragment.PatchOpportunityInput{
		EvidenceSHA256: strings.Repeat("3", 64), RepositoryScopeSHA256: strings.Repeat("4", 64),
		TaskImplementationSHA256: strings.Repeat("5", 64), RelativeSourcePath: "buildSrc/src/main/java/example/Task.java",
		SourcePreimageSHA256: strings.Repeat("6", 64), Facts: adaptivefragment.JavaTaskContractFacts{
			ExtendsDefaultTask: true, InternalInputCount: 1, InternalOutputCount: 1, TaskActionCount: 1,
		}, Observations: observations,
	}
}

func TestTaskProposalRetainsExistingNonAuthorityBoundary(t *testing.T) {
	proposal, err := TaskProposal(validTaskInput())
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != adaptivefragment.PatchOpportunityStatusProposed || proposal.PatchAuthorized || proposal.ActivationAuthorized {
		t.Fatalf("task proposal is not review-only: %+v", proposal)
	}
}
