package neutralenvelope

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	pilotSchemaVersion       = "1.0"
	pilotAssignmentType      = "CAUSAL_PILOT_ASSIGNMENT"
	pilotObservationType     = "CAUSAL_PILOT_OBSERVATION"
	experimentResultType     = "EXPERIMENT_RESULT"
	experimentResultStream   = "experiment-results.jsonl"
	maxPilotPairs            = 256
	pilotBootstrapReplicates = 4096
	maxPilotJSONLBytes       = 64 << 20
	maxPilotJSONLLineBytes   = 1 << 20
)

var pilotIdentifierPattern = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9._:/@+-]{0,255}$`,
)

// PilotDefinition fixes every stratum and treatment-independent input before
// a causal pilot assignment is persisted.
type PilotDefinition struct {
	ExperimentID             string
	MeasurementEpoch         int
	ActionID                 string
	BaselineDefinitionDigest string
	ControlDefinitionDigest  string
	CohortID                 string
	Environment              string
	PipelineClass            string
	RunnerClass              string
	WorkUnitsFingerprint     string
	RequiredDeliverable      string
}

// PilotAssignment is the immutable pre-outcome arm assignment consumed by the
// neutral measurement envelope.
type PilotAssignment struct {
	SchemaVersion            string   `json:"schemaVersion"`
	RecordType               string   `json:"recordType"`
	ExperimentID             string   `json:"experimentId"`
	MeasurementEpoch         int      `json:"measurementEpoch"`
	PairIndex                int      `json:"pairIndex"`
	OrderInPair              int      `json:"orderInPair"`
	Arm                      string   `json:"arm"`
	AssignmentProbability    float64  `json:"assignmentProbability"`
	AssignedAt               string   `json:"assignedAt"`
	EffectScope              string   `json:"effectScope"`
	ActionID                 string   `json:"actionId"`
	BaselineDefinitionDigest string   `json:"baselineDefinitionDigest"`
	ControlDefinitionDigest  string   `json:"controlDefinitionDigest"`
	MetricDefinitionVersion  string   `json:"metricDefinitionVersion"`
	MeasurementPolicyVersion string   `json:"measurementPolicyVersion"`
	CohortID                 string   `json:"cohortId"`
	AnalysisUnit             string   `json:"analysisUnit"`
	AssignmentUnit           string   `json:"assignmentUnit"`
	Estimand                 string   `json:"estimand"`
	Environment              string   `json:"environment"`
	PipelineClass            string   `json:"pipelineClass"`
	RunnerClass              string   `json:"runnerClass"`
	OutcomeStratum           string   `json:"outcomeStratum"`
	WorkUnitsFingerprint     string   `json:"workUnitsFingerprint"`
	WorkspaceState           string   `json:"workspaceState"`
	DaemonState              string   `json:"daemonState"`
	CacheState               string   `json:"cacheState"`
	CommandClass             string   `json:"commandClass"`
	RequiredDeliverable      string   `json:"requiredDeliverable"`
	ExclusionPolicy          []string `json:"exclusionPolicy"`
}

// PilotObservation is one externally timed assigned build, including failed
// and cancelled outcomes so the result cannot drop treatment regressions.
type PilotObservation struct {
	SchemaVersion       string          `json:"schemaVersion"`
	RecordType          string          `json:"recordType"`
	BuildSessionID      string          `json:"buildSessionId"`
	AssignmentDigest    string          `json:"assignmentDigest"`
	Assignment          PilotAssignment `json:"assignment"`
	StartedAt           string          `json:"startedAt"`
	CompletedAt         string          `json:"completedAt"`
	DurationMs          int64           `json:"durationMs"`
	Outcome             string          `json:"outcome"`
	ExitCode            int             `json:"exitCode"`
	DeliverableStatus   string          `json:"requiredDeliverableStatus"`
	DeliverableSHA256   string          `json:"deliverableSha256,omitempty"`
	DeliverableSizeByte int64           `json:"deliverableSizeBytes,omitempty"`
}

// ExperimentResult is the A0-009 producer model for the normative
// EXPERIMENT_RESULT v1 schema.
type ExperimentResult struct {
	SchemaVersion            string           `json:"schemaVersion"`
	RecordType               string           `json:"recordType"`
	ExperimentID             string           `json:"experimentId"`
	ResultVersion            int              `json:"resultVersion"`
	Status                   string           `json:"status"`
	AsOf                     string           `json:"asOf"`
	Window                   ResultWindow     `json:"window"`
	EffectScope              string           `json:"effectScope"`
	ActionIDs                []string         `json:"actionIds"`
	MeasurementEpoch         int              `json:"measurementEpoch"`
	BaselineDefinitionDigest string           `json:"baselineDefinitionDigest"`
	ControlDefinitionDigest  string           `json:"controlDefinitionDigest"`
	MetricDefinitionVersion  string           `json:"metricDefinitionVersion"`
	MeasurementPolicyVersion string           `json:"measurementPolicyVersion"`
	Population               ResultPopulation `json:"population"`
	Samples                  ResultSamples    `json:"samples"`
	Method                   ResultMethod     `json:"method"`
	Effects                  ResultEffects    `json:"effects"`
	Decision                 ResultDecision   `json:"decision"`
}

type ResultWindow struct {
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt"`
}

type ResultPopulation struct {
	CohortID       string   `json:"cohortId"`
	AnalysisUnit   string   `json:"analysisUnit"`
	AssignmentUnit string   `json:"assignmentUnit"`
	Estimand       string   `json:"estimand"`
	RequiredStrata []string `json:"requiredStrata"`
}

type ResultSamples struct {
	Assigned                 ResultArmCounts     `json:"assigned"`
	Analyzed                 ResultArmCounts     `json:"analyzed"`
	Outcomes                 ResultOutcomesByArm `json:"outcomes"`
	ExcludedSampleSize       int                 `json:"excludedSampleSize"`
	Exclusions               []ResultExclusion   `json:"exclusions"`
	MeasurementCoverageRatio float64             `json:"measurementCoverageRatio"`
}

type ResultArmCounts struct {
	Candidate int `json:"candidate"`
	Control   int `json:"control"`
}

type ResultOutcomesByArm struct {
	Candidate ResultOutcomeCounts `json:"candidate"`
	Control   ResultOutcomeCounts `json:"control"`
}

