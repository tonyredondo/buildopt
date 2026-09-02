package main

import (
	"testing"

	"github.com/tonyredondo/buildopt/internal/normalizationaware"
)

func TestDecodeManifestRejectsDuplicateFamilies(t *testing.T) {
	raw := []byte(`{
  "schemaVersion":"buildopt.specs/poc-reviewed-native-patch-customer-economics-subjects/v1",
  "reusePolicy":"PUBLIC_REVISIONS_ONLY_NO_PRIOR_RESULT_ROWS",
  "families":[
    {"key":"same","repository":"https://example.invalid/a.git","revision":"1111111111111111111111111111111111111111"},
    {"key":"same","repository":"https://example.invalid/b.git","revision":"2222222222222222222222222222222222222222"},
    {"key":"c","repository":"https://example.invalid/c.git","revision":"3333333333333333333333333333333333333333"},
    {"key":"d","repository":"https://example.invalid/d.git","revision":"4444444444444444444444444444444444444444"},
    {"key":"e","repository":"https://example.invalid/e.git","revision":"5555555555555555555555555555555555555555"}
  ]
}`)
	if _, err := decodeManifest(raw); err == nil {
		t.Fatal("expected duplicate family rejection")
	}
}

func TestFamilyHasActionUsesOnlyTypedDecisions(t *testing.T) {
	if familyHasAction([]normalizationaware.Candidate{{Decision: normalizationaware.ExplicitNonPortable}}) {
		t.Fatal("explicit non-portable input must not be actionable")
	}
	if !familyHasAction([]normalizationaware.Candidate{{Decision: normalizationaware.MarkerOnlyEligible}}) {
		t.Fatal("marker-only candidate should be actionable")
	}
	if !familyHasAction([]normalizationaware.Candidate{{Decision: normalizationaware.ReviewedRelativeProofNeeded}}) {
		t.Fatal("reviewed-relative candidate should be actionable")
	}
}
