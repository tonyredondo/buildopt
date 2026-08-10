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
