package launcher

import (
	"reflect"
	"testing"
)

func TestProposalTerminalSelectorsAcceptsSingleAndQualifiedWorkflows(t *testing.T) {
	selectors, err := proposalTerminalSelectors([]string{
		":instrumentation:spring:one:testClasses",
		":instrumentation:spring:two:testClasses",
		":instrumentation:spring:three:classes",
	})
	if err != nil {
		t.Fatalf("proposalTerminalSelectors: %v", err)
	}
	want := []string{"classes", "testClasses"}
	if !reflect.DeepEqual(selectors, want) {
		t.Fatalf("selectors = %v, want %v", selectors, want)
	}
}

func TestProposalTerminalSelectorsRejectsMalformedEntrypoints(t *testing.T) {
	for _, entrypoint := range []string{"", ":", "classes task", "module/classes"} {
		if _, err := proposalTerminalSelectors([]string{entrypoint}); err == nil {
			t.Fatalf("expected %q to be rejected", entrypoint)
		}
	}
}

func TestProposalOutputOwnerProjectsUsesReviewedOutputOwners(t *testing.T) {
	report := outputContractReport{Validations: []outputContractValidation{
		{Pattern: "service-a/build/libs/*.jar", Status: "VALIDATED", OwnerProjects: []string{":service-a"}},
		{Pattern: "service-b/build/libs/*.jar", Status: "VALIDATED", OwnerProjects: []string{":service-b", ":service-a"}},
	}}
	want := []string{":service-a", ":service-b"}
	if got := proposalOutputOwnerProjects(report); !reflect.DeepEqual(got, want) {
		t.Fatalf("proposalOutputOwnerProjects = %v, want %v", got, want)
	}
	wantEntrypoints := []string{":service-a:jar", ":service-b:jar"}
	if got := proposalOutputOwnerEntrypoints(report, []string{"jar"}); !reflect.DeepEqual(got, wantEntrypoints) {
		t.Fatalf("proposalOutputOwnerEntrypoints = %v, want %v", got, wantEntrypoints)
	}
}
