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
	"time"
)

const (
	edgePendingSchema = "buildopt.edge-cache/pending-entry/v1"

	pendingQueued      = "QUEUED"
	pendingReplicating = "REPLICATING"
	pendingReplicated  = "REPLICATED"
	pendingRejected    = "REJECTED"

	maximumReplicationBatch = 64
	maximumRetryDelay       = 5 * time.Minute
)

var ErrPendingConflict = errors.New("Edge pending key has different bytes")

type PendingWrite struct {
	Digest string
	Size   int64
	Added  bool
}

type PendingSnapshot struct {
	Bytes      int64
	Objects    int64
	Queued     int64
	Replicated int64
	Rejected   int64
}

type ReplicationReport struct {
	Claimed    int
	Replicated int
	Deferred   int
	Rejected   int
}

type pendingEntry struct {
	Tenant               string
	Repository           string
	TrustDomain          string
	Namespace            string
	NamespaceGeneration  int64
	AttemptID            string
	Key                  string
	SchemaVersion        string
	AuthorityDigest      string
	RevocationEpoch      int64
	RevocationDigest     string
	L1SecurityGeneration int64
	BlobDigest           string
	SizeBytes            int64
	CreatedAtUnixMillis  int64
	ExpiresAtUnixMillis  int64
	State                string
	ReplicationAttempts  int64
	NextRetryUnixMillis  int64
	LastErrorClass       string
}

// PutPendingDurable verifies and stores one attempt-private object. It returns
// before any Shared request; only the replicator may send the object upstream.
func (store *Store) PutPendingDurable(
	ctx context.Context,
	authority WriteAuthority,
	key string,
	size int64,
	body io.Reader,
	now time.Time,
) (PendingWrite, error) {
	if store == nil || ctx == nil || body == nil || !validCacheKey(key) ||
		!authority.current(now) || size < 0 || size > store.maximumObjectBytes {
		return PendingWrite{}, errors.New("persist Edge pending object: invalid input")
	}
	reservation, err := store.reserve(ctx, size)
	if err != nil {
		return PendingWrite{}, err
	}
	defer store.release(reservation)

	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return PendingWrite{}, errors.New("Edge store is closed")
	}
	temporary, err := os.CreateTemp(store.spool, ".pending-")
	if err != nil {
		return PendingWrite{}, err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return PendingWrite{}, err
	}
	hash := sha256.New()
	written, err := io.Copy(
		io.MultiWriter(temporary, hash),
		io.LimitReader(body, store.maximumObjectBytes+1),
	)
	if err != nil || written != size || written > store.maximumObjectBytes {
		return PendingWrite{}, errors.New("persist Edge pending object: incomplete or oversized body")
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if err := temporary.Sync(); err != nil {
		return PendingWrite{}, err
	}
	if err := temporary.Close(); err != nil {
		return PendingWrite{}, err
	}
	finalPath := store.blobPath(digest)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
		return PendingWrite{}, err
	}
	if err := os.Link(temporaryPath, finalPath); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return PendingWrite{}, err
		}
		existing, openErr := openNoFollow(finalPath)
		if openErr != nil {
			return PendingWrite{}, openErr
		}
		verifyErr := verifyOpenBlob(ctx, existing, digest, size)
		_ = existing.Close()
		if verifyErr != nil {
			return PendingWrite{}, verifyErr
		}
	} else if err := syncDirectory(filepath.Dir(finalPath)); err != nil {
		return PendingWrite{}, err
	}

	expiresAt := now.UTC().Add(store.pendingTTL)
	if authority.expiresAt.Before(expiresAt) {
		expiresAt = authority.expiresAt
	}
	entry := pendingEntry{
		Tenant:               authority.tenant,
		Repository:           authority.repository,
		TrustDomain:          authority.trustDomain,
		Namespace:            authority.namespace,
		NamespaceGeneration:  authority.namespaceGeneration,
		AttemptID:            authority.attemptID,
		Key:                  key,
		SchemaVersion:        edgePendingSchema,
		AuthorityDigest:      authority.authorityDigest,
		RevocationEpoch:      authority.revocationEpoch,
		RevocationDigest:     authority.revocationDigest,
		L1SecurityGeneration: authority.l1SecurityGeneration,
		BlobDigest:           digest,
		SizeBytes:            size,
		CreatedAtUnixMillis:  now.UTC().UnixMilli(),
		ExpiresAtUnixMillis:  expiresAt.UTC().UnixMilli(),
		State:                pendingQueued,
	}
	added, err := store.publishPendingLocked(ctx, entry)
	if err != nil {
		_, cleanupErr := store.deleteOrphanBlobsLocked(context.Background())
		return PendingWrite{}, errors.Join(err, cleanupErr)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return PendingWrite{}, err
	}
	if err := syncDirectory(store.spool); err != nil {
		return PendingWrite{}, err
	}
	return PendingWrite{Digest: digest, Size: size, Added: added}, nil
}

