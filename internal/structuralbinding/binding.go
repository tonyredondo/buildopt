// Package structuralbinding derives revision-independent compatibility
// fingerprints for qualified Gradle build profiles.
package structuralbinding

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"strings"
)

const SchemaVersion = "buildopt.poc/structural-profile-binding/v1"

type Task struct {
	Path        string
	ProjectPath string
	DependsOn   []string
}

type Output struct {
	Pattern       string
	Kind          string
	OwnerProjects []string
	ProducerTasks []string
}

type Input struct {
	RepositoryID         string
	WrapperSHA256        string
	OriginalEntrypoints  []string
	CandidateEntrypoints []string
	GradleOptions        []string
	RequiredOutputs      []string
	CandidateOutputs     []string
	ChangeFamily         string
	ChangedProjects      []string
	Tasks                []Task
	Outputs              []Output
}

type Binding struct {
	SchemaVersion         string `json:"schemaVersion"`
	RepositoryScopeSHA256 string `json:"repositoryScopeSha256"`
	WorkflowSHA256        string `json:"workflowSha256"`
	WrapperSHA256         string `json:"wrapperSha256"`
	ProducerLineageSHA256 string `json:"producerLineageSha256"`
	OutputContractSHA256  string `json:"outputContractSha256"`
	ChangeFamilySHA256    string `json:"changeFamilySha256"`
	SHA256                string `json:"sha256"`
}

func Derive(input Input) (Binding, error) {
	if strings.TrimSpace(input.RepositoryID) == "" || !validSHA(input.WrapperSHA256) ||
		strings.TrimSpace(input.ChangeFamily) == "" {
		return Binding{}, errors.New("structural binding identity is incomplete")
	}
	original, err := normalized(input.OriginalEntrypoints, false)
	if err != nil {
		return Binding{}, errors.New("structural binding workflow is incomplete")
	}
	candidate, err := normalized(input.CandidateEntrypoints, false)
	if err != nil {
		return Binding{}, errors.New("structural binding candidate workflow is incomplete")
	}
	options, err := normalized(input.GradleOptions, true)
	if err != nil {
		return Binding{}, errors.New("structural binding Gradle options are ambiguous")
	}
	required, err := normalized(input.RequiredOutputs, false)
	if err != nil {
		return Binding{}, errors.New("structural binding required outputs are incomplete")
	}
	candidateOutputs, err := normalized(input.CandidateOutputs, false)
	if err != nil || !subset(candidateOutputs, required) {
		return Binding{}, errors.New("structural binding candidate outputs are incomplete")
	}
	projects, err := normalized(input.ChangedProjects, false)
	if err != nil {
		return Binding{}, errors.New("structural binding change family is incomplete")
	}

	tasks, err := canonicalTasks(input.Tasks)
	if err != nil {
		return Binding{}, err
	}
	outputs, producers, err := canonicalOutputs(input.Outputs, required)
	if err != nil {
		return Binding{}, err
	}
	lineage, err := canonicalProducerLineage(tasks, producers, candidate)
	if err != nil {
		return Binding{}, err
	}

	repositoryScope, _ := RepositoryScopeSHA256(input.RepositoryID)
	workflow, _ := WorkflowSHA256(original, candidate, options)
	changeFamily, _ := ChangeFamilySHA256(input.ChangeFamily, projects)
	binding := Binding{
		SchemaVersion:         SchemaVersion,
		RepositoryScopeSHA256: repositoryScope,
		WorkflowSHA256:        workflow,
		WrapperSHA256:         input.WrapperSHA256,
		ProducerLineageSHA256: digest("buildopt-structural-producer-lineage-v1", lineage...),
		OutputContractSHA256: digest("buildopt-structural-output-contract-v1",
			digest("required-outputs-v1", required...),
			digest("candidate-outputs-v1", candidateOutputs...),
			digest("owned-outputs-v1", outputs...)),
		ChangeFamilySHA256: changeFamily,
	}
	binding.SHA256 = bindingDigest(binding)
	return binding, nil
}

func Valid(binding Binding) bool {
	return binding.SchemaVersion == SchemaVersion && validSHA(binding.RepositoryScopeSHA256) &&
		validSHA(binding.WorkflowSHA256) && validSHA(binding.WrapperSHA256) &&
		validSHA(binding.ProducerLineageSHA256) && validSHA(binding.OutputContractSHA256) &&
		validSHA(binding.ChangeFamilySHA256) && validSHA(binding.SHA256) &&
		binding.SHA256 == bindingDigest(binding)
}

func RepositoryScopeSHA256(repositoryID string) (string, error) {
	repositoryID = strings.TrimSpace(repositoryID)
	if repositoryID == "" {
		return "", errors.New("structural binding repository is incomplete")
	}
	return digest("buildopt-structural-repository-v1", repositoryID), nil
}

func WorkflowSHA256(original, candidate, options []string) (string, error) {
	original, err := normalized(original, false)
	if err != nil {
		return "", errors.New("structural binding workflow is incomplete")
	}
	candidate, err = normalized(candidate, false)
	if err != nil {
		return "", errors.New("structural binding candidate workflow is incomplete")
	}
	options, err = normalized(options, true)
	if err != nil {
		return "", errors.New("structural binding Gradle options are ambiguous")
	}
	return digest("buildopt-structural-workflow-v1",
		digest("original-entrypoints-v1", original...),
		digest("candidate-entrypoints-v1", candidate...),
		digest("gradle-options-v1", options...)), nil
}

