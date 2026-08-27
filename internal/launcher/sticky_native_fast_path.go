package launcher

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickywrapper"
)

// stickyNativeNoopPath describes the conservative path used by the committed
// wrapper when no BuildOpt integration is configured for the invocation. It
// deliberately carries no action or cache authority: Gradle remains the only
// executor and an unknown configuration falls back to the existing full path.
type stickyNativeNoopPath struct {
	root                  string
	credentialEnvironment string
	observationMode       string
}

// prepareStickyNativeNoopPath decides whether a sticky-wrapper invocation can
// skip the launcher infrastructure. A configured server credential or any
// explicit BuildOpt integration opts into the existing instrumented path.
// Missing central credentials are safe to treat as native no-op because the
// connection path cannot authorize a read without them.
func prepareStickyNativeNoopPath(
	root string,
	childArgs []string,
	getenv func(string) string,
) (stickyNativeNoopPath, bool) {
	if getenv == nil || root == "" || len(childArgs) == 0 {
		return stickyNativeNoopPath{}, false
	}
	if err := validateStickyWrapperInvocation(root, childArgs); err != nil {
		return stickyNativeNoopPath{}, false
	}
	credentialEnvironment := stickywrapper.CredentialEnvironment(root)
	if getenv(bypassEnvironment) == "1" {
		return stickyNativeNoopPath{
			root: root, credentialEnvironment: credentialEnvironment,
			observationMode: ordinaryObservationModeDisabled,
		}, true
	}
	if stickyNativeExplicitIntegrationRequested(getenv) {
		return stickyNativeNoopPath{}, false
	}
	config, err := stickywrapper.LoadConfig(root)
	if err != nil {
		// Older or test-only wrappers may not carry the optional config file.
		// Absence means that no remote integration was requested; a present but
		// malformed file still falls back to the established full path so its
		// diagnostic behavior is not hidden.
		if _, statErr := os.Lstat(filepath.Join(root, ".buildopt", "config.toml")); !errors.Is(statErr, os.ErrNotExist) {
			return stickyNativeNoopPath{}, false
		}
		observationMode, modeOK := stickyObservationMode(getenv)
		if !modeOK {
			return stickyNativeNoopPath{}, false
		}
		return stickyNativeNoopPath{
			root: root, credentialEnvironment: credentialEnvironment,
			observationMode: observationMode,
		}, true
	}
	if config.Mode == "off" {
		return stickyNativeNoopPath{
			root: root, credentialEnvironment: config.CredentialEnv,
			observationMode: ordinaryObservationModeDisabled,
		}, true
	}
	if config.ServerURL != "" && strings.TrimSpace(getenv(config.CredentialEnv)) != "" {
		return stickyNativeNoopPath{}, false
	}
	observationMode, ok := stickyObservationMode(getenv)
	if !ok {
		return stickyNativeNoopPath{}, false
	}
	return stickyNativeNoopPath{
		root: root, credentialEnvironment: config.CredentialEnv,
		observationMode: observationMode,
	}, true
}

func stickyNativeExplicitIntegrationRequested(getenv func(string) string) bool {
	if getenv == nil {
		return true
	}
	if gradleInstrumentationRequested(getenv) ||
		getenv(gradleProjectPluginModeEnvironment) != "" {
		return true
	}
	// These switches are evaluated separately from
	// gradleInstrumentationRequested because they select a cache adapter or
	// authority path without necessarily providing a plugin or server URL.
	// Treat every non-empty value as explicit, including malformed values, so
	// the established validation/error path remains visible to the caller.
	for _, name := range []string{
		gradleSafeCacheEnvironment,
		gradleStandardJarCacheEnvironment,
		gradlePOCEdgeCacheURLEnvironment,
		sharedCacheCAPathEnvironment,
		managedSharedModeEnvironment,
		managedAuthorityDigestEnvironment,
		managedPolicyDigestEnvironment,
		managedConfigurationDigestEnvironment,
		managedAuthorityContractEnvironment,
		managedSharedPolicyEnvironment,
		managedL1L2WriterEnvironment,
		managedGatewayStateRootEnvironment,
		managedRunnerSlotEnvironment,
		managedGatewayIdleTimeoutEnvironment,
		gatewayURLEnvironment,
		gatewayUsernameEnvironment,
		gatewayPasswordEnvironment,
		gatewayGenerationEnvironment,
		pluginAttemptIDEnvironment,
		pluginSocketEnvironment,
		pluginTokenEnvironment,
		managedL1DirectoryChildEnvironment,
		managedL1ModeChildEnvironment,
		managedL1RetentionChildEnvironment,
		"BUILDOPT_RUNTIME_CHECKSTYLE_MAX_HEAP",
		"BUILDOPT_CACHE_STANDARD_COPY_TASKS",
	} {
		if strings.TrimSpace(getenv(name)) != "" {
			return true
		}
	}
	return false
}

// runStickyNativeNoop executes the original Wrapper with no gateway,
// handshake, managed L1 or bootstrap state. Observation, when requested, is
// created lazily and written only after Gradle exits, so optional diagnostics
// cannot delay startup or alter the child process. Its optional executable
// digest may run concurrently, but is never part of the build critical path.
// The direct exec path removes the BuildOpt parent entirely when no observer
// is requested.
func runStickyNativeNoop(
	path stickyNativeNoopPath,
	childArgs []string,
	startedAt time.Time,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	reserved := []string{
		stickyWrapperRootEnvironment,
		stickyWrapperCAEnvironment,
		"BUILDOPT_TOKEN",
		stickyObservationOutputEnvironment,
		stickyObservationModeEnvironment,
	}
	if path.credentialEnvironment != "" && path.credentialEnvironment != "BUILDOPT_TOKEN" {
		reserved = append(reserved, path.credentialEnvironment)
	}
	var observation *ordinaryObservationState
	if path.observationMode != ordinaryObservationModeDisabled {
		observation = newOrdinaryObservationStateAt(path.root, childArgs, startedAt)
	}
	if observation == nil && nativeGradleProcessReplacementSupported(stdin, stdout, stderr) {
		err := replaceWithNativeGradleProcessWithReserved(childArgs, nil, reserved)
		return launchErrorExitCode(childArgs[0], err, stderr)
	}
	if observation != nil {
		decisionFinishedAt := time.Now()
		observation.markConnection(decisionFinishedAt, false)
		observation.finishCache(decisionFinishedAt)
	}
	execution := executeChildWithReserved(childArgs, nil, reserved, stdin, stdout, stderr)
	exitCode := 0
	if execution.started {
		exitCode = childWaitExitCode(childArgs[0], execution.err, stderr)
	} else {
		exitCode = launchErrorExitCode(childArgs[0], execution.err, stderr)
	}
	if observation != nil {
		observation.finishGradle(execution, childArgs)
		if err := observation.finish(exitCode, time.Now().UTC()); err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: ordinary observation unavailable: %v\n", err)
		}
	}
	return exitCode
}
