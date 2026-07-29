package schemavalidator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const buildSessionSchemaID = "https://schemas.buildopt.dev/build-session.v1.schema.json"

func TestBuildSessionV1Fixtures(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaPath := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"build-session.v1.schema.json",
	)

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()

	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", relativePath(repositoryRoot, schemaPath), err)
	}
	if schema.ID != buildSessionSchemaID {
		t.Fatalf(
			"compiled schema ID = %q, want %q",
			schema.ID,
			buildSessionSchemaID,
		)
	}
	if schema.DraftVersion != 2020 {
		t.Fatalf("compiled schema draft = %d, want 2020-12", schema.DraftVersion)
	}

	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"build-session.v1",
	)
	validFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	invalidFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))

	expectedValidFixtures := map[string]struct{}{
		"complete-build-failure.json":     {},
		"complete-ci-candidate.json":      {},
		"complete-local-passthrough.json": {},
		"partial-recovery.json":           {},
	}
	if len(validFixtures) != len(expectedValidFixtures) {
		t.Fatalf(
			"found %d valid fixtures, want %d documented fixtures",
			len(validFixtures),
			len(expectedValidFixtures),
		)
	}
	for _, fixturePath := range validFixtures {
		fixturePath := fixturePath
		t.Run("valid/"+filepath.Base(fixturePath), func(t *testing.T) {
			if _, ok := expectedValidFixtures[filepath.Base(fixturePath)]; !ok {
				t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
			}

			instance := readJSON(t, fixturePath)
			if err := schema.Validate(instance); err != nil {
				t.Fatalf(
					"%s must validate: %v",
					relativePath(repositoryRoot, fixturePath),
					err,
				)
			}
		})
	}

	expectedInvalidDiagnostics := map[string]string{
		"complete-with-partial-metadata.json": "/measurementMetadata/status",
		"future-aggregate-effect.json":        "observedNetBuildTimeSavedMs",
		"impossible-timestamp.json":           "/build/startedAt",
		"negative-duration.json":              "/performance/customerVisibleBuildMs/valueMs",
		"partial-without-recovery.json":       "recovery",
		"success-with-nonzero-exit.json":      "/build/exitCode",
		"unavailable-with-value.json":         "/performance/customerVisibleFeedbackMs",
	}
	if len(invalidFixtures) != len(expectedInvalidDiagnostics) {
		t.Fatalf(
			"found %d invalid fixtures, want %d documented diagnostics",
			len(invalidFixtures),
			len(expectedInvalidDiagnostics),
		)
	}

	for _, fixturePath := range invalidFixtures {
		fixturePath := fixturePath
		t.Run("invalid/"+filepath.Base(fixturePath), func(t *testing.T) {
			expectedDiagnostic, ok := expectedInvalidDiagnostics[filepath.Base(fixturePath)]
			if !ok {
				t.Fatalf("missing expected diagnostic for %s", filepath.Base(fixturePath))
			}

			instance := readJSON(t, fixturePath)
			err := schema.Validate(instance)
			if err == nil {
				t.Fatalf(
					"%s must be rejected",
					relativePath(repositoryRoot, fixturePath),
				)
			}
			if !strings.Contains(err.Error(), expectedDiagnostic) {
				t.Fatalf(
					"%s failed for an unexpected reason; want diagnostic containing %q, got: %v",
					relativePath(repositoryRoot, fixturePath),
					expectedDiagnostic,
					err,
				)
			}
		})
	}
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
}

func fixtureFiles(t *testing.T, directory string) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		t.Fatalf("list fixtures in %s: %v", directory, err)
	}
	if len(files) == 0 {
		t.Fatalf("no JSON fixtures found in %s", directory)
	}
	sort.Strings(files)
	return files
}

func readJSON(t *testing.T, path string) any {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close %s: %v", path, err)
		}
	}()

	instance, err := jsonschema.UnmarshalJSON(file)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return instance
}

func relativePath(repositoryRoot, path string) string {
	relative, err := filepath.Rel(repositoryRoot, path)
	if err != nil {
		return fmt.Sprintf("%s (%v)", path, err)
	}
	return filepath.ToSlash(relative)
}
