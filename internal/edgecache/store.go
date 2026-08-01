package edgecache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
	_ "modernc.org/sqlite"
)

const edgeEntrySchema = "buildopt.edge-cache/committed-entry/v1"

type Store struct {
	mutex              sync.RWMutex
	blobs              string
	spool              string
	database           *sql.DB
	writerLock         *os.File
	stableTTL          time.Duration
	maximumObjectBytes int64
	closed             bool
}

type storedEntry struct {
	SchemaVersion        string
	Tenant               string
	Repository           string
	TrustDomain          string
	Namespace            string
	NamespaceGeneration  int64
	Key                  string
	BlobDigest           string
	SizeBytes            int64
	DecisionDigest       string
	AuthorityDigest      string
	RevocationEpoch      int64
	RevocationDigest     string
	L1SecurityGeneration int64
	CachedAtUnixMillis   int64
	ExpiresAtUnixMillis  int64
}

// OpenStore prepares one private durable Edge committed-object store.
func OpenStore(config Config) (*Store, error) {
	root := config.Storage.StateDirectory
	if !absoluteCleanNonRoot(root) || config.Storage.FilesystemPolicy != FilesystemPolicy ||
		config.Storage.MaximumObjectBytes < 1 || config.Storage.MaximumObjectBytes > MaximumObjectBytes ||
		config.Storage.StableTTLSeconds < 1 ||
		config.Storage.StableTTLSeconds > int64(MaximumStableTTL/time.Second) {
		return nil, errors.New("open Edge store: invalid storage configuration")
	}
	if err := preparePrivateRoot(root); err != nil {
		return nil, fmt.Errorf("open Edge store: %w", err)
	}
	blobs := filepath.Join(root, "blobs", "sha256")
	spool := filepath.Join(root, "spool")
	for _, directory := range []string{filepath.Join(root, "blobs"), blobs, spool} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, fmt.Errorf("open Edge store: create layout: %w", err)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return nil, err
		}
	}
	lock, err := os.OpenFile(filepath.Join(root, "writer.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, errors.New("open Edge store: another writer owns the state directory")
	}
	if err := lock.Chmod(0o600); err != nil {
		_ = lock.Close()
		return nil, err
	}
	databasePath := filepath.Join(root, "edge.sqlite")
	databaseFile, err := os.OpenFile(databasePath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		_ = lock.Close()
		return nil, err
	}
	if err := databaseFile.Chmod(0o600); err != nil {
		_ = databaseFile.Close()
		_ = lock.Close()
		return nil, err
	}
	if err := databaseFile.Close(); err != nil {
		_ = lock.Close()
		return nil, err
	}
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		_ = lock.Close()
		return nil, err
	}
	database.SetMaxOpenConns(1)
	store := &Store{
		blobs:              blobs,
		spool:              spool,
		database:           database,
		writerLock:         lock,
		stableTTL:          time.Duration(config.Storage.StableTTLSeconds) * time.Second,
		maximumObjectBytes: config.Storage.MaximumObjectBytes,
	}
	if err := store.initialize(); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (store *Store) initialize() error {
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=FULL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := store.database.Exec(statement); err != nil {
			return fmt.Errorf("initialize Edge metadata: %w", err)
		}
	}
	var version int
	if err := store.database.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("initialize Edge metadata: %w", err)
	}
	if version != 0 && version != 1 {
		return errors.New("initialize Edge metadata: unsupported schema version")
	}
	if version == 0 {
		for _, statement := range []string{
			`CREATE TABLE IF NOT EXISTS edge_entries (
    tenant_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    trust_domain TEXT NOT NULL,
    namespace TEXT NOT NULL,
    namespace_generation INTEGER NOT NULL,
    cache_key TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    blob_digest TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    decision_digest TEXT NOT NULL,
    authority_digest TEXT NOT NULL,
    revocation_epoch INTEGER NOT NULL,
    revocation_digest TEXT NOT NULL,
    l1_security_generation INTEGER NOT NULL,
    cached_at_unix_ms INTEGER NOT NULL,
    expires_at_unix_ms INTEGER NOT NULL,
    PRIMARY KEY (tenant_id, repository_id, trust_domain, namespace,
                 namespace_generation, cache_key)
) STRICT`,
			"PRAGMA user_version=1",
		} {
			if _, err := store.database.Exec(statement); err != nil {
				return fmt.Errorf("initialize Edge metadata: %w", err)
			}
		}
	}
	var integrity string
	if err := store.database.QueryRow("PRAGMA quick_check").Scan(&integrity); err != nil || integrity != "ok" {
		return errors.New("initialize Edge metadata: integrity check failed")
	}
	rows, err := store.database.Query(`SELECT
    schema_version, tenant_id, repository_id, trust_domain, namespace,
    namespace_generation, cache_key, blob_digest, size_bytes,
    decision_digest, authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, cached_at_unix_ms, expires_at_unix_ms
FROM edge_entries LIMIT 0`)
	if err != nil {
		return fmt.Errorf("initialize Edge metadata: schema drift: %w", err)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return nil
}

