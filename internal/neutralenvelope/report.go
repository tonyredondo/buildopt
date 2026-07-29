package neutralenvelope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/metricscatalog"
)

const (
	reportSchemaVersion = "1.0"
	reportRecordType    = "WALKING_SKELETON_OVERHEAD_REPORT"
	observationType     = "NEUTRAL_ENVELOPE_OBSERVATION"
	maxDocumentBytes    = 1 << 20
	maxDeliverableBytes = 1 << 30

	baselineDefinition   = "buildopt:baseline:pre-product:walking-skeleton:v1"
	workUnitsFingerprint = "hmac-sha256:" +
		"dddddddddddddddddddddddddddddddd" +
		"dddddddddddddddddddddddddddddddd"
)

var strictLimitations = []string{
	"Phase 0 descriptive evidence only; no promotion gate is active.",
	"The sample is too small for MEASURE-001 causal or tail claims.",
	"Sequential arms can retain scheduler, filesystem, and Gradle cache order effects.",
	"Negative overhead observations are retained rather than truncated.",
}

// Observation is one externally timed command and deliverable validation.
type Observation struct {
	SchemaVersion       string `json:"schemaVersion"`
	RecordType          string `json:"recordType"`
	Arm                 string `json:"arm"`
	PairIndex           int    `json:"pairIndex"`
	OrderInPair         int    `json:"orderInPair"`
	CommandClass        string `json:"commandClass"`
	StartedAt           string `json:"startedAt"`
	CompletedAt         string `json:"completedAt"`
	DurationNs          int64  `json:"durationNs"`
	ExitCode            int    `json:"exitCode"`
	DeliverableSHA256   string `json:"deliverableSha256"`
	DeliverableSizeByte int64  `json:"deliverableSizeBytes"`
}

// Report is the immutable WS-009 baseline-versus-wrapper evidence.
type Report struct {
	SchemaVersion            string         `json:"schemaVersion"`
	RecordType               string         `json:"recordType"`
	CreatedAt                string         `json:"createdAt"`
	MetricDefinitionVersion  string         `json:"metricDefinitionVersion"`
	MeasurementPolicyVersion string         `json:"measurementPolicyVersion"`
	MeasurementKind          string         `json:"measurementKind"`
	ObservationMethod        string         `json:"observationMethod"`
	EffectScope              string         `json:"effectScope"`
	PromotionGateActive      bool           `json:"promotionGateActive"`
	ClockSource              string         `json:"clockSource"`
	DurationUnit             string         `json:"durationUnit"`
	StartBoundary            string         `json:"startBoundary"`
	EndBoundary              string         `json:"endBoundary"`
	ExecutionEnvironment     string         `json:"executionEnvironment"`
	RunnerClassQualified     bool           `json:"runnerClassQualified"`
	BaselineDefinitionDigest string         `json:"baselineDefinitionDigest"`
	Inputs                   ReportInputs   `json:"inputs"`
	Runner                   RunnerIdentity `json:"runner"`
	Workload                 Workload       `json:"workload"`
	Pairs                    []Pair         `json:"pairs"`
	Summary                  Summary        `json:"summary"`
	Limitations              []string       `json:"limitations"`
}

// ReportInputs bind the contracts and code exercised by the report.
type ReportInputs struct {
	RunnerSpecSHA256     string `json:"runnerSpecSha256"`
	MetricsCatalogSHA256 string `json:"metricsCatalogSha256"`
	EnvelopeSHA256       string `json:"envelopeSha256"`
	LauncherSHA256       string `json:"launcherSha256"`
	ServerSHA256         string `json:"serverSha256"`
	PluginSHA256         string `json:"pluginSha256"`
}

// RunnerIdentity records the pinned class and immutable container identity.
type RunnerIdentity struct {
	ID                    string `json:"id"`
	OperatingSystem       string `json:"operatingSystem"`
	Architecture          string `json:"architecture"`
	CPUCount              int    `json:"cpuCount"`
	MemoryBytes           int64  `json:"memoryBytes"`
	ContainerIndexDigest  string `json:"containerIndexDigest"`
	ContainerPlatformHash string `json:"containerPlatformDigest"`
	JDKVersion            string `json:"jdkVersion"`
	GradleVersion         string `json:"gradleVersion"`
}

