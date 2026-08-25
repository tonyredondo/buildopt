// Command adaptive-fragment-prior produces and validates the synthetic AF-007
// cross-repository ranking proof. It executes no Gradle build or fragment.
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
	"reflect"
	"strings"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-prior/v1"
	outcome      = "SAFE_HYPOTHESIS_PRIORS_AVAILABLE"
)

type report struct {
	SchemaVersion      string                              `json:"schemaVersion"`
	WorkItem           string                              `json:"workItem"`
	CapturedAt         string                              `json:"capturedAt"`
	Policy             policy                              `json:"policy"`
	SourceObservations []adaptivefragment.PriorObservation `json:"sourceObservations"`
	Holdouts           []holdoutProof                      `json:"holdouts"`
	Rename             renameProof                         `json:"renameProof"`
	Priority           priorityProof                       `json:"priorityProof"`
	NoMatch            adaptivefragment.PriorRanking       `json:"noMatch"`
	Summary            summary                             `json:"summary"`
	Boundaries         boundaries                          `json:"boundaries"`
	Outcome            string                              `json:"outcome"`
}

type policy struct {
	FeatureWeights                   map[string]uint64 `json:"featureWeights"`
	TaskOrPluginMatchRequired        bool              `json:"taskOrPluginMatchRequired"`
	RepositoryIdentityFeatureAllowed bool              `json:"repositoryIdentityFeatureAllowed"`
	MinimumIndependentSources        uint64            `json:"minimumIndependentSources"`
	LocalCorrectnessRequired         bool              `json:"localCorrectnessRequired"`
	LocalValueRequired               bool              `json:"localValueRequired"`
}

type holdoutProof struct {
	Query   adaptivefragment.PriorQuery   `json:"query"`
	Ranking adaptivefragment.PriorRanking `json:"ranking"`
}

type renameProof struct {
	SourceRepositoryNamesReplaced bool                          `json:"sourceRepositoryNamesReplaced"`
	HoldoutRepositoryNameReplaced bool                          `json:"holdoutRepositoryNameReplaced"`
	InputOrderReversed            bool                          `json:"inputOrderReversed"`
	RankingIdentical              bool                          `json:"rankingIdentical"`
	RenamedRanking                adaptivefragment.PriorRanking `json:"renamedRanking"`
}

type priorityProof struct {
	OriginalTopHypothesis        string                        `json:"originalTopHypothesis"`
	ChangedTopHypothesis         string                        `json:"changedTopHypothesis"`
	TransferredEvidenceReordered bool                          `json:"transferredEvidenceReordered"`
	ChangedRanking               adaptivefragment.PriorRanking `json:"changedRanking"`
	ActivationAuthorizations     uint64                        `json:"activationAuthorizations"`
}

type summary struct {
	SourceRepositories          uint64 `json:"sourceRepositories"`
	SourceObservations          uint64 `json:"sourceObservations"`
	HoldoutRepositories         uint64 `json:"holdoutRepositories"`
	RankedHypotheses            uint64 `json:"rankedHypotheses"`
	RepositoryRenameComparisons uint64 `json:"repositoryRenameComparisons"`
	PriorityChanges             uint64 `json:"priorityChanges"`
	NoMatchCandidates           uint64 `json:"noMatchCandidates"`
	RejectedUnsafeInputs        uint64 `json:"rejectedUnsafeInputs"`
}

type boundaries struct {
	ProofOfConcept              bool   `json:"proofOfConcept"`
	SyntheticRankingVectors     bool   `json:"syntheticRankingVectors"`
	SyntheticTimingClaim        bool   `json:"syntheticTimingClaim"`
	GradleExecutions            uint64 `json:"gradleExecutions"`
	LocalCorrectnessTransferred bool   `json:"localCorrectnessTransferred"`
	LocalValueTransferred       bool   `json:"localValueTransferred"`
	ActivationAuthorized        bool   `json:"activationAuthorized"`
	ProductionAuthorized        bool   `json:"productionAuthorized"`
	TestOptimization            string `json:"testOptimization"`
}

func main() {
	output := flag.String("output", "", "write the AF-007 prior report")
	validate := flag.String("validate", "", "validate an AF-007 prior report")
	flag.Parse()
	if flag.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-prior (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	expected, err := buildReport()
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("prior report does not match recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment prior failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment prior: SAFE_HYPOTHESIS_PRIORS_AVAILABLE")
	}
}