// Close releases the metadata database and the process-wide writer lease.
func (store *Store) Close() error {
	if store == nil {
		return nil
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return nil
	}
	store.closed = true
	var databaseErr, unlockErr, lockErr error
	if store.database != nil {
		databaseErr = store.database.Close()
	}
	if store.writerLock != nil {
		unlockErr = unix.Flock(int(store.writerLock.Fd()), unix.LOCK_UN)
		lockErr = store.writerLock.Close()
	}
	return errors.Join(databaseErr, unlockErr, lockErr)
}

// ReadThrough returns a verified local committed object or fetches it from the
// authenticated Shared committed route, verifies it completely, and only then
// publishes durable Edge metadata.
func (store *Store) ReadThrough(
	ctx context.Context,
	authority ReadAuthority,
	client *SharedClient,
	key string,
	now time.Time,
) (*os.File, error) {
	if store == nil {
		return nil, errors.New("Edge store is nil")
	}
	file, err := store.OpenCommitted(ctx, authority, key, now)
	if err == nil {
		return file, nil
	}
	if !errors.Is(err, ErrCacheMiss) {
		return nil, err
	}
	fetched, err := client.fetch(ctx, authority, key, now)
	if err != nil {
		return nil, err
	}
	if err := store.persistFetched(ctx, authority, key, fetched, now); err != nil {
		return nil, err
	}
	return store.OpenCommitted(ctx, authority, key, now)
}

// OpenCommitted verifies authority, metadata, and complete blob bytes before
// returning a rewound file. Missing or stale revocation state is always a miss.
func (store *Store) OpenCommitted(
	ctx context.Context,
	authority ReadAuthority,
	key string,
	now time.Time,
) (*os.File, error) {
	if store == nil || ctx == nil || !validCacheKey(key) || !authority.current(now) {
		return nil, ErrCacheMiss
	}
	store.mutex.RLock()
	if store.closed {
		store.mutex.RUnlock()
		return nil, errors.New("Edge store is closed")
	}
	entry, err := store.loadEntry(ctx, authority, key)
	if err != nil {
		store.mutex.RUnlock()
		return nil, err
	}
	if !authority.matches(entry) || entry.SchemaVersion != edgeEntrySchema ||
		entry.ExpiresAtUnixMillis <= now.UTC().UnixMilli() ||
		!validDigest(entry.BlobDigest) || !validDigest(entry.DecisionDigest) ||
		entry.SizeBytes < 0 || entry.SizeBytes > store.maximumObjectBytes {
		store.mutex.RUnlock()
		return nil, ErrCacheMiss
	}
	file, err := openNoFollow(store.blobPath(entry.BlobDigest))
	if err == nil {
		err = verifyOpenBlob(ctx, file, entry.BlobDigest, entry.SizeBytes)
	}
	store.mutex.RUnlock()
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		store.deleteEntry(ctx, entry)
		return nil, ErrCacheMiss
	}
	return file, nil
}

func (store *Store) persistFetched(
	ctx context.Context,
	authority ReadAuthority,
	key string,
	fetched fetchedObject,
	now time.Time,
) error {
	if fetched.body == nil {
		return errors.New("persist Shared object: missing body")
	}
	defer fetched.body.Close()
	if fetched.size < 0 || fetched.size > store.maximumObjectBytes ||
		!validDigest(fetched.digest) || !validDigest(fetched.decisionDigest) ||
		!authority.current(now) {
		return errors.New("persist Shared object: invalid committed metadata")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return errors.New("Edge store is closed")
	}
	temporary, err := os.CreateTemp(store.spool, ".committed-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(fetched.body, store.maximumObjectBytes+1))
	if err != nil || written != fetched.size || written > store.maximumObjectBytes {
		return errors.New("persist Shared object: incomplete or oversized body")
	}
	actualDigest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if actualDigest != fetched.digest {
		return errors.New("persist Shared object: digest mismatch")
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	finalPath := store.blobPath(fetched.digest)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, finalPath); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		existing, openErr := openNoFollow(finalPath)
		if openErr != nil {
			return openErr
		}
		verifyErr := verifyOpenBlob(ctx, existing, fetched.digest, fetched.size)
		_ = existing.Close()
		if verifyErr != nil {
			return verifyErr
		}
	} else if err := syncDirectory(filepath.Dir(finalPath)); err != nil {
		return err
	}
	entry := storedEntry{
		SchemaVersion:        edgeEntrySchema,
		Tenant:               authority.tenant,
		Repository:           authority.repository,
		TrustDomain:          authority.trustDomain,
		Namespace:            authority.namespace,
		NamespaceGeneration:  authority.namespaceGeneration,
		Key:                  key,
		BlobDigest:           fetched.digest,
		SizeBytes:            fetched.size,
		DecisionDigest:       fetched.decisionDigest,
		AuthorityDigest:      authority.authorityDigest,
		RevocationEpoch:      authority.revocationEpoch,
		RevocationDigest:     authority.revocationDigest,
		L1SecurityGeneration: authority.l1SecurityGeneration,
		CachedAtUnixMillis:   now.UTC().UnixMilli(),
		ExpiresAtUnixMillis:  now.UTC().Add(store.stableTTL).UnixMilli(),
	}
	if err := store.upsertEntry(ctx, entry); err != nil {
		return err
	}
	keep = true
	if err := os.Remove(temporaryPath); err != nil {
		return err
	}
	return syncDirectory(store.spool)
}

