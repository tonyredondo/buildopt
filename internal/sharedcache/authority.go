package sharedcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

const (
	// AuthorityDigestHeader binds a data-plane credential to one exact signed
	// local authority document.
	AuthorityDigestHeader = "X-BuildOpt-Authority-Digest"
)

// LocalAuthorityBinding is immutable server-side authority for one exact
// policy, revocation state, namespace, and optional pending writer.
type LocalAuthorityBinding struct {
	AuthorityDigest string
	AttemptID       string
	ExpiresAt       time.Time
	AllowRead       bool
	AllowWrite      bool

	httpBinding      HTTPBinding
	scopeDigest      string
	credentialDigest string
	canonical        []byte
	state            localauthority.State
}

// InstallLocalAuthority persists authenticated monotonic state, records the
// exact canonical document, and opens its pending attempt when writes are
// authorized. No raw data-plane credential is persisted.
func (storage *Storage) InstallLocalAuthority(
	ctx context.Context,
	verified localauthority.Verified,
	credential []byte,
	now time.Time,
) (LocalAuthorityBinding, bool, error) {
	if storage == nil || ctx == nil {
		return LocalAuthorityBinding{}, false, errors.New(
			"install local cache authority: invalid storage or context",
		)
	}
	if err := ctx.Err(); err != nil {
		return LocalAuthorityBinding{}, false, err
	}
	document := verified.Document()
	canonical := verified.CanonicalDocument()
	if len(canonical) == 0 ||
		document.Attempt.CredentialDigest != localCredentialDigest(credential) ||
		!verified.ExpiresAt().After(now.UTC()) {
		return LocalAuthorityBinding{}, false, errors.New(
			"install local cache authority: invalid verified binding",
		)
	}

	next := localauthority.StateFromVerified(verified, now.UTC())
	binding := LocalAuthorityBinding{
		AuthorityDigest: document.AuthorityDigest,
		AttemptID:       document.Attempt.AttemptID,
		ExpiresAt:       verified.ExpiresAt(),
		AllowRead:       document.Attempt.AllowRead,
		AllowWrite:      document.Attempt.AllowWrite,
		httpBinding: HTTPBinding{
			Tenant:              document.Repository.Tenant,
			NamespaceGeneration: document.Policy.RemoteCache.NamespaceGeneration,
			AllowRead:           document.Attempt.AllowRead,
			AllowWrite:          document.Attempt.AllowWrite,
		},
		scopeDigest:      next.ScopeDigest,
		credentialDigest: document.Attempt.CredentialDigest,
		canonical:        bytes.Clone(canonical),
		state:            next,
	}
	if binding.AllowWrite {
		binding.httpBinding.PendingAttemptID = binding.AttemptID
	}
	if _, err := NewHTTPHandler(storage, binding.httpBinding); err != nil {
		return LocalAuthorityBinding{}, false, fmt.Errorf(
			"install local cache authority: %w",
			err,
		)
	}

	storage.authorityMutex.Lock()
	defer storage.authorityMutex.Unlock()

	finish, err := storage.beginOperation()
	if err != nil {
		return LocalAuthorityBinding{}, false, err
	}
	changed, err := storage.installAuthorityControlState(
		ctx,
		binding,
		next,
		now.UTC(),
	)
	finish()
	if err != nil {
		return LocalAuthorityBinding{}, false, err
	}
	storage.currentAuthorityDigests[binding.scopeDigest] =
		binding.AuthorityDigest

	if binding.AllowWrite {
		status, created, err := storage.StartAttempt(
			ctx,
			StartAttemptRequest{
				RequestID:       document.Revocation.RequestID,
				AttemptID:       document.Attempt.AttemptID,
				AuthorityDigest: document.AuthorityDigest,
				Repository: RepositoryIdentity{
					Tenant:      document.Repository.Tenant,
					Repository:  document.Repository.Repository,
					TrustDomain: document.Repository.TrustDomain,
				},
				NamespaceGeneration: document.Policy.RemoteCache.
					NamespaceGeneration,
				SourceRevision:    document.SourceRevision,
				SourceStateDigest: document.SourceStateDigest,
				PolicyDigest:      document.Policy.PolicyDigest,
				ConfigurationPolicyDigest: document.Policy.
					ConfigurationPolicyDigest,
				CacheContractDigest: document.CacheContractDigest,
				OwnerID:             document.Attempt.OwnerID,
				LeaseID:             document.Attempt.LeaseID,
				LeaseExpiresAt: mustParseAuthorityTime(
					document.Attempt.LeaseExpiresAt,
				),
			},
		)
		if err != nil {
			return LocalAuthorityBinding{}, false, fmt.Errorf(
				"install local cache authority attempt: %w",
				err,
			)
		}
		if status.AuthorityDigest != binding.AuthorityDigest {
			return LocalAuthorityBinding{}, false, errors.New(
				"install local cache authority: attempt binding was not durable",
			)
		}
		changed = changed || created
	}

	current, err := storage.localAuthorityCurrentLocked(
		ctx,
		binding,
		now.UTC(),
	)
	if err != nil {
		return LocalAuthorityBinding{}, false, err
	}
	if !current {
		return LocalAuthorityBinding{}, false, errors.New(
			"install local cache authority: installed state is not current",
		)
	}
	return binding, changed, nil
}

