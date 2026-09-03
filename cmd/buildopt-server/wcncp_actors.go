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

func runWCNCPActor(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "grant" {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	return runWCNCPActorGrant(ctx, args[1:], stdout, stderr)
}

func runWCNCPActorGrant(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("buildopt-server wcncp-actor grant", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateRoot := flags.String("state-dir", "", "private central state root")
	tokenID := flags.String("token-id", "", "opaque central token identifier")
	actorText := flags.String("actor", "", "WCNCP actor")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	actor, ok := parseWCNCPActor(*actorText)
	if !ok {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid WCNCP actor")
		return exitConfiguration
	}
	storage, err := sharedcache.Open(ctx, *stateRoot)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: central state unavailable")
		return exitConfiguration
	}
	defer storage.Close()
	grant, err := storage.GrantWCNCPActor(ctx, *tokenID, actor, time.Now().UTC())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: WCNCP actor grant rejected")
		return exitConfiguration
	}
	payload := struct {
		SchemaVersion string                 `json:"schemaVersion"`
		TokenID       string                 `json:"tokenId"`
		Actor         sharedcache.WCNCPActor `json:"actor"`
		Granted       bool                   `json:"granted"`
	}{"buildopt.wcncp/actor-grant-result/v1", grant.TokenID, grant.Actor, true}
	if err := json.NewEncoder(stdout).Encode(payload); err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: write WCNCP actor grant result failed")
		return 1
	}
	return 0
}

func parseWCNCPActor(value string) (sharedcache.WCNCPActor, bool) {
	normalized := strings.ToUpper(strings.ReplaceAll(value, "-", "_"))
	actor := sharedcache.WCNCPActor(normalized)
	switch actor {
	case sharedcache.WCNCPActorDeveloper, sharedcache.WCNCPActorTrustedObserver,
		sharedcache.WCNCPActorValidator, sharedcache.WCNCPActorOwner,
		sharedcache.WCNCPActorAdmin:
		return actor, true
	default:
		return "", false
	}
}
