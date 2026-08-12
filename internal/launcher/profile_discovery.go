package launcher

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const (
	profileDiscoveryUsage = "usage: buildopt profile discover --manifest PATH --graph PATH --generated-manifest PATH --matrix-summary PATH --cell-evidence PATH --profile-contract PATH\n"
	profileAnalysisUsage  = "usage: buildopt profile analyze --manifest PATH --graph PATH --generated-manifest PATH\n"
	profileQualifyUsage   = "usage: buildopt profile qualify --manifest PATH --graph PATH --generated-manifest PATH --evidence PATH\n"
	profileEvaluateUsage  = "usage: buildopt profile evaluate --manifest PATH --graph PATH --generated-manifest PATH [--evidence PATH --profile-output PATH]\n"
)

type profileEvaluation struct {
	SchemaVersion        string                              `json:"schemaVersion"`
	Decision             string                              `json:"decision"`
	Reason               string                              `json:"reason"`
	Analysis             profilediscovery.AnalysisReport     `json:"analysis"`
	Profile              *profilediscovery.StructuralProfile `json:"profile"`
	ProfileOutput        string                              `json:"profileOutput,omitempty"`
	ReviewRequired       bool                                `json:"reviewRequired"`
	ActivationAutomatic  bool                                `json:"activationAutomatic"`
	ProductionAuthorized bool                                `json:"productionAuthorized"`
}

