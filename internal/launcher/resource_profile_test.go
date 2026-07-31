package launcher

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/runtimeoptimizer"
)

func TestApplyAuthorizedResourceProfileBeforeGradle(t *testing.T) {
	profile := runtimeoptimizer.GoldenResourceProfiles()[2]
	authority := &localAuthorityContext{
		resourceProfileAuthorized: true,
		resourceProfile: localauthority.ResourceProfileReference{
			ProfileID: profile.ProfileID, ProfileDigest: profile.ProfileDigest, CatalogVersion: profile.CatalogVersion,
		},
		managedL1Config: managedL1Config{compatibilityClass: runtimeoptimizer.GoldenCompatibilityClass},
	}
	got, err := applyAuthorizedResourceProfile(
		[]string{"./gradlew", "--max-workers=9", "assemble"},
		authority,
		resourceProfileTestEnvironment,
	)
	want := []string{"./gradlew", "--max-workers=3", "-Dorg.gradle.jvmargs=-Xmx4096m -Dfile.encoding=UTF-8", "assemble"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v/%v", got, err)
	}
}

func TestApplyAuthorizedResourceProfileFailsOpenBeforeExecution(t *testing.T) {
	original := []string{"./gradlew", "assemble"}
	profile := runtimeoptimizer.GoldenResourceProfiles()[3]
	authority := &localAuthorityContext{
		resourceProfileAuthorized: true,
		resourceProfile: localauthority.ResourceProfileReference{
			ProfileID: profile.ProfileID, ProfileDigest: profile.ProfileDigest, CatalogVersion: profile.CatalogVersion,
		},
		managedL1Config: managedL1Config{compatibilityClass: runtimeoptimizer.GoldenCompatibilityClass},
	}
	got, err := applyAuthorizedResourceProfile(original, authority, func(name string) string {
		if name == resourceAvailableMemoryEnvironment {
			return strconv.FormatInt(7*1024*1024*1024, 10)
		}
		return resourceProfileTestEnvironment(name)
	})
	if err == nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("fallback = %#v/%v", got, err)
	}
	authority.resourceProfileAuthorized = false
	got, err = applyAuthorizedResourceProfile(original, authority, nil)
	if err != nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("unauthorized = %#v/%v", got, err)
	}
}

func resourceProfileTestEnvironment(name string) string {
	values := map[string]string{
		resourceRunnerClassEnvironment:     runtimeoptimizer.GoldenRunnerClass,
		resourceBuildClassEnvironment:      runtimeoptimizer.GoldenBuildClass,
		resourceJDKVendorEnvironment:       runtimeoptimizer.GoldenJDKVendor,
		resourceJDKVersionEnvironment:      runtimeoptimizer.GoldenJDKVersion,
		resourceJDKArchitectureEnvironment: runtimeoptimizer.GoldenJDKArchitecture,
		resourceCgroupCPUEnvironment:       "4",
		resourceCgroupMemoryEnvironment:    strconv.FormatInt(16*1024*1024*1024, 10),
		resourceAvailableMemoryEnvironment: strconv.FormatInt(16*1024*1024*1024, 10),
	}
	return strings.TrimSpace(values[name])
}
