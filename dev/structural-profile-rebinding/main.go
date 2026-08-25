// Command structural-profile-rebinding generates and verifies the bounded
// cross-revision evidence for the structural profile binding POC.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tonyredondo/buildopt/internal/structuralbinding"
)

const schemaVersion = "buildopt.evidence/poc-structural-profile-rebinding/v1"

type compatibilityCase struct {
	Name           string `json:"name"`
	EvidenceCommit string `json:"evidenceCommit"`
	CurrentCommit  string `json:"currentCommit"`
	CheckoutRoot   string `json:"checkoutRoot"`
	BindingSHA256  string `json:"bindingSha256"`
	Compatible     bool   `json:"compatible"`
}

type rejectionCase struct {
	Name     string `json:"name"`
	Rejected bool   `json:"rejected"`
}

type boundaries struct {
	ProofOfConcept             bool   `json:"proofOfConcept"`
	ProductionAuthorized       bool   `json:"productionAuthorized"`
	RevisionIsCompatibilityKey bool   `json:"revisionIsCompatibilityKey"`
	CheckoutPathIsIdentity     bool   `json:"checkoutPathIsIdentity"`
	ExactOutputsRevisionBound  bool   `json:"exactOutputsRevisionBound"`
	SoakRequired               bool   `json:"soakRequired"`
	DesignPartnerRequired      bool   `json:"designPartnerRequired"`
	TestOptimization           string `json:"testOptimization"`
}

type evidence struct {
	SchemaVersion          string                    `json:"schemaVersion"`
	WorkItem               string                    `json:"workItem"`
	ImplementationRevision string                    `json:"implementationRevision"`
	Binding                structuralbinding.Binding `json:"binding"`
	CompatibleCases        []compatibilityCase       `json:"compatibleCases"`
	DriftCases             []rejectionCase           `json:"driftCases"`
	IncompleteCases        []rejectionCase           `json:"incompleteCases"`
	Decision               string                    `json:"decision"`
	ProductFailures        int                       `json:"productFailures"`
	PerformanceReplayRun   bool                      `json:"performanceReplayRun"`
	Boundaries             boundaries                `json:"boundaries"`
}

