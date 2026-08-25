// Package adaptivefragment defines the POC contracts for independently
// compatible optimization fragments, their immutable typed state and a cheap
// repository-local compatibility index, frozen-history shadow replay, signed
// economic assessment, requested-build online learning, conflict-aware
// pre-Gradle planning and exact producer-local Build Impact activation.
// Synchronization and timed multi-mechanism composition belong to later
// adaptive-fragment blocks.
package adaptivefragment

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"strings"
)

// SchemaVersion identifies the in-memory AF-001 identity contract. Persisted
// AF-002 records and the derived AF-003 index use their own versions.
const SchemaVersion = "buildopt.poc/adaptive-fragment-contract/v1"

// Kind identifies one independently invalidated optimization class.
type Kind string

const (
	KindSubgraph              Kind = "SUBGRAPH"
	KindOutputMaterialization Kind = "OUTPUT_MATERIALIZATION"
	KindTaskContract          Kind = "TASK_CONTRACT"
	KindPatch                 Kind = "PATCH"
	KindCacheLocality         Kind = "CACHE_LOCALITY"
)

// Authority identifies the external correctness basis for a fragment.
type Authority string

const (
	AuthorityGradleModel      Authority = "GRADLE_MODEL"
	AuthorityGradleNative     Authority = "GRADLE_NATIVE"
	AuthorityReviewedAdapter  Authority = "REVIEWED_ADAPTER"
	AuthorityReviewedPatch    Authority = "REVIEWED_PATCH"
	AuthorityVerifiedProducer Authority = "VERIFIED_PRODUCER"
)

// BindingKey identifies one semantic compatibility dimension.
type BindingKey string

const (
	BindingWorkflow           BindingKey = "WORKFLOW"
	BindingWrapper            BindingKey = "GRADLE_WRAPPER"
	BindingTaskImplementation BindingKey = "TASK_IMPLEMENTATION"
	BindingProducerLineage    BindingKey = "PRODUCER_LINEAGE"
	BindingOutputContract     BindingKey = "OUTPUT_CONTRACT"
	BindingChangeFamily       BindingKey = "CHANGE_FAMILY"
	BindingPlatform           BindingKey = "PLATFORM"
	BindingNetworkClass       BindingKey = "NETWORK_CLASS"
	BindingCacheNamespace     BindingKey = "CACHE_NAMESPACE"
	BindingPatchBase          BindingKey = "PATCH_BASE"
)

// State identifies a fragment's qualification lifecycle state.
type State string

const (
	StateObserved  State = "OBSERVED"
	StateShadow    State = "SHADOW"
	StateQualified State = "QUALIFIED"
	StateActive    State = "ACTIVE"
	StateSuspended State = "SUSPENDED"
	StateExpired   State = "EXPIRED"
)

// Input contains the authority and bindings used to derive canonical fragment
// identities. RepositoryID scopes data but is never interpreted as behavior.
type Input struct {
	RepositoryID    string
	Kind            Kind
	Selector        []string
	Authority       Authority
	AuthoritySHA256 string
	Bindings        map[BindingKey]string
	Requires        []string
	ConflictsWith   []string
}

// Fragment separates a logical family identity from a binding-specific
// revision. FamilyID remains stable when only evidence bindings change;
// RevisionID changes whenever authority, applicability, dependency or conflict
// evidence changes.
type Fragment struct {
	SchemaVersion         string
	FamilyID              string
	RevisionID            string
	RepositoryScopeSHA256 string
	Kind                  Kind
	SelectorSHA256        string
	Authority             Authority
	AuthoritySHA256       string
	Bindings              map[BindingKey]string
	Requires              []string
	ConflictsWith         []string
}

// Context contains the current repository scope and observed semantic
// bindings. Ambiguous bindings fail closed only for fragments that declare
// them.
type Context struct {
	RepositoryID string
	Bindings     map[BindingKey]string
	Ambiguous    []BindingKey
}

// Compatibility reports whether one exact fragment revision can be considered
// by later planning and, when not, the deterministic native-retention reason.
type Compatibility struct {
	Compatible bool
	Reason     string
	Bindings   []BindingKey
}

var requiredBindings = map[Kind][]BindingKey{
	KindSubgraph: {
		BindingWorkflow,
		BindingWrapper,
		BindingProducerLineage,
		BindingOutputContract,
		BindingChangeFamily,
	},
	KindOutputMaterialization: {
		BindingWrapper,
		BindingProducerLineage,
		BindingOutputContract,
	},
	KindTaskContract: {
		BindingWrapper,
		BindingTaskImplementation,
		BindingOutputContract,
	},
	KindPatch: {
		BindingTaskImplementation,
		BindingOutputContract,
		BindingPatchBase,
	},
	KindCacheLocality: {
		BindingWrapper,
		BindingNetworkClass,
		BindingCacheNamespace,
	},
}

