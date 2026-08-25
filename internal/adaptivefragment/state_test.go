package adaptivefragment

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type invalidStateVectors struct {
	Base  string                 `json:"base"`
	Cases []invalidStateMutation `json:"cases"`
}

type invalidStateMutation struct {
	Name       string          `json:"name"`
	Target     string          `json:"target"`
	Value      json.RawMessage `json:"value"`
	Layer      string          `json:"layer"`
	Diagnostic string          `json:"diagnostic"`
}

func TestPersistedStateValidLifecycleVectors(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(stateFixtureRoot(t), "valid", "*.json"))
	if err != nil || len(paths) != 2 {
		t.Fatalf("valid fixture paths = %v, err=%v", paths, err)
	}
	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			bundle := readStateBundle(t, path)
			if err := ValidateStateBundle(bundle); err != nil {
				t.Fatalf("valid state bundle: %v", err)
			}
			canonical, err := MarshalCanonicalDocument(bundle.Fragment)
			if err != nil {
				t.Fatal(err)
			}
			pretty, err := json.MarshalIndent(bundle.Fragment, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			canonicalDigest, err := CanonicalDocumentSHA256(canonical)
			if err != nil {
				t.Fatal(err)
			}
			prettyDigest, err := CanonicalDocumentSHA256(pretty)
			if err != nil || canonicalDigest != prettyDigest {
				t.Fatalf("canonical digest mismatch: %s / %s / %v", canonicalDigest, prettyDigest, err)
			}
		})
	}
}

func TestPersistedStateInvalidVectors(t *testing.T) {
	root := stateFixtureRoot(t)
	vectors := readInvalidStateVectors(t, filepath.Join(root, "invalid", "vectors.json"))
	if vectors.Base != "active-lifecycle.json" || len(vectors.Cases) != 7 {
		t.Fatalf("invalid vector manifest = %+v", vectors)
	}
	basePath := filepath.Join(root, "valid", vectors.Base)
	baseBytes, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatal(err)
	}
	base := readStateBundle(t, basePath)

	for _, vector := range vectors.Cases {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			mutated := mutateStateBundle(t, base, vector)
			switch vector.Layer {
			case "SCHEMA":
				// The isolated Draft 2020-12 checker owns the exact schema
				// diagnostic. Product semantics must still fail closed.
				if err := ValidateStateBundle(mutated); err == nil {
					t.Fatal("schema mutation was accepted by semantic validation")
				}
			case "SEMANTIC":
				err := ValidateStateBundle(mutated)
				if err == nil || !strings.Contains(err.Error(), vector.Diagnostic) {
					t.Fatalf("semantic error = %v, want %q", err, vector.Diagnostic)
				}
			case "DIGEST":
				if err := ValidateStateBundle(mutated); err != nil {
					t.Fatalf("digest mutation should remain structurally valid: %v", err)
				}
				baseDigest, digestErr := CanonicalDocumentSHA256(fragmentJSON(t, baseBytes))
				if digestErr != nil {
					t.Fatal(digestErr)
				}
				mutatedDocument, marshalErr := json.Marshal(mutated.Fragment)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				mutatedDigest, digestErr := CanonicalDocumentSHA256(mutatedDocument)
				if digestErr != nil || baseDigest == mutatedDigest {
					t.Fatalf("canonical tampering was not detected: %s / %s / %v", baseDigest, mutatedDigest, digestErr)
				}
			default:
				t.Fatalf("unknown invalid vector layer %q", vector.Layer)
			}
		})
	}
}

func TestPersistedStateRejectsDuplicateObservationsAndUnsortedLedger(t *testing.T) {
	base := readStateBundle(t, filepath.Join(stateFixtureRoot(t), "valid", "active-lifecycle.json"))

	duplicateObservation := base
	duplicateObservation.Observations = append([]Observation{}, base.Observations...)
	duplicateObservation.Observations[1].ObservationID = duplicateObservation.Observations[0].ObservationID
	if err := ValidateStateBundle(duplicateObservation); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("duplicate observation error = %v", err)
	}

	unsortedLedger := base
	unsortedLedger.Ledger.Entries = append([]LedgerEntry{}, base.Ledger.Entries...)
	other := unsortedLedger.Ledger.Entries[0]
	other.FamilyID = "0000000000000000000000000000000000000000000000000000000000000001"
	other.RevisionID = "0000000000000000000000000000000000000000000000000000000000000002"
	unsortedLedger.Ledger.Entries = append(unsortedLedger.Ledger.Entries, other)
	if err := ValidateStateBundle(unsortedLedger); err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("unsorted ledger error = %v", err)
	}
}

func mutateStateBundle(t *testing.T, base StateBundle, vector invalidStateMutation) StateBundle {
	t.Helper()
	document, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var result StateBundle
	if err := json.Unmarshal(document, &result); err != nil {
		t.Fatal(err)
	}
	switch vector.Target {
	case "fragment.schemaVersion":
		decodeMutation(t, vector.Value, &result.Fragment.SchemaVersion)
	case "fragment.familyId":
		decodeMutation(t, vector.Value, &result.Fragment.FamilyID)
	case "fragment.authority":
		decodeMutation(t, vector.Value, &result.Fragment.Authority)
	case "portfolio.repositoryScopeSha256":
		decodeMutation(t, vector.Value, &result.Portfolio.RepositoryScopeSHA256)
	case "portfolio.fragments[0].fragmentGeneration":
		decodeMutation(t, vector.Value, &result.Portfolio.Fragments[0].FragmentGeneration)
	case "observations[2].toState":
		decodeMutation(t, vector.Value, &result.Observations[2].ToState)
	case "fragment.updatedAt":
		decodeMutation(t, vector.Value, &result.Fragment.UpdatedAt)
		result.Observations[len(result.Observations)-1].ObservedAt = result.Fragment.UpdatedAt
		result.Portfolio.CreatedAt = result.Fragment.UpdatedAt
		result.Ledger.AsOf = result.Fragment.UpdatedAt
		result.Ledger.Entries[0].LastObservedAt = result.Fragment.UpdatedAt
	default:
		t.Fatalf("unknown mutation target %q", vector.Target)
	}
	return result
}

func decodeMutation(t *testing.T, raw json.RawMessage, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatal(err)
	}
}

func readStateBundle(t *testing.T, path string) StateBundle {
	t.Helper()
	document, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	var bundle StateBundle
	if err := decoder.Decode(&bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func readInvalidStateVectors(t *testing.T, path string) invalidStateVectors {
	t.Helper()
	document, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var vectors invalidStateVectors
	if err := json.Unmarshal(document, &vectors); err != nil {
		t.Fatal(err)
	}
	return vectors
}

func fragmentJSON(t *testing.T, bundle []byte) []byte {
	t.Helper()
	var object struct {
		Fragment json.RawMessage `json:"fragment"`
	}
	if err := json.Unmarshal(bundle, &object); err != nil {
		t.Fatal(err)
	}
	return object.Fragment
}

func stateFixtureRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("state fixture source location unavailable")
	}
	return filepath.Join(filepath.Dir(source), "..", "..", "contracts", "jsonschema", "testdata", "adaptive-fragment-state.v1")
}
