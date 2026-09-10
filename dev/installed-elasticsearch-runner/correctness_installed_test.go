package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This opt-in runs exactly two local fixture builds under the caller's owned
// service. It has no Elasticsearch or performance authority.
func TestCorrectnessInstalledCapture(t *testing.T) {
	root := os.Getenv("EIC_CORRECTNESS_PROBE_ROOT")
	if root == "" {
		t.Skip("requires a separately allocated two-build local fixture")
	}
	source, jdk, dist, pkg := os.Getenv("EIC_CORRECTNESS_SOURCE"), os.Getenv("EIC_CORRECTNESS_JDK"), os.Getenv("EIC_CORRECTNESS_DIST"), os.Getenv("EIC_CORRECTNESS_PACKAGE")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	packageSHA, err := hashFile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := startCorrectnessBackend(ctx, filepath.Join(root, "backend"), fileBinding{pkg, packageSHA})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := b.close(); err != nil {
			t.Error(err)
		}
	}()
	work := filepath.Join(root, "arms", "W1")
	for _, dir := range []string{work, filepath.Join(work, "server"), filepath.Join(work, "gradle/wrapper"), filepath.Join(root, "runtime"), filepath.Join(root, "state/W1/tmp"), filepath.Join(root, "state/W1/gradle/wrapper/dists")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"gradlew", "gradle/wrapper/gradle-wrapper.jar", "gradle/wrapper/gradle-wrapper.properties"} {
		if err := copyTree(filepath.Join(source, name), filepath.Join(work, name)); err != nil {
			t.Fatal(err)
		}
	}
	copy := exec.Command("/usr/bin/cp", "-a", "--reflink=auto", dist, filepath.Join(root, "state/W1/gradle/wrapper/dists"))
	if raw, err := copy.CombinedOutput(); err != nil {
		t.Fatalf("distribution copy: %s %v", raw, err)
	}
	files := map[string]string{
		"settings.gradle": "rootProject.name = 'owned-correctness-probe'\ninclude 'server'\n",
		"build.gradle": `project(':server') {
    tasks.register('forbiddenPatterns') {
        doLast {
            assert System.getenv().keySet().findAll { it.startsWith('BUILDOPT_') || it.startsWith('WCNCP_') }.isEmpty()
            if (file('fail').exists()) { throw new GradleException('owned-native-failure') }
        }
    }
    tasks.register('precommit') { dependsOn 'forbiddenPatterns' }
}
`,
	}
	for name, raw := range files {
		if err := writeNew(filepath.Join(work, name), []byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.install(ctx, work); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(filepath.Join(root, "runtime/correctness-diagnostic.init.gradle"), []byte(correctnessDiagnosticScript)); err != nil {
		t.Fatal(err)
	}
	f := correctnessRuntime{Root: root, Runner: fileBinding{"/local-probe", strings.Repeat("a", 64)}, Package: b.pkg, BackendURL: b.URL, CA: fileBinding{Path: b.CA}, Credential: b.Credential}
	freeze := map[string]any{"schemaVersion": "buildopt.eic/correctness-installed-probe/v1", "maximumStarts": 2, "maximumSeconds": 180, "maximumBytes": 1 << 30, "package": b.pkg, "diagnosticSha256": digest([]byte(correctnessDiagnosticScript)), "publicAuthority": false}
	if err := writeJSON(filepath.Join(root, "freeze.json"), freeze); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"C005", "C010"} {
		step := correctnessStepForTest(t, id)
		r, err := correctnessRequestFor(f, step)
		if err != nil {
			t.Fatal(err)
		}
		attempt := filepath.Join(root, "attempts", id)
		if err := os.MkdirAll(attempt, 0700); err != nil {
			t.Fatal(err)
		}
		if id == "C010" {
			if err := writeNew(filepath.Join(work, "server/fail"), []byte("fail\n")); err != nil {
				t.Fatal(err)
			}
		}
		for i, env := range r.Environment {
			switch {
			case strings.HasPrefix(env, "JAVA_HOME="):
				r.Environment[i] = "JAVA_HOME=" + jdk
			case strings.HasPrefix(env, "RUNTIME_JAVA_HOME="):
				r.Environment[i] = "RUNTIME_JAVA_HOME=" + jdk
			case strings.HasPrefix(env, "PATH="):
				r.Environment[i] = "PATH=" + filepath.Join(jdk, "bin") + ":/usr/bin:/bin"
			}
		}
		r.Arguments = append(r.Arguments, "--offline")
		if err := writeJSON(filepath.Join(attempt, "request.json"), r); err != nil {
			t.Fatal(err)
		}
		env, err := correctnessPrivateEnvironment(f, r.Environment)
		if err != nil {
			t.Fatal(err)
		}
		p, err := runConfiguredProcess(ctx, attempt, r.Arguments[0], r.Arguments[1:], r.Directory, env, 75*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if !p.Started || p.Signal != 0 || (id == "C005" && p.Outcome != "SUCCESS") || (id == "C010" && (p.Outcome != "FAILURE" || p.ExitCode != 1)) {
			t.Fatalf("native outcome: %+v", p)
		}
		graph, err := os.ReadFile(filepath.Join(attempt, "task-graph.jsonl"))
		if err != nil || !strings.Contains(string(graph), ":server:forbiddenPatterns") {
			t.Fatalf("graph missing: %v", err)
		}
		if err := verifyCorrectnessSupervision(attempt, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.close(); err != nil {
		t.Fatal(err)
	}
	posts := b.observations()
	if err := writeJSON(filepath.Join(root, "observations.json"), posts); err != nil {
		t.Fatal(err)
	}
	for _, post := range posts {
		for _, fact := range post.Facts {
			if fact.Authority.ProspectiveGateInput || fact.Duration.Classification == "CONTROLLED_VALUE_INPUT" {
				t.Fatal("functional probe acquired value authority")
			}
		}
	}
	if err := writeJSON(filepath.Join(root, "verified.json"), map[string]any{"gradleStarts": 2, "installedGraphVerified": true, "nativeFailurePreserved": true, "supervisionVerified": true, "publicAuthority": false}); err != nil {
		t.Fatal(err)
	}
}

func TestCorrectnessSupervisionRejectsInvalidBoundary(t *testing.T) {
	dir := t.TempDir()
	p := processRecord{Started: true, PID: 1, StartNS: 100, EndNS: 1000, ExitCode: 0, Outcome: "SUCCESS"}
	valid := correctnessSupervision{Schema: "buildopt.eic/native-supervision/v1", Started: true, PID: 2, Start: 110, End: 980, Duration: 800, Exit: 0, Cgroup: "0::/user.slice/probe.service", Diagnostic: true}
	for _, mutate := range []func(*correctnessSupervision){func(*correctnessSupervision) {}, func(v *correctnessSupervision) { v.Start = 90 }, func(v *correctnessSupervision) { v.End = 1001 }, func(v *correctnessSupervision) { v.Duration = 999 }, func(v *correctnessSupervision) { v.Exit = 1 }, func(v *correctnessSupervision) { v.Diagnostic = false }} {
		v := valid
		mutate(&v)
		raw, _ := json.Marshal(v)
		if err := os.WriteFile(filepath.Join(dir, "native-supervision.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		err := verifyCorrectnessSupervision(dir, p)
		if (v == valid) != (err == nil) {
			t.Fatalf("boundary accepted incorrectly: %+v %v", v, err)
		}
	}
}
