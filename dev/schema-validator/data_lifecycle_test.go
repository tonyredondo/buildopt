package schemavalidator

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type dataLifecycleSpec struct {
	SchemaVersion string `json:"schemaVersion"`
	Tokenization  struct {
		Algorithm          string `json:"algorithm"`
		KeyVersionRequired bool   `json:"keyVersionRequired"`
		BeforeBuffering    bool   `json:"beforeBuffering"`
		PlainDigestAllowed bool   `json:"plainDigestAllowed"`
	} `json:"tokenization"`
	Profiles []struct {
		ID                    string `json:"id"`
		ExplicitAuthorization bool   `json:"explicitAuthorization"`
	} `json:"profiles"`
	Classes []struct {
		ID       string `json:"id"`
		TTLHours int    `json:"ttlHours"`
		OptIn    bool   `json:"optIn"`
		Deletion string `json:"deletion"`
	} `json:"classes"`
	Spool struct {
		Bounded            bool   `json:"bounded"`
		Encrypted          bool   `json:"encrypted"`
		DropFirst          string `json:"dropFirst"`
		RetainFinalSummary bool   `json:"retainFinalSummary"`
	} `json:"spool"`
	DeletionCases []struct {
		ID       string `json:"id"`
		Expected string `json:"expected"`
	} `json:"deletionCases"`
}

type rawExportFixture struct {
	SchemaVersion             string `json:"schemaVersion"`
	BuildID                   string `json:"buildId"`
	ActionID                  string `json:"actionId"`
	Outcome                   string `json:"outcome"`
	DurationMS                int    `json:"durationMs"`
	TaskPath                  string `json:"taskPath"`
	Argument                  string `json:"argument"`
	SensitiveEnvironmentValue string `json:"sensitiveEnvironmentValue"`
	SourceContent             string `json:"sourceContent"`
	Fingerprint               string `json:"fingerprint"`
}

type redactedExportFixture struct {
	SchemaVersion         string `json:"schemaVersion"`
	Profile               string `json:"profile"`
	TokenKeyVersion       string `json:"tokenKeyVersion"`
	BuildID               string `json:"buildId"`
	ActionID              string `json:"actionId"`
	Outcome               string `json:"outcome"`
	DurationMS            int    `json:"durationMs"`
	TaskPathToken         string `json:"taskPathToken,omitempty"`
	Fingerprint           string `json:"fingerprint,omitempty"`
	ArgumentToken         string `json:"argumentToken,omitempty"`
	EnvironmentValueToken string `json:"environmentValueToken,omitempty"`
}

