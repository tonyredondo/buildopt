// Package profilediscovery derives reviewable POC profiles from checked evidence.
package profilediscovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

const (
	SchemaVersion     = "buildopt.poc/profile-discovery/v1"
	DecisionProfile   = "GENERATED_QUALIFIED_PROFILE"
	DecisionNative    = "NATIVE_FULL_GRAPH"
	profileSchema     = "buildopt.poc/qualified-profile/v2"
	profileOwnership  = "REPOSITORY_COMMITTED"
	profileClaimScope = "DECLARED_OUTPUTS_ONLY"
	profileFallback   = "NATIVE_FULL_GRAPH"
	edgeMode          = "READ_ONLY_LOOPBACK"
	maximumInputBytes = 4 << 20
	matrixSchema      = "buildopt.evidence/poc-qualified-profile-matrix/v1"
)

// Options names the checked inputs used for one read-only discovery decision.
type Options struct {
	RepositoryRoot      string
	ManifestPath        string
	GraphPath           string
	GeneratedPath       string
	MatrixSummaryPath   string
	CellEvidencePath    string
	ProfileContractPath string
}

// Report is deterministic discovery evidence. Profile remains nil whenever
// uncertainty requires native Gradle; discovery never activates the profile.
type Report struct {
	SchemaVersion        string              `json:"schemaVersion"`
	Decision             string              `json:"decision"`
	Reason               string              `json:"reason"`
	Subject              Subject             `json:"subject"`
	Inputs               []InputBinding      `json:"inputs"`
	SourceBindings       SourceBindings      `json:"sourceBindings"`
	Plan                 *Plan               `json:"plan,omitempty"`
	Mechanisms           []MechanismDecision `json:"mechanisms"`
	Profile              *Profile            `json:"profile"`
	ReviewRequired       bool                `json:"reviewRequired"`
	ActivationAutomatic  bool                `json:"activationAutomatic"`
	ProductionAuthorized bool                `json:"productionAuthorized"`
}

type Subject struct {
	RepositoryID       string `json:"repositoryId,omitempty"`
	RepositoryRevision string `json:"repositoryRevision,omitempty"`
	PipelineClass      string `json:"pipelineClass,omitempty"`
	GradleVersion      string `json:"gradleVersion,omitempty"`
	JDK                string `json:"jdk,omitempty"`
}

type InputBinding struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type SourceBindings struct {
	EvidenceRevision  string            `json:"evidenceRevision,omitempty"`
	ComponentEvidence map[string]string `json:"componentEvidence,omitempty"`
	ManifestDigest    string            `json:"manifestDigest,omitempty"`
	GraphDigest       string            `json:"graphDigest,omitempty"`
	DiscoveryDigest   string            `json:"discoveryDigest,omitempty"`
	NormalizedInput   string            `json:"normalizedInputSha256,omitempty"`
	RequiredOutput    string            `json:"requiredOutputSha256,omitempty"`
}

type Plan struct {
	AlternativeID       string   `json:"alternativeId"`
	Entrypoints         []string `json:"entrypoints"`
	FallbackEntrypoints []string `json:"fallbackEntrypoints"`
	RequiredOutputs     []string `json:"requiredOutputs"`
	OmittedProjectCount int      `json:"omittedProjectCount"`
}

type MechanismDecision struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type Profile struct {
	SchemaVersion  string            `json:"schemaVersion"`
	ProfileVersion uint64            `json:"profileVersion"`
	ProfileID      string            `json:"profileId"`
	Ownership      string            `json:"ownership"`
	ClaimScope     string            `json:"claimScope"`
	RepositoryID   string            `json:"repositoryId"`
	PipelineClass  string            `json:"pipelineClass"`
	Fallback       string            `json:"fallback"`
	Impact         ProfileImpact     `json:"impact"`
	Mechanisms     ProfileMechanisms `json:"mechanisms"`
	GradleOptions  []string          `json:"gradleOptions"`
	Preconditions  []Precondition    `json:"preconditions"`
	EdgeCache      EdgeCache         `json:"edgeCache"`
}

type ProfileImpact struct {
	Manifest          string `json:"manifest"`
	Graph             string `json:"graph"`
	GeneratedManifest string `json:"generatedManifest"`
}

