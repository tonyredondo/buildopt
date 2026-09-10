//go:build linux && amd64 && replay_integration && replay_gradle

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type OverheadSample struct {
	Arm    string        `json:"arm"`
	Pair   int           `json:"pair"`
	Style  string        `json:"style"`
	Warmup bool          `json:"warmup"`
	Begin  Stamp         `json:"begin"`
	End    Stamp         `json:"end"`
	Native Binding       `json:"native"`
	Graph  Binding       `json:"graph"`
	Logs   []Binding     `json:"logs"`
	Builds []GradleBuild `json:"builds"`
}

func TestGradleSymmetricCaptureOverhead(t *testing.T) {
	m := gradleFixture(t, 1, "tasks.register('replay')\n")
	measureCaptureOverhead(t, m, false)
}

func measureCaptureOverhead(t *testing.T, m Manifest, owner bool) {
	measureCaptureOverheadDesign(t, m, owner, 1, 4, "legacy")
}

// Precision runs keep the same native command, capture and receipts. Only the
// prospectively declared warmup/sample counts and qualification label differ.
func measureCaptureOverheadDesign(t *testing.T, m Manifest, owner bool, warmupCycles, pairsPerArm int, mode string) {
	t.Helper()
	totalStarts := 4 * (warmupCycles + pairsPerArm)
	if warmupCycles < 1 || pairsPerArm < 1 || (mode != "legacy" && mode != "precision-pilot" && mode != "precision-confirmation") {
		t.Fatal("invalid frozen overhead design")
	}
	m.Limits.MaxWorkflowStarts = totalStarts
	m.Limits.MaxGradleStarts = totalStarts
	r := runFixture(t, m)
	if err := beginReplication(m.RunRoot, 1); err != nil {
		t.Fatal(err)
	}
	status := "INCOMPLETE"
	defer func() {
		if err := endReplication(m.RunRoot, 1, 0, status, "capture overhead qualification only"); err != nil {
			t.Error(err)
		}
	}()
	if err := r.initialize(1); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(m.RunRoot, "overhead")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if owner {
		repo := filepath.Join(m.RunRoot, "r1", "I", "repo")
		ok, err := patchApplicable(repo, m.Candidate)
		if err != nil || !ok {
			t.Fatalf("owner candidate is not applicable: %v", err)
		}
		if err := applyPatch(repo, m.Candidate); err != nil {
			t.Fatal(err)
		}
	}
	// Declare all orders, warmups, metrics and limits before either native arm.
	plan := struct {
		Schema, Policy, Order, Scope                          string
		WarmupsPerArm, MeasuredPairsPerArm, TotalGradleStarts int
		MaximumWrapperNS, MaximumArmImbalanceNS               int64
		Manifest                                              Binding
	}{
		"buildopt.history-replay/overhead-plan/v1", "symmetric lean graph/outcomes; deep tracing diagnostic only", "Two warmups per arm with opposite style order; four measured pairs per arm; arm and style order alternate", "Small no-action native fixture only; BV-006 repeats qualification on the frozen owner workflow", 2, 4, 20, 10e6, 100e6, r.binding}
	if owner {
		plan.Scope = "Frozen Elasticsearch :server:precommit with exact native baseline and C5 candidate; no source evolution; full warmups retained, no value claim"
	}
	if mode != "legacy" {
		plan.Schema = "buildopt.history-replay/overhead-plan/v2"
		plan.Policy = mode + "; original capture and point gates; separate registered precision checker required"
		plan.Order = "Balanced warmup cycles; measured arm and style order alternate; no sample exclusion or optional success"
		plan.WarmupsPerArm = 2 * warmupCycles
		plan.MeasuredPairsPerArm = pairsPerArm
		plan.TotalGradleStarts = totalStarts
	}
	if err := writeExclusive(filepath.Join(root, "plan.json"), jsonBytes(plan), 0600); err != nil {
		t.Fatal(err)
	}
	workers := map[string]workerSession{}
	seen := map[string]map[string]bool{"N": {}, "I": {}}
	for index, arm := range []string{"N", "I"} {
		dir := filepath.Join(m.RunRoot, "sessions", "overhead-"+arm)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		worker, err := startWorker(m, r.binding, 1, arm, index+1, dir)
		if err != nil {
			t.Fatal(err)
		}
		workers[arm] = worker
		t.Cleanup(func() {
			if err := worker.stop(); err != nil {
				t.Error(err)
			}
		})
	}
	samples := []OverheadSample{}
	sequence := 0
	request := func(arm string, pair int, style string, warm bool) {
		t.Helper()
		sequence++
		if sequence > totalStarts {
			t.Fatal("prospective overhead start allocation exceeded")
		}
		if err := r.guard(); err != nil {
			t.Fatal(err)
		}
		id := fmt.Sprintf("overhead-%02d-%s-%s", sequence, arm, style)
		dir := filepath.Join(m.RunRoot, "attempts", id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		args, env := workflow(m, 1, arm, dir)
		if style == "plain" {
			filtered := []string{}
			for j := 0; j < len(args); j++ {
				if args[j] == "--init-script" {
					j++
					continue
				}
				filtered = append(filtered, args[j])
			}
			args = filtered
		}
		q := ProcessRequest{recordSchema, id, filepath.Join(m.RunRoot, "r1", arm, "repo"), args, env, dir, m.Limits.MaxRequestNS}
		if err := writeExclusive(filepath.Join(dir, "reservation.json"), jsonBytes(q), 0600); err != nil {
			t.Fatal(err)
		}
		if owner {
			patches := []Patch{m.Baseline}
			if arm == "I" {
				patches = append(patches, m.Candidate)
			}
			if _, err := verifySource(m.CommonGit, q.Directory, m.History[0].Commit, patches, m.GeneratedPaths); err != nil {
				t.Fatal(err)
			}
		}
		sample := OverheadSample{Arm: arm, Pair: pair, Style: style, Warmup: warm, Begin: stamp(), Logs: []Binding{}, Builds: []GradleBuild{}}
		native, err := workers[arm].invoke(q, func(ProcessReceipt) { sample.End = stamp() })
		if err != nil {
			t.Fatal(err)
		}
		if native.ExitCode != 0 || native.Outcome != "EXITED" {
			t.Fatalf("overhead native failure: %+v", native)
		}
		if err := r.guard(); err != nil {
			t.Fatal(err)
		}
		sample.Native = bound(t, filepath.Join(dir, "native-finish.json"))
		sample.Logs, sample.Builds, err = collectGradleLogs(filepath.Join(m.RunRoot, "r1", arm), dir, native, seen[arm])
		if err != nil {
			t.Fatal(err)
		}
		c := Capture{DaemonLogs: sample.Logs, GradleBuilds: sample.Builds}
		if _, err = checkGradleCommands(m, c, native, filepath.Join(m.RunRoot, "r1", arm), seen[arm]); err != nil {
			t.Fatal(err)
		}
		if len(sample.Builds) != 1 {
			t.Fatal("overhead invocation hidden or duplicated")
		}
		seen[arm][sample.Builds[0].ID] = true
		if style == "lean" {
			path := filepath.Join(dir, "graph.jsonl")
			_, outcomes, err := readGraphScope(path, q.Directory, owner)
			if err != nil {
				t.Fatal(err)
			}
			if owner {
				found := false
				for _, task := range outcomes {
					if task.Outcome == "FAILED" {
						t.Fatal("owner task failed during capture qualification")
					}
					found = found || task.Identity == ":server:precommit"
				}
				if !found {
					t.Fatal("owner workflow outcome missing")
				}
			} else if len(outcomes) != 1 || outcomes[0].Action {
				t.Fatal("overhead fixture unexpectedly executes work")
			}
			sample.Graph = bound(t, path)
		} else {
			out, err := os.ReadFile(filepath.Join(dir, "stdout.log"))
			valid := strings.Contains(string(out), ":replay UP-TO-DATE")
			if owner {
				valid = strings.Contains(string(out), "BUILD SUCCESSFUL") && strings.Contains(string(out), ":server:precommit")
			}
			if err != nil || !valid {
				t.Fatalf("uninstrumented no-action path not observed: %v %s", err, out)
			}
		}
		if err = writeExclusive(filepath.Join(root, fmt.Sprintf("sample-%02d.json", sequence)), jsonBytes(sample), 0600); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, sample)
	}
	for cycle := 0; cycle < warmupCycles; cycle++ {
		request("N", -1-cycle, "plain", true)
		request("I", -1-cycle, "lean", true)
		request("I", -1-cycle, "plain", true)
		request("N", -1-cycle, "lean", true)
	}
	for pair := 0; pair < pairsPerArm; pair++ {
		for _, arm := range armOrder(1, pair) {
			offset := 0
			if arm == "I" {
				offset = 1
			}
			styles := []string{"plain", "lean"}
			if (pair+offset)%2 == 1 {
				styles = []string{"lean", "plain"}
			}
			for _, style := range styles {
				request(arm, pair, style, false)
			}
		}
	}
	for _, worker := range workers {
		if err := worker.stop(); err != nil {
			t.Fatal(err)
		}
	}
	if sequence != totalStarts {
		t.Fatal("overhead allocation incomplete")
	}
	if owner {
		for _, arm := range []string{"N", "I"} {
			patches := []Patch{m.Baseline}
			if arm == "I" {
				patches = append(patches, m.Candidate)
			}
			repo := filepath.Join(m.RunRoot, "r1", arm, "repo")
			if _, err := verifySource(m.CommonGit, repo, m.History[0].Commit, patches, m.GeneratedPaths); err != nil {
				t.Fatal(err)
			}
			state, err := captureState(m, 1, arm, 0, "", true, arm == "I")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = saveBound(filepath.Join(root, "final-state-"+arm+".json"), state); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Re-read native bytes and both monotonic boundaries to derive measurements.
	diffs := map[string][]int64{"N": {}, "I": {}}
	wrappers := []int64{}
	for _, arm := range []string{"N", "I"} {
		for pair := 0; pair < pairsPerArm; pair++ {
			values := map[string]int64{}
			for _, sample := range samples {
				if sample.Arm != arm || sample.Pair != pair || sample.Warmup {
					continue
				}
				if err := checkBinding(sample.Native); err != nil {
					t.Fatal(err)
				}
				var native ProcessReceipt
				if err := readJSON(sample.Native.Path, &native); err != nil {
					t.Fatal(err)
				}
				if err := checkStampInterval(sample.Begin, sample.End); err != nil {
					t.Fatal(err)
				}
				duration := native.End.NS - native.Start.NS
				values[sample.Style] = duration
				wrappers = append(wrappers, sample.End.NS-sample.Begin.NS-duration)
			}
			if len(values) != 2 {
				t.Fatal("missing symmetric measurement")
			}
			diffs[arm] = append(diffs[arm], values["lean"]-values["plain"])
		}
	}
	nMedian, iMedian := nearestRank(diffs["N"], 50), nearestRank(diffs["I"], 50)
	wrapperP95 := nearestRank(wrappers, 95)
	imbalance := int64(math.Abs(float64(nMedian - iMedian)))
	passed := "QUALIFIED_SMALL_FIXTURE"
	if owner {
		passed = "QUALIFIED_OWNER"
	}
	verdict := passed
	if wrapperP95 > plan.MaximumWrapperNS || imbalance > plan.MaximumArmImbalanceNS {
		verdict = "OWNER_QUALIFICATION_REQUIRED"
	}
	schema := "buildopt.history-replay/overhead-result/v1"
	if mode != "legacy" {
		schema = "buildopt.history-replay/overhead-result/v2"
		if mode == "precision-pilot" {
			verdict = "PRECISION_PILOT_ONLY"
		} else if verdict == passed {
			verdict = "POINT_GATES_PASS_PRECISION_CHECK_REQUIRED"
		}
	}
	result := struct {
		Schema, Decision                                                          string
		NativeSamples, CandidateSamples                                           []int64
		NativeMedianExtraNS, CandidateMedianExtraNS, ArmImbalanceNS, WrapperP95NS int64
		GradleStarts                                                              int
	}{schema, verdict, diffs["N"], diffs["I"], nMedian, iMedian, imbalance, wrapperP95, sequence}
	if err := writeExclusive(filepath.Join(root, "result.json"), jsonBytes(result), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("frozen overhead result: %s; N lean median extra %s, I %s, arm imbalance %s, wrapper p95 %s", verdict, time.Duration(nMedian), time.Duration(iMedian), time.Duration(imbalance), time.Duration(wrapperP95))
	if verdict == "OWNER_QUALIFICATION_REQUIRED" {
		t.Fatal("instrument overhead exceeds prospectively declared envelope")
	}
	status = "COMPLETE"
}
