package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/historyadmission"
	"github.com/tonyredondo/buildopt/internal/ordinarylearning"
)

const (
	optimizePrequalificationSchemaVersion = "buildopt.poc/economic-prequalification/v1"
	optimizePrequalificationNotEvaluated  = "NOT_EVALUATED"
	optimizePrequalificationMeasure       = "MEASURE"
	optimizePrequalificationReject        = "REJECT"

	optimizePrequalificationReasonNoGraph       = "NO_VERIFIED_REUSABLE_GRAPH"
	optimizePrequalificationReasonSelected      = "EXISTING_PROFILE_SELECTED"
	optimizePrequalificationReasonNoReduction   = "CURRENT_GRAPH_HAS_NO_REDUCTION"
	optimizePrequalificationReasonHistory       = "RECENT_HISTORY_UNAVAILABLE"
	optimizePrequalificationReasonInsufficient  = "INSUFFICIENT_RECENT_MATCHING_CHANGES"
	optimizePrequalificationReasonMeasure       = "RECENT_MATCHING_CHANGES_SUPPORT_MEASUREMENT"
	optimizePrequalificationMaximumHistoryDepth = 64
)

// optimizeEconomicPrequalification is a bounded POC decision made from an
// already verified graph and Git history. It is not a performance claim: it
// only decides whether expensive discovery and calibration have enough
// recurrence evidence to be attempted.
type optimizeEconomicPrequalification struct {
	SchemaVersion          string   `json:"schemaVersion"`
	Decision               string   `json:"decision"`
	Reason                 string   `json:"reason"`
	EvidenceSource         string   `json:"evidenceSource"`
	Entrypoints            []string `json:"entrypoints"`
	ChangeFamily           string   `json:"changeFamily"`
	ChangedProjects        []string `json:"changedProjects"`
	GraphProjects          int      `json:"graphProjects"`
	SelectedProjects       int      `json:"selectedProjects"`
	OmittedProjects        int      `json:"omittedProjects"`
	HistoryWindowCommits   int      `json:"historyWindowCommits"`
	ObservedCommits        int      `json:"observedCommits"`
	AnalogousCommits       int      `json:"analogousCommits"`
	MinimumPaybackBuilds   int      `json:"minimumPaybackBuilds"`
	MaximumBreakEvenBuilds int      `json:"maximumBreakEvenBuilds"`
	EvaluationDurationNS   int64    `json:"evaluationDurationNs"`
	DiscoveryAuthorized    bool     `json:"discoveryAuthorized"`
	ProductionAuthorized   bool     `json:"productionAuthorized"`
	TestOptimization       string   `json:"testOptimization"`
}

func unevaluatedOptimizePrequalification(reason string) optimizeEconomicPrequalification {
	return optimizeEconomicPrequalification{
		SchemaVersion: optimizePrequalificationSchemaVersion,
		Decision:      optimizePrequalificationNotEvaluated, Reason: reason,
		Entrypoints: []string{}, ChangedProjects: []string{}, ProductionAuthorized: false,
		TestOptimization: "OUT_OF_SCOPE",
	}
}

