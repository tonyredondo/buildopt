//go:build linux && amd64 && replay_integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDriverDeathKillsDetachedTreeAndRetainsCandidateCost(t *testing.T) {
	m := runnableFixture(t, 3, "detached-after-anchor")
	m.DaemonPolicy = "REPLICATION"
	m.Limits.RetryPairs = 0
	m.Limits.MaxRequestNS = int64(20 * time.Second)
	root := filepath.Dir(m.RunRoot)
	path := filepath.Join(root, "fixture-manifest.json")
	mustWrite(t, path, jsonBytes(m))
	log, err := os.OpenFile(filepath.Join(root, "driver.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	child := exec.Command(m.Executable.Path, "fixture-driver", path, "none", "none")
	child.Stdout = log
	child.Stderr = log
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if child.ProcessState == nil {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	})
	evidence := filepath.Join(m.RunRoot, "attempts/r1-001-g0-I")
	var detached ProcessIdentity
	ready := false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		if err = readJSON(filepath.Join(evidence, "detached.json"), &detached); err == nil {
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatal("actual candidate descendant was not started")
	}
	parent, err := processIdentity(child.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "killed-driver.json"), jsonBytes(parent))
	if err = child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err = child.Wait(); err == nil {
		t.Fatal("driver did not actually die")
	}
	type terminationObservation struct {
		ElapsedNS       int64
		CgroupProcesses []ProcessIdentity
		CgroupError     string
		SameProcess     bool
		ProcStat        string
		ProcError       string
	}
	var observations []terminationObservation
	terminationStart := time.Now()
	defer func() { mustWrite(t, filepath.Join(root, "termination-observations.json"), jsonBytes(observations)) }()
	stopped := false
	for deadline := time.Now().Add(6 * time.Second); time.Now().Before(deadline); {
		ps, e := cgroupProcesses(detached.Cgroup)
		stat, statErr := os.ReadFile("/proc/" + strconv.Itoa(detached.PID) + "/stat")
		observation := terminationObservation{ElapsedNS: time.Since(terminationStart).Nanoseconds(), CgroupProcesses: ps, SameProcess: sameProcess(detached), ProcStat: string(stat)}
		if e != nil {
			observation.CgroupError = e.Error()
		}
		if statErr != nil {
			observation.ProcError = statErr.Error()
		}
		observations = append(observations, observation)
		// Cgroup membership and /proc identity disappear asynchronously. Wait
		// for both within the original deadline; an empty group alone is not
		// proof that the observed descendant has been reaped.
		if (os.IsNotExist(e) || e == nil && len(ps) == 0) && !observation.SameProcess {
			stopped = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !stopped || sameProcess(detached) {
		t.Fatalf("detached descendant survived driver death: %+v", observations[len(observations)-1])
	}
	if _, _, _, err = resumeRunner(m.RunRoot); err == nil || !strings.Contains(err.Error(), "warm daemon") {
		t.Fatalf("lost JVM memory accepted: %v", err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 3 || result.Slots[0][1].CandidateNS <= 0 || result.Slots[0][2].Class != notRunDependency || result.Decision != "INCOMPLETE_EVIDENCE" {
		t.Fatalf("lost candidate attempt received free/duplicate credit: %+v", result)
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
	t.Logf("killed driver PID %d; detached PID %d terminated; partial candidate charged %s", parent.PID, detached.PID, time.Duration(result.Slots[0][1].CandidateNS))
}