// Workload fixes the controlled real-Gradle measurement scenario.
type Workload struct {
	ID                    string `json:"id"`
	Task                  string `json:"task"`
	PairCount             int    `json:"pairCount"`
	AlternatingOrder      bool   `json:"alternatingOrder"`
	NoDaemon              bool   `json:"noDaemon"`
	Offline               bool   `json:"offline"`
	ConfigurationCache    bool   `json:"configurationCache"`
	OptimizationsEnabled  bool   `json:"optimizationsEnabled"`
	RequiredDeliverable   string `json:"requiredDeliverable"`
	DeliverableSHA256     string `json:"deliverableSha256"`
	PipelineClass         string `json:"pipelineClass"`
	OutcomeClass          string `json:"outcomeClass"`
	WorkUnitsFingerprint  string `json:"workUnitsFingerprint"`
	MeasurementEpoch      string `json:"measurementEpoch"`
	CacheState            string `json:"cacheState"`
	WorkspaceState        string `json:"workspaceState"`
	DaemonState           string `json:"daemonState"`
	CacheStateDescription string `json:"cacheStateDescription"`
}

// Pair is one native and one wrapper observation in recorded order.
type Pair struct {
	Index                           int                `json:"index"`
	FirstArm                        string             `json:"firstArm"`
	Native                          ObservationSummary `json:"native"`
	Wrapper                         ObservationSummary `json:"wrapper"`
	ProductSynchronousOverheadMs    float64            `json:"productSynchronousOverheadMs"`
	ProductSynchronousOverheadRatio float64            `json:"productSynchronousOverheadRatio"`
}

// ObservationSummary retains the raw timing and required-output facts.
type ObservationSummary struct {
	CommandClass        string  `json:"commandClass"`
	StartedAt           string  `json:"startedAt"`
	CompletedAt         string  `json:"completedAt"`
	DurationMs          float64 `json:"durationMs"`
	ExitCode            int     `json:"exitCode"`
	DeliverableSHA256   string  `json:"deliverableSha256"`
	DeliverableSizeByte int64   `json:"deliverableSizeBytes"`
}

// Summary retains the first pair and descriptive, non-promotional statistics.
type Summary struct {
	PairCount                int     `json:"pairCount"`
	FirstExecutionIncluded   bool    `json:"firstExecutionIncluded"`
	FirstNativeMs            float64 `json:"firstNativeMs"`
	FirstWrapperMs           float64 `json:"firstWrapperMs"`
	FirstProductOverheadMs   float64 `json:"firstProductSynchronousOverheadMs"`
	NativeP50Ms              float64 `json:"nativeP50Ms"`
	WrapperP50Ms             float64 `json:"wrapperP50Ms"`
	ProductOverheadMeanMs    float64 `json:"productSynchronousOverheadMeanMs"`
	ProductOverheadP50Ms     float64 `json:"productSynchronousOverheadP50Ms"`
	ProductOverheadP95Ms     float64 `json:"productSynchronousOverheadP95Ms"`
	ProductOverheadMinimumMs float64 `json:"productSynchronousOverheadMinimumMs"`
	ProductOverheadMaximumMs float64 `json:"productSynchronousOverheadMaximumMs"`
}

