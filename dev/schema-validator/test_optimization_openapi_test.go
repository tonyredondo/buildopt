package schemavalidator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testOptimizationContractVersion = "test-optimization/v1"

func TestTestOptimizationOpenAPIV1Document(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	path := filepath.Join(
		repositoryRoot,
		"contracts",
		"openapi",
		"test-optimization.v1.yaml",
	)
	document := loadOpenAPIDocument(t, path)
	assertOpenAPIDocumentPolicy(
		t,
		document,
		testOptimizationContractVersion,
		4,
	)

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, expectedRef := range []string{
		"../jsonschema/test-cache-grant.v1.schema.json",
		"../jsonschema/test-validation-result.v1.schema.json",
	} {
		if !bytes.Contains(source, []byte("$ref: '"+expectedRef+"'")) {
			t.Errorf(
				"test-optimization.v1.yaml does not bind %s",
				expectedRef,
			)
		}
	}
}

func TestTestOptimizationOpenAPIV1MockServer(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	document := loadOpenAPIDocument(
		t,
		filepath.Join(
			repositoryRoot,
			"contracts",
			"openapi",
			"test-optimization.v1.yaml",
		),
	)
	mock := newOpenAPIContractMock(
		t,
		document,
		testOptimizationMockReplies(t, repositoryRoot),
	)
	server := httptest.NewServer(mock)
	t.Cleanup(server.Close)

	grantRequest := openAPIMustJSON(t, map[string]any{
		"contractVersion": testOptimizationContractVersion,
		"context":         openAPIRequestContext("request-grant-01"),
		"policyDigest":    openAPIDigest("5"),
		"requestedSelectors": []any{
			map[string]any{
				"kind": "TASK_TYPE",
				"id":   "org.gradle.api.tasks.testing.Test",
			},
			map[string]any{
				"kind": "ADAPTER",
				"id":   "jvm-test-v1",
			},
		},
		"requestedCapabilities": []string{"READ", "WRITE"},
	})
	grantHeaders := openAPIPostHeaders(
		testOptimizationContractVersion,
		"request-grant-01",
		"",
	)
	grant := sendOpenAPIRequest(
		t,
		server.URL+"/v1/test-cache-grants:resolve",
		http.MethodPost,
		grantRequest,
		grantHeaders,
	)
	assertOpenAPIResponseStatus(t, grant, http.StatusOK)
	replayedGrant := sendOpenAPIRequest(
		t,
		server.URL+"/v1/test-cache-grants:resolve",
		http.MethodPost,
		grantRequest,
		grantHeaders,
	)
	assertSameOpenAPIResponse(t, grant, replayedGrant)

	conflictingGrantRequest := openAPIMustJSON(t, map[string]any{
		"contractVersion": testOptimizationContractVersion,
		"context":         openAPIRequestContext("request-grant-01"),
		"policyDigest":    openAPIDigest("5"),
		"requestedSelectors": []any{
			map[string]any{
				"kind": "ADAPTER",
				"id":   "jvm-test-v1",
			},
		},
		"requestedCapabilities": []string{"READ"},
	})
	conflict := sendOpenAPIRequest(
		t,
		server.URL+"/v1/test-cache-grants:resolve",
		http.MethodPost,
		conflictingGrantRequest,
		grantHeaders,
	)
	assertProblemResponse(
		t,
		conflict,
		http.StatusConflict,
		"IDEMPOTENCY_CONFLICT",
	)

	grantStatus := sendOpenAPIRequest(
		t,
		server.URL+"/v1/test-cache-grants/test-grant-01K4Y800/status",
		http.MethodGet,
		nil,
		openAPIReadHeaders(
			testOptimizationContractVersion,
			"request-grant-status-01",
		),
	)
	assertOpenAPIResponseStatus(t, grantStatus, http.StatusOK)

	validationRequest := openAPIMustJSON(t, map[string]any{
		"contractVersion": testOptimizationContractVersion,
		"context":         openAPIRequestContext("request-01K4Y800"),
		"actionId":        "action-01K4Y800",
		"validationMode":  "FULL_RELEVANT_VALIDATION",
		"policyDigest":    openAPIDigest("5"),
		"candidateArtifacts": []any{
			map[string]any{
				"artifactId": "candidate-classes",
				"digest":     openAPIDigest("2"),
				"sizeBytes":  2048,
				"mediaType":  "application/zip",
				"retrieval": map[string]any{
					"kind":    "CUSTOMER_CHANNEL",
					"locator": "ci-artifact:candidate-classes",
				},
			},
		},
	})
	submission := sendOpenAPIRequest(
		t,
		server.URL+"/v1/build-validations",
		http.MethodPost,
		validationRequest,
		openAPIPostHeaders(
			testOptimizationContractVersion,
			"request-01K4Y800/action-01K4Y800",
			"",
		),
	)
	assertOpenAPIResponseStatus(t, submission, http.StatusAccepted)

	result := sendOpenAPIRequest(
		t,
		server.URL+"/v1/build-validations/validation-operation-01K4Y800",
		http.MethodGet,
		nil,
		openAPIReadHeaders(
			testOptimizationContractVersion,
			"request-validation-status-01",
		),
	)
	assertOpenAPIResponseStatus(t, result, http.StatusOK)
	mock.assertAllOperationsExercised(t)
}

