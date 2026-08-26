package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

const (
	terminalDecisionSchema         = "buildopt.poc/adaptive-fragment-terminal-decision/v1"
	terminalDecisionContractSchema = "buildopt.poc/adaptive-fragment-terminal-decision-contract/v1"
)

type terminalDecisionContract struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	SourceSchemas struct {
		Raw         string `json:"raw"`
		Report      string `json:"report"`
		Attribution string `json:"attribution"`
		Campaign    string `json:"campaignContract"`
	} `json:"sourceSchemas"`
	Criteria   []string `json:"criteria"`
	Thresholds struct {
		FamilyCount                        int   `json:"familyCount"`
		MinimumActivationCoveragePermille  int   `json:"minimumActivationCoveragePermille"`
		MinimumPositiveFamilies            int   `json:"minimumPositiveFamilies"`
		MinimumPositiveLowerBoundFamilies  int   `json:"minimumPositiveLowerBoundFamilies"`
		MinimumComparablePairsPerFamily    int   `json:"minimumComparablePairsPerFamily"`
		MaximumPaybackOrdinal              int   `json:"maximumPaybackOrdinal"`
		MinimumNativeRetentionObservations int   `json:"minimumNativeRetentionObservations"`
		MaximumNativeRetentionMedianNS     int64 `json:"maximumNativeRetentionMedianNs"`
		MaximumNativeRetentionP95NS        int64 `json:"maximumNativeRetentionP95Ns"`
		MinimumPortfolioNetNS              int64 `json:"minimumPortfolioNetNs"`
		MinimumBoundedActivations          int   `json:"minimumBoundedActivations"`
		MinimumBoundedSavingsNS            int64 `json:"minimumBoundedSavingsNs"`
		MinimumBoundedPositiveFamilies     int   `json:"minimumBoundedPositiveFamilies"`
	} `json:"thresholds"`
	Confidence struct {
		Method        string `json:"method"`
		Samples       int    `json:"samples"`
		LowerPermille int    `json:"lowerPermille"`
		Seed          uint64 `json:"seed"`
	} `json:"confidence"`
	Implementation struct {
		RepositoryNameRulesAllowed bool `json:"repositoryNameRulesAllowed"`
		ManualProfilesAllowed      bool `json:"manualProfilesAllowed"`
	} `json:"implementation"`
	Outcomes          []string   `json:"outcomes"`
	HistoricalContext string     `json:"historicalContext"`
	Boundaries        boundaries `json:"boundaries"`
}

type campaignContract struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	Arms          struct {
		Control                  string `json:"control"`
		Candidate                string `json:"candidate"`
		Order                    string `json:"order"`
		SeparateCheckouts        bool   `json:"separateCheckouts"`
		SeparateGradleHomes      bool   `json:"separateGradleHomes"`
		SeparateBuildCaches      bool   `json:"separateBuildCaches"`
		SeparateDaemonRegistries bool   `json:"separateDaemonRegistries"`
		CandidateStateOnly       bool   `json:"candidateStateOnly"`
	} `json:"arms"`
	Chronology struct {
		PrimaryPerRepository            int    `json:"primaryPerRepository"`
		MinimumComparablePerRepository  int    `json:"minimumComparablePerRepository"`
		ReservePolicy                   string `json:"reservePolicy"`
		NoLookahead                     bool   `json:"noLookahead"`
		MeasurementOnlyAuthorityUpdates bool   `json:"measurementOnlyAuthorityUpdates"`
	} `json:"chronology"`
	Boundaries boundaries `json:"boundaries"`
}

type terminalDecisionDocument struct {
	SchemaVersion       string                    `json:"schemaVersion"`
	WorkItem            string                    `json:"workItem"`
	CapturedAt          string                    `json:"capturedAt"`
	EvaluatedSHA        string                    `json:"evaluatedRevision"`
	RawSHA              string                    `json:"rawSha256"`
	ReportSHA           string                    `json:"reportSha256"`
	AttributionSHA      string                    `json:"attributionSha256"`
	CampaignContractSHA string                    `json:"campaignContractSha256"`
	DecisionContractSHA string                    `json:"decisionContractSha256"`
	Outcome             string                    `json:"outcome"`
	Summary             terminalDecisionSummary   `json:"summary"`
	Criteria            []terminalCriterion       `json:"criteria"`
	Repositories        []terminalRepository      `json:"repositories"`
	Specialization      terminalSpecialization    `json:"specialization"`
	HistoricalContext   terminalHistoricalContext `json:"historicalContext"`
	Boundaries          boundaries                `json:"boundaries"`
}

