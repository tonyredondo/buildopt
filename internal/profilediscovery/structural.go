package profilediscovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/outputequivalence"
)

const (
	StructuralEvidenceSchema                      = "buildopt.evidence/structural-profile-qualification/v1"
	StructuralProfileSchema                       = "buildopt.poc/qualified-profile/v4"
	StructuralProfileID                           = "qualified-structural-build-impact"
	CandidateStabilizationAdaptiveExactTwoOfThree = "ADAPTIVE_EXACT_2_OF_3"
	structuralPairCount                           = 8
	structuralMinimumSavedMS                      = 500.0
	structuralMinimumRatio                        = 0.02
)

// StructuralOptions binds an independently measured structural optimization
// to the exact repository graph from which the candidate was derived.
type StructuralOptions struct {
	RepositoryRoot        string
	ManifestPath          string
	GraphPath             string
	GeneratedPath         string
	EvidencePath          string
	OutputEquivalencePath string
}

// StructuralProfile is the reviewable Build Impact-only profile accepted by
// the installed POC. It carries no repository-family switch or task adapter.
type StructuralProfile struct {
	SchemaVersion  string                    `json:"schemaVersion"`
	ProfileVersion uint64                    `json:"profileVersion"`
	ProfileID      string                    `json:"profileId"`
	Ownership      string                    `json:"ownership"`
	ClaimScope     string                    `json:"claimScope"`
	RepositoryID   string                    `json:"repositoryId"`
	PipelineClass  string                    `json:"pipelineClass"`
	Fallback       string                    `json:"fallback"`
	Impact         ProfileImpact             `json:"impact"`
	Mechanisms     ProfileMechanisms         `json:"mechanisms"`
	GradleOptions  []string                  `json:"gradleOptions"`
	Preconditions  []Precondition            `json:"preconditions"`
	Qualification  StructuralProfileEvidence `json:"qualification"`
}

// StructuralProfileEvidence preserves the exact evidence identity and the
// independently recomputed value result without authorizing production use.
type StructuralProfileEvidence struct {
	SchemaVersion      string    `json:"schemaVersion"`
	SHA256             string    `json:"sha256"`
	RepositoryRevision string    `json:"repositoryRevision"`
	Pairs              int       `json:"pairs"`
	MeanSavedMS        float64   `json:"meanSavedMs"`
	ReductionRatio     float64   `json:"reductionRatio"`
	Interval95SavedMS  []float64 `json:"interval95SavedMs"`
}

type structuralEvidence struct {
	SchemaVersion  string                   `json:"schemaVersion"`
	EvidenceState  string                   `json:"evidenceState"`
	CapturedAt     string                   `json:"capturedAt"`
	Subject        structuralSubject        `json:"subject"`
	SourceBindings structuralSourceBindings `json:"sourceBindings"`
	Plan           AnalysisPlan             `json:"plan"`
	Execution      structuralExecution      `json:"execution"`
	Observations   []structuralObservation  `json:"observations"`
	Fallback       structuralFallback       `json:"fallback"`
	Result         structuralResult         `json:"result"`
	Boundaries     structuralBoundaries     `json:"boundaries"`
}

type structuralSubject struct {
	RepositoryID       string `json:"repositoryId"`
	RepositoryRevision string `json:"repositoryRevision"`
	PipelineClass      string `json:"pipelineClass"`
}

type structuralSourceBindings struct {
	ManifestSHA256          string `json:"manifestSha256"`
	GraphSHA256             string `json:"graphSha256"`
	GeneratedSHA256         string `json:"generatedManifestSha256"`
	SourceEvidenceSHA256    string `json:"sourceEvidenceSha256"`
	OutputEquivalenceSHA256 string `json:"outputEquivalenceSha256,omitempty"`
}

type structuralExecution struct {
	CandidateSurface         string                        `json:"candidateSurface"`
	BuildOptRevision         string                        `json:"buildoptRevision"`
	ExecutableSHA256         string                        `json:"executableSha256,omitempty"`
	Mechanisms               []string                      `json:"mechanisms"`
	GradleOptions            []string                      `json:"gradleOptions"`
	LauncherOverheadIncluded bool                          `json:"launcherOverheadIncluded"`
	OutputEquivalenceMode    string                        `json:"outputEquivalenceMode,omitempty"`
	WarmupsPerArm            int                           `json:"warmupsPerArm,omitempty"`
	ControlWarmupCount       int                           `json:"controlWarmupCount,omitempty"`
	CandidateWarmupCount     int                           `json:"candidateWarmupCount,omitempty"`
	CandidateStabilization   string                        `json:"candidateStabilization,omitempty"`
	ControlWarmups           []StructuralWarmupObservation `json:"controlWarmups,omitempty"`
	CandidateWarmups         []StructuralWarmupObservation `json:"candidateWarmups,omitempty"`
}

type structuralObservation struct {
	Pair                          int                     `json:"pair"`
	Order                         string                  `json:"order"`
	ControlDurationMS             int64                   `json:"controlDurationMs"`
	CandidateDurationMS           int64                   `json:"candidateDurationMs"`
	SavedMS                       int64                   `json:"savedMs"`
	ControlRequiredOutputSHA256   string                  `json:"controlRequiredOutputSha256"`
	CandidateRequiredOutputSHA256 string                  `json:"candidateRequiredOutputSha256"`
	RequiredOutputCount           int                     `json:"requiredOutputCount"`
	ProductAttributableFailure    bool                    `json:"productAttributableFailure"`
	ControlLogSHA256              string                  `json:"controlLogSha256,omitempty"`
	CandidateLogSHA256            string                  `json:"candidateLogSha256,omitempty"`
	ControlTaskOutcomes           StructuralTaskOutcomes  `json:"controlTaskOutcomes,omitempty"`
	CandidateTaskOutcomes         StructuralTaskOutcomes  `json:"candidateTaskOutcomes,omitempty"`
	ControlHostPressure           *StructuralHostPressure `json:"controlHostPressure,omitempty"`
	CandidateHostPressure         *StructuralHostPressure `json:"candidateHostPressure,omitempty"`
}