func (store *Store) publishPendingLocked(
	ctx context.Context,
	entry pendingEntry,
) (bool, error) {
	transaction, err := store.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return false, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	var existingDigest string
	var existingSize int64
	err = transaction.QueryRowContext(ctx, `SELECT blob_digest, size_bytes
FROM edge_pending_objects
WHERE authority_digest = ? AND attempt_id = ? AND cache_key = ?`,
		entry.AuthorityDigest,
		entry.AttemptID,
		entry.Key,
	).Scan(&existingDigest, &existingSize)
	if err == nil {
		if existingDigest != entry.BlobDigest || existingSize != entry.SizeBytes {
			return false, ErrPendingConflict
		}
		if err := transaction.Commit(); err != nil {
			return false, err
		}
		rollback = false
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	usage, err := logicalBytes(ctx, transaction)
	if err != nil {
		return false, err
	}
	if usage+entry.SizeBytes > store.capacityBytes {
		return false, ErrCapacityExceeded
	}
	_, err = transaction.ExecContext(ctx, `INSERT INTO edge_pending_objects (
    tenant_id, repository_id, trust_domain, namespace, namespace_generation,
    attempt_id, cache_key, schema_version, authority_digest, revocation_epoch,
    revocation_digest, l1_security_generation, blob_digest, size_bytes,
    created_at_unix_ms, expires_at_unix_ms, state, replication_attempts,
    next_retry_unix_ms, last_error_class
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'QUEUED', 0, 0, '')`,
		entry.Tenant,
		entry.Repository,
		entry.TrustDomain,
		entry.Namespace,
		entry.NamespaceGeneration,
		entry.AttemptID,
		entry.Key,
		entry.SchemaVersion,
		entry.AuthorityDigest,
		entry.RevocationEpoch,
		entry.RevocationDigest,
		entry.L1SecurityGeneration,
		entry.BlobDigest,
		entry.SizeBytes,
		entry.CreatedAtUnixMillis,
		entry.ExpiresAtUnixMillis,
	)
	if err != nil {
		return false, err
	}
	if err := transaction.Commit(); err != nil {
		return false, err
	}
	rollback = false
	return true, nil
}

// OpenPending returns bytes only to the exact still-current attempt that wrote
// them. Rejected, expired, corrupt, or cross-attempt entries are byte-free
// misses and never fall through to committed authority.
func (store *Store) OpenPending(
	ctx context.Context,
	authority WriteAuthority,
	key string,
	now time.Time,
) (*os.File, error) {
	if store == nil || ctx == nil || !validCacheKey(key) ||
		!authority.current(now) {
		return nil, ErrCacheMiss
	}
	store.mutex.RLock()
	if store.closed {
		store.mutex.RUnlock()
		return nil, errors.New("Edge store is closed")
	}
	entry, err := store.loadPending(ctx, authority, key)
	if err != nil {
		store.mutex.RUnlock()
		return nil, err
	}
	if !authority.matchesPending(entry) ||
		entry.SchemaVersion != edgePendingSchema ||
		entry.State == pendingRejected ||
		entry.ExpiresAtUnixMillis <= now.UTC().UnixMilli() ||
		entry.SizeBytes < 0 || entry.SizeBytes > store.maximumObjectBytes ||
		!validDigest(entry.BlobDigest) {
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
		return nil, ErrCacheMiss
	}
	return file, nil
}

func (authority WriteAuthority) matchesPending(entry pendingEntry) bool {
	return entry.Tenant == authority.tenant &&
		entry.Repository == authority.repository &&
		entry.TrustDomain == authority.trustDomain &&
		entry.Namespace == authority.namespace &&
		entry.NamespaceGeneration == authority.namespaceGeneration &&
		entry.AttemptID == authority.attemptID &&
		entry.AuthorityDigest == authority.authorityDigest &&
		entry.RevocationEpoch == authority.revocationEpoch &&
		entry.RevocationDigest == authority.revocationDigest &&
		entry.L1SecurityGeneration == authority.l1SecurityGeneration
}

func (store *Store) loadPending(
	ctx context.Context,
	authority WriteAuthority,
	key string,
) (pendingEntry, error) {
	entry := pendingEntry{}
	err := store.database.QueryRowContext(ctx, `SELECT
    tenant_id, repository_id, trust_domain, namespace,
    namespace_generation, attempt_id, cache_key, schema_version,
    authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, blob_digest, size_bytes,
    created_at_unix_ms, expires_at_unix_ms, state,
    replication_attempts, next_retry_unix_ms, last_error_class
FROM edge_pending_objects
WHERE authority_digest = ? AND attempt_id = ? AND cache_key = ?`,
		authority.authorityDigest,
		authority.attemptID,
		key,
	).Scan(
		&entry.Tenant,
		&entry.Repository,
		&entry.TrustDomain,
		&entry.Namespace,
		&entry.NamespaceGeneration,
		&entry.AttemptID,
		&entry.Key,
		&entry.SchemaVersion,
		&entry.AuthorityDigest,
		&entry.RevocationEpoch,
		&entry.RevocationDigest,
		&entry.L1SecurityGeneration,
		&entry.BlobDigest,
		&entry.SizeBytes,
		&entry.CreatedAtUnixMillis,
		&entry.ExpiresAtUnixMillis,
		&entry.State,
		&entry.ReplicationAttempts,
		&entry.NextRetryUnixMillis,
		&entry.LastErrorClass,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return pendingEntry{}, ErrCacheMiss
	}
	return entry, err
}

// PendingSnapshot exposes aggregate pending state for operators and tests.
func (store *Store) PendingSnapshot(ctx context.Context) (PendingSnapshot, error) {
	if store == nil || ctx == nil {
		return PendingSnapshot{}, errors.New("inspect Edge pending: invalid input")
	}
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	if store.closed {
		return PendingSnapshot{}, errors.New("Edge store is closed")
	}
	snapshot := PendingSnapshot{}
	err := store.database.QueryRowContext(ctx, `SELECT
    coalesce(sum(size_bytes), 0),
    count(*),
    coalesce(sum(CASE WHEN state IN ('QUEUED', 'REPLICATING') THEN 1 ELSE 0 END), 0),
    coalesce(sum(CASE WHEN state = 'REPLICATED' THEN 1 ELSE 0 END), 0),
    coalesce(sum(CASE WHEN state = 'REJECTED' THEN 1 ELSE 0 END), 0)
FROM edge_pending_objects`).Scan(
		&snapshot.Bytes,
		&snapshot.Objects,
		&snapshot.Queued,
		&snapshot.Replicated,
		&snapshot.Rejected,
	)
	return snapshot, err
}

// ReplicatePendingOnce claims a bounded due batch, streams every verified blob
// to Shared, and durably records retry or terminal acknowledgement. It never
// creates committed metadata.
func (store *Store) ReplicatePendingOnce(
	ctx context.Context,
	authority WriteAuthority,
	client *SharedClient,
	now time.Time,
) (ReplicationReport, error) {
	if store == nil || ctx == nil || client == nil ||
		!authority.current(now) {
		return ReplicationReport{}, errors.New("replicate Edge pending: invalid input")
	}
	report := ReplicationReport{}
	for report.Claimed < maximumReplicationBatch {
		entry, file, found, err := store.claimPending(
			ctx,
			authority,
			now,
		)
		if err != nil {
			return report, err
		}
		if !found {
			return report, nil
		}
		report.Claimed++
		pushErr := client.pushPending(
			ctx,
			authority,
			entry.Key,
			file,
			entry.SizeBytes,
			entry.BlobDigest,
			now,
		)
		_ = file.Close()
		outcome, err := store.finishReplication(
			context.WithoutCancel(ctx),
			entry,
			now,
			pushErr,
		)
		if err != nil {
			return report, err
		}
		switch outcome {
		case pendingReplicated:
			report.Replicated++
		case pendingRejected:
			report.Rejected++
		default:
			report.Deferred++
		}
		if ctx.Err() != nil {
			return report, ctx.Err()
		}
	}
	return report, nil
}

func (store *Store) claimPending(
	ctx context.Context,
	authority WriteAuthority,
	now time.Time,
) (pendingEntry, *os.File, bool, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return pendingEntry{}, nil, false, errors.New("Edge store is closed")
	}
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return pendingEntry{}, nil, false, err
	}
	entry := pendingEntry{}
	err = transaction.QueryRowContext(ctx, `SELECT
    tenant_id, repository_id, trust_domain, namespace,
    namespace_generation, attempt_id, cache_key, schema_version,
    authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, blob_digest, size_bytes,
    created_at_unix_ms, expires_at_unix_ms, state,
    replication_attempts, next_retry_unix_ms, last_error_class
FROM edge_pending_objects
WHERE authority_digest = ? AND attempt_id = ? AND state = 'QUEUED'
  AND next_retry_unix_ms <= ? AND expires_at_unix_ms > ?
ORDER BY created_at_unix_ms, cache_key
LIMIT 1`,
		authority.authorityDigest,
		authority.attemptID,
		now.UTC().UnixMilli(),
		now.UTC().UnixMilli(),
	).Scan(
		&entry.Tenant,
		&entry.Repository,
		&entry.TrustDomain,
		&entry.Namespace,
		&entry.NamespaceGeneration,
		&entry.AttemptID,
		&entry.Key,
		&entry.SchemaVersion,
		&entry.AuthorityDigest,
		&entry.RevocationEpoch,
		&entry.RevocationDigest,
		&entry.L1SecurityGeneration,
		&entry.BlobDigest,
		&entry.SizeBytes,
		&entry.CreatedAtUnixMillis,
		&entry.ExpiresAtUnixMillis,
		&entry.State,
		&entry.ReplicationAttempts,
		&entry.NextRetryUnixMillis,
		&entry.LastErrorClass,
	)
	if errors.Is(err, sql.ErrNoRows) {
		_ = transaction.Rollback()
		return pendingEntry{}, nil, false, nil
	}
	if err != nil {
		_ = transaction.Rollback()
		return pendingEntry{}, nil, false, err
	}
	if !authority.matchesPending(entry) ||
		entry.SchemaVersion != edgePendingSchema ||
		!validDigest(entry.BlobDigest) ||
		entry.SizeBytes < 0 || entry.SizeBytes > store.maximumObjectBytes {
		_ = transaction.Rollback()
		return pendingEntry{}, nil, false, errors.New("replicate Edge pending: invalid durable metadata")
	}
	entry.ReplicationAttempts++
	result, err := transaction.ExecContext(ctx, `UPDATE edge_pending_objects
SET state = 'REPLICATING', replication_attempts = ?
WHERE authority_digest = ? AND attempt_id = ? AND cache_key = ?
  AND state = 'QUEUED'`,
		entry.ReplicationAttempts,
		entry.AuthorityDigest,
		entry.AttemptID,
		entry.Key,
	)
	if err != nil {
		_ = transaction.Rollback()
		return pendingEntry{}, nil, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		_ = transaction.Rollback()
		if err != nil {
			return pendingEntry{}, nil, false, err
		}
		return pendingEntry{}, nil, false, errors.New("replicate Edge pending: claim lost")
	}
	file, err := openNoFollow(store.blobPath(entry.BlobDigest))
	if err == nil {
		err = verifyOpenBlob(ctx, file, entry.BlobDigest, entry.SizeBytes)
	}
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		_ = transaction.Rollback()
		return pendingEntry{}, nil, false, errors.New("replicate Edge pending: corrupt blob")
	}
	if err := transaction.Commit(); err != nil {
		_ = file.Close()
		return pendingEntry{}, nil, false, err
	}
	entry.State = pendingReplicating
	return entry, file, true, nil
}