type ProfileMechanisms struct {
	BuildImpact         bool `json:"buildImpact"`
	StandardJarAdapter  bool `json:"standardJarAdapter"`
	SafeCache           bool `json:"safeCache"`
	RuntimeTuning       bool `json:"runtimeTuning"`
	HotState            bool `json:"hotState"`
	StandardCopyAdapter bool `json:"standardCopyAdapter"`
	SharedEdgeCache     bool `json:"sharedEdgeCache"`
}

type Precondition struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type EdgeCache struct {
	Mode string `json:"mode"`
}

type matrixSummary struct {
	SchemaVersion    string       `json:"schemaVersion"`
	EvidenceRevision string       `json:"evidenceRevision"`
	Cells            []matrixCell `json:"cells"`
	Decision         string       `json:"decision"`
}

type matrixCell struct {
	ID                string    `json:"id"`
	EvidenceFile      string    `json:"evidenceFile"`
	EvidenceSHA256    string    `json:"evidenceSha256"`
	Mechanisms        []string  `json:"mechanisms"`
	Pairs             int       `json:"pairs"`
	ControlMeanMS     *float64  `json:"controlMeanMs"`
	CandidateMeanMS   *float64  `json:"candidateMeanMs"`
	MeanSavedMS       *float64  `json:"meanSavedMs"`
	ReductionRatio    *float64  `json:"reductionRatio"`
	Interval95SavedMS []float64 `json:"interval95SavedMs"`
	PositivePairs     int       `json:"positivePairs"`
	Qualified         bool      `json:"qualified"`
}

type cellEvidence struct {
	SchemaVersion         string                     `json:"schemaVersion"`
	EvidenceState         string                     `json:"evidenceState"`
	Repository            evidenceRepository         `json:"repository"`
	Runner                evidenceRunner             `json:"runner"`
	Result                evidenceResult             `json:"result"`
	Mechanisms            evidenceMechanisms         `json:"mechanisms"`
	Safety                evidenceSafety             `json:"safety"`
	Boundaries            evidenceBoundaries         `json:"boundaries"`
	InputQualification    evidenceInputQualification `json:"inputQualification"`
	ExecutionPath         evidenceExecutionPath      `json:"executionPath"`
	MeasurementAssertions evidenceAssertions         `json:"measurementAssertions"`
	ComponentEvidence     map[string]string          `json:"componentEvidence"`
}

type evidenceRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
	Revision      string `json:"revision"`
	GradleVersion string `json:"gradleVersion"`
	JDK           string `json:"jdk"`
}

type evidenceRunner struct {
	MaxWorkers int `json:"maxWorkers"`
}

type evidenceResult struct {
	Pairs                 int       `json:"pairs"`
	ControlMeanMS         *float64  `json:"controlMeanMs"`
	CandidateMeanMS       *float64  `json:"candidateMeanMs"`
	MeanSavedMS           *float64  `json:"meanSavedMs"`
	ReductionRatio        *float64  `json:"reductionRatio"`
	Interval95SavedMS     []float64 `json:"interval95SavedMs"`
	PositivePairs         int       `json:"positivePairs"`
	Qualified             bool      `json:"qualified"`
	PerformanceGatePassed bool      `json:"performanceGatePassed"`
}

type evidenceMechanisms struct {
	BuildImpact         bool `json:"buildImpact"`
	EdgeLocality        bool `json:"edgeLocality"`
	HotState            bool `json:"hotState"`
	RuntimeTuning       bool `json:"runtimeTuning"`
	SafeCache           bool `json:"safeCache"`
	StandardCopy        bool `json:"standardCopyAdapter"`
	StandardJar         bool `json:"standardJarAdapter"`
	TestOptimization    bool `json:"testOptimization"`
	SourceNormalization bool `json:"sourceNormalization"`
}

type evidenceSafety struct {
	ImpactFallbackPassed              bool `json:"impactFallbackPassed"`
	EdgeFailureFallbackPassed         bool `json:"edgeFailureFallbackPassed"`
	EdgeFailureBuildSucceeded         bool `json:"edgeFailureBuildSucceeded"`
	EdgeFailureRemoteCacheDisabled    bool `json:"edgeFailureRemoteCacheDisabled"`
	EdgeFailureRequiredOutputProduced bool `json:"edgeFailureRequiredOutputProduced"`
	EdgeFailureOutputExact            bool `json:"edgeFailureOutputExact"`
}

