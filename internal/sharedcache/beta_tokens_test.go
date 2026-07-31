package sharedcache

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

func TestBetaTokensAreHashedScopedAndImmediatelyRevocable(t *testing.T) {
	storage := openLifecycleTestStorage(t)
	now := sharedAuthorityNow
	storage.clock = func() time.Time { return now }
	verified, credential, _, _ := sharedAuthorityFixture(t, func(
		document *localauthority.Document,
	) {
		document.Attempt.AllowRead = true
		document.Attempt.AllowWrite = true
	})
	binding, _, err := storage.InstallLocalAuthority(
		context.Background(),
		verified,
		credential,
		now,
	)
	if err != nil {
		t.Fatalf("install authority: %v", err)
	}
	handler, err := NewBetaTokenHTTPHandler(
		storage,
		binding,
		BetaTokenPlaneStable,
	)
	if err != nil {
		t.Fatalf("create beta token handler: %v", err)
	}

	scope := betaScopeForBinding(binding, BetaTokenPlaneStable)
	read := issueBetaTokenForTest(
		t,
		storage,
		scope,
		BetaTokenRead,
		now,
		now.Add(10*time.Minute),
	)
	readWrite := issueBetaTokenForTest(
		t,
		storage,
		scope,
		BetaTokenReadWrite,
		now,
		now.Add(10*time.Minute),
	)
	if read.Token == readWrite.Token || read.TokenID == readWrite.TokenID {
		t.Fatal("distinct permissions reused beta token material")
	}

	readMiss := authorityHTTPRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/read-miss",
		nil,
		read.Token,
		binding.AuthorityDigest,
	)
	if readMiss.Code != http.StatusNotFound {
		t.Fatalf("read token GET = %d, want 404", readMiss.Code)
	}
	readWriteDenied := authorityHTTPRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/read-denied",
		[]byte("payload"),
		read.Token,
		binding.AuthorityDigest,
	)
	if readWriteDenied.Code != http.StatusForbidden {
		t.Fatalf("read token PUT = %d, want 403", readWriteDenied.Code)
	}
	created := authorityHTTPRequest(
		t,
		handler,
		http.MethodPut,
		"/cache/read-write",
		[]byte("payload"),
		readWrite.Token,
		binding.AuthorityDigest,
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("read-write token PUT = %d, want 201", created.Code)
	}

	for name, mismatched := range map[string]BetaTokenScope{
		"repository": func() BetaTokenScope {
			candidate := scope
			candidate.Repository = "other/repository"
			return candidate
		}(),
		"namespace": func() BetaTokenScope {
			candidate := scope
			candidate.Namespace = "stable-other"
			return candidate
		}(),
		"namespace generation": func() BetaTokenScope {
			candidate := scope
			candidate.NamespaceGeneration++
			return candidate
		}(),
		"quarantine plane": func() BetaTokenScope {
			candidate := scope
			candidate.Plane = BetaTokenPlaneQuarantine
			return candidate
		}(),
		"control plane": func() BetaTokenScope {
			candidate := scope
			candidate.Plane = BetaTokenPlaneControl
			return candidate
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			token := issueBetaTokenForTest(
				t,
				storage,
				mismatched,
				BetaTokenReadWrite,
				now,
				now.Add(10*time.Minute),
			)
			response := authorityHTTPRequest(
				t,
				handler,
				http.MethodGet,
				"/cache/cross-scope",
				nil,
				token.Token,
				binding.AuthorityDigest,
			)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("cross-scope GET = %d, want 401", response.Code)
			}
		})
	}

	var storedDigest string
	if err := storage.control.database.QueryRow(
		"SELECT token_digest FROM beta_cache_tokens WHERE token_id = ?",
		read.TokenID,
	).Scan(&storedDigest); err != nil {
		t.Fatalf("read stored token digest: %v", err)
	}
	rawRead, err := base64.RawURLEncoding.DecodeString(read.Token)
	if err != nil {
		t.Fatal(err)
	}
	if storedDigest != betaTokenDigest(rawRead) ||
		strings.Contains(storedDigest, read.Token) {
		t.Fatalf("stored token value = %q, want only domain-separated hash", storedDigest)
	}
	assertTokenAbsentFromSQLiteFiles(t, storage.Layout(), read.Token, rawRead)

	registry, err := OpenBetaTokenRegistry(
		context.Background(),
		storage.Layout().Root,
	)
	if err != nil {
		t.Fatalf("open live beta token registry: %v", err)
	}
	defer registry.Close()
	revoked, err := registry.Revoke(
		context.Background(),
		read.TokenID,
		now.Add(time.Second),
	)
	if err != nil || !revoked {
		t.Fatalf("revoke beta token = %t/%v", revoked, err)
	}
	revokedRequest := authorityHTTPRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/revoked-before-next-build",
		nil,
		read.Token,
		binding.AuthorityDigest,
	)
	if revokedRequest.Code != http.StatusUnauthorized {
		t.Fatalf("revoked next request = %d, want 401", revokedRequest.Code)
	}
	if changed, err := registry.Revoke(
		context.Background(),
		read.TokenID,
		now.Add(2*time.Second),
	); err != nil || changed {
		t.Fatalf("idempotent revoke = %t/%v", changed, err)
	}

	expiring := issueBetaTokenForTest(
		t,
		storage,
		scope,
		BetaTokenRead,
		now,
		now.Add(time.Minute),
	)
	now = now.Add(2 * time.Minute)
	expired := authorityHTTPRequest(
		t,
		handler,
		http.MethodGet,
		"/cache/expired",
		nil,
		expiring.Token,
		binding.AuthorityDigest,
	)
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("expired token GET = %d, want 401", expired.Code)
	}
}

