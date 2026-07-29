package schemavalidator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

const (
	controlContractVersion = "buildopt-control/v1"
	cacheContractVersion   = "buildopt-cache-control/v1"
	mockBearerToken        = "contract-test-token"
)

func TestBuildOptOpenAPIV1Documents(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	tests := []struct {
		name             string
		filename         string
		version          string
		expectedPaths    int
		expectedRefs     []string
		forbidCacheBytes bool
	}{
		{
			name:          "control",
			filename:      "buildopt-control.v1.yaml",
			version:       controlContractVersion,
			expectedPaths: 4,
			expectedRefs: []string{
				"../jsonschema/optimization-policy.v1.schema.json",
				"../jsonschema/attempt-state.v1.schema.json",
				"../jsonschema/ci-validation-request.v1.schema.json",
			},
		},
		{
			name:          "cache-control",
			filename:      "buildopt-cache-control.v1.yaml",
			version:       cacheContractVersion,
			expectedPaths: 5,
			expectedRefs: []string{
				"../jsonschema/commit-decision.v1.schema.json",
			},
			forbidCacheBytes: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(
				repositoryRoot,
				"contracts",
				"openapi",
				test.filename,
			)
			document := loadOpenAPIDocument(t, path)
			assertOpenAPIDocumentPolicy(
				t,
				document,
				test.version,
				test.expectedPaths,
			)

			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, expectedRef := range test.expectedRefs {
				if !bytes.Contains(
					source,
					[]byte("$ref: '"+expectedRef+"'"),
				) {
					t.Errorf(
						"%s does not bind external schema %s",
						test.filename,
						expectedRef,
					)
				}
			}
			if test.forbidCacheBytes {
				assertNoOpaqueCachePayloadAPI(t, document, source)
			}
		})
	}
}

