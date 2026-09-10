//go:build linux && amd64

package main

import "errors"

type ExternalCostRecord struct {
	Schema      string    `json:"schema"`
	ID          string    `json:"id"`
	Class       string    `json:"class"`
	Purpose     string    `json:"purpose"`
	Replication int       `json:"replication"`
	Ordinal     int       `json:"ordinal"`
	Start       Stamp     `json:"start"`
	End         Stamp     `json:"end"`
	Evidence    []Binding `json:"evidence"`
	Explanation string    `json:"explanation"`
}

func externalCosts(m Manifest) ([]Cost, error) {
	result := []Cost{}
	seen := map[string]bool{}
	for _, binding := range m.ExternalCosts {
		if err := validateBinding(binding); err != nil {
			return nil, err
		}
		var raw ExternalCostRecord
		if err := readJSON(binding.Path, &raw); err != nil {
			return nil, err
		}
		if raw.Schema != "buildopt.history-replay/external-cost/v1" || !namePattern.MatchString(raw.ID) || seen[raw.ID] || raw.Explanation == "" || raw.Replication < 1 || raw.Replication > m.Replications || raw.Ordinal < 0 || raw.Ordinal > m.ExecutionEnd || len(raw.Evidence) == 0 {
			return nil, errors.New("invalid/unsubstantiated external cost")
		}
		seen[raw.ID] = true
		if err := checkStampInterval(raw.Start, raw.End); err != nil {
			return nil, err
		}
		for _, e := range raw.Evidence {
			if inside(m.RunRoot, e.Path) {
				return nil, errors.New("external preparation cannot borrow future run evidence")
			}
			if err := validateBinding(e); err != nil {
				return nil, err
			}
		}
		switch raw.Purpose {
		case "adoption":
			if raw.Class != "customer-machine" || raw.Ordinal > m.PrefixEnd {
				return nil, errors.New("external adoption misclassified")
			}
		case "maintenance":
			if raw.Class != "customer-machine" || raw.Ordinal <= m.PrefixEnd {
				return nil, errors.New("external maintenance misclassified")
			}
		case "human-review":
			if raw.Class != "customer-human" {
				return nil, errors.New("human review misclassified")
			}
		case "research-external":
			if raw.Class != "research" {
				return nil, errors.New("external research misclassified")
			}
		default:
			return nil, errors.New("unknown external cost purpose")
		}
		result = append(result, Cost{ID: "external-" + raw.ID, Class: raw.Class, Purpose: raw.Purpose, Replication: raw.Replication, Ordinal: raw.Ordinal, Arm: "I", StartNS: raw.Start.NS, EndNS: raw.End.NS, DurationNS: raw.End.NS - raw.Start.NS})
	}
	if m.Phase == "CONFIRMATION" {
		for rep := 1; rep <= m.Replications; rep++ {
			found := false
			for _, c := range result {
				if c.Replication == rep && c.Purpose == "adoption" {
					found = true
				}
			}
			if !found {
				return nil, errors.New("confirmation needs explicit evidenced external adoption cost for each replication, including measured zero")
			}
		}
	}
	return result, nil
}
