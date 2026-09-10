//go:build linux && amd64

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func bound(t *testing.T, path string) Binding {
	t.Helper()
	h, err := bindingDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	return Binding{path, h}
}
func testGit(t *testing.T, root string, input []byte, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", root}, args...)...)
	c.Stdin = bytes.NewReader(input)
	c.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Replay fixture", "GIT_AUTHOR_EMAIL=replay@example.invalid", "GIT_COMMITTER_NAME=Replay fixture", "GIT_COMMITTER_EMAIL=replay@example.invalid", "GIT_AUTHOR_DATE=2026-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z")
	b, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture git %v: %v %s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}

// Fixture history uses plumbing in a newly created bare repository. It does
// not stage, commit, or reconfigure anything in the BuildOpt checkout.
func fixtureManifest(t *testing.T, count int) Manifest {
	t.Helper()
	root := t.TempDir()
	if evidenceRoot := os.Getenv("BUILDOPT_REPLAY_TEST_ROOT"); evidenceRoot != "" {
		if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
			t.Fatal(err)
		}
		var err error
		root, err = os.MkdirTemp(evidenceRoot, strings.ReplaceAll(t.Name(), "/", "-")+"-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("retained fixture: %s", root)
	}
	common := filepath.Join(root, "subject.git")
	if err := os.Mkdir(common, 0755); err != nil {
		t.Fatal(err)
	}
	testGit(t, common, nil, "init", "--bare", "--quiet")
	parent := ""
	for i := 0; i < count; i++ {
		blob := testGit(t, common, []byte(fmt.Sprintf("revision %d\n", i)), "hash-object", "-w", "--stdin")
		tree := testGit(t, common, []byte("100644 blob "+blob+"\tsource.txt\n"), "mktree")
		args := []string{"commit-tree", tree}
		if parent != "" {
			args = append(args, "-p", parent)
		}
		parent = testGit(t, common, []byte(fmt.Sprintf("fixture %d\n", i)), args...)
	}
	history, err := reconstructHistory(common, parent, count)
	if err != nil {
		t.Fatal(err)
	}
	protocol, err := filepath.Abs("../../specs/poc-product-viability-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ownedExecutable := filepath.Join(root, "fixture-runner")
	if err = copyTree(self, ownedExecutable); err != nil {
		t.Fatal(err)
	}
	self = ownedExecutable
	post := filepath.Join(root, "patch-after.txt")
	mustWrite(t, post, []byte("candidate implementation\n"))
	return Manifest{ExternalCosts: []Binding{}, Schema: manifestSchema, Phase: "QUALIFICATION", Experiment: "FIXED_NI", Mode: "P", FrozenUTC: time.Now().UTC().Format(time.RFC3339Nano), Protocol: bound(t, protocol), Executable: bound(t, self), Package: []Binding{bound(t, protocol)}, Subject: "local-fixture", CommonGit: common, History: history, RunRoot: filepath.Join(root, "run"), Replications: 1, PrefixEnd: 0, ExecutionEnd: count - 1, DaemonPolicy: "REQUEST", Driver: "FIXTURE", Command: []string{self, "-test.run=TestFixtureChild", "--", "success"}, Environment: map[string]string{"PATH": "/usr/bin:/bin"}, Runtime: []Binding{bound(t, self)}, GeneratedPaths: []string{"out/**", ".gradle/**"}, Acquisition: []Layer{}, Baseline: Patch{Identity: "native", Files: []FilePatch{}, Prerequisites: map[string]string{}}, Candidate: Patch{Identity: "candidate", Prerequisites: map[string]string{}, Files: []FilePatch{{Path: "candidate.txt", BeforeSHA256: "", BeforeMode: 0, After: bound(t, post), AfterMode: 0644}}}, Outputs: OutputPolicy{Rules: []OutputRule{{"out/result.txt", ":fixture", "exact"}}, Projectors: []Projector{}, Diagnostics: []OutputRule{}}, Affinity: "0-1", Limits: Limits{DeadlineUTC: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), MaxRequestNS: int64(5 * time.Second), MaxRunNS: int64(time.Minute), MaxBytes: 64 << 20, MinimumFreeBytes: 1 << 20, MaxWorkflowStarts: 12, MaxGradleStarts: 0, NestedReservePerRequest: 0, RetryPairs: 1}}
}

