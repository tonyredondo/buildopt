package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	exitUsage         = 64
	exitCannotExecute = 126
	exitNotFound      = 127
	usage             = "usage: buildopt run -- <command> [args...]\n"
)

// Run executes the WS-001 passthrough command with the WS-002 process contract,
// exposes the neutral authenticated WS-003/WS-004 local rendezvous, delivers
// the WS-005 session ingest when configured, and returns the child process exit
// status.
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
	startedAt := time.Now()
	serverClient, serverConfigured, serverErr :=
		sessioningest.ClientFromEnvironment(os.Getenv)
	if serverErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: buildopt-server session ingest unavailable: %v\n",
			serverErr,
		)
	}
	gateway, gatewayErr := startLocalGateway()
	if gatewayErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: local gateway unavailable: %v\n",
			gatewayErr,
		)
	}
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
	rendezvousEnvironment := map[string]string(nil)
	if gateway != nil && handshake != nil {
		rendezvousEnvironment = map[string]string{
			pluginAttemptIDEnvironment:   handshake.attemptID,
			pluginSocketEnvironment:      handshake.listener.Addr().String(),
			pluginTokenEnvironment:       handshake.tokenText,
			gatewayURLEnvironment:        gateway.endpoint,
			gatewayUsernameEnvironment:   gateway.username,
			gatewayPasswordEnvironment:   gateway.password,
			gatewayGenerationEnvironment: gateway.generation,
		}
	}
	command.Env = replaceEnvironment(os.Environ(), rendezvousEnvironment)

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	err := command.Start()
	if err != nil {
		signal.Stop(signals)
		if handshake != nil {
			_ = handshake.finish()
		}
		if gateway != nil {
			_ = gateway.close()
		}
		return launchErrorExitCode(childArgs[0], err, stderr)
	}

	stopForwarding := make(chan struct{})
	forwardingStopped := make(chan struct{})
	go forwardSignals(command.Process.Pid, signals, stopForwarding, forwardingStopped, stderr)

	err = command.Wait()
	completedAt := time.Now()
	signal.Stop(signals)
	close(stopForwarding)
	<-forwardingStopped

	if handshake != nil {
		reportPluginHandshake(handshake.finish(), stderr)
	}
	exitCode := childWaitExitCode(childArgs[0], err, stderr)
	if serverConfigured && gateway != nil && handshake != nil {
		outcome := sessioningest.OutcomeBuildFailure
		if exitCode == 0 {
			outcome = sessioningest.OutcomeSuccess
		}
		record := sessioningest.NewRecord(
			handshake.attemptID,
			gateway.generation,
			startedAt,
			completedAt,
			outcome,
			exitCode,
		)
		ingestContext, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		result, ingestErr := gateway.deliverSession(
			ingestContext,
			serverClient,
			record,
		)
		cancel()
		reportSessionIngest(record.SessionID, result, ingestErr, stderr)
	}
	if gateway != nil {
		reportLocalGatewayClose(gateway.close(), stderr)
	}

	return exitCode
}

func childWaitExitCode(command string, err error, stderr io.Writer) int {
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
			command,
		)
		return 1
	}

	return launchErrorExitCode(command, err, stderr)
}

func reportLocalGatewayClose(err error, stderr io.Writer) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: local gateway shutdown incomplete: %v\n",
		err,
	)
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
		"buildopt: authenticated Gradle plugin handshake accepted (protocol 1.0, plugin %s)\n",
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
