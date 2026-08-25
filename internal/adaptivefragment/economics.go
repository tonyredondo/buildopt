package adaptivefragment

import (
	"errors"
	"sort"
	"time"
)

const (
	maxEconomicObservations = 4096
	maxEconomicCostEvents   = 2
	maxEconomicHorizon      = 1000
	maxEconomicComponentMs  = 1_000_000_000_000
	maxSchemaInteger        = 9_007_199_254_740_991
)

// EconomicScope distinguishes one independently valued fragment from a
// composition whose joint effect cannot be attributed safely to its members.
type EconomicScope string

const (
	EconomicScopeFragment    EconomicScope = "FRAGMENT"
	EconomicScopeComposition EconomicScope = "COMPOSITION"
)

// EconomicCostEvent is an asynchronous learning or publication cost. The
// stable event identity prevents retries and repeated evidence from charging
// the same cost more than once.
type EconomicCostEvent struct {
	ID       string
	AmountMs uint64
}

// EconomicObservation is one requested ordinary build in chronological order.
// GrossSavedMs is signed: a slower activated candidate must remain negative.
type EconomicObservation struct {
	ObservationID          string
	Sequence               uint64
	ObservedAt             string
	Compatible             bool
	Activated              bool
	GrossSavedMs           int64
	SynchronousOverheadMs  uint64
	AsynchronousCostEvents []EconomicCostEvent
}

// EconomicPolicy fixes projection horizons, evidence decay and the maximum
// observed downside accepted by a later planner. It never changes observed
// values.
type EconomicPolicy struct {
	DecayPermille  uint64
	Horizons       []uint64
	RegretBudgetMs uint64
}

// EconomicSeries contains the exact fragment or composition revision being
// evaluated. FamilyID and RevisionID use the canonical adaptive identities.
type EconomicSeries struct {
	Scope              EconomicScope
	FamilyID           string
	RevisionID         string
	FragmentGeneration uint64
	EvidenceExpiresAt  string
	Observations       []EconomicObservation
	Policy             EconomicPolicy
}

// EconomicRecurrence is an exact fraction rather than an additive percentage.
type EconomicRecurrence struct {
	CompatibleBuilds uint64 `json:"compatibleBuilds"`
	ActivatedBuilds  uint64 `json:"activatedBuilds"`
	RequestedBuilds  uint64 `json:"requestedBuilds"`
}

// EconomicProjection reports future signed value separately from immutable
// observed totals.
type EconomicProjection struct {
	AdditionalBuilds        uint64 `json:"additionalBuilds"`
	ProjectedGrossSavedMs   int64  `json:"projectedGrossSavedMs"`
	ProjectedOverheadMs     uint64 `json:"projectedOverheadMs"`
	ProjectedNetMs          int64  `json:"projectedNetMs"`
	ObservedPlusProjectedMs int64  `json:"observedPlusProjectedMs"`
}

// EconomicRegret reports actual observed downside against an explicit budget;
// it never clips or replaces cumulative net value.
type EconomicRegret struct {
	ObservedDownsideMs uint64 `json:"observedDownsideMs"`
	BudgetMs           uint64 `json:"budgetMs"`
	WithinBudget       bool   `json:"withinBudget"`
}

// EconomicAssessment is the recomputable AF-005 result. Entry is compatible
// with the AF-002 immutable ledger schema; all projections remain derived.
type EconomicAssessment struct {
	Scope                   EconomicScope        `json:"scope"`
	Entry                   LedgerEntry          `json:"entry"`
	Recurrence              EconomicRecurrence   `json:"recurrence"`
	UniqueCostEventCount    uint64               `json:"uniqueCostEventCount"`
	NegativeActivatedBuilds uint64               `json:"negativeActivatedBuilds"`
	ObservedBreakEvenBuild  uint64               `json:"observedBreakEvenBuild,omitempty"`
	ProjectedBreakEvenBuild uint64               `json:"projectedBreakEvenBuild,omitempty"`
	DecayPermille           uint64               `json:"decayPermille"`
	Projections             []EconomicProjection `json:"projections"`
	Regret                  EconomicRegret       `json:"regret"`
}

