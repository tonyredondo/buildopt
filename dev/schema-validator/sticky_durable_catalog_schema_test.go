package schemavalidator

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestStickyDurableCatalogV1Fixtures(t *testing.T) {
	t.Parallel()
	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "sticky-wrapper-durable-catalog.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	fixtureRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata", "sticky-wrapper-durable-catalog.v1")
	valid := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	invalid := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(valid) != 1 || len(invalid) != 1 {
		t.Fatalf("durable catalog fixtures = valid %d invalid %d, want one each", len(valid), len(invalid))
	}
	if err := schema.Validate(readJSON(t, valid[0])); err != nil {
		t.Fatalf("valid durable catalog fixture rejected: %v", err)
	}
	if err := schema.Validate(readJSON(t, invalid[0])); err == nil || !strings.Contains(err.Error(), "/pass") {
		t.Fatalf("invalid durable catalog fixture accepted or failed at wrong path: %v", err)
	}
}

func TestStickyDurableCatalogV1CheckedInResult(t *testing.T) {
	t.Parallel()
	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "sticky-wrapper-durable-catalog.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile durable catalog schema: %v", err)
	}
	resultPath := filepath.Join(repositoryRoot, "benchmarks", "results", "sticky-wrapper-durable-catalog-v1.json")
	if err := schema.Validate(readJSON(t, resultPath)); err != nil {
		t.Fatalf("checked-in durable catalog result rejected: %v", err)
	}
}
