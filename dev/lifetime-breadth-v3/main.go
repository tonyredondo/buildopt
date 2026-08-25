// Command lifetime-breadth-v3 assembles and verifies the five-repository
// ordinary-build lifetime experiment from its retained raw subject evidence.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	contractSchema  = "buildopt.specs/poc-lifetime-breadth/v3"
	summarySchema   = "buildopt.evidence/poc-lifetime-breadth/v3"
	earlySchema     = "buildopt.evidence/poc-lifetime-breadth-subject/v3-early"
	v2SubjectSchema = "buildopt.evidence/poc-qualified-lifetime-subject/v2"
)

type contract struct {
	SchemaVersion  string `json:"schemaVersion"`
	WorkItem       string `json:"workItem"`
	FrozenSubjects struct {
		Path            string   `json:"path"`
		SHA256          string   `json:"sha256"`
		RepositoryCount int      `json:"repositoryCount"`
		Repositories    []string `json:"repositories"`
	} `json:"frozenSubjects"`
	Learning struct {
		HistoricalCompatibleMatchesRequired    int   `json:"historicalCompatibleMatchesRequired"`
		MaximumProjectedPaybackMatches         int   `json:"maximumProjectedPaybackMatches"`
		MaximumRequestedBuildsPerQualification int   `json:"maximumRequestedBuildsPerQualification"`
		EarlyRetentionBuildCounts              []int `json:"earlyRetentionBuildCounts"`
		RobustPairs                            int   `json:"robustPairs"`
		MinimumPositivePairs                   int   `json:"minimumPositivePairs"`
		PositiveIntervalRequired               bool  `json:"positiveIntervalRequired"`
		CandidateP95NonRegressive              bool  `json:"candidateP95NonRegressive"`
		PositiveFirstPairOnlyContinuesLearning bool  `json:"positiveFirstPairOnlyContinuesLearning"`
		PositiveEconomicsDoesNotQualifyAlone   bool  `json:"positiveEconomicsDoesNotQualifyAlone"`
	} `json:"learning"`
	Acceptance struct {
		AllFiveSubjectsObserved                 bool    `json:"allFiveSubjectsObserved"`
		SameExecutableSHA256                    bool    `json:"sameExecutableSha256"`
		ExactOutputsForEveryRequestedBuild      bool    `json:"exactOutputsForEveryRequestedBuild"`
		ZeroProductAttributableFailures         bool    `json:"zeroProductAttributableFailures"`
		MinimumNetPositiveRepositoryFamilies    int     `json:"minimumNetPositiveRepositoryFamilies"`
		MinimumEligibleDescendantSelectionRatio float64 `json:"minimumEligibleDescendantSelectionRatio"`
		TerminalPassDecision                    string  `json:"terminalPassDecision"`
		TerminalFailDecision                    string  `json:"terminalFailDecision"`
	} `json:"acceptance"`
	Boundaries boundaries `json:"boundaries"`
}

type subjectSpec struct {
	Repositories []repositorySpec `json:"repositories"`
}

type repositorySpec struct {
	Key            string   `json:"key"`
	RepositoryID   string   `json:"repositoryId"`
	PublicRevision string   `json:"publicRevision"`
	TargetRevision string   `json:"targetRevision"`
	Observations   []string `json:"observations"`
}