type evidenceBoundaries struct {
	SameSharedOrigin              bool `json:"sameSharedOrigin"`
	SameCommittedObjectBytes      bool `json:"sameCommittedObjectBytes"`
	SameNormalizedSourceAndChange bool `json:"sameNormalizedSourceAndChange"`
	SameRequiredOutput            bool `json:"sameRequiredOutput"`
	TestTasksForbidden            bool `json:"testTasksForbidden"`
	TestOptimizationModified      bool `json:"testOptimizationModified"`
	ProductionReadinessClaimed    bool `json:"productionReadinessClaimed"`
}

type evidenceInputQualification struct {
	EvidenceSHA256            string `json:"evidenceSha256"`
	NormalizedBuildFileSHA256 string `json:"normalizedBuildFileSha256"`
	NormalizedArtifactSHA256  string `json:"normalizedArtifactSha256"`
	AppliedBeforePreparation  bool   `json:"appliedBeforeDependencyPreparation"`
	IdenticalInBothArms       bool   `json:"identicalInSeedControlAndCandidate"`
}

type evidenceExecutionPath struct {
	InstalledProfile                   bool   `json:"installedProfile"`
	ProfileID                          string `json:"profileId"`
	RepositoryOwnedProfile             bool   `json:"repositoryOwnedProfile"`
	ExperimentOnlySelectionEnvironment bool   `json:"experimentOnlySelectionEnvironmentUsed"`
}

type evidenceAssertions struct {
	SameCachedOutputEveryArm       bool `json:"sameCachedOutputInEveryMeasuredArm"`
	CandidatePlanSelectedEveryPair bool `json:"candidatePlanSelectedInEveryPair"`
	ControlFullGraphEveryPair      bool `json:"controlFullGraphInEveryPair"`
	StandardJarAbsent              bool `json:"standardJarAdapterAbsent"`
	ProductFailures                int  `json:"productAttributableFailures"`
	RepositoryProfileEveryPair     bool `json:"repositoryOwnedProfileSelectedInEveryPair"`
}

type profileContract struct {
	Profile struct {
		SchemaVersion        string `json:"schemaVersion"`
		ProfileID            string `json:"profileId"`
		Ownership            string `json:"ownership"`
		ClaimScope           string `json:"claimScope"`
		ProductionAuthorized bool   `json:"productionAuthorized"`
	} `json:"profile"`
	Activation struct {
		CandidateMechanisms []string `json:"candidateMechanisms"`
		DisabledMechanisms  []string `json:"disabledMechanisms"`
		EdgePush            bool     `json:"edgePush"`
	} `json:"activation"`
	Preconditions []Precondition `json:"preconditions"`
}

