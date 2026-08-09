package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	helperModeEnvironment          = "GO_WANT_BUILDOPT_HELPER_PROCESS"
	helperExitEnvironment          = "GO_BUILDOPT_HELPER_EXIT"
	helperStderrEnvironment        = "GO_BUILDOPT_HELPER_STDERR"
	passthroughEnvironment         = "WS001_PASSTHROUGH_VALUE"
	bypassEnvironment              = "BUILDOPT_BYPASS"
	pluginAttemptEnvironment       = "BUILDOPT_PLUGIN_ATTEMPT_ID"
	pluginSocketEnvironment        = "BUILDOPT_PLUGIN_EVENT_SOCKET"
	pluginTokenEnvironment         = "BUILDOPT_PLUGIN_EVENT_TOKEN"
	gatewayURLEnvironment          = "BUILDOPT_GATEWAY_URL"
	gatewayUserEnvironment         = "BUILDOPT_GATEWAY_USERNAME"
	gatewayPassEnvironment         = "BUILDOPT_GATEWAY_PASSWORD"
	gatewayGenEnvironment          = "BUILDOPT_GATEWAY_CONNECTION_GENERATION"
	serverURLEnvironment           = "BUILDOPT_SERVER_URL"
	serverTokenEnvironment         = "BUILDOPT_SERVER_INGEST_TOKEN"
	buildSessionContextEnvironment = "BUILDOPT_BUILD_SESSION_CONTEXT"
	managedStateRootEnvironment    = "BUILDOPT_GATEWAY_STATE_ROOT"
	managedRunnerSlotEnvironment   = "BUILDOPT_RUNNER_SLOT"
	managedIdleTimeoutEnvironment  = "BUILDOPT_GATEWAY_IDLE_TIMEOUT"
	managedL1StateRootEnvironment  = "BUILDOPT_L1_STATE_ROOT"
	managedL1TenantEnvironment     = "BUILDOPT_L1_TENANT_ID"
	managedL1RepositoryEnvironment = "BUILDOPT_L1_REPOSITORY_ID"
	managedL1TrustEnvironment      = "BUILDOPT_L1_TRUST_DOMAIN"
	managedL1CompatEnvironment     = "BUILDOPT_L1_COMPATIBILITY_CLASS"
	managedL1InputGenEnvironment   = "BUILDOPT_L1_SECURITY_GENERATION"
	managedL1WriterEnvironment     = "BUILDOPT_L1_L2_WRITE_AUTHORIZED"
	managedL1DirectoryEnvironment  = "BUILDOPT_MANAGED_L1_DIRECTORY"
	managedL1ModeEnvironment       = "BUILDOPT_MANAGED_L1_MODE"
	managedL1OutputGenEnvironment  = "BUILDOPT_MANAGED_L1_SECURITY_GENERATION"
	managedL1RetentionEnvironment  = "BUILDOPT_MANAGED_L1_RETENTION_DAYS"
	gatewayReadyPath               = "/_buildopt/ready"
	gatewayGenerationHeader        = "BuildOpt-Gateway-Connection-Generation"
	expectedUsage                  = "usage: buildopt run -- <command> [args...]\n       buildopt gradle [gradle args...]\n       buildopt poc --changes-file PATH [options]\n       buildopt impact --repository-id OWNER/REPO --changes-file PATH [options]\n       buildopt profile analyze [options]\n       buildopt profile qualify [options]\n       buildopt profile evaluate [options]\n       buildopt profile discover [options]\n       buildopt doctor\n"
)