func TestBuildOptOpenAPIV1MockServer(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	t.Run("control", func(t *testing.T) {
		document := loadOpenAPIDocument(
			t,
			filepath.Join(
				repositoryRoot,
				"contracts",
				"openapi",
				"buildopt-control.v1.yaml",
			),
		)
		replies := controlMockReplies(t, repositoryRoot)
		mock := newOpenAPIContractMock(t, document, replies)
		server := httptest.NewServer(mock)
		t.Cleanup(server.Close)

		contextBody := openAPIRequestContext("request-policy-01")
		policyRequest := openAPIMustJSON(t, map[string]any{
			"context":         contextBody,
			"launcherVersion": "1.3.0",
			"pluginVersion":   "1.3.0",
			"runnerClass":     "linux-amd64-4c-16g",
		})
		policyHeaders := openAPIPostHeaders(
			controlContractVersion,
			"request-policy-01",
			"",
		)
		firstPolicy := sendOpenAPIRequest(
			t,
			server.URL+"/v1/policies:resolve",
			http.MethodPost,
			policyRequest,
			policyHeaders,
		)
		assertOpenAPIResponseStatus(t, firstPolicy, http.StatusOK)
		replayedPolicy := sendOpenAPIRequest(
			t,
			server.URL+"/v1/policies:resolve",
			http.MethodPost,
			policyRequest,
			policyHeaders,
		)
		assertSameOpenAPIResponse(t, firstPolicy, replayedPolicy)

		conflictingPolicy := openAPIMustJSON(t, map[string]any{
			"context":         contextBody,
			"launcherVersion": "1.3.0",
			"pluginVersion":   "1.3.0",
			"runnerClass":     "linux-amd64-12c-32g",
		})
		conflict := sendOpenAPIRequest(
			t,
			server.URL+"/v1/policies:resolve",
			http.MethodPost,
			conflictingPolicy,
			policyHeaders,
		)
		assertProblemResponse(
			t,
			conflict,
			http.StatusConflict,
			"IDEMPOTENCY_CONFLICT",
		)

		attempt := readOpenAPIFixture(
			t,
			filepath.Join(
				repositoryRoot,
				"contracts",
				"jsonschema",
				"testdata",
				"attempt-commit.v1",
				"attempt-state",
				"valid",
				"committed.json",
			),
		)
		transitionRequest := openAPIMustJSON(t, attempt)
		transitionHeaders := openAPIPostHeaders(
			controlContractVersion,
			"command-commit-01K4Y800",
			`"5"`,
		)
		transitionHeaders.Set(
			"X-BuildOpt-Request-ID",
			"request-transition-01",
		)
		transitionHeaders.Set(
			"X-BuildOpt-Deadline",
			"2026-08-01T11:00:00Z",
		)
		transition := sendOpenAPIRequest(
			t,
			server.URL+"/v1/attempt-transitions",
			http.MethodPost,
			transitionRequest,
			transitionHeaders,
		)
		assertOpenAPIResponseStatus(t, transition, http.StatusOK)

		validation := readOpenAPIFixture(
			t,
			filepath.Join(
				repositoryRoot,
				"contracts",
				"jsonschema",
				"testdata",
				"attempt-commit.v1",
				"ci-validation-request",
				"valid",
				"full-relevant-validation.json",
			),
		)
		validationRequest := openAPIMustJSON(t, validation)
		submission := sendOpenAPIRequest(
			t,
			server.URL+"/v1/validation-requests",
			http.MethodPost,
			validationRequest,
			openAPIPostHeaders(
				controlContractVersion,
				"request-01K4Y800/action-01K4Y800",
				"",
			),
		)
		assertOpenAPIResponseStatus(t, submission, http.StatusAccepted)

		status := sendOpenAPIRequest(
			t,
			server.URL+"/v1/validation-requests/request-01K4Y800",
			http.MethodGet,
			nil,
			openAPIReadHeaders(
				controlContractVersion,
				"request-status-01",
			),
		)
		assertOpenAPIResponseStatus(t, status, http.StatusOK)
		mock.assertAllOperationsExercised(t)
	})

	t.Run("cache-control", func(t *testing.T) {
		document := loadOpenAPIDocument(
			t,
			filepath.Join(
				repositoryRoot,
				"contracts",
				"openapi",
				"buildopt-cache-control.v1.yaml",
			),
		)
		replies := cacheMockReplies(t)
		mock := newOpenAPIContractMock(t, document, replies)
		server := httptest.NewServer(mock)
		t.Cleanup(server.Close)

		startRequest := openAPIMustJSON(t, map[string]any{
			"context":   openAPIRequestContext("request-cache-start-01"),
			"attemptId": "attempt-01K4Y800",
			"owner": map[string]any{
				"ownerId":        "gateway-1",
				"leaseId":        "lease-01K4Y800",
				"leaseExpiresAt": "2026-08-01T10:15:00Z",
			},
			"namespaceGeneration":       12,
			"policyDigest":              openAPIDigest("5"),
			"configurationPolicyDigest": openAPIDigest("6"),
			"cacheContractDigest":       openAPIDigest("7"),
		})
		start := sendOpenAPIRequest(
			t,
			server.URL+"/v1/cache/attempts",
			http.MethodPost,
			startRequest,
			openAPIPostHeaders(
				cacheContractVersion,
				"request-cache-start-01",
				"",
			),
		)
		assertOpenAPIResponseStatus(t, start, http.StatusCreated)

		current := sendOpenAPIRequest(
			t,
			server.URL+"/v1/cache/attempts/attempt-01K4Y800",
			http.MethodGet,
			nil,
			openAPIReadHeaders(
				cacheContractVersion,
				"request-cache-read-01",
			),
		)
		assertOpenAPIResponseStatus(t, current, http.StatusOK)

		decision := readOpenAPIFixture(
			t,
			filepath.Join(
				repositoryRoot,
				"contracts",
				"jsonschema",
				"testdata",
				"attempt-commit.v1",
				"commit-decision",
				"valid",
				"complete-decision.json",
			),
		)
		commitRequest := openAPIMustJSON(t, decision)
		commitHeaders := openAPIPostHeaders(
			cacheContractVersion,
			"decision-01K4Y800",
			`"1"`,
		)
		commitHeaders.Set(
			"X-BuildOpt-Request-ID",
			"request-cache-commit-01",
		)
		commitHeaders.Set(
			"X-BuildOpt-Deadline",
			"2026-08-01T11:00:00Z",
		)
		firstCommit := sendOpenAPIRequest(
			t,
			server.URL+"/v1/cache/attempts/attempt-01K4Y800:commit",
			http.MethodPost,
			commitRequest,
			commitHeaders,
		)
		assertOpenAPIResponseStatus(t, firstCommit, http.StatusOK)
		replayedCommit := sendOpenAPIRequest(
			t,
			server.URL+"/v1/cache/attempts/attempt-01K4Y800:commit",
			http.MethodPost,
			commitRequest,
			commitHeaders,
		)
		assertSameOpenAPIResponse(t, firstCommit, replayedCommit)

		abortRequest := openAPIMustJSON(t, map[string]any{
			"context": openAPIRequestContext("request-cache-abort-01"),
			"reason":  "BUILD_FAILURE",
		})
		abort := sendOpenAPIRequest(
			t,
			server.URL+"/v1/cache/attempts/attempt-abort-01:abort",
			http.MethodPost,
			abortRequest,
			openAPIPostHeaders(
				cacheContractVersion,
				"request-cache-abort-01",
				`"1"`,
			),
		)
		assertOpenAPIResponseStatus(t, abort, http.StatusOK)

		revocation := sendOpenAPIRequest(
			t,
			server.URL+"/v1/cache/revocations/trusted-ci",
			http.MethodGet,
			nil,
			openAPIReadHeaders(
				cacheContractVersion,
				"request-revocation-01",
			),
		)
		assertOpenAPIResponseStatus(t, revocation, http.StatusOK)
		mock.assertAllOperationsExercised(t)
	})
}

