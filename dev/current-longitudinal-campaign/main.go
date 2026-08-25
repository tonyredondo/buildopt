// Command current-longitudinal-campaign validates AF-014C raw observations and
// deterministically derives the current longitudinal report. It never runs a
// build or mutates candidate authority.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

const (
	rawSchema    = "buildopt.poc/current-longitudinal-raw/v1"
	reportSchema = "buildopt.poc/current-longitudinal-report/v1"
	contractSHA  = "24c3b2de49452955143f790e3335a27d75d5cdd798cdf9767376b6aff3dbc309"
)

type rawEvidence struct {
	SchemaVersion   string          `json:"schemaVersion"`
	WorkItem        string          `json:"workItem"`
	CapturedAt      string          `json:"capturedAt"`
	EvaluatedSHA    string          `json:"evaluatedRevision"`
	ExecutableSHA   string          `json:"executableSha256"`
	CohortSHA       string          `json:"cohortSha256"`
	ContractSHA     string          `json:"contractSha256"`
	Repositories    []repositoryRaw `json:"repositories"`
	ProductFailures int             `json:"productFailures"`
	Boundaries      boundaries      `json:"boundaries"`
}

type repositoryRaw struct {
	Key          string        `json:"key"`
	RepositoryID string        `json:"repositoryId"`
	Workflow     []string      `json:"workflow"`
	Outputs      []string      `json:"requiredOutputs"`
	Observations []observation `json:"observations"`
	Exclusions   []exclusion   `json:"exclusions"`
}

type observation struct {
	CohortAttempt int               `json:"cohortAttempt"`
	Sequence      int               `json:"sequence"`
	Source        string            `json:"source"`
	FrozenOrdinal int               `json:"frozenOrdinal"`
	Revision      string            `json:"revision"`
	Parent        string            `json:"parentRevision"`
	ChangeShape   string            `json:"changeShape"`
	Order         string            `json:"order"`
	Control       arm               `json:"control"`
	Candidate     arm               `json:"candidate"`
	ExactOutputs  bool              `json:"exactOutputs"`
	SignedDeltaNS int64             `json:"signedDeltaNs"`
	CumulativeNS  int64             `json:"cumulativeNetNs"`
	Decision      candidateDecision `json:"candidateDecision"`
}

type arm struct {
	WallNS                 int64  `json:"wallNs"`
	ExitCode               int    `json:"exitCode"`
	OutputSHA              string `json:"outputSha256"`
	OutputCount            int    `json:"outputCount"`
	CheckoutSHA            string `json:"checkoutSha256"`
	GradleHomeSHA          string `json:"gradleHomeSha256"`
	BuildCacheSHA          string `json:"buildCacheSha256"`
	DaemonSHA              string `json:"daemonRegistrySha256"`
	BuildOptStateSHA       string `json:"buildoptStateSha256,omitempty"`
	BuildOptStateBeforeSHA string `json:"buildoptStateBeforeSha256,omitempty"`
}

type candidateDecision struct {
	Outcome                     string   `json:"outcome"`
	Reason                      string   `json:"reason"`
	Phase                       string   `json:"phase"`
	ExecutionMode               string   `json:"executionMode"`
	SelectionStatus             string   `json:"selectionStatus"`
	SelectionSelected           bool     `json:"selectionSelected"`
	RuntimeSurface              string   `json:"runtimeSurface"`
	ActivatedFragments          []string `json:"activatedFragments"`
	SuspendedFragments          []string `json:"suspendedFragments"`
	MeasurementAuthorityUpdated bool     `json:"measurementAuthorityUpdated"`
	Timing                      timing   `json:"timing"`
}

type timing struct {
	PreExecutionNS    int64 `json:"preExecutionNs"`
	GradleExecutionNS int64 `json:"gradleExecutionNs"`
	FinalizationNS    int64 `json:"finalizationNs"`
	UnattributedNS    int64 `json:"unattributedNs"`
	TotalNS           int64 `json:"totalNs"`
	Diagnostics       struct {
		GradleSetupNS        int64 `json:"gradleSetupNs"`
		MatchingNS           int64 `json:"matchingNs"`
		LocalStateNS         int64 `json:"localStateNs"`
		CentralStateNS       int64 `json:"centralStateNs"`
		MaterializationNS    int64 `json:"materializationNs"`
		OutputVerificationNS int64 `json:"outputVerificationNs"`
		DiscoveryLearningNS  int64 `json:"discoveryLearningNs"`
	} `json:"diagnostics"`
}

