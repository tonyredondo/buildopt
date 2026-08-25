// Command adaptive-fragment-longitudinal normalizes and independently checks
// the frozen AF-013 five-repository chronological evidence.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	contractSchema = "buildopt.specs/poc-adaptive-fragment-longitudinal/v1"
	reportSchema   = "buildopt.poc/adaptive-fragment-longitudinal/v1"
)

type contract struct {
	SchemaVersion  string           `json:"schemaVersion"`
	WorkItem       string           `json:"workItem"`
	FrozenAt       string           `json:"frozenAt"`
	Methodology    methodology      `json:"methodology"`
	Repositories   []repositorySpec `json:"repositories"`
	Classification classification   `json:"classification"`
	Boundaries     boundaries       `json:"boundaries"`
}

type methodology struct {
	Evaluation                        string `json:"evaluation"`
	Control                           string `json:"control"`
	Candidate                         string `json:"candidate"`
	LearningCostChargedOnce           bool   `json:"learningCostChargedOnce"`
	FutureObservationsAvailable       bool   `json:"futureObservationsAvailable"`
	PercentagesAdded                  bool   `json:"percentagesAdded"`
	NewBuildTiming                    bool   `json:"newBuildTiming"`
	InconclusiveWhenComparableMissing bool   `json:"inconclusiveWhenComparableDeltaMissing"`
}

type repositorySpec struct {
	Key                      string   `json:"key"`
	RepositoryID             string   `json:"repositoryId"`
	SourceType               string   `json:"sourceType"`
	SourcePath               string   `json:"sourcePath"`
	SourceSHA256             string   `json:"sourceSha256"`
	SourceSubjectKey         string   `json:"sourceSubjectKey,omitempty"`
	Workflow                 []string `json:"workflow"`
	ExpectedObservationCount int      `json:"expectedObservationCount"`
}

type classification struct {
	NetPositive             string `json:"netPositive"`
	NetNegative             string `json:"netNegative"`
	Inconclusive            string `json:"inconclusive"`
	TerminalBreadthRequired int    `json:"terminalBreadthRequired"`
}

type boundaries struct {
	ProofOfConcept                 bool   `json:"proofOfConcept"`
	ProductionAuthorized           bool   `json:"productionAuthorized"`
	FreshPerformanceClaim          bool   `json:"freshPerformanceClaim"`
	RepositorySpecificProductRules bool   `json:"repositorySpecificProductRules"`
	SoakRequired                   bool   `json:"soakRequired"`
	DesignPartnerRequired          bool   `json:"designPartnerRequired"`
	TestOptimization               string `json:"testOptimization"`
}

type rawSubject struct {
	SchemaVersion string `json:"schemaVersion"`
	Repository    struct {
		ID             string `json:"id"`
		TargetRevision string `json:"targetRevision"`
	} `json:"repository"`
	Key           string           `json:"key"`
	Qualification rawQualification `json:"qualification"`
	Observations  []rawObservation `json:"observations"`
	Economics     rawEconomics     `json:"economics"`
	Acceptance    rawAcceptance    `json:"acceptance"`
}

type rawQualification struct {
	TargetRevision string `json:"targetRevision"`
}

type rawObservation struct {
	Sequence                      int    `json:"sequence"`
	Revision                      string `json:"revision"`
	ParentRevision                string `json:"parentRevision"`
	Order                         string `json:"order"`
	ControlDurationMS             int64  `json:"controlDurationMs"`
	CandidateDurationMS           int64  `json:"candidateDurationMs"`
	SavedMS                       int64  `json:"savedMs"`
	ControlRequiredOutputSHA256   string `json:"controlRequiredOutputSha256"`
	CandidateRequiredOutputSHA256 string `json:"candidateRequiredOutputSha256"`
	ExactRequiredOutputs          bool   `json:"exactRequiredOutputs"`
	Selected                      bool   `json:"selected"`
	NativeRetained                bool   `json:"nativeRetained"`
	Reason                        string `json:"reason"`
	ExecutionMode                 string `json:"executionMode"`
	SourceOrigin                  string `json:"sourceOrigin"`
}

type rawEconomics struct {
	QualificationAndPublicationCostMS int64 `json:"qualificationAndPublicationCostMs"`
}

type rawAcceptance struct {
	ProductFailures      int  `json:"productFailures"`
	ExactRequiredOutputs bool `json:"exactRequiredOutputs"`
	PublicAncestry       bool `json:"publicAncestry"`
}

