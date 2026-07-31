package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func runBetaToken(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	switch args[0] {
	case "issue":
		return runBetaTokenIssue(ctx, args[1:], stdout, stderr)
	case "revoke":
		return runBetaTokenRevoke(ctx, args[1:], stdout, stderr)
	default:
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
}

func runBetaTokenIssue(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("buildopt-server token issue", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRoot := flags.String("state-dir", "", "private Shared state root")
	tenant := flags.String("tenant", "", "deployment tenant")
	repository := flags.String("repository", "", "repository identity")
	trustDomain := flags.String("trust-domain", "", "trust domain")
	namespace := flags.String("namespace", "", "cache namespace")
	namespaceGeneration := flags.Int64(
		"namespace-generation",
		0,
		"cache namespace generation",
	)
	planeText := flags.String("plane", "", "stable, quarantine, or control")
	accessText := flags.String("access", "", "read or read-write")
	expiresText := flags.String("expires-at", "", "RFC3339 expiration")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, *expiresText)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid beta token expiration")
		return exitConfiguration
	}
	plane, ok := parseBetaTokenPlane(*planeText)
	if !ok {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid beta token plane")
		return exitConfiguration
	}
	access, ok := parseBetaTokenAccess(*accessText)
	if !ok {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid beta token access")
		return exitConfiguration
	}
	registry, err := sharedcache.OpenBetaTokenRegistry(ctx, *stateRoot)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: beta token registry unavailable")
		return exitConfiguration
	}
	defer registry.Close()
	now := time.Now().UTC()
	issued, err := registry.Issue(
		ctx,
		sharedcache.BetaTokenIssueRequest{
			Scope: sharedcache.BetaTokenScope{
				Tenant:              *tenant,
				Repository:          *repository,
				TrustDomain:         *trustDomain,
				Namespace:           *namespace,
				NamespaceGeneration: *namespaceGeneration,
				Plane:               plane,
			},
			Access:    access,
			ExpiresAt: expiresAt,
		},
		now,
	)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: beta token issue rejected")
		return exitConfiguration
	}
	payload := struct {
		SchemaVersion string `json:"schemaVersion"`
		TokenID       string `json:"tokenId"`
		Token         string `json:"token"`
		Tenant        string `json:"tenant"`
		Repository    string `json:"repository"`
		TrustDomain   string `json:"trustDomain"`
		Namespace     string `json:"namespace"`
		Generation    int64  `json:"namespaceGeneration"`
		Plane         string `json:"plane"`
		Access        string `json:"access"`
		IssuedAt      string `json:"issuedAt"`
		ExpiresAt     string `json:"expiresAt"`
	}{
		SchemaVersion: "buildopt.private-beta-token/v1",
		TokenID:       issued.TokenID,
		Token:         issued.Token,
		Tenant:        issued.Scope.Tenant,
		Repository:    issued.Scope.Repository,
		TrustDomain:   issued.Scope.TrustDomain,
		Namespace:     issued.Scope.Namespace,
		Generation:    issued.Scope.NamespaceGeneration,
		Plane:         string(issued.Scope.Plane),
		Access:        string(issued.Access),
		IssuedAt:      issued.IssuedAt.Format(time.RFC3339Nano),
		ExpiresAt:     issued.ExpiresAt.Format(time.RFC3339Nano),
	}
	if err := json.NewEncoder(stdout).Encode(payload); err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: write beta token result failed")
		return 1
	}
	return 0
}

func runBetaTokenRevoke(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("buildopt-server token revoke", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRoot := flags.String("state-dir", "", "private Shared state root")
	tokenID := flags.String("token-id", "", "opaque token identifier")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	registry, err := sharedcache.OpenBetaTokenRegistry(ctx, *stateRoot)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: beta token registry unavailable")
		return exitConfiguration
	}
	defer registry.Close()
	revoked, err := registry.Revoke(ctx, *tokenID, time.Now().UTC())
	if err != nil || !revoked {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: beta token revoke rejected")
		return exitConfiguration
	}
	if err := json.NewEncoder(stdout).Encode(struct {
		TokenID string `json:"tokenId"`
		Revoked bool   `json:"revoked"`
	}{TokenID: *tokenID, Revoked: true}); err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: write beta token result failed")
		return 1
	}
	return 0
}

func parseBetaTokenPlane(value string) (sharedcache.BetaTokenPlane, bool) {
	switch strings.ToUpper(value) {
	case string(sharedcache.BetaTokenPlaneStable):
		return sharedcache.BetaTokenPlaneStable, true
	case string(sharedcache.BetaTokenPlaneQuarantine):
		return sharedcache.BetaTokenPlaneQuarantine, true
	case string(sharedcache.BetaTokenPlaneControl):
		return sharedcache.BetaTokenPlaneControl, true
	default:
		return "", false
	}
}

func parseBetaTokenAccess(value string) (sharedcache.BetaTokenAccess, bool) {
	switch strings.ToUpper(strings.ReplaceAll(value, "-", "_")) {
	case string(sharedcache.BetaTokenRead):
		return sharedcache.BetaTokenRead, true
	case string(sharedcache.BetaTokenReadWrite):
		return sharedcache.BetaTokenReadWrite, true
	default:
		return "", false
	}
}
