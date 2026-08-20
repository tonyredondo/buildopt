package betabenchmark

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	systemFaultObjectBytes     int64 = 100
	systemFaultLatencyDelay          = 250 * time.Millisecond
	systemFaultRequestDeadline       = 100 * time.Millisecond
	systemFaultOrphanFiles           = 256
)

type systemGatewayFixture struct {
	storage        *sharedcache.Storage
	authority      systemAuthorityFixture
	controller     *systemUpstreamController
	upstream       *httptest.Server
	expectedDigest string
}

type systemUpstreamController struct {
	handler http.Handler
	mode    atomic.Int32
	dropped atomic.Int64
}

type managedGatewayObservation struct {
	endpoint   string
	username   string
	password   string
	generation string
}

type managedGatewayInvocation struct {
	command    *exec.Cmd
	stdin      io.WriteCloser
	connection managedGatewayObservation
	stderr     bytes.Buffer
}

type systemServerProcess struct {
	command  *exec.Cmd
	endpoint string
	lines    chan string
	stderr   bytes.Buffer
}

// RunSystemFaults executes the final six restart, network, and revocation rows
// from the pinned benchmark against real BuildOpt processes and loopback data.
func RunSystemFaults(
	ctx context.Context,
	manifestPath string,
	stateDirectory string,
	outputDirectory string,
	executables SystemFaultExecutables,
) (string, error) {
	if ctx == nil {
		return "", errors.New("benchmark system-fault context is required")
	}
	if !filepath.IsAbs(stateDirectory) {
		return "", errors.New(
			"benchmark system-fault state directory must be absolute",
		)
	}
	if pathsOverlap(stateDirectory, outputDirectory) {
		return "", errors.New(
			"benchmark system-fault state and output directories must be separate",
		)
	}
	for name, path := range map[string]string{
		"buildopt":        executables.BuildOpt,
		"buildopt-server": executables.Server,
	} {
		if err := validateSystemFaultExecutable(path); err != nil {
			return "", fmt.Errorf(
				"benchmark system-fault %s executable: %w",
				name,
				err,
			)
		}
	}
	loaded, _, manifestDigest, err := loadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	if err := prepareOutputDirectory(outputDirectory); err != nil {
		return "", err
	}
	sink, err := newSystemFaultSink(outputDirectory)
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
	gatewayFixture, err := prepareSystemGatewayFixture(
		ctx,
		filepath.Join(stateDirectory, "gateway-data-plane"),
		manifestDigest,
		loaded.Seed,
	)
	if err != nil {
		return "", err
	}
	defer gatewayFixture.close()

	outcomes := make([]faultOutcome, 0, 6)
	triggers := make([]systemFaultTrigger, 0, 6)
	recovery := make([]recoveryObservation, 0, 6)
	appendResult := func(
		outcome faultOutcome,
		trigger systemFaultTrigger,
		recovered recoveryObservation,
		runErr error,
	) error {
		if runErr != nil {
			return runErr
		}
		outcomes = append(outcomes, outcome)
		triggers = append(triggers, trigger)
		recovery = append(recovery, recovered)
		return nil
	}

	if err := appendResult(runGatewayRestartFault(
		ctx,
		loaded,
		executables.BuildOpt,
		filepath.Join(stateDirectory, "managed-gateway"),
		gatewayFixture,
		sink,
	)); err != nil {
		return "", err
	}
	if err := appendResult(runServerRestartFault(
		ctx,
		loaded,
		executables.Server,
		filepath.Join(stateDirectory, "server-restart"),
		sink,
	)); err != nil {
		return "", err
	}
	if err := appendResult(runNetworkLatencyFault(
		ctx,
		loaded,
		executables.BuildOpt,
		filepath.Join(stateDirectory, "network-latency"),
		gatewayFixture,
		sink,
	)); err != nil {
		return "", err
	}
	if err := appendResult(runNetworkLossFault(
		ctx,
		loaded,
		executables.BuildOpt,
		filepath.Join(stateDirectory, "network-loss"),
		gatewayFixture,
		sink,
	)); err != nil {
		return "", err
	}
	if err := appendResult(runRevocationFault(
		ctx,
		loaded,
		manifestDigest,
		filepath.Join(stateDirectory, "revoked-policy"),
		sink,
		false,
	)); err != nil {
		return "", err
	}
	if err := appendResult(runRevocationFault(
		ctx,
		loaded,
		manifestDigest,
		filepath.Join(stateDirectory, "revoked-grant"),
		sink,
		true,
	)); err != nil {
		return "", err
	}

	rawIdentity, err := sink.close()
	if err != nil {
		return "", err
	}
	sinkClosed = true
	result := systemFaultResult{
		SchemaVersion:   systemFaultResultSchemaVersion,
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
	resultPath, err := writeSystemFaultResult(outputDirectory, result)
	if err != nil {
		return "", err
	}
	if err := ValidateSystemFaultResult(manifestPath, resultPath); err != nil {
		return "", fmt.Errorf("validate benchmark system-fault result: %w", err)
	}
	return resultPath, nil
}

func prepareSystemGatewayFixture(
	ctx context.Context,
	root string,
	manifestDigest string,
	seed int64,
) (*systemGatewayFixture, error) {
	storage, err := openSharedFaultStorage(
		ctx,
		filepath.Join(root, "shared"),
		systemFaultObjectBytes,
	)
	if err != nil {
		return nil, err
	}
	closeStorage := true
	defer func() {
		if closeStorage {
			_ = storage.Close()
		}
	}()
	staged, err := stageSharedFaultObject(
		ctx,
		storage,
		benchmarkSigningKey(manifestDigest),
		manifestDigest,
		seed,
		12,
		1,
		"system-hit",
		systemFaultObjectBytes,
	)
	if err != nil {
		return nil, err
	}
	if _, err := storage.CommitAttempt(
		ctx,
		staged.status.StateVersion,
		1,
		staged.verified,
	); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	authority, err := newSystemAuthorityFixture(
		ctx,
		filepath.Join(root, "authority"),
		manifestDigest,
		now,
		900,
		false,
		false,
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
	controller := &systemUpstreamController{handler: handler}
	upstream := httptest.NewServer(controller)
	closeStorage = false
	return &systemGatewayFixture{
		storage:        storage,
		authority:      authority,
		controller:     controller,
		upstream:       upstream,
		expectedDigest: staged.object.Checksum,
	}, nil
}

func (fixture *systemGatewayFixture) close() {
	if fixture == nil {
		return
	}
	if fixture.upstream != nil {
		fixture.upstream.Close()
	}
	if fixture.storage != nil {
		_ = fixture.storage.Close()
	}
}

func (controller *systemUpstreamController) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	switch controller.mode.Load() {
	case 1:
		timer := time.NewTimer(systemFaultLatencyDelay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-request.Context().Done():
			return
		}
	case 2:
		hijacker, ok := response.(http.Hijacker)
		if !ok {
			http.Error(
				response,
				"loopback fault transport cannot drop connections",
				http.StatusInternalServerError,
			)
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(
				response,
				"loopback fault transport failed to drop connection",
				http.StatusInternalServerError,
			)
			return
		}
		controller.dropped.Add(1)
		_ = connection.Close()
		return
	}
	controller.handler.ServeHTTP(response, request)
}

func runGatewayRestartFault(
	ctx context.Context,
	loaded manifest,
	buildoptExecutable string,
	stateDirectory string,
	fixture *systemGatewayFixture,
	sink *systemFaultSink,
) (
	faultOutcome,
	systemFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "GATEWAY_RESTART"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	environment := systemGatewayEnvironment(
		fixture,
		stateDirectory,
		"gateway-restart",
	)
	first, err := startManagedGatewayInvocation(
		ctx,
		buildoptExecutable,
		environment,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	firstStopped := false
	defer func() {
		if !firstStopped {
			_ = first.stop()
		}
	}()
	status, size, digestMatch, err := requestManagedGatewayObject(
		ctx,
		first.connection,
		fixture.expectedDigest,
	)
	if err != nil || status != http.StatusOK ||
		size != systemFaultObjectBytes || !digestMatch {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"gateway pre-restart hit: status=%d size=%d digest=%t err=%v",
				status,
				size,
				digestMatch,
				err,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "GATEWAY_COMPLETE_HIT_BEFORE_RESTART",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(status)},
			{Name: "responseBytes", Value: size},
			{Name: "digestMatch", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStarted := time.Now()
	if err := first.stop(); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	firstStopped = true
	if err := waitManagedGatewayUnavailable(
		first.connection.endpoint,
		3*time.Second,
	); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	second, err := startManagedGatewayInvocation(
		ctx,
		buildoptExecutable,
		environment,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	defer second.stop()
	endpointReused := second.connection.endpoint == first.connection.endpoint
	credentialReused := second.connection.password == first.connection.password
	generationReused := second.connection.generation == first.connection.generation
	if !endpointReused || !credentialReused || !generationReused {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			errors.New("managed gateway restart rotated stable identity")
	}
	triggerSequence, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "GATEWAY_PROCESS_RESTARTED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "endpointReused", Value: 1},
			{Name: "credentialReused", Value: 1},
			{Name: "generationReused", Value: 1},
		},
	})
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	status, size, digestMatch, err = requestManagedGatewayObject(
		ctx,
		second.connection,
		fixture.expectedDigest,
	)
	if err != nil || status != http.StatusOK ||
		size != systemFaultObjectBytes || !digestMatch {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"gateway post-restart hit: status=%d size=%d digest=%t err=%v",
				status,
				size,
				digestMatch,
				err,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "GATEWAY_COMPLETE_HIT_AFTER_RESTART",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(status)},
			{Name: "responseBytes", Value: size},
			{Name: "digestMatch", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	return passedSystemFault(
		faultID,
		expectedSafety,
		triggerSequence,
		time.Since(recoveryStarted).Nanoseconds(),
		"STABLE_IDENTITY_COMPLETE_HIT",
	)
}

func runServerRestartFault(
	ctx context.Context,
	loaded manifest,
	serverExecutable string,
	stateDirectory string,
	sink *systemFaultSink,
) (
	faultOutcome,
	systemFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "SERVER_RESTART"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	first, err := startSystemServer(ctx, serverExecutable, stateDirectory)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	firstStopped := false
	defer func() {
		if !firstStopped {
			_ = first.stop()
		}
	}()
	ready, err := waitHTTPStatus(
		ctx,
		first.endpoint+"/readyz",
		http.StatusOK,
		3*time.Second,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	if err := first.stop(); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	firstStopped = true
	recoveryStarted := time.Now()
	triggerSequence, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "SERVER_PROCESS_STOPPED",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "previousReadinessStatus", Value: int64(ready)},
			{Name: "exitCode", Value: 0},
		},
	})
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	if err := createSystemOrphanFiles(
		stateDirectory,
		systemFaultOrphanFiles,
	); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	restarted, err := startSystemServer(ctx, serverExecutable, stateDirectory)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	defer restarted.stop()
	liveness, _, err := requestHTTPStatus(
		ctx,
		restarted.endpoint+"/livez",
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	readiness, _, err := requestHTTPStatus(
		ctx,
		restarted.endpoint+"/readyz",
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	product, _, err := requestHTTPStatus(
		ctx,
		restarted.endpoint+"/cache/not-ready",
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	if liveness != http.StatusOK ||
		readiness != http.StatusServiceUnavailable ||
		product != http.StatusServiceUnavailable {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"restart readiness boundary: live=%d ready=%d product=%d",
				liveness,
				readiness,
				product,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "RESTART_LIVE_NOT_READY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "livenessStatus", Value: int64(liveness)},
			{Name: "readinessStatus", Value: int64(readiness)},
			{Name: "productStatus", Value: int64(product)},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	if err := restarted.waitForLine(
		"initialized and reconciled",
		// The race-enabled repository suite can saturate this host while the
		// restarted server scans and reconciles its persisted fault state. Keep
		// the readiness assertion intact, but give that real work a bounded
		// scheduling budget instead of treating CPU contention as corruption.
		30*time.Second,
	); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	ready, err = waitHTTPStatus(
		ctx,
		restarted.endpoint+"/readyz",
		http.StatusOK,
		3*time.Second,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	remaining, err := countRegularFiles(
		filepath.Join(stateDirectory, "blobs"),
	)
	if err != nil || remaining != 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"server restart orphan cleanup: remaining=%d err=%v",
				remaining,
				err,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "RECONCILED_AND_READY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "readinessStatus", Value: int64(ready)},
			{Name: "orphanFilesRemaining", Value: int64(remaining)},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	return passedSystemFault(
		faultID,
		expectedSafety,
		triggerSequence,
		time.Since(recoveryStarted).Nanoseconds(),
		"READY_AFTER_RECONCILIATION",
	)
}

func runNetworkLatencyFault(
	ctx context.Context,
	loaded manifest,
	buildoptExecutable string,
	stateDirectory string,
	fixture *systemGatewayFixture,
	sink *systemFaultSink,
) (
	faultOutcome,
	systemFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "NETWORK_LATENCY"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	invocation, err := startManagedGatewayInvocation(
		ctx,
		buildoptExecutable,
		systemGatewayEnvironment(fixture, stateDirectory, "network-latency"),
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	defer invocation.stop()
	fixture.controller.mode.Store(1)
	defer fixture.controller.mode.Store(0)
	recoveryStarted := time.Now()
	triggerSequence, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "UPSTREAM_LATENCY_INJECTED",
		DurationNs: 0,
		Metrics: []sharedFaultMetric{
			{Name: "delayMilliseconds", Value: systemFaultLatencyDelay.Milliseconds()},
			{
				Name:  "deadlineMilliseconds",
				Value: systemFaultRequestDeadline.Milliseconds(),
			},
		},
	})
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	requestContext, cancel := context.WithTimeout(
		ctx,
		systemFaultRequestDeadline,
	)
	_, size, _, requestErr := requestManagedGatewayObject(
		requestContext,
		invocation.connection,
		fixture.expectedDigest,
	)
	cancel()
	if !errors.Is(requestErr, context.DeadlineExceeded) || size != 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"network latency deadline: size=%d err=%v",
				size,
				requestErr,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "DEADLINE_EXCEEDED_AND_RECORDED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "deadlineExceeded", Value: 1},
			{Name: "responseBytes", Value: 0},
			{Name: "errorRecorded", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	fixture.controller.mode.Store(0)
	status, size, digestMatch, err := requestManagedGatewayObject(
		ctx,
		invocation.connection,
		fixture.expectedDigest,
	)
	if err != nil || status != http.StatusOK ||
		size != systemFaultObjectBytes || !digestMatch {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"network latency recovery: status=%d size=%d digest=%t err=%v",
				status,
				size,
				digestMatch,
				err,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "COMPLETE_HIT_AFTER_LATENCY_RECOVERY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(status)},
			{Name: "responseBytes", Value: size},
			{Name: "digestMatch", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	return passedSystemFault(
		faultID,
		expectedSafety,
		triggerSequence,
		time.Since(recoveryStarted).Nanoseconds(),
		"BOUNDED_DEADLINE_THEN_COMPLETE_HIT",
	)
}

func runNetworkLossFault(
	ctx context.Context,
	loaded manifest,
	buildoptExecutable string,
	stateDirectory string,
	fixture *systemGatewayFixture,
	sink *systemFaultSink,
) (
	faultOutcome,
	systemFaultTrigger,
	recoveryObservation,
	error,
) {
	const faultID = "NETWORK_LOSS"
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	invocation, err := startManagedGatewayInvocation(
		ctx,
		buildoptExecutable,
		systemGatewayEnvironment(fixture, stateDirectory, "network-loss"),
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	defer invocation.stop()
	fixture.controller.dropped.Store(0)
	fixture.controller.mode.Store(2)
	defer fixture.controller.mode.Store(0)
	recoveryStarted := time.Now()
	status, size, _, err := requestManagedGatewayObject(
		ctx,
		invocation.connection,
		fixture.expectedDigest,
	)
	drops := fixture.controller.dropped.Load()
	if err != nil ||
		status != http.StatusNotFound ||
		size != 0 ||
		drops != 1 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"network loss fail-open: status=%d size=%d drops=%d err=%v",
				status,
				size,
				drops,
				err,
			)
	}
	triggerSequence, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "UPSTREAM_CONNECTION_DROPPED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "connectionDropObserved", Value: drops},
		},
	})
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "BYTE_FREE_FAIL_OPEN_MISS",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(status)},
			{Name: "responseBytes", Value: size},
			{Name: "failOpen", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	fixture.controller.mode.Store(0)
	status, size, digestMatch, err := requestManagedGatewayObject(
		ctx,
		invocation.connection,
		fixture.expectedDigest,
	)
	if err != nil || status != http.StatusOK ||
		size != systemFaultObjectBytes || !digestMatch {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"network loss recovery: status=%d size=%d digest=%t err=%v",
				status,
				size,
				digestMatch,
				err,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "COMPLETE_HIT_AFTER_NETWORK_RECOVERY",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(status)},
			{Name: "responseBytes", Value: size},
			{Name: "digestMatch", Value: 1},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	return passedSystemFault(
		faultID,
		expectedSafety,
		triggerSequence,
		time.Since(recoveryStarted).Nanoseconds(),
		"FAIL_OPEN_THEN_COMPLETE_HIT",
	)
}

func runRevocationFault(
	ctx context.Context,
	loaded manifest,
	manifestDigest string,
	stateDirectory string,
	sink *systemFaultSink,
	revokeGrant bool,
) (
	faultOutcome,
	systemFaultTrigger,
	recoveryObservation,
	error,
) {
	faultID := "REVOKED_POLICY"
	ordinal := 910
	if revokeGrant {
		faultID = "REVOKED_GRANT"
		ordinal = 920
	}
	expectedSafety, err := expectedFaultSafety(loaded, faultID)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	storage, err := openSharedFaultStorage(
		ctx,
		filepath.Join(stateDirectory, "shared"),
		systemFaultObjectBytes,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	defer storage.Close()
	now := time.Now().UTC()
	current, err := newSystemAuthorityFixture(
		ctx,
		filepath.Join(stateDirectory, "authority"),
		manifestDigest,
		now,
		ordinal,
		true,
		revokeGrant,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	binding, _, err := storage.InstallLocalAuthority(
		ctx,
		current.verified,
		current.credential,
		now,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	handler, err := sharedcache.NewLocalAuthorityHTTPHandler(
		storage,
		binding,
		current.credential,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	payload := bytes.Repeat([]byte{byte(ordinal % 251)}, int(systemFaultObjectBytes))
	putStatus, _, digest, err := requestAuthorityObject(
		ctx,
		server.URL,
		http.MethodPut,
		"revoked-pending",
		payload,
		current.credential,
		binding.AuthorityDigest,
	)
	if err != nil || putStatus != http.StatusCreated || digest == "" {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"stage revoked pending object: status=%d digest=%q err=%v",
				putStatus,
				digest,
				err,
			)
	}
	status, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil || status.PendingObjectCount != 1 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("revocation pending status: %+v/%v", status, err)
	}
	request := sharedcache.StartAttemptRequest{
		RequestID:                 current.document.Revocation.RequestID,
		AttemptID:                 status.AttemptID,
		AuthorityDigest:           status.AuthorityDigest,
		Repository:                status.Repository,
		NamespaceGeneration:       status.NamespaceGeneration,
		SourceRevision:            status.SourceRevision,
		SourceStateDigest:         status.SourceStateDigest,
		PolicyDigest:              status.PolicyDigest,
		ConfigurationPolicyDigest: status.ConfigurationPolicyDigest,
		CacheContractDigest:       status.CacheContractDigest,
		OwnerID:                   status.OwnerID,
		LeaseID:                   status.LeaseID,
		LeaseExpiresAt:            status.LeaseExpiresAt,
	}
	grant := sharedcache.TestOptimizationGrant{
		State:  "NOT_REQUIRED",
		Reason: "NO_TEST_OUTPUTS",
	}
	if revokeGrant {
		grantEpoch := int64(7)
		grant = sharedcache.TestOptimizationGrant{
			State:      "PRESENT",
			GrantID:    "beta-system-grant",
			GrantEpoch: &grantEpoch,
			Digest: current.document.Policy.TestOptimizationGrant.
				Digest,
			ExpiresAt: current.document.Policy.TestOptimizationGrant.
				ExpiresAt,
		}
	}
	verifiedDecision, err := verifyBenchmarkDecisionWithAuthority(
		ctx,
		current.privateKey,
		request,
		status,
		[]sharedcache.CommitObject{{
			NamespaceGeneration: status.NamespaceGeneration,
			Key:                 "revoked-pending",
			Checksum:            digest,
			SizeBytes:           systemFaultObjectBytes,
		}},
		current.document.Revocation.RevocationEpoch,
		grant,
		now,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	advanced, err := advanceSystemAuthorityFixture(
		ctx,
		current,
		now.Add(time.Second),
		revokeGrant,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	nextBinding, changed, err := storage.InstallLocalAuthority(
		ctx,
		advanced.verified,
		advanced.credential,
		now.Add(time.Second),
	)
	if err != nil || !changed {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("install revocation: changed=%t err=%v", changed, err)
	}
	recoveryStarted := time.Now()
	event := "POLICY_REVOCATION_INSTALLED"
	metrics := []sharedFaultMetric{
		{
			Name:  "revocationEpochBefore",
			Value: current.document.Revocation.RevocationEpoch,
		},
		{
			Name:  "revocationEpochAfter",
			Value: advanced.document.Revocation.RevocationEpoch,
		},
		{
			Name:  "namespaceGenerationBefore",
			Value: current.document.Policy.RemoteCache.NamespaceGeneration,
		},
		{
			Name:  "namespaceGenerationAfter",
			Value: advanced.document.Policy.RemoteCache.NamespaceGeneration,
		},
	}
	if revokeGrant {
		event = "GRANT_REVOCATION_INSTALLED"
		metrics = []sharedFaultMetric{
			{Name: "grantDigestChanged", Value: 1},
			{
				Name:  "revocationEpochBefore",
				Value: current.document.Revocation.RevocationEpoch,
			},
			{
				Name:  "revocationEpochAfter",
				Value: advanced.document.Revocation.RevocationEpoch,
			},
		}
	}
	triggerSequence, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      event,
		DurationNs: 0,
		Metrics:    metrics,
	})
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	staleStatus, staleBytes, _, err := requestAuthorityObject(
		ctx,
		server.URL,
		http.MethodGet,
		"revoked-pending",
		nil,
		current.credential,
		binding.AuthorityDigest,
	)
	if err != nil ||
		staleStatus != http.StatusUnauthorized ||
		staleBytes != 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"stale revocation route: status=%d bytes=%d err=%v",
				staleStatus,
				staleBytes,
				err,
			)
	}
	staleEvent := "STALE_ROUTE_REJECTED"
	if revokeGrant {
		staleEvent = "STALE_GRANT_ROUTE_REJECTED"
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      staleEvent,
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "httpStatus", Value: int64(staleStatus)},
			{Name: "responseBytes", Value: staleBytes},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	if _, err := storage.CommitAttempt(
		ctx,
		status.StateVersion,
		advanced.document.Revocation.RevocationEpoch,
		verifiedDecision,
	); !errors.Is(err, sharedcache.ErrCommitRejected) {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("revoked commit was not rejected: %v", err)
	}
	aborted, err := storage.AttemptStatus(ctx, binding.AttemptID)
	if err != nil ||
		aborted.State != sharedcache.AttemptAborted ||
		aborted.AbortReason != "POLICY_CHANGED" ||
		aborted.PendingObjectCount != 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("revoked attempt status: %+v/%v", aborted, err)
	}
	capacity, err := storage.CapacitySnapshot(ctx)
	if err != nil || capacity.PendingBytes != 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf("revoked pending capacity: %+v/%v", capacity, err)
	}
	currentHandler, err := sharedcache.NewLocalAuthorityHTTPHandler(
		storage,
		nextBinding,
		advanced.credential,
	)
	if err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	currentServer := httptest.NewServer(currentHandler)
	currentStatus, currentBytes, _, err := requestAuthorityObject(
		ctx,
		currentServer.URL,
		http.MethodGet,
		"revoked-pending",
		nil,
		advanced.credential,
		nextBinding.AuthorityDigest,
	)
	currentServer.Close()
	if err != nil ||
		currentStatus != http.StatusNotFound ||
		currentBytes != 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			fmt.Errorf(
				"current revoked route: status=%d bytes=%d err=%v",
				currentStatus,
				currentBytes,
				err,
			)
	}
	if _, err := sink.append(systemFaultObservation{
		FaultID:    faultID,
		Event:      "PENDING_ABORTED_AND_GENERATION_ROTATED",
		DurationNs: time.Since(recoveryStarted).Nanoseconds(),
		Metrics: []sharedFaultMetric{
			{Name: "attemptStateAborted", Value: 1},
			{Name: "pendingBytes", Value: capacity.PendingBytes},
			{
				Name: "l1GenerationDelta",
				Value: advanced.document.Revocation.L1SecurityGeneration -
					current.document.Revocation.L1SecurityGeneration,
			},
		},
	}); err != nil {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{}, err
	}
	recoveryStatus := "POLICY_ABORTED_AND_ROTATED"
	if revokeGrant {
		recoveryStatus = "GRANT_ABORTED_AND_ROTATED"
	}
	return passedSystemFault(
		faultID,
		expectedSafety,
		triggerSequence,
		time.Since(recoveryStarted).Nanoseconds(),
		recoveryStatus,
	)
}

