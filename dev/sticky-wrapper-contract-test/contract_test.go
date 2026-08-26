// Package wrappercontracttest validates the repository wrapper format before
// the production generator and bootstrap exist.
package wrappercontracttest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

var (
	propertiesOrder = []string{
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
	configOrder = []string{
		"schema_version",
		"mode",
		"server_url",
		"project_scope",
		"credential_env",
		"trial_budget_percent",
	}
	semverPattern     = regexp.MustCompile(`^[0-9]+[.][0-9]+[.][0-9]+$`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	projectPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}/[a-z0-9][a-z0-9._-]{0,62}$`)
	credentialPattern = regexp.MustCompile(`^BUILDOPT_[A-Z0-9_]{1,55}$`)
)

type contractError struct {
	Class string
	Text  string
}

func (err *contractError) Error() string {
	return err.Class + ": " + err.Text
}

func reject(class, format string, values ...any) error {
	return &contractError{Class: class, Text: fmt.Sprintf(format, values...)}
}

func errorClass(err error) string {
	var typed *contractError
	if errors.As(err, &typed) {
		return typed.Class
	}
	return ""
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve contract test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
}

func readFixture(t *testing.T, relative string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(
		repositoryRoot(t),
		"fixtures",
		"sticky-wrapper-contract",
		relative,
	))
	if err != nil {
		t.Fatalf("read fixture %s: %v", relative, err)
	}
	return contents
}

func validateText(data []byte, ascii bool) error {
	if len(data) == 0 || !utf8.Valid(data) {
		return reject("ENCODING", "input is empty or invalid UTF-8")
	}
	if strings.HasPrefix(string(data), "\ufeff") {
		return reject("ENCODING", "BOM is forbidden")
	}
	if strings.ContainsRune(string(data), '\x00') || strings.ContainsRune(string(data), '\r') {
		return reject("NEWLINE", "NUL and CR are forbidden")
	}
	if data[len(data)-1] != '\n' {
		return reject("NEWLINE", "one final LF is required")
	}
	if ascii {
		for _, value := range data {
			if value > 0x7f {
				return reject("ENCODING", "properties must be US-ASCII")
			}
		}
	}
	return nil
}

func machineSpecific(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "file:") ||
		strings.Contains(lower, "/home/") ||
		strings.Contains(lower, "${home}") ||
		strings.Contains(lower, "%userprofile%") ||
		strings.HasPrefix(value, `\\`) {
		return true
	}
	return len(value) >= 3 &&
		((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) &&
		value[1] == ':' && (value[2] == '\\' || value[2] == '/')
}

func securitySensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, fragment := range []string{
		"token",
		"password",
		"secret",
		"private_key",
		"authorization",
		"proxy_url",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

type keyValue struct {
	key   string
	value string
}

func validatePairs(pairs []keyValue, order []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, reject("SYNTAX", "no values")
	}
	values := make(map[string]string, len(pairs))
	for position, pair := range pairs {
		if securitySensitiveKey(pair.key) {
			return nil, reject("SECURITY_SENSITIVE_KEY", "forbidden key %q", pair.key)
		}
		if _, found := values[pair.key]; found {
			return nil, reject("DUPLICATE_KEY", "duplicate key %q", pair.key)
		}
		if position >= len(order) || pair.key != order[position] {
			known := false
			for _, allowed := range order {
				known = known || pair.key == allowed
			}
			if !known {
				return nil, reject("UNKNOWN_KEY", "unknown key %q", pair.key)
			}
			return nil, reject("INVALID_ORDER", "key %q at position %d", pair.key, position)
		}
		if machineSpecific(pair.value) {
			return nil, reject("MACHINE_SPECIFIC_VALUE", "machine-specific value for %q", pair.key)
		}
		values[pair.key] = pair.value
	}
	if len(pairs) != len(order) {
		return nil, reject("MISSING_KEY", "expected %d keys, found %d", len(order), len(pairs))
	}
	return values, nil
}

func propertyLines(data []byte, windows bool) ([]keyValue, error) {
	if err := validateText(data, true); err != nil {
		return nil, err
	}
	text := strings.TrimSuffix(string(data), "\n")
	lines := strings.Split(text, "\n")
	pairs := make([]keyValue, 0, len(lines))
	for _, line := range lines {
		if line == "" || line != strings.TrimSpace(line) {
			return nil, reject("SYNTAX", "blank line or surrounding whitespace")
		}
		var key, value string
		var found bool
		if windows {
			separator := strings.IndexByte(line, '=')
			if separator >= 0 {
				key, value, found = line[:separator], line[separator+1:], true
			}
		} else {
			key, value, found = strings.Cut(line, "=")
		}
		if !found || key == "" || value == "" || strings.ContainsAny(key, " \t") {
			return nil, reject("SYNTAX", "invalid property line")
		}
		pairs = append(pairs, keyValue{key: key, value: value})
	}
	return pairs, nil
}

func parseProperties(data []byte, windows bool) (map[string]string, error) {
	pairs, err := propertyLines(data, windows)
	if err != nil {
		return nil, err
	}
	values, err := validatePairs(pairs, propertiesOrder)
	if err != nil {
		return nil, err
	}
	if values["schemaVersion"] != "buildopt.wrapper/v1" ||
		!semverPattern.MatchString(values["distributionVersion"]) {
		return nil, reject("INVALID_VALUE", "invalid wrapper schema or distribution version")
	}
	version := values["distributionVersion"]
	for _, platform := range []string{"linux-amd64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		distributionURL := values["distributionUrl."+platform]
		parsed, parseErr := url.Parse(distributionURL)
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" ||
			parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, reject("INSECURE_URL", "invalid distribution URL for %s", platform)
		}
		if !strings.Contains(parsed.EscapedPath(), "/v"+version+"/") ||
			strings.Contains(strings.ToLower(parsed.EscapedPath()), "latest") {
			return nil, reject("INVALID_VALUE", "distribution URL is not version-bound")
		}
		if !sha256Pattern.MatchString(values["distributionSha256."+platform]) {
			return nil, reject("INVALID_VALUE", "invalid SHA-256 for %s", platform)
		}
	}
	if values["network.connectTimeoutMs"] != "5000" ||
		values["network.readTimeoutMs"] != "30000" ||
		values["network.redirectPolicy"] != "https-only-max-5" ||
		values["network.proxyMode"] != "environment" {
		return nil, reject("INVALID_VALUE", "network policy differs from v1")
	}
	return values, nil
}

func configLines(data []byte, windows bool) ([]keyValue, error) {
	if err := validateText(data, false); err != nil {
		return nil, err
	}
	text := strings.TrimSuffix(string(data), "\n")
	lines := strings.Split(text, "\n")
	pairs := make([]keyValue, 0, len(lines))
	for _, line := range lines {
		if line == "" || line != strings.TrimSpace(line) || strings.ContainsRune(line, '\t') {
			return nil, reject("SYNTAX", "blank line, tab or surrounding whitespace")
		}
		var key, raw string
		var found bool
		if windows {
			separator := strings.Index(line, " = ")
			if separator >= 0 {
				key, raw, found = line[:separator], line[separator+3:], true
			}
		} else {
			key, raw, found = strings.Cut(line, " = ")
		}
		if !found || key == "" || raw == "" || strings.ContainsAny(key, " .") {
			return nil, reject("SYNTAX", "invalid configuration line")
		}
		value := raw
		if strings.HasPrefix(raw, `"`) {
			if len(raw) < 2 || !strings.HasSuffix(raw, `"`) ||
				strings.ContainsAny(raw[1:len(raw)-1], `"\\`) {
				return nil, reject("SYNTAX", "invalid quoted value")
			}
			value = raw[1 : len(raw)-1]
		} else if _, parseErr := strconv.Atoi(raw); parseErr != nil {
			return nil, reject("SYNTAX", "value is neither a quoted string nor an integer")
		}
		pairs = append(pairs, keyValue{key: key, value: value})
	}
	return pairs, nil
}

