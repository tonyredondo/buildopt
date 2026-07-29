package schemavalidator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestExperimentResultV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schema := compileContractSchema(
		t,
		repositoryRoot,
		"experiment-result.v1.schema.json",
		ExperimentResultSchemaID,
	)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"experiment-result.v1",
	)

	expectedValid := map[string]struct{}{
		"final-inconclusive.json":   {},
		"final-qualified.json":      {},
		"invalidated-result.json":   {},
		"preliminary-learning.json": {},
	}
	validFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	if len(validFixtures) != len(expectedValid) {
		t.Fatalf(
			"found %d valid EXPERIMENT_RESULT fixtures, want %d",
			len(validFixtures),
			len(expectedValid),
		)
	}
	for _, fixturePath := range validFixtures {
		fixturePath := fixturePath
		t.Run("valid/"+filepath.Base(fixturePath), func(t *testing.T) {
			if _, ok := expectedValid[filepath.Base(fixturePath)]; !ok {
				t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
			}
			if err := schema.Validate(readJSON(t, fixturePath)); err != nil {
				t.Fatalf("%s must validate: %v", relativePath(repositoryRoot, fixturePath), err)
			}
			result := readExperimentSummary(t, fixturePath)
			if err := validateExperimentSemantics(result); err != nil {
				t.Fatalf("%s has invalid lifecycle semantics: %v", relativePath(repositoryRoot, fixturePath), err)
			}
		})
	}

	expectedInvalidDiagnostics := map[string]string{
		"action-scope-without-action.json":     "actionIds",
		"final-without-gates.json":             "gateEvaluation",
		"invalidated-republishes-effects.json": "effects",
		"preliminary-promotes.json":            "/decision/state",
	}
	assertInvalidFixtures(
		t,
		repositoryRoot,
		schema,
		filepath.Join(fixtureRoot, "invalid"),
		expectedInvalidDiagnostics,
	)
}

func TestActionRecordV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schema := compileContractSchema(
		t,
		repositoryRoot,
		"action-record.v1.schema.json",
		ActionRecordSchemaID,
	)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"action-record.v1",
	)

	expectedValid := map[string]struct{}{
		"activate-in-ci-claimed-final.json": {},
		"activate-in-ci-inconclusive.json":  {},
		"activate-in-ci-stale-result.json":  {},
		"activate-in-ci.json":               {},
		"begin-shadow.json":                 {},
		"rollback-invalidated.json":         {},
	}
	validFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	if len(validFixtures) != len(expectedValid) {
		t.Fatalf(
			"found %d valid ACTION_RECORD fixtures, want %d",
			len(validFixtures),
			len(expectedValid),
		)
	}
	for _, fixturePath := range validFixtures {
		fixturePath := fixturePath
		t.Run("valid/"+filepath.Base(fixturePath), func(t *testing.T) {
			if _, ok := expectedValid[filepath.Base(fixturePath)]; !ok {
				t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
			}
			if err := schema.Validate(readJSON(t, fixturePath)); err != nil {
				t.Fatalf("%s must validate: %v", relativePath(repositoryRoot, fixturePath), err)
			}
			action := readActionSummary(t, fixturePath)
			if err := validateActionSemantics(action); err != nil {
				t.Fatalf("%s has invalid transition semantics: %v", relativePath(repositoryRoot, fixturePath), err)
			}
		})
	}

	expectedInvalidDiagnostics := map[string]string{
		"activate-with-preliminary-reference.json": "/authorization/experimentResultRef/status",
		"empty-evidence.json":                      "/evidenceRefs",
		"embeds-observed-effect.json":              "observedNetBuildTimeSavedMs",
		"wrong-transition-state.json":              "/fromState",
	}
	assertInvalidFixtures(
		t,
		repositoryRoot,
		schema,
		filepath.Join(fixtureRoot, "invalid"),
		expectedInvalidDiagnostics,
	)
}