func validateSystemFaultExecutable(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("path must be absolute and clean")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return errors.New("path is not an executable regular file")
	}
	return nil
}

func systemGatewayEnvironment(
	fixture *systemGatewayFixture,
	stateDirectory string,
	slot string,
) []string {
	overrides := map[string]string{
		"BUILDOPT_LOCAL_AUTHORITY_PATH":  fixture.authority.authorityPath,
		"BUILDOPT_LOCAL_TRUST_ROOT_PATH": fixture.authority.trustRootPath,
		"BUILDOPT_LOCAL_CACHE_CREDENTIAL_PATH": fixture.authority.
			credentialPath,
		"BUILDOPT_SHARED_CACHE_URL":       fixture.upstream.URL,
		"BUILDOPT_L1_STATE_ROOT":          filepath.Join(stateDirectory, "l1"),
		"BUILDOPT_L1_TENANT_ID":           "beta-smoke",
		"BUILDOPT_L1_REPOSITORY_ID":       "buildopt",
		"BUILDOPT_L1_TRUST_DOMAIN":        "local-benchmark",
		"BUILDOPT_L1_COMPATIBILITY_CLASS": "gradle-9.6.1-jdk-21-linux-amd64",
		"BUILDOPT_GATEWAY_STATE_ROOT":     filepath.Join(stateDirectory, "gateway"),
		"BUILDOPT_RUNNER_SLOT":            slot,
		"BUILDOPT_GATEWAY_IDLE_TIMEOUT":   "100ms",
		"CI":                              "false",
	}
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found && (strings.HasPrefix(key, "BUILDOPT_") || key == "CI") {
			continue
		}
		environment = append(environment, entry)
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}

