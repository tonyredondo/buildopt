package buildsession

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	schemaVersion = "1.0"
	recordType    = "BUILD_SESSION"

	metricDefinitionVersion = "build-impact-v1"
	envelopeVersion         = "local-envelope-v1"
	baselineDefinition      = "buildopt:baseline:pre-product:walking-skeleton:v1"
)

var versionPattern = regexp.MustCompile(
	`^(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:\.(?:0|[1-9][0-9]*))?(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`,
)

// Document is the dependency-free producer model for the normative
// BUILD_SESSION v1 JSON Schema.
type Document struct {
	SchemaVersion        string               `json:"schemaVersion"`
	RecordType           string               `json:"recordType"`
	Complete             bool                 `json:"complete"`
	Build                Build                `json:"build"`
	GradleInvocations    []GradleInvocation   `json:"gradleInvocations"`
	MeasurementMetadata  MeasurementMetadata  `json:"measurementMetadata"`
	ExperimentAssignment ExperimentAssignment `json:"experimentAssignment"`
	Performance          Performance          `json:"performance"`
	Workload             Workload             `json:"workload"`
	Capabilities         Capabilities         `json:"capabilities"`
}

// Build contains neutral-envelope facts for one completed build.
type Build struct {
	ID                         string `json:"id"`
	RepositoryID               string `json:"repositoryId"`
	Revision                   string `json:"revision"`
	StartedAt                  string `json:"startedAt"`
	CompletedAt                string `json:"completedAt"`
	Outcome                    string `json:"outcome"`
	ExitCode                   int    `json:"exitCode"`
	RequiredDeliverablesStatus string `json:"requiredDeliverablesStatus"`
	PluginVersion              string `json:"pluginVersion"`
}

// GradleInvocation contains the one authenticated Gradle process in the
// walking-skeleton session.
type GradleInvocation struct {
	ID             string              `json:"id"`
	RequestedTasks []string            `json:"requestedTasks"`
	StartedAt      string              `json:"startedAt"`
	CompletedAt    string              `json:"completedAt"`
	Outcome        string              `json:"outcome"`
	ExitCode       int                 `json:"exitCode"`
	ProcessMs      DurationMeasurement `json:"processMs"`
}

// MeasurementMetadata declares the neutral clock and envelope semantics.
type MeasurementMetadata struct {
	MetricDefinitionVersion   string `json:"metricDefinitionVersion"`
	Status                    string `json:"status"`
	DurationUnit              string `json:"durationUnit"`
	ClockSource               string `json:"clockSource"`
	TimestampFormat           string `json:"timestampFormat"`
	EnvelopeVersion           string `json:"envelopeVersion"`
	StartBoundary             string `json:"startBoundary"`
	EndBoundary               string `json:"endBoundary"`
	ReconciliationToleranceMs int64  `json:"reconciliationToleranceMs"`
}

// ExperimentAssignment records the pre-outcome passthrough assignment.
type ExperimentAssignment struct {
	MeasurementEpoch         int     `json:"measurementEpoch"`
	EffectScope              string  `json:"effectScope"`
	BaselineDefinitionDigest string  `json:"baselineDefinitionDigest"`
	AssignmentUnit           string  `json:"assignmentUnit"`
	Arm                      string  `json:"arm"`
	AssignmentProbability    float64 `json:"assignmentProbability"`
	AssignedAt               string  `json:"assignedAt"`
	Eligibility              string  `json:"eligibility"`
	ExclusionReason          string  `json:"exclusionReason"`
}

// Performance exposes observed durations and explicit unavailable metrics.
type Performance struct {
	CustomerVisibleBuildMs    DurationMeasurement `json:"customerVisibleBuildMs"`
	CustomerVisibleFeedbackMs DurationMeasurement `json:"customerVisibleFeedbackMs"`
	CIQueueMs                 DurationMeasurement `json:"ciQueueMs"`
	GradleProcessUnionMs      DurationMeasurement `json:"gradleProcessUnionMs"`
	BuildCriticalPathMs       DurationMeasurement `json:"buildCriticalPathMs"`
	TimeToFirstBuildFailureMs DurationMeasurement `json:"timeToFirstBuildFailureMs"`
}

