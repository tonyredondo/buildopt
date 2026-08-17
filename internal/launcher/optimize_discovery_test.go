package launcher

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

func TestSplitOptimizeGradleWorkflow(t *testing.T) {
	entrypoints, options, reason := splitOptimizeGradleWorkflow([]string{
		"--no-daemon", "--max-workers", "4", "-Pmode=ci", ":service:distZip",
	})
	if reason != "" || !reflect.DeepEqual(entrypoints, []string{":service:distZip"}) ||
		!reflect.DeepEqual(options, []string{"--no-daemon", "--max-workers", "4", "-Pmode=ci"}) {
		t.Fatalf("workflow = %v, options = %v, reason = %q", entrypoints, options, reason)
	}

	for _, test := range []struct {
		arguments []string
		reason    string
	}{
		{[]string{"--tests", "Example", "test"}, "WORKFLOW_BOUNDARY_UNSUPPORTED"},
		{[]string{"-p", "other", "build"}, "WORKFLOW_BOUNDARY_UNSUPPORTED"},
		{[]string{"--unknown", "build"}, "WORKFLOW_OPTION_UNSUPPORTED"},
		{[]string{"--offline"}, "WORKFLOW_ARGUMENTS_AMBIGUOUS"},
	} {
		if _, _, got := splitOptimizeGradleWorkflow(test.arguments); got != test.reason {
			t.Fatalf("arguments %v reason = %q, want %q", test.arguments, got, test.reason)
		}
	}
}

func TestSplitOptimizeGradleWorkflowAcceptsLargeRealWorkflowsWithinProposalBound(t *testing.T) {
	arguments := make([]string, maximumStructuralAlternativeEntrypoints)
	for index := range arguments {
		arguments[index] = ":module" + strconv.Itoa(index) + ":testClasses"
	}
	entrypoints, options, reason := splitOptimizeGradleWorkflow(arguments)
	if reason != "" || len(entrypoints) != maximumStructuralAlternativeEntrypoints || len(options) != 0 {
		t.Fatalf("bounded workflow = %d entrypoints/%v/%q", len(entrypoints), options, reason)
	}

	arguments = append(arguments, ":one-too-many:testClasses")
	if _, _, reason := splitOptimizeGradleWorkflow(arguments); reason != "WORKFLOW_ARGUMENTS_AMBIGUOUS" {
		t.Fatalf("oversized workflow reason = %q", reason)
	}
}

func TestOptimizeRepositoryIDFromRemote(t *testing.T) {
	for remote, want := range map[string]string{
		"git@github.com:owner/repository.git":      "owner/repository",
		"https://github.com/owner/repository.git":  "owner/repository",
		"ssh://git@example.test/owner/repository":  "owner/repository",
		"https://example.test/one/two/repository/": "two/repository",
	} {
		if got := optimizeRepositoryIDFromRemote(remote); got != want {
			t.Fatalf("remote %q repository = %q, want %q", remote, got, want)
		}
	}
	if got := optimizeRepositoryIDFromRemote("not-a-repository"); got != "" {
		t.Fatalf("invalid remote repository = %q", got)
	}
}

