package buildimpact

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	DiscoverySchemaVersion         = "buildopt.build-impact/gradle-discovery/v1"
	GeneratedManifestSchemaVersion = "buildopt.build-impact/generated-manifest/v1"
	GradleDiscoveryAdapterVersion  = "gradle-runtime-v1"
	maximumDiscoveryBytes          = 4 << 20
	maximumDiscoveredTasks         = 262144
)

//go:embed discovery.init.gradle
var discoveryInitScript []byte

type DiscoverySnapshot struct {
	SchemaVersion      string                 `json:"schemaVersion"`
	GradleVersion      string                 `json:"gradleVersion"`
	Complete           bool                   `json:"complete"`
	FallbackReasons    []string               `json:"fallbackReasons"`
	IncludedBuildPaths []string               `json:"includedBuildPaths,omitempty"`
	Projects           []DiscoveredProject    `json:"projects"`
	Tasks              []DiscoveredTask       `json:"tasks,omitempty"`
	Entrypoints        []DiscoveredEntrypoint `json:"entrypoints"`
}

type DiscoveredTask struct {
	Path        string   `json:"path"`
	ProjectPath string   `json:"projectPath"`
	DependsOn   []string `json:"dependsOn"`
}

type DiscoveredProject struct {
	Path                 string   `json:"path"`
	SourcePaths          []string `json:"sourcePaths"`
	OwnedSourcePaths     []string `json:"-"`
	DependsOn            []string `json:"dependsOn"`
	UnknownRelationships bool     `json:"unknownRelationships"`
}

type DiscoveredEntrypoint struct {
	Name                 string   `json:"name"`
	ReachesProjects      []string `json:"reachesProjects"`
	ContainsTestTasks    bool     `json:"containsTestTasks"`
	UnknownRelationships bool     `json:"unknownRelationships"`
}

// GeneratedManifest binds reviewable generated state to the customer-owned
// policy manifest and the canonical generated graph. It is evidence, never a
// replacement source of omission authority.
type GeneratedManifest struct {
	SchemaVersion   string   `json:"schemaVersion"`
	RepositoryID    string   `json:"repositoryId"`
	PipelineClass   string   `json:"pipelineClass"`
	ManifestDigest  string   `json:"manifestDigest"`
	GraphDigest     string   `json:"graphDigest"`
	DiscoveryDigest string   `json:"discoveryDigest"`
	AdapterVersion  string   `json:"adapterVersion"`
	GradleVersion   string   `json:"gradleVersion"`
	Complete        bool     `json:"complete"`
	FallbackReasons []string `json:"fallbackReasons"`
}

type DiscoveryOptions struct {
	RepositoryRoot string
	ManifestPath   string
	RepositoryID   string
	PipelineClass  string
	GradleCommand  string
	GradleArgs     []string
}

// ObservationOptions names one read-only Gradle model observation. It is used
// before a repository manifest exists so onboarding can propose, but never
// authorize, a smaller workflow from typed Gradle relationships.
type ObservationOptions struct {
	RepositoryRoot string
	Entrypoints    []string
	GradleCommand  string
	GradleArgs     []string
}

// InlineObservation describes a graph snapshot collected by the owner's
// useful Gradle invocation instead of a second configuration-only build.
type InlineObservation struct {
	InitPath          string
	OutputPath        string
	EntrypointsJSON   string
	WorkflowInputPath string
	ChangedPathsJSON  string
}

