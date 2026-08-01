package edgecache

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

type addressOnlyListener struct {
	address net.Addr
}

func (listener addressOnlyListener) Accept() (net.Conn, error) {
	panic("non-loopback listener reached Accept")
}

func (listener addressOnlyListener) Close() error {
	return nil
}

func (listener addressOnlyListener) Addr() net.Addr {
	return listener.address
}

func TestProxyRejectsUnsafeListenerAndInvalidHTTPUse(t *testing.T) {
	config := testStoreConfig(
		filepath.Join(t.TempDir(), "edge"),
		"http://127.0.0.1:8042",
	)
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client, err := NewSharedClient(
		config.Shared,
		[]byte("edge-token"),
		&http.Client{},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	read := testReadAuthority()
	proxy, err := NewProxy(
		store,
		client,
		read,
		nil,
		func() time.Time { return edgeTestNow },
	)
	if err != nil {
		t.Fatal(err)
	}
	unsafe := addressOnlyListener{address: &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 8043,
	}}
	if err := proxy.Serve(context.Background(), unsafe); err == nil {
		t.Fatal("non-loopback listener was accepted")
	}

	put := httptest.NewRequest(
		http.MethodPut,
		"/cache/key",
		io.NopCloser(pendingPanicReader{}),
	)
	put.ContentLength = 1
	putResponse := httptest.NewRecorder()
	proxy.Handler().ServeHTTP(putResponse, put)
	if putResponse.Code != http.StatusForbidden {
		t.Fatalf("read-only PUT = %d", putResponse.Code)
	}
	postResponse := httptest.NewRecorder()
	proxy.Handler().ServeHTTP(
		postResponse,
		httptest.NewRequest(http.MethodPost, "/cache/key", nil),
	)
	if postResponse.Code != http.StatusMethodNotAllowed ||
		postResponse.Header().Get("Allow") != "GET, PUT" {
		t.Fatalf("unsupported method = %d/%v", postResponse.Code, postResponse.Header())
	}
	pathResponse := httptest.NewRecorder()
	proxy.Handler().ServeHTTP(
		pathResponse,
		httptest.NewRequest(http.MethodGet, "/cache/key/extra", nil),
	)
	if pathResponse.Code != http.StatusNotFound {
		t.Fatalf("unsafe path = %d", pathResponse.Code)
	}

	write := testWriteAuthority()
	writeProxy, err := NewProxy(
		store,
		client,
		read,
		&write,
		func() time.Time { return edgeTestNow },
	)
	if err != nil {
		t.Fatal(err)
	}
	unknownLength := httptest.NewRequest(
		http.MethodPut,
		"/cache/key",
		io.NopCloser(pendingPanicReader{}),
	)
	unknownLength.ContentLength = -1
	lengthResponse := httptest.NewRecorder()
	writeProxy.Handler().ServeHTTP(lengthResponse, unknownLength)
	if lengthResponse.Code != http.StatusLengthRequired {
		t.Fatalf("unknown-length PUT = %d", lengthResponse.Code)
	}
}
