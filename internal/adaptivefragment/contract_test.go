package adaptivefragment

import (
	"strings"
	"testing"
)

func TestFragmentIdentityIsDeterministicAndPathIndependent(t *testing.T) {
	first, err := Derive(subgraphFixture())
	if err != nil {
		t.Fatal(err)
	}
	input := subgraphFixture()
	input.Selector = []string{"owner=:library", "entrypoint=:library:classes"}
	input.Bindings = reversedBindings(input.Bindings)
	second, err := Derive(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.FamilyID != second.FamilyID || first.RevisionID != second.RevisionID || !Valid(first) || !Valid(second) {
		t.Fatalf("fragment identity is not canonical: %+v / %+v", first, second)
	}
	// Checkout path and Git revision are deliberately absent from Input.
}

func TestEveryFragmentClassEnforcesItsMinimumBindings(t *testing.T) {
	fixtures := []Input{
		subgraphFixture(),
		outputMaterializationFixture(),
		taskFixture(),
		patchFixture(),
		cacheLocalityFixture(),
	}
	for _, fixture := range fixtures {
		t.Run(string(fixture.Kind), func(t *testing.T) {
			fragment := mustDerive(t, fixture)
			if !Valid(fragment) {
				t.Fatal("canonical fragment was rejected")
			}
			for _, binding := range requiredBindings[fixture.Kind] {
				invalid := fixture
				invalid.Bindings = copyBindings(fixture.Bindings)
				delete(invalid.Bindings, binding)
				if _, err := Derive(invalid); err == nil {
					t.Fatalf("fragment without required binding %s was accepted", binding)
				}
			}
		})
	}
}

func TestBindingDriftInvalidatesOnlyDependentFragments(t *testing.T) {
	subgraph := mustDerive(t, subgraphFixture())
	task := mustDerive(t, taskFixture())
	context := completeContext()

	changeDrift := copyContext(context)
	changeDrift.Bindings[BindingChangeFamily] = sha("changed-family")
	assertCompatibility(t, subgraph, changeDrift, false, "BINDING_DRIFT", BindingChangeFamily)
	assertCompatibility(t, task, changeDrift, true, "COMPATIBLE")

	taskDrift := copyContext(context)
	taskDrift.Bindings[BindingTaskImplementation] = sha("changed-task")
	assertCompatibility(t, subgraph, taskDrift, true, "COMPATIBLE")
	assertCompatibility(t, task, taskDrift, false, "BINDING_DRIFT", BindingTaskImplementation)

	outputDrift := copyContext(context)
	outputDrift.Bindings[BindingOutputContract] = sha("changed-output")
	assertCompatibility(t, subgraph, outputDrift, false, "BINDING_DRIFT", BindingOutputContract)
	assertCompatibility(t, task, outputDrift, false, "BINDING_DRIFT", BindingOutputContract)
}

func TestMissingAmbiguousAndCrossRepositoryContextRetainNative(t *testing.T) {
	fragment := mustDerive(t, subgraphFixture())

	missing := completeContext()
	delete(missing.Bindings, BindingProducerLineage)
	assertCompatibility(t, fragment, missing, false, "MISSING_BINDING", BindingProducerLineage)

	ambiguous := completeContext()
	ambiguous.Ambiguous = []BindingKey{BindingOutputContract}
	assertCompatibility(t, fragment, ambiguous, false, "AMBIGUOUS_BINDING", BindingOutputContract)

	otherRepository := completeContext()
	otherRepository.RepositoryID = "example/other"
	assertCompatibility(t, fragment, otherRepository, false, "REPOSITORY_SCOPE_MISMATCH")
}

func TestFragmentContractRejectsInvalidAuthorityBindingsAndCompositionReferences(t *testing.T) {
	cases := map[string]func(*Input){
		"authority":       func(input *Input) { input.Authority = AuthorityReviewedPatch },
		"missing binding": func(input *Input) { delete(input.Bindings, BindingProducerLineage) },
		"unknown binding": func(input *Input) { input.Bindings["REPOSITORY_NAME_RULE"] = sha("rule") },
		"invalid digest":  func(input *Input) { input.Bindings[BindingWrapper] = "invalid" },
		"require and conflict": func(input *Input) {
			value := sha("family")
			input.Requires, input.ConflictsWith = []string{value}, []string{value}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := subgraphFixture()
			mutate(&input)
			if _, err := Derive(input); err == nil {
				t.Fatal("invalid fragment contract was accepted")
			}
		})
	}
}

func TestFragmentValidationRejectsTamperedIdentity(t *testing.T) {
	fragment := mustDerive(t, subgraphFixture())
	fragment.FamilyID = sha("different-family")
	fragment.RevisionID = revisionID(fragment)
	if Valid(fragment) {
		t.Fatal("fragment with a self-consistent but non-canonical family identity was accepted")
	}
}

func TestFragmentLifecycleRequiresRequalificationAfterSuspension(t *testing.T) {
	allowed := [][2]State{
		{StateObserved, StateShadow},
		{StateShadow, StateQualified},
		{StateQualified, StateActive},
		{StateActive, StateSuspended},
		{StateSuspended, StateShadow},
		{StateObserved, StateExpired},
	}
	for _, transition := range allowed {
		if !CanTransition(transition[0], transition[1]) {
			t.Fatalf("expected transition %s -> %s", transition[0], transition[1])
		}
	}
	for _, transition := range [][2]State{
		{StateObserved, StateActive},
		{StateShadow, StateActive},
		{StateSuspended, StateActive},
		{StateExpired, StateObserved},
	} {
		if CanTransition(transition[0], transition[1]) {
			t.Fatalf("unexpected transition %s -> %s", transition[0], transition[1])
		}
	}
}

