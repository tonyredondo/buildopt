package metricscatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

const (
	// Version is the semantic version identifier embedded in BUILD_SESSION
	// records and the normative METRICS-001 catalog.
	Version = "build-impact-v1"

	maxCatalogBytes = 1 << 20
)

var requiredMetricIDs = []string{
	"buildCriticalPathMs",
	"cacheHitRate",
	"ciQueueMs",
	"ciQueueP95DeltaMs",
	"ciQueueP99DeltaMs",
	"configurationMs",
	"customerVisibleBuildMs",
	"customerVisibleBuildP95DeltaMs",
	"customerVisibleBuildP99DeltaMs",
	"customerVisibleFeedbackMs",
	"customerVisibleFeedbackP95DeltaMs",
	"customerVisibleFeedbackP99DeltaMs",
	"estimatedNetBuildTimeSavedMs",
	"executionOrRestoreMs",
	"finalizationMs",
	"gatewayStartupMs",
	"gradleProcessMs",
	"gradleProcessUnionMs",
	"gradleStartupAndInitializationMs",
	"incrementalActionOverheadMs",
	"launcherAndPolicyMs",
	"measurementCoverageRatio",
	"netInfrastructureValue",
	"observedBuildTimeReductionRatio",
	"observedNetBuildTimeSavedMs",
	"productAttributableFailureRate",
	"productSynchronousOverheadMs",
	"productSynchronousOverheadRatio",
	"requiredDeliverableDivergenceRate",
	"runnerOccupiedMs",
	"testOwnedTaskExecutionMs",
	"timeToFirstBuildFailureMs",
	"timeWeightedHitRate",
	"unattributedMs",
	"usefulHitRate",
}

var requiredDimensions = []string{
	"actionId",
	"cacheState",
	"cacheTier",
	"changeClass",
	"daemonState",
	"effectScope",
	"environment",
	"measurementEpoch",
	"outcome",
	"pipelineClass",
	"repositoryId",
	"runnerClass",
	"runnerPool",
	"trustDomain",
	"workUnitsFingerprint",
	"workspaceState",
}

var allowedDimensions = func() map[string]struct{} {
	result := make(map[string]struct{}, len(requiredDimensions))
	for _, dimension := range requiredDimensions {
		result[dimension] = struct{}{}
	}
	return result
}()

// Catalog is the machine-readable METRICS-001 definition set.
type Catalog struct {
	SchemaVersion           string          `json:"schemaVersion"`
	MetricDefinitionVersion string          `json:"metricDefinitionVersion"`
	DecisionIDs             []string        `json:"decisionIds"`
	MeasurementStates       []string        `json:"measurementStates"`
	ObservationMethods      []string        `json:"observationMethods"`
	OutcomeClasses          []string        `json:"outcomeClasses"`
	Dimensions              []string        `json:"dimensions"`
	ComparisonRules         ComparisonRules `json:"comparisonRules"`
	PromotionPolicy         PromotionPolicy `json:"promotionPolicy"`
	Metrics                 []Metric        `json:"metrics"`
}

// ComparisonRules fixes cross-metric aggregation and sign semantics.
type ComparisonRules struct {
	SavedAndReductionSign string   `json:"savedAndReductionSign"`
	DeltaSign             string   `json:"deltaSign"`
	OutcomeIsolation      bool     `json:"outcomeIsolation"`
	RequiredStrata        []string `json:"requiredStrata"`
	PairedRequirements    []string `json:"pairedRequirements"`
	InvalidComparison     string   `json:"invalidComparison"`
}

// PromotionPolicy materializes the MEASURE-001 private-beta defaults.
type PromotionPolicy struct {
	Version                      string       `json:"version"`
	DirectReversible             SampleGate   `json:"directReversible"`
	ProofGated                   SampleGate   `json:"proofGated"`
	MinimumBenefit               Threshold    `json:"minimumBenefit"`
	P95Regression                Threshold    `json:"p95Regression"`
	P99Regression                Threshold    `json:"p99Regression"`
	P99MinimumObservationsPerArm int          `json:"p99MinimumObservationsPerArm"`
	PostPromotionControlRatio    float64      `json:"postPromotionControlRatio"`
	MaximumAdditionalComputeDays int          `json:"maximumAdditionalComputeDays"`
	Correctness                  []ZeroTarget `json:"correctness"`
	InsufficientEvidenceState    string       `json:"insufficientEvidenceState"`
}

