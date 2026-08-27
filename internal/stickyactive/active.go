// Package stickyactive owns the fail-closed execution boundary for the
// sticky-wrapper learning POC. It is deliberately generic: an active profile
// supplies direct commands, an exact decision binding and required outputs;
// no repository, task or path rule is embedded here.
package stickyactive

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickytrial"
)

const (
	SchemaVersion = "buildopt.sticky/active-execution/v1"
	RecordType    = "STICKY_WRAPPER_ACTIVE_EXECUTION"

	StatusNativeRetained = "NATIVE_RETAINED"
	StatusActiveExecuted = "ACTIVE_EXECUTED"
	StatusSuspended     = "SUSPENDED"

	SelectedNative    = "NATIVE"
	SelectedCandidate = "CANDIDATE"

	ReasonBypass              = "BYPASS"
	ReasonPreviouslySuspended = "PROFILE_ALREADY_SUSPENDED"
	ReasonInvalidDecision     = "DECISION_INVALID"
	ReasonExpiredDecision     = "DECISION_EXPIRED"
	ReasonRevokedDecision     = "DECISION_REVOKED"
	ReasonBindingMismatch     = "DECISION_BINDING_MISMATCH"
	ReasonUnsupportedDecision = "DECISION_NOT_RUNTIME_ACTIVE"
	ReasonCandidateFailure    = "CANDIDATE_FAILURE"
	ReasonNativeFailure       = "NATIVE_COUNTERFACTUAL_FAILURE"
	ReasonOutputMismatch      = "OUTPUT_MISMATCH"
	ReasonRegression          = "CANDIDATE_REGRESSION"
	ReasonCancelled           = "INVOCATION_CANCELLED"
	ReasonActive              = "ACTIVE_COUNTERFACTUAL_PASSED"
	ReasonNoCounterfactual    = "COUNTERFACTUAL_NOT_DUE"

	OutcomeSuccess        = "SUCCESS"
	OutcomeFailure        = "FAILURE"
	OutcomeCancelled      = "CANCELLED"
	OutcomeInfrastructure = "INFRA_FAILURE"
)

// Command is a complete direct process invocation. Program must be an
// absolute executable path; no shell, PATH lookup or implicit environment is
// allowed at this boundary.
type Command struct {
	Program string
	Args    []string
	Dir     string
	Env     []string
}

// Profile is an action candidate plus the signed decision that authorizes it.
// RequiredOutputs are relative paths below each command's own directory and
// are hashed byte-for-byte before a candidate can receive credit.
type Profile struct {
	ActionID                    string
	DecisionRaw                 []byte
	ExpectedBinding             stickydecision.Binding
	PublicKeys                  map[string]ed25519.PublicKey
	RevocationEpoch             int64
	Candidate                   Command
	Native                      Command
	RequiredOutputs             []string
	CounterfactualEvery         uint64
	RegressionTolerancePermille uint64
	Now                         func() time.Time
}

// commandExecutor separates lifecycle policy from process timing. New always
// installs runCommand; package tests can inject deterministic arm evidence.
type commandExecutor func(context.Context, Command, []string) ArmResult

// ArmResult records one command attempt. Diagnostics stay in memory and are
// never written to evidence because they can contain repository details.
type ArmResult struct {
	Outcome      string `json:"outcome"`
	ExitCode     int    `json:"exitCode"`
	DurationNs   int64  `json:"durationNs"`
	OutputSHA256 string `json:"outputSha256,omitempty"`
	OutputBytes  int64  `json:"outputBytes,omitempty"`
	diagnostic   string
}

// Diagnostic returns a bounded local failure detail for a checker or caller.
func (result ArmResult) Diagnostic() string { return result.diagnostic }

// Execution is the auditable result of one wrapper invocation. Native is
// always retained as the authoritative fallback; it is present when a native
// counterfactual or fallback was run.
type Execution struct {
	SchemaVersion       string     `json:"schemaVersion"`
	RecordType          string     `json:"recordType"`
	Invocation          uint64     `json:"invocation"`
	StartedAt            string     `json:"startedAt"`
	CompletedAt          string     `json:"completedAt"`
	Status               string     `json:"status"`
	Reason               string     `json:"reason"`
	Decision             string     `json:"decision"`
	ActionID             string     `json:"actionId,omitempty"`
	Selected             string     `json:"selected"`
	CandidateExecuted    bool       `json:"candidateExecuted"`
	Counterfactual      bool       `json:"counterfactual"`
	Suspended           bool       `json:"suspended"`
	ExactOutputs        bool       `json:"exactOutputs"`
	SavingNs            int64      `json:"savingNs"`
	Candidate           *ArmResult `json:"candidate,omitempty"`
	Native              *ArmResult `json:"native,omitempty"`
}

