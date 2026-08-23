package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/nativevolatility"
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

func TestRunLearnsAndAppliesCrossRevisionPortfolio(t *testing.T) {
	root := t.TempDir()
	context := nativevolatility.PortfolioContext{
		RepositoryScopeSHA256: strings.Repeat("1", 64),
		WorkflowSHA256:        strings.Repeat("2", 64),
		WrapperSHA256:         strings.Repeat("3", 64),
		OutputContractSHA256:  strings.Repeat("4", 64),
	}
	contextPath := writeTestJSON(t, root, "context.json", context)
	learning := nativevolatility.Analyze(
		testObservation("a", "1", ":stable", "b", "2", ":generated"),
		testObservation("a", "1", ":stable", "b", "3", ":generated"),
	)
	learningPath := writeTestJSON(t, root, "learning.json", learning)
	var learned bytes.Buffer
	if err := run([]string{
		"--portfolio-context", contextPath,
		"--learn-result", learningPath,
		"--learn-revision", strings.Repeat("a", 64),
	}, &learned); err != nil {
		t.Fatal(err)
	}
	var portfolio nativevolatility.Portfolio
	if err := json.Unmarshal(learned.Bytes(), &portfolio); err != nil {
		t.Fatal(err)
	}
	portfolioPath := writeTestJSON(t, root, "portfolio.json", portfolio)
	current := nativevolatility.Analyze(
		testObservationWithBinding("b", "a", "4", ":stable", "b", "5", ":generated"),
		testObservationWithBinding("b", "a", "4", ":stable", "b", "5", ":generated"),
	)
	currentPath := writeTestJSON(t, root, "current.json", current)
	var applied bytes.Buffer
	if err := run([]string{
		"--portfolio-context", contextPath,
		"--portfolio", portfolioPath,
		"--apply-current", currentPath,
		"--apply-revision", strings.Repeat("c", 64),
	}, &applied); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(applied.String(), nativevolatility.PortfolioApplicationSchema) ||
		!strings.Contains(applied.String(), `"sha256": "`+strings.Repeat("5", 64)+`"`) {
		t.Fatalf("portfolio application = %s", applied.String())
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

func writeTestJSON(t *testing.T, root, name string, value any) string {
	t.Helper()
	path := filepath.Join(root, name)
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func testObservation(values ...string) nativevolatility.Observation {
	return testObservationWithBinding("a", values...)
}

func testObservationWithBinding(bindingSeed string, values ...string) nativevolatility.Observation {
	entries := make([]nativevolatility.Entry, 0, len(values)/3)
	for index := 0; index < len(values); index += 3 {
		entries = append(entries, nativevolatility.Entry{
			Path: values[index], SHA256: strings.Repeat(values[index+1], 64),
			ProducerTasks: []string{values[index+2]},
		})
	}
	return nativevolatility.Observation{
		SchemaVersion: nativevolatility.ObservationSchema,
		BindingSHA256: strings.Repeat(bindingSeed, 64), Entries: entries,
	}
}
