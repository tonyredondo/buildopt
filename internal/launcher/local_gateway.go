package launcher

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	gatewayURLEnvironment        = "BUILDOPT_GATEWAY_URL"
	gatewayUsernameEnvironment   = "BUILDOPT_GATEWAY_USERNAME"
	gatewayPasswordEnvironment   = "BUILDOPT_GATEWAY_PASSWORD"
	gatewayGenerationEnvironment = "BUILDOPT_GATEWAY_CONNECTION_GENERATION"

	gatewayReadyPath        = "/_buildopt/ready"
	gatewayGenerationHeader = "BuildOpt-Gateway-Connection-Generation"
	gatewayAuthorityHeader  = "X-BuildOpt-Authority-Digest"
	gatewayUsername         = "buildopt"

	gatewayOperationTimeout = 5 * time.Second
)

var gatewayCacheKeyPattern = regexp.MustCompile(
	`^[A-Za-z0-9._-]{1,256}$`,
)

type localGateway struct {
	address     string
	endpoint    string
	username    string
	password    string
	generation  string
	spool       *gatewaySpool
	readiness   func() bool
	cache       func() *gatewayCacheBinding
	openCircuit func(gatewayCircuitReason)
	cacheClient *http.Client
	release     func() error

	cacheSuppressed bool

	mutex     sync.Mutex
	listener  net.Listener
	server    *http.Server
	serveDone chan error
	closed    bool
}

func startLocalGateway() (*localGateway, error) {
	return startLocalGatewayWithCache(nil)
}

func startLocalGatewayWithCache(
	binding *gatewayCacheBinding,
) (*localGateway, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback gateway rendezvous: %w", err)
	}

	identity, err := newGatewayIdentity(listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	spool, err := newEphemeralGatewaySpool()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	gateway := localGatewayForListener(listener, identity, nil)
	gateway.spool = spool
	if binding != nil {
		staticBinding := binding.copy()
		gateway.cache = func() *gatewayCacheBinding {
			return staticBinding.copy()
		}
	}
	gateway.startServingLocked(listener)
	if err := gateway.probe(); err != nil {
		_ = gateway.close()
		return nil, fmt.Errorf("verify local gateway readiness: %w", err)
	}
	return gateway, nil
}

type gatewayCacheBinding struct {
	upstreamEndpoint string
	credential       string
	authorityDigest  string
	attemptID        string
	allowRead        bool
	allowWrite       bool
	expiresAt        time.Time
}

func newGatewayCacheBinding(
	upstreamEndpoint string,
	credential []byte,
	authorityDigest string,
	attemptID string,
	allowRead bool,
	allowWrite bool,
	expiresAt time.Time,
) (*gatewayCacheBinding, error) {
	endpoint, err := validateGatewayUpstreamEndpoint(upstreamEndpoint)
	if err != nil {
		return nil, err
	}
	if len(credential) != 32 {
		return nil, errors.New("gateway upstream credential must contain 32 bytes")
	}
	if !validGatewayAuthorityDigest(authorityDigest) {
		return nil, errors.New("gateway authority digest is invalid")
	}
	if !validPluginAttemptID(attemptID) {
		return nil, errors.New("gateway attempt ID is invalid")
	}
	if !allowRead && !allowWrite {
		return nil, errors.New("gateway cache binding grants no operation")
	}
	if expiresAt.IsZero() {
		return nil, errors.New("gateway cache binding has no expiration")
	}
	return &gatewayCacheBinding{
		upstreamEndpoint: endpoint,
		credential:       base64.RawURLEncoding.EncodeToString(credential),
		authorityDigest:  authorityDigest,
		attemptID:        attemptID,
		allowRead:        allowRead,
		allowWrite:       allowWrite,
		expiresAt:        expiresAt.UTC(),
	}, nil
}

func (binding *gatewayCacheBinding) copy() *gatewayCacheBinding {
	if binding == nil {
		return nil
	}
	result := *binding
	return &result
}

type gatewayIdentity struct {
	address    string
	username   string
	password   string
	generation string
}

func newGatewayIdentity(address string) (gatewayIdentity, error) {
	_, password, err := newLocalSecret(32)
	if err != nil {
		return gatewayIdentity{}, fmt.Errorf(
			"generate local gateway credential: %w",
			err,
		)
	}
	generation, err := newPluginAttemptID()
	if err != nil {
		return gatewayIdentity{}, fmt.Errorf(
			"generate gateway connection generation: %w",
			err,
		)
	}
	return gatewayIdentity{
		address:    address,
		username:   gatewayUsername,
		password:   password,
		generation: generation,
	}, nil
}