type exclusion struct {
	CohortAttempt int    `json:"cohortAttempt"`
	Source        string `json:"source"`
	FrozenOrdinal int    `json:"frozenOrdinal"`
	Revision      string `json:"revision"`
	Reason        string `json:"reason"`
	Detail        string `json:"detail"`
}

type boundaries struct {
	ProofOfConcept        bool   `json:"proofOfConcept"`
	ProductionAuthorized  bool   `json:"productionAuthorized"`
	SoakRequired          bool   `json:"soakRequired"`
	DesignPartnerRequired bool   `json:"designPartnerRequired"`
	TestOptimization      string `json:"testOptimization"`
}

type frozenCohort struct {
	Repositories []frozenRepository `json:"repositories"`
}

type frozenRepository struct {
	Policy struct {
		Key string `json:"key"`
	} `json:"policy"`
	Primary  []frozenCommit `json:"primary"`
	Reserves []frozenCommit `json:"reserves"`
}

type frozenCommit struct {
	Ordinal        int    `json:"ordinal"`
	Revision       string `json:"revision"`
	ParentRevision string `json:"parentRevision"`
}

type report struct {
	SchemaVersion string              `json:"schemaVersion"`
	WorkItem      string              `json:"workItem"`
	CapturedAt    string              `json:"capturedAt"`
	EvaluatedSHA  string              `json:"evaluatedRevision"`
	RawSHA        string              `json:"rawSha256"`
	Summary       reportSummary       `json:"summary"`
	Repositories  []repositorySummary `json:"repositories"`
	ByChangeShape []shapeSummary      `json:"byChangeShape"`
	Boundaries    boundaries          `json:"boundaries"`
}

type reportSummary struct {
	ComparablePairs     int   `json:"comparablePairs"`
	PositivePairs       int   `json:"positivePairs"`
	NegativePairs       int   `json:"negativePairs"`
	SelectedProfiles    int   `json:"selectedProfiles"`
	NativeRetentions    int   `json:"nativeRetentions"`
	FragmentActivations int   `json:"fragmentActivations"`
	Exclusions          int   `json:"exclusions"`
	ProductFailures     int   `json:"productFailures"`
	CumulativeNetNS     int64 `json:"cumulativeNetNs"`
}

type repositorySummary struct {
	Key                 string `json:"key"`
	RepositoryID        string `json:"repositoryId"`
	Outcome             string `json:"outcome"`
	ComparablePairs     int    `json:"comparablePairs"`
	PositivePairs       int    `json:"positivePairs"`
	NegativePairs       int    `json:"negativePairs"`
	SelectedProfiles    int    `json:"selectedProfiles"`
	NativeRetentions    int    `json:"nativeRetentions"`
	FragmentActivations int    `json:"fragmentActivations"`
	Exclusions          int    `json:"exclusions"`
	CumulativeNetNS     int64  `json:"cumulativeNetNs"`
	MeanSignedDeltaNS   int64  `json:"meanSignedDeltaNs"`
	MedianSignedDeltaNS int64  `json:"medianSignedDeltaNs"`
	P95RegressionNS     int64  `json:"p95RegressionNs"`
	WorstRegressionNS   int64  `json:"worstRegressionNs"`
}

type shapeSummary struct {
	ChangeShape     string `json:"changeShape"`
	ComparablePairs int    `json:"comparablePairs"`
	PositivePairs   int    `json:"positivePairs"`
	CumulativeNetNS int64  `json:"cumulativeNetNs"`
}

