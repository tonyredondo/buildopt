package edgecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var edgeTestNow = time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)

func testReadAuthority() ReadAuthority {
	return ReadAuthority{
		tenant:               "tenant-test",
		repository:           "tonyredondo/buildopt",
		trustDomain:          "private-beta",
		namespace:            "stable",
		namespaceGeneration:  12,
		authorityDigest:      "sha256:" + strings.Repeat("a", 64),
		revocationEpoch:      7,
		revocationDigest:     "sha256:" + strings.Repeat("b", 64),
		l1SecurityGeneration: 9,
		expiresAt:            edgeTestNow.Add(time.Hour),
	}
}

func testStoreConfig(root, sharedURL string) Config {
	return Config{
		Storage: Storage{
			StateDirectory:       root,
			FilesystemPolicy:     FilesystemPolicy,
			CapacityBytes:        MinimumCapacityBytes,
			MaximumObjectBytes:   MaximumObjectBytes,
			StableTTLSeconds:     int64(MaximumStableTTL / time.Second),
			PendingTTLSeconds:    int64(MaximumPendingTTL / time.Second),
			HighWatermarkPercent: HighWatermarkPercent,
			LowWatermarkPercent:  LowWatermarkPercent,
			ProtectedPercent:     ProtectedPercent,
		},
		Shared: Shared{
			BaseURL:               sharedURL,
			AllowInsecureLoopback: true,
		},
	}
}

func committedServer(t *testing.T, authority ReadAuthority, payload []byte) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	digestBytes := sha256.Sum256(payload)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	requests := &atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Method != http.MethodGet || request.URL.Path != "/cache/edge-key" ||
			request.Header.Get("Authorization") != "Bearer edge-token" ||
			request.Header.Get(sharedAuthorityDigestHeader) != authority.authorityDigest {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		response.Header().Set("ETag", `"`+digest+`"`)
		response.Header().Set(sharedCommitStateHeader, "COMMITTED")
		response.Header().Set(sharedDecisionDigestHeader, "sha256:"+strings.Repeat("c", 64))
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(payload)
	}))
	return server, requests
}

func TestReadThroughPersistsVerifiedCommitAndServesOfflineAfterRestart(t *testing.T) {
	payload := []byte("durable committed hit")
	authority := testReadAuthority()
	server, requests := committedServer(t, authority, payload)
	root := filepath.Join(t.TempDir(), "edge")
	config := testStoreConfig(root, server.URL)
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewSharedClient(config.Shared, []byte("edge-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	file, err := store.ReadThrough(context.Background(), authority, client, "edge-key", edgeTestNow)
	if err != nil {
		t.Fatal(err)
	}
	assertFileBytes(t, file, payload)
	if requests.Load() != 1 {
		t.Fatalf("Shared requests = %d, want 1", requests.Load())
	}
	client.Close()
	server.Close()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	file, err = reopened.OpenCommitted(context.Background(), authority, "edge-key", edgeTestNow.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	assertFileBytes(t, file, payload)
	for _, path := range []string{root, filepath.Join(root, "blobs"), filepath.Join(root, "spool")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("private directory %s = %v/%v", path, info, err)
		}
	}
}

func TestOfflineReadFailsClosedOnRevocationAuthorityAndTTLDrift(t *testing.T) {
	payload := []byte("durable committed hit")
	authority := testReadAuthority()
	server, _ := committedServer(t, authority, payload)
	defer server.Close()
	config := testStoreConfig(filepath.Join(t.TempDir(), "edge"), server.URL)
	config.Storage.StableTTLSeconds = 60
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client, err := NewSharedClient(config.Shared, []byte("edge-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	file, err := store.ReadThrough(context.Background(), authority, client, "edge-key", edgeTestNow)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	advanced := authority
	advanced.authorityDigest = "sha256:" + strings.Repeat("d", 64)
	advanced.revocationEpoch++
	advanced.revocationDigest = "sha256:" + strings.Repeat("e", 64)
	advanced.l1SecurityGeneration++
	if file, err := store.OpenCommitted(context.Background(), advanced, "edge-key", edgeTestNow.Add(time.Second)); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("advanced revocation read = %v/%v", file, err)
	}
	if file, err := store.OpenCommitted(context.Background(), authority, "edge-key", edgeTestNow.Add(61*time.Second)); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expired TTL read = %v/%v", file, err)
	}
	if file, err := store.OpenCommitted(context.Background(), ReadAuthority{}, "edge-key", edgeTestNow); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("absent authority read = %v/%v", file, err)
	}
}

func TestReadThroughRejectsNonCommittedAndCorruptResponses(t *testing.T) {
	authority := testReadAuthority()
	payload := []byte("payload")
	digestBytes := sha256.Sum256(payload)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	for name, mutate := range map[string]func(http.Header, *[]byte){
		"missing state": func(header http.Header, _ *[]byte) { header.Del(sharedCommitStateHeader) },
		"pending":       func(header http.Header, _ *[]byte) { header.Set(sharedCommitStateHeader, "PENDING") },
		"missing decision": func(header http.Header, _ *[]byte) {
			header.Del(sharedDecisionDigestHeader)
		},
		"bad etag":     func(header http.Header, _ *[]byte) { header.Set("ETag", `"sha256:bad"`) },
		"changed body": func(_ http.Header, body *[]byte) { *body = []byte("changed") },
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				body := append([]byte(nil), payload...)
				response.Header().Set("ETag", `"`+digest+`"`)
				response.Header().Set(sharedCommitStateHeader, "COMMITTED")
				response.Header().Set(sharedDecisionDigestHeader, "sha256:"+strings.Repeat("c", 64))
				mutate(response.Header(), &body)
				response.Header().Set("Content-Length", strconv.Itoa(len(body)))
				response.WriteHeader(http.StatusOK)
				_, _ = response.Write(body)
			}))
			defer server.Close()
			config := testStoreConfig(filepath.Join(t.TempDir(), "edge"), server.URL)
			store, err := OpenStore(config)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			client, err := NewSharedClient(config.Shared, []byte("edge-token"), server.Client())
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if file, err := store.ReadThrough(context.Background(), authority, client, "edge-key", edgeTestNow); file != nil || err == nil {
				t.Fatalf("invalid response read = %v/%v", file, err)
			}
			if file, err := store.OpenCommitted(context.Background(), authority, "edge-key", edgeTestNow); file != nil || !errors.Is(err, ErrCacheMiss) {
				t.Fatalf("invalid response became durable = %v/%v", file, err)
			}
		})
	}
}

