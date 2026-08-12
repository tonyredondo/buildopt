package profilediscovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"time"
)

const (
	// BalancedStructuralEvidenceSchema aggregates two independently captured v1
	// measurements without changing either immutable source document.
	BalancedStructuralEvidenceSchema = "buildopt.evidence/structural-profile-balanced-qualification/v2"
	balancedCaptureCount             = 2
	balancedBlockCount               = 8
	balancedMinimumPositiveBlocks    = 6
)

// BalancedStructuralOptions identifies two repository-relative v1 captures
// produced by the same installed BuildOpt executable and structural plan.
type BalancedStructuralOptions struct {
	RepositoryRoot string
	EvidencePaths  []string
	CapturedAt     time.Time
}

type balancedStructuralEvidence struct {
	SchemaVersion  string                      `json:"schemaVersion"`
	EvidenceState  string                      `json:"evidenceState"`
	CapturedAt     string                      `json:"capturedAt"`
	Subject        structuralSubject           `json:"subject"`
	SourceBindings structuralSourceBindings    `json:"sourceBindings"`
	Plan           AnalysisPlan                `json:"plan"`
	Execution      balancedStructuralExecution `json:"execution"`
	Captures       []balancedCapture           `json:"captures"`
	Blocks         []balancedBlock             `json:"blocks"`
	Qualification  balancedQualification       `json:"qualification"`
	Result         balancedResult              `json:"result"`
	Boundaries     structuralBoundaries        `json:"boundaries"`
}

type balancedStructuralExecution struct {
	CandidateSurface         string   `json:"candidateSurface"`
	BuildOptRevision         string   `json:"buildoptRevision"`
	ExecutableSHA256         string   `json:"executableSha256"`
	Mechanisms               []string `json:"mechanisms"`
	GradleOptions            []string `json:"gradleOptions"`
	LauncherOverheadIncluded bool     `json:"launcherOverheadIncluded"`
	CaptureCount             int      `json:"captureCount"`
	PairsPerCapture          int      `json:"pairsPerCapture"`
	BlockCount               int      `json:"blockCount"`
}

type balancedCapture struct {
	Index         int              `json:"index"`
	Path          string           `json:"path"`
	SHA256        string           `json:"sha256"`
	CapturedAt    string           `json:"capturedAt"`
	EvidenceState string           `json:"evidenceState"`
	Result        structuralResult `json:"result"`
}

type balancedBlock struct {
	CaptureIndex       int     `json:"captureIndex"`
	Block              int     `json:"block"`
	ControlFirstPair   int     `json:"controlFirstPair"`
	CandidateFirstPair int     `json:"candidateFirstPair"`
	ControlMeanMS      float64 `json:"controlMeanMs"`
	CandidateMeanMS    float64 `json:"candidateMeanMs"`
	SavedMS            float64 `json:"savedMs"`
}

type balancedQualification struct {
	MinimumMeanSavedMS        float64 `json:"minimumMeanSavedMs"`
	MinimumReductionRatio     float64 `json:"minimumReductionRatio"`
	PositiveLower95Bound      bool    `json:"positiveLower95Bound"`
	MinimumPositiveBlocks     int     `json:"minimumPositiveBlocks"`
	PositiveMedian            bool    `json:"positiveMedian"`
	CandidateP95NotRegressive bool    `json:"candidateP95NotRegressive"`
	RequiredOutputsIdentical  bool    `json:"requiredOutputsIdentical"`
	ExecutionShapeStable      bool    `json:"executionShapeStable"`
	FullGraphFallbackRequired bool    `json:"fullGraphFallbackRequired"`
}

type balancedResult struct {
	Captures                  int       `json:"captures"`
	Pairs                     int       `json:"pairs"`
	Blocks                    int       `json:"blocks"`
	ControlMeanMS             float64   `json:"controlMeanMs"`
	CandidateMeanMS           float64   `json:"candidateMeanMs"`
	MeanSavedMS               float64   `json:"meanSavedMs"`
	ReductionRatio            float64   `json:"reductionRatio"`
	MedianBlockSavedMS        float64   `json:"medianBlockSavedMs"`
	Interval95BlockSavedMS    []float64 `json:"interval95BlockSavedMs"`
	PositiveBlocks            int       `json:"positiveBlocks"`
	ControlP95MS              float64   `json:"controlP95Ms"`
	CandidateP95MS            float64   `json:"candidateP95Ms"`
	ControlFirstMeanSavedMS   float64   `json:"controlFirstMeanSavedMs"`
	CandidateFirstMeanSavedMS float64   `json:"candidateFirstMeanSavedMs"`
	OrderEffectMS             float64   `json:"orderEffectMs"`
	RequiredOutputsIdentical  bool      `json:"requiredOutputsIdentical"`
	ExecutionShapeStable      bool      `json:"executionShapeStable"`
	FallbacksSuccessful       bool      `json:"fallbacksSuccessful"`
	Qualified                 bool      `json:"qualified"`
	Decision                  string    `json:"decision"`
}

