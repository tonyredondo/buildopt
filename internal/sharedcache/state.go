package sharedcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	// StateKindPortfolio stores qualified structural-profile portfolios.
	StateKindPortfolio StateKind = "PORTFOLIO"
	// StateKindEvidence stores immutable qualification evidence.
	StateKindEvidence StateKind = "EVIDENCE"
	// StateKindCheckpoint stores bounded resumable calibration state.
	StateKindCheckpoint StateKind = "CHECKPOINT"

	maximumStateManifestBytes = 1 << 20
	maximumStateArtifactBytes = 16 << 20
	maximumStateGeneration    = 9007199254740991
	stagedStateTTL            = 24 * time.Hour
	supersededStateTTL        = 30 * 24 * time.Hour
)

var (
	// ErrStateNotFound means no visible typed state owns the requested identity.
	ErrStateNotFound = errors.New("BuildOpt typed state was not found")
	// ErrStateInvalid means a typed-state request violates the contract.
	ErrStateInvalid = errors.New("BuildOpt typed state is invalid")
	// ErrStateDigestMismatch means uploaded bytes do not match their address.
	ErrStateDigestMismatch = errors.New("BuildOpt typed state digest mismatch")
	// ErrStateManifestIncomplete means a manifest references unavailable state.
	ErrStateManifestIncomplete = errors.New("BuildOpt typed state manifest is incomplete")
	// ErrStateGenerationConflict means a manifest skips or reuses a generation.
	ErrStateGenerationConflict = errors.New("BuildOpt typed state generation conflicts")
	// ErrStateHeadPrecondition means another writer owns the expected head.
	ErrStateHeadPrecondition = errors.New("BuildOpt typed state head precondition failed")
	// ErrStateIdempotency means one key was reused for a changed CAS request.
	ErrStateIdempotency = errors.New("BuildOpt typed state idempotency conflict")
	// ErrStateCorrupt means durable metadata or bytes failed revalidation.
	ErrStateCorrupt = errors.New("BuildOpt typed state is corrupt")
)