// PrepareInlineObservation writes the same fail-closed discovery init script
// used by ObserveGradle, configured to preserve and share the owner's task work.
func PrepareInlineObservation(directory string, entrypoints, changedPaths []string) (InlineObservation, error) {
	if err := validateEntrypoints(entrypoints); err != nil {
		return InlineObservation{}, fmt.Errorf("invalid inline discovery entrypoints: %w", err)
	}
	if len(changedPaths) == 0 || len(changedPaths) > maximumWorkflowInputPaths || !uniqueStrings(changedPaths) {
		return InlineObservation{}, errors.New("inline discovery changed paths are invalid")
	}
	for _, path := range changedPaths {
		if !validRepositoryPath(path) {
			return InlineObservation{}, errors.New("inline discovery changed path is unsafe")
		}
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return InlineObservation{}, fmt.Errorf("create inline discovery directory: %w", err)
	}
	entrypointJSON, err := json.Marshal(entrypoints)
	if err != nil {
		return InlineObservation{}, fmt.Errorf("encode inline discovery entrypoints: %w", err)
	}
	changedPathJSON, err := json.Marshal(changedPaths)
	if err != nil {
		return InlineObservation{}, fmt.Errorf("encode inline discovery changed paths: %w", err)
	}
	observation := InlineObservation{
		InitPath:          filepath.Join(directory, "impact-discovery.init.gradle"),
		OutputPath:        filepath.Join(directory, "impact-snapshot.json"),
		EntrypointsJSON:   string(entrypointJSON),
		WorkflowInputPath: filepath.Join(directory, "workflow-inputs.json"),
		ChangedPathsJSON:  string(changedPathJSON),
	}
	if err := os.WriteFile(observation.InitPath, discoveryInitScript, 0o600); err != nil {
		return InlineObservation{}, fmt.Errorf("write inline discovery init script: %w", err)
	}
	_ = os.Remove(observation.OutputPath)
	_ = os.Remove(observation.WorkflowInputPath)
	return observation, nil
}

// ReadInlineObservation validates the snapshot emitted by a useful build.
func ReadInlineObservation(observation InlineObservation, entrypoints []string) (DiscoverySnapshot, error) {
	raw, err := os.ReadFile(observation.OutputPath)
	if err != nil {
		return DiscoverySnapshot{}, fmt.Errorf("read inline Gradle discovery: %w", err)
	}
	snapshot, _, err := parseDiscoverySnapshotForEntrypoints(raw, entrypoints, true)
	return snapshot, err
}

// ReadWorkflowInputRelevance validates the changed-path evidence emitted beside
// an inline task-graph observation.
func ReadWorkflowInputRelevance(observation InlineObservation, changedPaths []string) (WorkflowInputRelevance, error) {
	raw, err := os.ReadFile(observation.WorkflowInputPath)
	if err != nil {
		return WorkflowInputRelevance{}, fmt.Errorf("read workflow-input observation: %w", err)
	}
	return ParseWorkflowInputRelevance(raw, changedPaths)
}

// ParseObservedDiscoverySnapshot validates a stored read-only task-graph
// observation for the exact requested entrypoints. It supports POC evidence
// tools without exposing manifest-authorized profile generation.
func ParseObservedDiscoverySnapshot(raw []byte, entrypoints []string) (DiscoverySnapshot, error) {
	snapshot, _, err := parseDiscoverySnapshotForEntrypoints(raw, entrypoints, true)
	return snapshot, err
}