// SampleGate defines the minimum time and comparable observations per arm.
type SampleGate struct {
	MinimumWindowDays         int `json:"minimumWindowDays"`
	MinimumObservationsPerArm int `json:"minimumObservationsPerArm"`
}

// Threshold combines the absolute and relative comparison bounds.
type Threshold struct {
	Bound                string  `json:"bound"`
	Combination          string  `json:"combination"`
	AbsoluteMilliseconds int64   `json:"absoluteMilliseconds"`
	RelativeToControl    float64 `json:"relativeToControl"`
}

// ZeroTarget is a correctness metric that speed cannot offset.
type ZeroTarget struct {
	MetricID string `json:"metricId"`
	Maximum  int64  `json:"maximum"`
}

// Metric contains every governance field required by RFC section 22.9.
type Metric struct {
	ID             string   `json:"id"`
	Owner          string   `json:"owner"`
	Purpose        string   `json:"purpose"`
	Lifecycle      string   `json:"lifecycle"`
	Formula        string   `json:"formula"`
	Unit           string   `json:"unit"`
	Grain          string   `json:"grain"`
	Population     string   `json:"population"`
	Denominator    string   `json:"denominator"`
	Sources        []string `json:"sources"`
	StartBoundary  string   `json:"startBoundary"`
	EndBoundary    string   `json:"endBoundary"`
	Dimensions     []string `json:"dimensions"`
	NullPolicy     string   `json:"nullPolicy"`
	QualityRules   []string `json:"qualityRules"`
	Retention      string   `json:"retention"`
	Caveats        []string `json:"caveats"`
	SignConvention string   `json:"signConvention"`
	AllowedMethods []string `json:"allowedMethods"`
}

