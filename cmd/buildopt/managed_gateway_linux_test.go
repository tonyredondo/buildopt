//go:build linux

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type managedGatewayStateFixture struct {
	SchemaVersion int    `json:"schemaVersion"`
	Address       string `json:"address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Generation    string `json:"gatewayConnectionGeneration"`
}

type managedGatewayCacheRegistrationFixture struct {
	UpstreamEndpoint string `json:"upstreamEndpoint"`
	Credential       string `json:"credential"`
	AuthorityDigest  string `json:"authorityDigest"`
	AllowRead        bool   `json:"allowRead"`
	AllowWrite       bool   `json:"allowWrite"`
	ExpiresAt        string `json:"expiresAt"`
}

type managedGatewayControlRequestFixture struct {
	SchemaVersion int                                    `json:"schemaVersion"`
	Operation     string                                 `json:"operation"`
	AttemptID     string                                 `json:"attemptId"`
	Cache         managedGatewayCacheRegistrationFixture `json:"cache"`
}

type managedGatewayControlResponseFixture struct {
	SchemaVersion int    `json:"schemaVersion"`
	Endpoint      string `json:"endpoint"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Generation    string `json:"gatewayConnectionGeneration"`
	Error         string `json:"error"`
}

type managedGatewayCommandFixture struct {
	command *exec.Cmd
	stderr  bytes.Buffer
}

func TestBuildoptManagedGatewayPersistsAndRotatesSafely(t *testing.T) {
	buildoptBinary := buildBuildopt(t)
	stateRoot := filepath.Join(t.TempDir(), "runtime")
	const slot = "runner-01.internal"
	const idleTimeout = "250ms"

	first := runManagedBuildoptHelper(
		t,
		buildoptBinary,
		stateRoot,
		slot,
		idleTimeout,
		0,
	)
	if first.GatewayReady != http.StatusNoContent ||
		first.GatewayGeneration == "" ||
		first.PluginAttemptID == "" {
		t.Fatalf("first managed invocation did not receive readiness: %+v", first)
	}
	firstState := managedGatewayStateForEndpoint(
		t,
		stateRoot,
		first.GatewayURL,
	)
	if firstState.Generation != first.GatewayGeneration {
		t.Fatalf(
			"state generation = %q, want %q",
			firstState.Generation,
			first.GatewayGeneration,
		)
	}
	assertManagedGatewayStatePermissions(t, stateRoot)
	waitManagedGatewayStatus(
		t,
		firstState,
		http.StatusServiceUnavailable,
		2*time.Second,
	)
	status, err := requestManagedGatewayState(
		firstState,
		firstState.Password+"-wrong",
	)
	if err != nil || status != http.StatusUnauthorized {
		t.Fatalf("released gateway wrong credential = %d/%v, want 401", status, err)
	}

	waitManagedGatewayUnavailable(t, first.GatewayURL, 3*time.Second)
	second := runManagedBuildoptHelper(
		t,
		buildoptBinary,
		stateRoot,
		slot,
		idleTimeout,
		0,
	)
	if second.GatewayURL != first.GatewayURL ||
		second.GatewayGeneration != first.GatewayGeneration ||
		second.PluginAttemptID == first.PluginAttemptID {
		t.Fatalf(
			"managed restart changed identity or reused attempt: first=%+v second=%+v",
			first,
			second,
		)
	}
	secondState := managedGatewayStateForEndpoint(
		t,
		stateRoot,
		second.GatewayURL,
	)
	if secondState.Password != firstState.Password {
		t.Fatal("managed restart changed the local credential")
	}

	waitManagedGatewayUnavailable(t, second.GatewayURL, 3*time.Second)
	blockedAddress, err := net.Listen("tcp4", secondState.Address)
	if err != nil {
		t.Fatalf("occupy previous managed gateway endpoint: %v", err)
	}
	third := runManagedBuildoptHelper(
		t,
		buildoptBinary,
		stateRoot,
		slot,
		idleTimeout,
		0,
	)
	_ = blockedAddress.Close()
	if third.GatewayURL == second.GatewayURL ||
		third.GatewayGeneration == second.GatewayGeneration {
		t.Fatalf(
			"endpoint conflict did not rotate the complete identity: second=%+v third=%+v",
			second,
			third,
		)
	}
	thirdState := managedGatewayStateForEndpoint(
		t,
		stateRoot,
		third.GatewayURL,
	)
	if thirdState.Password == secondState.Password {
		t.Fatal("endpoint rotation retained the previous local credential")
	}
	waitManagedGatewayUnavailable(t, third.GatewayURL, 3*time.Second)
}