// DeriveProjectEntrypoints creates conservative per-project lifecycle reaches
// from the exact configured task graph when available, falling back to the
// complete project-dependency graph. Callers must separately prove that each
// requested selector is a conventional output producer.
func DeriveProjectEntrypoints(snapshot DiscoverySnapshot, entrypoints []string) (DiscoverySnapshot, error) {
	if !snapshot.Complete || len(snapshot.Projects) == 0 || len(entrypoints) == 0 {
		return DiscoverySnapshot{}, errors.New("complete project graph and candidate entrypoints are required")
	}
	projects := make(map[string]DiscoveredProject, len(snapshot.Projects))
	for _, project := range snapshot.Projects {
		if project.Path == "" || project.UnknownRelationships {
			return DiscoverySnapshot{}, errors.New("candidate project graph contains an unknown relationship")
		}
		projects[project.Path] = project
	}
	tasks := make(map[string]DiscoveredTask, len(snapshot.Tasks))
	for _, task := range snapshot.Tasks {
		tasks[task.Path] = task
	}
	derived := snapshot
	derived.Entrypoints = append([]DiscoveredEntrypoint(nil), snapshot.Entrypoints...)
	seen := make(map[string]bool, len(snapshot.Entrypoints)+len(entrypoints))
	for _, entrypoint := range snapshot.Entrypoints {
		seen[entrypoint.Name] = true
	}
	for _, entrypoint := range entrypoints {
		if seen[entrypoint] {
			continue
		}
		seen[entrypoint] = true
		separator := strings.LastIndex(entrypoint, ":")
		if separator < 0 || separator == len(entrypoint)-1 {
			return DiscoverySnapshot{}, fmt.Errorf("candidate entrypoint %q is not project-qualified", entrypoint)
		}
		owner := entrypoint[:separator]
		if owner == "" {
			owner = ":"
		}
		if _, ok := projects[owner]; !ok {
			return DiscoverySnapshot{}, fmt.Errorf("candidate entrypoint %q has no project", entrypoint)
		}
		reached := map[string]bool{}
		if _, exact := tasks[entrypoint]; exact {
			reachedTasks := map[string]bool{}
			pendingTasks := []string{entrypoint}
			for len(pendingTasks) > 0 {
				current := pendingTasks[len(pendingTasks)-1]
				pendingTasks = pendingTasks[:len(pendingTasks)-1]
				if reachedTasks[current] {
					continue
				}
				task, ok := tasks[current]
				if !ok {
					return DiscoverySnapshot{}, fmt.Errorf("candidate task %q has an unknown dependency", current)
				}
				reachedTasks[current] = true
				reached[task.ProjectPath] = true
				pendingTasks = append(pendingTasks, task.DependsOn...)
			}
		}
		pending := []string{owner}
		if len(reached) == 0 {
			for len(pending) > 0 {
				current := pending[len(pending)-1]
				pending = pending[:len(pending)-1]
				if reached[current] {
					continue
				}
				project, ok := projects[current]
				if !ok {
					return DiscoverySnapshot{}, fmt.Errorf("candidate project %q has an unknown dependency", current)
				}
				reached[current] = true
				pending = append(pending, project.DependsOn...)
			}
		}
		reaches := make([]string, 0, len(reached))
		for project := range reached {
			reaches = append(reaches, project)
		}
		sort.Strings(reaches)
		derived.Entrypoints = append(derived.Entrypoints, DiscoveredEntrypoint{
			Name: entrypoint, ReachesProjects: reaches,
		})
	}
	return derived, nil
}

type GeneratedImpact struct {
	Manifest       LoadedManifest
	Graph          LoadedGraph
	GraphJSON      []byte
	Generated      GeneratedManifest
	GeneratedJSON  []byte
	Snapshot       DiscoverySnapshot
	SnapshotDigest string
}

// Discover runs Gradle's configured model and task-dependency APIs, then
// converts their output into a deterministic, fail-closed impact graph.
func Discover(ctx context.Context, options DiscoveryOptions) (GeneratedImpact, error) {
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("resolve repository root: %w", err)
	}
	root = filepath.Clean(root)
	manifest, err := LoadRepositoryManifest(root, options.ManifestPath, options.RepositoryID, options.PipelineClass)
	if err != nil {
		return GeneratedImpact{}, err
	}
	return discoverWithManifest(ctx, options, manifest)
}

// DiscoverWithManifest validates an already parsed, reviewable manifest
// against Gradle's configured model. The manifest is still only proposal state;
// callers must persist and review it before any measurement or activation.
func DiscoverWithManifest(ctx context.Context, options DiscoveryOptions, manifest LoadedManifest) (GeneratedImpact, error) {
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("resolve repository root: %w", err)
	}
	options.RepositoryRoot = filepath.Clean(root)
	return discoverWithManifest(ctx, options, manifest)
}

