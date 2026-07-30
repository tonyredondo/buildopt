package neutralenvelope

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/tonyredondo/buildopt/internal/metricscatalog"
)

const (
	noHitReportRecordType = "NO_HIT_OVERHEAD_GATE_REPORT"
	noHitGateID           = "A0-G06"
	noHitNativeCommand    = "gradle-no-hit-native-v1"
	noHitWrapperCommand   = "buildopt-gradle-no-hit-wrapper-v1"
)

var noHitLimitations = []string{
	"A0 engineering gate only; this is not beta promotion or causal savings evidence.",
	"The four-pair strict sample uses nearest-rank p95, which is the maximum observed value.",
}

// NoHitInputs bind the contracts and executable bytes used by the gate.
type NoHitInputs struct {
	RunnerSpecSHA256      string `json:"runnerSpecSha256"`
	MetricsCatalogSHA256  string `json:"metricsCatalogSha256"`
	EnvelopeSHA256        string `json:"envelopeSha256"`
	LauncherSHA256        string `json:"launcherSha256"`
	ServerSHA256          string `json:"serverSha256"`
	PluginSHA256          string `json:"pluginSha256"`
	FixtureManifestSHA256 string `json:"fixtureManifestSha256"`
	HelperSHA256          string `json:"helperSha256"`
}

// NoHitWorkload describes the real Gradle no-hit and short-session strata.
type NoHitWorkload struct {
	ID                     string  `json:"id"`
	LongTask               string  `json:"longTask"`
	ShortTask              string  `json:"shortTask"`
	PairCount              int     `json:"pairCount"`
	AlternatingOrder       bool    `json:"alternatingOrder"`
	Offline                bool    `json:"offline"`
	ConfigurationCache     bool    `json:"configurationCache"`
	DaemonState            string  `json:"daemonState"`
	WarmupsPerArm          int     `json:"warmupsPerArm"`
	LongProbeDelayMs       int64   `json:"longProbeDelayMs"`
	LongSessionMinimumMs   float64 `json:"longSessionMinimumMs"`
	RequiredDeliverable    string  `json:"requiredDeliverable"`
	DeliverableSHA256      string  `json:"deliverableSha256"`
	L2Path                 string  `json:"l2Path"`
	LongRemoteMisses       int     `json:"longRemoteMisses"`
	ShortSessionPolicy     string  `json:"shortSessionPolicy"`
	ShortSessionDuration   float64 `json:"shortSessionDurationMs"`
	ShortRemoteRequests    int     `json:"shortRemoteRequests"`
	WorkspaceState         string  `json:"workspaceState"`
	OutcomeClass           string  `json:"outcomeClass"`
	MeasurementEpoch       string  `json:"measurementEpoch"`
	FirstExecutionIncluded bool    `json:"firstExecutionIncluded"`
}

// NoHitSummary is the recomputed tail summary used by A0-G06.
type NoHitSummary struct {
	PairCount                           int     `json:"pairCount"`
	NativeP95Ms                         float64 `json:"nativeP95Ms"`
	WrapperP95Ms                        float64 `json:"wrapperP95Ms"`
	ProductSynchronousOverheadP50Ms     float64 `json:"productSynchronousOverheadP50Ms"`
	ProductSynchronousOverheadP95Ms     float64 `json:"productSynchronousOverheadP95Ms"`
	ProductSynchronousOverheadP95Ratio  float64 `json:"productSynchronousOverheadP95Ratio"`
	ProductSynchronousOverheadMaximumMs float64 `json:"productSynchronousOverheadMaximumMs"`
}

// NoHitGate records the exact A0 thresholds and their result.
type NoHitGate struct {
	LongMaximumP95Ms       float64 `json:"longMaximumP95Ms"`
	LongMaximumP95Ratio    float64 `json:"longMaximumP95Ratio"`
	ShortMaximumDurationMs float64 `json:"shortMaximumDurationMs"`
	ShortAlternative       string  `json:"shortAlternative"`
	LongPassed             bool    `json:"longPassed"`
	ShortPassed            bool    `json:"shortPassed"`
	Passed                 bool    `json:"passed"`
}

