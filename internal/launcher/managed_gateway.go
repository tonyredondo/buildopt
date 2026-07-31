package launcher

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	managedGatewayStateRootEnvironment   = "BUILDOPT_GATEWAY_STATE_ROOT"
	managedRunnerSlotEnvironment         = "BUILDOPT_RUNNER_SLOT"
	managedGatewayIdleTimeoutEnvironment = "BUILDOPT_GATEWAY_IDLE_TIMEOUT"

	managedGatewayInternalCommand = "__managed-gateway"

	managedGatewayStateSchemaVersion   = 1
	managedGatewayControlSchemaVersion = 1
	managedGatewayDefaultIdleTimeout   = 5 * time.Minute
	managedGatewayMinimumIdleTimeout   = 100 * time.Millisecond
	managedGatewayMaximumIdleTimeout   = 24 * time.Hour
	managedGatewayStartupTimeout       = 5 * time.Second
	managedGatewayControlTimeout       = 5 * time.Second
	managedGatewayMaximumControlBytes  = 16 << 10
	managedGatewayMaximumStateBytes    = 4 << 10
)

var errManagedRunnerSlotBusy = errors.New("managed runner slot is already active")

type managedGatewayConfig struct {
	stateRoot   string
	directory   string
	idleTimeout time.Duration
}

type managedGatewayState struct {
	SchemaVersion int    `json:"schemaVersion"`
	Address       string `json:"address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Generation    string `json:"gatewayConnectionGeneration"`
}

type managedGatewayControlRequest struct {
	SchemaVersion int                              `json:"schemaVersion"`
	Operation     string                           `json:"operation"`
	AttemptID     string                           `json:"attemptId"`
	Cache         *managedGatewayCacheRegistration `json:"cache,omitempty"`
}

type managedGatewayCacheRegistration struct {
	UpstreamEndpoint string `json:"upstreamEndpoint"`
	Credential       string `json:"credential"`
	AuthorityDigest  string `json:"authorityDigest"`
	AllowRead        bool   `json:"allowRead"`
	AllowWrite       bool   `json:"allowWrite"`
	ExpiresAt        string `json:"expiresAt"`
}

type managedGatewayControlResponse struct {
	SchemaVersion   int    `json:"schemaVersion"`
	Endpoint        string `json:"endpoint,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	Generation      string `json:"gatewayConnectionGeneration,omitempty"`
	CacheSuppressed bool   `json:"cacheSuppressed,omitempty"`
	Error           string `json:"error,omitempty"`
}

type managedGatewayContext struct {
	mutex        sync.Mutex
	attemptID    string
	cacheBinding *gatewayCacheBinding
	circuit      *managedGatewayCircuitBreaker
}

// managedGatewayConfigFromEnvironment keeps the Phase 0 gateway as the
// compatibility path unless the internal pilot supplies its complete slot
// configuration.
func managedGatewayConfigFromEnvironment(
	getenv func(string) string,
) (managedGatewayConfig, bool, error) {
	stateRoot := getenv(managedGatewayStateRootEnvironment)
	slot := getenv(managedRunnerSlotEnvironment)
	configuredIdleTimeout := getenv(managedGatewayIdleTimeoutEnvironment)
	if stateRoot == "" && slot == "" && configuredIdleTimeout == "" {
		return managedGatewayConfig{}, false, nil
	}
	if stateRoot == "" || slot == "" {
		return managedGatewayConfig{}, true, errors.New(
			"managed gateway requires both BUILDOPT_GATEWAY_STATE_ROOT and BUILDOPT_RUNNER_SLOT",
		)
	}
	if !filepath.IsAbs(stateRoot) {
		return managedGatewayConfig{}, true, errors.New(
			"BUILDOPT_GATEWAY_STATE_ROOT must be an absolute path",
		)
	}
	if err := validateManagedRunnerSlot(slot); err != nil {
		return managedGatewayConfig{}, true, err
	}

	idleTimeout := managedGatewayDefaultIdleTimeout
	if configuredIdleTimeout != "" {
		parsed, err := time.ParseDuration(configuredIdleTimeout)
		if err != nil ||
			parsed < managedGatewayMinimumIdleTimeout ||
			parsed > managedGatewayMaximumIdleTimeout {
			return managedGatewayConfig{}, true, fmt.Errorf(
				"BUILDOPT_GATEWAY_IDLE_TIMEOUT must be between %s and %s",
				managedGatewayMinimumIdleTimeout,
				managedGatewayMaximumIdleTimeout,
			)
		}
		idleTimeout = parsed
	}

	stateRoot = filepath.Clean(stateRoot)
	slotDigest := sha256.Sum256([]byte(slot))
	slotDirectory := filepath.Join(
		stateRoot,
		"slots",
		hex.EncodeToString(slotDigest[:16]),
	)
	return managedGatewayConfig{
		stateRoot:   stateRoot,
		directory:   slotDirectory,
		idleTimeout: idleTimeout,
	}, true, nil
}

