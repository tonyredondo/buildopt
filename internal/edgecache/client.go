package edgecache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	sharedAuthorityDigestHeader = "X-BuildOpt-Authority-Digest"
	sharedCommitStateHeader     = "X-BuildOpt-Commit-State"
	sharedDecisionDigestHeader  = "X-BuildOpt-Decision-Digest"
)

var (
	ErrCacheMiss           = errors.New("Edge cache miss")
	ErrUpstreamRejected    = errors.New("Shared rejected Edge read authority")
	ErrUpstreamUnavailable = errors.New("Shared is unavailable to Edge")
)

type SharedClient struct {
	baseURL    *url.URL
	credential []byte
	httpClient *http.Client
}

type fetchedObject struct {
	body           io.ReadCloser
	size           int64
	digest         string
	decisionDigest string
}

// NewSharedClient creates a redirect-rejecting client for the authenticated
// Shared origin selected by an already validated Edge configuration.
func NewSharedClient(shared Shared, credential []byte, client *http.Client) (*SharedClient, error) {
	if err := validateSharedURL(shared); err != nil {
		return nil, err
	}
	if len(credential) < 1 || len(credential) > 4096 || bytes.ContainsAny(credential, "\r\n\t ") {
		return nil, errors.New("Shared credential is empty or unsafe")
	}
	parsed, err := url.Parse(shared.BaseURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	clone.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &SharedClient{
		baseURL:    parsed,
		credential: bytes.Clone(credential),
		httpClient: &clone,
	}, nil
}

// Close removes the in-memory copy of the remote credential.
func (client *SharedClient) Close() {
	if client == nil {
		return
	}
	clear(client.credential)
	client.credential = nil
}

func (client *SharedClient) fetch(
	ctx context.Context,
	authority ReadAuthority,
	key string,
	now time.Time,
) (fetchedObject, error) {
	if client == nil || client.httpClient == nil || client.baseURL == nil ||
		len(client.credential) == 0 || ctx == nil ||
		!authority.current(now) || !validCacheKey(key) {
		return fetchedObject{}, errors.New("invalid Shared fetch request")
	}
	requestURL := *client.baseURL
	requestURL.Path = "/cache/" + key
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return fetchedObject{}, err
	}
	request.Header.Set("Authorization", "Bearer "+string(client.credential))
	request.Header.Set(sharedAuthorityDigestHeader, authority.authorityDigest)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return fetchedObject{}, fmt.Errorf("%w: %v", ErrUpstreamUnavailable, err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		switch response.StatusCode {
		case http.StatusNotFound:
			return fetchedObject{}, ErrCacheMiss
		case http.StatusUnauthorized, http.StatusForbidden:
			return fetchedObject{}, ErrUpstreamRejected
		default:
			return fetchedObject{}, fmt.Errorf("%w: HTTP %d", ErrUpstreamUnavailable, response.StatusCode)
		}
	}
	if response.ContentLength < 0 || response.Header.Get("Content-Encoding") != "" ||
		!exactHeader(response.Header, sharedCommitStateHeader, "COMMITTED") {
		_ = response.Body.Close()
		return fetchedObject{}, fmt.Errorf("%w: invalid committed response framing", ErrUpstreamUnavailable)
	}
	digest, ok := quotedDigest(response.Header.Values("ETag"))
	if !ok {
		_ = response.Body.Close()
		return fetchedObject{}, fmt.Errorf("%w: invalid committed object digest", ErrUpstreamUnavailable)
	}
	decisions := response.Header.Values(sharedDecisionDigestHeader)
	if len(decisions) != 1 || !validDigest(decisions[0]) {
		_ = response.Body.Close()
		return fetchedObject{}, fmt.Errorf("%w: invalid commit decision digest", ErrUpstreamUnavailable)
	}
	return fetchedObject{
		body:           response.Body,
		size:           response.ContentLength,
		digest:         digest,
		decisionDigest: decisions[0],
	}, nil
}

func exactHeader(header http.Header, name, value string) bool {
	values := header.Values(name)
	return len(values) == 1 && values[0] == value
}

func quotedDigest(values []string) (string, bool) {
	if len(values) != 1 || len(values[0]) != len("sha256:")+64+2 ||
		values[0][0] != '"' || values[0][len(values[0])-1] != '"' {
		return "", false
	}
	digest := strings.TrimSuffix(strings.TrimPrefix(values[0], `"`), `"`)
	return digest, validDigest(digest)
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func validCacheKey(value string) bool {
	if len(value) < 1 || len(value) > 256 {
		return false
	}
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') ||
			(character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			strings.ContainsRune("._-", character) {
			continue
		}
		return false
	}
	return true
}
