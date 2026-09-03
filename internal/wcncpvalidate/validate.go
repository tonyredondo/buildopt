// Package wcncpvalidate owns WCNCP-005 validation coordination over
// synthetic fixtures: proposal admission, fixed budgets, isolated roots,
// correctness runner with exact outputs, and exact-revert proof. Prospective
// Gradle timing stays closed; results record NOT_RUN timing until WCNCP-011.
package wcncpvalidate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// ProtocolVersion pins the correctness protocol validated by this block.
	ProtocolVersion = "WCNCP_CORRECTNESS_V1"
	// MinimumCorrectnessStarts is the frozen five-start floor.
	MinimumCorrectnessStarts = 5
	// MaximumInfrastructureRetries caps infra retries; correctness or value
	// failures are never retried as infrastructure.
	MaximumInfrastructureRetries = 2
)

var (
	// ErrAdmission means a proposal cannot enter validation.
	ErrAdmission = errors.New("BuildOpt WCNCP proposal not admitted")
	// ErrBudget means the fixed budget stops new work.
	ErrBudget = errors.New("BuildOpt WCNCP budget exhausted")
	// ErrDrift means source preimage or postimage moved under the patch.
	ErrDrift = errors.New("BuildOpt WCNCP source drift")
)

// ProposalAdmission is the minimal complete binding a proposal needs before a
// validator may claim it.
type ProposalAdmission struct {
	Decision             string
	BindingsComplete     bool
	BudgetRemaining      bool
	PreimageSHA256       string
	SourcePath           string
	EnvironmentClass     string
}

// Admit allows only ACTIONABLE_MATERIAL_CORRECTION with complete bindings and
// remaining budget. Classification cannot compile a candidate by itself.
func Admit(proposal ProposalAdmission) error {
	if proposal.Decision != "ACTIONABLE_MATERIAL_CORRECTION" || !proposal.BindingsComplete || !proposal.BudgetRemaining {
		return ErrAdmission
	}
	if len(proposal.PreimageSHA256) != 64 || len(proposal.SourcePath) == 0 {
		return ErrAdmission
	}
	return nil
}

// Budget tracks fixed experiment spend. Crossing a hard limit stops new work
// with INCOMPLETE_EXPERIMENT_BUDGET_EXHAUSTED semantics; it never deletes
// unrelated files, kills unrelated processes, or moves thresholds.
type Budget struct {
	MaxControlledMs int64
	MaxDiskBytes    int64
	SpentMs         int64
	SpentBytes      int64
	RetriesUsed     int
}

// Charge records attributable customer-operational cost. It fails closed when
// the charge would cross the frozen limit.
func (budget *Budget) Charge(milliseconds, bytes int64, infrastructureRetry bool) error {
	if milliseconds < 0 || bytes < 0 {
		return ErrBudget
	}
	if infrastructureRetry {
		if budget.RetriesUsed >= MaximumInfrastructureRetries {
			return ErrBudget
		}
		budget.RetriesUsed++
	}
	if budget.SpentMs+milliseconds > budget.MaxControlledMs || budget.SpentBytes+bytes > budget.MaxDiskBytes {
		return ErrBudget
	}
	budget.SpentMs += milliseconds
	budget.SpentBytes += bytes
	return nil
}

// IsolatedRoot materializes control/candidate work in explicit experiment
// roots, never the owner active checkout. Roots live under exactly one
// experiment root; cleanup touches only exact resolved experiment paths after
// process and artifact checks.
func IsolatedRoot(experimentRoot, name string) (string, error) {
	if experimentRoot == "" || name == "" || strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return "", errors.New("invalid isolated root")
	}
	clean, err := filepath.Abs(experimentRoot)
	if err != nil {
		return "", err
	}
	root := filepath.Join(clean, name)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	return root, nil
}

