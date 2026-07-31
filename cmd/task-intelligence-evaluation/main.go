package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const inputSchema = "buildopt.task-intelligence-evaluation/input/v1"
const evidenceSchema = "buildopt.evidence/task-intelligence-pilot/v1"

type input struct {
	SchemaVersion     string   `json:"schemaVersion"`
	BuildOptRevision  string   `json:"buildoptRevision"`
	Workflow          workflow `json:"workflow"`
	Runner            runner   `json:"runner"`
	Pilot             pilot    `json:"pilot"`
	ControlMS         []int64  `json:"controlMs"`
	CandidateMS       []int64  `json:"candidateMs"`
	ControlSHA256     []string `json:"controlSha256"`
	CandidateSHA256   []string `json:"candidateSha256"`
	ControlOutcomes   []string `json:"controlOutcomes"`
	CandidateOutcomes []string `json:"candidateOutcomes"`
}

type workflow struct {
	Name  string `json:"name"`
	RunID int64  `json:"runId"`
	URL   string `json:"url"`
}
type runner struct {
	Class       string `json:"class"`
	CPUCount    int    `json:"cpuCount"`
	MemoryBytes int64  `json:"memoryBytes"`
	OOMBefore   int64  `json:"oomBefore"`
	OOMAfter    int64  `json:"oomAfter"`
}
type pilot struct {
	Repository       string `json:"repository"`
	BaseRevision     string `json:"baseRevision"`
	AcceptedRevision string `json:"acceptedRevision"`
	PullRequest      int    `json:"pullRequest"`
	PullRequestURL   string `json:"pullRequestUrl"`
}
type metrics struct {
	Pairs              int      `json:"pairs"`
	ControlMS          []int64  `json:"controlMs"`
	CandidateMS        []int64  `json:"candidateMs"`
	MeanSavedMS        int64    `json:"meanSavedMs"`
	Interval95MS       [2]int64 `json:"interval95Ms"`
	P95DeltaMS         int64    `json:"p95DeltaMs"`
	P99DeltaMS         int64    `json:"p99DeltaMs"`
	ReductionRatio     float64  `json:"reductionRatio"`
	ArtifactDivergence bool     `json:"artifactDivergence"`
}

type evidence struct {
	SchemaVersion    string                `json:"schemaVersion"`
	CapturedOn       string                `json:"capturedOn"`
	BuildOptRevision string                `json:"buildoptRevision"`
	Workflow         workflow              `json:"workflow"`
	Runner           runner                `json:"runner"`
	Pilot            pilot                 `json:"pilot"`
	Task             taskEvidence          `json:"task"`
	Qualification    qualificationEvidence `json:"qualification"`
	Measurement      metrics               `json:"measurement"`
	Outcomes         outcomeEvidence       `json:"outcomes"`
	Gate             gateEvidence          `json:"gate"`
}

