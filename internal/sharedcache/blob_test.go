package sharedcache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestBlobStorePublishesDeduplicatesAndVerifies(t *testing.T) {
	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer storage.Close()
	content := []byte("immutable Gradle cache object\n")

	blob, created, err := storage.Blobs().Put(ctx, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("put blob: %v", err)
	}
	if !created ||
		blob.Digest != "sha256:088c65de063c5585d1c0658a0cbd228aeab4f778e8c74c2fc7f41cc3378197d2" ||
		blob.Size != int64(len(content)) {
		t.Fatalf("blob = %+v created=%t", blob, created)
	}
	path, err := storage.blobs.pathForDigest(blob.Digest, false)
	if err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Dir(path), true, 0o700)
	assertMode(t, path, false, 0o600)
	assertEmptyDirectory(t, storage.Layout().Spool)

	duplicate, created, err := storage.Blobs().Put(
		ctx,
		bytes.NewReader(content),
	)
	if err != nil || created || duplicate != blob {
		t.Fatalf("duplicate = %+v/%t/%v, want %+v/false", duplicate, created, err, blob)
	}
	file, err := storage.Blobs().OpenVerified(ctx, blob)
	if err != nil {
		t.Fatalf("open verified blob: %v", err)
	}
	actual, err := io.ReadAll(file)
	closeErr := file.Close()
	if err != nil || closeErr != nil || !bytes.Equal(actual, content) {
		t.Fatalf("verified bytes = %q, read=%v close=%v", actual, err, closeErr)
	}
}

func TestBlobStoreConcurrentIdenticalPublishHasOneWinner(t *testing.T) {
	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	content := []byte(strings.Repeat("concurrent-content-", 1024))
	const writers = 24
	var (
		waitGroup sync.WaitGroup
		created   atomic.Int32
		digests   = make(chan Blob, writers)
		errs      = make(chan error, writers)
	)
	for range writers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			blob, wasCreated, err := storage.Blobs().Put(
				ctx,
				bytes.NewReader(content),
			)
			if err != nil {
				errs <- err
				return
			}
			if wasCreated {
				created.Add(1)
			}
			digests <- blob
		}()
	}
	waitGroup.Wait()
	close(errs)
	close(digests)
	for err := range errs {
		t.Errorf("concurrent put: %v", err)
	}
	if created.Load() != 1 {
		t.Fatalf("created writers = %d, want 1", created.Load())
	}
	var first Blob
	for blob := range digests {
		if first == (Blob{}) {
			first = blob
		} else if blob != first {
			t.Fatalf("different concurrent blob: %+v / %+v", first, blob)
		}
	}
	assertEmptyDirectory(t, storage.Layout().Spool)
}

func TestBlobStoreRejectsCorruptionAndUnsafeDigests(t *testing.T) {
	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	content := []byte("verified")
	blob, _, err := storage.Blobs().Put(ctx, bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	path, err := storage.blobs.pathForDigest(blob.Digest, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("poisoned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if file, err := storage.Blobs().OpenVerified(ctx, blob); err == nil ||
		!errors.Is(err, ErrBlobCorrupt) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("corrupt read = %+v/%v", file, err)
	}
	if duplicate, created, err := storage.Blobs().Put(
		ctx,
		bytes.NewReader(content),
	); err == nil || !errors.Is(err, ErrBlobCorrupt) {
		t.Fatalf("corrupt duplicate = %+v/%t/%v", duplicate, created, err)
	}

	for _, digest := range []string{
		"",
		strings.Repeat("a", 64),
		"sha256:" + strings.Repeat("A", 64),
		"sha256:" + strings.Repeat("g", 64),
		"sha256:" + strings.Repeat("a", 63),
		"../sha256:" + strings.Repeat("a", 64),
	} {
		if file, err := storage.Blobs().OpenVerified(
			ctx,
			Blob{Digest: digest, Size: 1},
		); err == nil || !errors.Is(err, ErrInvalidDigest) {
			if file != nil {
				_ = file.Close()
			}
			t.Errorf("unsafe digest %q = %+v/%v", digest, file, err)
		}
	}
}

func TestBlobStoreBoundsAndCleansFailedStreams(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shared")
	storage, err := openWithMaximumBlobBytes(
		context.Background(),
		root,
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	if blob, created, err := storage.Blobs().Put(
		context.Background(),
		strings.NewReader("123456789"),
	); !errors.Is(err, ErrBlobTooLarge) {
		t.Fatalf("oversized blob = %+v/%t/%v", blob, created, err)
	}
	assertEmptyDirectory(t, storage.Layout().Spool)

	streamErr := errors.New("fixture stream failure")
	if blob, created, err := storage.Blobs().Put(
		context.Background(),
		&failingReader{err: streamErr},
	); !errors.Is(err, streamErr) {
		t.Fatalf("failed stream = %+v/%t/%v", blob, created, err)
	}
	assertEmptyDirectory(t, storage.Layout().Spool)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if blob, created, err := storage.Blobs().Put(
		cancelled,
		strings.NewReader("small"),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled stream = %+v/%t/%v", blob, created, err)
	}
	assertEmptyDirectory(t, storage.Layout().Spool)
}

func TestBlobStoreRejectsOperationsAfterClose(t *testing.T) {
	storage, err := Open(
		context.Background(),
		filepath.Join(t.TempDir(), "shared"),
	)
	if err != nil {
		t.Fatal(err)
	}
	blobs := storage.Blobs()
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := blobs.Put(
		context.Background(),
		strings.NewReader("closed"),
	); !errors.Is(err, ErrClosed) {
		t.Fatalf("put after close = %v", err)
	}
	if _, err := blobs.OpenVerified(
		context.Background(),
		Blob{
			Digest: "sha256:" + strings.Repeat("a", 64),
			Size:   1,
		},
	); !errors.Is(err, ErrClosed) {
		t.Fatalf("open after close = %v", err)
	}
}

type failingReader struct {
	delivered bool
	err       error
}

func (reader *failingReader) Read(buffer []byte) (int, error) {
	if reader.delivered {
		return 0, reader.err
	}
	reader.delivered = true
	return copy(buffer, "partial"), nil
}

func assertEmptyDirectory(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s entries = %v, want empty", path, entries)
	}
}
