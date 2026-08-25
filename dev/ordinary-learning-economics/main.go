// Command ordinary-learning-economics generates and verifies the bounded
// evidence for ordinary-build learning economics in the BuildOpt POC.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tonyredondo/buildopt/internal/ordinarylearning"
)

const schemaVersion = "buildopt.evidence/poc-ordinary-learning-economics/v1"

type scenario struct {
	Name                             string                    `json:"name"`
	Decision                         ordinarylearning.Decision `json:"decision"`
	RemainingLearningBuildsAvoided   int                       `json:"remainingLearningBuildsAvoided"`
	RobustQualificationAutomatically bool                      `json:"robustQualificationAutomatically"`
}

type rejection struct {
	Name     string `json:"name"`
	Rejected bool   `json:"rejected"`
}

type boundaries struct {
	ProofOfConcept         bool   `json:"proofOfConcept"`
	ProductionAuthorized   bool   `json:"productionAuthorized"`
	PerformanceClaim       bool   `json:"performanceClaim"`
	ExtraMeasurementBuilds bool   `json:"extraMeasurementBuilds"`
	RobustGateWeakened     bool   `json:"robustGateWeakened"`
	SoakRequired           bool   `json:"soakRequired"`
	DesignPartnerRequired  bool   `json:"designPartnerRequired"`
	TestOptimization       string `json:"testOptimization"`
}

type evidence struct {
	SchemaVersion          string      `json:"schemaVersion"`
	WorkItem               string      `json:"workItem"`
	ImplementationRevision string      `json:"implementationRevision"`
	MaximumPaybackMatches  int         `json:"maximumPaybackMatches"`
	RequiredRobustPairs    int         `json:"requiredRobustPairs"`
	Scenarios              []scenario  `json:"scenarios"`
	UnsafeEvidence         []rejection `json:"unsafeEvidence"`
	Decision               string      `json:"decision"`
	ProductFailures        int         `json:"productFailures"`
	Boundaries             boundaries  `json:"boundaries"`
}

