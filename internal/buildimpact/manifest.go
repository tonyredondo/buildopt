// Package buildimpact owns conservative Build Impact Analysis decisions.
package buildimpact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	ManifestSchemaVersion = "buildopt.build-impact/manifest/v1"
	RepositoryOwnership   = "REPOSITORY_COMMITTED"
	FullGraphPolicy       = "FULL_GRAPH"
	BuildOptimization     = "BUILD_OPTIMIZATION"
	TestOptimization      = "TEST_OPTIMIZATION"
	maximumManifestBytes  = 256 << 10
)

var (
	idPattern          = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	repositoryPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}/[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)
	pipelinePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	taskSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
)

type Manifest struct {
	SchemaVersion       string          `json:"schemaVersion"`
	ManifestVersion     uint64          `json:"manifestVersion"`
	RepositoryID        string          `json:"repositoryId"`
	PipelineClass       string          `json:"pipelineClass"`
	Ownership           string          `json:"ownership"`
	OriginalEntrypoints []string        `json:"originalEntrypoints"`
	AllowedAlternatives []EntrypointSet `json:"allowedAlternatives"`
	RequiredArtifacts   []Artifact      `json:"requiredArtifacts"`
	RequiredChecks      []Check         `json:"requiredChecks"`
	GlobalChangePaths   []string        `json:"globalChangePaths"`
	UnknownChangePolicy string          `json:"unknownChangePolicy"`
}

type EntrypointSet struct {
	ID          string   `json:"id"`
	Entrypoints []string `json:"entrypoints"`
}

type Artifact struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Owner string `json:"owner"`
}

type Check struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
}

type LoadedManifest struct {
	Manifest Manifest
	Digest   string
}

// LoadRepositoryManifest reads one regular, repository-contained manifest and
// binds it to the repository and pipeline selected by the caller.
func LoadRepositoryManifest(repositoryRoot, manifestPath, repositoryID, pipelineClass string) (LoadedManifest, error) {
	raw, err := readRepositoryFile(repositoryRoot, manifestPath)
	if err != nil {
		return LoadedManifest{}, err
	}
	return ParseManifest(raw, repositoryID, pipelineClass)
}

