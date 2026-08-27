// Package stickytrial owns the bounded paired-trial runner for the sticky
// wrapper learning POC. A trial is an explicitly scheduled experiment, never
// part of an ordinary customer-requested build and never an authorization to
// activate an optimization.
package stickytrial

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	SchemaVersion = "buildopt.sticky/paired-trial/v1"
	RecordType    = "STICKY_WRAPPER_PAIRED_TRIAL"

	CandidateFirst Order = "CANDIDATE_FIRST"
	NativeFirst    Order = "NATIVE_FIRST"

	OutcomeSuccess        = "SUCCESS"
	OutcomeFailure        = "FAILURE"
	OutcomeCancelled      = "CANCELLED"
	OutcomeInfrastructure = "INFRA_FAILURE"

	ResultCandidateFaster = "CANDIDATE_FASTER"
	ResultNativeFaster    = "NATIVE_FASTER"
	ResultParity          = "PARITY"
	ResultInconclusive    = "INCONCLUSIVE"

	EquivalenceExact = "EXACT"
	EquivalenceNone  = "NONE"
)

// Order identifies which arm runs first in a pair.
type Order string

var (
	errNotTrustedCI    = errors.New("sticky trial scheduling requires trusted CI")
	errBudgetExhausted = errors.New("sticky trial learning budget is exhausted")
	errConcurrentTrial = errors.New("sticky trial already active for this repository")
	errReservation     = errors.New("sticky trial reservation is invalid")
	errIsolation       = errors.New("sticky trial isolation is invalid")
)

// Budget defines the additional compute that the POC may spend learning. The
// limit is MaxExtraPermille/1000 of NaturalRunnerNs (50 permille is five
// percent). MaxConcurrent is intentionally one for a repository.
type Budget struct {
	NaturalRunnerNs  int64
	MaxExtraPermille int64
	MaxConcurrent    int
	TrustedCI        bool
}

// BudgetSnapshot is an auditable view of reservations and actual trial cost.
type BudgetSnapshot struct {
	NaturalRunnerNs int64 `json:"naturalRunnerNs"`
	LimitNs         int64 `json:"limitNs"`
	ReservedNs      int64 `json:"reservedNs"`
	UsedNs          int64 `json:"usedNs"`
	ActiveTrials    int   `json:"activeTrials"`
	MaxConcurrent   int   `json:"maxConcurrent"`
	Exhausted       bool  `json:"exhausted"`
}

// Scheduler serializes assignments for one repository scope. Assignment is
// made before either arm runs, so observed timings cannot influence order.
type Scheduler struct {
	mu        sync.Mutex
	budget    Budget
	limitNs   int64
	reserved  int64
	used      int64
	active    int
	sequence  uint64
	exhausted bool
	activeIDs map[string]struct{}
}

// NewScheduler creates a scheduler only for the trusted CI path.
func NewScheduler(budget Budget) (*Scheduler, error) {
	if !budget.TrustedCI {
		return nil, errNotTrustedCI
	}
	if budget.NaturalRunnerNs < 1 || budget.MaxExtraPermille < 0 || budget.MaxExtraPermille > 1000 || budget.MaxConcurrent < 1 {
		return nil, errors.New("sticky trial budget is invalid")
	}
	if budget.MaxExtraPermille != 0 && budget.NaturalRunnerNs > (1<<63-1)/budget.MaxExtraPermille {
		return nil, errors.New("sticky trial budget overflows")
	}
	limit := budget.NaturalRunnerNs * budget.MaxExtraPermille / 1000
	if limit < 1 && budget.MaxExtraPermille > 0 {
		return nil, errors.New("sticky trial budget rounds below one nanosecond")
	}
	return &Scheduler{budget: budget, limitNs: limit, activeIDs: make(map[string]struct{})}, nil
}