func parseConfig(data []byte, windows bool) (map[string]string, error) {
	pairs, err := configLines(data, windows)
	if err != nil {
		return nil, err
	}
	values, err := validatePairs(pairs, configOrder)
	if err != nil {
		return nil, err
	}
	if values["schema_version"] != "buildopt.config/v1" {
		return nil, reject("INVALID_VALUE", "invalid configuration schema")
	}
	if values["mode"] != "auto" && values["mode"] != "observe" && values["mode"] != "off" {
		return nil, reject("INVALID_VALUE", "invalid mode")
	}
	serverURL := values["server_url"]
	projectScope := values["project_scope"]
	credentialEnv := values["credential_env"]
	serverPresent := serverURL != ""
	if serverPresent != (projectScope != "") || serverPresent != (credentialEnv != "") {
		return nil, reject("INCOMPLETE_SERVER_IDENTITY", "server fields must be all empty or all present")
	}
	if serverPresent {
		parsed, parseErr := url.Parse(serverURL)
		loopbackHTTP := parseErr == nil && parsed.Scheme == "http" &&
			(parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1")
		if parseErr != nil || parsed.Host == "" ||
			(parsed.Scheme != "https" && !loopbackHTTP) ||
			parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, reject("INSECURE_URL", "invalid server URL")
		}
		if !projectPattern.MatchString(projectScope) || !credentialPattern.MatchString(credentialEnv) {
			return nil, reject("INVALID_VALUE", "invalid project scope or credential environment")
		}
	}
	budget, parseErr := strconv.Atoi(values["trial_budget_percent"])
	if parseErr != nil || budget < 0 || budget > 5 {
		return nil, reject("INVALID_VALUE", "trial budget must be between zero and five")
	}
	return values, nil
}