type runnerSpec struct {
	SchemaVersion string `json:"schemaVersion"`
	RunnerClass   struct {
		ID                         string `json:"id"`
		OperatingSystem            string `json:"operatingSystem"`
		Architecture               string `json:"architecture"`
		CPUCount                   int    `json:"cpuCount"`
		MemoryBytes                int64  `json:"memoryBytes"`
		MinimumObservedMemoryBytes int64  `json:"minimumObservedMemoryBytes"`
		MaximumObservedMemoryBytes int64  `json:"maximumObservedMemoryBytes"`
	} `json:"runnerClass"`
	Container struct {
		Repository     string `json:"repository"`
		SourceTag      string `json:"sourceTag"`
		IndexDigest    string `json:"indexDigest"`
		PlatformDigest string `json:"platformDigest"`
		Reference      string `json:"reference"`
	} `json:"container"`
	JDK struct {
		Vendor                  string `json:"vendor"`
		Version                 string `json:"version"`
		RuntimeMajor            int    `json:"runtimeMajor"`
		CompiledBytecodeRelease int    `json:"compiledBytecodeRelease"`
	} `json:"jdk"`
	Gradle struct {
		Version              string `json:"version"`
		DistributionType     string `json:"distributionType"`
		DistributionURL      string `json:"distributionUrl"`
		DistributionSHA256   string `json:"distributionSha256"`
		WrapperJarSHA256     string `json:"wrapperJarSha256"`
		NetworkTimeoutMillis int    `json:"networkTimeoutMillis"`
		DownloadRetries      int    `json:"downloadRetries"`
		RetryBackOffMillis   int    `json:"retryBackOffMillis"`
		DSL                  string `json:"dsl"`
	} `json:"gradle"`
	Environment struct {
		Locale   string `json:"locale"`
		Timezone string `json:"timezone"`
	} `json:"environment"`
	ResolvedAt string `json:"resolvedAt"`
}

// NewObservation creates one successful neutral-envelope observation.
func NewObservation(
	arm string,
	pairIndex int,
	orderInPair int,
	commandClass string,
	startedAt time.Time,
	completedAt time.Time,
	exitCode int,
	deliverableSHA256 string,
	deliverableSize int64,
) (Observation, error) {
	observation := Observation{
		SchemaVersion:       reportSchemaVersion,
		RecordType:          observationType,
		Arm:                 arm,
		PairIndex:           pairIndex,
		OrderInPair:         orderInPair,
		CommandClass:        commandClass,
		StartedAt:           startedAt.UTC().Format(time.RFC3339Nano),
		CompletedAt:         completedAt.UTC().Format(time.RFC3339Nano),
		DurationNs:          completedAt.Sub(startedAt).Nanoseconds(),
		ExitCode:            exitCode,
		DeliverableSHA256:   deliverableSHA256,
		DeliverableSizeByte: deliverableSize,
	}
	if err := observation.Validate(); err != nil {
		return Observation{}, err
	}
	return observation, nil
}

// Validate enforces one comparable successful observation.
func (observation Observation) Validate() error {
	if observation.SchemaVersion != reportSchemaVersion ||
		observation.RecordType != observationType {
		return errors.New("unsupported neutral-envelope observation")
	}
	if observation.Arm != "NATIVE" && observation.Arm != "WRAPPER" {
		return errors.New("observation arm must be NATIVE or WRAPPER")
	}
	if observation.PairIndex < 1 ||
		(observation.OrderInPair != 1 && observation.OrderInPair != 2) ||
		strings.TrimSpace(observation.CommandClass) == "" {
		return errors.New("invalid observation pairing")
	}
	startedAt, err := parseCanonicalTime(observation.StartedAt)
	if err != nil {
		return errors.New("invalid observation startedAt")
	}
	completedAt, err := parseCanonicalTime(observation.CompletedAt)
	if err != nil || completedAt.Before(startedAt) {
		return errors.New("invalid observation completedAt")
	}
	if observation.DurationNs <= 0 ||
		observation.ExitCode != 0 ||
		!validSHA256(observation.DeliverableSHA256) ||
		observation.DeliverableSizeByte <= 0 {
		return errors.New("observation is not a successful measured deliverable")
	}
	return nil
}

