package openapiclient

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGeneratedEndpointInventory(t *testing.T) {
	t.Parallel()

	endpoints := Endpoints()
	if len(endpoints) != 13 {
		t.Fatalf("endpoint count = %d, want 13", len(endpoints))
	}
	operationIDs := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.OperationID == "" ||
			endpoint.Path == "" ||
			endpoint.ContractVersion == "" {
			t.Errorf("incomplete endpoint: %+v", endpoint)
		}
		if endpoint.Method != http.MethodGet &&
			endpoint.Method != http.MethodPost {
			t.Errorf("%s uses method %s", endpoint.OperationID, endpoint.Method)
		}
		if _, duplicate := operationIDs[endpoint.OperationID]; duplicate {
			t.Errorf("duplicate operation ID %s", endpoint.OperationID)
		}
		operationIDs[endpoint.OperationID] = struct{}{}
	}
	endpoints[0].OperationID = "mutated-copy"
	fresh := Endpoints()
	if fresh[0].OperationID == "mutated-copy" {
		t.Fatal("endpoint inventory leaked mutable generated state")
	}
}

func TestGeneratedCompatibilityVectors(t *testing.T) {
	t.Parallel()

	path := filepath.Join(
		findGeneratedClientRepositoryRoot(t),
		"contracts",
		"test-vectors",
		"compatibility",
		"n-n-minus-1.tsv",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open compatibility vectors: %v", err)
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 8 {
			t.Fatalf("compatibility row has %d fields: %q", len(fields), line)
		}
		localMajor := parseGeneratedClientInteger(t, fields[0], fields[1])
		localMinor := parseGeneratedClientInteger(t, fields[0], fields[2])
		remoteMajor := parseGeneratedClientInteger(t, fields[0], fields[3])
		remoteMinor := parseGeneratedClientInteger(t, fields[0], fields[4])
		unknownFields, err := strconv.ParseBool(fields[6])
		if err != nil {
			t.Fatalf("%s unknown_fields: %v", fields[0], err)
		}
		actual := Negotiate(
			localMajor,
			localMinor,
			remoteMajor,
			remoteMinor,
			Shape(fields[5]),
			unknownFields,
		)
		if actual != CompatibilityDecision(fields[7]) {
			t.Errorf("%s decision = %s, want %s", fields[0], actual, fields[7])
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read compatibility vectors: %v", err)
	}
	if count != 9 {
		t.Fatalf("compatibility vector count = %d, want 9", count)
	}
}

func TestGeneratedClientHTTPSRequest(t *testing.T) {
	t.Parallel()

	requestBody := []byte(`{"contractVersion":"test-optimization/v1"}`)
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodPost ||
				request.URL.Path != "/v1/test-cache-grants:resolve" {
				t.Errorf("request = %s %s", request.Method, request.URL.Path)
			}
			if request.Header.Get("Authorization") != "Bearer test-token" ||
				request.Header.Get("X-BuildOpt-Contract-Version") !=
					"test-optimization/v1" ||
				request.Header.Get("Idempotency-Key") != "request-grant-01" {
				t.Errorf("required generated headers are absent: %v", request.Header)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("read request: %v", err)
			}
			if !bytes.Equal(body, requestBody) {
				t.Errorf("body = %s, want %s", body, requestBody)
			}
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{"status":"ok"}`))
		},
	))
	t.Cleanup(server.Close)

	endpoint, ok := EndpointByOperationID("resolveTestCacheGrant")
	if !ok {
		t.Fatal("resolveTestCacheGrant endpoint is absent")
	}
	response, err := (Client{
		BaseURL:    server.URL,
		Token:      "test-token",
		HTTPClient: server.Client(),
	}).Do(
		context.Background(),
		endpoint,
		nil,
		"request-grant-01",
		"request-grant-01",
		"",
		time.Now().Add(time.Minute),
		requestBody,
	)
	if err != nil {
		t.Fatalf("generated request: %v", err)
	}
	if response.StatusCode != http.StatusOK ||
		string(response.Body) != `{"status":"ok"}` {
		t.Fatalf("response = %d %s", response.StatusCode, response.Body)
	}
}

func TestGeneratedClientRejectsUnsafeRequests(t *testing.T) {
	t.Parallel()

	endpoint, ok := EndpointByOperationID("resolveTestCacheGrant")
	if !ok {
		t.Fatal("resolveTestCacheGrant endpoint is absent")
	}
	_, err := (Client{
		BaseURL: "http://127.0.0.1",
		Token:   "token",
	}).Do(
		context.Background(),
		endpoint,
		nil,
		"request-1",
		"key-1",
		"",
		time.Now().Add(time.Minute),
		[]byte(`{}`),
	)
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("unsafe base URL error = %v", err)
	}

	_, err = (Client{
		BaseURL: "https://example.invalid",
		Token:   "token",
	}).Do(
		context.Background(),
		endpoint,
		nil,
		"request-1",
		"",
		"",
		time.Now().Add(time.Minute),
		[]byte(`{}`),
	)
	if err == nil || !strings.Contains(err.Error(), "idempotency") {
		t.Fatalf("missing idempotency error = %v", err)
	}

	readEndpoint, ok := EndpointByOperationID("getTestCacheGrantStatus")
	if !ok {
		t.Fatal("getTestCacheGrantStatus endpoint is absent")
	}
	_, err = (Client{
		BaseURL: "https://example.invalid",
		Token:   "token",
	}).Do(
		context.Background(),
		readEndpoint,
		nil,
		"request-1",
		"",
		"",
		time.Now().Add(time.Minute),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "unresolved path") {
		t.Fatalf("unresolved path error = %v", err)
	}
}

func TestGeneratedClientEscapesPathParameters(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.RequestURI !=
				"/v1/test-cache-grants/grant%2Fa/status" {
				t.Errorf("RequestURI = %q", request.RequestURI)
			}
			writer.WriteHeader(http.StatusOK)
		},
	))
	t.Cleanup(server.Close)
	endpoint, ok := EndpointByOperationID("getTestCacheGrantStatus")
	if !ok {
		t.Fatal("getTestCacheGrantStatus endpoint is absent")
	}
	_, err := (Client{
		BaseURL:    server.URL,
		Token:      "test-token",
		HTTPClient: server.Client(),
	}).Do(
		context.Background(),
		endpoint,
		map[string]string{"grantId": "grant/a"},
		"request-read-1",
		"",
		"",
		time.Now().Add(time.Minute),
		nil,
	)
	if err != nil {
		t.Fatalf("generated path request: %v", err)
	}
}

func parseGeneratedClientInteger(t *testing.T, name string, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("%s integer %q: %v", name, value, err)
	}
	return parsed
}

func findGeneratedClientRepositoryRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(current, "contracts")); err == nil {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("repository root not found")
		}
		current = parent
	}
}
