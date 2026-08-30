package requesthit_test

import (
	"testing"

	"github.com/tonyredondo/buildopt/internal/requesthit"
)

func TestShadowReplayMatchesNativeWithoutAuthority(t *testing.T) {
	record := loadRecord(t)
	quarantine := requesthit.NewQuarantine()
	verdict := requesthit.Replay(record, requesthit.MatchingProbe(record), matchingNative(record), quarantine, verificationTime())
	if verdict.Disposition != requesthit.ShadowDispositionMatched || !verdict.Predicted || !verdict.Matched || verdict.Quarantined || !verdict.NativeExecuted {
		t.Fatalf("shadow verdict = %+v", verdict)
	}
	if verdict.SelectionAuthorized || verdict.ActivationAuthorized || verdict.PerformanceMeasured {
		t.Fatalf("shadow replay invented authority: %+v", verdict)
	}
	if _, exists := quarantine.Reason(verdict.RecordSHA256); exists {
		t.Fatal("matching identity was quarantined")
	}
}

func TestShadowMismatchQuarantinesFirstIdentityUntilNewEvidence(t *testing.T) {
	record := loadRecord(t)
	probe := requesthit.MatchingProbe(record)
	quarantine := requesthit.NewQuarantine()
	native := matchingNative(record)
	native.Outputs[0].WorkspaceSHA256 = "9b8f65607d891eb44ea2119ebd24664622d0a6f620902a08f7b0fb7320aa42e8"

	first := requesthit.Replay(record, probe, native, quarantine, verificationTime())
	if first.Disposition != requesthit.ShadowDispositionMismatch || first.Reason != requesthit.ShadowReasonNativeOutputMismatch || !first.Quarantined || !first.NativeExecuted {
		t.Fatalf("first mismatch = %+v", first)
	}
	second := requesthit.Replay(record, probe, matchingNative(record), quarantine, verificationTime())
	if second.Disposition != requesthit.ShadowDispositionNativeRetained || second.Predicted || !second.Quarantined || !second.NativeExecuted {
		t.Fatalf("quarantined replay = %+v", second)
	}

	record.ExpiresAt = "2026-09-02T00:00:00Z"
	third := requesthit.Replay(record, requesthit.MatchingProbe(record), matchingNative(record), quarantine, verificationTime())
	if third.Disposition != requesthit.ShadowDispositionMatched || third.RecordSHA256 == first.RecordSHA256 {
		t.Fatalf("new evidence did not obtain a distinct shadow identity: %+v", third)
	}
}

func TestShadowRetainsNativeWhenSafetyIsIncomplete(t *testing.T) {
	record := loadRecord(t)
	probe := requesthit.MatchingProbe(record)
	probe.EvidenceComplete = false
	verdict := requesthit.Replay(record, probe, matchingNative(record), requesthit.NewQuarantine(), verificationTime())
	if verdict.Disposition != requesthit.ShadowDispositionNativeRetained || verdict.Reason != string(requesthit.ReasonCurrentEvidenceIncomplete) || verdict.Predicted || !verdict.NativeExecuted {
		t.Fatalf("incomplete safety verdict = %+v", verdict)
	}
}

func TestShadowRequiresExactCommandAndObservedGradle(t *testing.T) {
	for name, mutate := range map[string]func(*requesthit.NativeResult){
		"command": func(result *requesthit.NativeResult) { result.ExactCommandPreserved = false },
		"gradle":  func(result *requesthit.NativeResult) { result.GradleProcessObserved = false },
		"outcome": func(result *requesthit.NativeResult) { result.ExitCode = 37 },
	} {
		t.Run(name, func(t *testing.T) {
			record := loadRecord(t)
			result := matchingNative(record)
			mutate(&result)
			verdict := requesthit.Replay(record, requesthit.MatchingProbe(record), result, requesthit.NewQuarantine(), verificationTime())
			if verdict.Disposition != requesthit.ShadowDispositionMismatch || !verdict.Quarantined {
				t.Fatalf("verdict = %+v", verdict)
			}
		})
	}
}

func matchingNative(record requesthit.SafetyRecord) requesthit.NativeResult {
	result := requesthit.NativeResult{
		Outcome: record.PriorResult.Outcome, ExitCode: record.PriorResult.ExitCode,
		ExactCommandPreserved: true, GradleProcessObserved: true,
	}
	for _, output := range record.Outputs.States {
		observed := requesthit.ObservedOutput{Path: output.Path}
		if output.Exists {
			observed.WorkspaceExists = true
			observed.WorkspaceSHA256 = output.SHA256
		}
		result.Outputs = append(result.Outputs, observed)
	}
	return result
}
