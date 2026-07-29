//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
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

type processObservation struct {
	PID               int    `json:"pid"`
	PGID              int    `json:"pgid"`
	ParentPGID        int    `json:"parentPgid"`
	PluginAttemptID   string `json:"pluginAttemptId,omitempty"`
	PluginEventSocket string `json:"pluginEventSocket,omitempty"`
	GatewayURL        string `json:"gatewayUrl,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: signal-helper <tree|descendant|passive>")
		os.Exit(64)
	}

	var exitCode int
	switch os.Args[1] {
	case "tree":
		exitCode = runTreeLeader()
	case "descendant":
		exitCode = runDescendant()
	case "passive":
		exitCode = runPassive()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "signal-helper: unknown mode %q\n", os.Args[1])
		exitCode = 64
	}
	os.Exit(exitCode)
}

func runTreeLeader() int {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	executable, err := os.Executable()
	if err != nil {
		return fail("resolve helper executable", err)
	}
	descendant := exec.Command(executable, "descendant")
	descendant.Stdout = os.Stdout
	descendant.Stderr = os.Stderr
	if err := descendant.Start(); err != nil {
		return fail("start descendant", err)
	}

	descendantReady, err := requiredEnvironment(descendantReadyEnvironment)
	if err != nil {
		_ = descendant.Process.Kill()
		_ = descendant.Wait()
		return fail("read descendant-ready path", err)
	}
	if err := waitForFile(descendantReady, 5*time.Second); err != nil {
		_ = descendant.Process.Kill()
		_ = descendant.Wait()
		return fail("wait for descendant", err)
	}

	leaderReady, err := requiredEnvironment(leaderReadyEnvironment)
	if err != nil {
		_ = descendant.Process.Kill()
		_ = descendant.Wait()
		return fail("read leader-ready path", err)
	}
	if err := writeObservation(leaderReady); err != nil {
		_ = descendant.Process.Kill()
		_ = descendant.Wait()
		return fail("write leader observation", err)
	}

	received := <-signals
	if err := writeSignal(leaderSignalEnvironment, received); err != nil {
		_ = descendant.Process.Kill()
		_ = descendant.Wait()
		return fail("record leader signal", err)
	}

	delay, err := time.ParseDuration(os.Getenv(leaderCleanupDelayEnvironment))
	if err != nil {
		_ = descendant.Process.Kill()
		_ = descendant.Wait()
		return fail("parse cleanup delay", err)
	}
	time.Sleep(delay)

	if err := descendant.Wait(); err != nil {
		return fail("wait for descendant cleanup", err)
	}
	cleanupComplete, err := requiredEnvironment(cleanupCompleteEnvironment)
	if err != nil {
		return fail("read cleanup-complete path", err)
	}
	if err := os.WriteFile(cleanupComplete, []byte("complete\n"), 0o600); err != nil {
		return fail("record cleanup completion", err)
	}

	exitCode, err := strconv.Atoi(os.Getenv(leaderExitCodeEnvironment))
	if err != nil {
		return fail("parse leader exit code", err)
	}
	return exitCode
}

func runDescendant() int {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	ready, err := requiredEnvironment(descendantReadyEnvironment)
	if err != nil {
		return fail("read descendant-ready path", err)
	}
	if err := writeObservation(ready); err != nil {
		return fail("write descendant observation", err)
	}

	received := <-signals
	if err := writeSignal(descendantSignalEnvironment, received); err != nil {
		return fail("record descendant signal", err)
	}
	return 0
}

func runPassive() int {
	ready, err := requiredEnvironment(leaderReadyEnvironment)
	if err != nil {
		return fail("read passive-ready path", err)
	}
	if err := writeObservation(ready); err != nil {
		return fail("write passive observation", err)
	}
	for {
		time.Sleep(time.Hour)
	}
}

func writeObservation(path string) error {
	processGroupID, err := syscall.Getpgid(0)
	if err != nil {
		return fmt.Errorf("read process group: %w", err)
	}
	parentProcessGroupID, err := syscall.Getpgid(os.Getppid())
	if err != nil {
		return fmt.Errorf("read parent process group: %w", err)
	}

	output, err := json.Marshal(processObservation{
		PID:               os.Getpid(),
		PGID:              processGroupID,
		ParentPGID:        parentProcessGroupID,
		PluginAttemptID:   os.Getenv("BUILDOPT_PLUGIN_ATTEMPT_ID"),
		PluginEventSocket: os.Getenv("BUILDOPT_PLUGIN_EVENT_SOCKET"),
		GatewayURL:        os.Getenv("BUILDOPT_GATEWAY_URL"),
	})
	if err != nil {
		return fmt.Errorf("encode process observation: %w", err)
	}
	if err := os.WriteFile(path, output, 0o600); err != nil {
		return fmt.Errorf("write process observation: %w", err)
	}
	return nil
}

func writeSignal(pathEnvironment string, received os.Signal) error {
	path, err := requiredEnvironment(pathEnvironment)
	if err != nil {
		return err
	}
	unixSignal, ok := received.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unexpected signal type %T", received)
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(int(unixSignal))), 0o600); err != nil {
		return fmt.Errorf("write signal marker: %w", err)
	}
	return nil
}

func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func requiredEnvironment(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s is empty", name)
	}
	return value, nil
}

func fail(action string, err error) int {
	_, _ = fmt.Fprintf(os.Stderr, "signal-helper: %s: %v\n", action, err)
	return 90
}
