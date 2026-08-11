// Command generic-holdout-crossover combines two independently captured
// structural-measurement batches into preregistered reciprocal AB/BA blocks.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	requiredBatches = 2
	pairsPerBatch   = 8
	requiredBlocks  = 8
)

type repeatedFlag []string

func (values *repeatedFlag) String() string { return strings.Join(*values, ",") }
func (values *repeatedFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type specification struct {
	SchemaVersion         string          `json:"schemaVersion"`
	EvidenceSchemaVersion string          `json:"evidenceSchemaVersion"`
	WorkItem              string          `json:"workItem"`
	Subject               json.RawMessage `json:"subject"`
	Runner                struct {
		MeasurementBatches int `json:"measurementBatches"`
		PairsPerBatch      int `json:"pairsPerBatch"`
		ReciprocalBlocks   int `json:"reciprocalBlocks"`
	} `json:"runner"`
	Qualification struct {
		MinimumMeanSavedMS                  float64 `json:"minimumMeanSavedMs"`
		MinimumReductionRatio               float64 `json:"minimumReductionRatio"`
		PositiveLower95Bound                bool    `json:"positiveLower95Bound"`
		MinimumPositiveBlocks               int     `json:"minimumPositiveBlocks"`
		RequiredOutputsIdentical            bool    `json:"requiredOutputsIdentical"`
		MaximumProductAttributableFailures  int     `json:"maximumProductAttributableFailures"`
		FailedOrTimedOutObservationsDiscard bool    `json:"failedOrTimedOutObservationsMayBeDiscarded"`
		MeasuredExecutionShapeMustBeStable  bool    `json:"measuredExecutionShapeMustBeStable"`
		TargetWarmupShapeMustMatchPairs     bool    `json:"targetWarmupExecutionShapeMustMatchMeasuredPairs"`
		ExactTaskPathsMustBePresent         bool    `json:"exactTaskPathsMustBePresent"`
		BothBatchFallbacksMustPass          bool    `json:"bothBatchFallbacksMustPass"`
	} `json:"qualification"`
	Boundaries json.RawMessage `json:"boundaries"`
}

type task struct {
	Path    string `json:"path"`
	Outcome string `json:"outcome"`
}

type taskOutcomes struct {
	Total             int    `json:"total"`
	FingerprintSHA256 string `json:"fingerprintSha256"`
	Tasks             []task `json:"tasks"`
}

type warmup struct {
	Phase        string       `json:"phase"`
	TaskOutcomes taskOutcomes `json:"taskOutcomes"`
}

type observation struct {
	Pair                          int          `json:"pair"`
	Order                         string       `json:"order"`
	ControlDurationMS             int64        `json:"controlDurationMs"`
	CandidateDurationMS           int64        `json:"candidateDurationMs"`
	SavedMS                       int64        `json:"savedMs"`
	ControlRequiredOutputSHA256   string       `json:"controlRequiredOutputSha256"`
	CandidateRequiredOutputSHA256 string       `json:"candidateRequiredOutputSha256"`
	RequiredOutputCount           int          `json:"requiredOutputCount"`
	ProductAttributableFailure    bool         `json:"productAttributableFailure"`
	ControlTaskOutcomes           taskOutcomes `json:"controlTaskOutcomes"`
	CandidateTaskOutcomes         taskOutcomes `json:"candidateTaskOutcomes"`
}

type batchEvidence struct {
	SchemaVersion string `json:"schemaVersion"`
	CapturedAt    string `json:"capturedAt"`
	Subject       struct {
		RepositoryID       string `json:"repositoryId"`
		RepositoryRevision string `json:"repositoryRevision"`
		PipelineClass      string `json:"pipelineClass"`
	} `json:"subject"`
	Execution struct {
		BuildOptRevision string   `json:"buildoptRevision"`
		ExecutableSHA256 string   `json:"executableSha256"`
		ControlWarmups   []warmup `json:"controlWarmups"`
		CandidateWarmups []warmup `json:"candidateWarmups"`
	} `json:"execution"`
	Observations []observation `json:"observations"`
	Fallback     struct {
		Mode            string `json:"mode"`
		Reason          string `json:"reason"`
		BuildSuccessful bool   `json:"buildSuccessful"`
	} `json:"fallback"`
}

type block struct {
	Block              int     `json:"block"`
	Batch              int     `json:"batch"`
	ControlFirstPair   int     `json:"controlFirstPair"`
	CandidateFirstPair int     `json:"candidateFirstPair"`
	ControlMeanMS      float64 `json:"controlMeanMs"`
	CandidateMeanMS    float64 `json:"candidateMeanMs"`
	SavedMS            float64 `json:"savedMs"`
}

type taskDifference struct {
	FingerprintSHA256 string   `json:"fingerprintSha256"`
	Occurrences       int      `json:"occurrences"`
	AddedPaths        []string `json:"addedPaths,omitempty"`
	RemovedPaths      []string `json:"removedPaths,omitempty"`
	ChangedOutcomes   []string `json:"changedOutcomes,omitempty"`
}

type stability struct {
	Observed             bool             `json:"observed"`
	Stable               bool             `json:"stable"`
	ReferenceFingerprint string           `json:"referenceFingerprintSha256,omitempty"`
	Variants             []taskDifference `json:"variants,omitempty"`
}

type result struct {
	Blocks              int       `json:"blocks"`
	ControlMeanMS       float64   `json:"controlMeanMs"`
	CandidateMeanMS     float64   `json:"candidateMeanMs"`
	MeanSavedMS         float64   `json:"meanSavedMs"`
	ReductionRatio      float64   `json:"reductionRatio"`
	Interval95SavedMS   []float64 `json:"interval95SavedMs"`
	PositiveBlocks      int       `json:"positiveBlocks"`
	OutputsIdentical    bool      `json:"outputsIdentical"`
	FallbacksSuccessful bool      `json:"fallbacksSuccessful"`
	TargetShapeStable   bool      `json:"targetShapeStable"`
	Qualified           bool      `json:"qualified"`
	Decision            string    `json:"decision"`
}

type output struct {
	SchemaVersion string `json:"schemaVersion"`
	WorkItem      string `json:"workItem"`
	CapturedAt    string `json:"capturedAt"`
	BuildOpt      struct {
		Revision         string `json:"revision"`
		ExecutableSHA256 string `json:"executableSha256"`
	} `json:"buildopt"`
	Subject       json.RawMessage `json:"subject"`
	SourceBatches []struct {
		Batch  int    `json:"batch"`
		SHA256 string `json:"sha256"`
	} `json:"sourceBatches"`
	Blocks          []block `json:"blocks"`
	TargetStability struct {
		Control   stability `json:"control"`
		Candidate stability `json:"candidate"`
	} `json:"targetStability"`
	Result        result          `json:"result"`
	Qualification json.RawMessage `json:"qualification"`
	Boundaries    json.RawMessage `json:"boundaries"`
}

func main() {
	var batches repeatedFlag
	specPath := flag.String("spec", "", "preregistered v5 specification")
	capturedAt := flag.String("captured-at", "", "RFC3339 capture time")
	flag.Var(&batches, "batch", "structural evidence batch; repeat exactly twice")
	flag.Parse()
	if flag.NArg() != 0 || *specPath == "" || *capturedAt == "" || len(batches) != requiredBatches {
		fail("usage: generic-holdout-crossover --spec PATH --captured-at RFC3339 --batch PATH --batch PATH")
	}
	if _, err := time.Parse(time.RFC3339, *capturedAt); err != nil {
		fail("capture time is invalid: %v", err)
	}
	var spec specification
	readJSON(*specPath, &spec)
	if spec.SchemaVersion != "buildopt.specs/poc-generic-holdout/v5" ||
		spec.EvidenceSchemaVersion != "buildopt.evidence/poc-generic-holdout/v5" ||
		spec.Runner.MeasurementBatches != requiredBatches || spec.Runner.PairsPerBatch != pairsPerBatch ||
		spec.Runner.ReciprocalBlocks != requiredBlocks || spec.Qualification.MinimumPositiveBlocks != requiredBlocks {
		fail("crossover specification is not the frozen v5 contract")
	}

	loaded := make([]batchEvidence, requiredBatches)
	rawBatches := make([][]byte, requiredBatches)
	for index, path := range batches {
		raw, err := os.ReadFile(path)
		if err != nil {
			fail("read batch %d: %v", index+1, err)
		}
		rawBatches[index] = raw
		if err := json.Unmarshal(raw, &loaded[index]); err != nil {
			fail("decode batch %d: %v", index+1, err)
		}
		validateBatch(index+1, loaded[index])
	}
	if loaded[0].Subject != loaded[1].Subject ||
		loaded[0].Execution.BuildOptRevision != loaded[1].Execution.BuildOptRevision ||
		loaded[0].Execution.ExecutableSHA256 != loaded[1].Execution.ExecutableSHA256 {
		fail("crossover batch bindings differ")
	}

	var document output
	document.SchemaVersion = spec.EvidenceSchemaVersion
	document.WorkItem = spec.WorkItem
	document.CapturedAt = *capturedAt
	document.BuildOpt.Revision = loaded[0].Execution.BuildOptRevision
	document.BuildOpt.ExecutableSHA256 = loaded[0].Execution.ExecutableSHA256
	document.Subject = spec.Subject
	document.Qualification = mustRaw(spec.Qualification)
	document.Boundaries = spec.Boundaries
	for index, raw := range rawBatches {
		digest := sha256.Sum256(raw)
		document.SourceBatches = append(document.SourceBatches, struct {
			Batch  int    `json:"batch"`
			SHA256 string `json:"sha256"`
		}{Batch: index + 1, SHA256: hex.EncodeToString(digest[:])})
	}

	stableOutput := ""
	outputCount := 0
	outputsIdentical := true
	fallbacksSuccessful := true
	blockNumber := 0
	for batchIndex, batch := range loaded {
		fallbacksSuccessful = fallbacksSuccessful && batch.Fallback.Mode == "FULL_GRAPH" &&
			batch.Fallback.BuildSuccessful && batch.Fallback.Reason != ""
		for pairIndex := 0; pairIndex < len(batch.Observations); pairIndex += 2 {
			first := batch.Observations[pairIndex]
			second := batch.Observations[pairIndex+1]
			blockNumber++
			controlMean := float64(first.ControlDurationMS+second.ControlDurationMS) / 2
			candidateMean := float64(first.CandidateDurationMS+second.CandidateDurationMS) / 2
			document.Blocks = append(document.Blocks, block{
				Block: blockNumber, Batch: batchIndex + 1,
				ControlFirstPair: first.Pair, CandidateFirstPair: second.Pair,
				ControlMeanMS: controlMean, CandidateMeanMS: candidateMean,
				SavedMS: controlMean - candidateMean,
			})
			for _, current := range []observation{first, second} {
				if current.ControlRequiredOutputSHA256 != current.CandidateRequiredOutputSHA256 || current.RequiredOutputCount <= 0 {
					outputsIdentical = false
				}
				if stableOutput == "" {
					stableOutput = current.ControlRequiredOutputSHA256
					outputCount = current.RequiredOutputCount
				} else if stableOutput != current.ControlRequiredOutputSHA256 || outputCount != current.RequiredOutputCount {
					outputsIdentical = false
				}
			}
		}
	}

	document.TargetStability.Control = analyzeStability(loaded, false)
	document.TargetStability.Candidate = analyzeStability(loaded, true)
	document.Result = calculateResult(document.Blocks, spec, outputsIdentical, fallbacksSuccessful,
		document.TargetStability.Control.Stable && document.TargetStability.Candidate.Stable)
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		fail("encode crossover evidence: %v", err)
	}
	_, _ = os.Stdout.Write(append(raw, '\n'))
}

