package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func outputFixture(t *testing.T) (string, string, outputPolicy) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	attempt := filepath.Join(root, "attempt")
	if err := os.MkdirAll(filepath.Join(source, "build/libs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(attempt, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "build/libs/output.jar"), []byte("exact native bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	graph := `{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[{"identity":":producer","path":":producer","taskClass":"OwnerTask","dependencies":[],"outputs":["build/libs/output.jar"]}]}`
	if err := os.WriteFile(filepath.Join(attempt, "task-graph.jsonl"), []byte(graph+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return source, attempt, outputPolicy{Selectors: []string{"build/libs/*.jar"}, Producers: []string{":producer"}}
}

func TestOutputInventoryAndIndependentReconstruction(t *testing.T) {
	source, attempt, policy := outputFixture(t)
	if err := retainOutputs(source, attempt, policy); err != nil {
		t.Fatal(err)
	}
	if err := checkOutputs(attempt, policy); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, attempt)
	if err := checkOutputs(attempt, policy); err != nil {
		t.Fatal(err)
	}
	if before != snapshot(t, attempt) {
		t.Fatal("inventory check wrote evidence")
	}
	path := filepath.Join(attempt, "outputs/build/libs/output.jar")
	if err := os.WriteFile(path, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkOutputs(attempt, policy); err == nil {
		t.Fatal("changed output accepted")
	}
}

func TestOutputNegatives(t *testing.T) {
	for _, mode := range []string{"missing", "extra", "symlink", "owner", "duplicate", "summary"} {
		t.Run(mode, func(t *testing.T) {
			source, attempt, policy := outputFixture(t)
			path := filepath.Join(source, "build/libs/output.jar")
			graph := filepath.Join(attempt, "task-graph.jsonl")
			switch mode {
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "extra":
				if err := os.WriteFile(filepath.Join(source, "build/libs/extra.jar"), []byte("unowned"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(graph, path); err != nil {
					t.Fatal(err)
				}
			case "owner":
				policy.Producers = []string{":absent"}
			case "duplicate":
				data, err := os.ReadFile(graph)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(graph, append(data, data...), 0600); err != nil {
					t.Fatal(err)
				}
			case "summary":
				if err := retainOutputs(source, attempt, policy); err != nil {
					t.Fatal(err)
				}
				var rows []outputFile
				if err := readJSON(filepath.Join(attempt, "output-inventory.json"), &rows); err != nil {
					t.Fatal(err)
				}
				rows[0].Producer = ":forged"
				data, _ := json.Marshal(rows)
				if err := os.WriteFile(filepath.Join(attempt, "output-inventory.json"), data, 0600); err != nil {
					t.Fatal(err)
				}
				if err := checkOutputs(attempt, policy); err == nil {
					t.Fatal("summary trusted")
				}
				return
			}
			if err := retainOutputs(source, attempt, policy); err == nil {
				t.Fatal("invalid output accepted")
			}
		})
	}
}

func TestStrictStateJSONAndSymlinks(t *testing.T) {
	for _, raw := range []string{`{"maximumStarts":60,"maximumStarts":61}`, `{} {}`, `{"unknown":true}`, strings.Repeat(" ", 4<<20) + "{}"} {
		path := filepath.Join(t.TempDir(), "state.json")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		var state campaign
		if err := readJSON(path, &state); err == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
	root := t.TempDir()
	path := filepath.Join(root, "target.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	var state campaign
	if err := readJSON(link, &state); err == nil {
		t.Fatal("symlink state accepted")
	}
}
