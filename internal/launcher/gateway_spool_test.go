package launcher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestLocalGatewayVerifiesCompletePayloadBeforeOK(t *testing.T) {
	content := []byte("complete verified cache payload")
	digest := "sha256:" + sha256Hex(content)
	firstChunkWritten := make(chan struct{})
	releaseLastChunk := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("ETag", `"`+digest+`"`)
		writer.Header().Set("Content-Type", "application/octet-stream")
		_, _ = writer.Write(content[:8])
		writer.(http.Flusher).Flush()
		close(firstChunkWritten)
		<-releaseLastChunk
		_, _ = writer.Write(content[8:])
	}))
	defer upstream.Close()
	gateway := startCacheGatewayForTest(t, upstream.URL)
	defer gateway.close()

	type result struct {
		status  int
		headers http.Header
		body    []byte
		err     error
	}
	done := make(chan result, 1)
	go func() {
		status, headers, body, err := requestLocalGateway(
			gateway.endpoint,
			gateway.username,
			gateway.password,
			http.MethodGet,
			"/cache/verified",
		)
		done <- result{
			status:  status,
			headers: headers,
			body:    body,
			err:     err,
		}
	}()
	<-firstChunkWritten
	select {
	case early := <-done:
		t.Fatalf("gateway responded before complete verification: %+v", early)
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseLastChunk)
	actual := <-done
	if actual.err != nil ||
		actual.status != http.StatusOK ||
		!bytes.Equal(actual.body, content) ||
		actual.headers.Get("ETag") != `"`+digest+`"` ||
		actual.headers.Get("X-BuildOpt-Blob-Digest") != digest {
		t.Fatalf("verified gateway response = %+v", actual)
	}
	assertGatewaySpoolEmpty(t, gateway.spool)
}

func TestLocalGatewaySpoolFullDiskBecomesByteFreeMiss(t *testing.T) {
	content := []byte("payload that cannot reach disk")
	digest := "sha256:" + sha256Hex(content)
	upstream := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("ETag", `"`+digest+`"`)
		_, _ = writer.Write(content)
	}))
	defer upstream.Close()
	gateway := startCacheGatewayForTest(t, upstream.URL)
	originalSpool := gateway.spool
	spool := openGatewaySpoolForTest(t, 64, 128)
	spool.testWrite = func(_ *os.File, _ []byte) (int, error) {
		return 0, syscall.ENOSPC
	}
	gateway.spool = spool
	if err := originalSpool.close(); err != nil {
		t.Fatal(err)
	}
	defer gateway.close()

	status, _, body, err := requestLocalGateway(
		gateway.endpoint,
		gateway.username,
		gateway.password,
		http.MethodGet,
		"/cache/full-disk",
	)
	if err != nil || status != http.StatusNotFound || len(body) != 0 {
		t.Fatalf("full disk response = %d/%q/%v", status, body, err)
	}
	assertGatewaySpoolReleased(t, spool)
	assertGatewaySpoolEmpty(t, spool)
}

func TestGatewaySpoolReservesConcurrentPayloadsAtomically(t *testing.T) {
	spool := openGatewaySpoolForTest(t, 8, 10)
	reserved := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	spool.testAfterReserve = func() {
		once.Do(func() {
			close(reserved)
			<-release
		})
	}
	firstContent := []byte("123456")
	firstDigest := "sha256:" + sha256Hex(firstContent)
	type receiveResult struct {
		payload verifiedGatewayPayload
		err     error
	}
	firstDone := make(chan receiveResult, 1)
	go func() {
		payload, err := spool.receive(
			context.Background(),
			bytes.NewReader(firstContent),
			int64(len(firstContent)),
			`"`+firstDigest+`"`,
			"",
		)
		firstDone <- receiveResult{payload: payload, err: err}
	}()
	<-reserved

	secondContent := []byte("abcdef")
	secondDigest := "sha256:" + sha256Hex(secondContent)
	if payload, err := spool.receive(
		context.Background(),
		&gatewayPanicReader{},
		int64(len(secondContent)),
		`"`+secondDigest+`"`,
		"",
	); err == nil || payload.file != nil {
		if payload.file != nil {
			_ = payload.close()
		}
		t.Fatalf("concurrent over-reservation = %+v/%v", payload, err)
	}
	close(release)
	first := <-firstDone
	if first.err != nil || first.payload.file == nil {
		t.Fatalf("first reserved payload = %+v/%v", first.payload, first.err)
	}
	spool.mutex.Lock()
	if spool.reservedBytes != int64(len(firstContent)) {
		spool.mutex.Unlock()
		t.Fatalf(
			"verified open payload reservation = %d, want %d",
			spool.reservedBytes,
			len(firstContent),
		)
	}
	spool.mutex.Unlock()
	if err := first.payload.close(); err != nil {
		t.Fatal(err)
	}
	assertGatewaySpoolReleased(t, spool)
	assertGatewaySpoolEmpty(t, spool)
}

