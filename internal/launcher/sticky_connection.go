package launcher

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickywrapper"
)

const (
	stickyWrapperRootEnvironment = "BUILDOPT_STICKY_WRAPPER_ROOT"
	stickyConnectionTimeout      = 10 * time.Second
)

type stickyWrapperConnection struct {
	serverURL             string
	projectScope          string
	projectScopeSHA256    string
	connectionScopeSHA256 string
	namespace             string
	namespaceGeneration   int64
	token                 []byte
	http                  *http.Client
}

func (connection *stickyWrapperConnection) close() {
	if connection == nil {
		return
	}
	clear(connection.token)
	if transport, ok := connection.http.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

func prepareStickyWrapperConnection(
	root string,
	childArgs []string,
	getenv func(string) string,
	now time.Time,
	client *http.Client,
) (*stickyWrapperConnection, string, error) {
	credentialEnvironment := stickywrapper.CredentialEnvironment(root)
	if getenv == nil {
		return nil, credentialEnvironment, errors.New("wrapper credential environment is unavailable")
	}
	if err := validateStickyWrapperInvocation(root, childArgs); err != nil {
		return nil, credentialEnvironment, err
	}
	config, err := stickywrapper.LoadConfig(root)
	if err != nil {
		return nil, credentialEnvironment, err
	}
	credentialEnvironment = config.CredentialEnv
	if config.Mode == "off" || config.ServerURL == "" {
		return nil, credentialEnvironment, nil
	}
	rawCredential := strings.TrimSpace(getenv(credentialEnvironment))
	if rawCredential == "" {
		return nil, credentialEnvironment, nil
	}
	document, token, err := parseStickyAccessToken(rawCredential, config, now)
	if err != nil {
		return nil, credentialEnvironment, err
	}
	serverURL, err := canonicalStickyServerURL(config.ServerURL)
	if err != nil {
		clear(token)
		return nil, credentialEnvironment, err
	}
	projectScopeSHA256 := optimizePortfolioRepositoryScope(config.ProjectScope)
	connectionScopeSHA256 := stickyConnectionScopeSHA256(
		serverURL,
		projectScopeSHA256,
		document,
	)
	if client == nil {
		client, err = newStickyConnectionHTTPClient(serverURL)
		if err != nil {
			clear(token)
			return nil, credentialEnvironment, err
		}
	}
	connection := &stickyWrapperConnection{
		serverURL: serverURL, projectScope: config.ProjectScope,
		projectScopeSHA256:    projectScopeSHA256,
		connectionScopeSHA256: connectionScopeSHA256,
		namespace:             document.Namespace, namespaceGeneration: document.NamespaceGeneration,
		token: token, http: client,
	}
	if err := connection.probe(context.Background()); err != nil {
		connection.close()
		return nil, credentialEnvironment, fmt.Errorf("credential-bound connection rejected: %w", err)
	}
	return connection, credentialEnvironment, nil
}

func validateStickyWrapperInvocation(root string, childArgs []string) error {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return errors.New("wrapper root must be one clean absolute path")
	}
	wrapperName := "gradlew"
	if runtime.GOOS == "windows" {
		wrapperName = "gradlew.bat"
	}
	if len(childArgs) == 0 {
		return errors.New("repository Gradle Wrapper command is missing")
	}
	expected := filepath.Join(root, wrapperName)
	actual, err := filepath.Abs(childArgs[0])
	if err != nil {
		return errors.New("repository Gradle Wrapper command is invalid")
	}
	if runtime.GOOS == "windows" {
		if !strings.EqualFold(filepath.Clean(actual), filepath.Clean(expected)) {
			return errors.New("wrapper root does not own the Gradle command")
		}
	} else if filepath.Clean(actual) != filepath.Clean(expected) {
		return errors.New("wrapper root does not own the Gradle command")
	}
	return nil
}

