package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommittedAttributionMatchesCurrentCampaign(t *testing.T) {
	root := filepath.Join("..", "..")
	rawPath := filepath.Join(root, "benchmarks", "results", "current-longitudinal-raw-v1.json")
	reportPath := filepath.Join(root, "benchmarks", "results", "current-longitudinal-report-v1.json")
	contractPath := filepath.Join(root, "specs", "poc-current-longitudinal-attribution-v1.json")

	rawBytes, raw, reportBytes, reportValue, contractBytes, contract, err := readAttributionInputs(rawPath, reportPath, contractPath)
	if err != nil {
		t.Fatal(err)
	}
	document, err := deriveAttribution(rawBytes, raw, reportBytes, reportValue, contractBytes, contract)
	if err != nil {
		t.Fatal(err)
	}
	if document.Outcome != "CURRENT_VALUE_NOT_ATTRIBUTABLE" ||
		document.Summary.ComparablePairs != 100 ||
		document.Summary.ExactOutputPairs != 100 ||
		document.Summary.SelectedProfiles != 0 ||
		document.Summary.FragmentActivations != 0 ||
		document.Summary.NativeRetentions != 100 ||
		document.Summary.CumulativeSignedDeltaNS != -368622514697 ||
		document.Summary.RecordedBuildOptCostNS != 179029032102 ||
		document.Summary.ResidualGradleRunnerNS != -189593482595 ||
		document.Summary.AttributableMechanismSavingsNS != 0 {
		t.Fatalf("unexpected current attribution summary: %+v", document.Summary)
	}
	if document.Summary.Diagnostics.DiscoveryLearningNS != 98384963343 ||
		document.Summary.Diagnostics.OutputVerificationNS != 28889844657 ||
		document.Summary.Diagnostics.LocalStateNS != 18151850034 {
		t.Fatalf("unexpected current attribution diagnostics: %+v", document.Summary.Diagnostics)
	}

	committed, err := os.ReadFile(filepath.Join(root, "benchmarks", "results", "current-longitudinal-attribution-v1.json"))
	if err == nil && len(committed) == 0 {
		t.Fatal("committed attribution is empty")
	}
}

func TestWorkflowClassificationUsesOnlyGenericTaskRules(t *testing.T) {
	rules := []workflowRule{
		{ID: "TEST", Match: "CONTAINS", Pattern: "compileTest"},
		{ID: "ASSEMBLY", Match: "SUFFIX", Pattern: "assemble"},
	}
	for _, test := range []struct {
		workflow []string
		want     string
	}{
		{workflow: []string{":module:compileTestJava"}, want: "TEST"},
		{workflow: []string{":module:assemble"}, want: "ASSEMBLY"},
	} {
		got, err := classifyWorkflow(test.workflow, rules)
		if err != nil || got != test.want {
			t.Fatalf("classifyWorkflow(%q) = %q, %v; want %q", test.workflow, got, err, test.want)
		}
	}
	if _, err := classifyWorkflow([]string{"check"}, rules); err == nil {
		t.Fatal("expected unmatched workflow to fail closed")
	}
}
