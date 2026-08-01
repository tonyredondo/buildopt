package sharedcache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSelfHostedCapacityRequiresTwentyGiB(t *testing.T) {
	policy := CapacityPolicy{DeploymentBytes: SelfHostedMinimumDeploymentBytes}
	if err := validateSelfHostedCapacity(policy, uint64(SelfHostedMinimumDeploymentBytes)); err != nil {
		t.Fatal(err)
	}
	policy.DeploymentBytes--
	if err := validateSelfHostedCapacity(policy, uint64(SelfHostedMinimumDeploymentBytes)); err == nil {
		t.Fatal("undersized deployment accepted")
	}
	policy.DeploymentBytes = SelfHostedMinimumDeploymentBytes
	if err := validateSelfHostedCapacity(policy, uint64(SelfHostedMinimumDeploymentBytes-1)); err == nil {
		t.Fatal("insufficient available capacity accepted")
	}
}

func TestSelfHostedPreflightIsReadOnlyAndOpenUsesProductionStorage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	policy, err := ValidateSelfHostedStorageRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if policy.DeploymentBytes < SelfHostedMinimumDeploymentBytes {
		t.Fatalf("policy = %+v", policy)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("preflight mutated root: %v", err)
	}
	storage, err := OpenSelfHosted(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSelfHostedPreflightRejectsUnsafeRoots(t *testing.T) {
	if _, err := ValidateSelfHostedStorageRoot("relative"); err == nil {
		t.Fatal("relative root accepted")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateSelfHostedStorageRoot(filepath.Join(link, "state")); err == nil {
		t.Fatal("symlink ancestor accepted")
	}
	existingThroughLink := filepath.Join(target, "existing")
	if err := os.Mkdir(existingThroughLink, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateSelfHostedStorageRoot(filepath.Join(link, "existing")); err == nil {
		t.Fatal("existing directory through symlink ancestor accepted")
	}
}