// DurationMeasurement encodes availability and method without invented values.
type DurationMeasurement struct {
	State             string `json:"state"`
	Unit              string `json:"unit"`
	Method            string `json:"method"`
	ValueMs           *int64 `json:"valueMs,omitempty"`
	MethodDescription string `json:"methodDescription,omitempty"`
	Reason            string `json:"reason,omitempty"`
}

// Workload identifies the predeclared local walking-skeleton workload.
type Workload struct {
	Environment                        string `json:"environment"`
	PipelineClass                      string `json:"pipelineClass"`
	RunnerClass                        string `json:"runnerClass"`
	WorkspaceState                     string `json:"workspaceState"`
	DaemonState                        string `json:"daemonState"`
	CacheState                         string `json:"cacheState"`
	ChangeClass                        string `json:"changeClass"`
	SourceStateDigest                  string `json:"sourceStateDigest"`
	RequestedWorkManifestDigest        string `json:"requestedWorkManifestDigest"`
	RequiredDeliverablesManifestDigest string `json:"requiredDeliverablesManifestDigest"`
	WorkUnitsFingerprint               string `json:"workUnitsFingerprint"`
	TokenKeyVersion                    string `json:"tokenKeyVersion"`
	TrustDomain                        string `json:"trustDomain"`
}

// Capabilities states the method available for every required metric family.
type Capabilities struct {
	MeasurementEnvelope Capability `json:"measurementEnvelope"`
	TaskOutcomes        Capability `json:"taskOutcomes"`
	CriticalPath        Capability `json:"criticalPath"`
	CacheMissReasons    Capability `json:"cacheMissReasons"`
	ResourceUsage       Capability `json:"resourceUsage"`
	ProductOverhead     Capability `json:"productOverhead"`
	CostInputs          Capability `json:"costInputs"`
}

// Capability describes whether and how one metric family was observed.
type Capability struct {
	Method string `json:"method"`
	Reason string `json:"reason,omitempty"`
}

