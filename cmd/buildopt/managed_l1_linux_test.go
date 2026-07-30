//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildoptManagedL1ContextAndFallback(t *testing.T) {
	buildoptBinary := buildBuildopt(t)
	stateRoot := filepath.Join(t.TempDir(), "state")

	readWrite, stderr := runManagedL1Helper(
		t,
		buildoptBinary,
		stateRoot,
		"0",
	)
	if stderr != "" {
		t.Fatalf("read/write managed L1 stderr = %q", stderr)
	}
	if readWrite.ManagedL1Mode != "READ_WRITE" ||
		readWrite.ManagedL1Generation != "42" ||
		readWrite.ManagedL1Retention != "7" ||
		!filepath.IsAbs(readWrite.ManagedL1Directory) ||
		!strings.HasSuffix(
			readWrite.ManagedL1Directory,
			filepath.Join("generation-42", "cache"),
		) {
		t.Fatalf("unexpected read/write managed L1 context: %+v", readWrite)
	}
	for _, rawIdentity := range []string{
		"tenant-7",
		"tonyredondo/buildopt",
		"private-beta",
		"gradle-9.6-java-17-linux-amd64",
	} {
		if strings.Contains(readWrite.ManagedL1Directory, rawIdentity) {
			t.Fatalf(
				"managed L1 directory exposed raw identity %q: %s",
				rawIdentity,
				readWrite.ManagedL1Directory,
			)
		}
	}
	for path := readWrite.ManagedL1Directory; path != stateRoot; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("managed L1 directory %s = %v/%v", path, info, err)
		}
	}

	writer, stderr := runManagedL1Helper(
		t,
		buildoptBinary,
		filepath.Join(t.TempDir(), "unused-state"),
		"1",
	)
	if stderr != "" {
		t.Fatalf("L2-writer managed L1 stderr = %q", stderr)
	}
	if writer.ManagedL1Mode != "DISABLED_L2_WRITER" ||
		writer.ManagedL1Directory != "" ||
		writer.ManagedL1Generation != "42" ||
		writer.ManagedL1Retention != "7" {
		t.Fatalf("L2 writer retained managed L1 access: %+v", writer)
	}

	publicRoot := filepath.Join(t.TempDir(), "public-state")
	if err := os.Mkdir(publicRoot, 0o755); err != nil {
		t.Fatalf("create public L1 state: %v", err)
	}
	fallback, stderr := runManagedL1Helper(
		t,
		buildoptBinary,
		publicRoot,
		"0",
	)
	if fallback.ManagedL1Mode != "" ||
		fallback.ManagedL1Directory != "" ||
		!strings.Contains(stderr, "managed L1 unavailable") ||
		!strings.Contains(stderr, "mode 0700") {
		t.Fatalf(
			"unsafe L1 did not preserve baseline: observation=%+v stderr=%q",
			fallback,
			stderr,
		)
	}
}

func runManagedL1Helper(
	t *testing.T,
	buildoptBinary string,
	stateRoot string,
	l2Writer string,
) (helperObservation, string) {
	t.Helper()
	helperBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve helper executable: %v", err)
	}
	command := exec.Command(
		buildoptBinary,
		"run",
		"--",
		helperBinary,
		"-test.run=^TestBuildoptChildHelper$",
		"--",
	)
	command.Env = append(
		os.Environ(),
		helperModeEnvironment+"=1",
		helperExitEnvironment+"=0",
		managedL1StateRootEnvironment+"="+stateRoot,
		managedL1TenantEnvironment+"=tenant-7",
		managedL1RepositoryEnvironment+"=tonyredondo/buildopt",
		managedL1TrustEnvironment+"=private-beta",
		managedL1CompatEnvironment+"=gradle-9.6-java-17-linux-amd64",
		managedL1InputGenEnvironment+"=42",
		managedL1WriterEnvironment+"="+l2Writer,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("managed L1 helper failed: %v\n%s", err, stderr.String())
	}
	var observation helperObservation
	if err := json.Unmarshal(stdout.Bytes(), &observation); err != nil {
		t.Fatalf("decode managed L1 observation %q: %v", stdout.String(), err)
	}
	return observation, stderr.String()
}