func validateBatch(number int, batch batchEvidence) {
	if batch.SchemaVersion != "buildopt.evidence/structural-profile-qualification/v1" ||
		len(batch.Observations) != pairsPerBatch || len(batch.Execution.ControlWarmups) != 4 ||
		len(batch.Execution.CandidateWarmups) != 4 {
		fail("batch %d does not contain the frozen structural shape", number)
	}
	for index, observation := range batch.Observations {
		expected := "CONTROL_FIRST"
		if index%2 == 1 {
			expected = "CANDIDATE_FIRST"
		}
		if observation.Pair != index+1 || observation.Order != expected ||
			observation.ControlDurationMS <= 0 || observation.CandidateDurationMS <= 0 ||
			observation.SavedMS != observation.ControlDurationMS-observation.CandidateDurationMS ||
			observation.ProductAttributableFailure {
			fail("batch %d observation %d is invalid", number, index+1)
		}
		validateTasks(number, "control pair", observation.ControlTaskOutcomes)
		validateTasks(number, "candidate pair", observation.CandidateTaskOutcomes)
	}
	for _, arm := range [][]warmup{batch.Execution.ControlWarmups, batch.Execution.CandidateWarmups} {
		for _, item := range arm {
			validateTasks(number, "warm-up "+item.Phase, item.TaskOutcomes)
		}
	}
}