// NoHitReport is the immutable A0-G06 evidence.
type NoHitReport struct {
	SchemaVersion            string         `json:"schemaVersion"`
	RecordType               string         `json:"recordType"`
	CreatedAt                string         `json:"createdAt"`
	GateID                   string         `json:"gateId"`
	MetricDefinitionVersion  string         `json:"metricDefinitionVersion"`
	MeasurementPolicyVersion string         `json:"measurementPolicyVersion"`
	MeasurementKind          string         `json:"measurementKind"`
	ObservationMethod        string         `json:"observationMethod"`
	EffectScope              string         `json:"effectScope"`
	A0GateActive             bool           `json:"a0GateActive"`
	PromotionGateActive      bool           `json:"promotionGateActive"`
	ClockSource              string         `json:"clockSource"`
	DurationUnit             string         `json:"durationUnit"`
	StartBoundary            string         `json:"startBoundary"`
	EndBoundary              string         `json:"endBoundary"`
	ExecutionEnvironment     string         `json:"executionEnvironment"`
	RunnerClassQualified     bool           `json:"runnerClassQualified"`
	Inputs                   NoHitInputs    `json:"inputs"`
	Runner                   RunnerIdentity `json:"runner"`
	Workload                 NoHitWorkload  `json:"workload"`
	Pairs                    []Pair         `json:"pairs"`
	Summary                  NoHitSummary   `json:"summary"`
	Gate                     NoHitGate      `json:"gate"`
	Limitations              []string       `json:"limitations"`
}

