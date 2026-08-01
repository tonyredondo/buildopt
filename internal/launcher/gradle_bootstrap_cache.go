package launcher

import (
	"bufio"
	"bytes"
	"crypto/md5" // #nosec G501 -- Gradle uses MD5 only for its URL-derived directory name.
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/filelock"
	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	gradleBootstrapConfigPathEnvironment = "BUILDOPT_GRADLE_BOOTSTRAP_CONFIG_PATH"

	gradleUserHomeEnvironment            = "GRADLE_USER_HOME"
	gradleReadOnlyDependencyEnvironment  = "GRADLE_RO_DEP_CACHE"
	gradleBootstrapConfigVersion         = "buildopt-gradle-bootstrap-cache/v1"
	gradleDependencyManifestVersion      = "buildopt-gradle-dependency-cache/v1"
	gradleWrapperMarkerVersion           = "buildopt-gradle-wrapper-install/v1"
	gradleDependencyManifestName         = "buildopt-dependency-cache.json"
	gradleBootstrapScopeDomain           = "buildopt-gradle-bootstrap-scope-v1"
	gradleBootstrapMaximumConfigBytes    = 64 << 10
	gradleBootstrapMaximumManifestBytes  = 64 << 10
	gradleBootstrapMaximumPropertiesSize = 64 << 10
	gradleBootstrapMaximumArchiveBytes   = 1 << 30
)

var (
	errGradleBootstrapBusy = errors.New(
		"managed Gradle writable cache scope is already active",
	)
	gradleDistributionNamePattern = regexp.MustCompile(
		`^gradle-(8\.14\.3|9\.6\.1)-(bin|all)\.zip$`,
	)
)

type gradleBootstrapConfigDocument struct {
	SchemaVersion           string `json:"schemaVersion"`
	StateRoot               string `json:"stateRoot"`
	RunnerSlot              string `json:"runnerSlot"`
	CompatibilityClass      string `json:"compatibilityClass"`
	DependencyCacheRoot     string `json:"dependencyCacheRoot"`
	WrapperPropertiesPath   string `json:"wrapperPropertiesPath"`
	DistributionArchivePath string `json:"distributionArchivePath"`
	WrapperJarDigest        string `json:"wrapperJarDigest"`
}

type gradleDependencyCacheManifest struct {
	SchemaVersion             string `json:"schemaVersion"`
	GradleVersion             string `json:"gradleVersion"`
	CompatibilityClass        string `json:"compatibilityClass"`
	ConfigurationPolicyDigest string `json:"configurationPolicyDigest"`
	SnapshotID                string `json:"snapshotId"`
}

type gradleWrapperMarker struct {
	SchemaVersion             string `json:"schemaVersion"`
	DistributionURL           string `json:"distributionUrl"`
	DistributionDigest        string `json:"distributionDigest"`
	WrapperJarDigest          string `json:"wrapperJarDigest"`
	DependencySnapshotID      string `json:"dependencySnapshotId"`
	ConfigurationPolicyDigest string `json:"configurationPolicyDigest"`
}

type gradleWrapperProperties struct {
	version            string
	distributionURL    string
	distributionDigest string
	archiveName        string
	distributionName   string
	urlHash            string
}

type gradleDistributionLocation struct {
	root       string
	archive    string
	okMarker   string
	gradleHome string
}

type gradleBootstrapCache struct {
	userHome            string
	dependencyCacheRoot string
	scopeRoot           string
	lease               *os.File
	properties          gradleWrapperProperties
	marker              gradleWrapperMarker
	markerReady         bool
}

func startInvocationGradleBootstrapCache(
	childArgs []string,
	authority *localAuthorityContext,
	getenv func(string) string,
) (*gradleBootstrapCache, bool, error) {
	configPath := getenv(gradleBootstrapConfigPathEnvironment)
	if configPath == "" {
		return nil, false, nil
	}
	if authority == nil || !authority.dependencyCacheAuthorized {
		return nil, true, errors.New(
			"managed Gradle bootstrap cache requires signed DEPENDENCY_CACHE authority",
		)
	}
	config, err := loadGradleBootstrapConfig(configPath)
	if err != nil {
		return nil, true, err
	}
	cache, err := startGradleBootstrapCache(
		config,
		childArgs,
		authority,
	)
	return cache, true, err
}

