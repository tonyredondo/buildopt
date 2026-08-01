//go:build linux || darwin

package launcher

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

const (
	managedL1StateRootEnvironment     = "BUILDOPT_L1_STATE_ROOT"
	managedL1TenantEnvironment        = "BUILDOPT_L1_TENANT_ID"
	managedL1RepositoryEnvironment    = "BUILDOPT_L1_REPOSITORY_ID"
	managedL1TrustDomainEnvironment   = "BUILDOPT_L1_TRUST_DOMAIN"
	managedL1CompatibilityEnvironment = "BUILDOPT_L1_COMPATIBILITY_CLASS"
	managedL1GenerationEnvironment    = "BUILDOPT_L1_SECURITY_GENERATION"
	managedL1L2WriterEnvironment      = "BUILDOPT_L1_L2_WRITE_AUTHORIZED"

	managedL1DirectoryChildEnvironment  = "BUILDOPT_MANAGED_L1_DIRECTORY"
	managedL1ModeChildEnvironment       = "BUILDOPT_MANAGED_L1_MODE"
	managedL1GenerationChildEnvironment = "BUILDOPT_MANAGED_L1_SECURITY_GENERATION"
	managedL1RetentionChildEnvironment  = "BUILDOPT_MANAGED_L1_RETENTION_DAYS"

	managedL1ReadWriteMode       = "READ_WRITE"
	managedL1DisabledWriterMode  = "DISABLED_L2_WRITER"
	managedL1ScopeDomain         = "buildopt-managed-l1-scope-v1"
	managedL1RetentionDays       = 7
	managedL1MaximumIdentitySize = 256
)

var errManagedL1Busy = errors.New("managed L1 scope is already active")

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

type managedL1 struct {
	directory          string
	mode               string
	securityGeneration uint64
	lease              *os.File
	lifecycleLease     *datalifecycle.ManagedLease
}

func managedL1ConfigFromEnvironment(
	getenv func(string) string,
) (managedL1Config, bool, error) {
	stateRoot := getenv(managedL1StateRootEnvironment)
	tenantID := getenv(managedL1TenantEnvironment)
	repositoryID := getenv(managedL1RepositoryEnvironment)
	trustDomain := getenv(managedL1TrustDomainEnvironment)
	compatibilityClass := getenv(managedL1CompatibilityEnvironment)
	generationText := getenv(managedL1GenerationEnvironment)
	l2WriterText := getenv(managedL1L2WriterEnvironment)

	if stateRoot == "" &&
		tenantID == "" &&
		repositoryID == "" &&
		trustDomain == "" &&
		compatibilityClass == "" &&
		generationText == "" &&
		l2WriterText == "" {
		return managedL1Config{}, false, nil
	}
	if stateRoot == "" ||
		tenantID == "" ||
		repositoryID == "" ||
		trustDomain == "" ||
		compatibilityClass == "" ||
		generationText == "" {
		return managedL1Config{}, true, errors.New(
			"managed L1 requires state root, tenant, repository, trust domain, compatibility class, and security generation",
		)
	}
	if !filepath.IsAbs(stateRoot) {
		return managedL1Config{}, true, errors.New(
			"BUILDOPT_L1_STATE_ROOT must be an absolute path",
		)
	}
	identities := []struct {
		name  string
		value string
	}{
		{name: managedL1TenantEnvironment, value: tenantID},
		{name: managedL1RepositoryEnvironment, value: repositoryID},
		{name: managedL1TrustDomainEnvironment, value: trustDomain},
		{name: managedL1CompatibilityEnvironment, value: compatibilityClass},
	}
	for _, identity := range identities {
		if err := validateManagedL1Identity(identity.name, identity.value); err != nil {
			return managedL1Config{}, true, err
		}
	}

	securityGeneration, err := strconv.ParseUint(generationText, 10, 63)
	if err != nil || strconv.FormatUint(securityGeneration, 10) != generationText {
		return managedL1Config{}, true, errors.New(
			"BUILDOPT_L1_SECURITY_GENERATION must be a canonical non-negative 63-bit integer",
		)
	}
	l2WriteAuthorized := false
	switch l2WriterText {
	case "", "0":
	case "1":
		l2WriteAuthorized = true
	default:
		return managedL1Config{}, true, errors.New(
			"BUILDOPT_L1_L2_WRITE_AUTHORIZED must be 0 or 1",
		)
	}

	stateRoot = filepath.Clean(stateRoot)
	return managedL1Config{
		stateRoot:          stateRoot,
		tenantID:           tenantID,
		repositoryID:       repositoryID,
		trustDomain:        trustDomain,
		compatibilityClass: compatibilityClass,
		securityGeneration: securityGeneration,
		l2WriteAuthorized:  l2WriteAuthorized,
		scopeDigest: managedL1ScopeDigest(
			tenantID,
			repositoryID,
			trustDomain,
			compatibilityClass,
		),
	}, true, nil
}

