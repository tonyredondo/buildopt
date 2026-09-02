package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/normalizationaware"
)

func TestDecodeManifestRequiresTenUniqueFamilies(t *testing.T) {
	families := make([]string, 10)
	for index := range families {
		families[index] = fmt.Sprintf(`{"key":"f%d","repository":"https://example.invalid/%d.git","revision":"%040d"}`, index, index, index+1)
	}
	raw := []byte(fmt.Sprintf(`{"schemaVersion":%q,"reusePolicy":%q,"families":[%s]}`,
		subjectSchema, reusePolicy, strings.Join(families, ",")))
	if _, err := decodeManifest(raw); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	families[9] = families[0]
	raw = []byte(fmt.Sprintf(`{"schemaVersion":%q,"reusePolicy":%q,"families":[%s]}`,
		subjectSchema, reusePolicy, strings.Join(families, ",")))
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