func validateManagedRunnerSlot(slot string) error {
	if len(slot) == 0 || len(slot) > 128 {
		return errors.New(
			"BUILDOPT_RUNNER_SLOT must contain between 1 and 128 characters",
		)
	}
	for index, character := range []byte(slot) {
		valid := character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			index > 0 && (character == '.' || character == '_' || character == '-')
		if !valid {
			return errors.New(
				"BUILDOPT_RUNNER_SLOT contains an unsupported character",
			)
		}
	}
	return nil
}

func startInvocationGateway(attemptID string) (*localGateway, error) {
	return startInvocationGatewayWithCache(attemptID, nil)
}

func startInvocationGatewayWithCache(
	attemptID string,
	cacheBinding *gatewayCacheBinding,
) (*localGateway, error) {
	if cacheBinding != nil && cacheBinding.attemptID != attemptID {
		return nil, errors.New(
			"gateway cache binding does not match the invocation attempt",
		)
	}
	config, configured, err := managedGatewayConfigFromEnvironment(os.Getenv)
	if err != nil {
		return nil, err
	}
	if !configured {
		return startLocalGatewayWithCache(cacheBinding)
	}
	return startManagedInvocationGatewayWithCache(
		config,
		attemptID,
		cacheBinding,
	)
}

func startManagedInvocationGateway(
	config managedGatewayConfig,
	attemptID string,
) (*localGateway, error) {
	return startManagedInvocationGatewayWithCache(config, attemptID, nil)
}

func startManagedInvocationGatewayWithCache(
	config managedGatewayConfig,
	attemptID string,
	cacheBinding *gatewayCacheBinding,
) (*localGateway, error) {
	if cacheBinding != nil && cacheBinding.attemptID != attemptID {
		return nil, errors.New(
			"managed gateway cache binding does not match the invocation attempt",
		)
	}
	if err := prepareManagedGatewayDirectories(config); err != nil {
		return nil, err
	}

	// The launcher holds this lease for the complete child lifetime. The
	// detached gateway owns a separate process lease below.
	invocationLock, err := openPrivateLock(
		filepath.Join(config.directory, "invocation.lock"),
	)
	if err != nil {
		return nil, fmt.Errorf("open managed runner-slot lease: %w", err)
	}
	if err := syscall.Flock(
		int(invocationLock.Fd()),
		syscall.LOCK_EX|syscall.LOCK_NB,
	); err != nil {
		_ = invocationLock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) ||
			errors.Is(err, syscall.EAGAIN) {
			return nil, errManagedRunnerSlotBusy
		}
		return nil, fmt.Errorf("acquire managed runner-slot lease: %w", err)
	}

	connection, response, err := registerManagedGateway(
		config,
		attemptID,
		cacheBinding,
		false,
	)
	if err != nil {
		if startErr := startManagedGatewayProcess(config); startErr != nil {
			_ = releaseManagedLock(invocationLock)
			return nil, startErr
		}
		connection, response, err = registerManagedGateway(
			config,
			attemptID,
			cacheBinding,
			true,
		)
	}
	if err != nil {
		_ = releaseManagedLock(invocationLock)
		return nil, fmt.Errorf("register managed gateway invocation: %w", err)
	}

	gateway := &localGateway{
		address:         strings.TrimPrefix(response.Endpoint, "http://"),
		endpoint:        response.Endpoint,
		username:        response.Username,
		password:        response.Password,
		generation:      response.Generation,
		cacheSuppressed: response.CacheSuppressed,
		release: func() error {
			return errors.Join(
				connection.Close(),
				releaseManagedLock(invocationLock),
			)
		},
	}
	if err := gateway.probe(); err != nil {
		_ = gateway.close()
		return nil, fmt.Errorf("verify managed gateway readiness: %w", err)
	}
	return gateway, nil
}

