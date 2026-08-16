package launcher

import "testing"

func TestUpsertOptimizePortfolioEntryMaintainsIndependentFamilies(t *testing.T) {
	digest := func(character byte) string {
		value := make([]byte, 64)
		for index := range value {
			value[index] = character
		}
		return string(value)
	}
	existing := []optimizePortfolioEntry{
		{Family: optimizeFamilyResource, FamilySHA256: digest('b'), ProfileSHA256: digest('1')},
		{Family: optimizeFamilyLeaf, FamilySHA256: digest('d'), ProfileSHA256: digest('2')},
	}
	withDependency := upsertOptimizePortfolioEntry(existing, optimizePortfolioEntry{
		Family: optimizeFamilyDependency, FamilySHA256: digest('a'), ProfileSHA256: digest('3'),
	})
	withMixed := upsertOptimizePortfolioEntry(withDependency, optimizePortfolioEntry{
		Family: optimizeFamilyMixed, FamilySHA256: digest('c'), ProfileSHA256: digest('4'),
	})
	if len(withMixed) != 4 {
		t.Fatalf("profile count = %d, want 4", len(withMixed))
	}
	for index, want := range []string{digest('a'), digest('b'), digest('c'), digest('d')} {
		if withMixed[index].FamilySHA256 != want {
			t.Fatalf("profile %d family SHA = %q, want %q", index, withMixed[index].FamilySHA256, want)
		}
	}
	replaced := upsertOptimizePortfolioEntry(withMixed, optimizePortfolioEntry{
		Family: optimizeFamilyResource, FamilySHA256: digest('b'), ProfileSHA256: digest('9'),
	})
	if len(replaced) != 4 || replaced[1].ProfileSHA256 != digest('9') {
		t.Fatalf("same-family replacement = %+v", replaced)
	}
}

func TestOptimizePortfolioFamilyDigestIsPortableAndStructural(t *testing.T) {
	projects := []string{":core"}
	entrypoints := []string{"jar"}
	candidate := []string{":core:jar"}
	outputs := []string{"core/build/libs/core.jar"}
	first := optimizePortfolioFamilyDigest("owner/repository", optimizeFamilyDependency, projects, entrypoints, candidate, outputs)
	second := optimizePortfolioFamilyDigest("owner/repository", optimizeFamilyDependency, projects, entrypoints, candidate, outputs)
	if first != second || !validOptimizeSHA(first) {
		t.Fatalf("portable structural digest = %q, second = %q", first, second)
	}
	if first == optimizePortfolioFamilyDigest("other/repository", optimizeFamilyDependency, projects, entrypoints, candidate, outputs) {
		t.Fatal("repository identity did not scope the logical family")
	}
	if first == optimizePortfolioFamilyDigest("owner/repository", optimizeFamilyLeaf, projects, entrypoints, candidate, outputs) {
		t.Fatal("structural family did not affect the logical identity")
	}
}