func discoverWithManifest(ctx context.Context, options DiscoveryOptions, manifest LoadedManifest) (GeneratedImpact, error) {
	raw, err := observeGradle(ctx, ObservationOptions{
		RepositoryRoot: options.RepositoryRoot,
		Entrypoints:    allManifestEntrypoints(manifest.Manifest),
		GradleCommand:  options.GradleCommand,
		GradleArgs:     options.GradleArgs,
	})
	if err != nil {
		return GeneratedImpact{}, err
	}
	return GenerateImpact(manifest, raw)
}

// ObserveGradle returns a strictly validated configured-model snapshot for a
// bounded entrypoint set. It performs no task work and grants no omission or
// activation authority.
func ObserveGradle(ctx context.Context, options ObservationOptions) (DiscoverySnapshot, error) {
	raw, err := observeGradle(ctx, options)
	if err != nil {
		return DiscoverySnapshot{}, err
	}
	snapshot, _, err := parseDiscoverySnapshotForEntrypoints(raw, options.Entrypoints, true)
	return snapshot, err
}

func observeGradle(ctx context.Context, options ObservationOptions) ([]byte, error) {
	root, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	root = filepath.Clean(root)
	temporary, err := os.MkdirTemp("", "buildopt-impact-discovery-")
	if err != nil {
		return nil, fmt.Errorf("create discovery directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	initPath := filepath.Join(temporary, "discovery.init.gradle")
	outputPath := filepath.Join(temporary, "snapshot.json")
	if err := os.WriteFile(initPath, discoveryInitScript, 0o600); err != nil {
		return nil, fmt.Errorf("write discovery init script: %w", err)
	}
	if err := validateEntrypoints(options.Entrypoints); err != nil {
		return nil, fmt.Errorf("invalid discovery entrypoints: %w", err)
	}
	entrypoints, err := json.Marshal(options.Entrypoints)
	if err != nil {
		return nil, fmt.Errorf("encode discovery entrypoints: %w", err)
	}
	gradleCommand := options.GradleCommand
	if gradleCommand == "" {
		gradleCommand = filepath.Join(root, "gradlew")
		if runtime.GOOS == "windows" {
			gradleCommand += ".bat"
		}
	} else if !filepath.IsAbs(gradleCommand) {
		gradleCommand = filepath.Join(root, gradleCommand)
	}
	arguments := discoveryGradleArguments(options.GradleArgs, initPath)
	discoveryEnvironment := replaceDiscoveryEnvironment(os.Environ(), map[string]string{
		"BUILDOPT_IMPACT_DISCOVERY_OUTPUT": outputPath,
		"BUILDOPT_IMPACT_ENTRYPOINTS":      string(entrypoints),
	})
	command := exec.CommandContext(ctx, gradleCommand, arguments...)
	command.Dir = root
	command.Env = discoveryEnvironment
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run Gradle impact discovery: %w\n%s", err, output)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read Gradle impact discovery: %w", err)
	}
	return raw, nil
}

func discoveryGradleArguments(ownerOptions []string, initPath string) []string {
	arguments := make([]string, 0, len(ownerOptions)+6)
	for _, option := range ownerOptions {
		if option == "--daemon" || option == "--no-daemon" ||
			option == "--configure-on-demand" || option == "--no-configure-on-demand" ||
			strings.HasPrefix(option, "--console=") {
			continue
		}
		arguments = append(arguments, option)
	}
	return append(arguments, "--no-daemon", "--no-configure-on-demand", "--console=plain", "--init-script", initPath, "buildoptImpactDiscovery")
}

