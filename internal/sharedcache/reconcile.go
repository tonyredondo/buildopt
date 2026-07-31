package sharedcache

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"
)

// ReconciliationReport is the complete fail-closed startup/repair result.
type ReconciliationReport struct {
	StartedAt            time.Time
	CompletedAt          time.Time
	ExpiredAttempts      int
	InvalidatedDecisions int
	QuarantinedBlobs     int
	DeletedOrphanBlobs   int
	RepairedAuditRows    int
}

type invalidBlob struct {
	blob Blob
	err  error
}

type pendingCandidate struct {
	attemptID string
	key       string
	blob      Blob
}

// Reconcile expires dead pending writers, invalidates incomplete committed
// decisions, repairs the non-authoritative audit index, and deletes blobs that
// have no pending or committed metadata authority.
func (storage *Storage) Reconcile(
	ctx context.Context,
) (ReconciliationReport, error) {
	if ctx == nil {
		return ReconciliationReport{}, errors.New(
			"reconcile Shared cache: nil context",
		)
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return ReconciliationReport{}, err
	}
	defer finish()
	storage.reconcileMutex.Lock()
	defer storage.reconcileMutex.Unlock()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()
	return storage.reconcile(ctx, storage.now())
}

func (storage *Storage) reconcile(
	ctx context.Context,
	startedAt time.Time,
) (ReconciliationReport, error) {
	report := ReconciliationReport{StartedAt: startedAt}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	expired, err := storage.expirePendingAttempts(ctx, startedAt)
	if err != nil {
		return report, err
	}
	report.ExpiredAttempts = expired

	quarantinedPending, err := storage.reconcilePendingObjects(
		ctx,
		startedAt,
	)
	if err != nil {
		return report, err
	}
	report.QuarantinedBlobs += quarantinedPending

	invalidated, quarantinedCommitted, err :=
		storage.reconcileCommittedObjects(ctx, startedAt)
	if err != nil {
		return report, err
	}
	report.InvalidatedDecisions = invalidated
	report.QuarantinedBlobs += quarantinedCommitted

	repaired, err := storage.repairAuditIndex(ctx, startedAt)
	if err != nil {
		return report, err
	}
	report.RepairedAuditRows = repaired

	deleted, err := storage.deleteOrphanBlobs(ctx)
	if err != nil {
		return report, err
	}
	report.DeletedOrphanBlobs = deleted
	report.CompletedAt = storage.now()
	if report.CompletedAt.Before(report.StartedAt) {
		report.CompletedAt = report.StartedAt
	}
	if err := storage.recordReconciliation(ctx, report); err != nil {
		return report, err
	}
	return report, nil
}

