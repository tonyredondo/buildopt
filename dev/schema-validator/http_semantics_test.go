package schemavalidator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

type httpSemanticsCatalog struct {
	SchemaVersion             string                   `json:"schemaVersion"`
	MaxBackoffMS              int                      `json:"maxBackoffMs"`
	ErrorCodes                []httpSemanticsError     `json:"errorCodes"`
	UnknownResponseActions    []string                 `json:"unknownResponseActions"`
	DeadlineOutcomes          []string                 `json:"deadlineOutcomes"`
	ClientCancellationActions []string                 `json:"clientCancellationActions"`
	AcceptedMutationStates    []string                 `json:"acceptedMutationStates"`
	FaultCases                []httpSemanticsFaultCase `json:"faultCases"`
}

type httpSemanticsError struct {
	Code             string `json:"code"`
	HTTPStatus       int    `json:"httpStatus"`
	DefaultRetryable bool   `json:"defaultRetryable"`
}

type httpSemanticsFaultCase struct {
	ID             string                     `json:"id"`
	OperationClass string                     `json:"operationClass"`
	Idempotent     bool                       `json:"idempotent"`
	DeadlineMS     int                        `json:"deadlineMs"`
	PayloadDigests []string                   `json:"payloadDigests"`
	Attempts       []httpSemanticsAttempt     `json:"attempts"`
	Expected       httpSemanticsExpectedState `json:"expected"`
}

type httpSemanticsAttempt struct {
	LatencyMS    int    `json:"latencyMs"`
	Result       string `json:"result"`
	ErrorCode    string `json:"errorCode,omitempty"`
	RetryAfterMS int    `json:"retryAfterMs,omitempty"`
	Accepted     bool   `json:"accepted,omitempty"`
}

type httpSemanticsExpectedState struct {
	RequestCount   int    `json:"requestCount"`
	ElapsedMS      int    `json:"elapsedMs"`
	Outcome        string `json:"outcome"`
	RecoveryAction string `json:"recoveryAction"`
}

func TestHTTPSemanticsV1CatalogAndFaults(t *testing.T) {
	t.Parallel()

	catalog := loadHTTPSemanticsCatalog(t)
	errorsByCode := validateHTTPSemanticsCatalog(t, catalog)
	for _, faultCase := range catalog.FaultCases {
		faultCase := faultCase
		t.Run(faultCase.ID, func(t *testing.T) {
			t.Parallel()
			actual := evaluateHTTPFaultCase(
				t,
				catalog.MaxBackoffMS,
				errorsByCode,
				faultCase,
			)
			if actual != faultCase.Expected {
				t.Fatalf("state = %+v, want %+v", actual, faultCase.Expected)
			}
		})
	}
}

func TestHTTPSemanticsV1OpenAPIAudit(t *testing.T) {
	t.Parallel()

	catalog := loadHTTPSemanticsCatalog(t)
	errorsByCode := validateHTTPSemanticsCatalog(t, catalog)
	unknownActions := stringSet(catalog.UnknownResponseActions)
	deadlineOutcomes := stringSet(catalog.DeadlineOutcomes)
	cancellationActions := stringSet(catalog.ClientCancellationActions)
	acceptedStates := stringSet(catalog.AcceptedMutationStates)
	repositoryRoot := findRepositoryRoot(t)

	for _, filename := range []string{
		"buildopt-control.v1.yaml",
		"buildopt-cache-control.v1.yaml",
		"test-optimization.v1.yaml",
	} {
		filename := filename
		t.Run(filename, func(t *testing.T) {
			t.Parallel()
			document := loadOpenAPIDocument(
				t,
				filepath.Join(repositoryRoot, "contracts", "openapi", filename),
			)
			auditOpenAPIHTTPSemantics(
				t,
				document,
				catalog.MaxBackoffMS,
				errorsByCode,
				unknownActions,
				deadlineOutcomes,
				cancellationActions,
				acceptedStates,
			)
		})
	}
}