func buildReport() (report, error) {
	sources := sourceObservations()
	firstQuery := query("holdout-a")
	first, err := adaptivefragment.RankHypotheses(firstQuery, sources)
	if err != nil {
		return report{}, err
	}
	secondQuery := query("holdout-b")
	second, err := adaptivefragment.RankHypotheses(secondQuery, sources)
	if err != nil {
		return report{}, err
	}
	renamedSources := renameAndReverse(sources)
	renamed, err := adaptivefragment.RankHypotheses(query("renamed-holdout"), renamedSources)
	if err != nil {
		return report{}, err
	}
	changedSources := sourceObservations()
	for index := range changedSources {
		switch changedSources[index].Hypothesis {
		case "TASK_CONTRACT_PATCH":
			changedSources[index].Outcome = adaptivefragment.PriorOutcomeNonPositive
			changedSources[index].LocalValuePassed = false
		case "SUBGRAPH_REDUCTION":
			changedSources[index].Outcome = adaptivefragment.PriorOutcomePositive
			changedSources[index].LocalValuePassed = true
		}
	}
	changed, err := adaptivefragment.RankHypotheses(firstQuery, changedSources)
	if err != nil {
		return report{}, err
	}
	noMatchQuery := firstQuery
	noMatchQuery.Features.TaskImplementationSHA256 = strings.Repeat("e", 64)
	noMatchQuery.Features.PluginVersionSHA256 = strings.Repeat("f", 64)
	noMatch, err := adaptivefragment.RankHypotheses(noMatchQuery, sources)
	if err != nil {
		return report{}, err
	}

	result := report{
		SchemaVersion: reportSchema, WorkItem: "AF-007", CapturedAt: "2026-08-25T14:00:00Z",
		Policy: policy{
			FeatureWeights:            map[string]uint64{"taskImplementation": 35, "pluginVersion": 25, "gradleMajor": 10, "taskClass": 10, "graphShape": 10, "outputShape": 10},
			TaskOrPluginMatchRequired: true, MinimumIndependentSources: 2,
			LocalCorrectnessRequired: true, LocalValueRequired: true,
		},
		SourceObservations: sources,
		Holdouts:           []holdoutProof{{Query: firstQuery, Ranking: first}, {Query: secondQuery, Ranking: second}},
		Rename: renameProof{SourceRepositoryNamesReplaced: true, HoldoutRepositoryNameReplaced: true,
			InputOrderReversed: true, RankingIdentical: reflect.DeepEqual(first, renamed), RenamedRanking: renamed},
		Priority: priorityProof{OriginalTopHypothesis: first.Candidates[0].Hypothesis,
			ChangedTopHypothesis:         changed.Candidates[0].Hypothesis,
			TransferredEvidenceReordered: first.Candidates[0].Hypothesis != changed.Candidates[0].Hypothesis,
			ChangedRanking:               changed},
		NoMatch: noMatch,
		Summary: summary{SourceRepositories: first.SourceRepositoryCount, SourceObservations: uint64(len(sources)),
			HoldoutRepositories: 2, RankedHypotheses: uint64(len(first.Candidates)), RepositoryRenameComparisons: 3,
			PriorityChanges: 1, NoMatchCandidates: uint64(len(noMatch.Candidates)), RejectedUnsafeInputs: invalidRejections(firstQuery)},
		Boundaries: boundaries{ProofOfConcept: true, SyntheticRankingVectors: true, TestOptimization: "OUT_OF_SCOPE"},
		Outcome:    outcome,
	}
	return result, nil
}

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-007" || candidate.Outcome != outcome ||
		candidate.CapturedAt != "2026-08-25T14:00:00Z" || candidate.Policy.RepositoryIdentityFeatureAllowed ||
		!candidate.Policy.TaskOrPluginMatchRequired || candidate.Policy.MinimumIndependentSources != 2 ||
		!candidate.Policy.LocalCorrectnessRequired || !candidate.Policy.LocalValueRequired || len(candidate.Holdouts) != 2 {
		return errors.New("prior report identity is invalid")
	}
	if !reflect.DeepEqual(candidate.Holdouts[0].Ranking, candidate.Holdouts[1].Ranking) || !candidate.Rename.RankingIdentical ||
		!candidate.Rename.SourceRepositoryNamesReplaced || !candidate.Rename.HoldoutRepositoryNameReplaced || !candidate.Rename.InputOrderReversed ||
		!reflect.DeepEqual(candidate.Holdouts[0].Ranking, candidate.Rename.RenamedRanking) {
		return errors.New("prior repository invariance is invalid")
	}
	if !candidate.Priority.TransferredEvidenceReordered || candidate.Priority.OriginalTopHypothesis != "TASK_CONTRACT_PATCH" ||
		candidate.Priority.ChangedTopHypothesis != "SUBGRAPH_REDUCTION" || candidate.Priority.ActivationAuthorizations != 0 {
		return errors.New("prior exploration ordering proof is invalid")
	}
	for _, ranking := range []adaptivefragment.PriorRanking{candidate.Holdouts[0].Ranking, candidate.Holdouts[1].Ranking,
		candidate.Rename.RenamedRanking, candidate.Priority.ChangedRanking, candidate.NoMatch} {
		if ranking.RepositoryIdentityUsedForScoring || ranking.ActivationAuthorized {
			return errors.New("prior ranking transferred repository identity or activation authority")
		}
		for _, ranked := range ranking.Candidates {
			if !ranked.LocalCorrectnessRequired || !ranked.LocalValueRequired || ranked.ActivationAuthorized {
				return errors.New("prior candidate transferred local authority")
			}
		}
	}
	if candidate.Summary != (summary{SourceRepositories: 4, SourceObservations: 6, HoldoutRepositories: 2,
		RankedHypotheses: 3, RepositoryRenameComparisons: 3, PriorityChanges: 1, RejectedUnsafeInputs: 6}) {
		return errors.New("prior summary is invalid")
	}
	if !candidate.Boundaries.ProofOfConcept || !candidate.Boundaries.SyntheticRankingVectors || candidate.Boundaries.SyntheticTimingClaim ||
		candidate.Boundaries.GradleExecutions != 0 || candidate.Boundaries.LocalCorrectnessTransferred || candidate.Boundaries.LocalValueTransferred ||
		candidate.Boundaries.ActivationAuthorized || candidate.Boundaries.ProductionAuthorized || candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("prior boundaries are invalid")
	}
	return nil
}

