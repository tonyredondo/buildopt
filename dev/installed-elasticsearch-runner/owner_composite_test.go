package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

func TestOwnerCompositeProfileRequiresAcceptedPinnedDecision(t *testing.T) {
	f := ownerTestFreeze{Schema: ownerCompositeSchema, Runtime: correctnessRuntime{Root: t.TempDir()}}
	if _, _, err := ownerProfile(f); err == nil {
		t.Fatal("missing approval accepted")
	}
	f.Schema = "buildopt.eic/owner-test-freeze/v1"
	if root, steps, err := ownerProfile(f); err != nil || root != "owner-tests" || !reflect.DeepEqual(steps, ownerTestSteps()) {
		t.Fatal("legacy profile changed", err)
	}
	f.CompositeDecision = &fileBinding{}
	if _, _, err := ownerProfile(f); err == nil {
		t.Fatal("composite decision in legacy profile accepted")
	}
	f.Schema = ownerCompositeSchema
	p := filepath.Join(f.Runtime.Root, "decision.json")
	raw := []byte(`{"schemaVersion":"buildopt.eic/owner-composite-decision/v1","status":"PROPOSED","proposal":{"path":"/proposal","sha256":"` + ownerCompositeProposalSHA + `"},"instruction":"test"}`)
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	f.CompositeDecision = &fileBinding{p, digest(raw)}
	if _, _, err := ownerProfile(f); err == nil {
		t.Fatal("unaccepted decision accepted")
	}
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(string(raw), "PROPOSED", "ACCEPTED")), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ownerProfile(f); err == nil {
		t.Fatal("changed decision bytes accepted")
	}
}

func TestOwnerCompositeRequestsPreserveNativeEnvironmentAndLegacyProfile(t *testing.T) {
	f := ownerTestFreeze{Schema: ownerCompositeSchema, Runtime: correctnessRuntime{Root: t.TempDir(), Runner: fileBinding{"/runner", strings.Repeat("a", 64)}}}
	f.Runtime.EvidenceRoot = filepath.Join(f.Runtime.Root, "owner-tests-composite-v1")
	for i, step := range ownerCompositeSteps() {
		legacy := ownerTestSteps()[i]
		before, err := correctnessRequestFor(f.Runtime, legacy)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ownerRequestFor(f, step)
		if err != nil {
			t.Fatal(err)
		}
		want := before
		want.Row = step.Row
		want.Arguments = append(append(append([]string{}, before.Arguments[:3]...), step.Row.Arguments...), before.Arguments[3+len(legacy.Row.Arguments):]...)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("request changed beyond approved invocation: %#v", got)
		}
		if step.Row.Arguments[4] != ":build-tools-internal:"+legacy.Row.Arguments[6] || len(step.Row.Arguments) != 7 {
			t.Fatal(step.Row.Arguments)
		}
		if legacy.Row.Arguments[4] != "-p" {
			t.Fatal("legacy command was changed")
		}
		step.Row.Arguments = append(step.Row.Arguments, "--dependency-verification=off")
		if _, err := ownerRequestFor(f, step); err == nil {
			t.Fatal("unapproved argument accepted")
		}
	}
	f.Schema = "unknown"
	if _, err := ownerRequestFor(f, ownerCompositeSteps()[0]); err == nil {
		t.Fatal("unknown profile accepted")
	}
}

func TestOwnerTaskIdentityRequiresExactIncludedBuildAndExecution(t *testing.T) {
	good := gradlecriticalpath.Task{Identity: ":build-tools-internal:test", BuildPath: ":build-tools-internal", Path: ":test", Outcome: "EXECUTED"}
	for _, schema := range []string{ownerCompositeSchema, "buildopt.eic/owner-test-freeze/v1"} {
		f := ownerTestFreeze{Schema: schema}
		want := good
		if schema != ownerCompositeSchema {
			want.Identity = ":test"
			want.BuildPath = ":"
		}
		if err := checkOwnerTask(f, "test", []gradlecriticalpath.Task{want}); err != nil {
			t.Fatal(err)
		}
		for _, mutate := range []func(*gradlecriticalpath.Task){
			func(v *gradlecriticalpath.Task) { v.BuildPath = ":other" },
			func(v *gradlecriticalpath.Task) { v.Identity = ":other:test" },
			func(v *gradlecriticalpath.Task) { v.Path = ":other" },
			func(v *gradlecriticalpath.Task) { v.Outcome = "FROM-CACHE" },
		} {
			bad := want
			mutate(&bad)
			if err := checkOwnerTask(f, "test", []gradlecriticalpath.Task{bad}); err == nil {
				t.Fatal("drift accepted", bad)
			}
		}
		if err := checkOwnerTask(f, "test", []gradlecriticalpath.Task{want, want}); err == nil {
			t.Fatal("duplicate accepted")
		}
	}
}