// BuildNoHitReport validates raw observations and evaluates the A0-G06 gate.
func BuildNoHitReport(
	observations []Observation,
	executionEnvironment string,
	runnerSpecPath string,
	metricsCatalogPath string,
	envelopePath string,
	launcherPath string,
	serverPath string,
	pluginPath string,
	fixtureManifestPath string,
	helperPath string,
	longProbeDelayMs int64,
	longRemoteMisses int,
	shortSessionDurationMs float64,
	shortRemoteRequests int,
	createdAt time.Time,
) (NoHitReport, error) {
	if executionEnvironment != "HOST_SMOKE" &&
		executionEnvironment != "STRICT_GOLDEN_CONTAINER" {
		return NoHitReport{}, errors.New("unsupported execution environment")
	}
	catalog, err := metricscatalog.Load(metricsCatalogPath)
	if err != nil {
		return NoHitReport{}, err
	}
	if err := validateOverheadMetrics(catalog); err != nil {
		return NoHitReport{}, err
	}
	spec, err := loadRunnerSpec(runnerSpecPath)
	if err != nil {
		return NoHitReport{}, err
	}
	pairs, err := pairObservations(observations)
	if err != nil {
		return NoHitReport{}, err
	}
	deliverableDigest := pairs[0].Native.DeliverableSHA256
	for _, pair := range pairs {
		if pair.Native.DeliverableSHA256 != deliverableDigest ||
			pair.Wrapper.DeliverableSHA256 != deliverableDigest {
			return NoHitReport{}, errors.New(
				"native and wrapper deliverables are not byte-identical",
			)
		}
	}

	inputPaths := []string{
		runnerSpecPath,
		metricsCatalogPath,
		envelopePath,
		launcherPath,
		serverPath,
		pluginPath,
		fixtureManifestPath,
		helperPath,
	}
	inputDigests := make([]string, len(inputPaths))
	for index, path := range inputPaths {
		inputDigests[index], err = fileSHA256(path)
		if err != nil {
			return NoHitReport{}, err
		}
	}
	summary := summarizeNoHit(pairs)
	longPassed := noHitLongPassed(
		pairs,
		summary,
		longRemoteMisses,
	)
	shortPassed := shortSessionDurationMs > 0 &&
		shortSessionDurationMs < 5000 &&
		shortRemoteRequests == 0
	qualified := executionEnvironment == "STRICT_GOLDEN_CONTAINER"

	report := NoHitReport{
		SchemaVersion:            reportSchemaVersion,
		RecordType:               noHitReportRecordType,
		CreatedAt:                createdAt.UTC().Format(time.RFC3339Nano),
		GateID:                   noHitGateID,
		MetricDefinitionVersion:  catalog.MetricDefinitionVersion,
		MeasurementPolicyVersion: catalog.PromotionPolicy.Version,
		MeasurementKind:          "PAIRED_ALTERNATING_NO_HIT_BASELINE_VS_L2",
		ObservationMethod:        "EXACT",
		EffectScope:              "PRODUCT_TOTAL",
		A0GateActive:             true,
		PromotionGateActive:      false,
		ClockSource:              "MONOTONIC",
		DurationUnit:             "ms",
		StartBoundary:            "NEUTRAL_ENVELOPE_HANDOFF",
		EndBoundary:              "EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE",
		ExecutionEnvironment:     executionEnvironment,
		RunnerClassQualified:     qualified,
		Inputs: NoHitInputs{
			RunnerSpecSHA256:      inputDigests[0],
			MetricsCatalogSHA256:  inputDigests[1],
			EnvelopeSHA256:        inputDigests[2],
			LauncherSHA256:        inputDigests[3],
			ServerSHA256:          inputDigests[4],
			PluginSHA256:          inputDigests[5],
			FixtureManifestSHA256: inputDigests[6],
			HelperSHA256:          inputDigests[7],
		},
		Runner: RunnerIdentity{
			ID:                    spec.RunnerClass.ID,
			OperatingSystem:       spec.RunnerClass.OperatingSystem,
			Architecture:          spec.RunnerClass.Architecture,
			CPUCount:              spec.RunnerClass.CPUCount,
			MemoryBytes:           spec.RunnerClass.MemoryBytes,
			ContainerIndexDigest:  spec.Container.IndexDigest,
			ContainerPlatformHash: spec.Container.PlatformDigest,
			JDKVersion:            spec.JDK.Version,
			GradleVersion:         spec.Gradle.Version,
		},
		Workload: NoHitWorkload{
			ID:                     "gradle-tier-one-no-hit-v1",
			LongTask:               "noHitLongProbe",
			ShortTask:              "noHitShortProbe",
			PairCount:              len(pairs),
			AlternatingOrder:       true,
			Offline:                false,
			ConfigurationCache:     true,
			DaemonState:            "WARM_SHARED_DAEMON",
			WarmupsPerArm:          1,
			LongProbeDelayMs:       longProbeDelayMs,
			LongSessionMinimumMs:   5000,
			RequiredDeliverable:    "libs/no-hit-overhead.jar",
			DeliverableSHA256:      deliverableDigest,
			L2Path:                 "AUTHENTICATED_LOCAL_GATEWAY_TO_FORCED_404",
			LongRemoteMisses:       longRemoteMisses,
			ShortSessionPolicy:     "L2_OMITTED_PRE_OUTCOME",
			ShortSessionDuration:   shortSessionDurationMs,
			ShortRemoteRequests:    shortRemoteRequests,
			WorkspaceState:         "OUTPUT_AND_L1_REMOVED_BEFORE_EACH_OBSERVATION",
			OutcomeClass:           "SUCCESS",
			MeasurementEpoch:       "a0-g06-v1",
			FirstExecutionIncluded: true,
		},
		Pairs:   pairs,
		Summary: summary,
		Gate: NoHitGate{
			LongMaximumP95Ms:       500,
			LongMaximumP95Ratio:    0.02,
			ShortMaximumDurationMs: 5000,
			ShortAlternative:       "L2_OMITTED",
			LongPassed:             longPassed,
			ShortPassed:            shortPassed,
			Passed:                 qualified && longPassed && shortPassed,
		},
		Limitations: slices.Clone(noHitLimitations),
	}
	if err := report.Validate(); err != nil {
		return NoHitReport{}, err
	}
	return report, nil
}

