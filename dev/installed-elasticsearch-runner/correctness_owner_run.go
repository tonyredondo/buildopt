package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/gradlecriticalpath"
)

type ownerTestFreeze struct {
	Schema            string                         `json:"schemaVersion"`
	Correctness       fileBinding                    `json:"correctness"`
	Mutations         fileBinding                    `json:"mutations"`
	Runtime           correctnessRuntime             `json:"runtime"`
	JDK               []entry                        `json:"jdkInventory"`
	Steps             []correctnessStep              `json:"steps"`
	Definitions       map[string]ownerTestDefinition `json:"definitions"`
	MaxOuter          int                            `json:"maximumOuterStarts"`
	MaxNested         int                            `json:"maximumNestedStarts"`
	CompositeDecision *fileBinding                   `json:"compositeDecision,omitempty"`
	Continuation      *fileBinding                   `json:"continuation,omitempty"`
}

type ownerTestEvidence struct {
	RowID     string        `json:"rowId"`
	Process   processRecord `json:"process"`
	TestCount int           `json:"testCount"`
	Nested    testKitCount  `json:"nested"`
}

type ownerTestReceipt struct {
	Schema    string      `json:"schemaVersion"`
	Freeze    fileBinding `json:"freeze"`
	Artifacts []entry     `json:"artifacts"`
}

func ownerTestSteps() []correctnessStep {
	var steps []correctnessStep
	for _, step := range makeCorrectnessPlan().Steps {
		if step.Row.Block == "T" {
			steps = append(steps, step)
		}
	}
	return steps
}

func ownerDefinitionPath(workload string) string {
	if workload == "unit" {
		return "build-tools-internal/src/test/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTaskTests.java"
	}
	return "build-tools-internal/src/integTest/groovy/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsPrecommitPluginFuncTest.groovy"
}

func freezeOwnerTests(ctx context.Context, mutations fileBinding, decision *fileBinding) (fileBinding, error) {
	var binding fileBinding
	summary, err := checkMutations(mutations)
	if err != nil {
		return binding, err
	}
	if summary.Class != correctnessMutationClass || summary.Starts != 24 || !summary.Passed {
		return binding, errors.New("verified fresh public M allocation required")
	}
	var mr mutationReceipt
	if err = readJSONLimit(mutations.Path, &mr, 16<<20); err != nil {
		return binding, err
	}
	mf, err := loadMutationFreeze(mr.Freeze, false)
	if err != nil || mf.Correctness == nil {
		return binding, errors.New("public mutation parent missing")
	}
	var cr correctnessReceipt
	if err = readJSON(mf.Correctness.Path, &cr); err != nil {
		return binding, err
	}
	cf, err := loadCorrectnessFreeze(cr.Freeze)
	if err != nil {
		return binding, err
	}
	f := ownerTestFreeze{Schema: "buildopt.eic/owner-test-freeze/v1", Correctness: *mf.Correctness, Mutations: mutations, Runtime: cf.Runtime, JDK: cf.JDK, Definitions: map[string]ownerTestDefinition{}, MaxOuter: 4, MaxNested: 12, CompositeDecision: decision}
	if decision != nil {
		f.Schema = ownerCompositeSchema
	}
	profileRoot, steps, err := ownerProfile(f)
	if err != nil {
		return binding, err
	}
	f.Steps = steps
	root := filepath.Join(cf.Runtime.Root, profileRoot)
	if err = canonicalExisting(cf.Runtime.Root); err != nil {
		return binding, err
	}
	if err = os.Mkdir(root, 0700); err != nil {
		return binding, err
	}
	f.Runtime.EvidenceRoot = root
	runner, err := ownExecutable()
	if err != nil {
		return binding, err
	}
	sha, err := hashFile(runner)
	if err != nil {
		return binding, err
	}
	f.Runtime.Runner = fileBinding{runner, sha}
	f.Runtime.BootID, err = bootID()
	if err != nil {
		return binding, err
	}
	f.Runtime.StartNS, err = bootNow()
	if err != nil {
		return binding, err
	}
	f.Runtime.DeadlineNS = f.Runtime.StartNS + int64(2700*time.Second)
	for _, workload := range []string{"unit", "functional"} {
		source := filepath.Join(f.Runtime.Root, "arms/N0", ownerDefinitionPath(workload))
		raw, err := os.ReadFile(source)
		if err != nil {
			return binding, err
		}
		definition, err := parseOwnerTestDefinition(workload, raw)
		if err != nil {
			return binding, err
		}
		f.Definitions[workload] = definition
		for _, arm := range []string{"N0", "N1"} {
			if err = checkBinding(fileBinding{filepath.Join(f.Runtime.Root, "arms", arm, definition.Source), definition.SHA256}); err != nil {
				return binding, err
			}
		}
	}
	// Fresh output/report absence prevents a cached or stale XML result from
	// standing in for the requested owner test executions.
	for _, arm := range []string{"N0", "N1"} {
		step := f.Steps[0]
		step.Row.Arm = arm
		if _, err = verifyCorrectnessSource(ctx, f.Runtime, step, step.InputBefore); err != nil {
			return binding, err
		}
		for _, rel := range []string{"build-tools-internal/build/test-results/test", "build-tools-internal/build/test-results/integTest", "build-tools-internal/build/tmp/integTest/.gradle-test-kit"} {
			if _, err = os.Lstat(filepath.Join(f.Runtime.Root, "arms", arm, rel)); !os.IsNotExist(err) {
				return binding, errors.New("owner test outputs or TestKit state already exist")
			}
		}
	}
	if err = os.Mkdir(filepath.Join(root, "attempts"), 0700); err != nil {
		return binding, err
	}
	binding.Path = filepath.Join(root, "freeze.json")
	if err = writeJSON(binding.Path, f); err != nil {
		return binding, err
	}
	binding.SHA256, err = hashFile(binding.Path)
	return binding, err
}

