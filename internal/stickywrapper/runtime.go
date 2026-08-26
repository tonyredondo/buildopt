package stickywrapper

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LoadConfig reads the committed wrapper configuration used at invocation
// time. It applies the same canonical-byte and mode contract as wrapper check
// without requiring the other three generated files to be opened again.
func LoadConfig(root string) (Config, error) {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return Config{}, errors.New("wrapper root must be one clean absolute path")
	}
	path := filepath.Join(root, filepath.FromSlash(configPath))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 ||
		info.Size() < 1 || info.Size() > maximumBytes[configPath] {
		return Config{}, errors.New("wrapper configuration must be one bounded regular file without symlinks")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != fileModes[configPath] {
		return Config{}, errors.New("wrapper configuration has an unsafe mode")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, errors.New("wrapper configuration cannot be read")
	}
	if err := validateText(configPath, raw); err != nil {
		return Config{}, err
	}
	config, err := parseConfig(raw)
	if err != nil {
		return Config{}, err
	}
	if string(renderConfig(config)) != string(raw) {
		return Config{}, errors.New("wrapper configuration is not canonical")
	}
	return config, nil
}

// CredentialEnvironment returns a syntactically safe credential variable
// name even when another configuration field is invalid. The launcher uses
// this narrow recovery path only to keep a private value out of Gradle.
func CredentialEnvironment(root string) string {
	const fallback = "BUILDOPT_TOKEN"
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return fallback
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(configPath)))
	if err != nil || int64(len(raw)) > maximumBytes[configPath] {
		return fallback
	}
	const prefix = `credential_env = "`
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, `"`) {
			continue
		}
		value := strings.TrimSuffix(strings.TrimPrefix(line, prefix), `"`)
		if credentialPattern.MatchString(value) {
			return value
		}
	}
	return fallback
}