func TestExperimentActionLifecycleV1(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	experimentSchema := compileContractSchema(
		t,
		repositoryRoot,
		"experiment-result.v1.schema.json",
		ExperimentResultSchemaID,
	)
	actionSchema := compileContractSchema(
		t,
		repositoryRoot,
		"action-record.v1.schema.json",
		ActionRecordSchemaID,
	)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"experiment-action-lifecycle.v1",
	)

	expectedValid := map[string]struct{}{
		"final-promotes-action.json":         {},
		"invalidated-rolls-back-action.json": {},
		"policy-begins-shadow.json":          {},
	}
	validFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	if len(validFixtures) != len(expectedValid) {
		t.Fatalf(
			"found %d valid lifecycle fixtures, want %d",
			len(validFixtures),
			len(expectedValid),
		)
	}
	for _, fixturePath := range validFixtures {
		fixturePath := fixturePath
		t.Run("valid/"+filepath.Base(fixturePath), func(t *testing.T) {
			if _, ok := expectedValid[filepath.Base(fixturePath)]; !ok {
				t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
			}
			vector := readLifecycleVector(t, fixturePath)
			result, action := validateVectorDocuments(
				t,
				repositoryRoot,
				experimentSchema,
				actionSchema,
				fixturePath,
				vector,
			)
			if err := validateLinkedLifecycle(result, action); err != nil {
				t.Fatalf("%s must be linked: %v", relativePath(repositoryRoot, fixturePath), err)
			}
		})
	}

	expectedInvalidDiagnostics := map[string]string{
		"inconclusive-activation.json":   "decision",
		"preliminary-claimed-final.json": "status",
		"stale-result-version.json":      "resultVersion",
	}
	invalidFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(invalidFixtures) != len(expectedInvalidDiagnostics) {
		t.Fatalf(
			"found %d invalid lifecycle fixtures, want %d",
			len(invalidFixtures),
			len(expectedInvalidDiagnostics),
		)
	}
	for _, fixturePath := range invalidFixtures {
		fixturePath := fixturePath
		t.Run("invalid/"+filepath.Base(fixturePath), func(t *testing.T) {
			expectedDiagnostic, ok := expectedInvalidDiagnostics[filepath.Base(fixturePath)]
			if !ok {
				t.Fatalf("missing expected lifecycle diagnostic for %s", filepath.Base(fixturePath))
			}
			vector := readLifecycleVector(t, fixturePath)
			result, action := validateVectorDocuments(
				t,
				repositoryRoot,
				experimentSchema,
				actionSchema,
				fixturePath,
				vector,
			)
			err := validateLinkedLifecycle(result, action)
			if err == nil {
				t.Fatalf("%s must be rejected", relativePath(repositoryRoot, fixturePath))
			}
			if !strings.Contains(err.Error(), expectedDiagnostic) {
				t.Fatalf(
					"%s failed for an unexpected reason; want %q, got: %v",
					relativePath(repositoryRoot, fixturePath),
					expectedDiagnostic,
					err,
				)
			}
		})
	}
}

func TestExperimentAndActionValidatorEntrypoints(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	testdataRoot := filepath.Join(schemaRoot, "testdata")

	if err := ValidateExperimentResultV1(
		filepath.Join(schemaRoot, "experiment-result.v1.schema.json"),
		filepath.Join(
			testdataRoot,
			"experiment-result.v1",
			"valid",
			"final-qualified.json",
		),
	); err != nil {
		t.Fatalf("ValidateExperimentResultV1: %v", err)
	}
	if err := ValidateActionRecordV1(
		filepath.Join(schemaRoot, "action-record.v1.schema.json"),
		filepath.Join(
			testdataRoot,
			"action-record.v1",
			"valid",
			"activate-in-ci.json",
		),
	); err != nil {
		t.Fatalf("ValidateActionRecordV1: %v", err)
	}
}

func compileContractSchema(
	t *testing.T,
	repositoryRoot string,
	filename string,
	expectedID string,
) *jsonschema.Schema {
	t.Helper()

	schemaPath := filepath.Join(repositoryRoot, "contracts", "jsonschema", filename)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	if schema.ID != expectedID {
		t.Fatalf("compiled schema ID = %q, want %q", schema.ID, expectedID)
	}
	if schema.DraftVersion != 2020 {
		t.Fatalf("compiled schema draft = %d, want 2020-12", schema.DraftVersion)
	}
	return schema
}

