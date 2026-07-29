package schemavalidator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type betaBenchmarkManifest struct {
	SchemaVersion string `json:"schemaVersion"`
	Seed          int64  `json:"seed"`
	Runner        struct {
		Spec                    string `json:"spec"`
		SpecSHA256              string `json:"specSha256"`
		RunnerClass             string `json:"runnerClass"`
		ContainerPlatformDigest string `json:"containerPlatformDigest"`
	} `json:"runner"`
	Components struct {
		Gradle               string `json:"gradle"`
		JDK                  string `json:"jdk"`
		MetricsCatalog       string `json:"metricsCatalog"`
		MetricsCatalogSHA256 string `json:"metricsCatalogSha256"`
	} `json:"components"`
	Clients              []int                  `json:"clients"`
	ObjectCycleCount     int                    `json:"objectCycleCount"`
	ObjectMix            []betaBenchmarkObject  `json:"objectMix"`
	Phases               []betaBenchmarkPhase   `json:"phases"`
	GradleFixtures       []betaBenchmarkFixture `json:"gradleFixtures"`
	Faults               []betaBenchmarkFault   `json:"faults"`
	RequiredReportFields []string               `json:"requiredReportFields"`
	RequiredAlerts       []string               `json:"requiredAlerts"`
}

type betaBenchmarkObject struct {
	SizeBytes int64 `json:"sizeBytes"`
	Percent   int   `json:"percent"`
}

type betaBenchmarkPhase struct {
	ID               string `json:"id"`
	DurationSeconds  int    `json:"durationSeconds"`
	TargetHitPercent int    `json:"targetHitPercent"`
	AllClientCounts  bool   `json:"allClientCounts"`
}

type betaBenchmarkFixture struct {
	ID                   string `json:"id"`
	SizeClass            string `json:"sizeClass"`
	RequiredKnownOutputs bool   `json:"requiredKnownOutputs"`
	RequiredCriticalPath bool   `json:"requiredCriticalPath"`
}

type betaBenchmarkFault struct {
	ID             string `json:"id"`
	ExpectedSafety string `json:"expectedSafety"`
}

func TestBetaBenchmarkV1(t *testing.T) {
	t.Parallel()

	root := findRepositoryRoot(t)
	manifest, raw := loadBetaBenchmarkManifest(t, root)
	if manifest.SchemaVersion != "buildopt.benchmarks/beta/v1" ||
		manifest.Seed != 2026072901 {
		t.Errorf(
			"identity = %q/%d",
			manifest.SchemaVersion,
			manifest.Seed,
		)
	}
	if !slices.Equal(manifest.Clients, []int{1, 8, 32}) {
		t.Errorf("clients = %v", manifest.Clients)
	}
	if manifest.ObjectCycleCount != 10000 {
		t.Errorf("objectCycleCount = %d", manifest.ObjectCycleCount)
	}
	assertBetaObjectMix(t, manifest.ObjectMix)
	assertBetaPhases(t, manifest.Phases)
	assertBetaFixtures(t, manifest.GradleFixtures)
	assertBetaFaults(t, manifest.Faults)
	assertUniqueStrings(t, "requiredReportFields", manifest.RequiredReportFields, 18)
	assertUniqueStrings(t, "requiredAlerts", manifest.RequiredAlerts, 10)
	assertBetaRunnerBindings(t, root, manifest)
	sum := sha256.Sum256(raw)
	t.Logf("benchmark digest: sha256:%s", hex.EncodeToString(sum[:]))
}

func loadBetaBenchmarkManifest(
	t *testing.T,
	root string,
) (betaBenchmarkManifest, []byte) {
	t.Helper()
	path := filepath.Join(root, "benchmarks", "beta-v1.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	strict := json.NewDecoder(file)
	strict.DisallowUnknownFields()
	var manifest betaBenchmarkManifest
	if err := strict.Decode(&manifest); err != nil {
		t.Fatalf("decode JSON-compatible YAML %s: %v", path, err)
	}
	var trailing any
	if err := strict.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing data: %v", path, err)
	}
	return manifest, raw
}

func assertBetaObjectMix(t *testing.T, objects []betaBenchmarkObject) {
	t.Helper()
	want := []betaBenchmarkObject{
		{SizeBytes: 64 * 1024, Percent: 70},
		{SizeBytes: 1024 * 1024, Percent: 20},
		{SizeBytes: 10 * 1024 * 1024, Percent: 8},
		{SizeBytes: 100 * 1024 * 1024, Percent: 2},
	}
	if !slices.Equal(objects, want) {
		t.Errorf("objectMix = %+v, want %+v", objects, want)
	}
	total := 0
	for _, object := range objects {
		total += object.Percent
	}
	if total != 100 {
		t.Errorf("object percentages sum to %d", total)
	}
}

