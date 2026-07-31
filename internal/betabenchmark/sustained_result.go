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
	"slices"
	"syscall"
	"time"
)

const (
	sustainedResultSchemaVersion = "buildopt.benchmarks/sustained-result/v1"
	sustainedResultFilename      = "sustained-result.json"
	maximumSustainedRawBytes     = 32 << 20
)

type sustainedRunOptions struct {
	qualification   string
	objectMix       []objectMix
	cycleCount      int
	totalDuration   time.Duration
	requireRunner   bool
	expectedRawRows int64
}

type sustainedResult struct {
	SchemaVersion            string                   `json:"schemaVersion"`
	Qualification            string                   `json:"qualification"`
	BenchmarkDigest          string                   `json:"benchmarkDigest"`
	Seed                     int64                    `json:"seed"`
	StartedAt                string                   `json:"startedAt"`
	CompletedAt              string                   `json:"completedAt"`
	Hardware                 hardwareIdentity         `json:"hardware"`
	Cgroup                   cgroupIdentity           `json:"cgroup"`
	Components               components               `json:"components"`
	RunnerVerified           bool                     `json:"runnerVerified"`
	Transport                string                   `json:"transport"`
	CycleOperations          int                      `json:"cycleOperations"`
	ActualObjectDistribution []distribution           `json:"actualObjectDistribution"`
	P50                      []percentile             `json:"p50"`
	P95                      []percentile             `json:"p95"`
	P99                      []percentile             `json:"p99"`
	Throughput               []throughput             `json:"throughput"`
	Errors                   []errorCount             `json:"errors"`
	Bytes                    []byteCount              `json:"bytes"`
	LatencyTargets           []sustainedLatencyTarget `json:"latencyTargets"`
	Deviations               []string                 `json:"deviations"`
	RawObservations          rawObservationsIdentity  `json:"rawObservations"`
}

type sustainedLatencyTarget struct {
	Clients      int    `json:"clients"`
	Metric       string `json:"metric"`
	SizeBytes    int64  `json:"sizeBytes"`
	Samples      int64  `json:"samples"`
	P50Ns        int64  `json:"p50Ns"`
	P95Ns        int64  `json:"p95Ns"`
	P99Ns        int64  `json:"p99Ns"`
	MaximumP95Ns int64  `json:"maximumP95Ns"`
	Status       string `json:"status"`
}

func productionSustainedOptions(loaded manifest) sustainedRunOptions {
	duration := time.Duration(0)
	for _, current := range loaded.Phases {
		if current.ID == "SUSTAINED" {
			duration = time.Duration(current.DurationSeconds) * time.Second
		}
	}
	return sustainedRunOptions{
		qualification:   "SUSTAINED_SLICE",
		objectMix:       slices.Clone(loaded.ObjectMix),
		cycleCount:      loaded.ObjectCycleCount / 100,
		totalDuration:   duration,
		requireRunner:   true,
		expectedRawRows: int64(loaded.ObjectCycleCount * len(loaded.Clients)),
	}
}

func trialSustainedOptions() sustainedRunOptions {
	return sustainedRunOptions{
		qualification:   "TRIAL",
		objectMix:       slices.Clone(smokeObjectMix),
		cycleCount:      1,
		totalDuration:   900 * time.Millisecond,
		requireRunner:   false,
		expectedRawRows: 300,
	}
}

func sustainedDeviations(options sustainedRunOptions) []string {
	if options.qualification == "TRIAL" {
		return []string{
			"TRIAL_DURATION_AND_SCALED_SIZES",
			"RUNNER_QUALIFICATION_NOT_CLAIMED",
			"COLD_WARM_SETUP_ONLY",
			"GRADLE_FIXTURES_NOT_RUN",
			"SOAK_NOT_RUN",
		}
	}
	return []string{
		"COLD_WARM_SETUP_ONLY",
		"GRADLE_FIXTURES_NOT_RUN",
		"SOAK_NOT_RUN",
	}
}

