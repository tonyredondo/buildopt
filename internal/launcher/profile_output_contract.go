package launcher

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	profileOutputsUsage        = "usage: buildopt profile outputs --repository-id OWNER/REPO --pipeline-class CLASS --entrypoint TASK [--entrypoint TASK ...] [--required-output GLOB ...] [--gradle-command PATH] [--gradle-option VALUE ...] [--output PATH] [--timeout DURATION]\n"
	outputContractSchema       = "buildopt.poc/output-contract/v1"
	outputSnapshotSchema       = "buildopt.poc/output-contract-snapshot/v1"
	maximumOutputSnapshotBytes = 16 << 20
	maximumOutputContractFiles = 250000
)

//go:embed profile_output_contract.init.gradle
var profileOutputContractInit []byte

var (
	outputContractRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}/[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)
	outputContractPipelinePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
)

type outputContractConfig struct {
	repositoryID, pipelineClass, repositoryRevision string
	entrypoints, requiredOutputs, gradleOptions     []string
	gradleCommand                                   string
	timeout                                         time.Duration
}

type outputContractReport struct {
	SchemaVersion        string                     `json:"schemaVersion"`
	Decision             string                     `json:"decision"`
	Reason               string                     `json:"reason"`
	RepositoryID         string                     `json:"repositoryId"`
	PipelineClass        string                     `json:"pipelineClass"`
	RepositoryRevision   string                     `json:"repositoryRevision"`
	GradleVersion        string                     `json:"gradleVersion"`
	OriginalEntrypoints  []string                   `json:"originalEntrypoints"`
	DeclaredOutputs      []string                   `json:"declaredOutputs"`
	CandidateOutputs     []outputContractCandidate  `json:"candidateOutputs"`
	Validations          []outputContractValidation `json:"validations"`
	ReviewRequired       bool                       `json:"reviewRequired"`
	ActivationAutomatic  bool                       `json:"activationAutomatic"`
	ProductionAuthorized bool                       `json:"productionAuthorized"`
	TestOptimization     string                     `json:"testOptimization"`
}

type outputContractCandidate struct {
	Pattern       string   `json:"pattern"`
	Path          string   `json:"path"`
	Kind          string   `json:"kind"`
	FileCount     int      `json:"fileCount"`
	OwnerProjects []string `json:"ownerProjects"`
	ProducerTasks []string `json:"producerTasks"`
}

type outputContractValidation struct {
	Pattern       string   `json:"pattern"`
	Status        string   `json:"status"`
	MatchedFiles  int      `json:"matchedFiles"`
	OwnerProjects []string `json:"ownerProjects"`
	ProducerTasks []string `json:"producerTasks"`
}

type outputContractSnapshot struct {
	SchemaVersion string               `json:"schemaVersion"`
	GradleVersion string               `json:"gradleVersion"`
	Entrypoints   []string             `json:"entrypoints"`
	Tasks         []outputContractTask `json:"tasks"`
}

type outputContractTask struct {
	TaskPath         string               `json:"taskPath"`
	ProjectPath      string               `json:"projectPath"`
	ProjectDirectory string               `json:"projectDirectory"`
	Outputs          []outputContractRoot `json:"outputs"`
}

type outputContractRoot struct {
	Path             string `json:"path"`
	Kind             string `json:"kind"`
	Exists           bool   `json:"exists"`
	Symlink          bool   `json:"symlink"`
	InsideRepository bool   `json:"insideRepository"`
	FileCount        int    `json:"fileCount"`
}

type outputRootOwnership struct {
	path, kind string
	files      []string
	projects   map[string]bool
	tasks      map[string]bool
}

