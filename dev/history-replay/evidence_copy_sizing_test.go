//go:build linux && amd64 && replay_integration && replay_copy

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// An explicitly selected engineering diagnostic, never a product value test.
func TestRetainedOutputCopySizing(t *testing.T) {
	manifest := os.Getenv("BUILDOPT_REPLAY_COPY_SIZING_MANIFEST")
	if !filepath.IsAbs(manifest) {
		t.Fatal("copy sizing requires an explicit frozen input")
	}
	var plan struct {
		Schema, Root, Scope string
		Files               []Binding
		Order               []string
	}
	if err := readJSON(manifest, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Schema != "bv006-copy-sizing/v1" || len(plan.Files) != 512 || !filepath.IsAbs(plan.Root) || !equalJSON(plan.Order, []string{"serial", "bounded-eight", "bounded-eight", "serial"}) {
		t.Fatal("copy sizing differs from fixed engineering scope")
	}
	for _, file := range plan.Files {
		if err := checkBinding(file); err != nil {
			t.Fatal(err)
		}
		if inside(plan.Root, file.Path) {
			t.Fatal("diagnostic input overlaps output")
		}
	}
	if err := os.Mkdir(plan.Root, 0700); err != nil {
		t.Fatal(err)
	}
	type sample struct {
		Style           string
		Start, End      Stamp
		DurationNS      int64
		Files, Verified int
	}
	results := []sample{}
	for index, style := range plan.Order {
		directory := filepath.Join(plan.Root, fmt.Sprintf("%d-%s", index, style))
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		jobs := []fileCopy{}
		for n, source := range plan.Files {
			jobs = append(jobs, fileCopy{source.Path, filepath.Join(directory, fmt.Sprintf("%06d", n))})
		}
		s := sample{Style: style, Start: stamp(), Files: len(jobs)}
		if style == "serial" {
			for _, job := range jobs {
				if err := copyTree(job.Source, job.Destination); err != nil {
					t.Fatal(err)
				}
			}
		} else if err := copyEvidenceFiles(jobs); err != nil {
			t.Fatal(err)
		}
		if err := syncDir(directory); err != nil {
			t.Fatal(err)
		}
		s.End = stamp()
		s.DurationNS = s.End.NS - s.Start.NS
		for n, job := range jobs {
			if err := checkBinding(plan.Files[n]); err != nil {
				t.Fatal(err)
			}
			if err := checkBinding(Binding{job.Destination, plan.Files[n].SHA256}); err != nil {
				t.Fatal(err)
			}
			left, err := os.Stat(job.Source)
			if err != nil {
				t.Fatal(err)
			}
			right, err := os.Stat(job.Destination)
			if err != nil || left.Mode() != right.Mode() || left.Size() != right.Size() || os.SameFile(left, right) {
				t.Fatalf("output metadata or independent identity differs: %v", err)
			}
			s.Verified++
		}
		results = append(results, s)
	}
	if err := writeExclusive(filepath.Join(plan.Root, "result.json"), jsonBytes(results), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("copy sizing retained: %s", jsonBytes(results))
}