// Load reads and validates one strict catalog document.
func Load(path string) (Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return Catalog{}, fmt.Errorf("open metrics catalog: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, maxCatalogBytes+1))
	decoder.DisallowUnknownFields()
	var catalog Catalog
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode metrics catalog: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Catalog{}, errors.New(
			"metrics catalog must contain exactly one JSON value",
		)
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

// Validate enforces the stable catalog shape and METRICS-001 semantics.
func (catalog Catalog) Validate() error {
	if catalog.SchemaVersion != "1.0" {
		return errors.New("metrics catalog schemaVersion must be 1.0")
	}
	if catalog.MetricDefinitionVersion != Version {
		return fmt.Errorf(
			"metricDefinitionVersion must be %s",
			Version,
		)
	}
	if !slices.Equal(
		catalog.DecisionIDs,
		[]string{"METRICS-001", "MEASURE-001"},
	) {
		return errors.New("metrics catalog decisionIds are incomplete")
	}
	if !slices.Equal(
		catalog.MeasurementStates,
		[]string{"COMPLETE", "PARTIAL", "UNAVAILABLE"},
	) {
		return errors.New("measurement states do not match BUILD_SESSION v1")
	}
	if !slices.Equal(
		catalog.ObservationMethods,
		[]string{"EXACT", "APPROXIMATED", "UNAVAILABLE"},
	) {
		return errors.New("observation methods do not match BUILD_SESSION v1")
	}
	if !slices.Equal(
		catalog.OutcomeClasses,
		[]string{"SUCCESS", "BUILD_FAILURE", "INFRA_FAILURE", "CANCELLED"},
	) {
		return errors.New("outcome classes do not match BUILD_SESSION v1")
	}
	if err := validateDimensions("catalog", catalog.Dimensions); err != nil {
		return err
	}
	if !slices.Equal(catalog.Dimensions, requiredDimensions) {
		return errors.New("catalog dimensions do not match the bounded v1 set")
	}
	if err := catalog.ComparisonRules.validate(); err != nil {
		return err
	}
	if err := catalog.PromotionPolicy.validate(); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(catalog.Metrics))
	for index, metric := range catalog.Metrics {
		if err := metric.validate(); err != nil {
			return fmt.Errorf("metric %d: %w", index, err)
		}
		if _, exists := seen[metric.ID]; exists {
			return fmt.Errorf("duplicate metric ID %q", metric.ID)
		}
		seen[metric.ID] = struct{}{}
	}
	actualIDs := make([]string, 0, len(seen))
	for id := range seen {
		actualIDs = append(actualIDs, id)
	}
	slices.Sort(actualIDs)
	if !slices.Equal(actualIDs, requiredMetricIDs) {
		return errors.New("metrics catalog does not contain the required v1 set")
	}
	for _, target := range catalog.PromotionPolicy.Correctness {
		if _, exists := seen[target.MetricID]; !exists {
			return fmt.Errorf(
				"correctness target references unknown metric %q",
				target.MetricID,
			)
		}
	}
	return nil
}

func (rules ComparisonRules) validate() error {
	if rules.SavedAndReductionSign != "POSITIVE_IS_IMPROVEMENT" ||
		rules.DeltaSign != "NEGATIVE_IS_IMPROVEMENT" ||
		!rules.OutcomeIsolation ||
		rules.InvalidComparison != "INCONCLUSIVE" ||
		!slices.Equal(
			rules.RequiredStrata,
			[]string{
				"environment",
				"pipelineClass",
				"runnerClass",
				"outcome",
				"workUnitsFingerprint",
				"measurementEpoch",
			},
		) ||
		len(rules.PairedRequirements) != 5 {
		return errors.New("comparison rules do not implement METRICS-001")
	}
	for _, requirement := range rules.PairedRequirements {
		if strings.TrimSpace(requirement) == "" {
			return errors.New("comparison rules contain an empty requirement")
		}
	}
	return validateDimensions("comparison rules", rules.RequiredStrata)
}

func (policy PromotionPolicy) validate() error {
	if policy.Version != "beta-measurement-v1" ||
		policy.DirectReversible.MinimumWindowDays != 7 ||
		policy.DirectReversible.MinimumObservationsPerArm != 100 ||
		policy.ProofGated.MinimumWindowDays != 14 ||
		policy.ProofGated.MinimumObservationsPerArm != 200 ||
		policy.MinimumBenefit.Bound != "ONE_SIDED_95_LOWER" ||
		policy.MinimumBenefit.Combination != "MAX" ||
		policy.MinimumBenefit.AbsoluteMilliseconds != 500 ||
		policy.MinimumBenefit.RelativeToControl != 0.02 ||
		policy.P95Regression.Bound != "ONE_SIDED_95_UPPER" ||
		policy.P95Regression.Combination != "MAX" ||
		policy.P95Regression.AbsoluteMilliseconds != 500 ||
		policy.P95Regression.RelativeToControl != 0.03 ||
		policy.P99Regression.Bound != "OBSERVED_UPPER" ||
		policy.P99Regression.Combination != "MAX" ||
		policy.P99Regression.AbsoluteMilliseconds != 1000 ||
		policy.P99Regression.RelativeToControl != 0.05 ||
		policy.P99MinimumObservationsPerArm != 1000 ||
		policy.PostPromotionControlRatio != 0.05 ||
		policy.MaximumAdditionalComputeDays != 28 ||
		policy.InsufficientEvidenceState != "INCONCLUSIVE" ||
		len(policy.Correctness) != 2 ||
		policy.Correctness[0].MetricID !=
			"requiredDeliverableDivergenceRate" ||
		policy.Correctness[1].MetricID !=
			"productAttributableFailureRate" {
		return errors.New("promotion policy does not implement MEASURE-001")
	}
	for _, target := range policy.Correctness {
		if target.MetricID == "" || target.Maximum != 0 {
			return errors.New("correctness promotion targets must be zero")
		}
	}
	return nil
}

func (metric Metric) validate() error {
	requiredText := map[string]string{
		"id":             metric.ID,
		"owner":          metric.Owner,
		"purpose":        metric.Purpose,
		"lifecycle":      metric.Lifecycle,
		"formula":        metric.Formula,
		"unit":           metric.Unit,
		"grain":          metric.Grain,
		"population":     metric.Population,
		"denominator":    metric.Denominator,
		"startBoundary":  metric.StartBoundary,
		"endBoundary":    metric.EndBoundary,
		"nullPolicy":     metric.NullPolicy,
		"retention":      metric.Retention,
		"signConvention": metric.SignConvention,
	}
	for field, value := range requiredText {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is empty", field)
		}
	}
	if len(metric.Sources) == 0 ||
		len(metric.QualityRules) == 0 ||
		len(metric.Caveats) == 0 ||
		len(metric.AllowedMethods) == 0 {
		return fmt.Errorf("metric %q has incomplete governance fields", metric.ID)
	}
	if metric.NullPolicy != "UNAVAILABLE_WITH_REASON" {
		return fmt.Errorf("metric %q has an unsafe null policy", metric.ID)
	}
	if !slices.Contains(
		[]string{
			"BUILD_OPTIMIZATION",
			"CACHE_GATEWAY",
			"CI_ADAPTER",
			"ECONOMIC_ADAPTER",
			"EXPERIMENT_SERVICE",
			"GRADLE_ADAPTER",
			"TEST_OPTIMIZATION",
			"VALIDATION_SERVICE",
		},
		metric.Owner,
	) {
		return fmt.Errorf("metric %q has unsupported owner", metric.ID)
	}
	if !slices.Contains(
		[]string{
			"BUILD_SESSION",
			"DIAGNOSTIC_AGGREGATE",
			"EXPERIMENT_RESULT",
		},
		metric.Lifecycle,
	) {
		return fmt.Errorf("metric %q has unsupported lifecycle", metric.ID)
	}
	if !slices.Contains(
		[]string{"currency", "ms", "ratio"},
		metric.Unit,
	) {
		return fmt.Errorf("metric %q has unsupported unit", metric.ID)
	}
	if !slices.Contains(
		[]string{
			"BUILD_SESSION",
			"CACHE_TIER_WINDOW",
			"EXPERIMENT_STRATUM",
			"GRADLE_INVOCATION",
			"METRIC_EXPERIMENT_STRATUM",
			"PAIRED_EXECUTION",
			"RUNNER_POOL_EXPERIMENT_STRATUM",
		},
		metric.Grain,
	) {
		return fmt.Errorf("metric %q has unsupported grain", metric.ID)
	}
	if !slices.Contains(
		[]string{
			"BUILD_SESSION_RECORD",
			"EXPERIMENT_RESULT_RECORD",
			"POLICY_DEFINED_AGGREGATE",
		},
		metric.Retention,
	) {
		return fmt.Errorf("metric %q has unsupported retention", metric.ID)
	}
	if !slices.Contains(
		[]string{
			"CONTEXT_ONLY",
			"NEGATIVE_IS_IMPROVEMENT",
			"NON_NEGATIVE",
			"POSITIVE_IS_IMPROVEMENT",
			"POSITIVE_IS_REGRESSION",
			"ZERO_IS_TARGET",
		},
		metric.SignConvention,
	) {
		return fmt.Errorf("metric %q has unsupported sign", metric.ID)
	}
	if err := validateDimensions(metric.ID, metric.Dimensions); err != nil {
		return err
	}
	seenMethods := make(map[string]struct{}, len(metric.AllowedMethods))
	for _, method := range metric.AllowedMethods {
		if !slices.Contains(
			[]string{"EXACT", "APPROXIMATED", "UNAVAILABLE"},
			method,
		) {
			return fmt.Errorf(
				"metric %q has unsupported method %q",
				metric.ID,
				method,
			)
		}
		if _, duplicate := seenMethods[method]; duplicate {
			return fmt.Errorf("metric %q repeats method %q", metric.ID, method)
		}
		seenMethods[method] = struct{}{}
	}
	if _, allowsUnavailable := seenMethods["UNAVAILABLE"]; !allowsUnavailable {
		return fmt.Errorf("metric %q cannot represent unavailable data", metric.ID)
	}
	if (strings.Contains(metric.ID, "Saved") ||
		strings.Contains(metric.ID, "Reduction")) &&
		metric.SignConvention != "POSITIVE_IS_IMPROVEMENT" {
		return fmt.Errorf(
			"metric %q must use the saved/reduction sign",
			metric.ID,
		)
	}
	if strings.Contains(metric.ID, "Delta") &&
		metric.SignConvention != "NEGATIVE_IS_IMPROVEMENT" {
		return fmt.Errorf("metric %q must use the delta sign", metric.ID)
	}
	return nil
}

func validateDimensions(context string, dimensions []string) error {
	if len(dimensions) == 0 {
		return fmt.Errorf("%s dimensions are empty", context)
	}
	seen := make(map[string]struct{}, len(dimensions))
	for _, dimension := range dimensions {
		if _, allowed := allowedDimensions[dimension]; !allowed {
			return fmt.Errorf(
				"%s uses unsupported dimension %q",
				context,
				dimension,
			)
		}
		if _, duplicate := seen[dimension]; duplicate {
			return fmt.Errorf(
				"%s repeats dimension %q",
				context,
				dimension,
			)
		}
		seen[dimension] = struct{}{}
	}
	return nil
}
