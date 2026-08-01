package buildimpact

import (
	"errors"
	"math"
	"sort"
	"time"
)

const (
	PromotionReportSchemaVersion  = "buildopt.build-impact/promotion-report/v1"
	PromotionInconclusive         = "INCONCLUSIVE"
	PromotionQualified            = "QUALIFIED"
	PromotionSuspended            = "SUSPENDED"
	MinimumShadowDays             = 30
	MinimumEligibleDecisions      = 3000
	MinimumControlsPerChangeClass = 100
	MinimumValidationCoverage     = 0.99
	MaximumAggregateFalseNegative = 0.001
	MaximumStratumFalseNegative   = 0.03
	oneSidedAlpha                 = 0.05
	maximumPromotionResults       = 1_000_000
)

type PromotionInput struct {
	RepositoryID           string
	PipelineClass          string
	ManifestDigest         string
	GraphDigest            string
	AdapterVersion         string
	MandatoryChangeClasses []string
	Results                []ValidationResult
}

type PromotionReport struct {
	SchemaVersion       string          `json:"schemaVersion"`
	RepositoryID        string          `json:"repositoryId"`
	PipelineClass       string          `json:"pipelineClass"`
	ManifestDigest      string          `json:"manifestDigest"`
	GraphDigest         string          `json:"graphDigest"`
	AdapterVersion      string          `json:"adapterVersion"`
	State               string          `json:"state"`
	Reason              string          `json:"reason"`
	ShadowDays          float64         `json:"shadowDays"`
	EligibleDecisions   int             `json:"eligibleDecisions"`
	ValidatedDecisions  int             `json:"validatedDecisions"`
	ValidationCoverage  float64         `json:"validationCoverage"`
	FullControls        int             `json:"fullControls"`
	FalseNegatives      int             `json:"falseNegatives"`
	AggregateUpper95    float64         `json:"aggregateUpper95"`
	Strata              []StratumReport `json:"strata"`
	ExcludedPreReset    int             `json:"excludedPreReset"`
	ResetApplied        bool            `json:"resetApplied"`
	SelectionAuthorized bool            `json:"selectionAuthorized"`
}

type StratumReport struct {
	ChangeClass    string  `json:"changeClass"`
	FullControls   int     `json:"fullControls"`
	FalseNegatives int     `json:"falseNegatives"`
	Upper95        float64 `json:"upper95"`
}

// EvaluatePromotion applies BIA-002 to current-binding validation results.
// Qualification is an input to C3-005 and never directly enables selection.
func EvaluatePromotion(input PromotionInput) PromotionReport {
	report := PromotionReport{
		SchemaVersion:       PromotionReportSchemaVersion,
		RepositoryID:        input.RepositoryID,
		PipelineClass:       input.PipelineClass,
		ManifestDigest:      input.ManifestDigest,
		GraphDigest:         input.GraphDigest,
		AdapterVersion:      input.AdapterVersion,
		State:               PromotionInconclusive,
		Reason:              "INPUT_INVALID",
		SelectionAuthorized: false,
	}
	if err := validatePromotionInput(input); err != nil {
		return report
	}
	strata := map[string]*StratumReport{}
	for _, changeClass := range input.MandatoryChangeClasses {
		strata[changeClass] = &StratumReport{ChangeClass: changeClass}
	}
	seenObservations := map[string]bool{}
	var earliest time.Time
	var latest time.Time
	for _, result := range input.Results {
		if seenObservations[result.ObservationID] {
			report.Reason = "DUPLICATE_OBSERVATION"
			return report
		}
		seenObservations[result.ObservationID] = true
		if result.RepositoryID != input.RepositoryID || result.PipelineClass != input.PipelineClass {
			report.Reason = "RESULT_SCOPE_MISMATCH"
			return report
		}
		if result.ManifestDigest != input.ManifestDigest || result.GraphDigest != input.GraphDigest || result.AdapterVersion != input.AdapterVersion {
			report.ExcludedPreReset++
			continue
		}
		if !validValidationResult(result) {
			report.Reason = "RESULT_INVALID"
			return report
		}
		if !result.EligibleDecision {
			continue
		}
		observedAt, _ := time.Parse(time.RFC3339, result.ObservedAt)
		if earliest.IsZero() || observedAt.Before(earliest) {
			earliest = observedAt
		}
		if latest.IsZero() || observedAt.After(latest) {
			latest = observedAt
		}
		report.EligibleDecisions++
		if result.ValidationComplete {
			report.ValidatedDecisions++
		}
		if result.FullControl && result.ValidationComplete {
			report.FullControls++
			if stratum, ok := strata[result.ChangeClass]; ok {
				stratum.FullControls++
			}
		}
		if result.FalseNegative {
			report.FalseNegatives++
			if stratum, ok := strata[result.ChangeClass]; ok {
				stratum.FalseNegatives++
			}
		}
	}
	report.ResetApplied = report.ExcludedPreReset > 0
	if !earliest.IsZero() {
		report.ShadowDays = latest.Sub(earliest).Hours() / 24
	}
	if report.EligibleDecisions > 0 {
		report.ValidationCoverage = float64(report.ValidatedDecisions) / float64(report.EligibleDecisions)
	}
	report.AggregateUpper95 = conservativeFalseNegativeUpper(report.FalseNegatives, report.ValidatedDecisions)
	for _, changeClass := range input.MandatoryChangeClasses {
		stratum := strata[changeClass]
		stratum.Upper95 = conservativeFalseNegativeUpper(stratum.FalseNegatives, stratum.FullControls)
		report.Strata = append(report.Strata, *stratum)
	}
	sort.Slice(report.Strata, func(left, right int) bool {
		return report.Strata[left].ChangeClass < report.Strata[right].ChangeClass
	})
	if report.FalseNegatives > 0 {
		report.State = PromotionSuspended
		report.Reason = "FALSE_NEGATIVE_OBSERVED"
		return report
	}
	if report.ShadowDays < MinimumShadowDays {
		report.Reason = "SHADOW_WINDOW_INSUFFICIENT"
		return report
	}
	if report.EligibleDecisions < MinimumEligibleDecisions {
		report.Reason = "ELIGIBLE_DECISIONS_INSUFFICIENT"
		return report
	}
	if report.ValidationCoverage < MinimumValidationCoverage {
		report.Reason = "VALIDATION_COVERAGE_INSUFFICIENT"
		return report
	}
	for _, stratum := range report.Strata {
		if stratum.FullControls < MinimumControlsPerChangeClass {
			report.Reason = "STRATUM_CONTROLS_INSUFFICIENT"
			return report
		}
		if stratum.Upper95 > MaximumStratumFalseNegative {
			report.Reason = "STRATUM_UPPER_BOUND_EXCEEDED"
			return report
		}
	}
	if report.AggregateUpper95 > MaximumAggregateFalseNegative {
		report.Reason = "AGGREGATE_UPPER_BOUND_EXCEEDED"
		return report
	}
	report.State = PromotionQualified
	report.Reason = "BIA_002_SATISFIED"
	return report
}

