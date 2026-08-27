// Command sticky-active-benchmark exercises the SWL-011 active execution
// boundary with direct synthetic commands. It never runs a customer build;
// the executable is also used as a tiny helper process so the benchmark can
// prove output equivalence, fallback and suspension without a shell.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickyactive"
	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickytrial"
)

type scenario struct {
	Name   string                 `json:"name"`
	Result stickyactive.Execution `json:"result"`
}

type benchmarkResult struct {
	SchemaVersion string                         `json:"schemaVersion"`
	RecordType    string                         `json:"recordType"`
	CapturedAt    string                         `json:"capturedAt"`
	Environment   environment                    `json:"environment"`
	TrialReport   string                         `json:"trialReport"`
	Qualification stickyactive.Qualification `json:"qualification"`
	Scenarios     []scenario                 `json:"scenarios"`
	Summary       summary                    `json:"summary"`
	Pass          bool                       `json:"pass"`
}

type environment struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

type summary struct {
	ScenarioCount             int     `json:"scenarioCount"`
	ActiveExecutions          int     `json:"activeExecutions"`
	Suspensions               int     `json:"suspensions"`
	NativeRetentions          int     `json:"nativeRetentions"`
	CandidateExecutions       int     `json:"candidateExecutions"`
	NativeExecutions          int     `json:"nativeExecutions"`
	ExactCounterfactuals      int     `json:"exactCounterfactuals"`
	PositiveSavingMs          float64 `json:"positiveSavingMs"`
	NegativeQualificationSeen bool    `json:"negativeQualificationSeen"`
}

func main() {
	if os.Getenv("STICKY_ACTIVE_HELPER") == "1" {
		runHelper()
		return
	}
	flags := flag.NewFlagSet("sticky-active-benchmark", flag.ExitOnError)
	output := flags.String("output", "", "JSON result path")
	trialReport := flags.String("trial-report", "benchmarks/results/sticky-wrapper-trial-v1.json", "checked-in SWL-010 report")
	_ = flags.Parse(os.Args[1:])
	if *output == "" || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: sticky-active-benchmark --output PATH [--trial-report PATH]")
		os.Exit(64)
	}
	result, err := run(*trialReport)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-active-benchmark: %v\n", err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-active-benchmark: marshal result: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "sticky-active-benchmark: create output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, append(raw, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "sticky-active-benchmark: write result: %v\n", err)
		os.Exit(1)
	}
	if !result.Pass {
		os.Exit(1)
	}
}

