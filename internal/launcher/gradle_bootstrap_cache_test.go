package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	gradleBootstrapTestPolicyDigest = "sha256:" +
		"3333333333333333333333333333333333333333333333333333333333333333"
	gradleBootstrapTestScopeDigest = "sha256:" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

type gradleBootstrapFixture struct {
	config     gradleBootstrapConfigDocument
	authority  *localAuthorityContext
	childArgs  []string
	project    string
	archiveRaw []byte
}

func TestGradleBootstrapCacheLifecycle(t *testing.T) {
	fixture := newGradleBootstrapFixture(t, "9.6.1", nil)
	cache, err := startGradleBootstrapCache(
		fixture.config,
		fixture.childArgs,
		fixture.authority,
	)
	if err != nil {
		t.Fatalf("start managed Gradle bootstrap cache: %v", err)
	}

	environment := cache.childEnvironment()
	if environment[gradleUserHomeEnvironment] != cache.userHome ||
		environment[gradleReadOnlyDependencyEnvironment] !=
			fixture.config.DependencyCacheRoot {
		t.Fatalf("unexpected Gradle child environment: %+v", environment)
	}
	if info, err := os.Stat(cache.userHome); err != nil ||
		info.Mode().Perm() != 0o700 {
		t.Fatalf("private Gradle user home = %v/%v", info, err)
	}
	location, err := cache.distributionLocation()
	if err != nil {
		t.Fatalf("resolve distribution location: %v", err)
	}
	copied, err := os.ReadFile(location.archive)
	if err != nil || string(copied) != string(fixture.archiveRaw) {
		t.Fatalf("copied distribution = %q/%v", copied, err)
	}

	contender, err := startGradleBootstrapCache(
		fixture.config,
		fixture.childArgs,
		fixture.authority,
	)
	if !errors.Is(err, errGradleBootstrapBusy) || contender != nil {
		t.Fatalf("concurrent cache = %+v/%v, want busy", contender, err)
	}

	writeFakeGradleInstallation(t, location)
	if err := cache.finalize(); err != nil {
		t.Fatalf("finalize Gradle Wrapper installation: %v", err)
	}
	if err := cache.close(); err != nil {
		t.Fatalf("close managed Gradle bootstrap cache: %v", err)
	}
	markerPath := filepath.Join(cache.scopeRoot, "wrapper-install.json")
	if info, err := os.Stat(markerPath); err != nil ||
		info.Mode().Perm() != 0o600 {
		t.Fatalf("wrapper marker = %v/%v", info, err)
	}

	if err := os.Remove(fixture.config.DistributionArchivePath); err != nil {
		t.Fatalf("remove source archive after installation: %v", err)
	}
	reused, err := startGradleBootstrapCache(
		fixture.config,
		fixture.childArgs,
		fixture.authority,
	)
	if err != nil {
		t.Fatalf("reuse installed Gradle distribution: %v", err)
	}
	if !reused.markerReady {
		t.Fatal("reused Gradle distribution did not load its policy marker")
	}
	if err := reused.close(); err != nil {
		t.Fatalf("close reused managed Gradle bootstrap cache: %v", err)
	}
}

func TestInvocationGradleBootstrapCacheRequiresSignedAuthorityAndPrivateConfig(
	t *testing.T,
) {
	fixture := newGradleBootstrapFixture(t, "8.14.3", nil)
	configPath := filepath.Join(filepath.Dir(fixture.config.StateRoot), "config.json")
	if err := writeCanonicalPrivateJSON(configPath, fixture.config); err != nil {
		t.Fatalf("write bootstrap config: %v", err)
	}
	getenv := func(key string) string {
		if key == gradleBootstrapConfigPathEnvironment {
			return configPath
		}
		return ""
	}

	unauthorized := *fixture.authority
	unauthorized.dependencyCacheAuthorized = false
	cache, configured, err := startInvocationGradleBootstrapCache(
		fixture.childArgs,
		&unauthorized,
		getenv,
	)
	if err == nil || !configured || cache != nil {
		t.Fatalf(
			"unsigned dependency cache = %+v/%t/%v",
			cache,
			configured,
			err,
		)
	}

	cache, configured, err = startInvocationGradleBootstrapCache(
		fixture.childArgs,
		fixture.authority,
		getenv,
	)
	if err != nil || !configured || cache == nil {
		t.Fatalf("signed dependency cache = %+v/%t/%v", cache, configured, err)
	}
	if err := cache.close(); err != nil {
		t.Fatalf("close signed dependency cache: %v", err)
	}
}

