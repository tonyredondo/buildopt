package buildhistory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildsession"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const historyTestToken = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"

func TestHandlerListsFiltersPaginatesAndReturnsDetail(t *testing.T) {
	directory := createHistory(t, []historyFixture{
		{id: "session/old", offset: 0, outcome: sessioningest.OutcomeSuccess, exitCode: 0},
		{id: "session/failure", offset: 1, outcome: sessioningest.OutcomeBuildFailure, exitCode: 7},
		{id: "session/new", offset: 2, outcome: sessioningest.OutcomeSuccess, exitCode: 0},
	})
	handler, err := NewHandler(historyTestToken, directory)
	if err != nil {
		t.Fatalf("create history handler: %v", err)
	}

	unauthorized := serveHistory(handler, http.MethodGet, ListPath, "")
	if unauthorized.Code != http.StatusUnauthorized ||
		strings.Contains(unauthorized.Body.String(), historyTestToken) {
		t.Fatalf("unauthorized response = %d/%q", unauthorized.Code, unauthorized.Body.String())
	}

	first := serveHistory(handler, http.MethodGet, ListPath+"?limit=2", historyTestToken)
	if first.Code != http.StatusOK ||
		first.Header().Get("Cache-Control") != "no-store" ||
		first.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("first page response = %d/%v", first.Code, first.Header())
	}
	var firstPage ListResponse
	if err := json.Unmarshal(first.Body.Bytes(), &firstPage); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if firstPage.SchemaVersion != "buildopt.api/build-session-history/v1" ||
		firstPage.MatchedCount != 3 || len(firstPage.Items) != 2 ||
		firstPage.Items[0].ID != "session/new" ||
		firstPage.Items[1].ID != "session/failure" ||
		firstPage.NextCursor == "" {
		t.Fatalf("unexpected first page: %+v", firstPage)
	}
	if !strings.HasPrefix(firstPage.Items[0].RepositoryID, "hmac-sha256:") {
		t.Fatalf("repository identity was not redacted: %q", firstPage.Items[0].RepositoryID)
	}

	secondTarget := ListPath + "?limit=2&cursor=" + url.QueryEscape(firstPage.NextCursor)
	second := serveHistory(handler, http.MethodGet, secondTarget, historyTestToken)
	var secondPage ListResponse
	if second.Code != http.StatusOK || json.Unmarshal(second.Body.Bytes(), &secondPage) != nil ||
		len(secondPage.Items) != 1 || secondPage.Items[0].ID != "session/old" ||
		secondPage.NextCursor != "" {
		t.Fatalf("unexpected second page: %d/%+v", second.Code, secondPage)
	}

	filterTarget := ListPath + "?outcome=SUCCESS&repository=" +
		url.QueryEscape(firstPage.Items[0].RepositoryID)
	filtered := serveHistory(handler, http.MethodGet, filterTarget, historyTestToken)
	var filteredPage ListResponse
	if filtered.Code != http.StatusOK || json.Unmarshal(filtered.Body.Bytes(), &filteredPage) != nil ||
		filteredPage.MatchedCount != 2 || len(filteredPage.Items) != 2 {
		t.Fatalf("unexpected filtered page: %d/%+v", filtered.Code, filteredPage)
	}

	detail := serveHistory(
		handler,
		http.MethodGet,
		DetailPath+"?id="+url.QueryEscape("session/failure"),
		historyTestToken,
	)
	var detailResponse DetailResponse
	if detail.Code != http.StatusOK || json.Unmarshal(detail.Body.Bytes(), &detailResponse) != nil ||
		detailResponse.SchemaVersion != "buildopt.api/build-session-detail/v1" ||
		detailResponse.Session.Build.ID != "session/failure" ||
		detailResponse.Session.Build.Outcome != sessioningest.OutcomeBuildFailure ||
		len(detailResponse.Session.GradleInvocations) != 1 ||
		len(detailResponse.Session.GradleInvocations[0].RequestedTasks) != 1 ||
		!strings.HasPrefix(
			detailResponse.Session.GradleInvocations[0].RequestedTasks[0],
			"hmac-sha256:",
		) {
		t.Fatalf("unexpected detail: %d/%+v", detail.Code, detailResponse)
	}
}