func startManagedGatewayInvocation(
	ctx context.Context,
	buildoptExecutable string,
	environment []string,
) (*managedGatewayInvocation, error) {
	const observer = `printf '%s\n' \
"$BUILDOPT_GATEWAY_URL" \
"$BUILDOPT_GATEWAY_USERNAME" \
"$BUILDOPT_GATEWAY_PASSWORD" \
"$BUILDOPT_GATEWAY_CONNECTION_GENERATION"
IFS= read -r buildopt_release || true`
	command := exec.CommandContext(
		ctx,
		buildoptExecutable,
		"run",
		"--",
		"/bin/sh",
		"-c",
		observer,
	)
	command.Env = environment
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	invocation := &managedGatewayInvocation{
		command: command,
		stdin:   stdin,
	}
	command.Stderr = &invocation.stderr
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	type observed struct {
		connection managedGatewayObservation
		err        error
	}
	observedConnection := make(chan observed, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		values := make([]string, 0, 4)
		for len(values) < 4 && scanner.Scan() {
			values = append(values, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			observedConnection <- observed{err: err}
			return
		}
		if len(values) != 4 {
			observedConnection <- observed{
				err: errors.New("managed gateway observer was incomplete"),
			}
			return
		}
		observedConnection <- observed{
			connection: managedGatewayObservation{
				endpoint:   values[0],
				username:   values[1],
				password:   values[2],
				generation: values[3],
			},
		}
	}()
	timer := time.NewTimer(7 * time.Second)
	defer timer.Stop()
	select {
	case observed := <-observedConnection:
		if observed.err != nil ||
			validateManagedGatewayObservation(observed.connection) != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			return nil, fmt.Errorf(
				"observe managed gateway: %v; stderr=%q",
				errors.Join(
					observed.err,
					validateManagedGatewayObservation(observed.connection),
				),
				invocation.stderr.String(),
			)
		}
		invocation.connection = observed.connection
		return invocation, nil
	case <-timer.C:
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, fmt.Errorf(
			"managed gateway observer timed out; stderr=%q",
			invocation.stderr.String(),
		)
	case <-ctx.Done():
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, ctx.Err()
	}
}

