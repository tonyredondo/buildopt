// Command buildoptw is the ordinary Gradle entrypoint for WCNCP repositories.
//
//	./buildoptw <gradle arguments...>
//	./buildoptw --wcncp-status
//	./buildoptw --wcncp-explain
//
// The customer-visible behavior is: run the exact requested workflow through
// the repository native Gradle Wrapper, never make successful Gradle execution
// depend on BuildOpt service health, record a small private typed observation
// after the build, and explain whether BuildOpt is observing, has found an
// opportunity, is validating, or has a proposal ready. Expensive validation
// runs only on authorized validators within a fixed budget; source application
// and merge stay with the repository owner.
//
// Wrapper-reserved flags (--wcncp-status, --wcncp-explain) are stripped before
// Gradle passthrough and never forwarded, so native task selection is
// unchanged for every real Gradle invocation.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

const (
	wrapperScriptPOSIX   = "gradlew"
	wrapperScriptWindows = "gradlew.bat"
)

func main() {
	code := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, os.Getwd)
	if code < 0 {
		exitWithSignal(-code)
	}
	os.Exit(code)
}

func run(args []string, getenv func(string) string, stdin *os.File, stdout, stderr *os.File, getwd func() (string, error)) int {
	// Wrapper-reserved surfaces never reach Gradle.
	if len(args) == 1 && (args[0] == "--wcncp-status" || args[0] == "--wcncp-explain") {
		return runStatus(args[0], getenv, stdout)
	}
	cwd, err := getwd()
	if err != nil {
		cwd = "."
	}
	wrapper := wrapperScriptPOSIX
	if os.PathSeparator == '\\' {
		wrapper = wrapperScriptWindows
	}
	wrapperPath := filepath.Join(cwd, wrapper)
	if _, err := os.Stat(wrapperPath); err != nil {
		// No Wrapper: fail closed with a clear diagnostic and native-like exit.
		_, _ = fmt.Fprintf(stderr, "buildoptw: no Gradle wrapper at %s\n", wrapperPath)
		return 127
	}
	if wcncpobserve.BypassRequested(mapFromEnv(getenv)) {
		result := wcncpobserve.RunNativePassthrough(context.Background(), wrapperPath, args, cwd, stdin, stdout, stderr)
		return childExitCode(result)
	}
	repositoryScope := getenv("WCNCP_REPOSITORY_SCOPE")
	outboxDir := getenv("WCNCP_OUTBOX_DIR")
	observe := repositoryScope != "" && outboxDir != ""
	result := wcncpobserve.RunNativePassthrough(context.Background(), wrapperPath, args, cwd, stdin, stdout, stderr)
	code := childExitCode(result)
	if !observe {
		return code
	}
	// Observation starts only when configured and must avoid pre-child
	// network. Facts and local queue I/O run after the child; the optional
	// upload has its own 100ms deadline and cannot replace the child result.
	_ = (wcncpobserve.Observer{RepositoryScope: repositoryScope, OutboxDir: outboxDir,
		BackendURL: getenv("WCNCP_BACKEND_URL"), Token: getenv("WCNCP_BACKEND_TOKEN")}).Record(getenv, cwd, args, result)
	return code
}

func childExitCode(result wcncpobserve.PassthroughResult) int {
	if result.Child.ExitCode != nil {
		return *result.Child.ExitCode
	}
	if result.Child.Signal != nil {
		return -*result.Child.Signal
	}
	return 1
}

func runStatus(flag string, getenv func(string) string, stdout *os.File) int {
	status := (wcncpobserve.Observer{RepositoryScope: getenv("WCNCP_REPOSITORY_SCOPE"),
		OutboxDir: getenv("WCNCP_OUTBOX_DIR"), BackendURL: getenv("WCNCP_BACKEND_URL"),
		Token: getenv("WCNCP_BACKEND_TOKEN")}).Status()
	if flag == "--wcncp-explain" {
		_, _ = fmt.Fprintf(stdout, "buildoptw %s: %s\n", status.State, status.Detail)
	} else {
		_, _ = fmt.Fprintf(stdout, "%s\n", status.State)
	}
	return 0
}

func mapFromEnv(getenv func(string) string) map[string]string {
	return map[string]string{"BUILDOPT_BYPASS": getenv("BUILDOPT_BYPASS")}
}
