package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	exitUsage         = 64
	exitCannotExecute = 126
	exitNotFound      = 127
	exitConfiguration = 78
	usage             = "usage: buildopt run -- <command> [args...]\n       buildopt gradle [gradle args...]\n       buildopt poc --changes-file PATH [options]\n       buildopt impact --repository-id OWNER/REPO --changes-file PATH [options]\n       buildopt doctor\n"
	bypassEnvironment = "BUILDOPT_BYPASS"
)

// Run executes the WS-001 passthrough command with the WS-002 process contract,
// exposes the authenticated local rendezvous through either its walking-
// skeleton or managed runner-slot lifecycle, delivers the WS-005 session
// ingest when configured, installs the signed A0-007 Gradle bootstrap cache,
// honors the F0-039 local bypass, and returns the child process exit status.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) (runExitCode int) {
	var impactTiming *impactTimingState
	execute := func(
		childArgs []string,
		environmentOverrides map[string]string,
	) childExecution {
		if impactTiming != nil {
			return impactTiming.execute(
				childArgs,
				environmentOverrides,
				stdin,
				stdout,
				stderr,
			)
		}
		return executeChild(
			childArgs,
			environmentOverrides,
			stdin,
			stdout,
			stderr,
		)
	}
	if len(args) > 0 && args[0] == managedGatewayInternalCommand {
		return runManagedGatewayProcess(args, stderr)
	}
	if len(args) == 1 && args[0] == "doctor" {
		return runDoctor(stdout, stderr)
	}
	if isHelp(args) {
		_, _ = io.WriteString(stdout, usage)
		return 0
	}
	if len(args) == 2 && args[0] == "impact" && isHelp(args[1:]) {
		_, _ = io.WriteString(stdout, impactUsage)
		return 0
	}
	if len(args) == 2 && args[0] == "poc" && isHelp(args[1:]) {
		_, _ = io.WriteString(stdout, qualifiedPOCProfileUsage)
		return 0
	}
	gradleEnvironment := map[string]string(nil)
	var gradleManagedL1 *managedL1Config
	gradleNativeOnly := false
	gradleLocalOnly := false
	impactStandardJarCache := false
	impactStandardCopyCache := false
	impactPOCEdgeCacheURL := ""
	qualifiedPOCProfile := false
	if len(args) > 0 && (args[0] == "impact" || args[0] == "poc") {
		impactStartedAt := time.Now()
		var impact impactInvocation
		var err error
		if args[0] == "poc" {
			impact, err = prepareQualifiedPOCProfileInvocation(
				args[1:],
				os.Getenv(bypassEnvironment) == "1",
			)
		} else {
			impact, err = prepareImpactInvocation(
				args[1:],
				os.Getenv(bypassEnvironment) == "1",
			)
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: Build Impact POC unavailable: %v\n", err)
			if args[0] == "poc" {
				_, _ = io.WriteString(stderr, qualifiedPOCProfileUsage)
			} else {
				_, _ = io.WriteString(stderr, impactUsage)
			}
			return exitConfiguration
		}
		if impact.timingsPath != "" {
			impactTiming = newImpactTimingState(
				impact.timingsPath,
				impactStartedAt,
				impact,
			)
			defer func() {
				if timingErr := impactTiming.write(runExitCode); timingErr != nil {
					_, _ = fmt.Fprintf(
						stderr,
						"buildopt: Build Impact phase timing unavailable: %v\n",
						timingErr,
					)
					if runExitCode == 0 {
						runExitCode = exitConfiguration
					}
				}
			}()
		}
		if impact.plan.CandidateSelected {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt: explicit Build Impact POC candidate %s selected; this is not production authorization\n",
				impact.plan.AlternativeID,
			)
		} else {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt: Build Impact POC retained the full graph (%s)\n",
				impact.plan.Reason,
			)
		}
		if impact.hotStateHit {
			_, _ = fmt.Fprintln(stderr, "buildopt: exact-bound Build Impact POC hot state reused")
		}
		if impact.qualifiedProfile != nil {
			if err := writeQualifiedPOCProfilePlan(stderr, *impact.qualifiedProfile); err != nil {
				_, _ = fmt.Fprintf(stderr, "buildopt: qualified POC profile plan unavailable: %v\n", err)
				return exitConfiguration
			}
		}
		impactStandardJarCache = impact.standardJarCache
		impactStandardCopyCache = impact.standardCopyCache
		impactPOCEdgeCacheURL = impact.pocEdgeCacheURL
		qualifiedPOCProfile = impact.qualifiedProfile != nil
		args = append([]string{"gradle"}, impact.gradleArgs...)
	}
	if len(args) > 0 && args[0] == "gradle" {
		gradleSetupStartedAt := time.Now()
		gradleArgs, explicitStandardJarCache, parseErr := parseGradleProductOptions(args[1:])
		if parseErr != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: Gradle setup unavailable: %v\n", parseErr)
			return exitConfiguration
		}
		getenv := os.Getenv
		if qualifiedPOCProfile {
			getenv = qualifiedPOCProfileEnvironment(os.Getenv, impactStandardJarCache, impactPOCEdgeCacheURL)
		} else if explicitStandardJarCache || impactStandardJarCache || impactStandardCopyCache {
			getenv = func(name string) string {
				switch name {
				case gradleStandardJarCacheEnvironment:
					if explicitStandardJarCache || impactStandardJarCache {
						return "1"
					}
				case gradleStandardCopyCacheEnvironment:
					if impactStandardCopyCache {
						return "1"
					}
				}
				return os.Getenv(name)
			}
		}
		invocation, err := prepareGradleInvocationWithEnvironment(
			gradleArgs,
			os.Getenv(bypassEnvironment) == "1",
			getenv,
		)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: Gradle setup unavailable: %v\n", err)
			return exitConfiguration
		}
		if impactTiming != nil {
			impactTiming.finishGradleSetup(gradleSetupStartedAt)
		}
		args = append([]string{"run", "--"}, invocation.childArgs...)
		gradleEnvironment = invocation.environment
		gradleManagedL1 = invocation.managedL1
		gradleNativeOnly = invocation.nativeOnly
		gradleLocalOnly = invocation.localOnly
	}
	if len(args) < 3 || args[0] != "run" || args[1] != "--" {
		_, _ = io.WriteString(stderr, usage)
		return exitUsage
	}

	childArgs := args[2:]
	if os.Getenv(bypassEnvironment) == "1" {
		execution := execute(
			childArgs,
			gradleEnvironment,
		)
		if !execution.started {
			return launchErrorExitCode(childArgs[0], execution.err, stderr)
		}
		return childWaitExitCode(childArgs[0], execution.err, stderr)
	}
	if gradleNativeOnly || gradleLocalOnly {
		if impactTiming == nil && nativeGradleProcessReplacementSupported(stdin, stdout, stderr) {
			err := replaceWithNativeGradleProcess(childArgs, gradleEnvironment)
			return launchErrorExitCode(childArgs[0], err, stderr)
		}
		execution := execute(
			childArgs,
			gradleEnvironment,
		)
		if !execution.started {
			return launchErrorExitCode(childArgs[0], execution.err, stderr)
		}
		return childWaitExitCode(childArgs[0], execution.err, stderr)
	}

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
	exportContext, exportContextConfigured, exportContextErr :=
		sessioningest.ExportContextFromEnvironment(os.Getenv)
	if serverConfigured && exportContextErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: BUILD_SESSION export context unavailable: %v\n",
			exportContextErr,
		)
	}
	authority, authorityConfigured, authorityErr :=
		localAuthorityContextFromEnvironment(
			context.Background(),
			os.Getenv,
			startedAt,
		)
	if authorityErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: local cache authority unavailable: %v\n",
			authorityErr,
		)
	}
	gradleBootstrap, _, gradleBootstrapErr :=
		startInvocationGradleBootstrapCache(
			childArgs,
			authority,
			os.Getenv,
		)
	if gradleBootstrapErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: managed Gradle bootstrap cache unavailable: %v\n",
			gradleBootstrapErr,
		)
	}

	localCacheFastPath := useGradleLocalCacheFastPath(
		gradleManagedL1,
		serverConfigured,
		serverErr,
		authorityConfigured,
		authorityErr,
	)
	if gradleManagedL1 != nil {
		if localCacheFastPath {
			gradleEnvironment[gradleProjectPluginModeEnvironment] = gradleProjectPluginModeCacheOnly
		} else {
			gradleEnvironment[gradleProjectPluginModeEnvironment] = gradleProjectPluginModeFull
		}
	}
	var handshake *pluginHandshakeServer
	var handshakeErr error
	if !localCacheFastPath {
		if authority != nil {
			handshake, handshakeErr = startPluginHandshakeForAttempt(
				authority.attemptID,
			)
		} else {
			handshake, handshakeErr = startPluginHandshake()
		}
	}
	if handshakeErr != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: Gradle plugin handshake unavailable: %v\n",
			handshakeErr,
		)
	}
	var gateway *localGateway
	if handshake != nil {
		var gatewayErr error
		var cacheBinding *gatewayCacheBinding
		if authority != nil {
			cacheBinding = authority.cacheBinding
		}
		gateway, gatewayErr = startInvocationGatewayWithCache(
			handshake.attemptID,
			cacheBinding,
		)
		if gatewayErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt: local gateway unavailable: %v\n",
				gatewayErr,
			)
		}
	}
	var l1 *managedL1
	var l1Err error
	if authority != nil {
		l1, l1Err = startManagedL1(
			managedL1ConfigForInvocation(authority, gateway),
		)
	} else if !authorityConfigured {
		if gradleManagedL1 != nil {
			l1, l1Err = startManagedL1(*gradleManagedL1)
		} else {
			l1, l1Err = startInvocationManagedL1()
		}
	}
	if l1Err != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: managed L1 unavailable: %v\n",
			l1Err,
		)
	}

	childEnvironment := copyEnvironment(gradleEnvironment)
	if gateway != nil && handshake != nil {
		if childEnvironment == nil {
			childEnvironment = make(map[string]string)
		}
		for key, value := range map[string]string{
			pluginAttemptIDEnvironment:   handshake.attemptID,
			pluginSocketEnvironment:      handshake.endpoint,
			pluginTokenEnvironment:       handshake.tokenText,
			gatewayURLEnvironment:        gateway.endpoint,
			gatewayUsernameEnvironment:   gateway.username,
			gatewayPasswordEnvironment:   gateway.password,
			gatewayGenerationEnvironment: gateway.generation,
		} {
			childEnvironment[key] = value
		}
		if managedSharedAuthorityEnabled(authority, gateway) {
			for key, value := range authority.childEnvironment {
				childEnvironment[key] = value
			}
		}
	}
	if l1 != nil {
		if childEnvironment == nil {
			childEnvironment = make(map[string]string)
		}
		for key, value := range l1.childEnvironment() {
			childEnvironment[key] = value
		}
	}
	if gradleBootstrap != nil {
		if childEnvironment == nil {
			childEnvironment = make(map[string]string)
		}
		for key, value := range gradleBootstrap.childEnvironment() {
			childEnvironment[key] = value
		}
	}
	profiledArgs, resourceProfileErr := applyAuthorizedResourceProfile(childArgs, authority, os.Getenv)
	if resourceProfileErr != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: resource profile unavailable: %v\n", resourceProfileErr)
	} else {
		childArgs = profiledArgs
	}
	childArgs = applyConfigurationCachePolicy(childArgs, authority)
	execution := execute(
		childArgs,
		childEnvironment,
	)
	if !execution.started {
		if handshake != nil {
			_ = handshake.finish()
		}
		if gateway != nil {
			_ = gateway.close()
		}
		if l1 != nil {
			_ = l1.close()
		}
		if gradleBootstrap != nil {
			_ = gradleBootstrap.close()
		}
		return launchErrorExitCode(childArgs[0], execution.err, stderr)
	}
	if l1 != nil {
		reportManagedL1Close(l1.close(), stderr)
	}
	if gradleBootstrap != nil {
		reportGradleBootstrapFinalize(
			gradleBootstrap.finalize(),
			stderr,
		)
		reportGradleBootstrapClose(gradleBootstrap.close(), stderr)
	}

	handshakeResult := pluginHandshakeResult{}
	if handshake != nil {
		handshakeResult = handshake.finish()
		reportPluginHandshake(handshakeResult, stderr)
	}
	exitCode := childWaitExitCode(childArgs[0], execution.err, stderr)
	if serverConfigured && gateway != nil && handshake != nil {
		outcome := sessionOutcome(execution, exitCode)
		record := sessioningest.NewRecord(
			handshake.attemptID,
			gateway.generation,
			startedAt,
			execution.completedAt,
			outcome,
			exitCode,
		)
		if exportContextConfigured &&
			exportContextErr == nil &&
			handshakeResult.connected &&
			handshakeResult.err == nil {
			record.ExportContext = exportContext
			record.GradleInvocation = &sessioningest.GradleInvocation{
				ID:            handshakeResult.producerInstanceID,
				StartedAt:     execution.startedAt.UTC().Format(time.RFC3339Nano),
				CompletedAt:   execution.completedAt.UTC().Format(time.RFC3339Nano),
				DurationMs:    execution.completedAt.Sub(execution.startedAt).Milliseconds(),
				PluginVersion: handshakeResult.implementationVersion,
			}
		} else if exportContextConfigured && exportContextErr == nil {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt: BUILD_SESSION export unavailable: authenticated Gradle invocation was not observed",
			)
		}
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