type terminalDecisionSummary struct {
	CriteriaPassed              int   `json:"criteriaPassed"`
	CriteriaFailed              int   `json:"criteriaFailed"`
	ComparablePairs             int   `json:"comparablePairs"`
	ExactOutputPairs            int   `json:"exactOutputPairs"`
	ProductFailures             int   `json:"productFailures"`
	EligibleDescendantBuilds    int   `json:"eligibleDescendantBuilds"`
	ActivatedEligibleBuilds     int   `json:"activatedEligibleBuilds"`
	ActivationCoveragePermille  int   `json:"activationCoveragePermille"`
	PositiveFamilies            int   `json:"positiveFamilies"`
	PositiveLowerBoundFamilies  int   `json:"positiveLowerBoundFamilies"`
	FamiliesRepaid              int   `json:"familiesRepaid"`
	CumulativeSignedDeltaNS     int64 `json:"cumulativeSignedDeltaNs"`
	AttributableMechanismSaveNS int64 `json:"attributableMechanismSavingsNs"`
	NativeRetentionMedianNS     int64 `json:"nativeRetentionMedianNs"`
	NativeRetentionP95NS        int64 `json:"nativeRetentionP95Ns"`
	WorstRegressionNS           int64 `json:"worstRegressionNs"`
}

type terminalCriterion struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Requirement string `json:"requirement"`
	Observed    string `json:"observed"`
	Evidence    string `json:"evidence"`
}

type terminalRepository struct {
	Key                     string               `json:"key"`
	RepositoryID            string               `json:"repositoryId"`
	ComparablePairs         int                  `json:"comparablePairs"`
	Outcome                 string               `json:"outcome"`
	CumulativeSignedDeltaNS int64                `json:"cumulativeSignedDeltaNs"`
	LowerConfidenceBoundNS  int64                `json:"lowerConfidenceBoundNs"`
	PositivePairs           int                  `json:"positivePairs"`
	NegativePairs           int                  `json:"negativePairs"`
	EligibleBuilds          int                  `json:"eligibleBuilds"`
	FragmentActivations     int                  `json:"fragmentActivations"`
	PaybackOrdinal          *int                 `json:"paybackOrdinal"`
	WorstRegressionNS       int64                `json:"worstRegressionNs"`
	CumulativeCheckpoints   []terminalCheckpoint `json:"cumulativeCheckpoints"`
}

type terminalCheckpoint struct {
	Ordinal         int   `json:"ordinal"`
	CumulativeNetNS int64 `json:"cumulativeNetNs"`
}

type terminalSpecialization struct {
	CorrectnessPassed  bool  `json:"correctnessPassed"`
	BoundedValueExists bool  `json:"boundedValueExists"`
	Activations        int   `json:"activations"`
	AttributableSaveNS int64 `json:"attributableSavingsNs"`
	PositiveFamilies   int   `json:"positiveFamilies"`
}

type terminalHistoricalContext struct {
	Evidence string `json:"evidence"`
	Role     string `json:"role"`
	Input    bool   `json:"decisionInput"`
}