func TestRunAppliesAndFinalizesManagedGradleBootstrapCache(t *testing.T) {
	clearManagedGatewayTestEnvironment(t)
	fixture := newGradleBootstrapFixture(t, "9.6.1", nil)
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	authorityEnvironment, _ := writeLauncherAuthorityFixtureAt(
		t,
		upstream.URL,
		time.Now().UTC(),
	)
	for key, value := range authorityEnvironment {
		t.Setenv(key, value)
	}
	configPath := filepath.Join(
		filepath.Dir(fixture.config.StateRoot),
		"run-config.json",
	)
	if err := writeCanonicalPrivateJSON(configPath, fixture.config); err != nil {
		t.Fatalf("write Run bootstrap config: %v", err)
	}
	t.Setenv(gradleBootstrapConfigPathEnvironment, configPath)

	outputPath := filepath.Join(
		filepath.Dir(fixture.config.StateRoot),
		"child-environment.txt",
	)
	script := fmt.Sprintf(
		"set -eu\n"+
			"archive=$(find \"$GRADLE_USER_HOME/wrapper/dists\" -type f -name 'gradle-9.6.1-bin.zip' -print -quit)\n"+
			"test -n \"$archive\"\n"+
			"distribution_root=${archive%%/*}\n"+
			"mkdir -p \"$distribution_root/gradle-9.6.1/bin\"\n"+
			"printf '#!/bin/sh\\nexit 0\\n' > \"$distribution_root/gradle-9.6.1/bin/gradle\"\n"+
			"chmod 700 \"$distribution_root/gradle-9.6.1/bin/gradle\"\n"+
			": > \"$archive.ok\"\n"+
			"printf '%%s\\n%%s\\n%%s\\n' \"$GRADLE_USER_HOME\" \"$GRADLE_RO_DEP_CACHE\" \"${BUILDOPT_GRADLE_BOOTSTRAP_CONFIG_PATH-absent}\" > %q\n",
		outputPath,
	)
	if err := os.WriteFile(
		fixture.childArgs[0],
		[]byte("#!/bin/sh\n"+script),
		0o700,
	); err != nil {
		t.Fatalf("write lifecycle Wrapper command: %v", err)
	}

	var stderr bytes.Buffer
	exitCode := Run(
		[]string{"run", "--", fixture.childArgs[0]},
		strings.NewReader(""),
		&bytes.Buffer{},
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf(
			"managed Gradle lifecycle exit = %d; stderr=%q",
			exitCode,
			stderr.String(),
		)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read child Gradle environment: %v", err)
	}
	values := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(values) != 3 ||
		values[0] == "" ||
		values[1] != fixture.config.DependencyCacheRoot ||
		values[2] != "absent" {
		t.Fatalf("unexpected child Gradle environment: %q", raw)
	}
	scopeRoot := filepath.Dir(values[0])
	if _, err := os.Stat(
		filepath.Join(scopeRoot, "wrapper-install.json"),
	); err != nil {
		t.Fatalf("Run did not retain Wrapper installation: %v", err)
	}
}

