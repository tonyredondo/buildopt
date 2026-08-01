package edgecache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	segmentProbation = "PROBATION"
	segmentProtected = "PROTECTED"
)

var ErrCapacityExceeded = errors.New("Edge cache capacity exceeded")

type reservation struct {
	bytes    int64
	released bool
}

type CapacitySnapshot struct {
	CapacityBytes      int64 `json:"capacityBytes"`
	HighWatermarkBytes int64 `json:"highWatermarkBytes"`
	LowWatermarkBytes  int64 `json:"lowWatermarkBytes"`
	ProtectedBytes     int64 `json:"protectedBytes"`
	StableBytes        int64 `json:"stableBytes"`
	ProbationBytes     int64 `json:"probationBytes"`
	ProtectedUsedBytes int64 `json:"protectedUsedBytes"`
	PendingBytes       int64 `json:"pendingBytes"`
	TotalLogicalBytes  int64 `json:"totalLogicalBytes"`
	ReservedBytes      int64 `json:"reservedBytes"`
	Objects            int64 `json:"objects"`
}

type MaintenanceReport struct {
	StableBytesBefore int64 `json:"stableBytesBefore"`
	StableBytesAfter  int64 `json:"stableBytesAfter"`
	ExpiredObjects    int   `json:"expiredObjects"`
	ExpiredPending    int   `json:"expiredPending"`
	EvictedProbation  int   `json:"evictedProbation"`
	EvictedProtected  int   `json:"evictedProtected"`
	DemotedProtected  int   `json:"demotedProtected"`
	DeletedBlobs      int   `json:"deletedBlobs"`
}

type evictionRow struct {
	entry   storedEntry
	segment string
}

func percentage(value int64, percent int) int64 {
	return value * int64(percent) / 100
}

func (store *Store) reserve(ctx context.Context, size int64) (*reservation, error) {
	if store == nil || ctx == nil || size < 0 || size > store.maximumObjectBytes {
		return nil, ErrCapacityExceeded
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return nil, errors.New("Edge store is closed")
	}
	usage, err := logicalBytes(ctx, store.database)
	if err != nil {
		return nil, err
	}
	if usage+store.reservedBytes+size > store.capacityBytes {
		return nil, ErrCapacityExceeded
	}
	store.reservedBytes += size
	return &reservation{bytes: size}, nil
}

func (store *Store) release(value *reservation) {
	if store == nil || value == nil {
		return
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if value.released {
		return
	}
	store.reservedBytes -= value.bytes
	value.released = true
}

func (store *Store) publishEntryWithCapacityLocked(ctx context.Context, entry storedEntry) error {
	transaction, err := store.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	usage, err := logicalBytes(ctx, transaction)
	if err != nil {
		return err
	}
	var replaced int64
	err = transaction.QueryRowContext(ctx, `SELECT size_bytes FROM edge_entries
WHERE tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?`,
		entry.Tenant, entry.Repository, entry.TrustDomain, entry.Namespace,
		entry.NamespaceGeneration, entry.Key,
	).Scan(&replaced)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if usage-replaced+entry.SizeBytes > store.capacityBytes {
		return ErrCapacityExceeded
	}
	if _, err := transaction.ExecContext(ctx, `INSERT INTO edge_entries (
    tenant_id, repository_id, trust_domain, namespace, namespace_generation,
    cache_key, schema_version, blob_digest, size_bytes, decision_digest,
    authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, cached_at_unix_ms, expires_at_unix_ms,
    segment, last_access_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    expires_at_unix_ms = excluded.expires_at_unix_ms,
    segment = 'PROBATION',
    last_access_unix_ms = excluded.last_access_unix_ms`,
		entry.Tenant, entry.Repository, entry.TrustDomain, entry.Namespace,
		entry.NamespaceGeneration, entry.Key, entry.SchemaVersion, entry.BlobDigest,
		entry.SizeBytes, entry.DecisionDigest, entry.AuthorityDigest,
		entry.RevocationEpoch, entry.RevocationDigest, entry.L1SecurityGeneration,
		entry.CachedAtUnixMillis, entry.ExpiresAtUnixMillis,
		segmentProbation, entry.LastAccessUnixMillis,
	); err != nil {
		return err
	}
	if _, _, err := store.enforceWatermarksTx(ctx, transaction, entry); err != nil {
		return err
	}
	if _, err := store.enforceProtectedTargetTx(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	rollback = false
	_, err = store.deleteOrphanBlobsLocked(ctx)
	return err
}

func (store *Store) recordHit(ctx context.Context, entry storedEntry, now time.Time) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return errors.New("Edge store is closed")
	}
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := transaction.ExecContext(ctx, `UPDATE edge_entries
SET segment = 'PROTECTED', last_access_unix_ms = ?
WHERE tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?
  AND blob_digest = ? AND decision_digest = ?`,
		now.UTC().UnixMilli(), entry.Tenant, entry.Repository, entry.TrustDomain,
		entry.Namespace, entry.NamespaceGeneration, entry.Key,
		entry.BlobDigest, entry.DecisionDigest,
	)
	if err != nil {
		_ = transaction.Rollback()
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		_ = transaction.Rollback()
		if err != nil {
			return err
		}
		return ErrCacheMiss
	}
	if _, err := store.enforceProtectedTargetTx(ctx, transaction); err != nil {
		_ = transaction.Rollback()
		return err
	}
	return transaction.Commit()
}

