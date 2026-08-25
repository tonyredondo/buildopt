package adaptivefragment

import (
	"strings"
	"testing"
)

func TestCompatibilityIndexDecisions(t *testing.T) {
	snapshot := compatibilityTestSnapshot("fixture/repository")
	fragment := compatibilityTestFragment(t, snapshot, KindSubgraph, "main-build")
	index, err := NewCompatibilityIndex(snapshot, 1, []PersistedFragment{fragment})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		mutate      func(FingerprintSnapshot) FingerprintSnapshot
		disposition LookupDisposition
		reason      string
		binding     BindingKey
	}{
		{
			name: "exact fingerprints",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.GitRevisionSHA256 = digest("test-revision", "descendant")
				return candidate
			},
			disposition: DispositionCompatible,
			reason:      "COMPATIBLE_FRAGMENT_CANDIDATES",
		},
		{
			name: "unrelated platform drift",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.PlatformSHA256 = digest("test-platform", "changed")
				return candidate
			},
			disposition: DispositionCompatible,
			reason:      "COMPATIBLE_FRAGMENT_CANDIDATES",
		},
		{
			name: "declared wrapper drift",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.WrapperSHA256 = digest("test-wrapper", "changed")
				return candidate
			},
			disposition: DispositionSuspended,
			reason:      "DECLARED_BINDING_DRIFT",
			binding:     BindingWrapper,
		},
		{
			name: "ambiguous producer lineage",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.Ambiguous = []BindingKey{BindingProducerLineage}
				return candidate
			},
			disposition: DispositionNativeRetained,
			reason:      "NO_COMPATIBLE_FRAGMENT",
			binding:     BindingProducerLineage,
		},
		{
			name: "repository mismatch",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.RepositoryID = "fixture/other-repository"
				return candidate
			},
			disposition: DispositionNativeRetained,
			reason:      "REPOSITORY_SCOPE_MISMATCH",
		},
		{
			name: "missing output contract",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.OutputContractSHA256 = ""
				return candidate
			},
			disposition: DispositionNativeRetained,
			reason:      "NO_COMPATIBLE_FRAGMENT",
			binding:     BindingOutputContract,
		},
		{
			name: "invalid fingerprint",
			mutate: func(candidate FingerprintSnapshot) FingerprintSnapshot {
				candidate.OutputContractSHA256 = "invalid"
				return candidate
			},
			disposition: DispositionNativeRetained,
			reason:      "FINGERPRINT_INVALID",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := Lookup(index, test.mutate(snapshot))
			if result.Disposition != test.disposition || result.Reason != test.reason {
				t.Fatalf("lookup = %+v, want %s/%s", result, test.disposition, test.reason)
			}
			if test.binding != "" {
				if len(result.Decisions) != 1 || len(result.Decisions[0].Bindings) != 1 ||
					result.Decisions[0].Bindings[0] != test.binding {
					t.Fatalf("lookup binding = %+v, want %s", result.Decisions, test.binding)
				}
			}
		})
	}
}

func TestCompatibilityIndexIgnoresMissingUndeclaredFingerprint(t *testing.T) {
	snapshot := compatibilityTestSnapshot("fixture/repository")
	fragment := compatibilityTestFragment(t, snapshot, KindSubgraph, "main-build")
	index, err := NewCompatibilityIndex(snapshot, 1, []PersistedFragment{fragment})
	if err != nil {
		t.Fatal(err)
	}
	snapshot.PlatformSHA256 = ""
	snapshot.TaskImplementationSHA256 = ""
	result := Lookup(index, snapshot)
	if result.Disposition != DispositionCompatible {
		t.Fatalf("missing undeclared fingerprint lookup = %+v", result)
	}
}

func TestCompatibilityIndexRetainsExpiredAndCorruptState(t *testing.T) {
	snapshot := compatibilityTestSnapshot("fixture/repository")
	fragment := compatibilityTestFragment(t, snapshot, KindSubgraph, "main-build")
	index, err := NewCompatibilityIndex(snapshot, 1, []PersistedFragment{fragment})
	if err != nil {
		t.Fatal(err)
	}

	expired := index
	expired.Fragments = append([]PersistedFragment{}, index.Fragments...)
	expired.Fragments[0].EvidenceExpiresAt = snapshot.ObservedAt
	if result := Lookup(expired, snapshot); result.Disposition != DispositionNativeRetained ||
		result.Reason != "NO_COMPATIBLE_FRAGMENT" || len(result.Decisions) != 1 ||
		result.Decisions[0].Reason != "EVIDENCE_EXPIRED" {
		t.Fatalf("expired fragment lookup = %+v", result)
	}

	corrupt := index
	corrupt.SourceRevisionSHA256 = strings.Repeat("f", 63)
	if result := Lookup(corrupt, snapshot); result.Disposition != DispositionNativeRetained || result.Reason != "INDEX_INVALID" {
		t.Fatalf("corrupt index lookup = %+v", result)
	}
}