type crossCommit struct {
	SchemaVersion string `json:"schemaVersion"`
	Subjects      []struct {
		Key           string           `json:"key"`
		Repository    string           `json:"repository"`
		Qualification rawQualification `json:"qualification"`
		Observations  []rawObservation `json:"observations"`
		Economics     rawEconomics     `json:"economics"`
		Acceptance    rawAcceptance    `json:"acceptance"`
	} `json:"subjects"`
}

type producerBound struct {
	Repository struct {
		ID                      string `json:"id"`
		TargetRevision          string `json:"targetRevision"`
		FirstDescendantRevision string `json:"firstDescendantRevision"`
	} `json:"repository"`
	Economics       rawEconomics `json:"economics"`
	FirstDescendant struct {
		Selection struct {
			Reason string `json:"reason"`
		} `json:"selection"`
		Portability struct {
			Status        string `json:"status"`
			DifferingPath string `json:"differingPath"`
		} `json:"portability"`
	} `json:"firstDescendant"`
	Acceptance struct {
		ProductFailures             int  `json:"productFailures"`
		ExactTransportGatePreserved bool `json:"exactTransportGatePreserved"`
	} `json:"acceptance"`
}

type report struct {
	SchemaVersion string        `json:"schemaVersion"`
	WorkItem      string        `json:"workItem"`
	FrozenAt      string        `json:"frozenAt"`
	Contract      evidenceLink  `json:"contract"`
	Methodology   methodology   `json:"methodology"`
	Rows          []resultRow   `json:"rows"`
	Summary       reportSummary `json:"summary"`
	Boundaries    boundaries    `json:"boundaries"`
}

type evidenceLink struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type resultRow struct {
	Key                    string        `json:"key"`
	RepositoryID           string        `json:"repositoryId"`
	Workflow               []string      `json:"workflow"`
	TargetRevision         string        `json:"targetRevision"`
	Source                 evidenceLink  `json:"source"`
	SourceSchema           string        `json:"sourceSchema"`
	AttemptedBuilds        int           `json:"attemptedBuilds"`
	MeasuredBuilds         int           `json:"measuredBuilds"`
	SelectedBuilds         int           `json:"selectedBuilds"`
	NativeRetainedBuilds   int           `json:"nativeRetainedBuilds"`
	QualificationCostMS    int64         `json:"qualificationCostMs"`
	CumulativeGrossSavedMS int64         `json:"cumulativeGrossSavedMs"`
	CumulativeNetSavedMS   int64         `json:"cumulativeNetSavedMs"`
	WorstBuildRegressionMS int64         `json:"worstBuildRegressionMs"`
	FirstPaybackBuild      *int          `json:"firstPaybackBuild"`
	ExactComparableOutputs bool          `json:"exactComparableOutputs"`
	ProductFailures        int           `json:"productFailures"`
	Observations           []observation `json:"observations"`
	Exclusions             []exclusion   `json:"exclusions"`
	Outcome                string        `json:"outcome"`
	OutcomeReason          string        `json:"outcomeReason"`
}

type observation struct {
	Sequence             int    `json:"sequence"`
	Revision             string `json:"revision"`
	ParentRevision       string `json:"parentRevision"`
	MaxSourceSequence    int    `json:"maxSourceSequence"`
	Order                string `json:"order"`
	ControlWallMS        int64  `json:"controlWallMs"`
	BuildOptWallMS       int64  `json:"buildOptWallMs"`
	SignedDeltaMS        int64  `json:"signedDeltaMs"`
	CumulativeNetMS      int64  `json:"cumulativeNetMs"`
	Selected             bool   `json:"selected"`
	NativeRetained       bool   `json:"nativeRetained"`
	DispositionReason    string `json:"dispositionReason"`
	ExecutionMode        string `json:"executionMode"`
	ExactRequiredOutputs bool   `json:"exactRequiredOutputs"`
	SourceOrigin         string `json:"sourceOrigin"`
}

type exclusion struct {
	Revision string `json:"revision"`
	Reason   string `json:"reason"`
	Detail   string `json:"detail"`
}

type reportSummary struct {
	RepositoryCount              int  `json:"repositoryCount"`
	ClosedRows                   int  `json:"closedRows"`
	NetPositiveRows              int  `json:"netPositiveRows"`
	NetNegativeRows              int  `json:"netNegativeRows"`
	InconclusiveRows             int  `json:"inconclusiveRows"`
	MeasuredBuilds               int  `json:"measuredBuilds"`
	SelectedBuilds               int  `json:"selectedBuilds"`
	ExactComparableBuilds        int  `json:"exactComparableBuilds"`
	ProductFailures              int  `json:"productFailures"`
	CompleteSignedMeasuredDeltas bool `json:"completeSignedMeasuredDeltas"`
	TerminalBreadthRequired      int  `json:"terminalBreadthRequired"`
	TerminalBreadthObserved      int  `json:"terminalBreadthObserved"`
	AggregateDecisionDeferred    bool `json:"aggregateDecisionDeferred"`
}

