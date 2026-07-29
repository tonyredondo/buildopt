package schemavalidator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestFoundationContractV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	fixtureRoot := filepath.Join(
		schemaRoot,
		"testdata",
		"foundation-contracts.v1",
	)

	tests := []struct {
		name               string
		schemaFilename     string
		schemaID           string
		fixtureDirectory   string
		expectedValid      map[string]struct{}
		expectedDiagnostic map[string]string
	}{
		{
			name:             "evidence",
			schemaFilename:   "evidence-record.v1.schema.json",
			schemaID:         EvidenceRecordSchemaID,
			fixtureDirectory: "evidence",
			expectedValid: map[string]struct{}{
				"qualified-adapter.json": {},
			},
			expectedDiagnostic: map[string]string{
				"incomplete-trace-qualified.json": "/qualificationState",
			},
		},
		{
			name:             "policy",
			schemaFilename:   "optimization-policy.v1.schema.json",
			schemaID:         OptimizationPolicySchemaID,
			fixtureDirectory: "policy",
			expectedValid: map[string]struct{}{
				"verified-policy.json": {},
			},
			expectedDiagnostic: map[string]string{
				"active-kill-switch.json": "/allowedActions",
			},
		},
		{
			name:             "resource-profile",
			schemaFilename:   "resource-profile.v1.schema.json",
			schemaID:         ResourceProfileSchemaID,
			fixtureDirectory: "resource-profile",
			expectedValid: map[string]struct{}{
				"stable-control.json": {},
				"w2-h3g.json":         {},
				"w3-h4g.json":         {},
				"w4-h6g.json":         {},
			},
			expectedDiagnostic: map[string]string{
				"eligible-with-failed-memory.json": "/eligibility/memory",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			schema := compileContractSchema(
				t,
				repositoryRoot,
				test.schemaFilename,
				test.schemaID,
			)
			validDirectory := filepath.Join(
				fixtureRoot,
				test.fixtureDirectory,
				"valid",
			)
			validFixtures := fixtureFiles(t, validDirectory)
			if len(validFixtures) != len(test.expectedValid) {
				t.Fatalf(
					"found %d valid %s fixtures, want %d",
					len(validFixtures),
					test.name,
					len(test.expectedValid),
				)
			}
			for _, fixturePath := range validFixtures {
				if _, ok := test.expectedValid[filepath.Base(fixturePath)]; !ok {
					t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
				}
				if err := schema.Validate(readJSON(t, fixturePath)); err != nil {
					t.Fatalf(
						"%s must validate: %v",
						relativePath(repositoryRoot, fixturePath),
						err,
					)
				}
			}
			assertInvalidFixtures(
				t,
				repositoryRoot,
				schema,
				filepath.Join(
					fixtureRoot,
					test.fixtureDirectory,
					"invalid",
				),
				test.expectedDiagnostic,
			)
		})
	}
}

func TestFoundationGoldenRecordsV1(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"foundation-contracts.v1",
	)
	evidence := readJSONObject(
		t,
		filepath.Join(fixtureRoot, "evidence", "valid", "qualified-adapter.json"),
	)
	policy := readJSONObject(
		t,
		filepath.Join(fixtureRoot, "policy", "valid", "verified-policy.json"),
	)

	assertSameString(t, evidence, policy, "policyDigest", "policyDigest")
	assertNestedSameString(
		t,
		evidence,
		policy,
		"cacheNamespace",
		[]string{"remoteCache", "namespace"},
	)

	qualifiedTasks := objectArray(t, policy, "qualifiedTasks")
	if len(qualifiedTasks) != 1 {
		t.Fatalf("qualifiedTasks count = %d, want 1", len(qualifiedTasks))
	}
	qualifiedTask := qualifiedTasks[0]
	assertSameString(
		t,
		evidence,
		qualifiedTask,
		"taskImplementationHash",
		"implementationHash",
	)
	assertSameString(
		t,
		evidence,
		qualifiedTask,
		"cacheContractDigest",
		"cacheContractDigest",
	)
	assertSameString(
		t,
		evidence,
		qualifiedTask,
		"qualificationSource",
		"qualificationSource",
	)
	assertSameString(
		t,
		evidence,
		qualifiedTask,
		"qualificationState",
		"qualificationState",
	)

	issuedAt := parseObjectTime(t, policy, "issuedAt")
	recordedAt := parseObjectTime(t, evidence, "recordedAt")
	expiresAt := parseObjectTime(t, policy, "expiresAt")
	if recordedAt.Before(issuedAt) || !recordedAt.Before(expiresAt) {
		t.Fatalf(
			"evidence time %s must be within policy interval [%s, %s)",
			recordedAt,
			issuedAt,
			expiresAt,
		)
	}

	profileDirectory := filepath.Join(
		fixtureRoot,
		"resource-profile",
		"valid",
	)
	profilePaths := fixtureFiles(t, profileDirectory)
	if len(profilePaths) != 4 {
		t.Fatalf("golden catalog has %d profiles, want 4", len(profilePaths))
	}
	profiles := make([]map[string]any, 0, len(profilePaths))
	profileIDs := make([]string, 0, len(profilePaths))
	for _, profilePath := range profilePaths {
		profile := readJSONObject(t, profilePath)
		profiles = append(profiles, profile)
		profileIDs = append(profileIDs, objectString(t, profile, "profileId"))
		assertResourceHeadroom(t, profile)
	}
	sort.Strings(profileIDs)
	wantProfileIDs := []string{
		"STABLE_CONTROL",
		"W2_H3G",
		"W3_H4G",
		"W4_H6G",
	}
	sort.Strings(wantProfileIDs)
	if !reflect.DeepEqual(profileIDs, wantProfileIDs) {
		t.Fatalf("golden profile IDs = %v, want %v", profileIDs, wantProfileIDs)
	}

	wantInvariant := profileInvariant(t, profiles[0])
	for _, profile := range profiles[1:] {
		if got := profileInvariant(t, profile); !reflect.DeepEqual(got, wantInvariant) {
			t.Fatalf(
				"profile %s varies outside maxWorkers and gradleHeapMb",
				objectString(t, profile, "profileId"),
			)
		}
	}

	selected := objectValue(t, policy, "resourceProfile")
	selectedID := objectString(t, selected, "profileId")
	var selectedProfile map[string]any
	for _, profile := range profiles {
		if objectString(t, profile, "profileId") == selectedID {
			selectedProfile = profile
			break
		}
	}
	if selectedProfile == nil {
		t.Fatalf("policy selects unknown profile %q", selectedID)
	}
	assertSameString(t, selected, selectedProfile, "profileDigest", "profileDigest")
	assertSameString(t, selected, selectedProfile, "catalogVersion", "catalogVersion")
}

