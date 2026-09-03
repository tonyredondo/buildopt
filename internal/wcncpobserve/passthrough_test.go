package wcncpobserve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestPassthroughPreservesNativeBehavior(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX binary fixtures; Windows shapes covered statically")
	}
	ctx := context.Background()
	// Successful passthrough preserves stdout and exit zero.
	stdout, stderr, result := CaptureForTest(ctx, "/bin/echo", []string{"hello", "world"}, t.TempDir(), "")
	if stdout != "hello world\n" || stderr != "" || result.Child.Outcome != "SUCCESS" || *result.Child.ExitCode != 0 {
		t.Fatalf("echo = %q/%q/%+v", stdout, stderr, result.Child)
	}
	// Non-zero Gradle exit is preserved, not replaced.
	_, _, failed := CaptureForTest(ctx, "/bin/false", nil, t.TempDir(), "")
	if failed.Child.Outcome != "FAILED" || *failed.Child.ExitCode != 1 {
		t.Fatalf("false = %+v", failed.Child)
	}
	// stdin and cwd flow through unchanged.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, _, _ = CaptureForTest(ctx, "/bin/cat", nil, dir, "streamed")
	if stdout != "streamed" {
		t.Fatalf("stdin = %q", stdout)
	}
	// Missing wrapper surfaces as a failed start without inventing success.
	_, _, missing := CaptureForTest(ctx, filepath.Join(t.TempDir(), "no-wrapper"), nil, t.TempDir(), "")
	if missing.Child.Outcome != "FAILED" {
		t.Fatalf("missing wrapper = %+v", missing.Child)
	}
	// Cancellation classifies without replacing the authoritative outcome.
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, cancelled := CaptureForTest(cancelledCtx, "/bin/sleep", []string{"30"}, t.TempDir(), "")
	if cancelled.Child.Outcome != "FAILED" && cancelled.Child.Outcome != "SIGNALED" {
		t.Fatalf("cancelled = %+v", cancelled.Child)
	}
	if BypassRequested(map[string]string{"BUILDOPT_BYPASS": "1"}) != true {
		t.Fatal("bypass not detected")
	}
	if BypassRequested(map[string]string{}) != false {
		t.Fatal("bypass false positive")
	}
}

func TestUploadFailsOpenAndRejectsCorruptBackend(t *testing.T) {
	t.Parallel()
	item := json.RawMessage(`{"synthetic":true}`)
	newServer := func(handler http.HandlerFunc) (*httptest.Server, *http.Client) {
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		return server, server.Client()
	}
	// Successful upload reports exact count.
	okServer, okClient := newServer(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"published":1}`))
	})
	outcome := UploadBatch(context.Background(), okClient, okServer.URL, "token", []json.RawMessage{item}, 100*time.Millisecond)
	if !outcome.Attempted || outcome.Uploaded != 1 || outcome.Queued {
		t.Fatalf("ok upload = %+v", outcome)
	}
	// Backend outage queues locally and preserves the native result.
	unavailable := UploadBatch(context.Background(), okClient, "http://127.0.0.1:1/no-listener", "token", []json.RawMessage{item}, 50*time.Millisecond)
	if !unavailable.Queued || unavailable.Uploaded != 0 {
		t.Fatalf("outage = %+v", unavailable)
	}
	// Slow backend hits the post-child deadline and stays queued.
	slowServer, slowClient := newServer(func(response http.ResponseWriter, request *http.Request) {
		time.Sleep(500 * time.Millisecond)
		response.WriteHeader(http.StatusCreated)
	})
	slow := UploadBatch(context.Background(), slowClient, slowServer.URL, "token", []json.RawMessage{item}, 50*time.Millisecond)
	if !slow.Queued {
		t.Fatalf("slow backend = %+v", slow)
	}
	// Corrupt/rejecting backend fails closed without claiming success.
	rejectServer, rejectClient := newServer(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte(`not-json`))
	})
	rejected := UploadBatch(context.Background(), rejectClient, rejectServer.URL, "token", []json.RawMessage{item}, 100*time.Millisecond)
	if rejected.Uploaded != 0 || !rejected.Queued {
		t.Fatalf("corrupt backend = %+v", rejected)
	}
	wrongAckServer, wrongAckClient := newServer(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"published":0}`))
	})
	wrongAck := UploadBatch(context.Background(), wrongAckClient, wrongAckServer.URL, "token", []json.RawMessage{item}, 100*time.Millisecond)
	if wrongAck.Uploaded != 0 || !wrongAck.Queued || wrongAck.Reason != "invalid-acknowledgement" {
		t.Fatalf("wrong acknowledgement = %+v", wrongAck)
	}
}

func TestPassthroughReportsPOSIXSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX signal semantics")
	}
	_, _, result := CaptureForTest(context.Background(), "/bin/sh", []string{"-c", "kill -TERM $$"}, t.TempDir(), "")
	if result.Child.Outcome != "SIGNALED" || result.Child.Signal == nil || *result.Child.Signal != 15 {
		t.Fatalf("signal result = %+v", result.Child)
	}
}