func validateManagedGatewayObservation(
	observation managedGatewayObservation,
) error {
	parsed, err := url.Parse(observation.endpoint)
	if err != nil ||
		parsed.Scheme != "http" ||
		parsed.Hostname() != "127.0.0.1" ||
		parsed.Path != "" ||
		parsed.RawQuery != "" ||
		observation.username != "buildopt" ||
		observation.password == "" ||
		observation.generation == "" {
		return errors.New("managed gateway observer returned invalid identity")
	}
	return nil
}

func (invocation *managedGatewayInvocation) stop() error {
	if invocation == nil || invocation.command == nil {
		return nil
	}
	_ = invocation.stdin.Close()
	wait := make(chan error, 1)
	go func() {
		wait <- invocation.command.Wait()
	}()
	select {
	case err := <-wait:
		invocation.command = nil
		if err != nil {
			return fmt.Errorf(
				"managed gateway invocation exited: %v; stderr=%q",
				err,
				invocation.stderr.String(),
			)
		}
		return nil
	case <-time.After(7 * time.Second):
		_ = invocation.command.Process.Kill()
		err := <-wait
		invocation.command = nil
		return fmt.Errorf(
			"managed gateway invocation did not stop: %v; stderr=%q",
			err,
			invocation.stderr.String(),
		)
	}
}