type helperObservation struct {
	Argv0                      string   `json:"argv0"`
	Arguments                  []string `json:"arguments"`
	WorkingDirectory           string   `json:"workingDirectory"`
	StandardInput              string   `json:"standardInput"`
	EnvironmentValue           string   `json:"environmentValue"`
	PluginAttemptID            string   `json:"pluginAttemptId"`
	PluginEventSocket          string   `json:"pluginEventSocket"`
	PluginTokenLength          int      `json:"pluginTokenLength"`
	GatewayURL                 string   `json:"gatewayUrl"`
	GatewayGeneration          string   `json:"gatewayGeneration"`
	GatewayReady               int      `json:"gatewayReady"`
	GatewayRejected            int      `json:"gatewayRejected"`
	ReadyGeneration            string   `json:"readyGeneration"`
	BypassPresent              bool     `json:"bypassPresent"`
	ServerURLPresent           bool     `json:"serverUrlPresent"`
	ServerTokenPresent         bool     `json:"serverTokenPresent"`
	BuildSessionContextPresent bool     `json:"buildSessionContextPresent"`
	ManagedL1Directory         string   `json:"managedL1Directory"`
	ManagedL1Mode              string   `json:"managedL1Mode"`
	ManagedL1Generation        string   `json:"managedL1Generation"`
	ManagedL1Retention         string   `json:"managedL1Retention"`
}

