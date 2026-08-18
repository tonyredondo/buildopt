package launcher

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	optimizeSelectionSkipped        = "SKIPPED"
	optimizeSelectionRetained       = "NATIVE_RETAINED"
	optimizeSelectionSelected       = "SELECTED"
	optimizeSelectionReasonNone     = "NO_QUALIFIED_CHECKPOINT"
	optimizeSelectionReasonSelected = "QUALIFIED_PROFILE_SELECTED"
	optimizeSelectionReasonDrift    = "CHECKPOINT_BINDING_DRIFT"
	optimizeSelectionReasonInvalid  = "PROFILE_PORTFOLIO_INVALID"
	optimizeSelectionReasonNoFamily = "PROFILE_FAMILY_NOT_FOUND"
	optimizeSelectionReasonEvidence = "QUALIFICATION_EVIDENCE_DRIFT"
	optimizeSelectionReasonBindings = "PROFILE_BINDING_DRIFT"
	optimizeSelectionReasonPlan     = "PROFILE_PLAN_REJECTED"
)

var optimizeReplayBindingNames = []string{
	"BUILDOPT_EXECUTABLE",
	"CALIBRATION_EVIDENCE",
	"CHANGE_FAMILY",
	"DISCOVERY_DOCUMENTS",
	"GRADLE_OPTIONS",
	"GRADLE_WRAPPER_PROPERTIES",
	"PROFILE_ARTIFACTS",
	"PROFILE_PRECONDITIONS",
	"REPOSITORY_ID",
	"REPOSITORY_REVISION",
	"WORKFLOW_ENTRYPOINTS",
}

type optimizeSelectionResult struct {
	Status                        string   `json:"status"`
	Reason                        string   `json:"reason"`
	Performed                     bool     `json:"performed"`
	Selected                      bool     `json:"selected"`
	CompletedBeforeGradle         bool     `json:"completedBeforeGradle"`
	DurationNS                    int64    `json:"durationNs"`
	ChangeFamily                  string   `json:"changeFamily"`
	FamilySHA256                  string   `json:"familySha256"`
	ProfileSHA256                 string   `json:"profileSha256"`
	ProfileFile                   string   `json:"profileFile"`
	OriginalEntrypoints           []string `json:"originalEntrypoints"`
	SelectedEntrypoints           []string `json:"selectedEntrypoints"`
	ValidatedBindings             []string `json:"validatedBindings"`
	FailedBindings                []string `json:"failedBindings"`
	Source                        string   `json:"source,omitempty"`
	EvidenceRevision              string   `json:"evidenceRevision,omitempty"`
	RevalidatedRevision           string   `json:"revalidatedRevision,omitempty"`
	RemotePortfolioManifestSHA256 string   `json:"remotePortfolioManifestSha256,omitempty"`
	RemoteEvidenceManifestSHA256  string   `json:"remoteEvidenceManifestSha256,omitempty"`
	ProductionAuthorized          bool     `json:"productionAuthorized"`
	TestOptimization              string   `json:"testOptimization"`
}