func prepareManagedGatewayDirectories(config managedGatewayConfig) error {
	for _, directory := range []string{
		config.stateRoot,
		filepath.Join(config.stateRoot, "slots"),
		config.directory,
	} {
		if err := ensurePrivateDirectory(directory, true); err != nil {
			return fmt.Errorf("prepare managed gateway state: %w", err)
		}
	}
	return nil
}

func ensurePrivateDirectory(path string, create bool) error {
	if create {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a private directory", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("%s is not owned by the current user", path)
	}
	if info.Mode().Perm() != 0o700 {
		return fmt.Errorf("%s must have mode 0700", path)
	}
	return nil
}

func openPrivateLock(path string) (*os.File, error) {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW,
		0o600,
	)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 {
		_ = file.Close()
		return nil, errors.New("managed gateway lock is not a private regular file")
	}
	return file, nil
}

func releaseManagedLock(file *os.File) error {
	if file == nil {
		return nil
	}
	return errors.Join(
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN),
		file.Close(),
	)
}

func startManagedGatewayProcess(config managedGatewayConfig) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve buildopt executable: %w", err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open managed gateway output sink: %w", err)
	}
	defer null.Close()

	command := exec.Command(
		executable,
		managedGatewayInternalCommand,
		config.directory,
		config.idleTimeout.String(),
	)
	command.Stdin = null
	command.Stdout = null
	command.Stderr = null
	command.Env = []string{}
	command.Dir = string(filepath.Separator)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start managed gateway process: %w", err)
	}
	if err := command.Process.Release(); err != nil {
		_ = command.Process.Kill()
		return fmt.Errorf("release managed gateway process: %w", err)
	}
	return nil
}

func registerManagedGateway(
	config managedGatewayConfig,
	attemptID string,
	cacheBinding *gatewayCacheBinding,
	retry bool,
) (*net.UnixConn, managedGatewayControlResponse, error) {
	deadline := time.Now()
	if retry {
		deadline = deadline.Add(managedGatewayStartupTimeout)
	}
	var lastErr error
	for {
		connection, response, err := requestManagedGatewayRegistration(
			managedGatewayControlAddress(config.directory),
			attemptID,
			cacheBinding,
		)
		if err == nil {
			return connection, response, nil
		}
		lastErr = err
		if !retry || time.Now().After(deadline) {
			return nil, managedGatewayControlResponse{}, lastErr
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func requestManagedGatewayRegistration(
	socketPath string,
	attemptID string,
	cacheBinding *gatewayCacheBinding,
) (*net.UnixConn, managedGatewayControlResponse, error) {
	connection, err := net.DialUnix(
		"unix",
		nil,
		&net.UnixAddr{Name: socketPath, Net: "unix"},
	)
	if err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = connection.Close()
		}
	}()
	if err := verifyUnixPeerOwner(connection); err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	if err := connection.SetDeadline(
		time.Now().Add(managedGatewayControlTimeout),
	); err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	request := managedGatewayControlRequest{
		SchemaVersion: managedGatewayControlSchemaVersion,
		Operation:     "register",
		AttemptID:     attemptID,
		Cache:         managedGatewayCacheForRegistration(cacheBinding),
	}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	line, err := readManagedGatewayLine(bufio.NewReaderSize(
		connection,
		managedGatewayMaximumControlBytes+1,
	))
	if err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	var response managedGatewayControlResponse
	if err := decodeStrictJSON(line, &response); err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	if response.Error != "" {
		return nil, managedGatewayControlResponse{}, errors.New(response.Error)
	}
	if err := validateManagedGatewayControlResponse(response); err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		return nil, managedGatewayControlResponse{}, err
	}
	closeOnError = false
	return connection, response, nil
}

