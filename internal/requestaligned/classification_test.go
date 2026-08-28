package requestaligned

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/changeaware"
)

func TestClassifyRelevantTransitionBindsCurrentOmittedOutput(t *testing.T) {
	transition := validTransition()
	transition.BaseCapture.Tasks[1].Outputs[0].Path = "build/right-v1.bin"
	report, err := Classify(transition)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ClassificationRelevantComplete || report.Reason != "EXACT_RELEVANT_PRODUCER_CLOSURE" ||
		report.TestableActions != 1 || report.ActionBindingSHA256 == "" ||
		!reflect.DeepEqual(report.AffectedInputTasks, []string{":leftProducer"}) ||
		!reflect.DeepEqual(report.CandidateTasks, []string{":bundle", ":leftProducer"}) ||
		!reflect.DeepEqual(report.OmittedTasks, []string{":rightProducer"}) {
		t.Fatalf("unexpected relevant classification: %+v", report)
	}
	if len(report.OmittedOutputs) != 1 || report.OmittedOutputs[0].Path != "build/right-v2.bin" ||
		report.OmittedOutputs[0].ProducerTask != ":rightProducer" {
		t.Fatalf("current renamed output was not bound: %+v", report.OmittedOutputs)
	}
}

func TestClassifyRequiredStates(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Transition)
		status string
		reason string
	}{
		{
			name: "irrelevant", mutate: func(value *Transition) { value.ChangedPaths = []string{"docs/note.md"} },
			status: ClassificationIrrelevant, reason: "NO_CHANGED_PATH_INTERSECTS_REQUEST_INPUTS",
		},
		{
			name: "global identity drift", mutate: func(value *Transition) {
				value.TargetCapture.BuildLogicFiles[0].SHA256 = strings.Repeat("9", 64)
			},
			status: ClassificationGlobalOrAmbiguous, reason: "REQUEST_IDENTITY_CHANGED",
		},
		{
			name: "ambiguous producer", mutate: func(value *Transition) {
				value.TargetCapture.Tasks[0].Outputs = append(value.TargetCapture.Tasks[0].Outputs,
					value.TargetCapture.Tasks[1].Outputs[0])
			},
			status: ClassificationGlobalOrAmbiguous, reason: "CURRENT_OUTPUT_PRODUCER_AMBIGUOUS",
		},
		{
			name: "missing outputs", mutate: func(value *Transition) {
				for index := range value.TargetCapture.Tasks {
					value.TargetCapture.Tasks[index].Outputs = nil
				}
			},
			status: ClassificationInputUnavailable, reason: "CURRENT_PRODUCER_OUTPUTS_EMPTY",
		},
		{
			name: "producer failed", mutate: func(value *Transition) {
				value.TargetCapture.Status = CaptureFailed
				value.TargetCapture.Reason = "GRADLE_CAPTURE_FAILED"
				value.TargetCapture.Tasks = nil
			},
			status: ClassificationProducerFailed, reason: "REQUEST_PRODUCER_FAILED",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transition := validTransition()
			test.mutate(&transition)
			report, err := Classify(transition)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != test.status || report.Reason != test.reason ||
				report.TestableActions != 0 || report.PerformanceMeasured || report.ActivationAuthorized {
				t.Fatalf("unexpected classification: %+v", report)
			}
		})
	}
}

func TestClassifyRelevantFullGraphDoesNotInventAction(t *testing.T) {
	transition := validTransition()
	transition.ChangedPaths = []string{"inputs/left.txt", "inputs/right.txt"}
	report, err := Classify(transition)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ClassificationRelevantComplete || report.Reason != "FULL_REQUEST_GRAPH_REQUIRED" ||
		report.TestableActions != 0 || report.ActionBindingSHA256 != "" || len(report.OmittedOutputs) != 0 {
		t.Fatalf("full graph became an action: %+v", report)
	}
}

func TestClassifyRejectsInvalidTransitionIdentity(t *testing.T) {
	transition := validTransition()
	transition.TargetRevision = transition.BaseRevision
	if _, err := Classify(transition); err == nil {
		t.Fatal("equal revisions were accepted")
	}
}

func validTransition() Transition {
	base := classifierCapture()
	target := classifierCapture()
	return Transition{
		SchemaVersion: TransitionSchemaVersion, GeneratedAt: "2026-08-28T00:00:00Z",
		BaseRevision: strings.Repeat("a", 40), TargetRevision: strings.Repeat("b", 40),
		ChangedPaths: []string{"inputs/left.txt"}, BaseCapture: base, TargetCapture: target,
	}
}

func classifierCapture() Capture {
	digest := func(value string) string { return strings.Repeat(value, 64) }
	return Capture{
		SchemaVersion: CaptureSchemaVersion, GeneratedAt: "2026-08-28T00:00:00Z",
		Status: CaptureComplete, GradleArguments: []string{":bundle", "--no-daemon"},
		RequestedTasks: []string{":bundle"}, GradleVersion: "9.6.1",
		JavaRuntime: JavaRuntime{Version: "21.0.12", Vendor: "Eclipse Adoptium",
			RuntimeName: "OpenJDK Runtime Environment", VMName: "OpenJDK 64-Bit Server VM", Architecture: "amd64"},
		EnvironmentBindingSHA256: digest("a"),
		WrapperFiles:             []FileBinding{{Path: "gradle/wrapper/gradle-wrapper.properties", SHA256: digest("b")}},
		BuildLogicFiles:          []FileBinding{{Path: "build.gradle", SHA256: digest("c")}},
		Tasks: []changeaware.TaskEvidence{
			{Path: ":leftProducer", Inputs: []changeaware.PathEvidence{{Path: "inputs/left.txt", Kind: "FILE"}},
				Outputs: []changeaware.OutputEvidence{{Path: "build/left.bin", Kind: "FILE", SHA256: digest("d"), Exists: true}}},
			{Path: ":rightProducer", Inputs: []changeaware.PathEvidence{{Path: "inputs/right.txt", Kind: "FILE"}},
				Outputs: []changeaware.OutputEvidence{{Path: "build/right-v2.bin", Kind: "FILE", SHA256: digest("e"), Exists: true}}},
			{Path: ":bundle", DependsOn: []string{":leftProducer", ":rightProducer"},
				Inputs:  []changeaware.PathEvidence{{Path: "build/left.bin", Kind: "FILE"}, {Path: "build/right-v2.bin", Kind: "FILE"}},
				Outputs: []changeaware.OutputEvidence{{Path: "build/bundle.bin", Kind: "FILE", SHA256: digest("f"), Exists: true}}},
		},
	}
}
