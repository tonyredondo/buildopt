//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type runner struct {
	manifest           Manifest
	binding            Binding
	start              Stamp
	attempts           int
	gradleReservations int
	retries            int
	sessions           map[string]workerSession
	states             map[string]StatePin
	stateBindings      map[string]Binding
	builds             map[string]map[string]bool
	lastPair           string
	generations        map[string]int
	// Test-only process interruption points. Public manifests cannot weaken
	// the runner, inject arbitrary sleeps or forge native observations.
	fault func(string, string)
}

func (r *runner) phase(c Cost, work func() error) error {
	p, err := beginCost(r.manifest.RunRoot, c)
	if err != nil {
		return err
	}
	workErr := work()
	_, finishErr := finishCost(r.manifest.RunRoot, p)
	if workErr != nil {
		return workErr
	}
	return finishErr
}

func (r *runner) guard() error {
	if stamp().Boot != r.start.Boot || stamp().NS-r.start.NS > r.manifest.Limits.MaxRunNS {
		return errors.New("NOT_RUN_LIMIT: elapsed allocation")
	}
	deadline, err := time.Parse(time.RFC3339Nano, r.manifest.Limits.DeadlineUTC)
	if err != nil {
		return err
	}
	if time.Now().After(deadline) {
		return errors.New("NOT_RUN_LIMIT: deadline")
	}
	return diskGuard(r.manifest.RunRoot, r.manifest.Limits)
}

func checkInputs(m Manifest) error {
	if _, err := readObserverPolicy(m); err != nil {
		return err
	}
	if _, err := externalCosts(m); err != nil {
		return err
	}
	bindings := append([]Binding{m.Protocol, m.Executable}, m.Package...)
	bindings = append(bindings, m.Runtime...)
	for _, p := range allRecipes(m) {
		for _, f := range p.Files {
			bindings = append(bindings, f.After)
		}
	}
	for _, b := range []Binding{m.Correctness, m.Overhead, m.Subjects, m.Outputs.GraphCapture, m.Outputs.Owner} {
		if !absentBinding(b) {
			bindings = append(bindings, b)
		}
	}
	for _, p := range m.Outputs.Projectors {
		bindings = append(bindings, p.Executable, p.Qualification)
	}
	for _, b := range bindings {
		if err := checkBinding(b); err != nil {
			return err
		}
	}
	_, err := readOwnerPolicy(m)
	return err
}

func newRunner(path string) (*runner, error) {
	var m Manifest
	if err := readJSON(path, &m); err != nil {
		return nil, err
	}
	if err := validateManifest(m); err != nil {
		return nil, err
	}
	if err := checkDriverBinary(m); err != nil {
		return nil, err
	}
	if err := os.Mkdir(m.RunRoot, 0700); err != nil {
		return nil, fmt.Errorf("execution requires a new owned root: %w", err)
	}
	for _, dir := range []string{"attempts", "costs", "pairs", "checkpoints", "snapshots", "sessions", "results", "archives", "states", "lifecycle"} {
		if err := os.Mkdir(filepath.Join(m.RunRoot, dir), 0700); err != nil {
			return nil, err
		}
	}
	b, err := saveBound(filepath.Join(m.RunRoot, "manifest.json"), m)
	if err != nil {
		return nil, err
	}
	r := &runner{manifest: m, binding: b, start: stamp(), sessions: map[string]workerSession{}, states: map[string]StatePin{}, stateBindings: map[string]Binding{}, builds: map[string]map[string]bool{}, generations: map[string]int{}}
	if _, err = saveBound(filepath.Join(m.RunRoot, "run-start.json"), r.start); err != nil {
		return nil, err
	}
	if err = writeExclusive(filepath.Join(m.RunRoot, "ownership.json"), jsonBytes(struct {
		Schema    string          `json:"schema"`
		Manifest  Binding         `json:"manifest"`
		CommonGit string          `json:"commonGit"`
		Driver    ProcessIdentity `json:"driver"`
	}{recordSchema, b, m.CommonGit, mustIdentity()}), 0600); err != nil {
		return nil, err
	}
	return r, nil
}

