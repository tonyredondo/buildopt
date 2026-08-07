package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestImpactHelpAndInvalidArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"impact", "--help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != impactUsage || stderr.Len() != 0 {
		t.Fatalf("help stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"impact"}, strings.NewReader(""), &stdout, &stderr); code != exitConfiguration {
		t.Fatalf("invalid invocation exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), impactUsage) {
		t.Fatalf("invalid invocation stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestImpactInvocationSelectsCandidateAndFallsBackToFullGraph(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)

	candidate, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--pipeline-class", "pull-request",
		"--changes-file", "changed.txt",
		"--gradle-option=--no-daemon",
		"--cache-standard-jar-producers",
		"--cache-standard-copy-tasks",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !candidate.plan.CandidateSelected || candidate.plan.ProductionAuthorized ||
		!candidate.standardJarCache || !candidate.standardCopyCache ||
		strings.Join(candidate.gradleArgs, " ") != "--no-daemon :service-a:assemble" {
		t.Fatalf("candidate invocation = %+v", candidate)
	}

	if err := os.WriteFile(filepath.Join(repositoryRoot, "changed.txt"), []byte("unowned/file.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fallback, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if fallback.plan.CandidateSelected || fallback.plan.ProductionAuthorized || strings.Join(fallback.gradleArgs, " ") != "assemble" || fallback.plan.Reason != "IMPACT_UNKNOWN_CHANGE_PATH" {
		t.Fatalf("fallback invocation = %+v", fallback)
	}
}

func TestImpactRunActivatesExplicitStandardJarCache(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	clearGradleManagedL1Inputs(t)
	initScript := filepath.Join(repositoryRoot, "buildopt.init.gradle")
	pluginJar := filepath.Join(repositoryRoot, "buildopt-gradle-plugin.jar")
	writeGradleTestFile(t, initScript)
	writeGradleTestFile(t, pluginJar)
	t.Setenv(gradleInitScriptEnvironment, initScript)
	t.Setenv(gradlePluginJarEnvironment, pluginJar)
	writeGradleWrapperProperties(t, repositoryRoot, "distributionUrl=gradle-9.6.1-bin.zip\n")

	wrapper := filepath.Join(repositoryRoot, gradleWrapperName(runtime.GOOS))
	observation := filepath.Join(repositoryRoot, "jar-cache.env")
	contents := "#!/bin/sh\nprintf '%s\\n' \"$BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS\" > jar-cache.env\n"
	if runtime.GOOS == "windows" {
		contents = "@echo off\r\nset BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS>jar-cache.env\r\n"
	}
	if err := os.WriteFile(wrapper, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"impact",
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
		"--cache-standard-jar-producers",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("impact exit code = %d, stderr = %q", code, stderr.String())
	}
	raw, err := os.ReadFile(observation)
	if err != nil {
		t.Fatal(err)
	}
	want := "1\n"
	if runtime.GOOS == "windows" {
		want = gradleStandardJarCacheEnvironment + "=1\n"
	}
	if strings.ReplaceAll(string(raw), "\r\n", "\n") != want {
		t.Fatalf("standard Jar cache child environment = %q", raw)
	}
}

func TestImpactRunActivatesExplicitStandardCopyCache(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	clearGradleManagedL1Inputs(t)
	initScript := filepath.Join(repositoryRoot, "buildopt.init.gradle")
	pluginJar := filepath.Join(repositoryRoot, "buildopt-gradle-plugin.jar")
	writeGradleTestFile(t, initScript)
	writeGradleTestFile(t, pluginJar)
	t.Setenv(gradleInitScriptEnvironment, initScript)
	t.Setenv(gradlePluginJarEnvironment, pluginJar)
	writeGradleWrapperProperties(t, repositoryRoot, "distributionUrl=gradle-9.6.1-bin.zip\n")

	wrapper := filepath.Join(repositoryRoot, gradleWrapperName(runtime.GOOS))
	observation := filepath.Join(repositoryRoot, "copy-cache.env")
	contents := "#!/bin/sh\nprintf '%s\\n' \"$BUILDOPT_CACHE_STANDARD_COPY_TASKS\" > copy-cache.env\n"
	if runtime.GOOS == "windows" {
		contents = "@echo off\r\nset BUILDOPT_CACHE_STANDARD_COPY_TASKS>copy-cache.env\r\n"
	}
	if err := os.WriteFile(wrapper, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"impact",
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
		"--cache-standard-copy-tasks",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("impact exit code = %d, stderr = %q", code, stderr.String())
	}
	raw, err := os.ReadFile(observation)
	if err != nil {
		t.Fatal(err)
	}
	want := "1\n"
	if runtime.GOOS == "windows" {
		want = gradleStandardCopyCacheEnvironment + "=1\n"
	}
	if strings.ReplaceAll(string(raw), "\r\n", "\n") != want {
		t.Fatalf("standard Copy cache child environment = %q", raw)
	}
}

func TestImpactInvocationBypassRestoresOriginalEntrypoints(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	invocation, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.plan.CandidateSelected || invocation.plan.Reason != "LOCAL_BYPASS" || strings.Join(invocation.gradleArgs, " ") != "assemble" {
		t.Fatalf("bypass invocation = %+v", invocation)
	}
}

func TestImpactInvocationRejectsUnsafeChangesFiles(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	if err := os.WriteFile(filepath.Join(repositoryRoot, "changed.txt"), []byte("a\na\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
	}, false)
	if err == nil {
		t.Fatal("duplicate changed paths were accepted")
	}
}

func TestImpactInvocationRejectsChangesFileSymlinkOutsideRepository(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ordinary Windows users may not create symlinks")
	}
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	outside := filepath.Join(t.TempDir(), "changes.txt")
	if err := os.WriteFile(outside, []byte("library-c/src/main/java/synthetic/LibraryC.java\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repositoryRoot, "outside-changes.txt")); err != nil {
		t.Fatal(err)
	}
	_, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "outside-changes.txt",
	}, false)
	if err == nil {
		t.Fatal("changes file symlink outside the repository was accepted")
	}
}

func TestImpactInvocationRejectsGraphChangingGradleOptions(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	for _, option := range []string{"assemble", ":service-b:assemble", "--exclude-task=:service-a:assemble", "-p"} {
		_, err := prepareImpactInvocation([]string{
			"--repository-id", "tonyredondo/buildopt-impact-synthetic",
			"--changes-file", "changed.txt",
			"--gradle-option=" + option,
		}, false)
		if err == nil {
			t.Fatalf("unsafe Gradle option %q was accepted", option)
		}
	}
}

func TestImpactInvocationAcceptsBoundedSpringExecutionOptions(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	invocation, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
		"--gradle-option=--daemon",
		"--gradle-option=--parallel",
		"--gradle-option=--no-scan",
		"--gradle-option=--max-workers=12",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(invocation.gradleArgs, " "); got != "--daemon --parallel --no-scan --max-workers=12 :service-a:assemble" {
		t.Fatalf("Spring-compatible Gradle arguments = %q", got)
	}
}

func TestImpactHotStateRequiresExactBindingAndFailsClosedOnDrift(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	wrapperRoot := filepath.Join(repositoryRoot, "gradle", "wrapper")
	if err := os.MkdirAll(wrapperRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wrapperRoot, "gradle-wrapper.jar"), []byte("wrapper-jar"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wrapperRoot, "gradle-wrapper.properties"), []byte("distributionUrl=fixed"), 0o600); err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	if err := os.Chmod(stateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
		"--repository-revision", "0123456789abcdef0123456789abcdef01234567",
		"--hot-state-dir", stateRoot,
		"--gradle-option=--no-daemon",
	}
	miss, err := prepareImpactInvocation(args, false)
	if err != nil {
		t.Fatal(err)
	}
	if miss.hotStateHit || !miss.plan.CandidateSelected {
		t.Fatalf("initial hot-state miss = %+v", miss)
	}
	hit, err := prepareImpactInvocation(args, false)
	if err != nil {
		t.Fatal(err)
	}
	if !hit.hotStateHit || !reflect.DeepEqual(hit.plan.Entrypoints, miss.plan.Entrypoints) {
		t.Fatalf("exact hot-state hit = %+v", hit)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "changed.txt"), []byte("unowned/file.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	drift, err := prepareImpactInvocation(args, false)
	if err != nil {
		t.Fatal(err)
	}
	if drift.hotStateHit || drift.plan.CandidateSelected || drift.plan.Reason != "IMPACT_UNKNOWN_CHANGE_PATH" {
		t.Fatalf("changed-path drift reused state = %+v", drift)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "changed.txt"), []byte("library-c/src/main/java/synthetic/LibraryC.java\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wrapperRoot, "gradle-wrapper.jar"), []byte("drifted-wrapper-jar"), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapperDrift, err := prepareImpactInvocation(args, false)
	if err != nil {
		t.Fatal(err)
	}
	if wrapperDrift.hotStateHit || !wrapperDrift.plan.CandidateSelected {
		t.Fatalf("wrapper drift reused state = %+v", wrapperDrift)
	}
	revisionArgs := append([]string(nil), args...)
	for index := range revisionArgs {
		if revisionArgs[index] == "0123456789abcdef0123456789abcdef01234567" {
			revisionArgs[index] = "1123456789abcdef0123456789abcdef01234567"
		}
	}
	revisionDrift, err := prepareImpactInvocation(revisionArgs, false)
	if err != nil {
		t.Fatal(err)
	}
	if revisionDrift.hotStateHit || !revisionDrift.plan.CandidateSelected {
		t.Fatalf("revision drift reused state = %+v", revisionDrift)
	}
}

func TestImpactInvocationEmptyDiffRetainsFullGraph(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	if err := os.WriteFile(filepath.Join(repositoryRoot, "changed.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	invocation, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.plan.CandidateSelected || invocation.plan.Reason != "IMPACT_NO_DECLARED_CHANGES" || strings.Join(invocation.gradleArgs, " ") != "assemble" {
		t.Fatalf("empty-diff invocation = %+v", invocation)
	}
}

func TestImpactRunWritesReconciledPhaseTimings(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	clearGradleManagedL1Inputs(t)
	t.Setenv(gradleInitScriptEnvironment, "")
	t.Setenv(gradlePluginJarEnvironment, "")
	wrapper := filepath.Join(repositoryRoot, gradleWrapperName(runtime.GOOS))
	contents := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		contents = "@echo off\r\nexit /b 0\r\n"
	}
	if err := os.WriteFile(wrapper, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"impact",
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
		"--timings-file", "impact-timings.json",
		"--gradle-option=--no-daemon",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("impact exit code = %d, stderr = %q", code, stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, "impact-timings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report impactTimingReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != impactTimingSchemaVersion ||
		!report.CandidateSelected || report.AlternativeID != "affected-service-a" ||
		report.Reason != "EXPLICIT_OWNER_POC_CANDIDATE" ||
		report.EntrypointCount != 1 || report.ExitCode != 0 {
		t.Fatalf("phase timing identity = %+v", report)
	}
	attributed := report.ImpactPreparationNs + report.GradleSetupNs +
		report.RuntimeSetupNs + report.GradleExecutionNs + report.TeardownNs +
		report.UnattributedNs
	if report.TotalNs <= 0 || attributed != report.TotalNs ||
		report.GradleExecutionNs <= 0 || report.ImpactPreparationNs <= 0 {
		t.Fatalf("top-level phase timings do not reconcile: %+v", report)
	}
	plannerSum := report.Planner.ManifestLoadAndValidationNs +
		report.Planner.GraphLoadAndValidationNs +
		report.Planner.GeneratedStateLoadAndValidationNs +
		report.Planner.ImpactEvaluationNs
	if plannerSum <= 0 || report.Planner.TotalNs < plannerSum ||
		report.Planner.TotalNs > report.ImpactPreparationNs {
		t.Fatalf("planner phase timings do not reconcile: %+v", report.Planner)
	}
}

func TestImpactInvocationRejectsUnsafeTimingsPath(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	_, err := prepareImpactInvocation([]string{
		"--repository-id", "tonyredondo/buildopt-impact-synthetic",
		"--changes-file", "changed.txt",
		"--timings-file", "../impact-timings.json",
	}, false)
	if err == nil {
		t.Fatal("escaping timings path was accepted")
	}
}

func impactTestRepository(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve launcher test path")
	}
	source := filepath.Join(filepath.Dir(current), filepath.FromSlash("../../fixtures/build-impact/synthetic-repository"))
	destination := t.TempDir()
	for _, name := range []string{"buildopt-impact-manifest.json", "buildopt-impact-graph.generated.json", "buildopt-impact.generated.json"} {
		raw, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(destination, "changed.txt"), []byte("library-c/src/main/java/synthetic/LibraryC.java\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return destination
}
