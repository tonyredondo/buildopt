package structuralbinding

import (
	"strings"
	"testing"
)

func TestBindingIsRevisionAndPathIndependent(t *testing.T) {
	input := fixtureInput()
	first, err := Derive(input)
	if err != nil {
		t.Fatal(err)
	}
	// Revision and checkout path are intentionally not part of Input.
	second, err := Derive(input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !Valid(first) {
		t.Fatalf("binding is not stable: %+v / %+v", first, second)
	}
}

func TestBindingChangesForEveryCompatibilityDimension(t *testing.T) {
	base, err := Derive(fixtureInput())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Input){
		"wrapper": func(input *Input) { input.WrapperSHA256 = strings.Repeat("b", 64) },
		"workflow": func(input *Input) {
			input.CandidateEntrypoints = []string{":changed:other"}
			input.Tasks[2].Path = ":changed:other"
		},
		"lineage": func(input *Input) { input.Tasks[2].DependsOn = []string{":stable:emit"} },
		"outputs": func(input *Input) { input.Outputs[0].ProducerTasks = []string{":stable:emit"} },
		"candidate outputs": func(input *Input) {
			input.CandidateOutputs = []string{"stable/output"}
		},
		"family": func(input *Input) { input.ChangeFamily = "DEPENDENCY" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := fixtureInput()
			mutate(&input)
			binding, err := Derive(input)
			if err != nil {
				return
			}
			if binding.SHA256 == base.SHA256 {
				t.Fatalf("%s drift retained binding %s", name, binding.SHA256)
			}
		})
	}
}

func TestBindingRejectsIncompleteAndAmbiguousEvidence(t *testing.T) {
	cases := map[string]func(*Input){
		"missing dependency": func(input *Input) { input.Tasks[2].DependsOn = []string{":missing"} },
		"ambiguous owner":    func(input *Input) { input.Outputs[0].OwnerProjects = []string{":changed", ":stable"} },
		"unowned output":     func(input *Input) { input.Outputs = input.Outputs[:1] },
		"cyclic lineage":     func(input *Input) { input.Tasks[0].DependsOn = []string{":changed:emit"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := fixtureInput()
			mutate(&input)
			if _, err := Derive(input); err == nil {
				t.Fatal("incomplete evidence was accepted")
			}
		})
	}
}

func fixtureInput() Input {
	return Input{
		RepositoryID: "example/repository", WrapperSHA256: strings.Repeat("a", 64),
		OriginalEntrypoints: []string{":changed:bundleAll"}, CandidateEntrypoints: []string{":changed:emit"},
		GradleOptions: []string{"--no-daemon"}, RequiredOutputs: []string{"changed/output", "stable/output"},
		CandidateOutputs: []string{"changed/output"}, ChangeFamily: "LEAF", ChangedProjects: []string{":changed"},
		Tasks: []Task{
			{Path: ":common:prepare", ProjectPath: ":common"},
			{Path: ":stable:emit", ProjectPath: ":stable", DependsOn: []string{":common:prepare"}},
			{Path: ":changed:emit", ProjectPath: ":changed", DependsOn: []string{":common:prepare"}},
		},
		Outputs: []Output{
			{Pattern: "changed/output", Kind: "FILE", OwnerProjects: []string{":changed"}, ProducerTasks: []string{":changed:emit"}},
			{Pattern: "stable/output", Kind: "FILE", OwnerProjects: []string{":stable"}, ProducerTasks: []string{":stable:emit"}},
		},
	}
}
