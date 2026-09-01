// Package historyadmission classifies source changes against an observed Gradle graph.
package historyadmission

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

const (
	FamilyDependency = "DEPENDENCY_SOURCE"
	FamilyResource   = "RESOURCE"
	FamilyLeaf       = "LEAF_SOURCE"
	FamilyMixed      = "MIXED_SOURCE"
)

type Classification struct {
	Owners           []string `json:"owners"`
	Family           string   `json:"family"`
	AffectedProjects []string `json:"affectedProjects"`
}

// SnapshotFromDeclaredGraph preserves the source ownership and dependency facts
// from a strictly validated reviewed graph for source-only history analysis.
func SnapshotFromDeclaredGraph(graph buildimpact.DeclaredGraph) buildimpact.DiscoverySnapshot {
	snapshot := buildimpact.DiscoverySnapshot{Complete: graph.Complete}
	for _, project := range graph.Projects {
		snapshot.Projects = append(snapshot.Projects, buildimpact.DiscoveredProject{
			Path: project.Path, SourcePaths: append([]string(nil), project.SourcePaths...),
			OwnedSourcePaths:     append([]string(nil), project.OwnedSourcePaths...),
			DependsOn:            append([]string(nil), project.DependsOn...),
			UnknownRelationships: project.UnknownRelationships,
		})
	}
	for _, entrypoint := range graph.Entrypoints {
		snapshot.Entrypoints = append(snapshot.Entrypoints, buildimpact.DiscoveredEntrypoint{
			Name: entrypoint.Name, ReachesProjects: append([]string(nil), entrypoint.ReachesProjects...),
			ContainsTestTasks: entrypoint.ContainsTestTasks, UnknownRelationships: entrypoint.UnknownRelationships,
		})
	}
	return snapshot
}

// UnsafeStructuralChange reports changes that can invalidate an observed Gradle graph.
func UnsafeStructuralChange(changedPaths []string) bool {
	for _, changedPath := range changedPaths {
		normalized := strings.ToLower(filepath.ToSlash(changedPath))
		if normalized == "build.gradle" || normalized == "build.gradle.kts" ||
			normalized == "settings.gradle" || normalized == "settings.gradle.kts" ||
			normalized == "gradle.properties" || normalized == "libs.versions.toml" ||
			strings.HasPrefix(normalized, "buildsrc/") ||
			strings.HasPrefix(normalized, "build-logic/") ||
			strings.HasPrefix(normalized, "gradle/") ||
			strings.HasSuffix(normalized, ".gradle") ||
			strings.HasSuffix(normalized, ".gradle.kts") ||
			strings.HasSuffix(normalized, "/gradle.properties") ||
			strings.HasSuffix(normalized, "/libs.versions.toml") {
			return true
		}
	}
	return false
}

func Classify(snapshot buildimpact.DiscoverySnapshot, changedPaths []string) (Classification, error) {
	if !snapshot.Complete || len(snapshot.FallbackReasons) != 0 {
		return Classification{}, errors.New("discovery snapshot is incomplete")
	}
	for _, project := range snapshot.Projects {
		if project.UnknownRelationships {
			return Classification{}, errors.New("discovery snapshot has unknown relationships")
		}
	}
	for _, entrypoint := range snapshot.Entrypoints {
		if entrypoint.UnknownRelationships {
			return Classification{}, errors.New("discovery snapshot has unknown relationships")
		}
	}
	owners, err := buildimpact.ResolveProjectOwners(snapshot, changedPaths)
	if err != nil {
		return Classification{}, err
	}
	return ClassifyOwners(snapshot, changedPaths, owners), nil
}

func ClassifyOwners(snapshot buildimpact.DiscoverySnapshot, changedPaths, owners []string) Classification {
	owners = append([]string(nil), owners...)
	sort.Strings(owners)
	affected := AffectedProjects(snapshot, owners)
	resourceCount := 0
	for _, changedPath := range changedPaths {
		normalized := "/" + strings.ToLower(filepath.ToSlash(changedPath)) + "/"
		if strings.Contains(normalized, "/resources/") {
			resourceCount++
		}
	}
	family := FamilyLeaf
	if len(changedPaths) > 0 && resourceCount == len(changedPaths) {
		family = FamilyResource
	} else if resourceCount > 0 || len(owners) != 1 {
		family = FamilyMixed
	} else if len(affected) > 1 {
		family = FamilyDependency
	}
	return Classification{Owners: owners, Family: family, AffectedProjects: affected}
}

// AffectedProjects returns changed owners and every transitive consumer.
func AffectedProjects(snapshot buildimpact.DiscoverySnapshot, owners []string) []string {
	affected := make(map[string]bool, len(owners))
	for _, owner := range owners {
		affected[owner] = true
	}
	for changed := true; changed; {
		changed = false
		for _, project := range snapshot.Projects {
			if affected[project.Path] {
				continue
			}
			for _, dependency := range project.DependsOn {
				if affected[dependency] {
					affected[project.Path] = true
					changed = true
					break
				}
			}
		}
	}
	result := make([]string, 0, len(affected))
	for project := range affected {
		result = append(result, project)
	}
	sort.Strings(result)
	return result
}