// Discover validates the checked qualification chain and derives a profile
// without writing repository state or granting activation authority.
func Discover(options Options) (Report, error) {
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve repository root: %w", err)
	}
	report := nativeReport("EVIDENCE_NOT_QUALIFIED")
	summaryRaw, summaryInput, err := readInput(root, options.MatrixSummaryPath, "QUALIFICATION_MATRIX")
	if err != nil {
		return Report{}, err
	}
	evidenceRaw, evidenceInput, err := readInput(root, options.CellEvidencePath, "CELL_EVIDENCE")
	if err != nil {
		return Report{}, err
	}
	report.Inputs = append(report.Inputs, summaryInput, evidenceInput)
	var summary matrixSummary
	if err := decodeJSON(summaryRaw, &summary); err != nil {
		return Report{}, fmt.Errorf("decode matrix summary: %w", err)
	}
	var evidence cellEvidence
	if err := decodeJSON(evidenceRaw, &evidence); err != nil {
		return Report{}, fmt.Errorf("decode cell evidence: %w", err)
	}
	report.Subject = Subject{
		RepositoryID:       evidence.Repository.NameWithOwner,
		RepositoryRevision: evidence.Repository.Revision,
		GradleVersion:      evidence.Repository.GradleVersion,
		JDK:                evidence.Repository.JDK,
	}
	report.SourceBindings.EvidenceRevision = summary.EvidenceRevision
	report.SourceBindings.ComponentEvidence = sortedDigestMap(evidence.ComponentEvidence)
	if !validRevision(summary.EvidenceRevision) || !validRevision(evidence.Repository.Revision) {
		report.Reason = "EVIDENCE_REVISION_INVALID"
		return report, nil
	}
	cell, ok := matchingCell(summary, evidenceInput.SHA256)
	if summary.SchemaVersion != matrixSchema || summary.Decision != "SPECIALIZE_QUALIFIED_PROFILES" || !ok {
		report.Reason = "EVIDENCE_DRIFT"
		return report, nil
	}
	if !cell.Qualified || !evidence.Result.Qualified {
		return report, nil
	}
	if evidence.Repository.GradleVersion == "" || evidence.Repository.JDK == "" {
		report.Reason = "EVIDENCE_RUNTIME_INVALID"
		return report, nil
	}
	if err := validateQualification(cell, evidence); err != nil {
		report.Reason = err.Error()
		return report, nil
	}

	manifestRaw, manifestInput, err := readInput(root, options.ManifestPath, "BUILD_IMPACT_MANIFEST")
	if err != nil {
		return Report{}, err
	}
	manifest, err := buildimpact.ParseManifest(manifestRaw, evidence.Repository.NameWithOwner, manifestIdentity(manifestRaw, "pipelineClass"))
	if err != nil {
		report.Inputs = append(report.Inputs, manifestInput)
		report.Reason = "MANIFEST_INVALID"
		return report, nil
	}
	report.Subject.PipelineClass = manifest.Manifest.PipelineClass
	report.SourceBindings.ManifestDigest = manifest.Digest
	report.Inputs = append(report.Inputs, manifestInput)

	graphRaw, graphInput, err := readInput(root, options.GraphPath, "BUILD_IMPACT_GRAPH")
	if err != nil {
		return Report{}, err
	}
	report.Inputs = append(report.Inputs, graphInput)
	graph, err := buildimpact.ParseDeclaredGraph(graphRaw, manifest)
	if err != nil {
		report.Reason = "GRAPH_INVALID"
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
		return Report{}, err
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
	report.SourceBindings.DiscoveryDigest = generated.DiscoveryDigest

	contractRaw, contractInput, err := readInput(root, options.ProfileContractPath, "PROFILE_CONTRACT")
	if err != nil {
		return Report{}, err
	}
	report.Inputs = append(report.Inputs, contractInput)
	var contract profileContract
	if err := decodeJSON(contractRaw, &contract); err != nil {
		report.Reason = "PROFILE_CONTRACT_INVALID"
		return report, nil
	}
	if err := validateContract(contract, evidence); err != nil {
		report.Reason = err.Error()
		return report, nil
	}

	plan, reason := derivePlan(manifest.Manifest, graph.Graph)
	if reason != "" {
		report.Reason = reason
		return report, nil
	}
	precondition := contract.Preconditions[0]
	report.SourceBindings.NormalizedInput = precondition.SHA256
	report.SourceBindings.RequiredOutput = evidence.InputQualification.NormalizedArtifactSHA256
	report.Plan = &plan
	report.Mechanisms = mechanismDecisions()
	report.Profile = &Profile{
		SchemaVersion:  profileSchema,
		ProfileVersion: 2,
		ProfileID:      contract.Profile.ProfileID,
		Ownership:      profileOwnership,
		ClaimScope:     profileClaimScope,
		RepositoryID:   manifest.Manifest.RepositoryID,
		PipelineClass:  manifest.Manifest.PipelineClass,
		Fallback:       profileFallback,
		Impact: ProfileImpact{
			Manifest:          filepath.Base(options.ManifestPath),
			Graph:             filepath.Base(options.GraphPath),
			GeneratedManifest: filepath.Base(options.GeneratedPath),
		},
		Mechanisms:    ProfileMechanisms{BuildImpact: true, SharedEdgeCache: true},
		GradleOptions: []string{"--daemon", "--build-cache", "--no-configuration-cache", "--parallel", "--console=plain", fmt.Sprintf("--max-workers=%d", evidence.Runner.MaxWorkers), "--no-scan"},
		Preconditions: []Precondition{precondition},
		EdgeCache:     EdgeCache{Mode: edgeMode},
	}
	report.Decision = DecisionProfile
	report.Reason = "QUALIFIED_EVIDENCE_BOUND"
	return report, nil
}