func main() {
	implementation := flag.String("implementation-revision", "", "40-character implementation commit")
	output := flag.String("output", "", "output JSON path")
	check := flag.String("check", "", "verify a committed evidence JSON")
	flag.Parse()
	if flag.NArg() != 0 || (*check == "" && (*implementation == "" || *output == "")) ||
		(*check != "" && (*implementation != "" || *output != "")) {
		fmt.Fprintln(os.Stderr, "usage: ordinary-learning-economics --implementation-revision SHA --output PATH | --check PATH")
		os.Exit(64)
	}
	if *check != "" {
		if err := verify(*check); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := write(*implementation, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var observed evidence
	if err := decoder.Decode(&observed); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return errors.New("evidence contains trailing JSON")
	}
	expected, err := generate(observed.ImplementationRevision)
	if err != nil {
		return err
	}
	expectedRaw, _ := json.MarshalIndent(expected, "", "  ")
	expectedRaw = append(expectedRaw, '\n')
	if !bytes.Equal(raw, expectedRaw) {
		return errors.New("evidence does not match the canonical ordinary-learning evaluation")
	}
	return nil
}

func write(implementation, path string) error {
	value, err := generate(implementation)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func generate(implementation string) (evidence, error) {
	if len(implementation) != 40 || strings.Trim(implementation, "0123456789abcdef") != "" {
		return evidence{}, errors.New("implementation revision must be one lowercase 40-character Git SHA")
	}
	positive, err := ordinarylearning.Evaluate(input(900, 7, 1300, 900, 1))
	if err != nil {
		return evidence{}, err
	}
	short, err := ordinarylearning.Evaluate(input(100, 4, 1300, 900, 0))
	if err != nil {
		return evidence{}, err
	}
	expensive, err := ordinarylearning.Evaluate(input(2100, 12, 1300, 900, 1))
	if err != nil {
		return evidence{}, err
	}
	regressive, err := ordinarylearning.Evaluate(input(100, 12, 900, 1300, 1))
	if err != nil {
		return evidence{}, err
	}
	completeInput := input(900, 12, 1300, 900, 1)
	for pair := 2; pair <= ordinarylearning.RequiredQualificationPairs; pair++ {
		arms := []string{"CONTROL", "CANDIDATE"}
		if pair%2 == 0 {
			arms = []string{"CANDIDATE", "CONTROL"}
		}
		for _, arm := range arms {
			duration := int64(1300)
			if arm == "CANDIDATE" {
				duration = 900
			}
			completeInput.Observations = append(completeInput.Observations, observation(len(completeInput.Observations), pair, arm, duration))
		}
	}
	complete, err := ordinarylearning.Evaluate(completeInput)
	if err != nil {
		return evidence{}, err
	}

	rejections := []struct {
		name   string
		mutate func(*ordinarylearning.Input)
	}{
		{"measurement-only build", func(value *ordinarylearning.Input) { value.Observations[2].MeasurementOnly = true }},
		{"structural drift", func(value *ordinarylearning.Input) {
			value.Observations[2].StructuralFingerprintSHA256 = strings.Repeat("b", 64)
		}},
		{"output drift", func(value *ordinarylearning.Input) { value.Observations[2].ExactOutputs = false }},
		{"product failure", func(value *ordinarylearning.Input) { value.Observations[2].ProductAttributableFailure = true }},
	}
	unsafe := make([]rejection, 0, len(rejections))
	for _, candidate := range rejections {
		value := input(900, 7, 1300, 900, 1)
		candidate.mutate(&value)
		_, evaluationErr := ordinarylearning.Evaluate(value)
		unsafe = append(unsafe, rejection{Name: candidate.name, Rejected: evaluationErr != nil})
	}
	for _, candidate := range unsafe {
		if !candidate.Rejected {
			return evidence{}, errors.New("unsafe ordinary-build evidence was accepted")
		}
	}
	if positive.Decision != ordinarylearning.DecisionContinue || positive.ProjectedPaybackMatches != 3 ||
		positive.ProjectedNetSavedMS != 1100 || short.Reason != ordinarylearning.ReasonLifetime ||
		expensive.ProjectedPaybackMatches != 6 || regressive.Reason != ordinarylearning.ReasonNoSaving ||
		complete.Decision != ordinarylearning.DecisionQualificationReady {
		return evidence{}, errors.New("ordinary-learning fixture decisions changed")
	}

	return evidence{
		SchemaVersion: schemaVersion, WorkItem: "POC-ORDINARY-LEARNING-ECONOMICS-001",
		ImplementationRevision: implementation,
		MaximumPaybackMatches:  ordinarylearning.MaximumPaybackMatches,
		RequiredRobustPairs:    ordinarylearning.RequiredQualificationPairs,
		Scenarios: []scenario{
			{Name: "positive first pair", Decision: positive, RemainingLearningBuildsAvoided: 0, RobustQualificationAutomatically: false},
			{Name: "insufficient compatible lifetime", Decision: short, RemainingLearningBuildsAvoided: 16, RobustQualificationAutomatically: false},
			{Name: "payback exceeds five matches", Decision: expensive, RemainingLearningBuildsAvoided: 14, RobustQualificationAutomatically: false},
			{Name: "candidate is regressive", Decision: regressive, RemainingLearningBuildsAvoided: 14, RobustQualificationAutomatically: false},
			{Name: "eight ordinary pairs complete", Decision: complete, RemainingLearningBuildsAvoided: 0, RobustQualificationAutomatically: false},
		},
		UnsafeEvidence: unsafe,
		Decision:       "ALLOW_BOUNDED_ORDINARY_BUILD_LEARNING", ProductFailures: 0,
		Boundaries: boundaries{
			ProofOfConcept: true, ProductionAuthorized: false, PerformanceClaim: false,
			ExtraMeasurementBuilds: false, RobustGateWeakened: false, SoakRequired: false,
			DesignPartnerRequired: false, TestOptimization: "OUT_OF_SCOPE",
		},
	}, nil
}

func input(cost int64, history int, control, candidate int64, pairs int) ordinarylearning.Input {
	value := ordinarylearning.Input{
		LearningCostMS: cost, HistoricalCompatibleMatches: history,
		Observations: []ordinarylearning.Observation{observation(0, 0, "BASELINE", control)},
	}
	for pair := 1; pair <= pairs; pair++ {
		arms := []string{"CONTROL", "CANDIDATE"}
		if pair%2 == 0 {
			arms = []string{"CANDIDATE", "CONTROL"}
		}
		for _, arm := range arms {
			duration := control
			if arm == "CANDIDATE" {
				duration = candidate
			}
			value.Observations = append(value.Observations, observation(len(value.Observations), pair, arm, duration))
		}
	}
	return value
}

func observation(sequence, pair int, arm string, duration int64) ordinarylearning.Observation {
	return ordinarylearning.Observation{
		Sequence: sequence, Pair: pair, Arm: arm, Source: ordinarylearning.ObservationSource,
		DurationMS: duration, StructuralFingerprintSHA256: strings.Repeat("a", 64),
		Graph:        ordinarylearning.Graph{TotalProjects: 10, SelectedProjects: 3, OmittedProjects: 7},
		ExactOutputs: true, Successful: true, StructurallyPortable: true,
		VolatileProducerCount: 2,
	}
}
