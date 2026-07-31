package datalifecycle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

const (
	deploymentDataMarkerName = ".buildopt-deployment-data.json"
	deletionTombstoneName    = ".buildopt-data-deletion.json"
	deletionTemporaryName    = ".buildopt-data-deletion.tmp"
	lifecycleLockName        = ".buildopt-data-lifecycle.lock"

	deletionSchemaVersion = "buildopt.private-beta-deletion/v1"
	deletionStateRevoked  = "LOGICALLY_REVOKED"
	deletionStateComplete = "PHYSICAL_COMPLETE"
	deletionRetention     = 7 * 24 * time.Hour
)

var (
	// ErrManagedDataRevoked means a deletion tombstone blocks every managed
	// component until physical deletion completes.
	ErrManagedDataRevoked = errors.New("managed private-beta data is logically revoked")
	// ErrManagedDataBusy means a server or managed-L1 invocation still owns a
	// lease beneath the selected deployment data root.
	ErrManagedDataBusy = errors.New("managed private-beta data is active")
	// ErrRetentionHoldActive means an explicitly consented extraordinary hold
	// prevents deletion. An incomplete or implicit hold is rejected as invalid.
	ErrRetentionHoldActive = errors.New("explicit private-beta retention hold is active")
)

var managedComponents = []struct {
	class string
	name  string
}{
	{class: "STABLE_PENDING_QUARANTINE_METADATA", name: "shared"},
	{class: "MANAGED_L1", name: "l1"},
	{class: "SUMMARY_DIAGNOSTIC_TELEMETRY", name: "exports"},
	{class: "OPTIMIZATION_EVIDENCE", name: "evidence"},
	{class: "LOCAL_SPOOL_DLQ", name: "spool"},
}

// RetentionHold is the only accepted shape for extraordinary beta retention.
// DeleteManagedData never creates a hold; a complete, unexpired hold blocks it.
type RetentionHold struct {
	ConsentID string
	Reason    string
	ExpiresAt time.Time
}

// DeletionRequest revokes and deletes one complete isolated deployment data
// root. Raw scope and destination identities are tokenized before persistence.
type DeletionRequest struct {
	DataRoot                 string
	DeletionID               string
	Tenant                   string
	Repository               string
	TrustDomain              string
	NextNamespaceGeneration  int64
	NextL1SecurityGeneration uint64
	TokenKey                 []byte
	TokenKeyVersion          string
	ExternalDestinations     []string
	RequestedAt              time.Time
	RetentionHold            *RetentionHold
}

// ExternalObligation records that a customer-controlled copy is outside the
// physical deletion boundary and must consume the emitted tombstone.
type ExternalObligation struct {
	DestinationToken string `json:"destinationToken"`
	State            string `json:"state"`
}

// DeletionTombstone is the durable, secret-free deletion and rotation record.
type DeletionTombstone struct {
	SchemaVersion            string               `json:"schemaVersion"`
	DeletionID               string               `json:"deletionId"`
	RequestDigest            string               `json:"requestDigest"`
	TokenKeyVersion          string               `json:"tokenKeyVersion"`
	TenantToken              string               `json:"tenantToken"`
	RepositoryToken          string               `json:"repositoryToken"`
	TrustDomainToken         string               `json:"trustDomainToken"`
	State                    string               `json:"state"`
	RequestedAt              string               `json:"requestedAt"`
	PhysicalCompletedAt      string               `json:"physicalCompletedAt,omitempty"`
	RetainUntil              string               `json:"retainUntil"`
	NextNamespaceGeneration  int64                `json:"nextNamespaceGeneration"`
	NextL1SecurityGeneration uint64               `json:"nextL1SecurityGeneration"`
	ManagedClasses           []string             `json:"managedClasses"`
	ExternalObligations      []ExternalObligation `json:"externalObligations"`
	RemovedComponents        int                  `json:"removedComponents"`
}

// DeletionReport is the bounded operator result for one first run or replay.
type DeletionReport struct {
	Tombstone DeletionTombstone `json:"tombstone"`
	Replay    bool              `json:"replay"`
}

