package schemavalidator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestAttemptCommitV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"attempt-commit.v1",
	)
	tests := []struct {
		name               string
		schemaFilename     string
		schemaID           string
		fixtureDirectory   string
		expectedValid      map[string]struct{}
		expectedDiagnostic map[string]string
	}{
		{
			name:             "attempt-state",
			schemaFilename:   "attempt-state.v1.schema.json",
			schemaID:         AttemptStateSchemaID,
			fixtureDirectory: "attempt-state",
			expectedValid: map[string]struct{}{
				"committed.json": {},
			},
			expectedDiagnostic: map[string]string{
				"skipped-policy-bound.json": "/previousState",
			},
		},
		{
			name:             "ci-validation-request",
			schemaFilename:   "ci-validation-request.v1.schema.json",
			schemaID:         CIValidationRequestSchemaID,
			fixtureDirectory: "ci-validation-request",
			expectedValid: map[string]struct{}{
				"full-relevant-validation.json": {},
			},
			expectedDiagnostic: map[string]string{
				"shared-l1.json": "/isolation/separateL1Caches",
			},
		},
		{
			name:             "commit-decision",
			schemaFilename:   "commit-decision.v1.schema.json",
			schemaID:         CommitDecisionSchemaID,
			fixtureDirectory: "commit-decision",
			expectedValid: map[string]struct{}{
				"complete-decision.json": {},
			},
			expectedDiagnostic: map[string]string{
				"inconclusive-verdict.json": "/validation/status",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			schema := compileContractSchema(
				t,
				repositoryRoot,
				test.schemaFilename,
				test.schemaID,
			)
			validFixtures := fixtureFiles(
				t,
				filepath.Join(fixtureRoot, test.fixtureDirectory, "valid"),
			)
			if len(validFixtures) != len(test.expectedValid) {
				t.Fatalf(
					"found %d valid %s fixtures, want %d",
					len(validFixtures),
					test.name,
					len(test.expectedValid),
				)
			}
			for _, fixturePath := range validFixtures {
				if _, ok := test.expectedValid[filepath.Base(fixturePath)]; !ok {
					t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
				}
				if err := schema.Validate(readJSON(t, fixturePath)); err != nil {
					t.Fatalf(
						"%s must validate: %v",
						relativePath(repositoryRoot, fixturePath),
						err,
					)
				}
			}
			assertInvalidFixtures(
				t,
				repositoryRoot,
				schema,
				filepath.Join(fixtureRoot, test.fixtureDirectory, "invalid"),
				test.expectedDiagnostic,
			)
		})
	}
}

func TestAttemptLifecycleV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"attempt-commit.v1",
		"attempt-lifecycle",
	)
	commitSchema := compileContractSchema(
		t,
		repositoryRoot,
		"commit-decision.v1.schema.json",
		CommitDecisionSchemaID,
	)

	expectedValid := map[string]struct{}{
		"abort-before-task.json": {},
		"happy-commit.json":      {},
	}
	validFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	if len(validFixtures) != len(expectedValid) {
		t.Fatalf(
			"found %d valid attempt lifecycle fixtures, want %d",
			len(validFixtures),
			len(expectedValid),
		)
	}
	for _, fixturePath := range validFixtures {
		if _, ok := expectedValid[filepath.Base(fixturePath)]; !ok {
			t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
		}
		vector := readAttemptLifecycle(t, fixturePath)
		if err := validateAttemptLifecycle(fixturePath, vector, commitSchema); err != nil {
			t.Fatalf(
				"%s must be valid: %v",
				relativePath(repositoryRoot, fixturePath),
				err,
			)
		}
	}

	expectedInvalid := map[string]string{
		"incomplete-coverage.json": "coverage",
		"owner-conflict.json":      "owner",
		"skipped-state.json":       "transition",
	}
	invalidFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(invalidFixtures) != len(expectedInvalid) {
		t.Fatalf(
			"found %d invalid attempt lifecycle fixtures, want %d",
			len(invalidFixtures),
			len(expectedInvalid),
		)
	}
	for _, fixturePath := range invalidFixtures {
		wantDiagnostic, ok := expectedInvalid[filepath.Base(fixturePath)]
		if !ok {
			t.Fatalf("missing expected diagnostic for %s", filepath.Base(fixturePath))
		}
		vector := readAttemptLifecycle(t, fixturePath)
		err := validateAttemptLifecycle(fixturePath, vector, commitSchema)
		if err == nil {
			t.Fatalf("%s must be rejected", relativePath(repositoryRoot, fixturePath))
		}
		if !strings.Contains(err.Error(), wantDiagnostic) {
			t.Fatalf(
				"%s failed for an unexpected reason; want %q, got %v",
				relativePath(repositoryRoot, fixturePath),
				wantDiagnostic,
				err,
			)
		}
	}
}

