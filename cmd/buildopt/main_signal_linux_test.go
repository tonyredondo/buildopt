//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
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
	PID        int `json:"pid"`
	PGID       int `json:"pgid"`
	ParentPGID int `json:"parentPgid"`
}

func TestBuildoptForwardsSignalsToChildProcessGroup(t *testing.T) {
	t.Setenv(bypassEnvironment, "")
	t.Setenv(serverURLEnvironment, "")
	t.Setenv(serverTokenEnvironment, "")
	t.Setenv(buildSessionContextEnvironment, "")

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