// Qualification is the result of applying the immutable activation gate to a
// paired-trial report. A negative or incomplete report can never authorize an
// active profile.
type Qualification struct {
	Authorized       bool    `json:"authorized"`
	Reason           string  `json:"reason"`
	Pairs            int     `json:"pairs"`
	MinimumPairs     int     `json:"minimumPairs"`
	PositivePairs    int     `json:"positivePairs"`
	ExactOutputPairs int     `json:"exactOutputPairs"`
	CancelledPairs   int     `json:"cancelledPairs"`
	CandidateMeanMs  float64 `json:"candidateMeanMs"`
	NativeMeanMs     float64 `json:"nativeMeanMs"`
	MeanSavedMs     float64 `json:"meanSavedMs"`
}

const (
	QualificationAuthorized   = "QUALIFIED_PROFITABLE"
	QualificationInsufficient = "INSUFFICIENT_PAIRS"
	QualificationInexact      = "OUTPUTS_NOT_EXACT"
	QualificationCancelled    = "CANCELLED_PAIR"
	QualificationNegative     = "TRIAL_NOT_PROFITABLE"
	QualificationIncomplete   = "TRIAL_INCONCLUSIVE"
)

// QualifyTrial applies a conservative POC gate. Every pair must be successful,
// exact and candidate-positive; this intentionally rejects SWL-010's current
// 0/4 result rather than turning an ambiguous average into authority.
func QualifyTrial(report stickytrial.Report, minimumPairs int) Qualification {
	if minimumPairs < 1 {
		minimumPairs = 1
	}
	result := Qualification{
		Pairs: len(report.Trials), MinimumPairs: minimumPairs,
		PositivePairs: report.PositivePairs, ExactOutputPairs: report.ExactOutputPairs,
		CancelledPairs: report.CancelledPairs, CandidateMeanMs: report.CandidateMeanMs,
		NativeMeanMs: report.NativeMeanMs, MeanSavedMs: report.MeanSavedMs,
	}
	switch {
	case result.Pairs < minimumPairs:
		result.Reason = QualificationInsufficient
	case result.ExactOutputPairs != result.Pairs:
		result.Reason = QualificationInexact
	case result.CancelledPairs != 0:
		result.Reason = QualificationCancelled
	case result.PositivePairs != result.Pairs || result.MeanSavedMs <= 0:
		result.Reason = QualificationNegative
	case report.TotalInvocations != result.Pairs*2:
		result.Reason = QualificationIncomplete
	default:
		result.Authorized = true
		result.Reason = QualificationAuthorized
	}
	return result
}

// Runner serializes active invocations for one profile. Once suspended, it
// remains suspended until the owner supplies a new profile generation.
type Runner struct {
	mu        sync.Mutex
	profile   Profile
	execute commandExecutor
	invocation uint64
	suspended bool
}

// New validates and defensively copies a profile before any child process can
// run. CounterfactualEvery must be non-zero so active value is periodically
// compared with authoritative native Gradle.
func New(profile Profile) (*Runner, error) {
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	if profile.Now == nil {
		profile.Now = time.Now
	}
	profile.DecisionRaw = append([]byte(nil), profile.DecisionRaw...)
	profile.RequiredOutputs = append([]string(nil), profile.RequiredOutputs...)
	profile.Candidate = copyCommand(profile.Candidate)
	profile.Native = copyCommand(profile.Native)
	profile.PublicKeys = copyKeys(profile.PublicKeys)
	return &Runner{profile: profile, execute: runCommand}, nil
}

