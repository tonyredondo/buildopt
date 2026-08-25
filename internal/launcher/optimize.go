package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	optimizeUsage = "usage: buildopt optimize [--state-dir PATH] [--connection-dir PATH] [--resume auto|never] [--calibration-budget DURATION] [--calibration-pairs N] [--max-break-even-builds N] [--json] [--] <gradle args...>\n"

	optimizeStateSchemaVersion              = "buildopt.poc/optimize-state/v1"
	optimizeResultSchemaVersion             = "buildopt.poc/optimize-result/v1"
	optimizeDefaultStateDir                 = ".buildopt/optimize/v1"
	optimizeStateFile                       = "state.json"
	optimizeResultFile                      = "result.json"
	optimizeValueReportJSONFile             = "value-report.json"
	optimizeValueReportMDFile               = "value-report.md"
	optimizeOutcomeLearning                 = "LEARNING"
	optimizeOutcomeNative                   = "NATIVE_RETAINED"
	optimizePhaseUnseen                     = "UNSEEN"
	optimizeReasonPending                   = "AUTO_DISCOVERY_PENDING"
	optimizeResumeAuto                      = "AUTO"
	optimizeResumeNever                     = "NEVER"
	optimizeResumeNone                      = "NO_CHECKPOINT"
	optimizeResumeExact                     = "EXACT_BINDINGS"
	optimizeResumeDrift                     = "BINDING_DRIFT"
	optimizeResumeInvalid                   = "INVALID_CHECKPOINT"
	optimizeResumeDisabled                  = "RESUME_DISABLED"
	optimizeRetentionPreGradleCompatibility = "PRE_GRADLE_COMPATIBILITY"
	optimizeRetentionPreGradleEconomic      = "PRE_GRADLE_ECONOMIC"
	optimizeRetentionPostGradleDiscovery    = "POST_GRADLE_DISCOVERY"
	optimizeRetentionProfileSelected        = "PROFILE_SELECTED"
	optimizeBindingContractOnly             = "CONTRACT_ONLY"
	optimizeBindingDiscovery                = "DISCOVERY_COMPLETE"
	optimizeBindingReplay                   = "AUTOMATIC_REPLAY_COMPLETE"
	optimizeDefaultBudget                   = 30 * time.Minute
	optimizeDefaultPairs                    = 8
	optimizeDefaultBreakEven                = 30
	optimizeMaximumStateBytes               = 1 << 20
)

var errOptimizeUsage = errors.New("invalid optimize usage")

type optimizeInvocation struct {
	startedAt           time.Time
	repositoryRoot      string
	stateDirectory      string
	stateRelative       string
	connectionDirectory string
	connectionRelative  string
	gradleArgs          []string
	resumeMode          string
	calibrationBudget   time.Duration
	calibrationPairs    int
	maxBreakEvenBuilds  int
	jsonOutput          bool
	bindingSHA256       string
	executableSHA256    string
	wrapperSHA256       string
	invocationSHA256    string
	repositoryScopeSHA  string
	discoveryContextSHA string
	discovery           optimizeDiscoveryContext
}

type optimizeBudget struct {
	WallTimeSeconds    int64 `json:"wallTimeSeconds"`
	Pairs              int   `json:"pairs"`
	MaxBreakEvenBuilds int   `json:"maxBreakEvenBuilds"`
}

type optimizeBindings struct {
	SHA256                 string `json:"sha256"`
	Completeness           string `json:"completeness"`
	ExecutableSHA256       string `json:"executableSha256"`
	WrapperSHA256          string `json:"wrapperSha256"`
	InvocationSHA256       string `json:"invocationSha256"`
	RepositoryScopeSHA256  string `json:"repositoryScopeSha256"`
	DiscoveryContextSHA256 string `json:"discoveryContextSha256"`
}

type optimizeResume struct {
	Mode                string `json:"mode"`
	CheckpointFound     bool   `json:"checkpointFound"`
	Accepted            bool   `json:"accepted"`
	Reason              string `json:"reason"`
	PreviousStateSHA256 string `json:"previousStateSha256,omitempty"`
}

type optimizeState struct {
	SchemaVersion        string                      `json:"schemaVersion"`
	Generation           int                         `json:"generation"`
	Attempt              int                         `json:"attempt"`
	Phase                string                      `json:"phase"`
	LastOutcome          string                      `json:"lastOutcome"`
	LastReason           string                      `json:"lastReason"`
	Bindings             optimizeBindings            `json:"bindings"`
	Budget               optimizeBudget              `json:"budget"`
	Resume               optimizeResume              `json:"resume"`
	BuildStarted         bool                        `json:"buildStarted"`
	LastExitCode         int                         `json:"lastExitCode"`
	Discovery            optimizeDiscoveryResult     `json:"discovery"`
	IncrementalLearning  optimizeIncrementalLearning `json:"incrementalLearning"`
	Calibration          optimizeCalibrationResult   `json:"calibration"`
	Portfolio            optimizePortfolioResult     `json:"portfolio"`
	Selection            optimizeSelectionResult     `json:"selection"`
	Value                optimizeValueState          `json:"value"`
	UpdatedAt            string                      `json:"updatedAt"`
	ProductionAuthorized bool                        `json:"productionAuthorized"`
}

type optimizeNativeResult struct {
	Authoritative bool `json:"authoritative"`
	Started       bool `json:"started"`
	ExitCode      int  `json:"exitCode"`
}

type optimizeExecutionResult struct {
	Mode          string `json:"mode"`
	Authoritative bool   `json:"authoritative"`
	Started       bool   `json:"started"`
	ExitCode      int    `json:"exitCode"`
}

