package betabenchmark

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	diskFaultResultSchemaVersion = "buildopt.benchmarks/disk-fault-result/v1"
	diskFaultRawSchemaVersion    = "buildopt.benchmarks/disk-fault-observation/v1"
	diskFaultResultFilename      = "disk-fault-result.json"
	diskFaultRawFilename         = "disk-fault-observations.jsonl"
)

var diskFaultPolicy = sharedcache.CapacityPolicy{
	DeploymentBytes:        1_000,
	RepositoryBytes:        1_000,
	PendingQuarantineBytes: 100,
	StableTTL:              30 * 24 * time.Hour,
	QuarantineTTL:          7 * 24 * time.Hour,
	HighWatermarkPercent:   85,
	LowWatermarkPercent:    75,
	ProtectedPercent:       80,
	AccessUpdateInterval:   time.Minute,
}

type diskFaultResult struct {
	SchemaVersion   string                  `json:"schemaVersion"`
	Qualification   string                  `json:"qualification"`
	BenchmarkDigest string                  `json:"benchmarkDigest"`
	Seed            int64                   `json:"seed"`
	StartedAt       string                  `json:"startedAt"`
	CompletedAt     string                  `json:"completedAt"`
	FaultOutcomes   []faultOutcome          `json:"faultOutcomes"`
	Triggers        []diskFaultTrigger      `json:"triggers"`
	Recovery        []recoveryObservation   `json:"recovery"`
	RawObservations rawObservationsIdentity `json:"rawObservations"`
}

type diskFaultTrigger struct {
	ID       string `json:"id"`
	Sequence int64  `json:"sequence"`
}

type diskFaultObservation struct {
	SchemaVersion string            `json:"schemaVersion"`
	Sequence      int64             `json:"sequence"`
	FaultID       string            `json:"faultId"`
	Event         string            `json:"event"`
	DurationNs    int64             `json:"durationNs"`
	Metrics       []diskFaultMetric `json:"metrics"`
}

type diskFaultMetric struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type diskFaultSink struct {
	mutex    sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	hash     hashWriter
	sequence int64
	count    int64
	size     int64
}

type countedReader struct {
	reader *bytes.Reader
	bytes  atomic.Int64
}

// RunDiskFaults executes the high-watermark and out-of-space rows from the
// pinned benchmark manifest against reduced real Shared stores and loopback
// HTTP. It emits raw trigger observations before an exact fault summary.
func RunDiskFaults(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
) (string, error) {
	if ctx == nil {
		return "", errors.New("benchmark disk-fault context is required")
	}
	if !filepath.IsAbs(stateDirectory) {
		return "", errors.New(
			"benchmark disk-fault state directory must be absolute",
		)
	}
	if pathsOverlap(stateDirectory, outputDirectory) {
		return "", errors.New(
			"benchmark disk-fault state and output directories must be separate",
		)
	}
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	if err := prepareOutputDirectory(outputDirectory); err != nil {
		return "", err
	}
	sink, err := newDiskFaultSink(outputDirectory)
	if err != nil {
		return "", err
	}
	sinkClosed := false
	defer func() {
		if !sinkClosed {
			sink.abort()
		}
	}()

	startedAt := time.Now().UTC()
	highOutcome, highTrigger, highRecovery, err :=
		runHighWatermarkFault(
			ctx,
			loaded,
			manifestDigest,
			filepath.Join(stateDirectory, "high-watermark"),
			sink,
		)
	if err != nil {
		return "", fmt.Errorf("run high-watermark fault: %w", err)
	}
	spaceOutcome, spaceTrigger, spaceRecovery, err :=
		runOutOfSpaceFault(
			ctx,
			loaded,
			manifestDigest,
			filepath.Join(stateDirectory, "out-of-space"),
			sink,
		)
	if err != nil {
		return "", fmt.Errorf("run out-of-space fault: %w", err)
	}
	rawIdentity, err := sink.close()
	if err != nil {
		return "", err
	}
	sinkClosed = true

	result := diskFaultResult{
		SchemaVersion:   diskFaultResultSchemaVersion,
		Qualification:   "FAULT_SLICE",
		BenchmarkDigest: manifestDigest,
		Seed:            loaded.Seed,
		StartedAt:       startedAt.Format(time.RFC3339Nano),
		CompletedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		FaultOutcomes: []faultOutcome{
			highOutcome,
			spaceOutcome,
		},
		Triggers: []diskFaultTrigger{
			highTrigger,
			spaceTrigger,
		},
		Recovery: []recoveryObservation{
			highRecovery,
			spaceRecovery,
		},
		RawObservations: rawIdentity,
	}
	resultPath, err := writeDiskFaultResult(outputDirectory, result)
	if err != nil {
		return "", err
	}
	if err := ValidateDiskFaultResult(manifestPath, resultPath); err != nil {
		return "", fmt.Errorf("validate benchmark disk-fault result: %w", err)
	}
	return resultPath, nil
}

func runHighWatermarkFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *diskFaultSink,
) (
	faultOutcome,
	diskFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "DISK_HIGH_WATERMARK"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := sharedcache.OpenWithCapacity(
		ctx,
		stateDirectory,
		100,
		diskFaultPolicy,
	)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	signingKey := benchmarkSigningKey(manifestDigest)
	const namespace = int64(700)
	for index := 0; index < 8; index++ {
		if _, err := publishDiskFaultObject(
			ctx,
			storage,
			signingKey,
			manifestDigest,
			loaded.Seed,
			namespace,
			index,
		); err != nil {
			return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
		}
	}
	before, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if before.StableBytes != 800 ||
		before.HighWatermarkBytes != 850 ||
		before.LowWatermarkBytes != 750 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("unexpected pre-fault capacity: %+v", before)
	}
	triggerSequence, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "PROJECTED_HIGH_WATERMARK",
		DurationNs: 0,
		Metrics: []diskFaultMetric{
			{Name: "stableBytes", Value: before.StableBytes},
			{Name: "incomingBytes", Value: 100},
			{Name: "highWatermarkBytes", Value: before.HighWatermarkBytes},
			{Name: "lowWatermarkBytes", Value: before.LowWatermarkBytes},
		},
	})
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	if _, err := publishDiskFaultObject(
		ctx,
		storage,
		signingKey,
		manifestDigest,
		loaded.Seed,
		namespace,
		8,
	); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	after, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if after.StableBytes != 700 ||
		after.HighWatermarkReached ||
		after.ProbationBytes != 700 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("unexpected post-fault capacity: %+v", after)
	}
	if _, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "EVICTED_TO_LOW_WATERMARK",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []diskFaultMetric{
			{Name: "stableBytes", Value: after.StableBytes},
			{Name: "probationBytes", Value: after.ProbationBytes},
			{Name: "highWatermarkReached", Value: 0},
		},
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	misses, hits, err := observeHighWatermarkAuthority(
		ctx,
		storage,
		namespace,
	)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if misses != 2 || hits != 1 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"unexpected post-eviction authority: misses=%d hits=%d",
				misses,
				hits,
			)
	}
	if _, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "AUTHORITY_REMOVED_BEFORE_BLOB_CLEANUP",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []diskFaultMetric{
			{Name: "evictedMisses", Value: int64(misses)},
			{Name: "retainedHits", Value: int64(hits)},
		},
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	maintenance, err := storage.MaintainCapacity(ctx)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if maintenance.DeletedUnreferencedBlob != 2 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("unexpected high-watermark cleanup: %+v", maintenance)
	}
	if _, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "PHYSICAL_ORPHANS_COLLECTED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []diskFaultMetric{
			{
				Name:  "deletedUnreferencedBlobs",
				Value: int64(maintenance.DeletedUnreferencedBlob),
			},
		},
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return faultOutcome{
			ID:             faultID,
			ExpectedSafety: expectedSafety,
			Status:         "PASSED",
		}, diskFaultTrigger{
			ID:       faultID,
			Sequence: triggerSequence,
		}, recoveryObservation{
			ID:         faultID,
			DurationNs: recoveryDuration,
			Status:     "EVICTED_TO_LOW_AND_VERIFIED",
		}, nil
}

func runOutOfSpaceFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *diskFaultSink,
) (
	faultOutcome,
	diskFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "DISK_OUT_OF_SPACE"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	var available atomic.Uint64
	available.Store(50)
	storage, err := sharedcache.OpenWithCapacityProbe(
		ctx,
		stateDirectory,
		100,
		diskFaultPolicy,
		func(string) (uint64, uint64, error) {
			return 1_000, available.Load(), nil
		},
	)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	const namespace = int64(701)
	request := benchmarkAttemptRequest(
		manifestDigest,
		namespace,
		1,
		time.Now().UTC(),
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: namespace,
			PendingAttemptID:    request.AttemptID,
			AllowWrite:          true,
		},
	)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := benchmarkHTTPClient(1)
	defer client.CloseIdleConnections()

	triggerSequence, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "DISK_AVAILABILITY_BELOW_REQUEST",
		DurationNs: 0,
		Metrics: []diskFaultMetric{
			{Name: "diskAvailableBytes", Value: 50},
			{Name: "requestBytes", Value: 60},
		},
	})
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	body := &countedReader{reader: bytes.NewReader(bytes.Repeat([]byte{9}, 60))}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		server.URL+"/cache/out-of-space",
		body,
	)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	httpRequest.ContentLength = 60
	httpRequest.Header.Set("Expect", "100-continue")
	rejectedStarted := time.Now()
	response, err := client.Do(httpRequest)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	responseBytes, readErr := io.Copy(io.Discard, response.Body)
	closeErr := response.Body.Close()
	bodyReadBytes := body.bytes.Load()
	if readErr != nil ||
		closeErr != nil ||
		response.StatusCode != http.StatusRequestEntityTooLarge ||
		responseBytes != 0 ||
		bodyReadBytes != 0 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"unsafe out-of-space rejection: status=%d response=%d body-read=%d",
				response.StatusCode,
				responseBytes,
				bodyReadBytes,
			)
	}
	if _, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "PUT_REJECTED_BEFORE_BODY_READ",
		DurationNs: time.Since(rejectedStarted).Nanoseconds(),
		Metrics: []diskFaultMetric{
			{Name: "httpStatus", Value: http.StatusRequestEntityTooLarge},
			{Name: "bodyReadBytes", Value: bodyReadBytes},
		},
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	snapshot, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if snapshot.StableBytes != 0 ||
		snapshot.PendingBytes != 0 ||
		snapshot.ReservedBytes != 0 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("out-of-space left capacity state: %+v", snapshot)
	}
	if _, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "NO_PARTIAL_AUTHORITY",
		DurationNs: time.Since(rejectedStarted).Nanoseconds(),
		Metrics: []diskFaultMetric{
			{Name: "stableBytes", Value: snapshot.StableBytes},
			{Name: "pendingBytes", Value: snapshot.PendingBytes},
			{Name: "reservedBytes", Value: snapshot.ReservedBytes},
		},
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	available.Store(1_000)
	recoveryStarted := time.Now()
	operation := workloadOperation{
		object: workloadObject{
			index: 900,
			size:  60,
			key:   "out-of-space",
		},
	}
	recovery, digest := executePUT(
		ctx,
		client,
		server.URL,
		operation,
		loaded.Seed,
	)
	if recovery.ErrorClass != "" ||
		recovery.Status != http.StatusCreated ||
		recovery.RequestBytes != 60 ||
		recovery.ResponseBytes != 0 ||
		digest == "" {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("out-of-space recovery failed: %+v", recovery)
	}
	if _, err := sink.append(diskFaultObservation{
		FaultID:    faultID,
		Event:      "PUT_RECOVERED_AFTER_AVAILABILITY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []diskFaultMetric{
			{Name: "httpStatus", Value: int64(recovery.Status)},
			{Name: "requestBytes", Value: recovery.RequestBytes},
		},
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := storage.AbortAttempt(ctx, sharedcache.AbortAttemptRequest{
		RequestID:            "abort-" + request.AttemptID,
		AttemptID:            request.AttemptID,
		ExpectedStateVersion: status.StateVersion,
		Reason:               "CANCELLED",
	}); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	cleanup, err := storage.MaintainCapacity(ctx)
	if err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	if cleanup.DeletedUnreferencedBlob != 1 {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("unexpected out-of-space cleanup: %+v", cleanup)
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, diskFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return faultOutcome{
			ID:             faultID,
			ExpectedSafety: expectedSafety,
			Status:         "PASSED",
		}, diskFaultTrigger{
			ID:       faultID,
			Sequence: triggerSequence,
		}, recoveryObservation{
			ID:         faultID,
			DurationNs: recoveryDuration,
			Status:     "PUT_ACCEPTED_AFTER_DISK_RECOVERY",
		}, nil
}

func publishDiskFaultObject(
	ctx context.Context,
	storage *sharedcache.Storage,
	signingKey ed25519.PrivateKey,
	manifestDigest string,
	seed int64,
	namespace int64,
	index int,
) (sharedcache.CommitObject, error) {
	request := benchmarkAttemptRequest(
		manifestDigest,
		namespace,
		index+1,
		time.Now().UTC(),
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		return sharedcache.CommitObject{}, err
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: namespace,
			PendingAttemptID:    request.AttemptID,
			AllowWrite:          true,
		},
	)
	if err != nil {
		return sharedcache.CommitObject{}, err
	}
	server := httptest.NewServer(handler)
	client := benchmarkHTTPClient(1)
	operation := workloadOperation{
		object: workloadObject{
			index: index,
			size:  100,
			key:   fmt.Sprintf("disk-fault-object-%02d", index),
		},
	}
	observation, digest := executePUT(
		ctx,
		client,
		server.URL,
		operation,
		seed,
	)
	client.CloseIdleConnections()
	server.Close()
	if observation.ErrorClass != "" ||
		observation.Status != http.StatusCreated ||
		observation.RequestBytes != 100 ||
		observation.ResponseBytes != 0 ||
		digest == "" {
		return sharedcache.CommitObject{}, fmt.Errorf(
			"publish disk-fault object %d: %+v",
			index,
			observation,
		)
	}
	object := sharedcache.CommitObject{
		NamespaceGeneration: namespace,
		Key:                 operation.object.key,
		Checksum:            digest,
		SizeBytes:           100,
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return sharedcache.CommitObject{}, err
	}
	verified, err := verifyBenchmarkDecision(
		ctx,
		signingKey,
		request,
		status,
		[]sharedcache.CommitObject{object},
		time.Now().UTC(),
	)
	if err != nil {
		return sharedcache.CommitObject{}, err
	}
	if _, err := storage.CommitAttempt(
		ctx,
		status.StateVersion,
		1,
		verified,
	); err != nil {
		return sharedcache.CommitObject{}, err
	}
	return object, nil
}