func TestGradleBootstrapCacheRejectsUnsafeInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *gradleBootstrapFixture)
	}{
		{
			name: "policy mismatch",
			mutate: func(_ *testing.T, fixture *gradleBootstrapFixture) {
				fixture.authority.configurationPolicyDigest = "sha256:" +
					strings.Repeat("4", 64)
			},
		},
		{
			name: "writable shared root",
			mutate: func(t *testing.T, fixture *gradleBootstrapFixture) {
				if err := os.Chmod(
					fixture.config.DependencyCacheRoot,
					0o700,
				); err != nil {
					t.Fatalf("make shared root writable: %v", err)
				}
			},
		},
		{
			name: "shared lock file",
			mutate: func(t *testing.T, fixture *gradleBootstrapFixture) {
				modulesRoot := filepath.Join(
					fixture.config.DependencyCacheRoot,
					"modules-2",
				)
				if err := os.Chmod(modulesRoot, 0o700); err != nil {
					t.Fatalf("make modules root writable: %v", err)
				}
				if err := os.WriteFile(
					filepath.Join(modulesRoot, "modules-2.lock"),
					nil,
					0o400,
				); err != nil {
					t.Fatalf("write shared lock: %v", err)
				}
				if err := os.Chmod(modulesRoot, 0o500); err != nil {
					t.Fatalf("restore modules root mode: %v", err)
				}
			},
		},
		{
			name: "writable shared descendant",
			mutate: func(t *testing.T, fixture *gradleBootstrapFixture) {
				modulesRoot := filepath.Join(
					fixture.config.DependencyCacheRoot,
					"modules-2",
				)
				if err := os.Chmod(modulesRoot, 0o700); err != nil {
					t.Fatalf("make modules root writable: %v", err)
				}
			},
		},
		{
			name: "distribution digest mismatch",
			mutate: func(t *testing.T, fixture *gradleBootstrapFixture) {
				if err := os.Chmod(
					fixture.config.DistributionArchivePath,
					0o600,
				); err != nil {
					t.Fatalf("make archive writable: %v", err)
				}
				if err := os.WriteFile(
					fixture.config.DistributionArchivePath,
					[]byte("different archive"),
					0o400,
				); err != nil {
					t.Fatalf("replace distribution archive: %v", err)
				}
			},
		},
		{
			name: "non-wrapper command",
			mutate: func(_ *testing.T, fixture *gradleBootstrapFixture) {
				fixture.childArgs = []string{"/bin/true"}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newGradleBootstrapFixture(t, "9.6.1", nil)
			test.mutate(t, fixture)
			cache, err := startGradleBootstrapCache(
				fixture.config,
				fixture.childArgs,
				fixture.authority,
			)
			if err == nil || cache != nil {
				if cache != nil {
					_ = cache.close()
				}
				t.Fatalf("unsafe input = %+v/%v, want rejection", cache, err)
			}
		})
	}
}