func main() {
	if len(os.Args) != 4 || (os.Args[1] != "assemble" && os.Args[1] != "check") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-longitudinal assemble|check CONTRACT_JSON RESULT_JSON")
		os.Exit(64)
	}
	contractPath, err := filepath.Abs(os.Args[2])
	if err != nil {
		fatal(err)
	}
	// Contracts live under specs/. Resolve evidence paths from its parent.
	repositoryRoot := filepath.Dir(filepath.Dir(contractPath))
	contractBytes, err := os.ReadFile(os.Args[2])
	if err != nil {
		fatal(err)
	}
	var spec contract
	if err := decodeStrict(contractBytes, &spec); err != nil {
		fatal(err)
	}
	assembled, err := assemble(repositoryRoot, filepath.ToSlash(mustRelative(repositoryRoot, contractPath)), contractBytes, spec)
	if err != nil {
		fatal(err)
	}
	encoded, err := json.MarshalIndent(assembled, "", "  ")
	if err != nil {
		fatal(err)
	}
	encoded = append(encoded, '\n')
	if os.Args[1] == "assemble" {
		if err := os.WriteFile(os.Args[3], encoded, 0o600); err != nil {
			fatal(err)
		}
		return
	}
	actual, err := os.ReadFile(os.Args[3])
	if err != nil {
		fatal(err)
	}
	if !bytes.Equal(actual, encoded) {
		fatal(errors.New("adaptive fragment longitudinal result is not the canonical source recomputation"))
	}
	if err := validateReport(assembled); err != nil {
		fatal(err)
	}
}

func assemble(root, contractPath string, contractBytes []byte, spec contract) (report, error) {
	if err := validateContract(spec); err != nil {
		return report{}, err
	}
	result := report{
		SchemaVersion: reportSchema, WorkItem: spec.WorkItem, FrozenAt: spec.FrozenAt,
		Contract:    evidenceLink{Path: contractPath, SHA256: digest(contractBytes)},
		Methodology: spec.Methodology, Rows: []resultRow{}, Boundaries: spec.Boundaries,
	}
	for _, repository := range spec.Repositories {
		row, err := assembleRow(root, repository)
		if err != nil {
			return report{}, fmt.Errorf("%s: %w", repository.Key, err)
		}
		result.Rows = append(result.Rows, row)
	}
	result.Summary = summarize(result.Rows, spec.Classification.TerminalBreadthRequired)
	return result, validateReport(result)
}

