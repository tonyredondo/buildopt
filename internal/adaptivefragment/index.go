package adaptivefragment

import (
	"errors"
	"sort"
	"time"
)

const CompatibilityIndexSchemaVersion = "buildopt.adaptive/compatibility-index/v1"

// FingerprintSnapshot contains the repository-local facts available before
// Gradle starts. Git revision is provenance, not a compatibility binding:
// fragments remain reusable across commits when their declared facts match.
type FingerprintSnapshot struct {
	RepositoryID             string
	GitRevisionSHA256        string
	WrapperSHA256            string
	WorkflowSHA256           string
	ProducerLineageSHA256    string
	OutputContractSHA256     string
	ChangeFamilySHA256       string
	TaskImplementationSHA256 string
	PlatformSHA256           string
	NetworkClassSHA256       string
	CacheNamespaceSHA256     string
	PatchBaseSHA256          string
	Ambiguous                []BindingKey
	ObservedAt               string
}

// CompatibilityIndex is a derived, repository-local lookup cache. The
// fragment documents remain the authority; this index may be discarded and
// rebuilt without changing lifecycle state.
type CompatibilityIndex struct {
	SchemaVersion         string              `json:"schemaVersion"`
	RecordType            string              `json:"recordType"`
	RepositoryScopeSHA256 string              `json:"repositoryScopeSha256"`
	Generation            uint64              `json:"generation"`
	SourceRevisionSHA256  string              `json:"sourceRevisionSha256"`
	Fragments             []PersistedFragment `json:"fragments"`
	CreatedAt             string              `json:"createdAt"`
}

// LookupDisposition is the pre-Gradle decision for one fragment or index.
type LookupDisposition string

const (
	DispositionCompatible     LookupDisposition = "COMPATIBLE"
	DispositionSuspended      LookupDisposition = "SUSPENDED"
	DispositionNativeRetained LookupDisposition = "NATIVE_RETAINED"
)

// FragmentDecision records why one indexed fragment was retained, suspended,
// or returned as a compatible candidate.
type FragmentDecision struct {
	FamilyID    string            `json:"familyId"`
	RevisionID  string            `json:"revisionId"`
	Disposition LookupDisposition `json:"disposition"`
	Reason      string            `json:"reason"`
	Bindings    []BindingKey      `json:"bindings"`
}

// LookupResult is a pure pre-Gradle decision. Compatible families are inputs
// to later shadow/economic/planning gates and are not activation authority.
type LookupResult struct {
	Disposition        LookupDisposition  `json:"disposition"`
	Reason             string             `json:"reason"`
	Decisions          []FragmentDecision `json:"decisions"`
	CompatibleFamilies []string           `json:"compatibleFamilies"`
	SuspendedFamilies  []string           `json:"suspendedFamilies"`
	RetainedFamilies   []string           `json:"retainedFamilies"`
}

// NewCompatibilityIndex builds one canonical derived index from already
// validated fragment generations and a repository-local fingerprint snapshot.
func NewCompatibilityIndex(snapshot FingerprintSnapshot, generation uint64, fragments []PersistedFragment) (CompatibilityIndex, error) {
	context, observedAt, err := snapshotContext(snapshot)
	if err != nil {
		return CompatibilityIndex{}, err
	}
	if generation == 0 || len(fragments) == 0 {
		return CompatibilityIndex{}, errors.New("adaptive compatibility index input is incomplete")
	}
	index := CompatibilityIndex{
		SchemaVersion:         CompatibilityIndexSchemaVersion,
		RecordType:            "ADAPTIVE_FRAGMENT_COMPATIBILITY_INDEX",
		RepositoryScopeSHA256: digest("buildopt-adaptive-fragment-repository-v1", context.RepositoryID),
		Generation:            generation,
		SourceRevisionSHA256:  snapshot.GitRevisionSHA256,
		Fragments:             clonePersistedFragments(fragments),
		CreatedAt:             observedAt.Format(time.RFC3339Nano),
	}
	sort.Slice(index.Fragments, func(left, right int) bool {
		return index.Fragments[left].FamilyID < index.Fragments[right].FamilyID
	})
	if err := validateCompatibilityIndex(index); err != nil {
		return CompatibilityIndex{}, err
	}
	return index, nil
}

