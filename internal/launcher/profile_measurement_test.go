package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

func TestStructuralProfileMeasurementHelpDocumentsCalibrationMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := runStructuralProfileMeasurement([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[--calibration-only]") {
		t.Fatalf("help does not expose calibration mode: %q", stdout.String())
	}
}

func TestStructuralMeasurementDeadlineBoundsEveryChildTimeout(t *testing.T) {
	config := structuralMeasurementConfig{timeout: time.Hour, deadline: time.Now().Add(time.Minute)}
	timeout, err := structuralMeasurementTimeout(config)
	if err != nil || timeout <= 0 || timeout > time.Minute {
		t.Fatalf("bounded structural timeout = %s/%v", timeout, err)
	}
	config.deadline = time.Now().Add(-time.Millisecond)
	if _, err := structuralMeasurementTimeout(config); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired structural deadline = %v, want deadline exceeded", err)
	}
}

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
	t.Setenv(gradleReadOnlyDependencyEnvironment, "wrong-dependency-cache")
	environment := measurementEnvironment("isolated-home", "shared-readonly-dependencies")
	joined := strings.Join(environment, "\n")
	if strings.Contains(joined, "BUILDOPT_SERVER_URL=") || strings.Contains(joined, "GRADLE_USER_HOME=wrong") ||
		strings.Contains(joined, gradleReadOnlyDependencyEnvironment+"=wrong-dependency-cache") ||
		!strings.Contains(joined, "GRADLE_USER_HOME=isolated-home") ||
		!strings.Contains(joined, gradleReadOnlyDependencyEnvironment+"=shared-readonly-dependencies") {
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

func TestStructuralMeasurementSharesOnlyImmutableGradleCaches(t *testing.T) {
	gradleHome := t.TempDir()
	modulesRoot := filepath.Join(gradleHome, "caches", "modules-2")
	artifact := filepath.Join(modulesRoot, "files-2.1", "example", "artifact.jar")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("resolved dependency"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, transient := range []string{
		filepath.Join(modulesRoot, "modules-2.lock"),
		filepath.Join(modulesRoot, "gc.properties"),
	} {
		if err := os.WriteFile(transient, []byte("transient"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	nativeBuildCache := filepath.Join(gradleHome, "caches", "build-cache-1")
	if err := os.MkdirAll(nativeBuildCache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nativeBuildCache, "cache-entry"), []byte("native output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nativeBuildCache, "build-cache-1.lock"), []byte("transient"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRADLE_USER_HOME", gradleHome)
	seed, err := measurementGradleDependencySeed()
	if err != nil || seed != modulesRoot {
		t.Fatalf("dependency seed = %q/%v", seed, err)
	}
	buildCacheSeed, err := measurementGradleNativeBuildCacheSeed()
	if err != nil || buildCacheSeed != nativeBuildCache {
		t.Fatalf("native build-cache seed = %q/%v", buildCacheSeed, err)
	}
	measurementRoot := filepath.Join(t.TempDir(), "measurement")
	if err := os.Mkdir(measurementRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	defer cleanupStructuralMeasurementRoot(measurementRoot)
	config, err := prepareStructuralDependencySnapshot(structuralMeasurementConfig{
		gradleDependencySeed:       seed,
		gradleNativeBuildCacheSeed: buildCacheSeed,
		timeout:                    time.Minute,
	}, measurementRoot, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if config.gradleReadOnlyDependencyRoot == "" {
		t.Fatal("read-only dependency root was not prepared")
	}
	copied := filepath.Join(config.gradleReadOnlyDependencyRoot, "modules-2", "files-2.1", "example", "artifact.jar")
	info, err := os.Stat(copied)
	if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o400) {
		t.Fatalf("copied dependency = %v/%v", info, err)
	}
	for _, transient := range []string{"modules-2.lock", "gc.properties"} {
		if _, err := os.Stat(filepath.Join(config.gradleReadOnlyDependencyRoot, "modules-2", transient)); !os.IsNotExist(err) {
			t.Fatalf("transient dependency state %q was copied: %v", transient, err)
		}
	}
	if config.gradleSharedBuildCacheSeed == "" {
		t.Fatal("shared native build-cache seed was not prepared")
	}
	cacheEntry := filepath.Join(config.gradleSharedBuildCacheSeed, "cache-entry")
	cacheInfo, err := os.Stat(cacheEntry)
	if err != nil || (runtime.GOOS != "windows" && cacheInfo.Mode().Perm() != 0o400) {
		t.Fatalf("shared native cache entry = %v/%v", cacheInfo, err)
	}
	if _, err := os.Stat(filepath.Join(config.gradleSharedBuildCacheSeed, "build-cache-1.lock")); !os.IsNotExist(err) {
		t.Fatalf("native build-cache lock was copied: %v", err)
	}
	for _, armName := range []string{"control", "candidate"} {
		armCache := filepath.Join(measurementRoot, armName+"-home", "caches", "build-cache-1")
		if err := copyMeasurementTree(config.gradleSharedBuildCacheSeed, armCache); err != nil {
			t.Fatalf("seed %s arm cache: %v", armName, err)
		}
		raw, err := os.ReadFile(filepath.Join(armCache, "cache-entry"))
		if err != nil || string(raw) != "native output" {
			t.Fatalf("%s arm cache entry = %q/%v", armName, raw, err)
		}
	}
	control := strings.Join(measurementEnvironment("control-home", config.gradleReadOnlyDependencyRoot), "\n")
	candidate := strings.Join(measurementEnvironment("candidate-home", config.gradleReadOnlyDependencyRoot), "\n")
	if !strings.Contains(control, "GRADLE_USER_HOME=control-home") ||
		!strings.Contains(candidate, "GRADLE_USER_HOME=candidate-home") ||
		!strings.Contains(control, gradleReadOnlyDependencyEnvironment+"="+config.gradleReadOnlyDependencyRoot) ||
		!strings.Contains(candidate, gradleReadOnlyDependencyEnvironment+"="+config.gradleReadOnlyDependencyRoot) {
		t.Fatalf("isolated measurement environments = %q / %q", control, candidate)
	}
}

func TestMeasurementGradleDependencySeedRejectsSymlinks(t *testing.T) {
	gradleHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gradleHome, "caches"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(gradleHome, "caches", "modules-2")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	t.Setenv("GRADLE_USER_HOME", gradleHome)
	if _, err := measurementGradleDependencySeed(); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("symlink dependency seed error = %v", err)
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

func TestStructuralFallbackGradleOptionsReuseMeasuredSchedulingWithoutMutatingOptions(t *testing.T) {
	measured := []string{"--daemon", "--build-cache", "--parallel", "--no-configuration-cache", "--console=plain", "--no-scan", "--max-workers=12"}
	fallback := structuralFallbackGradleOptions(measured)
	if got := strings.Join(fallback, " "); got != "--daemon --build-cache --parallel --no-configuration-cache --console=plain --no-scan --max-workers=12" {
		t.Fatalf("fallback options = %q", got)
	}
	fallback[0] = "--no-daemon"
	if got := strings.Join(measured, " "); got != "--daemon --build-cache --parallel --no-configuration-cache --console=plain --no-scan --max-workers=12" {
		t.Fatalf("measured options mutated = %q", got)
	}
}

func TestValidateStructuralPairedTargetShapeRequiresEveryPairToMatchWarmup(t *testing.T) {
	controlFingerprint := strings.Repeat("1", 64)
	candidateFingerprint := strings.Repeat("2", 64)
	control := structuralMeasurementArm{warmups: []profilediscovery.StructuralWarmupObservation{
		{Phase: "CACHE_SEED"},
		{Phase: "TARGET_WORKLOAD_STABILIZATION", TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: controlFingerprint}},
	}}
	candidate := structuralMeasurementArm{warmups: []profilediscovery.StructuralWarmupObservation{
		{Phase: "TARGET_WORKLOAD_STABILIZATION", TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: candidateFingerprint}},
	}}
	controlResult := structuralArmResult{taskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: controlFingerprint}}
	candidateResult := structuralArmResult{taskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: candidateFingerprint}}
	if err := validateStructuralPairedTargetShape(control, candidate, controlResult, candidateResult); err != nil {
		t.Fatal(err)
	}
	candidateResult.taskOutcomes.FingerprintSHA256 = strings.Repeat("3", 64)
	if err := validateStructuralPairedTargetShape(control, candidate, controlResult, candidateResult); err == nil {
		t.Fatal("candidate task-shape drift was accepted")
	}
}

func TestStructuralTargetWarmupsRequireTwoFinalMatchingFingerprints(t *testing.T) {
	warmups := []profilediscovery.StructuralWarmupObservation{
		{TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: strings.Repeat("1", 64)}},
		{TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: strings.Repeat("2", 64)}},
		{TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: strings.Repeat("3", 64)}},
		{TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: strings.Repeat("4", 64)}},
		{TaskOutcomes: profilediscovery.StructuralTaskOutcomes{FingerprintSHA256: strings.Repeat("4", 64)}},
	}
	if !structuralTargetWarmupsConverged(warmups) {
		t.Fatal("two final matching target warm-ups did not converge")
	}
	warmups[4].TaskOutcomes.FingerprintSHA256 = strings.Repeat("5", 64)
	if structuralTargetWarmupsConverged(warmups) {
		t.Fatal("different final target warm-ups converged")
	}
	if structuralTargetWarmupsConverged(warmups[:4]) {
		t.Fatal("an incomplete five-phase sequence converged")
	}
	warmups[3].TaskOutcomes.FingerprintSHA256 = strings.Repeat("3", 64)
	if !structuralTargetWarmupsConverged(warmups[:4]) {
		t.Fatal("two matching adaptive target warm-ups did not converge")
	}
	if !shouldStopAdaptiveCandidateStabilization(true, true, 2, warmups[:4]) {
		t.Fatal("eligible adaptive candidate did not stop after two exact fingerprints")
	}
	if shouldStopAdaptiveCandidateStabilization(false, true, 2, warmups[:4]) ||
		shouldStopAdaptiveCandidateStabilization(true, false, 2, warmups[:4]) ||
		shouldStopAdaptiveCandidateStabilization(true, true, 3, warmups[:4]) {
		t.Fatal("adaptive stabilization changed an ineligible arm or confirmation")
	}
}

