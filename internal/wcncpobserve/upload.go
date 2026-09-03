package wcncpobserve

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UploadOutcome records whether a post-child batch reached the backend. The
// Gradle result is preserved regardless; an unavailable backend leaves items
// queued for a later invocation.
type UploadOutcome struct {
	Attempted bool
	Uploaded  int
	Queued    bool
	Reason    string
}

// UploadBatch attempts one bounded batch upload after the child completes
// under a strict post-child deadline. It never runs before the child and
// never delays the native result beyond the deadline.
func UploadBatch(ctx context.Context, client *http.Client, url, token string, items []json.RawMessage, timeout time.Duration) UploadOutcome {
	if len(items) == 0 {
		return UploadOutcome{Reason: "empty"}
	}
	if timeout <= 0 {
		timeout = PostChildUploadDeadlineMs * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := json.Marshal(items)
	if err != nil {
		return UploadOutcome{Queued: true, Reason: "encode"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return UploadOutcome{Queued: true, Reason: "request"}
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return UploadOutcome{Queued: true, Reason: "unavailable"}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		// Corrupt or rejecting backends fail closed without losing the local
		// queue: the items stay queued for a later invocation.
		return UploadOutcome{Attempted: true, Queued: true, Reason: "rejected"}
	}
	var acknowledgement struct {
		Published int `json:"published"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&acknowledgement); err != nil || acknowledgement.Published != len(items) {
		return UploadOutcome{Attempted: true, Queued: true, Reason: "invalid-acknowledgement"}
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return UploadOutcome{Attempted: true, Queued: true, Reason: "invalid-acknowledgement"}
	}
	return UploadOutcome{Attempted: true, Uploaded: len(items)}
}

// RepositoryScopeSHA256 derives the opaque route identity from the private
// repository scope stored inside an observation.
func RepositoryScopeSHA256(repositoryScope string) string {
	digest := sha256.Sum256([]byte(repositoryScope))
	return fmt.Sprintf("%x", digest[:])
}

// Endpoint resolves a WCNCP resource below an HTTPS backend origin. Plain
// HTTP is accepted only for an explicit loopback development server.
func Endpoint(baseURL, repositoryScope, resource string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
		return "", ErrObservationInvalid
	}
	hostname := parsed.Hostname()
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && (hostname == "127.0.0.1" || hostname == "::1" || hostname == "localhost")) {
		return "", ErrObservationInvalid
	}
	if repositoryScope == "" || resource == "" || strings.HasPrefix(resource, "/") || strings.Contains(resource, "..") || strings.ContainsAny(resource, "?#") {
		return "", ErrObservationInvalid
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/v1/repositories/" + RepositoryScopeSHA256(repositoryScope) + "/wcncp/" + resource
	return parsed.String(), nil
}

// FetchStatus returns only a TLS-authenticated projection supplied by the
// central state owner. Callers must treat every error as unverified.
func FetchStatus(ctx context.Context, client *http.Client, endpoint, token string, timeout time.Duration) (string, bool) {
	if timeout <= 0 {
		timeout = PostChildUploadDeadlineMs * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", false
	}
	var projection struct {
		State string `json:"state"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&projection); err != nil {
		return "", false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return "", false
	}
	switch projection.State {
	case "OBSERVING", "OPPORTUNITY_DETECTED", "VALIDATION_QUEUED", "VALIDATING", "REVIEW_READY", "OWNER_ACCEPTED", "OWNER_REJECTED", "OWNER_DEFERRED":
		return projection.State, true
	default:
		return "", false
	}
}
