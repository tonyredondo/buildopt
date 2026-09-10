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

type Recovery struct {
	Schema            string    `json:"schema"`
	Replication       int       `json:"replication"`
	Ordinal           int       `json:"ordinal"`
	Generation        int       `json:"generation"`
	Snapshot          Binding   `json:"snapshot"`
	AbandonedAttempts []Binding `json:"abandonedAttempts"`
	ArchivedStates    []Binding `json:"archivedStates"`
	RestoredStates    []Binding `json:"restoredStates"`
	ClosedCosts       []string  `json:"closedCosts"`
	Reason            string    `json:"reason"`
	At                Stamp     `json:"at"`
}

func resumeRunner(root string) (*runner, int, int, error) {
	var m Manifest
	if err := readJSON(filepath.Join(root, "manifest.json"), &m); err != nil {
		return nil, 0, 0, err
	}
	if m.RunRoot != root {
		return nil, 0, 0, errors.New("resume root identity differs")
	}
	if err := validateManifest(m); err != nil {
		return nil, 0, 0, err
	}
	if err := checkDriverBinary(m); err != nil {
		return nil, 0, 0, err
	}
	b, err := bind(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, 0, 0, err
	}
	r := &runner{manifest: m, binding: b, sessions: map[string]workerSession{}, states: map[string]StatePin{}, stateBindings: map[string]Binding{}, builds: map[string]map[string]bool{}, generations: map[string]int{}}
	if err = readJSON(filepath.Join(root, "run-start.json"), &r.start); err != nil {
		return nil, 0, 0, err
	}
	if err = r.guard(); err != nil {
		return nil, 0, 0, err
	}
	// Only this run's frozen service configurations can be stopped. A still
	// live different driver owns the run and prevents a second executor.
	sessionDirs, err := os.ReadDir(filepath.Join(root, "sessions"))
	if err != nil {
		return nil, 0, 0, err
	}
	for _, d := range sessionDirs {
		var cfg WorkerConfig
		path := filepath.Join(root, "sessions", d.Name(), "worker-config.json")
		if err = readJSON(path, &cfg); err != nil {
			return nil, 0, 0, err
		}
		if cfg.Manifest != b || cfg.RunRoot != root {
			return nil, 0, 0, errors.New("recovery worker identity differs")
		}
		if cfg.Parent.PID != os.Getpid() && sameProcess(cfg.Parent) {
			return nil, 0, 0, errors.New("another live driver still owns replay")
		}
		if _, err = os.Stat(filepath.Join(filepath.Dir(path), "closed.json")); err == nil {
			continue
		}
		if cfg.Schema != recordSchema || !strings.HasPrefix(cfg.Unit, "buildopt-replay-"+digest([]byte(root))[:16]+"-") {
			return nil, 0, 0, errors.New("unowned recovery service")
		}
		// The cgroup path was observed in every native receipt. Services with
		// no child yet can be queried by their exact manifest-derived unit.
		group, err := ownedUnitCgroup(cfg.Unit)
		if err != nil {
			return nil, 0, 0, err
		}
		if group == "" {
			// The watchdog may already have unloaded this service. Recover the exact
			// observed cgroup from its own immutable native receipts, never by PID alone.
			files, _ := filepath.Glob(filepath.Join(root, "attempts", "*", "native-finish.json"))
			for _, file := range files {
				var native ProcessReceipt
				if err = readJSON(file, &native); err != nil {
					return nil, 0, 0, err
				}
				if native.Unit == cfg.Unit {
					group = native.Supervisor.Cgroup
					break
				}
			}
			if group == "" {
				continue
			} // No child receipt exists; the attempt remains incomplete.
		}
		s := workerSession{cfg, path, group}
		if err = s.stop(); err != nil {
			return nil, 0, 0, err
		}
	}
	// An observed safety stop is a result, not an infrastructure crash that may
	// be retried until it disappears. Only a pause or missing terminal boundary
	// can enter the declared recovery path.
	for rep := 1; rep <= m.Replications; rep++ {
		_, end, e := latestBoundary(root, rep)
		if e == nil && end.Status != "PAUSED" && end.Status != "COMPLETE" {
			return nil, 0, 0, fmt.Errorf("INCOMPLETE_EVIDENCE: recorded %s stop cannot be retried", end.Status)
		}
	}
	if !absentBinding(m.Observer) {
		// Completed observation is part of the checkpoint's proof. Reject drift
		// before any new workflow, not after it has consumed another ordinal.
		ends, err := filepath.Glob(filepath.Join(root, "attempts", "*", "end.json"))
		if err != nil {
			return nil, 0, 0, err
		}
		for _, path := range ends {
			var end AttemptEnd
			if err = readJSON(path, &end); err != nil {
				return nil, 0, 0, err
			}
			if absentBinding(end.Native) {
				continue
			}
			if err = checkBinding(end.Native); err != nil {
				return nil, 0, 0, err
			}
			var native ProcessReceipt
			if err = readJSON(end.Native.Path, &native); err != nil {
				return nil, 0, 0, err
			}
			if err = checkObservation(m, filepath.Dir(path), native); err != nil {
				return nil, 0, 0, err
			}
		}
	}
	var checkpoint Checkpoint
	checkpointErr := readJSON(filepath.Join(root, "checkpoint.json"), &checkpoint)
	rep, ordinal := 1, 0
	if checkpointErr == nil {
		if checkpoint.Schema != recordSchema || checkpoint.Manifest != b || len(checkpoint.States) != 2 {
			return nil, 0, 0, errors.New("checkpoint identity differs")
		}
		if err = checkCheckpoint(m, checkpoint); err != nil {
			return nil, 0, 0, err
		}
		if err = checkBinding(checkpoint.LastPair); err != nil {
			return nil, 0, 0, err
		}
		r.lastPair = checkpoint.LastPair.SHA256
		rep = checkpoint.Replication
		ordinal = checkpoint.LastOrdinal + 1
		for _, pin := range checkpoint.States {
			if err = checkBinding(pin); err != nil {
				return nil, 0, 0, err
			}
			var state StatePin
			if err = readJSON(pin.Path, &state); err != nil {
				return nil, 0, 0, err
			}
			key := armKey(rep, state.Arm)
			r.states[key] = state
			r.stateBindings[key] = pin
			r.builds[key] = map[string]bool{}
		}
	} else if !os.IsNotExist(checkpointErr) {
		return nil, 0, 0, checkpointErr
	}
	if ordinal > m.ExecutionEnd {
		rep++
		ordinal = 0
		if rep > m.Replications {
			return nil, 0, 0, errors.New("replay already reached its full horizon")
		}
	}
	if m.DaemonPolicy == "REPLICATION" {
		at := stamp()
		closed, costErr := closeInterruptedCosts(root, at)
		if costErr != nil {
			return nil, 0, 0, costErr
		}
		if _, err = saveBound(filepath.Join(root, "archives", fmt.Sprintf("warm-loss-r%d-%03d.json", rep, ordinal)), struct {
			Schema, Reason string
			At             Stamp
			ClosedCosts    []string
		}{recordSchema, "warm daemon memory lost; unfinished request charged through recovery refusal", at, closed}); err != nil {
			return nil, 0, 0, err
		}
		if err = endReplication(root, rep, ordinal, harnessInvalid, "lost warm daemon session cannot reconstruct memory"); err != nil {
			return nil, 0, 0, err
		}
		return nil, 0, 0, errors.New("INCOMPLETE_EVIDENCE: lost warm daemon session cannot reconstruct memory")
	}
	if _, err = os.Stat(filepath.Join(root, "pairs", slotKey(rep, ordinal)+".json")); err == nil {
		if err = endReplication(root, rep, ordinal, harnessInvalid, "sealed pair lacks its completed checkpoint; no automatic duplicate attempt"); err != nil {
			return nil, 0, 0, err
		}
		return nil, 0, 0, errors.New("INCOMPLETE_EVIDENCE: sealed pair without completed checkpoint")
	} else if !os.IsNotExist(err) {
		return nil, 0, 0, err
	}
	attemptDirs, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return nil, 0, 0, err
	}
	abandoned := []AttemptStart{}
	for _, d := range attemptDirs {
		var a AttemptStart
		if err = readJSON(filepath.Join(root, "attempts", d.Name(), "start.json"), &a); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, 0, 0, err
		}
		r.attempts++
		r.gradleReservations += a.ReservedGradleStarts
		if a.Replication == rep && a.Ordinal == ordinal {
			abandoned = append(abandoned, a)
		}
		var capture Capture
		if err = readJSON(filepath.Join(root, "attempts", d.Name(), "capture.json"), &capture); err == nil {
			key := armKey(a.Replication, a.Arm)
			if r.builds[key] == nil {
				r.builds[key] = map[string]bool{}
			}
			for _, build := range capture.GradleBuilds {
				r.builds[key][build.ID] = true
			}
		}
	}
	recoveries, err := filepath.Glob(filepath.Join(root, "archives", "recovery-*.json"))
	if err != nil {
		return nil, 0, 0, err
	}
	r.retries = len(recoveries)
	if len(abandoned) == 0 {
		for _, state := range r.states {
			if err = verifyState(m, state); err != nil {
				return nil, 0, 0, fmt.Errorf("INCOMPLETE_EVIDENCE: no exact recoverable pre-pair state: %w", err)
			}
		}
		return r, rep, ordinal, nil
	}
	if r.retries >= m.Limits.RetryPairs {
		return nil, 0, 0, errors.New("NOT_RUN_LIMIT: retry reserve exhausted")
	}
	sort.Slice(abandoned, func(i, j int) bool { return abandoned[i].At.NS < abandoned[j].At.NS })
	latestGeneration := abandoned[len(abandoned)-1].Generation
	latest := []AttemptStart{}
	for _, attempt := range abandoned {
		if attempt.Generation == latestGeneration {
			latest = append(latest, attempt)
		}
	}
	abandoned = latest
	for _, a := range abandoned {
		var native ProcessReceipt
		path := filepath.Join(root, "attempts", a.ID, "native-finish.json")
		if err = readJSON(path, &native); err == nil && native.ExitCode != 0 && native.Error != "driver disappeared" {
			return nil, 0, 0, errors.New("candidate/native error is not an infrastructure retry")
		}
		var end AttemptEnd
		if err = readJSON(filepath.Join(root, "attempts", a.ID, "end.json"), &end); err == nil && (end.Class == candidateFailure || end.Class == nativeFailure) {
			return nil, 0, 0, errors.New("failed candidate/native result cannot be retried")
		}
	}
	key := slotKey(rep, ordinal)
	snapshotPath := filepath.Join(root, "snapshots", key, "snapshot.json")
	var snapshot pairSnapshot
	if err = readJSON(snapshotPath, &snapshot); err != nil {
		return nil, 0, 0, fmt.Errorf("INCOMPLETE_EVIDENCE: equal pre-attempt snapshot unavailable: %w", err)
	}
	if snapshot.Schema != recordSchema || snapshot.Replication != rep || snapshot.Ordinal != ordinal || len(snapshot.States) != 2 || len(snapshot.Copies) != 2 || len(snapshot.GitMetadata) != 2 {
		return nil, 0, 0, errors.New("invalid equal-state snapshot")
	}
	for _, pin := range append(append(append([]Binding{}, snapshot.States...), snapshot.Copies...), snapshot.GitMetadata...) {
		if err = checkBinding(pin); err != nil {
			return nil, 0, 0, err
		}
	}
	recovery := Recovery{Schema: recordSchema, Replication: rep, Ordinal: ordinal, Generation: abandoned[len(abandoned)-1].Generation + 1, AbandonedAttempts: []Binding{}, ArchivedStates: []Binding{}, RestoredStates: []Binding{}, ClosedCosts: []string{}, Reason: "Interrupted driver; both quiescent arms restored; failed candidate envelopes retained without saving", At: stamp()}
	recovery.Snapshot, err = bind(snapshotPath)
	if err != nil {
		return nil, 0, 0, err
	}
	// If the driver died before a phase receipt, charge the entire interval to
	// recovery as a conservative upper bound. Keep the original start and raw
	// partial receipt; the recovery record identifies every such closure.
	recovery.ClosedCosts, err = closeInterruptedCosts(root, recovery.At)
	if err != nil {
		return nil, 0, 0, err
	}
	for _, a := range abandoned {
		pin, err := bind(filepath.Join(root, "attempts", a.ID))
		if err != nil {
			return nil, 0, 0, err
		}
		recovery.AbandonedAttempts = append(recovery.AbandonedAttempts, pin)
	}
	archive := filepath.Join(root, "archives", fmt.Sprintf("%s-g%d", key, recovery.Generation))
	if err = os.Mkdir(archive, 0700); err != nil {
		return nil, 0, 0, err
	}
	for index, arm := range []string{"N", "I"} {
		var state StatePin
		if err = readJSON(snapshot.States[index].Path, &state); err != nil {
			return nil, 0, 0, err
		}
		if state.Arm != arm || state.Replication != rep || state.Ordinal != ordinal {
			return nil, 0, 0, errors.New("snapshot arm/ordinal drift")
		}
		gd, err := gitAt(filepath.Join(state.Root, "repo"), "rev-parse", "--absolute-git-dir")
		if err != nil {
			return nil, 0, 0, err
		}
		meta := strings.TrimSpace(string(gd))
		if !inside(filepath.Join(m.CommonGit, "worktrees"), meta) {
			return nil, 0, 0, errors.New("recovery refuses unowned Git metadata")
		}
		if err = os.Rename(state.Root, filepath.Join(archive, arm)); err != nil {
			return nil, 0, 0, err
		}
		if err = os.Rename(meta, filepath.Join(archive, arm+"-git")); err != nil {
			return nil, 0, 0, err
		}
		if err = copyTree(snapshot.Copies[index].Path, state.Root); err != nil {
			return nil, 0, 0, err
		}
		if err = copyTree(snapshot.GitMetadata[index].Path, meta); err != nil {
			return nil, 0, 0, err
		}
		if err = verifyState(m, state); err != nil {
			return nil, 0, 0, err
		}
		pin, err := bind(filepath.Join(archive, arm))
		if err != nil {
			return nil, 0, 0, err
		}
		recovery.ArchivedStates = append(recovery.ArchivedStates, pin)
		observed, err := captureState(m, rep, arm, ordinal, state.Previous, state.BaselineApplied, state.CandidateApplied)
		if err != nil {
			return nil, 0, 0, err
		}
		restored, err := saveBound(filepath.Join(archive, arm+"-restored.json"), observed)
		if err != nil {
			return nil, 0, 0, err
		}
		recovery.RestoredStates = append(recovery.RestoredStates, restored)
		k := armKey(rep, arm)
		r.states[k] = state
		r.stateBindings[k] = snapshot.States[index]
	}
	r.retries++
	r.generations[key] = recovery.Generation
	if _, err = saveBound(filepath.Join(root, "archives", fmt.Sprintf("recovery-%03d.json", r.retries)), recovery); err != nil {
		return nil, 0, 0, err
	}
	return r, rep, ordinal, nil
}

func retryCharge(root string, rep, ordinal, generation int) (int64, error) {
	if generation == 0 {
		return 0, nil
	}
	files, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return 0, err
	}
	var total int64
	for _, file := range files {
		var a AttemptStart
		if err = readJSON(filepath.Join(root, "attempts", file.Name(), "start.json"), &a); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return 0, err
		}
		if a.Replication != rep || a.Ordinal != ordinal || a.Generation >= generation || a.Arm != "I" {
			continue
		}
		var c Cost
		if err = readJSON(filepath.Join(root, "costs", a.ID+"-request.json"), &c); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return 0, err
		}
		total += c.DurationNS
	}
	return total, nil
}