func requestManagedGatewayObject(
	ctx context.Context,
	connection managedGatewayObservation,
	expectedDigest string,
) (int, int64, bool, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		connection.endpoint+"/cache/system-hit",
		nil,
	)
	if err != nil {
		return 0, 0, false, err
	}
	request.SetBasicAuth(connection.username, connection.password)
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, 0, false, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return response.StatusCode, 0, false, err
	}
	digest := sha256.Sum256(content)
	return response.StatusCode,
		int64(len(content)),
		"sha256:"+fmt.Sprintf("%x", digest[:]) == expectedDigest,
		nil
}

func waitManagedGatewayUnavailable(
	endpoint string,
	timeout time.Duration,
) error {
	address := strings.TrimPrefix(endpoint, "http://")
	deadline := time.Now().Add(timeout)
	for {
		connection, err := net.DialTimeout("tcp4", address, 50*time.Millisecond)
		if err != nil {
			return nil
		}
		_ = connection.Close()
		if time.Now().After(deadline) {
			return errors.New("managed gateway remained reachable after restart")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func startSystemServer(
	ctx context.Context,
	serverExecutable string,
	stateDirectory string,
) (*systemServerProcess, error) {
	command := exec.CommandContext(
		ctx,
		serverExecutable,
		"serve",
		"--listen",
		"127.0.0.1:0",
		"--state-dir",
		stateDirectory,
	)
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found && key == sessioningest.ServerTokenEnvironment {
			continue
		}
		environment = append(environment, entry)
	}
	command.Env = append(
		environment,
		sessioningest.ServerTokenEnvironment+"="+
			base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)),
	)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	process := &systemServerProcess{
		command: command,
		lines:   make(chan string, 32),
	}
	command.Stderr = &process.stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			process.lines <- scanner.Text()
		}
		close(process.lines)
	}()
	line, err := process.waitForLineValue("listening on ", 5*time.Second)
	if err != nil {
		_ = process.command.Process.Kill()
		_ = process.command.Wait()
		return nil, fmt.Errorf(
			"start system server: %v; stderr=%q",
			err,
			process.stderr.String(),
		)
	}
	index := strings.Index(line, "http://")
	if index < 0 {
		_ = process.command.Process.Kill()
		_ = process.command.Wait()
		return nil, errors.New("system server did not report endpoint")
	}
	process.endpoint = strings.TrimSpace(line[index:])
	return process, nil
}

