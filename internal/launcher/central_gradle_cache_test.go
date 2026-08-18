package launcher

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCentralGradleCacheConnectionIsReadOnlyAndCredentialContained(t *testing.T) {
	repository := t.TempDir()
	writeGradleWrapperProperties(
		t,
		repository,
		"distributionUrl=https://services.gradle.org/distributions/gradle-9.6.1-bin.zip\n",
	)
	connectionDirectory := filepath.Join(repository, filepath.FromSlash(centralConnectionDir))
	if err := os.MkdirAll(connectionDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	credential := bytes.Repeat([]byte{0x5a}, 32)
	token := base64.RawURLEncoding.EncodeToString(credential)
	if err := os.WriteFile(
		filepath.Join(connectionDirectory, centralTokenFile),
		[]byte(token),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	assets := t.TempDir()
	initScript := filepath.Join(assets, "buildopt.init.gradle")
	pluginJar := filepath.Join(assets, "buildopt-gradle-plugin.jar")
	for path, content := range map[string][]byte{
		initScript: []byte("// fixture\n"),
		pluginJar:  []byte("fixture"),
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(gradleInitScriptEnvironment, initScript)
	t.Setenv(gradlePluginJarEnvironment, pluginJar)

	now := time.Now().UTC().Truncate(time.Second)
	integration := &centralOptimizeIntegration{
		invocation: optimizeInvocation{
			repositoryRoot:      repository,
			connectionDirectory: connectionDirectory,
		},
		connection: centralConnection{
			ServerURL:             "https://central.example",
			RepositoryScopeSHA256: strings.Repeat("a", 64),
			TokenFile:             centralTokenFile,
			Cache: &centralCacheConnection{
				Namespace:           "gradle-9.6.1/linux-amd64/jdk-21/project",
				NamespaceGeneration: 1,
				ExpiresAt:           now.Add(time.Hour).Format(time.RFC3339Nano),
				Mode:                managedSharedReadOnlyMode,
			},
		},
		result: optimizeCentralResult{SelectionSource: "NONE"},
	}
	invocation := gradleInvocation{childArgs: []string{"./gradlew", "build"}, nativeOnly: true}
	if err := enableConnectedCentralCacheGradle(&invocation, repository); err != nil {
		t.Fatal(err)
	}
	if invocation.nativeOnly || invocation.managedL1 == nil ||
		invocation.environment[gradlePluginJarEnvironment] != pluginJar ||
		invocation.environment[gradleProjectPluginModeEnvironment] != gradleProjectPluginModeCacheOnly ||
		len(invocation.childArgs) < 4 || invocation.childArgs[1] != "--init-script" ||
		invocation.childArgs[2] != initScript {
		t.Fatalf("central Gradle invocation = %+v", invocation)
	}

	context, err := integration.centralGradleCacheContext(
		"11111111-1111-4111-8111-111111111111",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if context == nil || context.binding == nil || context.cacheClient == nil || !context.binding.allowRead ||
		context.binding.allowWrite || context.binding.namespace != integration.connection.Cache.Namespace ||
		context.childEnvironment[managedSharedModeEnvironment] != managedSharedReadOnlyMode ||
		context.childEnvironment[managedAuthorityContractEnvironment] != centralCacheAuthorityContract {
		t.Fatalf("central Gradle context = %+v", context)
	}
	for key, value := range context.childEnvironment {
		if strings.Contains(value, token) {
			t.Fatalf("central credential escaped through %s", key)
		}
	}
	if context.binding.credential != token {
		t.Fatal("gateway did not retain the exact upstream credential")
	}
}

func TestPlainGradleLoadsConnectedReadOnlyCentralCache(t *testing.T) {
	repository := t.TempDir()
	centralOptimizeGit(t, repository, "init", "-q")
	centralOptimizeGit(t, repository, "config", "user.email", "buildopt@example.invalid")
	centralOptimizeGit(t, repository, "config", "user.name", "BuildOpt fixture")
	centralOptimizeGit(t, repository, "config", "commit.gpgsign", "false")
	writeGradleWrapperProperties(
		t,
		repository,
		"distributionUrl=https://services.gradle.org/distributions/gradle-9.6.1-bin.zip\n",
	)
	centralOptimizeGit(t, repository, "add", ".")
	centralOptimizeGit(t, repository, "commit", "-qm", "fixture")

	connectionDirectory := filepath.Join(repository, filepath.FromSlash(centralConnectionDir))
	if err := os.MkdirAll(connectionDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	connection := centralConnection{
		SchemaVersion:  centralConnectionSchema,
		ServerURL:      "https://central.example",
		RepositoryID:   optimizeRepositoryID(repository, os.Getenv),
		StateDirectory: optimizeDefaultStateDir,
		TokenFile:      centralTokenFile,
		ConnectedAt:    now.Format(time.RFC3339Nano),
		Cache: &centralCacheConnection{
			Namespace:           "gradle-9.6.1/linux-amd64/jdk-21/project",
			NamespaceGeneration: 1,
			ExpiresAt:           now.Add(time.Hour).Format(time.RFC3339Nano),
			Mode:                managedSharedReadOnlyMode,
		},
		ProductionAuthorized: false,
		TestOptimization:     "OUT_OF_SCOPE",
	}
	connection.RepositoryScopeSHA256 = optimizePortfolioRepositoryScope(connection.RepositoryID)
	if err := writeCanonicalPrivateJSON(
		filepath.Join(connectionDirectory, centralConnectionFile),
		connection,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(connectionDirectory, centralTokenFile),
		[]byte(base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x44}, 32))),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	integration, err := prepareConnectedCentralGradleCache(repository)
	if err != nil {
		t.Fatal(err)
	}
	if integration == nil || !integration.hasReadOnlyCentralCache() ||
		integration.connection.RepositoryID != connection.RepositoryID ||
		integration.invocation.repositoryRoot != repository {
		t.Fatalf("plain Gradle central cache = %+v", integration)
	}
}

func TestCentralGradleCacheConnectionRejectsExpiredCredential(t *testing.T) {
	repository := t.TempDir()
	connectionDirectory := filepath.Join(repository, filepath.FromSlash(centralConnectionDir))
	if err := os.MkdirAll(connectionDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x19}, 32))
	if err := os.WriteFile(filepath.Join(connectionDirectory, centralTokenFile), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	integration := &centralOptimizeIntegration{
		invocation: optimizeInvocation{connectionDirectory: connectionDirectory},
		connection: centralConnection{
			ServerURL: "https://central.example", RepositoryScopeSHA256: strings.Repeat("b", 64),
			TokenFile: centralTokenFile,
			Cache: &centralCacheConnection{
				Namespace: "stable", NamespaceGeneration: 1,
				ExpiresAt: now.Add(-time.Second).Format(time.RFC3339Nano), Mode: managedSharedReadOnlyMode,
			},
		},
		result: optimizeCentralResult{SelectionSource: optimizeSelectionSourceCentral},
	}
	if _, err := integration.centralGradleCacheContext(
		"11111111-1111-4111-8111-111111111111",
		now,
	); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired central cache credential = %v", err)
	}
}
