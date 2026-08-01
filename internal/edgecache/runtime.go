package edgecache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	RuntimeStatusSchemaVersion = "buildopt.edge-cache/runtime-status/v1"
	AuthorityReloadInterval    = time.Second
	ReplicationInterval        = time.Second
	MaintenanceInterval        = 30 * time.Second
	StatusInterval             = time.Second
	StatusFreshness            = 5 * time.Second

	maximumRuntimeStatusBytes = 64 << 10
	maximumAuthorityBytes     = 4 << 20
	maximumTrustRootBytes     = 64 << 10
	maximumCredentialBytes    = 128
)

// RuntimeStatus is the private, path-local operational projection written by
// the Edge process. It contains aggregate state only and never credentials,
// object keys, repository identity, or filesystem paths.
type RuntimeStatus struct {
	SchemaVersion      string            `json:"schemaVersion"`
	EdgeID             string            `json:"edgeId"`
	State              string            `json:"state"`
	Listen             string            `json:"listen"`
	ObservedAt         string            `json:"observedAt"`
	AuthorityExpiresAt string            `json:"authorityExpiresAt,omitempty"`
	WriteEnabled       bool              `json:"writeEnabled"`
	AuthorityError     string            `json:"authorityError,omitempty"`
	Capacity           CapacitySnapshot  `json:"capacity"`
	Pending            PendingSnapshot   `json:"pending"`
	Maintenance        MaintenanceReport `json:"maintenance"`
}

type runtimeGeneration struct {
	handler     http.Handler
	client      *SharedClient
	write       *WriteAuthority
	expiresAt   time.Time
	fingerprint [sha256.Size]byte
}

// Runtime composes the durable store, signed authority, loopback proxy,
// asynchronous replication, byte maintenance, and private status lifecycle.
type Runtime struct {
	config     Config
	store      *Store
	stateStore *localauthority.FileStateStore
	clock      func() time.Time

	generationMutex sync.RWMutex
	generation      *runtimeGeneration

	statusMutex    sync.RWMutex
	state          string
	authorityError string
	maintenance    MaintenanceReport
	closed         bool
}

// OpenRuntime performs every mutable and authenticated preflight before a
// caller opens the configured listener.
func OpenRuntime(ctx context.Context, config Config) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("open Edge runtime: nil context")
	}
	store, err := OpenStore(config)
	if err != nil {
		return nil, err
	}
	stateStore, err := localauthority.NewFileStateStore(
		filepath.Join(config.Storage.StateDirectory, "authority-state"),
	)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("open Edge authority state: %w", err)
	}
	runtime := &Runtime{
		config:     config,
		store:      store,
		stateStore: stateStore,
		clock:      time.Now,
		state:      "STARTING",
	}
	generation, err := runtime.loadGeneration(ctx, runtime.clock().UTC())
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("open Edge authority: %w", err)
	}
	runtime.generation = generation
	if err := runtime.writeStatus(ctx, runtime.clock().UTC()); err != nil {
		generation.client.Close()
		_ = store.Close()
		return nil, fmt.Errorf("write initial Edge status: %w", err)
	}
	return runtime, nil
}