// AssessEconomics recomputes one immutable signed ledger entry and bounded
// future projections. It consumes no percentages and grants no activation.
func AssessEconomics(series EconomicSeries) (EconomicAssessment, error) {
	if err := validateEconomicSeries(series); err != nil {
		return EconomicAssessment{}, err
	}
	assessment := EconomicAssessment{
		Scope: series.Scope,
		Entry: LedgerEntry{
			FamilyID: series.FamilyID, RevisionID: series.RevisionID,
			FragmentGeneration:  series.FragmentGeneration,
			RequestedBuildCount: uint64(len(series.Observations)),
			EvidenceExpiresAt:   series.EvidenceExpiresAt,
		},
		DecayPermille: series.Policy.DecayPermille,
	}
	costs := map[string]uint64{}
	var cumulative int64
	for _, observation := range series.Observations {
		if observation.Compatible {
			assessment.Recurrence.CompatibleBuilds++
		}
		if observation.Activated {
			assessment.Recurrence.ActivatedBuilds++
			assessment.Entry.GrossSavedMs += observation.GrossSavedMs
			if observation.GrossSavedMs < 0 {
				assessment.NegativeActivatedBuilds++
			}
		}
		assessment.Entry.SynchronousOverheadMs += observation.SynchronousOverheadMs
		cumulative += activatedGross(observation) - int64(observation.SynchronousOverheadMs)
		for _, event := range observation.AsynchronousCostEvents {
			amount, exists := costs[event.ID]
			if exists && amount != event.AmountMs {
				return EconomicAssessment{}, errors.New("adaptive fragment economic cost identity changed amount")
			}
			if !exists {
				costs[event.ID] = event.AmountMs
				assessment.Entry.OutOfBandLearningCostMs += event.AmountMs
				cumulative -= int64(event.AmountMs)
			}
		}
		if assessment.ObservedBreakEvenBuild == 0 && assessment.Entry.GrossSavedMs > 0 && cumulative >= 0 {
			assessment.ObservedBreakEvenBuild = observation.Sequence
		}
		assessment.Entry.LastObservedAt = observation.ObservedAt
	}
	assessment.Recurrence.RequestedBuilds = uint64(len(series.Observations))
	assessment.UniqueCostEventCount = uint64(len(costs))
	assessment.Entry.CumulativeNetMs = assessment.Entry.GrossSavedMs -
		int64(assessment.Entry.SynchronousOverheadMs) - int64(assessment.Entry.OutOfBandLearningCostMs)
	if cumulative != assessment.Entry.CumulativeNetMs {
		return EconomicAssessment{}, errors.New("adaptive fragment economic cumulative value is inconsistent")
	}
	if assessment.Entry.GrossSavedMs < -maxSchemaInteger || assessment.Entry.GrossSavedMs > maxSchemaInteger ||
		assessment.Entry.SynchronousOverheadMs > maxSchemaInteger ||
		assessment.Entry.OutOfBandLearningCostMs > maxSchemaInteger ||
		assessment.Entry.CumulativeNetMs < -maxSchemaInteger || assessment.Entry.CumulativeNetMs > maxSchemaInteger {
		return EconomicAssessment{}, errors.New("adaptive fragment economic ledger exceeds its schema bounds")
	}
	assessment.Projections, assessment.ProjectedBreakEvenBuild = projectEconomics(assessment, series.Policy)
	downside := uint64(0)
	if assessment.Entry.CumulativeNetMs < 0 {
		downside = uint64(-assessment.Entry.CumulativeNetMs)
	}
	assessment.Regret = EconomicRegret{
		ObservedDownsideMs: downside, BudgetMs: series.Policy.RegretBudgetMs,
		WithinBudget: downside <= series.Policy.RegretBudgetMs,
	}
	return assessment, nil
}

