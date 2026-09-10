//go:build linux && amd64 && replay_integration

package main

import (
	"path/filepath"
	"testing"
)

func TestQualifiedProjectionAndSemanticCounterexample(t *testing.T) {
	m := runnableFixture(t, 1, "projector")
	root := filepath.Dir(m.RunRoot)
	p := Projector{Identity: "fixture-root", Executable: m.Executable, Arguments: []string{"fixture-projector", "{path}", "{root}"}}
	proof := ProjectorProof{Schema: "buildopt.history-replay/projector-proof/v1", Identity: p.Identity, Executable: p.Executable, Arguments: p.Arguments, Cases: []ProjectionCase{}}
	now := stamp()
	for _, item := range []struct {
		name, kind, root, value string
		failure                 bool
	}{{"native-root", "equivalent", "/fixture/native", "expected", false}, {"candidate-root", "equivalent", "/fixture/candidate", "expected", false}, {"real-change", "semantic-change", "/fixture/candidate", "changed", false}, {"malformed", "malformed", "/fixture/native", "expected", true}} {
		path := filepath.Join(root, "projector-cases", item.name+".txt")
		raw := "root=" + item.root + "\nvalue=" + item.value + "\n"
		if item.failure {
			raw += "unapproved semantic line\n"
		}
		mustWrite(t, path, []byte(raw))
		expected := ""
		if !item.failure {
			expected = digest([]byte("value=" + item.value + "\n"))
		}
		proof.Cases = append(proof.Cases, ProjectionCase{item.name, item.kind, bound(t, path), item.root, now, now, expected, item.failure})
	}
	path := filepath.Join(root, "projector-proof.json")
	mustWrite(t, path, jsonBytes(proof))
	p.Qualification = bound(t, path)
	m.Outputs.Projectors = []Projector{p}
	m.Outputs.Rules = []OutputRule{{"out", ":fixture", "exact"}, {"out/result*.txt", ":fixture", p.Identity}}
	r := runFixture(t, m)
	if err := r.execute(1, 0, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 2 || result.Decision != "FIXTURE_VERIFIED" {
		t.Fatal("qualified normalizer did not preserve equivalent output")
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
	// Keep the valid proof immutable; a separately named adversarial artifact
	// binds a constant projector to the same independent semantic expectations.
	bad := p
	bad.Arguments = append(append([]string{}, p.Arguments...), "constant")
	proof.Arguments = bad.Arguments
	badPath := filepath.Join(root, "constant-projector-proof.json")
	mustWrite(t, badPath, jsonBytes(proof))
	bad.Qualification = bound(t, badPath)
	if err = checkProjectorProof(bad, true); err == nil {
		t.Fatal("constant projector erased a semantic change")
	} else {
		t.Logf("expected semantic counterexample refusal: %s", err)
	}
}