// Serve owns the loopback HTTP lifecycle until cancellation or a local fatal
// worker failure. Invalid/expired authority disables the route without
// terminating the process so a corrected signed snapshot can recover it.
func (runtime *Runtime) Serve(ctx context.Context, listener net.Listener) error {
	if runtime == nil || ctx == nil || listener == nil {
		return errors.New("serve Edge runtime: invalid input")
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.IP == nil || !address.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		return errors.New("serve Edge runtime: listener is not IPv4 loopback")
	}
	_, configuredPortText, err := net.SplitHostPort(runtime.config.Server.Listen)
	configuredPort, portErr := strconv.Atoi(configuredPortText)
	if err != nil || portErr != nil ||
		(configuredPort != 0 && address.Port != configuredPort) {
		return errors.New("serve Edge runtime: listener does not match configuration")
	}
	runtime.setState("READY", "")
	if err := runtime.writeStatus(ctx, runtime.clock().UTC()); err != nil {
		return fmt.Errorf("publish Edge readiness: %w", err)
	}
	workerContext, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	failures := make(chan error, 4)
	var workers sync.WaitGroup
	startWorker := func(worker func(context.Context) error) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := worker(workerContext); err != nil &&
				!errors.Is(err, context.Canceled) {
				select {
				case failures <- err:
				case <-workerContext.Done():
				}
			}
		}()
	}
	startWorker(runtime.watchAuthority)
	startWorker(runtime.runReplicator)
	startWorker(runtime.runMaintenance)
	startWorker(runtime.runStatusWriter)

	server := &http.Server{
		Handler:           runtime,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	serveDone := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveDone <- err
	}()

	var result error
	serveCompleted := false
	select {
	case <-ctx.Done():
	case result = <-serveDone:
		serveCompleted = true
	case result = <-failures:
	}
	runtime.setState("STOPPING", "")
	_ = runtime.writeStatus(context.Background(), runtime.clock().UTC())
	cancelWorkers()
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := server.Shutdown(shutdownContext)
	shutdownCancel()
	if shutdownErr != nil {
		_ = server.Close()
	}
	if !serveCompleted {
		result = <-serveDone
	}
	workers.Wait()
	runtime.clearGeneration()
	runtime.setState("STOPPED", "")
	statusErr := runtime.writeStatus(context.Background(), runtime.clock().UTC())
	return errors.Join(result, shutdownErr, statusErr)
}

// ServeHTTP preserves the exact cache-only proxy surface. A missing current
// generation is a byte-free 503; it never falls back to stale authority.
func (runtime *Runtime) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	runtime.generationMutex.RLock()
	defer runtime.generationMutex.RUnlock()
	if runtime.generation == nil {
		response.Header().Set("Cache-Control", "no-store")
		writeEdgeStatus(response, http.StatusServiceUnavailable)
		return
	}
	runtime.generation.handler.ServeHTTP(response, request)
}

// Close clears the active credential and releases the durable writer lease.
func (runtime *Runtime) Close() error {
	if runtime == nil {
		return nil
	}
	runtime.statusMutex.Lock()
	if runtime.closed {
		runtime.statusMutex.Unlock()
		return nil
	}
	runtime.closed = true
	runtime.statusMutex.Unlock()
	runtime.clearGeneration()
	runtime.setState("STOPPED", "")
	_ = runtime.writeStatus(context.Background(), runtime.clock().UTC())
	return runtime.store.Close()
}

func (runtime *Runtime) loadGeneration(ctx context.Context, now time.Time) (*runtimeGeneration, error) {
	before, err := runtime.authorityFingerprint()
	if err != nil {
		return nil, err
	}
	verified, _, credential, err := localauthority.LoadFiles(
		ctx,
		runtime.config.Authority.SnapshotPath,
		runtime.config.Authority.TrustRootPath,
		runtime.config.Shared.CredentialPath,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer clearBytes(credential)
	if _, _, _, err := runtime.stateStore.Install(verified, now); err != nil {
		return nil, fmt.Errorf("install Edge anti-rollback state: %w", err)
	}
	readAuthority, err := NewReadAuthority(verified, now)
	if err != nil {
		return nil, err
	}
	var writeAuthority *WriteAuthority
	if verified.Document().Attempt.AllowWrite {
		write, writeErr := NewWriteAuthority(verified, now)
		if writeErr != nil {
			return nil, writeErr
		}
		writeAuthority = &write
	}
	client, err := NewSharedClient(
		runtime.config.Shared,
		credential,
		&http.Client{Timeout: 15 * time.Second},
	)
	if err != nil {
		return nil, err
	}
	after, err := runtime.authorityFingerprint()
	if err != nil || before != after {
		client.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Edge authority files changed during verification")
	}
	proxy, err := NewProxy(runtime.store, client, readAuthority, writeAuthority, runtime.clock)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &runtimeGeneration{
		handler:     proxy.Handler(),
		client:      client,
		write:       writeAuthority,
		expiresAt:   verified.ExpiresAt(),
		fingerprint: after,
	}, nil
}

func (runtime *Runtime) watchAuthority(ctx context.Context) error {
	ticker := time.NewTicker(AuthorityReloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			fingerprint, err := runtime.authorityFingerprint()
			runtime.generationMutex.RLock()
			active := runtime.generation
			unchanged := err == nil && active != nil &&
				fingerprint == active.fingerprint && active.expiresAt.After(now.UTC())
			runtime.generationMutex.RUnlock()
			if unchanged {
				continue
			}
			runtime.deactivate("AUTHORITY_UNAVAILABLE")
			next, loadErr := runtime.loadGeneration(ctx, now.UTC())
			if loadErr != nil {
				continue
			}
			runtime.activate(next)
		}
	}
}

