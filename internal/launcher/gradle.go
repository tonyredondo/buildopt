package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	gradlePluginJarEnvironment         = "BUILDOPT_GRADLE_PLUGIN_JAR"
	gradleInitScriptEnvironment        = "BUILDOPT_GRADLE_INIT_SCRIPT"
	gradleProjectPluginModeEnvironment = "BUILDOPT_GRADLE_PROJECT_PLUGIN_MODE"
	gradleProjectPluginModeFull        = "FULL"
	gradleProjectPluginModeCacheOnly   = "CACHE_ONLY"
	gradleSafeCacheEnvironment         = "BUILDOPT_SAFE_CACHE"
	gradleStandardJarCacheEnvironment  = "BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS"
	gradleStandardJarCacheFlag         = "--cache-standard-jar-producers"
	gradlePOCEdgeCacheURLEnvironment   = "BUILDOPT_POC_EDGE_CACHE_URL"
)

func parseGradleProductOptions(args []string) ([]string, bool, error) {
	if len(args) == 0 || args[0] != gradleStandardJarCacheFlag {
		return args, false, nil
	}
	if len(args) < 2 || args[1] != "--" {
		return nil, false, fmt.Errorf(
			"%s requires '--' before Gradle arguments",
			gradleStandardJarCacheFlag,
		)
	}
	return args[2:], true, nil
}

type gradleInvocation struct {
	childArgs   []string
	environment map[string]string
	managedL1   *managedL1Config
	nativeOnly  bool
	localOnly   bool
}

func prepareGradleInvocation(args []string, bypass bool) (gradleInvocation, error) {
	return prepareGradleInvocationWithEnvironment(args, bypass, os.Getenv)
}

func prepareGradleInvocationWithEnvironment(
	args []string,
	bypass bool,
	getenv func(string) string,
) (gradleInvocation, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return gradleInvocation{}, fmt.Errorf("resolve working directory: %w", err)
	}
	wrapper := filepath.Join(workingDirectory, gradleWrapperName(runtime.GOOS))
	if !regularFile(wrapper) {
		return gradleInvocation{}, fmt.Errorf(
			"%s is missing; run this command from a Gradle repository root",
			gradleWrapperName(runtime.GOOS),
		)
	}

	childArgs := []string{wrapper}
	environment := map[string]string(nil)
	var defaultManagedL1 *managedL1Config
	buildCacheConfigured, buildCacheEnabled := gradleBuildCacheMode(args)
	if !bypass {
		safeCacheEnabled, activationErr := gradleSafeCacheEnabled(getenv)
		if activationErr != nil {
			return gradleInvocation{}, activationErr
		}
		standardJarCache, jarErr := gradleStandardJarCacheEnabled(getenv)
		if jarErr != nil {
			return gradleInvocation{}, jarErr
		}
		if standardJarCache && buildCacheConfigured && !buildCacheEnabled {
			return gradleInvocation{}, errors.New("BuildOpt standard task cache adapters are incompatible with --no-build-cache")
		}
		pocEdgeCacheURL, pocEdgeErr := gradlePOCEdgeCacheURL(getenv)
		if pocEdgeErr != nil {
			return gradleInvocation{}, pocEdgeErr
		}
		if pocEdgeCacheURL != "" && buildCacheConfigured && !buildCacheEnabled {
			return gradleInvocation{}, errors.New("BuildOpt POC Edge cache is incompatible with --no-build-cache")
		}
		_, managedConfigured, _ := managedL1ConfigFromEnvironment(getenv)
		externalInstrumentation := gradleInstrumentationRequested(getenv)
		runtimeIntegration := gradleRuntimeIntegrationRequested(getenv)
		instrumented := safeCacheEnabled || standardJarCache ||
			pocEdgeCacheURL != "" || externalInstrumentation
		if instrumented {
			initScript, pluginJar, assetErr := resolveGradleAssets()
			if assetErr != nil {
				return gradleInvocation{}, assetErr
			}
			childArgs = append(childArgs, "--init-script", initScript)
			environment = map[string]string{gradlePluginJarEnvironment: pluginJar}
			if standardJarCache {
				environment[gradleStandardJarCacheEnvironment] = "1"
			}
			if pocEdgeCacheURL != "" {
				environment[gradlePOCEdgeCacheURLEnvironment] = pocEdgeCacheURL
			}
			if (standardJarCache && !runtimeIntegration ||
				pocEdgeCacheURL != "") && !safeCacheEnabled {
				environment[gradleProjectPluginModeEnvironment] = gradleProjectPluginModeCacheOnly
			}
		}
		if safeCacheEnabled && !managedConfigured &&
			(!buildCacheConfigured || buildCacheEnabled) {
			cacheRoot, cacheErr := os.UserCacheDir()
			if cacheErr != nil {
				return gradleInvocation{}, fmt.Errorf(
					"resolve user cache directory: %w",
					cacheErr,
				)
			}
			config, configErr := defaultGradleManagedL1Config(workingDirectory, cacheRoot)
			if configErr != nil {
				return gradleInvocation{}, configErr
			}
			defaultManagedL1 = &config
			if !buildCacheConfigured {
				childArgs = append(childArgs, "--build-cache")
			}
		} else if !buildCacheConfigured {
			childArgs = append(childArgs, "--build-cache")
		}
		childArgs = append(childArgs, args...)
		return gradleInvocation{
			childArgs: childArgs, environment: environment,
			managedL1: defaultManagedL1, nativeOnly: !instrumented,
			localOnly: (standardJarCache && !runtimeIntegration ||
				pocEdgeCacheURL != "") && !safeCacheEnabled,
		}, nil
	}
	childArgs = append(childArgs, args...)
	return gradleInvocation{
		childArgs: childArgs, environment: environment, managedL1: defaultManagedL1,
	}, nil
}

