package stickywrapper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Usage is the maintainer and read-only customer CLI surface owned by the
// sticky wrapper. Status and explain never modify repository or cache state.
const Usage = "usage: buildopt wrapper init [--server URL --project-scope SCOPE] [--mode auto|observe|off]\n" +
	"       buildopt wrapper check\n" +
	"       buildopt wrapper update --version VERSION [--allow-downgrade]\n" +
	"       buildopt wrapper status [--root ABSOLUTE_PATH] [--json]\n" +
	"       buildopt wrapper explain [--root ABSOLUTE_PATH] [--json]\n"

// RunCLI executes the wrapper generator in the current repository directory.
func RunCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && isHelp(args[0]) {
		_, _ = io.WriteString(stdout, Usage)
		return 0
	}
	root, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: wrapper generator unavailable: %v\n", err)
		return 70
	}
	generator := Generator{
		Root:     root,
		Resolver: NewGitHubResolver(os.Getenv("GITHUB_TOKEN")),
	}
	return runCLI(args, stdout, stderr, generator)
}

func runCLI(args []string, stdout, stderr io.Writer, generator Generator) int {
	if len(args) == 1 && isHelp(args[0]) {
		_, _ = io.WriteString(stdout, Usage)
		return 0
	}
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, Usage)
		return 64
	}
	switch args[0] {
	case "init":
		return runInit(context.Background(), generator, args[1:], stdout, stderr)
	case "check":
		if len(args) != 1 {
			_, _ = io.WriteString(stderr, Usage)
			return 64
		}
		snapshot, checkErr := generator.Check()
		if checkErr != nil {
			return reportError(checkErr, stderr)
		}
		_, _ = fmt.Fprintf(stdout, "BuildOpt wrapper check: OK (version %s)\n", snapshot.Release.Version)
		return 0
	case "update":
		return runUpdate(context.Background(), generator, args[1:], stdout, stderr)
	case "status":
		return runReport("STATUS", generator, args[1:], stdout, stderr)
	case "explain":
		return runReport("EXPLAIN", generator, args[1:], stdout, stderr)
	default:
		_, _ = io.WriteString(stderr, Usage)
		return 64
	}
}

func runReport(reportType string, generator Generator, args []string, stdout, stderr io.Writer) int {
	jsonOutput := false
	root := generator.Root
	rootSet := false
	for len(args) > 0 {
		switch args[0] {
		case "--json":
			if jsonOutput {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			jsonOutput = true
			args = args[1:]
		case "--root":
			if rootSet || len(args) < 2 {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			rootSet = true
			root = args[1]
			args = args[2:]
		default:
			_, _ = io.WriteString(stderr, Usage)
			return 64
		}
	}
	if root == "" {
		_, _ = io.WriteString(stderr, "buildopt: wrapper status root is unavailable\n")
		return 65
	}
	report, err := BuildStatus(root, reportType)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: wrapper %s unavailable: %v\n", strings.ToLower(reportType), err)
		return 65
	}
	if err := WriteReport(report, jsonOutput, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write wrapper %s: %v\n", strings.ToLower(reportType), err)
		return 70
	}
	return 0
}

func runInit(
	ctx context.Context,
	generator Generator,
	args []string,
	stdout, stderr io.Writer,
) int {
	config := DefaultConfig()
	serverSet := false
	projectSet := false
	modeSet := false
	for len(args) > 0 {
		switch args[0] {
		case "--server":
			if serverSet || len(args) < 2 {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			serverSet = true
			config.ServerURL = args[1]
			args = args[2:]
		case "--project-scope":
			if projectSet || len(args) < 2 {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			projectSet = true
			config.ProjectScope = args[1]
			args = args[2:]
		case "--mode":
			if modeSet || len(args) < 2 {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			modeSet = true
			config.Mode = args[1]
			args = args[2:]
		default:
			_, _ = io.WriteString(stderr, Usage)
			return 64
		}
	}
	if serverSet != projectSet {
		_, _ = io.WriteString(stderr, "buildopt: --server and --project-scope must be provided together\n")
		return 64
	}
	if serverSet {
		config.CredentialEnv = "BUILDOPT_TOKEN"
	}
	snapshot, err := generator.Init(ctx, config)
	if err != nil {
		return reportError(err, stderr)
	}
	_, _ = fmt.Fprintf(stdout, "BuildOpt wrapper initialized: version %s (4 files)\n", snapshot.Release.Version)
	return 0
}

func runUpdate(
	ctx context.Context,
	generator Generator,
	args []string,
	stdout, stderr io.Writer,
) int {
	version := ""
	allowDowngrade := false
	versionSet := false
	allowSet := false
	for len(args) > 0 {
		switch args[0] {
		case "--version":
			if versionSet || len(args) < 2 {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			versionSet = true
			version = args[1]
			args = args[2:]
		case "--allow-downgrade":
			if allowSet {
				_, _ = io.WriteString(stderr, Usage)
				return 64
			}
			allowSet = true
			allowDowngrade = true
			args = args[1:]
		default:
			_, _ = io.WriteString(stderr, Usage)
			return 64
		}
	}
	if version == "" {
		_, _ = io.WriteString(stderr, Usage)
		return 64
	}
	before, after, changed, err := generator.Update(ctx, version, allowDowngrade)
	if err != nil {
		return reportError(err, stderr)
	}
	if !changed {
		_, _ = fmt.Fprintf(stdout, "BuildOpt wrapper already at version %s\n", before.Release.Version)
		return 0
	}
	_, _ = fmt.Fprintf(
		stdout,
		"BuildOpt wrapper updated: %s -> %s\n",
		before.Release.Version,
		after.Release.Version,
	)
	return 0
}

func reportError(err error, stderr io.Writer) int {
	exitCode := 70
	var classifiedError *Error
	if errors.As(err, &classifiedError) {
		switch classifiedError.Kind {
		case ErrorUsage:
			exitCode = 64
		case ErrorCommittedData:
			exitCode = 65
		case ErrorNetwork:
			exitCode = 69
		case ErrorInternal:
			exitCode = 70
		}
	}
	_, _ = fmt.Fprintf(stderr, "buildopt: wrapper generator unavailable: %v\n", err)
	return exitCode
}

func isHelp(value string) bool {
	return value == "help" || value == "--help" || value == "-h"
}
