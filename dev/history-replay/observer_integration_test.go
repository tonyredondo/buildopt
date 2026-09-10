//go:build linux && amd64 && replay_integration

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The observed manifest must reach the real command parser; an unknown field
// or silently ignored policy would make unobserved evidence look qualified.
func TestObservedManifestReachesCLI(t *testing.T) {
	m := runnableFixture(t, 1, "success")
	policy := filepath.Join(filepath.Dir(m.RunRoot), "observer-policy.json")
	mustWrite(t, policy, []byte(`{"schema":"buildopt.history-replay/observer-policy/v1","samplerAffinity":"8","heartbeatAffinity":"10","sampleNS":100000000,"maxGapNS":500000000,"pendingRecords":128,"pendingBytes":16777216,"frameBytes":524288,"totalBytes":536870912,"drainNS":5000000000,"maxRows":10000}`))
	var raw map[string]any
	if err := json.Unmarshal(jsonBytes(m), &raw); err != nil {
		t.Fatal(err)
	}
	raw["schema"] = "buildopt.history-replay/manifest/v3"
	raw["observer"] = bound(t, policy)
	path := filepath.Join(filepath.Dir(m.RunRoot), "observed-manifest.json")
	mustWrite(t, path, jsonBytes(raw))
	if err := command([]string{"validate", path}); err != nil {
		t.Fatalf("observed manifest rejected by real CLI: %v", err)
	}
	if _, err := os.Stat(m.RunRoot); !os.IsNotExist(err) {
		t.Fatal("validate executed the workflow")
	}
	for _, field := range []string{"observer", "schema"} {
		changed := map[string]any{}
		for k, v := range raw {
			changed[k] = v
		}
		if field == "observer" {
			delete(changed, "observer")
		} else {
			changed["schema"] = manifestSchema
		}
		mustWrite(t, path, jsonBytes(changed))
		if err := command([]string{"validate", path}); err == nil {
			t.Fatal("accepted absent/implicit observer policy")
		}
	}
	mustWrite(t, path, jsonBytes(raw))
	var p map[string]any
	data, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"maxGapNS", "samplerAffinity", "pendingBytes"} {
		changed := map[string]any{}
		for k, v := range p {
			changed[k] = v
		}
		switch field {
		case "maxGapNS":
			changed[field] = 1000000000
		case "samplerAffinity":
			changed[field] = "0"
		case "pendingBytes":
			delete(changed, field)
		}
		mustWrite(t, policy, jsonBytes(changed))
		raw["observer"] = bound(t, policy)
		mustWrite(t, path, jsonBytes(raw))
		if err := command([]string{"validate", path}); err == nil {
			t.Fatal("accepted weakened or incomplete observation policy")
		}
	}
}

// Fault injection exists only in the test binary. The public observer policy
// has no fault field and production command dispatch always uses its bounds.
func runObserverFaultHelper(path, hash, fault string) error {
	if err := checkBinding(Binding{path, hash}); err != nil {
		return err
	}
	var cfg ObserverLaunch
	if err := readJSON(path, &cfg); err != nil {
		return err
	}
	p, err := readObserverPolicy(Manifest{Schema: observedManifestSchema, Observer: cfg.Policy, Affinity: cfg.NativeAffinity})
	if err != nil {
		return err
	}
	config := p.writer()
	switch fault {
	case "writer-error":
		config.Fault = "error"
	case "short-write":
		config.Fault = "short"
	case "writer-exit":
		config.Fault = "exit"
	case "drain-timeout":
		config.Fault = "stall"
	case "record-saturation":
		config.Fault = "delay"
	case "scan-error":
		cfg.Cgroup = "/buildopt-nonexistent-observer-fixture"
	}
	return runProcessObserver(cfg, p, samplerOptions{maxRows: p.MaxRows, maxBytes: p.TotalBytes, heartbeatAffinity: p.HeartbeatAffinity, buffered: &config})
}

func awaitObserverFile(t *testing.T, path string, out any) {
	t.Helper()
	for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); {
		if err := readJSON(path, out); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("observer evidence did not appear: " + path)
}

