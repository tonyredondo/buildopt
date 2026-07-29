package schemavalidator

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestTestOptimizationV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"test-optimization-contracts.v1",
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
			name:             "test-cache-grant",
			schemaFilename:   "test-cache-grant.v1.schema.json",
			schemaID:         TestCacheGrantSchemaID,
			fixtureDirectory: "test-cache-grant",
			expectedValid: map[string]struct{}{
				"revision-grant.json": {},
			},
			expectedDiagnostic: map[string]string{
				"no-capability.json":     "anyOf",
				"unsigned-grant.json":    "signature",
				"wildcard-selector.json": "/allowedTaskTypesOrAdapters/0/id",
			},
		},
		{
			name:             "test-validation-result",
			schemaFilename:   "test-validation-result.v1.schema.json",
			schemaID:         TestValidationResultSchemaID,
			fixtureDirectory: "test-validation-result",
			expectedValid: map[string]struct{}{
				"passed.json": {},
			},
			expectedDiagnostic: map[string]string{
				"passed-with-inconclusive.json": "not",
				"unsigned-result.json":          "signature",
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
			validFixtures := fixtureFiles(
				t,
				filepath.Join(fixtureRoot, test.fixtureDirectory, "valid"),
			)
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
				filepath.Join(fixtureRoot, test.fixtureDirectory, "invalid"),
				test.expectedDiagnostic,
			)
		})
	}
}

func TestTestOptimizationSemanticBindingsV1(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	fixtureRoot := filepath.Join(
		schemaRoot,
		"testdata",
		"test-optimization-contracts.v1",
	)
	grantSchema := compileContractSchema(
		t,
		repositoryRoot,
		"test-cache-grant.v1.schema.json",
		TestCacheGrantSchemaID,
	)
	resultSchema := compileContractSchema(
		t,
		repositoryRoot,
		"test-validation-result.v1.schema.json",
		TestValidationResultSchemaID,
	)
	policy := readJSONObject(
		t,
		filepath.Join(
			schemaRoot,
			"testdata",
			"foundation-contracts.v1",
			"policy",
			"valid",
			"verified-policy.json",
		),
	)
	request := readJSONObject(
		t,
		filepath.Join(
			schemaRoot,
			"testdata",
			"attempt-commit.v1",
			"ci-validation-request",
			"valid",
			"full-relevant-validation.json",
		),
	)
	grant := readJSONObject(
		t,
		filepath.Join(
			fixtureRoot,
			"test-cache-grant",
			"valid",
			"revision-grant.json",
		),
	)
	result := readJSONObject(
		t,
		filepath.Join(
			fixtureRoot,
			"test-validation-result",
			"valid",
			"passed.json",
		),
	)
	if err := grantSchema.Validate(grant); err != nil {
		t.Fatalf("valid grant schema: %v", err)
	}
	if err := resultSchema.Validate(result); err != nil {
		t.Fatalf("valid result schema: %v", err)
	}
	if err := validateGrantAgainstPolicyAndRequest(grant, policy, request); err != nil {
		t.Fatalf("valid grant binding: %v", err)
	}
	if err := validateResultAgainstRequestAndPolicy(result, request, policy); err != nil {
		t.Fatalf("valid result binding: %v", err)
	}

	reversedGrant := readJSONObject(
		t,
		filepath.Join(
			fixtureRoot,
			"test-cache-grant",
			"semantic-invalid",
			"reversed-window.json",
		),
	)
	if err := grantSchema.Validate(reversedGrant); err != nil {
		t.Fatalf("reversed-window fixture must be schema-valid: %v", err)
	}
	err := validateGrantAgainstPolicyAndRequest(reversedGrant, policy, request)
	if err == nil || !strings.Contains(err.Error(), "window") {
		t.Fatalf("reversed grant window diagnostic = %v", err)
	}

	reboundResult := readJSONObject(
		t,
		filepath.Join(
			fixtureRoot,
			"test-validation-result",
			"semantic-invalid",
			"artifact-rebinding.json",
		),
	)
	if err := resultSchema.Validate(reboundResult); err != nil {
		t.Fatalf("artifact-rebinding fixture must be schema-valid: %v", err)
	}
	err = validateResultAgainstRequestAndPolicy(reboundResult, request, policy)
	if err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("artifact rebinding diagnostic = %v", err)
	}
}

func TestTestOptimizationValidatorEntrypoints(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	fixtureRoot := filepath.Join(
		schemaRoot,
		"testdata",
		"test-optimization-contracts.v1",
	)
	tests := []struct {
		name     string
		validate func(string, string) error
		schema   string
		fixture  string
	}{
		{
			name:     "grant",
			validate: ValidateTestCacheGrantV1,
			schema:   "test-cache-grant.v1.schema.json",
			fixture: filepath.Join(
				"test-cache-grant",
				"valid",
				"revision-grant.json",
			),
		},
		{
			name:     "result",
			validate: ValidateTestValidationResultV1,
			schema:   "test-validation-result.v1.schema.json",
			fixture: filepath.Join(
				"test-validation-result",
				"valid",
				"passed.json",
			),
		},
	}
	for _, test := range tests {
		if err := test.validate(
			filepath.Join(schemaRoot, test.schema),
			filepath.Join(fixtureRoot, test.fixture),
		); err != nil {
			t.Fatalf("%s validator entrypoint: %v", test.name, err)
		}
	}
}

