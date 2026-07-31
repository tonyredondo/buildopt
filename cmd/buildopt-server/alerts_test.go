package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestOperationalAlertsExposeExactClassesAndRecover(t *testing.T) {
	current := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	monitor := newOperationalAlertMonitor(func() time.Time {
		return current
	})
	monitor.sampleStorage(sharedcache.OperationalSnapshot{
		DiskTotalBytes:         100,
		DiskAvailableBytes:     50,
		DiskProbeSucceeded:     true,
		CapacityHighWatermark:  true,
		ExpiredPendingAttempts: 1,
		QuarantineRecords:      1,
		IntegrityHealthy:       false,
		SQLiteProbeSucceeded:   false,
		SQLiteProbeDuration:    time.Second,
	}, nil)
	monitor.authorityLoaded(current.Add(30 * time.Second))
	monitor.authorityReloadStarted(current)
	monitor.exportStarted()
	monitor.exportFinished(errors.New("fixture export unavailable"))
	for index := 0; index < acceptanceMinimumSamples; index++ {
		monitor.recordAcceptance(
			acceptanceLatencyBudget+time.Millisecond,
			index == 0,
		)
	}

	current = current.Add(revocationLagBudget + time.Second)
	document := requestOperationalAlerts(t, monitor, true)
	if !document.Ready ||
		document.SchemaVersion != operationalAlertsSchemaVersion ||
		len(document.Alerts) != len(operationalAlertIDs) {
		t.Fatalf("alert document = %+v", document)
	}
	actualIDs := make([]string, 0, len(document.Alerts))
	for _, alert := range document.Alerts {
		actualIDs = append(actualIDs, alert.ID)
		if alert.State != operationalAlertStateFiring || alert.Since == "" {
			t.Fatalf("alert did not fire: %+v", alert)
		}
	}
	if !slices.Equal(actualIDs, operationalAlertIDs) {
		t.Fatalf("alert IDs = %v", actualIDs)
	}

	monitor.sampleStorage(sharedcache.OperationalSnapshot{
		DiskTotalBytes:       100,
		DiskAvailableBytes:   50,
		DiskProbeSucceeded:   true,
		IntegrityHealthy:     true,
		SQLiteProbeSucceeded: true,
		SQLiteProbeDuration:  time.Millisecond,
	}, nil)
	monitor.authorityLoaded(current.Add(time.Hour))
	monitor.exportStarted()
	monitor.exportFinished(nil)
	for index := 0; index < acceptanceWindowSize; index++ {
		monitor.recordAcceptance(time.Millisecond, false)
	}
	document = requestOperationalAlerts(t, monitor, false)
	if document.Ready {
		t.Fatal("alert document invented readiness")
	}
	for _, alert := range document.Alerts {
		if alert.State != operationalAlertStateOK || alert.Since != "" {
			t.Fatalf("alert did not recover: %+v", alert)
		}
	}
}

func TestAcceptanceInstrumentationRecordsServerErrors(t *testing.T) {
	now := time.Date(2026, time.July, 30, 13, 0, 0, 0, time.UTC)
	monitor := newOperationalAlertMonitor(func() time.Time {
		return now
	})
	handler := monitor.instrumentAcceptance(http.HandlerFunc(func(
		response http.ResponseWriter,
		_ *http.Request,
	) {
		response.WriteHeader(http.StatusServiceUnavailable)
	}))
	for index := 0; index < acceptanceMinimumSamples; index++ {
		request := httptest.NewRequest(
			http.MethodPost,
			"/internal/v1/build-sessions",
			nil,
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("instrumented status = %d", response.Code)
		}
	}
	document := monitor.snapshot(true)
	state := alertStateByID(t, document, "ACCEPTANCE_ERROR_RATE")
	if state != operationalAlertStateFiring {
		t.Fatalf("acceptance error state = %s", state)
	}
}

func TestOperationalAlertEndpointSupportsReadOnlyAccess(t *testing.T) {
	monitor := newOperationalAlertMonitor(func() time.Time {
		return time.Date(2026, time.July, 30, 14, 0, 0, 0, time.UTC)
	})
	router := &operationalRouter{alerts: monitor}

	get := httptest.NewRecorder()
	router.ServeHTTP(
		get,
		httptest.NewRequest(http.MethodGet, operationalAlertsPath, nil),
	)
	if get.Code != http.StatusOK ||
		get.Header().Get("Cache-Control") != "no-store" ||
		get.Header().Get("Content-Type") != "application/json" ||
		get.Body.Len() == 0 {
		t.Fatalf("GET alert endpoint = %d/%q/%q/%q",
			get.Code,
			get.Header().Get("Cache-Control"),
			get.Header().Get("Content-Type"),
			get.Body.String(),
		)
	}

	head := httptest.NewRecorder()
	router.ServeHTTP(
		head,
		httptest.NewRequest(http.MethodHead, operationalAlertsPath, nil),
	)
	if head.Code != http.StatusOK ||
		head.Body.Len() != 0 ||
		head.Header().Get("Content-Length") == "" {
		t.Fatalf("HEAD alert endpoint = %d/%q/%q",
			head.Code,
			head.Header().Get("Content-Length"),
			head.Body.String(),
		)
	}

	post := httptest.NewRecorder()
	router.ServeHTTP(
		post,
		httptest.NewRequest(http.MethodPost, operationalAlertsPath, nil),
	)
	if post.Code != http.StatusMethodNotAllowed ||
		post.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST alert endpoint = %d/%q", post.Code, post.Header().Get("Allow"))
	}
}

func requestOperationalAlerts(
	t *testing.T,
	monitor *operationalAlertMonitor,
	ready bool,
) operationalAlertsDocument {
	t.Helper()
	response := httptest.NewRecorder()
	monitor.serve(
		response,
		httptest.NewRequest(http.MethodGet, operationalAlertsPath, nil),
		ready,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("alert response = %d", response.Code)
	}
	var document operationalAlertsDocument
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		t.Fatal(err)
	}
	return document
}

func alertStateByID(
	t *testing.T,
	document operationalAlertsDocument,
	id string,
) string {
	t.Helper()
	for _, alert := range document.Alerts {
		if alert.ID == id {
			return alert.State
		}
	}
	t.Fatalf("missing alert %s", id)
	return ""
}
