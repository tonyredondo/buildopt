package schemavalidator

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestVerifiedRequestHitSafetyRecordV1(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "verified-request-hit-safety-record.v1.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	if schema.ID != "https://schemas.buildopt.dev/verified-request-hit-safety-record.v1.schema.json" || schema.DraftVersion != 2020 {
		t.Fatalf("unexpected compiled schema identity: %s draft %d", schema.ID, schema.DraftVersion)
	}

	fixturePath := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata", "verified-request-hit-safety-record.v1", "valid", "complete.json")
	valid := readJSON(t, fixturePath)
	if err := schema.Validate(valid); err != nil {
		t.Fatalf("complete fixture must validate: %v", err)
	}

	base, ok := valid.(map[string]any)
	if !ok {
		t.Fatal("complete fixture is not an object")
	}
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "unknown field", mutate: func(value map[string]any) { value["unexpected"] = true }},
		{name: "missing outputs", mutate: func(value map[string]any) { delete(value, "outputs") }},
		{name: "unsafe task", mutate: func(value map[string]any) {
			value["tasks"].([]any)[0].(map[string]any)["sideEffects"] = true
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(base)
			if err != nil {
				t.Fatal(err)
			}
			var candidate map[string]any
			if err := json.Unmarshal(raw, &candidate); err != nil {
				t.Fatal(err)
			}
			test.mutate(candidate)
			if err := schema.Validate(candidate); err == nil {
				t.Fatal("invalid fixture was accepted")
			}
		})
	}
}