func validatePromotionInput(input PromotionInput) error {
	if !repositoryPattern.MatchString(input.RepositoryID) || !pipelinePattern.MatchString(input.PipelineClass) || !sha256Pattern.MatchString(input.ManifestDigest) || !sha256Pattern.MatchString(input.GraphDigest) || !idPattern.MatchString(input.AdapterVersion) {
		return errors.New("promotion input binding is invalid")
	}
	if len(input.MandatoryChangeClasses) == 0 || len(input.MandatoryChangeClasses) > 32 || !uniqueStrings(input.MandatoryChangeClasses) || len(input.Results) > maximumPromotionResults {
		return errors.New("promotion input collections are invalid")
	}
	for _, changeClass := range input.MandatoryChangeClasses {
		if !idPattern.MatchString(changeClass) {
			return errors.New("mandatory change class is invalid")
		}
	}
	return nil
}

func validValidationResult(result ValidationResult) bool {
	if result.SchemaVersion != ValidationResultSchemaVersion || !observationIDPattern.MatchString(result.ObservationID) || !revisionPattern.MatchString(result.Revision) || !idPattern.MatchString(result.ChangeClass) || result.Reason == "" || result.SelectionAuthorized {
		return false
	}
	if result.Mode != ValidationShadow && result.Mode != ValidationPairedControl {
		return false
	}
	if result.AlternativeID != "" && !idPattern.MatchString(result.AlternativeID) {
		return false
	}
	if !result.EligibleDecision && (result.AlternativeID != "" || result.ValidationComplete || result.FullControl || result.FalseNegative) {
		return false
	}
	if result.EligibleDecision && result.AlternativeID == "" {
		return false
	}
	if result.FullControl && result.Mode != ValidationPairedControl {
		return false
	}
	observedAt, err := time.Parse(time.RFC3339, result.ObservedAt)
	if err != nil || observedAt.Format(time.RFC3339) != result.ObservedAt || observedAt.Location() != time.UTC {
		return false
	}
	switch result.Outcome {
	case ValidationShadowPassed:
		return result.Mode == ValidationShadow && result.EligibleDecision && result.ValidationComplete && !result.FullControl && !result.FalseNegative
	case ValidationControlPassed:
		return result.Mode == ValidationPairedControl && result.EligibleDecision && result.ValidationComplete && result.FullControl && !result.FalseNegative
	case ValidationFalseNegative:
		return result.Mode == ValidationPairedControl && result.EligibleDecision && result.ValidationComplete && result.FullControl && result.FalseNegative
	case ValidationInconclusive:
		return !result.ValidationComplete && !result.FalseNegative
	default:
		return false
	}
}

// conservativeFalseNegativeUpper is the exact one-sided 95% upper bound for
// zero observed failures. Any observed failure suspends before the bound can
// qualify, so returning 1 is the conservative bound for that terminal case.
func conservativeFalseNegativeUpper(failures, observations int) float64 {
	if observations <= 0 || failures > 0 {
		return 1
	}
	return 1 - math.Pow(oneSidedAlpha, 1/float64(observations))
}
