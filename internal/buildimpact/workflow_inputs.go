package buildimpact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

const (
	WorkflowInputRelevanceSchemaVersion = "buildopt.build-impact/workflow-input-relevance/v1"
	maximumWorkflowInputBytes           = 1 << 20
	maximumWorkflowInputPaths           = 4096
	maximumWorkflowInputConsumers       = 262144
)

// WorkflowInputRelevance records whether changed repository paths are declared
// inputs of the exact Gradle task graph requested by the owner. It is bounded
// evidence for one invocation and never authorizes omission when incomplete.
type WorkflowInputRelevance struct {
	SchemaVersion   string              `json:"schemaVersion"`
	Complete        bool                `json:"complete"`
	FallbackReasons []string            `json:"fallbackReasons"`
	Paths           []WorkflowInputPath `json:"paths"`
}

// WorkflowInputPath names the Gradle tasks that declared one changed path as
// an input. An empty ConsumingTasks collection means NOT_CONSUMED only when the
// enclosing observation is complete.
type WorkflowInputPath struct {
	Path           string   `json:"path"`
	ConsumingTasks []string `json:"consumingTasks"`
}

// ParseWorkflowInputRelevance strictly binds Gradle's observation to the exact
// changed-path set supplied by the launcher.
func ParseWorkflowInputRelevance(raw []byte, changedPaths []string) (WorkflowInputRelevance, error) {
	if len(raw) == 0 || len(raw) > maximumWorkflowInputBytes {
		return WorkflowInputRelevance{}, errors.New("workflow-input observation size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var observation WorkflowInputRelevance
	if err := decoder.Decode(&observation); err != nil {
		return WorkflowInputRelevance{}, fmt.Errorf("decode workflow-input observation: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return WorkflowInputRelevance{}, errors.New("workflow-input observation has trailing content")
	}
	if err := validateWorkflowInputRelevance(observation, changedPaths); err != nil {
		return WorkflowInputRelevance{}, err
	}
	sort.Slice(observation.Paths, func(left, right int) bool {
		return observation.Paths[left].Path < observation.Paths[right].Path
	})
	for index := range observation.Paths {
		sort.Strings(observation.Paths[index].ConsumingTasks)
	}
	return observation, nil
}

func validateWorkflowInputRelevance(observation WorkflowInputRelevance, changedPaths []string) error {
	if observation.SchemaVersion != WorkflowInputRelevanceSchemaVersion ||
		len(changedPaths) == 0 || len(changedPaths) > maximumWorkflowInputPaths ||
		len(observation.Paths) != len(changedPaths) ||
		!uniqueStrings(observation.FallbackReasons) ||
		(observation.Complete && len(observation.FallbackReasons) != 0) ||
		(!observation.Complete && len(observation.FallbackReasons) == 0) {
		return errors.New("workflow-input observation identity or completeness is invalid")
	}
	allowedFallbacks := map[string]bool{
		"CHANGED_PATH_UNRESOLVED":       true,
		"TASK_INPUTS_UNAVAILABLE":       true,
		"TASK_INPUT_SYMLINK_UNRESOLVED": true,
	}
	for _, reason := range observation.FallbackReasons {
		if !allowedFallbacks[reason] {
			return errors.New("workflow-input observation contains an unknown fallback reason")
		}
	}
	wanted := append([]string(nil), changedPaths...)
	sort.Strings(wanted)
	if !uniqueStrings(wanted) {
		return errors.New("workflow-input changed paths are duplicated")
	}
	consumers := 0
	seen := make(map[string]bool, len(observation.Paths))
	observed := make([]string, 0, len(observation.Paths))
	for _, entry := range observation.Paths {
		if !validRepositoryPath(entry.Path) || seen[entry.Path] || !uniqueStrings(entry.ConsumingTasks) {
			return errors.New("workflow-input observation contains an invalid path")
		}
		seen[entry.Path] = true
		observed = append(observed, entry.Path)
		consumers += len(entry.ConsumingTasks)
		for _, task := range entry.ConsumingTasks {
			if !validGradleEntrypoint(task) {
				return errors.New("workflow-input observation contains an invalid consuming task")
			}
		}
	}
	if consumers > maximumWorkflowInputConsumers {
		return errors.New("workflow-input observation has too many consuming tasks")
	}
	sort.Strings(observed)
	for index, path := range wanted {
		if observed[index] != path {
			return errors.New("workflow-input observation does not cover the exact changed-path set")
		}
	}
	return nil
}

// ResolveWorkflowProjectOwners extends strict project ownership only for paths
// that a complete observation proves are outside the requested workflow. It
// never suppresses ambiguous ownership or a path consumed by any task.
func ResolveWorkflowProjectOwners(snapshot DiscoverySnapshot, observation WorkflowInputRelevance, changedPaths []string) ([]string, []string, error) {
	if err := validateWorkflowInputRelevance(observation, changedPaths); err != nil {
		return nil, nil, err
	}
	if !observation.Complete {
		return nil, nil, errors.New("workflow-input observation is incomplete")
	}
	inputs := make(map[string]WorkflowInputPath, len(observation.Paths))
	for _, entry := range observation.Paths {
		inputs[entry.Path] = entry
	}
	owners := map[string]bool{}
	ignored := make([]string, 0)
	for _, changedPath := range changedPaths {
		matches := matchingProjectOwners(snapshot, changedPath)
		switch len(matches) {
		case 1:
			for owner := range matches {
				owners[owner] = true
			}
		case 0:
			if len(inputs[changedPath].ConsumingTasks) != 0 {
				return nil, nil, errors.New("unowned changed path is consumed by the requested workflow")
			}
			ignored = append(ignored, changedPath)
		default:
			return nil, nil, errors.New("changed path has ambiguous Gradle project ownership")
		}
	}
	result := make([]string, 0, len(owners))
	for owner := range owners {
		result = append(result, owner)
	}
	sort.Strings(result)
	sort.Strings(ignored)
	if len(result) == 0 {
		return nil, nil, errors.New("workflow-input filtering removed every owned change")
	}
	return result, ignored, nil
}

func matchingProjectOwners(snapshot DiscoverySnapshot, changedPath string) map[string]bool {
	bestSpecificity := -1
	matches := map[string]bool{}
	for _, project := range snapshot.Projects {
		projectSpecificity := -1
		sourcePaths := project.SourcePaths
		if len(project.OwnedSourcePaths) != 0 {
			sourcePaths = project.OwnedSourcePaths
		}
		for _, sourcePath := range sourcePaths {
			if matchRepositoryGlob(sourcePath, changedPath) {
				projectSpecificity = max(projectSpecificity, repositoryGlobSpecificity(sourcePath))
			}
		}
		if projectSpecificity < 0 {
			continue
		}
		if projectSpecificity > bestSpecificity {
			matches = map[string]bool{}
			bestSpecificity = projectSpecificity
		}
		if projectSpecificity == bestSpecificity {
			matches[project.Path] = true
		}
	}
	return matches
}
