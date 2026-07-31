package sharedcache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	privateBetaDeploymentLimit  int64 = 500 << 30
	privateBetaRepositoryLimit  int64 = 100 << 30
	privateBetaStableTTL              = 30 * 24 * time.Hour
	privateBetaQuarantineTTL          = 7 * 24 * time.Hour
	capacityHighPercent         int64 = 85
	capacityLowPercent          int64 = 75
	protectedTargetPercent      int64 = 80
	pendingPoolPercent          int64 = 10
	accessUpdateInterval              = time.Minute
	protectedAccessBatchWindow        = 2 * time.Millisecond
	protectedAccessBatchTimeout       = 5 * time.Second
)

const (
	segmentProbation = "PROBATION"
	segmentProtected = "PROTECTED"
)

// CapacityPolicy is the exact immutable admission and retention boundary for
// one Shared process. Byte values are logical accounting limits; disk
// availability is independently checked before each streamed reservation.
type CapacityPolicy struct {
	DeploymentBytes        int64
	RepositoryBytes        int64
	PendingQuarantineBytes int64
	StableTTL              time.Duration
	QuarantineTTL          time.Duration
	HighWatermarkPercent   int64
	LowWatermarkPercent    int64
	ProtectedPercent       int64
	AccessUpdateInterval   time.Duration
}

// CapacitySnapshot exposes bounded aggregate accounting without identities.
type CapacitySnapshot struct {
	DeploymentBytes        int64
	RepositoryBytes        int64
	PendingQuarantineBytes int64
	HighWatermarkBytes     int64
	LowWatermarkBytes      int64
	StableBytes            int64
	ProbationBytes         int64
	ProtectedBytes         int64
	PendingBytes           int64
	QuarantineBytes        int64
	ReservedBytes          int64
	StableObjects          int64
	AdmissionBlocked       bool
	HighWatermarkReached   bool
}

// CapacityMaintenanceReport describes one complete logical-first and
// physical-second maintenance cycle.
type CapacityMaintenanceReport struct {
	StartedAt               time.Time
	CompletedAt             time.Time
	StableBytesBefore       int64
	StableBytesAfter        int64
	ExpiredObjects          int
	EvictedProbation        int
	EvictedProtected        int
	DemotedProtected        int
	ExpiredAttempts         int
	ExpiredQuarantine       int
	DeletedUnreferencedBlob int
}

type pendingReservation struct {
	scope    capacityScope
	amount   int64
	released bool
}

type capacityScope struct {
	tenant              string
	repository          string
	trustDomain         string
	namespaceGeneration int64
}

type capacityUsage struct {
	stableBytes     int64
	probationBytes  int64
	protectedBytes  int64
	pendingBytes    int64
	quarantineBytes int64
	stableObjects   int64
}

type evictionCandidate struct {
	tenant              string
	namespaceGeneration int64
	key                 string
	size                int64
	segment             string
}

type protectedAccessKey struct {
	tenant              string
	namespaceGeneration int64
	key                 string
	blobDigest          string
	sizeBytes           int64
	decisionDigest      string
}

type protectedAccessBatch struct {
	entries map[protectedAccessKey]time.Time
	err     error
}

func defaultCapacityPolicy(
	root string,
	maximumBlobBytes int64,
) (CapacityPolicy, error) {
	total, _, err := storageDiskCapacity(root)
	if err != nil || total == 0 || total > math.MaxInt64 {
		if err == nil {
			err = errors.New("storage volume capacity is unavailable")
		}
		return CapacityPolicy{}, err
	}
	deployment := int64(total / 2)
	if deployment > privateBetaDeploymentLimit {
		deployment = privateBetaDeploymentLimit
	}
	repository := privateBetaRepositoryLimit
	if repository > deployment {
		repository = deployment
	}
	pending := percentageBytes(deployment, pendingPoolPercent)
	if pending < maximumBlobBytes {
		pending = maximumBlobBytes
	}
	return CapacityPolicy{
		DeploymentBytes:        deployment,
		RepositoryBytes:        repository,
		PendingQuarantineBytes: pending,
		StableTTL:              privateBetaStableTTL,
		QuarantineTTL:          privateBetaQuarantineTTL,
		HighWatermarkPercent:   capacityHighPercent,
		LowWatermarkPercent:    capacityLowPercent,
		ProtectedPercent:       protectedTargetPercent,
		AccessUpdateInterval:   accessUpdateInterval,
	}, nil
}

