package sharedcache

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	// CommitStateHeader proves that a successful remote read came from the
	// Shared committed route rather than a pending or generic byte endpoint.
	CommitStateHeader = "X-BuildOpt-Commit-State"
	// DecisionDigestHeader binds the hit to Shared's durable commit decision.
	DecisionDigestHeader = "X-BuildOpt-Decision-Digest"
)

// HTTPBinding is immutable authority already established by the local
// verifying gateway. The handler deliberately performs no credential parsing;
// callers must place it behind the A0-006 authenticated context boundary.
type HTTPBinding struct {
	Tenant              string
	NamespaceGeneration int64
	PendingAttemptID    string
	AllowRead           bool
	AllowWrite          bool
}

// NewHTTPHandler creates a Gradle-compatible opaque GET/PUT handler at
// /cache/{key}, bound to one already-authenticated namespace and attempt.
func NewHTTPHandler(
	storage *Storage,
	binding HTTPBinding,
) (http.Handler, error) {
	if storage == nil ||
		!validIdentifier(binding.Tenant) ||
		binding.NamespaceGeneration < 1 ||
		(!binding.AllowRead && !binding.AllowWrite) {
		return nil, errors.New("invalid Shared cache HTTP binding")
	}
	if binding.AllowWrite && !validIdentifier(binding.PendingAttemptID) {
		return nil, errors.New(
			"write-enabled Shared cache HTTP binding needs an attempt",
		)
	}
	if !binding.AllowWrite && binding.PendingAttemptID != "" {
		return nil, errors.New(
			"read-only Shared cache HTTP binding cannot name an attempt",
		)
	}
	return http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		serveBoundCacheHTTP(storage, binding, response, request)
	}), nil
}

func serveBoundCacheHTTP(
	storage *Storage,
	binding HTTPBinding,
	response http.ResponseWriter,
	request *http.Request,
) {
	key, ok := cacheKeyFromPath(request.URL.Path)
	if !ok {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	switch request.Method {
	case http.MethodGet:
		if !binding.AllowRead {
			writeCacheStatus(response, http.StatusForbidden)
			return
		}
		serveCommittedGET(
			request.Context(),
			storage,
			binding,
			key,
			response,
		)
	case http.MethodPut:
		if !binding.AllowWrite {
			writeCacheStatus(response, http.StatusForbidden)
			return
		}
		servePendingPUT(
			request.Context(),
			storage,
			binding,
			key,
			request,
			response,
		)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
	}
}

func serveCommittedGET(
	ctx context.Context,
	storage *Storage,
	binding HTTPBinding,
	key string,
	response http.ResponseWriter,
) {
	file, object, err := storage.OpenCommitted(
		ctx,
		binding.Tenant,
		binding.NamespaceGeneration,
		key,
	)
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		writeCacheStatus(response, http.StatusServiceUnavailable)
		return
	}
	defer file.Close()
	response.Header().Set("Content-Type", "application/octet-stream")
	response.Header().Set(
		"Content-Length",
		strconv.FormatInt(object.Blob.Size, 10),
	)
	response.Header().Set("ETag", `"`+object.Blob.Digest+`"`)
	response.Header().Set(CommitStateHeader, "COMMITTED")
	response.Header().Set(DecisionDigestHeader, object.DecisionDigest)
	response.WriteHeader(http.StatusOK)
	_, _ = io.Copy(response, file)
}

func servePendingPUT(
	ctx context.Context,
	storage *Storage,
	binding HTTPBinding,
	key string,
	request *http.Request,
	response http.ResponseWriter,
) {
	status, err := storage.AttemptStatus(ctx, binding.PendingAttemptID)
	if err != nil {
		if errors.Is(err, ErrAttemptNotFound) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		writeCacheStatus(response, http.StatusServiceUnavailable)
		return
	}
	if status.Repository.Tenant != binding.Tenant ||
		status.NamespaceGeneration != binding.NamespaceGeneration {
		writeCacheStatus(response, http.StatusForbidden)
		return
	}
	if status.State != AttemptPending {
		writeCacheStatus(response, http.StatusConflict)
		return
	}
	if request.ContentLength > storage.MaximumObjectBytes() {
		writeCacheStatus(response, http.StatusRequestEntityTooLarge)
		return
	}
	result, err := storage.PutPendingSized(
		ctx,
		binding.PendingAttemptID,
		key,
		request.ContentLength,
		request.Body,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrBlobTooLarge),
			errors.Is(err, ErrCapacityExceeded):
			writeCacheStatus(response, http.StatusRequestEntityTooLarge)
		case errors.Is(err, ErrAttemptNotFound):
			writeCacheStatus(response, http.StatusNotFound)
		case errors.Is(err, ErrAttemptConflict),
			errors.Is(err, ErrStatePrecondition):
			writeCacheStatus(response, http.StatusConflict)
		case errors.Is(err, context.Canceled),
			errors.Is(err, context.DeadlineExceeded):
			writeCacheStatus(response, http.StatusRequestTimeout)
		default:
			writeCacheStatus(response, http.StatusServiceUnavailable)
		}
		return
	}
	response.Header().Set("X-BuildOpt-Blob-Digest", result.Object.Checksum)
	if result.ObjectAdded {
		writeCacheStatus(response, http.StatusCreated)
		return
	}
	writeCacheStatus(response, http.StatusOK)
}

func cacheKeyFromPath(path string) (string, bool) {
	const prefix = "/cache/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	key := strings.TrimPrefix(path, prefix)
	if strings.Contains(key, "/") || !validCacheKey(key) {
		return "", false
	}
	return key, true
}

func writeCacheStatus(response http.ResponseWriter, status int) {
	response.Header().Set("Content-Length", "0")
	response.WriteHeader(status)
}
