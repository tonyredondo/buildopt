package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunAuthorityInspectReturnsOnlyVerifiedGenerationState(t *testing.T) {
	root := t.TempDir()
	authorityPath, trustRootPath, credentialPath, _, authorityDigest :=
		writeServerAuthorityFixture(t, root, time.Now().UTC())
	args := []string{
		"inspect",
		"--authority", authorityPath,
		"--trust-root", trustRootPath,
		"--credential", credentialPath,
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := runAuthorityInspect(
		context.Background(),
		args,
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("inspect authority = %d/%q", code, stderr.String())
	}
	var result authorityInspection
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != "buildopt.server/authority-inspection/v1" ||
		result.AuthorityDigest != authorityDigest ||
		result.Repository.Tenant != "tenant-internal" ||
		result.PolicyID != "internal-policy" ||
		result.PolicyVersion != 7 ||
		result.RevocationEpoch != 7 ||
		result.L1SecurityGeneration != 9 ||
		result.Namespace != "stable" ||
		result.NamespaceGeneration != 12 {
		t.Fatalf("authority inspection = %+v", result)
	}

	tampered := filepath.Join(root, "tampered-authority.json")
	raw, err := os.ReadFile(authorityPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`"namespaceGeneration":12`), []byte(`"namespaceGeneration":13`), 1)
	if err := os.WriteFile(tampered, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	args[2] = tampered
	if code := runAuthorityInspect(
		context.Background(),
		args,
		&stdout,
		&stderr,
	); code != exitConfiguration || stdout.Len() != 0 {
		t.Fatalf("tampered authority = %d/%q", code, stdout.String())
	}
}
