package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/requestportfolio"
	"github.com/tonyredondo/buildopt/internal/stickyobservation"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
)

const (
	requestPortfolioOutputEnvironment        = "BUILDOPT_REQUEST_PORTFOLIO_OUTPUT"
	requestPortfolioEvidenceEnvironment      = "BUILDOPT_REQUEST_PORTFOLIO_EVIDENCE"
	requestPortfolioObservationIDEnvironment = "BUILDOPT_REQUEST_PORTFOLIO_OBSERVATION_ID"
	requestPortfolioCaptureEnvironment       = "BUILDOPT_REQUEST_PORTFOLIO_CAPTURE"
	requestPortfolioArgumentsEnvironment     = "BUILDOPT_REQUEST_PORTFOLIO_ARGUMENTS"
	requestPortfolioGeneratedAtEnvironment   = "BUILDOPT_REQUEST_PORTFOLIO_GENERATED_AT"
)

// requestPortfolioState is a post-build observation companion. It consumes
// only the argv already received by the wrapper and optional finalized
// evidence emitted by that same invocation; it never starts or widens Gradle.
type requestPortfolioState struct {
	root                     string
	outputPath               string
	evidencePath             string
	observationID            string
	repositoryScope          string
	argumentsSHA             string
	workingDirectorySHA      string
	workingDirectoryEvidence string
	startedAt                time.Time
	bypassed                 bool
	execution                childExecution
	executionSeen            bool
	capturePath              string
	preserveCapture          bool
	initScriptPath           string
	captureArguments         string
	capturePrepared          bool
	captureErr               error
}

func newRequestPortfolioStateAt(root string, childArgs []string, startedAt time.Time) *requestPortfolioState {
	if root == "" || !isGradleChild(childArgs) {
		return nil
	}
	absolute, err := filepath.Abs(root)
	if err != nil || filepath.Clean(absolute) != absolute {
		return nil
	}
	identity := absolute
	if config, configErr := stickywrapper.LoadConfig(absolute); configErr == nil {
		if config.Mode == "off" {
			return nil
		}
		if config.ProjectScope != "" {
			identity = config.ProjectScope
		}
	}
	repositoryScope := stickyobservation.ScopeForRoot(identity)
	outputPath := os.Getenv(requestPortfolioOutputEnvironment)
	if outputPath == "" {
		cacheRoot, cacheErr := os.UserCacheDir()
		if cacheErr != nil {
			return nil
		}
		outputPath = filepath.Join(cacheRoot, "buildopt", "sticky", "portfolios", repositoryScope, "requests.json")
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	argumentsSHA := requestportfolio.ArgumentsSHA256(childArgs[1:])
	workingDirectorySHA := requestportfolio.CompatibilitySHA256("UNAVAILABLE-WORKING-DIRECTORY")
	workingDirectoryEvidence := "UNAVAILABLE"
	if workingDirectory, workingErr := os.Getwd(); workingErr == nil {
		if relative, relativeErr := filepath.Rel(absolute, workingDirectory); relativeErr == nil &&
			relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			workingDirectorySHA = requestportfolio.CompatibilitySHA256("working-directory", filepath.ToSlash(relative))
			workingDirectoryEvidence = "EXACT"
		}
	}
	observationID := stickyobservation.Digest(startedAt.Format(time.RFC3339Nano), repositoryScope, argumentsSHA, workingDirectorySHA)
	return &requestPortfolioState{
		root: absolute, outputPath: outputPath,
		evidencePath:    os.Getenv(requestPortfolioEvidenceEnvironment),
		observationID:   observationID,
		repositoryScope: repositoryScope, argumentsSHA: argumentsSHA,
		workingDirectorySHA: workingDirectorySHA, workingDirectoryEvidence: workingDirectoryEvidence,
		startedAt: startedAt, bypassed: os.Getenv(bypassEnvironment) == "1",
	}
}

func (state *requestPortfolioState) childEnvironment(existing map[string]string) map[string]string {
	if state == nil || state.evidencePath == "" {
		return existing
	}
	result := make(map[string]string, len(existing)+5)
	for name, value := range existing {
		result[name] = value
	}
	result[requestPortfolioEvidenceEnvironment] = state.evidencePath
	result[requestPortfolioObservationIDEnvironment] = state.observationID
	if state.capturePrepared {
		result[requestPortfolioCaptureEnvironment] = state.capturePath
		result[requestPortfolioArgumentsEnvironment] = state.captureArguments
		result[requestPortfolioGeneratedAtEnvironment] = state.startedAt.UTC().Format(time.RFC3339Nano)
	}
	return result
}

func (state *requestPortfolioState) finishGradle(execution childExecution) {
	if state == nil {
		return
	}
	state.execution = execution
	state.executionSeen = true
}

func (state *requestPortfolioState) finish(exitCode int, completedAt time.Time) error {
	if state == nil || state.outputPath == "" {
		return nil
	}
	defer state.cleanupCaptureArtifacts()
	if completedAt.Before(state.startedAt) {
		completedAt = state.startedAt
	}
	observation := requestportfolio.Observation{
		ObservationID: state.observationID, ObservedAt: completedAt.UTC().Format(time.RFC3339Nano),
		RepositoryScopeSHA256: state.repositoryScope, ArgumentsSHA256: state.argumentsSHA,
		WorkingDirectorySHA256: state.workingDirectorySHA, WorkingDirectoryEvidence: state.workingDirectoryEvidence,
		CompatibilityIdentitySHA256: requestportfolio.CompatibilitySHA256("UNAVAILABLE"),
		CompatibilityEvidence:       "UNAVAILABLE", RequestGraphEvidence: "UNAVAILABLE",
		Outcome: "INFRA_FAILURE", ExitCode: exitCode, Bypassed: state.bypassed,
	}
	if state.executionSeen && state.execution.started {
		switch {
		case state.execution.cancelled:
			observation.Outcome = "CANCELLED"
		case exitCode == 0:
			observation.Outcome = "SUCCESS"
		default:
			observation.Outcome = "BUILD_FAILURE"
		}
	}
	var evidenceErr error
	if state.evidencePath != "" {
		evidence, err := requestportfolio.LoadEvidence(state.evidencePath, state.observationID, state.argumentsSHA)
		if err != nil && state.captureErr == nil && state.capturePrepared && !state.bypassed {
			evidenceErr = state.materializeEvidence()
			evidence, err = requestportfolio.LoadEvidence(state.evidencePath, state.observationID, state.argumentsSHA)
		}
		if state.captureErr != nil {
			evidenceErr = errors.Join(evidenceErr, state.captureErr)
		}
		if err == nil {
			observation.CompatibilityIdentitySHA256 = evidence.CompatibilityIdentitySHA256
			observation.CompatibilityEvidence = "EXACT"
			observation.RequestedTasks = append([]string(nil), evidence.RequestedTasks...)
			observation.RequestGraphIdentitySHA256 = evidence.RequestGraphIdentitySHA256
			observation.RequestGraphEvidence = "EXACT"
		} else {
			evidenceErr = errors.Join(evidenceErr, err)
		}
	}
	store, err := requestportfolio.NewStore(state.outputPath)
	if err != nil {
		return errors.Join(evidenceErr, err)
	}
	_, storeErr := store.Observe(observation)
	return errors.Join(evidenceErr, storeErr)
}
