package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	helperModeEnvironment   = "GO_WANT_BUILDOPT_HELPER_PROCESS"
	helperExitEnvironment   = "GO_BUILDOPT_HELPER_EXIT"
	helperStderrEnvironment = "GO_BUILDOPT_HELPER_STDERR"
	passthroughEnvironment  = "WS001_PASSTHROUGH_VALUE"
	expectedUsage           = "usage: buildopt run -- <command> [args...]\n"
)

type helperObservation struct {
	Argv0            string   `json:"argv0"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"workingDirectory"`
	StandardInput    string   `json:"standardInput"`
	EnvironmentValue string   `json:"environmentValue"`
}

func TestBuildoptCLI(t *testing.T) {
	buildoptBinary := buildBuildopt(t)

	t.Run("preserves process contract and nonzero exit", func(t *testing.T) {
		helperBinary, err := os.Executable()
		if err != nil {
			t.Fatalf("resolve helper executable: %v", err)
		}
		workingDirectory := t.TempDir()
		input := "stdin with spaces\nand a second line\n"
		arguments := []string{
			"plain",
			"",
			"two words",
			`double"quote`,
			"'single-quote'",
			"$HOME",
			"*",
			"--",
			"café/東京",
			"line\nbreak",
		}

		commandArguments := []string{
			"run",
			"--",
			helperBinary,
			"-test.run=^TestBuildoptChildHelper$",
			"--",
		}
		commandArguments = append(commandArguments, arguments...)

		command := exec.Command(buildoptBinary, commandArguments...)
		command.Dir = workingDirectory
		command.Env = append(
			os.Environ(),
			helperModeEnvironment+"=1",
			helperExitEnvironment+"=37",
			helperStderrEnvironment+"=child stderr",
			passthroughEnvironment+"=inherited exactly",
		)
		command.Stdin = strings.NewReader(input)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr

		err = command.Run()
		if exitCode := processExitCode(t, err); exitCode != 37 {
			t.Fatalf("exit code = %d, want 37", exitCode)
		}
		if stderr.String() != "child stderr\n" {
			t.Fatalf("stderr = %q, want child marker", stderr.String())
		}

		var observation helperObservation
		if err := json.Unmarshal(stdout.Bytes(), &observation); err != nil {
			t.Fatalf("decode child observation %q: %v", stdout.String(), err)
		}
		if observation.Argv0 != helperBinary {
			t.Errorf("argv[0] = %q, want %q", observation.Argv0, helperBinary)
		}
		if !slices.Equal(observation.Arguments, arguments) {
			t.Errorf("arguments = %#v, want %#v", observation.Arguments, arguments)
		}
		if observation.WorkingDirectory != workingDirectory {
			t.Errorf(
				"working directory = %q, want %q",
				observation.WorkingDirectory,
				workingDirectory,
			)
		}
		if observation.StandardInput != input {
			t.Errorf("stdin = %q, want %q", observation.StandardInput, input)
		}
		if observation.EnvironmentValue != "inherited exactly" {
			t.Errorf(
				"environment value = %q, want inherited value",
				observation.EnvironmentValue,
			)
		}
	})

	t.Run("returns zero for a successful child", func(t *testing.T) {
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
		)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("successful child failed: %v\n%s", err, output)
		}
	})

	t.Run("reports usage errors without starting a child", func(t *testing.T) {
		testCases := []struct {
			name      string
			arguments []string
		}{
			{name: "no arguments"},
			{name: "run without delimiter", arguments: []string{"run", "command"}},
			{name: "missing command", arguments: []string{"run", "--"}},
			{name: "unknown command", arguments: []string{"unknown"}},
			{
				name:      "tokens before delimiter",
				arguments: []string{"run", "extra", "--", "command"},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				command := exec.Command(buildoptBinary, testCase.arguments...)
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				command.Stdout = &stdout
				command.Stderr = &stderr

				err := command.Run()
				if exitCode := processExitCode(t, err); exitCode != 64 {
					t.Fatalf("exit code = %d, want 64", exitCode)
				}
				if stdout.Len() != 0 {
					t.Errorf("stdout = %q, want empty", stdout.String())
				}
				if stderr.String() != expectedUsage {
					t.Errorf("stderr = %q, want %q", stderr.String(), expectedUsage)
				}
			})
		}
	})

	t.Run("prints help", func(t *testing.T) {
		for _, helpArgument := range []string{"help", "--help", "-h"} {
			t.Run(helpArgument, func(t *testing.T) {
				command := exec.Command(buildoptBinary, helpArgument)
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				command.Stdout = &stdout
				command.Stderr = &stderr

				if err := command.Run(); err != nil {
					t.Fatalf("help failed: %v", err)
				}
				if stdout.String() != expectedUsage {
					t.Errorf("stdout = %q, want %q", stdout.String(), expectedUsage)
				}
				if stderr.Len() != 0 {
					t.Errorf("stderr = %q, want empty", stderr.String())
				}
			})
		}
	})

	t.Run("returns command-not-found", func(t *testing.T) {
		missingCommand := filepath.Join(t.TempDir(), "missing-command")
		command := exec.Command(buildoptBinary, "run", "--", missingCommand)
		var stderr bytes.Buffer
		command.Stderr = &stderr

		err := command.Run()
		if exitCode := processExitCode(t, err); exitCode != 127 {
			t.Fatalf("exit code = %d, want 127", exitCode)
		}
		if !strings.Contains(stderr.String(), "cannot execute") ||
			!strings.Contains(stderr.String(), missingCommand) {
			t.Fatalf("unexpected command-not-found diagnostic: %q", stderr.String())
		}
	})

	t.Run("returns cannot-execute", func(t *testing.T) {
		nonExecutableCommand := t.TempDir()
		command := exec.Command(buildoptBinary, "run", "--", nonExecutableCommand)
		var stderr bytes.Buffer
		command.Stderr = &stderr

		err := command.Run()
		if exitCode := processExitCode(t, err); exitCode != 126 {
			t.Fatalf("exit code = %d, want 126", exitCode)
		}
		if !strings.Contains(stderr.String(), "cannot execute") ||
			!strings.Contains(stderr.String(), nonExecutableCommand) {
			t.Fatalf("unexpected cannot-execute diagnostic: %q", stderr.String())
		}
	})
}