// GenerateImpact strictly validates a Gradle snapshot and produces canonical
// graph and generated-manifest documents suitable for repository review.
func GenerateImpact(manifest LoadedManifest, raw []byte) (GeneratedImpact, error) {
	snapshot, discoveryDigest, err := parseDiscoverySnapshot(raw, manifest.Manifest)
	if err != nil {
		return GeneratedImpact{}, err
	}
	buildChecks, _ := manifestCheckIDs(manifest.Manifest)
	artifactIDs := sortedKeys(manifestArtifactIDs(manifest.Manifest))
	checkIDs := sortedKeys(buildChecks)
	graphValue := DeclaredGraph{
		SchemaVersion:     DeclaredGraphSchemaVersion,
		RepositoryID:      manifest.Manifest.RepositoryID,
		PipelineClass:     manifest.Manifest.PipelineClass,
		ManifestDigest:    manifest.Digest,
		AdapterVersion:    GradleDiscoveryAdapterVersion,
		Complete:          snapshot.Complete,
		GlobalChangePaths: append([]string{}, snapshot.IncludedBuildPaths...),
	}
	for _, project := range normalizeDependencyCycles(snapshot.Projects) {
		graphValue.Projects = append(graphValue.Projects, Project{
			Path:                 project.Path,
			SourcePaths:          cloneSlice(project.SourcePaths),
			OwnedSourcePaths:     cloneSlice(project.OwnedSourcePaths),
			DependsOn:            cloneSlice(project.DependsOn),
			UnknownRelationships: project.UnknownRelationships,
		})
	}
	for _, entrypoint := range snapshot.Entrypoints {
		graphValue.Entrypoints = append(graphValue.Entrypoints, DeclaredEntrypoint{
			Name:                 entrypoint.Name,
			Owner:                BuildOptimization,
			ReachesProjects:      cloneSlice(entrypoint.ReachesProjects),
			ArtifactIDs:          cloneSlice(artifactIDs),
			CheckIDs:             cloneSlice(checkIDs),
			ContainsTestTasks:    entrypoint.ContainsTestTasks,
			UnknownRelationships: entrypoint.UnknownRelationships,
		})
	}
	canonicalGraph, err := json.Marshal(graphValue)
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("encode generated impact graph: %w", err)
	}
	graph, err := ParseDeclaredGraph(canonicalGraph, manifest)
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("validate generated impact graph: %w", err)
	}
	generated := GeneratedManifest{
		SchemaVersion:   GeneratedManifestSchemaVersion,
		RepositoryID:    manifest.Manifest.RepositoryID,
		PipelineClass:   manifest.Manifest.PipelineClass,
		ManifestDigest:  manifest.Digest,
		GraphDigest:     graph.Digest,
		DiscoveryDigest: discoveryDigest,
		AdapterVersion:  GradleDiscoveryAdapterVersion,
		GradleVersion:   snapshot.GradleVersion,
		Complete:        snapshot.Complete,
		FallbackReasons: cloneSlice(snapshot.FallbackReasons),
	}
	graphJSON, err := renderReviewableJSON(graph.Graph)
	if err != nil {
		return GeneratedImpact{}, err
	}
	generatedJSON, err := renderReviewableJSON(generated)
	if err != nil {
		return GeneratedImpact{}, err
	}
	return GeneratedImpact{
		Manifest:       manifest,
		Graph:          graph,
		GraphJSON:      graphJSON,
		Generated:      generated,
		GeneratedJSON:  generatedJSON,
		Snapshot:       snapshot,
		SnapshotDigest: discoveryDigest,
	}, nil
}

