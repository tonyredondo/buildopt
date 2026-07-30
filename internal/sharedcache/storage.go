// Package sharedcache owns the private-beta single-node Shared storage.
package sharedcache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	// SchemaVersion is the only cache/control metadata schema understood by A0-004.
	SchemaVersion = 1
	// MaximumBlobBytes is the private-beta per-object ceiling.
	MaximumBlobBytes int64 = 100 << 20
)

var (
	// ErrWriterBusy means another buildopt-server owns the single-node writer.
	ErrWriterBusy = errors.New("single-node Shared writer is already active")
	// ErrClosed means the storage lifetime has ended.
	ErrClosed = errors.New("single-node Shared storage is closed")
	// ErrBlobTooLarge means a stream exceeded the private-beta object ceiling.
	ErrBlobTooLarge = errors.New("Shared blob exceeds the maximum object size")
	// ErrBlobCorrupt means durable bytes do not match their content address.
	ErrBlobCorrupt = errors.New("Shared blob is corrupt")
	// ErrInvalidDigest means a digest is not canonical SHA-256.
	ErrInvalidDigest = errors.New("invalid Shared blob digest")
)

// Layout is the complete private on-disk A0-004 layout.
type Layout struct {
	Root            string
	Blobs           string
	Spool           string
	Quarantine      string
	CacheDatabase   string
	ControlDatabase string
	WriterLock      string
}

// Blob is the verified content address and exact byte length of one object.
type Blob struct {
	Digest string
	Size   int64
}

// BlobStore is the implementation-independent immutable object boundary.
type BlobStore interface {
	Put(context.Context, io.Reader) (blob Blob, created bool, err error)
	OpenVerified(context.Context, Blob) (*os.File, error)
}

// MetadataStore exposes only operational schema health. A0-005 owns commit
// operations and keeps the concrete SQLite transaction private.
type MetadataStore interface {
	Role() string
	SchemaVersion(context.Context) (int, error)
	IntegrityCheck(context.Context) error
}

// Storage owns the process-wide writer lease, immutable blobs, and the two
// deliberately independent SQLite lifecycles.
type Storage struct {
	operationMutex sync.RWMutex
	closed         bool
	layout         Layout
	writerLock     *os.File
	blobs          *filesystemBlobStore
	cache          *sqliteMetadata
	control        *sqliteMetadata
}

// Open prepares and validates one local, private, single-writer storage root.
func Open(ctx context.Context, root string) (*Storage, error) {
	return openWithMaximumBlobBytes(ctx, root, MaximumBlobBytes)
}

