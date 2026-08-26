package stickydecision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/filelock"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	localRecordsDirectory = "records"
	localCacheDirectory   = "cache"
	localHeadFile         = "head.json"
	localRevocationFile   = "revocation.json"
	localLockFile         = "state.lock"
	maximumRecordBytes    = 4 << 20
)

// StoreOptions configures the POC stores. Public keys are optional for
// observation/action records; when supplied, every decision is cryptographically
// verified before it can be persisted or selected.
type StoreOptions struct {
	PublicKeys map[string]ed25519.PublicKey
	Now        func() time.Time
}

// AppendResult identifies one committed immutable record and the new head.
type AppendResult struct {
	Generation   uint64
	RecordDigest string
	HeadDigest   string
	Replayed     bool
}

type localRequest struct {
	SchemaVersion  string       `json:"schemaVersion"`
	RecordType     string       `json:"recordType"`
	IdempotencyKey string       `json:"idempotencyKey"`
	RequestDigest  string       `json:"requestDigest"`
	Result         AppendResult `json:"result"`
}

// LocalStore is a private filesystem-backed control-plane store. Records are
// immutable files; only head.json and revocation.json are mutable pointers.
type LocalStore struct {
	root       string
	scope      string
	publicKeys map[string]ed25519.PublicKey
	now        func() time.Time
}

// OpenLocal creates or opens one repository-scoped local decision store.
func OpenLocal(root, scope string) (*LocalStore, error) {
	return OpenLocalWithOptions(root, scope, StoreOptions{})
}

// OpenLocalWithOptions is OpenLocal with an optional decision-key registry and
// deterministic clock used by the conformance fixtures.
func OpenLocalWithOptions(root, scope string, options StoreOptions) (*LocalStore, error) {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, fmt.Errorf("%w: local root must be one clean absolute path", ErrInvalidDocument)
	}
	if !digestPattern.MatchString(scope) {
		return nil, fmt.Errorf("%w: local repository scope is invalid", ErrCrossScope)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if err := ensurePrivateDirectory(root); err != nil {
		return nil, err
	}
	for _, directory := range []string{localRecordsDirectory, localCacheDirectory} {
		if err := ensurePrivateDirectory(filepath.Join(root, directory)); err != nil {
			return nil, err
		}
	}
	if file, err := os.OpenFile(filepath.Join(root, localLockFile), os.O_RDWR|os.O_CREATE, 0o600); err != nil {
		return nil, fmt.Errorf("open sticky decision lock: %w", err)
	} else {
		_ = file.Close()
	}
	return &LocalStore{root: root, scope: scope, publicKeys: cloneKeys(options.PublicKeys), now: options.Now}, nil
}

