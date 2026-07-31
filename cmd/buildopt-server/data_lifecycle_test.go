package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

func TestRunDataLifecycleDeletesManagedCopiesAndReplays(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deployment-data")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create deployment data root: %v", err)
	}
	marker, err := json.Marshal(map[string]string{
		"deploymentRoot": filepath.Join(filepath.Dir(root), "deployment"),
		"schemaVersion":  "buildopt.dev/deployment-data/v1",
	})
	if err != nil {
		t.Fatalf("encode deployment data marker: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".buildopt-deployment-data.json"),
		append(marker, '\n'),
		0o600,
	); err != nil {
		t.Fatalf("write deployment data marker: %v", err)
	}
	for _, component := range []string{
		"shared",
		"l1",
		"exports",
		"evidence",
		"spool",
	} {
		if err := os.Mkdir(filepath.Join(root, component), 0o700); err != nil {
			t.Fatalf("create component %s: %v", component, err)
		}
	}
	keyPath := filepath.Join(t.TempDir(), "deletion.key")
	key := make([]byte, datalifecycle.RedactionKeyBytes)
	for index := range key {
		key[index] = byte(index + 1)
	}
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		t.Fatalf("write deletion key: %v", err)
	}
	args := []string{
		"delete",
		"--data-root", root,
		"--deletion-id", "delete-cli-001",
		"--tenant", "tenant-sensitive",
		"--repository", "repository-sensitive",
		"--trust-domain", "trust-sensitive",
		"--next-namespace-generation", "9",
		"--next-l1-security-generation", "13",
		"--token-key", keyPath,
		"--token-key-version", "delete-cli-key-v1",
		"--external-destination", "customer-warehouse",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := runDataLifecycle(
		context.Background(),
		args,
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("first deletion = %d/%q", code, stderr.String())
	}
	var first datalifecycle.DeletionReport
	if err := json.Unmarshal(stdout.Bytes(), &first); err != nil {
		t.Fatalf("decode first deletion report: %v", err)
	}
	if first.Replay || first.Tombstone.RemovedComponents != 5 {
		t.Fatalf("first deletion report = %+v", first)
	}
	for _, raw := range []string{
		"tenant-sensitive",
		"repository-sensitive",
		"trust-sensitive",
		"customer-warehouse",
	} {
		if bytes.Contains(stdout.Bytes(), []byte(raw)) {
			t.Fatalf("deletion report contains raw identity %q", raw)
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := runDataLifecycle(
		context.Background(),
		args,
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("replayed deletion = %d/%q", code, stderr.String())
	}
	var replay datalifecycle.DeletionReport
	if err := json.Unmarshal(stdout.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay deletion report: %v", err)
	}
	if !replay.Replay ||
		replay.Tombstone.RequestDigest != first.Tombstone.RequestDigest {
		t.Fatalf("replayed deletion report = %+v", replay)
	}
}

func TestParseExportPolicyRequiresBoundedDiagnosticOptIn(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	policy, err := parseExportPolicy("summary", false, "", now)
	if err != nil || policy.Profile != datalifecycle.ExportSummary {
		t.Fatalf("default export policy = %+v/%v", policy, err)
	}
	until := now.Add(datalifecycle.DiagnosticMaximumLifetime)
	policy, err = parseExportPolicy(
		"diagnostic",
		true,
		until.Format(time.RFC3339),
		now,
	)
	if err != nil ||
		policy.Profile != datalifecycle.ExportDiagnostic ||
		!policy.DiagnosticUntil.Equal(until) {
		t.Fatalf("diagnostic export policy = %+v/%v", policy, err)
	}
	for _, testCase := range []struct {
		authorized bool
		until      string
	}{
		{authorized: false, until: until.Format(time.RFC3339)},
		{authorized: true, until: ""},
		{
			authorized: true,
			until: now.Add(
				datalifecycle.DiagnosticMaximumLifetime + time.Second,
			).Format(time.RFC3339),
		},
	} {
		if _, err := parseExportPolicy(
			"diagnostic",
			testCase.authorized,
			testCase.until,
			now,
		); err == nil {
			t.Fatalf("accepted invalid diagnostic policy %+v", testCase)
		}
	}
}
