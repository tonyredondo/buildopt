package profilediscovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBalancedStructuralQualificationUsesIndependentOrderBalancedCaptures(t *testing.T) {
	repository := structuralTestRepository(
		t, repositoryRoot(t),
		"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
		"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
		"fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
	)
	first := qualifiedStructuralTestEvidence(t, repository)
	second := qualifiedStructuralTestEvidence(t, repository)
	second.CapturedAt = "2026-08-10T00:00:00Z"
	for index := range first.Observations {
		candidate := int64(9500)
		if index%2 == 0 {
			candidate = 6000
		}
		first.Observations[index].CandidateDurationMS = candidate
		first.Observations[index].SavedMS = 10000 - candidate
		second.Observations[index].CandidateDurationMS = candidate
		second.Observations[index].SavedMS = 10000 - candidate
	}
	first.Result = mustStructuralResult(t, first.Observations)
	second.Result = mustStructuralResult(t, second.Observations)
	writeBalancedCapture(t, repository, "capture-1.json", first)
	writeBalancedCapture(t, repository, "capture-2.json", second)

	raw, qualified, err := RenderBalancedStructuralEvidence(BalancedStructuralOptions{
		RepositoryRoot: repository,
		EvidencePaths:  []string{"capture-1.json", "capture-2.json"},
		CapturedAt:     time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC),
	})
	if err != nil || !qualified {
		t.Fatalf("balanced qualification = %v/%v", qualified, err)
	}
	var evidence balancedStructuralEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != BalancedStructuralEvidenceSchema || evidence.EvidenceState != "QUALIFIED" ||
		evidence.Result.Captures != 2 || evidence.Result.Pairs != 16 || evidence.Result.Blocks != 8 ||
		evidence.Result.PositiveBlocks != 8 || evidence.Result.MeanSavedMS != 2250 ||
		evidence.Result.MedianBlockSavedMS != 2250 || evidence.Result.OrderEffectMS != 3500 ||
		evidence.Result.ControlP95MS != 10000 || evidence.Result.CandidateP95MS != 9500 ||
		len(evidence.Result.Interval95BlockSavedMS) != 2 || evidence.Result.Interval95BlockSavedMS[0] <= 0 ||
		!evidence.Result.Qualified || evidence.Boundaries.ProductionAuthorized {
		t.Fatalf("balanced evidence = %+v", evidence)
	}
}

func TestBalancedStructuralQualificationFailsClosed(t *testing.T) {
	repository := structuralTestRepository(
		t, repositoryRoot(t),
		"fixtures/poc-kafka-packaging/buildopt-impact-manifest.json",
		"fixtures/poc-kafka-packaging/buildopt-impact-graph.generated.json",
		"fixtures/poc-kafka-packaging/buildopt-impact.generated.json",
	)
	first := qualifiedStructuralTestEvidence(t, repository)
	second := qualifiedStructuralTestEvidence(t, repository)
	second.CapturedAt = "2026-08-10T00:00:00Z"
	writeBalancedCapture(t, repository, "capture-1.json", first)
	writeBalancedCapture(t, repository, "capture-2.json", second)

	t.Run("identity drift", func(t *testing.T) {
		drifted := second
		drifted.Execution.BuildOptRevision = strings.Repeat("e", 40)
		writeBalancedCapture(t, repository, "drifted.json", drifted)
		_, _, err := RenderBalancedStructuralEvidence(BalancedStructuralOptions{
			RepositoryRoot: repository,
			EvidencePaths:  []string{"capture-1.json", "drifted.json"},
			CapturedAt:     time.Now(),
		})
		if err == nil || !strings.Contains(err.Error(), "identity drift") {
			t.Fatalf("identity drift error = %v", err)
		}
	})

	t.Run("tail regression", func(t *testing.T) {
		regressive := second
		regressive.CapturedAt = "2026-08-11T00:00:00Z"
		regressive.Observations[7].CandidateDurationMS = 15000
		regressive.Observations[7].SavedMS = -5000
		regressive.Result = mustStructuralResult(t, regressive.Observations)
		regressive.EvidenceState = "INCONCLUSIVE"
		writeBalancedCapture(t, repository, "regressive.json", regressive)
		raw, qualified, err := RenderBalancedStructuralEvidence(BalancedStructuralOptions{
			RepositoryRoot: repository,
			EvidencePaths:  []string{"capture-1.json", "regressive.json"},
			CapturedAt:     time.Now(),
		})
		if err != nil || qualified {
			t.Fatalf("tail-regressive qualification = %v/%v", qualified, err)
		}
		var evidence balancedStructuralEvidence
		if err := json.Unmarshal(raw, &evidence); err != nil {
			t.Fatal(err)
		}
		if evidence.Result.CandidateP95MS <= evidence.Result.ControlP95MS || evidence.Result.Qualified {
			t.Fatalf("tail regression qualified = %+v", evidence.Result)
		}
	})
}

func mustStructuralResult(t *testing.T, observations []structuralObservation) structuralResult {
	t.Helper()
	result, err := calculateStructuralResult(observations)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writeBalancedCapture(t *testing.T, repository, name string, evidence structuralEvidence) {
	t.Helper()
	raw, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, name), append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
