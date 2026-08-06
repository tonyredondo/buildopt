package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/platformfs"
)

const impactHotStateSchemaVersion = "buildopt.build-impact/poc-hot-state/v1"

var impactRevisionPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

type impactHotStateOptions struct {
	RepositoryRoot     string
	RepositoryID       string
	PipelineClass      string
	RepositoryRevision string
	ManifestPath       string
	GraphPath          string
	GeneratedPath      string
	ChangesPath        string
	GradleOptions      []string
	StateDirectory     string
}

type impactHotStateBinding struct {
	SchemaVersion      string   `json:"schemaVersion"`
	RepositoryID       string   `json:"repositoryId"`
	PipelineClass      string   `json:"pipelineClass"`
	RepositoryRevision string   `json:"repositoryRevision"`
	ManifestSha256     string   `json:"manifestSha256"`
	GraphSha256        string   `json:"graphSha256"`
	GeneratedSha256    string   `json:"generatedManifestSha256"`
	ChangesSha256      string   `json:"changesSha256"`
	WrapperJarSha256   string   `json:"wrapperJarSha256"`
	WrapperPropsSha256 string   `json:"wrapperPropertiesSha256"`
	ExecutableSha256   string   `json:"executableSha256"`
	GradleOptions      []string `json:"gradleOptions"`
}

type impactHotStateRecord struct {
	Binding impactHotStateBinding        `json:"binding"`
	Plan    buildimpact.POCCandidatePlan `json:"plan"`
}

type impactHotState struct {
	enabled bool
	path    string
	binding impactHotStateBinding
}

func prepareImpactHotState(options impactHotStateOptions) (impactHotState, error) {
	if options.StateDirectory == "" && options.RepositoryRevision == "" {
		return impactHotState{}, nil
	}
	if options.StateDirectory == "" || !filepath.IsAbs(options.StateDirectory) ||
		!impactRevisionPattern.MatchString(options.RepositoryRevision) {
		return impactHotState{}, errors.New("Build Impact hot state requires an absolute directory and immutable 40- or 64-character revision")
	}
	if err := os.MkdirAll(options.StateDirectory, 0o700); err != nil {
		return impactHotState{}, fmt.Errorf("create Build Impact hot-state directory: %w", err)
	}
	resolved := filepath.Clean(options.StateDirectory)
	if err := platformfs.ValidateNoLinks(resolved); err != nil {
		return impactHotState{}, errors.New("Build Impact hot-state directory must contain no symlink components")
	}
	stateInfo, err := os.Stat(resolved)
	if err != nil || !impactHotStateDirectoryPrivate(stateInfo) {
		return impactHotState{}, errors.New("Build Impact hot-state directory must be private")
	}
	executable, err := os.Executable()
	if err != nil {
		return impactHotState{}, fmt.Errorf("resolve BuildOpt executable: %w", err)
	}
	binding := impactHotStateBinding{
		SchemaVersion:      impactHotStateSchemaVersion,
		RepositoryID:       options.RepositoryID,
		PipelineClass:      options.PipelineClass,
		RepositoryRevision: options.RepositoryRevision,
		GradleOptions:      append([]string(nil), options.GradleOptions...),
	}
	files := []struct {
		path   string
		target *string
	}{
		{options.ManifestPath, &binding.ManifestSha256},
		{options.GraphPath, &binding.GraphSha256},
		{options.GeneratedPath, &binding.GeneratedSha256},
		{options.ChangesPath, &binding.ChangesSha256},
		{"gradle/wrapper/gradle-wrapper.jar", &binding.WrapperJarSha256},
		{"gradle/wrapper/gradle-wrapper.properties", &binding.WrapperPropsSha256},
	}
	for _, file := range files {
		digest, err := impactRegularFileDigest(options.RepositoryRoot, file.path)
		if err != nil {
			return impactHotState{}, err
		}
		*file.target = digest
	}
	binding.ExecutableSha256, err = impactAbsoluteFileDigest(executable)
	if err != nil {
		return impactHotState{}, err
	}
	canonical, err := json.Marshal(binding)
	if err != nil {
		return impactHotState{}, fmt.Errorf("encode Build Impact hot-state binding: %w", err)
	}
	key := sha256.Sum256(canonical)
	return impactHotState{
		enabled: true,
		path:    filepath.Join(resolved, hex.EncodeToString(key[:])+".json"),
		binding: binding,
	}, nil
}

func (state impactHotState) load() (buildimpact.POCCandidatePlan, bool, error) {
	if !state.enabled {
		return buildimpact.POCCandidatePlan{}, false, nil
	}
	raw, err := os.ReadFile(state.path)
	if errors.Is(err, os.ErrNotExist) {
		return buildimpact.POCCandidatePlan{}, false, nil
	}
	if err != nil {
		return buildimpact.POCCandidatePlan{}, false, fmt.Errorf("read Build Impact hot state: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var record impactHotStateRecord
	if err := decoder.Decode(&record); err != nil {
		return buildimpact.POCCandidatePlan{}, false, nil
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) ||
		!reflect.DeepEqual(record.Binding, state.binding) || record.Plan.SchemaVersion != buildimpact.POCCandidatePlanSchemaVersion ||
		len(record.Plan.Entrypoints) == 0 {
		return buildimpact.POCCandidatePlan{}, false, nil
	}
	return record.Plan, true, nil
}

func (state impactHotState) store(plan buildimpact.POCCandidatePlan) error {
	if !state.enabled {
		return nil
	}
	raw, err := json.MarshalIndent(impactHotStateRecord{Binding: state.binding, Plan: plan}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Build Impact hot state: %w", err)
	}
	raw = append(raw, '\n')
	temporary, err := os.CreateTemp(filepath.Dir(state.path), ".buildopt-impact-hot-*.tmp")
	if err != nil {
		return fmt.Errorf("create Build Impact hot state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, state.path); err != nil {
		return fmt.Errorf("publish Build Impact hot state: %w", err)
	}
	return nil
}

func impactRegularFileDigest(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.Clean(relative) != relative {
		return "", errors.New("Build Impact hot-state input path is unsafe")
	}
	return impactAbsoluteFileDigest(filepath.Join(root, relative))
}

func impactAbsoluteFileDigest(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("Build Impact hot-state input is not a regular file: %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read Build Impact hot-state input: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}