type jsonlExportEvent struct {
	EventID        string          `json:"eventId"`
	BuildID        string          `json:"buildId"`
	Sequence       int             `json:"sequence"`
	OccurredAt     string          `json:"occurredAt"`
	EmittedAt      string          `json:"emittedAt"`
	SchemaVersion  string          `json:"schemaVersion"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Profile        string          `json:"profile"`
	Payload        json.RawMessage `json:"payload"`
}

func TestDataLifecycleV1Policy(t *testing.T) {
	t.Parallel()

	spec := loadDataLifecycleSpec(t)
	if spec.SchemaVersion != "buildopt.specs/data-lifecycle/v1" ||
		spec.Tokenization.Algorithm != "HMAC-SHA-256" ||
		!spec.Tokenization.KeyVersionRequired ||
		!spec.Tokenization.BeforeBuffering ||
		spec.Tokenization.PlainDigestAllowed {
		t.Errorf("unsafe lifecycle/redaction identity")
	}
	wantProfiles := []string{"SUMMARY", "TASKS", "EVIDENCE", "DIAGNOSTIC"}
	if len(spec.Profiles) != len(wantProfiles) {
		t.Fatalf("profile count = %d, want %d", len(spec.Profiles), len(wantProfiles))
	}
	for index, profile := range spec.Profiles {
		if index >= len(wantProfiles) || profile.ID != wantProfiles[index] {
			t.Errorf("profile %d = %+v", index, profile)
		}
		if profile.ID == "SUMMARY" && profile.ExplicitAuthorization {
			t.Error("summary unexpectedly requires expanded authorization")
		}
		if profile.ID != "SUMMARY" && !profile.ExplicitAuthorization {
			t.Errorf("%s must require explicit authorization", profile.ID)
		}
	}
	wantTTLs := map[string]int{
		"STABLE_CACHE_BLOB":            720,
		"PENDING":                      24,
		"QUARANTINE_SECURITY_EVIDENCE": 168,
		"OPTIMIZATION_EVIDENCE":        720,
		"SUMMARY_TELEMETRY":            720,
		"DIAGNOSTIC_TELEMETRY":         168,
		"LOCAL_SPOOL_DLQ":              24,
		"SECURITY_AUDIT":               2160,
	}
	if len(spec.Classes) != 10 {
		t.Errorf("data class count = %d, want 10", len(spec.Classes))
	}
	for _, class := range spec.Classes {
		if expected, required := wantTTLs[class.ID]; required && class.TTLHours != expected {
			t.Errorf("%s ttl = %d, want %d", class.ID, class.TTLHours, expected)
		}
		if class.ID == "DIAGNOSTIC_TELEMETRY" && !class.OptIn {
			t.Error("diagnostic telemetry must be opt-in")
		}
	}
	if !spec.Spool.Bounded || !spec.Spool.Encrypted ||
		spec.Spool.DropFirst != "OLDEST_DIAGNOSTIC" ||
		!spec.Spool.RetainFinalSummary {
		t.Errorf("unsafe spool policy: %+v", spec.Spool)
	}
	if len(spec.DeletionCases) != 8 {
		t.Errorf("deletion case count = %d", len(spec.DeletionCases))
	}
}

func TestDataLifecycleV1RedactionFixtures(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	fixtureDir := filepath.Join(root, "fixtures", "data-lifecycle")
	raw := loadStrictJSON[rawExportFixture](
		t,
		filepath.Join(fixtureDir, "raw-input.json"),
	)
	key := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}
	for _, profile := range []string{"SUMMARY", "TASKS", "EVIDENCE", "DIAGNOSTIC"} {
		actual := redactFixture(raw, profile, key)
		expected := loadStrictJSON[redactedExportFixture](
			t,
			filepath.Join(
				fixtureDir,
				"expected-"+strings.ToLower(profile)+".json",
			),
		)
		if actual != expected {
			t.Errorf("%s redaction = %+v, want %+v", profile, actual, expected)
		}
		encoded, err := json.Marshal(actual)
		if err != nil {
			t.Fatalf("encode %s: %v", profile, err)
		}
		for _, sensitive := range []string{
			raw.TaskPath,
			raw.Argument,
			raw.SensitiveEnvironmentValue,
			raw.SourceContent,
		} {
			if bytes.Contains(encoded, []byte(sensitive)) {
				t.Errorf("%s output leaked %q", profile, sensitive)
			}
		}
	}
}

func TestDataLifecycleV1JSONLFixtures(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	validPath := filepath.Join(root, "fixtures", "data-lifecycle", "valid-events.jsonl")
	events, duplicateCount, conflict := readJSONLEvents(t, validPath)
	if conflict || duplicateCount != 1 || len(events) != 3 {
		t.Errorf(
			"valid stream = events %d duplicate %d conflict %t",
			len(events),
			duplicateCount,
			conflict,
		)
	}
	sequences := make([]int, 0, len(events))
	for _, event := range events {
		sequences = append(sequences, event.Sequence)
	}
	slices.Sort(sequences)
	if !slices.Equal(sequences, []int{1, 2, 4}) {
		t.Errorf("sequences = %v", sequences)
	}
	missing := []int{}
	for sequence := 1; sequence <= sequences[len(sequences)-1]; sequence++ {
		if !slices.Contains(sequences, sequence) {
			missing = append(missing, sequence)
		}
	}
	if !slices.Equal(missing, []int{3}) {
		t.Errorf("missing sequences = %v", missing)
	}
	conflictPath := filepath.Join(
		root,
		"fixtures",
		"data-lifecycle",
		"conflicting-duplicate.jsonl",
	)
	_, _, conflict = readJSONLEvents(t, conflictPath)
	if !conflict {
		t.Error("changed duplicate was not rejected")
	}
}

func loadDataLifecycleSpec(t *testing.T) dataLifecycleSpec {
	t.Helper()
	return loadStrictJSON[dataLifecycleSpec](
		t,
		filepath.Join(findRepositoryRoot(t), "specs", "data-lifecycle-v1.json"),
	)
}

func loadStrictJSON[T any](t *testing.T, path string) T {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing data: %v", path, err)
	}
	return value
}

func redactFixture(
	raw rawExportFixture,
	profile string,
	key []byte,
) redactedExportFixture {
	result := redactedExportFixture{
		SchemaVersion:   "buildopt.export/redacted/v1",
		Profile:         profile,
		TokenKeyVersion: "fixture-key-v1",
		BuildID:         raw.BuildID,
		ActionID:        raw.ActionID,
		Outcome:         raw.Outcome,
		DurationMS:      raw.DurationMS,
	}
	if profile == "TASKS" || profile == "EVIDENCE" || profile == "DIAGNOSTIC" {
		result.TaskPathToken = tokenFixtureValue(key, raw.TaskPath)
	}
	if profile == "EVIDENCE" || profile == "DIAGNOSTIC" {
		result.Fingerprint = raw.Fingerprint
	}
	if profile == "DIAGNOSTIC" {
		result.ArgumentToken = tokenFixtureValue(key, raw.Argument)
		result.EnvironmentValueToken = tokenFixtureValue(
			key,
			raw.SensitiveEnvironmentValue,
		)
	}
	return result
}

func tokenFixtureValue(key []byte, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func readJSONLEvents(
	t *testing.T,
	path string,
) ([]jsonlExportEvent, int, bool) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	seen := make(map[string][]byte)
	events := []jsonlExportEvent{}
	duplicates := 0
	conflict := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var event jsonlExportEvent
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if event.EventID == "" || event.BuildID == "" ||
			event.Sequence < 1 || event.OccurredAt == "" ||
			event.EmittedAt == "" || event.SchemaVersion == "" ||
			event.IdempotencyKey == "" || event.Profile == "" ||
			len(event.Payload) == 0 {
			t.Errorf("incomplete JSONL event: %+v", event)
		}
		if previous, exists := seen[event.EventID]; exists {
			if bytes.Equal(previous, line) {
				duplicates++
			} else {
				conflict = true
			}
			continue
		}
		seen[event.EventID] = line
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return events, duplicates, conflict
}

func TestDataLifecycleV1DeletionCases(t *testing.T) {
	t.Parallel()

	spec := loadDataLifecycleSpec(t)
	required := map[string]string{
		"logical-revocation-precedes-physical-delete":          "ACCESS_REVOKED_IMMEDIATELY",
		"managed-copies-cover-blob-metadata-l1-evidence-spool": "ALL_MANAGED_CLASSES_SCHEDULED",
		"l1-deletion-rotates-security-generation":              "GENERATION_ROTATED",
		"tombstone-outlives-referenced-objects":                "ORDER_ENFORCED",
		"downstream-copy-receives-tombstone":                   "EXTERNAL_OBLIGATION_RECORDED",
		"legal-hold-without-consent-is-rejected":               "NO_SILENT_HOLD",
		"diagnostic-default-is-disabled":                       "OPT_IN_REQUIRED",
		"spool-expiry-removes-managed-copy":                    "SPOOL_PURGED",
	}
	for _, testCase := range spec.DeletionCases {
		expected, exists := required[testCase.ID]
		if !exists || expected != testCase.Expected {
			t.Errorf("unexpected deletion case: %+v", testCase)
		}
		delete(required, testCase.ID)
	}
	if len(required) != 0 {
		t.Errorf("missing deletion cases: %v", required)
	}
}