// ParseManifest strictly decodes, validates, and canonically digests a manifest.
func ParseManifest(raw []byte, repositoryID, pipelineClass string) (LoadedManifest, error) {
	if len(raw) == 0 || len(raw) > maximumManifestBytes {
		return LoadedManifest{}, errors.New("build-impact manifest size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return LoadedManifest{}, fmt.Errorf("decode build-impact manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return LoadedManifest{}, errors.New("build-impact manifest has trailing content")
	}
	if err := validateManifest(manifest, repositoryID, pipelineClass); err != nil {
		return LoadedManifest{}, err
	}
	canonical, err := json.Marshal(manifest)
	if err != nil {
		return LoadedManifest{}, fmt.Errorf("canonicalize build-impact manifest: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return LoadedManifest{Manifest: defensiveManifest(manifest), Digest: "sha256:" + hex.EncodeToString(digest[:])}, nil
}

func validateManifest(manifest Manifest, repositoryID, pipelineClass string) error {
	if manifest.SchemaVersion != ManifestSchemaVersion || manifest.ManifestVersion == 0 {
		return errors.New("unsupported build-impact manifest identity")
	}
	if !repositoryPattern.MatchString(repositoryID) || manifest.RepositoryID != repositoryID {
		return errors.New("build-impact manifest repository binding does not match")
	}
	if !pipelinePattern.MatchString(pipelineClass) || manifest.PipelineClass != pipelineClass {
		return errors.New("build-impact manifest pipeline binding does not match")
	}
	if manifest.Ownership != RepositoryOwnership {
		return errors.New("build-impact manifest must be repository committed")
	}
	if manifest.UnknownChangePolicy != FullGraphPolicy {
		return errors.New("unknown changes must use FULL_GRAPH")
	}
	if err := validateEntrypoints(manifest.OriginalEntrypoints); err != nil {
		return fmt.Errorf("invalid original entrypoints: %w", err)
	}
	if len(manifest.AllowedAlternatives) == 0 || len(manifest.AllowedAlternatives) > 64 {
		return errors.New("build-impact manifest requires bounded allowed alternatives")
	}
	originalKey := setKey(manifest.OriginalEntrypoints)
	alternativeIDs := map[string]bool{}
	alternativeSets := map[string]bool{}
	for _, alternative := range manifest.AllowedAlternatives {
		if !idPattern.MatchString(alternative.ID) || alternativeIDs[alternative.ID] {
			return errors.New("allowed alternative IDs must be unique canonical IDs")
		}
		alternativeIDs[alternative.ID] = true
		if err := validateEntrypoints(alternative.Entrypoints); err != nil {
			return fmt.Errorf("invalid allowed alternative %s: %w", alternative.ID, err)
		}
		key := setKey(alternative.Entrypoints)
		if key == originalKey || alternativeSets[key] {
			return errors.New("allowed alternative entrypoint sets must be unique and differ from the original")
		}
		alternativeSets[key] = true
	}
	if len(manifest.RequiredArtifacts) == 0 || len(manifest.RequiredArtifacts) > 256 {
		return errors.New("build-impact manifest requires bounded artifacts")
	}
	ownedIDs := map[string]bool{}
	artifactPaths := map[string]bool{}
	for _, artifact := range manifest.RequiredArtifacts {
		if err := validateOwnedID(artifact.ID, ownedIDs); err != nil {
			return fmt.Errorf("invalid required artifact: %w", err)
		}
		if artifact.Owner != BuildOptimization || !validRepositoryGlob(artifact.Path) || artifactPaths[artifact.Path] {
			return errors.New("required artifacts need BUILD_OPTIMIZATION ownership and safe repository-relative paths")
		}
		artifactPaths[artifact.Path] = true
	}
	if manifest.RequiredChecks == nil || len(manifest.RequiredChecks) > 256 {
		return errors.New("build-impact manifest requires an explicit bounded check list")
	}
	for _, check := range manifest.RequiredChecks {
		if err := validateOwnedID(check.ID, ownedIDs); err != nil {
			return fmt.Errorf("invalid required check: %w", err)
		}
		if check.Owner != BuildOptimization && check.Owner != TestOptimization {
			return errors.New("required checks need an explicit product owner")
		}
	}
	if len(manifest.GlobalChangePaths) == 0 || len(manifest.GlobalChangePaths) > 256 {
		return errors.New("build-impact manifest requires bounded global change paths")
	}
	seenPaths := map[string]bool{}
	for _, globalPath := range manifest.GlobalChangePaths {
		if !validRepositoryGlob(globalPath) || seenPaths[globalPath] {
			return errors.New("global change paths must be unique safe repository-relative globs")
		}
		seenPaths[globalPath] = true
	}
	return nil
}

func validateEntrypoints(entrypoints []string) error {
	if len(entrypoints) == 0 || len(entrypoints) > 64 {
		return errors.New("entrypoint set must contain between 1 and 64 tasks")
	}
	seen := map[string]bool{}
	for _, entrypoint := range entrypoints {
		if !validGradleEntrypoint(entrypoint) || seen[entrypoint] {
			return errors.New("entrypoints must be unique canonical Gradle task names")
		}
		seen[entrypoint] = true
	}
	return nil
}

func validGradleEntrypoint(entrypoint string) bool {
	if entrypoint == "" || strings.TrimSpace(entrypoint) != entrypoint || strings.HasPrefix(entrypoint, "-") {
		return false
	}
	segments := strings.Split(strings.TrimPrefix(entrypoint, ":"), ":")
	if strings.HasPrefix(entrypoint, "::") || len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if !taskSegmentPattern.MatchString(segment) || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func validRepositoryGlob(candidate string) bool {
	if candidate == "" || strings.TrimSpace(candidate) != candidate || strings.ContainsAny(candidate, "\\\x00") || strings.HasPrefix(candidate, "/") {
		return false
	}
	for _, character := range candidate {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	if path.Clean(candidate) != candidate || candidate == "." || strings.HasPrefix(candidate, "../") {
		return false
	}
	for _, segment := range strings.Split(candidate, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	_, err := path.Match(candidate, candidate)
	return err == nil
}

func validateOwnedID(id string, seen map[string]bool) error {
	if !idPattern.MatchString(id) || seen[id] {
		return errors.New("IDs must be unique canonical IDs")
	}
	seen[id] = true
	return nil
}

func setKey(values []string) string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return strings.Join(copyValues, "\x00")
}

func readRepositoryFile(repositoryRoot, manifestPath string) ([]byte, error) {
	if !filepath.IsAbs(repositoryRoot) || filepath.Clean(repositoryRoot) != repositoryRoot {
		return nil, errors.New("repository root must be absolute and clean")
	}
	resolvedRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil || resolvedRoot != repositoryRoot {
		return nil, errors.New("repository root must exist and contain no symlink components")
	}
	if manifestPath == "" || filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || manifestPath == "." || strings.HasPrefix(manifestPath, ".."+string(filepath.Separator)) {
		return nil, errors.New("manifest path must be clean and repository relative")
	}
	current := repositoryRoot
	for _, segment := range strings.Split(manifestPath, string(filepath.Separator)) {
		if segment == "" || segment == "." || segment == ".." {
			return nil, errors.New("manifest path contains an unsafe segment")
		}
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("inspect build-impact manifest path: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("build-impact manifest path must not contain symlinks")
		}
	}
	info, err := os.Stat(current)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumManifestBytes {
		return nil, errors.New("build-impact manifest must be a bounded regular file")
	}
	file, err := os.Open(current)
	if err != nil {
		return nil, fmt.Errorf("open build-impact manifest: %w", err)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maximumManifestBytes+1))
	if err != nil || len(raw) > maximumManifestBytes {
		return nil, errors.New("read bounded build-impact manifest")
	}
	return raw, nil
}

func defensiveManifest(manifest Manifest) Manifest {
	manifest.OriginalEntrypoints = cloneSlice(manifest.OriginalEntrypoints)
	manifest.AllowedAlternatives = cloneSlice(manifest.AllowedAlternatives)
	for index := range manifest.AllowedAlternatives {
		manifest.AllowedAlternatives[index].Entrypoints = cloneSlice(manifest.AllowedAlternatives[index].Entrypoints)
	}
	manifest.RequiredArtifacts = cloneSlice(manifest.RequiredArtifacts)
	manifest.RequiredChecks = cloneSlice(manifest.RequiredChecks)
	manifest.GlobalChangePaths = cloneSlice(manifest.GlobalChangePaths)
	return manifest
}

func cloneSlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	result := make([]T, len(values))
	copy(result, values)
	return result
}
