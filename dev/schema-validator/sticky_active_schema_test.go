package schemavalidator

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestStickyActiveV1Fixtures(t *testing.T) {
	t.Parallel()
	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "sticky-wrapper-active.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	fixtureRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata", "sticky-wrapper-active.v1")
	valid := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	invalid := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(valid) != 1 || len(invalid) != 1 {
		t.Fatalf("sticky active fixtures = valid %d invalid %d, want one each", len(valid), len(invalid))
	}
	if err := schema.Validate(readJSON(t, valid[0])); err != nil {
		t.Fatalf("valid sticky active fixture rejected: %v", err)
	}
	if err := schema.Validate(readJSON(t, invalid[0])); err == nil || !strings.Contains(err.Error(), "/pass") {
		t.Fatalf("invalid sticky active fixture accepted or failed at wrong path: %v", err)
	}
}

func TestStickyActiveV1CheckedInResult(t *testing.T) {
	t.Parallel()
	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "sticky-wrapper-active.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	resultPath := filepath.Join(repositoryRoot, "benchmarks", "results", "sticky-wrapper-active-v1.json")
	if err := schema.Validate(readJSON(t, resultPath)); err != nil {
		t.Fatalf("checked-in active result rejected: %v", err)
	}
}