func gradlePOCEdgeCacheURL(getenv func(string) string) (string, error) {
	if getenv == nil {
		return "", errors.New("resolve BUILDOPT_POC_EDGE_CACHE_URL: environment reader is unavailable")
	}
	value := getenv(gradlePOCEdgeCacheURLEnvironment)
	if value == "" {
		return "", nil
	}
	canonical, err := canonicalLoopbackHTTPOrigin(value)
	if err != nil {
		return "", errors.New("BUILDOPT_POC_EDGE_CACHE_URL must be a canonical IPv4 loopback HTTP origin")
	}
	return canonical, nil
}

func canonicalLoopbackHTTPOrigin(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.RawPath != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("invalid loopback origin")
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("invalid loopback port")
	}
	canonical := "http://127.0.0.1:" + strconv.Itoa(port)
	if value != canonical && value != canonical+"/" {
		return "", errors.New("non-canonical loopback origin")
	}
	return canonical, nil
}

func gradleStandardJarCacheEnabled(getenv func(string) string) (bool, error) {
	if getenv == nil {
		return false, errors.New("resolve BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS: environment reader is unavailable")
	}
	switch getenv(gradleStandardJarCacheEnvironment) {
	case "", "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, errors.New("BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS must be 0 or 1")
	}
}

func gradleSafeCacheEnabled(getenv func(string) string) (bool, error) {
	if getenv == nil {
		return false, errors.New("resolve BUILDOPT_SAFE_CACHE: environment reader is unavailable")
	}
	switch getenv(gradleSafeCacheEnvironment) {
	case "", "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, errors.New("BUILDOPT_SAFE_CACHE must be 0 or 1")
	}
}

func gradleInstrumentationRequested(getenv func(string) string) bool {
	if getenv == nil {
		return false
	}
	if getenv(gradleInitScriptEnvironment) != "" ||
		getenv(gradlePluginJarEnvironment) != "" {
		return true
	}
	return gradleRuntimeIntegrationRequested(getenv)
}

func gradleRuntimeIntegrationRequested(getenv func(string) string) bool {
	if getenv == nil {
		return false
	}
	for _, name := range []string{
		sessioningest.ServerURLEnvironment,
		sessioningest.ServerTokenEnvironment,
		sessioningest.ExportContextEnvironment,
		localAuthorityPathEnvironment,
		localTrustRootPathEnvironment,
		localCredentialPathEnvironment,
		sharedCacheTokenPathEnvironment,
		sharedCacheURLEnvironment,
		managedL1StateRootEnvironment,
		managedL1TenantEnvironment,
		managedL1RepositoryEnvironment,
		managedL1TrustDomainEnvironment,
		managedL1CompatibilityEnvironment,
		managedL1GenerationEnvironment,
		managedL1L2WriterEnvironment,
		gradleBootstrapConfigPathEnvironment,
	} {
		if getenv(name) != "" {
			return true
		}
	}
	return false
}