func (store *Store) finishReplication(
	ctx context.Context,
	entry pendingEntry,
	now time.Time,
	pushErr error,
) (string, error) {
	state := pendingReplicated
	nextRetry := int64(0)
	errorClass := ""
	if pushErr != nil {
		switch {
		case errors.Is(pushErr, ErrUpstreamRejected):
			state = pendingRejected
			errorClass = "AUTHORITY_REJECTED"
		case errors.Is(pushErr, ErrUpstreamConflict):
			state = pendingRejected
			errorClass = "ATTEMPT_REJECTED"
		default:
			state = pendingQueued
			errorClass = "UPSTREAM_UNAVAILABLE"
			nextRetry = now.UTC().Add(replicationRetryDelay(
				entry.ReplicationAttempts,
			)).UnixMilli()
		}
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return "", errors.New("Edge store is closed")
	}
	result, err := store.database.ExecContext(ctx, `UPDATE edge_pending_objects
SET state = ?, next_retry_unix_ms = ?, last_error_class = ?
WHERE authority_digest = ? AND attempt_id = ? AND cache_key = ?
  AND state = 'REPLICATING' AND replication_attempts = ?`,
		state,
		nextRetry,
		errorClass,
		entry.AuthorityDigest,
		entry.AttemptID,
		entry.Key,
		entry.ReplicationAttempts,
	)
	if err != nil {
		return "", err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if affected != 1 {
		return "", errors.New("replicate Edge pending: durable outcome lost")
	}
	return state, nil
}

func replicationRetryDelay(attempt int64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 9 {
		attempt = 9
	}
	delay := time.Second << (attempt - 1)
	if delay > maximumRetryDelay {
		return maximumRetryDelay
	}
	return delay
}

// RunReplicator continuously executes bounded replication passes until the
// caller cancels the worker or a local durable-state error occurs.
func (store *Store) RunReplicator(
	ctx context.Context,
	authority WriteAuthority,
	client *SharedClient,
	interval time.Duration,
	clock func() time.Time,
) error {
	if store == nil || ctx == nil || client == nil || interval <= 0 ||
		clock == nil {
		return errors.New("run Edge replicator: invalid input")
	}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			now := clock().UTC()
			if !authority.current(now) {
				return errors.New("run Edge replicator: write authority expired")
			}
			if _, err := store.ReplicatePendingOnce(
				ctx,
				authority,
				client,
				now,
			); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("run Edge replicator: %w", err)
			}
			timer.Reset(interval)
		}
	}
}
