// Package ordinarylearning evaluates whether evidence collected from requested
// Gradle builds is economically strong enough to justify continued POC
// learning. It never authorizes production behavior or additional builds.
package ordinarylearning

import (
	"encoding/hex"
	"errors"
	"math"
	"strings"
)

const (
	SchemaVersion              = "buildopt.poc/ordinary-learning-economics/v1"
	ObservationSource          = "REQUESTED_ORDINARY_BUILD"
	DecisionPending            = "PENDING"
	DecisionContinue           = "CONTINUE_ROBUST_LEARNING"
	DecisionQualificationReady = "QUALIFICATION_READY"
	DecisionRetained           = "NATIVE_RETAINED"
	MaximumPaybackMatches      = 5
	RequiredQualificationPairs = 8
)

const (
	ReasonMoreEvidence       = "ORDINARY_PAIR_PENDING"
	ReasonLifetime           = "COMPATIBLE_LIFETIME_INSUFFICIENT"
	ReasonNoSaving           = "ORDINARY_SAVING_NOT_POSITIVE"
	ReasonPayback            = "FIVE_MATCH_PAYBACK_EXCEEDED"
	ReasonContinue           = "FIVE_MATCH_PAYBACK_PROJECTED"
	ReasonQualificationReady = "ROBUST_ORDINARY_EVIDENCE_READY"
)

// Graph records the complete project reduction observed during the requested
// build. It is structural evidence, not an estimate of time saved.
type Graph struct {
	TotalProjects    int `json:"totalProjects"`
	SelectedProjects int `json:"selectedProjects"`
	OmittedProjects  int `json:"omittedProjects"`
}

// Observation is one useful customer-requested build. MeasurementOnly must
// remain false: the POC cannot create extra customer work to improve its own
// evidence.
type Observation struct {
	Sequence                    int    `json:"sequence"`
	Pair                        int    `json:"pair"`
	Arm                         string `json:"arm"`
	Source                      string `json:"source"`
	DurationMS                  int64  `json:"durationMs"`
	StructuralFingerprintSHA256 string `json:"structuralFingerprintSha256"`
	Graph                       Graph  `json:"graph"`
	ExactOutputs                bool   `json:"exactOutputs"`
	Successful                  bool   `json:"successful"`
	StructurallyPortable        bool   `json:"structurallyPortable"`
	VolatileProducerCount       int    `json:"volatileProducerCount"`
	MeasurementOnly             bool   `json:"measurementOnly"`
	ProductAttributableFailure  bool   `json:"productAttributableFailure"`
}

// Input carries immutable ordinary-build observations plus a bounded
// recurrence projection from first-parent history. Historical matches are a
// planning signal only; they never replace direct build evidence.
type Input struct {
	LearningCostMS              int64         `json:"learningCostMs"`
	HistoricalCompatibleMatches int           `json:"historicalCompatibleMatches"`
	Observations                []Observation `json:"observations"`
}