// optimizeNativeRetentionResult attributes BuildOpt-owned wrapper work
// directly instead of inferring it from one noisy native/candidate pair.
// Output observation is explicit because it intentionally changes the Gradle
// invocation and is only justified when the configured model is still needed.
type optimizeNativeRetentionResult struct {
	DecisionPhase             string `json:"decisionPhase"`
	Reason                    string `json:"reason"`
	CompletedBeforeGradle     bool   `json:"completedBeforeGradle"`
	OutputObservationPrepared bool   `json:"outputObservationPrepared"`
	GradleDurationMS          int64  `json:"gradleDurationMs"`
	PreExecutionMS            int64  `json:"preExecutionMs"`
	PostExecutionMS           int64  `json:"postExecutionMs"`
	WrapperOverheadMS         int64  `json:"wrapperOverheadMs"`
}

// optimizeTimingResult separates the non-overlapping externally relevant
// phases from nested diagnostics. Diagnostics explain work inside the top-level
// phases and therefore must never be added to TotalNS.
type optimizeTimingResult struct {
	PreExecutionNS    int64                     `json:"preExecutionNs"`
	GradleExecutionNS int64                     `json:"gradleExecutionNs"`
	FinalizationNS    int64                     `json:"finalizationNs"`
	UnattributedNS    int64                     `json:"unattributedNs"`
	TotalNS           int64                     `json:"totalNs"`
	Diagnostics       optimizeTimingDiagnostics `json:"diagnostics"`
}

type optimizeTimingDiagnostics struct {
	GradleSetupNS        int64 `json:"gradleSetupNs"`
	MatchingNS           int64 `json:"matchingNs"`
	LocalStateNS         int64 `json:"localStateNs"`
	CentralStateNS       int64 `json:"centralStateNs"`
	MaterializationNS    int64 `json:"materializationNs"`
	OutputVerificationNS int64 `json:"outputVerificationNs"`
	DiscoveryLearningNS  int64 `json:"discoveryLearningNs"`
}

type optimizeGeneratedFiles struct {
	State         string   `json:"state"`
	Result        string   `json:"result"`
	Discovery     []string `json:"discovery"`
	Calibration   []string `json:"calibration"`
	Portfolio     []string `json:"portfolio"`
	ValueJSON     string   `json:"valueJson"`
	ValueMarkdown string   `json:"valueMarkdown"`
}

type optimizeResult struct {
	SchemaVersion        string                           `json:"schemaVersion"`
	Outcome              string                           `json:"outcome"`
	Reason               string                           `json:"reason"`
	Phase                string                           `json:"phase"`
	StartedAt            string                           `json:"startedAt"`
	CompletedAt          string                           `json:"completedAt"`
	DurationMs           int64                            `json:"durationMs"`
	Generation           int                              `json:"generation"`
	Attempt              int                              `json:"attempt"`
	Bindings             optimizeBindings                 `json:"bindings"`
	Budget               optimizeBudget                   `json:"budget"`
	Resume               optimizeResume                   `json:"resume"`
	Native               optimizeNativeResult             `json:"native"`
	Execution            optimizeExecutionResult          `json:"execution"`
	NativeRetention      optimizeNativeRetentionResult    `json:"nativeRetention"`
	Timing               optimizeTimingResult             `json:"timing"`
	Discovery            optimizeDiscoveryResult          `json:"discovery"`
	IncrementalLearning  optimizeIncrementalLearning      `json:"incrementalLearning"`
	Calibration          optimizeCalibrationResult        `json:"calibration"`
	Portfolio            optimizePortfolioResult          `json:"portfolio"`
	Selection            optimizeSelectionResult          `json:"selection"`
	Prequalification     optimizeEconomicPrequalification `json:"prequalification"`
	Central              optimizeCentralResult            `json:"central"`
	Value                optimizeValueState               `json:"value"`
	GeneratedFiles       optimizeGeneratedFiles           `json:"generatedFiles"`
	ManualFilesRequired  int                              `json:"manualFilesRequired"`
	CalibrationPerformed bool                             `json:"calibrationPerformed"`
	PortfolioPerformed   bool                             `json:"portfolioPerformed"`
	SelectionPerformed   bool                             `json:"selectionPerformed"`
	ProductionAuthorized bool                             `json:"productionAuthorized"`
	TestOptimization     string                           `json:"testOptimization"`
}

type optimizeRun struct {
	invocation       optimizeInvocation
	state            optimizeState
	statePath        string
	resultPath       string
	valueJSONPath    string
	valueMDPath      string
	startedAt        time.Time
	childStarted     bool
	childExecution   childExecution
	previousState    *optimizeState
	selection        optimizeSelectionResult
	prequalification optimizeEconomicPrequalification
	central          *centralOptimizeIntegration
	centralReplay    *centralOptimizeReplay
	// gradleBuildCacheSeed is the cache actually used by the authoritative
	// build. Connected invocations use managed L1 rather than Gradle's default
	// build-cache-1 directory, so calibration must snapshot this exact path.
	gradleBuildCacheSeed   string
	outputObservation      *optimizeOutputObservation
	incrementalArm         string
	incrementalCandidate   bool
	incrementalOutputSHA   string
	incrementalOutputCount int
	incrementalFailure     string
	incrementalFallback    childExecution
	incrementalObserved    bool
	materializationTime    time.Duration
	outputVerificationTime time.Duration
	discoveryTime          time.Duration
	gradleSetupTime        time.Duration
	localStateTime         time.Duration
	centralStateTime       time.Duration
	finishStartedAt        time.Time
	earlyRetentionReason   string
	retentionDecisionPhase string
}

