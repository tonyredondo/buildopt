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

func runCentralToken(
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
		return runCentralTokenIssue(ctx, args[1:], stdout, stderr)
	case "revoke":
		return runCentralTokenRevoke(ctx, args[1:], stdout, stderr)
	default:
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
}

func runCentralTokenIssue(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("buildopt-server central-token issue", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRoot := flags.String("state-dir", "", "private central state root")
	repositoryScope := flags.String("repository-scope-sha256", "", "BuildOpt state repository scope")
	tenant := flags.String("tenant", "", "cache tenant")
	repository := flags.String("repository", "", "repository identity")
	trustDomain := flags.String("trust-domain", "", "cache trust domain")
	namespace := flags.String("namespace", "", "cache namespace")
	namespaceGeneration := flags.Int64("namespace-generation", 0, "cache namespace generation")
	capabilitiesText := flags.String(
		"capabilities",
		"",
		"comma-separated cache-read,cache-write,state-read,state-write",
	)
	expiresText := flags.String("expires-at", "", "RFC3339 expiration")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, *expiresText)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid central token expiration")
		return exitConfiguration
	}
	capabilities, ok := parseCentralCapabilities(*capabilitiesText)
	if !ok {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid central token capabilities")
		return exitConfiguration
	}
	registry, err := sharedcache.OpenCentralTokenRegistry(ctx, *stateRoot)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: central token registry unavailable")
		return exitConfiguration
	}
	defer registry.Close()
	issued, err := registry.Issue(ctx, sharedcache.CentralTokenIssueRequest{
		Scope: sharedcache.CentralTokenScope{
			RepositoryScopeSHA256: *repositoryScope,
			Tenant:                *tenant, Repository: *repository, TrustDomain: *trustDomain,
			Namespace: *namespace, NamespaceGeneration: *namespaceGeneration,
		},
		Capabilities: capabilities,
		ExpiresAt:    expiresAt,
	}, time.Now().UTC())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: central token issue rejected")
		return exitConfiguration
	}
	payload := struct {
		SchemaVersion         string                          `json:"schemaVersion"`
		TokenID               string                          `json:"tokenId"`
		Token                 string                          `json:"token"`
		RepositoryScopeSHA256 string                          `json:"repositoryScopeSha256"`
		Tenant                string                          `json:"tenant"`
		Repository            string                          `json:"repository"`
		TrustDomain           string                          `json:"trustDomain"`
		Namespace             string                          `json:"namespace"`
		NamespaceGeneration   int64                           `json:"namespaceGeneration"`
		Capabilities          []sharedcache.CentralCapability `json:"capabilities"`
		IssuedAt              string                          `json:"issuedAt"`
		ExpiresAt             string                          `json:"expiresAt"`
	}{
		SchemaVersion: "buildopt.central/access-token/v1",
		TokenID:       issued.TokenID, Token: issued.Token,
		RepositoryScopeSHA256: issued.Scope.RepositoryScopeSHA256,
		Tenant:                issued.Scope.Tenant, Repository: issued.Scope.Repository,
		TrustDomain: issued.Scope.TrustDomain, Namespace: issued.Scope.Namespace,
		NamespaceGeneration: issued.Scope.NamespaceGeneration,
		Capabilities:        issued.Capabilities,
		IssuedAt:            issued.IssuedAt.Format(time.RFC3339Nano),
		ExpiresAt:           issued.ExpiresAt.Format(time.RFC3339Nano),
	}
	if err := json.NewEncoder(stdout).Encode(payload); err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: write central token result failed")
		return 1
	}
	return 0
}

func runCentralTokenRevoke(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("buildopt-server central-token revoke", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRoot := flags.String("state-dir", "", "private central state root")
	tokenID := flags.String("token-id", "", "opaque token identifier")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	registry, err := sharedcache.OpenCentralTokenRegistry(ctx, *stateRoot)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: central token registry unavailable")
		return exitConfiguration
	}
	defer registry.Close()
	revoked, err := registry.Revoke(ctx, *tokenID, time.Now().UTC())
	if err != nil || !revoked {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: central token revoke rejected")
		return exitConfiguration
	}
	if err := json.NewEncoder(stdout).Encode(struct {
		TokenID string `json:"tokenId"`
		Revoked bool   `json:"revoked"`
	}{TokenID: *tokenID, Revoked: true}); err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: write central token result failed")
		return 1
	}
	return 0
}

func parseCentralCapabilities(value string) ([]sharedcache.CentralCapability, bool) {
	if value == "" || strings.TrimSpace(value) != value {
		return nil, false
	}
	var capabilities []sharedcache.CentralCapability
	for _, item := range strings.Split(value, ",") {
		switch strings.ToUpper(strings.ReplaceAll(item, "-", "_")) {
		case string(sharedcache.CentralCacheRead):
			capabilities = append(capabilities, sharedcache.CentralCacheRead)
		case string(sharedcache.CentralCacheWrite):
			capabilities = append(capabilities, sharedcache.CentralCacheWrite)
		case string(sharedcache.CentralStateRead):
			capabilities = append(capabilities, sharedcache.CentralStateRead)
		case string(sharedcache.CentralStateWrite):
			capabilities = append(capabilities, sharedcache.CentralStateWrite)
		default:
			return nil, false
		}
	}
	return capabilities, true
}
