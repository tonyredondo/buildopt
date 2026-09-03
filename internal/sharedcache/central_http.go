package sharedcache

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	// CentralAttemptHeader names the pending Shared attempt for one authorized
	// cache write. Gradle never receives this upstream control identity.
	CentralAttemptHeader = "X-BuildOpt-Cache-Attempt"
	// CentralNamespaceHeader binds every cache request to the exact namespace
	// carried by its token. The invocation-local gateway, never Gradle, sets it.
	CentralNamespaceHeader = "X-BuildOpt-Cache-Namespace"
	centralStatePrefix     = "/api/v1/repositories/"
)

type centralStateRoute struct {
	repositoryScopeSHA256 string
	kind                  StateKind
	resource              string
	digest                string
}

type centralStateCASDocument struct {
	SchemaVersion      string    `json:"schemaVersion"`
	RecordType         string    `json:"recordType"`
	Operation          string    `json:"operation"`
	IdempotencyKey     string    `json:"idempotencyKey"`
	ExpectedGeneration int64     `json:"expectedGeneration"`
	ExpectedHeadSHA256 *string   `json:"expectedHeadSha256"`
	Next               StateHead `json:"next"`
}

// NewCentralHTTPSHandler exposes the existing Gradle object and typed-state
// planes behind one TLS-only, dynamically revocable POC credential boundary.
func NewCentralHTTPSHandler(storage *Storage) (http.Handler, error) {
	if storage == nil {
		return nil, errors.New("central HTTPS storage is nil")
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setCentralResponseHeaders(response)
		if request.TLS == nil {
			response.Header().Set("Upgrade", "TLS/1.3")
			writeCacheStatus(response, http.StatusUpgradeRequired)
			return
		}
		raw, ok := centralBearerToken(request)
		if !ok {
			writeCentralUnauthorized(response)
			return
		}
		defer clear(raw)
		// WCNCP control-plane routes refine transport authority by actor and
		// fork context. Authenticate once for actor grants; the grant inherits
		// token expiry and revocation.
		if strings.Contains(request.URL.Path, "/wcncp/") {
			authorization, grant, tokenID, authorized, err := storage.authenticateWCNCP(request.Context(), raw, storage.now())
			if err != nil {
				writeCacheStatus(response, http.StatusServiceUnavailable)
				return
			}
			if !authorized {
				writeCentralUnauthorized(response)
				return
			}
			ServeWCNCP(storage, authorization, grant, tokenID, response, request)
			return
		}
		finish, err := storage.beginOperation()
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		authorization, authorized, err := authenticateCentralToken(
			request.Context(),
			storage.control.database,
			raw,
			storage.now(),
		)
		finish()
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		if !authorized {
			writeCentralUnauthorized(response)
			return
		}
		if strings.HasPrefix(request.URL.Path, "/cache/") {
			serveCentralCache(storage, authorization, response, request)
			return
		}
		if strings.HasPrefix(request.URL.Path, centralStatePrefix) {
			serveCentralState(storage, authorization, response, request)
			return
		}
		writeCacheStatus(response, http.StatusNotFound)
	}), nil
}

