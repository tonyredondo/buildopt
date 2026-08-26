package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestPairedBootstrapLowerBoundIsDeterministic(t *testing.T) {
	values := []int64{-4, -2, 1, 3, 7}
	first := pairedBootstrapLowerBound(values, 10000, 50, 20260826)
	second := pairedBootstrapLowerBound(values, 10000, 50, 20260826)
	if first != second {
		t.Fatalf("bootstrap lower bound changed: %d != %d", first, second)
	}
}

func TestCurrentEvidenceStopsAdaptiveFragmentPOC(t *testing.T) {
	rawBytes := mustReadDecisionFixture(t, "../../benchmarks/results/current-longitudinal-raw-v1.json")
	raw, err := decodeRaw(rawBytes)
	if err != nil {
		t.Fatal(err)
	}
	reportBytes := mustReadDecisionFixture(t, "../../benchmarks/results/current-longitudinal-report-v1.json")
	var reportValue report
	if err := decodeStrict(reportBytes, &reportValue); err != nil {
		t.Fatal(err)
	}
	attributionBytes := mustReadDecisionFixture(t, "../../benchmarks/results/current-longitudinal-attribution-v1.json")
	var attribution attributionDocument
	if err := decodeStrict(attributionBytes, &attribution); err != nil {
		t.Fatal(err)
	}
	campaignBytes := mustReadDecisionFixture(t, "../../specs/poc-current-longitudinal-campaign-v1.json")
	var campaign campaignContract
	if err := json.Unmarshal(campaignBytes, &campaign); err != nil {
		t.Fatal(err)
	}
	contractBytes := mustReadDecisionFixture(t, "../../specs/poc-adaptive-fragment-terminal-decision-v1.json")
	var contract terminalDecisionContract
	if err := decodeStrict(contractBytes, &contract); err != nil {
		t.Fatal(err)
	}
	if err := validateTerminalContract(contract); err != nil {
		t.Fatal(err)
	}
	document, err := deriveTerminalDecision(rawBytes, raw, reportBytes, reportValue, attributionBytes, attribution, campaignBytes, campaign, contractBytes, contract)
	if err != nil {
		t.Fatal(err)
	}
	if document.Outcome != "STOP_ADAPTIVE_FRAGMENT_POC" {
		t.Fatalf("outcome = %s", document.Outcome)
	}
	if len(document.Criteria) != 15 || document.Summary.CriteriaPassed+document.Summary.CriteriaFailed != 15 {
		t.Fatalf("terminal scorecard is incomplete: %+v", document.Summary)
	}
	if document.Summary.CriteriaPassed != 9 || document.Summary.CriteriaFailed != 6 ||
		document.Summary.EligibleDescendantBuilds != 71 || document.Summary.ActivatedEligibleBuilds != 0 ||
		document.Summary.PositiveFamilies != 0 || document.Summary.PositiveLowerBoundFamilies != 0 {
		t.Fatalf("terminal scorecard changed: %+v", document.Summary)
	}
	if document.Specialization.BoundedValueExists || document.Specialization.Activations != 0 || document.Specialization.AttributableSaveNS != 0 {
		t.Fatalf("unsupported specialization: %+v", document.Specialization)
	}
	if document.HistoricalContext.Input {
		t.Fatal("historical AF-013 evidence became a current decision input")
	}
}

func TestTerminalContractRejectsMovedThreshold(t *testing.T) {
	contractBytes := mustReadDecisionFixture(t, "../../specs/poc-adaptive-fragment-terminal-decision-v1.json")
	var contract terminalDecisionContract
	if err := decodeStrict(contractBytes, &contract); err != nil {
		t.Fatal(err)
	}
	contract.Thresholds.MinimumPositiveFamilies = 2
	if err := validateTerminalContract(contract); err == nil {
		t.Fatal("moved terminal threshold unexpectedly passed")
	}
}

func mustReadDecisionFixture(t *testing.T, path string) []byte {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
