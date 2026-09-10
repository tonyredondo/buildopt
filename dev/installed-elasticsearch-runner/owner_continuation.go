package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

const ownerContinuationSchema = "buildopt.eic/owner-test-composite-continuation/v1"

func loadOwnerPartial(binding fileBinding) (ownerTestFreeze, ownerTestEvidence, error) {
	var receipt ownerTestReceipt
	var f ownerTestFreeze
	var result ownerTestEvidence
	if err := checkBinding(binding); err != nil {
		return f, result, err
	}
	if err := readJSONLimit(binding.Path, &receipt, 16<<20); err != nil {
		return f, result, err
	}
	var err error
	f, err = loadOwnerTestFreeze(receipt.Freeze)
	if err != nil {
		return f, result, err
	}
	if f.Schema != ownerCompositeSchema || receipt.Schema != "buildopt.eic/owner-test-receipt/v1" || binding.Path != filepath.Join(f.Runtime.EvidenceRoot, "receipt.json") {
		return f, result, errors.New("original composite partial receipt required")
	}
	actual, err := inventory(f.Runtime.EvidenceRoot, []string{"freeze.json", "run-start.json", "run-result.json", "attempts"})
	if err != nil || !reflect.DeepEqual(actual, receipt.Artifacts) {
		return f, result, errors.New("partial owner evidence changed")
	}
	entries, err := os.ReadDir(filepath.Join(f.Runtime.EvidenceRoot, "attempts"))
	if err != nil || len(entries) != 1 || entries[0].Name() != f.Steps[0].Row.ID {
		return f, result, errors.New("continuation requires exactly the first original owner capture")
	}
	var outcome struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := readJSON(filepath.Join(f.Runtime.EvidenceRoot, "run-result.json"), &outcome); err != nil {
		return f, result, err
	}
	if outcome.Status != "INCOMPLETE" || outcome.Error != "invalid or duplicate task graph document" {
		return f, result, errors.New("continuation is limited to the diagnosed graph-reader failure")
	}
	result, err = inspectOwnerTest(f, f.Steps[0])
	return f, result, err
}

func ownerPriorEvidence(f ownerTestFreeze) ([]ownerTestEvidence, error) {
	if f.Continuation == nil {
		return []ownerTestEvidence{}, nil
	}
	old, result, err := loadOwnerPartial(*f.Continuation)
	if err != nil {
		return nil, err
	}
	if old.Runtime.Root != f.Runtime.Root || old.Runtime.BootID != f.Runtime.BootID || old.Correctness != f.Correctness || old.Mutations != f.Mutations || !reflect.DeepEqual(old.CompositeDecision, f.CompositeDecision) || !reflect.DeepEqual(old.Definitions, f.Definitions) || !reflect.DeepEqual(old.JDK, f.JDK) || result.Process.EndNS > f.Runtime.StartNS {
		return nil, errors.New("owner continuation parent differs")
	}
	return []ownerTestEvidence{result}, nil
}

// Reuse the complete raw unit execution after independently repairing its
// reader. Keep the failed original receipt immutable and start only T002-T004.
func freezeOwnerContinuation(ctx context.Context, partial fileBinding) (fileBinding, error) {
	var binding fileBinding
	f, _, err := loadOwnerPartial(partial)
	if err != nil {
		return binding, err
	}
	oldRoot := f.Runtime.EvidenceRoot
	f.Schema = ownerContinuationSchema
	f.Continuation = &partial
	f.MaxOuter = 3
	f.Runtime.EvidenceRoot = filepath.Join(f.Runtime.Root, "owner-tests-composite-continuation-v1")
	self, err := ownExecutable()
	if err != nil {
		return binding, err
	}
	sha, err := hashFile(self)
	if err != nil {
		return binding, err
	}
	f.Runtime.Runner = fileBinding{self, sha}
	boot, err := bootID()
	if err != nil || boot != f.Runtime.BootID {
		return binding, errors.New("owner continuation boot changed")
	}
	f.Runtime.StartNS, err = bootNow()
	if err != nil {
		return binding, err
	}
	f.Runtime.DeadlineNS = f.Runtime.StartNS + int64(2700*time.Second)
	if _, err := ownerPriorEvidence(f); err != nil {
		return binding, err
	}
	for _, arm := range []string{"N0", "N1"} {
		step := f.Steps[0]
		step.Row.Arm = arm
		if _, err := verifyCorrectnessSource(ctx, f.Runtime, step, step.InputBefore); err != nil {
			return binding, err
		}
		for _, rel := range []string{"build-tools-internal/build/test-results/test", "build-tools-internal/build/test-results/integTest", "build-tools-internal/build/tmp/integTest/.gradle-test-kit"} {
			if arm == "N0" && rel == "build-tools-internal/build/test-results/test" {
				live, err := inventory(filepath.Join(f.Runtime.Root, "arms/N0"), []string{rel})
				if err != nil {
					return binding, err
				}
				retained, err := inventory(filepath.Join(oldRoot, "attempts/T001/outputs"), []string{rel})
				if err != nil || !reflect.DeepEqual(live, retained) {
					return binding, errors.New("completed unit outputs changed before continuation")
				}
			} else if _, err := os.Lstat(filepath.Join(f.Runtime.Root, "arms", arm, rel)); !os.IsNotExist(err) {
				return binding, errors.New("unstarted owner test outputs already exist")
			}
		}
	}
	if err := os.Mkdir(f.Runtime.EvidenceRoot, 0700); err != nil {
		return binding, err
	}
	if err := os.Mkdir(filepath.Join(f.Runtime.EvidenceRoot, "attempts"), 0700); err != nil {
		return binding, err
	}
	binding.Path = filepath.Join(f.Runtime.EvidenceRoot, "freeze.json")
	if err := writeJSON(binding.Path, f); err != nil {
		return binding, err
	}
	binding.SHA256, err = hashFile(binding.Path)
	return binding, err
}
