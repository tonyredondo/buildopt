package adaptivefragment

import (
	"errors"
	"sort"
)

// ShadowApplicability is a historical, non-authorizing compatibility result.
// It records what a fragment could have retained at one ordinary build without
// claiming that the fragment was safe or economical to activate.
type ShadowApplicability string

const (
	ShadowApplicable   ShadowApplicability = "APPLICABLE"
	ShadowSuspended    ShadowApplicability = "SUSPENDED"
	ShadowNotEvaluated ShadowApplicability = "NOT_EVALUATED"
)

// ShadowObservation contains only facts observed at one chronological build.
// Later observations must never be folded into this record.
type ShadowObservation struct {
	Sequence             int
	Revision             string
	OriginalSelected     bool
	OriginalReason       string
	WrapperMatches       bool
	ExactRequiredOutputs bool
}

// ShadowFragmentDecision records one fragment's compatibility at one build.
type ShadowFragmentDecision struct {
	Kind          Kind                `json:"kind"`
	Applicability ShadowApplicability `json:"applicability"`
	Reason        string              `json:"reason"`
}

// ShadowReplayDecision is a deterministic decomposition of one historical
// whole-profile decision. MaxSourceSequence makes the no-lookahead boundary
// explicit and independently checkable.
type ShadowReplayDecision struct {
	Sequence              int                      `json:"sequence"`
	Revision              string                   `json:"revision"`
	MaxSourceSequence     int                      `json:"maxSourceSequence"`
	OriginalSelected      bool                     `json:"originalSelected"`
	ReproducedSelected    bool                     `json:"reproducedSelected"`
	EconomicAuthorization bool                     `json:"economicAuthorization"`
	Fragments             []ShadowFragmentDecision `json:"fragments"`
}

// ReplayShadowProfile decomposes chronological whole-profile observations
// into subgraph and output-materialization compatibility. It is deliberately
// shadow-only: compatibility never authorizes execution.
func ReplayShadowProfile(observations []ShadowObservation) ([]ShadowReplayDecision, error) {
	if len(observations) == 0 {
		return nil, errors.New("adaptive fragment shadow history is empty")
	}
	replay := make([]ShadowReplayDecision, 0, len(observations))
	previous := 0
	for _, observation := range observations {
		if observation.Sequence <= previous || observation.Revision == "" || !observation.WrapperMatches || !observation.ExactRequiredOutputs {
			return nil, errors.New("adaptive fragment shadow observation is invalid")
		}
		decision, err := replayShadowObservation(observation)
		if err != nil {
			return nil, err
		}
		replay = append(replay, decision)
		previous = observation.Sequence
	}
	return replay, nil
}

func replayShadowObservation(observation ShadowObservation) (ShadowReplayDecision, error) {
	decision := ShadowReplayDecision{
		Sequence: observation.Sequence, Revision: observation.Revision,
		MaxSourceSequence: observation.Sequence, OriginalSelected: observation.OriginalSelected,
	}
	switch observation.OriginalReason {
	case "QUALIFIED_PROFILE_SELECTED":
		decision.ReproducedSelected = true
		decision.EconomicAuthorization = true
		decision.Fragments = []ShadowFragmentDecision{
			{Kind: KindSubgraph, Applicability: ShadowApplicable, Reason: "STRUCTURAL_PROFILE_COMPATIBLE"},
			{Kind: KindOutputMaterialization, Applicability: ShadowApplicable, Reason: "QUALIFIED_OUTPUTS_CURRENT"},
		}
	case "QUALIFIED_PROFILE_OUTPUTS_REFRESHED":
		decision.Fragments = []ShadowFragmentDecision{
			{Kind: KindSubgraph, Applicability: ShadowApplicable, Reason: "STRUCTURAL_PROFILE_COMPATIBLE"},
			{Kind: KindOutputMaterialization, Applicability: ShadowSuspended, Reason: "OUTPUT_BYTES_REFRESHED"},
		}
	case "ECONOMIC_PREQUALIFICATION_REJECTED":
		decision.Fragments = []ShadowFragmentDecision{
			{Kind: KindSubgraph, Applicability: ShadowApplicable, Reason: "STRUCTURAL_PROFILE_COMPATIBLE"},
			{Kind: KindOutputMaterialization, Applicability: ShadowNotEvaluated, Reason: "ECONOMIC_GATE_RETAINED_NATIVE"},
		}
	default:
		return ShadowReplayDecision{}, errors.New("adaptive fragment shadow reason is unsupported")
	}
	if decision.ReproducedSelected != observation.OriginalSelected {
		return ShadowReplayDecision{}, errors.New("adaptive fragment shadow replay did not reproduce selection")
	}
	sort.Slice(decision.Fragments, func(left, right int) bool {
		return decision.Fragments[left].Kind < decision.Fragments[right].Kind
	})
	return decision, nil
}
