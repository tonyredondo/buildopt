package launcher

import (
	"reflect"
	"testing"
)

func TestApplyConfigurationCachePolicyEnablesStrictLocalEntries(t *testing.T) {
	authority := &localAuthorityContext{
		configurationPolicyDigest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		configurationCacheEnabled:  true,
		configurationCacheContract: "configuration-cache-v1",
	}
	got := applyConfigurationCachePolicy([]string{
		"./gradlew", "assemble", "--no-configuration-cache", "--configuration-cache-problems=warn", "--stacktrace",
	}, authority)
	want := []string{
		"./gradlew",
		"--configuration-cache",
		"--configuration-cache-problems=fail",
		"-Dbuildopt.configuration-policy.digest=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-Dbuildopt.configuration-contract.version=configuration-cache-v1",
		"assemble",
		"--stacktrace",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestApplyConfigurationCachePolicyDisablesAndInvalidates(t *testing.T) {
	authority := &localAuthorityContext{
		configurationPolicyDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		configurationCacheContract: "configuration-cache-v1",
	}
	got := applyConfigurationCachePolicy([]string{"/workspace/gradlew", "--configuration-cache", "test"}, authority)
	want := []string{
		"/workspace/gradlew",
		"--no-configuration-cache",
		"-Dbuildopt.configuration-policy.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"-Dbuildopt.configuration-contract.version=configuration-cache-v1",
		"test",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestApplyConfigurationCachePolicyPreservesNonGradleCommand(t *testing.T) {
	args := []string{"go", "test", "./..."}
	authority := &localAuthorityContext{configurationCacheEnabled: true, configurationCacheContract: "configuration-cache-v1"}
	if got := applyConfigurationCachePolicy(args, authority); !reflect.DeepEqual(got, args) {
		t.Fatalf("args = %#v", got)
	}
}