func runProfileDiscovery(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "input" {
		return runProfileOwnerInput(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "outputs" {
		return runProfileOutputs(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "propose" {
		return runStructuralProfileProposal(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "analyze" {
		return runProfileAnalysis(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "qualify" {
		return runStructuralProfileQualification(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "evaluate" {
		return runStructuralProfileEvaluation(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "measure" {
		return runStructuralProfileMeasurement(args[1:], stdout, stderr)
	}
	if (len(args) == 1 && isHelp(args)) ||
		(len(args) == 2 && args[0] == "discover" && isHelp(args[1:])) {
		_, _ = io.WriteString(stdout, profileDiscoveryUsage)
		return 0
	}
	if len(args) == 0 || args[0] != "discover" {
		_, _ = io.WriteString(stderr, profileDiscoveryUsage)
		return exitUsage
	}
	flags := flag.NewFlagSet("buildopt profile discover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "checked Build Impact manifest")
	graph := flags.String("graph", "", "checked Build Impact graph")
	generated := flags.String("generated-manifest", "", "checked generated-state binding")
	matrix := flags.String("matrix-summary", "", "terminal installed-profile matrix")
	evidence := flags.String("cell-evidence", "", "one matrix cell evidence document")
	contract := flags.String("profile-contract", "", "reviewed POC profile contract")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *manifest == "" || *graph == "" || *generated == "" || *matrix == "" || *evidence == "" || *contract == "" {
		_, _ = io.WriteString(stderr, profileDiscoveryUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile discovery unavailable: %v\n", err)
		return exitConfiguration
	}
	report, err := profilediscovery.Discover(profilediscovery.Options{
		RepositoryRoot: repositoryRoot, ManifestPath: *manifest, GraphPath: *graph,
		GeneratedPath: *generated, MatrixSummaryPath: *matrix,
		CellEvidencePath: *evidence, ProfileContractPath: *contract,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile discovery unavailable: %v\n", err)
		return exitConfiguration
	}
	raw, err := profilediscovery.Render(report)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile discovery unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write profile discovery: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func runStructuralProfileEvaluation(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileEvaluateUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile evaluate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "checked Build Impact manifest")
	graph := flags.String("graph", "", "checked Build Impact graph")
	generated := flags.String("generated-manifest", "", "checked generated-state binding")
	evidence := flags.String("evidence", "", "installed-path structural evidence")
	profileOutput := flags.String("profile-output", "", "repository-relative qualified profile output")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *manifest == "" || *graph == "" || *generated == "" || (*evidence == "") != (*profileOutput == "") {
		_, _ = io.WriteString(stderr, profileEvaluateUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile evaluation unavailable: %v\n", err)
		return exitConfiguration
	}
	analysis, err := profilediscovery.AnalyzeOpportunity(profilediscovery.AnalysisOptions{
		RepositoryRoot: repositoryRoot,
		ManifestPath:   *manifest,
		GraphPath:      *graph,
		GeneratedPath:  *generated,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile evaluation unavailable: %v\n", err)
		return exitConfiguration
	}
	report := profileEvaluation{
		SchemaVersion:        "buildopt.poc/profile-evaluation/v1",
		Decision:             analysis.Decision,
		Reason:               analysis.Reason,
		Analysis:             analysis,
		ReviewRequired:       true,
		ActivationAutomatic:  false,
		ProductionAuthorized: false,
	}
	if analysis.Decision == profilediscovery.DecisionMeasure && *evidence != "" {
		profile, qualifyErr := profilediscovery.QualifyStructuralProfile(profilediscovery.StructuralOptions{
			RepositoryRoot: repositoryRoot,
			ManifestPath:   *manifest,
			GraphPath:      *graph,
			GeneratedPath:  *generated,
			EvidencePath:   *evidence,
		})
		if qualifyErr != nil {
			report.Decision = "NATIVE_FULL_GRAPH"
			report.Reason = "EVIDENCE_NOT_QUALIFIED"
		} else {
			if err := writeEvaluatedProfile(repositoryRoot, *profileOutput, profile); err != nil {
				_, _ = fmt.Fprintf(stderr, "buildopt: structural profile evaluation unavailable: %v\n", err)
				return exitConfiguration
			}
			report.Decision = "QUALIFY_STRUCTURAL_PROFILE"
			report.Reason = "EXACT_EVIDENCE_QUALIFIED"
			report.Profile = &profile
			report.ProfileOutput = *profileOutput
		}
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile evaluation unavailable: %v\n", err)
		return exitConfiguration
	}
	raw = append(raw, '\n')
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write structural profile evaluation: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func writeEvaluatedProfile(repositoryRoot, relativePath string, profile profilediscovery.StructuralProfile) error {
	raw, err := profilediscovery.RenderStructuralProfile(profile)
	if err != nil {
		return err
	}
	return writeRepositoryDocument(repositoryRoot, relativePath, raw, 0o644)
}

func runStructuralProfileQualification(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileQualifyUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile qualify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "checked Build Impact manifest")
	graph := flags.String("graph", "", "checked Build Impact graph")
	generated := flags.String("generated-manifest", "", "checked generated-state binding")
	evidence := flags.String("evidence", "", "qualified installed-path structural evidence")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *manifest == "" || *graph == "" || *generated == "" || *evidence == "" {
		_, _ = io.WriteString(stderr, profileQualifyUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile qualification unavailable: %v\n", err)
		return exitConfiguration
	}
	profile, err := profilediscovery.QualifyStructuralProfile(profilediscovery.StructuralOptions{
		RepositoryRoot: repositoryRoot,
		ManifestPath:   *manifest,
		GraphPath:      *graph,
		GeneratedPath:  *generated,
		EvidencePath:   *evidence,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile qualification unavailable: %v\n", err)
		return exitConfiguration
	}
	raw, err := profilediscovery.RenderStructuralProfile(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: structural profile qualification unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write structural profile: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func runProfileAnalysis(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileAnalysisUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile analyze", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "checked Build Impact manifest")
	graph := flags.String("graph", "", "checked Build Impact graph")
	generated := flags.String("generated-manifest", "", "checked generated-state binding")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *manifest == "" || *graph == "" || *generated == "" {
		_, _ = io.WriteString(stderr, profileAnalysisUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile analysis unavailable: %v\n", err)
		return exitConfiguration
	}
	report, err := profilediscovery.AnalyzeOpportunity(profilediscovery.AnalysisOptions{
		RepositoryRoot: repositoryRoot,
		ManifestPath:   *manifest,
		GraphPath:      *graph,
		GeneratedPath:  *generated,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile analysis unavailable: %v\n", err)
		return exitConfiguration
	}
	raw, err := profilediscovery.RenderAnalysis(report)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile analysis unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write profile analysis: %v\n", err)
		return exitConfiguration
	}
	return 0
}