type ResultOutcomeCounts struct {
	Success      int `json:"SUCCESS"`
	BuildFailure int `json:"BUILD_FAILURE"`
	InfraFailure int `json:"INFRA_FAILURE"`
	Cancelled    int `json:"CANCELLED"`
}

type ResultExclusion struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type ResultMethod struct {
	AnalysisMethod  string `json:"analysisMethod"`
	EffectStatistic string `json:"effectStatistic"`
	IntervalType    string `json:"intervalType"`
}

type ResultEffects struct {
	ObservedNetBuildTimeSavedMs           int64      `json:"observedNetBuildTimeSavedMs"`
	ObservedNetBuildTimeSavedInterval95Ms [2]int64   `json:"observedNetBuildTimeSavedInterval95Ms"`
	ObservedBuildTimeReductionRatio       float64    `json:"observedBuildTimeReductionRatio"`
	ObservedBuildTimeReductionInterval95  [2]float64 `json:"observedBuildTimeReductionInterval95"`
	BuildFailureRateDelta                 float64    `json:"buildFailureRateDelta"`
	CustomerVisibleBuildP95DeltaMs        int64      `json:"customerVisibleBuildP95DeltaMs"`
	CustomerVisibleBuildP95Interval95Ms   [2]int64   `json:"customerVisibleBuildP95DeltaInterval95Ms"`
	IncrementalActionOverheadMs           int64      `json:"incrementalActionOverheadMs"`
	RequiredDeliverableDivergenceRate     float64    `json:"requiredDeliverableDivergenceRate"`
	ProductAttributableFailureRate        float64    `json:"productAttributableFailureRate"`
}

type ResultDecision struct {
	State          string   `json:"state"`
	PromotionClass string   `json:"promotionClass"`
	EvaluatedAt    string   `json:"evaluatedAt"`
	Reasons        []string `json:"reasons"`
}

type pairedPilotObservation struct {
	control   PilotObservation
	candidate PilotObservation
}

// NewPilotAssignment creates one prescribed paired assignment. Odd pairs run
// control first and even pairs run candidate first.
func NewPilotAssignment(
	definition PilotDefinition,
	pairIndex int,
	arm string,
	assignedAt time.Time,
) (PilotAssignment, error) {
	order := 1
	expectedFirst := "CONTROL"
	if pairIndex%2 == 0 {
		expectedFirst = "CANDIDATE"
	}
	if arm != expectedFirst {
		order = 2
	}
	cacheState := "DISABLED"
	commandClass := "buildopt-gradle-control-v1"
	if arm == "CANDIDATE" {
		cacheState = "WARM_MANAGED_L1"
		commandClass = "buildopt-gradle-managed-l1-candidate-v1"
	}
	assignment := PilotAssignment{
		SchemaVersion:            pilotSchemaVersion,
		RecordType:               pilotAssignmentType,
		ExperimentID:             definition.ExperimentID,
		MeasurementEpoch:         definition.MeasurementEpoch,
		PairIndex:                pairIndex,
		OrderInPair:              order,
		Arm:                      arm,
		AssignmentProbability:    1,
		AssignedAt:               assignedAt.UTC().Format(time.RFC3339Nano),
		EffectScope:              "ACTION_INCREMENTAL",
		ActionID:                 definition.ActionID,
		BaselineDefinitionDigest: definition.BaselineDefinitionDigest,
		ControlDefinitionDigest:  definition.ControlDefinitionDigest,
		MetricDefinitionVersion:  "build-impact-v1",
		MeasurementPolicyVersion: "beta-measurement-v1",
		CohortID:                 definition.CohortID,
		AnalysisUnit:             "BUILD_SESSION",
		AssignmentUnit:           "CI_JOB",
		Estimand:                 "SUCCESSFUL_BUILD_SESSION_LATENCY",
		Environment:              definition.Environment,
		PipelineClass:            definition.PipelineClass,
		RunnerClass:              definition.RunnerClass,
		OutcomeStratum:           "SUCCESS",
		WorkUnitsFingerprint:     definition.WorkUnitsFingerprint,
		WorkspaceState:           "ISOLATED_OUTPUT_REMOVED",
		DaemonState:              "SINGLE_USE",
		CacheState:               cacheState,
		CommandClass:             commandClass,
		RequiredDeliverable:      definition.RequiredDeliverable,
		ExclusionPolicy: []string{
			"PREDECLARED_INFRA_FAILURE",
			"MISSING_WORK_UNITS_FINGERPRINT",
			"CROSS_ARM_CONTAMINATION",
		},
	}
	if err := assignment.Validate(); err != nil {
		return PilotAssignment{}, err
	}
	return assignment, nil
}

func (assignment PilotAssignment) Validate() error {
	if assignment.SchemaVersion != pilotSchemaVersion ||
		assignment.RecordType != pilotAssignmentType ||
		!pilotIdentifierPattern.MatchString(assignment.ExperimentID) ||
		assignment.MeasurementEpoch < 1 ||
		assignment.PairIndex < 1 ||
		assignment.PairIndex > maxPilotPairs ||
		(assignment.OrderInPair != 1 && assignment.OrderInPair != 2) ||
		(assignment.Arm != "CONTROL" && assignment.Arm != "CANDIDATE") ||
		assignment.AssignmentProbability != 1 ||
		assignment.EffectScope != "ACTION_INCREMENTAL" ||
		!pilotIdentifierPattern.MatchString(assignment.ActionID) ||
		!validSHA256(assignment.BaselineDefinitionDigest) ||
		!validSHA256(assignment.ControlDefinitionDigest) ||
		assignment.MetricDefinitionVersion != "build-impact-v1" ||
		assignment.MeasurementPolicyVersion != "beta-measurement-v1" ||
		!pilotIdentifierPattern.MatchString(assignment.CohortID) ||
		assignment.AnalysisUnit != "BUILD_SESSION" ||
		assignment.AssignmentUnit != "CI_JOB" ||
		assignment.Estimand != "SUCCESSFUL_BUILD_SESSION_LATENCY" ||
		!pilotIdentifierPattern.MatchString(assignment.Environment) ||
		!pilotIdentifierPattern.MatchString(assignment.PipelineClass) ||
		!pilotIdentifierPattern.MatchString(assignment.RunnerClass) ||
		assignment.OutcomeStratum != "SUCCESS" ||
		!validTokenizedDigest(assignment.WorkUnitsFingerprint) ||
		assignment.WorkspaceState != "ISOLATED_OUTPUT_REMOVED" ||
		assignment.DaemonState != "SINGLE_USE" ||
		strings.TrimSpace(assignment.RequiredDeliverable) == "" ||
		len(assignment.RequiredDeliverable) > 1024 {
		return errors.New("invalid causal-pilot assignment")
	}
	if _, err := parseCanonicalTime(assignment.AssignedAt); err != nil {
		return errors.New("invalid causal-pilot assignedAt")
	}
	expectedFirst := "CONTROL"
	if assignment.PairIndex%2 == 0 {
		expectedFirst = "CANDIDATE"
	}
	expectedOrder := 2
	if assignment.Arm == expectedFirst {
		expectedOrder = 1
	}
	expectedCache := "DISABLED"
	expectedCommand := "buildopt-gradle-control-v1"
	if assignment.Arm == "CANDIDATE" {
		expectedCache = "WARM_MANAGED_L1"
		expectedCommand = "buildopt-gradle-managed-l1-candidate-v1"
	}
	if assignment.OrderInPair != expectedOrder ||
		assignment.CacheState != expectedCache ||
		assignment.CommandClass != expectedCommand ||
		!slices.Equal(assignment.ExclusionPolicy, []string{
			"PREDECLARED_INFRA_FAILURE",
			"MISSING_WORK_UNITS_FINGERPRINT",
			"CROSS_ARM_CONTAMINATION",
		}) {
		return errors.New("causal-pilot assignment policy drift")
	}
	return nil
}

