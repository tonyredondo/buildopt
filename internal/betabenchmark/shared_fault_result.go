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
	sharedFaultResultSchemaVersion = "buildopt.benchmarks/shared-fault-result/v1"
	sharedFaultRawSchemaVersion    = "buildopt.benchmarks/shared-fault-observation/v1"
	sharedFaultResultFilename      = "shared-fault-result.json"
	sharedFaultRawFilename         = "shared-fault-observations.jsonl"
)

type sharedFaultResult struct {
	SchemaVersion   string                  `json:"schemaVersion"`
	Qualification   string                  `json:"qualification"`
	BenchmarkDigest string                  `json:"benchmarkDigest"`
	Seed            int64                   `json:"seed"`
	StartedAt       string                  `json:"startedAt"`
	CompletedAt     string                  `json:"completedAt"`
	FaultOutcomes   []faultOutcome          `json:"faultOutcomes"`
	Triggers        []sharedFaultTrigger    `json:"triggers"`
	Recovery        []recoveryObservation   `json:"recovery"`
	RawObservations rawObservationsIdentity `json:"rawObservations"`
}

type sharedFaultTrigger struct {
	ID       string `json:"id"`
	Sequence int64  `json:"sequence"`
}

type sharedFaultObservation struct {
	SchemaVersion string              `json:"schemaVersion"`
	Sequence      int64               `json:"sequence"`
	FaultID       string              `json:"faultId"`
	Event         string              `json:"event"`
	DurationNs    int64               `json:"durationNs"`
	Metrics       []sharedFaultMetric `json:"metrics"`
}

type sharedFaultMetric struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type sharedFaultSink struct {
	mutex    sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	hash     hashWriter
	sequence int64
	count    int64
	size     int64
}

func newSharedFaultSink(outputDirectory string) (*sharedFaultSink, error) {
	path := filepath.Join(outputDirectory, sharedFaultRawFilename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, errors.New("create benchmark Shared-fault observations")
	}
	digest := sha256.New()
	return &sharedFaultSink{
		file:   file,
		writer: bufio.NewWriter(io.MultiWriter(file, digest)),
		hash: hashWriter{
			sum: func() []byte {
				return digest.Sum(nil)
			},
		},
	}, nil
}

