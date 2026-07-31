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
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// BetaTokenBytes is the entropy carried by every opaque private-beta token.
	BetaTokenBytes = 32
	// BetaTokenMaximumLifetime is the private-beta credential lifetime ceiling.
	BetaTokenMaximumLifetime = 30 * 24 * time.Hour
)

// BetaTokenPlane keeps stable data, quarantine, and control credentials
// deliberately non-interchangeable.
type BetaTokenPlane string

const (
	BetaTokenPlaneStable     BetaTokenPlane = "STABLE"
	BetaTokenPlaneQuarantine BetaTokenPlane = "QUARANTINE"
	BetaTokenPlaneControl    BetaTokenPlane = "CONTROL"
)

// BetaTokenAccess is the complete private-beta operation vocabulary.
type BetaTokenAccess string

const (
	BetaTokenRead      BetaTokenAccess = "READ"
	BetaTokenReadWrite BetaTokenAccess = "READ_WRITE"
)

var (
	betaNamespacePattern = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,512}$`)
	betaTokenIDPattern   = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

// BetaTokenScope is the exact deployment-isolated remote credential scope.
type BetaTokenScope struct {
	Tenant              string
	Repository          string
	TrustDomain         string
	Namespace           string
	NamespaceGeneration int64
	Plane               BetaTokenPlane
}

// BetaTokenIssueRequest is one manual private-beta provisioning request.
type BetaTokenIssueRequest struct {
	Scope     BetaTokenScope
	Access    BetaTokenAccess
	ExpiresAt time.Time
}

// IssuedBetaToken returns the opaque secret exactly once. Only its digest is
// persisted by the registry.
type IssuedBetaToken struct {
	TokenID   string
	Token     string
	Scope     BetaTokenScope
	Access    BetaTokenAccess
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// BetaTokenRegistry is the narrow operator boundary for live manual issue and
// revocation against control.sqlite. It never opens cache.sqlite or the Shared
// writer lease.
type BetaTokenRegistry struct {
	metadata *sqliteMetadata
}

// OpenBetaTokenRegistry opens only the private control database. A running
// Shared process may remain active; SQLite WAL serializes the bounded update.
func OpenBetaTokenRegistry(
	ctx context.Context,
	stateRoot string,
) (*BetaTokenRegistry, error) {
	if ctx == nil || !filepath.IsAbs(stateRoot) ||
		filepath.Clean(stateRoot) != stateRoot {
		return nil, errors.New("open beta token registry: invalid state root")
	}
	if err := preparePrivateDirectory(stateRoot); err != nil {
		return nil, fmt.Errorf("open beta token registry: %w", err)
	}
	if err := validateStorageRootEntries(stateRoot); err != nil {
		return nil, fmt.Errorf("open beta token registry: %w", err)
	}
	metadata, err := openSQLiteMetadata(
		ctx,
		nil,
		controlMetadataDefinition(filepath.Join(stateRoot, "control.sqlite")),
	)
	if err != nil {
		return nil, fmt.Errorf("open beta token registry: %w", err)
	}
	return &BetaTokenRegistry{metadata: metadata}, nil
}

// Close releases the operator database connection.
func (registry *BetaTokenRegistry) Close() error {
	if registry == nil || registry.metadata == nil {
		return nil
	}
	return registry.metadata.close()
}

// Issue generates one opaque token and persists only its domain-separated
// SHA-256 digest and exact authorization scope.
func (registry *BetaTokenRegistry) Issue(
	ctx context.Context,
	request BetaTokenIssueRequest,
	now time.Time,
) (IssuedBetaToken, error) {
	if registry == nil || registry.metadata == nil {
		return IssuedBetaToken{}, errors.New("issue beta token: closed registry")
	}
	return issueBetaToken(ctx, registry.metadata.database, request, now)
}

// Revoke immediately invalidates one token ID for every subsequent request.
func (registry *BetaTokenRegistry) Revoke(
	ctx context.Context,
	tokenID string,
	now time.Time,
) (bool, error) {
	if registry == nil || registry.metadata == nil {
		return false, errors.New("revoke beta token: closed registry")
	}
	return revokeBetaToken(ctx, registry.metadata.database, tokenID, now)
}

// IssueBetaToken provisions a token through the live Shared storage owner.
func (storage *Storage) IssueBetaToken(
	ctx context.Context,
	request BetaTokenIssueRequest,
	now time.Time,
) (IssuedBetaToken, error) {
	if storage == nil {
		return IssuedBetaToken{}, errors.New("issue beta token: nil storage")
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return IssuedBetaToken{}, err
	}
	defer finish()
	return issueBetaToken(ctx, storage.control.database, request, now)
}

// RevokeBetaToken invalidates a token through the live Shared storage owner.
func (storage *Storage) RevokeBetaToken(
	ctx context.Context,
	tokenID string,
	now time.Time,
) (bool, error) {
	if storage == nil {
		return false, errors.New("revoke beta token: nil storage")
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return false, err
	}
	defer finish()
	return revokeBetaToken(ctx, storage.control.database, tokenID, now)
}

func issueBetaToken(
	ctx context.Context,
	database *sql.DB,
	request BetaTokenIssueRequest,
	now time.Time,
) (IssuedBetaToken, error) {
	now = now.UTC()
	request.ExpiresAt = request.ExpiresAt.UTC()
	if ctx == nil || database == nil || !validBetaTokenScope(request.Scope) ||
		!validBetaTokenAccess(request.Access) || now.IsZero() ||
		!request.ExpiresAt.After(now) ||
		request.ExpiresAt.Sub(now) > BetaTokenMaximumLifetime {
		return IssuedBetaToken{}, errors.New("issue beta token: invalid request")
	}
	if err := ctx.Err(); err != nil {
		return IssuedBetaToken{}, err
	}

	raw := make([]byte, BetaTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return IssuedBetaToken{}, fmt.Errorf("issue beta token: %w", err)
	}
	defer clear(raw)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return IssuedBetaToken{}, fmt.Errorf("issue beta token ID: %w", err)
	}
	tokenID := hex.EncodeToString(idBytes)
	token := base64.RawURLEncoding.EncodeToString(raw)
	if _, err := database.ExecContext(
		ctx,
		`INSERT INTO beta_cache_tokens (
    token_id, token_digest, tenant_id, repository_id, trust_domain,
    namespace, namespace_generation, plane, access,
    issued_at_unix_ms, expires_at_unix_ms, revoked_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		tokenID,
		betaTokenDigest(raw),
		request.Scope.Tenant,
		request.Scope.Repository,
		request.Scope.TrustDomain,
		request.Scope.Namespace,
		request.Scope.NamespaceGeneration,
		request.Scope.Plane,
		request.Access,
		now.UnixMilli(),
		request.ExpiresAt.UnixMilli(),
	); err != nil {
		return IssuedBetaToken{}, fmt.Errorf("issue beta token: %w", err)
	}
	return IssuedBetaToken{
		TokenID:   tokenID,
		Token:     token,
		Scope:     request.Scope,
		Access:    request.Access,
		IssuedAt:  now,
		ExpiresAt: request.ExpiresAt,
	}, nil
}

