package wcncpobserve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRedactArgumentsAllowlistAndSecrets(t *testing.T) {
	t.Parallel()
	args, err := RedactArguments("example/repository", []string{"build", "--configuration-cache", "--max-workers=8", "--api-token=secret-value", "/home/user/secret", "assemble"})
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "build" || args[1] != "--configuration-cache" || args[5] != "assemble" {
		t.Fatalf("allowlist passthrough = %q", args)
	}
	for _, arg := range args {
		if strings.Contains(arg, "secret-value") || strings.Contains(arg, "/home/user") {
			t.Fatalf("secret leaked in %q", args)
		}
	}
	if !strings.Contains(strings.Join(args, " "), "REDACTED_SECRET") {
		t.Fatalf("secret hint not redacted in %q", args)
	}
}

func TestRedactPathOmitsOutsideRoot(t *testing.T) {
	t.Parallel()
	if got := RedactPath("/repo/checkout", "/repo/checkout/build/output.jar"); got != "build/output.jar" {
		t.Fatalf("relative path = %q", got)
	}
	if got := RedactPath("/repo/checkout", "/home/user/.gradle"); got != "OMITTED" {
		t.Fatalf("outside root = %q", got)
	}
	if got := RedactPath("/repo/checkout", `C:\Users\owner\project\output.jar`); got == "" {
		t.Fatal("windows-shaped path must classify, not crash")
	}
}

func TestObservationFactsRejectRawSecretsAndAbsolutePaths(t *testing.T) {
	t.Parallel()
	facts := ObservationFacts{
		SchemaVersion: ObservationSchemaVersion, RecordType: "WCNCP_OBSERVATION",
		ObservationID: strings.Repeat("b", 64), RepositoryScope: "example/repository", RunnerID: strings.Repeat("a", 64),
		IdempotencyKey: "observation:0001-secure-test", InvocationOrdinal: 1,
		EnvironmentClass: "STANDARD_HOSTED_CI", Arguments: []string{"build", "--token=abc"},
		Bindings: ObservationBindings{
			RepositoryRevision: strings.Repeat("1", 40), SourceTreeSHA256: strings.Repeat("1", 64),
			WrapperSHA256: strings.Repeat("2", 64), GradleVersion: "9.6.1", JDKSHA256: strings.Repeat("3", 64),
			BuildOptPackageSHA256: strings.Repeat("4", 64), WorkflowSHA256: strings.Repeat("5", 64),
			EnvironmentSHA256: strings.Repeat("6", 64), OutputContractSHA256: strings.Repeat("7", 64),
		},
		Duration:           TypedDuration{State: "COMPLETE", ValueMs: int64Pointer(1), Classification: "DIAGNOSTIC_ONLY"},
		ConfigurationCache: "NOT_REQUESTED", BuildCacheMode: "ENABLED",
		OutputManifest: OutputManifest{State: "COMPLETE", SHA256: stringPointer(strings.Repeat("8", 64))},
		Child:          ChildResult{Outcome: "SUCCESS"}, Completeness: "COMPLETE", Authority: ObservationAuthority{},
	}
	if err := facts.Validate(); err == nil {
		t.Fatal("raw secret accepted")
	}
	facts.Arguments = []string{"build", "/abs/path"}
	if err := facts.Validate(); err == nil {
		t.Fatal("absolute path accepted")
	}
}

func int64Pointer(value int64) *int64    { return &value }
func stringPointer(value string) *string { return &value }

func TestDeriveStatusDistinguishesQueuedObservingAndProposals(t *testing.T) {
	t.Parallel()
	if got := DeriveStatus(3, "", false); got.State != "QUEUED" {
		t.Fatalf("offline queued = %+v", got)
	}
	if got := DeriveStatus(0, "", false); got.State != "UNAVAILABLE" {
		t.Fatalf("offline empty = %+v", got)
	}
	if got := DeriveStatus(0, "OBSERVING", true); got.State != "OBSERVING" {
		t.Fatalf("observing = %+v", got)
	}
	if got := DeriveStatus(0, "REVIEW_READY", true); got.State != "REVIEW_READY" {
		t.Fatalf("review = %+v", got)
	}
}

func TestOutboxAtomicBoundedOldestFirst(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "outbox")
	outbox := Outbox{Dir: dir}
	now := time.Now().UTC()
	if err := outbox.Enqueue("obs-0001.json", []byte(`{"one":true}`), now); err != nil {
		t.Fatal(err)
	}
	// Duplicate retry with the same idempotency filename overwrites atomically.
	if err := outbox.Enqueue("obs-0001.json", []byte(`{"one":true}`), now); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "obs-0001.json"))
	if err != nil || string(content) != `{"one":true}` {
		t.Fatalf("outbox content = %q/%v", content, err)
	}
	// Age eviction: an item older than 30 days is removed on next enqueue.
	oldPath := filepath.Join(dir, "obs-old.json")
	if err := os.WriteFile(oldPath, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := now.Add(-31 * 24 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := outbox.Enqueue("obs-0002.json", []byte(`{"two":true}`), now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("aged item was not evicted")
	}
	if err := os.WriteFile(filepath.Join(dir, "runner.id"), []byte(strings.Repeat("a", 64)), 0o600); err != nil {
		t.Fatal(err)
	}
	pending, err := outbox.Pending(32, 1<<20)
	if err != nil || len(pending) != 2 {
		t.Fatalf("pending = %d/%v", len(pending), err)
	}
	if err := outbox.Acknowledge(pending[:1]); err != nil {
		t.Fatal(err)
	}
	remaining, err := outbox.Pending(32, 1<<20)
	if err != nil || len(remaining) != 1 {
		t.Fatalf("remaining = %d/%v", len(remaining), err)
	}
	if _, err := os.Stat(filepath.Join(dir, "runner.id")); err != nil {
		t.Fatalf("runner identity was removed: %v", err)
	}
}