func TestCanonicalFilesArePortable(t *testing.T) {
	root := filepath.Join(repositoryRoot(t), "fixtures", "sticky-wrapper-contract", "valid")
	tests := []struct {
		path       string
		executable bool
		maximum    int
		prefix     string
		properties bool
	}{
		{path: "buildoptw", executable: true, maximum: 32768, prefix: "#!/bin/sh\n"},
		{path: "buildoptw.bat", maximum: 32768, prefix: "@echo off\n"},
		{path: filepath.Join(".buildopt", "wrapper.properties"), maximum: 16384, properties: true},
		{path: filepath.Join(".buildopt", "config.toml"), maximum: 16384},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			path := filepath.Join(root, test.path)
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			hasEveryExecuteBit := info.Mode().Perm()&0o111 == 0o111
			if hasEveryExecuteBit != test.executable || info.Mode().Perm()&0o002 != 0 {
				t.Fatalf("mode = %04o, executable=%t and world-writable=false required", info.Mode().Perm(), test.executable)
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(contents) > test.maximum {
				t.Fatalf("size = %d, maximum %d", len(contents), test.maximum)
			}
			if err := validateText(contents, test.properties); err != nil {
				t.Fatal(err)
			}
			if test.prefix != "" && !strings.HasPrefix(string(contents), test.prefix) {
				t.Fatalf("missing prefix %q", test.prefix)
			}
			lower := strings.ToLower(string(contents))
			for _, forbidden := range []string{"/home/", `c:\\users\\`, "%userprofile%", "${home}", "bearer "} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("machine-specific or secret-looking fixture text: %q", forbidden)
				}
			}
		})
	}
}