func loadOpenAPIDocument(t *testing.T, path string) *openapi3.T {
	t.Helper()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	document, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	if err := document.Validate(loader.Context); err != nil {
		t.Fatalf("validate %s: %v", path, err)
	}
	return document
}

func assertOpenAPIDocumentPolicy(
	t *testing.T,
	document *openapi3.T,
	contractVersion string,
	expectedPaths int,
) {
	t.Helper()

	if document.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
	}
	if document.JSONSchemaDialect !=
		"https://json-schema.org/draft/2020-12/schema" {
		t.Errorf(
			"JSON Schema dialect = %q",
			document.JSONSchemaDialect,
		)
	}
	pathCount := 0
	if document.Paths != nil {
		pathCount = document.Paths.Len()
	}
	if pathCount != expectedPaths {
		t.Fatalf(
			"document has %d paths, want %d",
			pathCount,
			expectedPaths,
		)
	}
	if len(document.Servers) != 1 ||
		!strings.HasPrefix(document.Servers[0].URL, "https://") {
		t.Error("control API must expose exactly one TLS server template")
	}
	if len(document.Security) != 1 {
		t.Fatalf("document has %d security requirements, want 1", len(document.Security))
	}
	if _, ok := document.Security[0]["bearerAuth"]; !ok {
		t.Fatal("top-level security does not require bearerAuth")
	}
	if document.Components == nil {
		t.Fatal("document components are absent")
	}
	securityRef := document.Components.SecuritySchemes["bearerAuth"]
	if securityRef == nil || securityRef.Value == nil ||
		securityRef.Value.Type != "http" ||
		securityRef.Value.Scheme != "bearer" {
		t.Fatal("bearerAuth must be an HTTP bearer security scheme")
	}

	operationIDs := make(map[string]string)
	for path, pathItem := range document.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			location := method + " " + path
			if operation.OperationID == "" || operation.Summary == "" {
				t.Errorf("%s must define operationId and summary", location)
			}
			if previous, duplicate := operationIDs[operation.OperationID]; duplicate {
				t.Errorf(
					"operationId %s is shared by %s and %s",
					operation.OperationID,
					previous,
					location,
				)
			}
			operationIDs[operation.OperationID] = location
			assertOpenAPIOperationPolicy(
				t,
				method,
				location,
				operation,
				contractVersion,
			)
		}
	}
}

