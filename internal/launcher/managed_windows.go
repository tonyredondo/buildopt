//go:build windows

package launcher

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	managedGatewayStateRootEnvironment   = "BUILDOPT_GATEWAY_STATE_ROOT"
	managedRunnerSlotEnvironment         = "BUILDOPT_RUNNER_SLOT"
	managedGatewayIdleTimeoutEnvironment = "BUILDOPT_GATEWAY_IDLE_TIMEOUT"
	managedGatewayInternalCommand        = "__managed-gateway"

	managedL1StateRootEnvironment     = "BUILDOPT_L1_STATE_ROOT"
	managedL1TenantEnvironment        = "BUILDOPT_L1_TENANT_ID"
	managedL1RepositoryEnvironment    = "BUILDOPT_L1_REPOSITORY_ID"
	managedL1TrustDomainEnvironment   = "BUILDOPT_L1_TRUST_DOMAIN"
	managedL1CompatibilityEnvironment = "BUILDOPT_L1_COMPATIBILITY_CLASS"
	managedL1GenerationEnvironment    = "BUILDOPT_L1_SECURITY_GENERATION"
	managedL1L2WriterEnvironment      = "BUILDOPT_L1_L2_WRITE_AUTHORIZED"

	managedL1DirectoryChildEnvironment   = "BUILDOPT_MANAGED_L1_DIRECTORY"
	managedL1ModeChildEnvironment        = "BUILDOPT_MANAGED_L1_MODE"
	managedL1GenerationChildEnvironment  = "BUILDOPT_MANAGED_L1_SECURITY_GENERATION"
	managedL1RetentionChildEnvironment   = "BUILDOPT_MANAGED_L1_RETENTION_DAYS"
	managedL1ScopeDomain                 = "buildopt-managed-l1-scope-v1"
	managedL1MaximumIdentitySize         = 256
	gradleBootstrapConfigPathEnvironment = "BUILDOPT_GRADLE_BOOTSTRAP_CONFIG_PATH"

	gatewayCircuitFlood          gatewayCircuitReason = "FLOOD"
	gatewayCircuitObjectTooLarge gatewayCircuitReason = "OBJECT_TOO_LARGE"
	gatewayCircuitDiskPressure   gatewayCircuitReason = "DISK_PRESSURE"
)

type gatewayCircuitReason string

type managedL1Config struct {
	stateRoot          string
	tenantID           string
	repositoryID       string
	trustDomain        string
	compatibilityClass string
	securityGeneration uint64
	l2WriteAuthorized  bool
	scopeDigest        string
}

type managedL1 struct{}
type gradleBootstrapCache struct{}

func startInvocationGatewayWithCache(attemptID string, cacheBinding *gatewayCacheBinding) (*localGateway, error) {
	if cacheBinding != nil && cacheBinding.attemptID != attemptID {
		return nil, errors.New("gateway cache binding does not match the invocation attempt")
	}
	if os.Getenv(managedGatewayStateRootEnvironment) != "" || os.Getenv(managedRunnerSlotEnvironment) != "" || os.Getenv(managedGatewayIdleTimeoutEnvironment) != "" {
		return nil, errors.New("persistent managed gateway is not supported on Windows")
	}
	return startLocalGatewayWithCache(cacheBinding)
}

func runManagedGatewayProcess([]string, io.Writer) int { return 64 }

func startInvocationManagedL1() (*managedL1, error) {
	for _, key := range []string{managedL1StateRootEnvironment, managedL1TenantEnvironment, managedL1RepositoryEnvironment, managedL1TrustDomainEnvironment, managedL1CompatibilityEnvironment, managedL1GenerationEnvironment, managedL1L2WriterEnvironment} {
		if os.Getenv(key) != "" {
			return nil, errors.New("managed L1 is not supported on Windows")
		}
	}
	return nil, nil
}

func startManagedL1(managedL1Config) (*managedL1, error) {
	return nil, errors.New("managed L1 is not supported on Windows")
}

func (*managedL1) childEnvironment() map[string]string { return nil }
func (*managedL1) close() error                        { return nil }

func managedL1ScopeDigest(tenantID, repositoryID, trustDomain, compatibilityClass string) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(managedL1ScopeDomain))
	for _, value := range []string{tenantID, repositoryID, trustDomain, compatibilityClass} {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(value)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(value))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func validateManagedL1Identity(name, value string) error {
	if len(value) == 0 || len(value) > managedL1MaximumIdentitySize {
		return fmt.Errorf("%s must contain between 1 and %d bytes", name, managedL1MaximumIdentitySize)
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return fmt.Errorf("%s must contain visible ASCII only", name)
		}
	}
	return nil
}

func startInvocationGradleBootstrapCache(_ []string, _ *localAuthorityContext, getenv func(string) string) (*gradleBootstrapCache, bool, error) {
	if getenv(gradleBootstrapConfigPathEnvironment) == "" {
		return nil, false, nil
	}
	return nil, true, errors.New("managed Gradle bootstrap cache is not supported on Windows")
}

func (*gradleBootstrapCache) childEnvironment() map[string]string { return nil }
func (*gradleBootstrapCache) finalize() error                     { return nil }
func (*gradleBootstrapCache) close() error                        { return nil }

func ensurePrivateDirectory(path string, create bool) error {
	if create {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a private directory", path)
	}
	return nil
}

func openPrivateLock(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("lock path is a symlink")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, errors.New("lock is not a regular file")
	}
	return file, nil
}

func releaseManagedLock(file *os.File) error {
	if file == nil {
		return nil
	}
	return file.Close()
}

func validPluginAttemptID(identifier string) bool {
	if len(identifier) != 36 {
		return false
	}
	for index, character := range []byte(identifier) {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}