func observeHighWatermarkAuthority(
	ctx context.Context,
	storage *sharedcache.Storage,
	namespace int64,
) (int, int, error) {
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              "beta-smoke",
			NamespaceGeneration: namespace,
			AllowRead:           true,
		},
	)
	if err != nil {
		return 0, 0, err
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := benchmarkHTTPClient(1)
	defer client.CloseIdleConnections()
	misses := 0
	hits := 0
	for _, check := range []struct {
		index       int
		expectedHit bool
	}{
		{index: 0, expectedHit: false},
		{index: 1, expectedHit: false},
		{index: 8, expectedHit: true},
	} {
		operation := workloadOperation{
			object: workloadObject{
				index: check.index,
				size:  100,
				key: fmt.Sprintf(
					"disk-fault-object-%02d",
					check.index,
				),
			},
			expectedHit: check.expectedHit,
			key:         fmt.Sprintf("disk-fault-object-%02d", check.index),
		}
		observation := executeGET(ctx, client, server.URL, operation)
		if observation.ErrorClass != "" {
			return misses, hits, fmt.Errorf(
				"observe disk-fault authority: %+v",
				observation,
			)
		}
		if check.expectedHit {
			hits++
		} else {
			misses++
		}
	}
	return misses, hits, nil
}

func expectedFaultSafety(loaded manifest, id string) (string, error) {
	for _, fault := range loaded.Faults {
		if fault.ID == id {
			return fault.ExpectedSafety, nil
		}
	}
	return "", fmt.Errorf("benchmark manifest lacks fault %q", id)
}

func newDiskFaultSink(outputDirectory string) (*diskFaultSink, error) {
	path := filepath.Join(outputDirectory, diskFaultRawFilename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, errors.New("create benchmark disk-fault observations")
	}
	digest := sha256.New()
	return &diskFaultSink{
		file:   file,
		writer: bufio.NewWriter(io.MultiWriter(file, digest)),
		hash: hashWriter{
			sum: func() []byte {
				return digest.Sum(nil)
			},
		},
	}, nil
}

