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
	"sync"
	"time"
)

const (
	systemFaultResultSchemaVersion = "buildopt.benchmarks/system-fault-result/v1"
	systemFaultRawSchemaVersion    = "buildopt.benchmarks/system-fault-observation/v1"
	systemFaultResultFilename      = "system-fault-result.json"
	systemFaultRawFilename         = "system-fault-observations.jsonl"
)

type SystemFaultExecutables struct {
	BuildOpt string
	Server   string
}

type systemFaultResult struct {
	SchemaVersion   string                  `json:"schemaVersion"`
	Qualification   string                  `json:"qualification"`
	BenchmarkDigest string                  `json:"benchmarkDigest"`
	Seed            int64                   `json:"seed"`
	StartedAt       string                  `json:"startedAt"`
	CompletedAt     string                  `json:"completedAt"`
	FaultOutcomes   []faultOutcome          `json:"faultOutcomes"`
	Triggers        []systemFaultTrigger    `json:"triggers"`
	Recovery        []recoveryObservation   `json:"recovery"`
	RawObservations rawObservationsIdentity `json:"rawObservations"`
}

type systemFaultTrigger struct {
	ID       string `json:"id"`
	Sequence int64  `json:"sequence"`
}

type systemFaultObservation struct {
	SchemaVersion string              `json:"schemaVersion"`
	Sequence      int64               `json:"sequence"`
	FaultID       string              `json:"faultId"`
	Event         string              `json:"event"`
	DurationNs    int64               `json:"durationNs"`
	Metrics       []sharedFaultMetric `json:"metrics"`
}

type systemFaultSink struct {
	mutex    sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	hash     hashWriter
	sequence int64
	count    int64
	size     int64
}

func newSystemFaultSink(outputDirectory string) (*systemFaultSink, error) {
	path := filepath.Join(outputDirectory, systemFaultRawFilename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, errors.New("create benchmark system-fault observations")
	}
	digest := sha256.New()
	return &systemFaultSink{
		file:   file,
		writer: bufio.NewWriter(io.MultiWriter(file, digest)),
		hash: hashWriter{
			sum: func() []byte {
				return digest.Sum(nil)
			},
		},
	}, nil
}

func (sink *systemFaultSink) append(
	observation systemFaultObservation,
) (int64, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	sink.sequence++
	observation.SchemaVersion = systemFaultRawSchemaVersion
	observation.Sequence = sink.sequence
	content, err := json.Marshal(observation)
	if err != nil {
		return 0, err
	}
	content = append(content, '\n')
	written, err := sink.writer.Write(content)
	sink.size += int64(written)
	if err != nil {
		return 0, err
	}
	sink.count++
	return sink.sequence, nil
}

