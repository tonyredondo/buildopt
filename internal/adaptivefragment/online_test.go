package adaptivefragment

import (
	"reflect"
	"strings"
	"testing"
)

func TestOnlineLearnerResumesAndSuspendsOnlyDependents(t *testing.T) {
	checkpoint := onlineFixture(t)
	original := checkpoint

	first := applyOnlineFixture(t, checkpoint, 1, 200)
	if !reflect.DeepEqual(checkpoint, original) {
		t.Fatal("online learner mutated the prior checkpoint")
	}
	assertOnlineStates(t, first.Checkpoint, StateObserved, StateObserved, StateObserved)
	second := applyOnlineFixture(t, first.Checkpoint, 2, 200)
	assertOnlineStates(t, second.Checkpoint, StateShadow, StateShadow, StateShadow)

	document, err := MarshalOnlineCheckpoint(second.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := ResumeOnlineCheckpoint(document, second.CheckpointSHA256,
		second.Checkpoint.RepositoryScopeSHA256, second.Checkpoint.ContextBindingsSHA256)
	if err != nil || !reflect.DeepEqual(resumed, second.Checkpoint) {
		t.Fatalf("resumed checkpoint = %+v, err=%v", resumed, err)
	}

	third := applyOnlineFixture(t, resumed, 3, 200)
	fourth := applyOnlineFixture(t, third.Checkpoint, 4, 200)
	assertOnlineStates(t, fourth.Checkpoint, StateQualified, StateQualified, StateQualified)
	if len(fourth.QualifiedFamilies) != 3 {
		t.Fatalf("qualified families = %v", fourth.QualifiedFamilies)
	}

	fifth := applyOnlineFixture(t, fourth.Checkpoint, 5, -1000)
	assertOnlineStates(t, fifth.Checkpoint, StateSuspended, StateSuspended, StateQualified)
	if !reflect.DeepEqual(fifth.SuspendedFamilies, []string{onlineFamilyA, onlineFamilyB}) {
		t.Fatalf("suspended families = %v", fifth.SuspendedFamilies)
	}
	if fifth.Checkpoint.Fragments[0].Assessment.Entry.CumulativeNetMs != -250 ||
		fifth.Checkpoint.Fragments[2].Assessment.Entry.CumulativeNetMs != 200 {
		t.Fatalf("unexpected signed economics: %+v / %+v",
			fifth.Checkpoint.Fragments[0].Assessment, fifth.Checkpoint.Fragments[2].Assessment)
	}
	if fifth.Checkpoint.Fragments[0].SuspensionReason != "VALUE_REGRESSION" ||
		fifth.Checkpoint.Fragments[1].SuspensionReason != "DEPENDENCY_SUSPENDED" ||
		fifth.Checkpoint.Fragments[2].SuspensionReason != "" {
		t.Fatalf("suspension reasons = %q / %q / %q", fifth.Checkpoint.Fragments[0].SuspensionReason,
			fifth.Checkpoint.Fragments[1].SuspensionReason, fifth.Checkpoint.Fragments[2].SuspensionReason)
	}
	if fifth.Checkpoint.RequestedBuildCount != 5 {
		t.Fatalf("requested builds = %d", fifth.Checkpoint.RequestedBuildCount)
	}
	sixthBuild := onlineBuild(fifth.Checkpoint, 6, 0)
	for _, index := range []int{0, 1} {
		sixthBuild.Samples[index].CandidateValueObserved = false
		sixthBuild.Samples[index].GrossSavedMs = 0
	}
	sixth, err := ApplyOrdinaryBuild(fifth.Checkpoint, sixthBuild)
	if err != nil {
		t.Fatal(err)
	}
	assertOnlineStates(t, sixth.Checkpoint, StateSuspended, StateSuspended, StateQualified)
	if len(sixth.Checkpoint.Fragments[0].Observations) != 5 || len(sixth.Checkpoint.Fragments[2].Observations) != 6 {
		t.Fatalf("suspended/unrelated observation counts = %d / %d",
			len(sixth.Checkpoint.Fragments[0].Observations), len(sixth.Checkpoint.Fragments[2].Observations))
	}
}

func TestOnlineLearnerRejectsNonOrdinaryOrDriftedBuilds(t *testing.T) {
	checkpoint := onlineFixture(t)
	valid := onlineBuild(checkpoint, 1, 200)
	cases := map[string]func(*OrdinaryBuildUpdate){
		"measurement only": func(build *OrdinaryBuildUpdate) { build.MeasurementOnly = true },
		"wrong source":     func(build *OrdinaryBuildUpdate) { build.Source = "MEASUREMENT_ONLY" },
		"binding drift":    func(build *OrdinaryBuildUpdate) { build.ContextBindingsSHA256 = strings.Repeat("f", 64) },
		"cohort drift":     func(build *OrdinaryBuildUpdate) { build.Samples[0].CohortSHA256 = strings.Repeat("f", 64) },
		"inexact outputs":  func(build *OrdinaryBuildUpdate) { build.Samples[0].ExactOutputs = false },
		"product failure":  func(build *OrdinaryBuildUpdate) { build.Samples[0].ProductAttributableFailure = true },
		"wrong sequence":   func(build *OrdinaryBuildUpdate) { build.Sequence = 2 },
		"unobserved value": func(build *OrdinaryBuildUpdate) {
			build.Samples[0].CandidateValueObserved = false
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			build := valid
			build.Samples = append([]OnlineFragmentSample{}, valid.Samples...)
			mutate(&build)
			if _, err := ApplyOrdinaryBuild(checkpoint, build); err == nil {
				t.Fatal("invalid ordinary build was accepted")
			}
		})
	}
}

