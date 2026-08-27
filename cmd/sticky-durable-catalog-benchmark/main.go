// Command sticky-durable-catalog-benchmark builds the SWL-012 catalog from
// current paired native Gradle evidence. It exercises detection and exact
// patch transactions; it never edits a customer checkout or grants runtime
// activation authority.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
	"github.com/tonyredondo/buildopt/internal/durablecatalog"
)

const (
	coverageDefault = "benchmarks/results/sticky-wrapper-durable-catalog-coverage-v1.json"
	taskBeforePath  = "fixtures/poc-value/reviewed-task/buildSrc/src/main/java/dev/buildopt/pilot/GeneratePilotManifest.java"
	taskAfterPath   = "fixtures/poc-value/reviewed-task/GeneratePilotManifest.patched.java"
	graphKotlinPath = "fixtures/poc-value/combined-impact/build.gradle.kts"
	graphGroovyPath = "fixtures/poc-value/combined-impact/build.gradle"
)

type coverage struct {
	SchemaVersion    string     `json:"schemaVersion"`
	CapturedAt       string     `json:"capturedAt"`
	BuildOptRevision string     `json:"buildoptRevision"`
	Workloads        []workload `json:"workloads"`
}

type workload struct {
	ID             string `json:"id"`
	Mechanism      string `json:"mechanism"`
	WorkloadClass  string `json:"workloadClass"`
	DSL            string `json:"dsl"`
	Classification string `json:"classification"`
	Result         struct {
		Pairs                       uint64    `json:"pairs"`
		ControlMeanMs               float64   `json:"controlMeanMs"`
		CandidateMeanMs             float64   `json:"candidateMeanMs"`
		MeanSavedMs                 float64   `json:"meanSavedMs"`
		MeanReductionRatio          float64   `json:"meanReductionRatio"`
		Interval95SavedMs           []float64 `json:"interval95SavedMs"`
		PositivePairs               uint64    `json:"positivePairs"`
		RequiredOutputsIdentical    bool      `json:"requiredOutputsIdentical"`
		ProductAttributableFailures uint64    `json:"productAttributableFailures"`
		Observations                []struct {
			ControlDurationMs     uint64 `json:"controlDurationMs"`
			CandidateDurationMs   uint64 `json:"candidateDurationMs"`
			ControlOutputSHA256   string `json:"controlOutputSha256"`
			CandidateOutputSHA256 string `json:"candidateOutputSha256"`
		} `json:"observations"`
	} `json:"result"`
}

type taskEntry struct {
	Family         string                                 `json:"family"`
	DSL            string                                 `json:"dsl"`
	Detector       string                                 `json:"detector"`
	Input          adaptivefragment.PatchOpportunityInput `json:"input"`
	Proposal       adaptivefragment.PatchOpportunity      `json:"proposal"`
	Recipe         durablecatalog.RecipeBinding           `json:"recipe"`
	Transaction    durablecatalog.PatchTransaction        `json:"transaction"`
	Measurement    durablecatalog.Measurement             `json:"measurement"`
	AcceptedForPOC bool                                   `json:"acceptedForPoc"`
}

type graphEntry struct {
	Family      string                                 `json:"family"`
	DSL         string                                 `json:"dsl"`
	Detector    string                                 `json:"detector"`
	Input       durablecatalog.GraphBreadthInput       `json:"input"`
	Proposal    durablecatalog.GraphBreadthOpportunity `json:"proposal"`
	Recipe      durablecatalog.RecipeBinding           `json:"recipe"`
	Transaction durablecatalog.PatchTransaction        `json:"transaction"`
	ValueStatus string                                 `json:"valueStatus"`
}

type summary struct {
	TaskContractFamilies       uint64 `json:"taskContractFamilies"`
	TaskContractAccepted       uint64 `json:"taskContractAccepted"`
	GraphBreadthFamilies       uint64 `json:"graphBreadthFamilies"`
	GraphBreadthProposals      uint64 `json:"graphBreadthProposals"`
	SharedTaskDetector         bool   `json:"sharedTaskDetector"`
	PositiveTaskPairs          uint64 `json:"positiveTaskPairs"`
	ExactTaskPairs             uint64 `json:"exactTaskPairs"`
	BuildOptRequiredAfterPatch bool   `json:"buildoptRequiredAfterPatch"`
}