func validateCapacityPolicy(
	policy CapacityPolicy,
	maximumBlobBytes int64,
) error {
	if policy.DeploymentBytes < maximumBlobBytes ||
		policy.DeploymentBytes > privateBetaDeploymentLimit ||
		policy.RepositoryBytes < maximumBlobBytes ||
		policy.RepositoryBytes > policy.DeploymentBytes ||
		policy.RepositoryBytes > privateBetaRepositoryLimit ||
		policy.PendingQuarantineBytes < maximumBlobBytes ||
		policy.PendingQuarantineBytes > policy.DeploymentBytes ||
		policy.PendingQuarantineBytes >
			percentageBytes(policy.DeploymentBytes, pendingPoolPercent) ||
		policy.StableTTL <= 0 ||
		policy.StableTTL > privateBetaStableTTL ||
		policy.QuarantineTTL <= 0 ||
		policy.QuarantineTTL > privateBetaQuarantineTTL ||
		policy.LowWatermarkPercent < 1 ||
		policy.HighWatermarkPercent <= policy.LowWatermarkPercent ||
		policy.HighWatermarkPercent > 100 ||
		policy.ProtectedPercent < 1 ||
		policy.ProtectedPercent > 100 ||
		policy.AccessUpdateInterval <= 0 {
		return errors.New("capacity values violate the private-beta boundary")
	}
	return nil
}

func percentageBytes(value int64, percent int64) int64 {
	return value * percent / 100
}

func (storage *Storage) applyCapacityPolicy(ctx context.Context) error {
	ttlMillis := storage.capacity.StableTTL.Milliseconds()
	if ttlMillis < 1 {
		return errors.New("stable TTL cannot be represented in milliseconds")
	}
	_, err := storage.cache.database.ExecContext(
		ctx,
		`UPDATE storage_entries
SET expires_at_unix_ms = min(
    expires_at_unix_ms,
    (
        SELECT object.committed_at_unix_ms + ?
        FROM committed_objects AS object
        WHERE object.tenant_id = storage_entries.tenant_id
          AND object.namespace_generation =
              storage_entries.namespace_generation
          AND object.cache_key = storage_entries.cache_key
    )
)`,
		ttlMillis,
	)
	return err
}

func (storage *Storage) diskCapacity() (uint64, uint64, error) {
	if storage.testHooks.diskCapacity != nil {
		return storage.testHooks.diskCapacity(storage.layout.Root)
	}
	return storageDiskCapacity(storage.layout.Root)
}

func (storage *Storage) reservePendingLocked(
	ctx context.Context,
	scope capacityScope,
	declaredBytes int64,
) (*pendingReservation, error) {
	amount := declaredBytes
	if amount < 0 {
		amount = storage.blobs.maximumBlobBytes
	}
	if amount > storage.blobs.maximumBlobBytes {
		return nil, ErrBlobTooLarge
	}
	storage.capacityMutex.Lock()
	defer storage.capacityMutex.Unlock()

	namespaceUsage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		scope,
	)
	if err != nil {
		return nil, err
	}
	repositoryScope := scope
	repositoryScope.namespaceGeneration = 0
	repositoryUsage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		repositoryScope,
	)
	if err != nil {
		return nil, err
	}
	globalUsage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		capacityScope{},
	)
	if err != nil {
		return nil, err
	}
	reservedTotal, reservedRepository, reservedNamespace :=
		storage.reservedBytesLocked(scope)
	if amount > storage.capacity.PendingQuarantineBytes-
		globalUsage.pendingBytes-globalUsage.quarantineBytes-reservedTotal ||
		amount > storage.capacity.DeploymentBytes-
			globalUsage.stableBytes-globalUsage.pendingBytes-
			globalUsage.quarantineBytes-reservedTotal ||
		amount > storage.capacity.RepositoryBytes-
			repositoryUsage.stableBytes-repositoryUsage.pendingBytes-
			reservedRepository ||
		amount > storage.capacity.RepositoryBytes-
			namespaceUsage.stableBytes-namespaceUsage.pendingBytes-
			reservedNamespace {
		return nil, ErrCapacityExceeded
	}
	_, available, err := storage.diskCapacity()
	if err != nil ||
		available > math.MaxInt64 ||
		amount > int64(available)-reservedTotal {
		return nil, ErrCapacityExceeded
	}
	reservation := &pendingReservation{
		scope:  scope,
		amount: amount,
	}
	storage.reservations[reservation] = struct{}{}
	return reservation, nil
}

