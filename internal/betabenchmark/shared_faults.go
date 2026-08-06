package betabenchmark

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	sharedFaultSmallObjectBytes int64 = 1_000
	sharedFaultLargeObjectBytes int64 = 1 << 20
)

type stagedSharedFaultObject struct {
	request  sharedcache.StartAttemptRequest
	status   sharedcache.AttemptStatus
	object   sharedcache.CommitObject
	verified sharedcache.VerifiedCommitDecision
}

type sharedCommitResult struct {
	result sharedcache.CommitResult
	err    error
}

// RunSharedFaults executes the seven Shared data-plane rows from the pinned
// benchmark manifest and writes their trigger/recovery observations before
// the strict slice summary.
func RunSharedFaults(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
) (string, error) {
	if ctx == nil {
		return "", errors.New("benchmark Shared-fault context is required")
	}
	if !filepath.IsAbs(stateDirectory) {
		return "", errors.New(
			"benchmark Shared-fault state directory must be absolute",
		)
	}
	if pathsOverlap(stateDirectory, outputDirectory) {
		return "", errors.New(
			"benchmark Shared-fault state and output directories must be separate",
		)
	}
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	if err := prepareOutputDirectory(outputDirectory); err != nil {
		return "", err
	}
	sink, err := newSharedFaultSink(outputDirectory)
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
	type faultRun func() (
		faultOutcome,
		sharedFaultTrigger,
		recoveryObservation,
		error,
	)
	runs := []faultRun{
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runMidPutCancelFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "mid-put-cancel"),
				sink,
			)
		},
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runMidGetCancelFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "mid-get-cancel"),
				sink,
			)
		},
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runBlobIntegrityFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "truncated-blob"),
				sink,
				"TRUNCATED_BLOB",
			)
		},
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runBlobIntegrityFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "corrupt-blob"),
				sink,
				"CORRUPT_BLOB",
			)
		},
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runSQLiteBusyFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "sqlite-busy"),
				sink,
			)
		},
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runExpiredLeaseFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "expired-lease"),
				sink,
			)
		},
		func() (
			faultOutcome,
			sharedFaultTrigger,
			recoveryObservation,
			error,
		) {
			return runDeathBeforeCommitFault(
				ctx,
				loaded,
				manifestDigest,
				filepath.Join(stateDirectory, "death-before-commit"),
				sink,
			)
		},
	}
	outcomes := make([]faultOutcome, 0, len(runs))
	triggers := make([]sharedFaultTrigger, 0, len(runs))
	recovery := make([]recoveryObservation, 0, len(runs))
	for _, run := range runs {
		outcome, trigger, recovered, err := run()
		if err != nil {
			return "", err
		}
		outcomes = append(outcomes, outcome)
		triggers = append(triggers, trigger)
		recovery = append(recovery, recovered)
	}
	rawIdentity, err := sink.close()
	if err != nil {
		return "", err
	}
	sinkClosed = true
	result := sharedFaultResult{
		SchemaVersion:   sharedFaultResultSchemaVersion,
		Qualification:   "FAULT_SLICE",
		BenchmarkDigest: manifestDigest,
		Seed:            loaded.Seed,
		StartedAt:       startedAt.Format(time.RFC3339Nano),
		CompletedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		FaultOutcomes:   outcomes,
		Triggers:        triggers,
		Recovery:        recovery,
		RawObservations: rawIdentity,
	}
	resultPath, err := writeSharedFaultResult(outputDirectory, result)
	if err != nil {
		return "", err
	}
	if err := ValidateSharedFaultResult(manifestPath, resultPath); err != nil {
		return "", fmt.Errorf("validate benchmark Shared-fault result: %w", err)
	}
	return resultPath, nil
}

func runMidPutCancelFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *sharedFaultSink,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "MID_PUT_CANCEL"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	request := benchmarkAttemptRequest(
		manifestDigest,
		800,
		1,
		time.Now().UTC(),
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: request.NamespaceGeneration,
			PendingAttemptID:    request.AttemptID,
			AllowWrite:          true,
		},
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := benchmarkHTTPClient(1)
	defer client.CloseIdleConnections()
	putContext, cancel := context.WithCancel(ctx)
	defer cancel()
	bodyReader, bodyWriter := io.Pipe()
	httpRequest, err := http.NewRequestWithContext(
		putContext,
		http.MethodPut,
		server.URL+"/cache/mid-put",
		bodyReader,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	httpRequest.ContentLength = sharedFaultSmallObjectBytes
	requestDone := make(chan error, 1)
	go func() {
		response, requestErr := client.Do(httpRequest)
		if response != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()
	partial := bytes.Repeat([]byte{3}, 100)
	if _, err := bodyWriter.Write(partial); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	spoolFiles, err := waitForSpoolFiles(storage.Layout().Spool, 1)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	triggerSequence, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "PARTIAL_PUT_STARTED",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "bodyBytesSent", Value: int64(len(partial))},
			{Name: "spoolFiles", Value: int64(spoolFiles)},
		},
	})
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	cancel()
	_ = bodyWriter.CloseWithError(context.Canceled)
	select {
	case <-requestDone:
	case <-time.After(2 * time.Second):
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			errors.New("cancelled PUT did not terminate")
	}
	if err := waitForEmptySpool(storage.Layout().Spool); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	snapshot, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if snapshot.PendingBytes != 0 ||
		snapshot.StableBytes != 0 ||
		snapshot.ReservedBytes != 0 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("cancelled PUT left capacity: %+v", snapshot)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "CANCELLED_WITHOUT_PARTIAL_AUTHORITY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "pendingBytes", Value: snapshot.PendingBytes},
			{Name: "stableBytes", Value: snapshot.StableBytes},
			{Name: "reservedBytes", Value: snapshot.ReservedBytes},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	retry := workloadOperation{
		object: workloadObject{
			index: 1_000,
			size:  sharedFaultSmallObjectBytes,
			key:   "mid-put",
		},
	}
	retryObservation, digest := executePUT(
		ctx,
		client,
		server.URL,
		retry,
		loaded.Seed,
	)
	if retryObservation.ErrorClass != "" ||
		retryObservation.Status != http.StatusCreated ||
		digest == "" {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("cancelled PUT retry failed: %+v", retryObservation)
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := storage.AbortAttempt(ctx, sharedcache.AbortAttemptRequest{
		RequestID:            "abort-" + request.AttemptID,
		AttemptID:            request.AttemptID,
		ExpectedStateVersion: status.StateVersion,
		Reason:               "CANCELLED",
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	cleanup, err := storage.MaintainCapacity(ctx)
	if err != nil || cleanup.DeletedUnreferencedBlob != 1 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("cancelled PUT cleanup: %+v/%v", cleanup, err)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "COMPLETE_RETRY_ACCEPTED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(retryObservation.Status)},
			{Name: "requestBytes", Value: retryObservation.RequestBytes},
			{
				Name:  "deletedUnreferencedBlobs",
				Value: int64(cleanup.DeletedUnreferencedBlob),
			},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return passedSharedFault(
		faultID,
		expectedSafety,
		triggerSequence,
		recoveryDuration,
		"COMPLETE_RETRY_ACCEPTED",
	)
}

func runMidGetCancelFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *sharedFaultSink,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "MID_GET_CANCEL"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultLargeObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	signingKey := benchmarkSigningKey(manifestDigest)
	staged, err := stageSharedFaultObject(
		ctx,
		storage,
		signingKey,
		manifestDigest,
		loaded.Seed,
		801,
		1,
		"mid-get",
		sharedFaultLargeObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := storage.CommitAttempt(
		ctx,
		staged.status.StateVersion,
		1,
		staged.verified,
	); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              staged.request.Repository.Tenant,
			NamespaceGeneration: staged.request.NamespaceGeneration,
			AllowRead:           true,
		},
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := benchmarkHTTPClient(1)
	defer client.CloseIdleConnections()
	getContext, cancel := context.WithCancel(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(
		getContext,
		http.MethodGet,
		server.URL+"/cache/mid-get",
		nil,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	partial := make([]byte, 4_096)
	read, readErr := io.ReadFull(response.Body, partial)
	if readErr != nil || read != len(partial) {
		_ = response.Body.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("read partial GET: %d/%v", read, readErr)
	}
	triggerSequence, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "GET_CANCELLED_AFTER_PARTIAL_READ",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "bodyBytesRead", Value: int64(read)},
			{Name: "objectBytes", Value: sharedFaultLargeObjectBytes},
			{Name: "acceptedHit", Value: 0},
		},
	})
	if err != nil {
		_ = response.Body.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	cancel()
	_ = response.Body.Close()

	retryRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		server.URL+"/cache/mid-get",
		nil,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	retryResponse, err := client.Do(retryRequest)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	content, readErr := io.ReadAll(retryResponse.Body)
	closeErr := retryResponse.Body.Close()
	digest := sha256.Sum256(content)
	digestMatches := "sha256:"+fmt.Sprintf("%x", digest[:]) ==
		staged.object.Checksum
	if readErr != nil ||
		closeErr != nil ||
		retryResponse.StatusCode != http.StatusOK ||
		int64(len(content)) != sharedFaultLargeObjectBytes ||
		!digestMatches {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"incomplete GET retry: status=%d bytes=%d digest=%t read=%v close=%v",
				retryResponse.StatusCode,
				len(content),
				digestMatches,
				readErr,
				closeErr,
			)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "COMPLETE_RETRY_VERIFIED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(retryResponse.StatusCode)},
			{Name: "responseBytes", Value: int64(len(content))},
			{Name: "digestMatch", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return passedSharedFault(
		faultID,
		expectedSafety,
		triggerSequence,
		recoveryDuration,
		"COMPLETE_RETRY_VERIFIED",
	)
}

func runBlobIntegrityFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *sharedFaultSink,
	faultID string,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	namespace := int64(802)
	key := "truncated"
	if faultID == "CORRUPT_BLOB" {
		namespace = 803
		key = "corrupt"
	}
	staged, err := stageSharedFaultObject(
		ctx,
		storage,
		benchmarkSigningKey(manifestDigest),
		manifestDigest,
		loaded.Seed,
		namespace,
		1,
		key,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := storage.CommitAttempt(
		ctx,
		staged.status.StateVersion,
		1,
		staged.verified,
	); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	path, err := sharedFaultBlobPath(storage.Layout(), staged.object.Checksum)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	event := "BLOB_TRUNCATED"
	actualBytes := sharedFaultSmallObjectBytes - 1
	if faultID == "TRUNCATED_BLOB" {
		err = os.Truncate(path, actualBytes)
	} else {
		event = "BLOB_CORRUPTED"
		actualBytes = sharedFaultSmallObjectBytes
		err = os.WriteFile(
			path,
			bytes.Repeat([]byte{0xff}, int(sharedFaultSmallObjectBytes)),
			0o600,
		)
	}
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	triggerSequence, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      event,
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "expectedBytes", Value: sharedFaultSmallObjectBytes},
			{Name: "actualBytes", Value: actualBytes},
		},
	})
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              staged.request.Repository.Tenant,
			NamespaceGeneration: staged.request.NamespaceGeneration,
			AllowRead:           true,
		},
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	server := httptest.NewServer(handler)
	client := benchmarkHTTPClient(1)
	observation := executeGET(
		ctx,
		client,
		server.URL,
		workloadOperation{
			object: workloadObject{
				index: 1,
				size:  sharedFaultSmallObjectBytes,
				key:   key,
			},
			key: key,
		},
	)
	client.CloseIdleConnections()
	server.Close()
	if observation.ErrorClass != "" ||
		observation.Status != http.StatusNotFound ||
		observation.ResponseBytes != 0 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("unsafe corrupt read: %+v", observation)
	}
	snapshot, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	quarantineFiles, err := countRegularFiles(storage.Layout().Quarantine)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if snapshot.StableBytes != 0 ||
		snapshot.QuarantineBytes != sharedFaultSmallObjectBytes ||
		quarantineFiles != 1 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"corrupt read did not quarantine: %+v files=%d",
				snapshot,
				quarantineFiles,
			)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "MISS_AND_QUARANTINE",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(observation.Status)},
			{Name: "responseBytes", Value: observation.ResponseBytes},
			{Name: "stableBytes", Value: snapshot.StableBytes},
			{Name: "quarantineFiles", Value: int64(quarantineFiles)},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return passedSharedFault(
		faultID,
		expectedSafety,
		triggerSequence,
		recoveryDuration,
		"MISS_AND_QUARANTINED",
	)
}

func runSQLiteBusyFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *sharedFaultSink,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "SQLITE_BUSY"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	staged, err := stageSharedFaultObject(
		ctx,
		storage,
		benchmarkSigningKey(manifestDigest),
		manifestDigest,
		loaded.Seed,
		804,
		1,
		"sqlite-busy",
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	lockDatabase, lockConnection, err := lockSharedCacheDatabase(
		ctx,
		storage.Layout().CacheDatabase,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	lockHeld := true
	defer func() {
		if lockHeld {
			_, _ = lockConnection.ExecContext(
				context.Background(),
				"ROLLBACK",
			)
		}
		_ = lockConnection.Close()
		_ = lockDatabase.Close()
	}()
	triggerSequence, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "SQLITE_WRITE_LOCK_HELD",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "committedObjects", Value: 0},
		},
	})
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	commitDone := make(chan sharedCommitResult, 1)
	go func() {
		result, commitErr := storage.CommitAttempt(
			ctx,
			staged.status.StateVersion,
			1,
			staged.verified,
		)
		commitDone <- sharedCommitResult{result: result, err: commitErr}
	}()
	timer := time.NewTimer(150 * time.Millisecond)
	defer timer.Stop()
	firstReturnedBusy := false
	select {
	case result := <-commitDone:
		if !isSQLiteBusyError(result.err) {
			return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
				fmt.Errorf(
					"SQLite-busy commit returned unexpected result: %+v/%v",
					result.result,
					result.err,
				)
		}
		firstReturnedBusy = true
	case <-timer.C:
	}
	var visible int64
	if err := lockConnection.QueryRowContext(
		ctx,
		`SELECT count(*) FROM committed_objects
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?`,
		staged.request.Repository.Tenant,
		staged.request.NamespaceGeneration,
		staged.object.Key,
	).Scan(&visible); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if visible != 0 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			errors.New("SQLite-busy commit became partially visible")
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "SQLITE_BUSY_WITHOUT_PARTIAL_VISIBILITY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "busyReturned", Value: boolMetric(firstReturnedBusy)},
			{Name: "committedObjects", Value: visible},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := lockConnection.ExecContext(ctx, "ROLLBACK"); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	lockHeld = false
	var committed sharedCommitResult
	if firstReturnedBusy {
		committed.result, committed.err = storage.CommitAttempt(
			ctx,
			staged.status.StateVersion,
			1,
			staged.verified,
		)
	} else {
		select {
		case committed = <-commitDone:
		case <-time.After(6 * time.Second):
			return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
				errors.New("SQLite-busy commit did not recover")
		}
	}
	if committed.err != nil ||
		committed.result.Outcome != "COMMITTED" ||
		committed.result.ObjectCount != 1 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"SQLite-busy commit recovery: %+v/%v",
				committed.result,
				committed.err,
			)
	}
	file, _, err := storage.OpenCommitted(
		ctx,
		staged.request.Repository.Tenant,
		staged.request.NamespaceGeneration,
		staged.object.Key,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	_ = file.Close()
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "COMMIT_RECOVERED_AFTER_LOCK_RELEASE",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "committedObjects", Value: 1},
			{Name: "completeHit", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := lockConnection.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if err := lockDatabase.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if err := storage.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return passedSharedFault(
		faultID,
		expectedSafety,
		triggerSequence,
		recoveryDuration,
		"COMMITTED_AFTER_LOCK_RELEASE",
	)
}

func runExpiredLeaseFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *sharedFaultSink,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "EXPIRED_LEASE"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()
	request := benchmarkAttemptRequest(
		manifestDigest,
		805,
		1,
		time.Now().UTC(),
	)
	// Leave enough time for the deliberately staged PUT to complete under the
	// race detector and contended CI hosts; the fault begins only after the
	// explicit wait below observes this lease as expired.
	request.LeaseExpiresAt = time.Now().UTC().Add(2 * time.Second)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	operation := workloadOperation{
		object: workloadObject{
			index: 1_001,
			size:  100,
			key:   "expired-lease",
		},
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: request.NamespaceGeneration,
			PendingAttemptID:    request.AttemptID,
			AllowWrite:          true,
		},
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	server := httptest.NewServer(handler)
	client := benchmarkHTTPClient(1)
	observation, digest := executePUT(
		ctx,
		client,
		server.URL,
		operation,
		loaded.Seed,
	)
	client.CloseIdleConnections()
	server.Close()
	if observation.ErrorClass != "" ||
		observation.Status != http.StatusCreated ||
		digest == "" {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("stage expired lease: %+v", observation)
	}
	before, err := storage.CapacitySnapshot(ctx)
	if err != nil || before.PendingBytes != 100 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("expired-lease precondition: %+v/%v", before, err)
	}
	wait := time.Until(request.LeaseExpiresAt) + 25*time.Millisecond
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
				ctx.Err()
		case <-timer.C:
		}
	}
	triggerSequence, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "LEASE_EXPIRED_WITH_PENDING_OBJECT",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "pendingBytes", Value: before.PendingBytes},
			{Name: "pendingObjects", Value: 1},
		},
	})
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	report, err := storage.Reconcile(ctx)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	after, err := storage.CapacitySnapshot(ctx)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if report.ExpiredAttempts != 1 ||
		report.DeletedOrphanBlobs != 1 ||
		status.State != sharedcache.AttemptAborted ||
		status.AbortReason != "LEASE_EXPIRED" ||
		after.PendingBytes != 0 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"expired lease recovery: report=%+v status=%+v capacity=%+v",
				report,
				status,
				after,
			)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "ABORTED_AND_OWNER_RELEASED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "expiredAttempts", Value: int64(report.ExpiredAttempts)},
			{
				Name:  "deletedOrphanBlobs",
				Value: int64(report.DeletedOrphanBlobs),
			},
			{Name: "pendingBytes", Value: after.PendingBytes},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storageClosed = true
	return passedSharedFault(
		faultID,
		expectedSafety,
		triggerSequence,
		recoveryDuration,
		"ABORTED_AND_OWNER_RELEASED",
	)
}

func runDeathBeforeCommitFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *sharedFaultSink,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "DEATH_BETWEEN_PENDING_AND_COMMIT"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	staged, err := stageSharedFaultObject(
		ctx,
		storage,
		benchmarkSigningKey(manifestDigest),
		manifestDigest,
		loaded.Seed,
		806,
		1,
		"death-before-commit",
		100,
	)
	if err != nil {
		_ = storage.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	before, err := storage.CapacitySnapshot(ctx)
	if err != nil || before.PendingBytes != 100 {
		_ = storage.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("death precondition: %+v/%v", before, err)
	}
	triggerSequence, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "PROCESS_DIED_WITH_PENDING_OBJECT",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "pendingBytes", Value: before.PendingBytes},
			{Name: "committedObjects", Value: 0},
		},
	})
	if err != nil {
		_ = storage.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	if err := storage.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	reopened, err := openSharedFaultStorage(
		ctx,
		stateDirectory,
		sharedFaultSmallObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	reopenedClosed := false
	defer func() {
		if !reopenedClosed {
			_ = reopened.Close()
		}
	}()
	readHandler, err := sharedcache.NewHTTPHandler(
		reopened,
		sharedcache.HTTPBinding{
			Tenant:              staged.request.Repository.Tenant,
			NamespaceGeneration: staged.request.NamespaceGeneration,
			AllowRead:           true,
		},
	)
	if err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	readServer := httptest.NewServer(readHandler)
	client := benchmarkHTTPClient(1)
	miss := executeGET(
		ctx,
		client,
		readServer.URL,
		workloadOperation{
			object: workloadObject{
				index: 1,
				size:  100,
				key:   staged.object.Key,
			},
			key: staged.object.Key,
		},
	)
	if miss.ErrorClass != "" || miss.Status != http.StatusNotFound {
		client.CloseIdleConnections()
		readServer.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("pending object became restart hit: %+v", miss)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "RESTARTED_PENDING_REMAINS_MISS",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(miss.Status)},
			{Name: "responseBytes", Value: miss.ResponseBytes},
		},
	}); err != nil {
		client.CloseIdleConnections()
		readServer.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	status, err := reopened.AttemptStatus(ctx, staged.request.AttemptID)
	if err != nil {
		client.CloseIdleConnections()
		readServer.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := reopened.CommitAttempt(
		ctx,
		status.StateVersion,
		1,
		staged.verified,
	); err != nil {
		client.CloseIdleConnections()
		readServer.Close()
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	hit := executeGET(
		ctx,
		client,
		readServer.URL,
		workloadOperation{
			object: workloadObject{
				index: 1,
				size:  100,
				key:   staged.object.Key,
			},
			expectedHit: true,
			key:         staged.object.Key,
		},
	)
	client.CloseIdleConnections()
	readServer.Close()
	if hit.ErrorClass != "" ||
		hit.Status != http.StatusOK ||
		hit.ResponseBytes != 100 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("authorized restart commit did not hit: %+v", hit)
	}
	if _, err := sink.append(sharedFaultObservation{
		FaultID:    faultID,
		Event:      "AUTHORIZED_COMMIT_RECOVERED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(hit.Status)},
			{Name: "responseBytes", Value: hit.ResponseBytes},
		},
	}); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryDuration := time.Since(recoveryStarted).Nanoseconds()
	if err := reopened.Close(); err != nil {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{}, err
	}
	reopenedClosed = true
	return passedSharedFault(
		faultID,
		expectedSafety,
		triggerSequence,
		recoveryDuration,
		"AUTHORIZED_COMMIT_RECOVERED",
	)
}