func assembleRow(root string, spec repositorySpec) (resultRow, error) {
	path := filepath.Join(root, filepath.FromSlash(spec.SourcePath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return resultRow{}, err
	}
	if digest(raw) != spec.SourceSHA256 {
		return resultRow{}, errors.New("declared source digest does not match")
	}
	var subject rawSubject
	sourceSchema := ""
	exclusions := []exclusion{}
	attempted := spec.ExpectedObservationCount
	switch spec.SourceType {
	case "QUALIFIED_LIFETIME_SUBJECT_V2":
		if err := decodeStrict(raw, &subject); err != nil {
			return resultRow{}, err
		}
		sourceSchema = "buildopt.evidence/poc-qualified-lifetime-subject/v2"
		if subject.SchemaVersion != sourceSchema {
			return resultRow{}, errors.New("declared source schema is incompatible")
		}
	case "CROSS_COMMIT_BREADTH_V2_SUBJECT":
		var source crossCommit
		if err := decodeStrict(raw, &source); err != nil {
			return resultRow{}, err
		}
		if source.SchemaVersion != "buildopt.evidence/poc-cross-commit-breadth-v2/v1" {
			return resultRow{}, errors.New("declared source schema is incompatible")
		}
		found := false
		for _, candidate := range source.Subjects {
			if candidate.Key == spec.SourceSubjectKey {
				subject.Key, subject.Repository.ID = candidate.Key, candidate.Repository
				subject.Qualification, subject.Observations = candidate.Qualification, candidate.Observations
				subject.Economics, subject.Acceptance = candidate.Economics, candidate.Acceptance
				found = true
				break
			}
		}
		if !found {
			return resultRow{}, errors.New("declared source subject is unavailable")
		}
		sourceSchema = "buildopt.evidence/poc-cross-commit-breadth-v2/v1"
	case "PRODUCER_BOUND_LIFETIME_GENERALIZATION_V1":
		var source producerBound
		if err := decodeStrict(raw, &source); err != nil {
			return resultRow{}, err
		}
		subject.Repository.ID = source.Repository.ID
		subject.Qualification.TargetRevision = source.Repository.TargetRevision
		subject.Economics = source.Economics
		subject.Acceptance.ProductFailures = source.Acceptance.ProductFailures
		attempted = 1
		exclusions = append(exclusions, exclusion{
			Revision: source.Repository.FirstDescendantRevision,
			Reason:   source.FirstDescendant.Selection.Reason,
			Detail:   source.FirstDescendant.Portability.Status + ":" + source.FirstDescendant.Portability.DifferingPath,
		})
		sourceSchema = "buildopt.evidence/poc-producer-bound-lifetime-generalization/v1"
	default:
		return resultRow{}, errors.New("unsupported longitudinal source type")
	}
	if subject.Repository.ID != spec.RepositoryID || len(subject.Observations) != spec.ExpectedObservationCount {
		return resultRow{}, errors.New("source repository or observation count does not match the frozen contract")
	}
	target := subject.Qualification.TargetRevision
	if target == "" {
		target = subject.Repository.TargetRevision
	}
	row := resultRow{
		Key: spec.Key, RepositoryID: spec.RepositoryID, Workflow: append([]string{}, spec.Workflow...),
		TargetRevision: target, Source: evidenceLink{Path: spec.SourcePath, SHA256: digest(raw)},
		SourceSchema: sourceSchema, AttemptedBuilds: attempted, MeasuredBuilds: len(subject.Observations),
		QualificationCostMS:    subject.Economics.QualificationAndPublicationCostMS,
		ExactComparableOutputs: len(subject.Observations) > 0, ProductFailures: subject.Acceptance.ProductFailures,
		Observations: []observation{}, Exclusions: exclusions,
	}
	cumulative := -row.QualificationCostMS
	for index, source := range subject.Observations {
		if source.Sequence != index+1 || source.SavedMS != source.ControlDurationMS-source.CandidateDurationMS ||
			source.ControlDurationMS <= 0 || source.CandidateDurationMS <= 0 ||
			!source.ExactRequiredOutputs || source.ControlRequiredOutputSHA256 != source.CandidateRequiredOutputSHA256 {
			return resultRow{}, errors.New("source observation is not a complete exact signed pair")
		}
		if !subject.Acceptance.PublicAncestry || source.SourceOrigin != "PUBLIC_UPSTREAM_COMMIT" {
			return resultRow{}, errors.New("observation sequence is not frozen public ancestry")
		}
		cumulative += source.SavedMS
		row.CumulativeGrossSavedMS += source.SavedMS
		if source.Selected {
			row.SelectedBuilds++
		}
		if source.NativeRetained {
			row.NativeRetainedBuilds++
		}
		if source.SavedMS < row.WorstBuildRegressionMS {
			row.WorstBuildRegressionMS = source.SavedMS
		}
		if row.FirstPaybackBuild == nil && cumulative > 0 {
			ordinal := source.Sequence
			row.FirstPaybackBuild = &ordinal
		}
		row.Observations = append(row.Observations, observation{
			Sequence: source.Sequence, Revision: source.Revision, ParentRevision: source.ParentRevision,
			MaxSourceSequence: source.Sequence - 1, Order: source.Order,
			ControlWallMS: source.ControlDurationMS, BuildOptWallMS: source.CandidateDurationMS,
			SignedDeltaMS: source.SavedMS, CumulativeNetMS: cumulative,
			Selected: source.Selected, NativeRetained: source.NativeRetained,
			DispositionReason: source.Reason, ExecutionMode: source.ExecutionMode,
			ExactRequiredOutputs: source.ExactRequiredOutputs, SourceOrigin: source.SourceOrigin,
		})
	}
	row.CumulativeNetSavedMS = cumulative
	if len(row.Exclusions) > 0 || row.MeasuredBuilds != row.AttemptedBuilds {
		row.Outcome, row.OutcomeReason = "INCONCLUSIVE", "COMPARABLE_LONGITUDINAL_DELTA_UNAVAILABLE"
	} else if cumulative > 0 {
		row.Outcome, row.OutcomeReason = "NET_POSITIVE", "CUMULATIVE_NET_VALUE_POSITIVE"
	} else {
		row.Outcome, row.OutcomeReason = "NET_NEGATIVE", "CUMULATIVE_NET_VALUE_NON_POSITIVE"
	}
	return row, nil
}

func summarize(rows []resultRow, breadth int) reportSummary {
	summary := reportSummary{RepositoryCount: len(rows), ClosedRows: len(rows), TerminalBreadthRequired: breadth, AggregateDecisionDeferred: true, CompleteSignedMeasuredDeltas: true}
	for _, row := range rows {
		summary.MeasuredBuilds += row.MeasuredBuilds
		summary.SelectedBuilds += row.SelectedBuilds
		summary.ProductFailures += row.ProductFailures
		for _, observation := range row.Observations {
			if observation.ExactRequiredOutputs {
				summary.ExactComparableBuilds++
			}
			if observation.SignedDeltaMS != observation.ControlWallMS-observation.BuildOptWallMS {
				summary.CompleteSignedMeasuredDeltas = false
			}
		}
		switch row.Outcome {
		case "NET_POSITIVE":
			summary.NetPositiveRows++
		case "NET_NEGATIVE":
			summary.NetNegativeRows++
		case "INCONCLUSIVE":
			summary.InconclusiveRows++
		}
	}
	summary.TerminalBreadthObserved = summary.NetPositiveRows
	return summary
}

func validateContract(spec contract) error {
	if spec.SchemaVersion != contractSchema || spec.WorkItem != "AF-013" || len(spec.Repositories) != 5 ||
		spec.Methodology.Evaluation != "CHRONOLOGICAL_PREQUENTIAL_REPLAY_OF_DIRECT_MEASUREMENTS" ||
		spec.Methodology.Control != "OPTIMIZED_NATIVE_GRADLE" || !spec.Methodology.LearningCostChargedOnce ||
		spec.Methodology.FutureObservationsAvailable || spec.Methodology.PercentagesAdded || spec.Methodology.NewBuildTiming ||
		!spec.Methodology.InconclusiveWhenComparableMissing || spec.Classification.TerminalBreadthRequired != 3 ||
		!spec.Boundaries.ProofOfConcept || spec.Boundaries.ProductionAuthorized || spec.Boundaries.RepositorySpecificProductRules ||
		spec.Boundaries.SoakRequired || spec.Boundaries.DesignPartnerRequired || spec.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("adaptive fragment longitudinal contract is invalid")
	}
	seen := map[string]bool{}
	for _, repository := range spec.Repositories {
		if repository.Key == "" || repository.RepositoryID == "" || repository.SourcePath == "" || len(repository.SourceSHA256) != 64 || len(repository.Workflow) == 0 || seen[repository.RepositoryID] {
			return errors.New("adaptive fragment longitudinal repository contract is invalid")
		}
		seen[repository.RepositoryID] = true
	}
	return nil
}