func TestDescribeStructuralTaskOutcomeDifferenceIsBoundedAndSpecific(t *testing.T) {
	task := func(path, outcome string) profilediscovery.StructuralTaskObservation {
		return profilediscovery.StructuralTaskObservation{Path: path, Outcome: outcome}
	}
	previous := profilediscovery.StructuralTaskOutcomes{Tasks: []profilediscovery.StructuralTaskObservation{
		task(":alpha", "FROM_CACHE"),
		task(":removed", "EXECUTED"),
		task(":stable", "NO_SOURCE"),
	}}
	current := profilediscovery.StructuralTaskOutcomes{Tasks: []profilediscovery.StructuralTaskObservation{
		task(":added", "EXECUTED"),
		task(":alpha", "EXECUTED"),
		task(":stable", "NO_SOURCE"),
	}}
	got := describeStructuralTaskOutcomeDifference(previous, current)
	want := ":added ABSENT -> EXECUTED; :alpha FROM_CACHE -> EXECUTED; :removed EXECUTED -> ABSENT"
	if got != want {
		t.Fatalf("task difference = %q, want %q", got, want)
	}

	previous.Tasks = previous.Tasks[:0]
	current.Tasks = current.Tasks[:0]
	for index := 0; index < 18; index++ {
		path := fmt.Sprintf(":task-%02d", index)
		previous.Tasks = append(previous.Tasks, task(path, "FROM_CACHE"))
		current.Tasks = append(current.Tasks, task(path, "EXECUTED"))
	}
	got = describeStructuralTaskOutcomeDifference(previous, current)
	if strings.Count(got, " -> ") != 16 || !strings.HasSuffix(got, "; and 2 more") {
		t.Fatalf("bounded task difference = %q", got)
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