func localGatewayForListener(
	listener net.Listener,
	identity gatewayIdentity,
	readiness func() bool,
) *localGateway {
	return &localGateway{
		address:     listener.Addr().String(),
		endpoint:    "http://" + listener.Addr().String(),
		username:    identity.username,
		password:    identity.password,
		generation:  identity.generation,
		readiness:   readiness,
		cacheClient: newGatewayCacheClient(),
	}
}

func newGatewayCacheClient() *http.Client {
	return &http.Client{
		Timeout: gatewayOperationTimeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DisableCompression:    true,
			MaxIdleConns:          32,
			MaxIdleConnsPerHost:   32,
			MaxConnsPerHost:       32,
			IdleConnTimeout:       15 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
		CheckRedirect: func(
			_ *http.Request,
			_ []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}
}

func (gateway *localGateway) childEnvironment(
	environment []string,
) []string {
	return replaceEnvironment(
		environment,
		map[string]string{
			gatewayURLEnvironment:        gateway.endpoint,
			gatewayUsernameEnvironment:   gateway.username,
			gatewayPasswordEnvironment:   gateway.password,
			gatewayGenerationEnvironment: gateway.generation,
		},
	)
}

func (gateway *localGateway) restart() error {
	gateway.mutex.Lock()
	defer gateway.mutex.Unlock()

	if gateway.closed {
		return errors.New("restart closed local gateway")
	}
	if gateway.release != nil {
		return errors.New("restart is owned by the managed gateway process")
	}
	if err := gateway.stopServingLocked(); err != nil {
		return fmt.Errorf("stop local gateway for restart: %w", err)
	}

	listener, err := net.Listen("tcp4", gateway.address)
	if err != nil {
		return fmt.Errorf("restore loopback gateway rendezvous: %w", err)
	}
	gateway.startServingLocked(listener)
	if err := gateway.probe(); err != nil {
		_ = gateway.stopServingLocked()
		return fmt.Errorf("verify restarted local gateway: %w", err)
	}
	return nil
}

func (gateway *localGateway) close() error {
	gateway.mutex.Lock()
	defer gateway.mutex.Unlock()

	if gateway.closed {
		return nil
	}
	gateway.closed = true
	if gateway.release != nil {
		return gateway.release()
	}
	gateway.cacheClient.CloseIdleConnections()
	return errors.Join(
		gateway.stopServingLocked(),
		gateway.spool.close(),
	)
}

func (gateway *localGateway) startServingLocked(listener net.Listener) {
	server := &http.Server{
		Handler:           gateway,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       gatewayOperationTimeout,
		WriteTimeout:      gatewayOperationTimeout,
		IdleTimeout:       10 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	serveDone := make(chan error, 1)
	gateway.listener = listener
	gateway.server = server
	gateway.serveDone = serveDone

	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveDone <- err
	}()
}

func (gateway *localGateway) stopServingLocked() error {
	if gateway.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		gatewayOperationTimeout,
	)
	defer cancel()

	shutdownErr := gateway.server.Shutdown(ctx)
	if shutdownErr != nil {
		_ = gateway.server.Close()
	}
	serveErr := <-gateway.serveDone
	gateway.listener = nil
	gateway.server = nil
	gateway.serveDone = nil
	return errors.Join(shutdownErr, serveErr)
}

