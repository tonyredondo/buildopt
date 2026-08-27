package launcher

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickyactive"
	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickytrial"
	"github.com/tonyredondo/buildopt/internal/stickyvalue"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
)

const stickyLearningEnvironment = "BUILDOPT_STICKY_LEARNING"

// These four private seams are the deterministic boundary of the composed
// learning fixture. Production defaults are conservative: no detector result,
// no trial and no decision publication can occur without explicit authority.
var (
	stickyLearningClock    = time.Now
	stickyLearningDetector = func(context.Context, stickyLearningContext) ([]stickyLearningProposal, error) {
		return nil, nil
	}
	stickyLearningTrialRunner = func(context.Context, stickyLearningProposal) ([]stickytrial.PairedTrial, stickyvalue.Costs, error) {
		return nil, stickyvalue.Costs{}, errors.New("sticky learning trial runner is not configured")
	}
	stickyLearningDecisionPublisher = func(context.Context, stickyLearningProposal, stickyvalue.Evaluation) error {
		return errors.New("sticky learning decision publisher is not configured")
	}
)

type stickyLearningContext struct {
	Root       string
	Config     stickywrapper.Config
	Connection *stickyWrapperConnection
	Now        time.Time
}

func runTrustedStickyLearning(ctx context.Context, input stickyLearningContext) error {
	if ctx == nil {
		return errors.New("sticky learning context is nil")
	}
	if input.Now.IsZero() {
		input.Now = stickyLearningClock().UTC()
	}
	proposals, err := stickyLearningDetector(ctx, input)
	if err != nil {
		return err
	}
	for _, proposal := range proposals {
		if proposal.ActionID == "" || proposal.Binding.RepositoryScopeSHA256 == "" {
			return errors.New("sticky learning detector returned an invalid proposal")
		}
		trials, costs, err := stickyLearningTrialRunner(ctx, proposal)
		if err != nil {
			return err
		}
		pairs := make([]stickyvalue.Pair, 0, len(trials))
		for _, trial := range trials {
			if err := trial.Validate(); err != nil {
				return err
			}
			pairs = append(pairs, stickyvalue.Pair{
				PairID: trial.TrialID, Order: string(trial.Order),
				NativeWallNs: trial.Native.DurationNs, CandidateWallNs: trial.Candidate.DurationNs,
				OutputsEquivalent: trial.Equivalence == stickytrial.EquivalenceExact,
				NativeFailure:     trial.Native.Outcome != stickytrial.OutcomeSuccess,
				CandidateFailure:  trial.Candidate.Outcome != stickytrial.OutcomeSuccess,
			})
		}
		evaluation, err := stickyvalue.Evaluate(pairs, costs)
		if err != nil {
			return err
		}
		if !evaluation.Qualified {
			continue
		}
		if err := stickyLearningDecisionPublisher(ctx, proposal, evaluation); err != nil {
			return err
		}
	}
	return nil
}

type stickyLearningProposal struct {
	ActionID string
	Binding  stickydecision.Binding
}

// stickyLearningEntry is the sole initial routing decision for a committed
// wrapper invocation. Learning requests stay on the composed path; every
// other invocation may retain the previously proven native fast path.
type stickyLearningEntry struct {
	nativePath stickyNativeNoopPath
	nativeOnly bool
}

func prepareStickyLearningEntry(root string, childArgs []string, getenv func(string) string) stickyLearningEntry {
	if getenv != nil && strings.TrimSpace(getenv(stickyLearningEnvironment)) == "1" {
		return stickyLearningEntry{}
	}
	path, ok := prepareStickyNativeNoopPath(root, childArgs, getenv)
	return stickyLearningEntry{nativePath: path, nativeOnly: ok}
}

