package metricscatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormativeCatalog(t *testing.T) {
	path := filepath.Join(
		"..",
		"..",
		"contracts",
		"metrics",
		"build-impact-v1.json",
	)
	catalog, err := Load(path)
	if err != nil {
		t.Fatalf("load normative metrics catalog: %v", err)
	}
	if len(catalog.Metrics) != len(requiredMetricIDs) {
		t.Fatalf(
			"catalog metrics = %d, want %d",
			len(catalog.Metrics),
			len(requiredMetricIDs),
		)
	}
}

func TestCatalogRejectsSemanticDrift(t *testing.T) {
	catalog := loadTestCatalog(t)
	testCases := []struct {
		name   string
		mutate func(*Catalog)
		want   string
	}{
		{
			name: "definition version",
			mutate: func(candidate *Catalog) {
				candidate.MetricDefinitionVersion = "build-impact-v2"
			},
			want: "metricDefinitionVersion",
		},
		{
			name: "missing required metric",
			mutate: func(candidate *Catalog) {
				candidate.Metrics = candidate.Metrics[1:]
			},
			want: "required v1 set",
		},
		{
			name: "duplicate metric",
			mutate: func(candidate *Catalog) {
				candidate.Metrics[1].ID = candidate.Metrics[0].ID
			},
			want: "duplicate metric",
		},
		{
			name: "saved sign reversed",
			mutate: func(candidate *Catalog) {
				metric := findMetric(t, candidate, "observedNetBuildTimeSavedMs")
				metric.SignConvention = "NEGATIVE_IS_IMPROVEMENT"
			},
			want: "saved/reduction sign",
		},
		{
			name: "delta sign reversed",
			mutate: func(candidate *Catalog) {
				metric := findMetric(t, candidate, "customerVisibleBuildP95DeltaMs")
				metric.SignConvention = "POSITIVE_IS_IMPROVEMENT"
			},
			want: "delta sign",
		},
		{
			name: "unsafe null policy",
			mutate: func(candidate *Catalog) {
				candidate.Metrics[0].NullPolicy = "ZERO"
			},
			want: "unsafe null policy",
		},
		{
			name: "unsupported unit",
			mutate: func(candidate *Catalog) {
				candidate.Metrics[0].Unit = "seconds"
			},
			want: "unsupported unit",
		},
		{
			name: "unsupported sign",
			mutate: func(candidate *Catalog) {
				candidate.Metrics[0].SignConvention = "LOWER_IS_BETTER"
			},
			want: "unsupported sign",
		},
		{
			name: "unknown dimension",
			mutate: func(candidate *Catalog) {
				candidate.Metrics[0].Dimensions = append(
					candidate.Metrics[0].Dimensions,
					"taskPath",
				)
			},
			want: "unsupported dimension",
		},
		{
			name: "promotion sample",
			mutate: func(candidate *Catalog) {
				candidate.PromotionPolicy.DirectReversible.
					MinimumObservationsPerArm = 99
			},
			want: "MEASURE-001",
		},
		{
			name: "correctness is nonzero",
			mutate: func(candidate *Catalog) {
				candidate.PromotionPolicy.Correctness[0].Maximum = 1
			},
			want: "zero",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := cloneCatalog(t, catalog)
			testCase.mutate(&candidate)
			if err := candidate.Validate(); err == nil ||
				!strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("validation error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestLoadRejectsUnknownAndTrailingJSON(t *testing.T) {
	path := filepath.Join(
		"..",
		"..",
		"contracts",
		"metrics",
		"build-impact-v1.json",
	)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read normative catalog: %v", err)
	}

	testCases := []struct {
		name    string
		content string
	}{
		{
			name: "unknown field",
			content: strings.Replace(
				string(content),
				`"schemaVersion": "1.0"`,
				`"schemaVersion": "1.0", "unknown": true`,
				1,
			),
		},
		{
			name:    "trailing JSON",
			content: string(content) + `{}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := filepath.Join(t.TempDir(), "catalog.json")
			if err := os.WriteFile(
				candidate,
				[]byte(testCase.content),
				0o600,
			); err != nil {
				t.Fatalf("write candidate catalog: %v", err)
			}
			if _, err := Load(candidate); err == nil {
				t.Fatal("invalid catalog passed")
			}
		})
	}
}

func loadTestCatalog(t *testing.T) Catalog {
	t.Helper()
	path := filepath.Join(
		"..",
		"..",
		"contracts",
		"metrics",
		"build-impact-v1.json",
	)
	catalog, err := Load(path)
	if err != nil {
		t.Fatalf("load test catalog: %v", err)
	}
	return catalog
}

func cloneCatalog(t *testing.T, catalog Catalog) Catalog {
	t.Helper()
	content, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("encode test catalog: %v", err)
	}
	var clone Catalog
	if err := json.Unmarshal(content, &clone); err != nil {
		t.Fatalf("decode test catalog: %v", err)
	}
	return clone
}

func findMetric(t *testing.T, catalog *Catalog, id string) *Metric {
	t.Helper()
	for index := range catalog.Metrics {
		if catalog.Metrics[index].ID == id {
			return &catalog.Metrics[index]
		}
	}
	t.Fatalf("missing test metric %s", id)
	return nil
}
