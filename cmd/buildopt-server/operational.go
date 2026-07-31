package main

import (
	"net/http"
	"sync"
)

const (
	livenessPath  = "/livez"
	readinessPath = "/readyz"
)

// operationalRouter keeps liveness independent from safe serving readiness.
// Product routes stay unavailable until startup reconciliation and authority
// loading have completed.
type operationalRouter struct {
	mutex       sync.RWMutex
	application http.Handler
	ready       bool
	alerts      *operationalAlertMonitor
}

type switchableHandler struct {
	mutex   sync.RWMutex
	handler http.Handler
}

func (router *operationalRouter) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	switch request.URL.Path {
	case livenessPath:
		writeOperationalStatus(response, request, http.StatusOK)
		return
	case readinessPath:
		router.mutex.RLock()
		ready := router.ready
		router.mutex.RUnlock()
		status := http.StatusServiceUnavailable
		if ready {
			status = http.StatusOK
		}
		writeOperationalStatus(response, request, status)
		return
	case operationalAlertsPath:
		router.mutex.RLock()
		ready := router.ready
		alerts := router.alerts
		router.mutex.RUnlock()
		if alerts == nil {
			http.NotFound(response, request)
			return
		}
		alerts.serve(response, request, ready)
		return
	}

	router.mutex.RLock()
	application := router.application
	ready := router.ready
	router.mutex.RUnlock()
	if !ready || application == nil {
		writeOperationalStatus(
			response,
			request,
			http.StatusServiceUnavailable,
		)
		return
	}
	application.ServeHTTP(response, request)
}

func (router *operationalRouter) activate(application http.Handler) {
	router.mutex.Lock()
	defer router.mutex.Unlock()
	router.application = application
	router.ready = application != nil
}

func (router *operationalRouter) deactivate() {
	router.mutex.Lock()
	defer router.mutex.Unlock()
	router.ready = false
}

func (handler *switchableHandler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	handler.mutex.RLock()
	current := handler.handler
	handler.mutex.RUnlock()
	if current == nil {
		writeOperationalStatus(
			response,
			request,
			http.StatusServiceUnavailable,
		)
		return
	}
	current.ServeHTTP(response, request)
}

func (handler *switchableHandler) set(current http.Handler) {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()
	handler.handler = current
}

func writeOperationalStatus(
	response http.ResponseWriter,
	request *http.Request,
	status int,
) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Length", "0")
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	response.WriteHeader(status)
}
