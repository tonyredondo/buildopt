package schemavalidator

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type adaptiveRawBundle struct {
	Fragment     json.RawMessage   `json:"fragment"`
	Observations []json.RawMessage `json:"observations"`
	Portfolio    json.RawMessage   `json:"portfolio"`
	Ledger       json.RawMessage   `json:"ledger"`
}

type adaptiveInvalidVectors struct {
	Base  string `json:"base"`
	Cases []struct {
		Name       string          `json:"name"`
		Target     string          `json:"target"`
		Value      json.RawMessage `json:"value"`
		Layer      string          `json:"layer"`
		Diagnostic string          `json:"diagnostic"`
	} `json:"cases"`
}

func TestAdaptiveFragmentStateSchemas(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemas := map[string]*jsonschema.Schema{
		"fragment":    compileContractSchema(t, repositoryRoot, "adaptive-fragment.v1.schema.json", AdaptiveFragmentSchemaID),
		"observation": compileContractSchema(t, repositoryRoot, "adaptive-fragment-observation.v1.schema.json", AdaptiveFragmentObservationSchemaID),
		"portfolio":   compileContractSchema(t, repositoryRoot, "adaptive-fragment-portfolio.v1.schema.json", AdaptiveFragmentPortfolioSchemaID),
		"ledger":      compileContractSchema(t, repositoryRoot, "adaptive-fragment-economic-ledger.v1.schema.json", AdaptiveFragmentEconomicLedgerSchemaID),
	}
	fixtureRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata", "adaptive-fragment-state.v1")
	validPaths := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	if len(validPaths) != 2 {
		t.Fatalf("valid adaptive fragment bundles = %d, want 2", len(validPaths))
	}
	for _, path := range validPaths {
		path := path
		t.Run("valid/"+filepath.Base(path), func(t *testing.T) {
			validateAdaptiveRawBundle(t, schemas, readAdaptiveRawBundle(t, path))
		})
	}

	vectorsPath := filepath.Join(fixtureRoot, "invalid", "vectors.json")
	var vectors adaptiveInvalidVectors
	readAdaptiveJSON(t, vectorsPath, &vectors)
	if vectors.Base != "active-lifecycle.json" || len(vectors.Cases) != 7 {
		t.Fatalf("adaptive invalid vectors = %+v", vectors)
	}
	base := readAdaptiveRawBundle(t, filepath.Join(fixtureRoot, "valid", vectors.Base))
	for _, vector := range vectors.Cases {
		vector := vector
		t.Run("invalid/"+vector.Name, func(t *testing.T) {
			mutated := mutateAdaptiveRawBundle(t, base, vector.Target, vector.Value)
			if vector.Layer == "SCHEMA" {
				err := schemas["fragment"].Validate(decodeAdaptiveRaw(t, mutated.Fragment))
				if err == nil || !strings.Contains(err.Error(), vector.Diagnostic) {
					t.Fatalf("schema error = %v, want %q", err, vector.Diagnostic)
				}
				return
			}
			// Cross-record and digest vectors must remain structurally valid so
			// the semantic/digest layer, rather than an accidental schema rule,
			// owns their rejection.
			validateAdaptiveRawBundle(t, schemas, mutated)
		})
	}
}

func validateAdaptiveRawBundle(t *testing.T, schemas map[string]*jsonschema.Schema, bundle adaptiveRawBundle) {
	t.Helper()
	if err := schemas["fragment"].Validate(decodeAdaptiveRaw(t, bundle.Fragment)); err != nil {
		t.Fatalf("fragment schema: %v", err)
	}
	if len(bundle.Observations) == 0 {
		t.Fatal("observation fixture is empty")
	}
	for index, observation := range bundle.Observations {
		if err := schemas["observation"].Validate(decodeAdaptiveRaw(t, observation)); err != nil {
			t.Fatalf("observation %d schema: %v", index, err)
		}
	}
	if err := schemas["portfolio"].Validate(decodeAdaptiveRaw(t, bundle.Portfolio)); err != nil {
		t.Fatalf("portfolio schema: %v", err)
	}
	if err := schemas["ledger"].Validate(decodeAdaptiveRaw(t, bundle.Ledger)); err != nil {
		t.Fatalf("ledger schema: %v", err)
	}
}

func mutateAdaptiveRawBundle(t *testing.T, base adaptiveRawBundle, target string, raw json.RawMessage) adaptiveRawBundle {
	t.Helper()
	document, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(document, &object); err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	fragment := object["fragment"].(map[string]any)
	portfolio := object["portfolio"].(map[string]any)
	observations := object["observations"].([]any)
	switch target {
	case "fragment.schemaVersion":
		fragment["schemaVersion"] = value
	case "fragment.familyId":
		fragment["familyId"] = value
	case "fragment.authority":
		fragment["authority"] = value
	case "portfolio.repositoryScopeSha256":
		portfolio["repositoryScopeSha256"] = value
	case "portfolio.fragments[0].fragmentGeneration":
		portfolio["fragments"].([]any)[0].(map[string]any)["fragmentGeneration"] = value
	case "observations[2].toState":
		observations[2].(map[string]any)["toState"] = value
	case "fragment.updatedAt":
		fragment["updatedAt"] = value
	default:
		t.Fatalf("unknown adaptive mutation target %q", target)
	}
	mutated, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	var result adaptiveRawBundle
	if err := json.Unmarshal(mutated, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func readAdaptiveRawBundle(t *testing.T, path string) adaptiveRawBundle {
	t.Helper()
	var bundle adaptiveRawBundle
	readAdaptiveJSON(t, path, &bundle)
	return bundle
}

func readAdaptiveJSON(t *testing.T, path string, target any) {
	t.Helper()
	document, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func decodeAdaptiveRaw(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return value
}