func validateTasks(batch int, label string, outcomes taskOutcomes) {
	if outcomes.Total <= 0 || len(outcomes.Tasks) != outcomes.Total || len(outcomes.FingerprintSHA256) != 64 {
		fail("batch %d %s exact task evidence is incomplete", batch, label)
	}
	previous := ""
	for _, item := range outcomes.Tasks {
		if !strings.HasPrefix(item.Path, ":") || (previous != "" && item.Path <= previous) {
			fail("batch %d %s task paths are invalid", batch, label)
		}
		previous = item.Path
	}
}

func analyzeStability(batches []batchEvidence, candidate bool) stability {
	var samples []taskOutcomes
	for _, batch := range batches {
		warmups := batch.Execution.ControlWarmups
		if candidate {
			warmups = batch.Execution.CandidateWarmups
		}
		samples = append(samples, warmups[2].TaskOutcomes, warmups[3].TaskOutcomes)
		for _, observation := range batch.Observations {
			outcomes := observation.ControlTaskOutcomes
			if candidate {
				outcomes = observation.CandidateTaskOutcomes
			}
			samples = append(samples, outcomes)
		}
	}
	answer := stability{Observed: len(samples) > 0}
	if len(samples) == 0 {
		return answer
	}
	answer.ReferenceFingerprint = samples[0].FingerprintSHA256
	counts := map[string]int{}
	representatives := map[string]taskOutcomes{}
	for _, sample := range samples {
		counts[sample.FingerprintSHA256]++
		if _, exists := representatives[sample.FingerprintSHA256]; !exists {
			representatives[sample.FingerprintSHA256] = sample
		}
	}
	answer.Stable = len(counts) == 1
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		difference := compareTasks(samples[0], representatives[key])
		difference.FingerprintSHA256 = key
		difference.Occurrences = counts[key]
		answer.Variants = append(answer.Variants, difference)
	}
	return answer
}

