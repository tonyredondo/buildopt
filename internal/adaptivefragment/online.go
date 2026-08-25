package adaptivefragment

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"
)

const (
	OnlineCheckpointSchemaVersion = "buildopt.adaptive/online-checkpoint/v1"
	OrdinaryBuildEvidenceSource   = "REQUESTED_ORDINARY_BUILD"
	minimumShadowBuilds           = 2
	minimumQualificationBuilds    = 4
)

// OnlineFragment is the resumable learning state for one exact fragment
// revision. Requires contains family identities from the same checkpoint.
type OnlineFragment struct {
	FamilyID          string                `json:"familyId"`
	RevisionID        string                `json:"revisionId"`
	Generation        uint64                `json:"generation"`
	Requires          []string              `json:"requires"`
	State             State                 `json:"state"`
	SuspensionReason  string                `json:"suspensionReason,omitempty"`
	EvidenceExpiresAt string                `json:"evidenceExpiresAt"`
	Observations      []EconomicObservation `json:"observations"`
	Assessment        *EconomicAssessment   `json:"assessment,omitempty"`
}

// OnlineCheckpoint is one immutable repository-scoped learner generation.
// Its digest is carried externally so the document never hashes itself.
type OnlineCheckpoint struct {
	SchemaVersion         string           `json:"schemaVersion"`
	RecordType            string           `json:"recordType"`
	RepositoryScopeSHA256 string           `json:"repositoryScopeSha256"`
	ContextBindingsSHA256 string           `json:"contextBindingsSha256"`
	Generation            uint64           `json:"generation"`
	SupersedesSHA256      string           `json:"supersedesSha256,omitempty"`
	RequestedBuildCount   uint64           `json:"requestedBuildCount"`
	LastBuildID           string           `json:"lastBuildId,omitempty"`
	AsOf                  string           `json:"asOf"`
	Policy                EconomicPolicy   `json:"policy"`
	Fragments             []OnlineFragment `json:"fragments"`
}

// OnlineFragmentSample is the evidence contributed by one requested build to
// one fragment. CandidateValueObserved is evidence only; it grants no runtime
// activation authority.
type OnlineFragmentSample struct {
	FamilyID                   string
	RevisionID                 string
	CohortSHA256               string
	EvidenceDocumentSHA256     string
	Compatible                 bool
	CandidateValueObserved     bool
	GrossSavedMs               int64
	SynchronousOverheadMs      uint64
	AsynchronousCostEvents     []EconomicCostEvent
	ExactOutputs               bool
	ProductAttributableFailure bool
}

// OrdinaryBuildUpdate is one complete customer-requested build. Measurement
// builds are rejected rather than silently excluded from the checkpoint.
type OrdinaryBuildUpdate struct {
	BuildID               string
	Sequence              uint64
	Source                string
	RepositoryScopeSHA256 string
	ContextBindingsSHA256 string
	ObservedAt            string
	MeasurementOnly       bool
	Samples               []OnlineFragmentSample
}

// OnlineUpdate reports the new immutable checkpoint and only the lifecycle
// changes caused by the accepted requested build.
type OnlineUpdate struct {
	Checkpoint        OnlineCheckpoint
	CheckpointSHA256  string
	QualifiedFamilies []string
	SuspendedFamilies []string
}

// NewOnlineCheckpoint validates and freezes an empty learner generation.
func NewOnlineCheckpoint(repositoryScope, contextBindings, asOf string, policy EconomicPolicy, fragments []OnlineFragment) (OnlineCheckpoint, error) {
	checkpoint := OnlineCheckpoint{
		SchemaVersion: OnlineCheckpointSchemaVersion, RecordType: "ADAPTIVE_FRAGMENT_ONLINE_CHECKPOINT",
		RepositoryScopeSHA256: repositoryScope, ContextBindingsSHA256: contextBindings,
		Generation: 1, AsOf: asOf, Policy: policy, Fragments: cloneOnlineFragments(fragments),
	}
	if err := validateOnlineCheckpoint(checkpoint); err != nil {
		return OnlineCheckpoint{}, err
	}
	return checkpoint, nil
}

