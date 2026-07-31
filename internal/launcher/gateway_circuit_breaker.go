package launcher

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	managedGatewayCircuitSchemaVersion = 1
	managedGatewayCircuitCooldown      = 5 * time.Minute
	managedGatewayCircuitStateFile     = "circuit-breaker.json"

	gatewayCircuitFlood          gatewayCircuitReason = "FLOOD"
	gatewayCircuitObjectTooLarge gatewayCircuitReason = "OBJECT_TOO_LARGE"
	gatewayCircuitDiskPressure   gatewayCircuitReason = "DISK_PRESSURE"
)

type gatewayCircuitReason string

type managedGatewayCircuitState struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Reason        gatewayCircuitReason `json:"reason"`
	OpenedAt      string               `json:"openedAt"`
	RetryAfter    string               `json:"retryAfter"`
}

type managedGatewayCircuitBreaker struct {
	directory string
	now       func() time.Time
	cooldown  time.Duration

	mutex       sync.Mutex
	memoryState *managedGatewayCircuitState
}

func newManagedGatewayCircuitBreaker(
	directory string,
) *managedGatewayCircuitBreaker {
	return &managedGatewayCircuitBreaker{
		directory: directory,
		now:       time.Now,
		cooldown:  managedGatewayCircuitCooldown,
	}
}

func (breaker *managedGatewayCircuitBreaker) trip(
	reason gatewayCircuitReason,
) error {
	if breaker == nil || !validGatewayCircuitReason(reason) {
		return errors.New("invalid managed gateway circuit transition")
	}
	breaker.mutex.Lock()
	defer breaker.mutex.Unlock()

	now := breaker.now().UTC()
	state := managedGatewayCircuitState{
		SchemaVersion: managedGatewayCircuitSchemaVersion,
		Reason:        reason,
		OpenedAt:      now.Format(time.RFC3339Nano),
		RetryAfter:    now.Add(breaker.cooldown).Format(time.RFC3339Nano),
	}
	breaker.memoryState = &state
	if err := writeManagedGatewayCircuitState(breaker.directory, state); err != nil {
		return fmt.Errorf("persist managed gateway circuit: %w", err)
	}
	return nil
}

// cacheSuppressed fails closed for malformed or unsafe durable state. An
// expired valid state is removed durably before L2 can be enabled again.
func (breaker *managedGatewayCircuitBreaker) cacheSuppressed() bool {
	if breaker == nil {
		return false
	}
	breaker.mutex.Lock()
	defer breaker.mutex.Unlock()

	now := breaker.now().UTC()
	if breaker.memoryState != nil {
		_, retryAfter, err := validateManagedGatewayCircuitState(
			*breaker.memoryState,
		)
		if err != nil || now.Before(retryAfter) {
			return true
		}
		breaker.memoryState = nil
	}

	state, err := readManagedGatewayCircuitState(breaker.directory)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	_, retryAfter, err := validateManagedGatewayCircuitState(state)
	if err != nil || now.Before(retryAfter) {
		return true
	}
	if err := removeManagedGatewayCircuitState(breaker.directory); err != nil {
		return true
	}
	return false
}

func validGatewayCircuitReason(reason gatewayCircuitReason) bool {
	switch reason {
	case gatewayCircuitFlood,
		gatewayCircuitObjectTooLarge,
		gatewayCircuitDiskPressure:
		return true
	default:
		return false
	}
}

func validateManagedGatewayCircuitState(
	state managedGatewayCircuitState,
) (time.Time, time.Time, error) {
	if state.SchemaVersion != managedGatewayCircuitSchemaVersion ||
		!validGatewayCircuitReason(state.Reason) {
		return time.Time{}, time.Time{}, errors.New(
			"managed gateway circuit state is invalid",
		)
	}
	openedAt, err := parseCanonicalUTCTimestamp(state.OpenedAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	retryAfter, err := parseCanonicalUTCTimestamp(state.RetryAfter)
	if err != nil || !retryAfter.After(openedAt) {
		return time.Time{}, time.Time{}, errors.New(
			"managed gateway circuit window is invalid",
		)
	}
	return openedAt, retryAfter, nil
}

func parseCanonicalUTCTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil ||
		parsed.Location() != time.UTC ||
		parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New(
			"managed gateway circuit timestamp is invalid",
		)
	}
	return parsed, nil
}

func readManagedGatewayCircuitState(
	directory string,
) (managedGatewayCircuitState, error) {
	path := filepath.Join(directory, managedGatewayCircuitStateFile)
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return managedGatewayCircuitState{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return managedGatewayCircuitState{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		info.Size() > managedGatewayMaximumStateBytes {
		return managedGatewayCircuitState{}, errors.New(
			"managed gateway circuit state is not a private bounded regular file",
		)
	}
	content, err := io.ReadAll(io.LimitReader(
		file,
		managedGatewayMaximumStateBytes+1,
	))
	if err != nil || len(content) > managedGatewayMaximumStateBytes {
		return managedGatewayCircuitState{}, errors.New(
			"read managed gateway circuit state",
		)
	}
	var state managedGatewayCircuitState
	if err := decodeStrictJSON(content, &state); err != nil {
		return managedGatewayCircuitState{}, err
	}
	if _, _, err := validateManagedGatewayCircuitState(state); err != nil {
		return managedGatewayCircuitState{}, err
	}
	return state, nil
}

func writeManagedGatewayCircuitState(
	directory string,
	state managedGatewayCircuitState,
) error {
	if _, _, err := validateManagedGatewayCircuitState(state); err != nil {
		return err
	}
	path := filepath.Join(directory, managedGatewayCircuitStateFile)
	if err := writeCanonicalPrivateJSON(path, state); err != nil {
		return err
	}
	return nil
}

func removeManagedGatewayCircuitState(directory string) error {
	path := filepath.Join(directory, managedGatewayCircuitStateFile)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer directoryHandle.Close()
	return directoryHandle.Sync()
}
