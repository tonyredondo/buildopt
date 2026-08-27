package durablecatalog

import "testing"

func TestPublicScreenQualifiesRecurringGraphAction(t *testing.T) {
	input := validPublicScreenInput()
	report, err := ScreenPublicOpportunities(input)
	if err != nil {
		t.Fatal(err)
	}
	if !report.GatePassed || report.PassingFamilies != 5 || report.Outcome != "READY_FOR_SWL_014D" {
		t.Fatalf("unexpected gate: %+v", report)
	}
	for _, family := range report.Families {
		if len(family.TestableActions) != 1 || family.TestableActions[0].CompatibleBuildsToRepay != 3 ||
			family.TestableActions[0].ProjectedSavingNs != 10 || !family.TestableActions[0].OwnerReviewRequired {
			t.Fatalf("unexpected family action: %+v", family)
		}
	}
}

func TestPublicScreenPreservesUnavailableEvidence(t *testing.T) {
	input := validPublicScreenInput()
	for index := range input.Families {
		input.Families[index].Detectors[1].UnavailableReason = "EXACT_CANDIDATE_PLAN_AND_CRITICAL_PATH_INPUT_UNAVAILABLE"
		input.Families[index].Detectors[1].GraphObservations = nil
	}
	report, err := ScreenPublicOpportunities(input)
	if err != nil {
		t.Fatal(err)
	}
	if report.GatePassed || report.PassingFamilies != 0 || report.Outcome != "SWL_014C_WITH_STOP_EVIDENCE" {
		t.Fatalf("unexpected stop evidence: %+v", report)
	}
	for _, family := range report.Families {
		if family.DetectorResults[1].Status != DetectorStatusInputUnavailable || len(family.TestableActions) != 0 {
			t.Fatalf("unavailable detector was rewritten: %+v", family)
		}
	}
}

func TestPublicScreenRejectsMissingBinding(t *testing.T) {
	input := validPublicScreenInput()
	input.Families[0].Detectors[1].GraphObservations[0].CandidatePlanSHA256 = ""
	if _, err := ScreenPublicOpportunities(input); err == nil {
		t.Fatal("missing candidate plan binding was accepted")
	}
}

func TestPublicScreenRejectsDuplicateRecurrence(t *testing.T) {
	input := validPublicScreenInput()
	input.Families[0].Detectors[1].GraphObservations[1].ObservationID =
		input.Families[0].Detectors[1].GraphObservations[0].ObservationID
	if _, err := ScreenPublicOpportunities(input); err == nil {
		t.Fatal("duplicate graph recurrence was accepted")
	}
}

func validPublicScreenInput() PublicScreenInput {
	digest := func(character byte) string {
		value := make([]byte, 64)
		for index := range value {
			value[index] = character
		}
		return string(value)
	}
	families := make([]PublicFamilyInput, 0, 5)
	for index, key := range []string{"one", "two", "three", "four", "five"} {
		observations := make([]PublicGraphObservation, 0, 3)
		for ordinal := uint64(1); ordinal <= 3; ordinal++ {
			observations = append(observations, PublicGraphObservation{
				Ordinal: ordinal, ObservationID: key + "-observation-" + string(rune('0'+ordinal)), CandidatePlanSHA256: digest('c'),
				BindingDigest: digest('d'), FullProjectCount: 10, CandidateProjectCount: uint64(index + 1),
				FullOutputSHA256: digest('e'), CandidateOutputSHA256: digest('e'), ProjectedSavingNs: 10,
			})
		}
		families = append(families, PublicFamilyInput{
			FamilyKey: key, RevisionWindowSHA256: digest('a'), RepositoryScopeSHA256: digest('b'),
			WorkflowArgumentsSHA256: digest('f'), OutputContract: []string{"build/output"},
			Detectors: []PublicDetectorInput{
				{DetectorID: DetectorTaskContract, InputEvidenceSHA256: digest('1'), UnavailableReason: "INPUT_UNAVAILABLE_NO_GENERIC_SOURCE_PRODUCER"},
				{DetectorID: DetectorGraphBreadth, InputEvidenceSHA256: digest('2'), GraphObservations: observations, TrialCostNs: 11, ValidationCostNs: 9, PublicationCostNs: 1},
			},
		})
	}
	return PublicScreenInput{GeneratedAt: "2026-08-27T00:00:00Z", CohortManifestSHA256: digest('0'),
		DetectorOrder: []string{DetectorTaskContract, DetectorGraphBreadth}, MinimumPassing: 3,
		MaximumBuildsToRepay: 30, Families: families}
}
