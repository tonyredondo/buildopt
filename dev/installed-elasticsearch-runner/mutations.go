package main

import (
	"bytes"
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const problemsSourcePath = "build-conventions/src/main/java/org/elasticsearch/gradle/internal/conventions/problems/ElasticsearchBuildProblems.java"
const problemsSourceSHA256 = "851636f0393d5fe81636dd91dd633aa993a6ec530390fae8208dc5a588be31a7"

//go:embed mutation-fixture-plugin.java.txt
var mutationPlugin []byte

func mutationRows() []row {
	var rows []row
	for _, r := range makeProtocol().Rows {
		if r.Block == "M" {
			r.ID = "Q" + r.ID
			r.Block = "QM"
			rows = append(rows, r)
		}
	}
	return rows
}

const correctnessMutationClass = "ELASTICSEARCH_CORRECTNESS_MUTATIONS"

func mutationRowsForClass(class string) []row {
	if class != correctnessMutationClass {
		return mutationRows()
	}
	var rows []row
	for _, r := range makeProtocol().Rows {
		if r.Block == "M" {
			rows = append(rows, r)
		}
	}
	return rows
}

// Produce a standalone fixture from the exact owner class, without changing any
// subject worktree. The seed task and measured task use that same class; a cache
// restore before each mutation must be observed from Gradle, never inferred.
func prepareMutationProject(root string, r row, task, problems []byte) (string, error) {
	valid := false
	for _, expected := range append(mutationRows(), mutationRowsForClass(correctnessMutationClass)...) {
		if reflect.DeepEqual(r, expected) {
			valid = true
		}
	}
	if !valid || digest(task) != sourceInputs().TaskPreimageSHA256 || digest(problems) != problemsSourceSHA256 {
		return "", errors.New("mutation row or upstream source drift")
	}
	kind, phase, ok := strings.Cut(r.State, ":")
	if !ok {
		return "", errors.New("invalid mutation state")
	}
	dir := filepath.Join(root, "work", kind, r.Arm)
	if phase == "before" {
		if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
			return "", err
		}
		if err := os.Mkdir(dir, 0700); err != nil {
			return "", err
		}
		if r.Arm == "N1" {
			task = bytes.Replace(task, []byte("import org.gradle.api.tasks.IgnoreEmptyDirectories;"), []byte("import org.gradle.api.tasks.CacheableTask;\nimport org.gradle.api.tasks.IgnoreEmptyDirectories;"), 1)
			task = bytes.Replace(task, []byte("public abstract class ForbiddenPatternsTask"), []byte("@CacheableTask\npublic abstract class ForbiddenPatternsTask"), 1)
			if digest(task) != sourceInputs().TaskPostimageSHA256 {
				return "", errors.New("candidate differs from exact approved annotation patch")
			}
		}
		files := mutationBaseFiles(task, problems)
		for name, data := range files {
			p := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				return "", err
			}
			if err := writeNew(p, data); err != nil {
				return "", err
			}
		}
		return dir, nil
	}
	previous, err := os.ReadFile(filepath.Join(dir, ".eic-phase"))
	if err != nil || string(previous) != "before" {
		return "", errors.New("mutation requires its untouched before state")
	}
	switch kind {
	case "relative-rename":
		err = os.Rename(filepath.Join(dir, "src/input.txt"), filepath.Join(dir, "src/renamed.txt"))
	case "all-source-removal":
		for _, name := range []string{"input.txt", "excluded.txt"} {
			if e := os.Remove(filepath.Join(dir, "src", name)); e != nil {
				return "", e
			}
		}
	case "malformed-utf8":
		err = os.WriteFile(filepath.Join(dir, "src/input.txt"), []byte{0xc3, 0x28}, 0600)
	// Rules, excludes and relative root are changed only by the frozen plugin's
	// phase property. The input source bytes remain identical in those cases.
	case "rules", "excludes", "relative-root":
	default:
		return "", errors.New("unknown mutation")
	}
	if err != nil {
		return "", err
	}
	return dir, os.WriteFile(filepath.Join(dir, ".eic-phase"), []byte("after"), 0600)
}

func mutationBaseFiles(task, problems []byte) map[string][]byte {
	return map[string][]byte{
		"settings.gradle":       []byte("rootProject.name = 'eic-forbidden-patterns-mutation'\n"),
		"build.gradle":          []byte("apply plugin: org.elasticsearch.gradle.internal.precommit.MutationFixturePlugin\n"),
		"buildSrc/build.gradle": []byte("plugins { id 'java' }\ndependencies { implementation gradleApi() }\n"),
		"buildSrc/src/main/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTask.java":                 task,
		"buildSrc/src/main/java/org/elasticsearch/gradle/internal/conventions/problems/ElasticsearchBuildProblems.java": problems,
		"buildSrc/src/main/java/org/elasticsearch/gradle/internal/precommit/MutationFixturePlugin.java":                 mutationPlugin,
		"src/input.txt": []byte("valid source input\n"), "src/excluded.txt": []byte("another valid input\n"), ".eic-phase": []byte("before"),
	}
}