var (
	stateRevisionPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	stateGradlePattern   = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`)
	stateSchemaPattern   = regexp.MustCompile(`^buildopt\.[a-z0-9.-]+/[a-z0-9.-]+/v[1-9][0-9]*$`)
)

// StateKind is one typed BuildOpt namespace. It never aliases a Gradle cache
// key, even when both namespaces happen to contain identical physical bytes.
type StateKind string

// StateAuthority keeps locally revalidated POC state from becoming production
// or Test Optimization authority merely because it is durable.
type StateAuthority struct {
	SelectionRequiresLocalRevalidation bool   `json:"selectionRequiresLocalRevalidation"`
	ProductionAuthorized               bool   `json:"productionAuthorized"`
	TestOptimization                   string `json:"testOptimization"`
}

// StateOrigin binds an immutable generation to the exact BuildOpt and Gradle
// inputs that produced it.
type StateOrigin struct {
	BaseRevision             string `json:"baseRevision"`
	TargetRevision           string `json:"targetRevision"`
	BuildOptExecutableSHA256 string `json:"buildoptExecutableSha256"`
	WrapperSHA256            string `json:"wrapperSha256"`
	GradleVersion            string `json:"gradleVersion"`
}

// StateArtifact is one content-addressed payload required by a manifest.
type StateArtifact struct {
	Role                 string `json:"role"`
	SHA256               string `json:"sha256"`
	SizeBytes            int64  `json:"sizeBytes"`
	PayloadSchemaVersion string `json:"payloadSchemaVersion"`
}

// StateReference keeps portfolio qualification evidence reachable for as long
// as the portfolio remains retained.
type StateReference struct {
	Kind           StateKind `json:"kind"`
	ManifestSHA256 string    `json:"manifestSha256"`
	Relation       string    `json:"relation"`
}

// StateManifest is the immutable typed-state publication unit.
type StateManifest struct {
	SchemaVersion         string           `json:"schemaVersion"`
	RecordType            string           `json:"recordType"`
	Kind                  StateKind        `json:"kind"`
	RepositoryScopeSHA256 string           `json:"repositoryScopeSha256"`
	Generation            int64            `json:"generation"`
	CompatibilitySHA256   string           `json:"compatibilitySha256"`
	BindingsSHA256        string           `json:"bindingsSha256"`
	Origin                StateOrigin      `json:"origin"`
	Artifacts             []StateArtifact  `json:"artifacts"`
	References            []StateReference `json:"references"`
	Status                string           `json:"status"`
	RetentionClass        string           `json:"retentionClass"`
	CreatedAt             string           `json:"createdAt"`
	ExpiresAt             string           `json:"expiresAt,omitempty"`
	Authority             StateAuthority   `json:"authority"`
}

// StateHead is the sole mutable pointer for one repository and state kind.
type StateHead struct {
	SchemaVersion          string         `json:"schemaVersion"`
	RecordType             string         `json:"recordType"`
	Kind                   StateKind      `json:"kind"`
	RepositoryScopeSHA256  string         `json:"repositoryScopeSha256"`
	Generation             int64          `json:"generation"`
	ManifestSHA256         string         `json:"manifestSha256"`
	PreviousManifestSHA256 string         `json:"previousManifestSha256,omitempty"`
	CompatibilitySHA256    string         `json:"compatibilitySha256"`
	UpdatedAt              string         `json:"updatedAt"`
	Authority              StateAuthority `json:"authority"`
}

// StateObject identifies a namespaced CAS object after its bytes are verified.
type StateObject struct {
	RepositoryScopeSHA256 string
	Kind                  StateKind
	SHA256                string
	SizeBytes             int64
}

// StateCASRequest advances one head by exactly one immutable generation.
type StateCASRequest struct {
	RepositoryScopeSHA256 string
	Kind                  StateKind
	IdempotencyKey        string
	ExpectedGeneration    int64
	ExpectedHeadSHA256    string
	ManifestSHA256        string
}

// StateCASResult records either the newly committed head or an exact replay.
type StateCASResult struct {
	Head       StateHead
	HeadSHA256 string
	Replayed   bool
}

// StateSnapshot is a fully verified current head and manifest. Callers must
// still revalidate compatibility and repository bindings before selection.
type StateSnapshot struct {
	Head           StateHead
	HeadSHA256     string
	Manifest       StateManifest
	ManifestSHA256 string
}

// StateMaintenanceReport describes bounded retention work; it makes no value
// or production-authority claim.
type StateMaintenanceReport struct {
	ExpiredCheckpoints      int
	ExpiredStagedManifests  int
	ExpiredSuperseded       int
	ExpiredStagedObjects    int
	DeletedUnreferencedBlob int
}

// PutStateObject verifies expectedSHA256 before granting visibility in exactly
// one repository/kind namespace. Physical CAS deduplication is independent.
func (storage *Storage) PutStateObject(
	ctx context.Context,
	repositoryScopeSHA256 string,
	kind StateKind,
	expectedSHA256 string,
	reader io.Reader,
) (StateObject, bool, error) {
	if ctx == nil || reader == nil || !validSHA256(repositoryScopeSHA256) ||
		!validStateKind(kind) || !validSHA256(expectedSHA256) {
		return StateObject{}, false, ErrStateInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return StateObject{}, false, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	blob, _, err := storage.blobs.putLocked(ctx, reader)
	if err != nil {
		return StateObject{}, false, err
	}
	if blob.Size < 1 || blob.Size > maximumStateArtifactBytes ||
		strings.TrimPrefix(blob.Digest, digestPrefix) != expectedSHA256 {
		return StateObject{}, false, ErrStateDigestMismatch
	}
	result, err := storage.state.database.ExecContext(
		ctx,
		`INSERT INTO state_objects (
    repository_scope_sha256, kind, blob_digest, size_bytes, created_at_unix_ms
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(repository_scope_sha256, kind, blob_digest) DO NOTHING`,
		repositoryScopeSHA256,
		kind,
		blob.Digest,
		blob.Size,
		storage.now().UnixMilli(),
	)
	if err != nil {
		return StateObject{}, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return StateObject{}, false, err
	}
	return StateObject{
		RepositoryScopeSHA256: repositoryScopeSHA256,
		Kind:                  kind,
		SHA256:                expectedSHA256,
		SizeBytes:             blob.Size,
	}, rows == 1, nil
}

// OpenStateObject returns verified bytes only when the exact repository and
// kind own metadata for the digest. The caller owns the returned file.
func (storage *Storage) OpenStateObject(
	ctx context.Context,
	repositoryScopeSHA256 string,
	kind StateKind,
	digest string,
) (*os.File, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) ||
		!validStateKind(kind) || !validSHA256(digest) {
		return nil, ErrStateInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return nil, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	var size int64
	err = storage.state.database.QueryRowContext(
		ctx,
		`SELECT size_bytes FROM state_objects
WHERE repository_scope_sha256 = ? AND kind = ? AND blob_digest = ?`,
		repositoryScopeSHA256,
		kind,
		digestPrefix+digest,
	).Scan(&size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStateNotFound
	}
	if err != nil {
		return nil, err
	}
	file, err := storage.blobs.openVerified(ctx, Blob{
		Digest: digestPrefix + digest,
		Size:   size,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStateCorrupt, err)
	}
	return file, nil
}

// PutStateManifest validates a complete immutable generation before making the
// manifest addressable. It does not change the current head.
func (storage *Storage) PutStateManifest(
	ctx context.Context,
	raw []byte,
) (StateSnapshot, bool, error) {
	if ctx == nil || len(raw) == 0 || len(raw) > maximumStateManifestBytes {
		return StateSnapshot{}, false, ErrStateInvalid
	}
	manifest, canonical, digest, err := decodeStateManifest(raw)
	if err != nil {
		return StateSnapshot{}, false, err
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return StateSnapshot{}, false, err
	}
	defer finish()
	transaction, err := storage.state.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return StateSnapshot{}, false, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()

	var existing []byte
	err = transaction.QueryRowContext(
		ctx,
		`SELECT canonical_document FROM state_manifests
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
		manifest.RepositoryScopeSHA256,
		manifest.Kind,
		digestPrefix+digest,
	).Scan(&existing)
	if err == nil {
		if !bytes.Equal(existing, canonical) {
			return StateSnapshot{}, false, ErrStateCorrupt
		}
		if err := transaction.Commit(); err != nil {
			return StateSnapshot{}, false, err
		}
		rollback = false
		return StateSnapshot{Manifest: manifest, ManifestSHA256: digest}, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return StateSnapshot{}, false, err
	}

	for _, artifact := range manifest.Artifacts {
		var size int64
		err := transaction.QueryRowContext(
			ctx,
			`SELECT size_bytes FROM state_objects
WHERE repository_scope_sha256 = ? AND kind = ? AND blob_digest = ?`,
			manifest.RepositoryScopeSHA256,
			manifest.Kind,
			digestPrefix+artifact.SHA256,
		).Scan(&size)
		if errors.Is(err, sql.ErrNoRows) || size != artifact.SizeBytes {
			return StateSnapshot{}, false, ErrStateManifestIncomplete
		}
		if err != nil {
			return StateSnapshot{}, false, err
		}
	}
	for _, reference := range manifest.References {
		var present int
		err := transaction.QueryRowContext(
			ctx,
			`SELECT 1 FROM state_manifests
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
			manifest.RepositoryScopeSHA256,
			reference.Kind,
			digestPrefix+reference.ManifestSHA256,
		).Scan(&present)
		if errors.Is(err, sql.ErrNoRows) {
			return StateSnapshot{}, false, ErrStateManifestIncomplete
		}
		if err != nil {
			return StateSnapshot{}, false, err
		}
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	var expiresAt any
	if manifest.ExpiresAt != "" {
		parsed, _ := time.Parse(time.RFC3339Nano, manifest.ExpiresAt)
		expiresAt = parsed.UnixMilli()
	}
	_, err = transaction.ExecContext(
		ctx,
		`INSERT INTO state_manifests (
    repository_scope_sha256, kind, generation, manifest_digest,
    canonical_document, compatibility_sha256, bindings_sha256, status,
    created_at_unix_ms, expires_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		manifest.RepositoryScopeSHA256,
		manifest.Kind,
		manifest.Generation,
		digestPrefix+digest,
		canonical,
		manifest.CompatibilitySHA256,
		manifest.BindingsSHA256,
		manifest.Status,
		createdAt.UnixMilli(),
		expiresAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return StateSnapshot{}, false, ErrStateGenerationConflict
		}
		return StateSnapshot{}, false, err
	}
	for _, artifact := range manifest.Artifacts {
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO state_manifest_artifacts (
    repository_scope_sha256, kind, manifest_digest, role, blob_digest, size_bytes
) VALUES (?, ?, ?, ?, ?, ?)`,
			manifest.RepositoryScopeSHA256,
			manifest.Kind,
			digestPrefix+digest,
			artifact.Role,
			digestPrefix+artifact.SHA256,
			artifact.SizeBytes,
		); err != nil {
			return StateSnapshot{}, false, err
		}
	}
	for _, reference := range manifest.References {
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO state_manifest_references (
    repository_scope_sha256, source_kind, source_manifest_digest,
    target_kind, target_manifest_digest, relation
) VALUES (?, ?, ?, ?, ?, ?)`,
			manifest.RepositoryScopeSHA256,
			manifest.Kind,
			digestPrefix+digest,
			reference.Kind,
			digestPrefix+reference.ManifestSHA256,
			reference.Relation,
		); err != nil {
			return StateSnapshot{}, false, err
		}
	}
	if err := transaction.Commit(); err != nil {
		return StateSnapshot{}, false, err
	}
	rollback = false
	return StateSnapshot{Manifest: manifest, ManifestSHA256: digest}, true, nil
}

