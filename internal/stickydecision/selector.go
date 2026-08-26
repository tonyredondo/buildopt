package stickydecision

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SelectionStatus describes what the caller may execute after a local lookup.
// NATIVE is deliberately the safe value for every lookup failure; ACTIVE is a
// recognition result only in SWL-008 and does not execute an action yet.
type SelectionStatus string

const (
	SelectionNative SelectionStatus = "NATIVE"
	SelectionActive SelectionStatus = "ACTIVE"
)

const (
	SelectionReasonVerifiedNoop       = "VERIFIED_NATIVE_NOOP"
	SelectionReasonVerifiedActive     = "VERIFIED_ACTIVE_DEFERRED"
	SelectionReasonNoLocalSnapshot    = "NO_LOCAL_SNAPSHOT"
	SelectionReasonExpired            = "EXPIRED_SNAPSHOT"
	SelectionReasonRevoked            = "REVOKED_SNAPSHOT"
	SelectionReasonCorrupt            = "CORRUPT_SNAPSHOT"
	SelectionReasonIncompatible       = "INCOMPATIBLE_SNAPSHOT"
	SelectionReasonBusy               = "SNAPSHOT_BUSY"
	SelectionReasonInvalidRequest     = "INVALID_SELECTOR_REQUEST"
	SelectionReasonUnavailable        = "LOCAL_STATE_UNAVAILABLE"
	SelectionReasonDecisionNotAllowed = "DECISION_NOT_EXECUTABLE"
)

// RefreshFunc is intentionally a best-effort callback. It may contact the
// central state service, but selection never calls it synchronously.
type RefreshFunc func(context.Context) error

// SelectorOptions supplies the owner key registry and a deterministic clock
// for tests. A non-nil Scheduler and Refresh opt into one asynchronous refresh
// when local state is missing, stale, busy or incompatible.
type SelectorOptions struct {
	PublicKeys map[string]ed25519.PublicKey
	Now        func() time.Time
	Scheduler  *RefreshScheduler
	Refresh    RefreshFunc
}

// Selection is the fail-closed result of a local decision lookup. Decision is
// always the decision observed in the snapshot or NATIVE_NOOP when no action
// may execute. StoredDecision preserves the rejected/recognized value for
// explain output without granting it authority.
type Selection struct {
	Status          SelectionStatus `json:"status"`
	Decision        string          `json:"decision"`
	StoredDecision  string          `json:"storedDecision,omitempty"`
	Reason          string          `json:"reason"`
	Generation      uint64          `json:"generation,omitempty"`
	RecordDigest    string          `json:"recordDigest,omitempty"`
	ReadDurationNs  int64           `json:"readDurationNs"`
	RefreshScheduled bool           `json:"refreshScheduled,omitempty"`
}

// RefreshScheduler coalesces asynchronous refreshes so repeated native
// invocations cannot create an unbounded number of network requests.
type RefreshScheduler struct {
	mu      sync.Mutex
	working bool
}

// Schedule starts one refresh without blocking the caller. A second request
// while the first is running is coalesced and returns false.
func (scheduler *RefreshScheduler) Schedule(ctx context.Context, refresh RefreshFunc) bool {
	if scheduler == nil || refresh == nil || ctx == nil {
		return false
	}
	scheduler.mu.Lock()
	if scheduler.working {
		scheduler.mu.Unlock()
		return false
	}
	scheduler.working = true
	scheduler.mu.Unlock()
	go func() {
		defer func() {
			scheduler.mu.Lock()
			scheduler.working = false
			scheduler.mu.Unlock()
		}()
		_ = refresh(ctx)
	}()
	return true
}