// NewPilotObservation binds an outcome to its pre-existing assignment.
func NewPilotObservation(
	assignment PilotAssignment,
	startedAt time.Time,
	completedAt time.Time,
	outcome string,
	exitCode int,
	deliverableStatus string,
	deliverableSHA256 string,
	deliverableSize int64,
) (PilotObservation, error) {
	assignmentDigest, err := PilotAssignmentDigest(assignment)
	if err != nil {
		return PilotObservation{}, err
	}
	sessionDigest := sha256.Sum256([]byte(
		"buildopt-causal-pilot-session-v1\x00" + assignmentDigest,
	))
	observation := PilotObservation{
		SchemaVersion:       pilotSchemaVersion,
		RecordType:          pilotObservationType,
		BuildSessionID:      "pilot-session-" + hex.EncodeToString(sessionDigest[:16]),
		AssignmentDigest:    assignmentDigest,
		Assignment:          assignment,
		StartedAt:           startedAt.UTC().Format(time.RFC3339Nano),
		CompletedAt:         completedAt.UTC().Format(time.RFC3339Nano),
		DurationMs:          completedAt.Sub(startedAt).Milliseconds(),
		Outcome:             outcome,
		ExitCode:            exitCode,
		DeliverableStatus:   deliverableStatus,
		DeliverableSHA256:   deliverableSHA256,
		DeliverableSizeByte: deliverableSize,
	}
	if err := observation.Validate(); err != nil {
		return PilotObservation{}, err
	}
	return observation, nil
}

func (observation PilotObservation) Validate() error {
	if observation.SchemaVersion != pilotSchemaVersion ||
		observation.RecordType != pilotObservationType ||
		!pilotIdentifierPattern.MatchString(observation.BuildSessionID) ||
		observation.DurationMs < 0 {
		return errors.New("invalid causal-pilot observation")
	}
	if err := observation.Assignment.Validate(); err != nil {
		return err
	}
	expectedDigest, err := PilotAssignmentDigest(observation.Assignment)
	if err != nil || observation.AssignmentDigest != expectedDigest {
		return errors.New("causal-pilot observation assignment digest mismatch")
	}
	expectedSessionDigest := sha256.Sum256([]byte(
		"buildopt-causal-pilot-session-v1\x00" + expectedDigest,
	))
	if observation.BuildSessionID !=
		"pilot-session-"+hex.EncodeToString(expectedSessionDigest[:16]) {
		return errors.New("causal-pilot build-session identity mismatch")
	}
	assignedAt, _ := parseCanonicalTime(observation.Assignment.AssignedAt)
	startedAt, err := parseCanonicalTime(observation.StartedAt)
	if err != nil || startedAt.Before(assignedAt) {
		return errors.New("causal-pilot command started before assignment")
	}
	completedAt, err := parseCanonicalTime(observation.CompletedAt)
	if err != nil || completedAt.Before(startedAt) ||
		observation.DurationMs != completedAt.Sub(startedAt).Milliseconds() {
		return errors.New("invalid causal-pilot completion")
	}
	switch observation.Outcome {
	case "SUCCESS":
		if observation.ExitCode != 0 ||
			observation.DeliverableStatus != "AVAILABLE" ||
			!validSHA256(observation.DeliverableSHA256) ||
			observation.DeliverableSizeByte <= 0 {
			return errors.New("successful pilot outcome lacks its deliverable")
		}
	case "BUILD_FAILURE", "CANCELLED":
		if observation.ExitCode < 1 || observation.ExitCode > 255 {
			return errors.New("failed pilot outcome has invalid exit code")
		}
	case "INFRA_FAILURE":
		if observation.ExitCode < 0 || observation.ExitCode > 255 {
			return errors.New("infrastructure outcome has invalid exit code")
		}
	default:
		return errors.New("unsupported causal-pilot outcome")
	}
	if observation.Outcome != "SUCCESS" {
		switch observation.DeliverableStatus {
		case "AVAILABLE":
			if !validSHA256(observation.DeliverableSHA256) ||
				observation.DeliverableSizeByte <= 0 {
				return errors.New("invalid failed-outcome deliverable")
			}
		case "NOT_AVAILABLE":
			if observation.DeliverableSHA256 != "" ||
				observation.DeliverableSizeByte != 0 {
				return errors.New("unavailable deliverable contains facts")
			}
		default:
			return errors.New("invalid failed-outcome deliverable status")
		}
	}
	return nil
}

// PilotAssignmentDigest returns the exact SHA-256 binding used by observations.
func PilotAssignmentDigest(assignment PilotAssignment) (string, error) {
	if err := assignment.Validate(); err != nil {
		return "", err
	}
	content, err := json.Marshal(assignment)
	if err != nil {
		return "", errors.New("encode causal-pilot assignment")
	}
	return digestBytes(content), nil
}

