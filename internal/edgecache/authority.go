package edgecache

import (
	"errors"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

// ReadAuthority is an immutable projection of one fully verified local
// authority. Its unexported fields prevent callers from constructing cache
// read authority without passing localauthority.Verify first.
type ReadAuthority struct {
	tenant               string
	repository           string
	trustDomain          string
	namespace            string
	namespaceGeneration  int64
	authorityDigest      string
	revocationEpoch      int64
	revocationDigest     string
	l1SecurityGeneration int64
	expiresAt            time.Time
}

// WriteAuthority is an immutable projection of one fully verified,
// attempt-scoped pending-write grant. Edge never derives stable-read authority
// from this projection.
type WriteAuthority struct {
	tenant               string
	repository           string
	trustDomain          string
	namespace            string
	namespaceGeneration  int64
	authorityDigest      string
	revocationEpoch      int64
	revocationDigest     string
	l1SecurityGeneration int64
	attemptID            string
	expiresAt            time.Time
}

// NewReadAuthority projects a current signed authority into the exact state
// that every online or offline Edge read must match.
func NewReadAuthority(verified localauthority.Verified, now time.Time) (ReadAuthority, error) {
	document := verified.Document()
	now = now.UTC()
	if now.IsZero() || !verified.ExpiresAt().After(now) ||
		!document.Policy.RemoteCache.Read ||
		!verified.AllowsAction("REMOTE_CACHE_ALLOWLISTED") ||
		document.AuthorityDigest == "" ||
		document.Repository.Tenant == "" ||
		document.Repository.Repository == "" ||
		document.Repository.TrustDomain == "" ||
		document.Policy.RemoteCache.Namespace == "" ||
		document.Policy.RemoteCache.NamespaceGeneration < 1 ||
		document.Revocation.RevocationEpoch < 1 ||
		document.Revocation.CumulativeStateDigest == "" ||
		document.Revocation.L1SecurityGeneration < 1 {
		return ReadAuthority{}, errors.New("Edge read authority is absent, expired, or incomplete")
	}
	return ReadAuthority{
		tenant:               document.Repository.Tenant,
		repository:           document.Repository.Repository,
		trustDomain:          document.Repository.TrustDomain,
		namespace:            document.Policy.RemoteCache.Namespace,
		namespaceGeneration:  document.Policy.RemoteCache.NamespaceGeneration,
		authorityDigest:      document.AuthorityDigest,
		revocationEpoch:      document.Revocation.RevocationEpoch,
		revocationDigest:     document.Revocation.CumulativeStateDigest,
		l1SecurityGeneration: document.Revocation.L1SecurityGeneration,
		expiresAt:            verified.ExpiresAt(),
	}, nil
}

// NewWriteAuthority projects a current signed authority into the exact
// attempt scope accepted for local pending writes and Shared replication.
func NewWriteAuthority(verified localauthority.Verified, now time.Time) (WriteAuthority, error) {
	document := verified.Document()
	now = now.UTC()
	leaseExpiresAt, err := time.Parse(time.RFC3339Nano, document.Attempt.LeaseExpiresAt)
	if err != nil || now.IsZero() || !verified.ExpiresAt().After(now) ||
		!leaseExpiresAt.After(now) || !document.Attempt.AllowWrite ||
		document.Policy.RemoteCache.Write != "TRUSTED_CI_ONLY" ||
		!verified.AllowsAction("REMOTE_CACHE_ALLOWLISTED") ||
		document.AuthorityDigest == "" ||
		document.Repository.Tenant == "" ||
		document.Repository.Repository == "" ||
		document.Repository.TrustDomain == "" ||
		document.Policy.RemoteCache.Namespace == "" ||
		document.Policy.RemoteCache.NamespaceGeneration < 1 ||
		document.Attempt.AttemptID == "" ||
		document.Revocation.RevocationEpoch < 1 ||
		document.Revocation.CumulativeStateDigest == "" ||
		document.Revocation.L1SecurityGeneration < 1 {
		return WriteAuthority{}, errors.New("Edge write authority is absent, expired, or incomplete")
	}
	expiresAt := verified.ExpiresAt()
	if leaseExpiresAt.Before(expiresAt) {
		expiresAt = leaseExpiresAt
	}
	return WriteAuthority{
		tenant:               document.Repository.Tenant,
		repository:           document.Repository.Repository,
		trustDomain:          document.Repository.TrustDomain,
		namespace:            document.Policy.RemoteCache.Namespace,
		namespaceGeneration:  document.Policy.RemoteCache.NamespaceGeneration,
		authorityDigest:      document.AuthorityDigest,
		revocationEpoch:      document.Revocation.RevocationEpoch,
		revocationDigest:     document.Revocation.CumulativeStateDigest,
		l1SecurityGeneration: document.Revocation.L1SecurityGeneration,
		attemptID:            document.Attempt.AttemptID,
		expiresAt:            expiresAt,
	}, nil
}

func (authority ReadAuthority) current(now time.Time) bool {
	return !now.IsZero() && authority.authorityDigest != "" &&
		authority.expiresAt.After(now.UTC())
}

func (authority ReadAuthority) matches(entry storedEntry) bool {
	return entry.Tenant == authority.tenant &&
		entry.Repository == authority.repository &&
		entry.TrustDomain == authority.trustDomain &&
		entry.Namespace == authority.namespace &&
		entry.NamespaceGeneration == authority.namespaceGeneration &&
		entry.AuthorityDigest == authority.authorityDigest &&
		entry.RevocationEpoch == authority.revocationEpoch &&
		entry.RevocationDigest == authority.revocationDigest &&
		entry.L1SecurityGeneration == authority.l1SecurityGeneration
}

func (authority WriteAuthority) current(now time.Time) bool {
	return !now.IsZero() && authority.authorityDigest != "" &&
		authority.attemptID != "" && authority.expiresAt.After(now.UTC())
}
