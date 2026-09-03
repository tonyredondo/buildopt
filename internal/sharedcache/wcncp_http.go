package sharedcache

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	wcncpRoutePrefix       = "/api/v1/repositories/"
	wcncpForkHeader        = "X-BuildOpt-Fork"
	wcncpRequestHeader     = "X-BuildOpt-Request-ID"
	wcncpLeaseHolderHeader = "X-BuildOpt-Lease-Holder"
	// Bulk observation upload bounds keep wrapper post-child work bounded.
	wcncpBatchMaxItems = 32
	wcncpBatchMaxBytes = 1 << 20
)

type wcncpHTTPRoute struct {
	repositoryScopeSHA256 string
	kind                  StateKind
	resource              string
	digest                string
}

func parseWCNCPRoute(path, rawQuery string) (wcncpHTTPRoute, bool) {
	var route wcncpHTTPRoute
	if rawQuery != "" || !strings.HasPrefix(path, wcncpRoutePrefix) {
		return route, false
	}
	rest := strings.TrimPrefix(path, wcncpRoutePrefix)
	parts := strings.Split(rest, "/")
	if len(parts) == 3 && parts[1] == "wcncp" && parts[2] == "status" && validSHA256(parts[0]) {
		return wcncpHTTPRoute{repositoryScopeSHA256: parts[0], resource: "status"}, true
	}
	// /api/v1/repositories/{scope}/wcncp/{kind}/{resource}[/{digest}]
	if len(parts) < 4 || parts[1] != "wcncp" {
		return route, false
	}
	route.repositoryScopeSHA256 = parts[0]
	if !validSHA256(route.repositoryScopeSHA256) {
		return route, false
	}
	switch StateKind(parts[2]) {
	case WCNCPKindObservation, WCNCPKindOpportunity, WCNCPKindProposal, WCNCPKindValidation, WCNCPKindDecision:
		route.kind = StateKind(parts[2])
	default:
		return route, false
	}
	route.resource = parts[3]
	switch route.resource {
	case "objects", "manifests":
		if len(parts) != 5 || !validSHA256(parts[4]) {
			return route, false
		}
		route.digest = parts[4]
	case "head", "cas", "batch", "snapshot", "claim", "heartbeat", "release":
		if len(parts) != 4 {
			return route, false
		}
	default:
		return route, false
	}
	return route, true
}

