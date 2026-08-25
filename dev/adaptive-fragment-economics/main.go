// Command adaptive-fragment-economics produces and validates the AF-005
// recomputable signed-value report. It executes no Gradle build or fragment.
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

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-economics/v1"
	outcome      = "FRAGMENT_ECONOMICS_RECOMPUTABLE"
)

type report struct {
	SchemaVersion string                                         `json:"schemaVersion"`
	WorkItem      string                                         `json:"workItem"`
	CapturedAt    string                                         `json:"capturedAt"`
	Source        source                                         `json:"source"`
	Policy        policy                                         `json:"policy"`
	Assessments   map[string]adaptivefragment.EconomicAssessment `json:"assessments"`
	Invariants    invariants                                     `json:"invariants"`
	Summary       summary                                        `json:"summary"`
	Boundaries    boundaries                                     `json:"boundaries"`
	Outcome       string                                         `json:"outcome"`
}

type source struct {
	SchemaVersion string `json:"schemaVersion"`
	SummarySHA256 string `json:"summarySha256"`
	KafkaSHA256   string `json:"kafkaResultSha256"`
	TimeBasis     string `json:"timeBasis"`
}

type policy struct {
	DecayPermille  uint64   `json:"decayPermille"`
	Horizons       []uint64 `json:"horizons"`
	RegretBudgetMs uint64   `json:"regretBudgetMs"`
}

type invariants struct {
	NegativeBuildValueDeltaMs   int64  `json:"negativeBuildValueDeltaMs"`
	DeclaredAsyncCostReferences uint64 `json:"declaredAsyncCostReferences"`
	UniqueAsyncCostEvents       uint64 `json:"uniqueAsyncCostEvents"`
	AsyncCostChargedMs          uint64 `json:"asyncCostChargedMs"`
	ObservedRewriteCount        uint64 `json:"observedRewriteCount"`
	AdditivePercentageCount     uint64 `json:"additivePercentageCount"`
	MeasurementOnlyBuildCount   uint64 `json:"measurementOnlyBuildCount"`
}

type summary struct {
	AssessmentCount              int    `json:"assessmentCount"`
	RetainedRequestedBuilds      uint64 `json:"retainedRequestedBuilds"`
	RetainedActivatedBuilds      uint64 `json:"retainedActivatedBuilds"`
	RetainedGrossSavedMs         int64  `json:"retainedGrossSavedMs"`
	RetainedSynchronousCostMs    uint64 `json:"retainedSynchronousCostMs"`
	RetainedAsynchronousCostMs   uint64 `json:"retainedAsynchronousCostMs"`
	RetainedCumulativeNetMs      int64  `json:"retainedCumulativeNetMs"`
	RetainedObservedPaybackBuild uint64 `json:"retainedObservedPaybackBuild"`
}