// BuildPilotResult aggregates comparable paired outcomes after both arms have
// completed. The result remains PRELIMINARY regardless of observed benefit.
func BuildPilotResult(
	observations []PilotObservation,
	incrementalActionOverheadMs int64,
	evaluatedAt time.Time,
) (ExperimentResult, error) {
	if incrementalActionOverheadMs < 0 {
		return ExperimentResult{}, errors.New(
			"incremental action overhead must be non-negative",
		)
	}
	pairs, first, err := pairPilotObservations(observations)
	if err != nil {
		return ExperimentResult{}, err
	}
	successPairs := make([]pairedPilotObservation, 0, len(pairs))
	outcomes := ResultOutcomesByArm{}
	divergentDeliverables := 0
	for _, pair := range pairs {
		incrementOutcome(&outcomes.Control, pair.control.Outcome)
		incrementOutcome(&outcomes.Candidate, pair.candidate.Outcome)
		if pair.control.Outcome == "SUCCESS" &&
			pair.candidate.Outcome == "SUCCESS" {
			successPairs = append(successPairs, pair)
			if pair.control.DeliverableSHA256 !=
				pair.candidate.DeliverableSHA256 ||
				pair.control.DeliverableSizeByte !=
					pair.candidate.DeliverableSizeByte {
				divergentDeliverables++
			}
		}
	}
	if len(successPairs) < 2 {
		return ExperimentResult{}, errors.New(
			"causal pilot requires at least two successful complete pairs",
		)
	}

	assigned := ResultArmCounts{Candidate: len(pairs), Control: len(pairs)}
	analyzed := ResultArmCounts{
		Candidate: len(successPairs),
		Control:   len(successPairs),
	}
	excluded := 2 * (len(pairs) - len(successPairs))
	exclusions := []ResultExclusion{}
	if excluded > 0 {
		exclusions = append(exclusions, ResultExclusion{
			Reason: "OTHER_PREDECLARED",
			Count:  excluded,
		})
	}
	effects := calculatePilotEffects(
		successPairs,
		assigned,
		outcomes,
		incrementalActionOverheadMs,
		divergentDeliverables,
	)
	windowStart, windowEnd := pilotWindow(observations)
	asOf := evaluatedAt.UTC()
	if asOf.Before(windowEnd) {
		return ExperimentResult{}, errors.New(
			"causal-pilot evaluation precedes the experiment window",
		)
	}
	result := ExperimentResult{
		SchemaVersion: pilotSchemaVersion,
		RecordType:    experimentResultType,
		ExperimentID:  first.ExperimentID,
		ResultVersion: 1,
		Status:        "PRELIMINARY",
		AsOf:          asOf.Format(time.RFC3339Nano),
		Window: ResultWindow{
			StartedAt: windowStart.Format(time.RFC3339Nano),
			EndedAt:   windowEnd.Format(time.RFC3339Nano),
		},
		EffectScope:              first.EffectScope,
		ActionIDs:                []string{first.ActionID},
		MeasurementEpoch:         first.MeasurementEpoch,
		BaselineDefinitionDigest: first.BaselineDefinitionDigest,
		ControlDefinitionDigest:  first.ControlDefinitionDigest,
		MetricDefinitionVersion:  first.MetricDefinitionVersion,
		MeasurementPolicyVersion: first.MeasurementPolicyVersion,
		Population: ResultPopulation{
			CohortID:       first.CohortID,
			AnalysisUnit:   first.AnalysisUnit,
			AssignmentUnit: first.AssignmentUnit,
			Estimand:       first.Estimand,
			RequiredStrata: []string{
				"environment",
				"pipelineClass",
				"runnerClass",
				"outcome",
				"workUnitsFingerprint",
				"measurementEpoch",
			},
		},
		Samples: ResultSamples{
			Assigned:                 assigned,
			Analyzed:                 analyzed,
			Outcomes:                 outcomes,
			ExcludedSampleSize:       excluded,
			Exclusions:               exclusions,
			MeasurementCoverageRatio: float64(2*len(successPairs)) / float64(2*len(pairs)),
		},
		Method: ResultMethod{
			AnalysisMethod:  "PAIRED_BOOTSTRAP",
			EffectStatistic: "PAIRED_MEAN_DELTA",
			IntervalType:    "CONFIDENCE_95",
		},
		Effects: effects,
		Decision: ResultDecision{
			State:          "PRELIMINARY",
			PromotionClass: "DIRECT_REVERSIBLE",
			EvaluatedAt:    asOf.Format(time.RFC3339Nano),
			Reasons: []string{
				"The internal sample is below the beta promotion minimum",
				"Feedback and CI queue effects are unavailable in the local pilot",
				"A PRELIMINARY result cannot authorize promotion",
			},
		},
	}
	if err := result.Validate(); err != nil {
		return ExperimentResult{}, err
	}
	return result, nil
}

// ValidatePilotResult recomputes the immutable aggregate from its raw inputs.
func ValidatePilotResult(
	result ExperimentResult,
	observations []PilotObservation,
) error {
	if err := result.Validate(); err != nil {
		return err
	}
	evaluatedAt, _ := parseCanonicalTime(result.Decision.EvaluatedAt)
	expected, err := BuildPilotResult(
		observations,
		result.Effects.IncrementalActionOverheadMs,
		evaluatedAt,
	)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(result, expected) {
		return errors.New(
			"EXPERIMENT_RESULT does not reconcile with pilot observations",
		)
	}
	return nil
}

