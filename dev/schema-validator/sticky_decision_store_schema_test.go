package schemavalidator

import (
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestStickyWrapperDecisionStoreV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "sticky-wrapper-decision-store.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	if schema.ID != "https://schemas.buildopt.dev/sticky-wrapper-decision-store.v1.schema.json" {
		t.Fatalf("compiled schema ID = %q", schema.ID)
	}
	if schema.DraftVersion != 2020 {
		t.Fatalf("compiled schema draft = %d, want 2020-12", schema.DraftVersion)
	}

	fixtureRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata", "sticky-wrapper-decision-store.v1")
	valid := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	invalid := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(valid) != 7 || len(invalid) != 6 {
		t.Fatalf("fixture count = %d valid/%d invalid, want 7/6", len(valid), len(invalid))
	}
	for _, path := range valid {
		path := path
		t.Run("valid/"+filepath.Base(path), func(t *testing.T) {
			if err := schema.Validate(readJSON(t, path)); err != nil {
				t.Fatalf("%s must validate: %v", relativePath(repositoryRoot, path), err)
			}
		})
	}
	for _, path := range invalid {
		path := path
		t.Run("invalid/"+filepath.Base(path), func(t *testing.T) {
			if err := schema.Validate(readJSON(t, path)); err == nil {
				t.Fatalf("%s must be rejected", relativePath(repositoryRoot, path))
			}
		})
	}
}