func TestBuildoptManagedGatewayEnforcesRunnerSlotLeaseAndIsolation(t *testing.T) {
	buildoptBinary := buildBuildopt(t)
	stateRoot := filepath.Join(t.TempDir(), "runtime")
	testDirectory := t.TempDir()
	readyPath := filepath.Join(testDirectory, "ready")
	releasePath := filepath.Join(testDirectory, "release")
	const idleTimeout = "250ms"

	command := exec.Command(
		buildoptBinary,
		"run",
		"--",
		"/bin/sh",
		"-c",
		`printf '%s\n' "$BUILDOPT_GATEWAY_URL" >"$1"
while test ! -f "$2"; do sleep 0.01; done`,
		"managed-slot-holder",
		readyPath,
		releasePath,
	)
	command.Env = managedGatewayTestEnvironment(
		stateRoot,
		"runner-01.internal",
		idleTimeout,
	)
	var firstStderr bytes.Buffer
	command.Stderr = &firstStderr
	if err := command.Start(); err != nil {
		t.Fatalf("start first managed invocation: %v", err)
	}
	waitResult := make(chan error, 1)
	go func() {
		waitResult <- command.Wait()
	}()
	finished := false
	t.Cleanup(func() {
		if finished {
			return
		}
		_ = os.WriteFile(releasePath, []byte("release\n"), 0o600)
		_ = command.Process.Kill()
	})

	firstEndpoint := waitManagedGatewayReadyFile(t, readyPath, 5*time.Second)
	if !strings.HasPrefix(firstEndpoint, "http://127.0.0.1:") {
		t.Fatalf("first managed endpoint = %q", firstEndpoint)
	}

	blocked := runManagedBuildoptHelper(
		t,
		buildoptBinary,
		stateRoot,
		"runner-01.internal",
		idleTimeout,
		37,
	)
	if blocked.GatewayURL != "" ||
		blocked.GatewayGeneration != "" ||
		blocked.PluginAttemptID != "" {
		t.Fatalf("busy runner slot exposed partial context: %+v", blocked)
	}

	isolated := runManagedBuildoptHelper(
		t,
		buildoptBinary,
		stateRoot,
		"runner-02.internal",
		idleTimeout,
		0,
	)
	if isolated.GatewayURL == "" ||
		isolated.GatewayURL == firstEndpoint ||
		isolated.GatewayGeneration == "" {
		t.Fatalf("second runner slot was not isolated: %+v", isolated)
	}

	states := readManagedGatewayStates(t, stateRoot)
	if len(states) != 2 {
		t.Fatalf("managed gateway state count = %d, want 2", len(states))
	}
	firstState := stateForEndpoint(t, states, firstEndpoint)
	secondState := stateForEndpoint(t, states, isolated.GatewayURL)
	status, err := requestManagedGatewayState(firstState, secondState.Password)
	if err != nil || status != http.StatusUnauthorized {
		t.Fatalf("cross-slot credential = %d/%v, want 401", status, err)
	}

	if err := os.WriteFile(releasePath, []byte("release\n"), 0o600); err != nil {
		t.Fatalf("release first managed invocation: %v", err)
	}
	if err := waitForCommand(t, command, waitResult, 5*time.Second); err != nil {
		t.Fatalf("first managed invocation failed: %v", err)
	}
	finished = true
	if firstStderr.Len() != 0 {
		t.Fatalf("first managed invocation stderr = %q", firstStderr.String())
	}
	waitManagedGatewayUnavailable(t, firstEndpoint, 3*time.Second)
	waitManagedGatewayUnavailable(t, isolated.GatewayURL, 3*time.Second)
}

