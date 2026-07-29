package sessioningest

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecordValidation(t *testing.T) {
	startedAt := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	record := NewRecord(
		"session-1",
		"generation-1",
		startedAt,
		startedAt.Add(1250*time.Millisecond),
		OutcomeSuccess,
		0,
	)
	if err := record.Validate(); err != nil {
		t.Fatalf("validate record: %v", err)
	}
	if record.DurationMs != 1250 {
		t.Fatalf("duration = %d, want 1250", record.DurationMs)
	}

	testCases := []struct {
		name   string
		mutate func(*Record)
		want   string
	}{
		{
			name: "protocol",
			mutate: func(candidate *Record) {
				candidate.ProtocolVersion = "2.0"
			},
			want: "protocol",
		},
		{
			name: "record type",
			mutate: func(candidate *Record) {
				candidate.RecordType = "BUILD_SESSION"
			},
			want: "record type",
		},
		{
			name: "blank session",
			mutate: func(candidate *Record) {
				candidate.SessionID = ""
			},
			want: "session ID",
		},
		{
			name: "generation whitespace",
			mutate: func(candidate *Record) {
				candidate.GatewayConnectionGeneration = "bad generation"
			},
			want: "generation",
		},
		{
			name: "non UTC timestamp",
			mutate: func(candidate *Record) {
				candidate.StartedAt = "2026-07-29T12:00:00+02:00"
			},
			want: "UTC",
		},
		{
			name: "impossible timestamps",
			mutate: func(candidate *Record) {
				candidate.CompletedAt = "2026-07-29T09:59:59Z"
			},
			want: "precedes",
		},
		{
			name: "negative duration",
			mutate: func(candidate *Record) {
				candidate.DurationMs = -1
			},
			want: "non-negative",
		},
		{
			name: "successful nonzero",
			mutate: func(candidate *Record) {
				candidate.ExitCode = 1
			},
			want: "exitCode 0",
		},
		{
			name: "failed zero",
			mutate: func(candidate *Record) {
				candidate.Outcome = OutcomeBuildFailure
			},
			want: "between 1 and 255",
		},
		{
			name: "unknown outcome",
			mutate: func(candidate *Record) {
				candidate.Outcome = "CANCELLED"
			},
			want: "unsupported session outcome",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := record
			testCase.mutate(&candidate)
			err := candidate.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("validation error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestExportContextFromEnvironment(t *testing.T) {
	context := validTestExportContext()
	raw, err := json.Marshal(context)
	if err != nil {
		t.Fatalf("encode context: %v", err)
	}
	values := map[string]string{ExportContextEnvironment: string(raw)}
	getenv := func(key string) string {
		return values[key]
	}

	parsed, configured, err := ExportContextFromEnvironment(getenv)
	if err != nil || !configured || parsed == nil {
		t.Fatalf("parse context = %+v/%v/%v", parsed, configured, err)
	}
	if parsed.RepositoryID != context.RepositoryID ||
		len(parsed.RequestedTasks) != 1 ||
		parsed.RequestedTasks[0] != "neutralProbe" {
		t.Fatalf("unexpected parsed context: %+v", parsed)
	}

	testCases := []struct {
		name string
		raw  string
	}{
		{
			name: "unknown field",
			raw: strings.TrimSuffix(string(raw), "}") +
				`,"unexpected":true}`,
		},
		{
			name: "trailing JSON",
			raw:  string(raw) + `{}`,
		},
		{
			name: "invalid digest",
			raw: strings.Replace(
				string(raw),
				context.SourceStateDigest,
				"hmac-sha256:not-a-digest",
				1,
			),
		},
		{
			name: "duplicate task",
			raw: strings.Replace(
				string(raw),
				`["neutralProbe"]`,
				`["neutralProbe","neutralProbe"]`,
				1,
			),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			values[ExportContextEnvironment] = testCase.raw
			if context, configured, err := ExportContextFromEnvironment(
				getenv,
			); err == nil || configured || context != nil {
				t.Fatalf(
					"invalid context = %+v/%v/%v",
					context,
					configured,
					err,
				)
			}
		})
	}

	delete(values, ExportContextEnvironment)
	if context, configured, err := ExportContextFromEnvironment(
		getenv,
	); err != nil || configured || context != nil {
		t.Fatalf("absent context = %+v/%v/%v", context, configured, err)
	}
}

func TestRecordValidatesCompleteExportHandoff(t *testing.T) {
	record := validTestRecord()
	record.ExportContext = validTestExportContext()
	record.GradleInvocation = &GradleInvocation{
		ID:            "gradle-invocation-test",
		StartedAt:     "2026-07-29T10:00:00.100Z",
		CompletedAt:   "2026-07-29T10:00:01.900Z",
		DurationMs:    1800,
		PluginVersion: "0.1.0-SNAPSHOT",
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("validate complete export handoff: %v", err)
	}

	record.GradleInvocation.CompletedAt = "2026-07-29T10:00:02.100Z"
	if err := record.Validate(); err == nil ||
		!strings.Contains(err.Error(), "session envelope") {
		t.Fatalf("outside-envelope error = %v", err)
	}
}

func TestStoreIsIdempotentAndConcurrent(t *testing.T) {
	record := validTestRecord()
	record.ExportContext = validTestExportContext()
	record.GradleInvocation = &GradleInvocation{
		ID:            "gradle-invocation-store-test",
		StartedAt:     "2026-07-29T10:00:00.100Z",
		CompletedAt:   "2026-07-29T10:00:01.900Z",
		DurationMs:    1800,
		PluginVersion: "0.1.0-SNAPSHOT",
	}
	store := NewStore()

	const writers = 32
	results := make(chan PutResult, writers)
	errors := make(chan error, writers)
	var waitGroup sync.WaitGroup
	for range writers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			result, err := store.Put(record)
			results <- result
			errors <- err
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errors)

	created := 0
	duplicates := 0
	for result := range results {
		switch result {
		case PutCreated:
			created++
		case PutDuplicate:
			duplicates++
		default:
			t.Errorf("unexpected put result %d", result)
		}
	}
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent put: %v", err)
		}
	}
	if created != 1 || duplicates != writers-1 {
		t.Fatalf(
			"created/duplicates = %d/%d, want 1/%d",
			created,
			duplicates,
			writers-1,
		)
	}

	replayed := cloneRecord(record)
	if _, resultErr := store.Put(replayed); resultErr != nil {
		t.Fatalf("separately allocated replay: %v", resultErr)
	}
	replayed.ExportContext.RequestedTasks[0] = "mutated-after-put"
	if snapshot := store.Snapshot(); snapshot[0].ExportContext.RequestedTasks[0] !=
		"neutralProbe" {
		t.Fatal("store retained mutable input aliases")
	}

	conflicting := cloneRecord(record)
	conflicting.ExitCode = 37
	conflicting.Outcome = OutcomeBuildFailure
	if _, err := store.Put(conflicting); err != ErrSessionConflict {
		t.Fatalf("conflicting put error = %v", err)
	}
	if snapshot := store.Snapshot(); len(snapshot) != 1 ||
		!reflect.DeepEqual(snapshot[0], record) {
		t.Fatalf("unexpected store snapshot: %+v", snapshot)
	}
	snapshot := store.Snapshot()
	snapshot[0].ExportContext.RequestedTasks[0] = "mutated-snapshot"
	if next := store.Snapshot(); next[0].ExportContext.RequestedTasks[0] !=
		"neutralProbe" {
		t.Fatal("snapshot exposed mutable store aliases")
	}
}

func validTestExportContext() *ExportContext {
	return &ExportContext{
		RepositoryID:         "repository-test",
		Revision:             "revision-test",
		RequestedTasks:       []string{"neutralProbe"},
		SourceStateDigest:    "hmac-sha256:" + strings.Repeat("a", 64),
		WorkUnitsFingerprint: "hmac-sha256:" + strings.Repeat("b", 64),
		TokenKeyVersion:      "fixture-token-v1",
		TrustDomain:          "fixture-local",
	}
}

func validTestRecord() Record {
	startedAt := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	return NewRecord(
		"session-test",
		"gateway-generation-test",
		startedAt,
		startedAt.Add(2*time.Second),
		OutcomeSuccess,
		0,
	)
}