func (sink *diskFaultSink) append(
	observation diskFaultObservation,
) (int64, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	sink.sequence++
	observation.SchemaVersion = diskFaultRawSchemaVersion
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

func (sink *diskFaultSink) close() (rawObservationsIdentity, error) {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file == nil {
		return rawObservationsIdentity{}, errors.New(
			"benchmark disk-fault observations are already closed",
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
		Path:      diskFaultRawFilename,
		SHA256:    "sha256:" + hex.EncodeToString(sink.hash.sum()),
		Count:     sink.count,
		SizeBytes: sink.size,
	}, nil
}

func (sink *diskFaultSink) abort() {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	if sink.file != nil {
		_ = sink.file.Close()
		sink.file = nil
	}
}

func (reader *countedReader) Read(destination []byte) (int, error) {
	count, err := reader.reader.Read(destination)
	reader.bytes.Add(int64(count))
	return count, err
}

func writeDiskFaultResult(
	outputDirectory string,
	result diskFaultResult,
) (string, error) {
	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	content = append(content, '\n')
	temporary, err := os.CreateTemp(
		outputDirectory,
		".disk-fault-result-*.tmp",
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
	target := filepath.Join(outputDirectory, diskFaultResultFilename)
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

// ValidateDiskFaultResult verifies the exact disk fault slice and its raw
// trigger stream without treating it as the complete 15-fault qualification.
func ValidateDiskFaultResult(manifestPath string, resultPath string) error {
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
			"benchmark disk-fault result is not a bounded private file",
		)
	}
	raw, err := os.ReadFile(resultPath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var result diskFaultResult
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode benchmark disk-fault result: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New(
			"benchmark disk-fault result contains trailing JSON",
		)
	}
	if result.SchemaVersion != diskFaultResultSchemaVersion ||
		result.Qualification != "FAULT_SLICE" ||
		result.BenchmarkDigest != manifestDigest ||
		result.Seed != loaded.Seed ||
		result.RawObservations.Path != diskFaultRawFilename ||
		result.RawObservations.Count != 8 ||
		result.RawObservations.SizeBytes < 1 ||
		result.RawObservations.SizeBytes > maximumResultBytes {
		return errors.New("benchmark disk-fault result is incomplete")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, result.StartedAt)
	if err != nil {
		return errors.New(
			"benchmark disk-fault result has invalid startedAt",
		)
	}
	completedAt, err := time.Parse(time.RFC3339Nano, result.CompletedAt)
	if err != nil || completedAt.Before(startedAt) {
		return errors.New(
			"benchmark disk-fault result has invalid completedAt",
		)
	}
	expectedIDs := []string{
		"DISK_HIGH_WATERMARK",
		"DISK_OUT_OF_SPACE",
	}
	if len(result.FaultOutcomes) != len(expectedIDs) ||
		len(result.Triggers) != len(expectedIDs) ||
		len(result.Recovery) != len(expectedIDs) {
		return errors.New("benchmark disk-fault summary is incomplete")
	}
	for index, id := range expectedIDs {
		expectedSafety, err := expectedFaultSafety(loaded, id)
		if err != nil {
			return err
		}
		expectedTrigger := int64(1)
		if index == 1 {
			expectedTrigger = 5
		}
		expectedRecoveryStatus := "EVICTED_TO_LOW_AND_VERIFIED"
		if index == 1 {
			expectedRecoveryStatus = "PUT_ACCEPTED_AFTER_DISK_RECOVERY"
		}
		if result.FaultOutcomes[index] != (faultOutcome{
			ID:             id,
			ExpectedSafety: expectedSafety,
			Status:         "PASSED",
		}) ||
			result.Triggers[index] != (diskFaultTrigger{
				ID:       id,
				Sequence: expectedTrigger,
			}) ||
			result.Recovery[index].ID != id ||
			result.Recovery[index].DurationNs <= 0 ||
			result.Recovery[index].Status != expectedRecoveryStatus {
			return errors.New("benchmark disk-fault summary drifted")
		}
	}
	observations, err := readDiskFaultObservations(
		filepath.Dir(resultPath),
		result.RawObservations,
	)
	if err != nil {
		return err
	}
	return validateRawDiskFaultObservations(observations)
}

func readDiskFaultObservations(
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
			"benchmark disk-fault observations are not a private file",
		)
	}
	content, err := os.ReadFile(path)
	if err != nil || int64(len(content)) != identity.SizeBytes {
		return nil, errors.New(
			"benchmark disk-fault observations are unavailable",
		)
	}
	digest := sha256.Sum256(content)
	if identity.SHA256 != "sha256:"+hex.EncodeToString(digest[:]) {
		return nil, errors.New(
			"benchmark disk-fault observation digest mismatched",
		)
	}
	if int64(bytes.Count(content, []byte{'\n'})) != identity.Count {
		return nil, errors.New(
			"benchmark disk-fault observation count mismatched",
		)
	}
	return content, nil
}

