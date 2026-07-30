package launcher

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLocalGatewayAuthenticationAndRestart(t *testing.T) {
	gateway, err := startLocalGateway()
	if err != nil {
		t.Fatalf("start local gateway: %v", err)
	}
	defer func() {
		if err := gateway.close(); err != nil {
			t.Errorf("close local gateway: %v", err)
		}
	}()

	initialEndpoint := gateway.endpoint
	initialPassword := gateway.password
	initialGeneration := gateway.generation

	status, headers, body, err := requestLocalGateway(
		gateway.endpoint,
		gateway.username,
		gateway.password,
		http.MethodGet,
		gatewayReadyPath,
	)
	if err != nil {
		t.Fatalf("request authenticated readiness: %v", err)
	}
	if status != http.StatusNoContent ||
		headers.Get(gatewayGenerationHeader) != gateway.generation ||
		len(body) != 0 {
		t.Fatalf(
			"authenticated readiness = %d/%q/%q",
			status,
			headers.Get(gatewayGenerationHeader),
			body,
		)
	}

	status, headers, body, err = requestLocalGateway(
		gateway.endpoint,
		gateway.username,
		"wrong-"+gateway.password,
		http.MethodGet,
		gatewayReadyPath,
	)
	if err != nil {
		t.Fatalf("request rejected readiness: %v", err)
	}
	if status != http.StatusUnauthorized ||
		headers.Get("WWW-Authenticate") == "" ||
		strings.Contains(string(body), gateway.password) {
		t.Fatalf("invalid credential response = %d/%q", status, body)
	}

	status, _, _, err = requestLocalGateway(
		gateway.endpoint,
		gateway.username,
		gateway.password,
		http.MethodGet,
		"/cache/not-enabled",
	)
	if err != nil || status != http.StatusNotFound {
		t.Fatalf("neutral data route = %d/%v, want 404", status, err)
	}

	if err := gateway.restart(); err != nil {
		t.Fatalf("restart local gateway: %v", err)
	}
	if gateway.endpoint != initialEndpoint ||
		gateway.password != initialPassword ||
		gateway.generation != initialGeneration {
		t.Fatal("gateway restart changed its rendezvous identity")
	}
	if err := gateway.probe(); err != nil {
		t.Fatalf("probe restarted gateway: %v", err)
	}

	if err := gateway.close(); err != nil {
		t.Fatalf("close restarted gateway: %v", err)
	}
	_, _, _, err = requestLocalGateway(
		gateway.endpoint,
		gateway.username,
		gateway.password,
		http.MethodGet,
		gatewayReadyPath,
	)
	if err == nil {
		t.Fatal("closed gateway still accepts requests")
	}
}