type structuralFallback struct {
	Mode            string `json:"mode"`
	Reason          string `json:"reason"`
	BuildSuccessful bool   `json:"buildSuccessful"`
}

type structuralResult struct {
	Pairs                     int       `json:"pairs"`
	ControlMeanMS             float64   `json:"controlMeanMs"`
	CandidateMeanMS           float64   `json:"candidateMeanMs"`
	MeanSavedMS               float64   `json:"meanSavedMs"`
	ReductionRatio            float64   `json:"reductionRatio"`
	Interval95SavedMS         []float64 `json:"interval95SavedMs"`
	PositivePairs             int       `json:"positivePairs"`
	ExecutionShapeObserved    bool      `json:"executionShapeObserved,omitempty"`
	ExecutionShapeStable      bool      `json:"executionShapeStable,omitempty"`
	TargetWarmupShapeObserved bool      `json:"targetWarmupShapeObserved,omitempty"`
	TargetWarmupShapeStable   bool      `json:"targetWarmupShapeStable,omitempty"`
	Qualified                 bool      `json:"qualified"`
	Decision                  string    `json:"decision"`
}

type structuralBoundaries struct {
	ProofOfConcept              bool `json:"proofOfConcept"`
	ProductionAuthorized        bool `json:"productionAuthorized"`
	RepositorySpecificRuleAdded bool `json:"repositorySpecificRuleAdded"`
	TestOptimizationModified    bool `json:"testOptimizationModified"`
}

// StructuralMeasurementObservation is one temporally paired optimized-native
// and installed-BuildOpt measurement over byte-identical required outputs.
type StructuralMeasurementObservation struct {
	Pair                       int
	Order                      string
	ControlDurationMS          int64
	CandidateDurationMS        int64
	RequiredOutputSHA256       string
	RequiredOutputCount        int
	ProductAttributableFailure bool
	ControlLogSHA256           string
	CandidateLogSHA256         string
	ControlTaskOutcomes        StructuralTaskOutcomes
	CandidateTaskOutcomes      StructuralTaskOutcomes
	ControlHostPressure        *StructuralHostPressure
	CandidateHostPressure      *StructuralHostPressure
}

// StructuralTaskOutcomes is a bounded execution-shape summary parsed from
// Gradle's plain console output. It helps diagnose timing outliers without
// retaining repository build logs or changing the measured Gradle invocation.
type StructuralTaskOutcomes struct {
	Total             int                         `json:"total"`
	Executed          int                         `json:"executed"`
	FromCache         int                         `json:"fromCache"`
	UpToDate          int                         `json:"upToDate"`
	NoSource          int                         `json:"noSource"`
	Skipped           int                         `json:"skipped"`
	FingerprintSHA256 string                      `json:"fingerprintSha256,omitempty"`
	Tasks             []StructuralTaskObservation `json:"tasks,omitempty"`
}

// StructuralTaskObservation preserves the exact Gradle task path and observed
// outcome used to derive an execution-shape fingerprint. The bounded list is
// diagnostic evidence; it never authorizes repository-specific selection.
type StructuralTaskObservation struct {
	Path                      string   `json:"path"`
	Outcome                   string   `json:"outcome"`
	ConsoleOutcomeTransitions []string `json:"consoleOutcomeTransitions,omitempty"`
}

// StructuralHostPressure records the system-wide Linux PSI time accumulated
// while one Gradle arm was running. It is diagnostic only and never changes a
// measured duration or qualifies an otherwise failing observation.
type StructuralHostPressure struct {
	Available         bool  `json:"available"`
	CPUSomeTotalUS    int64 `json:"cpuSomeTotalUs"`
	MemorySomeTotalUS int64 `json:"memorySomeTotalUs"`
	MemoryFullTotalUS int64 `json:"memoryFullTotalUs"`
	IOSomeTotalUS     int64 `json:"ioSomeTotalUs"`
	IOFullTotalUS     int64 `json:"ioFullTotalUs"`
}

// StructuralWarmupObservation records excluded cache-seeding and daemon-
// stabilization invocations. Warm-ups never contribute to qualification.
type StructuralWarmupObservation struct {
	Phase        string                  `json:"phase"`
	DurationMS   int64                   `json:"durationMs"`
	LogSHA256    string                  `json:"logSha256"`
	TaskOutcomes StructuralTaskOutcomes  `json:"taskOutcomes"`
	HostPressure *StructuralHostPressure `json:"hostPressure,omitempty"`
}

// StructuralMeasurementOptions carries the independently observed values used
// to render evidence for the existing structural profile qualifier. It grants
// no activation or production authority.
type StructuralMeasurementOptions struct {
	CapturedAt              time.Time
	Analysis                AnalysisReport
	RepositoryRevision      string
	BuildOptRevision        string
	ExecutableSHA256        string
	SourceEvidenceSHA256    string
	OutputEquivalenceSHA256 string
	GradleOptions           []string
	ControlWarmups          []StructuralWarmupObservation
	CandidateWarmups        []StructuralWarmupObservation
	CandidateStabilization  string
	Observations            []StructuralMeasurementObservation
	FallbackReason          string
	FallbackSuccessful      bool
}