func validateManagedL1Identity(name string, value string) error {
	if len(value) == 0 || len(value) > managedL1MaximumIdentitySize {
		return fmt.Errorf(
			"%s must contain between 1 and %d bytes",
			name,
			managedL1MaximumIdentitySize,
		)
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return fmt.Errorf("%s must contain visible ASCII only", name)
		}
	}
	return nil
}

func managedL1ScopeDigest(
	tenantID string,
	repositoryID string,
	trustDomain string,
	compatibilityClass string,
) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(managedL1ScopeDomain))
	for _, value := range []string{
		tenantID,
		repositoryID,
		trustDomain,
		compatibilityClass,
	} {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(value)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(value))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func startInvocationManagedL1() (*managedL1, error) {
	config, configured, err := managedL1ConfigFromEnvironment(os.Getenv)
	if err != nil || !configured {
		return nil, err
	}
	return startManagedL1(config)
}

func startManagedL1(config managedL1Config) (*managedL1, error) {
	lifecycleLease, boundary, err := datalifecycle.AcquireManagedLease(
		config.stateRoot,
	)
	if err != nil {
		return nil, fmt.Errorf("inspect managed L1 lifecycle: %w", err)
	}
	keepLifecycleLease := false
	defer func() {
		if !keepLifecycleLease {
			_ = lifecycleLease.Close()
		}
	}()
	if config.securityGeneration < boundary.MinimumL1SecurityGeneration {
		return nil, errors.New(
			"managed L1 security generation predates managed deletion",
		)
	}
	if config.l2WriteAuthorized {
		l1 := &managedL1{
			mode:               managedL1DisabledWriterMode,
			securityGeneration: config.securityGeneration,
			lifecycleLease:     lifecycleLease,
		}
		keepLifecycleLease = true
		return l1, nil
	}

	l1Root := filepath.Join(config.stateRoot, "l1")
	scopeRoot := filepath.Join(l1Root, "scopes", config.scopeDigest)
	generationName := "generation-" +
		strconv.FormatUint(config.securityGeneration, 10)
	directory := filepath.Join(scopeRoot, generationName, "cache")
	lockRoot := filepath.Join(l1Root, "locks")
	for _, path := range []string{
		config.stateRoot,
		l1Root,
		filepath.Join(l1Root, "scopes"),
		scopeRoot,
		filepath.Join(scopeRoot, generationName),
		directory,
		lockRoot,
	} {
		if err := ensurePrivateDirectory(path, true); err != nil {
			return nil, fmt.Errorf("prepare managed L1: %w", err)
		}
	}

	lease, err := openPrivateLock(
		filepath.Join(
			lockRoot,
			config.scopeDigest+"-"+generationName+".lock",
		),
	)
	if err != nil {
		return nil, fmt.Errorf("open managed L1 lease: %w", err)
	}
	if err := syscall.Flock(
		int(lease.Fd()),
		syscall.LOCK_EX|syscall.LOCK_NB,
	); err != nil {
		_ = lease.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) ||
			errors.Is(err, syscall.EAGAIN) {
			return nil, errManagedL1Busy
		}
		return nil, fmt.Errorf("acquire managed L1 lease: %w", err)
	}

	l1 := &managedL1{
		directory:          directory,
		mode:               managedL1ReadWriteMode,
		securityGeneration: config.securityGeneration,
		lease:              lease,
		lifecycleLease:     lifecycleLease,
	}
	keepLifecycleLease = true
	return l1, nil
}

func (l1 *managedL1) childEnvironment() map[string]string {
	environment := map[string]string{
		managedL1ModeChildEnvironment:       l1.mode,
		managedL1GenerationChildEnvironment: strconv.FormatUint(l1.securityGeneration, 10),
		managedL1RetentionChildEnvironment:  strconv.Itoa(managedL1RetentionDays),
	}
	if l1.directory != "" {
		environment[managedL1DirectoryChildEnvironment] = l1.directory
	}
	return environment
}

func (l1 *managedL1) close() error {
	if l1 == nil {
		return nil
	}
	var leaseErr error
	if l1.lease != nil {
		lease := l1.lease
		l1.lease = nil
		leaseErr = releaseManagedLock(lease)
	}
	lifecycleErr := l1.lifecycleLease.Close()
	l1.lifecycleLease = nil
	return errors.Join(leaseErr, lifecycleErr)
}
