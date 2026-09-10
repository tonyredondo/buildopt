package main

import (
	"testing"
)

func TestCorrectnessMutationAllocationDoesNotReuseQualificationRows(t *testing.T) {
	f := mutationFreeze{Class: "ELASTICSEARCH_CORRECTNESS_MUTATIONS", Root: t.TempDir(), JDK: "/jdk", Gradle: "/gradle", Runner: fileBinding{Path: "/runner"}}
	task, problems := mutationTestSources(t)
	starts := 0
	for _, step := range makeCorrectnessPlan().Steps {
		if step.Row.Block != "M" {
			continue
		}
		r, err := mutationRequestFor(f, step.Row, "freeze")
		if err != nil {
			t.Fatal(err)
		}
		if r.Row.ID != step.Row.ID || r.Arguments[2] != "0-7" || !containsNativeEnv(r.Arguments, "--max-workers=8") || containsNativeEnv(r.Arguments, "--no-configuration-cache") {
			t.Fatal("public mutation request drift", r.Arguments)
		}
		project, err := prepareMutationProject(f.Root, step.Row, task, problems)
		if err != nil {
			t.Fatal(err)
		}
		if project != r.Directory {
			t.Fatal("project path drift")
		}
		inputs, err := inventory(project, mutationInputSelectors)
		if err != nil {
			t.Fatal(err)
		}
		if err := checkMutationInputs(step.Row, inputs); err != nil {
			t.Fatal(err)
		}
		starts++
	}
	if starts != 24 {
		t.Fatal("wrong public mutation allocation", starts)
	}
	if _, err := mutationRequestFor(f, mutationRows()[0], "freeze"); err == nil {
		t.Fatal("local QM row admitted to public M allocation")
	}
	f.Class = mutationClass
	if _, err := mutationRequestFor(f, makeCorrectnessPlan().Steps[12].Row, "freeze"); err == nil {
		t.Fatal("public M row admitted to historical QM allocation")
	}
}