// RenderStructuralMeasurementEvidence recomputes the frozen qualification
// result and emits the exact evidence document consumed by
// QualifyStructuralProfile. A valid but non-positive result remains
// INCONCLUSIVE and therefore cannot materialize a profile.
func RenderStructuralMeasurementEvidence(options StructuralMeasurementOptions) ([]byte, bool, error) {
	if options.Analysis.Decision != DecisionMeasure || options.Analysis.Plan == nil {
		return nil, false, errors.New("structural measurement requires a complete candidate analysis")
	}
	if !validRevision(options.RepositoryRevision) || !validRevision(options.BuildOptRevision) ||
		!validSHA(options.ExecutableSHA256) ||
		options.ExecutableSHA256 != strings.ToLower(options.ExecutableSHA256) ||
		!validSHA(options.SourceEvidenceSHA256) ||
		options.SourceEvidenceSHA256 != strings.ToLower(options.SourceEvidenceSHA256) ||
		!validStructuralGradleOptions(options.GradleOptions) {
		return nil, false, errors.New("structural measurement identity or Gradle options are invalid")
	}
	if options.CapturedAt.IsZero() || options.FallbackReason == "" || !options.FallbackSuccessful {
		return nil, false, errors.New("structural measurement fallback is unproven")
	}
	if options.CandidateStabilization != "" && options.CandidateStabilization != CandidateStabilizationAdaptiveExactTwoOfThree {
		return nil, false, errors.New("structural candidate stabilization policy is invalid")
	}
	if options.CandidateStabilization == "" && len(options.ControlWarmups) != len(options.CandidateWarmups) {
		return nil, false, errors.New("structural legacy warm-up counts must match")
	}
	if options.CandidateStabilization == CandidateStabilizationAdaptiveExactTwoOfThree &&
		(len(options.ControlWarmups) != 5 || (len(options.CandidateWarmups) != 4 && len(options.CandidateWarmups) != 5)) {
		return nil, false, errors.New("adaptive candidate stabilization requires five control and four or five candidate warm-ups")
	}
	outputEquivalenceMode := ""
	if options.OutputEquivalenceSHA256 != "" {
		if !validSHA(options.OutputEquivalenceSHA256) || options.OutputEquivalenceSHA256 != strings.ToLower(options.OutputEquivalenceSHA256) {
			return nil, false, errors.New("structural output-equivalence binding is invalid")
		}
		outputEquivalenceMode = "OWNER_REVIEWED_SEMANTIC_V1"
	}
	inputs := make(map[string]InputBinding, len(options.Analysis.Inputs))
	for _, input := range options.Analysis.Inputs {
		inputs[input.Role] = input
	}
	for _, role := range []string{"BUILD_IMPACT_MANIFEST", "BUILD_IMPACT_GRAPH", "GENERATED_MANIFEST"} {
		if !validSHA(trimSHA(inputs[role].SHA256)) {
			return nil, false, fmt.Errorf("structural measurement lacks %s binding", role)
		}
	}
	observations := make([]structuralObservation, len(options.Observations))
	for index, observation := range options.Observations {
		observations[index] = structuralObservation{
			Pair:                          observation.Pair,
			Order:                         observation.Order,
			ControlDurationMS:             observation.ControlDurationMS,
			CandidateDurationMS:           observation.CandidateDurationMS,
			SavedMS:                       observation.ControlDurationMS - observation.CandidateDurationMS,
			ControlRequiredOutputSHA256:   observation.RequiredOutputSHA256,
			CandidateRequiredOutputSHA256: observation.RequiredOutputSHA256,
			RequiredOutputCount:           observation.RequiredOutputCount,
			ProductAttributableFailure:    observation.ProductAttributableFailure,
			ControlLogSHA256:              observation.ControlLogSHA256,
			CandidateLogSHA256:            observation.CandidateLogSHA256,
			ControlTaskOutcomes:           observation.ControlTaskOutcomes,
			CandidateTaskOutcomes:         observation.CandidateTaskOutcomes,
			ControlHostPressure:           observation.ControlHostPressure,
			CandidateHostPressure:         observation.CandidateHostPressure,
		}
	}
	if err := validateStructuralDiagnostics(options.ControlWarmups, options.CandidateWarmups, observations); err != nil {
		return nil, false, err
	}
	result, err := calculateStructuralResult(observations)
	if err != nil {
		return nil, false, err
	}
	result = applyStructuralTargetWarmupShape(result, options.ControlWarmups, options.CandidateWarmups, observations)
	evidenceState := "INCONCLUSIVE"
	if result.Qualified {
		evidenceState = "QUALIFIED"
	}
	evidence := structuralEvidence{
		SchemaVersion: StructuralEvidenceSchema,
		EvidenceState: evidenceState,
		CapturedAt:    options.CapturedAt.UTC().Truncate(time.Second).Format(time.RFC3339),
		Subject: structuralSubject{
			RepositoryID:       options.Analysis.Subject.RepositoryID,
			RepositoryRevision: options.RepositoryRevision,
			PipelineClass:      options.Analysis.Subject.PipelineClass,
		},
		SourceBindings: structuralSourceBindings{
			ManifestSHA256:          trimSHA(inputs["BUILD_IMPACT_MANIFEST"].SHA256),
			GraphSHA256:             trimSHA(inputs["BUILD_IMPACT_GRAPH"].SHA256),
			GeneratedSHA256:         trimSHA(inputs["GENERATED_MANIFEST"].SHA256),
			SourceEvidenceSHA256:    options.SourceEvidenceSHA256,
			OutputEquivalenceSHA256: options.OutputEquivalenceSHA256,
		},
		Plan: *options.Analysis.Plan,
		Execution: structuralExecution{
			CandidateSurface:         "INSTALLED_BUILDOPT_STRUCTURAL_IMPACT_ONLY",
			BuildOptRevision:         options.BuildOptRevision,
			ExecutableSHA256:         options.ExecutableSHA256,
			Mechanisms:               []string{"BUILD_IMPACT"},
			GradleOptions:            append([]string(nil), options.GradleOptions...),
			LauncherOverheadIncluded: true,
			OutputEquivalenceMode:    outputEquivalenceMode,
			WarmupsPerArm:            equalStructuralWarmupCount(options.ControlWarmups, options.CandidateWarmups),
			ControlWarmupCount:       adaptiveStructuralWarmupCount(options.CandidateStabilization, len(options.ControlWarmups)),
			CandidateWarmupCount:     adaptiveStructuralWarmupCount(options.CandidateStabilization, len(options.CandidateWarmups)),
			CandidateStabilization:   options.CandidateStabilization,
			ControlWarmups:           append([]StructuralWarmupObservation(nil), options.ControlWarmups...),
			CandidateWarmups:         append([]StructuralWarmupObservation(nil), options.CandidateWarmups...),
		},
		Observations: observations,
		Fallback: structuralFallback{
			Mode:            "FULL_GRAPH",
			Reason:          options.FallbackReason,
			BuildSuccessful: true,
		},
		Result: result,
		Boundaries: structuralBoundaries{
			ProofOfConcept:              true,
			ProductionAuthorized:        false,
			RepositorySpecificRuleAdded: false,
			TestOptimizationModified:    false,
		},
	}
	raw, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("render structural measurement evidence: %w", err)
	}
	return append(raw, '\n'), result.Qualified, nil
}