func validateReport(value report) error {
	if value.SchemaVersion != reportSchema || value.WorkItem != "AF-013" || len(value.Rows) != 5 ||
		value.Summary.RepositoryCount != 5 || value.Summary.ClosedRows != 5 ||
		value.Summary.MeasuredBuilds != 14 || value.Summary.ExactComparableBuilds != 14 ||
		value.Summary.SelectedBuilds != 2 || value.Summary.ProductFailures != 0 ||
		!value.Summary.CompleteSignedMeasuredDeltas || value.Summary.NetPositiveRows != 2 ||
		value.Summary.NetNegativeRows != 2 || value.Summary.InconclusiveRows != 1 ||
		value.Summary.TerminalBreadthRequired != 3 || value.Summary.TerminalBreadthObserved != 2 ||
		!value.Summary.AggregateDecisionDeferred {
		return errors.New("adaptive fragment longitudinal summary is invalid")
	}
	for _, row := range value.Rows {
		if row.ProductFailures != 0 || len(row.Observations) != row.MeasuredBuilds {
			return errors.New("adaptive fragment longitudinal row is invalid")
		}
		for index, observation := range row.Observations {
			if observation.Sequence != index+1 || observation.MaxSourceSequence != index ||
				observation.SignedDeltaMS != observation.ControlWallMS-observation.BuildOptWallMS ||
				!observation.ExactRequiredOutputs {
				return errors.New("adaptive fragment longitudinal observation is invalid")
			}
		}
	}
	return nil
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func mustRelative(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		panic(err)
	}
	return relative
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
