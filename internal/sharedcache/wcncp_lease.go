package sharedcache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

var (
	// ErrWCNCPLeaseHeld means another validator owns the active proposal lease.
	ErrWCNCPLeaseHeld = errors.New("BuildOpt WCNCP proposal lease is held")
	// ErrWCNCPLeaseLost means the lease expired, was released, or never existed.
	ErrWCNCPLeaseLost = errors.New("BuildOpt WCNCP proposal lease was lost")
)

// WCNCPLease is one active validation claim. Identity binds repository,
// proposal digest, protocol version, and environment class.
type WCNCPLease struct {
	LeaseID               string
	RepositoryScopeSHA256 string
	ProposalDigest        string
	ProtocolVersion       string
	EnvironmentClass      string
	Holder                string
	Attempt               int64
	StartUnixMs           int64
	ExpiryUnixMs          int64
}

func wcncpLeaseID(repo, proposal, protocol, env string, attempt int64) string {
	digest := sha256.Sum256([]byte("wcncp-lease-v1\x00" + repo + "\x00" + proposal + "\x00" + protocol + "\x00" + env + "\x00" + string(rune(attempt))))
	return hex.EncodeToString(digest[:])
}

// ClaimWCNCPLease acquires the single active lease for one proposal. Competing
// validators from the same generation yield one winner and one held error.
func (storage *Storage) ClaimWCNCPLease(ctx context.Context, repositoryScopeSHA256, proposalDigest, protocolVersion, environmentClass, holder string, ttl time.Duration, now time.Time) (WCNCPLease, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) || !validSHA256(proposalDigest) || len(protocolVersion) == 0 || len(protocolVersion) > 64 || len(holder) == 0 || len(holder) > 128 || ttl <= 0 || now.IsZero() {
		return WCNCPLease{}, ErrWCNCPInvalid
	}
	if environmentClass != "CONTROLLED_PERFORMANCE" && environmentClass != "STANDARD_HOSTED_CI" && environmentClass != "LOCAL_FUNCTIONAL" {
		return WCNCPLease{}, ErrWCNCPInvalid
	}
	now = now.UTC()
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPLease{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	// Reclaim visibly expired leases before claiming so loss is observable.
	if _, err := storage.expireWCNCPLeasesTx(ctx, now); err != nil {
		return WCNCPLease{}, err
	}
	var attemptNull sql.NullInt64
	err = storage.state.database.QueryRowContext(ctx, `SELECT max(attempt) FROM wcncp_leases
WHERE repository_scope_sha256 = ? AND proposal_digest = ? AND protocol_version = ? AND environment_class = ?`,
		repositoryScopeSHA256, proposalDigest, protocolVersion, environmentClass).Scan(&attemptNull)
	if err != nil {
		return WCNCPLease{}, err
	}
	var attempt int64 = 1
	if attemptNull.Valid {
		attempt = attemptNull.Int64 + 1
	}
	lease := WCNCPLease{
		RepositoryScopeSHA256: repositoryScopeSHA256, ProposalDigest: proposalDigest,
		ProtocolVersion: protocolVersion, EnvironmentClass: environmentClass,
		Holder: holder, Attempt: attempt,
		StartUnixMs: now.UnixMilli(), ExpiryUnixMs: now.Add(ttl).UnixMilli(),
	}
	lease.LeaseID = wcncpLeaseID(repositoryScopeSHA256, proposalDigest, protocolVersion, environmentClass, attempt)
	_, err = storage.state.database.ExecContext(ctx, `INSERT INTO wcncp_leases (
    lease_id, repository_scope_sha256, proposal_digest, protocol_version,
    environment_class, holder, attempt, start_unix_ms, expiry_unix_ms,
    heartbeat_unix_ms, state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE')`,
		lease.LeaseID, lease.RepositoryScopeSHA256, lease.ProposalDigest, lease.ProtocolVersion,
		lease.EnvironmentClass, lease.Holder, lease.Attempt, lease.StartUnixMs, lease.ExpiryUnixMs, lease.StartUnixMs)
	if err != nil {
		if containsUniqueViolation(err) {
			return WCNCPLease{}, ErrWCNCPLeaseHeld
		}
		return WCNCPLease{}, err
	}
	return lease, nil
}

