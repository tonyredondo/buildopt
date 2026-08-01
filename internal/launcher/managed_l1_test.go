//go:build linux || darwin

package launcher

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

func TestManagedL1ConfigurationAndScopeBinding(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	environment := map[string]string{
		managedL1StateRootEnvironment:     stateRoot,
		managedL1TenantEnvironment:        "tenant-7",
		managedL1RepositoryEnvironment:    "tonyredondo/buildopt",
		managedL1TrustDomainEnvironment:   "private-beta",
		managedL1CompatibilityEnvironment: "gradle-9.6-java-17-linux-amd64",
		managedL1GenerationEnvironment:    "42",
		managedL1L2WriterEnvironment:      "0",
	}
	config, configured, err := managedL1ConfigFromEnvironment(
		func(key string) string {
			return environment[key]
		},
	)
	if err != nil {
		t.Fatalf("parse managed L1 configuration: %v", err)
	}
	if !configured ||
		config.stateRoot != stateRoot ||
		config.securityGeneration != 42 ||
		config.l2WriteAuthorized ||
		len(config.scopeDigest) != 64 {
		t.Fatalf("unexpected managed L1 configuration: %+v", config)
	}
	if _, err := hex.DecodeString(config.scopeDigest); err != nil {
		t.Fatalf("scope digest is not lowercase hexadecimal: %q", config.scopeDigest)
	}

	values := []string{
		config.tenantID,
		config.repositoryID,
		config.trustDomain,
		config.compatibilityClass,
	}
	for index := range values {
		changed := append([]string(nil), values...)
		changed[index] += "-changed"
		actual := managedL1ScopeDigest(
			changed[0],
			changed[1],
			changed[2],
			changed[3],
		)
		if actual == config.scopeDigest {
			t.Fatalf("scope dimension %d did not change the digest", index)
		}
	}

	empty, configured, err := managedL1ConfigFromEnvironment(
		func(string) string { return "" },
	)
	if err != nil || configured || empty != (managedL1Config{}) {
		t.Fatalf("empty configuration = %+v/%t/%v", empty, configured, err)
	}
}

func TestManagedL1RejectsIncompleteOrUnsafeConfiguration(t *testing.T) {
	valid := map[string]string{
		managedL1StateRootEnvironment:     filepath.Join(t.TempDir(), "state"),
		managedL1TenantEnvironment:        "tenant-7",
		managedL1RepositoryEnvironment:    "tonyredondo/buildopt",
		managedL1TrustDomainEnvironment:   "private-beta",
		managedL1CompatibilityEnvironment: "gradle-9.6-java-17-linux-amd64",
		managedL1GenerationEnvironment:    "42",
	}
	testCases := []struct {
		name  string
		key   string
		value string
	}{
		{
			name: "missing repository",
			key:  managedL1RepositoryEnvironment,
		},
		{
			name:  "relative state root",
			key:   managedL1StateRootEnvironment,
			value: "relative/state",
		},
		{
			name:  "identity with whitespace",
			key:   managedL1TenantEnvironment,
			value: "tenant 7",
		},
		{
			name:  "noncanonical generation",
			key:   managedL1GenerationEnvironment,
			value: "042",
		},
		{
			name:  "negative generation",
			key:   managedL1GenerationEnvironment,
			value: "-1",
		},
		{
			name:  "invalid writer flag",
			key:   managedL1L2WriterEnvironment,
			value: "true",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			environment := make(map[string]string, len(valid)+1)
			for key, value := range valid {
				environment[key] = value
			}
			environment[testCase.key] = testCase.value
			_, configured, err := managedL1ConfigFromEnvironment(
				func(key string) string {
					return environment[key]
				},
			)
			if !configured || err == nil {
				t.Fatalf("unsafe configuration was accepted: %v", err)
			}
		})
	}
}

func TestManagedL1PrivateDirectoryGenerationAndLease(t *testing.T) {
	config := managedL1TestConfig(filepath.Join(t.TempDir(), "state"))
	first, err := startManagedL1(config)
	if err != nil {
		t.Fatalf("start first managed L1: %v", err)
	}
	defer func() {
		if err := first.close(); err != nil {
			t.Errorf("close first managed L1: %v", err)
		}
	}()
	if first.mode != managedL1ReadWriteMode ||
		!strings.Contains(first.directory, config.scopeDigest) ||
		!strings.HasSuffix(
			first.directory,
			filepath.Join("generation-42", "cache"),
		) {
		t.Fatalf("unexpected managed L1: %+v", first)
	}
	for path := first.directory; path != config.stateRoot; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("managed L1 directory %s = %v/%v", path, info, err)
		}
	}

	second, err := startManagedL1(config)
	if second != nil || !errors.Is(err, errManagedL1Busy) {
		if second != nil {
			_ = second.close()
		}
		t.Fatalf("concurrent managed L1 = %+v/%v, want busy", second, err)
	}

	rotatedConfig := config
	rotatedConfig.securityGeneration++
	rotated, err := startManagedL1(rotatedConfig)
	if err != nil {
		t.Fatalf("start rotated generation: %v", err)
	}
	if rotated.directory == first.directory ||
		!strings.HasSuffix(
			rotated.directory,
			filepath.Join("generation-43", "cache"),
		) {
		t.Fatalf("generation did not rotate the directory: %+v", rotated)
	}
	if err := rotated.close(); err != nil {
		t.Fatalf("close rotated generation: %v", err)
	}

	if err := first.close(); err != nil {
		t.Fatalf("release first managed L1: %v", err)
	}
	reopened, err := startManagedL1(config)
	if err != nil {
		t.Fatalf("reopen released managed L1: %v", err)
	}
	if err := reopened.close(); err != nil {
		t.Fatalf("close reopened managed L1: %v", err)
	}
}