func run(trialReportPath string) (benchmarkResult, error) {
	now := time.Now().UTC().Truncate(time.Nanosecond)
	root, err := os.MkdirTemp("", "buildopt-sticky-active-")
	if err != nil {
		return benchmarkResult{}, err
	}
	defer os.RemoveAll(root)
	trialRaw, err := os.ReadFile(trialReportPath)
	if err != nil {
		return benchmarkResult{}, fmt.Errorf("read trial report: %w", err)
	}
	var trialReport stickytrial.Report
	if err := json.Unmarshal(trialRaw, &trialReport); err != nil {
		return benchmarkResult{}, fmt.Errorf("decode trial report: %w", err)
	}
	qualification := stickyactive.QualifyTrial(trialReport, 4)
	if qualification.Authorized || qualification.Reason != stickyactive.QualificationNegative {
		return benchmarkResult{}, fmt.Errorf("current SWL-010 report was unexpectedly authorized: %+v", qualification)
	}

	results := make([]scenario, 0, 8)
	positive, err := runProfileScenario(root, now, "qualified-positive", 1, 25, 20, "same", "same", false, func(profile *stickyactive.Profile) {})
	if err != nil {
		return benchmarkResult{}, err
	}
	results = append(results, scenario{Name: "qualified-positive", Result: positive})
	regression, err := runProfileScenario(root, now, "regression-suspends", 45, 1, 10, "same", "same", false, func(profile *stickyactive.Profile) {})
	if err != nil {
		return benchmarkResult{}, err
	}
	results = append(results, scenario{Name: "regression-suspends", Result: regression})
	mismatch, err := runProfileScenario(root, now, "output-mismatch", 1, 12, 20, "candidate", "native", false, func(profile *stickyactive.Profile) {})
	if err != nil {
		return benchmarkResult{}, err
	}
	results = append(results, scenario{Name: "output-mismatch", Result: mismatch})
	bypass, err := runProfileScenario(root, now, "bypass", 1, 12, 20, "candidate", "same", true, func(profile *stickyactive.Profile) {})
	if err != nil {
		return benchmarkResult{}, err
	}
	results = append(results, scenario{Name: "bypass", Result: bypass})
	for _, item := range []struct {
		name   string
		reason string
		mutate func(*stickyactive.Profile)
	}{
		{name: "binding-drift", reason: stickyactive.ReasonBindingMismatch, mutate: func(profile *stickyactive.Profile) { profile.ExpectedBinding.Workflow = "other/build" }},
		{name: "expired-decision", reason: stickyactive.ReasonExpiredDecision, mutate: func(profile *stickyactive.Profile) { profile.Now = func() time.Time { return now.Add(2 * time.Hour) } }},
		{name: "revoked-decision", reason: stickyactive.ReasonRevokedDecision, mutate: func(profile *stickyactive.Profile) { profile.RevocationEpoch = 1 }},
		{name: "candidate-failure", reason: stickyactive.ReasonCandidateFailure, mutate: func(profile *stickyactive.Profile) { profile.Candidate = helperCommand(root, "candidate-failure", "same", 1, 37) }},
	} {
		result, scenarioErr := runProfileScenario(root, now, item.name, 1, 12, 20, "same", "same", false, item.mutate)
		if scenarioErr != nil {
			return benchmarkResult{}, scenarioErr
		}
		if result.Reason != item.reason {
			return benchmarkResult{}, fmt.Errorf("scenario %s reason %s, want %s", item.name, result.Reason, item.reason)
		}
		results = append(results, scenario{Name: item.name, Result: result})
	}

	result := benchmarkResult{
		SchemaVersion: "buildopt.poc/sticky-active-result/v1", RecordType: "STICKY_WRAPPER_ACTIVE_BENCHMARK",
		CapturedAt: now.Format(time.RFC3339Nano), Environment: environment{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH},
		TrialReport: trialReportPath, Qualification: qualification, Scenarios: results,
	}
	for _, item := range results {
		switch item.Result.Status {
		case stickyactive.StatusActiveExecuted:
			result.Summary.ActiveExecutions++
		case stickyactive.StatusSuspended:
			result.Summary.Suspensions++
		case stickyactive.StatusNativeRetained:
			result.Summary.NativeRetentions++
		}
		result.Summary.ScenarioCount++
		if item.Result.CandidateExecuted {
			result.Summary.CandidateExecutions++
		}
		if item.Result.Native != nil {
			result.Summary.NativeExecutions++
		}
		if item.Result.ExactOutputs {
			result.Summary.ExactCounterfactuals++
		}
		if item.Result.SavingNs > 0 {
			result.Summary.PositiveSavingMs += float64(item.Result.SavingNs) / float64(time.Millisecond)
		}
	}
	result.Summary.NegativeQualificationSeen = !qualification.Authorized
	result.Pass = qualification.Reason == stickyactive.QualificationNegative &&
		result.Summary.ActiveExecutions == 1 && result.Summary.Suspensions == 3 &&
		result.Summary.NativeRetentions == 4 && result.Summary.ExactCounterfactuals == 2 &&
		positive.Status == stickyactive.StatusActiveExecuted && positive.SavingNs > 0 &&
		regression.Status == stickyactive.StatusSuspended && mismatch.Status == stickyactive.StatusSuspended
	return result, nil
}