func TestHandlerRejectsMalformedRequestsAndUnsafeHistory(t *testing.T) {
	directory := createHistory(t, []historyFixture{{
		id: "session-one", outcome: sessioningest.OutcomeSuccess, exitCode: 0,
	}})
	handler, err := NewHandler(historyTestToken, directory)
	if err != nil {
		t.Fatalf("create history handler: %v", err)
	}

	cases := []struct {
		name   string
		method string
		target string
		status int
	}{
		{name: "method", method: http.MethodPost, target: ListPath, status: http.StatusMethodNotAllowed},
		{name: "unknown query", method: http.MethodGet, target: ListPath + "?unknown=1", status: http.StatusBadRequest},
		{name: "duplicate query", method: http.MethodGet, target: ListPath + "?limit=1&limit=2", status: http.StatusBadRequest},
		{name: "invalid limit", method: http.MethodGet, target: ListPath + "?limit=101", status: http.StatusBadRequest},
		{name: "invalid outcome", method: http.MethodGet, target: ListPath + "?outcome=OTHER", status: http.StatusBadRequest},
		{name: "missing identity", method: http.MethodGet, target: DetailPath, status: http.StatusBadRequest},
		{name: "absent identity", method: http.MethodGet, target: DetailPath + "?id=absent", status: http.StatusNotFound},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := serveHistory(handler, testCase.method, testCase.target, historyTestToken)
			if response.Code != testCase.status {
				t.Fatalf("status = %d, want %d", response.Code, testCase.status)
			}
		})
	}

	documents, err := filepath.Glob(filepath.Join(directory, "build-session-*.json"))
	if err != nil || len(documents) != 1 {
		t.Fatalf("history documents = %v/%v", documents, err)
	}
	if err := os.Chmod(documents[0], 0o644); err != nil {
		t.Fatalf("make history document unsafe: %v", err)
	}
	unsafe := serveHistory(handler, http.MethodGet, ListPath, historyTestToken)
	if unsafe.Code != http.StatusInternalServerError ||
		strings.Contains(unsafe.Body.String(), documents[0]) {
		t.Fatalf("unsafe history response = %d/%q", unsafe.Code, unsafe.Body.String())
	}
}

func TestNewHandlerRejectsInvalidConfiguration(t *testing.T) {
	directory := t.TempDir()
	if _, err := NewHandler("short", directory); err == nil {
		t.Fatal("short history token accepted")
	}
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatalf("make directory permissive: %v", err)
	}
	if _, err := NewHandler(historyTestToken, directory); err == nil {
		t.Fatal("permissive history directory accepted")
	}
}

type historyFixture struct {
	id       string
	offset   int
	outcome  string
	exitCode int
}

func createHistory(t *testing.T, fixtures []historyFixture) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "exports")
	exporter, err := buildsession.NewExporter(directory)
	if err != nil {
		t.Fatalf("create history exporter: %v", err)
	}
	t.Cleanup(func() { _ = exporter.Close() })
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	for index, fixture := range fixtures {
		startedAt := base.Add(time.Duration(fixture.offset) * time.Minute)
		completedAt := startedAt.Add(30 * time.Second)
		record := sessioningest.NewRecord(
			fixture.id,
			"generation-1",
			startedAt,
			completedAt,
			fixture.outcome,
			fixture.exitCode,
		)
		record.ExportContext = &sessioningest.ExportContext{
			RepositoryID:         "owner/repository",
			Revision:             "revision-" + fixture.id,
			RequestedTasks:       []string{"build"},
			SourceStateDigest:    "hmac-sha256:" + strings.Repeat("a", 64),
			WorkUnitsFingerprint: "hmac-sha256:" + strings.Repeat("b", 64),
			TokenKeyVersion:      "test-v1",
			TrustDomain:          "test-domain",
		}
		record.GradleInvocation = &sessioningest.GradleInvocation{
			ID:            "invocation-" + string(rune('a'+index)),
			StartedAt:     startedAt.Format(time.RFC3339Nano),
			CompletedAt:   completedAt.Format(time.RFC3339Nano),
			DurationMs:    completedAt.Sub(startedAt).Milliseconds(),
			PluginVersion: "1.0.0",
		}
		if _, _, err := exporter.Export(record); err != nil {
			t.Fatalf("export history fixture %s: %v", fixture.id, err)
		}
	}
	return directory
}

func serveHistory(
	handler http.Handler,
	method string,
	target string,
	token string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
