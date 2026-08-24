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

// ErrConfigurationInputOwnershipUnproven means every changed path is outside
// declared source roots and has no task consumer in an otherwise complete
// requested graph. Such a path may still affect Gradle configuration, so the
// caller must retain the native workflow.
var ErrConfigurationInputOwnershipUnproven = errors.New("configuration input ownership is unproven")

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

// WorkflowProjectOwnership explains how complete workflow evidence resolved
// each path that did not have one direct source owner.
type WorkflowProjectOwnership struct {
	Owners               []string
	IgnoredPaths         []string
	ConsumedUnownedPaths []string
	UnattributedPaths    []string
}

// ResolveWorkflowProjectOwnership extends strict source ownership with exact
// consuming-task ownership from a complete requested graph. Unowned paths with
// no consumer are ignorable only when another changed path proves the affected
// project set; an all-unattributed change remains fail-closed because it may be
// read during Gradle configuration.
func ResolveWorkflowProjectOwnership(snapshot DiscoverySnapshot, observation WorkflowInputRelevance, changedPaths []string) (WorkflowProjectOwnership, error) {
	if err := validateWorkflowInputRelevance(observation, changedPaths); err != nil {
		return WorkflowProjectOwnership{}, err
	}
	if !observation.Complete {
		return WorkflowProjectOwnership{}, errors.New("workflow-input observation is incomplete")
	}
	inputs := make(map[string]WorkflowInputPath, len(observation.Paths))
	for _, entry := range observation.Paths {
		inputs[entry.Path] = entry
	}
	projects := make(map[string]bool, len(snapshot.Projects))
	for _, project := range snapshot.Projects {
		projects[project.Path] = true
	}
	taskProjects := make(map[string]string, len(snapshot.Tasks))
	for _, task := range snapshot.Tasks {
		if previous, exists := taskProjects[task.Path]; exists && previous != task.ProjectPath {
			return WorkflowProjectOwnership{}, errors.New("workflow task ownership is ambiguous")
		}
		if !projects[task.ProjectPath] {
			return WorkflowProjectOwnership{}, errors.New("workflow task references an unknown project")
		}
		taskProjects[task.Path] = task.ProjectPath
	}
	owners := map[string]bool{}
	consumedUnowned := make([]string, 0)
	unattributed := make([]string, 0)
	for _, changedPath := range changedPaths {
		matches := matchingProjectOwners(snapshot, changedPath)
		switch len(matches) {
		case 1:
			for owner := range matches {
				owners[owner] = true
			}
		case 0:
			consumers := inputs[changedPath].ConsumingTasks
			if len(consumers) == 0 {
				unattributed = append(unattributed, changedPath)
				continue
			}
			for _, consumer := range consumers {
				project, exists := taskProjects[consumer]
				if !exists {
					return WorkflowProjectOwnership{}, errors.New("workflow input references an unknown consuming task")
				}
				owners[project] = true
			}
			consumedUnowned = append(consumedUnowned, changedPath)
		default:
			return WorkflowProjectOwnership{}, errors.New("changed path has ambiguous Gradle project ownership")
		}
	}
	result := make([]string, 0, len(owners))
	for owner := range owners {
		result = append(result, owner)
	}
	sort.Strings(result)
	sort.Strings(consumedUnowned)
	sort.Strings(unattributed)
	resolution := WorkflowProjectOwnership{
		Owners:               result,
		ConsumedUnownedPaths: consumedUnowned,
		UnattributedPaths:    unattributed,
	}
	if len(result) == 0 {
		return resolution, ErrConfigurationInputOwnershipUnproven
	}
	resolution.IgnoredPaths = append([]string(nil), unattributed...)
	return resolution, nil
}

// ResolveWorkflowProjectOwners preserves the original compact API for callers
// that need only the proven projects and safely ignored paths.
func ResolveWorkflowProjectOwners(snapshot DiscoverySnapshot, observation WorkflowInputRelevance, changedPaths []string) ([]string, []string, error) {
	resolution, err := ResolveWorkflowProjectOwnership(snapshot, observation, changedPaths)
	return resolution.Owners, resolution.IgnoredPaths, err
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
