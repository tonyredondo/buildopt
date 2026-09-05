package configurationinput

import (
	"os"
	"reflect"
	"testing"
)

func TestFixtureDecisions(t *testing.T) {
	report := readFixture(t, "testdata/report.html")
	factsContent := readFixture(t, "testdata/facts.json")
	problems, err := ParseReport(report)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := DecodeFacts(factsContent)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		DecisionSimpleProviderExec, DecisionValueSourceReview, DecisionTypedProvider,
		DecisionTypedProvider, DecisionExternalOwner, DecisionUnsafe, DecisionAmbiguous,
		DecisionAlreadySupported, DecisionSourceDrifted, DecisionNoAction, DecisionUnsafe,
	}
	var got []string
	for index := range problems {
		row, classifyErr := Classify(problems[index], facts[index])
		if classifyErr != nil {
			t.Fatalf("row %d: %v", index, classifyErr)
		}
		got = append(got, row.Decision)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decisions = %v, want %v", got, want)
	}
}

func TestLabelsCannotChangeDecision(t *testing.T) {
	problems, err := ParseReport(readFixture(t, "testdata/report.html"))
	if err != nil {
		t.Fatal(err)
	}
	facts, err := DecodeFacts(readFixture(t, "testdata/facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := Classify(problems[0], facts[0])
	if err != nil {
		t.Fatal(err)
	}
	renamed := facts[0]
	renamed.RepositoryLabel = "unrelated-repository"
	renamed.TaskLabel = "unrelated-task"
	renamed.ExecutableLabel = "unrelated-executable"
	second, err := Classify(problems[0], renamed)
	if err != nil {
		t.Fatal(err)
	}
	if first.Decision != second.Decision || first.RecipeClass != second.RecipeClass {
		t.Fatalf("labels changed classification: %+v != %+v", first, second)
	}
}

func TestReportAndFactsFailClosed(t *testing.T) {
	for _, report := range [][]byte{
		[]byte("no embedded report"),
		[]byte(reportMarker + `{}`),
		[]byte(reportMarker + `{"diagnostics":[],"totalProblemCount":1}`),
		[]byte(reportMarker + `{"diagnostics":[{"trace":[],"problem":[]}],"totalProblemCount":1}`),
	} {
		if _, err := ParseReport(report); err == nil {
			t.Fatalf("invalid report accepted: %q", report)
		}
	}
	if _, err := DecodeFacts([]byte(`[{"unknown":true}]`)); err == nil {
		t.Fatal("unknown source fact accepted")
	}
	if _, err := DecodeFacts([]byte(`null`)); err == nil {
		t.Fatal("null source facts accepted")
	}
}

func TestZeroProblemReportIsConclusive(t *testing.T) {
	report := []byte(reportMarker + "\n" + reportDataMarker + "\n" + `{"diagnostics":[{"trace":[{"kind":"BuildLogic","location":"settings file"}],"input":[{"text":"Gradle property "},{"name":"version"}]}],"totalProblemCount":0});}`)
	problems, err := ParseReport(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %d, want 0", len(problems))
	}
	facts, err := DecodeFacts([]byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 0 {
		t.Fatalf("facts = %d, want 0", len(facts))
	}
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