func TestBuildoptManagedGatewayConcurrentCacheBindingsRemainIsolated(
	t *testing.T,
) {
	buildoptBinary := buildBuildopt(t)
	stateRoot := filepath.Join(t.TempDir(), "runtime")
	slotA := filepath.Join(stateRoot, "slots", "slot-a")
	slotB := filepath.Join(stateRoot, "slots", "slot-b")
	for _, directory := range []string{slotA, slotB} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("create managed slot directory: %v", err)
		}
	}

	credentialA := bytes.Repeat([]byte{0xa1}, 32)
	credentialB := bytes.Repeat([]byte{0xb2}, 32)
	authorityA := "sha256:" + strings.Repeat("a", 64)
	authorityB := "sha256:" + strings.Repeat("b", 64)
	var hitsA atomic.Int32
	var hitsB atomic.Int32
	upstreamA := isolatedManagedUpstream(
		credentialA,
		authorityA,
		"namespace-a",
		&hitsA,
	)
	defer upstreamA.Close()
	upstreamB := isolatedManagedUpstream(
		credentialB,
		authorityB,
		"namespace-b",
		&hitsB,
	)
	defer upstreamB.Close()

	processA := startManagedGatewayCommand(
		t,
		buildoptBinary,
		slotA,
	)
	defer stopManagedGatewayCommand(t, processA)
	processB := startManagedGatewayCommand(
		t,
		buildoptBinary,
		slotB,
	)
	defer stopManagedGatewayCommand(t, processB)

	connectionA, gatewayA := registerManagedGatewayFixture(
		t,
		slotA,
		"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		upstreamA.URL,
		credentialA,
		authorityA,
	)
	defer connectionA.Close()
	connectionB, gatewayB := registerManagedGatewayFixture(
		t,
		slotB,
		"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
		upstreamB.URL,
		credentialB,
		authorityB,
	)
	defer connectionB.Close()

	statusA, bodyA := requestManagedCacheFixture(
		t,
		gatewayA,
		gatewayA.Password,
	)
	statusB, bodyB := requestManagedCacheFixture(
		t,
		gatewayB,
		gatewayB.Password,
	)
	if statusA != http.StatusOK || string(bodyA) != "namespace-a" ||
		statusB != http.StatusOK || string(bodyB) != "namespace-b" ||
		hitsA.Load() != 1 || hitsB.Load() != 1 {
		t.Fatalf(
			"isolated cache routes = %d/%q/%d and %d/%q/%d",
			statusA,
			bodyA,
			hitsA.Load(),
			statusB,
			bodyB,
			hitsB.Load(),
		)
	}
	crossStatus, crossBody := requestManagedCacheFixture(
		t,
		gatewayA,
		gatewayB.Password,
	)
	if crossStatus != http.StatusUnauthorized || len(crossBody) != 0 {
		t.Fatalf(
			"cross-slot local credential = %d/%q, want 401/empty",
			crossStatus,
			crossBody,
		)
	}

	for _, fixture := range []struct {
		directory  string
		upstream   string
		credential []byte
		authority  string
	}{
		{slotA, upstreamA.URL, credentialA, authorityA},
		{slotB, upstreamB.URL, credentialB, authorityB},
	} {
		state, err := os.ReadFile(
			filepath.Join(fixture.directory, "gateway-state.json"),
		)
		if err != nil {
			t.Fatalf("read managed gateway state: %v", err)
		}
		upstreamCredential := base64.RawURLEncoding.EncodeToString(
			fixture.credential,
		)
		for _, forbidden := range []string{
			fixture.upstream,
			upstreamCredential,
			fixture.authority,
			"namespace-a",
			"namespace-b",
		} {
			if bytes.Contains(state, []byte(forbidden)) {
				t.Fatalf(
					"managed state serialized cache binding %q: %s",
					forbidden,
					state,
				)
			}
		}
	}
}

func isolatedManagedUpstream(
	credential []byte,
	authorityDigest string,
	namespace string,
	hits *atomic.Int32,
) *httptest.Server {
	expectedAuthorization := "Bearer " +
		base64.RawURLEncoding.EncodeToString(credential)
	return httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Method != http.MethodGet ||
			request.Header.Get("Authorization") != expectedAuthorization ||
			request.Header.Get("X-BuildOpt-Authority-Digest") !=
				authorityDigest {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		hits.Add(1)
		_, _ = io.WriteString(writer, namespace)
	}))
}