func writeSustainedResult(
	outputDirectory string,
	result sustainedResult,
) (string, error) {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	content = append(content, '\n')
	temporary, err := os.CreateTemp(
		outputDirectory,
		".sustained-result-*.tmp",
	)
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
	target := filepath.Join(outputDirectory, sustainedResultFilename)
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

// ValidateSustainedResult verifies an exact one-hour result and its bound raw
// stream. Trial results are intentionally rejected by this entrypoint.
func ValidateSustainedResult(manifestPath string, resultPath string) error {
	loaded, _, _, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	return validateSustainedResult(
		manifestPath,
		resultPath,
		productionSustainedOptions(loaded),
	)
}

// ValidateSustainedTrial verifies only the short non-qualifying CI execution.
func ValidateSustainedTrial(manifestPath string, resultPath string) error {
	return validateSustainedResult(
		manifestPath,
		resultPath,
		trialSustainedOptions(),
	)
}

func validateSustainedResult(
	manifestPath string,
	resultPath string,
	options sustainedRunOptions,
) error {
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	rawResult, err := readPrivateBoundedFile(resultPath, maximumResultBytes)
	if err != nil {
		return errors.New("benchmark sustained result is not a bounded private file")
	}
	decoder := json.NewDecoder(bytes.NewReader(rawResult))
	decoder.DisallowUnknownFields()
	var result sustainedResult
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode benchmark sustained result: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("benchmark sustained result contains trailing JSON")
	}
	if result.SchemaVersion != sustainedResultSchemaVersion ||
		result.Qualification != options.qualification ||
		result.BenchmarkDigest != manifestDigest ||
		result.Seed != loaded.Seed ||
		result.Components != loaded.Components ||
		result.Hardware.OperatingSystem == "" ||
		result.Hardware.Architecture == "" ||
		result.Hardware.CPUCount < 1 ||
		result.Transport != "REAL_MANAGED_GATEWAY_LOOPBACK_HTTP" ||
		result.CycleOperations != options.cycleCount*100 ||
		result.RunnerVerified != options.requireRunner ||
		result.RawObservations.Path != observationsFilename ||
		result.RawObservations.Count != options.expectedRawRows ||
		result.RawObservations.SizeBytes < 1 ||
		result.RawObservations.SizeBytes > maximumSustainedRawBytes ||
		!slices.Equal(result.Deviations, sustainedDeviations(options)) {
		return errors.New("benchmark sustained result is incomplete")
	}
	if options.requireRunner && !goldenRunnerCgroup(result.Cgroup) {
		return errors.New("benchmark sustained result lacks golden runner cgroup")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, result.StartedAt)
	if err != nil {
		return errors.New("benchmark sustained result has invalid startedAt")
	}
	completedAt, err := time.Parse(time.RFC3339Nano, result.CompletedAt)
	if err != nil || completedAt.Sub(startedAt) < options.totalDuration {
		return errors.New("benchmark sustained result has invalid completedAt")
	}
	observations, err := readSustainedObservations(
		filepath.Dir(resultPath),
		result.RawObservations,
	)
	if err != nil {
		return err
	}
	summaries, err := validateSustainedRaw(
		observations,
		loaded,
		options,
	)
	if err != nil {
		return err
	}
	distributions, p50, p95, p99, _, errorsByClass, bytesByStratum :=
		summarizeStrata(summaries)
	targets := summarizeSustainedLatencyTargets(observations, loaded.Clients)
	if !slices.Equal(result.ActualObjectDistribution, distributions) ||
		!slices.Equal(result.P50, p50) ||
		!slices.Equal(result.P95, p95) ||
		!slices.Equal(result.P99, p99) ||
		!slices.Equal(result.Errors, errorsByClass) ||
		!slices.Equal(result.Bytes, bytesByStratum) ||
		!slices.Equal(result.LatencyTargets, targets) {
		return errors.New("benchmark sustained summary drifted")
	}
	if len(result.Errors) != 0 {
		return errors.New("benchmark sustained load observed errors")
	}
	if err := validateSustainedThroughput(
		result.Throughput,
		bytesByStratum,
		loaded.Clients,
		options,
	); err != nil {
		return err
	}
	if options.requireRunner {
		for _, target := range result.LatencyTargets {
			if target.Status != "PASSED" {
				return fmt.Errorf(
					"benchmark sustained latency target failed: "+
						"clients=%d metric=%s sizeBytes=%d "+
						"p95Ns=%d maximumP95Ns=%d",
					target.Clients,
					target.Metric,
					target.SizeBytes,
					target.P95Ns,
					target.MaximumP95Ns,
				)
			}
		}
	}
	return nil
}

func readPrivateBoundedFile(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		info.Size() < 1 ||
		info.Size() > maximum {
		return nil, errors.New("file is not bounded and private")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("file owner is invalid")
	}
	return os.ReadFile(path)
}