// ServeWCNCP exposes the typed WCNCP control plane behind the existing TLS
// and revocable credential boundary with actor-refined authority. It shares
// the central HTTPS handler's TLS and token verification but never shares
// Gradle cache routes, metadata, or decision authority.
func ServeWCNCP(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, response http.ResponseWriter, request *http.Request) {
	route, ok := parseWCNCPRoute(request.URL.Path, request.URL.RawQuery)
	if !ok || route.repositoryScopeSHA256 != authorization.Scope.RepositoryScopeSHA256 {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	forked := isWCNCPFrok(request)
	requestID := wcncpRequestID(request)
	switch route.resource {
	case "status":
		serveWCNCPProjection(storage, authorization, route, response, request)
	case "objects":
		serveWCNCPObject(storage, authorization, grant, tokenID, route, requestID, forked, response, request)
	case "manifests":
		serveWCNCPManifest(storage, authorization, grant, tokenID, route, requestID, forked, response, request)
	case "head", "snapshot":
		serveWCNCPSnapshot(storage, authorization, grant, tokenID, route, requestID, response, request)
	case "cas":
		serveWCNCPHeadCAS(storage, authorization, grant, tokenID, route, requestID, forked, response, request)
	case "batch":
		serveWCNCPBatch(storage, authorization, grant, tokenID, route, requestID, forked, response, request)
	case "claim", "heartbeat", "release":
		serveWCNCPLease(storage, authorization, grant, tokenID, route, requestID, forked, response, request)
	default:
		writeCacheStatus(response, http.StatusNotFound)
	}
}

func isWCNCPFrok(request *http.Request) bool {
	values := request.Header.Values(wcncpForkHeader)
	return len(values) == 1 && (values[0] == "1" || strings.EqualFold(values[0], "true"))
}

func wcncpRequestID(request *http.Request) string {
	values := request.Header.Values(wcncpRequestHeader)
	if len(values) == 1 && len(values[0]) >= 8 && len(values[0]) <= 128 {
		return values[0]
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "wcncp-request"
	}
	return hex.EncodeToString(raw)
}

func serveWCNCPObject(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, route wcncpHTTPRoute, requestID string, forked bool, response http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpSnapshotRead, route.repositoryScopeSHA256, route.kind, false); err != nil {
			writeWCNCPAuthError(response, err)
			return
		}
		file, err := storage.OpenWCNCPObject(request.Context(), route.repositoryScopeSHA256, route.kind, route.digest)
		if errors.Is(err, ErrWCNCPNotFound) {
			recordWCNCPEvent(request, storage, requestID, tokenID, route, "object-get", route.digest, "not-found")
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		if err != nil {
			writeWCNCPStorageError(response, err)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, "object-get", route.digest, "ok")
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		response.Header().Set("ETag", `"`+route.digest+`"`)
		response.WriteHeader(http.StatusOK)
		_, _ = io.Copy(response, file)
	case http.MethodPut:
		if err := authorizeWCNCPRecordWrite(authorization, grant, route, forked); err != nil {
			writeWCNCPAuthError(response, err)
			return
		}
		if !validCentralImmutablePutPrecondition(request) {
			writeCacheStatus(response, http.StatusPreconditionRequired)
			return
		}
		request.Body = http.MaxBytesReader(response, request.Body, maximumWCNCPArtifactBytes+1)
		var input io.Reader = request.Body
		if route.kind == WCNCPKindValidation {
			raw, err := io.ReadAll(request.Body)
			if err != nil {
				writeCacheStatus(response, http.StatusBadRequest)
				return
			}
			var validation WCNCPValidation
			if err := decodeStrictWCNCPJSON(raw, &validation); err != nil {
				writeCacheStatus(response, http.StatusBadRequest)
				return
			}
			holder := request.Header.Get(wcncpLeaseHolderHeader)
			if err := storage.RequireWCNCPLease(request.Context(), validation.LeaseSHA256, holder, validation.ProposalSHA256, storage.now()); err != nil {
				recordWCNCPEvent(request, storage, requestID, tokenID, route, "object-put", route.digest, "lease-rejected")
				writeWCNCPLeaseError(response, err)
				return
			}
			input = bytes.NewReader(raw)
		}
		object, _, err := storage.PutWCNCPObject(request.Context(), route.repositoryScopeSHA256, route.kind, route.digest, input)
		if errors.Is(err, ErrWCNCPInvalid) || errors.Is(err, ErrWCNCPDigestMismatch) {
			recordWCNCPEvent(request, storage, requestID, tokenID, route, "object-put", route.digest, "rejected")
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		if err != nil {
			writeWCNCPStorageError(response, err)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, "object-put", object.SHA256, "ok")
		response.Header().Set("ETag", `"`+object.SHA256+`"`)
		response.WriteHeader(http.StatusCreated)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
	}
}

func authorizeWCNCPRecordWrite(authorization CentralTokenAuthorization, grant WCNCPGrant, route wcncpHTTPRoute, forked bool) error {
	switch route.kind {
	case WCNCPKindObservation:
		return AuthorizeWCNCPOperation(authorization, grant, WCNCPOpObservationWrite, route.repositoryScopeSHA256, route.kind, forked)
	case WCNCPKindValidation:
		return AuthorizeWCNCPOperation(authorization, grant, WCNCPOpValidationWrite, route.repositoryScopeSHA256, route.kind, forked)
	case WCNCPKindDecision:
		return AuthorizeWCNCPOperation(authorization, grant, WCNCPOpDecisionWrite, route.repositoryScopeSHA256, route.kind, forked)
	case WCNCPKindOpportunity, WCNCPKindProposal:
		return ErrWCNCPForbidden
	default:
		return ErrWCNCPInvalid
	}
}

func serveWCNCPManifest(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, route wcncpHTTPRoute, requestID string, forked bool, response http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpSnapshotRead, route.repositoryScopeSHA256, route.kind, false); err != nil {
			writeWCNCPAuthError(response, err)
			return
		}
		snapshot, err := storage.LoadWCNCPManifest(request.Context(), route.repositoryScopeSHA256, route.kind, route.digest)
		if errors.Is(err, ErrWCNCPNotFound) {
			writeCacheStatus(response, http.StatusNotFound)
			return
		}
		if err != nil {
			writeWCNCPStorageError(response, err)
			return
		}
		raw, _, err := canonicalStateValue(snapshot.Manifest)
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, "manifest-get", route.digest, "ok")
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Content-Length", strconv.Itoa(len(raw)))
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(raw)
	case http.MethodPut:
		if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpManifestWrite, route.repositoryScopeSHA256, route.kind, forked); err != nil {
			writeWCNCPAuthError(response, err)
			return
		}
		request.Body = http.MaxBytesReader(response, request.Body, maximumWCNCPManifestBytes+1)
		raw, err := io.ReadAll(request.Body)
		if err != nil {
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		snapshot, _, err := storage.PutWCNCPManifest(request.Context(), raw)
		if errors.Is(err, ErrWCNCPInvalid) {
			recordWCNCPEvent(request, storage, requestID, tokenID, route, "manifest-put", route.digest, "rejected")
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrWCNCPManifestIncomplete) {
			writeCacheStatus(response, http.StatusPreconditionFailed)
			return
		}
		if errors.Is(err, ErrWCNCPGenerationConflict) {
			writeCacheStatus(response, http.StatusConflict)
			return
		}
		if err != nil {
			writeWCNCPStorageError(response, err)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, "manifest-put", snapshot.ManifestSHA256, "ok")
		response.Header().Set("ETag", `"`+snapshot.ManifestSHA256+`"`)
		response.WriteHeader(http.StatusCreated)
	default:
		response.Header().Set("Allow", "GET, PUT")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
	}
}