// RenderBalancedStructuralEvidence independently validates two v1 captures,
// constructs four AB/BA blocks per capture, and applies the repository-neutral
// v2 materiality, uncertainty, tail, correctness, and fallback gates.
func RenderBalancedStructuralEvidence(options BalancedStructuralOptions) ([]byte, bool, error) {
	if len(options.EvidencePaths) != balancedCaptureCount || options.CapturedAt.IsZero() {
		return nil, false, errors.New("balanced structural qualification requires two captures and a capture time")
	}
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return nil, false, fmt.Errorf("resolve repository root: %w", err)
	}

	var captures [balancedCaptureCount]structuralEvidence
	bindings := make([]InputBinding, balancedCaptureCount)
	refs := make([]balancedCapture, balancedCaptureCount)
	for index, path := range options.EvidencePaths {
		raw, binding, readErr := readInput(root, path, "STRUCTURAL_QUALIFICATION_CAPTURE")
		if readErr != nil {
			return nil, false, readErr
		}
		if decodeErr := decodeStrictJSON(raw, &captures[index]); decodeErr != nil {
			return nil, false, fmt.Errorf("decode structural capture %d: %w", index+1, decodeErr)
		}
		analysis := AnalysisReport{Subject: Subject{
			RepositoryID:  captures[index].Subject.RepositoryID,
			PipelineClass: captures[index].Subject.PipelineClass,
		}, Decision: DecisionMeasure, Plan: &captures[index].Plan}
		if validateErr := validateStructuralCaptureEvidence(captures[index], analysis); validateErr != nil {
			return nil, false, fmt.Errorf("validate structural capture %d: %w", index+1, validateErr)
		}
		bindings[index] = binding
		refs[index] = balancedCapture{
			Index: index + 1, Path: path, SHA256: trimSHA(binding.SHA256),
			CapturedAt: captures[index].CapturedAt, EvidenceState: captures[index].EvidenceState,
			Result: captures[index].Result,
		}
	}
	if refs[0].SHA256 == refs[1].SHA256 || refs[0].CapturedAt == refs[1].CapturedAt {
		return nil, false, errors.New("balanced structural captures are not independent")
	}
	if err := validateBalancedCaptureIdentity(captures[0], captures[1]); err != nil {
		return nil, false, err
	}

	blocks := make([]balancedBlock, 0, balancedBlockCount)
	controlDurations := make([]float64, 0, structuralPairCount*balancedCaptureCount)
	candidateDurations := make([]float64, 0, structuralPairCount*balancedCaptureCount)
	controlFirstSaved := make([]float64, 0, balancedBlockCount)
	candidateFirstSaved := make([]float64, 0, balancedBlockCount)
	for captureIndex, capture := range captures {
		for pair := 0; pair < structuralPairCount; pair++ {
			observation := capture.Observations[pair]
			controlDurations = append(controlDurations, float64(observation.ControlDurationMS))
			candidateDurations = append(candidateDurations, float64(observation.CandidateDurationMS))
			if observation.Order == "CONTROL_FIRST" {
				controlFirstSaved = append(controlFirstSaved, float64(observation.SavedMS))
			} else {
				candidateFirstSaved = append(candidateFirstSaved, float64(observation.SavedMS))
			}
		}
		for block := 0; block < structuralPairCount/2; block++ {
			first := capture.Observations[block*2]
			second := capture.Observations[block*2+1]
			blocks = append(blocks, balancedBlock{
				CaptureIndex: captureIndex + 1, Block: block + 1,
				ControlFirstPair: first.Pair, CandidateFirstPair: second.Pair,
				ControlMeanMS:   float64(first.ControlDurationMS+second.ControlDurationMS) / 2,
				CandidateMeanMS: float64(first.CandidateDurationMS+second.CandidateDurationMS) / 2,
				SavedMS:         float64(first.SavedMS+second.SavedMS) / 2,
			})
		}
	}
	result := calculateBalancedResult(blocks, controlDurations, candidateDurations, controlFirstSaved, candidateFirstSaved)
	state := "INCONCLUSIVE"
	if result.Qualified {
		state = "QUALIFIED"
	}
	evidence := balancedStructuralEvidence{
		SchemaVersion: BalancedStructuralEvidenceSchema, EvidenceState: state,
		CapturedAt: options.CapturedAt.UTC().Truncate(time.Second).Format(time.RFC3339),
		Subject:    captures[0].Subject, SourceBindings: captures[0].SourceBindings,
		Plan: captures[0].Plan,
		Execution: balancedStructuralExecution{
			CandidateSurface:         captures[0].Execution.CandidateSurface,
			BuildOptRevision:         captures[0].Execution.BuildOptRevision,
			ExecutableSHA256:         captures[0].Execution.ExecutableSHA256,
			Mechanisms:               append([]string(nil), captures[0].Execution.Mechanisms...),
			GradleOptions:            append([]string(nil), captures[0].Execution.GradleOptions...),
			LauncherOverheadIncluded: true, CaptureCount: balancedCaptureCount,
			PairsPerCapture: structuralPairCount, BlockCount: balancedBlockCount,
		},
		Captures: refs, Blocks: blocks,
		Qualification: balancedQualification{
			MinimumMeanSavedMS: structuralMinimumSavedMS, MinimumReductionRatio: structuralMinimumRatio,
			PositiveLower95Bound: true, MinimumPositiveBlocks: balancedMinimumPositiveBlocks,
			PositiveMedian: true, CandidateP95NotRegressive: true,
			RequiredOutputsIdentical: true, ExecutionShapeStable: true,
			FullGraphFallbackRequired: true,
		},
		Result:     result,
		Boundaries: structuralBoundaries{ProofOfConcept: true},
	}
	raw, err := jsonMarshalIndent(evidence)
	if err != nil {
		return nil, false, err
	}
	return raw, result.Qualified, nil
}