func TestOnlineLearnerResumeRequiresExactDigestAndBindings(t *testing.T) {
	first := applyOnlineFixture(t, onlineFixture(t), 1, 200)
	document, err := MarshalOnlineCheckpoint(first.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		digest, repository, context string
	}{
		{strings.Repeat("f", 64), first.Checkpoint.RepositoryScopeSHA256, first.Checkpoint.ContextBindingsSHA256},
		{first.CheckpointSHA256, strings.Repeat("f", 64), first.Checkpoint.ContextBindingsSHA256},
		{first.CheckpointSHA256, first.Checkpoint.RepositoryScopeSHA256, strings.Repeat("f", 64)},
	}
	for _, test := range cases {
		if _, err := ResumeOnlineCheckpoint(document, test.digest, test.repository, test.context); err == nil {
			t.Fatal("incompatible checkpoint resume was accepted")
		}
	}
}

func TestOnlineLearnerRejectsDependencyCycle(t *testing.T) {
	fragments := onlineFragments()
	fragments[0].Requires = []string{onlineFamilyB}
	if _, err := NewOnlineCheckpoint(strings.Repeat("d", 64), strings.Repeat("e", 64),
		"2026-08-25T10:00:00Z", onlinePolicy(), fragments); err == nil {
		t.Fatal("dependency cycle was accepted")
	}
}

func TestOnlineLearnerRejectsUnsupportedSuspensionState(t *testing.T) {
	fourth := applyOnlineFixture(t, applyOnlineFixture(t, applyOnlineFixture(t,
		applyOnlineFixture(t, onlineFixture(t), 1, 200).Checkpoint, 2, 200).Checkpoint, 3, 200).Checkpoint, 4, 200)

	unsupportedValue := fourth.Checkpoint
	unsupportedValue.Fragments = cloneOnlineFragments(fourth.Checkpoint.Fragments)
	unsupportedValue.Fragments[0].State = StateSuspended
	unsupportedValue.Fragments[0].SuspensionReason = "VALUE_REGRESSION"
	if err := validateOnlineCheckpoint(unsupportedValue); err == nil {
		t.Fatal("positive fragment accepted a value-regression suspension")
	}

	regressed := applyOnlineFixture(t, fourth.Checkpoint, 5, -1000).Checkpoint
	brokenDependency := regressed
	brokenDependency.Fragments = cloneOnlineFragments(regressed.Fragments)
	brokenDependency.Fragments[1].State = StateQualified
	brokenDependency.Fragments[1].SuspensionReason = ""
	if err := validateOnlineCheckpoint(brokenDependency); err == nil {
		t.Fatal("qualified fragment accepted a suspended requirement")
	}
}