func (result ExperimentResult) Validate() error {
	if result.SchemaVersion != pilotSchemaVersion ||
		result.RecordType != experimentResultType ||
		!pilotIdentifierPattern.MatchString(result.ExperimentID) ||
		result.ResultVersion != 1 ||
		result.Status != "PRELIMINARY" ||
		result.EffectScope != "ACTION_INCREMENTAL" ||
		len(result.ActionIDs) != 1 ||
		!pilotIdentifierPattern.MatchString(result.ActionIDs[0]) ||
		result.MeasurementEpoch < 1 ||
		!validSHA256(result.BaselineDefinitionDigest) ||
		!validSHA256(result.ControlDefinitionDigest) ||
		result.MetricDefinitionVersion != "build-impact-v1" ||
		result.MeasurementPolicyVersion != "beta-measurement-v1" {
		return errors.New("unsupported causal-pilot EXPERIMENT_RESULT")
	}
	asOf, err := parseCanonicalTime(result.AsOf)
	if err != nil {
		return errors.New("invalid EXPERIMENT_RESULT asOf")
	}
	startedAt, err := parseCanonicalTime(result.Window.StartedAt)
	if err != nil {
		return errors.New("invalid EXPERIMENT_RESULT window start")
	}
	endedAt, err := parseCanonicalTime(result.Window.EndedAt)
	if err != nil || !startedAt.Before(endedAt) || endedAt.After(asOf) {
		return errors.New("invalid EXPERIMENT_RESULT window")
	}
	if result.Population.AnalysisUnit != "BUILD_SESSION" ||
		result.Population.AssignmentUnit != "CI_JOB" ||
		result.Population.Estimand != "SUCCESSFUL_BUILD_SESSION_LATENCY" ||
		!pilotIdentifierPattern.MatchString(result.Population.CohortID) ||
		!slices.Equal(result.Population.RequiredStrata, []string{
			"environment",
			"pipelineClass",
			"runnerClass",
			"outcome",
			"workUnitsFingerprint",
			"measurementEpoch",
		}) {
		return errors.New("invalid EXPERIMENT_RESULT population")
	}
	if result.Method != (ResultMethod{
		AnalysisMethod:  "PAIRED_BOOTSTRAP",
		EffectStatistic: "PAIRED_MEAN_DELTA",
		IntervalType:    "CONFIDENCE_95",
	}) {
		return errors.New("invalid EXPERIMENT_RESULT method")
	}
	if result.Samples.Assigned.Candidate < 2 ||
		result.Samples.Assigned.Candidate != result.Samples.Assigned.Control ||
		result.Samples.Analyzed.Candidate < 2 ||
		result.Samples.Analyzed.Candidate != result.Samples.Analyzed.Control ||
		result.Samples.Analyzed.Candidate > result.Samples.Assigned.Candidate ||
		result.Samples.ExcludedSampleSize !=
			2*(result.Samples.Assigned.Candidate-result.Samples.Analyzed.Candidate) ||
		result.Samples.MeasurementCoverageRatio !=
			float64(2*result.Samples.Analyzed.Candidate)/
				float64(2*result.Samples.Assigned.Candidate) {
		return errors.New("invalid EXPERIMENT_RESULT samples")
	}
	if result.Samples.Outcomes.Candidate.total() !=
		result.Samples.Assigned.Candidate ||
		result.Samples.Outcomes.Control.total() !=
			result.Samples.Assigned.Control {
		return errors.New("EXPERIMENT_RESULT outcomes do not reconcile")
	}
	exclusionTotal := 0
	for _, exclusion := range result.Samples.Exclusions {
		if exclusion.Reason != "OTHER_PREDECLARED" || exclusion.Count < 1 {
			return errors.New("invalid EXPERIMENT_RESULT exclusion")
		}
		exclusionTotal += exclusion.Count
	}
	if exclusionTotal != result.Samples.ExcludedSampleSize {
		return errors.New("EXPERIMENT_RESULT exclusions do not reconcile")
	}
	if result.Effects.IncrementalActionOverheadMs < 0 ||
		result.Effects.RequiredDeliverableDivergenceRate < 0 ||
		result.Effects.RequiredDeliverableDivergenceRate > 1 ||
		result.Effects.ProductAttributableFailureRate < 0 ||
		result.Effects.ProductAttributableFailureRate > 1 ||
		!containsInt64(
			result.Effects.ObservedNetBuildTimeSavedInterval95Ms,
			result.Effects.ObservedNetBuildTimeSavedMs,
		) ||
		!containsFloat64(
			result.Effects.ObservedBuildTimeReductionInterval95,
			result.Effects.ObservedBuildTimeReductionRatio,
		) ||
		!containsInt64(
			result.Effects.CustomerVisibleBuildP95Interval95Ms,
			result.Effects.CustomerVisibleBuildP95DeltaMs,
		) {
		return errors.New("invalid EXPERIMENT_RESULT effects")
	}
	evaluatedAt, err := parseCanonicalTime(result.Decision.EvaluatedAt)
	if err != nil || !evaluatedAt.Equal(asOf) ||
		result.Decision.State != "PRELIMINARY" ||
		result.Decision.PromotionClass != "DIRECT_REVERSIBLE" ||
		!slices.Equal(result.Decision.Reasons, []string{
			"The internal sample is below the beta promotion minimum",
			"Feedback and CI queue effects are unavailable in the local pilot",
			"A PRELIMINARY result cannot authorize promotion",
		}) {
		return errors.New("invalid EXPERIMENT_RESULT decision")
	}
	return nil
}

func (counts ResultOutcomeCounts) total() int {
	return counts.Success + counts.BuildFailure + counts.InfraFailure +
		counts.Cancelled
}

// DemonstratesNetCausalSavings is the narrow A0-G09 check. It does not promote
// the result or claim that beta sample/tail gates have passed.
func (result ExperimentResult) DemonstratesNetCausalSavings() bool {
	return result.Validate() == nil &&
		result.Samples.Analyzed == result.Samples.Assigned &&
		result.Effects.ObservedNetBuildTimeSavedMs > 0 &&
		result.Effects.ObservedNetBuildTimeSavedInterval95Ms[0] > 0 &&
		result.Effects.RequiredDeliverableDivergenceRate == 0 &&
		result.Effects.ProductAttributableFailureRate == 0 &&
		result.Effects.BuildFailureRateDelta == 0
}