// CASStateHead publishes exactly the next complete generation. Idempotency and
// the head update commit in one SQLite transaction.
func (storage *Storage) CASStateHead(
	ctx context.Context,
	request StateCASRequest,
) (StateCASResult, error) {
	if ctx == nil || !validSHA256(request.RepositoryScopeSHA256) ||
		!validStateKind(request.Kind) || !validSHA256(request.IdempotencyKey) ||
		!validSHA256(request.ManifestSHA256) || request.ExpectedGeneration < 0 ||
		request.ExpectedGeneration >= maximumStateGeneration ||
		(request.ExpectedGeneration == 0 && request.ExpectedHeadSHA256 != "") ||
		(request.ExpectedGeneration > 0 && !validSHA256(request.ExpectedHeadSHA256)) {
		return StateCASResult{}, ErrStateInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return StateCASResult{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	storage.stateCASMutex.Lock()
	defer storage.stateCASMutex.Unlock()

	fingerprint, err := stateCASFingerprint(request)
	if err != nil {
		return StateCASResult{}, err
	}
	transaction, err := storage.state.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return StateCASResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()

	var priorFingerprint, priorHeadDigest string
	var priorHead []byte
	err = transaction.QueryRowContext(
		ctx,
		`SELECT request_digest, head_digest, head_canonical_document
FROM state_cas_requests WHERE idempotency_key = ?`,
		request.IdempotencyKey,
	).Scan(&priorFingerprint, &priorHeadDigest, &priorHead)
	if err == nil {
		if priorFingerprint != digestPrefix+fingerprint {
			return StateCASResult{}, ErrStateIdempotency
		}
		head, err := decodeStateHead(priorHead)
		if err != nil || digestBytes(priorHead) != strings.TrimPrefix(priorHeadDigest, digestPrefix) {
			return StateCASResult{}, ErrStateCorrupt
		}
		if err := transaction.Commit(); err != nil {
			return StateCASResult{}, err
		}
		rollback = false
		return StateCASResult{
			Head: head, HeadSHA256: strings.TrimPrefix(priorHeadDigest, digestPrefix), Replayed: true,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return StateCASResult{}, err
	}

	manifest, err := loadStateManifestTx(
		ctx,
		transaction,
		request.RepositoryScopeSHA256,
		request.Kind,
		request.ManifestSHA256,
	)
	if errors.Is(err, ErrStateNotFound) {
		return StateCASResult{}, ErrStateManifestIncomplete
	}
	if err != nil {
		return StateCASResult{}, err
	}
	if manifest.Generation != request.ExpectedGeneration+1 {
		return StateCASResult{}, ErrStateGenerationConflict
	}
	if err := storage.verifyStateManifestForPublication(
		ctx,
		transaction,
		manifest,
		request.ManifestSHA256,
	); err != nil {
		return StateCASResult{}, err
	}

	current, currentDigest, exists, err := currentStateHeadTx(
		ctx,
		transaction,
		request.RepositoryScopeSHA256,
		request.Kind,
	)
	if err != nil {
		return StateCASResult{}, err
	}
	if !exists {
		if request.ExpectedGeneration != 0 {
			return StateCASResult{}, ErrStateHeadPrecondition
		}
	} else if current.Generation != request.ExpectedGeneration ||
		currentDigest != request.ExpectedHeadSHA256 {
		return StateCASResult{}, ErrStateHeadPrecondition
	}

	now := storage.now()
	createdAt, _ := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	if now.Before(createdAt) {
		return StateCASResult{}, ErrStateInvalid
	}
	if manifest.Kind == StateKindCheckpoint {
		expiresAt, _ := time.Parse(time.RFC3339Nano, manifest.ExpiresAt)
		if !now.Before(expiresAt) {
			return StateCASResult{}, ErrStateInvalid
		}
	}
	head := StateHead{
		SchemaVersion:         "buildopt.central/state-head/v1",
		RecordType:            "CENTRAL_STATE_HEAD",
		Kind:                  request.Kind,
		RepositoryScopeSHA256: request.RepositoryScopeSHA256,
		Generation:            manifest.Generation,
		ManifestSHA256:        request.ManifestSHA256,
		CompatibilitySHA256:   manifest.CompatibilitySHA256,
		UpdatedAt:             now.Format(time.RFC3339Nano),
		Authority: StateAuthority{
			SelectionRequiresLocalRevalidation: true,
			ProductionAuthorized:               false,
			TestOptimization:                   "OUT_OF_SCOPE",
		},
	}
	if exists {
		head.PreviousManifestSHA256 = current.ManifestSHA256
	}
	headCanonical, headDigest, err := canonicalStateValue(head)
	if err != nil {
		return StateCASResult{}, err
	}
	if exists {
		result, err := transaction.ExecContext(
			ctx,
			`UPDATE state_heads SET
    generation = ?, head_digest = ?, canonical_document = ?,
    manifest_digest = ?, compatibility_sha256 = ?, updated_at_unix_ms = ?
WHERE repository_scope_sha256 = ? AND kind = ?
  AND generation = ? AND head_digest = ?`,
			head.Generation,
			digestPrefix+headDigest,
			headCanonical,
			digestPrefix+head.ManifestSHA256,
			head.CompatibilitySHA256,
			now.UnixMilli(),
			request.RepositoryScopeSHA256,
			request.Kind,
			request.ExpectedGeneration,
			digestPrefix+request.ExpectedHeadSHA256,
		)
		if err != nil {
			return StateCASResult{}, err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return StateCASResult{}, ErrStateHeadPrecondition
		}
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE state_manifests SET retention_started_at_unix_ms = ?
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
			now.UnixMilli(),
			request.RepositoryScopeSHA256,
			request.Kind,
			digestPrefix+current.ManifestSHA256,
		); err != nil {
			return StateCASResult{}, err
		}
	} else if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO state_heads (
    repository_scope_sha256, kind, generation, head_digest,
    canonical_document, manifest_digest, compatibility_sha256, updated_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		request.RepositoryScopeSHA256,
		request.Kind,
		head.Generation,
		digestPrefix+headDigest,
		headCanonical,
		digestPrefix+head.ManifestSHA256,
		head.CompatibilitySHA256,
		now.UnixMilli(),
	); err != nil {
		return StateCASResult{}, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE state_manifests SET
    published_at_unix_ms = coalesce(published_at_unix_ms, ?),
    retention_started_at_unix_ms = NULL
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
		now.UnixMilli(),
		request.RepositoryScopeSHA256,
		request.Kind,
		digestPrefix+request.ManifestSHA256,
	); err != nil {
		return StateCASResult{}, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO state_cas_requests (
    idempotency_key, request_digest, repository_scope_sha256, kind,
    generation, head_digest, head_canonical_document, created_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		request.IdempotencyKey,
		digestPrefix+fingerprint,
		request.RepositoryScopeSHA256,
		request.Kind,
		head.Generation,
		digestPrefix+headDigest,
		headCanonical,
		now.UnixMilli(),
	); err != nil {
		return StateCASResult{}, err
	}
	if err := transaction.Commit(); err != nil {
		return StateCASResult{}, err
	}
	rollback = false
	return StateCASResult{Head: head, HeadSHA256: headDigest}, nil
}

// LoadCurrentState verifies the current head, manifest, references and every
// artifact byte before returning a snapshot.
func (storage *Storage) LoadCurrentState(
	ctx context.Context,
	repositoryScopeSHA256 string,
	kind StateKind,
) (StateSnapshot, error) {
	if ctx == nil || !validSHA256(repositoryScopeSHA256) || !validStateKind(kind) {
		return StateSnapshot{}, ErrStateInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return StateSnapshot{}, err
	}
	defer finish()
	storage.reconcileMutex.RLock()
	defer storage.reconcileMutex.RUnlock()
	var headRaw, manifestRaw []byte
	var headDigest, manifestDigest string
	err = storage.state.database.QueryRowContext(
		ctx,
		`SELECT head.canonical_document, head.head_digest,
       manifest.canonical_document, manifest.manifest_digest
FROM state_heads AS head
JOIN state_manifests AS manifest
  ON manifest.repository_scope_sha256 = head.repository_scope_sha256
 AND manifest.kind = head.kind
 AND manifest.manifest_digest = head.manifest_digest
WHERE head.repository_scope_sha256 = ? AND head.kind = ?`,
		repositoryScopeSHA256,
		kind,
	).Scan(&headRaw, &headDigest, &manifestRaw, &manifestDigest)
	if errors.Is(err, sql.ErrNoRows) {
		return StateSnapshot{}, ErrStateNotFound
	}
	if err != nil {
		return StateSnapshot{}, err
	}
	head, err := decodeStateHead(headRaw)
	if err != nil || digestBytes(headRaw) != strings.TrimPrefix(headDigest, digestPrefix) {
		return StateSnapshot{}, ErrStateCorrupt
	}
	manifest, canonical, decodedDigest, err := decodeStateManifest(manifestRaw)
	if err != nil || !bytes.Equal(canonical, manifestRaw) ||
		decodedDigest != strings.TrimPrefix(manifestDigest, digestPrefix) ||
		head.ManifestSHA256 != decodedDigest || head.Generation != manifest.Generation ||
		head.RepositoryScopeSHA256 != manifest.RepositoryScopeSHA256 ||
		head.Kind != manifest.Kind || head.CompatibilitySHA256 != manifest.CompatibilitySHA256 {
		return StateSnapshot{}, ErrStateCorrupt
	}
	if kind == StateKindCheckpoint {
		expiresAt, _ := time.Parse(time.RFC3339Nano, manifest.ExpiresAt)
		if !storage.now().Before(expiresAt) {
			return StateSnapshot{}, ErrStateNotFound
		}
	}
	for _, artifact := range manifest.Artifacts {
		var size int64
		err := storage.state.database.QueryRowContext(
			ctx,
			`SELECT object.size_bytes FROM state_manifest_artifacts AS artifact
JOIN state_objects AS object
  ON object.repository_scope_sha256 = artifact.repository_scope_sha256
 AND object.kind = artifact.kind
 AND object.blob_digest = artifact.blob_digest
WHERE artifact.repository_scope_sha256 = ? AND artifact.kind = ?
  AND artifact.manifest_digest = ? AND artifact.role = ?
  AND artifact.blob_digest = ? AND artifact.size_bytes = ?`,
			repositoryScopeSHA256,
			kind,
			manifestDigest,
			artifact.Role,
			digestPrefix+artifact.SHA256,
			artifact.SizeBytes,
		).Scan(&size)
		if err != nil || size != artifact.SizeBytes {
			return StateSnapshot{}, ErrStateCorrupt
		}
		file, err := storage.blobs.openVerified(ctx, Blob{
			Digest: digestPrefix + artifact.SHA256,
			Size:   artifact.SizeBytes,
		})
		if file != nil {
			_ = file.Close()
		}
		if err != nil {
			return StateSnapshot{}, fmt.Errorf("%w: %v", ErrStateCorrupt, err)
		}
	}
	for _, reference := range manifest.References {
		var present int
		err := storage.state.database.QueryRowContext(
			ctx,
			`SELECT 1 FROM state_manifest_references AS reference
JOIN state_manifests AS target
  ON target.repository_scope_sha256 = reference.repository_scope_sha256
 AND target.kind = reference.target_kind
 AND target.manifest_digest = reference.target_manifest_digest
WHERE reference.repository_scope_sha256 = ?
  AND reference.source_kind = ? AND reference.source_manifest_digest = ?
  AND reference.target_kind = ? AND reference.target_manifest_digest = ?
  AND reference.relation = ?`,
			repositoryScopeSHA256,
			kind,
			manifestDigest,
			reference.Kind,
			digestPrefix+reference.ManifestSHA256,
			reference.Relation,
		).Scan(&present)
		if err != nil {
			return StateSnapshot{}, ErrStateCorrupt
		}
	}
	return StateSnapshot{
		Head: head, HeadSHA256: strings.TrimPrefix(headDigest, digestPrefix),
		Manifest: manifest, ManifestSHA256: strings.TrimPrefix(manifestDigest, digestPrefix),
	}, nil
}

// MaintainState applies only typed-state retention. Gradle cache SLRU and
// capacity policy remain independent.
func (storage *Storage) MaintainState(
	ctx context.Context,
) (StateMaintenanceReport, error) {
	if ctx == nil {
		return StateMaintenanceReport{}, ErrStateInvalid
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return StateMaintenanceReport{}, err
	}
	defer finish()
	storage.reconcileMutex.Lock()
	defer storage.reconcileMutex.Unlock()
	storage.stateCASMutex.Lock()
	defer storage.stateCASMutex.Unlock()
	report, err := storage.maintainStateMetadata(ctx, storage.now())
	if err != nil {
		return report, err
	}
	deleted, err := storage.deleteOrphanBlobs(ctx)
	if err != nil {
		return report, err
	}
	report.DeletedUnreferencedBlob = deleted
	return report, nil
}

func (storage *Storage) maintainStateMetadata(
	ctx context.Context,
	now time.Time,
) (StateMaintenanceReport, error) {
	var report StateMaintenanceReport
	transaction, err := storage.state.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return report, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()

	result, err := transaction.ExecContext(
		ctx,
		`DELETE FROM state_heads
WHERE kind = 'CHECKPOINT' AND EXISTS (
    SELECT 1 FROM state_manifests AS manifest
    WHERE manifest.repository_scope_sha256 = state_heads.repository_scope_sha256
      AND manifest.kind = state_heads.kind
      AND manifest.manifest_digest = state_heads.manifest_digest
      AND manifest.expires_at_unix_ms <= ?
)`,
		now.UnixMilli(),
	)
	if err != nil {
		return report, err
	}
	report.ExpiredCheckpoints, err = affectedRows(result)
	if err != nil {
		return report, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		`DELETE FROM state_manifests
WHERE kind = 'CHECKPOINT' AND expires_at_unix_ms <= ?`,
		now.UnixMilli(),
	); err != nil {
		return report, err
	}

	result, err = transaction.ExecContext(
		ctx,
		`DELETE FROM state_manifests
WHERE published_at_unix_ms IS NULL AND created_at_unix_ms <= ?`,
		now.Add(-stagedStateTTL).UnixMilli(),
	)
	if err != nil {
		return report, err
	}
	report.ExpiredStagedManifests, err = affectedRows(result)
	if err != nil {
		return report, err
	}

	result, err = transaction.ExecContext(
		ctx,
		`DELETE FROM state_manifests
WHERE kind = 'PORTFOLIO'
  AND retention_started_at_unix_ms IS NOT NULL
  AND retention_started_at_unix_ms <= ?
  AND NOT EXISTS (
      SELECT 1 FROM state_heads AS head
      WHERE head.repository_scope_sha256 = state_manifests.repository_scope_sha256
        AND head.kind = state_manifests.kind
        AND head.manifest_digest = state_manifests.manifest_digest
  )`,
		now.Add(-supersededStateTTL).UnixMilli(),
	)
	if err != nil {
		return report, err
	}
	report.ExpiredSuperseded, err = affectedRows(result)
	if err != nil {
		return report, err
	}

	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE state_manifests SET retention_started_at_unix_ms = NULL
WHERE kind = 'EVIDENCE' AND EXISTS (
    SELECT 1 FROM state_manifest_references AS reference
    WHERE reference.repository_scope_sha256 = state_manifests.repository_scope_sha256
      AND reference.target_kind = state_manifests.kind
      AND reference.target_manifest_digest = state_manifests.manifest_digest
)`,
	); err != nil {
		return report, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE state_manifests SET retention_started_at_unix_ms = ?
WHERE kind = 'EVIDENCE' AND published_at_unix_ms IS NOT NULL
  AND retention_started_at_unix_ms IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM state_heads AS head
      WHERE head.repository_scope_sha256 = state_manifests.repository_scope_sha256
        AND head.kind = state_manifests.kind
        AND head.manifest_digest = state_manifests.manifest_digest
  )
  AND NOT EXISTS (
      SELECT 1 FROM state_manifest_references AS reference
      WHERE reference.repository_scope_sha256 = state_manifests.repository_scope_sha256
        AND reference.target_kind = state_manifests.kind
        AND reference.target_manifest_digest = state_manifests.manifest_digest
  )`,
		now.UnixMilli(),
	); err != nil {
		return report, err
	}
	result, err = transaction.ExecContext(
		ctx,
		`DELETE FROM state_manifests
WHERE kind = 'EVIDENCE' AND retention_started_at_unix_ms IS NOT NULL
  AND retention_started_at_unix_ms <= ?
  AND NOT EXISTS (
      SELECT 1 FROM state_heads AS head
      WHERE head.repository_scope_sha256 = state_manifests.repository_scope_sha256
        AND head.kind = state_manifests.kind
        AND head.manifest_digest = state_manifests.manifest_digest
  )
  AND NOT EXISTS (
      SELECT 1 FROM state_manifest_references AS reference
      WHERE reference.repository_scope_sha256 = state_manifests.repository_scope_sha256
        AND reference.target_kind = state_manifests.kind
        AND reference.target_manifest_digest = state_manifests.manifest_digest
  )`,
		now.Add(-supersededStateTTL).UnixMilli(),
	)
	if err != nil {
		return report, err
	}
	evidenceExpired, err := affectedRows(result)
	if err != nil {
		return report, err
	}
	report.ExpiredSuperseded += evidenceExpired

	result, err = transaction.ExecContext(
		ctx,
		`DELETE FROM state_objects
WHERE created_at_unix_ms <= ? AND NOT EXISTS (
    SELECT 1 FROM state_manifest_artifacts AS artifact
    WHERE artifact.repository_scope_sha256 = state_objects.repository_scope_sha256
      AND artifact.kind = state_objects.kind
      AND artifact.blob_digest = state_objects.blob_digest
)`,
		now.Add(-stagedStateTTL).UnixMilli(),
	)
	if err != nil {
		return report, err
	}
	report.ExpiredStagedObjects, err = affectedRows(result)
	if err != nil {
		return report, err
	}
	if err := transaction.Commit(); err != nil {
		return report, err
	}
	rollback = false
	return report, nil
}

func loadStateManifestTx(
	ctx context.Context,
	transaction *sql.Tx,
	scope string,
	kind StateKind,
	digest string,
) (StateManifest, error) {
	var raw []byte
	err := transaction.QueryRowContext(
		ctx,
		`SELECT canonical_document FROM state_manifests
WHERE repository_scope_sha256 = ? AND kind = ? AND manifest_digest = ?`,
		scope,
		kind,
		digestPrefix+digest,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return StateManifest{}, ErrStateNotFound
	}
	if err != nil {
		return StateManifest{}, err
	}
	manifest, canonical, actual, err := decodeStateManifest(raw)
	if err != nil || !bytes.Equal(raw, canonical) || actual != digest {
		return StateManifest{}, ErrStateCorrupt
	}
	return manifest, nil
}

func (storage *Storage) verifyStateManifestForPublication(
	ctx context.Context,
	transaction *sql.Tx,
	manifest StateManifest,
	manifestDigest string,
) error {
	for _, artifact := range manifest.Artifacts {
		var size int64
		err := transaction.QueryRowContext(
			ctx,
			`SELECT object.size_bytes FROM state_manifest_artifacts AS artifact
JOIN state_objects AS object
  ON object.repository_scope_sha256 = artifact.repository_scope_sha256
 AND object.kind = artifact.kind
 AND object.blob_digest = artifact.blob_digest
WHERE artifact.repository_scope_sha256 = ? AND artifact.kind = ?
  AND artifact.manifest_digest = ? AND artifact.role = ?
  AND artifact.blob_digest = ? AND artifact.size_bytes = ?`,
			manifest.RepositoryScopeSHA256,
			manifest.Kind,
			digestPrefix+manifestDigest,
			artifact.Role,
			digestPrefix+artifact.SHA256,
			artifact.SizeBytes,
		).Scan(&size)
		if err != nil || size != artifact.SizeBytes {
			return ErrStateManifestIncomplete
		}
		file, err := storage.blobs.openVerified(ctx, Blob{
			Digest: digestPrefix + artifact.SHA256,
			Size:   artifact.SizeBytes,
		})
		if file != nil {
			_ = file.Close()
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrStateCorrupt, err)
		}
	}
	for _, reference := range manifest.References {
		var present int
		err := transaction.QueryRowContext(
			ctx,
			`SELECT 1 FROM state_manifest_references AS reference
JOIN state_manifests AS target
  ON target.repository_scope_sha256 = reference.repository_scope_sha256
 AND target.kind = reference.target_kind
 AND target.manifest_digest = reference.target_manifest_digest
WHERE reference.repository_scope_sha256 = ?
  AND reference.source_kind = ? AND reference.source_manifest_digest = ?
  AND reference.target_kind = ? AND reference.target_manifest_digest = ?
  AND reference.relation = ?`,
			manifest.RepositoryScopeSHA256,
			manifest.Kind,
			digestPrefix+manifestDigest,
			reference.Kind,
			digestPrefix+reference.ManifestSHA256,
			reference.Relation,
		).Scan(&present)
		if err != nil {
			return ErrStateManifestIncomplete
		}
	}
	return nil
}

func currentStateHeadTx(
	ctx context.Context,
	transaction *sql.Tx,
	scope string,
	kind StateKind,
) (StateHead, string, bool, error) {
	var raw []byte
	var digest string
	err := transaction.QueryRowContext(
		ctx,
		`SELECT canonical_document, head_digest FROM state_heads
WHERE repository_scope_sha256 = ? AND kind = ?`,
		scope,
		kind,
	).Scan(&raw, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return StateHead{}, "", false, nil
	}
	if err != nil {
		return StateHead{}, "", false, err
	}
	head, err := decodeStateHead(raw)
	digest = strings.TrimPrefix(digest, digestPrefix)
	if err != nil || digestBytes(raw) != digest {
		return StateHead{}, "", false, ErrStateCorrupt
	}
	return head, digest, true, nil
}

func decodeStateManifest(raw []byte) (StateManifest, []byte, string, error) {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		return StateManifest{}, nil, "", ErrStateInvalid
	}
	var manifest StateManifest
	if err := decodeStrictStateJSON(canonical, &manifest); err != nil ||
		validateStateManifest(manifest) != nil {
		return StateManifest{}, nil, "", ErrStateInvalid
	}
	return manifest, canonical, digestBytes(canonical), nil
}

