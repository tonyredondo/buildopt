// Package edgecache owns the optional single-node Edge Cache boundary.
package edgecache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	SchemaVersion             = "buildopt.edge-cache/config/v1"
	Profile                   = "OWNER_POC_SINGLE_NODE"
	FilesystemPolicy          = "ALLOWLIST_PROVEN_LOCAL"
	CommitAuthority           = "SHARED_ONLY"
	CollisionAuthority        = "SHARED_ONLY"
	OfflineReadPolicy         = "COMMITTED_CURRENT_REVOCATION"
	OfflineWriteVisibility    = "ATTEMPT_ONLY_PENDING"
	CompressionPolicy         = "DISABLED_UNTIL_MEASURED"
	MinimumCapacityBytes      = int64(1 << 30)
	MaximumCapacityBytes      = int64(500 << 30)
	MaximumObjectBytes        = int64(100 << 20)
	MaximumStableTTL          = 30 * 24 * time.Hour
	MaximumPendingTTL         = 24 * time.Hour
	HighWatermarkPercent      = 85
	LowWatermarkPercent       = 75
	ProtectedPercent          = 80
	maximumConfigurationBytes = 64 << 10
)

type Config struct {
	SchemaVersion string    `json:"schemaVersion"`
	Profile       string    `json:"profile"`
	EdgeID        string    `json:"edgeId"`
	Server        Server    `json:"server"`
	Shared        Shared    `json:"shared"`
	Storage       Storage   `json:"storage"`
	Authority     Authority `json:"authority"`
	Policy        Policy    `json:"policy"`
}

type Server struct {
	Listen string `json:"listen"`
}

type Shared struct {
	BaseURL               string `json:"baseUrl"`
	CredentialPath        string `json:"credentialPath"`
	AllowInsecureLoopback bool   `json:"allowInsecureLoopback"`
}

type Storage struct {
	StateDirectory       string `json:"stateDirectory"`
	FilesystemPolicy     string `json:"filesystemPolicy"`
	CapacityBytes        int64  `json:"capacityBytes"`
	MaximumObjectBytes   int64  `json:"maximumObjectBytes"`
	StableTTLSeconds     int64  `json:"stableTtlSeconds"`
	PendingTTLSeconds    int64  `json:"pendingTtlSeconds"`
	HighWatermarkPercent int    `json:"highWatermarkPercent"`
	LowWatermarkPercent  int    `json:"lowWatermarkPercent"`
	ProtectedPercent     int    `json:"protectedPercent"`
}

type Authority struct {
	TrustRootPath string `json:"trustRootPath"`
	SnapshotPath  string `json:"snapshotPath"`
}

type Policy struct {
	CommitAuthority        string `json:"commitAuthority"`
	CollisionAuthority     string `json:"collisionAuthority"`
	OfflineReadPolicy      string `json:"offlineReadPolicy"`
	OfflineWriteVisibility string `json:"offlineWriteVisibility"`
	CompressionPolicy      string `json:"compressionPolicy"`
}

// Load reads one private, strict, path-only Edge deployment configuration.
func Load(path string) (Config, error) {
	if !absoluteCleanNonRoot(path) {
		return Config{}, errors.New("Edge configuration path must be absolute, clean, and non-root")
	}
	raw, err := localauthority.ReadPrivateFile(path, maximumConfigurationBytes)
	if err != nil {
		return Config{}, fmt.Errorf("read Edge configuration: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode Edge configuration: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("Edge configuration has trailing content")
	}
	if err := validate(config, path); err != nil {
		return Config{}, err
	}
	return config, nil
}

func validate(config Config, configPath string) error {
	if config.SchemaVersion != SchemaVersion || config.Profile != Profile {
		return errors.New("unsupported Edge configuration identity")
	}
	if !validIdentifier(config.EdgeID) {
		return errors.New("Edge ID must be a canonical identifier")
	}
	if err := validateLoopbackListener(config.Server.Listen); err != nil {
		return err
	}
	if err := validateSharedURL(config.Shared); err != nil {
		return err
	}
	if config.Storage.FilesystemPolicy != FilesystemPolicy ||
		config.Storage.CapacityBytes < MinimumCapacityBytes ||
		config.Storage.CapacityBytes > MaximumCapacityBytes ||
		config.Storage.MaximumObjectBytes < 1 ||
		config.Storage.MaximumObjectBytes > MaximumObjectBytes ||
		config.Storage.StableTTLSeconds < 1 ||
		config.Storage.StableTTLSeconds > int64(MaximumStableTTL/time.Second) ||
		config.Storage.PendingTTLSeconds < 1 ||
		config.Storage.PendingTTLSeconds > int64(MaximumPendingTTL/time.Second) ||
		config.Storage.HighWatermarkPercent != HighWatermarkPercent ||
		config.Storage.LowWatermarkPercent != LowWatermarkPercent ||
		config.Storage.ProtectedPercent != ProtectedPercent {
		return errors.New("Edge storage policy violates the owner-operated POC bounds")
	}
	if config.Policy.CommitAuthority != CommitAuthority ||
		config.Policy.CollisionAuthority != CollisionAuthority ||
		config.Policy.OfflineReadPolicy != OfflineReadPolicy ||
		config.Policy.OfflineWriteVisibility != OfflineWriteVisibility ||
		config.Policy.CompressionPolicy != CompressionPolicy {
		return errors.New("Edge authority policy cannot be relaxed")
	}
	paths := []string{
		config.Storage.StateDirectory,
		config.Shared.CredentialPath,
		config.Authority.TrustRootPath,
		config.Authority.SnapshotPath,
	}
	for _, candidate := range paths {
		if !absoluteCleanNonRoot(candidate) {
			return errors.New("Edge paths must be absolute, clean, and non-root")
		}
	}
	for index, left := range paths {
		for _, right := range paths[index+1:] {
			if overlaps(left, right) {
				return errors.New("Edge state and authority paths must be disjoint")
			}
		}
	}
	if overlaps(configPath, config.Storage.StateDirectory) {
		return errors.New("Edge configuration must be outside managed state")
	}
	return nil
}

func validateLoopbackListener(address string) error {
	host, portText, err := net.SplitHostPort(address)
	port, portErr := strconv.Atoi(portText)
	if err != nil || portErr != nil || host != "127.0.0.1" || port < 0 || port > 65535 {
		return errors.New("Edge listener must be a canonical IPv4 loopback address and valid port")
	}
	return nil
}

func validateSharedURL(shared Shared) error {
	parsed, err := url.Parse(shared.BaseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" {
		return errors.New("Shared base URL must be an origin without credentials, query, fragment, or path")
	}
	if portText := parsed.Port(); portText != "" {
		port, portErr := strconv.Atoi(portText)
		if portErr != nil || port < 1 || port > 65535 {
			return errors.New("Shared base URL has an invalid port")
		}
	}
	switch parsed.Scheme {
	case "https":
		if shared.AllowInsecureLoopback {
			return errors.New("HTTPS Shared origins cannot enable the loopback exception")
		}
	case "http":
		if !shared.AllowInsecureLoopback || parsed.Hostname() != "127.0.0.1" {
			return errors.New("insecure Shared transport is limited to explicit IPv4 loopback")
		}
	default:
		return errors.New("Shared base URL must use HTTPS")
	}
	return nil
}

func validIdentifier(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			(index > 0 && (character == '.' || character == '_' || character == '-')) {
			continue
		}
		return false
	}
	return true
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
		if err == nil && relative != "." && relative != ".." && !filepath.IsAbs(relative) &&
			!strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