func serveWCNCPSnapshot(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, route wcncpHTTPRoute, requestID string, response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", "GET")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpSnapshotRead, route.repositoryScopeSHA256, route.kind, false); err != nil {
		writeWCNCPAuthError(response, err)
		return
	}
	snapshot, err := storage.LoadCurrentWCNCP(request.Context(), route.repositoryScopeSHA256, route.kind)
	if errors.Is(err, ErrWCNCPNotFound) {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	if err != nil {
		writeWCNCPStorageError(response, err)
		return
	}
	headRaw, _, err := canonicalStateValue(snapshot.Head)
	if err != nil {
		writeCacheStatus(response, http.StatusServiceUnavailable)
		return
	}
	manifestRaw, _, err := canonicalStateValue(snapshot.Manifest)
	if err != nil {
		writeCacheStatus(response, http.StatusServiceUnavailable)
		return
	}
	// Verified local snapshot: head and manifest are revalidated by
	// LoadCurrentWCNCP before encoding. It cannot advance proposal or
	// validation state; it only reports it.
	body := append(append([]byte(`{"head":`), headRaw...), append([]byte(`,"manifest":`), manifestRaw...)...)
	body = append(body, []byte(`}`)...)
	recordWCNCPEvent(request, storage, requestID, tokenID, route, "snapshot-get", snapshot.ManifestSHA256, "ok")
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Content-Length", strconv.Itoa(len(body)))
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(body)
}