func startManagedGatewayCommand(
	t *testing.T,
	buildoptBinary string,
	directory string,
) *managedGatewayCommandFixture {
	t.Helper()
	fixture := &managedGatewayCommandFixture{}
	fixture.command = exec.Command(
		buildoptBinary,
		"__managed-gateway",
		directory,
		"10s",
	)
	fixture.command.Stderr = &fixture.stderr
	if err := fixture.command.Start(); err != nil {
		t.Fatalf("start managed gateway command: %v", err)
	}
	return fixture
}

func stopManagedGatewayCommand(
	t *testing.T,
	fixture *managedGatewayCommandFixture,
) {
	t.Helper()
	command := fixture.command
	if command == nil || command.ProcessState != nil {
		return
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		_ = command.Process.Kill()
	}
	wait := make(chan error, 1)
	go func() {
		wait <- command.Wait()
	}()
	select {
	case err := <-wait:
		if err != nil {
			t.Errorf("managed gateway command exit: %v", err)
		}
	case <-time.After(3 * time.Second):
		_ = command.Process.Kill()
		<-wait
		t.Error("managed gateway command did not stop")
	}
	if fixture.stderr.Len() != 0 {
		t.Errorf("managed gateway stderr = %q", fixture.stderr.String())
	}
}

func registerManagedGatewayFixture(
	t *testing.T,
	directory string,
	attemptID string,
	upstream string,
	credential []byte,
	authorityDigest string,
) (*net.UnixConn, managedGatewayControlResponseFixture) {
	t.Helper()
	controlDigest := sha256.Sum256([]byte(directory))
	controlAddress := "@buildopt-gateway-" +
		hex.EncodeToString(controlDigest[:16])
	deadline := time.Now().Add(5 * time.Second)
	var connection *net.UnixConn
	var err error
	for {
		connection, err = net.DialUnix(
			"unix",
			nil,
			&net.UnixAddr{Name: controlAddress, Net: "unix"},
		)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect managed gateway control: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	request := managedGatewayControlRequestFixture{
		SchemaVersion: 1,
		Operation:     "register",
		AttemptID:     attemptID,
		Cache: managedGatewayCacheRegistrationFixture{
			UpstreamEndpoint: upstream,
			Credential: base64.RawURLEncoding.EncodeToString(
				credential,
			),
			AuthorityDigest: authorityDigest,
			AllowRead:       true,
			AllowWrite:      false,
			ExpiresAt: time.Now().
				Add(time.Hour).
				UTC().
				Format(time.RFC3339Nano),
		},
	}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		_ = connection.Close()
		t.Fatalf("register managed gateway fixture: %v", err)
	}
	var response managedGatewayControlResponseFixture
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		_ = connection.Close()
		t.Fatalf("decode managed gateway registration: %v", err)
	}
	if response.SchemaVersion != 1 ||
		response.Endpoint == "" ||
		response.Username != "buildopt" ||
		response.Password == "" ||
		response.Generation == "" ||
		response.Error != "" {
		_ = connection.Close()
		t.Fatalf("managed gateway registration = %+v", response)
	}
	return connection, response
}

func requestManagedCacheFixture(
	t *testing.T,
	gateway managedGatewayControlResponseFixture,
	password string,
) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(
		http.MethodGet,
		gateway.Endpoint+"/cache/concurrent-key",
		nil,
	)
	if err != nil {
		t.Fatalf("create managed cache request: %v", err)
	}
	request.SetBasicAuth(gateway.Username, password)
	response, err := gatewayTestClient().Do(request)
	if err != nil {
		t.Fatalf("request managed cache: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read managed cache response: %v", err)
	}
	return response.StatusCode, body
}

func runManagedBuildoptHelper(
	t *testing.T,
	buildoptBinary string,
	stateRoot string,
	slot string,
	idleTimeout string,
	exitCode int,
) helperObservation {
	t.Helper()
	helperBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve managed helper executable: %v", err)
	}
	command := exec.Command(
		buildoptBinary,
		"run",
		"--",
		helperBinary,
		"-test.run=^TestBuildoptChildHelper$",
		"--",
	)
	command.Env = append(
		managedGatewayTestEnvironment(stateRoot, slot, idleTimeout),
		helperModeEnvironment+"=1",
		helperExitEnvironment+"="+strconv.Itoa(exitCode),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if actual := processExitCode(t, err); actual != exitCode {
		t.Fatalf(
			"managed helper exit = %d, want %d; stderr=%q",
			actual,
			exitCode,
			stderr.String(),
		)
	}

	var observation helperObservation
	if err := json.Unmarshal(stdout.Bytes(), &observation); err != nil {
		t.Fatalf("decode managed helper observation %q: %v", stdout.String(), err)
	}
	if observation.GatewayURL == "" {
		if !strings.Contains(
			stderr.String(),
			"managed runner slot is already active",
		) {
			t.Fatalf("missing runner-slot fallback diagnostic: %q", stderr.String())
		}
	} else if stderr.Len() != 0 {
		t.Fatalf("managed helper stderr = %q", stderr.String())
	}
	return observation
}

