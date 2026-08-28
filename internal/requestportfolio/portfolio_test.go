package requestportfolio

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStoreTracksExactLifecycleAndTypedOutcomes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "portfolio.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 28, 15, 0, 0, 0, time.UTC)
	incomplete := validObservation(base)
	incomplete.CompatibilityEvidence = "UNAVAILABLE"
	incomplete.RequestGraphEvidence = "UNAVAILABLE"
	incomplete.RequestGraphIdentitySHA256 = ""
	incomplete.RequestedTasks = nil
	portfolio, err := store.Observe(incomplete)
	if err != nil {
		t.Fatal(err)
	}
	if len(portfolio.Entries) != 1 || portfolio.Entries[0].Lifecycle != "OBSERVED_INCOMPLETE" || portfolio.Entries[0].CandidateEligible {
		t.Fatalf("incomplete portfolio = %+v", portfolio)
	}

	failed := validObservation(base.Add(time.Second))
	failed.ObservationID = digest("failed")
	failed.Outcome = "BUILD_FAILURE"
	failed.ExitCode = 37
	portfolio, err = store.Observe(failed)
	if err != nil {
		t.Fatal(err)
	}
	if len(portfolio.Entries) != 2 || countLifecycle(portfolio, "INELIGIBLE_OUTCOME") != 1 {
		t.Fatalf("failed portfolio = %+v", portfolio)
	}

	success := validObservation(base.Add(2 * time.Second))
	success.ObservationID = digest("success")
	portfolio, err = store.Observe(success)
	if err != nil {
		t.Fatal(err)
	}
	complete := entryByLifecycle(t, portfolio, "EVIDENCE_COMPLETE")
	if complete.ObservationCount != 2 || complete.EligibleObservationCount != 1 || complete.Outcomes.BuildFailure != 1 || complete.Outcomes.Success != 1 {
		t.Fatalf("complete entry = %+v", complete)
	}
	generation := portfolio.Generation
	portfolio, err = store.Observe(success)
	if err != nil || portfolio.Generation != generation {
		t.Fatalf("idempotent observation = generation %d/%v", portfolio.Generation, err)
	}
	loaded, err := Load(path)
	if err != nil || loaded.Generation != generation || loaded.SelectionAuthorized || loaded.ActivationAuthorized || loaded.PerformanceMeasured {
		t.Fatalf("loaded portfolio = %+v/%v", loaded, err)
	}
}