func main() {
	implementation := flag.String("implementation-revision", "", "40-character implementation commit")
	output := flag.String("output", "", "output JSON path")
	check := flag.String("check", "", "verify a committed evidence JSON")
	flag.Parse()
	if flag.NArg() != 0 || (*check == "" && (*implementation == "" || *output == "")) ||
		(*check != "" && (*implementation != "" || *output != "")) {
		fmt.Fprintln(os.Stderr, "usage: structural-profile-rebinding --implementation-revision SHA --output PATH | --check PATH")
		os.Exit(64)
	}
	if *check != "" {
		if err := verify(*check); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := write(*implementation, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var observed evidence
	if err := decoder.Decode(&observed); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return errors.New("evidence contains trailing JSON")
	}
	expected, err := generate(observed.ImplementationRevision)
	if err != nil {
		return err
	}
	expectedRaw, _ := json.MarshalIndent(expected, "", "  ")
	expectedRaw = append(expectedRaw, '\n')
	if !bytes.Equal(raw, expectedRaw) {
		return errors.New("evidence does not match the canonical structural binding evaluation")
	}
	return nil
}

func write(implementation, path string) error {
	value, err := generate(implementation)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func generate(implementation string) (evidence, error) {
	if len(implementation) != 40 || strings.Trim(implementation, "0123456789abcdef") != "" {
		return evidence{}, errors.New("implementation revision must be one lowercase 40-character Git SHA")
	}
	baseInput := fixtureInput()
	base, err := structuralbinding.Derive(baseInput)
	if err != nil {
		return evidence{}, err
	}
	compatible := []compatibilityCase{
		{Name: "next source commit", EvidenceCommit: strings.Repeat("1", 40), CurrentCommit: strings.Repeat("2", 40), CheckoutRoot: "/runner/work/repository-a", BindingSHA256: base.SHA256, Compatible: true},
		{Name: "independent checkout", EvidenceCommit: strings.Repeat("1", 40), CurrentCommit: strings.Repeat("3", 40), CheckoutRoot: "D:/agent/work/repository-b", BindingSHA256: base.SHA256, Compatible: true},
	}
	drifts := []struct {
		name   string
		mutate func(*structuralbinding.Input)
	}{
		{"Wrapper", func(value *structuralbinding.Input) { value.WrapperSHA256 = strings.Repeat("b", 64) }},
		{"workflow", func(value *structuralbinding.Input) { value.GradleOptions = []string{"--parallel"} }},
		{"producer lineage", func(value *structuralbinding.Input) { value.Tasks[2].DependsOn = []string{":stable:emit"} }},
		{"output contract", func(value *structuralbinding.Input) { value.CandidateOutputs = []string{"stable/output"} }},
		{"change family", func(value *structuralbinding.Input) { value.ChangeFamily = "RESOURCE" }},
	}
	driftCases := make([]rejectionCase, 0, len(drifts))
	for _, candidate := range drifts {
		input := fixtureInput()
		candidate.mutate(&input)
		binding, bindingErr := structuralbinding.Derive(input)
		driftCases = append(driftCases, rejectionCase{
			Name: candidate.name, Rejected: bindingErr != nil || binding.SHA256 != base.SHA256,
		})
	}
	incomplete := []struct {
		name   string
		mutate func(*structuralbinding.Input)
	}{
		{"missing task dependency", func(value *structuralbinding.Input) { value.Tasks[2].DependsOn = []string{":missing"} }},
		{"ambiguous output owner", func(value *structuralbinding.Input) { value.Outputs[0].OwnerProjects = []string{":changed", ":stable"} }},
		{"missing output evidence", func(value *structuralbinding.Input) { value.Outputs = value.Outputs[:1] }},
		{"cyclic producer lineage", func(value *structuralbinding.Input) { value.Tasks[0].DependsOn = []string{":changed:emit"} }},
	}
	incompleteCases := make([]rejectionCase, 0, len(incomplete))
	for _, candidate := range incomplete {
		input := fixtureInput()
		candidate.mutate(&input)
		_, bindingErr := structuralbinding.Derive(input)
		incompleteCases = append(incompleteCases, rejectionCase{Name: candidate.name, Rejected: bindingErr != nil})
	}
	for _, candidate := range driftCases {
		if !candidate.Rejected {
			return evidence{}, errors.New("structural drift was not rejected")
		}
	}
	for _, candidate := range incompleteCases {
		if !candidate.Rejected {
			return evidence{}, errors.New("incomplete structural evidence was not rejected")
		}
	}
	return evidence{
		SchemaVersion: schemaVersion, WorkItem: "POC-STRUCTURAL-PROFILE-REBINDING-001",
		ImplementationRevision: implementation, Binding: base,
		CompatibleCases: compatible, DriftCases: driftCases, IncompleteCases: incompleteCases,
		Decision: "ALLOW_STRUCTURALLY_COMPATIBLE_PROFILE_REBINDING", ProductFailures: 0,
		PerformanceReplayRun: false,
		Boundaries: boundaries{
			ProofOfConcept: true, ProductionAuthorized: false,
			RevisionIsCompatibilityKey: false, CheckoutPathIsIdentity: false,
			ExactOutputsRevisionBound: true, SoakRequired: false, DesignPartnerRequired: false,
			TestOptimization: "OUT_OF_SCOPE",
		},
	}, nil
}

func fixtureInput() structuralbinding.Input {
	return structuralbinding.Input{
		RepositoryID: "example/repository", WrapperSHA256: strings.Repeat("a", 64),
		OriginalEntrypoints: []string{":changed:bundleAll"}, CandidateEntrypoints: []string{":changed:emit"},
		GradleOptions: []string{"--no-daemon"}, RequiredOutputs: []string{"changed/output", "stable/output"},
		CandidateOutputs: []string{"changed/output"}, ChangeFamily: "LEAF_SOURCE", ChangedProjects: []string{":changed"},
		Tasks: []structuralbinding.Task{
			{Path: ":common:prepare", ProjectPath: ":common"},
			{Path: ":stable:emit", ProjectPath: ":stable", DependsOn: []string{":common:prepare"}},
			{Path: ":changed:emit", ProjectPath: ":changed", DependsOn: []string{":common:prepare"}},
		},
		Outputs: []structuralbinding.Output{
			{Pattern: "changed/output", Kind: "FILE", OwnerProjects: []string{":changed"}, ProducerTasks: []string{":changed:emit"}},
			{Pattern: "stable/output", Kind: "FILE", OwnerProjects: []string{":stable"}, ProducerTasks: []string{":stable:emit"}},
		},
	}
}