func (store *Store) loadEntry(ctx context.Context, authority ReadAuthority, key string) (storedEntry, error) {
	entry := storedEntry{}
	err := store.database.QueryRowContext(ctx, `SELECT
    schema_version, tenant_id, repository_id, trust_domain, namespace,
    namespace_generation, cache_key, blob_digest, size_bytes,
    decision_digest, authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, cached_at_unix_ms, expires_at_unix_ms
FROM edge_entries
WHERE tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?`,
		authority.tenant, authority.repository, authority.trustDomain,
		authority.namespace, authority.namespaceGeneration, key,
	).Scan(
		&entry.SchemaVersion, &entry.Tenant, &entry.Repository, &entry.TrustDomain,
		&entry.Namespace, &entry.NamespaceGeneration, &entry.Key, &entry.BlobDigest,
		&entry.SizeBytes, &entry.DecisionDigest, &entry.AuthorityDigest,
		&entry.RevocationEpoch, &entry.RevocationDigest, &entry.L1SecurityGeneration,
		&entry.CachedAtUnixMillis, &entry.ExpiresAtUnixMillis,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return storedEntry{}, ErrCacheMiss
	}
	return entry, err
}

func (store *Store) upsertEntry(ctx context.Context, entry storedEntry) error {
	_, err := store.database.ExecContext(ctx, `INSERT INTO edge_entries (
    tenant_id, repository_id, trust_domain, namespace, namespace_generation,
    cache_key, schema_version, blob_digest, size_bytes, decision_digest,
    authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, cached_at_unix_ms, expires_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (tenant_id, repository_id, trust_domain, namespace,
             namespace_generation, cache_key) DO UPDATE SET
    schema_version = excluded.schema_version,
    blob_digest = excluded.blob_digest,
    size_bytes = excluded.size_bytes,
    decision_digest = excluded.decision_digest,
    authority_digest = excluded.authority_digest,
    revocation_epoch = excluded.revocation_epoch,
    revocation_digest = excluded.revocation_digest,
    l1_security_generation = excluded.l1_security_generation,
    cached_at_unix_ms = excluded.cached_at_unix_ms,
    expires_at_unix_ms = excluded.expires_at_unix_ms`,
		entry.Tenant, entry.Repository, entry.TrustDomain, entry.Namespace,
		entry.NamespaceGeneration, entry.Key, entry.SchemaVersion, entry.BlobDigest,
		entry.SizeBytes, entry.DecisionDigest, entry.AuthorityDigest,
		entry.RevocationEpoch, entry.RevocationDigest, entry.L1SecurityGeneration,
		entry.CachedAtUnixMillis, entry.ExpiresAtUnixMillis,
	)
	return err
}

func (store *Store) deleteEntry(ctx context.Context, entry storedEntry) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return
	}
	_, _ = store.database.ExecContext(ctx, `DELETE FROM edge_entries
WHERE tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?`,
		entry.Tenant, entry.Repository, entry.TrustDomain, entry.Namespace,
		entry.NamespaceGeneration, entry.Key,
	)
}

func (store *Store) blobPath(digest string) string {
	hexDigest := digest[len("sha256:"):]
	return filepath.Join(store.blobs, hexDigest[:2], hexDigest)
}

func preparePrivateRoot(root string) error {
	existing := root
	for {
		info, err := os.Lstat(existing)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("Edge state ancestor must be a real directory")
			}
			resolved, err := filepath.EvalSymlinks(existing)
			if err != nil || resolved != existing {
				return errors.New("Edge state ancestors must not contain symlinks")
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return errors.New("Edge state has no existing ancestor")
		}
		existing = parent
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return errors.New("Edge state root must not resolve through symlinks")
	}
	return os.Chmod(root, 0o700)
}

func openNoFollow(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func verifyOpenBlob(ctx context.Context, file *os.File, digest string, size int64) error {
	if file == nil || ctx == nil || !validDigest(digest) || size < 0 {
		return errors.New("invalid Edge blob verification")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return errors.New("Edge blob size or type mismatch")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if "sha256:"+hex.EncodeToString(hash.Sum(nil)) != digest {
		return errors.New("Edge blob digest mismatch")
	}
	_, err = file.Seek(0, io.SeekStart)
	return err
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
