package stickywrapper

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadConfigRequiresCanonicalPortableFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(configPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	config := Config{
		Mode: "auto", ServerURL: "https://buildopt.example.com",
		ProjectScope: "example/repository", CredentialEnv: "BUILDOPT_TEAM_TOKEN",
		TrialBudgetPercent: 5,
	}
	if err := os.WriteFile(path, renderConfig(config), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(root)
	if err != nil || loaded != config {
		t.Fatalf("load config = %+v/%v", loaded, err)
	}
	if actual := CredentialEnvironment(root); actual != "BUILDOPT_TEAM_TOKEN" {
		t.Fatalf("credential environment = %q", actual)
	}

	if err := os.WriteFile(path, append(renderConfig(config), '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(root); err == nil {
		t.Fatal("non-canonical configuration was accepted")
	}
	if actual := CredentialEnvironment(root); actual != "BUILDOPT_TEAM_TOKEN" {
		t.Fatalf("safe credential recovery = %q", actual)
	}
}

func TestLoadConfigRejectsUnsafeFileAndUsesSafeCredentialFallback(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(configPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(outside, renderConfig(DefaultConfig()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(root); err == nil {
		t.Fatal("symlinked configuration was accepted")
	}
	if actual := CredentialEnvironment(root); actual != "BUILDOPT_TOKEN" {
		t.Fatalf("fallback credential environment = %q", actual)
	}
	if runtime.GOOS != "windows" {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, renderConfig(DefaultConfig()), 0o666); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o666); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(root); err == nil {
			t.Fatal("unsafe configuration mode was accepted")
		}
	}
}
