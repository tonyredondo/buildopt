package selfhosted

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig(root string) Config {
	return Config{
		SchemaVersion: SchemaVersion,
		Profile:       Profile,
		Server:        Server{Listen: "127.0.0.1:8042"},
		Storage:       Storage{StateDirectory: filepath.Join(root, "state"), FilesystemPolicy: FilesystemPolicy, MinimumDeploymentBytes: MinimumDeploymentBytes, MaximumDeploymentBytes: MaximumDeploymentBytes, UsableVolumePercent: UsableVolumePercent},
		Export:        Export{Directory: filepath.Join(root, "exports"), Profile: "summary"},
		Cache:         Cache{AuthorityPath: filepath.Join(root, "secrets", "authority.json"), TrustRootPath: filepath.Join(root, "secrets", "trust.json"), CredentialPath: filepath.Join(root, "secrets", "credential"), BetaTokenAuthentication: true},
	}
}

func writeConfig(t *testing.T, root string, config any, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(root, "config.json")
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAcceptsExactPrivatePathOnlyConfiguration(t *testing.T) {
	root := t.TempDir()
	path := writeConfig(t, root, validConfig(root), 0o600)
	config, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Profile != Profile || !config.Cache.BetaTokenAuthentication {
		t.Fatalf("config = %+v", config)
	}
}

func TestLoadRejectsUnknownFieldsModesAndTrailingDocuments(t *testing.T) {
	root := t.TempDir()
	config := validConfig(root)
	path := writeConfig(t, root, config, 0o644)
	if _, err := Load(path); err == nil {
		t.Fatal("permissive mode accepted")
	}
	linkPath := filepath.Join(root, "config-link.json")
	if err := os.Symlink(path, linkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(linkPath); err == nil {
		t.Fatal("configuration symlink accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
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
}

func TestLoadRejectsUnsafePolicyPathsAndCapabilities(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"network listener":  func(c *Config) { c.Server.Listen = "0.0.0.0:8042" },
		"relative state":    func(c *Config) { c.Storage.StateDirectory = "state" },
		"overlap":           func(c *Config) { c.Export.Directory = filepath.Join(c.Storage.StateDirectory, "exports") },
		"storage drift":     func(c *Config) { c.Storage.UsableVolumePercent = 75 },
		"diagnostic export": func(c *Config) { c.Export.Profile = "diagnostic" },
		"token bypass":      func(c *Config) { c.Cache.BetaTokenAuthentication = false },
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			config := validConfig(root)
			mutate(&config)
			if _, err := Load(writeConfig(t, root, config, 0o600)); err == nil {
				t.Fatal("unsafe configuration accepted")
			}
		})
	}
}

func TestLoadRejectsConfigurationInsideManagedState(t *testing.T) {
	root := t.TempDir()
	config := validConfig(root)
	config.Storage.StateDirectory = root
	path := writeConfig(t, root, config, 0o600)
	if _, err := Load(path); err == nil {
		t.Fatal("managed configuration accepted")
	}
}