// Validate checks report invariants and recomputes all A0-G06 decisions.
func (report NoHitReport) Validate() error {
	if report.SchemaVersion != reportSchemaVersion ||
		report.RecordType != noHitReportRecordType ||
		report.GateID != noHitGateID ||
		report.MetricDefinitionVersion != metricscatalog.Version ||
		report.MeasurementPolicyVersion != "beta-measurement-v1" ||
		report.MeasurementKind !=
			"PAIRED_ALTERNATING_NO_HIT_BASELINE_VS_L2" ||
		report.ObservationMethod != "EXACT" ||
		report.EffectScope != "PRODUCT_TOTAL" ||
		!report.A0GateActive ||
		report.PromotionGateActive ||
		report.ClockSource != "MONOTONIC" ||
		report.DurationUnit != "ms" ||
		report.StartBoundary != "NEUTRAL_ENVELOPE_HANDOFF" ||
		report.EndBoundary !=
			"EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE" {
		return errors.New("unsupported no-hit overhead report contract")
	}
	if _, err := parseCanonicalTime(report.CreatedAt); err != nil {
		return errors.New("invalid no-hit report createdAt")
	}
	if report.ExecutionEnvironment != "HOST_SMOKE" &&
		report.ExecutionEnvironment != "STRICT_GOLDEN_CONTAINER" {
		return errors.New("invalid no-hit execution environment")
	}
	qualified := report.ExecutionEnvironment == "STRICT_GOLDEN_CONTAINER"
	if report.RunnerClassQualified != qualified {
		return errors.New("no-hit runner qualification is inconsistent")
	}
	for _, digest := range []string{
		report.Inputs.RunnerSpecSHA256,
		report.Inputs.MetricsCatalogSHA256,
		report.Inputs.EnvelopeSHA256,
		report.Inputs.LauncherSHA256,
		report.Inputs.ServerSHA256,
		report.Inputs.PluginSHA256,
		report.Inputs.FixtureManifestSHA256,
		report.Inputs.HelperSHA256,
	} {
		if !validSHA256(digest) {
			return errors.New("no-hit report contains an invalid input digest")
		}
	}
	if report.Runner.ID == "" ||
		report.Runner.OperatingSystem != "linux" ||
		report.Runner.Architecture != "amd64" ||
		report.Runner.CPUCount != 4 ||
		report.Runner.MemoryBytes != 17179869184 ||
		!validSHA256(report.Runner.ContainerIndexDigest) ||
		!validSHA256(report.Runner.ContainerPlatformHash) ||
		report.Runner.JDKVersion == "" ||
		report.Runner.GradleVersion != "9.6.1" {
		return errors.New("no-hit runner does not match the golden class")
	}
	if report.Workload.ID != "gradle-tier-one-no-hit-v1" ||
		report.Workload.LongTask != "noHitLongProbe" ||
		report.Workload.ShortTask != "noHitShortProbe" ||
		report.Workload.PairCount != len(report.Pairs) ||
		!report.Workload.AlternatingOrder ||
		report.Workload.Offline ||
		!report.Workload.ConfigurationCache ||
		report.Workload.DaemonState != "WARM_SHARED_DAEMON" ||
		report.Workload.WarmupsPerArm != 1 ||
		report.Workload.LongProbeDelayMs < 1 ||
		report.Workload.LongSessionMinimumMs != 5000 ||
		report.Workload.RequiredDeliverable !=
			"libs/no-hit-overhead.jar" ||
		!validSHA256(report.Workload.DeliverableSHA256) ||
		report.Workload.L2Path !=
			"AUTHENTICATED_LOCAL_GATEWAY_TO_FORCED_404" ||
		report.Workload.LongRemoteMisses < len(report.Pairs) ||
		report.Workload.ShortSessionPolicy !=
			"L2_OMITTED_PRE_OUTCOME" ||
		report.Workload.ShortSessionDuration <= 0 ||
		report.Workload.ShortSessionDuration >= 5000 ||
		report.Workload.ShortRemoteRequests != 0 ||
		report.Workload.WorkspaceState !=
			"OUTPUT_AND_L1_REMOVED_BEFORE_EACH_OBSERVATION" ||
		report.Workload.OutcomeClass != "SUCCESS" ||
		report.Workload.MeasurementEpoch != "a0-g06-v1" ||
		!report.Workload.FirstExecutionIncluded {
		return errors.New("invalid no-hit workload")
	}
	expectedPairs := 2
	if qualified {
		expectedPairs = 4
	}
	if len(report.Pairs) != expectedPairs {
		return fmt.Errorf(
			"no-hit report has %d pairs, want %d",
			len(report.Pairs),
			expectedPairs,
		)
	}
	if qualified && report.Workload.LongProbeDelayMs != 25000 {
		return errors.New("strict no-hit probe delay does not match the contract")
	}
	for index, pair := range report.Pairs {
		if err := pair.validateForCommands(
			index+1,
			report.Workload.DeliverableSHA256,
			noHitNativeCommand,
			noHitWrapperCommand,
		); err != nil {
			return err
		}
	}
	expectedSummary := summarizeNoHit(report.Pairs)
	if !noHitSummariesEqual(report.Summary, expectedSummary) {
		return errors.New("no-hit summary does not match raw pairs")
	}
	longPassed := noHitLongPassed(
		report.Pairs,
		expectedSummary,
		report.Workload.LongRemoteMisses,
	)
	shortPassed := report.Workload.ShortSessionDuration > 0 &&
		report.Workload.ShortSessionDuration < 5000 &&
		report.Workload.ShortRemoteRequests == 0
	if report.Gate.LongMaximumP95Ms != 500 ||
		report.Gate.LongMaximumP95Ratio != 0.02 ||
		report.Gate.ShortMaximumDurationMs != 5000 ||
		report.Gate.ShortAlternative != "L2_OMITTED" ||
		report.Gate.LongPassed != longPassed ||
		report.Gate.ShortPassed != shortPassed ||
		report.Gate.Passed != (qualified && longPassed && shortPassed) {
		return errors.New("no-hit gate result does not reconcile")
	}
	if !slices.Equal(report.Limitations, noHitLimitations) {
		return errors.New("no-hit limitations are incomplete")
	}
	return nil
}