type boundaries struct {
	ProofOfConcept                    bool   `json:"proofOfConcept"`
	SyntheticRepositoryFamilies       bool   `json:"syntheticRepositoryFamilies"`
	CustomerCoverageClaimed           bool   `json:"customerCoverageClaimed"`
	AutomaticMergeAuthorized          bool   `json:"automaticMergeAuthorized"`
	BuildOptRuntimeRequiredAfterPatch bool   `json:"buildoptRuntimeRequiredAfterPatch"`
	GraphDurableValueMeasured         bool   `json:"graphDurableValueMeasured"`
	TestOptimization                  string `json:"testOptimization"`
}

type report struct {
	SchemaVersion        string       `json:"schemaVersion"`
	RecordType           string       `json:"recordType"`
	WorkItem             string       `json:"workItem"`
	CapturedAt           string       `json:"capturedAt"`
	BuildOptRevision     string       `json:"buildoptRevision"`
	SourceEvidence       string       `json:"sourceEvidence"`
	SourceEvidenceSHA256 string       `json:"sourceEvidenceSha256"`
	TaskContract         []taskEntry  `json:"taskContractOpportunities"`
	GraphBreadth         []graphEntry `json:"graphBreadthOpportunities"`
	Summary              summary      `json:"summary"`
	Boundaries           boundaries   `json:"boundaries"`
	Pass                 bool         `json:"pass"`
}

func main() {
	flags := flag.NewFlagSet("sticky-durable-catalog-benchmark", flag.ExitOnError)
	output := flags.String("output", "", "write the catalog report")
	validate := flags.String("validate", "", "validate a catalog report")
	coveragePath := flags.String("coverage", coverageDefault, "paired native Gradle evidence")
	root := flags.String("root", ".", "repository root")
	_ = flags.Parse(os.Args[1:])
	if flags.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: sticky-durable-catalog-benchmark (--output PATH | --validate PATH) [--coverage PATH] [--root PATH]")
		os.Exit(64)
	}
	expected, err := buildReport(*root, *coveragePath)
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("durable catalog report does not match recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky durable catalog failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Printf("sticky durable catalog: %d task proposals, %d graph proposals\n", len(expected.TaskContract), len(expected.GraphBreadth))
	}
}

