//go:build linux

package localauthority

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const maximumLocalAuthorityFileBytes = 4 << 20

// ReadPrivateFile reads one bounded, current-user-owned mode-0600 regular file
// without following a final symlink.
func ReadPrivateFile(path string, maximumBytes int64) ([]byte, error) {
	if !filepath.IsAbs(path) ||
		filepath.Clean(path) != path ||
		maximumBytes < 1 ||
		maximumBytes > maximumLocalAuthorityFileBytes {
		return nil, errors.New("invalid private file request")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		info.Size() < 1 ||
		info.Size() > maximumBytes {
		return nil, errors.New(
			"local authority file is not a private bounded regular file",
		)
	}
	content, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil || int64(len(content)) > maximumBytes {
		return nil, errors.New("read bounded local authority file")
	}
	return content, nil
}

// LoadFiles reads and verifies one authority, trust root, and data credential.
func LoadFiles(
	ctx context.Context,
	authorityPath string,
	trustRootPath string,
	credentialPath string,
	now time.Time,
) (Verified, map[string]ed25519.PublicKey, []byte, error) {
	authority, err := ReadPrivateFile(
		authorityPath,
		maximumAuthorityBytes,
	)
	if err != nil {
		return Verified{}, nil, nil, fmt.Errorf(
			"read local authority: %w",
			err,
		)
	}
	trustRoot, err := ReadPrivateFile(
		trustRootPath,
		maximumTrustRootBytes,
	)
	if err != nil {
		return Verified{}, nil, nil, fmt.Errorf(
			"read local trust root: %w",
			err,
		)
	}
	keys, err := ParseTrustRoot(trustRoot)
	if err != nil {
		return Verified{}, nil, nil, err
	}
	credentialFile, err := ReadPrivateFile(credentialPath, 128)
	if err != nil {
		return Verified{}, nil, nil, fmt.Errorf(
			"read local cache credential: %w",
			err,
		)
	}
	credential, err := ParseCredential(credentialFile)
	if err != nil {
		return Verified{}, nil, nil, err
	}
	verified, err := Verify(ctx, authority, keys, credential, now)
	if err != nil {
		return Verified{}, nil, nil, err
	}
	return verified, keys, credential, nil
}

// LoadSigningKeyFile reads the private deployment key and proves it belongs to
// the pinned trust root.
func LoadSigningKeyFile(
	path string,
	keys map[string]ed25519.PublicKey,
) (string, []byte, error) {
	raw, err := ReadPrivateFile(path, maximumSigningKeyBytes)
	if err != nil {
		return "", nil, err
	}
	keyID, privateKey, err := ParseSigningKey(raw)
	if err != nil {
		return "", nil, err
	}
	publicKey, ok := keys[keyID]
	derivedPublicKey, publicOK := privateKey.Public().(ed25519.PublicKey)
	if !ok || !publicOK || !bytes.Equal(derivedPublicKey, publicKey) {
		return "", nil, errors.New(
			"local signing key does not match the pinned trust root",
		)
	}
	return keyID, bytes.Clone(privateKey), nil
}

// FileStateStore owns launcher-side anti-rollback state beneath one private
// managed root.
type FileStateStore struct {
	root string
}

// NewFileStateStore validates or creates the private local state hierarchy.
func NewFileStateStore(root string) (*FileStateStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, errors.New(
			"local authority state root must be absolute and clean",
		)
	}
	for _, directory := range []string{
		root,
		filepath.Join(root, "policy"),
		filepath.Join(root, "policy", "scopes"),
	} {
		if err := preparePrivateDirectory(directory); err != nil {
			return nil, err
		}
	}
	return &FileStateStore{root: root}, nil
}

// Install advances one scope under a process-safe lease and returns the prior
// and current state plus whether durable state changed.
func (store *FileStateStore) Install(
	verified Verified,
	now time.Time,
) (State, State, bool, error) {
	if store == nil {
		return State{}, State{}, false, errors.New("nil authority state store")
	}
	next := StateFromVerified(verified, now)
	lockPath := filepath.Join(
		store.root,
		"policy",
		"scopes",
		next.ScopeDigest+".lock",
	)
	lock, err := openPrivateLock(lockPath)
	if err != nil {
		return State{}, State{}, false, err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return State{}, State{}, false, err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	statePath := filepath.Join(
		store.root,
		"policy",
		"scopes",
		next.ScopeDigest+".json",
	)
	current, err := readState(statePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return State{}, State{}, false, err
	}
	if err := Advance(current, next); err != nil {
		return current, State{}, false, err
	}
	if equivalentState(current, next) {
		return current, current, false, nil
	}
	if err := writeState(statePath, next); err != nil {
		return current, State{}, false, err
	}
	return current, next, true, nil
}

func preparePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.IsDir() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o700 {
		return fmt.Errorf("%s is not a private directory", path)
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
		return nil, errors.New(
			"local authority lock is not a private regular file",
		)
	}
	return file, nil
}

func readState(path string) (State, error) {
	raw, err := ReadPrivateFile(path, 64<<10)
	if err != nil {
		return State{}, err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return State{}, errors.New(
			"persisted local authority state is not canonical",
		)
	}
	var state State
	if err := decodeStrict(canonical, &state); err != nil {
		return State{}, err
	}
	if err := validateState(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func writeState(path string, state State) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".authority-state-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(canonical); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer directoryHandle.Close()
	return directoryHandle.Sync()
}

func equivalentState(left State, right State) bool {
	left.InstalledAt = ""
	right.InstalledAt = ""
	return left == right
}