func runProfileScenario(root string, now time.Time, name string, candidateDelay, nativeDelay, tolerance int, candidateValue, nativeValue string, bypass bool, mutate func(*stickyactive.Profile)) (stickyactive.Execution, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return stickyactive.Execution{}, err
	}
	scope := digest("active-benchmark-scope")
	binding := testBinding(scope)
	actionID := "benchmark/" + name
	profile := stickyactive.Profile{
		ActionID: actionID, ExpectedBinding: binding,
		PublicKeys: map[string]ed25519.PublicKey{"benchmark-owner": publicKey},
		Candidate: helperCommand(root, name+"-candidate", candidateValue, candidateDelay, 0),
		Native: helperCommand(root, name+"-native", nativeValue, nativeDelay, 0),
		RequiredOutputs: []string{"out.txt"}, CounterfactualEvery: 1,
		RegressionTolerancePermille: uint64(tolerance), Now: func() time.Time { return now },
	}
	profile.DecisionRaw = signedDecision(privateKey, binding, actionID, now)
	mutate(&profile)
	runner, err := stickyactive.New(profile)
	if err != nil {
		return stickyactive.Execution{}, err
	}
	return runner.Run(context.Background(), bypass)
}

func helperCommand(root, name, value string, delay, exitCode int) stickyactive.Command {
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		panic(err)
	}
	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return stickyactive.Command{
		Program: executable, Args: []string{"--helper"}, Dir: dir,
		Env: append(os.Environ(), "STICKY_ACTIVE_HELPER=1", "STICKY_ACTIVE_OUT="+filepath.Join(dir, "out.txt"), "STICKY_ACTIVE_VALUE="+value, "STICKY_ACTIVE_DELAY_MS="+strconv.Itoa(delay), "STICKY_ACTIVE_EXIT="+strconv.Itoa(exitCode)),
	}
}

func runHelper() {
	output := os.Getenv("STICKY_ACTIVE_OUT")
	if output == "" {
		os.Exit(2)
	}
	if delay, _ := strconv.Atoi(os.Getenv("STICKY_ACTIVE_DELAY_MS")); delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if code, _ := strconv.Atoi(os.Getenv("STICKY_ACTIVE_EXIT")); code != 0 {
		os.Exit(code)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		os.Exit(3)
	}
	if err := os.WriteFile(output, []byte(os.Getenv("STICKY_ACTIVE_VALUE")), 0o600); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}

func signedDecision(privateKey ed25519.PrivateKey, binding stickydecision.Binding, actionID string, now time.Time) []byte {
	decision := stickydecision.Decision{
		SchemaVersion: stickydecision.DecisionSchemaVersion, RecordType: stickydecision.DecisionRecordType,
		DecisionID: "benchmark-active-decision", StoreGeneration: 1, IdempotencyKey: digest("active-decision"),
		Binding: binding, ActionID: actionID, ActionGeneration: 1,
		QualificationState: "QUARANTINE_VALIDATED", RolloutState: "ACTIVE_IN_CI",
		ExecutionDecision: stickydecision.ExecutionActiveRuntime, PolicyDigest: digest("policy"),
		CacheContractDigest: digest("cache"), EvidenceRefs: []string{digest("trial")},
		IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		Authentication: stickydecision.Authentication{Algorithm: "Ed25519", KeyID: "benchmark-owner"},
	}
	raw, _, err := stickydecision.SignDecision(decision, privateKey)
	if err != nil {
		panic(err)
	}
	return raw
}

func testBinding(scope string) stickydecision.Binding {
	return stickydecision.Binding{
		RepositoryScopeSHA256: scope, Workflow: "benchmark/build", SourceRevision: strings.Repeat("a", 40),
		GradleVersion: "9.6.1", WrapperSHA256: digest("wrapper"), OptionsSHA256: digest("options"),
		OutputContractSHA256: digest("outputs"), BuildOptExecutableSHA256: digest("buildopt"),
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
