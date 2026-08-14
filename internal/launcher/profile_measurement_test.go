package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

func TestHashMeasurementOutputsIsPathAndContentBound(t *testing.T) {
	repository := t.TempDir()
	for path, content := range map[string]string{
		"service/build/libs/a.jar": "alpha",
		"service/build/libs/b.jar": "beta",
		"other/build/libs/c.jar":   "ignored",
	} {
		absolute := filepath.Join(repository, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	firstSHA, firstCount, err := hashMeasurementOutputs(repository, []string{"service/build/libs/**"})
	if err != nil || firstCount != 2 || !validMeasurementRevision(firstSHA[:40]) {
		t.Fatalf("first output set = %s/%d/%v", firstSHA, firstCount, err)
	}
	if err := os.WriteFile(filepath.Join(repository, "service", "build", "libs", "b.jar"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondSHA, secondCount, err := hashMeasurementOutputs(repository, []string{"service/build/libs/**"})
	if err != nil || secondCount != firstCount || secondSHA == firstSHA {
		t.Fatalf("changed output set = %s/%d/%v", secondSHA, secondCount, err)
	}
}

func TestMeasurementHelpersFailClosed(t *testing.T) {
	if got := measurementFallbackReason("buildopt: Build Impact POC retained the full graph (IMPACT_GLOBAL_CHANGE)\n"); got != "IMPACT_GLOBAL_CHANGE" {
		t.Fatalf("fallback reason = %q", got)
	}
	if got := measurementFallbackReason("candidate selected"); got != "" {
		t.Fatalf("unexpected fallback reason = %q", got)
	}
	if !matchMeasurementGlob("service/**/libs/*.jar", "service/build/libs/a.jar") ||
		matchMeasurementGlob("service/**/libs/*.jar", "other/build/libs/a.jar") {
		t.Fatal("recursive required-output glob is not conservative")
	}
	if validMeasurementRevision(strings.Repeat("A", 40)) || validMeasurementRevision("abc") {
		t.Fatal("invalid revision was accepted")
	}
	if got := nullDelimitedPaths("with space.txt\x00café.txt\x00"); len(got) != 2 || got[0] != "with space.txt" || got[1] != "café.txt" {
		t.Fatalf("NUL-delimited paths = %#v", got)
	}
	file := filepath.Join(t.TempDir(), "buildopt")
	if err := os.WriteFile(file, []byte("exact executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	digest, err := hashMeasurementFile(file)
	if err != nil || digest != "c724d05a236c3cd227411e93cdf5ef0d23b7a9e4e254dc0ff9ef0b699284dffc" {
		t.Fatalf("executable digest = %q/%v", digest, err)
	}
}

func TestCopyMeasurementInputRejectsSymlinkParent(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "contracts", "input.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(target, "contracts")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := copyMeasurementInput(source, target, filepath.Join("contracts", "input.json")); err == nil {
		t.Fatal("measurement input escaped through a symlink parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "input.json")); !os.IsNotExist(err) {
		t.Fatalf("measurement input escaped the isolated repository: %v", err)
	}
}

func TestMeasurementEnvironmentRemovesExternalBuildOptState(t *testing.T) {
	t.Setenv("BUILDOPT_SERVER_URL", "https://should-not-propagate.invalid")
	t.Setenv("GRADLE_USER_HOME", "wrong")
	environment := measurementEnvironment("isolated-home")
	joined := strings.Join(environment, "\n")
	if strings.Contains(joined, "BUILDOPT_SERVER_URL=") || strings.Contains(joined, "GRADLE_USER_HOME=wrong") ||
		!strings.Contains(joined, "GRADLE_USER_HOME=isolated-home") {
		t.Fatalf("measurement environment = %q", joined)
	}
}

func TestMeasurementGradleDistributionSeedIsPrivateAndExecutable(t *testing.T) {
	gradleHome := t.TempDir()
	seed := filepath.Join(gradleHome, "wrapper", "dists")
	bin := filepath.Join(seed, "gradle-9.6.1-bin", "checksum", "gradle-9.6.1", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gradle"), []byte("launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seed, "marker.ok"), []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRADLE_USER_HOME", gradleHome)
	resolved, err := measurementGradleDistributionSeed()
	if err != nil || resolved != seed {
		t.Fatalf("distribution seed = %q/%v", resolved, err)
	}
	target := filepath.Join(t.TempDir(), "wrapper", "dists")
	if err := copyMeasurementDistributionTree(resolved, target); err != nil {
		t.Fatal(err)
	}
	launcherInfo, err := os.Stat(filepath.Join(target, "gradle-9.6.1-bin", "checksum", "gradle-9.6.1", "bin", "gradle"))
	if err != nil || (runtime.GOOS != "windows" && launcherInfo.Mode().Perm() != 0o700) {
		t.Fatalf("copied launcher mode = %v/%v", launcherInfo, err)
	}
	markerInfo, err := os.Stat(filepath.Join(target, "marker.ok"))
	if err != nil || (runtime.GOOS != "windows" && markerInfo.Mode().Perm() != 0o600) {
		t.Fatalf("copied marker mode = %v/%v", markerInfo, err)
	}
}

func TestMeasurementGradleDistributionSeedRejectsSymlinks(t *testing.T) {
	seed := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(seed, "escaped")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := copyMeasurementDistributionTree(seed, filepath.Join(t.TempDir(), "target")); err == nil ||
		!strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("symlink seed error = %v", err)
	}
}

func TestSummarizeStructuralTaskOutcomes(t *testing.T) {
	log := strings.Join([]string{
		"configuration output",
		"> Task :compileJava",
		"> Task :resources FROM-CACHE",
		"> Task :classes UP-TO-DATE",
		"> Task :empty NO-SOURCE",
		"> Task :disabled SKIPPED",
		"> Task :other",
	}, "\n")
	outcomes := summarizeStructuralTaskOutcomes(log)
	if outcomes.Total != 6 || outcomes.Executed != 2 || outcomes.FromCache != 1 ||
		outcomes.UpToDate != 1 || outcomes.NoSource != 1 || outcomes.Skipped != 1 ||
		outcomes.FingerprintSHA256 == "" || len(outcomes.Tasks) != outcomes.Total {
		t.Fatalf("task outcomes = %+v", outcomes)
	}
	if outcomes.Tasks[0].Path != ":classes" || outcomes.Tasks[0].Outcome != "UP_TO_DATE" ||
		outcomes.Tasks[5].Path != ":resources" || outcomes.Tasks[5].Outcome != "FROM_CACHE" {
		t.Fatalf("exact task observations = %+v", outcomes.Tasks)
	}
	reordered := summarizeStructuralTaskOutcomes(strings.Join([]string{
		"> Task :disabled SKIPPED",
		"> Task :other",
		"> Task :empty NO-SOURCE",
		"> Task :classes UP-TO-DATE",
		"> Task :resources FROM-CACHE",
		"> Task :compileJava",
	}, "\n"))
	if reordered.FingerprintSHA256 != outcomes.FingerprintSHA256 {
		t.Fatalf("task fingerprint depends on console order: %q != %q", reordered.FingerprintSHA256, outcomes.FingerprintSHA256)
	}
	changed := summarizeStructuralTaskOutcomes(strings.Replace(log, ":other", ":different", 1))
	if changed.FingerprintSHA256 == outcomes.FingerprintSHA256 {
		t.Fatal("task fingerprint did not bind the task path")
	}
}

func TestSummarizeStructuralTaskOutcomesNormalizesRepeatedConsoleLines(t *testing.T) {
	repeated := summarizeStructuralTaskOutcomes(strings.Join([]string{
		"> Task :javadoc",
		"warning emitted by javadoc",
		"> Task :javadoc",
		"> Task :classes UP-TO-DATE",
	}, "\n"))
	if repeated.Total != 2 || repeated.Executed != 1 || repeated.UpToDate != 1 || len(repeated.Tasks) != 2 {
		t.Fatalf("repeated task lines = %+v", repeated)
	}
	if err := profilediscovery.ValidateStructuralTaskOutcomes(repeated); err != nil {
		t.Fatalf("normalized task evidence: %v", err)
	}

	conflicting := summarizeStructuralTaskOutcomes(strings.Join([]string{
		"> Task :compileJava",
		"> Task :compileJava FROM-CACHE",
	}, "\n"))
	if err := profilediscovery.ValidateStructuralTaskOutcomes(conflicting); err != nil {
		t.Fatalf("build-tree task-path collision was rejected: %v", err)
	}
	if conflicting.Total != 1 || conflicting.FromCache != 1 || len(conflicting.Tasks) != 1 ||
		conflicting.Tasks[0].Outcome != "FROM_CACHE" ||
		!reflect.DeepEqual(conflicting.Tasks[0].ConsoleOutcomeTransitions, []string{"EXECUTED", "FROM_CACHE"}) {
		t.Fatalf("build-tree task-path collision = %+v", conflicting)
	}
	terminalOnly := summarizeStructuralTaskOutcomes("> Task :compileJava FROM-CACHE\n")
	if conflicting.FingerprintSHA256 != terminalOnly.FingerprintSHA256 {
		t.Fatal("console transition changed the terminal task fingerprint")
	}
}

func TestStructuralTaskEvidenceErrorPreservesDiagnosticLog(t *testing.T) {
	err := structuralTaskEvidenceError("candidate", errors.New("conflicting task"), strings.Join([]string{
		"> Task :compileJava",
		"> Task :compileJava FROM-CACHE",
	}, "\n"))
	if !strings.Contains(err.Error(), "candidate arm produced invalid exact task evidence: conflicting task") ||
		!strings.Contains(err.Error(), "> Task :compileJava FROM-CACHE") {
		t.Fatalf("diagnostic error = %q", err)
	}
}

func TestStructuralPressureParsingAndDelta(t *testing.T) {
	some, full, err := parseStructuralPressure([]byte(
		"some avg10=0.10 avg60=0.20 avg300=0.30 total=12345\n"+
			"full avg10=0.00 avg60=0.00 avg300=0.00 total=2345\n"), true)
	if err != nil || some != 12345 || full != 2345 {
		t.Fatalf("pressure totals = %d/%d/%v", some, full, err)
	}
	if _, _, err := parseStructuralPressure([]byte("some total=1\n"), true); err == nil {
		t.Fatal("incomplete full-pressure record was accepted")
	}
	before := structuralPressureSnapshot{
		available: true, cpuSomeTotalUS: 10, memorySomeTotalUS: 20,
		memoryFullTotalUS: 3, ioSomeTotalUS: 40, ioFullTotalUS: 5,
	}
	after := structuralPressureSnapshot{
		available: true, cpuSomeTotalUS: 18, memorySomeTotalUS: 22,
		memoryFullTotalUS: 4, ioSomeTotalUS: 55, ioFullTotalUS: 9,
	}
	delta := structuralPressureDelta(before, after)
	if !delta.Available || delta.CPUSomeTotalUS != 8 || delta.MemorySomeTotalUS != 2 ||
		delta.MemoryFullTotalUS != 1 || delta.IOSomeTotalUS != 15 || delta.IOFullTotalUS != 4 {
		t.Fatalf("pressure delta = %+v", delta)
	}
	if structuralPressureDelta(after, before).Available {
		t.Fatal("decreasing Linux PSI total was accepted")
	}
}

func TestStructuralFallbackGradleOptionsPreserveMeasuredSchedulingWithoutMutatingOptions(t *testing.T) {
	measured := []string{"--daemon", "--build-cache", "--parallel", "--no-configuration-cache", "--console=plain", "--no-scan", "--max-workers=12"}
	fallback := structuralFallbackGradleOptions(measured)
	if got := strings.Join(fallback, " "); got != "--build-cache --parallel --no-configuration-cache --console=plain --no-scan --max-workers=12 --no-daemon" {
		t.Fatalf("fallback options = %q", got)
	}
	if got := strings.Join(measured, " "); got != "--daemon --build-cache --parallel --no-configuration-cache --console=plain --no-scan --max-workers=12" {
		t.Fatalf("measured options mutated = %q", got)
	}
}

func TestDescribeMeasurementOutputDifferenceIsBoundedAndPathSpecific(t *testing.T) {
	expected := t.TempDir()
	actual := t.TempDir()
	writeOutput := func(root, relative, content string) {
		t.Helper()
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeOutput(expected, "build/classes/changed.class", "before")
	writeOutput(actual, "build/classes/changed.class", "after")
	writeOutput(expected, "build/classes/missing.class", "missing")
	writeOutput(actual, "build/classes/unexpected.class", "unexpected")

	difference, err := describeMeasurementOutputDifference(expected, actual, []string{"build/classes/**"})
	if err != nil {
		t.Fatal(err)
	}
	for _, expectedFragment := range []string{
		"changed build/classes/changed.class",
		"missing build/classes/missing.class",
		"unexpected build/classes/unexpected.class",
	} {
		if !strings.Contains(difference, expectedFragment) {
			t.Fatalf("difference %q does not contain %q", difference, expectedFragment)
		}
	}
}