// Render emits stable reviewable JSON suitable for byte-for-byte comparison.
func Render(report Report) ([]byte, error) {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render profile discovery: %w", err)
	}
	return append(raw, '\n'), nil
}

func nativeReport(reason string) Report {
	return Report{
		SchemaVersion: SchemaVersion, Decision: DecisionNative, Reason: reason,
		Mechanisms: nativeMechanismDecisions(), ReviewRequired: true,
		ActivationAutomatic: false, ProductionAuthorized: false,
	}
}

func matchingCell(summary matrixSummary, evidenceSHA string) (matrixCell, bool) {
	for _, cell := range summary.Cells {
		if "sha256:"+cell.EvidenceSHA256 == evidenceSHA {
			return cell, true
		}
	}
	return matrixCell{}, false
}

func validateQualification(cell matrixCell, evidence cellEvidence) error {
	if evidence.SchemaVersion == "" || evidence.EvidenceState != "TERMINAL" {
		return errors.New("EVIDENCE_INCOMPLETE")
	}
	if cell.Pairs != evidence.Result.Pairs || cell.PositivePairs != evidence.Result.PositivePairs || cell.Pairs < 4 || cell.PositivePairs != cell.Pairs {
		return errors.New("EVIDENCE_GATE_FAILED")
	}
	if !sameOptionalFloat(cell.ControlMeanMS, evidence.Result.ControlMeanMS) ||
		!sameOptionalFloat(cell.CandidateMeanMS, evidence.Result.CandidateMeanMS) ||
		!sameOptionalFloat(cell.MeanSavedMS, evidence.Result.MeanSavedMS) ||
		!sameOptionalFloat(cell.ReductionRatio, evidence.Result.ReductionRatio) ||
		!sameFloatSlice(cell.Interval95SavedMS, evidence.Result.Interval95SavedMS) ||
		len(evidence.Result.Interval95SavedMS) != 2 || evidence.Result.Interval95SavedMS[0] <= 0 {
		return errors.New("EVIDENCE_DRIFT")
	}
	if evidence.Result.MeanSavedMS == nil || *evidence.Result.MeanSavedMS < 500 || evidence.Result.ReductionRatio == nil || *evidence.Result.ReductionRatio < .02 || !evidence.Result.PerformanceGatePassed {
		return errors.New("EVIDENCE_GATE_FAILED")
	}
	if strings.Join(cell.Mechanisms, ",") != "BUILD_IMPACT,SHARED_EDGE_CACHE" {
		return errors.New("MECHANISM_SET_UNSUPPORTED")
	}
	m := evidence.Mechanisms
	if !m.BuildImpact || !m.EdgeLocality || !m.SourceNormalization || m.HotState || m.RuntimeTuning || m.SafeCache || m.StandardCopy || m.StandardJar || m.TestOptimization {
		return errors.New("MECHANISM_SET_UNSUPPORTED")
	}
	s := evidence.Safety
	if !s.ImpactFallbackPassed || !s.EdgeFailureFallbackPassed || !s.EdgeFailureBuildSucceeded || !s.EdgeFailureRemoteCacheDisabled || !s.EdgeFailureRequiredOutputProduced || !s.EdgeFailureOutputExact {
		return errors.New("SAFETY_EVIDENCE_INCOMPLETE")
	}
	b := evidence.Boundaries
	if !b.SameSharedOrigin || !b.SameCommittedObjectBytes || !b.SameNormalizedSourceAndChange || !b.SameRequiredOutput || !b.TestTasksForbidden || b.TestOptimizationModified || b.ProductionReadinessClaimed {
		return errors.New("BOUNDARY_EVIDENCE_INCOMPLETE")
	}
	i := evidence.InputQualification
	if !validSHA(i.EvidenceSHA256) || !validSHA(i.NormalizedBuildFileSHA256) || !validSHA(i.NormalizedArtifactSHA256) || !i.AppliedBeforePreparation || !i.IdenticalInBothArms {
		return errors.New("INPUT_QUALIFICATION_INCOMPLETE")
	}
	a := evidence.MeasurementAssertions
	if !a.SameCachedOutputEveryArm || !a.CandidatePlanSelectedEveryPair || !a.ControlFullGraphEveryPair || !a.StandardJarAbsent || a.ProductFailures != 0 || !a.RepositoryProfileEveryPair {
		return errors.New("MEASUREMENT_ASSERTION_FAILED")
	}
	if len(evidence.ComponentEvidence) == 0 {
		return errors.New("TRACE_EVIDENCE_INCOMPLETE")
	}
	for _, digest := range evidence.ComponentEvidence {
		if !validSHA(digest) {
			return errors.New("TRACE_EVIDENCE_INCOMPLETE")
		}
	}
	if evidence.Runner.MaxWorkers < 1 || evidence.Runner.MaxWorkers > 256 || !evidence.ExecutionPath.InstalledProfile || !evidence.ExecutionPath.RepositoryOwnedProfile || evidence.ExecutionPath.ExperimentOnlySelectionEnvironment || evidence.ExecutionPath.ProfileID == "" {
		return errors.New("EXECUTION_EVIDENCE_INCOMPLETE")
	}
	return nil
}

