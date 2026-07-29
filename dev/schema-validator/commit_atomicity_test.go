package schemavalidator

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type commitAtomicityCatalog struct {
	SchemaVersion       string                `json:"schemaVersion"`
	VisibilityAuthority string                `json:"visibilityAuthority"`
	AuditIndex          string                `json:"auditIndex"`
	TransactionMembers  []string              `json:"transactionMembers"`
	Cases               []commitAtomicityCase `json:"cases"`
}

type commitAtomicityCase struct {
	ID           string                  `json:"id"`
	ObjectCount  int                     `json:"objectCount"`
	Precondition string                  `json:"precondition"`
	FaultPoint   string                  `json:"faultPoint"`
	Expected     commitAtomicityExpected `json:"expected"`
}

type commitAtomicityExpected struct {
	Outcome           string `json:"outcome"`
	VisibleObjects    int    `json:"visibleObjects"`
	DecisionPersisted bool   `json:"decisionPersisted"`
	OrphanBlobs       int    `json:"orphanBlobs"`
	QuarantinedBlobs  int    `json:"quarantinedBlobs"`
	ControlIndexed    bool   `json:"controlIndexed"`
	RequiresReconcile bool   `json:"requiresReconcile"`
}

func TestCommitAtomicityV1Policy(t *testing.T) {
	t.Parallel()

	catalog := loadCommitAtomicityCatalog(t)
	if catalog.SchemaVersion != "buildopt.specs/commit-atomicity/v1" {
		t.Errorf("schemaVersion = %q", catalog.SchemaVersion)
	}
	if catalog.VisibilityAuthority != "cache.sqlite" {
		t.Errorf("visibilityAuthority = %q", catalog.VisibilityAuthority)
	}
	if catalog.AuditIndex != "control.sqlite" {
		t.Errorf("auditIndex = %q", catalog.AuditIndex)
	}
	wantMembers := []string{
		"COMMIT_DECISION",
		"ALL_COMMITTED_OBJECT_RECORDS",
	}
	if !slices.Equal(catalog.TransactionMembers, wantMembers) {
		t.Errorf(
			"transactionMembers = %v, want %v",
			catalog.TransactionMembers,
			wantMembers,
		)
	}
}

func TestCommitAtomicityV1Cases(t *testing.T) {
	t.Parallel()

	catalog := loadCommitAtomicityCatalog(t)
	if len(catalog.Cases) < 13 {
		t.Fatalf("case count = %d, want at least 13", len(catalog.Cases))
	}
	seen := make(map[string]struct{}, len(catalog.Cases))
	for _, testCase := range catalog.Cases {
		if testCase.ID == "" || testCase.ObjectCount < 1 {
			t.Fatalf("invalid case identity/count: %+v", testCase)
		}
		if _, duplicate := seen[testCase.ID]; duplicate {
			t.Fatalf("duplicate case ID %q", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
		actual := evaluateCommitAtomicity(testCase)
		if actual != testCase.Expected {
			t.Errorf("%s result = %+v, want %+v", testCase.ID, actual, testCase.Expected)
		}
		if actual.DecisionPersisted &&
			actual.VisibleObjects != testCase.ObjectCount {
			t.Errorf(
				"%s partially visible: %d/%d",
				testCase.ID,
				actual.VisibleObjects,
				testCase.ObjectCount,
			)
		}
	}
}

func loadCommitAtomicityCatalog(t *testing.T) commitAtomicityCatalog {
	t.Helper()
	path := filepath.Join(
		findRepositoryRoot(t),
		"specs",
		"commit-atomicity-v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var catalog commitAtomicityCatalog
	if err := decoder.Decode(&catalog); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing JSON: %v", path, err)
	}
	return catalog
}

func evaluateCommitAtomicity(
	testCase commitAtomicityCase,
) commitAtomicityExpected {
	result := commitAtomicityExpected{}
	switch testCase.Precondition {
	case "EXACT_REPLAY":
		return commitAtomicityExpected{
			Outcome:           "REPLAYED",
			VisibleObjects:    testCase.ObjectCount,
			DecisionPersisted: true,
			ControlIndexed:    true,
		}
	case "CONFLICTING_REPLAY":
		return commitAtomicityExpected{
			Outcome:           "IDEMPOTENCY_CONFLICT",
			VisibleObjects:    testCase.ObjectCount,
			DecisionPersisted: true,
			ControlIndexed:    true,
		}
	case "INCOMPLETE_DECISION", "EXPIRED_DECISION", "REVOKED_DECISION":
		result.Outcome = "ABORTED"
		return result
	case "MISSING_BLOB", "CORRUPT_BLOB":
		result.Outcome = "QUARANTINED_MISS"
		result.QuarantinedBlobs = 1
		result.RequiresReconcile = true
		return result
	case "CAS_LOST":
		result.Outcome = "CAS_LOST_ABORTED"
		result.OrphanBlobs = testCase.ObjectCount
		result.RequiresReconcile = true
		return result
	case "VALID":
	default:
		result.Outcome = "INVALID_PRECONDITION"
		return result
	}

	switch testCase.FaultPoint {
	case "BEFORE_BLOB_DURABLE":
		result.Outcome = "CRASHED_MISS"
		result.RequiresReconcile = true
		return result
	case "AFTER_BLOB_DURABLE":
		result.Outcome = "CRASHED_MISS"
		result.OrphanBlobs = testCase.ObjectCount
		result.RequiresReconcile = true
		return result
	case "BEFORE_CACHE_TX_COMMIT":
		result.Outcome = "ROLLED_BACK_MISS"
		result.OrphanBlobs = testCase.ObjectCount
		result.RequiresReconcile = true
		return result
	case "NONE", "AFTER_CACHE_TX_COMMIT", "CONTROL_INDEX_WRITE":
		result.Outcome = "COMMITTED"
		result.VisibleObjects = testCase.ObjectCount
		result.DecisionPersisted = true
		result.ControlIndexed = testCase.FaultPoint == "NONE"
		result.RequiresReconcile = testCase.FaultPoint != "NONE"
		return result
	default:
		result.Outcome = "INVALID_FAULT_POINT"
		return result
	}
}