// Lookup returns fragment candidates without starting Gradle, performing
// remote calls, materializing outputs, or mutating lifecycle state.
func Lookup(index CompatibilityIndex, snapshot FingerprintSnapshot) LookupResult {
	if err := validateCompatibilityIndex(index); err != nil {
		return retainedLookup("INDEX_INVALID", nil)
	}
	context, observedAt, err := snapshotContext(snapshot)
	if err != nil {
		return retainedLookup("FINGERPRINT_INVALID", nil)
	}
	if digest("buildopt-adaptive-fragment-repository-v1", context.RepositoryID) != index.RepositoryScopeSHA256 {
		return retainedLookup("REPOSITORY_SCOPE_MISMATCH", nil)
	}
	createdAt, _ := parseUTC(index.CreatedAt)
	if observedAt.Before(createdAt) {
		return retainedLookup("FINGERPRINT_PREDATES_INDEX", nil)
	}

	result := LookupResult{Decisions: make([]FragmentDecision, 0, len(index.Fragments))}
	for _, fragment := range index.Fragments {
		decision := evaluateIndexedFragment(fragment, context, observedAt)
		result.Decisions = append(result.Decisions, decision)
		switch decision.Disposition {
		case DispositionCompatible:
			result.CompatibleFamilies = append(result.CompatibleFamilies, decision.FamilyID)
		case DispositionSuspended:
			result.SuspendedFamilies = append(result.SuspendedFamilies, decision.FamilyID)
		default:
			result.RetainedFamilies = append(result.RetainedFamilies, decision.FamilyID)
		}
	}
	if len(result.CompatibleFamilies) > 0 {
		result.Disposition = DispositionCompatible
		result.Reason = "COMPATIBLE_FRAGMENT_CANDIDATES"
	} else if len(result.SuspendedFamilies) > 0 {
		result.Disposition = DispositionSuspended
		result.Reason = "DECLARED_BINDING_DRIFT"
	} else {
		result.Disposition = DispositionNativeRetained
		result.Reason = "NO_COMPATIBLE_FRAGMENT"
	}
	return result
}

func evaluateIndexedFragment(fragment PersistedFragment, context Context, observedAt time.Time) FragmentDecision {
	decision := FragmentDecision{FamilyID: fragment.FamilyID, RevisionID: fragment.RevisionID, Bindings: []BindingKey{}}
	expiresAt, err := parseUTC(fragment.EvidenceExpiresAt)
	if err != nil || !expiresAt.After(observedAt) || fragment.State == StateExpired {
		decision.Disposition = DispositionNativeRetained
		decision.Reason = "EVIDENCE_EXPIRED"
		return decision
	}
	if fragment.State == StateSuspended {
		decision.Disposition = DispositionNativeRetained
		decision.Reason = "LIFECYCLE_SUSPENDED"
		return decision
	}
	compatibility, err := Evaluate(persistedFragmentContract(fragment), context)
	if err != nil {
		decision.Disposition = DispositionNativeRetained
		decision.Reason = "CONTEXT_INVALID"
		return decision
	}
	decision.Bindings = append(decision.Bindings, compatibility.Bindings...)
	if compatibility.Compatible {
		decision.Disposition = DispositionCompatible
		decision.Reason = compatibility.Reason
		return decision
	}
	if compatibility.Reason == "BINDING_DRIFT" {
		decision.Disposition = DispositionSuspended
		decision.Reason = compatibility.Reason
		return decision
	}
	decision.Disposition = DispositionNativeRetained
	decision.Reason = compatibility.Reason
	return decision
}