func (runtime *Runtime) runReplicator(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-timer.C:
			runtime.generationMutex.RLock()
			generation := runtime.generation
			if generation != nil && generation.write != nil {
				_, err := runtime.store.ReplicatePendingOnce(
					ctx, *generation.write, generation.client, now.UTC(),
				)
				if err != nil && !errors.Is(err, context.Canceled) {
					runtime.generationMutex.RUnlock()
					return err
				}
			}
			runtime.generationMutex.RUnlock()
			timer.Reset(ReplicationInterval)
		}
	}
}

func (runtime *Runtime) runMaintenance(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-timer.C:
			report, err := runtime.store.Maintain(ctx, now.UTC())
			if err != nil {
				return err
			}
			runtime.statusMutex.Lock()
			runtime.maintenance = report
			runtime.statusMutex.Unlock()
			timer.Reset(MaintenanceInterval)
		}
	}
}

func (runtime *Runtime) runStatusWriter(ctx context.Context) error {
	ticker := time.NewTicker(StatusInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if err := runtime.writeStatus(ctx, now.UTC()); err != nil {
				return err
			}
		}
	}
}

func (runtime *Runtime) activate(next *runtimeGeneration) {
	runtime.generationMutex.Lock()
	previous := runtime.generation
	runtime.generation = next
	runtime.generationMutex.Unlock()
	if previous != nil {
		previous.client.Close()
	}
	runtime.setState("READY", "")
}

func (runtime *Runtime) deactivate(class string) {
	runtime.generationMutex.Lock()
	previous := runtime.generation
	runtime.generation = nil
	runtime.generationMutex.Unlock()
	if previous != nil {
		previous.client.Close()
	}
	runtime.setState("NOT_READY", class)
}

func (runtime *Runtime) clearGeneration() {
	runtime.generationMutex.Lock()
	previous := runtime.generation
	runtime.generation = nil
	runtime.generationMutex.Unlock()
	if previous != nil {
		previous.client.Close()
	}
}

func (runtime *Runtime) setState(state, authorityError string) {
	runtime.statusMutex.Lock()
	runtime.state = state
	runtime.authorityError = authorityError
	runtime.statusMutex.Unlock()
}

func (runtime *Runtime) status(ctx context.Context, now time.Time) (RuntimeStatus, error) {
	capacity, err := runtime.store.CapacitySnapshot(ctx)
	if err != nil {
		return RuntimeStatus{}, err
	}
	pending, err := runtime.store.PendingSnapshot(ctx)
	if err != nil {
		return RuntimeStatus{}, err
	}
	runtime.statusMutex.RLock()
	state := runtime.state
	authorityError := runtime.authorityError
	maintenance := runtime.maintenance
	runtime.statusMutex.RUnlock()
	runtime.generationMutex.RLock()
	generation := runtime.generation
	status := RuntimeStatus{
		SchemaVersion:  RuntimeStatusSchemaVersion,
		EdgeID:         runtime.config.EdgeID,
		State:          state,
		Listen:         runtime.config.Server.Listen,
		ObservedAt:     now.UTC().Format(time.RFC3339Nano),
		AuthorityError: authorityError,
		Capacity:       capacity,
		Pending:        pending,
		Maintenance:    maintenance,
	}
	if generation != nil {
		status.AuthorityExpiresAt = generation.expiresAt.UTC().Format(time.RFC3339Nano)
		status.WriteEnabled = generation.write != nil
	}
	runtime.generationMutex.RUnlock()
	return status, nil
}