func mustIdentity() ProcessIdentity {
	p, e := processIdentity(os.Getpid())
	if e != nil {
		panic(e)
	}
	return p
}
func armKey(rep int, arm string) string { return fmt.Sprintf("r%d-%s", rep, arm) }

func (r *runner) initialize(rep int) error {
	if err := r.guard(); err != nil {
		return err
	}
	if len(r.manifest.Outputs.Projectors) > 0 {
		if err := r.phase(Cost{ID: fmt.Sprintf("r%d-projectors", rep), Class: "research", Purpose: "projector-qualification", Replication: rep, Ordinal: 0}, func() error {
			for _, p := range r.manifest.Outputs.Projectors {
				if err := checkProjectorProof(p, true); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	m := r.manifest
	parent := filepath.Join(m.RunRoot, fmt.Sprintf("r%d", rep))
	if err := os.Mkdir(parent, 0700); err != nil {
		return err
	}
	for _, arm := range []string{"N", "I"} {
		key := armKey(rep, arm)
		root := filepath.Join(parent, arm)
		repo := filepath.Join(root, "repo")
		err := r.phase(Cost{ID: key + "-acquisition", Class: "research", Purpose: "native-state-preparation", Replication: rep, Ordinal: 0, Arm: arm}, func() error {
			if err := os.Mkdir(root, 0700); err != nil {
				return err
			}
			for _, dir := range []string{"home", "gradle", "tmp", "cache"} {
				if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
					return err
				}
			}
			for _, layer := range m.Acquisition {
				if err := checkBinding(layer.Source); err != nil {
					return err
				}
				dest := filepath.Join(root, layer.Destination)
				if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
					return err
				}
				if err := copyTree(layer.Source.Path, dest); err != nil {
					return err
				}
				if err := checkBinding(Binding{dest, layer.Source.SHA256}); err != nil {
					return err
				}
			}
			branch := fmt.Sprintf("bv-replay-%s-%s-000", digest([]byte(m.RunRoot))[:12], key)
			if _, err := gitAt(m.CommonGit, "worktree", "add", "--quiet", "-b", branch, repo, m.History[0].Commit); err != nil {
				return err
			}
			ok, err := patchApplicable(repo, baselineForArm(m, arm))
			if err != nil {
				return err
			}
			if ok {
				if err = applyPatch(repo, baselineForArm(m, arm)); err != nil {
					return err
				}
			}
			s, err := captureState(m, rep, arm, 0, "", ok, false)
			if err != nil {
				return err
			}
			b, err := saveBound(filepath.Join(m.RunRoot, "states", key+"-initial.json"), s)
			if err != nil {
				return err
			}
			r.states[key] = s
			r.stateBindings[key] = b
			r.builds[key] = map[string]bool{}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func patchesFor(m Manifest, s StatePin) []Patch {
	p := []Patch{}
	if s.BaselineApplied {
		p = append(p, baselineForArm(m, s.Arm))
	}
	if s.CandidateApplied {
		p = append(p, m.Candidate)
	}
	return p
}

func (r *runner) advance(rep, ordinal int, arm string) error {
	m := r.manifest
	key := armKey(rep, arm)
	s := r.states[key]
	if ordinal != s.Ordinal+1 {
		return errors.New("source advancement must consume exactly the next first-parent ordinal")
	}
	repo := filepath.Join(s.Root, "repo")
	if err := r.guard(); err != nil {
		return err
	}
	if err := verifyState(m, s); err != nil {
		return err
	}
	if _, err := verifySource(m.CommonGit, repo, s.Revision, patchesFor(m, s), m.GeneratedPaths); err != nil {
		return err
	}
	if s.CandidateApplied {
		if err := r.phase(Cost{ID: slotKey(rep, ordinal) + "-I-inverse", Class: "customer-machine", Purpose: "candidate-inverse", Replication: rep, Ordinal: ordinal, Arm: arm}, func() error { return reversePatch(m.CommonGit, repo, s.Revision, m.Candidate) }); err != nil {
			return err
		}
	}
	return r.phase(Cost{ID: slotKey(rep, ordinal) + "-" + arm + "-source", Class: "research", Purpose: "neutral-source-advancement", Replication: rep, Ordinal: ordinal, Arm: arm}, func() error {
		if s.BaselineApplied {
			if err := reversePatch(m.CommonGit, repo, s.Revision, baselineForArm(m, arm)); err != nil {
				return err
			}
		}
		if _, err := verifySource(m.CommonGit, repo, s.Revision, []Patch{}, m.GeneratedPaths); err != nil {
			return err
		}
		branch := fmt.Sprintf("bv-replay-%s-%s-%03d", digest([]byte(m.RunRoot))[:12], key, ordinal)
		if _, err := gitAt(repo, "switch", "--quiet", "--no-overwrite-ignore", "-c", branch, m.History[ordinal].Commit); err != nil {
			return err
		}
		ok, err := patchApplicable(repo, baselineForArm(m, arm))
		if err != nil {
			return err
		}
		if ok {
			if err = applyPatch(repo, baselineForArm(m, arm)); err != nil {
				return err
			}
		}
		next, err := captureState(m, rep, arm, ordinal, r.stateBindings[key].SHA256, ok, false)
		if err != nil {
			return err
		}
		b, err := saveBound(filepath.Join(m.RunRoot, "states", key+fmt.Sprintf("-%03d-before.json", ordinal)), next)
		if err != nil {
			return err
		}
		r.states[key] = next
		r.stateBindings[key] = b
		return nil
	})
}

type pairSnapshot struct {
	Schema      string    `json:"schema"`
	Replication int       `json:"replication"`
	Ordinal     int       `json:"ordinal"`
	States      []Binding `json:"states"`
	Copies      []Binding `json:"copies"`
	GitMetadata []Binding `json:"gitMetadata"`
}

func (r *runner) snapshot(rep, ordinal int) error {
	m := r.manifest
	if m.DaemonPolicy != "REQUEST" || m.Limits.RetryPairs == 0 {
		return nil
	}
	key := slotKey(rep, ordinal)
	dir := filepath.Join(m.RunRoot, "snapshots", key)
	return r.phase(Cost{ID: key + "-snapshot", Class: "research", Purpose: "equal-state-recovery-checkpoint", Replication: rep, Ordinal: ordinal}, func() error {
		if err := os.Mkdir(dir, 0700); err != nil {
			return err
		}
		s := pairSnapshot{Schema: recordSchema, Replication: rep, Ordinal: ordinal, States: []Binding{}, Copies: []Binding{}, GitMetadata: []Binding{}}
		for _, arm := range []string{"N", "I"} {
			key := armKey(rep, arm)
			state := r.states[key]
			if err := verifyState(m, state); err != nil {
				return err
			}
			dst := filepath.Join(dir, arm)
			if err := copyTree(state.Root, dst); err != nil {
				return err
			}
			b, err := bind(dst)
			if err != nil {
				return err
			}
			s.Copies = append(s.Copies, b)
			s.States = append(s.States, r.stateBindings[key])
			gd, err := gitAt(filepath.Join(state.Root, "repo"), "rev-parse", "--absolute-git-dir")
			if err != nil {
				return err
			}
			meta := strings.TrimSpace(string(gd))
			if !inside(filepath.Join(m.CommonGit, "worktrees"), meta) {
				return errors.New("snapshot requires registered owned worktree metadata")
			}
			dst = filepath.Join(dir, arm+"-git")
			if err = copyTree(meta, dst); err != nil {
				return err
			}
			b, err = bind(dst)
			if err != nil {
				return err
			}
			s.GitMetadata = append(s.GitMetadata, b)
		}
		_, err := saveBound(filepath.Join(dir, "snapshot.json"), s)
		return err
	})
}

func (r *runner) stopWorkers() error {
	keys := []string{}
	for k := range r.sessions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var first error
	for _, key := range keys {
		s := r.sessions[key]
		if err := s.stop(); err != nil && first == nil {
			first = err
		}
		delete(r.sessions, key)
	}
	return first
}

func (r *runner) request(rep, ordinal, generation int, arm string) (AttemptEnd, Binding, error) {
	m := r.manifest
	key := armKey(rep, arm)
	id := fmt.Sprintf("%s-g%d-%s", slotKey(rep, ordinal), generation, arm)
	evidence := filepath.Join(m.RunRoot, "attempts", id)
	end := AttemptEnd{Schema: recordSchema, ID: id, Costs: []string{}}
	if err := r.guard(); err != nil {
		return end, Binding{}, err
	}
	reserve := 0
	if m.Driver == "GRADLE" {
		reserve = 1 + m.Limits.NestedReservePerRequest
	}
	if r.attempts+1 > m.Limits.MaxWorkflowStarts || r.gradleReservations+reserve > m.Limits.MaxGradleStarts {
		return end, Binding{}, errors.New("NOT_RUN_LIMIT: prospective invocation allocation")
	}
	if err := os.Mkdir(evidence, 0700); err != nil {
		return end, Binding{}, err
	}
	s := r.states[key]
	repo := filepath.Join(s.Root, "repo")
	cmd, env := workflow(m, rep, arm, evidence)
	var source string
	err := r.phase(Cost{ID: id + "-preflight", Class: "research", Purpose: "source-input-state-proof", Replication: rep, Ordinal: ordinal, Arm: arm, Attempt: id}, func() error {
		if err := checkInputs(m); err != nil {
			return err
		}
		if err := verifyState(m, s); err != nil {
			return err
		}
		var err error
		source, err = verifySource(m.CommonGit, repo, s.Revision, patchesFor(m, s), m.GeneratedPaths)
		return err
	})
	if err != nil {
		return end, Binding{}, err
	}
	start := AttemptStart{recordSchema, id, rep, ordinal, arm, generation, r.binding.SHA256, r.stateBindings[key], source, cmd, env, reserve, stamp()}
	if _, err = saveBound(filepath.Join(evidence, "start.json"), start); err != nil {
		return end, Binding{}, err
	}
	r.attempts++
	r.gradleReservations += reserve
	if r.fault != nil {
		r.fault("reserved", id)
	}
	worker, ok := r.sessions[key]
	if !ok {
		serviceEvidence := filepath.Join(m.RunRoot, "sessions", id)
		if err = os.Mkdir(serviceEvidence, 0700); err != nil {
			return end, Binding{}, err
		}
		err = r.phase(Cost{ID: id + "-containment", Class: "research", Purpose: "owned-process-supervision", Replication: rep, Ordinal: ordinal, Arm: arm, Attempt: id}, func() error {
			var err error
			worker, err = startWorker(m, r.binding, rep, arm, r.attempts, serviceEvidence)
			return err
		})
		if err != nil {
			return end, Binding{}, err
		}
		r.sessions[key] = worker
	}
	var observer *processObserver
	if !absentBinding(m.Observer) {
		err = r.phase(Cost{ID: id + "-observer-start", Class: "research", Purpose: "observer-start", Replication: rep, Ordinal: ordinal, Arm: arm, Attempt: id}, func() error { var err error; observer, err = startProcessObserver(m, worker, evidence); return err })
		if err != nil {
			return end, Binding{}, err
		}
	}
	// Every early return after startup must close the observer and its helpers.
	observerFinished := false
	defer func() {
		if observer != nil && !observerFinished {
			observer.finish()
		}
	}()
	p, err := beginCost(m.RunRoot, Cost{ID: id + "-request", Class: "customer-machine", Purpose: "complete-request", Replication: rep, Ordinal: ordinal, Arm: arm, Attempt: id, InsideEnvelope: true, WorkflowStarts: 1})
	if err != nil {
		return end, Binding{}, err
	}
	end.Begin = p.At
	end.Costs = append(end.Costs, p.Cost.ID)
	if arm == "I" {
		end.CandidateApplied, err = patchApplicable(repo, m.Candidate)
		if err == nil && end.CandidateApplied {
			err = applyPatch(repo, m.Candidate)
		}
	}
	var native ProcessReceipt
	if err == nil {
		remaining := m.Limits.MaxRequestNS - (stamp().NS - end.Begin.NS)
		if remaining <= 0 {
			err = errors.New("NOT_RUN_LIMIT: customer preparation exhausted request timeout")
		} else {
			q := ProcessRequest{recordSchema, id, repo, cmd, env, evidence, remaining}
			if observer == nil {
				native, err = worker.invoke(q, func(ProcessReceipt) { end.End = stamp() })
			} else {
				native, err = invokeObserved(worker, q, observer, func(ProcessReceipt) { end.End = stamp() })
			}
		}
	}
	requestErr := err
	if end.End.NS == 0 {
		end.End = stamp()
	}
	cost, finishErr := finishCostAt(m.RunRoot, p, end.End)
	if finishErr != nil {
		return end, Binding{}, finishErr
	}
	if err = readJSON(filepath.Join(m.RunRoot, "costs", cost.ID+".end.json"), &end.End); err != nil {
		return end, Binding{}, err
	}
	end.DurationNS = cost.DurationNS
	if observer != nil {
		observerErr := r.phase(Cost{ID: id + "-observer-close", Class: "research", Purpose: "observer-close", Replication: rep, Ordinal: ordinal, Arm: arm, Attempt: id}, func() error {
			_, err := observer.finish()
			observerFinished = true
			if err != nil {
				return err
			}
			return checkObservation(m, evidence, native)
		})
		requestErr = errors.Join(requestErr, observerErr)
	}
	if r.fault != nil {
		r.fault("customer-finished", id)
	}
	if m.DaemonPolicy == "REQUEST" || native.Outcome != "EXITED" {
		if err = worker.stop(); err != nil {
			return end, Binding{}, err
		}
		delete(r.sessions, key)
	}
	if requestErr != nil {
		end.Class = harnessInvalid
		end.Reason = requestErr.Error()
		return end, Binding{}, requestErr
	}
	end.Native, err = bind(filepath.Join(evidence, "native-finish.json"))
	if err != nil {
		return end, Binding{}, err
	}
	err = r.phase(Cost{ID: id + "-capture", Class: "research", Purpose: "complete-output-state-and-invocation-capture", Replication: rep, Ordinal: ordinal, Arm: arm, Attempt: id}, func() error {
		// The live scan can stop at child exit. A complete postflight sample
		// still enforces the allocation, outside the customer request envelope.
		if err := r.guard(); err != nil {
			return err
		}
		if err := checkInputs(m); err != nil {
			return err
		}
		patches := patchesFor(m, s)
		if end.CandidateApplied {
			patches = append(patches, m.Candidate)
		}
		if _, err := verifySource(m.CommonGit, repo, s.Revision, patches, m.GeneratedPaths); err != nil {
			return err
		}
		capture, captureErr := captureOutputs(m, repo, evidence, native, r.builds[key])
		if captureErr != nil {
			capture.Error = captureErr.Error()
		}
		var err error
		end.Capture, err = saveBound(filepath.Join(evidence, "capture.json"), capture)
		if err != nil {
			return err
		}
		for _, build := range capture.GradleBuilds {
			r.builds[key][build.ID] = true
		}
		if captureErr != nil {
			return captureErr
		}
		if len(capture.GradleBuilds) > reserve {
			return errors.New("unreserved nested Gradle command")
		}
		state, err := captureState(m, rep, arm, ordinal, r.stateBindings[key].SHA256, s.BaselineApplied, end.CandidateApplied)
		if err != nil {
			return err
		}
		end.After, err = saveBound(filepath.Join(evidence, "after.json"), state)
		if err != nil {
			return err
		}
		r.states[key] = state
		r.stateBindings[key] = end.After
		return nil
	})
	end.Class = comparable
	if !end.CandidateApplied && arm == "I" {
		end.Class = nativeRetained
		end.Reason = "NATIVE_RETAINED_UNSUPPORTED_CHANGE"
	}
	if native.ExitCode != 0 {
		end.Class = nativeFailure
		end.Reason = "native process failed; pair classification required"
	}
	if err != nil || native.Outcome != "EXITED" {
		end.Class = harnessInvalid
		if err != nil {
			end.Reason = err.Error()
		} else {
			end.Reason = native.Outcome
		}
	}
	if r.fault != nil {
		r.fault("before-receipt", id)
	}
	b, writeErr := saveBound(filepath.Join(evidence, "end.json"), end)
	if writeErr != nil {
		return end, b, writeErr
	}
	if r.fault != nil {
		r.fault("after-receipt", id)
	}
	return end, b, nil
}

func classifyPair(m Manifest, n, i AttemptEnd) (Slot, error) {
	s := Slot{Attempts: []string{n.ID, i.ID}, NativeNS: n.DurationNS, CandidateNS: i.DurationNS, Class: comparable}
	var np, ip ProcessReceipt
	var nc, ic Capture
	for _, v := range []struct {
		b   Binding
		out any
	}{{n.Native, &np}, {i.Native, &ip}, {n.Capture, &nc}, {i.Capture, &ic}} {
		if absentBinding(v.b) {
			s.Class = harnessInvalid
			s.Reason = "missing request capture"
			return s, nil
		}
		if err := checkBinding(v.b); err != nil {
			return s, err
		}
		if err := readJSON(v.b.Path, v.out); err != nil {
			return s, err
		}
	}
	for _, t := range nc.Tasks {
		if t.Action {
			s.NativeActions++
		}
	}
	for _, t := range ic.Tasks {
		if t.Action {
			s.CandidateActions++
		}
	}
	if n.Class == harnessInvalid || i.Class == harnessInvalid {
		s.Class = harnessInvalid
		s.Reason = n.Reason + " " + i.Reason
		return s, nil
	}
	if np.ExitCode != ip.ExitCode {
		s.Class = candidateFailure
		s.Reason = "candidate/native exit differs"
		return s, nil
	}
	if absentBinding(m.Outputs.Owner) {
		if err := compareCaptures(nc, ic); err != nil {
			s.Class = candidateFailure
			s.Reason = err.Error()
			return s, nil
		}
	}
	var ns, is StatePin
	if err := readJSON(n.After.Path, &ns); err != nil {
		return s, err
	}
	if err := readJSON(i.After.Path, &is); err != nil {
		return s, err
	}
	var comparisonErr error
	if absentBinding(m.Outputs.Owner) {
		comparisonErr = compareWorkflowGraphs(nc, ic, filepath.Join(ns.Root, "repo"), filepath.Join(is.Root, "repo"))
	} else {
		s.OwnerMetadataJVMStarts, comparisonErr = compareOwnerCaptures(m, n, i)
	}
	if err := comparisonErr; err != nil {
		s.Class = candidateFailure
		s.Reason = err.Error()
		return s, nil
	}
	if np.ExitCode != 0 {
		s.Class = nativeFailure
		s.Reason = "native and candidate fail equivalently"
		if np.ExitCode == 69 {
			s.Class = dependencyUnavailable
			s.Reason = "explicit unavailable-artifact exit"
		}
		return s, nil
	}
	if !i.CandidateApplied {
		s.Class = nativeRetained
		s.Reason = i.Reason
	}
	return s, nil
}

func (r *runner) checkpoint(rep, ordinal int, pair Binding) error {
	c := Checkpoint{Schema: recordSchema, Manifest: r.binding, Replication: rep, LastOrdinal: ordinal, LastPair: pair, States: []Binding{}, Remaining: []int{}, Attempts: r.attempts, GradleReservations: r.gradleReservations, RetryPairs: r.retries, Sessions: []WorkerConfig{}, At: stamp()}
	for _, arm := range []string{"N", "I"} {
		key := armKey(rep, arm)
		c.States = append(c.States, r.stateBindings[key])
		if s, ok := r.sessions[key]; ok {
			c.Sessions = append(c.Sessions, s.Config)
		}
	}
	for i := ordinal + 1; i <= r.manifest.ExecutionEnd; i++ {
		c.Remaining = append(c.Remaining, i)
	}
	path := filepath.Join(r.manifest.RunRoot, "checkpoints", slotKey(rep, ordinal)+".json")
	if _, err := saveBound(path, c); err != nil {
		return err
	}
	return atomicJSON(filepath.Join(r.manifest.RunRoot, "checkpoint.json"), c)
}

func (r *runner) execute(firstRep, firstOrdinal int, pauseAfter int) (resultErr error) {
	currentRep, currentOrdinal := firstRep, firstOrdinal
	openBoundary := false
	defer func() {
		resultErr = errors.Join(resultErr, r.stopWorkers())
		if openBoundary {
			reason := "explicit qualification pause"
			if resultErr != nil {
				reason = resultErr.Error()
			}
			resultErr = errors.Join(resultErr, endReplication(r.manifest.RunRoot, currentRep, currentOrdinal, stopClass(resultErr), reason))
		}
	}()
	for rep := firstRep; rep <= r.manifest.Replications; rep++ {
		currentRep = rep
		currentOrdinal = 0
		if err := beginReplication(r.manifest.RunRoot, rep); err != nil {
			return err
		}
		openBoundary = true
		start := 0
		if rep == firstRep {
			start = firstOrdinal
		}
		if start == 0 {
			if _, ok := r.states[armKey(rep, "N")]; !ok {
				if err := r.initialize(rep); err != nil {
					return err
				}
			}
		}
		for ordinal := start; ordinal <= r.manifest.ExecutionEnd; ordinal++ {
			currentOrdinal = ordinal
			if ordinal > 0 {
				for _, arm := range []string{"N", "I"} {
					if r.states[armKey(rep, arm)].Ordinal < ordinal {
						if err := r.advance(rep, ordinal, arm); err != nil {
							return err
						}
					}
				}
			}
			generation := r.generations[slotKey(rep, ordinal)]
			if generation == 0 {
				if err := r.snapshot(rep, ordinal); err != nil {
					return err
				}
			}
			ends := map[string]AttemptEnd{}
			bindings := []Binding{}
			for _, arm := range armOrder(rep, ordinal) {
				end, b, err := r.request(rep, ordinal, generation, arm)
				if err != nil {
					return err
				}
				ends[arm] = end
				bindings = append(bindings, b)
				if end.Class == harnessInvalid {
					return fmt.Errorf("HARNESS_INVALID: %s", end.Reason)
				}
			}
			var slot Slot
			var err error
			if absentBinding(r.manifest.Outputs.Owner) {
				slot, err = classifyPair(r.manifest, ends["N"], ends["I"])
			} else {
				err = r.phase(Cost{ID: slotKey(rep, ordinal) + "-comparison", Class: "research", Purpose: "owner-output-comparison", Replication: rep, Ordinal: ordinal}, func() error {
					var compareErr error
					slot, compareErr = classifyPair(r.manifest, ends["N"], ends["I"])
					return compareErr
				})
			}
			if err != nil {
				return err
			}
			slot.Ordinal = ordinal
			slot.ExtraCandidateNS, err = retryCharge(r.manifest.RunRoot, rep, ordinal, generation)
			if err != nil {
				return err
			}
			p := PairRecord{recordSchema, rep, ordinal, generation, bindings, slot, r.lastPair, stamp()}
			b, err := saveBound(filepath.Join(r.manifest.RunRoot, "pairs", slotKey(rep, ordinal)+".json"), p)
			if err != nil {
				return err
			}
			r.lastPair = b.SHA256
			if r.fault != nil {
				r.fault("pair-sealed", slotKey(rep, ordinal))
			}
			if err = r.checkpoint(rep, ordinal, b); err != nil {
				return err
			}
			if slot.Class == candidateFailure || slot.Class == harnessInvalid {
				return fmt.Errorf("%s: %s", slot.Class, slot.Reason)
			}
			if r.fault != nil {
				r.fault("pair-checkpoint", slotKey(rep, ordinal))
			}
			if pauseAfter >= 0 && ordinal == pauseAfter {
				return nil
			}
		}
		if err := r.stopWorkers(); err != nil {
			return err
		}
		if err := endReplication(r.manifest.RunRoot, rep, r.manifest.ExecutionEnd, "COMPLETE", ""); err != nil {
			return err
		}
		openBoundary = false
	}
	return nil
}
