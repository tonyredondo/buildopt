// Package stickyvalue owns conservative, integer-only value qualification for
// sticky-wrapper paired evidence. It never authorizes execution by itself.
package stickyvalue

import (
	"errors"
	"math"
	"sort"
)

const bootstrapSamples = 4096

// Pair is one preassigned native/candidate comparison. Effect is defined as
// native minus candidate, so a positive value favors the candidate.
type Pair struct {
	PairID            string `json:"pairId"`
	Order             string `json:"order"`
	NativeWallNs      int64  `json:"nativeWallNs"`
	CandidateWallNs   int64  `json:"candidateWallNs"`
	OutputsEquivalent bool   `json:"outputsEquivalent"`
	NativeFailure     bool   `json:"nativeFailure"`
	CandidateFailure  bool   `json:"candidateFailure"`
}

// Costs contains every BuildOpt-owned cost category. Pointers distinguish an
// observed zero from unavailable evidence; all fields are required.
type Costs struct {
	BootstrapNs   *int64 `json:"bootstrapNs"`
	ObservationNs *int64 `json:"observationNs"`
	TrialNs       *int64 `json:"trialNs"`
	CacheNs       *int64 `json:"cacheNs"`
	StateNs       *int64 `json:"stateNs"`
	FallbackNs    *int64 `json:"fallbackNs"`
	ExecutionNs   *int64 `json:"executionNs"`
	PublicationNs *int64 `json:"publicationNs"`
	ValidationNs  *int64 `json:"validationNs"`
}

// Evaluation is fully recomputable from Pair and Costs evidence. Qualification
// requires exact successful pairs, a positive conservative interval and net
// saving after every known BuildOpt cost.
type Evaluation struct {
	PairEffectsNs     []int64  `json:"pairEffectsNs"`
	MeanEffectNs      int64    `json:"meanEffectNs"`
	LowerBoundNs      int64    `json:"lowerBoundNs"`
	UpperBoundNs      int64    `json:"upperBoundNs"`
	PositivePairCount int      `json:"positivePairCount"`
	NativeP95Ns       int64    `json:"nativeP95Ns"`
	CandidateP95Ns    int64    `json:"candidateP95Ns"`
	GrossSavingNs     int64    `json:"grossSavingNs"`
	TotalCostNs       int64    `json:"totalCostNs"`
	NetSavingNs       int64    `json:"netSavingNs"`
	Qualified         bool     `json:"qualified"`
	Reasons           []string `json:"reasons"`
}

// Evaluate computes a deterministic paired bootstrap and exact cost ledger.
func Evaluate(pairs []Pair, costs Costs) (Evaluation, error) {
	result := Evaluation{}
	if len(pairs) == 0 {
		return result, errors.New("sticky value requires paired evidence")
	}
	native := make([]int64, 0, len(pairs))
	candidate := make([]int64, 0, len(pairs))
	seen := make(map[string]struct{}, len(pairs))
	var effectTotal int64
	for _, pair := range pairs {
		if pair.PairID == "" || (pair.Order != "NATIVE_FIRST" && pair.Order != "CANDIDATE_FIRST") || pair.NativeWallNs < 0 || pair.CandidateWallNs < 0 {
			return Evaluation{}, errors.New("sticky value pair is invalid")
		}
		if _, ok := seen[pair.PairID]; ok {
			return Evaluation{}, errors.New("sticky value pair identifier is duplicated")
		}
		seen[pair.PairID] = struct{}{}
		if pair.NativeFailure || pair.CandidateFailure || !pair.OutputsEquivalent {
			result.Reasons = appendUnique(result.Reasons, "PAIR_NOT_COMPARABLE")
		}
		effect, ok := checkedSub(pair.NativeWallNs, pair.CandidateWallNs)
		if !ok {
			return Evaluation{}, errors.New("sticky value pair effect overflows")
		}
		effectTotal, ok = checkedAdd(effectTotal, effect)
		if !ok {
			return Evaluation{}, errors.New("sticky value effect total overflows")
		}
		result.PairEffectsNs = append(result.PairEffectsNs, effect)
		if effect > 0 {
			result.PositivePairCount++
		}
		native = append(native, pair.NativeWallNs)
		candidate = append(candidate, pair.CandidateWallNs)
	}
	result.MeanEffectNs = effectTotal / int64(len(pairs))
	result.GrossSavingNs = effectTotal
	result.NativeP95Ns = nearestRank95(native)
	result.CandidateP95Ns = nearestRank95(candidate)
	lowerBound, upperBound, bootstrapOK := pairedBootstrap95(result.PairEffectsNs)
	if !bootstrapOK {
		return Evaluation{}, errors.New("sticky value bootstrap overflows")
	}
	result.LowerBoundNs, result.UpperBoundNs = lowerBound, upperBound

	values := []*int64{costs.BootstrapNs, costs.ObservationNs, costs.TrialNs, costs.CacheNs, costs.StateNs, costs.FallbackNs, costs.ExecutionNs, costs.PublicationNs, costs.ValidationNs}
	for _, value := range values {
		if value == nil || *value < 0 {
			result.Reasons = appendUnique(result.Reasons, "COST_UNAVAILABLE")
			continue
		}
		var ok bool
		result.TotalCostNs, ok = checkedAdd(result.TotalCostNs, *value)
		if !ok {
			return Evaluation{}, errors.New("sticky value cost total overflows")
		}
	}
	var ok bool
	result.NetSavingNs, ok = checkedSub(result.GrossSavingNs, result.TotalCostNs)
	if !ok {
		return Evaluation{}, errors.New("sticky value net saving overflows")
	}
	if result.LowerBoundNs <= 0 {
		result.Reasons = appendUnique(result.Reasons, "LOWER_BOUND_NOT_POSITIVE")
	}
	if result.CandidateP95Ns > result.NativeP95Ns {
		result.Reasons = appendUnique(result.Reasons, "P95_REGRESSED")
	}
	if result.NetSavingNs <= 0 {
		result.Reasons = appendUnique(result.Reasons, "NET_VALUE_NOT_POSITIVE")
	}
	result.Qualified = len(result.Reasons) == 0
	if result.Qualified {
		result.Reasons = []string{"QUALIFIED_CONSERVATIVE_VALUE"}
	}
	return result, nil
}

func checkedAdd(a, b int64) (int64, bool) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, false
	}
	return a + b, true
}

func checkedSub(a, b int64) (int64, bool) {
	if b == math.MinInt64 {
		if a >= 0 {
			return 0, false
		}
		return a - b, true
	}
	return checkedAdd(a, -b)
}

func nearestRank95(values []int64) int64 {
	copyValues := append([]int64(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	index := (95*len(copyValues)+99)/100 - 1
	return copyValues[index]
}

func pairedBootstrap95(effects []int64) (int64, int64, bool) {
	replicates := make([]int64, bootstrapSamples)
	state := uint64(0x9e3779b97f4a7c15)
	for sample := range replicates {
		var sum int64
		for range effects {
			state = state*6364136223846793005 + 1442695040888963407
			value := effects[int(state%uint64(len(effects)))]
			var ok bool
			sum, ok = checkedAdd(sum, value)
			if !ok {
				return 0, 0, false
			}
		}
		replicates[sample] = sum / int64(len(effects))
	}
	sort.Slice(replicates, func(i, j int) bool { return replicates[i] < replicates[j] })
	return replicates[102], replicates[3993], true
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