func assertOpenAPIOperationPolicy(
	t *testing.T,
	method string,
	location string,
	operation *openapi3.Operation,
	contractVersion string,
) {
	t.Helper()

	for _, extension := range []string{
		"x-buildopt-deadline",
		"x-buildopt-cancellation",
		"x-buildopt-retry",
		"x-buildopt-error-codes",
	} {
		if _, ok := operation.Extensions[extension]; !ok {
			t.Errorf("%s is missing %s", location, extension)
		}
	}
	retry := openAPIExtensionObject(t, operation, "x-buildopt-retry")
	if retry["idempotent"] != true {
		t.Errorf("%s is not explicitly idempotent", location)
	}
	maxBackoff, ok := retry["maxBackoffMs"].(float64)
	if !ok || maxBackoff < 0 || maxBackoff > 5000 {
		t.Errorf("%s has invalid maxBackoffMs %v", location, retry["maxBackoffMs"])
	}
	unknownResponse, ok := retry["unknownResponse"].(string)
	if !ok || unknownResponse == "" {
		t.Errorf("%s has no unknown-response behavior", location)
	}
	errorCodes := openAPIExtensionArray(
		t,
		operation,
		"x-buildopt-error-codes",
	)
	if len(errorCodes) == 0 {
		t.Errorf("%s has no stable error codes", location)
	}
	if operation.Responses == nil ||
		operation.Responses.Value("401") == nil ||
		operation.Responses.Value("504") == nil ||
		operation.Responses.Value("default") == nil {
		t.Errorf("%s lacks required fail-closed responses", location)
	}
	if !operationHasHeaderParameter(
		operation,
		"X-BuildOpt-Contract-Version",
		contractVersion,
	) {
		t.Errorf("%s lacks required contract-version header", location)
	}

	switch method {
	case http.MethodPost:
		if operation.RequestBody == nil ||
			operation.RequestBody.Value == nil ||
			!operation.RequestBody.Value.Required {
			t.Errorf("%s must require a request body", location)
		}
		if !operationHasRequiredHeader(operation, "Idempotency-Key") {
			t.Errorf("%s lacks required idempotency header", location)
		}
		if _, ok := operation.Extensions["x-buildopt-idempotency"]; !ok {
			t.Errorf("%s lacks idempotency semantics", location)
		}
	case http.MethodGet:
		if !operationHasRequiredHeader(operation, "X-BuildOpt-Request-ID") ||
			!operationHasRequiredHeader(operation, "X-BuildOpt-Deadline") {
			t.Errorf("%s lacks read request/deadline headers", location)
		}
	default:
		t.Errorf("%s uses unsupported method", location)
	}
	if _, stateful := operation.Extensions["x-buildopt-state-precondition"]; stateful &&
		!operationHasRequiredHeader(operation, "If-Match") {
		t.Errorf("%s declares state semantics without If-Match", location)
	}
}

func operationHasRequiredHeader(
	operation *openapi3.Operation,
	name string,
) bool {
	for _, parameterRef := range operation.Parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		parameter := parameterRef.Value
		if parameter.In == openapi3.ParameterInHeader &&
			strings.EqualFold(parameter.Name, name) &&
			parameter.Required {
			return true
		}
	}
	return false
}

