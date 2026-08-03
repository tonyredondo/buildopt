package launcher

import (
	"errors"
	"fmt"
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
	if !bypass {
		initScript, pluginJar, assetErr := resolveGradleAssets()
		if assetErr != nil {
			return gradleInvocation{}, assetErr
		}
		childArgs = append(childArgs, "--init-script", initScript)
		environment = map[string]string{gradlePluginJarEnvironment: pluginJar}
	}
	childArgs = append(childArgs, args...)
	return gradleInvocation{childArgs: childArgs, environment: environment}, nil
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