// Append validates and atomically publishes the next immutable local record.
// The idempotency key is part of the record and the request fingerprint, so a
// replay returns the original result while a changed request is rejected.
func (store *LocalStore) Append(
	ctx context.Context,
	raw []byte,
	expectedGeneration uint64,
	expectedHeadDigest string,
	idempotencyKey string,
) (AppendResult, error) {
	if err := validContext(ctx); err != nil {
		return AppendResult{}, err
	}
	if len(raw) == 0 || len(raw) > maximumRecordBytes || !digestPattern.MatchString(idempotencyKey) {
		return AppendResult{}, ErrInvalidDocument
	}
	document, err := DecodeDocument(raw, store.now().UTC())
	if err != nil {
		return AppendResult{}, err
	}
	if err := validateAppendIdentity(document, store.scope, expectedGeneration, expectedHeadDigest, idempotencyKey); err != nil {
		return AppendResult{}, err
	}
	lock, err := os.OpenFile(filepath.Join(store.root, localLockFile), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return AppendResult{}, fmt.Errorf("open sticky decision lock: %w", err)
	}
	defer lock.Close()
	if err := filelock.Try(lock, filelock.Exclusive); err != nil {
		if errors.Is(err, filelock.ErrBusy) {
			return AppendResult{}, ErrBusy
		}
		return AppendResult{}, err
	}
	defer filelock.Unlock(lock)

	revocationEpoch, err := store.loadRevocationUnlocked()
	if err != nil {
		return AppendResult{}, err
	}
	if document.Binding.RevocationEpoch != revocationEpoch {
		return AppendResult{}, ErrRevoked
	}
	if err := store.verifyDecision(document, store.now().UTC(), revocationEpoch); err != nil {
		return AppendResult{}, err
	}
	requestDigest := appendRequestDigest(raw, expectedGeneration, expectedHeadDigest, idempotencyKey)
	requestPath := filepath.Join(store.root, "requests", idempotencyKey+".json")
	if previous, readErr := os.ReadFile(requestPath); readErr == nil {
		var request localRequest
		if err := decodeStrict(previous, &request); err != nil || request.RequestDigest != requestDigest || request.IdempotencyKey != idempotencyKey {
			return AppendResult{}, ErrIdempotencyConflict
		}
		result := request.Result
		result.Replayed = true
		return result, nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return AppendResult{}, fmt.Errorf("read sticky decision idempotency record: %w", readErr)
	}
	if err := ensurePrivateDirectory(filepath.Join(store.root, "requests")); err != nil {
		return AppendResult{}, err
	}
	current, currentErr := store.loadHeadUnlocked()
	if errors.Is(currentErr, ErrNotFound) {
		if expectedGeneration != 0 || expectedHeadDigest != "" {
			return AppendResult{}, ErrGenerationConflict
		}
	} else if currentErr != nil {
		return AppendResult{}, currentErr
	} else if current.Head.Generation != expectedGeneration || current.HeadDigest != expectedHeadDigest {
		return AppendResult{}, ErrGenerationConflict
	}
	if document.StoreGeneration != expectedGeneration+1 {
		return AppendResult{}, ErrGenerationConflict
	}
	if currentErr == nil && current.Head.RevocationEpoch != revocationEpoch {
		return AppendResult{}, ErrRevoked
	}
	if currentErr == nil && current.Document.Binding.RepositoryScopeSHA256 != store.scope {
		return AppendResult{}, ErrCrossScope
	}

	recordPath := filepath.Join(store.root, localRecordsDirectory, recordFilename(document.StoreGeneration, document.Digest))
	if err := writeImmutable(recordPath, document.Raw, 0o600); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return AppendResult{}, fmt.Errorf("publish sticky decision record: %w", err)
		}
		previous, readErr := os.ReadFile(recordPath)
		if readErr != nil || !bytes.Equal(previous, document.Raw) {
			return AppendResult{}, ErrCorrupt
		}
	}
	head := StateHead{
		SchemaVersion: StateHeadSchemaVersion, RecordType: StateHeadRecordType,
		Plane: ControlPlane, RepositoryScopeSHA256: store.scope,
		Generation: document.StoreGeneration, RecordTypeAtHead: document.RecordType,
		RecordDigest: document.Digest, RevocationEpoch: revocationEpoch,
		UpdatedAt: store.now().UTC().Format(time.RFC3339Nano),
	}
	headRaw, headDigest, err := CanonicalDocument(head)
	if err != nil {
		return AppendResult{}, err
	}
	if err := replaceFile(filepath.Join(store.root, localHeadFile), headRaw, 0o600); err != nil {
		return AppendResult{}, fmt.Errorf("publish sticky decision head: %w", err)
	}
	result := AppendResult{Generation: head.Generation, RecordDigest: document.Digest, HeadDigest: headDigest}
	requestRaw, _, err := CanonicalDocument(localRequest{
		SchemaVersion: StateHeadSchemaVersion, RecordType: "STICKY_STATE_CAS",
		IdempotencyKey: idempotencyKey, RequestDigest: requestDigest, Result: result,
	})
	if err != nil {
		return AppendResult{}, err
	}
	if err := writeImmutable(requestPath, requestRaw, 0o600); err != nil && !errors.Is(err, os.ErrExist) {
		return AppendResult{}, fmt.Errorf("publish sticky decision idempotency record: %w", err)
	}
	return result, nil
}

// Current verifies and returns the current local record. Expiry, revocation,
// corruption and scope mismatches are fail-closed errors.
func (store *LocalStore) Current(ctx context.Context) (HeadSnapshot, error) {
	if err := validContext(ctx); err != nil {
		return HeadSnapshot{}, err
	}
	lock, err := os.OpenFile(filepath.Join(store.root, localLockFile), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return HeadSnapshot{}, err
	}
	defer lock.Close()
	if err := filelock.Try(lock, filelock.Shared); err != nil {
		if errors.Is(err, filelock.ErrBusy) {
			return HeadSnapshot{}, ErrBusy
		}
		return HeadSnapshot{}, err
	}
	defer filelock.Unlock(lock)
	snapshot, err := store.loadHeadUnlocked()
	if err != nil {
		return HeadSnapshot{}, err
	}
	revocationEpoch, err := store.loadRevocationUnlocked()
	if err != nil {
		return HeadSnapshot{}, err
	}
	if snapshot.Head.RevocationEpoch != revocationEpoch || snapshot.Document.Binding.RevocationEpoch != revocationEpoch {
		return HeadSnapshot{}, ErrRevoked
	}
	if err := store.verifyDecision(snapshot.Document, store.now().UTC(), revocationEpoch); err != nil {
		return HeadSnapshot{}, err
	}
	return snapshot, nil
}