// stickyLearningEligibility proves every committed and credential-bound gate
// before a trusted trial may be scheduled. It performs no writes.
func stickyLearningEligibility(root string, connection *stickyWrapperConnection, getenv func(string) string) (stickywrapper.Config, string) {
	if getenv == nil || getenv(bypassEnvironment) == "1" {
		return stickywrapper.Config{}, "BYPASS"
	}
	config, err := stickywrapper.LoadConfig(root)
	if err != nil {
		return stickywrapper.Config{}, "CONFIG_UNAVAILABLE"
	}
	if config.Mode != "auto" {
		return config, "MODE_NOT_AUTO"
	}
	if config.TrialBudgetPercent <= 0 {
		return config, "ZERO_TRIAL_BUDGET"
	}
	if strings.TrimSpace(getenv(stickyLearningEnvironment)) != "1" {
		return config, "MISSING_LEARNING_ENVIRONMENT"
	}
	if connection == nil || !connection.hasCapability(stickyStateWriteCapability) {
		return config, "MISSING_STATE_WRITE"
	}
	if !connection.hasCapability(stickyCacheReadCapability) || !connection.hasCapability(stickyStateReadCapability) {
		return config, "MISSING_READ_CAPABILITY"
	}
	return config, "ELIGIBLE"
}

// stickyLearningActiveExecutor adapts the launcher supervisor to stickyactive
// without allowing that package to own customer process groups or signals.
func stickyLearningActiveExecutor(reserved []string, stdin io.Reader, stdout, stderr io.Writer) stickyactive.Executor {
	return func(ctx context.Context, command stickyactive.Command, outputs []string) stickyactive.ArmResult {
		_ = ctx
		execution := executeChildWithReservedDirectory(commandArgs(command.Program, command.Args), environmentMap(command.Env), reserved, command.Dir, stdin, stdout, stderr)
		return activeArmResult(execution, command.Dir, outputs)
	}
}

func stickyLearningTrialExecutor(reserved []string, stdin io.Reader, stdout, stderr io.Writer) stickytrial.Executor {
	return func(ctx context.Context, command stickytrial.Command, isolation stickytrial.Isolation, outputs []string) stickytrial.ArmResult {
		_ = ctx
		_ = isolation
		execution := executeChildWithReservedDirectory(commandArgs(command.Program, command.Args), environmentMap(command.Env), reserved, command.Dir, stdin, stdout, stderr)
		active := activeArmResult(execution, command.Dir, outputs)
		return stickytrial.ArmResult{
			Outcome: active.Outcome, ExitCode: active.ExitCode,
			DurationNs: active.DurationNs, OutputSHA256: active.OutputSHA256,
			OutputBytes: active.OutputBytes,
		}
	}
}

func commandArgs(program string, args []string) []string {
	return append([]string{program}, args...)
}

func environmentMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		name, content, ok := strings.Cut(value, "=")
		if ok {
			result[name] = content
		}
	}
	return result
}

func activeArmResult(execution childExecution, root string, outputs []string) stickyactive.ArmResult {
	duration := execution.completedAt.Sub(execution.startedAt)
	if !execution.started {
		duration = 0
	}
	result := stickyactive.ArmResult{DurationNs: duration.Nanoseconds()}
	switch {
	case !execution.started:
		result.Outcome, result.ExitCode = stickyactive.OutcomeInfrastructure, exitNotFound
	case execution.err == nil:
		result.Outcome = stickyactive.OutcomeSuccess
	default:
		result.Outcome, result.ExitCode = stickyactive.OutcomeFailure, childExitCode(execution.err)
	}
	if result.Outcome == stickyactive.OutcomeSuccess {
		digest, size, err := stickytrial.HashOutputs(root, outputs)
		if err != nil {
			result.Outcome, result.ExitCode = stickyactive.OutcomeInfrastructure, 125
		} else {
			result.OutputSHA256, result.OutputBytes = digest, size
		}
	}
	return result
}

func childExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		return exitErr.ExitCode()
	}
	return exitCannotExecute
}

// ensure the learning switch is always scrubbed before Gradle, even when a
// caller entered the full connected path.
func stickyLearningReservedEnvironment() []string {
	return []string{stickyLearningEnvironment}
}