func validateManagedGatewayControlResponse(
	response managedGatewayControlResponse,
) error {
	if response.SchemaVersion != managedGatewayControlSchemaVersion ||
		response.Username != gatewayUsername ||
		response.Error != "" {
		return errors.New("managed gateway returned an invalid registration")
	}
	address, ok := strings.CutPrefix(response.Endpoint, "http://")
	if !ok || validateManagedGatewayAddress(address) != nil {
		return errors.New("managed gateway returned an invalid endpoint")
	}
	if err := validateManagedGatewayCredential(response.Password); err != nil {
		return errors.New("managed gateway returned an invalid credential")
	}
	if !validPluginAttemptID(response.Generation) {
		return errors.New("managed gateway returned an invalid connection generation")
	}
	return nil
}

func managedGatewayCacheForRegistration(
	binding *gatewayCacheBinding,
) *managedGatewayCacheRegistration {
	if binding == nil {
		return nil
	}
	return &managedGatewayCacheRegistration{
		UpstreamEndpoint: binding.upstreamEndpoint,
		Credential:       binding.credential,
		AuthorityDigest:  binding.authorityDigest,
		AllowRead:        binding.allowRead,
		AllowWrite:       binding.allowWrite,
		ExpiresAt: binding.expiresAt.UTC().Format(
			time.RFC3339Nano,
		),
	}
}

func (registration *managedGatewayCacheRegistration) binding(
	attemptID string,
) (*gatewayCacheBinding, error) {
	if registration == nil {
		return nil, nil
	}
	credential, err := base64.RawURLEncoding.DecodeString(
		registration.Credential,
	)
	if err != nil ||
		base64.RawURLEncoding.EncodeToString(credential) !=
			registration.Credential {
		return nil, errors.New("managed gateway cache credential is invalid")
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, registration.ExpiresAt)
	if err != nil ||
		expiresAt.Location() != time.UTC ||
		expiresAt.Format(time.RFC3339Nano) != registration.ExpiresAt {
		return nil, errors.New("managed gateway cache expiration is invalid")
	}
	return newGatewayCacheBinding(
		registration.UpstreamEndpoint,
		credential,
		registration.AuthorityDigest,
		attemptID,
		registration.AllowRead,
		registration.AllowWrite,
		expiresAt,
	)
}

func managedGatewayControlAddress(directory string) string {
	// Linux abstract sockets avoid filesystem path limits. SO_PEERCRED, the
	// private state directory, and the two slot leases provide the boundary.
	digest := sha256.Sum256([]byte(directory))
	return "@buildopt-gateway-" + hex.EncodeToString(digest[:16])
}

func runManagedGatewayProcess(args []string, stderr io.Writer) int {
	if len(args) != 3 || args[0] != managedGatewayInternalCommand {
		_, _ = fmt.Fprintln(stderr, "buildopt: invalid managed gateway invocation")
		return exitUsage
	}
	directory := filepath.Clean(args[1])
	if !filepath.IsAbs(directory) {
		_, _ = fmt.Fprintln(stderr, "buildopt: invalid managed gateway directory")
		return exitUsage
	}
	idleTimeout, err := time.ParseDuration(args[2])
	if err != nil ||
		idleTimeout < managedGatewayMinimumIdleTimeout ||
		idleTimeout > managedGatewayMaximumIdleTimeout {
		_, _ = fmt.Fprintln(stderr, "buildopt: invalid managed gateway idle timeout")
		return exitUsage
	}
	if err := serveManagedGateway(directory, idleTimeout); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: managed gateway unavailable: %v\n", err)
		return 70
	}
	return 0
}

