package launcher

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	localAuthorityPathEnvironment  = "BUILDOPT_LOCAL_AUTHORITY_PATH"
	localTrustRootPathEnvironment  = "BUILDOPT_LOCAL_TRUST_ROOT_PATH"
	localCredentialPathEnvironment = "BUILDOPT_LOCAL_CACHE_CREDENTIAL_PATH"
	sharedCacheURLEnvironment      = "BUILDOPT_SHARED_CACHE_URL"

	managedSharedModeEnvironment          = "BUILDOPT_MANAGED_SHARED_MODE"
	managedAuthorityDigestEnvironment     = "BUILDOPT_MANAGED_AUTHORITY_DIGEST"
	managedPolicyDigestEnvironment        = "BUILDOPT_MANAGED_POLICY_DIGEST"
	managedConfigurationDigestEnvironment = "BUILDOPT_MANAGED_CONFIGURATION_POLICY_DIGEST"
	managedAuthorityContractEnvironment   = "BUILDOPT_MANAGED_AUTHORITY_CONTRACT"

	managedSharedReadOnlyMode  = "READ_ONLY"
	managedSharedReadWriteMode = "READ_WRITE"

	launcherComponentVersion = "0.1.0"
	pluginComponentVersion   = "0.1.0"
)

type localAuthorityContext struct {
	attemptID                 string
	authorityScopeDigest      string
	configurationPolicyDigest string
	dependencyCacheAuthorized bool
	cacheBinding              *gatewayCacheBinding
	managedL1Config           managedL1Config
	childEnvironment          map[string]string
}

func localAuthorityContextFromEnvironment(
	ctx context.Context,
	getenv func(string) string,
	now time.Time,
) (*localAuthorityContext, bool, error) {
	authorityPath := getenv(localAuthorityPathEnvironment)
	trustRootPath := getenv(localTrustRootPathEnvironment)
	credentialPath := getenv(localCredentialPathEnvironment)
	sharedCacheURL := getenv(sharedCacheURLEnvironment)
	primary := []string{
		authorityPath,
		trustRootPath,
		credentialPath,
		sharedCacheURL,
	}
	configured := false
	complete := true
	for _, value := range primary {
		configured = configured || value != ""
		complete = complete && value != ""
	}
	if !configured {
		return nil, false, nil
	}
	if !complete {
		return nil, true, errors.New(
			"local authority requires authority, trust root, credential, and Shared cache URL",
		)
	}

	stateRoot := getenv(managedL1StateRootEnvironment)
	tenantID := getenv(managedL1TenantEnvironment)
	repositoryID := getenv(managedL1RepositoryEnvironment)
	trustDomain := getenv(managedL1TrustDomainEnvironment)
	compatibilityClass := getenv(managedL1CompatibilityEnvironment)
	if stateRoot == "" ||
		tenantID == "" ||
		repositoryID == "" ||
		trustDomain == "" ||
		compatibilityClass == "" {
		return nil, true, errors.New(
			"local authority requires managed L1 state root, tenant, repository, trust domain, and compatibility class",
		)
	}
	if !filepath.IsAbs(stateRoot) || filepath.Clean(stateRoot) != stateRoot {
		return nil, true, errors.New(
			"BUILDOPT_L1_STATE_ROOT must be an absolute clean path",
		)
	}
	for _, identity := range []struct {
		name  string
		value string
	}{
		{name: managedL1TenantEnvironment, value: tenantID},
		{name: managedL1RepositoryEnvironment, value: repositoryID},
		{name: managedL1TrustDomainEnvironment, value: trustDomain},
		{
			name:  managedL1CompatibilityEnvironment,
			value: compatibilityClass,
		},
	} {
		if err := validateManagedL1Identity(
			identity.name,
			identity.value,
		); err != nil {
			return nil, true, err
		}
	}

	verified, _, credential, err := localauthority.LoadFiles(
		ctx,
		authorityPath,
		trustRootPath,
		credentialPath,
		now.UTC(),
	)
	if err != nil {
		return nil, true, err
	}
	defer clearBytes(credential)
	if !verified.SupportsComponents(
		launcherComponentVersion,
		pluginComponentVersion,
	) {
		return nil, true, errors.New(
			"local authority does not support this launcher and plugin version",
		)
	}
	document := verified.Document()
	if document.Repository.Tenant != tenantID ||
		document.Repository.Repository != repositoryID ||
		document.Repository.TrustDomain != trustDomain {
		return nil, true, errors.New(
			"local authority repository binding does not match local configuration",
		)
	}
	ci := getenv("CI") == "true" || getenv("CI") == "1"
	if ci && !document.Policy.AffectedBuild.EnabledInCI ||
		!ci && !document.Policy.AffectedBuild.EnabledLocally {
		return nil, true, errors.New(
			"local authority does not enable this build class",
		)
	}
	if document.Attempt.AllowWrite && !ci {
		return nil, true, errors.New(
			"local authority write access requires an authenticated CI build",
		)
	}

	store, err := localauthority.NewFileStateStore(stateRoot)
	if err != nil {
		return nil, true, fmt.Errorf(
			"prepare local authority state: %w",
			err,
		)
	}
	if _, _, _, err := store.Install(verified, now.UTC()); err != nil {
		return nil, true, fmt.Errorf(
			"install local authority state: %w",
			err,
		)
	}
	cacheBinding, err := newGatewayCacheBinding(
		sharedCacheURL,
		credential,
		document.AuthorityDigest,
		document.Attempt.AttemptID,
		document.Attempt.AllowRead,
		document.Attempt.AllowWrite,
		verified.ExpiresAt(),
	)
	if err != nil {
		return nil, true, err
	}

	mode := managedSharedReadOnlyMode
	if document.Attempt.AllowWrite {
		mode = managedSharedReadWriteMode
	}
	generation := uint64(document.Revocation.L1SecurityGeneration)
	l1Config := managedL1Config{
		stateRoot:          stateRoot,
		tenantID:           tenantID,
		repositoryID:       repositoryID,
		trustDomain:        trustDomain,
		compatibilityClass: compatibilityClass,
		securityGeneration: generation,
		l2WriteAuthorized:  document.Attempt.AllowWrite,
		scopeDigest: managedL1ScopeDigest(
			tenantID,
			repositoryID,
			trustDomain,
			compatibilityClass,
		),
	}
	return &localAuthorityContext{
		attemptID:                 document.Attempt.AttemptID,
		authorityScopeDigest:      localauthority.ScopeDigest(document.Repository),
		configurationPolicyDigest: document.Policy.ConfigurationPolicyDigest,
		dependencyCacheAuthorized: verified.AllowsAction("DEPENDENCY_CACHE"),
		cacheBinding:              cacheBinding,
		managedL1Config:           l1Config,
		childEnvironment: map[string]string{
			managedSharedModeEnvironment:      mode,
			managedAuthorityDigestEnvironment: document.AuthorityDigest,
			managedPolicyDigestEnvironment:    document.Policy.PolicyDigest,
			managedConfigurationDigestEnvironment: document.Policy.
				ConfigurationPolicyDigest,
			managedAuthorityContractEnvironment: localauthority.AuthorityContractVersion,
			managedL1GenerationChildEnvironment: strconv.FormatUint(generation, 10),
		},
	}, true, nil
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