func assertInvalidFixtures(
	t *testing.T,
	repositoryRoot string,
	schema *jsonschema.Schema,
	fixtureDirectory string,
	expectedDiagnostics map[string]string,
) {
	t.Helper()

	fixtures := fixtureFiles(t, fixtureDirectory)
	if len(fixtures) != len(expectedDiagnostics) {
		t.Fatalf(
			"found %d invalid fixtures in %s, want %d",
			len(fixtures),
			relativePath(repositoryRoot, fixtureDirectory),
			len(expectedDiagnostics),
		)
	}
	for _, fixturePath := range fixtures {
		fixturePath := fixturePath
		t.Run("invalid/"+filepath.Base(fixturePath), func(t *testing.T) {
			expectedDiagnostic, ok := expectedDiagnostics[filepath.Base(fixturePath)]
			if !ok {
				t.Fatalf("missing expected diagnostic for %s", filepath.Base(fixturePath))
			}
			err := schema.Validate(readJSON(t, fixturePath))
			if err == nil {
				t.Fatalf("%s must be rejected", relativePath(repositoryRoot, fixturePath))
			}
			if !strings.Contains(err.Error(), expectedDiagnostic) {
				t.Fatalf(
					"%s failed for an unexpected reason; want %q, got: %v",
					relativePath(repositoryRoot, fixturePath),
					expectedDiagnostic,
					err,
				)
			}
		})
	}
}

type experimentSummary struct {
	ExperimentID             string   `json:"experimentId"`
	ResultVersion            int      `json:"resultVersion"`
	SupersedesResultVersion  int      `json:"supersedesResultVersion"`
	Status                   string   `json:"status"`
	AsOf                     string   `json:"asOf"`
	EffectScope              string   `json:"effectScope"`
	ActionIDs                []string `json:"actionIds"`
	MeasurementPolicyVersion string   `json:"measurementPolicyVersion"`
	Window                   struct {
		StartedAt string `json:"startedAt"`
		EndedAt   string `json:"endedAt"`
	} `json:"window"`
	Samples struct {
		Assigned struct {
			Candidate int `json:"candidate"`
			Control   int `json:"control"`
		} `json:"assigned"`
		Analyzed struct {
			Candidate int `json:"candidate"`
			Control   int `json:"control"`
		} `json:"analyzed"`
		Outcomes struct {
			Candidate outcomeCounts `json:"candidate"`
			Control   outcomeCounts `json:"control"`
		} `json:"outcomes"`
		ExcludedSampleSize int `json:"excludedSampleSize"`
		Exclusions         []struct {
			Count int `json:"count"`
		} `json:"exclusions"`
	} `json:"samples"`
	Effects *struct {
		ObservedNetBuildTimeSavedMs            int       `json:"observedNetBuildTimeSavedMs"`
		ObservedNetBuildTimeSavedInterval95Ms  []float64 `json:"observedNetBuildTimeSavedInterval95Ms"`
		ObservedBuildTimeReductionRatio        float64   `json:"observedBuildTimeReductionRatio"`
		ObservedBuildTimeReductionInterval95   []float64 `json:"observedBuildTimeReductionInterval95"`
		CustomerVisibleBuildP95DeltaMs         int       `json:"customerVisibleBuildP95DeltaMs"`
		CustomerVisibleBuildP95Interval95Ms    []float64 `json:"customerVisibleBuildP95DeltaInterval95Ms"`
		CustomerVisibleFeedbackP95DeltaMs      int       `json:"customerVisibleFeedbackP95DeltaMs"`
		CustomerVisibleFeedbackP95Interval95Ms []float64 `json:"customerVisibleFeedbackP95DeltaInterval95Ms"`
		CIQueueP95DeltaMs                      int       `json:"ciQueueP95DeltaMs"`
		CIQueueP95Interval95Ms                 []float64 `json:"ciQueueP95DeltaInterval95Ms"`
	} `json:"effects"`
	Economics *struct {
		BuildComputeCostAvoided         float64 `json:"buildComputeCostAvoided"`
		ProductRuntimeCost              float64 `json:"productRuntimeCost"`
		ValidationAndControlComputeCost float64 `json:"validationAndControlComputeCost"`
		IncrementalStorageCost          float64 `json:"incrementalStorageCost"`
		IncrementalNetworkCost          float64 `json:"incrementalNetworkCost"`
		NetInfrastructureValue          float64 `json:"netInfrastructureValue"`
	} `json:"economics"`
	Decision struct {
		State          string `json:"state"`
		PromotionClass string `json:"promotionClass"`
		EvaluatedAt    string `json:"evaluatedAt"`
	} `json:"decision"`
	Invalidation *struct {
		InvalidatedResultVersion int    `json:"invalidatedResultVersion"`
		InvalidatedAt            string `json:"invalidatedAt"`
	} `json:"invalidation"`
}

type outcomeCounts struct {
	Success      int `json:"SUCCESS"`
	BuildFailure int `json:"BUILD_FAILURE"`
	InfraFailure int `json:"INFRA_FAILURE"`
	Cancelled    int `json:"CANCELLED"`
}

