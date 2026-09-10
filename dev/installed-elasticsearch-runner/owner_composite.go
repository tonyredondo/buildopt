package main

import (
	"errors"
	"path/filepath"
	"reflect"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

const ownerCompositeSchema = "buildopt.eic/owner-test-composite-freeze/v1"
const ownerCompositeProposalSHA = "0896a2f4643db749dffd47838923bbc4e859758b03d4ff365bd03e073e9dc9c7"

// Preserve the original standalone profile for historical receipt replay.
// Upstream owner tests use the included build from the repository root.
func ownerCompositeSteps() []correctnessStep {
	steps := ownerTestSteps()
	for i := range steps {
		args := steps[i].Row.Arguments
		steps[i].Row.Arguments = append(append(append([]string{}, args[:4]...), ":build-tools-internal:"+args[6]), args[7:]...)
	}
	return steps
}

func checkOwnerCompositeDecision(binding fileBinding, root string) error {
	if err := checkBinding(binding); err != nil {
		return err
	}
	var decision struct {
		Schema      string      `json:"schemaVersion"`
		Status      string      `json:"status"`
		Proposal    fileBinding `json:"proposal"`
		UTC         string      `json:"utc"`
		Instruction string      `json:"instruction"`
		Scope       string      `json:"scope"`
	}
	if err := readJSON(binding.Path, &decision); err != nil {
		return err
	}
	if decision.Schema != "buildopt.eic/owner-composite-decision/v1" || decision.Status != "ACCEPTED" || decision.Proposal.SHA256 != ownerCompositeProposalSHA || decision.Proposal.Path != filepath.Join(root, "acquisition/owner-composite-invocation-proposal.json") || decision.Instruction == "" {
		return errors.New("accepted composite owner-test proposal required")
	}
	return checkBinding(decision.Proposal)
}

func ownerProfile(f ownerTestFreeze) (string, []correctnessStep, error) {
	switch f.Schema {
	case "buildopt.eic/owner-test-freeze/v1":
		if f.CompositeDecision != nil || f.Continuation != nil {
			return "", nil, errors.New("legacy owner-test profile cannot carry a composite decision")
		}
		return "owner-tests", ownerTestSteps(), nil
	case ownerCompositeSchema, ownerContinuationSchema:
		if f.CompositeDecision == nil {
			return "", nil, errors.New("composite owner-test decision missing")
		}
		if err := checkOwnerCompositeDecision(*f.CompositeDecision, f.Runtime.Root); err != nil {
			return "", nil, err
		}
		if f.Schema == ownerContinuationSchema {
			if f.Continuation == nil {
				return "", nil, errors.New("owner continuation parent missing")
			}
			return "owner-tests-composite-continuation-v1", ownerCompositeSteps(), nil
		}
		if f.Continuation != nil {
			return "", nil, errors.New("original owner profile cannot carry continuation")
		}
		return "owner-tests-composite-v1", ownerCompositeSteps(), nil
	default:
		return "", nil, errors.New("unknown owner-test profile")
	}
}

func ownerRequestFor(f ownerTestFreeze, step correctnessStep) (nativeRequest, error) {
	if f.Schema == "buildopt.eic/owner-test-freeze/v1" {
		return correctnessRequestFor(f.Runtime, step)
	}
	if f.Schema != ownerCompositeSchema && f.Schema != ownerContinuationSchema {
		return nativeRequest{}, errors.New("unknown owner-test profile")
	}
	for i, expected := range ownerCompositeSteps() {
		if !reflect.DeepEqual(step, expected) {
			continue
		}
		legacy := ownerTestSteps()[i]
		r, err := correctnessRequestFor(f.Runtime, legacy)
		if err != nil {
			return r, err
		}
		r.Row = step.Row
		args := append([]string{}, r.Arguments[:3]...)
		args = append(args, step.Row.Arguments...)
		r.Arguments = append(args, r.Arguments[3+len(legacy.Row.Arguments):]...)
		return r, nil
	}
	return nativeRequest{}, errors.New("only exact composite owner-test requests are permitted")
}

func checkOwnerTask(f ownerTestFreeze, name string, tasks []gradlecriticalpath.Task) error {
	build, identity := ":", ":"+name
	if f.Schema == ownerCompositeSchema || f.Schema == ownerContinuationSchema {
		build = ":build-tools-internal"
		identity = build + ":" + name
	}
	count := 0
	for _, task := range tasks {
		if task.Identity != identity {
			continue
		}
		count++
		if task.BuildPath != build || task.Path != ":"+name || task.Outcome != "EXECUTED" {
			return errors.New("owner test task identity or execution mismatch")
		}
	}
	if count != 1 {
		return errors.New("owner test task missing or duplicated in actual graph")
	}
	return nil
}
