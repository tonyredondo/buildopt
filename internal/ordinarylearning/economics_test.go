package ordinarylearning

import (
	"strings"
	"testing"
)

func TestEvaluateRequiresOrdinaryExactCompatibleBuilds(t *testing.T) {
	input := usefulInput(900, 5, 1300, 900)
	decision, err := Evaluate(input)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != DecisionContinue || !decision.CalibrationAuthorized ||
		decision.ProjectedPaybackMatches != 3 || decision.ProjectedNetSavedMS != 1100 ||
		decision.RequestedBuilds != 3 || decision.MeasurementOnlyBuilds != 0 {
		t.Fatalf("ordinary decision = %+v", decision)
	}

	for name, mutate := range map[string]func(*Input){
		"measurement-only": func(value *Input) { value.Observations[2].MeasurementOnly = true },
		"output drift":     func(value *Input) { value.Observations[2].ExactOutputs = false },
		"structural drift": func(value *Input) { value.Observations[2].StructuralFingerprintSHA256 = strings.Repeat("b", 64) },
		"product failure":  func(value *Input) { value.Observations[2].ProductAttributableFailure = true },
	} {
		t.Run(name, func(t *testing.T) {
			value := usefulInput(900, 5, 1300, 900)
			mutate(&value)
			if _, err := Evaluate(value); err == nil {
				t.Fatal("unsafe ordinary evidence was accepted")
			}
		})
	}
}

func TestEvaluateRetainsNativeWhenLifetimeOrPaybackIsInsufficient(t *testing.T) {
	short := usefulInput(100, 4, 1300, 900)
	decision, err := Evaluate(short)
	if err != nil || decision.Decision != DecisionRetained || decision.Reason != ReasonLifetime {
		t.Fatalf("short lifetime = %+v / %v", decision, err)
	}

	expensive := usefulInput(2100, 12, 1300, 900)
	decision, err = Evaluate(expensive)
	if err != nil || decision.Decision != DecisionRetained || decision.Reason != ReasonPayback ||
		decision.ProjectedPaybackMatches != 6 {
		t.Fatalf("expensive learning = %+v / %v", decision, err)
	}

	regressive := usefulInput(100, 12, 900, 1300)
	decision, err = Evaluate(regressive)
	if err != nil || decision.Decision != DecisionRetained || decision.Reason != ReasonNoSaving {
		t.Fatalf("regressive learning = %+v / %v", decision, err)
	}
}

func TestEvaluateKeepsRobustQualificationSeparate(t *testing.T) {
	input := usefulInput(900, 12, 1300, 900)
	for pair := 2; pair <= RequiredQualificationPairs; pair++ {
		order := []string{"CONTROL", "CANDIDATE"}
		if pair%2 == 0 {
			order = []string{"CANDIDATE", "CONTROL"}
		}
		for _, arm := range order {
			duration := int64(1300)
			if arm == "CANDIDATE" {
				duration = 900
			}
			input.Observations = append(input.Observations, observation(len(input.Observations), pair, arm, duration))
		}
	}
	decision, err := Evaluate(input)
	if err != nil || decision.Decision != DecisionQualificationReady || !decision.CalibrationAuthorized ||
		decision.ObservedPairs != RequiredQualificationPairs {
		t.Fatalf("complete ordinary evidence = %+v / %v", decision, err)
	}
}

func usefulInput(cost int64, history int, control, candidate int64) Input {
	return Input{
		LearningCostMS: cost, HistoricalCompatibleMatches: history,
		Observations: []Observation{
			observation(0, 0, "BASELINE", control),
			observation(1, 1, "CONTROL", control),
			observation(2, 1, "CANDIDATE", candidate),
		},
	}
}

func observation(sequence, pair int, arm string, duration int64) Observation {
	return Observation{
		Sequence: sequence, Pair: pair, Arm: arm, Source: ObservationSource,
		DurationMS: duration, StructuralFingerprintSHA256: strings.Repeat("a", 64),
		Graph:        Graph{TotalProjects: 10, SelectedProjects: 3, OmittedProjects: 7},
		ExactOutputs: true, Successful: true, StructurallyPortable: true,
		VolatileProducerCount: 2,
	}
}