func loadGradleBootstrapConfig(
	path string,
) (gradleBootstrapConfigDocument, error) {
	raw, err := localauthority.ReadPrivateFile(
		path,
		gradleBootstrapMaximumConfigBytes,
	)
	if err != nil {
		return gradleBootstrapConfigDocument{}, fmt.Errorf(
			"read managed Gradle bootstrap config: %w",
			err,
		)
	}
	var config gradleBootstrapConfigDocument
	if err := decodeCanonicalJSON(raw, &config); err != nil {
		return gradleBootstrapConfigDocument{}, fmt.Errorf(
			"decode managed Gradle bootstrap config: %w",
			err,
		)
	}
	if config.SchemaVersion != gradleBootstrapConfigVersion {
		return gradleBootstrapConfigDocument{}, errors.New(
			"unsupported managed Gradle bootstrap config",
		)
	}
	if !filepath.IsAbs(config.StateRoot) ||
		filepath.Clean(config.StateRoot) != config.StateRoot ||
		!filepath.IsAbs(config.DependencyCacheRoot) ||
		filepath.Clean(config.DependencyCacheRoot) !=
			config.DependencyCacheRoot ||
		!filepath.IsAbs(config.WrapperPropertiesPath) ||
		filepath.Clean(config.WrapperPropertiesPath) !=
			config.WrapperPropertiesPath ||
		!filepath.IsAbs(config.DistributionArchivePath) ||
		filepath.Clean(config.DistributionArchivePath) !=
			config.DistributionArchivePath {
		return gradleBootstrapConfigDocument{}, errors.New(
			"managed Gradle bootstrap paths must be absolute and clean",
		)
	}
	if err := validateManagedRunnerSlot(config.RunnerSlot); err != nil {
		return gradleBootstrapConfigDocument{}, fmt.Errorf(
			"invalid managed Gradle runner slot: %w",
			err,
		)
	}
	if err := validateManagedL1Identity(
		"managed Gradle compatibility class",
		config.CompatibilityClass,
	); err != nil {
		return gradleBootstrapConfigDocument{}, err
	}
	if !validGatewayAuthorityDigest(config.WrapperJarDigest) {
		return gradleBootstrapConfigDocument{}, errors.New(
			"managed Gradle wrapper JAR digest is invalid",
		)
	}
	return config, nil
}