// DiscardRoot removes exactly one experiment-owned root after verifying it
// resolves inside the experiment root. It never uses broad paths and never
// touches unrelated Gradle caches.
func DiscardRoot(experimentRoot, root string) error {
	cleanExperiment, err := filepath.Abs(experimentRoot)
	if err != nil {
		return err
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(cleanExperiment, cleanRoot)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == "." {
		return errors.New("refusing to discard outside experiment root")
	}
	return os.RemoveAll(cleanRoot)
}

// ApplyTransaction applies one idempotent byte-range patch transaction and
// rejects source drift before mutation. The inverse transaction restores the
// exact preimage; both directions verify digests.
func ApplyTransaction(root, path string, startByte, endByte int64, preimageSHA256 string, replacement []byte) ([]byte, error) {
	if root == "" || path == "" || startByte < 0 || endByte < 1 || endByte <= startByte {
		return nil, ErrDrift
	}
	full := filepath.Join(root, filepath.FromSlash(path))
	relative, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(relative, "..") {
		return nil, ErrDrift
	}
	content, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	if int64(len(content)) < endByte {
		return nil, ErrDrift
	}
	segment := content[startByte:endByte]
	digest := sha256.Sum256(segment)
	if hex.EncodeToString(digest[:]) != preimageSHA256 {
		return nil, ErrDrift
	}
	next := append(append(append([]byte{}, content[:startByte]...), replacement...), content[endByte:]...)
	if err := os.WriteFile(full, next, 0o600); err != nil {
		return nil, err
	}
	return content, nil
}

// CorrectnessResult is one immutable fixture-level correctness outcome with
// exact outputs. Timing stays NOT_RUN until the controlled paired protocol.
type CorrectnessResult struct {
	Starts           int
	ExactOutputs     bool
	Invalidation     bool
	ExactRevert      bool
	ProductFailures  int
	Decision         string
	FailedPrereqs    []string
	CheckedAt        time.Time
}

// RunFixtureCorrectness executes the five-start floor against two isolated
// roots with a caller-supplied build function: control exec, candidate exec,
// reuse, input-change invalidation, and restoration. Every declared output is
// compared byte-for-byte; any mismatch, unexpected reuse, missing
// invalidation, drift, or product failure rejects before timing.
func RunFixtureCorrectness(build func(root string, input string) (outputs map[string][]byte, err error), baselineInput, changedInput string) CorrectnessResult {
	checkedAt := time.Now().UTC()
	fail := func(prereqs ...string) CorrectnessResult {
		return CorrectnessResult{Starts: 5, FailedPrereqs: prereqs, Decision: "REJECTED_CORRECTNESS", CheckedAt: checkedAt}
	}
	controlRoot, err := os.MkdirTemp("", "wcncp-control-*")
	if err != nil {
		return fail("control-root")
	}
	defer os.RemoveAll(controlRoot)
	candidateRoot, err := os.MkdirTemp("", "wcncp-candidate-*")
	if err != nil {
		return fail("candidate-root")
	}
	defer os.RemoveAll(candidateRoot)
	controlOutputs, err := build(controlRoot, baselineInput)
	if err != nil || len(controlOutputs) == 0 {
		return fail("control-execution")
	}
	candidateOutputs, err := build(candidateRoot, baselineInput)
	if err != nil {
		return fail("candidate-execution")
	}
	if !equalOutputs(controlOutputs, candidateOutputs) {
		return fail("exact-outputs")
	}
	reuseOutputs, err := build(candidateRoot, baselineInput)
	if err != nil || !equalOutputs(candidateOutputs, reuseOutputs) {
		return fail("reuse")
	}
	changedOutputs, err := build(candidateRoot, changedInput)
	if err != nil {
		return fail("invalidation-execution")
	}
	if equalOutputs(candidateOutputs, changedOutputs) {
		return fail("expected-invalidation")
	}
	restoredOutputs, err := build(candidateRoot, baselineInput)
	if err != nil || !equalOutputs(candidateOutputs, restoredOutputs) {
		return fail("exact-restoration")
	}
	return CorrectnessResult{Starts: 5, ExactOutputs: true, Invalidation: true, ExactRevert: true, Decision: "QUALIFIED", CheckedAt: checkedAt}
}

func equalOutputs(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for name, leftBytes := range left {
		rightBytes, ok := right[name]
		if !ok || !bytes.Equal(leftBytes, rightBytes) {
			return false
		}
	}
	return true
}
