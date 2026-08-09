package buildimpact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

const (
	DeclaredGraphSchemaVersion = "buildopt.build-impact/declared-graph/v1"
	maximumDeclaredGraphBytes  = 2 << 20
	DecisionFullGraph          = "FULL_GRAPH"
	DecisionShadowAlternative  = "SHADOW_ALTERNATIVE"
)

var mandatoryGlobalChangePaths = []string{
	"settings.gradle",
	"settings.gradle.kts",
	"**/settings.gradle",
	"**/settings.gradle.kts",
	"build.gradle",
	"build.gradle.kts",
	"**/build.gradle",
	"**/build.gradle.kts",
	"buildSrc/**",
	"build-logic/**",
	"gradle/**",
	"gradle.properties",
	"gradlew",
	"gradlew.bat",
}

type DeclaredGraph struct {
	SchemaVersion     string               `json:"schemaVersion"`
	RepositoryID      string               `json:"repositoryId"`
	PipelineClass     string               `json:"pipelineClass"`
	ManifestDigest    string               `json:"manifestDigest"`
	AdapterVersion    string               `json:"adapterVersion"`
	Complete          bool                 `json:"complete"`
	Projects          []Project            `json:"projects"`
	Entrypoints       []DeclaredEntrypoint `json:"entrypoints"`
	GlobalChangePaths []string             `json:"globalChangePaths"`
}

type Project struct {
	Path                 string   `json:"path"`
	SourcePaths          []string `json:"sourcePaths"`
	OwnedSourcePaths     []string `json:"ownedSourcePaths,omitempty"`
	DependsOn            []string `json:"dependsOn"`
	UnknownRelationships bool     `json:"unknownRelationships"`
}

type DeclaredEntrypoint struct {
	Name                 string   `json:"name"`
	Owner                string   `json:"owner"`
	ReachesProjects      []string `json:"reachesProjects"`
	ArtifactIDs          []string `json:"artifactIds"`
	CheckIDs             []string `json:"checkIds"`
	ContainsTestTasks    bool     `json:"containsTestTasks"`
	UnknownRelationships bool     `json:"unknownRelationships"`
}

type LoadedGraph struct {
	Graph  DeclaredGraph
	Digest string
}

type ImpactDecision struct {
	Mode                   string
	Reason                 string
	ExecutableEntrypoints  []string
	PredictedAlternativeID string
	PredictedEntrypoints   []string
	AffectedProjects       []string
	OmittedProjects        []string
	PreservedTestCheckIDs  []string
	SelectionAuthorized    bool
}

