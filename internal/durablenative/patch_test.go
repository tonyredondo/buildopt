package durablenative

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestPatchTransactionIsExactAndIdempotent(t *testing.T) {
	original := []byte("package sample;\n\npublic abstract class Work extends DefaultTask {\n  @Input abstract String getInput();\n  @OutputFile abstract File getOutput();\n  @TaskAction void run() {}\n}\n")
	candidate, ok := ScanSource("Work.java", original)
	if !ok {
		t.Fatal("fixture was not detected")
	}
	patch, err := CompilePatch(original, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePatch(patch); err != nil {
		t.Fatal(err)
	}
	patched, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatal(err)
	}
	patchedAgain, err := ApplyPatch(patched, patch)
	if err != nil || !bytes.Equal(patchedAgain, patched) {
		t.Fatalf("apply is not idempotent: %v", err)
	}
	reverted, err := RevertPatch(patched, patch)
	if err != nil || !bytes.Equal(reverted, original) {
		t.Fatalf("revert is not exact: %v", err)
	}
	revertedAgain, err := RevertPatch(reverted, patch)
	if err != nil || !bytes.Equal(revertedAgain, original) {
		t.Fatalf("revert is not idempotent: %v", err)
	}
}

func TestPatchRejectsDriftAndAmbiguity(t *testing.T) {
	original := []byte("class Work extends DefaultTask { @Input String in; @OutputFile File out; @TaskAction void run() {} }\n")
	candidate, ok := ScanSource("Work.groovy", original)
	if !ok {
		t.Fatal("fixture was not detected")
	}
	if _, err := CompilePatch(append(original, ' '), candidate); err == nil {
		t.Fatal("source drift passed")
	}

	ambiguous := append(append([]byte(nil), original...), original...)
	sum := sha256.Sum256(ambiguous)
	candidate.SourceSHA256 = hex.EncodeToString(sum[:])
	if _, err := CompilePatch(ambiguous, candidate); err == nil {
		t.Fatal("ambiguous class passed")
	}
}

func TestPatchSupportsKotlinDeclaration(t *testing.T) {
	original := []byte("abstract class Work : DefaultTask() {\n @get:Input abstract val input: Property<String>\n @get:OutputDirectory abstract val output: DirectoryProperty\n @TaskAction fun run() {}\n}\n")
	candidate, ok := ScanSource("Work.kt", original)
	if !ok {
		t.Fatal("fixture was not detected")
	}
	patch, err := CompilePatch(original, candidate)
	if err != nil {
		t.Fatal(err)
	}
	patched, err := ApplyPatch(original, patch)
	if err != nil || !bytes.Contains(patched, []byte(cacheableAnnotation)) {
		t.Fatalf("Kotlin patch failed: %v", err)
	}
}
