package main

import (
	"errors"
	"math"
)

type costEstimate struct {
	Profile      string `json:"profile"`
	UpperBoundMs int64  `json:"upperBoundMs"`
	Provenance   string `json:"provenance"`
}

type budgetInput struct {
	CeilingSeconds     int64          `json:"ceilingSeconds,omitempty"`
	ChargedSeconds     int64          `json:"chargedSeconds"`
	PreparationSeconds int64          `json:"preparationSeconds"`
	ManagementSeconds  int64          `json:"managementSeconds"`
	Estimates          []costEstimate `json:"estimates"`
}

type budgetReport struct {
	CeilingSeconds            int64    `json:"ceilingSeconds"`
	Schema                    string   `json:"schemaVersion"`
	Decision                  string   `json:"decision"`
	RemainingSeconds          int64    `json:"remainingSeconds"`
	RequiredMilliseconds      int64    `json:"requiredMilliseconds"`
	UnsizedProfiles           []string `json:"unsizedProfiles"`
	PublicExecutionAuthorized bool     `json:"publicExecutionAuthorized"`
	Prerequisites             []string `json:"prerequisites"`
}

// admit checks arithmetic, never grants execution authority. An operator's
// estimate is explicitly hypothetical; historical or fixture timing is not a
// current-host estimate. Nested TestKit time is included in each outer bound,
// while its six starts are counted separately by the protocol.
func admit(p protocol, b budgetInput) (budgetReport, error) {
	r := budgetReport{Schema: "buildopt.eic/budget/v1", Decision: "UNPROVED", UnsizedProfiles: []string{}, Prerequisites: []string{
		"public subject source/runtime/dependency and daemon/compiler binding",
		"native graph-derived output contract and six Gradle mutation fixtures",
		"measured wrapper/native boundaries and observed nested Gradle start accounting",
		"same-host sizing and explicit execution allocation",
	}}
	if err := validateProtocol(p); err != nil {
		return r, err
	}
	ceiling := b.CeilingSeconds
	if ceiling == 0 {
		ceiling = int64(p.CeilingSeconds)
	}
	if ceiling < int64(p.CeilingSeconds) || ceiling > math.MaxInt32 {
		return r, errors.New("invalid total planning envelope")
	}
	r.CeilingSeconds = ceiling
	if b.ChargedSeconds < 4440 || b.ChargedSeconds > math.MaxInt32 || b.PreparationSeconds < 0 || b.PreparationSeconds > ceiling || b.ManagementSeconds < 0 || b.ManagementSeconds > ceiling {
		return r, errors.New("invalid cost ledger; cannot reset prior 4440 seconds")
	}
	r.RemainingSeconds = max(0, ceiling-b.ChargedSeconds)
	profiles := costProfiles(p)
	allowed := map[string]bool{}
	for _, key := range profiles {
		allowed[key] = true
	}
	values := map[string]int64{}
	seen := map[string]bool{}
	for _, estimate := range b.Estimates {
		if !allowed[estimate.Profile] || seen[estimate.Profile] || estimate.UpperBoundMs <= 0 || estimate.UpperBoundMs > 7200000 {
			return r, errors.New("unknown/duplicate profile or invalid bound")
		}
		seen[estimate.Profile] = true
		switch estimate.Provenance {
		case "OPERATOR_ESTIMATE":
			values[estimate.Profile] = estimate.UpperBoundMs
		case "HISTORICAL", "LOCAL_FIXTURE": // These cannot size the public workflow.
		default:
			return r, errors.New("unknown estimate provenance")
		}
	}
	r.RequiredMilliseconds = (b.PreparationSeconds + b.ManagementSeconds + int64(p.QuiescenceSeconds+p.ClosureSeconds)) * 1000
	for _, key := range profiles {
		if values[key] == 0 {
			r.UnsizedProfiles = append(r.UnsizedProfiles, key)
		}
	}
	for _, row := range p.Rows {
		r.RequiredMilliseconds += values[costProfile(row)]
	}
	r.RequiredMilliseconds += int64(p.ReserveStarts) * values["infrastructure-reserve"]
	switch {
	case r.RemainingSeconds == 0:
		r.Decision = "BUDGET_EXHAUSTED"
	case r.RequiredMilliseconds > r.RemainingSeconds*1000:
		r.Decision = "DOES_NOT_FIT"
	case len(r.UnsizedProfiles) > 0:
		r.Decision = "UNPROVED"
	default:
		r.Decision = "FITS_ESTIMATE_ONLY"
	}
	return r, nil
}
