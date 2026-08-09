package launcher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

const impactTimingSchemaVersion = "buildopt.build-impact/poc-phase-timings/v1"

type impactTimingReport struct {
	SchemaVersion       string                               `json:"schemaVersion"`
	CandidateSelected   bool                                 `json:"candidateSelected"`
	AlternativeID       string                               `json:"alternativeId,omitempty"`
	Reason              string                               `json:"reason"`
	EntrypointCount     int                                  `json:"entrypointCount"`
	ExitCode            int                                  `json:"exitCode"`
	Planner             buildimpact.POCCandidatePhaseTimings `json:"planner"`
	ImpactPreparationNs int64                                `json:"impactPreparationNs"`
	GradleSetupNs       int64                                `json:"gradleSetupNs"`
	RuntimeSetupNs      int64                                `json:"runtimeSetupNs"`
	GradleExecutionNs   int64                                `json:"gradleExecutionNs"`
	TeardownNs          int64                                `json:"teardownNs"`
	UnattributedNs      int64                                `json:"unattributedNs"`
	TotalNs             int64                                `json:"totalNs"`
}

type impactTimingState struct {
	path                  string
	startedAt             time.Time
	gradleSetupFinishedAt time.Time
	childReturnedAt       time.Time
	report                impactTimingReport
}

func newImpactTimingState(path string, startedAt time.Time, invocation impactInvocation) *impactTimingState {
	return &impactTimingState{
		path:      path,
		startedAt: startedAt,
		report: impactTimingReport{
			SchemaVersion:       impactTimingSchemaVersion,
			CandidateSelected:   invocation.plan.CandidateSelected,
			AlternativeID:       invocation.plan.AlternativeID,
			Reason:              invocation.plan.Reason,
			EntrypointCount:     len(invocation.plan.Entrypoints),
			Planner:             invocation.plan.PhaseTimings,
			ImpactPreparationNs: invocation.preparationNs,
		},
	}
}

func (state *impactTimingState) finishGradleSetup(startedAt time.Time) {
	state.report.GradleSetupNs = time.Since(startedAt).Nanoseconds()
	state.gradleSetupFinishedAt = time.Now()
}

func (state *impactTimingState) execute(
	childArgs []string,
	environmentOverrides map[string]string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) childExecution {
	startedAt := time.Now()
	if !state.gradleSetupFinishedAt.IsZero() {
		state.report.RuntimeSetupNs = startedAt.Sub(state.gradleSetupFinishedAt).Nanoseconds()
	}
	execution := executeChild(childArgs, environmentOverrides, stdin, stdout, stderr)
	state.childReturnedAt = time.Now()
	state.report.GradleExecutionNs = state.childReturnedAt.Sub(startedAt).Nanoseconds()
	return execution
}

func (state *impactTimingState) write(exitCode int) error {
	finishedAt := time.Now()
	state.report.ExitCode = exitCode
	if !state.childReturnedAt.IsZero() {
		state.report.TeardownNs = finishedAt.Sub(state.childReturnedAt).Nanoseconds()
	}
	state.report.TotalNs = finishedAt.Sub(state.startedAt).Nanoseconds()
	attributed := state.report.ImpactPreparationNs +
		state.report.GradleSetupNs +
		state.report.RuntimeSetupNs +
		state.report.GradleExecutionNs +
		state.report.TeardownNs
	state.report.UnattributedNs = state.report.TotalNs - attributed
	if state.report.UnattributedNs < 0 {
		return errors.New("Build Impact phase timings do not reconcile")
	}
	return writeCanonicalPrivateJSON(state.path, state.report)
}

func resolveImpactTimingsPath(repositoryRoot, relativePath string) (string, error) {
	if relativePath == "" {
		return "", nil
	}
	if filepath.IsAbs(relativePath) || filepath.Clean(relativePath) != relativePath || relativePath == "." || relativePath == ".." {
		return "", errors.New("Build Impact timings file must be clean and repository relative")
	}
	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("resolve Build Impact repository root: %w", err)
	}
	path := filepath.Join(canonicalRoot, relativePath)
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("resolve Build Impact timings directory: %w", err)
	}
	relative, err := filepath.Rel(canonicalRoot, parent)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("Build Impact timings file escapes the repository")
	}
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", errors.New("Build Impact timings file must be a regular file")
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("inspect Build Impact timings file: %w", statErr)
	}
	return path, nil
}