func emptyOptimizeSelection(status, reason string, performed bool) optimizeSelectionResult {
	return optimizeSelectionResult{
		Status: status, Reason: reason, Performed: performed,
		CompletedBeforeGradle: true,
		OriginalEntrypoints:   []string{}, SelectedEntrypoints: []string{},
		ValidatedBindings: []string{}, FailedBindings: []string{},
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
}

func validOptimizeSelectionCheckpoint(state optimizeState) bool {
	selection := state.Selection
	if selection.ProductionAuthorized || selection.Selected != (selection.Status == optimizeSelectionSelected) {
		return false
	}
	if selection.Status == "" {
		return state.Phase == optimizePhaseUnseen && !state.BuildStarted &&
			selection.Reason == "" && selection.DurationNS == 0 &&
			selection.TestOptimization == ""
	}
	if selection.Reason == "" || selection.DurationNS < 1 ||
		!selection.CompletedBeforeGradle || selection.TestOptimization != "OUT_OF_SCOPE" {
		return false
	}
	switch selection.Status {
	case optimizeSelectionSkipped:
		return !selection.Performed && selection.Reason == optimizeSelectionReasonNone &&
			validEmptyOptimizeSelection(selection)
	case optimizeSelectionRetained:
		return selection.Performed && !selection.Selected && len(selection.FailedBindings) > 0 &&
			len(selection.ValidatedBindings) <= len(optimizeReplayBindingNames) &&
			len(selection.OriginalEntrypoints) == 0 && len(selection.SelectedEntrypoints) == 0
	case optimizeSelectionSelected:
		localSelection := (selection.Source == "" || selection.Source == optimizeSelectionSourceLocal) &&
			selection.Reason == optimizeSelectionReasonSelected &&
			selection.EvidenceRevision == "" && selection.RevalidatedRevision == "" &&
			selection.RemotePortfolioManifestSHA256 == "" && selection.RemoteEvidenceManifestSHA256 == "" &&
			equalOptimizeStrings(selection.ValidatedBindings, optimizeReplayBindingNames)
		centralSelection := selection.Source == optimizeSelectionSourceCentral &&
			selection.Reason == optimizeCentralReasonSelected &&
			validMeasurementRevision(selection.EvidenceRevision) &&
			validMeasurementRevision(selection.RevalidatedRevision) &&
			validOptimizeSHA(selection.RemotePortfolioManifestSHA256) &&
			validOptimizeSHA(selection.RemoteEvidenceManifestSHA256) &&
			equalOptimizeStrings(selection.ValidatedBindings, optimizeCentralReplayBindings)
		return selection.Performed && (localSelection || centralSelection) &&
			optimizeStringIn(state.Phase, "ACTIVE", "STALE") &&
			validOptimizeFamily(selection.ChangeFamily) &&
			validOptimizeSHA(selection.FamilySHA256) && validOptimizeSHA(selection.ProfileSHA256) &&
			validOptimizeGeneratedPath(selection.ProfileFile) &&
			len(selection.OriginalEntrypoints) > 0 && len(selection.SelectedEntrypoints) > 0 &&
			uniqueMeasurementStrings(selection.OriginalEntrypoints) &&
			uniqueMeasurementStrings(selection.SelectedEntrypoints) &&
			len(selection.FailedBindings) == 0
	default:
		return false
	}
}

func validEmptyOptimizeSelection(selection optimizeSelectionResult) bool {
	return selection.ChangeFamily == "" && selection.FamilySHA256 == "" &&
		selection.ProfileSHA256 == "" && selection.ProfileFile == "" &&
		len(selection.OriginalEntrypoints) == 0 && len(selection.SelectedEntrypoints) == 0 &&
		len(selection.ValidatedBindings) == 0 && len(selection.FailedBindings) == 0 &&
		selection.Source == "" && selection.EvidenceRevision == "" &&
		selection.RevalidatedRevision == "" && selection.RemotePortfolioManifestSHA256 == "" &&
		selection.RemoteEvidenceManifestSHA256 == ""
}

func (run *optimizeRun) prepareAutomaticReplay() (selected *impactInvocation) {
	startedAt := time.Now()
	selection := emptyOptimizeSelection(optimizeSelectionSkipped, optimizeSelectionReasonNone, false)
	defer func() {
		selection.DurationNS = time.Since(startedAt).Nanoseconds()
		if selection.DurationNS < 1 {
			selection.DurationNS = 1
		}
		run.selection = selection
	}()

	if !run.state.Resume.CheckpointFound {
		return nil
	}
	if !run.state.Resume.Accepted || run.previousState == nil {
		selection = retainedOptimizeSelection(optimizeSelectionReasonDrift, "CHECKPOINT")
		return nil
	}
	previous := run.previousState
	if !optimizeStringIn(previous.Phase, "QUALIFIED", "ACTIVE") || previous.LastExitCode != 0 ||
		previous.Calibration.Status != optimizeCalibrationComplete || !previous.Calibration.Qualified ||
		previous.Portfolio.Status != optimizePortfolioComplete {
		return nil
	}
	discovery := previous.Discovery
	calibration := previous.Calibration
	if err := validateOptimizeCalibrationEvidence(run.invocation, discovery, calibration); err != nil {
		selection = retainedOptimizeSelection(optimizeSelectionReasonEvidence, "CALIBRATION_EVIDENCE")
		return nil
	}
	indexPath := filepath.ToSlash(filepath.Join(run.invocation.stateRelative, "portfolio", optimizePortfolioIndexFile))
	portfolio, valid := loadOptimizePortfolio(
		run.invocation.repositoryRoot,
		indexPath,
		optimizePortfolioRepositoryScope(discovery.RepositoryID),
	)
	if !valid {
		selection = retainedOptimizeSelection(optimizeSelectionReasonInvalid, "PROFILE_ARTIFACTS")
		return nil
	}
	familySHA := optimizePortfolioFamilySHA(discovery)
	entry, found := findOptimizePortfolioEntry(portfolio.Profiles, familySHA)
	if !found {
		selection = retainedOptimizeSelection(optimizeSelectionReasonNoFamily, "CHANGE_FAMILY")
		return nil
	}
	failures := validateOptimizeReplayEntry(run.invocation, discovery, calibration, entry)
	if len(failures) > 0 {
		selection = retainedOptimizeSelection(optimizeSelectionReasonBindings, failures...)
		return nil
	}
	changesPath := filepath.ToSlash(filepath.Join(run.invocation.stateRelative, "discovery", "changes.txt"))
	arguments := []string{"--config", entry.ProfilePath, "--changes-file", changesPath}
	expectedOptions, _ := optimizeCalibrationGradleOptions(run.invocation.discovery.gradleOptions)
	for _, option := range expectedOptions {
		arguments = append(arguments, "--gradle-option="+option)
	}
	impact, err := prepareQualifiedPOCProfileInvocation(arguments, false)
	if err != nil || !impact.plan.CandidateSelected || impact.qualifiedProfile == nil ||
		!equalOptimizeStrings(impact.plan.Entrypoints, entry.CandidateEntrypoints) {
		selection = retainedOptimizeSelection(optimizeSelectionReasonPlan, "PROFILE_PLAN")
		return nil
	}
	selection = optimizeSelectionResult{
		Status: optimizeSelectionSelected, Reason: optimizeSelectionReasonSelected,
		Performed: true, Selected: true, CompletedBeforeGradle: true,
		ChangeFamily: entry.Family, FamilySHA256: entry.FamilySHA256,
		ProfileSHA256: entry.ProfileSHA256, ProfileFile: entry.ProfilePath,
		OriginalEntrypoints: append([]string(nil), entry.Entrypoints...),
		SelectedEntrypoints: append([]string(nil), entry.CandidateEntrypoints...),
		ValidatedBindings:   append([]string(nil), optimizeReplayBindingNames...), FailedBindings: []string{},
		Source:               optimizeSelectionSourceLocal,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	return &impact
}

func retainedOptimizeSelection(reason string, failed ...string) optimizeSelectionResult {
	result := emptyOptimizeSelection(optimizeSelectionRetained, reason, true)
	result.FailedBindings = append([]string(nil), failed...)
	sort.Strings(result.FailedBindings)
	return result
}

func findOptimizePortfolioEntry(entries []optimizePortfolioEntry, familySHA string) (optimizePortfolioEntry, bool) {
	index := sort.Search(len(entries), func(index int) bool { return entries[index].FamilySHA256 >= familySHA })
	if index >= len(entries) || entries[index].FamilySHA256 != familySHA {
		return optimizePortfolioEntry{}, false
	}
	return entries[index], true
}

func validateOptimizeReplayEntry(
	invocation optimizeInvocation,
	discovery optimizeDiscoveryResult,
	calibration optimizeCalibrationResult,
	entry optimizePortfolioEntry,
) []string {
	failures := []string{}
	if entry.RepositoryID != discovery.RepositoryID ||
		optimizePortfolioRepositoryScope(entry.RepositoryID) != optimizePortfolioRepositoryScope(invocation.discovery.RepositoryID) {
		failures = append(failures, "REPOSITORY_ID")
	}
	head, err := gitOutput(invocation.repositoryRoot, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != discovery.TargetRevision || entry.TargetRevision != discovery.TargetRevision {
		failures = append(failures, "REPOSITORY_REVISION")
	}
	if entry.WrapperSHA256 != invocation.wrapperSHA256 {
		failures = append(failures, "GRADLE_WRAPPER_PROPERTIES")
	}
	if entry.ExecutableSHA256 != invocation.executableSHA256 {
		failures = append(failures, "BUILDOPT_EXECUTABLE")
	}
	if entry.Family != discovery.ChangeFamily || entry.FamilySHA256 != optimizePortfolioFamilySHA(discovery) ||
		!equalOptimizeStrings(entry.ChangedProjects, discovery.ChangedProjects) {
		failures = append(failures, "CHANGE_FAMILY")
	}
	if !equalOptimizeStrings(entry.Entrypoints, discovery.Entrypoints) ||
		!equalOptimizeStrings(entry.CandidateEntrypoints, discovery.CandidateEntrypoints) {
		failures = append(failures, "WORKFLOW_ENTRYPOINTS")
	}
	if !equalOptimizeStrings(entry.RequiredOutputs, discovery.RequiredOutputs) {
		failures = append(failures, "DISCOVERY_DOCUMENTS")
	}
	profile, err := loadQualifiedPOCProfile(invocation.repositoryRoot, entry.ProfilePath)
	if err != nil {
		failures = append(failures, "PROFILE_ARTIFACTS")
	} else {
		expectedOptions, reason := optimizeCalibrationGradleOptions(invocation.discovery.gradleOptions)
		if reason != "" || !equalOptimizeStrings(profile.GradleOptions, expectedOptions) {
			failures = append(failures, "GRADLE_OPTIONS")
		}
		if _, valid := evaluateQualifiedPOCPreconditions(invocation.repositoryRoot, profile.Preconditions); !valid {
			failures = append(failures, "PROFILE_PRECONDITIONS")
		}
		if profile.Qualification == nil || profile.Qualification.SHA256 != calibration.EvidenceSHA256 ||
			profile.Qualification.RepositoryRevision != discovery.TargetRevision {
			failures = append(failures, "CALIBRATION_EVIDENCE")
		}
	}
	if err := validateOptimizeCalibrationEvidence(invocation, discovery, calibration); err != nil {
		failures = append(failures, "CALIBRATION_EVIDENCE")
	}
	sort.Strings(failures)
	return uniqueOptimizeStrings(failures)
}

func equalOptimizeStrings(left, right []string) bool {
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

func uniqueOptimizeStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func optimizeSelectionDescription(selection optimizeSelectionResult) string {
	if selection.Selected {
		return fmt.Sprintf("qualified %s profile", selection.ChangeFamily)
	}
	return "optimized native Gradle"
}

func optimizeExecutionMode(selection optimizeSelectionResult) string {
	if selection.Selected {
		return "SELECTIVE_PROFILE"
	}
	return "OPTIMIZED_NATIVE"
}
