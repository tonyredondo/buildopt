package betabenchmark

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

var smokeObjectMix = []objectMix{
	{SizeBytes: 4096, Percent: 70},
	{SizeBytes: 65536, Percent: 20},
	{SizeBytes: 524288, Percent: 8},
	{SizeBytes: 1048576, Percent: 2},
}

type workloadObject struct {
	index int
	size  int64
	key   string
}

type workloadOperation struct {
	object      workloadObject
	expectedHit bool
	key         string
}

type readEndpoint struct {
	server *httptest.Server
}

type deterministicReader struct {
	state     uint64
	remaining int64
	buffer    [8]byte
	used      int
}

// RunSmoke executes one exact 100-operation object-mix cycle for every
// manifest phase and client count through the real Shared HTTP data plane.
func RunSmoke(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
) (string, error) {
	if ctx == nil {
		return "", errors.New("benchmark smoke context is required")
	}
	if !filepath.IsAbs(stateDirectory) {
		return "", errors.New("benchmark state directory must be absolute")
	}
	if pathsOverlap(stateDirectory, outputDirectory) {
		return "", errors.New(
			"benchmark state and output directories must be separate",
		)
	}
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	if err := prepareOutputDirectory(outputDirectory); err != nil {
		return "", err
	}
	sink, err := newObservationSink(outputDirectory)
	if err != nil {
		return "", err
	}
	sinkClosed := false
	defer func() {
		if !sinkClosed {
			sink.abort()
		}
	}()

	storage, err := sharedcache.Open(ctx, filepath.Clean(stateDirectory))
	if err != nil {
		return "", err
	}
	storageClosed := false
	defer func() {
		if !storageClosed {
			_ = storage.Close()
		}
	}()

	startedAt := time.Now().UTC()
	signingKey := benchmarkSigningKey(manifestDigest)
	objects := makeSmokeObjects(loaded.Seed)
	endpoints := make(map[int]*readEndpoint, len(loaded.Clients))
	defer func() {
		for _, endpoint := range endpoints {
			endpoint.server.Close()
		}
	}()

	var summaries []*stratumSummary
	for _, currentPhase := range loaded.Phases {
		for clientIndex, clients := range loaded.Clients {
			namespace := int64(100 + clientIndex)
			operations := makeSmokeOperations(
				objects,
				currentPhase,
				loaded.Seed+int64(clientIndex),
			)
			var summary *stratumSummary
			if currentPhase.ID == "COLD" {
				var endpoint *readEndpoint
				summary, endpoint, err = runSmokeColdStratum(
					ctx,
					storage,
					signingKey,
					manifestDigest,
					loaded.Seed,
					namespace,
					clients,
					operations,
					sink,
				)
				if err == nil {
					endpoints[clients] = endpoint
				}
			} else {
				endpoint := endpoints[clients]
				if endpoint == nil {
					return "", errors.New(
						"benchmark warm phase lacks committed cold data",
					)
				}
				summary, err = runSmokeReadStratum(
					ctx,
					endpoint,
					currentPhase.ID,
					clients,
					operations,
					sink,
				)
			}
			if err != nil {
				return "", err
			}
			summaries = append(summaries, summary)
		}
	}

	rawIdentity, err := sink.close()
	if err != nil {
		return "", err
	}
	sinkClosed = true
	for clients, endpoint := range endpoints {
		endpoint.server.Close()
		delete(endpoints, clients)
	}
	if err := storage.Close(); err != nil {
		return "", err
	}
	storageClosed = true
	distributions, p50, p95, p99, rates, errorsByClass, bytesByStratum :=
		summarizeStrata(summaries)
	faults := make([]faultOutcome, 0, len(loaded.Faults))
	for _, fault := range loaded.Faults {
		faults = append(faults, faultOutcome{
			ID:             fault.ID,
			ExpectedSafety: fault.ExpectedSafety,
			Status:         "NOT_RUN_SMOKE",
		})
	}
	result := Result{
		SchemaVersion:            resultSchemaVersion,
		Qualification:            "SMOKE",
		BenchmarkDigest:          manifestDigest,
		Seed:                     loaded.Seed,
		StartedAt:                startedAt.Format(time.RFC3339Nano),
		CompletedAt:              time.Now().UTC().Format(time.RFC3339Nano),
		Hardware:                 currentHardware(),
		Cgroup:                   currentCgroup(),
		Components:               loaded.Components,
		ActualObjectDistribution: distributions,
		P50:                      p50,
		P95:                      p95,
		P99:                      p99,
		Throughput:               rates,
		Errors:                   errorsByClass,
		Bytes:                    bytesByStratum,
		Recovery:                 []recoveryObservation{},
		ReadinessTransitions:     []readinessTransition{},
		FaultOutcomes:            faults,
		Deviations: []string{
			"SMOKE_PROFILE_SCALED_OBJECT_SIZES",
			"SMOKE_PROFILE_ONE_CYCLE_PER_STRATUM",
			"SMOKE_PROFILE_PHASE_DURATIONS_NOT_QUALIFYING",
			"FAULT_MATRIX_NOT_RUN",
			"GRADLE_FIXTURES_NOT_RUN",
			"RUNNER_QUALIFICATION_NOT_CLAIMED",
		},
		RawObservations: rawIdentity,
	}
	resultPath, err := writeResult(outputDirectory, result)
	if err != nil {
		return "", err
	}
	if err := ValidateResult(manifestPath, resultPath); err != nil {
		return "", fmt.Errorf("validate benchmark smoke result: %w", err)
	}
	return resultPath, nil
}

