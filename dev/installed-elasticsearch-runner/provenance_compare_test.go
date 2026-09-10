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

func replayProvenanceFixture(dir string) (gradlecriticalpath.Report, error) {
	return gradlecriticalpath.Analyze(filepath.Join(dir, "operations-log.txt"), filepath.Join(dir, "task-graph.jsonl"), "control")
}

func provenanceFixture(t *testing.T, root, id, outcome string, shift int64) (comparisonCapture, dateOutputRule) {
	t.Helper()
	p, contract := compareFixture(t, firstManifest, outcome)
	dir := filepath.Join(root, "attempts", id)
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Dir(p.Path), dir); err != nil {
		t.Fatal(err)
	}
	var request nativeRequest
	readJSON(filepath.Join(dir, "native-request.json"), &request)
	request.Root = root
	request.Row.ID = id
	request.Directory = filepath.Join(root, "arms/N0")
	var result nativeResult
	readJSON(filepath.Join(dir, "native-result.json"), &result)
	result.RowID = id
	for name, v := range map[string]any{"native-request.json": request, "native-result.json": result} {
		raw, _ := json.Marshal(v)
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if shift != 0 {
		p := filepath.Join(dir, "operations-log.txt")
		raw, _ := os.ReadFile(p)
		var out bytes.Buffer
		for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
			var r map[string]json.RawMessage
			json.Unmarshal(line, &r)
			for _, k := range []string{"startTime", "endTime"} {
				if v, ok := r[k]; ok {
					var n int64
					json.Unmarshal(v, &n)
					r[k], _ = json.Marshal(n + shift)
				}
			}
			b, _ := json.Marshal(r)
			out.Write(b)
			out.WriteByte('\n')
		}
		os.WriteFile(p, out.Bytes(), 0600)
		report, err := replayProvenanceFixture(dir)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(report)
		os.WriteFile(filepath.Join(dir, "critical-path.json"), b, 0600)
	}
	resealCorrectnessDiagnosticFixture(t, dir)
	p.Path = filepath.Join(dir, "diagnostic-complete.json")
	p.SHA256, _ = hashFile(p.Path)
	c, err := loadComparisonCapture(p)
	if err != nil {
		t.Fatal(err)
	}
	return c, contract.Allowlist[0]
}

func TestReusedDateRequiresExactEarlierExecutedArtifact(t *testing.T) {
	for _, outcome := range []string{"UP-TO-DATE", "FROM-CACHE"} {
		t.Run(outcome, func(t *testing.T) {
			root := t.TempDir()
			origin, rule := provenanceFixture(t, root, "C001", "EXECUTED", 0)
			current, _ := provenanceFixture(t, root, "C002", outcome, 10000)
			s := &provenanceSession{Root: root}
			if _, _, err := newProvenanceView(s).date(current, rule); err == nil {
				t.Fatal("missing origin accepted")
			}
			if err := s.rememberVerified(origin.Binding); err != nil {
				t.Fatal(err)
			}
			got, pin, err := newProvenanceView(s).date(current, rule)
			raw, _ := os.ReadFile(filepath.Join(origin.Root, rule.Path))
			want, _ := archiveProjection(raw, rule.AllowedMembers, origin.Window)
			if err != nil || got != want || pin != origin.Binding {
				t.Fatal("valid reused origin lost", err)
			}
			if current.Report.Tasks[1].Outcome == "EXECUTED" {
				t.Fatal("current task relabelled")
			}
		})
	}
}