func pairPilotObservations(
	observations []PilotObservation,
) ([]pairedPilotObservation, PilotAssignment, error) {
	if len(observations) < 4 || len(observations)%2 != 0 ||
		len(observations) > 2*maxPilotPairs {
		return nil, PilotAssignment{}, errors.New(
			"causal pilot observations must form 2 to 256 complete pairs",
		)
	}
	seenAssignments := make(map[string]struct{}, len(observations))
	byPair := make(map[int]*pairedPilotObservation)
	var first PilotAssignment
	for index, observation := range observations {
		if err := observation.Validate(); err != nil {
			return nil, PilotAssignment{}, err
		}
		assignment := observation.Assignment
		if index == 0 {
			first = assignment
		} else if !samePilotStratum(first, assignment) {
			return nil, PilotAssignment{}, errors.New(
				"causal-pilot assignment stratum drift",
			)
		}
		if _, exists := seenAssignments[observation.AssignmentDigest]; exists {
			return nil, PilotAssignment{}, errors.New(
				"causal-pilot assignment was observed more than once",
			)
		}
		seenAssignments[observation.AssignmentDigest] = struct{}{}
		pair := byPair[assignment.PairIndex]
		if pair == nil {
			pair = &pairedPilotObservation{}
			byPair[assignment.PairIndex] = pair
		}
		if assignment.Arm == "CONTROL" {
			if pair.control.RecordType != "" {
				return nil, PilotAssignment{}, errors.New(
					"causal-pilot pair repeats CONTROL",
				)
			}
			pair.control = observation
		} else {
			if pair.candidate.RecordType != "" {
				return nil, PilotAssignment{}, errors.New(
					"causal-pilot pair repeats CANDIDATE",
				)
			}
			pair.candidate = observation
		}
	}
	pairs := make([]pairedPilotObservation, len(byPair))
	for pairIndex := 1; pairIndex <= len(byPair); pairIndex++ {
		pair := byPair[pairIndex]
		if pair == nil ||
			pair.control.RecordType == "" ||
			pair.candidate.RecordType == "" {
			return nil, PilotAssignment{}, fmt.Errorf(
				"causal-pilot pair %d is incomplete",
				pairIndex,
			)
		}
		firstObservation := pair.control
		secondObservation := pair.candidate
		if pair.candidate.Assignment.OrderInPair == 1 {
			firstObservation = pair.candidate
			secondObservation = pair.control
		}
		firstCompleted, _ := parseCanonicalTime(firstObservation.CompletedAt)
		secondStarted, _ := parseCanonicalTime(secondObservation.StartedAt)
		if firstCompleted.After(secondStarted) {
			return nil, PilotAssignment{}, fmt.Errorf(
				"causal-pilot pair %d arms overlap or violate assigned order",
				pairIndex,
			)
		}
		pairs[pairIndex-1] = *pair
	}
	return pairs, first, nil
}

func samePilotStratum(left, right PilotAssignment) bool {
	return left.ExperimentID == right.ExperimentID &&
		left.MeasurementEpoch == right.MeasurementEpoch &&
		left.EffectScope == right.EffectScope &&
		left.ActionID == right.ActionID &&
		left.BaselineDefinitionDigest == right.BaselineDefinitionDigest &&
		left.ControlDefinitionDigest == right.ControlDefinitionDigest &&
		left.MetricDefinitionVersion == right.MetricDefinitionVersion &&
		left.MeasurementPolicyVersion == right.MeasurementPolicyVersion &&
		left.CohortID == right.CohortID &&
		left.AnalysisUnit == right.AnalysisUnit &&
		left.AssignmentUnit == right.AssignmentUnit &&
		left.Estimand == right.Estimand &&
		left.Environment == right.Environment &&
		left.PipelineClass == right.PipelineClass &&
		left.RunnerClass == right.RunnerClass &&
		left.OutcomeStratum == right.OutcomeStratum &&
		left.WorkUnitsFingerprint == right.WorkUnitsFingerprint &&
		left.WorkspaceState == right.WorkspaceState &&
		left.DaemonState == right.DaemonState &&
		left.RequiredDeliverable == right.RequiredDeliverable &&
		slices.Equal(left.ExclusionPolicy, right.ExclusionPolicy)
}

func calculatePilotEffects(
	pairs []pairedPilotObservation,
	assigned ResultArmCounts,
	outcomes ResultOutcomesByArm,
	incrementalActionOverheadMs int64,
	divergentDeliverables int,
) ResultEffects {
	control := make([]float64, len(pairs))
	candidate := make([]float64, len(pairs))
	savings := make([]float64, len(pairs))
	var controlTotal, savingsTotal float64
	for index, pair := range pairs {
		control[index] = float64(pair.control.DurationMs)
		candidate[index] = float64(pair.candidate.DurationMs)
		savings[index] = control[index] - candidate[index]
		controlTotal += control[index]
		savingsTotal += savings[index]
	}
	meanSavings := savingsTotal / float64(len(pairs))
	reductionRatio := savingsTotal / controlTotal
	controlSorted := slices.Clone(control)
	candidateSorted := slices.Clone(candidate)
	slices.Sort(controlSorted)
	slices.Sort(candidateSorted)
	p95Delta := nearestRank(candidateSorted, 0.95) -
		nearestRank(controlSorted, 0.95)
	savingsInterval, ratioInterval, p95Interval := bootstrapPilotIntervals(
		pairs,
	)
	controlFailureRate := float64(
		outcomes.Control.BuildFailure,
	) / float64(assigned.Control)
	candidateFailureRate := float64(
		outcomes.Candidate.BuildFailure,
	) / float64(assigned.Candidate)
	candidateProductFailureRate := float64(
		outcomes.Candidate.BuildFailure+outcomes.Candidate.Cancelled,
	) / float64(assigned.Candidate)
	return ResultEffects{
		ObservedNetBuildTimeSavedMs: int64(math.Round(meanSavings)),
		ObservedNetBuildTimeSavedInterval95Ms: [2]int64{
			int64(math.Floor(savingsInterval[0])),
			int64(math.Ceil(savingsInterval[1])),
		},
		ObservedBuildTimeReductionRatio:      reductionRatio,
		ObservedBuildTimeReductionInterval95: ratioInterval,
		BuildFailureRateDelta: candidateFailureRate -
			controlFailureRate,
		CustomerVisibleBuildP95DeltaMs: int64(math.Round(p95Delta)),
		CustomerVisibleBuildP95Interval95Ms: [2]int64{
			int64(math.Floor(p95Interval[0])),
			int64(math.Ceil(p95Interval[1])),
		},
		IncrementalActionOverheadMs: incrementalActionOverheadMs,
		RequiredDeliverableDivergenceRate: float64(divergentDeliverables) /
			float64(len(pairs)),
		ProductAttributableFailureRate: candidateProductFailureRate,
	}
}

