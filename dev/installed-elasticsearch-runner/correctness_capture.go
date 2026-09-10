package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type correctnessRowReceipt struct {
	RowID    string      `json:"rowId"`
	Complete fileBinding `json:"complete"`
}

type correctnessClosed struct {
	Unit   string `json:"unit"`
	Cgroup string `json:"cgroup"`
	Absent bool   `json:"absent"`
	EndNS  int64  `json:"endBootNanoseconds"`
}

func validateCorrectnessSequence(prior []correctnessRowReceipt, next string) error {
	steps := makeCorrectnessPlan().Steps[:12]
	if len(prior) >= len(steps) || steps[len(prior)].Row.ID != next {
		return errors.New("missing, duplicate or unallocated correctness row")
	}
	for i, r := range prior {
		if r.RowID != steps[i].Row.ID || filepath.Base(r.Complete.Path) != "row-complete.json" || filepath.Base(filepath.Dir(r.Complete.Path)) != r.RowID || !validSHA256(r.Complete.SHA256) {
			return errors.New("incomplete or reordered prior correctness row")
		}
	}
	return nil
}

func correctnessFailureSignature(log string) (string, error) {
	matches := regexp.MustCompile(`(?m)^\s*- (\S+) on line (\d+) of (\S+)\s*$`).FindAllStringSubmatch(log, -1)
	line := strings.Count(string(correctnessOwnerInput), "\n") + 1
	if !strings.Contains(log, "Found invalid patterns:") || len(matches) != 1 || matches[0][1] != "tab" || matches[0][2] != strconv.Itoa(line) || matches[0][3] != sourceInputs().OwnerInputPath {
		return "", errors.New("expected exact tab rule, owner source and appended line")
	}
	return strings.Join(matches[0][1:], ":"), nil
}

// Source validation permits only the reviewed task postimage and the allocated
// owner input. Installed bootstrap files are separately frozen, not ignored.
func verifyCorrectnessSource(ctx context.Context, f correctnessRuntime, step correctnessStep, expectedInput string) (string, error) {
	work := filepath.Join(f.Root, "arms", step.Row.Arm)
	for _, p := range []string{work, filepath.Join(work, sourceInputs().TaskPath), filepath.Join(work, sourceInputs().OwnerInputPath)} {
		if err := canonicalExisting(p); err != nil {
			return "", err
		}
	}
	for _, test := range []struct {
		args []string
		want string
	}{{[]string{"rev-parse", "--show-toplevel"}, work}, {[]string{"rev-parse", "--path-format=absolute", "--git-common-dir"}, f.CommonGit}, {[]string{"rev-parse", "HEAD"}, step.Row.Revision}} {
		got, err := nativeGit(ctx, work, test.args...)
		if err != nil || got != test.want {
			return "", errors.New("correctness worktree identity drift")
		}
	}
	changed, err := nativeGit(ctx, work, "diff", "--name-only", "HEAD")
	if err != nil {
		return "", err
	}
	allowed := map[string]bool{sourceInputs().TaskPath: step.Row.Arm != "N0", sourceInputs().OwnerInputPath: expectedInput != sourceInputs().OwnerInputSHA256}
	for _, name := range strings.Split(changed, "\n") {
		if name != "" && !allowed[name] {
			return "", fmt.Errorf("unexpected tracked source change: %s", name)
		}
	}
	index, err := nativeGit(ctx, work, "diff", "--cached", "--name-only")
	if err != nil || index != "" {
		return "", errors.New("candidate index must remain empty")
	}
	untracked, err := nativeGit(ctx, work, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}
	installedNames := map[string]bool{}
	if step.Row.Arm == "W1" {
		actual, err := inventory(work, []string{"buildoptw", "buildoptw.bat", ".buildopt/wrapper.properties", ".buildopt/config.toml"})
		if err != nil || !reflect.DeepEqual(actual, f.Installed) {
			return "", errors.New("installed wrapper source drift")
		}
		for _, entry := range f.Installed {
			installedNames[entry.Path] = true
		}
	}
	for _, name := range strings.Split(untracked, "\n") {
		if name != "" && !installedNames[name] {
			return "", fmt.Errorf("unexpected untracked source: %s", name)
		}
	}
	taskSHA := sourceInputs().TaskPreimageSHA256
	if step.Row.Arm != "N0" {
		taskSHA = sourceInputs().TaskPostimageSHA256
	}
	for _, v := range []fileBinding{{filepath.Join(work, sourceInputs().TaskPath), taskSHA}, {filepath.Join(work, sourceInputs().OwnerInputPath), expectedInput}, {filepath.Join(work, "gradlew"), "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"}} {
		if err := checkBinding(v); err != nil {
			return "", err
		}
	}
	return nativeGit(ctx, work, "rev-parse", "HEAD^{tree}")
}

