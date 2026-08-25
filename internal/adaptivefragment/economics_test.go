package adaptivefragment

import (
	"reflect"
	"testing"
)

func TestEconomicAssessmentKeepsSignedValueAndDeduplicatesAsyncCost(t *testing.T) {
	assessment, err := AssessEconomics(economicFixture())
	if err != nil {
		t.Fatal(err)
	}
	entry := assessment.Entry
	if entry.RequestedBuildCount != 3 || entry.GrossSavedMs != 600 || entry.SynchronousOverheadMs != 60 ||
		entry.OutOfBandLearningCostMs != 500 || entry.CumulativeNetMs != 40 {
		t.Fatalf("economic entry = %+v", entry)
	}
	if assessment.UniqueCostEventCount != 1 || assessment.NegativeActivatedBuilds != 1 ||
		assessment.ObservedBreakEvenBuild != 2 || assessment.Recurrence != (EconomicRecurrence{CompatibleBuilds: 3, ActivatedBuilds: 3, RequestedBuilds: 3}) {
		t.Fatalf("economic assessment = %+v", assessment)
	}
	if len(assessment.Projections) != 2 || assessment.Projections[0].AdditionalBuilds != 1 ||
		assessment.Projections[0].ProjectedNetMs != 180 || assessment.Projections[1].AdditionalBuilds != 3 ||
		assessment.Projections[1].ObservedPlusProjectedMs != 522 {
		t.Fatalf("economic projections = %+v", assessment.Projections)
	}
	if assessment.Regret.ObservedDownsideMs != 0 || !assessment.Regret.WithinBudget {
		t.Fatalf("economic regret = %+v", assessment.Regret)
	}
}

func TestNegativeBuildReducesValueAndProjectionCannotRewriteObservation(t *testing.T) {
	withNegative := economicFixture()
	withoutNegative := economicFixture()
	withoutNegative.Observations = withoutNegative.Observations[:2]
	first, err := AssessEconomics(withNegative)
	if err != nil {
		t.Fatal(err)
	}
	positiveOnly, err := AssessEconomics(withoutNegative)
	if err != nil {
		t.Fatal(err)
	}
	if first.Entry.CumulativeNetMs != positiveOnly.Entry.CumulativeNetMs-120 {
		t.Fatalf("negative build did not reduce net: %d / %d", first.Entry.CumulativeNetMs, positiveOnly.Entry.CumulativeNetMs)
	}

	differentProjection := withNegative
	differentProjection.Policy.DecayPermille = 500
	differentProjection.Policy.Horizons = []uint64{2, 8}
	second, err := AssessEconomics(differentProjection)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Entry, second.Entry) || !reflect.DeepEqual(first.Recurrence, second.Recurrence) ||
		reflect.DeepEqual(first.Projections, second.Projections) {
		t.Fatalf("projection changed observed evidence or did not change itself: %+v / %+v", first, second)
	}
}

func TestEconomicAssessmentReportsUnclippedRegret(t *testing.T) {
	series := economicFixture()
	series.Observations = []EconomicObservation{
		{ObservationID: sha("negative"), Sequence: 1, ObservedAt: "2026-08-25T10:00:00Z", Compatible: true, Activated: true, GrossSavedMs: -300, SynchronousOverheadMs: 50},
	}
	series.Policy.RegretBudgetMs = 100
	assessment, err := AssessEconomics(series)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Entry.CumulativeNetMs != -350 || assessment.Regret.ObservedDownsideMs != 350 || assessment.Regret.WithinBudget {
		t.Fatalf("regret was clipped or misclassified: %+v", assessment)
	}
}

func TestEconomicAssessmentRejectsInvalidChronologyAndCostIdentity(t *testing.T) {
	cases := map[string]func(*EconomicSeries){
		"out of order":                     func(series *EconomicSeries) { series.Observations[1].Sequence = 1 },
		"activation without compatibility": func(series *EconomicSeries) { series.Observations[0].Compatible = false },
		"unactivated value": func(series *EconomicSeries) {
			series.Observations[0].Activated = false
		},
		"duplicate observation": func(series *EconomicSeries) {
			series.Observations[1].ObservationID = series.Observations[0].ObservationID
		},
		"cost amount drift": func(series *EconomicSeries) {
			series.Observations[1].AsynchronousCostEvents[0].AmountMs = 501
		},
		"unsorted horizons": func(series *EconomicSeries) { series.Policy.Horizons = []uint64{3, 1} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			series := economicFixture()
			mutate(&series)
			if _, err := AssessEconomics(series); err == nil {
				t.Fatal("invalid economic series was accepted")
			}
		})
	}
}

func economicFixture() EconomicSeries {
	learning := EconomicCostEvent{ID: sha("learning-cost"), AmountMs: 500}
	return EconomicSeries{
		Scope: EconomicScopeFragment, FamilyID: sha("family"), RevisionID: sha("revision"), FragmentGeneration: 3,
		EvidenceExpiresAt: "2026-09-25T10:00:00Z",
		Observations: []EconomicObservation{
			{ObservationID: sha("observation-1"), Sequence: 1, ObservedAt: "2026-08-25T10:00:00Z", Compatible: true, Activated: true, GrossSavedMs: 400, SynchronousOverheadMs: 20, AsynchronousCostEvents: []EconomicCostEvent{learning}},
			{ObservationID: sha("observation-2"), Sequence: 2, ObservedAt: "2026-08-25T10:01:00Z", Compatible: true, Activated: true, GrossSavedMs: 300, SynchronousOverheadMs: 20, AsynchronousCostEvents: []EconomicCostEvent{learning}},
			{ObservationID: sha("observation-3"), Sequence: 3, ObservedAt: "2026-08-25T10:02:00Z", Compatible: true, Activated: true, GrossSavedMs: -100, SynchronousOverheadMs: 20},
		},
		Policy: EconomicPolicy{DecayPermille: 900, Horizons: []uint64{1, 3}, RegretBudgetMs: 200},
	}
}
