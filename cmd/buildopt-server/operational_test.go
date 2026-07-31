package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestOperationalRouterSeparatesLivenessAndReadiness(t *testing.T) {
	router := &operationalRouter{}
	applicationCalls := 0
	application := http.HandlerFunc(func(
		response http.ResponseWriter,
		_ *http.Request,
	) {
		applicationCalls++
		response.WriteHeader(http.StatusNoContent)
	})

	requireOperationalStatus(t, router, http.MethodGet, livenessPath, http.StatusOK)
	requireOperationalStatus(
		t,
		router,
		http.MethodGet,
		readinessPath,
		http.StatusServiceUnavailable,
	)
	requireOperationalStatus(
		t,
		router,
		http.MethodGet,
		"/cache/object",
		http.StatusServiceUnavailable,
	)

	router.activate(application)
	requireOperationalStatus(t, router, http.MethodHead, readinessPath, http.StatusOK)
	requireOperationalStatus(
		t,
		router,
		http.MethodGet,
		"/cache/object",
		http.StatusNoContent,
	)
	if applicationCalls != 1 {
		t.Fatalf("application calls = %d, want 1", applicationCalls)
	}

	router.deactivate()
	requireOperationalStatus(t, router, http.MethodGet, livenessPath, http.StatusOK)
	requireOperationalStatus(
		t,
		router,
		http.MethodGet,
		readinessPath,
		http.StatusServiceUnavailable,
	)
	requireOperationalStatus(
		t,
		router,
		http.MethodGet,
		"/cache/object",
		http.StatusServiceUnavailable,
	)
	if applicationCalls != 1 {
		t.Fatalf("deactivated application calls = %d, want 1", applicationCalls)
	}
}

func TestOperationalRouterRejectsMutationMethods(t *testing.T) {
	router := &operationalRouter{}
	router.activate(http.NotFoundHandler())

	for _, path := range []string{livenessPath, readinessPath} {
		requireOperationalStatus(
			t,
			router,
			http.MethodPost,
			path,
			http.StatusMethodNotAllowed,
		)
	}
}

func TestServerIsLiveButNotReadyDuringSharedReconciliation(t *testing.T) {
	originalOpen := openSharedStorage
	reconciliationStarted := make(chan struct{})
	continueReconciliation := make(chan struct{})
	openSharedStorage = func(
		ctx context.Context,
		root string,
	) (*sharedcache.Storage, error) {
		close(reconciliationStarted)
		select {
		case <-continueReconciliation:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return originalOpen(ctx, root)
	}
	t.Cleanup(func() {
		openSharedStorage = originalOpen
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stateDirectory := filepath.Join(t.TempDir(), "shared")
	output := newNotifyingWriter()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() {
		exited <- run(
			ctx,
			[]string{
				"serve",
				"--listen",
				"127.0.0.1:0",
				"--state-dir",
				stateDirectory,
			},
			func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return serverTestToken
				}
				return ""
			},
			output,
			&stderr,
		)
	}()

	waitForServerOutput(t, output, "listening on ")
	endpoint := serverEndpointFromOutput(t, output.String())
	select {
	case <-reconciliationStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("Shared reconciliation did not start")
	}
	requireServerStatus(t, endpoint+livenessPath, http.StatusOK)
	requireServerStatus(
		t,
		endpoint+readinessPath,
		http.StatusServiceUnavailable,
	)
	requireServerStatus(t, endpoint+operationalAlertsPath, http.StatusOK)
	requireServerStatus(
		t,
		endpoint+"/cache/not-ready",
		http.StatusServiceUnavailable,
	)

	close(continueReconciliation)
	waitForServerOutput(t, output, "initialized and reconciled")
	requireServerStatus(t, endpoint+readinessPath, http.StatusOK)

	cancel()
	select {
	case exitCode := <-exited:
		if exitCode != 0 {
			t.Fatalf(
				"server exit = %d, stderr = %q",
				exitCode,
				stderr.String(),
			)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}

func requireOperationalStatus(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	expected int,
) {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != expected {
		t.Fatalf("%s %s = %d, want %d", method, path, response.Code, expected)
	}
	if (path == livenessPath ||
		path == readinessPath ||
		expected == http.StatusServiceUnavailable) &&
		response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%s %s did not disable caching", method, path)
	}
}

func requireServerStatus(t *testing.T, url string, expected int) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = response.Body.Close()
	if response.StatusCode != expected {
		t.Fatalf("GET %s = %d, want %d", url, response.StatusCode, expected)
	}
	if strings.TrimSpace(response.Header.Get("Cache-Control")) != "no-store" {
		t.Fatalf("GET %s did not disable caching", url)
	}
}
