package strictdiagnostic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectRootReport(t *testing.T) {
	root := t.TempDir()
	report := filepath.Join(root, "key", "entry", "configuration-cache-report.html")
	mustWriteReport(t, report)

	selection := SelectRootReport([]byte("See the complete report at "+fileURI(report)+"\n"), root)
	if selection.Outcome != OutcomeCaptured || selection.ReferenceCount != 1 ||
		selection.SelectedRelative != "key/entry/configuration-cache-report.html" ||
		selection.SelectedAbsolute != report {
		t.Fatalf("unexpected selection: %+v", selection)
	}
}

func TestSelectRootReportRejectsInvalidOwnership(t *testing.T) {
	root := t.TempDir()
	report := filepath.Join(root, "key", "entry", "configuration-cache-report.html")
	mustWriteReport(t, report)
	outside := filepath.Join(t.TempDir(), "configuration-cache-report.html")
	mustWriteReport(t, outside)

	tests := []struct {
		name    string
		log     string
		outcome Outcome
	}{
		{name: "missing", outcome: OutcomeReferenceMissing},
		{name: "ambiguous", log: "See the complete report at " + fileURI(report) + "\nSee the complete report at " + fileURI(outside), outcome: OutcomeReferenceAmbiguous},
		{name: "outside", log: "See the complete report at " + fileURI(outside), outcome: OutcomeOutsideRoot},
		{name: "network", log: "See the complete report at file://host/tmp/configuration-cache-report.html", outcome: OutcomeOutsideRoot},
		{name: "wrong-name", log: "See the complete report at " + fileURI(filepath.Dir(report)), outcome: OutcomeOutsideRoot},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selection := SelectRootReport([]byte(test.log), root)
			if selection.Outcome != test.outcome {
				t.Fatalf("outcome = %s, want %s: %+v", selection.Outcome, test.outcome, selection)
			}
		})
	}
}

func TestSelectRootReportRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	realDirectory := t.TempDir()
	report := filepath.Join(realDirectory, "configuration-cache-report.html")
	mustWriteReport(t, report)
	link := filepath.Join(root, "key")
	if err := os.Symlink(realDirectory, link); err != nil {
		t.Fatal(err)
	}
	selection := SelectRootReport([]byte("See the complete report at "+fileURI(filepath.Join(link, filepath.Base(report)))), root)
	if selection.Outcome != OutcomeOutsideRoot {
		t.Fatalf("symlink outcome = %s: %+v", selection.Outcome, selection)
	}
}

func TestAddProjectDirectory(t *testing.T) {
	for _, test := range []struct {
		arguments []string
		want      bool
	}{
		{arguments: []string{"build"}, want: true},
		{arguments: []string{"-p", "sample", "build"}, want: false},
		{arguments: []string{"--project-dir", "sample", "build"}, want: false},
		{arguments: []string{"--project-dir=sample", "build"}, want: false},
	} {
		if got := AddProjectDirectory(test.arguments); got != test.want {
			t.Fatalf("AddProjectDirectory(%q) = %t, want %t", test.arguments, got, test.want)
		}
	}
}

func mustWriteReport(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("report"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fileURI(path string) string {
	return "file://" + filepath.ToSlash(path)
}
