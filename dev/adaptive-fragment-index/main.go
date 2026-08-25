// Command adaptive-fragment-index produces and validates the bounded AF-003
// pre-Gradle lookup benchmark. It performs no process execution or networking.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"time"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-lookup-benchmark/v1"
	outcomeFast  = "FAST_FRAGMENT_LOOKUP_AVAILABLE"
)

type subject struct {
	RepositoryID string
	Revision     string
}

var subjects = []subject{
	{RepositoryID: "spring-projects/spring-framework", Revision: "1c4b20287f1d8f635867fb58c576849f9313e907"},
	{RepositoryID: "open-telemetry/opentelemetry-java-instrumentation", Revision: "f330e4c0b764d80dcc1f7c6d4c01e305593c49f1"},
	{RepositoryID: "apache/kafka", Revision: "2e961afeff5cb27d60767a783edf20be00cc28e8"},
	{RepositoryID: "micronaut-projects/micronaut-core", Revision: "eb60c6c35f355750c6bced793e85c30629d27c4e"},
	{RepositoryID: "apache/groovy", Revision: "14fa934340c46bc1ee96cba7ae594258bd8d0fd0"},
}

type report struct {
	SchemaVersion string           `json:"schemaVersion"`
	WorkItem      string           `json:"workItem"`
	CapturedAt    string           `json:"capturedAt"`
	Runner        runner           `json:"runner"`
	Subjects      []subjectResult  `json:"subjects"`
	Decisions     []decisionResult `json:"decisions"`
	Summary       summary          `json:"summary"`
	SideEffects   sideEffects      `json:"sideEffects"`
	Boundaries    boundaries       `json:"boundaries"`
	Outcome       string           `json:"outcome"`
}

type runner struct {
	GOOS     string `json:"goos"`
	GOARCH   string `json:"goarch"`
	CPUCount int    `json:"cpuCount"`
}

type subjectResult struct {
	RepositoryID         string `json:"repositoryId"`
	Revision             string `json:"revision"`
	GitFingerprintSHA256 string `json:"gitFingerprintSha256"`
	DecisionCount        int    `json:"decisionCount"`
}

type decisionResult struct {
	RepositoryID string                             `json:"repositoryId"`
	Scenario     string                             `json:"scenario"`
	Disposition  adaptivefragment.LookupDisposition `json:"disposition"`
	Reason       string                             `json:"reason"`
	DurationNS   int64                              `json:"durationNs"`
}

type summary struct {
	RepositoryCount   int            `json:"repositoryCount"`
	DecisionCount     int            `json:"decisionCount"`
	DispositionCounts map[string]int `json:"dispositionCounts"`
	MedianMS          float64        `json:"medianMs"`
	P95MS             float64        `json:"p95Ms"`
	MaximumMS         float64        `json:"maximumMs"`
}

type sideEffects struct {
	GradleStarts           int `json:"gradleStarts"`
	RemoteCalls            int `json:"remoteCalls"`
	OutputMaterializations int `json:"outputMaterializations"`
	LifecycleMutations     int `json:"lifecycleMutations"`
}

