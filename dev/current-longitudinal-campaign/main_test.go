package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestTimedCommandRecordsMonotonicDurationAndExitCode(t *testing.T) {
	durationPath := t.TempDir() + "/duration"
	t.Setenv("BUILDOPT_TIMER_HELPER_EXIT", "37")
	status := runTimedCommand(durationPath, []string{os.Args[0], "-test.run=^TestTimedCommandHelper$"})
	if status != 37 {
		t.Fatalf("expected child exit 37, got %d", status)
	}
	raw, err := os.ReadFile(durationPath)
	if err != nil {
		t.Fatal(err)
	}
	duration, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || duration <= 0 {
		t.Fatalf("invalid monotonic duration %q: %v", raw, err)
	}
}

func TestTimedCommandHelper(t *testing.T) {
	if os.Getenv("BUILDOPT_TIMER_HELPER_EXIT") == "37" {
		os.Exit(37)
	}
}

func TestAggregateKeepsSignedEconomicsAndNoFragmentRuntime(t *testing.T) {
	raw := rawEvidence{
		SchemaVersion: rawSchema, WorkItem: "AF-014C", CapturedAt: "2026-08-26T00:00:00Z",
		EvaluatedSHA: strings.Repeat("a", 64), ExecutableSHA: strings.Repeat("b", 64),
		CohortSHA: cohortSHA, ContractSHA: strings.Repeat("c", 64),
		Boundaries: boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"},
	}
	for repositoryIndex := range 5 {
		repository := repositoryRaw{Key: "repository-" + string(rune('a'+repositoryIndex)),
			RepositoryID: "owner/repository", Workflow: []string{"assemble"}, Outputs: []string{"build/**"},
			DependencyPreparation: validDependencyPreparation()}
		var cumulative int64
		for sequence := 1; sequence <= 15; sequence++ {
			delta := int64(100)
			if sequence == 15 {
				delta = -50
			}
			cumulative += delta
			order := "CONTROL_FIRST"
			if sequence%2 == 0 {
				order = "CANDIDATE_FIRST"
			}
			stateBefore := ""
			if sequence > 1 {
				stateBefore = strings.Repeat("6", 64)
			}
			repository.Observations = append(repository.Observations, observation{
				CohortAttempt: sequence, Sequence: sequence, Source: "PRIMARY", FrozenOrdinal: sequence,
				Revision: strings.Repeat("d", 40), Parent: strings.Repeat("e", 40),
				ChangeShape: "PRODUCTION_SOURCE", Order: order,
				Control: validArm(1100, ""), Candidate: validArm(1100-delta, stateBefore),
				ExactOutputs: true, SignedDeltaNS: delta, CumulativeNS: cumulative,
				Decision: candidateDecision{Outcome: "NATIVE_RETAINED", Reason: "NO_PROFILE",
					Phase: "ACTIVE", ExecutionMode: "OPTIMIZED_NATIVE", SelectionStatus: "NATIVE_RETAINED",
					CalibrationReason: "NOT_REQUIRED", NativeRetention: nativeRetention{DecisionPhase: "PRE_GRADLE_COMPATIBILITY", Reason: "NO_PROFILE"},
					RuntimeSurface: "NO_FRAGMENT_RUNTIME", ActivatedFragments: []string{}, SuspendedFragments: []string{},
					Timing: validTiming(1000)},
			})
		}
		repository.DependencyPreparation.Attempts = 16
		repository.DependencyPreparation.Successes = 16
		raw.Repositories = append(raw.Repositories, repository)
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	validated, err := decodeRaw(rawBytes)
	if err != nil {
		t.Fatal(err)
	}
	report, err := aggregate(rawBytes, validated)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ComparablePairs != 75 || report.Summary.FragmentActivations != 0 ||
		report.Repositories[0].Outcome != "NET_POSITIVE" || report.Repositories[0].CumulativeNetNS != 1350 {
		t.Fatalf("unexpected aggregate: %+v", report)
	}
}

func TestRawAcceptsInsufficientRepositoryWithOnlyDeclaredExclusions(t *testing.T) {
	raw := rawEvidence{
		SchemaVersion: rawSchema, WorkItem: "AF-014C", CapturedAt: "2026-08-26T00:00:00Z",
		EvaluatedSHA: strings.Repeat("a", 40), ExecutableSHA: strings.Repeat("b", 64),
		CohortSHA: cohortSHA, ContractSHA: strings.Repeat("c", 64),
		Boundaries: boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"},
	}
	for repositoryIndex := range 5 {
		repository := repositoryRaw{Key: "repository-" + string(rune('a'+repositoryIndex)),
			RepositoryID: "owner/repository", Workflow: []string{"assemble"}, Outputs: []string{"build/**"},
			DependencyPreparation: validDependencyPreparation()}
		for attempt := 1; attempt <= 30; attempt++ {
			source := "PRIMARY"
			ordinal := attempt
			if attempt > 20 {
				source = "RESERVE"
				ordinal = attempt - 20
			}
			repository.Exclusions = append(repository.Exclusions, exclusion{CohortAttempt: attempt,
				Source: source, FrozenOrdinal: ordinal, Revision: strings.Repeat("d", 40),
				Reason: "DEPENDENCY_UNAVAILABLE_AFTER_PREPARATION", Detail: "CONTROL_ARM_FAILED"})
		}
		raw.Repositories = append(raw.Repositories, repository)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeRaw(encoded); err != nil {
		t.Fatalf("expected declared insufficient cohorts to remain valid: %v", err)
	}
}

func TestRawRejectsInvalidGitObjectIdentity(t *testing.T) {
	if validGitOID(strings.Repeat("a", 39)) || validGitOID(strings.Repeat("z", 40)) ||
		!validGitOID(strings.Repeat("a", 40)) || !validGitOID(strings.Repeat("b", 64)) {
		t.Fatal("unexpected Git object identity validation")
	}
}

func TestRawRejectsRelabelledWholeProfile(t *testing.T) {
	value := observation{Decision: candidateDecision{RuntimeSurface: "NO_FRAGMENT_RUNTIME", ActivatedFragments: []string{"fake"}}}
	if err := validateObservation(value, 1, 0); err == nil {
		t.Fatal("expected relabelled whole profile to fail")
	}
}

func TestRawRejectsCalibrationWithoutAttributableCost(t *testing.T) {
	value := validObservation()
	value.Decision.CalibrationPerformed = true
	if err := validateObservation(value, 1, 0); err == nil {
		t.Fatal("expected calibration without cost and samples to fail")
	}
}

func TestRawRejectsSelectedProfileWithoutIdentity(t *testing.T) {
	value := validObservation()
	value.Decision.SelectionSelected = true
	value.Decision.SelectionPerformed = true
	if err := validateObservation(value, 1, 0); err == nil {
		t.Fatal("expected selected profile without identity to fail")
	}
}

func validObservation() observation {
	return observation{CohortAttempt: 1, Sequence: 1, Source: "PRIMARY", FrozenOrdinal: 1,
		Revision: strings.Repeat("d", 40), Parent: strings.Repeat("e", 40), ChangeShape: "PRODUCTION_SOURCE",
		Order: "CONTROL_FIRST", Control: validArm(1100, ""), Candidate: validArm(1000, ""),
		ExactOutputs: true, SignedDeltaNS: 100, CumulativeNS: 100,
		Decision: candidateDecision{Outcome: "NATIVE_RETAINED", Reason: "NO_PROFILE", Phase: "ACTIVE",
			ExecutionMode: "OPTIMIZED_NATIVE", SelectionStatus: "NATIVE_RETAINED", CalibrationReason: "NOT_REQUIRED",
			NativeRetention: nativeRetention{DecisionPhase: "PRE_GRADLE_COMPATIBILITY", Reason: "NO_PROFILE"},
			RuntimeSurface:  "NO_FRAGMENT_RUNTIME", ActivatedFragments: []string{}, SuspendedFragments: []string{},
			Timing: validTiming(1000)}}
}

func validDependencyPreparation() dependencyPreparation {
	return dependencyPreparation{Mode: "ANCHOR_AND_EACH_ATTEMPT_UNMEASURED", Attempts: 1, Successes: 1,
		WallNS: 1, CopiedSurfaces: []string{"GRADLE_DEPENDENCY_MODULES", "GRADLE_DEPENDENCY_VERIFICATION", "GRADLE_WRAPPER_DISTRIBUTIONS"}}
}

func validArm(wall int64, stateBefore string) arm {
	return arm{WallNS: wall, ExitCode: 0, OutputSHA: strings.Repeat("1", 64), OutputCount: 1,
		CheckoutSHA: strings.Repeat("2", 64), GradleHomeSHA: strings.Repeat("3", 64),
		BuildCacheSHA: strings.Repeat("4", 64), DaemonSHA: strings.Repeat("5", 64),
		BuildOptStateSHA: strings.Repeat("6", 64), BuildOptStateBeforeSHA: stateBefore}
}

func validTiming(total int64) timing {
	return timing{PreExecutionNS: 100, GradleExecutionNS: total - 300, FinalizationNS: 100, UnattributedNS: 100, TotalNS: total}
}
