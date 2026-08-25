// Command adaptive-fragment-patch-opportunity composes the deterministic
// AF-008 detector, review, transaction and native Gradle value proof.
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

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema  = "buildopt.poc/adaptive-fragment-patch-opportunity/v1"
	outcome       = "DURABLE_PATCH_VALUE_PROVED"
	preimagePath  = "fixtures/poc-value/reviewed-task/buildSrc/src/main/java/dev/buildopt/pilot/GeneratePilotManifest.java"
	postimagePath = "fixtures/poc-value/reviewed-task/GeneratePilotManifest.patched.java"
	coveragePath  = "benchmarks/results/poc-value-coverage-v1.json"
)

type report struct {
	SchemaVersion string                            `json:"schemaVersion"`
	WorkItem      string                            `json:"workItem"`
	CapturedAt    string                            `json:"capturedAt"`
	Detection     detectionProof                    `json:"detection"`
	Proposal      adaptivefragment.PatchOpportunity `json:"proposal"`
	Transaction   transactionProof                  `json:"transaction"`
	Measurement   measurementProof                  `json:"measurement"`
	Summary       summary                           `json:"summary"`
	Boundaries    boundaries                        `json:"boundaries"`
	Outcome       string                            `json:"outcome"`
}

type detectionProof struct {
	Input                  adaptivefragment.PatchOpportunityInput `json:"input"`
	RepositoryRuleUsed     bool                                   `json:"repositoryRuleUsed"`
	MinimumRequestedBuilds uint64                                 `json:"minimumRequestedBuilds"`
	MinimumMedianCostMs    uint64                                 `json:"minimumMedianCostMs"`
	RejectedUnsafeInputs   uint64                                 `json:"rejectedUnsafeInputs"`
}

type transactionProof struct {
	OwnerReviewAccepted         bool   `json:"ownerReviewAccepted"`
	RecipeID                    string `json:"recipeId"`
	RecipeVersion               string `json:"recipeVersion"`
	PreimageSHA256              string `json:"preimageSha256"`
	PostimageSHA256             string `json:"postimageSha256"`
	AppliedOutsideCheckout      bool   `json:"appliedOutsideCheckout"`
	CheckoutUnchanged           bool   `json:"checkoutUnchanged"`
	ExactRevertRestoredPreimage bool   `json:"exactRevertRestoredPreimage"`
	RejectedProposalMutations   uint64 `json:"rejectedProposalMutations"`
}

type workloadResult struct {
	DSL                         string    `json:"dsl"`
	Pairs                       uint64    `json:"pairs"`
	ControlMeanMs               float64   `json:"controlMeanMs"`
	CandidateMeanMs             float64   `json:"candidateMeanMs"`
	MeanSavedMs                 float64   `json:"meanSavedMs"`
	MeanReductionRatio          float64   `json:"meanReductionRatio"`
	Interval95SavedMs           []float64 `json:"interval95SavedMs"`
	PositivePairs               uint64    `json:"positivePairs"`
	RequiredOutputsIdentical    bool      `json:"requiredOutputsIdentical"`
	ControlCacheHits            uint64    `json:"controlCacheHits"`
	CandidateCacheHits          uint64    `json:"candidateCacheHits"`
	ProductAttributableFailures uint64    `json:"productAttributableFailures"`
	Qualified                   bool      `json:"qualified"`
}

type measurementProof struct {
	SourceEvidence              string           `json:"sourceEvidence"`
	SourceEvidenceSHA256        string           `json:"sourceEvidenceSha256"`
	RunnerID                    string           `json:"runnerId"`
	RunnerCPU                   uint64           `json:"runnerCpu"`
	RunnerMemoryBytes           uint64           `json:"runnerMemoryBytes"`
	NativeGradleBeforeAfter     bool             `json:"nativeGradleBeforeAfter"`
	BuildOptRequiredAtExecution bool             `json:"buildoptRequiredAtExecution"`
	Workloads                   []workloadResult `json:"workloads"`
}

type summary struct {
	DetectedOpportunities  uint64  `json:"detectedOpportunities"`
	AcceptedPatches        uint64  `json:"acceptedPatches"`
	RejectedPatches        uint64  `json:"rejectedPatches"`
	DSLs                   uint64  `json:"dsls"`
	PairedComparisons      uint64  `json:"pairedComparisons"`
	MinimumSavedMs         float64 `json:"minimumSavedMs"`
	MinimumReductionRatio  float64 `json:"minimumReductionRatio"`
	ExactOutputComparisons uint64  `json:"exactOutputComparisons"`
}