func TestPOSIXAndWindowsParsersAgree(t *testing.T) {
	properties := readFixture(t, filepath.Join("valid", ".buildopt", "wrapper.properties"))
	posixProperties, err := parseProperties(properties, false)
	if err != nil {
		t.Fatal(err)
	}
	windowsProperties, err := parseProperties(properties, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(posixProperties, windowsProperties) {
		t.Fatal("POSIX and Windows property parsers disagree")
	}

	config := readFixture(t, filepath.Join("valid", ".buildopt", "config.toml"))
	posixConfig, err := parseConfig(config, false)
	if err != nil {
		t.Fatal(err)
	}
	windowsConfig, err := parseConfig(config, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(posixConfig, windowsConfig) {
		t.Fatal("POSIX and Windows configuration parsers disagree")
	}
}

type invalidCases struct {
	SchemaVersion string        `json:"schemaVersion"`
	Cases         []invalidCase `json:"cases"`
}

type invalidCase struct {
	ID         string `json:"id"`
	File       string `json:"file"`
	Operation  string `json:"operation"`
	Old        string `json:"old"`
	Value      string `json:"value"`
	ErrorClass string `json:"errorClass"`
}

func mutateFixture(t *testing.T, data []byte, fixture invalidCase) []byte {
	t.Helper()
	switch fixture.Operation {
	case "append":
		return append(append([]byte(nil), data...), []byte(fixture.Value)...)
	case "replace-first":
		if !strings.Contains(string(data), fixture.Old) {
			t.Fatalf("case %s old value is absent", fixture.ID)
		}
		return []byte(strings.Replace(string(data), fixture.Old, fixture.Value, 1))
	default:
		t.Fatalf("case %s has unknown operation %q", fixture.ID, fixture.Operation)
		return nil
	}
}

func TestInvalidFixturesRejectOnBothPlatforms(t *testing.T) {
	var fixtures invalidCases
	if err := json.Unmarshal(readFixture(t, "invalid-cases.json"), &fixtures); err != nil {
		t.Fatal(err)
	}
	if fixtures.SchemaVersion != "buildopt.fixtures/sticky-wrapper-contract-invalid/v1" ||
		len(fixtures.Cases) != 13 {
		t.Fatalf("invalid fixture registry is incomplete: %#v", fixtures)
	}
	properties := readFixture(t, filepath.Join("valid", ".buildopt", "wrapper.properties"))
	config := readFixture(t, filepath.Join("valid", ".buildopt", "config.toml"))
	for _, fixture := range fixtures.Cases {
		t.Run(fixture.ID, func(t *testing.T) {
			base := properties
			parse := parseProperties
			if fixture.File == "config.toml" {
				base = config
				parse = parseConfig
			}
			mutated := mutateFixture(t, base, fixture)
			for _, windows := range []bool{false, true} {
				_, err := parse(mutated, windows)
				if class := errorClass(err); class != fixture.ErrorClass {
					t.Fatalf("windows=%t error=%v class=%q, want %q", windows, err, class, fixture.ErrorClass)
				}
			}
		})
	}
}

type route struct {
	kind string
	args []string
}

func routeArguments(args []string) (route, error) {
	if len(args) == 0 {
		return route{kind: "GRADLE", args: []string{}}, nil
	}
	if args[0] == "--gradle" {
		return route{kind: "GRADLE", args: append([]string(nil), args[1:]...)}, nil
	}
	if args[0] != "--buildopt" {
		return route{kind: "GRADLE", args: append([]string(nil), args...)}, nil
	}
	if len(args) < 2 || (args[1] != "status" && args[1] != "explain" && args[1] != "version") {
		return route{}, reject("USAGE", "unknown management command")
	}
	if len(args) > 3 || (len(args) == 3 && args[2] != "--json") {
		return route{}, reject("USAGE", "invalid management arguments")
	}
	return route{kind: "BUILDOPT", args: append([]string(nil), args[1:]...)}, nil
}

func TestArgumentRouting(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		want     route
		wantFail bool
	}{
		{name: "empty Gradle", input: []string{}, want: route{kind: "GRADLE", args: []string{}}},
		{name: "Gradle status task", input: []string{"status"}, want: route{kind: "GRADLE", args: []string{"status"}}},
		{name: "difficult Gradle values", input: []string{"task name", "", `-Pquote="x"`, "*"}, want: route{kind: "GRADLE", args: []string{"task name", "", `-Pquote="x"`, "*"}}},
		{name: "management status", input: []string{"--buildopt", "status"}, want: route{kind: "BUILDOPT", args: []string{"status"}}},
		{name: "management JSON", input: []string{"--buildopt", "explain", "--json"}, want: route{kind: "BUILDOPT", args: []string{"explain", "--json"}}},
		{name: "escaped prefix", input: []string{"--gradle", "--buildopt", "status"}, want: route{kind: "GRADLE", args: []string{"--buildopt", "status"}}},
		{name: "unknown management", input: []string{"--buildopt", "unknown"}, wantFail: true},
		{name: "bad management flag", input: []string{"--buildopt", "status", "--verbose"}, wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := routeArguments(test.input)
			if test.wantFail {
				if errorClass(err) != "USAGE" {
					t.Fatalf("error = %v, want USAGE", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("route = %#v, want %#v", got, test.want)
			}
		})
	}
}

func semanticVersion(version string) ([3]int, error) {
	var result [3]int
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return result, reject("INVALID_VALUE", "version must contain three components")
	}
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return result, reject("INVALID_VALUE", "invalid version component")
		}
		result[index] = value
	}
	return result, nil
}

func updateAllowed(current, next string, allowDowngrade bool) (bool, error) {
	currentVersion, err := semanticVersion(current)
	if err != nil {
		return false, err
	}
	nextVersion, err := semanticVersion(next)
	if err != nil {
		return false, err
	}
	for index := range currentVersion {
		if nextVersion[index] > currentVersion[index] {
			return true, nil
		}
		if nextVersion[index] < currentVersion[index] {
			return allowDowngrade, nil
		}
	}
	return true, nil
}

func TestUpdateAndDowngradePolicy(t *testing.T) {
	for _, test := range []struct {
		current        string
		next           string
		allowDowngrade bool
		want           bool
	}{
		{current: "0.6.1", next: "0.6.1", want: true},
		{current: "0.6.1", next: "0.7.0", want: true},
		{current: "1.0.0", next: "0.9.9", want: false},
		{current: "1.0.0", next: "0.9.9", allowDowngrade: true, want: true},
	} {
		got, err := updateAllowed(test.current, test.next, test.allowDowngrade)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("updateAllowed(%q, %q, %t) = %t, want %t", test.current, test.next, test.allowDowngrade, got, test.want)
		}
	}
}
