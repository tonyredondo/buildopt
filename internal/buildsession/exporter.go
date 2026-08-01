package buildsession

import (
	"bytes"
	"crypto/rand"
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
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const exportRedactionKeyName = ".buildopt-export-redaction.key"

// ErrExportConflict reports an existing immutable export with different bytes.
var ErrExportConflict = errors.New(
	"BUILD_SESSION export already exists with different content",
)

// Exporter publishes immutable BUILD_SESSION JSON and bounded JSONL events.
type Exporter struct {
	directory      string
	stream         *jsonlStream
	redactor       *datalifecycle.Redactor
	profile        datalifecycle.ExportProfile
	policy         datalifecycle.ExportPolicy
	lifecycleLease *datalifecycle.ManagedLease
	now            func() time.Time
	mutex          sync.Mutex
}

// NewExporter prepares a local directory for BUILD_SESSION exports.
func NewExporter(directory string) (*Exporter, error) {
	return NewExporterWithPolicy(directory, datalifecycle.ExportPolicy{
		Profile: datalifecycle.ExportSummary,
	})
}

// NewExporterWithPolicy prepares an exporter with one explicit profile
// ceiling. Diagnostic exports always require a bounded opt-in.
func NewExporterWithPolicy(
	directory string,
	policy datalifecycle.ExportPolicy,
) (*Exporter, error) {
	return newExporterWithPolicy(directory, policy, func() time.Time {
		return time.Now().UTC()
	})
}

func newExporter(
	directory string,
	now func() time.Time,
) (*Exporter, error) {
	return newExporterWithPolicy(directory, datalifecycle.ExportPolicy{
		Profile: datalifecycle.ExportSummary,
	}, now)
}

func newExporterWithPolicy(
	directory string,
	policy datalifecycle.ExportPolicy,
	now func() time.Time,
) (*Exporter, error) {
	if directory == "" {
		return nil, errors.New("BUILD_SESSION export directory is required")
	}
	if now == nil {
		return nil, errors.New("BUILD_SESSION export clock is required")
	}
	if policy.Profile == "" {
		policy.Profile = datalifecycle.ExportSummary
	}
	if err := datalifecycle.ValidateExportPolicy(policy, now().UTC()); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, errors.New("resolve BUILD_SESSION export directory")
	}
	lifecycleLease, _, err := datalifecycle.AcquireManagedLease(absolute)
	if err != nil {
		return nil, err
	}
	keepLifecycleLease := false
	defer func() {
		if !keepLifecycleLease {
			_ = lifecycleLease.Close()
		}
	}()
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, errors.New("create BUILD_SESSION export directory")
	}
	info, err := os.Lstat(absolute)
	if err != nil ||
		!privateDirectoryInfo(info) {
		return nil, errors.New(
			"BUILD_SESSION export path is not a private directory",
		)
	}
	key, version, err := loadOrCreateExportRedactionKey(absolute)
	if err != nil {
		return nil, err
	}
	redactor, err := datalifecycle.NewRedactor(key, version)
	for index := range key {
		key[index] = 0
	}
	if err != nil {
		return nil, err
	}
	exporter := &Exporter{
		directory: absolute,
		stream: &jsonlStream{
			path:              filepath.Join(absolute, "buildopt-events.jsonl"),
			authorizedProfile: policy.Profile,
		},
		redactor:       redactor,
		profile:        policy.Profile,
		policy:         policy,
		lifecycleLease: lifecycleLease,
		now:            now,
	}
	if err := exporter.recoverPartialDocuments(); err != nil {
		return nil, err
	}
	keepLifecycleLease = true
	return exporter, nil
}

