//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func checkDriverBinary(m Manifest) error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	hash, err := fileDigest(path)
	if err != nil {
		return err
	}
	if hash != m.Executable.SHA256 {
		return errors.New("running driver differs from the frozen executable")
	}
	return nil
}

func checkCheckpoint(m Manifest, c Checkpoint) error {
	if c.Replication < 1 || c.Replication > m.Replications || c.LastOrdinal < 0 || c.LastOrdinal > m.ExecutionEnd || c.LastPair.Path != filepath.Join(m.RunRoot, "pairs", slotKey(c.Replication, c.LastOrdinal)+".json") {
		return errors.New("checkpoint refers to a future/different pair")
	}
	var pair PairRecord
	if err := checkBinding(c.LastPair); err != nil {
		return err
	}
	if err := readJSON(c.LastPair.Path, &pair); err != nil {
		return err
	}
	if pair.Schema != recordSchema || pair.Replication != c.Replication || pair.Ordinal != c.LastOrdinal || len(pair.Attempts) != 2 {
		return errors.New("checkpoint skips the sealed pair")
	}
	expected := map[string]Binding{}
	for _, binding := range pair.Attempts {
		if err := checkBinding(binding); err != nil {
			return err
		}
		var end AttemptEnd
		if err := readJSON(binding.Path, &end); err != nil {
			return err
		}
		var start AttemptStart
		if err := readJSON(filepath.Join(filepath.Dir(binding.Path), "start.json"), &start); err != nil {
			return err
		}
		if start.Replication != c.Replication || start.Ordinal != c.LastOrdinal {
			return errors.New("checkpoint attempt ordinal drift")
		}
		expected[start.Arm] = end.After
	}
	if len(expected) != 2 || !equalJSON(c.States, []Binding{expected["N"], expected["I"]}) {
		return errors.New("checkpoint state is not the sealed pair state")
	}
	remaining := []int{}
	for i := c.LastOrdinal + 1; i <= m.ExecutionEnd; i++ {
		remaining = append(remaining, i)
	}
	if !equalJSON(c.Remaining, remaining) {
		return errors.New("checkpoint omits scheduled work")
	}
	return checkStampInterval(pair.At, c.At)
}

func checkSession(m Manifest, a AttemptStart, native ProcessReceipt) error {
	dirs, err := os.ReadDir(filepath.Join(m.RunRoot, "sessions"))
	if err != nil {
		return err
	}
	found := false
	for _, d := range dirs {
		dir := filepath.Join(m.RunRoot, "sessions", d.Name())
		var cfg WorkerConfig
		if err = readJSON(filepath.Join(dir, "worker-config.json"), &cfg); err != nil {
			return err
		}
		if cfg.Unit != native.Unit {
			continue
		}
		if found {
			return errors.New("duplicate process ownership configuration")
		}
		found = true
		if cfg.Schema != recordSchema || cfg.Manifest.SHA256 != a.ManifestSHA256 || cfg.Executable != m.Executable || cfg.ArmRoot != filepath.Join(m.RunRoot, fmt.Sprintf("r%d", a.Replication), a.Arm) || cfg.RunRoot != m.RunRoot || cfg.Policy != m.DaemonPolicy || !equalJSON(cfg.Limits, m.Limits) {
			return errors.New("session configuration differs from frozen arm")
		}
		var closed SessionClosure
		if err = readJSON(filepath.Join(dir, "closed.json"), &closed); err != nil {
			return err
		}
		if closed.Schema != recordSchema || closed.Unit != native.Unit || closed.Cgroup != native.Supervisor.Cgroup || len(closed.Remaining) != 0 || closed.At.Boot != native.End.Boot || closed.At.NS < native.End.NS {
			return errors.New("session closure does not prove owned termination")
		}
	}
	if !found {
		return errors.New("missing owned session configuration")
	}
	return nil
}

func requirePhase(costs map[string]Cost, id, purpose string, rep, ordinal int, arm string) error {
	c, ok := costs[id]
	if !ok || c.Purpose != purpose || c.Replication != rep || c.Ordinal != ordinal || c.Arm != arm {
		return errors.New("missing/changed required phase: " + id)
	}
	return nil
}

// Count every attempt, including infrastructure failures and abandoned retries.
// A missing lifecycle never becomes an inferred zero-cost native invocation.
func accountInvocations(m Manifest, ordered []AttemptStart, result *RunResult) error {
	seen := map[string]bool{}
	for _, a := range ordered {
		dir := filepath.Join(m.RunRoot, "attempts", a.ID)
		if m.Driver != "GRADLE" {
			if a.ReservedGradleStarts != 0 {
				return errors.New("fixture claims Gradle starts")
			}
			continue
		}
		if a.ReservedGradleStarts != 1+m.Limits.NestedReservePerRequest {
			return errors.New("attempt changes nested reservation")
		}
		var native ProcessReceipt
		nativeErr := readJSON(filepath.Join(dir, "native-finish.json"), &native)
		var c Capture
		captureErr := readJSON(filepath.Join(dir, "capture.json"), &c)
		if os.IsNotExist(nativeErr) || os.IsNotExist(captureErr) {
			result.UnknownGradleReservations += a.ReservedGradleStarts
			continue
		}
		if nativeErr != nil {
			return nativeErr
		}
		if captureErr != nil {
			return captureErr
		}
		if c.Error != "" {
			// Retain a lower bound from the raw logs of an incomplete request. Coverage
			// and the remaining reservation explicitly prevent a positive decision.
			for _, log := range c.DaemonLogs {
				if err := checkBinding(log); err != nil {
					return err
				}
				builds, _ := readDaemonBuilds(log)
				for _, b := range builds {
					if !seen[b.ID] {
						seen[b.ID] = true
						result.ActualGradleStarts++
					}
				}
			}
			result.UnknownGradleReservations += a.ReservedGradleStarts
			continue
		}
		builds, err := checkGradleCommands(m, c, native, filepath.Join(m.RunRoot, fmt.Sprintf("r%d", a.Replication), a.Arm), seen)
		if err != nil {
			return err
		}
		if len(builds) > a.ReservedGradleStarts {
			return errors.New("unreserved nested workflow")
		}
		for _, build := range builds {
			seen[build.ID] = true
		}
		result.ActualGradleStarts += len(builds)
		result.NestedStarts += len(builds) - 1
	}
	return nil
}