// ApplyOrdinaryBuild returns a new checkpoint. The supplied checkpoint is not
// mutated, so a process interruption leaves the previous generation usable.
func ApplyOrdinaryBuild(checkpoint OnlineCheckpoint, build OrdinaryBuildUpdate) (OnlineUpdate, error) {
	if err := validateOnlineCheckpoint(checkpoint); err != nil {
		return OnlineUpdate{}, err
	}
	if err := validateOrdinaryBuild(checkpoint, build); err != nil {
		return OnlineUpdate{}, err
	}
	previousDigest, err := onlineCheckpointSHA256(checkpoint)
	if err != nil {
		return OnlineUpdate{}, err
	}
	next := checkpoint
	next.Generation++
	next.SupersedesSHA256 = previousDigest
	next.RequestedBuildCount++
	next.LastBuildID = build.BuildID
	next.AsOf = build.ObservedAt
	next.Fragments = cloneOnlineFragments(checkpoint.Fragments)

	newlySuspended := map[string]bool{}
	newlyQualified := map[string]bool{}
	for index, sample := range build.Samples {
		fragment := &next.Fragments[index]
		if fragment.State == StateSuspended || fragment.State == StateExpired {
			continue
		}
		if !sample.Compatible {
			newlySuspended[fragment.FamilyID] = true
			fragment.SuspensionReason = "BINDING_INCOMPATIBLE"
			continue
		}
		observation := EconomicObservation{
			ObservationID: sample.EvidenceDocumentSHA256, Sequence: build.Sequence,
			ObservedAt: build.ObservedAt, Compatible: true,
			Activated: sample.CandidateValueObserved, GrossSavedMs: sample.GrossSavedMs,
			SynchronousOverheadMs:  sample.SynchronousOverheadMs,
			AsynchronousCostEvents: append([]EconomicCostEvent{}, sample.AsynchronousCostEvents...),
		}
		fragment.Observations = append(fragment.Observations, observation)
		assessment, assessmentErr := AssessEconomics(EconomicSeries{
			Scope: EconomicScopeFragment, FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
			FragmentGeneration: fragment.Generation, EvidenceExpiresAt: fragment.EvidenceExpiresAt,
			Observations: fragment.Observations, Policy: next.Policy,
		})
		if assessmentErr != nil {
			return OnlineUpdate{}, assessmentErr
		}
		fragment.Assessment = &assessment
		previousState := fragment.State
		switch {
		case previousState == StateQualified && assessment.Entry.CumulativeNetMs < 0:
			newlySuspended[fragment.FamilyID] = true
			fragment.SuspensionReason = "VALUE_REGRESSION"
		case previousState == StateShadow && len(fragment.Observations) >= minimumQualificationBuilds &&
			assessment.Entry.CumulativeNetMs > 0 && assessment.Recurrence.ActivatedBuilds >= minimumQualificationBuilds:
			fragment.State = StateQualified
			newlyQualified[fragment.FamilyID] = true
		case previousState == StateObserved && len(fragment.Observations) >= minimumShadowBuilds:
			fragment.State = StateShadow
		}
	}
	propagateSuspension(next.Fragments, newlySuspended)
	for index := range next.Fragments {
		if newlySuspended[next.Fragments[index].FamilyID] {
			next.Fragments[index].State = StateSuspended
			if next.Fragments[index].SuspensionReason == "" {
				next.Fragments[index].SuspensionReason = "DEPENDENCY_SUSPENDED"
			}
			delete(newlyQualified, next.Fragments[index].FamilyID)
		}
	}
	if err := validateOnlineCheckpoint(next); err != nil {
		return OnlineUpdate{}, err
	}
	digest, err := onlineCheckpointSHA256(next)
	if err != nil {
		return OnlineUpdate{}, err
	}
	return OnlineUpdate{
		Checkpoint: next, CheckpointSHA256: digest,
		QualifiedFamilies: sortedSet(newlyQualified), SuspendedFamilies: sortedSet(newlySuspended),
	}, nil
}

