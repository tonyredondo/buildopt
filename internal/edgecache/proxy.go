package edgecache

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const edgeStateHeader = "X-BuildOpt-Edge-State"

// Proxy exposes one already-authenticated authority through a loopback-only
// Gradle-compatible HTTP cache route.
type Proxy struct {
	store          *Store
	client         *SharedClient
	readAuthority  ReadAuthority
	writeAuthority *WriteAuthority
	clock          func() time.Time
}

func NewProxy(
	store *Store,
	client *SharedClient,
	readAuthority ReadAuthority,
	writeAuthority *WriteAuthority,
	clock func() time.Time,
) (*Proxy, error) {
	if store == nil || client == nil || clock == nil {
		return nil, errors.New("create Edge proxy: invalid dependency")
	}
	now := clock().UTC()
	if !readAuthority.current(now) ||
		(writeAuthority != nil && !writeAuthority.current(now)) {
		return nil, errors.New("create Edge proxy: authority is not current")
	}
	proxy := &Proxy{
		store:         store,
		client:        client,
		readAuthority: readAuthority,
		clock:         clock,
	}
	if writeAuthority != nil {
		copy := *writeAuthority
		proxy.writeAuthority = &copy
	}
	return proxy, nil
}

// Handler returns the exact /cache/{key} GET/PUT route. It carries no generic
// filesystem or administrative endpoints.
func (proxy *Proxy) Handler() http.Handler {
	return http.HandlerFunc(proxy.serveHTTP)
}

func (proxy *Proxy) serveHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Cache-Control", "no-store")
	key, ok := edgeCacheKey(request.URL.Path)
	if !ok {
		writeEdgeStatus(response, http.StatusNotFound)
		return
	}
	switch request.Method {
	case http.MethodGet:
		proxy.serveGET(response, request, key)
	case http.MethodPut:
		proxy.servePUT(response, request, key)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeEdgeStatus(response, http.StatusMethodNotAllowed)
	}
}

func (proxy *Proxy) serveGET(
	response http.ResponseWriter,
	request *http.Request,
	key string,
) {
	now := proxy.clock().UTC()
	file, err := proxy.store.ReadThrough(
		request.Context(),
		proxy.readAuthority,
		proxy.client,
		key,
		now,
	)
	state := "COMMITTED"
	if err != nil && proxy.writeAuthority != nil &&
		(errors.Is(err, ErrCacheMiss) ||
			errors.Is(err, ErrUpstreamUnavailable)) {
		file, err = proxy.store.OpenPending(
			request.Context(),
			*proxy.writeAuthority,
			key,
			now,
		)
		state = "PENDING_ATTEMPT"
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrCacheMiss):
			writeEdgeStatus(response, http.StatusNotFound)
		case errors.Is(err, ErrUpstreamRejected):
			writeEdgeStatus(response, http.StatusUnauthorized)
		default:
			writeEdgeStatus(response, http.StatusServiceUnavailable)
		}
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		writeEdgeStatus(response, http.StatusServiceUnavailable)
		return
	}
	response.Header().Set("Content-Type", "application/octet-stream")
	response.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	response.Header().Set(edgeStateHeader, state)
	response.WriteHeader(http.StatusOK)
	_, _ = io.Copy(response, file)
}

func (proxy *Proxy) servePUT(
	response http.ResponseWriter,
	request *http.Request,
	key string,
) {
	if proxy.writeAuthority == nil {
		writeEdgeStatus(response, http.StatusForbidden)
		return
	}
	if request.ContentLength < 0 {
		writeEdgeStatus(response, http.StatusLengthRequired)
		return
	}
	if request.ContentLength > proxy.store.maximumObjectBytes {
		writeEdgeStatus(response, http.StatusRequestEntityTooLarge)
		return
	}
	result, err := proxy.store.PutPendingDurable(
		request.Context(),
		*proxy.writeAuthority,
		key,
		request.ContentLength,
		request.Body,
		proxy.clock().UTC(),
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrCapacityExceeded):
			writeEdgeStatus(response, http.StatusRequestEntityTooLarge)
		case errors.Is(err, ErrPendingConflict):
			writeEdgeStatus(response, http.StatusConflict)
		case errors.Is(err, context.Canceled),
			errors.Is(err, context.DeadlineExceeded):
			writeEdgeStatus(response, http.StatusRequestTimeout)
		default:
			writeEdgeStatus(response, http.StatusServiceUnavailable)
		}
		return
	}
	response.Header().Set("X-BuildOpt-Blob-Digest", result.Digest)
	if result.Added {
		writeEdgeStatus(response, http.StatusCreated)
		return
	}
	writeEdgeStatus(response, http.StatusOK)
}

// Serve runs the proxy only on an explicitly provided IPv4 loopback TCP
// listener and shuts it down when ctx is canceled.
func (proxy *Proxy) Serve(ctx context.Context, listener net.Listener) error {
	if proxy == nil || ctx == nil || listener == nil {
		return errors.New("serve Edge proxy: invalid input")
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.IP == nil ||
		!address.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		return errors.New("serve Edge proxy: listener is not IPv4 loopback")
	}
	server := &http.Server{
		Handler:           proxy.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdownContext, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			_ = server.Shutdown(shutdownContext)
			cancel()
		case <-done:
		}
	}()
	err := server.Serve(listener)
	close(done)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func edgeCacheKey(path string) (string, bool) {
	const prefix = "/cache/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	key := strings.TrimPrefix(path, prefix)
	return key, !strings.Contains(key, "/") && validCacheKey(key)
}

func writeEdgeStatus(response http.ResponseWriter, status int) {
	response.Header().Set("Content-Length", "0")
	response.WriteHeader(status)
}
