package nativevolatility

import (
	"os"
	"path/filepath"
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
	if err := ValidateResult(result); err != nil {
		t.Fatalf("validate result: %v", err)
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
	tamperedResult := result
	tamperedResult.TransportedOutputs = append([]Entry(nil), result.TransportedOutputs...)
	tamperedResult.TransportedOutputs[0].SHA256 = digest("9")
	if err := ValidateResult(tamperedResult); err == nil {
		t.Fatal("tampered transport result was accepted")
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

func TestObserveHashesProducerBoundInventoryAndRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "module", "build", "value.bin")
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("native-output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory := []Entry{{Path: "module/build/value.bin", ProducerTasks: []string{":module:produce"}}}
	observed, err := Observe(root, testBinding, inventory)
	if err != nil {
		t.Fatal(err)
	}
	if len(observed.Entries) != 1 || observed.Entries[0].SHA256 !=
		"eae571c9f5c5bfb917357ce4cc94ecbb0862a14c0cf413b26b3f47473e72e257" ||
		!equalStrings(observed.Entries[0].ProducerTasks, []string{":module:produce"}) {
		t.Fatalf("observed inventory = %+v", observed)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside.bin"), output); err != nil {
		t.Fatal(err)
	}
	if _, err := Observe(root, testBinding, inventory); err == nil {
		t.Fatal("symlinked native output was accepted")
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