func validateBalancedCaptureIdentity(left, right structuralEvidence) error {
	if left.Subject != right.Subject || left.SourceBindings != right.SourceBindings ||
		!sameAnalysisPlan(left.Plan, right.Plan) ||
		left.Execution.CandidateSurface != right.Execution.CandidateSurface ||
		left.Execution.BuildOptRevision != right.Execution.BuildOptRevision ||
		left.Execution.ExecutableSHA256 != right.Execution.ExecutableSHA256 ||
		left.Execution.LauncherOverheadIncluded != right.Execution.LauncherOverheadIncluded ||
		!sameStrings(left.Execution.Mechanisms, right.Execution.Mechanisms) ||
		!sameStrings(left.Execution.GradleOptions, right.Execution.GradleOptions) {
		return errors.New("balanced structural capture identity drift")
	}
	return nil
}

func calculateBalancedResult(blocks []balancedBlock, control, candidate, controlFirst, candidateFirst []float64) balancedResult {
	blockSaved := make([]float64, len(blocks))
	positive := 0
	for index, block := range blocks {
		blockSaved[index] = block.SavedMS
		if block.SavedMS > 0 {
			positive++
		}
	}
	controlMean := meanFloats(control)
	candidateMean := meanFloats(candidate)
	meanSaved := controlMean - candidateMean
	interval := balancedBootstrap95(blockSaved)
	median := medianFloats(blockSaved)
	controlP95 := nearestRank(control, .95)
	candidateP95 := nearestRank(candidate, .95)
	result := balancedResult{
		Captures: balancedCaptureCount, Pairs: len(control), Blocks: len(blocks),
		ControlMeanMS: controlMean, CandidateMeanMS: candidateMean,
		MeanSavedMS: meanSaved, ReductionRatio: meanSaved / controlMean,
		MedianBlockSavedMS: median, Interval95BlockSavedMS: interval,
		PositiveBlocks: positive, ControlP95MS: controlP95, CandidateP95MS: candidateP95,
		ControlFirstMeanSavedMS:   meanFloats(controlFirst),
		CandidateFirstMeanSavedMS: meanFloats(candidateFirst),
		RequiredOutputsIdentical:  true, ExecutionShapeStable: true, FallbacksSuccessful: true,
	}
	result.OrderEffectMS = result.ControlFirstMeanSavedMS - result.CandidateFirstMeanSavedMS
	result.Qualified = result.MeanSavedMS >= structuralMinimumSavedMS &&
		result.ReductionRatio >= structuralMinimumRatio && result.MedianBlockSavedMS > 0 &&
		result.Interval95BlockSavedMS[0] > 0 && result.PositiveBlocks >= balancedMinimumPositiveBlocks &&
		result.CandidateP95MS <= result.ControlP95MS
	result.Decision = "RETAIN_NATIVE_GRADLE"
	if result.Qualified {
		result.Decision = "QUALIFY_BALANCED_STRUCTURAL_PROFILE"
	}
	return result
}

func meanFloats(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func medianFloats(values []float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	return (ordered[middle-1] + ordered[middle]) / 2
}

func nearestRank(values []float64, percentile float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	rank := int(math.Ceil(percentile*float64(len(ordered)))) - 1
	return ordered[rank]
}

func balancedBootstrap95(saved []float64) []float64 {
	means := make([]float64, 4096)
	bucketWidth := (uint64(1) << 32) / uint64(len(saved))
	for sample := range means {
		state := uint64(2246822519*uint64(sample+1)) % (uint64(1) << 32)
		total := 0.0
		for draw := 0; draw < len(saved); draw++ {
			state = (1664525*state + 1013904223) % (uint64(1) << 32)
			total += saved[int(state/bucketWidth)]
		}
		means[sample] = total / float64(len(saved))
	}
	sort.Float64s(means)
	return []float64{means[102], means[3993]}
}

func jsonMarshalIndent(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render balanced structural evidence: %w", err)
	}
	return append(raw, '\n'), nil
}
