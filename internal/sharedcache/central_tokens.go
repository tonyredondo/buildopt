package sharedcache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

const (
	// CentralTokenBytes is the entropy carried by one owner-issued POC token.
	CentralTokenBytes = 32
	// CentralTokenMaximumLifetime bounds manually managed POC credentials.
	CentralTokenMaximumLifetime = 30 * 24 * time.Hour
)

// CentralCapability is one independently authorized central-service action.
type CentralCapability string

const (
	CentralCacheRead  CentralCapability = "CACHE_READ"
	CentralCacheWrite CentralCapability = "CACHE_WRITE"
	CentralStateRead  CentralCapability = "STATE_READ"
	CentralStateWrite CentralCapability = "STATE_WRITE"
)

var centralCapabilityOrder = []CentralCapability{
	CentralCacheRead,
	CentralCacheWrite,
	CentralStateRead,
	CentralStateWrite,
}

// CentralTokenScope binds one credential to both the Gradle-cache namespace
// and the independently hashed BuildOpt state repository namespace.
type CentralTokenScope struct {
	RepositoryScopeSHA256 string
	Tenant                string
	Repository            string
	TrustDomain           string
	Namespace             string
	NamespaceGeneration   int64
}

// CentralTokenIssueRequest is one manual owner-operated POC provisioning
// request. Capabilities remain separate even when one token carries several.
type CentralTokenIssueRequest struct {
	Scope        CentralTokenScope
	Capabilities []CentralCapability
	ExpiresAt    time.Time
}

