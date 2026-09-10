//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func checkStampInterval(start, end Stamp) error {
	if start.Boot == "" || start.Boot != end.Boot || start.NS <= 0 || end.NS < start.NS {
		return errors.New("invalid monotonic clock interval")
	}
	return nil
}

func loadAttempt(m Manifest, b Binding, costs map[string]Cost, sourceHashes map[string]string) (AttemptStart, AttemptEnd, error) {
	var start AttemptStart
	var end AttemptEnd
	if err := checkBinding(b); err != nil {
		return start, end, err
	}
	if !inside(filepath.Join(m.RunRoot, "attempts"), b.Path) || filepath.Base(b.Path) != "end.json" {
		return start, end, errors.New("attempt artifact outside owned evidence")
	}
	dir := filepath.Dir(b.Path)
	if err := readJSON(filepath.Join(dir, "start.json"), &start); err != nil {
		return start, end, err
	}
	if err := readJSON(b.Path, &end); err != nil {
		return start, end, err
	}
	if start.Schema != recordSchema || end.Schema != recordSchema || start.ID != end.ID || filepath.Base(dir) != start.ID || start.Ordinal < 0 || start.Ordinal > m.ExecutionEnd || start.Replication < 1 || start.Replication > m.Replications || (start.Arm != "N" && start.Arm != "I") {
		return start, end, errors.New("attempt identity/ordinal differs")
	}
	manifestHash, err := fileDigest(filepath.Join(m.RunRoot, "manifest.json"))
	if err != nil {
		return start, end, err
	}
	if start.ManifestSHA256 != manifestHash {
		return start, end, errors.New("attempt uses different manifest")
	}
	command, env := workflow(m, start.Replication, start.Arm, dir)
	if !equalJSON(command, start.Command) || !equalJSON(env, start.Environment) {
		return start, end, errors.New("workflow/environment drift")
	}
	if err := checkStampInterval(end.Begin, end.End); err != nil {
		return start, end, err
	}
	if end.DurationNS != end.End.NS-end.Begin.NS || end.DurationNS <= 0 {
		return start, end, errors.New("forged request duration")
	}
	if len(end.Costs) != 1 {
		return start, end, errors.New("request lacks unique envelope cost")
	}
	for _, required := range []struct{ id, purpose string }{{start.ID + "-preflight", "source-input-state-proof"}, {start.ID + "-capture", "complete-output-state-and-invocation-capture"}} {
		if err := requirePhase(costs, required.id, required.purpose, start.Replication, start.Ordinal, start.Arm); err != nil {
			return start, end, err
		}
	}
	cost, ok := costs[end.Costs[0]]
	if !ok || cost.Attempt != start.ID || cost.Class != "customer-machine" || !cost.InsideEnvelope || cost.StartNS != end.Begin.NS || cost.EndNS != end.End.NS || cost.DurationNS != end.DurationNS {
		return start, end, errors.New("hidden or double-counted customer envelope")
	}
	var before, after StatePin
	for _, v := range []struct {
		b   Binding
		out any
	}{{start.Before, &before}, {end.After, &after}} {
		if err := checkBinding(v.b); err != nil {
			return start, end, err
		}
		if err := readJSON(v.b.Path, v.out); err != nil {
			return start, end, err
		}
	}
	for _, s := range []StatePin{before, after} {
		if s.Schema != recordSchema || s.Replication != start.Replication || s.Arm != start.Arm || s.Ordinal != start.Ordinal || s.Revision != m.History[start.Ordinal].Commit || s.Root != filepath.Join(m.RunRoot, fmt.Sprintf("r%d", start.Replication), start.Arm) || s.CommonGit != m.CommonGit || s.SemanticSHA256 != objectDigest(semanticEntries(s.Entries)) {
			return start, end, errors.New("state identity, chronology or digest differs")
		}
		if err := checkRecordedSource(m, s, sourceHashes); err != nil {
			return start, end, err
		}
	}
	if after.Previous != start.Before.SHA256 || after.CandidateApplied != end.CandidateApplied {
		return start, end, errors.New("state lineage/candidate decision differs")
	}
	if start.Ordinal == 0 && start.Generation == 0 {
		if err := checkInitialState(m, before); err != nil {
			return start, end, err
		}
	}
	if start.Arm == "N" && (before.CandidateApplied || after.CandidateApplied) {
		return start, end, errors.New("native arm contains candidate state")
	}
	if start.Arm == "I" && recordedApplicable(before, m.Candidate) != end.CandidateApplied {
		return start, end, errors.New("candidate admission differs from frozen source prerequisites")
	}
	if start.SourceSHA256 != objectDigest(struct {
		Revision string
		Patches  []Patch
	}{before.Revision, patchesFor(m, before)}) {
		return start, end, errors.New("source receipt differs from expected baseline")
	}
	var native ProcessReceipt
	if err := checkBinding(end.Native); err != nil {
		return start, end, err
	}
	if err := readJSON(end.Native.Path, &native); err != nil {
		return start, end, err
	}
	if !absentBinding(m.Observer) {
		for _, purpose := range []string{"observer-start", "observer-close"} {
			if err := requirePhase(costs, start.ID+"-"+purpose, purpose, start.Replication, start.Ordinal, start.Arm); err != nil {
				return start, end, err
			}
		}
		if costs[start.ID+"-observer-start"].EndNS > end.Begin.NS || costs[start.ID+"-observer-close"].StartNS < end.End.NS {
			return start, end, errors.New("observer preparation/drain charged inside customer envelope")
		}
		if err := checkObservation(m, dir, native); err != nil {
			return start, end, err
		}
	}
	if native.Schema != recordSchema || native.Attempt != start.ID || native.Process.PID <= 0 || native.Process.StartTicks == 0 || native.Supervisor.PID <= 0 || native.InvocationID == "" || !strings.HasSuffix(native.Supervisor.Cgroup, "/"+native.Unit) || !strings.HasPrefix(native.Unit, "buildopt-replay-"+digest([]byte(m.RunRoot))[:16]+"-") {
		return start, end, errors.New("missing process ownership")
	}
	if !strings.Contains(native.Unit, fmt.Sprintf("-r%d-%s-", start.Replication, start.Arm)) {
		return start, end, errors.New("process service belongs to another arm")
	}
	if err := checkStampInterval(native.Start, native.End); err != nil {
		return start, end, err
	}
	if native.Start.Boot != end.Begin.Boot || native.Start.NS < end.Begin.NS || native.End.NS > end.End.NS {
		return start, end, errors.New("native execution outside customer envelope")
	}
	if native.ResourcesBefore.At.Boot != native.Start.Boot || native.ResourcesBefore.At.NS > native.Start.NS || native.ResourcesAfter.At.Boot != native.End.Boot || native.ResourcesAfter.At.NS < native.End.NS || len(native.ResourcesBefore.CgroupFiles) != 5 || len(native.ResourcesAfter.CgroupFiles) != 5 || native.ResourcesBefore.CPUAffinity == "" {
		return start, end, errors.New("missing native resource/host interval")
	}
	var rawPID ProcessIdentity
	if err := readJSON(filepath.Join(dir, "native-pid.json"), &rawPID); err != nil {
		return start, end, err
	}
	if !equalJSON(rawPID, native.Process) {
		return start, end, errors.New("process identity differs from launch record")
	}
	var rawStart struct {
		Request    ProcessRequest  `json:"request"`
		Supervisor ProcessIdentity `json:"supervisor"`
		At         Stamp           `json:"at"`
	}
	if err := readJSON(filepath.Join(dir, "native-start.json"), &rawStart); err != nil {
		return start, end, err
	}
	if !equalJSON(rawStart.Supervisor, native.Supervisor) || !equalJSON(rawStart.Request.Command, start.Command) || !equalJSON(rawStart.Request.Environment, start.Environment) || rawStart.Request.Attempt != start.ID {
		return start, end, errors.New("native command differs from reserved request")
	}
	if rawStart.Request.Directory != filepath.Join(before.Root, "repo") || rawStart.Request.Evidence != dir || !equalJSON(rawStart.At, native.Start) {
		return start, end, errors.New("native directory/boundary differs from declared request")
	}
	for _, v := range []struct{ path, hash string }{{"stdout.log", native.StdoutSHA256}, {"stderr.log", native.StderrSHA256}} {
		if err := checkBinding(Binding{filepath.Join(dir, v.path), v.hash}); err != nil {
			return start, end, err
		}
	}
	if native.Process.Cgroup != native.Supervisor.Cgroup && !strings.HasPrefix(native.Process.Cgroup, native.Supervisor.Cgroup+"/") {
		return start, end, errors.New("native process outside owned cgroup")
	}
	if err := checkSession(m, start, native); err != nil {
		return start, end, err
	}
	for _, p := range native.Observed {
		if p.Cgroup != native.Supervisor.Cgroup && !strings.HasPrefix(p.Cgroup, native.Supervisor.Cgroup+"/") {
			return start, end, errors.New("observed process escaped containment")
		}
	}
	var capture Capture
	if err := checkBinding(end.Capture); err != nil {
		return start, end, err
	}
	if err := readJSON(end.Capture.Path, &capture); err != nil {
		return start, end, err
	}
	if err := checkCapture(m, capture, filepath.Join(before.Root, "repo"), dir, native); err != nil {
		return start, end, err
	}
	if err := checkOutputCoverage(m, capture, after); err != nil {
		return start, end, err
	}
	if capture.Error != "" && end.Class != harnessInvalid {
		return start, end, errors.New("capture error hidden by classification")
	}
	if m.Driver == "FIXTURE" {
		var rawTasks []TaskOutcome
		if err := readJSON(filepath.Join(dir, "fixture-tasks.json"), &rawTasks); err != nil {
			return start, end, err
		}
		if !equalJSON(rawTasks, capture.Tasks) {
			return start, end, errors.New("forged fixture task outcomes")
		}
	}
	return start, end, nil
}