func (gateway *localGateway) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Cache-Control", "no-store")

	username, password, authenticated := request.BasicAuth()
	if !authenticated ||
		!constantTimeEqual(username, gateway.username) ||
		!constantTimeEqual(password, gateway.password) {
		writer.Header().Set(
			"WWW-Authenticate",
			`Basic realm="buildopt-local-gateway"`,
		)
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	if request.URL.Path == gatewayReadyPath && request.URL.RawQuery == "" {
		gateway.serveReadiness(writer, request)
		return
	}
	if request.URL.RawQuery != "" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	key, valid := strings.CutPrefix(request.URL.Path, "/cache/")
	if !valid || !gatewayCacheKeyPattern.MatchString(key) || gateway.cache == nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	binding := gateway.cache()
	if binding == nil || !binding.expiresAt.After(time.Now().UTC()) {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	gateway.serveCache(writer, request, binding)
}

func (gateway *localGateway) serveReadiness(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if gateway.readiness != nil && !gateway.readiness() {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	writer.Header().Set(gatewayGenerationHeader, gateway.generation)
	writer.WriteHeader(http.StatusNoContent)
}

func (gateway *localGateway) serveCache(
	writer http.ResponseWriter,
	request *http.Request,
	binding *gatewayCacheBinding,
) {
	switch request.Method {
	case http.MethodGet:
		if !binding.allowRead {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
	case http.MethodPut:
		if !binding.allowWrite {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	default:
		writer.Header().Set("Allow", "GET, PUT")
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	upstreamRequest, err := http.NewRequestWithContext(
		request.Context(),
		request.Method,
		binding.upstreamEndpoint+request.URL.Path,
		request.Body,
	)
	if err != nil {
		gateway.writeUpstreamFailure(writer, request.Method)
		return
	}
	upstreamRequest.ContentLength = request.ContentLength
	for _, header := range []string{"Content-Type", "Expect"} {
		if value := request.Header.Get(header); value != "" {
			upstreamRequest.Header.Set(header, value)
		}
	}
	upstreamRequest.Header.Set(
		"Authorization",
		"Bearer "+binding.credential,
	)
	upstreamRequest.Header.Set(
		gatewayAuthorityHeader,
		binding.authorityDigest,
	)

	response, err := gateway.cacheClient.Do(upstreamRequest)
	if err != nil {
		gateway.writeUpstreamFailure(writer, request.Method)
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		gateway.writeUpstreamFailure(writer, request.Method)
		return
	}
	if request.Method == http.MethodGet && response.StatusCode != http.StatusOK {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	if request.Method == http.MethodPut &&
		(response.StatusCode < 200 || response.StatusCode >= 300) &&
		response.StatusCode != http.StatusRequestEntityTooLarge {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if request.Method == http.MethodPut &&
		response.StatusCode == http.StatusRequestEntityTooLarge {
		gateway.tripCircuit(gatewayCircuitObjectTooLarge)
	}
	if request.Method == http.MethodGet {
		gateway.serveVerifiedCacheGET(writer, response)
		return
	}

	writer.Header().Set("Content-Length", "0")
	if value := response.Header.Get("X-BuildOpt-Blob-Digest"); value != "" {
		writer.Header().Set("X-BuildOpt-Blob-Digest", value)
	}
	writer.WriteHeader(response.StatusCode)
}

func (gateway *localGateway) serveVerifiedCacheGET(
	writer http.ResponseWriter,
	response *http.Response,
) {
	if gateway.spool == nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	payload, err := gateway.spool.receive(
		response.Request.Context(),
		response.Body,
		response.ContentLength,
		response.Header.Get("ETag"),
		response.Header.Get("X-BuildOpt-Blob-Digest"),
	)
	if err != nil {
		switch {
		case errors.Is(err, errGatewaySpoolFlood):
			gateway.tripCircuit(gatewayCircuitFlood)
		case errors.Is(err, errGatewaySpoolObjectTooLarge):
			gateway.tripCircuit(gatewayCircuitObjectTooLarge)
		case errors.Is(err, errGatewaySpoolDiskPressure):
			gateway.tripCircuit(gatewayCircuitDiskPressure)
		}
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	defer payload.close()
	writer.Header().Set("Content-Length", strconv.FormatInt(payload.size, 10))
	writer.Header().Set("ETag", `"`+payload.digest+`"`)
	writer.Header().Set("X-BuildOpt-Blob-Digest", payload.digest)
	if value := response.Header.Get("Content-Type"); value != "" {
		writer.Header().Set("Content-Type", value)
	}
	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, payload.file)
}

func (gateway *localGateway) tripCircuit(reason gatewayCircuitReason) {
	if gateway.openCircuit != nil {
		gateway.openCircuit(reason)
	}
}

func (gateway *localGateway) writeUpstreamFailure(
	writer http.ResponseWriter,
	method string,
) {
	if method == http.MethodGet {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	writer.WriteHeader(http.StatusServiceUnavailable)
}

func validateGatewayUpstreamEndpoint(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" ||
		parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", errors.New(
			"gateway upstream endpoint is not canonical HTTP(S)",
		)
	}
	loopbackHTTP := parsed.Scheme == "http" &&
		parsed.Hostname() == "127.0.0.1"
	remoteTLS := parsed.Scheme == "https"
	if !loopbackHTTP && !remoteTLS {
		return "", errors.New(
			"gateway upstream endpoint requires TLS outside loopback",
		)
	}
	port := parsed.Port()
	if port != "" {
		numericPort, portErr := strconv.Atoi(port)
		if portErr != nil || numericPort < 1 || numericPort > 65535 {
			return "", errors.New("gateway upstream endpoint has an invalid port")
		}
	}
	host := parsed.Hostname()
	if port != "" {
		host = net.JoinHostPort(host, port)
	}
	canonical := parsed.Scheme + "://" + host
	if value != canonical && value != canonical+"/" {
		return "", errors.New("gateway upstream endpoint is not canonical")
	}
	return canonical, nil
}

func validGatewayAuthorityDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if !(character >= '0' && character <= '9') &&
			!(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func (gateway *localGateway) probe() error {
	client := &http.Client{
		Timeout: gatewayOperationTimeout,
		Transport: &http.Transport{
			Proxy:             nil,
			DisableKeepAlives: true,
		},
	}
	request, err := http.NewRequest(
		http.MethodGet,
		gateway.endpoint+gatewayReadyPath,
		nil,
	)
	if err != nil {
		return err
	}
	request.SetBasicAuth(gateway.username, gateway.password)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf(
			"readiness returned HTTP %d",
			response.StatusCode,
		)
	}
	if generation := response.Header.Get(gatewayGenerationHeader); generation != gateway.generation {
		return errors.New("readiness returned a mismatched connection generation")
	}
	return nil
}

func constantTimeEqual(actual string, expected string) bool {
	return subtle.ConstantTimeCompare(
		[]byte(actual),
		[]byte(expected),
	) == 1
}

func newLocalSecret(size int) ([]byte, string, error) {
	if size <= 0 {
		return nil, "", errors.New("local secret size must be positive")
	}
	secret := make([]byte, size)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}
	return secret, base64.RawURLEncoding.EncodeToString(secret), nil
}

var reservedChildEnvironment = []string{
	bypassEnvironment,
	pluginAttemptIDEnvironment,
	pluginSocketEnvironment,
	pluginTokenEnvironment,
	gatewayURLEnvironment,
	gatewayUsernameEnvironment,
	gatewayPasswordEnvironment,
	gatewayGenerationEnvironment,
	managedGatewayStateRootEnvironment,
	managedRunnerSlotEnvironment,
	managedGatewayIdleTimeoutEnvironment,
	managedL1StateRootEnvironment,
	managedL1TenantEnvironment,
	managedL1RepositoryEnvironment,
	managedL1TrustDomainEnvironment,
	managedL1CompatibilityEnvironment,
	managedL1GenerationEnvironment,
	managedL1L2WriterEnvironment,
	managedL1DirectoryChildEnvironment,
	managedL1ModeChildEnvironment,
	managedL1GenerationChildEnvironment,
	managedL1RetentionChildEnvironment,
	localAuthorityPathEnvironment,
	localTrustRootPathEnvironment,
	localCredentialPathEnvironment,
	sharedCacheTokenPathEnvironment,
	sharedCacheURLEnvironment,
	managedSharedModeEnvironment,
	managedAuthorityDigestEnvironment,
	managedPolicyDigestEnvironment,
	managedConfigurationDigestEnvironment,
	managedAuthorityContractEnvironment,
	gradleBootstrapConfigPathEnvironment,
	serverURLEnvironment,
	serverTokenEnvironment,
	exportContextEnvironment,
	gradleCheckstyleHeapEnvironment,
	gradleStandardJarCacheEnvironment,
	resourceAvailableMemoryEnvironment,
}

func replaceEnvironment(
	environment []string,
	overrides map[string]string,
) []string {
	reserved := make(map[string]struct{}, len(reservedChildEnvironment))
	for _, key := range reservedChildEnvironment {
		reserved[key] = struct{}{}
	}

	result := make([]string, 0, len(environment)+len(overrides))
	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found {
			if _, remove := reserved[key]; remove {
				continue
			}
			if _, replace := overrides[key]; replace {
				continue
			}
		}
		result = append(result, entry)
	}
	appended := make(map[string]struct{}, len(overrides))
	for _, key := range reservedChildEnvironment {
		if value, ok := overrides[key]; ok {
			result = append(result, key+"="+value)
			appended[key] = struct{}{}
		}
	}
	extraKeys := make([]string, 0, len(overrides)-len(appended))
	for key := range overrides {
		if _, ok := appended[key]; !ok {
			extraKeys = append(extraKeys, key)
		}
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		result = append(result, key+"="+overrides[key])
	}
	return result
}
