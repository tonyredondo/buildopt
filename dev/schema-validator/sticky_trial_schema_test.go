package schemavalidator

import (
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestStickyWrapperTrialV1Fixtures(t *testing.T) {
	t.Parallel()
	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "sticky-wrapper-trial.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	if schema.ID != "https://schemas.buildopt.dev/sticky-wrapper-trial.v1.schema.json" || schema.DraftVersion != 2020 {
		t.Fatalf("compiled trial schema identity = %q draft %d", schema.ID, schema.DraftVersion)
	}
	fixtureRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata", "sticky-wrapper-trial.v1")
	valid := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	invalid := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(valid) != 1 || len(invalid) != 1 {
		t.Fatalf("trial fixture count = %d valid/%d invalid, want 1/1", len(valid), len(invalid))
	}
	if err := schema.Validate(readJSON(t, valid[0])); err != nil {
		t.Fatalf("valid trial fixture rejected: %v", err)
	}
	if err := schema.Validate(readJSON(t, invalid[0])); err == nil {
		t.Fatal("invalid trial fixture accepted")
	}
}