// HeartbeatWCNCPLease extends only the same lease holder. Heartbeats never
// extend the experiment budget; they only keep the claim visible.
func (storage *Storage) HeartbeatWCNCPLease(ctx context.Context, leaseID, holder string, now time.Time) error {
	if ctx == nil || !validSHA256(leaseID) || len(holder) == 0 || now.IsZero() {
		return ErrWCNCPInvalid
	}
	now = now.UTC()
	finish, err := storage.beginOperation()
	if err != nil {
		return err
	}
	defer finish()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	result, err := storage.state.database.ExecContext(ctx, `UPDATE wcncp_leases SET
    heartbeat_unix_ms = ?, expiry_unix_ms = max(expiry_unix_ms, ?)
WHERE lease_id = ? AND holder = ? AND state = 'ACTIVE' AND expiry_unix_ms > ?`,
		now.UnixMilli(), now.UnixMilli(), leaseID, holder, now.UnixMilli())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrWCNCPLeaseLost
	}
	return nil
}

// ReleaseWCNCPLease marks one lease released or consumed. Only the holder may
// release; expiry is handled by ExpireWCNCPLeases.
func (storage *Storage) ReleaseWCNCPLease(ctx context.Context, leaseID, holder, state string) error {
	if ctx == nil || !validSHA256(leaseID) || len(holder) == 0 || (state != "RELEASED" && state != "CONSUMED") {
		return ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return err
	}
	defer finish()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	result, err := storage.state.database.ExecContext(ctx, `UPDATE wcncp_leases SET state = ?
WHERE lease_id = ? AND holder = ? AND state = 'ACTIVE'`, state, leaseID, holder)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrWCNCPLeaseLost
	}
	return nil
}

// ExpireWCNCPLeases marks every past-expiry ACTIVE lease EXPIRED so the
// attempt is visible and requeueable. Lease failure never delays or fails
// ordinary wrapper builds; it only frees the proposal for reclaim.
func (storage *Storage) ExpireWCNCPLeases(ctx context.Context, now time.Time) (int, error) {
	if ctx == nil || now.IsZero() {
		return 0, ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return 0, err
	}
	defer finish()
	storage.stateMutationMutex.Lock()
	defer storage.stateMutationMutex.Unlock()
	return storage.expireWCNCPLeasesTx(ctx, now.UTC())
}

func (storage *Storage) expireWCNCPLeasesTx(ctx context.Context, now time.Time) (int, error) {
	result, err := storage.state.database.ExecContext(ctx, `UPDATE wcncp_leases SET state = 'EXPIRED'
WHERE state = 'ACTIVE' AND expiry_unix_ms <= ?`, now.UnixMilli())
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}

// RequireWCNCPLease verifies the still-current lease and exact proposal digest
// before a validation publication. Stale holders, late results, and
// conflicting digests fail closed.
func (storage *Storage) RequireWCNCPLease(ctx context.Context, leaseID, holder, proposalDigest string, now time.Time) error {
	if ctx == nil || !validSHA256(leaseID) || len(holder) == 0 || !validSHA256(proposalDigest) || now.IsZero() {
		return ErrWCNCPInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	var storedHolder, storedProposal string
	var expiry int64
	err = storage.state.database.QueryRowContext(ctx, `SELECT holder, proposal_digest, expiry_unix_ms FROM wcncp_leases
WHERE lease_id = ? AND state = 'ACTIVE'`, leaseID).Scan(&storedHolder, &storedProposal, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWCNCPLeaseLost
	}
	if err != nil {
		return err
	}
	if storedHolder != holder || storedProposal != proposalDigest || expiry <= now.UTC().UnixMilli() {
		return ErrWCNCPLeaseLost
	}
	return nil
}

func containsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return containsSubstring(message, "UNIQUE constraint failed") || containsSubstring(message, "PRIMARY KEY")
}

func containsSubstring(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