func runSmokeColdStratum(
	ctx context.Context,
	storage *sharedcache.Storage,
	signingKey ed25519.PrivateKey,
	manifestDigest string,
	seed int64,
	namespace int64,
	clients int,
	operations []workloadOperation,
	sink *observationSink,
) (*stratumSummary, *readEndpoint, error) {
	request := benchmarkAttemptRequest(
		manifestDigest,
		namespace,
		clients,
		time.Now().UTC(),
	)
	_, _, err := storage.StartAttempt(ctx, request)
	if err != nil {
		return nil, nil, err
	}
	handler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: namespace,
			PendingAttemptID:    request.AttemptID,
			AllowRead:           false,
			AllowWrite:          true,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	server := httptest.NewServer(handler)
	client := benchmarkHTTPClient(clients)
	objects := make([]sharedcache.CommitObject, len(operations))
	summary, err := runOperations(
		ctx,
		"COLD",
		clients,
		operations,
		sink,
		func(
			operation workloadOperation,
		) rawObservation {
			observation, digest := executePUT(
				ctx,
				client,
				server.URL,
				operation,
				seed,
			)
			if observation.ErrorClass == "" {
				objects[operation.object.index] = sharedcache.CommitObject{
					NamespaceGeneration: namespace,
					Key:                 operation.object.key,
					Checksum:            digest,
					SizeBytes:           operation.object.size,
				}
			}
			return observation
		},
	)
	client.CloseIdleConnections()
	server.Close()
	if err != nil {
		return nil, nil, err
	}
	for _, object := range objects {
		if object.Checksum == "" {
			return nil, nil, errors.New(
				"benchmark cold phase did not publish every object",
			)
		}
	}
	slices.SortFunc(objects, func(first, second sharedcache.CommitObject) int {
		return strings.Compare(first.Key, second.Key)
	})
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	verified, err := verifyBenchmarkDecision(
		ctx,
		signingKey,
		request,
		status,
		objects,
		now,
	)
	if err != nil {
		return nil, nil, err
	}
	if _, err := storage.CommitAttempt(
		ctx,
		status.StateVersion,
		1,
		verified,
	); err != nil {
		return nil, nil, err
	}
	readHandler, err := sharedcache.NewHTTPHandler(
		storage,
		sharedcache.HTTPBinding{
			Tenant:              request.Repository.Tenant,
			NamespaceGeneration: namespace,
			AllowRead:           true,
			AllowWrite:          false,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	return summary, &readEndpoint{
		server: httptest.NewServer(readHandler),
	}, nil
}

func runSmokeReadStratum(
	ctx context.Context,
	endpoint *readEndpoint,
	phaseID string,
	clients int,
	operations []workloadOperation,
	sink *observationSink,
) (*stratumSummary, error) {
	client := benchmarkHTTPClient(clients)
	defer client.CloseIdleConnections()
	return runOperations(
		ctx,
		phaseID,
		clients,
		operations,
		sink,
		func(operation workloadOperation) rawObservation {
			return executeGET(
				ctx,
				client,
				endpoint.server.URL,
				operation,
			)
		},
	)
}

func runOperations(
	ctx context.Context,
	phaseID string,
	clients int,
	operations []workloadOperation,
	sink *observationSink,
	execute func(workloadOperation) rawObservation,
) (*stratumSummary, error) {
	summary := newStratumSummary(phaseID, clients)
	summary.startedAt = time.Now()
	jobs := make(chan workloadOperation)
	results := make(chan rawObservation, len(operations))
	var workers sync.WaitGroup
	for worker := 0; worker < clients; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for operation := range jobs {
				select {
				case results <- execute(operation):
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, operation := range operations {
			select {
			case jobs <- operation:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	for observation := range results {
		observation.Phase = phaseID
		observation.Clients = clients
		if err := sink.append(observation); err != nil {
			return nil, err
		}
		summary.observe(observation)
	}
	summary.completedAt = time.Now()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(summary.errors) != 0 {
		return nil, fmt.Errorf(
			"benchmark %s/%d observed errors: %v",
			phaseID,
			clients,
			summary.errors,
		)
	}
	if len(summary.latencies) != len(operations) {
		return nil, errors.New("benchmark stratum lost observations")
	}
	return summary, nil
}

func executePUT(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	operation workloadOperation,
	seed int64,
) (rawObservation, string) {
	started := time.Now()
	reader := newDeterministicReader(
		seed,
		operation.object.index,
		operation.object.size,
	)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		endpoint+"/cache/"+operation.object.key,
		reader,
	)
	if err != nil {
		return failedObservation(operation, "PUT", started, "REQUEST"), ""
	}
	request.ContentLength = operation.object.size
	response, err := client.Do(request)
	if err != nil {
		return failedObservation(operation, "PUT", started, "TRANSPORT"), ""
	}
	responseBytes, readErr := io.Copy(io.Discard, response.Body)
	closeErr := response.Body.Close()
	observation := rawObservation{
		Operation:     "PUT",
		ObjectIndex:   operation.object.index,
		SizeBytes:     operation.object.size,
		Status:        response.StatusCode,
		DurationNs:    time.Since(started).Nanoseconds(),
		RequestBytes:  operation.object.size,
		ResponseBytes: responseBytes,
	}
	switch {
	case readErr != nil || closeErr != nil:
		observation.ErrorClass = "BODY_READ"
	case response.StatusCode != http.StatusCreated:
		observation.ErrorClass = "STATUS_" + strconv.Itoa(response.StatusCode)
	case responseBytes != 0:
		observation.ErrorClass = "BODY_LENGTH"
	}
	return observation, response.Header.Get("X-BuildOpt-Blob-Digest")
}

func executeGET(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	operation workloadOperation,
) rawObservation {
	started := time.Now()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint+"/cache/"+operation.key,
		nil,
	)
	if err != nil {
		return failedObservation(operation, "GET", started, "REQUEST")
	}
	response, err := client.Do(request)
	if err != nil {
		return failedObservation(operation, "GET", started, "TRANSPORT")
	}
	responseBytes, readErr := io.Copy(io.Discard, response.Body)
	closeErr := response.Body.Close()
	expectedStatus := http.StatusNotFound
	expectedBytes := int64(0)
	if operation.expectedHit {
		expectedStatus = http.StatusOK
		expectedBytes = operation.object.size
	}
	observation := rawObservation{
		Operation:     "GET",
		ObjectIndex:   operation.object.index,
		SizeBytes:     operation.object.size,
		ExpectedHit:   operation.expectedHit,
		Status:        response.StatusCode,
		DurationNs:    time.Since(started).Nanoseconds(),
		ResponseBytes: responseBytes,
	}
	switch {
	case readErr != nil || closeErr != nil:
		observation.ErrorClass = "BODY_READ"
	case response.StatusCode != expectedStatus:
		observation.ErrorClass = "STATUS_" + strconv.Itoa(response.StatusCode)
	case responseBytes != expectedBytes:
		observation.ErrorClass = "BODY_LENGTH"
	}
	return observation
}

func failedObservation(
	operation workloadOperation,
	method string,
	started time.Time,
	class string,
) rawObservation {
	return rawObservation{
		Operation:   method,
		ObjectIndex: operation.object.index,
		SizeBytes:   operation.object.size,
		ExpectedHit: operation.expectedHit,
		DurationNs:  time.Since(started).Nanoseconds(),
		ErrorClass:  class,
	}
}

func benchmarkHTTPClient(clients int) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
			MaxIdleConns:          clients,
			MaxIdleConnsPerHost:   clients,
			MaxConnsPerHost:       clients,
			IdleConnTimeout:       30 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
		CheckRedirect: func(
			_ *http.Request,
			_ []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}
}

func makeSmokeObjects(seed int64) []workloadObject {
	objects := make([]workloadObject, 0, 100)
	index := 0
	for _, mix := range smokeObjectMix {
		for count := 0; count < mix.Percent; count++ {
			objects = append(objects, workloadObject{
				index: index,
				size:  mix.SizeBytes,
				key:   fmt.Sprintf("object-%05d", index),
			})
			index++
		}
	}
	shuffleObjects(objects, uint64(seed))
	return objects
}

func makeSmokeOperations(
	objects []workloadObject,
	currentPhase phase,
	seed int64,
) []workloadOperation {
	operations := make([]workloadOperation, 0, len(objects))
	hits := make([]bool, len(objects))
	hitCount := len(objects) * currentPhase.TargetHitPercent / 100
	for index := 0; index < hitCount; index++ {
		hits[index] = true
	}
	shuffleBools(hits, uint64(seed)^hashString(currentPhase.ID))
	for position, object := range objects {
		key := object.key
		if !hits[position] && currentPhase.ID != "COLD" {
			key = fmt.Sprintf("miss-%05d", object.index)
		}
		operations = append(operations, workloadOperation{
			object:      object,
			expectedHit: hits[position] && currentPhase.ID != "COLD",
			key:         key,
		})
	}
	return operations
}

func benchmarkAttemptRequest(
	manifestDigest string,
	namespace int64,
	clients int,
	now time.Time,
) sharedcache.StartAttemptRequest {
	identity := digestText(
		manifestDigest + "\x00" + strconv.FormatInt(namespace, 10),
	)
	attemptID := fmt.Sprintf("beta-smoke-n%d-c%d", namespace, clients)
	return sharedcache.StartAttemptRequest{
		RequestID: "start-" + attemptID,
		AttemptID: attemptID,
		Repository: sharedcache.RepositoryIdentity{
			Tenant:      "beta-smoke",
			Repository:  "buildopt",
			TrustDomain: "local-benchmark",
		},
		NamespaceGeneration:       namespace,
		SourceRevision:            strings.TrimPrefix(identity, "sha256:")[:40],
		SourceStateDigest:         identity,
		PolicyDigest:              digestText("policy-" + attemptID),
		ConfigurationPolicyDigest: digestText("configuration-" + attemptID),
		CacheContractDigest:       digestText("cache-" + attemptID),
		OwnerID:                   "owner-" + attemptID,
		LeaseID:                   "lease-" + attemptID,
		LeaseExpiresAt:            now.Add(2 * time.Hour),
	}
}

func newDeterministicReader(
	seed int64,
	index int,
	size int64,
) *deterministicReader {
	digest := sha256.Sum256([]byte(
		strconv.FormatInt(seed, 10) + "\x00" +
			strconv.Itoa(index) + "\x00" +
			strconv.FormatInt(size, 10),
	))
	state := uint64(0)
	for index := 0; index < 8; index++ {
		state = state<<8 | uint64(digest[index])
	}
	if state == 0 {
		state = 1
	}
	return &deterministicReader{
		state:     state,
		remaining: size,
		used:      len((deterministicReader{}).buffer),
	}
}

func (reader *deterministicReader) Read(destination []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(destination)) > reader.remaining {
		destination = destination[:reader.remaining]
	}
	for index := range destination {
		if reader.used == len(reader.buffer) {
			value := reader.next()
			for byteIndex := range reader.buffer {
				reader.buffer[byteIndex] = byte(value >> (8 * byteIndex))
			}
			reader.used = 0
		}
		destination[index] = reader.buffer[reader.used]
		reader.used++
	}
	reader.remaining -= int64(len(destination))
	return len(destination), nil
}

func (reader *deterministicReader) next() uint64 {
	value := reader.state
	value ^= value >> 12
	value ^= value << 25
	value ^= value >> 27
	reader.state = value
	return value * 2685821657736338717
}

func shuffleObjects(values []workloadObject, seed uint64) {
	for index := len(values) - 1; index > 0; index-- {
		seed = nextSeed(seed)
		swap := int(seed % uint64(index+1))
		values[index], values[swap] = values[swap], values[index]
	}
}

func shuffleBools(values []bool, seed uint64) {
	for index := len(values) - 1; index > 0; index-- {
		seed = nextSeed(seed)
		swap := int(seed % uint64(index+1))
		values[index], values[swap] = values[swap], values[index]
	}
}

func nextSeed(seed uint64) uint64 {
	seed ^= seed >> 12
	seed ^= seed << 25
	seed ^= seed >> 27
	return seed * 2685821657736338717
}

func hashString(value string) uint64 {
	digest := sha256.Sum256([]byte(value))
	result := uint64(0)
	for index := 0; index < 8; index++ {
		result = result<<8 | uint64(digest[index])
	}
	return result
}

func digestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func pathsOverlap(first string, second string) bool {
	if !filepath.IsAbs(first) || !filepath.IsAbs(second) {
		return true
	}
	first = filepath.Clean(first)
	second = filepath.Clean(second)
	return pathContains(first, second) || pathContains(second, first)
}

func pathContains(parent string, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil &&
		(relative == "." ||
			(relative != ".." &&
				!strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}
