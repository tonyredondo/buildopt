package launcher

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestQualifiedPOCProfileSelectsOnlyQualifiedMechanisms(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)

	invocation, err := prepareQualifiedPOCProfileInvocation([]string{
		"--changes-file", "changed.txt",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !invocation.plan.CandidateSelected || !invocation.standardJarCache ||
		invocation.standardCopyCache || invocation.hotStateHit ||
		invocation.qualifiedProfile == nil {
		t.Fatalf("qualified invocation = %+v", invocation)
	}
	plan := invocation.qualifiedProfile
	if plan.ProfileID != qualifiedPOCProfileID ||
		plan.ClaimScope != qualifiedPOCProfileClaimScope ||
		plan.SelectionMode != "POC_CANDIDATE" ||
		plan.AlternativeID != "affected-service-a" ||
		strings.Join(plan.Entrypoints, " ") != ":service-a:assemble" ||
		strings.Join(plan.FallbackEntrypoints, " ") != "assemble" ||
		strings.Join(plan.EnabledAdapters, " ") != "STANDARD_JAR" ||
		strings.Join(plan.ExpectedOutputs, " ") != "service-a/build/libs/service-a-*.jar" ||
		plan.OmittedProjectCount != 1 || plan.ProductionAuthorized {
		t.Fatalf("qualified plan = %+v", plan)
	}
}

func TestQualifiedPOCProfileUnknownChangeUsesNativeFullGraph(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	if err := os.WriteFile(filepath.Join(repositoryRoot, "changed.txt"), []byte("unowned/file.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	invocation, err := prepareQualifiedPOCProfileInvocation([]string{
		"--changes-file", "changed.txt",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.plan.CandidateSelected || invocation.standardJarCache ||
		invocation.standardCopyCache || invocation.qualifiedProfile == nil ||
		strings.Join(invocation.gradleArgs, " ") != "--no-daemon assemble" ||
		len(invocation.qualifiedProfile.EnabledAdapters) != 0 ||
		invocation.qualifiedProfile.SelectionMode != "FULL_GRAPH" {
		t.Fatalf("fallback invocation = %+v", invocation)
	}
}

func TestQualifiedPOCProfileBypassUsesNativeFullGraph(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)

	invocation, err := prepareQualifiedPOCProfileInvocation([]string{
		"--changes-file", "changed.txt",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.plan.CandidateSelected || invocation.standardJarCache ||
		invocation.qualifiedProfile == nil ||
		invocation.qualifiedProfile.SelectionReason != "LOCAL_BYPASS" ||
		strings.Join(invocation.gradleArgs, " ") != "--no-daemon assemble" {
		t.Fatalf("bypass invocation = %+v", invocation)
	}
}

func TestQualifiedPOCProfileRejectsUnqualifiedMechanismsAndDrift(t *testing.T) {
	for name, replacement := range map[string]string{
		"safe cache":       `"safeCache": true`,
		"runtime tuning":   `"runtimeTuning": true`,
		"hot state":        `"hotState": true`,
		"standard copy":    `"standardCopyAdapter": true`,
		"shared edge":      `"sharedEdgeCache": true`,
		"unknown property": `"unexpected": false`,
	} {
		t.Run(name, func(t *testing.T) {
			repositoryRoot := impactTestRepository(t)
			t.Chdir(repositoryRoot)
			path := filepath.Join(repositoryRoot, qualifiedPOCProfileDefaultPath)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if name == "unknown property" {
				raw = bytes.Replace(raw, []byte(`"profileVersion": 1,`), []byte(`"profileVersion": 1, "unexpected": false,`), 1)
			} else {
				field := strings.Replace(replacement, "true", "false", 1)
				raw = bytes.Replace(raw, []byte(field), []byte(replacement), 1)
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := prepareQualifiedPOCProfileInvocation([]string{"--changes-file", "changed.txt"}, false); err == nil {
				t.Fatal("drifted qualified profile was accepted")
			}
		})
	}
	for _, field := range []string{"buildImpact", "standardJarAdapter"} {
		t.Run(field+" disabled", func(t *testing.T) {
			repositoryRoot := impactTestRepository(t)
			t.Chdir(repositoryRoot)
			path := filepath.Join(repositoryRoot, qualifiedPOCProfileDefaultPath)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			raw = bytes.Replace(
				raw,
				[]byte(`"`+field+`": true`),
				[]byte(`"`+field+`": false`),
				1,
			)
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := prepareQualifiedPOCProfileInvocation([]string{"--changes-file", "changed.txt"}, false); err == nil {
				t.Fatal("incomplete qualified profile was accepted")
			}
		})
	}
}

func TestQualifiedPOCProfileRunReportsPlanBeforeGradle(t *testing.T) {
	repositoryRoot := impactTestRepository(t)
	t.Chdir(repositoryRoot)
	clearGradleManagedL1Inputs(t)
	initScript := filepath.Join(repositoryRoot, "buildopt.init.gradle")
	pluginJar := filepath.Join(repositoryRoot, "buildopt-gradle-plugin.jar")
	writeGradleTestFile(t, initScript)
	writeGradleTestFile(t, pluginJar)
	t.Setenv(gradleInitScriptEnvironment, initScript)
	t.Setenv(gradlePluginJarEnvironment, pluginJar)
	t.Setenv(gradleSafeCacheEnvironment, "1")
	t.Setenv(gradleStandardCopyCacheEnvironment, "1")
	t.Setenv(gradleCheckstyleHeapEnvironment, "2g")
	t.Setenv(sessioningestServerURLForTest, "not-a-valid-url")
	writeGradleWrapperProperties(t, repositoryRoot, "distributionUrl=gradle-9.6.1-bin.zip\n")

	wrapper := filepath.Join(repositoryRoot, gradleWrapperName(runtime.GOOS))
	contents := "#!/bin/sh\nprintf 'wrapper-start jar=%s safe=%s copy=%s mode=%s server=%s\\n' \"$BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS\" \"$BUILDOPT_SAFE_CACHE\" \"$BUILDOPT_CACHE_STANDARD_COPY_TASKS\" \"$BUILDOPT_GRADLE_PROJECT_PLUGIN_MODE\" \"$BUILDOPT_SERVER_URL\" >&2\n"
	if runtime.GOOS == "windows" {
		contents = "@echo off\r\necho wrapper-start jar=%BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS% safe=%BUILDOPT_SAFE_CACHE% copy=%BUILDOPT_CACHE_STANDARD_COPY_TASKS% mode=%BUILDOPT_GRADLE_PROJECT_PLUGIN_MODE% server=%BUILDOPT_SERVER_URL% 1>&2\r\n"
	}
	if err := os.WriteFile(wrapper, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"poc", "--changes-file", "changed.txt"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("qualified POC exit code = %d, stderr = %q", code, stderr.String())
	}
	planIndex := strings.Index(stderr.String(), "buildopt: qualified POC profile plan {")
	wrapperIndex := strings.Index(stderr.String(), "wrapper-start ")
	if planIndex < 0 || wrapperIndex < 0 || planIndex >= wrapperIndex {
		t.Fatalf("plan was not reported before Gradle: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), `"enabledAdapters":["STANDARD_JAR"]`) ||
		!strings.Contains(stderr.String(), `"expectedOutputs":["service-a/build/libs/service-a-*.jar"]`) ||
		!strings.Contains(stderr.String(), "jar=1 safe= copy= mode=CACHE_ONLY server=") {
		t.Fatalf("qualified POC output = %q", stderr.String())
	}
}

func TestQualifiedPOCProfileHelpAndInvalidArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"poc", "--help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != qualifiedPOCProfileUsage || stderr.Len() != 0 {
		t.Fatalf("help stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"poc"}, strings.NewReader(""), &stdout, &stderr); code != exitConfiguration {
		t.Fatalf("invalid invocation exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), qualifiedPOCProfileUsage) {
		t.Fatalf("invalid invocation stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

const sessioningestServerURLForTest = "BUILDOPT_SERVER_URL"
