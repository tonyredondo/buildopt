package launcher

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagedGatewayCircuitStateIsPrivateStrictAndRecovers(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	breaker := newManagedGatewayCircuitBreaker(directory)
	breaker.now = func() time.Time { return now }
	breaker.cooldown = time.Minute

	if err := breaker.trip(gatewayCircuitDiskPressure); err != nil {
		t.Fatalf("open managed gateway circuit: %v", err)
	}
	state, err := readManagedGatewayCircuitState(directory)
	if err != nil {
		t.Fatalf("read managed gateway circuit: %v", err)
	}
	if state.Reason != gatewayCircuitDiskPressure ||
		state.OpenedAt != now.Format(time.RFC3339Nano) ||
		state.RetryAfter != now.Add(time.Minute).Format(time.RFC3339Nano) ||
		!breaker.cacheSuppressed() {
		t.Fatalf("unexpected managed gateway circuit state: %+v", state)
	}
	restarted := newManagedGatewayCircuitBreaker(directory)
	restarted.now = func() time.Time { return now }
	if !restarted.cacheSuppressed() {
		t.Fatal("managed gateway restart lost the durable open circuit")
	}
	path := filepath.Join(directory, managedGatewayCircuitStateFile)
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("managed gateway circuit mode = %v/%v", info, err)
	}

	now = now.Add(time.Minute + time.Nanosecond)
	if restarted.cacheSuppressed() {
		t.Fatal("expired managed gateway circuit still suppresses cache")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired managed gateway circuit state remains: %v", err)
	}

	if err := os.WriteFile(
		path,
		[]byte(`{"schemaVersion":1,"reason":"FLOOD","openedAt":"2026-07-31T13:00:00Z","retryAfter":"2026-07-31T13:01:00Z","unknown":true}`),
		0o600,
	); err != nil {
		t.Fatalf("write malformed managed gateway circuit: %v", err)
	}
	if !breaker.cacheSuppressed() {
		t.Fatal("malformed managed gateway circuit failed open")
	}
}

func TestManagedGatewayCircuitSuppressesL2BetweenInvocations(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, time.July, 31, 13, 0, 0, 0, time.UTC)
	breaker := newManagedGatewayCircuitBreaker(directory)
	breaker.now = func() time.Time { return now }
	breaker.cooldown = time.Minute
	context := &managedGatewayContext{circuit: breaker}
	binding := gatewayCircuitBindingForTest(t)

	registered, suppressed := context.registerWithCacheStatus(
		binding.attemptID,
		binding,
	)
	if !registered || suppressed || context.cache() == nil {
		t.Fatal("healthy managed gateway registration suppressed L2")
	}
	context.tripCircuit(gatewayCircuitFlood)
	if context.cache() != nil {
		t.Fatal("opened circuit retained the active L2 binding")
	}
	context.unregister(binding.attemptID)

	registered, suppressed = context.registerWithCacheStatus(
		binding.attemptID,
		binding,
	)
	if !registered || !suppressed || context.cache() != nil {
		t.Fatal("next invocation did not suppress L2 while circuit was open")
	}
	context.unregister(binding.attemptID)

	now = now.Add(time.Minute + time.Nanosecond)
	registered, suppressed = context.registerWithCacheStatus(
		binding.attemptID,
		binding,
	)
	if !registered || suppressed || context.cache() == nil {
		t.Fatal("managed gateway did not recover L2 after circuit cooldown")
	}
}

func TestManagedGatewayCircuitOmissionSelectsWritableL1(t *testing.T) {
	authority := &localAuthorityContext{
		managedL1Config: managedL1Config{l2WriteAuthorized: true},
	}
	healthy := &localGateway{}
	if !managedSharedAuthorityEnabled(authority, healthy) ||
		!managedL1ConfigForInvocation(authority, healthy).l2WriteAuthorized {
		t.Fatal("healthy invocation did not retain its authorized L2 writer")
	}

	openCircuit := &localGateway{cacheSuppressed: true}
	if managedSharedAuthorityEnabled(authority, openCircuit) ||
		managedL1ConfigForInvocation(
			authority,
			openCircuit,
		).l2WriteAuthorized {
		t.Fatal("open circuit did not omit Shared and select writable L1")
	}
}

func gatewayCircuitBindingForTest(t *testing.T) *gatewayCacheBinding {
	t.Helper()
	binding, err := newGatewayCacheBinding(
		"http://127.0.0.1:8042",
		bytes.Repeat([]byte{0x42}, 32),
		"sha256:"+strings.Repeat("a", 64),
		"11111111-1111-4111-8111-111111111111",
		true,
		true,
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("create circuit cache binding: %v", err)
	}
	return binding
}
