package buildhistory

import (
	_ "embed"
	"net/http"
)

const (
	// DashboardPath is the embedded local build-history interface.
	DashboardPath    = "/buildopt/"
	dashboardCSSPath = "/buildopt/app.css"
	dashboardJSPath  = "/buildopt/app.js"
)

//go:embed web/index.html
var dashboardHTML []byte

//go:embed web/app.css
var dashboardCSS []byte

//go:embed web/app.js
var dashboardJS []byte

type dashboardHandler struct{}

// NewDashboardHandler returns the dependency-free embedded local interface.
// The shell contains no history data or credential; its API calls use a
// bearer retained only in the page memory.
func NewDashboardHandler() http.Handler {
	return dashboardHandler{}
}

func (dashboardHandler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Security-Policy",
		"default-src 'none'; script-src 'self'; style-src 'self'; "+
			"connect-src 'self'; img-src 'self' data:; base-uri 'none'; "+
			"form-action 'self'; frame-ancestors 'none'")
	writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("X-Frame-Options", "DENY")

	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var content []byte
	switch request.URL.Path {
	case DashboardPath:
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		content = dashboardHTML
	case dashboardCSSPath:
		writer.Header().Set("Content-Type", "text/css; charset=utf-8")
		content = dashboardCSS
	case dashboardJSPath:
		writer.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		content = dashboardJS
	default:
		http.NotFound(writer, request)
		return
	}
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodHead {
		return
	}
	_, _ = writer.Write(content)
}