func loadOwnerTestFreeze(binding fileBinding) (ownerTestFreeze, error) {
	var f ownerTestFreeze
	if err := checkBinding(binding); err != nil {
		return f, err
	}
	if err := readJSONLimit(binding.Path, &f, 16<<20); err != nil {
		return f, err
	}
	profileRoot, steps, err := ownerProfile(f)
	if err != nil {
		return f, err
	}
	outer := 4
	if f.Schema == ownerContinuationSchema {
		outer = 3
	}
	if f.Runtime.EvidenceRoot != filepath.Join(f.Runtime.Root, profileRoot) || binding.Path != filepath.Join(f.Runtime.EvidenceRoot, "freeze.json") || !reflect.DeepEqual(f.Steps, steps) || f.MaxOuter != outer || f.MaxNested != 12 || f.Runtime.DeadlineNS-f.Runtime.StartNS != int64(2700*time.Second) || len(f.Definitions) != 2 {
		return f, errors.New("owner test freeze drift")
	}
	for _, b := range []fileBinding{f.Correctness, f.Mutations, f.Runtime.Runner} {
		if err := checkBinding(b); err != nil {
			return f, err
		}
	}
	return f, nil
}

func ownerTestChild(args []string) int {
	if len(args) != 5 {
		return 64
	}
	path, pin, unit := args[0], args[1], args[2]
	if checkBinding(fileBinding{path, pin}) != nil {
		return 65
	}
	f, err := loadOwnerTestFreeze(fileBinding{args[3], args[4]})
	if err != nil {
		return 65
	}
	var request nativeRequest
	if readJSON(path, &request) != nil {
		return 65
	}
	var step correctnessStep
	for _, s := range f.Steps {
		if s.Row.ID == request.Row.ID {
			step = s
		}
	}
	expected, err := ownerRequestFor(f, step)
	if err != nil {
		return 65
	}
	self, err := ownExecutable()
	if err != nil || self != f.Runtime.Runner.Path {
		return 65
	}
	if request.Schema != expected.Schema || !reflect.DeepEqual(request.Row, expected.Row) || !reflect.DeepEqual(request.Arguments, expected.Arguments) || !reflect.DeepEqual(request.Environment, expected.Environment) || request.Directory != expected.Directory || request.Root != expected.Root || request.Runner != expected.Runner || !reflect.DeepEqual(request.Runtime, f.JDK) || !reflect.DeepEqual(request.Diagnostic, expected.Diagnostic) || checkBinding(*request.Diagnostic) != nil || request.TimeoutSeconds != 600 {
		return 65
	}
	boot, err := bootID()
	if err != nil || boot != request.BootID || boot != f.Runtime.BootID {
		return 65
	}
	now, err := bootNow()
	if err != nil || now < request.ReservedNS || now+int64(600*time.Second) > f.Runtime.DeadlineNS {
		return 65
	}
	rawGroup, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || !strings.HasSuffix(strings.TrimSpace(string(rawGroup)), "/"+unit) {
		return 65
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	definition := f.Definitions[step.Row.Workload]
	checkSource := func() bool {
		tree, e := verifyCorrectnessSource(ctx, f.Runtime, step, step.InputAfter)
		if e != nil || tree != request.SourceTree {
			return false
		}
		if checkBinding(fileBinding{filepath.Join(request.Directory, definition.Source), definition.SHA256}) != nil {
			return false
		}
		actual, e := inventory(filepath.Join(f.Runtime.Root, "runtime/jdk-21"), []string{"bin", "conf", "lib", "release"})
		return e == nil && reflect.DeepEqual(actual, f.JDK)
	}
	if !checkSource() {
		return 65
	}
	dir := filepath.Dir(path)
	syscall.Umask(022)
	ownership := map[string]string{"unit": unit, "cgroup": strings.TrimSpace(string(rawGroup)), "invocationId": os.Getenv("INVOCATION_ID")}
	if writeJSON(filepath.Join(dir, "ownership.json"), ownership) != nil {
		return 65
	}
	inputs := []string{sourceInputs().TaskPath, sourceInputs().OwnerInputPath, "gradlew", definition.Source}
	before, err := retainSelected(request.Directory, filepath.Join(dir, "inputs"), inputs)
	if err != nil {
		return 65
	}
	if writeJSON(filepath.Join(dir, "source-before.json"), before) != nil {
		return 65
	}
	process, err := runConfiguredProcess(ctx, dir, request.Arguments[0], request.Arguments[1:], request.Directory, request.Environment, 600*time.Second)
	result := nativeResult{Schema: "buildopt.eic/native-result/v1", RowID: step.Row.ID, Process: process, Cgroup: ownership["cgroup"], InvocationID: ownership["invocationId"], InputState: "UNVERIFIED"}
	if checkSource() {
		result.InputState = "VERIFIED"
	}
	if writeJSON(filepath.Join(dir, "native-result.json"), result) != nil {
		return 65
	}
	after, e := inventory(request.Directory, inputs)
	if e != nil || !reflect.DeepEqual(before, after) || writeJSON(filepath.Join(dir, "source-after.json"), after) != nil {
		return 65
	}
	if err != nil || result.InputState != "VERIFIED" {
		return 1
	}
	return 0
}

func runOwnerTests(ctx context.Context, binding fileBinding) (receiptBinding fileBinding, retErr error) {
	f, err := loadOwnerTestFreeze(binding)
	if err != nil {
		return receiptBinding, err
	}
	self, err := ownExecutable()
	if err != nil || self != f.Runtime.Runner.Path {
		return receiptBinding, errors.New("owner test runner differs from freeze")
	}
	root := f.Runtime.EvidenceRoot
	if err = writeJSON(filepath.Join(root, "run-start.json"), binding); err != nil {
		return receiptBinding, err
	}
	defer func() {
		if retErr != nil {
			_ = writeJSON(filepath.Join(root, "run-result.json"), map[string]string{"status": "INCOMPLETE", "error": retErr.Error()})
		}
		artifacts, err := inventory(root, []string{"freeze.json", "run-start.json", "run-result.json", "attempts"})
		if err != nil {
			retErr = errors.Join(retErr, err)
			return
		}
		receiptBinding.Path = filepath.Join(root, "receipt.json")
		if err = writeJSON(receiptBinding.Path, ownerTestReceipt{"buildopt.eic/owner-test-receipt/v1", binding, artifacts}); err != nil {
			retErr = errors.Join(retErr, err)
			return
		}
		receiptBinding.SHA256, err = hashFile(receiptBinding.Path)
		retErr = errors.Join(retErr, err)
	}()
	results, err := ownerPriorEvidence(f)
	if err != nil {
		return receiptBinding, err
	}
	prefix := len(results)
	for _, step := range f.Steps[prefix:] {
		if ctx.Err() != nil {
			return receiptBinding, ctx.Err()
		}
		if err = nativeDiskGuard(f.Runtime.Root); err != nil {
			return receiptBinding, err
		}
		now, err := bootNow()
		if err != nil || now+int64(622*time.Second) > f.Runtime.DeadlineNS {
			return receiptBinding, errors.New("owner test deadline cannot cover next capture")
		}
		request, err := ownerRequestFor(f, step)
		if err != nil {
			return receiptBinding, err
		}
		request.Runtime = f.JDK
		request.BootID = f.Runtime.BootID
		request.ReservedNS = now
		request.SourceTree, err = verifyCorrectnessSource(ctx, f.Runtime, step, step.InputAfter)
		if err != nil {
			return receiptBinding, err
		}
		dir := filepath.Join(root, "attempts", step.Row.ID)
		if err = os.Mkdir(dir, 0700); err != nil {
			return receiptBinding, err
		}
		path := filepath.Join(dir, "native-request.json")
		if err = writeJSON(path, request); err != nil {
			return receiptBinding, err
		}
		sha, err := hashFile(path)
		if err != nil {
			return receiptBinding, err
		}
		unit := "buildopt-eic-t-" + sha[:20] + ".service"
		args := nativeServiceArguments(unit, self, path, sha)
		args[len(args)-4] = "owner-test-child"
		args = append(args, binding.Path, binding.SHA256)
		if err = writeJSON(filepath.Join(dir, "service-request.json"), args); err != nil {
			return receiptBinding, err
		}
		raw, runErr := runNativeService(ctx, args, unit)
		if err = writeNew(filepath.Join(dir, "service.log"), raw); err != nil {
			return receiptBinding, err
		}
		var native nativeResult
		if err = readJSON(filepath.Join(dir, "native-result.json"), &native); err != nil {
			return receiptBinding, errors.Join(err, runErr)
		}
		if !strings.HasPrefix(native.Cgroup, "0::/") || !strings.HasSuffix(native.Cgroup, "/"+unit) {
			return receiptBinding, errors.New("owner test cgroup identity drift")
		}
		if _, err = os.Stat(filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(native.Cgroup, "0::"))); !os.IsNotExist(err) {
			return receiptBinding, errors.New("owner test service still exists")
		}
		closed, err := bootNow()
		if err != nil {
			return receiptBinding, err
		}
		if err = writeJSON(filepath.Join(dir, "closed.json"), correctnessClosed{unit, native.Cgroup, true, closed}); err != nil {
			return receiptBinding, err
		}
		// Retain owner XML and ordinary TestKit daemon logs even after a failing
		// outer build, so failed tests and partial nested starts remain countable.
		task := "test"
		if step.Row.Workload == "functional" {
			task = "integTest"
		}
		resultsRel := filepath.Join("build-tools-internal/build/test-results", task)
		if _, err = retainSelected(request.Directory, filepath.Join(dir, "outputs"), []string{resultsRel}); err != nil {
			return receiptBinding, errors.Join(err, runErr)
		}
		if step.Row.NestedStarts > 0 {
			logs, err := filepath.Glob(filepath.Join(request.Directory, "build-tools-internal/build/tmp/integTest/.gradle-test-kit/test-kit-daemon/9.7.1/daemon-*.out.log"))
			if err != nil {
				return receiptBinding, err
			}
			logRoot := filepath.Join(dir, "daemon-logs")
			if err = os.Mkdir(logRoot, 0700); err != nil {
				return receiptBinding, err
			}
			manifest := testKitLogManifest{Schema: "buildopt.eic/testkit-logs/v1", Logs: []fileBinding{}}
			for _, source := range logs {
				target := filepath.Join(logRoot, filepath.Base(source))
				if err = copyTree(source, target); err != nil {
					return receiptBinding, err
				}
				sha, err := hashFile(target)
				if err != nil {
					return receiptBinding, err
				}
				manifest.Logs = append(manifest.Logs, fileBinding{target, sha})
			}
			if err = writeJSON(filepath.Join(dir, "testkit-logs.json"), manifest); err != nil {
				return receiptBinding, err
			}
		}
		if runErr != nil {
			return receiptBinding, runErr
		}
		result, err := inspectOwnerTest(f, step)
		if err != nil {
			return receiptBinding, err
		}
		if err = writeJSON(filepath.Join(dir, "verified.json"), result); err != nil {
			return receiptBinding, err
		}
		results = append(results, result)
		fmt.Fprintf(os.Stderr, "%s verified (%d tests, %d nested requests)\n", step.Row.ID, result.TestCount, result.Nested.Starts)
	}
	return receiptBinding, writeJSON(filepath.Join(root, "run-result.json"), results)
}