func TestOptimizeAffectedProjectsAndRequiredOutputs(t *testing.T) {
	snapshot := buildimpact.DiscoverySnapshot{Projects: []buildimpact.DiscoveredProject{
		{Path: ":library"},
		{Path: ":service", DependsOn: []string{":library"}},
		{Path: ":unrelated"},
	}}
	affected := optimizeAffectedProjects(snapshot, []string{":library"})
	if !affected[":library"] || !affected[":service"] || affected[":unrelated"] {
		t.Fatalf("affected projects = %v", affected)
	}

	candidates := []outputContractCandidate{
		{Pattern: "service/build/distributions/**", Path: "service/build/distributions/service.zip", FileCount: 1, OwnerProjects: []string{":service"}, ProducerTasks: []string{":service:distZip"}},
		{Pattern: "service/build/publications/maven/pom-default.xml", Path: "service/build/publications/maven/pom-default.xml", FileCount: 1, OwnerProjects: []string{":service"}, ProducerTasks: []string{":service:generatePomFileForMavenPublication"}},
		{Pattern: "library/build/classes/**", Path: "library/build/classes/java/test/Example.class", FileCount: 1, OwnerProjects: []string{":library"}, ProducerTasks: []string{":library:compileTestJava"}},
		{Pattern: "library/build/tmp/compileJava/**", Path: "library/build/tmp/compileJava/previous-compilation-data.bin", FileCount: 1, OwnerProjects: []string{":library"}, ProducerTasks: []string{":library:compileJava"}},
		{Pattern: "unrelated/build/libs/**", Path: "unrelated/build/libs/unrelated.jar", FileCount: 1, OwnerProjects: []string{":unrelated"}, ProducerTasks: []string{":unrelated:jar"}},
	}
	if got := optimizeRequiredOutputPatterns(candidates, []string{"distZip"}, affected); !reflect.DeepEqual(got, []string{"service/build/distributions/**"}) {
		t.Fatalf("direct output patterns = %v", got)
	}
	if got := optimizeRequiredOutputPatterns(candidates, []string{"testClasses"}, affected); !reflect.DeepEqual(got, []string{"library/build/classes/**"}) {
		t.Fatalf("test-preparation output patterns = %v", got)
	}
	if got := optimizeRequiredOutputPatterns(candidates, []string{"distZip", "testClasses"}, affected); !reflect.DeepEqual(got, []string{"library/build/classes/**", "service/build/distributions/**"}) {
		t.Fatalf("multi-entrypoint output patterns = %v", got)
	}
	if got := optimizeRequiredOutputPatterns(candidates, []string{"assemble"}, affected); !reflect.DeepEqual(got, []string{"service/build/distributions/**", "service/build/publications/maven/pom-default.xml"}) {
		t.Fatalf("assemble lifecycle output patterns = %v", got)
	}
}