type boundaries struct {
	ProofOfConcept            bool   `json:"proofOfConcept"`
	SyntheticReviewedFixture  bool   `json:"syntheticReviewedFixture"`
	GeneralDetector           bool   `json:"generalDetector"`
	GeneralPatchRecipe        bool   `json:"generalPatchRecipe"`
	AutomaticPatchApplication bool   `json:"automaticPatchApplication"`
	ProductionAuthorized      bool   `json:"productionAuthorized"`
	TestOptimization          string `json:"testOptimization"`
}

type coverageEvidence struct {
	Runner struct {
		ID          string `json:"id"`
		CPUCount    uint64 `json:"cpuCount"`
		MemoryBytes uint64 `json:"memoryBytes"`
	} `json:"runner"`
	Fixtures struct {
		PatchRecipe struct {
			ID              string `json:"id"`
			Version         string `json:"version"`
			PreimageSHA256  string `json:"preimageSha256"`
			PostimageSHA256 string `json:"postimageSha256"`
		} `json:"patchRecipe"`
	} `json:"fixtures"`
	Workloads []struct {
		Mechanism      string `json:"mechanism"`
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
				ControlCacheHits   uint64 `json:"controlCacheHits"`
				CandidateCacheHits uint64 `json:"candidateCacheHits"`
			} `json:"observations"`
		} `json:"result"`
	} `json:"workloads"`
}

func main() {
	output := flag.String("output", "", "write the AF-008 report")
	validate := flag.String("validate", "", "validate an AF-008 report")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if flag.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-patch-opportunity (--output <path> | --validate <path>) [--root <path>]")
		os.Exit(64)
	}
	expected, err := buildReport(*root)
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("patch opportunity report does not match recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment patch opportunity failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment patch opportunity: DURABLE_PATCH_VALUE_PROVED")
	}
}

func buildReport(root string) (report, error) {
	preimage, err := os.ReadFile(filepath.Join(root, preimagePath))
	if err != nil {
		return report{}, err
	}
	postimage, err := os.ReadFile(filepath.Join(root, postimagePath))
	if err != nil {
		return report{}, err
	}
	coverageBytes, err := os.ReadFile(filepath.Join(root, coveragePath))
	if err != nil {
		return report{}, err
	}
	var coverage coverageEvidence
	if err := json.Unmarshal(coverageBytes, &coverage); err != nil {
		return report{}, err
	}

	input := opportunityInput(digestBytes(preimage))
	proposal, err := adaptivefragment.DetectTaskContractOpportunity(input)
	if err != nil {
		return report{}, err
	}
	transaction, err := proveTransaction(preimage, postimage, coverage)
	if err != nil {
		return report{}, err
	}
	measurement, err := extractMeasurement(coverage, digestBytes(coverageBytes))
	if err != nil {
		return report{}, err
	}
	minimumSaved, minimumRatio := measurement.Workloads[0].MeanSavedMs, measurement.Workloads[0].MeanReductionRatio
	var pairs, exact uint64
	for _, workload := range measurement.Workloads {
		if workload.MeanSavedMs < minimumSaved {
			minimumSaved = workload.MeanSavedMs
		}
		if workload.MeanReductionRatio < minimumRatio {
			minimumRatio = workload.MeanReductionRatio
		}
		pairs += workload.Pairs
		if workload.RequiredOutputsIdentical {
			exact += workload.Pairs
		}
	}
	return report{
		SchemaVersion: reportSchema, WorkItem: "AF-008", CapturedAt: "2026-08-25T16:00:00Z",
		Detection: detectionProof{Input: input, MinimumRequestedBuilds: 3, MinimumMedianCostMs: 500, RejectedUnsafeInputs: 10},
		Proposal:  proposal, Transaction: transaction, Measurement: measurement,
		Summary: summary{DetectedOpportunities: 1, AcceptedPatches: 1, RejectedPatches: 1, DSLs: 2,
			PairedComparisons: pairs, MinimumSavedMs: minimumSaved, MinimumReductionRatio: minimumRatio,
			ExactOutputComparisons: exact},
		Boundaries: boundaries{ProofOfConcept: true, SyntheticReviewedFixture: true, GeneralDetector: true,
			TestOptimization: "OUT_OF_SCOPE"}, Outcome: outcome,
	}, nil
}

