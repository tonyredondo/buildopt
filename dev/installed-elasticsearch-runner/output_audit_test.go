package main

import "testing"

func TestOutputAuditAccountsForAllDifferencesWithoutAdmittingThem(t *testing.T) {
	a, _ := compareFixture(t, firstManifest, "EXECUTED")
	b, _ := compareFixture(t, firstManifest, "EXECUTED")
	report, err := auditNativeOutputs(a, b)
	if err != nil || !report.GatePassed || report.ValueAdmitted || report.EqualEntries != report.Entries {
		t.Fatalf("equal audit: %+v %v", report, err)
	}
	c, _ := compareFixture(t, secondManifest(), "EXECUTED")
	report, err = auditNativeOutputs(a, c)
	if err != nil || report.GatePassed || report.ValueAdmitted || len(report.Unapproved) != 1 || len(report.AcceptedDates) != 0 || report.EqualEntries+len(report.Unapproved) != report.Entries {
		t.Fatalf("unapproved fixture output was hidden: %+v %v", report, err)
	}
	if len(report.Unapproved[0].Owners) != 1 || report.Unapproved[0].Owners[0] != ":server:jar" {
		t.Fatal("owner provenance missing", report.Unapproved)
	}
}
