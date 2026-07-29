package neutralenvelope

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/metricscatalog"
)

func TestBuildReportRetainsFirstAndNegativeOverhead(t *testing.T) {
	report := buildTestReport(t)

	if report.Summary.PairCount != 3 ||
		!report.Summary.FirstExecutionIncluded ||
		report.Summary.FirstProductOverheadMs != 10 ||
		report.Summary.ProductOverheadP50Ms != 10 ||
		report.Summary.ProductOverheadP95Ms != 20 ||
		math.Abs(report.Summary.ProductOverheadMeanMs-25.0/3.0) > 1e-9 {
		t.Fatalf("unexpected report summary: %+v", report.Summary)
	}
	if report.Pairs[1].ProductSynchronousOverheadMs != -5 {
		t.Fatalf("negative overhead was not retained: %+v", report.Pairs[1])
	}
	if report.PromotionGateActive {
		t.Fatal("walking-skeleton report activated a promotion gate")
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("validate report: %v", err)
	}
}

func TestReportRejectsDriftAndInvalidPairs(t *testing.T) {
	report := buildTestReport(t)
	testCases := []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "promotion gate",
			mutate: func(candidate *Report) {
				candidate.PromotionGateActive = true
			},
			want: "contract",
		},
		{
			name: "summary",
			mutate: func(candidate *Report) {
				candidate.Summary.ProductOverheadP50Ms++
			},
			want: "summary",
		},
		{
			name: "deliverable mismatch",
			mutate: func(candidate *Report) {
				candidate.Pairs[0].Wrapper.DeliverableSHA256 =
					"sha256:" + strings.Repeat("b", 64)
			},
			want: "pair",
		},
		{
			name: "order",
			mutate: func(candidate *Report) {
				candidate.Pairs[1].FirstArm = "NATIVE"
			},
			want: "pair",
		},
		{
			name: "runner",
			mutate: func(candidate *Report) {
				candidate.Runner.CPUCount = 12
			},
			want: "golden",
		},
		{
			name: "runner qualification",
			mutate: func(candidate *Report) {
				candidate.RunnerClassQualified = false
			},
			want: "qualification",
		},
		{
			name: "workload",
			mutate: func(candidate *Report) {
				candidate.Workload.OptimizationsEnabled = true
			},
			want: "workload",
		},
		{
			name: "timestamp order",
			mutate: func(candidate *Report) {
				candidate.Pairs[0].Wrapper.StartedAt =
					candidate.Pairs[0].Native.StartedAt
			},
			want: "not first",
		},
		{
			name: "limitations",
			mutate: func(candidate *Report) {
				candidate.Limitations[0] = "promotion ready"
			},
			want: "limitations",
		},
		{
			name: "input digest",
			mutate: func(candidate *Report) {
				candidate.Inputs.LauncherSHA256 = "sha256:bad"
			},
			want: "digest",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := cloneReport(t, report)
			testCase.mutate(&candidate)
			if err := candidate.Validate(); err == nil ||
				!strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("validation error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestBuildReportRejectsMissingArmAndOrderBias(t *testing.T) {
	observations := testObservations(t)
	inputs := testInputPaths(t)

	testCases := []struct {
		name   string
		mutate func([]Observation) []Observation
		want   string
	}{
		{
			name: "missing arm",
			mutate: func(candidate []Observation) []Observation {
				candidate[1].Arm = "NATIVE"
				return candidate
			},
			want: "repeats",
		},
		{
			name: "order bias",
			mutate: func(candidate []Observation) []Observation {
				candidate[2].OrderInPair = 2
				candidate[3].OrderInPair = 1
				return candidate
			},
			want: "alternate",
		},
		{
			name: "incomplete",
			mutate: func(candidate []Observation) []Observation {
				return candidate[:len(candidate)-1]
			},
			want: "complete pairs",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := append([]Observation(nil), observations...)
			candidate = testCase.mutate(candidate)
			if _, err := BuildReport(
				candidate,
				"STRICT_GOLDEN_CONTAINER",
				inputs.runnerSpec,
				inputs.metricsCatalog,
				inputs.envelope,
				inputs.launcher,
				inputs.server,
				inputs.plugin,
				time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC),
			); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("build report error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestBuildReportRejectsOverheadMetricDrift(t *testing.T) {
	paths := testInputPaths(t)
	content, err := os.ReadFile(paths.metricsCatalog)
	if err != nil {
		t.Fatalf("read metrics catalog: %v", err)
	}
	var catalog metricscatalog.Catalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatalf("decode metrics catalog: %v", err)
	}
	for index := range catalog.Metrics {
		if catalog.Metrics[index].ID == "productSynchronousOverheadMs" {
			catalog.Metrics[index].SignConvention = "NON_NEGATIVE"
		}
	}
	driftedPath := filepath.Join(t.TempDir(), "metrics.json")
	driftedContent, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("encode drifted catalog: %v", err)
	}
	if err := os.WriteFile(driftedPath, driftedContent, 0o600); err != nil {
		t.Fatalf("write drifted catalog: %v", err)
	}
	if _, err := BuildReport(
		testObservations(t),
		"STRICT_GOLDEN_CONTAINER",
		paths.runnerSpec,
		driftedPath,
		paths.envelope,
		paths.launcher,
		paths.server,
		paths.plugin,
		time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC),
	); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("BuildReport error = %v, want incompatible metric", err)
	}
}

