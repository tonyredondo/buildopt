package sharedcache

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const maximumPendingLifetime = 24 * time.Hour

var abortReasons = map[string]struct{}{
	"BUILD_FAILURE":              {},
	"INFRA_FAILURE":              {},
	"CANCELLED":                  {},
	"TIMEOUT":                    {},
	"POLICY_CHANGED":             {},
	"SOURCE_CHANGED":             {},
	"LEASE_EXPIRED":              {},
	"VALIDATION_FAILED":          {},
	"INCOMPLETE_COMMIT_DECISION": {},
}

// AttemptState is the only durable pending-publication lifecycle.
type AttemptState string

const (
	AttemptPending   AttemptState = "PENDING"
	AttemptCommitted AttemptState = "COMMITTED"
	AttemptAborted   AttemptState = "ABORTED"
)

// StartAttemptRequest binds one pending writer to immutable source and policy
// context. RequestID is the start idempotency identity.
type StartAttemptRequest struct {
	RequestID                 string
	AttemptID                 string
	AuthorityDigest           string
	Repository                RepositoryIdentity
	NamespaceGeneration       int64
	SourceRevision            string
	SourceStateDigest         string
	PolicyDigest              string
	ConfigurationPolicyDigest string
	CacheContractDigest       string
	OwnerID                   string
	LeaseID                   string
	LeaseExpiresAt            time.Time
}