// LocalDecisionRoot derives the portable user-cache location for one
// repository scope. The scope is the same digest used by LocalStore, so two
// checkouts of one repository share snapshots while unrelated repositories do
// not. It performs no filesystem access.
func LocalDecisionRoot(cacheRoot, scope string) (string, error) {
	if cacheRoot == "" || !filepath.IsAbs(cacheRoot) || filepath.Clean(cacheRoot) != cacheRoot {
		return "", fmt.Errorf("%w: decision cache root must be one clean absolute path", ErrInvalidDocument)
	}
	if !digestPattern.MatchString(scope) {
		return "", ErrCrossScope
	}
	return filepath.Join(cacheRoot, "buildopt", "sticky", "state", scope), nil
}

// SelectLocal reads one verified local snapshot and returns the action that is
// safe for the current invocation. No network or writer is on this path. A
// current NATIVE_NOOP is accepted, while a compatible ACTIVE decision is only
// reported for the later execution block; SWL-008 still executes native
// Gradle. All other states retain native execution.
func SelectLocal(ctx context.Context, root, scope string, expected Binding, options SelectorOptions) Selection {
	started := time.Now()
	selection := Selection{Status: SelectionNative, Decision: ExecutionNativeNoop}
	finish := func(reason string) Selection {
		selection.Reason = reason
		selection.ReadDurationNs = time.Since(started).Nanoseconds()
		if options.Scheduler != nil && options.Refresh != nil && shouldRefresh(reason) {
			selection.RefreshScheduled = options.Scheduler.Schedule(ctx, options.Refresh)
		}
		return selection
	}
	if ctx == nil || ctx.Err() != nil || root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root || !digestPattern.MatchString(scope) || expected.RepositoryScopeSHA256 != scope || len(options.PublicKeys) == 0 {
		return finish(SelectionReasonInvalidRequest)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	store, err := OpenLocalReadOnly(root, scope, StoreOptions{PublicKeys: options.PublicKeys, Now: options.Now})
	if err != nil {
		return finish(classifySelectionError(err))
	}
	snapshot, err := store.CurrentReadOnly(ctx)
	if err != nil {
		return finish(classifySelectionError(err))
	}
	if snapshot.Document.Decision == nil || snapshot.Document.RecordType != DecisionRecordType {
		return finish(SelectionReasonIncompatible)
	}
	decision := snapshot.Document.Decision
	selection.StoredDecision = decision.ExecutionDecision
	selection.Generation = snapshot.Head.Generation
	selection.RecordDigest = snapshot.RecordDigest
	if decision.Binding != expected {
		return finish(SelectionReasonIncompatible)
	}
	switch decision.ExecutionDecision {
	case ExecutionNativeNoop:
		selection.Reason = SelectionReasonVerifiedNoop
		selection.ReadDurationNs = time.Since(started).Nanoseconds()
		return selection
	case ExecutionActiveRuntime, ExecutionActivePatch:
		selection.Status = SelectionActive
		selection.Decision = decision.ExecutionDecision
		selection.Reason = SelectionReasonVerifiedActive
		selection.ReadDurationNs = time.Since(started).Nanoseconds()
		return selection
	default:
		return finish(SelectionReasonDecisionNotAllowed)
	}
}

func shouldRefresh(reason string) bool {
	switch reason {
	case SelectionReasonNoLocalSnapshot, SelectionReasonExpired, SelectionReasonBusy,
		SelectionReasonCorrupt, SelectionReasonIncompatible, SelectionReasonUnavailable:
		return true
	default:
		return false
	}
}

func classifySelectionError(err error) string {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, os.ErrNotExist):
		return SelectionReasonNoLocalSnapshot
	case errors.Is(err, ErrExpired):
		return SelectionReasonExpired
	case errors.Is(err, ErrRevoked):
		return SelectionReasonRevoked
	case errors.Is(err, ErrBusy):
		return SelectionReasonBusy
	case errors.Is(err, ErrCorrupt):
		return SelectionReasonCorrupt
	case errors.Is(err, ErrCrossScope), errors.Is(err, ErrInvalidDocument), errors.Is(err, ErrCrossPlane):
		return SelectionReasonIncompatible
	default:
		return SelectionReasonUnavailable
	}
}
