package adaptivefragment

import "testing"

func TestShadowReplayReproducesWholeProfileAndRetainsPartialCompatibility(t *testing.T) {
	replay, err := ReplayShadowProfile([]ShadowObservation{
		{Sequence: 1, Revision: "first", OriginalReason: "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", WrapperMatches: true, ExactRequiredOutputs: true},
		{Sequence: 2, Revision: "second", OriginalSelected: true, OriginalReason: "QUALIFIED_PROFILE_SELECTED", WrapperMatches: true, ExactRequiredOutputs: true},
		{Sequence: 3, Revision: "third", OriginalReason: "ECONOMIC_PREQUALIFICATION_REJECTED", WrapperMatches: true, ExactRequiredOutputs: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 3 || replay[0].ReproducedSelected || !replay[1].ReproducedSelected || replay[2].EconomicAuthorization {
		t.Fatalf("unexpected shadow replay: %+v", replay)
	}
	for index, decision := range replay {
		if decision.MaxSourceSequence != decision.Sequence {
			t.Fatalf("decision %d used future evidence: %+v", index, decision)
		}
		if decision.Fragments[1].Kind != KindSubgraph || decision.Fragments[1].Applicability != ShadowApplicable {
			t.Fatalf("decision %d lost structural compatibility: %+v", index, decision)
		}
	}
	if replay[0].Fragments[0].Applicability != ShadowSuspended || replay[2].Fragments[0].Applicability != ShadowNotEvaluated {
		t.Fatalf("materialization compatibility was conflated with structural compatibility: %+v", replay)
	}
}

func TestShadowReplayRejectsLookaheadAmbiguityAndUnsupportedHistory(t *testing.T) {
	cases := map[string][]ShadowObservation{
		"out of order": {
			{Sequence: 2, Revision: "second", OriginalReason: "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", WrapperMatches: true, ExactRequiredOutputs: true},
			{Sequence: 1, Revision: "first", OriginalReason: "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", WrapperMatches: true, ExactRequiredOutputs: true},
		},
		"selection mismatch": {
			{Sequence: 1, Revision: "first", OriginalSelected: true, OriginalReason: "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", WrapperMatches: true, ExactRequiredOutputs: true},
		},
		"unknown reason": {
			{Sequence: 1, Revision: "first", OriginalReason: "UNKNOWN", WrapperMatches: true, ExactRequiredOutputs: true},
		},
		"wrapper drift": {
			{Sequence: 1, Revision: "first", OriginalReason: "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", ExactRequiredOutputs: true},
		},
		"output mismatch": {
			{Sequence: 1, Revision: "first", OriginalReason: "QUALIFIED_PROFILE_OUTPUTS_REFRESHED", WrapperMatches: true},
		},
	}
	for name, observations := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ReplayShadowProfile(observations); err == nil {
				t.Fatal("invalid shadow history was accepted")
			}
		})
	}
}
