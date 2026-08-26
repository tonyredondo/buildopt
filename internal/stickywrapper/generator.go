// Package stickywrapper owns the repository-committed BuildOpt Wrapper files,
// their deterministic generation and the invocation-time portable
// configuration boundary. Download, process execution and authenticated
// central-service probes remain launcher responsibilities.
package stickywrapper

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	propertiesPath = ".buildopt/wrapper.properties"
	configPath     = ".buildopt/config.toml"
	posixPath      = "buildoptw"
	windowsPath    = "buildoptw.bat"
	lockPath       = ".buildopt-wrapper.lock"
)

var (
	targetPaths = []string{posixPath, windowsPath, propertiesPath, configPath}
	fileModes   = map[string]fs.FileMode{
		posixPath:      0o755,
		windowsPath:    0o644,
		propertiesPath: 0o644,
		configPath:     0o644,
	}
	maximumBytes = map[string]int64{
		posixPath:      32 << 10,
		windowsPath:    32 << 10,
		propertiesPath: 16 << 10,
		configPath:     16 << 10,
	}
	propertyKeys = []string{
		"schemaVersion",
		"distributionVersion",
		"distributionUrl.linux-amd64",
		"distributionSha256.linux-amd64",
		"distributionUrl.macos-amd64",
		"distributionSha256.macos-amd64",
		"distributionUrl.macos-arm64",
		"distributionSha256.macos-arm64",
		"distributionUrl.windows-amd64",
		"distributionSha256.windows-amd64",
		"network.connectTimeoutMs",
		"network.readTimeoutMs",
		"network.redirectPolicy",
		"network.proxyMode",
	}
	configKeys = []string{
		"schema_version",
		"mode",
		"server_url",
		"project_scope",
		"credential_env",
		"trial_budget_percent",
	}
	versionPattern    = regexp.MustCompile(`^([0-9]+)[.]([0-9]+)[.]([0-9]+)$`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	projectPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}/[a-z0-9][a-z0-9._-]{0,62}$`)
	credentialPattern = regexp.MustCompile(`^BUILDOPT_[A-Z0-9_]{1,55}$`)
)

//go:embed templates/buildoptw
var posixWrapper []byte

//go:embed templates/buildoptw.bat
var windowsWrapper []byte

// ErrorKind classifies generator failures into stable CLI exit categories.
type ErrorKind uint8

const (
	ErrorUsage ErrorKind = iota + 1
	ErrorCommittedData
	ErrorNetwork
	ErrorInternal
)

// Error reports a classified wrapper-generator failure without sensitive data.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (err *Error) Error() string { return err.Err.Error() }
func (err *Error) Unwrap() error { return err.Err }

func classified(kind ErrorKind, format string, args ...any) error {
	return &Error{Kind: kind, Err: fmt.Errorf(format, args...)}
}

// Distribution identifies one immutable published platform archive.
type Distribution struct {
	URL    string
	SHA256 string
}

// Release is the immutable distribution identity written to wrapper.properties.
type Release struct {
	Version       string
	Distributions map[string]Distribution
}

// Resolver supplies immutable public release metadata without downloading or
// executing a distribution archive.
type Resolver interface {
	Latest(context.Context) (Release, error)
	Version(context.Context, string) (Release, error)
}

// Config contains the portable owner-controlled wrapper configuration.
type Config struct {
	Mode               string
	ServerURL          string
	ProjectScope       string
	CredentialEnv      string
	TrialBudgetPercent int
}

// DefaultConfig returns the deterministic configuration used by init.
func DefaultConfig() Config {
	return Config{Mode: "auto", TrialBudgetPercent: 5}
}

// Snapshot is a validated, canonical wrapper state.
type Snapshot struct {
	Release Release
	Config  Config
	Files   map[string][]byte
}

// Generator creates and validates the four repository wrapper files.
type Generator struct {
	Root     string
	Resolver Resolver
	rename   func(string, string) error
}

// Init creates all four files and refuses to overwrite any existing target.
func (generator Generator) Init(ctx context.Context, config Config) (Snapshot, error) {
	if err := generator.validateRoot(); err != nil {
		return Snapshot{}, err
	}
	if err := validateConfig(config); err != nil {
		return Snapshot{}, err
	}
	if err := generator.ensureTargetsAbsent(); err != nil {
		return Snapshot{}, err
	}
	if generator.Resolver == nil {
		return Snapshot{}, classified(ErrorInternal, "release resolver is unavailable")
	}
	release, err := generator.Resolver.Latest(ctx)
	if err != nil {
		return Snapshot{}, classified(ErrorNetwork, "resolve latest release: %v", err)
	}
	if err := validateRelease(release); err != nil {
		return Snapshot{}, err
	}
	snapshot := canonicalSnapshot(release, config)
	if err := generator.publish(snapshot.Files, false); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// Check validates all four files without creating, locking or modifying state.
func (generator Generator) Check() (Snapshot, error) {
	if err := generator.validateRoot(); err != nil {
		return Snapshot{}, err
	}
	files := make(map[string][]byte, len(targetPaths))
	for _, relative := range targetPaths {
		absolute := filepath.Join(generator.Root, filepath.FromSlash(relative))
		info, err := os.Lstat(absolute)
		if err != nil {
			return Snapshot{}, classified(ErrorCommittedData, "%s is unavailable: %v", relative, err)
		}
		if !info.Mode().IsRegular() {
			return Snapshot{}, classified(ErrorCommittedData, "%s is not a regular file", relative)
		}
		if info.Size() > maximumBytes[relative] {
			return Snapshot{}, classified(ErrorCommittedData, "%s exceeds its size limit", relative)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != fileModes[relative] {
			return Snapshot{}, classified(
				ErrorCommittedData,
				"%s mode is %04o, want %04o",
				relative,
				info.Mode().Perm(),
				fileModes[relative],
			)
		}
		raw, readErr := os.ReadFile(absolute)
		if readErr != nil {
			return Snapshot{}, classified(ErrorCommittedData, "read %s: %v", relative, readErr)
		}
		if err := validateText(relative, raw); err != nil {
			return Snapshot{}, err
		}
		files[relative] = raw
	}

	release, err := parseProperties(files[propertiesPath])
	if err != nil {
		return Snapshot{}, err
	}
	config, err := parseConfig(files[configPath])
	if err != nil {
		return Snapshot{}, err
	}
	canonical := canonicalSnapshot(release, config)
	for _, relative := range targetPaths {
		if string(files[relative]) != string(canonical.Files[relative]) {
			return Snapshot{}, classified(ErrorCommittedData, "%s is not canonical", relative)
		}
	}
	return canonical, nil
}

// Update changes only immutable distribution identity after validating that
// all current files are canonical. Same-version updates perform no writes.
func (generator Generator) Update(
	ctx context.Context,
	version string,
	allowDowngrade bool,
) (before Snapshot, after Snapshot, changed bool, err error) {
	before, err = generator.Check()
	if err != nil {
		return Snapshot{}, Snapshot{}, false, err
	}
	comparison, err := compareVersions(version, before.Release.Version)
	if err != nil {
		return Snapshot{}, Snapshot{}, false, err
	}
	if comparison == 0 {
		return before, before, false, nil
	}
	if comparison < 0 && !allowDowngrade {
		return Snapshot{}, Snapshot{}, false, classified(
			ErrorCommittedData,
			"distribution downgrade %s -> %s requires --allow-downgrade",
			before.Release.Version,
			version,
		)
	}
	if generator.Resolver == nil {
		return Snapshot{}, Snapshot{}, false, classified(ErrorInternal, "release resolver is unavailable")
	}
	release, resolveErr := generator.Resolver.Version(ctx, version)
	if resolveErr != nil {
		return Snapshot{}, Snapshot{}, false, classified(
			ErrorNetwork,
			"resolve release %s: %v",
			version,
			resolveErr,
		)
	}
	if err := validateRelease(release); err != nil {
		return Snapshot{}, Snapshot{}, false, err
	}
	if release.Version != version {
		return Snapshot{}, Snapshot{}, false, classified(
			ErrorCommittedData,
			"resolved release version %s does not match requested %s",
			release.Version,
			version,
		)
	}
	after = canonicalSnapshot(release, before.Config)
	if err := generator.publish(after.Files, true); err != nil {
		return Snapshot{}, Snapshot{}, false, err
	}
	return before, after, true, nil
}

func (generator Generator) validateRoot() error {
	if generator.Root == "" || !filepath.IsAbs(generator.Root) {
		return classified(ErrorInternal, "wrapper root must be absolute")
	}
	info, err := os.Stat(generator.Root)
	if err != nil || !info.IsDir() {
		return classified(ErrorInternal, "wrapper root is unavailable")
	}
	return nil
}

func (generator Generator) ensureTargetsAbsent() error {
	for _, relative := range targetPaths {
		_, err := os.Lstat(filepath.Join(generator.Root, filepath.FromSlash(relative)))
		switch {
		case err == nil:
			return classified(ErrorCommittedData, "%s already exists", relative)
		case errors.Is(err, fs.ErrNotExist):
			continue
		default:
			return classified(ErrorInternal, "inspect %s: %v", relative, err)
		}
	}
	return nil
}

func canonicalSnapshot(release Release, config Config) Snapshot {
	files := map[string][]byte{
		posixPath:      append([]byte(nil), posixWrapper...),
		windowsPath:    append([]byte(nil), windowsWrapper...),
		propertiesPath: renderProperties(release),
		configPath:     renderConfig(config),
	}
	return Snapshot{Release: release, Config: config, Files: files}
}

func renderProperties(release Release) []byte {
	values := map[string]string{
		"schemaVersion":                    "buildopt.wrapper/v1",
		"distributionVersion":              release.Version,
		"distributionUrl.linux-amd64":      release.Distributions["linux-amd64"].URL,
		"distributionSha256.linux-amd64":   release.Distributions["linux-amd64"].SHA256,
		"distributionUrl.macos-amd64":      release.Distributions["macos-amd64"].URL,
		"distributionSha256.macos-amd64":   release.Distributions["macos-amd64"].SHA256,
		"distributionUrl.macos-arm64":      release.Distributions["macos-arm64"].URL,
		"distributionSha256.macos-arm64":   release.Distributions["macos-arm64"].SHA256,
		"distributionUrl.windows-amd64":    release.Distributions["windows-amd64"].URL,
		"distributionSha256.windows-amd64": release.Distributions["windows-amd64"].SHA256,
		"network.connectTimeoutMs":         "5000",
		"network.readTimeoutMs":            "30000",
		"network.redirectPolicy":           "https-only-max-5",
		"network.proxyMode":                "environment",
	}
	var builder strings.Builder
	for _, key := range propertyKeys {
		_, _ = fmt.Fprintf(&builder, "%s=%s\n", key, values[key])
	}
	return []byte(builder.String())
}

func renderConfig(config Config) []byte {
	values := map[string]string{
		"schema_version":       `"buildopt.config/v1"`,
		"mode":                 strconv.Quote(config.Mode),
		"server_url":           strconv.Quote(config.ServerURL),
		"project_scope":        strconv.Quote(config.ProjectScope),
		"credential_env":       strconv.Quote(config.CredentialEnv),
		"trial_budget_percent": strconv.Itoa(config.TrialBudgetPercent),
	}
	var builder strings.Builder
	for _, key := range configKeys {
		_, _ = fmt.Fprintf(&builder, "%s = %s\n", key, values[key])
	}
	return []byte(builder.String())
}

func validateText(relative string, raw []byte) error {
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		return classified(ErrorCommittedData, "%s must end with LF", relative)
	}
	if strings.ContainsRune(string(raw), '\r') || strings.HasPrefix(string(raw), "\ufeff") {
		return classified(ErrorCommittedData, "%s must be LF-only without BOM", relative)
	}
	if relative == propertiesPath {
		for _, value := range raw {
			if value > 0x7f {
				return classified(ErrorCommittedData, "%s must be US-ASCII", relative)
			}
		}
	}
	return nil
}

func parseOrdered(raw []byte, keys []string, separator string) (map[string]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	values := make(map[string]string, len(keys))
	index := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || index >= len(keys) {
			return nil, classified(ErrorCommittedData, "unexpected blank or extra entry")
		}
		parts := strings.SplitN(line, separator, 2)
		if len(parts) != 2 || parts[0] != keys[index] {
			return nil, classified(ErrorCommittedData, "entry %d must be %s", index+1, keys[index])
		}
		if _, duplicate := values[parts[0]]; duplicate {
			return nil, classified(ErrorCommittedData, "duplicate key %s", parts[0])
		}
		values[parts[0]] = parts[1]
		index++
	}
	if err := scanner.Err(); err != nil {
		return nil, classified(ErrorCommittedData, "read committed data: %v", err)
	}
	if index != len(keys) {
		return nil, classified(ErrorCommittedData, "missing committed entries")
	}
	return values, nil
}

func parseProperties(raw []byte) (Release, error) {
	values, err := parseOrdered(raw, propertyKeys, "=")
	if err != nil {
		return Release{}, err
	}
	if values["schemaVersion"] != "buildopt.wrapper/v1" ||
		values["network.connectTimeoutMs"] != "5000" ||
		values["network.readTimeoutMs"] != "30000" ||
		values["network.redirectPolicy"] != "https-only-max-5" ||
		values["network.proxyMode"] != "environment" {
		return Release{}, classified(ErrorCommittedData, "wrapper properties contain unsupported fixed values")
	}
	release := Release{
		Version:       values["distributionVersion"],
		Distributions: map[string]Distribution{},
	}
	for _, platform := range []string{"linux-amd64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		release.Distributions[platform] = Distribution{
			URL:    values["distributionUrl."+platform],
			SHA256: values["distributionSha256."+platform],
		}
	}
	if err := validateRelease(release); err != nil {
		return Release{}, err
	}
	return release, nil
}

func parseConfig(raw []byte) (Config, error) {
	values, err := parseOrdered(raw, configKeys, " = ")
	if err != nil {
		return Config{}, err
	}
	unquote := func(key string) (string, error) {
		value := values[key]
		if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' || strings.Contains(value[1:len(value)-1], `\`) {
			return "", classified(ErrorCommittedData, "%s must be a plain quoted string", key)
		}
		return value[1 : len(value)-1], nil
	}
	schema, err := unquote("schema_version")
	if err != nil || schema != "buildopt.config/v1" {
		return Config{}, classified(ErrorCommittedData, "unsupported config schema")
	}
	mode, err := unquote("mode")
	if err != nil {
		return Config{}, err
	}
	server, err := unquote("server_url")
	if err != nil {
		return Config{}, err
	}
	project, err := unquote("project_scope")
	if err != nil {
		return Config{}, err
	}
	credential, err := unquote("credential_env")
	if err != nil {
		return Config{}, err
	}
	budget, err := strconv.Atoi(values["trial_budget_percent"])
	if err != nil {
		return Config{}, classified(ErrorCommittedData, "trial_budget_percent must be an integer")
	}
	config := Config{
		Mode: mode, ServerURL: server, ProjectScope: project,
		CredentialEnv: credential, TrialBudgetPercent: budget,
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func validateRelease(release Release) error {
	if !versionPattern.MatchString(release.Version) {
		return classified(ErrorCommittedData, "invalid distribution version %q", release.Version)
	}
	for _, platform := range []string{"linux-amd64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		distribution, ok := release.Distributions[platform]
		if !ok {
			return classified(ErrorCommittedData, "missing %s distribution", platform)
		}
		parsed, err := url.Parse(distribution.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return classified(ErrorCommittedData, "invalid %s distribution URL", platform)
		}
		if !strings.Contains(parsed.EscapedPath(), release.Version) || strings.Contains(strings.ToLower(parsed.EscapedPath()), "/latest/") {
			return classified(ErrorCommittedData, "%s distribution URL is not immutable", platform)
		}
		if !sha256Pattern.MatchString(distribution.SHA256) {
			return classified(ErrorCommittedData, "invalid %s distribution SHA-256", platform)
		}
	}
	return nil
}

func validateConfig(config Config) error {
	for name, value := range map[string]string{
		"mode": config.Mode, "server URL": config.ServerURL,
		"project scope": config.ProjectScope, "credential environment": config.CredentialEnv,
	} {
		if strings.ContainsAny(value, "\"\\\r\n") {
			return classified(ErrorCommittedData, "%s contains unsupported characters", name)
		}
	}
	switch config.Mode {
	case "auto", "observe", "off":
	default:
		return classified(ErrorCommittedData, "invalid wrapper mode %q", config.Mode)
	}
	if config.TrialBudgetPercent < 0 || config.TrialBudgetPercent > 5 {
		return classified(ErrorCommittedData, "trial budget must be between 0 and 5")
	}
	present := config.ServerURL != "" || config.ProjectScope != "" || config.CredentialEnv != ""
	if present && (config.ServerURL == "" || config.ProjectScope == "" || config.CredentialEnv == "") {
		return classified(ErrorCommittedData, "server URL, project scope and credential environment must be all empty or all present")
	}
	if !present {
		return nil
	}
	parsed, err := url.Parse(config.ServerURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return classified(ErrorCommittedData, "invalid server URL")
	}
	if parsed.Scheme != "https" {
		host := parsed.Hostname()
		address := net.ParseIP(host)
		if parsed.Scheme != "http" || address == nil || !address.IsLoopback() {
			return classified(ErrorCommittedData, "server URL must use HTTPS or an IP loopback HTTP address")
		}
	}
	if !projectPattern.MatchString(config.ProjectScope) {
		return classified(ErrorCommittedData, "invalid project scope")
	}
	if !credentialPattern.MatchString(config.CredentialEnv) {
		return classified(ErrorCommittedData, "invalid credential environment name")
	}
	return nil
}

func compareVersions(left, right string) (int, error) {
	parse := func(value string) ([3]uint64, error) {
		matches := versionPattern.FindStringSubmatch(value)
		if matches == nil {
			return [3]uint64{}, classified(ErrorCommittedData, "invalid distribution version %q", value)
		}
		var result [3]uint64
		for index := range result {
			component, err := strconv.ParseUint(matches[index+1], 10, 32)
			if err != nil {
				return [3]uint64{}, classified(ErrorCommittedData, "invalid distribution version %q", value)
			}
			result[index] = component
		}
		return result, nil
	}
	leftParts, err := parse(left)
	if err != nil {
		return 0, err
	}
	rightParts, err := parse(right)
	if err != nil {
		return 0, err
	}
	for index := range leftParts {
		if leftParts[index] < rightParts[index] {
			return -1, nil
		}
		if leftParts[index] > rightParts[index] {
			return 1, nil
		}
	}
	return 0, nil
}

type stagedFile struct {
	relative string
	target   string
	temp     string
	backup   string
}

func (generator Generator) publish(files map[string][]byte, replace bool) (returnErr error) {
	transactionLock := filepath.Join(generator.Root, lockPath)
	if err := os.Mkdir(transactionLock, 0o700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return classified(ErrorInternal, "wrapper transaction is busy")
		}
		return classified(ErrorInternal, "create wrapper transaction lock: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(transactionLock); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			returnErr = errors.Join(returnErr, removeErr)
		}
	}()
	if replace {
		if _, err := generator.Check(); err != nil {
			return err
		}
	} else if err := generator.ensureTargetsAbsent(); err != nil {
		return err
	}

	staged := make([]stagedFile, 0, len(targetPaths))
	defer func() {
		for _, file := range staged {
			if file.temp != "" {
				_ = os.Remove(file.temp)
			}
			if file.backup != "" {
				_ = os.Remove(file.backup)
			}
		}
	}()
	for _, relative := range targetPaths {
		target := filepath.Join(generator.Root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return classified(ErrorInternal, "create wrapper directory: %v", err)
		}
		temporary, err := os.CreateTemp(filepath.Dir(target), ".buildopt-wrapper-*.tmp")
		if err != nil {
			return classified(ErrorInternal, "stage %s: %v", relative, err)
		}
		temporaryPath := temporary.Name()
		if _, err = temporary.Write(files[relative]); err == nil {
			err = temporary.Chmod(fileModes[relative])
		}
		if err == nil {
			err = temporary.Sync()
		}
		closeErr := temporary.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(temporaryPath)
			return classified(ErrorInternal, "stage %s: %v", relative, err)
		}
		staged = append(staged, stagedFile{relative: relative, target: target, temp: temporaryPath})
	}

	rename := generator.rename
	if rename == nil {
		rename = os.Rename
	}
	if replace {
		for index := range staged {
			backup, err := os.CreateTemp(filepath.Dir(staged[index].target), ".buildopt-wrapper-backup-*")
			if err != nil {
				return classified(ErrorInternal, "reserve backup for %s: %v", staged[index].relative, err)
			}
			backupPath := backup.Name()
			if closeErr := backup.Close(); closeErr != nil {
				_ = os.Remove(backupPath)
				return classified(ErrorInternal, "close backup for %s: %v", staged[index].relative, closeErr)
			}
			if err := os.Remove(backupPath); err != nil {
				return classified(ErrorInternal, "prepare backup for %s: %v", staged[index].relative, err)
			}
			if err := rename(staged[index].target, backupPath); err != nil {
				for rollback := index - 1; rollback >= 0; rollback-- {
					_ = rename(staged[rollback].backup, staged[rollback].target)
					staged[rollback].backup = ""
				}
				return classified(ErrorInternal, "backup %s: %v", staged[index].relative, err)
			}
			staged[index].backup = backupPath
		}
	}

	installed := 0
	for index := range staged {
		if err := rename(staged[index].temp, staged[index].target); err != nil {
			for rollback := 0; rollback < installed; rollback++ {
				_ = os.Remove(staged[rollback].target)
			}
			if replace {
				for rollback := range staged {
					if staged[rollback].backup != "" {
						_ = rename(staged[rollback].backup, staged[rollback].target)
						staged[rollback].backup = ""
					}
				}
			}
			return classified(ErrorInternal, "publish %s: %v", staged[index].relative, err)
		}
		staged[index].temp = ""
		installed++
	}
	for index := range staged {
		if staged[index].backup != "" {
			if err := os.Remove(staged[index].backup); err != nil {
				return classified(ErrorInternal, "remove backup for %s: %v", staged[index].relative, err)
			}
			staged[index].backup = ""
		}
	}
	return nil
}