// LoadNoHitReport decodes one strict A0-G06 report.
func LoadNoHitReport(path string) (NoHitReport, error) {
	var report NoHitReport
	if err := loadJSON(path, &report); err != nil {
		return NoHitReport{}, err
	}
	if err := report.Validate(); err != nil {
		return NoHitReport{}, err
	}
	return report, nil
}

func summarizeNoHit(pairs []Pair) NoHitSummary {
	base := summarize(pairs)
	ratios := make([]float64, len(pairs))
	for index, pair := range pairs {
		ratios[index] = pair.ProductSynchronousOverheadRatio
	}
	slices.Sort(ratios)
	return NoHitSummary{
		PairCount:                           len(pairs),
		NativeP95Ms:                         nearestRankDurations(pairs, true),
		WrapperP95Ms:                        nearestRankDurations(pairs, false),
		ProductSynchronousOverheadP50Ms:     base.ProductOverheadP50Ms,
		ProductSynchronousOverheadP95Ms:     base.ProductOverheadP95Ms,
		ProductSynchronousOverheadP95Ratio:  nearestRank(ratios, 0.95),
		ProductSynchronousOverheadMaximumMs: base.ProductOverheadMaximumMs,
	}
}

func nearestRankDurations(pairs []Pair, native bool) float64 {
	values := make([]float64, len(pairs))
	for index, pair := range pairs {
		if native {
			values[index] = pair.Native.DurationMs
		} else {
			values[index] = pair.Wrapper.DurationMs
		}
	}
	slices.Sort(values)
	return nearestRank(values, 0.95)
}

func noHitLongPassed(
	pairs []Pair,
	summary NoHitSummary,
	longRemoteMisses int,
) bool {
	if len(pairs) == 0 || longRemoteMisses < len(pairs) ||
		summary.ProductSynchronousOverheadP95Ms > 500 ||
		summary.ProductSynchronousOverheadP95Ratio > 0.02 {
		return false
	}
	for _, pair := range pairs {
		if pair.Native.DurationMs < 5000 ||
			pair.Wrapper.DurationMs < 5000 {
			return false
		}
	}
	return true
}

func noHitSummariesEqual(actual, expected NoHitSummary) bool {
	return actual.PairCount == expected.PairCount &&
		closeFloat(actual.NativeP95Ms, expected.NativeP95Ms) &&
		closeFloat(actual.WrapperP95Ms, expected.WrapperP95Ms) &&
		closeFloat(
			actual.ProductSynchronousOverheadP50Ms,
			expected.ProductSynchronousOverheadP50Ms,
		) &&
		closeFloat(
			actual.ProductSynchronousOverheadP95Ms,
			expected.ProductSynchronousOverheadP95Ms,
		) &&
		closeFloat(
			actual.ProductSynchronousOverheadP95Ratio,
			expected.ProductSynchronousOverheadP95Ratio,
		) &&
		closeFloat(
			actual.ProductSynchronousOverheadMaximumMs,
			expected.ProductSynchronousOverheadMaximumMs,
		)
}