func compareTasks(reference, current taskOutcomes) taskDifference {
	left := map[string]string{}
	right := map[string]string{}
	for _, item := range reference.Tasks {
		left[item.Path] = item.Outcome
	}
	for _, item := range current.Tasks {
		right[item.Path] = item.Outcome
	}
	var difference taskDifference
	for path, outcome := range right {
		previous, exists := left[path]
		if !exists {
			difference.AddedPaths = append(difference.AddedPaths, path)
		} else if previous != outcome {
			difference.ChangedOutcomes = append(difference.ChangedOutcomes, path+":"+previous+"->"+outcome)
		}
	}
	for path := range left {
		if _, exists := right[path]; !exists {
			difference.RemovedPaths = append(difference.RemovedPaths, path)
		}
	}
	sort.Strings(difference.AddedPaths)
	sort.Strings(difference.RemovedPaths)
	sort.Strings(difference.ChangedOutcomes)
	return difference
}

func calculateResult(blocks []block, spec specification, outputs, fallbacks, stable bool) result {
	if len(blocks) != requiredBlocks {
		fail("crossover produced %d blocks instead of %d", len(blocks), requiredBlocks)
	}
	saved := make([]float64, len(blocks))
	controlTotal := 0.0
	candidateTotal := 0.0
	positive := 0
	for index, block := range blocks {
		controlTotal += block.ControlMeanMS
		candidateTotal += block.CandidateMeanMS
		saved[index] = block.SavedMS
		if block.SavedMS > 0 {
			positive++
		}
	}
	meanSaved := (controlTotal - candidateTotal) / float64(len(blocks))
	ratio := (controlTotal - candidateTotal) / controlTotal
	interval := bootstrap95(saved)
	qualified := meanSaved >= spec.Qualification.MinimumMeanSavedMS &&
		ratio >= spec.Qualification.MinimumReductionRatio && interval[0] > 0 &&
		positive == spec.Qualification.MinimumPositiveBlocks && outputs && fallbacks && stable
	decision := "RETAIN_NATIVE_GRADLE"
	if qualified {
		decision = "REVIEW_STRUCTURAL_PROFILE"
	}
	return result{
		Blocks: len(blocks), ControlMeanMS: controlTotal / float64(len(blocks)),
		CandidateMeanMS: candidateTotal / float64(len(blocks)), MeanSavedMS: meanSaved,
		ReductionRatio: ratio, Interval95SavedMS: interval, PositiveBlocks: positive,
		OutputsIdentical: outputs, FallbacksSuccessful: fallbacks,
		TargetShapeStable: stable, Qualified: qualified, Decision: decision,
	}
}

func bootstrap95(saved []float64) []float64 {
	means := make([]float64, 4096)
	for sample := range means {
		state := uint64(2654435761*uint64(sample+1)) % (uint64(1) << 32)
		total := 0.0
		for range requiredBlocks {
			state = (1664525*state + 1013904223) % (uint64(1) << 32)
			total += saved[int(state/536870912)]
		}
		means[sample] = total / requiredBlocks
	}
	sort.Float64s(means)
	return []float64{means[102], means[3993]}
}

func readJSON(path string, target any) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fail("read %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		fail("decode %s: %v", path, err)
	}
}

func mustRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		fail("encode specification binding: %v", err)
	}
	return raw
}

func fail(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