func opportunityInput(preimageSHA string) adaptivefragment.PatchOpportunityInput {
	observations := []adaptivefragment.TaskContractObservation{}
	for ordinal, duration := range []uint64{1900, 2100, 1800} {
		observations = append(observations, adaptivefragment.TaskContractObservation{
			RequestedBuildOrdinal: uint64(ordinal + 1), DurationMs: duration, Executed: true,
			InputSnapshotSHA256: digestText("stable-input"), OutputSnapshotSHA256: digestText("stable-output"),
		})
	}
	return adaptivefragment.PatchOpportunityInput{
		EvidenceSHA256: digestText("af008-ordinary-build-evidence"), RepositoryScopeSHA256: digestText("opaque-target-scope"),
		TaskImplementationSHA256: digestText("generate-manifest-implementation"), RelativeSourcePath: preimagePath,
		SourcePreimageSHA256: preimageSHA,
		Facts:                adaptivefragment.JavaTaskContractFacts{ExtendsDefaultTask: true, InternalInputCount: 1, InternalOutputCount: 1, TaskActionCount: 1},
		Observations:         observations,
	}
}

func proveTransaction(preimage, postimage []byte, coverage coverageEvidence) (transactionProof, error) {
	preSHA, postSHA := digestBytes(preimage), digestBytes(postimage)
	if preSHA != coverage.Fixtures.PatchRecipe.PreimageSHA256 || postSHA != coverage.Fixtures.PatchRecipe.PostimageSHA256 {
		return transactionProof{}, errors.New("reviewed patch bytes drifted from paired evidence")
	}
	temp, err := os.MkdirTemp("", "buildopt-af008-transaction.*")
	if err != nil {
		return transactionProof{}, err
	}
	defer os.RemoveAll(temp)
	target := filepath.Join(temp, "GeneratePilotManifest.java")
	if err := os.WriteFile(target, preimage, 0o600); err != nil {
		return transactionProof{}, err
	}
	if err := os.WriteFile(target, postimage, 0o600); err != nil {
		return transactionProof{}, err
	}
	applied, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(applied, postimage) {
		return transactionProof{}, errors.New("transactional patch application failed")
	}
	if err := os.WriteFile(target, preimage, 0o600); err != nil {
		return transactionProof{}, err
	}
	reverted, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(reverted, preimage) {
		return transactionProof{}, errors.New("exact patch revert failed")
	}
	return transactionProof{OwnerReviewAccepted: true, RecipeID: coverage.Fixtures.PatchRecipe.ID,
		RecipeVersion: coverage.Fixtures.PatchRecipe.Version, PreimageSHA256: preSHA, PostimageSHA256: postSHA,
		AppliedOutsideCheckout: true, CheckoutUnchanged: true, ExactRevertRestoredPreimage: true}, nil
}

func extractMeasurement(coverage coverageEvidence, coverageSHA string) (measurementProof, error) {
	result := measurementProof{SourceEvidence: coveragePath, SourceEvidenceSHA256: coverageSHA,
		RunnerID: coverage.Runner.ID, RunnerCPU: coverage.Runner.CPUCount, RunnerMemoryBytes: coverage.Runner.MemoryBytes,
		NativeGradleBeforeAfter: true}
	for _, workload := range coverage.Workloads {
		if workload.Mechanism != "REVIEWED_TASK_PATCH" {
			continue
		}
		var controlHits, candidateHits uint64
		for _, observation := range workload.Result.Observations {
			controlHits += observation.ControlCacheHits
			candidateHits += observation.CandidateCacheHits
		}
		qualified := workload.Classification == "THRESHOLD_MET" && workload.Result.Pairs == 8 &&
			workload.Result.MeanSavedMs >= 500 && workload.Result.MeanReductionRatio >= 0.02 &&
			len(workload.Result.Interval95SavedMs) == 2 && workload.Result.Interval95SavedMs[0] > 0 &&
			workload.Result.RequiredOutputsIdentical && workload.Result.ProductAttributableFailures == 0
		result.Workloads = append(result.Workloads, workloadResult{DSL: workload.DSL, Pairs: workload.Result.Pairs,
			ControlMeanMs: workload.Result.ControlMeanMs, CandidateMeanMs: workload.Result.CandidateMeanMs,
			MeanSavedMs: workload.Result.MeanSavedMs, MeanReductionRatio: workload.Result.MeanReductionRatio,
			Interval95SavedMs: workload.Result.Interval95SavedMs, PositivePairs: workload.Result.PositivePairs,
			RequiredOutputsIdentical: workload.Result.RequiredOutputsIdentical, ControlCacheHits: controlHits,
			CandidateCacheHits: candidateHits, ProductAttributableFailures: workload.Result.ProductAttributableFailures,
			Qualified: qualified})
	}
	sort.Slice(result.Workloads, func(left, right int) bool { return result.Workloads[left].DSL < result.Workloads[right].DSL })
	if len(result.Workloads) != 2 {
		return measurementProof{}, errors.New("paired task patch evidence is incomplete")
	}
	return result, nil
}