func TestGatewaySpoolCoalescesConcurrentDirectorySync(t *testing.T) {
	const requests = 32
	spool := openGatewaySpoolForTest(t, 64, 128)
	joined := make(chan struct{}, requests)
	releaseLeader := make(chan struct{})
	spool.testAfterDirectorySyncJoin = func(leader bool) {
		joined <- struct{}{}
		if leader {
			<-releaseLeader
		}
	}
	var callsMutex sync.Mutex
	calls := 0
	spool.testSyncDirectory = func(string) error {
		callsMutex.Lock()
		calls++
		callsMutex.Unlock()
		return nil
	}

	start := make(chan struct{})
	results := make(chan error, requests)
	for range requests {
		go func() {
			<-start
			results <- spool.syncUnlinkedDirectory()
		}()
	}
	close(start)
	for range requests {
		select {
		case <-joined:
		case <-time.After(2 * time.Second):
			close(releaseLeader)
			t.Fatal("concurrent directory sync did not join its batch")
		}
	}
	close(releaseLeader)
	for range requests {
		if err := <-results; err != nil {
			t.Fatalf("coalesced directory sync: %v", err)
		}
	}
	callsMutex.Lock()
	defer callsMutex.Unlock()
	if calls != 1 {
		t.Fatalf("directory sync calls = %d, want 1", calls)
	}
}

func TestGatewaySpoolCancellationDeletesPartialPayload(t *testing.T) {
	spool := openGatewaySpoolForTest(t, 64, 128)
	ctx, cancel := context.WithCancel(context.Background())
	reader := &gatewayCancellationReader{
		ctx:       ctx,
		content:   []byte("partial"),
		delivered: make(chan struct{}),
	}
	digest := "sha256:" + sha256Hex([]byte("complete"))
	done := make(chan error, 1)
	go func() {
		payload, err := spool.receive(
			ctx,
			reader,
			-1,
			`"`+digest+`"`,
			"",
		)
		if payload.file != nil {
			_ = payload.close()
		}
		done <- err
	}()
	<-reader.delivered
	cancel()
	if err := <-done; err == nil ||
		!errors.Is(err, errGatewaySpoolUnavailable) {
		t.Fatalf("cancelled receive error = %v", err)
	}
	assertGatewaySpoolReleased(t, spool)
	assertGatewaySpoolEmpty(t, spool)
}

func TestLocalGatewayRejectsLateChecksumAsByteFreeMiss(t *testing.T) {
	content := []byte("complete but corrupted")
	wrongDigest := "sha256:" + sha256Hex([]byte("expected bytes"))
	upstream := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("ETag", `"`+wrongDigest+`"`)
		_, _ = writer.Write(content)
	}))
	defer upstream.Close()
	gateway := startCacheGatewayForTest(t, upstream.URL)
	defer gateway.close()

	status, _, body, err := requestLocalGateway(
		gateway.endpoint,
		gateway.username,
		gateway.password,
		http.MethodGet,
		"/cache/late-checksum",
	)
	if err != nil || status != http.StatusNotFound || len(body) != 0 {
		t.Fatalf("late checksum response = %d/%q/%v", status, body, err)
	}
	assertGatewaySpoolReleased(t, gateway.spool)
	assertGatewaySpoolEmpty(t, gateway.spool)
}

func TestGatewaySpoolStartupCleansCrashedPayload(t *testing.T) {
	root := filepath.Join(t.TempDir(), "spool")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(root, gatewaySpoolFilePrefix+"crashed")
	if err := os.WriteFile(stale, []byte("partial secret payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	spool, err := openGatewaySpool(root, false, 64, 128)
	if err != nil {
		t.Fatalf("recover crashed spool: %v", err)
	}
	defer spool.close()
	assertGatewaySpoolEmpty(t, spool)
}

type gatewayCancellationReader struct {
	ctx       context.Context
	content   []byte
	delivered chan struct{}
	once      sync.Once
}

type gatewayPanicReader struct{}

func (*gatewayPanicReader) Read([]byte) (int, error) {
	panic("gateway read a request rejected by spool reservation")
}

func (reader *gatewayCancellationReader) Read(buffer []byte) (int, error) {
	if len(reader.content) > 0 {
		content := reader.content
		reader.content = nil
		count := copy(buffer, content)
		reader.once.Do(func() {
			if reader.delivered != nil {
				close(reader.delivered)
			}
		})
		return count, nil
	}
	<-reader.ctx.Done()
	return 0, reader.ctx.Err()
}

func openGatewaySpoolForTest(
	t *testing.T,
	maximumBytes int64,
	quotaBytes int64,
) *gatewaySpool {
	t.Helper()
	spool, err := openGatewaySpool(
		filepath.Join(t.TempDir(), "spool"),
		false,
		maximumBytes,
		quotaBytes,
	)
	if err != nil {
		t.Fatalf("open test gateway spool: %v", err)
	}
	t.Cleanup(func() {
		if err := spool.close(); err != nil {
			t.Errorf("close test gateway spool: %v", err)
		}
	})
	return spool
}

func assertGatewaySpoolReleased(t *testing.T, spool *gatewaySpool) {
	t.Helper()
	spool.mutex.Lock()
	defer spool.mutex.Unlock()
	if spool.reservedBytes != 0 {
		t.Fatalf("gateway spool reserved bytes = %d, want 0", spool.reservedBytes)
	}
}

func assertGatewaySpoolEmpty(t *testing.T, spool *gatewaySpool) {
	t.Helper()
	entries, err := os.ReadDir(spool.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("gateway spool entries = %v, want empty", entries)
	}
}

var _ io.Reader = (*gatewayCancellationReader)(nil)