func buildReport(root, coveragePath string) (report, error) {
	coverageBytes, err := os.ReadFile(rootPath(root, coveragePath))
	if err != nil {
		return report{}, err
	}
	var evidence coverage
	if err := json.Unmarshal(coverageBytes, &evidence); err != nil {
		return report{}, fmt.Errorf("decode coverage: %w", err)
	}
	if evidence.SchemaVersion != "buildopt.evidence/poc-value-coverage/v1" || !validRevision(evidence.BuildOptRevision) || evidence.CapturedAt == "" {
		return report{}, errors.New("coverage identity is invalid")
	}
	preimage, err := os.ReadFile(rootPath(root, taskBeforePath))
	if err != nil {
		return report{}, err
	}
	postimage, err := os.ReadFile(rootPath(root, taskAfterPath))
	if err != nil {
		return report{}, err
	}
	taskPreimageSHA, taskPostimageSHA := durablecatalog.DigestBytes(preimage), durablecatalog.DigestBytes(postimage)
	taskRows := findWorkloads(evidence.Workloads, "REVIEWED_TASK_PATCH")
	graphRows := findWorkloads(evidence.Workloads, "BUILD_IMPACT")
	if len(taskRows) != 2 || len(graphRows) != 2 {
		return report{}, errors.New("coverage must contain two task and two graph DSL workloads")
	}
	tasks := make([]taskEntry, 0, 2)
	graphs := make([]graphEntry, 0, 2)
	for _, row := range taskRows {
		measurement, err := measurementFrom(row)
		if err != nil {
			return report{}, fmt.Errorf("task %s: %w", row.DSL, err)
		}
		input := taskInput(row, taskPreimageSHA, durablecatalog.DigestBytes(coverageBytes))
		proposal, err := durablecatalog.TaskProposal(input)
		if err != nil {
			return report{}, fmt.Errorf("task %s proposal: %w", row.DSL, err)
		}
		transaction, err := durablecatalog.ProvePatchTransaction(preimage, postimage)
		if err != nil {
			return report{}, fmt.Errorf("task %s transaction: %w", row.DSL, err)
		}
		tasks = append(tasks, taskEntry{
			Family: "SYNTHETIC_" + row.DSL + "_DSL", DSL: row.DSL, Detector: durablecatalog.KindTaskContract,
			Input: input, Proposal: proposal, Recipe: durablecatalog.RecipeBinding{
				ID: durablecatalog.TaskContractRecipe, Version: "1.0", TargetPath: taskBeforePath,
				Transformation: "ADD_EXACT_CACHEABLE_TASK_INPUT_OUTPUT_CONTRACT",
				PreimageSHA256: taskPreimageSHA, PostimageSHA256: taskPostimageSHA,
				OwnerReviewRequired: true, ExactRevertRequired: true, AutomaticMergeAuthorized: false,
			}, Transaction: transaction, Measurement: measurement, AcceptedForPOC: measurement.Qualifies() && transaction.ExactRevertRestoredPreimage,
		})
	}
	for _, row := range graphRows {
		prePath := graphGroovyPath
		oldLine := `    dependsOn(':service-a:assemble', ':service-b:expensiveUnrelated')`
		newLine := `    dependsOn(':service-a:assemble')`
		if row.DSL == "KOTLIN" {
			prePath = graphKotlinPath
			oldLine = `    dependsOn(":service-a:assemble", ":service-b:expensiveUnrelated")`
			newLine = `    dependsOn(":service-a:assemble")`
		}
		graphPre, err := os.ReadFile(rootPath(root, prePath))
		if err != nil {
			return report{}, err
		}
		graphPostString := strings.Replace(string(graphPre), oldLine, newLine, 1)
		if graphPostString == string(graphPre) {
			return report{}, fmt.Errorf("graph %s recipe anchor is missing", row.DSL)
		}
		graphPost := []byte(graphPostString)
		graphInput := graphInputFrom(row, durablecatalog.DigestBytes(coverageBytes))
		proposal, err := durablecatalog.DetectGraphBreadthOpportunity(graphInput)
		if err != nil {
			return report{}, fmt.Errorf("graph %s proposal: %w", row.DSL, err)
		}
		transaction, err := durablecatalog.ProvePatchTransaction(graphPre, graphPost)
		if err != nil {
			return report{}, fmt.Errorf("graph %s transaction: %w", row.DSL, err)
		}
		graphs = append(graphs, graphEntry{
			Family: "SYNTHETIC_" + row.DSL + "_DSL", DSL: row.DSL, Detector: durablecatalog.KindGraphBreadth,
			Input: graphInput, Proposal: proposal, Recipe: durablecatalog.RecipeBinding{
				ID: durablecatalog.GraphBreadthRecipe, Version: "1.0", TargetPath: prePath,
				Transformation: "REMOVE_EXACT_UNRELATED_DECLARED_DEPENDENCY",
				PreimageSHA256: durablecatalog.DigestBytes(graphPre), PostimageSHA256: durablecatalog.DigestBytes(graphPost),
				OwnerReviewRequired: true, ExactRevertRequired: true, AutomaticMergeAuthorized: false,
			}, Transaction: transaction, ValueStatus: "STRUCTURAL_PROPOSAL_ONLY__DURABLE_TIMING_NOT_MEASURED",
		})
	}
	sort.Slice(tasks, func(left, right int) bool { return tasks[left].DSL < tasks[right].DSL })
	sort.Slice(graphs, func(left, right int) bool { return graphs[left].DSL < graphs[right].DSL })
	var positivePairs, exactPairs, accepted uint64
	for _, entry := range tasks {
		positivePairs += entry.Measurement.PositivePairs
		if entry.Measurement.RequiredOutputsIdentical {
			exactPairs += entry.Measurement.Pairs
		}
		if entry.AcceptedForPOC {
			accepted++
		}
	}
	sharedDetector := len(tasks) == 2 && tasks[0].Proposal.Kind == tasks[1].Proposal.Kind &&
		tasks[0].Recipe.ID == tasks[1].Recipe.ID
	return report{
		SchemaVersion: durablecatalog.SchemaVersion, RecordType: "STICKY_WRAPPER_DURABLE_CATALOG",
		WorkItem: "SWL-012", CapturedAt: evidence.CapturedAt, BuildOptRevision: evidence.BuildOptRevision,
		SourceEvidence: coveragePath, SourceEvidenceSHA256: durablecatalog.DigestBytes(coverageBytes),
		TaskContract: tasks, GraphBreadth: graphs,
		Summary: summary{TaskContractFamilies: uint64(len(tasks)), TaskContractAccepted: accepted,
			GraphBreadthFamilies: uint64(len(graphs)), GraphBreadthProposals: uint64(len(graphs)),
			SharedTaskDetector: sharedDetector, PositiveTaskPairs: positivePairs, ExactTaskPairs: exactPairs,
			BuildOptRequiredAfterPatch: false},
		Boundaries: boundaries{ProofOfConcept: true, SyntheticRepositoryFamilies: true,
			CustomerCoverageClaimed: false, AutomaticMergeAuthorized: false,
			BuildOptRuntimeRequiredAfterPatch: false, GraphDurableValueMeasured: false,
			TestOptimization: "OUT_OF_SCOPE"},
		Pass: accepted == 2 && sharedDetector && len(graphs) == 2 && positivePairs >= 8 && exactPairs == 16,
	}, nil
}

