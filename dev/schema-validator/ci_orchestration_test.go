package schemavalidator

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type ciOrchestrationCatalog struct {
	SchemaVersion string `json:"schemaVersion"`
	GitHub        struct {
		ScheduleMinutes                       int    `json:"scheduleMinutes"`
		WorkflowDispatch                      bool   `json:"workflowDispatch"`
		TrustedRevisionsOnly                  bool   `json:"trustedRevisionsOnly"`
		MaxLeasesPerRun                       int    `json:"maxLeasesPerRun"`
		MaxConcurrentValidationsPerRepository int    `json:"maxConcurrentValidationsPerRepository"`
		ConcurrencyGroup                      string `json:"concurrencyGroup"`
		CancelInProgress                      bool   `json:"cancelInProgress"`
		PullRequestTargetAllowed              bool   `json:"pullRequestTargetAllowed"`
	} `json:"github"`
	Budget struct {
		RollingSevenDayPercent       int  `json:"rollingSevenDayPercent"`
		RollingTwentyFourHourPercent int  `json:"rollingTwentyFourHourPercent"`
		BorrowingAllowed             bool `json:"borrowingAllowed"`
	} `json:"budget"`
	IsolationKeys []string              `json:"isolationKeys"`
	AttemptStates []string              `json:"attemptStates"`
	Cases         []ciOrchestrationCase `json:"cases"`
}

type ciOrchestrationCase struct {
	ID                            string   `json:"id"`
	JobKind                       string   `json:"jobKind"`
	TrustedRevision               bool     `json:"trustedRevision"`
	Arms                          []string `json:"arms"`
	NaturalRunnerMs24h            int64    `json:"naturalRunnerMs24h"`
	NaturalRunnerMs7d             int64    `json:"naturalRunnerMs7d"`
	UsedRunnerMs24h               int64    `json:"usedRunnerMs24h"`
	UsedRunnerMs7d                int64    `json:"usedRunnerMs7d"`
	RequestedRunnerMs             int64    `json:"requestedRunnerMs"`
	ActiveValidations             int      `json:"activeValidations"`
	Isolated                      bool     `json:"isolated"`
	TerminalEvent                 string   `json:"terminalEvent"`
	ExpectedDecision              string   `json:"expectedDecision"`
	ExpectedReservation           string   `json:"expectedReservation"`
	ExpectedAttempt               string   `json:"expectedAttempt"`
	NormalJobRemainsAuthoritative bool     `json:"normalJobRemainsAuthoritative"`
}

type ciOrchestrationResult struct {
	Decision                      string
	Reservation                   string
	Attempt                       string
	NormalJobRemainsAuthoritative bool
}

func TestCIOrchestrationV1Policy(t *testing.T) {
	t.Parallel()

	catalog := loadCIOrchestrationCatalog(t)
	if catalog.SchemaVersion != "buildopt.specs/ci-orchestration/v1" {
		t.Errorf("schemaVersion = %q", catalog.SchemaVersion)
	}
	if catalog.GitHub.ScheduleMinutes != 15 ||
		!catalog.GitHub.WorkflowDispatch ||
		!catalog.GitHub.TrustedRevisionsOnly ||
		catalog.GitHub.MaxLeasesPerRun != 1 ||
		catalog.GitHub.MaxConcurrentValidationsPerRepository != 1 ||
		catalog.GitHub.ConcurrencyGroup != "buildopt-validation-${repository}" ||
		catalog.GitHub.CancelInProgress ||
		catalog.GitHub.PullRequestTargetAllowed {
		t.Errorf("unsafe GitHub binding: %+v", catalog.GitHub)
	}
	if catalog.Budget.RollingSevenDayPercent != 5 ||
		catalog.Budget.RollingTwentyFourHourPercent != 10 ||
		catalog.Budget.BorrowingAllowed {
		t.Errorf("unsafe budget: %+v", catalog.Budget)
	}
	wantIsolation := []string{
		"worktree",
		"writableOutputs",
		"gradleUserHome",
		"configurationCache",
		"daemon",
		"l1Cache",
		"writeCredential",
	}
	if !slices.Equal(catalog.IsolationKeys, wantIsolation) {
		t.Errorf("isolationKeys = %v, want %v", catalog.IsolationKeys, wantIsolation)
	}
	assertStateMachineSchemaEnum(
		t,
		filepath.Join(
			findRepositoryRoot(t),
			"contracts",
			"jsonschema",
			"attempt-state.v1.schema.json",
		),
		"state",
		catalog.AttemptStates,
	)
}

