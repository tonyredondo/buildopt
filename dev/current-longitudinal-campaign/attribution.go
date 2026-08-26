package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	attributionSchema         = "buildopt.poc/current-longitudinal-attribution/v1"
	attributionContractSchema = "buildopt.poc/current-longitudinal-attribution-contract/v1"
)

type attributionContract struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	SourceSchemas struct {
		Raw    string `json:"raw"`
		Report string `json:"report"`
	} `json:"sourceSchemas"`
	WorkflowClasses []workflowRule `json:"workflowClasses"`
	Mechanisms      []string       `json:"mechanisms"`
	Outcomes        []string       `json:"outcomes"`
	Boundaries      boundaries     `json:"boundaries"`
}

type workflowRule struct {
	ID      string `json:"id"`
	Match   string `json:"match"`
	Pattern string `json:"pattern"`
}

type attributionDocument struct {
	SchemaVersion   string                 `json:"schemaVersion"`
	WorkItem        string                 `json:"workItem"`
	CapturedAt      string                 `json:"capturedAt"`
	EvaluatedSHA    string                 `json:"evaluatedRevision"`
	RawSHA          string                 `json:"rawSha256"`
	ReportSHA       string                 `json:"reportSha256"`
	ContractSHA     string                 `json:"contractSha256"`
	Outcome         string                 `json:"outcome"`
	Summary         attributionSummary     `json:"summary"`
	Repositories    []attributionRow       `json:"repositories"`
	ByChangeShape   []attributionGroup     `json:"byChangeShape"`
	ByWorkflowClass []attributionGroup     `json:"byWorkflowClass"`
	DecisionReasons []countedValue         `json:"decisionReasons"`
	Mechanisms      []mechanismAttribution `json:"mechanisms"`
	Boundaries      boundaries             `json:"boundaries"`
}

type attributionSummary struct {
	ComparablePairs                int                    `json:"comparablePairs"`
	ExactOutputPairs               int                    `json:"exactOutputPairs"`
	PositivePairs                  int                    `json:"positivePairs"`
	NegativePairs                  int                    `json:"negativePairs"`
	SelectedProfiles               int                    `json:"selectedProfiles"`
	NativeRetentions               int                    `json:"nativeRetentions"`
	FragmentActivations            int                    `json:"fragmentActivations"`
	Exclusions                     int                    `json:"exclusions"`
	ProductFailures                int                    `json:"productFailures"`
	CumulativeSignedDeltaNS        int64                  `json:"cumulativeSignedDeltaNs"`
	RecordedBuildOptCostNS         int64                  `json:"recordedBuildOptCostNs"`
	ResidualGradleRunnerNS         int64                  `json:"residualGradleRunnerNs"`
	AttributableMechanismSavingsNS int64                  `json:"attributableMechanismSavingsNs"`
	DependencyPreparationNS        int64                  `json:"dependencyPreparationNs"`
	CalibrationCostMS              int64                  `json:"calibrationCostMs"`
	PaybackOrdinal                 *int                   `json:"paybackOrdinal"`
	NativeRetentionMedianNS        int64                  `json:"nativeRetentionMedianNs"`
	NativeRetentionP95NS           int64                  `json:"nativeRetentionP95Ns"`
	WorstRegressionNS              int64                  `json:"worstRegressionNs"`
	Diagnostics                    attributionDiagnostics `json:"diagnostics"`
}

type attributionDiagnostics struct {
	MatchingNS              int64 `json:"matchingNs"`
	LocalStateNS            int64 `json:"localStateNs"`
	CentralStateNS          int64 `json:"centralStateNs"`
	DiscoveryLearningNS     int64 `json:"discoveryLearningNs"`
	GradleSetupNS           int64 `json:"gradleSetupNs"`
	MaterializationNS       int64 `json:"materializationNs"`
	OutputVerificationNS    int64 `json:"outputVerificationNs"`
	InternalUnattributedNS  int64 `json:"internalUnattributedNs"`
	ExternalGapNS           int64 `json:"externalGapNs"`
	OtherRecordedBuildOptNS int64 `json:"otherRecordedBuildOptNs"`
}

