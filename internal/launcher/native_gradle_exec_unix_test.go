//go:build !windows

package launcher

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestReplaceWithNativeGradleProcess(t *testing.T) {
	wrapper := writeNativeGradleTestWrapper(t)
	t.Setenv("BUILDOPT_TEST_RETAINED", "yes")
	t.Setenv(pluginSocketEnvironment, "must-not-leak")

	wantErr := errors.New("exec stopped for test")
	previous := nativeGradleExec
	t.Cleanup(func() { nativeGradleExec = previous })
	var gotPath string
	var gotArgs, gotEnvironment []string
	nativeGradleExec = func(path string, args, environment []string) error {
		gotPath = path
		gotArgs = append([]string(nil), args...)
		gotEnvironment = append([]string(nil), environment...)
		return wantErr
	}

	err := replaceWithNativeGradleProcess(
		[]string{wrapper, "--build-cache", "help"},
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("replace error = %v, want %v", err, wantErr)
	}
	if gotPath != wrapper {
		t.Fatalf("exec path = %q, want %q", gotPath, wrapper)
	}
	wantArgs := []string{wrapper, "--build-cache", "help"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("exec args = %q, want %q", gotArgs, wantArgs)
	}
	if !environmentContains(gotEnvironment, "BUILDOPT_TEST_RETAINED=yes") {
		t.Fatalf("ordinary environment was not retained: %q", gotEnvironment)
	}
	if environmentHasPrefix(gotEnvironment, pluginSocketEnvironment+"=") {
		t.Fatalf("reserved environment leaked: %q", gotEnvironment)
	}
}

func TestNativeGradleProcessReplacementRequiresRealStandardStreams(t *testing.T) {
	if !nativeGradleProcessReplacementSupported(os.Stdin, os.Stdout, os.Stderr) {
		t.Fatal("real standard streams did not enable process replacement")
	}
	if nativeGradleProcessReplacementSupported(strings.NewReader(""), os.Stdout, os.Stderr) {
		t.Fatal("test stream enabled process replacement")
	}
}

func writeNativeGradleTestWrapper(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/gradlew"
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func environmentContains(environment []string, want string) bool {
	for _, entry := range environment {
		if entry == want {
			return true
		}
	}
	return false
}

func environmentHasPrefix(environment []string, prefix string) bool {
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}