// MarshalOnlineCheckpoint returns canonical JSON suitable for atomic storage.
func MarshalOnlineCheckpoint(checkpoint OnlineCheckpoint) ([]byte, error) {
	if err := validateOnlineCheckpoint(checkpoint); err != nil {
		return nil, err
	}
	return MarshalCanonicalDocument(checkpoint)
}

// ResumeOnlineCheckpoint accepts only the exact canonical checkpoint digest,
// repository scope and context binding expected by the caller.
func ResumeOnlineCheckpoint(document []byte, expectedDigest, repositoryScope, contextBindings string) (OnlineCheckpoint, error) {
	actualDigest, err := CanonicalDocumentSHA256(document)
	if err != nil || actualDigest != expectedDigest {
		return OnlineCheckpoint{}, errors.New("adaptive fragment online checkpoint digest is incompatible")
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	var checkpoint OnlineCheckpoint
	if err := decoder.Decode(&checkpoint); err != nil {
		return OnlineCheckpoint{}, errors.New("adaptive fragment online checkpoint document is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return OnlineCheckpoint{}, errors.New("adaptive fragment online checkpoint contains multiple documents")
	}
	if checkpoint.RepositoryScopeSHA256 != repositoryScope || checkpoint.ContextBindingsSHA256 != contextBindings {
		return OnlineCheckpoint{}, errors.New("adaptive fragment online checkpoint binding is incompatible")
	}
	if err := validateOnlineCheckpoint(checkpoint); err != nil {
		return OnlineCheckpoint{}, err
	}
	return checkpoint, nil
}

func validateOnlineCheckpoint(checkpoint OnlineCheckpoint) error {
	if checkpoint.SchemaVersion != OnlineCheckpointSchemaVersion || checkpoint.RecordType != "ADAPTIVE_FRAGMENT_ONLINE_CHECKPOINT" ||
		!validSHA(checkpoint.RepositoryScopeSHA256) || !validSHA(checkpoint.ContextBindingsSHA256) || checkpoint.Generation == 0 ||
		checkpoint.Generation != checkpoint.RequestedBuildCount+1 || len(checkpoint.Fragments) == 0 {
		return errors.New("adaptive fragment online checkpoint identity is invalid")
	}
	if checkpoint.Generation == 1 {
		if checkpoint.SupersedesSHA256 != "" || checkpoint.LastBuildID != "" {
			return errors.New("adaptive fragment online checkpoint ancestry is invalid")
		}
	} else if !validSHA(checkpoint.SupersedesSHA256) || !validSHA(checkpoint.LastBuildID) {
		return errors.New("adaptive fragment online checkpoint ancestry is invalid")
	}
	asOf, err := parseUTC(checkpoint.AsOf)
	if err != nil {
		return errors.New("adaptive fragment online checkpoint time is invalid")
	}
	if checkpoint.Policy.DecayPermille > 1000 || len(checkpoint.Policy.Horizons) == 0 {
		return errors.New("adaptive fragment online checkpoint policy is invalid")
	}
	if !sort.SliceIsSorted(checkpoint.Fragments, func(left, right int) bool {
		return checkpoint.Fragments[left].FamilyID < checkpoint.Fragments[right].FamilyID
	}) {
		return errors.New("adaptive fragment online fragments are not canonical")
	}
	known := map[string]bool{}
	states := map[string]State{}
	for _, fragment := range checkpoint.Fragments {
		if !validSHA(fragment.FamilyID) || !validSHA(fragment.RevisionID) || fragment.Generation == 0 || known[fragment.FamilyID] ||
			(fragment.State != StateObserved && fragment.State != StateShadow && fragment.State != StateQualified && fragment.State != StateSuspended) ||
			!sort.StringsAreSorted(fragment.Requires) {
			return errors.New("adaptive fragment online fragment is invalid")
		}
		if fragment.State == StateSuspended {
			if fragment.SuspensionReason != "BINDING_INCOMPATIBLE" && fragment.SuspensionReason != "VALUE_REGRESSION" &&
				fragment.SuspensionReason != "DEPENDENCY_SUSPENDED" {
				return errors.New("adaptive fragment online suspension reason is invalid")
			}
		} else if fragment.SuspensionReason != "" {
			return errors.New("adaptive fragment online suspension reason is invalid")
		}
		known[fragment.FamilyID] = true
		states[fragment.FamilyID] = fragment.State
	}
	for _, fragment := range checkpoint.Fragments {
		seenRequires := map[string]bool{}
		hasSuspendedRequirement := false
		for _, required := range fragment.Requires {
			if !known[required] || required == fragment.FamilyID || seenRequires[required] {
				return errors.New("adaptive fragment online dependency is invalid")
			}
			seenRequires[required] = true
			if states[required] == StateSuspended {
				hasSuspendedRequirement = true
			}
		}
		if fragment.State != StateSuspended && hasSuspendedRequirement {
			return errors.New("adaptive fragment online dependency suspension is inconsistent")
		}
		if fragment.SuspensionReason == "DEPENDENCY_SUSPENDED" && !hasSuspendedRequirement {
			return errors.New("adaptive fragment online dependency suspension is unsupported")
		}
		expiresAt, expiryErr := parseUTC(fragment.EvidenceExpiresAt)
		if expiryErr != nil || !expiresAt.After(asOf) || uint64(len(fragment.Observations)) > checkpoint.RequestedBuildCount ||
			(fragment.State != StateSuspended && uint64(len(fragment.Observations)) != checkpoint.RequestedBuildCount) {
			return errors.New("adaptive fragment online evidence is invalid")
		}
		if len(fragment.Observations) == 0 {
			if fragment.Assessment != nil || fragment.State != StateObserved {
				return errors.New("adaptive fragment online empty state is invalid")
			}
			continue
		}
		assessment, assessmentErr := AssessEconomics(EconomicSeries{
			Scope: EconomicScopeFragment, FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID,
			FragmentGeneration: fragment.Generation, EvidenceExpiresAt: fragment.EvidenceExpiresAt,
			Observations: fragment.Observations, Policy: checkpoint.Policy,
		})
		if assessmentErr != nil || fragment.Assessment == nil || !reflect.DeepEqual(*fragment.Assessment, assessment) {
			return errors.New("adaptive fragment online assessment is invalid")
		}
		if fragment.State != StateSuspended && fragment.Assessment.Entry.LastObservedAt != checkpoint.AsOf {
			return errors.New("adaptive fragment online current observation is stale")
		}
		if fragment.SuspensionReason == "VALUE_REGRESSION" && fragment.Assessment.Entry.CumulativeNetMs >= 0 {
			return errors.New("adaptive fragment online value suspension is unsupported")
		}
		switch fragment.State {
		case StateObserved:
			if len(fragment.Observations) >= minimumShadowBuilds {
				return errors.New("adaptive fragment online observed state has sufficient shadow evidence")
			}
		case StateShadow:
			if len(fragment.Observations) < minimumShadowBuilds ||
				(len(fragment.Observations) >= minimumQualificationBuilds && assessment.Entry.CumulativeNetMs > 0 &&
					assessment.Recurrence.ActivatedBuilds >= minimumQualificationBuilds) {
				return errors.New("adaptive fragment online shadow state is inconsistent")
			}
		case StateQualified:
			if len(fragment.Observations) < minimumQualificationBuilds || assessment.Entry.CumulativeNetMs <= 0 ||
				assessment.Recurrence.ActivatedBuilds < minimumQualificationBuilds {
				return errors.New("adaptive fragment online qualification is unsupported")
			}
		}
	}
	if onlineDependencyCycle(checkpoint.Fragments) {
		return errors.New("adaptive fragment online dependency graph is cyclic")
	}
	return nil
}

func validateOrdinaryBuild(checkpoint OnlineCheckpoint, build OrdinaryBuildUpdate) error {
	observedAt, err := parseUTC(build.ObservedAt)
	checkpointAt, checkpointErr := parseUTC(checkpoint.AsOf)
	if err != nil || checkpointErr != nil || !observedAt.After(checkpointAt) || !validSHA(build.BuildID) ||
		build.Sequence != checkpoint.RequestedBuildCount+1 || build.Source != OrdinaryBuildEvidenceSource || build.MeasurementOnly ||
		build.RepositoryScopeSHA256 != checkpoint.RepositoryScopeSHA256 || build.ContextBindingsSHA256 != checkpoint.ContextBindingsSHA256 ||
		len(build.Samples) != len(checkpoint.Fragments) {
		return errors.New("adaptive fragment ordinary build is invalid")
	}
	for index, sample := range build.Samples {
		fragment := checkpoint.Fragments[index]
		if sample.FamilyID != fragment.FamilyID || sample.RevisionID != fragment.RevisionID ||
			sample.CohortSHA256 != checkpoint.ContextBindingsSHA256 || !validSHA(sample.EvidenceDocumentSHA256) ||
			!sample.ExactOutputs || sample.ProductAttributableFailure ||
			(fragment.State == StateSuspended && (sample.CandidateValueObserved || sample.GrossSavedMs != 0)) ||
			(!sample.CandidateValueObserved && sample.GrossSavedMs != 0) ||
			(!sample.Compatible && (sample.CandidateValueObserved || sample.GrossSavedMs != 0)) {
			return errors.New("adaptive fragment ordinary build sample is invalid")
		}
	}
	return nil
}

func onlineCheckpointSHA256(checkpoint OnlineCheckpoint) (string, error) {
	document, err := MarshalCanonicalDocument(checkpoint)
	if err != nil {
		return "", err
	}
	return CanonicalDocumentSHA256(document)
}

func propagateSuspension(fragments []OnlineFragment, suspended map[string]bool) {
	changed := true
	for changed {
		changed = false
		for _, fragment := range fragments {
			if suspended[fragment.FamilyID] {
				continue
			}
			for _, required := range fragment.Requires {
				if suspended[required] {
					suspended[fragment.FamilyID] = true
					changed = true
					break
				}
			}
		}
	}
}

func cloneOnlineFragments(fragments []OnlineFragment) []OnlineFragment {
	result := make([]OnlineFragment, len(fragments))
	for index, fragment := range fragments {
		result[index] = fragment
		result[index].Requires = append([]string{}, fragment.Requires...)
		result[index].Observations = append([]EconomicObservation{}, fragment.Observations...)
		for observationIndex := range result[index].Observations {
			result[index].Observations[observationIndex].AsynchronousCostEvents = append([]EconomicCostEvent{}, fragment.Observations[observationIndex].AsynchronousCostEvents...)
		}
		if fragment.Assessment != nil {
			assessment := *fragment.Assessment
			assessment.Projections = append([]EconomicProjection{}, fragment.Assessment.Projections...)
			result[index].Assessment = &assessment
		}
	}
	return result
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func onlineDependencyCycle(fragments []OnlineFragment) bool {
	requires := make(map[string][]string, len(fragments))
	for _, fragment := range fragments {
		requires[fragment.FamilyID] = fragment.Requires
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(family string) bool {
		if visiting[family] {
			return true
		}
		if visited[family] {
			return false
		}
		visiting[family] = true
		for _, required := range requires[family] {
			if visit(required) {
				return true
			}
		}
		visiting[family] = false
		visited[family] = true
		return false
	}
	for family := range requires {
		if visit(family) {
			return true
		}
	}
	return false
}