func revokeBetaToken(
	ctx context.Context,
	database *sql.DB,
	tokenID string,
	now time.Time,
) (bool, error) {
	now = now.UTC()
	if ctx == nil || database == nil ||
		!betaTokenIDPattern.MatchString(tokenID) || now.IsZero() {
		return false, errors.New("revoke beta token: invalid request")
	}
	result, err := database.ExecContext(
		ctx,
		`UPDATE beta_cache_tokens
SET revoked_at_unix_ms = ?
WHERE token_id = ? AND revoked_at_unix_ms IS NULL`,
		now.UnixMilli(),
		tokenID,
	)
	if err != nil {
		return false, fmt.Errorf("revoke beta token: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("revoke beta token: %w", err)
	}
	return changed == 1, nil
}

// NewBetaTokenHTTPHandler authenticates the remote hop with a dynamically
// revocable token while retaining the signed authority and attempt boundary.
func NewBetaTokenHTTPHandler(
	storage *Storage,
	binding LocalAuthorityBinding,
	plane BetaTokenPlane,
) (http.Handler, error) {
	if storage == nil || binding.AuthorityDigest == "" ||
		!validBetaTokenPlane(plane) || binding.state.Namespace == "" {
		return nil, errors.New("invalid beta token Shared cache binding")
	}
	if _, err := NewHTTPHandler(storage, binding.httpBinding); err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		response.Header().Set("Cache-Control", "no-store")
		raw, ok := betaBearerToken(request)
		digests := request.Header.Values(AuthorityDigestHeader)
		if !ok || len(digests) != 1 ||
			digests[0] != binding.AuthorityDigest {
			writeBetaUnauthorized(response)
			return
		}
		defer clear(raw)

		finish, err := storage.beginOperation()
		if err != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		access, authorized, authErr := authenticateBetaToken(
			request.Context(),
			storage.control.database,
			raw,
			binding,
			plane,
			storage.now(),
		)
		finish()
		if authErr != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		if !authorized {
			writeBetaUnauthorized(response)
			return
		}

		storage.authorityMutex.RLock()
		current, currentErr := storage.localAuthorityCurrentLocked(
			request.Context(),
			binding,
			storage.now(),
		)
		storage.authorityMutex.RUnlock()
		if currentErr != nil {
			writeCacheStatus(response, http.StatusServiceUnavailable)
			return
		}
		if !current {
			writeBetaUnauthorized(response)
			return
		}

		requestBinding := binding.httpBinding
		requestBinding.AllowRead = binding.AllowRead
		requestBinding.AllowWrite = binding.AllowWrite &&
			access == BetaTokenReadWrite
		if request.Method == http.MethodPut && !requestBinding.AllowWrite {
			writeCacheStatus(response, http.StatusForbidden)
			return
		}
		serveBoundCacheHTTP(storage, requestBinding, response, request)
	}), nil
}

