package configurationinputbinding

import (
	"testing"

	"github.com/tonyredondo/buildopt/internal/configurationinput"
	"github.com/tonyredondo/buildopt/internal/configurationinputsource"
)

func TestBindExactSourceCommand(t *testing.T) {
	source := []byte(`ProcessBuilder("git", "rev-parse", "HEAD")`)
	rows := []configurationinputsource.Row{pureRow(1, 1, 43)}
	problems := []configurationinput.Problem{{Index: 2, Kind: configurationinput.ProblemExternalProcess, Message: "external process started git rev-parse HEAD"}}
	result := Bind(source, rows, problems)
	if result.Decision != DecisionBound || len(result.Bindings) != 1 || result.Bindings[0].Command != "git rev-parse HEAD" {
		t.Fatalf("unexpected binding: %+v", result)
	}
}

func TestBindRejectsMissingAndDifferentDiagnostics(t *testing.T) {
	source := []byte(`Runtime.getRuntime().exec("tool read")`)
	rows := []configurationinputsource.Row{pureRow(1, 1, 38)}
	for _, problems := range [][]configurationinput.Problem{
		nil,
		{{Index: 0, Kind: configurationinput.ProblemOther, Message: "unrelated"}},
		{{Index: 0, Kind: configurationinput.ProblemExternalProcess, Message: "external process started 'tool write'"}},
	} {
		if result := Bind(source, rows, problems); result.Decision != DecisionUnbound || len(result.Bindings) != 0 {
			t.Fatalf("unexpected unbound result: %+v", result)
		}
	}
}

func TestBindRejectsAmbiguousSourceOperations(t *testing.T) {
	source := []byte("ProcessBuilder(\"tool\", \"read\")\nProcessBuilder(\"tool\", \"read\")")
	rows := []configurationinputsource.Row{pureRow(1, 1, 30), pureRow(2, 1, 30)}
	problems := []configurationinput.Problem{{Index: 0, Kind: configurationinput.ProblemExternalProcess, Message: "external process started 'tool read'"}}
	if result := Bind(source, rows, problems); result.Decision != DecisionAmbiguous || len(result.Bindings) != 2 {
		t.Fatalf("unexpected ambiguous result: %+v", result)
	}
}

func pureRow(line, startColumn, endColumn int) configurationinputsource.Row {
	return configurationinputsource.Row{
		Decision:      configurationinputsource.DecisionPure,
		OperationSpan: configurationinputsource.Span{StartLine: line, StartColumn: startColumn, EndLine: line, EndColumn: endColumn},
	}
}