type attributionRow struct {
	Key                     string                 `json:"key"`
	RepositoryID            string                 `json:"repositoryId"`
	WorkflowClass           string                 `json:"workflowClass"`
	Outcome                 string                 `json:"outcome"`
	ComparablePairs         int                    `json:"comparablePairs"`
	PositivePairs           int                    `json:"positivePairs"`
	NegativePairs           int                    `json:"negativePairs"`
	SelectedProfiles        int                    `json:"selectedProfiles"`
	NativeRetentions        int                    `json:"nativeRetentions"`
	FragmentActivations     int                    `json:"fragmentActivations"`
	Exclusions              int                    `json:"exclusions"`
	CumulativeSignedDeltaNS int64                  `json:"cumulativeSignedDeltaNs"`
	ColdStartSignedDeltaNS  int64                  `json:"coldStartSignedDeltaNs"`
	ContinuingSignedDeltaNS int64                  `json:"continuingSignedDeltaNs"`
	RecordedBuildOptCostNS  int64                  `json:"recordedBuildOptCostNs"`
	ResidualGradleRunnerNS  int64                  `json:"residualGradleRunnerNs"`
	DependencyPreparationNS int64                  `json:"dependencyPreparationNs"`
	CalibrationCostMS       int64                  `json:"calibrationCostMs"`
	PaybackOrdinal          *int                   `json:"paybackOrdinal"`
	NativeRetentionMedianNS int64                  `json:"nativeRetentionMedianNs"`
	NativeRetentionP95NS    int64                  `json:"nativeRetentionP95Ns"`
	WorstRegressionNS       int64                  `json:"worstRegressionNs"`
	Diagnostics             attributionDiagnostics `json:"diagnostics"`
}

type attributionGroup struct {
	ID                      string `json:"id"`
	ComparablePairs         int    `json:"comparablePairs"`
	PositivePairs           int    `json:"positivePairs"`
	CumulativeSignedDeltaNS int64  `json:"cumulativeSignedDeltaNs"`
	RecordedBuildOptCostNS  int64  `json:"recordedBuildOptCostNs"`
	ResidualGradleRunnerNS  int64  `json:"residualGradleRunnerNs"`
	WorstRegressionNS       int64  `json:"worstRegressionNs"`
}

type countedValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type mechanismAttribution struct {
	ID                    string `json:"id"`
	Activations           int    `json:"activations"`
	ObservedCostNS        int64  `json:"observedCostNs"`
	AttributableSavingsNS int64  `json:"attributableSavingsNs"`
	Status                string `json:"status"`
}

type attributionAccumulator struct {
	pairs       int
	positive    int
	negative    int
	delta       int64
	recorded    int64
	residual    int64
	worst       int64
	overheads   []int64
	diagnostics attributionDiagnostics
}

func runAttribution(rawPath, reportPath, contractPath, attributionPath string, write bool) {
	rawBytes, raw, reportBytes, report, contractBytes, contract, err := readAttributionInputs(rawPath, reportPath, contractPath)
	if err != nil {
		fatal(err)
	}
	document, err := deriveAttribution(rawBytes, raw, reportBytes, report, contractBytes, contract)
	if err != nil {
		fatal(err)
	}
	if attributionPath == "" || write {
		encoded, marshalErr := json.MarshalIndent(document, "", "  ")
		if marshalErr != nil {
			fatal(marshalErr)
		}
		if write {
			encoded = append(encoded, '\n')
			if err := os.WriteFile(attributionPath, encoded, 0o644); err != nil {
				fatal(err)
			}
			return
		}
		fmt.Printf("%s\n", encoded)
		return
	}
	actualBytes, err := os.ReadFile(attributionPath)
	if err != nil {
		fatal(err)
	}
	var actual attributionDocument
	if err := decodeStrict(actualBytes, &actual); err != nil {
		fatal(err)
	}
	wanted, _ := json.Marshal(document)
	got, _ := json.Marshal(actual)
	if string(wanted) != string(got) {
		fatal(errors.New("current longitudinal attribution is not the deterministic source aggregate"))
	}
	fmt.Println("current longitudinal attribution: valid")
}

func readAttributionInputs(rawPath, reportPath, contractPath string) ([]byte, rawEvidence, []byte, report, []byte, attributionContract, error) {
	var raw rawEvidence
	var reportValue report
	var contract attributionContract
	rawBytes, err := os.ReadFile(rawPath)
	if err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	if raw, err = decodeRaw(rawBytes); err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	if err = decodeStrict(reportBytes, &reportValue); err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	if err = decodeStrict(contractBytes, &contract); err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	if err = validateAttributionContract(contract); err != nil {
		return nil, raw, nil, reportValue, nil, contract, err
	}
	return rawBytes, raw, reportBytes, reportValue, contractBytes, contract, nil
}

