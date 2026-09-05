package configurationinputsource

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"
)

func TestScanDecisions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		source     string
		change     func(*Facts)
		decision   string
		recipe     string
		operations int
	}{
		{name: "provider exec", source: `val value = ProcessBuilder("tool", "read").start().inputReader().readText()`, decision: DecisionPure, recipe: RecipeProviderExec, operations: 1},
		{name: "value source", source: `val value = Runtime.getRuntime().exec("tool read").inputStream.reader().readText()`, change: func(f *Facts) { f.DirectProviderAvailable = false }, decision: DecisionPure, recipe: RecipeValueSource, operations: 1},
		{name: "groovy", source: `def value = ["tool", "read"].execute().text`, decision: DecisionPure, recipe: RecipeProviderExec, operations: 1},
		{name: "side effect syntax", source: `val value = ProcessBuilder("tool", "update-index").start()`, decision: DecisionSideEffect, operations: 1},
		{name: "side effect fact", source: `val value = ProcessBuilder("tool", "read").start()`, change: func(f *Facts) { f.SemanticSideEffect = true }, decision: DecisionSideEffect, operations: 1},
		{name: "secret", source: `val value = ProcessBuilder("tool", "read").start()`, change: func(f *Facts) { f.SecretBearing = true }, decision: DecisionSecretInteractive, operations: 1},
		{name: "interactive", source: `val value = ProcessBuilder("tool", "read").start()`, change: func(f *Facts) { f.Interactive = true }, decision: DecisionSecretInteractive, operations: 1},
		{name: "task action", source: `val value = ProcessBuilder("tool", "read").start()`, change: func(f *Facts) { f.ConfigurationBound = false; f.TaskActionBound = true }, decision: DecisionTaskAction, operations: 1},
		{name: "already tracked", source: `val value = providers.exec { commandLine("tool", "read") }`, decision: DecisionAlreadyTracked, operations: 1},
		{name: "ambiguous", source: `val value = ProcessBuilder("tool", "read").start()`, change: func(f *Facts) { f.BindingAmbiguous = true }, decision: DecisionAmbiguous, operations: 1},
		{name: "external", source: `val value = ProcessBuilder("tool", "read").start()`, change: func(f *Facts) { f.SourceOwned = false }, decision: DecisionAmbiguous, operations: 1},
		{name: "no action", source: `val value = "constant"`, decision: DecisionNoAction, operations: 1},
		{name: "multiple", source: "val a = ProcessBuilder(\"tool\").start()\nval b = Runtime.getRuntime().exec(\"tool\")", decision: DecisionPure, recipe: RecipeProviderExec, operations: 2},
		{name: "redirect is not a process", source: `val output = ProcessBuilder.Redirect.PIPE`, decision: DecisionNoAction, operations: 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			facts := completeFacts(test.source)
			if test.change != nil {
				test.change(&facts)
			}
			rows, err := Scan([]byte(test.source), facts)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != test.operations {
				t.Fatalf("operations = %d, want %d", len(rows), test.operations)
			}
			for _, row := range rows {
				if row.Decision != test.decision || row.Recipe != test.recipe {
					t.Fatalf("row = %#v, want decision=%s recipe=%s", row, test.decision, test.recipe)
				}
			}
		})
	}
}

func TestSourceDriftAndInvalidFacts(t *testing.T) {
	t.Parallel()
	source := `ProcessBuilder("tool").start()`
	facts := completeFacts(source)
	facts.ExpectedSHA256 = digest([]byte(source + " drift"))
	rows, err := Scan([]byte(source), facts)
	if err != nil || len(rows) != 1 || rows[0].Decision != DecisionSourceDrifted {
		t.Fatalf("drift rows=%#v err=%v", rows, err)
	}
	facts = completeFacts(source)
	facts.SourcePath = ""
	if _, err := Scan([]byte(source), facts); err != ErrInvalidFacts {
		t.Fatalf("invalid facts error = %v", err)
	}
}

func TestLabelsCannotChangeClassification(t *testing.T) {
	t.Parallel()
	source := `val value = ProcessBuilder("tool", "read").start().inputReader().readText()`
	first := completeFacts(source)
	second := first
	first.RepositoryLabel, first.TaskLabel, first.VariableLabel = "one", "alpha", "left"
	second.RepositoryLabel, second.TaskLabel, second.VariableLabel = "two", "omega", "right"
	firstRows, err := Scan([]byte(source), first)
	if err != nil {
		t.Fatal(err)
	}
	secondRows, err := Scan([]byte(source), second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstRows, secondRows) {
		t.Fatalf("labels changed classification:\n%#v\n%#v", firstRows, secondRows)
	}
}

func completeFacts(source string) Facts {
	sum := sha256.Sum256([]byte(source))
	lines := 1
	lastColumn := 1
	for _, char := range source {
		if char == '\n' {
			lines++
			lastColumn = 1
		} else {
			lastColumn++
		}
	}
	return Facts{
		SourcePath: "build-logic.kt", ExpectedSHA256: fmt.Sprintf("%x", sum), Language: "KOTLIN",
		DeclarationSpan: Span{StartLine: 1, StartColumn: 1, EndLine: lines, EndColumn: lastColumn},
		CallSites:       []CallSite{{Path: "build.gradle.kts", Line: 1}}, SourceOwned: true,
		ConfigurationBound: true, DirectProviderAvailable: true,
	}
}