var knownBindings = map[BindingKey]bool{
	BindingWorkflow: true, BindingWrapper: true,
	BindingTaskImplementation: true, BindingProducerLineage: true,
	BindingOutputContract: true, BindingChangeFamily: true,
	BindingPlatform: true, BindingNetworkClass: true,
	BindingCacheNamespace: true, BindingPatchBase: true,
}

// Derive validates an input and returns canonical family and revision
// identities independent of checkout path and Git revision.
func Derive(input Input) (Fragment, error) {
	repository := strings.TrimSpace(input.RepositoryID)
	if repository == "" {
		return Fragment{}, errors.New("adaptive fragment repository scope is incomplete")
	}
	if !authorityAllowed(input.Kind, input.Authority) || !validSHA(input.AuthoritySHA256) {
		return Fragment{}, errors.New("adaptive fragment correctness authority is invalid")
	}
	selector, err := normalizedStrings(input.Selector, false)
	if err != nil {
		return Fragment{}, errors.New("adaptive fragment selector is incomplete")
	}
	bindings, err := normalizedBindings(input.Kind, input.Bindings)
	if err != nil {
		return Fragment{}, err
	}
	requires, err := normalizedDigests(input.Requires)
	if err != nil {
		return Fragment{}, errors.New("adaptive fragment requirements are invalid")
	}
	conflicts, err := normalizedDigests(input.ConflictsWith)
	if err != nil {
		return Fragment{}, errors.New("adaptive fragment conflicts are invalid")
	}
	if overlaps(requires, conflicts) {
		return Fragment{}, errors.New("adaptive fragment cannot require and conflict with the same family")
	}

	repositoryScope := digest("buildopt-adaptive-fragment-repository-v1", repository)
	selectorSHA := digest("buildopt-adaptive-fragment-selector-v1", selector...)
	familyID := canonicalFamilyID(repositoryScope, input.Kind, selectorSHA, input.Authority)
	if contains(requires, familyID) || contains(conflicts, familyID) {
		return Fragment{}, errors.New("adaptive fragment cannot reference its own family")
	}

	fragment := Fragment{
		SchemaVersion:         SchemaVersion,
		FamilyID:              familyID,
		RepositoryScopeSHA256: repositoryScope,
		Kind:                  input.Kind,
		SelectorSHA256:        selectorSHA,
		Authority:             input.Authority,
		AuthoritySHA256:       input.AuthoritySHA256,
		Bindings:              bindings,
		Requires:              requires,
		ConflictsWith:         conflicts,
	}
	fragment.RevisionID = revisionID(fragment)
	return fragment, nil
}

// Valid reports whether a fragment is canonical and internally consistent.
func Valid(fragment Fragment) bool {
	if fragment.SchemaVersion != SchemaVersion || !validSHA(fragment.FamilyID) ||
		!validSHA(fragment.RevisionID) || !validSHA(fragment.RepositoryScopeSHA256) ||
		!validSHA(fragment.SelectorSHA256) || !validSHA(fragment.AuthoritySHA256) ||
		!authorityAllowed(fragment.Kind, fragment.Authority) {
		return false
	}
	if fragment.FamilyID != canonicalFamilyID(fragment.RepositoryScopeSHA256,
		fragment.Kind, fragment.SelectorSHA256, fragment.Authority) {
		return false
	}
	bindings, err := normalizedBindings(fragment.Kind, fragment.Bindings)
	if err != nil || !equalBindings(bindings, fragment.Bindings) {
		return false
	}
	requires, err := normalizedDigests(fragment.Requires)
	if err != nil || !equalStrings(requires, fragment.Requires) {
		return false
	}
	conflicts, err := normalizedDigests(fragment.ConflictsWith)
	if err != nil || !equalStrings(conflicts, fragment.ConflictsWith) || overlaps(requires, conflicts) ||
		contains(requires, fragment.FamilyID) || contains(conflicts, fragment.FamilyID) {
		return false
	}
	return fragment.RevisionID == revisionID(fragment)
}

// Evaluate compares only the bindings declared by the fragment. Unrelated
// context drift does not invalidate it; missing, ambiguous or changed declared
// bindings retain native Gradle for that fragment.
func Evaluate(fragment Fragment, context Context) (Compatibility, error) {
	if !Valid(fragment) {
		return Compatibility{}, errors.New("adaptive fragment is invalid")
	}
	repository := strings.TrimSpace(context.RepositoryID)
	if repository == "" {
		return Compatibility{}, errors.New("adaptive fragment context repository is incomplete")
	}
	if digest("buildopt-adaptive-fragment-repository-v1", repository) != fragment.RepositoryScopeSHA256 {
		return Compatibility{Reason: "REPOSITORY_SCOPE_MISMATCH"}, nil
	}
	ambiguous := make(map[BindingKey]bool, len(context.Ambiguous))
	for _, key := range context.Ambiguous {
		if !knownBindings[key] || ambiguous[key] {
			return Compatibility{}, errors.New("adaptive fragment context ambiguity is invalid")
		}
		ambiguous[key] = true
	}

	missing, drifted, uncertain := []BindingKey{}, []BindingKey{}, []BindingKey{}
	for key, expected := range fragment.Bindings {
		if ambiguous[key] {
			uncertain = append(uncertain, key)
			continue
		}
		observed, exists := context.Bindings[key]
		if !exists || !validSHA(observed) {
			missing = append(missing, key)
			continue
		}
		if observed != expected {
			drifted = append(drifted, key)
		}
	}
	if len(uncertain) > 0 {
		sortBindingKeys(uncertain)
		return Compatibility{Reason: "AMBIGUOUS_BINDING", Bindings: uncertain}, nil
	}
	if len(missing) > 0 {
		sortBindingKeys(missing)
		return Compatibility{Reason: "MISSING_BINDING", Bindings: missing}, nil
	}
	if len(drifted) > 0 {
		sortBindingKeys(drifted)
		return Compatibility{Reason: "BINDING_DRIFT", Bindings: drifted}, nil
	}
	return Compatibility{Compatible: true, Reason: "COMPATIBLE"}, nil
}