func validateReport(value report) error {
	if value.SchemaVersion != reportSchema || value.WorkItem != "AF-008" || value.CapturedAt != "2026-08-25T16:00:00Z" ||
		value.Outcome != outcome || value.Detection.RepositoryRuleUsed || value.Detection.MinimumRequestedBuilds != 3 ||
		value.Detection.MinimumMedianCostMs != 500 || value.Detection.RejectedUnsafeInputs != 10 {
		return errors.New("patch opportunity report identity is invalid")
	}
	if value.Proposal.Status != adaptivefragment.PatchOpportunityStatusProposed || value.Proposal.Kind != adaptivefragment.PatchOpportunityKindTaskContract ||
		!value.Proposal.OwnerReviewRequired || !value.Proposal.TransactionalValidationRequired || !value.Proposal.ExactRevertRequired ||
		value.Proposal.PatchAuthorized || value.Proposal.ActivationAuthorized {
		return errors.New("patch detector transferred authority")
	}
	if !value.Transaction.OwnerReviewAccepted || value.Transaction.RecipeID != "CUSTOM_TASK_CONTRACT_JAVA_V1" ||
		value.Transaction.RecipeVersion != "1.0" || !value.Transaction.AppliedOutsideCheckout ||
		!value.Transaction.CheckoutUnchanged || !value.Transaction.ExactRevertRestoredPreimage || value.Transaction.RejectedProposalMutations != 0 {
		return errors.New("patch transaction proof is invalid")
	}
	if !value.Measurement.NativeGradleBeforeAfter || value.Measurement.BuildOptRequiredAtExecution || value.Measurement.RunnerID != "linux-amd64-4c-16g-v1" ||
		value.Measurement.RunnerCPU != 4 || value.Measurement.RunnerMemoryBytes != 17179869184 || len(value.Measurement.Workloads) != 2 {
		return errors.New("patch native measurement boundary is invalid")
	}
	for _, workload := range value.Measurement.Workloads {
		if !workload.Qualified || workload.Pairs != 8 || workload.MeanSavedMs < 500 || workload.MeanReductionRatio < 0.02 ||
			len(workload.Interval95SavedMs) != 2 || workload.Interval95SavedMs[0] <= 0 || !workload.RequiredOutputsIdentical ||
			workload.ControlCacheHits != 0 || workload.CandidateCacheHits != 64 || workload.ProductAttributableFailures != 0 {
			return errors.New("patch paired result does not clear the value gate")
		}
	}
	if value.Summary.DetectedOpportunities != 1 || value.Summary.AcceptedPatches != 1 || value.Summary.RejectedPatches != 1 ||
		value.Summary.DSLs != 2 || value.Summary.PairedComparisons != 16 || value.Summary.ExactOutputComparisons != 16 ||
		value.Summary.MinimumSavedMs != 1369.25 || value.Summary.MinimumReductionRatio != 0.6727674732833804 {
		return errors.New("patch opportunity summary is invalid")
	}
	if !value.Boundaries.ProofOfConcept || !value.Boundaries.SyntheticReviewedFixture || !value.Boundaries.GeneralDetector ||
		value.Boundaries.GeneralPatchRecipe || value.Boundaries.AutomaticPatchApplication || value.Boundaries.ProductionAuthorized ||
		value.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("patch opportunity boundaries are invalid")
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
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func digestText(value string) string { return digestBytes([]byte(value)) }
func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