func TestLocalGatewayConcurrentIsolation(t *testing.T) {
	const gatewayCount = 4
	const requestsPerGateway = 16

	gateways := make([]*localGateway, 0, gatewayCount)
	for range gatewayCount {
		gateway, err := startLocalGateway()
		if err != nil {
			t.Fatalf("start concurrent gateway: %v", err)
		}
		gateways = append(gateways, gateway)
	}
	defer func() {
		for _, gateway := range gateways {
			if err := gateway.close(); err != nil {
				t.Errorf("close concurrent gateway: %v", err)
			}
		}
	}()

	endpoints := make(map[string]struct{}, gatewayCount)
	passwords := make(map[string]struct{}, gatewayCount)
	generations := make(map[string]struct{}, gatewayCount)
	for _, gateway := range gateways {
		endpoints[gateway.endpoint] = struct{}{}
		passwords[gateway.password] = struct{}{}
		generations[gateway.generation] = struct{}{}
	}
	if len(endpoints) != gatewayCount ||
		len(passwords) != gatewayCount ||
		len(generations) != gatewayCount {
		t.Fatal("concurrent gateways reused endpoint, credential, or generation")
	}

	for index, gateway := range gateways {
		other := gateways[(index+1)%len(gateways)]
		status, _, _, err := requestLocalGateway(
			gateway.endpoint,
			other.username,
			other.password,
			http.MethodGet,
			gatewayReadyPath,
		)
		if err != nil {
			t.Fatalf("request cross-slot credential: %v", err)
		}
		if status != http.StatusUnauthorized {
			t.Fatalf("cross-slot credential returned %d, want 401", status)
		}
	}

	var waitGroup sync.WaitGroup
	failures := make(chan error, gatewayCount*requestsPerGateway)
	for _, gateway := range gateways {
		for range requestsPerGateway {
			waitGroup.Add(1)
			go func(target *localGateway) {
				defer waitGroup.Done()
				status, headers, _, err := requestLocalGateway(
					target.endpoint,
					target.username,
					target.password,
					http.MethodGet,
					gatewayReadyPath,
				)
				if err != nil {
					failures <- err
					return
				}
				if status != http.StatusNoContent ||
					headers.Get(gatewayGenerationHeader) != target.generation {
					failures <- fmt.Errorf(
						"gateway %s returned %d/%q",
						target.endpoint,
						status,
						headers.Get(gatewayGenerationHeader),
					)
				}
			}(gateway)
		}
	}
	waitGroup.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func TestLocalGatewayRoutesOnlyCurrentAuthenticatedCacheContext(
	t *testing.T,
) {
	credential := bytes.Repeat([]byte{0x42}, 32)
	authorityDigest := "sha256:" + strings.Repeat("a", 64)
	attemptID := "01234567-89ab-cdef-0123-456789abcdef"
	var receivedAuthorization string
	var receivedAuthority string
	var receivedBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		receivedAuthorization = request.Header.Get("Authorization")
		receivedAuthority = request.Header.Get(gatewayAuthorityHeader)
		switch request.Method {
		case http.MethodGet:
			writer.Header().Set("ETag", `"payload"`)
			_, _ = io.WriteString(writer, "cached")
		case http.MethodPut:
			receivedBody, _ = io.ReadAll(request.Body)
			writer.Header().Set(
				"X-BuildOpt-Blob-Digest",
				"sha256:"+strings.Repeat("b", 64),
			)
			writer.WriteHeader(http.StatusCreated)
		default:
			writer.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer upstream.Close()
	binding, err := newGatewayCacheBinding(
		upstream.URL,
		credential,
		authorityDigest,
		attemptID,
		true,
		true,
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("create cache binding: %v", err)
	}
	gateway, err := startLocalGatewayWithCache(binding)
	if err != nil {
		t.Fatalf("start local gateway with cache: %v", err)
	}
	defer gateway.close()

	request := func(method string, path string, body io.Reader) (
		int,
		http.Header,
		[]byte,
	) {
		t.Helper()
		httpRequest, requestErr := http.NewRequest(
			method,
			gateway.endpoint+path,
			body,
		)
		if requestErr != nil {
			t.Fatalf("create local cache request: %v", requestErr)
		}
		httpRequest.SetBasicAuth(gateway.username, gateway.password)
		httpRequest.Header.Set("Authorization", "Bearer untrusted-parent")
		httpRequest.SetBasicAuth(gateway.username, gateway.password)
		response, requestErr := (&http.Client{
			Timeout:   2 * time.Second,
			Transport: &http.Transport{Proxy: nil},
		}).Do(httpRequest)
		if requestErr != nil {
			t.Fatalf("request local cache: %v", requestErr)
		}
		defer response.Body.Close()
		responseBody, requestErr := io.ReadAll(response.Body)
		if requestErr != nil {
			t.Fatalf("read local cache response: %v", requestErr)
		}
		return response.StatusCode, response.Header.Clone(), responseBody
	}

	status, headers, body := request(http.MethodGet, "/cache/key-1", nil)
	if status != http.StatusOK ||
		string(body) != "cached" ||
		headers.Get("ETag") != `"payload"` {
		t.Fatalf("cache GET = %d/%q/%q", status, headers, body)
	}
	expectedCredential := base64.RawURLEncoding.EncodeToString(credential)
	if receivedAuthorization != "Bearer "+expectedCredential ||
		receivedAuthority != authorityDigest {
		t.Fatalf(
			"upstream authority = %q/%q",
			receivedAuthorization,
			receivedAuthority,
		)
	}

	status, headers, body = request(
		http.MethodPut,
		"/cache/key-1",
		strings.NewReader("candidate"),
	)
	if status != http.StatusCreated ||
		len(body) != 0 ||
		string(receivedBody) != "candidate" ||
		headers.Get("X-BuildOpt-Blob-Digest") == "" {
		t.Fatalf(
			"cache PUT = %d/%q/%q/%q",
			status,
			headers,
			body,
			receivedBody,
		)
	}
}

func TestLocalGatewayCacheFailsClosed(t *testing.T) {
	tests := []struct {
		name           string
		upstreamStatus int
		method         string
		allowRead      bool
		allowWrite     bool
		expiresAt      time.Time
		want           int
	}{
		{
			name:           "get upstream error becomes miss",
			upstreamStatus: http.StatusServiceUnavailable,
			method:         http.MethodGet,
			allowRead:      true,
			expiresAt:      time.Now().Add(time.Hour),
			want:           http.StatusNotFound,
		},
		{
			name:           "put upstream error disables remote",
			upstreamStatus: http.StatusConflict,
			method:         http.MethodPut,
			allowWrite:     true,
			expiresAt:      time.Now().Add(time.Hour),
			want:           http.StatusServiceUnavailable,
		},
		{
			name:           "expired context is absent",
			upstreamStatus: http.StatusOK,
			method:         http.MethodGet,
			allowRead:      true,
			expiresAt:      time.Now().Add(-time.Second),
			want:           http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				writer.WriteHeader(test.upstreamStatus)
			}))
			defer upstream.Close()
			binding, err := newGatewayCacheBinding(
				upstream.URL,
				bytes.Repeat([]byte{0x13}, 32),
				"sha256:"+strings.Repeat("c", 64),
				"01234567-89ab-cdef-0123-456789abcdef",
				test.allowRead,
				test.allowWrite,
				test.expiresAt,
			)
			if err != nil {
				t.Fatalf("create cache binding: %v", err)
			}
			gateway, err := startLocalGatewayWithCache(binding)
			if err != nil {
				t.Fatalf("start local gateway: %v", err)
			}
			defer gateway.close()
			status, _, _, err := requestLocalGateway(
				gateway.endpoint,
				gateway.username,
				gateway.password,
				test.method,
				"/cache/key",
			)
			if err != nil || status != test.want {
				t.Fatalf(
					"cache failure response = %d/%v, want %d",
					status,
					err,
					test.want,
				)
			}
		})
	}
}

