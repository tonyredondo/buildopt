// Command adaptive-fragment-shadow produces and validates the AF-004 frozen
// history decomposition report. It never executes a build or activates a
// fragment.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-shadow/v1"
	outcome      = "FRAGMENT_COVERAGE_HYPOTHESIS_SUPPORTED"
)

var expectedRepositories = []string{
	"apache/groovy",
	"apache/kafka",
	"micronaut-projects/micronaut-core",
	"open-telemetry/opentelemetry-java-instrumentation",
	"spring-projects/spring-framework",
}

type sourceSummary struct {
	SchemaVersion string          `json:"schemaVersion"`
	CapturedAt    string          `json:"capturedAt"`
	Subjects      []sourceSubject `json:"subjects"`
}

type sourceSubject struct {
	Key            string `json:"key"`
	RepositoryID   string `json:"repositoryId"`
	TargetRevision string `json:"targetRevision"`
	Qualification struct {
		Status string `json:"status"`
	} `json:"qualification"`
	Lifetime struct {
		EligibleDescendants int `json:"eligibleDescendants"`
		SelectedReplays     int `json:"selectedReplays"`
	} `json:"lifetime"`
}

type subjectResult struct {
	RepositoryID              string                  `json:"repositoryId"`
	TargetRevision            string                  `json:"targetRevision"`
	QualificationStatus       string                  `json:"qualificationStatus"`
	SourceResultSHA256        string                  `json:"sourceResultSha256"`
	SourceCaptureSHA256       string                  `json:"sourceCaptureSha256"`
	CandidateFragments        []adaptivefragment.Kind `json:"candidateFragments"`
	EligibleDescendants       int                     `json:"eligibleDescendants"`
	WholeProfileSelections    int                     `json:"wholeProfileSelections"`
	ReproducedSelections      int                     `json:"reproducedSelections"`
	DescendantsWithFragment   int                     `json:"descendantsWithFragment"`
	PartialCompatibilityCount int                     `json:"partialCompatibilityCount"`
}

type report struct {
	SchemaVersion string                                  `json:"schemaVersion"`
	WorkItem      string                                  `json:"workItem"`
	CapturedAt    string                                  `json:"capturedAt"`
	Source        source                                  `json:"source"`
	Decomposition []fragmentDefinition                    `json:"decomposition"`
	Subjects      []subjectResult                         `json:"subjects"`
	Replay        []adaptivefragment.ShadowReplayDecision `json:"replay"`
	Summary       reportSummary                           `json:"summary"`
	Boundaries    boundaries                              `json:"boundaries"`
	Outcome       string                                  `json:"outcome"`
}

type source struct {
	SchemaVersion string `json:"schemaVersion"`
	SummarySHA256 string `json:"summarySha256"`
}

type fragmentDefinition struct {
	Kind     adaptivefragment.Kind         `json:"kind"`
	Bindings []adaptivefragment.BindingKey `json:"bindings"`
}

type reportSummary struct {
	RepositoryCount                int     `json:"repositoryCount"`
	QualifiedProfileCount          int     `json:"qualifiedProfileCount"`
	EarlyRetainedRepositoryCount   int     `json:"earlyRetainedRepositoryCount"`
	EligibleDescendants            int     `json:"eligibleDescendants"`
	WholeProfileSelections         int     `json:"wholeProfileSelections"`
	ReproducedWholeSelections      int     `json:"reproducedWholeSelections"`
	DescendantsWithFragment        int     `json:"descendantsWithFragment"`
	PartialCompatibilityCount      int     `json:"partialCompatibilityCount"`
	FragmentRetentionRatio         float64 `json:"fragmentRetentionRatio"`
	LookaheadObservationCount      int     `json:"lookaheadObservationCount"`
	ActivationAuthorizationCount   int     `json:"activationAuthorizationCount"`
	MeasurementOnlyBuildCount      int     `json:"measurementOnlyBuildCount"`
}