// ParseDeclaredGraph strictly decodes a graph bound to one validated manifest.
func ParseDeclaredGraph(raw []byte, manifest LoadedManifest) (LoadedGraph, error) {
	if len(raw) == 0 || len(raw) > maximumDeclaredGraphBytes {
		return LoadedGraph{}, errors.New("declared graph size is invalid")
	}
	if err := requireDeclaredGraphFields(raw); err != nil {
		return LoadedGraph{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var graph DeclaredGraph
	if err := decoder.Decode(&graph); err != nil {
		return LoadedGraph{}, fmt.Errorf("decode declared graph: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return LoadedGraph{}, errors.New("declared graph has trailing content")
	}
	if err := validateDeclaredGraph(manifest, graph); err != nil {
		return LoadedGraph{}, err
	}
	canonical, err := json.Marshal(graph)
	if err != nil {
		return LoadedGraph{}, fmt.Errorf("canonicalize declared graph: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return LoadedGraph{Graph: defensiveGraph(graph), Digest: "sha256:" + hex.EncodeToString(digest[:])}, nil
}

// EvaluateImpact computes a shadow prediction. The effective entrypoints stay
// equal to the customer's originals until the later BIA-002 gate authorizes use.
func EvaluateImpact(manifest LoadedManifest, graph DeclaredGraph, changedPaths []string) ImpactDecision {
	return evaluateImpact(manifest, graph, changedPaths, true)
}

// evaluatePOCImpact evaluates only the repository-declared required output and
// check scope. It is restricted to the explicit owner-operated POC path; the
// production evaluator above continues to require the complete affected-project
// closure.
func evaluatePOCImpact(manifest LoadedManifest, graph DeclaredGraph, changedPaths []string) ImpactDecision {
	return evaluateImpact(manifest, graph, changedPaths, false)
}

func evaluateImpact(manifest LoadedManifest, graph DeclaredGraph, changedPaths []string, requireAffectedClosure bool) ImpactDecision {
	decision := fullGraphDecision(manifest.Manifest, "GRAPH_INVALID")
	if err := validateDeclaredGraph(manifest, graph); err != nil {
		return decision
	}
	if !graph.Complete {
		decision.Reason = "GRAPH_INCOMPLETE"
		return decision
	}
	for _, project := range graph.Projects {
		if project.UnknownRelationships {
			decision.Reason = "UNKNOWN_RELATIONSHIP"
			return decision
		}
	}
	for _, entrypoint := range graph.Entrypoints {
		if entrypoint.UnknownRelationships {
			decision.Reason = "UNKNOWN_RELATIONSHIP"
			return decision
		}
	}
	if len(changedPaths) == 0 {
		decision.Reason = "NO_DECLARED_CHANGES"
		return decision
	}
	projectByPath := make(map[string]Project, len(graph.Projects))
	for _, project := range graph.Projects {
		projectByPath[project.Path] = project
	}
	changedProjects := map[string]bool{}
	affectedSeeds := map[string]bool{}
	for _, changedPath := range changedPaths {
		if !validRepositoryPath(changedPath) {
			decision.Reason = "INVALID_CHANGE_PATH"
			return decision
		}
		if matchesAnyGlob(appendGlobalPaths(manifest.Manifest.GlobalChangePaths, graph.GlobalChangePaths), changedPath) {
			decision.Reason = "GLOBAL_CHANGE"
			return decision
		}
		affectedMatches := matchingChangedProjects(graph.Projects, changedPath, false, false)
		if len(affectedMatches) == 0 {
			decision.Reason = "UNKNOWN_CHANGE_PATH"
			return decision
		}
		addAll(affectedSeeds, sortedKeys(affectedMatches))
		matches := affectedMatches
		if !requireAffectedClosure {
			matches = matchingChangedProjects(graph.Projects, changedPath, true, true)
			if len(matches) == 0 {
				decision.Reason = "UNKNOWN_CHANGE_PATH"
				return decision
			}
		}
		for projectPath := range matches {
			changedProjects[projectPath] = true
		}
	}
	affected := reverseDependencyClosure(projectByPath, affectedSeeds)
	decision.AffectedProjects = sortedKeys(affected)
	requiredProjects := affected
	if !requireAffectedClosure {
		requiredProjects = changedProjects
	}
	entrypointByName := make(map[string]DeclaredEntrypoint, len(graph.Entrypoints))
	for _, entrypoint := range graph.Entrypoints {
		entrypointByName[entrypoint.Name] = entrypoint
	}
	requiredArtifacts := manifestArtifactIDs(manifest.Manifest)
	requiredBuildChecks, preservedTestChecks := manifestCheckIDs(manifest.Manifest)
	decision.PreservedTestCheckIDs = preservedTestChecks
	type candidate struct {
		alternative EntrypointSet
		covered     map[string]bool
	}
	candidates := make([]candidate, 0, len(manifest.Manifest.AllowedAlternatives))
	for _, alternative := range manifest.Manifest.AllowedAlternatives {
		coveredProjects := map[string]bool{}
		artifacts := map[string]bool{}
		checks := map[string]bool{}
		eligible := true
		for _, name := range alternative.Entrypoints {
			entrypoint, ok := entrypointByName[name]
			if !ok || entrypoint.Owner != BuildOptimization || entrypoint.ContainsTestTasks {
				eligible = false
				break
			}
			addAll(coveredProjects, entrypoint.ReachesProjects)
			addAll(artifacts, entrypoint.ArtifactIDs)
			addAll(checks, entrypoint.CheckIDs)
		}
		if eligible && containsAll(coveredProjects, requiredProjects) && containsAll(artifacts, requiredArtifacts) && containsAll(checks, requiredBuildChecks) {
			candidates = append(candidates, candidate{alternative: alternative, covered: coveredProjects})
		}
	}
	if len(candidates) == 0 {
		decision.Reason = "NO_AUTHORIZED_ALTERNATIVE"
		return decision
	}
	sort.Slice(candidates, func(left, right int) bool {
		leftCount := len(candidates[left].alternative.Entrypoints)
		rightCount := len(candidates[right].alternative.Entrypoints)
		if leftCount != rightCount {
			return leftCount < rightCount
		}
		return candidates[left].alternative.ID < candidates[right].alternative.ID
	})
	selected := candidates[0]
	decision.Mode = DecisionShadowAlternative
	decision.Reason = "CUSTOMER_ALTERNATIVE_AND_DECLARED_GRAPH"
	if !requireAffectedClosure {
		decision.Reason = "CUSTOMER_ALTERNATIVE_AND_DECLARED_OUTPUT_SCOPE"
	}
	decision.PredictedAlternativeID = selected.alternative.ID
	decision.PredictedEntrypoints = append([]string(nil), selected.alternative.Entrypoints...)
	for projectPath := range projectByPath {
		if !selected.covered[projectPath] {
			decision.OmittedProjects = append(decision.OmittedProjects, projectPath)
		}
	}
	sort.Strings(decision.OmittedProjects)
	return decision
}

// matchingChangedProjects keeps production selection conservative while the
// explicit POC path resolves nested project-directory globs to their closest
// declared owner. Equal-specificity matches remain ambiguous and are all kept.
func matchingChangedProjects(projects []Project, changedPath string, mostSpecificOnly, ownedOnly bool) map[string]bool {
	matches := map[string]bool{}
	bestSpecificity := -1
	for _, project := range projects {
		projectSpecificity := -1
		sourcePaths := project.SourcePaths
		if ownedOnly && len(project.OwnedSourcePaths) != 0 {
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
		if mostSpecificOnly && projectSpecificity > bestSpecificity {
			clear(matches)
			bestSpecificity = projectSpecificity
		}
		if !mostSpecificOnly || projectSpecificity == bestSpecificity {
			matches[project.Path] = true
		}
	}
	return matches
}

func repositoryGlobSpecificity(pattern string) int {
	if wildcard := strings.IndexAny(pattern, "*?["); wildcard >= 0 {
		return wildcard
	}
	return len(pattern)
}

func validateDeclaredGraph(manifest LoadedManifest, graph DeclaredGraph) error {
	if graph.SchemaVersion != DeclaredGraphSchemaVersion || graph.RepositoryID != manifest.Manifest.RepositoryID || graph.PipelineClass != manifest.Manifest.PipelineClass {
		return errors.New("declared graph repository or pipeline binding does not match manifest")
	}
	if graph.ManifestDigest != manifest.Digest {
		return fmt.Errorf(
			"declared graph manifest digest %q does not match %q",
			graph.ManifestDigest,
			manifest.Digest,
		)
	}
	if !idPattern.MatchString(graph.AdapterVersion) {
		return errors.New("declared graph adapter version is invalid")
	}
	if len(graph.Projects) == 0 || len(graph.Projects) > 4096 || len(graph.Entrypoints) == 0 || len(graph.Entrypoints) > 4096 || len(graph.GlobalChangePaths) > 256 {
		return errors.New("declared graph collections are outside bounds")
	}
	projects := map[string]bool{}
	for _, project := range graph.Projects {
		if !validGradleProject(project.Path) || projects[project.Path] || len(project.SourcePaths) == 0 || len(project.SourcePaths) > 256 || len(project.OwnedSourcePaths) > 256 || len(project.DependsOn) > 1024 {
			return errors.New("declared graph project is invalid")
		}
		projects[project.Path] = true
		if !uniqueSafeGlobs(project.SourcePaths) || !uniqueSafeGlobs(project.OwnedSourcePaths) || !uniqueStrings(project.DependsOn) {
			return errors.New("declared graph project paths or dependencies are invalid")
		}
		if len(project.OwnedSourcePaths) != 0 {
			sourceSet := map[string]bool{}
			addAll(sourceSet, project.SourcePaths)
			for _, owned := range project.OwnedSourcePaths {
				if !sourceSet[owned] {
					return errors.New("declared graph owned source is outside its conservative source boundary")
				}
			}
		}
	}
	for _, project := range graph.Projects {
		for _, dependency := range project.DependsOn {
			if !projects[dependency] || dependency == project.Path {
				return errors.New("declared graph dependency is missing or self-referential")
			}
		}
	}
	if hasDependencyCycle(graph.Projects) {
		return errors.New("declared graph project dependencies contain a cycle")
	}
	artifactOwners := map[string]string{}
	for _, artifact := range manifest.Manifest.RequiredArtifacts {
		artifactOwners[artifact.ID] = artifact.Owner
	}
	checkOwners := map[string]string{}
	for _, check := range manifest.Manifest.RequiredChecks {
		checkOwners[check.ID] = check.Owner
	}
	entrypoints := map[string]bool{}
	for _, entrypoint := range graph.Entrypoints {
		if !validGradleEntrypoint(entrypoint.Name) || entrypoints[entrypoint.Name] || entrypoint.Owner != BuildOptimization || len(entrypoint.ReachesProjects) == 0 || len(entrypoint.ReachesProjects) > 4096 {
			return errors.New("declared graph entrypoint is invalid")
		}
		entrypoints[entrypoint.Name] = true
		if !uniqueStrings(entrypoint.ReachesProjects) || !uniqueStrings(entrypoint.ArtifactIDs) || !uniqueStrings(entrypoint.CheckIDs) {
			return errors.New("declared graph entrypoint contains duplicate references")
		}
		for _, projectPath := range entrypoint.ReachesProjects {
			if !projects[projectPath] {
				return errors.New("declared graph entrypoint reaches an unknown project")
			}
		}
		for _, artifactID := range entrypoint.ArtifactIDs {
			if artifactOwners[artifactID] != BuildOptimization {
				return errors.New("declared graph entrypoint references an unknown artifact")
			}
		}
		for _, checkID := range entrypoint.CheckIDs {
			if checkOwners[checkID] != BuildOptimization {
				return errors.New("declared graph entrypoint cannot own Test Optimization checks")
			}
		}
	}
	for _, required := range allManifestEntrypoints(manifest.Manifest) {
		if !entrypoints[required] {
			return errors.New("declared graph omits a customer manifest entrypoint")
		}
	}
	if !uniqueSafeGlobs(graph.GlobalChangePaths) {
		return errors.New("declared graph global paths are invalid")
	}
	return nil
}

func requireDeclaredGraphFields(raw []byte) error {
	var presence struct {
		Complete          *bool     `json:"complete"`
		GlobalChangePaths *[]string `json:"globalChangePaths"`
		Projects          []struct {
			UnknownRelationships *bool     `json:"unknownRelationships"`
			SourcePaths          *[]string `json:"sourcePaths"`
			DependsOn            *[]string `json:"dependsOn"`
		} `json:"projects"`
		Entrypoints []struct {
			ContainsTestTasks    *bool     `json:"containsTestTasks"`
			UnknownRelationships *bool     `json:"unknownRelationships"`
			ReachesProjects      *[]string `json:"reachesProjects"`
			ArtifactIDs          *[]string `json:"artifactIds"`
			CheckIDs             *[]string `json:"checkIds"`
		} `json:"entrypoints"`
	}
	if err := json.Unmarshal(raw, &presence); err != nil {
		return fmt.Errorf("inspect declared graph fields: %w", err)
	}
	if presence.Complete == nil || presence.GlobalChangePaths == nil {
		return errors.New("declared graph must explicitly state completeness and global paths")
	}
	for _, project := range presence.Projects {
		if project.UnknownRelationships == nil || project.SourcePaths == nil || project.DependsOn == nil {
			return errors.New("declared graph projects must explicitly state sources, dependencies, and unknown relationships")
		}
	}
	for _, entrypoint := range presence.Entrypoints {
		if entrypoint.ContainsTestTasks == nil || entrypoint.UnknownRelationships == nil || entrypoint.ReachesProjects == nil || entrypoint.ArtifactIDs == nil || entrypoint.CheckIDs == nil {
			return errors.New("declared graph entrypoints must explicitly state reach, artifacts, checks, Test tasks, and unknown relationships")
		}
	}
	return nil
}

func fullGraphDecision(manifest Manifest, reason string) ImpactDecision {
	_, testChecks := manifestCheckIDs(manifest)
	return ImpactDecision{
		Mode:                  DecisionFullGraph,
		Reason:                reason,
		ExecutableEntrypoints: append([]string(nil), manifest.OriginalEntrypoints...),
		PredictedEntrypoints:  append([]string(nil), manifest.OriginalEntrypoints...),
		PreservedTestCheckIDs: testChecks,
		SelectionAuthorized:   false,
	}
}

func validGradleProject(projectPath string) bool {
	if projectPath == ":" {
		return true
	}
	if !strings.HasPrefix(projectPath, ":") || !validGradleEntrypoint(projectPath+":placeholder") {
		return false
	}
	return true
}

func validRepositoryPath(candidate string) bool {
	return validRepositoryGlob(candidate) && !strings.ContainsAny(candidate, "*?[")
}

func matchRepositoryGlob(pattern, candidate string) bool {
	patternParts := strings.Split(pattern, "/")
	candidateParts := strings.Split(candidate, "/")
	var match func(int, int) bool
	match = func(patternIndex, candidateIndex int) bool {
		if patternIndex == len(patternParts) {
			return candidateIndex == len(candidateParts)
		}
		if patternParts[patternIndex] == "**" {
			for next := candidateIndex; next <= len(candidateParts); next++ {
				if match(patternIndex+1, next) {
					return true
				}
			}
			return false
		}
		if candidateIndex == len(candidateParts) {
			return false
		}
		matched, err := path.Match(patternParts[patternIndex], candidateParts[candidateIndex])
		return err == nil && matched && match(patternIndex+1, candidateIndex+1)
	}
	return match(0, 0)
}

func matchesAnyGlob(patterns []string, candidate string) bool {
	for _, pattern := range patterns {
		if matchRepositoryGlob(pattern, candidate) {
			return true
		}
	}
	return false
}

func appendGlobalPaths(manifestPaths, graphPaths []string) []string {
	result := make([]string, 0, len(mandatoryGlobalChangePaths)+len(manifestPaths)+len(graphPaths))
	result = append(result, mandatoryGlobalChangePaths...)
	result = append(result, manifestPaths...)
	return append(result, graphPaths...)
}

func reverseDependencyClosure(projects map[string]Project, initial map[string]bool) map[string]bool {
	result := map[string]bool{}
	for projectPath := range initial {
		result[projectPath] = true
	}
	changed := true
	for changed {
		changed = false
		for projectPath, project := range projects {
			if result[projectPath] {
				continue
			}
			for _, dependency := range project.DependsOn {
				if result[dependency] {
					result[projectPath] = true
					changed = true
					break
				}
			}
		}
	}
	return result
}

func manifestArtifactIDs(manifest Manifest) map[string]bool {
	result := map[string]bool{}
	for _, artifact := range manifest.RequiredArtifacts {
		result[artifact.ID] = true
	}
	return result
}

func manifestCheckIDs(manifest Manifest) (map[string]bool, []string) {
	buildChecks := map[string]bool{}
	testChecks := []string{}
	for _, check := range manifest.RequiredChecks {
		if check.Owner == BuildOptimization {
			buildChecks[check.ID] = true
		} else if check.Owner == TestOptimization {
			testChecks = append(testChecks, check.ID)
		}
	}
	sort.Strings(testChecks)
	return buildChecks, testChecks
}

func allManifestEntrypoints(manifest Manifest) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, name := range manifest.OriginalEntrypoints {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	for _, alternative := range manifest.AllowedAlternatives {
		for _, name := range alternative.Entrypoints {
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

func uniqueSafeGlobs(values []string) bool {
	if len(values) == 0 {
		return true
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !validRepositoryGlob(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func uniqueStrings(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func containsAll(have, required map[string]bool) bool {
	for value := range required {
		if !have[value] {
			return false
		}
	}
	return true
}

func addAll(destination map[string]bool, values []string) {
	for _, value := range values {
		destination[value] = true
	}
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func hasDependencyCycle(projects []Project) bool {
	dependencies := map[string][]string{}
	for _, project := range projects {
		dependencies[project.Path] = project.DependsOn
	}
	state := map[string]uint8{}
	var visit func(string) bool
	visit = func(projectPath string) bool {
		if state[projectPath] == 1 {
			return true
		}
		if state[projectPath] == 2 {
			return false
		}
		state[projectPath] = 1
		for _, dependency := range dependencies[projectPath] {
			if visit(dependency) {
				return true
			}
		}
		state[projectPath] = 2
		return false
	}
	for projectPath := range dependencies {
		if visit(projectPath) {
			return true
		}
	}
	return false
}

func defensiveGraph(graph DeclaredGraph) DeclaredGraph {
	graph.Projects = cloneSlice(graph.Projects)
	for index := range graph.Projects {
		graph.Projects[index].SourcePaths = cloneSlice(graph.Projects[index].SourcePaths)
		graph.Projects[index].OwnedSourcePaths = cloneSlice(graph.Projects[index].OwnedSourcePaths)
		graph.Projects[index].DependsOn = cloneSlice(graph.Projects[index].DependsOn)
	}
	graph.Entrypoints = cloneSlice(graph.Entrypoints)
	for index := range graph.Entrypoints {
		graph.Entrypoints[index].ReachesProjects = cloneSlice(graph.Entrypoints[index].ReachesProjects)
		graph.Entrypoints[index].ArtifactIDs = cloneSlice(graph.Entrypoints[index].ArtifactIDs)
		graph.Entrypoints[index].CheckIDs = cloneSlice(graph.Entrypoints[index].CheckIDs)
	}
	graph.GlobalChangePaths = cloneSlice(graph.GlobalChangePaths)
	return graph
}