// Snapshot returns the current budget without changing it.
func (scheduler *Scheduler) Snapshot() BudgetSnapshot {
	if scheduler == nil {
		return BudgetSnapshot{}
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	return scheduler.snapshotLocked()
}

func (scheduler *Scheduler) snapshotLocked() BudgetSnapshot {
	return BudgetSnapshot{
		NaturalRunnerNs: scheduler.budget.NaturalRunnerNs,
		LimitNs:         scheduler.limitNs,
		ReservedNs:      scheduler.reserved,
		UsedNs:          scheduler.used,
		ActiveTrials:    scheduler.active,
		MaxConcurrent:   scheduler.budget.MaxConcurrent,
		Exhausted:       scheduler.exhausted || scheduler.used >= scheduler.limitNs,
	}
}

// Assignment is immutable scheduling evidence. The unexported scheduler link
// prevents a caller from fabricating completion against another repository.
type Assignment struct {
	TrialID         string `json:"trialId"`
	Sequence        uint64 `json:"sequence"`
	Pair            int    `json:"pair"`
	Order           Order  `json:"order"`
	ReservedExtraNs int64  `json:"reservedExtraNs"`
	NaturalRunnerNs int64  `json:"naturalRunnerNs"`
	BudgetLimitNs   int64  `json:"budgetLimitNs"`
	assigned        bool
	scheduler       *Scheduler
}

// Assign reserves estimated compute before a trial can run. Pair parity is
// the only input to order, giving equal candidate-first/native-first arms.
func (scheduler *Scheduler) Assign(trialID string, estimatedExtraNs int64) (Assignment, error) {
	if scheduler == nil || trialID == "" || estimatedExtraNs < 1 {
		return Assignment{}, errReservation
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	if scheduler.exhausted || scheduler.used >= scheduler.limitNs || scheduler.reserved > scheduler.limitNs-scheduler.used {
		return Assignment{}, errBudgetExhausted
	}
	if scheduler.active >= scheduler.budget.MaxConcurrent {
		return Assignment{}, errConcurrentTrial
	}
	if estimatedExtraNs > scheduler.limitNs-scheduler.used-scheduler.reserved {
		return Assignment{}, errBudgetExhausted
	}
	if _, exists := scheduler.activeIDs[trialID]; exists {
		return Assignment{}, errReservation
	}
	scheduler.sequence++
	pair := int(scheduler.sequence)
	order := NativeFirst
	if pair%2 == 1 {
		order = CandidateFirst
	}
	scheduler.reserved += estimatedExtraNs
	scheduler.active++
	scheduler.activeIDs[trialID] = struct{}{}
	return Assignment{
		TrialID: trialID, Sequence: scheduler.sequence, Pair: pair, Order: order,
		ReservedExtraNs: estimatedExtraNs, NaturalRunnerNs: scheduler.budget.NaturalRunnerNs,
		BudgetLimitNs: scheduler.limitNs, assigned: true, scheduler: scheduler,
	}, nil
}

func (assignment Assignment) valid() bool {
	return assignment.assigned && assignment.scheduler != nil && assignment.TrialID != "" && assignment.Pair > 0 && assignment.ReservedExtraNs > 0
}

// Complete charges actual paired compute and releases any unused reservation.
// A run that exceeded its reservation fails closed and exhausts scheduling.
func (scheduler *Scheduler) Complete(assignment Assignment, actualExtraNs int64) (BudgetSnapshot, error) {
	if scheduler == nil || assignment.scheduler != scheduler || !assignment.valid() || actualExtraNs < 0 {
		return BudgetSnapshot{}, errReservation
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	if _, exists := scheduler.activeIDs[assignment.TrialID]; !exists {
		return scheduler.snapshotLocked(), errReservation
	}
	delete(scheduler.activeIDs, assignment.TrialID)
	scheduler.active--
	scheduler.reserved -= assignment.ReservedExtraNs
	if actualExtraNs > assignment.ReservedExtraNs {
		scheduler.used += actualExtraNs
		scheduler.exhausted = true
		return scheduler.snapshotLocked(), fmt.Errorf("%w: actual %d ns exceeded reservation %d ns", errBudgetExhausted, actualExtraNs, assignment.ReservedExtraNs)
	}
	if actualExtraNs > scheduler.limitNs-scheduler.used {
		scheduler.used += actualExtraNs
		scheduler.exhausted = true
		return scheduler.snapshotLocked(), errBudgetExhausted
	}
	scheduler.used += actualExtraNs
	if scheduler.used >= scheduler.limitNs {
		scheduler.exhausted = true
	}
	return scheduler.snapshotLocked(), nil
}

// Cancel records compute already spent by a cancelled trial and releases the
// unused reservation, allowing a later trial only if budget remains.
func (scheduler *Scheduler) Cancel(assignment Assignment, actualExtraNs int64) (BudgetSnapshot, error) {
	return scheduler.Complete(assignment, actualExtraNs)
}

// Isolation names the writable roots for both arms. Every path must be
// absolute, clean and distinct; this prevents a candidate from warming or
// reading the native arm's Gradle/cache/state directories.
type Isolation struct {
	CandidateDir        string `json:"candidateDir"`
	NativeDir           string `json:"nativeDir"`
	CandidateGradleHome string `json:"candidateGradleHome"`
	NativeGradleHome    string `json:"nativeGradleHome"`
	CandidateCache      string `json:"candidateCache"`
	NativeCache         string `json:"nativeCache"`
	CandidateState      string `json:"candidateState"`
	NativeState         string `json:"nativeState"`
}

func (isolation Isolation) paths() []string {
	return []string{isolation.CandidateDir, isolation.NativeDir, isolation.CandidateGradleHome, isolation.NativeGradleHome, isolation.CandidateCache, isolation.NativeCache, isolation.CandidateState, isolation.NativeState}
}

func validateIsolation(isolation Isolation) error {
	seen := make(map[string]struct{}, len(isolation.paths()))
	for _, path := range isolation.paths() {
		if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return errIsolation
		}
		if _, exists := seen[path]; exists {
			return fmt.Errorf("%w: paths are not distinct", errIsolation)
		}
		seen[path] = struct{}{}
	}
	return nil
}

func isolationDigest(isolation Isolation) string {
	return Digest(isolation.CandidateDir, isolation.NativeDir, isolation.CandidateGradleHome, isolation.NativeGradleHome, isolation.CandidateCache, isolation.NativeCache, isolation.CandidateState, isolation.NativeState)
}

// Command is executed directly, without a shell. Env is the complete child
// environment, which makes hidden state and credentials auditable by callers.
type Command struct {
	Program string
	Args    []string
	Dir     string
	Env     []string
}

// Executor runs one isolated arm. The launcher supplies this adapter so every
// customer-triggered process retains its process-group and signal semantics.
type Executor func(context.Context, Command, Isolation, []string) ArmResult

func validateCommand(command Command, expectedDir string) error {
	if command.Program == "" || command.Dir == "" || command.Dir != expectedDir || !filepath.IsAbs(command.Dir) || filepath.Clean(command.Dir) != command.Dir || len(command.Env) == 0 {
		return errors.New("sticky trial command is not explicit")
	}
	for _, value := range command.Env {
		if !strings.Contains(value, "=") {
			return errors.New("sticky trial environment is not explicit")
		}
	}
	return nil
}

// ArmResult records one direct command invocation and its required-output
// digest. Logs are intentionally not persisted in the report.
type ArmResult struct {
	Outcome      string `json:"outcome"`
	ExitCode     int    `json:"exitCode"`
	DurationNs   int64  `json:"durationNs"`
	OutputSHA256 string `json:"outputSha256,omitempty"`
	OutputBytes  int64  `json:"outputBytes,omitempty"`
	diagnostic   string
}

// Diagnostic returns a bounded non-authoritative failure message for a local
// checker. It is intentionally omitted from persisted reports.
func (result ArmResult) Diagnostic() string {
	return result.diagnostic
}

// PairedTrial is immutable evidence from one candidate/native pair. It is
// descriptive only: later qualification still needs the decision-store gate.
type PairedTrial struct {
	SchemaVersion       string    `json:"schemaVersion"`
	RecordType          string    `json:"recordType"`
	TrialID             string    `json:"trialId"`
	Sequence            uint64    `json:"sequence"`
	Pair                int       `json:"pair"`
	Order               Order     `json:"order"`
	AssignmentBeforeRun bool      `json:"assignmentBeforeRun"`
	IsolationDigest     string    `json:"isolationDigest"`
	InvocationCount     int       `json:"invocationCount"`
	Candidate           ArmResult `json:"candidate"`
	Native              ArmResult `json:"native"`
	Equivalence         string    `json:"equivalence"`
	Result              string    `json:"result"`
	NaturalRunnerNs     int64     `json:"naturalRunnerNs"`
	BudgetLimitNs       int64     `json:"budgetLimitNs"`
	ReservedExtraNs     int64     `json:"reservedExtraNs"`
	ActualExtraNs       int64     `json:"actualExtraNs"`
	StartedAt           string    `json:"startedAt"`
	CompletedAt         string    `json:"completedAt"`
	Cancellation        bool      `json:"cancellation"`
}

// RunPaired executes exactly two arms in the preassigned order. A cancelled
// command releases unused budget and retains the native result semantics.
func RunPaired(ctx context.Context, assignment Assignment, isolation Isolation, candidate, native Command, outputs []string) (PairedTrial, error) {
	return RunPairedWithExecutor(ctx, assignment, isolation, candidate, native, outputs, runArm)
}

// RunPairedWithExecutor preserves scheduling, isolation, equivalence and
// budget accounting while delegating only direct process execution.
func RunPairedWithExecutor(ctx context.Context, assignment Assignment, isolation Isolation, candidate, native Command, outputs []string, execute Executor) (PairedTrial, error) {
	if ctx == nil || !assignment.valid() || assignment.Pair != int(assignment.Sequence) {
		return PairedTrial{}, errReservation
	}
	if execute == nil {
		return PairedTrial{}, errors.New("sticky trial executor is nil")
	}
	if err := validateIsolation(isolation); err != nil {
		return PairedTrial{}, err
	}
	if len(outputs) == 0 {
		return PairedTrial{}, errors.New("sticky trial requires required outputs")
	}
	if err := validateCommand(candidate, isolation.CandidateDir); err != nil {
		return PairedTrial{}, err
	}
	if err := validateCommand(native, isolation.NativeDir); err != nil {
		return PairedTrial{}, err
	}
	started := time.Now().UTC()
	trial := PairedTrial{
		SchemaVersion: SchemaVersion, RecordType: RecordType, TrialID: assignment.TrialID,
		Sequence: assignment.Sequence, Pair: assignment.Pair, Order: assignment.Order,
		AssignmentBeforeRun: true, IsolationDigest: isolationDigest(isolation), InvocationCount: 0,
		NaturalRunnerNs: assignment.NaturalRunnerNs, BudgetLimitNs: assignment.BudgetLimitNs,
		ReservedExtraNs: assignment.ReservedExtraNs, StartedAt: started.Format(time.RFC3339Nano),
	}
	var first, second ArmResult
	var firstCommand, secondCommand Command
	if assignment.Order == CandidateFirst {
		firstCommand, secondCommand = candidate, native
	} else if assignment.Order == NativeFirst {
		firstCommand, secondCommand = native, candidate
	} else {
		_, _ = assignment.scheduler.Cancel(assignment, 0)
		return PairedTrial{}, errors.New("sticky trial order is invalid")
	}
	first = execute(ctx, firstCommand, isolation, outputs)
	trial.InvocationCount++
	second = execute(ctx, secondCommand, isolation, outputs)
	trial.InvocationCount++
	if assignment.Order == CandidateFirst {
		trial.Candidate, trial.Native = first, second
	} else {
		trial.Native, trial.Candidate = first, second
	}
	trial.Cancellation = trial.Candidate.Outcome == OutcomeCancelled || trial.Native.Outcome == OutcomeCancelled
	trial.ActualExtraNs = trial.Candidate.DurationNs + trial.Native.DurationNs
	if trial.Candidate.Outcome == OutcomeSuccess && trial.Native.Outcome == OutcomeSuccess && trial.Candidate.OutputSHA256 != "" && trial.Candidate.OutputSHA256 == trial.Native.OutputSHA256 {
		trial.Equivalence = EquivalenceExact
		switch {
		case trial.Candidate.DurationNs < trial.Native.DurationNs:
			trial.Result = ResultCandidateFaster
		case trial.Candidate.DurationNs > trial.Native.DurationNs:
			trial.Result = ResultNativeFaster
		default:
			trial.Result = ResultParity
		}
	} else {
		trial.Equivalence = EquivalenceNone
		trial.Result = ResultInconclusive
	}
	trial.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	_, budgetErr := assignment.scheduler.Complete(assignment, trial.ActualExtraNs)
	if budgetErr != nil {
		return trial, budgetErr
	}
	if trial.Cancellation || ctx.Err() != nil {
		return trial, context.Canceled
	}
	if trial.Candidate.Outcome == OutcomeInfrastructure || trial.Native.Outcome == OutcomeInfrastructure {
		return trial, errors.New("sticky trial infrastructure command failed")
	}
	return trial, nil
}

func runArm(ctx context.Context, command Command, isolation Isolation, outputs []string) ArmResult {
	started := time.Now()
	process := exec.CommandContext(ctx, command.Program, command.Args...)
	process.Dir = command.Dir
	process.Env = append([]string(nil), command.Env...)
	var stdout, stderr bytes.Buffer
	process.Stdout = &stdout
	process.Stderr = &stderr
	err := process.Run()
	duration := time.Since(started).Nanoseconds()
	result := ArmResult{DurationNs: duration}
	if err == nil {
		result.Outcome = OutcomeSuccess
	} else if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.Outcome = OutcomeCancelled
		result.ExitCode = 130
	} else if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
		result.Outcome = OutcomeFailure
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.Outcome = OutcomeInfrastructure
		result.ExitCode = 127
	}
	if result.Outcome == OutcomeSuccess {
		digest, size, hashErr := HashOutputs(command.Dir, outputs)
		if hashErr != nil {
			result.Outcome = OutcomeInfrastructure
			result.ExitCode = 125
		} else {
			result.OutputSHA256, result.OutputBytes = digest, size
		}
	}
	if result.Outcome != OutcomeSuccess {
		diagnostic := strings.TrimSpace(stderr.String())
		if diagnostic == "" {
			diagnostic = strings.TrimSpace(errString(err))
		}
		if len(diagnostic) > 1024 {
			diagnostic = diagnostic[len(diagnostic)-1024:]
		}
		result.diagnostic = diagnostic
	}
	_ = isolation
	_ = stdout
	return result
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Report is the aggregate emitted by the real checker. It retains raw pairs
// and budget accounting so a later evaluator can recompute every statistic.
type Report struct {
	SchemaVersion    string         `json:"schemaVersion"`
	RecordType       string         `json:"recordType"`
	CapturedAt       string         `json:"capturedAt"`
	GradleVersion    string         `json:"gradleVersion"`
	TrustedCI        bool           `json:"trustedCI"`
	OrderPolicy      string         `json:"orderPolicy"`
	NoLookahead      bool           `json:"noLookahead"`
	InvocationPolicy string         `json:"invocationPolicy"`
	IsolationPolicy  string         `json:"isolationPolicy"`
	Budget           BudgetSnapshot `json:"budget"`
	Trials           []PairedTrial  `json:"trials"`
	CandidateMeanMs  float64        `json:"candidateMeanMs"`
	NativeMeanMs     float64        `json:"nativeMeanMs"`
	MeanSavedMs      float64        `json:"meanSavedMs"`
	PositivePairs    int            `json:"positivePairs"`
	ExactOutputPairs int            `json:"exactOutputPairs"`
	CancelledPairs   int            `json:"cancelledPairs"`
	TotalInvocations int            `json:"totalInvocations"`
}

// Aggregate builds an immutable report from complete or inconclusive trials.
func Aggregate(snapshot BudgetSnapshot, trials []PairedTrial, capturedAt time.Time) (Report, error) {
	if len(trials) == 0 {
		return Report{}, errors.New("sticky trial report has no trials")
	}
	var candidateTotal, nativeTotal int64
	var completed int
	report := Report{
		SchemaVersion: SchemaVersion, RecordType: RecordType, CapturedAt: capturedAt.UTC().Format(time.RFC3339Nano),
		TrustedCI: true, OrderPolicy: "ALTERNATING_PAIR_PARITY", NoLookahead: true,
		InvocationPolicy: "EXACTLY_TWO_DIRECT_COMMANDS_PER_TRIAL", IsolationPolicy: "EIGHT_DISTINCT_PRIVATE_ROOTS",
		Budget: snapshot, Trials: append([]PairedTrial(nil), trials...),
	}
	for _, trial := range trials {
		if err := trial.Validate(); err != nil {
			return Report{}, err
		}
		report.TotalInvocations += trial.InvocationCount
		if trial.Cancellation {
			report.CancelledPairs++
		}
		if trial.Equivalence == EquivalenceExact {
			report.ExactOutputPairs++
			candidateTotal += trial.Candidate.DurationNs
			nativeTotal += trial.Native.DurationNs
			completed++
			if trial.Result == ResultCandidateFaster {
				report.PositivePairs++
			}
		}
	}
	if completed > 0 {
		report.CandidateMeanMs = float64(candidateTotal) / float64(completed) / 1e6
		report.NativeMeanMs = float64(nativeTotal) / float64(completed) / 1e6
		report.MeanSavedMs = report.NativeMeanMs - report.CandidateMeanMs
	}
	return report, nil
}

// Validate checks one report record without making a qualification decision.
func (trial PairedTrial) Validate() error {
	if trial.SchemaVersion != SchemaVersion || trial.RecordType != RecordType || trial.TrialID == "" || trial.Sequence == 0 || trial.Pair != int(trial.Sequence) || (trial.Order != CandidateFirst && trial.Order != NativeFirst) || !trial.AssignmentBeforeRun || trial.InvocationCount != 2 || trial.NaturalRunnerNs < 1 || trial.BudgetLimitNs < 1 || trial.ReservedExtraNs < 1 || trial.ActualExtraNs < 0 || trial.CompletedAt == "" || trial.StartedAt == "" {
		return errors.New("sticky trial record shape is invalid")
	}
	if trial.Candidate.DurationNs < 0 || trial.Native.DurationNs < 0 || trial.ActualExtraNs != trial.Candidate.DurationNs+trial.Native.DurationNs {
		return errors.New("sticky trial timing is invalid")
	}
	if trial.Equivalence == EquivalenceExact {
		if trial.Candidate.Outcome != OutcomeSuccess || trial.Native.Outcome != OutcomeSuccess || !validDigest(trial.Candidate.OutputSHA256) || trial.Candidate.OutputSHA256 != trial.Native.OutputSHA256 || trial.Result == ResultInconclusive {
			return errors.New("exact sticky trial lacks equivalent successful outputs")
		}
	} else if trial.Result != ResultInconclusive {
		return errors.New("non-equivalent sticky trial has a value result")
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

// HashOutputs hashes required files/directories by sorted relative path and
// bytes. Symlinks and missing outputs are rejected to keep equivalence exact.
func HashOutputs(root string, outputs []string) (string, int64, error) {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root || len(outputs) == 0 {
		return "", 0, errors.New("output root is invalid")
	}
	entries := make(map[string][]byte)
	for _, output := range outputs {
		if output == "" || filepath.IsAbs(output) || filepath.Clean(output) != output || output == "." || strings.HasPrefix(output, ".."+string(filepath.Separator)) || output == ".." {
			return "", 0, errors.New("required output path is invalid")
		}
		path := filepath.Join(root, output)
		info, err := os.Lstat(path)
		if err != nil {
			return "", 0, fmt.Errorf("required output is missing: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", 0, errors.New("required output may not be a symlink")
		}
		if info.IsDir() {
			err = filepath.Walk(path, func(current string, currentInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if currentInfo.Mode()&os.ModeSymlink != 0 {
					return errors.New("required output tree contains a symlink")
				}
				if currentInfo.IsDir() {
					return nil
				}
				data, readErr := os.ReadFile(current)
				if readErr != nil {
					return readErr
				}
				relative, relErr := filepath.Rel(root, current)
				if relErr != nil {
					return relErr
				}
				entries[filepath.ToSlash(relative)] = data
				return nil
			})
		} else {
			var data []byte
			data, err = os.ReadFile(path)
			if err == nil {
				entries[filepath.ToSlash(output)] = data
			}
		}
		if err != nil {
			return "", 0, err
		}
	}
	if len(entries) == 0 {
		return "", 0, errors.New("required output set is empty")
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hasher := sha256.New()
	var size int64
	for _, key := range keys {
		data := entries[key]
		_, _ = io.WriteString(hasher, key)
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(data)
		_, _ = hasher.Write([]byte{0})
		size += int64(len(data))
	}
	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}

// Digest creates a stable SHA-256 over ordered strings.
func Digest(values ...string) string {
	hasher := sha256.New()
	for _, value := range values {
		_, _ = hasher.Write([]byte(value))
		_, _ = hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// MarshalReport uses stable indentation for reviewable checked-in evidence.
func MarshalReport(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
