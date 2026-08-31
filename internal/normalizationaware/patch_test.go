package normalizationaware

import (
	"bytes"
	"testing"
)

func TestPatchV2MarkerAndReviewedRelativeTransactions(t *testing.T) {
	for _, tc := range []struct {
		name, normalization string
		proof               *SemanticProof
		action              string
	}{
		{"marker", "@PathSensitive(PathSensitivity.RELATIVE)", nil, "ADD_CACHEABLE_TASK_MARKER_V2"},
		{"relative", "", &SemanticProof{true, true, true, true, "OWNER-REVIEWED-FIXTURE"}, "ADD_RELATIVE_PATH_NORMALIZATION_AND_CACHEABLE_MARKER_V1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("abstract class Work extends DefaultTask {\n" + tc.normalization + "\n@InputDirectory\nabstract File getInput()\n@OutputDirectory abstract File getOutput()\n@TaskAction void run() {}\n}\n")
			candidate := ScanSourceV2("Work.groovy", source)[0]
			patch, err := CompilePatchV2(source, candidate, tc.proof)
			if err != nil {
				t.Fatal(err)
			}
			if patch.Action != tc.action {
				t.Fatalf("action=%s", patch.Action)
			}
			patched, err := ApplyPatchV2(source, patch)
			if err != nil {
				t.Fatal(err)
			}
			again, err := ApplyPatchV2(patched, patch)
			if err != nil || !bytes.Equal(patched, again) {
				t.Fatal("apply not idempotent")
			}
			reverted, err := RevertPatchV2(patched, patch)
			if err != nil || !bytes.Equal(source, reverted) {
				t.Fatal("revert not exact")
			}
		})
	}
}

func TestPatchV2RejectsMissingProofAmbiguityAndDrift(t *testing.T) {
	source := []byte("class Work extends DefaultTask {\n@InputDirectory File getInput()\n@OutputDirectory File getOutput()\n@TaskAction void run() {}\n}\n")
	candidate := ScanSourceV2("Work.java", source)[0]
	if _, err := CompilePatchV2(source, candidate, nil); err == nil {
		t.Fatal("missing proof accepted")
	}
	proof := &SemanticProof{true, true, true, true, "reviewed"}
	if _, err := CompilePatchV2(append(source, ' '), candidate, proof); err == nil {
		t.Fatal("source drift accepted")
	}
	ambiguous := append(append([]byte(nil), source...), source...)
	candidate.SourceSHA256 = sourceDigest(ambiguous)
	if _, err := CompilePatchV2(ambiguous, candidate, proof); err == nil {
		t.Fatal("ambiguity accepted")
	}
}
