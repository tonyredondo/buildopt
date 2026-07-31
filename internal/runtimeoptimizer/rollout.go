package runtimeoptimizer

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	rolloutSchemaVersion = 1
	DirectRolloutClass   = "DIRECT_REVERSIBLE"
	ProofRolloutClass    = "PROOF_GATED"
	RolloutActive        = "ACTIVE"
	RolloutSuspended     = "SUSPENDED"
	RolloutRolledBack    = "ROLLED_BACK"
	KillSwitchSchema     = "buildopt.runtime-kill-switch/v1"
	MaximumWeeklyPercent = 5
	MaximumDailyPercent  = 10
)

// LearningBudget limits additional compute; natural control is not charged.
type LearningBudget struct {
	WeeklyPercent int `json:"weeklyPercent"`
	DailyPercent  int `json:"dailyPercent"`
}

// RunnerUsage is one idempotent natural or additional compute observation.
type RunnerUsage struct {
	EventID  string    `json:"eventId"`
	At       time.Time `json:"at"`
	RunnerMS int64     `json:"runnerMs"`
}

// BudgetReservation reserves additional runner time before scheduling.
type BudgetReservation struct {
	ReservationID    string    `json:"reservationId"`
	RepositoryID     string    `json:"repositoryId"`
	ReservedRunnerMS int64     `json:"reservedRunnerMs"`
	ActualRunnerMS   int64     `json:"actualRunnerMs"`
	State            string    `json:"state"`
	CreatedAt        time.Time `json:"createdAt"`
	CompletedAt      time.Time `json:"completedAt,omitempty"`
}

// RolloutAction is one progressively exposed reversible action.
type RolloutAction struct {
	ActionID             string `json:"actionId"`
	Class                string `json:"class"`
	Stage                string `json:"stage"`
	CandidateBasisPoints int    `json:"candidateBasisPoints"`
	ControlBasisPoints   int    `json:"controlBasisPoints"`
	State                string `json:"state"`
	SuspensionReason     string `json:"suspensionReason,omitempty"`
	Generation           int64  `json:"generation"`
}

// RolloutEvidence gates every forward stage transition.
type RolloutEvidence struct {
	CorrectnessPassed  bool
	SampleReady        bool
	BudgetAvailable    bool
	TelemetryComplete  bool
	ContractComplete   bool
	QuarantineComplete bool
	RevalidationPassed bool
}

// RolloutObservation contains non-compensable safety signals.
type RolloutObservation struct {
	ObservationID       string
	TelemetryComplete   bool
	DriftDetected       bool
	P95Regression       bool
	QueueRegression     bool
	OOM                 bool
	SustainedSwapping   bool
	ArtifactDivergence  bool
	AttributableFailure bool
}

// RolloutSelection is a pre-outcome candidate/control decision.
type RolloutSelection struct {
	Arm                   string `json:"arm"`
	ResourceProfileID     string `json:"resourceProfileId"`
	PropensityBasisPoints int    `json:"propensityBasisPoints"`
	Reason                string `json:"reason"`
}

// SelectionContext excludes releases, effects, urgent jobs, and local exploration.
type SelectionContext struct {
	AssignmentID       string
	SeedDigest         string
	CI                 bool
	Release            bool
	Urgent             bool
	ExternalEffects    bool
	CandidateProfileID string
}

