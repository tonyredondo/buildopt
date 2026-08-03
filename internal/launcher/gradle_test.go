package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrepareGradleInvocation(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	initScript := filepath.Join(root, "buildopt.init.gradle")
	pluginJar := filepath.Join(root, "buildopt-gradle-plugin.jar")
	writeGradleTestFile(t, initScript)
	writeGradleTestFile(t, pluginJar)

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	t.Setenv(gradleInitScriptEnvironment, initScript)
	t.Setenv(gradlePluginJarEnvironment, pluginJar)

	invocation, err := prepareGradleInvocation([]string{"--no-daemon", "build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{wrapper, "--init-script", initScript, "--no-daemon", "build"}
	if strings.Join(invocation.childArgs, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("child args = %q, want %q", invocation.childArgs, want)
	}
	if invocation.environment[gradlePluginJarEnvironment] != pluginJar {
		t.Fatalf("plugin environment = %q", invocation.environment)
	}
}

func TestPrepareGradleInvocationBypassNeedsNoAssets(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	t.Setenv(gradleInitScriptEnvironment, "")
	t.Setenv(gradlePluginJarEnvironment, "")

	invocation, err := prepareGradleInvocation([]string{"build"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(invocation.childArgs, "\x00") != strings.Join([]string{wrapper, "build"}, "\x00") {
		t.Fatalf("bypass child args = %q", invocation.childArgs)
	}
	if invocation.environment != nil {
		t.Fatalf("bypass environment = %q", invocation.environment)
	}
}

func TestPrepareGradleInvocationRejectsIncompleteSetup(t *testing.T) {
	root := t.TempDir()
	writeGradleTestFile(t, filepath.Join(root, gradleWrapperName(runtime.GOOS)))
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	t.Setenv(gradleInitScriptEnvironment, filepath.Join(root, "init.gradle"))
	t.Setenv(gradlePluginJarEnvironment, "")

	_, err = prepareGradleInvocation(nil, false)
	if err == nil || !strings.Contains(err.Error(), "must be set together") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrepareGradleInvocationRequiresWrapper(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	_, err = prepareGradleInvocation(nil, true)
	if err == nil || !strings.Contains(err.Error(), "Gradle repository root") {
		t.Fatalf("error = %v", err)
	}
}

func writeGradleTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("fixture\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