func TestManagedL1DisablesLocalCacheForL2Writer(t *testing.T) {
	config := managedL1TestConfig(filepath.Join(t.TempDir(), "unused-state"))
	config.l2WriteAuthorized = true

	l1, err := startManagedL1(config)
	if err != nil {
		t.Fatalf("start L2-writer managed L1 context: %v", err)
	}
	if l1.mode != managedL1DisabledWriterMode ||
		l1.directory != "" ||
		l1.lease != nil {
		t.Fatalf("L2 writer retained a local cache: %+v", l1)
	}
	if _, err := os.Stat(config.stateRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("L2 writer created managed L1 state: %v", err)
	}
	environment := l1.childEnvironment()
	if environment[managedL1ModeChildEnvironment] != managedL1DisabledWriterMode ||
		environment[managedL1GenerationChildEnvironment] != "42" ||
		environment[managedL1RetentionChildEnvironment] != "7" {
		t.Fatalf("unexpected L2 writer child context: %+v", environment)
	}
	if _, present := environment[managedL1DirectoryChildEnvironment]; present {
		t.Fatal("L2 writer child context exposed a local directory")
	}
}

func TestManagedL1RejectsGenerationPredatingManagedDeletion(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deployment-data")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create managed root: %v", err)
	}
	marker, err := json.Marshal(map[string]string{
		"deploymentRoot": filepath.Join(filepath.Dir(root), "deployment"),
		"schemaVersion":  "buildopt.dev/deployment-data/v1",
	})
	if err != nil {
		t.Fatalf("encode managed root marker: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".buildopt-deployment-data.json"),
		append(marker, '\n'),
		0o600,
	); err != nil {
		t.Fatalf("write managed root marker: %v", err)
	}
	key := make([]byte, datalifecycle.RedactionKeyBytes)
	for index := range key {
		key[index] = byte(index + 1)
	}
	if _, err := datalifecycle.DeleteManagedData(
		context.Background(),
		datalifecycle.DeletionRequest{
			DataRoot:                 root,
			DeletionID:               "l1-generation-floor",
			Tenant:                   "tenant-7",
			Repository:               "tonyredondo/buildopt",
			TrustDomain:              "private-beta",
			NextNamespaceGeneration:  8,
			NextL1SecurityGeneration: 43,
			TokenKey:                 key,
			TokenKeyVersion:          "l1-floor-v1",
			RequestedAt: time.Date(
				2026,
				time.July,
				31,
				12,
				0,
				0,
				0,
				time.UTC,
			),
		},
	); err != nil {
		t.Fatalf("establish managed deletion boundary: %v", err)
	}
	config := managedL1TestConfig(root)
	if _, err := startManagedL1(config); err == nil ||
		!strings.Contains(err.Error(), "predates managed deletion") {
		t.Fatalf("stale managed L1 error = %v", err)
	}
	config.securityGeneration = 43
	current, err := startManagedL1(config)
	if err != nil {
		t.Fatalf("start rotated managed L1: %v", err)
	}
	if err := current.close(); err != nil {
		t.Fatalf("close rotated managed L1: %v", err)
	}
}

func TestManagedL1RequiresPrivateStateRoot(t *testing.T) {
	config := managedL1TestConfig(filepath.Join(t.TempDir(), "state"))
	if err := os.Mkdir(config.stateRoot, 0o755); err != nil {
		t.Fatalf("create public state root: %v", err)
	}
	if l1, err := startManagedL1(config); err == nil ||
		!strings.Contains(err.Error(), "mode 0700") {
		if l1 != nil {
			_ = l1.close()
		}
		t.Fatalf("public state root was accepted: %+v/%v", l1, err)
	}
}

func managedL1TestConfig(stateRoot string) managedL1Config {
	tenantID := "tenant-7"
	repositoryID := "tonyredondo/buildopt"
	trustDomain := "private-beta"
	compatibilityClass := "gradle-9.6-java-17-linux-amd64"
	return managedL1Config{
		stateRoot:          stateRoot,
		tenantID:           tenantID,
		repositoryID:       repositoryID,
		trustDomain:        trustDomain,
		compatibilityClass: compatibilityClass,
		securityGeneration: 42,
		scopeDigest: managedL1ScopeDigest(
			tenantID,
			repositoryID,
			trustDomain,
			compatibilityClass,
		),
	}
}
