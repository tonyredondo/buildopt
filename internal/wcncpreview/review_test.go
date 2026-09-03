package wcncpreview

import (
	"strings"
	"testing"
)

func fixtureFields() map[string]string {
	return map[string]string{
		"problem": "Configuration Cache blocker in owner build logic wastes critical-path time.",
		"expectedImprovement": "Small reversible correction removes the blocker without changing semantics.",
		"diff": "--- a/buildSrc/src/main/kotlin/Example.kt\n+++ b/buildSrc/src/main/kotlin/Example.kt\n@@ -10,7 +10,7 @@\n-unsafe\n+safe\n",
		"declarationLocation": "buildSrc/src/main/kotlin/Example.kt:10-20",
		"safetyRationale": "Behavior-preserving API swap with identical observable outputs.",
		"correctnessEvidence": "5/5 fixture starts with byte-exact outputs and exact revert.",
		"limitations": "Applies only to the bound workflow and revision.",
		"applyCommand": "buildoptw --apply wcncp:proposal:<digest>",
		"revertCommand": "buildoptw --revert wcncp:proposal:<digest>",
	}
}

func TestDraftBindsDigestsAndRendersRequiredSections(t *testing.T) {
	t.Parallel()
	proposal := strings.Repeat("c", 64)
	validation := strings.Repeat("d", 64)
	patch := []byte("patch-bytes")
	inverse := []byte("inverse-bytes")
	artifact, digest, err := Draft(proposal, validation, patch, inverse, SystemFixtureLabel, fixtureFields())
	if err != nil || len(digest) != 64 {
		t.Fatalf("draft = %v/%q", digest, err)
	}
	if artifact.ProposalSHA256 != proposal || artifact.ValidationSHA256 != validation || artifact.Lane != SystemFixtureLabel {
		t.Fatalf("bindings = %+v", artifact)
	}
	if artifact.AuthorityStatement != NoAutoMutationStatement {
		t.Fatal("authority statement missing")
	}
	for _, section := range []string{artifact.Problem, artifact.Diff, artifact.SafetyRationale, artifact.ApplyCommand, artifact.RevertCommand} {
		if section == "" {
			t.Fatal("required section empty")
		}
	}
	// Replay is deterministic: same inputs yield the same digest.
	_, replayed, err := Draft(proposal, validation, patch, inverse, SystemFixtureLabel, fixtureFields())
	if err != nil || replayed != digest {
		t.Fatal("replay digest mismatch")
	}
}

func TestTamperAndStaleFailClosed(t *testing.T) {
	t.Parallel()
	proposal := strings.Repeat("c", 64)
	validation := strings.Repeat("d", 64)
	if _, _, err := Draft(proposal, validation, []byte("patch"), []byte("inverse"), "PROSPECTIVE", map[string]string{}); err == nil {
		t.Fatal("empty safety fields accepted")
	}
	artifact, _, err := Draft(proposal, validation, []byte("patch"), []byte("inverse"), SystemFixtureLabel, fixtureFields())
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPatchBinding(artifact.PatchSHA256, []byte("patch")); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPatchBinding(artifact.PatchSHA256, []byte("tampered")); err == nil {
		t.Fatal("tampered patch accepted")
	}
	if got := Lifecycle(nil, true); got != "STALE" {
		t.Fatalf("drift = %s", got)
	}
}

func TestAcceptRejectDeferLifecycle(t *testing.T) {
	t.Parallel()
	if got := Lifecycle(nil, false); got != "REVIEW_READY" {
		t.Fatalf("no decision = %s", got)
	}
	if got := Lifecycle([]string{"ACCEPT"}, false); got != "OWNER_ACCEPTED" {
		t.Fatalf("accept = %s", got)
	}
	if got := Lifecycle([]string{"REJECT"}, false); got != "OWNER_REJECTED" {
		t.Fatalf("reject = %s", got)
	}
	if got := Lifecycle([]string{"DEFER"}, false); got != "OWNER_DEFERRED" {
		t.Fatalf("defer = %s", got)
	}
}