// normalizeDependencyCycles preserves every declared dependency while making
// strongly connected projects conservative peers in the acyclic selection
// graph. Each peer receives the component's complete source boundary and its
// external dependencies, so a change to any member affects the whole component
// and all of its downstream consumers.
func normalizeDependencyCycles(projects []DiscoveredProject) []DiscoveredProject {
	byPath := make(map[string]DiscoveredProject, len(projects))
	for _, project := range projects {
		byPath[project.Path] = project
	}
	index := 0
	indices := make(map[string]int, len(projects))
	lowLinks := make(map[string]int, len(projects))
	onStack := make(map[string]bool, len(projects))
	stack := make([]string, 0, len(projects))
	components := make([][]string, 0, len(projects))
	var visit func(string)
	visit = func(projectPath string) {
		indices[projectPath] = index
		lowLinks[projectPath] = index
		index++
		stack = append(stack, projectPath)
		onStack[projectPath] = true
		for _, dependency := range byPath[projectPath].DependsOn {
			if _, seen := indices[dependency]; !seen {
				visit(dependency)
				lowLinks[projectPath] = min(lowLinks[projectPath], lowLinks[dependency])
			} else if onStack[dependency] {
				lowLinks[projectPath] = min(lowLinks[projectPath], indices[dependency])
			}
		}
		if lowLinks[projectPath] != indices[projectPath] {
			return
		}
		component := []string{}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == projectPath {
				break
			}
		}
		components = append(components, component)
	}
	for _, project := range projects {
		if _, seen := indices[project.Path]; !seen {
			visit(project.Path)
		}
	}

	normalized := make(map[string]DiscoveredProject, len(projects))
	for _, component := range components {
		members := make(map[string]bool, len(component))
		sources := map[string]bool{}
		externalDependencies := map[string]bool{}
		unknown := false
		for _, projectPath := range component {
			members[projectPath] = true
		}
		for _, projectPath := range component {
			project := byPath[projectPath]
			unknown = unknown || project.UnknownRelationships
			for _, source := range project.SourcePaths {
				sources[source] = true
			}
			for _, dependency := range project.DependsOn {
				if !members[dependency] {
					externalDependencies[dependency] = true
				}
			}
		}
		componentSources := sortedKeys(sources)
		componentDependencies := sortedKeys(externalDependencies)
		for _, projectPath := range component {
			project := byPath[projectPath]
			if len(component) > 1 {
				project.OwnedSourcePaths = cloneSlice(project.SourcePaths)
			}
			project.SourcePaths = cloneSlice(componentSources)
			project.DependsOn = cloneSlice(componentDependencies)
			project.UnknownRelationships = unknown
			normalized[projectPath] = project
		}
	}
	result := make([]DiscoveredProject, 0, len(projects))
	for _, project := range projects {
		result = append(result, normalized[project.Path])
	}
	return result
}

func parseDiscoverySnapshot(raw []byte, manifest Manifest) (DiscoverySnapshot, string, error) {
	return parseDiscoverySnapshotForEntrypoints(raw, allManifestEntrypoints(manifest), false)
}