// ManagedBoundary describes an optional deployment root and its current
// deletion generation floor.
type ManagedBoundary struct {
	Managed                     bool
	DataRoot                    string
	Revoked                     bool
	MinimumNamespaceGeneration  int64
	MinimumL1SecurityGeneration uint64
}

// ManagedLease holds a shared lifecycle lock while one managed component is
// open. Deletion takes the exclusive side of the same lock.
type ManagedLease struct {
	file *os.File
}

// AcquireManagedLease discovers the marked root for path, takes its shared
// lifecycle lock, then rechecks revocation under that lock.
func AcquireManagedLease(
	path string,
) (*ManagedLease, ManagedBoundary, error) {
	boundary, err := InspectManagedPath(path)
	if err != nil {
		return nil, ManagedBoundary{}, err
	}
	lease := &ManagedLease{}
	if !boundary.Managed {
		return lease, boundary, nil
	}
	file, err := os.OpenFile(
		filepath.Join(boundary.DataRoot, lifecycleLockName),
		os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW,
		0o600,
	)
	if err != nil {
		return nil, ManagedBoundary{}, err
	}
	if err := requirePrivateOpenFile(file); err != nil {
		_ = file.Close()
		return nil, ManagedBoundary{}, err
	}
	if err := syscall.Flock(
		int(file.Fd()),
		syscall.LOCK_SH|syscall.LOCK_NB,
	); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) ||
			errors.Is(err, syscall.EAGAIN) {
			return nil, ManagedBoundary{}, ErrManagedDataBusy
		}
		return nil, ManagedBoundary{}, err
	}
	lease.file = file
	boundary, err = InspectManagedPath(path)
	if err != nil {
		_ = lease.Close()
		return nil, ManagedBoundary{}, err
	}
	if boundary.Revoked {
		_ = lease.Close()
		return nil, ManagedBoundary{}, ErrManagedDataRevoked
	}
	return lease, boundary, nil
}

// Close releases one shared managed-data lifecycle lease.
func (lease *ManagedLease) Close() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	file := lease.file
	lease.file = nil
	unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	closeErr := file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

type deletionManager struct {
	afterLogicalRevocation func(DeletionTombstone) error
	removeAll              func(string) error
	now                    func() time.Time
}

// DeleteManagedData writes logical revocation first, then removes every known
// managed copy and completes the retained tombstone. Exact retries are safe.
func DeleteManagedData(
	ctx context.Context,
	request DeletionRequest,
) (DeletionReport, error) {
	return deletionManager{
		removeAll: os.RemoveAll,
		now:       time.Now,
	}.delete(ctx, request)
}