func readSustainedObservations(
	outputDirectory string,
	identity rawObservationsIdentity,
) ([]rawObservation, error) {
	path := filepath.Join(outputDirectory, identity.Path)
	content, err := readPrivateBoundedFile(path, maximumSustainedRawBytes)
	if err != nil || int64(len(content)) != identity.SizeBytes {
		return nil, errors.New("benchmark sustained observations are unavailable")
	}
	digest := sha256.Sum256(content)
	if identity.SHA256 != "sha256:"+hex.EncodeToString(digest[:]) ||
		int64(bytes.Count(content, []byte{'\n'})) != identity.Count {
		return nil, errors.New("benchmark sustained observation identity mismatched")
	}
	observations := make([]rawObservation, 0, identity.Count)
	reader := bufio.NewReader(bytes.NewReader(content))
	for {
		line, readErr := reader.ReadBytes('\n')
		if errors.Is(readErr, io.EOF) && len(line) == 0 {
			break
		}
		if readErr != nil {
			return nil, errors.New("read benchmark sustained observation")
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var observation rawObservation
		if err := decoder.Decode(&observation); err != nil {
			return nil, errors.New("decode benchmark sustained observation")
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return nil, errors.New(
				"benchmark sustained observation contains trailing JSON",
			)
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func validateSustainedRaw(
	observations []rawObservation,
	loaded manifest,
	options sustainedRunOptions,
) ([]*stratumSummary, error) {
	if int64(len(observations)) != options.expectedRawRows {
		return nil, errors.New("benchmark sustained observation count drifted")
	}
	summaries := make(map[int]*stratumSummary, len(loaded.Clients))
	for _, clients := range loaded.Clients {
		summary := newStratumSummary("SUSTAINED", clients)
		summary.startedAt = time.Unix(0, 0)
		summaries[clients] = summary
	}
	objects := makeSustainedObjects(options.objectMix, loaded.Seed)
	expectedOperations := repeatSustainedOperations(
		makeSmokeOperations(
			objects,
			phase{
				ID:               "SUSTAINED",
				TargetHitPercent: 70,
			},
			loaded.Seed+500,
		),
		options.cycleCount,
	)
	for index, observation := range observations {
		clientIndex := index / len(expectedOperations)
		expectedOperation := expectedOperations[index%len(expectedOperations)]
		if observation.SchemaVersion != observationSchemaVersion ||
			observation.Sequence != int64(index+1) ||
			observation.Phase != "SUSTAINED" ||
			clientIndex >= len(loaded.Clients) ||
			observation.Clients != loaded.Clients[clientIndex] ||
			observation.Operation != "GET" ||
			observation.ObjectIndex != expectedOperation.object.index ||
			observation.SizeBytes != expectedOperation.object.size ||
			observation.ExpectedHit != expectedOperation.expectedHit ||
			observation.ReadyNs <= 0 ||
			observation.DurationNs < observation.ReadyNs ||
			observation.RequestBytes != 0 ||
			observation.ErrorClass != "" ||
			!validObjectSize(options.objectMix, observation.SizeBytes) {
			return nil, errors.New("benchmark sustained observation drifted")
		}
		expectedStatus := 404
		expectedBytes := int64(0)
		if observation.ExpectedHit {
			expectedStatus = 200
			expectedBytes = observation.SizeBytes
		}
		if observation.Status != expectedStatus ||
			observation.ResponseBytes != expectedBytes {
			return nil, errors.New("benchmark sustained response drifted")
		}
		summaries[observation.Clients].observe(observation)
	}
	ordered := make([]*stratumSummary, 0, len(loaded.Clients))
	segment := options.totalDuration / time.Duration(len(loaded.Clients))
	for _, clients := range loaded.Clients {
		summary := summaries[clients]
		if len(summary.latencies) != options.cycleCount*100 {
			return nil, errors.New("benchmark sustained stratum is incomplete")
		}
		hits := int64(0)
		for _, item := range summary.distributions {
			hits += item.ExpectedHits
		}
		if hits != int64(options.cycleCount*70) {
			return nil, errors.New("benchmark sustained hit target drifted")
		}
		summary.completedAt = summary.startedAt.Add(segment)
		ordered = append(ordered, summary)
	}
	return ordered, nil
}

func summarizeSustainedLatencyTargets(
	observations []rawObservation,
	clientsValues []int,
) []sustainedLatencyTarget {
	type key struct {
		clients int
		metric  string
		size    int64
	}
	values := make(map[key][]int64)
	for _, observation := range observations {
		if !observation.ExpectedHit {
			current := key{
				clients: observation.Clients,
				metric:  "GATEWAY_MISS",
				size:    observation.SizeBytes,
			}
			values[current] = append(
				values[current],
				observation.ReadyNs,
			)
			continue
		}
		ready := key{
			clients: observation.Clients,
			metric:  "VERIFIED_HIT_READY",
			size:    observation.SizeBytes,
		}
		values[ready] = append(values[ready], observation.ReadyNs)
		materialization := key{
			clients: observation.Clients,
			metric:  "DOWNSTREAM_MATERIALIZATION",
			size:    observation.SizeBytes,
		}
		values[materialization] = append(
			values[materialization],
			observation.DurationNs-observation.ReadyNs,
		)
	}
	targets := make([]sustainedLatencyTarget, 0, len(values))
	for _, clients := range clientsValues {
		for _, metric := range []string{
			"GATEWAY_MISS",
			"VERIFIED_HIT_READY",
			"DOWNSTREAM_MATERIALIZATION",
		} {
			var sizes []int64
			for current := range values {
				if current.clients == clients &&
					current.metric == metric {
					sizes = append(sizes, current.size)
				}
			}
			slices.Sort(sizes)
			for _, size := range sizes {
				durations := values[key{
					clients: clients,
					metric:  metric,
					size:    size,
				}]
				slices.Sort(durations)
				maximum := sustainedP95Limit(metric, size)
				p95 := nearestDuration(durations, 95)
				status := "PASSED"
				if p95 > maximum {
					status = "FAILED"
				}
				targets = append(targets, sustainedLatencyTarget{
					Clients:      clients,
					Metric:       metric,
					SizeBytes:    size,
					Samples:      int64(len(durations)),
					P50Ns:        nearestDuration(durations, 50),
					P95Ns:        p95,
					P99Ns:        nearestDuration(durations, 99),
					MaximumP95Ns: maximum,
					Status:       status,
				})
			}
		}
	}
	return targets
}

func sustainedP95Limit(metric string, size int64) int64 {
	if metric == "GATEWAY_MISS" {
		return (50 * time.Millisecond).Nanoseconds()
	}
	if metric == "DOWNSTREAM_MATERIALIZATION" {
		return (150*time.Millisecond +
			time.Duration(
				(size*int64(time.Second))/(200<<20),
			)).Nanoseconds()
	}
	switch {
	case size <= 1<<20:
		return (150 * time.Millisecond).Nanoseconds()
	case size <= 10<<20:
		return (400 * time.Millisecond).Nanoseconds()
	default:
		return (2500 * time.Millisecond).Nanoseconds()
	}
}

func nearestDuration(values []int64, percentile int) int64 {
	if len(values) == 0 {
		return 0
	}
	index := (percentile*len(values)+99)/100 - 1
	return values[index]
}

func validObjectSize(mix []objectMix, size int64) bool {
	for _, item := range mix {
		if item.SizeBytes == size {
			return true
		}
	}
	return false
}

func goldenRunnerCgroup(cgroup cgroupIdentity) bool {
	return cgroup.CPUQuota == "400000 100000" &&
		cgroup.MemoryLimit == "17179869184"
}

func validateSustainedThroughput(
	rates []throughput,
	bytesByStratum []byteCount,
	clientsValues []int,
	options sustainedRunOptions,
) error {
	if len(rates) != len(clientsValues) ||
		len(bytesByStratum) != len(clientsValues) {
		return errors.New("benchmark sustained throughput matrix is incomplete")
	}
	minimumDuration := options.totalDuration /
		time.Duration(len(clientsValues))
	totalDuration := time.Duration(0)
	for index, clients := range clientsValues {
		rate := rates[index]
		byteSummary := bytesByStratum[index]
		duration := time.Duration(rate.DurationNs)
		durationSeconds := duration.Seconds()
		expectedOperationsPerSecond := float64(options.cycleCount*100) /
			durationSeconds
		expectedBytesPerSecond := float64(
			byteSummary.RequestBytes+byteSummary.ResponseBytes,
		) / durationSeconds
		if rate.Phase != "SUSTAINED" ||
			rate.Clients != clients ||
			duration < minimumDuration ||
			rate.OperationsPerSecond != expectedOperationsPerSecond ||
			rate.BytesPerSecond != expectedBytesPerSecond {
			return errors.New("benchmark sustained throughput drifted")
		}
		totalDuration += duration
	}
	if totalDuration < options.totalDuration {
		return errors.New("benchmark sustained duration is incomplete")
	}
	return nil
}