type boundaries struct {
	ProofOfConcept       bool   `json:"proofOfConcept"`
	BuildTimingClaim     bool   `json:"buildTimingClaim"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	TestOptimization     string `json:"testOptimization"`
}

type scenario struct {
	name        string
	mutate      func(adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot
	disposition adaptivefragment.LookupDisposition
	reason      string
}

func main() {
	output := flag.String("output", "", "write a new AF-003 benchmark report")
	validate := flag.String("validate", "", "validate an existing AF-003 report")
	flag.Parse()
	if flag.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-index (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	if *validate != "" {
		candidate, err := readReport(*validate)
		if err == nil {
			err = validateReport(candidate)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "adaptive fragment lookup report is invalid: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("adaptive fragment lookup report: FAST_FRAGMENT_LOOKUP_AVAILABLE")
		return
	}
	candidate, err := runBenchmark()
	if err == nil {
		err = validateReport(candidate)
	}
	if err == nil {
		err = writeReport(*output, candidate)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment lookup benchmark failed: %v\n", err)
		os.Exit(1)
	}
}

func runBenchmark() (report, error) {
	candidate := report{
		SchemaVersion: reportSchema,
		WorkItem:      "AF-003",
		CapturedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Runner:        runner{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUCount: runtime.NumCPU()},
		SideEffects:   sideEffects{},
		Boundaries: boundaries{
			ProofOfConcept: true, BuildTimingClaim: false,
			ActivationAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
		},
		Outcome: outcomeFast,
	}

	for _, current := range subjects {
		snapshot := benchmarkSnapshot(current)
		fragment, err := benchmarkFragment(snapshot)
		if err != nil {
			return report{}, err
		}
		index, err := adaptivefragment.NewCompatibilityIndex(snapshot, 1, []adaptivefragment.PersistedFragment{fragment})
		if err != nil {
			return report{}, fmt.Errorf("build %s index: %w", current.RepositoryID, err)
		}
		cases := benchmarkScenarios()
		candidate.Subjects = append(candidate.Subjects, subjectResult{
			RepositoryID: current.RepositoryID, Revision: current.Revision,
			GitFingerprintSHA256: snapshot.GitRevisionSHA256, DecisionCount: len(cases),
		})
		for _, test := range cases {
			started := time.Now()
			result := adaptivefragment.Lookup(index, test.mutate(snapshot))
			duration := time.Since(started)
			if result.Disposition != test.disposition || result.Reason != test.reason {
				return report{}, fmt.Errorf("%s/%s = %s/%s, want %s/%s", current.RepositoryID,
					test.name, result.Disposition, result.Reason, test.disposition, test.reason)
			}
			candidate.Decisions = append(candidate.Decisions, decisionResult{
				RepositoryID: current.RepositoryID, Scenario: test.name,
				Disposition: result.Disposition, Reason: result.Reason, DurationNS: duration.Nanoseconds(),
			})
		}
	}
	candidate.Summary = summarize(candidate.Decisions)
	return candidate, nil
}

func benchmarkScenarios() []scenario {
	return []scenario{
		{name: "exact-bindings", mutate: unchanged, disposition: adaptivefragment.DispositionCompatible, reason: "COMPATIBLE_FRAGMENT_CANDIDATES"},
		{name: "unrelated-platform-drift", mutate: func(value adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot {
			value.PlatformSHA256 = stableDigest("changed-platform")
			return value
		}, disposition: adaptivefragment.DispositionCompatible, reason: "COMPATIBLE_FRAGMENT_CANDIDATES"},
		{name: "declared-wrapper-drift", mutate: func(value adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot {
			value.WrapperSHA256 = stableDigest("changed-wrapper")
			return value
		}, disposition: adaptivefragment.DispositionSuspended, reason: "DECLARED_BINDING_DRIFT"},
		{name: "ambiguous-producer-lineage", mutate: func(value adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot {
			value.Ambiguous = []adaptivefragment.BindingKey{adaptivefragment.BindingProducerLineage}
			return value
		}, disposition: adaptivefragment.DispositionNativeRetained, reason: "NO_COMPATIBLE_FRAGMENT"},
		{name: "cross-repository-scope", mutate: func(value adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot {
			value.RepositoryID += "-other"
			return value
		}, disposition: adaptivefragment.DispositionNativeRetained, reason: "REPOSITORY_SCOPE_MISMATCH"},
		{name: "missing-output-fingerprint", mutate: func(value adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot {
			value.OutputContractSHA256 = ""
			return value
		}, disposition: adaptivefragment.DispositionNativeRetained, reason: "NO_COMPATIBLE_FRAGMENT"},
	}
}

func unchanged(value adaptivefragment.FingerprintSnapshot) adaptivefragment.FingerprintSnapshot {
	return value
}

func benchmarkSnapshot(current subject) adaptivefragment.FingerprintSnapshot {
	return adaptivefragment.FingerprintSnapshot{
		RepositoryID: current.RepositoryID, GitRevisionSHA256: stableDigest("git:" + current.Revision),
		WrapperSHA256: stableDigest("wrapper:gradle-9.6.1"), WorkflowSHA256: stableDigest("workflow:build"),
		ProducerLineageSHA256: stableDigest("producer:declared-graph"), OutputContractSHA256: stableDigest("output:required"),
		ChangeFamilySHA256: stableDigest("change:dependency-source"), PlatformSHA256: stableDigest("platform:linux-amd64"),
		ObservedAt: "2026-08-25T12:00:00Z",
	}
}

func benchmarkFragment(snapshot adaptivefragment.FingerprintSnapshot) (adaptivefragment.PersistedFragment, error) {
	fragment, err := adaptivefragment.Derive(adaptivefragment.Input{
		RepositoryID: snapshot.RepositoryID, Kind: adaptivefragment.KindSubgraph,
		Selector: []string{"declared-build-subgraph"}, Authority: adaptivefragment.AuthorityGradleModel,
		AuthoritySHA256: stableDigest("authority:gradle-model"),
		Bindings: map[adaptivefragment.BindingKey]string{
			adaptivefragment.BindingWorkflow: snapshot.WorkflowSHA256, adaptivefragment.BindingWrapper: snapshot.WrapperSHA256,
			adaptivefragment.BindingProducerLineage: snapshot.ProducerLineageSHA256,
			adaptivefragment.BindingOutputContract:  snapshot.OutputContractSHA256,
			adaptivefragment.BindingChangeFamily:    snapshot.ChangeFamilySHA256,
		},
	})
	if err != nil {
		return adaptivefragment.PersistedFragment{}, err
	}
	return adaptivefragment.PersistedFragment{
		SchemaVersion: adaptivefragment.FragmentStateSchemaVersion, RecordType: "ADAPTIVE_FRAGMENT",
		FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID, RepositoryScopeSHA256: fragment.RepositoryScopeSHA256,
		Kind: fragment.Kind, SelectorSHA256: fragment.SelectorSHA256, Authority: fragment.Authority,
		AuthoritySHA256: fragment.AuthoritySHA256, Bindings: fragment.Bindings, Requires: fragment.Requires,
		ConflictsWith: fragment.ConflictsWith, State: adaptivefragment.StateActive, Generation: 4,
		CreatedAt: "2026-08-25T10:00:00Z", UpdatedAt: "2026-08-25T10:03:00Z", EvidenceExpiresAt: "2026-09-25T10:03:00Z",
	}, nil
}

func summarize(decisions []decisionResult) summary {
	durations := make([]int64, len(decisions))
	counts := map[string]int{}
	for index, decision := range decisions {
		durations[index] = decision.DurationNS
		counts[string(decision.Disposition)]++
	}
	sort.Slice(durations, func(left, right int) bool { return durations[left] < durations[right] })
	return summary{
		RepositoryCount: len(subjects), DecisionCount: len(decisions), DispositionCounts: counts,
		MedianMS: durationMS(nearestRank(durations, 0.50)), P95MS: durationMS(nearestRank(durations, 0.95)),
		MaximumMS: durationMS(durations[len(durations)-1]),
	}
}

func nearestRank(values []int64, percentile float64) int64 {
	index := int(math.Ceil(percentile*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	return values[index]
}

func durationMS(nanoseconds int64) float64 { return float64(nanoseconds) / float64(time.Millisecond) }

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-003" || candidate.Outcome != outcomeFast {
		return errors.New("report identity is invalid")
	}
	if capturedAt, err := time.Parse(time.RFC3339Nano, candidate.CapturedAt); err != nil ||
		candidate.CapturedAt == "" || capturedAt.Location() != time.UTC || candidate.Runner.GOOS == "" ||
		candidate.Runner.GOARCH == "" || candidate.Runner.CPUCount <= 0 {
		return errors.New("report runner or capture time is invalid")
	}
	if len(candidate.Subjects) != 5 || len(candidate.Decisions) < 30 || candidate.Summary.RepositoryCount != 5 ||
		candidate.Summary.DecisionCount != len(candidate.Decisions) {
		return errors.New("report breadth is insufficient")
	}
	seen := map[string]bool{}
	for index, current := range candidate.Subjects {
		expected := subjects[index]
		if current.RepositoryID != expected.RepositoryID || current.Revision != expected.Revision ||
			current.GitFingerprintSHA256 != stableDigest("git:"+expected.Revision) ||
			!isSHA(current.GitFingerprintSHA256) || current.DecisionCount != 6 || seen[current.RepositoryID] {
			return errors.New("report subject is invalid")
		}
		seen[current.RepositoryID] = true
	}
	expectedScenarios := map[string]scenario{}
	for _, expected := range benchmarkScenarios() {
		expectedScenarios[expected.name] = expected
	}
	seenDecisions := map[string]bool{}
	for _, decision := range candidate.Decisions {
		expected, exists := expectedScenarios[decision.Scenario]
		key := decision.RepositoryID + "\x00" + decision.Scenario
		if !seen[decision.RepositoryID] || !exists || decision.DurationNS < 0 || seenDecisions[key] ||
			decision.Disposition != expected.disposition || decision.Reason != expected.reason {
			return errors.New("report decision is invalid")
		}
		seenDecisions[key] = true
	}
	if len(seenDecisions) != len(subjects)*len(expectedScenarios) {
		return errors.New("report decision matrix is incomplete")
	}
	expectedSummary := summarize(candidate.Decisions)
	if !reflect.DeepEqual(candidate.Summary, expectedSummary) {
		return errors.New("report summary is not recomputable")
	}
	if candidate.Summary.MedianMS >= 500 || candidate.Summary.P95MS >= 1000 || candidate.Summary.MaximumMS >= 1000 {
		return errors.New("report latency exceeds the AF-003 budget")
	}
	if candidate.SideEffects != (sideEffects{}) || candidate.Boundaries.BuildTimingClaim ||
		candidate.Boundaries.ActivationAuthorized || !candidate.Boundaries.ProofOfConcept ||
		candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("report POC boundary is invalid")
	}
	return nil
}

func readReport(path string) (report, error) {
	document, err := os.ReadFile(path)
	if err != nil {
		return report{}, err
	}
	var candidate report
	if err := json.Unmarshal(document, &candidate); err != nil {
		return report{}, err
	}
	return candidate, nil
}

func writeReport(path string, candidate report) error {
	document, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".adaptive-fragment-index-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(document, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func stableDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func isSHA(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