func startGradleBootstrapCache(
	config gradleBootstrapConfigDocument,
	childArgs []string,
	authority *localAuthorityContext,
) (*gradleBootstrapCache, error) {
	properties, projectRoot, err := loadGradleWrapperProperties(
		config.WrapperPropertiesPath,
	)
	if err != nil {
		return nil, err
	}
	if err := validateGradleWrapperCommand(childArgs, projectRoot); err != nil {
		return nil, err
	}
	wrapperJarPath := filepath.Join(
		filepath.Dir(config.WrapperPropertiesPath),
		"gradle-wrapper.jar",
	)
	if err := verifyRegularFileDigest(
		wrapperJarPath,
		config.WrapperJarDigest,
		gradleBootstrapMaximumArchiveBytes,
		false,
	); err != nil {
		return nil, fmt.Errorf("verify Gradle Wrapper JAR: %w", err)
	}
	manifest, err := loadGradleDependencyManifest(
		config.DependencyCacheRoot,
	)
	if err != nil {
		return nil, err
	}
	if manifest.GradleVersion != properties.version ||
		manifest.CompatibilityClass != config.CompatibilityClass ||
		manifest.ConfigurationPolicyDigest !=
			authority.configurationPolicyDigest {
		return nil, errors.New(
			"read-only dependency cache is incompatible with current signed policy",
		)
	}

	scopeDigest := gradleBootstrapScopeDigest(
		authority.authorityScopeDigest,
		authority.configurationPolicyDigest,
		config.RunnerSlot,
		config.CompatibilityClass,
		config.DependencyCacheRoot,
		manifest.SnapshotID,
		properties.distributionURL,
		properties.distributionDigest,
		config.WrapperJarDigest,
	)
	bootstrapRoot := filepath.Join(config.StateRoot, "gradle-bootstrap")
	scopeRoot := filepath.Join(bootstrapRoot, "scopes", scopeDigest)
	userHome := filepath.Join(scopeRoot, "home")
	lockRoot := filepath.Join(bootstrapRoot, "locks")
	for _, directory := range []string{
		config.StateRoot,
		bootstrapRoot,
		filepath.Join(bootstrapRoot, "scopes"),
		scopeRoot,
		userHome,
		lockRoot,
	} {
		if err := ensurePrivateDirectory(directory, true); err != nil {
			return nil, fmt.Errorf(
				"prepare managed Gradle writable cache: %w",
				err,
			)
		}
	}
	lease, err := openPrivateLock(
		filepath.Join(lockRoot, scopeDigest+".lock"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open managed Gradle writable cache lease: %w",
			err,
		)
	}
	if err := filelock.Try(lease, filelock.Exclusive); err != nil {
		_ = lease.Close()
		if errors.Is(err, filelock.ErrBusy) {
			return nil, errGradleBootstrapBusy
		}
		return nil, fmt.Errorf(
			"acquire managed Gradle writable cache lease: %w",
			err,
		)
	}

	cache := &gradleBootstrapCache{
		userHome:            userHome,
		dependencyCacheRoot: config.DependencyCacheRoot,
		scopeRoot:           scopeRoot,
		lease:               lease,
		properties:          properties,
		marker: gradleWrapperMarker{
			SchemaVersion:             gradleWrapperMarkerVersion,
			DistributionURL:           properties.distributionURL,
			DistributionDigest:        properties.distributionDigest,
			WrapperJarDigest:          config.WrapperJarDigest,
			DependencySnapshotID:      manifest.SnapshotID,
			ConfigurationPolicyDigest: authority.configurationPolicyDigest,
		},
	}
	if err := cache.prepareWrapper(config.DistributionArchivePath); err != nil {
		_ = cache.close()
		return nil, err
	}
	return cache, nil
}

func (cache *gradleBootstrapCache) childEnvironment() map[string]string {
	return map[string]string{
		gradleUserHomeEnvironment:           cache.userHome,
		gradleReadOnlyDependencyEnvironment: cache.dependencyCacheRoot,
	}
}

func (cache *gradleBootstrapCache) prepareWrapper(
	distributionArchivePath string,
) error {
	location, err := cache.distributionLocation()
	if err != nil {
		return err
	}
	markerPath := filepath.Join(cache.scopeRoot, "wrapper-install.json")
	persisted, err := readGradleWrapperMarker(markerPath)
	if err == nil {
		if persisted != cache.marker {
			return errors.New(
				"managed Gradle Wrapper marker conflicts with current policy",
			)
		}
		if err := validateInstalledGradleDistribution(location); err != nil {
			return fmt.Errorf(
				"managed Gradle Wrapper installation is corrupt: %w",
				err,
			)
		}
		cache.markerReady = true
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read managed Gradle Wrapper marker: %w", err)
	}

	if err := resetPrivateDistributionRoot(location.root); err != nil {
		return err
	}
	if err := ensurePrivateDirectory(location.root, true); err != nil {
		return err
	}
	if err := copyVerifiedDistributionArchive(
		distributionArchivePath,
		location.archive,
		cache.properties.distributionDigest,
	); err != nil {
		return fmt.Errorf(
			"prepare verified Gradle Wrapper distribution: %w",
			err,
		)
	}
	return nil
}

func (cache *gradleBootstrapCache) finalize() error {
	if cache == nil || cache.markerReady {
		return nil
	}
	location, err := cache.distributionLocation()
	if err != nil {
		return err
	}
	if err := validateInstalledGradleDistribution(location); err != nil {
		return err
	}
	if err := writeCanonicalPrivateJSON(
		filepath.Join(cache.scopeRoot, "wrapper-install.json"),
		cache.marker,
	); err != nil {
		return err
	}
	cache.markerReady = true
	return nil
}