func bootstrapPilotIntervals(
	pairs []pairedPilotObservation,
) ([2]float64, [2]float64, [2]float64) {
	meanSavings := make([]float64, pilotBootstrapReplicates)
	reductionRatios := make([]float64, pilotBootstrapReplicates)
	p95Deltas := make([]float64, pilotBootstrapReplicates)
	seedHash := sha256.New()
	_, _ = seedHash.Write([]byte("buildopt-causal-pilot-bootstrap-v1"))
	for _, pair := range pairs {
		_, _ = seedHash.Write([]byte{0})
		_, _ = seedHash.Write([]byte(pair.control.AssignmentDigest))
		_, _ = seedHash.Write([]byte{0})
		_, _ = seedHash.Write([]byte(pair.candidate.AssignmentDigest))
	}
	seed := uint64(0)
	for _, value := range seedHash.Sum(nil)[:8] {
		seed = seed<<8 | uint64(value)
	}
	random := splitMix64{state: seed}
	for replicate := 0; replicate < pilotBootstrapReplicates; replicate++ {
		control := make([]float64, len(pairs))
		candidate := make([]float64, len(pairs))
		var controlTotal, savingsTotal float64
		for index := range pairs {
			selected := pairs[random.nextIndex(len(pairs))]
			control[index] = float64(selected.control.DurationMs)
			candidate[index] = float64(selected.candidate.DurationMs)
			controlTotal += control[index]
			savingsTotal += control[index] - candidate[index]
		}
		slices.Sort(control)
		slices.Sort(candidate)
		meanSavings[replicate] = savingsTotal / float64(len(pairs))
		reductionRatios[replicate] = savingsTotal / controlTotal
		p95Deltas[replicate] = nearestRank(candidate, 0.95) -
			nearestRank(control, 0.95)
	}
	return percentile95(meanSavings),
		percentile95(reductionRatios),
		percentile95(p95Deltas)
}

type splitMix64 struct {
	state uint64
}

func (random *splitMix64) nextIndex(bound int) int {
	random.state += 0x9e3779b97f4a7c15
	value := random.state
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	value ^= value >> 31
	return int(value % uint64(bound))
}

func percentile95(values []float64) [2]float64 {
	slices.Sort(values)
	lower := int(math.Floor(0.025 * float64(len(values)-1)))
	upper := int(math.Ceil(0.975 * float64(len(values)-1)))
	return [2]float64{values[lower], values[upper]}
}

func incrementOutcome(counts *ResultOutcomeCounts, outcome string) {
	switch outcome {
	case "SUCCESS":
		counts.Success++
	case "BUILD_FAILURE":
		counts.BuildFailure++
	case "INFRA_FAILURE":
		counts.InfraFailure++
	case "CANCELLED":
		counts.Cancelled++
	}
}

func pilotWindow(observations []PilotObservation) (time.Time, time.Time) {
	start, _ := parseCanonicalTime(observations[0].StartedAt)
	end, _ := parseCanonicalTime(observations[0].CompletedAt)
	for _, observation := range observations[1:] {
		currentStart, _ := parseCanonicalTime(observation.StartedAt)
		currentEnd, _ := parseCanonicalTime(observation.CompletedAt)
		if currentStart.Before(start) {
			start = currentStart
		}
		if currentEnd.After(end) {
			end = currentEnd
		}
	}
	return start, end
}

func containsInt64(interval [2]int64, estimate int64) bool {
	return interval[0] <= interval[1] &&
		estimate >= interval[0] &&
		estimate <= interval[1]
}

func containsFloat64(interval [2]float64, estimate float64) bool {
	return !math.IsNaN(estimate) &&
		!math.IsInf(estimate, 0) &&
		interval[0] <= interval[1] &&
		estimate >= interval[0] &&
		estimate <= interval[1]
}

func validTokenizedDigest(value string) bool {
	if !strings.HasPrefix(value, "hmac-sha256:") ||
		len(value) != len("hmac-sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(value[len("hmac-sha256:"):])
	return err == nil
}

// WritePilotAssignment publishes an immutable mode-0600 assignment.
func WritePilotAssignment(path string, assignment PilotAssignment) error {
	if err := assignment.Validate(); err != nil {
		return err
	}
	return writeImmutablePilotJSON(path, assignment, "causal-pilot assignment")
}

func LoadPilotAssignment(path string) (PilotAssignment, error) {
	var assignment PilotAssignment
	if err := loadJSON(path, &assignment); err != nil {
		return PilotAssignment{}, err
	}
	if err := assignment.Validate(); err != nil {
		return PilotAssignment{}, err
	}
	return assignment, nil
}

// WritePilotObservation publishes an immutable mode-0600 outcome.
func WritePilotObservation(path string, observation PilotObservation) error {
	if err := observation.Validate(); err != nil {
		return err
	}
	return writeImmutablePilotJSON(
		path,
		observation,
		"causal-pilot observation",
	)
}

func LoadPilotObservation(path string) (PilotObservation, error) {
	var observation PilotObservation
	if err := loadJSON(path, &observation); err != nil {
		return PilotObservation{}, err
	}
	if err := observation.Validate(); err != nil {
		return PilotObservation{}, err
	}
	return observation, nil
}

func LoadExperimentResult(path string) (ExperimentResult, error) {
	var result ExperimentResult
	if err := loadJSON(path, &result); err != nil {
		return ExperimentResult{}, err
	}
	if err := result.Validate(); err != nil {
		return ExperimentResult{}, err
	}
	return result, nil
}

// PublishPilotResult writes one immutable JSON document and appends its compact
// bytes once to the bounded private EXPERIMENT_RESULT JSONL stream.
func PublishPilotResult(
	directory string,
	result ExperimentResult,
) (string, string, error) {
	if err := result.Validate(); err != nil {
		return "", "", err
	}
	if err := ensurePrivatePilotDirectory(directory); err != nil {
		return "", "", err
	}
	document, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", "", errors.New("encode EXPERIMENT_RESULT JSON")
	}
	document = append(document, '\n')
	identity := sha256.Sum256([]byte(result.ExperimentID))
	filename := "experiment-result-" + hex.EncodeToString(identity[:]) +
		"-v" + strconv.Itoa(result.ResultVersion) + ".json"
	documentPath := filepath.Join(directory, filename)
	if err := publishImmutablePilotFile(
		directory,
		documentPath,
		document,
		"EXPERIMENT_RESULT",
	); err != nil {
		return "", "", err
	}
	line, err := json.Marshal(result)
	if err != nil {
		return "", "", errors.New("encode EXPERIMENT_RESULT JSONL")
	}
	line = append(line, '\n')
	streamPath := filepath.Join(directory, experimentResultStream)
	if err := appendPilotResultLine(streamPath, result, line); err != nil {
		return "", "", err
	}
	return documentPath, streamPath, nil
}

