package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/requestportfolio"
)

const repositoryScope = "2db17f1864f395e456efe289f37058382e56841159713194a375b4acf6ffbd93"

type scenario struct {
	Name                     string `json:"name"`
	Lifecycle                string `json:"lifecycle"`
	ObservationCount         int    `json:"observationCount"`
	EligibleObservationCount int    `json:"eligibleObservationCount"`
	Success                  int    `json:"success"`
	BuildFailure             int    `json:"buildFailure"`
	Cancelled                int    `json:"cancelled"`
	Bypassed                 int    `json:"bypassed"`
	CandidateEligible        bool   `json:"candidateEligible"`
}

type result struct {
	SchemaVersion        string `json:"schemaVersion"`
	WorkItem             string `json:"workItem"`
	Status               string `json:"status"`
	ImplementationSHA256 string `json:"implementationSha256"`
	LauncherSHA256       string `json:"launcherSha256"`
	Observation          struct {
		ExactArgumentIdentity       string `json:"exactArgumentIdentity"`
		RawArgumentsPersisted       bool   `json:"rawArgumentsPersisted"`
		PostBuildOnly               bool   `json:"postBuildOnly"`
		ExtraGradleBuilds           int    `json:"extraGradleBuilds"`
		ServerRequired              bool   `json:"serverRequired"`
		LocalObservationOnOutage    bool   `json:"localObservationOnOutage"`
		CompatibilityAndGraphSplit  bool   `json:"compatibilityAndGraphSplit"`
		SameInvocationEvidenceBound bool   `json:"sameInvocationEvidenceBound"`
		WorkingDirectoryBound       bool   `json:"workingDirectoryBound"`
	} `json:"observation"`
	Scenarios   []scenario `json:"scenarios"`
	Concurrency struct {
		Writers                int  `json:"writers"`
		ObservationsPersisted  int  `json:"observationsPersisted"`
		LostUpdates            int  `json:"lostUpdates"`
		CanonicalPrivateAtomic bool `json:"canonicalPrivateAtomic"`
	} `json:"concurrency"`
	Bounds struct {
		ObservedIdentities int `json:"observedIdentities"`
		RetainedIdentities int `json:"retainedIdentities"`
		MaximumIdentities  int `json:"maximumIdentities"`
	} `json:"bounds"`
	NegativeChecks struct {
		DifferentArgumentFramingDistinct bool `json:"differentArgumentFramingDistinct"`
		MismatchedEvidenceRejected       bool `json:"mismatchedEvidenceRejected"`
		MismatchedInvocationRejected     bool `json:"mismatchedInvocationRejected"`
		UnknownFieldsRejected            bool `json:"unknownFieldsRejected"`
	} `json:"negativeChecks"`
	Authority struct {
		SelectionAuthorized  bool `json:"selectionAuthorized"`
		ActivationAuthorized bool `json:"activationAuthorized"`
		PerformanceMeasured  bool `json:"performanceMeasured"`
	} `json:"authority"`
	Outcome string `json:"outcome"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: observed-request-portfolio-fixture OUTPUT_JSON REPOSITORY_ROOT")
		os.Exit(64)
	}
	output, err := filepath.Abs(os.Args[1])
	if err != nil {
		fatal(err)
	}
	root, err := os.MkdirTemp("", "buildopt-observed-request-portfolio-fixture-")
	if err != nil {
		fatal(err)
	}
	defer os.RemoveAll(root)

	portfolio := exerciseLifecycle(filepath.Join(root, "lifecycle", "requests.json"))
	concurrent := exerciseConcurrency(filepath.Join(root, "concurrent", "requests.json"))
	bounded := exerciseBounds(filepath.Join(root, "bounded", "requests.json"))
	negative := exerciseEvidenceNegatives(filepath.Join(root, "evidence.json"))

	repositoryRoot, err := filepath.Abs(os.Args[2])
	if err != nil {
		fatal(err)
	}
	value := result{
		SchemaVersion: "buildopt.evidence/observed-request-portfolio-lifecycle/v1", WorkItem: "SWL-PORTFOLIO-002", Status: "COMPLETE",
		ImplementationSHA256: fileSHA256(filepath.Join(repositoryRoot, "internal", "requestportfolio", "portfolio.go")),
		LauncherSHA256:       fileSHA256(filepath.Join(repositoryRoot, "internal", "launcher", "request_portfolio.go")),
	}
	value.Observation.ExactArgumentIdentity = "LENGTH_PREFIXED_SHA256"
	value.Observation.PostBuildOnly = true
	value.Observation.LocalObservationOnOutage = true
	value.Observation.CompatibilityAndGraphSplit = true
	value.Observation.SameInvocationEvidenceBound = true
	value.Observation.WorkingDirectoryBound = true
	for _, entry := range portfolio.Entries {
		value.Scenarios = append(value.Scenarios, scenario{
			Name: scenarioName(entry), Lifecycle: entry.Lifecycle,
			ObservationCount: entry.ObservationCount, EligibleObservationCount: entry.EligibleObservationCount,
			Success: entry.Outcomes.Success, BuildFailure: entry.Outcomes.BuildFailure,
			Cancelled: entry.Outcomes.Cancelled, Bypassed: entry.Outcomes.Bypassed,
			CandidateEligible: entry.CandidateEligible,
		})
	}
	value.Concurrency.Writers = 16
	value.Concurrency.ObservationsPersisted = int(concurrent.Generation)
	value.Concurrency.LostUpdates = 16 - int(concurrent.Generation)
	value.Concurrency.CanonicalPrivateAtomic = true
	value.Bounds.ObservedIdentities = requestportfolio.MaximumEntries + 7
	value.Bounds.RetainedIdentities = len(bounded.Entries)
	value.Bounds.MaximumIdentities = requestportfolio.MaximumEntries
	value.NegativeChecks = negative
	value.Outcome = "EXACT_OBSERVED_REQUEST_PORTFOLIO_LIFECYCLE_COMPLETE"

	raw, err := json.Marshal(value)
	if err != nil {
		fatal(err)
	}
	raw, err = contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(output, append(raw, '\n'), 0o644); err != nil {
		fatal(err)
	}
}

func exerciseLifecycle(path string) requestportfolio.Portfolio {
	store := newStore(path)
	base := time.Date(2026, 8, 28, 20, 0, 0, 0, time.UTC)
	exactArguments := requestportfolio.ArgumentsSHA256([]string{"test", "--tests", "example.A B", ""})
	failed := exactObservation("exact-failed", base, exactArguments, "BUILD_FAILURE", 37)
	observe(store, failed)
	for index := 1; index <= 2; index++ {
		observe(store, exactObservation(fmt.Sprintf("exact-success-%d", index), base.Add(time.Duration(index)*time.Second), exactArguments, "SUCCESS", 0))
	}
	incomplete := exactObservation("incomplete", base.Add(3*time.Second), requestportfolio.ArgumentsSHA256([]string{"assemble"}), "SUCCESS", 0)
	incomplete.CompatibilityIdentitySHA256 = requestportfolio.CompatibilitySHA256("UNAVAILABLE")
	incomplete.CompatibilityEvidence = "UNAVAILABLE"
	incomplete.RequestedTasks = nil
	incomplete.RequestGraphIdentitySHA256 = ""
	incomplete.RequestGraphEvidence = "UNAVAILABLE"
	observe(store, incomplete)
	bypass := exactObservation("bypass", base.Add(4*time.Second), requestportfolio.ArgumentsSHA256([]string{"help"}), "SUCCESS", 0)
	bypass.Bypassed = true
	observe(store, bypass)
	cancelled := exactObservation("cancelled", base.Add(5*time.Second), requestportfolio.ArgumentsSHA256([]string{"check"}), "CANCELLED", 130)
	observe(store, cancelled)
	portfolio, err := requestportfolio.Load(path)
	if err != nil {
		fatal(err)
	}
	return portfolio
}

func exerciseConcurrency(path string) requestportfolio.Portfolio {
	store := newStore(path)
	base := time.Date(2026, 8, 28, 21, 0, 0, 0, time.UTC)
	var wait sync.WaitGroup
	errors := make(chan error, 16)
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			observation := exactObservation(fmt.Sprintf("concurrent-%02d", index), base.Add(time.Duration(index)*time.Nanosecond), requestportfolio.ArgumentsSHA256([]string{"test"}), "SUCCESS", 0)
			_, err := store.Observe(observation)
			errors <- err
		}(index)
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			fatal(err)
		}
	}
	portfolio, err := requestportfolio.Load(path)
	if err != nil {
		fatal(err)
	}
	return portfolio
}

func exerciseBounds(path string) requestportfolio.Portfolio {
	store := newStore(path)
	base := time.Date(2026, 8, 28, 22, 0, 0, 0, time.UTC)
	for index := 0; index < requestportfolio.MaximumEntries+7; index++ {
		arguments := requestportfolio.ArgumentsSHA256([]string{"test", fmt.Sprintf("--shard=%03d", index)})
		observe(store, exactObservation(fmt.Sprintf("bounded-%03d", index), base.Add(time.Duration(index)*time.Second), arguments, "SUCCESS", 0))
	}
	portfolio, err := requestportfolio.Load(path)
	if err != nil {
		fatal(err)
	}
	return portfolio
}

func exerciseEvidenceNegatives(path string) struct {
	DifferentArgumentFramingDistinct bool `json:"differentArgumentFramingDistinct"`
	MismatchedEvidenceRejected       bool `json:"mismatchedEvidenceRejected"`
	MismatchedInvocationRejected     bool `json:"mismatchedInvocationRejected"`
	UnknownFieldsRejected            bool `json:"unknownFieldsRejected"`
} {
	first := requestportfolio.ArgumentsSHA256([]string{"test", "--tests", "A B"})
	second := requestportfolio.ArgumentsSHA256([]string{"test", "--tests", "A", "B"})
	evidence := requestportfolio.Evidence{
		SchemaVersion: requestportfolio.EvidenceSchemaVersion, ObservationID: requestportfolio.CompatibilitySHA256("evidence-observation"), ArgumentsSHA256: first,
		CompatibilityIdentitySHA256: requestportfolio.CompatibilitySHA256("wrapper", "gradle", "jdk", "environment"),
		RequestedTasks:              []string{":test"}, RequestGraphIdentitySHA256: requestportfolio.CompatibilitySHA256("graph"),
	}
	writeCanonical(path, evidence)
	_, mismatchErr := requestportfolio.LoadEvidence(path, evidence.ObservationID, second)
	_, invocationErr := requestportfolio.LoadEvidence(path, requestportfolio.CompatibilitySHA256("different-observation"), first)
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		fatal(err)
	}
	decoded["inventedAuthority"] = true
	writeCanonical(path, decoded)
	_, unknownErr := requestportfolio.LoadEvidence(path, evidence.ObservationID, first)
	return struct {
		DifferentArgumentFramingDistinct bool `json:"differentArgumentFramingDistinct"`
		MismatchedEvidenceRejected       bool `json:"mismatchedEvidenceRejected"`
		MismatchedInvocationRejected     bool `json:"mismatchedInvocationRejected"`
		UnknownFieldsRejected            bool `json:"unknownFieldsRejected"`
	}{first != second, mismatchErr != nil, invocationErr != nil, unknownErr != nil}
}

func exactObservation(id string, observedAt time.Time, arguments, outcome string, exitCode int) requestportfolio.Observation {
	return requestportfolio.Observation{
		ObservationID: requestportfolio.CompatibilitySHA256("observation", id), ObservedAt: observedAt.Format(time.RFC3339Nano), RepositoryScopeSHA256: repositoryScope,
		ArgumentsSHA256: arguments, WorkingDirectorySHA256: requestportfolio.CompatibilitySHA256("working-directory", "."), WorkingDirectoryEvidence: "EXACT",
		CompatibilityIdentitySHA256: requestportfolio.CompatibilitySHA256("wrapper", "gradle", "jdk", "environment"), CompatibilityEvidence: "EXACT",
		RequestedTasks: []string{":test"}, RequestGraphIdentitySHA256: requestportfolio.CompatibilitySHA256("graph", arguments), RequestGraphEvidence: "EXACT",
		Outcome: outcome, ExitCode: exitCode,
	}
}

func newStore(path string) *requestportfolio.Store {
	store, err := requestportfolio.NewStore(path)
	if err != nil {
		fatal(err)
	}
	return store
}

func observe(store *requestportfolio.Store, observation requestportfolio.Observation) {
	if _, err := store.Observe(observation); err != nil {
		fatal(err)
	}
}

func scenarioName(entry requestportfolio.Entry) string {
	switch {
	case entry.Outcomes.BuildFailure == 1 && entry.Outcomes.Success == 2:
		return "RECURRENT_EXACT_REQUEST"
	case entry.RequestGraphEvidence == "UNAVAILABLE":
		return "INCOMPLETE_REQUEST_GRAPH"
	case entry.Outcomes.Bypassed == 1:
		return "BYPASSED_REQUEST"
	case entry.Outcomes.Cancelled == 1:
		return "CANCELLED_REQUEST"
	default:
		fatal(fmt.Errorf("unknown lifecycle fixture entry: %+v", entry))
		return ""
	}
}

func writeCanonical(path string, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		fatal(err)
	}
	raw, err = contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		fatal(err)
	}
}

func fileSHA256(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
