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
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

const (
	// SchemaVersion is the current cache/control metadata schema.
	SchemaVersion = 6
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
	// ErrAttemptNotFound means no durable attempt owns the supplied identity.
	ErrAttemptNotFound = errors.New("Shared cache attempt was not found")
	// ErrAttemptConflict means an attempt cannot accept the requested operation.
	ErrAttemptConflict = errors.New("Shared cache attempt conflicts with the request")
	// ErrStatePrecondition means the attempt version is not the expected version.
	ErrStatePrecondition = errors.New("Shared cache attempt state precondition failed")
	// ErrIdempotencyConflict means an identity was reused with different bytes.
	ErrIdempotencyConflict = errors.New("Shared cache idempotency identity conflict")
	// ErrCommitRejected means a decision failed authorization or exact coverage.
	ErrCommitRejected = errors.New("Shared cache commit decision was rejected")
	// ErrCASLost means another attempt already committed at least one identity.
	ErrCASLost = errors.New("Shared cache first-writer CAS was lost")
	// ErrCacheMiss means no fully verified committed object may be returned.
	ErrCacheMiss = errors.New("Shared cache miss")
	// ErrCapacityExceeded means admission cannot complete within a hard pool,
	// repository, deployment, or currently available disk boundary.
	ErrCapacityExceeded = errors.New("Shared cache capacity exceeded")
)

// Layout is the complete private on-disk cache, control, and typed-state layout.
type Layout struct {
	Root            string
	Blobs           string
	Spool           string
	Quarantine      string
	CacheDatabase   string
	ControlDatabase string
	StateDatabase   string
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

// MetadataStore exposes only operational schema health. Publication operations
// remain Storage methods so the concrete SQLite transaction stays private.
type MetadataStore interface {
	Role() string
	SchemaVersion(context.Context) (int, error)
	IntegrityCheck(context.Context) error
}

// DiskCapacityProbe reports total and currently available bytes for one
// storage root. Production opens use the host filesystem probe; deterministic
// benchmark fault runs may inject a reduced availability reading at open time.
type DiskCapacityProbe func(root string) (total uint64, available uint64, err error)

// Storage owns the process-wide writer lease, immutable blobs, and the three
// deliberately independent SQLite lifecycles.
type Storage struct {
	operationMutex             sync.RWMutex
	lifecycleMutex             sync.RWMutex
	reconcileMutex             sync.RWMutex
	authorityMutex             sync.RWMutex
	currentAuthorityDigests    map[string]string
	capacityMutex              sync.Mutex
	accessBatchMutex           sync.Mutex
	currentAccessBatch         *protectedAccessBatch
	protectedAccessError       error
	closed                     bool
	layout                     Layout
	writerLock                 *os.File
	blobs                      *filesystemBlobStore
	cache                      *sqliteMetadata
	control                    *sqliteMetadata
	state                      *sqliteMetadata
	stateCASMutex              sync.Mutex
	capacity                   CapacityPolicy
	reservations               map[*pendingReservation]struct{}
	blobCleanupPending         bool
	quarantineCleanupPending   bool
	minimumNamespaceGeneration int64
	lifecycleLease             *datalifecycle.ManagedLease
	clock                      func() time.Time
	testHooks                  storageTestHooks
}

type storageTestHooks struct {
	beforeCacheCommit          func() error
	beforeControlIndex         func() error
	beforeCommittedBlobVerify  func()
	beforeProtectedAccessBatch func(int) error
	afterProtectedAccessBatch  func(error)
	afterPendingBlob           func()
	diskCapacity               func(string) (uint64, uint64, error)
}

// Open prepares and validates one local, private, single-writer storage root.
func Open(ctx context.Context, root string) (*Storage, error) {
	return openWithConfiguration(
		ctx,
		root,
		MaximumBlobBytes,
		CapacityPolicy{},
	)
}

func openWithMaximumBlobBytes(
	ctx context.Context,
	root string,
	maximumBlobBytes int64,
) (*Storage, error) {
	return openWithConfiguration(
		ctx,
		root,
		maximumBlobBytes,
		CapacityPolicy{},
	)
}

// OpenWithCapacity prepares storage with an explicit reduced private-beta
// capacity policy. It exists for controlled deployments and fault tests;
// zero-valued policies are rejected rather than silently defaulted.
func OpenWithCapacity(
	ctx context.Context,
	root string,
	maximumBlobBytes int64,
	capacity CapacityPolicy,
) (*Storage, error) {
	if capacity == (CapacityPolicy{}) {
		return nil, errors.New(
			"open single-node Shared storage: explicit capacity policy is empty",
		)
	}
	return openWithConfiguration(ctx, root, maximumBlobBytes, capacity)
}

// OpenWithCapacityProbe prepares reduced-capacity storage with an explicit
// disk probe. It is limited to deterministic in-repository benchmark fault
// execution; normal deployments must use Open or OpenWithCapacity.
func OpenWithCapacityProbe(
	ctx context.Context,
	root string,
	maximumBlobBytes int64,
	capacity CapacityPolicy,
	probe DiskCapacityProbe,
) (*Storage, error) {
	if probe == nil {
		return nil, errors.New(
			"open single-node Shared storage: disk capacity probe is nil",
		)
	}
	storage, err := OpenWithCapacity(
		ctx,
		root,
		maximumBlobBytes,
		capacity,
	)
	if err != nil {
		return nil, err
	}
	storage.testHooks.diskCapacity = probe
	return storage, nil
}

func openWithConfiguration(
	ctx context.Context,
	root string,
	maximumBlobBytes int64,
	capacity CapacityPolicy,
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
	lifecycleLease, boundary, err := datalifecycle.AcquireManagedLease(root)
	if err != nil {
		return nil, fmt.Errorf(
			"open single-node Shared storage: inspect managed lifecycle: %w",
			err,
		)
	}
	keepLifecycleLease := false
	defer func() {
		if !keepLifecycleLease {
			_ = lifecycleLease.Close()
		}
	}()
	layout := Layout{
		Root:            root,
		Blobs:           filepath.Join(root, "blobs", "sha256"),
		Spool:           filepath.Join(root, "spool"),
		Quarantine:      filepath.Join(root, "quarantine"),
		CacheDatabase:   filepath.Join(root, "cache.sqlite"),
		ControlDatabase: filepath.Join(root, "control.sqlite"),
		StateDatabase:   filepath.Join(root, "state.sqlite"),
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
	if capacity == (CapacityPolicy{}) {
		derivedCapacity, err := defaultCapacityPolicy(
			layout.Root,
			maximumBlobBytes,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"open single-node Shared storage: derive capacity: %w",
				err,
			)
		}
		capacity = derivedCapacity
	}
	if err := validateCapacityPolicy(capacity, maximumBlobBytes); err != nil {
		return nil, fmt.Errorf(
			"open single-node Shared storage: invalid capacity: %w",
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
		layout:                     layout,
		writerLock:                 writerLock,
		capacity:                   capacity,
		reservations:               make(map[*pendingReservation]struct{}),
		currentAuthorityDigests:    make(map[string]string),
		minimumNamespaceGeneration: boundary.MinimumNamespaceGeneration,
		lifecycleLease:             lifecycleLease,
		clock:                      time.Now,
	}
	cleanup := func(openErr error) (*Storage, error) {
		if storage.state != nil {
			_ = storage.state.close()
		}
		if storage.control != nil {
			_ = storage.control.close()
		}
		if storage.cache != nil {
			_ = storage.cache.close()
		}
		_ = releaseExclusiveLock(writerLock)
		_ = writerLock.Close()
		_ = storage.lifecycleLease.Close()
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

	state, err := openSQLiteMetadata(
		ctx,
		storage,
		stateMetadataDefinition(layout.StateDatabase),
	)
	if err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: state metadata: %w",
			err,
		))
	}
	storage.state = state
	if err := storage.applyCapacityPolicy(ctx); err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: apply capacity policy: %w",
			err,
		))
	}
	storage.blobs = &filesystemBlobStore{
		owner:            storage,
		blobRoot:         layout.Blobs,
		spoolRoot:        layout.Spool,
		maximumBlobBytes: maximumBlobBytes,
	}
	if _, err := storage.maintainStateMetadata(ctx, storage.now()); err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: maintain typed state: %w",
			err,
		))
	}
	if _, err := storage.reconcile(ctx, storage.now()); err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: reconcile: %w",
			err,
		))
	}
	if _, err := storage.maintainCapacityLocked(ctx, storage.now()); err != nil {
		return cleanup(fmt.Errorf(
			"open single-node Shared storage: capacity maintenance: %w",
			err,
		))
	}
	keepLifecycleLease = true
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
		"state.sqlite":           {},
		"state.sqlite-journal":   {},
		"state.sqlite-shm":       {},
		"state.sqlite-wal":       {},
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