func (cache *gradleBootstrapCache) close() error {
	if cache == nil || cache.lease == nil {
		return nil
	}
	lease := cache.lease
	cache.lease = nil
	return releaseManagedLock(lease)
}

func loadGradleDependencyManifest(
	root string,
) (gradleDependencyCacheManifest, error) {
	if err := validateReadOnlyDirectory(root); err != nil {
		return gradleDependencyCacheManifest{}, fmt.Errorf(
			"invalid read-only dependency cache root: %w",
			err,
		)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return gradleDependencyCacheManifest{}, err
	}
	if len(entries) != 2 ||
		entries[0].Name() != gradleDependencyManifestName ||
		entries[1].Name() != "modules-2" {
		return gradleDependencyCacheManifest{}, errors.New(
			"read-only dependency cache root must contain only its manifest and modules-2",
		)
	}
	modulesRoot := filepath.Join(root, "modules-2")
	if err := validateReadOnlyDirectory(modulesRoot); err != nil {
		return gradleDependencyCacheManifest{}, fmt.Errorf(
			"invalid read-only modules-2 directory: %w",
			err,
		)
	}
	if err := validateReadOnlyDependencyTree(modulesRoot); err != nil {
		return gradleDependencyCacheManifest{}, fmt.Errorf(
			"invalid read-only dependency cache tree: %w",
			err,
		)
	}
	raw, err := readBoundedRegularFile(
		filepath.Join(root, gradleDependencyManifestName),
		gradleBootstrapMaximumManifestBytes,
		true,
	)
	if err != nil {
		return gradleDependencyCacheManifest{}, err
	}
	var manifest gradleDependencyCacheManifest
	if err := decodeCanonicalJSON(raw, &manifest); err != nil {
		return gradleDependencyCacheManifest{}, err
	}
	if manifest.SchemaVersion != gradleDependencyManifestVersion ||
		!supportedGradleBootstrapVersion(manifest.GradleVersion) ||
		!validGatewayAuthorityDigest(
			manifest.ConfigurationPolicyDigest,
		) ||
		validateManagedL1Identity(
			"dependency cache compatibility class",
			manifest.CompatibilityClass,
		) != nil ||
		validateManagedL1Identity(
			"dependency cache snapshot ID",
			manifest.SnapshotID,
		) != nil {
		return gradleDependencyCacheManifest{}, errors.New(
			"invalid read-only dependency cache manifest",
		)
	}
	return manifest, nil
}