func TestBuildoptChildHelper(t *testing.T) {
	if os.Getenv(helperModeEnvironment) != "1" {
		return
	}

	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		_, _ = fmt.Fprintln(os.Stderr, "helper argument separator is missing")
		os.Exit(90)
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read helper stdin: %v\n", err)
		os.Exit(91)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read helper working directory: %v\n", err)
		os.Exit(92)
	}

	observation := helperObservation{
		Argv0:            os.Args[0],
		Arguments:        os.Args[separator+1:],
		WorkingDirectory: workingDirectory,
		StandardInput:    string(input),
		EnvironmentValue: os.Getenv(passthroughEnvironment),
	}
	if err := json.NewEncoder(os.Stdout).Encode(observation); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "encode helper observation: %v\n", err)
		os.Exit(93)
	}
	if marker := os.Getenv(helperStderrEnvironment); marker != "" {
		_, _ = fmt.Fprintln(os.Stderr, marker)
	}

	exitCode, err := strconv.Atoi(os.Getenv(helperExitEnvironment))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parse helper exit code: %v\n", err)
		os.Exit(94)
	}
	os.Exit(exitCode)
}

func buildBuildopt(t *testing.T) string {
	t.Helper()

	repositoryRoot := findRepositoryRoot(t)
	binaryName := "buildopt"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	command := exec.Command(
		"go",
		"build",
		"-buildvcs=false",
		"-trimpath",
		"-o",
		binaryPath,
		"./cmd/buildopt",
	)
	command.Dir = repositoryRoot
	command.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build buildopt: %v\n%s", err, output)
	}
	return binaryPath
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
}

func processExitCode(t *testing.T, err error) int {
	t.Helper()

	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	t.Fatalf("command did not return an exit status: %v", err)
	return -1
}