// MaximumObjectBytes is the configured hard limit accepted by this storage.
func (storage *Storage) MaximumObjectBytes() int64 {
	if storage == nil || storage.blobs == nil {
		return 0
	}
	return storage.blobs.maximumBlobBytes
}

// CapacityPolicy returns the immutable effective policy for this process.
func (storage *Storage) CapacityPolicy() CapacityPolicy {
	if storage == nil {
		return CapacityPolicy{}
	}
	return storage.capacity
}

// CacheMetadata returns the sole future visibility-authority lifecycle.
func (storage *Storage) CacheMetadata() MetadataStore {
	return storage.cache
}

// ControlMetadata returns the independent repairable audit-index lifecycle.
func (storage *Storage) ControlMetadata() MetadataStore {
	return storage.control
}

// StateMetadata returns the independent typed BuildOpt-state lifecycle.
func (storage *Storage) StateMetadata() MetadataStore {
	return storage.state
}

func (storage *Storage) now() time.Time {
	return storage.clock().UTC()
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
	storage.accessBatchMutex.Lock()
	closeErrors = append(closeErrors, storage.protectedAccessError)
	storage.accessBatchMutex.Unlock()
	if storage.control != nil {
		closeErrors = append(closeErrors, storage.control.close())
	}
	if storage.state != nil {
		closeErrors = append(closeErrors, storage.state.close())
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
	if storage.lifecycleLease != nil {
		closeErrors = append(closeErrors, storage.lifecycleLease.Close())
		storage.lifecycleLease = nil
	}
	return errors.Join(closeErrors...)
}