func TestStoreBoundsPortfolioAndRejectsTamper(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "portfolio.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 28, 16, 0, 0, 0, time.UTC)
	for index := 0; index < MaximumEntries+7; index++ {
		observation := validObservation(base.Add(time.Duration(index) * time.Second))
		observation.ObservationID = digest("observation", string(rune(index+1000)))
		observation.ArgumentsSHA256 = digest("args", string(rune(index+1000)))
		if _, err := store.Observe(observation); err != nil {
			t.Fatal(err)
		}
	}
	portfolio, err := Load(path)
	if err != nil || len(portfolio.Entries) != MaximumEntries || portfolio.Generation != MaximumEntries+7 {
		t.Fatalf("bounded portfolio = %d/%d/%v", len(portfolio.Entries), portfolio.Generation, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"observationCount":1`, `"observationCount":2`, 1))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("tampered portfolio was accepted")
	}
}

func TestEvidenceBindsExactArgumentVector(t *testing.T) {
	arguments := []string{"./gradlew", "test", "--tests", "example.A B", ""}
	other := []string{"./gradlew", "test", "--tests", "example.A", "B"}
	if ArgumentsSHA256(arguments) == ArgumentsSHA256(other) {
		t.Fatal("argument framing collided")
	}
	path := filepath.Join(t.TempDir(), "evidence.json")
	evidence := Evidence{
		SchemaVersion: EvidenceSchemaVersion, ObservationID: digest("evidence-observation"), ArgumentsSHA256: ArgumentsSHA256(arguments),
		CompatibilityIdentitySHA256: digest("compatibility"), RequestedTasks: []string{":test"},
		RequestGraphIdentitySHA256: digest("graph"),
	}
	if err := writeCanonicalAtomic(path, evidence); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadEvidence(path, evidence.ObservationID, evidence.ArgumentsSHA256)
	if err != nil || loaded.RequestGraphIdentitySHA256 != evidence.RequestGraphIdentitySHA256 {
		t.Fatalf("evidence = %+v/%v", loaded, err)
	}
	if _, err := LoadEvidence(path, evidence.ObservationID, ArgumentsSHA256(other)); err == nil {
		t.Fatal("evidence attached to different argv")
	}
	if _, err := LoadEvidence(path, digest("different-observation"), evidence.ArgumentsSHA256); err == nil {
		t.Fatal("evidence attached to different invocation")
	}
}

func TestStoreSerializesConcurrentObservations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "portfolio.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)
	const observations = 16
	var wait sync.WaitGroup
	errors := make(chan error, observations)
	for index := 0; index < observations; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			observation := validObservation(base.Add(time.Duration(index) * time.Nanosecond))
			observation.ObservationID = digest("concurrent-observation", string(rune(index)))
			_, observeErr := store.Observe(observation)
			errors <- observeErr
		}(index)
	}
	wait.Wait()
	close(errors)
	for observeErr := range errors {
		if observeErr != nil {
			t.Fatal(observeErr)
		}
	}
	portfolio, err := Load(path)
	if err != nil || portfolio.Generation != observations || len(portfolio.Entries) != 1 {
		t.Fatalf("concurrent portfolio = %+v/%v", portfolio, err)
	}
	entry := portfolio.Entries[0]
	if entry.ObservationCount != observations || entry.EligibleObservationCount != observations || entry.Outcomes.Success != observations {
		t.Fatalf("concurrent entry = %+v", entry)
	}
	if entry.FirstObservedAt != base.Format(time.RFC3339Nano) || entry.LastObservedAt != base.Add(15*time.Nanosecond).Format(time.RFC3339Nano) {
		t.Fatalf("concurrent bounds = %s..%s", entry.FirstObservedAt, entry.LastObservedAt)
	}
}

func TestUnavailableWorkingDirectoryCannotQualify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "portfolio.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	observation := validObservation(time.Date(2026, 8, 28, 19, 0, 0, 0, time.UTC))
	observation.WorkingDirectorySHA256 = digest("unavailable-working-directory")
	observation.WorkingDirectoryEvidence = "UNAVAILABLE"
	portfolio, err := store.Observe(observation)
	if err != nil {
		t.Fatal(err)
	}
	entry := portfolio.Entries[0]
	if entry.CandidateEligible || entry.Lifecycle != "OBSERVED_INCOMPLETE" || entry.EligibleObservationCount != 0 {
		t.Fatalf("unavailable working directory entry = %+v", entry)
	}
}

func TestLoadRejectsSymlinkedPortfolio(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "private", "portfolio.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Observe(validObservation(time.Date(2026, 8, 28, 20, 0, 0, 0, time.UTC))); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "portfolio-link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlinked portfolio was accepted")
	}
}

func validObservation(at time.Time) Observation {
	return Observation{
		ObservationID: digest("observation"), ObservedAt: at.Format(time.RFC3339Nano),
		RepositoryScopeSHA256: digest("scope"), ArgumentsSHA256: digest("args"),
		WorkingDirectorySHA256: digest("working-directory"), WorkingDirectoryEvidence: "EXACT",
		CompatibilityIdentitySHA256: digest("compatibility"), CompatibilityEvidence: "EXACT",
		RequestedTasks: []string{":test"}, RequestGraphIdentitySHA256: digest("graph"), RequestGraphEvidence: "EXACT",
		Outcome: "SUCCESS", ExitCode: 0,
	}
}

func countLifecycle(portfolio Portfolio, lifecycle string) int {
	count := 0
	for _, entry := range portfolio.Entries {
		if entry.Lifecycle == lifecycle {
			count++
		}
	}
	return count
}

func entryByLifecycle(t *testing.T, portfolio Portfolio, lifecycle string) Entry {
	t.Helper()
	for _, entry := range portfolio.Entries {
		if entry.Lifecycle == lifecycle {
			return entry
		}
	}
	t.Fatalf("missing lifecycle %s in %+v", lifecycle, portfolio)
	return Entry{}
}