func prequalifyOptimizeDiscovery(
	invocation optimizeInvocation,
	snapshot buildimpact.DiscoverySnapshot,
	expectedOwners []string,
	family string,
) (result optimizeEconomicPrequalification) {
	startedAt := time.Now()
	result = optimizeEconomicPrequalification{
		SchemaVersion:          optimizePrequalificationSchemaVersion,
		Decision:               optimizePrequalificationReject,
		Reason:                 optimizePrequalificationReasonHistory,
		EvidenceSource:         "VERIFIED_CENTRAL_GRAPH_AND_FIRST_PARENT_GIT_HISTORY",
		Entrypoints:            append([]string(nil), invocation.discovery.Entrypoints...),
		ChangeFamily:           family,
		ChangedProjects:        append([]string(nil), expectedOwners...),
		GraphProjects:          len(snapshot.Projects),
		MinimumPaybackBuilds:   ordinarylearning.MaximumPaybackMatches,
		MaximumBreakEvenBuilds: invocation.maxBreakEvenBuilds,
		ProductionAuthorized:   false, TestOptimization: "OUT_OF_SCOPE",
	}
	defer func() {
		result.EvaluationDurationNS = time.Since(startedAt).Nanoseconds()
		if result.EvaluationDurationNS < 1 {
			result.EvaluationDurationNS = 1
		}
	}()

	affected := optimizeAffectedProjects(snapshot, expectedOwners)
	result.SelectedProjects = len(affected)
	result.OmittedProjects = len(snapshot.Projects) - len(affected)
	if result.GraphProjects < 1 || result.SelectedProjects < 1 || result.OmittedProjects < 1 {
		result.Reason = optimizePrequalificationReasonNoReduction
		return result
	}
	if invocation.maxBreakEvenBuilds < ordinarylearning.MaximumPaybackMatches {
		result.Reason = optimizePrequalificationReasonInsufficient
		return result
	}

	window := min(invocation.maxBreakEvenBuilds, optimizePrequalificationMaximumHistoryDepth)
	commits, analogous, err := countOptimizeCompatibleCommits(
		invocation.repositoryRoot, invocation.discovery.TargetRevision, window,
		snapshot, expectedOwners, family,
	)
	if err != nil {
		return result
	}
	result.HistoryWindowCommits = window
	result.ObservedCommits = len(commits)
	result.AnalogousCommits = analogous
	if result.AnalogousCommits < result.MinimumPaybackBuilds {
		result.Reason = optimizePrequalificationReasonInsufficient
		return result
	}
	result.Decision = optimizePrequalificationMeasure
	result.Reason = optimizePrequalificationReasonMeasure
	result.DiscoveryAuthorized = true
	return result
}

// predictOptimizeCompatibleMatches estimates only whether the current change
// family recurs often enough to repay learning inside the fixed POC horizon.
// It uses no timing claim and starts no Gradle process. Direct durations,
// outputs, portability and volatility still come only from requested builds.
func predictOptimizeCompatibleMatches(invocation optimizeInvocation, discovery optimizeDiscoveryResult) (int, error) {
	directory := filepath.Join(invocation.repositoryRoot, filepath.FromSlash(invocation.stateRelative), "discovery")
	manifestRaw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return 0, err
	}
	manifest, err := buildimpact.ParseManifest(
		manifestRaw,
		discovery.RepositoryID,
		optimizePipelineClass(discovery.Entrypoints, discovery.ChangeSHA256),
	)
	if err != nil {
		return 0, err
	}
	graphRaw, err := os.ReadFile(filepath.Join(directory, "graph.json"))
	if err != nil {
		return 0, err
	}
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		return 0, err
	}
	snapshot := centralOptimizeDiscoverySnapshot(graph.Graph)
	_, compatible, err := countOptimizeCompatibleCommits(
		invocation.repositoryRoot, discovery.TargetRevision,
		optimizePrequalificationMaximumHistoryDepth, snapshot,
		discovery.ChangedProjects, discovery.ChangeFamily,
	)
	return compatible, err
}

func countOptimizeCompatibleCommits(
	repositoryRoot string,
	targetRevision string,
	maximum int,
	snapshot buildimpact.DiscoverySnapshot,
	expectedOwners []string,
	family string,
) ([]string, int, error) {
	commits, err := optimizeRecentFirstParentCommits(repositoryRoot, targetRevision, maximum)
	if err != nil {
		return nil, 0, err
	}
	compatible := 0
	for _, commit := range commits {
		paths, pathErr := optimizeCommitChangedPaths(repositoryRoot, commit)
		if pathErr != nil || optimizeUnsafeEconomicChange(paths) {
			continue
		}
		classification, classifyErr := historyadmission.Classify(snapshot, paths)
		if classifyErr == nil && equalOptimizeStrings(classification.Owners, expectedOwners) &&
			classification.Family == family {
			compatible++
		}
	}
	return commits, compatible, nil
}

func optimizeRecentFirstParentCommits(repositoryRoot, targetRevision string, maximum int) ([]string, error) {
	raw, err := gitOutput(
		repositoryRoot,
		"rev-list", "--first-parent", "--max-count="+strconv.Itoa(maximum), targetRevision,
	)
	if err != nil {
		return nil, err
	}
	commits := strings.Fields(raw)
	if len(commits) == 0 || len(commits) > maximum {
		return nil, errOptimizeHistoryUnavailable
	}
	for _, commit := range commits {
		if !validMeasurementRevision(commit) {
			return nil, errOptimizeHistoryUnavailable
		}
	}
	return commits, nil
}

