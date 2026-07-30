package launcher

import (
	"fmt"
	"io"
	"net/http"
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