func runProfileOutputs(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileOutputsUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile outputs", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repositoryID := flags.String("repository-id", "", "owner/repository identity")
	pipelineClass := flags.String("pipeline-class", "", "pipeline class")
	gradleCommand := flags.String("gradle-command", "", "repository-relative or absolute Gradle command")
	output := flags.String("output", "buildopt-output-contract.json", "reviewable output-contract artifact")
	timeout := flags.Duration("timeout", 30*time.Minute, "owner workflow timeout")
	var entrypoints, requiredOutputs proposalStringFlag
	var gradleOptions repeatedStringFlag
	flags.Var(&entrypoints, "entrypoint", "original Gradle task selector; repeat for multiple tasks")
	flags.Var(&requiredOutputs, "required-output", "repository-owned output glob to validate")
	flags.Var(&gradleOptions, "gradle-option", "owner workflow Gradle option; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *repositoryID == "" || *pipelineClass == "" || len(entrypoints) == 0 || *timeout <= 0 {
		_, _ = io.WriteString(stderr, profileOutputsUsage)
		return exitUsage
	}
	root, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: output-contract preflight unavailable: %v\n", err)
		return exitConfiguration
	}
	revision, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil || !validMeasurementRevision(strings.TrimSpace(revision)) {
		_, _ = fmt.Fprintln(stderr, "buildopt: output-contract preflight unavailable: repository HEAD must be an immutable Git revision")
		return exitConfiguration
	}
	report, err := prepareOutputContract(context.Background(), root, outputContractConfig{
		repositoryID: *repositoryID, pipelineClass: *pipelineClass,
		repositoryRevision: strings.TrimSpace(revision),
		entrypoints:        append([]string(nil), entrypoints...),
		requiredOutputs:    append([]string(nil), requiredOutputs...),
		gradleCommand:      *gradleCommand, gradleOptions: append([]string(nil), gradleOptions...),
		timeout: *timeout,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: output-contract preflight unavailable: %v\n", err)
		return exitConfiguration
	}
	raw, err := renderOutputContract(report)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: output-contract preflight unavailable: %v\n", err)
		return exitConfiguration
	}
	if err := writeRepositoryDocument(root, *output, raw, 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: output-contract preflight unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write output-contract preflight: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func prepareOutputContract(ctx context.Context, repositoryRoot string, config outputContractConfig) (outputContractReport, error) {
	if !outputContractRepositoryPattern.MatchString(config.repositoryID) || !outputContractPipelinePattern.MatchString(config.pipelineClass) {
		return outputContractReport{}, errors.New("output-contract repository and pipeline identities are invalid")
	}
	if len(config.entrypoints) == 0 || len(config.entrypoints) > 256 || !uniqueMeasurementStrings(config.entrypoints) {
		return outputContractReport{}, errors.New("output-contract entrypoints must be unique and bounded")
	}
	if _, err := proposalTerminalSelectors(config.entrypoints); err != nil {
		return outputContractReport{}, err
	}
	if len(config.requiredOutputs) > 256 || !uniqueMeasurementStrings(config.requiredOutputs) {
		return outputContractReport{}, errors.New("declared outputs must be unique and bounded")
	}
	if len(config.gradleOptions) > 32 || !uniqueMeasurementStrings(config.gradleOptions) {
		return outputContractReport{}, errors.New("Gradle workflow options must be unique and bounded")
	}
	for _, pattern := range config.requiredOutputs {
		if !validOutputContractPattern(pattern) {
			return outputContractReport{}, fmt.Errorf("declared output %q is not a safe repository-relative glob", pattern)
		}
	}
	if !validMeasurementRevision(config.repositoryRevision) {
		return outputContractReport{}, errors.New("output-contract repository revision must be an immutable Git revision")
	}
	if dirty, err := gitOutput(repositoryRoot, "status", "--porcelain", "--untracked-files=no"); err != nil || strings.TrimSpace(dirty) != "" {
		return outputContractReport{}, errors.New("tracked repository state must be clean before output-contract preflight")
	}

	snapshot, err := observeOutputContract(ctx, repositoryRoot, config)
	if err != nil {
		return outputContractReport{}, err
	}
	if head, headErr := gitOutput(repositoryRoot, "rev-parse", "HEAD"); headErr != nil || strings.TrimSpace(head) != config.repositoryRevision {
		return outputContractReport{}, errors.New("owner workflow changed the repository revision during output-contract preflight")
	}
	if dirty, dirtyErr := gitOutput(repositoryRoot, "status", "--porcelain", "--untracked-files=no"); dirtyErr != nil || strings.TrimSpace(dirty) != "" {
		return outputContractReport{}, errors.New("owner workflow modified tracked files during output-contract preflight")
	}

	owners, candidates, err := collectOutputOwnership(repositoryRoot, snapshot)
	if err != nil {
		return outputContractReport{}, err
	}
	sort.Strings(config.entrypoints)
	sort.Strings(config.requiredOutputs)
	report := outputContractReport{
		SchemaVersion: outputContractSchema, Decision: "NATIVE_FULL_GRAPH",
		Reason: "OUTPUTS_MISSING", RepositoryID: config.repositoryID,
		PipelineClass: config.pipelineClass, RepositoryRevision: config.repositoryRevision,
		GradleVersion:       snapshot.GradleVersion,
		OriginalEntrypoints: append([]string(nil), config.entrypoints...),
		DeclaredOutputs:     append([]string{}, config.requiredOutputs...),
		CandidateOutputs:    candidates,
		Validations:         []outputContractValidation{},
		ReviewRequired:      true,
		ActivationAutomatic: false, ProductionAuthorized: false,
		TestOptimization: "OUT_OF_SCOPE",
	}
	if len(candidates) == 0 {
		return report, nil
	}
	if len(config.requiredOutputs) == 0 {
		report.Decision = "REVIEW_REQUIRED_OUTPUTS"
		report.Reason = "OUTPUTS_DISCOVERED"
		return report, nil
	}
	report.Validations = validateDeclaredOutputs(config.requiredOutputs, owners)
	for _, validation := range report.Validations {
		switch validation.Status {
		case "EMPTY":
			report.Reason = "REQUIRED_OUTPUTS_EMPTY"
			return report, nil
		case "AMBIGUOUS":
			report.Reason = "OUTPUT_OWNERSHIP_AMBIGUOUS"
			return report, nil
		}
	}
	report.Decision = "VALIDATED_REQUIRED_OUTPUTS"
	report.Reason = "DECLARED_OUTPUTS_MATCH_EXECUTED_WORKFLOW"
	return report, nil
}

func observeOutputContract(ctx context.Context, repositoryRoot string, config outputContractConfig) (outputContractSnapshot, error) {
	temporary, err := os.MkdirTemp("", "buildopt-output-contract-")
	if err != nil {
		return outputContractSnapshot{}, fmt.Errorf("create output-contract directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	initPath := filepath.Join(temporary, "output-contract.init.gradle")
	snapshotPath := filepath.Join(temporary, "snapshot.json")
	if err := os.WriteFile(initPath, profileOutputContractInit, 0o600); err != nil {
		return outputContractSnapshot{}, fmt.Errorf("write output-contract init script: %w", err)
	}
	entrypoints, err := json.Marshal(config.entrypoints)
	if err != nil {
		return outputContractSnapshot{}, fmt.Errorf("encode output-contract entrypoints: %w", err)
	}
	gradleCommand := config.gradleCommand
	if gradleCommand == "" {
		gradleCommand = filepath.Join(repositoryRoot, "gradlew")
		if runtime.GOOS == "windows" {
			gradleCommand += ".bat"
		}
	} else if !filepath.IsAbs(gradleCommand) {
		gradleCommand = filepath.Join(repositoryRoot, gradleCommand)
	}
	arguments := outputContractGradleArguments(config.gradleOptions, initPath)
	workflowContext, cancel := context.WithTimeout(ctx, config.timeout)
	defer cancel()
	command := exec.CommandContext(workflowContext, gradleCommand, arguments...)
	command.Dir = repositoryRoot
	command.Env = replaceOutputContractEnvironment(os.Environ(), map[string]string{
		"BUILDOPT_OUTPUT_CONTRACT_SNAPSHOT":    snapshotPath,
		"BUILDOPT_OUTPUT_CONTRACT_ENTRYPOINTS": string(entrypoints),
	})
	output, err := command.CombinedOutput()
	if err != nil {
		return outputContractSnapshot{}, fmt.Errorf("run owner Gradle workflow for output-contract preflight: %w\n%s", err, output)
	}
	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		return outputContractSnapshot{}, fmt.Errorf("read output-contract snapshot: %w", err)
	}
	return parseOutputContractSnapshot(raw, config.entrypoints)
}

func outputContractGradleArguments(ownerOptions []string, initPath string) []string {
	arguments := make([]string, 0, len(ownerOptions)+4)
	for _, option := range ownerOptions {
		if option == "--daemon" || option == "--no-daemon" ||
			option == "--configure-on-demand" || option == "--no-configure-on-demand" ||
			strings.HasPrefix(option, "--console=") {
			continue
		}
		arguments = append(arguments, option)
	}
	return append(arguments, "--no-daemon", "--no-configure-on-demand", "--console=plain", "--init-script", initPath, "buildoptOutputContract")
}

func parseOutputContractSnapshot(raw []byte, entrypoints []string) (outputContractSnapshot, error) {
	if len(raw) == 0 || len(raw) > maximumOutputSnapshotBytes {
		return outputContractSnapshot{}, errors.New("output-contract snapshot size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var snapshot outputContractSnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return outputContractSnapshot{}, fmt.Errorf("decode output-contract snapshot: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return outputContractSnapshot{}, errors.New("output-contract snapshot has trailing content")
	}
	if snapshot.SchemaVersion != outputSnapshotSchema || snapshot.GradleVersion == "" || len(snapshot.Tasks) > 50000 || !sameStringSet(snapshot.Entrypoints, entrypoints) {
		return outputContractSnapshot{}, errors.New("output-contract snapshot identity is invalid")
	}
	seenTasks := map[string]bool{}
	for _, task := range snapshot.Tasks {
		if task.TaskPath == "" || task.ProjectPath == "" || seenTasks[task.TaskPath] || len(task.Outputs) > 10000 {
			return outputContractSnapshot{}, errors.New("output-contract task identity is invalid")
		}
		seenTasks[task.TaskPath] = true
		for _, output := range task.Outputs {
			if output.FileCount < 0 || (output.Kind != "FILE" && output.Kind != "DIRECTORY") || (output.InsideRepository && !validObservedOutputPath(output.Path)) {
				return outputContractSnapshot{}, errors.New("output-contract task output is invalid")
			}
		}
	}
	return snapshot, nil
}

func collectOutputOwnership(repositoryRoot string, snapshot outputContractSnapshot) ([]outputRootOwnership, []outputContractCandidate, error) {
	byRoot := map[string]*outputRootOwnership{}
	for _, task := range snapshot.Tasks {
		for _, output := range task.Outputs {
			if !output.InsideRepository || !output.Exists || output.Symlink || output.FileCount == 0 || output.Path == "" {
				continue
			}
			key := output.Kind + "\x00" + output.Path
			owner := byRoot[key]
			if owner == nil {
				files, err := regularOutputFiles(repositoryRoot, output)
				if err != nil {
					return nil, nil, err
				}
				owner = &outputRootOwnership{path: output.Path, kind: output.Kind, files: files, projects: map[string]bool{}, tasks: map[string]bool{}}
				byRoot[key] = owner
			}
			owner.projects[task.ProjectPath] = true
			owner.tasks[task.TaskPath] = true
		}
	}
	owners := make([]outputRootOwnership, 0, len(byRoot))
	for _, owner := range byRoot {
		owners = append(owners, *owner)
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].path != owners[j].path {
			return owners[i].path < owners[j].path
		}
		return owners[i].kind < owners[j].kind
	})
	candidates := make([]outputContractCandidate, 0, len(owners))
	for _, owner := range owners {
		pattern := owner.path
		if owner.kind == "DIRECTORY" {
			pattern += "/**"
		}
		candidates = append(candidates, outputContractCandidate{
			Pattern: pattern, Path: owner.path, Kind: owner.kind, FileCount: len(owner.files),
			OwnerProjects: sortedSet(owner.projects), ProducerTasks: sortedSet(owner.tasks),
		})
	}
	return owners, candidates, nil
}

func regularOutputFiles(repositoryRoot string, output outputContractRoot) ([]string, error) {
	absolute := filepath.Join(repositoryRoot, filepath.FromSlash(output.Path))
	if output.Kind == "FILE" {
		return []string{output.Path}, nil
	}
	files := make([]string, 0, output.FileCount)
	err := filepath.WalkDir(absolute, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(repositoryRoot, candidate)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		if len(files) > maximumOutputContractFiles {
			return errors.New("output-contract file set exceeds the review bound")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk output-contract root %s: %w", output.Path, err)
	}
	sort.Strings(files)
	return files, nil
}

func validateDeclaredOutputs(patterns []string, owners []outputRootOwnership) []outputContractValidation {
	validations := make([]outputContractValidation, 0, len(patterns))
	for _, pattern := range patterns {
		matchedFiles := map[string]bool{}
		projects, tasks := map[string]bool{}, map[string]bool{}
		ambiguous := false
		for _, owner := range owners {
			for _, file := range owner.files {
				if !matchProposalGlob(pattern, file) {
					continue
				}
				matchedFiles[file] = true
				fileProjects, fileTasks := mostSpecificOutputOwners(file, owners)
				if len(fileProjects) != 1 {
					ambiguous = true
				}
				for project := range fileProjects {
					projects[project] = true
				}
				for task := range fileTasks {
					tasks[task] = true
				}
			}
		}
		status := "VALIDATED"
		if len(matchedFiles) == 0 {
			status = "EMPTY"
		} else if ambiguous {
			status = "AMBIGUOUS"
		}
		validations = append(validations, outputContractValidation{
			Pattern: pattern, Status: status, MatchedFiles: len(matchedFiles),
			OwnerProjects: sortedSet(projects), ProducerTasks: sortedSet(tasks),
		})
	}
	return validations
}

func mostSpecificOutputOwners(file string, owners []outputRootOwnership) (map[string]bool, map[string]bool) {
	projects, tasks := map[string]bool{}, map[string]bool{}
	maximumSpecificity := -1
	for _, owner := range owners {
		matches := owner.kind == "FILE" && owner.path == file
		if owner.kind == "DIRECTORY" {
			matches = file == owner.path || strings.HasPrefix(file, owner.path+"/")
		}
		if !matches {
			continue
		}
		specificity := len(strings.Split(owner.path, "/"))
		if specificity < maximumSpecificity {
			continue
		}
		if specificity > maximumSpecificity {
			projects, tasks = map[string]bool{}, map[string]bool{}
			maximumSpecificity = specificity
		}
		for project := range owner.projects {
			projects[project] = true
		}
		for task := range owner.tasks {
			tasks[task] = true
		}
	}
	return projects, tasks
}

func validOutputContractPattern(candidate string) bool {
	if candidate == "" || strings.TrimSpace(candidate) != candidate || filepath.IsAbs(candidate) || strings.ContainsAny(candidate, "\\\x00") {
		return false
	}
	parts := strings.Split(candidate, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
		if part != "**" {
			if _, err := filepath.Match(part, part); err != nil {
				return false
			}
		}
	}
	return true
}

func validObservedOutputPath(candidate string) bool {
	if candidate == "" || filepath.IsAbs(candidate) || strings.Contains(candidate, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(candidate)))
	return clean == candidate && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func replaceOutputContractEnvironment(environment []string, replacements map[string]string) []string {
	result := make([]string, 0, len(environment)+len(replacements))
	for _, item := range environment {
		name := item
		if index := strings.IndexByte(item, '='); index >= 0 {
			name = item[:index]
		}
		if _, replaced := replacements[name]; !replaced {
			result = append(result, item)
		}
	}
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, key+"="+replacements[key])
	}
	return result
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	a, b := append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func renderOutputContract(report outputContractReport) ([]byte, error) {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