func optimizeCommitChangedPaths(repositoryRoot, commit string) ([]string, error) {
	raw, err := gitOutput(
		repositoryRoot,
		"diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", "-z", commit, "--",
	)
	if err != nil {
		return nil, err
	}
	paths := nullDelimitedPaths(raw)
	if len(paths) == 0 || len(paths) > maximumImpactChangedPaths || !uniqueMeasurementStrings(paths) {
		return nil, errOptimizeHistoryUnavailable
	}
	for _, path := range paths {
		if !validObservedOutputPath(path) || strings.ContainsAny(path, "\r\n\x00") {
			return nil, errOptimizeHistoryUnavailable
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func optimizeUnsafeEconomicChange(paths []string) bool {
	return historyadmission.UnsafeStructuralChange(paths)
}

var errOptimizeHistoryUnavailable = errors.New("bounded Git history is unavailable")

func validOptimizePrequalification(result optimizeEconomicPrequalification) bool {
	if result.SchemaVersion != optimizePrequalificationSchemaVersion ||
		result.ProductionAuthorized || result.TestOptimization != "OUT_OF_SCOPE" ||
		result.EvaluationDurationNS < 0 || result.MinimumPaybackBuilds < 0 ||
		result.MaximumBreakEvenBuilds < 0 || result.HistoryWindowCommits < 0 ||
		result.HistoryWindowCommits > optimizePrequalificationMaximumHistoryDepth ||
		result.ObservedCommits < 0 || result.AnalogousCommits < 0 ||
		result.ObservedCommits > result.HistoryWindowCommits ||
		result.AnalogousCommits > result.ObservedCommits ||
		result.DiscoveryAuthorized != (result.Decision == optimizePrequalificationMeasure) {
		return false
	}
	switch result.Decision {
	case optimizePrequalificationNotEvaluated:
		return result.Reason != "" && result.EvaluationDurationNS == 0 &&
			len(result.Entrypoints) == 0 && len(result.ChangedProjects) == 0
	case optimizePrequalificationReject:
		validShape := result.EvidenceSource != "" && validOptimizeFamily(result.ChangeFamily) &&
			len(result.Entrypoints) > 0 && uniqueMeasurementStrings(result.Entrypoints) &&
			len(result.ChangedProjects) > 0 && uniqueMeasurementStrings(result.ChangedProjects) &&
			result.GraphProjects > 0 && result.SelectedProjects > 0 &&
			result.SelectedProjects+result.OmittedProjects == result.GraphProjects &&
			result.MinimumPaybackBuilds >= 2 && result.MaximumBreakEvenBuilds >= 1 &&
			result.EvaluationDurationNS > 0
		if !validShape {
			return false
		}
		switch result.Reason {
		case optimizePrequalificationReasonNoReduction:
			return result.OmittedProjects == 0
		case optimizePrequalificationReasonHistory:
			return result.HistoryWindowCommits == 0 && result.ObservedCommits == 0
		case optimizePrequalificationReasonInsufficient:
			return result.AnalogousCommits < result.MinimumPaybackBuilds
		default:
			return false
		}
	case optimizePrequalificationMeasure:
		return result.Reason == optimizePrequalificationReasonMeasure &&
			result.EvidenceSource != "" && validOptimizeFamily(result.ChangeFamily) &&
			len(result.Entrypoints) > 0 && uniqueMeasurementStrings(result.Entrypoints) &&
			len(result.ChangedProjects) > 0 && uniqueMeasurementStrings(result.ChangedProjects) &&
			result.GraphProjects > 0 && result.SelectedProjects > 0 && result.OmittedProjects > 0 &&
			result.SelectedProjects+result.OmittedProjects == result.GraphProjects &&
			result.MinimumPaybackBuilds >= 2 && result.MaximumBreakEvenBuilds >= 1 &&
			result.AnalogousCommits >= result.MinimumPaybackBuilds &&
			result.EvaluationDurationNS > 0
	default:
		return false
	}
}