func loadOrCreateExportRedactionKey(
	directory string,
) ([]byte, string, error) {
	path := filepath.Join(directory, exportRedactionKeyName)
	key, err := readPrivateRedactionKey(path)
	if errors.Is(err, fs.ErrNotExist) {
		entries, readErr := os.ReadDir(directory)
		if readErr != nil {
			return nil, "", errors.New("inspect BUILD_SESSION export directory")
		}
		if len(entries) != 0 {
			return nil, "", errors.New(
				"BUILD_SESSION export directory has no redaction key",
			)
		}
		key = make([]byte, datalifecycle.RedactionKeyBytes)
		if _, randomErr := rand.Read(key); randomErr != nil {
			return nil, "", errors.New("generate BUILD_SESSION redaction key")
		}
		file, createErr := openNoFollow(
			path,
			os.O_CREATE|os.O_EXCL|os.O_WRONLY,
			0o600,
		)
		if createErr != nil {
			return nil, "", errors.New("create BUILD_SESSION redaction key")
		}
		written, writeErr := file.Write(key)
		syncErr := file.Sync()
		closeErr := file.Close()
		if writeErr != nil || written != len(key) ||
			syncErr != nil || closeErr != nil {
			_ = os.Remove(path)
			return nil, "", errors.New("persist BUILD_SESSION redaction key")
		}
		if syncErr := syncDirectory(directory); syncErr != nil {
			return nil, "", syncErr
		}
	} else if err != nil {
		return nil, "", err
	}

	versionDigest := sha256.New()
	_, _ = versionDigest.Write([]byte("buildopt-export-key-version-v1"))
	_, _ = versionDigest.Write([]byte{0})
	_, _ = versionDigest.Write(key)
	version := "managed-v1-" + hex.EncodeToString(versionDigest.Sum(nil)[:8])
	return key, version, nil
}

func readPrivateRedactionKey(path string) ([]byte, error) {
	file, err := openNoFollow(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil ||
		!privateFileInfo(info) ||
		info.Size() != datalifecycle.RedactionKeyBytes {
		return nil, errors.New(
			"BUILD_SESSION redaction key is not a private fixed-size file",
		)
	}
	key, err := io.ReadAll(io.LimitReader(
		file,
		datalifecycle.RedactionKeyBytes+1,
	))
	if err != nil || len(key) != datalifecycle.RedactionKeyBytes {
		return nil, errors.New("read BUILD_SESSION redaction key")
	}
	return key, nil
}

// Directory returns the canonical private export root for in-process,
// read-only consumers such as the local build-history API.
func (exporter *Exporter) Directory() string {
	if exporter == nil {
		return ""
	}
	return exporter.directory
}

// Export atomically publishes one immutable JSON document. An identical
// replay returns created=false without rewriting the existing file.
func (exporter *Exporter) Export(
	record sessioningest.Record,
) (path string, created bool, err error) {
	if err := exporter.validateCurrentPolicy(); err != nil {
		return "", false, err
	}
	document, err := exporter.newDocument(record)
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
	events, err := newBuildSessionEvents(
		document,
		target,
		content,
		exporter.profile,
	)
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

func (exporter *Exporter) validateCurrentPolicy() error {
	if exporter == nil || exporter.now == nil {
		return errors.New("BUILD_SESSION exporter has no policy clock")
	}
	return datalifecycle.ValidateExportPolicy(
		exporter.policy,
		exporter.now().UTC(),
	)
}

func (exporter *Exporter) newDocument(
	record sessioningest.Record,
) (Document, error) {
	if exporter == nil || exporter.redactor == nil {
		return Document{}, errors.New("BUILD_SESSION exporter has no redactor")
	}
	if record.ExportContext == nil {
		return Document{}, errors.New("BUILD_SESSION export context is required")
	}
	contextCopy := *record.ExportContext
	contextCopy.RepositoryID = exporter.redactor.Token(
		"repository",
		contextCopy.RepositoryID,
	)
	contextCopy.TrustDomain = exporter.redactor.Token(
		"trust-domain",
		contextCopy.TrustDomain,
	)
	contextCopy.RequestedTasks = append(
		[]string(nil),
		contextCopy.RequestedTasks...,
	)
	if exporter.profile == datalifecycle.ExportSummary {
		taskSet, err := json.Marshal(contextCopy.RequestedTasks)
		if err != nil {
			return Document{}, errors.New("encode BUILD_SESSION task set")
		}
		contextCopy.RequestedTasks = []string{exporter.redactor.Token(
			"gradle-task-set",
			string(taskSet),
		)}
	} else {
		for index, task := range contextCopy.RequestedTasks {
			contextCopy.RequestedTasks[index] = exporter.redactor.Token(
				"gradle-task",
				task,
			)
		}
	}
	contextCopy.TokenKeyVersion = exporter.redactor.Version()
	record.ExportContext = &contextCopy
	return NewDocument(record)
}

// WriteJSONL validates and copies the durable at-least-once stream to writer.
func (exporter *Exporter) WriteJSONL(writer io.Writer) error {
	if writer == nil {
		return errors.New("JSONL export writer is required")
	}
	if err := exporter.validateCurrentPolicy(); err != nil {
		return err
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

// Close releases the managed-data lifecycle lease held by the exporter.
func (exporter *Exporter) Close() error {
	if exporter == nil || exporter.lifecycleLease == nil {
		return nil
	}
	return exporter.lifecycleLease.Close()
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
