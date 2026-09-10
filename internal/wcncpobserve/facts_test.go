package wcncpobserve

import (
	"strings"
	"testing"
)

func TestFactsDoNotInferCacheOutcomesFromInvocation(t *testing.T) {
	exit := 0
	for _, test := range []struct {
		args                 []string
		configuration, build string
	}{
		{nil, "UNAVAILABLE", "UNAVAILABLE"},
		{[]string{"--configuration-cache", "--build-cache"}, "UNAVAILABLE", "ENABLED"},
		{[]string{"--no-configuration-cache", "--no-build-cache"}, "NOT_REQUESTED", "DISABLED"},
		{[]string{"--no-build-cache", "--build-cache"}, "UNAVAILABLE", "ENABLED"},
	} {
		facts := BuildObservation("example/cache", strings.Repeat("a", 64), func(string) string { return "" }, t.TempDir(), test.args, []string{}, PassthroughResult{Child: ChildResult{Outcome: "SUCCESS", ExitCode: &exit}})
		if facts.ConfigurationCache != test.configuration || facts.BuildCacheMode != test.build {
			t.Fatalf("args=%q cache=%s/%s", test.args, facts.ConfigurationCache, facts.BuildCacheMode)
		}
	}
}

func TestFactsMissingOutputCannotBeControlledValue(t *testing.T) {
	values := map[string]string{"WCNCP_REPOSITORY_REVISION": strings.Repeat("a", 40), "WCNCP_GRADLE_VERSION": "9.6.1", "WCNCP_ENVIRONMENT_CLASS": "CONTROLLED_PERFORMANCE", "WCNCP_PROSPECTIVE_GATE_INPUT": "1"}
	for _, name := range []string{"WCNCP_SOURCE_TREE_SHA256", "WCNCP_WRAPPER_SHA256", "WCNCP_JDK_SHA256", "WCNCP_PACKAGE_SHA256", "WCNCP_WORKFLOW_SHA256", "WCNCP_ENVIRONMENT_SHA256", "WCNCP_OUTPUT_CONTRACT_SHA256"} {
		values[name] = strings.Repeat("b", 64)
	}
	exit := 0
	facts := BuildObservation("example/incomplete", strings.Repeat("c", 64), func(key string) string { return values[key] }, t.TempDir(), nil, []string{}, PassthroughResult{Child: ChildResult{Outcome: "SUCCESS", ExitCode: &exit}})
	if facts.Completeness != "INCOMPLETE" || facts.Authority.ProspectiveGateInput || facts.Duration.Classification != "NOT_EVALUATED" || facts.Duration.State != "UNAVAILABLE" {
		t.Fatalf("missing output gained controlled value: %+v", facts)
	}
}
