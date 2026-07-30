package neutralenvelope

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildPilotResultDemonstratesNetCausalSavings(t *testing.T) {
	observations := testPilotObservations(t, -1)
	result, err := BuildPilotResult(
		observations,
		7,
		time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("BuildPilotResult: %v", err)
	}
	if err := ValidatePilotResult(result, observations); err != nil {
		t.Fatalf("ValidatePilotResult: %v", err)
	}
	if !result.DemonstratesNetCausalSavings() {
		t.Fatalf("result does not demonstrate net savings: %+v", result.Effects)
	}
	if result.Status != "PRELIMINARY" ||
		result.Decision.State != "PRELIMINARY" ||
		result.Samples.Assigned != (ResultArmCounts{Candidate: 4, Control: 4}) ||
		result.Samples.Analyzed != result.Samples.Assigned ||
		result.Effects.ObservedNetBuildTimeSavedMs != 600 ||
		result.Effects.ObservedNetBuildTimeSavedInterval95Ms[0] <= 0 ||
		result.Effects.ObservedBuildTimeReductionRatio != 0.6 ||
		result.Effects.CustomerVisibleBuildP95DeltaMs != -600 ||
		result.Effects.IncrementalActionOverheadMs != 7 {
		t.Fatalf("unexpected pilot result: %+v", result)
	}
}

func TestPilotResultRetainsFailuresAndCannotPassGate(t *testing.T) {
	observations := testPilotObservations(t, 2)
	result, err := BuildPilotResult(
		observations,
		0,
		time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("BuildPilotResult: %v", err)
	}
	if result.Samples.Outcomes.Candidate.BuildFailure != 1 ||
		result.Samples.Analyzed.Candidate != 3 ||
		result.Samples.ExcludedSampleSize != 2 ||
		result.Effects.ProductAttributableFailureRate != 0.25 ||
		result.DemonstratesNetCausalSavings() {
		t.Fatalf("failed outcome was hidden: %+v", result)
	}
}

func TestPilotAssignmentAndObservationRejectDrift(t *testing.T) {
	definition := testPilotDefinition()
	assignedAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	assignment, err := NewPilotAssignment(
		definition,
		1,
		"CONTROL",
		assignedAt,
	)
	if err != nil {
		t.Fatalf("NewPilotAssignment: %v", err)
	}
	if assignment.OrderInPair != 1 || assignment.CacheState != "DISABLED" {
		t.Fatalf("unexpected control assignment: %+v", assignment)
	}
	assignment.CacheState = "WARM_MANAGED_L1"
	if err := assignment.Validate(); err == nil {
		t.Fatal("treatment drift passed assignment validation")
	}

	assignment, err = NewPilotAssignment(
		definition,
		1,
		"CONTROL",
		assignedAt,
	)
	if err != nil {
		t.Fatalf("NewPilotAssignment: %v", err)
	}
	if _, err := NewPilotObservation(
		assignment,
		assignedAt.Add(-time.Millisecond),
		assignedAt.Add(time.Second),
		"SUCCESS",
		0,
		"AVAILABLE",
		"sha256:"+strings.Repeat("d", 64),
		10,
	); err == nil || !strings.Contains(err.Error(), "before assignment") {
		t.Fatalf("pre-assignment execution error = %v", err)
	}
}

func TestPilotResultPublicationIsPrivateImmutableAndExact(t *testing.T) {
	observations := testPilotObservations(t, -1)
	result, err := BuildPilotResult(
		observations,
		0,
		time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("BuildPilotResult: %v", err)
	}
	directory := filepath.Join(t.TempDir(), "results")
	documentPath, streamPath, err := PublishPilotResult(directory, result)
	if err != nil {
		t.Fatalf("PublishPilotResult: %v", err)
	}
	if _, _, err := PublishPilotResult(directory, result); err != nil {
		t.Fatalf("idempotent PublishPilotResult: %v", err)
	}
	for _, path := range []string{documentPath, streamPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", path, info.Mode().Perm())
		}
	}
	directoryInfo, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("stat result directory: %v", err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("result directory mode = %o, want 700", directoryInfo.Mode().Perm())
	}
	streamContent, err := os.ReadFile(streamPath)
	if err != nil {
		t.Fatalf("read result stream: %v", err)
	}
	if bytes.Count(streamContent, []byte{'\n'}) != 1 {
		t.Fatalf("idempotent result duplicated JSONL: %q", streamContent)
	}
	var exported bytes.Buffer
	if err := WritePilotResultStream(directory, &exported); err != nil {
		t.Fatalf("WritePilotResultStream: %v", err)
	}
	if !bytes.Equal(exported.Bytes(), streamContent) {
		t.Fatal("stdout export differs from durable JSONL bytes")
	}

	conflict := result
	later := time.Date(2026, 7, 30, 14, 0, 1, 0, time.UTC).
		Format(time.RFC3339Nano)
	conflict.AsOf = later
	conflict.Decision.EvaluatedAt = later
	if _, _, err := PublishPilotResult(
		directory,
		conflict,
	); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("conflicting immutable result error = %v", err)
	}
}