func loadHTTPSemanticsCatalog(t *testing.T) httpSemanticsCatalog {
	t.Helper()
	path := filepath.Join(
		findRepositoryRoot(t),
		"contracts",
		"test-vectors",
		"http-semantics",
		"http-semantics.v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var catalog httpSemanticsCatalog
	if err := decoder.Decode(&catalog); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if decoder.More() {
		t.Fatalf("%s contains trailing JSON", path)
	}
	return catalog
}

func validateHTTPSemanticsCatalog(
	t *testing.T,
	catalog httpSemanticsCatalog,
) map[string]httpSemanticsError {
	t.Helper()
	if catalog.SchemaVersion != "buildopt.contracts/http-semantics/v1" {
		t.Errorf("schemaVersion = %q", catalog.SchemaVersion)
	}
	if catalog.MaxBackoffMS != 5000 {
		t.Errorf("maxBackoffMs = %d, want 5000", catalog.MaxBackoffMS)
	}
	if len(catalog.ErrorCodes) == 0 || len(catalog.FaultCases) == 0 {
		t.Fatal("error catalog and fault cases must be non-empty")
	}
	errorsByCode := make(map[string]httpSemanticsError)
	var orderedCodes []string
	for _, errorPolicy := range catalog.ErrorCodes {
		if _, duplicate := errorsByCode[errorPolicy.Code]; duplicate {
			t.Errorf("duplicate error code %s", errorPolicy.Code)
		}
		if errorPolicy.HTTPStatus < 400 || errorPolicy.HTTPStatus > 599 {
			t.Errorf(
				"%s has invalid HTTP status %d",
				errorPolicy.Code,
				errorPolicy.HTTPStatus,
			)
		}
		errorsByCode[errorPolicy.Code] = errorPolicy
		orderedCodes = append(orderedCodes, errorPolicy.Code)
	}
	sortedCodes := append([]string(nil), orderedCodes...)
	sort.Strings(sortedCodes)
	for index := range orderedCodes {
		if orderedCodes[index] != sortedCodes[index] {
			t.Errorf("error catalog is not sorted at %s", orderedCodes[index])
			break
		}
	}
	for label, values := range map[string][]string{
		"unknownResponseActions":    catalog.UnknownResponseActions,
		"deadlineOutcomes":          catalog.DeadlineOutcomes,
		"clientCancellationActions": catalog.ClientCancellationActions,
		"acceptedMutationStates":    catalog.AcceptedMutationStates,
	} {
		if len(values) == 0 || len(stringSet(values)) != len(values) {
			t.Errorf("%s must be non-empty and unique", label)
		}
	}
	return errorsByCode
}

func evaluateHTTPFaultCase(
	t *testing.T,
	maxBackoffMS int,
	errorsByCode map[string]httpSemanticsError,
	faultCase httpSemanticsFaultCase,
) httpSemanticsExpectedState {
	t.Helper()
	if faultCase.DeadlineMS <= 0 ||
		len(faultCase.Attempts) == 0 ||
		len(faultCase.PayloadDigests) != len(faultCase.Attempts) {
		t.Fatalf("malformed fault case %+v", faultCase)
	}
	initialDigest := faultCase.PayloadDigests[0]
	elapsed := 0
	requests := 0
	for index, attempt := range faultCase.Attempts {
		if index > 0 {
			previous := faultCase.Attempts[index-1]
			backoff := min(previous.RetryAfterMS, maxBackoffMS)
			if elapsed+backoff >= faultCase.DeadlineMS {
				return failClosedHTTPState(
					faultCase.OperationClass,
					requests,
					faultCase.DeadlineMS,
				)
			}
			elapsed += backoff
		}
		requests++
		if faultCase.PayloadDigests[index] != initialDigest {
			return httpSemanticsExpectedState{
				RequestCount:   requests,
				ElapsedMS:      elapsed,
				Outcome:        "IDEMPOTENCY_CONFLICT",
				RecoveryAction: "NONE",
			}
		}
		if attempt.LatencyMS < 0 || attempt.RetryAfterMS < 0 {
			t.Fatalf("%s has negative time", faultCase.ID)
		}
		if elapsed+attempt.LatencyMS > faultCase.DeadlineMS {
			return failClosedHTTPState(
				faultCase.OperationClass,
				requests,
				faultCase.DeadlineMS,
			)
		}
		elapsed += attempt.LatencyMS
		switch attempt.Result {
		case "SUCCESS":
			return httpSemanticsExpectedState{
				RequestCount:   requests,
				ElapsedMS:      elapsed,
				Outcome:        "SUCCESS",
				RecoveryAction: "NONE",
			}
		case "ERROR":
			errorPolicy, exists := errorsByCode[attempt.ErrorCode]
			if !exists {
				t.Fatalf("%s uses unknown error %s", faultCase.ID, attempt.ErrorCode)
			}
			if !errorPolicy.DefaultRetryable {
				return httpSemanticsExpectedState{
					RequestCount:   requests,
					ElapsedMS:      elapsed,
					Outcome:        errorPolicy.Code,
					RecoveryAction: "NONE",
				}
			}
			if !faultCase.Idempotent {
				return httpSemanticsExpectedState{
					RequestCount:   requests,
					ElapsedMS:      elapsed,
					Outcome:        "NO_RETRY",
					RecoveryAction: "NONE",
				}
			}
		case "UNKNOWN":
			return httpSemanticsExpectedState{
				RequestCount:   requests,
				ElapsedMS:      elapsed,
				Outcome:        "UNKNOWN",
				RecoveryAction: unknownHTTPRecovery(faultCase.OperationClass),
			}
		case "CANCELLED":
			if attempt.Accepted {
				return httpSemanticsExpectedState{
					RequestCount:   requests,
					ElapsedMS:      elapsed,
					Outcome:        "ACCEPTED_DURABLE",
					RecoveryAction: unknownHTTPRecovery(faultCase.OperationClass),
				}
			}
			return httpSemanticsExpectedState{
				RequestCount:   requests,
				ElapsedMS:      elapsed,
				Outcome:        "CANCELLED",
				RecoveryAction: "NONE",
			}
		default:
			t.Fatalf("%s has unknown result %s", faultCase.ID, attempt.Result)
		}
	}
	return failClosedHTTPState(faultCase.OperationClass, requests, elapsed)
}

func failClosedHTTPState(
	operationClass string,
	requests int,
	elapsed int,
) httpSemanticsExpectedState {
	outcome := map[string]string{
		"IDEMPOTENT_MUTATION": "FAILED_CLOSED",
		"POLICY_RESOLUTION":   "BASELINE",
		"POSITIVE_GATE":       "INCONCLUSIVE",
		"READ":                "UNAVAILABLE",
		"STATEFUL_MUTATION":   "ABORTED",
	}[operationClass]
	if outcome == "" {
		outcome = "FAILED_CLOSED"
	}
	return httpSemanticsExpectedState{
		RequestCount:   requests,
		ElapsedMS:      elapsed,
		Outcome:        outcome,
		RecoveryAction: "NONE",
	}
}

func unknownHTTPRecovery(operationClass string) string {
	switch operationClass {
	case "STATEFUL_MUTATION":
		return "READ_STATE"
	case "POSITIVE_GATE":
		return "POLL_STATUS"
	case "READ":
		return "RETRY_READ"
	default:
		return "RETRY_SAME_KEY"
	}
}

func auditOpenAPIHTTPSemantics(
	t *testing.T,
	document *openapi3.T,
	maxBackoffMS int,
	errorsByCode map[string]httpSemanticsError,
	unknownActions map[string]struct{},
	deadlineOutcomes map[string]struct{},
	cancellationActions map[string]struct{},
	acceptedStates map[string]struct{},
) {
	t.Helper()
	for path, pathItem := range document.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			location := fmt.Sprintf("%s %s", method, path)
			for _, value := range openAPIExtensionArray(
				t,
				operation,
				"x-buildopt-error-codes",
			) {
				code, ok := value.(string)
				if !ok {
					t.Errorf("%s has non-string error code %v", location, value)
					continue
				}
				if _, exists := errorsByCode[code]; !exists {
					t.Errorf("%s uses uncatalogued error code %s", location, code)
				}
			}
			retry := openAPIExtensionObject(t, operation, "x-buildopt-retry")
			backoff, ok := retry["maxBackoffMs"].(float64)
			if !ok || backoff < 0 || backoff > float64(maxBackoffMS) {
				t.Errorf("%s has out-of-contract backoff %v", location, retry["maxBackoffMs"])
			}
			assertCataloguedExtensionValue(
				t,
				location,
				retry,
				"unknownResponse",
				unknownActions,
			)
			deadline := openAPIExtensionObject(t, operation, "x-buildopt-deadline")
			assertCataloguedExtensionValue(
				t,
				location,
				deadline,
				"exceededOutcome",
				deadlineOutcomes,
			)
			cancellation := openAPIExtensionObject(
				t,
				operation,
				"x-buildopt-cancellation",
			)
			assertCataloguedExtensionValue(
				t,
				location,
				cancellation,
				"clientDisconnect",
				cancellationActions,
			)
			assertCataloguedExtensionValue(
				t,
				location,
				cancellation,
				"acceptedMutation",
				acceptedStates,
			)
		}
	}
}

func assertCataloguedExtensionValue(
	t *testing.T,
	location string,
	extension map[string]any,
	field string,
	allowed map[string]struct{},
) {
	t.Helper()
	value, ok := extension[field].(string)
	if !ok {
		t.Errorf("%s has no string %s", location, field)
		return
	}
	if _, exists := allowed[value]; !exists {
		t.Errorf("%s has uncatalogued %s %s", location, field, value)
	}
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