func startObservedCLI(t *testing.T, m Manifest, extraEnvironment ...string) *exec.Cmd {
	t.Helper()
	path := filepath.Join(filepath.Dir(m.RunRoot), "manifest.json")
	mustWrite(t, path, jsonBytes(m))
	log, err := os.OpenFile(filepath.Join(filepath.Dir(m.RunRoot), "driver.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	c := exec.Command(m.Executable.Path, "run", path)
	c.Stdout = log
	c.Stderr = log
	c.Env = append(os.Environ(), extraEnvironment...)
	if err = c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if c.ProcessState == nil {
			c.Process.Kill()
			c.Wait()
		}
	})
	return c
}

func TestObservedWriterPauseKeepsCustomerBoundary(t *testing.T) {
	m := observedFixture(t, 1)
	c := startObservedCLI(t, m)
	path := filepath.Join(m.RunRoot, "attempts/r1-000-g0-N/proc-samples.jsonl")
	var writer ProcessIdentity
	awaitObserverFile(t, path+".writer-identity.json", &writer)
	if err := syscall.Kill(writer.PID, syscall.SIGSTOP); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sameProcess(writer) {
			syscall.Kill(writer.PID, syscall.SIGCONT)
		}
	})
	time.Sleep(2104 * time.Millisecond)
	if err := syscall.Kill(writer.PID, syscall.SIGCONT); err != nil {
		t.Fatal(err)
	}
	if err := c.Wait(); err != nil {
		t.Fatal("paused-writer replay failed", err)
	}
	var report observerReport
	var end AttemptEnd
	var receipt ObservationReceipt
	for _, item := range []struct {
		path string
		out  any
	}{{path + ".phases.json", &report}, {filepath.Join(filepath.Dir(path), "end.json"), &end}, {filepath.Join(filepath.Dir(path), "observation.json"), &receipt}} {
		if err := readJSON(item.path, item.out); err != nil {
			t.Fatal(err)
		}
	}
	if report.Persistence.PeakPendingRecords < 3 || report.Persistence.Accepted != report.Persistence.Acknowledged || report.Persistence.PendingRecords != 0 {
		t.Fatal("sampler lost data or waited for persistence")
	}
	if receipt.End.NS-end.End.NS < int64(time.Second) || end.End.NS > report.End.BootNS {
		t.Fatal("writer drain contaminated customer completion")
	}
	if err := checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}

func TestObservedFailureStopsReplay(t *testing.T) {
	expected := map[string]string{"writer-error": "deliberate writer error", "short-write": "short write", "writer-exit": "writer acknowledgement", "drain-timeout": "writer drain deadline", "record-saturation": "pending record limit", "byte-saturation": "pending byte limit", "total-output-limit": "sampler byte limit", "scan-error": "no such file"}
	for _, name := range []string{"writer-error", "short-write", "writer-exit", "drain-timeout", "record-saturation", "byte-saturation", "total-output-limit", "scan-error"} {
		t.Run(name, func(t *testing.T) {
			m := observedFixture(t, 1)
			p, err := readObserverPolicy(m)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "drain-timeout":
				p.DrainNS = int64(200 * time.Millisecond)
			case "record-saturation":
				p.PendingRecords = 3
			case "byte-saturation":
				p.PendingBytes = 1
			case "total-output-limit":
				p.TotalBytes = 1
			}
			mustWrite(t, m.Observer.Path, jsonBytes(p))
			m.Observer = bound(t, m.Observer.Path)
			c := startObservedCLI(t, m, "BUILDOPT_OBSERVER_TEST_FAULT="+name)
			if err = c.Wait(); err == nil {
				t.Fatal("failed observation advanced replay")
			}
			dir := filepath.Join(m.RunRoot, "attempts/r1-000-g0-N")
			var receipt ObservationReceipt
			if err = readJSON(filepath.Join(dir, "observation.json"), &receipt); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(receipt.Error, expected[name]) {
				t.Fatalf("wrong failure: %q", receipt.Error)
			}
			if _, err = os.Stat(filepath.Join(m.RunRoot, "attempts/r1-000-g0-I")); !os.IsNotExist(err) {
				t.Fatal("replay continued after observer failure")
			}
			if sameProcess(receipt.Sampler) {
				t.Fatal("sampler survived failure")
			}
			var report observerReport
			if err = readJSON(filepath.Join(dir, "proc-samples.jsonl.phases.json"), &report); err != nil {
				t.Fatal(err)
			}
			if sameProcess(report.Persistence.Writer) {
				t.Fatal("writer survived failure")
			}
			var result RunResult
			if err = readJSON(filepath.Join(m.RunRoot, "result.json"), &result); err != nil {
				t.Fatal(err)
			}
			if result.Decision == "FIXTURE_VERIFIED" || result.WorkflowStarts != 1 {
				t.Fatal("failed observation credited as usable replay")
			}
			if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
				t.Fatal("checker could not retain incomplete result", err)
			}
		})
	}
}