// Revoke advances the local revocation epoch. It never rewrites an immutable
// record; all later Current and Append calls reject older bindings.
func (store *LocalStore) Revoke(ctx context.Context, epoch int64) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	if epoch < 1 {
		return ErrInvalidDocument
	}
	lock, err := os.OpenFile(filepath.Join(store.root, localLockFile), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := filelock.Try(lock, filelock.Exclusive); err != nil {
		if errors.Is(err, filelock.ErrBusy) {
			return ErrBusy
		}
		return err
	}
	defer filelock.Unlock(lock)
	current, err := store.loadRevocationUnlocked()
	if err != nil {
		return err
	}
	if epoch <= current {
		return ErrGenerationConflict
	}
	revocation := Revocation{
		SchemaVersion: RevocationSchemaVersion, RecordType: RevocationRecordType,
		RepositoryScopeSHA256: store.scope, Epoch: epoch,
		UpdatedAt: store.now().UTC().Format(time.RFC3339Nano),
	}
	raw, _, err := CanonicalDocument(revocation)
	if err != nil {
		return err
	}
	return replaceFile(filepath.Join(store.root, localRevocationFile), raw, 0o600)
}

// PutCacheObject writes bytes into the separate local Gradle data-plane test
// namespace. Current and decision selectors never inspect this directory.
func (store *LocalStore) PutCacheObject(key string, raw []byte) (string, error) {
	if key == "" || strings.ContainsAny(key, `/\\`) || len(raw) == 0 || len(raw) > maximumRecordBytes {
		return "", ErrCrossPlane
	}
	digest := digestBytes(raw)
	path := filepath.Join(store.root, localCacheDirectory, key+"-"+digest)
	if err := writeImmutable(path, raw, 0o600); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	return digest, nil
}

// OpenCacheObject proves that data-plane bytes are independently addressable;
// it is intentionally not part of Current or any decision authorization API.
func (store *LocalStore) OpenCacheObject(key, digest string) ([]byte, error) {
	if key == "" || strings.ContainsAny(key, `/\\`) || !digestPattern.MatchString(digest) {
		return nil, ErrCrossPlane
	}
	return os.ReadFile(filepath.Join(store.root, localCacheDirectory, key+"-"+digest))
}

func (store *LocalStore) loadHeadUnlocked() (HeadSnapshot, error) {
	raw, err := os.ReadFile(filepath.Join(store.root, localHeadFile))
	if errors.Is(err, os.ErrNotExist) {
		return HeadSnapshot{}, ErrNotFound
	}
	if err != nil {
		return HeadSnapshot{}, err
	}
	canonicalRaw, err := equalCanonical(raw)
	if err != nil {
		return HeadSnapshot{}, ErrCorrupt
	}
	var head StateHead
	if err := decodeStrict(canonicalRaw, &head); err != nil || head.SchemaVersion != StateHeadSchemaVersion || head.RecordType != StateHeadRecordType || head.Plane != ControlPlane || head.RepositoryScopeSHA256 != store.scope || head.Generation == 0 || !digestPattern.MatchString(head.RecordDigest) || head.RevocationEpoch < 0 || !contractcryptoValidTimestamp(head.UpdatedAt) {
		return HeadSnapshot{}, ErrCorrupt
	}
	headDigest := digestBytes(canonicalRaw)
	recordRaw, err := os.ReadFile(filepath.Join(store.root, localRecordsDirectory, recordFilename(head.Generation, head.RecordDigest)))
	if err != nil {
		return HeadSnapshot{}, ErrCorrupt
	}
	document, err := DecodeDocument(recordRaw, store.now().UTC())
	if err != nil || document.Digest != head.RecordDigest || document.StoreGeneration != head.Generation || document.RecordType != head.RecordTypeAtHead || document.Binding.RepositoryScopeSHA256 != store.scope {
		if errors.Is(err, ErrExpired) {
			return HeadSnapshot{}, ErrExpired
		}
		return HeadSnapshot{}, ErrCorrupt
	}
	return HeadSnapshot{Head: head, HeadDigest: headDigest, Document: document, RecordDigest: document.Digest}, nil
}