func openSharedFaultStorage(
	ctx context.Context,
	stateDirectory string,
	maximumObjectBytes int64,
) (*sharedcache.Storage, error) {
	deploymentBytes := maximumObjectBytes * 10
	return sharedcache.OpenWithCapacity(
		ctx,
		stateDirectory,
		maximumObjectBytes,
		sharedcache.CapacityPolicy{
			DeploymentBytes:        deploymentBytes,
			RepositoryBytes:        deploymentBytes,
			PendingQuarantineBytes: maximumObjectBytes,
			StableTTL:              30 * 24 * time.Hour,
			QuarantineTTL:          7 * 24 * time.Hour,
			HighWatermarkPercent:   85,
			LowWatermarkPercent:    75,
			ProtectedPercent:       80,
			AccessUpdateInterval:   time.Minute,
		},
	)
}

func stageSharedFaultObject(
	ctx context.Context,
	storage *sharedcache.Storage,
	signingKey ed25519.PrivateKey,
	manifestDigest string,
	seed int64,
	namespace int64,
	attemptOrdinal int,
	key string,
	size int64,
) (stagedSharedFaultObject, error) {
	request := benchmarkAttemptRequest(
		manifestDigest,
		namespace,
		attemptOrdinal,
		time.Now().UTC(),
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		return stagedSharedFaultObject{}, err
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: request.NamespaceGeneration,
			PendingAttemptID:    request.AttemptID,
			AllowWrite:          true,
		},
	)
	if err != nil {
		return stagedSharedFaultObject{}, err
	}
	server := httptest.NewServer(handler)
	client := benchmarkHTTPClient(1)
	operation := workloadOperation{
		object: workloadObject{
			index: int(namespace) + attemptOrdinal,
			size:  size,
			key:   key,
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
		observation.RequestBytes != size ||
		observation.ResponseBytes != 0 ||
		digest == "" {
		return stagedSharedFaultObject{}, fmt.Errorf(
			"stage Shared fault object: %+v",
			observation,
		)
	}
	object := sharedcache.CommitObject{
		NamespaceGeneration: request.NamespaceGeneration,
		Key:                 key,
		Checksum:            digest,
		SizeBytes:           size,
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return stagedSharedFaultObject{}, err
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
		return stagedSharedFaultObject{}, err
	}
	return stagedSharedFaultObject{
		request:  request,
		status:   status,
		object:   object,
		verified: verified,
	}, nil
}

func waitForSpoolFiles(path string, minimum int) (int, error) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		count, err := countRegularFiles(path)
		if err != nil {
			return 0, err
		}
		if count >= minimum {
			return count, nil
		}
		if !time.Now().Before(deadline) {
			return 0, errors.New("Shared fault spool did not become active")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForEmptySpool(path string) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		count, err := countRegularFiles(path)
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		if !time.Now().Before(deadline) {
			return errors.New("Shared fault spool did not become empty")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func countRegularFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(
		_ string,
		entry os.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			count++
		}
		return nil
	})
	return count, err
}

