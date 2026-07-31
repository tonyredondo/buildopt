package betabenchmark

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"sync"
	"syscall"
	"time"
)

const (
	resultSchemaVersion      = "buildopt.benchmarks/result/v1"
	observationSchemaVersion = "buildopt.benchmarks/observation/v1"
	resultFilename           = "result.json"
	observationsFilename     = "observations.jsonl"
	maximumResultBytes       = 8 << 20
)

type Result struct {
	SchemaVersion            string                  `json:"schemaVersion"`
	Qualification            string                  `json:"qualification"`
	BenchmarkDigest          string                  `json:"benchmarkDigest"`
	Seed                     int64                   `json:"seed"`
	StartedAt                string                  `json:"startedAt"`
	CompletedAt              string                  `json:"completedAt"`
	Hardware                 hardwareIdentity        `json:"hardware"`
	Cgroup                   cgroupIdentity          `json:"cgroup"`
	Components               components              `json:"components"`
	ActualObjectDistribution []distribution          `json:"actualObjectDistribution"`
	P50                      []percentile            `json:"p50"`
	P95                      []percentile            `json:"p95"`
	P99                      []percentile            `json:"p99"`
	Throughput               []throughput            `json:"throughput"`
	Errors                   []errorCount            `json:"errors"`
	Bytes                    []byteCount             `json:"bytes"`
	Recovery                 []recoveryObservation   `json:"recovery"`
	ReadinessTransitions     []readinessTransition   `json:"readinessTransitions"`
	FaultOutcomes            []faultOutcome          `json:"faultOutcomes"`
	Deviations               []string                `json:"deviations"`
	RawObservations          rawObservationsIdentity `json:"rawObservations"`
}

type hardwareIdentity struct {
	OperatingSystem string `json:"operatingSystem"`
	Architecture    string `json:"architecture"`
	CPUCount        int    `json:"cpuCount"`
}

type cgroupIdentity struct {
	CPUQuota     string `json:"cpuQuota"`
	MemoryLimit  string `json:"memoryLimit"`
	ObservedMode string `json:"observedMode"`
}

type distribution struct {
	Phase         string `json:"phase"`
	Clients       int    `json:"clients"`
	SizeBytes     int64  `json:"sizeBytes"`
	Operations    int64  `json:"operations"`
	ExpectedHits  int64  `json:"expectedHits"`
	RequestBytes  int64  `json:"requestBytes"`
	ResponseBytes int64  `json:"responseBytes"`
}

type percentile struct {
	Phase      string `json:"phase"`
	Clients    int    `json:"clients"`
	DurationNs int64  `json:"durationNs"`
	Samples    int64  `json:"samples"`
}

type throughput struct {
	Phase               string  `json:"phase"`
	Clients             int     `json:"clients"`
	OperationsPerSecond float64 `json:"operationsPerSecond"`
	BytesPerSecond      float64 `json:"bytesPerSecond"`
	DurationNs          int64   `json:"durationNs"`
}

type errorCount struct {
	Phase   string `json:"phase"`
	Clients int    `json:"clients"`
	Class   string `json:"class"`
	Count   int64  `json:"count"`
}

type byteCount struct {
	Phase         string `json:"phase"`
	Clients       int    `json:"clients"`
	RequestBytes  int64  `json:"requestBytes"`
	ResponseBytes int64  `json:"responseBytes"`
}

type recoveryObservation struct {
	ID         string `json:"id"`
	DurationNs int64  `json:"durationNs"`
	Status     string `json:"status"`
}

type readinessTransition struct {
	ObservedAt string `json:"observedAt"`
	From       string `json:"from"`
	To         string `json:"to"`
}

type faultOutcome struct {
	ID             string `json:"id"`
	ExpectedSafety string `json:"expectedSafety"`
	Status         string `json:"status"`
}