func operationHasHeaderParameter(
	operation *openapi3.Operation,
	name string,
	constant string,
) bool {
	for _, parameterRef := range operation.Parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		parameter := parameterRef.Value
		if parameter.In != openapi3.ParameterInHeader ||
			!strings.EqualFold(parameter.Name, name) ||
			!parameter.Required ||
			parameter.Schema == nil ||
			parameter.Schema.Value == nil {
			continue
		}
		if parameter.Schema.Value.Const == constant {
			return true
		}
	}
	return false
}

func openAPIExtensionObject(
	t *testing.T,
	operation *openapi3.Operation,
	name string,
) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(operation.Extensions[name])
	if err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	var value map[string]any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return value
}

func openAPIExtensionArray(
	t *testing.T,
	operation *openapi3.Operation,
	name string,
) []any {
	t.Helper()

	encoded, err := json.Marshal(operation.Extensions[name])
	if err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	var value []any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return value
}

func assertNoOpaqueCachePayloadAPI(
	t *testing.T,
	document *openapi3.T,
	source []byte,
) {
	t.Helper()

	if bytes.Contains(source, []byte("application/octet-stream")) {
		t.Error("cache control API must not transport opaque cache payloads")
	}
	for path, item := range document.Paths.Map() {
		if item.Put != nil {
			t.Errorf("cache control path %s exposes a PUT payload operation", path)
		}
	}
}

type openAPIMockReply struct {
	status  int
	headers http.Header
	body    []byte
}

type openAPIMockReplay struct {
	digest [sha256.Size]byte
	reply  openAPIMockReply
}

type openAPIContractMock struct {
	t          *testing.T
	router     routers.Router
	replies    map[string]openAPIMockReply
	mu         sync.Mutex
	replays    map[string]openAPIMockReplay
	operations map[string]int
}

func newOpenAPIContractMock(
	t *testing.T,
	document *openapi3.T,
	replies map[string]openAPIMockReply,
) *openAPIContractMock {
	t.Helper()

	// httptest owns an ephemeral HTTP origin. The production contract keeps
	// its HTTPS server boundary; routing in this in-process mock is path-only.
	document.Servers = nil
	router, err := gorillamux.NewRouter(document)
	if err != nil {
		t.Fatalf("create OpenAPI router: %v", err)
	}
	return &openAPIContractMock{
		t:          t,
		router:     router,
		replies:    replies,
		replays:    make(map[string]openAPIMockReplay),
		operations: make(map[string]int),
	}
}

func (mock *openAPIContractMock) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		mock.t.Errorf("read mock request: %v", err)
		http.Error(writer, "request read failure", http.StatusInternalServerError)
		return
	}
	request.Body = io.NopCloser(bytes.NewReader(body))

	route, pathParameters, err := mock.router.FindRoute(request)
	if err != nil {
		mock.t.Errorf("route mock request %s %s: %v", request.Method, request.URL, err)
		http.NotFound(writer, request)
		return
	}
	validationInput := &openapi3filter.RequestValidationInput{
		Request:    request,
		PathParams: pathParameters,
		Route:      route,
		Options: &openapi3filter.Options{
			AuthenticationFunc: openAPIMockAuthenticate,
		},
	}
	if err := openapi3filter.ValidateRequest(
		request.Context(),
		validationInput,
	); err != nil {
		mock.t.Errorf(
			"mock request %s does not match %s: %v",
			request.URL,
			route.Operation.OperationID,
			err,
		)
		http.Error(writer, "request contract failure", http.StatusBadRequest)
		return
	}

	reply, ok := mock.replies[route.Operation.OperationID]
	if !ok {
		mock.t.Errorf("no mock reply for %s", route.Operation.OperationID)
		http.Error(writer, "mock operation unavailable", http.StatusNotImplemented)
		return
	}
	mock.mu.Lock()
	mock.operations[route.Operation.OperationID]++
	if request.Method == http.MethodPost {
		key := route.Operation.OperationID + ":" +
			request.Header.Get("Idempotency-Key")
		digest := sha256.Sum256(body)
		if previous, exists := mock.replays[key]; exists {
			if previous.digest != digest {
				reply = openAPIConflictReply(request.Header.Get("Idempotency-Key"))
			} else {
				reply = previous.reply
			}
		} else {
			mock.replays[key] = openAPIMockReplay{
				digest: digest,
				reply:  cloneOpenAPIReply(reply),
			}
		}
	}
	mock.mu.Unlock()

	responseInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: validationInput,
		Status:                 reply.status,
		Header:                 reply.headers.Clone(),
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
		},
	}
	responseInput.SetBodyBytes(reply.body)
	if err := openapi3filter.ValidateResponse(
		request.Context(),
		responseInput,
	); err != nil {
		mock.t.Errorf(
			"mock response for %s does not match contract: %v",
			route.Operation.OperationID,
			err,
		)
		http.Error(writer, "response contract failure", http.StatusInternalServerError)
		return
	}
	for name, values := range reply.headers {
		for _, value := range values {
			writer.Header().Add(name, value)
		}
	}
	writer.WriteHeader(reply.status)
	if _, err := writer.Write(reply.body); err != nil {
		mock.t.Errorf("write mock response: %v", err)
	}
}

