//go:build linux && amd64 && replay_integration && replay_owner

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnerQualificationCannotBorrowDifferentReaders(t *testing.T) {
	path := os.Getenv("BUILDOPT_REPLAY_OWNER_POLICY")
	if !filepath.IsAbs(path) {
		t.Fatal("owner policy qualification requires the frozen C5 reader inventory")
	}
	var policy OwnerPolicy
	if err := readJSON(path, &policy); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	proof := filepath.Join(root, "proof.json")
	// Only the proof-binding contract is under test here. Actual execution,
	// output rejection and native reuse are independently qualified fixtures.
	mustWrite(t, proof, []byte("reader binding fixture; no owner/value claim\n"))
	q := OwnerQualification{Schema: "buildopt.history-replay/owner-qualification/v1", Decision: "OWNER_OUTPUT_POLICY_QUALIFIED", ComparatorSHA256: policy.Comparator.SHA256, ReadersSHA256: ownerReadersDigest(policy), Cases: []Binding{bound(t, proof)}, NativeReuse: []Binding{bound(t, proof)}}
	qualification := filepath.Join(root, "qualification.json")
	mustWrite(t, qualification, jsonBytes(q))
	policy.Qualification = bound(t, qualification)
	ownerPath := filepath.Join(root, "owner.json")
	mustWrite(t, ownerPath, jsonBytes(policy))
	m := Manifest{Phase: "ENGINEERING", Driver: "GRADLE", Outputs: OutputPolicy{Owner: bound(t, ownerPath)}}
	if _, err := readOwnerPolicy(m); err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(root, "changed-reader")
	mustWrite(t, changed, []byte("different compiler-state reader"))
	policy.MetadataClasspath[0] = bound(t, changed)
	mustWrite(t, ownerPath, jsonBytes(policy))
	m.Outputs.Owner = bound(t, ownerPath)
	if _, err := readOwnerPolicy(m); err == nil || !strings.Contains(err.Error(), "qualification is incomplete") {
		t.Fatalf("different reader borrowed old proof: %v", err)
	}
	m.Outputs.Diagnostics = []OutputRule{{Path: "**/reports/**", Producer: ":server:checkstyleMain", Transform: "exact"}}
	if _, err := readOwnerPolicy(m); err == nil || !strings.Contains(err.Error(), "without extra output rules") {
		t.Fatalf("owner contract accepted arbitrary report rules: %v", err)
	}
}