func openWithMaximumBlobBytes(
	ctx context.Context,
	root string,
	maximumBlobBytes int64,
) (*Storage, error) {
	if ctx == nil {
		return nil, errors.New("open single-node Shared storage: nil context")
	}
	if !filepath.IsAbs(root) {
		return nil, errors.New(
			"open single-node Shared storage: root must be absolute",
		)
	}
	if maximumBlobBytes < 1 || maximumBlobBytes > MaximumBlobBytes {
		return nil, errors.New(
			"open single-node Shared storage: invalid maximum blob size",
		)
	}

	root = filepath.Clean(root)
	layout := Layout{
		Root:            root,
		Blobs:           filepath.Join(root, "blobs", "sha256"),
		Spool:           filepath.Join(root, "spool"),
		Quarantine:      filepath.Join(root, "quarantine"),
		CacheDatabase:   filepath.Join(root, "cache.sqlite"),
		ControlDatabase: filepath.Join(root, "control.sqlite"),
		WriterLock:      filepath.Join(root, "writer.lock"),
	}
	if err := preparePrivateDirectory(layout.Root); err != nil {
		return nil, fmt.Errorf(
			"open single-node Shared storage: prepare root: %w",
			err,
		)
	}
	if err := validateStorageRootEntries(layout.Root); err != nil {
		return nil, fmt.Errorf(
			"open single-node Shared storage: validate root: %w",
			err,
		)
	}
	for _, directory := range []string{
		filepath.Join(layout.Root, "blobs"),
		layout.Blobs,
		layout.Spool,
		layout.Quarantine,
	} {
		if err := preparePrivateDirectory(directory); err != nil {
			return nil, fmt.Errorf(
				"open single-node Shared storage: prepare layout: %w",
				err,
			)
		}
	}
	if err := validateLocalStorageFilesystem(
		layout.Root,
		layout.Blobs,
		layout.Spool,
		layout.Quarantine,
	); err != nil {
		return nil, fmt.Errorf(
			"open single-node Shared storage: validate filesystem: %w",
			err,
		)
	}

	writerLock, err := openPrivateLock(layout.WriterLock)
	if err != nil {
		return nil, fmt.Errorf(
			"open single-node Shared storage: open writer lock: %w",
			err,
		)
	}
	if err := acquireExclusiveLock(writerLock); err != nil {
		_ = writerLock.Close()
		if isLockBusy(err) {
			return nil, ErrWriterBusy
		}
		return nil, fmt.Errorf(
			"open single-node Shared storage: acquire writer lock: %w",
			err,
		)
	}

	storage := &Storage{
		layout:     layout,
		writerLock: writerLock,
	}
	cleanup := func(openErr error) (*Storage, error) {
		if storage.control != nil {
			_ = storage.control.close()
		}
		if storage.cache != nil {
			_ = storage.cache.close()
		}
		_ = releaseExclusiveLock(writerLock)
		_ = writerLock.Close()
		return nil, openErr
	}

	cache, err := openSQLiteMetadata(
		ctx,
		storage,
		cacheMetadataDefinition(layout.CacheDatabase),
	)
	if err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: cache metadata: %w",
			err,
		))
	}
	storage.cache = cache

	control, err := openSQLiteMetadata(
		ctx,
		storage,
		controlMetadataDefinition(layout.ControlDatabase),
	)
	if err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: control metadata: %w",
			err,
		))
	}
	storage.control = control
	storage.blobs = &filesystemBlobStore{
		owner:            storage,
		blobRoot:         layout.Blobs,
		spoolRoot:        layout.Spool,
		maximumBlobBytes: maximumBlobBytes,
	}
	return storage, nil
}

func validateStorageRootEntries(root string) error {
	allowed := map[string]struct{}{
		"blobs":                  {},
		"spool":                  {},
		"quarantine":             {},
		"writer.lock":            {},
		"cache.sqlite":           {},
		"cache.sqlite-journal":   {},
		"cache.sqlite-shm":       {},
		"cache.sqlite-wal":       {},
		"control.sqlite":         {},
		"control.sqlite-journal": {},
		"control.sqlite-shm":     {},
		"control.sqlite-wal":     {},
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, ok := allowed[entry.Name()]; !ok {
			return fmt.Errorf("unexpected root entry %q", entry.Name())
		}
	}
	return nil
}

// Layout returns a copy of the validated on-disk paths.
func (storage *Storage) Layout() Layout {
	return storage.layout
}

// Blobs returns the immutable content-addressed implementation boundary.
func (storage *Storage) Blobs() BlobStore {
	return storage.blobs
}

// CacheMetadata returns the sole future visibility-authority lifecycle.
func (storage *Storage) CacheMetadata() MetadataStore {
	return storage.cache
}

// ControlMetadata returns the independent repairable audit-index lifecycle.
func (storage *Storage) ControlMetadata() MetadataStore {
	return storage.control
}

func (storage *Storage) beginOperation() (func(), error) {
	storage.operationMutex.RLock()
	if storage.closed {
		storage.operationMutex.RUnlock()
		return nil, ErrClosed
	}
	return storage.operationMutex.RUnlock, nil
}

// Close ends all operations before releasing the single-node writer lease.
func (storage *Storage) Close() error {
	if storage == nil {
		return nil
	}
	storage.operationMutex.Lock()
	defer storage.operationMutex.Unlock()
	if storage.closed {
		return nil
	}
	storage.closed = true

	var closeErrors []error
	if storage.control != nil {
		closeErrors = append(closeErrors, storage.control.close())
	}
	if storage.cache != nil {
		closeErrors = append(closeErrors, storage.cache.close())
	}
	if storage.writerLock != nil {
		closeErrors = append(
			closeErrors,
			releaseExclusiveLock(storage.writerLock),
			storage.writerLock.Close(),
		)
		storage.writerLock = nil
	}
	return errors.Join(closeErrors...)
}