func sharedFaultBlobPath(
	layout sharedcache.Layout,
	digest string,
) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(digest, prefix) ||
		len(digest) != len(prefix)+sha256.Size*2 {
		return "", errors.New("Shared fault object has invalid digest")
	}
	hexDigest := strings.TrimPrefix(digest, prefix)
	return filepath.Join(
		layout.Blobs,
		hexDigest[:2],
		hexDigest[2:],
	), nil
}

func lockSharedCacheDatabase(
	ctx context.Context,
	path string,
) (*sql.DB, *sql.Conn, error) {
	databaseURL := &url.URL{Scheme: "file", Path: path}
	query := url.Values{}
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "wal")
	query.Set("_synchronous", "full")
	databaseURL.RawQuery = query.Encode()
	database, err := sql.Open("sqlite", databaseURL.String())
	if err != nil {
		return nil, nil, err
	}
	database.SetMaxOpenConns(1)
	connection, err := database.Conn(ctx)
	if err != nil {
		_ = database.Close()
		return nil, nil, err
	}
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		_ = connection.Close()
		_ = database.Close()
		return nil, nil, err
	}
	return database, connection, nil
}

func passedSharedFault(
	id string,
	expectedSafety string,
	triggerSequence int64,
	recoveryDuration int64,
	recoveryStatus string,
) (
	faultOutcome,
	sharedFaultTrigger,
	recoveryObservation,
	error,
) {
	if triggerSequence < 1 || recoveryDuration <= 0 {
		return faultOutcome{}, sharedFaultTrigger{}, recoveryObservation{},
			errors.New("Shared fault did not record trigger/recovery timing")
	}
	return faultOutcome{
			ID:             id,
			ExpectedSafety: expectedSafety,
			Status:         "PASSED",
		}, sharedFaultTrigger{
			ID:       id,
			Sequence: triggerSequence,
		}, recoveryObservation{
			ID:         id,
			DurationNs: recoveryDuration,
			Status:     recoveryStatus,
		}, nil
}

func isSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "sqlite_busy") ||
		strings.Contains(message, "database is locked")
}

func boolMetric(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
