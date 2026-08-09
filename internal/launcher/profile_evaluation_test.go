package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

func TestStructuralProfileEvaluationProposesMeasurementWithoutActivation(t *testing.T) {
	repository := evaluationFixture(t)
	withWorkingDirectory(t, repository)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runStructuralProfileEvaluation([]string{
		"--manifest", "buildopt-impact-manifest.json",
		"--graph", "buildopt-impact-graph.generated.json",
		"--generated-manifest", "buildopt-impact.generated.json",
	}, &stdout, &stderr)
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("evaluation failed with %d: %s", exitCode, stderr.String())
	}
	var report profileEvaluation
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode evaluation: %v", err)
	}
	if report.SchemaVersion != "buildopt.poc/profile-evaluation/v1" ||
		report.Decision != profilediscovery.DecisionMeasure ||
		report.Reason != "COMPLETE_STRUCTURAL_REDUCTION" || report.Profile != nil ||
		!report.ReviewRequired || report.ActivationAutomatic || report.ProductionAuthorized {
		t.Fatalf("unexpected evaluation: %#v", report)
	}
}

func TestStructuralProfileEvaluationRetainsNativeForUnqualifiedEvidence(t *testing.T) {
	repository := evaluationFixture(t)
	if err := os.WriteFile(filepath.Join(repository, "evidence.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	withWorkingDirectory(t, repository)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runStructuralProfileEvaluation([]string{
		"--manifest", "buildopt-impact-manifest.json",
		"--graph", "buildopt-impact-graph.generated.json",
		"--generated-manifest", "buildopt-impact.generated.json",
		"--evidence", "evidence.json",
		"--profile-output", "buildopt-qualified-profile.json",
	}, &stdout, &stderr)
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("evaluation failed with %d: %s", exitCode, stderr.String())
	}
	var report profileEvaluation
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode evaluation: %v", err)
	}
	if report.Decision != "NATIVE_FULL_GRAPH" || report.Reason != "EVIDENCE_NOT_QUALIFIED" || report.Profile != nil {
		t.Fatalf("unqualified evidence did not fail closed: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(repository, "buildopt-qualified-profile.json")); !os.IsNotExist(err) {
		t.Fatalf("unqualified evidence materialized a profile: %v", err)
	}
}

func TestWriteEvaluatedProfileRejectsSymlink(t *testing.T) {
	repository := t.TempDir()
	target := filepath.Join(repository, "target.json")
	if err := os.WriteFile(target, []byte("owned\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(repository, "profile.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := writeEvaluatedProfile(repository, "profile.json", profilediscovery.StructuralProfile{})
	if err == nil {
		t.Fatal("symlink profile output was accepted")
	}
	raw, readErr := os.ReadFile(target)
	if readErr != nil || string(raw) != "owned\n" {
		t.Fatalf("symlink target changed: %q, %v", raw, readErr)
	}
}

func TestWriteEvaluatedProfileRejectsSymlinkDirectory(t *testing.T) {
	repository := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repository, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := writeEvaluatedProfile(repository, filepath.Join("linked", "profile.json"), profilediscovery.StructuralProfile{})
	if err == nil {
		t.Fatal("symlink profile directory was accepted")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "profile.json")); !os.IsNotExist(statErr) {
		t.Fatalf("profile escaped repository: %v", statErr)
	}
}

func TestWriteEvaluatedProfilePublishesExactJSON(t *testing.T) {
	repository := t.TempDir()
	profile := profilediscovery.StructuralProfile{
		SchemaVersion: "buildopt.poc/qualified-profile/v4",
		ProfileID:     "qualified-structural-build-impact",
	}
	if err := writeEvaluatedProfile(repository, "profile.json", profile); err != nil {
		t.Fatal(err)
	}
	want, err := profilediscovery.RenderStructuralProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(repository, "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("profile output differs:\n%s", got)
	}
}

func evaluationFixture(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	source := filepath.Join("..", "..", "fixtures", "poc-kafka-packaging")
	for _, name := range []string{
		"buildopt-impact-manifest.json",
		"buildopt-impact-graph.generated.json",
		"buildopt-impact.generated.json",
	} {
		raw, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repository, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return repository
}

func withWorkingDirectory(t *testing.T, directory string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
