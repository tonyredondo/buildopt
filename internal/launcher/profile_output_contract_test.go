package launcher

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateDeclaredOutputsAcceptsMultipleUnambiguousProjects(t *testing.T) {
	owners := []outputRootOwnership{
		{path: "service-a/target/libs", kind: "DIRECTORY", files: []string{"service-a/target/libs/a.jar"}, projects: map[string]bool{":service-a": true}, tasks: map[string]bool{":service-a:jar": true}},
		{path: "service-b/target/libs", kind: "DIRECTORY", files: []string{"service-b/target/libs/b.jar"}, projects: map[string]bool{":service-b": true}, tasks: map[string]bool{":service-b:jar": true}},
	}
	validations := validateDeclaredOutputs([]string{"**/target/libs/**"}, owners)
	if len(validations) != 1 || validations[0].Status != "VALIDATED" || validations[0].MatchedFiles != 2 || !reflect.DeepEqual(validations[0].OwnerProjects, []string{":service-a", ":service-b"}) {
		t.Fatalf("validations = %#v", validations)
	}
}

func TestValidateDeclaredOutputsRejectsEmptyAndAmbiguousOwnership(t *testing.T) {
	owners := []outputRootOwnership{{
		path: "shared", kind: "DIRECTORY", files: []string{"shared/result.txt"},
		projects: map[string]bool{":service-a": true, ":service-b": true},
		tasks:    map[string]bool{":service-a:sharedOutput": true, ":service-b:sharedOutput": true},
	}}
	validations := validateDeclaredOutputs([]string{"build/libs/**", "shared/**"}, owners)
	if validations[0].Status != "EMPTY" || validations[1].Status != "AMBIGUOUS" {
		t.Fatalf("validations = %#v", validations)
	}
}

func TestCollectOutputOwnershipIgnoresEmptyOutsideAndSymlinkOutputs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "service", "target", "libs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "service", "target", "libs", "service.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := outputContractSnapshot{Tasks: []outputContractTask{{
		TaskPath: ":service:jar", ProjectPath: ":service",
		Outputs: []outputContractRoot{
			{Path: "service/target/libs", Kind: "DIRECTORY", Exists: true, InsideRepository: true, FileCount: 1},
			{Path: "service/target/empty", Kind: "DIRECTORY", Exists: true, InsideRepository: true},
			{Kind: "DIRECTORY", Exists: true, InsideRepository: false, FileCount: 1},
			{Path: "service/target/link", Kind: "FILE", Exists: true, InsideRepository: true, Symlink: true, FileCount: 1},
		},
	}}}
	owners, candidates, err := collectOutputOwnership(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 1 || len(candidates) != 1 || candidates[0].Pattern != "service/target/libs/**" || candidates[0].FileCount != 1 {
		t.Fatalf("owners = %#v, candidates = %#v", owners, candidates)
	}
}

func TestValidOutputContractPatternRejectsEscapes(t *testing.T) {
	for _, candidate := range []string{"../build/**", "/tmp/build/**", "build\\libs\\**", "build/[/**", "build//libs/**"} {
		if validOutputContractPattern(candidate) {
			t.Fatalf("expected %q to be rejected", candidate)
		}
	}
	if !validOutputContractPattern("hibernate-core/target/libs/**") {
		t.Fatal("expected Hibernate output pattern to be valid")
	}
}

func TestOutputContractGradleArgumentsOwnDaemonAndConsoleWithoutDroppingOwnerProperties(t *testing.T) {
	got := outputContractGradleArguments([]string{
		"--daemon", "--build-cache", "--configure-on-demand", "--console=plain", "-Ptarget.posix=false",
	}, "/private/output-contract.init.gradle")
	want := []string{
		"--build-cache", "-Ptarget.posix=false", "--no-daemon", "--no-configure-on-demand", "--console=plain",
		"--init-script", "/private/output-contract.init.gradle", "buildoptOutputContract",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output-contract Gradle arguments = %#v, want %#v", got, want)
	}
}

func TestPrepareOutputContractReportReusesBoundSnapshot(t *testing.T) {
	root := t.TempDir()
	runOptimizeGit(t, root, "init", "--quiet")
	runOptimizeGit(t, root, "config", "user.name", "BuildOpt Test")
	runOptimizeGit(t, root, "config", "user.email", "buildopt@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "settings.gradle"), []byte("rootProject.name = 'fixture'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runOptimizeGit(t, root, "add", "settings.gradle")
	runOptimizeGit(t, root, "commit", "--quiet", "-m", "fixture")
	revision := strings.TrimSpace(runOptimizeGit(t, root, "rev-parse", "HEAD"))
	if err := os.MkdirAll(filepath.Join(root, "build", "libs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "libs", "fixture.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := outputContractSnapshot{
		SchemaVersion: outputSnapshotSchema, GradleVersion: "9.6.1", Entrypoints: []string{"jar"},
		Tasks: []outputContractTask{{
			TaskPath: ":jar", ProjectPath: ":", ProjectDirectory: "",
			Outputs: []outputContractRoot{{Path: "build/libs", Kind: "DIRECTORY", Exists: true, InsideRepository: true, FileCount: 1}},
		}},
	}
	report, err := prepareOutputContractReportFromSnapshot(root, outputContractConfig{
		repositoryID: "example/fixture", pipelineClass: "jar-change", repositoryRevision: revision,
		entrypoints: []string{"jar"}, requiredOutputs: []string{"build/libs/**"},
	}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != "VALIDATED_REQUIRED_OUTPUTS" || len(report.Validations) != 1 || report.Validations[0].MatchedFiles != 1 || report.snapshot.GradleVersion != "9.6.1" {
		t.Fatalf("reused output report = %#v", report)
	}
	snapshot.Entrypoints = []string{"assemble"}
	if _, err := prepareOutputContractReportFromSnapshot(root, outputContractConfig{
		repositoryID: "example/fixture", pipelineClass: "jar-change", repositoryRevision: revision,
		entrypoints: []string{"jar"}, requiredOutputs: []string{"build/libs/**"},
	}, snapshot); err == nil {
		t.Fatal("snapshot from a different owner workflow was reused")
	}
}