func (storage *Storage) resizePendingReservationLocked(
	ctx context.Context,
	reservation *pendingReservation,
	actualBytes int64,
	writtenBytes int64,
) error {
	if reservation == nil || actualBytes < 0 || writtenBytes < 0 ||
		writtenBytes > actualBytes ||
		actualBytes > storage.blobs.maximumBlobBytes {
		return ErrCapacityExceeded
	}
	storage.capacityMutex.Lock()
	defer storage.capacityMutex.Unlock()
	if reservation.released {
		return ErrCapacityExceeded
	}
	if actualBytes <= reservation.amount {
		reservation.amount = actualBytes
		return nil
	}
	additional := actualBytes - reservation.amount
	namespaceUsage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		reservation.scope,
	)
	if err != nil {
		return err
	}
	repositoryScope := reservation.scope
	repositoryScope.namespaceGeneration = 0
	repositoryUsage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		repositoryScope,
	)
	if err != nil {
		return err
	}
	globalUsage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		capacityScope{},
	)
	if err != nil {
		return err
	}
	reservedTotal, reservedRepository, reservedNamespace :=
		storage.reservedBytesLocked(
			reservation.scope,
		)
	if additional > storage.capacity.PendingQuarantineBytes-
		globalUsage.pendingBytes-globalUsage.quarantineBytes-reservedTotal ||
		additional > storage.capacity.DeploymentBytes-
			globalUsage.stableBytes-globalUsage.pendingBytes-
			globalUsage.quarantineBytes-reservedTotal ||
		additional > storage.capacity.RepositoryBytes-
			repositoryUsage.stableBytes-repositoryUsage.pendingBytes-
			reservedRepository ||
		additional > storage.capacity.RepositoryBytes-
			namespaceUsage.stableBytes-namespaceUsage.pendingBytes-
			reservedNamespace {
		return ErrCapacityExceeded
	}
	_, available, err := storage.diskCapacity()
	otherReserved := reservedTotal - reservation.amount
	remainingWrite := actualBytes - writtenBytes
	if err != nil ||
		available > math.MaxInt64 ||
		remainingWrite > int64(available)-otherReserved {
		return ErrCapacityExceeded
	}
	reservation.amount = actualBytes
	return nil
}

type reservationReader struct {
	ctx         context.Context
	storage     *Storage
	reservation *pendingReservation
	reader      io.Reader
	written     int64
}

func (reader *reservationReader) Read(destination []byte) (int, error) {
	count, readErr := reader.reader.Read(destination)
	if count <= 0 {
		return count, readErr
	}
	target := reader.written + int64(count)
	if target > reader.reservation.amount {
		if err := reader.storage.resizePendingReservationLocked(
			reader.ctx,
			reader.reservation,
			target,
			reader.written,
		); err != nil {
			return 0, err
		}
	}
	reader.written = target
	return count, readErr
}

func (storage *Storage) releasePendingReservation(
	reservation *pendingReservation,
) {
	if reservation == nil {
		return
	}
	storage.capacityMutex.Lock()
	defer storage.capacityMutex.Unlock()
	if reservation.released {
		return
	}
	reservation.released = true
	delete(storage.reservations, reservation)
}

func (storage *Storage) reservedBytesLocked(
	scope capacityScope,
) (int64, int64, int64) {
	var total int64
	var repository int64
	var namespace int64
	for reservation := range storage.reservations {
		if reservation.released {
			continue
		}
		total += reservation.amount
		if reservation.scope.tenant == scope.tenant &&
			reservation.scope.repository == scope.repository &&
			reservation.scope.trustDomain == scope.trustDomain {
			repository += reservation.amount
		}
		if reservation.scope == scope {
			namespace += reservation.amount
		}
	}
	return total, repository, namespace
}

