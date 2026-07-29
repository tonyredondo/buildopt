package sessioningest

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// ServerURLEnvironment configures the launcher's local server endpoint.
	ServerURLEnvironment = "BUILDOPT_SERVER_URL"
	// ServerTokenEnvironment configures the launcher's ingest bearer token.
	ServerTokenEnvironment = "BUILDOPT_SERVER_INGEST_TOKEN"

	// IngestPath is the provisional WS-005 session ingest endpoint.
	IngestPath = "/internal/v1/build-sessions"
	// SessionIDHeader acknowledges the session accepted by the server.
	SessionIDHeader = "BuildOpt-Session-ID"

	maxRequestBytes = 64 << 10
)

// Client delivers provisional session records to a local buildopt-server.
type Client struct {
	endpoint *url.URL
	token    string
	client   *http.Client
}

// NewClient validates the walking-skeleton endpoint and token configuration.
func NewClient(endpoint string, token string) (*Client, error) {
	parsedEndpoint, err := validateEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	if err := validateToken(token); err != nil {
		return nil, err
	}

	return &Client{
		endpoint: parsedEndpoint,
		token:    token,
		client: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				Proxy:             nil,
				DisableKeepAlives: true,
			},
			CheckRedirect: func(
				_ *http.Request,
				_ []*http.Request,
			) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// ClientFromEnvironment resolves optional local server ingest configuration.
func ClientFromEnvironment(
	getenv func(string) string,
) (*Client, bool, error) {
	endpoint := getenv(ServerURLEnvironment)
	token := getenv(ServerTokenEnvironment)
	if endpoint == "" && token == "" {
		return nil, false, nil
	}
	if endpoint == "" || token == "" {
		return nil, false, errors.New(
			"incomplete buildopt-server ingest configuration",
		)
	}
	client, err := NewClient(endpoint, token)
	if err != nil {
		return nil, false, err
	}
	return client, true, nil
}

// Deliver sends one session record and verifies the server acknowledgement.
func (client *Client) Deliver(
	ctx context.Context,
	record Record,
) (PutResult, error) {
	if err := record.Validate(); err != nil {
		return 0, fmt.Errorf("validate session ingest record: %w", err)
	}
	body, err := json.Marshal(record)
	if err != nil {
		return 0, errors.New("encode session ingest record")
	}

	target := *client.endpoint
	target.Path = IngestPath
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, errors.New("create session ingest request")
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", record.SessionID)

	response, err := client.client.Do(request)
	if err != nil {
		return 0, errors.New("contact buildopt-server session ingest")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))

	var result PutResult
	switch response.StatusCode {
	case http.StatusAccepted:
		result = PutCreated
	case http.StatusNoContent:
		result = PutDuplicate
	default:
		return 0, fmt.Errorf(
			"buildopt-server session ingest returned HTTP %d",
			response.StatusCode,
		)
	}
	if response.Header.Get(SessionIDHeader) != record.SessionID {
		return 0, errors.New(
			"buildopt-server returned a mismatched session acknowledgement",
		)
	}
	return result, nil
}

// Observer receives each accepted or deduplicated session.
type Observer func(Record, PutResult)

// Handler authenticates, validates, and stores WS-005 session ingest requests.
type Handler struct {
	token    string
	store    *Store
	observer Observer
}

// NewHandler creates an ingest handler backed by store.
func NewHandler(
	token string,
	store *Store,
	observer Observer,
) (*Handler, error) {
	if err := validateToken(token); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, errors.New("session ingest store is required")
	}
	return &Handler{token: token, store: store, observer: observer}, nil
}

func (handler *Handler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Cache-Control", "no-store")
	if !handler.authenticate(request) {
		writer.Header().Set(
			"WWW-Authenticate",
			`Bearer realm="buildopt-server-session-ingest"`,
		)
		http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if request.URL.Path != IngestPath || request.URL.RawQuery != "" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		http.Error(
			writer,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}
	if request.Header.Get("Content-Encoding") != "" {
		http.Error(
			writer,
			http.StatusText(http.StatusUnsupportedMediaType),
			http.StatusUnsupportedMediaType,
		)
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(
			writer,
			http.StatusText(http.StatusUnsupportedMediaType),
			http.StatusUnsupportedMediaType,
		)
		return
	}
	if request.ContentLength > maxRequestBytes {
		http.Error(
			writer,
			http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		handler.writeDecodeError(writer, err)
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if request.Header.Get("Idempotency-Key") != record.SessionID ||
		len(request.Header.Values("Idempotency-Key")) != 1 {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	result, err := handler.store.Put(record)
	if errors.Is(err, ErrSessionConflict) {
		http.Error(writer, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	writer.Header().Set(SessionIDHeader, record.SessionID)
	if result == PutDuplicate {
		writer.WriteHeader(http.StatusNoContent)
	} else {
		writer.WriteHeader(http.StatusAccepted)
	}
	if handler.observer != nil {
		handler.observer(record, result)
	}
}

func (handler *Handler) authenticate(request *http.Request) bool {
	values := request.Header.Values("Authorization")
	if len(values) != 1 ||
		!strings.HasPrefix(values[0], "Bearer ") {
		return false
	}
	actual := strings.TrimPrefix(values[0], "Bearer ")
	return subtle.ConstantTimeCompare(
		[]byte(actual),
		[]byte(handler.token),
	) == 1
}

func (handler *Handler) writeDecodeError(
	writer http.ResponseWriter,
	err error,
) {
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		http.Error(
			writer,
			http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge,
		)
		return
	}
	http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("session ingest body must contain one JSON value")
	}
	return nil
}

func validateEndpoint(endpoint string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.New("invalid buildopt-server URL")
	}
	if parsed.Scheme != "http" ||
		parsed.Hostname() != "127.0.0.1" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		parsed.RawPath != "" ||
		!(parsed.Path == "" || parsed.Path == "/") {
		return nil, errors.New(
			"buildopt-server URL must be an HTTP loopback endpoint",
		)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New(
			"buildopt-server URL must include an explicit port",
		)
	}
	if host, _, err := net.SplitHostPort(parsed.Host); err != nil ||
		host != "127.0.0.1" {
		return nil, errors.New(
			"buildopt-server URL must use the canonical loopback address",
		)
	}
	parsed.Path = ""
	return parsed, nil
}

func validateToken(token string) error {
	if len(token) < 32 || len(token) > 512 {
		return errors.New(
			"buildopt-server ingest token must contain 32 to 512 bytes",
		)
	}
	for _, character := range token {
		if unicode.IsControl(character) || unicode.IsSpace(character) {
			return errors.New(
				"buildopt-server ingest token contains whitespace or control characters",
			)
		}
	}
	return nil
}
