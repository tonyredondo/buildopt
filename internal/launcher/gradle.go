package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	gradlePluginJarEnvironment  = "BUILDOPT_GRADLE_PLUGIN_JAR"
	gradleInitScriptEnvironment = "BUILDOPT_GRADLE_INIT_SCRIPT"
)

type gradleInvocation struct {
	childArgs   []string
	environment map[string]string
	managedL1   *managedL1Config
}

func prepareGradleInvocation(args []string, bypass bool) (gradleInvocation, error) {
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
		initScript, pluginJar, assetErr := resolveGradleAssets()
		if assetErr != nil {
			return gradleInvocation{}, assetErr
		}
		childArgs = append(childArgs, "--init-script", initScript)
		environment = map[string]string{gradlePluginJarEnvironment: pluginJar}
		if _, configured, _ := managedL1ConfigFromEnvironment(os.Getenv); !configured &&
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
		}
	}
	childArgs = append(childArgs, args...)
	return gradleInvocation{
		childArgs: childArgs, environment: environment, managedL1: defaultManagedL1,
	}, nil
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
