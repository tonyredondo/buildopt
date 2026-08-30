package requestaligned

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/changeaware"
)

func TestProduceStablePortableIdentityAndCurrentVersionedOutput(t *testing.T) {
	capture := validCapture()
	first, err := Produce(capture)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Produce(capture)
	if err != nil {
		t.Fatal(err)
	}
	if first.RequestIdentitySHA256 != second.RequestIdentitySHA256 ||
		first.Status != StatusComplete || first.PerformanceMeasured || first.ActivationAuthorized {
		t.Fatalf("unexpected portable identity: first=%+v second=%+v", first, second)
	}
	found := false
	for _, output := range first.CurrentOutputs {
		if output.ProducerTask == ":jar" && output.Path == "build/libs/groovy-raw-6.0.0-SNAPSHOT-raw.jar" {
			found = true
		}
	}
	if !found {
		t.Fatalf("current versioned producer output was not discovered: %+v", first.CurrentOutputs)
	}
}

func TestProduceChangesIdentityForCompatibilityDimensions(t *testing.T) {
	base := validCapture()
	original, err := Produce(base)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*Capture)
	}{
		{name: "argument vector", mutate: func(value *Capture) { value.GradleArguments = []string{":jar", "--stacktrace"} }},
		{name: "requested task", mutate: func(value *Capture) { value.RequestedTasks = []string{":compileJava"} }},
		{name: "Wrapper", mutate: func(value *Capture) { value.WrapperFiles[0].SHA256 = strings.Repeat("9", 64) }},
		{name: "Gradle", mutate: func(value *Capture) { value.GradleVersion = "8.14.3" }},
		{name: "JDK", mutate: func(value *Capture) { value.JavaRuntime.Version = "17.0.12" }},
		{name: "environment", mutate: func(value *Capture) { value.EnvironmentBindingSHA256 = strings.Repeat("8", 64) }},
		{name: "build logic", mutate: func(value *Capture) { value.BuildLogicFiles[0].SHA256 = strings.Repeat("7", 64) }},
		{name: "task graph", mutate: func(value *Capture) { value.Tasks[1].DependsOn = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := validCapture()
			test.mutate(&changed)
			observation, produceErr := Produce(changed)
			if produceErr != nil {
				t.Fatal(produceErr)
			}
			if observation.RequestIdentitySHA256 == original.RequestIdentitySHA256 {
				t.Fatalf("%s did not change request identity", test.name)
			}
		})
	}
}

func TestProduceRejectsIncompleteCurrentProducerEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Capture)
		reason string
	}{
		{
			name: "ambiguous producer",
			mutate: func(value *Capture) {
				value.Tasks[0].Outputs = append(value.Tasks[0].Outputs, value.Tasks[1].Outputs[0])
			},
			reason: "CURRENT_OUTPUT_PRODUCER_AMBIGUOUS",
		},
		{
			name: "empty outputs",
			mutate: func(value *Capture) {
				for index := range value.Tasks {
					value.Tasks[index].Outputs = nil
				}
			},
			reason: "CURRENT_PRODUCER_OUTPUTS_EMPTY",
		},
		{
			name:   "requested task outside graph",
			mutate: func(value *Capture) { value.RequestedTasks = []string{":missing"} },
			reason: "REQUESTED_TASK_UNAVAILABLE",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := validCapture()
			test.mutate(&capture)
			observation, err := Produce(capture)
			if err != nil {
				t.Fatal(err)
			}
			if observation.Status != StatusUnavailable || observation.Reason != test.reason ||
				observation.PerformanceMeasured || observation.ActivationAuthorized {
				t.Fatalf("unexpected unavailable observation: %+v", observation)
			}
		})
	}
}

func TestProduceRejectsInvalidBindingsAndPreservesFailedBoundary(t *testing.T) {
	invalid := validCapture()
	invalid.BuildLogicFiles[0].Path = "../settings.gradle"
	if _, err := Produce(invalid); err == nil {
		t.Fatal("outside-repository build logic was accepted")
	}

	failed := validCapture()
	failed.Status, failed.Reason, failed.Tasks = CaptureFailed, "GRADLE_CAPTURE_FAILED", nil
	observation, err := Produce(failed)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Status != StatusFailed || observation.Reason != "GRADLE_CAPTURE_FAILED" ||
		observation.PerformanceMeasured || observation.ActivationAuthorized {
		t.Fatalf("failed boundary changed authority: %+v", observation)
	}
}