func main() {
	if (len(os.Args) != 3 || os.Args[1] != "--aggregate") &&
		(len(os.Args) != 5 || os.Args[1] != "--validate") {
		fmt.Fprintln(os.Stderr, "usage: current-longitudinal-campaign --aggregate RAW_JSON | --validate COHORT_JSON RAW_JSON REPORT_JSON")
		os.Exit(64)
	}
	rawIndex := 2
	if os.Args[1] == "--validate" {
		rawIndex = 3
	}
	rawBytes, err := os.ReadFile(os.Args[rawIndex])
	if err != nil {
		fatal(err)
	}
	raw, err := decodeRaw(rawBytes)
	if err != nil {
		fatal(err)
	}
	derived, err := aggregate(rawBytes, raw)
	if err != nil {
		fatal(err)
	}
	if os.Args[1] == "--aggregate" {
		encoded, marshalErr := json.MarshalIndent(derived, "", "  ")
		if marshalErr != nil {
			fatal(marshalErr)
		}
		fmt.Printf("%s\n", encoded)
		return
	}
	cohortBytes, err := os.ReadFile(os.Args[2])
	if err != nil {
		fatal(err)
	}
	if err := validateFrozenCohort(raw, cohortBytes); err != nil {
		fatal(err)
	}
	reportBytes, err := os.ReadFile(os.Args[4])
	if err != nil {
		fatal(err)
	}
	var actual report
	if err := decodeStrict(reportBytes, &actual); err != nil {
		fatal(err)
	}
	wanted, _ := json.Marshal(derived)
	got, _ := json.Marshal(actual)
	if !bytes.Equal(wanted, got) {
		fatal(errors.New("current longitudinal report is not the deterministic raw-evidence aggregate"))
	}
	fmt.Println("current longitudinal campaign: valid")
}

func decodeRaw(raw []byte) (rawEvidence, error) {
	var result rawEvidence
	if err := decodeStrict(raw, &result); err != nil {
		return result, err
	}
	if result.SchemaVersion != rawSchema || result.WorkItem != "AF-014C" ||
		!validSHA(result.EvaluatedSHA) || !validSHA(result.ExecutableSHA) ||
		result.CohortSHA != contractSHA || !validSHA(result.ContractSHA) || len(result.Repositories) != 5 ||
		result.Boundaries != (boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"}) {
		return result, errors.New("current longitudinal raw evidence identity is invalid")
	}
	seen := map[string]bool{}
	computedFailures := 0
	for _, repository := range result.Repositories {
		if repository.Key == "" || repository.RepositoryID == "" || seen[repository.Key] || len(repository.Workflow) == 0 || len(repository.Outputs) == 0 {
			return result, errors.New("current longitudinal repository identity is invalid")
		}
		seen[repository.Key] = true
		if len(repository.Observations) > 20 || len(repository.Observations)+len(repository.Exclusions) > 30 {
			return result, errors.New("current longitudinal cohort cardinality is invalid")
		}
		var cumulative int64
		previousState := ""
		for index, observation := range repository.Observations {
			if err := validateObservation(observation, index+1, cumulative); err != nil {
				return result, fmt.Errorf("%s observation %d: %w", repository.Key, index+1, err)
			}
			if index > 0 && observation.Candidate.BuildOptStateBeforeSHA != previousState {
				return result, fmt.Errorf("%s observation %d does not consume exactly N-1 state", repository.Key, index+1)
			}
			previousState = observation.Candidate.BuildOptStateSHA
			cumulative = observation.CumulativeNS
		}
		for _, excluded := range repository.Exclusions {
			if excluded.CohortAttempt <= 0 || !oneOf(excluded.Source, "PRIMARY", "RESERVE") || !oneOf(excluded.Reason,
				"NATIVE_BUILD_FAILURE", "DEPENDENCY_UNAVAILABLE_AFTER_PREPARATION",
				"RUNNER_ENVIRONMENT_FAILURE", "NATIVE_OUTPUT_NONDETERMINISM") || !validSHA(excluded.Revision) {
				return result, errors.New("current longitudinal exclusion is invalid")
			}
		}
		for _, observation := range repository.Observations {
			if observation.Candidate.ExitCode != 0 {
				computedFailures++
			}
		}
	}
	if computedFailures != result.ProductFailures {
		return result, errors.New("current longitudinal product-failure count is inconsistent")
	}
	return result, nil
}

