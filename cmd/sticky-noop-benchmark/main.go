// Command sticky-noop-benchmark measures the pre-Gradle local decision path.
// It creates only synthetic, signed state in a temporary directory and never
// contacts the central service or executes a Gradle build.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickydecision"
)

const (
	maxP50 = int64(100 * time.Millisecond)
	maxP95 = int64(250 * time.Millisecond)
	maxUnavailableP95 = int64(500 * time.Millisecond)
)

type caseResult struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	P50Ns  int64  `json:"p50Ns"`
	P95Ns  int64  `json:"p95Ns"`
	MaxNs  int64  `json:"maxNs"`
	Reason string `json:"reason"`
}

type benchmarkResult struct {
	SchemaVersion string `json:"schemaVersion"`
	CapturedAt   string `json:"capturedAt"`
	Iterations   int    `json:"iterations"`
	Environment  struct {
		GOOS string `json:"goos"`
		GOARCH string `json:"goarch"`
	} `json:"environment"`
	Cases struct {
		VerifiedLocal      caseResult `json:"verifiedLocal"`
		MissingLocal       caseResult `json:"missingLocal"`
		ServiceUnavailable caseResult `json:"serviceUnavailable"`
	} `json:"cases"`
	Requirements struct {
		LocalP50NsMax         int64 `json:"localP50NsMax"`
		LocalP95NsMax         int64 `json:"localP95NsMax"`
		UnavailableP95NsMax   int64 `json:"unavailableP95NsMax"`
	} `json:"requirements"`
	Pass bool `json:"pass"`
}

func main() {
	flags := flag.NewFlagSet("sticky-noop-benchmark", flag.ExitOnError)
	output := flags.String("output", "", "JSON result path")
	iterations := flags.Int("iterations", 200, "number of measured selections per case")
	_ = flags.Parse(os.Args[1:])
	if *output == "" || *iterations < 20 || *iterations > 10000 || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: sticky-noop-benchmark --output PATH [--iterations N]")
		os.Exit(64)
	}
	result, err := run(*iterations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-noop-benchmark: %v\n", err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-noop-benchmark: marshal result: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "sticky-noop-benchmark: create output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, append(raw, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "sticky-noop-benchmark: write result: %v\n", err)
		os.Exit(1)
	}
	if !result.Pass {
		os.Exit(1)
	}
}

