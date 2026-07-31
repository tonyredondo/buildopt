package betabenchmark

import (
	"context"
	"errors"
	"fmt"
	"io"
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

const sustainedGatewayInFlightBytes int64 = 200 << 20
const sustainedAuthorityLifetime = 2 * time.Hour

type sustainedFixture struct {
	storage    *sharedcache.Storage
	authority  systemAuthorityFixture
	upstream   *httptest.Server
	invocation *managedGatewayInvocation
}

// RunSustained executes the exact one-hour SUSTAINED phase. It refuses to
// produce qualifying evidence outside the pinned 4-CPU/16-GiB cgroup.
func RunSustained(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
	buildoptExecutable string,
) (string, error) {
	loaded, _, _, err := loadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	return runSustained(
		ctx,
		manifestPath,
		stateDirectory,
		outputDirectory,
		buildoptExecutable,
		productionSustainedOptions(loaded),
	)
}

// RunSustainedTrial exercises the same managed-gateway scheduler with a short,
// scaled and explicitly non-qualifying workload for CI.
func RunSustainedTrial(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
	buildoptExecutable string,
) (string, error) {
	return runSustained(
		ctx,
		manifestPath,
		stateDirectory,
		outputDirectory,
		buildoptExecutable,
		trialSustainedOptions(),
	)
}

func runSustained(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
	buildoptExecutable string,
	options sustainedRunOptions,
) (string, error) {
	if ctx == nil {
		return "", errors.New("benchmark sustained context is required")
	}
	if !filepath.IsAbs(stateDirectory) {
		return "", errors.New(
			"benchmark sustained state directory must be absolute",
		)
	}
	if pathsOverlap(stateDirectory, outputDirectory) {
		return "", errors.New(
			"benchmark sustained state and output directories must be separate",
		)
	}
	if err := validateSystemFaultExecutable(buildoptExecutable); err != nil {
		return "", fmt.Errorf(
			"benchmark sustained buildopt executable: %w",
			err,
		)
	}
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	if options.requireRunner && !goldenRunnerCgroup(currentCgroup()) {
		return "", errors.New(
			"benchmark sustained qualification requires the golden runner cgroup",
		)
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
	objects := makeSustainedObjects(options.objectMix, loaded.Seed)
	fixture, err := prepareSustainedFixture(
		ctx,
		filepath.Join(stateDirectory, "data-plane"),
		buildoptExecutable,
		manifestDigest,
		loaded.Seed,
		objects,
	)
	if err != nil {
		return "", err
	}
	defer fixture.close()

	phase := phase{
		ID:               "SUSTAINED",
		TargetHitPercent: 70,
	}
	baseOperations := makeSmokeOperations(
		objects,
		phase,
		loaded.Seed+500,
	)
	operations := repeatSustainedOperations(
		baseOperations,
		options.cycleCount,
	)
	startedAt := time.Now().UTC()
	segmentDuration := options.totalDuration /
		time.Duration(len(loaded.Clients))
	summaries := make([]*stratumSummary, 0, len(loaded.Clients))
	for _, clients := range loaded.Clients {
		summary, err := runPacedSustainedStratum(
			ctx,
			fixture.invocation.connection,
			clients,
			operations,
			segmentDuration,
			sink,
		)
		if err != nil {
			return "", err
		}
		summaries = append(summaries, summary)
	}
	rawIdentity, err := sink.close()
	if err != nil {
		return "", err
	}
	sinkClosed = true
	if rawIdentity.Count != options.expectedRawRows ||
		rawIdentity.SizeBytes > maximumSustainedRawBytes {
		return "", errors.New(
			"benchmark sustained raw evidence exceeded its bound",
		)
	}
	distributions, p50, p95, p99, rates, errorsByClass, bytesByStratum :=
		summarizeStrata(summaries)
	observations, err := readSustainedObservations(
		outputDirectory,
		rawIdentity,
	)
	if err != nil {
		return "", err
	}
	result := sustainedResult{
		SchemaVersion:            sustainedResultSchemaVersion,
		Qualification:            options.qualification,
		BenchmarkDigest:          manifestDigest,
		Seed:                     loaded.Seed,
		StartedAt:                startedAt.Format(time.RFC3339Nano),
		CompletedAt:              time.Now().UTC().Format(time.RFC3339Nano),
		Hardware:                 currentHardware(),
		Cgroup:                   currentCgroup(),
		Components:               loaded.Components,
		RunnerVerified:           options.requireRunner,
		Transport:                "REAL_MANAGED_GATEWAY_LOOPBACK_HTTP",
		CycleOperations:          len(operations),
		ActualObjectDistribution: distributions,
		P50:                      p50,
		P95:                      p95,
		P99:                      p99,
		Throughput:               rates,
		Errors:                   errorsByClass,
		Bytes:                    bytesByStratum,
		LatencyTargets: summarizeSustainedLatencyTargets(
			observations,
			loaded.Clients,
		),
		Deviations:      sustainedDeviations(options),
		RawObservations: rawIdentity,
	}
	resultPath, err := writeSustainedResult(outputDirectory, result)
	if err != nil {
		return "", err
	}
	if err := validateSustainedResult(
		manifestPath,
		resultPath,
		options,
	); err != nil {
		return "", fmt.Errorf("validate benchmark sustained result: %w", err)
	}
	return resultPath, nil
}

func prepareSustainedFixture(
	ctx context.Context,
	root string,
	buildoptExecutable string,
	manifestDigest string,
	seed int64,
	objects []workloadObject,
) (*sustainedFixture, error) {
	storage, err := sharedcache.Open(ctx, filepath.Join(root, "shared"))
	if err != nil {
		return nil, err
	}
	closeStorage := true
	defer func() {
		if closeStorage {
			_ = storage.Close()
		}
	}()
	if err := publishSustainedObjects(
		ctx,
		storage,
		manifestDigest,
		seed,
		objects,
	); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	authority, err := newSystemAuthorityFixtureWithDurations(
		ctx,
		filepath.Join(root, "authority"),
		manifestDigest,
		now,
		940,
		false,
		false,
		sustainedAuthorityLifetime,
		sustainedAuthorityLifetime,
		sustainedAuthorityLifetime,
	)
	if err != nil {
		return nil, err
	}
	binding, _, err := storage.InstallLocalAuthority(
		ctx,
		authority.verified,
		authority.credential,
		now,
	)
	if err != nil {
		return nil, err
	}
	handler, err := sharedcache.NewLocalAuthorityHTTPHandler(
		storage,
		binding,
		authority.credential,
	)
	if err != nil {
		return nil, err
	}
	upstream := httptest.NewServer(handler)
	fixture := &sustainedFixture{
		storage:   storage,
		authority: authority,
		upstream:  upstream,
	}
	invocation, err := startManagedGatewayInvocation(
		ctx,
		buildoptExecutable,
		systemGatewayEnvironment(
			&systemGatewayFixture{
				authority: authority,
				upstream:  upstream,
			},
			filepath.Join(root, "managed"),
			"sustained-load",
		),
	)
	if err != nil {
		upstream.Close()
		return nil, err
	}
	fixture.invocation = invocation
	closeStorage = false
	return fixture, nil
}

func (fixture *sustainedFixture) close() {
	if fixture == nil {
		return
	}
	if fixture.invocation != nil {
		_ = fixture.invocation.stop()
	}
	if fixture.upstream != nil {
		fixture.upstream.Close()
	}
	if fixture.storage != nil {
		_ = fixture.storage.Close()
	}
}

func publishSustainedObjects(
	ctx context.Context,
	storage *sharedcache.Storage,
	manifestDigest string,
	seed int64,
	objects []workloadObject,
) error {
	request := benchmarkAttemptRequest(
		manifestDigest,
		12,
		940,
		time.Now().UTC(),
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		return err
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
		return err
	}
	server := httptest.NewServer(handler)
	client := benchmarkHTTPClient(1)
	defer client.CloseIdleConnections()
	defer server.Close()
	commitObjects := make([]sharedcache.CommitObject, 0, len(objects))
	for _, object := range objects {
		observation, digest := executePUT(
			ctx,
			client,
			server.URL,
			workloadOperation{object: object},
			seed,
		)
		if observation.ErrorClass != "" ||
			observation.Status != http.StatusCreated ||
			digest == "" {
			return fmt.Errorf(
				"publish benchmark sustained object: %+v",
				observation,
			)
		}
		commitObjects = append(commitObjects, sharedcache.CommitObject{
			NamespaceGeneration: request.NamespaceGeneration,
			Key:                 object.key,
			Checksum:            digest,
			SizeBytes:           object.size,
		})
	}
	slices.SortFunc(
		commitObjects,
		func(first sharedcache.CommitObject, second sharedcache.CommitObject) int {
			return strings.Compare(first.Key, second.Key)
		},
	)
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil {
		return err
	}
	verified, err := verifyBenchmarkDecision(
		ctx,
		benchmarkSigningKey(manifestDigest),
		request,
		status,
		commitObjects,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	_, err = storage.CommitAttempt(
		ctx,
		status.StateVersion,
		1,
		verified,
	)
	return err
}

func makeSustainedObjects(mix []objectMix, seed int64) []workloadObject {
	objects := make([]workloadObject, 0, 100)
	index := 0
	for _, item := range mix {
		for count := 0; count < item.Percent; count++ {
			objects = append(objects, workloadObject{
				index: index,
				size:  item.SizeBytes,
				key:   fmt.Sprintf("sustained-%05d", index),
			})
			index++
		}
	}
	shuffleObjects(objects, uint64(seed)^hashString("SUSTAINED"))
	return objects
}

func repeatSustainedOperations(
	base []workloadOperation,
	cycles int,
) []workloadOperation {
	operations := make([]workloadOperation, 0, len(base)*cycles)
	for cycle := 0; cycle < cycles; cycle++ {
		for position, operation := range base {
			repeated := operation
			repeated.object.index = cycle*len(base) + position
			operations = append(operations, repeated)
		}
	}
	return operations
}

func runPacedSustainedStratum(
	ctx context.Context,
	connection managedGatewayObservation,
	clients int,
	operations []workloadOperation,
	duration time.Duration,
	sink *observationSink,
) (*stratumSummary, error) {
	summary := newStratumSummary("SUSTAINED", clients)
	summary.startedAt = time.Now()
	client := benchmarkHTTPClient(clients)
	defer client.CloseIdleConnections()
	for offset := 0; offset < len(operations); {
		end := sustainedBatchEnd(operations, offset, clients)
		batch := operations[offset:end]
		results := make([]rawObservation, len(batch))
		var workers sync.WaitGroup
		for index, operation := range batch {
			workers.Add(1)
			go func(index int, operation workloadOperation) {
				defer workers.Done()
				results[index] = executeSustainedGET(
					ctx,
					client,
					connection,
					operation,
				)
			}(index, operation)
		}
		workers.Wait()
		for _, observation := range results {
			observation.Phase = "SUSTAINED"
			observation.Clients = clients
			if err := sink.append(observation); err != nil {
				return nil, err
			}
			summary.observe(observation)
		}
		offset = end
		if err := waitForSustainedPace(
			ctx,
			summary.startedAt,
			duration,
			offset,
			len(operations),
		); err != nil {
			return nil, err
		}
	}
	summary.completedAt = time.Now()
	if len(summary.errors) != 0 {
		return nil, fmt.Errorf(
			"benchmark SUSTAINED/%d observed errors: %v",
			clients,
			summary.errors,
		)
	}
	if len(summary.latencies) != len(operations) {
		return nil, errors.New("benchmark sustained stratum lost observations")
	}
	return summary, nil
}

func sustainedBatchEnd(
	operations []workloadOperation,
	offset int,
	clients int,
) int {
	end := offset
	bytes := int64(0)
	for end < len(operations) && end-offset < clients {
		operationBytes := int64(0)
		if operations[end].expectedHit {
			operationBytes = operations[end].object.size
		}
		if end > offset &&
			bytes+operationBytes > sustainedGatewayInFlightBytes {
			break
		}
		bytes += operationBytes
		end++
	}
	return end
}

func executeSustainedGET(
	ctx context.Context,
	client *http.Client,
	connection managedGatewayObservation,
	operation workloadOperation,
) rawObservation {
	started := time.Now()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		connection.endpoint+"/cache/"+operation.key,
		nil,
	)
	if err != nil {
		return failedObservation(operation, "GET", started, "REQUEST")
	}
	request.SetBasicAuth(connection.username, connection.password)
	response, err := client.Do(request)
	if err != nil {
		return failedObservation(operation, "GET", started, "TRANSPORT")
	}
	ready := time.Since(started)
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
		ReadyNs:       ready.Nanoseconds(),
		DurationNs:    time.Since(started).Nanoseconds(),
		ResponseBytes: responseBytes,
	}
	switch {
	case readErr != nil || closeErr != nil:
		observation.ErrorClass = "BODY_READ"
	case response.StatusCode != expectedStatus:
		observation.ErrorClass = "STATUS_" +
			strconv.Itoa(response.StatusCode)
	case responseBytes != expectedBytes:
		observation.ErrorClass = "BODY_LENGTH"
	}
	return observation
}

func waitForSustainedPace(
	ctx context.Context,
	startedAt time.Time,
	duration time.Duration,
	completed int,
	total int,
) error {
	target := startedAt.Add(
		time.Duration(
			(int64(duration) * int64(completed)) / int64(total),
		),
	)
	wait := time.Until(target)
	if wait <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