func decodeStateHead(raw []byte) (StateHead, error) {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return StateHead{}, ErrStateInvalid
	}
	var head StateHead
	if err := decodeStrictStateJSON(canonical, &head); err != nil ||
		head.SchemaVersion != "buildopt.central/state-head/v1" ||
		head.RecordType != "CENTRAL_STATE_HEAD" || !validStateKind(head.Kind) ||
		!validSHA256(head.RepositoryScopeSHA256) || head.Generation < 1 ||
		head.Generation > maximumStateGeneration ||
		!validSHA256(head.ManifestSHA256) || !validSHA256(head.CompatibilitySHA256) ||
		!contractcrypto.ValidUTCTimestamp(head.UpdatedAt) || !validStateAuthority(head.Authority) ||
		(head.Generation == 1 && head.PreviousManifestSHA256 != "") ||
		(head.Generation > 1 && !validSHA256(head.PreviousManifestSHA256)) {
		return StateHead{}, ErrStateInvalid
	}
	return head, nil
}

func decodeStrictStateJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}

func validateStateManifest(manifest StateManifest) error {
	if manifest.SchemaVersion != "buildopt.central/state-manifest/v1" ||
		manifest.RecordType != "CENTRAL_STATE_MANIFEST" ||
		!validStateKind(manifest.Kind) || !validSHA256(manifest.RepositoryScopeSHA256) ||
		manifest.Generation < 1 || manifest.Generation > maximumStateGeneration ||
		!validSHA256(manifest.CompatibilitySHA256) ||
		!validSHA256(manifest.BindingsSHA256) ||
		!stateRevisionPattern.MatchString(manifest.Origin.BaseRevision) ||
		!stateRevisionPattern.MatchString(manifest.Origin.TargetRevision) ||
		!validSHA256(manifest.Origin.BuildOptExecutableSHA256) ||
		!validSHA256(manifest.Origin.WrapperSHA256) ||
		!stateGradlePattern.MatchString(manifest.Origin.GradleVersion) ||
		len(manifest.Artifacts) < 1 || len(manifest.Artifacts) > 64 ||
		len(manifest.References) > 64 ||
		!contractcrypto.ValidUTCTimestamp(manifest.CreatedAt) ||
		!validStateAuthority(manifest.Authority) {
		return ErrStateInvalid
	}
	artifactRoles := map[string]bool{}
	artifactRoleCounts := map[string]int{}
	artifactIdentities := map[string]bool{}
	validRoles := map[string]bool{
		"PORTFOLIO_INDEX": true, "PROFILE": true, "IMPACT_MANIFEST": true,
		"IMPACT_GRAPH": true, "GENERATED_MANIFEST": true,
		"CALIBRATION_EVIDENCE": true, "DISCOVERY_EVIDENCE": true,
		"TASK_SHAPE_EVIDENCE": true, "OUTPUT_MANIFEST": true,
		"OPTIMIZE_STATE": true, "DISCOVERY_SNAPSHOT": true,
		"CALIBRATION_CHECKPOINT": true,
	}
	for _, artifact := range manifest.Artifacts {
		identity := fmt.Sprintf("%s\x00%s\x00%d\x00%s", artifact.Role, artifact.SHA256, artifact.SizeBytes, artifact.PayloadSchemaVersion)
		if !validRoles[artifact.Role] || !validSHA256(artifact.SHA256) ||
			artifact.SizeBytes < 1 || artifact.SizeBytes > maximumStateArtifactBytes ||
			!stateSchemaPattern.MatchString(artifact.PayloadSchemaVersion) ||
			artifactIdentities[identity] {
			return ErrStateInvalid
		}
		artifactIdentities[identity] = true
		artifactRoles[artifact.Role] = true
		artifactRoleCounts[artifact.Role]++
	}
	referenceIdentities := map[string]bool{}
	for _, reference := range manifest.References {
		identity := string(reference.Kind) + "\x00" + reference.ManifestSHA256 + "\x00" + reference.Relation
		if reference.Kind != StateKindEvidence || !validSHA256(reference.ManifestSHA256) ||
			reference.Relation != "QUALIFICATION" || referenceIdentities[identity] {
			return ErrStateInvalid
		}
		referenceIdentities[identity] = true
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	switch manifest.Kind {
	case StateKindPortfolio:
		if manifest.Status != "COMPLETE" ||
			manifest.RetentionClass != "CURRENT_PLUS_30_DAYS_AFTER_SUPERSEDED" ||
			manifest.ExpiresAt != "" || !artifactRoles["PORTFOLIO_INDEX"] ||
			len(manifest.References) == 0 {
			return ErrStateInvalid
		}
		if artifactRoleCounts["PORTFOLIO_INDEX"] != 1 {
			return ErrStateInvalid
		}
	case StateKindEvidence:
		if manifest.Status != "COMPLETE" ||
			manifest.RetentionClass != "WHILE_REFERENCED_PLUS_30_DAYS" ||
			manifest.ExpiresAt != "" || len(manifest.References) != 0 ||
			!artifactRoles["CALIBRATION_EVIDENCE"] {
			return ErrStateInvalid
		}
	case StateKindCheckpoint:
		if manifest.Status != "RESUMABLE" ||
			manifest.RetentionClass != "24_HOURS_FROM_CREATED_AT" ||
			!contractcrypto.ValidUTCTimestamp(manifest.ExpiresAt) ||
			len(manifest.References) != 0 ||
			artifactRoleCounts["OPTIMIZE_STATE"] != 1 {
			return ErrStateInvalid
		}
		expiresAt, _ := time.Parse(time.RFC3339Nano, manifest.ExpiresAt)
		if !expiresAt.Equal(createdAt.Add(stagedStateTTL)) {
			return ErrStateInvalid
		}
	}
	return nil
}

func validStateAuthority(authority StateAuthority) bool {
	return authority.SelectionRequiresLocalRevalidation &&
		!authority.ProductionAuthorized && authority.TestOptimization == "OUT_OF_SCOPE"
}

func validStateKind(kind StateKind) bool {
	return kind == StateKindPortfolio || kind == StateKindEvidence || kind == StateKindCheckpoint
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func canonicalStateValue(value any) ([]byte, string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		return nil, "", err
	}
	return canonical, digestBytes(canonical), nil
}

func stateCASFingerprint(request StateCASRequest) (string, error) {
	_, digest, err := canonicalStateValue(map[string]any{
		"expectedGeneration":    request.ExpectedGeneration,
		"expectedHeadSha256":    nullableStateDigest(request.ExpectedHeadSHA256),
		"idempotencyKey":        request.IdempotencyKey,
		"kind":                  request.Kind,
		"manifestSha256":        request.ManifestSHA256,
		"repositoryScopeSha256": request.RepositoryScopeSHA256,
	})
	return digest, err
}

func nullableStateDigest(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func affectedRows(result sql.Result) (int, error) {
	count, err := result.RowsAffected()
	return int(count), err
}