func TestOptimizeChangeFamilyUsesOnlyGraphAndChangedPaths(t *testing.T) {
	snapshot := buildimpact.DiscoverySnapshot{Projects: []buildimpact.DiscoveredProject{
		{Path: ":core"},
		{Path: ":service", DependsOn: []string{":core"}},
		{Path: ":leaf"},
	}}
	tests := []struct {
		name   string
		paths  []string
		owners []string
		want   string
	}{
		{"dependency source", []string{"core/src/main/java/Api.java"}, []string{":core"}, optimizeFamilyDependency},
		{"resource", []string{"service/src/main/resources/application.yaml"}, []string{":service"}, optimizeFamilyResource},
		{"leaf source", []string{"leaf/src/main/java/Leaf.java"}, []string{":leaf"}, optimizeFamilyLeaf},
		{"mixed source", []string{"core/src/main/java/Api.java", "leaf/src/main/java/Leaf.java"}, []string{":core", ":leaf"}, optimizeFamilyMixed},
		{"source and resource", []string{"leaf/src/main/java/Leaf.java", "leaf/src/main/resources/leaf.txt"}, []string{":leaf"}, optimizeFamilyMixed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := optimizeChangeFamily(snapshot, test.paths, test.owners); got != test.want {
				t.Fatalf("change family = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInspectOptimizeDiscoveryContextUsesExactLocalUpstream(t *testing.T) {
	repository := t.TempDir()
	runOptimizeGit(t, repository, "init", "-b", "main")
	runOptimizeGit(t, repository, "config", "user.name", "BuildOpt Test")
	runOptimizeGit(t, repository, "config", "user.email", "buildopt@example.invalid")
	path := filepath.Join(repository, "source.txt")
	if err := os.WriteFile(path, []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runOptimizeGit(t, repository, "add", "source.txt")
	runOptimizeGit(t, repository, "commit", "-m", "base")
	base := strings.TrimSpace(runOptimizeGit(t, repository, "rev-parse", "HEAD"))
	runOptimizeGit(t, repository, "switch", "-c", "feature")
	runOptimizeGit(t, repository, "config", "branch.feature.remote", ".")
	runOptimizeGit(t, repository, "config", "branch.feature.merge", "refs/heads/main")
	if err := os.WriteFile(path, []byte("target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runOptimizeGit(t, repository, "add", "source.txt")
	runOptimizeGit(t, repository, "commit", "-m", "target")
	target := strings.TrimSpace(runOptimizeGit(t, repository, "rev-parse", "HEAD"))

	context := inspectOptimizeDiscoveryContext(repository, []string{"jar", "--offline"}, func(string) string { return "" })
	if !context.Ready || context.Source != "LOCAL_UPSTREAM" || context.BaseRevision != base ||
		context.TargetRevision != target || !reflect.DeepEqual(context.changedPaths, []string{"source.txt"}) ||
		context.RepositoryID == "" || len(context.ChangeSHA256) != 64 {
		t.Fatalf("discovery context = %+v", context)
	}

	if err := os.WriteFile(path, []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirty := inspectOptimizeDiscoveryContext(repository, []string{"jar"}, func(string) string { return "" })
	if dirty.Ready || dirty.Reason != "WORKTREE_DIRTY" {
		t.Fatalf("dirty discovery context = %+v", dirty)
	}
	if err := os.WriteFile(path, []byte("target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "new-source.txt"), []byte("untracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	untracked := inspectOptimizeDiscoveryContext(repository, []string{"jar"}, func(string) string { return "" })
	if untracked.Ready || untracked.Reason != "WORKTREE_DIRTY" {
		t.Fatalf("untracked discovery context = %+v", untracked)
	}
}

func TestOptimizeGitHubBaseRejectsTrailingJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event.json")
	valid := `{"pull_request":{"base":{"sha":"0123456789abcdef0123456789abcdef01234567"}}}`
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	base, err := optimizeGitHubBase(path)
	if err != nil || base != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("GitHub base = %q, err = %v", base, err)
	}
	if err := os.WriteFile(path, []byte(valid+"{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := optimizeGitHubBase(path); err == nil {
		t.Fatal("trailing GitHub event JSON was accepted")
	}
}

func TestOptimizeGeneratedStatePath(t *testing.T) {
	for _, path := range []string{".buildopt/state.json", ".gradle/9.6.1/cache.bin"} {
		if !optimizeGeneratedStatePath(path) {
			t.Fatalf("generated state path %q was rejected", path)
		}
	}
	for _, path := range []string{"src/New.java", "gradle/init.gradle", "build.gradle"} {
		if optimizeGeneratedStatePath(path) {
			t.Fatalf("customer path %q was accepted as generated state", path)
		}
	}
}

func TestOptimizeDiscoveryErrorReasonIdentifiesFailedStage(t *testing.T) {
	tests := map[string]string{
		"output preflight: command failed": "OUTPUT_PREFLIGHT_FAILED",
		"graph preflight: command failed":  "GRAPH_PREFLIGHT_FAILED",
		"proposal: command failed":         "PROPOSAL_PREFLIGHT_FAILED",
		"unexpected failure":               "DISCOVERY_EXECUTION_FAILED",
	}
	for message, expected := range tests {
		if actual := optimizeDiscoveryErrorReason(errors.New(message)); actual != expected {
			t.Fatalf("optimizeDiscoveryErrorReason(%q) = %q, want %q", message, actual, expected)
		}
	}
	if actual := optimizeDiscoveryErrorReason(context.DeadlineExceeded); actual != "DISCOVERY_BUDGET_EXHAUSTED" {
		t.Fatalf("deadline reason = %q, want DISCOVERY_BUDGET_EXHAUSTED", actual)
	}
}

func runOptimizeGit(t *testing.T, repository string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = repository
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return string(output)
}