// NewLocalAuthorityHTTPHandler creates the authenticated server-side
// Gradle-compatible route for one installed authority. The durable tables are
// validated at installation; the hot path rejects superseded state through
// the matching monotonic in-process projection before delegating to the
// already context-bound cache handler.
func NewLocalAuthorityHTTPHandler(
	storage *Storage,
	binding LocalAuthorityBinding,
	credential []byte,
) (http.Handler, error) {
	if storage == nil ||
		binding.AuthorityDigest == "" ||
		binding.credentialDigest != localCredentialDigest(credential) {
		return nil, errors.New("invalid authenticated Shared cache binding")
	}
	bound, err := NewHTTPHandler(storage, binding.httpBinding)
	if err != nil {
		return nil, err
	}
	expectedAuthorization := []byte(
		"Bearer " + base64.RawURLEncoding.EncodeToString(credential),
	)
	return http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		response.Header().Set("Cache-Control", "no-store")
		authorization := []byte(request.Header.Get("Authorization"))
		digests := request.Header.Values(AuthorityDigestHeader)
		if len(authorization) != len(expectedAuthorization) ||
			subtle.ConstantTimeCompare(
				authorization,
				expectedAuthorization,
			) != 1 ||
			len(digests) != 1 ||
			digests[0] != binding.AuthorityDigest {
			response.Header().Set(
				"WWW-Authenticate",
				`Bearer realm="buildopt-shared-cache"`,
			)
			writeCacheStatus(response, http.StatusUnauthorized)
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
			writeCacheStatus(response, http.StatusUnauthorized)
			return
		}
		bound.ServeHTTP(response, request)
	}), nil
}