func (store *LocalStore) loadRevocationUnlocked() (int64, error) {
	raw, err := os.ReadFile(filepath.Join(store.root, localRevocationFile))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var revocation Revocation
	if err := decodeStrict(raw, &revocation); err != nil || revocation.SchemaVersion != RevocationSchemaVersion || revocation.RecordType != RevocationRecordType || revocation.RepositoryScopeSHA256 != store.scope || revocation.Epoch < 1 || !contractcryptoValidTimestamp(revocation.UpdatedAt) {
		return 0, ErrCorrupt
	}
	return revocation.Epoch, nil
}

func (store *LocalStore) verifyDecision(document Document, now time.Time, epoch int64) error {
	if document.Decision == nil || len(store.publicKeys) == 0 {
		return nil
	}
	_, err := VerifyDecision(context.Background(), document.Raw, store.publicKeys, epoch, now)
	return err
}

// CentralStore adapts the existing Shared typed-state SQLite lifecycle to the
// sticky records. It stores each canonical record as one EVIDENCE artifact and
// advances the existing generation CAS head; Gradle blobs remain separate.
type CentralStore struct {
	storage    *sharedcache.Storage
	scope      string
	publicKeys map[string]ed25519.PublicKey
	now        func() time.Time
	mu         sync.RWMutex
	revocation int64
}

