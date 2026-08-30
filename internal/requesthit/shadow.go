package requesthit

import (
	"sync"
	"time"
)

const (
	ShadowDispositionMatched          = "SHADOW_MATCHED_NATIVE"
	ShadowDispositionMismatch         = "SHADOW_MISMATCH_QUARANTINED"
	ShadowDispositionNativeRetained   = "SHADOW_NOT_PREDICTED"
	ShadowReasonIdentityQuarantined   = "IDENTITY_QUARANTINED"
	ShadowReasonNativeCommandChanged  = "NATIVE_COMMAND_CHANGED"
	ShadowReasonGradleNotObserved     = "GRADLE_NOT_OBSERVED"
	ShadowReasonNativeOutcomeMismatch = "NATIVE_OUTCOME_MISMATCH"
	ShadowReasonNativeOutputMismatch  = "NATIVE_OUTPUT_MISMATCH"
)

// NativeResult is the post-execution evidence from the exact native Gradle
// command. VRH-003 observes this result only after Gradle has completed.
type NativeResult struct {
	Outcome               string           `json:"outcome"`
	ExitCode              int              `json:"exitCode"`
	Outputs               []ObservedOutput `json:"outputs"`
	ExactCommandPreserved bool             `json:"exactCommandPreserved"`
	GradleProcessObserved bool             `json:"gradleProcessObserved"`
}

// ShadowVerdict compares a predicted request hit with the native result. It
// never authorizes selection, activation, output restoration, or early return.
type ShadowVerdict struct {
	Disposition          string `json:"disposition"`
	Reason               string `json:"reason"`
	RecordSHA256         string `json:"recordSha256"`
	Predicted            bool   `json:"predicted"`
	Matched              bool   `json:"matched"`
	Quarantined          bool   `json:"quarantined"`
	NativeExecuted       bool   `json:"nativeExecuted"`
	SelectionAuthorized  bool   `json:"selectionAuthorized"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	PerformanceMeasured  bool   `json:"performanceMeasured"`
}

// Quarantine is an in-memory POC registry keyed by canonical record identity.
// A new record digest is new evidence and is therefore a distinct identity.
type Quarantine struct {
	mu      sync.RWMutex
	reasons map[string]string
}

// NewQuarantine constructs an empty shadow-only identity quarantine.
func NewQuarantine() *Quarantine {
	return &Quarantine{reasons: make(map[string]string)}
}

// Reason returns the first mismatch reason retained for an identity.
func (q *Quarantine) Reason(identity string) (string, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	reason, ok := q.reasons[identity]
	return reason, ok
}

func (q *Quarantine) retain(identity, reason string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, exists := q.reasons[identity]; !exists {
		q.reasons[identity] = reason
	}
}

// Replay predicts only when VRH-002 is complete, then compares that prediction
// with an already-observed native execution. Callers must always execute native
// Gradle before calling Replay, including when the identity is quarantined.
func Replay(record SafetyRecord, probe Probe, native NativeResult, quarantine *Quarantine, now time.Time) ShadowVerdict {
	if quarantine == nil {
		quarantine = NewQuarantine()
	}
	safety := Verify(record, probe, now)
	base := ShadowVerdict{RecordSHA256: safety.RecordSHA256, NativeExecuted: native.GradleProcessObserved}
	if reason, quarantined := quarantine.Reason(safety.RecordSHA256); quarantined {
		base.Disposition = ShadowDispositionNativeRetained
		base.Reason = ShadowReasonIdentityQuarantined + ":" + reason
		base.Quarantined = true
		return base
	}
	if safety.Disposition != DispositionContractComplete {
		base.Disposition = ShadowDispositionNativeRetained
		base.Reason = string(safety.Reason)
		return base
	}
	base.Predicted = true
	reason := compareNative(record, native)
	if reason == "" {
		base.Disposition = ShadowDispositionMatched
		base.Matched = true
		return base
	}
	quarantine.retain(safety.RecordSHA256, reason)
	base.Disposition = ShadowDispositionMismatch
	base.Reason = reason
	base.Quarantined = true
	return base
}

func compareNative(record SafetyRecord, native NativeResult) string {
	if !native.ExactCommandPreserved {
		return ShadowReasonNativeCommandChanged
	}
	if !native.GradleProcessObserved {
		return ShadowReasonGradleNotObserved
	}
	if native.Outcome != record.PriorResult.Outcome || native.ExitCode != record.PriorResult.ExitCode {
		return ShadowReasonNativeOutcomeMismatch
	}
	if compareOutputs(record.Outputs, native.Outputs) != ReasonNone {
		return ShadowReasonNativeOutputMismatch
	}
	return ""
}
