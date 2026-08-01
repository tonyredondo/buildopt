//go:build !linux

package launcher

import (
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPortableHandshakeUsesAuthenticatedLoopbackTCP(t *testing.T) {
	server, err := startPluginHandshake()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(server.endpoint, "tcp://127.0.0.1:") {
		t.Fatalf("portable endpoint = %q", server.endpoint)
	}
	endpoint, err := url.Parse(server.endpoint)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := net.Dial("tcp4", endpoint.Host)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Write(make([]byte, len(pluginAuthMagic)+pluginTokenBytes)); err != nil {
		t.Fatal(err)
	}
	// Wait for the server to reject the complete preface before finish closes
	// the selected connection during cleanup.
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var response [1]byte
	if _, err := connection.Read(response[:]); err == nil {
		t.Fatal("unauthenticated portable connection remained open")
	}
	_ = connection.Close()
	result := server.finish()
	if !result.connected || result.err == nil || !strings.Contains(result.err.Error(), "credential") {
		t.Fatalf("unauthenticated portable result = %+v, err = %v", result, result.err)
	}
}