func TestAttemptCommitSemanticBindingsV1(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"attempt-commit.v1",
	)
	request := readJSONObject(
		t,
		filepath.Join(
			fixtureRoot,
			"ci-validation-request",
			"valid",
			"full-relevant-validation.json",
		),
	)
	wantIdempotencyKey := objectString(t, request, "requestId") +
		"/" + objectString(t, request, "actionId")
	if got := objectString(t, request, "idempotencyKey"); got != wantIdempotencyKey {
		t.Fatalf("idempotencyKey = %q, want %q", got, wantIdempotencyKey)
	}
	createdAt := parseObjectTime(t, request, "createdAt")
	deadline := parseObjectTime(t, request, "deadline")
	if !createdAt.Before(deadline) {
		t.Fatal("validation request deadline must follow creation")
	}
	reservation := objectValue(t, request, "budgetReservation")
	if parseObjectTime(t, reservation, "expiresAt").Before(deadline) {
		t.Fatal("budget reservation expires before validation deadline")
	}
	assertDisjointArtifactSets(t, request)

	decision := readJSONObject(
		t,
		filepath.Join(
			fixtureRoot,
			"commit-decision",
			"valid",
			"complete-decision.json",
		),
	)
	if err := validateCommitDecisionSemantics(decision); err != nil {
		t.Fatalf("valid commit decision semantics: %v", err)
	}
}

func TestAttemptCommitValidatorEntrypoints(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	fixtureRoot := filepath.Join(schemaRoot, "testdata", "attempt-commit.v1")
	tests := []struct {
		name     string
		validate func(string, string) error
		schema   string
		fixture  string
	}{
		{
			name:     "attempt-state",
			validate: ValidateAttemptStateV1,
			schema:   "attempt-state.v1.schema.json",
			fixture:  filepath.Join("attempt-state", "valid", "committed.json"),
		},
		{
			name:     "ci-validation-request",
			validate: ValidateCIValidationRequestV1,
			schema:   "ci-validation-request.v1.schema.json",
			fixture: filepath.Join(
				"ci-validation-request",
				"valid",
				"full-relevant-validation.json",
			),
		},
		{
			name:     "commit-decision",
			validate: ValidateCommitDecisionV1,
			schema:   "commit-decision.v1.schema.json",
			fixture: filepath.Join(
				"commit-decision",
				"valid",
				"complete-decision.json",
			),
		},
	}
	for _, test := range tests {
		if err := test.validate(
			filepath.Join(schemaRoot, test.schema),
			filepath.Join(fixtureRoot, test.fixture),
		); err != nil {
			t.Fatalf("%s validator entrypoint: %v", test.name, err)
		}
	}
}

type attemptLifecycle struct {
	AttemptID         string         `json:"attemptId"`
	SourceStateDigest string         `json:"sourceStateDigest"`
	PolicyDigest      string         `json:"policyDigest"`
	OwnerID           string         `json:"ownerId"`
	PendingObjects    []commitObject `json:"pendingObjects"`
	Transitions       []struct {
		Sequence             int    `json:"sequence"`
		StateVersion         int    `json:"stateVersion"`
		ExpectedStateVersion int    `json:"expectedStateVersion"`
		CommandID            string `json:"commandId"`
		OwnerID              string `json:"ownerId"`
		FromState            string `json:"fromState"`
		ToState              string `json:"toState"`
		OccurredAt           string `json:"occurredAt"`
		TaskActionBoundary   string `json:"taskActionBoundary"`
		ValidationStatus     string `json:"validationStatus"`
		CommitDecisionID     string `json:"commitDecisionId"`
		AbortReason          string `json:"abortReason"`
	} `json:"transitions"`
	CommitDecisionFixture string `json:"commitDecisionFixture"`
}

type commitObject struct {
	NamespaceGeneration int64  `json:"namespaceGeneration"`
	Key                 string `json:"key"`
	Checksum            string `json:"checksum"`
	SizeBytes           int64  `json:"sizeBytes"`
}

type commitDecisionSummary struct {
	DecisionID             string         `json:"decisionId"`
	DecisionDigest         string         `json:"decisionDigest"`
	AttemptID              string         `json:"attemptId"`
	SourceStateDigest      string         `json:"sourceStateDigest"`
	Objects                []commitObject `json:"objects"`
	PolicyDigest           string         `json:"policyDigest"`
	ConfigurationPolicyDig string         `json:"configurationPolicyDigest"`
	CacheContractDigest    string         `json:"cacheContractDigest"`
	RevocationEpoch        int64          `json:"revocationEpoch"`
	IssuedAt               string         `json:"issuedAt"`
	ExpiresAt              string         `json:"expiresAt"`
	Validation             struct {
		Status      string `json:"status"`
		CompletedAt string `json:"completedAt"`
		ExpiresAt   string `json:"expiresAt"`
	} `json:"validation"`
}

