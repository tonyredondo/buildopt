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
)

//go:embed discovery.init.gradle
var discoveryInitScript []byte

type DiscoverySnapshot struct {
	SchemaVersion   string                 `json:"schemaVersion"`
	GradleVersion   string                 `json:"gradleVersion"`
	Complete        bool                   `json:"complete"`
	FallbackReasons []string               `json:"fallbackReasons"`
	Projects        []DiscoveredProject    `json:"projects"`
	Entrypoints     []DiscoveredEntrypoint `json:"entrypoints"`
}

type DiscoveredProject struct {
	Path                 string   `json:"path"`
	SourcePaths          []string `json:"sourcePaths"`
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
	temporary, err := os.MkdirTemp("", "buildopt-impact-discovery-")
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("create discovery directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	initPath := filepath.Join(temporary, "discovery.init.gradle")
	outputPath := filepath.Join(temporary, "snapshot.json")
	if err := os.WriteFile(initPath, discoveryInitScript, 0o600); err != nil {
		return GeneratedImpact{}, fmt.Errorf("write discovery init script: %w", err)
	}
	entrypoints, err := json.Marshal(allManifestEntrypoints(manifest.Manifest))
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("encode discovery entrypoints: %w", err)
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
	arguments := append([]string{}, options.GradleArgs...)
	arguments = append(arguments, "--no-daemon", "--console=plain", "--init-script", initPath, "buildoptImpactDiscovery")
	command := exec.CommandContext(ctx, gradleCommand, arguments...)
	command.Dir = root
	command.Env = replaceDiscoveryEnvironment(os.Environ(), map[string]string{
		"BUILDOPT_IMPACT_DISCOVERY_OUTPUT": outputPath,
		"BUILDOPT_IMPACT_ENTRYPOINTS":      string(entrypoints),
	})
	output, err := command.CombinedOutput()
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("run Gradle impact discovery: %w\n%s", err, output)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return GeneratedImpact{}, fmt.Errorf("read Gradle impact discovery: %w", err)
	}
	return GenerateImpact(manifest, raw)
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
		GlobalChangePaths: []string{},
	}
	for _, project := range snapshot.Projects {
		graphValue.Projects = append(graphValue.Projects, Project{
			Path:                 project.Path,
			SourcePaths:          cloneSlice(project.SourcePaths),
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

func parseDiscoverySnapshot(raw []byte, manifest Manifest) (DiscoverySnapshot, string, error) {
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
	wantedEntrypoints := allManifestEntrypoints(manifest)
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
		if !validGradleEntrypoint(entrypoint.Name) || entrypointSet[entrypoint.Name] || len(entrypoint.ReachesProjects) == 0 || !uniqueStrings(entrypoint.ReachesProjects) {
			return DiscoverySnapshot{}, "", errors.New("Gradle discovery contains an invalid entrypoint")
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
	sort.Slice(snapshot.Projects, func(left, right int) bool { return snapshot.Projects[left].Path < snapshot.Projects[right].Path })
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