func prepareOptimizeInvocation(args []string, stateEnabled bool) (optimizeInvocation, error) {
	startedAt := time.Now()
	flags := flag.NewFlagSet("buildopt optimize", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDirectory := flags.String("state-dir", optimizeDefaultStateDir, "generated repository-local optimize state")
	connectionDirectory := flags.String("connection-dir", centralConnectionDir, "private repository-local central connection")
	resume := flags.String("resume", "auto", "resume exact state automatically or never")
	budget := flags.Duration("calibration-budget", optimizeDefaultBudget, "maximum calibration wall time per invocation")
	pairs := flags.Int("calibration-pairs", optimizeDefaultPairs, "maximum balanced calibration pairs")
	maxBreakEven := flags.Int("max-break-even-builds", optimizeDefaultBreakEven, "largest acceptable matching-build payback")
	jsonOutput := flags.Bool("json", false, "reserve stdout for the final machine-readable result")
	if err := flags.Parse(args); err != nil || flags.NArg() == 0 {
		return optimizeInvocation{}, fmt.Errorf("%w: invalid arguments", errOptimizeUsage)
	}
	resumeMode := strings.ToUpper(*resume)
	if resumeMode != optimizeResumeAuto && resumeMode != optimizeResumeNever {
		return optimizeInvocation{}, fmt.Errorf("%w: --resume must be auto or never", errOptimizeUsage)
	}
	if *budget <= 0 || *budget > 24*time.Hour {
		return optimizeInvocation{}, fmt.Errorf("%w: --calibration-budget must be greater than zero and at most 24h", errOptimizeUsage)
	}
	if *pairs < 2 || *pairs > 16 {
		return optimizeInvocation{}, fmt.Errorf("%w: --calibration-pairs must be between 2 and 16", errOptimizeUsage)
	}
	if *maxBreakEven < 1 || *maxBreakEven > 1000 {
		return optimizeInvocation{}, fmt.Errorf("%w: --max-break-even-builds must be between 1 and 1000", errOptimizeUsage)
	}
	if !validOptimizeStateRelative(*stateDirectory) {
		return optimizeInvocation{}, fmt.Errorf("%w: --state-dir must be a clean repository-relative path under .buildopt", errOptimizeUsage)
	}
	invocation := optimizeInvocation{
		startedAt:  startedAt,
		gradleArgs: append([]string(nil), flags.Args()...), resumeMode: resumeMode,
		calibrationBudget: *budget, calibrationPairs: *pairs,
		maxBreakEvenBuilds: *maxBreakEven, jsonOutput: *jsonOutput,
	}
	if !stateEnabled {
		return invocation, nil
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		return optimizeInvocation{}, err
	}
	statePath, stateRelative, err := resolveOptimizeStateDirectory(repositoryRoot, *stateDirectory, false)
	if err != nil {
		return optimizeInvocation{}, err
	}
	invocation.repositoryRoot = repositoryRoot
	invocation.stateDirectory = statePath
	invocation.stateRelative = stateRelative
	connectionPath, connectionRelative, err := resolveCentralConnectionDirectory(repositoryRoot, *connectionDirectory, false)
	if err != nil {
		return optimizeInvocation{}, err
	}
	invocation.connectionDirectory = connectionPath
	invocation.connectionRelative = connectionRelative
	invocation.discovery = inspectOptimizeDiscoveryContext(repositoryRoot, invocation.gradleArgs, os.Getenv)
	if err := bindOptimizeInvocation(&invocation); err != nil {
		return optimizeInvocation{}, err
	}
	return invocation, nil
}

func resolveOptimizeStateDirectory(repositoryRoot, relative string, create bool) (string, string, error) {
	normalized, valid := normalizeOptimizeStateRelative(relative)
	if !valid {
		return "", "", errors.New("--state-dir must be a clean repository-relative path under .buildopt")
	}
	current := repositoryRoot
	for _, segment := range strings.Split(normalized, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if !create {
				continue
			}
			if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
				return "", "", fmt.Errorf("create optimize state directory: %w", err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", "", errors.New("optimize state path must contain only private directories without symlinks")
		}
	}
	return current, filepath.ToSlash(normalized), nil
}

func validOptimizeStateRelative(relative string) bool {
	_, valid := normalizeOptimizeStateRelative(relative)
	return valid
}

func normalizeOptimizeStateRelative(relative string) (string, bool) {
	portable := filepath.FromSlash(relative)
	normalized := filepath.Clean(portable)
	valid := relative != "" && !filepath.IsAbs(normalized) && normalized == portable &&
		(normalized == ".buildopt" ||
			strings.HasPrefix(normalized, ".buildopt"+string(filepath.Separator)))
	return normalized, valid
}

func bindOptimizeInvocation(invocation *optimizeInvocation) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve BuildOpt executable: %w", err)
	}
	invocation.executableSHA256, err = optimizeFileSHA256(executable, false)
	if err != nil {
		return fmt.Errorf("hash BuildOpt executable: %w", err)
	}
	wrapper := filepath.Join(invocation.repositoryRoot, "gradle", "wrapper", "gradle-wrapper.properties")
	invocation.wrapperSHA256, err = optimizeFileSHA256(wrapper, true)
	if err != nil {
		return fmt.Errorf("hash Gradle Wrapper properties: %w", err)
	}
	invocation.invocationSHA256 = optimizeDigest("buildopt-optimize-invocation-v1", invocation.gradleArgs...)
	invocation.repositoryScopeSHA, err = optimizeRepositoryScopeSHA(invocation, os.Getenv)
	if err != nil {
		return err
	}
	invocation.discoveryContextSHA = optimizeDiscoveryContextSHA(invocation.discovery)
	invocation.bindingSHA256 = optimizeDigest(
		"buildopt-optimize-bindings-v1",
		invocation.executableSHA256,
		invocation.wrapperSHA256,
		invocation.invocationSHA256,
		invocation.repositoryScopeSHA,
		invocation.discoveryContextSHA,
		strconv.FormatInt(int64(invocation.calibrationBudget/time.Second), 10),
		strconv.Itoa(invocation.calibrationPairs),
		strconv.Itoa(invocation.maxBreakEvenBuilds),
	)
	return nil
}