func TestProducePreservesBoundedUnavailableCaptureDiagnostics(t *testing.T) {
	capture := precisionCapture()
	capture.Status, capture.Reason, capture.Tasks = CaptureUnavailable, "TASK_INPUT_EVIDENCE_UNAVAILABLE", nil
	capture.Diagnostics = &CaptureDiagnostics{
		Phase: "TASK_INPUT_EVIDENCE_UNAVAILABLE",
		InputFailures: []CaptureInputFailure{
			{TaskPath: ":compileTestJava", FailureClass: "org.gradle.api.GradleException"},
		},
		MissingAfterTaskPaths: []string{":testClasses"},
	}
	observation, err := Produce(capture)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Status != StatusUnavailable || observation.Reason != "TASK_INPUT_EVIDENCE_UNAVAILABLE" ||
		observation.PerformanceMeasured || observation.ActivationAuthorized {
		t.Fatalf("diagnostic capture changed authority: %+v", observation)
	}
}

func TestProduceRejectsUnsafeOrMisboundCaptureDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CaptureDiagnostics)
	}{
		{name: "wrong phase", mutate: func(value *CaptureDiagnostics) { value.Phase = "OTHER" }},
		{name: "exception message", mutate: func(value *CaptureDiagnostics) { value.InputFailures[0].FailureClass = "failure\n/home/user" }},
		{name: "duplicate task", mutate: func(value *CaptureDiagnostics) { value.MissingAfterTaskPaths = []string{":compileTestJava"} }},
		{name: "unsorted failures", mutate: func(value *CaptureDiagnostics) {
			value.InputFailures = append(value.InputFailures, CaptureInputFailure{TaskPath: ":a", FailureClass: "java.lang.IllegalStateException"})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := precisionCapture()
			capture.Status, capture.Reason, capture.Tasks = CaptureUnavailable, "TASK_INPUT_EVIDENCE_UNAVAILABLE", nil
			capture.Diagnostics = &CaptureDiagnostics{
				Phase:         "TASK_INPUT_EVIDENCE_UNAVAILABLE",
				InputFailures: []CaptureInputFailure{{TaskPath: ":compileTestJava", FailureClass: "org.gradle.api.GradleException"}},
			}
			test.mutate(capture.Diagnostics)
			if _, err := Produce(capture); err == nil {
				t.Fatal("unsafe diagnostics were accepted")
			}
		})
	}

	complete := precisionCapture()
	complete.Diagnostics = &CaptureDiagnostics{Phase: "TASK_INPUT_EVIDENCE_UNAVAILABLE", MissingAfterTaskPaths: []string{":testClasses"}}
	if _, err := Produce(complete); err == nil {
		t.Fatal("diagnostics on a complete capture were accepted")
	}
}

func validCapture() Capture {
	digest := func(value string) string { return strings.Repeat(value, 64) }
	return Capture{
		SchemaVersion: CaptureSchemaVersionV1, GeneratedAt: "2026-08-28T00:00:00Z",
		Status: CaptureComplete, GradleArguments: []string{":jar", "--no-daemon"},
		RequestedTasks: []string{":jar"}, GradleVersion: "9.6.1",
		JavaRuntime: JavaRuntime{
			Version: "21.0.12", Vendor: "Eclipse Adoptium", RuntimeName: "OpenJDK Runtime Environment",
			VMName: "OpenJDK 64-Bit Server VM", Architecture: "amd64",
		},
		EnvironmentBindingSHA256: digest("a"),
		WrapperFiles: []FileBinding{
			{Path: "gradle/wrapper/gradle-wrapper.jar", SHA256: digest("b")},
			{Path: "gradle/wrapper/gradle-wrapper.properties", SHA256: digest("c")},
		},
		BuildLogicFiles: []FileBinding{{Path: "build.gradle", SHA256: digest("d")}, {Path: "settings.gradle", SHA256: digest("e")}},
		Tasks: []changeaware.TaskEvidence{
			{
				Path: ":compileJava", Inputs: []changeaware.PathEvidence{{Path: "src/main/java/Example.java", Kind: "FILE"}},
				Outputs: []changeaware.OutputEvidence{{Path: "build/classes/java/main", Kind: "DIRECTORY", SHA256: digest("f"), Exists: true}},
			},
			{
				Path: ":jar", DependsOn: []string{":compileJava"},
				Outputs: []changeaware.OutputEvidence{{Path: "build/libs/groovy-raw-6.0.0-SNAPSHOT-raw.jar", Kind: "FILE", SHA256: digest("1"), Exists: true}},
			},
		},
	}
}