func validateProfile(profile Profile) error {
	if profile.ActionID == "" || len(profile.DecisionRaw) == 0 || len(profile.PublicKeys) == 0 ||
		profile.CounterfactualEvery == 0 || profile.RegressionTolerancePermille > 1000 {
		return errors.New("sticky active profile is incomplete")
	}
	if err := validateCommand(profile.Candidate); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if err := validateCommand(profile.Native); err != nil {
		return fmt.Errorf("native: %w", err)
	}
	if profile.Candidate.Dir == profile.Native.Dir {
		return errors.New("candidate and native directories must be distinct")
	}
	if len(profile.RequiredOutputs) == 0 {
		return errors.New("sticky active profile requires outputs")
	}
	for _, output := range profile.RequiredOutputs {
		if output == "" || filepath.IsAbs(output) || filepath.Clean(output) != output || output == "." || output == ".." || strings.HasPrefix(output, ".."+string(filepath.Separator)) {
			return fmt.Errorf("required output path is invalid: %s", output)
		}
	}
	return nil
}

func validateCommand(command Command) error {
	if command.Program == "" || !filepath.IsAbs(command.Program) || filepath.Clean(command.Program) != command.Program ||
		command.Dir == "" || !filepath.IsAbs(command.Dir) || filepath.Clean(command.Dir) != command.Dir || len(command.Env) == 0 {
		return errors.New("direct command is not explicit")
	}
	info, err := os.Stat(command.Program)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return errors.New("direct command is not an executable regular file")
	}
	for _, value := range command.Env {
		if !strings.Contains(value, "=") {
			return errors.New("direct command environment is malformed")
		}
	}
	return nil
}

func copyCommand(command Command) Command {
	command.Args = append([]string(nil), command.Args...)
	command.Env = append([]string(nil), command.Env...)
	return command
}

func copyKeys(keys map[string]ed25519.PublicKey) map[string]ed25519.PublicKey {
	copy := make(map[string]ed25519.PublicKey, len(keys))
	for key, value := range keys {
		copy[key] = append(ed25519.PublicKey(nil), value...)
	}
	return copy
}