func serveWCNCPHeadCAS(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, route wcncpHTTPRoute, requestID string, forked bool, response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", "POST")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpHeadCAS, route.repositoryScopeSHA256, route.kind, forked); err != nil {
		writeWCNCPAuthError(response, err)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, maximumWCNCPManifestBytes+1)
	var document struct {
		IdempotencyKey     string     `json:"idempotencyKey"`
		ExpectedGeneration int64      `json:"expectedGeneration"`
		ExpectedHeadSHA256 *string    `json:"expectedHeadSha256"`
		ManifestSHA256     string     `json:"manifestSha256"`
		Next               *WCNCPHead `json:"next"`
	}
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	if err := decodeStrictWCNCPJSON(bytes.Clone(raw), &document); err != nil {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	casRequest := WCNCPCASRequest{
		RepositoryScopeSHA256: route.repositoryScopeSHA256, Kind: route.kind,
		IdempotencyKey: document.IdempotencyKey, ExpectedGeneration: document.ExpectedGeneration,
		ManifestSHA256: document.ManifestSHA256, ProposedHead: document.Next,
	}
	if document.ExpectedHeadSHA256 != nil {
		casRequest.ExpectedHeadSHA256 = *document.ExpectedHeadSHA256
	}
	result, err := storage.CASWCNCPHead(request.Context(), casRequest)
	if errors.Is(err, ErrWCNCPInvalid) {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrWCNCPIdempotency) {
		writeCacheStatus(response, http.StatusConflict)
		return
	}
	if errors.Is(err, ErrWCNCPHeadPrecondition) || errors.Is(err, ErrWCNCPGenerationConflict) {
		writeCacheStatus(response, http.StatusPreconditionFailed)
		return
	}
	if errors.Is(err, ErrWCNCPManifestIncomplete) {
		writeCacheStatus(response, http.StatusPreconditionFailed)
		return
	}
	if err != nil {
		writeWCNCPStorageError(response, err)
		return
	}
	recordWCNCPEvent(request, storage, requestID, tokenID, route, "head-cas", result.HeadSHA256, "ok")
	encoded, _, err := canonicalStateValue(result.Head)
	if err != nil {
		writeCacheStatus(response, http.StatusServiceUnavailable)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(encoded)
}

func serveWCNCPBatch(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, route wcncpHTTPRoute, requestID string, forked bool, response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", "POST")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	// Bulk observation upload reduces wrapper overhead with a fixed item and
	// byte maximum. Partial batches fail closed: nothing is published unless
	// every item validates and persists.
	if route.kind != WCNCPKindObservation {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpBatchWrite, route.repositoryScopeSHA256, route.kind, forked); err != nil {
		writeWCNCPAuthError(response, err)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, wcncpBatchMaxBytes+1)
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	var batch []interface{}
	decoder := strictWCNCPDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&batch); err != nil {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	if len(batch) == 0 || len(batch) > wcncpBatchMaxItems {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	// Re-encode each item canonically and validate before any persistence so
	// a single bad item rejects the whole batch without side effects.
	type staged struct {
		canonical []byte
		digest    string
	}
	items := make([]staged, 0, len(batch))
	for _, item := range batch {
		encoded, _, err := canonicalStateValue(item)
		if err != nil {
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		canonical, err := contractCanonical(encoded)
		if err != nil {
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		if err := ValidateWCNCPRecord(route.kind, canonical); err != nil {
			recordWCNCPEvent(request, storage, requestID, tokenID, route, "batch-put", "", "rejected")
			writeCacheStatus(response, http.StatusBadRequest)
			return
		}
		items = append(items, staged{canonical: canonical, digest: digestOf(canonical)})
	}
	batchInputs := make([]WCNCPObjectInput, 0, len(items))
	for _, item := range items {
		batchInputs = append(batchInputs, WCNCPObjectInput{ExpectedSHA256: item.digest, Raw: item.canonical})
	}
	if _, _, err := storage.PutWCNCPObjectBatch(request.Context(), route.repositoryScopeSHA256, route.kind, batchInputs); err != nil {
		writeWCNCPStorageError(response, err)
		return
	}
	recordWCNCPEvent(request, storage, requestID, tokenID, route, "batch-put", strconv.Itoa(len(items)), "ok")
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusCreated)
	_, _ = response.Write([]byte(`{"published":` + strconv.Itoa(len(items)) + `}`))
}

func serveWCNCPLease(storage *Storage, authorization CentralTokenAuthorization, grant WCNCPGrant, tokenID string, route wcncpHTTPRoute, requestID string, forked bool, response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || route.kind != WCNCPKindProposal {
		response.Header().Set("Allow", "POST")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	if err := AuthorizeWCNCPOperation(authorization, grant, WCNCPOpProposalClaim, route.repositoryScopeSHA256, route.kind, forked); err != nil {
		writeWCNCPAuthError(response, err)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, 16<<10)
	var document struct {
		ProposalDigest   string `json:"proposalDigest"`
		ProtocolVersion  string `json:"protocolVersion"`
		EnvironmentClass string `json:"environmentClass"`
		Holder           string `json:"holder"`
		TTLMillis        int64  `json:"ttlMillis"`
		LeaseID          string `json:"leaseId"`
		State            string `json:"state"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeCacheStatus(response, http.StatusBadRequest)
		return
	}
	operation := "lease-" + route.resource
	switch route.resource {
	case "claim":
		lease, err := storage.ClaimWCNCPLease(request.Context(), route.repositoryScopeSHA256, document.ProposalDigest, document.ProtocolVersion, document.EnvironmentClass, document.Holder, time.Duration(document.TTLMillis)*time.Millisecond, storage.now())
		if errors.Is(err, ErrWCNCPLeaseHeld) {
			writeCacheStatus(response, http.StatusConflict)
			return
		}
		if err != nil {
			writeWCNCPLeaseError(response, err)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, operation, lease.LeaseID, "ok")
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(response).Encode(lease)
	case "heartbeat":
		if err := storage.HeartbeatWCNCPLease(request.Context(), document.LeaseID, document.Holder, storage.now()); err != nil {
			writeWCNCPLeaseError(response, err)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, operation, document.LeaseID, "ok")
		writeCacheStatus(response, http.StatusOK)
	case "release":
		if err := storage.ReleaseWCNCPLease(request.Context(), document.LeaseID, document.Holder, document.State); err != nil {
			writeWCNCPLeaseError(response, err)
			return
		}
		recordWCNCPEvent(request, storage, requestID, tokenID, route, operation, document.LeaseID, "ok")
		writeCacheStatus(response, http.StatusOK)
	}
}

func writeWCNCPLeaseError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrWCNCPInvalid):
		writeCacheStatus(response, http.StatusBadRequest)
	case errors.Is(err, ErrWCNCPLeaseHeld):
		writeCacheStatus(response, http.StatusConflict)
	case errors.Is(err, ErrWCNCPLeaseLost):
		writeCacheStatus(response, http.StatusPreconditionFailed)
	default:
		writeWCNCPStorageError(response, err)
	}
}

func serveWCNCPProjection(storage *Storage, authorization CentralTokenAuthorization, route wcncpHTTPRoute, response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", "GET")
		writeCacheStatus(response, http.StatusMethodNotAllowed)
		return
	}
	if authorization.Scope.RepositoryScopeSHA256 != route.repositoryScopeSHA256 || (!authorization.Has(CentralStateRead) && !authorization.Has(CentralStateWrite)) {
		writeCacheStatus(response, http.StatusForbidden)
		return
	}
	state := ""
	for _, kind := range []StateKind{WCNCPKindDecision, WCNCPKindValidation, WCNCPKindProposal, WCNCPKindOpportunity, WCNCPKindObservation} {
		snapshot, err := storage.LoadCurrentWCNCP(request.Context(), route.repositoryScopeSHA256, kind)
		if errors.Is(err, ErrWCNCPNotFound) {
			continue
		}
		if err != nil {
			writeWCNCPStorageError(response, err)
			return
		}
		state, err = projectionState(storage, request, snapshot)
		if err != nil {
			writeWCNCPStorageError(response, err)
			return
		}
		if state != "" {
			break
		}
	}
	if state == "" {
		writeCacheStatus(response, http.StatusNotFound)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(map[string]string{"state": state})
}

func projectionState(storage *Storage, request *http.Request, snapshot WCNCPSnapshot) (string, error) {
	var raw []byte
	for _, artifact := range snapshot.Manifest.Artifacts {
		if artifact.Role != string(snapshot.Manifest.Kind) {
			continue
		}
		file, err := storage.OpenWCNCPObject(request.Context(), snapshot.Manifest.RepositoryScopeSHA256, snapshot.Manifest.Kind, artifact.SHA256)
		if err != nil {
			return "", err
		}
		raw, err = io.ReadAll(io.LimitReader(file, maximumWCNCPArtifactBytes+1))
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			return "", ErrWCNPCorrupt
		}
		break
	}
	if len(raw) == 0 {
		return "", ErrWCNPCorrupt
	}
	switch snapshot.Manifest.Kind {
	case WCNCPKindDecision:
		var record WCNCPDecision
		if err := decodeStrictWCNCPJSON(raw, &record); err != nil {
			return "", ErrWCNPCorrupt
		}
		switch record.Decision {
		case "ACCEPT":
			return "OWNER_ACCEPTED", nil
		case "REJECT":
			return "OWNER_REJECTED", nil
		case "DEFER":
			return "OWNER_DEFERRED", nil
		}
	case WCNCPKindValidation:
		var record WCNCPValidation
		if err := decodeStrictWCNCPJSON(raw, &record); err != nil {
			return "", ErrWCNPCorrupt
		}
		if record.Decision == "QUALIFIED" {
			return "REVIEW_READY", nil
		}
		return "OBSERVING", nil
	case WCNCPKindProposal:
		return "VALIDATION_QUEUED", nil
	case WCNCPKindOpportunity:
		return "OPPORTUNITY_DETECTED", nil
	case WCNCPKindObservation:
		return "OBSERVING", nil
	}
	return "", ErrWCNPCorrupt
}

func writeWCNCPAuthError(response http.ResponseWriter, err error) {
	if errors.Is(err, ErrWCNCPForkReadOnly) {
		writeCacheStatus(response, http.StatusForbidden)
		return
	}
	if errors.Is(err, ErrWCNCPForbidden) {
		writeCacheStatus(response, http.StatusForbidden)
		return
	}
	writeCacheStatus(response, http.StatusBadRequest)
}

func writeWCNCPStorageError(response http.ResponseWriter, err error) {
	if errors.Is(err, ErrWCNPCorrupt) {
		writeCacheStatus(response, http.StatusInternalServerError)
		return
	}
	writeStateStorageError(response, err)
}

func recordWCNCPEvent(request *http.Request, storage *Storage, requestID, tokenID string, route wcncpHTTPRoute, operation, digest, result string) {
	// Audit events carry request IDs and content digests only. Credentials,
	// source content, and raw unsafe arguments never enter logs or the audit
	// table.
	if storage == nil || request == nil {
		return
	}
	ctx := request.Context()
	if ctx == nil {
		return
	}
	_, _ = storage.control.database.ExecContext(ctx, `INSERT INTO wcncp_audit_events (
    occurred_at_unix_ms, request_id, token_id, repository_scope_sha256,
    kind, operation, manifest_digest, result
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		storage.now().UnixMilli(), requestID, tokenID, route.repositoryScopeSHA256,
		string(route.kind), operation, nullIfEmpty(digest), result)
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func strictWCNCPDecoder(reader io.Reader) *json.Decoder {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	return decoder
}

func contractCanonical(raw []byte) ([]byte, error) {
	return contractcrypto.CanonicalizeJCS(raw)
}

func digestOf(canonical []byte) string {
	return digestBytes(canonical)
}