func (mock *openAPIContractMock) assertAllOperationsExercised(t *testing.T) {
	t.Helper()

	mock.mu.Lock()
	defer mock.mu.Unlock()
	for operationID := range mock.replies {
		if mock.operations[operationID] == 0 {
			t.Errorf("mock operation %s was not exercised", operationID)
		}
	}
}

func openAPIMockAuthenticate(
	_ context.Context,
	input *openapi3filter.AuthenticationInput,
) error {
	if input.SecuritySchemeName != "bearerAuth" {
		return input.NewError(errors.New("unexpected security scheme"))
	}
	if input.RequestValidationInput.Request.Header.Get("Authorization") !=
		"Bearer "+mockBearerToken {
		return input.NewError(errors.New("invalid mock bearer token"))
	}
	return nil
}

func controlMockReplies(
	t *testing.T,
	repositoryRoot string,
) map[string]openAPIMockReply {
	t.Helper()

	policy := readOpenAPIBytes(
		t,
		filepath.Join(
			repositoryRoot,
			"contracts",
			"jsonschema",
			"testdata",
			"foundation-contracts.v1",
			"policy",
			"valid",
			"verified-policy.json",
		),
	)
	attempt := readOpenAPIBytes(
		t,
		filepath.Join(
			repositoryRoot,
			"contracts",
			"jsonschema",
			"testdata",
			"attempt-commit.v1",
			"attempt-state",
			"valid",
			"committed.json",
		),
	)
	return map[string]openAPIMockReply{
		"resolveOptimizationPolicy": openAPIJSONReply(
			http.StatusOK,
			"request-policy-01",
			policy,
			nil,
		),
		"appendAttemptTransition": openAPIJSONReply(
			http.StatusOK,
			"request-transition-01",
			attempt,
			http.Header{"ETag": []string{`"6"`}},
		),
		"submitValidationRequest": openAPIJSONReply(
			http.StatusAccepted,
			"request-01K4Y800",
			openAPIMustJSON(t, map[string]any{
				"contractVersion": controlContractVersion,
				"requestId":       "request-01K4Y800",
				"operationId":     "validation-operation-01K4Y800",
				"status":          "QUEUED",
				"pollAfterMs":     1000,
			}),
			http.Header{"Retry-After": []string{"1"}},
		),
		"getValidationRequest": openAPIJSONReply(
			http.StatusOK,
			"request-status-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion": controlContractVersion,
				"requestId":       "request-01K4Y800",
				"operationId":     "validation-operation-01K4Y800",
				"status":          "RUNNING",
				"updatedAt":       "2026-07-21T10:01:00Z",
				"retryAfterMs":    1000,
			}),
			http.Header{"Retry-After": []string{"1"}},
		),
	}
}