// CanTransition reports whether a lifecycle transition preserves the AF-001
// requalification rule. Suspended fragments cannot return directly to active.
func CanTransition(from, to State) bool {
	allowed := map[State]map[State]bool{
		StateObserved:  {StateShadow: true, StateSuspended: true, StateExpired: true},
		StateShadow:    {StateQualified: true, StateSuspended: true, StateExpired: true},
		StateQualified: {StateActive: true, StateSuspended: true, StateExpired: true},
		StateActive:    {StateSuspended: true, StateExpired: true},
		StateSuspended: {StateShadow: true, StateExpired: true},
		StateExpired:   {},
	}
	return allowed[from][to]
}

func authorityAllowed(kind Kind, authority Authority) bool {
	switch kind {
	case KindSubgraph:
		return authority == AuthorityGradleModel || authority == AuthorityReviewedAdapter
	case KindOutputMaterialization:
		return authority == AuthorityVerifiedProducer
	case KindTaskContract:
		return authority == AuthorityGradleNative || authority == AuthorityReviewedAdapter ||
			authority == AuthorityReviewedPatch
	case KindPatch:
		return authority == AuthorityReviewedPatch
	case KindCacheLocality:
		return authority == AuthorityGradleNative
	default:
		return false
	}
}

func normalizedBindings(kind Kind, values map[BindingKey]string) (map[BindingKey]string, error) {
	if len(values) == 0 {
		return nil, errors.New("adaptive fragment bindings are incomplete")
	}
	result := make(map[BindingKey]string, len(values))
	for key, value := range values {
		if !knownBindings[key] || !validSHA(value) {
			return nil, errors.New("adaptive fragment binding is invalid")
		}
		result[key] = value
	}
	for _, key := range requiredBindings[kind] {
		if _, exists := result[key]; !exists {
			return nil, errors.New("adaptive fragment required binding is missing")
		}
	}
	return result, nil
}

func revisionID(fragment Fragment) string {
	rows := make([]string, 0, len(fragment.Bindings))
	for key, value := range fragment.Bindings {
		rows = append(rows, string(key)+"="+value)
	}
	sort.Strings(rows)
	return digest("buildopt-adaptive-fragment-revision-v1", fragment.FamilyID,
		fragment.AuthoritySHA256, digest("bindings-v1", rows...),
		digest("requires-v1", fragment.Requires...),
		digest("conflicts-v1", fragment.ConflictsWith...))
}

func canonicalFamilyID(repositoryScope string, kind Kind, selectorSHA string, authority Authority) string {
	return digest("buildopt-adaptive-fragment-family-v1",
		repositoryScope, string(kind), selectorSHA, string(authority))
}

func normalizedStrings(values []string, allowEmpty bool) ([]string, error) {
	if len(values) == 0 {
		if allowEmpty {
			return []string{}, nil
		}
		return nil, errors.New("values are empty")
	}
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strings.TrimSpace(value)
		if result[index] == "" {
			return nil, errors.New("value is empty")
		}
	}
	sort.Strings(result)
	for index := 1; index < len(result); index++ {
		if result[index] == result[index-1] {
			return nil, errors.New("values are not unique")
		}
	}
	return result, nil
}

func normalizedDigests(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}
	result, err := normalizedStrings(values, true)
	if err != nil {
		return nil, err
	}
	for _, value := range result {
		if !validSHA(value) {
			return nil, errors.New("digest is invalid")
		}
	}
	return result, nil
}

func equalBindings(left, right map[BindingKey]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func overlaps(left, right []string) bool {
	values := make(map[string]bool, len(left))
	for _, value := range left {
		values[value] = true
	}
	for _, value := range right {
		if values[value] {
			return true
		}
	}
	return false
}

func contains(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func sortBindingKeys(values []BindingKey) {
	sort.Slice(values, func(left, right int) bool { return values[left] < values[right] })
}

func validSHA(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func digest(domain string, values ...string) string {
	hash := sha256.New()
	writeHashValue(hash, domain)
	for _, value := range values {
		writeHashValue(hash, value)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeHashValue(writer io.Writer, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = io.WriteString(writer, value)
}