func managedL1ConfigForInvocation(
	authority *localAuthorityContext,
	gateway *localGateway,
) managedL1Config {
	config := authority.managedL1Config
	if gateway != nil && gateway.cacheSuppressed {
		config.l2WriteAuthorized = false
	}
	return config
}

func managedSharedAuthorityEnabled(
	authority *localAuthorityContext,
	gateway *localGateway,
) bool {
	return authority != nil && gateway != nil && !gateway.cacheSuppressed
}

type childExecution struct {
	started     bool
	startedAt   time.Time
	completedAt time.Time
	cancelled   bool
	err         error
}

func sessionOutcome(execution childExecution, exitCode int) string {
	if execution.cancelled {
		return sessioningest.OutcomeCancelled
	}
	if exitCode == 0 {
		return sessioningest.OutcomeSuccess
	}
	return sessioningest.OutcomeBuildFailure
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
		if exitCode, ok := platformSignalExitCode(exitError); ok {
			return exitCode
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

func reportManagedL1Close(err error, stderr io.Writer) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: managed L1 release incomplete: %v\n",
		err,
	)
}

func reportGradleBootstrapFinalize(err error, stderr io.Writer) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: managed Gradle Wrapper installation was not retained: %v\n",
		err,
	)
}

func reportGradleBootstrapClose(err error, stderr io.Writer) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: managed Gradle writable cache release incomplete: %v\n",
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