func validateAttributionContract(contract attributionContract) error {
	if contract.SchemaVersion != attributionContractSchema || contract.WorkItem != "AF-014D" ||
		contract.SourceSchemas.Raw != rawSchema || contract.SourceSchemas.Report != reportSchema ||
		len(contract.WorkflowClasses) != 4 || len(contract.Mechanisms) != 11 || len(contract.Outcomes) != 2 ||
		!oneOf("CURRENT_VALUE_ATTRIBUTED", contract.Outcomes...) ||
		!oneOf("CURRENT_VALUE_NOT_ATTRIBUTABLE", contract.Outcomes...) ||
		!contract.Boundaries.ProofOfConcept || contract.Boundaries.ProductionAuthorized ||
		contract.Boundaries.SoakRequired || contract.Boundaries.DesignPartnerRequired ||
		contract.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("current longitudinal attribution contract is invalid")
	}
	seen := map[string]bool{}
	requiredWorkflowClasses := []string{"TEST_COMPILATION", "SHADOW_JAR", "ASSEMBLY", "PRODUCTION_CLASSES"}
	for _, rule := range contract.WorkflowClasses {
		if rule.ID == "" || rule.Pattern == "" || !oneOf(rule.ID, requiredWorkflowClasses...) ||
			!oneOf(rule.Match, "CONTAINS", "SUFFIX") || seen[rule.ID] {
			return errors.New("workflow classification rule is invalid")
		}
		seen[rule.ID] = true
	}
	requiredMechanisms := []string{"BUILD_IMPACT", "REVIEWED_PATCH_TASK", "SAFE_CACHE_LOCALITY", "MATCHING",
		"STATE_SYNC", "DISCOVERY_LEARNING", "OUTPUT_MATERIALIZATION", "OUTPUT_VERIFICATION", "GRADLE_SETUP",
		"COMPOSED_PATH", "NATIVE_RETENTION_PATH"}
	seen = map[string]bool{}
	for _, mechanism := range contract.Mechanisms {
		if !oneOf(mechanism, requiredMechanisms...) || seen[mechanism] {
			return errors.New("mechanism attribution set is invalid")
		}
		seen[mechanism] = true
	}
	return nil
}

