// Package selfhosted owns the declarative configuration boundary for the
// isolated single-node private-beta deployment.
package selfhosted

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	SchemaVersion             = "buildopt.self-hosted/config/v1"
	Profile                   = "PRIVATE_BETA_ISOLATED_SINGLE_NODE"
	FilesystemPolicy          = "ALLOWLIST_PROVEN_LOCAL"
	MinimumDeploymentBytes    = int64(20 << 30)
	MaximumDeploymentBytes    = int64(500 << 30)
	UsableVolumePercent       = 50
	maximumConfigurationBytes = 64 << 10
)

type Config struct {
	SchemaVersion string       `json:"schemaVersion"`
	Profile       string       `json:"profile"`
	Server        Server       `json:"server"`
	Storage       Storage      `json:"storage"`
	Export        Export       `json:"export"`
	Cache         Cache        `json:"cache"`
	GitHubQueue   *GitHubQueue `json:"githubQueue,omitempty"`
}

type Server struct {
	Listen string `json:"listen"`
}

type Storage struct {
	StateDirectory         string `json:"stateDirectory"`
	FilesystemPolicy       string `json:"filesystemPolicy"`
	MinimumDeploymentBytes int64  `json:"minimumDeploymentBytes"`
	MaximumDeploymentBytes int64  `json:"maximumDeploymentBytes"`
	UsableVolumePercent    int    `json:"usableVolumePercent"`
}

type Export struct {
	Directory string `json:"directory"`
	Profile   string `json:"profile"`
}

type Cache struct {
	AuthorityPath           string `json:"authorityPath"`
	TrustRootPath           string `json:"trustRootPath"`
	CredentialPath          string `json:"credentialPath"`
	BetaTokenAuthentication bool   `json:"betaTokenAuthentication"`
}

type GitHubQueue struct {
	WebhookSecretPath string `json:"webhookSecretPath"`
}

// Load reads one private, strict, path-only deployment configuration.
func Load(path string) (Config, error) {
	if !absoluteCleanNonRoot(path) {
		return Config{}, errors.New("self-hosted configuration path must be absolute, clean, and non-root")
	}
	raw, err := localauthority.ReadPrivateFile(path, maximumConfigurationBytes)
	if err != nil {
		return Config{}, fmt.Errorf("read self-hosted configuration: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode self-hosted configuration: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("self-hosted configuration has trailing content")
	}
	if err := validate(config, path); err != nil {
		return Config{}, err
	}
	return config, nil
}

func validate(config Config, configPath string) error {
	if config.SchemaVersion != SchemaVersion || config.Profile != Profile {
		return errors.New("unsupported self-hosted configuration identity")
	}
	host, portText, err := net.SplitHostPort(config.Server.Listen)
	port, portErr := strconv.Atoi(portText)
	if err != nil || portErr != nil || host != "127.0.0.1" || port < 0 || port > 65535 {
		return errors.New("self-hosted listener must be a canonical IPv4 loopback address and valid port")
	}
	if config.Storage.FilesystemPolicy != FilesystemPolicy ||
		config.Storage.MinimumDeploymentBytes != MinimumDeploymentBytes ||
		config.Storage.MaximumDeploymentBytes != MaximumDeploymentBytes ||
		config.Storage.UsableVolumePercent != UsableVolumePercent {
		return errors.New("self-hosted storage policy must match the private-beta bounds")
	}
	if config.Export.Profile != "summary" {
		return errors.New("self-hosted POC export profile must be summary")
	}
	paths := []string{
		config.Storage.StateDirectory,
		config.Export.Directory,
		config.Cache.AuthorityPath,
		config.Cache.TrustRootPath,
		config.Cache.CredentialPath,
	}
	if config.GitHubQueue != nil {
		paths = append(paths, config.GitHubQueue.WebhookSecretPath)
	}
	for _, candidate := range paths {
		if !absoluteCleanNonRoot(candidate) {
			return errors.New("self-hosted paths must be absolute, clean, and non-root")
		}
	}
	if !config.Cache.BetaTokenAuthentication {
		return errors.New("self-hosted cache requires scoped beta-token authentication")
	}
	for i, left := range paths {
		for _, right := range paths[i+1:] {
			if overlaps(left, right) {
				return errors.New("self-hosted state, exports, and secret paths must be disjoint")
			}
		}
	}
	for _, managed := range paths[:2] {
		if overlaps(configPath, managed) {
			return errors.New("self-hosted configuration must be outside managed state and exports")
		}
	}
	return nil
}

func absoluteCleanNonRoot(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && path != string(filepath.Separator)
}

func overlaps(left, right string) bool {
	if left == right {
		return true
	}
	for _, pair := range [][2]string{{left, right}, {right, left}} {
		relative, err := filepath.Rel(pair[0], pair[1])
		if err == nil &&
			relative != "." &&
			relative != ".." &&
			!filepath.IsAbs(relative) &&
			!strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
