package runtimeoptimizer

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurationCachePromotesOnlyAfterCorrectNaturalHits(t *testing.T) {
	report := testConfigurationCacheCompatibility()
	rollout, decision, err := StartConfigurationCacheRollout(report, 2)
	if err != nil || rollout.Phase != ConfigurationCacheCanary || !decision.Enabled {
		t.Fatalf("start = %+v/%+v/%v", rollout, decision, err)
	}
	initialDigest := decision.ConfigurationPolicyDigest
	rollout, _, err = rollout.Observe(ConfigurationCacheObservation{ObservationID: "creation-1", Natural: true, EntryCreated: true})
	if err != nil || rollout.CorrectNaturalHits != 0 || rollout.Phase != ConfigurationCacheCanary {
		t.Fatalf("creation = %+v/%v", rollout, err)
	}
	rollout, _, err = rollout.Observe(ConfigurationCacheObservation{ObservationID: "hit-1", Natural: true, EntryReused: true, Correct: true})
	if err != nil || rollout.CorrectNaturalHits != 1 || rollout.Phase != ConfigurationCacheCanary {
		t.Fatalf("first hit = %+v/%v", rollout, err)
	}
	rollout, _, err = rollout.Observe(ConfigurationCacheObservation{ObservationID: "hit-1", Natural: true, EntryReused: true, Correct: true})
	if err != nil || rollout.CorrectNaturalHits != 1 || rollout.Phase != ConfigurationCacheCanary {
		t.Fatalf("replayed hit = %+v/%v", rollout, err)
	}
	rollout, promoted, err := rollout.Observe(ConfigurationCacheObservation{ObservationID: "hit-2", Natural: true, EntryReused: true, Correct: true})
	if err != nil || rollout.Phase != ConfigurationCachePromoted || !promoted.Enabled {
		t.Fatalf("promoted = %+v/%+v/%v", rollout, promoted, err)
	}
	if promoted.ConfigurationPolicyDigest != initialDigest {
		t.Fatal("rollout-only promotion invalidated a compatible local entry")
	}
}

func TestConfigurationCacheFailureDisablesAndInvalidates(t *testing.T) {
	rollout, initial, err := StartConfigurationCacheRollout(testConfigurationCacheCompatibility(), 1)
	if err != nil {
		t.Fatal(err)
	}
	rollout, decision, err := rollout.Observe(ConfigurationCacheObservation{ObservationID: "failure-1", Natural: true, EntryReused: true, AttributableFailure: true})
	if err != nil || rollout.Phase != ConfigurationCacheSuspended || decision.Enabled {
		t.Fatalf("suspended = %+v/%+v/%v", rollout, decision, err)
	}
	if decision.ConfigurationPolicyDigest == initial.ConfigurationPolicyDigest || decision.InvalidationGeneration != initial.InvalidationGeneration+1 {
		t.Fatal("attributable failure did not invalidate the decision")
	}
	_, late, err := rollout.Observe(ConfigurationCacheObservation{ObservationID: "failure-late", AttributableFailure: true})
	if err != nil || late.InvalidationGeneration != decision.InvalidationGeneration || late.ConfigurationPolicyDigest != decision.ConfigurationPolicyDigest {
		t.Fatalf("late failure changed suspended policy = %+v/%v", late, err)
	}
}

func TestConfigurationCacheCompatibilityOnlyNeverClaimsSavings(t *testing.T) {
	report := testConfigurationCacheCompatibility()
	report.PersistentWorkspace = false
	rollout, decision, err := StartConfigurationCacheRollout(report, 1)
	if err != nil || rollout.Phase != ConfigurationCacheCompatibilityOnly || rollout.ExpectedSavings || !decision.Enabled {
		t.Fatalf("compatibility only = %+v/%+v/%v", rollout, decision, err)
	}
	rollout, _, err = rollout.Observe(ConfigurationCacheObservation{ObservationID: "compatibility-1", Natural: true, EntryCreated: true})
	if err != nil || rollout.Phase != ConfigurationCacheCompatibilityOnly {
		t.Fatalf("compatibility observation = %+v/%v", rollout, err)
	}
}

func TestConfigurationCacheRejectsWarningModeAndCriticalInitialEnablement(t *testing.T) {
	report := testConfigurationCacheCompatibility()
	report.ProblemMode = "WARN"
	if _, _, err := StartConfigurationCacheRollout(report, 1); err == nil {
		t.Fatal("warning mode was accepted")
	}
	report = testConfigurationCacheCompatibility()
	report.NoncriticalCI = false
	rollout, decision, err := StartConfigurationCacheRollout(report, 1)
	if err != nil || rollout.Phase != ConfigurationCacheDisabled || decision.Enabled {
		t.Fatalf("critical initial rollout = %+v/%+v/%v", rollout, decision, err)
	}
}

func TestConfigurationCacheDecisionContainsNoLocalEntryMaterial(t *testing.T) {
	_, decision, err := StartConfigurationCacheRollout(testConfigurationCacheCompatibility(), 1)
	if err != nil {
		t.Fatal(err)
	}
	first, err := MaterializeLocalConfigurationCache(decision, LocalConfigurationCacheIdentity{
		Workspace: "/work/one", HostID: "host-1", TrustDomain: "trusted", RepositoryID: "repository-1", GradleVersion: "9.6.1", EncryptionStrategy: "machine-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := MaterializeLocalConfigurationCache(decision, LocalConfigurationCacheIdentity{
		Workspace: "/work/two", HostID: "host-2", TrustDomain: "trusted", RepositoryID: "repository-1", GradleVersion: "9.6.1", EncryptionStrategy: "machine-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Directory != filepath.Join("/work/one", ".gradle", "configuration-cache") || first.IdentityDigest == second.IdentityDigest || !first.Reusable {
		t.Fatalf("local entries = %+v/%+v", first, second)
	}
	encoded, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"/work/", ".gradle", "machine-key", "identityDigest", "encryptionStrategy"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("distributed decision contains %q: %s", forbidden, encoded)
		}
	}
}

func testConfigurationCacheCompatibility() ConfigurationCacheCompatibility {
	return ConfigurationCacheCompatibility{
		RepositoryID: "repository-1", PipelineClass: "pull-request", GradleVersion: "9.6.1", ContractVersion: "configuration-cache-v1",
		ConfigurationPolicyDigest: "sha256:" + repeat("4", 64), Compatible: true, ProblemMode: "FAIL", NoncriticalCI: true,
		PersistentWorkspace: true, PreservesEncryptionKey: true,
	}
}