func validateContract(contract profileContract, evidence cellEvidence) error {
	if contract.Profile.SchemaVersion != profileSchema || contract.Profile.ProfileID != evidence.ExecutionPath.ProfileID || contract.Profile.Ownership != profileOwnership || contract.Profile.ClaimScope != profileClaimScope || contract.Profile.ProductionAuthorized {
		return errors.New("PROFILE_CONTRACT_INVALID")
	}
	if strings.Join(contract.Activation.CandidateMechanisms, ",") != "BUILD_IMPACT,READ_ONLY_EDGE" || contract.Activation.EdgePush {
		return errors.New("PROFILE_CONTRACT_INVALID")
	}
	expectedDisabled := "HOT_STATE,RUNTIME_TUNING,SAFE_CACHE,STANDARD_COPY,STANDARD_JAR,TEST_OPTIMIZATION"
	if strings.Join(contract.Activation.DisabledMechanisms, ",") != expectedDisabled || len(contract.Preconditions) != 1 {
		return errors.New("PROFILE_CONTRACT_INVALID")
	}
	p := contract.Preconditions[0]
	if p.Type != "FILE_SHA256" || p.Path == "" || filepath.IsAbs(p.Path) || filepath.Clean(p.Path) != p.Path || p.SHA256 != evidence.InputQualification.NormalizedBuildFileSHA256 {
		return errors.New("PROFILE_PRECONDITION_DRIFT")
	}
	return nil
}

func derivePlan(manifest buildimpact.Manifest, graph buildimpact.DeclaredGraph) (Plan, string) {
	if len(manifest.AllowedAlternatives) != 1 {
		return Plan{}, "AMBIGUOUS_ALTERNATIVE"
	}
	alternative := manifest.AllowedAlternatives[0]
	entrypoints := map[string]buildimpact.DeclaredEntrypoint{}
	for _, entrypoint := range graph.Entrypoints {
		entrypoints[entrypoint.Name] = entrypoint
	}
	covered := map[string]bool{}
	for _, name := range alternative.Entrypoints {
		entrypoint, ok := entrypoints[name]
		if !ok {
			return Plan{}, "ALTERNATIVE_NOT_IN_GRAPH"
		}
		if entrypoint.ContainsTestTasks {
			return Plan{}, "UNSUPPORTED_TEST_TASK"
		}
		for _, project := range entrypoint.ReachesProjects {
			covered[project] = true
		}
	}
	outputs := make([]string, 0, len(manifest.RequiredArtifacts))
	for _, artifact := range manifest.RequiredArtifacts {
		outputs = append(outputs, artifact.Path)
	}
	sort.Strings(outputs)
	return Plan{
		AlternativeID:       alternative.ID,
		Entrypoints:         append([]string{}, alternative.Entrypoints...),
		FallbackEntrypoints: append([]string{}, manifest.OriginalEntrypoints...),
		RequiredOutputs:     outputs,
		OmittedProjectCount: len(graph.Projects) - len(covered),
	}, ""
}

func graphHasUnknownRelationships(graph buildimpact.DeclaredGraph) bool {
	for _, p := range graph.Projects {
		if p.UnknownRelationships {
			return true
		}
	}
	for _, e := range graph.Entrypoints {
		if e.UnknownRelationships {
			return true
		}
	}
	return false
}