const (
	onlineFamilyA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	onlineFamilyB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	onlineFamilyC = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func onlineFixture(t *testing.T) OnlineCheckpoint {
	t.Helper()
	checkpoint, err := NewOnlineCheckpoint(strings.Repeat("d", 64), strings.Repeat("e", 64),
		"2026-08-25T10:00:00Z", onlinePolicy(), onlineFragments())
	if err != nil {
		t.Fatal(err)
	}
	return checkpoint
}

func onlineFragments() []OnlineFragment {
	return []OnlineFragment{
		{FamilyID: onlineFamilyA, RevisionID: strings.Repeat("1", 64), Generation: 1, State: StateObserved, EvidenceExpiresAt: "2026-09-25T10:00:00Z"},
		{FamilyID: onlineFamilyB, RevisionID: strings.Repeat("2", 64), Generation: 1, Requires: []string{onlineFamilyA}, State: StateObserved, EvidenceExpiresAt: "2026-09-25T10:00:00Z"},
		{FamilyID: onlineFamilyC, RevisionID: strings.Repeat("3", 64), Generation: 1, State: StateObserved, EvidenceExpiresAt: "2026-09-25T10:00:00Z"},
	}
}

func onlinePolicy() EconomicPolicy {
	return EconomicPolicy{DecayPermille: 900, Horizons: []uint64{1, 5, 10}, RegretBudgetMs: 1000}
}

func onlineBuild(checkpoint OnlineCheckpoint, sequence uint64, grossA int64) OrdinaryBuildUpdate {
	gross := []int64{grossA, 100, 50}
	samples := make([]OnlineFragmentSample, len(checkpoint.Fragments))
	for index, fragment := range checkpoint.Fragments {
		samples[index] = OnlineFragmentSample{
			FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
			CohortSHA256:           checkpoint.ContextBindingsSHA256,
			EvidenceDocumentSHA256: sha("online-evidence-" + string(rune('0'+sequence)) + fragment.FamilyID),
			Compatible:             true, CandidateValueObserved: true, GrossSavedMs: gross[index],
			SynchronousOverheadMs: 10, ExactOutputs: true,
		}
	}
	return OrdinaryBuildUpdate{
		BuildID: sha("online-build-" + string(rune('0'+sequence))), Sequence: sequence,
		Source: OrdinaryBuildEvidenceSource, RepositoryScopeSHA256: checkpoint.RepositoryScopeSHA256,
		ContextBindingsSHA256: checkpoint.ContextBindingsSHA256,
		ObservedAt:            "2026-08-25T10:0" + string(rune('0'+sequence)) + ":00Z", Samples: samples,
	}
}

func applyOnlineFixture(t *testing.T, checkpoint OnlineCheckpoint, sequence uint64, grossA int64) OnlineUpdate {
	t.Helper()
	update, err := ApplyOrdinaryBuild(checkpoint, onlineBuild(checkpoint, sequence, grossA))
	if err != nil {
		t.Fatal(err)
	}
	return update
}

func assertOnlineStates(t *testing.T, checkpoint OnlineCheckpoint, first, second, third State) {
	t.Helper()
	want := []State{first, second, third}
	for index, fragment := range checkpoint.Fragments {
		if fragment.State != want[index] {
			t.Fatalf("fragment %d state = %s, want %s", index, fragment.State, want[index])
		}
	}
}
