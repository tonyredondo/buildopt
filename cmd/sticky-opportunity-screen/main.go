// Command sticky-opportunity-screen evaluates the preregistered SWL-014C
// detector gate from frozen public-repository evidence. It never edits a
// subject checkout or authorizes an optimization.
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
	"strings"

	"github.com/tonyredondo/buildopt/internal/durablecatalog"
)

const contractSchema = "buildopt.poc/sticky-wrapper-opportunity-gate-contract/v1"

type fileBinding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type costContract struct {
	TrialCostNs       uint64 `json:"trialCostNs"`
	ValidationCostNs  uint64 `json:"validationCostNs"`
	PublicationCostNs uint64 `json:"publicationCostNs"`
}

type gateContract struct {
	SchemaVersion          string       `json:"schemaVersion"`
	WorkItem               string       `json:"workItem"`
	GeneratedAt            string       `json:"generatedAt"`
	CohortManifest         fileBinding  `json:"cohortManifest"`
	SourceEvidence         fileBinding  `json:"sourceEvidence"`
	DetectorOrder          []string     `json:"detectorOrder"`
	MinimumPassingFamilies uint64       `json:"minimumPassingFamilies"`
	MaximumBuildsToRepay   uint64       `json:"maximumCompatibleBuildsToRepay"`
	Costs                  costContract `json:"costs"`
	Boundaries             struct {
		ProofOfConcept       bool   `json:"proofOfConcept"`
		ProductionAuthorized bool   `json:"productionAuthorized"`
		RepositoryNameRules  bool   `json:"repositoryNameRulesAllowed"`
		TaskNameRules        bool   `json:"taskNameRulesAllowed"`
		PathExtensionRules   bool   `json:"pathExtensionRulesAllowed"`
		ManualProfiles       bool   `json:"manualEvaluatedProfilesAllowed"`
		TestOptimization     string `json:"testOptimization"`
	} `json:"boundaries"`
}

type cohortManifest struct {
	SchemaVersion string            `json:"schemaVersion"`
	Repositories  []json.RawMessage `json:"repositories"`
}

type cohortFamily struct {
	Policy struct {
		Key             string   `json:"key"`
		RepositoryID    string   `json:"repositoryId"`
		Workflow        []string `json:"workflow"`
		RequiredOutputs []string `json:"requiredOutputs"`
	} `json:"policy"`
	Anchor   json.RawMessage   `json:"anchor"`
	Primary  []json.RawMessage `json:"primary"`
	Reserves []json.RawMessage `json:"reserves"`
}

type rawEvidence struct {
	SchemaVersion string            `json:"schemaVersion"`
	Repositories  []json.RawMessage `json:"repositories"`
}

type rawFamily struct {
	Key          string           `json:"key"`
	RepositoryID string           `json:"repositoryId"`
	Observations []rawObservation `json:"observations"`
}

type rawObservation struct {
	Sequence     uint64 `json:"sequence"`
	ExactOutputs bool   `json:"exactOutputs"`
	Candidate    struct {
		OutputSHA256 string `json:"outputSha256"`
	} `json:"candidate"`
	Control struct {
		OutputSHA256 string `json:"outputSha256"`
	} `json:"control"`
	CandidateDecision struct {
		GraphSelectedProjects uint64 `json:"graphSelectedProjects"`
		GraphOmittedProjects  uint64 `json:"graphOmittedProjects"`
		GraphTotalProjects    uint64 `json:"graphTotalProjects"`
		PublicAction          *struct {
			CandidatePlanSHA256 string `json:"candidatePlanSha256"`
			BindingDigest       string `json:"bindingDigest"`
			ProjectedSavingNs   uint64 `json:"projectedSavingNs"`
		} `json:"publicAction,omitempty"`
	} `json:"candidateDecision"`
}