func (process *systemServerProcess) waitForLine(
	contains string,
	timeout time.Duration,
) error {
	_, err := process.waitForLineValue(contains, timeout)
	return err
}

func (process *systemServerProcess) waitForLineValue(
	contains string,
	timeout time.Duration,
) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-process.lines:
			if !ok {
				return "", errors.New("system server output closed")
			}
			if strings.Contains(line, contains) {
				return line, nil
			}
		case <-timer.C:
			return "", fmt.Errorf(
				"system server did not report %q",
				contains,
			)
		}
	}
}

func (process *systemServerProcess) stop() error {
	if process == nil || process.command == nil {
		return nil
	}
	if err := process.command.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	wait := make(chan error, 1)
	go func() {
		wait <- process.command.Wait()
	}()
	select {
	case err := <-wait:
		process.command = nil
		if err != nil {
			return fmt.Errorf(
				"system server exited: %v; stderr=%q",
				err,
				process.stderr.String(),
			)
		}
		return nil
	case <-time.After(7 * time.Second):
		_ = process.command.Process.Kill()
		err := <-wait
		process.command = nil
		return fmt.Errorf(
			"system server did not stop: %v; stderr=%q",
			err,
			process.stderr.String(),
		)
	}
}

func createSystemOrphanFiles(root string, count int) error {
	blobs := filepath.Join(root, "blobs", "sha256")
	for index := 1; index <= count; index++ {
		digest := fmt.Sprintf("%064x", index)
		directory := filepath.Join(blobs, digest[:2])
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		path := filepath.Join(directory, digest[2:])
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func waitHTTPStatus(
	ctx context.Context,
	endpoint string,
	expected int,
	timeout time.Duration,
) (int, error) {
	deadline := time.Now().Add(timeout)
	var lastStatus int
	var lastErr error
	for {
		lastStatus, _, lastErr = requestHTTPStatus(ctx, endpoint)
		if lastErr == nil && lastStatus == expected {
			return lastStatus, nil
		}
		if time.Now().After(deadline) {
			return lastStatus, fmt.Errorf(
				"wait for HTTP %d at %s: status=%d err=%v",
				expected,
				endpoint,
				lastStatus,
				lastErr,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func requestHTTPStatus(
	ctx context.Context,
	endpoint string,
) (int, int64, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return 0, 0, err
	}
	response, err := (&http.Client{Timeout: time.Second}).Do(request)
	if err != nil {
		return 0, 0, err
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return response.StatusCode, int64(len(content)), err
}

func requestAuthorityObject(
	ctx context.Context,
	endpoint string,
	method string,
	key string,
	content []byte,
	credential []byte,
	authorityDigest string,
) (int, int64, string, error) {
	var body io.Reader
	if content != nil {
		body = bytes.NewReader(content)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		endpoint+"/cache/"+key,
		body,
	)
	if err != nil {
		return 0, 0, "", err
	}
	if content != nil {
		request.ContentLength = int64(len(content))
	}
	request.Header.Set(
		"Authorization",
		"Bearer "+base64.RawURLEncoding.EncodeToString(credential),
	)
	request.Header.Set(
		sharedcache.AuthorityDigestHeader,
		authorityDigest,
	)
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return 0, 0, "", err
	}
	defer response.Body.Close()
	responseContent, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return response.StatusCode,
		int64(len(responseContent)),
		response.Header.Get("X-BuildOpt-Blob-Digest"),
		err
}

func passedSystemFault(
	id string,
	expectedSafety string,
	triggerSequence int64,
	recoveryDuration int64,
	recoveryStatus string,
) (
	faultOutcome,
	systemFaultTrigger,
	recoveryObservation,
	error,
) {
	if triggerSequence < 1 || recoveryDuration <= 0 {
		return faultOutcome{}, systemFaultTrigger{}, recoveryObservation{},
			errors.New("system fault did not record trigger/recovery timing")
	}
	return faultOutcome{
			ID:             id,
			ExpectedSafety: expectedSafety,
			Status:         "PASSED",
		}, systemFaultTrigger{
			ID:       id,
			Sequence: triggerSequence,
		}, recoveryObservation{
			ID:         id,
			DurationNs: recoveryDuration,
			Status:     recoveryStatus,
		}, nil
}