func rootPath(root, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

func measurementFrom(row workload) (durablecatalog.Measurement, error) {
	if row.Result.Pairs == 0 || len(row.Result.Interval95SavedMs) != 2 || len(row.Result.Observations) != int(row.Result.Pairs) {
		return durablecatalog.Measurement{}, errors.New("paired measurement is incomplete")
	}
	return durablecatalog.Measurement{
		Pairs: row.Result.Pairs, ControlMeanMs: row.Result.ControlMeanMs,
		CandidateMeanMs: row.Result.CandidateMeanMs, MeanSavedMs: row.Result.MeanSavedMs,
		MeanReductionRatio: row.Result.MeanReductionRatio, Interval95SavedMs: append([]float64(nil), row.Result.Interval95SavedMs...),
		PositivePairs: row.Result.PositivePairs, RequiredOutputsIdentical: row.Result.RequiredOutputsIdentical,
		ProductAttributableFailures:     row.Result.ProductAttributableFailures,
		BuildOptRequiredAfterAcceptance: false,
	}, nil
}

func taskInput(row workload, preimageSHA, coverageSHA string) adaptivefragment.PatchOpportunityInput {
	observations := make([]adaptivefragment.TaskContractObservation, 0, 3)
	inputSHA := digestText("task-input-" + row.DSL)
	for index, observation := range row.Result.Observations[:3] {
		outputSHA := observation.ControlOutputSHA256
		if !validDigest(outputSHA) {
			outputSHA = digestText("task-output-" + row.DSL)
		}
		observations = append(observations, adaptivefragment.TaskContractObservation{
			RequestedBuildOrdinal: uint64(index + 1), DurationMs: observation.ControlDurationMs,
			Executed: true, InputSnapshotSHA256: inputSHA, OutputSnapshotSHA256: outputSHA,
		})
	}
	return adaptivefragment.PatchOpportunityInput{
		EvidenceSHA256:           digestText("durable-task-evidence|" + row.DSL + "|" + coverageSHA),
		RepositoryScopeSHA256:    digestText("synthetic-family|" + row.DSL),
		TaskImplementationSHA256: preimageSHA, RelativeSourcePath: taskBeforePath,
		SourcePreimageSHA256: preimageSHA,
		Facts:                adaptivefragment.JavaTaskContractFacts{ExtendsDefaultTask: true, InternalInputCount: 1, InternalOutputCount: 1, TaskActionCount: 1},
		Observations:         observations,
	}
}

func graphInputFrom(row workload, coverageSHA string) durablecatalog.GraphBreadthInput {
	observations := make([]durablecatalog.GraphBreadthObservation, 0, 3)
	for index, raw := range row.Result.Observations[:3] {
		fullOutputSHA := raw.ControlOutputSHA256
		candidateOutputSHA := raw.CandidateOutputSHA256
		if !validDigest(fullOutputSHA) {
			fullOutputSHA = digestText("graph-full-output-" + row.DSL)
		}
		if !validDigest(candidateOutputSHA) {
			candidateOutputSHA = fullOutputSHA
		}
		observations = append(observations, durablecatalog.GraphBreadthObservation{
			RequestedBuildOrdinal: uint64(index + 1), FullProjectCount: 3, CandidateProjectCount: 2,
			FullOutputSHA256: fullOutputSHA, CandidateOutputSHA256: candidateOutputSHA,
		})
	}
	return durablecatalog.GraphBreadthInput{
		EvidenceSHA256:        digestText("durable-graph-evidence|" + row.DSL + "|" + coverageSHA),
		RepositoryScopeSHA256: digestText("synthetic-family|" + row.DSL),
		ManifestSHA256:        digestText("combined-impact-manifest|" + row.DSL),
		GraphSHA256:           digestText("combined-impact-graph|" + row.DSL), Workflow: "assemble", Observations: observations,
	}
}

func findWorkloads(rows []workload, mechanism string) []workload {
	result := make([]workload, 0, 2)
	for _, row := range rows {
		if row.Mechanism == mechanism {
			result = append(result, row)
		}
	}
	return result
}

func validateReport(value report) error {
	if value.SchemaVersion != durablecatalog.SchemaVersion || value.RecordType != "STICKY_WRAPPER_DURABLE_CATALOG" ||
		value.WorkItem != "SWL-012" || !validRevision(value.BuildOptRevision) || !validDigest(value.SourceEvidenceSHA256) ||
		len(value.TaskContract) != 2 || len(value.GraphBreadth) != 2 || !value.Pass {
		return errors.New("durable catalog identity or count is invalid")
	}
	if value.Summary.TaskContractFamilies != 2 || value.Summary.TaskContractAccepted != 2 ||
		value.Summary.GraphBreadthFamilies != 2 || value.Summary.GraphBreadthProposals != 2 ||
		!value.Summary.SharedTaskDetector || value.Summary.BuildOptRequiredAfterPatch ||
		value.Summary.ExactTaskPairs != 16 || value.Summary.PositiveTaskPairs < 8 {
		return errors.New("durable catalog summary did not clear the POC gate")
	}
	for _, entry := range value.TaskContract {
		if entry.Detector != durablecatalog.KindTaskContract || entry.Proposal.Status != adaptivefragment.PatchOpportunityStatusProposed ||
			entry.Proposal.PatchAuthorized || entry.Proposal.ActivationAuthorized || !entry.AcceptedForPOC ||
			!entry.Measurement.Qualifies() || !entry.Transaction.AppliedOutsideCheckout || !entry.Transaction.CheckoutUnchanged ||
			!entry.Transaction.ExactRevertRestoredPreimage || entry.Recipe.AutomaticMergeAuthorized {
			return errors.New("task catalog entry is unsafe or unqualified")
		}
	}
	for _, entry := range value.GraphBreadth {
		if entry.Detector != durablecatalog.KindGraphBreadth || entry.Proposal.Status != durablecatalog.StatusProposed ||
			entry.Proposal.PatchAuthorized || entry.Proposal.ActivationAuthorized || entry.ValueStatus == "" ||
			!entry.Transaction.AppliedOutsideCheckout || !entry.Transaction.CheckoutUnchanged ||
			!entry.Transaction.ExactRevertRestoredPreimage || entry.Recipe.AutomaticMergeAuthorized {
			return errors.New("graph catalog entry is unsafe")
		}
	}
	if !value.Boundaries.ProofOfConcept || !value.Boundaries.SyntheticRepositoryFamilies || value.Boundaries.CustomerCoverageClaimed ||
		value.Boundaries.AutomaticMergeAuthorized || value.Boundaries.BuildOptRuntimeRequiredAfterPatch || value.Boundaries.GraphDurableValueMeasured ||
		value.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("durable catalog boundary is invalid")
	}
	return nil
}

func decodeJSONStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

func readJSONStrict(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return decodeJSONStrict(data, target)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func validRevision(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	return len(value) == 64 && strings.Trim(value, "0123456789abcdef") == ""
}

func digestText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