func serveManagedGateway(directory string, idleTimeout time.Duration) error {
	if err := ensurePrivateDirectory(directory, false); err != nil {
		return fmt.Errorf("validate managed gateway directory: %w", err)
	}
	// This lease elects exactly one detached gateway even if two launchers race
	// while recovering an unavailable control channel.
	gatewayLock, err := openPrivateLock(filepath.Join(directory, "gateway.lock"))
	if err != nil {
		return fmt.Errorf("open managed gateway ownership lease: %w", err)
	}
	defer gatewayLock.Close()
	if err := syscall.Flock(
		int(gatewayLock.Fd()),
		syscall.LOCK_EX|syscall.LOCK_NB,
	); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) ||
			errors.Is(err, syscall.EAGAIN) {
			return nil
		}
		return fmt.Errorf("acquire managed gateway ownership lease: %w", err)
	}
	defer syscall.Flock(int(gatewayLock.Fd()), syscall.LOCK_UN)

	httpListener, identity, stateChanged, err :=
		openManagedGatewayListener(directory)
	if err != nil {
		return err
	}
	if stateChanged {
		if err := writeManagedGatewayState(directory, identity); err != nil {
			_ = httpListener.Close()
			return err
		}
	}

	circuit := newManagedGatewayCircuitBreaker(directory)
	context := &managedGatewayContext{circuit: circuit}
	spool, err := openGatewaySpool(
		filepath.Join(directory, "spool"),
		false,
		gatewayMaximumVerifiedPayloadBytes,
		gatewayVerifiedSpoolQuotaBytes,
	)
	if err != nil {
		_ = httpListener.Close()
		return err
	}
	gateway := localGatewayForListener(
		httpListener,
		identity,
		context.ready,
	)
	gateway.spool = spool
	gateway.cache = context.cache
	gateway.openCircuit = context.tripCircuit
	gateway.startServingLocked(httpListener)
	defer gateway.close()

	controlAddress := managedGatewayControlAddress(directory)
	controlListener, err := net.ListenUnix(
		"unix",
		&net.UnixAddr{Name: controlAddress, Net: "unix"},
	)
	if err != nil {
		return fmt.Errorf("listen on managed gateway control socket: %w", err)
	}
	defer controlListener.Close()

	activity := make(chan struct{}, 1)
	serveDone := serveManagedGatewayControl(
		controlListener,
		context,
		identity,
		activity,
	)
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	for {
		select {
		case err := <-serveDone:
			return err
		case <-signals:
			return nil
		case <-activity:
			resetTimer(timer, idleTimeout)
		case <-timer.C:
			if context.ready() {
				timer.Reset(idleTimeout)
				continue
			}
			return nil
		}
	}
}

func openManagedGatewayListener(
	directory string,
) (net.Listener, gatewayIdentity, bool, error) {
	// Rebinding the complete identity keeps Configuration Cache inputs stable.
	// Failure to recover the endpoint rotates endpoint, credential, and
	// generation together before any invocation can register.
	state, err := readManagedGatewayState(directory)
	if err == nil {
		listener, listenErr := net.Listen("tcp4", state.Address)
		if listenErr == nil {
			return listener, gatewayIdentity{
				address:    state.Address,
				username:   state.Username,
				password:   state.Password,
				generation: state.Generation,
			}, false, nil
		}
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, gatewayIdentity{}, false, fmt.Errorf(
			"listen on managed gateway rendezvous: %w",
			err,
		)
	}
	identity, err := newGatewayIdentity(listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		return nil, gatewayIdentity{}, false, err
	}
	return listener, identity, true, nil
}