// WritePilotResultStream copies the exact validated durable JSONL bytes.
func WritePilotResultStream(directory string, writer io.Writer) error {
	if err := ensurePrivatePilotDirectory(directory); err != nil {
		return err
	}
	path := filepath.Join(directory, experimentResultStream)
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return errors.New("open EXPERIMENT_RESULT JSONL stream")
	}
	defer file.Close()
	info, err := validatePrivatePilotFile(file, maxPilotJSONLBytes)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxPilotJSONLBytes+1))
	if err != nil || int64(len(raw)) != info.Size() || len(raw) == 0 {
		return errors.New("read bounded EXPERIMENT_RESULT JSONL stream")
	}
	if _, err := decodePilotResultLines(raw); err != nil {
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		return errors.New("write EXPERIMENT_RESULT JSONL stream")
	}
	return nil
}

func writeImmutablePilotJSON(path string, value any, label string) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s", label)
	}
	content = append(content, '\n')
	directory := filepath.Dir(path)
	if err := ensurePrivatePilotDirectory(directory); err != nil {
		return err
	}
	return publishImmutablePilotFile(directory, path, content, label)
}

func ensurePrivatePilotDirectory(path string) error {
	if path == "" || !filepath.IsAbs(path) {
		return errors.New("causal-pilot output directory must be absolute")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return errors.New("create causal-pilot output directory")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return errors.New("inspect causal-pilot output directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode().Perm() != 0o700 ||
		stat.Uid != uint32(os.Geteuid()) {
		return errors.New(
			"causal-pilot output directory is not private and current-user owned",
		)
	}
	return nil
}

func publishImmutablePilotFile(
	directory string,
	target string,
	content []byte,
	label string,
) error {
	temporary, err := os.CreateTemp(directory, ".causal-pilot-*.tmp")
	if err != nil {
		return fmt.Errorf("create %s temporary file", label)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set %s permissions", label)
	}
	if _, err := temporary.Write(content); err != nil {
		return fmt.Errorf("write %s", label)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync %s", label)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s", label)
	}
	if err := os.Link(temporaryPath, target); err != nil {
		if errors.Is(err, fs.ErrExist) {
			identical, compareErr := identicalPrivatePilotFile(
				target,
				content,
			)
			if compareErr != nil {
				return compareErr
			}
			if identical {
				return nil
			}
			return fmt.Errorf("%s identity conflicts with immutable content", label)
		}
		return fmt.Errorf("publish %s", label)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove %s temporary file", label)
	}
	return syncPilotDirectory(directory)
}

func identicalPrivatePilotFile(
	path string,
	expected []byte,
) (bool, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return false, errors.New("open existing causal-pilot file")
	}
	defer file.Close()
	info, err := validatePrivatePilotFile(file, maxDocumentBytes)
	if err != nil {
		return false, err
	}
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentBytes+1))
	if err != nil || int64(len(content)) != info.Size() {
		return false, errors.New("read existing causal-pilot file")
	}
	return bytes.Equal(content, expected), nil
}

func appendPilotResultLine(
	path string,
	result ExperimentResult,
	line []byte,
) error {
	if len(line) > maxPilotJSONLLineBytes {
		return errors.New("EXPERIMENT_RESULT JSONL line exceeds 1 MiB")
	}
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_EXCL|os.O_RDWR|os.O_APPEND|syscall.O_NOFOLLOW,
		0o600,
	)
	created := err == nil
	if errors.Is(err, fs.ErrExist) {
		file, err = os.OpenFile(
			path,
			os.O_RDWR|os.O_APPEND|syscall.O_NOFOLLOW,
			0,
		)
	}
	if err != nil {
		return errors.New("open EXPERIMENT_RESULT JSONL stream")
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return errors.New("lock EXPERIMENT_RESULT JSONL stream")
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}()
	info, err := validatePrivatePilotFile(file, maxPilotJSONLBytes)
	if err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.New("seek EXPERIMENT_RESULT JSONL stream")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxPilotJSONLBytes+1))
	if err != nil || int64(len(raw)) != info.Size() {
		return errors.New("read EXPERIMENT_RESULT JSONL stream")
	}
	results, err := decodePilotResultLines(raw)
	if err != nil {
		return err
	}
	key := result.ExperimentID + "/" + strconv.Itoa(result.ResultVersion)
	if previous, exists := results[key]; exists {
		if bytes.Equal(previous, bytes.TrimSuffix(line, []byte{'\n'})) {
			return nil
		}
		return errors.New(
			"EXPERIMENT_RESULT identity was reused with different content",
		)
	}
	if info.Size()+int64(len(line)) > maxPilotJSONLBytes {
		return errors.New("EXPERIMENT_RESULT JSONL stream reached 64 MiB")
	}
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return errors.New("seek EXPERIMENT_RESULT JSONL append")
	}
	written, err := file.Write(line)
	if err != nil || written != len(line) {
		return errors.New("append complete EXPERIMENT_RESULT JSONL line")
	}
	if err := file.Sync(); err != nil {
		return errors.New("sync EXPERIMENT_RESULT JSONL stream")
	}
	if created {
		return syncPilotDirectory(filepath.Dir(path))
	}
	return nil
}

func decodePilotResultLines(raw []byte) (map[string][]byte, error) {
	results := make(map[string][]byte)
	if len(raw) != 0 && raw[len(raw)-1] != '\n' {
		return nil, errors.New(
			"EXPERIMENT_RESULT JSONL stream has a truncated final line",
		)
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), maxPilotJSONLLineBytes)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var result ExperimentResult
		if err := decodeStrictPilotJSON(line, &result); err != nil {
			return nil, errors.New("decode EXPERIMENT_RESULT JSONL line")
		}
		if err := result.Validate(); err != nil {
			return nil, err
		}
		key := result.ExperimentID + "/" + strconv.Itoa(result.ResultVersion)
		if previous, exists := results[key]; exists &&
			!bytes.Equal(previous, line) {
			return nil, errors.New(
				"EXPERIMENT_RESULT JSONL identity conflict",
			)
		}
		results[key] = line
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.New("scan EXPERIMENT_RESULT JSONL stream")
	}
	return results, nil
}

func decodeStrictPilotJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing content")
	}
	return nil
}

func validatePrivatePilotFile(
	file *os.File,
	maximum int64,
) (os.FileInfo, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, errors.New("inspect causal-pilot file")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		info.Size() < 0 ||
		info.Size() > maximum {
		return nil, errors.New(
			"causal-pilot file is not a bounded private regular file",
		)
	}
	return info, nil
}

func syncPilotDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return errors.New("open causal-pilot output directory")
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return errors.New("sync causal-pilot output directory")
	}
	return nil
}
