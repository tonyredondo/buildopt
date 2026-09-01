package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/historyadmission"
)

func TestRunAuditAdmitsFiveExactGraphMatches(t *testing.T) {
	repository := t.TempDir()
	git(t, repository, "init", "-q")
	git(t, repository, "config", "user.email", "buildopt@example.invalid")
	git(t, repository, "config", "user.name", "BuildOpt fixture")
	git(t, repository, "config", "commit.gpgsign", "false")
	writeFile(t, repository, "settings.gradle.kts", "rootProject.name = \"fixture\"\n")
	git(t, repository, "add", ".")
	git(t, repository, "commit", "-qm", "initial")
	for revision := 1; revision <= 5; revision++ {
		writeFile(t, repository, "library/src/main/java/Example.java", fmt.Sprintf("class Example { int revision = %d; }\n", revision))
		git(t, repository, "add", "library")
		git(t, repository, "commit", "-qm", fmt.Sprintf("source %d", revision))
	}
	snapshotPath, snapshotSHA := writeSnapshot(t, repository)
	target := strings.TrimSpace(git(t, repository, "rev-parse", "HEAD"))
	report, err := runAudit(auditOptions{repository: repository, snapshotPath: snapshotPath, snapshotSHA: snapshotSHA,
		target: target, family: historyadmission.FamilyDependency, entrypoints: []string{"testClasses"},
		owners: []string{":library"}, maximum: 64, minimum: 5})
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != "ADMIT" || report.CompatibleCommits != 5 || report.HistoryWindow != 6 || report.GradleExecuted || report.RepositoryNameRule {
		t.Fatalf("report = %+v", report)
	}
	if report.Rows[len(report.Rows)-1].Reason != "UNSAFE_STRUCTURAL_CHANGE" {
		t.Fatalf("root row = %+v", report.Rows[len(report.Rows)-1])
	}
}

func TestRunAuditRejectsSnapshotDrift(t *testing.T) {
	repository := t.TempDir()
	snapshotPath, _ := writeSnapshot(t, repository)
	_, err := runAudit(auditOptions{repository: repository, snapshotPath: snapshotPath, snapshotSHA: strings.Repeat("0", 64),
		target: strings.Repeat("a", 40), family: historyadmission.FamilyDependency, entrypoints: []string{"testClasses"},
		owners: []string{":library"}, maximum: 64, minimum: 5})
	if err == nil || !strings.Contains(err.Error(), "binding does not match") {
		t.Fatalf("error = %v", err)
	}
}

func writeSnapshot(t *testing.T, directory string) (string, string) {
	t.Helper()
	snapshot := buildimpact.DiscoverySnapshot{SchemaVersion: buildimpact.DiscoverySchemaVersion, GradleVersion: "9.0", Complete: true,
		Projects: []buildimpact.DiscoveredProject{
			{Path: ":library", SourcePaths: []string{"library/**"}, DependsOn: []string{}},
			{Path: ":consumer", SourcePaths: []string{"consumer/**"}, DependsOn: []string{":library"}},
		}, Entrypoints: []buildimpact.DiscoveredEntrypoint{{Name: "testClasses", ReachesProjects: []string{":consumer", ":library"}}}}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "snapshot.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	return path, hex.EncodeToString(digest[:])
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", arguments[0], err, output)
	}
	return string(output)
}
