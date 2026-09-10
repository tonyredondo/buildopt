package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

func TestNativeDiagnosticPreservesWorkflowAndOwnsInstrumentation(t *testing.T) {
	root := t.TempDir()
	r, err := makeNativeDiagnosticRequest(root, "/runner", makeProtocol().Rows[3])
	if err != nil {
		t.Fatal(err)
	}
	prefix := append([]string{"/usr/bin/taskset", "--cpu-list", "0-7"}, makeProtocol().Rows[3].Arguments...)
	if !reflect.DeepEqual(r.Arguments[:len(prefix)], prefix) || r.Diagnostic == nil || r.Diagnostic.SHA256 != digest([]byte(nativeDiagnosticScript)) {
		t.Fatalf("workflow or instrumentation drift: %+v", r)
	}
	for _, arg := range r.Arguments {
		if arg == "--rerun-tasks" || arg == "clean" || arg == "--offline" || arg == "--configuration-cache" {
			t.Fatalf("diagnostic changes native state: %s", arg)
		}
	}
	if _, err := makeNativeDiagnosticRequest(root, "/runner", makeProtocol().Rows[5]); err == nil {
		t.Fatal("diagnostic entrypoint admits candidate")
	}
}

func TestNativeDiagnosticSequenceRequiresCompletePreparation(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "attempts"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := validateNativeSequence(root, "D001"); err == nil {
		t.Fatal("diagnostic bypasses preparation")
	}
	for _, r := range makeProtocol().Rows[:3] {
		dir := filepath.Join(root, "attempts", r.ID)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := writeJSON(filepath.Join(dir, "native-result.json"), nativeResult{RowID: r.ID, Process: processRecord{Started: true, ExitCode: 0, Outcome: "SUCCESS"}, InputState: "VERIFIED"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateNativeSequence(root, "D001"); err != nil {
		t.Fatal(err)
	}
	if err := validateNativeSequence(root, "D002"); err == nil {
		t.Fatal("missing first diagnostic")
	}
	dir := filepath.Join(root, "attempts", "D001")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, "native-result.json"), nativeResult{RowID: "D001", Process: processRecord{Started: true, ExitCode: 0, Outcome: "SUCCESS"}, InputState: "VERIFIED"}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, "output-inventory.json"), []entry{}); err != nil {
		t.Fatal(err)
	}
	if err := validateNativeSequence(root, "D002"); err == nil {
		t.Fatal("partial diagnostic retention accepted")
	}
}