// Run revalidates the signed decision on every invocation. Bypass, invalid
// state, drift and suspension execute only the native command. A valid active
// profile runs the candidate and, when due, an isolated native counterfactual;
// any failure or regression suspends the profile before another candidate run.
func (runner *Runner) Run(ctx context.Context, bypass bool) (Execution, error) {
	if runner == nil {
		return Execution{}, errors.New("sticky active runner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.invocation++
	now := runner.profile.Now().UTC()
	execution := Execution{
		SchemaVersion: SchemaVersion, RecordType: RecordType,
		Invocation: runner.invocation, StartedAt: now.Format(time.RFC3339Nano),
		Selected: SelectedNative,
	}
	finish := func() (Execution, error) {
		execution.CompletedAt = runner.profile.Now().UTC().Format(time.RFC3339Nano)
		return execution, nil
	}
	nativeOnly := func(reason string, status string) (Execution, error) {
		execution.Status, execution.Reason = status, reason
		execution.Native = armPointer(runner.execute(ctx, runner.profile.Native, runner.profile.RequiredOutputs))
		return finish()
	}
	if bypass {
		return nativeOnly(ReasonBypass, StatusNativeRetained)
	}
	if runner.suspended {
		return nativeOnly(ReasonPreviouslySuspended, StatusSuspended)
	}
	verified, err := stickydecision.VerifyDecision(ctx, runner.profile.DecisionRaw, runner.profile.PublicKeys, runner.profile.RevocationEpoch, now)
	if err != nil {
		return nativeOnly(classifyDecisionError(err), StatusNativeRetained)
	}
	decision := verified.Decision()
	execution.Decision, execution.ActionID = decision.ExecutionDecision, decision.ActionID
	if decision.ExecutionDecision != stickydecision.ExecutionActiveRuntime {
		return nativeOnly(ReasonUnsupportedDecision, StatusNativeRetained)
	}
	if decision.ActionID != runner.profile.ActionID || decision.Binding != runner.profile.ExpectedBinding {
		return nativeOnly(ReasonBindingMismatch, StatusNativeRetained)
	}
	if ctx.Err() != nil {
		return nativeOnly(ReasonCancelled, StatusNativeRetained)
	}
	execution.CandidateExecuted = true
	candidate := runner.execute(ctx, runner.profile.Candidate, runner.profile.RequiredOutputs)
	execution.Candidate = armPointer(candidate)
	if candidate.Outcome != OutcomeSuccess {
		runner.suspended = true
		execution.Suspended = true
		execution.Status, execution.Reason = StatusSuspended, reasonForArm(candidate, ReasonCandidateFailure)
		execution.Native = armPointer(runner.execute(ctx, runner.profile.Native, runner.profile.RequiredOutputs))
		return finish()
	}
	if runner.invocation%runner.profile.CounterfactualEvery != 0 {
		execution.Status, execution.Reason = StatusActiveExecuted, ReasonNoCounterfactual
		execution.Selected = SelectedCandidate
		return finish()
	}
	execution.Counterfactual = true
	native := runner.execute(ctx, runner.profile.Native, runner.profile.RequiredOutputs)
	execution.Native = armPointer(native)
	if native.Outcome != OutcomeSuccess {
		runner.suspended = true
		execution.Suspended = true
		execution.Status, execution.Reason = StatusSuspended, reasonForArm(native, ReasonNativeFailure)
		return finish()
	}
	execution.ExactOutputs = candidate.OutputSHA256 != "" && candidate.OutputSHA256 == native.OutputSHA256
	if !execution.ExactOutputs {
		runner.suspended = true
		execution.Suspended = true
		execution.Status, execution.Reason = StatusSuspended, ReasonOutputMismatch
		return finish()
	}
	if exceedsRegression(candidate.DurationNs, native.DurationNs, runner.profile.RegressionTolerancePermille) {
		runner.suspended = true
		execution.Suspended = true
		execution.Status, execution.Reason = StatusSuspended, ReasonRegression
		return finish()
	}
	execution.SavingNs = native.DurationNs - candidate.DurationNs
	execution.Status, execution.Reason = StatusActiveExecuted, ReasonActive
	execution.Selected = SelectedCandidate
	return finish()
}

func armPointer(result ArmResult) *ArmResult { return &result }

func classifyDecisionError(err error) string {
	switch {
	case errors.Is(err, stickydecision.ErrExpired):
		return ReasonExpiredDecision
	case errors.Is(err, stickydecision.ErrRevoked):
		return ReasonRevokedDecision
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return ReasonCancelled
	default:
		return ReasonInvalidDecision
	}
}

func reasonForArm(result ArmResult, fallback string) string {
	if result.Outcome == OutcomeCancelled {
		return ReasonCancelled
	}
	return fallback
}

func exceedsRegression(candidate, native int64, tolerance uint64) bool {
	if candidate <= native || native <= 0 {
		return false
	}
	// Compare without floating point and guard the multiplication overflow.
	if native > (int64(^uint64(0)>>1))/(int64(1000)+int64(tolerance)) {
		return candidate > native
	}
	return candidate*int64(1000) > native*(int64(1000)+int64(tolerance))
}

func runCommand(ctx context.Context, command Command, outputs []string) ArmResult {
	started := time.Now()
	process := exec.CommandContext(ctx, command.Program, command.Args...)
	process.Dir = command.Dir
	process.Env = append([]string(nil), command.Env...)
	var stdout, stderr bytes.Buffer
	process.Stdout, process.Stderr = &stdout, &stderr
	err := process.Run()
	result := ArmResult{DurationNs: time.Since(started).Nanoseconds()}
	switch {
	case err == nil:
		result.Outcome = OutcomeSuccess
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.Outcome, result.ExitCode = OutcomeCancelled, 130
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.Outcome, result.ExitCode = OutcomeFailure, exitErr.ExitCode()
		} else {
			result.Outcome, result.ExitCode = OutcomeInfrastructure, 127
		}
	}
	if result.Outcome == OutcomeSuccess {
		digest, size, hashErr := stickytrial.HashOutputs(command.Dir, outputs)
		if hashErr != nil {
			result.Outcome, result.ExitCode = OutcomeInfrastructure, 125
			result.diagnostic = hashErr.Error()
		} else {
			result.OutputSHA256, result.OutputBytes = digest, size
		}
	}
	if result.Outcome != OutcomeSuccess && result.diagnostic == "" {
		result.diagnostic = strings.TrimSpace(stderr.String())
		if result.diagnostic == "" && err != nil {
			result.diagnostic = err.Error()
		}
		if len(result.diagnostic) > 1024 {
			result.diagnostic = result.diagnostic[len(result.diagnostic)-1024:]
		}
	}
	_ = stdout
	return result
}

// MarshalExecution provides stable, reviewable evidence for checkers.
func MarshalExecution(execution Execution) ([]byte, error) {
	return json.MarshalIndent(execution, "", "  ")
}

// MarshalQualification provides stable machine-readable activation evidence.
func MarshalQualification(qualification Qualification) ([]byte, error) {
	return json.MarshalIndent(qualification, "", "  ")
}
