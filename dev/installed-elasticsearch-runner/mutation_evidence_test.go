package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMutationEvidenceRequiresActualInvalidation(t *testing.T) {
	before := mutationObservation{Outcome: "FROM-CACHE", Process: processRecord{Started: true, Outcome: "SUCCESS", ExitCode: 0}, Cache: taskSnapshot{Hash: "before", Class: "owner", Loader: "loader", Actions: []string{"action"}}}
	after := before
	after.Outcome, after.Cache.Hash = "EXECUTED", "after"
	if err := checkMutationTransition("rules", before, after); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*mutationObservation){
		func(o *mutationObservation) { o.Outcome = "FROM-CACHE" },
		func(o *mutationObservation) { o.Cache.Hash = "before" },
		func(o *mutationObservation) { o.Cache.Loader = "different implementation" },
		func(o *mutationObservation) { o.Process.Outcome = "TIMEOUT" },
	} {
		bad := after
		change(&bad)
		if checkMutationTransition("rules", before, bad) == nil {
			t.Fatal("invalid mutation transition accepted")
		}
	}
	after.Outcome = "NO-SOURCE"
	if err := checkMutationTransition("all-source-removal", before, after); err != nil {
		t.Fatal(err)
	}
	after.Outcome, after.Process.Outcome, after.Process.ExitCode = "FAILED", "FAILURE", 1
	after.UTF8Failure = true
	if err := checkMutationTransition("malformed-utf8", before, after); err != nil {
		t.Fatal(err)
	}
	after.UTF8Failure = false
	if checkMutationTransition("malformed-utf8", before, after) == nil {
		t.Fatal("unrelated failure accepted")
	}
}

func TestMutationSnapshotUsesParentTaskIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	raw := `{"id":1,"startTime":1,"detailsClassName":"ExecuteTask","details":{"buildPath":":","taskPath":":forbiddenPatterns"}}
{"id":2,"parentId":1,"startTime":2}
{"id":3,"parentId":2,"startTime":3}
{"id":3,"endTime":4,"resultClassName":"org.gradle.api.internal.tasks.SnapshotTaskInputsBuildOperationResult","result":{"hash":"key","implementationClassName":"owner","classLoaderHash":"loader","actionClassLoaderHashes":["action"],"inputValueHashes":{"rules":"ruleshash"}}}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	snapshots, err := mutationSnapshots(path)
	if err != nil || snapshots[":forbiddenPatterns"].Hash != "key" {
		t.Fatalf("snapshot ownership: %+v %v", snapshots, err)
	}
	if err = os.WriteFile(path, []byte(raw+`{"id":3,"endTime":5,"resultClassName":"SnapshotTaskInputs","result":{"hash":"other"}}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = mutationSnapshots(path); err == nil {
		t.Fatal("duplicate snapshot accepted")
	}
}