func managedGatewayTestEnvironment(
	stateRoot string,
	slot string,
	idleTimeout string,
) []string {
	overrides := map[string]string{
		managedStateRootEnvironment:   stateRoot,
		managedRunnerSlotEnvironment:  slot,
		managedIdleTimeoutEnvironment: idleTimeout,
	}
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		environment = append(environment, entry)
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}

func managedGatewayStateForEndpoint(
	t *testing.T,
	stateRoot string,
	endpoint string,
) managedGatewayStateFixture {
	t.Helper()
	return stateForEndpoint(t, readManagedGatewayStates(t, stateRoot), endpoint)
}

func stateForEndpoint(
	t *testing.T,
	states []managedGatewayStateFixture,
	endpoint string,
) managedGatewayStateFixture {
	t.Helper()
	for _, state := range states {
		if "http://"+state.Address == endpoint {
			return state
		}
	}
	t.Fatalf("no managed state found for endpoint %s: %+v", endpoint, states)
	return managedGatewayStateFixture{}
}

func readManagedGatewayStates(
	t *testing.T,
	stateRoot string,
) []managedGatewayStateFixture {
	t.Helper()
	paths, err := filepath.Glob(
		filepath.Join(stateRoot, "slots", "*", "gateway-state.json"),
	)
	if err != nil {
		t.Fatalf("glob managed gateway states: %v", err)
	}
	states := make([]managedGatewayStateFixture, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read managed gateway state %s: %v", path, err)
		}
		var state managedGatewayStateFixture
		if err := json.Unmarshal(content, &state); err != nil {
			t.Fatalf("decode managed gateway state %s: %v", path, err)
		}
		states = append(states, state)
	}
	return states
}

func assertManagedGatewayStatePermissions(t *testing.T, stateRoot string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(stateRoot, "slots", "*"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("resolve managed slot directory: %v/%v", paths, err)
	}
	for _, directory := range []string{
		stateRoot,
		filepath.Join(stateRoot, "slots"),
		paths[0],
	} {
		info, err := os.Stat(directory)
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("managed directory %s mode = %v/%v, want 0700", directory, info, err)
		}
	}
	statePath := filepath.Join(paths[0], "gateway-state.json")
	info, err := os.Stat(statePath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("managed state mode = %v/%v, want 0600", info, err)
	}
}

func requestManagedGatewayState(
	state managedGatewayStateFixture,
	password string,
) (int, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		"http://"+state.Address+gatewayReadyPath,
		nil,
	)
	if err != nil {
		return 0, err
	}
	request.SetBasicAuth(state.Username, password)
	response, err := gatewayTestClient().Do(request)
	if err != nil {
		return 0, err
	}
	_ = response.Body.Close()
	return response.StatusCode, nil
}

func waitManagedGatewayStatus(
	t *testing.T,
	state managedGatewayStateFixture,
	expected int,
	timeout time.Duration,
) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		status, err := requestManagedGatewayState(state, state.Password)
		if err == nil && status == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"managed gateway status = %d/%v, want %d",
				status,
				err,
				expected,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitManagedGatewayUnavailable(
	t *testing.T,
	endpoint string,
	timeout time.Duration,
) {
	t.Helper()
	address := strings.TrimPrefix(endpoint, "http://")
	deadline := time.Now().Add(timeout)
	for {
		connection, err := net.DialTimeout("tcp4", address, 50*time.Millisecond)
		if err != nil {
			return
		}
		_ = connection.Close()
		if time.Now().After(deadline) {
			t.Fatalf("managed gateway %s remained reachable", endpoint)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitManagedGatewayReadyFile(
	t *testing.T,
	path string,
	timeout time.Duration,
) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		content, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(content))
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read managed gateway ready file: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for managed gateway ready file %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
