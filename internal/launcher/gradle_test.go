package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

func TestPrepareGradleInvocation(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	writeGradleWrapperProperties(t, root, "distributionUrl=gradle-9.6.1-bin.zip\n")
	clearGradleManagedL1Inputs(t)
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
	t.Setenv(gradleSafeCacheEnvironment, "1")

	invocation, err := prepareGradleInvocation([]string{"--no-daemon", "build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{wrapper, "--init-script", initScript, "--build-cache", "--no-daemon", "build"}
	if strings.Join(invocation.childArgs, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("child args = %q, want %q", invocation.childArgs, want)
	}
	if invocation.environment[gradlePluginJarEnvironment] != pluginJar {
		t.Fatalf("plugin environment = %q", invocation.environment)
	}
	if invocation.managedL1 == nil ||
		!filepath.IsAbs(invocation.managedL1.stateRoot) ||
		invocation.managedL1.repositoryID == "" ||
		invocation.managedL1.compatibilityClass == "" {
		t.Fatalf("default managed L1 = %+v", invocation.managedL1)
	}
	if invocation.nativeOnly {
		t.Fatal("explicit Safe Cache selected native-only execution")
	}
	t.Setenv(managedL1StateRootEnvironment, filepath.Join(root, "explicit-l1"))
	explicit, err := prepareGradleInvocation([]string{"build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if explicit.managedL1 != nil {
		t.Fatalf("explicit managed L1 was replaced: %+v", explicit.managedL1)
	}
	if slicesContain(explicit.childArgs, "--build-cache") {
		t.Fatalf("explicit L1 received implicit build-cache flag: %q", explicit.childArgs)
	}

	clearGradleManagedL1Inputs(t)
	disabled, err := prepareGradleInvocation([]string{"--no-build-cache", "build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.managedL1 != nil {
		t.Fatalf("disabled cache started automatic managed L1: %+v", disabled.managedL1)
	}
	if !slicesContain(disabled.childArgs, "--no-build-cache") ||
		slicesContain(disabled.childArgs, "--build-cache") {
		t.Fatalf("disabled cache args = %q", disabled.childArgs)
	}
}

func TestParseGradleProductOptions(t *testing.T) {
	t.Run("keeps ordinary Gradle arguments", func(t *testing.T) {
		arguments := []string{"--parallel", "test"}
		parsed, standardJar, err := parseGradleProductOptions(arguments)
		if err != nil || standardJar || !reflect.DeepEqual(parsed, arguments) {
			t.Fatalf("parsed = %q, standard Jar = %t, error = %v", parsed, standardJar, err)
		}
	})

	t.Run("selects the diagnostic standard Jar adapter", func(t *testing.T) {
		parsed, standardJar, err := parseGradleProductOptions([]string{
			gradleStandardJarCacheFlag, "--", "--parallel", "test",
		})
		if err != nil || !standardJar || !reflect.DeepEqual(parsed, []string{"--parallel", "test"}) {
			t.Fatalf("parsed = %q, standard Jar = %t, error = %v", parsed, standardJar, err)
		}
	})

	t.Run("requires an explicit delimiter", func(t *testing.T) {
		if _, _, err := parseGradleProductOptions([]string{
			gradleStandardJarCacheFlag, "test",
		}); err == nil || !strings.Contains(err.Error(), "requires '--'") {
			t.Fatalf("delimiter error = %v", err)
		}
	})
}

func TestPrepareGradleInvocationDefaultsToNativeCacheOnly(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	writeGradleWrapperProperties(t, root, "distributionUrl=gradle-9.6.1-bin.zip\n")
	clearGradleManagedL1Inputs(t)

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

	invocation, err := prepareGradleInvocation([]string{"--parallel", "build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{wrapper, "--build-cache", "--parallel", "build"}
	if !reflect.DeepEqual(invocation.childArgs, want) ||
		!invocation.nativeOnly || invocation.environment != nil || invocation.managedL1 != nil {
		t.Fatalf("native invocation = %+v, want args %q", invocation, want)
	}

	disabled, err := prepareGradleInvocation([]string{"--no-build-cache", "build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !disabled.nativeOnly || slicesContain(disabled.childArgs, "--build-cache") {
		t.Fatalf("native cache opt-out = %+v", disabled)
	}
}

func TestPrepareGradleInvocationRejectsInvalidSafeCacheMode(t *testing.T) {
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
	clearGradleManagedL1Inputs(t)
	t.Setenv(gradleSafeCacheEnvironment, "yes")

	if _, err := prepareGradleInvocation([]string{"build"}, false); err == nil ||
		!strings.Contains(err.Error(), "must be 0 or 1") {
		t.Fatalf("invalid Safe Cache error = %v", err)
	}
}

func TestPrepareGradleInvocationEnablesExplicitStandardJarCache(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	writeGradleWrapperProperties(t, root, "distributionUrl=gradle-9.6.1-bin.zip\n")
	clearGradleManagedL1Inputs(t)
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
	t.Setenv(gradleStandardJarCacheEnvironment, "1")

	invocation, err := prepareGradleInvocation([]string{"build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.managedL1 != nil || invocation.nativeOnly || !invocation.localOnly ||
		invocation.environment[gradleStandardJarCacheEnvironment] != "1" {
		t.Fatalf("standard Jar cache invocation = %+v", invocation)
	}

	if _, err := prepareGradleInvocation([]string{"--no-build-cache", "build"}, false); err == nil ||
		!strings.Contains(err.Error(), "incompatible with --no-build-cache") {
		t.Fatalf("disabled build cache error = %v", err)
	}
	t.Setenv(gradleStandardJarCacheEnvironment, "yes")
	if _, err := prepareGradleInvocation([]string{"build"}, false); err == nil ||
		!strings.Contains(err.Error(), "must be 0 or 1") {
		t.Fatalf("invalid standard Jar cache error = %v", err)
	}
}

func TestPrepareGradleInvocationEnablesExplicitStandardCopyCache(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	writeGradleWrapperProperties(t, root, "distributionUrl=gradle-9.6.1-bin.zip\n")
	clearGradleManagedL1Inputs(t)
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
	t.Setenv(gradleStandardCopyCacheEnvironment, "1")

	invocation, err := prepareGradleInvocation([]string{"build"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.managedL1 != nil || invocation.nativeOnly || !invocation.localOnly ||
		invocation.environment[gradleStandardCopyCacheEnvironment] != "1" {
		t.Fatalf("standard Copy cache invocation = %+v", invocation)
	}

	if _, err := prepareGradleInvocation([]string{"--no-build-cache", "build"}, false); err == nil ||
		!strings.Contains(err.Error(), "incompatible with --no-build-cache") {
		t.Fatalf("disabled build cache error = %v", err)
	}
	t.Setenv(gradleStandardCopyCacheEnvironment, "yes")
	if _, err := prepareGradleInvocation([]string{"build"}, false); err == nil ||
		!strings.Contains(err.Error(), "must be 0 or 1") {
		t.Fatalf("invalid standard Copy cache error = %v", err)
	}
}

func TestPrepareGradleInvocationSelectsLocalCheckstyleTuning(t *testing.T) {
	root := t.TempDir()
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	writeGradleTestFile(t, wrapper)
	writeGradleWrapperProperties(t, root, "distributionUrl=gradle-9.6.1-bin.zip\n")
	clearGradleManagedL1Inputs(t)
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
	t.Setenv(gradleCheckstyleHeapEnvironment, gradleCheckstyleHeap2G)
	t.Setenv(resourceAvailableMemoryEnvironment, "16659865600")

	invocation, err := prepareGradleInvocation([]string{"check"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.localOnly || invocation.nativeOnly || invocation.managedL1 != nil {
		t.Fatalf("runtime tuning invocation = %+v", invocation)
	}
	if invocation.environment[gradleCheckstyleHeapEnvironment] != gradleCheckstyleHeap2G {
		t.Fatalf("runtime tuning environment = %q", invocation.environment)
	}
	if !slicesContain(invocation.childArgs, "--init-script") ||
		!slicesContain(invocation.childArgs, "--build-cache") {
		t.Fatalf("runtime tuning args = %q", invocation.childArgs)
	}

	t.Setenv(gradleCheckstyleHeapEnvironment, "4g")
	if _, err := prepareGradleInvocation([]string{"check"}, false); err == nil ||
		!strings.Contains(err.Error(), "must be empty or 2g") {
		t.Fatalf("invalid Checkstyle heap error = %v", err)
	}
	t.Setenv(gradleCheckstyleHeapEnvironment, gradleCheckstyleHeap2G)
	t.Setenv(resourceAvailableMemoryEnvironment, "15032385535")
	if _, err := prepareGradleInvocation([]string{"check"}, false); err == nil ||
		!strings.Contains(err.Error(), "at least 14 GiB") {
		t.Fatalf("insufficient Checkstyle memory error = %v", err)
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
	if invocation.environment != nil || invocation.managedL1 != nil {
		t.Fatalf("bypass integration = %q/%+v", invocation.environment, invocation.managedL1)
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

func TestGradleLocalCacheFastPathRequiresUnconfiguredHealthyControlPlane(t *testing.T) {
	config := &managedL1Config{}
	if !useGradleLocalCacheFastPath(config, false, nil, false, nil) {
		t.Fatal("default local cache did not select the fast path")
	}
	tests := []struct {
		name                string
		managedL1           *managedL1Config
		serverConfigured    bool
		serverErr           error
		authorityConfigured bool
		authorityErr        error
	}{
		{name: "no automatic cache"},
		{name: "session ingest", managedL1: config, serverConfigured: true},
		{name: "invalid session ingest", managedL1: config, serverErr: errors.New("invalid")},
		{name: "local authority", managedL1: config, authorityConfigured: true},
		{name: "invalid local authority", managedL1: config, authorityErr: errors.New("invalid")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if useGradleLocalCacheFastPath(test.managedL1, test.serverConfigured, test.serverErr, test.authorityConfigured, test.authorityErr) {
				t.Fatal("instrumented invocation selected the local-cache fast path")
			}
		})
	}
}

func TestDefaultGradleManagedL1ConfigIsStableAndSeparated(t *testing.T) {
	cacheRoot := t.TempDir()
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	writeGradleWrapperProperties(t, firstRoot, "distributionUrl=gradle-9.6.1-bin.zip\n")
	writeGradleWrapperProperties(t, secondRoot, "distributionUrl=gradle-9.6.1-bin.zip\n")

	first, err := defaultGradleManagedL1Config(firstRoot, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := defaultGradleManagedL1Config(firstRoot, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first != repeated {
		t.Fatalf("default managed L1 drifted: %+v/%+v", first, repeated)
	}

	second, err := defaultGradleManagedL1Config(secondRoot, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first.repositoryID == second.repositoryID ||
		first.scopeDigest == second.scopeDigest {
		t.Fatalf("repository scopes collided: %+v/%+v", first, second)
	}

	writeGradleWrapperProperties(t, firstRoot, "distributionUrl=gradle-8.14.3-bin.zip\n")
	changed, err := defaultGradleManagedL1Config(firstRoot, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first.compatibilityClass == changed.compatibilityClass ||
		first.scopeDigest == changed.scopeDigest {
		t.Fatalf("Wrapper change reused compatibility scope: %+v/%+v", first, changed)
	}
}

func writeGradleWrapperProperties(t *testing.T, root, contents string) {
	t.Helper()
	path := filepath.Join(root, "gradle", "wrapper", "gradle-wrapper.properties")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func clearGradleManagedL1Inputs(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		gradleSafeCacheEnvironment,
		gradleStandardJarCacheEnvironment,
		gradleStandardCopyCacheEnvironment,
		gradleCheckstyleHeapEnvironment,
		sessioningest.ServerURLEnvironment,
		sessioningest.ServerTokenEnvironment,
		sessioningest.ExportContextEnvironment,
		localAuthorityPathEnvironment,
		localTrustRootPathEnvironment,
		localCredentialPathEnvironment,
		sharedCacheTokenPathEnvironment,
		sharedCacheURLEnvironment,
		gradleBootstrapConfigPathEnvironment,
		managedL1StateRootEnvironment,
		managedL1TenantEnvironment,
		managedL1RepositoryEnvironment,
		managedL1TrustDomainEnvironment,
		managedL1CompatibilityEnvironment,
		managedL1GenerationEnvironment,
		managedL1L2WriterEnvironment,
	} {
		t.Setenv(key, "")
	}
}

func slicesContain(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeGradleTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("fixture\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