func inspectOwnerTest(f ownerTestFreeze, step correctnessStep) (ownerTestEvidence, error) {
	result := ownerTestEvidence{RowID: step.Row.ID}
	dir := filepath.Join(f.Runtime.EvidenceRoot, "attempts", step.Row.ID)
	var request nativeRequest
	var native nativeResult
	var process processRecord
	var closed correctnessClosed
	for name, target := range map[string]any{"native-request.json": &request, "native-result.json": &native, "process.json": &process, "closed.json": &closed} {
		if err := readJSON(filepath.Join(dir, name), target); err != nil {
			return result, err
		}
	}
	expected, err := ownerRequestFor(f, step)
	if err != nil {
		return result, err
	}
	if request.Schema != expected.Schema || !reflect.DeepEqual(request.Row, expected.Row) || !reflect.DeepEqual(request.Arguments, expected.Arguments) || !reflect.DeepEqual(request.Environment, expected.Environment) || request.Directory != expected.Directory || request.Root != expected.Root || request.Runner != expected.Runner || !reflect.DeepEqual(request.Runtime, f.JDK) || !reflect.DeepEqual(request.Diagnostic, expected.Diagnostic) || request.BootID != f.Runtime.BootID || request.TimeoutSeconds != 600 || request.ReservedNS < f.Runtime.StartNS || request.ReservedNS > process.StartNS || !reflect.DeepEqual(process, native.Process) || native.RowID != step.Row.ID || native.InputState != "VERIFIED" || !process.Started || process.ExitCode != 0 || process.Signal != 0 || process.Outcome != "SUCCESS" || process.EndNS <= process.StartNS || process.EndNS > f.Runtime.DeadlineNS || !closed.Absent || closed.Cgroup != native.Cgroup || closed.EndNS < process.EndNS || closed.EndNS > f.Runtime.DeadlineNS || native.InvocationID == "" || !strings.HasSuffix(native.Cgroup, "/"+closed.Unit) {
		return result, errors.New("owner request, source, native result or closure mismatch")
	}
	result.Process = process
	definition := f.Definitions[step.Row.Workload]
	task := "test"
	if step.Row.Workload == "functional" {
		task = "integTest"
	}
	var report gradlecriticalpath.Report
	if f.Schema == ownerCompositeSchema || f.Schema == ownerContinuationSchema {
		report, err = analyzeOwnerOperations(filepath.Join(dir, "operations-log.txt"))
	} else {
		report, err = gradlecriticalpath.Analyze(filepath.Join(dir, "operations-log.txt"), filepath.Join(dir, "task-graph.jsonl"), "control")
	}
	if err != nil {
		return result, err
	}
	if err := checkOwnerTask(f, task, report.Tasks); err != nil {
		return result, err
	}
	var before, after []entry
	for name, target := range map[string]any{"source-before.json": &before, "source-after.json": &after} {
		if err := readJSON(filepath.Join(dir, name), target); err != nil {
			return result, err
		}
	}
	source, err := inventory(filepath.Join(dir, "inputs"), []string{sourceInputs().TaskPath, sourceInputs().OwnerInputPath, "gradlew", definition.Source})
	if err != nil || !reflect.DeepEqual(source, before) || !reflect.DeepEqual(before, after) {
		return result, errors.New("retained owner test source differs")
	}
	bytes, err := os.ReadFile(filepath.Join(dir, "inputs", definition.Source))
	if err != nil {
		return result, err
	}
	derived, err := parseOwnerTestDefinition(step.Row.Workload, bytes)
	if err != nil || !reflect.DeepEqual(derived, definition) {
		return result, errors.New("owner test definition drift")
	}
	taskSHA := sourceInputs().TaskPreimageSHA256
	if step.Row.Arm == "N1" {
		taskSHA = sourceInputs().TaskPostimageSHA256
	}
	for path, pin := range map[string]string{sourceInputs().TaskPath: taskSHA, sourceInputs().OwnerInputPath: step.InputAfter, "gradlew": "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"} {
		if err := checkBinding(fileBinding{filepath.Join(dir, "inputs", path), pin}); err != nil {
			return result, err
		}
	}
	xmlPath := filepath.Join(dir, "outputs/build-tools-internal/build/test-results", task, "TEST-"+definition.Class+".xml")
	result.TestCount, err = checkOwnerJUnit(xmlPath, definition)
	if err != nil {
		return result, err
	}
	if step.Row.NestedStarts > 0 {
		path := filepath.Join(dir, "testkit-logs.json")
		sha, err := hashFile(path)
		if err != nil {
			return result, err
		}
		result.Nested, err = countTestKitBuilds(fileBinding{path, sha})
		if err != nil {
			return result, err
		}
		if result.Nested.Starts != step.Row.NestedStarts {
			return result, fmt.Errorf("expected %d nested Gradle starts, observed %d", step.Row.NestedStarts, result.Nested.Starts)
		}
		window, err := nativeInvocationWindow(filepath.Join(dir, "operations-log.txt"))
		if err != nil {
			return result, err
		}
		for _, build := range result.Nested.Builds {
			if build.JavaHome != filepath.Join(f.Runtime.Root, "runtime/jdk-21") || build.ReceivedMs < window.StartMs || build.FinishedMs > window.EndMs || !strings.HasPrefix(build.ClientDirectory, request.Directory+string(filepath.Separator)) {
				return result, errors.New("nested Gradle runtime, root or outer interval mismatch")
			}
		}
	}
	return result, nil
}