func (manager deletionManager) delete(
	ctx context.Context,
	request DeletionRequest,
) (DeletionReport, error) {
	if ctx == nil {
		return DeletionReport{}, errors.New("delete managed data: nil context")
	}
	request.DataRoot = filepath.Clean(request.DataRoot)
	request.RequestedAt = request.RequestedAt.UTC()
	if err := validateDeletionRequest(request); err != nil {
		return DeletionReport{}, err
	}
	if err := validateDataRoot(request.DataRoot); err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}
	rootLock, err := acquirePrivateLock(
		filepath.Join(request.DataRoot, lifecycleLockName),
	)
	if err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}
	defer releasePrivateLock(rootLock)
	if err := validateDataRootEntries(request.DataRoot); err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}

	expected, err := deletionTombstoneForRequest(request)
	if err != nil {
		return DeletionReport{}, err
	}
	existing, exists, err := loadDeletionTombstone(request.DataRoot)
	if err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}
	if exists {
		if existing.RequestDigest != expected.RequestDigest ||
			existing.DeletionID != expected.DeletionID {
			return DeletionReport{}, errors.New(
				"delete managed data: conflicting deletion tombstone",
			)
		}
		if existing.State == deletionStateComplete {
			return DeletionReport{Tombstone: existing, Replay: true}, nil
		}
		expected = existing
	}

	leases, err := acquireComponentLeases(request.DataRoot)
	if err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}
	defer releasePrivateLocks(leases)
	if !exists {
		if err := writeDeletionTombstone(request.DataRoot, expected); err != nil {
			return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
		}
	}
	if manager.afterLogicalRevocation != nil {
		if err := manager.afterLogicalRevocation(expected); err != nil {
			return DeletionReport{}, err
		}
	}

	removed := expected.RemovedComponents
	for _, component := range managedComponents {
		if err := ctx.Err(); err != nil {
			return DeletionReport{}, err
		}
		original := filepath.Join(request.DataRoot, component.name)
		staged := stagedComponentPath(request.DataRoot, component.name)
		originalExists, err := privateDirectoryExists(original)
		if err != nil {
			return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
		}
		stagedExists, err := privateDirectoryExists(staged)
		if err != nil {
			return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
		}
		if originalExists && stagedExists {
			return DeletionReport{}, errors.New(
				"delete managed data: original and staged component both exist",
			)
		}
		if originalExists {
			if err := os.Rename(original, staged); err != nil {
				return DeletionReport{}, fmt.Errorf(
					"delete managed data: stage %s: %w",
					component.name,
					err,
				)
			}
			stagedExists = true
			removed++
		}
		if stagedExists {
			if err := removeVerifiedStagedTree(
				request.DataRoot,
				staged,
				manager.removeAll,
			); err != nil {
				return DeletionReport{}, err
			}
		}
	}
	if err := syncPrivateDirectory(request.DataRoot); err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}
	completedAt := time.Now().UTC()
	if manager.now != nil {
		completedAt = manager.now().UTC()
	}
	if completedAt.Before(request.RequestedAt) {
		return DeletionReport{}, errors.New(
			"delete managed data: completion clock predates request",
		)
	}
	expected.State = deletionStateComplete
	expected.PhysicalCompletedAt = completedAt.Format(time.RFC3339Nano)
	expected.RemovedComponents = removed
	if err := writeDeletionTombstone(request.DataRoot, expected); err != nil {
		return DeletionReport{}, fmt.Errorf("delete managed data: %w", err)
	}
	return DeletionReport{Tombstone: expected}, nil
}

func validateDeletionRequest(request DeletionRequest) error {
	if !filepath.IsAbs(request.DataRoot) || request.DataRoot == "/" ||
		!validIdentifier(request.DeletionID) ||
		!validIdentifier(request.Tenant) ||
		!validIdentifier(request.Repository) ||
		!validIdentifier(request.TrustDomain) ||
		request.NextNamespaceGeneration < 1 ||
		request.NextL1SecurityGeneration < 1 ||
		len(request.TokenKey) != RedactionKeyBytes ||
		!validIdentifier(request.TokenKeyVersion) ||
		request.RequestedAt.IsZero() {
		return errors.New("delete managed data: invalid request")
	}
	seenDestinations := make(map[string]struct{}, len(request.ExternalDestinations))
	for _, destination := range request.ExternalDestinations {
		if !validIdentifier(destination) {
			return errors.New("delete managed data: invalid external destination")
		}
		if _, exists := seenDestinations[destination]; exists {
			return errors.New("delete managed data: duplicate external destination")
		}
		seenDestinations[destination] = struct{}{}
	}
	if request.RetentionHold != nil {
		hold := request.RetentionHold
		if !validIdentifier(hold.ConsentID) || !validIdentifier(hold.Reason) ||
			!hold.ExpiresAt.UTC().After(request.RequestedAt) {
			return errors.New(
				"delete managed data: retention requires explicit consent, reason, and expiry",
			)
		}
		return ErrRetentionHoldActive
	}
	return nil
}