func readManagedGatewayState(directory string) (managedGatewayState, error) {
	path := filepath.Join(directory, "gateway-state.json")
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return managedGatewayState{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return managedGatewayState{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		info.Size() > managedGatewayMaximumStateBytes {
		return managedGatewayState{}, errors.New(
			"managed gateway state is not a private bounded regular file",
		)
	}
	content, err := io.ReadAll(io.LimitReader(
		file,
		managedGatewayMaximumStateBytes+1,
	))
	if err != nil || len(content) > managedGatewayMaximumStateBytes {
		return managedGatewayState{}, errors.New("read managed gateway state")
	}
	var state managedGatewayState
	if err := decodeStrictJSON(content, &state); err != nil {
		return managedGatewayState{}, err
	}
	if err := validateManagedGatewayState(state); err != nil {
		return managedGatewayState{}, err
	}
	return state, nil
}

func validateManagedGatewayState(state managedGatewayState) error {
	if state.SchemaVersion != managedGatewayStateSchemaVersion ||
		state.Username != gatewayUsername ||
		validateManagedGatewayAddress(state.Address) != nil ||
		validateManagedGatewayCredential(state.Password) != nil ||
		!validPluginAttemptID(state.Generation) {
		return errors.New("managed gateway state is invalid")
	}
	return nil
}

func validateManagedGatewayAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return errors.New("gateway address is not canonical loopback")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 ||
		address != net.JoinHostPort(host, strconv.Itoa(port)) {
		return errors.New("gateway address has an invalid port")
	}
	return nil
}

func validateManagedGatewayCredential(password string) error {
	credential, err := base64.RawURLEncoding.DecodeString(password)
	if err != nil || len(credential) != 32 {
		return errors.New("gateway credential is invalid")
	}
	return nil
}

func validPluginAttemptID(identifier string) bool {
	if len(identifier) != 36 {
		return false
	}
	for index, character := range []byte(identifier) {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !(character >= '0' && character <= '9') &&
			!(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func writeManagedGatewayState(
	directory string,
	identity gatewayIdentity,
) error {
	state := managedGatewayState{
		SchemaVersion: managedGatewayStateSchemaVersion,
		Address:       identity.address,
		Username:      identity.username,
		Password:      identity.password,
		Generation:    identity.generation,
	}
	temporary, err := os.CreateTemp(directory, "gateway-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create managed gateway state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect managed gateway state: %w", err)
	}
	if err := json.NewEncoder(temporary).Encode(state); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode managed gateway state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync managed gateway state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close managed gateway state: %w", err)
	}
	path := filepath.Join(directory, "gateway-state.json")
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish managed gateway state: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open managed gateway state directory: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync managed gateway state directory: %w", err)
	}
	return nil
}

func serveManagedGatewayControl(
	listener *net.UnixListener,
	context *managedGatewayContext,
	identity gatewayIdentity,
	activity chan<- struct{},
) <-chan error {
	done := make(chan error, 1)
	go func() {
		for {
			connection, err := listener.AcceptUnix()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					done <- nil
				} else {
					done <- fmt.Errorf(
						"accept managed gateway registration: %w",
						err,
					)
				}
				return
			}
			go handleManagedGatewayControl(
				connection,
				context,
				identity,
				activity,
			)
		}
	}()
	return done
}

func handleManagedGatewayControl(
	connection *net.UnixConn,
	context *managedGatewayContext,
	identity gatewayIdentity,
	activity chan<- struct{},
) {
	defer connection.Close()
	if err := connection.SetDeadline(
		time.Now().Add(managedGatewayControlTimeout),
	); err != nil {
		return
	}
	if err := verifyUnixPeerOwner(connection); err != nil {
		return
	}
	buffered := bufio.NewReaderSize(
		connection,
		managedGatewayMaximumControlBytes+1,
	)
	line, err := readManagedGatewayLine(buffered)
	if err != nil {
		return
	}
	var request managedGatewayControlRequest
	if err := decodeStrictJSON(line, &request); err != nil ||
		request.SchemaVersion != managedGatewayControlSchemaVersion ||
		request.Operation != "register" ||
		!validPluginAttemptID(request.AttemptID) {
		_ = writeManagedGatewayControlResponse(
			connection,
			managedGatewayControlResponse{
				SchemaVersion: managedGatewayControlSchemaVersion,
				Error:         "invalid managed gateway registration",
			},
		)
		return
	}
	cacheBinding, err := request.Cache.binding(request.AttemptID)
	if err != nil {
		_ = writeManagedGatewayControlResponse(
			connection,
			managedGatewayControlResponse{
				SchemaVersion: managedGatewayControlSchemaVersion,
				Error:         "invalid managed gateway registration",
			},
		)
		return
	}
	registered, cacheSuppressed := context.registerWithCacheStatus(
		request.AttemptID,
		cacheBinding,
	)
	if !registered {
		_ = writeManagedGatewayControlResponse(
			connection,
			managedGatewayControlResponse{
				SchemaVersion: managedGatewayControlSchemaVersion,
				Error:         "managed gateway already has an active invocation",
			},
		)
		return
	}
	// The open control connection is the invocation registration. EOF or a
	// launcher crash removes the context before another build can become ready.
	defer context.unregister(request.AttemptID)
	notifyManagedGatewayActivity(activity)

	response := managedGatewayControlResponse{
		SchemaVersion:   managedGatewayControlSchemaVersion,
		Endpoint:        "http://" + identity.address,
		Username:        identity.username,
		Password:        identity.password,
		Generation:      identity.generation,
		CacheSuppressed: cacheSuppressed,
	}
	if err := writeManagedGatewayControlResponse(connection, response); err != nil {
		return
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		return
	}
	_, _ = io.Copy(io.Discard, buffered)
	notifyManagedGatewayActivity(activity)
}

