package launcher

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
	"github.com/tonyredondo/buildopt/internal/stickyactive"
	"github.com/tonyredondo/buildopt/internal/stickydecision"
	"github.com/tonyredondo/buildopt/internal/stickyobservation"
	"github.com/tonyredondo/buildopt/internal/stickytrial"
	"github.com/tonyredondo/buildopt/internal/stickyvalue"
)

func TestPrepareStickyLearningEntryKeepsLearningOnComposedPath(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	clearStickyNativeIntegrationEnvironment(t)
	child := []string{filepath.Join(root, gradleWrapperName(runtime.GOOS)), "help"}

	entry := prepareStickyLearningEntry(root, child, os.Getenv)
	if !entry.nativeOnly {
		t.Fatal("ordinary invocation did not retain native fast path")
	}
	t.Setenv(stickyLearningEnvironment, "1")
	entry = prepareStickyLearningEntry(root, child, os.Getenv)
	if entry.nativeOnly {
		t.Fatal("trusted learning request bypassed composition root")
	}
}

func TestRunStickyLearningComposesBeforeCustomerGradle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX Gradle Wrapper-shaped script")
	}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	storage, err := sharedcache.Open(ctx, filepath.Join(t.TempDir(), "server"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	issued := issueStickyConnectionToken(t, storage, now, "example/learning", "gradle-9.6.1/linux-amd64/jdk-21/learning", []sharedcache.CentralCapability{
		sharedcache.CentralCacheRead, sharedcache.CentralStateRead, sharedcache.CentralStateWrite,
	})
	handler, err := sharedcache.NewCentralHTTPSHandler(storage)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	server.StartTLS()
	defer server.Close()

	root := writeStickyConnectionRepository(t, server.URL, "example/learning", "BUILDOPT_LEARNING_TOKEN")
	marker := filepath.Join(root, "gradle-environment.txt")
	wrapper := stickyConnectionGradleCommand(root)
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nprintf '%s|%s' \"${BUILDOPT_STICKY_LEARNING-}\" \"${BUILDOPT_STICKY_LIFECYCLE_OUTPUT-}\" > "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	caPath := filepath.Join(root, "central-ca.pem")
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(caPath, certificate, 0o600); err != nil {
		t.Fatal(err)
	}

	previousClock := stickyLearningClock
	previousDetector := stickyLearningDetector
	previousTrial := stickyLearningTrialRunner
	previousPublisher := stickyLearningDecisionPublisher
	t.Cleanup(func() {
		stickyLearningClock = previousClock
		stickyLearningDetector = previousDetector
		stickyLearningTrialRunner = previousTrial
		stickyLearningDecisionPublisher = previousPublisher
	})
	zero := int64(0)
	published := 0
	stickyLearningDetector = func(_ context.Context, input stickyLearningContext) ([]stickyLearningProposal, error) {
		if input.Root != root || input.Connection == nil || input.Config.Mode != "auto" {
			t.Fatalf("learning context = %+v", input)
		}
		return []stickyLearningProposal{{ActionID: "generic-action", Binding: stickydecision.Binding{RepositoryScopeSHA256: input.Connection.projectScopeSHA256}}}, nil
	}
	stickyLearningTrialRunner = func(context.Context, stickyLearningProposal) ([]stickytrial.PairedTrial, stickyvalue.Costs, error) {
		return []stickytrial.PairedTrial{
				fixturePairedTrial("wrapper-pair-1", 1, stickytrial.CandidateFirst),
				fixturePairedTrial("wrapper-pair-2", 2, stickytrial.NativeFirst),
			}, stickyvalue.Costs{
				BootstrapNs: &zero, ObservationNs: &zero, TrialNs: &zero,
				CacheNs: &zero, StateNs: &zero, FallbackNs: &zero,
				ExecutionNs: &zero, PublicationNs: &zero, ValidationNs: &zero,
			}, nil
	}
	stickyLearningDecisionPublisher = func(context.Context, stickyLearningProposal, stickyvalue.Evaluation) error {
		published++
		return nil
	}

	clearStickyNativeIntegrationEnvironment(t)
	t.Setenv(stickyWrapperRootEnvironment, root)
	t.Setenv(stickyWrapperCAEnvironment, caPath)
	t.Setenv("BUILDOPT_LEARNING_TOKEN", stickyConnectionTokenJSON(t, issued))
	t.Setenv(stickyLearningEnvironment, "1")
	t.Setenv("BUILDOPT_STICKY_LIFECYCLE_OUTPUT", filepath.Join(root, "private-lifecycle.json"))
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"run", "--", wrapper, "help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("run exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if published != 1 {
		t.Fatalf("published decisions = %d, want 1", published)
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "|" {
		t.Fatalf("private learning environment reached Gradle: %q", contents)
	}
}

func TestStickyLearningLifecycleEvidenceRecomputesValueAndLedger(t *testing.T) {
	path := filepath.Join("..", "..", "benchmarks", "results", "sticky-wrapper-learning-lifecycle-v1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var evidence struct {
		SchemaVersion string                        `json:"schemaVersion"`
		WorkItem      string                        `json:"workItem"`
		Ledger        stickydecision.EconomicLedger `json:"ledger"`
		Recomputed    stickyvalue.Evaluation        `json:"recomputed"`
		Outcome       string                        `json:"outcome"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err == nil {
		// The partial struct intentionally rejects the other required top-level
		// fields. Decode them below with a complete envelope.
		t.Fatal("partial evidence shape unexpectedly accepted")
	}
	var complete map[string]json.RawMessage
	if err := json.Unmarshal(raw, &complete); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"schemaVersion", "workItem", "ledger", "recomputed", "outcome"} {
		if _, ok := complete[name]; !ok {
			t.Fatalf("missing evidence field %s", name)
		}
	}
	if err := json.Unmarshal(complete["ledger"], &evidence.Ledger); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(complete["recomputed"], &evidence.Recomputed); err != nil {
		t.Fatal(err)
	}
	ledgerRaw, err := stickydecision.MarshalCanonical(evidence.Ledger)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stickydecision.DecodeDocument(ledgerRaw, stickyLearningClock().UTC()); err != nil {
		t.Fatalf("canonical ledger invalid: %v", err)
	}
	zero := int64(0)
	cost := int64(900)
	evaluation, err := stickyvalue.Evaluate([]stickyvalue.Pair{
		{PairID: "p1", Order: "CANDIDATE_FIRST", NativeWallNs: 2000, CandidateWallNs: 1000, OutputsEquivalent: true},
		{PairID: "p2", Order: "NATIVE_FIRST", NativeWallNs: 2000, CandidateWallNs: 1000, OutputsEquivalent: true},
		{PairID: "p3", Order: "CANDIDATE_FIRST", NativeWallNs: 2000, CandidateWallNs: 1000, OutputsEquivalent: true},
		{PairID: "p4", Order: "NATIVE_FIRST", NativeWallNs: 2000, CandidateWallNs: 1000, OutputsEquivalent: true},
	}, stickyvalue.Costs{
		BootstrapNs: &zero, ObservationNs: &zero, TrialNs: &zero,
		CacheNs: &zero, StateNs: &zero, FallbackNs: &zero,
		ExecutionNs: &zero, PublicationNs: &zero, ValidationNs: &cost,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !evaluation.Qualified || evaluation.NetSavingNs != evidence.Recomputed.NetSavingNs || evaluation.LowerBoundNs != evidence.Recomputed.LowerBoundNs {
		t.Fatalf("recomputed value = %+v, committed=%+v", evaluation, evidence.Recomputed)
	}

	now := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	binding := evidence.Ledger.Binding
	observation := stickyobservation.Record{
		SchemaVersion: stickyobservation.SchemaVersion, RecordType: stickyobservation.RecordType,
		ObservationID: "learning-observation-1", IdempotencyKey: strings.Repeat("1", 64),
		Provenance: stickyobservation.Provenance{
			RepositoryScopeSHA256: binding.RepositoryScopeSHA256,
			SourceRevision:        binding.SourceRevision, SourceRevisionEvidence: "EXACT",
			GradleVersion: binding.GradleVersion, WrapperSHA256: binding.WrapperSHA256,
			BuildOptSHA256: binding.BuildOptExecutableSHA256, ArgumentsSHA256: binding.OptionsSHA256,
		},
		Outcome: "SUCCESS", StartedAt: now.Format(time.RFC3339Nano), CompletedAt: now.Add(time.Nanosecond).Format(time.RFC3339Nano),
		Timing: stickyobservation.Timing{
			TotalNs: 1, Decision: stickyobservation.Phase{Evidence: "UNAVAILABLE"},
			Network: stickyobservation.Phase{Evidence: "UNAVAILABLE"}, Cache: stickyobservation.Phase{Evidence: "UNAVAILABLE"},
			Gradle: stickyobservation.Phase{DurationNs: 1, Evidence: "EXACT"}, Observation: stickyobservation.Phase{Evidence: "UNAVAILABLE"},
			Wrapper: stickyobservation.Phase{Evidence: "UNAVAILABLE"}, Bootstrap: stickyobservation.Phase{Evidence: "UNAVAILABLE"},
		},
		ConfigurationCache: stickyobservation.ConfigurationCache{State: "NOT_REQUESTED"},
	}
	if err := observation.Validate(); err != nil {
		t.Fatalf("canonical observation invalid: %v", err)
	}
	action := stickydecision.ActionRecord{
		SchemaVersion: stickydecision.ActionSchemaVersion, RecordType: stickydecision.ActionRecordType,
		ActionID: "runtime-profile-v1", StoreGeneration: 1, IdempotencyKey: strings.Repeat("2", 64), Sequence: 1,
		Transition: "PROPOSE", FromQualificationState: "UNKNOWN", ToQualificationState: "OBSERVING",
		FromRolloutState: "PROPOSED", ToRolloutState: "PROPOSED", Binding: binding,
		OccurredAt: now.Format(time.RFC3339Nano),
	}
	actionRaw, err := stickydecision.MarshalCanonical(action)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stickydecision.DecodeDocument(actionRaw, now); err != nil {
		t.Fatalf("canonical action invalid: %v", err)
	}
	seed := bytes.Repeat([]byte{7}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	decision := stickydecision.Decision{
		SchemaVersion: stickydecision.DecisionSchemaVersion, RecordType: stickydecision.DecisionRecordType,
		DecisionID: "learning-decision-1", StoreGeneration: 2, IdempotencyKey: strings.Repeat("3", 64), Binding: binding,
		ActionID: action.ActionID, ActionGeneration: 1, QualificationState: "QUARANTINE_VALIDATED", RolloutState: "ACTIVE_IN_CI",
		ExecutionDecision: stickydecision.ExecutionActiveRuntime, PolicyDigest: strings.Repeat("4", 64), CacheContractDigest: strings.Repeat("5", 64),
		EvidenceRefs: []string{strings.Repeat("6", 64)}, IssuedAt: now.Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
		Authentication: stickydecision.Authentication{Algorithm: "Ed25519", KeyID: "fixture-key"},
	}
	decisionRaw, _, err := stickydecision.SignDecision(decision, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stickydecision.VerifyDecision(context.Background(), decisionRaw, map[string]ed25519.PublicKey{"fixture-key": privateKey.Public().(ed25519.PublicKey)}, 0, now); err != nil {
		t.Fatalf("canonical signed decision invalid: %v", err)
	}
}

func TestStickyLearningEligibilityRequiresCompleteAuthority(t *testing.T) {
	root := writeStickyNativeNoopFixture(t)
	t.Setenv(stickyLearningEnvironment, "1")
	connection := &stickyWrapperConnection{capabilities: []sharedcache.CentralCapability{
		sharedcache.CentralCacheRead, sharedcache.CentralStateRead, sharedcache.CentralStateWrite,
	}}
	config, reason := stickyLearningEligibility(root, connection, os.Getenv)
	if reason != "ELIGIBLE" || config.Mode != "auto" || config.TrialBudgetPercent != 5 {
		t.Fatalf("eligibility = %s, config=%+v", reason, config)
	}
	connection.capabilities = connection.capabilities[:2]
	if _, reason := stickyLearningEligibility(root, connection, os.Getenv); reason != "MISSING_STATE_WRITE" {
		t.Fatalf("missing write reason = %s", reason)
	}
	t.Setenv(stickyLearningEnvironment, "")
	if _, reason := stickyLearningEligibility(root, connection, os.Getenv); reason != "MISSING_LEARNING_ENVIRONMENT" {
		t.Fatalf("missing environment reason = %s", reason)
	}
	t.Setenv(stickyLearningEnvironment, "1")
	connection.capabilities = []sharedcache.CentralCapability{sharedcache.CentralStateWrite}
	if _, reason := stickyLearningEligibility(root, connection, os.Getenv); reason != "MISSING_READ_CAPABILITY" {
		t.Fatalf("missing read reason = %s", reason)
	}
	connection.capabilities = []sharedcache.CentralCapability{sharedcache.CentralCacheRead, sharedcache.CentralStateRead, sharedcache.CentralStateWrite}
	for _, testCase := range []struct {
		name, mode, budget, want string
	}{
		{name: "observe", mode: "observe", budget: "5", want: "MODE_NOT_AUTO"},
		{name: "off", mode: "off", budget: "5", want: "MODE_NOT_AUTO"},
		{name: "zero budget", mode: "auto", budget: "0", want: "ZERO_TRIAL_BUDGET"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			writeStickyLearningConfig(t, root, testCase.mode, testCase.budget)
			if _, reason := stickyLearningEligibility(root, connection, os.Getenv); reason != testCase.want {
				t.Fatalf("reason = %s, want %s", reason, testCase.want)
			}
		})
	}
	writeStickyLearningConfig(t, root, "auto", "5")
	t.Setenv(bypassEnvironment, "1")
	if _, reason := stickyLearningEligibility(root, connection, os.Getenv); reason != "BYPASS" {
		t.Fatalf("bypass reason = %s", reason)
	}
}

func writeStickyLearningConfig(t *testing.T, root, mode, budget string) {
	t.Helper()
	contents := "schema_version = \"buildopt.config/v1\"\n" +
		"mode = \"" + mode + "\"\n" +
		"server_url = \"\"\n" +
		"project_scope = \"\"\n" +
		"credential_env = \"\"\n" +
		"trial_budget_percent = " + budget + "\n"
	if err := os.WriteFile(filepath.Join(root, ".buildopt", "config.toml"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStickyLearningActiveExecutorUsesLauncherAndScrubsLearning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell")
	}
	root := t.TempDir()
	program := filepath.Join(root, "arm.sh")
	if err := os.WriteFile(program, []byte("#!/bin/sh\nprintf '%s' \"${BUILDOPT_STICKY_LEARNING-}\" > result.txt\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(stickyLearningEnvironment, "1")
	var stdout, stderr bytes.Buffer
	execute := stickyLearningActiveExecutor(stickyLearningReservedEnvironment(), strings.NewReader(""), &stdout, &stderr)
	result := execute(context.Background(), stickyactive.Command{
		Program: program, Dir: root, Env: []string{"PATH=/usr/bin:/bin", stickyLearningEnvironment + "=1"},
	}, []string{"result.txt"})
	if result.Outcome != stickyactive.OutcomeSuccess || result.OutputSHA256 == "" {
		t.Fatalf("active arm = %+v; stderr=%s", result, stderr.String())
	}
	contents, err := os.ReadFile(filepath.Join(root, "result.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Fatalf("learning authority leaked to child: %q", contents)
	}
}

func TestRunTrustedStickyLearningComposesDetectorTrialValueAndPublisher(t *testing.T) {
	previousClock := stickyLearningClock
	previousDetector := stickyLearningDetector
	previousTrial := stickyLearningTrialRunner
	previousPublisher := stickyLearningDecisionPublisher
	t.Cleanup(func() {
		stickyLearningClock = previousClock
		stickyLearningDetector = previousDetector
		stickyLearningTrialRunner = previousTrial
		stickyLearningDecisionPublisher = previousPublisher
	})
	zero := int64(0)
	published := 0
	stickyLearningClock = func() time.Time { return time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC) }
	stickyLearningDetector = func(context.Context, stickyLearningContext) ([]stickyLearningProposal, error) {
		return []stickyLearningProposal{{ActionID: "generic-action", Binding: stickydecision.Binding{RepositoryScopeSHA256: strings.Repeat("a", 64)}}}, nil
	}
	stickyLearningTrialRunner = func(context.Context, stickyLearningProposal) ([]stickytrial.PairedTrial, stickyvalue.Costs, error) {
		trials := []stickytrial.PairedTrial{
			fixturePairedTrial("p1", 1, stickytrial.CandidateFirst),
			fixturePairedTrial("p2", 2, stickytrial.NativeFirst),
		}
		return trials, stickyvalue.Costs{BootstrapNs: &zero, ObservationNs: &zero, TrialNs: &zero, CacheNs: &zero, StateNs: &zero, FallbackNs: &zero, ExecutionNs: &zero, PublicationNs: &zero, ValidationNs: &zero}, nil
	}
	stickyLearningDecisionPublisher = func(_ context.Context, proposal stickyLearningProposal, evaluation stickyvalue.Evaluation) error {
		if proposal.ActionID != "generic-action" || !evaluation.Qualified {
			t.Fatalf("published proposal=%+v evaluation=%+v", proposal, evaluation)
		}
		published++
		return nil
	}
	if err := runTrustedStickyLearning(context.Background(), stickyLearningContext{}); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published decisions = %d, want 1", published)
	}
}

func fixturePairedTrial(id string, sequence uint64, order stickytrial.Order) stickytrial.PairedTrial {
	return stickytrial.PairedTrial{
		SchemaVersion: stickytrial.SchemaVersion, RecordType: stickytrial.RecordType,
		TrialID: id, Sequence: sequence, Pair: int(sequence), Order: order,
		AssignmentBeforeRun: true, IsolationDigest: strings.Repeat("a", 64), InvocationCount: 2,
		Candidate:   stickytrial.ArmResult{Outcome: stickytrial.OutcomeSuccess, DurationNs: 1000, OutputSHA256: strings.Repeat("b", 64), OutputBytes: 4},
		Native:      stickytrial.ArmResult{Outcome: stickytrial.OutcomeSuccess, DurationNs: 2000, OutputSHA256: strings.Repeat("b", 64), OutputBytes: 4},
		Equivalence: stickytrial.EquivalenceExact, Result: stickytrial.ResultCandidateFaster,
		NaturalRunnerNs: 1_000_000, BudgetLimitNs: 50_000, ReservedExtraNs: 10_000, ActualExtraNs: 3000,
		StartedAt: "2026-08-27T00:00:00Z", CompletedAt: "2026-08-27T00:00:01Z",
	}
}