func (sink *sharedFaultSink) append(
	observation sharedFaultObservation,
) (int64, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	sink.sequence++
	observation.SchemaVersion = sharedFaultRawSchemaVersion
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

func (sink *sharedFaultSink) close() (rawObservationsIdentity, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file == nil {
		return rawObservationsIdentity{}, errors.New(
			"benchmark Shared-fault observations are already closed",
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
		Path:      sharedFaultRawFilename,
		SHA256:    "sha256:" + hex.EncodeToString(sink.hash.sum()),
		Count:     sink.count,
		SizeBytes: sink.size,
	}, nil
}

func (sink *sharedFaultSink) abort() {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file != nil {
		_ = sink.file.Close()
		sink.file = nil
	}
}

func writeSharedFaultResult(
	outputDirectory string,
	result sharedFaultResult,
) (string, error) {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	content = append(content, '\n')
	temporary, err := os.CreateTemp(
		outputDirectory,
		".shared-fault-result-*.tmp",
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
	target := filepath.Join(outputDirectory, sharedFaultResultFilename)
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

// ValidateSharedFaultResult verifies the exact seven-row Shared slice and its
// raw trigger stream without treating it as the complete fault matrix.
func ValidateSharedFaultResult(manifestPath string, resultPath string) error {
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
			"benchmark Shared-fault result is not a bounded private file",
		)
	}
	raw, err := os.ReadFile(resultPath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var result sharedFaultResult
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode benchmark Shared-fault result: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New(
			"benchmark Shared-fault result contains trailing JSON",
		)
	}
	if result.SchemaVersion != sharedFaultResultSchemaVersion ||
		result.Qualification != "FAULT_SLICE" ||
		result.BenchmarkDigest != manifestDigest ||
		result.Seed != loaded.Seed ||
		result.RawObservations.Path != sharedFaultRawFilename ||
		result.RawObservations.Count != 17 ||
		result.RawObservations.SizeBytes < 1 ||
		result.RawObservations.SizeBytes > maximumResultBytes {
		return errors.New("benchmark Shared-fault result is incomplete")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, result.StartedAt)
	if err != nil {
		return errors.New(
			"benchmark Shared-fault result has invalid startedAt",
		)
	}
	completedAt, err := time.Parse(time.RFC3339Nano, result.CompletedAt)
	if err != nil || completedAt.Before(startedAt) {
		return errors.New(
			"benchmark Shared-fault result has invalid completedAt",
		)
	}
	expectedIDs := []string{
		"MID_PUT_CANCEL",
		"MID_GET_CANCEL",
		"TRUNCATED_BLOB",
		"CORRUPT_BLOB",
		"SQLITE_BUSY",
		"EXPIRED_LEASE",
		"DEATH_BETWEEN_PENDING_AND_COMMIT",
	}
	expectedTriggers := []int64{1, 4, 6, 8, 10, 13, 15}
	expectedRecovery := []string{
		"COMPLETE_RETRY_ACCEPTED",
		"COMPLETE_RETRY_VERIFIED",
		"MISS_AND_QUARANTINED",
		"MISS_AND_QUARANTINED",
		"COMMITTED_AFTER_LOCK_RELEASE",
		"ABORTED_AND_OWNER_RELEASED",
		"AUTHORIZED_COMMIT_RECOVERED",
	}
	if len(result.FaultOutcomes) != len(expectedIDs) ||
		len(result.Triggers) != len(expectedIDs) ||
		len(result.Recovery) != len(expectedIDs) {
		return errors.New("benchmark Shared-fault summary is incomplete")
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
			result.Triggers[index] != (sharedFaultTrigger{
				ID:       id,
				Sequence: expectedTriggers[index],
			}) ||
			result.Recovery[index].ID != id ||
			result.Recovery[index].DurationNs <= 0 ||
			result.Recovery[index].Status != expectedRecovery[index] {
			return errors.New("benchmark Shared-fault summary drifted")
		}
	}
	observations, err := readSharedFaultObservations(
		filepath.Dir(resultPath),
		result.RawObservations,
	)
	if err != nil {
		return err
	}
	return validateRawSharedFaultObservations(observations)
}

func readSharedFaultObservations(
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
			"benchmark Shared-fault observations are not a private file",
		)
	}
	content, err := os.ReadFile(path)
	if err != nil ||
		int64(len(content)) != identity.SizeBytes ||
		len(content) > maximumResultBytes {
		return nil, errors.New(
			"benchmark Shared-fault observations are unavailable",
		)
	}
	digest := sha256.Sum256(content)
	if identity.SHA256 != "sha256:"+hex.EncodeToString(digest[:]) {
		return nil, errors.New(
			"benchmark Shared-fault observation digest mismatched",
		)
	}
	if int64(bytes.Count(content, []byte{'\n'})) != identity.Count {
		return nil, errors.New(
			"benchmark Shared-fault observation count mismatched",
		)
	}
	return content, nil
}

func validateRawSharedFaultObservations(content []byte) error {
	expected := expectedSharedFaultObservations()
	reader := bufio.NewReader(bytes.NewReader(content))
	index := 0
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		if err != nil || index >= len(expected) {
			return errors.New(
				"read benchmark Shared-fault observation",
			)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var observation sharedFaultObservation
		if err := decoder.Decode(&observation); err != nil {
			return errors.New(
				"decode benchmark Shared-fault observation",
			)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return errors.New(
				"benchmark Shared-fault observation contains trailing JSON",
			)
		}
		want := expected[index]
		if observation.SchemaVersion != sharedFaultRawSchemaVersion ||
			observation.Sequence != int64(index+1) ||
			observation.FaultID != want.FaultID ||
			observation.Event != want.Event ||
			observation.DurationNs < 0 ||
			!slices.Equal(observation.Metrics, want.Metrics) {
			return errors.New(
				"benchmark Shared-fault observation drifted",
			)
		}
		index++
	}
	if index != len(expected) {
		return errors.New(
			"benchmark Shared-fault observation matrix is incomplete",
		)
	}
	return nil
}

