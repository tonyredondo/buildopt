//go:build windows

package launcher

import (
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsManagedControlUsesCredentialAndReplacesState(t *testing.T) {
	directory := t.TempDir()
	first, firstCredential, err := listenManagedGatewayControl(directory)
	if err != nil {
		t.Fatal(err)
	}
	_ = first.Close()
	second, secondCredential, err := listenManagedGatewayControl(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if firstCredential == secondCredential {
		t.Fatal("managed control credential was reused")
	}
	client, err := net.Dial("tcp4", second.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := second.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if err := verifyManagedGatewayControlPeer(server, secondCredential, secondCredential); err != nil {
		t.Fatal(err)
	}
	if err := verifyManagedGatewayControlPeer(server, secondCredential, firstCredential); err == nil {
		t.Fatal("stale managed control credential was accepted")
	}
}

func TestWindowsManagedL1AndGradleBootstrapAreAvailable(t *testing.T) {
	config, configured, err := managedL1ConfigFromEnvironment(func(key string) string {
		values := map[string]string{
			managedL1StateRootEnvironment:     filepath.Join(t.TempDir(), "state"),
			managedL1TenantEnvironment:        "owner",
			managedL1RepositoryEnvironment:    "buildopt",
			managedL1TrustDomainEnvironment:   "native-ci",
			managedL1CompatibilityEnvironment: "gradle-9.6.1-jdk-21-windows-amd64",
			managedL1GenerationEnvironment:    "1",
		}
		return values[key]
	})
	if err != nil || !configured || config.securityGeneration != 1 {
		t.Fatalf("managed L1 config = %+v/%t/%v", config, configured, err)
	}
	_, configured, err = startInvocationGradleBootstrapCache(nil, nil, func(key string) string {
		if key == gradleBootstrapConfigPathEnvironment {
			return filepath.Join(t.TempDir(), "bootstrap.json")
		}
		return ""
	})
	if !configured || err == nil || !strings.Contains(err.Error(), "requires signed DEPENDENCY_CACHE authority") ||
		errors.Is(err, errGradleBootstrapBusy) {
		t.Fatalf("managed Gradle bootstrap availability = %t/%v", configured, err)
	}
}