// QualifyStructuralProfile materializes a repository-name-independent
// Build Impact profile only when exact installed-path evidence beats optimized
// native Gradle and every correctness and fallback gate remains satisfied.
func QualifyStructuralProfile(options StructuralOptions) (StructuralProfile, error) {
	analysis, err := AnalyzeOpportunity(AnalysisOptions{
		RepositoryRoot: options.RepositoryRoot,
		ManifestPath:   options.ManifestPath,
		GraphPath:      options.GraphPath,
		GeneratedPath:  options.GeneratedPath,
	})
	if err != nil {
		return StructuralProfile{}, err
	}
	if analysis.Decision != DecisionMeasure || analysis.Plan == nil {
		return StructuralProfile{}, fmt.Errorf("structural profile retained native Gradle: %s", analysis.Reason)
	}
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return StructuralProfile{}, fmt.Errorf("resolve repository root: %w", err)
	}
	evidenceRaw, evidenceInput, err := readInput(root, options.EvidencePath, "STRUCTURAL_QUALIFICATION_EVIDENCE")
	if err != nil {
		return StructuralProfile{}, err
	}
	var evidence structuralEvidence
	if err := decodeStrictJSON(evidenceRaw, &evidence); err != nil {
		return StructuralProfile{}, fmt.Errorf("decode structural qualification evidence: %w", err)
	}
	if err := validateStructuralEvidence(evidence, analysis); err != nil {
		return StructuralProfile{}, err
	}
	var equivalenceInput *InputBinding
	if evidence.SourceBindings.OutputEquivalenceSHA256 != "" {
		if options.OutputEquivalencePath == "" {
			return StructuralProfile{}, errors.New("owner-reviewed output-equivalence contract is required")
		}
		equivalenceRaw, input, err := readInput(root, options.OutputEquivalencePath, "OUTPUT_EQUIVALENCE_CONTRACT")
		if err != nil {
			return StructuralProfile{}, err
		}
		if _, err := outputequivalence.Parse(equivalenceRaw); err != nil {
			return StructuralProfile{}, err
		}
		if !sameEvidenceSHA(input.SHA256, evidence.SourceBindings.OutputEquivalenceSHA256) {
			return StructuralProfile{}, errors.New("output-equivalence contract binding drift")
		}
		equivalenceInput = &input
	} else if options.OutputEquivalencePath != "" {
		return StructuralProfile{}, errors.New("exact-byte evidence cannot add an output-equivalence contract")
	}
	inputByRole := make(map[string]InputBinding, len(analysis.Inputs))
	for _, input := range analysis.Inputs {
		inputByRole[input.Role] = input
	}
	if !sameEvidenceSHA(inputByRole["BUILD_IMPACT_MANIFEST"].SHA256, evidence.SourceBindings.ManifestSHA256) ||
		!sameEvidenceSHA(inputByRole["BUILD_IMPACT_GRAPH"].SHA256, evidence.SourceBindings.GraphSHA256) ||
		!sameEvidenceSHA(inputByRole["GENERATED_MANIFEST"].SHA256, evidence.SourceBindings.GeneratedSHA256) {
		return StructuralProfile{}, errors.New("structural qualification source binding drift")
	}
	preconditions := []Precondition{
		{Type: "FILE_SHA256", Path: options.ManifestPath, SHA256: evidence.SourceBindings.ManifestSHA256},
		{Type: "FILE_SHA256", Path: options.GraphPath, SHA256: evidence.SourceBindings.GraphSHA256},
		{Type: "FILE_SHA256", Path: options.GeneratedPath, SHA256: evidence.SourceBindings.GeneratedSHA256},
	}
	if equivalenceInput != nil {
		preconditions = append(preconditions, Precondition{
			Type: "FILE_SHA256", Path: options.OutputEquivalencePath,
			SHA256: evidence.SourceBindings.OutputEquivalenceSHA256,
		})
	}
	evidenceSHA := trimSHA(evidenceInput.SHA256)
	return StructuralProfile{
		SchemaVersion:  StructuralProfileSchema,
		ProfileVersion: 4,
		ProfileID:      StructuralProfileID,
		Ownership:      profileOwnership,
		ClaimScope:     profileClaimScope,
		RepositoryID:   analysis.Subject.RepositoryID,
		PipelineClass:  analysis.Subject.PipelineClass,
		Fallback:       profileFallback,
		Impact: ProfileImpact{
			Manifest:          options.ManifestPath,
			Graph:             options.GraphPath,
			GeneratedManifest: options.GeneratedPath,
		},
		Mechanisms:    ProfileMechanisms{BuildImpact: true},
		GradleOptions: append([]string(nil), evidence.Execution.GradleOptions...),
		Preconditions: preconditions,
		Qualification: StructuralProfileEvidence{
			SchemaVersion:      evidence.SchemaVersion,
			SHA256:             evidenceSHA,
			RepositoryRevision: evidence.Subject.RepositoryRevision,
			Pairs:              evidence.Result.Pairs,
			MeanSavedMS:        evidence.Result.MeanSavedMS,
			ReductionRatio:     evidence.Result.ReductionRatio,
			Interval95SavedMS:  append([]float64(nil), evidence.Result.Interval95SavedMS...),
		},
	}, nil
}

// RenderStructuralProfile emits canonical reviewable JSON for repository
// ownership. The command never writes or activates the profile itself.
func RenderStructuralProfile(profile StructuralProfile) ([]byte, error) {
	raw, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render structural profile: %w", err)
	}
	return append(raw, '\n'), nil
}