type boundaries struct {
	ProofOfConcept       bool   `json:"proofOfConcept"`
	SyntheticTimingClaim bool   `json:"syntheticTimingClaim"`
	FragmentAttribution  bool   `json:"fragmentAttribution"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	ProductionAuthorized bool   `json:"productionAuthorized"`
	TestOptimization     string `json:"testOptimization"`
}

type frozenSummary struct {
	SchemaVersion string `json:"schemaVersion"`
	CapturedAt    string `json:"capturedAt"`
	Subjects      []struct {
		RepositoryID string         `json:"repositoryId"`
		Lifetime     frozenLifetime `json:"lifetime"`
	} `json:"subjects"`
}

type frozenLifetime struct {
	EligibleDescendants               int    `json:"eligibleDescendants"`
	SelectedReplays                   int    `json:"selectedReplays"`
	SelectedReplaySavedMs             int64  `json:"selectedReplaySavedMs"`
	NativeRetentionWrapperCostMs      uint64 `json:"nativeRetentionWrapperCostMs"`
	QualificationAndPublicationCostMs uint64 `json:"qualificationAndPublicationCostMs"`
	CumulativeNetSavedMs              int64  `json:"cumulativeNetSavedMs"`
}

type frozenKafka struct {
	Observations []struct {
		Sequence          uint64 `json:"sequence"`
		Revision          string `json:"revision"`
		Selected          bool   `json:"selected"`
		SavedMs           int64  `json:"savedMs"`
		WrapperOverheadMs uint64 `json:"wrapperOverheadMs"`
	} `json:"observations"`
}

func main() {
	sourceDir := flag.String("source", "", "frozen lifetime breadth V3 evidence directory")
	output := flag.String("output", "", "write an AF-005 economics report")
	validate := flag.String("validate", "", "validate an AF-005 economics report")
	flag.Parse()
	if flag.NArg() != 0 || *sourceDir == "" || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-economics --source <dir> (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	expected, err := buildReport(*sourceDir)
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("economic report does not match recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment economics failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment economics: FRAGMENT_ECONOMICS_RECOMPUTABLE")
	}
}

func buildReport(sourceDir string) (report, error) {
	summaryPath := filepath.Join(sourceDir, "summary.json")
	kafkaPath := filepath.Join(sourceDir, "apache-kafka", "result.json")
	var frozen frozenSummary
	var kafka frozenKafka
	if err := readJSON(summaryPath, &frozen); err != nil {
		return report{}, err
	}
	if err := readJSON(kafkaPath, &kafka); err != nil {
		return report{}, err
	}
	currentPolicy := policy{DecayPermille: 900, Horizons: []uint64{1, 5, 10}, RegretBudgetMs: 100000}
	retained, err := retainedKafkaAssessment(frozen, kafka, currentPolicy)
	if err != nil {
		return report{}, err
	}
	positiveSeries := syntheticPositiveSeries(currentPolicy)
	positive, err := adaptivefragment.AssessEconomics(positiveSeries)
	if err != nil {
		return report{}, err
	}
	negativeSeries := syntheticNegativeSeries(currentPolicy)
	negative, err := adaptivefragment.AssessEconomics(negativeSeries)
	if err != nil {
		return report{}, err
	}
	withoutNegative := positiveSeries
	withoutNegative.Observations = withoutNegative.Observations[:2]
	positiveOnly, err := adaptivefragment.AssessEconomics(withoutNegative)
	if err != nil {
		return report{}, err
	}
	differentProjection := positiveSeries
	differentProjection.Policy.DecayPermille = 500
	differentProjection.Policy.Horizons = []uint64{2, 8}
	projectedDifferently, err := adaptivefragment.AssessEconomics(differentProjection)
	if err != nil {
		return report{}, err
	}
	observedRewriteCount := uint64(0)
	if !reflect.DeepEqual(positive.Entry, projectedDifferently.Entry) || !reflect.DeepEqual(positive.Recurrence, projectedDifferently.Recurrence) {
		observedRewriteCount = 1
	}
	result := report{
		SchemaVersion: reportSchema, WorkItem: "AF-005", CapturedAt: frozen.CapturedAt,
		Source: source{
			SchemaVersion: frozen.SchemaVersion, SummarySHA256: fileSHA(summaryPath), KafkaSHA256: fileSHA(kafkaPath),
			TimeBasis: "ORDER_PRESERVING_SEQUENCE_ANCHORED_FOR_TYPED_LEDGER",
		},
		Policy: currentPolicy,
		Assessments: map[string]adaptivefragment.EconomicAssessment{
			"retainedKafkaComposition":  retained,
			"syntheticNegativeFragment": negative,
			"syntheticSignedFragment":   positive,
		},
		Invariants: invariants{
			NegativeBuildValueDeltaMs:   positive.Entry.CumulativeNetMs - positiveOnly.Entry.CumulativeNetMs,
			DeclaredAsyncCostReferences: 2, UniqueAsyncCostEvents: positive.UniqueCostEventCount,
			AsyncCostChargedMs:   positive.Entry.OutOfBandLearningCostMs,
			ObservedRewriteCount: observedRewriteCount,
		},
		Boundaries: boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"},
		Outcome:    outcome,
	}
	result.Summary = summarize(result.Assessments)
	return result, nil
}

func retainedKafkaAssessment(frozen frozenSummary, kafka frozenKafka, currentPolicy policy) (adaptivefragment.EconomicAssessment, error) {
	var lifetime *frozenLifetime
	for index := range frozen.Subjects {
		if frozen.Subjects[index].RepositoryID == "apache/kafka" {
			lifetime = &frozen.Subjects[index].Lifetime
			break
		}
	}
	if lifetime == nil || len(kafka.Observations) != lifetime.EligibleDescendants {
		return adaptivefragment.EconomicAssessment{}, errors.New("retained Kafka evidence is incomplete")
	}
	learningCost := adaptivefragment.EconomicCostEvent{ID: digest("kafka-qualification-publication"), AmountMs: lifetime.QualificationAndPublicationCostMs}
	series := adaptivefragment.EconomicSeries{
		Scope: adaptivefragment.EconomicScopeComposition, FamilyID: digest("kafka-retained-composition-family"),
		RevisionID: digest("kafka-retained-composition-revision"), FragmentGeneration: 1,
		EvidenceExpiresAt: "2026-09-25T11:00:00Z",
		Policy:            adaptivefragment.EconomicPolicy{DecayPermille: currentPolicy.DecayPermille, Horizons: currentPolicy.Horizons, RegretBudgetMs: currentPolicy.RegretBudgetMs},
	}
	for index, observation := range kafka.Observations {
		current := adaptivefragment.EconomicObservation{
			ObservationID: digest("kafka:" + observation.Revision), Sequence: observation.Sequence,
			ObservedAt: fmt.Sprintf("2026-08-25T10:%02d:00Z", index), Compatible: true, Activated: observation.Selected,
		}
		if observation.Selected {
			current.GrossSavedMs = observation.SavedMs
		} else {
			current.SynchronousOverheadMs = observation.WrapperOverheadMs
		}
		if index == 0 {
			current.AsynchronousCostEvents = []adaptivefragment.EconomicCostEvent{learningCost}
		}
		series.Observations = append(series.Observations, current)
	}
	assessment, err := adaptivefragment.AssessEconomics(series)
	if err != nil {
		return adaptivefragment.EconomicAssessment{}, err
	}
	if assessment.Recurrence.ActivatedBuilds != uint64(lifetime.SelectedReplays) ||
		assessment.Entry.GrossSavedMs != lifetime.SelectedReplaySavedMs ||
		assessment.Entry.SynchronousOverheadMs != lifetime.NativeRetentionWrapperCostMs ||
		assessment.Entry.OutOfBandLearningCostMs != lifetime.QualificationAndPublicationCostMs ||
		assessment.Entry.CumulativeNetMs != lifetime.CumulativeNetSavedMs {
		return adaptivefragment.EconomicAssessment{}, errors.New("retained Kafka economics are not reproducible")
	}
	return assessment, nil
}

func syntheticPositiveSeries(currentPolicy policy) adaptivefragment.EconomicSeries {
	learning := adaptivefragment.EconomicCostEvent{ID: digest("synthetic-learning"), AmountMs: 500}
	return adaptivefragment.EconomicSeries{
		Scope: adaptivefragment.EconomicScopeFragment, FamilyID: digest("synthetic-positive-family"),
		RevisionID: digest("synthetic-positive-revision"), FragmentGeneration: 1,
		EvidenceExpiresAt: "2026-09-25T11:00:00Z",
		Observations: []adaptivefragment.EconomicObservation{
			{ObservationID: digest("positive-1"), Sequence: 1, ObservedAt: "2026-08-25T10:00:00Z", Compatible: true, Activated: true, GrossSavedMs: 400, SynchronousOverheadMs: 20, AsynchronousCostEvents: []adaptivefragment.EconomicCostEvent{learning}},
			{ObservationID: digest("positive-2"), Sequence: 2, ObservedAt: "2026-08-25T10:01:00Z", Compatible: true, Activated: true, GrossSavedMs: 300, SynchronousOverheadMs: 20, AsynchronousCostEvents: []adaptivefragment.EconomicCostEvent{learning}},
			{ObservationID: digest("positive-3"), Sequence: 3, ObservedAt: "2026-08-25T10:02:00Z", Compatible: true, Activated: true, GrossSavedMs: -100, SynchronousOverheadMs: 20},
		},
		Policy: adaptivefragment.EconomicPolicy{DecayPermille: currentPolicy.DecayPermille, Horizons: currentPolicy.Horizons, RegretBudgetMs: 200},
	}
}

func syntheticNegativeSeries(currentPolicy policy) adaptivefragment.EconomicSeries {
	return adaptivefragment.EconomicSeries{
		Scope: adaptivefragment.EconomicScopeFragment, FamilyID: digest("synthetic-negative-family"),
		RevisionID: digest("synthetic-negative-revision"), FragmentGeneration: 1,
		EvidenceExpiresAt: "2026-09-25T11:00:00Z",
		Observations: []adaptivefragment.EconomicObservation{
			{ObservationID: digest("negative-1"), Sequence: 1, ObservedAt: "2026-08-25T10:00:00Z", Compatible: true, Activated: true, GrossSavedMs: -300, SynchronousOverheadMs: 50},
		},
		Policy: adaptivefragment.EconomicPolicy{DecayPermille: currentPolicy.DecayPermille, Horizons: currentPolicy.Horizons, RegretBudgetMs: 100},
	}
}

func summarize(assessments map[string]adaptivefragment.EconomicAssessment) summary {
	retained := assessments["retainedKafkaComposition"]
	return summary{
		AssessmentCount: len(assessments), RetainedRequestedBuilds: retained.Entry.RequestedBuildCount,
		RetainedActivatedBuilds: retained.Recurrence.ActivatedBuilds, RetainedGrossSavedMs: retained.Entry.GrossSavedMs,
		RetainedSynchronousCostMs:    retained.Entry.SynchronousOverheadMs,
		RetainedAsynchronousCostMs:   retained.Entry.OutOfBandLearningCostMs,
		RetainedCumulativeNetMs:      retained.Entry.CumulativeNetMs,
		RetainedObservedPaybackBuild: retained.ObservedBreakEvenBuild,
	}
}

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-005" || candidate.Outcome != outcome ||
		candidate.CapturedAt == "" || candidate.Source.SchemaVersion != "buildopt.evidence/poc-lifetime-breadth/v3" ||
		!isSHA(candidate.Source.SummarySHA256) || !isSHA(candidate.Source.KafkaSHA256) ||
		candidate.Source.TimeBasis != "ORDER_PRESERVING_SEQUENCE_ANCHORED_FOR_TYPED_LEDGER" || len(candidate.Assessments) != 3 {
		return errors.New("economic report identity is invalid")
	}
	if candidate.Invariants.NegativeBuildValueDeltaMs >= 0 || candidate.Invariants.DeclaredAsyncCostReferences != 2 ||
		candidate.Invariants.UniqueAsyncCostEvents != 1 || candidate.Invariants.AsyncCostChargedMs != 500 ||
		candidate.Invariants.ObservedRewriteCount != 0 || candidate.Invariants.AdditivePercentageCount != 0 ||
		candidate.Invariants.MeasurementOnlyBuildCount != 0 {
		return errors.New("economic report invariants are invalid")
	}
	if !reflect.DeepEqual(candidate.Summary, summarize(candidate.Assessments)) || candidate.Summary.AssessmentCount != 3 ||
		candidate.Summary.RetainedRequestedBuilds != 6 || candidate.Summary.RetainedActivatedBuilds != 1 ||
		candidate.Summary.RetainedGrossSavedMs != 135127 || candidate.Summary.RetainedSynchronousCostMs != 42040 ||
		candidate.Summary.RetainedAsynchronousCostMs != 10560 || candidate.Summary.RetainedCumulativeNetMs != 82527 ||
		candidate.Summary.RetainedObservedPaybackBuild != 2 {
		return errors.New("economic report retained evidence is invalid")
	}
	negative := candidate.Assessments["syntheticNegativeFragment"]
	if negative.Entry.CumulativeNetMs != -350 || negative.Regret.ObservedDownsideMs != 350 || negative.Regret.WithinBudget {
		return errors.New("economic report regret vector is invalid")
	}
	if !candidate.Boundaries.ProofOfConcept || candidate.Boundaries.SyntheticTimingClaim || candidate.Boundaries.FragmentAttribution ||
		candidate.Boundaries.ActivationAuthorized || candidate.Boundaries.ProductionAuthorized || candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("economic report boundaries are invalid")
	}
	return nil
}

func readJSONStrict(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON documents are not allowed")
	}
	return nil
}

func readJSON(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, target)
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte("buildopt-af005-v1\x00" + value))
	return hex.EncodeToString(sum[:])
}

func fileSHA(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func isSHA(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