func TestLocalGatewayChildEnvironment(t *testing.T) {
	gateway, err := startLocalGateway()
	if err != nil {
		t.Fatalf("start local gateway: %v", err)
	}
	defer func() {
		if err := gateway.close(); err != nil {
			t.Errorf("close local gateway: %v", err)
		}
	}()

	environment := gateway.childEnvironment([]string{
		"PATH=/usr/bin",
		bypassEnvironment + "=parent-bypass",
		gatewayURLEnvironment + "=http://127.0.0.1:1",
		gatewayUsernameEnvironment + "=parent-user",
		gatewayPasswordEnvironment + "=parent-password",
		gatewayGenerationEnvironment + "=parent-generation",
		managedGatewayStateRootEnvironment + "=/tmp/untrusted-state",
		managedRunnerSlotEnvironment + "=untrusted-slot",
		managedGatewayIdleTimeoutEnvironment + "=1ms",
		localAuthorityPathEnvironment + "=/tmp/untrusted-authority",
		localTrustRootPathEnvironment + "=/tmp/untrusted-root",
		localCredentialPathEnvironment + "=/tmp/untrusted-credential",
		sharedCacheURLEnvironment + "=http://127.0.0.1:2",
		managedSharedModeEnvironment + "=READ_WRITE",
		managedAuthorityDigestEnvironment + "=untrusted-authority",
		managedPolicyDigestEnvironment + "=untrusted-policy",
		managedConfigurationDigestEnvironment + "=untrusted-configuration",
		managedAuthorityContractEnvironment + "=untrusted-contract",
	})
	expected := map[string]string{
		gatewayURLEnvironment:        gateway.endpoint,
		gatewayUsernameEnvironment:   gateway.username,
		gatewayPasswordEnvironment:   gateway.password,
		gatewayGenerationEnvironment: gateway.generation,
	}
	for key, value := range expected {
		if actual := environmentValue(environment, key); actual != value {
			t.Fatalf("%s = %q, want invocation value", key, actual)
		}
		if count := environmentKeyCount(environment, key); count != 1 {
			t.Fatalf("child environment contains %d %s entries", count, key)
		}
	}
	if count := environmentKeyCount(environment, bypassEnvironment); count != 0 {
		t.Fatalf("child environment contains %d bypass entries, want 0", count)
	}
	for _, key := range []string{
		managedGatewayStateRootEnvironment,
		managedRunnerSlotEnvironment,
		managedGatewayIdleTimeoutEnvironment,
		localAuthorityPathEnvironment,
		localTrustRootPathEnvironment,
		localCredentialPathEnvironment,
		sharedCacheURLEnvironment,
		managedSharedModeEnvironment,
		managedAuthorityDigestEnvironment,
		managedPolicyDigestEnvironment,
		managedConfigurationDigestEnvironment,
		managedAuthorityContractEnvironment,
	} {
		if count := environmentKeyCount(environment, key); count != 0 {
			t.Fatalf("child environment contains %d %s entries, want 0", count, key)
		}
	}
}

func requestLocalGateway(
	endpoint string,
	username string,
	password string,
	method string,
	path string,
) (int, http.Header, []byte, error) {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			Proxy:             nil,
			DisableKeepAlives: true,
		},
	}
	request, err := http.NewRequest(method, endpoint+path, nil)
	if err != nil {
		return 0, nil, nil, err
	}
	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, nil, err
	}
	return response.StatusCode, response.Header.Clone(), body, nil
}

func TestLocalGatewayCloseIsIdempotent(t *testing.T) {
	gateway, err := startLocalGateway()
	if err != nil {
		t.Fatalf("start local gateway: %v", err)
	}
	if err := gateway.close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := gateway.close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if err := gateway.restart(); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("restart closed gateway error = %v", err)
	}
}