func (runtime *Runtime) writeStatus(ctx context.Context, now time.Time) error {
	status, err := runtime.status(ctx, now)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		return err
	}
	path := RuntimeStatusPath(runtime.config)
	temporary, err := os.CreateTemp(filepath.Dir(path), ".edge-status-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(encoded); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func (runtime *Runtime) authorityFingerprint() ([sha256.Size]byte, error) {
	hash := sha256.New()
	for _, source := range []struct {
		path    string
		maximum int64
	}{
		{runtime.config.Authority.SnapshotPath, maximumAuthorityBytes},
		{runtime.config.Authority.TrustRootPath, maximumTrustRootBytes},
		{runtime.config.Shared.CredentialPath, maximumCredentialBytes},
	} {
		content, err := localauthority.ReadPrivateFile(source.path, source.maximum)
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		_, _ = hash.Write([]byte(source.path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(content)
		_, _ = hash.Write([]byte{0})
	}
	var fingerprint [sha256.Size]byte
	copy(fingerprint[:], hash.Sum(nil))
	return fingerprint, nil
}

// RuntimeStatusPath returns the fixed private status path below managed state.
func RuntimeStatusPath(config Config) string {
	return filepath.Join(config.Storage.StateDirectory, "edge-status.json")
}

// LoadRuntimeStatus strictly reads the aggregate status without acquiring the
// Edge writer lease, so operators can inspect a live process safely.
func LoadRuntimeStatus(config Config) (RuntimeStatus, error) {
	raw, err := localauthority.ReadPrivateFile(
		RuntimeStatusPath(config), maximumRuntimeStatusBytes,
	)
	if err != nil {
		return RuntimeStatus{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var status RuntimeStatus
	if err := decoder.Decode(&status); err != nil {
		return RuntimeStatus{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return RuntimeStatus{}, errors.New("Edge runtime status has trailing content")
	}
	if status.SchemaVersion != RuntimeStatusSchemaVersion ||
		status.EdgeID != config.EdgeID || status.Listen != config.Server.Listen {
		return RuntimeStatus{}, errors.New("Edge runtime status identity mismatch")
	}
	if _, err := time.Parse(time.RFC3339Nano, status.ObservedAt); err != nil {
		return RuntimeStatus{}, errors.New("Edge runtime status timestamp is invalid")
	}
	switch status.State {
	case "STARTING", "READY", "NOT_READY", "STOPPING", "STOPPED":
	default:
		return RuntimeStatus{}, errors.New("Edge runtime status state is invalid")
	}
	if status.State == "READY" {
		if expiresAt, err := time.Parse(time.RFC3339Nano, status.AuthorityExpiresAt); err != nil || status.AuthorityError != "" || expiresAt.IsZero() {
			return RuntimeStatus{}, errors.New("Edge READY status has invalid authority")
		}
	} else if status.AuthorityError != "" &&
		(status.State != "NOT_READY" || status.AuthorityError != "AUTHORITY_UNAVAILABLE") {
		return RuntimeStatus{}, errors.New("Edge runtime status error class is invalid")
	}
	if status.Capacity.CapacityBytes != config.Storage.CapacityBytes ||
		status.Capacity.HighWatermarkBytes != percentage(config.Storage.CapacityBytes, HighWatermarkPercent) ||
		status.Capacity.LowWatermarkBytes != percentage(config.Storage.CapacityBytes, LowWatermarkPercent) ||
		status.Capacity.ProtectedBytes != percentage(config.Storage.CapacityBytes, ProtectedPercent) ||
		status.Capacity.TotalLogicalBytes != status.Capacity.StableBytes+status.Capacity.PendingBytes ||
		status.Capacity.StableBytes < 0 || status.Capacity.PendingBytes < 0 ||
		status.Capacity.ReservedBytes < 0 || status.Capacity.Objects < 0 ||
		status.Pending.Bytes < 0 || status.Pending.Objects < 0 ||
		status.Pending.Queued < 0 || status.Pending.Replicated < 0 ||
		status.Pending.Rejected < 0 ||
		status.Pending.Objects != status.Pending.Queued+status.Pending.Replicated+status.Pending.Rejected {
		return RuntimeStatus{}, errors.New("Edge runtime status counters are invalid")
	}
	return status, nil
}

// StatusReady returns true only for a current READY observation.
func StatusReady(status RuntimeStatus, now time.Time) bool {
	observedAt, err := time.Parse(time.RFC3339Nano, status.ObservedAt)
	expiresAt, expiresErr := time.Parse(time.RFC3339Nano, status.AuthorityExpiresAt)
	return err == nil && expiresErr == nil && status.State == "READY" &&
		!now.UTC().Before(observedAt) && now.UTC().Sub(observedAt) <= StatusFreshness &&
		expiresAt.After(now.UTC())
}

func clearBytes(content []byte) {
	clear(content)
}
