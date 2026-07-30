package neutralenvelope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildNoHitReportPassesExactA0Budget(t *testing.T) {
	report := buildNoHitTestReport(t)

	if !report.Gate.Passed ||
		!report.Gate.LongPassed ||
		!report.Gate.ShortPassed ||
		report.Summary.ProductSynchronousOverheadP95Ms != 200 ||
		report.Summary.ProductSynchronousOverheadP95Ratio != 0.01 {
		t.Fatalf("unexpected no-hit gate: %+v/%+v", report.Gate, report.Summary)
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("validate no-hit report: %v", err)
	}
}

func TestNoHitReportRejectsBudgetAndOmissionDrift(t *testing.T) {
	report := buildNoHitTestReport(t)
	testCases := []struct {
		name   string
		mutate func(*NoHitReport)
		want   string
	}{
		{
			name: "millisecond budget",
			mutate: func(candidate *NoHitReport) {
				candidate.Pairs[3].Wrapper.DurationMs += 400
				candidate.Pairs[3].ProductSynchronousOverheadMs += 400
				candidate.Pairs[3].ProductSynchronousOverheadRatio =
					candidate.Pairs[3].ProductSynchronousOverheadMs /
						candidate.Pairs[3].Native.DurationMs
			},
			want: "summary",
		},
		{
			name: "ratio budget",
			mutate: func(candidate *NoHitReport) {
				candidate.Gate.LongMaximumP95Ratio = 0.03
			},
			want: "gate",
		},
		{
			name: "short remote request",
			mutate: func(candidate *NoHitReport) {
				candidate.Workload.ShortRemoteRequests = 1
			},
			want: "workload",
		},
		{
			name: "short threshold",
			mutate: func(candidate *NoHitReport) {
				candidate.Workload.ShortSessionDuration = 5000
			},
			want: "workload",
		},
		{
			name: "missing miss",
			mutate: func(candidate *NoHitReport) {
				candidate.Workload.LongRemoteMisses = 3
			},
			want: "workload",
		},
		{
			name: "strict delay",
			mutate: func(candidate *NoHitReport) {
				candidate.Workload.LongProbeDelayMs = 15000
			},
			want: "strict no-hit probe delay",
		},
		{
			name: "command class",
			mutate: func(candidate *NoHitReport) {
				candidate.Pairs[0].Wrapper.CommandClass = "unmeasured"
			},
			want: "pair",
		},
		{
			name: "false pass",
			mutate: func(candidate *NoHitReport) {
				candidate.Gate.Passed = false
			},
			want: "gate",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := cloneNoHitReport(t, report)
			testCase.mutate(&candidate)
			if err := candidate.Validate(); err == nil ||
				!strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("validation error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestNoHitHostSmokeCannotPassQualifiedGate(t *testing.T) {
	inputs := testInputPaths(t)
	report, err := BuildNoHitReport(
		noHitTestObservations(t)[:4],
		"HOST_SMOKE",
		inputs.runnerSpec,
		inputs.metricsCatalog,
		inputs.envelope,
		inputs.launcher,
		inputs.server,
		inputs.plugin,
		inputs.server,
		inputs.server,
		15000,
		2,
		1000,
		0,
		time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("build host no-hit report: %v", err)
	}
	if report.RunnerClassQualified || report.Gate.Passed ||
		!report.Gate.LongPassed || !report.Gate.ShortPassed {
		t.Fatalf("host report qualification = %+v", report.Gate)
	}
}

func TestLoadNoHitReportRejectsUnknownJSON(t *testing.T) {
	report := buildNoHitTestReport(t)
	content, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	content = []byte(strings.Replace(
		string(content),
		`"schemaVersion":"1.0"`,
		`"schemaVersion":"1.0","unknown":true`,
		1,
	))
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNoHitReport(path); err == nil {
		t.Fatal("unknown no-hit report field passed")
	}
}

func buildNoHitTestReport(t *testing.T) NoHitReport {
	t.Helper()
	inputs := testInputPaths(t)
	report, err := BuildNoHitReport(
		noHitTestObservations(t),
		"STRICT_GOLDEN_CONTAINER",
		inputs.runnerSpec,
		inputs.metricsCatalog,
		inputs.envelope,
		inputs.launcher,
		inputs.server,
		inputs.plugin,
		inputs.server,
		inputs.server,
		25000,
		4,
		1000,
		0,
		time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("build no-hit report: %v", err)
	}
	return report
}

func noHitTestObservations(t *testing.T) []Observation {
	t.Helper()
	base := time.Date(2026, 7, 30, 11, 0, 0, 0, time.UTC)
	digest := "sha256:" + strings.Repeat("a", 64)
	overheads := []time.Duration{
		100 * time.Millisecond,
		-50 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
	}
	observations := make([]Observation, 0, len(overheads)*2)
	for index, overhead := range overheads {
		pair := index + 1
		firstArm := "NATIVE"
		secondArm := "WRAPPER"
		if pair%2 == 0 {
			firstArm, secondArm = secondArm, firstArm
		}
		pairStart := base.Add(time.Duration(index) * time.Minute)
		for order, arm := range []string{firstArm, secondArm} {
			duration := 20 * time.Second
			commandClass := noHitNativeCommand
			if arm == "WRAPPER" {
				duration += overhead
				commandClass = noHitWrapperCommand
			}
			startedAt := pairStart.Add(
				time.Duration(order) * 25 * time.Second,
			)
			observation, err := NewObservation(
				arm,
				pair,
				order+1,
				commandClass,
				startedAt,
				startedAt.Add(duration),
				0,
				digest,
				32,
			)
			if err != nil {
				t.Fatalf("create no-hit observation: %v", err)
			}
			observations = append(observations, observation)
		}
	}
	return observations
}

func cloneNoHitReport(t *testing.T, report NoHitReport) NoHitReport {
	t.Helper()
	content, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var clone NoHitReport
	if err := json.Unmarshal(content, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