func validateObservation(value observation, sequence int, previous int64) error {
	wantOrder := "CONTROL_FIRST"
	if sequence%2 == 0 {
		wantOrder = "CANDIDATE_FIRST"
	}
	if value.CohortAttempt <= 0 || value.Sequence != sequence || value.Order != wantOrder || !oneOf(value.Source, "PRIMARY", "RESERVE") ||
		!validSHA(value.Revision) || !validSHA(value.Parent) || value.ChangeShape == "" ||
		value.Control.WallNS <= 0 || value.Candidate.WallNS <= 0 || value.Control.ExitCode != 0 ||
		!validSHA(value.Control.OutputSHA) || !validSHA(value.Candidate.OutputSHA) ||
		value.Control.OutputCount <= 0 || value.Control.OutputCount != value.Candidate.OutputCount ||
		!value.ExactOutputs || value.Control.OutputSHA != value.Candidate.OutputSHA ||
		value.SignedDeltaNS != value.Control.WallNS-value.Candidate.WallNS ||
		value.CumulativeNS != previous+value.SignedDeltaNS {
		return errors.New("pair identity, output or signed economics is invalid")
	}
	for _, digest := range []string{value.Control.CheckoutSHA, value.Control.GradleHomeSHA,
		value.Control.BuildCacheSHA, value.Control.DaemonSHA, value.Candidate.CheckoutSHA,
		value.Candidate.GradleHomeSHA, value.Candidate.BuildCacheSHA, value.Candidate.DaemonSHA,
		value.Candidate.BuildOptStateSHA} {
		if !validSHA(digest) {
			return errors.New("pair state fingerprint is invalid")
		}
	}
	if sequence > 1 && !validSHA(value.Candidate.BuildOptStateBeforeSHA) {
		return errors.New("candidate prior-state fingerprint is missing")
	}
	decision := value.Decision
	if decision.Outcome == "" || decision.Reason == "" || decision.Phase == "" || decision.ExecutionMode == "" ||
		decision.SelectionStatus == "" || decision.MeasurementAuthorityUpdated ||
		decision.Timing.TotalNS <= 0 || decision.Timing.TotalNS != decision.Timing.PreExecutionNS+
		decision.Timing.GradleExecutionNS+decision.Timing.FinalizationNS+decision.Timing.UnattributedNS {
		return errors.New("candidate decision or phase timing is invalid")
	}
	if len(decision.ActivatedFragments) != 0 || len(decision.SuspendedFragments) != 0 || decision.RuntimeSurface != "NO_FRAGMENT_RUNTIME" {
		return errors.New("whole-profile runtime was relabelled as adaptive fragments")
	}
	return nil
}

func validateFrozenCohort(raw rawEvidence, document []byte) error {
	digest := sha256.Sum256(document)
	if hex.EncodeToString(digest[:]) != raw.CohortSHA {
		return errors.New("current longitudinal cohort digest is inconsistent")
	}
	var cohort frozenCohort
	if err := json.Unmarshal(document, &cohort); err != nil || len(cohort.Repositories) != len(raw.Repositories) {
		return errors.New("current longitudinal frozen cohort is invalid")
	}
	byKey := map[string]frozenRepository{}
	for _, repository := range cohort.Repositories {
		byKey[repository.Policy.Key] = repository
	}
	for _, repository := range raw.Repositories {
		frozen, exists := byKey[repository.Key]
		if !exists || len(frozen.Primary) != 20 || len(frozen.Reserves) < 10 {
			return errors.New("current longitudinal repository is absent from the frozen cohort")
		}
		type attempted struct {
			attempt  int
			source   string
			ordinal  int
			revision string
			parent   string
		}
		attempts := make([]attempted, 0, len(repository.Observations)+len(repository.Exclusions))
		for _, value := range repository.Observations {
			attempts = append(attempts, attempted{value.CohortAttempt, value.Source, value.FrozenOrdinal, value.Revision, value.Parent})
		}
		for _, value := range repository.Exclusions {
			attempts = append(attempts, attempted{value.CohortAttempt, value.Source, value.FrozenOrdinal, value.Revision, ""})
		}
		sort.Slice(attempts, func(i, j int) bool { return attempts[i].attempt < attempts[j].attempt })
		for index, value := range attempts {
			if value.attempt != index+1 {
				return errors.New("current longitudinal cohort attempts are not contiguous")
			}
			var expected frozenCommit
			expectedSource := "PRIMARY"
			if index < 20 {
				expected = frozen.Primary[index]
			} else {
				expectedSource = "RESERVE"
				expected = frozen.Reserves[index-20]
			}
			if value.source != expectedSource || value.ordinal != expected.Ordinal || value.revision != expected.Revision ||
				(value.parent != "" && value.parent != expected.ParentRevision) {
				return errors.New("current longitudinal cohort attempt drifted from the frozen order")
			}
		}
		if len(repository.Observations) < 20 && len(attempts) != 30 {
			return errors.New("insufficient cohort did not exhaust its ordered reserves")
		}
	}
	return nil
}