func reconstruct(root string) (RunResult, error) {
	result := RunResult{Schema: resultSchema, Replications: []Economics{}, Slots: [][]Slot{}, Decision: "INCOMPLETE_EVIDENCE", Reasons: []string{}}
	var m Manifest
	if err := readJSON(filepath.Join(root, "manifest.json"), &m); err != nil {
		return result, err
	}
	if m.RunRoot != root {
		return result, errors.New("run-root identity differs")
	}
	if err := validateManifest(m); err != nil {
		return result, err
	}
	for _, p := range m.Outputs.Projectors {
		if err := checkProjectorProof(p, true); err != nil {
			return result, err
		}
	}
	h, err := fileDigest(filepath.Join(root, "manifest.json"))
	if err != nil {
		return result, err
	}
	result.ManifestSHA256 = h
	costList, err := loadCosts(root)
	if err != nil {
		return result, err
	}
	extra, err := externalCosts(m)
	if err != nil {
		return result, err
	}
	costList = append(costList, extra...)
	costs := map[string]Cost{}
	for _, c := range costList {
		if _, ok := costs[c.ID]; ok {
			return result, errors.New("duplicate phase ID")
		}
		costs[c.ID] = c
	}
	attemptDirs, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return result, err
	}
	starts := map[string]AttemptStart{}
	ordered := []AttemptStart{}
	for _, d := range attemptDirs {
		if !d.IsDir() {
			return result, errors.New("unexpected attempts entry")
		}
		path := filepath.Join(root, "attempts", d.Name(), "start.json")
		var a AttemptStart
		if err = readJSON(path, &a); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return result, err
		}
		if a.ID != d.Name() || starts[a.ID].ID != "" {
			return result, errors.New("duplicate attempt identity")
		}
		starts[a.ID] = a
		ordered = append(ordered, a)
		result.GradleReservations += a.ReservedGradleStarts
		if _, err = os.Stat(filepath.Join(root, "attempts", a.ID, "native-pid.json")); err == nil {
			result.WorkflowStarts++
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].At.NS < ordered[j].At.NS })
	lastNS := int64(0)
	for _, a := range ordered {
		if a.At.NS <= lastNS {
			return result, errors.New("reordered/ambiguous start clock")
		}
		lastNS = a.At.NS
	}
	if len(ordered) > m.Limits.MaxWorkflowStarts || result.GradleReservations > m.Limits.MaxGradleStarts {
		return result, errors.New("invocation allocation exceeded")
	}
	seenAttempts := map[string]bool{}
	sourceHashes := map[string]string{}
	recoveryGenerations := map[string]int{}
	recoveryFiles, err := filepath.Glob(filepath.Join(root, "archives", "recovery-*.json"))
	if err != nil {
		return result, err
	}
	if len(recoveryFiles) > m.Limits.RetryPairs {
		return result, errors.New("retry reserve exceeded")
	}
	for _, path := range recoveryFiles {
		var recovery Recovery
		if err = readJSON(path, &recovery); err != nil {
			return result, err
		}
		if recovery.Schema != recordSchema || m.DaemonPolicy != "REQUEST" || recovery.Generation < 1 || len(recovery.RestoredStates) != 2 || len(recovery.ArchivedStates) != 2 {
			return result, errors.New("unsupported recovery")
		}
		if err = checkBinding(recovery.Snapshot); err != nil {
			return result, err
		}
		var snapshot pairSnapshot
		if err = readJSON(recovery.Snapshot.Path, &snapshot); err != nil {
			return result, err
		}
		if snapshot.Replication != recovery.Replication || snapshot.Ordinal != recovery.Ordinal {
			return result, errors.New("recovery snapshot from another ordinal")
		}
		for _, pin := range append(append(append([]Binding{}, snapshot.Copies...), snapshot.GitMetadata...), recovery.ArchivedStates...) {
			if err = checkBinding(pin); err != nil {
				return result, err
			}
		}
		for index, pin := range recovery.RestoredStates {
			if err = checkBinding(pin); err != nil {
				return result, err
			}
			if pin.SHA256 != snapshot.States[index].SHA256 {
				return result, errors.New("unequal pre-attempt recovery state")
			}
		}
		key := slotKey(recovery.Replication, recovery.Ordinal)
		if recovery.Generation != recoveryGenerations[key]+1 {
			return result, errors.New("missing/reordered recovery generation")
		}
		recoveryGenerations[key] = recovery.Generation
		for _, pin := range recovery.AbandonedAttempts {
			if err = checkBinding(pin); err != nil {
				return result, err
			}
			id := filepath.Base(pin.Path)
			a, ok := starts[id]
			if !ok || seenAttempts[id] || a.Replication != recovery.Replication || a.Ordinal != recovery.Ordinal || a.Generation >= recovery.Generation {
				return result, errors.New("invalid/duplicate abandoned attempt")
			}
			seenAttempts[id] = true
		}
	}
	if err = accountInvocations(m, ordered, &result); err != nil {
		return result, err
	}
	previousState := map[string]string{}
	previousInventories := map[string]StatePin{}
	previousPair := ""
	lastEnd := int64(0)
	pairFiles, err := os.ReadDir(filepath.Join(root, "pairs"))
	if err != nil {
		return result, err
	}
	pairs := map[string]PairRecord{}
	for _, f := range pairFiles {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			return result, errors.New("unexpected pair record")
		}
		var p PairRecord
		if err = readJSON(filepath.Join(root, "pairs", f.Name()), &p); err != nil {
			return result, err
		}
		key := slotKey(p.Replication, p.Ordinal)
		if f.Name() != key+".json" || p.Replication < 1 || p.Replication > m.Replications || p.Ordinal < 0 || p.Ordinal > m.ExecutionEnd {
			return result, errors.New("duplicate or outside-horizon pair")
		}
		pairs[key] = p
	}
	stopped := false
	for rep := 1; rep <= m.Replications; rep++ {
		_, boundary, boundaryErr := latestBoundary(root, rep)
		if boundaryErr != nil && !os.IsNotExist(boundaryErr) {
			return result, boundaryErr
		}
		slots := []Slot{}
		for ordinal := 0; ordinal <= m.ExecutionEnd; ordinal++ {
			key := slotKey(rep, ordinal)
			p, ok := pairs[key]
			if !ok {
				stopped = true
				slot := Slot{Ordinal: ordinal, Class: notRunDependency, Reason: "scheduled request depends on incomplete preceding work", Attempts: []string{}}
				if boundary.Status == notRunLimit || boundary.Status == "PAUSED" {
					slot.Class = notRunLimit
					slot.Reason = boundary.Reason
				}
				for _, a := range ordered {
					if a.Replication == rep && a.Ordinal == ordinal {
						slot.Attempts = append(slot.Attempts, a.ID)
						seenAttempts[a.ID] = true
						slot.Class = harnessInvalid
						slot.Reason = "partial pair retained without saving"
						if a.Arm == "I" {
							if c, ok := costs[a.ID+"-request"]; ok {
								slot.CandidateNS += c.DurationNS
							}
						}
					}
				}
				slots = append(slots, slot)
				continue
			}
			if stopped {
				return result, errors.New("completed ordinal after unexplained coverage gap")
			}
			if p.Schema != recordSchema || p.Previous != previousPair || p.Generation != recoveryGenerations[key] || len(p.Attempts) != 2 {
				return result, errors.New("pair lineage/generation differs")
			}
			ends := map[string]AttemptEnd{}
			order := armOrder(rep, ordinal)
			for index, b := range p.Attempts {
				a, e, err := loadAttempt(m, b, costs, sourceHashes)
				if err != nil {
					return result, err
				}
				if seenAttempts[a.ID] || a.Replication != rep || a.Ordinal != ordinal || a.Arm != order[index] || a.Generation != p.Generation {
					return result, errors.New("duplicate/reordered/cross-ordinal arm")
				}
				seenAttempts[a.ID] = true
				ends[a.Arm] = e
				if e.Begin.NS < lastEnd {
					return result, errors.New("overlapping arm execution")
				}
				lastEnd = e.End.NS
				key := armKey(rep, a.Arm)
				var state StatePin
				if err = readJSON(a.Before.Path, &state); err != nil {
					return result, err
				}
				if ordinal == 0 && state.Previous != "" {
					return result, errors.New("future/pretrained initial state")
				}
				if ordinal > 0 && state.Previous != previousState[key] {
					return result, errors.New("state borrows another arm/future request")
				}
				if ordinal > 0 {
					if err := requirePhase(costs, slotKey(rep, ordinal)+"-"+a.Arm+"-source", "neutral-source-advancement", rep, ordinal, a.Arm); err != nil {
						return result, err
					}
					if a.Arm == "I" && previousInventories[key].CandidateApplied {
						if err := requirePhase(costs, slotKey(rep, ordinal)+"-I-inverse", "candidate-inverse", rep, ordinal, "I"); err != nil {
							return result, err
						}
					}
					if err = checkStateTransition(m, previousInventories[key], state); err != nil {
						return result, err
					}
				}
				if ordinal == 0 {
					if err := requirePhase(costs, key+"-acquisition", "native-state-preparation", rep, 0, a.Arm); err != nil {
						return result, err
					}
				}
				previousState[key] = e.After.SHA256
				var after StatePin
				if err = readJSON(e.After.Path, &after); err != nil {
					return result, err
				}
				previousInventories[key] = after
				var c Capture
				if err = readJSON(e.Capture.Path, &c); err != nil {
					return result, err
				}
				if m.Driver == "GRADLE" {
					if a.ReservedGradleStarts != 1+m.Limits.NestedReservePerRequest || len(c.GradleBuilds) < 1 || len(c.GradleBuilds) > a.ReservedGradleStarts {
						return result, errors.New("missing or uncharged nested starts")
					}
				} else if a.ReservedGradleStarts != 0 || len(c.GradleBuilds) != 0 {
					return result, errors.New("fixture falsely claims Gradle")
				}
			}
			if !absentBinding(m.Outputs.Owner) {
				if err := requirePhase(costs, slotKey(rep, ordinal)+"-comparison", "owner-output-comparison", rep, ordinal, ""); err != nil {
					return result, err
				}
			}
			slot, err := classifyPair(m, ends["N"], ends["I"])
			if err != nil {
				return result, err
			}
			slot.Ordinal = ordinal
			slot.ExtraCandidateNS, err = retryCharge(root, rep, ordinal, p.Generation)
			if err != nil {
				return result, err
			}
			if !equalJSON(slot, p.Slot) {
				return result, errors.New("forged pair durations/classification/actions")
			}
			slots = append(slots, slot)
			previousPair = objectDigest(p)
			if slot.Class == candidateFailure || slot.Class == harnessInvalid {
				stopped = true
			}
		}
		e, err := calculate(m, rep, slots, costList)
		if err != nil {
			return result, err
		}
		if err = applyResearchAccounting(m, rep, &e, costList); err != nil {
			return result, err
		}
		if !e.Complete {
			stopped = true
		}
		result.Slots = append(result.Slots, slots)
		result.Replications = append(result.Replications, e)
	}
	if len(seenAttempts) != len(starts) {
		return result, errors.New("hidden/unaccounted retry attempt")
	}
	if result.UnknownGradleReservations != 0 {
		stopped = true
		result.Reasons = append(result.Reasons, "incomplete native invocation accounting")
		for index := range result.Replications {
			result.Replications[index].Complete = false
			result.Replications[index].Gate = "INCOMPLETE_EVIDENCE"
			result.Replications[index].PaybackOrdinal = -1
		}
	}
	if !stopped {
		result.Decision = "FIXTURE_VERIFIED"
		if m.Phase == "ENGINEERING" {
			result.Decision = "ENGINEERING_COMPLETE"
		}
		if m.Phase == "CONFIRMATION" {
			result.Decision = "G3_PASS"
			for _, e := range result.Replications {
				if e.Gate != "G3_PASS" {
					result.Decision = e.Gate
				}
			}
		}
	} else {
		result.Reasons = append(result.Reasons, "incomplete or unsafe scheduled horizon")
	}
	return result, nil
}

func checkResult(root, path string) error {
	actual, err := reconstruct(root)
	if err != nil {
		return err
	}
	var claimed RunResult
	if err = readJSON(path, &claimed); err != nil {
		return err
	}
	if !equalJSON(actual, claimed) {
		return errors.New("claimed summary/payback/decision differs from reconstructed raw records")
	}
	return checkExports(root, actual)
}

func writeResult(root string) (RunResult, error) {
	r, err := reconstruct(root)
	if err != nil {
		return r, err
	}
	entries, err := os.ReadDir(filepath.Join(root, "results"))
	if err != nil {
		return r, err
	}
	path := filepath.Join(root, "results", fmt.Sprintf("%04d.json", len(entries)+1))
	if _, err = saveBound(path, r); err != nil {
		return r, err
	}
	if err = atomicJSON(filepath.Join(root, "result.json"), r); err != nil {
		return r, err
	}
	if err = writeExports(root, fmt.Sprintf("%04d", len(entries)+1), r); err != nil {
		return r, err
	}
	return r, nil
}