func TestNativeGraphInventoriesEveryMainBuildOutput(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "graph.jsonl")
	graph := `{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[{"identity":":server:precommit","path":":server:precommit","taskClass":"PrecommitTask","dependencies":[":server:forbiddenPatterns"],"outputs":[]},{"identity":":server:forbiddenPatterns","path":":server:forbiddenPatterns","taskClass":"ForbiddenPatternsTask","dependencies":[],"outputs":["server/build/markers/forbiddenPatterns"]},{"identity":":server:reports","path":":server:reports","taskClass":"ReportTask","dependencies":[],"outputs":["server/build/reports","server/build/reports/detail.xml"]}]}`
	if err := os.WriteFile(path, []byte(graph+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	selectors, err := nativeOutputSelectors(path)
	if err != nil || !reflect.DeepEqual(selectors, []string{"server/build/markers/forbiddenPatterns", "server/build/reports"}) {
		t.Fatalf("%v %v", selectors, err)
	}
	withSibling := strings.Replace(graph, `"server/build/reports/detail.xml"`, `"server/build/reports-extra","server/build/reports/detail.xml"`, 1)
	if err := os.WriteFile(path, []byte(withSibling+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	selectors, err = nativeOutputSelectors(path)
	if err != nil || !reflect.DeepEqual(selectors, []string{"server/build/markers/forbiddenPatterns", "server/build/reports", "server/build/reports-extra"}) {
		t.Fatalf("interleaved sibling: %v %v", selectors, err)
	}
	for _, bad := range []string{
		`{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[]}`,
		`{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[{"identity":":server:precommit","path":":server:precommit","taskClass":"X","dependencies":[],"outputs":["../outside"]}]}`,
		graph + "\n" + graph,
	} {
		if err := os.WriteFile(path, []byte(bad+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := nativeOutputSelectors(path); err == nil {
			t.Fatal("incomplete/escaping/duplicate graph accepted")
		}
	}
}

func TestNativeMaterialityRequiresActualCriticalWork(t *testing.T) {
	for _, tt := range []struct {
		name, outcome  string
		duration       int64
		critical, pass bool
	}{
		{"below floor", "EXECUTED", 499, true, false},
		{"up to date", "UP_TO_DATE", 700, true, false},
		{"off critical path", "EXECUTED", 700, false, false},
		{"sufficient", "EXECUTED", 500, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			report := gradlecriticalpath.Report{Summary: gradlecriticalpath.Summary{MainBuildCriticalPathMs: 20000}, Tasks: []gradlecriticalpath.Task{{Identity: ":server:forbiddenPatterns", DurationMs: tt.duration, Outcome: tt.outcome, CriticalPath: tt.critical}}}
			result, err := nativeMateriality(report)
			if err != nil || result.Passed != tt.pass {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
	if _, err := nativeMateriality(gradlecriticalpath.Report{}); err == nil {
		t.Fatal("missing task accepted")
	}
}

func TestNativeDiagnosticRealGradleFixture(t *testing.T) {
	gradle, jdk := os.Getenv("EIC_TEST_GRADLE"), os.Getenv("EIC_TEST_JDK")
	if gradle == "" || jdk == "" {
		t.Skip("set EIC_TEST_GRADLE and EIC_TEST_JDK for actual diagnostic integration")
	}
	root := t.TempDir()
	source := filepath.Join(root, "arms", "N0")
	if err := os.MkdirAll(filepath.Join(source, "server"), 0700); err != nil {
		t.Fatal(err)
	}
	for path, raw := range map[string]string{
		"settings.gradle":        "rootProject.name = 'eic-diagnostic-fixture'\ninclude 'server'\n",
		"server/build.gradle":    "tasks.register('forbiddenPatterns') { outputs.file(layout.buildDirectory.file('markers/forbiddenPatterns')); doLast { def f=layout.buildDirectory.file('markers/forbiddenPatterns').get().asFile; f.parentFile.mkdirs(); f.text='done' } }\ntasks.register('precommit') { dependsOn 'forbiddenPatterns' }\n",
		"diagnostic.init.gradle": nativeDiagnosticScript,
	} {
		if err := os.WriteFile(filepath.Join(source, path), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, slot := range []string{"D001", "D002"} {
		dir := filepath.Join(root, "attempts", slot)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, gradle, "--no-daemon", "--offline", "--console=plain", "--max-workers=2", "--build-cache", ":server:precommit", "--init-script", filepath.Join(source, "diagnostic.init.gradle"), "-Dorg.gradle.internal.operations.trace="+filepath.Join(dir, "operations"))
		cmd.Dir = source
		cmd.Env = []string{"PATH=" + filepath.Join(jdk, "bin") + ":/usr/bin:/bin", "JAVA_HOME=" + jdk, "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TZ=UTC", "GRADLE_USER_HOME=" + filepath.Join(root, "gradle-home"), "BUILDOPT_TASK_GRAPH_OUTPUT=" + filepath.Join(dir, "task-graph.jsonl"), "JAVA_TOOL_OPTIONS=-Duser.home=" + filepath.Join(root, "user")}
		if err := writeJSON(filepath.Join(dir, "native-request.json"), map[string]any{"environmentClass": fixtureClass, "arguments": cmd.Args, "environment": cmd.Env}); err != nil {
			t.Fatal(err)
		}
		process, err := runConfiguredProcess(ctx, dir, cmd.Args[0], cmd.Args[1:], source, cmd.Env, 60*time.Second)
		cancel()
		if err != nil || process.Outcome != "SUCCESS" {
			t.Fatalf("%s: %+v: %v", slot, process, err)
		}
		if err = writeJSON(filepath.Join(dir, "native-result.json"), nativeResult{RowID: slot, Process: process, InputState: fixtureClass}); err != nil {
			t.Fatal(err)
		}
		if err = retainNativeDiagnostic(root, slot); err != nil {
			t.Fatal(err)
		}
		if err = verifyNativeDiagnosticComplete(dir); err != nil {
			t.Fatal(err)
		}
		var materiality nativeMaterialityResult
		if err = readJSON(filepath.Join(dir, "materiality.json"), &materiality); err != nil {
			t.Fatal(err)
		}
		if slot == "D002" && (materiality.Outcome != "UP-TO-DATE" || materiality.Passed) {
			t.Fatalf("warm native fixture: %+v", materiality)
		}
		marker, err := os.ReadFile(filepath.Join(dir, "outputs/server/build/markers/forbiddenPatterns"))
		if err != nil || string(marker) != "done" {
			t.Fatalf("%q %v", marker, err)
		}
	}
}