func TestReusedDateRejectsOriginDrift(t *testing.T) {
	for _, kind := range []string{"future", "same-row", "cross-campaign", "revision", "unexecuted", "bytes", "mode", "origin-bytes", "origin-mode", "owner", "window", "source", "receipt"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			origin, rule := provenanceFixture(t, root, "C001", "EXECUTED", 0)
			current, _ := provenanceFixture(t, root, "C002", "UP-TO-DATE", 10000)
			s := &provenanceSession{Root: root}
			if err := s.rememberVerified(origin.Binding); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "future":
				current.Window.StartMs = origin.Window.StartMs - 1
			case "same-row":
				current.RowID = origin.RowID
				current.Binding.Path = origin.Binding.Path
			case "cross-campaign":
				current.Campaign = t.TempDir()
			case "revision":
				current.Revision = strings.Repeat("a", 40)
			case "unexecuted":
				s.Origins[0].Report.Tasks[1].Outcome = "UP-TO-DATE"
			case "bytes":
				os.WriteFile(filepath.Join(current.Root, rule.Path), []byte("changed"), 0644)
			case "mode":
				os.Chmod(filepath.Join(current.Root, rule.Path), 0755)
			case "origin-bytes":
				os.WriteFile(filepath.Join(origin.Root, rule.Path), []byte("changed"), 0644)
			case "origin-mode":
				os.Chmod(filepath.Join(origin.Root, rule.Path), 0755)
			case "owner":
				current.Graph.Tasks[1].Class = "Other"
			case "window":
				s.Origins[0].Window.StartMs += 10000
				s.Origins[0].Window.EndMs += 10000
			case "source":
				rule.ManifestProvenance[0].SourceJar = "unknown.jar"
			case "receipt":
				os.WriteFile(origin.Binding.Path, []byte("{}"), 0600)
			}
			if _, _, err := newProvenanceView(s).date(current, rule); err == nil {
				t.Fatal("origin drift accepted")
			}
		})
	}
}