func gradleBuildCacheMode(args []string) (configured bool, enabled bool) {
	for _, argument := range args {
		switch argument {
		case "--build-cache":
			configured = true
			enabled = true
		case "--no-build-cache":
			configured = true
			enabled = false
		}
	}
	return configured, enabled
}

func useGradleLocalCacheFastPath(
	managedL1 *managedL1Config,
	serverConfigured bool,
	serverErr error,
	authorityConfigured bool,
	authorityErr error,
) bool {
	return managedL1 != nil &&
		!serverConfigured &&
		serverErr == nil &&
		!authorityConfigured &&
		authorityErr == nil
}

func defaultGradleManagedL1Config(repositoryRoot, cacheRoot string) (managedL1Config, error) {
	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return managedL1Config{}, fmt.Errorf("resolve Gradle repository root: %w", err)
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return managedL1Config{}, fmt.Errorf("make Gradle repository root absolute: %w", err)
	}
	cacheRoot, err = filepath.Abs(cacheRoot)
	if err != nil {
		return managedL1Config{}, fmt.Errorf("make user cache directory absolute: %w", err)
	}

	wrapperProperties := filepath.Join(
		canonicalRoot,
		"gradle",
		"wrapper",
		"gradle-wrapper.properties",
	)
	properties, err := os.Open(wrapperProperties)
	if err != nil {
		return managedL1Config{}, fmt.Errorf("open Gradle Wrapper properties: %w", err)
	}
	defer properties.Close()

	repositoryDigest := sha256.Sum256([]byte(canonicalRoot))
	compatibility := sha256.New()
	_, _ = io.WriteString(compatibility, "buildopt-local-l1-compatibility-v1\\x00")
	_, _ = io.WriteString(compatibility, runtime.GOOS+"\\x00"+runtime.GOARCH+"\\x00")
	if _, err := io.Copy(compatibility, properties); err != nil {
		return managedL1Config{}, fmt.Errorf("hash Gradle Wrapper properties: %w", err)
	}

	config := managedL1Config{
		stateRoot:          filepath.Join(cacheRoot, "buildopt", "state"),
		tenantID:           "local-user",
		repositoryID:       "repository-" + hex.EncodeToString(repositoryDigest[:]),
		trustDomain:        "local-machine",
		compatibilityClass: "compatibility-" + hex.EncodeToString(compatibility.Sum(nil)),
		securityGeneration: 0,
		l2WriteAuthorized:  false,
	}
	config.scopeDigest = managedL1ScopeDigest(
		config.tenantID,
		config.repositoryID,
		config.trustDomain,
		config.compatibilityClass,
	)
	return config, nil
}

func resolveGradleAssets() (string, string, error) {
	initScript := os.Getenv(gradleInitScriptEnvironment)
	pluginJar := os.Getenv(gradlePluginJarEnvironment)
	if initScript != "" || pluginJar != "" {
		if initScript == "" || pluginJar == "" {
			return "", "", errors.New(
				"BUILDOPT_GRADLE_INIT_SCRIPT and BUILDOPT_GRADLE_PLUGIN_JAR must be set together",
			)
		}
		if !regularFile(initScript) || !regularFile(pluginJar) {
			return "", "", errors.New("configured Gradle init script or plugin JAR is unavailable")
		}
		return initScript, pluginJar, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("resolve BuildOpt executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", "", fmt.Errorf("resolve BuildOpt executable symlinks: %w", err)
	}
	share := filepath.Join(filepath.Dir(filepath.Dir(executable)), "share", "buildopt")
	initScript = filepath.Join(share, "buildopt.init.gradle")
	pluginJar = filepath.Join(share, "buildopt-gradle-plugin.jar")
	if !regularFile(initScript) || !regularFile(pluginJar) {
		return "", "", errors.New(
			"packaged Gradle assets are unavailable; reinstall BuildOpt or set BUILDOPT_GRADLE_INIT_SCRIPT and BUILDOPT_GRADLE_PLUGIN_JAR",
		)
	}
	return initScript, pluginJar, nil
}

func gradleWrapperName(goos string) string {
	if goos == "windows" {
		return "gradlew.bat"
	}
	return "gradlew"
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func copyEnvironment(overrides map[string]string) map[string]string {
	if len(overrides) == 0 {
		return nil
	}
	result := make(map[string]string, len(overrides))
	for key, value := range overrides {
		result[key] = value
	}
	return result
}
