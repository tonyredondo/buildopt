package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOptimizeContractRunsNativeAndResumesOnlyExactBindings(t *testing.T) {
	repository := newOptimizeTestRepository(t)
	t.Chdir(repository)
	t.Setenv(bypassEnvironment, "")
	t.Setenv("BUILDOPT_OPTIMIZE_TEST_EXIT", "37")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"optimize",
		"--state-dir", ".buildopt/optimize-test/v1",
		"--calibration-budget", "25m",
		"--calibration-pairs", "6",
		"--max-break-even-builds", "20",
		"build", "--no-daemon",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 37 {
		t.Fatalf("optimize exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "native:--build-cache build --no-daemon") {
		t.Fatalf("native stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "NATIVE_RETAINED (NATIVE_BUILD_FAILED)") ||
		!strings.Contains(stderr.String(), "production authorization remains false") {
		t.Fatalf("optimize stderr = %q", stderr.String())
	}

	resultPath := filepath.Join(repository, ".buildopt", "optimize-test", "v1", optimizeResultFile)
	statePath := filepath.Join(repository, ".buildopt", "optimize-test", "v1", optimizeStateFile)
	result := readOptimizeResultForTest(t, resultPath)
	if result.SchemaVersion != optimizeResultSchemaVersion || result.Outcome != optimizeOutcomeNative ||
		result.Reason != "NATIVE_BUILD_FAILED" || result.Phase != "NATIVE_RETAINED" ||
		result.Generation != 1 || result.Attempt != 1 || result.Native.ExitCode != 37 ||
		!result.Native.Authoritative || !result.Native.Started || result.ManualFilesRequired != 0 ||
		result.CalibrationPerformed || result.SelectionPerformed || result.ProductionAuthorized ||
		result.TestOptimization != "OUT_OF_SCOPE" || result.Budget.WallTimeSeconds != 1500 ||
		result.Budget.Pairs != 6 || result.Budget.MaxBreakEvenBuilds != 20 ||
		result.Resume.CheckpointFound || result.Resume.Accepted ||
		result.Resume.Reason != optimizeResumeNone {
		t.Fatalf("first optimize result = %+v", result)
	}
	assertPrivateOptimizeFile(t, statePath)
	assertPrivateOptimizeFile(t, resultPath)

	t.Setenv("BUILDOPT_OPTIMIZE_TEST_EXIT", "0")
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"optimize", "--state-dir", ".buildopt/optimize-test/v1",
		"--calibration-budget", "25m", "--calibration-pairs", "6",
		"--max-break-even-builds", "20", "build", "--no-daemon",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resumed optimize exit code = %d, stderr = %q", code, stderr.String())
	}
	result = readOptimizeResultForTest(t, resultPath)
	if result.Generation != 1 || result.Attempt != 2 || !result.Resume.CheckpointFound ||
		!result.Resume.Accepted || result.Resume.Reason != optimizeResumeExact ||
		len(result.Resume.PreviousStateSHA256) != 64 {
		t.Fatalf("exact resume result = %+v", result)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"optimize", "--state-dir", ".buildopt/optimize-test/v1",
		"--calibration-budget", "25m", "--calibration-pairs", "6",
		"--max-break-even-builds", "20", "test", "--no-daemon",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("drifted optimize exit code = %d, stderr = %q", code, stderr.String())
	}
	result = readOptimizeResultForTest(t, resultPath)
	if result.Generation != 2 || result.Attempt != 1 || !result.Resume.CheckpointFound ||
		result.Resume.Accepted || result.Resume.Reason != optimizeResumeDrift {
		t.Fatalf("binding-drift result = %+v", result)
	}
}

func TestOptimizeJSONAndBypass(t *testing.T) {
	repository := newOptimizeTestRepository(t)
	t.Chdir(repository)
	t.Setenv("BUILDOPT_OPTIMIZE_TEST_EXIT", "0")
	t.Setenv(bypassEnvironment, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"optimize", "--json", "--state-dir", ".buildopt/json/v1", "assemble",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("JSON optimize exit code = %d, stderr = %q", code, stderr.String())
	}
	var result optimizeResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode JSON stdout %q: %v", stdout.String(), err)
	}
	if result.Outcome != optimizeOutcomeNative || !result.Native.Started ||
		result.Reason != "TARGET_REVISION_UNAVAILABLE" ||
		strings.Contains(stdout.String(), "native:") ||
		!strings.Contains(stderr.String(), "native:--build-cache assemble") ||
		strings.Contains(stderr.String(), "BuildOpt optimize:") {
		t.Fatalf("JSON result = %+v, stdout = %q, stderr = %q", result, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(repository, ".buildopt", "json", "v1", optimizeResultFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) != string(raw) {
		t.Fatalf("JSON stdout differs from result file\nstdout: %s\nfile: %s", stdout.String(), raw)
	}

	t.Setenv(bypassEnvironment, "1")
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"optimize", "--state-dir", ".buildopt/bypass/v1", "build",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "native:build") ||
		strings.Contains(stderr.String(), "BuildOpt optimize:") {
		t.Fatalf("bypass code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(repository, ".buildopt", "bypass")); !os.IsNotExist(err) {
		t.Fatalf("bypass created optimize state: %v", err)
	}
}

func TestOptimizeHelpAndInvalidArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"optimize", "--help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != optimizeUsage || stderr.Len() != 0 {
		t.Fatalf("help stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}

	for _, arguments := range [][]string{
		{"optimize"},
		{"optimize", "--resume", "sometimes", "build"},
		{"optimize", "--calibration-budget", "0s", "build"},
		{"optimize", "--calibration-pairs", "1", "build"},
		{"optimize", "--state-dir", "../outside", "build"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := Run(arguments, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
			t.Fatalf("arguments %q exit code = %d, want %d", arguments, code, exitUsage)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), optimizeUsage) {
			t.Fatalf("arguments %q stdout = %q, stderr = %q", arguments, stdout.String(), stderr.String())
		}
	}
}

func TestOptimizeStateDirectoryNormalization(t *testing.T) {
	for _, relative := range []string{
		"",
		".",
		"../outside",
		".buildopt/../outside",
		"state/optimize",
	} {
		if normalized, valid := normalizeOptimizeStateRelative(relative); valid {
			t.Fatalf("state directory %q normalized to %q, want rejection", relative, normalized)
		}
	}

	normalized, valid := normalizeOptimizeStateRelative(".buildopt/optimize/v1")
	if !valid || filepath.ToSlash(normalized) != ".buildopt/optimize/v1" {
		t.Fatalf("portable state directory normalized to %q, valid = %t", normalized, valid)
	}
}

func TestOptimizeInvalidCheckpointRunsNativeWithoutReuse(t *testing.T) {
	repository := newOptimizeTestRepository(t)
	t.Chdir(repository)
	t.Setenv(bypassEnvironment, "")
	t.Setenv("BUILDOPT_OPTIMIZE_TEST_EXIT", "0")
	stateDirectory := filepath.Join(repository, ".buildopt", "corrupt", "v1")
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDirectory, optimizeStateFile), []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"optimize", "--state-dir", ".buildopt/corrupt/v1", "check",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "native:--build-cache check") {
		t.Fatalf("invalid-checkpoint code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	result := readOptimizeResultForTest(t, filepath.Join(stateDirectory, optimizeResultFile))
	if !result.Resume.CheckpointFound || result.Resume.Accepted ||
		result.Resume.Reason != optimizeResumeInvalid ||
		len(result.Resume.PreviousStateSHA256) != 64 || !result.Native.Started {
		t.Fatalf("invalid-checkpoint result = %+v", result)
	}
}

func TestOptimizeRejectsDiscoveryAuthorityInCheckpoint(t *testing.T) {
	repository := newOptimizeTestRepository(t)
	t.Chdir(repository)
	t.Setenv(bypassEnvironment, "")
	t.Setenv("BUILDOPT_OPTIMIZE_TEST_EXIT", "0")
	stateDirectory := filepath.Join(repository, ".buildopt", "authority", "v1")
	arguments := []string{"optimize", "--state-dir", ".buildopt/authority/v1", "jar"}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run(arguments, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("initial optimize exit code = %d, stderr = %q", code, stderr.String())
	}
	statePath := filepath.Join(stateDirectory, optimizeStateFile)
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var state optimizeState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	state.Discovery.ProductionAuthorized = true
	mutated, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, mutated, 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run(arguments, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("mutated optimize exit code = %d, stderr = %q", code, stderr.String())
	}
	result := readOptimizeResultForTest(t, filepath.Join(stateDirectory, optimizeResultFile))
	if result.Resume.Reason != optimizeResumeInvalid || result.Resume.Accepted ||
		result.Generation != 1 || result.Attempt != 1 {
		t.Fatalf("mutated discovery checkpoint result = %+v", result)
	}
}

func newOptimizeTestRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	wrapper := filepath.Join(repository, gradleWrapperName(runtime.GOOS))
	contents := "#!/bin/sh\nprintf 'native:%s\\n' \"$*\"\nexit \"${BUILDOPT_OPTIMIZE_TEST_EXIT:-0}\"\n"
	if runtime.GOOS == "windows" {
		contents = "@echo off\r\necho native:%*\r\nexit /b %BUILDOPT_OPTIMIZE_TEST_EXIT%\r\n"
	}
	if err := os.WriteFile(wrapper, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGradleWrapperProperties(t, repository, "distributionUrl=gradle-9.6.1-bin.zip\n")
	return repository
}

func readOptimizeResultForTest(t *testing.T, path string) optimizeResult {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result optimizeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertPrivateOptimizeFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode for %s = %o, want 600", path, info.Mode().Perm())
	}
}
