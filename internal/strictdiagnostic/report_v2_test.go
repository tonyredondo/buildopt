package strictdiagnostic

import (
	"path/filepath"
	"testing"
)

func TestSelectRootReportV2DeduplicatesIdenticalLogReferences(t *testing.T) {
	root := t.TempDir()
	report := filepath.Join(root, "key", "entry", "configuration-cache-report.html")
	mustWriteReport(t, report)
	reference := "See the complete report at " + fileURI(report) + "\n"

	selection := SelectRootReportV2([]byte(reference+reference), root)
	if selection.Outcome != OutcomeCaptured || selection.ReferenceCount != 1 ||
		selection.SelectedRelative != "key/entry/configuration-cache-report.html" {
		t.Fatalf("unexpected duplicate selection: %+v", selection)
	}
}

func TestSelectRootReportV2KeepsDistinctReferencesAmbiguous(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "one", "entry", "configuration-cache-report.html")
	second := filepath.Join(root, "two", "entry", "configuration-cache-report.html")
	mustWriteReport(t, first)
	mustWriteReport(t, second)
	log := "See the complete report at " + fileURI(first) + "\nSee the complete report at " + fileURI(second)

	selection := SelectRootReportV2([]byte(log), root)
	if selection.Outcome != OutcomeReferenceAmbiguous || selection.ReferenceCount != 2 {
		t.Fatalf("unexpected distinct selection: %+v", selection)
	}
}
