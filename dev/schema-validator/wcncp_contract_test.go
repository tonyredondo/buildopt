package schemavalidator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestWCNCPV1SchemasAndVectors(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	vectorPath := filepath.Join(root, "contracts", "test-vectors", "wcncp", "wcncp-control-state.v1.json")
	content, err := os.ReadFile(vectorPath)
	if err != nil {
		t.Fatalf("read WCNCP vectors: %v", err)
	}
	var catalog struct {
		SchemaVersion  string                     `json:"schemaVersion"`
		Records        map[string]json.RawMessage `json:"records"`
		InvalidRecords map[string]json.RawMessage `json:"invalidRecords"`
		Authority      struct {
			SyntheticOnly        bool `json:"syntheticOnly"`
			ProspectiveEvidence  bool `json:"prospectiveEvidence"`
			ProductionAuthorized bool `json:"productionAuthorized"`
		} `json:"authority"`
	}
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatalf("decode WCNCP vectors: %v", err)
	}
	if catalog.SchemaVersion != "buildopt.test-vectors/wcncp-control-state/v1" ||
		!catalog.Authority.SyntheticOnly || catalog.Authority.ProspectiveEvidence ||
		catalog.Authority.ProductionAuthorized {
		t.Fatalf("WCNCP vector authority is inconsistent")
	}

	schemas := map[string]string{
		"observation": "wcncp-observation.v1.schema.json",
		"opportunity": "wcncp-opportunity.v1.schema.json",
		"proposal":    "wcncp-proposal.v1.schema.json",
		"validation":  "wcncp-validation.v1.schema.json",
		"decision":    "wcncp-decision.v1.schema.json",
	}
	if len(catalog.Records) != len(schemas) || len(catalog.InvalidRecords) != len(schemas) {
		t.Fatalf("WCNCP vectors must contain one valid and invalid record for each schema")
	}
	for kind, filename := range schemas {
		kind, filename := kind, filename
		t.Run(kind, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			schema, err := compiler.Compile(filepath.Join(root, "contracts", "jsonschema", filename))
			if err != nil {
				t.Fatalf("compile %s: %v", filename, err)
			}
			var valid any
			if err := json.Unmarshal(catalog.Records[kind], &valid); err != nil {
				t.Fatalf("decode valid %s: %v", kind, err)
			}
			if err := schema.Validate(valid); err != nil {
				t.Fatalf("valid %s rejected: %v", kind, err)
			}
			var invalid any
			if err := json.Unmarshal(catalog.InvalidRecords[kind], &invalid); err != nil {
				t.Fatalf("decode invalid %s: %v", kind, err)
			}
			if err := schema.Validate(invalid); err == nil {
				t.Fatalf("invalid %s accepted", kind)
			}
		})
	}
}