func (storage *Storage) expirePendingAttempts(
	ctx context.Context,
	now time.Time,
) (int, error) {
	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return 0, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT attempt_id
FROM cache_attempts
WHERE state = 'PENDING' AND lease_expires_at_unix_ms <= ?
ORDER BY attempt_id`,
		now.UnixMilli(),
	)
	if err != nil {
		return 0, err
	}
	var attemptIDs []string
	for rows.Next() {
		var attemptID string
		if err := rows.Scan(&attemptID); err != nil {
			_ = rows.Close()
			return 0, err
		}
		attemptIDs = append(attemptIDs, attemptID)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, attemptID := range attemptIDs {
		record, err := storage.loadAttempt(ctx, transaction, attemptID)
		if err != nil {
			return 0, err
		}
		if err := storage.abortExpiredAttempt(
			ctx,
			transaction,
			record,
			now,
		); err != nil {
			return 0, err
		}
	}
	if err := transaction.Commit(); err != nil {
		return 0, err
	}
	rollback = false
	return len(attemptIDs), nil
}

func (storage *Storage) reconcilePendingObjects(
	ctx context.Context,
	now time.Time,
) (int, error) {
	rows, err := storage.cache.database.QueryContext(
		ctx,
		`SELECT pending.attempt_id, pending.cache_key,
       pending.blob_digest, pending.size_bytes
FROM pending_objects AS pending
JOIN cache_attempts AS attempt ON attempt.attempt_id = pending.attempt_id
WHERE attempt.state = 'PENDING'
ORDER BY pending.attempt_id, pending.cache_key`,
	)
	if err != nil {
		return 0, err
	}
	var candidates []pendingCandidate
	for rows.Next() {
		var candidate pendingCandidate
		if err := rows.Scan(
			&candidate.attemptID,
			&candidate.key,
			&candidate.blob.Digest,
			&candidate.blob.Size,
		); err != nil {
			_ = rows.Close()
			return 0, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	processed := make(map[string]struct{})
	quarantined := 0
	for _, candidate := range candidates {
		if _, ok := processed[candidate.attemptID]; ok {
			continue
		}
		file, verifyErr := storage.blobs.openVerified(ctx, candidate.blob)
		if file != nil {
			_ = file.Close()
		}
		if verifyErr == nil {
			continue
		}
		record, err := storage.loadAttempt(
			ctx,
			storage.cache.database,
			candidate.attemptID,
		)
		if err != nil {
			return quarantined, err
		}
		if record.status.State != AttemptPending {
			continue
		}
		if err := storage.quarantinePendingRecord(
			ctx,
			record,
			"",
			"reconcile:invalid-pending",
			candidate.blob,
			verifyErr,
			now,
		); err != nil {
			return quarantined, err
		}
		processed[candidate.attemptID] = struct{}{}
		quarantined++
	}
	return quarantined, nil
}

func (storage *Storage) reconcileCommittedObjects(
	ctx context.Context,
	now time.Time,
) (int, int, error) {
	rows, err := storage.cache.database.QueryContext(
		ctx,
		`SELECT decision_digest, blob_digest, size_bytes
FROM committed_objects
ORDER BY decision_digest, tenant_id, namespace_generation, cache_key`,
	)
	if err != nil {
		return 0, 0, err
	}
	type committedCandidate struct {
		decisionDigest string
		blob           Blob
	}
	var candidates []committedCandidate
	for rows.Next() {
		var candidate committedCandidate
		if err := rows.Scan(
			&candidate.decisionDigest,
			&candidate.blob.Digest,
			&candidate.blob.Size,
		); err != nil {
			_ = rows.Close()
			return 0, 0, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return 0, 0, err
	}
	invalid := make(map[string][]invalidBlob)
	for _, candidate := range candidates {
		file, verifyErr := storage.blobs.openVerified(ctx, candidate.blob)
		if file != nil {
			_ = file.Close()
		}
		if verifyErr != nil {
			invalid[candidate.decisionDigest] = append(
				invalid[candidate.decisionDigest],
				invalidBlob{blob: candidate.blob, err: verifyErr},
			)
		}
	}
	quarantined := 0
	for decisionDigest, invalidBlobs := range invalid {
		count, err := storage.invalidateDecision(
			ctx,
			decisionDigest,
			invalidBlobs,
			now,
		)
		if err != nil {
			return 0, quarantined, err
		}
		quarantined += count
	}
	return len(invalid), quarantined, nil
}

func (storage *Storage) quarantinePendingFailure(
	ctx context.Context,
	record attemptRecord,
	verified VerifiedCommitDecision,
	object CommitObject,
	verifyErr error,
	now time.Time,
) error {
	return storage.quarantinePendingRecord(
		ctx,
		record,
		verified.decision.DecisionDigest,
		verified.decision.DecisionID,
		Blob{Digest: object.Checksum, Size: object.SizeBytes},
		verifyErr,
		now,
	)
}

func (storage *Storage) quarantinePendingRecord(
	ctx context.Context,
	record attemptRecord,
	decisionDigest string,
	terminalID string,
	blob Blob,
	verifyErr error,
	now time.Time,
) error {
	reason := classifyBlobFailure(verifyErr)
	if reason == "CORRUPT" {
		if _, err := storage.blobs.quarantine(blob, reason, now); err != nil {
			return err
		}
	}
	fingerprint, err := fingerprintValue(struct {
		AttemptID string `json:"attemptId"`
		Blob      Blob   `json:"blob"`
		Reason    string `json:"reason"`
	}{
		AttemptID: record.status.AttemptID,
		Blob:      blob,
		Reason:    reason,
	})
	if err != nil {
		return err
	}
	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	if err := insertQuarantineRecord(
		ctx,
		transaction,
		nullableString(decisionDigest),
		nullableString(record.status.AttemptID),
		blob,
		reason,
		now,
	); err != nil {
		return err
	}
	if err := storage.abortAttemptTx(
		ctx,
		transaction,
		record,
		terminalID,
		fingerprint,
		"INCOMPLETE_COMMIT_DECISION",
		now,
	); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	rollback = false
	if record.status.PendingObjectCount > 0 {
		storage.blobCleanupPending = true
	}
	return nil
}

func (storage *Storage) invalidateDecision(
	ctx context.Context,
	decisionDigest string,
	invalidBlobs []invalidBlob,
	now time.Time,
) (int, error) {
	for _, invalid := range invalidBlobs {
		if classifyBlobFailure(invalid.err) != "CORRUPT" {
			continue
		}
		if _, err := storage.blobs.quarantine(
			invalid.blob,
			"CORRUPT",
			now,
		); err != nil {
			return 0, err
		}
	}
	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return 0, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	var attemptID sql.NullString
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT attempt_id FROM commit_decisions WHERE decision_digest = ?",
		decisionDigest,
	).Scan(&attemptID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	for _, invalid := range invalidBlobs {
		if err := insertQuarantineRecord(
			ctx,
			transaction,
			nullableString(decisionDigest),
			attemptID,
			invalid.blob,
			classifyBlobFailure(invalid.err),
			now,
		); err != nil {
			return 0, err
		}
	}
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM committed_objects WHERE decision_digest = ?",
		decisionDigest,
	); err != nil {
		return 0, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM commit_decisions WHERE decision_digest = ?",
		decisionDigest,
	); err != nil {
		return 0, err
	}
	if err := transaction.Commit(); err != nil {
		return 0, err
	}
	rollback = false
	storage.blobCleanupPending = true
	return len(invalidBlobs), nil
}

func insertQuarantineRecord(
	ctx context.Context,
	executor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	},
	decisionDigest sql.NullString,
	attemptID sql.NullString,
	blob Blob,
	reason string,
	now time.Time,
) error {
	_, err := executor.ExecContext(
		ctx,
		`INSERT INTO quarantine_records
    (decision_digest, attempt_id, blob_digest, size_bytes, reason,
     quarantined_at_unix_ms)
VALUES (?, ?, ?, ?, ?, ?)`,
		nullStringValue(decisionDigest),
		nullStringValue(attemptID),
		blob.Digest,
		blob.Size,
		reason,
		now.UnixMilli(),
	)
	return err
}

func (storage *Storage) repairAuditIndex(
	ctx context.Context,
	now time.Time,
) (int, error) {
	rows, err := storage.cache.database.QueryContext(
		ctx,
		"SELECT decision_digest FROM commit_decisions ORDER BY decision_digest",
	)
	if err != nil {
		return 0, err
	}
	var digests []string
	for rows.Next() {
		var digest string
		if err := rows.Scan(&digest); err != nil {
			_ = rows.Close()
			return 0, err
		}
		digests = append(digests, digest)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	repaired := 0
	for _, digest := range digests {
		result, err := storage.control.database.ExecContext(
			ctx,
			`INSERT INTO decision_audit_index
    (decision_digest, indexed_at_unix_ms)
VALUES (?, ?)
ON CONFLICT(decision_digest) DO NOTHING`,
			digest,
			now.UnixMilli(),
		)
		if err != nil {
			return repaired, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return repaired, err
		}
		repaired += int(affected)
	}
	return repaired, nil
}

func (storage *Storage) deleteOrphanBlobs(ctx context.Context) (int, error) {
	rows, err := storage.cache.database.QueryContext(
		ctx,
		`SELECT blob_digest FROM pending_objects
UNION
SELECT blob_digest FROM committed_objects`,
	)
	if err != nil {
		return 0, err
	}
	referenced := make(map[string]struct{})
	for rows.Next() {
		var digest string
		if err := rows.Scan(&digest); err != nil {
			_ = rows.Close()
			return 0, err
		}
		referenced[digest] = struct{}{}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	blobs, err := storage.blobs.list()
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, blob := range blobs {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		if _, ok := referenced[blob.Digest]; ok {
			continue
		}
		if err := storage.blobs.remove(blob); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (storage *Storage) recordReconciliation(
	ctx context.Context,
	report ReconciliationReport,
) error {
	_, err := storage.control.database.ExecContext(
		ctx,
		`INSERT INTO reconciliation_runs (
    started_at_unix_ms, completed_at_unix_ms, expired_attempts,
    invalidated_decisions, quarantined_blobs, deleted_orphan_blobs,
    repaired_audit_rows, status
) VALUES (?, ?, ?, ?, ?, ?, ?, 'COMPLETE')`,
		report.StartedAt.UnixMilli(),
		report.CompletedAt.UnixMilli(),
		report.ExpiredAttempts,
		report.InvalidatedDecisions,
		report.QuarantinedBlobs,
		report.DeletedOrphanBlobs,
		report.RepairedAuditRows,
	)
	return err
}

func classifyBlobFailure(err error) string {
	if errors.Is(err, os.ErrNotExist) {
		return "MISSING"
	}
	return "CORRUPT"
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}