type correctnessRowCheck struct {
	RowID            string            `json:"rowId"`
	TaskOutcome      string            `json:"taskOutcome"`
	OuterNanoseconds int64             `json:"outerNanoseconds"`
	FailureSignature string            `json:"failureSignature,omitempty"`
	Outputs          *nativeComparison `json:"outputs,omitempty"`
}

func inspectCorrectnessRow(f correctnessRuntime, step correctnessStep) (correctnessRowCheck, error) {
	result := correctnessRowCheck{RowID: step.Row.ID}
	dir := filepath.Join(f.Root, "attempts", step.Row.ID)
	if err := verifyNativeDiagnosticComplete(dir); err != nil {
		return result, err
	}
	var request nativeRequest
	var native nativeResult
	var p processRecord
	for name, target := range map[string]any{"native-request.json": &request, "native-result.json": &native, "process.json": &p} {
		if err := readJSON(filepath.Join(dir, name), target); err != nil {
			return result, err
		}
	}
	expected, err := correctnessRequestFor(f, step)
	if err != nil {
		return result, err
	}
	if request.Schema != expected.Schema || !reflect.DeepEqual(expected.Arguments, request.Arguments) || !reflect.DeepEqual(expected.Environment, request.Environment) || !reflect.DeepEqual(expected.Row, request.Row) || request.Directory != expected.Directory || request.Runner != f.Runner || !reflect.DeepEqual(request.Diagnostic, expected.Diagnostic) || request.Root != f.Root || request.BootID != f.BootID || request.TimeoutSeconds != 600 || request.ReservedNS < f.StartNS || request.ReservedNS > p.StartNS || p.EndNS > f.DeadlineNS || !reflect.DeepEqual(p, native.Process) || native.RowID != step.Row.ID || native.InputState != "VERIFIED" || !p.Started || p.Signal != 0 || p.EndNS <= p.StartNS {
		return result, errors.New("captured request, source, timing or native result drift")
	}
	desired := "SUCCESS"
	code := 0
	if step.Row.ExpectedFailure {
		desired = "FAILURE"
		code = 1
	}
	if p.Outcome != desired || p.ExitCode != code {
		return result, fmt.Errorf("unexpected native result: %s exit %d", p.Outcome, p.ExitCode)
	}
	var before, after []entry
	if err := readJSON(filepath.Join(dir, "source-before.json"), &before); err != nil {
		return result, err
	}
	if err := readJSON(filepath.Join(dir, "source-after.json"), &after); err != nil {
		return result, err
	}
	if !reflect.DeepEqual(before, after) || len(before) != 3 {
		return result, errors.New("captured source changed across native execution")
	}
	expectedSource := map[string]string{sourceInputs().TaskPath: sourceInputs().TaskPreimageSHA256, sourceInputs().OwnerInputPath: step.InputAfter, "gradlew": "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"}
	if step.Row.Arm != "N0" {
		expectedSource[sourceInputs().TaskPath] = sourceInputs().TaskPostimageSHA256
	}
	for _, entry := range before {
		mode := uint32(0644)
		if entry.Path == "gradlew" {
			mode = 0755
		}
		if entry.Type != "file" || entry.Mode != mode || expectedSource[entry.Path] != entry.SHA256 {
			return result, errors.New("captured exact source binding drift")
		}
		delete(expectedSource, entry.Path)
	}
	if len(expectedSource) != 0 {
		return result, errors.New("missing captured source")
	}
	var ownership map[string]string
	if err := readJSON(filepath.Join(dir, "ownership.json"), &ownership); err != nil {
		return result, err
	}
	if ownership["cgroup"] != native.Cgroup || ownership["invocationId"] != native.InvocationID || native.InvocationID == "" || !strings.HasSuffix(native.Cgroup, "/"+ownership["unit"]) {
		return result, errors.New("owned native service binding drift")
	}
	var closed correctnessClosed
	if err := readJSON(filepath.Join(dir, "closed.json"), &closed); err != nil {
		return result, err
	}
	if !closed.Absent || closed.Unit != ownership["unit"] || closed.Cgroup != native.Cgroup || closed.EndNS < p.EndNS || closed.EndNS > f.DeadlineNS {
		return result, errors.New("owned correctness service closure is unverified")
	}
	report, err := gradlecriticalpath.Analyze(filepath.Join(dir, "operations-log.txt"), filepath.Join(dir, "task-graph.jsonl"), "control")
	if err != nil {
		return result, err
	}
	var capturedReport gradlecriticalpath.Report
	if err := readJSONLimit(filepath.Join(dir, "critical-path.json"), &capturedReport, 16<<20); err != nil {
		return result, err
	}
	if !reflect.DeepEqual(capturedReport, report) {
		return result, errors.New("task report differs from raw build operations")
	}
	count := 0
	for _, task := range report.Tasks {
		if task.Identity == ":server:forbiddenPatterns" {
			count++
			result.TaskOutcome = task.Outcome
		}
	}
	if count != 1 || result.TaskOutcome != step.TaskOutcome {
		return result, fmt.Errorf("%s expected %s; observed %s", step.Row.ID, step.TaskOutcome, result.TaskOutcome)
	}
	result.OuterNanoseconds = p.EndNS - p.StartNS
	if !step.Row.ExpectedFailure {
		path := filepath.Join(dir, "diagnostic-complete.json")
		sha, err := hashFile(path)
		if err != nil {
			return result, err
		}
		capture, err := loadComparisonCapture(fileBinding{path, sha})
		if err != nil {
			return result, err
		}
		if f.Metadata != nil && f.metadata == nil {
			return result, errors.New("metadata comparison session required")
		}
		if f.metadata != nil {
			if _, err := f.metadata.projectCapture(capture); err != nil {
				return result, err
			}
		}
		if f.Provenance != nil && f.provenance == nil {
			return result, errors.New("provenance session required")
		}
		if f.provenance != nil {
			if _, _, err := newProvenanceView(f.provenance).checkstyle(capture); err != nil {
				return result, err
			}
		}
	}
	if step.Row.Arm == "W1" {
		if err := verifyCorrectnessSupervision(dir, p); err != nil {
			return result, err
		}
		var trace correctnessSupervision
		if err := readJSON(filepath.Join(dir, "native-supervision.json"), &trace); err != nil {
			return result, err
		}
		if trace.Cgroup != native.Cgroup {
			return result, errors.New("installed native process escaped owned service")
		}
	}
	if step.Row.ExpectedFailure {
		stdout, err := os.ReadFile(filepath.Join(dir, "stdout.log"))
		if err != nil {
			return result, err
		}
		stderr, err := os.ReadFile(filepath.Join(dir, "stderr.log"))
		if err != nil {
			return result, err
		}
		result.FailureSignature, err = correctnessFailureSignature(string(stdout) + "\n" + string(stderr))
		if err != nil {
			return result, err
		}
		if step.CompareWith != "" {
			var other correctnessRowCheck
			if err := readJSON(filepath.Join(f.Root, "attempts", step.CompareWith, "row-check.json"), &other); err != nil {
				return result, err
			}
			if other.FailureSignature != result.FailureSignature {
				return result, errors.New("native and installed failure differ")
			}
		}
	} else if step.CompareWith != "" {
		bindings := []fileBinding{}
		for _, id := range []string{step.CompareWith, step.Row.ID} {
			path := filepath.Join(f.Root, "attempts", id, "diagnostic-complete.json")
			sha, err := hashFile(path)
			if err != nil {
				return result, err
			}
			bindings = append(bindings, fileBinding{path, sha})
		}
		contract, err := loadAcceptedDateContract()
		if err != nil {
			return result, err
		}
		comparison, err := compareOutputsWithProvenance(bindings[0], bindings[1], contract, f.metadata, f.provenance)
		if err != nil {
			return result, err
		}
		if !comparison.Passed {
			return result, errors.New("correctness output comparison failed")
		}
		result.Outputs = &comparison
	}
	// Verify the exact transition from retained inventories, independently of the
	// helper's summary or current mutable worktree.
	var transition correctnessTransition
	if err := readJSONLimit(filepath.Join(f.Root, "transitions", step.Row.ID, "after.json"), &transition, 64<<20); err != nil {
		return result, err
	}
	if transition.RowID != step.Row.ID {
		return result, errors.New("transition row drift")
	}
	for _, check := range []struct {
		entries []entry
		pin     string
	}{{transition.Before, step.InputBefore}, {transition.After, step.InputAfter}} {
		found := false
		for _, e := range check.entries {
			if e.Path == sourceInputs().OwnerInputPath {
				found = e.Type == "file" && e.SHA256 == check.pin
			}
		}
		if !found {
			return result, errors.New("transition input digest drift")
		}
	}
	if step.RemoveMarker {
		found := false
		for _, e := range transition.After {
			if e.Path == forbiddenMarker {
				found = e.Type == "absent"
			}
		}
		if !found {
			return result, errors.New("cache restore did not remove owner marker")
		}
	}
	if step.CacheFrom != "" && len(transition.Cache) == 0 {
		return result, errors.New("cross-root native cache binding missing")
	}
	return result, nil
}