// optimizeRepositoryScopeSHA keeps local state checkout-bound while allowing
// exact CI state to survive an ephemeral workspace path. Provider identity is
// only one binding; revision, Wrapper, executable, argv, discovery and budget
// remain independently bound by bindOptimizeInvocation.
func optimizeRepositoryScopeSHA(invocation *optimizeInvocation, getenv func(string) string) (string, error) {
	localScope := optimizeDigest("buildopt-optimize-repository-scope-v1", invocation.repositoryRoot)
	provider := ""
	repositoryIDVariable := ""
	repositoryPathVariable := ""
	targetVariable := ""
	switch {
	case getenv("GITHUB_ACTIONS") == "true":
		provider = "GITHUB"
		repositoryIDVariable = "GITHUB_REPOSITORY_ID"
		repositoryPathVariable = "GITHUB_REPOSITORY"
		targetVariable = "GITHUB_SHA"
	case getenv("GITLAB_CI") == "true":
		provider = "GITLAB"
		repositoryIDVariable = "CI_PROJECT_ID"
		repositoryPathVariable = "CI_PROJECT_PATH"
		targetVariable = "CI_COMMIT_SHA"
	default:
		return localScope, nil
	}
	if invocation.discovery.RepositoryID == "" || invocation.discovery.TargetRevision == "" {
		return localScope, nil
	}

	repositoryNumericID := strings.TrimSpace(getenv(repositoryIDVariable))
	repositoryPath := strings.TrimSpace(getenv(repositoryPathVariable))
	targetRevision := strings.ToLower(strings.TrimSpace(getenv(targetVariable)))
	expectedRepositoryID := repositoryPath
	if provider == "GITLAB" {
		expectedRepositoryID = optimizeGitLabRepositoryID(repositoryPath)
	}
	if repositoryNumericID == "" || len(repositoryNumericID) > 32 ||
		strings.Trim(repositoryNumericID, "0123456789") != "" ||
		expectedRepositoryID == "" || expectedRepositoryID != invocation.discovery.RepositoryID ||
		!validMeasurementRevision(targetRevision) ||
		targetRevision != invocation.discovery.TargetRevision {
		// Provider metadata can be absent or refer to an outer checkout when
		// optimize is invoked in a nested/generated repository. Such a context
		// may run native Gradle, but it cannot accept state from another runner.
		return localScope, nil
	}
	return optimizeDigest(
		"buildopt-optimize-ci-repository-scope-v1",
		provider,
		repositoryNumericID,
		repositoryPath,
	), nil
}