func ChangeFamilySHA256(family string, projects []string) (string, error) {
	family = strings.TrimSpace(family)
	projects, err := normalized(projects, false)
	if family == "" || err != nil {
		return "", errors.New("structural binding change family is incomplete")
	}
	return digest("buildopt-structural-change-family-v1", family,
		digest("changed-projects-v1", projects...)), nil
}

func bindingDigest(binding Binding) string {
	return digest("buildopt-structural-profile-binding-v1", binding.SchemaVersion,
		binding.RepositoryScopeSHA256, binding.WorkflowSHA256, binding.WrapperSHA256,
		binding.ProducerLineageSHA256, binding.OutputContractSHA256,
		binding.ChangeFamilySHA256)
}

type canonicalTask struct {
	project      string
	dependencies []string
}

func canonicalTasks(values []Task) (map[string]canonicalTask, error) {
	if len(values) == 0 || len(values) > 250000 {
		return nil, errors.New("structural binding task lineage is incomplete")
	}
	result := make(map[string]canonicalTask, len(values))
	for _, value := range values {
		path, project := strings.TrimSpace(value.Path), strings.TrimSpace(value.ProjectPath)
		dependencies, err := normalized(value.DependsOn, true)
		if path == "" || project == "" || err != nil {
			return nil, errors.New("structural binding task lineage is incomplete")
		}
		if _, exists := result[path]; exists {
			return nil, errors.New("structural binding task lineage is ambiguous")
		}
		result[path] = canonicalTask{project: project, dependencies: dependencies}
	}
	for _, task := range result {
		for _, dependency := range task.dependencies {
			if _, exists := result[dependency]; !exists {
				return nil, errors.New("structural binding task lineage is incomplete")
			}
		}
	}
	return result, nil
}

func canonicalOutputs(values []Output, required []string) ([]string, []string, error) {
	byPattern := make(map[string]Output, len(values))
	for _, value := range values {
		pattern := strings.TrimSpace(value.Pattern)
		if pattern == "" {
			return nil, nil, errors.New("structural binding output contract is incomplete")
		}
		if _, exists := byPattern[pattern]; exists {
			return nil, nil, errors.New("structural binding output ownership is ambiguous")
		}
		byPattern[pattern] = value
	}
	rows := make([]string, 0, len(required))
	producers := []string{}
	for _, pattern := range required {
		value, exists := byPattern[pattern]
		owners, ownerErr := normalized(value.OwnerProjects, false)
		producerTasks, producerErr := normalized(value.ProducerTasks, false)
		if !exists || ownerErr != nil || producerErr != nil || len(owners) != 1 || strings.TrimSpace(value.Kind) == "" {
			return nil, nil, errors.New("structural binding output ownership is incomplete")
		}
		rows = append(rows, digest("output-v1", pattern, strings.TrimSpace(value.Kind), owners[0], digest("producers-v1", producerTasks...)))
		producers = append(producers, producerTasks...)
	}
	producers, err := normalized(producers, false)
	if err != nil {
		return nil, nil, errors.New("structural binding producer ownership is ambiguous")
	}
	return rows, producers, nil
}

func canonicalProducerLineage(tasks map[string]canonicalTask, producers, candidateEntrypoints []string) ([]string, error) {
	rootSet := make(map[string]bool, len(producers)+len(candidateEntrypoints))
	for _, root := range append(append([]string{}, producers...), candidateEntrypoints...) {
		root = strings.TrimSpace(root)
		if root == "" {
			return nil, errors.New("structural binding producer lineage is incomplete")
		}
		rootSet[root] = true
	}
	roots := make([]string, 0, len(rootSet))
	for root := range rootSet {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	visiting, visited := map[string]bool{}, map[string]bool{}
	rows := []string{}
	var visit func(string) error
	visit = func(path string) error {
		if visiting[path] {
			return errors.New("structural binding producer lineage is cyclic")
		}
		if visited[path] {
			return nil
		}
		task, exists := tasks[path]
		if !exists {
			return errors.New("structural binding producer lineage is incomplete")
		}
		visiting[path] = true
		for _, dependency := range task.dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[path], visited[path] = false, true
		rows = append(rows, digest("task-v1", path, task.project, digest("dependencies-v1", task.dependencies...)))
		return nil
	}
	for _, root := range roots {
		if err := visit(root); err != nil {
			return nil, err
		}
	}
	sort.Strings(rows)
	return rows, nil
}

func normalized(values []string, allowEmpty bool) ([]string, error) {
	if len(values) == 0 {
		if allowEmpty {
			return []string{}, nil
		}
		return nil, errors.New("values are empty")
	}
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strings.TrimSpace(value)
		if result[index] == "" {
			return nil, errors.New("value is empty")
		}
	}
	sort.Strings(result)
	for index := 1; index < len(result); index++ {
		if result[index] == result[index-1] {
			return nil, errors.New("values are not unique")
		}
	}
	return result, nil
}

func subset(values, complete []string) bool {
	available := make(map[string]bool, len(complete))
	for _, value := range complete {
		available[value] = true
	}
	for _, value := range values {
		if !available[value] {
			return false
		}
	}
	return true
}

func digest(domain string, values ...string) string {
	hash := sha256.New()
	writeValue(hash, domain)
	for _, value := range values {
		writeValue(hash, value)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeValue(writer io.Writer, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = io.WriteString(writer, value)
}

func validSHA(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