type taskEvidence struct {
	Path            string `json:"path"`
	Type            string `json:"type"`
	Recipe          string `json:"recipe"`
	PreimageDigest  string `json:"preimageDigest"`
	PostimageDigest string `json:"postimageDigest"`
}
type qualificationEvidence struct {
	Route                   string   `json:"route"`
	States                  []string `json:"states"`
	AgentRoute              string   `json:"agentRoute"`
	HelperRoute             string   `json:"helperRoute"`
	IncompleteCoverage      string   `json:"incompleteCoverage"`
	UnattributedPublication string   `json:"unattributedPublication"`
	TestTasks               string   `json:"testTasks"`
}
type outcomeEvidence struct {
	Control   []string `json:"control"`
	Candidate []string `json:"candidate"`
}
type gateEvidence struct {
	State                         string   `json:"state"`
	Closes                        []string `json:"closes"`
	ProductionPromotionAuthorized bool     `json:"productionPromotionAuthorized"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: task-intelligence-evaluation INPUT_JSON OUTPUT_JSON")
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
		return err
	}
	if request.SchemaVersion != inputSchema || len(request.BuildOptRevision) != 40 || request.Workflow.Name != "Task Intelligence Pilot Evaluation" || request.Workflow.RunID <= 0 || request.Workflow.URL == "" || request.Runner.Class != "linux-amd64-4c-16g-v1" || request.Runner.CPUCount != 4 || request.Runner.MemoryBytes < 14<<30 || request.Runner.MemoryBytes > 18<<30 || request.Runner.OOMBefore != request.Runner.OOMAfter || request.Pilot.Repository != "tonyredondo/buildopt-pilot" || len(request.Pilot.BaseRevision) != 40 || len(request.Pilot.AcceptedRevision) != 40 || request.Pilot.PullRequest != 1 || request.Pilot.PullRequestURL == "" {
		return errors.New("task intelligence evaluation: invalid immutable identity")
	}
	measured, err := calculate(request)
	if err != nil {
		return err
	}
	if measured.MeanSavedMS <= 0 || measured.Interval95MS[0] <= 0 || measured.P95DeltaMS > 0 || measured.P99DeltaMS > 0 || measured.ArtifactDivergence {
		return errors.New("task intelligence evaluation: causal gate did not pass")
	}
	result := evidence{
		SchemaVersion: evidenceSchema, CapturedOn: time.Now().UTC().Format("2006-01-02"), BuildOptRevision: request.BuildOptRevision, Workflow: request.Workflow, Runner: request.Runner, Pilot: request.Pilot,
		Task:          taskEvidence{Path: "buildSrc/src/main/java/dev/buildopt/pilot/GeneratePilotManifest.java", Type: "GeneratePilotManifest", Recipe: "CUSTOM_TASK_CONTRACT_JAVA_V1", PreimageDigest: "sha256:392e0197d9143304d8ce3c1a598aa552d62e5f2cc200a5766c5d06e516ca5083", PostimageDigest: "sha256:79fa100a4e2d962dd506259311f90ecb92ad62500391d140065b6f157a172a2b"},
		Qualification: qualificationEvidence{Route: "REVIEWED_SOURCE_PATCH", States: []string{"UNKNOWN", "OBSERVING", "CONTRACT_QUALIFIED", "QUARANTINE_VALIDATED", "ACTIVE"}, AgentRoute: "UNAVAILABLE_FAIL_CLOSED", HelperRoute: "UNAVAILABLE_FAIL_CLOSED", IncompleteCoverage: "INCONCLUSIVE_ABORT_PENDING", UnattributedPublication: "ABORT_WHOLE_ATTEMPT", TestTasks: "EXCLUDED"},
		Measurement:   measured, Outcomes: outcomeEvidence{Control: request.ControlOutcomes, Candidate: request.CandidateOutcomes},
		Gate: gateEvidence{State: "PASSED", Closes: []string{"C1-001", "C1-002", "C1-003", "C1-004", "C1-005", "C1-006", "C1-007", "C1-008", "C1-009", "C1-G01", "C1-G02", "C1-G03", "C1-G04", "C1-G05", "C1-G06", "C1-G07", "C1-G08", "C1-G09", "C4-004", "C4-G06"}, ProductionPromotionAuthorized: false},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, append(encoded, '\n'), 0o600)
}

func calculate(request input) (metrics, error) {
	if len(request.ControlMS) != 4 || len(request.CandidateMS) != 4 || len(request.ControlSHA256) != 4 || len(request.CandidateSHA256) != 4 || len(request.ControlOutcomes) != 4 || len(request.CandidateOutcomes) != 4 {
		return metrics{}, errors.New("exactly four complete pairs are required")
	}
	deltas := make([]int64, 4)
	expected := request.ControlSHA256[0]
	divergence := false
	for i := range deltas {
		if request.ControlMS[i] <= 0 || request.CandidateMS[i] <= 0 || !validSHA(request.ControlSHA256[i]) || !validSHA(request.CandidateSHA256[i]) || request.ControlOutcomes[i] != "EXECUTED" || request.CandidateOutcomes[i] != "FROM_CACHE" {
			return metrics{}, errors.New("invalid duration, digest, or task outcome")
		}
		deltas[i] = request.ControlMS[i] - request.CandidateMS[i]
		divergence = divergence || request.ControlSHA256[i] != expected || request.CandidateSHA256[i] != expected
	}
	controlSum, candidateSum := sum(request.ControlMS), sum(request.CandidateMS)
	return metrics{Pairs: 4, ControlMS: append([]int64(nil), request.ControlMS...), CandidateMS: append([]int64(nil), request.CandidateMS...), MeanSavedMS: mean(deltas), Interval95MS: interval(deltas), P95DeltaMS: percentile(request.CandidateMS, 95) - percentile(request.ControlMS, 95), P99DeltaMS: percentile(request.CandidateMS, 99) - percentile(request.ControlMS, 99), ReductionRatio: float64(controlSum-candidateSum) / float64(controlSum), ArtifactDivergence: divergence}, nil
}

func interval(values []int64) [2]int64 {
	means := make([]int64, 0, 256)
	for a := range values {
		for b := range values {
			for c := range values {
				for d := range values {
					means = append(means, (values[a]+values[b]+values[c]+values[d])/4)
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
	return ordered[(percent*len(ordered)+99)/100-1]
}
func sum(values []int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
}
func mean(values []int64) int64 { return sum(values) / int64(len(values)) }
func validSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