func sourceObservations() []adaptivefragment.PriorObservation {
	features := commonFeatures()
	return []adaptivefragment.PriorObservation{
		observation("source-a", "TASK_CONTRACT_PATCH", adaptivefragment.PriorOutcomePositive, true, features),
		observation("source-b", "TASK_CONTRACT_PATCH", adaptivefragment.PriorOutcomePositive, true, features),
		observation("source-c", "SUBGRAPH_REDUCTION", adaptivefragment.PriorOutcomePositive, true, features),
		observation("source-d", "SUBGRAPH_REDUCTION", adaptivefragment.PriorOutcomeNonPositive, false, features),
		observation("source-a", "OUTPUT_MATERIALIZATION", adaptivefragment.PriorOutcomeNonPositive, false, features),
		observation("source-b", "OUTPUT_MATERIALIZATION", adaptivefragment.PriorOutcomeNonPositive, false, features),
	}
}

func commonFeatures() adaptivefragment.PriorFeatures {
	return adaptivefragment.PriorFeatures{TaskImplementationSHA256: strings.Repeat("1", 64), PluginVersionSHA256: strings.Repeat("2", 64),
		GradleMajor: 9, TaskClass: "ARCHIVE", GraphShape: "MULTI_PROJECT", OutputShape: "MULTI_ARTIFACT"}
}

func query(repository string) adaptivefragment.PriorQuery {
	return adaptivefragment.PriorQuery{RepositoryScopeSHA256: digest(repository), Features: commonFeatures()}
}

func observation(repository, hypothesis, result string, valuePassed bool, features adaptivefragment.PriorFeatures) adaptivefragment.PriorObservation {
	return adaptivefragment.PriorObservation{EvidenceSHA256: digest(repository + hypothesis), RepositoryScopeSHA256: digest(repository),
		Hypothesis: hypothesis, Features: features, Outcome: result, ExactOutputs: true,
		LocalCorrectnessPassed: true, LocalValuePassed: valuePassed}
}

func renameAndReverse(sources []adaptivefragment.PriorObservation) []adaptivefragment.PriorObservation {
	renamed := append([]adaptivefragment.PriorObservation{}, sources...)
	repositories := map[string]string{}
	for index := range renamed {
		original := renamed[index].RepositoryScopeSHA256
		if repositories[original] == "" {
			repositories[original] = digest(fmt.Sprintf("renamed-source-%d", len(repositories)+1))
		}
		renamed[index].RepositoryScopeSHA256 = repositories[original]
		renamed[index].EvidenceSHA256 = digest(fmt.Sprintf("renamed-evidence-%d", index+1))
	}
	for left, right := 0, len(renamed)-1; left < right; left, right = left+1, right-1 {
		renamed[left], renamed[right] = renamed[right], renamed[left]
	}
	return renamed
}

func invalidRejections(query adaptivefragment.PriorQuery) uint64 {
	base := sourceObservations()
	mutations := []func([]adaptivefragment.PriorObservation){
		func(values []adaptivefragment.PriorObservation) {
			values[0].RepositoryScopeSHA256 = query.RepositoryScopeSHA256
		},
		func(values []adaptivefragment.PriorObservation) { values[0].ExactOutputs = false },
		func(values []adaptivefragment.PriorObservation) { values[0].ProductAttributableFail = true },
		func(values []adaptivefragment.PriorObservation) { values[0].LocalValuePassed = false },
		func(values []adaptivefragment.PriorObservation) { values[1].EvidenceSHA256 = values[0].EvidenceSHA256 },
		func(values []adaptivefragment.PriorObservation) {
			values[1].RepositoryScopeSHA256 = values[0].RepositoryScopeSHA256
		},
	}
	var rejected uint64
	for _, mutate := range mutations {
		candidate := append([]adaptivefragment.PriorObservation{}, base...)
		mutate(candidate)
		if _, err := adaptivefragment.RankHypotheses(query, candidate); err != nil {
			rejected++
		}
	}
	return rejected
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

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte("buildopt-af007-v1\x00" + value))
	return hex.EncodeToString(sum[:])
}
