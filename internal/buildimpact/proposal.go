package buildimpact

import (
	"errors"
	"sort"
)

// ResolveProjectOwners maps exact changed paths to one most-specific Gradle
// project each. Direct ownership boundaries take precedence over conservative
// transitive input closures when the verified graph supplies both. Equal-
// specificity ownership and unowned paths fail closed.
func ResolveProjectOwners(snapshot DiscoverySnapshot, changedPaths []string) ([]string, error) {
	if len(changedPaths) == 0 {
		return nil, errors.New("proposal requires at least one changed path")
	}
	owners := map[string]bool{}
	for _, changedPath := range changedPaths {
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
		if len(matches) != 1 {
			return nil, errors.New("changed path has missing or ambiguous Gradle project ownership")
		}
		for projectPath := range matches {
			owners[projectPath] = true
		}
	}
	result := make([]string, 0, len(owners))
	for owner := range owners {
		result = append(result, owner)
	}
	sort.Strings(result)
	return result, nil
}