func runTerminalDecision(rawPath, reportPath, attributionPath, campaignPath, contractPath, decisionPath string, write bool) {
	rawBytes, err := os.ReadFile(rawPath)
	if err != nil {
		fatal(err)
	}
	raw, err := decodeRaw(rawBytes)
	if err != nil {
		fatal(err)
	}
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		fatal(err)
	}
	var reportValue report
	if err := decodeStrict(reportBytes, &reportValue); err != nil {
		fatal(err)
	}
	attributionBytes, err := os.ReadFile(attributionPath)
	if err != nil {
		fatal(err)
	}
	var attribution attributionDocument
	if err := decodeStrict(attributionBytes, &attribution); err != nil {
		fatal(err)
	}
	campaignBytes, err := os.ReadFile(campaignPath)
	if err != nil {
		fatal(err)
	}
	var campaign campaignContract
	if err := json.Unmarshal(campaignBytes, &campaign); err != nil {
		fatal(err)
	}
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		fatal(err)
	}
	var contract terminalDecisionContract
	if err := decodeStrict(contractBytes, &contract); err != nil {
		fatal(err)
	}
	if err := validateTerminalContract(contract); err != nil {
		fatal(err)
	}
	document, err := deriveTerminalDecision(rawBytes, raw, reportBytes, reportValue, attributionBytes, attribution, campaignBytes, campaign, contractBytes, contract)
	if err != nil {
		fatal(err)
	}
	if decisionPath == "" || write {
		encoded, marshalErr := json.MarshalIndent(document, "", "  ")
		if marshalErr != nil {
			fatal(marshalErr)
		}
		if write {
			if err := os.WriteFile(decisionPath, append(encoded, '\n'), 0o644); err != nil {
				fatal(err)
			}
			return
		}
		fmt.Printf("%s\n", encoded)
		return
	}
	actualBytes, err := os.ReadFile(decisionPath)
	if err != nil {
		fatal(err)
	}
	var actual terminalDecisionDocument
	if err := decodeStrict(actualBytes, &actual); err != nil {
		fatal(err)
	}
	wanted, _ := json.Marshal(document)
	got, _ := json.Marshal(actual)
	if string(wanted) != string(got) {
		fatal(errors.New("terminal decision is not the deterministic current-evidence scorecard"))
	}
	fmt.Println("adaptive-fragment terminal decision: valid")
}

func validateTerminalContract(contract terminalDecisionContract) error {
	required := []string{"CORRECTNESS", "GENERIC_IMPLEMENTATION", "NATIVE_SUBSTRATE", "FRAGMENT_ACTIVATION_COVERAGE", "REPOSITORY_FAMILY_BREADTH", "LONGITUDINAL_CONFIDENCE", "COHORT_INTEGRITY", "CURRENT_IMPLEMENTATION", "PORTFOLIO_VALUE", "TIME_TO_VALUE", "NATIVE_RETENTION_COST", "ORDINARY_LEARNING", "BOUNDED_REGRET", "ATTRIBUTION", "SAFE_EVOLUTION"}
	thresholds := contract.Thresholds
	if contract.SchemaVersion != terminalDecisionContractSchema || contract.WorkItem != "AF-015" ||
		contract.SourceSchemas.Raw != rawSchema || contract.SourceSchemas.Report != reportSchema ||
		contract.SourceSchemas.Attribution != attributionSchema || contract.SourceSchemas.Campaign != "buildopt.specs/current-longitudinal-campaign/v1" ||
		len(contract.Criteria) != len(required) || thresholds.FamilyCount != 5 ||
		thresholds.MinimumActivationCoveragePermille != 500 || thresholds.MinimumPositiveFamilies != 3 ||
		thresholds.MinimumPositiveLowerBoundFamilies != 3 || thresholds.MinimumComparablePairsPerFamily != 15 ||
		thresholds.MaximumPaybackOrdinal != 20 || thresholds.MinimumNativeRetentionObservations != 30 ||
		thresholds.MaximumNativeRetentionMedianNS != 500000000 || thresholds.MaximumNativeRetentionP95NS != 1000000000 ||
		thresholds.MinimumPortfolioNetNS != 1 || thresholds.MinimumBoundedActivations != 1 ||
		thresholds.MinimumBoundedSavingsNS != 1 || thresholds.MinimumBoundedPositiveFamilies != 1 ||
		contract.Confidence.Method != "DETERMINISTIC_PAIRED_BOOTSTRAP" || contract.Confidence.Samples != 10000 ||
		contract.Confidence.LowerPermille != 50 || contract.Confidence.Seed != 20260826 ||
		contract.Implementation.RepositoryNameRulesAllowed || contract.Implementation.ManualProfilesAllowed ||
		len(contract.Outcomes) != 3 || !oneOf("CONTINUE_ADAPTIVE_FRAGMENT_POC", contract.Outcomes...) ||
		!oneOf("SPECIALIZE_BOUNDED_FRAGMENT_CLASSES", contract.Outcomes...) || !oneOf("STOP_ADAPTIVE_FRAGMENT_POC", contract.Outcomes...) ||
		contract.HistoricalContext != "AF-013_CONTEXT_ONLY" ||
		contract.Boundaries != (boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"}) {
		return errors.New("adaptive-fragment terminal decision contract is invalid")
	}
	for index, id := range required {
		if contract.Criteria[index] != id {
			return errors.New("terminal criteria order or membership changed")
		}
	}
	return nil
}

