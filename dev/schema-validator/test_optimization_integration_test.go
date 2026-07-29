package schemavalidator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type testOptimizationIntegrationCatalog struct {
	SchemaVersion string `json:"schemaVersion"`
	ContractMajor int    `json:"contractMajor"`
	CurrentMinor  int    `json:"currentMinor"`
	Artifacts     struct {
		ValidPath   string `json:"validPath"`
		ValidSize   int    `json:"validSize"`
		ValidSHA256 string `json:"validSha256"`
		CorruptPath string `json:"corruptPath"`
	} `json:"artifacts"`
	Cases []testOptimizationIntegrationCase `json:"cases"`
}

type testOptimizationIntegrationCase struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	RemoteMajor      int    `json:"remoteMajor"`
	RemoteMinor      int    `json:"remoteMinor"`
	Authority        string `json:"authority"`
	Transport        string `json:"transport"`
	ExpectedProducer string `json:"expectedProducer"`
	ExpectedConsumer string `json:"expectedConsumer"`
}

type testOptimizationIntegrationResult struct {
	Producer string
	Consumer string
}

func TestTestOptimizationIntegrationV1(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	catalog := loadTestOptimizationIntegrationCatalog(t, root)
	if catalog.SchemaVersion !=
		"buildopt.specs/test-optimization-integration/v1" ||
		catalog.ContractMajor != 1 ||
		catalog.CurrentMinor != 1 {
		t.Errorf("invalid integration identity: %+v", catalog)
	}
	assertTestOptimizationArtifacts(t, root, catalog)
	if len(catalog.Cases) != 16 {
		t.Fatalf("case count = %d, want 16", len(catalog.Cases))
	}
	seen := make(map[string]struct{}, len(catalog.Cases))
	for _, testCase := range catalog.Cases {
		if testCase.ID == "" {
			t.Fatal("empty case ID")
		}
		if _, duplicate := seen[testCase.ID]; duplicate {
			t.Fatalf("duplicate case ID %q", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
		actual := evaluateTestOptimizationIntegration(catalog, testCase)
		expected := testOptimizationIntegrationResult{
			Producer: testCase.ExpectedProducer,
			Consumer: testCase.ExpectedConsumer,
		}
		if actual != expected {
			t.Errorf("%s result = %+v, want %+v", testCase.ID, actual, expected)
		}
	}
}

func loadTestOptimizationIntegrationCatalog(
	t *testing.T,
	root string,
) testOptimizationIntegrationCatalog {
	t.Helper()
	path := filepath.Join(
		root,
		"specs",
		"test-optimization-integration-v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var catalog testOptimizationIntegrationCatalog
	if err := decoder.Decode(&catalog); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing data: %v", path, err)
	}
	return catalog
}

func assertTestOptimizationArtifacts(
	t *testing.T,
	root string,
	catalog testOptimizationIntegrationCatalog,
) {
	t.Helper()
	valid, err := os.ReadFile(filepath.Join(root, catalog.Artifacts.ValidPath))
	if err != nil {
		t.Fatalf("read valid artifact: %v", err)
	}
	sum := sha256.Sum256(valid)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if len(valid) != catalog.Artifacts.ValidSize ||
		digest != catalog.Artifacts.ValidSHA256 {
		t.Errorf(
			"valid artifact = size %d digest %s",
			len(valid),
			digest,
		)
	}
	corrupt, err := os.ReadFile(filepath.Join(root, catalog.Artifacts.CorruptPath))
	if err != nil {
		t.Fatalf("read corrupt artifact: %v", err)
	}
	corruptSum := sha256.Sum256(corrupt)
	if corruptSum == sum {
		t.Error("corrupt artifact has the valid digest")
	}
}

func evaluateTestOptimizationIntegration(
	catalog testOptimizationIntegrationCatalog,
	testCase testOptimizationIntegrationCase,
) testOptimizationIntegrationResult {
	if testCase.RemoteMajor != catalog.ContractMajor ||
		testCase.RemoteMinor < catalog.CurrentMinor-1 ||
		testCase.RemoteMinor > catalog.CurrentMinor+1 {
		return testOptimizationIntegrationResult{
			Producer: "CONTRACT_INCOMPATIBLE",
			Consumer: "INCONCLUSIVE",
		}
	}
	switch testCase.Kind {
	case "GRANT":
		switch testCase.Authority {
		case "ACTIVE_TRUSTED":
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_GRANT",
				Consumer: "ENABLE_TEST_CACHE",
			}
		case "MISSING":
			return testOptimizationIntegrationResult{
				Producer: "GRANT_UNAVAILABLE",
				Consumer: "DISABLE_TEST_CACHE",
			}
		case "EXPIRED":
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_EXPIRED_GRANT",
				Consumer: "DISABLE_TEST_CACHE",
			}
		default:
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_GRANT",
				Consumer: "DISABLE_TEST_CACHE",
			}
		}
	case "GRANT_STATUS":
		if testCase.Transport == "TIMEOUT" {
			return testOptimizationIntegrationResult{
				Producer: "DEADLINE_EXCEEDED",
				Consumer: "ABORT_PENDING",
			}
		}
		if testCase.Authority == "REVOKED" {
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_REVOKED_STATUS",
				Consumer: "ABORT_PENDING",
			}
		}
		if testCase.Authority == "DIGEST_MISMATCH" {
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_ACTIVE_STATUS",
				Consumer: "ABORT_PENDING",
			}
		}
	case "VALIDATION":
		switch testCase.Transport {
		case "CHANGED_RETRY":
			return testOptimizationIntegrationResult{
				Producer: "IDEMPOTENCY_CONFLICT",
				Consumer: "BLOCK_ACTION",
			}
		case "TIMEOUT":
			return testOptimizationIntegrationResult{
				Producer: "DEADLINE_EXCEEDED",
				Consumer: "INCONCLUSIVE",
			}
		case "DELAYED_POLL":
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_RESULT_AFTER_POLL",
				Consumer: validationConsumerOutcome(testCase.Authority),
			}
		case "TRANSIENT_EXACT_RETRY":
			return testOptimizationIntegrationResult{
				Producer: "SIGNED_RESULT_REPLAY",
				Consumer: validationConsumerOutcome(testCase.Authority),
			}
		}
		if testCase.Authority == "CORRUPT_ARTIFACT" ||
			testCase.Authority == "ARBITRARY_PATH" {
			return testOptimizationIntegrationResult{
				Producer: "ARTIFACT_REJECTED",
				Consumer: "BLOCK_ACTION",
			}
		}
		return testOptimizationIntegrationResult{
			Producer: "SIGNED_RESULT",
			Consumer: validationConsumerOutcome(testCase.Authority),
		}
	}
	return testOptimizationIntegrationResult{
		Producer: "INVALID_FIXTURE",
		Consumer: "INCONCLUSIVE",
	}
}

func validationConsumerOutcome(authority string) string {
	switch authority {
	case "PASSED_TRUSTED":
		return "ALLOW_ACTION"
	case "INCONCLUSIVE_TRUSTED":
		return "INCONCLUSIVE"
	default:
		return "BLOCK_ACTION"
	}
}
