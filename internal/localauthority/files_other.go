//go:build !linux

package localauthority

import (
	"context"
	"crypto/ed25519"
	"errors"
	"time"
)

// FileStateStore is unavailable outside the supported Linux private-beta lane.
type FileStateStore struct{}

func ReadPrivateFile(string, int64) ([]byte, error) {
	return nil, errors.New("local cache authority requires Linux")
}

func LoadFiles(
	context.Context,
	string,
	string,
	string,
	time.Time,
) (Verified, map[string]ed25519.PublicKey, []byte, error) {
	return Verified{}, nil, nil, errors.New(
		"local cache authority requires Linux",
	)
}

func LoadSigningKeyFile(
	string,
	map[string]ed25519.PublicKey,
) (string, []byte, error) {
	return "", nil, errors.New("local cache authority requires Linux")
}

func NewFileStateStore(string) (*FileStateStore, error) {
	return nil, errors.New("local cache authority requires Linux")
}

func (*FileStateStore) Install(
	Verified,
	time.Time,
) (State, State, bool, error) {
	return State{}, State{}, false, errors.New(
		"local cache authority requires Linux",
	)
}
