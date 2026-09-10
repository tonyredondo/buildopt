package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProvenanceQualifiedRuntime(t *testing.T) {
	path := os.Getenv("BUILDOPT_PROVENANCE_RUNTIME")
	if path == "" {
		t.Skip("qualified runtime was not requested")
	}
	h, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r provenanceRuntime
	if err := readJSON(path, &r); err != nil {
		t.Fatal(err)
	}
	s, err := newProvenanceSession(&fileBinding{path, h}, r.Root)
	if err != nil || s == nil || len(s.Origins) != 0 {
		t.Fatal("qualified runtime failed", err)
	}
	for _, change := range []string{"approval", "sources", "checks", "passed", "contract"} {
		t.Run(change, func(t *testing.T) {
			x := r
			dir := t.TempDir()
			target := x.Qualification
			if change == "approval" {
				target = x.Decision
			}
			var obj map[string]any
			if err := readJSON(target.Path, &obj); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "approval":
				obj["ownerMessage"] = "not approved"
			case "sources":
				obj["sources"] = []any{}
			case "checks":
				obj["checks"] = []any{}
			case "passed":
				obj["passed"] = false
			case "contract":
				obj["contractSha256"] = "wrong"
			}
			p := filepath.Join(dir, "tamper.json")
			b, _ := json.Marshal(obj)
			os.WriteFile(p, b, 0600)
			sha, _ := hashFile(p)
			if change == "approval" {
				x.Decision = fileBinding{p, sha}
			} else {
				x.Qualification = fileBinding{p, sha}
			}
			p = filepath.Join(dir, "runtime.json")
			if err := writeJSON(p, x); err != nil {
				t.Fatal(err)
			}
			sha, _ = hashFile(p)
			if _, err := newProvenanceSession(&fileBinding{p, sha}, r.Root); err == nil {
				t.Fatal("tampered qualification or approval accepted")
			}
		})
	}
}

// A retained comparison qualifies the new implementation, never the stopped
// receipt. Fresh campaign admission still requires its own complete C/M/T.
func TestProvenanceRetainedIntegration(t *testing.T) {
	root := os.Getenv("BUILDOPT_PROVENANCE_RETAINED_ROOT")
	if root == "" {
		t.Skip("retained local qualification was not requested")
	}
	receiptPin := fileBinding{filepath.Join(root, "correctness-receipt.json"), "3f41068969ff05bb3f6f974713d04cf198ce050daf97dca06fe63ff9ea426356"}
	if err := checkBinding(receiptPin); err != nil {
		t.Fatal(err)
	}
	var receipt correctnessReceipt
	if err := readJSON(receiptPin.Path, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Stopped != "changed manifest requires an executed producer" || len(receipt.Rows) != 7 {
		t.Fatal("historical rejection changed")
	}
	f, err := loadCorrectnessFreeze(receipt.Freeze)
	if err != nil {
		t.Fatal(err)
	}
	if f.Runtime.Provenance != nil || f.Runtime.Metadata == nil {
		t.Fatal("historical freeze changed")
	}
	metadata, err := newMetadataSession(f.Runtime.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := loadComparisonCaptureMetadata(metadata.Runtime.SeedCapture)
	if err != nil {
		t.Fatal(err)
	}
	s := &provenanceSession{Root: root, Seed: seed}
	for _, index := range []int{0, 2, 4} {
		row := receipt.Rows[index]
		if err := checkBinding(row.Complete); err != nil {
			t.Fatal(err)
		}
		var files map[string]string
		if err := readJSON(row.Complete.Path, &files); err != nil {
			t.Fatal(err)
		}
		for name, sha := range files {
			if filepath.Base(name) != name {
				t.Fatal("unsafe origin binding")
			}
			if err := checkBinding(fileBinding{filepath.Join(filepath.Dir(row.Complete.Path), name), sha}); err != nil {
				t.Fatal(err)
			}
		}
		pin := fileBinding{filepath.Join(filepath.Dir(row.Complete.Path), "diagnostic-complete.json"), files["diagnostic-complete.json"]}
		if err := s.rememberVerified(pin); err != nil {
			t.Fatal(err)
		}
	}
	// Prove the exact seeded report as well as the executed C007/C008 pair.
	if _, origin, err := newProvenanceView(s).checkstyle(s.Origins[0]); err != nil || origin != seed.Binding {
		t.Fatal("seeded Checkstyle provenance failed", err)
	}
	var pins []fileBinding
	for _, id := range []string{"C007", "C008"} {
		p := filepath.Join(root, "attempts", id, "diagnostic-complete.json")
		h, err := hashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		pins = append(pins, fileBinding{p, h})
	}
	contract, err := loadAcceptedDateContract()
	if err != nil {
		t.Fatal(err)
	}
	result, err := compareOutputsWithProvenance(pins[0], pins[1], contract, metadata, s)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.ValueAdmitted || result.Entries != 50258 || len(result.MetadataDifferences) != 138 || len(result.DateDifferences) != 64 || len(result.ProvenanceDifferences) != 1 || len(result.Origins) != 130 {
		t.Fatalf("unexpected retained comparison %+v", result)
	}
	for _, origin := range result.Origins {
		if origin.Path == checkstyleOutputPath {
			if origin.Capture != origin.Origin {
				t.Fatal("executed Checkstyle lost own origin")
			}
			continue
		}
		want := "C001"
		if origin.Capture == pins[1] {
			want = "C005"
		}
		if filepath.Base(filepath.Dir(origin.Origin.Path)) != want {
			t.Fatal("wrong retained date producer", origin)
		}
	}
	if err := checkBinding(receiptPin); err != nil {
		t.Fatal("historical receipt mutated", err)
	}
	if out := os.Getenv("BUILDOPT_PROVENANCE_QUALIFICATION_OUTPUT"); out != "" {
		if err := writeJSON(out, result); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("50258 entries independently rehashed; 64 dates use exact earlier executed artifacts; 138 metadata differences; one Checkstyle prefix-only difference; original C008 rejection preserved")
}
