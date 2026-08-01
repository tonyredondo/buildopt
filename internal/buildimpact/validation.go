package buildimpact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"
)

const (
	ValidationObservationSchemaVersion = "buildopt.build-impact/validation-observation/v1"
	ValidationResultSchemaVersion      = "buildopt.build-impact/validation-result/v1"
	ValidationShadow                   = "SHADOW"
	ValidationPairedControl            = "PAIRED_CONTROL"
	ValidationShadowPassed             = "SHADOW_VALIDATED"
	ValidationControlPassed            = "CONTROL_PASSED"
	ValidationFalseNegative            = "FALSE_NEGATIVE"
	ValidationInconclusive             = "INCONCLUSIVE"
	RunSuccess                         = "SUCCESS"
	RunBuildFailure                    = "BUILD_FAILURE"
	RunInfrastructureFailure           = "INFRA_FAILURE"
	RunCancelled                       = "CANCELLED"
	maximumValidationObservationBytes  = 2 << 20
)

var (
	observationIDPattern = regexp.MustCompile(`^bia-[a-z0-9][a-z0-9-]{0,62}$`)
	revisionPattern      = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern        = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

type ValidationObservation struct {
	SchemaVersion  string          `json:"schemaVersion"`
	ObservationID  string          `json:"observationId"`
	RepositoryID   string          `json:"repositoryId"`
	PipelineClass  string          `json:"pipelineClass"`
	Revision       string          `json:"revision"`
	ManifestDigest string          `json:"manifestDigest"`
	GraphDigest    string          `json:"graphDigest"`
	AdapterVersion string          `json:"adapterVersion"`
	ChangeClass    string          `json:"changeClass"`
	ObservedAt     string          `json:"observedAt"`
	Mode           string          `json:"mode"`
	ChangedPaths   []string        `json:"changedPaths"`
	Baseline       RunObservation  `json:"baseline"`
	Candidate      *RunObservation `json:"candidate"`
}

type RunObservation struct {
	Outcome     string             `json:"outcome"`
	Entrypoints []string           `json:"entrypoints"`
	Projects    []string           `json:"projects"`
	Artifacts   []ObservedArtifact `json:"artifacts"`
	Checks      []ObservedCheck    `json:"checks"`
}

type ObservedArtifact struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Digest    string `json:"digest"`
	SizeBytes int64  `json:"sizeBytes"`
}

type ObservedCheck struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Outcome string `json:"outcome"`
}

type ValidationResult struct {
	SchemaVersion       string `json:"schemaVersion"`
	ObservationID       string `json:"observationId"`
	RepositoryID        string `json:"repositoryId"`
	PipelineClass       string `json:"pipelineClass"`
	Revision            string `json:"revision"`
	ManifestDigest      string `json:"manifestDigest"`
	GraphDigest         string `json:"graphDigest"`
	AdapterVersion      string `json:"adapterVersion"`
	ChangeClass         string `json:"changeClass"`
	ObservedAt          string `json:"observedAt"`
	Mode                string `json:"mode"`
	AlternativeID       string `json:"alternativeId,omitempty"`
	Outcome             string `json:"outcome"`
	Reason              string `json:"reason"`
	EligibleDecision    bool   `json:"eligibleDecision"`
	ValidationComplete  bool   `json:"validationComplete"`
	FullControl         bool   `json:"fullControl"`
	FalseNegative       bool   `json:"falseNegative"`
	SelectionAuthorized bool   `json:"selectionAuthorized"`
}