func writeManagedGatewayControlResponse(
	writer io.Writer,
	response managedGatewayControlResponse,
) error {
	return json.NewEncoder(writer).Encode(response)
}

func readManagedGatewayLine(reader *bufio.Reader) ([]byte, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > managedGatewayMaximumControlBytes {
		return nil, errors.New("managed gateway control message is too large")
	}
	return line, nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON contains more than one value")
		}
		return err
	}
	return nil
}

func verifyUnixPeerOwner(connection *net.UnixConn) error {
	rawConnection, err := connection.SyscallConn()
	if err != nil {
		return errors.New("inspect managed gateway peer")
	}
	var peer *syscall.Ucred
	var credentialErr error
	if err := rawConnection.Control(func(fileDescriptor uintptr) {
		peer, credentialErr = syscall.GetsockoptUcred(
			int(fileDescriptor),
			syscall.SOL_SOCKET,
			syscall.SO_PEERCRED,
		)
	}); err != nil || credentialErr != nil || peer == nil {
		return errors.New("inspect managed gateway peer")
	}
	if peer.Uid != uint32(os.Geteuid()) {
		return errors.New("managed gateway peer has a different user")
	}
	return nil
}

func (context *managedGatewayContext) register(attemptID string) bool {
	return context.registerWithCache(attemptID, nil)
}

func (context *managedGatewayContext) registerWithCache(
	attemptID string,
	cacheBinding *gatewayCacheBinding,
) bool {
	registered, _ := context.registerWithCacheStatus(attemptID, cacheBinding)
	return registered
}

func (context *managedGatewayContext) registerWithCacheStatus(
	attemptID string,
	cacheBinding *gatewayCacheBinding,
) (bool, bool) {
	context.mutex.Lock()
	defer context.mutex.Unlock()
	if context.attemptID != "" {
		return false, false
	}
	cacheSuppressed := cacheBinding != nil &&
		context.circuit != nil &&
		context.circuit.cacheSuppressed()
	context.attemptID = attemptID
	if !cacheSuppressed {
		context.cacheBinding = cacheBinding.copy()
	}
	return true, cacheSuppressed
}

func (context *managedGatewayContext) tripCircuit(
	reason gatewayCircuitReason,
) {
	if context.circuit != nil {
		_ = context.circuit.trip(reason)
	}
	context.mutex.Lock()
	context.cacheBinding = nil
	context.mutex.Unlock()
}

func (context *managedGatewayContext) unregister(attemptID string) {
	context.mutex.Lock()
	defer context.mutex.Unlock()
	if context.attemptID == attemptID {
		context.attemptID = ""
		context.cacheBinding = nil
	}
}

func (context *managedGatewayContext) ready() bool {
	context.mutex.Lock()
	defer context.mutex.Unlock()
	return context.attemptID != ""
}

func (context *managedGatewayContext) cache() *gatewayCacheBinding {
	context.mutex.Lock()
	defer context.mutex.Unlock()
	if context.attemptID == "" {
		return nil
	}
	return context.cacheBinding.copy()
}

func notifyManagedGatewayActivity(activity chan<- struct{}) {
	select {
	case activity <- struct{}{}:
	default:
	}
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}