func rewriteProvenanceCapture(t *testing.T, c comparisonCapture) comparisonCapture {
	t.Helper()
	dir := filepath.Dir(c.Binding.Path)
	b, _ := json.Marshal(c.Graph)
	if err := os.WriteFile(filepath.Join(dir, "task-graph.jsonl"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := replayProvenanceFixture(dir)
	if err != nil {
		t.Fatal(err)
	}
	selectors, err := nativeOutputSelectors(filepath.Join(dir, "task-graph.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := inventory(c.Root, selectors)
	if err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]any{"critical-path.json": report, "output-inventory.json": items, "output-selectors.json": selectors} {
		raw, _ := json.Marshal(v)
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	resealCorrectnessDiagnosticFixture(t, dir)
	c.Binding.SHA256, _ = hashFile(c.Binding.Path)
	c, err = loadComparisonCapture(c.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReusedDatePreservesCurrentManifestPropagationAndExecutedWindow(t *testing.T) {
	root := t.TempDir()
	origin, rule := provenanceFixture(t, root, "C001", "EXECUTED", 0)
	current, _ := provenanceFixture(t, root, "C002", "UP-TO-DATE", 10000)
	manifestPath := "server/build/tmp/manifest.mf"
	add := func(c comparisonCapture) comparisonCapture {
		p := filepath.Join(c.Root, manifestPath)
		os.MkdirAll(filepath.Dir(p), 0700)
		os.WriteFile(p, []byte(firstManifest), 0644)
		c.Graph.Tasks[1].Outputs = append(c.Graph.Tasks[1].Outputs, manifestPath)
		return rewriteProvenanceCapture(t, c)
	}
	origin = add(origin)
	current = add(current)
	rule.Path = manifestPath
	rule.AllowedMembers = nil
	rule.ManifestProvenance[0].Member = ""
	s := &provenanceSession{Root: root}
	if err := s.rememberVerified(origin.Binding); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newProvenanceView(s).date(current, rule); err != nil {
		t.Fatal("valid propagated manifest", err)
	}
	jar := rule.ManifestProvenance[0].SourceJar
	os.WriteFile(filepath.Join(current.Root, jar), projectionJar(t, secondManifest(), "x/A.class", "class bytes", 0644), 0644)
	current = rewriteProvenanceCapture(t, current)
	if _, _, err := newProvenanceView(s).date(current, rule); err == nil {
		t.Fatal("changed current source manifest accepted")
	}
	executed, executedRule := provenanceFixture(t, root, "C003", "EXECUTED", 20000)
	if _, _, err := newProvenanceView(s).date(executed, executedRule); err == nil {
		t.Fatal("executed output borrowed prior date window")
	}
}

func checkstyleCaptureFixture(t *testing.T, root, id, outcome, producer string, shift int64) comparisonCapture {
	t.Helper()
	c, _ := provenanceFixture(t, root, id, "EXECUTED", shift)
	p := filepath.Join(c.Root, checkstyleOutputPath)
	os.MkdirAll(filepath.Dir(p), 0700)
	os.WriteFile(p, []byte(`<checkstyle version="13.11.0"><file name="`+producer+`/A.java"><error line="7" message="failure"/></file></checkstyle>`), 0644)
	var g nativeGraph
	b, _ := json.Marshal(map[string]any{"tasks": []any{map[string]any{"identity": checkstyleOwner, "path": checkstyleOwner, "taskClass": checkstyleClass, "outputs": []string{checkstyleOutputPath}}}})
	json.Unmarshal(b, &g)
	c.Graph.Tasks = append(c.Graph.Tasks, g.Tasks[0])
	dir := filepath.Dir(c.Binding.Path)
	p = filepath.Join(dir, "operations-log.txt")
	raw, _ := os.ReadFile(p)
	var log bytes.Buffer
	log.Write(raw)
	enc := json.NewEncoder(&log)
	enc.Encode(map[string]any{"id": 90, "startTime": c.Window.StartMs + 50, "detailsClassName": "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationDetails", "details": map[string]string{"buildPath": ":", "taskPath": checkstyleOwner, "taskClass": checkstyleClass}})
	result := map[string]any{"actionable": true}
	if outcome != "EXECUTED" {
		result["skipMessage"] = outcome
	}
	enc.Encode(map[string]any{"id": 90, "endTime": c.Window.StartMs + 60, "resultClassName": "org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationResult", "result": result})
	os.WriteFile(p, log.Bytes(), 0600)
	return rewriteProvenanceCapture(t, c)
}

func TestCheckstyleRequiresExecutedOrExactRetainedProvenance(t *testing.T) {
	for _, outcome := range []string{"UP-TO-DATE", "FROM-CACHE"} {
		root := t.TempDir()
		producer := filepath.Join(root, "arms/N0")
		origin := checkstyleCaptureFixture(t, root, "C001", "EXECUTED", producer, 0)
		current := checkstyleCaptureFixture(t, root, "C002", outcome, producer, 10000)
		s := &provenanceSession{Root: root}
		if _, _, err := newProvenanceView(s).checkstyle(current); err == nil {
			t.Fatal("unproven report accepted")
		}
		if err := s.rememberVerified(origin.Binding); err != nil {
			t.Fatal(err)
		}
		p, pin, err := newProvenanceView(s).checkstyle(current)
		if err != nil || p == "" || pin != origin.Binding {
			t.Fatal("verified report rejected", err)
		}
		path := filepath.Join(origin.Root, checkstyleOutputPath)
		os.Chmod(path, 0755)
		if _, _, err := newProvenanceView(s).checkstyle(current); err == nil {
			t.Fatal("changed origin mode accepted")
		}
	}
	root := t.TempDir()
	c := checkstyleCaptureFixture(t, root, "C001", "EXECUTED", "/wrong-producer", 0)
	if _, _, err := newProvenanceView(&provenanceSession{Root: root}).checkstyle(c); err == nil {
		t.Fatal("wrong executed workspace accepted")
	}
}

func TestProvenanceRuntimePreservesOldContractsAndRejectsMissingApproval(t *testing.T) {
	if s, err := newProvenanceSession(nil, t.TempDir()); err != nil || s != nil {
		t.Fatal("old comparison contract changed")
	}
	for _, change := range []string{"schema", "contract", "root", "seed", "decision"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			r := provenanceRuntime{Schema: "buildopt.eic/reused-output-provenance-runtime/v1", Root: root, ContractSHA: digest(acceptedProvenanceContract), Seed: fileBinding{SHA256: provenanceSeedSHA}}
			switch change {
			case "schema":
				r.Schema = "other"
			case "contract":
				r.ContractSHA = "other"
			case "root":
				r.Root = "/other"
			case "seed":
				r.Seed.SHA256 = "other"
			}
			p := filepath.Join(root, "runtime.json")
			if err := writeJSON(p, r); err != nil {
				t.Fatal(err)
			}
			h, _ := hashFile(p)
			if _, err := newProvenanceSession(&fileBinding{p, h}, root); err == nil {
				t.Fatal("unqualified or unapproved runtime accepted")
			}
		})
	}
}
