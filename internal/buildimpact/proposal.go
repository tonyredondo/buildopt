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
		matches := matchingProjectOwners(snapshot, changedPath)
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
