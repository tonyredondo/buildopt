//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	leaderReadyEnvironment        = "BUILDOPT_TEST_LEADER_READY"
	descendantReadyEnvironment    = "BUILDOPT_TEST_DESCENDANT_READY"
	leaderSignalEnvironment       = "BUILDOPT_TEST_LEADER_SIGNAL"
	descendantSignalEnvironment   = "BUILDOPT_TEST_DESCENDANT_SIGNAL"
	cleanupCompleteEnvironment    = "BUILDOPT_TEST_CLEANUP_COMPLETE"
	leaderExitCodeEnvironment     = "BUILDOPT_TEST_LEADER_EXIT_CODE"
	leaderCleanupDelayEnvironment = "BUILDOPT_TEST_CLEANUP_DELAY"
)

type signalProcessObservation struct {
	PID               int    `json:"pid"`
	PGID              int    `json:"pgid"`
	ParentPGID        int    `json:"parentPgid"`
	PluginAttemptID   string `json:"pluginAttemptId"`
	PluginEventSocket string `json:"pluginEventSocket"`
	GatewayURL        string `json:"gatewayUrl"`
}

func TestBuildoptForwardsSignalsToChildProcessGroup(t *testing.T) {
	t.Setenv(bypassEnvironment, "")
	t.Setenv(serverURLEnvironment, "")
	t.Setenv(serverTokenEnvironment, "")
	t.Setenv(buildSessionContextEnvironment, "")
	clearManagedGatewayEnvironment(t)

	buildoptBinary := buildBuildopt(t)
	signalHelper := buildSignalHelper(t)

	testCases := []struct {
		name         string
		signal       syscall.Signal
		childExit    int
		cleanupDelay time.Duration
		bypass       bool
	}{
		{
			name:         "SIGINT",
			signal:       syscall.SIGINT,
			childExit:    41,
			cleanupDelay: 0,
		},
		{
			name:         "SIGTERM cancellation waits for cleanup",
			signal:       syscall.SIGTERM,
			childExit:    42,
			cleanupDelay: 150 * time.Millisecond,
		},
		{
			name:         "SIGTERM bypass preserves process-group cleanup",
			signal:       syscall.SIGTERM,
			childExit:    43,
			cleanupDelay: 150 * time.Millisecond,
			bypass:       true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testDirectory := t.TempDir()
			leaderReady := filepath.Join(testDirectory, "leader-ready.json")
			descendantReady := filepath.Join(testDirectory, "descendant-ready.json")
			leaderSignal := filepath.Join(testDirectory, "leader-signal")
			descendantSignal := filepath.Join(testDirectory, "descendant-signal")
			cleanupComplete := filepath.Join(testDirectory, "cleanup-complete")

			command := exec.Command(buildoptBinary, "run", "--", signalHelper, "tree")
			command.Env = append(
				os.Environ(),
				leaderReadyEnvironment+"="+leaderReady,
				descendantReadyEnvironment+"="+descendantReady,
				leaderSignalEnvironment+"="+leaderSignal,
				descendantSignalEnvironment+"="+descendantSignal,
				cleanupCompleteEnvironment+"="+cleanupComplete,
				leaderExitCodeEnvironment+"="+strconv.Itoa(testCase.childExit),
				leaderCleanupDelayEnvironment+"="+testCase.cleanupDelay.String(),
			)
			if testCase.bypass {
				command.Env = append(
					command.Env,
					bypassEnvironment+"=1",
					serverURLEnvironment+"=https://control-plane.invalid",
					serverTokenEnvironment+"=parent-server-token",
					buildSessionContextEnvironment+"={invalid-json",
				)
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr

			if err := command.Start(); err != nil {
				t.Fatalf("start buildopt: %v", err)
			}
			waitResult := make(chan error, 1)
			go func() {
				waitResult <- command.Wait()
			}()
			finished := false
			t.Cleanup(func() {
				if finished {
					return
				}
				killObservedProcess(leaderReady)
				killObservedProcess(descendantReady)
				_ = command.Process.Kill()
			})

			leader := readProcessObservation(t, leaderReady, 5*time.Second)
			descendant := readProcessObservation(t, descendantReady, 5*time.Second)
			launcherProcessGroup, err := syscall.Getpgid(command.Process.Pid)
			if err != nil {
				t.Fatalf("read launcher process group: %v", err)
			}
			if leader.PGID != leader.PID {
				t.Errorf("child process group = %d, want child PID %d", leader.PGID, leader.PID)
			}
			if leader.PGID == launcherProcessGroup {
				t.Errorf("child process group = launcher process group %d", launcherProcessGroup)
			}
			if leader.ParentPGID != launcherProcessGroup {
				t.Errorf(
					"child observed parent process group = %d, want %d",
					leader.ParentPGID,
					launcherProcessGroup,
				)
			}
			if descendant.PGID != leader.PGID {
				t.Errorf(
					"descendant process group = %d, want child group %d",
					descendant.PGID,
					leader.PGID,
				)
			}

			if err := command.Process.Signal(testCase.signal); err != nil {
				t.Fatalf("signal buildopt with %s: %v", testCase.signal, err)
			}
			err = waitForCommand(t, command, waitResult, 5*time.Second)
			finished = true
			if exitCode := processExitCode(t, err); exitCode != testCase.childExit {
				t.Fatalf("exit code = %d, want child exit %d", exitCode, testCase.childExit)
			}

			assertFileContent(t, leaderSignal, strconv.Itoa(int(testCase.signal)))
			assertFileContent(t, descendantSignal, strconv.Itoa(int(testCase.signal)))
			assertFileContent(t, cleanupComplete, "complete\n")
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestBuildoptReturnsConventionalStatusForUnhandledSignal(t *testing.T) {
	t.Setenv(bypassEnvironment, "")
	t.Setenv(serverURLEnvironment, "")
	t.Setenv(serverTokenEnvironment, "")
	t.Setenv(buildSessionContextEnvironment, "")
	clearManagedGatewayEnvironment(t)

	buildoptBinary := buildBuildopt(t)
	signalHelper := buildSignalHelper(t)
	ready := filepath.Join(t.TempDir(), "passive-ready.json")

	command := exec.Command(buildoptBinary, "run", "--", signalHelper, "passive")
	command.Env = append(os.Environ(), leaderReadyEnvironment+"="+ready)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Start(); err != nil {
		t.Fatalf("start buildopt: %v", err)
	}
	waitResult := make(chan error, 1)
	go func() {
		waitResult <- command.Wait()
	}()
	finished := false
	t.Cleanup(func() {
		if finished {
			return
		}
		killObservedProcess(ready)
		_ = command.Process.Kill()
	})

	readProcessObservation(t, ready, 5*time.Second)
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal buildopt with SIGTERM: %v", err)
	}
	err := waitForCommand(t, command, waitResult, 5*time.Second)
	finished = true
	if exitCode := processExitCode(t, err); exitCode != 128+int(syscall.SIGTERM) {
		t.Fatalf("exit code = %d, want %d", exitCode, 128+int(syscall.SIGTERM))
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestBuildoptCancellationClosesInvocationResources(t *testing.T) {
	t.Setenv(bypassEnvironment, "")
	t.Setenv(buildSessionContextEnvironment, "")
	clearManagedGatewayEnvironment(t)

	const token = "signal-session-token-0123456789abcdefghijkl"
	store := sessioningest.NewStore()
	handler, err := sessioningest.NewHandler(token, store, nil)
	if err != nil {
		t.Fatalf("create session ingest handler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	buildoptBinary := buildBuildopt(t)
	signalHelper := buildSignalHelper(t)
	testDirectory := t.TempDir()
	leaderReady := filepath.Join(testDirectory, "leader-ready.json")
	descendantReady := filepath.Join(testDirectory, "descendant-ready.json")
	leaderSignal := filepath.Join(testDirectory, "leader-signal")
	descendantSignal := filepath.Join(testDirectory, "descendant-signal")
	cleanupComplete := filepath.Join(testDirectory, "cleanup-complete")

	command := exec.Command(buildoptBinary, "run", "--", signalHelper, "tree")
	command.Env = append(
		os.Environ(),
		serverURLEnvironment+"="+server.URL,
		serverTokenEnvironment+"="+token,
		leaderReadyEnvironment+"="+leaderReady,
		descendantReadyEnvironment+"="+descendantReady,
		leaderSignalEnvironment+"="+leaderSignal,
		descendantSignalEnvironment+"="+descendantSignal,
		cleanupCompleteEnvironment+"="+cleanupComplete,
		leaderExitCodeEnvironment+"=42",
		leaderCleanupDelayEnvironment+"=50ms",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Start(); err != nil {
		t.Fatalf("start buildopt: %v", err)
	}
	waitResult := make(chan error, 1)
	go func() {
		waitResult <- command.Wait()
	}()
	finished := false
	t.Cleanup(func() {
		if finished {
			return
		}
		killObservedProcess(leaderReady)
		killObservedProcess(descendantReady)
		_ = command.Process.Kill()
	})

	leader := readProcessObservation(t, leaderReady, 5*time.Second)
	descendant := readProcessObservation(t, descendantReady, 5*time.Second)
	if leader.PluginAttemptID == "" ||
		leader.PluginEventSocket == "" ||
		leader.GatewayURL == "" {
		t.Fatalf("missing active invocation observation: %+v", leader)
	}
	if _, err := os.Stat(leader.PluginEventSocket); err != nil {
		t.Fatalf("active plugin attempt socket: %v", err)
	}
	response, err := gatewayTestClient().Get(leader.GatewayURL + gatewayReadyPath)
	if err != nil {
		t.Fatalf("contact active local gateway: %v", err)
	}
	_ = response.Body.Close()

	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal buildopt with SIGTERM: %v", err)
	}
	err = waitForCommand(t, command, waitResult, 5*time.Second)
	finished = true
	if exitCode := processExitCode(t, err); exitCode != 42 {
		t.Fatalf("exit code = %d, want child cleanup exit 42", exitCode)
	}

	sessions := store.Snapshot()
	if len(sessions) != 1 ||
		sessions[0].Outcome != sessioningest.OutcomeCancelled ||
		sessions[0].ExitCode != 42 ||
		sessions[0].SessionID != leader.PluginAttemptID {
		t.Fatalf("unexpected cancelled session records: %+v", sessions)
	}
	assertFileContent(t, leaderSignal, strconv.Itoa(int(syscall.SIGTERM)))
	assertFileContent(t, descendantSignal, strconv.Itoa(int(syscall.SIGTERM)))
	assertFileContent(t, cleanupComplete, "complete\n")
	if _, err := os.Stat(leader.PluginEventSocket); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("plugin attempt socket remains after cancellation: %v", err)
	}
	if response, err := gatewayTestClient().Get(
		leader.GatewayURL + gatewayReadyPath,
	); err == nil {
		_ = response.Body.Close()
		t.Error("local gateway remains reachable after cancellation")
	}
	assertProcessGone(t, leader.PID)
	assertProcessGone(t, descendant.PID)
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !bytes.Contains(
		stderr.Bytes(),
		[]byte("buildopt-server accepted session "+sessions[0].SessionID),
	) {
		t.Fatalf("missing cancelled session acknowledgement: %q", stderr.String())
	}
	if bytes.Contains(stderr.Bytes(), []byte(token)) {
		t.Fatal("cancellation diagnostics exposed the server token")
	}
}

func buildSignalHelper(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "signal-helper")
	command := exec.Command(
		"go",
		"build",
		"-buildvcs=false",
		"-trimpath",
		"-o",
		binaryPath,
		"./cmd/buildopt/testdata/signal-helper",
	)
	command.Dir = findRepositoryRoot(t)
	command.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build signal helper: %v\n%s", err, output)
	}
	return binaryPath
}

func readProcessObservation(
	t *testing.T,
	path string,
	timeout time.Duration,
) signalProcessObservation {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		content, err := os.ReadFile(path)
		if err == nil {
			var observation signalProcessObservation
			if err := json.Unmarshal(content, &observation); err == nil {
				return observation
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read process observation %s: %v", path, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for process observation %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForCommand(
	t *testing.T,
	command *exec.Cmd,
	waitResult <-chan error,
	timeout time.Duration,
) error {
	t.Helper()

	select {
	case err := <-waitResult:
		return err
	case <-time.After(timeout):
		_ = command.Process.Kill()
		t.Fatalf("timed out waiting for buildopt")
		return nil
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(content) != expected {
		t.Errorf("%s = %q, want %q", path, content, expected)
	}
}

func killObservedProcess(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var observation signalProcessObservation
	if err := json.Unmarshal(content, &observation); err != nil || observation.PID <= 0 {
		return
	}
	_ = syscall.Kill(observation.PID, syscall.SIGKILL)
}

func assertProcessGone(t *testing.T, processID int) {
	t.Helper()

	if err := syscall.Kill(processID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("process %d remains after cancellation: %v", processID, err)
	}
}