func expectedSharedFaultObservations() []sharedFaultObservation {
	return []sharedFaultObservation{
		{
			FaultID: "MID_PUT_CANCEL",
			Event:   "PARTIAL_PUT_STARTED",
			Metrics: []sharedFaultMetric{
				{Name: "bodyBytesSent", Value: 100},
				{Name: "spoolFiles", Value: 1},
			},
		},
		{
			FaultID: "MID_PUT_CANCEL",
			Event:   "CANCELLED_WITHOUT_PARTIAL_AUTHORITY",
			Metrics: []sharedFaultMetric{
				{Name: "pendingBytes", Value: 0},
				{Name: "stableBytes", Value: 0},
				{Name: "reservedBytes", Value: 0},
			},
		},
		{
			FaultID: "MID_PUT_CANCEL",
			Event:   "COMPLETE_RETRY_ACCEPTED",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 201},
				{Name: "requestBytes", Value: 1_000},
				{Name: "deletedUnreferencedBlobs", Value: 1},
			},
		},
		{
			FaultID: "MID_GET_CANCEL",
			Event:   "GET_CANCELLED_AFTER_PARTIAL_READ",
			Metrics: []sharedFaultMetric{
				{Name: "bodyBytesRead", Value: 4_096},
				{Name: "objectBytes", Value: 1_048_576},
				{Name: "acceptedHit", Value: 0},
			},
		},
		{
			FaultID: "MID_GET_CANCEL",
			Event:   "COMPLETE_RETRY_VERIFIED",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 200},
				{Name: "responseBytes", Value: 1_048_576},
				{Name: "digestMatch", Value: 1},
			},
		},
		{
			FaultID: "TRUNCATED_BLOB",
			Event:   "BLOB_TRUNCATED",
			Metrics: []sharedFaultMetric{
				{Name: "expectedBytes", Value: 1_000},
				{Name: "actualBytes", Value: 999},
			},
		},
		{
			FaultID: "TRUNCATED_BLOB",
			Event:   "MISS_AND_QUARANTINE",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 404},
				{Name: "responseBytes", Value: 0},
				{Name: "stableBytes", Value: 0},
				{Name: "quarantineFiles", Value: 1},
			},
		},
		{
			FaultID: "CORRUPT_BLOB",
			Event:   "BLOB_CORRUPTED",
			Metrics: []sharedFaultMetric{
				{Name: "expectedBytes", Value: 1_000},
				{Name: "actualBytes", Value: 1_000},
			},
		},
		{
			FaultID: "CORRUPT_BLOB",
			Event:   "MISS_AND_QUARANTINE",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 404},
				{Name: "responseBytes", Value: 0},
				{Name: "stableBytes", Value: 0},
				{Name: "quarantineFiles", Value: 1},
			},
		},
		{
			FaultID: "SQLITE_BUSY",
			Event:   "SQLITE_WRITE_LOCK_HELD",
			Metrics: []sharedFaultMetric{
				{Name: "committedObjects", Value: 0},
			},
		},
		{
			FaultID: "SQLITE_BUSY",
			Event:   "SQLITE_BUSY_WITHOUT_PARTIAL_VISIBILITY",
			Metrics: []sharedFaultMetric{
				{Name: "busyReturned", Value: 1},
				{Name: "committedObjects", Value: 0},
			},
		},
		{
			FaultID: "SQLITE_BUSY",
			Event:   "COMMIT_RECOVERED_AFTER_LOCK_RELEASE",
			Metrics: []sharedFaultMetric{
				{Name: "committedObjects", Value: 1},
				{Name: "completeHit", Value: 1},
			},
		},
		{
			FaultID: "EXPIRED_LEASE",
			Event:   "LEASE_EXPIRED_WITH_PENDING_OBJECT",
			Metrics: []sharedFaultMetric{
				{Name: "pendingBytes", Value: 100},
				{Name: "pendingObjects", Value: 1},
			},
		},
		{
			FaultID: "EXPIRED_LEASE",
			Event:   "ABORTED_AND_OWNER_RELEASED",
			Metrics: []sharedFaultMetric{
				{Name: "expiredAttempts", Value: 1},
				{Name: "deletedOrphanBlobs", Value: 1},
				{Name: "pendingBytes", Value: 0},
			},
		},
		{
			FaultID: "DEATH_BETWEEN_PENDING_AND_COMMIT",
			Event:   "PROCESS_DIED_WITH_PENDING_OBJECT",
			Metrics: []sharedFaultMetric{
				{Name: "pendingBytes", Value: 100},
				{Name: "committedObjects", Value: 0},
			},
		},
		{
			FaultID: "DEATH_BETWEEN_PENDING_AND_COMMIT",
			Event:   "RESTARTED_PENDING_REMAINS_MISS",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 404},
				{Name: "responseBytes", Value: 0},
			},
		},
		{
			FaultID: "DEATH_BETWEEN_PENDING_AND_COMMIT",
			Event:   "AUTHORIZED_COMMIT_RECOVERED",
			Metrics: []sharedFaultMetric{
				{Name: "httpStatus", Value: 200},
				{Name: "responseBytes", Value: 100},
			},
		},
	}
}