func (storage *Storage) capacityUsage(
	ctx context.Context,
	queryer rowQueryer,
	scope capacityScope,
) (capacityUsage, error) {
	var usage capacityUsage
	err := queryer.QueryRowContext(
		ctx,
		`SELECT
    coalesce(sum(object.size_bytes), 0),
    coalesce(sum(CASE WHEN entry.segment = 'PROBATION'
                      THEN object.size_bytes ELSE 0 END), 0),
    coalesce(sum(CASE WHEN entry.segment = 'PROTECTED'
                      THEN object.size_bytes ELSE 0 END), 0),
    count(*)
FROM committed_objects AS object
JOIN storage_entries AS entry
  ON entry.tenant_id = object.tenant_id
 AND entry.namespace_generation = object.namespace_generation
 AND entry.cache_key = object.cache_key
WHERE (? = '' OR (
    object.tenant_id = ?
    AND entry.repository_id = ?
    AND entry.trust_domain = ?
    AND (? = 0 OR object.namespace_generation = ?)
))`,
		scope.repository,
		scope.tenant,
		scope.repository,
		scope.trustDomain,
		scope.namespaceGeneration,
		scope.namespaceGeneration,
	).Scan(
		&usage.stableBytes,
		&usage.probationBytes,
		&usage.protectedBytes,
		&usage.stableObjects,
	)
	if err != nil {
		return capacityUsage{}, err
	}
	err = queryer.QueryRowContext(
		ctx,
		`SELECT coalesce(sum(pending.size_bytes), 0)
FROM pending_objects AS pending
JOIN cache_attempts AS attempt ON attempt.attempt_id = pending.attempt_id
WHERE attempt.state = 'PENDING'
  AND (? = '' OR (
      attempt.tenant_id = ?
      AND attempt.repository_id = ?
      AND attempt.trust_domain = ?
      AND (? = 0 OR attempt.namespace_generation = ?)
  ))`,
		scope.repository,
		scope.tenant,
		scope.repository,
		scope.trustDomain,
		scope.namespaceGeneration,
		scope.namespaceGeneration,
	).Scan(&usage.pendingBytes)
	if err != nil {
		return capacityUsage{}, err
	}
	if scope == (capacityScope{}) {
		err = queryer.QueryRowContext(
			ctx,
			"SELECT coalesce(sum(size_bytes), 0) FROM quarantine_records",
		).Scan(&usage.quarantineBytes)
	}
	return usage, err
}

// CapacitySnapshot returns current logical pool accounting and reservation
// state without exposing tenant, repository, namespace, key, or digest values.
func (storage *Storage) CapacitySnapshot(
	ctx context.Context,
) (CapacitySnapshot, error) {
	if ctx == nil {
		return CapacitySnapshot{}, errors.New(
			"inspect Shared capacity: nil context",
		)
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return CapacitySnapshot{}, err
	}
	defer finish()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()
	return storage.capacitySnapshotLocked(ctx)
}

func (storage *Storage) capacitySnapshotLocked(
	ctx context.Context,
) (CapacitySnapshot, error) {
	storage.accessBatchMutex.Lock()
	accessErr := storage.protectedAccessError
	storage.accessBatchMutex.Unlock()
	if accessErr != nil {
		return CapacitySnapshot{}, fmt.Errorf(
			"persist Shared protected access metadata: %w",
			accessErr,
		)
	}
	usage, err := storage.capacityUsage(
		ctx,
		storage.cache.database,
		capacityScope{},
	)
	if err != nil {
		return CapacitySnapshot{}, err
	}
	storage.capacityMutex.Lock()
	reserved, _, _ := storage.reservedBytesLocked(capacityScope{})
	storage.capacityMutex.Unlock()
	high := percentageBytes(
		storage.capacity.DeploymentBytes,
		storage.capacity.HighWatermarkPercent,
	)
	low := percentageBytes(
		storage.capacity.DeploymentBytes,
		storage.capacity.LowWatermarkPercent,
	)
	return CapacitySnapshot{
		DeploymentBytes:        storage.capacity.DeploymentBytes,
		RepositoryBytes:        storage.capacity.RepositoryBytes,
		PendingQuarantineBytes: storage.capacity.PendingQuarantineBytes,
		HighWatermarkBytes:     high,
		LowWatermarkBytes:      low,
		StableBytes:            usage.stableBytes,
		ProbationBytes:         usage.probationBytes,
		ProtectedBytes:         usage.protectedBytes,
		PendingBytes:           usage.pendingBytes,
		QuarantineBytes:        usage.quarantineBytes,
		ReservedBytes:          reserved,
		StableObjects:          usage.stableObjects,
		AdmissionBlocked: usage.pendingBytes+usage.quarantineBytes+reserved >=
			storage.capacity.PendingQuarantineBytes,
		HighWatermarkReached: usage.stableBytes >= high,
	}, nil
}

// MaintainCapacity expires durable TTL state, enforces byte watermarks and
// SLRU ordering, then physically removes blobs left without authority.
func (storage *Storage) MaintainCapacity(
	ctx context.Context,
) (CapacityMaintenanceReport, error) {
	if ctx == nil {
		return CapacityMaintenanceReport{}, errors.New(
			"maintain Shared capacity: nil context",
		)
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return CapacityMaintenanceReport{}, err
	}
	defer finish()
	storage.reconcileMutex.Lock()
	defer storage.reconcileMutex.Unlock()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()
	return storage.maintainCapacityLocked(ctx, storage.now())
}