func mustDerive(t *testing.T, input Input) Fragment {
	t.Helper()
	fragment, err := Derive(input)
	if err != nil {
		t.Fatal(err)
	}
	return fragment
}

func assertCompatibility(t *testing.T, fragment Fragment, context Context, compatible bool, reason string, bindings ...BindingKey) {
	t.Helper()
	result, err := Evaluate(fragment, context)
	if err != nil {
		t.Fatal(err)
	}
	if result.Compatible != compatible || result.Reason != reason || len(result.Bindings) != len(bindings) {
		t.Fatalf("compatibility = %+v, want compatible=%v reason=%s bindings=%v", result, compatible, reason, bindings)
	}
	for index := range bindings {
		if result.Bindings[index] != bindings[index] {
			t.Fatalf("compatibility bindings = %v, want %v", result.Bindings, bindings)
		}
	}
}

func subgraphFixture() Input {
	return Input{
		RepositoryID:    "example/repository",
		Kind:            KindSubgraph,
		Selector:        []string{"entrypoint=:library:classes", "owner=:library"},
		Authority:       AuthorityGradleModel,
		AuthoritySHA256: sha("gradle-model"),
		Bindings: map[BindingKey]string{
			BindingWorkflow:        sha("workflow"),
			BindingWrapper:         sha("wrapper"),
			BindingProducerLineage: sha("lineage"),
			BindingOutputContract:  sha("outputs"),
			BindingChangeFamily:    sha("change-family"),
		},
	}
}

func taskFixture() Input {
	return Input{
		RepositoryID:    "example/repository",
		Kind:            KindTaskContract,
		Selector:        []string{"task-type=JavaCompile"},
		Authority:       AuthorityGradleNative,
		AuthoritySHA256: sha("gradle-native-contract"),
		Bindings: map[BindingKey]string{
			BindingWrapper:            sha("wrapper"),
			BindingTaskImplementation: sha("task-implementation"),
			BindingOutputContract:     sha("outputs"),
		},
	}
}

func outputMaterializationFixture() Input {
	return Input{
		RepositoryID:    "example/repository",
		Kind:            KindOutputMaterialization,
		Selector:        []string{"producer=:library:jar"},
		Authority:       AuthorityVerifiedProducer,
		AuthoritySHA256: sha("verified-producer"),
		Bindings: map[BindingKey]string{
			BindingWrapper:         sha("wrapper"),
			BindingProducerLineage: sha("lineage"),
			BindingOutputContract:  sha("outputs"),
		},
	}
}

func patchFixture() Input {
	return Input{
		RepositoryID:    "example/repository",
		Kind:            KindPatch,
		Selector:        []string{"recipe=declare-cacheable-task"},
		Authority:       AuthorityReviewedPatch,
		AuthoritySHA256: sha("reviewed-patch"),
		Bindings: map[BindingKey]string{
			BindingTaskImplementation: sha("task-implementation"),
			BindingOutputContract:     sha("outputs"),
			BindingPatchBase:          sha("patch-base"),
		},
	}
}

func cacheLocalityFixture() Input {
	return Input{
		RepositoryID:    "example/repository",
		Kind:            KindCacheLocality,
		Selector:        []string{"tier=edge"},
		Authority:       AuthorityGradleNative,
		AuthoritySHA256: sha("gradle-native-cache"),
		Bindings: map[BindingKey]string{
			BindingWrapper:        sha("wrapper"),
			BindingNetworkClass:   sha("network"),
			BindingCacheNamespace: sha("cache"),
		},
	}
}

func completeContext() Context {
	return Context{
		RepositoryID: "example/repository",
		Bindings: map[BindingKey]string{
			BindingWorkflow:           sha("workflow"),
			BindingWrapper:            sha("wrapper"),
			BindingTaskImplementation: sha("task-implementation"),
			BindingProducerLineage:    sha("lineage"),
			BindingOutputContract:     sha("outputs"),
			BindingChangeFamily:       sha("change-family"),
			BindingPlatform:           sha("platform"),
			BindingNetworkClass:       sha("network"),
			BindingCacheNamespace:     sha("cache"),
			BindingPatchBase:          sha("patch"),
		},
	}
}

func copyContext(input Context) Context {
	result := Context{RepositoryID: input.RepositoryID, Bindings: map[BindingKey]string{}, Ambiguous: append([]BindingKey{}, input.Ambiguous...)}
	for key, value := range input.Bindings {
		result.Bindings[key] = value
	}
	return result
}

func reversedBindings(input map[BindingKey]string) map[BindingKey]string {
	keys := make([]BindingKey, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	result := make(map[BindingKey]string, len(input))
	for index := len(keys) - 1; index >= 0; index-- {
		result[keys[index]] = input[keys[index]]
	}
	return result
}

func copyBindings(input map[BindingKey]string) map[BindingKey]string {
	result := make(map[BindingKey]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func sha(value string) string {
	return digest("adaptive-fragment-test-v1", strings.TrimSpace(value))
}