func testOptimizationMockReplies(
	t *testing.T,
	repositoryRoot string,
) map[string]openAPIMockReply {
	t.Helper()

	grant := readOpenAPIBytes(
		t,
		filepath.Join(
			repositoryRoot,
			"contracts",
			"jsonschema",
			"testdata",
			"test-optimization-contracts.v1",
			"test-cache-grant",
			"valid",
			"revision-grant.json",
		),
	)
	result := readOpenAPIBytes(
		t,
		filepath.Join(
			repositoryRoot,
			"contracts",
			"jsonschema",
			"testdata",
			"test-optimization-contracts.v1",
			"test-validation-result",
			"valid",
			"passed.json",
		),
	)
	return map[string]openAPIMockReply{
		"resolveTestCacheGrant": openAPIJSONReply(
			http.StatusOK,
			"request-grant-01",
			grant,
			nil,
		),
		"getTestCacheGrantStatus": openAPIJSONReply(
			http.StatusOK,
			"request-grant-status-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion": testOptimizationContractVersion,
				"grantId":         "test-grant-01K4Y800",
				"grantDigest":     openAPIDigest("9"),
				"grantEpoch":      7,
				"repository": map[string]any{
					"tenant":      "tenant-7",
					"repository":  "repo-42",
					"trustDomain": "trusted-ci",
				},
				"status":     "ACTIVE",
				"observedAt": "2026-07-21T10:05:00Z",
				"validUntil": "2026-07-21T10:10:00Z",
				"signature": map[string]any{
					"algorithm":           "Ed25519",
					"canonicalization":    "JCS",
					"keyId":               "testopt-signing-2026-q3",
					"signedPayloadDigest": openAPIDigest("d"),
					"value":               strings.Repeat("A", 86),
				},
			}),
			nil,
		),
		"submitTestBuildValidation": openAPIJSONReply(
			http.StatusAccepted,
			"request-01K4Y800",
			openAPIMustJSON(t, map[string]any{
				"contractVersion": testOptimizationContractVersion,
				"requestId":       "request-01K4Y800",
				"actionId":        "action-01K4Y800",
				"operationId":     "validation-operation-01K4Y800",
				"status":          "PENDING",
				"pollAfterMs":     1000,
				"expiresAt":       "2026-08-01T11:00:00Z",
			}),
			http.Header{"Retry-After": []string{"1"}},
		),
		"getTestBuildValidation": openAPIJSONReply(
			http.StatusOK,
			"request-validation-status-01",
			result,
			nil,
		),
	}
}