func serveCentralCache(
	storage *Storage,
	authorization CentralTokenAuthorization,
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.URL.RawQuery != "" {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	key, ok := cacheKeyFromPath(request.URL.Path)
	if !ok {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	scope := authorization.Scope
	namespaceValues := request.Header.Values(CentralNamespaceHeader)
	if len(namespaceValues) != 1 || namespaceValues[0] != scope.Namespace {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	switch request.Method {
	case http.MethodGet:
		if !authorization.Has(CentralCacheRead) {
			writeCacheStatus(response, http.StatusForbidden)
			return
		}
		file, object, err := storage.OpenCommittedScoped(
			request.Context(),
			scope.Tenant,
			scope.Repository,
			scope.TrustDomain,
			scope.NamespaceGeneration,
			key,
		)
		if errors.Is(err, ErrCacheMiss) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		defer file.Close()
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", strconv.FormatInt(object.Blob.Size, 10))
		response.Header().Set("ETag", `"`+object.Blob.Digest+`"`)
		response.Header().Set(CommitStateHeader, "COMMITTED")
		response.Header().Set(DecisionDigestHeader, object.DecisionDigest)
		response.WriteHeader(http.StatusOK)
		_, _ = io.Copy(response, file)
	case http.MethodPut:
		if !authorization.Has(CentralCacheWrite) {
			writeCacheStatus(response, http.StatusForbidden)
			return
		}
		attemptValues := request.Header.Values(CentralAttemptHeader)
		if len(attemptValues) != 1 || !validIdentifier(attemptValues[0]) {
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		status, err := storage.AttemptStatus(request.Context(), attemptValues[0])
		if errors.Is(err, ErrAttemptNotFound) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		if status.Repository.Tenant != scope.Tenant ||
			status.Repository.Repository != scope.Repository ||
			status.Repository.TrustDomain != scope.TrustDomain ||
			status.NamespaceGeneration != scope.NamespaceGeneration {
			writeCacheStatus(response, http.StatusForbidden)
			return
		}
		serveBoundCacheHTTP(storage, HTTPBinding{
			Tenant: scope.Tenant, NamespaceGeneration: scope.NamespaceGeneration,
			PendingAttemptID: status.AttemptID, AllowWrite: true,
		}, response, request)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
	}
}

func serveCentralState(
	storage *Storage,
	authorization CentralTokenAuthorization,
	response http.ResponseWriter,
	request *http.Request,
) {
	route, ok := parseCentralStateRoute(request.URL.Path, request.URL.RawQuery)
	if !ok || route.repositoryScopeSHA256 != authorization.Scope.RepositoryScopeSHA256 {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	write := request.Method == http.MethodPut || request.Method == http.MethodPost
	required := CentralStateRead
	if write {
		required = CentralStateWrite
	}
	if !authorization.Has(required) {
		writeCacheStatus(response, http.StatusForbidden)
		return
	}
	switch route.resource {
	case "objects":
		serveCentralStateObject(storage, route, response, request)
	case "manifests":
		serveCentralStateManifest(storage, route, response, request)
	case "head":
		serveCentralStateHead(storage, route, response, request)
	case "head:cas":
		serveCentralStateCAS(storage, route, response, request)
	default:
		writeCacheStatus(response, http.StatusNotFound)
	}
}

func serveCentralStateObject(
	storage *Storage,
	route centralStateRoute,
	response http.ResponseWriter,
	request *http.Request,
) {
	switch request.Method {
	case http.MethodGet:
		file, err := storage.OpenStateObject(
			request.Context(), route.repositoryScopeSHA256, route.kind, route.digest,
		)
		if errors.Is(err, ErrStateNotFound) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		if err != nil {
			writeStateStorageError(response, err)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		response.Header().Set("ETag", `"`+route.digest+`"`)
		response.WriteHeader(http.StatusOK)
		_, _ = io.Copy(response, file)
	case http.MethodPut:
		if !validCentralImmutablePutPrecondition(request) {
			writeCacheStatus(response, http.StatusPreconditionRequired)
			return
		}
		request.Body = http.MaxBytesReader(response, request.Body, maximumStateArtifactBytes+1)
		object, created, err := storage.PutStateObject(
			request.Context(), route.repositoryScopeSHA256, route.kind,
			route.digest, request.Body,
		)
		if err != nil {
			writeStateStorageError(response, err)
			return
		}
		response.Header().Set("ETag", `"`+object.SHA256+`"`)
		if created {
			writeCacheStatus(response, http.StatusCreated)
			return
		}
		writeCacheStatus(response, http.StatusOK)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
	}
}

func serveCentralStateManifest(
	storage *Storage,
	route centralStateRoute,
	response http.ResponseWriter,
	request *http.Request,
) {
	switch request.Method {
	case http.MethodGet:
		snapshot, err := storage.LoadStateManifest(
			request.Context(), route.repositoryScopeSHA256, route.kind, route.digest,
		)
		if errors.Is(err, ErrStateNotFound) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		if err != nil {
			writeStateStorageError(response, err)
			return
		}
		writeCentralJSON(response, snapshot.Manifest, snapshot.ManifestSHA256, http.StatusOK)
	case http.MethodPut:
		if !validCentralImmutablePutPrecondition(request) {
			writeCacheStatus(response, http.StatusPreconditionRequired)
			return
		}
		raw, err := readCentralBody(response, request, maximumStateManifestBytes)
		if err != nil {
			return
		}
		manifest, _, digest, err := decodeStateManifest(raw)
		if err != nil || digest != route.digest ||
			manifest.RepositoryScopeSHA256 != route.repositoryScopeSHA256 ||
			manifest.Kind != route.kind {
			writeCacheStatus(response, http.StatusUnprocessableEntity)
			return
		}
		snapshot, created, err := storage.PutStateManifest(request.Context(), raw)
		if err != nil {
			writeStateStorageError(response, err)
			return
		}
		response.Header().Set("ETag", `"`+snapshot.ManifestSHA256+`"`)
		if created {
			writeCacheStatus(response, http.StatusCreated)
			return
		}
		writeCacheStatus(response, http.StatusOK)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
	}
}

func serveCentralStateHead(
	storage *Storage,
	route centralStateRoute,
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", "GET")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	snapshot, err := storage.LoadCurrentState(
		request.Context(), route.repositoryScopeSHA256, route.kind,
	)
	if errors.Is(err, ErrStateNotFound) {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	if err != nil {
		writeStateStorageError(response, err)
		return
	}
	writeCentralJSON(response, snapshot.Head, snapshot.HeadSHA256, http.StatusOK)
}

func serveCentralStateCAS(
	storage *Storage,
	route centralStateRoute,
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", "POST")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	raw, err := readCentralBody(response, request, maximumStateManifestBytes)
	if err != nil {
		return
	}
	var document centralStateCASDocument
	if decodeStrictStateJSON(raw, &document) != nil ||
		document.SchemaVersion != "buildopt.central/state-cas/v1" ||
		document.RecordType != "CENTRAL_STATE_CAS" ||
		document.Operation != "CREATE_OR_ADVANCE" ||
		!validSHA256(document.IdempotencyKey) || document.ExpectedGeneration < 0 ||
		document.ExpectedGeneration >= maximumStateGeneration ||
		document.Next.RepositoryScopeSHA256 != route.repositoryScopeSHA256 ||
		document.Next.Kind != route.kind ||
		document.Next.Generation != document.ExpectedGeneration+1 {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	expectedHead := ""
	if document.ExpectedHeadSHA256 != nil {
		expectedHead = *document.ExpectedHeadSHA256
	}
	if !validCentralCASPrecondition(request, document.ExpectedGeneration, expectedHead) {
		writeCacheStatus(response, http.StatusPreconditionRequired)
		return
	}
	result, err := storage.CASStateHead(request.Context(), StateCASRequest{
		RepositoryScopeSHA256: route.repositoryScopeSHA256,
		Kind:                  route.kind, IdempotencyKey: document.IdempotencyKey,
		ExpectedGeneration: document.ExpectedGeneration,
		ExpectedHeadSHA256: expectedHead,
		ManifestSHA256:     document.Next.ManifestSHA256,
		ProposedHead:       &document.Next,
	})
	if err != nil {
		writeStateStorageError(response, err)
		return
	}
	response.Header().Set("X-BuildOpt-Idempotent-Replay", strconv.FormatBool(result.Replayed))
	status := http.StatusOK
	if document.ExpectedGeneration == 0 && !result.Replayed {
		status = http.StatusCreated
	}
	writeCentralJSON(response, result.Head, result.HeadSHA256, status)
}

func parseCentralStateRoute(path, query string) (centralStateRoute, bool) {
	if query != "" || !strings.HasPrefix(path, centralStatePrefix) {
		return centralStateRoute{}, false
	}
	segments := strings.Split(strings.TrimPrefix(path, centralStatePrefix), "/")
	if len(segments) < 4 || len(segments) > 5 || !validSHA256(segments[0]) ||
		segments[1] != "state" {
		return centralStateRoute{}, false
	}
	kinds := map[string]StateKind{
		"portfolios":  StateKindPortfolio,
		"evidence":    StateKindEvidence,
		"checkpoints": StateKindCheckpoint,
	}
	kind, ok := kinds[segments[2]]
	if !ok {
		return centralStateRoute{}, false
	}
	route := centralStateRoute{
		repositoryScopeSHA256: segments[0], kind: kind, resource: segments[3],
	}
	if route.resource == "head" || route.resource == "head:cas" {
		return route, len(segments) == 4
	}
	if (route.resource != "objects" && route.resource != "manifests") ||
		len(segments) != 5 || !validSHA256(segments[4]) {
		return centralStateRoute{}, false
	}
	route.digest = segments[4]
	return route, true
}

func validCentralCASPrecondition(request *http.Request, generation int64, digest string) bool {
	if generation == 0 {
		return digest == "" && len(request.Header.Values("If-Match")) == 0 &&
			len(request.Header.Values("If-None-Match")) == 1 &&
			request.Header.Get("If-None-Match") == "*"
	}
	return validSHA256(digest) && len(request.Header.Values("If-None-Match")) == 0 &&
		len(request.Header.Values("If-Match")) == 1 &&
		request.Header.Get("If-Match") == `"`+digest+`"`
}

func validCentralImmutablePutPrecondition(request *http.Request) bool {
	return len(request.Header.Values("If-Match")) == 0 &&
		len(request.Header.Values("If-None-Match")) == 1 &&
		request.Header.Get("If-None-Match") == "*"
}

func readCentralBody(
	response http.ResponseWriter,
	request *http.Request,
	maximum int64,
) ([]byte, error) {
	request.Body = http.MaxBytesReader(response, request.Body, maximum+1)
	raw, err := io.ReadAll(request.Body)
	if err != nil || len(raw) == 0 || int64(len(raw)) > maximum {
		writeCacheStatus(response, http.StatusRequestEntityTooLarge)
		return nil, errors.New("central request body is invalid")
	}
	return raw, nil
}

func writeCentralJSON(response http.ResponseWriter, value any, digest string, status int) {
	raw, actual, err := canonicalStateValue(value)
	if err != nil || actual != digest {
		writeCacheStatus(response, http.StatusServiceUnavailable)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	response.Header().Set("ETag", `"`+digest+`"`)
	response.WriteHeader(status)
	_, _ = io.Copy(response, bytes.NewReader(raw))
}

func writeStateStorageError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrStateNotFound):
		writeCacheStatus(response, http.StatusNotFound)
	case errors.Is(err, ErrStateHeadPrecondition):
		writeCacheStatus(response, http.StatusPreconditionFailed)
	case errors.Is(err, ErrStateGenerationConflict), errors.Is(err, ErrStateIdempotency):
		writeCacheStatus(response, http.StatusConflict)
	case errors.Is(err, ErrStateDigestMismatch),
		errors.Is(err, ErrStateManifestIncomplete), errors.Is(err, ErrStateInvalid):
		writeCacheStatus(response, http.StatusUnprocessableEntity)
	default:
		writeCacheStatus(response, http.StatusServiceUnavailable)
	}
}

func writeCentralUnauthorized(response http.ResponseWriter) {
	response.Header().Set("WWW-Authenticate", `Bearer realm="buildopt-central-poc"`)
	writeCacheStatus(response, http.StatusUnauthorized)
}

func setCentralResponseHeaders(response http.ResponseWriter) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Referrer-Policy", "no-referrer")
}