func validateStructuralEvidence(evidence structuralEvidence, analysis AnalysisReport) error {
	if err := validateStructuralCaptureEvidence(evidence, analysis); err != nil {
		return err
	}
	if evidence.EvidenceState != "QUALIFIED" || !evidence.Result.Qualified {
		return errors.New("structural qualification evidence is not qualified")
	}
	return nil
}

// validateStructuralCaptureEvidence accepts both qualified and inconclusive
// v1 captures, but recomputes every correctness and timing field. Balanced v2
// qualification uses it so a noisy individual capture can contribute raw
// observations without weakening output, execution-shape, or fallback gates.
func validateStructuralCaptureEvidence(evidence structuralEvidence, analysis AnalysisReport) error {
	if evidence.SchemaVersion != StructuralEvidenceSchema ||
		(evidence.EvidenceState != "QUALIFIED" && evidence.EvidenceState != "INCONCLUSIVE") {
		return errors.New("structural qualification capture state is invalid")
	}
	if _, err := time.Parse(time.RFC3339, evidence.CapturedAt); err != nil {
		return errors.New("structural qualification capture time is invalid")
	}
	if evidence.Subject.RepositoryID != analysis.Subject.RepositoryID ||
		evidence.Subject.PipelineClass != analysis.Subject.PipelineClass ||
		!validRevision(evidence.Subject.RepositoryRevision) {
		return errors.New("structural qualification subject drift")
	}
	if !sameAnalysisPlan(evidence.Plan, *analysis.Plan) {
		return errors.New("structural qualification plan drift")
	}
	if evidence.Execution.CandidateSurface != "INSTALLED_BUILDOPT_STRUCTURAL_IMPACT_ONLY" ||
		!validRevision(evidence.Execution.BuildOptRevision) ||
		(evidence.Execution.ExecutableSHA256 != "" &&
			(!validSHA(evidence.Execution.ExecutableSHA256) ||
				evidence.Execution.ExecutableSHA256 != strings.ToLower(evidence.Execution.ExecutableSHA256))) ||
		!evidence.Execution.LauncherOverheadIncluded ||
		len(evidence.Execution.Mechanisms) != 1 || evidence.Execution.Mechanisms[0] != "BUILD_IMPACT" ||
		!validStructuralGradleOptions(evidence.Execution.GradleOptions) {
		return errors.New("structural qualification execution surface is invalid")
	}
	if evidence.SourceBindings.OutputEquivalenceSHA256 == "" {
		if evidence.Execution.OutputEquivalenceMode != "" {
			return errors.New("structural output-equivalence execution binding is invalid")
		}
	} else if evidence.Execution.OutputEquivalenceMode != "OWNER_REVIEWED_SEMANTIC_V1" {
		return errors.New("structural output-equivalence execution binding is invalid")
	}
	if !validStructuralWarmupCounts(evidence.Execution) {
		return errors.New("structural qualification warm-up evidence is invalid")
	}
	if err := validateStructuralDiagnostics(evidence.Execution.ControlWarmups, evidence.Execution.CandidateWarmups, evidence.Observations); err != nil {
		return err
	}
	calculated, err := calculateStructuralResult(evidence.Observations)
	if err != nil {
		return err
	}
	calculated = applyStructuralTargetWarmupShape(
		calculated,
		evidence.Execution.ControlWarmups,
		evidence.Execution.CandidateWarmups,
		evidence.Observations,
	)
	if !sameStructuralResult(calculated, evidence.Result) ||
		(evidence.EvidenceState == "QUALIFIED") != calculated.Qualified {
		return errors.New("structural qualification result is invalid")
	}
	if evidence.Fallback.Mode != "FULL_GRAPH" || evidence.Fallback.Reason == "" || !evidence.Fallback.BuildSuccessful {
		return errors.New("structural qualification native fallback is unproven")
	}
	if !evidence.Boundaries.ProofOfConcept || evidence.Boundaries.ProductionAuthorized ||
		evidence.Boundaries.RepositorySpecificRuleAdded || evidence.Boundaries.TestOptimizationModified {
		return errors.New("structural qualification boundary is invalid")
	}
	for _, digest := range []string{
		evidence.SourceBindings.ManifestSHA256,
		evidence.SourceBindings.GraphSHA256,
		evidence.SourceBindings.GeneratedSHA256,
		evidence.SourceBindings.SourceEvidenceSHA256,
	} {
		if !validSHA(digest) || digest != strings.ToLower(digest) {
			return errors.New("structural qualification source digest is invalid")
		}
	}
	if digest := evidence.SourceBindings.OutputEquivalenceSHA256; digest != "" &&
		(!validSHA(digest) || digest != strings.ToLower(digest)) {
		return errors.New("structural output-equivalence source digest is invalid")
	}
	return nil
}

