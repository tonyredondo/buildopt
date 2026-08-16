package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

func writeCanonicalPrivateJSON(path string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return err
	}
	return writePrivateAtomicFile(path, canonical)
}

func writePrivateAtomicFile(path string, contents []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".buildopt-state-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
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
	if err := replaceManagedFile(temporaryPath, path); err != nil {
		return err
	}
	return syncManagedDirectory(filepath.Dir(path))
}
