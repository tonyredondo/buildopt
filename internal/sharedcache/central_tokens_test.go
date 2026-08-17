package sharedcache

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestCentralTokensAreScopedHashedExpiringAndRevocable(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, t.TempDir()+"/shared")
	defer storage.Close()
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	storage.clock = func() time.Time { return now }
	scope := centralTestScope()

	issued, err := storage.IssueCentralToken(ctx, CentralTokenIssueRequest{
		Scope: scope,
		Capabilities: []CentralCapability{
			CentralStateWrite,
			CentralCacheRead,
		},
		ExpiresAt: now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(issued.Token) != 43 || len(issued.TokenID) != 32 ||
		len(issued.Capabilities) != 2 || issued.Capabilities[0] != CentralCacheRead ||
		issued.Capabilities[1] != CentralStateWrite {
		t.Fatalf("issued token = %+v", issued)
	}
	var digest string
	if err := storage.control.database.QueryRow(
		`SELECT token_digest FROM central_access_tokens WHERE token_id = ?`,
		issued.TokenID,
	).Scan(&digest); err != nil {
		t.Fatal(err)
	}
	if digest == issued.Token || strings.Contains(digest, issued.Token) ||
		!strings.HasPrefix(digest, digestPrefix) {
		t.Fatalf("durable token digest = %q", digest)
	}
	raw, err := base64.RawURLEncoding.DecodeString(issued.Token)
	if err != nil {
		t.Fatal(err)
	}
	authorization, ok, err := authenticateCentralToken(ctx, storage.control.database, raw, now)
	if err != nil || !ok || authorization.Scope != scope ||
		!authorization.Has(CentralCacheRead) ||
		!authorization.Has(CentralStateWrite) ||
		authorization.Has(CentralCacheWrite) {
		t.Fatalf("authorization = %+v/%t/%v", authorization, ok, err)
	}
	if _, ok, err := authenticateCentralToken(
		ctx, storage.control.database, raw, now.Add(time.Hour),
	); err != nil || ok {
		t.Fatalf("expired authorization = %t/%v", ok, err)
	}
	revoked, err := storage.RevokeCentralToken(ctx, issued.TokenID, now.Add(time.Minute))
	if err != nil || !revoked {
		t.Fatalf("revoke = %t/%v", revoked, err)
	}
	if _, ok, err := authenticateCentralToken(
		ctx, storage.control.database, raw, now.Add(2*time.Minute),
	); err != nil || ok {
		t.Fatalf("revoked authorization = %t/%v", ok, err)
	}
	if revoked, err := storage.RevokeCentralToken(
		ctx, issued.TokenID, now.Add(3*time.Minute),
	); err != nil || revoked {
		t.Fatalf("repeat revoke = %t/%v", revoked, err)
	}
}

func TestCentralTokenIssueRejectsAmbiguousAuthority(t *testing.T) {
	ctx := context.Background()
	storage := openStateTestStorage(t, ctx, t.TempDir()+"/shared")
	defer storage.Close()
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	scope := centralTestScope()
	testCases := []CentralTokenIssueRequest{
		{Scope: scope, ExpiresAt: now.Add(time.Hour)},
		{Scope: scope, Capabilities: []CentralCapability{CentralCacheRead, CentralCacheRead}, ExpiresAt: now.Add(time.Hour)},
		{Scope: scope, Capabilities: []CentralCapability{"ADMIN"}, ExpiresAt: now.Add(time.Hour)},
		{Scope: scope, Capabilities: []CentralCapability{CentralCacheRead}, ExpiresAt: now.Add(CentralTokenMaximumLifetime + time.Second)},
	}
	invalidScope := CentralTokenIssueRequest{
		Scope: scope, Capabilities: []CentralCapability{CentralCacheRead}, ExpiresAt: now.Add(time.Hour),
	}
	invalidScope.Scope.RepositoryScopeSHA256 = strings.Repeat("a", 63)
	testCases = append(testCases, invalidScope)
	for index, request := range testCases {
		if _, err := storage.IssueCentralToken(ctx, request, now); err == nil {
			t.Fatalf("case %d accepted", index)
		}
	}
}

func centralTestScope() CentralTokenScope {
	return CentralTokenScope{
		RepositoryScopeSHA256: stateTestScope,
		Tenant:                "tenant-test",
		Repository:            "repository-test",
		TrustDomain:           "trust-test",
		Namespace:             "gradle-9.6.1/linux-amd64/jdk-21/project",
		NamespaceGeneration:   1,
	}
}