// SignedKillSwitch is an Ed25519-authenticated monotonic repository directive.
type SignedKillSwitch struct {
	SchemaVersion string    `json:"schemaVersion"`
	RepositoryID  string    `json:"repositoryId"`
	Generation    int64     `json:"generation"`
	Enabled       bool      `json:"enabled"`
	Reason        string    `json:"reason"`
	IssuedAt      time.Time `json:"issuedAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
	KeyID         string    `json:"keyId"`
	Signature     string    `json:"signature"`
}

// FallbackContext describes the only facts that permit an automatic retry.
type FallbackContext struct {
	CandidateFailed       bool
	TaskActionsStarted    bool
	OriginalRetryCount    int
	IsolatedBaselineReady bool
	ManifestHasNoEffects  bool
	BaselinePassed        bool
}

// FallbackDecision states what remains authoritative after candidate failure.
type FallbackDecision struct {
	Action           string
	SuspendCandidate bool
	EnableKillSwitch bool
}

type repositoryRolloutState struct {
	Budget            LearningBudget                `json:"budget"`
	Natural           map[string]RunnerUsage        `json:"natural"`
	Additional        map[string]RunnerUsage        `json:"additional"`
	Reservations      map[string]BudgetReservation  `json:"reservations"`
	Actions           map[string]RolloutAction      `json:"actions"`
	Observations      map[string]RolloutObservation `json:"observations"`
	KillGeneration    int64                         `json:"killGeneration"`
	KillEnabled       bool                          `json:"killEnabled"`
	LastKillDirective *SignedKillSwitch             `json:"lastKillDirective,omitempty"`
}

type rolloutState struct {
	SchemaVersion int                               `json:"schemaVersion"`
	Repositories  map[string]repositoryRolloutState `json:"repositories"`
}

// RolloutController durably owns budget, rollout, rollback, and kill state.
type RolloutController struct {
	mutex sync.Mutex
	root  string
	path  string
	now   func() time.Time
	keys  map[string]ed25519.PublicKey
	state rolloutState
}

// OpenRolloutController opens a private durable controller with pinned keys.
func OpenRolloutController(root string, now func() time.Time, keys map[string]ed25519.PublicKey) (*RolloutController, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || now == nil || len(keys) == 0 {
		return nil, errors.New("open rollout controller: invalid configuration")
	}
	keyCopy := make(map[string]ed25519.PublicKey, len(keys))
	for id, key := range keys {
		if !identifierPattern.MatchString(id) || len(key) != ed25519.PublicKeySize {
			return nil, errors.New("open rollout controller: invalid trust key")
		}
		keyCopy[id] = append(ed25519.PublicKey(nil), key...)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("open rollout controller: unsafe state root")
	}
	controller := &RolloutController{root: root, path: filepath.Join(root, "rollout.json"), now: now, keys: keyCopy, state: rolloutState{SchemaVersion: rolloutSchemaVersion, Repositories: map[string]repositoryRolloutState{}}}
	if err := controller.load(); err != nil {
		return nil, err
	}
	return controller, nil
}

// ConfigureRepository creates an exact budget that may only tighten defaults.
func (controller *RolloutController) ConfigureRepository(repositoryID string, budget LearningBudget) error {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	if !identifierPattern.MatchString(repositoryID) || !validLearningBudget(budget) {
		return errors.New("configure rollout repository: invalid input")
	}
	if _, exists := controller.state.Repositories[repositoryID]; exists {
		return ErrConflict
	}
	controller.state.Repositories[repositoryID] = newRepositoryRolloutState(budget)
	if err := controller.persist(); err != nil {
		delete(controller.state.Repositories, repositoryID)
		return err
	}
	return nil
}

// RecordNaturalUsage records eligible natural runner time without charging it.
func (controller *RolloutController) RecordNaturalUsage(repositoryID string, usage RunnerUsage) error {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, ok := controller.state.Repositories[repositoryID]
	if !ok {
		return os.ErrNotExist
	}
	if !validRunnerUsage(usage, controller.now().UTC()) {
		return errors.New("record natural usage: invalid input")
	}
	if current, exists := repository.Natural[usage.EventID]; exists {
		if current != usage {
			return ErrConflict
		}
		return nil
	}
	repository.Natural[usage.EventID] = usage
	controller.state.Repositories[repositoryID] = repository
	return controller.persist()
}

// ReserveValidation reserves bounded additional compute before a lease exists.
func (controller *RolloutController) ReserveValidation(repositoryID, reservationID string, requestedRunnerMS int64) (BudgetReservation, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, ok := controller.state.Repositories[repositoryID]
	if !ok {
		return BudgetReservation{}, os.ErrNotExist
	}
	if !identifierPattern.MatchString(reservationID) || requestedRunnerMS <= 0 || requestedRunnerMS > int64(7*24*time.Hour/time.Millisecond) {
		return BudgetReservation{}, errors.New("reserve validation: invalid input")
	}
	if current, exists := repository.Reservations[reservationID]; exists {
		if current.ReservedRunnerMS != requestedRunnerMS {
			return BudgetReservation{}, ErrConflict
		}
		return current, nil
	}
	for _, reservation := range repository.Reservations {
		if reservation.State == "RESERVED" {
			return BudgetReservation{}, ErrRepositoryBusy
		}
	}
	now := controller.now().UTC()
	natural24, natural7 := usageWithin(repository.Natural, now, 24*time.Hour), usageWithin(repository.Natural, now, 7*24*time.Hour)
	additional24, additional7 := usageWithin(repository.Additional, now, 24*time.Hour), usageWithin(repository.Additional, now, 7*24*time.Hour)
	if exceedsBudget(additional24+requestedRunnerMS, natural24, repository.Budget.DailyPercent) || exceedsBudget(additional7+requestedRunnerMS, natural7, repository.Budget.WeeklyPercent) {
		return BudgetReservation{}, errors.New("reserve validation: learning budget exhausted")
	}
	reservation := BudgetReservation{ReservationID: reservationID, RepositoryID: repositoryID, ReservedRunnerMS: requestedRunnerMS, State: "RESERVED", CreatedAt: now}
	repository.Reservations[reservationID] = reservation
	controller.state.Repositories[repositoryID] = repository
	if err := controller.persist(); err != nil {
		delete(repository.Reservations, reservationID)
		controller.state.Repositories[repositoryID] = repository
		return BudgetReservation{}, err
	}
	return reservation, nil
}

// FinishValidation charges actual time or releases a cancelled reservation.
func (controller *RolloutController) FinishValidation(repositoryID, reservationID string, actualRunnerMS int64, cancelled bool) (BudgetReservation, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, ok := controller.state.Repositories[repositoryID]
	if !ok {
		return BudgetReservation{}, os.ErrNotExist
	}
	reservation, ok := repository.Reservations[reservationID]
	if !ok {
		return BudgetReservation{}, os.ErrNotExist
	}
	if reservation.State != "RESERVED" {
		wantState := "COMPLETED"
		if cancelled {
			wantState = "CANCELLED"
		}
		if reservation.State != wantState || reservation.ActualRunnerMS != actualRunnerMS {
			return BudgetReservation{}, ErrConflict
		}
		return reservation, nil
	}
	if actualRunnerMS < 0 || actualRunnerMS > reservation.ReservedRunnerMS || (cancelled && actualRunnerMS != 0) {
		return BudgetReservation{}, errors.New("finish validation: invalid usage")
	}
	reservation.ActualRunnerMS, reservation.CompletedAt = actualRunnerMS, controller.now().UTC()
	if cancelled {
		reservation.State = "CANCELLED"
	} else {
		reservation.State = "COMPLETED"
		if actualRunnerMS > 0 {
			repository.Additional[reservationID] = RunnerUsage{EventID: reservationID, At: reservation.CompletedAt, RunnerMS: actualRunnerMS}
		}
	}
	repository.Reservations[reservationID] = reservation
	controller.state.Repositories[repositoryID] = repository
	if err := controller.persist(); err != nil {
		return BudgetReservation{}, err
	}
	return reservation, nil
}

// StartAction starts at the first contractually valid stage.
func (controller *RolloutController) StartAction(repositoryID, actionID, class string, smokePassed bool) (RolloutAction, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, ok := controller.state.Repositories[repositoryID]
	if !ok {
		return RolloutAction{}, os.ErrNotExist
	}
	if !identifierPattern.MatchString(actionID) || (class != DirectRolloutClass && class != ProofRolloutClass) {
		return RolloutAction{}, errors.New("start rollout action: invalid input")
	}
	if current, exists := repository.Actions[actionID]; exists {
		if current.Class != class || (class == DirectRolloutClass && !smokePassed) {
			return RolloutAction{}, ErrConflict
		}
		return current, nil
	}
	action := RolloutAction{ActionID: actionID, Class: class, ControlBasisPoints: MinimumControlBasisPoints, State: RolloutActive, Generation: 1}
	if class == DirectRolloutClass {
		if !smokePassed {
			return RolloutAction{}, errors.New("start rollout action: smoke gate failed")
		}
		action.Stage, action.CandidateBasisPoints = "CANARY_5", 500
	} else {
		action.Stage = "SHADOW"
	}
	repository.Actions[actionID] = action
	controller.state.Repositories[repositoryID] = repository
	if err := controller.persist(); err != nil {
		return RolloutAction{}, err
	}
	return action, nil
}

// AdvanceAction moves exactly one stage and never skips a gate.
func (controller *RolloutController) AdvanceAction(repositoryID, actionID string, evidence RolloutEvidence) (RolloutAction, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, action, err := controller.action(repositoryID, actionID)
	if err != nil {
		return RolloutAction{}, err
	}
	if repository.KillEnabled || action.State != RolloutActive || !evidence.CorrectnessPassed || !evidence.SampleReady || !evidence.BudgetAvailable || !evidence.TelemetryComplete || !evidence.RevalidationPassed || (action.Class == ProofRolloutClass && (!evidence.ContractComplete || !evidence.QuarantineComplete)) {
		return action, nil
	}
	stages := directStages()
	if action.Class == ProofRolloutClass {
		stages = proofStages()
	}
	for index, stage := range stages {
		if action.Stage == stage.Name && index+1 < len(stages) {
			action.Stage, action.CandidateBasisPoints, action.Generation = stages[index+1].Name, stages[index+1].BasisPoints, action.Generation+1
			repository.Actions[actionID] = action
			controller.state.Repositories[repositoryID] = repository
			if err := controller.persist(); err != nil {
				return RolloutAction{}, err
			}
			return action, nil
		}
	}
	return action, nil
}

// SelectAction retains permanent control and excludes unsafe contexts.
func (controller *RolloutController) SelectAction(repositoryID, actionID string, context SelectionContext) (RolloutSelection, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, action, err := controller.action(repositoryID, actionID)
	if err != nil {
		return RolloutSelection{}, err
	}
	control := RolloutSelection{Arm: "CONTROL", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: BasisPointTotal, Reason: "CONSERVATIVE_FALLBACK"}
	if repository.KillEnabled {
		control.Reason = "KILL_SWITCH"
		return control, nil
	}
	if action.State != RolloutActive {
		control.Reason = action.State
		return control, nil
	}
	if !context.CI || context.Release || context.Urgent || context.ExternalEffects || action.Stage == "SHADOW" || !identifierPattern.MatchString(context.AssignmentID) || !validDigest(context.SeedDigest) || !isGoldenResourceArm(context.CandidateProfileID) || context.CandidateProfileID == "STABLE_CONTROL" {
		return control, nil
	}
	point := assignmentBasisPoint(CohortAssignmentRequest{AssignmentID: context.AssignmentID, RepositoryID: repositoryID, MeasurementEpoch: action.Stage, BucketDigest: context.SeedDigest, SeedDigest: context.SeedDigest})
	return selectRolloutPoint(action, point, context.CandidateProfileID), nil
}

// ObserveAction suspends immediately on any non-compensable signal.
func (controller *RolloutController) ObserveAction(repositoryID, actionID string, observation RolloutObservation) (RolloutAction, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, action, err := controller.action(repositoryID, actionID)
	if err != nil {
		return RolloutAction{}, err
	}
	if !identifierPattern.MatchString(observation.ObservationID) {
		return RolloutAction{}, errors.New("observe rollout: invalid observation")
	}
	if current, duplicate := repository.Observations[observation.ObservationID]; duplicate {
		if current != observation {
			return RolloutAction{}, ErrConflict
		}
		return action, nil
	}
	repository.Observations[observation.ObservationID] = observation
	reason := observationSuspensionReason(observation)
	if reason != "" && action.State == RolloutActive {
		action.State, action.Stage, action.CandidateBasisPoints, action.SuspensionReason, action.Generation = RolloutSuspended, "CONTROL", 0, reason, action.Generation+1
		repository.Actions[actionID] = action
	}
	controller.state.Repositories[repositoryID] = repository
	if err := controller.persist(); err != nil {
		return RolloutAction{}, err
	}
	return action, nil
}

// RollbackAction permanently restores the conservative profile for this generation.
func (controller *RolloutController) RollbackAction(repositoryID, actionID, reason string) (RolloutAction, error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, action, err := controller.action(repositoryID, actionID)
	if err != nil {
		return RolloutAction{}, err
	}
	if !identifierPattern.MatchString(reason) {
		return RolloutAction{}, errors.New("rollback rollout: invalid reason")
	}
	action.State, action.Stage, action.CandidateBasisPoints, action.SuspensionReason, action.Generation = RolloutRolledBack, "CONTROL", 0, reason, action.Generation+1
	repository.Actions[actionID] = action
	controller.state.Repositories[repositoryID] = repository
	if err := controller.persist(); err != nil {
		return RolloutAction{}, err
	}
	return action, nil
}

// ApplyKillSwitch verifies, persists, and applies a monotonic signed directive.
func (controller *RolloutController) ApplyKillSwitch(directive SignedKillSwitch) error {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	repository, ok := controller.state.Repositories[directive.RepositoryID]
	if !ok {
		return os.ErrNotExist
	}
	if err := verifyKillSwitch(directive, controller.keys, controller.now().UTC()); err != nil {
		return errors.New("apply kill switch: invalid or stale directive")
	}
	if directive.Generation == repository.KillGeneration && repository.LastKillDirective != nil && *repository.LastKillDirective == directive {
		return nil
	}
	if directive.Generation <= repository.KillGeneration {
		return errors.New("apply kill switch: invalid or stale directive")
	}
	repository.KillGeneration, repository.KillEnabled = directive.Generation, directive.Enabled
	copy := directive
	repository.LastKillDirective = &copy
	if directive.Enabled {
		for id, action := range repository.Actions {
			if action.State == RolloutActive {
				action.State, action.Stage, action.CandidateBasisPoints, action.SuspensionReason, action.Generation = RolloutSuspended, "CONTROL", 0, "SIGNED_KILL_SWITCH", action.Generation+1
				repository.Actions[id] = action
			}
		}
	}
	controller.state.Repositories[directive.RepositoryID] = repository
	return controller.persist()
}

// ResolveFallback applies the pre-task/side-effect safety boundary.
func ResolveFallback(context FallbackContext) FallbackDecision {
	if !context.CandidateFailed {
		return FallbackDecision{Action: "KEEP_CANDIDATE"}
	}
	if !context.TaskActionsStarted && context.OriginalRetryCount == 0 {
		return FallbackDecision{Action: "RETRY_ORIGINAL_ONCE", SuspendCandidate: true}
	}
	if context.IsolatedBaselineReady {
		if context.BaselinePassed {
			return FallbackDecision{Action: "RETURN_ISOLATED_BASELINE", SuspendCandidate: true}
		}
		return FallbackDecision{Action: "PRESERVE_BASELINE_FAILURE"}
	}
	if context.ManifestHasNoEffects {
		return FallbackDecision{Action: "RUN_ISOLATED_BASELINE", SuspendCandidate: true}
	}
	return FallbackDecision{Action: "PRESERVE_FAILURE", SuspendCandidate: true, EnableKillSwitch: true}
}

// KillSwitchSigningPayload returns the exact bytes covered by Ed25519.
func KillSwitchSigningPayload(directive SignedKillSwitch) ([]byte, error) {
	directive.Signature = ""
	return json.Marshal(directive)
}

type rolloutStage struct {
	Name        string
	BasisPoints int
}

func directStages() []rolloutStage {
	return []rolloutStage{{"CANARY_5", 500}, {"CANARY_25", 2500}, {"CANARY_50", 5000}, {"ACTIVE_95", 9500}}
}
func proofStages() []rolloutStage {
	return []rolloutStage{{"SHADOW", 0}, {"CANARY_1", 100}, {"CANARY_5", 500}, {"CANARY_25", 2500}, {"CANARY_50", 5000}, {"ACTIVE_95", 9500}}
}

func (controller *RolloutController) action(repositoryID, actionID string) (repositoryRolloutState, RolloutAction, error) {
	repository, ok := controller.state.Repositories[repositoryID]
	if !ok {
		return repositoryRolloutState{}, RolloutAction{}, os.ErrNotExist
	}
	action, ok := repository.Actions[actionID]
	if !ok {
		return repositoryRolloutState{}, RolloutAction{}, os.ErrNotExist
	}
	return repository, action, nil
}

func newRepositoryRolloutState(budget LearningBudget) repositoryRolloutState {
	return repositoryRolloutState{Budget: budget, Natural: map[string]RunnerUsage{}, Additional: map[string]RunnerUsage{}, Reservations: map[string]BudgetReservation{}, Actions: map[string]RolloutAction{}, Observations: map[string]RolloutObservation{}}
}

func validLearningBudget(budget LearningBudget) bool {
	return budget.WeeklyPercent >= 0 && budget.WeeklyPercent <= MaximumWeeklyPercent && budget.DailyPercent >= 0 && budget.DailyPercent <= MaximumDailyPercent
}
func validRunnerUsage(usage RunnerUsage, now time.Time) bool {
	return identifierPattern.MatchString(usage.EventID) && usage.RunnerMS > 0 && usage.RunnerMS <= int64(7*24*time.Hour/time.Millisecond) && !usage.At.IsZero() && !usage.At.After(now)
}

func usageWithin(events map[string]RunnerUsage, now time.Time, window time.Duration) int64 {
	start, total := now.Add(-window), int64(0)
	for _, event := range events {
		if event.At.After(start) && !event.At.After(now) {
			total += event.RunnerMS
		}
	}
	return total
}

func exceedsBudget(additional, natural int64, percent int) bool {
	return natural <= 0 || additional < 0 || additional*100 > natural*int64(percent)
}

func selectRolloutPoint(action RolloutAction, point int, candidateProfileID string) RolloutSelection {
	if point < action.ControlBasisPoints {
		return RolloutSelection{Arm: "CONTROL", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: BasisPointTotal - action.CandidateBasisPoints, Reason: "PERMANENT_CONTROL"}
	}
	if point < action.ControlBasisPoints+action.CandidateBasisPoints {
		return RolloutSelection{Arm: "CANDIDATE", ResourceProfileID: candidateProfileID, PropensityBasisPoints: action.CandidateBasisPoints, Reason: action.Stage}
	}
	return RolloutSelection{Arm: "CONTROL", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: BasisPointTotal - action.CandidateBasisPoints, Reason: "STAGE_HOLDOUT"}
}

func observationSuspensionReason(observation RolloutObservation) string {
	switch {
	case !observation.TelemetryComplete:
		return "INCOMPLETE_TELEMETRY"
	case observation.DriftDetected:
		return "DRIFT"
	case observation.ArtifactDivergence:
		return "ARTIFACT_DIVERGENCE"
	case observation.AttributableFailure:
		return "ATTRIBUTABLE_FAILURE"
	case observation.OOM:
		return "OOM"
	case observation.SustainedSwapping:
		return "SUSTAINED_SWAPPING"
	case observation.P95Regression:
		return "P95_REGRESSION"
	case observation.QueueRegression:
		return "QUEUE_REGRESSION"
	default:
		return ""
	}
}

func isGoldenResourceArm(arm string) bool {
	for _, profile := range GoldenResourceProfiles() {
		if profile.ProfileID == arm {
			return true
		}
	}
	return false
}

func verifyKillSwitch(directive SignedKillSwitch, keys map[string]ed25519.PublicKey, now time.Time) error {
	if directive.SchemaVersion != KillSwitchSchema || !identifierPattern.MatchString(directive.RepositoryID) || directive.Generation < 1 || !identifierPattern.MatchString(directive.Reason) || !identifierPattern.MatchString(directive.KeyID) || directive.IssuedAt.IsZero() || directive.ExpiresAt.IsZero() || directive.IssuedAt.After(now) || !directive.ExpiresAt.After(now) || directive.ExpiresAt.Sub(directive.IssuedAt) > 30*24*time.Hour {
		return errors.New("invalid kill switch fields")
	}
	key, ok := keys[directive.KeyID]
	if !ok {
		return errors.New("unknown kill switch key")
	}
	signature, err := base64.RawURLEncoding.DecodeString(directive.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("invalid kill switch signature")
	}
	payload, err := KillSwitchSigningPayload(directive)
	if err != nil || !ed25519.Verify(key, payload, signature) {
		return errors.New("invalid kill switch signature")
	}
	return nil
}

func (controller *RolloutController) load() error {
	file, err := os.Open(controller.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 16<<20 {
		return errors.New("open rollout controller: unsafe state file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 16<<20))
	decoder.DisallowUnknownFields()
	var state rolloutState
	if err := decoder.Decode(&state); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("open rollout controller: trailing state")
	}
	if state.SchemaVersion != rolloutSchemaVersion || !validRolloutState(state, controller.keys, controller.now().UTC()) {
		return errors.New("open rollout controller: unsupported state")
	}
	controller.state = state
	return nil
}

func validRolloutState(state rolloutState, keys map[string]ed25519.PublicKey, now time.Time) bool {
	if state.Repositories == nil {
		return false
	}
	for repositoryID, repository := range state.Repositories {
		if !identifierPattern.MatchString(repositoryID) || !validLearningBudget(repository.Budget) || repository.Natural == nil || repository.Additional == nil || repository.Reservations == nil || repository.Actions == nil || repository.Observations == nil || repository.KillGeneration < 0 {
			return false
		}
		for id, usage := range repository.Natural {
			if id != usage.EventID || !validRunnerUsage(usage, now) {
				return false
			}
		}
		for id, usage := range repository.Additional {
			if id != usage.EventID || !validRunnerUsage(usage, now) {
				return false
			}
		}
		for id, reservation := range repository.Reservations {
			if id != reservation.ReservationID || reservation.RepositoryID != repositoryID || reservation.ReservedRunnerMS <= 0 || reservation.ActualRunnerMS < 0 || reservation.ActualRunnerMS > reservation.ReservedRunnerMS || reservation.CreatedAt.IsZero() {
				return false
			}
			switch reservation.State {
			case "RESERVED":
				if !reservation.CompletedAt.IsZero() || reservation.ActualRunnerMS != 0 {
					return false
				}
			case "COMPLETED":
				if reservation.CompletedAt.IsZero() {
					return false
				}
			case "CANCELLED":
				if reservation.CompletedAt.IsZero() || reservation.ActualRunnerMS != 0 {
					return false
				}
			default:
				return false
			}
		}
		for id, action := range repository.Actions {
			if id != action.ActionID || !validRolloutAction(action) {
				return false
			}
		}
		for id, observation := range repository.Observations {
			if !identifierPattern.MatchString(id) || id != observation.ObservationID {
				return false
			}
		}
		if repository.LastKillDirective == nil {
			if repository.KillGeneration != 0 || repository.KillEnabled {
				return false
			}
		} else if repository.LastKillDirective.RepositoryID != repositoryID || repository.LastKillDirective.Generation != repository.KillGeneration || repository.LastKillDirective.Enabled != repository.KillEnabled || verifyKillSwitch(*repository.LastKillDirective, keys, repository.LastKillDirective.IssuedAt) != nil {
			return false
		}
	}
	return true
}

func validRolloutAction(action RolloutAction) bool {
	if !identifierPattern.MatchString(action.ActionID) || action.Generation < 1 || action.ControlBasisPoints != MinimumControlBasisPoints || (action.Class != DirectRolloutClass && action.Class != ProofRolloutClass) || (action.State != RolloutActive && action.State != RolloutSuspended && action.State != RolloutRolledBack) {
		return false
	}
	if action.State != RolloutActive {
		return action.Stage == "CONTROL" && action.CandidateBasisPoints == 0 && action.SuspensionReason != ""
	}
	stages := directStages()
	if action.Class == ProofRolloutClass {
		stages = proofStages()
	}
	for _, stage := range stages {
		if action.Stage == stage.Name {
			return action.CandidateBasisPoints == stage.BasisPoints
		}
	}
	return false
}

func (controller *RolloutController) persist() error {
	data, err := json.Marshal(controller.state)
	if err != nil || len(data) > 16<<20 {
		return errors.New("persist rollout controller: state exceeds bound")
	}
	file, err := os.CreateTemp(controller.root, ".rollout-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err = file.Write(append(data, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(temporary, controller.path); err != nil {
		return err
	}
	directory, err := os.Open(controller.root)
	if err != nil {
		return err
	}
	err = directory.Sync()
	return errors.Join(err, directory.Close())
}