func TestObservedCheckpointResume(t *testing.T) {
	m := observedFixture(t, 2)
	r := runFixture(t, m)
	if err := r.execute(1, 0, 0); err != nil {
		t.Fatal(err)
	}
	phase := filepath.Join(m.RunRoot, "attempts/r1-000-g0-N/proc-samples.jsonl.phases.json")
	original, err := os.ReadFile(phase)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(phase, []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, _, resumeErr := resumeRunner(m.RunRoot)
	if err = os.WriteFile(phase, original, 0600); err != nil {
		t.Fatal(err)
	}
	if resumeErr == nil {
		t.Fatal("resume accepted corrupted completed observation before launching new work")
	}
	next, rep, ordinal, err := resumeRunner(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if ordinal != 1 || rep != 1 {
		t.Fatal("wrong observed continuation")
	}
	if err = next.execute(rep, ordinal, -1); err != nil {
		t.Fatal(err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 4 || result.Decision != "FIXTURE_VERIFIED" {
		t.Fatal("resume duplicated/lost observed requests")
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}

// Reuse the CLI fixture to reject a launcher masquerading as daemon coverage.
func checkObservedDaemonBoundary(t *testing.T, m Manifest, dir string) {
	t.Helper()
	var native ProcessReceipt
	if err := readJSON(filepath.Join(dir, "native-finish.json"), &native); err != nil {
		t.Fatal(err)
	}
	if native.End.NS-native.Start.NS <= int64(500*time.Millisecond) {
		t.Fatal("fixture is too short to expose missing daemon coverage")
	}
	if err := checkObservation(m, dir, native); err != nil {
		t.Fatal(err)
	}
	m.Driver = "GRADLE"
	if err := checkObservation(m, dir, native); err == nil {
		t.Fatal("CLI snapshots substituted for absent Gradle daemon snapshots")
	}
}

func TestObservedDriverDeath(t *testing.T) {
	m := observedFixture(t, 2)
	m.Command[len(m.Command)-1] = "detached-after-anchor"
	m.DaemonPolicy = "REPLICATION"
	m.Limits.RetryPairs = 0
	m.Limits.MaxRequestNS = int64(20 * time.Second)
	c := startObservedCLI(t, m)
	dir := filepath.Join(m.RunRoot, "attempts/r1-001-g0-I")
	var descendant ProcessIdentity
	awaitObserverFile(t, filepath.Join(dir, "detached.json"), &descendant)
	var sampler struct {
		Identity         ProcessIdentity   `json:"identity"`
		ThreadAffinities map[string]string `json:"threadAffinities"`
	}
	awaitObserverFile(t, filepath.Join(dir, "proc-samples.jsonl.sampler.json"), &sampler)
	var writer ProcessIdentity
	awaitObserverFile(t, filepath.Join(dir, "proc-samples.jsonl.writer-identity.json"), &writer)
	if err := c.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := c.Wait(); err == nil {
		t.Fatal("driver survived kill")
	}
	for _, identity := range []ProcessIdentity{sampler.Identity, writer, descendant} {
		for deadline := time.Now().Add(8 * time.Second); sameProcess(identity) && time.Now().Before(deadline); {
			time.Sleep(10 * time.Millisecond)
		}
		if sameProcess(identity) {
			t.Fatalf("owned process survived driver death: %+v", identity)
		}
	}
	if _, _, _, err := resumeRunner(m.RunRoot); err == nil || !strings.Contains(err.Error(), "warm daemon") {
		t.Fatal("driver death fabricated warm state", err)
	}
	result, err := writeResult(m.RunRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowStarts != 3 || result.Decision != "INCOMPLETE_EVIDENCE" {
		t.Fatal("driver death lost attempt/cost")
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}

func observedFixture(t *testing.T, count int) Manifest {
	m := runnableFixture(t, count, "observed")
	policy := filepath.Join(filepath.Dir(m.RunRoot), "observer-policy.json")
	mustWrite(t, policy, []byte(`{"schema":"buildopt.history-replay/observer-policy/v1","samplerAffinity":"8","heartbeatAffinity":"10","sampleNS":100000000,"maxGapNS":500000000,"pendingRecords":128,"pendingBytes":16777216,"frameBytes":524288,"totalBytes":536870912,"drainNS":5000000000,"maxRows":10000}`))
	m.Schema = observedManifestSchema
	m.Observer = bound(t, policy)
	return m
}

// Detects an observer that exists in a fixture but is absent from run/check,
// or whose drain is charged to the customer rather than research.
func TestObservedReplayCLI(t *testing.T) {
	m := observedFixture(t, 2)
	path := filepath.Join(filepath.Dir(m.RunRoot), "manifest.json")
	mustWrite(t, path, jsonBytes(m))
	for _, args := range [][]string{{"validate", path}, {"run", path}, {"check", m.RunRoot}} {
		c := exec.Command(m.Executable.Path, args...)
		raw, err := c.CombinedOutput()
		mustWrite(t, filepath.Join(filepath.Dir(m.RunRoot), args[0]+".log"), raw)
		if err != nil {
			t.Fatalf("CLI %v: %v %s", args, err, raw)
		}
	}
	var result RunResult
	if err := readJSON(filepath.Join(m.RunRoot, "result.json"), &result); err != nil {
		t.Fatal(err)
	}
	if result.Decision != "FIXTURE_VERIFIED" || result.WorkflowStarts != 4 || result.ActualGradleStarts != 0 {
		t.Fatalf("unexpected replay result: %+v", result)
	}
	raw, err := os.ReadFile(filepath.Join(m.RunRoot, "observations.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "\"status\":\"COMPLETE\"") != 4 {
		t.Fatal("observation export omitted attempts")
	}
	paths, err := filepath.Glob(filepath.Join(m.RunRoot, "attempts", "*", "observation.json"))
	if err != nil || len(paths) != 4 {
		t.Fatal("missing actual observation receipts", err)
	}
	checkObservedDaemonBoundary(t, m, filepath.Dir(paths[0]))
	for _, path := range paths {
		var receipt ObservationReceipt
		if err = readJSON(path, &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.Error != "" || receipt.SamplerCPUNS <= 0 || receipt.WriterCPUNS <= 0 || receipt.HeartbeatCPUNS <= 0 {
			t.Fatal("missing process cost", receipt)
		}
		var native ProcessReceipt
		if err = readJSON(filepath.Join(filepath.Dir(path), "native-finish.json"), &native); err != nil {
			t.Fatal(err)
		}
		if native.ResourcesBefore.CPUAffinity != "8" || native.ResourcesAfter.CPUAffinity != "8" {
			t.Fatal("supervisor shares native CPUs or did not restore observer affinity")
		}
	}
	// Each mutation starts from the original evidence and must be rejected.
	for _, name := range []string{"proc-samples.jsonl", "proc-samples.jsonl.phases.json", "observation.json"} {
		path := filepath.Join(filepath.Dir(paths[0]), name)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, []byte("{}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err == nil {
			t.Fatal("accepted altered " + name)
		}
		if err = os.WriteFile(path, original, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = checkResult(m.RunRoot, filepath.Join(m.RunRoot, "result.json")); err != nil {
		t.Fatal(err)
	}
}