func readAttemptLifecycle(t *testing.T, path string) attemptLifecycle {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}()
	var vector attemptLifecycle
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&vector); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return vector
}

func validateAttemptLifecycle(
	fixturePath string,
	vector attemptLifecycle,
	commitSchema *jsonschema.Schema,
) error {
	if len(vector.Transitions) == 0 {
		return errors.New("attempt has no transitions")
	}
	allowed := map[string]map[string]struct{}{
		"CREATED": {
			"POLICY_BOUND": {},
			"ABORTED":      {},
		},
		"POLICY_BOUND": {
			"GRADLE_STARTED": {},
			"ABORTED":        {},
		},
		"GRADLE_STARTED": {
			"TASK_ACTION_STARTED": {},
			"ABORTED":             {},
		},
		"TASK_ACTION_STARTED": {
			"VALIDATED": {},
			"ABORTED":   {},
		},
		"VALIDATED": {
			"COMMITTED": {},
			"ABORTED":   {},
		},
	}
	commands := make(map[string]struct{}, len(vector.Transitions))
	var priorTime time.Time
	boundaryUnknown := false
	for index, transition := range vector.Transitions {
		wantVersion := index + 1
		if transition.Sequence != wantVersion ||
			transition.StateVersion != wantVersion {
			return fmt.Errorf(
				"sequence/state version at transition %d is not contiguous",
				index,
			)
		}
		if _, duplicate := commands[transition.CommandID]; duplicate {
			return fmt.Errorf("command %q is not idempotently unique", transition.CommandID)
		}
		commands[transition.CommandID] = struct{}{}
		occurredAt, err := time.Parse(time.RFC3339, transition.OccurredAt)
		if err != nil {
			return fmt.Errorf("transition %d timestamp: %w", index, err)
		}
		if index > 0 && !priorTime.Before(occurredAt) {
			return fmt.Errorf("transition %d timestamp is not ordered", index)
		}
		priorTime = occurredAt
		if transition.TaskActionBoundary == "UNKNOWN" {
			boundaryUnknown = true
		}

		ownerID := transition.OwnerID
		if ownerID == "" {
			ownerID = vector.OwnerID
		}
		if ownerID != vector.OwnerID {
			return fmt.Errorf("owner changed to %q", ownerID)
		}

		if index == 0 {
			if transition.ToState != "CREATED" ||
				transition.FromState != "" ||
				transition.ExpectedStateVersion != 0 {
				return errors.New("first transition must create version 1")
			}
			continue
		}
		previous := vector.Transitions[index-1]
		if transition.ExpectedStateVersion != previous.StateVersion {
			return fmt.Errorf(
				"transition %d CAS expected version %d, want %d",
				index,
				transition.ExpectedStateVersion,
				previous.StateVersion,
			)
		}
		if transition.FromState != previous.ToState {
			return fmt.Errorf(
				"transition %d from %s does not match %s",
				index,
				transition.FromState,
				previous.ToState,
			)
		}
		if _, ok := allowed[transition.FromState][transition.ToState]; !ok {
			return fmt.Errorf(
				"transition %s -> %s is not allowed",
				transition.FromState,
				transition.ToState,
			)
		}
		if previous.ToState == "COMMITTED" || previous.ToState == "ABORTED" {
			return errors.New("transition follows a terminal state")
		}
	}

	final := vector.Transitions[len(vector.Transitions)-1]
	switch final.ToState {
	case "ABORTED":
		if final.AbortReason == "" {
			return errors.New("aborted attempt has no reason")
		}
		if vector.CommitDecisionFixture != "" {
			return errors.New("aborted attempt references a commit decision")
		}
		return nil
	case "COMMITTED":
	default:
		return fmt.Errorf("attempt does not end in a terminal state: %s", final.ToState)
	}

	if final.TaskActionBoundary != "CONFIRMED" || boundaryUnknown {
		return errors.New("commit has unknown task-action boundary")
	}
	if len(vector.Transitions) < 2 ||
		vector.Transitions[len(vector.Transitions)-2].ValidationStatus != "PASSED" {
		return errors.New("commit lacks a passed validation transition")
	}
	if vector.CommitDecisionFixture == "" {
		return errors.New("commit lacks decision fixture")
	}
	decisionPath := filepath.Clean(
		filepath.Join(filepath.Dir(fixturePath), vector.CommitDecisionFixture),
	)
	instance, err := readJSONFile(decisionPath)
	if err != nil {
		return err
	}
	if err := commitSchema.Validate(instance); err != nil {
		return fmt.Errorf("commit decision schema: %w", err)
	}
	var decision commitDecisionSummary
	encoded, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("encode commit decision: %w", err)
	}
	if err := json.Unmarshal(encoded, &decision); err != nil {
		return fmt.Errorf("decode commit decision: %w", err)
	}
	if decision.AttemptID != vector.AttemptID ||
		decision.SourceStateDigest != vector.SourceStateDigest ||
		decision.PolicyDigest != vector.PolicyDigest {
		return errors.New("commit decision binding differs from attempt")
	}
	if decision.DecisionID != final.CommitDecisionID {
		return errors.New("commit decision identity differs from final transition")
	}
	if !sameCommitObjects(vector.PendingObjects, decision.Objects) {
		return errors.New("commit decision object coverage is incomplete")
	}
	decisionIssuedAt, err := time.Parse(time.RFC3339, decision.IssuedAt)
	if err != nil {
		return fmt.Errorf("commit decision issuedAt: %w", err)
	}
	decisionExpiresAt, err := time.Parse(time.RFC3339, decision.ExpiresAt)
	if err != nil {
		return fmt.Errorf("commit decision expiresAt: %w", err)
	}
	if decisionIssuedAt.After(priorTime) || !priorTime.Before(decisionExpiresAt) {
		return errors.New("commit transition is outside decision lifetime")
	}
	return validateCommitDecisionSummarySemantics(decision)
}

