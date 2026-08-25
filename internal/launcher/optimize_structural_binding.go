package launcher

import (
	"sort"

	"github.com/tonyredondo/buildopt/internal/structuralbinding"
)

func deriveOptimizeStructuralBinding(invocation optimizeInvocation, discovery optimizeDiscoveryResult) (structuralbinding.Binding, error) {
	tasks := make([]structuralbinding.Task, 0)
	if discovery.taskLineage != nil {
		paths := make([]string, 0, len(discovery.taskLineage.dependencies))
		for path := range discovery.taskLineage.dependencies {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			tasks = append(tasks, structuralbinding.Task{
				Path: path, ProjectPath: discovery.taskLineage.projects[path],
				DependsOn: append([]string(nil), discovery.taskLineage.dependencies[path]...),
			})
		}
	}
	outputs := make([]structuralbinding.Output, 0, len(discovery.outputCandidates))
	for _, output := range discovery.outputCandidates {
		outputs = append(outputs, structuralbinding.Output{
			Pattern: output.Pattern, Kind: output.Kind,
			OwnerProjects: append([]string(nil), output.OwnerProjects...),
			ProducerTasks: append([]string(nil), output.ProducerTasks...),
		})
	}
	return structuralbinding.Derive(structuralbinding.Input{
		RepositoryID: discovery.RepositoryID, WrapperSHA256: invocation.wrapperSHA256,
		OriginalEntrypoints:  append([]string(nil), discovery.Entrypoints...),
		CandidateEntrypoints: append([]string(nil), discovery.CandidateEntrypoints...),
		GradleOptions:        append([]string(nil), invocation.discovery.gradleOptions...),
		RequiredOutputs:      append([]string(nil), discovery.RequiredOutputs...),
		CandidateOutputs:     append([]string(nil), discovery.CandidateOutputs...),
		ChangeFamily:         discovery.ChangeFamily, ChangedProjects: append([]string(nil), discovery.ChangedProjects...),
		Tasks: tasks, Outputs: outputs,
	})
}

func optimizeStructuralBindingMatchesCurrent(entry optimizePortfolioEntry, invocation optimizeInvocation, family string, owners []string) bool {
	if !structuralbinding.Valid(entry.StructuralBinding) || entry.StructuralBinding.WrapperSHA256 != invocation.wrapperSHA256 {
		return false
	}
	repository, err := structuralbinding.RepositoryScopeSHA256(invocation.discovery.RepositoryID)
	if err != nil || repository != entry.StructuralBinding.RepositoryScopeSHA256 {
		return false
	}
	workflow, err := structuralbinding.WorkflowSHA256(invocation.discovery.Entrypoints, entry.CandidateEntrypoints, invocation.discovery.gradleOptions)
	if err != nil || workflow != entry.StructuralBinding.WorkflowSHA256 {
		return false
	}
	changeFamily, err := structuralbinding.ChangeFamilySHA256(family, owners)
	return err == nil && changeFamily == entry.StructuralBinding.ChangeFamilySHA256
}

func validOptimizeStructuralBinding(binding structuralbinding.Binding) bool {
	return structuralbinding.Valid(binding)
}

func emptyOptimizeStructuralBinding(binding structuralbinding.Binding) bool {
	return binding == (structuralbinding.Binding{})
}