// IssuedCentralToken returns the opaque secret exactly once. Durable storage
// contains only a domain-separated digest and the exact scope/capabilities.
type IssuedCentralToken struct {
	TokenID      string
	Token        string
	Scope        CentralTokenScope
	Capabilities []CentralCapability
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// CentralTokenAuthorization is the authenticated request context. It never
// contains the raw credential.
type CentralTokenAuthorization struct {
	Scope        CentralTokenScope
	Capabilities []CentralCapability
}

// Has reports whether the authenticated token grants one exact capability.
func (authorization CentralTokenAuthorization) Has(capability CentralCapability) bool {
	return slices.Contains(authorization.Capabilities, capability)
}

// CentralTokenRegistry owns manual issue/revocation without taking the Shared
// writer lease, allowing operators to revoke a live server credential.
type CentralTokenRegistry struct {
	metadata                   *sqliteMetadata
	minimumNamespaceGeneration int64
	lifecycleLease             *datalifecycle.ManagedLease
}

// OpenCentralTokenRegistry opens the control metadata needed for manual token
// operations without taking the server's process-lifetime writer lock.
func OpenCentralTokenRegistry(
	ctx context.Context,
	stateRoot string,
) (*CentralTokenRegistry, error) {
	if ctx == nil || !filepath.IsAbs(stateRoot) || filepath.Clean(stateRoot) != stateRoot {
		return nil, errors.New("open central token registry: invalid state root")
	}
	lifecycleLease, boundary, err := datalifecycle.AcquireManagedLease(stateRoot)
	if err != nil {
		return nil, fmt.Errorf("open central token registry: inspect managed lifecycle: %w", err)
	}
	if err := preparePrivateDirectory(stateRoot); err != nil {
		_ = lifecycleLease.Close()
		return nil, fmt.Errorf("open central token registry: %w", err)
	}
	if err := validateStorageRootEntries(stateRoot); err != nil {
		_ = lifecycleLease.Close()
		return nil, fmt.Errorf("open central token registry: %w", err)
	}
	metadata, err := openSQLiteMetadata(
		ctx,
		nil,
		controlMetadataDefinition(filepath.Join(stateRoot, "control.sqlite")),
	)
	if err != nil {
		_ = lifecycleLease.Close()
		return nil, fmt.Errorf("open central token registry: %w", err)
	}
	return &CentralTokenRegistry{
		metadata:                   metadata,
		minimumNamespaceGeneration: boundary.MinimumNamespaceGeneration,
		lifecycleLease:             lifecycleLease,
	}, nil
}

// Close releases the registry database and its managed lifecycle lease.
func (registry *CentralTokenRegistry) Close() error {
	if registry == nil || registry.metadata == nil {
		return nil
	}
	metadataErr := registry.metadata.close()
	registry.metadata = nil
	leaseErr := registry.lifecycleLease.Close()
	registry.lifecycleLease = nil
	return errors.Join(metadataErr, leaseErr)
}

// Issue creates one scoped credential and returns its raw value exactly once.
func (registry *CentralTokenRegistry) Issue(
	ctx context.Context,
	request CentralTokenIssueRequest,
	now time.Time,
) (IssuedCentralToken, error) {
	if registry == nil || registry.metadata == nil {
		return IssuedCentralToken{}, errors.New("issue central token: closed registry")
	}
	if request.Scope.NamespaceGeneration < registry.minimumNamespaceGeneration {
		return IssuedCentralToken{}, errors.New("issue central token: namespace generation predates managed deletion")
	}
	return issueCentralToken(ctx, registry.metadata.database, request, now)
}

// Revoke invalidates one token identifier for every subsequent request.
func (registry *CentralTokenRegistry) Revoke(
	ctx context.Context,
	tokenID string,
	now time.Time,
) (bool, error) {
	if registry == nil || registry.metadata == nil {
		return false, errors.New("revoke central token: closed registry")
	}
	return revokeCentralToken(ctx, registry.metadata.database, tokenID, now)
}

// IssueCentralToken creates a token through an already open Shared store.
func (storage *Storage) IssueCentralToken(
	ctx context.Context,
	request CentralTokenIssueRequest,
	now time.Time,
) (IssuedCentralToken, error) {
	if storage == nil {
		return IssuedCentralToken{}, errors.New("issue central token: nil storage")
	}
	if request.Scope.NamespaceGeneration < storage.minimumNamespaceGeneration {
		return IssuedCentralToken{}, errors.New("issue central token: namespace generation predates managed deletion")
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return IssuedCentralToken{}, err
	}
	defer finish()
	return issueCentralToken(ctx, storage.control.database, request, now)
}

// RevokeCentralToken revokes a token through an already open Shared store.
func (storage *Storage) RevokeCentralToken(
	ctx context.Context,
	tokenID string,
	now time.Time,
) (bool, error) {
	if storage == nil {
		return false, errors.New("revoke central token: nil storage")
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return false, err
	}
	defer finish()
	return revokeCentralToken(ctx, storage.control.database, tokenID, now)
}

func issueCentralToken(
	ctx context.Context,
	database *sql.DB,
	request CentralTokenIssueRequest,
	now time.Time,
) (IssuedCentralToken, error) {
	now = now.UTC()
	request.ExpiresAt = request.ExpiresAt.UTC()
	capabilities, flags, ok := normalizeCentralCapabilities(request.Capabilities)
	if ctx == nil || database == nil || !validCentralTokenScope(request.Scope) ||
		!ok || now.IsZero() || !request.ExpiresAt.After(now) ||
		request.ExpiresAt.Sub(now) > CentralTokenMaximumLifetime {
		return IssuedCentralToken{}, errors.New("issue central token: invalid request")
	}
	if err := ctx.Err(); err != nil {
		return IssuedCentralToken{}, err
	}
	raw := make([]byte, CentralTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return IssuedCentralToken{}, fmt.Errorf("issue central token: %w", err)
	}
	defer clear(raw)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return IssuedCentralToken{}, fmt.Errorf("issue central token ID: %w", err)
	}
	tokenID := hex.EncodeToString(idBytes)
	token := base64.RawURLEncoding.EncodeToString(raw)
	if _, err := database.ExecContext(
		ctx,
		`INSERT INTO central_access_tokens (
    token_id, token_digest, repository_scope_sha256,
    tenant_id, repository_id, trust_domain, namespace, namespace_generation,
    cache_read, cache_write, state_read, state_write,
    issued_at_unix_ms, expires_at_unix_ms, revoked_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		tokenID,
		centralTokenDigest(raw),
		request.Scope.RepositoryScopeSHA256,
		request.Scope.Tenant,
		request.Scope.Repository,
		request.Scope.TrustDomain,
		request.Scope.Namespace,
		request.Scope.NamespaceGeneration,
		flags[CentralCacheRead],
		flags[CentralCacheWrite],
		flags[CentralStateRead],
		flags[CentralStateWrite],
		now.UnixMilli(),
		request.ExpiresAt.UnixMilli(),
	); err != nil {
		return IssuedCentralToken{}, fmt.Errorf("issue central token: %w", err)
	}
	return IssuedCentralToken{
		TokenID: tokenID, Token: token, Scope: request.Scope,
		Capabilities: capabilities, IssuedAt: now, ExpiresAt: request.ExpiresAt,
	}, nil
}

func revokeCentralToken(
	ctx context.Context,
	database *sql.DB,
	tokenID string,
	now time.Time,
) (bool, error) {
	now = now.UTC()
	if ctx == nil || database == nil || !betaTokenIDPattern.MatchString(tokenID) || now.IsZero() {
		return false, errors.New("revoke central token: invalid request")
	}
	result, err := database.ExecContext(
		ctx,
		`UPDATE central_access_tokens SET revoked_at_unix_ms = ?
WHERE token_id = ? AND revoked_at_unix_ms IS NULL`,
		now.UnixMilli(),
		tokenID,
	)
	if err != nil {
		return false, fmt.Errorf("revoke central token: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("revoke central token: %w", err)
	}
	return changed == 1, nil
}

func authenticateCentralToken(
	ctx context.Context,
	database *sql.DB,
	raw []byte,
	now time.Time,
) (CentralTokenAuthorization, bool, error) {
	var (
		authorization                                CentralTokenAuthorization
		cacheRead, cacheWrite, stateRead, stateWrite int
		expiresAt                                    int64
		revokedAt                                    sql.NullInt64
	)
	err := database.QueryRowContext(
		ctx,
		`SELECT repository_scope_sha256, tenant_id, repository_id, trust_domain,
       namespace, namespace_generation,
       cache_read, cache_write, state_read, state_write,
       expires_at_unix_ms, revoked_at_unix_ms
FROM central_access_tokens WHERE token_digest = ?`,
		centralTokenDigest(raw),
	).Scan(
		&authorization.Scope.RepositoryScopeSHA256,
		&authorization.Scope.Tenant,
		&authorization.Scope.Repository,
		&authorization.Scope.TrustDomain,
		&authorization.Scope.Namespace,
		&authorization.Scope.NamespaceGeneration,
		&cacheRead, &cacheWrite, &stateRead, &stateWrite,
		&expiresAt, &revokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return CentralTokenAuthorization{}, false, nil
	}
	if err != nil {
		return CentralTokenAuthorization{}, false, err
	}
	if revokedAt.Valid || !time.UnixMilli(expiresAt).After(now.UTC()) ||
		!validCentralTokenScope(authorization.Scope) {
		return CentralTokenAuthorization{}, false, nil
	}
	flags := map[CentralCapability]int{
		CentralCacheRead: cacheRead, CentralCacheWrite: cacheWrite,
		CentralStateRead: stateRead, CentralStateWrite: stateWrite,
	}
	for _, capability := range centralCapabilityOrder {
		if flags[capability] == 1 {
			authorization.Capabilities = append(authorization.Capabilities, capability)
		} else if flags[capability] != 0 {
			return CentralTokenAuthorization{}, false, errors.New("invalid central capability state")
		}
	}
	return authorization, len(authorization.Capabilities) != 0, nil
}

func centralBearerToken(request *http.Request) ([]byte, bool) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return nil, false
	}
	encoded := strings.TrimPrefix(values[0], "Bearer ")
	if encoded == "" || strings.TrimSpace(encoded) != encoded {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) != CentralTokenBytes ||
		base64.RawURLEncoding.EncodeToString(raw) != encoded {
		clear(raw)
		return nil, false
	}
	return raw, true
}

func centralTokenDigest(raw []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("buildopt-central-access-token-v1"))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(raw)
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func normalizeCentralCapabilities(
	values []CentralCapability,
) ([]CentralCapability, map[CentralCapability]int, bool) {
	flags := map[CentralCapability]int{}
	for _, capability := range values {
		if !slices.Contains(centralCapabilityOrder, capability) || flags[capability] != 0 {
			return nil, nil, false
		}
		flags[capability] = 1
	}
	if len(flags) == 0 {
		return nil, nil, false
	}
	ordered := make([]CentralCapability, 0, len(flags))
	for _, capability := range centralCapabilityOrder {
		if flags[capability] == 1 {
			ordered = append(ordered, capability)
		}
	}
	return ordered, flags, true
}

func validCentralTokenScope(scope CentralTokenScope) bool {
	return validSHA256(scope.RepositoryScopeSHA256) &&
		validIdentifier(scope.Tenant) && validIdentifier(scope.Repository) &&
		validIdentifier(scope.TrustDomain) && validBetaNamespace(scope.Namespace) &&
		scope.NamespaceGeneration > 0
}
