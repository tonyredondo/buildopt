package adaptivefragment

import (
	"errors"
	"sort"
)

const (
	PriorOutcomePositive    = "VALUE_POSITIVE"
	PriorOutcomeNonPositive = "VALUE_NON_POSITIVE"
)

// PriorFeatures describe repository-independent evidence that can be compared
// across Gradle builds. Repository identity is deliberately absent.
type PriorFeatures struct {
	TaskImplementationSHA256 string `json:"taskImplementationSha256"`
	PluginVersionSHA256      string `json:"pluginVersionSha256"`
	GradleMajor              uint64 `json:"gradleMajor"`
	TaskClass                string `json:"taskClass"`
	GraphShape               string `json:"graphShape"`
	OutputShape              string `json:"outputShape"`
}

// PriorObservation is locally checked source evidence for one generic
// hypothesis. RepositoryScopeSHA256 is provenance and same-scope exclusion
// only; RankHypotheses never uses it as a feature or score input.
type PriorObservation struct {
	EvidenceSHA256          string        `json:"evidenceSha256"`
	RepositoryScopeSHA256   string        `json:"repositoryScopeSha256"`
	Hypothesis              string        `json:"hypothesis"`
	Features                PriorFeatures `json:"features"`
	Outcome                 string        `json:"outcome"`
	ExactOutputs            bool          `json:"exactOutputs"`
	LocalCorrectnessPassed  bool          `json:"localCorrectnessPassed"`
	LocalValuePassed        bool          `json:"localValuePassed"`
	ProductAttributableFail bool          `json:"productAttributableFailure"`
}

// PriorQuery is a holdout workload for which BuildOpt may prioritize local
// exploration. The repository scope is excluded from its fingerprint.
type PriorQuery struct {
	RepositoryScopeSHA256 string        `json:"repositoryScopeSha256"`
	Features              PriorFeatures `json:"features"`
}

// PriorCandidate is a non-authorizing exploration hint. A caller must produce
// fresh local correctness and value evidence before any later activation.
type PriorCandidate struct {
	Rank                     uint64 `json:"rank"`
	Hypothesis               string `json:"hypothesis"`
	PriorityScore            int64  `json:"priorityScore"`
	BestStructuralSimilarity uint64 `json:"bestStructuralSimilarity"`
	PositiveSourceCount      uint64 `json:"positiveSourceCount"`
	NonPositiveSourceCount   uint64 `json:"nonPositiveSourceCount"`
	LocalCorrectnessRequired bool   `json:"localCorrectnessRequired"`
	LocalValueRequired       bool   `json:"localValueRequired"`
	ActivationAuthorized     bool   `json:"activationAuthorized"`
}

// PriorRanking is deterministic for structural features and source evidence.
// RepositoryIdentityUsedForScoring is always false by contract.
type PriorRanking struct {
	QueryFingerprint                 string           `json:"queryFingerprint"`
	SourceRepositoryCount            uint64           `json:"sourceRepositoryCount"`
	RepositoryIdentityUsedForScoring bool             `json:"repositoryIdentityUsedForScoring"`
	Candidates                       []PriorCandidate `json:"candidates"`
	ActivationAuthorized             bool             `json:"activationAuthorized"`
}

