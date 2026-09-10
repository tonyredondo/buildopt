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

func TestMetadataRATProvenanceRequiresActualPriorBytes(t *testing.T) {
	s := &metadataSession{Reports: map[string][]byte{}}
	raw := []byte(`<rat-report timestamp='2026-09-07T11:22:48+00:00'><resource name='/producer/A'/></rat-report>`)
	task := gradlecriticalpath.Task{Outcome: "FROM-CACHE", StartTimeMs: 1788780168000, EndTimeMs: 1788780168999}
	if _, err := s.projectReport(raw, "/consumer", task); err == nil {
		t.Fatal("unproven cached report accepted")
	}
	task.Outcome = "EXECUTED"
	p, err := s.projectReport(raw, "/producer", task)
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"UP-TO-DATE", "FROM-CACHE"} {
		task.Outcome = outcome
		cached, err := s.projectReport(raw, "/consumer", task)
		if err != nil || !bytes.Equal(cached, p) {
			t.Fatal("prior provenance lost", err)
		}
		changed := bytes.Replace(raw, []byte("/producer/A"), []byte("/consumer/A"), 1)
		if _, err := s.projectReport(changed, "/consumer", task); err == nil {
			t.Fatal("cached path rewrite without raw provenance accepted")
		}
	}
	task.Outcome = "EXECUTED"
	task.StartTimeMs += 1000
	task.EndTimeMs += 1000
	if _, err := s.projectReport(raw, "/producer", task); err == nil {
		t.Fatal("cached bytes bypassed actual execution timestamp")
	}
	task.Outcome = "FAILED"
	if _, err := s.projectReport(raw, "/producer", task); err == nil {
		t.Fatal("failed report accepted")
	}
}

func TestMetadataOwnerAndAllowlistRemainExact(t *testing.T) {
	c, err := loadMetadataContract()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Selectors) != 227 || c.Dates != digest(acceptedDateContract) {
		t.Fatal("approved selector or date drift")
	}
	var graph nativeGraph
	if err := json.Unmarshal([]byte(`{"tasks":[{"identity":":compileJava","taskClass":"JavaCompile","outputs":["build/tmp/compileJava"]}]}`), &graph); err != nil {
		t.Fatal(err)
	}
	capture := comparisonCapture{Graph: graph, Report: gradlecriticalpath.Report{Tasks: []gradlecriticalpath.Task{{Identity: ":compileJava"}}}}
	path := "build/tmp/compileJava/previous-compilation-data.bin"
	if _, err := metadataOwner(capture, path, ":compileJava", "JavaCompile"); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"owner", "class", "output", "duplicate", "report"} {
		t.Run(change, func(t *testing.T) {
			var g nativeGraph
			raw, _ := json.Marshal(graph)
			json.Unmarshal(raw, &g)
			x := capture
			x.Graph = g
			switch change {
			case "owner":
				x.Graph.Tasks[0].Identity = ":other"
			case "class":
				x.Graph.Tasks[0].Class = "Other"
			case "output":
				x.Graph.Tasks[0].Outputs = []string{"another/path"}
			case "duplicate":
				x.Graph.Tasks = append(x.Graph.Tasks, x.Graph.Tasks[0])
			case "report":
				x.Report.Tasks = nil
			}
			if _, err := metadataOwner(x, path, ":compileJava", "JavaCompile"); err == nil {
				t.Fatal("metadata owner drift accepted")
			}
		})
	}
}

func TestMetadataRuntimeRefusesUnqualifiedOrUnapprovedInput(t *testing.T) {
	if s, err := newMetadataSession(nil); err != nil || s != nil {
		t.Fatal("old exact contract changed")
	}
	for _, change := range []string{"schema", "contract", "source", "tests", "decision"} {
		t.Run(change, func(t *testing.T) {
			r := metadataRuntime{Schema: "buildopt.eic/metadata-runtime/v1", ContractSHA: digest(acceptedMetadataContract), SourceSHA: digest(metadataJavaSource), TestsSHA: digest(metadataJavaTests)}
			switch change {
			case "schema":
				r.Schema = "other"
			case "contract":
				r.ContractSHA = strings.Repeat("0", 64)
			case "source":
				r.SourceSHA = strings.Repeat("0", 64)
			case "tests":
				r.TestsSHA = strings.Repeat("0", 64)
			}
			p := filepath.Join(t.TempDir(), "runtime.json")
			if err := writeJSON(p, r); err != nil {
				t.Fatal(err)
			}
			h, _ := hashFile(p)
			if _, err := newMetadataSession(&fileBinding{p, h}); err == nil {
				t.Fatal("unqualified runtime accepted")
			}
		})
	}
}

// Explicit local qualification only; ordinary Go tests need no Java install.
func TestMetadataRetainedIntegration(t *testing.T) {
	path := os.Getenv("BUILDOPT_METADATA_QUALIFICATION_RUNTIME")
	if path == "" {
		t.Skip("retained local qualification was not requested")
	}
	sha, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := newMetadataSession(&fileBinding{path, sha})
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("BUILDOPT_METADATA_RETAINED_ROOT")
	var pins []fileBinding
	for _, id := range []string{"C001", "C003"} {
		p := filepath.Join(root, "attempts", id, "diagnostic-complete.json")
		h, e := hashFile(p)
		if e != nil {
			t.Fatal(e)
		}
		pins = append(pins, fileBinding{p, h})
	}
	c, err := loadAcceptedDateContract()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compareNativeOutputs(pins[0], pins[1], c); err == nil {
		t.Fatal("historical rejection changed")
	}
	r, err := compareOutputs(pins[0], pins[1], c, s)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Passed || r.ValueAdmitted || r.Entries != 50258 || len(r.MetadataDifferences) != 139 || len(r.DateDifferences) != 64 {
		t.Fatalf("unexpected comparison %+v", r)
	}
	t.Logf("50258 entries; 139 qualified metadata differences; 64 original date differences; old rejection retained")
}
