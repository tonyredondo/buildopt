package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

//go:embed native-diagnostic.init.gradle
var nativeDiagnosticScript string

type nativeGraph struct {
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

type nativeMaterialityResult struct {
	Task           string  `json:"task"`
	Outcome        string  `json:"outcome"`
	DurationMs     int64   `json:"durationMs"`
	Critical       bool    `json:"onCriticalPath"`
	CriticalPathMs int64   `json:"criticalPathMs"`
	Fraction       float64 `json:"criticalPathFraction"`
	Passed         bool    `json:"passed"`
}

func makeNativeDiagnosticRequest(root, runner string, r row) (nativeRequest, error) {
	if !reflect.DeepEqual(r, makeProtocol().Rows[3]) && !reflect.DeepEqual(r, makeProtocol().Rows[4]) {
		return nativeRequest{}, errors.New("only frozen D001/D002 native diagnostics admitted")
	}
	return makeObservedNativeRequest(root, runner, r)
}

func makeObservedNativeRequest(root, runner string, r row) (nativeRequest, error) {
	request, err := makeNativeRequest(root, runner, makeProtocol().Rows[0])
	if err != nil {
		return request, err
	}
	request.Row = r
	request.Diagnostic = &fileBinding{Path: filepath.Join(root, "runtime", "native-diagnostic.init.gradle"), SHA256: digest([]byte(nativeDiagnosticScript))}
	attempt := filepath.Join(root, "attempts", r.ID)
	request.Arguments = append(request.Arguments, "--init-script", request.Diagnostic.Path, "-Dorg.gradle.internal.operations.trace="+filepath.Join(attempt, "operations"))
	request.Environment = append(request.Environment, "BUILDOPT_TASK_GRAPH_OUTPUT="+filepath.Join(attempt, "task-graph.jsonl"))
	return request, nil
}

// Preserve all main-build declarations. Collapse only contained selectors for
// inventory traversal, retaining the full task graph as the ownership contract.
func nativeOutputSelectors(path string) ([]string, error) {
	return graphOutputSelectors(path, []string{":server:precommit", ":server:forbiddenPatterns"})
}

func graphOutputSelectors(path string, required []string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	builds, tasks, outputs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for scanner.Scan() {
		if _, err = contractcrypto.CanonicalizeJCS(scanner.Bytes()); err != nil {
			return nil, err
		}
		var graph nativeGraph
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&graph); err != nil {
			return nil, err
		}
		if graph.Schema != gradlecriticalpath.GraphSchema || graph.Build == "" || builds[graph.Build] || len(graph.Tasks) == 0 {
			return nil, errors.New("missing/duplicate native graph")
		}
		builds[graph.Build] = true
		for _, t := range graph.Tasks {
			if t.Identity == "" || tasks[t.Identity] || t.Class == "" {
				return nil, errors.New("missing/duplicate native task")
			}
			tasks[t.Identity] = true
			if graph.Build != ":" {
				continue
			}
			if t.Identity != t.Path {
				return nil, errors.New("main-build task identity drift")
			}
			for _, output := range t.Outputs {
				if !filepath.IsLocal(output) || filepath.Clean(output) != output || output == "." || strings.Contains(output, "\\") {
					return nil, errors.New("native output escapes root")
				}
				outputs[output] = true
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	for _, owner := range required {
		if !tasks[owner] {
			return nil, errors.New("required owner missing")
		}
	}
	if len(outputs) == 0 {
		return nil, errors.New("owner workflow or required outputs missing")
	}
	all := make([]string, 0, len(outputs))
	for output := range outputs {
		all = append(all, output)
	}
	sort.Strings(all)
	selected := []string{}
	for _, output := range all {
		contained := false
		for _, prior := range selected {
			if strings.HasPrefix(output, prior+"/") {
				contained = true
				break
			}
		}
		if !contained {
			selected = append(selected, output)
		}
	}
	return selected, nil
}

func nativeMateriality(report gradlecriticalpath.Report) (nativeMaterialityResult, error) {
	result := nativeMaterialityResult{Task: ":server:forbiddenPatterns", CriticalPathMs: report.Summary.MainBuildCriticalPathMs}
	if result.CriticalPathMs <= 0 {
		return result, errors.New("missing native critical path")
	}
	count := 0
	for _, t := range report.Tasks {
		if t.Identity != result.Task {
			continue
		}
		count++
		result.Outcome, result.DurationMs, result.Critical = t.Outcome, t.DurationMs, t.CriticalPath
	}
	if count != 1 {
		return result, errors.New("missing/duplicate materiality task")
	}
	result.Fraction = float64(result.DurationMs) / float64(result.CriticalPathMs)
	result.Passed = result.Outcome == "EXECUTED" && result.Critical && result.DurationMs >= 500 && result.DurationMs*100 >= result.CriticalPathMs*2
	return result, nil
}

func retainNativeDiagnostic(root, slot string) error {
	return retainArmDiagnostic(root, slot, "N0", []string{":server:precommit", ":server:forbiddenPatterns"})
}

func retainArmDiagnostic(root, slot, arm string, required []string) error {
	dir := filepath.Join(root, "attempts", slot)
	report, err := gradlecriticalpath.Analyze(filepath.Join(dir, "operations-log.txt"), filepath.Join(dir, "task-graph.jsonl"), "control")
	if err != nil {
		return err
	}
	if err = writeJSON(filepath.Join(dir, "critical-path.json"), report); err != nil {
		return err
	}
	materiality, err := nativeMateriality(report)
	if err != nil {
		return err
	}
	if err = writeJSON(filepath.Join(dir, "materiality.json"), materiality); err != nil {
		return err
	}
	selectors, err := graphOutputSelectors(filepath.Join(dir, "task-graph.jsonl"), required)
	if err != nil {
		return err
	}
	if err = writeJSON(filepath.Join(dir, "output-selectors.json"), selectors); err != nil {
		return err
	}
	source := filepath.Join(root, "arms", arm)
	before, err := inventory(source, selectors)
	if err != nil {
		return err
	}
	if err = writeJSON(filepath.Join(dir, "output-inventory.json"), before); err != nil {
		return err
	}
	for _, selector := range selectors {
		src, dst := filepath.Join(source, selector), filepath.Join(dir, "outputs", selector)
		_, e := os.Lstat(src)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return e
		}
		if e = os.MkdirAll(filepath.Dir(dst), 0700); e != nil {
			return e
		}
		e = copyTree(src, dst)
		if e != nil {
			return e
		}
	}
	after, err := inventory(filepath.Join(dir, "outputs"), selectors)
	if err != nil || !reflect.DeepEqual(before, after) {
		return errors.New("native output retention changed bytes/types/modes/absence")
	}
	bindings := map[string]string{}
	for _, name := range nativeDiagnosticArtifacts {
		pin, e := hashFile(filepath.Join(dir, name))
		if e != nil {
			return e
		}
		bindings[name] = pin
	}
	return writeJSON(filepath.Join(dir, "diagnostic-complete.json"), bindings)
}

var nativeDiagnosticArtifacts = []string{"native-request.json", "native-result.json", "process.json", "stdout.log", "stderr.log", "operations-log.txt", "task-graph.jsonl", "critical-path.json", "materiality.json", "output-selectors.json", "output-inventory.json"}

func verifyNativeDiagnosticComplete(dir string) error {
	var bindings map[string]string
	if err := readJSON(filepath.Join(dir, "diagnostic-complete.json"), &bindings); err != nil {
		return err
	}
	if len(bindings) != len(nativeDiagnosticArtifacts) {
		return errors.New("diagnostic completion artifact set drift")
	}
	for _, name := range nativeDiagnosticArtifacts {
		if err := checkBinding(fileBinding{Path: filepath.Join(dir, name), SHA256: bindings[name]}); err != nil {
			return err
		}
	}
	return nil
}
