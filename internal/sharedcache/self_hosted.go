package sharedcache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const SelfHostedMinimumDeploymentBytes int64 = 20 << 30

// OpenSelfHosted rejects an unsupported or undersized volume before the
// single-node server can expose a listener.
func OpenSelfHosted(ctx context.Context, root string) (*Storage, error) {
	if _, err := ValidateSelfHostedStorageRoot(root); err != nil {
		return nil, err
	}
	return Open(ctx, root)
}

// ValidateSelfHostedStorageRoot applies the A2 local-filesystem and effective
// capacity boundary against the nearest existing ancestor without mutation.
func ValidateSelfHostedStorageRoot(root string) (CapacityPolicy, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return CapacityPolicy{}, errors.New("self-hosted storage root must be absolute, clean, and non-root")
	}
	existing := root
	for {
		info, err := os.Lstat(existing)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return CapacityPolicy{}, errors.New("self-hosted storage ancestor must be a real directory")
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return CapacityPolicy{}, err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return CapacityPolicy{}, errors.New("self-hosted storage has no existing ancestor")
		}
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return CapacityPolicy{}, err
	}
	if resolved != existing {
		return CapacityPolicy{}, errors.New("self-hosted storage ancestors must not contain symlinks")
	}
	if err := validateLocalStorageFilesystem(existing); err != nil {
		return CapacityPolicy{}, fmt.Errorf("validate self-hosted filesystem: %w", err)
	}
	policy, err := defaultCapacityPolicy(existing, MaximumBlobBytes)
	if err != nil {
		return CapacityPolicy{}, err
	}
	_, available, err := storageDiskCapacity(existing)
	if err != nil {
		return CapacityPolicy{}, err
	}
	if err := validateSelfHostedCapacity(policy, available); err != nil {
		return CapacityPolicy{}, err
	}
	return policy, nil
}

func validateSelfHostedCapacity(policy CapacityPolicy, available uint64) error {
	if policy.DeploymentBytes < SelfHostedMinimumDeploymentBytes || available < uint64(SelfHostedMinimumDeploymentBytes) {
		return errors.New("self-hosted storage requires at least 20 GiB effective and currently available capacity")
	}
	return nil
}