// AttemptStatus is the durable state visible to retrying control clients.
type AttemptStatus struct {
	AttemptID                 string
	AuthorityDigest           string
	Repository                RepositoryIdentity
	NamespaceGeneration       int64
	SourceRevision            string
	SourceStateDigest         string
	PolicyDigest              string
	ConfigurationPolicyDigest string
	CacheContractDigest       string
	OwnerID                   string
	LeaseID                   string
	LeaseExpiresAt            time.Time
	State                     AttemptState
	StateVersion              int64
	PendingObjectCount        int
	DecisionDigest            string
	AbortReason               string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// PendingObjectResult describes one durable, still-invisible candidate.
type PendingObjectResult struct {
	Object       CommitObject
	BlobCreated  bool
	ObjectAdded  bool
	StateVersion int64
}

// AbortAttemptRequest is an idempotent preconditioned terminal command.
type AbortAttemptRequest struct {
	RequestID            string
	AttemptID            string
	ExpectedStateVersion int64
	Reason               string
}

// AbortResult distinguishes the first terminal transition from exact replay.
type AbortResult struct {
	Status  AttemptStatus
	Outcome string
}

// CommitResult reports cache authority independently from the repairable
// control audit index.
type CommitResult struct {
	AttemptID         string
	DecisionDigest    string
	Outcome           string
	ObjectCount       int
	StateVersion      int64
	CommittedAt       time.Time
	AuditIndexed      bool
	RequiresReconcile bool
}

// CommittedObject is the metadata accompanying a fully verified hit.
type CommittedObject struct {
	RepositoryTenant    string
	NamespaceGeneration int64
	Key                 string
	Blob                Blob
	DecisionDigest      string
}

type attemptRecord struct {
	status              AttemptStatus
	requestFingerprint  string
	terminalID          sql.NullString
	terminalFingerprint sql.NullString
}

// StartAttempt creates one bounded durable attempt or returns its exact replay.
func (storage *Storage) StartAttempt(
	ctx context.Context,
	request StartAttemptRequest,
) (AttemptStatus, bool, error) {
	if ctx == nil {
		return AttemptStatus{}, false, errors.New(
			"start Shared cache attempt: nil context",
		)
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return AttemptStatus{}, false, err
	}
	defer finish()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()

	now := storage.now()
	request.LeaseExpiresAt = request.LeaseExpiresAt.UTC()
	if err := validateStartAttemptRequest(request, now); err != nil {
		return AttemptStatus{}, false, err
	}
	fingerprint, err := fingerprintValue(request)
	if err != nil {
		return AttemptStatus{}, false, err
	}
	_, err = storage.cache.database.ExecContext(
		ctx,
		`INSERT INTO cache_attempts (
    attempt_id, request_fingerprint, tenant_id, repository_id, trust_domain,
    namespace_generation, source_revision, source_state_digest, policy_digest,
    configuration_policy_digest, cache_contract_digest, owner_id, lease_id,
    lease_expires_at_unix_ms, state, state_version, created_at_unix_ms,
    updated_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', 1, ?, ?)`,
		request.AttemptID,
		fingerprint,
		request.Repository.Tenant,
		request.Repository.Repository,
		request.Repository.TrustDomain,
		request.NamespaceGeneration,
		request.SourceRevision,
		request.SourceStateDigest,
		request.PolicyDigest,
		request.ConfigurationPolicyDigest,
		request.CacheContractDigest,
		request.OwnerID,
		request.LeaseID,
		request.LeaseExpiresAt.UnixMilli(),
		now.UnixMilli(),
		now.UnixMilli(),
	)
	if err != nil {
		existing, loadErr := storage.loadAttempt(
			ctx,
			storage.cache.database,
			request.AttemptID,
		)
		if loadErr != nil {
			return AttemptStatus{}, false, fmt.Errorf(
				"start Shared cache attempt: %w",
				err,
			)
		}
		if existing.requestFingerprint != fingerprint {
			return AttemptStatus{}, false, ErrIdempotencyConflict
		}
		if err := storage.ensureAttemptAuthority(
			ctx,
			request.AttemptID,
			request.AuthorityDigest,
		); err != nil {
			return AttemptStatus{}, false, err
		}
		if request.AuthorityDigest != "" {
			existing.status.AuthorityDigest = request.AuthorityDigest
		}
		return existing.status, false, nil
	}
	record, err := storage.loadAttempt(
		ctx,
		storage.cache.database,
		request.AttemptID,
	)
	if err != nil {
		return AttemptStatus{}, false, err
	}
	if err := storage.ensureAttemptAuthority(
		ctx,
		request.AttemptID,
		request.AuthorityDigest,
	); err != nil {
		return AttemptStatus{}, false, err
	}
	if request.AuthorityDigest != "" {
		record.status.AuthorityDigest = request.AuthorityDigest
	}
	return record.status, true, nil
}

func (storage *Storage) ensureAttemptAuthority(
	ctx context.Context,
	attemptID string,
	authorityDigest string,
) error {
	if authorityDigest == "" {
		return nil
	}
	if _, err := storage.cache.database.ExecContext(
		ctx,
		`INSERT INTO attempt_authorities
    (attempt_id, authority_digest)
VALUES (?, ?)
ON CONFLICT(attempt_id) DO NOTHING`,
		attemptID,
		authorityDigest,
	); err != nil {
		return err
	}
	var current string
	if err := storage.cache.database.QueryRowContext(
		ctx,
		`SELECT authority_digest
FROM attempt_authorities
WHERE attempt_id = ?`,
		attemptID,
	).Scan(&current); err != nil {
		return err
	}
	if current != authorityDigest {
		return ErrIdempotencyConflict
	}
	return nil
}

func validateStartAttemptRequest(
	request StartAttemptRequest,
	now time.Time,
) error {
	for name, value := range map[string]string{
		"request ID":   request.RequestID,
		"attempt ID":   request.AttemptID,
		"tenant":       request.Repository.Tenant,
		"repository":   request.Repository.Repository,
		"trust domain": request.Repository.TrustDomain,
		"owner ID":     request.OwnerID,
		"lease ID":     request.LeaseID,
	} {
		if !validIdentifier(value) {
			return fmt.Errorf("start Shared cache attempt: invalid %s", name)
		}
	}
	if request.NamespaceGeneration < 1 {
		return errors.New(
			"start Shared cache attempt: namespace generation must be positive",
		)
	}
	if request.AuthorityDigest != "" &&
		!validSHA256Digest(request.AuthorityDigest) {
		return errors.New(
			"start Shared cache attempt: invalid local authority digest",
		)
	}
	if !revisionPattern.MatchString(request.SourceRevision) ||
		!validSourceDigest(request.SourceStateDigest) {
		return errors.New("start Shared cache attempt: invalid source binding")
	}
	for _, digest := range []string{
		request.PolicyDigest,
		request.ConfigurationPolicyDigest,
		request.CacheContractDigest,
	} {
		if !validSHA256Digest(digest) {
			return errors.New(
				"start Shared cache attempt: invalid policy binding",
			)
		}
	}
	if !request.LeaseExpiresAt.After(now) ||
		request.LeaseExpiresAt.After(now.Add(maximumPendingLifetime)) {
		return errors.New(
			"start Shared cache attempt: lease must be within 24 hours",
		)
	}
	return nil
}

// AttemptStatus resolves durable state after a retry or unknown response.
func (storage *Storage) AttemptStatus(
	ctx context.Context,
	attemptID string,
) (AttemptStatus, error) {
	if ctx == nil {
		return AttemptStatus{}, errors.New(
			"read Shared cache attempt: nil context",
		)
	}
	if !validIdentifier(attemptID) {
		return AttemptStatus{}, ErrAttemptNotFound
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return AttemptStatus{}, err
	}
	defer finish()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()

	record, err := storage.loadAttempt(
		ctx,
		storage.cache.database,
		attemptID,
	)
	if err != nil {
		return AttemptStatus{}, err
	}
	if record.status.State == AttemptPending &&
		!record.status.LeaseExpiresAt.After(storage.now()) {
		if err := storage.abortExpiredAttempt(
			ctx,
			storage.cache.database,
			record,
			storage.now(),
		); err != nil {
			return AttemptStatus{}, err
		}
		record, err = storage.loadAttempt(
			ctx,
			storage.cache.database,
			attemptID,
		)
		if err != nil {
			return AttemptStatus{}, err
		}
	}
	return record.status, nil
}

// PutPending streams one candidate to immutable storage and only then records
// it beneath a still-pending attempt.
func (storage *Storage) PutPending(
	ctx context.Context,
	attemptID string,
	key string,
	reader io.Reader,
) (PendingObjectResult, error) {
	return storage.PutPendingSized(ctx, attemptID, key, -1, reader)
}

// PutPendingSized reserves a declared body before reading it. A negative size
// is an explicitly unknown length and conservatively reserves the complete
// configured object limit.
func (storage *Storage) PutPendingSized(
	ctx context.Context,
	attemptID string,
	key string,
	declaredBytes int64,
	reader io.Reader,
) (PendingObjectResult, error) {
	if ctx == nil {
		return PendingObjectResult{}, errors.New(
			"put pending Shared object: nil context",
		)
	}
	if !validIdentifier(attemptID) || !validCacheKey(key) {
		return PendingObjectResult{}, errors.New(
			"put pending Shared object: invalid identity",
		)
	}
	if reader == nil {
		return PendingObjectResult{}, errors.New(
			"put pending Shared object: nil reader",
		)
	}
	if declaredBytes < -1 {
		return PendingObjectResult{}, errors.New(
			"put pending Shared object: invalid declared size",
		)
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return PendingObjectResult{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()

	storage.lifecycleMutex.Lock()
	record, err := storage.loadAttempt(
		ctx,
		storage.cache.database,
		attemptID,
	)
	if err != nil {
		storage.lifecycleMutex.Unlock()
		return PendingObjectResult{}, err
	}
	now := storage.now()
	if record.status.State != AttemptPending {
		storage.lifecycleMutex.Unlock()
		return PendingObjectResult{}, ErrAttemptConflict
	}
	if !record.status.LeaseExpiresAt.After(now) {
		err := storage.abortExpiredAttempt(
			ctx,
			storage.cache.database,
			record,
			now,
		)
		storage.lifecycleMutex.Unlock()
		if err != nil {
			return PendingObjectResult{}, err
		}
		return PendingObjectResult{}, ErrAttemptConflict
	}
	generation := record.status.NamespaceGeneration
	reservation, err := storage.reservePendingLocked(
		ctx,
		capacityScope{
			tenant:              record.status.Repository.Tenant,
			repository:          record.status.Repository.Repository,
			trustDomain:         record.status.Repository.TrustDomain,
			namespaceGeneration: record.status.NamespaceGeneration,
		},
		declaredBytes,
	)
	if err != nil {
		storage.lifecycleMutex.Unlock()
		return PendingObjectResult{}, err
	}
	storage.lifecycleMutex.Unlock()
	defer storage.releasePendingReservation(reservation)

	reservedReader := &reservationReader{
		ctx:         ctx,
		storage:     storage,
		reservation: reservation,
		reader:      reader,
	}
	blob, blobCreated, err := storage.blobs.putLocked(ctx, reservedReader)
	if err != nil {
		return PendingObjectResult{}, err
	}
	needsBlobCleanup := blobCreated
	defer func() {
		if !needsBlobCleanup {
			return
		}
		storage.lifecycleMutex.Lock()
		storage.blobCleanupPending = true
		storage.lifecycleMutex.Unlock()
	}()
	if err := storage.resizePendingReservationLocked(
		ctx,
		reservation,
		blob.Size,
		blob.Size,
	); err != nil {
		return PendingObjectResult{}, err
	}
	if storage.testHooks.afterPendingBlob != nil {
		storage.testHooks.afterPendingBlob()
	}

	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()
	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return PendingObjectResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	record, err = storage.loadAttempt(ctx, transaction, attemptID)
	if err != nil {
		return PendingObjectResult{}, err
	}
	now = storage.now()
	if record.status.State != AttemptPending {
		return PendingObjectResult{}, ErrAttemptConflict
	}
	if !record.status.LeaseExpiresAt.After(now) {
		if err := storage.abortExpiredAttempt(
			ctx,
			transaction,
			record,
			now,
		); err != nil {
			return PendingObjectResult{}, err
		}
		if err := transaction.Commit(); err != nil {
			return PendingObjectResult{}, err
		}
		rollback = false
		return PendingObjectResult{}, ErrAttemptConflict
	}

	var existing Blob
	err = transaction.QueryRowContext(
		ctx,
		`SELECT blob_digest, size_bytes
FROM pending_objects
WHERE attempt_id = ? AND cache_key = ?`,
		attemptID,
		key,
	).Scan(&existing.Digest, &existing.Size)
	switch {
	case err == nil:
		if existing != blob {
			return PendingObjectResult{}, ErrAttemptConflict
		}
		if err := transaction.Commit(); err != nil {
			return PendingObjectResult{}, err
		}
		rollback = false
		needsBlobCleanup = false
		return PendingObjectResult{
			Object: CommitObject{
				NamespaceGeneration: generation,
				Key:                 key,
				Checksum:            blob.Digest,
				SizeBytes:           blob.Size,
			},
			BlobCreated:  blobCreated,
			ObjectAdded:  false,
			StateVersion: record.status.StateVersion,
		}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return PendingObjectResult{}, err
	}
	if record.status.PendingObjectCount >= maximumCommitObjects {
		return PendingObjectResult{}, ErrAttemptConflict
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO pending_objects
    (attempt_id, cache_key, blob_digest, size_bytes, created_at_unix_ms)
VALUES (?, ?, ?, ?, ?)`,
		attemptID,
		key,
		blob.Digest,
		blob.Size,
		now.UnixMilli(),
	); err != nil {
		return PendingObjectResult{}, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE cache_attempts
SET state_version = state_version + 1, updated_at_unix_ms = ?
WHERE attempt_id = ? AND state = 'PENDING'`,
		now.UnixMilli(),
		attemptID,
	); err != nil {
		return PendingObjectResult{}, err
	}
	if err := transaction.Commit(); err != nil {
		return PendingObjectResult{}, err
	}
	rollback = false
	needsBlobCleanup = false
	return PendingObjectResult{
		Object: CommitObject{
			NamespaceGeneration: generation,
			Key:                 key,
			Checksum:            blob.Digest,
			SizeBytes:           blob.Size,
		},
		BlobCreated:  blobCreated,
		ObjectAdded:  true,
		StateVersion: record.status.StateVersion + 1,
	}, nil
}

// AbortAttempt durably releases all pending candidates without making any
// object visible.
func (storage *Storage) AbortAttempt(
	ctx context.Context,
	request AbortAttemptRequest,
) (AbortResult, error) {
	if ctx == nil {
		return AbortResult{}, errors.New(
			"abort Shared cache attempt: nil context",
		)
	}
	if !validIdentifier(request.RequestID) ||
		!validIdentifier(request.AttemptID) ||
		request.ExpectedStateVersion < 1 {
		return AbortResult{}, errors.New(
			"abort Shared cache attempt: invalid request",
		)
	}
	if _, ok := abortReasons[request.Reason]; !ok {
		return AbortResult{}, errors.New(
			"abort Shared cache attempt: invalid reason",
		)
	}
	fingerprint, err := fingerprintValue(request)
	if err != nil {
		return AbortResult{}, err
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return AbortResult{}, err
	}
	defer finish()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()

	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return AbortResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	record, err := storage.loadAttempt(ctx, transaction, request.AttemptID)
	if err != nil {
		return AbortResult{}, err
	}
	if record.status.State == AttemptAborted {
		if record.terminalID.String != request.RequestID ||
			record.terminalFingerprint.String != fingerprint {
			return AbortResult{}, ErrIdempotencyConflict
		}
		if err := transaction.Commit(); err != nil {
			return AbortResult{}, err
		}
		rollback = false
		return AbortResult{
			Status:  record.status,
			Outcome: "ALREADY_ABORTED",
		}, nil
	}
	if record.status.State != AttemptPending {
		return AbortResult{}, ErrAttemptConflict
	}
	if record.status.StateVersion != request.ExpectedStateVersion {
		return AbortResult{}, ErrStatePrecondition
	}
	now := storage.now()
	if err := storage.abortAttemptTx(
		ctx,
		transaction,
		record,
		request.RequestID,
		fingerprint,
		request.Reason,
		now,
	); err != nil {
		return AbortResult{}, err
	}
	if err := transaction.Commit(); err != nil {
		return AbortResult{}, err
	}
	rollback = false
	if record.status.PendingObjectCount > 0 {
		storage.blobCleanupPending = true
	}
	record, err = storage.loadAttempt(
		ctx,
		storage.cache.database,
		request.AttemptID,
	)
	if err != nil {
		return AbortResult{}, err
	}
	return AbortResult{Status: record.status, Outcome: "ABORTED"}, nil
}

// CommitAttempt performs decision persistence and every visibility CAS in one
// cache.sqlite transaction, then independently indexes the immutable digest.
func (storage *Storage) CommitAttempt(
	ctx context.Context,
	expectedStateVersion int64,
	currentRevocationEpoch int64,
	verified VerifiedCommitDecision,
) (CommitResult, error) {
	if ctx == nil {
		return CommitResult{}, errors.New(
			"commit Shared cache attempt: nil context",
		)
	}
	if expectedStateVersion < 1 ||
		len(verified.canonical) == 0 {
		return CommitResult{}, ErrCommitRejected
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return CommitResult{}, err
	}
	defer finish()
	storage.lifecycleMutex.Lock()
	defer storage.lifecycleMutex.Unlock()

	now := storage.now()
	decision := verified.decision
	record, err := storage.loadAttempt(
		ctx,
		storage.cache.database,
		decision.AttemptID,
	)
	if err != nil {
		return CommitResult{}, err
	}
	if record.status.State == AttemptCommitted {
		return storage.replayCommit(ctx, record, verified)
	}
	if record.status.State == AttemptAborted &&
		record.terminalID.String == decision.DecisionID {
		if record.terminalFingerprint.String !=
			canonicalDocumentFingerprint(verified.canonical) {
			return CommitResult{}, ErrIdempotencyConflict
		}
		if record.status.AbortReason == "CAS_LOST" {
			return CommitResult{}, ErrCASLost
		}
		return CommitResult{}, ErrCommitRejected
	}
	if record.status.State != AttemptPending {
		return CommitResult{}, ErrAttemptConflict
	}
	if record.status.StateVersion != expectedStateVersion {
		return CommitResult{}, ErrStatePrecondition
	}
	if decision.RevocationEpoch != currentRevocationEpoch {
		if abortErr := storage.abortRejectedAttempt(
			ctx,
			record,
			verified,
			"POLICY_CHANGED",
			now,
		); abortErr != nil {
			return CommitResult{}, fmt.Errorf(
				"abort stale Shared cache commit: %w",
				abortErr,
			)
		}
		return CommitResult{}, ErrCommitRejected
	}
	if !record.status.LeaseExpiresAt.After(now) ||
		!verified.expiresAt.After(now) ||
		verified.issuedAt.Before(record.status.CreatedAt) ||
		verified.expiresAt.After(record.status.LeaseExpiresAt) {
		if abortErr := storage.abortRejectedAttempt(
			ctx,
			record,
			verified,
			"INCOMPLETE_COMMIT_DECISION",
			now,
		); abortErr != nil {
			return CommitResult{}, fmt.Errorf(
				"abort rejected Shared cache commit: %w",
				abortErr,
			)
		}
		return CommitResult{}, ErrCommitRejected
	}
	pending, err := storage.pendingObjects(
		ctx,
		storage.cache.database,
		record,
	)
	if err != nil {
		return CommitResult{}, err
	}
	if err := validateDecisionAgainstAttempt(
		decision,
		record.status,
		pending,
	); err != nil {
		if abortErr := storage.abortRejectedAttempt(
			ctx,
			record,
			verified,
			"INCOMPLETE_COMMIT_DECISION",
			now,
		); abortErr != nil {
			return CommitResult{}, fmt.Errorf(
				"abort rejected Shared cache commit: %w",
				abortErr,
			)
		}
		return CommitResult{}, err
	}
	for _, object := range pending {
		file, verifyErr := storage.blobs.openVerified(
			ctx,
			Blob{Digest: object.Checksum, Size: object.SizeBytes},
		)
		if file != nil {
			_ = file.Close()
		}
		if verifyErr != nil {
			if quarantineErr := storage.quarantinePendingFailure(
				ctx,
				record,
				verified,
				object,
				verifyErr,
				now,
			); quarantineErr != nil {
				return CommitResult{}, fmt.Errorf(
					"quarantine rejected Shared cache commit: %w",
					quarantineErr,
				)
			}
			return CommitResult{}, fmt.Errorf(
				"%w: pending object verification failed",
				ErrCommitRejected,
			)
		}
	}

	transaction, err := storage.cache.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return CommitResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	record, err = storage.loadAttempt(ctx, transaction, decision.AttemptID)
	if err != nil {
		return CommitResult{}, err
	}
	if record.status.State != AttemptPending ||
		record.status.StateVersion != expectedStateVersion {
		return CommitResult{}, ErrStatePrecondition
	}
	pending, err = storage.pendingObjects(ctx, transaction, record)
	if err != nil {
		return CommitResult{}, err
	}
	if err := validateDecisionAgainstAttempt(
		decision,
		record.status,
		pending,
	); err != nil {
		return CommitResult{}, err
	}
	for _, object := range pending {
		var exists int
		if err := transaction.QueryRowContext(
			ctx,
			`SELECT EXISTS (
    SELECT 1
    FROM committed_objects
    WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?
)`,
			record.status.Repository.Tenant,
			object.NamespaceGeneration,
			object.Key,
		).Scan(&exists); err != nil {
			return CommitResult{}, err
		}
		if exists != 0 {
			if err := storage.abortAttemptTx(
				ctx,
				transaction,
				record,
				decision.DecisionID,
				canonicalDocumentFingerprint(verified.canonical),
				"CAS_LOST",
				now,
			); err != nil {
				return CommitResult{}, err
			}
			if err := transaction.Commit(); err != nil {
				return CommitResult{}, err
			}
			rollback = false
			if record.status.PendingObjectCount > 0 {
				storage.blobCleanupPending = true
			}
			return CommitResult{}, ErrCASLost
		}
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO commit_decisions
    (decision_digest, attempt_id, canonical_document, committed_at_unix_ms)
VALUES (?, ?, ?, ?)`,
		decision.DecisionDigest,
		decision.AttemptID,
		verified.canonical,
		now.UnixMilli(),
	); err != nil {
		return CommitResult{}, err
	}
	for _, object := range pending {
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO committed_objects (
    tenant_id, namespace_generation, cache_key, blob_digest, size_bytes,
    decision_digest, committed_at_unix_ms, last_access_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			record.status.Repository.Tenant,
			object.NamespaceGeneration,
			object.Key,
			object.Checksum,
			object.SizeBytes,
			decision.DecisionDigest,
			now.UnixMilli(),
			now.UnixMilli(),
		); err != nil {
			return CommitResult{}, err
		}
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO storage_entries (
    tenant_id, namespace_generation, cache_key, repository_id, trust_domain,
    expires_at_unix_ms, segment
) VALUES (?, ?, ?, ?, ?, ?, 'PROBATION')`,
			record.status.Repository.Tenant,
			object.NamespaceGeneration,
			object.Key,
			record.status.Repository.Repository,
			record.status.Repository.TrustDomain,
			now.Add(storage.capacity.StableTTL).UnixMilli(),
		); err != nil {
			return CommitResult{}, err
		}
	}
	evictedProbation, evictedProtected, err :=
		storage.enforceCommitCapacityTx(
			ctx,
			transaction,
			capacityScope{
				tenant:              record.status.Repository.Tenant,
				repository:          record.status.Repository.Repository,
				trustDomain:         record.status.Repository.TrustDomain,
				namespaceGeneration: record.status.NamespaceGeneration,
			},
			decision.DecisionDigest,
		)
	if err != nil {
		return CommitResult{}, err
	}
	if evictedProbation > 0 || evictedProtected > 0 {
		storage.blobCleanupPending = true
	}
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE cache_attempts
SET state = 'COMMITTED',
    state_version = state_version + 1,
    terminal_id = ?,
    terminal_fingerprint = ?,
    decision_digest = ?,
    updated_at_unix_ms = ?
WHERE attempt_id = ? AND state = 'PENDING' AND state_version = ?`,
		decision.DecisionID,
		canonicalDocumentFingerprint(verified.canonical),
		decision.DecisionDigest,
		now.UnixMilli(),
		decision.AttemptID,
		expectedStateVersion,
	); err != nil {
		return CommitResult{}, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM pending_objects WHERE attempt_id = ?",
		decision.AttemptID,
	); err != nil {
		return CommitResult{}, err
	}
	if storage.testHooks.beforeCacheCommit != nil {
		if err := storage.testHooks.beforeCacheCommit(); err != nil {
			return CommitResult{}, err
		}
	}
	if err := transaction.Commit(); err != nil {
		return CommitResult{}, err
	}
	rollback = false

	auditIndexed := storage.indexDecision(
		ctx,
		decision.DecisionDigest,
		now,
	) == nil
	return CommitResult{
		AttemptID:         decision.AttemptID,
		DecisionDigest:    decision.DecisionDigest,
		Outcome:           "COMMITTED",
		ObjectCount:       len(pending),
		StateVersion:      expectedStateVersion + 1,
		CommittedAt:       now,
		AuditIndexed:      auditIndexed,
		RequiresReconcile: !auditIndexed,
	}, nil
}

func validateDecisionAgainstAttempt(
	decision CommitDecision,
	status AttemptStatus,
	pending []CommitObject,
) error {
	if decision.AttemptID != status.AttemptID ||
		decision.Repository != status.Repository ||
		decision.SourceRevision != status.SourceRevision ||
		decision.SourceStateDigest != status.SourceStateDigest ||
		decision.PolicyDigest != status.PolicyDigest ||
		decision.ConfigurationPolicyDigest !=
			status.ConfigurationPolicyDigest ||
		decision.CacheContractDigest != status.CacheContractDigest ||
		len(pending) == 0 ||
		!sameCommitObjects(decision.Objects, pending) {
		return fmt.Errorf(
			"%w: attempt binding or object coverage mismatch",
			ErrCommitRejected,
		)
	}
	for _, object := range pending {
		if object.NamespaceGeneration != status.NamespaceGeneration {
			return fmt.Errorf(
				"%w: namespace generation mismatch",
				ErrCommitRejected,
			)
		}
	}
	return nil
}

func (storage *Storage) replayCommit(
	ctx context.Context,
	record attemptRecord,
	verified VerifiedCommitDecision,
) (CommitResult, error) {
	var (
		canonical   []byte
		committedAt int64
		objectCount int
	)
	if err := storage.cache.database.QueryRowContext(
		ctx,
		`SELECT canonical_document, committed_at_unix_ms
FROM commit_decisions
WHERE attempt_id = ?`,
		record.status.AttemptID,
	).Scan(&canonical, &committedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CommitResult{}, ErrCommitRejected
		}
		return CommitResult{}, err
	}
	if !bytes.Equal(canonical, verified.canonical) {
		return CommitResult{}, ErrIdempotencyConflict
	}
	if err := storage.cache.database.QueryRowContext(
		ctx,
		`SELECT count(*)
FROM committed_objects
WHERE decision_digest = ?`,
		verified.decision.DecisionDigest,
	).Scan(&objectCount); err != nil {
		return CommitResult{}, err
	}
	now := storage.now()
	auditIndexed := storage.indexDecision(
		ctx,
		verified.decision.DecisionDigest,
		now,
	) == nil
	return CommitResult{
		AttemptID:         record.status.AttemptID,
		DecisionDigest:    verified.decision.DecisionDigest,
		Outcome:           "IDEMPOTENT_REPLAY",
		ObjectCount:       objectCount,
		StateVersion:      record.status.StateVersion,
		CommittedAt:       time.UnixMilli(committedAt).UTC(),
		AuditIndexed:      auditIndexed,
		RequiresReconcile: !auditIndexed,
	}, nil
}

// OpenCommitted returns only a complete verified object authorized by durable
// cache metadata. Presence of a blob is never sufficient.
func (storage *Storage) OpenCommitted(
	ctx context.Context,
	tenant string,
	namespaceGeneration int64,
	key string,
) (*os.File, CommittedObject, error) {
	if ctx == nil {
		return nil, CommittedObject{}, errors.New(
			"open committed Shared object: nil context",
		)
	}
	if !validIdentifier(tenant) ||
		namespaceGeneration < 1 ||
		!validCacheKey(key) {
		return nil, CommittedObject{}, ErrCacheMiss
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return nil, CommittedObject{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()

	storage.lifecycleMutex.RLock()
	object, segment, lastAccess, expiresAt, now, err :=
		storage.loadCommittedObjectReadLocked(
			ctx,
			tenant,
			namespaceGeneration,
			key,
		)
	if err != nil {
		storage.lifecycleMutex.RUnlock()
		return nil, CommittedObject{}, err
	}
	if expiresAt <= now.UnixMilli() {
		storage.lifecycleMutex.RUnlock()
		storage.lifecycleMutex.Lock()
		err := storage.deleteExpiredCommittedObjectLocked(
			ctx,
			tenant,
			namespaceGeneration,
			key,
			now,
		)
		storage.lifecycleMutex.Unlock()
		if err != nil {
			return nil, CommittedObject{}, err
		}
		return nil, CommittedObject{}, ErrCacheMiss
	}
	if storage.testHooks.beforeCommittedBlobVerify != nil {
		storage.testHooks.beforeCommittedBlobVerify()
	}
	file, err := storage.blobs.openVerified(ctx, object.Blob)
	if err != nil {
		storage.lifecycleMutex.RUnlock()
		storage.lifecycleMutex.Lock()
		_, _ = storage.invalidateDecision(
			ctx,
			object.DecisionDigest,
			[]invalidBlob{{
				blob: object.Blob,
				err:  err,
			}},
			storage.now(),
		)
		storage.lifecycleMutex.Unlock()
		return nil, CommittedObject{}, ErrCacheMiss
	}
	if !storage.accessUpdateRequired(segment, lastAccess, now) {
		storage.lifecycleMutex.RUnlock()
		return file, object, nil
	}
	storage.lifecycleMutex.RUnlock()
	if segment == segmentProtected {
		if err := storage.batchProtectedAccess(ctx, object, now); err != nil {
			_ = file.Close()
			return nil, CommittedObject{}, err
		}
		return file, object, nil
	}

	storage.lifecycleMutex.Lock()
	current, segment, lastAccess, expiresAt, now, err :=
		storage.loadCommittedObjectReadLocked(
			ctx,
			tenant,
			namespaceGeneration,
			key,
		)
	if err == nil && expiresAt <= now.UnixMilli() {
		err = storage.deleteExpiredCommittedObjectLocked(
			ctx,
			tenant,
			namespaceGeneration,
			key,
			now,
		)
		if err == nil {
			err = ErrCacheMiss
		}
	}
	if err == nil && current != object {
		err = ErrCacheMiss
	}
	if err == nil {
		err = storage.promoteAndBatchAccess(
			ctx,
			tenant,
			namespaceGeneration,
			key,
			segment,
			lastAccess,
			now,
		)
	}
	storage.lifecycleMutex.Unlock()
	if err != nil {
		_ = file.Close()
		return nil, CommittedObject{}, err
	}
	return file, object, nil
}

func (storage *Storage) loadCommittedObjectReadLocked(
	ctx context.Context,
	tenant string,
	namespaceGeneration int64,
	key string,
) (CommittedObject, string, int64, int64, time.Time, error) {
	object := CommittedObject{
		RepositoryTenant:    tenant,
		NamespaceGeneration: namespaceGeneration,
		Key:                 key,
	}
	var (
		expiresAt  int64
		segment    string
		lastAccess int64
	)
	if err := storage.cache.database.QueryRowContext(
		ctx,
		`SELECT object.blob_digest, object.size_bytes, object.decision_digest,
       entry.expires_at_unix_ms, entry.segment, object.last_access_unix_ms
FROM committed_objects AS object
JOIN commit_decisions AS decision
    ON decision.decision_digest = object.decision_digest
JOIN storage_entries AS entry
    ON entry.tenant_id = object.tenant_id
   AND entry.namespace_generation = object.namespace_generation
   AND entry.cache_key = object.cache_key
WHERE object.tenant_id = ?
  AND object.namespace_generation = ?
  AND object.cache_key = ?`,
		tenant,
		namespaceGeneration,
		key,
	).Scan(
		&object.Blob.Digest,
		&object.Blob.Size,
		&object.DecisionDigest,
		&expiresAt,
		&segment,
		&lastAccess,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CommittedObject{}, "", 0, 0, time.Time{}, ErrCacheMiss
		}
		return CommittedObject{}, "", 0, 0, time.Time{}, err
	}
	now := storage.now()
	return object, segment, lastAccess, expiresAt, now, nil
}

func (storage *Storage) deleteExpiredCommittedObjectLocked(
	ctx context.Context,
	tenant string,
	namespaceGeneration int64,
	key string,
	now time.Time,
) error {
	result, err := storage.cache.database.ExecContext(
		ctx,
		`DELETE FROM committed_objects
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?
  AND EXISTS (
      SELECT 1
      FROM storage_entries
      WHERE tenant_id = ?
        AND namespace_generation = ?
        AND cache_key = ?
        AND expires_at_unix_ms <= ?
  )`,
		tenant,
		namespaceGeneration,
		key,
		tenant,
		namespaceGeneration,
		key,
		now.UnixMilli(),
	)
	if err != nil {
		return err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if deleted > 0 {
		storage.blobCleanupPending = true
	}
	return nil
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (storage *Storage) loadAttempt(
	ctx context.Context,
	queryer rowQueryer,
	attemptID string,
) (attemptRecord, error) {
	var (
		record          attemptRecord
		leaseExpiresAt  int64
		createdAt       int64
		updatedAt       int64
		state           string
		decisionDigest  sql.NullString
		abortReason     sql.NullString
		authorityDigest sql.NullString
	)
	err := queryer.QueryRowContext(
		ctx,
		`SELECT
    attempt_id, request_fingerprint, tenant_id, repository_id, trust_domain,
    namespace_generation, source_revision, source_state_digest, policy_digest,
    configuration_policy_digest, cache_contract_digest, owner_id, lease_id,
    lease_expires_at_unix_ms, state, state_version, terminal_id,
    terminal_fingerprint, decision_digest, abort_reason, created_at_unix_ms,
    updated_at_unix_ms,
    (SELECT count(*) FROM pending_objects WHERE attempt_id = cache_attempts.attempt_id),
    (SELECT authority_digest
     FROM attempt_authorities
     WHERE attempt_id = cache_attempts.attempt_id)
FROM cache_attempts
WHERE attempt_id = ?`,
		attemptID,
	).Scan(
		&record.status.AttemptID,
		&record.requestFingerprint,
		&record.status.Repository.Tenant,
		&record.status.Repository.Repository,
		&record.status.Repository.TrustDomain,
		&record.status.NamespaceGeneration,
		&record.status.SourceRevision,
		&record.status.SourceStateDigest,
		&record.status.PolicyDigest,
		&record.status.ConfigurationPolicyDigest,
		&record.status.CacheContractDigest,
		&record.status.OwnerID,
		&record.status.LeaseID,
		&leaseExpiresAt,
		&state,
		&record.status.StateVersion,
		&record.terminalID,
		&record.terminalFingerprint,
		&decisionDigest,
		&abortReason,
		&createdAt,
		&updatedAt,
		&record.status.PendingObjectCount,
		&authorityDigest,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return attemptRecord{}, ErrAttemptNotFound
	}
	if err != nil {
		return attemptRecord{}, err
	}
	record.status.State = AttemptState(state)
	record.status.LeaseExpiresAt = time.UnixMilli(leaseExpiresAt).UTC()
	record.status.CreatedAt = time.UnixMilli(createdAt).UTC()
	record.status.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	record.status.DecisionDigest = decisionDigest.String
	record.status.AbortReason = abortReason.String
	record.status.AuthorityDigest = authorityDigest.String
	return record, nil
}

func (storage *Storage) pendingObjects(
	ctx context.Context,
	queryer interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	},
	record attemptRecord,
) ([]CommitObject, error) {
	rows, err := queryer.QueryContext(
		ctx,
		`SELECT cache_key, blob_digest, size_bytes
FROM pending_objects
WHERE attempt_id = ?
ORDER BY cache_key`,
		record.status.AttemptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects []CommitObject
	for rows.Next() {
		object := CommitObject{
			NamespaceGeneration: record.status.NamespaceGeneration,
		}
		if err := rows.Scan(
			&object.Key,
			&object.Checksum,
			&object.SizeBytes,
		); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, rows.Err()
}

func (storage *Storage) abortAttemptTx(
	ctx context.Context,
	executor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	},
	record attemptRecord,
	terminalID string,
	terminalFingerprint string,
	reason string,
	now time.Time,
) error {
	result, err := executor.ExecContext(
		ctx,
		`UPDATE cache_attempts
SET state = 'ABORTED',
    state_version = state_version + 1,
    terminal_id = ?,
    terminal_fingerprint = ?,
    abort_reason = ?,
    updated_at_unix_ms = ?
WHERE attempt_id = ? AND state = 'PENDING' AND state_version = ?`,
		terminalID,
		terminalFingerprint,
		reason,
		now.UnixMilli(),
		record.status.AttemptID,
		record.status.StateVersion,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrStatePrecondition
	}
	_, err = executor.ExecContext(
		ctx,
		"DELETE FROM pending_objects WHERE attempt_id = ?",
		record.status.AttemptID,
	)
	return err
}

func (storage *Storage) abortExpiredAttempt(
	ctx context.Context,
	executor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	},
	record attemptRecord,
	now time.Time,
) error {
	fingerprint, err := fingerprintValue(struct {
		AttemptID string `json:"attemptId"`
		Reason    string `json:"reason"`
	}{
		AttemptID: record.status.AttemptID,
		Reason:    "LEASE_EXPIRED",
	})
	if err != nil {
		return err
	}
	return storage.abortAttemptTx(
		ctx,
		executor,
		record,
		"reconcile:lease-expired",
		fingerprint,
		"LEASE_EXPIRED",
		now,
	)
}

func (storage *Storage) abortRejectedAttempt(
	ctx context.Context,
	record attemptRecord,
	verified VerifiedCommitDecision,
	reason string,
	now time.Time,
) error {
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
	if err := storage.abortAttemptTx(
		ctx,
		transaction,
		record,
		verified.decision.DecisionID,
		canonicalDocumentFingerprint(verified.canonical),
		reason,
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

func (storage *Storage) indexDecision(
	ctx context.Context,
	decisionDigest string,
	now time.Time,
) error {
	if storage.testHooks.beforeControlIndex != nil {
		if err := storage.testHooks.beforeControlIndex(); err != nil {
			return err
		}
	}
	_, err := storage.control.database.ExecContext(
		ctx,
		`INSERT INTO decision_audit_index
    (decision_digest, indexed_at_unix_ms)
VALUES (?, ?)
ON CONFLICT(decision_digest) DO NOTHING`,
		decisionDigest,
		now.UnixMilli(),
	)
	return err
}