// Decision is the fail-closed economic gate persisted with the learning
// checkpoint. CalibrationAuthorized means only that ordinary paired learning
// may continue; it is never product or production authority.
type Decision struct {
	SchemaVersion                string  `json:"schemaVersion"`
	Decision                     string  `json:"decision"`
	Reason                       string  `json:"reason"`
	EvidenceSource               string  `json:"evidenceSource"`
	RequestedBuilds              int     `json:"requestedBuilds"`
	MeasurementOnlyBuilds        int     `json:"measurementOnlyBuilds"`
	CompatibleBuilds             int     `json:"compatibleBuilds"`
	SuccessfulBuilds             int     `json:"successfulBuilds"`
	ExactOutputBuilds            int     `json:"exactOutputBuilds"`
	StructurallyPortableBuilds   int     `json:"structurallyPortableBuilds"`
	MaximumVolatileProducerCount int     `json:"maximumVolatileProducerCount"`
	ObservedPairs                int     `json:"observedPairs"`
	ControlMeanMS                float64 `json:"controlMeanMs"`
	CandidateMeanMS              float64 `json:"candidateMeanMs"`
	MeanSavedMS                  float64 `json:"meanSavedMs"`
	ReductionRatio               float64 `json:"reductionRatio"`
	LearningCostMS               int64   `json:"learningCostMs"`
	HistoricalCompatibleMatches  int     `json:"historicalCompatibleMatches"`
	ProjectedCompatibleMatches   int     `json:"projectedCompatibleMatches"`
	MaximumPaybackMatches        int     `json:"maximumPaybackMatches"`
	ProjectedPaybackMatches      int     `json:"projectedPaybackMatches"`
	ProjectedGrossSavedMS        float64 `json:"projectedGrossSavedMs"`
	ProjectedNetSavedMS          float64 `json:"projectedNetSavedMs"`
	CalibrationAuthorized        bool    `json:"calibrationAuthorized"`
	ProductionAuthorized         bool    `json:"productionAuthorized"`
	TestOptimization             string  `json:"testOptimization"`
}

// Evaluate validates every observation and applies the fixed five-match POC
// payback horizon. It deliberately makes no statistical qualification claim;
// the existing eight-pair robust gate remains mandatory after this decision.
func Evaluate(input Input) (Decision, error) {
	decision := Decision{
		SchemaVersion: SchemaVersion, Decision: DecisionPending,
		Reason: ReasonMoreEvidence, EvidenceSource: ObservationSource,
		LearningCostMS:              input.LearningCostMS,
		HistoricalCompatibleMatches: input.HistoricalCompatibleMatches,
		MaximumPaybackMatches:       MaximumPaybackMatches,
		ProductionAuthorized:        false, TestOptimization: "OUT_OF_SCOPE",
	}
	if input.LearningCostMS < 1 || input.HistoricalCompatibleMatches < 0 || len(input.Observations) == 0 {
		return Decision{}, errors.New("ordinary learning input is incomplete")
	}
	first := input.Observations[0]
	if err := validateObservation(first, true, first); err != nil {
		return Decision{}, err
	}
	for index, observation := range input.Observations {
		if observation.Sequence != index {
			return Decision{}, errors.New("ordinary learning observation sequence is invalid")
		}
		if err := validateObservation(observation, index == 0, first); err != nil {
			return Decision{}, err
		}
		decision.RequestedBuilds++
		if observation.MeasurementOnly {
			decision.MeasurementOnlyBuilds++
		}
		if observation.StructuralFingerprintSHA256 == first.StructuralFingerprintSHA256 && observation.Graph == first.Graph {
			decision.CompatibleBuilds++
		}
		if observation.Successful {
			decision.SuccessfulBuilds++
		}
		if observation.ExactOutputs {
			decision.ExactOutputBuilds++
		}
		if observation.StructurallyPortable {
			decision.StructurallyPortableBuilds++
		}
		decision.MaximumVolatileProducerCount = max(decision.MaximumVolatileProducerCount, observation.VolatileProducerCount)
	}
	if decision.MeasurementOnlyBuilds != 0 || decision.CompatibleBuilds != decision.RequestedBuilds ||
		decision.SuccessfulBuilds != decision.RequestedBuilds || decision.ExactOutputBuilds != decision.RequestedBuilds ||
		decision.StructurallyPortableBuilds != decision.RequestedBuilds {
		return Decision{}, errors.New("ordinary learning observations are not safe and compatible")
	}
	decision.ProjectedCompatibleMatches = min(input.HistoricalCompatibleMatches, MaximumPaybackMatches)
	if decision.ProjectedCompatibleMatches < MaximumPaybackMatches {
		decision.Decision = DecisionRetained
		decision.Reason = ReasonLifetime
		return decision, nil
	}
	if len(input.Observations) == 1 || (len(input.Observations)-1)%2 != 0 {
		return decision, nil
	}

	var controlTotal, candidateTotal int64
	for index := 1; index < len(input.Observations); index += 2 {
		firstArm := input.Observations[index]
		secondArm := input.Observations[index+1]
		if firstArm.Pair != secondArm.Pair || firstArm.Pair != (index+1)/2 ||
			!pairedArms(firstArm.Arm, secondArm.Arm) {
			return Decision{}, errors.New("ordinary learning pair is invalid")
		}
		control, candidate := firstArm, secondArm
		if firstArm.Arm == "CANDIDATE" {
			control, candidate = secondArm, firstArm
		}
		controlTotal += control.DurationMS
		candidateTotal += candidate.DurationMS
		decision.ObservedPairs++
	}
	decision.ControlMeanMS = float64(controlTotal) / float64(decision.ObservedPairs)
	decision.CandidateMeanMS = float64(candidateTotal) / float64(decision.ObservedPairs)
	decision.MeanSavedMS = decision.ControlMeanMS - decision.CandidateMeanMS
	decision.ReductionRatio = decision.MeanSavedMS / decision.ControlMeanMS
	if decision.MeanSavedMS <= 0 {
		decision.Decision = DecisionRetained
		decision.Reason = ReasonNoSaving
		return decision, nil
	}
	decision.ProjectedPaybackMatches = int(math.Ceil(float64(input.LearningCostMS) / decision.MeanSavedMS))
	decision.ProjectedGrossSavedMS = decision.MeanSavedMS * float64(decision.ProjectedCompatibleMatches)
	decision.ProjectedNetSavedMS = decision.ProjectedGrossSavedMS - float64(input.LearningCostMS)
	if decision.ProjectedPaybackMatches < 1 || decision.ProjectedPaybackMatches > MaximumPaybackMatches ||
		decision.ProjectedPaybackMatches > decision.ProjectedCompatibleMatches || decision.ProjectedNetSavedMS <= 0 {
		decision.Decision = DecisionRetained
		decision.Reason = ReasonPayback
		return decision, nil
	}
	decision.Decision = DecisionContinue
	decision.Reason = ReasonContinue
	decision.CalibrationAuthorized = true
	if decision.ObservedPairs == RequiredQualificationPairs {
		decision.Decision = DecisionQualificationReady
		decision.Reason = ReasonQualificationReady
	}
	return decision, nil
}