func cacheMockReplies(t *testing.T) map[string]openAPIMockReply {
	t.Helper()

	return map[string]openAPIMockReply{
		"startCacheAttempt": openAPIJSONReply(
			http.StatusCreated,
			"request-cache-start-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion":    cacheContractVersion,
				"requestId":          "request-cache-start-01",
				"attemptId":          "attempt-01K4Y800",
				"state":              "PENDING",
				"stateVersion":       1,
				"pendingObjectCount": 0,
				"expiresAt":          "2026-08-01T10:15:00Z",
			}),
			http.Header{"ETag": []string{`"1"`}},
		),
		"getCacheAttempt": openAPIJSONReply(
			http.StatusOK,
			"request-cache-read-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion":    cacheContractVersion,
				"requestId":          "request-cache-read-01",
				"attemptId":          "attempt-01K4Y800",
				"state":              "PENDING",
				"stateVersion":       1,
				"pendingObjectCount": 2,
				"expiresAt":          "2026-08-01T10:15:00Z",
			}),
			http.Header{"ETag": []string{`"1"`}},
		),
		"commitCacheAttempt": openAPIJSONReply(
			http.StatusOK,
			"request-cache-commit-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion": cacheContractVersion,
				"requestId":       "request-cache-commit-01",
				"attemptId":       "attempt-01K4Y800",
				"decisionDigest":  openAPIDigest("a"),
				"state":           "COMMITTED",
				"outcome":         "COMMITTED",
				"objectCount":     2,
				"committedAt":     "2026-07-21T10:06:00Z",
			}),
			http.Header{"ETag": []string{`"2"`}},
		),
		"abortCacheAttempt": openAPIJSONReply(
			http.StatusOK,
			"request-cache-abort-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion": cacheContractVersion,
				"requestId":       "request-cache-abort-01",
				"attemptId":       "attempt-abort-01",
				"state":           "ABORTED",
				"outcome":         "ABORTED",
				"abortedAt":       "2026-07-21T10:06:00Z",
			}),
			http.Header{"ETag": []string{`"2"`}},
		),
		"getCacheRevocationState": openAPIJSONReply(
			http.StatusOK,
			"request-revocation-01",
			openAPIMustJSON(t, map[string]any{
				"contractVersion":       cacheContractVersion,
				"requestId":             "request-revocation-01",
				"trustDomain":           "trusted-ci",
				"revocationEpoch":       42,
				"l1SecurityGeneration":  42,
				"cumulativeStateDigest": openAPIDigest("c"),
				"validUntil":            "2026-08-01T10:15:00Z",
				"authentication": map[string]any{
					"algorithm": "Ed25519",
					"keyId":     "cache-signing-2026-q3",
					"signature": strings.Repeat("A", 86),
				},
			}),
			nil,
		),
	}
}

func openAPIJSONReply(
	status int,
	requestID string,
	body []byte,
	additionalHeaders http.Header,
) openAPIMockReply {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("X-BuildOpt-Request-ID", requestID)
	for name, values := range additionalHeaders {
		for _, value := range values {
			headers.Add(name, value)
		}
	}
	return openAPIMockReply{
		status:  status,
		headers: headers,
		body:    append([]byte(nil), body...),
	}
}

func openAPIConflictReply(requestID string) openAPIMockReply {
	body, err := json.Marshal(map[string]any{
		"type":            "https://errors.buildopt.dev/idempotency-conflict",
		"title":           "Idempotency key was reused with another payload",
		"status":          http.StatusConflict,
		"errorCode":       "IDEMPOTENCY_CONFLICT",
		"requestId":       requestID,
		"retryable":       false,
		"maxBackoffMs":    0,
		"unknownResponse": "NONE",
	})
	if err != nil {
		panic(err)
	}
	return openAPIMockReply{
		status: http.StatusConflict,
		headers: func() http.Header {
			headers := make(http.Header)
			headers.Set("Content-Type", "application/problem+json")
			headers.Set("X-BuildOpt-Request-ID", requestID)
			return headers
		}(),
		body: body,
	}
}