func TestCIOrchestrationV1Cases(t *testing.T) {
	t.Parallel()

	catalog := loadCIOrchestrationCatalog(t)
	if len(catalog.Cases) < 12 {
		t.Fatalf("case count = %d, want at least 12", len(catalog.Cases))
	}
	seen := make(map[string]struct{}, len(catalog.Cases))
	for _, testCase := range catalog.Cases {
		testCase := testCase
		t.Run(testCase.ID, func(t *testing.T) {
			if testCase.ID == "" {
				t.Fatal("case ID is empty")
			}
			if _, duplicate := seen[testCase.ID]; duplicate {
				t.Fatalf("duplicate case ID %q", testCase.ID)
			}
			seen[testCase.ID] = struct{}{}
			actual := evaluateCIOrchestration(catalog, testCase)
			expected := ciOrchestrationResult{
				Decision:                      testCase.ExpectedDecision,
				Reservation:                   testCase.ExpectedReservation,
				Attempt:                       testCase.ExpectedAttempt,
				NormalJobRemainsAuthoritative: testCase.NormalJobRemainsAuthoritative,
			}
			if actual != expected {
				t.Fatalf("result = %+v, want %+v", actual, expected)
			}
		})
	}
}

func TestCIOrchestrationV1GitHubFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join(
		findRepositoryRoot(t),
		"fixtures",
		"ci-orchestration",
		"github-validation.yml",
	)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(content)
	required := []string{
		"cron: \"*/15 * * * *\"",
		"workflow_dispatch:",
		"contents: read",
		"group: buildopt-validation-${{ github.repository }}",
		"cancel-in-progress: false",
		"github.event.repository.default_branch",
		"persist-credentials: false",
		"--max-count 1",
		"--sequential-randomized",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Errorf("GitHub fixture is missing %q", fragment)
		}
	}
	forbidden := []string{
		"pull_request_target",
		"contents: write",
		"pull-requests: write",
		"cancel-in-progress: true",
	}
	for _, fragment := range forbidden {
		if strings.Contains(text, fragment) {
			t.Errorf("GitHub fixture contains forbidden %q", fragment)
		}
	}
}

func loadCIOrchestrationCatalog(t *testing.T) ciOrchestrationCatalog {
	t.Helper()
	path := filepath.Join(
		findRepositoryRoot(t),
		"specs",
		"ci-orchestration-v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var catalog ciOrchestrationCatalog
	if err := decoder.Decode(&catalog); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing JSON: %v", path, err)
	}
	return catalog
}

func evaluateCIOrchestration(
	catalog ciOrchestrationCatalog,
	testCase ciOrchestrationCase,
) ciOrchestrationResult {
	result := ciOrchestrationResult{
		NormalJobRemainsAuthoritative: true,
	}
	if testCase.JobKind == "NORMAL" {
		result.Reservation = "NOT_REQUIRED"
		result.Attempt = "NOT_CREATED"
		if len(testCase.Arms) != 1 {
			result.Decision = "REJECT_MULTIPLE_AUTHORITATIVE_ARMS"
			return result
		}
		result.Decision = "RUN_AUTHORITATIVE"
		return result
	}
	if testCase.JobKind != "VALIDATION" {
		result.Decision = "REJECT_UNKNOWN_JOB"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "NOT_CREATED"
		return result
	}
	if !testCase.TrustedRevision {
		result.Decision = "REJECT_UNTRUSTED_REVISION"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "NOT_CREATED"
		return result
	}
	if len(testCase.Arms) != 2 ||
		!slices.Contains(testCase.Arms, "CANDIDATE") ||
		!slices.Contains(testCase.Arms, "CONTROL") {
		result.Decision = "REJECT_ARM_PAIR"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "ABORTED"
		return result
	}
	if testCase.ActiveValidations >=
		catalog.GitHub.MaxConcurrentValidationsPerRepository {
		result.Decision = "REJECT_CONCURRENCY"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "QUEUED"
		return result
	}
	if !testCase.Isolated {
		result.Decision = "REJECT_ISOLATION"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "ABORTED"
		return result
	}
	if exceedsPercent(
		testCase.UsedRunnerMs24h+testCase.RequestedRunnerMs,
		testCase.NaturalRunnerMs24h,
		catalog.Budget.RollingTwentyFourHourPercent,
	) {
		result.Decision = "REJECT_DAILY_BUDGET"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "INCONCLUSIVE"
		return result
	}
	if exceedsPercent(
		testCase.UsedRunnerMs7d+testCase.RequestedRunnerMs,
		testCase.NaturalRunnerMs7d,
		catalog.Budget.RollingSevenDayPercent,
	) {
		result.Decision = "REJECT_WEEKLY_BUDGET"
		result.Reservation = "NOT_RESERVED"
		result.Attempt = "INCONCLUSIVE"
		return result
	}
	result.Decision = "LEASE"
	result.Reservation = "RELEASE_UNUSED"
	switch testCase.TerminalEvent {
	case "COMPLETE":
		result.Attempt = "COMMITTED"
	case "OWNER_DEAD":
		result.Attempt = "ABORTED_RECONCILED"
	case "CANCEL", "TIMEOUT", "EXPIRE", "INFRA_FAILURE",
		"UNKNOWN_TASK_BOUNDARY":
		result.Attempt = "ABORTED"
	default:
		result.Attempt = "INCONCLUSIVE"
	}
	return result
}

func exceedsPercent(used int64, natural int64, percent int) bool {
	if natural <= 0 || used < 0 {
		return true
	}
	return used*100 > natural*int64(percent)
}