func readJSONFile(path string) (any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	var value any
	if err := json.NewDecoder(file).Decode(&value); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

func sameCommitObjects(left []commitObject, right []commitObject) bool {
	leftKeys := commitObjectKeys(left)
	rightKeys := commitObjectKeys(right)
	return reflect.DeepEqual(leftKeys, rightKeys)
}

func commitObjectKeys(objects []commitObject) []string {
	keys := make([]string, 0, len(objects))
	for _, object := range objects {
		keys = append(
			keys,
			fmt.Sprintf(
				"%d/%s/%s/%d",
				object.NamespaceGeneration,
				object.Key,
				object.Checksum,
				object.SizeBytes,
			),
		)
	}
	sort.Strings(keys)
	return keys
}

func validateCommitDecisionSemantics(decision map[string]any) error {
	encoded, err := json.Marshal(decision)
	if err != nil {
		return err
	}
	var summary commitDecisionSummary
	if err := json.Unmarshal(encoded, &summary); err != nil {
		return err
	}
	return validateCommitDecisionSummarySemantics(summary)
}

func validateCommitDecisionSummarySemantics(decision commitDecisionSummary) error {
	issuedAt, err := time.Parse(time.RFC3339, decision.IssuedAt)
	if err != nil {
		return fmt.Errorf("issuedAt: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, decision.ExpiresAt)
	if err != nil {
		return fmt.Errorf("expiresAt: %w", err)
	}
	if !issuedAt.Before(expiresAt) {
		return errors.New("commit decision expires before issuance")
	}
	if decision.Validation.Status == "PASSED" {
		completedAt, err := time.Parse(
			time.RFC3339,
			decision.Validation.CompletedAt,
		)
		if err != nil {
			return fmt.Errorf("validation completedAt: %w", err)
		}
		validationExpiresAt, err := time.Parse(
			time.RFC3339,
			decision.Validation.ExpiresAt,
		)
		if err != nil {
			return fmt.Errorf("validation expiresAt: %w", err)
		}
		if issuedAt.Before(completedAt) ||
			validationExpiresAt.Before(expiresAt) {
			return errors.New("validation does not cover commit decision lifetime")
		}
	}
	keys := commitObjectIdentities(decision.Objects)
	for index := 1; index < len(keys); index++ {
		if keys[index] == keys[index-1] {
			return errors.New("commit decision contains duplicate objects")
		}
	}
	return nil
}

func commitObjectIdentities(objects []commitObject) []string {
	keys := make([]string, 0, len(objects))
	for _, object := range objects {
		keys = append(
			keys,
			fmt.Sprintf("%d/%s", object.NamespaceGeneration, object.Key),
		)
	}
	sort.Strings(keys)
	return keys
}

func assertDisjointArtifactSets(t *testing.T, request map[string]any) {
	t.Helper()

	candidate := objectArray(t, request, "candidateArtifactRefs")
	control := objectArray(t, request, "controlArtifactRefs")
	candidateDigests := make(map[string]struct{}, len(candidate))
	for _, artifact := range candidate {
		candidateDigests[objectString(t, artifact, "digest")] = struct{}{}
	}
	for _, artifact := range control {
		digest := objectString(t, artifact, "digest")
		if _, exists := candidateDigests[digest]; exists {
			t.Fatalf("candidate/control share artifact digest %s", digest)
		}
	}
}
