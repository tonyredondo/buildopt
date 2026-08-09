package profilediscovery

import (
	"bytes"
	"crypto/sha256"
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
)

const (
	StructuralEvidenceSchema = "buildopt.evidence/structural-profile-qualification/v1"
	StructuralProfileSchema  = "buildopt.poc/qualified-profile/v4"
	StructuralProfileID      = "qualified-structural-build-impact"
	structuralPairCount      = 8
	structuralMinimumSavedMS = 500.0
	structuralMinimumRatio   = 0.02
)

// StructuralOptions binds an independently measured structural optimization
// to the exact repository graph from which the candidate was derived.
type StructuralOptions struct {
	RepositoryRoot string
	ManifestPath   string
	GraphPath      string
	GeneratedPath  string
	EvidencePath   string
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
	ManifestSHA256       string `json:"manifestSha256"`
	GraphSHA256          string `json:"graphSha256"`
	GeneratedSHA256      string `json:"generatedManifestSha256"`
	SourceEvidenceSHA256 string `json:"sourceEvidenceSha256"`
}

type structuralExecution struct {
	CandidateSurface         string   `json:"candidateSurface"`
	BuildOptRevision         string   `json:"buildoptRevision"`
	ExecutableSHA256         string   `json:"executableSha256,omitempty"`
	Mechanisms               []string `json:"mechanisms"`
	GradleOptions            []string `json:"gradleOptions"`
	LauncherOverheadIncluded bool     `json:"launcherOverheadIncluded"`
}

type structuralObservation struct {
	Pair                          int    `json:"pair"`
	Order                         string `json:"order"`
	ControlDurationMS             int64  `json:"controlDurationMs"`
	CandidateDurationMS           int64  `json:"candidateDurationMs"`
	SavedMS                       int64  `json:"savedMs"`
	ControlRequiredOutputSHA256   string `json:"controlRequiredOutputSha256"`
	CandidateRequiredOutputSHA256 string `json:"candidateRequiredOutputSha256"`
	RequiredOutputCount           int    `json:"requiredOutputCount"`
	ProductAttributableFailure    bool   `json:"productAttributableFailure"`
}

type structuralFallback struct {
	Mode            string `json:"mode"`
	Reason          string `json:"reason"`
	BuildSuccessful bool   `json:"buildSuccessful"`
}

type structuralResult struct {
	Pairs             int       `json:"pairs"`
	ControlMeanMS     float64   `json:"controlMeanMs"`
	CandidateMeanMS   float64   `json:"candidateMeanMs"`
	MeanSavedMS       float64   `json:"meanSavedMs"`
	ReductionRatio    float64   `json:"reductionRatio"`
	Interval95SavedMS []float64 `json:"interval95SavedMs"`
	PositivePairs     int       `json:"positivePairs"`
	Qualified         bool      `json:"qualified"`
	Decision          string    `json:"decision"`
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
}

// StructuralMeasurementOptions carries the independently observed values used
// to render evidence for the existing structural profile qualifier. It grants
// no activation or production authority.
type StructuralMeasurementOptions struct {
	CapturedAt           time.Time
	Analysis             AnalysisReport
	RepositoryRevision   string
	BuildOptRevision     string
	ExecutableSHA256     string
	SourceEvidenceSHA256 string
	GradleOptions        []string
	Observations         []StructuralMeasurementObservation
	FallbackReason       string
	FallbackSuccessful   bool
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
		}
	}
	result, err := calculateStructuralResult(observations)
	if err != nil {
		return nil, false, err
	}
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
			ManifestSHA256:       trimSHA(inputs["BUILD_IMPACT_MANIFEST"].SHA256),
			GraphSHA256:          trimSHA(inputs["BUILD_IMPACT_GRAPH"].SHA256),
			GeneratedSHA256:      trimSHA(inputs["GENERATED_MANIFEST"].SHA256),
			SourceEvidenceSHA256: options.SourceEvidenceSHA256,
		},
		Plan: *options.Analysis.Plan,
		Execution: structuralExecution{
			CandidateSurface:         "INSTALLED_BUILDOPT_STRUCTURAL_IMPACT_ONLY",
			BuildOptRevision:         options.BuildOptRevision,
			ExecutableSHA256:         options.ExecutableSHA256,
			Mechanisms:               []string{"BUILD_IMPACT"},
			GradleOptions:            append([]string(nil), options.GradleOptions...),
			LauncherOverheadIncluded: true,
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
	if evidence.SchemaVersion != StructuralEvidenceSchema || evidence.EvidenceState != "QUALIFIED" {
		return errors.New("structural qualification evidence is not qualified")
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
	calculated, err := calculateStructuralResult(evidence.Observations)
	if err != nil {
		return err
	}
	if !sameStructuralResult(calculated, evidence.Result) || !calculated.Qualified ||
		evidence.Result.Decision != "QUALIFY_STRUCTURAL_PROFILE" {
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
	}
	meanSaved := float64(controlTotal-candidateTotal) / structuralPairCount
	ratio := float64(controlTotal-candidateTotal) / float64(controlTotal)
	interval := structuralBootstrap95(saved)
	qualified := meanSaved >= structuralMinimumSavedMS && ratio >= structuralMinimumRatio &&
		interval[0] > 0 && positive == structuralPairCount
	decision := "RETAIN_NATIVE_GRADLE"
	if qualified {
		decision = "QUALIFY_STRUCTURAL_PROFILE"
	}
	return structuralResult{
		Pairs:             structuralPairCount,
		ControlMeanMS:     float64(controlTotal) / structuralPairCount,
		CandidateMeanMS:   float64(candidateTotal) / structuralPairCount,
		MeanSavedMS:       meanSaved,
		ReductionRatio:    ratio,
		Interval95SavedMS: interval,
		PositivePairs:     positive,
		Qualified:         qualified,
		Decision:          decision,
	}, nil
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
