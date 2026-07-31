package runtimeoptimizer

import (
	"reflect"
	"testing"
)

func TestGoldenResourceCatalogContainsOnlyFourExactArms(t *testing.T) {
	profiles := GoldenResourceProfiles()
	wantIDs := []string{"STABLE_CONTROL", "W2_H3G", "W3_H4G", "W4_H6G"}
	if len(profiles) != len(wantIDs) {
		t.Fatalf("profiles = %d", len(profiles))
	}
	for index, profile := range profiles {
		if profile.ProfileID != wantIDs[index] || profile.CatalogVersion != GoldenResourceCatalogVersion ||
			profile.RunnerClass != GoldenRunnerClass || profile.BuildClass != GoldenBuildClass ||
			profile.CompatibilityClass != GoldenCompatibilityClass || profile.CompilerForking || profile.CompilerMaxProcesses != 1 ||
			!profile.StartupPassed || !profile.MemoryPassed || !profile.RollbackPassed {
			t.Fatalf("profile[%d] = %+v", index, profile)
		}
	}
	profiles[0].GradleJVMArgs[0] = "mutated"
	if GoldenResourceProfiles()[0].GradleJVMArgs[0] != "-Dfile.encoding=UTF-8" {
		t.Fatal("catalog returned mutable shared state")
	}
}

func TestSelectGoldenResourceProfilesMaterializesOnlyWorkersAndHeap(t *testing.T) {
	context := goldenResourceProfileContext()
	wants := map[string][]string{
		"STABLE_CONTROL": {"--max-workers=2", "-Dorg.gradle.jvmargs=-Xmx4096m -Dfile.encoding=UTF-8"},
		"W2_H3G":         {"--max-workers=2", "-Dorg.gradle.jvmargs=-Xmx3072m -Dfile.encoding=UTF-8"},
		"W3_H4G":         {"--max-workers=3", "-Dorg.gradle.jvmargs=-Xmx4096m -Dfile.encoding=UTF-8"},
		"W4_H6G":         {"--max-workers=4", "-Dorg.gradle.jvmargs=-Xmx6144m -Dfile.encoding=UTF-8"},
	}
	for _, profile := range GoldenResourceProfiles() {
		selection, err := SelectGoldenResourceProfile(profile.ProfileID, profile.ProfileDigest, profile.CatalogVersion, context)
		if err != nil || !reflect.DeepEqual(selection.Arguments, wants[profile.ProfileID]) {
			t.Fatalf("selection %s = %+v/%v", profile.ProfileID, selection, err)
		}
	}
}

func TestSelectGoldenResourceProfileRejectsUnknownOrIneligibleContexts(t *testing.T) {
	profile := GoldenResourceProfiles()[3]
	tests := []struct {
		name    string
		id      string
		digest  string
		catalog string
		mutate  func(*ResourceProfileContext)
	}{
		{name: "unknown arm", id: "W8_H12G", digest: profile.ProfileDigest, catalog: profile.CatalogVersion},
		{name: "wrong digest", id: profile.ProfileID, digest: "sha256:" + repeat("e", 64), catalog: profile.CatalogVersion},
		{name: "wrong catalog", id: profile.ProfileID, digest: profile.ProfileDigest, catalog: "other-v1"},
		{name: "other runner", id: profile.ProfileID, digest: profile.ProfileDigest, catalog: profile.CatalogVersion, mutate: func(context *ResourceProfileContext) { context.RunnerClass = "linux-amd64-12c-32g-v1" }},
		{name: "other JDK", id: profile.ProfileID, digest: profile.ProfileDigest, catalog: profile.CatalogVersion, mutate: func(context *ResourceProfileContext) { context.JDKVersion = "21.0.9+10" }},
		{name: "other cgroup", id: profile.ProfileID, digest: profile.ProfileDigest, catalog: profile.CatalogVersion, mutate: func(context *ResourceProfileContext) { context.CgroupCPUCount = 12 }},
		{name: "headroom", id: profile.ProfileID, digest: profile.ProfileDigest, catalog: profile.CatalogVersion, mutate: func(context *ResourceProfileContext) { context.AvailableMemoryBytes = int64(7 * 1024 * 1024 * 1024) }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			context := goldenResourceProfileContext()
			if testCase.mutate != nil {
				testCase.mutate(&context)
			}
			if _, err := SelectGoldenResourceProfile(testCase.id, testCase.digest, testCase.catalog, context); err == nil {
				t.Fatal("ineligible profile was selected")
			}
		})
	}
}

func TestApplyResourceProfileArgumentsReplacesOnlyTunableInputs(t *testing.T) {
	profile := GoldenResourceProfiles()[2]
	selection, err := SelectGoldenResourceProfile(profile.ProfileID, profile.ProfileDigest, profile.CatalogVersion, goldenResourceProfileContext())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyResourceProfileArguments([]string{
		"./gradlew", "--max-workers", "9", "-Dorg.gradle.jvmargs=-Xmx10g -Dfile.encoding=UTF-8", "assemble", "--parallel",
	}, selection)
	want := []string{
		"./gradlew", "--max-workers=3", "-Dorg.gradle.jvmargs=-Xmx4096m -Dfile.encoding=UTF-8", "assemble", "--parallel",
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("arguments = %#v/%v", got, err)
	}
	if _, err := ApplyResourceProfileArguments([]string{"./gradlew", "--max-workers"}, selection); err == nil {
		t.Fatal("missing --max-workers value was accepted")
	}
}

func goldenResourceProfileContext() ResourceProfileContext {
	return ResourceProfileContext{
		RunnerClass: GoldenRunnerClass, BuildClass: GoldenBuildClass, CompatibilityClass: GoldenCompatibilityClass,
		JDKVendor: GoldenJDKVendor, JDKVersion: GoldenJDKVersion, JDKArchitecture: GoldenJDKArchitecture,
		CgroupCPUCount: 4, CgroupMemoryBytes: goldenMemoryBytes, AvailableMemoryBytes: goldenMemoryBytes,
	}
}