func validateGrantAgainstPolicyAndRequest(
	grant map[string]any,
	policy map[string]any,
	request map[string]any,
) error {
	issuedAt, err := parseMapTime(grant, "issuedAt")
	if err != nil {
		return err
	}
	expiresAt, err := parseMapTime(grant, "expiresAt")
	if err != nil {
		return err
	}
	if !issuedAt.Before(expiresAt) {
		return errors.New("grant window is empty or reversed")
	}
	policyGrant := mapObject(policy, "testOptimizationGrant")
	if policyGrant == nil {
		return errors.New("policy has no test grant reference")
	}
	if mapString(grant, "grantDigest") != mapString(policyGrant, "digest") ||
		mapString(grant, "expiresAt") != mapString(policyGrant, "expiresAt") ||
		mapString(grant, "policyDigest") != mapString(policy, "policyDigest") {
		return errors.New("grant does not match policy reference")
	}
	if !reflect.DeepEqual(
		mapObject(grant, "repository"),
		mapObject(request, "repository"),
	) {
		return errors.New("grant repository differs from request")
	}
	scope := mapObject(grant, "revisionOrPolicyRange")
	if scope == nil || mapString(scope, "kind") != "REVISION" {
		return errors.New("fixture grant does not use revision scope")
	}
	if mapString(scope, "revision") != mapString(request, "revision") ||
		mapString(scope, "sourceStateDigest") !=
			mapString(request, "sourceStateDigest") {
		return errors.New("grant revision/source binding differs from request")
	}
	selectors, err := mapObjectArray(grant, "allowedTaskTypesOrAdapters")
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(selectors))
	for _, selector := range selectors {
		key := mapString(selector, "kind") + "/" + mapString(selector, "id")
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate grant selector %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateResultAgainstRequestAndPolicy(
	result map[string]any,
	request map[string]any,
	policy map[string]any,
) error {
	for _, key := range []string{"requestId", "actionId", "revision", "sourceStateDigest"} {
		if mapString(result, key) != mapString(request, key) {
			return fmt.Errorf("result %s differs from request", key)
		}
	}
	if !reflect.DeepEqual(
		mapObject(result, "repository"),
		mapObject(request, "repository"),
	) {
		return errors.New("result repository differs from request")
	}
	if mapString(result, "validationMode") != mapString(request, "requestedMode") {
		return errors.New("result validation mode differs from request")
	}
	if mapString(result, "policyDigest") != mapString(policy, "policyDigest") {
		return errors.New("result policy digest differs from policy")
	}
	completedAt, err := parseMapTime(result, "completedAt")
	if err != nil {
		return err
	}
	expiresAt, err := parseMapTime(result, "expiresAt")
	if err != nil {
		return err
	}
	createdAt, err := parseMapTime(request, "createdAt")
	if err != nil {
		return err
	}
	deadline, err := parseMapTime(request, "deadline")
	if err != nil {
		return err
	}
	if completedAt.Before(createdAt) || deadline.Before(completedAt) ||
		!completedAt.Before(expiresAt) {
		return errors.New("result time window is outside request")
	}
	wantArtifacts, err := artifactKeys(request, "candidateArtifactRefs")
	if err != nil {
		return err
	}
	gotArtifacts, err := artifactKeys(result, "validatedArtifactRefs")
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(gotArtifacts, wantArtifacts) {
		return errors.New("result artifact set differs from candidate artifacts")
	}
	return nil
}

func artifactKeys(object map[string]any, key string) ([]string, error) {
	artifacts, err := mapObjectArray(object, key)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		size, ok := artifact["sizeBytes"].(float64)
		if !ok {
			return nil, fmt.Errorf("%s artifact size is not numeric", key)
		}
		keys = append(
			keys,
			fmt.Sprintf(
				"%s/%s/%.0f/%s",
				mapString(artifact, "artifactId"),
				mapString(artifact, "digest"),
				size,
				mapString(artifact, "mediaType"),
			),
		)
	}
	sort.Strings(keys)
	return keys, nil
}

func mapObject(object map[string]any, key string) map[string]any {
	value, _ := object[key].(map[string]any)
	return value
}

func mapObjectArray(object map[string]any, key string) ([]map[string]any, error) {
	values, ok := object[key].([]any)
	if !ok {
		return nil, fmt.Errorf("%s is not an array", key)
	}
	objects := make([]map[string]any, 0, len(values))
	for index, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s[%d] is not an object", key, index)
		}
		objects = append(objects, item)
	}
	return objects, nil
}

func mapString(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}

func parseMapTime(object map[string]any, key string) (time.Time, error) {
	value := mapString(object, key)
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
