package buildhistory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardServesEmbeddedPrivateShell(t *testing.T) {
	handler := NewDashboardHandler()
	response := serveDashboard(handler, http.MethodGet, DashboardPath)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d", response.Code)
	}
	body := response.Body.String()
	for _, marker := range []string{
		`<title>Build history · BuildOpt</title>`,
		`id="credential-form"`,
		`id="build-table-body"`,
		`src="/buildopt/app.js"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("dashboard is missing %q", marker)
		}
	}
	if strings.Contains(body, historyTestToken) ||
		strings.Contains(body, "localStorage") ||
		strings.Contains(body, "sessionStorage") {
		t.Fatal("dashboard shell contains a credential or persistent browser storage")
	}
	if response.Header().Get("Cache-Control") != "no-store" ||
		!strings.Contains(response.Header().Get("Content-Security-Policy"), "default-src 'none'") ||
		response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("dashboard security headers = %v", response.Header())
	}
}

func TestDashboardAssetsAndFailures(t *testing.T) {
	handler := NewDashboardHandler()
	testCases := []struct {
		name        string
		method      string
		path        string
		status      int
		contentType string
		bodyMarker  string
	}{
		{name: "css", method: http.MethodGet, path: dashboardCSSPath, status: http.StatusOK, contentType: "text/css", bodyMarker: "--ink:"},
		{name: "javascript", method: http.MethodGet, path: dashboardJSPath, status: http.StatusOK, contentType: "text/javascript", bodyMarker: "Authorization"},
		{name: "head", method: http.MethodHead, path: DashboardPath, status: http.StatusOK, contentType: "text/html"},
		{name: "missing", method: http.MethodGet, path: DashboardPath + "missing", status: http.StatusNotFound},
		{name: "method", method: http.MethodPost, path: DashboardPath, status: http.StatusMethodNotAllowed},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := serveDashboard(handler, testCase.method, testCase.path)
			if response.Code != testCase.status {
				t.Fatalf("status = %d, want %d", response.Code, testCase.status)
			}
			if testCase.contentType != "" &&
				!strings.HasPrefix(response.Header().Get("Content-Type"), testCase.contentType) {
				t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
			}
			if testCase.bodyMarker != "" && !strings.Contains(response.Body.String(), testCase.bodyMarker) {
				t.Fatalf("body is missing %q", testCase.bodyMarker)
			}
			if testCase.method == http.MethodHead && response.Body.Len() != 0 {
				t.Fatalf("HEAD body length = %d", response.Body.Len())
			}
		})
	}
}

func serveDashboard(handler http.Handler, method string, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