func TestBetaTokenProvisioningRejectsUnsafeLifetimeAndScope(t *testing.T) {
	storage := openLifecycleTestStorage(t)
	now := sharedAuthorityNow
	valid := BetaTokenIssueRequest{
		Scope: BetaTokenScope{
			Tenant:              "tenant-test",
			Repository:          "tonyredondo/buildopt",
			TrustDomain:         "private-beta",
			Namespace:           "stable",
			NamespaceGeneration: 12,
			Plane:               BetaTokenPlaneStable,
		},
		Access:    BetaTokenRead,
		ExpiresAt: now.Add(BetaTokenMaximumLifetime),
	}
	if _, err := storage.IssueBetaToken(
		context.Background(),
		valid,
		now,
	); err != nil {
		t.Fatalf("issue maximum-lifetime token: %v", err)
	}
	tooLong := valid
	tooLong.ExpiresAt = now.Add(BetaTokenMaximumLifetime + time.Nanosecond)
	if _, err := storage.IssueBetaToken(
		context.Background(),
		tooLong,
		now,
	); err == nil {
		t.Fatal("accepted token beyond 30-day maximum")
	}
	badScope := valid
	badScope.Scope.Namespace = "../stable"
	if _, err := storage.IssueBetaToken(
		context.Background(),
		badScope,
		now,
	); err == nil {
		t.Fatal("accepted traversal-like namespace")
	}
}

func betaScopeForBinding(
	binding LocalAuthorityBinding,
	plane BetaTokenPlane,
) BetaTokenScope {
	state := binding.state
	return BetaTokenScope{
		Tenant:              state.Repository.Tenant,
		Repository:          state.Repository.Repository,
		TrustDomain:         state.Repository.TrustDomain,
		Namespace:           state.Namespace,
		NamespaceGeneration: state.NamespaceGeneration,
		Plane:               plane,
	}
}

func issueBetaTokenForTest(
	t *testing.T,
	storage *Storage,
	scope BetaTokenScope,
	access BetaTokenAccess,
	now time.Time,
	expiresAt time.Time,
) IssuedBetaToken {
	t.Helper()
	issued, err := storage.IssueBetaToken(
		context.Background(),
		BetaTokenIssueRequest{
			Scope:     scope,
			Access:    access,
			ExpiresAt: expiresAt,
		},
		now,
	)
	if err != nil {
		t.Fatalf("issue beta token: %v", err)
	}
	return issued
}

func assertTokenAbsentFromSQLiteFiles(
	t *testing.T,
	layout Layout,
	encoded string,
	raw []byte,
) {
	t.Helper()
	for _, path := range []string{
		layout.ControlDatabase,
		layout.ControlDatabase + "-wal",
		layout.ControlDatabase + "-shm",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("read %s: %v", filepath.Base(path), err)
		}
		if bytes.Contains(content, []byte(encoded)) || bytes.Contains(content, raw) {
			t.Fatalf("raw beta token persisted in %s", filepath.Base(path))
		}
	}
}
