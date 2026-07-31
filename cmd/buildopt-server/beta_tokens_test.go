package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestBetaTokenCLIissuesAndRevokesLiveRegistryToken(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "shared")
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano)
	var issuedOutput bytes.Buffer
	var stderr bytes.Buffer
	issueCode := runBetaToken(
		context.Background(),
		[]string{
			"issue",
			"--state-dir", stateRoot,
			"--tenant", "tenant-pilot",
			"--repository", "tonyredondo/buildopt",
			"--trust-domain", "private-beta",
			"--namespace", "stable",
			"--namespace-generation", "12",
			"--plane", "stable",
			"--access", "read-write",
			"--expires-at", expiresAt,
		},
		&issuedOutput,
		&stderr,
	)
	if issueCode != 0 || stderr.Len() != 0 {
		t.Fatalf("token issue = %d/%q", issueCode, stderr.String())
	}
	var issued struct {
		SchemaVersion string `json:"schemaVersion"`
		TokenID       string `json:"tokenId"`
		Token         string `json:"token"`
		Plane         string `json:"plane"`
		Access        string `json:"access"`
	}
	if err := json.Unmarshal(issuedOutput.Bytes(), &issued); err != nil {
		t.Fatalf("decode issued token: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(issued.Token)
	if err != nil || len(raw) != 32 || len(issued.TokenID) != 32 ||
		issued.SchemaVersion != "buildopt.private-beta-token/v1" ||
		issued.Plane != "STABLE" || issued.Access != "READ_WRITE" {
		t.Fatalf("unexpected issued token: %+v/%d/%v", issued, len(raw), err)
	}

	var revokedOutput bytes.Buffer
	stderr.Reset()
	revokeCode := runBetaToken(
		context.Background(),
		[]string{
			"revoke",
			"--state-dir", stateRoot,
			"--token-id", issued.TokenID,
		},
		&revokedOutput,
		&stderr,
	)
	if revokeCode != 0 || stderr.Len() != 0 {
		t.Fatalf("token revoke = %d/%q", revokeCode, stderr.String())
	}
	var revoked struct {
		TokenID string `json:"tokenId"`
		Revoked bool   `json:"revoked"`
	}
	if err := json.Unmarshal(revokedOutput.Bytes(), &revoked); err != nil ||
		revoked.TokenID != issued.TokenID || !revoked.Revoked {
		t.Fatalf("unexpected revoke result: %+v/%v", revoked, err)
	}
	stderr.Reset()
	if code := runBetaToken(
		context.Background(),
		[]string{
			"revoke",
			"--state-dir", stateRoot,
			"--token-id", issued.TokenID,
		},
		&bytes.Buffer{},
		&stderr,
	); code != exitConfiguration {
		t.Fatalf("duplicate revoke = %d/%q", code, stderr.String())
	}
}
