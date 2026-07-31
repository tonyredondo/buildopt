package taskintelligence

import "testing"

const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func pilotContract() ReviewedContract {
	return ReviewedContract{TaskType: "GeneratePilotManifest", ImplementationDigest: digest, ContractDigest: digest, Route: "REVIEWED_SOURCE_PATCH", Inputs: []RegisteredPath{{Path: "pilot/payload.txt", Digest: digest}}, Outputs: []string{"build/pilot/manifest.txt"}}
}

func activePilot(t *testing.T) Qualification {
	t.Helper()
	q, err := NewQualification().Observe()
	if err != nil {
		t.Fatal(err)
	}
	q, err = q.QualifyReviewed(pilotContract())
	if err != nil {
		t.Fatal(err)
	}
	q, err = q.ValidateQuarantine(QuarantineEvidence{EveryInputMutationChangedKey: true, Repeatable: true, Relocatable: true, ArtifactsExact: true, RelevantValidationPassed: true})
	if err != nil {
		t.Fatal(err)
	}
	q, err = q.Activate()
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestReviewedSourcePatchReachesActiveThroughEveryState(t *testing.T) {
	if got := activePilot(t).State; got != Active {
		t.Fatalf("state = %s", got)
	}
}

func TestHistoryAgentAndIncompleteCoverageNeverQualify(t *testing.T) {
	q, _ := NewQualification().Observe()
	for _, route := range []string{"HISTORY", "JVM_AGENT_DIAGNOSTIC", "UNAVAILABLE_HELPER"} {
		contract := pilotContract()
		contract.Route = route
		if _, err := q.QualifyReviewed(contract); err == nil {
			t.Fatalf("route %s qualified", route)
		}
	}
	coverage := TraceCoverage{Dimensions: map[string]CoverageStatus{"FILESYSTEM": CoverageExact}}
	if decision := EvaluateTrace(coverage); decision.TraceComplete || decision.Qualification != "INCONCLUSIVE" {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestTestTasksAndIncompleteContractsStayExcluded(t *testing.T) {
	q, _ := NewQualification().Observe()
	contract := pilotContract()
	contract.IsTest = true
	if _, err := q.QualifyReviewed(contract); err == nil {
		t.Fatal("Test task qualified")
	}
	contract = pilotContract()
	contract.Inputs = nil
	if _, err := q.QualifyReviewed(contract); err == nil {
		t.Fatal("input-free task qualified")
	}
}

func TestEveryInputMutationAndValidationAreMandatory(t *testing.T) {
	q, _ := NewQualification().Observe()
	q, _ = q.QualifyReviewed(pilotContract())
	complete := QuarantineEvidence{EveryInputMutationChangedKey: true, Repeatable: true, Relocatable: true, ArtifactsExact: true, RelevantValidationPassed: true}
	for index := 0; index < 5; index++ {
		candidate := complete
		switch index {
		case 0:
			candidate.EveryInputMutationChangedKey = false
		case 1:
			candidate.Repeatable = false
		case 2:
			candidate.Relocatable = false
		case 3:
			candidate.ArtifactsExact = false
		case 4:
			candidate.RelevantValidationPassed = false
		}
		if _, err := q.ValidateQuarantine(candidate); err == nil {
			t.Fatalf("missing gate %d passed", index)
		}
	}
}

func TestDiscrepancySuspendsWithoutChangingBaseline(t *testing.T) {
	q, _ := NewQualification().Observe()
	q, _ = q.QualifyReviewed(pilotContract())
	q, err := q.ValidateQuarantine(QuarantineEvidence{Discrepancy: true})
	if err != nil || q.State != Suspended {
		t.Fatalf("result = %+v, %v", q, err)
	}
}

func TestExactCorrelationPublishesOnlyActiveTaskKeys(t *testing.T) {
	q := activePilot(t)
	decision := Correlate(q.State, []CorrelationEvent{{TaskExecutionID: "task-1", CacheKey: "key-1", TaskOutcome: "SUCCESS", PutCompleted: true, Attributed: true}})
	if decision.Capability != "EXACT" || decision.AttemptAborted || len(decision.Keys) != 1 {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestAnyUnattributedOrCrashAbortsWholeAttempt(t *testing.T) {
	q := activePilot(t)
	events := []CorrelationEvent{{TaskExecutionID: "task-1", CacheKey: "key-1", TaskOutcome: "SUCCESS", PutCompleted: true, Attributed: true}, {CacheKey: "global", PutCompleted: true}}
	if decision := Correlate(q.State, events); !decision.AttemptAborted || len(decision.Keys) != 0 || decision.Reason != "UNATTRIBUTED" {
		t.Fatalf("decision = %+v", decision)
	}
	if decision := Correlate(q.Suspend("AGENT_CRASH").State, events[:1]); !decision.AttemptAborted {
		t.Fatalf("suspended decision = %+v", decision)
	}
}

func TestCompleteCoverageRequiresEveryDimensionAndNoDrops(t *testing.T) {
	dimensions := map[string]CoverageStatus{}
	for _, dimension := range requiredDimensions {
		dimensions[dimension] = CoverageExact
	}
	if decision := EvaluateTrace(TraceCoverage{Dimensions: dimensions}); !decision.TraceComplete {
		t.Fatalf("decision = %+v", decision)
	}
	if decision := EvaluateTrace(TraceCoverage{Dimensions: dimensions, Dropped: 1}); decision.TraceComplete {
		t.Fatalf("dropped decision = %+v", decision)
	}
}