func TestProduceV2AcceptsOrderedEquivalentProducersAndAbsentOutput(t *testing.T) {
	capture := precisionCapture()
	observation, err := Produce(capture)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Status != StatusComplete || observation.CompatibilityIdentitySHA256 == "" ||
		observation.RequestGraphIdentitySHA256 == "" || len(observation.CurrentOutputStates) != 4 {
		t.Fatalf("unexpected v2 observation: %+v", observation)
	}
	if state := findOutputState(t, observation.CurrentOutputStates, "build/right.bin"); !reflect.DeepEqual(state.ProducerTasks, []string{":rightAlias", ":rightProducer"}) || !state.Exists {
		t.Fatalf("equivalent producer group was not retained: %+v", state)
	}
	if state := findOutputState(t, observation.CurrentOutputStates, "build/optional.bin"); state.Exists || state.SHA256 != "" {
		t.Fatalf("absence was not represented explicitly: %+v", state)
	}
}

func TestProduceV2RejectsUnsafeEquivalentProducerEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Capture)
		reason string
	}{
		{name: "unordered peers", mutate: func(value *Capture) {
			value.Tasks[2].DependsOn = nil
			value.Tasks[4].DependsOn = append(value.Tasks[4].DependsOn, ":rightProducer")
		}, reason: "EQUIVALENT_PRODUCERS_UNORDERED"},
		{name: "different hash", mutate: func(value *Capture) { value.Tasks[2].Outputs[0].SHA256 = strings.Repeat("9", 64) }, reason: "EQUIVALENT_PRODUCER_STATE_MISMATCH"},
		{name: "different kind", mutate: func(value *Capture) { value.Tasks[2].Outputs[0].Kind = "DIRECTORY" }, reason: "EQUIVALENT_PRODUCER_KIND_MISMATCH"},
		{name: "outside requested graph", mutate: func(value *Capture) {
			value.Tasks = append(value.Tasks, changeaware.TaskEvidence{Path: ":outside", Outputs: value.Tasks[1].Outputs})
		}, reason: "TASK_OUTSIDE_REQUESTED_GRAPH"},
		{name: "cycle", mutate: func(value *Capture) { value.Tasks[1].DependsOn = []string{":rightAlias"} }, reason: "TASK_GRAPH_CYCLIC"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := precisionCapture()
			test.mutate(&capture)
			observation, err := Produce(capture)
			if err != nil {
				t.Fatal(err)
			}
			if observation.Status != StatusUnavailable || observation.Reason != test.reason {
				t.Fatalf("unexpected rejection: %+v", observation)
			}
		})
	}
}

func TestValidateOutputStatesPreservesAbsence(t *testing.T) {
	capture := precisionCapture()
	observation, err := Produce(capture)
	if err != nil {
		t.Fatal(err)
	}
	expected := []OutputState{findOutputState(t, observation.CurrentOutputStates, "build/optional.bin")}
	if err := ValidateOutputStates(expected, capture); err != nil {
		t.Fatalf("stable absence was rejected: %v", err)
	}
	capture.Tasks[3].Outputs[0].Exists = true
	capture.Tasks[3].Outputs[0].SHA256 = strings.Repeat("8", 64)
	if err := ValidateOutputStates(expected, capture); err == nil {
		t.Fatal("an output appearing after the candidate was accepted")
	}
}

func precisionCapture() Capture {
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
				Outputs: []changeaware.OutputEvidence{{Path: "build/right.bin", Kind: "FILE", SHA256: digest("e"), Exists: true}}},
			{Path: ":rightAlias", DependsOn: []string{":rightProducer"},
				Outputs: []changeaware.OutputEvidence{{Path: "build/right.bin", Kind: "FILE", SHA256: digest("e"), Exists: true}}},
			{Path: ":optionalProducer", Outputs: []changeaware.OutputEvidence{{Path: "build/optional.bin", Kind: "FILE", Exists: false}}},
			{Path: ":bundle", DependsOn: []string{":leftProducer", ":rightAlias", ":optionalProducer"},
				Inputs:  []changeaware.PathEvidence{{Path: "build/left.bin", Kind: "FILE"}, {Path: "build/right.bin", Kind: "FILE"}},
				Outputs: []changeaware.OutputEvidence{{Path: "build/bundle.bin", Kind: "FILE", SHA256: digest("f"), Exists: true}}},
		},
	}
}

func findOutputState(t *testing.T, values []OutputState, path string) OutputState {
	t.Helper()
	for _, value := range values {
		if value.Path == path {
			return value
		}
	}
	t.Fatalf("output state %s was not found: %+v", path, values)
	return OutputState{}
}