func optimizeFileSHA256(path string, allowMissing bool) (string, error) {
	raw, err := os.ReadFile(path)
	if allowMissing && os.IsNotExist(err) {
		return optimizeDigest("buildopt-optimize-missing-file-v1", filepath.Base(path)), nil
	}
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func optimizeDigest(domain string, values ...string) string {
	digest := sha256.New()
	optimizeWriteDigestValue(digest, domain)
	for _, value := range values {
		optimizeWriteDigestValue(digest, value)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func optimizeWriteDigestValue(writer io.Writer, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = io.WriteString(writer, value)
}

func beginOptimizeRun(invocation optimizeInvocation) (*optimizeRun, error) {
	localStateStartedAt := time.Now()
	if _, _, err := resolveOptimizeStateDirectory(
		invocation.repositoryRoot,
		invocation.stateRelative,
		true,
	); err != nil {
		return nil, err
	}
	startedAt := invocation.startedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	statePath := filepath.Join(invocation.stateDirectory, optimizeStateFile)
	resultPath := filepath.Join(invocation.stateDirectory, optimizeResultFile)
	valueJSONPath := filepath.Join(invocation.stateDirectory, optimizeValueReportJSONFile)
	valueMDPath := filepath.Join(invocation.stateDirectory, optimizeValueReportMDFile)
	resume, generation, attempt, previous := inspectOptimizeCheckpoint(statePath, invocation)
	state := optimizeState{
		SchemaVersion: optimizeStateSchemaVersion, Generation: generation, Attempt: attempt,
		Phase: optimizePhaseUnseen, LastOutcome: optimizeOutcomeLearning,
		LastReason: optimizeReasonPending, Bindings: optimizeInvocationBindings(invocation),
		Budget: optimizeInvocationBudget(invocation), Resume: resume,
		BuildStarted: false, LastExitCode: 0,
		UpdatedAt: startedAt.Format(time.RFC3339Nano), ProductionAuthorized: false,
	}
	if err := writeCanonicalPrivateJSON(statePath, state); err != nil {
		return nil, err
	}
	return &optimizeRun{
		invocation: invocation, state: state, statePath: statePath,
		resultPath: resultPath, valueJSONPath: valueJSONPath, valueMDPath: valueMDPath,
		startedAt: startedAt, previousState: previous,
		prequalification:     unevaluatedOptimizePrequalification(optimizePrequalificationReasonNoGraph),
		earlyRetentionReason: staticOptimizeRetentionReason(invocation.discovery),
		localStateTime:       time.Since(localStateStartedAt),
	}, nil
}

func staticOptimizeRetentionReason(discovery optimizeDiscoveryContext) string {
	if !discovery.Ready {
		return ""
	}
	for _, changed := range discovery.changedPaths {
		if matchesAnyProposalGlob(optimizeGlobalChangePaths, changed) ||
			centralOptimizeBuildLogicPath(changed) {
			return "GLOBAL_CHANGE_REQUIRES_FULL_GRAPH"
		}
	}
	return ""
}

func inspectOptimizeCheckpoint(path string, invocation optimizeInvocation) (optimizeResume, int, int, *optimizeState) {
	resume := optimizeResume{Mode: invocation.resumeMode, Reason: optimizeResumeNone}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if invocation.resumeMode == optimizeResumeNever {
			resume.Reason = optimizeResumeDisabled
		}
		return resume, 1, 1, nil
	}
	if err != nil {
		resume.CheckpointFound = true
		resume.Reason = optimizeResumeInvalid
		return resume, 1, 1, nil
	}
	resume.CheckpointFound = true
	digest := sha256.Sum256(raw)
	resume.PreviousStateSHA256 = hex.EncodeToString(digest[:])
	if len(raw) > optimizeMaximumStateBytes {
		resume.Reason = optimizeResumeInvalid
		return resume, 1, 1, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var previous optimizeState
	if err := decoder.Decode(&previous); err != nil {
		resume.Reason = optimizeResumeInvalid
		return resume, 1, 1, nil
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) ||
		!validOptimizeState(previous) {
		resume.Reason = optimizeResumeInvalid
		return resume, 1, 1, nil
	}
	if invocation.resumeMode == optimizeResumeNever {
		resume.Reason = optimizeResumeDisabled
		return resume, previous.Generation + 1, 1, nil
	}
	if !sameOptimizeBindingIdentity(previous.Bindings, optimizeInvocationBindings(invocation)) ||
		previous.Budget != optimizeInvocationBudget(invocation) {
		resume.Reason = optimizeResumeDrift
		return resume, previous.Generation + 1, 1, nil
	}
	if err := validateOptimizeIncrementalEvidence(invocation, previous.IncrementalLearning); err != nil {
		resume.Reason = optimizeResumeInvalid
		return resume, previous.Generation + 1, 1, nil
	}
	resume.Accepted = true
	resume.Reason = optimizeResumeExact
	return resume, previous.Generation, previous.Attempt + 1, &previous
}

func sameOptimizeBindingIdentity(left, right optimizeBindings) bool {
	left.Completeness = ""
	right.Completeness = ""
	return left == right
}

func validOptimizeState(state optimizeState) bool {
	if state.SchemaVersion != optimizeStateSchemaVersion || state.Generation < 1 ||
		state.Attempt < 1 || state.Phase == "" || state.LastOutcome == "" ||
		state.LastReason == "" || state.ProductionAuthorized ||
		!validOptimizeDiscoveryCheckpoint(state) ||
		!validOptimizeIncrementalCheckpoint(state) ||
		!validOptimizeCalibrationCheckpoint(state) ||
		!validOptimizePortfolioCheckpoint(state) ||
		!validOptimizeSelectionCheckpoint(state) ||
		!validOptimizeValueState(state) ||
		!optimizeStringIn(state.Bindings.Completeness, optimizeBindingContractOnly, optimizeBindingDiscovery, optimizeBindingCalibration, optimizeBindingPortfolio, optimizeBindingReplay) ||
		state.Budget.WallTimeSeconds < 1 || state.Budget.Pairs < 2 ||
		state.Budget.Pairs > 16 || state.Budget.MaxBreakEvenBuilds < 1 ||
		state.Budget.MaxBreakEvenBuilds > 1000 {
		return false
	}
	if !optimizeStringIn(state.Phase,
		"UNSEEN", "DISCOVERED", "CALIBRATING", "QUALIFIED", "ACTIVE",
		"NATIVE_RETAINED", "STALE", "RECALIBRATING") ||
		!optimizeStringIn(state.LastOutcome,
			"LEARNING", "QUALIFIED_AND_USED", "NATIVE_RETAINED") ||
		!optimizeStringIn(state.Resume.Reason,
			optimizeResumeNone, optimizeResumeExact, optimizeResumeDrift,
			optimizeResumeInvalid, optimizeResumeDisabled) {
		return false
	}
	for _, digest := range []string{
		state.Bindings.SHA256,
		state.Bindings.ExecutableSHA256,
		state.Bindings.WrapperSHA256,
		state.Bindings.InvocationSHA256,
		state.Bindings.RepositoryScopeSHA256,
		state.Bindings.DiscoveryContextSHA256,
	} {
		if len(digest) != 64 || strings.ToLower(digest) != digest {
			return false
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	if state.Resume.Mode != optimizeResumeAuto && state.Resume.Mode != optimizeResumeNever {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, state.UpdatedAt); err != nil {
		return false
	}
	return true
}

func validOptimizeDiscoveryCheckpoint(state optimizeState) bool {
	discovery := state.Discovery
	if discovery.ProductionAuthorized || !validOptimizeNativeObservationShape(discovery.NativeObservation) {
		return false
	}
	switch discovery.Status {
	case "":
		return state.Phase == optimizePhaseUnseen && !state.BuildStarted &&
			discovery.Reason == "" && discovery.TestOptimization == ""
	case optimizeDiscoveryComplete:
		return optimizeStringIn(state.Phase, "DISCOVERED", "CALIBRATING", "QUALIFIED", "ACTIVE", "STALE", "NATIVE_RETAINED") &&
			discovery.Reason == optimizeDiscoveryReasonFound &&
			discovery.ReviewRequired && discovery.TestOptimization == "OUT_OF_SCOPE" &&
			validOptimizeDiscoveryFiles(discovery.GeneratedFiles, true) && discovery.Graph.TotalProjects > 0 &&
			discovery.Graph.SelectedProjects > 0 && discovery.Graph.OmittedProjects > 0 &&
			validOptimizeFamily(discovery.ChangeFamily) && len(discovery.ChangedProjects) > 0 &&
			uniqueMeasurementStrings(discovery.ChangedProjects) &&
			len(discovery.RequiredOutputs) > 0 && len(discovery.CandidateOutputs) > 0 &&
			uniqueMeasurementStrings(discovery.RequiredOutputs) &&
			uniqueMeasurementStrings(discovery.CandidateOutputs) &&
			validOptimizeOutputMaterializationShape(discovery.Materialization, true) &&
			validOptimizeStructuralBinding(discovery.StructuralBinding)
	case optimizeDiscoveryRemoteRevalidated:
		return optimizeStringIn(state.Phase, "ACTIVE", "STALE") &&
			discovery.Reason == "REMOTE_STRUCTURAL_PROFILE_REVALIDATED" &&
			discovery.ReviewRequired && discovery.TestOptimization == "OUT_OF_SCOPE" &&
			validOptimizeDiscoveryFiles(discovery.GeneratedFiles, true) &&
			discovery.Graph.TotalProjects > 0 && discovery.Graph.SelectedProjects > 0 &&
			discovery.Graph.OmittedProjects > 0 &&
			validOptimizeFamily(discovery.ChangeFamily) && len(discovery.ChangedProjects) > 0 &&
			uniqueMeasurementStrings(discovery.ChangedProjects) &&
			len(discovery.RequiredOutputs) > 0 && len(discovery.CandidateOutputs) > 0 &&
			uniqueMeasurementStrings(discovery.RequiredOutputs) &&
			uniqueMeasurementStrings(discovery.CandidateOutputs) &&
			validOptimizeOutputMaterializationShape(discovery.Materialization, true) &&
			validOptimizeStructuralBinding(discovery.StructuralBinding)
	case optimizeDiscoveryRetained, optimizeDiscoverySkipped:
		return state.Phase == "NATIVE_RETAINED" && discovery.Reason != "" &&
			discovery.ReviewRequired && discovery.TestOptimization == "OUT_OF_SCOPE" &&
			validOptimizeDiscoveryFiles(discovery.GeneratedFiles, false) &&
			validOptimizeOutputMaterializationShape(discovery.Materialization, false) &&
			emptyOptimizeStructuralBinding(discovery.StructuralBinding)
	default:
		return false
	}
}

func validOptimizeNativeObservationShape(observation optimizeNativeObservation) bool {
	if observation.Status == "" {
		return observation == (optimizeNativeObservation{})
	}
	if observation.Status != optimizeNativeObservationCaptured ||
		observation.Reason != optimizeNativeObservationReason || observation.ProductionAuthorized ||
		observation.OutputCount < 1 || observation.OutputCount > maximumOutputContractFiles ||
		!validOptimizeSHA(observation.SHA256) || !validOptimizeSHA(observation.OutputContractSHA256) {
		return false
	}
	native := filepath.FromSlash(observation.File)
	return observation.File != "" && !filepath.IsAbs(native) && filepath.Clean(native) == native &&
		strings.HasPrefix(observation.File, ".buildopt/")
}

func validOptimizeDiscoveryFiles(paths []string, complete bool) bool {
	// Local discovery retains the configured-model snapshot needed to rewrite
	// producer-bound quarantine plans. Central replay materializes only the
	// seven already-bound profile documents and does not recalibrate them.
	if len(paths) > 8 || (complete && len(paths) != 7 && len(paths) != 8) {
		return false
	}
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		native := filepath.FromSlash(path)
		if path == "" || filepath.IsAbs(native) || filepath.Clean(native) != native ||
			!strings.HasPrefix(path, ".buildopt/") || seen[path] {
			return false
		}
		seen[path] = true
	}
	return true
}

func optimizeStringIn(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func optimizeInvocationBindings(invocation optimizeInvocation) optimizeBindings {
	completeness := optimizeBindingContractOnly
	if invocation.discovery.Ready {
		completeness = optimizeBindingDiscovery
	}
	return optimizeBindings{
		SHA256: invocation.bindingSHA256, Completeness: completeness,
		ExecutableSHA256: invocation.executableSHA256, WrapperSHA256: invocation.wrapperSHA256,
		InvocationSHA256:       invocation.invocationSHA256,
		RepositoryScopeSHA256:  invocation.repositoryScopeSHA,
		DiscoveryContextSHA256: invocation.discoveryContextSHA,
	}
}

func optimizeInvocationBudget(invocation optimizeInvocation) optimizeBudget {
	return optimizeBudget{
		WallTimeSeconds:    int64(invocation.calibrationBudget / time.Second),
		Pairs:              invocation.calibrationPairs,
		MaxBreakEvenBuilds: invocation.maxBreakEvenBuilds,
	}
}

func (run *optimizeRun) finish(exitCode int, stdout, stderr io.Writer) error {
	run.finishStartedAt = time.Now()
	defer run.cleanupOutputObservation()
	budgetContext, cancelBudget := context.WithTimeout(context.Background(), run.invocation.calibrationBudget)
	defer cancelBudget()
	learningContext, stopSignals := notifyOptimizeLearningContext(budgetContext)
	defer stopSignals()
	discovery, calibration, resumed := run.resumeCalibration()
	learning := emptyOptimizeIncrementalLearning()
	if run.state.Resume.Accepted && run.previousState != nil {
		learning = cloneOptimizeIncrementalLearning(run.previousState.IncrementalLearning)
	}
	if run.earlyRetentionReason != "" {
		discovery = retainedOptimizeDiscovery(run.invocation, run.earlyRetentionReason)
		calibration = emptyOptimizeCalibration(run.invocation, discovery.Reason)
	} else if run.centralReplay != nil {
		discovery = run.centralReplay.discovery
		calibration = run.centralReplay.calibration
		resumed = true
	} else if !resumed && run.prequalification.Decision == optimizePrequalificationReject &&
		!(run.previousState != nil && run.previousState.IncrementalLearning.Status == optimizeIncrementalCollecting) {
		discovery = retainedOptimizeDiscovery(run.invocation, "ECONOMIC_PREQUALIFICATION_REJECTED")
		calibration = emptyOptimizeCalibration(run.invocation, discovery.Reason)
	} else if !resumed {
		if run.state.Resume.Accepted && run.previousState != nil &&
			run.previousState.Discovery.Status == optimizeDiscoveryComplete {
			discovery = run.previousState.Discovery
		} else {
			discoveryStarted := time.Now()
			discovery = run.discover(learningContext, exitCode, stderr)
			run.discoveryTime += time.Since(discoveryStarted)
		}
		learning, calibration = run.collectIncrementalLearning(discovery, exitCode)
	}
	portfolio := run.materializePortfolio(discovery, calibration)
	if run.centralReplay != nil {
		portfolio = run.centralReplay.portfolio
	} else if run.central != nil {
		if refreshedCalibration, refreshedPortfolio, refreshed := run.central.refreshQualifiedProfile(run, discovery); refreshed {
			calibration = refreshedCalibration
			portfolio = refreshedPortfolio
			learning = emptyOptimizeIncrementalLearning()
		}
	}
	selection := run.selection
	completedAt := time.Now().UTC()
	run.state.LastOutcome = optimizeOutcomeNative
	run.state.LastReason = calibration.Reason
	run.state.Phase = "NATIVE_RETAINED"
	if calibration.Status == optimizeCalibrationComplete && calibration.Qualified ||
		calibration.Status == optimizeCalibrationRemoteQualified &&
			portfolio.Reason == optimizePortfolioReasonRefreshed {
		run.state.LastOutcome = optimizeOutcomeLearning
		run.state.Phase = "QUALIFIED"
	} else if discovery.Status == optimizeDiscoveryComplete && calibration.Status == optimizeCalibrationSkipped {
		run.state.LastOutcome = optimizeOutcomeLearning
		if learning.Status == optimizeIncrementalCollecting {
			run.state.LastReason = learning.Reason
			run.state.Phase = "CALIBRATING"
		} else {
			run.state.Phase = "DISCOVERED"
		}
	}
	if portfolio.Status == optimizePortfolioComplete {
		run.state.LastReason = portfolio.Reason
	} else if calibration.Status == optimizeCalibrationComplete && calibration.Qualified && portfolio.Status == optimizePortfolioRetained {
		run.state.LastReason = portfolio.Reason
	}
	if selection.Selected {
		run.state.LastOutcome = "QUALIFIED_AND_USED"
		run.state.LastReason = selection.Reason
		run.state.Phase = "ACTIVE"
		if exitCode != 0 {
			run.state.LastReason = "SELECTED_BUILD_FAILED"
			run.state.Phase = "STALE"
		}
	}
	run.state.BuildStarted = run.childStarted
	run.state.LastExitCode = exitCode
	run.state.Discovery = discovery
	run.state.IncrementalLearning = learning
	run.state.Calibration = calibration
	run.state.Portfolio = portfolio
	run.state.Selection = selection
	previousValue := optimizeValueState{}
	if run.state.Resume.Accepted && run.previousState != nil {
		previousValue = run.previousState.Value
	}
	run.state.Value = nextOptimizeValueState(previousValue, calibration, selection, exitCode)
	if calibration.Status == optimizeCalibrationComplete {
		run.state.Bindings.Completeness = optimizeBindingCalibration
	}
	if portfolio.Status == optimizePortfolioComplete {
		run.state.Bindings.Completeness = optimizeBindingPortfolio
	}
	if selection.Selected {
		run.state.Bindings.Completeness = optimizeBindingReplay
	}
	run.state.UpdatedAt = completedAt.Format(time.RFC3339Nano)
	nativeAuthoritative := !selection.Selected && (!run.incrementalCandidate || run.incrementalFallback.started)
	nativeStarted := run.childStarted && nativeAuthoritative
	nativeExitCode := exitCode
	if !nativeAuthoritative {
		nativeExitCode = 0
	}
	result := optimizeResult{
		SchemaVersion: optimizeResultSchemaVersion,
		Outcome:       run.state.LastOutcome, Reason: run.state.LastReason,
		Phase:       run.state.Phase,
		StartedAt:   run.startedAt.Format(time.RFC3339Nano),
		CompletedAt: completedAt.Format(time.RFC3339Nano),
		DurationMs:  completedAt.Sub(run.startedAt).Milliseconds(),
		Generation:  run.state.Generation, Attempt: run.state.Attempt,
		Bindings: run.state.Bindings, Budget: run.state.Budget, Resume: run.state.Resume,
		Native: optimizeNativeResult{
			Authoritative: nativeAuthoritative,
			Started:       nativeStarted,
			ExitCode:      nativeExitCode,
		},
		Execution: optimizeExecutionResult{
			Mode:          run.executionMode(selection),
			Authoritative: true,
			Started:       run.childStarted,
			ExitCode:      exitCode,
		},
		Discovery:           discovery,
		IncrementalLearning: learning,
		Calibration:         calibration,
		Portfolio:           portfolio,
		Selection:           selection,
		Prequalification:    run.prequalification,
		Central:             disconnectedOptimizeCentralResult(),
		Value:               run.state.Value,
		GeneratedFiles: optimizeGeneratedFiles{
			State:         filepath.ToSlash(filepath.Join(run.invocation.stateRelative, optimizeStateFile)),
			Result:        filepath.ToSlash(filepath.Join(run.invocation.stateRelative, optimizeResultFile)),
			Discovery:     append([]string{}, discovery.GeneratedFiles...),
			Calibration:   append(append([]string{}, learning.GeneratedFiles...), calibration.GeneratedFiles...),
			Portfolio:     append([]string{}, portfolio.GeneratedFiles...),
			ValueJSON:     filepath.ToSlash(filepath.Join(run.invocation.stateRelative, optimizeValueReportJSONFile)),
			ValueMarkdown: filepath.ToSlash(filepath.Join(run.invocation.stateRelative, optimizeValueReportMDFile)),
		},
		ManualFilesRequired: 0, CalibrationPerformed: (calibration.Performed && !calibration.Reused) || run.incrementalObserved,
		PortfolioPerformed: portfolio.Performed && !portfolio.Reused, SelectionPerformed: selection.Performed,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	valueReport := newOptimizeValueReport(result)
	valueMarkdown := renderOptimizeValueMarkdown(valueReport)
	localStateStartedAt := time.Now()
	if err := writeCanonicalPrivateJSON(run.statePath, run.state); err != nil {
		return err
	}
	if err := writeCanonicalPrivateJSON(run.valueJSONPath, valueReport); err != nil {
		return err
	}
	if err := writePrivateAtomicFile(run.valueMDPath, valueMarkdown); err != nil {
		return err
	}
	run.localStateTime += time.Since(localStateStartedAt)
	if run.central != nil {
		centralStateStartedAt := time.Now()
		run.central.publish(run, stderr)
		run.centralStateTime += time.Since(centralStateStartedAt)
		result.Central = run.central.result
	}
	timingCompletedAt := time.Now().UTC()
	result.CompletedAt = timingCompletedAt.Format(time.RFC3339Nano)
	result.DurationMs = timingCompletedAt.Sub(run.startedAt).Milliseconds()
	result.NativeRetention = run.nativeRetentionResult(result.Reason, timingCompletedAt)
	result.Timing = run.timingResult(timingCompletedAt)
	// Publish the invocation result only after its customer-readable evidence is
	// complete, so a newly visible result never points at a partial report set.
	if err := writeCanonicalPrivateJSON(run.resultPath, result); err != nil {
		return err
	}
	if run.invocation.jsonOutput {
		raw, err := os.ReadFile(run.resultPath)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%s\n", raw)
		return err
	}
	_, err := fmt.Fprintf(
		stderr,
		"BuildOpt optimize: %s (%s)\nCurrent build: %s\nValue report: %s\nState: %s\nNext: %s; production authorization remains false\n",
		result.Outcome,
		result.Reason,
		run.executionDescription(selection),
		result.GeneratedFiles.ValueMarkdown,
		result.GeneratedFiles.Result,
		optimizeNextStep(discovery, learning, calibration, portfolio, selection),
	)
	return err
}

func (run *optimizeRun) timingResult(completedAt time.Time) optimizeTimingResult {
	total := completedAt.Sub(run.startedAt)
	preExecution := time.Duration(0)
	gradleExecution := time.Duration(0)
	if !run.childExecution.startedAt.IsZero() {
		preExecution = run.childExecution.startedAt.Sub(run.startedAt)
	}
	if !run.childExecution.startedAt.IsZero() && !run.childExecution.completedAt.IsZero() {
		gradleExecution = run.childExecution.completedAt.Sub(run.childExecution.startedAt)
	}
	finalization := time.Duration(0)
	if !run.finishStartedAt.IsZero() {
		finalization = completedAt.Sub(run.finishStartedAt)
	}
	preExecution = nonNegativeOptimizeDuration(preExecution)
	gradleExecution = nonNegativeOptimizeDuration(gradleExecution)
	finalization = nonNegativeOptimizeDuration(finalization)
	unattributed := total - preExecution - gradleExecution - finalization
	unattributed = nonNegativeOptimizeDuration(unattributed)
	return optimizeTimingResult{
		PreExecutionNS:    preExecution.Nanoseconds(),
		GradleExecutionNS: gradleExecution.Nanoseconds(),
		FinalizationNS:    finalization.Nanoseconds(),
		UnattributedNS:    unattributed.Nanoseconds(),
		TotalNS:           nonNegativeOptimizeDuration(total).Nanoseconds(),
		Diagnostics: optimizeTimingDiagnostics{
			GradleSetupNS:        nonNegativeOptimizeDuration(run.gradleSetupTime).Nanoseconds(),
			MatchingNS:           int64(run.selection.DurationNS),
			LocalStateNS:         nonNegativeOptimizeDuration(run.localStateTime).Nanoseconds(),
			CentralStateNS:       nonNegativeOptimizeDuration(run.centralStateTime).Nanoseconds(),
			MaterializationNS:    nonNegativeOptimizeDuration(run.materializationTime).Nanoseconds(),
			OutputVerificationNS: nonNegativeOptimizeDuration(run.outputVerificationTime).Nanoseconds(),
			DiscoveryLearningNS:  nonNegativeOptimizeDuration(run.discoveryTime).Nanoseconds(),
		},
	}
}

func nonNegativeOptimizeDuration(duration time.Duration) time.Duration {
	if duration < 0 {
		return 0
	}
	return duration
}

func (run *optimizeRun) nativeRetentionResult(reason string, completedAt time.Time) optimizeNativeRetentionResult {
	phase := run.retentionDecisionPhase
	if run.selection.Selected {
		phase = optimizeRetentionProfileSelected
	} else if phase == "" {
		phase = optimizeRetentionPostGradleDiscovery
	}
	pre, post, gradle := int64(0), int64(0), int64(0)
	if !run.childExecution.startedAt.IsZero() {
		pre = run.childExecution.startedAt.Sub(run.startedAt).Milliseconds()
	}
	if !run.childExecution.completedAt.IsZero() {
		gradle = run.childExecution.completedAt.Sub(run.childExecution.startedAt).Milliseconds()
		post = completedAt.Sub(run.childExecution.completedAt).Milliseconds()
	}
	if pre < 0 {
		pre = 0
	}
	if post < 0 {
		post = 0
	}
	if gradle < 0 {
		gradle = 0
	}
	return optimizeNativeRetentionResult{
		DecisionPhase: phase, Reason: reason,
		CompletedBeforeGradle: phase == optimizeRetentionPreGradleCompatibility ||
			phase == optimizeRetentionPreGradleEconomic || phase == optimizeRetentionProfileSelected,
		OutputObservationPrepared: run.outputObservation != nil,
		GradleDurationMS:          gradle, PreExecutionMS: pre, PostExecutionMS: post,
		WrapperOverheadMS: pre + post,
	}
}

func optimizeNextStep(discovery optimizeDiscoveryResult, learning optimizeIncrementalLearning, calibration optimizeCalibrationResult, portfolio optimizePortfolioResult, selection optimizeSelectionResult) string {
	if selection.Selected {
		return "repeat the same command while exact bindings remain valid; any drift retains native Gradle"
	}
	if portfolio.Status == optimizePortfolioComplete {
		return "match and replay an exact qualified family while native Gradle remains the fallback"
	}
	if calibration.Status == optimizeCalibrationComplete && calibration.Qualified {
		return "store the qualified change-family profile for automatic replay"
	}
	if learning.Status == optimizeIncrementalCollecting {
		return fmt.Sprintf(
			"run the same command again; %d/%d ordinary control/candidate pairs are complete and the next arm is %s",
			learning.PairsCompleted,
			learning.TargetPairs,
			learning.NextArm,
		)
	}
	if discovery.Status == optimizeDiscoveryComplete && !calibration.Performed {
		return "repeat the same command to collect exact-bound ordinary-build evidence"
	}
	return "native Gradle remains authoritative until discovery is safe"
}