// RankHypotheses ranks cross-repository hypotheses by generic structural
// similarity and already-gated source outcomes. It cannot grant correctness,
// value or activation authority to the holdout repository.
func RankHypotheses(query PriorQuery, observations []PriorObservation) (PriorRanking, error) {
	if !validSHA(query.RepositoryScopeSHA256) {
		return PriorRanking{}, errors.New("adaptive fragment prior query repository is invalid")
	}
	if err := validatePriorFeatures(query.Features); err != nil {
		return PriorRanking{}, err
	}
	if len(observations) < 2 {
		return PriorRanking{}, errors.New("adaptive fragment prior requires cross-repository evidence")
	}

	fingerprintDocument, err := MarshalCanonicalDocument(query.Features)
	if err != nil {
		return PriorRanking{}, err
	}
	fingerprint, err := CanonicalDocumentSHA256(fingerprintDocument)
	if err != nil {
		return PriorRanking{}, err
	}

	type aggregate struct {
		bestSimilarity uint64
		positive       uint64
		nonPositive    uint64
		evidenceScore  int64
	}
	aggregates := map[string]*aggregate{}
	knownEvidence := map[string]bool{}
	knownRepositoryHypothesis := map[string]bool{}
	sourceRepositories := map[string]bool{}
	for _, observation := range observations {
		if err := validatePriorObservation(query, observation); err != nil {
			return PriorRanking{}, err
		}
		if knownEvidence[observation.EvidenceSHA256] {
			return PriorRanking{}, errors.New("adaptive fragment prior evidence is duplicated")
		}
		knownEvidence[observation.EvidenceSHA256] = true
		provenanceKey := observation.RepositoryScopeSHA256 + "\x00" + observation.Hypothesis
		if knownRepositoryHypothesis[provenanceKey] {
			return PriorRanking{}, errors.New("adaptive fragment prior repository hypothesis is duplicated")
		}
		knownRepositoryHypothesis[provenanceKey] = true
		sourceRepositories[observation.RepositoryScopeSHA256] = true

		similarity, eligible := priorSimilarity(query.Features, observation.Features)
		if !eligible {
			continue
		}
		entry := aggregates[observation.Hypothesis]
		if entry == nil {
			entry = &aggregate{}
			aggregates[observation.Hypothesis] = entry
		}
		if similarity > entry.bestSimilarity {
			entry.bestSimilarity = similarity
		}
		if observation.Outcome == PriorOutcomePositive {
			entry.positive++
			entry.evidenceScore += int64(similarity)
		} else {
			entry.nonPositive++
			entry.evidenceScore -= int64(similarity)
		}
	}
	if len(sourceRepositories) < 2 {
		return PriorRanking{}, errors.New("adaptive fragment prior requires two independent source repositories")
	}

	candidates := make([]PriorCandidate, 0, len(aggregates))
	for hypothesis, entry := range aggregates {
		candidates = append(candidates, PriorCandidate{
			Hypothesis:               hypothesis,
			PriorityScore:            int64(entry.bestSimilarity)*10 + entry.evidenceScore,
			BestStructuralSimilarity: entry.bestSimilarity,
			PositiveSourceCount:      entry.positive,
			NonPositiveSourceCount:   entry.nonPositive,
			LocalCorrectnessRequired: true,
			LocalValueRequired:       true,
		})
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].PriorityScore != candidates[right].PriorityScore {
			return candidates[left].PriorityScore > candidates[right].PriorityScore
		}
		if candidates[left].BestStructuralSimilarity != candidates[right].BestStructuralSimilarity {
			return candidates[left].BestStructuralSimilarity > candidates[right].BestStructuralSimilarity
		}
		return candidates[left].Hypothesis < candidates[right].Hypothesis
	})
	for index := range candidates {
		candidates[index].Rank = uint64(index + 1)
	}
	return PriorRanking{
		QueryFingerprint: fingerprint, SourceRepositoryCount: uint64(len(sourceRepositories)),
		Candidates: candidates,
	}, nil
}

func validatePriorObservation(query PriorQuery, observation PriorObservation) error {
	if !validSHA(observation.EvidenceSHA256) || !validSHA(observation.RepositoryScopeSHA256) ||
		observation.RepositoryScopeSHA256 == query.RepositoryScopeSHA256 || !validPriorHypothesis(observation.Hypothesis) {
		return errors.New("adaptive fragment prior observation identity is invalid")
	}
	if err := validatePriorFeatures(observation.Features); err != nil {
		return err
	}
	if !observation.ExactOutputs || !observation.LocalCorrectnessPassed || observation.ProductAttributableFail {
		return errors.New("adaptive fragment prior source correctness is invalid")
	}
	if (observation.Outcome == PriorOutcomePositive) != observation.LocalValuePassed ||
		(observation.Outcome != PriorOutcomePositive && observation.Outcome != PriorOutcomeNonPositive) {
		return errors.New("adaptive fragment prior source value is invalid")
	}
	return nil
}

func validatePriorFeatures(features PriorFeatures) error {
	if !validSHA(features.TaskImplementationSHA256) || !validSHA(features.PluginVersionSHA256) ||
		features.GradleMajor == 0 || !oneOf(features.TaskClass, "ARCHIVE", "CODE_GENERATION", "COMPILE", "CUSTOM") ||
		!oneOf(features.GraphShape, "INCLUDED_BUILD", "MULTI_PROJECT", "SINGLE_PROJECT") ||
		!oneOf(features.OutputShape, "AGGREGATE", "MULTI_ARTIFACT", "SINGLE_ARTIFACT") {
		return errors.New("adaptive fragment prior features are invalid")
	}
	return nil
}

func validPriorHypothesis(hypothesis string) bool {
	return oneOf(hypothesis, "CACHE_LOCALITY", "OUTPUT_MATERIALIZATION", "SUBGRAPH_REDUCTION", "TASK_CONTRACT_PATCH")
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func priorSimilarity(query, source PriorFeatures) (uint64, bool) {
	if query.TaskImplementationSHA256 != source.TaskImplementationSHA256 && query.PluginVersionSHA256 != source.PluginVersionSHA256 {
		return 0, false
	}
	var score uint64
	if query.TaskImplementationSHA256 == source.TaskImplementationSHA256 {
		score += 35
	}
	if query.PluginVersionSHA256 == source.PluginVersionSHA256 {
		score += 25
	}
	if query.GradleMajor == source.GradleMajor {
		score += 10
	}
	if query.TaskClass == source.TaskClass {
		score += 10
	}
	if query.GraphShape == source.GraphShape {
		score += 10
	}
	if query.OutputShape == source.OutputShape {
		score += 10
	}
	return score, true
}