func loadGradleWrapperProperties(
	path string,
) (gradleWrapperProperties, string, error) {
	raw, err := readBoundedRegularFile(
		path,
		gradleBootstrapMaximumPropertiesSize,
		false,
	)
	if err != nil {
		return gradleWrapperProperties{}, "", fmt.Errorf(
			"read Gradle Wrapper properties: %w",
			err,
		)
	}
	values, err := parseGradleWrapperPropertyFile(raw)
	if err != nil {
		return gradleWrapperProperties{}, "", err
	}
	for key, expected := range map[string]string{
		"distributionBase": "GRADLE_USER_HOME",
		"distributionPath": "wrapper/dists",
		"zipStoreBase":     "GRADLE_USER_HOME",
		"zipStorePath":     "wrapper/dists",
	} {
		if values[key] != expected {
			return gradleWrapperProperties{}, "", fmt.Errorf(
				"unsupported Gradle Wrapper %s",
				key,
			)
		}
	}
	if values["validateDistributionUrl"] != "true" {
		return gradleWrapperProperties{}, "", errors.New(
			"Gradle Wrapper URL validation must be enabled",
		)
	}
	digest := values["distributionSha256Sum"]
	if len(digest) != 64 ||
		!validGatewayAuthorityDigest("sha256:"+digest) {
		return gradleWrapperProperties{}, "", errors.New(
			"Gradle Wrapper distribution SHA-256 is invalid",
		)
	}
	distributionURL, err := url.Parse(values["distributionUrl"])
	if err != nil ||
		distributionURL.Scheme != "https" ||
		distributionURL.Host == "" ||
		distributionURL.User != nil ||
		distributionURL.RawQuery != "" ||
		distributionURL.Fragment != "" {
		return gradleWrapperProperties{}, "", errors.New(
			"Gradle Wrapper distribution URL must be canonical HTTPS",
		)
	}
	canonicalURL := distributionURL.String()
	if canonicalURL != values["distributionUrl"] {
		return gradleWrapperProperties{}, "", errors.New(
			"Gradle Wrapper distribution URL is not canonical",
		)
	}
	archiveName := filepath.Base(distributionURL.Path)
	matches := gradleDistributionNamePattern.FindStringSubmatch(archiveName)
	if len(matches) != 3 ||
		!supportedGradleBootstrapVersion(matches[1]) {
		return gradleWrapperProperties{}, "", errors.New(
			"Gradle Wrapper distribution is outside the tested matrix",
		)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	if filepath.Join(
		projectRoot,
		"gradle",
		"wrapper",
		"gradle-wrapper.properties",
	) != path {
		return gradleWrapperProperties{}, "", errors.New(
			"Gradle Wrapper properties path has an unsupported layout",
		)
	}
	hash := md5.Sum([]byte(canonicalURL)) // #nosec G401 -- compatibility path only.
	return gradleWrapperProperties{
		version:            matches[1],
		distributionURL:    canonicalURL,
		distributionDigest: "sha256:" + digest,
		archiveName:        archiveName,
		distributionName:   strings.TrimSuffix(archiveName, ".zip"),
		urlHash:            new(big.Int).SetBytes(hash[:]).Text(36),
	}, projectRoot, nil
}

func parseGradleWrapperPropertyFile(raw []byte) (map[string]string, error) {
	allowed := map[string]struct{}{
		"distributionBase":        {},
		"distributionPath":        {},
		"distributionSha256Sum":   {},
		"distributionUrl":         {},
		"networkTimeout":          {},
		"retries":                 {},
		"retryBackOffMs":          {},
		"validateDistributionUrl": {},
		"zipStoreBase":            {},
		"zipStorePath":            {},
	}
	required := []string{
		"distributionBase",
		"distributionPath",
		"distributionSha256Sum",
		"distributionUrl",
		"validateDistributionUrl",
		"zipStoreBase",
		"zipStorePath",
	}
	values := make(map[string]string, len(allowed))
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), gradleBootstrapMaximumPropertiesSize)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "!") {
			continue
		}
		separator := strings.IndexByte(trimmed, '=')
		if separator < 1 {
			return nil, errors.New(
				"unsupported Gradle Wrapper properties syntax",
			)
		}
		key := strings.TrimSpace(trimmed[:separator])
		value, err := unescapeGradleWrapperProperty(
			strings.TrimSpace(trimmed[separator+1:]),
		)
		if err != nil {
			return nil, err
		}
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf(
				"unsupported Gradle Wrapper property %q",
				key,
			)
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf(
				"duplicate Gradle Wrapper property %q",
				key,
			)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, key := range required {
		if values[key] == "" {
			return nil, fmt.Errorf(
				"missing Gradle Wrapper property %q",
				key,
			)
		}
	}
	for _, key := range []string{
		"networkTimeout",
		"retries",
		"retryBackOffMs",
	} {
		if value := values[key]; value != "" {
			parsed, err := strconv.ParseUint(value, 10, 31)
			if err != nil ||
				strconv.FormatUint(parsed, 10) != value {
				return nil, fmt.Errorf(
					"Gradle Wrapper property %q is not canonical",
					key,
				)
			}
		}
	}
	return values, nil
}

func unescapeGradleWrapperProperty(value string) (string, error) {
	var result strings.Builder
	result.Grow(len(value))
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' {
			result.WriteByte(value[index])
			continue
		}
		index++
		if index >= len(value) {
			return "", errors.New(
				"Gradle Wrapper property continuation is unsupported",
			)
		}
		switch value[index] {
		case '\\', ':', '=', ' ':
			result.WriteByte(value[index])
		default:
			return "", errors.New(
				"unsupported Gradle Wrapper property escape",
			)
		}
	}
	return result.String(), nil
}

