package profilediscovery

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

const (
	AnalysisSchemaVersion = "buildopt.poc/opportunity-analysis/v1"
	DecisionMeasure       = "MEASURE_STRUCTURAL_CANDIDATE"
)

// AnalysisOptions names the reviewable repository state used to find a
// structural optimization opportunity. Analysis does not consume performance
// evidence and therefore cannot qualify or activate a profile.
type AnalysisOptions struct {
	RepositoryRoot string
	ManifestPath   string
	GraphPath      string
	GeneratedPath  string
}

// AnalysisReport describes whether a complete Build Impact graph contains a
// smaller reviewed alternative. Additional mechanisms remain disabled until
// exact workload evidence independently qualifies them.
type AnalysisReport struct {
	SchemaVersion        string              `json:"schemaVersion"`
	Decision             string              `json:"decision"`
	Reason               string              `json:"reason"`
	Subject              Subject             `json:"subject"`
	Inputs               []InputBinding      `json:"inputs"`
	SourceBindings       SourceBindings      `json:"sourceBindings"`
	Plan                 *AnalysisPlan       `json:"plan"`
	Mechanisms           []AnalysisMechanism `json:"mechanisms"`
	MeasurementRequired  bool                `json:"measurementRequired"`
	ReviewRequired       bool                `json:"reviewRequired"`
	ActivationAutomatic  bool                `json:"activationAutomatic"`
	ProductionAuthorized bool                `json:"productionAuthorized"`
}

// AnalysisPlan quantifies the graph reduction without estimating wall-clock
// savings. Saved time must come from a direct optimized-native comparison.
type AnalysisPlan struct {
	AlternativeID        string   `json:"alternativeId"`
	Entrypoints          []string `json:"entrypoints"`
	FallbackEntrypoints  []string `json:"fallbackEntrypoints"`
	RequiredOutputs      []string `json:"requiredOutputs"`
	TotalProjectCount    int      `json:"totalProjectCount"`
	SelectedProjectCount int      `json:"selectedProjectCount"`
	OmittedProjectCount  int      `json:"omittedProjectCount"`
	OmittedProjectRatio  float64  `json:"omittedProjectRatio"`
}

// AnalysisMechanism distinguishes a proposed measurement from an enabled
// optimization. Analysis never emits an activation decision.
type AnalysisMechanism struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// AnalyzeOpportunity finds a repository-name-independent graph-reduction
// candidate. It is deliberately a measurement proposal rather than a profile
// discovery decision: structure can identify avoidable work but cannot prove
// that avoiding it beats optimized native Gradle.
func AnalyzeOpportunity(options AnalysisOptions) (AnalysisReport, error) {
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return AnalysisReport{}, fmt.Errorf("resolve repository root: %w", err)
	}
	report := nativeAnalysisReport("GRAPH_INVALID")

	manifestRaw, manifestInput, err := readInput(root, options.ManifestPath, "BUILD_IMPACT_MANIFEST")
	if err != nil {
		return AnalysisReport{}, err
	}
	report.Inputs = append(report.Inputs, manifestInput)
	repositoryID := manifestIdentity(manifestRaw, "repositoryId")
	pipelineClass := manifestIdentity(manifestRaw, "pipelineClass")
	manifest, err := buildimpact.ParseManifest(manifestRaw, repositoryID, pipelineClass)
	if err != nil {
		return report, nil
	}
	report.Subject.RepositoryID = repositoryID
	report.Subject.PipelineClass = pipelineClass
	report.SourceBindings.ManifestDigest = manifest.Digest

	graphRaw, graphInput, err := readInput(root, options.GraphPath, "BUILD_IMPACT_GRAPH")
	if err != nil {
		return AnalysisReport{}, err
	}
	report.Inputs = append(report.Inputs, graphInput)
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		return report, nil
	}
	report.SourceBindings.GraphDigest = graph.Digest
	if !graph.Graph.Complete {
		report.Reason = "GRAPH_INCOMPLETE"
		return report, nil
	}
	if graphHasUnknownRelationships(graph.Graph) {
		report.Reason = "GRAPH_UNKNOWN_RELATIONSHIP"
		return report, nil
	}
	if selectedAlternativeContainsTestTasks(manifest.Manifest, graph.Graph) {
		report.Reason = "UNSUPPORTED_TEST_TASK"
		return report, nil
	}

	generatedRaw, generatedInput, err := readInput(root, options.GeneratedPath, "GENERATED_MANIFEST")
	if err != nil {
		return AnalysisReport{}, err
	}
	report.Inputs = append(report.Inputs, generatedInput)
	generated, err := parseGeneratedManifest(generatedRaw)
	if err != nil {
		report.Reason = "GENERATED_STATE_INVALID"
		return report, nil
	}
	if !generated.Complete || len(generated.FallbackReasons) != 0 {
		report.Reason = "GENERATED_STATE_INCOMPLETE"
		return report, nil
	}
	if !generatedStateBound(generated, manifest, graph) {
		report.Reason = "GENERATED_STATE_DRIFT"
		return report, nil
	}
	report.Subject.GradleVersion = generated.GradleVersion
	report.SourceBindings.DiscoveryDigest = generated.DiscoveryDigest

	plan, reason := derivePlan(manifest.Manifest, graph.Graph)
	if reason != "" {
		report.Reason = reason
		return report, nil
	}
	if plan.OmittedProjectCount <= 0 {
		report.Reason = "NO_GRAPH_REDUCTION"
		return report, nil
	}
	selectedCount := len(graph.Graph.Projects) - plan.OmittedProjectCount
	report.Plan = &AnalysisPlan{
		AlternativeID:        plan.AlternativeID,
		Entrypoints:          plan.Entrypoints,
		FallbackEntrypoints:  plan.FallbackEntrypoints,
		RequiredOutputs:      plan.RequiredOutputs,
		TotalProjectCount:    len(graph.Graph.Projects),
		SelectedProjectCount: selectedCount,
		OmittedProjectCount:  plan.OmittedProjectCount,
		OmittedProjectRatio:  float64(plan.OmittedProjectCount) / float64(len(graph.Graph.Projects)),
	}
	report.Decision = DecisionMeasure
	report.Reason = "COMPLETE_STRUCTURAL_REDUCTION"
	report.Mechanisms = analysisMechanismDecisions(true)
	report.MeasurementRequired = true
	return report, nil
}