func deriveAttribution(rawBytes []byte, raw rawEvidence, reportBytes []byte, reportValue report, contractBytes []byte, contract attributionContract) (attributionDocument, error) {
	rawDigest := sha256.Sum256(rawBytes)
	reportDigest := sha256.Sum256(reportBytes)
	contractDigest := sha256.Sum256(contractBytes)
	if reportValue.RawSHA != hex.EncodeToString(rawDigest[:]) || reportValue.EvaluatedSHA != raw.EvaluatedSHA ||
		reportValue.Summary.ComparablePairs != countObservations(raw) ||
		reportValue.Summary.ProductFailures != raw.ProductFailures {
		return attributionDocument{}, errors.New("current longitudinal report is not bound to the raw evidence")
	}
	document := attributionDocument{SchemaVersion: attributionSchema, WorkItem: "AF-014D",
		CapturedAt: raw.CapturedAt, EvaluatedSHA: raw.EvaluatedSHA,
		RawSHA: hex.EncodeToString(rawDigest[:]), ReportSHA: hex.EncodeToString(reportDigest[:]),
		ContractSHA: hex.EncodeToString(contractDigest[:]), Outcome: "CURRENT_VALUE_NOT_ATTRIBUTABLE",
		Boundaries: contract.Boundaries}
	shapeGroups := map[string]*attributionAccumulator{}
	workflowGroups := map[string]*attributionAccumulator{}
	reasons := map[string]int{}
	all := attributionAccumulator{}
	for _, repository := range raw.Repositories {
		workflowClass, err := classifyWorkflow(repository.Workflow, contract.WorkflowClasses)
		if err != nil {
			return attributionDocument{}, err
		}
		rowAccumulator := attributionAccumulator{}
		row := attributionRow{Key: repository.Key, RepositoryID: repository.RepositoryID,
			WorkflowClass: workflowClass, ComparablePairs: len(repository.Observations),
			Exclusions: len(repository.Exclusions), DependencyPreparationNS: repository.DependencyPreparation.WallNS}
		for _, observation := range repository.Observations {
			addObservation(&rowAccumulator, observation)
			addObservation(&all, observation)
			if shapeGroups[observation.ChangeShape] == nil {
				shapeGroups[observation.ChangeShape] = &attributionAccumulator{}
			}
			addObservation(shapeGroups[observation.ChangeShape], observation)
			if workflowGroups[workflowClass] == nil {
				workflowGroups[workflowClass] = &attributionAccumulator{}
			}
			addObservation(workflowGroups[workflowClass], observation)
			reasons[observation.Decision.Reason]++
			if observation.Sequence == 1 {
				row.ColdStartSignedDeltaNS = observation.SignedDeltaNS
			} else {
				row.ContinuingSignedDeltaNS += observation.SignedDeltaNS
			}
			if observation.Decision.SelectionSelected {
				row.SelectedProfiles++
			}
			if !observation.Decision.SelectionSelected {
				row.NativeRetentions++
			}
			row.FragmentActivations += len(observation.Decision.ActivatedFragments)
			if observation.Decision.CalibrationPerformed {
				row.CalibrationCostMS += observation.Decision.CalibrationCostMS
			}
		}
		completeRow(&row, rowAccumulator)
		for _, summary := range reportValue.Repositories {
			if summary.Key == row.Key {
				row.Outcome = summary.Outcome
			}
		}
		document.Repositories = append(document.Repositories, row)
		document.Summary.DependencyPreparationNS += row.DependencyPreparationNS
		document.Summary.CalibrationCostMS += row.CalibrationCostMS
		document.Summary.SelectedProfiles += row.SelectedProfiles
		document.Summary.NativeRetentions += row.NativeRetentions
		document.Summary.FragmentActivations += row.FragmentActivations
		document.Summary.Exclusions += row.Exclusions
	}
	sort.Slice(document.Repositories, func(i, j int) bool { return document.Repositories[i].Key < document.Repositories[j].Key })
	document.Summary.ComparablePairs = all.pairs
	document.Summary.ExactOutputPairs = all.pairs
	document.Summary.PositivePairs = all.positive
	document.Summary.NegativePairs = all.negative
	document.Summary.ProductFailures = raw.ProductFailures
	document.Summary.CumulativeSignedDeltaNS = all.delta
	document.Summary.RecordedBuildOptCostNS = all.recorded
	document.Summary.ResidualGradleRunnerNS = all.residual
	document.Summary.NativeRetentionMedianNS = nearestRank(all.overheads, 50)
	document.Summary.NativeRetentionP95NS = nearestRank(all.overheads, 95)
	document.Summary.WorstRegressionNS = all.worst
	document.Summary.Diagnostics = all.diagnostics
	document.ByChangeShape = groups(shapeGroups)
	document.ByWorkflowClass = groups(workflowGroups)
	for value, count := range reasons {
		document.DecisionReasons = append(document.DecisionReasons, countedValue{Value: value, Count: count})
	}
	sort.Slice(document.DecisionReasons, func(i, j int) bool { return document.DecisionReasons[i].Value < document.DecisionReasons[j].Value })
	mechanismCosts := map[string]int64{
		"MATCHING":               all.diagnostics.MatchingNS,
		"STATE_SYNC":             all.diagnostics.LocalStateNS + all.diagnostics.CentralStateNS,
		"DISCOVERY_LEARNING":     all.diagnostics.DiscoveryLearningNS,
		"OUTPUT_MATERIALIZATION": all.diagnostics.MaterializationNS,
		"OUTPUT_VERIFICATION":    all.diagnostics.OutputVerificationNS,
		"GRADLE_SETUP":           all.diagnostics.GradleSetupNS,
		"NATIVE_RETENTION_PATH":  all.recorded,
	}
	for _, id := range contract.Mechanisms {
		entry := mechanismAttribution{ID: id, ObservedCostNS: mechanismCosts[id], Status: "NOT_ACTIVATED"}
		if entry.ObservedCostNS > 0 {
			entry.Status = "OBSERVED_COST_ONLY"
		}
		if id == "NATIVE_RETENTION_PATH" {
			entry.Activations = document.Summary.NativeRetentions
			entry.Status = "MEASURED"
		}
		document.Mechanisms = append(document.Mechanisms, entry)
	}
	if document.Summary.FragmentActivations > 0 || document.Summary.SelectedProfiles > 0 {
		document.Outcome = "CURRENT_VALUE_ATTRIBUTED"
	}
	return document, nil
}