func TestBuildReportMarksHostSmokeNonQualifying(t *testing.T) {
	paths := testInputPaths(t)
	report, err := BuildReport(
		testObservations(t),
		"HOST_SMOKE",
		paths.runnerSpec,
		paths.metricsCatalog,
		paths.envelope,
		paths.launcher,
		paths.server,
		paths.plugin,
		time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("build host report: %v", err)
	}
	if report.RunnerClassQualified ||
		!strings.Contains(report.Limitations[len(report.Limitations)-1], "does not qualify") {
		t.Fatalf("host qualification is misleading: %+v", report)
	}
}

func TestLoadReportRejectsUnknownAndTrailingJSON(t *testing.T) {
	report := buildTestReport(t)
	path := filepath.Join(t.TempDir(), "report.json")
	if err := WriteJSON(path, report); err != nil {
		t.Fatalf("write report: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	testCases := []string{
		strings.Replace(
			string(content),
			`"schemaVersion": "1.0"`,
			`"schemaVersion": "1.0", "unknown": true`,
			1,
		),
		string(content) + `{}`,
	}
	for index, candidateContent := range testCases {
		candidate := filepath.Join(
			t.TempDir(),
			fmtName("invalid", index),
		)
		if err := os.WriteFile(
			candidate,
			[]byte(candidateContent),
			0o600,
		); err != nil {
			t.Fatalf("write invalid report: %v", err)
		}
		if _, err := LoadReport(candidate); err == nil {
			t.Fatal("invalid report passed")
		}
	}
}

func buildTestReport(t *testing.T) Report {
	t.Helper()
	inputs := testInputPaths(t)
	report, err := BuildReport(
		testObservations(t),
		"STRICT_GOLDEN_CONTAINER",
		inputs.runnerSpec,
		inputs.metricsCatalog,
		inputs.envelope,
		inputs.launcher,
		inputs.server,
		inputs.plugin,
		time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("build test report: %v", err)
	}
	return report
}

func testObservations(t *testing.T) []Observation {
	t.Helper()
	base := time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC)
	digest := "sha256:" + strings.Repeat("a", 64)
	type input struct {
		arm      string
		pair     int
		order    int
		duration time.Duration
	}
	inputs := []input{
		{arm: "NATIVE", pair: 1, order: 1, duration: 1000 * time.Millisecond},
		{arm: "WRAPPER", pair: 1, order: 2, duration: 1010 * time.Millisecond},
		{arm: "WRAPPER", pair: 2, order: 1, duration: 995 * time.Millisecond},
		{arm: "NATIVE", pair: 2, order: 2, duration: 1000 * time.Millisecond},
		{arm: "NATIVE", pair: 3, order: 1, duration: 1000 * time.Millisecond},
		{arm: "WRAPPER", pair: 3, order: 2, duration: 1020 * time.Millisecond},
	}
	observations := make([]Observation, 0, len(inputs))
	for index, input := range inputs {
		startedAt := base.Add(time.Duration(index) * 2 * time.Second)
		commandClass := "gradle-neutral-probe-native-v1"
		if input.arm == "WRAPPER" {
			commandClass = "buildopt-gradle-neutral-probe-wrapper-v1"
		}
		observation, err := NewObservation(
			input.arm,
			input.pair,
			input.order,
			commandClass,
			startedAt,
			startedAt.Add(input.duration),
			0,
			digest,
			32,
		)
		if err != nil {
			t.Fatalf("create test observation: %v", err)
		}
		observations = append(observations, observation)
	}
	return observations
}

type inputPaths struct {
	runnerSpec     string
	metricsCatalog string
	envelope       string
	launcher       string
	server         string
	plugin         string
}

func testInputPaths(t *testing.T) inputPaths {
	t.Helper()
	root := filepath.Join("..", "..")
	directory := t.TempDir()
	paths := inputPaths{
		runnerSpec: filepath.Join(
			root,
			"specs",
			"golden-lane-runner-v1.json",
		),
		metricsCatalog: filepath.Join(
			root,
			"contracts",
			"metrics",
			"build-impact-v1.json",
		),
		envelope: filepath.Join(directory, "neutral-envelope"),
		launcher: filepath.Join(directory, "buildopt"),
		server:   filepath.Join(directory, "buildopt-server"),
		plugin:   filepath.Join(directory, "plugin.jar"),
	}
	for _, path := range []string{
		paths.envelope,
		paths.launcher,
		paths.server,
		paths.plugin,
	} {
		if err := os.WriteFile(path, []byte(filepath.Base(path)), 0o600); err != nil {
			t.Fatalf("write test input: %v", err)
		}
	}
	return paths
}

func cloneReport(t *testing.T, report Report) Report {
	t.Helper()
	content, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("encode report: %v", err)
	}
	var clone Report
	if err := json.Unmarshal(content, &clone); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return clone
}

func fmtName(prefix string, index int) string {
	if index == 0 {
		return prefix + "-unknown.json"
	}
	return prefix + "-trailing.json"
}