func validateGradleWrapperCommand(
	childArgs []string,
	projectRoot string,
) error {
	if len(childArgs) == 0 {
		return errors.New("managed Gradle bootstrap command is empty")
	}
	command := childArgs[0]
	if !filepath.IsAbs(command) {
		absolute, err := filepath.Abs(command)
		if err != nil {
			return err
		}
		command = absolute
	}
	command = filepath.Clean(command)
	expected := filepath.Join(projectRoot, "gradlew")
	if command != expected {
		return errors.New(
			"managed Gradle bootstrap cache requires the repository Wrapper command",
		)
	}
	info, err := os.Lstat(command)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm()&0o100 == 0 {
		return errors.New("Gradle Wrapper command is not a regular executable")
	}
	return nil
}

func (cache *gradleBootstrapCache) distributionLocation() (
	gradleDistributionLocation,
	error,
) {
	root := filepath.Join(
		cache.userHome,
		"wrapper",
		"dists",
		cache.properties.distributionName,
		cache.properties.urlHash,
	)
	archive := filepath.Join(root, cache.properties.archiveName)
	gradleHome := filepath.Join(
		root,
		"gradle-"+cache.properties.version,
	)
	if filepath.Clean(root) != root ||
		!strings.HasPrefix(root, cache.userHome+string(filepath.Separator)) {
		return gradleDistributionLocation{}, errors.New(
			"invalid managed Gradle distribution location",
		)
	}
	return gradleDistributionLocation{
		root:       root,
		archive:    archive,
		okMarker:   archive + ".ok",
		gradleHome: gradleHome,
	}, nil
}

func validateInstalledGradleDistribution(
	location gradleDistributionLocation,
) error {
	marker, markerInfo, err := openBoundedRegularFile(
		location.okMarker,
		gradleBootstrapMaximumManifestBytes,
		false,
	)
	if err != nil {
		return err
	}
	_ = marker.Close()
	if markerInfo.Size() != 0 {
		return errors.New("Gradle distribution marker is not empty")
	}
	if err := ensurePrivateDirectory(location.root, false); err != nil {
		return err
	}
	info, err := os.Lstat(location.gradleHome)
	if err != nil ||
		!info.IsDir() ||
		info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Gradle distribution root is unavailable")
	}
	gradleExecutable := gradleDistributionExecutable(location.gradleHome)
	executableInfo, err := os.Lstat(gradleExecutable)
	if err != nil ||
		!validGradleDistributionExecutable(executableInfo) {
		return errors.New("Gradle distribution executable is unavailable")
	}
	return nil
}

func resetPrivateDistributionRoot(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !privateManagedDirectoryInfo(info) {
		return errors.New(
			"partial Gradle distribution root is not private",
		)
	}
	return os.RemoveAll(path)
}

func copyVerifiedDistributionArchive(
	sourcePath string,
	destinationPath string,
	expectedDigest string,
) error {
	source, sourceInfo, err := openBoundedRegularFile(
		sourcePath,
		gradleBootstrapMaximumArchiveBytes,
		true,
	)
	if err != nil {
		return err
	}
	defer source.Close()
	if sourceInfo.Size() == 0 {
		return errors.New("Gradle distribution archive is empty")
	}
	temporary, err := os.CreateTemp(
		filepath.Dir(destinationPath),
		".distribution-*",
	)
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	digest := sha256.New()
	written, err := io.Copy(
		io.MultiWriter(temporary, digest),
		io.LimitReader(source, gradleBootstrapMaximumArchiveBytes+1),
	)
	if err != nil ||
		written != sourceInfo.Size() ||
		written > gradleBootstrapMaximumArchiveBytes {
		_ = temporary.Close()
		return errors.New("copy bounded Gradle distribution archive")
	}
	actualDigest := "sha256:" + hex.EncodeToString(digest.Sum(nil))
	if actualDigest != expectedDigest {
		_ = temporary.Close()
		return errors.New("Gradle distribution archive SHA-256 mismatch")
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, destinationPath); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(destinationPath))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func verifyRegularFileDigest(
	path string,
	expectedDigest string,
	maximumBytes int64,
	requireReadOnly bool,
) error {
	file, info, err := openBoundedRegularFile(
		path,
		maximumBytes,
		requireReadOnly,
	)
	if err != nil {
		return err
	}
	defer file.Close()
	digest := sha256.New()
	read, err := io.Copy(
		digest,
		io.LimitReader(file, maximumBytes+1),
	)
	if err != nil || read != info.Size() || read > maximumBytes {
		return errors.New("hash bounded regular file")
	}
	if "sha256:"+hex.EncodeToString(digest.Sum(nil)) != expectedDigest {
		return errors.New("regular file SHA-256 mismatch")
	}
	return nil
}