func deletionTombstoneForRequest(
	request DeletionRequest,
) (DeletionTombstone, error) {
	redactor, err := NewRedactor(request.TokenKey, request.TokenKeyVersion)
	if err != nil {
		return DeletionTombstone{}, err
	}
	classes := make([]string, 0, len(managedComponents))
	for _, component := range managedComponents {
		classes = append(classes, component.class)
	}
	obligations := make([]ExternalObligation, 0, len(request.ExternalDestinations))
	for _, destination := range request.ExternalDestinations {
		obligations = append(obligations, ExternalObligation{
			DestinationToken: redactor.Token("external-destination", destination),
			State:            "TOMBSTONE_REQUIRED",
		})
	}
	slices.SortFunc(obligations, func(left, right ExternalObligation) int {
		return strings.Compare(left.DestinationToken, right.DestinationToken)
	})
	tombstone := DeletionTombstone{
		SchemaVersion:            deletionSchemaVersion,
		DeletionID:               request.DeletionID,
		TokenKeyVersion:          redactor.Version(),
		TenantToken:              redactor.Token("tenant", request.Tenant),
		RepositoryToken:          redactor.Token("repository", request.Repository),
		TrustDomainToken:         redactor.Token("trust-domain", request.TrustDomain),
		State:                    deletionStateRevoked,
		RequestedAt:              request.RequestedAt.Format(time.RFC3339Nano),
		RetainUntil:              request.RequestedAt.Add(deletionRetention).Format(time.RFC3339Nano),
		NextNamespaceGeneration:  request.NextNamespaceGeneration,
		NextL1SecurityGeneration: request.NextL1SecurityGeneration,
		ManagedClasses:           classes,
		ExternalObligations:      obligations,
	}
	fingerprint, err := json.Marshal(struct {
		DeletionID               string
		TokenKeyVersion          string
		TenantToken              string
		RepositoryToken          string
		TrustDomainToken         string
		NextNamespaceGeneration  int64
		NextL1SecurityGeneration uint64
		ManagedClasses           []string
		ExternalObligations      []ExternalObligation
	}{
		DeletionID:               tombstone.DeletionID,
		TokenKeyVersion:          tombstone.TokenKeyVersion,
		TenantToken:              tombstone.TenantToken,
		RepositoryToken:          tombstone.RepositoryToken,
		TrustDomainToken:         tombstone.TrustDomainToken,
		NextNamespaceGeneration:  tombstone.NextNamespaceGeneration,
		NextL1SecurityGeneration: tombstone.NextL1SecurityGeneration,
		ManagedClasses:           tombstone.ManagedClasses,
		ExternalObligations:      tombstone.ExternalObligations,
	})
	if err != nil {
		return DeletionTombstone{}, errors.New("delete managed data: encode request fingerprint")
	}
	digest := sha256.Sum256(fingerprint)
	tombstone.RequestDigest = "sha256:" + hex.EncodeToString(digest[:])
	return tombstone, nil
}

// InspectManagedPath returns deletion state for a marked data root itself or
// one direct component beneath it. Unmarked paths remain ordinary local paths.
func InspectManagedPath(path string) (ManagedBoundary, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
		return ManagedBoundary{}, errors.New("inspect managed data: invalid path")
	}
	for _, candidate := range []string{path, filepath.Dir(path)} {
		marker := filepath.Join(candidate, deploymentDataMarkerName)
		if _, err := os.Lstat(marker); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return ManagedBoundary{}, errors.New("inspect managed data: marker unavailable")
		}
		if err := validateDataRoot(candidate); err != nil {
			return ManagedBoundary{}, fmt.Errorf("inspect managed data: %w", err)
		}
		tombstone, exists, err := loadDeletionTombstone(candidate)
		if err != nil {
			return ManagedBoundary{}, fmt.Errorf("inspect managed data: %w", err)
		}
		boundary := ManagedBoundary{Managed: true, DataRoot: candidate}
		if !exists {
			return boundary, nil
		}
		boundary.MinimumNamespaceGeneration = tombstone.NextNamespaceGeneration
		boundary.MinimumL1SecurityGeneration = tombstone.NextL1SecurityGeneration
		boundary.Revoked = tombstone.State == deletionStateRevoked
		return boundary, nil
	}
	return ManagedBoundary{}, nil
}

func validateDataRoot(root string) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return errors.New("data root must be an absolute non-root path")
	}
	if err := requirePrivateDirectory(root); err != nil {
		return err
	}
	markerPath := filepath.Join(root, deploymentDataMarkerName)
	content, err := readPrivateFile(markerPath, 4096)
	if err != nil {
		return errors.New("deployment data marker is unavailable")
	}
	var marker struct {
		DeploymentRoot string `json:"deploymentRoot"`
		SchemaVersion  string `json:"schemaVersion"`
	}
	if err := decodeStrictJSON(content, &marker); err != nil ||
		marker.SchemaVersion != "buildopt.dev/deployment-data/v1" ||
		!filepath.IsAbs(marker.DeploymentRoot) || marker.DeploymentRoot == "/" ||
		filepath.Clean(marker.DeploymentRoot) != marker.DeploymentRoot {
		return errors.New("deployment data marker is invalid")
	}
	return nil
}

