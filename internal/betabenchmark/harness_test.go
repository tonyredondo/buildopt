package betabenchmark

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
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