func TestCompatibilityIndexSortsFragmentsAndRejectsDuplicates(t *testing.T) {
	snapshot := compatibilityTestSnapshot("fixture/repository")
	first := compatibilityTestFragment(t, snapshot, KindSubgraph, "first")
	second := compatibilityTestFragment(t, snapshot, KindSubgraph, "second")
	index, err := NewCompatibilityIndex(snapshot, 1, []PersistedFragment{second, first})
	if err != nil {
		t.Fatal(err)
	}
	if index.Fragments[0].FamilyID > index.Fragments[1].FamilyID {
		t.Fatalf("index is not canonical: %+v", index.Fragments)
	}
	if _, err := NewCompatibilityIndex(snapshot, 1, []PersistedFragment{first, first}); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("duplicate fragment error = %v", err)
	}
}

func TestCompatibilityIndexDoesNotAliasSourceFragments(t *testing.T) {
	snapshot := compatibilityTestSnapshot("fixture/repository")
	fragment := compatibilityTestFragment(t, snapshot, KindSubgraph, "main-build")
	index, err := NewCompatibilityIndex(snapshot, 1, []PersistedFragment{fragment})
	if err != nil {
		t.Fatal(err)
	}
	fragment.Bindings[BindingWrapper] = digest("test-wrapper", "mutated-after-build")
	fragment.Requires = append(fragment.Requires, digest("test-requirement", "mutated-after-build"))
	result := Lookup(index, snapshot)
	if result.Disposition != DispositionCompatible {
		t.Fatalf("source mutation changed index = %+v", result)
	}
}

func compatibilityTestSnapshot(repository string) FingerprintSnapshot {
	return FingerprintSnapshot{
		RepositoryID:          repository,
		GitRevisionSHA256:     digest("test-git", repository),
		WrapperSHA256:         digest("test-wrapper", "gradle-9.6.1"),
		WorkflowSHA256:        digest("test-workflow", "build"),
		ProducerLineageSHA256: digest("test-producer", "root"),
		OutputContractSHA256:  digest("test-output", "required"),
		ChangeFamilySHA256:    digest("test-change", "leaf-source"),
		PlatformSHA256:        digest("test-platform", "linux-amd64"),
		ObservedAt:            "2026-08-25T12:00:00Z",
	}
}

func compatibilityTestFragment(t *testing.T, snapshot FingerprintSnapshot, kind Kind, selector string) PersistedFragment {
	t.Helper()
	fragment, err := Derive(Input{
		RepositoryID:    snapshot.RepositoryID,
		Kind:            kind,
		Selector:        []string{selector},
		Authority:       AuthorityGradleModel,
		AuthoritySHA256: digest("test-authority", "gradle-model"),
		Bindings: map[BindingKey]string{
			BindingWorkflow:        snapshot.WorkflowSHA256,
			BindingWrapper:         snapshot.WrapperSHA256,
			BindingProducerLineage: snapshot.ProducerLineageSHA256,
			BindingOutputContract:  snapshot.OutputContractSHA256,
			BindingChangeFamily:    snapshot.ChangeFamilySHA256,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return PersistedFragment{
		SchemaVersion:         FragmentStateSchemaVersion,
		RecordType:            "ADAPTIVE_FRAGMENT",
		FamilyID:              fragment.FamilyID,
		RevisionID:            fragment.RevisionID,
		RepositoryScopeSHA256: fragment.RepositoryScopeSHA256,
		Kind:                  fragment.Kind,
		SelectorSHA256:        fragment.SelectorSHA256,
		Authority:             fragment.Authority,
		AuthoritySHA256:       fragment.AuthoritySHA256,
		Bindings:              fragment.Bindings,
		Requires:              fragment.Requires,
		ConflictsWith:         fragment.ConflictsWith,
		State:                 StateActive,
		Generation:            4,
		CreatedAt:             "2026-08-25T10:00:00Z",
		UpdatedAt:             "2026-08-25T10:03:00Z",
		EvidenceExpiresAt:     "2026-09-25T10:03:00Z",
	}
}