func (counts outcomeCounts) total() int {
	return counts.Success + counts.BuildFailure + counts.InfraFailure + counts.Cancelled
}

type actionSummary struct {
	ActionID     string `json:"actionId"`
	Sequence     int    `json:"sequence"`
	OccurredAt   string `json:"occurredAt"`
	EffectScope  string `json:"effectScope"`
	Transition   string `json:"transition"`
	FromState    string `json:"fromState"`
	Precondition struct {
		ExpectedState    string `json:"expectedState"`
		ExpectedSequence int    `json:"expectedSequence"`
	} `json:"precondition"`
	Policy struct {
		MeasurementPolicyVersion string `json:"measurementPolicyVersion"`
	} `json:"policy"`
	Authorization struct {
		Basis        string `json:"basis"`
		AuthorizedAt string `json:"authorizedAt"`
		ResultRef    *struct {
			ExperimentID             string `json:"experimentId"`
			ResultVersion            int    `json:"resultVersion"`
			Status                   string `json:"status"`
			MeasurementPolicyVersion string `json:"measurementPolicyVersion"`
		} `json:"experimentResultRef"`
	} `json:"authorization"`
}

type lifecycleVector struct {
	ExperimentResult string `json:"experimentResult"`
	ActionRecord     string `json:"actionRecord"`
}

func readExperimentSummary(t *testing.T, path string) *experimentSummary {
	t.Helper()
	var result experimentSummary
	readJSONFields(t, path, &result, false)
	return &result
}

func readActionSummary(t *testing.T, path string) *actionSummary {
	t.Helper()
	var action actionSummary
	readJSONFields(t, path, &action, false)
	return &action
}

func readLifecycleVector(t *testing.T, path string) lifecycleVector {
	t.Helper()
	var vector lifecycleVector
	readJSONFields(t, path, &vector, true)
	if vector.ActionRecord == "" {
		t.Fatalf("%s has no actionRecord", path)
	}
	return vector
}

func readJSONFields(t *testing.T, path string, target any, rejectUnknown bool) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	decoder := json.NewDecoder(file)
	if rejectUnknown {
		decoder.DisallowUnknownFields()
	}
	decodeErr := decoder.Decode(target)
	var trailing any
	trailingErr := decoder.Decode(&trailing)
	closeErr := file.Close()
	if decodeErr != nil {
		t.Fatalf("decode %s: %v", path, decodeErr)
	}
	if !errors.Is(trailingErr, io.EOF) {
		t.Fatalf("decode %s trailing content: %v", path, trailingErr)
	}
	if closeErr != nil {
		t.Fatalf("close %s: %v", path, closeErr)
	}
}

func validateVectorDocuments(
	t *testing.T,
	repositoryRoot string,
	experimentSchema *jsonschema.Schema,
	actionSchema *jsonschema.Schema,
	vectorPath string,
	vector lifecycleVector,
) (*experimentSummary, *actionSummary) {
	t.Helper()

	actionPath := resolveFixtureReference(t, repositoryRoot, vectorPath, vector.ActionRecord)
	if err := actionSchema.Validate(readJSON(t, actionPath)); err != nil {
		t.Fatalf("%s action must be schema-valid: %v", relativePath(repositoryRoot, vectorPath), err)
	}
	action := readActionSummary(t, actionPath)
	if err := validateActionSemantics(action); err != nil {
		t.Fatalf("%s action semantics: %v", relativePath(repositoryRoot, vectorPath), err)
	}

	if vector.ExperimentResult == "" {
		return nil, action
	}
	resultPath := resolveFixtureReference(t, repositoryRoot, vectorPath, vector.ExperimentResult)
	if err := experimentSchema.Validate(readJSON(t, resultPath)); err != nil {
		t.Fatalf("%s result must be schema-valid: %v", relativePath(repositoryRoot, vectorPath), err)
	}
	result := readExperimentSummary(t, resultPath)
	if err := validateExperimentSemantics(result); err != nil {
		t.Fatalf("%s result semantics: %v", relativePath(repositoryRoot, vectorPath), err)
	}
	return result, action
}

func resolveFixtureReference(
	t *testing.T,
	repositoryRoot string,
	vectorPath string,
	reference string,
) string {
	t.Helper()
	resolved := filepath.Clean(filepath.Join(filepath.Dir(vectorPath), reference))
	testdataRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema", "testdata")
	relative, err := filepath.Rel(testdataRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("fixture reference escapes testdata: %q", reference)
	}
	if filepath.Ext(resolved) != ".json" {
		t.Fatalf("fixture reference is not JSON: %q", reference)
	}
	return resolved
}