func (storage *Storage) maintainCapacityLocked(
	ctx context.Context,
	now time.Time,
) (CapacityMaintenanceReport, error) {
	report := CapacityMaintenanceReport{StartedAt: now}
	expiredAttempts, err := storage.expirePendingAttempts(ctx, now)
	if err != nil {
		return report, err
	}
	report.ExpiredAttempts = expiredAttempts

	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return report, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	before, err := storage.capacityUsage(
		ctx,
		transaction,
		capacityScope{},
	)
	if err != nil {
		return report, err
	}
	report.StableBytesBefore = before.stableBytes
	expired, err := deleteExpiredEntries(ctx, transaction, now)
	if err != nil {
		return report, err
	}
	report.ExpiredObjects = expired

	probation, protected, err := storage.enforceWatermarkTx(
		ctx,
		transaction,
		capacityScope{},
		"",
	)
	if err != nil {
		return report, err
	}
	report.EvictedProbation = probation
	report.EvictedProtected = protected
	demoted, err := storage.enforceProtectedTargetTx(ctx, transaction)
	if err != nil {
		return report, err
	}
	report.DemotedProtected = demoted
	expiredQuarantine, err := storage.expireQuarantineTx(
		ctx,
		transaction,
		now,
	)
	if err != nil {
		return report, err
	}
	report.ExpiredQuarantine = expiredQuarantine
	after, err := storage.capacityUsage(
		ctx,
		transaction,
		capacityScope{},
	)
	if err != nil {
		return report, err
	}
	report.StableBytesAfter = after.stableBytes
	if err := transaction.Commit(); err != nil {
		return report, err
	}
	rollback = false

	storage.blobCleanupPending = storage.blobCleanupPending ||
		report.ExpiredObjects > 0 ||
		report.EvictedProbation > 0 ||
		report.EvictedProtected > 0 ||
		report.ExpiredAttempts > 0
	if storage.blobCleanupPending {
		deleted, err := storage.deleteOrphanBlobs(ctx)
		if err != nil {
			return report, err
		}
		report.DeletedUnreferencedBlob = deleted
		storage.blobCleanupPending = false
	}
	storage.quarantineCleanupPending = storage.quarantineCleanupPending ||
		report.ExpiredQuarantine > 0
	if storage.quarantineCleanupPending {
		if err := storage.deleteExpiredQuarantineFiles(now); err != nil {
			return report, err
		}
		storage.quarantineCleanupPending = false
	}
	report.CompletedAt = storage.now()
	if report.CompletedAt.Before(report.StartedAt) {
		report.CompletedAt = report.StartedAt
	}
	if capacityMaintenanceChanged(report) {
		if err := storage.recordCapacityMaintenance(ctx, report); err != nil {
			return report, err
		}
	}
	return report, nil
}

func capacityMaintenanceChanged(report CapacityMaintenanceReport) bool {
	return report.ExpiredObjects > 0 ||
		report.EvictedProbation > 0 ||
		report.EvictedProtected > 0 ||
		report.DemotedProtected > 0 ||
		report.ExpiredAttempts > 0 ||
		report.ExpiredQuarantine > 0 ||
		report.DeletedUnreferencedBlob > 0
}