func selectedAlternativeContainsTestTasks(manifest buildimpact.Manifest, graph buildimpact.DeclaredGraph) bool {
	selected := map[string]bool{}
	for _, alternative := range manifest.AllowedAlternatives {
		for _, name := range alternative.Entrypoints {
			selected[name] = true
		}
	}
	for _, entrypoint := range graph.Entrypoints {
		if selected[entrypoint.Name] && entrypoint.ContainsTestTasks {
			return true
		}
	}
	return false
}

func mechanismDecisions() []MechanismDecision {
	return []MechanismDecision{
		{"BUILD_IMPACT", true, "complete digest-bound graph and qualified installed-path evidence"},
		{"SHARED_EDGE_CACHE", true, "qualified read-only locality with exact HTTP-failure fallback"},
		{"HOT_STATE", false, "not present in the qualified mechanism set"},
		{"RUNTIME_TUNING", false, "not present in the qualified mechanism set"},
		{"SAFE_CACHE", false, "not present in the qualified mechanism set"},
		{"STANDARD_COPY", false, "not present in the qualified mechanism set"},
		{"STANDARD_JAR", false, "required output is not produced by the standard Jar adapter"},
		{"TEST_OPTIMIZATION", false, "outside BuildOpt scope and forbidden by the evidence boundary"},
	}
}

func nativeMechanismDecisions() []MechanismDecision {
	decisions := mechanismDecisions()
	for index := range decisions {
		decisions[index].Enabled = false
		decisions[index].Reason = "native fallback retains the mechanism disabled"
	}
	return decisions
}

func readInput(root, relative, role string) ([]byte, InputBinding, error) {
	if relative == "" || relative == ".." || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, InputBinding{}, fmt.Errorf("%s path must be repository relative", role)
	}
	path := filepath.Join(root, relative)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, InputBinding{}, fmt.Errorf("inspect %s: %w", role, err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumInputBytes {
		return nil, InputBinding{}, fmt.Errorf("%s must be a bounded regular file", role)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, InputBinding{}, fmt.Errorf("resolve repository root: %w", err)
	}
	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, InputBinding{}, fmt.Errorf("resolve %s: %w", role, err)
	}
	contained, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
		return nil, InputBinding{}, fmt.Errorf("%s resolves outside the repository", role)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, InputBinding{}, fmt.Errorf("read %s: %w", role, err)
	}
	digest := sha256.Sum256(raw)
	return raw, InputBinding{Role: role, Path: relative, SHA256: "sha256:" + hex.EncodeToString(digest[:])}, nil
}

func decodeJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON content")
	}
	return nil
}

func manifestIdentity(raw []byte, field string) string {
	var value map[string]json.RawMessage
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	var result string
	_ = json.Unmarshal(value[field], &result)
	return result
}

func parseGeneratedManifest(raw []byte) (buildimpact.GeneratedManifest, error) {
	var generated buildimpact.GeneratedManifest
	if err := decodeJSON(raw, &generated); err != nil {
		return buildimpact.GeneratedManifest{}, err
	}
	return generated, nil
}

func generatedStateBound(g buildimpact.GeneratedManifest, manifest buildimpact.LoadedManifest, graph buildimpact.LoadedGraph) bool {
	return g.SchemaVersion == buildimpact.GeneratedManifestSchemaVersion && g.RepositoryID == manifest.Manifest.RepositoryID && g.PipelineClass == manifest.Manifest.PipelineClass && g.ManifestDigest == manifest.Digest && g.GraphDigest == graph.Digest && g.AdapterVersion == graph.Graph.AdapterVersion && validPrefixedSHA(g.DiscoveryDigest)
}

func sortedDigestMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		if validSHA(value) {
			result[key] = value
		}
	}
	return result
}

func validSHA(value string) bool {
	return len(value) == 64 && validHex(value)
}
func validRevision(value string) bool {
	return len(value) == 40 && validHex(value)
}
func validHex(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil
}
func validPrefixedSHA(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validSHA(strings.TrimPrefix(value, "sha256:"))
}
func sameOptionalFloat(left, right *float64) bool {
	return left != nil && right != nil && *left == *right
}
func sameFloatSlice(left, right []float64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