func TestFoundationContractValidatorEntrypoints(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	fixtureRoot := filepath.Join(
		schemaRoot,
		"testdata",
		"foundation-contracts.v1",
	)

	tests := []struct {
		name     string
		validate func(string, string) error
		schema   string
		instance string
	}{
		{
			name:     "evidence",
			validate: ValidateEvidenceRecordV1,
			schema:   "evidence-record.v1.schema.json",
			instance: filepath.Join("evidence", "valid", "qualified-adapter.json"),
		},
		{
			name:     "policy",
			validate: ValidateOptimizationPolicyV1,
			schema:   "optimization-policy.v1.schema.json",
			instance: filepath.Join("policy", "valid", "verified-policy.json"),
		},
		{
			name:     "resource-profile",
			validate: ValidateResourceProfileV1,
			schema:   "resource-profile.v1.schema.json",
			instance: filepath.Join("resource-profile", "valid", "w3-h4g.json"),
		},
	}
	for _, test := range tests {
		if err := test.validate(
			filepath.Join(schemaRoot, test.schema),
			filepath.Join(fixtureRoot, test.instance),
		); err != nil {
			t.Fatalf("%s validator entrypoint: %v", test.name, err)
		}
	}
}

func readJSONObject(t *testing.T, path string) map[string]any {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}()
	var object map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&object); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return object
}

func objectValue(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := object[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object", key)
	}
	return value
}

func objectArray(t *testing.T, object map[string]any, key string) []map[string]any {
	t.Helper()

	values, ok := object[key].([]any)
	if !ok {
		t.Fatalf("%s is not an array", key)
	}
	objects := make([]map[string]any, 0, len(values))
	for index, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("%s[%d] is not an object", key, index)
		}
		objects = append(objects, item)
	}
	return objects
}

func objectString(t *testing.T, object map[string]any, key string) string {
	t.Helper()

	value, ok := object[key].(string)
	if !ok {
		t.Fatalf("%s is not a string", key)
	}
	return value
}

func objectNumber(t *testing.T, object map[string]any, key string) float64 {
	t.Helper()

	value, ok := object[key].(float64)
	if !ok {
		t.Fatalf("%s is not a number", key)
	}
	return value
}

func parseObjectTime(t *testing.T, object map[string]any, key string) time.Time {
	t.Helper()

	value, err := time.Parse(time.RFC3339, objectString(t, object, key))
	if err != nil {
		t.Fatalf("parse %s: %v", key, err)
	}
	return value
}

func assertSameString(
	t *testing.T,
	left map[string]any,
	right map[string]any,
	leftKey string,
	rightKey string,
) {
	t.Helper()

	leftValue := objectString(t, left, leftKey)
	rightValue := objectString(t, right, rightKey)
	if leftValue != rightValue {
		t.Fatalf("%s = %q, want %s %q", leftKey, leftValue, rightKey, rightValue)
	}
}

func assertNestedSameString(
	t *testing.T,
	left map[string]any,
	right map[string]any,
	leftKey string,
	rightPath []string,
) {
	t.Helper()

	nested := right
	for _, key := range rightPath[:len(rightPath)-1] {
		nested = objectValue(t, nested, key)
	}
	assertSameString(t, left, nested, leftKey, rightPath[len(rightPath)-1])
}

func assertResourceHeadroom(t *testing.T, profile map[string]any) {
	t.Helper()

	cgroup := objectValue(t, profile, "cgroupLimits")
	memoryBytes := objectNumber(t, cgroup, "memoryBytes")
	headroomBytes := objectNumber(t, cgroup, "minHeadroomBytes")
	heapBytes := objectNumber(t, profile, "gradleHeapMb") * 1024 * 1024
	if heapBytes+headroomBytes > memoryBytes {
		t.Fatalf(
			"profile %s exceeds cgroup memory after required headroom",
			objectString(t, profile, "profileId"),
		)
	}
}

func profileInvariant(t *testing.T, profile map[string]any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("copy profile: %v", err)
	}
	var invariant map[string]any
	if err := json.Unmarshal(encoded, &invariant); err != nil {
		t.Fatalf("decode profile copy: %v", err)
	}
	for _, key := range []string{
		"profileId",
		"profileDigest",
		"profileClass",
		"maxWorkers",
		"gradleHeapMb",
	} {
		delete(invariant, key)
	}
	eligibility := objectValue(t, invariant, "eligibility")
	delete(eligibility, "evidenceRefs")
	return invariant
}
