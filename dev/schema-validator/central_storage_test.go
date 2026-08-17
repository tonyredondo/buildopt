package schemavalidator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type centralStorageCatalog struct {
	SchemaVersion         string                     `json:"schemaVersion"`
	RepositoryScopeSHA256 string                     `json:"repositoryScopeSha256"`
	Manifests             map[string]json.RawMessage `json:"manifests"`
	InvalidManifests      map[string]json.RawMessage `json:"invalidManifests"`
	Scenarios             []centralStorageScenario   `json:"scenarios"`
	Authority             centralStorageAuthority    `json:"authority"`
}

type centralStorageAuthority struct {
	ProofOfConcept        bool   `json:"proofOfConcept"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	SoakRequired          bool   `json:"soakRequired"`
	DesignPartnerRequired bool   `json:"designPartnerRequired"`
	TestOptimization      string `json:"testOptimization"`
}

type centralStorageScenario struct {
	ID            string               `json:"id"`
	Steps         []centralStorageStep `json:"steps"`
	ExpectedHeads map[string]int       `json:"expectedHeads"`
}

type centralStorageStep struct {
	Action                  string `json:"action"`
	Kind                    string `json:"kind"`
	Digest                  string `json:"digest"`
	ContentSHA256           string `json:"contentSha256"`
	Manifest                string `json:"manifest"`
	CommandID               string `json:"commandId"`
	ExpectedGeneration      int    `json:"expectedGeneration"`
	LocalSnapshotCompatible bool   `json:"localSnapshotCompatible"`
	Expected                string `json:"expected"`
}

type centralStateManifest struct {
	Kind                  string                  `json:"kind"`
	RepositoryScopeSHA256 string                  `json:"repositoryScopeSha256"`
	Generation            int                     `json:"generation"`
	CompatibilitySHA256   string                  `json:"compatibilitySha256"`
	Artifacts             []centralStateArtifact  `json:"artifacts"`
	References            []centralStateReference `json:"references"`
	CreatedAt             string                  `json:"createdAt"`
}

type centralStateArtifact struct {
	SHA256 string `json:"sha256"`
}

type centralStateReference struct {
	Kind           string `json:"kind"`
	ManifestSHA256 string `json:"manifestSha256"`
}

type centralStoredManifest struct {
	Digest   string
	Document centralStateManifest
}

type centralStoredHead struct {
	Digest         string
	ManifestDigest string
	Generation     int
}

type centralCommand struct {
	Fingerprint string
}

type centralStorageMachine struct {
	repositoryScope string
	manifestSchema  *jsonschema.Schema
	headSchema      *jsonschema.Schema
	casSchema       *jsonschema.Schema
	manifestRaw     map[string]json.RawMessage
	invalidRaw      map[string]json.RawMessage
	manifestDigest  map[string]string
	cache           map[string]bool
	objects         map[string]map[string]bool
	manifests       map[string]map[string]centralStoredManifest
	heads           map[string]centralStoredHead
	commands        map[string]centralCommand
}

func TestCentralStorageV1Schemas(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	catalog := readCentralStorageCatalog(t, root)
	manifestSchema := compileCentralSchema(
		t,
		filepath.Join(root, "contracts", "jsonschema", "central-state-manifest.v1.schema.json"),
		CentralStateManifestSchemaID,
	)
	compileCentralSchema(
		t,
		filepath.Join(root, "contracts", "jsonschema", "central-state-head.v1.schema.json"),
		CentralStateHeadSchemaID,
	)
	compileCentralSchema(
		t,
		filepath.Join(root, "contracts", "jsonschema", "central-state-cas.v1.schema.json"),
		CentralStateCASSchemaID,
	)

	if len(catalog.Manifests) != 6 {
		t.Fatalf("valid manifest count = %d, want 6", len(catalog.Manifests))
	}
	for name, raw := range catalog.Manifests {
		name, raw := name, raw
		t.Run("valid/"+name, func(t *testing.T) {
			if err := manifestSchema.Validate(decodeCentralJSON(t, raw)); err != nil {
				t.Fatalf("manifest %q must validate: %v", name, err)
			}
		})
	}

	if len(catalog.InvalidManifests) != 2 {
		t.Fatalf("invalid manifest count = %d, want 2", len(catalog.InvalidManifests))
	}
	for name, raw := range catalog.InvalidManifests {
		name, raw := name, raw
		t.Run("invalid/"+name, func(t *testing.T) {
			if err := manifestSchema.Validate(decodeCentralJSON(t, raw)); err == nil {
				t.Fatalf("manifest %q must be rejected", name)
			}
		})
	}
}

func TestCentralStorageV1Contract(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	path := filepath.Join(root, "specs", "poc-central-storage-contract-v1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var contract struct {
		SchemaVersion string `json:"schemaVersion"`
		WorkItem      string `json:"workItem"`
		Planes        map[string]struct {
			OptimizationAuthority *bool `json:"optimizationAuthority"`
			SelectionRevalidation *bool `json:"selectionAlwaysRequiresLocalRevalidation"`
		} `json:"planes"`
		StateRoutes []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"stateRoutes"`
		Capabilities []string                `json:"capabilities"`
		Authority    centralStorageAuthority `json:"authority"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("decode contract: %v", err)
	}
	if contract.SchemaVersion != "buildopt.poc/central-storage-contract/v1" ||
		contract.WorkItem != "POC-CENTRAL-STORAGE-CONTRACT-001" {
		t.Fatalf("unexpected contract identity: %q %q", contract.SchemaVersion, contract.WorkItem)
	}
	if len(contract.Planes) != 2 || contract.Planes["gradleCache"].OptimizationAuthority == nil ||
		*contract.Planes["gradleCache"].OptimizationAuthority {
		t.Fatal("Gradle cache plane must be present and must not grant optimization authority")
	}
	statePlane, ok := contract.Planes["buildoptState"]
	if !ok || statePlane.SelectionRevalidation == nil || !*statePlane.SelectionRevalidation {
		t.Fatal("BuildOpt state plane must require local revalidation")
	}
	if len(contract.StateRoutes) != 6 {
		t.Fatalf("state route count = %d, want 6", len(contract.StateRoutes))
	}
	wantCapabilities := []string{"CACHE_READ", "CACHE_WRITE", "STATE_READ", "STATE_WRITE"}
	if fmt.Sprint(contract.Capabilities) != fmt.Sprint(wantCapabilities) {
		t.Fatalf("capabilities = %v, want %v", contract.Capabilities, wantCapabilities)
	}
	if !contract.Authority.ProofOfConcept || contract.Authority.ProductionAuthorized ||
		contract.Authority.SoakRequired || contract.Authority.DesignPartnerRequired ||
		contract.Authority.TestOptimization != "OUT_OF_SCOPE" {
		t.Fatalf("contract authority exceeds the POC boundary: %+v", contract.Authority)
	}
}

func TestCentralStorageV1Scenarios(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	catalog := readCentralStorageCatalog(t, root)
	if catalog.SchemaVersion != "buildopt.contracts/central-storage/v1" {
		t.Fatalf("catalog schemaVersion = %q", catalog.SchemaVersion)
	}
	if !catalog.Authority.ProofOfConcept || catalog.Authority.ProductionAuthorized ||
		catalog.Authority.SoakRequired || catalog.Authority.DesignPartnerRequired ||
		catalog.Authority.TestOptimization != "OUT_OF_SCOPE" {
		t.Fatalf("catalog authority exceeds the POC boundary: %+v", catalog.Authority)
	}
	if len(catalog.Scenarios) != 6 {
		t.Fatalf("scenario count = %d, want 6", len(catalog.Scenarios))
	}

	manifestSchema := compileCentralSchema(
		t,
		filepath.Join(root, "contracts", "jsonschema", "central-state-manifest.v1.schema.json"),
		CentralStateManifestSchemaID,
	)
	headSchema := compileCentralSchema(
		t,
		filepath.Join(root, "contracts", "jsonschema", "central-state-head.v1.schema.json"),
		CentralStateHeadSchemaID,
	)
	casSchema := compileCentralSchema(
		t,
		filepath.Join(root, "contracts", "jsonschema", "central-state-cas.v1.schema.json"),
		CentralStateCASSchemaID,
	)

	seen := make(map[string]bool)
	for _, scenario := range catalog.Scenarios {
		scenario := scenario
		t.Run(scenario.ID, func(t *testing.T) {
			if scenario.ID == "" || seen[scenario.ID] {
				t.Fatalf("scenario ID must be non-empty and unique: %q", scenario.ID)
			}
			seen[scenario.ID] = true
			machine := newCentralStorageMachine(t, catalog, manifestSchema, headSchema, casSchema)
			for index, step := range scenario.Steps {
				actual := machine.apply(t, step)
				if actual != step.Expected {
					t.Fatalf("step %d %s = %s, want %s", index+1, step.Action, actual, step.Expected)
				}
			}
			machine.assertHeads(t, scenario.ExpectedHeads)
		})
	}
}

func newCentralStorageMachine(
	t *testing.T,
	catalog centralStorageCatalog,
	manifestSchema, headSchema, casSchema *jsonschema.Schema,
) *centralStorageMachine {
	t.Helper()
	digests := make(map[string]string, len(catalog.Manifests))
	for name, raw := range catalog.Manifests {
		digests[name] = canonicalCentralDigest(t, raw)
	}
	return &centralStorageMachine{
		repositoryScope: catalog.RepositoryScopeSHA256,
		manifestSchema:  manifestSchema,
		headSchema:      headSchema,
		casSchema:       casSchema,
		manifestRaw:     catalog.Manifests,
		invalidRaw:      catalog.InvalidManifests,
		manifestDigest:  digests,
		cache:           make(map[string]bool),
		objects:         make(map[string]map[string]bool),
		manifests:       make(map[string]map[string]centralStoredManifest),
		heads:           make(map[string]centralStoredHead),
		commands:        make(map[string]centralCommand),
	}
}

func (machine *centralStorageMachine) apply(t *testing.T, step centralStorageStep) string {
	t.Helper()
	switch step.Action {
	case "PUT_CACHE":
		if step.Digest != step.ContentSHA256 {
			return "DIGEST_MISMATCH"
		}
		machine.cache[step.Digest] = true
		return "APPLIED"
	case "GET_CACHE":
		if machine.cache[step.Digest] {
			return "HIT"
		}
		return "MISS"
	case "EVICT_CACHE":
		delete(machine.cache, step.Digest)
		return "APPLIED"
	case "SELECT_CACHE":
		return "NATIVE_RETAINED"
	case "CACHE_OUTAGE":
		return "MISS"
	case "PUT_OBJECT":
		if step.Digest != step.ContentSHA256 {
			return "DIGEST_MISMATCH"
		}
		machine.kindObjects(step.Kind)[step.Digest] = true
		return "APPLIED"
	case "GET_OBJECT":
		if machine.kindObjects(step.Kind)[step.Digest] {
			return "FOUND"
		}
		return "NOT_FOUND"
	case "PUT_INVALID_MANIFEST":
		raw, ok := machine.invalidRaw[step.Manifest]
		if !ok {
			t.Fatalf("unknown invalid manifest %q", step.Manifest)
		}
		if err := machine.manifestSchema.Validate(decodeCentralJSON(t, raw)); err == nil {
			return "APPLIED"
		}
		return "INVALID_MANIFEST"
	case "PUT_MANIFEST":
		return machine.putManifest(t, step.Manifest)
	case "CAS_HEAD":
		return machine.casHead(t, step)
	case "GET_HEAD":
		if _, ok := machine.heads[step.Kind]; ok {
			return "FOUND"
		}
		return "NOT_FOUND"
	case "SELECT_HEAD":
		if _, ok := machine.heads[step.Kind]; !ok {
			return "NATIVE_RETAINED"
		}
		if step.Kind == "PORTFOLIO" {
			return "LOCAL_REVALIDATION_REQUIRED"
		}
		return "NATIVE_RETAINED"
	case "EXPIRE":
		return machine.expire(step.Kind)
	case "STATE_OUTAGE":
		if step.LocalSnapshotCompatible {
			return "VERIFIED_LOCAL_SNAPSHOT"
		}
		return "NATIVE_RETAINED"
	default:
		t.Fatalf("unsupported action %q", step.Action)
		return ""
	}
}

func (machine *centralStorageMachine) putManifest(t *testing.T, name string) string {
	t.Helper()
	raw, ok := machine.manifestRaw[name]
	if !ok {
		t.Fatalf("unknown manifest %q", name)
	}
	if err := machine.manifestSchema.Validate(decodeCentralJSON(t, raw)); err != nil {
		t.Fatalf("valid manifest %q rejected: %v", name, err)
	}
	var manifest centralStateManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode manifest %q: %v", name, err)
	}
	if manifest.RepositoryScopeSHA256 != machine.repositoryScope {
		return "WRONG_NAMESPACE"
	}
	for _, artifact := range manifest.Artifacts {
		if !machine.kindObjects(manifest.Kind)[artifact.SHA256] {
			return "MANIFEST_INCOMPLETE"
		}
	}
	for _, reference := range manifest.References {
		if _, ok := machine.kindManifests(reference.Kind)[reference.ManifestSHA256]; !ok {
			return "MANIFEST_INCOMPLETE"
		}
	}
	digest := machine.manifestDigest[name]
	machine.kindManifests(manifest.Kind)[digest] = centralStoredManifest{
		Digest: digest, Document: manifest,
	}
	return "APPLIED"
}

func (machine *centralStorageMachine) casHead(t *testing.T, step centralStorageStep) string {
	t.Helper()
	raw, ok := machine.manifestRaw[step.Manifest]
	if !ok {
		t.Fatalf("unknown manifest %q", step.Manifest)
	}
	var manifest centralStateManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode manifest %q: %v", step.Manifest, err)
	}
	manifestDigest := machine.manifestDigest[step.Manifest]
	fingerprint := fmt.Sprintf("%s:%d:%s", manifest.Kind, step.ExpectedGeneration, manifestDigest)
	if prior, ok := machine.commands[step.CommandID]; ok {
		if prior.Fingerprint == fingerprint {
			return "REPLAYED"
		}
		return "IDEMPOTENCY_CONFLICT"
	}
	if _, ok := machine.kindManifests(manifest.Kind)[manifestDigest]; !ok {
		return "MANIFEST_INCOMPLETE"
	}
	current, exists := machine.heads[manifest.Kind]
	currentGeneration := 0
	if exists {
		currentGeneration = current.Generation
	}
	if currentGeneration != step.ExpectedGeneration {
		return "STATE_PRECONDITION_FAILED"
	}
	if manifest.Generation != step.ExpectedGeneration+1 {
		return "GENERATION_CONFLICT"
	}

	next := map[string]any{
		"schemaVersion":         "buildopt.central/state-head/v1",
		"recordType":            "CENTRAL_STATE_HEAD",
		"kind":                  manifest.Kind,
		"repositoryScopeSha256": machine.repositoryScope,
		"generation":            manifest.Generation,
		"manifestSha256":        manifestDigest,
		"compatibilitySha256":   manifest.CompatibilitySHA256,
		"updatedAt":             manifest.CreatedAt,
		"authority": map[string]any{
			"selectionRequiresLocalRevalidation": true,
			"productionAuthorized":               false,
			"testOptimization":                   "OUT_OF_SCOPE",
		},
	}
	var expectedHead any
	if exists {
		next["previousManifestSha256"] = current.ManifestDigest
		expectedHead = current.Digest
	}
	if err := machine.headSchema.Validate(next); err != nil {
		t.Fatalf("generated head failed schema: %v", err)
	}
	request := map[string]any{
		"schemaVersion":      "buildopt.central/state-cas/v1",
		"recordType":         "CENTRAL_STATE_CAS",
		"operation":          "CREATE_OR_ADVANCE",
		"idempotencyKey":     step.CommandID,
		"expectedGeneration": step.ExpectedGeneration,
		"expectedHeadSha256": expectedHead,
		"next":               next,
	}
	if err := machine.casSchema.Validate(request); err != nil {
		t.Fatalf("generated CAS request failed schema: %v", err)
	}
	headDigest := canonicalCentralDigest(t, mustMarshalCentral(t, next))
	machine.heads[manifest.Kind] = centralStoredHead{
		Digest: headDigest, ManifestDigest: manifestDigest, Generation: manifest.Generation,
	}
	machine.commands[step.CommandID] = centralCommand{Fingerprint: fingerprint}
	return "APPLIED"
}

func (machine *centralStorageMachine) expire(kind string) string {
	head, ok := machine.heads[kind]
	if !ok {
		return "NOT_FOUND"
	}
	if kind == "EVIDENCE" {
		if portfolio, exists := machine.heads["PORTFOLIO"]; exists {
			manifest := machine.manifests["PORTFOLIO"][portfolio.ManifestDigest].Document
			for _, reference := range manifest.References {
				if reference.Kind == "EVIDENCE" && reference.ManifestSHA256 == head.ManifestDigest {
					return "RETENTION_BLOCKED"
				}
			}
		}
	}
	delete(machine.heads, kind)
	return "EXPIRED"
}

func (machine *centralStorageMachine) kindObjects(kind string) map[string]bool {
	if machine.objects[kind] == nil {
		machine.objects[kind] = make(map[string]bool)
	}
	return machine.objects[kind]
}

func (machine *centralStorageMachine) kindManifests(kind string) map[string]centralStoredManifest {
	if machine.manifests[kind] == nil {
		machine.manifests[kind] = make(map[string]centralStoredManifest)
	}
	return machine.manifests[kind]
}

func (machine *centralStorageMachine) assertHeads(t *testing.T, expected map[string]int) {
	t.Helper()
	actual := make(map[string]int, len(machine.heads))
	for kind, head := range machine.heads {
		actual[kind] = head.Generation
	}
	if fmt.Sprint(sortedCentralHeads(actual)) != fmt.Sprint(sortedCentralHeads(expected)) {
		t.Fatalf("final heads = %v, want %v", actual, expected)
	}
}

func sortedCentralHeads(heads map[string]int) []string {
	result := make([]string, 0, len(heads))
	for kind, generation := range heads {
		result = append(result, fmt.Sprintf("%s=%d", kind, generation))
	}
	sort.Strings(result)
	return result
}

func readCentralStorageCatalog(t *testing.T, root string) centralStorageCatalog {
	t.Helper()
	path := filepath.Join(root, "contracts", "test-vectors", "central-storage", "central-storage.v1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read central-storage vectors: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var catalog centralStorageCatalog
	if err := decoder.Decode(&catalog); err != nil {
		t.Fatalf("decode central-storage vectors: %v", err)
	}
	return catalog
}

func compileCentralSchema(t *testing.T, path, expectedID string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if expectedID == CentralStateCASSchemaID {
		headPath := filepath.Join(filepath.Dir(path), "central-state-head.v1.schema.json")
		headFile, err := os.Open(headPath)
		if err != nil {
			t.Fatalf("open referenced head schema: %v", err)
		}
		headDocument, decodeErr := jsonschema.UnmarshalJSON(headFile)
		closeErr := headFile.Close()
		if decodeErr != nil {
			t.Fatalf("decode referenced head schema: %v", decodeErr)
		}
		if closeErr != nil {
			t.Fatalf("close referenced head schema: %v", closeErr)
		}
		if err := compiler.AddResource(CentralStateHeadSchemaID, headDocument); err != nil {
			t.Fatalf("register referenced head schema: %v", err)
		}
	}
	schema, err := compiler.Compile(path)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(findRepositoryRoot(t), path), err)
	}
	if schema.ID != expectedID || schema.DraftVersion != 2020 {
		t.Fatalf("schema identity = %q draft %d, want %q draft 2020", schema.ID, schema.DraftVersion, expectedID)
	}
	return schema
}

func decodeCentralJSON(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode JSON instance: %v", err)
	}
	return instance
}

func canonicalCentralDigest(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode JSON for canonical digest: %v", err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode canonical JSON: %v", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}

func mustMarshalCentral(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal generated document: %v", err)
	}
	return data
}