func TestBuildoptCLI(t *testing.T) {
	t.Setenv(bypassEnvironment, "")
	t.Setenv(serverURLEnvironment, "")
	t.Setenv(serverTokenEnvironment, "")
	t.Setenv(buildSessionContextEnvironment, "")
	clearManagedGatewayEnvironment(t)
	clearManagedL1Environment(t)

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
			pluginAttemptEnvironment+"=untrusted-parent-attempt",
			pluginSocketEnvironment+"=/tmp/untrusted-parent.sock",
			pluginTokenEnvironment+"=parent-token",
			gatewayURLEnvironment+"=http://127.0.0.1:1",
			gatewayUserEnvironment+"=parent-user",
			gatewayPassEnvironment+"=parent-password",
			gatewayGenEnvironment+"=parent-generation",
			buildSessionContextEnvironment+"=untrusted-parent-context",
			managedL1DirectoryEnvironment+"=/tmp/untrusted-l1",
			managedL1ModeEnvironment+"=READ_WRITE",
			managedL1OutputGenEnvironment+"=999",
			managedL1RetentionEnvironment+"=999",
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
		if observation.PluginAttemptID == "" ||
			observation.PluginAttemptID == "untrusted-parent-attempt" {
			t.Errorf(
				"plugin attempt ID = %q, want fresh invocation value",
				observation.PluginAttemptID,
			)
		}
		if observation.PluginEventSocket == "" ||
			observation.PluginEventSocket == "/tmp/untrusted-parent.sock" {
			t.Errorf(
				"plugin event socket = %q, want fresh invocation value",
				observation.PluginEventSocket,
			)
		}
		if _, err := os.Stat(observation.PluginEventSocket); !errors.Is(err, os.ErrNotExist) {
			t.Errorf(
				"plugin event socket remains after child exit: %v",
				err,
			)
		}
		if observation.PluginTokenLength != 43 {
			t.Errorf(
				"encoded plugin token length = %d, want 43",
				observation.PluginTokenLength,
			)
		}
		if !strings.HasPrefix(
			observation.GatewayURL,
			"http://127.0.0.1:",
		) || observation.GatewayURL == "http://127.0.0.1:1" {
			t.Errorf(
				"gateway URL = %q, want fresh loopback endpoint",
				observation.GatewayURL,
			)
		}
		if observation.GatewayGeneration == "" ||
			observation.GatewayGeneration == "parent-generation" ||
			observation.ReadyGeneration != observation.GatewayGeneration {
			t.Errorf(
				"gateway generation = %q/%q, want matching fresh value",
				observation.GatewayGeneration,
				observation.ReadyGeneration,
			)
		}
		if observation.GatewayReady != http.StatusNoContent ||
			observation.GatewayRejected != http.StatusUnauthorized {
			t.Errorf(
				"gateway statuses = %d/%d, want 204/401",
				observation.GatewayReady,
				observation.GatewayRejected,
			)
		}
		if observation.BuildSessionContextPresent {
			t.Error("BUILD_SESSION export context reached the child")
		}
		if observation.ManagedL1Directory != "" ||
			observation.ManagedL1Mode != "" ||
			observation.ManagedL1Generation != "" ||
			observation.ManagedL1Retention != "" {
			t.Errorf(
				"unconfigured child received managed L1 context: %+v",
				observation,
			)
		}
		if response, err := gatewayTestClient().Get(
			observation.GatewayURL + gatewayReadyPath,
		); err == nil {
			_ = response.Body.Close()
			t.Error("local gateway remains reachable after child exit")
		}
	})

	t.Run("bypass preserves the child and skips all product integration", func(t *testing.T) {
		helperBinary, err := os.Executable()
		if err != nil {
			t.Fatalf("resolve helper executable: %v", err)
		}
		workingDirectory := t.TempDir()
		input := "bypass stdin\n"
		arguments := []string{"", "two words", "*", "$HOME"}
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
			helperExitEnvironment+"=38",
			helperStderrEnvironment+"=bypass child stderr",
			passthroughEnvironment+"=still inherited",
			bypassEnvironment+"=1",
			pluginAttemptEnvironment+"=untrusted-parent-attempt",
			pluginSocketEnvironment+"=/tmp/untrusted-parent.sock",
			pluginTokenEnvironment+"=parent-token",
			gatewayURLEnvironment+"=http://127.0.0.1:1",
			gatewayUserEnvironment+"=parent-user",
			gatewayPassEnvironment+"=parent-password",
			gatewayGenEnvironment+"=parent-generation",
			serverURLEnvironment+"=https://control-plane.invalid",
			serverTokenEnvironment+"=parent-server-token",
			buildSessionContextEnvironment+"={invalid-json",
			managedL1StateRootEnvironment+"=/tmp/untrusted-l1-state",
			managedL1TenantEnvironment+"=tenant-7",
			managedL1RepositoryEnvironment+"=tonyredondo/buildopt",
			managedL1TrustEnvironment+"=private-beta",
			managedL1CompatEnvironment+"=gradle-9.6-java-17-linux-amd64",
			managedL1InputGenEnvironment+"=42",
			managedL1WriterEnvironment+"=0",
			managedL1DirectoryEnvironment+"=/tmp/untrusted-l1",
			managedL1ModeEnvironment+"=READ_WRITE",
			managedL1OutputGenEnvironment+"=999",
			managedL1RetentionEnvironment+"=999",
		)
		command.Stdin = strings.NewReader(input)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr

		err = command.Run()
		if exitCode := processExitCode(t, err); exitCode != 38 {
			t.Fatalf("exit code = %d, want 38", exitCode)
		}
		if stderr.String() != "bypass child stderr\n" {
			t.Fatalf(
				"stderr = %q, want only the child marker",
				stderr.String(),
			)
		}

		var observation helperObservation
		if err := json.Unmarshal(stdout.Bytes(), &observation); err != nil {
			t.Fatalf("decode child observation %q: %v", stdout.String(), err)
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
		if observation.EnvironmentValue != "still inherited" {
			t.Errorf(
				"environment value = %q, want inherited value",
				observation.EnvironmentValue,
			)
		}
		if observation.BypassPresent ||
			observation.PluginAttemptID != "" ||
			observation.PluginEventSocket != "" ||
			observation.PluginTokenLength != 0 ||
			observation.GatewayURL != "" ||
			observation.GatewayGeneration != "" ||
			observation.GatewayReady != 0 ||
			observation.GatewayRejected != 0 ||
			observation.ReadyGeneration != "" ||
			observation.ServerURLPresent ||
			observation.ServerTokenPresent ||
			observation.BuildSessionContextPresent ||
			observation.ManagedL1Directory != "" ||
			observation.ManagedL1Mode != "" ||
			observation.ManagedL1Generation != "" ||
			observation.ManagedL1Retention != "" {
			t.Errorf(
				"bypassed child received product integration context: %+v",
				observation,
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

func clearManagedGatewayEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv(managedStateRootEnvironment, "")
	t.Setenv(managedRunnerSlotEnvironment, "")
	t.Setenv(managedIdleTimeoutEnvironment, "")
}

func clearManagedL1Environment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		managedL1StateRootEnvironment,
		managedL1TenantEnvironment,
		managedL1RepositoryEnvironment,
		managedL1TrustEnvironment,
		managedL1CompatEnvironment,
		managedL1InputGenEnvironment,
		managedL1WriterEnvironment,
		managedL1DirectoryEnvironment,
		managedL1ModeEnvironment,
		managedL1OutputGenEnvironment,
		managedL1RetentionEnvironment,
	} {
		t.Setenv(key, "")
	}
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
	gatewayReady, gatewayRejected, readyGeneration, err :=
		observeLocalGateway()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "probe helper gateway: %v\n", err)
		os.Exit(93)
	}

	observation := helperObservation{
		Argv0:              os.Args[0],
		Arguments:          os.Args[separator+1:],
		WorkingDirectory:   workingDirectory,
		StandardInput:      string(input),
		EnvironmentValue:   os.Getenv(passthroughEnvironment),
		PluginAttemptID:    os.Getenv(pluginAttemptEnvironment),
		PluginEventSocket:  os.Getenv(pluginSocketEnvironment),
		PluginTokenLength:  len(os.Getenv(pluginTokenEnvironment)),
		GatewayURL:         os.Getenv(gatewayURLEnvironment),
		GatewayGeneration:  os.Getenv(gatewayGenEnvironment),
		GatewayReady:       gatewayReady,
		GatewayRejected:    gatewayRejected,
		ReadyGeneration:    readyGeneration,
		BypassPresent:      os.Getenv(bypassEnvironment) != "",
		ServerURLPresent:   os.Getenv(serverURLEnvironment) != "",
		ServerTokenPresent: os.Getenv(serverTokenEnvironment) != "",
		BuildSessionContextPresent: os.Getenv(
			buildSessionContextEnvironment,
		) != "",
		ManagedL1Directory:  os.Getenv(managedL1DirectoryEnvironment),
		ManagedL1Mode:       os.Getenv(managedL1ModeEnvironment),
		ManagedL1Generation: os.Getenv(managedL1OutputGenEnvironment),
		ManagedL1Retention:  os.Getenv(managedL1RetentionEnvironment),
	}
	if err := json.NewEncoder(os.Stdout).Encode(observation); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "encode helper observation: %v\n", err)
		os.Exit(94)
	}
	if marker := os.Getenv(helperStderrEnvironment); marker != "" {
		_, _ = fmt.Fprintln(os.Stderr, marker)
	}

	exitCode, err := strconv.Atoi(os.Getenv(helperExitEnvironment))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parse helper exit code: %v\n", err)
		os.Exit(95)
	}
	os.Exit(exitCode)
}

