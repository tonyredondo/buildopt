package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

func TestAcceptedEligibilityUsesOuterTimeAndPreservesFrozenGate(t *testing.T) {
	for _, tt := range []struct {
		ms, outer int64
		outcome   string
		pass      bool
	}{{500, 25000000000, "EXECUTED", true}, {500, 25000000001, "EXECUTED", false}, {499, 1000000000, "EXECUTED", false}, {3000, 40000000000, "FROM-CACHE", false}} {
		report := gradlecriticalpath.Report{Summary: gradlecriticalpath.Summary{MainBuildCriticalPathMs: 10000}, Tasks: []gradlecriticalpath.Task{{Identity: ":server:forbiddenPatterns", Outcome: tt.outcome, DurationMs: tt.ms, CriticalPath: false}}}
		r, err := nativeEligibility(report, processRecord{Started: true, Outcome: "SUCCESS", StartNS: 100, EndNS: 100 + tt.outer})
		if err != nil || r.Passed != tt.pass {
			t.Fatalf("%+v: %+v %v", tt, r, err)
		}
		old, err := nativeMateriality(report)
		if err != nil || old.Passed {
			t.Fatal("historical gate changed")
		}
	}
}

func compareFixture(t *testing.T, manifest, outcome string) (fileBinding, dateOutputContract) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "arms/N0")
	dir := filepath.Join(root, "attempts/CD001")
	for _, p := range []string{dir, filepath.Join(source, "server/build/distributions"), filepath.Join(source, "server/build/markers")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	jarPath := "server/build/distributions/result.jar"
	if err := os.WriteFile(filepath.Join(source, jarPath), projectionJar(t, manifest, "x/A.class", "class bytes", 0644), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "server/build/markers/forbiddenPatterns"), []byte("done"), 0644); err != nil {
		t.Fatal(err)
	}
	graph := `{"schemaVersion":"buildopt.diagnostics/gradle-task-graph/v1","buildPath":":","tasks":[{"identity":":server:forbiddenPatterns","path":":server:forbiddenPatterns","taskClass":"ForbiddenPatternsTask","dependencies":[],"outputs":["server/build/markers/forbiddenPatterns"]},{"identity":":server:jar","path":":server:jar","taskClass":"Jar","dependencies":[],"outputs":["server/build/distributions/result.jar"]},{"identity":":server:precommit","path":":server:precommit","taskClass":"PrecommitTask","dependencies":[":server:forbiddenPatterns",":server:jar"],"outputs":[]}]}`
	var log bytes.Buffer
	enc := json.NewEncoder(&log)
	enc.Encode(map[string]any{"id": 99, "startTime": manifestWindow.StartMs})
	for i, task := range []struct{ name, class string }{{":server:forbiddenPatterns", "ForbiddenPatternsTask"}, {":server:jar", "Jar"}, {":server:precommit", "PrecommitTask"}} {
		enc.Encode(map[string]any{"id": i + 1, "startTime": manifestWindow.StartMs + int64(i)*700, "detailsClassName": "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails", "details": map[string]string{"buildPath": ":", "taskPath": task.name, "taskClass": task.class}})
		result := map[string]any{"actionable": true}
		if i == 1 && outcome != "EXECUTED" {
			result["skipMessage"] = outcome
		}
		enc.Encode(map[string]any{"id": i + 1, "endTime": manifestWindow.StartMs + int64(i)*700 + 600, "resultClassName": "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationResult", "result": result})
	}
	enc.Encode(map[string]any{"id": 99, "endTime": manifestWindow.EndMs})
	for name, data := range map[string][]byte{"operations-log.txt": log.Bytes(), "task-graph.jsonl": []byte(graph + "\n"), "stdout.log": []byte("fixture"), "stderr.log": {}} {
		if err := writeNew(filepath.Join(dir, name), data); err != nil {
			t.Fatal(err)
		}
	}
	p := processRecord{Started: true, Outcome: "SUCCESS", StartNS: 1, EndNS: 10000000001}
	for name, value := range map[string]any{"process.json": p, "native-request.json": nativeRequest{Row: row{ID: "CD001", Revision: makeProtocol().Revisions[0]}}, "native-result.json": nativeResult{RowID: "CD001", InputState: "VERIFIED", Process: p}} {
		if err := writeJSON(filepath.Join(dir, name), value); err != nil {
			t.Fatal(err)
		}
	}
	if err := retainNativeDiagnostic(root, "CD001"); err != nil {
		t.Fatal(err)
	}
	pin, err := hashFile(filepath.Join(dir, "diagnostic-complete.json"))
	if err != nil {
		t.Fatal(err)
	}
	contract := dateOutputContract{Allowlist: []dateOutputRule{{Path: jarPath, Producer: ":server:jar", AllowedMembers: []string{"META-INF/MANIFEST.MF"}, ManifestProvenance: []manifestSource{{Member: "META-INF/MANIFEST.MF", SourceJar: jarPath, SourceProducer: ":server:jar"}}}}}
	return fileBinding{Path: filepath.Join(dir, "diagnostic-complete.json"), SHA256: pin}, contract
}

