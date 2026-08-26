package launcher

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const centralCacheAuthorityContract = "buildopt-central-cache-connection/v1"

type centralGradleCacheContext struct {
	binding          *gatewayCacheBinding
	cacheClient      *http.Client
	childEnvironment map[string]string
}

func (integration *centralOptimizeIntegration) hasReadOnlyCentralCache() bool {
	return integration != nil &&
		integration.connection.Cache != nil &&
		integration.connection.Cache.Mode == managedSharedReadOnlyMode
}

func prepareConnectedCentralGradleCache(
	repositoryRoot string,
) (*centralOptimizeIntegration, error) {
	connectionDirectory, _, err := resolveCentralConnectionDirectory(
		repositoryRoot,
		centralConnectionDir,
		false,
	)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(filepath.Join(connectionDirectory, centralConnectionFile)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect central Gradle cache connection: %w", err)
	}
	connection, err := loadCentralConnection(repositoryRoot, connectionDirectory)
	if err != nil {
		return nil, err
	}
	integration := &centralOptimizeIntegration{
		invocation: optimizeInvocation{
			repositoryRoot:      repositoryRoot,
			connectionDirectory: connectionDirectory,
		},
		connection: connection,
		result:     disconnectedOptimizeCentralResult(),
	}
	if connection.Cache == nil {
		return nil, nil
	}
	if !integration.hasReadOnlyCentralCache() {
		return nil, errors.New("connected central Gradle cache is not read-only")
	}
	return integration, nil
}

func enableConnectedCentralCacheGradle(
	invocation *gradleInvocation,
	repositoryRoot string,
) error {
	if invocation == nil || repositoryRoot == "" {
		return errors.New("central Gradle cache invocation is unavailable")
	}
	initScript, pluginJar, err := resolveGradleAssets()
	if err != nil {
		return err
	}
	if len(invocation.childArgs) == 0 {
		return errors.New("central Gradle cache command is unavailable")
	}
	buildCacheConfigured, buildCacheEnabled := gradleBuildCacheMode(invocation.childArgs[1:])
	if buildCacheConfigured && !buildCacheEnabled {
		return errors.New("connected central Gradle cache is incompatible with --no-build-cache")
	}
	gradleArguments := append([]string(nil), invocation.childArgs[1:]...)
	invocation.childArgs = append(
		[]string{invocation.childArgs[0], "--init-script", initScript},
		gradleArguments...,
	)
	if !buildCacheConfigured {
		invocation.childArgs = append(
			[]string{invocation.childArgs[0], "--init-script", initScript, "--build-cache"},
			gradleArguments...,
		)
	}
	if invocation.environment == nil {
		invocation.environment = make(map[string]string)
	}
	invocation.environment[gradlePluginJarEnvironment] = pluginJar
	invocation.environment[gradleProjectPluginModeEnvironment] = gradleProjectPluginModeCacheOnly
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("resolve central Gradle local cache directory: %w", err)
	}
	managedL1, err := defaultGradleManagedL1Config(repositoryRoot, cacheRoot)
	if err != nil {
		return err
	}
	invocation.managedL1 = &managedL1
	invocation.nativeOnly = false
	invocation.localOnly = false
	return nil
}

// centralGradleCacheContext creates the read-only binding used by one sticky
// wrapper invocation. The credential remains in the in-memory connection and
// is never written to the repository or exposed to Gradle.
func (connection *stickyWrapperConnection) centralGradleCacheContext(
	attemptID string,
	now time.Time,
) (*centralGradleCacheContext, error) {
	if connection == nil {
		return nil, nil
	}
	if connection.expiresAt.IsZero() || !connection.expiresAt.After(now.UTC()) {
		return nil, errors.New("sticky wrapper Gradle cache credential is expired")
	}
	if connection.http == nil || len(connection.token) != sharedcache.CentralTokenBytes {
		return nil, errors.New("sticky wrapper Gradle cache connection is incomplete")
	}
	authorityDigest := "sha256:" + connection.projectScopeSHA256
	binding, err := newGatewayCacheBinding(
		connection.serverURL,
		connection.token,
		authorityDigest,
		attemptID,
		connection.namespace,
		true,
		false,
		connection.expiresAt,
	)
	if err != nil {
		return nil, err
	}
	return &centralGradleCacheContext{
		binding:     binding,
		cacheClient: connection.http,
		childEnvironment: map[string]string{
			managedSharedModeEnvironment:      managedSharedReadOnlyMode,
			managedSharedPolicyEnvironment:    managedNativeSharedPolicy,
			managedAuthorityDigestEnvironment: authorityDigest,
			managedPolicyDigestEnvironment: centralCacheConnectionDigest(
				"policy",
				connection.projectScopeSHA256,
				connection.namespace,
				connection.namespaceGeneration,
			),
			managedConfigurationDigestEnvironment: centralCacheConnectionDigest(
				"configuration",
				connection.projectScopeSHA256,
				connection.namespace,
				connection.namespaceGeneration,
			),
			managedAuthorityContractEnvironment: centralCacheAuthorityContract,
		},
	}, nil
}