func run(iterations int) (benchmarkResult, error) {
	now := time.Now().UTC().Truncate(time.Nanosecond)
	scope := digest("sticky-noop-scope")
	binding := testBinding(scope)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return benchmarkResult{}, err
	}
	keys := map[string]ed25519.PublicKey{"benchmark-owner": publicKey}
	root, err := os.MkdirTemp("", "buildopt-sticky-noop-")
	if err != nil {
		return benchmarkResult{}, err
	}
	defer os.RemoveAll(root)
	store, err := stickydecision.OpenLocalWithOptions(root, scope, stickydecision.StoreOptions{
		PublicKeys: keys, Now: func() time.Time { return now },
	})
	if err != nil {
		return benchmarkResult{}, err
	}
	decision := stickydecision.Decision{
		SchemaVersion: stickydecision.DecisionSchemaVersion, RecordType: stickydecision.DecisionRecordType,
		DecisionID: "benchmark-native-noop", StoreGeneration: 1, IdempotencyKey: digest("benchmark-native-noop"),
		Binding: binding, QualificationState: "OBSERVING", RolloutState: "PROPOSED",
		ExecutionDecision: stickydecision.ExecutionNativeNoop, PolicyDigest: digest("policy"),
		CacheContractDigest: digest("cache-contract"), EvidenceRefs: []string{},
		IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano),
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		Authentication: stickydecision.Authentication{Algorithm: "Ed25519", KeyID: "benchmark-owner"},
	}
	raw, _, err := stickydecision.SignDecision(decision, privateKey)
	if err != nil {
		return benchmarkResult{}, err
	}
	if _, err := store.Append(context.Background(), raw, 0, "", decision.IdempotencyKey); err != nil {
		return benchmarkResult{}, err
	}
	selectLocal := func(path string, options stickydecision.SelectorOptions) int64 {
		started := time.Now()
		selection := stickydecision.SelectLocal(context.Background(), path, scope, binding, options)
		if selection.Status != stickydecision.SelectionNative {
			return -1
		}
		return time.Since(started).Nanoseconds()
	}
	verified := measure(iterations, func() int64 {
		return selectLocal(root, stickydecision.SelectorOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	})
	missingRoot := filepath.Join(root, "missing")
	missing := measure(iterations, func() int64 {
		return selectLocal(missingRoot, stickydecision.SelectorOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	})
	// An unavailable service is deliberately represented by the same local
	// native fallback. No synchronous refresh is attempted, so a remote outage
	// cannot enter the Gradle critical path.
	unavailable := measure(iterations, func() int64 {
		return selectLocal(missingRoot, stickydecision.SelectorOptions{PublicKeys: keys, Now: func() time.Time { return now }})
	})
	if verified.invalid || missing.invalid || unavailable.invalid {
		return benchmarkResult{}, errors.New("selector returned an unexpected active or invalid result")
	}
	result := benchmarkResult{SchemaVersion: "buildopt.poc/sticky-noop-result/v1", CapturedAt: now.Format(time.RFC3339Nano), Iterations: iterations}
	result.Environment.GOOS, result.Environment.GOARCH = runtime.GOOS, runtime.GOARCH
	result.Requirements.LocalP50NsMax, result.Requirements.LocalP95NsMax = maxP50, maxP95
	result.Requirements.UnavailableP95NsMax = maxUnavailableP95
	result.Cases.VerifiedLocal = summarize(verified.values, "VERIFIED_NATIVE_NOOP")
	result.Cases.MissingLocal = summarize(missing.values, "NO_LOCAL_SNAPSHOT")
	result.Cases.ServiceUnavailable = summarize(unavailable.values, "NO_SYNCHRONOUS_REFRESH")
	result.Pass = result.Cases.VerifiedLocal.P50Ns <= maxP50 &&
		result.Cases.VerifiedLocal.P95Ns <= maxP95 &&
		result.Cases.ServiceUnavailable.P95Ns <= maxUnavailableP95
	for _, value := range []caseResult{result.Cases.VerifiedLocal, result.Cases.MissingLocal, result.Cases.ServiceUnavailable} {
		if value.Count != iterations {
			result.Pass = false
		}
	}
	return result, nil
}

type measurements struct {
	values  []int64
	invalid bool
}

func measure(iterations int, sample func() int64) measurements {
	for index := 0; index < 10; index++ {
		_ = sample()
	}
	values := make([]int64, 0, iterations)
	for index := 0; index < iterations; index++ {
		value := sample()
		if value < 0 {
			return measurements{invalid: true}
		}
		values = append(values, value)
	}
	return measurements{values: values}
}

func summarize(values []int64, reason string) caseResult {
	if len(values) == 0 {
		return caseResult{Status: "FAIL", Reason: reason}
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	return caseResult{
		Status: "PASS", Count: len(values), P50Ns: percentile(sorted, 50),
		P95Ns: percentile(sorted, 95), MaxNs: sorted[len(sorted)-1], Reason: reason,
	}
}

func percentile(sorted []int64, percentile int) int64 {
	index := (len(sorted)*percentile + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func testBinding(scope string) stickydecision.Binding {
	return stickydecision.Binding{
		RepositoryScopeSHA256: scope, Workflow: "benchmark/build",
		SourceRevision: strings.Repeat("a", 40), GradleVersion: "9.6.1",
		WrapperSHA256: digest("wrapper"), OptionsSHA256: digest("options"),
		OutputContractSHA256: digest("outputs"), BuildOptExecutableSHA256: digest("buildopt"),
	}
}