func observeLocalGateway() (int, int, string, error) {
	endpoint := os.Getenv(gatewayURLEnvironment)
	if endpoint == "" {
		return 0, 0, "", nil
	}
	username := os.Getenv(gatewayUserEnvironment)
	password := os.Getenv(gatewayPassEnvironment)

	readyRequest, err := http.NewRequest(
		http.MethodGet,
		endpoint+gatewayReadyPath,
		nil,
	)
	if err != nil {
		return 0, 0, "", err
	}
	readyRequest.SetBasicAuth(username, password)
	readyResponse, err := gatewayTestClient().Do(readyRequest)
	if err != nil {
		return 0, 0, "", err
	}
	_ = readyResponse.Body.Close()

	rejectedRequest, err := http.NewRequest(
		http.MethodGet,
		endpoint+gatewayReadyPath,
		nil,
	)
	if err != nil {
		return 0, 0, "", err
	}
	rejectedRequest.SetBasicAuth(username, password+"-wrong")
	rejectedResponse, err := gatewayTestClient().Do(rejectedRequest)
	if err != nil {
		return 0, 0, "", err
	}
	_ = rejectedResponse.Body.Close()

	return readyResponse.StatusCode,
		rejectedResponse.StatusCode,
		readyResponse.Header.Get(gatewayGenerationHeader),
		nil
}

func gatewayTestClient() *http.Client {
	return &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			Proxy:             nil,
			DisableKeepAlives: true,
		},
	}
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
