package launcher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const (
	exitUsage         = 64
	exitCannotExecute = 126
	exitNotFound      = 127
	usage             = "usage: buildopt run -- <command> [args...]\n"
)

// Run executes the WS-001 passthrough command with the WS-002 process contract,
// exposes the neutral WS-003 plugin handshake, and returns the child process
// exit status.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, usage)
		return 0
	}
	if len(args) < 3 || args[0] != "run" || args[1] != "--" {
		_, _ = io.WriteString(stderr, usage)
		return exitUsage
	}

	childArgs := args[2:]
	handshake, handshakeErr := startPluginHandshake()
	if handshakeErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: Gradle plugin handshake unavailable: %v\n",
			handshakeErr,
		)
	}

	command := exec.Command(childArgs[0], childArgs[1:]...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if handshake != nil {
		command.Env = handshake.childEnvironment(os.Environ())
	}

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	err := command.Start()
	if err != nil {
		signal.Stop(signals)
		if handshake != nil {
			_ = handshake.finish()
		}
		return launchErrorExitCode(childArgs[0], err, stderr)
	}

	stopForwarding := make(chan struct{})
	forwardingStopped := make(chan struct{})
	go forwardSignals(command.Process.Pid, signals, stopForwarding, forwardingStopped, stderr)

	err = command.Wait()
	signal.Stop(signals)
	close(stopForwarding)
	<-forwardingStopped

	if handshake != nil {
		reportPluginHandshake(handshake.finish(), stderr)
	}

	if err == nil {
		return 0
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if exitCode := exitError.ExitCode(); exitCode >= 0 {
			return exitCode
		}
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal())
		}
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: command %q terminated without an exit code\n",
			childArgs[0],
		)
		return 1
	}

	return launchErrorExitCode(childArgs[0], err, stderr)
}

func reportPluginHandshake(result pluginHandshakeResult, stderr io.Writer) {
	if !result.connected && result.err == nil {
		return
	}
	if result.err != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: Gradle plugin handshake unavailable: %v\n",
			result.err,
		)
		return
	}
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: Gradle plugin handshake accepted (protocol 1.0, plugin %s)\n",
		result.implementationVersion,
	)
}

func forwardSignals(
	processGroupID int,
	signals <-chan os.Signal,
	stop <-chan struct{},
	stopped chan<- struct{},
	stderr io.Writer,
) {
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
			if err := syscall.Kill(-processGroupID, unixSignal); err != nil &&
				!errors.Is(err, syscall.ESRCH) {
				_, _ = fmt.Fprintf(
					stderr,
					"buildopt: cannot forward %s to process group %d: %v\n",
					incoming,
					processGroupID,
					err,
				)
			}
		}
	}
}

func launchErrorExitCode(command string, err error, stderr io.Writer) int {
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: cannot execute %q: %v\n",
		command,
		err,
	)
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return exitNotFound
	}
	return exitCannotExecute
}

func isHelp(args []string) bool {
	if len(args) != 1 {
		return false
	}
	return args[0] == "help" || args[0] == "--help" || args[0] == "-h"
}
