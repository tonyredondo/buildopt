package main

import (
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
	"strings"
	"testing"
	"time"
)

func TestInstalledBoundaryRejectsFalseRecorderTiming(t *testing.T) {
	env := map[string]string{"WCNCP_REPOSITORY_REVISION": strings.Repeat("a", 40), "WCNCP_ENVIRONMENT_CLASS": "CONTROLLED_PERFORMANCE", "WCNCP_PROSPECTIVE_GATE_INPUT": "1", "WCNCP_GRADLE_VERSION": "9.7.1"}
	for _, key := range []string{"WCNCP_SOURCE_TREE_SHA256", "WCNCP_WRAPPER_SHA256", "WCNCP_PACKAGE_SHA256", "WCNCP_JDK_SHA256", "WCNCP_WORKFLOW_SHA256", "WCNCP_ENVIRONMENT_SHA256", "WCNCP_OUTPUT_CONTRACT_SHA256", "WCNCP_OUTPUT_MANIFEST_SHA256"} {
		env[key] = strings.Repeat("b", 64)
	}
	exit := 0
	fact := wcncpobserve.BuildObservation("example/boundary", strings.Repeat("a", 64), func(k string) string { return env[k] }, t.TempDir(), []string{"build"}, []string{"build"}, wcncpobserve.PassthroughResult{Duration: 3001 * time.Millisecond, Child: wcncpobserve.ChildResult{Outcome: "SUCCESS", ExitCode: &exit}})
	outer := boundaryProcess{StartNS: 1e9, EndNS: 7e9, PID: 2, Cgroup: "0::/owned.service"}
	child := boundaryProcess{StartNS: 2e9, EndNS: 5e9, PID: 3, Cgroup: outer.Cgroup}
	observation := boundaryObservation{AtNS: 6e9, Facts: []wcncpobserve.ObservationFacts{fact}}
	if err := validateInstalledBoundary(outer, child, &observation, 0); err != nil {
		t.Fatal(err)
	}
	for _, at := range []int64{4e9, 8e9} {
		bad := observation
		bad.AtNS = at
		if validateInstalledBoundary(outer, child, &bad, 0) == nil {
			t.Fatal("upload outside child/outer boundaries accepted")
		}
	}
	for _, ms := range []int64{1, 7000} {
		bad := boundaryObservation{AtNS: observation.AtNS, Facts: []wcncpobserve.ObservationFacts{fact}}
		bad.Facts[0].Duration.ValueMs = &ms
		if validateInstalledBoundary(outer, child, &bad, 0) == nil {
			t.Fatal("fabricated child duration accepted")
		}
	}
}

func TestInstalledBoundaryRejectsUncontainedNativeProcess(t *testing.T) {
	outer := boundaryProcess{StartNS: 1, EndNS: 10, PID: 2, Cgroup: "0::/owned.service"}
	child := boundaryProcess{StartNS: 2, EndNS: 9, PID: 3, Cgroup: outer.Cgroup}
	if err := validateInstalledBoundary(outer, child, nil, 0); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*boundaryProcess){
		func(p *boundaryProcess) { p.StartNS = 0 },
		func(p *boundaryProcess) { p.EndNS = 11 },
		func(p *boundaryProcess) { p.ExitCode = 1 },
		func(p *boundaryProcess) { p.Cgroup = "0::/foreign.service" },
	} {
		bad := child
		change(&bad)
		if validateInstalledBoundary(outer, bad, nil, 0) == nil {
			t.Fatal("invalid installed child boundary accepted")
		}
	}
}