func validateExperimentSemantics(result *experimentSummary) error {
	startedAt, err := time.Parse(time.RFC3339Nano, result.Window.StartedAt)
	if err != nil {
		return fmt.Errorf("window.startedAt: %w", err)
	}
	endedAt, err := time.Parse(time.RFC3339Nano, result.Window.EndedAt)
	if err != nil {
		return fmt.Errorf("window.endedAt: %w", err)
	}
	asOf, err := time.Parse(time.RFC3339Nano, result.AsOf)
	if err != nil {
		return fmt.Errorf("asOf: %w", err)
	}
	if !startedAt.Before(endedAt) {
		return errors.New("window.startedAt must precede window.endedAt")
	}
	if endedAt.After(asOf) {
		return errors.New("window.endedAt must not follow asOf")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, result.Decision.EvaluatedAt)
	if err != nil {
		return fmt.Errorf("decision.evaluatedAt: %w", err)
	}
	if evaluatedAt.After(asOf) {
		return errors.New("decision.evaluatedAt must not follow asOf")
	}
	if result.ResultVersion > 1 &&
		result.SupersedesResultVersion != result.ResultVersion-1 {
		return errors.New("supersedesResultVersion must be the immediate predecessor")
	}
	if result.Status == "FINAL" && result.Decision.State == "PROMOTE" {
		minimumWindow := 7 * 24 * time.Hour
		if result.Decision.PromotionClass == "PROOF_GATED" {
			minimumWindow = 14 * 24 * time.Hour
		}
		if endedAt.Sub(startedAt) < minimumWindow {
			return errors.New("promotion window is below the beta minimum")
		}
	}
	if result.Samples.Analyzed.Candidate > result.Samples.Assigned.Candidate ||
		result.Samples.Analyzed.Control > result.Samples.Assigned.Control {
		return errors.New("analyzed sample cannot exceed assigned sample")
	}
	if result.Samples.Outcomes.Candidate.total() != result.Samples.Assigned.Candidate ||
		result.Samples.Outcomes.Control.total() != result.Samples.Assigned.Control {
		return errors.New("outcome counts must reconcile with assigned sample")
	}
	excludedTotal := 0
	for _, exclusion := range result.Samples.Exclusions {
		excludedTotal += exclusion.Count
	}
	if excludedTotal != result.Samples.ExcludedSampleSize {
		return errors.New("exclusion counts must reconcile with excludedSampleSize")
	}
	if result.Effects != nil {
		if err := orderedInterval(
			"observedNetBuildTimeSavedInterval95Ms",
			result.Effects.ObservedNetBuildTimeSavedInterval95Ms,
			float64(result.Effects.ObservedNetBuildTimeSavedMs),
		); err != nil {
			return err
		}
		if err := orderedInterval(
			"observedBuildTimeReductionInterval95",
			result.Effects.ObservedBuildTimeReductionInterval95,
			result.Effects.ObservedBuildTimeReductionRatio,
		); err != nil {
			return err
		}
		if err := orderedInterval(
			"customerVisibleBuildP95DeltaInterval95Ms",
			result.Effects.CustomerVisibleBuildP95Interval95Ms,
			float64(result.Effects.CustomerVisibleBuildP95DeltaMs),
		); err != nil {
			return err
		}
		if len(result.Effects.CustomerVisibleFeedbackP95Interval95Ms) > 0 {
			if err := orderedInterval(
				"customerVisibleFeedbackP95DeltaInterval95Ms",
				result.Effects.CustomerVisibleFeedbackP95Interval95Ms,
				float64(result.Effects.CustomerVisibleFeedbackP95DeltaMs),
			); err != nil {
				return err
			}
		}
		if len(result.Effects.CIQueueP95Interval95Ms) > 0 {
			if err := orderedInterval(
				"ciQueueP95DeltaInterval95Ms",
				result.Effects.CIQueueP95Interval95Ms,
				float64(result.Effects.CIQueueP95DeltaMs),
			); err != nil {
				return err
			}
		}
	}
	if result.Economics != nil {
		expectedNet := result.Economics.BuildComputeCostAvoided -
			result.Economics.ProductRuntimeCost -
			result.Economics.ValidationAndControlComputeCost -
			result.Economics.IncrementalStorageCost -
			result.Economics.IncrementalNetworkCost
		if math.Abs(expectedNet-result.Economics.NetInfrastructureValue) > 1e-9 {
			return errors.New("economics components do not reconcile")
		}
	}
	if result.Invalidation != nil &&
		result.Invalidation.InvalidatedResultVersion != result.SupersedesResultVersion {
		return errors.New("invalidation must identify the superseded result version")
	}
	if result.Invalidation != nil {
		invalidatedAt, err := time.Parse(
			time.RFC3339Nano,
			result.Invalidation.InvalidatedAt,
		)
		if err != nil {
			return fmt.Errorf("invalidation.invalidatedAt: %w", err)
		}
		if invalidatedAt.After(asOf) {
			return errors.New("invalidation.invalidatedAt must not follow asOf")
		}
	}
	return nil
}