func validateStructuralDiagnostics(control, candidate []StructuralWarmupObservation, observations []structuralObservation) error {
	if len(control) == 0 && len(candidate) == 0 {
		for _, observation := range observations {
			if observation.ControlLogSHA256 != "" || observation.CandidateLogSHA256 != "" ||
				observation.ControlTaskOutcomes.Total != 0 || observation.CandidateTaskOutcomes.Total != 0 ||
				observation.ControlHostPressure != nil || observation.CandidateHostPressure != nil {
				return errors.New("structural measurement diagnostics are incomplete")
			}
		}
		return nil
	}
	if !validStructuralWarmupLength(len(control)) || !validStructuralWarmupLength(len(candidate)) {
		return errors.New("structural measurement requires complete two-, three-, four-, or five-phase warm-ups")
	}
	fingerprintPresent := false
	fingerprintAbsent := false
	pressurePresent := false
	pressureAbsent := false
	observeDiagnostics := func(outcomes StructuralTaskOutcomes, pressure *StructuralHostPressure) {
		if outcomes.FingerprintSHA256 == "" {
			fingerprintAbsent = true
		} else {
			fingerprintPresent = true
		}
		if pressure == nil {
			pressureAbsent = true
		} else {
			pressurePresent = true
		}
	}
	for _, warmups := range [][]StructuralWarmupObservation{control, candidate} {
		expectedPhases := structuralWarmupPhases(len(warmups))
		for index, warmup := range warmups {
			if warmup.Phase != expectedPhases[index] || warmup.DurationMS <= 0 ||
				!validSHA(warmup.LogSHA256) || !validStructuralTaskOutcomes(warmup.TaskOutcomes) ||
				!validStructuralHostPressure(warmup.HostPressure) {
				return errors.New("structural measurement warm-up diagnostic is invalid")
			}
			observeDiagnostics(warmup.TaskOutcomes, warmup.HostPressure)
		}
	}
	for _, observation := range observations {
		if !validSHA(observation.ControlLogSHA256) || !validSHA(observation.CandidateLogSHA256) ||
			!validStructuralTaskOutcomes(observation.ControlTaskOutcomes) ||
			!validStructuralTaskOutcomes(observation.CandidateTaskOutcomes) ||
			!validStructuralHostPressure(observation.ControlHostPressure) ||
			!validStructuralHostPressure(observation.CandidateHostPressure) {
			return errors.New("structural measurement pair diagnostic is invalid")
		}
		observeDiagnostics(observation.ControlTaskOutcomes, observation.ControlHostPressure)
		observeDiagnostics(observation.CandidateTaskOutcomes, observation.CandidateHostPressure)
	}
	if fingerprintPresent && fingerprintAbsent {
		return errors.New("structural measurement task fingerprints are incomplete")
	}
	if pressurePresent && pressureAbsent {
		return errors.New("structural measurement host-pressure diagnostics are incomplete")
	}
	return nil
}

func equalStructuralWarmupCount(control, candidate []StructuralWarmupObservation) int {
	if len(control) == len(candidate) {
		return len(control)
	}
	return 0
}

func adaptiveStructuralWarmupCount(policy string, count int) int {
	if policy == CandidateStabilizationAdaptiveExactTwoOfThree {
		return count
	}
	return 0
}

func validStructuralWarmupCounts(execution structuralExecution) bool {
	control := len(execution.ControlWarmups)
	candidate := len(execution.CandidateWarmups)
	if execution.CandidateStabilization == "" {
		return execution.ControlWarmupCount == 0 && execution.CandidateWarmupCount == 0 &&
			execution.WarmupsPerArm == control && execution.WarmupsPerArm == candidate
	}
	return execution.CandidateStabilization == CandidateStabilizationAdaptiveExactTwoOfThree &&
		execution.WarmupsPerArm == 0 && execution.ControlWarmupCount == control &&
		execution.CandidateWarmupCount == candidate && control == 5 &&
		(candidate == 4 || candidate == 5)
}

func validStructuralWarmupLength(count int) bool {
	return count >= 2 && count <= 5
}

func structuralWarmupPhases(count int) []string {
	phases := []string{"CACHE_SEED", "DAEMON_STABILIZATION"}
	if count >= 3 {
		phases = []string{"CACHE_SEED", "BASE_DAEMON_STABILIZATION", "TARGET_WORKLOAD_STABILIZATION"}
	}
	if count >= 4 {
		phases = append(phases, "TARGET_WORKLOAD_STABILITY_CONFIRMATION")
	}
	if count == 5 {
		phases = append(phases, "TARGET_WORKLOAD_STABILITY_RECONFIRMATION")
	}
	return phases
}

func structuralWarmupBaselineIndex(count int) int {
	if count == 5 {
		return 4
	}
	return 2
}

func structuralWarmupComparisonStart() int {
	return 3
}

func validStructuralTaskOutcomes(outcomes StructuralTaskOutcomes) bool {
	return ValidateStructuralTaskOutcomes(outcomes) == nil
}

// ValidateStructuralTaskOutcomes checks that the bounded exact-task evidence
// recomputes the same canonical fingerprint as the Gradle console summary.
// Measurement callers use the detailed error to fail on the first malformed
// sample instead of discovering it only after an expensive paired run.
func ValidateStructuralTaskOutcomes(outcomes StructuralTaskOutcomes) error {
	if outcomes.Total < 0 || outcomes.Executed < 0 || outcomes.FromCache < 0 ||
		outcomes.UpToDate < 0 || outcomes.NoSource < 0 || outcomes.Skipped < 0 ||
		outcomes.Total != outcomes.Executed+outcomes.FromCache+outcomes.UpToDate+outcomes.NoSource+outcomes.Skipped {
		return errors.New("task outcome counters are inconsistent")
	}
	if outcomes.FingerprintSHA256 != "" &&
		(!validSHA(outcomes.FingerprintSHA256) || outcomes.FingerprintSHA256 != strings.ToLower(outcomes.FingerprintSHA256)) {
		return errors.New("task fingerprint is not a lowercase SHA-256 digest")
	}
	if len(outcomes.Tasks) == 0 {
		return nil
	}
	if len(outcomes.Tasks) != outcomes.Total || outcomes.FingerprintSHA256 == "" {
		return fmt.Errorf("exact task evidence has %d entries for total %d", len(outcomes.Tasks), outcomes.Total)
	}
	canonical := make([]string, len(outcomes.Tasks))
	previous := ""
	var executed, fromCache, upToDate, noSource, skipped int
	for index, task := range outcomes.Tasks {
		if task.Path == "" || !strings.HasPrefix(task.Path, ":") {
			return fmt.Errorf("exact task %d has invalid path %q", index+1, task.Path)
		}
		if !validStructuralTaskOutcome(task.Outcome) {
			return fmt.Errorf("exact task %s has invalid outcome %q", task.Path, task.Outcome)
		}
		if len(task.ConsoleOutcomeTransitions) != 0 {
			if len(task.ConsoleOutcomeTransitions) < 2 || len(task.ConsoleOutcomeTransitions) > 8 {
				return fmt.Errorf("exact task %s has invalid console outcome transitions", task.Path)
			}
			previousOutcome := ""
			for _, outcome := range task.ConsoleOutcomeTransitions {
				if !validStructuralTaskOutcome(outcome) || outcome == previousOutcome {
					return fmt.Errorf("exact task %s has invalid console outcome transitions", task.Path)
				}
				previousOutcome = outcome
			}
			if task.ConsoleOutcomeTransitions[len(task.ConsoleOutcomeTransitions)-1] != task.Outcome {
				return fmt.Errorf("exact task %s does not use its terminal console outcome", task.Path)
			}
		}
		if previous != "" && task.Path <= previous {
			return fmt.Errorf("exact task paths are not strictly sorted: %q followed by %q", previous, task.Path)
		}
		previous = task.Path
		canonical[index] = structuralTaskFingerprintLine(task)
		switch task.Outcome {
		case "FROM_CACHE":
			fromCache++
		case "UP_TO_DATE":
			upToDate++
		case "NO_SOURCE":
			noSource++
		case "SKIPPED":
			skipped++
		default:
			executed++
		}
	}
	if outcomes.Executed != executed || outcomes.FromCache != fromCache || outcomes.UpToDate != upToDate ||
		outcomes.NoSource != noSource || outcomes.Skipped != skipped {
		return errors.New("exact task outcome counters do not match task evidence")
	}
	digest := sha256.Sum256([]byte(strings.Join(canonical, "\n") + "\n"))
	calculated := hex.EncodeToString(digest[:])
	if calculated != outcomes.FingerprintSHA256 {
		return fmt.Errorf("exact task fingerprint mismatch: calculated %s, recorded %s", calculated, outcomes.FingerprintSHA256)
	}
	return nil
}

