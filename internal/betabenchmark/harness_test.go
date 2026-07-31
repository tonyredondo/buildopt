package betabenchmark

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRunSmokeExercisesEveryPhaseAndClientStratum(t *testing.T) {
	root := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(
		workingDirectory,
		"..",
		"..",
		"benchmarks",
		"beta-v1.yaml",
	)
	manifestPath, err = filepath.Abs(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	resultPath, err := RunSmoke(
		context.Background(),
		manifestPath,
		filepath.Join(root, "state"),
		filepath.Join(root, "output"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateResult(manifestPath, resultPath); err != nil {
		t.Fatal(err)
	}

	observationsPath := filepath.Join(
		filepath.Dir(resultPath),
		observationsFilename,
	)
	observations, err := os.ReadFile(observationsPath)
	if err != nil {
		t.Fatal(err)
	}
	observations[0] ^= 1
	if err := os.WriteFile(observationsPath, observations, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateResult(manifestPath, resultPath); err == nil {
		t.Fatal("tampered observations passed validation")
	}
}

func TestRunDiskFaultsProducesBoundTamperEvidentEvidence(t *testing.T) {
	root := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(
		workingDirectory,
		"..",
		"..",
		"benchmarks",
		"beta-v1.yaml",
	)
	manifestPath, err = filepath.Abs(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	resultPath, err := RunDiskFaults(
		context.Background(),
		manifestPath,
		filepath.Join(root, "state"),
		filepath.Join(root, "output"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiskFaultResult(manifestPath, resultPath); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	unknownResult := bytes.Replace(
		result,
		[]byte("{\n"),
		[]byte("{\n  \"unknown\": true,\n"),
		1,
	)
	if err := os.WriteFile(resultPath, unknownResult, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiskFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("unknown disk-fault result field passed validation")
	}
	trailingResult := append(append([]byte{}, result...), []byte("{}\n")...)
	if err := os.WriteFile(resultPath, trailingResult, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiskFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("trailing disk-fault result JSON passed validation")
	}
	if err := os.WriteFile(resultPath, result, 0o600); err != nil {
		t.Fatal(err)
	}

	observationsPath := filepath.Join(
		filepath.Dir(resultPath),
		diskFaultRawFilename,
	)
	observations, err := os.ReadFile(observationsPath)
	if err != nil {
		t.Fatal(err)
	}
	observations[0] ^= 1
	if err := os.WriteFile(observationsPath, observations, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiskFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("tampered disk-fault observations passed validation")
	}
}

func TestRunSharedFaultsProducesBoundTamperEvidentEvidence(t *testing.T) {
	root := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(
		workingDirectory,
		"..",
		"..",
		"benchmarks",
		"beta-v1.yaml",
	)
	manifestPath, err = filepath.Abs(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	resultPath, err := RunSharedFaults(
		context.Background(),
		manifestPath,
		filepath.Join(root, "state"),
		filepath.Join(root, "output"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSharedFaultResult(manifestPath, resultPath); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	unknownResult := bytes.Replace(
		result,
		[]byte("{\n"),
		[]byte("{\n  \"unknown\": true,\n"),
		1,
	)
	if err := os.WriteFile(resultPath, unknownResult, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSharedFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("unknown Shared-fault result field passed validation")
	}
	if err := os.WriteFile(resultPath, result, 0o600); err != nil {
		t.Fatal(err)
	}
	observationsPath := filepath.Join(
		filepath.Dir(resultPath),
		sharedFaultRawFilename,
	)
	observations, err := os.ReadFile(observationsPath)
	if err != nil {
		t.Fatal(err)
	}
	observations[len(observations)-2] ^= 1
	if err := os.WriteFile(observationsPath, observations, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSharedFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("tampered Shared-fault observations passed validation")
	}
}

func TestRunSystemFaultsProducesBoundTamperEvidentEvidence(t *testing.T) {
	root := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join(workingDirectory, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(repositoryRoot, "benchmarks", "beta-v1.yaml")
	executables := SystemFaultExecutables{
		BuildOpt: buildSystemFaultExecutable(
			t,
			repositoryRoot,
			root,
			"buildopt",
			"./cmd/buildopt",
		),
		Server: buildSystemFaultExecutable(
			t,
			repositoryRoot,
			root,
			"buildopt-server",
			"./cmd/buildopt-server",
		),
	}
	resultPath, err := RunSystemFaults(
		context.Background(),
		manifestPath,
		filepath.Join(root, "state"),
		filepath.Join(root, "output"),
		executables,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSystemFaultResult(manifestPath, resultPath); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	unknownResult := bytes.Replace(
		result,
		[]byte("{\n"),
		[]byte("{\n  \"unknown\": true,\n"),
		1,
	)
	if err := os.WriteFile(resultPath, unknownResult, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSystemFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("unknown system-fault result field passed validation")
	}
	if err := os.WriteFile(resultPath, result, 0o600); err != nil {
		t.Fatal(err)
	}
	observationsPath := filepath.Join(
		filepath.Dir(resultPath),
		systemFaultRawFilename,
	)
	observations, err := os.ReadFile(observationsPath)
	if err != nil {
		t.Fatal(err)
	}
	observations[len(observations)-2] ^= 1
	if err := os.WriteFile(observationsPath, observations, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSystemFaultResult(manifestPath, resultPath); err == nil {
		t.Fatal("tampered system-fault observations passed validation")
	}
}

func TestRunSustainedTrialUsesManagedGatewayAndRejectsTampering(
	t *testing.T,
) {
	root := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join(workingDirectory, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(repositoryRoot, "benchmarks", "beta-v1.yaml")
	buildoptExecutable := buildSystemFaultExecutable(
		t,
		repositoryRoot,
		root,
		"buildopt",
		"./cmd/buildopt",
	)
	resultPath, err := RunSustainedTrial(
		context.Background(),
		manifestPath,
		filepath.Join(root, "state"),
		filepath.Join(root, "output"),
		buildoptExecutable,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSustainedTrial(manifestPath, resultPath); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSustainedResult(manifestPath, resultPath); err == nil {
		t.Fatal("trial passed one-hour sustained validation")
	}
	result, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	unknownResult := bytes.Replace(
		result,
		[]byte("{\n"),
		[]byte("{\n  \"unknown\": true,\n"),
		1,
	)
	if err := os.WriteFile(resultPath, unknownResult, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSustainedTrial(manifestPath, resultPath); err == nil {
		t.Fatal("unknown sustained result field passed validation")
	}
	if err := os.WriteFile(resultPath, result, 0o600); err != nil {
		t.Fatal(err)
	}
	observationsPath := filepath.Join(
		filepath.Dir(resultPath),
		observationsFilename,
	)
	observations, err := os.ReadFile(observationsPath)
	if err != nil {
		t.Fatal(err)
	}
	observations[len(observations)-2] ^= 1
	if err := os.WriteFile(observationsPath, observations, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSustainedTrial(manifestPath, resultPath); err == nil {
		t.Fatal("tampered sustained observations passed validation")
	}
}

func TestGoldenRunnerCgroupRequiresExactLimits(t *testing.T) {
	if !goldenRunnerCgroup(cgroupIdentity{
		CPUQuota:    "400000 100000",
		MemoryLimit: "17179869184",
	}) {
		t.Fatal("exact golden cgroup was rejected")
	}
	if goldenRunnerCgroup(cgroupIdentity{
		CPUQuota:    "1200000 100000",
		MemoryLimit: "17179869184",
	}) {
		t.Fatal("non-golden CPU quota passed")
	}
}

func TestSustainedAuthorityCoversQualificationWindow(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	fixture, err := newSystemAuthorityFixtureWithDurations(
		context.Background(),
		t.TempDir(),
		"sha256:"+string(bytes.Repeat([]byte{'1'}, 64)),
		now,
		941,
		false,
		false,
		sustainedAuthorityLifetime,
		sustainedAuthorityLifetime,
		sustainedAuthorityLifetime,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fixture.verified.ExpiresAt(), now.Add(
		sustainedAuthorityLifetime,
	); !got.Equal(want) {
		t.Fatalf("sustained authority expiry = %s, want %s", got, want)
	}
}

func TestSustainedLatencyTargetsSeparateReadyAndMaterialization(
	t *testing.T,
) {
	targets := summarizeSustainedLatencyTargets(
		[]rawObservation{
			{
				Clients:     1,
				SizeBytes:   1 << 20,
				ExpectedHit: false,
				ReadyNs:     (40 * time.Millisecond).Nanoseconds(),
				DurationNs:  (41 * time.Millisecond).Nanoseconds(),
			},
			{
				Clients:     1,
				SizeBytes:   1 << 20,
				ExpectedHit: true,
				ReadyNs:     (100 * time.Millisecond).Nanoseconds(),
				DurationNs:  (200 * time.Millisecond).Nanoseconds(),
			},
		},
		[]int{1},
	)
	if len(targets) != 3 {
		t.Fatalf("latency target count = %d, want 3", len(targets))
	}
	expected := []struct {
		metric  string
		p95     time.Duration
		maximum time.Duration
	}{
		{"GATEWAY_MISS", 40 * time.Millisecond, 50 * time.Millisecond},
		{
			"VERIFIED_HIT_READY",
			100 * time.Millisecond,
			150 * time.Millisecond,
		},
		{
			"DOWNSTREAM_MATERIALIZATION",
			100 * time.Millisecond,
			155 * time.Millisecond,
		},
	}
	for index, want := range expected {
		got := targets[index]
		if got.Metric != want.metric ||
			got.P95Ns != want.p95.Nanoseconds() ||
			got.MaximumP95Ns != want.maximum.Nanoseconds() ||
			got.Status != "PASSED" {
			t.Fatalf("latency target[%d] = %+v, want %+v", index, got, want)
		}
	}
}

func buildSystemFaultExecutable(
	t *testing.T,
	repositoryRoot string,
	outputDirectory string,
	name string,
	packagePath string,
) string {
	t.Helper()
	path := filepath.Join(outputDirectory, name)
	command := exec.Command("go", "build", "-o", path, packagePath)
	command.Dir = repositoryRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", packagePath, err, output)
	}
	return path
}

func TestDeterministicReaderIsStableAndSizeBounded(t *testing.T) {
	read := func(seed int64, index int, size int64) []byte {
		t.Helper()
		content := make([]byte, size)
		reader := newDeterministicReader(seed, index, size)
		offset := 0
		for offset < len(content) {
			count, readErr := reader.Read(content[offset:])
			offset += count
			if readErr != nil {
				t.Fatalf("read deterministic bytes: %v", readErr)
			}
		}
		return content
	}
	first := read(17, 3, 4097)
	second := read(17, 3, 4097)
	changed := read(17, 4, 4097)
	if !bytes.Equal(first, second) {
		t.Fatal("deterministic reader changed identical bytes")
	}
	if bytes.Equal(first, changed) {
		t.Fatal("deterministic reader ignored object identity")
	}
}
