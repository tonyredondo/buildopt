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
	gatewayUsername         = "buildopt"

	gatewayOperationTimeout = 5 * time.Second
)

type localGateway struct {
	address    string
	endpoint   string
	username   string
	password   string
	generation string

	mutex     sync.Mutex
	listener  net.Listener
	server    *http.Server
	serveDone chan error
	closed    bool
}

func startLocalGateway() (*localGateway, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback gateway rendezvous: %w", err)
	}

	_, password, err := newLocalSecret(32)
	if err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("generate local gateway credential: %w", err)
	}
	generation, err := newPluginAttemptID()
	if err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("generate gateway connection generation: %w", err)
	}

	gateway := &localGateway{
		address:    listener.Addr().String(),
		endpoint:   "http://" + listener.Addr().String(),
		username:   gatewayUsername,
		password:   password,
		generation: generation,
	}
	gateway.startServingLocked(listener)
	if err := gateway.probe(); err != nil {
		_ = gateway.close()
		return nil, fmt.Errorf("verify local gateway readiness: %w", err)
	}
	return gateway, nil
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
	return gateway.stopServingLocked()
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
	if request.URL.Path != gatewayReadyPath || request.URL.RawQuery != "" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set(gatewayGenerationHeader, gateway.generation)
	writer.WriteHeader(http.StatusNoContent)
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
	pluginAttemptIDEnvironment,
	pluginSocketEnvironment,
	pluginTokenEnvironment,
	gatewayURLEnvironment,
	gatewayUsernameEnvironment,
	gatewayPasswordEnvironment,
	gatewayGenerationEnvironment,
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
		}
		result = append(result, entry)
	}
	for _, key := range reservedChildEnvironment {
		if value, ok := overrides[key]; ok {
			result = append(result, key+"="+value)
		}
	}
	return result
}