func aggregate(rawBytes []byte, raw rawEvidence) (report, error) {
	digest := sha256.Sum256(rawBytes)
	result := report{SchemaVersion: reportSchema, WorkItem: "AF-014C", CapturedAt: raw.CapturedAt,
		EvaluatedSHA: raw.EvaluatedSHA, RawSHA: hex.EncodeToString(digest[:]), Boundaries: raw.Boundaries}
	shape := map[string]*shapeSummary{}
	for _, repository := range raw.Repositories {
		deltas := make([]int64, 0, len(repository.Observations))
		row := repositorySummary{Key: repository.Key, RepositoryID: repository.RepositoryID,
			ComparablePairs: len(repository.Observations), Exclusions: len(repository.Exclusions)}
		for _, observation := range repository.Observations {
			delta := observation.SignedDeltaNS
			deltas = append(deltas, delta)
			row.CumulativeNetNS += delta
			if delta > 0 {
				row.PositivePairs++
			} else if delta < 0 {
				row.NegativePairs++
			}
			if observation.Decision.SelectionSelected {
				row.SelectedProfiles++
			} else {
				row.NativeRetentions++
			}
			row.FragmentActivations += len(observation.Decision.ActivatedFragments)
			regression := -delta
			if regression > row.WorstRegressionNS {
				row.WorstRegressionNS = regression
			}
			current := shape[observation.ChangeShape]
			if current == nil {
				current = &shapeSummary{ChangeShape: observation.ChangeShape}
				shape[observation.ChangeShape] = current
			}
			current.ComparablePairs++
			current.CumulativeNetNS += delta
			if delta > 0 {
				current.PositivePairs++
			}
		}
		if len(deltas) > 0 {
			row.MeanSignedDeltaNS = row.CumulativeNetNS / int64(len(deltas))
			row.MedianSignedDeltaNS = percentile(deltas, 50)
			regressions := make([]int64, len(deltas))
			for i, delta := range deltas {
				regressions[i] = -delta
			}
			row.P95RegressionNS = percentile(regressions, 95)
		}
		row.Outcome = classify(row)
		result.Repositories = append(result.Repositories, row)
		result.Summary.ComparablePairs += row.ComparablePairs
		result.Summary.PositivePairs += row.PositivePairs
		result.Summary.NegativePairs += row.NegativePairs
		result.Summary.SelectedProfiles += row.SelectedProfiles
		result.Summary.NativeRetentions += row.NativeRetentions
		result.Summary.FragmentActivations += row.FragmentActivations
		result.Summary.Exclusions += row.Exclusions
		result.Summary.CumulativeNetNS += row.CumulativeNetNS
	}
	result.Summary.ProductFailures = raw.ProductFailures
	sort.Slice(result.Repositories, func(i, j int) bool { return result.Repositories[i].Key < result.Repositories[j].Key })
	for _, current := range shape {
		result.ByChangeShape = append(result.ByChangeShape, *current)
	}
	sort.Slice(result.ByChangeShape, func(i, j int) bool { return result.ByChangeShape[i].ChangeShape < result.ByChangeShape[j].ChangeShape })
	return result, nil
}

func classify(row repositorySummary) string {
	if row.ComparablePairs < 15 {
		return "INSUFFICIENT_COHORT"
	}
	directional := (row.ComparablePairs*600 + 999) / 1000
	if row.CumulativeNetNS > 0 && row.PositivePairs >= directional {
		return "NET_POSITIVE"
	}
	if row.CumulativeNetNS < 0 && row.NegativePairs >= directional {
		return "NET_NEGATIVE"
	}
	return "INCONCLUSIVE"
}

func percentile(values []int64, percent int) int64 {
	copyValues := append([]int64{}, values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	index := (percent*len(copyValues)+99)/100 - 1
	if index < 0 {
		index = 0
	}
	return copyValues[index]
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

func validSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