func parseStickyAccessToken(
	raw string,
	config stickywrapper.Config,
	now time.Time,
) (centralIssuedTokenDocument, []byte, error) {
	var document centralIssuedTokenDocument
	if decodeCentralStrictJSON([]byte(raw), &document) != nil ||
		document.SchemaVersion != "buildopt.central/access-token/v1" ||
		document.Repository != config.ProjectScope ||
		document.RepositoryScopeSHA256 != optimizePortfolioRepositoryScope(config.ProjectScope) ||
		document.TokenID == "" || document.Tenant == "" || document.TrustDomain == "" ||
		!validGatewayNamespace(document.Namespace) || document.NamespaceGeneration < 1 {
		return centralIssuedTokenDocument{}, nil, errors.New("wrapper access token document does not match the committed project scope")
	}
	issuedAt, issuedErr := time.Parse(time.RFC3339Nano, document.IssuedAt)
	expiresAt, expiresErr := time.Parse(time.RFC3339Nano, document.ExpiresAt)
	if issuedErr != nil || expiresErr != nil || issuedAt.Location() != time.UTC ||
		expiresAt.Location() != time.UTC || issuedAt.Format(time.RFC3339Nano) != document.IssuedAt ||
		expiresAt.Format(time.RFC3339Nano) != document.ExpiresAt || issuedAt.After(now) ||
		!expiresAt.After(issuedAt) || !now.Before(expiresAt) {
		return centralIssuedTokenDocument{}, nil, errors.New("wrapper access token time binding is invalid or expired")
	}
	if !stickyCapabilitiesCanonical(document.Capabilities) ||
		!centralHasCapability(document.Capabilities, sharedcache.CentralCacheRead) ||
		!centralHasCapability(document.Capabilities, sharedcache.CentralStateRead) {
		return centralIssuedTokenDocument{}, nil, errors.New("wrapper access token requires CACHE_READ and STATE_READ")
	}
	token, err := base64.RawURLEncoding.DecodeString(document.Token)
	if err != nil || len(token) != sharedcache.CentralTokenBytes ||
		base64.RawURLEncoding.EncodeToString(token) != document.Token {
		clear(token)
		return centralIssuedTokenDocument{}, nil, errors.New("wrapper access token credential is invalid")
	}
	return document, token, nil
}

func stickyCapabilitiesCanonical(capabilities []sharedcache.CentralCapability) bool {
	order := map[sharedcache.CentralCapability]int{
		sharedcache.CentralCacheRead: 0, sharedcache.CentralCacheWrite: 1,
		sharedcache.CentralStateRead: 2, sharedcache.CentralStateWrite: 3,
	}
	previous := -1
	for _, capability := range capabilities {
		position, ok := order[capability]
		if !ok || position <= previous {
			return false
		}
		previous = position
	}
	return len(capabilities) > 0
}

func stickyConnectionScopeSHA256(
	serverURL string,
	projectScopeSHA256 string,
	document centralIssuedTokenDocument,
) string {
	return optimizeDigest(
		"buildopt-sticky-connection-scope-v1",
		serverURL,
		projectScopeSHA256,
		document.Tenant,
		document.Repository,
		document.TrustDomain,
		document.Namespace,
		strconv.FormatInt(document.NamespaceGeneration, 10),
	)
}

func newStickyConnectionHTTPClient(serverURL string) (*http.Client, error) {
	if _, err := canonicalStickyServerURL(serverURL); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS13},
		ForceAttemptHTTP2: true, MaxIdleConns: 2, MaxIdleConnsPerHost: 2,
		IdleConnTimeout: 30 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   stickyConnectionTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("wrapper connection redirects are disabled")
		},
	}, nil
}

func canonicalStickyServerURL(serverURL string) (string, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.RawPath != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("wrapper server URL must be one canonical origin")
	}
	switch parsed.Scheme {
	case "https":
	case "http":
		address := net.ParseIP(parsed.Hostname())
		if address == nil || !address.IsLoopback() {
			return "", errors.New("wrapper server URL must use HTTPS outside numeric loopback fixtures")
		}
	default:
		return "", errors.New("wrapper server URL must use HTTPS outside numeric loopback fixtures")
	}
	canonical := parsed.Scheme + "://" + parsed.Host
	if serverURL != canonical && serverURL != canonical+"/" {
		return "", errors.New("wrapper server URL must be one canonical origin")
	}
	return canonical, nil
}

func (connection *stickyWrapperConnection) probe(ctx context.Context) error {
	statePath := centralStatePath(
		connection.projectScopeSHA256,
		sharedcache.StateKindEvidence,
		"head",
		"",
	)
	stateStatus, err := connection.request(ctx, http.MethodHead, statePath, nil)
	if err != nil {
		return err
	}
	if stateStatus != http.StatusMethodNotAllowed && stateStatus != http.StatusNotFound {
		return fmt.Errorf("state-read probe returned HTTP %d", stateStatus)
	}
	probeKey := sha256.Sum256([]byte(connection.connectionScopeSHA256 + "\x00cache-read-probe"))
	cacheStatus, err := connection.request(
		ctx,
		http.MethodGet,
		"/cache/"+hex.EncodeToString(probeKey[:]),
		map[string]string{sharedcache.CentralNamespaceHeader: connection.namespace},
	)
	if err != nil {
		return err
	}
	if cacheStatus != http.StatusNotFound && cacheStatus != http.StatusOK {
		return fmt.Errorf("cache-read probe returned HTTP %d", cacheStatus)
	}
	return nil
}

func (connection *stickyWrapperConnection) request(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
) (int, error) {
	request, err := http.NewRequestWithContext(ctx, method, connection.serverURL+path, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+base64.RawURLEncoding.EncodeToString(connection.token))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := connection.http.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4097))
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return response.StatusCode, fmt.Errorf("server authorization returned HTTP %d", response.StatusCode)
	}
	return response.StatusCode, nil
}
