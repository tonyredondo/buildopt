package sessioningest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const testIngestToken = "test-session-ingest-token-0123456789abcdef"

func TestHTTPClientAndHandlerRoundTrip(t *testing.T) {
	store := NewStore()
	var observedMutex sync.Mutex
	observed := make([]PutResult, 0, 2)
	handler, err := NewHandler(
		testIngestToken,
		store,
		func(_ Record, result PutResult) {
			observedMutex.Lock()
			defer observedMutex.Unlock()
			observed = append(observed, result)
		},
	)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := NewClient(server.URL, testIngestToken)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	record := validTestRecord()
	result, err := client.Deliver(context.Background(), record)
	if err != nil || result != PutCreated {
		t.Fatalf("first delivery = %d/%v", result, err)
	}
	result, err = client.Deliver(context.Background(), record)
	if err != nil || result != PutDuplicate {
		t.Fatalf("duplicate delivery = %d/%v", result, err)
	}

	conflicting := record
	conflicting.Outcome = OutcomeBuildFailure
	conflicting.ExitCode = 37
	if _, err := client.Deliver(context.Background(), conflicting); err == nil ||
		!strings.Contains(err.Error(), "HTTP 409") {
		t.Fatalf("conflicting delivery error = %v", err)
	}

	snapshot := store.Snapshot()
	if len(snapshot) != 1 || snapshot[0] != record {
		t.Fatalf("unexpected accepted sessions: %+v", snapshot)
	}
	observedMutex.Lock()
	defer observedMutex.Unlock()
	if len(observed) != 2 ||
		observed[0] != PutCreated ||
		observed[1] != PutDuplicate {
		t.Fatalf("unexpected observations: %+v", observed)
	}
}

func TestHTTPHandlerRejectsInvalidRequestsWithoutLeakingToken(t *testing.T) {
	handler, err := NewHandler(testIngestToken, NewStore(), nil)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	recordBody, err := json.Marshal(validTestRecord())
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}

	testCases := []struct {
		name        string
		method      string
		path        string
		token       string
		contentType string
		idempotency string
		body        []byte
		wantStatus  int
	}{
		{
			name:        "missing authentication",
			method:      http.MethodPost,
			path:        IngestPath,
			contentType: "application/json",
			idempotency: validTestRecord().SessionID,
			body:        recordBody,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "wrong authentication",
			method:      http.MethodPost,
			path:        IngestPath,
			token:       strings.Repeat("x", len(testIngestToken)),
			contentType: "application/json",
			idempotency: validTestRecord().SessionID,
			body:        recordBody,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "unknown path",
			method:      http.MethodPost,
			path:        "/internal/v1/unknown",
			token:       testIngestToken,
			contentType: "application/json",
			idempotency: validTestRecord().SessionID,
			body:        recordBody,
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "wrong method",
			method:      http.MethodGet,
			path:        IngestPath,
			token:       testIngestToken,
			contentType: "application/json",
			idempotency: validTestRecord().SessionID,
			body:        recordBody,
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			path:        IngestPath,
			token:       testIngestToken,
			contentType: "text/plain",
			idempotency: validTestRecord().SessionID,
			body:        recordBody,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "wrong idempotency key",
			method:      http.MethodPost,
			path:        IngestPath,
			token:       testIngestToken,
			contentType: "application/json",
			idempotency: "other-session",
			body:        recordBody,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "unknown JSON field",
			method:      http.MethodPost,
			path:        IngestPath,
			token:       testIngestToken,
			contentType: "application/json",
			idempotency: validTestRecord().SessionID,
			body: bytes.Replace(
				recordBody,
				[]byte(`"exitCode":0`),
				[]byte(`"exitCode":0,"unexpected":true`),
				1,
			),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "oversized body",
			method:      http.MethodPost,
			path:        IngestPath,
			token:       testIngestToken,
			contentType: "application/json",
			idempotency: validTestRecord().SessionID,
			body:        []byte(strings.Repeat("x", maxRequestBytes+1)),
			wantStatus:  http.StatusRequestEntityTooLarge,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request, err := http.NewRequest(
				testCase.method,
				server.URL+testCase.path,
				bytes.NewReader(testCase.body),
			)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			if testCase.token != "" {
				request.Header.Set(
					"Authorization",
					"Bearer "+testCase.token,
				)
			}
			request.Header.Set("Content-Type", testCase.contentType)
			request.Header.Set("Idempotency-Key", testCase.idempotency)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("send request: %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				t.Fatalf("read response: %v", readErr)
			}
			if response.StatusCode != testCase.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					response.StatusCode,
					testCase.wantStatus,
				)
			}
			if bytes.Contains(body, []byte(testIngestToken)) {
				t.Fatal("rejection response exposed the ingest token")
			}
		})
	}
}

func TestClientConfiguration(t *testing.T) {
	values := map[string]string{}
	getenv := func(key string) string {
		return values[key]
	}
	if client, configured, err := ClientFromEnvironment(getenv); err != nil ||
		configured ||
		client != nil {
		t.Fatalf("absent config = %v/%v/%v", client, configured, err)
	}

	values[ServerURLEnvironment] = "http://127.0.0.1:8042"
	if _, configured, err := ClientFromEnvironment(getenv); err == nil ||
		configured {
		t.Fatalf("incomplete config = %v/%v", configured, err)
	}
	values[ServerTokenEnvironment] = testIngestToken
	if client, configured, err := ClientFromEnvironment(getenv); err != nil ||
		!configured ||
		client == nil {
		t.Fatalf("complete config = %v/%v/%v", client, configured, err)
	}

	for _, endpoint := range []string{
		"https://127.0.0.1:8042",
		"http://localhost:8042",
		"http://127.0.0.1",
		"http://127.0.0.1:8042/path",
		"http://user@127.0.0.1:8042",
	} {
		if _, err := NewClient(endpoint, testIngestToken); err == nil {
			t.Errorf("accepted invalid endpoint %q", endpoint)
		}
	}
	if _, err := NewClient(
		"http://127.0.0.1:8042",
		"short-token",
	); err == nil {
		t.Fatal("accepted a short ingest token")
	}
}
