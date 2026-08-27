//go:build !windows

package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func notifyOptimizeLearningContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

func executeChild(childArgs []string, environmentOverrides map[string]string, stdin io.Reader, stdout, stderr io.Writer) childExecution {
	return executeChildWithReserved(childArgs, environmentOverrides, nil, stdin, stdout, stderr)
}

func executeChildWithReserved(childArgs []string, environmentOverrides map[string]string, additionalReserved []string, stdin io.Reader, stdout, stderr io.Writer) childExecution {
	return executeChildWithReservedDirectory(childArgs, environmentOverrides, additionalReserved, "", stdin, stdout, stderr)
}

func executeChildWithReservedDirectory(childArgs []string, environmentOverrides map[string]string, additionalReserved []string, directory string, stdin io.Reader, stdout, stderr io.Writer) childExecution {
	command := exec.Command(childArgs[0], childArgs[1:]...)
	command.Dir = directory
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Env = replaceEnvironmentWithReserved(os.Environ(), environmentOverrides, additionalReserved)
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	startedAt := time.Now()
	err := command.Start()
	if err != nil {
		signal.Stop(signals)
		return childExecution{startedAt: startedAt, err: err}
	}
	stopForwarding := make(chan struct{})
	forwardingStopped := make(chan struct{})
	cancellationForwarded := make(chan struct{}, 1)
	go forwardUnixSignals(command.Process.Pid, signals, stopForwarding, forwardingStopped, cancellationForwarded, stderr)
	err = command.Wait()
	completedAt := time.Now()
	signal.Stop(signals)
	close(stopForwarding)
	<-forwardingStopped
	return childExecution{started: true, startedAt: startedAt, completedAt: completedAt, cancelled: len(cancellationForwarded) > 0 || platformChildWasCancelled(err), err: err}
}

func platformChildWasCancelled(err error) bool {
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return false
	}
	status, ok := exitError.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && (status.Signal() == syscall.SIGINT || status.Signal() == syscall.SIGTERM)
}

func platformSignalExitCode(exitError *exec.ExitError) (int, bool) {
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0, false
	}
	return 128 + int(status.Signal()), true
}

func forwardUnixSignals(processGroupID int, signals <-chan os.Signal, stop <-chan struct{}, stopped chan<- struct{}, cancellationForwarded chan<- struct{}, stderr io.Writer) {
	defer close(stopped)
	for {
		select {
		case <-stop:
			return
		default:
		}
		select {
		case <-stop:
			return
		case incoming := <-signals:
			unixSignal, ok := incoming.(syscall.Signal)
			if !ok {
				continue
			}
			err := syscall.Kill(-processGroupID, unixSignal)
			if err == nil {
				select {
				case cancellationForwarded <- struct{}{}:
				default:
				}
			} else if !errors.Is(err, syscall.ESRCH) {
				_, _ = fmt.Fprintf(stderr, "buildopt: cannot forward %s to process group %d: %v\n", incoming, processGroupID, err)
			}
		}
	}
}