func assertBetaPhases(t *testing.T, phases []betaBenchmarkPhase) {
	t.Helper()
	want := []betaBenchmarkPhase{
		{ID: "COLD", DurationSeconds: 300, TargetHitPercent: 0, AllClientCounts: true},
		{ID: "WARM_70", DurationSeconds: 300, TargetHitPercent: 70, AllClientCounts: true},
		{ID: "SUSTAINED", DurationSeconds: 3600, TargetHitPercent: 70, AllClientCounts: true},
		{ID: "SOAK", DurationSeconds: 28800, TargetHitPercent: 70, AllClientCounts: true},
	}
	if !slices.Equal(phases, want) {
		t.Errorf("phases = %+v, want %+v", phases, want)
	}
}

func assertBetaFixtures(t *testing.T, fixtures []betaBenchmarkFixture) {
	t.Helper()
	wantClasses := []string{"SMALL", "MEDIUM", "LARGE"}
	if len(fixtures) != len(wantClasses) {
		t.Fatalf("fixture count = %d", len(fixtures))
	}
	for index, fixture := range fixtures {
		if fixture.SizeClass != wantClasses[index] ||
			!fixture.RequiredKnownOutputs ||
			!fixture.RequiredCriticalPath {
			t.Errorf("fixture %d = %+v", index, fixture)
		}
	}
}

func assertBetaFaults(t *testing.T, faults []betaBenchmarkFault) {
	t.Helper()
	want := []string{
		"GATEWAY_RESTART",
		"SERVER_RESTART",
		"MID_PUT_CANCEL",
		"MID_GET_CANCEL",
		"TRUNCATED_BLOB",
		"CORRUPT_BLOB",
		"NETWORK_LATENCY",
		"NETWORK_LOSS",
		"SQLITE_BUSY",
		"EXPIRED_LEASE",
		"DISK_HIGH_WATERMARK",
		"DISK_OUT_OF_SPACE",
		"REVOKED_POLICY",
		"REVOKED_GRANT",
		"DEATH_BETWEEN_PENDING_AND_COMMIT",
	}
	if len(faults) != len(want) {
		t.Fatalf("fault count = %d, want %d", len(faults), len(want))
	}
	for index, fault := range faults {
		if fault.ID != want[index] || fault.ExpectedSafety == "" {
			t.Errorf("fault %d = %+v", index, fault)
		}
	}
}

func assertBetaRunnerBindings(
	t *testing.T,
	root string,
	manifest betaBenchmarkManifest,
) {
	t.Helper()
	runnerPath := filepath.Join(root, filepath.FromSlash(manifest.Runner.Spec))
	raw, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("read runner spec: %v", err)
	}
	sum := sha256.Sum256(raw)
	actualDigest := "sha256:" + hex.EncodeToString(sum[:])
	if actualDigest != manifest.Runner.SpecSHA256 {
		t.Errorf(
			"runner spec digest = %s, want %s",
			actualDigest,
			manifest.Runner.SpecSHA256,
		)
	}
	var runner struct {
		RunnerClass struct {
			ID string `json:"id"`
		} `json:"runnerClass"`
		Container struct {
			PlatformDigest string `json:"platformDigest"`
		} `json:"container"`
		JDK struct {
			Version string `json:"version"`
		} `json:"jdk"`
		Gradle struct {
			Version string `json:"version"`
		} `json:"gradle"`
	}
	if err := json.Unmarshal(raw, &runner); err != nil {
		t.Fatalf("decode runner spec: %v", err)
	}
	if runner.RunnerClass.ID != manifest.Runner.RunnerClass ||
		runner.Container.PlatformDigest != manifest.Runner.ContainerPlatformDigest ||
		runner.JDK.Version != manifest.Components.JDK ||
		runner.Gradle.Version != manifest.Components.Gradle {
		t.Errorf("runner/component binding mismatch")
	}
	metricsPath := filepath.Join(root, "contracts", "metrics", "build-impact-v1.json")
	metrics, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatalf("read metrics catalog: %v", err)
	}
	metricsSum := sha256.Sum256(metrics)
	if got := fmt.Sprintf("sha256:%x", metricsSum); got != manifest.Components.MetricsCatalogSHA256 {
		t.Errorf("metrics catalog digest = %s", got)
	}
}

func assertUniqueStrings(
	t *testing.T,
	name string,
	values []string,
	minimum int,
) {
	t.Helper()
	if len(values) < minimum {
		t.Errorf("%s count = %d, want at least %d", name, len(values), minimum)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			t.Errorf("%s contains an empty value", name)
		}
		if _, duplicate := seen[value]; duplicate {
			t.Errorf("%s contains duplicate %q", name, value)
		}
		seen[value] = struct{}{}
	}
}
