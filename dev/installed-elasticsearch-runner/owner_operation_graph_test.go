package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnerOperationGraphIncludesLateCompositeTasks(t *testing.T) {
	raw := `{"id":1,"resultClassName":"` + ownerTaskPlanResult + `","result":{"taskPlan":[]}}
{"id":2,"resultClassName":"` + ownerTaskPlanResult + `","result":{"taskPlan":[{"task":{"nodeType":"TASK","buildPath":":logic","taskId":7,"taskPath":":test"},"nodeIdentity":{"nodeType":"TASK","buildPath":":logic","taskId":7,"taskPath":":test"},"nodeDependencies":[]}]}}
{"id":3,"startTime":100,"detailsClassName":"` + ownerTaskStartDetails + `","details":{"buildPath":":logic","taskPath":":test","taskId":7,"taskClass":"Test"}}
{"id":3,"endTime":120,"resultClassName":"org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationResult","result":{"actionable":true}}
`
	for _, item := range []struct {
		name, raw string
		valid     bool
	}{
		{"late-task", raw, true},
		{"missing-plan", strings.Join(strings.Split(raw, "\n")[2:], "\n"), false},
		{"changed-runtime-id", strings.Replace(raw, `"taskId":7,"taskClass"`, `"taskId":8,"taskClass"`, 1), false},
		{"missing-end", strings.Join(strings.Split(raw, "\n")[:3], "\n") + "\n", false},
		{"duplicate-plan", raw + strings.Split(raw, "\n")[1] + "\n", false},
		{"unknown-node", strings.ReplaceAll(raw, `"nodeType":"TASK"`, `"nodeType":"UNKNOWN"`), false},
		{"missing-dependency", strings.Replace(raw, `"nodeDependencies":[]`, `"nodeDependencies":[{"nodeType":"TASK","buildPath":":logic","taskId":8,"taskPath":":missing"}]`, 1), false},
	} {
		t.Run(item.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "operations-log.txt")
			if err := os.WriteFile(path, []byte(item.raw), 0600); err != nil {
				t.Fatal(err)
			}
			r, err := analyzeOwnerOperations(path)
			if (err == nil) != item.valid {
				t.Fatalf("accepted=%t: %v", err == nil, err)
			}
			if item.valid && (r.Summary.TaskCount != 1 || r.Tasks[0].Identity != ":logic:test" || r.Tasks[0].Outcome != "EXECUTED") {
				t.Fatal(r)
			}
		})
	}
}

func TestOwnerRetainedCompositeOperations(t *testing.T) {
	path := os.Getenv("EIC_OWNER_RETAINED_FREEZE")
	if path == "" {
		t.Skip("requires a retained composite owner capture")
	}
	pin := os.Getenv("EIC_OWNER_RETAINED_FREEZE_SHA256")
	f, err := loadOwnerTestFreeze(fileBinding{path, pin})
	if err != nil {
		t.Fatal(err)
	}
	r, err := inspectOwnerTest(f, f.Steps[0])
	if err != nil {
		t.Fatal(err)
	}
	if r.TestCount != 20 || r.Nested.Starts != 0 {
		t.Fatal(r)
	}
	if output := os.Getenv("EIC_OWNER_RETAINED_RESULT"); output != "" {
		if err := writeJSON(output, r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOwnerRetainedContinuationKeepsParentAndChronology(t *testing.T) {
	path := os.Getenv("EIC_OWNER_RETAINED_PARTIAL")
	if path == "" {
		t.Skip("requires a retained partial owner receipt")
	}
	parent := fileBinding{path, os.Getenv("EIC_OWNER_RETAINED_PARTIAL_SHA256")}
	old, _, err := loadOwnerPartial(parent)
	if err != nil {
		t.Fatal(err)
	}
	f := old
	f.Schema = ownerContinuationSchema
	f.Continuation = &parent
	f.Runtime.StartNS, err = bootNow()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*ownerTestFreeze)
		valid  bool
	}{
		{"valid", func(*ownerTestFreeze) {}, true},
		{"different-c-parent", func(v *ownerTestFreeze) { v.Correctness.SHA256 = strings.Repeat("a", 64) }, false},
		{"different-boot", func(v *ownerTestFreeze) { v.Runtime.BootID = "different" }, false},
		{"overlap", func(v *ownerTestFreeze) { v.Runtime.StartNS = old.Runtime.StartNS }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := f
			test.change(&candidate)
			rows, err := ownerPriorEvidence(candidate)
			if (err == nil) != test.valid {
				t.Fatalf("accepted=%t error=%v", err == nil, err)
			}
			if test.valid && (len(rows) != 1 || rows[0].TestCount != 20 || rows[0].RowID != "T001") {
				t.Fatal(rows)
			}
		})
	}
}