func main() {
	flags := flag.NewFlagSet("sticky-opportunity-screen", flag.ExitOnError)
	root := flags.String("root", ".", "repository root")
	contractPath := flags.String("contract", "specs/poc-sticky-wrapper-opportunity-gate-v1.json", "gate contract")
	output := flags.String("output", "", "write the recomputed report")
	validate := flags.String("validate", "", "validate an existing report")
	_ = flags.Parse(os.Args[1:])
	if flags.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: sticky-opportunity-screen [--root PATH] [--contract PATH] (--output PATH | --validate PATH)")
		os.Exit(64)
	}
	report, err := buildReport(*root, *contractPath)
	if err == nil && *validate != "" {
		var candidate durablecatalog.PublicScreenReport
		err = readJSONStrict(resolve(*root, *validate), &candidate)
		if err == nil && !reflect.DeepEqual(candidate, report) {
			err = errors.New("opportunity report differs from independently recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(resolve(*root, *output), report)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky opportunity screen failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Printf("sticky opportunity screen: %d/5 passing families; %s\n", report.PassingFamilies, report.Outcome)
	}
}

func buildReport(root, contractPath string) (durablecatalog.PublicScreenReport, error) {
	var contract gateContract
	if err := readJSONStrict(resolve(root, contractPath), &contract); err != nil {
		return durablecatalog.PublicScreenReport{}, err
	}
	if err := validateContract(contract); err != nil {
		return durablecatalog.PublicScreenReport{}, err
	}
	cohortRaw, err := readBoundFile(root, contract.CohortManifest)
	if err != nil {
		return durablecatalog.PublicScreenReport{}, err
	}
	evidenceRaw, err := readBoundFile(root, contract.SourceEvidence)
	if err != nil {
		return durablecatalog.PublicScreenReport{}, err
	}
	var cohort cohortManifest
	if err := json.Unmarshal(cohortRaw, &cohort); err != nil {
		return durablecatalog.PublicScreenReport{}, fmt.Errorf("decode cohort: %w", err)
	}
	var evidence rawEvidence
	if err := json.Unmarshal(evidenceRaw, &evidence); err != nil {
		return durablecatalog.PublicScreenReport{}, fmt.Errorf("decode source evidence: %w", err)
	}
	if cohort.SchemaVersion != "buildopt.poc/current-longitudinal-cohorts/v1" ||
		evidence.SchemaVersion != "buildopt.poc/current-longitudinal-raw/v1" ||
		len(cohort.Repositories) != 5 || len(evidence.Repositories) != 5 {
		return durablecatalog.PublicScreenReport{}, errors.New("public source evidence identity is invalid")
	}
	evidenceByKey := make(map[string]rawFamily, 5)
	evidenceSHAByKey := make(map[string]string, 5)
	for _, raw := range evidence.Repositories {
		var family rawFamily
		if err := json.Unmarshal(raw, &family); err != nil {
			return durablecatalog.PublicScreenReport{}, err
		}
		if family.Key == "" || evidenceByKey[family.Key].Key != "" {
			return durablecatalog.PublicScreenReport{}, errors.New("source family identity is invalid")
		}
		normalized, err := json.Marshal(family)
		if err != nil {
			return durablecatalog.PublicScreenReport{}, err
		}
		evidenceByKey[family.Key], evidenceSHAByKey[family.Key] = family, digest(normalized)
	}
	input := durablecatalog.PublicScreenInput{
		GeneratedAt: contract.GeneratedAt, CohortManifestSHA256: contract.CohortManifest.SHA256,
		DetectorOrder:  append([]string(nil), contract.DetectorOrder...),
		MinimumPassing: contract.MinimumPassingFamilies, MaximumBuildsToRepay: contract.MaximumBuildsToRepay,
	}
	for _, raw := range cohort.Repositories {
		var family cohortFamily
		if err := json.Unmarshal(raw, &family); err != nil {
			return durablecatalog.PublicScreenReport{}, err
		}
		evidenceFamily, ok := evidenceByKey[family.Policy.Key]
		if !ok || evidenceFamily.RepositoryID != family.Policy.RepositoryID {
			return durablecatalog.PublicScreenReport{}, fmt.Errorf("cohort family %s has no matching evidence", family.Policy.Key)
		}
		normalized, err := json.Marshal(family)
		if err != nil {
			return durablecatalog.PublicScreenReport{}, err
		}
		workflowRaw, err := json.Marshal(family.Policy.Workflow)
		if err != nil {
			return durablecatalog.PublicScreenReport{}, err
		}
		familyInput := durablecatalog.PublicFamilyInput{
			FamilyKey: family.Policy.Key, RevisionWindowSHA256: digest(normalized),
			RepositoryScopeSHA256:   digest([]byte("buildopt-public-repository-scope-v1\x00" + family.Policy.RepositoryID)),
			WorkflowArgumentsSHA256: digest(workflowRaw), OutputContract: append([]string(nil), family.Policy.RequiredOutputs...),
		}
		familyInput.Detectors = append(familyInput.Detectors, durablecatalog.PublicDetectorInput{
			DetectorID: durablecatalog.DetectorTaskContract, InputEvidenceSHA256: evidenceSHAByKey[family.Policy.Key],
			UnavailableReason: "INPUT_UNAVAILABLE_NO_GENERIC_SOURCE_PRODUCER",
		})
		familyInput.Detectors = append(familyInput.Detectors, graphDetectorInput(evidenceFamily, evidenceSHAByKey[family.Policy.Key], contract.Costs))
		input.Families = append(input.Families, familyInput)
	}
	return durablecatalog.ScreenPublicOpportunities(input)
}

func graphDetectorInput(family rawFamily, evidenceSHA string, costs costContract) durablecatalog.PublicDetectorInput {
	detector := durablecatalog.PublicDetectorInput{
		DetectorID: durablecatalog.DetectorGraphBreadth, InputEvidenceSHA256: evidenceSHA,
		TrialCostNs: costs.TrialCostNs, ValidationCostNs: costs.ValidationCostNs, PublicationCostNs: costs.PublicationCostNs,
	}
	graphRows, boundRows, criticalRows := 0, 0, 0
	for _, observation := range family.Observations {
		decision := observation.CandidateDecision
		if decision.GraphTotalProjects == 0 {
			continue
		}
		graphRows++
		if decision.PublicAction == nil || !validSHA(decision.PublicAction.CandidatePlanSHA256) || !validSHA(decision.PublicAction.BindingDigest) {
			continue
		}
		boundRows++
		if decision.PublicAction.ProjectedSavingNs == 0 {
			continue
		}
		criticalRows++
		raw, _ := json.Marshal(observation)
		detector.GraphObservations = append(detector.GraphObservations, durablecatalog.PublicGraphObservation{
			Ordinal: observation.Sequence, ObservationID: digest(raw),
			CandidatePlanSHA256:   decision.PublicAction.CandidatePlanSHA256,
			BindingDigest:         decision.PublicAction.BindingDigest,
			FullProjectCount:      decision.GraphTotalProjects,
			CandidateProjectCount: decision.GraphSelectedProjects,
			FullOutputSHA256:      observation.Control.OutputSHA256,
			CandidateOutputSHA256: observation.Candidate.OutputSHA256,
			ProjectedSavingNs:     decision.PublicAction.ProjectedSavingNs,
			ProductFailure:        !observation.ExactOutputs,
		})
	}
	switch {
	case graphRows == 0:
		detector.UnavailableReason = "GRAPH_PROPOSAL_INPUT_UNAVAILABLE"
	case boundRows == 0:
		detector.UnavailableReason = "EXACT_CANDIDATE_PLAN_INPUT_UNAVAILABLE"
	case criticalRows == 0:
		detector.UnavailableReason = "OMITTED_CRITICAL_PATH_INPUT_UNAVAILABLE"
	}
	return detector
}

func validateContract(contract gateContract) error {
	if contract.SchemaVersion != contractSchema || contract.WorkItem != "SWL-014C" || contract.GeneratedAt == "" ||
		contract.MinimumPassingFamilies != 3 || contract.MaximumBuildsToRepay != 30 ||
		len(contract.DetectorOrder) != 2 || contract.DetectorOrder[0] != durablecatalog.DetectorTaskContract ||
		contract.DetectorOrder[1] != durablecatalog.DetectorGraphBreadth ||
		!contract.Boundaries.ProofOfConcept || contract.Boundaries.ProductionAuthorized ||
		contract.Boundaries.RepositoryNameRules || contract.Boundaries.TaskNameRules || contract.Boundaries.PathExtensionRules ||
		contract.Boundaries.ManualProfiles || contract.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("opportunity gate contract is invalid")
	}
	return nil
}

func readBoundFile(root string, binding fileBinding) ([]byte, error) {
	clean := filepath.Clean(binding.Path)
	if binding.Path == "" || filepath.IsAbs(binding.Path) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || !validSHA(binding.SHA256) {
		return nil, errors.New("source binding is invalid")
	}
	raw, err := os.ReadFile(resolve(root, binding.Path))
	if err != nil {
		return nil, err
	}
	if digest(raw) != binding.SHA256 {
		return nil, fmt.Errorf("source binding drifted: %s", binding.Path)
	}
	return raw, nil
}

func readJSONStrict(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON has trailing content")
	}
	return nil
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func resolve(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func digest(raw []byte) string {
	value := sha256.Sum256(raw)
	return hex.EncodeToString(value[:])
}

func validSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}
