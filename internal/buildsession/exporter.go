package buildsession

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

// ErrExportConflict reports an existing immutable export with different bytes.
var ErrExportConflict = errors.New(
	"BUILD_SESSION export already exists with different content",
)

// Exporter atomically publishes immutable BUILD_SESSION JSON files.
type Exporter struct {
	directory string
}

// NewExporter prepares a local directory for BUILD_SESSION exports.
func NewExporter(directory string) (*Exporter, error) {
	if directory == "" {
		return nil, errors.New("BUILD_SESSION export directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, errors.New("resolve BUILD_SESSION export directory")
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, errors.New("create BUILD_SESSION export directory")
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.IsDir() {
		return nil, errors.New("BUILD_SESSION export path is not a directory")
	}
	return &Exporter{directory: absolute}, nil
}

// Export atomically publishes one immutable JSON document. An identical
// replay returns created=false without rewriting the existing file.
func (exporter *Exporter) Export(
	record sessioningest.Record,
) (path string, created bool, err error) {
	document, err := NewDocument(record)
	if err != nil {
		return "", false, err
	}
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return "", false, errors.New("encode BUILD_SESSION export")
	}
	content = append(content, '\n')

	target := filepath.Join(
		exporter.directory,
		exportFilename(record.SessionID),
	)
	temporary, err := os.CreateTemp(
		exporter.directory,
		".build-session-*.tmp",
	)
	if err != nil {
		return "", false, errors.New("create BUILD_SESSION export temporary file")
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return "", false, errors.New(
			"set BUILD_SESSION export permissions",
		)
	}
	if _, err := temporary.Write(content); err != nil {
		return "", false, errors.New("write BUILD_SESSION export")
	}
	if err := temporary.Sync(); err != nil {
		return "", false, errors.New("sync BUILD_SESSION export")
	}
	if err := temporary.Close(); err != nil {
		return "", false, errors.New("close BUILD_SESSION export")
	}

	if err := os.Link(temporaryPath, target); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return "", false, errors.New(
				"publish BUILD_SESSION export",
			)
		}
		identical, compareErr := identicalRegularFile(target, content)
		if compareErr != nil {
			return "", false, compareErr
		}
		if !identical {
			return "", false, ErrExportConflict
		}
		return target, false, nil
	}
	if err := os.Remove(temporaryPath); err != nil {
		return "", false, errors.New(
			"remove BUILD_SESSION export temporary file",
		)
	}
	if err := syncDirectory(exporter.directory); err != nil {
		return "", false, err
	}
	return target, true, nil
}

func exportFilename(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return "build-session-" + hex.EncodeToString(sum[:]) + ".json"
}

func identicalRegularFile(path string, expected []byte) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, errors.New("inspect existing BUILD_SESSION export")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return false, errors.New(
			"existing BUILD_SESSION export is not a private regular file",
		)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false, errors.New("read existing BUILD_SESSION export")
	}
	return bytes.Equal(content, expected), nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return errors.New("open BUILD_SESSION export directory")
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync BUILD_SESSION export directory: %w", err)
	}
	return nil
}