func deleteExpiredEntries(
	ctx context.Context,
	transaction *sql.Tx,
	now time.Time,
) (int, error) {
	result, err := transaction.ExecContext(
		ctx,
		`DELETE FROM committed_objects
WHERE EXISTS (
    SELECT 1
    FROM storage_entries AS entry
    WHERE entry.tenant_id = committed_objects.tenant_id
      AND entry.namespace_generation = committed_objects.namespace_generation
      AND entry.cache_key = committed_objects.cache_key
      AND entry.expires_at_unix_ms <= ?
)`,
		now.UnixMilli(),
	)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func (storage *Storage) enforceCommitCapacityTx(
	ctx context.Context,
	transaction *sql.Tx,
	scope capacityScope,
	decisionDigest string,
) (int, int, error) {
	namespaceUsage, err := storage.capacityUsage(
		ctx,
		transaction,
		scope,
	)
	if err != nil {
		return 0, 0, err
	}
	repositoryScope := scope
	repositoryScope.namespaceGeneration = 0
	repositoryUsage, err := storage.capacityUsage(
		ctx,
		transaction,
		repositoryScope,
	)
	if err != nil {
		return 0, 0, err
	}
	globalUsage, err := storage.capacityUsage(
		ctx,
		transaction,
		capacityScope{},
	)
	if err != nil {
		return 0, 0, err
	}
	if namespaceUsage.stableBytes > storage.capacity.RepositoryBytes ||
		repositoryUsage.stableBytes > storage.capacity.RepositoryBytes ||
		globalUsage.stableBytes > storage.capacity.DeploymentBytes {
		return 0, 0, ErrCapacityExceeded
	}
	namespaceProbation, namespaceProtected, err :=
		storage.enforceWatermarkTx(
			ctx,
			transaction,
			scope,
			decisionDigest,
		)
	if err != nil {
		return 0, 0, err
	}
	repositoryProbation, repositoryProtected, err :=
		storage.enforceWatermarkTx(
			ctx,
			transaction,
			repositoryScope,
			decisionDigest,
		)
	if err != nil {
		return 0, 0, err
	}
	globalProbation, globalProtected, err := storage.enforceWatermarkTx(
		ctx,
		transaction,
		capacityScope{},
		decisionDigest,
	)
	return namespaceProbation + repositoryProbation + globalProbation,
		namespaceProtected + repositoryProtected + globalProtected,
		err
}

func (storage *Storage) enforceWatermarkTx(
	ctx context.Context,
	transaction *sql.Tx,
	scope capacityScope,
	excludedDecision string,
) (int, int, error) {
	limit := storage.capacity.DeploymentBytes
	if scope != (capacityScope{}) {
		limit = storage.capacity.RepositoryBytes
	}
	usage, err := storage.capacityUsage(ctx, transaction, scope)
	if err != nil {
		return 0, 0, err
	}
	high := percentageBytes(limit, storage.capacity.HighWatermarkPercent)
	if usage.stableBytes < high {
		return 0, 0, nil
	}
	target := percentageBytes(limit, storage.capacity.LowWatermarkPercent)
	toFree := usage.stableBytes - target
	candidates, err := evictionCandidates(
		ctx,
		transaction,
		scope,
		excludedDecision,
	)
	if err != nil {
		return 0, 0, err
	}
	probation := 0
	protected := 0
	for _, candidate := range candidates {
		if toFree <= 0 {
			break
		}
		result, err := transaction.ExecContext(
			ctx,
			`DELETE FROM committed_objects
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?`,
			candidate.tenant,
			candidate.namespaceGeneration,
			candidate.key,
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
		toFree -= candidate.size
		if candidate.segment == segmentProbation {
			probation++
		} else {
			protected++
		}
	}
	if toFree > 0 {
		return probation, protected, ErrCapacityExceeded
	}
	return probation, protected, nil
}

func evictionCandidates(
	ctx context.Context,
	transaction *sql.Tx,
	scope capacityScope,
	excludedDecision string,
) ([]evictionCandidate, error) {
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT object.tenant_id, object.namespace_generation,
       object.cache_key, object.size_bytes, entry.segment
FROM committed_objects AS object
JOIN storage_entries AS entry
  ON entry.tenant_id = object.tenant_id
 AND entry.namespace_generation = object.namespace_generation
 AND entry.cache_key = object.cache_key
WHERE (? = '' OR (
    object.tenant_id = ?
    AND entry.repository_id = ?
    AND entry.trust_domain = ?
    AND (? = 0 OR object.namespace_generation = ?)
))
  AND (? = '' OR object.decision_digest <> ?)
ORDER BY CASE entry.segment WHEN 'PROBATION' THEN 0 ELSE 1 END,
         object.last_access_unix_ms,
         object.size_bytes DESC,
         object.tenant_id,
         object.namespace_generation,
         object.cache_key`,
		scope.repository,
		scope.tenant,
		scope.repository,
		scope.trustDomain,
		scope.namespaceGeneration,
		scope.namespaceGeneration,
		excludedDecision,
		excludedDecision,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var candidates []evictionCandidate
	for rows.Next() {
		var candidate evictionCandidate
		if err := rows.Scan(
			&candidate.tenant,
			&candidate.namespaceGeneration,
			&candidate.key,
			&candidate.size,
			&candidate.segment,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (storage *Storage) promoteAndBatchAccess(
	ctx context.Context,
	tenant string,
	namespaceGeneration int64,
	key string,
	segment string,
	lastAccess int64,
	now time.Time,
) error {
	if !storage.accessUpdateRequired(segment, lastAccess, now) {
		return nil
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
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE committed_objects
SET last_access_unix_ms = ?
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?`,
		now.UnixMilli(),
		tenant,
		namespaceGeneration,
		key,
	); err != nil {
		return err
	}
	if segment == segmentProbation {
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE storage_entries
SET segment = 'PROTECTED'
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?`,
			tenant,
			namespaceGeneration,
			key,
		); err != nil {
			return err
		}
		if _, err := storage.enforceProtectedTargetTx(
			ctx,
			transaction,
		); err != nil {
			return err
		}
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	rollback = false
	return nil
}

func (storage *Storage) batchProtectedAccess(
	ctx context.Context,
	object CommittedObject,
	now time.Time,
) error {
	key := protectedAccessKey{
		tenant:              object.RepositoryTenant,
		namespaceGeneration: object.NamespaceGeneration,
		key:                 object.Key,
		blobDigest:          object.Blob.Digest,
		sizeBytes:           object.Blob.Size,
		decisionDigest:      object.DecisionDigest,
	}
	storage.accessBatchMutex.Lock()
	batch := storage.currentAccessBatch
	leader := batch == nil
	var finish func()
	if leader {
		batch = &protectedAccessBatch{
			entries: make(map[protectedAccessKey]time.Time),
		}
		storage.currentAccessBatch = batch
		var err error
		finish, err = storage.beginOperation()
		if err != nil {
			storage.currentAccessBatch = nil
			storage.accessBatchMutex.Unlock()
			return err
		}
	}
	if previous, exists := batch.entries[key]; !exists || now.After(previous) {
		batch.entries[key] = now
	}
	storage.accessBatchMutex.Unlock()

	if leader {
		go func() {
			defer finish()
			timer := time.NewTimer(protectedAccessBatchWindow)
			<-timer.C
			storage.accessBatchMutex.Lock()
			if storage.currentAccessBatch == batch {
				storage.currentAccessBatch = nil
			}
			storage.accessBatchMutex.Unlock()
			storage.flushProtectedAccessBatch(batch)
			storage.accessBatchMutex.Lock()
			if batch.err != nil {
				storage.protectedAccessError = errors.Join(
					storage.protectedAccessError,
					batch.err,
				)
			}
			storage.accessBatchMutex.Unlock()
			if storage.testHooks.afterProtectedAccessBatch != nil {
				storage.testHooks.afterProtectedAccessBatch(batch.err)
			}
		}()
	}
	return ctx.Err()
}

func (storage *Storage) flushProtectedAccessBatch(
	batch *protectedAccessBatch,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		protectedAccessBatchTimeout,
	)
	defer cancel()
	// Protected access time is eviction metadata, not cache authority. The
	// conditional transaction serializes with lifecycle writers in SQLite and
	// fails closed if authority, identity, expiry, or segment changed, without
	// forcing unrelated verified readers to drain first.
	if storage.testHooks.beforeProtectedAccessBatch != nil {
		if err := storage.testHooks.beforeProtectedAccessBatch(
			len(batch.entries),
		); err != nil {
			batch.err = err
			return
		}
	}
	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		batch.err = err
		return
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	for key, accessedAt := range batch.entries {
		validAt := storage.now().UnixMilli()
		result, updateErr := transaction.ExecContext(
			ctx,
			`UPDATE committed_objects
SET last_access_unix_ms = ?
WHERE tenant_id = ?
  AND namespace_generation = ?
  AND cache_key = ?
  AND blob_digest = ?
  AND size_bytes = ?
  AND decision_digest = ?
  AND EXISTS (
      SELECT 1
      FROM storage_entries AS entry
      WHERE entry.tenant_id = committed_objects.tenant_id
        AND entry.namespace_generation =
            committed_objects.namespace_generation
        AND entry.cache_key = committed_objects.cache_key
        AND entry.segment = 'PROTECTED'
        AND entry.expires_at_unix_ms > ?
  )`,
			accessedAt.UnixMilli(),
			key.tenant,
			key.namespaceGeneration,
			key.key,
			key.blobDigest,
			key.sizeBytes,
			key.decisionDigest,
			validAt,
		)
		if updateErr != nil {
			batch.err = updateErr
			return
		}
		if _, updateErr := result.RowsAffected(); updateErr != nil {
			batch.err = updateErr
			return
		}
	}
	if err := transaction.Commit(); err != nil {
		batch.err = err
		return
	}
	rollback = false
}

func (storage *Storage) accessUpdateRequired(
	segment string,
	lastAccess int64,
	now time.Time,
) bool {
	return segment == segmentProbation ||
		lastAccess <= now.Add(
			-storage.capacity.AccessUpdateInterval,
		).UnixMilli()
}

func (storage *Storage) enforceProtectedTargetTx(
	ctx context.Context,
	transaction *sql.Tx,
) (int, error) {
	var protectedBytes int64
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT coalesce(sum(object.size_bytes), 0)
FROM committed_objects AS object
JOIN storage_entries AS entry
  ON entry.tenant_id = object.tenant_id
 AND entry.namespace_generation = object.namespace_generation
 AND entry.cache_key = object.cache_key
WHERE entry.segment = 'PROTECTED'`,
	).Scan(&protectedBytes); err != nil {
		return 0, err
	}
	target := percentageBytes(
		percentageBytes(
			storage.capacity.DeploymentBytes,
			storage.capacity.LowWatermarkPercent,
		),
		storage.capacity.ProtectedPercent,
	)
	if protectedBytes <= target {
		return 0, nil
	}
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT object.tenant_id, object.namespace_generation,
       object.cache_key, object.size_bytes
FROM committed_objects AS object
JOIN storage_entries AS entry
  ON entry.tenant_id = object.tenant_id
 AND entry.namespace_generation = object.namespace_generation
 AND entry.cache_key = object.cache_key
WHERE entry.segment = 'PROTECTED'
ORDER BY object.last_access_unix_ms,
         object.tenant_id,
         object.namespace_generation,
         object.cache_key`,
	)
	if err != nil {
		return 0, err
	}
	var candidates []evictionCandidate
	for rows.Next() {
		var candidate evictionCandidate
		if err := rows.Scan(
			&candidate.tenant,
			&candidate.namespaceGeneration,
			&candidate.key,
			&candidate.size,
		); err != nil {
			_ = rows.Close()
			return 0, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	demoted := 0
	for _, candidate := range candidates {
		if protectedBytes <= target {
			break
		}
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE storage_entries
SET segment = 'PROBATION'
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?`,
			candidate.tenant,
			candidate.namespaceGeneration,
			candidate.key,
		); err != nil {
			return demoted, err
		}
		protectedBytes -= candidate.size
		demoted++
	}
	return demoted, nil
}