// NewDocument converts a complete authenticated ingest record into the
// normative BUILD_SESSION v1 producer model.
func NewDocument(record sessioningest.Record) (Document, error) {
	if err := record.Validate(); err != nil {
		return Document{}, err
	}
	if record.ExportContext == nil || record.GradleInvocation == nil {
		return Document{}, errors.New(
			"authenticated Gradle invocation and export context are required",
		)
	}
	if !versionPattern.MatchString(record.GradleInvocation.PluginVersion) {
		return Document{}, errors.New(
			"Gradle plugin version is not schema-compatible",
		)
	}

	requestedTasks := append(
		[]string(nil),
		record.ExportContext.RequestedTasks...,
	)
	requestedManifest, err := json.Marshal(requestedTasks)
	if err != nil {
		return Document{}, errors.New("encode requested work manifest")
	}

	timeToFailure := unavailable(
		"The build completed successfully",
	)
	if record.Outcome == sessioningest.OutcomeBuildFailure {
		timeToFailure = partialApproximation(
			record.DurationMs,
			"Only the final process failure boundary was observed",
			"The final child exit is a conservative upper bound for the first actionable failure",
		)
	}

	return Document{
		SchemaVersion: schemaVersion,
		RecordType:    recordType,
		Complete:      true,
		Build: Build{
			ID:                         record.SessionID,
			RepositoryID:               record.ExportContext.RepositoryID,
			Revision:                   record.ExportContext.Revision,
			StartedAt:                  record.StartedAt,
			CompletedAt:                record.CompletedAt,
			Outcome:                    record.Outcome,
			ExitCode:                   record.ExitCode,
			RequiredDeliverablesStatus: "NOT_REQUIRED",
			PluginVersion:              record.GradleInvocation.PluginVersion,
		},
		GradleInvocations: []GradleInvocation{
			{
				ID:             record.GradleInvocation.ID,
				RequestedTasks: requestedTasks,
				StartedAt:      record.GradleInvocation.StartedAt,
				CompletedAt:    record.GradleInvocation.CompletedAt,
				Outcome:        record.Outcome,
				ExitCode:       record.ExitCode,
				ProcessMs: approximated(
					record.GradleInvocation.DurationMs,
					"Launcher interval from child start request through wait completion",
				),
			},
		},
		MeasurementMetadata: MeasurementMetadata{
			MetricDefinitionVersion:   metricDefinitionVersion,
			Status:                    "COMPLETE",
			DurationUnit:              "ms",
			ClockSource:               "MONOTONIC",
			TimestampFormat:           "RFC3339_UTC",
			EnvelopeVersion:           envelopeVersion,
			StartBoundary:             "NEUTRAL_ENVELOPE_HANDOFF",
			EndBoundary:               "EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE",
			ReconciliationToleranceMs: 5,
		},
		ExperimentAssignment: ExperimentAssignment{
			MeasurementEpoch: 0,
			EffectScope:      "PRODUCT_TOTAL",
			BaselineDefinitionDigest: digest(
				"sha256:",
				[]byte(baselineDefinition),
			),
			AssignmentUnit:        "BUILD_SESSION",
			Arm:                   "PASSTHROUGH",
			AssignmentProbability: 0,
			AssignedAt:            record.StartedAt,
			Eligibility:           "EXCLUDED",
			ExclusionReason:       "No active experiment in the walking skeleton",
		},
		Performance: Performance{
			CustomerVisibleBuildMs: exact(record.DurationMs),
			CustomerVisibleFeedbackMs: unavailable(
				"A local build has no authenticated CI eligibility timestamp",
			),
			CIQueueMs: unavailable(
				"A local build has no authenticated CI runner-assignment timestamp",
			),
			GradleProcessUnionMs: approximated(
				record.GradleInvocation.DurationMs,
				"One launcher-observed Gradle child interval",
			),
			BuildCriticalPathMs: unavailable(
				"Critical-path observation is not active in the walking skeleton",
			),
			TimeToFirstBuildFailureMs: timeToFailure,
		},
		Workload: Workload{
			Environment:                 "LOCAL",
			PipelineClass:               "walking-skeleton",
			RunnerClass:                 "unknown",
			WorkspaceState:              "UNKNOWN",
			DaemonState:                 "UNKNOWN",
			CacheState:                  "DISABLED",
			ChangeClass:                 "UNKNOWN",
			SourceStateDigest:           record.ExportContext.SourceStateDigest,
			RequestedWorkManifestDigest: digest("sha256:", requestedManifest),
			RequiredDeliverablesManifestDigest: digest(
				"sha256:",
				[]byte("[]"),
			),
			WorkUnitsFingerprint: record.ExportContext.WorkUnitsFingerprint,
			TokenKeyVersion:      record.ExportContext.TokenKeyVersion,
			TrustDomain:          record.ExportContext.TrustDomain,
		},
		Capabilities: Capabilities{
			MeasurementEnvelope: Capability{Method: "EXACT"},
			TaskOutcomes: unavailableCapability(
				"Only the authenticated Gradle handshake is active",
			),
			CriticalPath: unavailableCapability(
				"Critical-path observation is not active in the walking skeleton",
			),
			CacheMissReasons: unavailableCapability(
				"The managed cache is disabled",
			),
			ResourceUsage: unavailableCapability(
				"Resource instrumentation is disabled",
			),
			ProductOverhead: unavailableCapability(
				"Product-overhead decomposition is not active in the walking skeleton",
			),
			CostInputs: unavailableCapability(
				"No pricing adapter is configured",
			),
		},
	}, nil
}

func exact(value int64) DurationMeasurement {
	return DurationMeasurement{
		State:   "COMPLETE",
		Unit:    "ms",
		Method:  "EXACT",
		ValueMs: &value,
	}
}

func approximated(value int64, description string) DurationMeasurement {
	return DurationMeasurement{
		State:             "COMPLETE",
		Unit:              "ms",
		Method:            "APPROXIMATED",
		ValueMs:           &value,
		MethodDescription: description,
	}
}

func partialApproximation(
	value int64,
	reason string,
	description string,
) DurationMeasurement {
	return DurationMeasurement{
		State:             "PARTIAL",
		Unit:              "ms",
		Method:            "APPROXIMATED",
		ValueMs:           &value,
		MethodDescription: description,
		Reason:            reason,
	}
}

func unavailable(reason string) DurationMeasurement {
	return DurationMeasurement{
		State:  "UNAVAILABLE",
		Unit:   "ms",
		Method: "UNAVAILABLE",
		Reason: reason,
	}
}

func unavailableCapability(reason string) Capability {
	return Capability{Method: "UNAVAILABLE", Reason: reason}
}

func digest(prefix string, content []byte) string {
	sum := sha256.Sum256(content)
	return prefix + hex.EncodeToString(sum[:])
}
