package launcher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
)

const (
	exitUsage         = 64
	exitCannotExecute = 126
	exitNotFound      = 127
	usage             = "usage: buildopt run -- <command> [args...]\n"
)

// Run executes the WS-001 passthrough command and returns the process exit code.
// Process-group creation and signal forwarding are intentionally owned by WS-002.
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
	command := exec.Command(childArgs[0], childArgs[1:]...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()
	if err == nil {
		return 0
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if exitCode := exitError.ExitCode(); exitCode >= 0 {
			return exitCode
		}
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: command %q terminated without an exit code\n",
			childArgs[0],
		)
		return 1
	}

	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: cannot execute %q: %v\n",
		childArgs[0],
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