// ParseValidationObservation strictly loads an observation bound to the exact
// manifest, graph, and adapter that produced the decision.
func ParseValidationObservation(raw []byte, manifest LoadedManifest, graph LoadedGraph) (ValidationObservation, error) {
	if len(raw) == 0 || len(raw) > maximumValidationObservationBytes {
		return ValidationObservation{}, errors.New("validation observation size is invalid")
	}
	if err := requireValidationFields(raw); err != nil {
		return ValidationObservation{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var observation ValidationObservation
	if err := decoder.Decode(&observation); err != nil {
		return ValidationObservation{}, fmt.Errorf("decode validation observation: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ValidationObservation{}, errors.New("validation observation has trailing content")
	}
	if err := validateObservation(observation, manifest, graph); err != nil {
		return ValidationObservation{}, err
	}
	return defensiveObservation(observation), nil
}

// EvaluateValidation compares the full baseline with the shadow model or one
// isolated candidate. It never authorizes selection.
func EvaluateValidation(manifest LoadedManifest, graph LoadedGraph, observation ValidationObservation) ValidationResult {
	result := baseValidationResult(observation)
	if err := validateObservation(observation, manifest, graph); err != nil {
		result.Reason = "OBSERVATION_INVALID"
		return result
	}
	decision := EvaluateImpact(manifest, graph.Graph, observation.ChangedPaths)
	if decision.Mode != DecisionShadowAlternative {
		result.Reason = "DECISION_" + decision.Reason
		return result
	}
	result.AlternativeID = decision.PredictedAlternativeID
	result.EligibleDecision = true
	if !equalStrings(observation.Baseline.Entrypoints, manifest.Manifest.OriginalEntrypoints) {
		result.Reason = "BASELINE_NOT_ORIGINAL_ENTRYPOINTS"
		return result
	}
	if observation.Baseline.Outcome != RunSuccess {
		result.Reason = "BASELINE_NOT_SUCCESSFUL"
		return result
	}
	if reason := validateSuccessfulRun(manifest.Manifest, observation.Baseline, originalReach(graph.Graph, manifest.Manifest)); reason != "" {
		result.Reason = "BASELINE_" + reason
		return result
	}
	if observation.Mode == ValidationShadow {
		result.Outcome = ValidationShadowPassed
		result.Reason = "FULL_EXECUTION_VALIDATES_SHADOW_MODEL"
		result.ValidationComplete = true
		return result
	}
	result.FullControl = true
	if observation.Candidate == nil {
		result.Reason = "CANDIDATE_MISSING"
		return result
	}
	candidate := *observation.Candidate
	if !equalStrings(candidate.Entrypoints, decision.PredictedEntrypoints) {
		return falseNegativeResult(result, "CANDIDATE_ENTRYPOINT_DIVERGENCE")
	}
	switch candidate.Outcome {
	case RunInfrastructureFailure, RunCancelled:
		result.Reason = "CANDIDATE_INFRASTRUCTURE_INCONCLUSIVE"
		return result
	case RunBuildFailure:
		return falseNegativeResult(result, "CANDIDATE_BUILD_FAILURE")
	case RunSuccess:
	default:
		result.Reason = "CANDIDATE_OUTCOME_INVALID"
		return result
	}
	if reason := validateSuccessfulRun(manifest.Manifest, candidate, alternativeReach(graph.Graph, decision.PredictedEntrypoints)); reason != "" {
		return falseNegativeResult(result, "CANDIDATE_"+reason)
	}
	if reason := compareRuns(observation.Baseline, candidate); reason != "" {
		return falseNegativeResult(result, reason)
	}
	result.Outcome = ValidationControlPassed
	result.Reason = "PAIRED_CONTROL_MATCHED"
	result.ValidationComplete = true
	return result
}

func validateObservation(observation ValidationObservation, manifest LoadedManifest, graph LoadedGraph) error {
	if observation.SchemaVersion != ValidationObservationSchemaVersion || !observationIDPattern.MatchString(observation.ObservationID) {
		return errors.New("validation observation identity is invalid")
	}
	if observation.RepositoryID != manifest.Manifest.RepositoryID || observation.PipelineClass != manifest.Manifest.PipelineClass || observation.ManifestDigest != manifest.Digest || observation.GraphDigest != graph.Digest || observation.AdapterVersion != graph.Graph.AdapterVersion {
		return errors.New("validation observation binding does not match manifest and graph")
	}
	if !revisionPattern.MatchString(observation.Revision) || !idPattern.MatchString(observation.ChangeClass) {
		return errors.New("validation revision or change class is invalid")
	}
	parsedTime, err := time.Parse(time.RFC3339, observation.ObservedAt)
	if err != nil || parsedTime.Format(time.RFC3339) != observation.ObservedAt || parsedTime.Location() != time.UTC {
		return errors.New("validation observation time must be canonical UTC RFC3339")
	}
	if observation.Mode != ValidationShadow && observation.Mode != ValidationPairedControl {
		return errors.New("validation mode is invalid")
	}
	if (observation.Mode == ValidationShadow && observation.Candidate != nil) || (observation.Mode == ValidationPairedControl && observation.Candidate == nil) {
		return errors.New("validation candidate presence does not match mode")
	}
	if len(observation.ChangedPaths) == 0 || len(observation.ChangedPaths) > 4096 || !uniqueStrings(observation.ChangedPaths) {
		return errors.New("validation changed paths are invalid")
	}
	for _, changedPath := range observation.ChangedPaths {
		if !validRepositoryPath(changedPath) {
			return errors.New("validation changed path is unsafe")
		}
	}
	if err := validateRunShape(observation.Baseline, graph.Graph); err != nil {
		return fmt.Errorf("invalid baseline observation: %w", err)
	}
	if observation.Candidate != nil {
		if err := validateRunShape(*observation.Candidate, graph.Graph); err != nil {
			return fmt.Errorf("invalid candidate observation: %w", err)
		}
	}
	return nil
}

func validateRunShape(run RunObservation, graph DeclaredGraph) error {
	if run.Outcome != RunSuccess && run.Outcome != RunBuildFailure && run.Outcome != RunInfrastructureFailure && run.Outcome != RunCancelled {
		return errors.New("run outcome is invalid")
	}
	if len(run.Entrypoints) == 0 || !uniqueStrings(run.Entrypoints) || !uniqueStrings(run.Projects) {
		return errors.New("run entrypoints or projects are invalid")
	}
	entrypointNames := map[string]bool{}
	projectNames := map[string]bool{}
	for _, entrypoint := range graph.Entrypoints {
		entrypointNames[entrypoint.Name] = true
	}
	for _, project := range graph.Projects {
		projectNames[project.Path] = true
	}
	for _, entrypoint := range run.Entrypoints {
		if !entrypointNames[entrypoint] {
			return errors.New("run references an unknown entrypoint")
		}
	}
	for _, project := range run.Projects {
		if !projectNames[project] {
			return errors.New("run references an unknown project")
		}
	}
	artifactIDs := map[string]bool{}
	for _, artifact := range run.Artifacts {
		if !idPattern.MatchString(artifact.ID) || artifactIDs[artifact.ID] || !validRepositoryPath(artifact.Path) || !sha256Pattern.MatchString(artifact.Digest) || artifact.SizeBytes <= 0 {
			return errors.New("run artifact is invalid")
		}
		artifactIDs[artifact.ID] = true
	}
	checkIDs := map[string]bool{}
	for _, check := range run.Checks {
		if !idPattern.MatchString(check.ID) || checkIDs[check.ID] || (check.Owner != BuildOptimization && check.Owner != TestOptimization) || (check.Outcome != "PASSED" && check.Outcome != "FAILED") {
			return errors.New("run check is invalid")
		}
		checkIDs[check.ID] = true
	}
	return nil
}

func validateSuccessfulRun(manifest Manifest, run RunObservation, expectedProjects map[string]bool) string {
	if !equalSet(sliceSet(run.Projects), expectedProjects) {
		return "PROJECT_DIVERGENCE"
	}
	expectedArtifacts := map[string]Artifact{}
	for _, artifact := range manifest.RequiredArtifacts {
		expectedArtifacts[artifact.ID] = artifact
	}
	if len(run.Artifacts) != len(expectedArtifacts) {
		return "ARTIFACT_SET_DIVERGENCE"
	}
	for _, observed := range run.Artifacts {
		expected, ok := expectedArtifacts[observed.ID]
		if !ok || !matchRepositoryGlob(expected.Path, observed.Path) {
			return "ARTIFACT_SET_DIVERGENCE"
		}
	}
	expectedChecks := map[string]Check{}
	for _, check := range manifest.RequiredChecks {
		expectedChecks[check.ID] = check
	}
	if len(run.Checks) != len(expectedChecks) {
		return "CHECK_SET_DIVERGENCE"
	}
	for _, observed := range run.Checks {
		expected, ok := expectedChecks[observed.ID]
		if !ok || observed.Owner != expected.Owner || observed.Outcome != "PASSED" {
			return "CHECK_SET_DIVERGENCE"
		}
	}
	return ""
}

func compareRuns(baseline, candidate RunObservation) string {
	baselineArtifacts := map[string]ObservedArtifact{}
	for _, artifact := range baseline.Artifacts {
		baselineArtifacts[artifact.ID] = artifact
	}
	for _, artifact := range candidate.Artifacts {
		baselineArtifact := baselineArtifacts[artifact.ID]
		if artifact.Path != baselineArtifact.Path || artifact.Digest != baselineArtifact.Digest || artifact.SizeBytes != baselineArtifact.SizeBytes {
			return "REQUIRED_ARTIFACT_DIVERGENCE"
		}
	}
	baselineChecks := map[string]ObservedCheck{}
	for _, check := range baseline.Checks {
		baselineChecks[check.ID] = check
	}
	for _, check := range candidate.Checks {
		baselineCheck := baselineChecks[check.ID]
		if check.Owner != baselineCheck.Owner || check.Outcome != baselineCheck.Outcome {
			return "REQUIRED_CHECK_DIVERGENCE"
		}
	}
	return ""
}

func originalReach(graph DeclaredGraph, manifest Manifest) map[string]bool {
	return alternativeReach(graph, manifest.OriginalEntrypoints)
}

func alternativeReach(graph DeclaredGraph, entrypoints []string) map[string]bool {
	wanted := sliceSet(entrypoints)
	result := map[string]bool{}
	for _, entrypoint := range graph.Entrypoints {
		if wanted[entrypoint.Name] {
			addAll(result, entrypoint.ReachesProjects)
		}
	}
	return result
}

func baseValidationResult(observation ValidationObservation) ValidationResult {
	return ValidationResult{
		SchemaVersion:       ValidationResultSchemaVersion,
		ObservationID:       observation.ObservationID,
		RepositoryID:        observation.RepositoryID,
		PipelineClass:       observation.PipelineClass,
		Revision:            observation.Revision,
		ManifestDigest:      observation.ManifestDigest,
		GraphDigest:         observation.GraphDigest,
		AdapterVersion:      observation.AdapterVersion,
		ChangeClass:         observation.ChangeClass,
		ObservedAt:          observation.ObservedAt,
		Mode:                observation.Mode,
		Outcome:             ValidationInconclusive,
		Reason:              "INCONCLUSIVE",
		SelectionAuthorized: false,
	}
}

func falseNegativeResult(result ValidationResult, reason string) ValidationResult {
	result.Outcome = ValidationFalseNegative
	result.Reason = reason
	result.ValidationComplete = true
	result.FalseNegative = true
	return result
}

func requireValidationFields(raw []byte) error {
	var presence struct {
		ChangedPaths *[]string `json:"changedPaths"`
		Baseline     *struct {
			Entrypoints *[]string           `json:"entrypoints"`
			Projects    *[]string           `json:"projects"`
			Artifacts   *[]ObservedArtifact `json:"artifacts"`
			Checks      *[]ObservedCheck    `json:"checks"`
		} `json:"baseline"`
		Candidate json.RawMessage `json:"candidate"`
	}
	if err := json.Unmarshal(raw, &presence); err != nil {
		return fmt.Errorf("inspect validation observation fields: %w", err)
	}
	if presence.ChangedPaths == nil || presence.Baseline == nil || presence.Baseline.Entrypoints == nil || presence.Baseline.Projects == nil || presence.Baseline.Artifacts == nil || presence.Baseline.Checks == nil || presence.Candidate == nil {
		return errors.New("validation observation must explicitly state changes, baseline collections, and candidate")
	}
	if !bytes.Equal(bytes.TrimSpace(presence.Candidate), []byte("null")) {
		var candidate struct {
			Entrypoints *[]string           `json:"entrypoints"`
			Projects    *[]string           `json:"projects"`
			Artifacts   *[]ObservedArtifact `json:"artifacts"`
			Checks      *[]ObservedCheck    `json:"checks"`
		}
		if err := json.Unmarshal(presence.Candidate, &candidate); err != nil || candidate.Entrypoints == nil || candidate.Projects == nil || candidate.Artifacts == nil || candidate.Checks == nil {
			return errors.New("validation candidate must explicitly state every collection")
		}
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sliceSet(values []string) map[string]bool {
	result := map[string]bool{}
	addAll(result, values)
	return result
}

func equalSet(left, right map[string]bool) bool {
	return len(left) == len(right) && containsAll(left, right)
}

func defensiveObservation(observation ValidationObservation) ValidationObservation {
	observation.ChangedPaths = append([]string(nil), observation.ChangedPaths...)
	observation.Baseline = defensiveRun(observation.Baseline)
	if observation.Candidate != nil {
		candidate := defensiveRun(*observation.Candidate)
		observation.Candidate = &candidate
	}
	return observation
}

func defensiveRun(run RunObservation) RunObservation {
	run.Entrypoints = append([]string(nil), run.Entrypoints...)
	run.Projects = append([]string(nil), run.Projects...)
	run.Artifacts = append([]ObservedArtifact(nil), run.Artifacts...)
	run.Checks = append([]ObservedCheck(nil), run.Checks...)
	sort.Strings(run.Projects)
	return run
}