// BuildReport validates, pairs, and summarizes raw observations.
func BuildReport(
	observations []Observation,
	executionEnvironment string,
	runnerSpecPath string,
	metricsCatalogPath string,
	envelopePath string,
	launcherPath string,
	serverPath string,
	pluginPath string,
	createdAt time.Time,
) (Report, error) {
	if executionEnvironment != "HOST_SMOKE" &&
		executionEnvironment != "STRICT_GOLDEN_CONTAINER" {
		return Report{}, errors.New("unsupported execution environment")
	}
	catalog, err := metricscatalog.Load(metricsCatalogPath)
	if err != nil {
		return Report{}, err
	}
	if err := validateOverheadMetrics(catalog); err != nil {
		return Report{}, err
	}
	spec, err := loadRunnerSpec(runnerSpecPath)
	if err != nil {
		return Report{}, err
	}
	pairs, err := pairObservations(observations)
	if err != nil {
		return Report{}, err
	}
	deliverableDigest := pairs[0].Native.DeliverableSHA256
	for _, pair := range pairs {
		if pair.Native.DeliverableSHA256 != deliverableDigest ||
			pair.Wrapper.DeliverableSHA256 != deliverableDigest {
			return Report{}, errors.New(
				"native and wrapper deliverables are not byte-identical",
			)
		}
	}

	runnerSpecDigest, err := fileSHA256(runnerSpecPath)
	if err != nil {
		return Report{}, err
	}
	catalogDigest, err := fileSHA256(metricsCatalogPath)
	if err != nil {
		return Report{}, err
	}
	envelopeDigest, err := fileSHA256(envelopePath)
	if err != nil {
		return Report{}, err
	}
	launcherDigest, err := fileSHA256(launcherPath)
	if err != nil {
		return Report{}, err
	}
	serverDigest, err := fileSHA256(serverPath)
	if err != nil {
		return Report{}, err
	}
	pluginDigest, err := fileSHA256(pluginPath)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		SchemaVersion:            reportSchemaVersion,
		RecordType:               reportRecordType,
		CreatedAt:                createdAt.UTC().Format(time.RFC3339Nano),
		MetricDefinitionVersion:  catalog.MetricDefinitionVersion,
		MeasurementPolicyVersion: catalog.PromotionPolicy.Version,
		MeasurementKind:          "PAIRED_ALTERNATING_BASELINE_VS_WRAPPER",
		ObservationMethod:        "EXACT",
		EffectScope:              "PRODUCT_TOTAL",
		PromotionGateActive:      false,
		ClockSource:              "MONOTONIC",
		DurationUnit:             "ms",
		StartBoundary:            "NEUTRAL_ENVELOPE_HANDOFF",
		EndBoundary:              "EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE",
		ExecutionEnvironment:     executionEnvironment,
		RunnerClassQualified: executionEnvironment ==
			"STRICT_GOLDEN_CONTAINER",
		BaselineDefinitionDigest: digestBytes(
			[]byte(baselineDefinition),
		),
		Inputs: ReportInputs{
			RunnerSpecSHA256:     runnerSpecDigest,
			MetricsCatalogSHA256: catalogDigest,
			EnvelopeSHA256:       envelopeDigest,
			LauncherSHA256:       launcherDigest,
			ServerSHA256:         serverDigest,
			PluginSHA256:         pluginDigest,
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
		Workload: Workload{
			ID:                    "gradle-neutral-probe-v1",
			Task:                  "neutralProbe",
			PairCount:             len(pairs),
			AlternatingOrder:      true,
			NoDaemon:              true,
			Offline:               true,
			ConfigurationCache:    true,
			OptimizationsEnabled:  false,
			RequiredDeliverable:   "neutral.properties",
			DeliverableSHA256:     deliverableDigest,
			PipelineClass:         "LOCAL_GRADLE_FIXTURE",
			OutcomeClass:          "SUCCESS",
			WorkUnitsFingerprint:  workUnitsFingerprint,
			MeasurementEpoch:      "walking-skeleton-v1",
			CacheState:            "ARM_COMPATIBLE_REUSE_AFTER_FIRST_PAIR",
			WorkspaceState:        "OUTPUT_REMOVED_BEFORE_EACH_OBSERVATION",
			DaemonState:           "SINGLE_USE",
			CacheStateDescription: "First measured pair retained; later pairs may reuse arm-specific configuration state.",
		},
		Pairs:       pairs,
		Summary:     summarize(pairs),
		Limitations: slices.Clone(strictLimitations),
	}
	if executionEnvironment == "HOST_SMOKE" {
		report.Limitations = append(
			report.Limitations,
			"HOST_SMOKE does not qualify the pinned 4-CPU/16-GiB runner class.",
		)
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

// Validate checks report invariants and recomputes every descriptive statistic.
func (report Report) Validate() error {
	if report.SchemaVersion != reportSchemaVersion ||
		report.RecordType != reportRecordType ||
		report.MetricDefinitionVersion != metricscatalog.Version ||
		report.MeasurementPolicyVersion != "beta-measurement-v1" ||
		report.MeasurementKind != "PAIRED_ALTERNATING_BASELINE_VS_WRAPPER" ||
		report.ObservationMethod != "EXACT" ||
		report.EffectScope != "PRODUCT_TOTAL" ||
		report.PromotionGateActive ||
		report.ClockSource != "MONOTONIC" ||
		report.DurationUnit != "ms" ||
		report.StartBoundary != "NEUTRAL_ENVELOPE_HANDOFF" ||
		report.EndBoundary !=
			"EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE" {
		return errors.New("unsupported overhead report contract")
	}
	if _, err := parseCanonicalTime(report.CreatedAt); err != nil {
		return errors.New("invalid report createdAt")
	}
	if report.ExecutionEnvironment != "HOST_SMOKE" &&
		report.ExecutionEnvironment != "STRICT_GOLDEN_CONTAINER" {
		return errors.New("invalid report execution environment")
	}
	if report.RunnerClassQualified !=
		(report.ExecutionEnvironment == "STRICT_GOLDEN_CONTAINER") {
		return errors.New("report runner qualification is inconsistent")
	}
	if !validSHA256(report.BaselineDefinitionDigest) ||
		!validSHA256(report.Inputs.RunnerSpecSHA256) ||
		!validSHA256(report.Inputs.MetricsCatalogSHA256) ||
		!validSHA256(report.Inputs.EnvelopeSHA256) ||
		!validSHA256(report.Inputs.LauncherSHA256) ||
		!validSHA256(report.Inputs.ServerSHA256) ||
		!validSHA256(report.Inputs.PluginSHA256) {
		return errors.New("report contains an invalid input digest")
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
		return errors.New("report runner does not match the golden class")
	}
	if report.Workload.ID != "gradle-neutral-probe-v1" ||
		report.Workload.Task != "neutralProbe" ||
		report.Workload.PairCount != len(report.Pairs) ||
		!report.Workload.AlternatingOrder ||
		!report.Workload.NoDaemon ||
		!report.Workload.Offline ||
		!report.Workload.ConfigurationCache ||
		report.Workload.OptimizationsEnabled ||
		report.Workload.RequiredDeliverable != "neutral.properties" ||
		!validSHA256(report.Workload.DeliverableSHA256) ||
		report.Workload.PipelineClass != "LOCAL_GRADLE_FIXTURE" ||
		report.Workload.OutcomeClass != "SUCCESS" ||
		report.Workload.WorkUnitsFingerprint != workUnitsFingerprint ||
		report.Workload.MeasurementEpoch != "walking-skeleton-v1" ||
		report.Workload.CacheState !=
			"ARM_COMPATIBLE_REUSE_AFTER_FIRST_PAIR" ||
		report.Workload.WorkspaceState !=
			"OUTPUT_REMOVED_BEFORE_EACH_OBSERVATION" ||
		report.Workload.DaemonState != "SINGLE_USE" ||
		report.Workload.CacheStateDescription !=
			"First measured pair retained; later pairs may reuse arm-specific configuration state." {
		return errors.New("invalid overhead workload")
	}
	if len(report.Pairs) < 1 {
		return errors.New("overhead report contains no pairs")
	}
	for index, pair := range report.Pairs {
		if err := pair.validate(index+1, report.Workload.DeliverableSHA256); err != nil {
			return err
		}
	}
	expected := summarize(report.Pairs)
	if !summariesEqual(report.Summary, expected) {
		return errors.New("overhead report summary does not match raw pairs")
	}
	expectedLimitations := slices.Clone(strictLimitations)
	if report.ExecutionEnvironment == "HOST_SMOKE" {
		expectedLimitations = append(
			expectedLimitations,
			"HOST_SMOKE does not qualify the pinned 4-CPU/16-GiB runner class.",
		)
	}
	if !slices.Equal(report.Limitations, expectedLimitations) {
		return errors.New("overhead report limitations are incomplete")
	}
	return nil
}

// WriteJSON atomically writes a private report or observation.
func WriteJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode neutral-envelope JSON: %w", err)
	}
	content = append(content, '\n')
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create neutral-envelope output directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".neutral-envelope-*.tmp")
	if err != nil {
		return fmt.Errorf("create neutral-envelope temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("set neutral-envelope output mode: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return fmt.Errorf("write neutral-envelope output: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync neutral-envelope output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close neutral-envelope output: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish neutral-envelope output: %w", err)
	}
	return nil
}