func checkOwnerTests(binding fileBinding) ([]ownerTestEvidence, error) {
	if err := checkBinding(binding); err != nil {
		return nil, err
	}
	var receipt ownerTestReceipt
	if err := readJSONLimit(binding.Path, &receipt, 16<<20); err != nil {
		return nil, err
	}
	f, err := loadOwnerTestFreeze(receipt.Freeze)
	if err != nil {
		return nil, err
	}
	if receipt.Schema != "buildopt.eic/owner-test-receipt/v1" || binding.Path != filepath.Join(f.Runtime.EvidenceRoot, "receipt.json") {
		return nil, errors.New("owner test receipt drift")
	}
	artifacts, err := inventory(f.Runtime.EvidenceRoot, []string{"freeze.json", "run-start.json", "run-result.json", "attempts"})
	if err != nil || !reflect.DeepEqual(artifacts, receipt.Artifacts) {
		return nil, errors.New("owner test evidence changed")
	}
	if _, err = checkMutations(f.Mutations); err != nil {
		return nil, err
	}
	attempts, err := os.ReadDir(filepath.Join(f.Runtime.EvidenceRoot, "attempts"))
	if err != nil || len(attempts) != f.MaxOuter {
		return nil, errors.New("missing or extra owner test outer starts")
	}
	results, err := ownerPriorEvidence(f)
	if err != nil {
		return nil, err
	}
	prefix := len(results)
	last := f.Runtime.StartNS
	nested := 0
	for _, step := range f.Steps[prefix:] {
		r, err := inspectOwnerTest(f, step)
		if err != nil {
			return nil, err
		}
		if r.Process.StartNS < last {
			return nil, errors.New("owner test rows overlap or are reordered")
		}
		last = r.Process.EndNS
		nested += r.Nested.Starts
		results = append(results, r)
	}
	if nested != 12 {
		return nil, errors.New("nested owner test starts incomplete")
	}
	var recorded []ownerTestEvidence
	if err = readJSONLimit(filepath.Join(f.Runtime.EvidenceRoot, "run-result.json"), &recorded, 16<<20); err != nil || !reflect.DeepEqual(recorded, results) {
		return nil, errors.New("owner test summary differs from reconstruction")
	}
	return results, nil
}