// NewCentralStore creates a repository-scoped adapter over an existing Shared
// storage root. The Shared writer remains owned by its caller.
func NewCentralStore(storage *sharedcache.Storage, scope string, options StoreOptions) (*CentralStore, error) {
	if storage == nil {
		return nil, errors.New("sticky central store requires Shared storage")
	}
	if !digestPattern.MatchString(scope) {
		return nil, ErrCrossScope
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &CentralStore{storage: storage, scope: scope, publicKeys: cloneKeys(options.PublicKeys), now: options.Now}, nil
}

// Append publishes one canonical record through the existing typed state
// object/manifest/CAS sequence. A CAS conflict leaves the object invisible.
func (store *CentralStore) Append(ctx context.Context, raw []byte, expectedGeneration uint64, expectedHeadDigest, idempotencyKey string) (AppendResult, error) {
	if err := validContext(ctx); err != nil {
		return AppendResult{}, err
	}
	if len(raw) == 0 || len(raw) > maximumRecordBytes || !digestPattern.MatchString(idempotencyKey) {
		return AppendResult{}, ErrInvalidDocument
	}
	store.mu.RLock()
	revocationEpoch := store.revocation
	store.mu.RUnlock()
	document, err := DecodeDocument(raw, store.now().UTC())
	if err != nil {
		return AppendResult{}, err
	}
	if err := validateAppendIdentity(document, store.scope, expectedGeneration, expectedHeadDigest, idempotencyKey); err != nil {
		return AppendResult{}, err
	}
	if document.Binding.RevocationEpoch != revocationEpoch {
		return AppendResult{}, ErrRevoked
	}
	if err := store.verifyDecision(document, store.now().UTC(), revocationEpoch); err != nil {
		return AppendResult{}, err
	}
	_, currentErr := store.Current(ctx)
	if errors.Is(currentErr, ErrNotFound) {
		if expectedGeneration != 0 || expectedHeadDigest != "" {
			return AppendResult{}, ErrGenerationConflict
		}
	} else if currentErr != nil {
		return AppendResult{}, currentErr
	}
	if document.StoreGeneration != expectedGeneration+1 {
		return AppendResult{}, ErrGenerationConflict
	}
	manifest := stateManifestForDocument(document, store.now().UTC())
	manifestRaw, manifestDigest, err := sharedcache.CanonicalCentralStateValue(manifest)
	if err != nil {
		return AppendResult{}, err
	}
	if _, _, err := store.storage.PutStateObject(ctx, store.scope, sharedcache.StateKindEvidence, document.Digest, bytes.NewReader(document.Raw)); err != nil {
		return AppendResult{}, err
	}
	if _, _, err := store.storage.PutStateManifest(ctx, manifestRaw); err != nil {
		return AppendResult{}, err
	}
	cas, err := store.storage.CASStateHead(ctx, sharedcache.StateCASRequest{
		RepositoryScopeSHA256: store.scope, Kind: sharedcache.StateKindEvidence,
		IdempotencyKey: idempotencyKey, ExpectedGeneration: int64(expectedGeneration),
		ExpectedHeadSHA256: expectedHeadDigest, ManifestSHA256: manifestDigest,
	})
	if err != nil {
		if errors.Is(err, sharedcache.ErrStateHeadPrecondition) || errors.Is(err, sharedcache.ErrStateGenerationConflict) {
			return AppendResult{}, ErrGenerationConflict
		}
		if errors.Is(err, sharedcache.ErrStateIdempotency) {
			return AppendResult{}, ErrIdempotencyConflict
		}
		return AppendResult{}, err
	}
	return AppendResult{Generation: uint64(cas.Head.Generation), RecordDigest: document.Digest, HeadDigest: cas.HeadSHA256, Replayed: cas.Replayed}, nil
}

// Current loads and verifies the current typed-state record. The current
// revocation epoch is process-local in this adapter and is normally supplied
// by the owner-operated Shared authority; the local HTTP path persists it.
func (store *CentralStore) Current(ctx context.Context) (HeadSnapshot, error) {
	if err := validContext(ctx); err != nil {
		return HeadSnapshot{}, err
	}
	snapshot, err := store.storage.LoadCurrentState(ctx, store.scope, sharedcache.StateKindEvidence)
	if errors.Is(err, sharedcache.ErrStateNotFound) {
		return HeadSnapshot{}, ErrNotFound
	}
	if err != nil {
		return HeadSnapshot{}, err
	}
	file, err := store.storage.OpenStateObject(ctx, store.scope, sharedcache.StateKindEvidence, strings.TrimPrefix(snapshot.Manifest.Artifacts[0].SHA256, "sha256:"))
	if err != nil {
		return HeadSnapshot{}, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maximumRecordBytes+1))
	if err != nil || len(raw) > maximumRecordBytes {
		return HeadSnapshot{}, ErrCorrupt
	}
	document, err := DecodeDocument(raw, store.now().UTC())
	if err != nil {
		return HeadSnapshot{}, err
	}
	if document.StoreGeneration != uint64(snapshot.Head.Generation) || document.Digest != strings.TrimPrefix(snapshot.Manifest.Artifacts[0].SHA256, "sha256:") || document.Binding.RepositoryScopeSHA256 != store.scope {
		return HeadSnapshot{}, ErrCorrupt
	}
	store.mu.RLock()
	revocationEpoch := store.revocation
	store.mu.RUnlock()
	if document.Binding.RevocationEpoch != revocationEpoch {
		return HeadSnapshot{}, ErrRevoked
	}
	if err := store.verifyDecision(document, store.now().UTC(), revocationEpoch); err != nil {
		return HeadSnapshot{}, err
	}
	head := StateHead{
		SchemaVersion: StateHeadSchemaVersion, RecordType: StateHeadRecordType,
		Plane: ControlPlane, RepositoryScopeSHA256: store.scope,
		Generation: uint64(snapshot.Head.Generation), RecordTypeAtHead: document.RecordType,
		RecordDigest: document.Digest, RevocationEpoch: revocationEpoch,
		UpdatedAt: snapshot.Head.UpdatedAt,
	}
	return HeadSnapshot{Head: head, HeadDigest: snapshot.HeadSHA256, Document: document, RecordDigest: document.Digest}, nil
}

