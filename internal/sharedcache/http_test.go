package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestBoundHTTPHandlerKeepsPendingInvisibleAndServesVerifiedCommit(
	t *testing.T,
) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	request := lifecycleAttemptRequest(
		"attempt-http",
		"start-http",
		"owner-http",
		11,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	handler, err := NewHTTPHandler(storage, HTTPBinding{
		Tenant:              request.Repository.Tenant,
		NamespaceGeneration: request.NamespaceGeneration,
		PendingAttemptID:    request.AttemptID,
		AllowRead:           true,
		AllowWrite:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	mismatched, err := NewHTTPHandler(storage, HTTPBinding{
		Tenant:              "another-tenant",
		NamespaceGeneration: request.NamespaceGeneration,
		PendingAttemptID:    request.AttemptID,
		AllowWrite:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	response := serveCacheRequest(
		t,
		mismatched,
		http.MethodPut,
		"/cache/http-key",
		&panicReader{},
	)
	assertCacheResponse(t, response, http.StatusForbidden, nil)
	content := []byte("opaque Gradle HTTP cache payload")

	response = serveCacheRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/http-key",
		nil,
	)
	assertCacheResponse(t, response, http.StatusNotFound, nil)

	response = serveCacheRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/http-key",
		bytes.NewReader(content),
	)
	assertCacheResponse(t, response, http.StatusCreated, nil)
	digest := response.Header.Get("X-BuildOpt-Blob-Digest")
	if !validSHA256Digest(digest) {
		t.Fatalf("PUT digest = %q", digest)
	}
	response = serveCacheRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/http-key",
		bytes.NewReader(content),
	)
	assertCacheResponse(t, response, http.StatusOK, nil)
	response = serveCacheRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/http-key",
		nil,
	)
	assertCacheResponse(t, response, http.StatusNotFound, nil)

	oversizedRequest := httptest.NewRequest(
		http.MethodPut,
		"/cache/oversized",
		&panicReader{},
	)
	oversizedRequest.ContentLength = MaximumBlobBytes + 1
	oversizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(oversizedResponse, oversizedRequest)
	assertCacheResponse(
		t,
		oversizedResponse.Result(),
		http.StatusRequestEntityTooLarge,
		nil,
	)

	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil ||
		status.StateVersion != 2 ||
		status.PendingObjectCount != 1 {
		t.Fatalf("HTTP pending status = %+v/%v", status, err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifyLifecycleDecision(
		t,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		signLifecycleDecision(
			t,
			privateKey,
			request,
			"decision-http",
			[]CommitObject{{
				NamespaceGeneration: request.NamespaceGeneration,
				Key:                 "http-key",
				Checksum:            digest,
				SizeBytes:           int64(len(content)),
			}},
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	if _, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch,
		verified,
	); err != nil {
		t.Fatal(err)
	}
	response = serveCacheRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/http-key",
		nil,
	)
	assertCacheResponse(t, response, http.StatusOK, content)
	if response.Header.Get("Content-Length") != strconv.Itoa(len(content)) ||
		response.Header.Get("ETag") != `"`+digest+`"` {
		t.Fatalf("GET headers = %v", response.Header)
	}

	response = serveCacheRequest(
		t,
		handler,
		http.MethodPost,
		"/cache/http-key",
		nil,
	)
	assertCacheResponse(t, response, http.StatusMethodNotAllowed, nil)
	if response.Header.Get("Allow") != "GET, PUT" {
		t.Fatalf("Allow = %q", response.Header.Get("Allow"))
	}

	path, err := storage.blobs.pathForDigest(digest, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("poisoned"), 0o600); err != nil {
		t.Fatal(err)
	}
	response = serveCacheRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/http-key",
		nil,
	)
	assertCacheResponse(t, response, http.StatusNotFound, nil)
	assertRowCount(t, storage.cache.database, "committed_objects", 0)
}

func TestBoundHTTPHandlerRejectsInvalidOrUnscopedUse(t *testing.T) {
	storage := openLifecycleTestStorage(t)
	for name, binding := range map[string]HTTPBinding{
		"empty": {},
		"write without attempt": {
			Tenant:              "tenant",
			NamespaceGeneration: 1,
			AllowWrite:          true,
		},
		"read with attempt": {
			Tenant:              "tenant",
			NamespaceGeneration: 1,
			PendingAttemptID:    "attempt",
			AllowRead:           true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if handler, err := NewHTTPHandler(
				storage,
				binding,
			); err == nil || handler != nil {
				t.Fatalf("invalid binding = %+v/%v", handler, err)
			}
		})
	}
	readOnly, err := NewHTTPHandler(storage, HTTPBinding{
		Tenant:              "tenant",
		NamespaceGeneration: 1,
		AllowRead:           true,
	})
	if err != nil {
		t.Fatal(err)
	}
	response := serveCacheRequest(
		t,
		readOnly,
		http.MethodPut,
		"/cache/key",
		strings.NewReader("bytes"),
	)
	assertCacheResponse(t, response, http.StatusForbidden, nil)
	response = serveCacheRequest(
		t,
		readOnly,
		http.MethodGet,
		"/cache/key/extra",
		nil,
	)
	assertCacheResponse(t, response, http.StatusNotFound, nil)
}

func serveCacheRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body io.Reader,
) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, path, body)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Result()
}

func assertCacheResponse(
	t *testing.T,
	response *http.Response,
	wantStatus int,
	wantBody []byte,
) {
	t.Helper()
	defer response.Body.Close()
	actual, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus || !bytes.Equal(actual, wantBody) {
		t.Fatalf(
			"response = %d/%q, want %d/%q",
			response.StatusCode,
			actual,
			wantStatus,
			wantBody,
		)
	}
}

type panicReader struct{}

func (*panicReader) Read([]byte) (int, error) {
	panic("oversized request body was read")
}
