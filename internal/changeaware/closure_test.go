package changeaware

import (
	"strings"
	"testing"
)

func TestAnalyzeProducesExactPartialClosureForBothDSLs(t *testing.T) {
	for _, dsl := range []string{"KOTLIN", "GROOVY"} {
		t.Run(dsl, func(t *testing.T) {
			report, err := Analyze(validCapture(dsl))
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != StatusTestableActions || !report.InputComplete || report.TestableActions != 1 ||
				report.PerformanceMeasured || report.ActivationAuthorized ||
				len(report.CandidateTasks) != 2 || report.CandidateTasks[0] != ":changed:bundleAll" ||
				report.CandidateTasks[1] != ":changed:emitPayload" ||
				len(report.OmittedOutputs) != 1 || report.OmittedOutputs[0].ProducerTask != ":stable:emitPayload" ||
				!validSHA(report.ActionBindingSHA256) {
				t.Fatalf("unexpected report: %+v", report)
			}
		})
	}
}

func TestAnalyzeConservativeOutcomes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Capture)
		status string
		reason string
	}{
		{
			name:   "global unowned change",
			mutate: func(c *Capture) { c.ChangedPaths = []string{"settings.gradle.kts"} },
			status: StatusNoSafeAction, reason: "CHANGE_INPUT_OWNERSHIP_UNPROVEN",
		},
		{
			name: "ambiguous required producer",
			mutate: func(c *Capture) {
				c.Tasks[1].Outputs = append(c.Tasks[1].Outputs, c.Tasks[0].Outputs[0])
			},
			status: StatusNoSafeAction, reason: "REQUIRED_OUTPUT_PRODUCER_AMBIGUOUS",
		},
		{
			name: "missing omitted output",
			mutate: func(c *Capture) {
				c.Tasks[0].Outputs[0].Exists = false
				c.Tasks[0].Outputs[0].SHA256 = ""
			},
			status: StatusNoSafeAction, reason: "OMITTED_OUTPUT_EVIDENCE_INCOMPLETE",
		},
		{
			name: "unrelated existing input",
			mutate: func(c *Capture) {
				c.ChangedPaths = []string{"unrelated/input.txt"}
				c.RequestedTasks = append(c.RequestedTasks, ":unrelated:observe")
				c.Tasks = append(c.Tasks, TaskEvidence{
					Path:   ":unrelated:observe",
					Inputs: []PathEvidence{{Path: "unrelated/input.txt", Kind: "FILE"}},
				})
			},
			status: StatusNotApplicable, reason: "CHANGE_DOES_NOT_REACH_REQUIRED_OUTPUT",
		},
		{
			name: "producer unavailable",
			mutate: func(c *Capture) {
				c.Status, c.Reason, c.Tasks = CaptureUnavailable, "TASK_INPUTS_UNAVAILABLE", nil
			},
			status: StatusInputUnavailable, reason: "TASK_INPUTS_UNAVAILABLE",
		},
		{
			name: "producer failed",
			mutate: func(c *Capture) {
				c.Status, c.Reason, c.Tasks = CaptureFailed, "GRADLE_CAPTURE_FAILED", nil
			},
			status: StatusProducerFailed, reason: "GRADLE_CAPTURE_FAILED",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := validCapture("KOTLIN")
			test.mutate(&capture)
			report, err := Analyze(capture)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != test.status || report.Reason != test.reason ||
				report.PerformanceMeasured || report.ActivationAuthorized {
				t.Fatalf("unexpected outcome: %+v", report)
			}
		})
	}
}

func TestAnalyzeRejectsInvalidAndCyclicEvidence(t *testing.T) {
	duplicate := validCapture("GROOVY")
	duplicate.ChangedPaths = append(duplicate.ChangedPaths, duplicate.ChangedPaths[0])
	if _, err := Analyze(duplicate); err == nil {
		t.Fatal("duplicate changed path was accepted")
	}

	cycle := validCapture("GROOVY")
	cycle.Tasks[0].DependsOn = []string{":changed:bundleAll"}
	report, err := Analyze(cycle)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusNoSafeAction || report.Reason != "TASK_GRAPH_CYCLIC" {
		t.Fatalf("cyclic evidence did not fail closed: %+v", report)
	}
}

func validCapture(dsl string) Capture {
	digest := func(value string) string { return strings.Repeat(value, 64) }
	return Capture{
		SchemaVersion: CaptureSchemaVersion, GeneratedAt: "2026-08-27T00:00:00Z",
		Family: "fixture-" + strings.ToLower(dsl), DSL: dsl,
		BaseRevision: strings.Repeat("1", 40), TargetRevision: strings.Repeat("2", 40),
		Status: CaptureComplete, ChangedPaths: []string{"changed/input.txt"},
		RequestedTasks: []string{":changed:bundleAll"},
		RequiredOutputs: []string{
			"changed/build/custom-output/payload.bin",
			"stable/build/custom-output/payload.bin",
		},
		Tasks: []TaskEvidence{
			{
				Path:    ":stable:emitPayload",
				Inputs:  []PathEvidence{{Path: "stable/input.txt", Kind: "FILE"}},
				Outputs: []OutputEvidence{{Path: "stable/build/custom-output/payload.bin", Kind: "FILE", SHA256: digest("a"), Exists: true}},
			},
			{
				Path:    ":changed:emitPayload",
				Inputs:  []PathEvidence{{Path: "changed/input.txt", Kind: "FILE"}},
				Outputs: []OutputEvidence{{Path: "changed/build/custom-output/payload.bin", Kind: "FILE", SHA256: digest("b"), Exists: true}},
			},
			{
				Path:      ":changed:bundleAll",
				DependsOn: []string{":changed:emitPayload", ":stable:emitPayload"},
			},
		},
	}
}