func TestReadThroughNeverFollowsSharedRedirect(t *testing.T) {
	authority := testReadAuthority()
	var redirected atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/redirected" {
			redirected.Add(1)
			response.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(response, request, "/redirected", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	config := testStoreConfig(filepath.Join(t.TempDir(), "edge"), server.URL)
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client, err := NewSharedClient(config.Shared, []byte("edge-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if file, err := store.ReadThrough(context.Background(), authority, client, "edge-key", edgeTestNow); file != nil || !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("redirect read = %v/%v", file, err)
	}
	if redirected.Load() != 0 {
		t.Fatalf("redirect target requests = %d, want 0", redirected.Load())
	}
}

func TestCorruptDurableBlobBecomesMissAndWriterLeaseIsExclusive(t *testing.T) {
	payload := []byte("durable committed hit")
	authority := testReadAuthority()
	server, _ := committedServer(t, authority, payload)
	defer server.Close()
	config := testStoreConfig(filepath.Join(t.TempDir(), "edge"), server.URL)
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if second, err := OpenStore(config); second != nil || err == nil {
		t.Fatalf("second writer = %v/%v", second, err)
	}
	client, err := NewSharedClient(config.Shared, []byte("edge-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	file, err := store.ReadThrough(context.Background(), authority, client, "edge-key", edgeTestNow)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	digestBytes := sha256.Sum256(payload)
	hexDigest := hex.EncodeToString(digestBytes[:])
	blobPath := filepath.Join(config.Storage.StateDirectory, "blobs", "sha256", hexDigest[:2], hexDigest)
	if err := os.WriteFile(blobPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if file, err := store.OpenCommitted(context.Background(), authority, "edge-key", edgeTestNow); file != nil || !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("corrupt read = %v/%v", file, err)
	}
}

func TestOpenStoreRejectsFutureMetadataAndSymlinkState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "edge")
	config := testStoreConfig(root, "http://127.0.0.1:8042")
	store, err := OpenStore(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.database.Exec("PRAGMA user_version=4"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := OpenStore(config); reopened != nil || err == nil {
		t.Fatalf("future metadata reopen = %v/%v", reopened, err)
	}

	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	linkedConfig := testStoreConfig(filepath.Join(link, "edge"), "http://127.0.0.1:8042")
	if opened, err := OpenStore(linkedConfig); opened != nil || err == nil {
		t.Fatalf("symlink state open = %v/%v", opened, err)
	}
}

func assertFileBytes(t *testing.T, file *os.File, want []byte) {
	t.Helper()
	defer file.Close()
	actual, err := io.ReadAll(file)
	if err != nil || string(actual) != string(want) {
		t.Fatalf("file bytes = %q/%v, want %q", actual, err, want)
	}
}
