package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	operationalAlertsPath          = "/ops/v1/alerts"
	storageAlertInterval           = 30 * time.Second
	revocationLagBudget            = 60 * time.Second
	policyFreshnessBudget          = 60 * time.Second
	sqliteContentionBudget         = 250 * time.Millisecond
	acceptanceLatencyBudget        = 500 * time.Millisecond
	acceptanceMinimumSamples       = 20
	acceptanceWindowSize           = 100
	acceptanceErrorRatePercent     = 5
	exportBacklogThreshold         = 16
	diskAvailableThresholdPercent  = 10
	operationalAlertsSchemaVersion = "buildopt.ops/alerts/v1"
	operationalAlertStateOK        = "OK"
	operationalAlertStateFiring    = "FIRING"
)

var operationalAlertIDs = []string{
	"DISK_QUOTA",
	"CORRUPTION",
	"STUCK_ATTEMPT_OR_LEASE",
	"REVOCATION_LAG",
	"POLICY_FRESHNESS",
	"CIRCUIT_BREAKER",
	"SQLITE_CONTENTION",
	"EXPORT_BACKLOG",
	"ACCEPTANCE_ERROR_RATE",
	"ACCEPTANCE_LATENCY",
}

type operationalAlertState struct {
	active bool
	since  time.Time
}

type acceptanceObservation struct {
	duration time.Duration
	failed   bool
}

type operationalAlertMonitor struct {
	mutex                    sync.Mutex
	now                      func() time.Time
	states                   map[string]operationalAlertState
	authorityConfigured      bool
	authorityExpiresAt       time.Time
	authorityReloadStartedAt time.Time
	exportPending            int
	exportFailed             bool
	acceptance               []acceptanceObservation
}

type operationalAlertsDocument struct {
	SchemaVersion string                     `json:"schemaVersion"`
	ObservedAt    string                     `json:"observedAt"`
	Ready         bool                       `json:"ready"`
	Alerts        []operationalAlertDocument `json:"alerts"`
}

type operationalAlertDocument struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Since string `json:"since,omitempty"`
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func newOperationalAlertMonitor(
	now func() time.Time,
) *operationalAlertMonitor {
	if now == nil {
		now = time.Now
	}
	return &operationalAlertMonitor{
		now:    now,
		states: make(map[string]operationalAlertState, len(operationalAlertIDs)),
	}
}

func (monitor *operationalAlertMonitor) instrumentAcceptance(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		if request.URL.Path != "/internal/v1/build-sessions" ||
			request.Method != http.MethodPost {
			next.ServeHTTP(response, request)
			return
		}
		started := monitor.now()
		capturing := &statusCapturingResponseWriter{
			ResponseWriter: response,
		}
		next.ServeHTTP(capturing, request)
		status := capturing.status
		if status == 0 {
			status = http.StatusOK
		}
		monitor.recordAcceptance(
			monitor.now().Sub(started),
			status >= http.StatusInternalServerError,
		)
	})
}

func (response *statusCapturingResponseWriter) WriteHeader(status int) {
	if response.status != 0 {
		return
	}
	response.status = status
	response.ResponseWriter.WriteHeader(status)
}

func (response *statusCapturingResponseWriter) Write(content []byte) (int, error) {
	if response.status == 0 {
		response.WriteHeader(http.StatusOK)
	}
	return response.ResponseWriter.Write(content)
}

func (monitor *operationalAlertMonitor) recordAcceptance(
	duration time.Duration,
	failed bool,
) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	monitor.acceptance = append(
		monitor.acceptance,
		acceptanceObservation{duration: duration, failed: failed},
	)
	if len(monitor.acceptance) > acceptanceWindowSize {
		monitor.acceptance = slices.Clone(
			monitor.acceptance[len(monitor.acceptance)-acceptanceWindowSize:],
		)
	}
	monitor.evaluateAcceptanceLocked(monitor.now().UTC())
}

func (monitor *operationalAlertMonitor) exportStarted() {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	monitor.exportPending++
	monitor.evaluateExportLocked(monitor.now().UTC())
}

func (monitor *operationalAlertMonitor) exportFinished(err error) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	if monitor.exportPending > 0 {
		monitor.exportPending--
	}
	monitor.exportFailed = err != nil
	monitor.evaluateExportLocked(monitor.now().UTC())
}

func (monitor *operationalAlertMonitor) authorityLoaded(expiresAt time.Time) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	monitor.authorityConfigured = true
	monitor.authorityExpiresAt = expiresAt.UTC()
	monitor.authorityReloadStartedAt = time.Time{}
	monitor.setLocked("REVOCATION_LAG", false, monitor.now().UTC())
	monitor.setLocked("CIRCUIT_BREAKER", false, monitor.now().UTC())
	monitor.evaluateTimedLocked(monitor.now().UTC())
}

func (monitor *operationalAlertMonitor) authorityReloadStarted(
	now time.Time,
) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	if monitor.authorityReloadStartedAt.IsZero() {
		monitor.authorityReloadStartedAt = now.UTC()
	}
	monitor.setLocked("CIRCUIT_BREAKER", true, now.UTC())
	monitor.evaluateTimedLocked(now.UTC())
}