func orderedInterval(name string, interval []float64, estimate float64) error {
	if len(interval) != 2 || interval[0] > interval[1] {
		return fmt.Errorf("%s endpoints are not ordered", name)
	}
	if estimate < interval[0] || estimate > interval[1] {
		return fmt.Errorf("%s does not contain its estimate", name)
	}
	return nil
}

func validateActionSemantics(action *actionSummary) error {
	if action.Precondition.ExpectedState != action.FromState {
		return errors.New("precondition.expectedState must equal fromState")
	}
	if action.Precondition.ExpectedSequence != action.Sequence-1 {
		return errors.New("precondition.expectedSequence must precede sequence")
	}
	authorizedAt, err := time.Parse(time.RFC3339Nano, action.Authorization.AuthorizedAt)
	if err != nil {
		return fmt.Errorf("authorization.authorizedAt: %w", err)
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, action.OccurredAt)
	if err != nil {
		return fmt.Errorf("occurredAt: %w", err)
	}
	if authorizedAt.After(occurredAt) {
		return errors.New("authorization must not follow the recorded transition")
	}
	if action.Authorization.ResultRef != nil &&
		action.Authorization.ResultRef.MeasurementPolicyVersion !=
			action.Policy.MeasurementPolicyVersion {
		return errors.New("authorization measurementPolicyVersion must match policy")
	}
	return nil
}

func validateLinkedLifecycle(result *experimentSummary, action *actionSummary) error {
	ref := action.Authorization.ResultRef
	if result == nil {
		if ref != nil {
			return errors.New("experimentResultRef has no linked result")
		}
		if action.Authorization.Basis != "POLICY" {
			return errors.New("result-free transition must have POLICY basis")
		}
		return nil
	}
	if ref == nil {
		return errors.New("linked result requires experimentResultRef")
	}
	if ref.ExperimentID != result.ExperimentID {
		return errors.New("experimentId does not match linked result")
	}
	if ref.ResultVersion != result.ResultVersion {
		return errors.New("resultVersion does not match linked result")
	}
	if ref.Status != result.Status {
		return errors.New("status does not match linked result")
	}
	if ref.MeasurementPolicyVersion != result.MeasurementPolicyVersion {
		return errors.New("measurementPolicyVersion does not match linked result")
	}
	if action.EffectScope != result.EffectScope {
		return errors.New("effectScope does not match linked result")
	}
	if result.EffectScope == "ACTION_INCREMENTAL" &&
		!containsString(result.ActionIDs, action.ActionID) {
		return errors.New("actionId is outside linked result population")
	}
	asOf, err := time.Parse(time.RFC3339Nano, result.AsOf)
	if err != nil {
		return fmt.Errorf("result asOf: %w", err)
	}
	authorizedAt, err := time.Parse(time.RFC3339Nano, action.Authorization.AuthorizedAt)
	if err != nil {
		return fmt.Errorf("authorization.authorizedAt: %w", err)
	}
	if authorizedAt.Before(asOf) {
		return errors.New("authorization predates the linked result")
	}
	switch action.Transition {
	case "ACTIVATE_IN_CI", "ACTIVATE_LOCALLY":
		if result.Status != "FINAL" {
			return errors.New("activation requires FINAL status")
		}
		if result.Decision.State != "PROMOTE" {
			return errors.New("activation requires a PROMOTE decision")
		}
	case "ROLLBACK":
		if result.Status != "INVALIDATED" || result.Decision.State != "INVALIDATED" {
			return errors.New("linked rollback result must be INVALIDATED")
		}
	}
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
