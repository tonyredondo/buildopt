package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

type outputPolicy struct {
	Selectors          []string `json:"selectors"`
	Producers          []string `json:"producers"`
	Comparison         string   `json:"comparison"`
	Inventory          string   `json:"inventory"`
	AllowMissing       bool     `json:"allowMissing"`
	AllowSymlinks      bool     `json:"allowSymlinks"`
	AllowUnowned       bool     `json:"allowUnowned"`
	AllowNormalization bool     `json:"allowNormalization"`
}
type outputFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Producer string `json:"producer"`
}
type graphDocument struct {
	Schema string `json:"schemaVersion"`
	Build  string `json:"buildPath"`
	Tasks  []struct {
		Identity     string   `json:"identity"`
		Path         string   `json:"path"`
		Class        string   `json:"taskClass"`
		Dependencies []string `json:"dependencies"`
		Outputs      []string `json:"outputs"`
	} `json:"tasks"`
}

func outputProducers(graph string, policy outputPolicy) (map[string][]string, error) {
	if policy.AllowMissing || policy.AllowSymlinks || policy.AllowUnowned || policy.AllowNormalization || len(policy.Selectors) == 0 || len(policy.Producers) == 0 {
		return nil, errors.New("unsupported output policy")
	}
	file, err := os.Open(graph)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	producers := map[string][]string{}
	builds := map[string]bool{}
	for scanner.Scan() {
		var document graphDocument
		if err = decodeDocument(scanner.Bytes(), &document); err != nil {
			return nil, err
		}
		if document.Schema != "buildopt.diagnostics/gradle-task-graph/v1" || document.Build == "" || builds[document.Build] {
			return nil, errors.New("ambiguous graph document")
		}
		builds[document.Build] = true
		if document.Build != ":" {
			continue
		}
		for _, task := range document.Tasks {
			if task.Identity != task.Path || task.Identity == "" || task.Class == "" {
				return nil, errors.New("graph task identity drift")
			}
			if _, exists := producers[task.Identity]; exists {
				return nil, errors.New("duplicate graph producer")
			}
			for _, path := range task.Outputs {
				if !filepath.IsLocal(path) || filepath.Clean(path) != path || path == "." {
					return nil, errors.New("output declaration escapes source")
				}
			}
			producers[task.Identity] = task.Outputs
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	for _, name := range policy.Producers {
		if len(producers[name]) == 0 {
			return nil, errors.New("required output producer is absent")
		}
	}
	return producers, nil
}

func inventoryOutputs(root, graph string, policy outputPolicy) ([]outputFile, error) {
	producers, err := outputProducers(graph, policy)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, name := range policy.Producers {
		allowed[name] = true
	}
	seen := map[string]bool{}
	outputs := []outputFile{}
	ownedCounts := map[string]int{}
	for _, selector := range policy.Selectors {
		if !filepath.IsLocal(selector) || filepath.Clean(selector) != selector {
			return nil, errors.New("unsafe output selector")
		}
		matches, err := filepath.Glob(filepath.Join(root, selector))
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, errors.New("required output selector is empty")
		}
		for _, path := range matches {
			if err = contained(root, path); err != nil {
				return nil, err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil, err
			}
			if seen[rel] {
				return nil, errors.New("overlapping output selectors")
			}
			seen[rel] = true
			producer := ""
			for name, declarations := range producers {
				for _, declared := range declarations {
					if rel == declared || strings.HasPrefix(rel, declared+string(filepath.Separator)) {
						if producer != "" && producer != name {
							return nil, errors.New("ambiguous output ownership")
						}
						producer = name
					}
				}
			}
			if !allowed[producer] {
				return nil, errors.New("unexpected output producer")
			}
			hash, err := fileDigest(path)
			if err != nil {
				return nil, err
			}
			info, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, outputFile{filepath.ToSlash(rel), info.Size(), hash, producer})
			ownedCounts[producer]++
		}
	}
	for _, name := range policy.Producers {
		if ownedCounts[name] == 0 {
			return nil, errors.New("producer has no required output")
		}
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	return outputs, nil
}

func retainOutputs(source, attempt string, policy outputPolicy) error {
	outputs, err := inventoryOutputs(source, filepath.Join(attempt, "task-graph.jsonl"), policy)
	if err != nil {
		return err
	}
	for _, output := range outputs {
		target := filepath.Join(attempt, "outputs", output.Path)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err = copyNewFile(filepath.Join(source, output.Path), target); err != nil {
			return err
		}
		hash, err := fileDigest(target)
		if err != nil || hash != output.SHA256 {
			return errors.New("output changed during retention")
		}
	}
	return writeNewJSON(filepath.Join(attempt, "output-inventory.json"), outputs)
}

func checkOutputs(attempt string, policy outputPolicy) error {
	var retained []outputFile
	if err := readJSON(filepath.Join(attempt, "output-inventory.json"), &retained); err != nil {
		return err
	}
	actual, err := inventoryOutputs(filepath.Join(attempt, "outputs"), filepath.Join(attempt, "task-graph.jsonl"), policy)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, retained) {
		return errors.New("retained output inventory drift")
	}
	return nil
}