func validateCompatibilityIndex(index CompatibilityIndex) error {
	if index.SchemaVersion != CompatibilityIndexSchemaVersion ||
		index.RecordType != "ADAPTIVE_FRAGMENT_COMPATIBILITY_INDEX" ||
		!validSHA(index.RepositoryScopeSHA256) || index.Generation == 0 ||
		!validSHA(index.SourceRevisionSHA256) || len(index.Fragments) == 0 || len(index.Fragments) > 4096 {
		return errors.New("adaptive compatibility index identity is invalid")
	}
	createdAt, err := parseUTC(index.CreatedAt)
	if err != nil {
		return errors.New("adaptive compatibility index time is invalid")
	}
	seen := map[string]bool{}
	previousFamily := ""
	for _, fragment := range index.Fragments {
		if err := validatePersistedFragment(fragment); err != nil || fragment.RepositoryScopeSHA256 != index.RepositoryScopeSHA256 {
			return errors.New("adaptive compatibility index fragment is invalid")
		}
		updatedAt, _ := parseUTC(fragment.UpdatedAt)
		if updatedAt.After(createdAt) || seen[fragment.FamilyID] || (previousFamily != "" && fragment.FamilyID < previousFamily) {
			return errors.New("adaptive compatibility index is not canonical")
		}
		seen[fragment.FamilyID] = true
		previousFamily = fragment.FamilyID
	}
	return nil
}

func snapshotContext(snapshot FingerprintSnapshot) (Context, time.Time, error) {
	if snapshot.RepositoryID == "" || !validSHA(snapshot.GitRevisionSHA256) {
		return Context{}, time.Time{}, errors.New("adaptive compatibility fingerprint identity is invalid")
	}
	observedAt, err := parseUTC(snapshot.ObservedAt)
	if err != nil {
		return Context{}, time.Time{}, errors.New("adaptive compatibility fingerprint time is invalid")
	}
	bindings := map[BindingKey]string{}
	available := map[BindingKey]string{
		BindingWrapper:         snapshot.WrapperSHA256,
		BindingWorkflow:        snapshot.WorkflowSHA256,
		BindingProducerLineage: snapshot.ProducerLineageSHA256,
		BindingOutputContract:  snapshot.OutputContractSHA256,
		BindingChangeFamily:    snapshot.ChangeFamilySHA256,
	}
	optional := map[BindingKey]string{
		BindingTaskImplementation: snapshot.TaskImplementationSHA256,
		BindingPlatform:           snapshot.PlatformSHA256,
		BindingNetworkClass:       snapshot.NetworkClassSHA256,
		BindingCacheNamespace:     snapshot.CacheNamespaceSHA256,
		BindingPatchBase:          snapshot.PatchBaseSHA256,
	}
	for key, value := range optional {
		available[key] = value
	}
	for key, value := range available {
		if value == "" {
			continue
		}
		if !validSHA(value) {
			return Context{}, time.Time{}, errors.New("adaptive compatibility fingerprint binding is invalid")
		}
		bindings[key] = value
	}
	return Context{
		RepositoryID: snapshot.RepositoryID,
		Bindings:     bindings,
		Ambiguous:    append([]BindingKey{}, snapshot.Ambiguous...),
	}, observedAt, nil
}

func retainedLookup(reason string, decisions []FragmentDecision) LookupResult {
	return LookupResult{
		Disposition:        DispositionNativeRetained,
		Reason:             reason,
		Decisions:          decisions,
		CompatibleFamilies: []string{},
		SuspendedFamilies:  []string{},
		RetainedFamilies:   []string{},
	}
}

func clonePersistedFragments(fragments []PersistedFragment) []PersistedFragment {
	result := make([]PersistedFragment, len(fragments))
	for index, fragment := range fragments {
		result[index] = fragment
		result[index].Bindings = make(map[BindingKey]string, len(fragment.Bindings))
		for key, value := range fragment.Bindings {
			result[index].Bindings[key] = value
		}
		result[index].Requires = append([]string{}, fragment.Requires...)
		result[index].ConflictsWith = append([]string{}, fragment.ConflictsWith...)
	}
	return result
}