// LoadObservation decodes one strict raw observation.
func LoadObservation(path string) (Observation, error) {
	var observation Observation
	if err := loadJSON(path, &observation); err != nil {
		return Observation{}, err
	}
	if err := observation.Validate(); err != nil {
		return Observation{}, err
	}
	return observation, nil
}

// LoadReport decodes one strict overhead report.
func LoadReport(path string) (Report, error) {
	var report Report
	if err := loadJSON(path, &report); err != nil {
		return Report{}, err
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

// FileSHA256 returns the canonical digest and size of one deliverable.
func FileSHA256(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open required deliverable: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(
		hash,
		io.LimitReader(file, maxDeliverableBytes+1),
	)
	if err != nil {
		return "", 0, fmt.Errorf("hash required deliverable: %w", err)
	}
	if size <= 0 {
		return "", 0, errors.New("required deliverable is empty")
	}
	if size > maxDeliverableBytes {
		return "", 0, errors.New("required deliverable exceeds 1 GiB")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), size, nil
}

func pairObservations(observations []Observation) ([]Pair, error) {
	if len(observations) == 0 || len(observations)%2 != 0 {
		return nil, errors.New("observations must form complete pairs")
	}
	for _, observation := range observations {
		if err := observation.Validate(); err != nil {
			return nil, err
		}
	}
	pairCount := len(observations) / 2
	pairs := make([]Pair, pairCount)
	for pairIndex := 1; pairIndex <= pairCount; pairIndex++ {
		var native, wrapper *Observation
		for index := range observations {
			observation := &observations[index]
			if observation.PairIndex != pairIndex {
				continue
			}
			switch observation.Arm {
			case "NATIVE":
				if native != nil {
					return nil, errors.New("pair repeats NATIVE observation")
				}
				native = observation
			case "WRAPPER":
				if wrapper != nil {
					return nil, errors.New("pair repeats WRAPPER observation")
				}
				wrapper = observation
			}
		}
		if native == nil || wrapper == nil ||
			native.OrderInPair == wrapper.OrderInPair {
			return nil, fmt.Errorf("pair %d is incomplete", pairIndex)
		}
		expectedFirst := "NATIVE"
		if pairIndex%2 == 0 {
			expectedFirst = "WRAPPER"
		}
		first := native
		if wrapper.OrderInPair == 1 {
			first = wrapper
		}
		if first.Arm != expectedFirst {
			return nil, fmt.Errorf("pair %d does not alternate order", pairIndex)
		}
		nativeSummary := summarizeObservation(*native)
		wrapperSummary := summarizeObservation(*wrapper)
		overhead := wrapperSummary.DurationMs - nativeSummary.DurationMs
		ratio := overhead / nativeSummary.DurationMs
		pairs[pairIndex-1] = Pair{
			Index:                           pairIndex,
			FirstArm:                        first.Arm,
			Native:                          nativeSummary,
			Wrapper:                         wrapperSummary,
			ProductSynchronousOverheadMs:    overhead,
			ProductSynchronousOverheadRatio: ratio,
		}
	}
	return pairs, nil
}

func (pair Pair) validate(index int, expectedDigest string) error {
	if pair.Index != index ||
		(pair.FirstArm != "NATIVE" && pair.FirstArm != "WRAPPER") ||
		(index%2 == 1 && pair.FirstArm != "NATIVE") ||
		(index%2 == 0 && pair.FirstArm != "WRAPPER") ||
		pair.Native.DurationMs <= 0 ||
		pair.Wrapper.DurationMs <= 0 ||
		pair.Native.ExitCode != 0 ||
		pair.Wrapper.ExitCode != 0 ||
		pair.Native.CommandClass != "gradle-neutral-probe-native-v1" ||
		pair.Wrapper.CommandClass !=
			"buildopt-gradle-neutral-probe-wrapper-v1" ||
		pair.Native.DeliverableSHA256 != expectedDigest ||
		pair.Wrapper.DeliverableSHA256 != expectedDigest ||
		pair.Native.DeliverableSizeByte <= 0 ||
		pair.Wrapper.DeliverableSizeByte <= 0 {
		return fmt.Errorf("invalid overhead pair %d", index)
	}
	nativeStarted, nativeCompleted, err := validateObservationSummary(pair.Native)
	if err != nil {
		return fmt.Errorf("invalid overhead pair %d native timestamps", index)
	}
	wrapperStarted, wrapperCompleted, err := validateObservationSummary(
		pair.Wrapper,
	)
	if err != nil {
		return fmt.Errorf("invalid overhead pair %d wrapper timestamps", index)
	}
	if pair.FirstArm == "NATIVE" && nativeCompleted.After(wrapperStarted) {
		return fmt.Errorf("overhead pair %d native arm was not first", index)
	}
	if pair.FirstArm == "WRAPPER" && wrapperCompleted.After(nativeStarted) {
		return fmt.Errorf("overhead pair %d wrapper arm was not first", index)
	}
	expectedOverhead := pair.Wrapper.DurationMs - pair.Native.DurationMs
	expectedRatio := expectedOverhead / pair.Native.DurationMs
	if !closeFloat(pair.ProductSynchronousOverheadMs, expectedOverhead) ||
		!closeFloat(pair.ProductSynchronousOverheadRatio, expectedRatio) {
		return fmt.Errorf("overhead pair %d does not reconcile", index)
	}
	return nil
}

func summarizeObservation(observation Observation) ObservationSummary {
	return ObservationSummary{
		CommandClass:        observation.CommandClass,
		StartedAt:           observation.StartedAt,
		CompletedAt:         observation.CompletedAt,
		DurationMs:          float64(observation.DurationNs) / 1e6,
		ExitCode:            observation.ExitCode,
		DeliverableSHA256:   observation.DeliverableSHA256,
		DeliverableSizeByte: observation.DeliverableSizeByte,
	}
}

func summarize(pairs []Pair) Summary {
	native := make([]float64, len(pairs))
	wrapper := make([]float64, len(pairs))
	overhead := make([]float64, len(pairs))
	var sum float64
	for index, pair := range pairs {
		native[index] = pair.Native.DurationMs
		wrapper[index] = pair.Wrapper.DurationMs
		overhead[index] = pair.ProductSynchronousOverheadMs
		sum += overhead[index]
	}
	slices.Sort(native)
	slices.Sort(wrapper)
	slices.Sort(overhead)
	return Summary{
		PairCount:                len(pairs),
		FirstExecutionIncluded:   true,
		FirstNativeMs:            pairs[0].Native.DurationMs,
		FirstWrapperMs:           pairs[0].Wrapper.DurationMs,
		FirstProductOverheadMs:   pairs[0].ProductSynchronousOverheadMs,
		NativeP50Ms:              nearestRank(native, 0.50),
		WrapperP50Ms:             nearestRank(wrapper, 0.50),
		ProductOverheadMeanMs:    sum / float64(len(pairs)),
		ProductOverheadP50Ms:     nearestRank(overhead, 0.50),
		ProductOverheadP95Ms:     nearestRank(overhead, 0.95),
		ProductOverheadMinimumMs: overhead[0],
		ProductOverheadMaximumMs: overhead[len(overhead)-1],
	}
}

func summariesEqual(actual, expected Summary) bool {
	return actual.PairCount == expected.PairCount &&
		actual.FirstExecutionIncluded == expected.FirstExecutionIncluded &&
		closeFloat(actual.FirstNativeMs, expected.FirstNativeMs) &&
		closeFloat(actual.FirstWrapperMs, expected.FirstWrapperMs) &&
		closeFloat(actual.FirstProductOverheadMs, expected.FirstProductOverheadMs) &&
		closeFloat(actual.NativeP50Ms, expected.NativeP50Ms) &&
		closeFloat(actual.WrapperP50Ms, expected.WrapperP50Ms) &&
		closeFloat(actual.ProductOverheadMeanMs, expected.ProductOverheadMeanMs) &&
		closeFloat(actual.ProductOverheadP50Ms, expected.ProductOverheadP50Ms) &&
		closeFloat(actual.ProductOverheadP95Ms, expected.ProductOverheadP95Ms) &&
		closeFloat(actual.ProductOverheadMinimumMs, expected.ProductOverheadMinimumMs) &&
		closeFloat(actual.ProductOverheadMaximumMs, expected.ProductOverheadMaximumMs)
}

func nearestRank(sorted []float64, quantile float64) float64 {
	index := int(math.Ceil(quantile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func closeFloat(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9*math.Max(1, math.Abs(right))
}

func loadRunnerSpec(path string) (runnerSpec, error) {
	var spec runnerSpec
	if err := loadJSON(path, &spec); err != nil {
		return runnerSpec{}, fmt.Errorf("load golden runner spec: %w", err)
	}
	if spec.SchemaVersion != "buildopt.dev/golden-lane-runner/v1" ||
		spec.RunnerClass.ID == "" ||
		spec.RunnerClass.OperatingSystem != "linux" ||
		spec.RunnerClass.Architecture != "amd64" ||
		spec.RunnerClass.CPUCount != 4 ||
		spec.RunnerClass.MemoryBytes != 17179869184 ||
		!validSHA256(spec.Container.IndexDigest) ||
		!validSHA256(spec.Container.PlatformDigest) ||
		spec.JDK.Version == "" ||
		spec.Gradle.Version != "9.6.1" {
		return runnerSpec{}, errors.New("golden runner spec is incompatible")
	}
	return spec, nil
}

func loadJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxDocumentBytes {
		return errors.New("JSON document exceeds 1 MiB")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxDocumentBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("JSON document must contain exactly one value")
	}
	return nil
}

func validateObservationSummary(
	observation ObservationSummary,
) (time.Time, time.Time, error) {
	startedAt, err := parseCanonicalTime(observation.StartedAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	completedAt, err := parseCanonicalTime(observation.CompletedAt)
	if err != nil || completedAt.Before(startedAt) {
		return time.Time{}, time.Time{}, errors.New("invalid completion time")
	}
	return startedAt, completedAt, nil
}

func parseCanonicalTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || value != parsed.UTC().Format(time.RFC3339Nano) {
		return time.Time{}, errors.New(
			"timestamp must be canonical UTC RFC3339",
		)
	}
	return parsed, nil
}

func validateOverheadMetrics(catalog metricscatalog.Catalog) error {
	required := map[string]struct {
		formula        string
		unit           string
		grain          string
		signConvention string
	}{
		"customerVisibleBuildMs": {
			formula:        "monotonic(endBoundary - startBoundary)",
			unit:           "ms",
			grain:          "BUILD_SESSION",
			signConvention: "NON_NEGATIVE",
		},
		"productSynchronousOverheadMs": {
			formula:        "policy-off wrapper customerVisibleBuildMs - native baseline customerVisibleBuildMs",
			unit:           "ms",
			grain:          "PAIRED_EXECUTION",
			signConvention: "POSITIVE_IS_REGRESSION",
		},
		"productSynchronousOverheadRatio": {
			formula:        "productSynchronousOverheadMs / native baseline customerVisibleBuildMs",
			unit:           "ratio",
			grain:          "PAIRED_EXECUTION",
			signConvention: "POSITIVE_IS_REGRESSION",
		},
	}
	for _, metric := range catalog.Metrics {
		expected, exists := required[metric.ID]
		if !exists {
			continue
		}
		if metric.Formula != expected.formula ||
			metric.Unit != expected.unit ||
			metric.Grain != expected.grain ||
			metric.SignConvention != expected.signConvention ||
			metric.StartBoundary != "NEUTRAL_ENVELOPE_HANDOFF" ||
			metric.EndBoundary !=
				"EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE" {
			return fmt.Errorf(
				"metric %s is incompatible with the overhead report",
				metric.ID,
			)
		}
		delete(required, metric.ID)
	}
	if len(required) != 0 {
		return errors.New("metrics catalog lacks overhead report definitions")
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 ||
		!strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && value == strings.ToLower(value)
}
