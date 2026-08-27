package stickywrapper

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCLIInitCheckAndUpdate(t *testing.T) {
	resolver := &fakeResolver{
		latest: fixtureRelease("1.2.3", 'a'),
		versions: map[string]Release{
			"1.2.4": fixtureRelease("1.2.4", 'b'),
		},
	}
	root := t.TempDir()
	generator := Generator{Root: root, Resolver: resolver}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := runCLI([]string{
		"init", "--server", "https://buildopt.example.invalid",
		"--project-scope", "example/pilot", "--mode", "observe",
	}, &stdout, &stderr, generator)
	if exit != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "version 1.2.3") {
		t.Fatalf("init exit/stdout/stderr = %d/%q/%q", exit, stdout.String(), stderr.String())
	}
	stdout.Reset()
	exit = runCLI([]string{"check"}, &stdout, &stderr, generator)
	if exit != 0 || !strings.Contains(stdout.String(), "check: OK") {
		t.Fatalf("check exit/stdout/stderr = %d/%q/%q", exit, stdout.String(), stderr.String())
	}
	stdout.Reset()
	exit = runCLI([]string{"update", "--version", "1.2.4"}, &stdout, &stderr, generator)
	if exit != 0 || !strings.Contains(stdout.String(), "1.2.3 -> 1.2.4") {
		t.Fatalf("update exit/stdout/stderr = %d/%q/%q", exit, stdout.String(), stderr.String())
	}
	snapshot, err := generator.Check()
	if err != nil || snapshot.Config.Mode != "observe" || snapshot.Config.CredentialEnv != "BUILDOPT_TOKEN" {
		t.Fatalf("generated config = %#v/%v", snapshot.Config, err)
	}
}

func TestCLIErrorCodesAndUsage(t *testing.T) {
	root := t.TempDir()
	resolver := &fakeResolver{latest: fixtureRelease("1.2.3", 'a'), err: context.DeadlineExceeded}
	generator := Generator{Root: root, Resolver: resolver}
	testCases := []struct {
		name string
		args []string
		want int
	}{
		{name: "empty", want: 64},
		{name: "unknown", args: []string{"unknown"}, want: 64},
		{name: "partial server", args: []string{"init", "--server", "https://example.invalid"}, want: 64},
		{name: "network", args: []string{"init"}, want: 69},
		{name: "missing check", args: []string{"check"}, want: 65},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exit := runCLI(testCase.args, &stdout, &stderr, generator)
			if exit != testCase.want || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("exit/stdout/stderr = %d/%q/%q, want %d/empty/nonempty", exit, stdout.String(), stderr.String(), testCase.want)
			}
		})
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exit := runCLI([]string{"--help"}, &stdout, &stderr, generator); exit != 0 || stdout.String() != Usage || stderr.Len() != 0 {
		t.Fatalf("help exit/stdout/stderr = %d/%q/%q", exit, stdout.String(), stderr.String())
	}
}

func TestCLIStatusAndExplainAreReadOnlyManagementCommands(t *testing.T) {
	root := t.TempDir()
	generator := Generator{Root: root, Resolver: &fakeResolver{latest: fixtureRelease("1.2.3", 'a')}}
	if _, err := generator.Init(context.Background(), configuredFixture()); err != nil {
		t.Fatal(err)
	}
	before := captureTree(t, root)
	for _, testCase := range []struct {
		name string
		args []string
		want string
	}{
		{name: "status json", args: []string{"status", "--root", root, "--json"}, want: `"reportType": "STATUS"`},
		{name: "explain human", args: []string{"explain", "--root", root}, want: "Decision: native"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if exit := runCLI(testCase.args, &stdout, &stderr, generator); exit != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), testCase.want) {
				t.Fatalf("exit/stdout/stderr = %d/%q/%q", exit, stdout.String(), stderr.String())
			}
		})
	}
	if after := captureTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("status/explain modified repository: before=%#v after=%#v", before, after)
	}
}

func TestCLIUpdateRefusesDriftAndDowngrade(t *testing.T) {
	resolver := &fakeResolver{
		latest: fixtureRelease("2.0.0", 'b'),
		versions: map[string]Release{
			"1.9.0": fixtureRelease("1.9.0", 'a'),
			"2.1.0": fixtureRelease("2.1.0", 'c'),
		},
	}
	root := t.TempDir()
	generator := Generator{Root: root, Resolver: resolver}
	if _, err := generator.Init(context.Background(), DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"update", "--version", "1.9.0"},
		{"update", "--version", "1.9.0", "--allow-downgrade", "--allow-downgrade"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exit := runCLI(args, &stdout, &stderr, generator)
		if exit != 65 && exit != 64 {
			t.Fatalf("args %v exit = %d, stderr %q", args, exit, stderr.String())
		}
	}
	path := filepath.Join(root, configPath)
	if err := os.WriteFile(path, appendMarker(mustRead(t, path)), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exit := runCLI([]string{"update", "--version", "2.1.0"}, &stdout, &stderr, generator); exit != 65 {
		t.Fatalf("drifted update exit = %d, stderr %q", exit, stderr.String())
	}
}
