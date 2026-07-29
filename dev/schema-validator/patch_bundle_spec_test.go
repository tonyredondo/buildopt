package schemavalidator

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type patchBundleApplicationSpec struct {
	SchemaVersion         string                `json:"schemaVersion"`
	IdempotencyIdentity   []string              `json:"idempotencyIdentity"`
	Phases                []string              `json:"phases"`
	Cases                 []patchBundleSpecCase `json:"cases"`
	ForbiddenCapabilities []string              `json:"forbiddenCapabilities"`
}

type patchBundleSpecCase struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Expected string `json:"expected"`
}

func TestPatchBundleApplicationSpecV1(t *testing.T) {
	t.Parallel()

	spec := loadPatchBundleApplicationSpec(t)
	if spec.SchemaVersion !=
		"buildopt.specs/patch-bundle-application/v1" {
		t.Errorf("schemaVersion = %q", spec.SchemaVersion)
	}
	wantIdentity := []string{"repository", "actionId", "bundleDigest"}
	if !slices.Equal(spec.IdempotencyIdentity, wantIdentity) {
		t.Errorf("idempotency identity = %v", spec.IdempotencyIdentity)
	}
	wantPhases := []string{
		"STRICT_PARSE",
		"VERIFY_SIGNATURE_AND_EXPIRATION",
		"VERIFY_SOURCE_AND_BLOBS",
		"VALIDATE_PATH_GRAPH",
		"CREATE_DETACHED_STAGING_WORKTREE",
		"VERIFY_EXACT_PREIMAGES",
		"APPLY_EXACT_REPLACEMENTS",
		"VERIFY_POSTIMAGES",
		"RUN_ISOLATED_VALIDATION",
		"CREATE_OR_RECOVER_EPHEMERAL_BRANCH",
		"CREATE_OR_RECOVER_DRAFT_PR",
	}
	if !slices.Equal(spec.Phases, wantPhases) {
		t.Errorf("phases = %v, want %v", spec.Phases, wantPhases)
	}
	requiredCases := map[string]string{
		"apply-archive-reproducibility-recipe":                   "APPLIED_STAGED",
		"apply-custom-task-contract-recipe":                      "APPLIED_STAGED",
		"exact-repeat-is-idempotent":                             "EXISTING_DRAFT_PR",
		"divergent-preimage-rejected":                            "PROPOSED",
		"symlink-target-rejected":                                "REJECTED",
		"symlink-parent-rejected":                                "REJECTED",
		"submodule-or-nested-repository-rejected":                "REJECTED",
		"branch-without-pr-is-recovered":                         "DRAFT_PR_CREATED",
		"existing-branch-with-different-marker-conflicts":        "PROPOSED",
		"interrupted-staging-leaves-customer-checkout-unchanged": "UNCHANGED",
	}
	if len(spec.Cases) != 15 {
		t.Fatalf("case count = %d, want 15", len(spec.Cases))
	}
	seen := make(map[string]struct{}, len(spec.Cases))
	for _, testCase := range spec.Cases {
		if testCase.ID == "" || testCase.Category == "" ||
			testCase.Expected == "" {
			t.Errorf("incomplete case: %+v", testCase)
		}
		if _, duplicate := seen[testCase.ID]; duplicate {
			t.Errorf("duplicate case %q", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
		if expected, required := requiredCases[testCase.ID]; required && expected != testCase.Expected {
			t.Errorf(
				"%s expected = %s, want %s",
				testCase.ID,
				testCase.Expected,
				expected,
			)
		}
	}
	for required := range requiredCases {
		if _, exists := seen[required]; !exists {
			t.Errorf("missing required case %q", required)
		}
	}
	requiredForbidden := []string{
		"EXECUTE_BUNDLE_CONTENT",
		"FUZZY_PATCH",
		"FOLLOW_SYMLINK",
		"ENTER_SUBMODULE",
		"AUTO_REBASE",
		"FORCE_PUSH",
		"WRITE_DEFAULT_BRANCH",
		"AUTO_MERGE",
	}
	for _, capability := range requiredForbidden {
		if !slices.Contains(spec.ForbiddenCapabilities, capability) {
			t.Errorf("forbidden capability %q is absent", capability)
		}
	}
}

func loadPatchBundleApplicationSpec(
	t *testing.T,
) patchBundleApplicationSpec {
	t.Helper()
	path := filepath.Join(
		findRepositoryRoot(t),
		"specs",
		"patch-bundle-v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var spec patchBundleApplicationSpec
	if err := decoder.Decode(&spec); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing data: %v", path, err)
	}
	return spec
}