func countObservations(raw rawEvidence) int {
	total := 0
	for _, repository := range raw.Repositories {
		total += len(repository.Observations)
	}
	return total
}

func classifyWorkflow(workflow []string, rules []workflowRule) (string, error) {
	if len(workflow) == 0 {
		return "", errors.New("repository workflow is empty")
	}
	task := workflow[0]
	for _, rule := range rules {
		if (rule.Match == "CONTAINS" && strings.Contains(task, rule.Pattern)) || (rule.Match == "SUFFIX" && strings.HasSuffix(task, rule.Pattern)) {
			return rule.ID, nil
		}
	}
	return "", fmt.Errorf("workflow %q has no generic classification", task)
}

func addObservation(acc *attributionAccumulator, observation observation) {
	t := observation.Decision.Timing
	recorded := observation.Candidate.WallNS - t.GradleExecutionNS
	if recorded < 0 {
		recorded = 0
	}
	residual := observation.SignedDeltaNS + recorded
	acc.pairs++
	if observation.SignedDeltaNS > 0 {
		acc.positive++
	} else if observation.SignedDeltaNS < 0 {
		acc.negative++
	}
	acc.delta += observation.SignedDeltaNS
	acc.recorded += recorded
	acc.residual += residual
	if regression := -observation.SignedDeltaNS; regression > acc.worst {
		acc.worst = regression
	}
	acc.overheads = append(acc.overheads, recorded)
	d := &acc.diagnostics
	d.MatchingNS += t.Diagnostics.MatchingNS
	d.LocalStateNS += t.Diagnostics.LocalStateNS
	d.CentralStateNS += t.Diagnostics.CentralStateNS
	d.DiscoveryLearningNS += t.Diagnostics.DiscoveryLearningNS
	d.GradleSetupNS += t.Diagnostics.GradleSetupNS
	d.MaterializationNS += t.Diagnostics.MaterializationNS
	d.OutputVerificationNS += t.Diagnostics.OutputVerificationNS
	d.InternalUnattributedNS += t.UnattributedNS
	externalGap := observation.Candidate.WallNS - t.TotalNS
	if externalGap > 0 {
		d.ExternalGapNS += externalGap
	}
	known := t.Diagnostics.MatchingNS + t.Diagnostics.LocalStateNS + t.Diagnostics.CentralStateNS +
		t.Diagnostics.DiscoveryLearningNS + t.Diagnostics.GradleSetupNS + t.Diagnostics.MaterializationNS +
		t.Diagnostics.OutputVerificationNS + t.UnattributedNS
	if externalGap > 0 {
		known += externalGap
	}
	if other := recorded - known; other > 0 {
		d.OtherRecordedBuildOptNS += other
	}
}

func completeRow(row *attributionRow, acc attributionAccumulator) {
	row.PositivePairs = acc.positive
	row.NegativePairs = acc.negative
	row.CumulativeSignedDeltaNS = acc.delta
	row.RecordedBuildOptCostNS = acc.recorded
	row.ResidualGradleRunnerNS = acc.residual
	row.NativeRetentionMedianNS = nearestRank(acc.overheads, 50)
	row.NativeRetentionP95NS = nearestRank(acc.overheads, 95)
	row.WorstRegressionNS = acc.worst
	row.Diagnostics = acc.diagnostics
}

func groups(values map[string]*attributionAccumulator) []attributionGroup {
	result := []attributionGroup{}
	for id, acc := range values {
		result = append(result, attributionGroup{ID: id, ComparablePairs: acc.pairs, PositivePairs: acc.positive, CumulativeSignedDeltaNS: acc.delta, RecordedBuildOptCostNS: acc.recorded, ResidualGradleRunnerNS: acc.residual, WorstRegressionNS: acc.worst})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func nearestRank(values []int64, percent int) int64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]int64{}, values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := (percent*len(ordered)+99)/100 - 1
	if index < 0 {
		index = 0
	}
	return ordered[index]
}
