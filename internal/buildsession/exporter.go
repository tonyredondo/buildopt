package buildsession

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

// ErrExportConflict reports an existing immutable export with different bytes.
var ErrExportConflict = errors.New(
	"BUILD_SESSION export already exists with different content",
)

// Exporter publishes immutable BUILD_SESSION JSON and bounded JSONL events.
type Exporter struct {
	directory string
	stream    *jsonlStream
	now       func() time.Time
	mutex     sync.Mutex
}

// NewExporter prepares a local directory for BUILD_SESSION exports.
func NewExporter(directory string) (*Exporter, error) {
	return newExporter(directory, func() time.Time {
		return time.Now().UTC()
	})
}

func newExporter(
	directory string,
	now func() time.Time,
) (*Exporter, error) {
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
	info, err := os.Lstat(absolute)
	stat, ownerAvailable := infoSyscallStat(info)
	if err != nil ||
		!info.IsDir() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o700 ||
		!ownerAvailable ||
		stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.New(
			"BUILD_SESSION export path is not a private directory",
		)
	}
	exporter := &Exporter{
		directory: absolute,
		stream: &jsonlStream{
			path: filepath.Join(absolute, "buildopt-events.jsonl"),
		},
		now: now,
	}
	if err := exporter.recoverPartialDocuments(); err != nil {
		return nil, err
	}
	return exporter, nil
}

func infoSyscallStat(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
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
	events, err := newBuildSessionEvents(document, target, content)
	if err != nil {
		return "", false, err
	}

	exporter.mutex.Lock()
	defer exporter.mutex.Unlock()

	exists, identical, err := inspectOptionalPrivateFile(target, content)
	if err != nil {
		return "", false, err
	}
	if exists && !identical {
		return "", false, ErrExportConflict
	}
	if err := exporter.stream.ensureCapacity(events[:]...); err != nil {
		return "", false, err
	}
	if err := exporter.stream.append(events[0]); err != nil {
		return "", false, err
	}
	if !exists {
		if err := publishPrivateFile(
			exporter.directory,
			target,
			content,
			"BUILD_SESSION export",
		); err != nil {
			return "", false, err
		}
		created = true
	}
	if err := exporter.stream.append(events[1]); err != nil {
		return target, created, err
	}
	return target, created, nil
}

// WriteJSONL validates and copies the durable at-least-once stream to writer.
func (exporter *Exporter) WriteJSONL(writer io.Writer) error {
	if writer == nil {
		return errors.New("JSONL export writer is required")
	}
	exporter.mutex.Lock()
	defer exporter.mutex.Unlock()
	raw, _, err := exporter.stream.load(true)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return errors.New("JSONL export stream is empty")
	}
	if _, err := writer.Write(raw); err != nil {
		return errors.New("write JSONL export stream")
	}
	return nil
}

func publishPrivateFile(
	directory string,
	target string,
	content []byte,
	label string,
) error {
	temporary, err := os.CreateTemp(
		directory,
		".build-session-*.tmp",
	)
	if err != nil {
		return fmt.Errorf("create %s temporary file", label)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set %s permissions", label)
	}
	if _, err := temporary.Write(content); err != nil {
		return fmt.Errorf("write %s", label)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync %s", label)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s", label)
	}

	if err := os.Link(temporaryPath, target); err != nil {
		if errors.Is(err, fs.ErrExist) {
			identical, compareErr := identicalRegularFile(target, content)
			if compareErr != nil {
				return compareErr
			}
			if identical {
				return nil
			}
			return ErrExportConflict
		}
		return fmt.Errorf("publish %s", label)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove %s temporary file", label)
	}
	if err := syncDirectory(directory); err != nil {
		return err
	}
	return nil
}

func exportFilename(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return "build-session-" + hex.EncodeToString(sum[:]) + ".json"
}

func identicalRegularFile(path string, expected []byte) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
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

func inspectOptionalPrivateFile(
	path string,
	expected []byte,
) (exists bool, identical bool, err error) {
	identical, err = identicalRegularFile(path, expected)
	if errors.Is(err, fs.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, identical, nil
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
