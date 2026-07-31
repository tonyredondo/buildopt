package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/runtimeoptimizer"
)

const inputSchema = "buildopt.runtime-evaluation/input/v1"
const evidenceSchema = "buildopt.evidence/runtime-owner-evaluation/v1"

type input struct {
	SchemaVersion    string      `json:"schemaVersion"`
	BuildOptRevision string      `json:"buildoptRevision"`
	WorkflowRunID    int64       `json:"workflowRunId"`
	WorkflowRunURL   string      `json:"workflowRunUrl"`
	Runner           runnerInput `json:"runner"`
	AA               pairedInput `json:"aa"`
	Autotuning       pairedInput `json:"autotuning"`
}

type runnerInput struct {
	Class            string `json:"class"`
	CPUCount         int    `json:"cpuCount"`
	MemoryBytes      int64  `json:"memoryBytes"`
	OOMBefore        int64  `json:"oomBefore"`
	OOMAfter         int64  `json:"oomAfter"`
	CIQueueDeltaMS   int64  `json:"ciQueueDeltaMs"`
	QueueMeasurement string `json:"queueMeasurement"`
}

type pairedInput struct {
	ControlMS       []int64  `json:"controlMs"`
	CandidateMS     []int64  `json:"candidateMs"`
	ControlSHA256   []string `json:"controlSha256"`
	CandidateSHA256 []string `json:"candidateSha256"`
}

type evidence struct {
	SchemaVersion    string             `json:"schemaVersion"`
	CapturedOn       string             `json:"capturedOn"`
	BuildOptRevision string             `json:"buildoptRevision"`
	Workflow         workflowEvidence   `json:"workflow"`
	Runner           runnerInput        `json:"runner"`
	AA               aaEvidence         `json:"aa"`
	Autotuning       autotuningEvidence `json:"autotuning"`
	Gate             gateEvidence       `json:"gate"`
}

type workflowEvidence struct {
	Name  string `json:"name"`
	RunID int64  `json:"runId"`
	URL   string `json:"url"`
}

type pairedMetrics struct {
	Pairs              int      `json:"pairs"`
	ControlMS          []int64  `json:"controlMs"`
	CandidateMS        []int64  `json:"candidateMs"`
	MeanDeltaMS        int64    `json:"meanDeltaMs"`
	Interval95MS       [2]int64 `json:"interval95Ms"`
	P95DeltaMS         int64    `json:"p95DeltaMs"`
	P99DeltaMS         int64    `json:"p99DeltaMs"`
	ReductionRatio     float64  `json:"reductionRatio"`
	ArtifactDivergence bool     `json:"artifactDivergence"`
}

type aaEvidence struct {
	Metrics                 pairedMetrics                      `json:"metrics"`
	Assignments             int                                `json:"assignments"`
	ExpectedPropensityBP    int                                `json:"expectedPropensityBasisPoints"`
	SampleRatio             runtimeoptimizer.SampleRatioReport `json:"sampleRatio"`
	DelayedOutcomesUpdated  int                                `json:"delayedOutcomesUpdated"`
	DuplicateOutcomeUpdated bool                               `json:"duplicateOutcomeUpdated"`
	RewardDelaySeconds      int64                              `json:"rewardDelaySeconds"`
	State                   string                             `json:"state"`
}

type profileEvidence struct {
	ID        string   `json:"id"`
	Digest    string   `json:"digest"`
	Arguments []string `json:"arguments"`
}

type autotuningEvidence struct {
	Metrics             pairedMetrics   `json:"metrics"`
	ControlProfile      profileEvidence `json:"controlProfile"`
	CandidateProfile    profileEvidence `json:"candidateProfile"`
	AdditionalComputeMS int64           `json:"additionalComputeMs"`
	OOMDelta            int64           `json:"oomDelta"`
	State               string          `json:"state"`
}