func validateDataRootEntries(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return errors.New("read deployment data root")
	}
	allowed := map[string]struct{}{
		deploymentDataMarkerName: {},
		deletionTombstoneName:    {},
		deletionTemporaryName:    {},
		lifecycleLockName:        {},
		"audit":                  {},
	}
	for _, component := range managedComponents {
		allowed[component.name] = struct{}{}
		allowed[filepath.Base(stagedComponentPath(root, component.name))] = struct{}{}
	}
	for _, entry := range entries {
		if _, ok := allowed[entry.Name()]; !ok {
			return fmt.Errorf("unexpected path in deployment data root: %s", entry.Name())
		}
		path := filepath.Join(root, entry.Name())
		if entry.Name() == deploymentDataMarkerName ||
			entry.Name() == deletionTombstoneName ||
			entry.Name() == deletionTemporaryName ||
			entry.Name() == lifecycleLockName {
			if err := requirePrivateFile(path); err != nil {
				return err
			}
			continue
		}
		if err := requirePrivateDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func acquireComponentLeases(root string) ([]*os.File, error) {
	var leases []*os.File
	releaseOnError := func(err error) ([]*os.File, error) {
		releasePrivateLocks(leases)
		return nil, err
	}
	writerPath := filepath.Join(root, "shared", "writer.lock")
	if _, err := os.Lstat(writerPath); err == nil {
		lease, lockErr := acquireExistingPrivateLock(writerPath)
		if lockErr != nil {
			return releaseOnError(lockErr)
		}
		leases = append(leases, lease)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return releaseOnError(err)
	}
	locksRoot := filepath.Join(root, "l1", "locks")
	entries, err := os.ReadDir(locksRoot)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return releaseOnError(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return releaseOnError(errors.New("managed L1 lock entry is not a file"))
		}
		lease, lockErr := acquireExistingPrivateLock(
			filepath.Join(locksRoot, entry.Name()),
		)
		if lockErr != nil {
			return releaseOnError(lockErr)
		}
		leases = append(leases, lease)
	}
	return leases, nil
}

func acquirePrivateLock(path string) (*os.File, error) {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW,
		0o600,
	)
	if err != nil {
		return nil, err
	}
	if err := requirePrivateOpenFile(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrManagedDataBusy
		}
		return nil, err
	}
	return file, nil
}

func acquireExistingPrivateLock(path string) (*os.File, error) {
	if err := requirePrivateFile(path); err != nil {
		return nil, err
	}
	return acquirePrivateLock(path)
}

func releasePrivateLocks(files []*os.File) {
	for index := len(files) - 1; index >= 0; index-- {
		releasePrivateLock(files[index])
	}
}

func releasePrivateLock(file *os.File) {
	if file == nil {
		return
	}
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	_ = file.Close()
}

func stagedComponentPath(root string, name string) string {
	return filepath.Join(root, ".buildopt-deleting-"+name)
}

func removeVerifiedStagedTree(
	root string,
	path string,
	removeAll func(string) error,
) error {
	if removeAll == nil || filepath.Dir(path) != root ||
		!strings.HasPrefix(filepath.Base(path), ".buildopt-deleting-") {
		return errors.New("delete managed data: unsafe staged path")
	}
	if err := requirePrivateDirectory(path); err != nil {
		return err
	}
	if err := removeAll(path); err != nil {
		return fmt.Errorf("delete managed data: remove staged component: %w", err)
	}
	return nil
}

func privateDirectoryExists(path string) (bool, error) {
	if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, requirePrivateDirectory(path)
}