func validateObservation(observation Observation, baseline bool, expected Observation) error {
	if observation.Sequence < 0 || observation.DurationMS < 1 || observation.Source != ObservationSource ||
		!validSHA(observation.StructuralFingerprintSHA256) || observation.Graph.TotalProjects < 2 ||
		observation.Graph.SelectedProjects < 1 || observation.Graph.OmittedProjects < 1 ||
		observation.Graph.SelectedProjects+observation.Graph.OmittedProjects != observation.Graph.TotalProjects ||
		observation.VolatileProducerCount < 0 || observation.MeasurementOnly ||
		!observation.Successful || !observation.ExactOutputs || !observation.StructurallyPortable ||
		observation.ProductAttributableFailure {
		return errors.New("ordinary learning observation is invalid")
	}
	if observation.StructuralFingerprintSHA256 != expected.StructuralFingerprintSHA256 || observation.Graph != expected.Graph {
		return errors.New("ordinary learning structural context drifted")
	}
	if baseline {
		if observation.Sequence != 0 || observation.Pair != 0 || observation.Arm != "BASELINE" {
			return errors.New("ordinary learning baseline is invalid")
		}
		return nil
	}
	if observation.Sequence < 1 || observation.Pair < 1 || (observation.Arm != "CONTROL" && observation.Arm != "CANDIDATE") {
		return errors.New("ordinary learning paired observation is invalid")
	}
	return nil
}

func pairedArms(first, second string) bool {
	return first == "CONTROL" && second == "CANDIDATE" || first == "CANDIDATE" && second == "CONTROL"
}

func validSHA(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
