package launcher

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

const (
	maximumImpactChangesBytes = 1 << 20
	maximumImpactChangedPaths = 10_000
	impactUsage               = "usage: buildopt impact --repository-id OWNER/REPO --changes-file PATH [--pipeline-class CLASS] [--manifest PATH] [--graph PATH] [--generated-manifest PATH] [--gradle-option VALUE ...]\n"
)

type repeatedStringFlag []string

func (values *repeatedStringFlag) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedStringFlag) Set(value string) error {
	if !validImpactGradleOption(value) {
		return errors.New("option is not allowed for a Build Impact POC candidate")
	}
	*values = append(*values, value)
	return nil
}

func validImpactGradleOption(value string) bool {
	switch value {
	case "--offline", "--daemon", "--no-daemon", "--parallel", "--no-parallel",
		"--no-scan", "--stacktrace", "--full-stacktrace",
		"--info", "--debug", "--warn", "--build-cache", "--no-build-cache",
		"--configuration-cache", "--no-configuration-cache":
		return true
	}
	if strings.HasPrefix(value, "--console=") {
		switch strings.TrimPrefix(value, "--console=") {
		case "plain", "auto", "rich", "verbose":
			return true
		}
	}
	if strings.HasPrefix(value, "--max-workers=") {
		workers, err := strconv.Atoi(strings.TrimPrefix(value, "--max-workers="))
		return err == nil && workers > 0
	}
	return false
}

type impactInvocation struct {
	gradleArgs []string
	plan       buildimpact.POCCandidatePlan
}

func prepareImpactInvocation(args []string, bypass bool) (impactInvocation, error) {
	flags := flag.NewFlagSet("buildopt impact", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repositoryID := flags.String("repository-id", "", "bound owner/repository identity")
	pipelineClass := flags.String("pipeline-class", "pull-request", "bound pipeline class")
	changesFile := flags.String("changes-file", "", "repository-relative newline-delimited changed paths")
	manifestPath := flags.String("manifest", "buildopt-impact-manifest.json", "repository-relative customer manifest")
	graphPath := flags.String("graph", "buildopt-impact-graph.generated.json", "repository-relative generated graph")
	generatedPath := flags.String("generated-manifest", "buildopt-impact.generated.json", "repository-relative generated manifest")
	var gradleOptions repeatedStringFlag
	flags.Var(&gradleOptions, "gradle-option", "Gradle option passed before the selected entrypoints; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *repositoryID == "" || *changesFile == "" {
		return impactInvocation{}, errors.New("invalid Build Impact POC arguments")
	}
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return impactInvocation{}, fmt.Errorf("resolve repository root: %w", err)
	}
	repositoryRoot, err = filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return impactInvocation{}, fmt.Errorf("canonicalize repository root: %w", err)
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return impactInvocation{}, fmt.Errorf("make repository root absolute: %w", err)
	}
	changedPaths, err := readImpactChangedPaths(repositoryRoot, *changesFile)
	if err != nil {
		return impactInvocation{}, err
	}
	plan, err := buildimpact.PlanPOCCandidate(buildimpact.POCCandidateOptions{
		RepositoryRoot:        repositoryRoot,
		ManifestPath:          *manifestPath,
		GraphPath:             *graphPath,
		GeneratedManifestPath: *generatedPath,
		RepositoryID:          *repositoryID,
		PipelineClass:         *pipelineClass,
		ChangedPaths:          changedPaths,
		LocalBypass:           bypass,
	})
	if err != nil {
		return impactInvocation{}, err
	}
	gradleArgs := append([]string(nil), gradleOptions...)
	gradleArgs = append(gradleArgs, plan.Entrypoints...)
	return impactInvocation{gradleArgs: gradleArgs, plan: plan}, nil
}

func readImpactChangedPaths(repositoryRoot, relativePath string) ([]string, error) {
	if relativePath == "" || filepath.IsAbs(relativePath) || filepath.Clean(relativePath) != relativePath || relativePath == "." || relativePath == ".." {
		return nil, errors.New("Build Impact changes file must be clean and repository relative")
	}
	path := filepath.Join(repositoryRoot, relativePath)
	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve Build Impact repository root: %w", err)
	}
	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve Build Impact changes file: %w", err)
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return nil, errors.New("Build Impact changes file escapes the repository")
	}
	file, err := os.Open(canonicalPath)
	if err != nil {
		return nil, fmt.Errorf("open Build Impact changes file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect Build Impact changes file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maximumImpactChangesBytes {
		return nil, errors.New("Build Impact changes file must be a bounded regular file")
	}
	paths := make([]string, 0)
	seen := map[string]bool{}
	scanner := bufio.NewScanner(io.LimitReader(file, maximumImpactChangesBytes+1))
	for scanner.Scan() {
		candidate := strings.TrimSuffix(scanner.Text(), "\r")
		if candidate == "" || seen[candidate] {
			return nil, errors.New("Build Impact changes file requires unique non-empty paths")
		}
		seen[candidate] = true
		paths = append(paths, candidate)
		if len(paths) > maximumImpactChangedPaths {
			return nil, errors.New("Build Impact changes file contains too many paths")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read Build Impact changes file: %w", err)
	}
	return paths, nil
}
