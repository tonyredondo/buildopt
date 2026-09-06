package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func documents(t *testing.T) ([]byte, []byte) {
	t.Helper()
	contract, err := os.ReadFile("../../specs/poc-complete-native-correction-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	subjects, err := os.ReadFile("../../specs/poc-complete-native-correction-v1.subjects.json")
	if err != nil {
		t.Fatal(err)
	}
	return contract, subjects
}

func mutate(t *testing.T, data []byte, change func(map[string]any)) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	change(object)
	result, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func object(m map[string]any, key string) map[string]any { return m[key].(map[string]any) }
func array(m map[string]any, key string) []any           { return m[key].([]any) }
func firstSubject(m map[string]any) map[string]any       { return array(m, "subjects")[0].(map[string]any) }

func TestContractAndLabelIndependence(t *testing.T) {
	contract, subjects := documents(t)
	if err := validate(contract, subjects); err != nil {
		t.Fatal(err)
	}
	relabeled := mutate(t, subjects, func(m map[string]any) { firstSubject(m)["label"] = "Unrelated label; no admission rule" })
	if err := validate(contract, relabeled); err != nil {
		t.Fatalf("label affected acceptance: %v", err)
	}
}

func TestContractNegatives(t *testing.T) {
	contract, subjects := documents(t)
	cases := []struct {
		name   string
		change func(map[string]any)
	}{
		{"unknown field", func(m map[string]any) { m["unreviewed"] = true }},
		{"missing policy", func(m map[string]any) { delete(m, "environment") }},
		{"unknown nested field", func(m map[string]any) { object(m, "budget")["freeBuilds"] = 1 }},
		{"longer deadline", func(m map[string]any) { object(m, "budget")["maximumElapsedSeconds"] = 28800 }},
		{"missing checkpoint", func(m map[string]any) { delete(object(m, "budget"), "progressReviewSeconds") }},
		{"extra start", func(m map[string]any) { object(m, "budget")["maximumStarts"] = 61 }},
		{"untrue summary", func(m map[string]any) { object(m, "allocation")["fixtures"] = 23 }},
		{"missing proof", func(m map[string]any) { m["rows"] = array(m, "rows")[1:] }},
		{"duplicate row", func(m map[string]any) { r := array(m, "rows"); r[1] = r[0] }},
		{"late prerequisite", func(m map[string]any) { r := array(m, "rows"); r[2], r[3] = r[3], r[2] }},
		{"execution order drift", func(m map[string]any) { r := array(m, "executionOrder"); r[0], r[6] = r[6], r[0] }},
		{"masked input tracking", func(m map[string]any) {
			array(m, "fixtureInputProofs")[1].(map[string]any)["maskingChangeAllowed"] = true
		}},
		{"forced task invalidation", func(m map[string]any) {
			object(m, "commands")["candidateCorrectness"] = append(array(object(m, "commands"), "candidateCorrectness"), "--rerun-tasks")
		}},
		{"unknown row field", func(m map[string]any) { array(m, "rows")[0].(map[string]any)["skip"] = true }},
		{"missing cache proof", func(m map[string]any) { array(m, "rows")[4].(map[string]any)["cache"] = "fresh" }},
		{"increased reserve", func(m map[string]any) { object(m, "budget")["maximumInfrastructureReplacements"] = 3 }},
		{"name rule", func(m map[string]any) { object(m, "authority")["repositoryOrTaskNameRules"] = true }},
		{"historical row", func(m map[string]any) { object(m, "authority")["predecessorRowsAsEvidence"] = true }},
		{"execution escalation", func(m map[string]any) { object(m, "authority")["realGradleExecution"] = true }},
		{"ci gate", func(m map[string]any) { object(m, "gates")["ciWallTimeGate"] = true }},
		{"output weakening", func(m map[string]any) { object(m, "gates")["exactOutputBytes"] = false }},
		{"clock reset", func(m map[string]any) { object(m, "budget")["resetOnResume"] = true }},
		{"version override", func(m map[string]any) { object(m, "commands")["native"] = []any{"assemble", "-PreleaseVersion=fixed"} }},
		{"wrong percentile", func(m map[string]any) { object(m, "gates")["intervalIndices"] = []any{103, 3994} }},
		{"dropped owner test", func(m map[string]any) { object(m, "commands")["nativeCorrectness"] = []any{"assemble"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validate(mutate(t, contract, tc.change), subjects); err == nil {
				t.Fatal("accepted invalid contract")
			}
		})
	}
	for _, data := range [][]byte{[]byte("{"), append(append([]byte{}, contract...), []byte(" {}")...), bytes.Replace(contract, []byte(`"maximumStarts": 60`), []byte(`"maximumStarts": 60, "maximumStarts": 60`), 1)} {
		if err := validate(data, subjects); err == nil {
			t.Fatal("accepted malformed, trailing or duplicate JSON")
		}
	}
}

func TestSubjectNegatives(t *testing.T) {
	contract, subjects := documents(t)
	cases := []struct {
		name   string
		change func(map[string]any)
	}{
		{"source drift", func(m map[string]any) {
			array(firstSubject(m), "files")[0].(map[string]any)["sha256"] = strings.Repeat("0", 64)
		}},
		{"span drift", func(m map[string]any) { array(firstSubject(m), "bindings")[0].(map[string]any)["endLine"] = 145 }},
		{"unowned output", func(m map[string]any) { object(firstSubject(m), "outputs")["allowUnowned"] = true }},
		{"missing subproject", func(m map[string]any) { object(firstSubject(m), "outputs")["selectors"] = []any{"build/libs/*.jar"} }},
		{"runtime drift", func(m map[string]any) { array(m, "runtimes")[0].(map[string]any)["version"] = "latest" }},
		{"unknown name rule", func(m map[string]any) { firstSubject(m)["taskNameRule"] = "jar" }},
		{"missing proof binding", func(m map[string]any) { array(firstSubject(m), "bindings")[1].(map[string]any)["proof"] = []any{} }},
		{"different revision", func(m map[string]any) { firstSubject(m)["revision"] = strings.Repeat("0", 40) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validate(contract, mutate(t, subjects, tc.change)); err == nil {
				t.Fatal("accepted invalid subject")
			}
		})
	}
}

func TestRegularFilesAndSpans(t *testing.T) {
	root := t.TempDir()
	source := []byte("first\r\nsecond\r\nthird")
	if err := os.WriteFile(filepath.Join(root, "build.gradle"), source, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readRegular(root, "build.gradle")
	if err != nil || !bytes.Equal(source, got) {
		t.Fatalf("read: %q %v", got, err)
	}
	want := fmt.Sprintf("%x", sha256.Sum256([]byte("second\r\n")))
	if err := verifySpan(source, 2, 2, want); err != nil {
		t.Fatal(err)
	}
	for _, span := range [][2]int{{0, 2}, {2, 1}, {2, 5}} {
		if err := verifySpan(source, span[0], span[1], want); err == nil {
			t.Fatal("accepted invalid span")
		}
	}
	if err := verifySpan([]byte("first\nsecond\nthird"), 2, 2, want); err == nil {
		t.Fatal("normalized line endings")
	}
	for _, path := range []string{"../build.gradle", "/etc/passwd", "a/../build.gradle", "."} {
		if _, err := readRegular(root, path); err == nil {
			t.Fatalf("accepted unsafe path %q", path)
		}
	}
	if err := os.Symlink("build.gradle", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegular(root, "link"); err == nil {
		t.Fatal("accepted symlink")
	}
}

func TestFileAndSpanDriftReconstruction(t *testing.T) {
	root := t.TempDir()
	data := []byte("one\ntwo\n")
	path := filepath.Join(root, "build.gradle")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(fmt.Sprintf(`{"subjects":[{"files":[{"path":"build.gradle","sha256":"%x"}],"bindings":[{"path":"build.gradle","startLine":2,"endLine":2,"sha256":"%x"}]}]}`, sha256.Sum256(data), sha256.Sum256([]byte("two\n"))))
	if err := verifySources(root, manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one\nchanged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifySources(root, manifest); err == nil {
		t.Fatal("accepted changed source bytes")
	}
}