func TestNativeComparisonVerifiesCapturedOutputsAndEligibility(t *testing.T) {
	a, c := compareFixture(t, firstManifest, "EXECUTED")
	b, _ := compareFixture(t, secondManifest(), "EXECUTED")
	r, err := compareNativeOutputs(a, b, c)
	if err != nil || !r.Passed || r.RawEqual || len(r.DateDifferences) != 1 || !r.Eligibility[0].Passed || r.ValueAdmitted {
		t.Fatalf("%+v %v", r, err)
	}
	original, err := os.ReadFile(filepath.Join(filepath.Dir(b.Path), "outputs/server/build/distributions/result.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, projectionJar(t, secondManifest(), "x/A.class", "class bytes", 0644)) {
		t.Fatal("comparison rewrote capture")
	}
	for _, tamper := range []string{"bytes", "extra", "mode", "receipt"} {
		t.Run(tamper, func(t *testing.T) {
			x, _ := compareFixture(t, secondManifest(), "EXECUTED")
			p := filepath.Join(filepath.Dir(x.Path), "outputs/server/build/markers/forbiddenPatterns")
			switch tamper {
			case "bytes":
				os.WriteFile(p, []byte("tampered"), 0644)
			case "extra":
				os.WriteFile(filepath.Join(filepath.Dir(x.Path), "outputs/server/build/distributions/extra.jar"), []byte("unexpected"), 0644)
			case "mode":
				os.Chmod(p, 0755)
			case "receipt":
				x.SHA256 = strings.Repeat("0", 64)
			}
			if _, err := compareNativeOutputs(a, x, c); err == nil {
				t.Fatal("changed retained evidence accepted")
			}
		})
	}
}

func TestNativeComparisonRejectsChangedCachedOrUnapprovedOutputs(t *testing.T) {
	a, c := compareFixture(t, firstManifest, "EXECUTED")
	for _, outcome := range []string{"FROM-CACHE", "UP-TO-DATE"} {
		b, _ := compareFixture(t, secondManifest(), outcome)
		if _, err := compareNativeOutputs(a, b, c); err == nil {
			t.Fatal("changed cached dates accepted")
		}
	}
	b, _ := compareFixture(t, strings.Replace(secondManifest(), "Version: 1\r", "Version: 2\r", 1), "EXECUTED")
	if _, err := compareNativeOutputs(a, b, c); err == nil {
		t.Fatal("changed application metadata accepted")
	}
	b, _ = compareFixture(t, secondManifest(), "EXECUTED")
	c.Allowlist = nil
	if _, err := compareNativeOutputs(a, b, c); err == nil {
		t.Fatal("unapproved differing output accepted")
	}
}

func TestNativeCompareCLIRequiresPinsAndCannotAdmitValue(t *testing.T) {
	a, _ := compareFixture(t, firstManifest, "EXECUTED")
	b, _ := compareFixture(t, firstManifest, "EXECUTED")
	args := []string{"native-compare", "--first", a.Path, "--first-sha256", a.SHA256, "--second", b.Path, "--second-sha256", b.SHA256}
	var out bytes.Buffer
	if err := runCLI(args, &out); err != nil {
		t.Fatal(err)
	}
	var result nativeComparison
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || !result.RawEqual || !result.Passed || result.ValueAdmitted || result.ProspectiveAdmission {
		t.Fatalf("%+v %v", result, err)
	}
	if result.ContractSHA256 != digest(acceptedDateContract) {
		t.Fatal("accepted contract bytes are not bound")
	}
	args[4] = ""
	if err := runCLI(args, &bytes.Buffer{}); err == nil {
		t.Fatal("missing external pin accepted")
	}
}