func parseDiscoverySnapshotForEntrypoints(raw []byte, wantedEntrypoints []string, allowEmptyReach bool) (DiscoverySnapshot, string, error) {
	if len(raw) == 0 || len(raw) > maximumDiscoveryBytes {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var snapshot DiscoverySnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return DiscoverySnapshot{}, "", fmt.Errorf("decode Gradle discovery: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery has trailing content")
	}
	if snapshot.SchemaVersion != DiscoverySchemaVersion || snapshot.GradleVersion == "" || len(snapshot.Projects) == 0 || len(snapshot.Projects) > 4096 {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery identity or project collection is invalid")
	}
	if !uniqueStrings(snapshot.FallbackReasons) || (snapshot.Complete && len(snapshot.FallbackReasons) != 0) || (!snapshot.Complete && len(snapshot.FallbackReasons) == 0) {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery completeness is inconsistent")
	}
	sort.Strings(snapshot.IncludedBuildPaths)
	if len(snapshot.IncludedBuildPaths) > 256 || !uniqueSafeGlobs(snapshot.IncludedBuildPaths) {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery included-build paths are invalid")
	}
	if len(snapshot.Entrypoints) != len(wantedEntrypoints) {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery does not cover every manifest entrypoint")
	}
	projectSet := map[string]bool{}
	for index := range snapshot.Projects {
		project := &snapshot.Projects[index]
		sort.Strings(project.SourcePaths)
		sort.Strings(project.DependsOn)
		if !validGradleProject(project.Path) || projectSet[project.Path] || !uniqueSafeGlobs(project.SourcePaths) || len(project.SourcePaths) == 0 || !uniqueStrings(project.DependsOn) {
			return DiscoverySnapshot{}, "", errors.New("Gradle discovery contains an invalid project")
		}
		projectSet[project.Path] = true
	}
	entrypointSet := map[string]bool{}
	for index := range snapshot.Entrypoints {
		entrypoint := &snapshot.Entrypoints[index]
		sort.Strings(entrypoint.ReachesProjects)
		if !validGradleEntrypoint(entrypoint.Name) {
			return DiscoverySnapshot{}, "", fmt.Errorf("Gradle discovery contains invalid entrypoint name %q", entrypoint.Name)
		}
		if entrypointSet[entrypoint.Name] {
			return DiscoverySnapshot{}, "", fmt.Errorf("Gradle discovery repeats entrypoint %q", entrypoint.Name)
		}
		if (!allowEmptyReach && len(entrypoint.ReachesProjects) == 0) || !uniqueStrings(entrypoint.ReachesProjects) {
			return DiscoverySnapshot{}, "", fmt.Errorf("Gradle discovery entrypoint %q has an invalid project reach", entrypoint.Name)
		}
		entrypointSet[entrypoint.Name] = true
		for _, projectPath := range entrypoint.ReachesProjects {
			if !projectSet[projectPath] {
				return DiscoverySnapshot{}, "", errors.New("Gradle discovery entrypoint reaches an unknown project")
			}
		}
	}
	for _, name := range wantedEntrypoints {
		if !entrypointSet[name] {
			return DiscoverySnapshot{}, "", errors.New("Gradle discovery omits a manifest entrypoint")
		}
	}
	for _, project := range snapshot.Projects {
		for _, dependency := range project.DependsOn {
			if !projectSet[dependency] || dependency == project.Path {
				return DiscoverySnapshot{}, "", errors.New("Gradle discovery contains an invalid project dependency")
			}
		}
	}
	if len(snapshot.Tasks) > maximumDiscoveredTasks {
		return DiscoverySnapshot{}, "", errors.New("Gradle discovery task collection is invalid")
	}
	taskSet := map[string]bool{}
	for index := range snapshot.Tasks {
		task := &snapshot.Tasks[index]
		sort.Strings(task.DependsOn)
		if !validGradleEntrypoint(task.Path) || !projectSet[task.ProjectPath] ||
			taskSet[task.Path] || !uniqueStrings(task.DependsOn) {
			return DiscoverySnapshot{}, "", errors.New("Gradle discovery contains an invalid task")
		}
		taskSet[task.Path] = true
	}
	for _, task := range snapshot.Tasks {
		for _, dependency := range task.DependsOn {
			if !taskSet[dependency] || dependency == task.Path {
				return DiscoverySnapshot{}, "", errors.New("Gradle discovery contains an invalid task dependency")
			}
		}
	}
	sort.Slice(snapshot.Projects, func(left, right int) bool { return snapshot.Projects[left].Path < snapshot.Projects[right].Path })
	sort.Slice(snapshot.Tasks, func(left, right int) bool { return snapshot.Tasks[left].Path < snapshot.Tasks[right].Path })
	sort.Slice(snapshot.Entrypoints, func(left, right int) bool { return snapshot.Entrypoints[left].Name < snapshot.Entrypoints[right].Name })
	sort.Strings(snapshot.FallbackReasons)
	canonical, err := json.Marshal(snapshot)
	if err != nil {
		return DiscoverySnapshot{}, "", fmt.Errorf("canonicalize Gradle discovery: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return snapshot, "sha256:" + hex.EncodeToString(digest[:]), nil
}

func renderReviewableJSON(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render generated Build Impact state: %w", err)
	}
	return append(raw, '\n'), nil
}

func replaceDiscoveryEnvironment(environment []string, overrides map[string]string) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, value := range environment {
		key, _, found := strings.Cut(value, "=")
		if !found {
			continue
		}
		if _, replaced := overrides[key]; !replaced {
			result = append(result, value)
		}
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, key+"="+overrides[key])
	}
	return result
}