func TestPilotPairRejectsOrderOverlapAndStratumDrift(t *testing.T) {
	observations := testPilotObservations(t, -1)
	overlap := append([]PilotObservation(nil), observations...)
	secondStartedAt, _ := parseCanonicalTime(overlap[1].StartedAt)
	firstStartedAt, _ := parseCanonicalTime(overlap[0].StartedAt)
	overlap[0].CompletedAt = secondStartedAt.Add(time.Millisecond).
		Format(time.RFC3339Nano)
	overlap[0].DurationMs = secondStartedAt.Add(time.Millisecond).
		Sub(firstStartedAt).Milliseconds()
	if _, err := BuildPilotResult(
		overlap,
		0,
		time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
	); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("overlap error = %v", err)
	}

	drift := append([]PilotObservation(nil), observations...)
	driftedAssignment := drift[2].Assignment
	driftedAssignment.PipelineClass = "different-pipeline"
	digest, err := PilotAssignmentDigest(driftedAssignment)
	if err != nil {
		t.Fatalf("PilotAssignmentDigest: %v", err)
	}
	drift[2].Assignment = driftedAssignment
	drift[2].AssignmentDigest = digest
	sessionDigest := sha256DigestForTest(
		"buildopt-causal-pilot-session-v1\x00" + digest,
	)
	drift[2].BuildSessionID = "pilot-session-" + sessionDigest[:32]
	if _, err := BuildPilotResult(
		drift,
		0,
		time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC),
	); err == nil || !strings.Contains(err.Error(), "stratum drift") {
		t.Fatalf("stratum drift error = %v", err)
	}
}

func testPilotObservations(
	t *testing.T,
	failedCandidatePair int,
) []PilotObservation {
	t.Helper()
	definition := testPilotDefinition()
	base := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	observations := make([]PilotObservation, 0, 8)
	for pairIndex := 1; pairIndex <= 4; pairIndex++ {
		firstArm := "CONTROL"
		if pairIndex%2 == 0 {
			firstArm = "CANDIDATE"
		}
		arms := []string{firstArm, oppositePilotArm(firstArm)}
		for orderIndex, arm := range arms {
			assignedAt := base.Add(
				time.Duration((pairIndex-1)*4+orderIndex*2) * time.Second,
			)
			assignment, err := NewPilotAssignment(
				definition,
				pairIndex,
				arm,
				assignedAt,
			)
			if err != nil {
				t.Fatalf("NewPilotAssignment: %v", err)
			}
			startedAt := assignedAt.Add(100 * time.Millisecond)
			duration := time.Second
			if arm == "CANDIDATE" {
				duration = 400 * time.Millisecond
			}
			outcome := "SUCCESS"
			exitCode := 0
			status := "AVAILABLE"
			digest := "sha256:" + strings.Repeat("d", 64)
			size := int64(10)
			if arm == "CANDIDATE" && pairIndex == failedCandidatePair {
				outcome = "BUILD_FAILURE"
				exitCode = 37
				status = "NOT_AVAILABLE"
				digest = ""
				size = 0
			}
			observation, err := NewPilotObservation(
				assignment,
				startedAt,
				startedAt.Add(duration),
				outcome,
				exitCode,
				status,
				digest,
				size,
			)
			if err != nil {
				t.Fatalf("NewPilotObservation: %v", err)
			}
			observations = append(observations, observation)
		}
	}
	return observations
}

func testPilotDefinition() PilotDefinition {
	return PilotDefinition{
		ExperimentID:             "a0-009-managed-l1-pilot",
		MeasurementEpoch:         1,
		ActionID:                 "managed-l1-cache-hit-v1",
		BaselineDefinitionDigest: "sha256:" + strings.Repeat("a", 64),
		ControlDefinitionDigest:  "sha256:" + strings.Repeat("b", 64),
		CohortID:                 "internal-linux-amd64",
		Environment:              "LOCAL",
		PipelineClass:            "a0-internal-pilot",
		RunnerClass:              "host-smoke",
		WorkUnitsFingerprint:     "hmac-sha256:" + strings.Repeat("c", 64),
		RequiredDeliverable:      "libs/pilot.jar",
	}
}

func oppositePilotArm(arm string) string {
	if arm == "CONTROL" {
		return "CANDIDATE"
	}
	return "CONTROL"
}

func sha256DigestForTest(value string) string {
	return strings.TrimPrefix(digestBytes([]byte(value)), "sha256:")
}