func validateRawDiskFaultObservations(content []byte) error {
	expected := []diskFaultObservation{
		{
			FaultID: "DISK_HIGH_WATERMARK",
			Event:   "PROJECTED_HIGH_WATERMARK",
			Metrics: []diskFaultMetric{
				{Name: "stableBytes", Value: 800},
				{Name: "incomingBytes", Value: 100},
				{Name: "highWatermarkBytes", Value: 850},
				{Name: "lowWatermarkBytes", Value: 750},
			},
		},
		{
			FaultID: "DISK_HIGH_WATERMARK",
			Event:   "EVICTED_TO_LOW_WATERMARK",
			Metrics: []diskFaultMetric{
				{Name: "stableBytes", Value: 700},
				{Name: "probationBytes", Value: 700},
				{Name: "highWatermarkReached", Value: 0},
			},
		},
		{
			FaultID: "DISK_HIGH_WATERMARK",
			Event:   "AUTHORITY_REMOVED_BEFORE_BLOB_CLEANUP",
			Metrics: []diskFaultMetric{
				{Name: "evictedMisses", Value: 2},
				{Name: "retainedHits", Value: 1},
			},
		},
		{
			FaultID: "DISK_HIGH_WATERMARK",
			Event:   "PHYSICAL_ORPHANS_COLLECTED",
			Metrics: []diskFaultMetric{
				{Name: "deletedUnreferencedBlobs", Value: 2},
			},
		},
		{
			FaultID: "DISK_OUT_OF_SPACE",
			Event:   "DISK_AVAILABILITY_BELOW_REQUEST",
			Metrics: []diskFaultMetric{
				{Name: "diskAvailableBytes", Value: 50},
				{Name: "requestBytes", Value: 60},
			},
		},
		{
			FaultID: "DISK_OUT_OF_SPACE",
			Event:   "PUT_REJECTED_BEFORE_BODY_READ",
			Metrics: []diskFaultMetric{
				{Name: "httpStatus", Value: 413},
				{Name: "bodyReadBytes", Value: 0},
			},
		},
		{
			FaultID: "DISK_OUT_OF_SPACE",
			Event:   "NO_PARTIAL_AUTHORITY",
			Metrics: []diskFaultMetric{
				{Name: "stableBytes", Value: 0},
				{Name: "pendingBytes", Value: 0},
				{Name: "reservedBytes", Value: 0},
			},
		},
		{
			FaultID: "DISK_OUT_OF_SPACE",
			Event:   "PUT_RECOVERED_AFTER_AVAILABILITY",
			Metrics: []diskFaultMetric{
				{Name: "httpStatus", Value: 201},
				{Name: "requestBytes", Value: 60},
			},
		},
	}
	reader := bufio.NewReader(bytes.NewReader(content))
	index := 0
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) && len(line) == 0 {
			break
		}
		if err != nil || index >= len(expected) {
			return errors.New(
				"read benchmark disk-fault observation",
			)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		var observation diskFaultObservation
		if err := decoder.Decode(&observation); err != nil {
			return errors.New(
				"decode benchmark disk-fault observation",
			)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return errors.New(
				"benchmark disk-fault observation contains trailing JSON",
			)
		}
		want := expected[index]
		if observation.SchemaVersion != diskFaultRawSchemaVersion ||
			observation.Sequence != int64(index+1) ||
			observation.FaultID != want.FaultID ||
			observation.Event != want.Event ||
			observation.DurationNs < 0 ||
			!slices.Equal(observation.Metrics, want.Metrics) {
			return errors.New(
				"benchmark disk-fault observation drifted",
			)
		}
		index++
	}
	if index != len(expected) {
		return errors.New(
			"benchmark disk-fault observation matrix is incomplete",
		)
	}
	return nil
}