func readBoundedRegularFile(
	path string,
	maximumBytes int64,
	requireReadOnly bool,
) ([]byte, error) {
	file, info, err := openBoundedRegularFile(
		path,
		maximumBytes,
		requireReadOnly,
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil ||
		int64(len(raw)) != info.Size() ||
		int64(len(raw)) > maximumBytes {
		return nil, errors.New("read bounded regular file")
	}
	return raw, nil
}

func openBoundedRegularFile(
	path string,
	maximumBytes int64,
	requireReadOnly bool,
) (*os.File, os.FileInfo, error) {
	if !filepath.IsAbs(path) ||
		filepath.Clean(path) != path ||
		maximumBytes < 1 {
		return nil, nil, errors.New("invalid bounded regular file request")
	}
	file, err := openTrustedManagedFile(path)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if !trustedManagedFileInfo(info) ||
		info.Size() < 0 ||
		info.Size() > maximumBytes ||
		requireReadOnly && !managedReadOnlyInfo(info) {
		_ = file.Close()
		return nil, nil, errors.New(
			"file is not a bounded trusted regular file",
		)
	}
	return file, info, nil
}

func validateReadOnlyDirectory(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("read-only directory path is not absolute and clean")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !trustedManagedDirectoryInfo(info) || !managedReadOnlyInfo(info) {
		return errors.New("directory is not a trusted read-only directory")
	}
	return nil
}

func validateReadOnlyDependencyTree(root string) error {
	return filepath.WalkDir(root, func(
		path string,
		entry os.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !trustedManagedEntryInfo(info) || !managedReadOnlyInfo(info) {
			return fmt.Errorf(
				"dependency cache entry %q is not trusted and read-only",
				path,
			)
		}
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"dependency cache entry %q is not a regular file",
				path,
			)
		}
		if strings.HasSuffix(entry.Name(), ".lock") ||
			strings.HasSuffix(entry.Name(), ".lock.lock") {
			return fmt.Errorf(
				"read-only dependency cache contains shared lock %q",
				path,
			)
		}
		return nil
	})
}

func readGradleWrapperMarker(path string) (gradleWrapperMarker, error) {
	raw, err := localauthority.ReadPrivateFile(
		path,
		gradleBootstrapMaximumManifestBytes,
	)
	if err != nil {
		return gradleWrapperMarker{}, err
	}
	var marker gradleWrapperMarker
	if err := decodeCanonicalJSON(raw, &marker); err != nil {
		return gradleWrapperMarker{}, err
	}
	if marker.SchemaVersion != gradleWrapperMarkerVersion ||
		!validGatewayAuthorityDigest(marker.DistributionDigest) ||
		!validGatewayAuthorityDigest(marker.WrapperJarDigest) ||
		!validGatewayAuthorityDigest(
			marker.ConfigurationPolicyDigest,
		) ||
		validateManagedL1Identity(
			"dependency snapshot ID",
			marker.DependencySnapshotID,
		) != nil {
		return gradleWrapperMarker{}, errors.New(
			"invalid managed Gradle Wrapper marker",
		)
	}
	return marker, nil
}

func decodeCanonicalJSON(raw []byte, target any) error {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return errors.New("document is not canonical JCS")
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func gradleBootstrapScopeDigest(values ...string) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(gradleBootstrapScopeDomain))
	for _, value := range values {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(value)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(value))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func supportedGradleBootstrapVersion(version string) bool {
	return version == "8.14.3" || version == "9.6.1"
}