func (store *Store) CapacitySnapshot(ctx context.Context) (CapacitySnapshot, error) {
	if store == nil || ctx == nil {
		return CapacitySnapshot{}, errors.New("inspect Edge capacity: invalid input")
	}
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	if store.closed {
		return CapacitySnapshot{}, errors.New("Edge store is closed")
	}
	return store.capacitySnapshot(ctx, store.database)
}

func (store *Store) capacitySnapshot(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (CapacitySnapshot, error) {
	snapshot := CapacitySnapshot{
		CapacityBytes:      store.capacityBytes,
		HighWatermarkBytes: store.highWatermarkBytes,
		LowWatermarkBytes:  store.lowWatermarkBytes,
		ProtectedBytes:     store.protectedBytes,
		ReservedBytes:      store.reservedBytes,
	}
	err := query.QueryRowContext(ctx, `SELECT
    coalesce(sum(size_bytes), 0),
    coalesce(sum(CASE WHEN segment = 'PROBATION' THEN size_bytes ELSE 0 END), 0),
    coalesce(sum(CASE WHEN segment = 'PROTECTED' THEN size_bytes ELSE 0 END), 0),
    count(*)
FROM edge_entries`).Scan(
		&snapshot.StableBytes,
		&snapshot.ProbationBytes,
		&snapshot.ProtectedUsedBytes,
		&snapshot.Objects,
	)
	if err != nil {
		return snapshot, err
	}
	err = query.QueryRowContext(
		ctx,
		"SELECT coalesce(sum(size_bytes), 0) FROM edge_pending_objects",
	).Scan(&snapshot.PendingBytes)
	snapshot.TotalLogicalBytes = snapshot.StableBytes + snapshot.PendingBytes
	return snapshot, err
}

func (store *Store) Maintain(ctx context.Context, now time.Time) (MaintenanceReport, error) {
	if store == nil || ctx == nil || now.IsZero() {
		return MaintenanceReport{}, errors.New("maintain Edge capacity: invalid input")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.closed {
		return MaintenanceReport{}, errors.New("Edge store is closed")
	}
	transaction, err := store.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return MaintenanceReport{}, err
	}
	report := MaintenanceReport{}
	before, err := stableBytes(ctx, transaction)
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	report.StableBytesBefore = before
	result, err := transaction.ExecContext(ctx, "DELETE FROM edge_entries WHERE expires_at_unix_ms <= ?", now.UTC().UnixMilli())
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	expired, err := result.RowsAffected()
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	report.ExpiredObjects = int(expired)
	pendingResult, err := transaction.ExecContext(
		ctx,
		"DELETE FROM edge_pending_objects WHERE expires_at_unix_ms <= ?",
		now.UTC().UnixMilli(),
	)
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	expiredPending, err := pendingResult.RowsAffected()
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	report.ExpiredPending = int(expiredPending)
	report.EvictedProbation, report.EvictedProtected, err = store.enforceWatermarksTx(ctx, transaction, storedEntry{})
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	report.DemotedProtected, err = store.enforceProtectedTargetTx(ctx, transaction)
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	report.StableBytesAfter, err = stableBytes(ctx, transaction)
	if err != nil {
		_ = transaction.Rollback()
		return report, err
	}
	if err := transaction.Commit(); err != nil {
		return report, err
	}
	report.DeletedBlobs, err = store.deleteOrphanBlobsLocked(ctx)
	return report, err
}

func (store *Store) enforceWatermarksTx(ctx context.Context, transaction *sql.Tx, excluded storedEntry) (int, int, error) {
	usage, err := stableBytes(ctx, transaction)
	if err != nil || usage < store.highWatermarkBytes {
		return 0, 0, err
	}
	rows, err := transaction.QueryContext(ctx, `SELECT
    schema_version, tenant_id, repository_id, trust_domain, namespace,
    namespace_generation, cache_key, blob_digest, size_bytes,
    decision_digest, authority_digest, revocation_epoch, revocation_digest,
    l1_security_generation, cached_at_unix_ms, expires_at_unix_ms,
    segment, last_access_unix_ms
FROM edge_entries
WHERE NOT (tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?)
ORDER BY CASE segment WHEN 'PROBATION' THEN 0 ELSE 1 END,
         last_access_unix_ms, size_bytes DESC, tenant_id, repository_id,
         trust_domain, namespace, namespace_generation, cache_key`,
		excluded.Tenant, excluded.Repository, excluded.TrustDomain,
		excluded.Namespace, excluded.NamespaceGeneration, excluded.Key,
	)
	if err != nil {
		return 0, 0, err
	}
	var candidates []evictionRow
	for rows.Next() {
		entry := storedEntry{}
		if err := rows.Scan(
			&entry.SchemaVersion, &entry.Tenant, &entry.Repository, &entry.TrustDomain,
			&entry.Namespace, &entry.NamespaceGeneration, &entry.Key, &entry.BlobDigest,
			&entry.SizeBytes, &entry.DecisionDigest, &entry.AuthorityDigest,
			&entry.RevocationEpoch, &entry.RevocationDigest, &entry.L1SecurityGeneration,
			&entry.CachedAtUnixMillis, &entry.ExpiresAtUnixMillis,
			&entry.Segment, &entry.LastAccessUnixMillis,
		); err != nil {
			_ = rows.Close()
			return 0, 0, err
		}
		candidates = append(candidates, evictionRow{entry: entry, segment: entry.Segment})
	}
	if err := rows.Close(); err != nil {
		return 0, 0, err
	}
	probation, protected := 0, 0
	for _, candidate := range candidates {
		if usage <= store.lowWatermarkBytes {
			break
		}
		result, err := transaction.ExecContext(ctx, `DELETE FROM edge_entries
WHERE tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?`,
			candidate.entry.Tenant, candidate.entry.Repository,
			candidate.entry.TrustDomain, candidate.entry.Namespace,
			candidate.entry.NamespaceGeneration, candidate.entry.Key,
		)
		if err != nil {
			return probation, protected, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return probation, protected, err
		}
		if affected == 0 {
			continue
		}
		usage -= candidate.entry.SizeBytes
		if candidate.segment == segmentProbation {
			probation++
		} else {
			protected++
		}
	}
	if usage > store.lowWatermarkBytes {
		return probation, protected, ErrCapacityExceeded
	}
	return probation, protected, nil
}

func (store *Store) enforceProtectedTargetTx(ctx context.Context, transaction *sql.Tx) (int, error) {
	var protected int64
	if err := transaction.QueryRowContext(ctx, "SELECT coalesce(sum(size_bytes), 0) FROM edge_entries WHERE segment = 'PROTECTED'").Scan(&protected); err != nil {
		return 0, err
	}
	if protected <= store.protectedBytes {
		return 0, nil
	}
	rows, err := transaction.QueryContext(ctx, `SELECT tenant_id, repository_id,
    trust_domain, namespace, namespace_generation, cache_key, size_bytes
FROM edge_entries WHERE segment = 'PROTECTED'
ORDER BY last_access_unix_ms, size_bytes DESC, tenant_id, repository_id,
         trust_domain, namespace, namespace_generation, cache_key`)
	if err != nil {
		return 0, err
	}
	type candidate struct {
		tenant, repository, trust, namespace, key string
		generation, size                          int64
	}
	var candidates []candidate
	for rows.Next() {
		value := candidate{}
		if err := rows.Scan(&value.tenant, &value.repository, &value.trust,
			&value.namespace, &value.generation, &value.key, &value.size); err != nil {
			_ = rows.Close()
			return 0, err
		}
		candidates = append(candidates, value)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	demoted := 0
	for _, value := range candidates {
		if protected <= store.protectedBytes {
			break
		}
		result, err := transaction.ExecContext(ctx, `UPDATE edge_entries
SET segment = 'PROBATION'
WHERE tenant_id = ? AND repository_id = ? AND trust_domain = ?
  AND namespace = ? AND namespace_generation = ? AND cache_key = ?
  AND segment = 'PROTECTED'`, value.tenant, value.repository, value.trust,
			value.namespace, value.generation, value.key)
		if err != nil {
			return demoted, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return demoted, err
		}
		if affected == 1 {
			protected -= value.size
			demoted++
		}
	}
	return demoted, nil
}

func stableBytes(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (int64, error) {
	var total int64
	err := query.QueryRowContext(ctx, "SELECT coalesce(sum(size_bytes), 0) FROM edge_entries").Scan(&total)
	return total, err
}

func logicalBytes(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (int64, error) {
	var total int64
	err := query.QueryRowContext(ctx, `SELECT
    (SELECT coalesce(sum(size_bytes), 0) FROM edge_entries) +
    (SELECT coalesce(sum(size_bytes), 0) FROM edge_pending_objects)`).Scan(&total)
	return total, err
}

func (store *Store) deleteOrphanBlobsLocked(ctx context.Context) (int, error) {
	rows, err := store.database.QueryContext(ctx, "SELECT DISTINCT blob_digest FROM edge_entries")
	if err != nil {
		return 0, err
	}
	referenced := map[string]bool{}
	for rows.Next() {
		var digest string
		if err := rows.Scan(&digest); err != nil {
			_ = rows.Close()
			return 0, err
		}
		referenced[digest] = true
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	pendingRows, err := store.database.QueryContext(
		ctx,
		"SELECT DISTINCT blob_digest FROM edge_pending_objects",
	)
	if err != nil {
		return 0, err
	}
	for pendingRows.Next() {
		var digest string
		if err := pendingRows.Scan(&digest); err != nil {
			_ = pendingRows.Close()
			return 0, err
		}
		referenced[digest] = true
	}
	if err := pendingRows.Close(); err != nil {
		return 0, err
	}
	deleted := 0
	err = filepath.WalkDir(store.blobs, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return os.Remove(path)
		}
		digest := "sha256:" + filepath.Base(path)
		if !validDigest(digest) || !referenced[digest] {
			if err := os.Remove(path); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	if err != nil {
		return deleted, fmt.Errorf("clean Edge orphan blobs: %w", err)
	}
	return deleted, nil
}
