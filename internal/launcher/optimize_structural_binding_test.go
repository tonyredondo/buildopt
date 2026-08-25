package launcher

import (
	"testing"

	"github.com/tonyredondo/buildopt/internal/structuralbinding"
)

func testOptimizeStructuralBinding(t *testing.T, entry optimizePortfolioEntry) structuralbinding.Binding {
	t.Helper()
	producer := entry.CandidateEntrypoints[0]
	project := entry.ChangedProjects[0]
	outputs := make([]structuralbinding.Output, 0, len(entry.RequiredOutputs))
	for _, pattern := range entry.RequiredOutputs {
		outputs = append(outputs, structuralbinding.Output{
			Pattern: pattern, Kind: "FILE_GLOB",
			OwnerProjects: []string{project}, ProducerTasks: []string{producer},
		})
	}
	binding, err := structuralbinding.Derive(structuralbinding.Input{
		RepositoryID: entry.RepositoryID, WrapperSHA256: entry.WrapperSHA256,
		OriginalEntrypoints: entry.Entrypoints, CandidateEntrypoints: entry.CandidateEntrypoints,
		GradleOptions: entry.GradleOptions, RequiredOutputs: entry.RequiredOutputs,
		CandidateOutputs: entry.CandidateOutputs, ChangeFamily: entry.Family,
		ChangedProjects: entry.ChangedProjects,
		Tasks:           []structuralbinding.Task{{Path: producer, ProjectPath: project, DependsOn: []string{}}},
		Outputs:         outputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func TestOptimizeStructuralBindingMatchesAcrossRevisionAndRejectsCurrentDrift(t *testing.T) {
	entry := optimizePortfolioEntry{
		Family: optimizeFamilyLeaf, ChangedProjects: []string{":app"}, RepositoryID: "example/repository",
		Entrypoints: []string{"assemble"}, CandidateEntrypoints: []string{":app:assemble"},
		RequiredOutputs: []string{"app/build/libs/app.jar"}, CandidateOutputs: []string{"app/build/libs/app.jar"},
		WrapperSHA256: optimizeDigest("wrapper"), GradleOptions: []string{"--no-daemon"},
	}
	entry.StructuralBinding = testOptimizeStructuralBinding(t, entry)
	invocation := optimizeInvocation{
		wrapperSHA256: entry.WrapperSHA256,
		discovery: optimizeDiscoveryContext{
			RepositoryID: entry.RepositoryID, TargetRevision: "2222222222222222222222222222222222222222",
			Entrypoints: append([]string(nil), entry.Entrypoints...), gradleOptions: append([]string(nil), entry.GradleOptions...),
		},
	}
	if !optimizeStructuralBindingMatchesCurrent(entry, invocation, entry.Family, entry.ChangedProjects) {
		t.Fatal("a new revision with the same structure did not match")
	}

	drifts := []struct {
		name   string
		mutate func(*optimizeInvocation, *string, *[]string)
	}{
		{"wrapper", func(value *optimizeInvocation, _ *string, _ *[]string) {
			value.wrapperSHA256 = optimizeDigest("different-wrapper")
		}},
		{"workflow", func(value *optimizeInvocation, _ *string, _ *[]string) {
			value.discovery.gradleOptions = []string{"--parallel"}
		}},
		{"family", func(_ *optimizeInvocation, family *string, _ *[]string) { *family = optimizeFamilyResource }},
		{"owners", func(_ *optimizeInvocation, _ *string, owners *[]string) { *owners = []string{":other"} }},
	}
	for _, test := range drifts {
		t.Run(test.name, func(t *testing.T) {
			current, family, owners := invocation, entry.Family, append([]string(nil), entry.ChangedProjects...)
			test.mutate(&current, &family, &owners)
			if optimizeStructuralBindingMatchesCurrent(entry, current, family, owners) {
				t.Fatal("structural drift matched the qualified binding")
			}
		})
	}
}
