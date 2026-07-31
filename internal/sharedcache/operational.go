package sharedcache

import (
	"context"
	"errors"
	"strings"
	"time"
)

// OperationalSnapshot is a bounded, non-sensitive view of the single-node
// storage signals needed by the private-beta alert surface.
type OperationalSnapshot struct {
	CapturedAt             time.Time
	DiskTotalBytes         uint64
	DiskAvailableBytes     uint64
	DiskProbeSucceeded     bool
	PendingAttempts        int64
	ExpiredPendingAttempts int64
	QuarantineRecords      int64
	IntegrityHealthy       bool
	SQLiteProbeSucceeded   bool
	SQLiteProbeDuration    time.Duration
	StableLogicalBytes     int64
	PendingPoolBytes       int64
	CapacityHighWatermark  bool
	CapacityAdmissionBlock bool
}

// OperationalSnapshot probes disk, lifecycle, and metadata health without
// exposing tenant, repository, key, digest, or credential material.
func (storage *Storage) OperationalSnapshot(
	ctx context.Context,
) (OperationalSnapshot, error) {
	if ctx == nil {
		return OperationalSnapshot{}, errors.New(
			"inspect Shared operational state: nil context",
		)
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return OperationalSnapshot{}, err
	}
	defer finish()

	snapshot := OperationalSnapshot{
		CapturedAt:       storage.now(),
		IntegrityHealthy: true,
	}
	total, available, err := storage.diskCapacity()
	if err == nil {
		snapshot.DiskTotalBytes = total
		snapshot.DiskAvailableBytes = available
		snapshot.DiskProbeSucceeded = total > 0 && available <= total
	}

	probeStarted := time.Now()
	var expired int64
	probeErr := storage.cache.database.QueryRowContext(
		ctx,
		`SELECT count(*),
       coalesce(sum(CASE WHEN lease_expires_at_unix_ms <= ? THEN 1 ELSE 0 END), 0)
FROM cache_attempts
WHERE state = 'PENDING'`,
		snapshot.CapturedAt.UnixMilli(),
	).Scan(&snapshot.PendingAttempts, &expired)
	if probeErr == nil {
		snapshot.ExpiredPendingAttempts = expired
		probeErr = storage.cache.database.QueryRowContext(
			ctx,
			"SELECT count(*) FROM quarantine_records",
		).Scan(&snapshot.QuarantineRecords)
	}
	if probeErr == nil {
		probeErr = storage.cache.integrityCheck(ctx)
	}
	if probeErr == nil {
		probeErr = storage.control.integrityCheck(ctx)
	}
	if probeErr == nil {
		storage.lifecycleMutex.Lock()
		capacity, capacityErr := storage.capacitySnapshotLocked(ctx)
		storage.lifecycleMutex.Unlock()
		if capacityErr != nil {
			probeErr = capacityErr
		} else {
			snapshot.StableLogicalBytes = capacity.StableBytes
			snapshot.PendingPoolBytes = capacity.PendingBytes +
				capacity.QuarantineBytes + capacity.ReservedBytes
			snapshot.CapacityHighWatermark = capacity.HighWatermarkReached
			snapshot.CapacityAdmissionBlock = capacity.AdmissionBlocked
		}
	}
	snapshot.SQLiteProbeDuration = time.Since(probeStarted)
	snapshot.SQLiteProbeSucceeded = probeErr == nil
	if probeErr != nil && !isSQLiteContention(probeErr) {
		snapshot.IntegrityHealthy = false
	}
	return snapshot, nil
}

func isSQLiteContention(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database is busy") ||
		strings.Contains(message, "sqlite_busy")
}