func TestGradleBootstrapCacheRealGradle(t *testing.T) {
	version := os.Getenv("BUILDOPT_TEST_GRADLE_VERSION")
	gradleHome := os.Getenv("BUILDOPT_TEST_GRADLE_HOME")
	distributionArchive := os.Getenv("BUILDOPT_TEST_GRADLE_ARCHIVE")
	javaHome := os.Getenv("BUILDOPT_TEST_JAVA_HOME")
	if version == "" ||
		gradleHome == "" ||
		distributionArchive == "" ||
		javaHome == "" {
		t.Skip("real Gradle bootstrap inputs are not configured")
	}
	if !supportedGradleBootstrapVersion(version) {
		t.Fatalf("unsupported real Gradle version %q", version)
	}

	root := t.TempDir()
	t.Cleanup(func() {
		if err := makeTreeWritableForCleanup(root); err != nil {
			t.Errorf("restore fixture permissions for cleanup: %v", err)
		}
	})
	project := filepath.Join(root, "project")
	repository := filepath.Join(root, "repository")
	writeDependencyFixture(t, project, repository)
	seedHome := filepath.Join(root, "seed-home")
	if err := os.Mkdir(seedHome, 0o700); err != nil {
		t.Fatalf("create Gradle seed home: %v", err)
	}
	runGradleCommand(
		t,
		filepath.Join(gradleHome, "bin", "gradle"),
		project,
		javaHome,
		seedHome,
		"",
		"verifyDependency",
	)

	sharedRoot := filepath.Join(root, "shared-dependencies")
	modulesRoot := filepath.Join(sharedRoot, "modules-2")
	if err := copyGradleCacheTree(
		filepath.Join(seedHome, "caches", "modules-2"),
		modulesRoot,
	); err != nil {
		t.Fatalf("copy Gradle dependency cache: %v", err)
	}
	manifest := gradleDependencyCacheManifest{
		SchemaVersion:             gradleDependencyManifestVersion,
		GradleVersion:             version,
		CompatibilityClass:        "gradle-" + version + "-dependency-format-v1",
		ConfigurationPolicyDigest: gradleBootstrapTestPolicyDigest,
		SnapshotID:                "fixture-" + version,
	}
	if err := writeCanonicalPrivateJSON(
		filepath.Join(sharedRoot, gradleDependencyManifestName),
		manifest,
	); err != nil {
		t.Fatalf("write dependency manifest: %v", err)
	}
	if err := makeTreeReadOnly(sharedRoot); err != nil {
		t.Fatalf("make dependency cache read-only: %v", err)
	}

	wrapperRoot := filepath.Join(project, "gradle", "wrapper")
	if err := os.MkdirAll(wrapperRoot, 0o755); err != nil {
		t.Fatalf("create Wrapper directory: %v", err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	wrapperJarPath := filepath.Join(wrapperRoot, "gradle-wrapper.jar")
	if err := copyRegularFile(
		filepath.Join(repositoryRoot, "gradle", "wrapper", "gradle-wrapper.jar"),
		wrapperJarPath,
		0o600,
	); err != nil {
		t.Fatalf("copy Wrapper JAR: %v", err)
	}
	wrapperScript := filepath.Join(project, "gradlew")
	if err := copyRegularFile(
		filepath.Join(repositoryRoot, "gradlew"),
		wrapperScript,
		0o700,
	); err != nil {
		t.Fatalf("copy Wrapper command: %v", err)
	}
	sourceArchive := filepath.Join(
		root,
		"gradle-"+version+"-bin.zip",
	)
	if err := copyRegularFile(
		distributionArchive,
		sourceArchive,
		0o400,
	); err != nil {
		t.Fatalf("copy distribution archive: %v", err)
	}
	distributionDigest := regularFileSHA256(t, sourceArchive)
	wrapperDigest := "sha256:" + regularFileSHA256(t, wrapperJarPath)
	propertiesPath := filepath.Join(
		wrapperRoot,
		"gradle-wrapper.properties",
	)
	writeWrapperProperties(t, propertiesPath, version, distributionDigest)

	config := gradleBootstrapConfigDocument{
		SchemaVersion:           gradleBootstrapConfigVersion,
		StateRoot:               filepath.Join(root, "state"),
		RunnerSlot:              "real-runner",
		CompatibilityClass:      manifest.CompatibilityClass,
		DependencyCacheRoot:     sharedRoot,
		WrapperPropertiesPath:   propertiesPath,
		DistributionArchivePath: sourceArchive,
		WrapperJarDigest:        wrapperDigest,
	}
	authority := &localAuthorityContext{
		authorityScopeDigest:      gradleBootstrapTestScopeDigest,
		configurationPolicyDigest: gradleBootstrapTestPolicyDigest,
		dependencyCacheAuthorized: true,
	}
	childArgs := []string{wrapperScript}
	cache, err := startGradleBootstrapCache(config, childArgs, authority)
	if err != nil {
		t.Fatalf("start real Gradle bootstrap cache: %v", err)
	}
	runGradleCommand(
		t,
		wrapperScript,
		project,
		javaHome,
		cache.userHome,
		cache.dependencyCacheRoot,
		"--offline",
		"verifyDependency",
	)
	if err := cache.finalize(); err != nil {
		t.Fatalf("retain real Wrapper installation: %v", err)
	}
	if err := cache.close(); err != nil {
		t.Fatalf("close real Gradle bootstrap cache: %v", err)
	}
	if pathExists(filepath.Join(
		cache.userHome,
		"caches",
		"modules-2",
		"files-2.1",
		"com.example",
		"fixture",
		"1.0",
	)) {
		t.Fatal("read-only dependency artifact was copied into the writable cache")
	}

	if err := os.Remove(sourceArchive); err != nil {
		t.Fatalf("remove source distribution archive: %v", err)
	}
	reused, err := startGradleBootstrapCache(config, childArgs, authority)
	if err != nil {
		t.Fatalf("reuse real Wrapper installation without source archive: %v", err)
	}
	runGradleCommand(
		t,
		wrapperScript,
		project,
		javaHome,
		reused.userHome,
		reused.dependencyCacheRoot,
		"--offline",
		"verifyDependency",
	)
	if err := reused.close(); err != nil {
		t.Fatalf("close reused real Gradle bootstrap cache: %v", err)
	}
}

func newGradleBootstrapFixture(
	t *testing.T,
	version string,
	mutateManifest func(*gradleDependencyCacheManifest),
) *gradleBootstrapFixture {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() {
		if err := makeTreeWritableForCleanup(root); err != nil {
			t.Errorf("restore fixture permissions for cleanup: %v", err)
		}
	})
	project := filepath.Join(root, "project")
	wrapperRoot := filepath.Join(project, "gradle", "wrapper")
	if err := os.MkdirAll(wrapperRoot, 0o755); err != nil {
		t.Fatalf("create Wrapper fixture: %v", err)
	}
	wrapperCommand := filepath.Join(project, "gradlew")
	if err := os.WriteFile(
		wrapperCommand,
		[]byte("#!/bin/sh\nexit 0\n"),
		0o700,
	); err != nil {
		t.Fatalf("write Wrapper command: %v", err)
	}
	wrapperRaw := []byte("fixture-wrapper")
	wrapperPath := filepath.Join(wrapperRoot, "gradle-wrapper.jar")
	if err := os.WriteFile(wrapperPath, wrapperRaw, 0o600); err != nil {
		t.Fatalf("write Wrapper JAR: %v", err)
	}
	archiveRaw := []byte("fixture-distribution-" + version)
	archivePath := filepath.Join(root, "gradle-"+version+"-bin.zip")
	if err := os.WriteFile(archivePath, archiveRaw, 0o400); err != nil {
		t.Fatalf("write distribution archive: %v", err)
	}
	archiveDigest := sha256.Sum256(archiveRaw)
	propertiesPath := filepath.Join(
		wrapperRoot,
		"gradle-wrapper.properties",
	)
	writeWrapperProperties(
		t,
		propertiesPath,
		version,
		hex.EncodeToString(archiveDigest[:]),
	)

	dependencyRoot := filepath.Join(root, "dependencies")
	modulesRoot := filepath.Join(dependencyRoot, "modules-2")
	if err := os.MkdirAll(modulesRoot, 0o700); err != nil {
		t.Fatalf("create dependency cache: %v", err)
	}
	manifest := gradleDependencyCacheManifest{
		SchemaVersion:             gradleDependencyManifestVersion,
		GradleVersion:             version,
		CompatibilityClass:        "gradle-" + version + "-dependency-format-v1",
		ConfigurationPolicyDigest: gradleBootstrapTestPolicyDigest,
		SnapshotID:                "fixture-" + version,
	}
	if mutateManifest != nil {
		mutateManifest(&manifest)
	}
	if err := writeCanonicalPrivateJSON(
		filepath.Join(dependencyRoot, gradleDependencyManifestName),
		manifest,
	); err != nil {
		t.Fatalf("write dependency cache manifest: %v", err)
	}
	if err := makeTreeReadOnly(dependencyRoot); err != nil {
		t.Fatalf("make fixture dependency cache read-only: %v", err)
	}
	wrapperHash := sha256.Sum256(wrapperRaw)
	return &gradleBootstrapFixture{
		config: gradleBootstrapConfigDocument{
			SchemaVersion:           gradleBootstrapConfigVersion,
			StateRoot:               filepath.Join(root, "state"),
			RunnerSlot:              "runner-01",
			CompatibilityClass:      manifest.CompatibilityClass,
			DependencyCacheRoot:     dependencyRoot,
			WrapperPropertiesPath:   propertiesPath,
			DistributionArchivePath: archivePath,
			WrapperJarDigest: "sha256:" +
				hex.EncodeToString(wrapperHash[:]),
		},
		authority: &localAuthorityContext{
			authorityScopeDigest:      gradleBootstrapTestScopeDigest,
			configurationPolicyDigest: gradleBootstrapTestPolicyDigest,
			dependencyCacheAuthorized: true,
		},
		childArgs:  []string{wrapperCommand},
		project:    project,
		archiveRaw: archiveRaw,
	}
}

func writeWrapperProperties(
	t *testing.T,
	path string,
	version string,
	distributionDigest string,
) {
	t.Helper()
	content := fmt.Sprintf(
		"distributionBase=GRADLE_USER_HOME\n"+
			"distributionPath=wrapper/dists\n"+
			"distributionSha256Sum=%s\n"+
			"distributionUrl=https\\://services.gradle.org/distributions/gradle-%s-bin.zip\n"+
			"networkTimeout=30000\n"+
			"retries=3\n"+
			"retryBackOffMs=1000\n"+
			"validateDistributionUrl=true\n"+
			"zipStoreBase=GRADLE_USER_HOME\n"+
			"zipStorePath=wrapper/dists\n",
		distributionDigest,
		version,
	)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write Wrapper properties: %v", err)
	}
}