// AnalyzeGeneratedOpportunity evaluates already parsed discovery state before
// it is persisted. This lets onboarding remain transactional: rejected
// proposals write only their native decision and no candidate documents.
func AnalyzeGeneratedOpportunity(manifest buildimpact.LoadedManifest, graph buildimpact.LoadedGraph, generated buildimpact.GeneratedManifest) AnalysisReport {
	report := nativeAnalysisReport("GRAPH_INVALID")
	report.Subject.RepositoryID = manifest.Manifest.RepositoryID
	report.Subject.PipelineClass = manifest.Manifest.PipelineClass
	report.Subject.GradleVersion = generated.GradleVersion
	report.SourceBindings.ManifestDigest = manifest.Digest
	report.SourceBindings.GraphDigest = graph.Digest
	report.SourceBindings.DiscoveryDigest = generated.DiscoveryDigest
	if !graph.Graph.Complete {
		report.Reason = "GRAPH_INCOMPLETE"
		return report
	}
	if graphHasUnknownRelationships(graph.Graph) {
		report.Reason = "GRAPH_UNKNOWN_RELATIONSHIP"
		return report
	}
	if selectedAlternativeContainsTestTasks(manifest.Manifest, graph.Graph) {
		report.Reason = "UNSUPPORTED_TEST_TASK"
		return report
	}
	if !generated.Complete || len(generated.FallbackReasons) != 0 {
		report.Reason = "GENERATED_STATE_INCOMPLETE"
		return report
	}
	if !generatedStateBound(generated, manifest, graph) {
		report.Reason = "GENERATED_STATE_DRIFT"
		return report
	}
	plan, reason := derivePlan(manifest.Manifest, graph.Graph)
	if reason != "" {
		report.Reason = reason
		return report
	}
	if plan.OmittedProjectCount <= 0 {
		report.Reason = "NO_GRAPH_REDUCTION"
		return report
	}
	selectedCount := len(graph.Graph.Projects) - plan.OmittedProjectCount
	report.Plan = &AnalysisPlan{
		AlternativeID: plan.AlternativeID, Entrypoints: plan.Entrypoints,
		FallbackEntrypoints: plan.FallbackEntrypoints,
		RequiredOutputs: plan.RequiredOutputs, TotalProjectCount: len(graph.Graph.Projects),
		SelectedProjectCount: selectedCount, OmittedProjectCount: plan.OmittedProjectCount,
		OmittedProjectRatio: float64(plan.OmittedProjectCount) / float64(len(graph.Graph.Projects)),
	}
	report.Decision = DecisionMeasure
	report.Reason = "COMPLETE_STRUCTURAL_REDUCTION"
	report.Mechanisms = analysisMechanismDecisions(true)
	report.MeasurementRequired = true
	return report
}

// RenderAnalysis emits stable reviewable JSON.
func RenderAnalysis(report AnalysisReport) ([]byte, error) {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render opportunity analysis: %w", err)
	}
	return append(raw, '\n'), nil
}

func nativeAnalysisReport(reason string) AnalysisReport {
	return AnalysisReport{
		SchemaVersion:        AnalysisSchemaVersion,
		Decision:             DecisionNative,
		Reason:               reason,
		Mechanisms:           analysisMechanismDecisions(false),
		MeasurementRequired:  false,
		ReviewRequired:       true,
		ActivationAutomatic:  false,
		ProductionAuthorized: false,
	}
}

func analysisMechanismDecisions(buildImpact bool) []AnalysisMechanism {
	impactReason := "native fallback retains the mechanism disabled"
	impactStatus := "NOT_AUTHORIZED"
	if buildImpact {
		impactReason = "complete graph exposes a smaller reviewed alternative; direct timing is still required"
		impactStatus = "MEASURE_CANDIDATE"
	}
	return []AnalysisMechanism{
		{"BUILD_IMPACT", impactStatus, impactReason},
		{"SHARED_EDGE_CACHE", "NOT_AUTHORIZED", "requires independent locality and failure-fallback evidence for this workload"},
		{"HOT_STATE", "NOT_AUTHORIZED", "requires independent end-to-end value evidence for this workload"},
		{"RUNTIME_TUNING", "NOT_AUTHORIZED", "requires independent end-to-end value evidence for this workload"},
		{"SAFE_CACHE", "NOT_AUTHORIZED", "requires independent end-to-end value evidence for this workload"},
		{"STANDARD_COPY", "NOT_AUTHORIZED", "requires exact task-shape and end-to-end value evidence"},
		{"STANDARD_JAR", "NOT_AUTHORIZED", "requires exact task-shape and end-to-end value evidence"},
		{"TEST_OPTIMIZATION", "NOT_AUTHORIZED", "outside BuildOpt scope"},
	}
}
