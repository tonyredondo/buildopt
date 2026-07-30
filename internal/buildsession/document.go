package buildsession

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"time"

	"github.com/tonyredondo/buildopt/internal/metricscatalog"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	schemaVersion = "1.0"
	recordType    = "BUILD_SESSION"

	envelopeVersion    = "local-envelope-v1"
	baselineDefinition = "buildopt:baseline:pre-product:walking-skeleton:v1"
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
	Recovery             *Recovery            `json:"recovery,omitempty"`
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

// Recovery records an immutable partial assembly and its missing event ranges.
type Recovery struct {
	Source                string                 `json:"source"`
	RecoveredAt           string                 `json:"recoveredAt"`
	Reason                string                 `json:"reason"`
	MissingSequenceRanges []MissingSequenceRange `json:"missingSequenceRanges"`
}

// MissingSequenceRange is one inclusive gap in a per-build JSONL sequence.
type MissingSequenceRange struct {
	First int `json:"first"`
	Last  int `json:"last"`
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
	} else if record.Outcome == sessioningest.OutcomeCancelled {
		timeToFailure = unavailable(
			"The build was cancelled before a build failure was observed",
		)
	}

	requiredDeliverablesStatus := "NOT_REQUIRED"
	if record.Outcome == sessioningest.OutcomeCancelled {
		requiredDeliverablesStatus = "UNKNOWN"
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
			RequiredDeliverablesStatus: requiredDeliverablesStatus,
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
			MetricDefinitionVersion:   metricscatalog.Version,
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

// RecoverPartial derives an immutable partial BUILD_SESSION from an observed
// complete candidate when final JSONL publication events are missing. It never
// fills missing measurements or incorporates later aggregate effects.
func RecoverPartial(
	candidate Document,
	recoveredAt time.Time,
	missing []MissingSequenceRange,
) (Document, error) {
	if !candidate.Complete ||
		candidate.RecordType != recordType ||
		candidate.SchemaVersion != schemaVersion ||
		candidate.Recovery != nil {
		return Document{}, errors.New(
			"partial recovery requires a complete BUILD_SESSION candidate",
		)
	}
	if recoveredAt.IsZero() ||
		recoveredAt.Location() != time.UTC {
		return Document{}, errors.New(
			"partial recovery timestamp must be non-zero UTC",
		)
	}
	completedAt, err := time.Parse(time.RFC3339Nano, candidate.Build.CompletedAt)
	if err != nil || recoveredAt.Before(completedAt) {
		return Document{}, errors.New(
			"partial recovery timestamp precedes build completion",
		)
	}
	ranges, err := validateMissingSequenceRanges(missing)
	if err != nil {
		return Document{}, err
	}

	candidate.Complete = false
	candidate.MeasurementMetadata.Status = "PARTIAL"
	measurement := &candidate.Performance.CustomerVisibleBuildMs
	measurement.State = "PARTIAL"
	measurement.Reason =
		"The final BUILD_SESSION publication sequence is incomplete"
	candidate.Recovery = &Recovery{
		Source:      "EVENT_REPLAY",
		RecoveredAt: recoveredAt.Format(time.RFC3339Nano),
		Reason: "The JSONL producer stopped before the final immutable " +
			"BUILD_SESSION publication event",
		MissingSequenceRanges: ranges,
	}
	return candidate, nil
}

func validateMissingSequenceRanges(
	missing []MissingSequenceRange,
) ([]MissingSequenceRange, error) {
	if len(missing) == 0 || len(missing) > 1024 {
		return nil, errors.New(
			"partial recovery requires between 1 and 1024 missing ranges",
		)
	}
	ranges := append([]MissingSequenceRange(nil), missing...)
	slices.SortFunc(ranges, func(left, right MissingSequenceRange) int {
		if left.First != right.First {
			return left.First - right.First
		}
		return left.Last - right.Last
	})
	previousLast := -1
	for index, interval := range ranges {
		if interval.First < 0 ||
			interval.Last < interval.First ||
			index > 0 && interval.First <= previousLast+1 {
			return nil, errors.New(
				"partial recovery ranges must be non-negative, disjoint, and non-adjacent",
			)
		}
		previousLast = interval.Last
	}
	return ranges, nil
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
