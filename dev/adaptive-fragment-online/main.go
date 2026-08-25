// Command adaptive-fragment-online produces and validates the synthetic AF-006
// online-learning state-machine proof. It executes no Gradle build or fragment.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
)

const (
	reportSchema = "buildopt.poc/adaptive-fragment-online/v1"
	outcome      = "ONLINE_FRAGMENT_LEARNING_AVAILABLE"
	familyA      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	familyB      = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	familyC      = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

type report struct {
	SchemaVersion string                            `json:"schemaVersion"`
	WorkItem      string                            `json:"workItem"`
	CapturedAt    string                            `json:"capturedAt"`
	Policy        policy                            `json:"policy"`
	Timeline      []timelineEntry                   `json:"timeline"`
	Resume        resumeProof                       `json:"resume"`
	Regression    regressionProof                   `json:"regression"`
	Summary       summary                           `json:"summary"`
	FinalState    adaptivefragment.OnlineCheckpoint `json:"finalState"`
	Boundaries    boundaries                        `json:"boundaries"`
	Outcome       string                            `json:"outcome"`
}

type policy struct {
	EvidenceSource             string `json:"evidenceSource"`
	MinimumShadowBuilds        uint64 `json:"minimumShadowBuilds"`
	MinimumQualificationBuilds uint64 `json:"minimumQualificationBuilds"`
	ExactOutputsRequired       bool   `json:"exactOutputsRequired"`
	ComparableCohortRequired   bool   `json:"comparableCohortRequired"`
}

type timelineEntry struct {
	RequestedBuilds uint64            `json:"requestedBuilds"`
	States          map[string]string `json:"states"`
	Qualified       []string          `json:"qualified"`
	Suspended       []string          `json:"suspended"`
}

type resumeProof struct {
	InterruptedGeneration       uint64 `json:"interruptedGeneration"`
	InterruptedCheckpointSHA256 string `json:"interruptedCheckpointSha256"`
	ExactResumeAccepted         bool   `json:"exactResumeAccepted"`
	DigestMismatchRejected      bool   `json:"digestMismatchRejected"`
	RepositoryMismatchRejected  bool   `json:"repositoryMismatchRejected"`
	BindingMismatchRejected     bool   `json:"bindingMismatchRejected"`
	UnknownFieldRejected        bool   `json:"unknownFieldRejected"`
}

type regressionProof struct {
	RegressedFamily          string   `json:"regressedFamily"`
	RegressedNetMs           int64    `json:"regressedNetMs"`
	SuspendedFamilies        []string `json:"suspendedFamilies"`
	DependentFamily          string   `json:"dependentFamily"`
	UnaffectedFamily         string   `json:"unaffectedFamily"`
	UnaffectedState          string   `json:"unaffectedState"`
	UnaffectedNetMs          int64    `json:"unaffectedNetMs"`
	UnrelatedSuspensionCount uint64   `json:"unrelatedSuspensionCount"`
}

type summary struct {
	RequestedBuilds           uint64 `json:"requestedBuilds"`
	MeasurementOnlyBuilds     uint64 `json:"measurementOnlyBuilds"`
	AcceptedOrdinaryBuilds    uint64 `json:"acceptedOrdinaryBuilds"`
	ComparableFragmentSamples uint64 `json:"comparableFragmentSamples"`
	QualifiedBeforeRegression uint64 `json:"qualifiedBeforeRegression"`
	SuspendedAfterRegression  uint64 `json:"suspendedAfterRegression"`
	IndependentQualifiedAfter uint64 `json:"independentQualifiedAfter"`
	RejectedInvalidUpdates    uint64 `json:"rejectedInvalidUpdates"`
	PriorCheckpointMutations  uint64 `json:"priorCheckpointMutations"`
}

type boundaries struct {
	ProofOfConcept       bool   `json:"proofOfConcept"`
	SyntheticTimingClaim bool   `json:"syntheticTimingClaim"`
	GradleExecutions     uint64 `json:"gradleExecutions"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	ProductionAuthorized bool   `json:"productionAuthorized"`
	TestOptimization     string `json:"testOptimization"`
}

func main() {
	output := flag.String("output", "", "write the AF-006 online-learning report")
	validate := flag.String("validate", "", "validate an AF-006 online-learning report")
	flag.Parse()
	if flag.NArg() != 0 || (*output == "") == (*validate == "") {
		fmt.Fprintln(os.Stderr, "usage: adaptive-fragment-online (--output <path> | --validate <path>)")
		os.Exit(64)
	}
	expected, err := buildReport()
	if err == nil {
		err = validateReport(expected)
	}
	if err == nil && *validate != "" {
		var candidate report
		err = readJSONStrict(*validate, &candidate)
		if err == nil && !reflect.DeepEqual(candidate, expected) {
			err = errors.New("online-learning report does not match recomputed evidence")
		}
	}
	if err == nil && *output != "" {
		err = writeJSON(*output, expected)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "adaptive fragment online learning failed: %v\n", err)
		os.Exit(1)
	}
	if *validate != "" {
		fmt.Println("adaptive fragment online learning: ONLINE_FRAGMENT_LEARNING_AVAILABLE")
	}
}

func buildReport() (report, error) {
	checkpoint, err := newCheckpoint()
	if err != nil {
		return report{}, err
	}
	initial := checkpoint
	timeline := []timelineEntry{}
	first, err := apply(checkpoint, 1, 200)
	if err != nil {
		return report{}, err
	}
	timeline = append(timeline, timelineFor(first))
	priorMutations := uint64(0)
	if !reflect.DeepEqual(initial, checkpoint) {
		priorMutations++
	}
	second, err := apply(first.Checkpoint, 2, 200)
	if err != nil {
		return report{}, err
	}
	timeline = append(timeline, timelineFor(second))
	document, err := adaptivefragment.MarshalOnlineCheckpoint(second.Checkpoint)
	if err != nil {
		return report{}, err
	}
	resumed, err := adaptivefragment.ResumeOnlineCheckpoint(document, second.CheckpointSHA256,
		second.Checkpoint.RepositoryScopeSHA256, second.Checkpoint.ContextBindingsSHA256)
	if err != nil {
		return report{}, err
	}
	resume := resumeProof{
		InterruptedGeneration:       second.Checkpoint.Generation,
		InterruptedCheckpointSHA256: second.CheckpointSHA256,
		ExactResumeAccepted:         reflect.DeepEqual(resumed, second.Checkpoint),
		DigestMismatchRejected:      resumeRejected(document, strings.Repeat("f", 64), second.Checkpoint.RepositoryScopeSHA256, second.Checkpoint.ContextBindingsSHA256),
		RepositoryMismatchRejected:  resumeRejected(document, second.CheckpointSHA256, strings.Repeat("f", 64), second.Checkpoint.ContextBindingsSHA256),
		BindingMismatchRejected:     resumeRejected(document, second.CheckpointSHA256, second.Checkpoint.RepositoryScopeSHA256, strings.Repeat("f", 64)),
	}
	unknown := append([]byte{}, document...)
	unknown = bytes.Replace(unknown, []byte(`"asOf":`), []byte(`"unknown":true,"asOf":`), 1)
	unknownDigest, _ := adaptivefragment.CanonicalDocumentSHA256(unknown)
	resume.UnknownFieldRejected = resumeRejected(unknown, unknownDigest, second.Checkpoint.RepositoryScopeSHA256, second.Checkpoint.ContextBindingsSHA256)

	third, err := apply(resumed, 3, 200)
	if err != nil {
		return report{}, err
	}
	fourth, err := apply(third.Checkpoint, 4, 200)
	if err != nil {
		return report{}, err
	}
	timeline = append(timeline, timelineFor(fourth))
	fifth, err := apply(fourth.Checkpoint, 5, -1000)
	if err != nil {
		return report{}, err
	}
	timeline = append(timeline, timelineFor(fifth))

	rejected := invalidUpdateRejections(checkpoint)
	result := report{
		SchemaVersion: reportSchema, WorkItem: "AF-006", CapturedAt: "2026-08-25T11:05:00Z",
		Policy: policy{EvidenceSource: adaptivefragment.OrdinaryBuildEvidenceSource, MinimumShadowBuilds: 2,
			MinimumQualificationBuilds: 4, ExactOutputsRequired: true, ComparableCohortRequired: true},
		Timeline: timeline, Resume: resume,
		Regression: regressionProof{
			RegressedFamily: familyA, RegressedNetMs: fifth.Checkpoint.Fragments[0].Assessment.Entry.CumulativeNetMs,
			SuspendedFamilies: fifth.SuspendedFamilies, DependentFamily: familyB,
			UnaffectedFamily: familyC, UnaffectedState: string(fifth.Checkpoint.Fragments[2].State),
			UnaffectedNetMs: fifth.Checkpoint.Fragments[2].Assessment.Entry.CumulativeNetMs,
		},
		Summary: summary{
			RequestedBuilds: fifth.Checkpoint.RequestedBuildCount, AcceptedOrdinaryBuilds: fifth.Checkpoint.RequestedBuildCount,
			ComparableFragmentSamples: fifth.Checkpoint.RequestedBuildCount * uint64(len(fifth.Checkpoint.Fragments)),
			QualifiedBeforeRegression: uint64(len(fourth.QualifiedFamilies)), SuspendedAfterRegression: uint64(len(fifth.SuspendedFamilies)),
			IndependentQualifiedAfter: 1, RejectedInvalidUpdates: rejected, PriorCheckpointMutations: priorMutations,
		},
		FinalState: fifth.Checkpoint,
		Boundaries: boundaries{ProofOfConcept: true, TestOptimization: "OUT_OF_SCOPE"}, Outcome: outcome,
	}
	return result, nil
}

func newCheckpoint() (adaptivefragment.OnlineCheckpoint, error) {
	fragments := []adaptivefragment.OnlineFragment{
		{FamilyID: familyA, RevisionID: strings.Repeat("1", 64), Generation: 1, State: adaptivefragment.StateObserved, EvidenceExpiresAt: "2026-09-25T11:00:00Z"},
		{FamilyID: familyB, RevisionID: strings.Repeat("2", 64), Generation: 1, Requires: []string{familyA}, State: adaptivefragment.StateObserved, EvidenceExpiresAt: "2026-09-25T11:00:00Z"},
		{FamilyID: familyC, RevisionID: strings.Repeat("3", 64), Generation: 1, State: adaptivefragment.StateObserved, EvidenceExpiresAt: "2026-09-25T11:00:00Z"},
	}
	return adaptivefragment.NewOnlineCheckpoint(strings.Repeat("d", 64), strings.Repeat("e", 64), "2026-08-25T11:00:00Z",
		adaptivefragment.EconomicPolicy{DecayPermille: 900, Horizons: []uint64{1, 5, 10}, RegretBudgetMs: 1000}, fragments)
}

func apply(checkpoint adaptivefragment.OnlineCheckpoint, sequence uint64, grossA int64) (adaptivefragment.OnlineUpdate, error) {
	return adaptivefragment.ApplyOrdinaryBuild(checkpoint, ordinaryBuild(checkpoint, sequence, grossA))
}

func ordinaryBuild(checkpoint adaptivefragment.OnlineCheckpoint, sequence uint64, grossA int64) adaptivefragment.OrdinaryBuildUpdate {
	gross := []int64{grossA, 100, 50}
	samples := make([]adaptivefragment.OnlineFragmentSample, len(checkpoint.Fragments))
	for index, fragment := range checkpoint.Fragments {
		samples[index] = adaptivefragment.OnlineFragmentSample{
			FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
			CohortSHA256:           checkpoint.ContextBindingsSHA256,
			EvidenceDocumentSHA256: digest(fmt.Sprintf("evidence-%d-%s", sequence, fragment.FamilyID)),
			Compatible:             true, CandidateValueObserved: true, GrossSavedMs: gross[index],
			SynchronousOverheadMs: 10, ExactOutputs: true,
		}
	}
	return adaptivefragment.OrdinaryBuildUpdate{
		BuildID: digest(fmt.Sprintf("build-%d", sequence)), Sequence: sequence,
		Source:                adaptivefragment.OrdinaryBuildEvidenceSource,
		RepositoryScopeSHA256: checkpoint.RepositoryScopeSHA256, ContextBindingsSHA256: checkpoint.ContextBindingsSHA256,
		ObservedAt: fmt.Sprintf("2026-08-25T11:%02d:00Z", sequence), Samples: samples,
	}
}

func invalidUpdateRejections(checkpoint adaptivefragment.OnlineCheckpoint) uint64 {
	base := ordinaryBuild(checkpoint, 1, 200)
	mutations := []func(*adaptivefragment.OrdinaryBuildUpdate){
		func(build *adaptivefragment.OrdinaryBuildUpdate) { build.MeasurementOnly = true },
		func(build *adaptivefragment.OrdinaryBuildUpdate) {
			build.ContextBindingsSHA256 = strings.Repeat("f", 64)
		},
		func(build *adaptivefragment.OrdinaryBuildUpdate) {
			build.Samples[0].CohortSHA256 = strings.Repeat("f", 64)
		},
		func(build *adaptivefragment.OrdinaryBuildUpdate) { build.Samples[0].ExactOutputs = false },
		func(build *adaptivefragment.OrdinaryBuildUpdate) { build.Samples[0].ProductAttributableFailure = true },
	}
	var rejected uint64
	for _, mutate := range mutations {
		candidate := base
		candidate.Samples = append([]adaptivefragment.OnlineFragmentSample{}, base.Samples...)
		mutate(&candidate)
		if _, err := adaptivefragment.ApplyOrdinaryBuild(checkpoint, candidate); err != nil {
			rejected++
		}
	}
	return rejected
}

func timelineFor(update adaptivefragment.OnlineUpdate) timelineEntry {
	states := map[string]string{}
	for _, fragment := range update.Checkpoint.Fragments {
		states[fragment.FamilyID] = string(fragment.State)
	}
	return timelineEntry{RequestedBuilds: update.Checkpoint.RequestedBuildCount, States: states,
		Qualified: update.QualifiedFamilies, Suspended: update.SuspendedFamilies}
}

func resumeRejected(document []byte, expectedDigest, repository, context string) bool {
	_, err := adaptivefragment.ResumeOnlineCheckpoint(document, expectedDigest, repository, context)
	return err != nil
}

func validateReport(candidate report) error {
	if candidate.SchemaVersion != reportSchema || candidate.WorkItem != "AF-006" || candidate.Outcome != outcome ||
		candidate.CapturedAt != "2026-08-25T11:05:00Z" || candidate.Policy.EvidenceSource != adaptivefragment.OrdinaryBuildEvidenceSource ||
		candidate.Policy.MinimumShadowBuilds != 2 || candidate.Policy.MinimumQualificationBuilds != 4 ||
		!candidate.Policy.ExactOutputsRequired || !candidate.Policy.ComparableCohortRequired || len(candidate.Timeline) != 4 {
		return errors.New("online-learning report identity is invalid")
	}
	if candidate.Timeline[0].States[familyA] != "OBSERVED" || candidate.Timeline[1].States[familyA] != "SHADOW" ||
		candidate.Timeline[2].States[familyA] != "QUALIFIED" || candidate.Timeline[3].States[familyA] != "SUSPENDED" ||
		candidate.Timeline[3].States[familyB] != "SUSPENDED" || candidate.Timeline[3].States[familyC] != "QUALIFIED" {
		return errors.New("online-learning lifecycle evidence is invalid")
	}
	if !candidate.Resume.ExactResumeAccepted || !candidate.Resume.DigestMismatchRejected ||
		!candidate.Resume.RepositoryMismatchRejected || !candidate.Resume.BindingMismatchRejected || !candidate.Resume.UnknownFieldRejected ||
		candidate.Resume.InterruptedGeneration != 3 || !isSHA(candidate.Resume.InterruptedCheckpointSHA256) {
		return errors.New("online-learning resume evidence is invalid")
	}
	if candidate.Regression.RegressedFamily != familyA || candidate.Regression.RegressedNetMs != -250 ||
		!reflect.DeepEqual(candidate.Regression.SuspendedFamilies, []string{familyA, familyB}) ||
		candidate.Regression.DependentFamily != familyB || candidate.Regression.UnaffectedFamily != familyC ||
		candidate.Regression.UnaffectedState != "QUALIFIED" || candidate.Regression.UnaffectedNetMs != 200 ||
		candidate.Regression.UnrelatedSuspensionCount != 0 {
		return errors.New("online-learning regression evidence is invalid")
	}
	if candidate.Summary != (summary{RequestedBuilds: 5, AcceptedOrdinaryBuilds: 5, ComparableFragmentSamples: 15,
		QualifiedBeforeRegression: 3, SuspendedAfterRegression: 2, IndependentQualifiedAfter: 1,
		RejectedInvalidUpdates: 5}) {
		return errors.New("online-learning summary is invalid")
	}
	if !candidate.Boundaries.ProofOfConcept || candidate.Boundaries.SyntheticTimingClaim || candidate.Boundaries.GradleExecutions != 0 ||
		candidate.Boundaries.ActivationAuthorized || candidate.Boundaries.ProductionAuthorized || candidate.Boundaries.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("online-learning boundaries are invalid")
	}
	return nil
}

func readJSONStrict(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON documents are not allowed")
	}
	return nil
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte("buildopt-af006-v1\x00" + value))
	return hex.EncodeToString(sum[:])
}

func isSHA(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
