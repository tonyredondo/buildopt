package durablecatalog

import (
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

func TestFreshEvidenceProducesTaskActionForBothDSLs(t *testing.T) {
	for _, dsl := range []string{"KOTLIN", "GROOVY"} {
		t.Run(dsl, func(t *testing.T) {
			input := validFreshEvidenceInput(dsl)
			input.DeclaredGraph.Candidates = nil
			report, err := ProduceFreshEvidence(input)
			if err != nil {
				t.Fatal(err)
			}
			if !report.InputComplete || report.TestableActions != 1 ||
				report.Detectors[0].Status != DetectorStatusTestableActions ||
				report.Detectors[1].Status != DetectorStatusNoOpportunity {
				t.Fatalf("unexpected report: %+v", report)
			}
			action := report.Detectors[0].Actions[0]
			if action.PatchAuthorized || action.ActivationAuthorized || !action.OwnerReviewRequired ||
				action.Transaction.PreimageSHA256 == action.Transaction.PostimageSHA256 ||
				action.TrialCost.Status != "UNAVAILABLE_NOT_MEASURED" ||
				action.TaskImplementationSHA256 == "" || action.InputSnapshotSHA256 == "" ||
				action.OutputSnapshotSHA256 == "" || action.RequestedBuilds != 3 ||
				action.DeclaredInputCount != 1 || action.DeclaredOutputCount != 1 ||
				action.CacheableBefore || !action.RequestedWorkflowReachable {
				t.Fatalf("unsafe task action: %+v", action)
			}
		})
	}
}

func TestFreshEvidenceProducesGraphAction(t *testing.T) {
	input := validFreshEvidenceInput("GROOVY")
	input.TaskContract.Candidates = nil
	report, err := ProduceFreshEvidence(input)
	if err != nil {
		t.Fatal(err)
	}
	if !report.InputComplete || report.TestableActions != 1 ||
		report.Detectors[0].Status != DetectorStatusNotApplicable ||
		report.Detectors[1].Status != DetectorStatusTestableActions {
		t.Fatalf("unexpected report: %+v", report)
	}
	action := report.Detectors[1].Actions[0]
	if action.FullGraphSHA256 == "" || action.OmittedCriticalPathSHA256 == "" ||
		action.ExactOutputClosureSHA256 == "" || action.RequestedBuilds != 3 ||
		!action.RequestedWorkflowReachable {
		t.Fatalf("incomplete graph binding: %+v", action)
	}
}

func TestFreshEvidenceConclusiveNoOpportunity(t *testing.T) {
	input := validFreshEvidenceInput("KOTLIN")
	input.TaskContract.Candidates[0].Input.Observations[1].OutputSnapshotSHA256 = strings.Repeat("9", 64)
	input.DeclaredGraph.Candidates = nil
	report, err := ProduceFreshEvidence(input)
	if err != nil {
		t.Fatal(err)
	}
	if !report.InputComplete || report.TestableActions != 0 ||
		report.Detectors[0].Status != DetectorStatusNoOpportunity ||
		report.Detectors[1].Status != DetectorStatusNoOpportunity {
		t.Fatalf("unexpected no-opportunity report: %+v", report)
	}
}

func TestFreshEvidenceNotApplicable(t *testing.T) {
	input := validFreshEvidenceInput("GROOVY")
	input.TaskContract.Candidates = nil
	input.DeclaredGraph.Candidates = nil
	report, err := ProduceFreshEvidence(input)
	if err != nil {
		t.Fatal(err)
	}
	if !report.InputComplete || report.Detectors[0].Status != DetectorStatusNotApplicable ||
		report.Detectors[1].Status != DetectorStatusNoOpportunity {
		t.Fatalf("unexpected not-applicable report: %+v", report)
	}
}

func TestFreshEvidenceUnavailableBlocksCompleteness(t *testing.T) {
	input := validFreshEvidenceInput("KOTLIN")
	input.TaskContract = TaskProducerInput{CaptureStatus: CaptureUnavailable, Reason: "TASK_TRACE_UNAVAILABLE"}
	input.DeclaredGraph.Candidates = nil
	report, err := ProduceFreshEvidence(input)
	if err != nil {
		t.Fatal(err)
	}
	if report.InputComplete || report.Detectors[0].Status != DetectorStatusInputUnavailable ||
		report.Detectors[1].Status != DetectorStatusNoOpportunity {
		t.Fatalf("unavailable input became conclusive: %+v", report)
	}
}

func TestFreshEvidenceProducerFailureBlocksCompleteness(t *testing.T) {
	input := validFreshEvidenceInput("GROOVY")
	input.TaskContract.Candidates = nil
	input.DeclaredGraph = GraphProducerInput{CaptureStatus: CaptureFailed, Reason: "TRACE_DECODER_FAILED"}
	report, err := ProduceFreshEvidence(input)
	if err != nil {
		t.Fatal(err)
	}
	if report.InputComplete || report.Detectors[1].Status != DetectorStatusProducerFailed {
		t.Fatalf("producer failure became conclusive: %+v", report)
	}
}

func TestFreshEvidenceRejectsUnboundSource(t *testing.T) {
	input := validFreshEvidenceInput("KOTLIN")
	input.TaskContract.Candidates[0].SourcePreimage = "changed"
	if _, err := ProduceFreshEvidence(input); err == nil {
		t.Fatal("source preimage drift was accepted")
	}
}

func validFreshEvidenceInput(dsl string) FreshEvidenceInput {
	digest := func(character string) string { return strings.Repeat(character, 64) }
	preimage := "class Generate extends DefaultTask {}"
	postimage := "@CacheableTask class Generate extends DefaultTask {}"
	observations := make([]adaptivefragment.TaskContractObservation, 0, 3)
	graphObservations := make([]GraphBreadthObservation, 0, 3)
	for ordinal := uint64(1); ordinal <= 3; ordinal++ {
		observations = append(observations, adaptivefragment.TaskContractObservation{
			RequestedBuildOrdinal: ordinal, DurationMs: 700, Executed: true,
			InputSnapshotSHA256: digest("4"), OutputSnapshotSHA256: digest("5"),
		})
		graphObservations = append(graphObservations, GraphBreadthObservation{
			RequestedBuildOrdinal: ordinal, FullProjectCount: 12, CandidateProjectCount: 4,
			FullOutputSHA256: digest("6"), CandidateOutputSHA256: digest("6"),
		})
	}
	return FreshEvidenceInput{
		SchemaVersion: FreshEvidenceSchemaVersion, GeneratedAt: "2026-08-27T00:00:00Z",
		FamilyKey: "fixture-family", DSL: dsl, RepositoryScopeSHA256: digest("1"),
		WorkflowArgumentsSHA256: digest("2"), OutputContract: []string{"build/libs/app.jar"},
		TaskContract: TaskProducerInput{CaptureStatus: CaptureComplete, Candidates: []TaskCandidateEvidence{{
			Input: adaptivefragment.PatchOpportunityInput{
				EvidenceSHA256: digest("3"), RepositoryScopeSHA256: digest("1"),
				TaskImplementationSHA256: digest("7"), RelativeSourcePath: "buildSrc/src/main/java/Generate.java",
				SourcePreimageSHA256: DigestBytes([]byte(preimage)),
				Facts: adaptivefragment.JavaTaskContractFacts{ExtendsDefaultTask: true, InternalInputCount: 1,
					InternalOutputCount: 1, TaskActionCount: 1}, Observations: observations,
			},
			SourcePreimage: preimage, SourcePostimage: postimage,
			ValidationCommand: []string{"./gradlew", "assemble"},
		}}},
		DeclaredGraph: GraphProducerInput{CaptureStatus: CaptureComplete, Candidates: []GraphCandidateEvidence{{
			Input: GraphBreadthInput{EvidenceSHA256: digest("8"), RepositoryScopeSHA256: digest("1"),
				ManifestSHA256: digest("9"), GraphSHA256: digest("a"), Workflow: "assemble",
				Observations: graphObservations},
			CandidatePlanSHA256: digest("a"), BindingDigest: digest("9"), ProjectedSavingNs: 900000000,
			OmittedCriticalPathSHA256: digest("b"), ExactOutputClosureSHA256: digest("6"),
			SourcePreimage:    "tasks.named(\"assemble\") { dependsOn(\":a:jar\", \"b:jar\") }",
			SourcePostimage:   "tasks.named(\"assemble\") { dependsOn(\":a:jar\") }",
			ValidationCommand: []string{"./gradlew", "assemble"},
		}}},
	}
}
