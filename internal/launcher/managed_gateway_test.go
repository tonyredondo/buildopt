//go:build linux || darwin

package launcher

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagedGatewayConfiguration(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "runtime")
	environment := map[string]string{
		managedGatewayStateRootEnvironment:   stateRoot,
		managedRunnerSlotEnvironment:         "runner-01.internal",
		managedGatewayIdleTimeoutEnvironment: "750ms",
	}
	config, configured, err := managedGatewayConfigFromEnvironment(
		func(key string) string {
			return environment[key]
		},
	)
	if err != nil {
		t.Fatalf("parse managed gateway configuration: %v", err)
	}
	if !configured ||
		config.stateRoot != stateRoot ||
		config.idleTimeout != 750*time.Millisecond ||
		!strings.HasPrefix(
			config.directory,
			filepath.Join(stateRoot, "slots")+string(filepath.Separator),
		) {
		t.Fatalf("unexpected managed gateway configuration: %+v", config)
	}
	if filepath.Base(config.directory) == environment[managedRunnerSlotEnvironment] {
		t.Fatal("managed runner-slot name was not tokenized in the state path")
	}

	empty, configured, err := managedGatewayConfigFromEnvironment(
		func(string) string { return "" },
	)
	if err != nil || configured || empty != (managedGatewayConfig{}) {
		t.Fatalf("empty configuration = %+v/%t/%v", empty, configured, err)
	}
}

func TestManagedGatewayRejectsIncompleteOrUnsafeConfiguration(t *testing.T) {
	testCases := []struct {
		name        string
		stateRoot   string
		slot        string
		idleTimeout string
	}{
		{
			name:      "missing slot",
			stateRoot: filepath.Join(t.TempDir(), "runtime"),
		},
		{
			name: "missing state root",
			slot: "runner-01",
		},
		{
			name:        "idle timeout without a slot",
			idleTimeout: "1s",
		},
		{
			name:      "relative state root",
			stateRoot: "relative/runtime",
			slot:      "runner-01",
		},
		{
			name:      "path-like slot",
			stateRoot: filepath.Join(t.TempDir(), "runtime"),
			slot:      "../runner",
		},
		{
			name:        "short idle timeout",
			stateRoot:   filepath.Join(t.TempDir(), "runtime"),
			slot:        "runner-01",
			idleTimeout: "99ms",
		},
		{
			name:        "long idle timeout",
			stateRoot:   filepath.Join(t.TempDir(), "runtime"),
			slot:        "runner-01",
			idleTimeout: "25h",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			environment := map[string]string{
				managedGatewayStateRootEnvironment:   testCase.stateRoot,
				managedRunnerSlotEnvironment:         testCase.slot,
				managedGatewayIdleTimeoutEnvironment: testCase.idleTimeout,
			}
			_, configured, err := managedGatewayConfigFromEnvironment(
				func(key string) string {
					return environment[key]
				},
			)
			if !configured || err == nil {
				t.Fatalf("unsafe configuration was accepted: %v", err)
			}
		})
	}
}

func TestManagedGatewayRequiresPrivateStateDirectories(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "runtime")
	if err := os.Mkdir(stateRoot, 0o755); err != nil {
		t.Fatalf("create public state root: %v", err)
	}
	config, configured, err := managedGatewayConfigFromEnvironment(
		func(key string) string {
			switch key {
			case managedGatewayStateRootEnvironment:
				return stateRoot
			case managedRunnerSlotEnvironment:
				return "runner-01"
			default:
				return ""
			}
		},
	)
	if err != nil || !configured {
		t.Fatalf("parse state root fixture: %v", err)
	}
	if err := prepareManagedGatewayDirectories(config); err == nil ||
		!strings.Contains(err.Error(), "mode 0700") {
		t.Fatalf("public state root error = %v", err)
	}
}

func TestManagedGatewayContextAllowsOneCurrentInvocation(t *testing.T) {
	context := &managedGatewayContext{}
	if context.ready() {
		t.Fatal("new managed gateway context is ready")
	}
	if !context.register("first") || !context.ready() {
		t.Fatal("first managed gateway context was not registered")
	}
	if context.register("second") {
		t.Fatal("second managed gateway context replaced the active invocation")
	}
	context.unregister("second")
	if !context.ready() {
		t.Fatal("mismatched unregister removed the active invocation")
	}
	context.unregister("first")
	if context.ready() {
		t.Fatal("managed gateway context remained ready after release")
	}
}

func TestManagedGatewayContextCarriesOnlyTheActiveCacheBinding(t *testing.T) {
	binding, err := newGatewayCacheBinding(
		"http://127.0.0.1:8042",
		bytes.Repeat([]byte{0x31}, 32),
		"sha256:"+strings.Repeat("d", 64),
		"11111111-1111-4111-8111-111111111111",
		true,
		false,
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("create cache binding: %v", err)
	}
	registration := managedGatewayCacheForRegistration(binding)
	decoded, err := registration.binding(binding.attemptID)
	if err != nil {
		t.Fatalf("decode cache registration: %v", err)
	}

	context := &managedGatewayContext{}
	if !context.registerWithCache(binding.attemptID, decoded) {
		t.Fatal("register cache binding")
	}
	current := context.cache()
	if current == nil ||
		current.authorityDigest != binding.authorityDigest ||
		current.credential != binding.credential {
		t.Fatalf("current cache binding = %+v", current)
	}
	current.credential = "mutated"
	if context.cache().credential != binding.credential {
		t.Fatal("managed gateway exposed mutable cache context")
	}
	context.unregister(binding.attemptID)
	if context.cache() != nil {
		t.Fatal("managed gateway retained cache context after EOF")
	}
}

func TestManagedGatewayStateIsPrivateStrictAndRoundTrips(t *testing.T) {
	directory := t.TempDir()
	identity, err := newGatewayIdentity("127.0.0.1:43210")
	if err != nil {
		t.Fatalf("create managed gateway identity: %v", err)
	}
	if err := writeManagedGatewayState(directory, identity); err != nil {
		t.Fatalf("write managed gateway state: %v", err)
	}
	state, err := readManagedGatewayState(directory)
	if err != nil {
		t.Fatalf("read managed gateway state: %v", err)
	}
	if state.Address != identity.address ||
		state.Username != identity.username ||
		state.Password != identity.password ||
		state.Generation != identity.generation {
		t.Fatalf("managed gateway state changed identity: %+v", state)
	}

	path := filepath.Join(directory, "gateway-state.json")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("make managed gateway state public: %v", err)
	}
	if _, err := readManagedGatewayState(directory); err == nil ||
		!strings.Contains(err.Error(), "private bounded regular file") {
		t.Fatalf("public managed gateway state error = %v", err)
	}

	if err := os.WriteFile(
		path,
		[]byte(`{"schemaVersion":1,"unknown":true}`+"\n"),
		0o600,
	); err != nil {
		t.Fatalf("write malformed managed gateway state: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("restore managed gateway state mode: %v", err)
	}
	if _, err := readManagedGatewayState(directory); err == nil {
		t.Fatal("managed gateway state accepted unknown fields")
	}
}