func validStructuralTaskOutcome(outcome string) bool {
	switch outcome {
	case "EXECUTED", "FROM_CACHE", "UP_TO_DATE", "NO_SOURCE", "SKIPPED":
		return true
	default:
		return false
	}
}

func structuralTaskFingerprintLine(task StructuralTaskObservation) string {
	line := "> Task " + task.Path
	switch task.Outcome {
	case "FROM_CACHE":
		line += " FROM-CACHE"
	case "UP_TO_DATE":
		line += " UP-TO-DATE"
	case "NO_SOURCE":
		line += " NO-SOURCE"
	case "SKIPPED":
		line += " SKIPPED"
	default:
	}
	return line
}

func validStructuralHostPressure(pressure *StructuralHostPressure) bool {
	return ValidateStructuralHostPressure(pressure) == nil
}

// ValidateStructuralHostPressure verifies an optional interval PSI sample.
func ValidateStructuralHostPressure(pressure *StructuralHostPressure) error {
	if pressure == nil {
		return nil
	}
	if !pressure.Available || pressure.CPUSomeTotalUS < 0 || pressure.MemorySomeTotalUS < 0 ||
		pressure.MemoryFullTotalUS < 0 || pressure.IOSomeTotalUS < 0 || pressure.IOFullTotalUS < 0 {
		return errors.New("host-pressure interval is unavailable or contains a negative counter")
	}
	return nil
}

func calculateStructuralResult(observations []structuralObservation) (structuralResult, error) {
	if len(observations) != structuralPairCount {
		return structuralResult{}, errors.New("structural qualification requires eight pairs")
	}
	saved := make([]float64, 0, structuralPairCount)
	var controlTotal, candidateTotal int64
	positive := 0
	requiredOutputSHA := ""
	executionShapeObserved := true
	executionShapeStable := true
	controlTaskFingerprint := ""
	candidateTaskFingerprint := ""
	for index, observation := range observations {
		expectedOrder := "CANDIDATE_FIRST"
		if index%2 == 0 {
			expectedOrder = "CONTROL_FIRST"
		}
		if observation.Pair != index+1 || observation.Order != expectedOrder ||
			observation.ControlDurationMS <= 0 || observation.CandidateDurationMS <= 0 ||
			observation.SavedMS != observation.ControlDurationMS-observation.CandidateDurationMS ||
			observation.RequiredOutputCount <= 0 || observation.ProductAttributableFailure ||
			!validSHA(observation.ControlRequiredOutputSHA256) ||
			observation.ControlRequiredOutputSHA256 != observation.CandidateRequiredOutputSHA256 {
			return structuralResult{}, errors.New("structural qualification observation is invalid")
		}
		if requiredOutputSHA == "" {
			requiredOutputSHA = observation.ControlRequiredOutputSHA256
		} else if observation.ControlRequiredOutputSHA256 != requiredOutputSHA {
			return structuralResult{}, errors.New("structural qualification outputs are not stable across pairs")
		}
		controlTotal += observation.ControlDurationMS
		candidateTotal += observation.CandidateDurationMS
		saved = append(saved, float64(observation.SavedMS))
		if observation.SavedMS > 0 {
			positive++
		}
		if observation.ControlTaskOutcomes.FingerprintSHA256 == "" ||
			observation.CandidateTaskOutcomes.FingerprintSHA256 == "" {
			executionShapeObserved = false
			executionShapeStable = false
		} else if controlTaskFingerprint == "" {
			controlTaskFingerprint = observation.ControlTaskOutcomes.FingerprintSHA256
			candidateTaskFingerprint = observation.CandidateTaskOutcomes.FingerprintSHA256
		} else if observation.ControlTaskOutcomes.FingerprintSHA256 != controlTaskFingerprint ||
			observation.CandidateTaskOutcomes.FingerprintSHA256 != candidateTaskFingerprint {
			executionShapeStable = false
		}
	}
	meanSaved := float64(controlTotal-candidateTotal) / structuralPairCount
	ratio := float64(controlTotal-candidateTotal) / float64(controlTotal)
	interval := structuralBootstrap95(saved)
	qualified := meanSaved >= structuralMinimumSavedMS && ratio >= structuralMinimumRatio &&
		interval[0] > 0 && positive == structuralPairCount &&
		(!executionShapeObserved || executionShapeStable)
	decision := "RETAIN_NATIVE_GRADLE"
	if qualified {
		decision = "QUALIFY_STRUCTURAL_PROFILE"
	}
	return structuralResult{
		Pairs:                  structuralPairCount,
		ControlMeanMS:          float64(controlTotal) / structuralPairCount,
		CandidateMeanMS:        float64(candidateTotal) / structuralPairCount,
		MeanSavedMS:            meanSaved,
		ReductionRatio:         ratio,
		Interval95SavedMS:      interval,
		PositivePairs:          positive,
		ExecutionShapeObserved: executionShapeObserved,
		ExecutionShapeStable:   executionShapeStable,
		Qualified:              qualified,
		Decision:               decision,
	}, nil
}