func (storage *Storage) expireQuarantineTx(
	ctx context.Context,
	transaction *sql.Tx,
	now time.Time,
) (int, error) {
	result, err := transaction.ExecContext(
		ctx,
		`DELETE FROM quarantine_records
WHERE quarantined_at_unix_ms <= ?`,
		now.Add(-storage.capacity.QuarantineTTL).UnixMilli(),
	)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func (storage *Storage) deleteExpiredQuarantineFiles(now time.Time) error {
	entries, err := os.ReadDir(storage.layout.Quarantine)
	if err != nil {
		return err
	}
	cutoff := now.Add(-storage.capacity.QuarantineTTL).UnixMilli()
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("unexpected quarantine directory %q", entry.Name())
		}
		parts := strings.Split(entry.Name(), ".")
		if len(parts) < 3 {
			return fmt.Errorf("unexpected quarantine entry %q", entry.Name())
		}
		millis, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return fmt.Errorf("unexpected quarantine entry %q", entry.Name())
		}
		if millis > cutoff {
			continue
		}
		if err := os.Remove(filepath.Join(storage.layout.Quarantine, entry.Name())); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return syncDirectory(storage.layout.Quarantine)
}

func (storage *Storage) recordCapacityMaintenance(
	ctx context.Context,
	report CapacityMaintenanceReport,
) error {
	_, err := storage.control.database.ExecContext(
		ctx,
		`INSERT INTO storage_maintenance_runs (
    started_at_unix_ms, completed_at_unix_ms, stable_bytes_before,
    stable_bytes_after, expired_objects, evicted_probation,
    evicted_protected, demoted_protected, expired_attempts,
    expired_quarantine, deleted_unreferenced_blobs, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'COMPLETE')`,
		report.StartedAt.UnixMilli(),
		report.CompletedAt.UnixMilli(),
		report.StableBytesBefore,
		report.StableBytesAfter,
		report.ExpiredObjects,
		report.EvictedProbation,
		report.EvictedProtected,
		report.DemotedProtected,
		report.ExpiredAttempts,
		report.ExpiredQuarantine,
		report.DeletedUnreferencedBlob,
	)
	return err
}