func validateEconomicSeries(series EconomicSeries) error {
	if (series.Scope != EconomicScopeFragment && series.Scope != EconomicScopeComposition) ||
		!validSHA(series.FamilyID) || !validSHA(series.RevisionID) || series.FragmentGeneration == 0 ||
		series.FragmentGeneration > maxSchemaInteger ||
		len(series.Observations) == 0 || len(series.Observations) > maxEconomicObservations ||
		series.Policy.DecayPermille > 1000 || len(series.Policy.Horizons) == 0 ||
		series.Policy.RegretBudgetMs > maxEconomicComponentMs {
		return errors.New("adaptive fragment economic series identity is invalid")
	}
	expiresAt, err := parseUTC(series.EvidenceExpiresAt)
	if err != nil {
		return errors.New("adaptive fragment economic expiry is invalid")
	}
	if !sort.SliceIsSorted(series.Policy.Horizons, func(left, right int) bool { return series.Policy.Horizons[left] < series.Policy.Horizons[right] }) {
		return errors.New("adaptive fragment economic horizons are not canonical")
	}
	seenHorizons := map[uint64]bool{}
	seenObservations := map[string]bool{}
	var previousSequence uint64
	var previousTime time.Time
	for _, horizon := range series.Policy.Horizons {
		if horizon == 0 || horizon > maxEconomicHorizon || seenHorizons[horizon] {
			return errors.New("adaptive fragment economic horizon is invalid")
		}
		seenHorizons[horizon] = true
	}
	for _, observation := range series.Observations {
		observedAt, err := parseUTC(observation.ObservedAt)
		if err != nil || observation.Sequence <= previousSequence || (!previousTime.IsZero() && observedAt.Before(previousTime)) ||
			observation.Sequence > maxSchemaInteger ||
			!validSHA(observation.ObservationID) || seenObservations[observation.ObservationID] ||
			observation.Activated && !observation.Compatible || !observation.Activated && observation.GrossSavedMs != 0 ||
			observation.GrossSavedMs < -maxEconomicComponentMs || observation.GrossSavedMs > maxEconomicComponentMs ||
			observation.SynchronousOverheadMs > maxEconomicComponentMs ||
			len(observation.AsynchronousCostEvents) > maxEconomicCostEvents || !expiresAt.After(observedAt) {
			return errors.New("adaptive fragment economic observation is invalid")
		}
		seenObservations[observation.ObservationID] = true
		for _, event := range observation.AsynchronousCostEvents {
			if !validSHA(event.ID) || event.AmountMs == 0 || event.AmountMs > maxEconomicComponentMs {
				return errors.New("adaptive fragment economic cost event is invalid")
			}
		}
		previousSequence = observation.Sequence
		previousTime = observedAt
	}
	return nil
}

func activatedGross(observation EconomicObservation) int64 {
	if observation.Activated {
		return observation.GrossSavedMs
	}
	return 0
}

func projectEconomics(assessment EconomicAssessment, policy EconomicPolicy) ([]EconomicProjection, uint64) {
	requested := int64(assessment.Recurrence.RequestedBuilds)
	meanExpectedGross := assessment.Entry.GrossSavedMs / requested
	meanOverhead := assessment.Entry.SynchronousOverheadMs / assessment.Recurrence.RequestedBuilds
	maximum := policy.Horizons[len(policy.Horizons)-1]
	projections := make([]EconomicProjection, 0, len(policy.Horizons))
	var projectedGross int64
	var projectedOverhead uint64
	var projectedBreakEven uint64
	decayedGross := meanExpectedGross
	horizonIndex := 0
	for build := uint64(1); build <= maximum; build++ {
		projectedGross += decayedGross
		projectedOverhead += meanOverhead
		projectedNet := projectedGross - int64(projectedOverhead)
		total := assessment.Entry.CumulativeNetMs + projectedNet
		if assessment.ObservedBreakEvenBuild == 0 && projectedBreakEven == 0 && total >= 0 && projectedGross > 0 {
			projectedBreakEven = assessment.Recurrence.RequestedBuilds + build
		}
		if build == policy.Horizons[horizonIndex] {
			projections = append(projections, EconomicProjection{
				AdditionalBuilds: build, ProjectedGrossSavedMs: projectedGross,
				ProjectedOverheadMs: projectedOverhead, ProjectedNetMs: projectedNet,
				ObservedPlusProjectedMs: total,
			})
			horizonIndex++
		}
		decayedGross = decayedGross * int64(policy.DecayPermille) / 1000
	}
	return projections, projectedBreakEven
}