func applyStructuralTargetWarmupShape(
	result structuralResult,
	control, candidate []StructuralWarmupObservation,
	observations []structuralObservation,
) structuralResult {
	if len(control) < 3 || len(control) > 5 || len(candidate) < 3 || len(candidate) > 5 || len(observations) == 0 {
		return result
	}
	controlBaseline := structuralWarmupBaselineIndex(len(control))
	candidateBaseline := structuralWarmupBaselineIndex(len(candidate))
	controlFingerprint := control[controlBaseline].TaskOutcomes.FingerprintSHA256
	candidateFingerprint := candidate[candidateBaseline].TaskOutcomes.FingerprintSHA256
	if controlFingerprint == "" || candidateFingerprint == "" {
		return result
	}
	result.TargetWarmupShapeObserved = true
	result.TargetWarmupShapeStable = true
	for _, warmup := range control[structuralWarmupComparisonStart():] {
		result.TargetWarmupShapeStable = result.TargetWarmupShapeStable &&
			warmup.TaskOutcomes.FingerprintSHA256 == controlFingerprint
	}
	for _, warmup := range candidate[structuralWarmupComparisonStart():] {
		result.TargetWarmupShapeStable = result.TargetWarmupShapeStable &&
			warmup.TaskOutcomes.FingerprintSHA256 == candidateFingerprint
	}
	for _, observation := range observations {
		result.TargetWarmupShapeStable = result.TargetWarmupShapeStable &&
			observation.ControlTaskOutcomes.FingerprintSHA256 == controlFingerprint &&
			observation.CandidateTaskOutcomes.FingerprintSHA256 == candidateFingerprint
	}
	if !result.TargetWarmupShapeStable {
		result.Qualified = false
		result.Decision = "RETAIN_NATIVE_GRADLE"
	}
	return result
}

func structuralBootstrap95(saved []float64) []float64 {
	means := make([]float64, 4096)
	for sample := 0; sample < len(means); sample++ {
		state := uint64(2654435761*uint64(sample+1)) % (uint64(1) << 32)
		total := 0.0
		for draw := 0; draw < structuralPairCount; draw++ {
			state = (1664525*state + 1013904223) % (uint64(1) << 32)
			total += saved[int(state/536870912)]
		}
		means[sample] = total / structuralPairCount
	}
	sort.Float64s(means)
	return []float64{means[102], means[3993]}
}

func validStructuralGradleOptions(options []string) bool {
	if len(options) == 0 || len(options) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, option := range options {
		valid := option == "--offline" || option == "--daemon" || option == "--no-daemon" ||
			option == "--parallel" || option == "--no-parallel" || option == "--no-scan" ||
			option == "--stacktrace" || option == "--full-stacktrace" || option == "--info" ||
			option == "--debug" || option == "--warn" || option == "--build-cache" ||
			option == "--no-build-cache" || option == "--configuration-cache" ||
			option == "--no-configuration-cache" ||
			(option == "--console=plain" || option == "--console=auto" || option == "--console=rich" || option == "--console=verbose")
		if !valid && strings.HasPrefix(option, "--max-workers=") {
			workers, err := strconv.Atoi(strings.TrimPrefix(option, "--max-workers="))
			valid = err == nil && workers > 0
		}
		if !valid || seen[option] {
			return false
		}
		seen[option] = true
	}
	return true
}

func sameAnalysisPlan(left, right AnalysisPlan) bool {
	return left.AlternativeID == right.AlternativeID &&
		sameStrings(left.Entrypoints, right.Entrypoints) &&
		sameStrings(left.FallbackEntrypoints, right.FallbackEntrypoints) &&
		sameStrings(left.RequiredOutputs, right.RequiredOutputs) &&
		left.TotalProjectCount == right.TotalProjectCount &&
		left.SelectedProjectCount == right.SelectedProjectCount &&
		left.OmittedProjectCount == right.OmittedProjectCount &&
		sameNumber(left.OmittedProjectRatio, right.OmittedProjectRatio)
}

func sameStructuralResult(left, right structuralResult) bool {
	return left.Pairs == right.Pairs && sameNumber(left.ControlMeanMS, right.ControlMeanMS) &&
		sameNumber(left.CandidateMeanMS, right.CandidateMeanMS) &&
		sameNumber(left.MeanSavedMS, right.MeanSavedMS) && sameNumber(left.ReductionRatio, right.ReductionRatio) &&
		sameNumbers(left.Interval95SavedMS, right.Interval95SavedMS) && left.PositivePairs == right.PositivePairs &&
		left.ExecutionShapeObserved == right.ExecutionShapeObserved &&
		left.ExecutionShapeStable == right.ExecutionShapeStable &&
		left.TargetWarmupShapeObserved == right.TargetWarmupShapeObserved &&
		left.TargetWarmupShapeStable == right.TargetWarmupShapeStable &&
		left.Qualified == right.Qualified && left.Decision == right.Decision
}

func decodeStrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON content")
	}
	return nil
}

func sameStrings(left, right []string) bool {
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

func sameNumbers(left, right []float64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !sameNumber(left[index], right[index]) {
			return false
		}
	}
	return true
}

func sameNumber(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9*math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
}

func sameEvidenceSHA(prefixed, plain string) bool { return trimSHA(prefixed) == plain }

func trimSHA(value string) string {
	if len(value) == len("sha256:")+sha256.Size*2 && value[:len("sha256:")] == "sha256:" {
		return value[len("sha256:"):]
	}
	return value
}