type rawObservationsIdentity struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"sizeBytes"`
}

type rawObservation struct {
	SchemaVersion string `json:"schemaVersion"`
	Sequence      int64  `json:"sequence"`
	Phase         string `json:"phase"`
	Clients       int    `json:"clients"`
	Operation     string `json:"operation"`
	ObjectIndex   int    `json:"objectIndex"`
	SizeBytes     int64  `json:"sizeBytes"`
	ExpectedHit   bool   `json:"expectedHit"`
	Status        int    `json:"status"`
	DurationNs    int64  `json:"durationNs"`
	RequestBytes  int64  `json:"requestBytes"`
	ResponseBytes int64  `json:"responseBytes"`
	ErrorClass    string `json:"errorClass,omitempty"`
}

type observationSink struct {
	mutex    sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	hash     hashWriter
	sequence int64
	count    int64
	size     int64
}

type hashWriter struct {
	sum func() []byte
}

type stratumSummary struct {
	phase         string
	clients       int
	startedAt     time.Time
	completedAt   time.Time
	latencies     []int64
	distributions map[int64]*distribution
	errors        map[string]int64
	requestBytes  int64
	responseBytes int64
}

func prepareOutputDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("benchmark output directory must be absolute")
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return errors.New("create benchmark output directory")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errors.New("benchmark output is not a private directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("benchmark output has the wrong owner")
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		return errors.New("benchmark output directory must be empty")
	}
	return nil
}

func newObservationSink(outputDirectory string) (*observationSink, error) {
	path := filepath.Join(outputDirectory, observationsFilename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, errors.New("create benchmark observation stream")
	}
	digest := sha256.New()
	return &observationSink{
		file:   file,
		writer: bufio.NewWriterSize(io.MultiWriter(file, digest), 1<<20),
		hash: hashWriter{
			sum: func() []byte {
				return digest.Sum(nil)
			},
		},
	}, nil
}

func (sink *observationSink) append(observation rawObservation) error {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	sink.sequence++
	observation.SchemaVersion = observationSchemaVersion
	observation.Sequence = sink.sequence
	content, err := json.Marshal(observation)
	if err != nil {
		return err
	}
	content = append(content, '\n')
	written, err := sink.writer.Write(content)
	sink.size += int64(written)
	if err != nil {
		return err
	}
	sink.count++
	return nil
}

func (sink *observationSink) close() (rawObservationsIdentity, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file == nil {
		return rawObservationsIdentity{}, errors.New(
			"benchmark observation stream is already closed",
		)
	}
	if err := sink.writer.Flush(); err != nil {
		return rawObservationsIdentity{}, err
	}
	if err := sink.file.Sync(); err != nil {
		return rawObservationsIdentity{}, err
	}
	if err := sink.file.Close(); err != nil {
		return rawObservationsIdentity{}, err
	}
	sink.file = nil
	return rawObservationsIdentity{
		Path:      observationsFilename,
		SHA256:    "sha256:" + hex.EncodeToString(sink.hash.sum()),
		Count:     sink.count,
		SizeBytes: sink.size,
	}, nil
}

func (sink *observationSink) abort() {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file != nil {
		_ = sink.file.Close()
		sink.file = nil
	}
}

func newStratumSummary(phaseID string, clients int) *stratumSummary {
	return &stratumSummary{
		phase:         phaseID,
		clients:       clients,
		distributions: make(map[int64]*distribution),
		errors:        make(map[string]int64),
	}
}

func (summary *stratumSummary) observe(observation rawObservation) {
	summary.latencies = append(summary.latencies, observation.DurationNs)
	item := summary.distributions[observation.SizeBytes]
	if item == nil {
		item = &distribution{
			Phase:     summary.phase,
			Clients:   summary.clients,
			SizeBytes: observation.SizeBytes,
		}
		summary.distributions[observation.SizeBytes] = item
	}
	item.Operations++
	if observation.ExpectedHit {
		item.ExpectedHits++
	}
	item.RequestBytes += observation.RequestBytes
	item.ResponseBytes += observation.ResponseBytes
	summary.requestBytes += observation.RequestBytes
	summary.responseBytes += observation.ResponseBytes
	if observation.ErrorClass != "" {
		summary.errors[observation.ErrorClass]++
	}
}

func summarizeStrata(
	summaries []*stratumSummary,
) (
	[]distribution,
	[]percentile,
	[]percentile,
	[]percentile,
	[]throughput,
	[]errorCount,
	[]byteCount,
) {
	var distributions []distribution
	var p50 []percentile
	var p95 []percentile
	var p99 []percentile
	var rates []throughput
	errorsByClass := []errorCount{}
	var bytesByStratum []byteCount
	for _, summary := range summaries {
		sizes := make([]int64, 0, len(summary.distributions))
		for size := range summary.distributions {
			sizes = append(sizes, size)
		}
		slices.Sort(sizes)
		for _, size := range sizes {
			distributions = append(
				distributions,
				*summary.distributions[size],
			)
		}
		sort.Slice(summary.latencies, func(first, second int) bool {
			return summary.latencies[first] < summary.latencies[second]
		})
		samples := int64(len(summary.latencies))
		p50 = append(p50, percentileFor(summary, 50, samples))
		p95 = append(p95, percentileFor(summary, 95, samples))
		p99 = append(p99, percentileFor(summary, 99, samples))
		duration := summary.completedAt.Sub(summary.startedAt)
		durationSeconds := duration.Seconds()
		if durationSeconds <= 0 {
			durationSeconds = 1e-9
		}
		rates = append(rates, throughput{
			Phase:               summary.phase,
			Clients:             summary.clients,
			OperationsPerSecond: float64(samples) / durationSeconds,
			BytesPerSecond: float64(
				summary.requestBytes+summary.responseBytes,
			) / durationSeconds,
			DurationNs: duration.Nanoseconds(),
		})
		errorClasses := make([]string, 0, len(summary.errors))
		for class := range summary.errors {
			errorClasses = append(errorClasses, class)
		}
		slices.Sort(errorClasses)
		for _, class := range errorClasses {
			errorsByClass = append(errorsByClass, errorCount{
				Phase:   summary.phase,
				Clients: summary.clients,
				Class:   class,
				Count:   summary.errors[class],
			})
		}
		bytesByStratum = append(bytesByStratum, byteCount{
			Phase:         summary.phase,
			Clients:       summary.clients,
			RequestBytes:  summary.requestBytes,
			ResponseBytes: summary.responseBytes,
		})
	}
	return distributions, p50, p95, p99, rates, errorsByClass, bytesByStratum
}

func percentileFor(
	summary *stratumSummary,
	percent int,
	samples int64,
) percentile {
	value := int64(0)
	if samples > 0 {
		index := (percent*int(samples)+99)/100 - 1
		value = summary.latencies[index]
	}
	return percentile{
		Phase:      summary.phase,
		Clients:    summary.clients,
		DurationNs: value,
		Samples:    samples,
	}
}

func writeResult(outputDirectory string, result Result) (string, error) {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	content = append(content, '\n')
	temporary, err := os.CreateTemp(outputDirectory, ".result-*.tmp")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return "", err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	target := filepath.Join(outputDirectory, resultFilename)
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", err
	}
	directory, err := os.Open(outputDirectory)
	if err != nil {
		return "", err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		return "", syncErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return target, nil
}

// ValidateResult verifies a smoke result and its exact raw observation stream.
func ValidateResult(manifestPath string, resultPath string) error {
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	info, err := os.Lstat(resultPath)
	stat, ownerAvailable := infoSyscallStat(info)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		!ownerAvailable ||
		stat.Uid != uint32(os.Geteuid()) ||
		info.Size() < 1 ||
		info.Size() > maximumResultBytes {
		return errors.New("benchmark result is not a bounded private file")
	}
	raw, err := os.ReadFile(resultPath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var result Result
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode benchmark result: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("benchmark result contains trailing JSON")
	}
	if result.SchemaVersion != resultSchemaVersion ||
		result.Qualification != "SMOKE" ||
		result.BenchmarkDigest != manifestDigest ||
		result.Seed != loaded.Seed ||
		result.Hardware.OperatingSystem == "" ||
		result.Hardware.Architecture == "" ||
		result.Hardware.CPUCount < 1 ||
		result.Components != loaded.Components ||
		len(result.ActualObjectDistribution) != 48 ||
		len(result.P50) != 12 ||
		len(result.P95) != 12 ||
		len(result.P99) != 12 ||
		len(result.Throughput) != 12 ||
		len(result.Bytes) != 12 ||
		len(result.Errors) != 0 ||
		len(result.Recovery) != 0 ||
		len(result.ReadinessTransitions) != 0 ||
		len(result.FaultOutcomes) != len(loaded.Faults) ||
		result.RawObservations.Path != observationsFilename ||
		result.RawObservations.Count != 1200 ||
		result.RawObservations.SizeBytes < 1 {
		return errors.New("benchmark smoke result is incomplete")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, result.StartedAt)
	if err != nil {
		return errors.New("benchmark result has invalid startedAt")
	}
	completedAt, err := time.Parse(time.RFC3339Nano, result.CompletedAt)
	if err != nil || completedAt.Before(startedAt) {
		return errors.New("benchmark result has invalid completedAt")
	}
	if err := validateSmokeSummary(loaded, result); err != nil {
		return err
	}
	observationsPath := filepath.Join(
		filepath.Dir(resultPath),
		result.RawObservations.Path,
	)
	observationsInfo, err := os.Lstat(observationsPath)
	observationsStat, observationsOwnerAvailable := infoSyscallStat(
		observationsInfo,
	)
	if err != nil ||
		!observationsInfo.Mode().IsRegular() ||
		observationsInfo.Mode().Perm() != 0o600 ||
		!observationsOwnerAvailable ||
		observationsStat.Uid != uint32(os.Geteuid()) {
		return errors.New("benchmark raw observations are not a private file")
	}
	observations, err := os.ReadFile(observationsPath)
	if err != nil || int64(len(observations)) != result.RawObservations.SizeBytes {
		return errors.New("benchmark raw observations are unavailable")
	}
	digest := sha256.Sum256(observations)
	if result.RawObservations.SHA256 !=
		"sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("benchmark raw observations digest mismatched")
	}
	lines := bytes.Count(observations, []byte{'\n'})
	if int64(lines) != result.RawObservations.Count {
		return errors.New("benchmark raw observation count mismatched")
	}
	if err := validateRawSmokeObservations(observations); err != nil {
		return err
	}
	return nil
}

func validateSmokeSummary(loaded manifest, result Result) error {
	expectedDeviations := []string{
		"SMOKE_PROFILE_SCALED_OBJECT_SIZES",
		"SMOKE_PROFILE_ONE_CYCLE_PER_STRATUM",
		"SMOKE_PROFILE_PHASE_DURATIONS_NOT_QUALIFYING",
		"FAULT_MATRIX_NOT_RUN",
		"GRADLE_FIXTURES_NOT_RUN",
		"RUNNER_QUALIFICATION_NOT_CLAIMED",
	}
	if !slices.Equal(result.Deviations, expectedDeviations) {
		return errors.New("benchmark smoke deviations drifted")
	}
	for index, fault := range result.FaultOutcomes {
		if fault.ID != loaded.Faults[index].ID ||
			fault.ExpectedSafety != loaded.Faults[index].ExpectedSafety ||
			fault.Status != "NOT_RUN_SMOKE" {
			return errors.New("benchmark smoke invented a fault outcome")
		}
	}

	distributionIndex := 0
	stratumIndex := 0
	for _, currentPhase := range loaded.Phases {
		for _, clients := range loaded.Clients {
			expectedHits := int64(
				100 * currentPhase.TargetHitPercent / 100,
			)
			actualHits := int64(0)
			requestBytes := int64(0)
			responseBytes := int64(0)
			for _, mix := range smokeObjectMix {
				item := result.ActualObjectDistribution[distributionIndex]
				distributionIndex++
				if item.Phase != currentPhase.ID ||
					item.Clients != clients ||
					item.SizeBytes != mix.SizeBytes ||
					item.Operations != int64(mix.Percent) ||
					item.ExpectedHits < 0 ||
					item.ExpectedHits > item.Operations ||
					item.ResponseBytes !=
						item.ExpectedHits*item.SizeBytes {
					return errors.New(
						"benchmark smoke object distribution drifted",
					)
				}
				if currentPhase.ID == "COLD" {
					if item.ExpectedHits != 0 ||
						item.RequestBytes !=
							item.Operations*item.SizeBytes {
						return errors.New(
							"benchmark smoke cold bytes drifted",
						)
					}
				} else if item.RequestBytes != 0 {
					return errors.New(
						"benchmark smoke read bytes drifted",
					)
				}
				actualHits += item.ExpectedHits
				requestBytes += item.RequestBytes
				responseBytes += item.ResponseBytes
			}
			if actualHits != expectedHits {
				return errors.New("benchmark smoke hit target drifted")
			}
			for _, values := range [][]percentile{
				result.P50,
				result.P95,
				result.P99,
			} {
				percentile := values[stratumIndex]
				if percentile.Phase != currentPhase.ID ||
					percentile.Clients != clients ||
					percentile.DurationNs < 0 ||
					percentile.Samples != 100 {
					return errors.New(
						"benchmark smoke percentile drifted",
					)
				}
			}
			rate := result.Throughput[stratumIndex]
			byteSummary := result.Bytes[stratumIndex]
			if rate.Phase != currentPhase.ID ||
				rate.Clients != clients ||
				rate.OperationsPerSecond <= 0 ||
				rate.BytesPerSecond < 0 ||
				rate.DurationNs <= 0 ||
				byteSummary.Phase != currentPhase.ID ||
				byteSummary.Clients != clients ||
				byteSummary.RequestBytes != requestBytes ||
				byteSummary.ResponseBytes != responseBytes {
				return errors.New("benchmark smoke stratum summary drifted")
			}
			stratumIndex++
		}
	}
	return nil
}

func validateRawSmokeObservations(content []byte) error {
	type stratumKey struct {
		phase   string
		clients int
	}
	counts := make(map[stratumKey]int)
	hits := make(map[stratumKey]int)
	reader := bufio.NewReader(bytes.NewReader(content))
	sequence := int64(0)
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		if err != nil {
			return errors.New("read benchmark raw observation")
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var observation rawObservation
		if err := decoder.Decode(&observation); err != nil {
			return errors.New("decode benchmark raw observation")
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return errors.New(
				"benchmark raw observation contains trailing JSON",
			)
		}
		sequence++
		if observation.SchemaVersion != observationSchemaVersion ||
			observation.Sequence != sequence ||
			!slices.Contains([]int{1, 8, 32}, observation.Clients) ||
			observation.ObjectIndex < 0 ||
			observation.ObjectIndex >= 100 ||
			observation.DurationNs < 0 ||
			observation.ErrorClass != "" ||
			!validSmokeObjectSize(observation.SizeBytes) {
			return errors.New("benchmark raw observation drifted")
		}
		key := stratumKey{
			phase:   observation.Phase,
			clients: observation.Clients,
		}
		switch observation.Phase {
		case "COLD":
			if observation.Operation != "PUT" ||
				observation.ExpectedHit ||
				observation.Status != 201 ||
				observation.RequestBytes != observation.SizeBytes ||
				observation.ResponseBytes != 0 {
				return errors.New("benchmark raw cold observation drifted")
			}
		case "WARM_70", "SUSTAINED", "SOAK":
			expectedStatus := 404
			expectedBytes := int64(0)
			if observation.ExpectedHit {
				expectedStatus = 200
				expectedBytes = observation.SizeBytes
				hits[key]++
			}
			if observation.Operation != "GET" ||
				observation.Status != expectedStatus ||
				observation.RequestBytes != 0 ||
				observation.ResponseBytes != expectedBytes {
				return errors.New("benchmark raw read observation drifted")
			}
		default:
			return errors.New("benchmark raw observation has unknown phase")
		}
		counts[key]++
	}
	if sequence != 1200 || len(counts) != 12 {
		return errors.New("benchmark raw observation matrix is incomplete")
	}
	for key, count := range counts {
		if count != 100 {
			return errors.New("benchmark raw stratum count drifted")
		}
		expectedHits := 70
		if key.phase == "COLD" {
			expectedHits = 0
		}
		if hits[key] != expectedHits {
			return errors.New("benchmark raw stratum hit count drifted")
		}
	}
	return nil
}

func infoSyscallStat(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func validSmokeObjectSize(size int64) bool {
	for _, mix := range smokeObjectMix {
		if mix.SizeBytes == size {
			return true
		}
	}
	return false
}

func currentHardware() hardwareIdentity {
	return hardwareIdentity{
		OperatingSystem: runtime.GOOS,
		Architecture:    runtime.GOARCH,
		CPUCount:        runtime.NumCPU(),
	}
}

func currentCgroup() cgroupIdentity {
	return cgroupIdentity{
		CPUQuota:     readIdentityFile("/sys/fs/cgroup/cpu.max"),
		MemoryLimit:  readIdentityFile("/sys/fs/cgroup/memory.max"),
		ObservedMode: "CGROUP_V2_OR_UNAVAILABLE",
	}
}

func readIdentityFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return "UNAVAILABLE"
	}
	content = bytes.TrimSpace(content)
	if len(content) == 0 || len(content) > 128 {
		return "UNAVAILABLE"
	}
	return string(content)
}