type gateEvidence struct {
	State  string   `json:"state"`
	Closes []string `json:"closes"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: runtime-evaluation INPUT_JSON OUTPUT_JSON")
		os.Exit(64)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath, outputPath string) error {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	var request input
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return fmt.Errorf("decode runtime evaluation: %w", err)
	}
	if request.SchemaVersion != inputSchema || len(request.BuildOptRevision) != 40 ||
		request.WorkflowRunID <= 0 || request.WorkflowRunURL == "" ||
		request.Runner.Class != runtimeoptimizer.GoldenRunnerClass ||
		request.Runner.CPUCount != 4 || request.Runner.MemoryBytes < 14<<30 ||
		request.Runner.MemoryBytes > 18<<30 || request.Runner.OOMAfter != request.Runner.OOMBefore ||
		request.Runner.CIQueueDeltaMS != 0 || request.Runner.QueueMeasurement != "COMMON_WORKFLOW_JOB_EXACT_ZERO_INCREMENTAL" {
		return errors.New("runtime evaluation: runner identity or guardrail mismatch")
	}
	aaMetrics, err := metrics(request.AA)
	if err != nil {
		return fmt.Errorf("runtime evaluation A/A: %w", err)
	}
	autotuningMetrics, err := metrics(request.Autotuning)
	if err != nil {
		return fmt.Errorf("runtime evaluation autotuning: %w", err)
	}
	aa, err := exerciseAA(request.AA, aaMetrics)
	if err != nil {
		return err
	}
	profiles, err := selectedProfiles()
	if err != nil {
		return err
	}
	additionalCompute := sum(request.Autotuning.CandidateMS) - sum(request.Autotuning.ControlMS)
	autotuning := autotuningEvidence{
		Metrics: autotuningMetrics, ControlProfile: profiles[0], CandidateProfile: profiles[1],
		AdditionalComputeMS: additionalCompute, OOMDelta: request.Runner.OOMAfter - request.Runner.OOMBefore,
		State: "PASSED",
	}
	if aa.State != "PASSED" || autotuningMetrics.MeanDeltaMS <= 0 ||
		autotuningMetrics.Interval95MS[0] <= 0 || autotuningMetrics.P95DeltaMS > 0 ||
		autotuningMetrics.P99DeltaMS > 0 || autotuningMetrics.ArtifactDivergence ||
		additionalCompute > 0 || autotuning.OOMDelta != 0 {
		return errors.New("runtime evaluation: B gate did not pass")
	}
	result := evidence{
		SchemaVersion: evidenceSchema, CapturedOn: time.Now().UTC().Format("2006-01-02"),
		BuildOptRevision: request.BuildOptRevision,
		Workflow:         workflowEvidence{Name: "Runtime Owner Evaluation", RunID: request.WorkflowRunID, URL: request.WorkflowRunURL},
		Runner:           request.Runner, AA: aa, Autotuning: autotuning,
		Gate: gateEvidence{State: "PASSED", Closes: []string{"B-G01", "B-G03"}},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(outputPath, encoded, 0o600); err != nil {
		return err
	}
	return nil
}

func exerciseAA(values pairedInput, observed pairedMetrics) (aaEvidence, error) {
	root, err := os.MkdirTemp("", "buildopt-runtime-aa-")
	if err != nil {
		return aaEvidence{}, err
	}
	defer os.RemoveAll(root)
	if err := os.Chmod(root, 0o700); err != nil {
		return aaEvidence{}, err
	}
	assignedAt := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	ledger, err := runtimeoptimizer.OpenCohortLedger(filepath.Join(root, "cohorts"), func() time.Time { return assignedAt })
	if err != nil {
		return aaEvidence{}, err
	}
	engine, err := runtimeoptimizer.OpenBanditEngine(filepath.Join(root, "bandit"), func() time.Time { return assignedAt.Add(2 * time.Hour) })
	if err != nil {
		return aaEvidence{}, err
	}
	policy := runtimeoptimizer.FixedCohortPolicy{
		PolicyVersion: "owner-aa-v1", CatalogVersion: runtimeoptimizer.GoldenResourceCatalogVersion,
		Mode: runtimeoptimizer.FixedAAMode, MinimumAssignments: 200, MaximumChiSquare: 6.635,
		Allocations: []runtimeoptimizer.CohortAllocation{
			{Cohort: "A", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 5000},
			{Cohort: "B", ResourceProfileID: "STABLE_CONTROL", PropensityBasisPoints: 5000},
		},
	}
	assignments := make([]runtimeoptimizer.CohortAssignment, 0, 200)
	updated := 0
	duplicateUpdated := false
	for index := 0; index < 200; index++ {
		id := fmt.Sprintf("owner-aa-%03d", index)
		assignment, created, err := ledger.Assign(runtimeoptimizer.CohortAssignmentRequest{
			AssignmentID: id, RepositoryID: "tonyredondo-buildopt", MeasurementEpoch: "owner-aa-2026-07-31",
			BucketDigest: digest("owner-aa-bucket"), ContextDigest: digest("owner-aa-context"),
			SeedDigest: digest("owner-aa-seed"), Policy: policy,
		})
		if err != nil || !created || assignment.PropensityBasisPoints != 5000 {
			return aaEvidence{}, fmt.Errorf("runtime evaluation: invalid A/A assignment: created=%t propensity=%d err=%v", created, assignment.PropensityBasisPoints, err)
		}
		assignments = append(assignments, assignment)
		duration := values.ControlMS[index%len(values.ControlMS)]
		if assignment.Cohort == "B" {
			duration = values.CandidateMS[index%len(values.CandidateMS)]
		}
		outcome := runtimeoptimizer.FixedCohortOutcome{
			OutcomeID: "outcome-" + id, CompletedAt: assignedAt.Add(time.Hour),
			Reward:    runtimeoptimizer.RewardComponents{Complete: true, BaselineCustomerVisibleBuildMS: duration, CustomerVisibleBuildMS: duration},
			Guardrail: "NONE",
		}
		_, changed, err := engine.RecordFixedOutcome(assignment, outcome)
		if err != nil || !changed {
			return aaEvidence{}, errors.New("runtime evaluation: delayed A/A outcome was not updated")
		}
		updated++
		if index == 199 {
			_, duplicateUpdated, err = engine.RecordFixedOutcome(assignment, outcome)
			if err != nil {
				return aaEvidence{}, err
			}
		}
	}
	ratio := runtimeoptimizer.AnalyzeSampleRatio(assignments, policy)
	tolerance := int64(math.Max(250, float64(mean(values.ControlMS))*0.05))
	state := "PASSED"
	if ratio.Status != runtimeoptimizer.SampleRatioValid || updated != 200 || duplicateUpdated ||
		abs(observed.MeanDeltaMS) > tolerance || abs(observed.P95DeltaMS) > 500 || observed.ArtifactDivergence {
		state = "FAILED"
	}
	return aaEvidence{
		Metrics: observed, Assignments: 200, ExpectedPropensityBP: 5000, SampleRatio: ratio,
		DelayedOutcomesUpdated: updated, DuplicateOutcomeUpdated: duplicateUpdated,
		RewardDelaySeconds: 3600, State: state,
	}, nil
}

func selectedProfiles() ([2]profileEvidence, error) {
	context := runtimeoptimizer.ResourceProfileContext{
		RunnerClass: runtimeoptimizer.GoldenRunnerClass, BuildClass: runtimeoptimizer.GoldenBuildClass,
		CompatibilityClass: runtimeoptimizer.GoldenCompatibilityClass, JDKVendor: runtimeoptimizer.GoldenJDKVendor,
		JDKVersion: runtimeoptimizer.GoldenJDKVersion, JDKArchitecture: runtimeoptimizer.GoldenJDKArchitecture,
		CgroupCPUCount: 4, CgroupMemoryBytes: 16 << 30, AvailableMemoryBytes: 16 << 30,
	}
	profiles := runtimeoptimizer.GoldenResourceProfiles()
	result := [2]profileEvidence{}
	for index, id := range []string{"STABLE_CONTROL", "W4_H6G"} {
		var profile runtimeoptimizer.ResourceProfile
		for _, candidate := range profiles {
			if candidate.ProfileID == id {
				profile = candidate
			}
		}
		selection, err := runtimeoptimizer.SelectGoldenResourceProfile(profile.ProfileID, profile.ProfileDigest, profile.CatalogVersion, context)
		if err != nil {
			return result, err
		}
		result[index] = profileEvidence{ID: profile.ProfileID, Digest: profile.ProfileDigest, Arguments: selection.Arguments}
	}
	return result, nil
}

func metrics(values pairedInput) (pairedMetrics, error) {
	if len(values.ControlMS) != 4 || len(values.CandidateMS) != 4 ||
		len(values.ControlSHA256) != 4 || len(values.CandidateSHA256) != 4 {
		return pairedMetrics{}, errors.New("exactly four complete pairs are required")
	}
	deltas := make([]int64, 4)
	for index := range deltas {
		if values.ControlMS[index] <= 0 || values.CandidateMS[index] <= 0 ||
			!validSHA(values.ControlSHA256[index]) || !validSHA(values.CandidateSHA256[index]) {
			return pairedMetrics{}, errors.New("invalid duration or artifact digest")
		}
		deltas[index] = values.ControlMS[index] - values.CandidateMS[index]
	}
	interval := exhaustiveBootstrapInterval(deltas)
	controlSum, candidateSum := sum(values.ControlMS), sum(values.CandidateMS)
	return pairedMetrics{
		Pairs: 4, ControlMS: append([]int64(nil), values.ControlMS...), CandidateMS: append([]int64(nil), values.CandidateMS...),
		MeanDeltaMS: mean(deltas), Interval95MS: interval,
		P95DeltaMS:         percentile(values.CandidateMS, 95) - percentile(values.ControlMS, 95),
		P99DeltaMS:         percentile(values.CandidateMS, 99) - percentile(values.ControlMS, 99),
		ReductionRatio:     float64(controlSum-candidateSum) / float64(controlSum),
		ArtifactDivergence: !sameDigests(values.ControlSHA256, values.CandidateSHA256),
	}, nil
}

func exhaustiveBootstrapInterval(deltas []int64) [2]int64 {
	means := make([]int64, 0, 256)
	for a := range deltas {
		for b := range deltas {
			for c := range deltas {
				for d := range deltas {
					means = append(means, (deltas[a]+deltas[b]+deltas[c]+deltas[d])/4)
				}
			}
		}
	}
	sort.Slice(means, func(i, j int) bool { return means[i] < means[j] })
	return [2]int64{means[6], means[249]}
}

func percentile(values []int64, percent int) int64 {
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := (percent*len(ordered) + 99) / 100
	return ordered[index-1]
}

func mean(values []int64) int64 { return sum(values) / int64(len(values)) }
func sum(values []int64) int64 {
	var result int64
	for _, value := range values {
		result += value
	}
	return result
}
func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameDigests(control, candidate []string) bool {
	if len(control) != len(candidate) {
		return false
	}
	expected := control[0]
	for _, values := range [][]string{control, candidate} {
		for _, value := range values {
			if value != expected {
				return false
			}
		}
	}
	return true
}