func (sink *systemFaultSink) close() (rawObservationsIdentity, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file == nil {
		return rawObservationsIdentity{}, errors.New(
			"benchmark system-fault observations are already closed",
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
		Path:      systemFaultRawFilename,
		SHA256:    "sha256:" + hex.EncodeToString(sink.hash.sum()),
		Count:     sink.count,
		SizeBytes: sink.size,
	}, nil
}

func (sink *systemFaultSink) abort() {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file != nil {
		_ = sink.file.Close()
		sink.file = nil
	}
}

func writeSystemFaultResult(
	outputDirectory string,
	result systemFaultResult,
) (string, error) {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	content = append(content, '\n')
	temporary, err := os.CreateTemp(
		outputDirectory,
		".system-fault-result-*.tmp",
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
	target := filepath.Join(outputDirectory, systemFaultResultFilename)
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

// ValidateSystemFaultResult verifies the final six fault rows and their raw
// trigger stream without treating them as load or soak evidence.
func ValidateSystemFaultResult(manifestPath string, resultPath string) error {
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
		return errors.New(
			"benchmark system-fault result is not a bounded private file",
		)
	}
	raw, err := os.ReadFile(resultPath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var result systemFaultResult
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode benchmark system-fault result: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New(
			"benchmark system-fault result contains trailing JSON",
		)
	}
	if result.SchemaVersion != systemFaultResultSchemaVersion ||
		result.Qualification != "FAULT_SLICE" ||
		result.BenchmarkDigest != manifestDigest ||
		result.Seed != loaded.Seed ||
		result.RawObservations.Path != systemFaultRawFilename ||
		result.RawObservations.Count != 18 ||
		result.RawObservations.SizeBytes < 1 ||
		result.RawObservations.SizeBytes > maximumResultBytes {
		return errors.New("benchmark system-fault result is incomplete")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, result.StartedAt)
	if err != nil {
		return errors.New(
			"benchmark system-fault result has invalid startedAt",
		)
	}
	completedAt, err := time.Parse(time.RFC3339Nano, result.CompletedAt)
	if err != nil || completedAt.Before(startedAt) {
		return errors.New(
			"benchmark system-fault result has invalid completedAt",
		)
	}
	expectedIDs := []string{
		"GATEWAY_RESTART",
		"SERVER_RESTART",
		"NETWORK_LATENCY",
		"NETWORK_LOSS",
		"REVOKED_POLICY",
		"REVOKED_GRANT",
	}
	expectedTriggers := []int64{2, 4, 7, 10, 13, 16}
	expectedRecovery := []string{
		"STABLE_IDENTITY_COMPLETE_HIT",
		"READY_AFTER_RECONCILIATION",
		"BOUNDED_DEADLINE_THEN_COMPLETE_HIT",
		"FAIL_OPEN_THEN_COMPLETE_HIT",
		"POLICY_ABORTED_AND_ROTATED",
		"GRANT_ABORTED_AND_ROTATED",
	}
	if len(result.FaultOutcomes) != len(expectedIDs) ||
		len(result.Triggers) != len(expectedIDs) ||
		len(result.Recovery) != len(expectedIDs) {
		return errors.New("benchmark system-fault summary is incomplete")
	}
	for index, id := range expectedIDs {
		expectedSafety, err := expectedFaultSafety(loaded, id)
		if err != nil {
			return err
		}
		if result.FaultOutcomes[index] != (faultOutcome{
			ID:             id,
			ExpectedSafety: expectedSafety,
			Status:         "PASSED",
		}) ||
			result.Triggers[index] != (systemFaultTrigger{
				ID:       id,
				Sequence: expectedTriggers[index],
			}) ||
			result.Recovery[index].ID != id ||
			result.Recovery[index].DurationNs <= 0 ||
			result.Recovery[index].Status != expectedRecovery[index] {
			return errors.New("benchmark system-fault summary drifted")
		}
	}
	observations, err := readSystemFaultObservations(
		filepath.Dir(resultPath),
		result.RawObservations,
	)
	if err != nil {
		return err
	}
	return validateRawSystemFaultObservations(observations)
}

func readSystemFaultObservations(
	outputDirectory string,
	identity rawObservationsIdentity,
) ([]byte, error) {
	path := filepath.Join(outputDirectory, identity.Path)
	info, err := os.Lstat(path)
	stat, ownerAvailable := infoSyscallStat(info)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		!ownerAvailable ||
		stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.New(
			"benchmark system-fault observations are not a private file",
		)
	}
	content, err := os.ReadFile(path)
	if err != nil ||
		int64(len(content)) != identity.SizeBytes ||
		len(content) > maximumResultBytes {
		return nil, errors.New(
			"benchmark system-fault observations are unavailable",
		)
	}
	digest := sha256.Sum256(content)
	if identity.SHA256 != "sha256:"+hex.EncodeToString(digest[:]) {
		return nil, errors.New(
			"benchmark system-fault observation digest mismatched",
		)
	}
	if int64(bytes.Count(content, []byte{'\n'})) != identity.Count {
		return nil, errors.New(
			"benchmark system-fault observation count mismatched",
		)
	}
	return content, nil
}

func validateRawSystemFaultObservations(content []byte) error {
	expected := expectedSystemFaultObservations()
	reader := bufio.NewReader(bytes.NewReader(content))
	index := 0
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		if err != nil || index >= len(expected) {
			return errors.New(
				"read benchmark system-fault observation",
			)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var observation systemFaultObservation
		if err := decoder.Decode(&observation); err != nil {
			return errors.New(
				"decode benchmark system-fault observation",
			)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return errors.New(
				"benchmark system-fault observation contains trailing JSON",
			)
		}
		want := expected[index]
		if observation.SchemaVersion != systemFaultRawSchemaVersion ||
			observation.Sequence != int64(index+1) ||
			observation.FaultID != want.FaultID ||
			observation.Event != want.Event ||
			observation.DurationNs < 0 ||
			!slices.Equal(observation.Metrics, want.Metrics) {
			return errors.New(
				"benchmark system-fault observation drifted",
			)
		}
		index++
	}
	if index != len(expected) {
		return errors.New(
			"benchmark system-fault observation matrix is incomplete",
		)
	}
	return nil
}

func expectedSystemFaultObservations() []systemFaultObservation {
	return []systemFaultObservation{
		{
			FaultID: "GATEWAY_RESTART",
			Event:   "GATEWAY_COMPLETE_HIT_BEFORE_RESTART",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 200},
				{Name: "responseBytes", Value: 100},
				{Name: "digestMatch", Value: 1},
			},
		},
		{
			FaultID: "GATEWAY_RESTART",
			Event:   "GATEWAY_PROCESS_RESTARTED",
			Metrics: []sharedFaultMetric{
				{Name: "endpointReused", Value: 1},
				{Name: "credentialReused", Value: 1},
				{Name: "generationReused", Value: 1},
			},
		},
		{
			FaultID: "GATEWAY_RESTART",
			Event:   "GATEWAY_COMPLETE_HIT_AFTER_RESTART",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 200},
				{Name: "responseBytes", Value: 100},
				{Name: "digestMatch", Value: 1},
			},
		},
		{
			FaultID: "SERVER_RESTART",
			Event:   "SERVER_PROCESS_STOPPED",
			Metrics: []sharedFaultMetric{
				{Name: "previousReadinessStatus", Value: 200},
				{Name: "exitCode", Value: 0},
			},
		},
		{
			FaultID: "SERVER_RESTART",
			Event:   "RESTART_LIVE_NOT_READY",
			Metrics: []sharedFaultMetric{
				{Name: "livenessStatus", Value: 200},
				{Name: "readinessStatus", Value: 503},
				{Name: "productStatus", Value: 503},
			},
		},
		{
			FaultID: "SERVER_RESTART",
			Event:   "RECONCILED_AND_READY",
			Metrics: []sharedFaultMetric{
				{Name: "readinessStatus", Value: 200},
				{Name: "orphanFilesRemaining", Value: 0},
			},
		},
		{
			FaultID: "NETWORK_LATENCY",
			Event:   "UPSTREAM_LATENCY_INJECTED",
			Metrics: []sharedFaultMetric{
				{Name: "delayMilliseconds", Value: 250},
				{Name: "deadlineMilliseconds", Value: 100},
			},
		},
		{
			FaultID: "NETWORK_LATENCY",
			Event:   "DEADLINE_EXCEEDED_AND_RECORDED",
			Metrics: []sharedFaultMetric{
				{Name: "deadlineExceeded", Value: 1},
				{Name: "responseBytes", Value: 0},
				{Name: "errorRecorded", Value: 1},
			},
		},
		{
			FaultID: "NETWORK_LATENCY",
			Event:   "COMPLETE_HIT_AFTER_LATENCY_RECOVERY",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 200},
				{Name: "responseBytes", Value: 100},
				{Name: "digestMatch", Value: 1},
			},
		},
		{
			FaultID: "NETWORK_LOSS",
			Event:   "UPSTREAM_CONNECTION_DROPPED",
			Metrics: []sharedFaultMetric{
				{Name: "connectionDropObserved", Value: 1},
			},
		},
		{
			FaultID: "NETWORK_LOSS",
			Event:   "BYTE_FREE_FAIL_OPEN_MISS",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 404},
				{Name: "responseBytes", Value: 0},
				{Name: "failOpen", Value: 1},
			},
		},
		{
			FaultID: "NETWORK_LOSS",
			Event:   "COMPLETE_HIT_AFTER_NETWORK_RECOVERY",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 200},
				{Name: "responseBytes", Value: 100},
				{Name: "digestMatch", Value: 1},
			},
		},
		{
			FaultID: "REVOKED_POLICY",
			Event:   "POLICY_REVOCATION_INSTALLED",
			Metrics: []sharedFaultMetric{
				{Name: "revocationEpochBefore", Value: 7},
				{Name: "revocationEpochAfter", Value: 8},
				{Name: "namespaceGenerationBefore", Value: 12},
				{Name: "namespaceGenerationAfter", Value: 13},
			},
		},
		{
			FaultID: "REVOKED_POLICY",
			Event:   "STALE_ROUTE_REJECTED",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 401},
				{Name: "responseBytes", Value: 0},
			},
		},
		{
			FaultID: "REVOKED_POLICY",
			Event:   "PENDING_ABORTED_AND_GENERATION_ROTATED",
			Metrics: []sharedFaultMetric{
				{Name: "attemptStateAborted", Value: 1},
				{Name: "pendingBytes", Value: 0},
				{Name: "l1GenerationDelta", Value: 1},
			},
		},
		{
			FaultID: "REVOKED_GRANT",
			Event:   "GRANT_REVOCATION_INSTALLED",
			Metrics: []sharedFaultMetric{
				{Name: "grantDigestChanged", Value: 1},
				{Name: "revocationEpochBefore", Value: 7},
				{Name: "revocationEpochAfter", Value: 8},
			},
		},
		{
			FaultID: "REVOKED_GRANT",
			Event:   "STALE_GRANT_ROUTE_REJECTED",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 401},
				{Name: "responseBytes", Value: 0},
			},
		},
		{
			FaultID: "REVOKED_GRANT",
			Event:   "PENDING_ABORTED_AND_GENERATION_ROTATED",
			Metrics: []sharedFaultMetric{
				{Name: "attemptStateAborted", Value: 1},
				{Name: "pendingBytes", Value: 0},
				{Name: "l1GenerationDelta", Value: 1},
			},
		},
	}
}