func authenticateBetaToken(
	ctx context.Context,
	database *sql.DB,
	raw []byte,
	binding LocalAuthorityBinding,
	plane BetaTokenPlane,
	now time.Time,
) (BetaTokenAccess, bool, error) {
	var (
		tenant              string
		repository          string
		trustDomain         string
		namespace           string
		namespaceGeneration int64
		storedPlane         BetaTokenPlane
		access              BetaTokenAccess
		expiresAt           int64
		revokedAt           sql.NullInt64
	)
	err := database.QueryRowContext(
		ctx,
		`SELECT tenant_id, repository_id, trust_domain, namespace,
       namespace_generation, plane, access, expires_at_unix_ms,
       revoked_at_unix_ms
FROM beta_cache_tokens
WHERE token_digest = ?`,
		betaTokenDigest(raw),
	).Scan(
		&tenant,
		&repository,
		&trustDomain,
		&namespace,
		&namespaceGeneration,
		&storedPlane,
		&access,
		&expiresAt,
		&revokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	state := binding.state
	authorized := tenant == state.Repository.Tenant &&
		repository == state.Repository.Repository &&
		trustDomain == state.Repository.TrustDomain &&
		namespace == state.Namespace &&
		namespaceGeneration == state.NamespaceGeneration &&
		storedPlane == plane && revokedAt.Valid == false &&
		time.UnixMilli(expiresAt).After(now.UTC()) &&
		validBetaTokenAccess(access)
	return access, authorized, nil
}

func betaBearerToken(request *http.Request) ([]byte, bool) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return nil, false
	}
	encoded := strings.TrimPrefix(values[0], "Bearer ")
	if encoded == "" || strings.TrimSpace(encoded) != encoded {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) != BetaTokenBytes ||
		base64.RawURLEncoding.EncodeToString(raw) != encoded {
		clear(raw)
		return nil, false
	}
	return raw, true
}

func betaTokenDigest(raw []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("buildopt-private-beta-token-v1"))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(raw)
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func validBetaTokenScope(scope BetaTokenScope) bool {
	return validIdentifier(scope.Tenant) &&
		validIdentifier(scope.Repository) &&
		validIdentifier(scope.TrustDomain) &&
		validBetaNamespace(scope.Namespace) &&
		scope.NamespaceGeneration > 0 && validBetaTokenPlane(scope.Plane)
}

func validBetaNamespace(namespace string) bool {
	if !betaNamespacePattern.MatchString(namespace) ||
		strings.HasPrefix(namespace, "/") || path.Clean(namespace) != namespace {
		return false
	}
	for _, segment := range strings.Split(namespace, "/") {
		if segment == "." || segment == ".." || segment == "" {
			return false
		}
	}
	return true
}

func validBetaTokenPlane(plane BetaTokenPlane) bool {
	return plane == BetaTokenPlaneStable ||
		plane == BetaTokenPlaneQuarantine ||
		plane == BetaTokenPlaneControl
}

func validBetaTokenAccess(access BetaTokenAccess) bool {
	return access == BetaTokenRead || access == BetaTokenReadWrite
}

func writeBetaUnauthorized(response http.ResponseWriter) {
	response.Header().Set(
		"WWW-Authenticate",
		`Bearer realm="buildopt-private-beta-cache"`,
	)
	writeCacheStatus(response, http.StatusUnauthorized)
}
