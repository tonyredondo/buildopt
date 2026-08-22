package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunObserveMaterialization(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "module", "build", "value.bin")
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		t.Fatal(err)
	}
	raw := []byte("observed-output\n")
	if err := os.WriteFile(output, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "manifest.json")
	manifestRaw := `{
  "schemaVersion":"buildopt.poc/verified-output-materialization/v2",
  "repositoryId":"example/repository",
  "targetRevision":"0123456789abcdef0123456789abcdef01234567",
  "requiredOutputs":["module/build/**"],
  "candidateOutputs":["changed/build/**"],
  "packFile":".buildopt/pack",
  "packSha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "packSize":16,
  "entries":[{
    "path":"module/build/value.bin",
    "sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "size":16,"mode":384,"offset":0,
    "producerTasks":[":module:produce"]
  }]
}`
	if err := os.WriteFile(manifest, []byte(manifestRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	binding := strings.Repeat("c", 64)
	if err := run([]string{
		"--observe-root", root, "--materialization-manifest", manifest,
		"--binding", binding,
	}, &stdout); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if !strings.Contains(stdout.String(), hex.EncodeToString(digest[:])) ||
		!strings.Contains(stdout.String(), `":module:produce"`) ||
		!strings.Contains(stdout.String(), binding) {
		t.Fatalf("observation = %s", stdout.String())
	}
}

func TestRunObserveMaterializationRequiresProducerAttribution(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	manifestRaw := `{
  "schemaVersion":"buildopt.poc/verified-output-materialization/v2",
  "repositoryId":"example/repository",
  "targetRevision":"0123456789abcdef0123456789abcdef01234567",
  "requiredOutputs":[],"candidateOutputs":[],"packFile":"pack",
  "packSha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "packSize":0,
  "entries":[{"path":"missing.bin","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":0,"mode":384,"offset":0}]
}`
	if err := os.WriteFile(manifest, []byte(manifestRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"--observe-root", root, "--materialization-manifest", manifest,
		"--binding", strings.Repeat("c", 64),
	}, &bytes.Buffer{}); err == nil {
		t.Fatal("manifest without producer attribution was accepted")
	}
}
