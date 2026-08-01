package edgecache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validConfig(root string) Config {
	return Config{
		SchemaVersion: SchemaVersion,
		Profile:       Profile,
		EdgeID:        "edge-poc-1",
		Server:        Server{Listen: "127.0.0.1:8043"},
		Shared: Shared{
			BaseURL:        "https://shared.example.test:8443",
			CredentialPath: filepath.Join(root, "secrets", "shared-token"),
		},
		Storage: Storage{
			StateDirectory:       filepath.Join(root, "state"),
			FilesystemPolicy:     FilesystemPolicy,
			CapacityBytes:        10 << 30,
			MaximumObjectBytes:   MaximumObjectBytes,
			StableTTLSeconds:     int64(MaximumStableTTL.Seconds()),
			PendingTTLSeconds:    int64(MaximumPendingTTL.Seconds()),
			HighWatermarkPercent: HighWatermarkPercent,
			LowWatermarkPercent:  LowWatermarkPercent,
			ProtectedPercent:     ProtectedPercent,
		},
		Authority: Authority{
			TrustRootPath: filepath.Join(root, "secrets", "trust-root.json"),
			SnapshotPath:  filepath.Join(root, "authority", "current.json"),
		},
		Policy: Policy{
			CommitAuthority:        CommitAuthority,
			CollisionAuthority:     CollisionAuthority,
			OfflineReadPolicy:      OfflineReadPolicy,
			OfflineWriteVisibility: OfflineWriteVisibility,
			CompressionPolicy:      CompressionPolicy,
		},
	}
}

func writeConfig(t *testing.T, root string, value any, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(root, "edge.json")
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAcceptsExactPrivateConfiguration(t *testing.T) {
	root := t.TempDir()
	config, err := Load(writeConfig(t, root, validConfig(root), 0o600))
	if err != nil {
		t.Fatal(err)
	}
	if config.EdgeID != "edge-poc-1" || config.Policy.CommitAuthority != CommitAuthority {
		t.Fatalf("config = %+v", config)
	}
}

func TestLoadAcceptsOnlyExplicitLoopbackHTTPException(t *testing.T) {
	root := t.TempDir()
	config := validConfig(root)
	config.Shared.BaseURL = "http://127.0.0.1:8042"
	config.Shared.AllowInsecureLoopback = true
	if _, err := Load(writeConfig(t, root, config, 0o600)); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"remote http": func(value *Config) { value.Shared.BaseURL = "http://shared.example.test:8042" },
		"implicit":    func(value *Config) { value.Shared.AllowInsecureLoopback = false },
		"userinfo":    func(value *Config) { value.Shared.BaseURL = "http://user@127.0.0.1:8042" },
		"path":        func(value *Config) { value.Shared.BaseURL = "http://127.0.0.1:8042/cache" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := config
			mutate(&candidate)
			if _, err := Load(writeConfig(t, t.TempDir(), candidate, 0o600)); err == nil {
				t.Fatal("unsafe Shared URL accepted")
			}
		})
	}
}

func TestLoadRejectsUnknownFieldsModesAndTrailingDocuments(t *testing.T) {
	root := t.TempDir()
	config := validConfig(root)
	path := writeConfig(t, root, config, 0o644)
	if runtime.GOOS != "windows" {
		if _, err := Load(path); err == nil {
			t.Fatal("permissive configuration mode accepted")
		}
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	unknown := strings.Replace(string(raw), `"profile":`, `"unknown":true,"profile":`, 1)
	if err := os.WriteFile(path, []byte(unknown), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("unknown field accepted")
	}
	if err := os.WriteFile(path, append(raw, []byte("\n{}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("trailing document accepted")
	}
	link := filepath.Join(root, "edge-link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("configuration symlink accepted")
	}
}

func TestLoadRejectsRelaxedAuthorityAndStoragePolicies(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"commit authority": func(value *Config) { value.Policy.CommitAuthority = "EDGE" },
		"collision":        func(value *Config) { value.Policy.CollisionAuthority = "FIRST_EDGE" },
		"offline read":     func(value *Config) { value.Policy.OfflineReadPolicy = "ANY_LOCAL" },
		"offline write":    func(value *Config) { value.Policy.OfflineWriteVisibility = "STABLE" },
		"compression":      func(value *Config) { value.Policy.CompressionPolicy = "ALWAYS" },
		"small capacity":   func(value *Config) { value.Storage.CapacityBytes = MinimumCapacityBytes - 1 },
		"large object":     func(value *Config) { value.Storage.MaximumObjectBytes = MaximumObjectBytes + 1 },
		"long stable ttl":  func(value *Config) { value.Storage.StableTTLSeconds++ },
		"long pending ttl": func(value *Config) { value.Storage.PendingTTLSeconds++ },
		"watermark":        func(value *Config) { value.Storage.LowWatermarkPercent = 50 },
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			config := validConfig(root)
			mutate(&config)
			if _, err := Load(writeConfig(t, root, config, 0o600)); err == nil {
				t.Fatal("relaxed Edge policy accepted")
			}
		})
	}
}

func TestLoadRejectsUnsafeOrOverlappingPaths(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"relative state": func(value *Config) { value.Storage.StateDirectory = "state" },
		"root state":     func(value *Config) { value.Storage.StateDirectory = "/" },
		"secret in state": func(value *Config) {
			value.Shared.CredentialPath = filepath.Join(value.Storage.StateDirectory, "token")
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			config := validConfig(root)
			mutate(&config)
			if _, err := Load(writeConfig(t, root, config, 0o600)); err == nil {
				t.Fatal("unsafe Edge path accepted")
			}
		})
	}
	root := t.TempDir()
	config := validConfig(root)
	config.Storage.StateDirectory = root
	if _, err := Load(writeConfig(t, root, config, 0o600)); err == nil {
		t.Fatal("configuration inside managed state accepted")
	}
}

func TestLoadRejectsUnsafeIdentityAndListener(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"uppercase id": func(value *Config) { value.EdgeID = "Edge-1" },
		"network":      func(value *Config) { value.Server.Listen = "0.0.0.0:8043" },
		"hostname":     func(value *Config) { value.Server.Listen = "localhost:8043" },
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			config := validConfig(root)
			mutate(&config)
			if _, err := Load(writeConfig(t, root, config, 0o600)); err == nil {
				t.Fatal("unsafe Edge identity or listener accepted")
			}
		})
	}
}

func TestCheckedInExampleLoadsThroughProductionBoundary(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	example := "edge-cache.example.json"
	if runtime.GOOS == "windows" {
		example = "edge-cache.windows.example.json"
	}
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, "specs", example))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "edge.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.EdgeID != "edge-poc-1" || config.Shared.AllowInsecureLoopback {
		t.Fatalf("example config = %+v", config)
	}
}