func sealCorrectnessRow(f correctnessRuntime, step correctnessStep) (correctnessRowReceipt, error) {
	receipt := correctnessRowReceipt{RowID: step.Row.ID}
	result, err := inspectCorrectnessRow(f, step)
	if err != nil {
		return receipt, err
	}
	dir := filepath.Join(f.Root, "attempts", step.Row.ID)
	if err = writeJSON(filepath.Join(dir, "row-check.json"), result); err != nil {
		return receipt, err
	}
	files := append(append([]string{}, nativeDiagnosticArtifacts...), "diagnostic-complete.json", "row-check.json", "ownership.json", "service-request.json", "service.log", "started.json")
	files = append(files, "source-before.json", "source-after.json", "closed.json")
	if step.Row.Arm == "W1" {
		files = append(files, "native-supervision.json")
	}
	bindings := map[string]string{}
	for _, name := range files {
		sha, err := hashFile(filepath.Join(dir, name))
		if err != nil {
			return receipt, err
		}
		bindings[name] = sha
	}
	receipt.Complete.Path = filepath.Join(dir, "row-complete.json")
	if err = writeJSON(receipt.Complete.Path, bindings); err != nil {
		return receipt, err
	}
	receipt.Complete.SHA256, err = hashFile(receipt.Complete.Path)
	return receipt, err
}

func rememberCorrectnessOrigin(f correctnessRuntime, step correctnessStep) error {
	if f.provenance == nil || step.Row.ExpectedFailure {
		return nil
	}
	p := filepath.Join(f.Root, "attempts", step.Row.ID, "diagnostic-complete.json")
	h, err := hashFile(p)
	if err != nil {
		return err
	}
	return f.provenance.rememberVerified(fileBinding{p, h})
}

func validSHA256(pin string) bool {
	if len(pin) != 64 {
		return false
	}
	for _, r := range pin {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func correctnessRuntimeDigest(f correctnessRuntime) string {
	raw, _ := json.Marshal(f)
	return digest(raw)
}
