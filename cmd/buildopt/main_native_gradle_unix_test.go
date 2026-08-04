//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeGradleReplacesBuildoptProcess(t *testing.T) {
	buildoptBinary := buildBuildopt(t)
	repository := t.TempDir()
	wrapper := filepath.Join(repository, "gradlew")
	wrapperSource := `#!/bin/sh
printf '{"pid":%s,"arguments":[' "$$"
separator=
for argument in "$@"; do
    if [ -n "$separator" ]; then printf ','; fi
    printf '"%s"' "$argument"
    separator=,
done
printf ']}\n'
`
	if err := os.WriteFile(wrapper, []byte(wrapperSource), 0o700); err != nil {
		t.Fatal(err)
	}

	command := exec.Command(buildoptBinary, "gradle", "help")
	command.Dir = repository
	command.Env = append(
		os.Environ(),
		"BUILDOPT_SAFE_CACHE=0",
		"BUILDOPT_GRADLE_INIT_SCRIPT=",
		"BUILDOPT_GRADLE_PLUGIN_JAR=",
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	buildoptPID := command.Process.Pid
	if err := command.Wait(); err != nil {
		t.Fatalf("native Gradle invocation failed: %v\n%s", err, stderr.String())
	}

	var observation struct {
		PID       int      `json:"pid"`
		Arguments []string `json:"arguments"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &observation); err != nil {
		t.Fatalf("decode wrapper observation %q: %v", stdout.String(), err)
	}
	if observation.PID != buildoptPID {
		t.Fatalf("wrapper PID = %d, want BuildOpt PID %d", observation.PID, buildoptPID)
	}
	wantArguments := []string{"--build-cache", "help"}
	if len(observation.Arguments) != len(wantArguments) {
		t.Fatalf("wrapper arguments = %q, want %q", observation.Arguments, wantArguments)
	}
	for index := range wantArguments {
		if observation.Arguments[index] != wantArguments[index] {
			t.Fatalf("wrapper arguments = %q, want %q", observation.Arguments, wantArguments)
		}
	}
}
