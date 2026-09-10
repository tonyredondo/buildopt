package launcher

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

const stickyWCNCPMode = "wcncp"

// The committed scope and server are authoritative. Standalone WCNCP routing
// variables cannot redirect installed observations or credentials to another scope.
func installedWCNCPObserver(root string, forWrite bool) wcncpobserve.Observer {
	config, err := stickywrapper.LoadConfig(root)
	if err != nil || config.Mode == "off" || os.Getenv(bypassEnvironment) == "1" || config.ProjectScope == "" {
		return wcncpobserve.Observer{}
	}
	base := os.Getenv("WCNCP_OUTBOX_DIR")
	if base == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return wcncpobserve.Observer{}
		}
		base = filepath.Join(cache, "buildopt", "wcncp")
	}
	if !filepath.IsAbs(base) || filepath.Clean(base) != base {
		return wcncpobserve.Observer{}
	}
	observer := wcncpobserve.Observer{RepositoryScope: config.ProjectScope,
		RouteScopeSHA256: optimizePortfolioRepositoryScope(config.ProjectScope),
		OutboxDir:        filepath.Join(base, optimizePortfolioRepositoryScope(config.ProjectScope))}
	document, token, err := parseStickyAccessTokenWithCapabilities(os.Getenv(config.CredentialEnv), config, time.Now().UTC(), sharedcache.CentralStateRead)
	clear(token)
	if err != nil || (forWrite && !centralHasCapability(document.Capabilities, sharedcache.CentralStateWrite)) {
		return observer
	}
	client, err := newStickyConnectionHTTPClientWithCA(config.ServerURL, os.Getenv(stickyWrapperCAEnvironment))
	if err != nil {
		return observer
	}
	observer.Client = client
	observer.BackendURL = config.ServerURL
	observer.Token = document.Token
	return observer
}

// runStickyWCNCP bypasses optional launcher infrastructure and starts no recorder
// or network request before Gradle. Its sole recorder runs after the supervised
// native child, and never changes the child's stdio, arguments, or exit status.
func runStickyWCNCP(root string, childArgs []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Config credentials are restricted to BUILDOPT_; no config I/O is needed before the child.
	var reserved []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		privateName := name
		if runtime.GOOS == "windows" {
			privateName = strings.ToUpper(name)
		}
		if strings.HasPrefix(privateName, "BUILDOPT_") || strings.HasPrefix(privateName, "WCNCP_") {
			reserved = append(reserved, name)
		}
	}
	finishTrace := beginStickyWCNCPTrace()
	execution := executeChildWithReserved(childArgs, nil, reserved, stdin, stdout, stderr)
	finishTrace(execution)
	if !execution.started {
		return launchErrorExitCode(childArgs[0], execution.err, stderr)
	}
	code := childWaitExitCode(childArgs[0], execution.err, stderr)
	if os.Getenv(bypassEnvironment) == "1" || validateStickyWrapperInvocation(root, childArgs) != nil {
		return code
	}
	observer := installedWCNCPObserver(root, true)
	if observer.Client != nil {
		defer observer.Client.CloseIdleConnections()
	}
	child := wcncpobserve.ChildResult{Outcome: "SUCCESS", ExitCode: &code}
	if code != 0 {
		child.Outcome = "FAILED"
	}
	var exitError *exec.ExitError
	if errors.As(execution.err, &exitError) {
		if signalCode, ok := platformSignalExitCode(exitError); ok {
			signal := signalCode - 128
			child = wcncpobserve.ChildResult{Outcome: "SIGNALED", Signal: &signal}
		}
	}
	_ = observer.Record(os.Getenv, root, childArgs[1:], wcncpobserve.PassthroughResult{Child: child, Duration: execution.completedAt.Sub(execution.startedAt)})
	return code
}

func installedWCNCPStatus(root, reportType string) (stickywrapper.StatusReport, error) {
	report, err := stickywrapper.BuildNativeStatus(root, reportType)
	if err != nil {
		return report, err
	}
	observer := installedWCNCPObserver(root, false)
	if observer.Client != nil {
		defer observer.Client.CloseIdleConnections()
	}
	status := observer.Status()
	report.NativeCorrection = &status
	report.Explanation = []string{"Native correction: " + status.State + " — " + status.Detail + ".",
		"The requested native Gradle workflow is retained.",
		"Savings are unavailable until controlled validation supplies a verified ledger."}
	return report, report.Validate()
}