func (monitor *operationalAlertMonitor) sampleStorage(
	snapshot sharedcache.OperationalSnapshot,
	err error,
) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	now := monitor.now().UTC()
	diskAlert := err != nil || !snapshot.DiskProbeSucceeded
	if err == nil &&
		snapshot.DiskProbeSucceeded &&
		snapshot.DiskTotalBytes > 0 {
		diskAlert = snapshot.DiskAvailableBytes <=
			snapshot.DiskTotalBytes/
				(100/diskAvailableThresholdPercent)
	}
	monitor.setLocked("DISK_QUOTA", diskAlert, now)
	monitor.setLocked(
		"CORRUPTION",
		err == nil &&
			(!snapshot.IntegrityHealthy || snapshot.QuarantineRecords > 0),
		now,
	)
	monitor.setLocked(
		"STUCK_ATTEMPT_OR_LEASE",
		err == nil && snapshot.ExpiredPendingAttempts > 0,
		now,
	)
	monitor.setLocked(
		"SQLITE_CONTENTION",
		err != nil ||
			!snapshot.SQLiteProbeSucceeded ||
			snapshot.SQLiteProbeDuration > sqliteContentionBudget,
		now,
	)
}

func (monitor *operationalAlertMonitor) serve(
	response http.ResponseWriter,
	request *http.Request,
	ready bool,
) {
	response.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		response.Header().Set("Content-Length", "0")
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	document := monitor.snapshot(ready)
	content, err := json.Marshal(document)
	if err != nil {
		response.Header().Set("Content-Length", "0")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	content = append(content, '\n')
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Content-Length", strconv.Itoa(len(content)))
	response.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		_, _ = bytes.NewReader(content).WriteTo(response)
	}
}

func (monitor *operationalAlertMonitor) snapshot(
	ready bool,
) operationalAlertsDocument {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	now := monitor.now().UTC()
	monitor.evaluateTimedLocked(now)
	alerts := make([]operationalAlertDocument, 0, len(operationalAlertIDs))
	for _, id := range operationalAlertIDs {
		state := monitor.states[id]
		document := operationalAlertDocument{
			ID:    id,
			State: operationalAlertStateOK,
		}
		if state.active {
			document.State = operationalAlertStateFiring
			document.Since = state.since.UTC().Format(time.RFC3339Nano)
		}
		alerts = append(alerts, document)
	}
	return operationalAlertsDocument{
		SchemaVersion: operationalAlertsSchemaVersion,
		ObservedAt:    now.Format(time.RFC3339Nano),
		Ready:         ready,
		Alerts:        alerts,
	}
}

func (monitor *operationalAlertMonitor) evaluateTimedLocked(now time.Time) {
	revocationLag := !monitor.authorityReloadStartedAt.IsZero() &&
		now.Sub(monitor.authorityReloadStartedAt) > revocationLagBudget
	monitor.setLocked("REVOCATION_LAG", revocationLag, now)
	policyStale := monitor.authorityConfigured &&
		!monitor.authorityExpiresAt.After(now.Add(policyFreshnessBudget))
	monitor.setLocked("POLICY_FRESHNESS", policyStale, now)
}

func (monitor *operationalAlertMonitor) evaluateExportLocked(now time.Time) {
	monitor.setLocked(
		"EXPORT_BACKLOG",
		monitor.exportFailed || monitor.exportPending > exportBacklogThreshold,
		now,
	)
}

func (monitor *operationalAlertMonitor) evaluateAcceptanceLocked(now time.Time) {
	if len(monitor.acceptance) < acceptanceMinimumSamples {
		monitor.setLocked("ACCEPTANCE_ERROR_RATE", false, now)
		monitor.setLocked("ACCEPTANCE_LATENCY", false, now)
		return
	}
	failed := 0
	durations := make([]time.Duration, 0, len(monitor.acceptance))
	for _, observation := range monitor.acceptance {
		if observation.failed {
			failed++
		}
		durations = append(durations, observation.duration)
	}
	slices.Sort(durations)
	p95Index := (95*len(durations)+99)/100 - 1
	monitor.setLocked(
		"ACCEPTANCE_ERROR_RATE",
		failed*100 >= len(monitor.acceptance)*acceptanceErrorRatePercent,
		now,
	)
	monitor.setLocked(
		"ACCEPTANCE_LATENCY",
		durations[p95Index] > acceptanceLatencyBudget,
		now,
	)
}

func (monitor *operationalAlertMonitor) setLocked(
	id string,
	active bool,
	now time.Time,
) {
	state := monitor.states[id]
	if active && !state.active {
		state.since = now.UTC()
	}
	if !active {
		state.since = time.Time{}
	}
	state.active = active
	monitor.states[id] = state
}

func watchStorageAlerts(
	ctx context.Context,
	storage *sharedcache.Storage,
	monitor *operationalAlertMonitor,
	interval time.Duration,
) {
	sample := func() {
		probeContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		snapshot, err := storage.OperationalSnapshot(probeContext)
		monitor.sampleStorage(snapshot, err)
	}
	sample()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sample()
		}
	}
}
