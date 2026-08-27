package launcher

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/stickyobservation"
)

func TestPrepareStickyNativeNoopPathIsConservative(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	clearStickyNativeIntegrationEnvironment(t)

	path, ok := prepareStickyNativeNoopPath(
		root,
		[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
		os.Getenv,
	)
	if !ok || path.observationMode != ordinaryObservationModeLight {
		t.Fatalf("default path = %+v, ok=%t", path, ok)
	}

	t.Setenv(sessioningest.ServerURLEnvironment, "https://buildopt.example.test")
	if _, ok := prepareStickyNativeNoopPath(
		root,
		[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
		os.Getenv,
	); ok {
		t.Fatal("explicit session integration selected native no-op")
	}
	t.Setenv(sessioningest.ServerURLEnvironment, "")
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
		pluginAttemptIDEnvironment,
		pluginSocketEnvironment,
		pluginTokenEnvironment,
		managedL1DirectoryChildEnvironment,
		managedL1ModeChildEnvironment,
		managedL1RetentionChildEnvironment,
		"BUILDOPT_RUNTIME_CHECKSTYLE_MAX_HEAP",
		"BUILDOPT_CACHE_STANDARD_COPY_TASKS",
	} {
		t.Setenv(name, "configured")
		if _, ok := prepareStickyNativeNoopPath(
			root,
			[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
			os.Getenv,
		); ok {
			t.Fatalf("explicit integration %s selected native no-op", name)
		}
		t.Setenv(name, "")
	}

	t.Setenv(stickyObservationModeEnvironment, "full")
	path, ok = prepareStickyNativeNoopPath(
		root,
		[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
		os.Getenv,
	)
	if !ok || path.observationMode != ordinaryObservationModeFull {
		t.Fatalf("full observation path = %+v, ok=%t", path, ok)
	}

	t.Setenv(stickyObservationModeEnvironment, "invalid")
	if _, ok := prepareStickyNativeNoopPath(
		root,
		[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
		os.Getenv,
	); ok {
		t.Fatal("invalid observation mode selected fast path")
	}
}

func TestPrepareStickyNativeNoopPathAllowsMissingOptionalConfig(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	if err := os.Remove(filepath.Join(root, ".buildopt", "config.toml")); err != nil {
		t.Fatal(err)
	}
	clearStickyNativeIntegrationEnvironment(t)
	t.Setenv(gradleSafeCacheEnvironment, "1")
	if _, ok := prepareStickyNativeNoopPath(
		root,
		[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
		os.Getenv,
	); ok {
		t.Fatal("explicit cache integration selected native no-op without config")
	}
	t.Setenv(gradleSafeCacheEnvironment, "")

	path, ok := prepareStickyNativeNoopPath(
		root,
		[]string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"},
		os.Getenv,
	)
	if !ok || path.observationMode != ordinaryObservationModeLight {
		t.Fatalf("missing-config path = %+v, ok=%t", path, ok)
	}
}

func TestRunStickyNativeNoopScrubsWrapperEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture uses a POSIX Gradle Wrapper-shaped script")
	}
	root := writeStickyNativeNoopFixture(t)
	marker := filepath.Join(root, "child-env.txt")
	wrapper := filepath.Join(root, "gradlew")
	if err := os.WriteFile(filepath.Join(root, ".buildopt", "config.toml"), []byte(
		"schema_version = \"buildopt.config/v1\"\n"+
			"mode = \"off\"\n"+
			"server_url = \"https://buildopt.example.test\"\n"+
			"project_scope = \"example/project\"\n"+
			"credential_env = \"BUILDOPT_TEAM_TOKEN\"\n"+
			"trial_budget_percent = 5\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrapper, []byte(
		"#!/bin/sh\n"+
			"printf '%s|%s|%s\\n' \"${BUILDOPT_STICKY_WRAPPER_ROOT-}\" \"${BUILDOPT_STICKY_OBSERVATION-}\" \"${BUILDOPT_TEAM_TOKEN-}\" > "+marker+"\n"+
			"exit 37\n",
	), 0o755); err != nil {
		t.Fatal(err)
	}
	clearStickyNativeIntegrationEnvironment(t)
	t.Setenv(stickyObservationModeEnvironment, "0")
	t.Setenv("BUILDOPT_TEAM_TOKEN", "private-value")
	t.Setenv(stickyWrapperRootEnvironment, root)

	var stdout, stderr bytes.Buffer
	if got := Run(
		[]string{"run", "--", wrapper, "help"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	); got != 37 {
		t.Fatalf("exit code = %d, want 37; stderr=%s", got, stderr.String())
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(contents)); got != "||" {
		t.Fatalf("scrubbed child environment = %q, want ||", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("native no-op emitted diagnostics: %s", stderr.String())
	}
}

func TestOrdinaryObservationCreatesRecorderOnlyAfterBuild(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "observations", "builds.jsonl")
	started := time.Now().UTC()
	state := newOrdinaryObservationStateAt(
		root,
		[]string{filepath.Join(root, "gradlew"), "help"},
		started,
	)
	if state == nil {
		t.Fatal("ordinary observation was not initialized")
	}
	state.outputPath = output
	if _, err := os.Stat(filepath.Dir(output)); !os.IsNotExist(err) {
		t.Fatalf("observation directory was created before the child: %v", err)
	}
	decision := time.Now().UTC()
	if decision.Before(started) {
		decision = started
	}
	state.markConnection(decision, false)
	state.finishCache(decision)
	completed := decision.Add(10 * time.Millisecond)
	state.finishGradle(childExecution{
		started: true, startedAt: decision, completedAt: completed,
	}, []string{"gradlew", "help"})
	if err := state.finish(0, completed); err != nil {
		t.Fatal(err)
	}
	if _, err := stickyobservation.Load(output); err != nil {
		t.Fatal(err)
	}
}

func TestLightObservationDefersExecutableDigest(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	clearStickyNativeIntegrationEnvironment(t)
	state := newOrdinaryObservationStateAt(
		root,
		[]string{filepath.Join(root, "gradlew"), "help"},
		time.Now().UTC(),
	)
	if state == nil {
		t.Fatal("ordinary observation was not initialized")
	}
	if state.buildOptHash == nil {
		t.Fatal("light observation did not defer the executable digest")
	}
	if got, want := state.record.Provenance.BuildOptSHA256, stickyobservation.Digest("unavailable-buildopt"); got != want {
		t.Fatalf("initial executable digest = %q, want unavailable placeholder %q", got, want)
	}
}

func writeStickyNativeNoopFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".buildopt"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "schema_version = \"buildopt.config/v1\"\n" +
		"mode = \"auto\"\n" +
		"server_url = \"\"\n" +
		"project_scope = \"\"\n" +
		"credential_env = \"\"\n" +
		"trial_budget_percent = 5\n"
	if err := os.WriteFile(filepath.Join(root, ".buildopt", "config.toml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, gradleWrapperName(runtime.GOOS))
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func clearStickyNativeIntegrationEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		bypassEnvironment,
		gradleInitScriptEnvironment,
		gradlePluginJarEnvironment,
		gradleProjectPluginModeEnvironment,
		gradleSafeCacheEnvironment,
		gradleStandardJarCacheEnvironment,
		gradlePOCEdgeCacheURLEnvironment,
		sessioningest.ServerURLEnvironment,
		sessioningest.ServerTokenEnvironment,
		sessioningest.ExportContextEnvironment,
		localAuthorityPathEnvironment,
		localTrustRootPathEnvironment,
		localCredentialPathEnvironment,
		sharedCacheTokenPathEnvironment,
		sharedCacheCAPathEnvironment,
		sharedCacheURLEnvironment,
		managedSharedModeEnvironment,
		managedAuthorityDigestEnvironment,
		managedPolicyDigestEnvironment,
		managedConfigurationDigestEnvironment,
		managedAuthorityContractEnvironment,
		managedSharedPolicyEnvironment,
		managedL1StateRootEnvironment,
		managedL1TenantEnvironment,
		managedL1RepositoryEnvironment,
		managedL1TrustDomainEnvironment,
		managedL1CompatibilityEnvironment,
		managedL1GenerationEnvironment,
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
		gradleBootstrapConfigPathEnvironment,
		"BUILDOPT_RUNTIME_CHECKSTYLE_MAX_HEAP",
		"BUILDOPT_CACHE_STANDARD_COPY_TASKS",
	} {
		t.Setenv(name, "")
	}
}