func cloneOpenAPIReply(reply openAPIMockReply) openAPIMockReply {
	return openAPIMockReply{
		status:  reply.status,
		headers: reply.headers.Clone(),
		body:    append([]byte(nil), reply.body...),
	}
}

func openAPIRequestContext(requestID string) map[string]any {
	return map[string]any{
		"requestId": requestID,
		"repository": map[string]any{
			"tenant":      "tenant-7",
			"repository":  "repo-42",
			"trustDomain": "trusted-ci",
		},
		"revision":          "8c74f2a",
		"sourceStateDigest": "hmac-sha256:" + strings.Repeat("1", 64),
		"deadline":          "2026-08-01T11:00:00Z",
	}
}

func openAPIPostHeaders(
	contractVersion string,
	idempotencyKey string,
	statePrecondition string,
) http.Header {
	headers := http.Header{
		"X-BuildOpt-Contract-Version": []string{contractVersion},
		"Idempotency-Key":             []string{idempotencyKey},
	}
	if statePrecondition != "" {
		headers.Set("If-Match", statePrecondition)
	}
	return headers
}

func openAPIReadHeaders(
	contractVersion string,
	requestID string,
) http.Header {
	return http.Header{
		"X-BuildOpt-Contract-Version": []string{contractVersion},
		"X-BuildOpt-Request-ID":       []string{requestID},
		"X-BuildOpt-Deadline":         []string{"2026-08-01T11:00:00Z"},
	}
}

type openAPITestResponse struct {
	status  int
	headers http.Header
	body    []byte
}

func sendOpenAPIRequest(
	t *testing.T,
	url string,
	method string,
	body []byte,
	headers http.Header,
) openAPITestResponse {
	t.Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("create %s %s: %v", method, url, err)
	}
	request.Header.Set("Authorization", "Bearer "+mockBearerToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, values := range headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send %s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, url, err)
	}
	return openAPITestResponse{
		status:  response.StatusCode,
		headers: response.Header.Clone(),
		body:    responseBody,
	}
}

func assertOpenAPIResponseStatus(
	t *testing.T,
	response openAPITestResponse,
	want int,
) {
	t.Helper()
	if response.status != want {
		t.Fatalf(
			"response status = %d, want %d; body=%s",
			response.status,
			want,
			response.body,
		)
	}
}

func assertSameOpenAPIResponse(
	t *testing.T,
	first openAPITestResponse,
	second openAPITestResponse,
) {
	t.Helper()
	if first.status != second.status ||
		!bytes.Equal(first.body, second.body) ||
		first.headers.Get("ETag") != second.headers.Get("ETag") ||
		first.headers.Get("X-BuildOpt-Request-ID") !=
			second.headers.Get("X-BuildOpt-Request-ID") {
		t.Fatalf(
			"idempotent replay differs:\nfirst=%d %s\nsecond=%d %s",
			first.status,
			first.body,
			second.status,
			second.body,
		)
	}
}

func assertProblemResponse(
	t *testing.T,
	response openAPITestResponse,
	status int,
	errorCode string,
) {
	t.Helper()
	assertOpenAPIResponseStatus(t, response, status)
	var problem struct {
		ErrorCode string `json:"errorCode"`
	}
	if err := json.Unmarshal(response.body, &problem); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}
	if problem.ErrorCode != errorCode {
		t.Errorf("problem errorCode = %q, want %q", problem.ErrorCode, errorCode)
	}
}

func openAPIMustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode OpenAPI fixture: %v", err)
	}
	return encoded
}

func readOpenAPIFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	content := readOpenAPIBytes(t, path)
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func readOpenAPIBytes(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func openAPIDigest(character string) string {
	if len(character) != 1 {
		panic(fmt.Sprintf("digest character must have length 1: %q", character))
	}
	return "sha256:" + strings.Repeat(character, 64)
}
