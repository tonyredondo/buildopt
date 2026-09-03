package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFakeWrapper(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "gradlew")
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "gradlew.bat")
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestBuildoptwPassthroughPreservesNativeBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake-wrapper fixtures; Windows shapes covered statically and by hosted Native CI")
	}
	dir := t.TempDir()
	writeFakeWrapper(t, dir, "#!/bin/sh\necho \"out:$1:$2\"\necho \"err:$1\" >&2\nexit 0\n")
	stdoutFile := filepath.Join(t.TempDir(), "stdout.txt")
	stderrFile := filepath.Join(t.TempDir(), "stderr.txt")
	stdout, err := os.Create(stdoutFile)
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	stderr, err := os.Create(stderrFile)
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	getenv := func(string) string { return "" }
	code := run([]string{"hello", "world"}, getenv, nil, stdout, stderr, func() (string, error) { return dir, nil })
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	_ = stdout.Close()
	_ = stderr.Close()
	out, _ := os.ReadFile(stdoutFile)
	errOut, _ := os.ReadFile(stderrFile)
	if string(out) != "out:hello:world\n" || string(errOut) != "err:hello\n" {
		t.Fatalf("stdio = %q/%q", out, errOut)
	}
}

func TestBuildoptwPreservesNonZeroExitAndBypass(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixtures")
	}
	dir := t.TempDir()
	writeFakeWrapper(t, dir, "#!/bin/sh\nexit 3\n")
	null, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	defer null.Close()
	getenv := func(string) string { return "" }
	code := run(nil, getenv, nil, null, null, func() (string, error) { return dir, nil })
	if code != 3 {
		t.Fatalf("non-zero exit = %d", code)
	}
	bypassEnv := func(key string) string {
		if key == "BUILDOPT_BYPASS" {
			return "1"
		}
		return ""
	}
	code = run(nil, bypassEnv, nil, null, null, func() (string, error) { return dir, nil })
	if code != 3 {
		t.Fatalf("bypass exit = %d", code)
	}
}

func TestBuildoptwNoWrapperFailsClosed(t *testing.T) {
	null, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	defer null.Close()
	code := run(nil, func(string) string { return "" }, nil, null, null, func() (string, error) { return t.TempDir(), nil })
	if code != 127 {
		t.Fatalf("no wrapper = %d", code)
	}
}

func TestBuildoptwStatusAndRedaction(t *testing.T) {
	dir := t.TempDir()
	writeFakeWrapper(t, dir, "#!/bin/sh\nexit 0\n")
	null, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	defer null.Close()
	outbox := filepath.Join(t.TempDir(), "outbox")
	getenv := func(key string) string {
		switch key {
		case "WCNCP_REPOSITORY_SCOPE":
			return "example/repository"
		case "WCNCP_OUTBOX_DIR":
			return outbox
		case "WCNCP_ENVIRONMENT_CLASS":
			return "LOCAL_FUNCTIONAL"
		default:
			return ""
		}
	}
	code := run([]string{"build", "--token=secret-value"}, getenv, nil, null, null, func() (string, error) { return dir, nil })
	if code != 0 {
		t.Fatalf("observed exit = %d", code)
	}
	entries, err := os.ReadDir(outbox)
	if err != nil {
		t.Fatal(err)
	}
	// runner.id plus one observation item.
	observations := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "obs-") {
			observations++
			raw, _ := os.ReadFile(filepath.Join(outbox, entry.Name()))
			if strings.Contains(string(raw), "secret-value") {
				t.Fatal("secret leaked into outbox")
			}
		}
	}
	if observations != 1 {
		t.Fatalf("outbox observations = %d", observations)
	}
}