func TestStrictContract(t *testing.T) {
	m := fixtureManifest(t, 3)
	raw := jsonBytes(m)
	var got Manifest
	if err := decodeStrict(raw, &got); err != nil {
		t.Fatal(err)
	}
	if err := validateManifest(got); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string][]byte{
		"missing":   bytes.Replace(raw, []byte(`"mode": "P",`), nil, 1),
		"unknown":   bytes.Replace(raw, []byte(`"mode": "P",`), []byte(`"mode": "P", "surprise": true,`), 1),
		"duplicate": bytes.Replace(raw, []byte(`"mode": "P",`), []byte(`"mode": "P", "mode": "E",`), 1),
		"null":      bytes.Replace(raw, []byte(`"acquisition": [],`), []byte(`"acquisition": null,`), 1),
		"trailing":  append(append([]byte{}, raw...), []byte("{}")...),
	} {
		t.Run(name, func(t *testing.T) {
			var x Manifest
			if decodeStrict(raw, &x) == nil {
				t.Fatal("accepted malformed contract")
			}
		})
	}
	for _, change := range []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"mode", func(m *Manifest) { m.Mode = "E" }},
		{"adaptive", func(m *Manifest) { m.Experiment = "ADAPTIVE_NFA" }},
		{"version", func(m *Manifest) { m.Schema += "2" }},
		{"unbound-launcher", func(m *Manifest) { m.Command[0] = "/usr/bin/true" }},
		{"ordinal", func(m *Manifest) { m.History[1].Ordinal = 2 }},
		{"parent", func(m *Manifest) { m.History[2].Parent = m.History[0].Commit }},
		{"duplicate", func(m *Manifest) { m.History[2] = m.History[1] }},
		{"tree", func(m *Manifest) { m.History[1].Tree = strings.Repeat("a", 40) }},
		{"future", func(m *Manifest) {
			m.Acquisition = []Layer{{"dependencies", Binding{filepath.Join(m.RunRoot, "future"), strings.Repeat("a", 64)}, "gradle/caches/modules-2"}}
		}},
		{"shared-cache", func(m *Manifest) { m.Environment["GRADLE_USER_HOME"] = "/tmp/shared" }},
		{"unqualified-normalizer", func(m *Manifest) { m.Outputs.Rules[0].Transform = "drop-differences" }},
		{"warm-retry", func(m *Manifest) { m.DaemonPolicy = "REPLICATION" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			var x Manifest
			if err := decodeStrict(raw, &x); err != nil {
				t.Fatal(err)
			}
			change.mutate(&x)
			if validateManifest(x) == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
}

func TestExactPatchAndSource(t *testing.T) {
	m := fixtureManifest(t, 2)
	repo := filepath.Join(filepath.Dir(m.RunRoot), "owned-worktree")
	testGit(t, m.CommonGit, nil, "worktree", "add", "--quiet", "-b", "fixture-owned", repo, m.History[0].Commit)
	if _, err := verifySource(m.CommonGit, repo, m.History[0].Commit, []Patch{}, m.GeneratedPaths); err != nil {
		t.Fatal(err)
	}
	if err := applyPatch(repo, m.Candidate); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySource(m.CommonGit, repo, m.History[0].Commit, []Patch{m.Candidate}, m.GeneratedPaths); err != nil {
		t.Fatal(err)
	}
	if err := applyPatch(repo, m.Candidate); err != nil {
		t.Fatalf("idempotent owned application failed: %v", err)
	}
	if err := reversePatch(m.CommonGit, repo, m.History[0].Commit, m.Candidate); err != nil {
		t.Fatal(err)
	}
	testGit(t, repo, nil, "switch", "--quiet", "--no-overwrite-ignore", "-c", "fixture-ordinal-1", m.History[1].Commit)
	if _, err := verifySource(m.CommonGit, repo, m.History[1].Commit, []Patch{}, m.GeneratedPaths); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, "out/result.txt"), []byte("generated"))
	if _, err := verifySource(m.CommonGit, repo, m.History[1].Commit, []Patch{}, m.GeneratedPaths); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, "source.txt"), []byte("unapproved edit"))
	if _, err := verifySource(m.CommonGit, repo, m.History[1].Commit, []Patch{}, []string{"**"}); err == nil {
		t.Fatal("generated allowlist hid tracked source drift")
	}
}
