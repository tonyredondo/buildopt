package launcher

import (
	"errors"
	"sort"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

var (
	errOptimizeTaskLineageIncomplete = errors.New("Gradle task producer lineage is incomplete")
	errOptimizeTaskLineageAmbiguous  = errors.New("Gradle task producer lineage is ambiguous")
	errOptimizeTaskLineageCyclic     = errors.New("Gradle task producer lineage is cyclic")
)

// optimizeTaskLineage is the exact task-dependency graph observed during the
// useful owner build. It is kept out of optimize state because every captured
// output stores its own revision-bound transitive lineage in the materialized
// manifest.
type optimizeTaskLineage struct {
	dependencies map[string][]string
}

func newOptimizeTaskLineage(tasks []buildimpact.DiscoveredTask) (*optimizeTaskLineage, error) {
	if len(tasks) == 0 {
		return nil, errOptimizeTaskLineageIncomplete
	}
	dependencies := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		if task.Path == "" || task.ProjectPath == "" {
			return nil, errOptimizeTaskLineageIncomplete
		}
		if _, exists := dependencies[task.Path]; exists {
			return nil, errOptimizeTaskLineageAmbiguous
		}
		values := append([]string(nil), task.DependsOn...)
		sort.Strings(values)
		for index, dependency := range values {
			if dependency == "" || dependency == task.Path {
				return nil, errOptimizeTaskLineageIncomplete
			}
			if index > 0 && values[index-1] == dependency {
				return nil, errOptimizeTaskLineageAmbiguous
			}
		}
		dependencies[task.Path] = values
	}
	for _, values := range dependencies {
		for _, dependency := range values {
			if _, exists := dependencies[dependency]; !exists {
				return nil, errOptimizeTaskLineageIncomplete
			}
		}
	}
	lineage := &optimizeTaskLineage{dependencies: dependencies}
	if lineage.cyclic() {
		return nil, errOptimizeTaskLineageCyclic
	}
	return lineage, nil
}

func (lineage *optimizeTaskLineage) ancestors(producers []string) ([]string, error) {
	if lineage == nil || len(producers) == 0 {
		return nil, errOptimizeTaskLineageIncomplete
	}
	direct := make(map[string]bool, len(producers))
	pending := make([]string, 0, len(producers))
	for _, producer := range producers {
		if producer == "" || direct[producer] {
			return nil, errOptimizeTaskLineageAmbiguous
		}
		if _, exists := lineage.dependencies[producer]; !exists {
			return nil, errOptimizeTaskLineageIncomplete
		}
		direct[producer] = true
		pending = append(pending, lineage.dependencies[producer]...)
	}
	ancestors := map[string]bool{}
	for len(pending) > 0 {
		last := len(pending) - 1
		current := pending[last]
		pending = pending[:last]
		if direct[current] || ancestors[current] {
			continue
		}
		dependencies, exists := lineage.dependencies[current]
		if !exists {
			return nil, errOptimizeTaskLineageIncomplete
		}
		ancestors[current] = true
		pending = append(pending, dependencies...)
	}
	result := make([]string, 0, len(ancestors))
	for task := range ancestors {
		result = append(result, task)
	}
	sort.Strings(result)
	return result, nil
}

func (lineage *optimizeTaskLineage) cyclic() bool {
	state := make(map[string]uint8, len(lineage.dependencies))
	var visit func(string) bool
	visit = func(task string) bool {
		switch state[task] {
		case 1:
			return true
		case 2:
			return false
		}
		state[task] = 1
		for _, dependency := range lineage.dependencies[task] {
			if visit(dependency) {
				return true
			}
		}
		state[task] = 2
		return false
	}
	for task := range lineage.dependencies {
		if visit(task) {
			return true
		}
	}
	return false
}