// Revoke advances the central adapter's authority epoch. In a deployed server
// this value is read from the existing signed authority/revocation endpoint;
// keeping it explicit here makes the fail-closed POC behavior testable without
// inventing a second revocation database.
func (store *CentralStore) Revoke(ctx context.Context, epoch int64) error {
	if err := validContext(ctx); err != nil {
		return err
	}
	if epoch < 1 {
		return ErrInvalidDocument
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if epoch <= store.revocation {
		return ErrGenerationConflict
	}
	store.revocation = epoch
	return nil
}

// PutCacheObject writes an opaque Gradle data-plane blob without creating a
// typed-state object or head. The returned digest cannot authorize a decision.
func (store *CentralStore) PutCacheObject(ctx context.Context, raw []byte) (sharedcache.Blob, error) {
	if err := validContext(ctx); err != nil {
		return sharedcache.Blob{}, err
	}
	if len(raw) == 0 || len(raw) > maximumRecordBytes {
		return sharedcache.Blob{}, ErrCrossPlane
	}
	blob, _, err := store.storage.Blobs().Put(ctx, bytes.NewReader(raw))
	return blob, err
}

func (store *CentralStore) verifyDecision(document Document, now time.Time, epoch int64) error {
	if document.Decision == nil || len(store.publicKeys) == 0 {
		return nil
	}
	_, err := VerifyDecision(context.Background(), document.Raw, store.publicKeys, epoch, now)
	return err
}

func stateManifestForDocument(document Document, now time.Time) sharedcache.StateManifest {
	compatibility, _, _ := CanonicalDocument(struct {
		Workflow             string `json:"workflow"`
		GradleVersion        string `json:"gradleVersion"`
		WrapperSHA256        string `json:"wrapperSha256"`
		OptionsSHA256        string `json:"optionsSha256"`
		OutputContractSHA256 string `json:"outputContractSha256"`
	}{document.Binding.Workflow, document.Binding.GradleVersion, document.Binding.WrapperSHA256, document.Binding.OptionsSHA256, document.Binding.OutputContractSHA256})
	bindings, _, _ := CanonicalDocument(document.Binding)
	return sharedcache.StateManifest{
		SchemaVersion: "buildopt.central/state-manifest/v1", RecordType: "CENTRAL_STATE_MANIFEST",
		Kind: sharedcache.StateKindEvidence, RepositoryScopeSHA256: document.Binding.RepositoryScopeSHA256,
		Generation: int64(document.StoreGeneration), CompatibilitySHA256: digestBytes(compatibility), BindingsSHA256: digestBytes(bindings),
		Origin: sharedcache.StateOrigin{
			BaseRevision: document.Binding.SourceRevision, TargetRevision: document.Binding.SourceRevision,
			BuildOptExecutableSHA256: document.Binding.BuildOptExecutableSHA256, WrapperSHA256: document.Binding.WrapperSHA256,
			GradleVersion: document.Binding.GradleVersion,
		},
		Artifacts: []sharedcache.StateArtifact{{Role: "CALIBRATION_EVIDENCE", SHA256: document.Digest, SizeBytes: int64(len(document.Raw)), PayloadSchemaVersion: "buildopt.sticky/record/v1"}},
		Status:    "COMPLETE", RetentionClass: "WHILE_REFERENCED_PLUS_30_DAYS", CreatedAt: now.Format(time.RFC3339Nano),
		Authority: sharedcache.StateAuthority{SelectionRequiresLocalRevalidation: true, ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE"},
	}
}

func validateAppendIdentity(document Document, scope string, expectedGeneration uint64, expectedHeadDigest, idempotencyKey string) error {
	if !document.HasControlPlaneIdentity() {
		return ErrCrossPlane
	}
	if document.Binding.RepositoryScopeSHA256 != scope {
		return ErrCrossScope
	}
	if !digestPattern.MatchString(idempotencyKey) || document.IdempotencyKey != idempotencyKey {
		return ErrIdempotencyConflict
	}
	if expectedGeneration == 0 && expectedHeadDigest != "" {
		return ErrGenerationConflict
	}
	if expectedGeneration > 0 && !digestPattern.MatchString(expectedHeadDigest) {
		return ErrGenerationConflict
	}
	return nil
}

func appendRequestDigest(raw []byte, expectedGeneration uint64, expectedHeadDigest, idempotencyKey string) string {
	value := append([]byte(nil), raw...)
	value = append(value, 0)
	value = append(value, []byte(strconv.FormatUint(expectedGeneration, 10))...)
	value = append(value, 0)
	value = append(value, []byte(expectedHeadDigest)...)
	value = append(value, 0)
	value = append(value, []byte(idempotencyKey)...)
	return digestBytes(value)
}

func recordFilename(generation uint64, digest string) string {
	return strconv.FormatUint(generation, 20) + "-" + digest + ".json"
}

func validContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidDocument
	}
	return ctx.Err()
}

func cloneKeys(input map[string]ed25519.PublicKey) map[string]ed25519.PublicKey {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]ed25519.PublicKey, len(input))
	for key, value := range input {
		output[key] = append(ed25519.PublicKey(nil), value...)
	}
	return output
}

func ensurePrivateDirectory(path string) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("sticky decision path is not a private directory: %s", path)
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return err
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}

func writeImmutable(path string, raw []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func replaceFile(path string, raw []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".sticky-state-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
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
	return os.Rename(temporaryPath, path)
}

func contractcryptoValidTimestamp(value string) bool {
	return contractcrypto.ValidUTCTimestamp(value)
}