type boundaries struct {
	ProofOfConcept        bool   `json:"proofOfConcept"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	SoakRequired          bool   `json:"soakRequired"`
	DesignPartnerRequired bool   `json:"designPartnerRequired"`
	TestOptimization      string `json:"testOptimization"`
}

type ordinaryEconomics struct {
	SchemaVersion               string  `json:"schemaVersion"`
	Decision                    string  `json:"decision"`
	Reason                      string  `json:"reason"`
	RequestedBuilds             int     `json:"requestedBuilds"`
	MeasurementOnlyBuilds       int     `json:"measurementOnlyBuilds"`
	CompatibleBuilds            int     `json:"compatibleBuilds"`
	SuccessfulBuilds            int     `json:"successfulBuilds"`
	ExactOutputBuilds           int     `json:"exactOutputBuilds"`
	StructurallyPortableBuilds  int     `json:"structurallyPortableBuilds"`
	ObservedPairs               int     `json:"observedPairs"`
	MeanSavedMS                 float64 `json:"meanSavedMs"`
	LearningCostMS              int64   `json:"learningCostMs"`
	HistoricalCompatibleMatches int     `json:"historicalCompatibleMatches"`
	MaximumPaybackMatches       int     `json:"maximumPaybackMatches"`
	ProjectedPaybackMatches     int     `json:"projectedPaybackMatches"`
	ProjectedNetSavedMS         float64 `json:"projectedNetSavedMs"`
	CalibrationAuthorized       bool    `json:"calibrationAuthorized"`
	ProductionAuthorized        bool    `json:"productionAuthorized"`
	TestOptimization            string  `json:"testOptimization"`
}

type capture struct {
	BuildOpt struct {
		Revision         string `json:"revision"`
		ExecutableSHA256 string `json:"executableSha256"`
	} `json:"buildopt"`
	ProductFailure              bool `json:"productFailure"`
	MeasurementOnlyWorkflowRuns int  `json:"measurementOnlyWorkflowRuns"`
	IncrementalLearning         struct {
		OrdinaryEconomics ordinaryEconomics `json:"ordinaryEconomics"`
	} `json:"incrementalLearning"`
}

type rawResult struct {
	SchemaVersion string `json:"schemaVersion"`
	Key           string `json:"key"`
	BuildOpt      struct {
		Revision         string `json:"revision"`
		ExecutableSHA256 string `json:"executableSha256"`
	} `json:"buildopt"`
	Repository struct {
		ID string `json:"id"`
	} `json:"repository"`
	Qualification struct {
		ParentRevision string `json:"parentRevision"`
		TargetRevision string `json:"targetRevision"`
		Calibration    struct {
			Qualified          bool      `json:"qualified"`
			PairsMeasured      int       `json:"pairsMeasured"`
			PositivePairs      int       `json:"positivePairs"`
			MeanSavedMS        float64   `json:"meanSavedMs"`
			Interval95SavedMS  []float64 `json:"interval95SavedMs"`
			ControlP95MS       float64   `json:"controlP95Ms"`
			CandidateP95MS     float64   `json:"candidateP95Ms"`
			FallbackSuccessful bool      `json:"fallbackSuccessful"`
		} `json:"calibration"`
		Portability struct {
			Status string `json:"status"`
		} `json:"portability"`
		OrdinaryEconomics ordinaryEconomics `json:"ordinaryEconomics"`
	} `json:"qualification"`
	Observations []struct {
		Selected                    bool   `json:"selected"`
		NativeRetained              bool   `json:"nativeRetained"`
		WrapperMatchesQualification bool   `json:"wrapperMatchesQualification"`
		Reason                      string `json:"reason"`
		SavedMS                     int64  `json:"savedMs"`
		WrapperOverheadMS           int64  `json:"wrapperOverheadMs"`
		ExactRequiredOutputs        bool   `json:"exactRequiredOutputs"`
	} `json:"observations"`
	Economics struct {
		QualificationAndPublicationCostMS int64 `json:"qualificationAndPublicationCostMs"`
	} `json:"economics"`
	Acceptance struct {
		ProductFailures int `json:"productFailures"`
	} `json:"acceptance"`
}

type subjectSummary struct {
	Key              string `json:"key"`
	RepositoryID     string `json:"repositoryId"`
	TargetRevision   string `json:"targetRevision"`
	BuildOptRevision string `json:"buildoptRevision"`
	ExecutableSHA256 string `json:"executableSha256"`
	Qualification    struct {
		Status                      string  `json:"status"`
		RequestedBuilds             int     `json:"requestedBuilds"`
		MeasurementOnlyBuilds       int     `json:"measurementOnlyBuilds"`
		HistoricalCompatibleMatches int     `json:"historicalCompatibleMatches"`
		OrdinaryDecision            string  `json:"ordinaryDecision"`
		OrdinaryReason              string  `json:"ordinaryReason"`
		ObservedPairs               int     `json:"observedPairs"`
		RobustQualified             bool    `json:"robustQualified"`
		PositivePairs               int     `json:"positivePairs"`
		MeanSavedMS                 float64 `json:"meanSavedMs"`
		ExactOutputBuilds           int     `json:"exactOutputBuilds"`
		LearningCostMS              int64   `json:"learningCostMs"`
		ProjectedPaybackMatches     int     `json:"projectedPaybackMatches"`
	} `json:"qualification"`
	Lifetime struct {
		ObservedBuilds                    int    `json:"observedBuilds"`
		EligibleDescendants               int    `json:"eligibleDescendants"`
		SelectedReplays                   int    `json:"selectedReplays"`
		NativeRetainedBuilds              int    `json:"nativeRetainedBuilds"`
		ExactOutputBuilds                 int    `json:"exactOutputBuilds"`
		SelectedReplaySavedMS             int64  `json:"selectedReplaySavedMs"`
		NativeRetentionWrapperCostMS      int64  `json:"nativeRetentionWrapperCostMs"`
		QualificationAndPublicationCostMS int64  `json:"qualificationAndPublicationCostMs"`
		CumulativeNetSavedMS              int64  `json:"cumulativeNetSavedMs"`
		Conclusion                        string `json:"conclusion"`
	} `json:"lifetime"`
	ProductFailures            int    `json:"productFailures"`
	RawResultSHA256            string `json:"rawResultSha256"`
	QualificationCaptureSHA256 string `json:"qualificationCaptureSha256"`
}

type summary struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	CapturedAt    string `json:"capturedAt"`
	BuildOpt      struct {
		SourceRevisions   []string `json:"sourceRevisions"`
		ExecutableSHA256  string   `json:"executableSha256"`
		ContractSHA256    string   `json:"contractSha256"`
		SubjectSpecSHA256 string   `json:"subjectSpecSha256"`
		InstalledPackage  bool     `json:"installedPackage"`
	} `json:"buildopt"`
	Runner struct {
		OperatingSystem string `json:"operatingSystem"`
		Architecture    string `json:"architecture"`
		CPUCount        int    `json:"cpuCount"`
	} `json:"runner"`
	Subjects    []subjectSummary `json:"subjects"`
	Aggregation struct {
		SubjectCount                  int     `json:"subjectCount"`
		NetPositiveSubjects           int     `json:"netPositiveSubjects"`
		RequestedQualificationBuilds  int     `json:"requestedQualificationBuilds"`
		MeasurementOnlyBuilds         int     `json:"measurementOnlyBuilds"`
		EligibleDescendants           int     `json:"eligibleDescendants"`
		SelectedReplays               int     `json:"selectedReplays"`
		SelectionRatio                float64 `json:"selectionRatio"`
		ExactOutputBuilds             int     `json:"exactOutputBuilds"`
		ProductFailures               int     `json:"productFailures"`
		SignedNetSavedMS              int64   `json:"signedNetSavedMs"`
		RepositoryPercentagesAveraged bool    `json:"repositoryPercentagesAveraged"`
		MechanismPercentagesAdded     bool    `json:"mechanismPercentagesAdded"`
	} `json:"aggregation"`
	Decision   string     `json:"decision"`
	Boundaries boundaries `json:"boundaries"`
}

func main() {
	if len(os.Args) != 4 || (os.Args[1] != "assemble" && os.Args[1] != "check") {
		fmt.Fprintln(os.Stderr, "usage: lifetime-breadth-v3 assemble|check CONTRACT_JSON OUTPUT_DIRECTORY")
		os.Exit(64)
	}
	contractPath, output := os.Args[2], os.Args[3]
	want, err := buildSummary(contractPath, output, "", 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := filepath.Join(output, "summary.json")
	if os.Args[1] == "assemble" {
		want.CapturedAt = time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
		raw, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		raw = append(raw, '\n')
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var got summary
	if err := strictJSON(raw, &got); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	want, err = buildSummary(contractPath, output, got.CapturedAt, got.Runner.CPUCount)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !reflect.DeepEqual(got, want) {
		fmt.Fprintln(os.Stderr, "lifetime breadth V3 summary does not match raw evidence")
		os.Exit(1)
	}
	fmt.Printf("Lifetime breadth V3 OK: %d subjects, %d net positive, %d/%d eligible descendants selected, decision %s\n", got.Aggregation.SubjectCount, got.Aggregation.NetPositiveSubjects, got.Aggregation.SelectedReplays, got.Aggregation.EligibleDescendants, got.Decision)
}

func buildSummary(contractPath, output, capturedAt string, cpuCount int) (summary, error) {
	var result summary
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return result, err
	}
	var c contract
	if err := json.Unmarshal(contractRaw, &c); err != nil {
		return result, err
	}
	if err := validateContract(c); err != nil {
		return result, err
	}
	repoRoot := filepath.Dir(filepath.Dir(contractPath))
	subjectPath := filepath.Join(repoRoot, filepath.FromSlash(c.FrozenSubjects.Path))
	subjectRaw, err := os.ReadFile(subjectPath)
	if err != nil {
		return result, err
	}
	if digest(subjectRaw) != c.FrozenSubjects.SHA256 {
		return result, errors.New("frozen lifetime subject specification drifted")
	}
	var subjects subjectSpec
	if err := json.Unmarshal(subjectRaw, &subjects); err != nil {
		return result, err
	}
	if len(subjects.Repositories) != c.FrozenSubjects.RepositoryCount {
		return result, errors.New("unexpected lifetime subject count")
	}
	for index, subject := range subjects.Repositories {
		if subject.RepositoryID != c.FrozenSubjects.Repositories[index] {
			return result, errors.New("frozen lifetime repository list drifted")
		}
	}
	result.SchemaVersion, result.WorkItem, result.CapturedAt = summarySchema, c.WorkItem, capturedAt
	result.BuildOpt.ContractSHA256 = digest(contractRaw)
	result.BuildOpt.SubjectSpecSHA256 = digest(subjectRaw)
	result.BuildOpt.InstalledPackage = true
	result.Runner.OperatingSystem, result.Runner.Architecture = "Linux", "x86_64"
	if cpuCount == 0 {
		cpuCount = runtime.NumCPU()
	}
	result.Runner.CPUCount = cpuCount
	for _, expected := range subjects.Repositories {
		subject, err := readSubject(output, expected, c)
		if err != nil {
			return result, fmt.Errorf("%s: %w", expected.Key, err)
		}
		result.Subjects = append(result.Subjects, subject)
	}
	sort.Slice(result.Subjects, func(i, j int) bool { return result.Subjects[i].Key < result.Subjects[j].Key })
	revisions := map[string]bool{}
	for _, row := range result.Subjects {
		if result.BuildOpt.ExecutableSHA256 == "" {
			result.BuildOpt.ExecutableSHA256 = row.ExecutableSHA256
		}
		if row.ExecutableSHA256 != result.BuildOpt.ExecutableSHA256 {
			return result, errors.New("lifetime subjects did not use one exact executable")
		}
		revisions[row.BuildOptRevision] = true
		result.Aggregation.SubjectCount++
		if row.Lifetime.CumulativeNetSavedMS > 0 {
			result.Aggregation.NetPositiveSubjects++
		}
		result.Aggregation.RequestedQualificationBuilds += row.Qualification.RequestedBuilds
		result.Aggregation.MeasurementOnlyBuilds += row.Qualification.MeasurementOnlyBuilds
		result.Aggregation.EligibleDescendants += row.Lifetime.EligibleDescendants
		result.Aggregation.SelectedReplays += row.Lifetime.SelectedReplays
		result.Aggregation.ExactOutputBuilds += row.Qualification.ExactOutputBuilds + row.Lifetime.ExactOutputBuilds
		result.Aggregation.ProductFailures += row.ProductFailures
		result.Aggregation.SignedNetSavedMS += row.Lifetime.CumulativeNetSavedMS
	}
	for revision := range revisions {
		result.BuildOpt.SourceRevisions = append(result.BuildOpt.SourceRevisions, revision)
	}
	sort.Strings(result.BuildOpt.SourceRevisions)
	if result.Aggregation.EligibleDescendants > 0 {
		result.Aggregation.SelectionRatio = float64(result.Aggregation.SelectedReplays) / float64(result.Aggregation.EligibleDescendants)
	}
	result.Decision = c.Acceptance.TerminalFailDecision
	if result.Aggregation.SubjectCount == c.FrozenSubjects.RepositoryCount && result.Aggregation.NetPositiveSubjects >= c.Acceptance.MinimumNetPositiveRepositoryFamilies && result.Aggregation.ProductFailures == 0 && result.Aggregation.MeasurementOnlyBuilds == 0 && result.Aggregation.SelectionRatio >= c.Acceptance.MinimumEligibleDescendantSelectionRatio {
		result.Decision = c.Acceptance.TerminalPassDecision
	}
	result.Boundaries = c.Boundaries
	return result, nil
}

func readSubject(output string, expected repositorySpec, c contract) (subjectSummary, error) {
	var out subjectSummary
	resultPath := filepath.Join(output, expected.Key, "result.json")
	capturePath := filepath.Join(output, expected.Key, "qualification-capture.json")
	resultRaw, err := os.ReadFile(resultPath)
	if err != nil {
		return out, err
	}
	captureRaw, err := os.ReadFile(capturePath)
	if err != nil {
		return out, err
	}
	var raw rawResult
	if err := json.Unmarshal(resultRaw, &raw); err != nil {
		return out, err
	}
	var cap capture
	if err := json.Unmarshal(captureRaw, &cap); err != nil {
		return out, err
	}
	ordinary := cap.IncrementalLearning.OrdinaryEconomics
	if raw.SchemaVersion == earlySchema {
		ordinary = raw.Qualification.OrdinaryEconomics
	}
	if err := validateOrdinary(ordinary, c); err != nil {
		return out, err
	}
	if raw.Key != expected.Key || raw.Repository.ID != expected.RepositoryID || raw.Qualification.TargetRevision != expected.TargetRevision || raw.Qualification.ParentRevision != expected.PublicRevision {
		return out, errors.New("subject identity drifted")
	}
	if cap.BuildOpt != raw.BuildOpt || cap.ProductFailure || cap.MeasurementOnlyWorkflowRuns != 0 {
		return out, errors.New("qualification capture is invalid")
	}
	out.Key, out.RepositoryID, out.TargetRevision = raw.Key, raw.Repository.ID, raw.Qualification.TargetRevision
	out.BuildOptRevision, out.ExecutableSHA256 = raw.BuildOpt.Revision, raw.BuildOpt.ExecutableSHA256
	out.RawResultSHA256, out.QualificationCaptureSHA256 = digest(resultRaw), digest(captureRaw)
	out.Qualification.RequestedBuilds = ordinary.RequestedBuilds
	out.Qualification.MeasurementOnlyBuilds = ordinary.MeasurementOnlyBuilds
	out.Qualification.HistoricalCompatibleMatches = ordinary.HistoricalCompatibleMatches
	out.Qualification.OrdinaryDecision, out.Qualification.OrdinaryReason = ordinary.Decision, ordinary.Reason
	out.Qualification.ObservedPairs, out.Qualification.MeanSavedMS = ordinary.ObservedPairs, ordinary.MeanSavedMS
	out.Qualification.ExactOutputBuilds, out.Qualification.LearningCostMS = ordinary.ExactOutputBuilds, ordinary.LearningCostMS
	out.Qualification.ProjectedPaybackMatches = ordinary.ProjectedPaybackMatches
	out.ProductFailures = raw.Acceptance.ProductFailures
	if raw.SchemaVersion == earlySchema {
		out.Qualification.Status = "ECONOMICALLY_RETAINED"
		out.Lifetime.QualificationAndPublicationCostMS = ordinary.LearningCostMS
		out.Lifetime.CumulativeNetSavedMS = -ordinary.LearningCostMS
		out.Lifetime.Conclusion = "EARLY_ECONOMIC_RETENTION"
		return out, nil
	}
	if raw.SchemaVersion != v2SubjectSchema {
		return out, errors.New("unsupported lifetime subject schema")
	}
	out.Qualification.RobustQualified = raw.Qualification.Calibration.Qualified
	out.Qualification.PositivePairs = raw.Qualification.Calibration.PositivePairs
	if raw.Qualification.Calibration.PairsMeasured != c.Learning.RobustPairs || len(raw.Qualification.Calibration.Interval95SavedMS) != 2 || !raw.Qualification.Calibration.FallbackSuccessful {
		return out, errors.New("robust qualification evidence is incomplete")
	}
	if raw.Qualification.Calibration.Qualified {
		if raw.Qualification.Calibration.PositivePairs < c.Learning.MinimumPositivePairs || raw.Qualification.Calibration.Interval95SavedMS[0] <= 0 || raw.Qualification.Calibration.CandidateP95MS > raw.Qualification.Calibration.ControlP95MS {
			return out, errors.New("qualified subject violates the robust gate")
		}
		out.Qualification.Status = "QUALIFIED"
	} else {
		out.Qualification.Status = "ROBUST_VALUE_NOT_PROVEN"
	}
	out.Lifetime.ObservedBuilds = len(raw.Observations)
	out.Lifetime.QualificationAndPublicationCostMS = raw.Economics.QualificationAndPublicationCostMS
	for _, observation := range raw.Observations {
		if !observation.ExactRequiredOutputs {
			return out, errors.New("descendant required outputs drifted")
		}
		out.Lifetime.ExactOutputBuilds++
		if eligibleObservation(observation.WrapperMatchesQualification, observation.Reason, observation.Selected) {
			out.Lifetime.EligibleDescendants++
		}
		if observation.Selected {
			out.Lifetime.SelectedReplays++
			out.Lifetime.SelectedReplaySavedMS += observation.SavedMS
		}
		if observation.NativeRetained {
			out.Lifetime.NativeRetainedBuilds++
			out.Lifetime.NativeRetentionWrapperCostMS += observation.WrapperOverheadMS
		}
	}
	out.Lifetime.CumulativeNetSavedMS = out.Lifetime.SelectedReplaySavedMS - out.Lifetime.QualificationAndPublicationCostMS - out.Lifetime.NativeRetentionWrapperCostMS
	switch {
	case out.Lifetime.CumulativeNetSavedMS > 0:
		out.Lifetime.Conclusion = "PAID_BACK_IN_OBSERVED_WINDOW"
	case out.Lifetime.SelectedReplays > 0:
		out.Lifetime.Conclusion = "NOT_PAID_BACK_IN_OBSERVED_WINDOW"
	default:
		out.Lifetime.Conclusion = "NO_SELECTED_REPLAY_IN_OBSERVED_WINDOW"
	}
	return out, nil
}

func eligibleObservation(wrapper bool, reason string, selected bool) bool {
	if !wrapper {
		return false
	}
	if selected {
		return true
	}
	switch reason {
	case "ORDINARY_OBSERVATIONS_PENDING",
		"QUALIFIED_PROFILE_SELECTED",
		"STRUCTURAL_PROFILE_REBOUND",
		"QUALIFIED_PROFILE_OUTPUTS_REFRESHED",
		"ECONOMIC_PREQUALIFICATION_REJECTED":
		return true
	default:
		return false
	}
}

func validateContract(c contract) error {
	wantRepositories := []string{
		"spring-projects/spring-framework",
		"open-telemetry/opentelemetry-java-instrumentation",
		"apache/kafka",
		"micronaut-projects/micronaut-core",
		"apache/groovy",
	}
	if c.SchemaVersion != contractSchema || c.WorkItem != "POC-LIFETIME-BREADTH-V3-001" || c.FrozenSubjects.RepositoryCount != 5 || !reflect.DeepEqual(c.FrozenSubjects.Repositories, wantRepositories) || c.Learning.HistoricalCompatibleMatchesRequired != 5 || c.Learning.MaximumProjectedPaybackMatches != 5 || c.Learning.MaximumRequestedBuildsPerQualification != 17 || !reflect.DeepEqual(c.Learning.EarlyRetentionBuildCounts, []int{1, 3}) || c.Learning.RobustPairs != 8 || c.Learning.MinimumPositivePairs != 6 || !c.Learning.PositiveIntervalRequired || !c.Learning.CandidateP95NonRegressive || !c.Learning.PositiveFirstPairOnlyContinuesLearning || !c.Learning.PositiveEconomicsDoesNotQualifyAlone || c.Acceptance.MinimumNetPositiveRepositoryFamilies != 3 || c.Acceptance.MinimumEligibleDescendantSelectionRatio != 0.5 || !c.Acceptance.AllFiveSubjectsObserved || !c.Acceptance.SameExecutableSHA256 || !c.Acceptance.ExactOutputsForEveryRequestedBuild || !c.Acceptance.ZeroProductAttributableFailures || c.Acceptance.TerminalPassDecision != "FUNCTIONAL_COVERAGE_PROVEN" || c.Acceptance.TerminalFailDecision != "FUNCTIONAL_COVERAGE_NOT_PROVEN" || !c.Boundaries.ProofOfConcept || c.Boundaries.ProductionAuthorized || c.Boundaries.SoakRequired || c.Boundaries.DesignPartnerRequired || c.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("lifetime breadth V3 contract is invalid")
	}
	return nil
}

func validateOrdinary(e ordinaryEconomics, c contract) error {
	if e.SchemaVersion != "buildopt.poc/ordinary-learning-economics/v1" || e.RequestedBuilds < 1 || e.RequestedBuilds > c.Learning.MaximumRequestedBuildsPerQualification || e.MeasurementOnlyBuilds != 0 || e.CompatibleBuilds != e.RequestedBuilds || e.SuccessfulBuilds != e.RequestedBuilds || e.ExactOutputBuilds != e.RequestedBuilds || e.StructurallyPortableBuilds != e.RequestedBuilds || e.LearningCostMS < 1 || e.MaximumPaybackMatches != c.Learning.MaximumProjectedPaybackMatches || e.ProductionAuthorized || e.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("ordinary learning evidence is invalid")
	}
	if e.Decision == "NATIVE_RETAINED" {
		if e.RequestedBuilds != 1 && e.RequestedBuilds != 3 {
			return errors.New("economic retention occurred outside the bounded stop points")
		}
	} else if e.Decision != "QUALIFICATION_READY" || e.RequestedBuilds != 17 || e.ObservedPairs != 8 || !e.CalibrationAuthorized {
		return errors.New("ordinary learning did not reach the robust gate correctly")
	}
	return nil
}

func strictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("multiple JSON values")
	}
	return nil
}

func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