func (integration *centralOptimizeIntegration) centralGradleCacheContext(
	attemptID string,
	now time.Time,
) (*centralGradleCacheContext, error) {
	if !integration.hasReadOnlyCentralCache() {
		return nil, nil
	}
	cache := integration.connection.Cache
	expiresAt, err := time.Parse(time.RFC3339Nano, cache.ExpiresAt)
	if err != nil || !expiresAt.After(now.UTC()) {
		return nil, errors.New("central Gradle cache credential is expired")
	}
	tokenPath := filepath.Join(
		integration.invocation.connectionDirectory,
		integration.connection.TokenFile,
	)
	token, err := readPrivateCentralCredential(tokenPath, centralMaximumConfig)
	if err != nil {
		return nil, fmt.Errorf("read central Gradle cache credential: %w", err)
	}
	defer clear(token)
	credential, err := base64.RawURLEncoding.DecodeString(string(token))
	if err != nil || base64.RawURLEncoding.EncodeToString(credential) != string(token) {
		clear(credential)
		return nil, errors.New("central Gradle cache credential is invalid")
	}
	defer clear(credential)
	var ca []byte
	if integration.connection.CAFile != "" {
		ca, err = readPrivateCentralCredential(
			filepath.Join(
				integration.invocation.connectionDirectory,
				integration.connection.CAFile,
			),
			centralMaximumCA,
		)
		if err != nil {
			return nil, fmt.Errorf("read central Gradle cache CA: %w", err)
		}
	}
	client, err := newCentralStateClient(
		integration.connection.ServerURL,
		string(token),
		ca,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare central Gradle cache TLS client: %w", err)
	}
	authorityDigest := "sha256:" + integration.connection.RepositoryScopeSHA256
	binding, err := newGatewayCacheBinding(
		integration.connection.ServerURL,
		credential,
		authorityDigest,
		attemptID,
		cache.Namespace,
		true,
		false,
		expiresAt,
	)
	if err != nil {
		return nil, err
	}
	policyDigest := centralCacheConnectionDigest(
		"policy",
		integration.connection.RepositoryScopeSHA256,
		cache.Namespace,
		cache.NamespaceGeneration,
	)
	configurationDigest := centralCacheConnectionDigest(
		"configuration",
		integration.connection.RepositoryScopeSHA256,
		cache.Namespace,
		cache.NamespaceGeneration,
	)
	return &centralGradleCacheContext{
		binding: binding, cacheClient: client.http,
		childEnvironment: map[string]string{
			managedSharedModeEnvironment:          managedSharedReadOnlyMode,
			managedSharedPolicyEnvironment:        managedNativeSharedPolicy,
			managedAuthorityDigestEnvironment:     authorityDigest,
			managedPolicyDigestEnvironment:        policyDigest,
			managedConfigurationDigestEnvironment: configurationDigest,
			managedAuthorityContractEnvironment:   centralCacheAuthorityContract,
		},
	}, nil
}

func centralCacheConnectionDigest(
	kind string,
	repositoryScope string,
	namespace string,
	generation int64,
) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf(
		"buildopt-central-cache-%s-v1\x00%s\x00%s\x00%d",
		kind,
		repositoryScope,
		namespace,
		generation,
	)))
	return "sha256:" + hex.EncodeToString(digest[:])
}