type boundaries struct {
	ProofOfConcept       bool   `json:"proofOfConcept"`
	ShadowOnly           bool   `json:"shadowOnly"`
	BuildTimingClaim     bool   `json:"buildTimingClaim"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	ProductionAuthorized bool   `json:"productionAuthorized"`
	TestOptimization     string `json:"testOptimization"`
}

type sourceResult struct {
	Repository    struct {
		ID string `json:"id"`
	} `json:"repository"`
	Qualification struct {
		TargetRevision string `json:"targetRevision"`
		CaptureSHA256  string `json:"captureSha256"`
	} `json:"qualification"`
	Raw []struct {
		Sequence             int    `json:"sequence"`
		Revision             string `json:"revision"`
		Selected             bool   `json:"selected"`
		Reason               string `json:"reason"`
		WrapperMatches       bool   `json:"wrapperMatchesQualification"`
		ExactRequiredOutputs bool   `json:"exactRequiredOutputs"`
	} `json:"observations"`
}

func main() {
	sourceDir := flag.String("source", "", "frozen lifetime breadth V3 evidence directory")
	output := flag.String("output", "", "write an AF-004 shadow report")
	validate := flag.String("validate", "", "validate an AF-004 shadow report")
	flag.Parse()
	if flag.NArg() != 0 || *sourceDir == "" || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-shadow --source <dir> (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	expected, err := buildReport(*sourceDir)
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSON(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("shadow report does not match recomputed frozen history")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment shadow replay failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment shadow replay: FRAGMENT_COVERAGE_HYPOTHESIS_SUPPORTED")
	}
}

func buildReport(sourceDir string) (report, error) {
	summaryPath := filepath.Join(sourceDir, "summary.json")
	var frozen sourceSummary
	if err := readJSON(summaryPath, &frozen); err != nil {
		return report{}, err
	}
	result := report{
		SchemaVersion: reportSchema, WorkItem: "AF-004", CapturedAt: frozen.CapturedAt,
		Source: source{SchemaVersion: frozen.SchemaVersion, SummarySHA256: fileSHA(summaryPath)},
		Decomposition: []fragmentDefinition{
			{Kind: adaptivefragment.KindOutputMaterialization, Bindings: []adaptivefragment.BindingKey{adaptivefragment.BindingWrapper, adaptivefragment.BindingProducerLineage, adaptivefragment.BindingOutputContract}},
			{Kind: adaptivefragment.KindSubgraph, Bindings: []adaptivefragment.BindingKey{adaptivefragment.BindingWorkflow, adaptivefragment.BindingWrapper, adaptivefragment.BindingProducerLineage, adaptivefragment.BindingOutputContract, adaptivefragment.BindingChangeFamily}},
		},
		Boundaries: boundaries{ProofOfConcept: true, ShadowOnly: true, TestOptimization: "OUT_OF_SCOPE"},
		Outcome: outcome,
	}
	for _, frozenSubject := range frozen.Subjects {
		resultPath := filepath.Join(sourceDir, frozenSubject.Key, "result.json")
		capturePath := filepath.Join(sourceDir, frozenSubject.Key, "qualification-capture.json")
		var raw sourceResult
		if err := readJSON(resultPath, &raw); err != nil {
			return report{}, err
		}
		captureSHA := fileSHA(capturePath)
		if raw.Repository.ID != frozenSubject.RepositoryID || raw.Qualification.TargetRevision != frozenSubject.TargetRevision ||
			raw.Qualification.CaptureSHA256 != captureSHA || captureSHA == "" {
			return report{}, errors.New("frozen subject provenance does not match its summary")
		}
		current := subjectResult{
			RepositoryID: frozenSubject.RepositoryID, TargetRevision: frozenSubject.TargetRevision,
			QualificationStatus: frozenSubject.Qualification.Status,
			SourceResultSHA256: fileSHA(resultPath), SourceCaptureSHA256: captureSHA,
			CandidateFragments: []adaptivefragment.Kind{}, EligibleDescendants: frozenSubject.Lifetime.EligibleDescendants,
			WholeProfileSelections: frozenSubject.Lifetime.SelectedReplays,
		}
		if frozenSubject.Qualification.Status == "QUALIFIED" {
			current.CandidateFragments = []adaptivefragment.Kind{adaptivefragment.KindOutputMaterialization, adaptivefragment.KindSubgraph}
			observations := make([]adaptivefragment.ShadowObservation, 0, len(raw.Raw))
			for _, observation := range raw.Raw {
				observations = append(observations, adaptivefragment.ShadowObservation{
					Sequence: observation.Sequence, Revision: observation.Revision, OriginalSelected: observation.Selected,
					OriginalReason: observation.Reason, WrapperMatches: observation.WrapperMatches,
					ExactRequiredOutputs: observation.ExactRequiredOutputs,
				})
			}
			replay, err := adaptivefragment.ReplayShadowProfile(observations)
			if err != nil {
				return report{}, err
			}
			result.Replay = append(result.Replay, replay...)
			if len(replay) != frozenSubject.Lifetime.EligibleDescendants {
				return report{}, errors.New("frozen descendant count does not match replay input")
			}
			for _, decision := range replay {
				if decision.ReproducedSelected {
					current.ReproducedSelections++
				}
				applicable := 0
				for _, fragment := range decision.Fragments {
					if fragment.Applicability == adaptivefragment.ShadowApplicable {
						applicable++
					}
				}
				if applicable > 0 {
					current.DescendantsWithFragment++
				}
				if applicable > 0 && !decision.ReproducedSelected {
					current.PartialCompatibilityCount++
				}
			}
		}
		result.Subjects = append(result.Subjects, current)
	}
	sort.Slice(result.Subjects, func(left, right int) bool { return result.Subjects[left].RepositoryID < result.Subjects[right].RepositoryID })
	result.Summary = summarize(result)
	return result, nil
}

func summarize(candidate report) reportSummary {
	result := reportSummary{RepositoryCount: len(candidate.Subjects)}
	for _, subject := range candidate.Subjects {
		if subject.QualificationStatus == "QUALIFIED" {
			result.QualifiedProfileCount++
		} else {
			result.EarlyRetainedRepositoryCount++
		}
		result.EligibleDescendants += subject.EligibleDescendants
		result.WholeProfileSelections += subject.WholeProfileSelections
		result.ReproducedWholeSelections += subject.ReproducedSelections
		result.DescendantsWithFragment += subject.DescendantsWithFragment
		result.PartialCompatibilityCount += subject.PartialCompatibilityCount
	}
	for _, decision := range candidate.Replay {
		if decision.MaxSourceSequence > decision.Sequence {
			result.LookaheadObservationCount++
		}
	}
	if result.EligibleDescendants > 0 {
		result.FragmentRetentionRatio = float64(result.DescendantsWithFragment) / float64(result.EligibleDescendants)
	}
	return result
}

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-004" || candidate.Outcome != outcome ||
		candidate.CapturedAt == "" || candidate.Source.SchemaVersion != "buildopt.evidence/poc-lifetime-breadth/v3" ||
		!isSHA(candidate.Source.SummarySHA256) || len(candidate.Subjects) != 5 || len(candidate.Decomposition) != 2 {
		return errors.New("shadow report identity is invalid")
	}
	for index, repository := range expectedRepositories {
		if candidate.Subjects[index].RepositoryID != repository || !isSHA(candidate.Subjects[index].SourceResultSHA256) ||
			!isSHA(candidate.Subjects[index].SourceCaptureSHA256) {
			return errors.New("shadow report subject is invalid")
		}
	}
	if !reflect.DeepEqual(candidate.Summary, summarize(candidate)) || candidate.Summary.QualifiedProfileCount != 1 ||
		candidate.Summary.EarlyRetainedRepositoryCount != 4 || candidate.Summary.EligibleDescendants != 6 ||
		candidate.Summary.WholeProfileSelections != 1 || candidate.Summary.ReproducedWholeSelections != 1 ||
		candidate.Summary.DescendantsWithFragment < 3 || candidate.Summary.PartialCompatibilityCount != 5 ||
		candidate.Summary.FragmentRetentionRatio < 0.5 || candidate.Summary.LookaheadObservationCount != 0 ||
		candidate.Summary.ActivationAuthorizationCount != 0 || candidate.Summary.MeasurementOnlyBuildCount != 0 {
		return errors.New("shadow report exit gate is not satisfied")
	}
	if !candidate.Boundaries.ProofOfConcept || !candidate.Boundaries.ShadowOnly || candidate.Boundaries.BuildTimingClaim ||
		candidate.Boundaries.ActivationAuthorized || candidate.Boundaries.ProductionAuthorized ||
		candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("shadow report boundaries are invalid")
	}
	return nil
}

func readJSON(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, target); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o644)
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
