package nativevolatility

import (
	"strings"
	"testing"
)

const testBinding = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestAnalyzeQuarantinesWholeProducerAndVerifiesCandidate(t *testing.T) {
	first := observation(
		entry("stable/a.bin", "1", ":stable:produce"),
		entry("stable/b.bin", "2", ":stable:produce"),
		entry("volatile/changed.bin", "3", ":volatile:produce"),
		entry("volatile/unchanged.bin", "4", ":volatile:produce"),
	)
	second := observation(
		entry("stable/a.bin", "1", ":stable:produce"),
		entry("stable/b.bin", "2", ":stable:produce"),
		entry("volatile/changed.bin", "5", ":volatile:produce"),
		entry("volatile/unchanged.bin", "4", ":volatile:produce"),
	)

	result := Analyze(first, second)
	if result.Decision != DecisionTransportReady || result.Reason != ReasonQuarantined {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.VolatilePaths) != 1 || len(result.QuarantinedProducers) != 1 ||
		len(result.QuarantinedOutputs) != 2 || len(result.TransportedOutputs) != 2 {
		t.Fatalf("producer quarantine was not atomic: %+v", result)
	}
	reused := observation(result.TransportedOutputs...)
	rebuilt := observation(second.Entries[2:]...)
	if err := VerifyCandidate(result, reused, rebuilt); err != nil {
		t.Fatalf("verify candidate: %v", err)
	}

	tampered := observation(result.TransportedOutputs...)
	tampered.Entries[0].SHA256 = digest("9")
	if err := VerifyCandidate(result, tampered, rebuilt); err == nil {
		t.Fatal("tampered reused output was accepted")
	}
}

func TestAnalyzeRetainsNativeOnUnsafeEvidence(t *testing.T) {
	tests := []struct {
		name   string
		first  Observation
		second Observation
		reason string
	}{
		{
			name: "binding mismatch", first: observation(entry("a", "1", ":a")),
			second: func() Observation {
				value := observation(entry("a", "1", ":a"))
				value.BindingSHA256 = digest("b")
				return value
			}(), reason: ReasonBindingMismatch,
		},
		{
			name: "path mismatch", first: observation(entry("a", "1", ":a")),
			second: observation(entry("b", "1", ":a")), reason: ReasonPathMismatch,
		},
		{
			name: "producer mismatch", first: observation(entry("a", "1", ":a")),
			second: observation(entry("a", "1", ":b")), reason: ReasonProducerAmbiguous,
		},
		{
			name: "producer missing", first: observation(Entry{Path: "a", SHA256: digest("1")}),
			second: observation(Entry{Path: "a", SHA256: digest("1")}), reason: ReasonInvalidEvidence,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Analyze(test.first, test.second)
			if result.Decision != DecisionNativeRetained || result.Reason != test.reason {
				t.Fatalf("unexpected fail-closed result: %+v", result)
			}
		})
	}
}

func TestAnalyzeExactObservationsTransportEverything(t *testing.T) {
	first := observation(entry("a", "1", ":a"), entry("b", "2", ":b"))
	result := Analyze(first, first)
	if result.Decision != DecisionTransportReady || result.Reason != ReasonExact ||
		len(result.QuarantinedOutputs) != 0 || len(result.TransportedOutputs) != 2 {
		t.Fatalf("unexpected exact result: %+v", result)
	}
	if err := VerifyCandidate(result, observation(result.TransportedOutputs...), observation()); err != nil {
		t.Fatalf("verify exact candidate with no rebuilt outputs: %v", err)
	}
}

func TestAnalyzeRejectsNonCanonicalPath(t *testing.T) {
	for _, candidate := range []string{
		"output/../output/a.bin",
		`output\a.bin`,
		"/absolute/output.bin",
	} {
		result := Analyze(
			observation(entry(candidate, "1", ":a")),
			observation(entry(candidate, "1", ":a")),
		)
		if result.Decision != DecisionNativeRetained || result.Reason != ReasonInvalidEvidence {
			t.Fatalf("non-canonical path %q did not retain native: %+v", candidate, result)
		}
	}
}

func observation(entries ...Entry) Observation {
	cloned := make([]Entry, len(entries))
	for index, value := range entries {
		cloned[index] = cloneEntry(value)
	}
	return Observation{SchemaVersion: ObservationSchema, BindingSHA256: testBinding, Entries: cloned}
}

func entry(path, seed, producer string) Entry {
	return Entry{Path: path, SHA256: digest(seed), ProducerTasks: []string{producer}}
}

func digest(seed string) string {
	return strings.Repeat(seed, 64)
}