func writeDeletionTombstone(root string, tombstone DeletionTombstone) error {
	content, err := json.MarshalIndent(tombstone, "", "  ")
	if err != nil {
		return errors.New("encode deletion tombstone")
	}
	content = append(content, '\n')
	temporaryPath := filepath.Join(root, deletionTemporaryName)
	if _, err := os.Lstat(temporaryPath); err == nil {
		if err := requirePrivateFile(temporaryPath); err != nil {
			return err
		}
		if err := os.Remove(temporaryPath); err != nil {
			return errors.New("remove stale deletion tombstone temporary file")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(
		temporaryPath,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return errors.New("create deletion tombstone temporary file")
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return errors.New("write deletion tombstone")
	}
	if err := file.Sync(); err != nil {
		return errors.New("sync deletion tombstone")
	}
	if err := file.Close(); err != nil {
		return errors.New("close deletion tombstone")
	}
	if err := os.Rename(
		temporaryPath,
		filepath.Join(root, deletionTombstoneName),
	); err != nil {
		return errors.New("publish deletion tombstone")
	}
	remove = false
	return syncPrivateDirectory(root)
}

func loadDeletionTombstone(
	root string,
) (DeletionTombstone, bool, error) {
	path := filepath.Join(root, deletionTombstoneName)
	content, err := readPrivateFile(path, 64<<10)
	if errors.Is(err, fs.ErrNotExist) {
		return DeletionTombstone{}, false, nil
	}
	if err != nil {
		return DeletionTombstone{}, false, err
	}
	var tombstone DeletionTombstone
	if err := decodeStrictJSON(content, &tombstone); err != nil {
		return DeletionTombstone{}, false, errors.New("invalid deletion tombstone")
	}
	if tombstone.SchemaVersion != deletionSchemaVersion ||
		!validIdentifier(tombstone.DeletionID) ||
		!validIdentifier(tombstone.TokenKeyVersion) ||
		!validSHA256(tombstone.RequestDigest) ||
		!validHMAC(tombstone.TenantToken) ||
		!validHMAC(tombstone.RepositoryToken) ||
		!validHMAC(tombstone.TrustDomainToken) ||
		(tombstone.State != deletionStateRevoked &&
			tombstone.State != deletionStateComplete) ||
		tombstone.NextNamespaceGeneration < 1 ||
		tombstone.NextL1SecurityGeneration < 1 {
		return DeletionTombstone{}, false, errors.New("invalid deletion tombstone")
	}
	requestedAt, requestErr := time.Parse(time.RFC3339Nano, tombstone.RequestedAt)
	retainUntil, retainErr := time.Parse(time.RFC3339Nano, tombstone.RetainUntil)
	if requestErr != nil || retainErr != nil ||
		retainUntil.Sub(requestedAt) != deletionRetention ||
		(tombstone.State == deletionStateComplete) !=
			(tombstone.PhysicalCompletedAt != "") {
		return DeletionTombstone{}, false, errors.New("invalid deletion tombstone times")
	}
	if tombstone.PhysicalCompletedAt != "" {
		completedAt, err := time.Parse(
			time.RFC3339Nano,
			tombstone.PhysicalCompletedAt,
		)
		if err != nil || completedAt.Before(requestedAt) {
			return DeletionTombstone{}, false, errors.New("invalid deletion completion time")
		}
	}
	return tombstone, true, nil
}

func requirePrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || !info.IsDir() ||
		info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return fmt.Errorf("unsafe private directory: %s", path)
	}
	return nil
}

func requirePrivateFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || !info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 {
		return fmt.Errorf("unsafe private file: %s", path)
	}
	return nil
}

func requirePrivateOpenFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || !info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 {
		return errors.New("unsafe private lock file")
	}
	return nil
}

func readPrivateFile(path string, maximum int64) ([]byte, error) {
	if err := requirePrivateFile(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(content)) > maximum {
		return nil, errors.New("read bounded private file")
	}
	return content, nil
}

func decodeStrictJSON(content []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func syncPrivateDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return errors.New("open private directory for sync")
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return errors.New("sync private directory")
	}
	return nil
}

func validSHA256(value string) bool {
	return len(value) == 71 && strings.HasPrefix(value, "sha256:") &&
		isLowerHex(value[7:])
}

func validHMAC(value string) bool {
	return len(value) == 76 && strings.HasPrefix(value, "hmac-sha256:") &&
		isLowerHex(value[12:])
}

func isLowerHex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}