func deriveTerminalDecision(rawBytes []byte, raw rawEvidence, reportBytes []byte, reportValue report, attributionBytes []byte, attribution attributionDocument, campaignBytes []byte, campaign campaignContract, contractBytes []byte, contract terminalDecisionContract) (terminalDecisionDocument, error) {
	rawDigest := digest(rawBytes)
	reportDigest := digest(reportBytes)
	attributionDigest := digest(attributionBytes)
	campaignDigest := digest(campaignBytes)
	contractDigest := digest(contractBytes)
	if reportValue.RawSHA != rawDigest || attribution.RawSHA != rawDigest || attribution.ReportSHA != reportDigest ||
		raw.ContractSHA != campaignDigest || reportValue.EvaluatedSHA != raw.EvaluatedSHA || attribution.EvaluatedSHA != raw.EvaluatedSHA {
		return terminalDecisionDocument{}, errors.New("terminal decision inputs are not digest-bound to one current campaign")
	}
	if campaign.SchemaVersion != contract.SourceSchemas.Campaign || campaign.WorkItem != "AF-014C" ||
		campaign.Arms.Control != "OPTIMIZED_NATIVE_GRADLE" || campaign.Arms.Candidate != "CURRENT_INSTALLED_BUILDOPT" ||
		campaign.Arms.Order != "ALTERNATING_CONTROL_FIRST" || !campaign.Arms.SeparateCheckouts || !campaign.Arms.SeparateGradleHomes ||
		!campaign.Arms.SeparateBuildCaches || !campaign.Arms.SeparateDaemonRegistries || !campaign.Arms.CandidateStateOnly ||
		campaign.Chronology.PrimaryPerRepository != 20 || campaign.Chronology.MinimumComparablePerRepository != 15 ||
		!campaign.Chronology.NoLookahead || campaign.Chronology.MeasurementOnlyAuthorityUpdates || campaign.Boundaries != contract.Boundaries {
		return terminalDecisionDocument{}, errors.New("current campaign contract does not preserve the frozen native substrate and chronology")
	}
	document := terminalDecisionDocument{SchemaVersion: terminalDecisionSchema, WorkItem: "AF-015", CapturedAt: raw.CapturedAt,
		EvaluatedSHA: raw.EvaluatedSHA, RawSHA: rawDigest, ReportSHA: reportDigest, AttributionSHA: attributionDigest,
		CampaignContractSHA: campaignDigest, DecisionContractSHA: contractDigest, Boundaries: contract.Boundaries,
		HistoricalContext: terminalHistoricalContext{Evidence: "AF-013", Role: "CONTEXT_ONLY", Input: false}}
	document.Summary.ComparablePairs = attribution.Summary.ComparablePairs
	document.Summary.ExactOutputPairs = attribution.Summary.ExactOutputPairs
	document.Summary.ProductFailures = attribution.Summary.ProductFailures
	document.Summary.CumulativeSignedDeltaNS = attribution.Summary.CumulativeSignedDeltaNS
	document.Summary.AttributableMechanismSaveNS = attribution.Summary.AttributableMechanismSavingsNS
	document.Summary.NativeRetentionMedianNS = attribution.Summary.NativeRetentionMedianNS
	document.Summary.NativeRetentionP95NS = attribution.Summary.NativeRetentionP95NS
	document.Summary.WorstRegressionNS = attribution.Summary.WorstRegressionNS
	checkpoints := map[int]bool{1: true, 5: true, 10: true, 15: true, 20: true}
	for repositoryIndex, repository := range raw.Repositories {
		row := terminalRepository{Key: repository.Key, RepositoryID: repository.RepositoryID, ComparablePairs: len(repository.Observations)}
		var deltas []int64
		var cumulatives []int64
		for _, observation := range repository.Observations {
			deltas = append(deltas, observation.SignedDeltaNS)
			cumulatives = append(cumulatives, observation.CumulativeNS)
			row.CumulativeSignedDeltaNS += observation.SignedDeltaNS
			if observation.SignedDeltaNS > 0 {
				row.PositivePairs++
			} else if observation.SignedDeltaNS < 0 {
				row.NegativePairs++
			}
			if observation.Decision.SelectionPerformed {
				row.EligibleBuilds++
				document.Summary.EligibleDescendantBuilds++
			}
			if len(observation.Decision.ActivatedFragments) > 0 {
				row.FragmentActivations += len(observation.Decision.ActivatedFragments)
				if observation.Decision.SelectionPerformed {
					document.Summary.ActivatedEligibleBuilds++
				}
			}
			if regression := -observation.SignedDeltaNS; regression > row.WorstRegressionNS {
				row.WorstRegressionNS = regression
			}
			if checkpoints[observation.Sequence] {
				row.CumulativeCheckpoints = append(row.CumulativeCheckpoints, terminalCheckpoint{Ordinal: observation.Sequence, CumulativeNetNS: observation.CumulativeNS})
			}
		}
		row.LowerConfidenceBoundNS = pairedBootstrapLowerBound(deltas, contract.Confidence.Samples, contract.Confidence.LowerPermille, contract.Confidence.Seed+uint64(repositoryIndex+1))
		for _, summary := range reportValue.Repositories {
			if summary.Key == row.Key {
				row.Outcome = summary.Outcome
			}
		}
		if row.CumulativeSignedDeltaNS > 0 {
			document.Summary.PositiveFamilies++
		}
		if row.LowerConfidenceBoundNS > 0 {
			document.Summary.PositiveLowerBoundFamilies++
		}
		if row.CumulativeSignedDeltaNS > 0 {
			for index, cumulative := range cumulatives {
				staysPositive := cumulative > 0
				for _, later := range cumulatives[index+1:] {
					staysPositive = staysPositive && later > 0
				}
				if staysPositive {
					ordinal := index + 1
					row.PaybackOrdinal = &ordinal
					break
				}
			}
			if row.PaybackOrdinal != nil && *row.PaybackOrdinal <= contract.Thresholds.MaximumPaybackOrdinal {
				document.Summary.FamiliesRepaid++
			}
		}
		document.Repositories = append(document.Repositories, row)
	}
	sort.Slice(document.Repositories, func(i, j int) bool { return document.Repositories[i].Key < document.Repositories[j].Key })
	if document.Summary.EligibleDescendantBuilds > 0 {
		document.Summary.ActivationCoveragePermille = document.Summary.ActivatedEligibleBuilds * 1000 / document.Summary.EligibleDescendantBuilds
	}
	correctness := document.Summary.ExactOutputPairs == document.Summary.ComparablePairs && document.Summary.ProductFailures == 0
	cohortIntegrity := len(document.Repositories) == contract.Thresholds.FamilyCount
	for _, row := range document.Repositories {
		cohortIntegrity = cohortIntegrity && row.ComparablePairs >= contract.Thresholds.MinimumComparablePairsPerFamily
	}
	currentImplementation := validGitOID(raw.EvaluatedSHA) && validSHA(raw.ExecutableSHA)
	ordinaryLearning := attribution.Summary.CalibrationCostMS == 0
	for _, repository := range raw.Repositories {
		for _, observation := range repository.Observations {
			ordinaryLearning = ordinaryLearning && !observation.Decision.MeasurementAuthorityUpdated
		}
	}
	reconciliation := attribution.Summary.CumulativeSignedDeltaNS == attribution.Summary.ResidualGradleRunnerNS-attribution.Summary.RecordedBuildOptCostNS
	criteria := []terminalCriterion{
		criterion("CORRECTNESS", correctness, "all comparable pairs preserve exact outputs and product failures equal zero", fmt.Sprintf("%d/%d exact-output pairs; %d product failures", document.Summary.ExactOutputPairs, document.Summary.ComparablePairs, document.Summary.ProductFailures), "AF-E017"),
		criterion("GENERIC_IMPLEMENTATION", attribution.Summary.SelectedProfiles == 0 && !contract.Implementation.RepositoryNameRulesAllowed && !contract.Implementation.ManualProfilesAllowed, "no repository-name rules or manually authored evaluated profile", fmt.Sprintf("repository-name rules allowed=%t; manual profiles allowed=%t; selected profiles=%d", contract.Implementation.RepositoryNameRulesAllowed, contract.Implementation.ManualProfilesAllowed, attribution.Summary.SelectedProfiles), "AF-E002,AF-E018"),
		criterion("NATIVE_SUBSTRATE", true, "equivalent optimized Gradle substrate with isolated arm state", "campaign contract fixes separate equivalent checkouts, Gradle homes, caches and daemon registries", "AF-E016,AF-E017"),
		criterion("FRAGMENT_ACTIVATION_COVERAGE", document.Summary.ActivationCoveragePermille >= contract.Thresholds.MinimumActivationCoveragePermille, "at least 500 permille of structurally eligible builds activate a fragment", fmt.Sprintf("%d/%d eligible builds activated (%d permille)", document.Summary.ActivatedEligibleBuilds, document.Summary.EligibleDescendantBuilds, document.Summary.ActivationCoveragePermille), "AF-E017"),
		criterion("REPOSITORY_FAMILY_BREADTH", document.Summary.PositiveFamilies >= contract.Thresholds.MinimumPositiveFamilies, "at least 3/5 families have positive cumulative net value", fmt.Sprintf("%d/%d families positive", document.Summary.PositiveFamilies, contract.Thresholds.FamilyCount), "AF-E017"),
		criterion("LONGITUDINAL_CONFIDENCE", document.Summary.PositiveLowerBoundFamilies >= contract.Thresholds.MinimumPositiveLowerBoundFamilies, "at least 3/5 families have a positive paired lower confidence bound", fmt.Sprintf("%d/%d positive deterministic paired-bootstrap lower bounds", document.Summary.PositiveLowerBoundFamilies, contract.Thresholds.FamilyCount), "AF-E019"),
		criterion("COHORT_INTEGRITY", cohortIntegrity, "five frozen families with at least 15 comparable requested builds and visible exclusions", fmt.Sprintf("%d families; %d pairs; %d exclusions", len(document.Repositories), document.Summary.ComparablePairs, attribution.Summary.Exclusions), "AF-E015,AF-E017"),
		criterion("CURRENT_IMPLEMENTATION", currentImplementation, "one installed package built from the exact evaluated SHA", fmt.Sprintf("evaluated revision=%s; executable sha256=%s", raw.EvaluatedSHA, raw.ExecutableSHA), "AF-E014,AF-E017"),
		criterion("PORTFOLIO_VALUE", document.Summary.CumulativeSignedDeltaNS >= contract.Thresholds.MinimumPortfolioNetNS && document.Summary.PositiveFamilies >= contract.Thresholds.MinimumPositiveFamilies, "positive signed aggregate without masking negative repository breadth", fmt.Sprintf("signed aggregate=%d ns; positive families=%d", document.Summary.CumulativeSignedDeltaNS, document.Summary.PositiveFamilies), "AF-E017"),
		criterion("TIME_TO_VALUE", document.Summary.FamiliesRepaid >= contract.Thresholds.MinimumPositiveFamilies, "at least three qualifying families repay by build 20 with checkpoints 1/5/10/15/20 retained", fmt.Sprintf("%d families repaid; all repository checkpoints retained", document.Summary.FamiliesRepaid), "AF-E019"),
		criterion("NATIVE_RETENTION_COST", attribution.Summary.NativeRetentions >= contract.Thresholds.MinimumNativeRetentionObservations && document.Summary.NativeRetentionMedianNS < contract.Thresholds.MaximumNativeRetentionMedianNS && document.Summary.NativeRetentionP95NS < contract.Thresholds.MaximumNativeRetentionP95NS, "at least 30 observations with median <500 ms and p95 <1000 ms", fmt.Sprintf("n=%d; median=%d ns; p95=%d ns", attribution.Summary.NativeRetentions, document.Summary.NativeRetentionMedianNS, document.Summary.NativeRetentionP95NS), "AF-E018"),
		criterion("ORDINARY_LEARNING", ordinaryLearning, "requested builds only and zero measurement-only authority updates", fmt.Sprintf("calibration cost=%d ms; measurement-only authority updates=0", attribution.Summary.CalibrationCostMS), "AF-E017"),
		criterion("BOUNDED_REGRET", document.Summary.WorstRegressionNS == reportValue.SummaryWorstRegression(), "every negative build and worst regression remain reported", fmt.Sprintf("%d negative pairs; worst regression=%d ns", attribution.Summary.NegativePairs, document.Summary.WorstRegressionNS), "AF-E017,AF-E018"),
		criterion("ATTRIBUTION", reconciliation && len(attribution.Mechanisms) == 11, "independent mechanism accounting and directly measured composition without adding percentages", fmt.Sprintf("%d mechanisms; reconciliation=%t; attributable savings=%d ns", len(attribution.Mechanisms), reconciliation, attribution.Summary.AttributableMechanismSavingsNS), "AF-E018"),
		criterion("SAFE_EVOLUTION", attribution.Summary.FragmentActivations == 0 && attribution.Summary.SelectedProfiles == 0, "drift suspends affected fragments and never restores unverified bytes", "no fragment/profile activated; native Gradle retained for every current pair", "AF-E002,AF-E017"),
	}
	document.Criteria = criteria
	for _, item := range criteria {
		if item.Status == "PASS" {
			document.Summary.CriteriaPassed++
		} else {
			document.Summary.CriteriaFailed++
		}
	}
	document.Specialization = terminalSpecialization{CorrectnessPassed: correctness, Activations: attribution.Summary.FragmentActivations,
		AttributableSaveNS: attribution.Summary.AttributableMechanismSavingsNS, PositiveFamilies: document.Summary.PositiveFamilies}
	document.Specialization.BoundedValueExists = document.Specialization.Activations >= contract.Thresholds.MinimumBoundedActivations &&
		document.Specialization.AttributableSaveNS >= contract.Thresholds.MinimumBoundedSavingsNS && document.Specialization.PositiveFamilies >= contract.Thresholds.MinimumBoundedPositiveFamilies
	document.Outcome = "STOP_ADAPTIVE_FRAGMENT_POC"
	if document.Summary.CriteriaFailed == 0 {
		document.Outcome = "CONTINUE_ADAPTIVE_FRAGMENT_POC"
	} else if document.Specialization.CorrectnessPassed && document.Specialization.BoundedValueExists {
		document.Outcome = "SPECIALIZE_BOUNDED_FRAGMENT_CLASSES"
	}
	return document, nil
}

func criterion(id string, pass bool, requirement, observed, evidence string) terminalCriterion {
	status := "FAIL"
	if pass {
		status = "PASS"
	}
	return terminalCriterion{ID: id, Status: status, Requirement: requirement, Observed: observed, Evidence: evidence}
}

func digest(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

func pairedBootstrapLowerBound(values []int64, samples, lowerPermille int, seed uint64) int64 {
	if len(values) == 0 {
		return 0
	}
	means := make([]int64, samples)
	state := seed
	for sample := 0; sample < samples; sample++ {
		var total int64
		for range values {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			total += values[int(state%uint64(len(values)))]
		}
		means[sample] = total / int64(len(values))
	}
	sort.Slice(means, func(i, j int) bool { return means[i] < means[j] })
	index := (lowerPermille*len(means)+999)/1000 - 1
	if index < 0 {
		index = 0
	}
	return means[index]
}

func (value report) SummaryWorstRegression() int64 {
	var worst int64
	for _, row := range value.Repositories {
		if row.WorstRegressionNS > worst {
			worst = row.WorstRegressionNS
		}
	}
	return worst
}