func (storage *Storage) installAuthorityControlState(
	ctx context.Context,
	binding LocalAuthorityBinding,
	next localauthority.State,
	now time.Time,
) (bool, error) {
	transaction, err := storage.control.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return false, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()

	current, found, err := loadLocalAuthorityState(
		ctx,
		transaction,
		next.ScopeDigest,
	)
	if err != nil {
		return false, err
	}
	if err := localauthority.Advance(current, next); err != nil {
		return false, err
	}
	stateChanged := !found || !sameAuthorityState(current, next)
	if stateChanged {
		policyExpiresAt, err := time.Parse(
			time.RFC3339Nano,
			next.PolicyExpiresAt,
		)
		if err != nil {
			return false, err
		}
		_, err = transaction.ExecContext(
			ctx,
			`INSERT INTO local_authority_state (
    scope_digest, tenant_id, repository_id, trust_domain, policy_id,
    policy_version, policy_digest, configuration_policy_digest,
    revocation_epoch, revocation_digest, l1_security_generation,
    gateway_connection_generation, namespace, namespace_generation,
    policy_expires_at_unix_ms, installed_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scope_digest) DO UPDATE SET
    tenant_id = excluded.tenant_id,
    repository_id = excluded.repository_id,
    trust_domain = excluded.trust_domain,
    policy_id = excluded.policy_id,
    policy_version = excluded.policy_version,
    policy_digest = excluded.policy_digest,
    configuration_policy_digest = excluded.configuration_policy_digest,
    revocation_epoch = excluded.revocation_epoch,
    revocation_digest = excluded.revocation_digest,
    l1_security_generation = excluded.l1_security_generation,
    gateway_connection_generation = excluded.gateway_connection_generation,
    namespace = excluded.namespace,
    namespace_generation = excluded.namespace_generation,
    policy_expires_at_unix_ms = excluded.policy_expires_at_unix_ms,
    installed_at_unix_ms = excluded.installed_at_unix_ms`,
			next.ScopeDigest,
			next.Repository.Tenant,
			next.Repository.Repository,
			next.Repository.TrustDomain,
			next.PolicyID,
			next.PolicyVersion,
			next.PolicyDigest,
			next.ConfigurationPolicyDigest,
			next.RevocationEpoch,
			next.RevocationDigest,
			next.L1SecurityGeneration,
			next.GatewayConnectionGeneration,
			next.Namespace,
			next.NamespaceGeneration,
			policyExpiresAt.UnixMilli(),
			now.UnixMilli(),
		)
		if err != nil {
			return false, err
		}
	}

	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO local_authority_documents (
    authority_digest, scope_digest, attempt_id, credential_digest,
    allow_read, allow_write, canonical_document, expires_at_unix_ms,
    registered_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(authority_digest) DO NOTHING`,
		binding.AuthorityDigest,
		binding.scopeDigest,
		binding.AttemptID,
		binding.credentialDigest,
		boolInteger(binding.AllowRead),
		boolInteger(binding.AllowWrite),
		binding.canonical,
		binding.ExpiresAt.UnixMilli(),
		now.UnixMilli(),
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	documentAdded := rows == 1
	if !documentAdded {
		var canonical []byte
		if err := transaction.QueryRowContext(
			ctx,
			`SELECT canonical_document
FROM local_authority_documents
WHERE authority_digest = ?`,
			binding.AuthorityDigest,
		).Scan(&canonical); err != nil {
			return false, err
		}
		if !bytes.Equal(canonical, binding.canonical) {
			return false, ErrIdempotencyConflict
		}
	}

	if err := transaction.Commit(); err != nil {
		return false, err
	}
	rollback = false
	return stateChanged || documentAdded, nil
}

func (storage *Storage) localAuthorityCurrentLocked(
	ctx context.Context,
	binding LocalAuthorityBinding,
	now time.Time,
) (bool, error) {
	if ctx == nil || !binding.ExpiresAt.After(now.UTC()) {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	current, found := storage.currentAuthorityDigests[binding.scopeDigest]
	return found && current == binding.AuthorityDigest, nil
}

type localAuthorityStateQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadLocalAuthorityState(
	ctx context.Context,
	queryer localAuthorityStateQuery,
	scopeDigest string,
) (localauthority.State, bool, error) {
	var (
		state           localauthority.State
		policyExpiresAt int64
		installedAt     int64
	)
	err := queryer.QueryRowContext(
		ctx,
		`SELECT tenant_id, repository_id, trust_domain, policy_id,
       policy_version, policy_digest, configuration_policy_digest,
       revocation_epoch, revocation_digest, l1_security_generation,
       gateway_connection_generation, namespace, namespace_generation,
       policy_expires_at_unix_ms, installed_at_unix_ms
FROM local_authority_state
WHERE scope_digest = ?`,
		scopeDigest,
	).Scan(
		&state.Repository.Tenant,
		&state.Repository.Repository,
		&state.Repository.TrustDomain,
		&state.PolicyID,
		&state.PolicyVersion,
		&state.PolicyDigest,
		&state.ConfigurationPolicyDigest,
		&state.RevocationEpoch,
		&state.RevocationDigest,
		&state.L1SecurityGeneration,
		&state.GatewayConnectionGeneration,
		&state.Namespace,
		&state.NamespaceGeneration,
		&policyExpiresAt,
		&installedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return localauthority.State{}, false, nil
	}
	if err != nil {
		return localauthority.State{}, false, err
	}
	state.SchemaVersion = localauthority.StateContractVersion
	state.ScopeDigest = scopeDigest
	state.PolicyExpiresAt = time.UnixMilli(policyExpiresAt).
		UTC().
		Format(time.RFC3339Nano)
	state.InstalledAt = time.UnixMilli(installedAt).
		UTC().
		Format(time.RFC3339Nano)
	return state, true, nil
}

func sameAuthorityState(left localauthority.State, right localauthority.State) bool {
	leftExpiry, leftErr := time.Parse(
		time.RFC3339Nano,
		left.PolicyExpiresAt,
	)
	rightExpiry, rightErr := time.Parse(
		time.RFC3339Nano,
		right.PolicyExpiresAt,
	)
	left.InstalledAt = ""
	right.InstalledAt = ""
	left.PolicyExpiresAt = ""
	right.PolicyExpiresAt = ""
	return leftErr == nil &&
		rightErr == nil &&
		leftExpiry.UnixMilli() == rightExpiry.UnixMilli() &&
		left == right
}

func localCredentialDigest(credential []byte) string {
	digest := sha256.Sum256(credential)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func boolInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}

func mustParseAuthorityTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		panic("verified authority contains invalid timestamp")
	}
	return parsed
}