func writeFakeGradleInstallation(
	t *testing.T,
	location gradleDistributionLocation,
) {
	t.Helper()
	bin := filepath.Join(location.gradleHome, "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatalf("create fake Gradle distribution: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(bin, "gradle"),
		[]byte("#!/bin/sh\nexit 0\n"),
		0o700,
	); err != nil {
		t.Fatalf("write fake Gradle executable: %v", err)
	}
	if err := os.WriteFile(location.okMarker, nil, 0o644); err != nil {
		t.Fatalf("write Gradle .ok marker: %v", err)
	}
}

func writeDependencyFixture(
	t *testing.T,
	project string,
	repository string,
) {
	t.Helper()
	module := filepath.Join(
		repository,
		"com",
		"example",
		"fixture",
		"1.0",
	)
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatalf("create fixture Maven module: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(module, "fixture-1.0.jar"),
		[]byte("dependency-payload"),
		0o644,
	); err != nil {
		t.Fatalf("write fixture dependency: %v", err)
	}
	pom := "<project><modelVersion>4.0.0</modelVersion>" +
		"<groupId>com.example</groupId><artifactId>fixture</artifactId>" +
		"<version>1.0</version></project>"
	if err := os.WriteFile(
		filepath.Join(module, "fixture-1.0.pom"),
		[]byte(pom),
		0o644,
	); err != nil {
		t.Fatalf("write fixture POM: %v", err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("create fixture project: %v", err)
	}
	build := fmt.Sprintf(
		"repositories { maven { url = uri(%q) } }\n"+
			"configurations { probe }\n"+
			"dependencies { probe 'com.example:fixture:1.0' }\n"+
			"tasks.register('verifyDependency') {\n"+
			"  doLast {\n"+
			"    def files = configurations.probe.files\n"+
			"    assert files.size() == 1\n"+
			"    assert files.first().text == 'dependency-payload'\n"+
			"  }\n"+
			"}\n",
		filepath.ToSlash(repository),
	)
	if err := os.WriteFile(
		filepath.Join(project, "build.gradle"),
		[]byte(build),
		0o644,
	); err != nil {
		t.Fatalf("write fixture build: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(project, "settings.gradle"),
		[]byte("rootProject.name = 'dependency-cache-fixture'\n"),
		0o644,
	); err != nil {
		t.Fatalf("write fixture settings: %v", err)
	}
}

func runGradleCommand(
	t *testing.T,
	executable string,
	project string,
	javaHome string,
	userHome string,
	readOnlyDependencyCache string,
	arguments ...string,
) {
	t.Helper()
	command := exec.Command(executable, append(
		[]string{"--no-daemon", "--no-configuration-cache", "--stacktrace"},
		arguments...,
	)...)
	command.Dir = project
	overrides := map[string]string{
		"JAVA_HOME":               javaHome,
		gradleUserHomeEnvironment: userHome,
	}
	if readOnlyDependencyCache != "" {
		overrides[gradleReadOnlyDependencyEnvironment] = readOnlyDependencyCache
	}
	command.Env = replaceEnvironment(os.Environ(), overrides)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Gradle command failed: %v\n%s", err, output)
	}
}

func copyGradleCacheTree(source string, destination string) error {
	return filepath.Walk(source, func(
		path string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported dependency cache entry %s", path)
		}
		if strings.HasSuffix(info.Name(), ".lock") {
			return nil
		}
		return copyRegularFile(path, target, 0o600)
	})
}

func copyRegularFile(source string, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("copy source is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	output, err := os.OpenFile(
		destination,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		mode,
	)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	return errors.Join(copyErr, output.Close())
}

func makeTreeReadOnly(root string) error {
	var directories []string
	err := filepath.Walk(root, func(
		path string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			directories = append(directories, path)
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported read-only cache entry %s", path)
		}
		return os.Chmod(path, 0o400)
	})
	if err != nil {
		return err
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Chmod(directories[index], 0o500); err != nil {
			return err
		}
	}
	return nil
}

func makeTreeWritableForCleanup(root string) error {
	return filepath.Walk(root, func(
		path string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return os.Chmod(path, 0o700)
		}
		if info.Mode().IsRegular() {
			return os.Chmod(path, 0o600)
		}
		return nil
	})
}

func regularFileSHA256(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open digest source: %v", err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		t.Fatalf("hash digest source: %v", err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
