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
)

type correctnessPreparation struct {
	Schema     string       `json:"schemaVersion"`
	Root       string       `json:"root"`
	CommonGit  string       `json:"commonGit"`
	Package    fileBinding  `json:"package"`
	Admission  fileBinding  `json:"nativeAdmission"`
	Delivery   fileBinding  `json:"delivery"`
	Probe      fileBinding  `json:"installedProbe"`
	Metadata   *fileBinding `json:"metadataRuntime,omitempty"`
	Provenance *fileBinding `json:"provenanceRuntime,omitempty"`
}

type correctnessFreeze struct {
	Schema      string             `json:"schemaVersion"`
	Preparation fileBinding        `json:"preparation"`
	Runtime     correctnessRuntime `json:"runtime"`
	JDK         []entry            `json:"jdkInventory"`
	Initial     map[string]string  `json:"initialStateSha256"`
	Steps       []correctnessStep  `json:"steps"`
	MaxStarts   int                `json:"maximumStarts"`
	MaxBytes    int64              `json:"maximumBytes"`
}

type correctnessReceipt struct {
	Schema       string                  `json:"schemaVersion"`
	Freeze       fileBinding             `json:"freeze"`
	Rows         []correctnessRowReceipt `json:"rows"`
	Observations fileBinding             `json:"observations"`
	Transitions  map[string]string       `json:"transitions"`
	Stopped      string                  `json:"stopped,omitempty"`
}

func loadCorrectnessFreeze(binding fileBinding) (correctnessFreeze, error) {
	var f correctnessFreeze
	if err := checkBinding(binding); err != nil {
		return f, err
	}
	if err := readJSONLimit(binding.Path, &f, 64<<20); err != nil {
		return f, err
	}
	if f.Schema != "buildopt.eic/correctness-freeze/v1" || binding.Path != filepath.Join(f.Runtime.Root, "correctness-freeze.json") || f.MaxStarts != 12 || f.MaxBytes != 30<<30 || !reflect.DeepEqual(f.Steps, makeCorrectnessPlan().Steps[:12]) || f.Runtime.StartNS <= 0 || f.Runtime.DeadlineNS-f.Runtime.StartNS != int64(2700*time.Second) {
		return f, errors.New("correctness freeze drift")
	}
	for _, pin := range []fileBinding{f.Preparation, f.Runtime.Runner, f.Runtime.Package, f.Runtime.CA} {
		if err := checkBinding(pin); err != nil {
			return f, err
		}
	}
	var prep correctnessPreparation
	if err := readJSON(f.Preparation.Path, &prep); err != nil {
		return f, err
	}
	if prep.Root != f.Runtime.Root || prep.Package != f.Runtime.Package || prep.CommonGit != f.Runtime.CommonGit || len(f.Initial) != 6 || len(f.JDK) == 0 {
		return f, errors.New("frozen runtime or initial state binding drift")
	}
	if !reflect.DeepEqual(prep.Metadata, f.Runtime.Metadata) {
		return f, errors.New("frozen metadata contract drift")
	}
	if f.Runtime.Metadata != nil {
		if err := checkBinding(*f.Runtime.Metadata); err != nil {
			return f, err
		}
	}
	if !reflect.DeepEqual(prep.Provenance, f.Runtime.Provenance) {
		return f, errors.New("frozen provenance contract drift")
	}
	if f.Runtime.Provenance != nil {
		if f.Runtime.Metadata == nil {
			return f, errors.New("provenance requires the approved metadata contract")
		}
		if err := checkBinding(*f.Runtime.Provenance); err != nil {
			return f, err
		}
	}
	for _, kind := range []string{"arms", "state"} {
		for _, arm := range []string{"N0", "N1", "W1"} {
			if !validSHA256(f.Initial[kind+"/"+arm]) {
				return f, errors.New("missing frozen arm state")
			}
		}
	}
	return f, nil
}

// Only the C allocation is opened here. M and T retain their own capture
// contracts; neither an incomplete C run nor C alone can admit value rows.
func runCorrectness(ctx context.Context, binding fileBinding) (receiptBinding fileBinding, retErr error) {
	if err := checkBinding(binding); err != nil {
		return receiptBinding, err
	}
	var prep correctnessPreparation
	if err := readJSON(binding.Path, &prep); err != nil {
		return receiptBinding, err
	}
	if prep.Schema != "buildopt.eic/correctness-preparation/v1" || binding.Path != filepath.Join(prep.Root, "preparation.json") {
		return receiptBinding, errors.New("correctness preparation identity drift")
	}
	for _, path := range []string{prep.Root, prep.CommonGit} {
		if err := canonicalExisting(path); err != nil {
			return receiptBinding, err
		}
	}
	if prep.Admission.SHA256 != "029ac769145df3dd22707f3c3768e276884374b4707c96e3f22ba2d4aa551b30" {
		return receiptBinding, errors.New("fresh accepted native admission required")
	}
	for _, pin := range []fileBinding{prep.Package, prep.Admission, prep.Delivery, prep.Probe} {
		if err := checkBinding(pin); err != nil {
			return receiptBinding, err
		}
	}
	var probe struct {
		Starts      int  `json:"gradleStarts"`
		Graph       bool `json:"installedGraphVerified"`
		Failure     bool `json:"nativeFailurePreserved"`
		Supervision bool `json:"supervisionVerified"`
		Public      bool `json:"publicAuthority"`
	}
	if err := readJSON(prep.Probe.Path, &probe); err != nil {
		return receiptBinding, err
	}
	if probe.Starts != 2 || !probe.Graph || !probe.Failure || !probe.Supervision || probe.Public {
		return receiptBinding, errors.New("installed collector not qualified")
	}
	runner, err := ownExecutable()
	if err != nil {
		return receiptBinding, err
	}
	runnerSHA, err := hashFile(runner)
	if err != nil {
		return receiptBinding, err
	}
	runtime := correctnessRuntime{Root: prep.Root, Runner: fileBinding{runner, runnerSHA}, Package: prep.Package, CommonGit: prep.CommonGit}
	runtime.Metadata = prep.Metadata
	runtime.metadata, err = newMetadataSession(prep.Metadata)
	if err != nil {
		return receiptBinding, err
	}
	runtime.Provenance = prep.Provenance
	if prep.Provenance != nil && prep.Metadata == nil {
		return receiptBinding, errors.New("provenance requires approved metadata")
	}
	runtime.provenance, err = newProvenanceSession(prep.Provenance, prep.Root)
	if err != nil {
		return receiptBinding, err
	}
	// Exclusive run reservation prevents replay even when setup or a row fails.
	if err := os.Mkdir(filepath.Join(prep.Root, "attempts"), 0700); err != nil {
		return receiptBinding, err
	}
	b, err := startCorrectnessBackend(ctx, filepath.Join(prep.Root, "backend"), prep.Package)
	if err != nil {
		return receiptBinding, err
	}
	defer func() { retErr = errors.Join(retErr, b.close()) }()
	runtime.BackendURL = b.URL
	runtime.Credential = b.Credential
	caSHA, err := hashFile(b.CA)
	if err != nil {
		return receiptBinding, err
	}
	runtime.CA = fileBinding{b.CA, caSHA}
	runtime.Installed, err = b.install(ctx, filepath.Join(prep.Root, "arms", "W1"))
	if err != nil {
		return receiptBinding, err
	}
	runtime.BootID, err = bootID()
	if err != nil {
		return receiptBinding, err
	}
	runtime.StartNS, err = bootNow()
	if err != nil {
		return receiptBinding, err
	}
	runtime.DeadlineNS = runtime.StartNS + int64(2700*time.Second)
	f := correctnessFreeze{Schema: "buildopt.eic/correctness-freeze/v1", Preparation: binding, Runtime: runtime, Steps: makeCorrectnessPlan().Steps[:12], MaxStarts: 12, MaxBytes: 30 << 30, Initial: map[string]string{}}
	f.JDK, err = inventory(filepath.Join(prep.Root, "runtime/jdk-21"), []string{"bin", "conf", "lib", "release"})
	if err != nil {
		return receiptBinding, err
	}
	release, err := os.ReadFile(filepath.Join(prep.Root, "runtime/jdk-21/release"))
	if err != nil || !strings.Contains(string(release), `JAVA_RUNTIME_VERSION="21.0.12+8-LTS"`) {
		return receiptBinding, errors.New("locked JDK drift")
	}
	if err = writeNew(filepath.Join(prep.Root, "runtime/correctness-diagnostic.init.gradle"), []byte(correctnessDiagnosticScript)); err != nil {
		return receiptBinding, err
	}
	for _, arm := range []string{"N0", "N1", "W1"} {
		step := f.Steps[0]
		step.Row.Arm = arm
		if _, err = verifyCorrectnessSource(ctx, runtime, step, sourceInputs().OwnerInputSHA256); err != nil {
			return receiptBinding, err
		}
		for _, kind := range []string{"arms", "state"} {
			path := filepath.Join(prep.Root, kind, arm)
			snapshot, err := cleanSnapshotDigest(path)
			if err != nil {
				return receiptBinding, err
			}
			f.Initial[kind+"/"+arm] = snapshot
		}
	}
	freezePath := filepath.Join(prep.Root, "correctness-freeze.json")
	if err = writeJSON(freezePath, f); err != nil {
		return receiptBinding, err
	}
	freezeSHA, err := hashFile(freezePath)
	if err != nil {
		return receiptBinding, err
	}
	receipt := correctnessReceipt{Schema: "buildopt.eic/correctness-receipt/v1", Freeze: fileBinding{freezePath, freezeSHA}, Rows: []correctnessRowReceipt{}, Transitions: map[string]string{}}
	defer func() {
		closeErr := b.close()
		retErr = errors.Join(retErr, closeErr)
		observationPath := filepath.Join(prep.Root, "observations.json")
		if err := writeJSON(observationPath, b.observations()); err != nil {
			retErr = errors.Join(retErr, err)
			return
		}
		sha, err := hashFile(observationPath)
		if err != nil {
			retErr = errors.Join(retErr, err)
			return
		}
		receipt.Observations = fileBinding{observationPath, sha}
		if retErr != nil {
			receipt.Stopped = retErr.Error()
		}
		receiptBinding.Path = filepath.Join(prep.Root, "correctness-receipt.json")
		if err := writeJSON(receiptBinding.Path, receipt); err != nil {
			retErr = errors.Join(retErr, err)
			return
		}
		receiptBinding.SHA256, err = hashFile(receiptBinding.Path)
		retErr = errors.Join(retErr, err)
	}()
	for _, step := range f.Steps {
		if err = validateCorrectnessSequence(receipt.Rows, step.Row.ID); err != nil {
			return receiptBinding, err
		}
		if err = nativeDiskGuard(prep.Root); err != nil {
			return receiptBinding, err
		}
		now, err := bootNow()
		if err != nil || now+int64(622*time.Second) > runtime.DeadlineNS {
			return receiptBinding, errors.New("correctness deadline cannot cover another owned capture")
		}
		if _, err = verifyCorrectnessSource(ctx, runtime, step, step.InputBefore); err != nil {
			return receiptBinding, err
		}
		if _, err = applyCorrectnessTransition(prep.Root, step); err != nil {
			return receiptBinding, err
		}
		for _, name := range []string{"before.json", "after.json"} {
			rel := filepath.Join("transitions", step.Row.ID, name)
			sha, err := hashFile(filepath.Join(prep.Root, rel))
			if err != nil {
				return receiptBinding, err
			}
			receipt.Transitions[rel] = sha
		}
		request, err := correctnessRequestFor(runtime, step)
		if err != nil {
			return receiptBinding, err
		}
		request.SourceTree, err = verifyCorrectnessSource(ctx, runtime, step, step.InputAfter)
		if err != nil {
			return receiptBinding, err
		}
		request.Runtime = f.JDK
		request.BootID = runtime.BootID
		request.ReservedNS, err = bootNow()
		if err != nil {
			return receiptBinding, err
		}
		dir := filepath.Join(prep.Root, "attempts", step.Row.ID)
		if err = os.Mkdir(dir, 0700); err != nil {
			return receiptBinding, err
		}
		path := filepath.Join(dir, "native-request.json")
		if err = writeJSON(path, request); err != nil {
			return receiptBinding, err
		}
		pin, err := hashFile(path)
		if err != nil {
			return receiptBinding, err
		}
		unit := "buildopt-eic-c-" + pin[:20] + ".service"
		args := nativeServiceArguments(unit, runner, path, pin)
		args[len(args)-4] = "correctness-child"
		args = append(args, freezePath, freezeSHA)
		if err = writeJSON(filepath.Join(dir, "service-request.json"), args); err != nil {
			return receiptBinding, err
		}
		raw, runErr := runNativeService(ctx, args, unit)
		if err = writeNew(filepath.Join(dir, "service.log"), raw); err != nil {
			return receiptBinding, err
		}
		if runErr != nil {
			return receiptBinding, fmt.Errorf("%s owned capture failed: %w", step.Row.ID, runErr)
		}
		var native nativeResult
		if err = readJSON(filepath.Join(dir, "native-result.json"), &native); err != nil {
			return receiptBinding, err
		}
		if !strings.HasPrefix(native.Cgroup, "0::/") || !strings.HasSuffix(native.Cgroup, "/"+unit) {
			return receiptBinding, errors.New("owned correctness cgroup mismatch")
		}
		cgroup := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(native.Cgroup, "0::"))
		if _, err = os.Stat(cgroup); !os.IsNotExist(err) {
			return receiptBinding, errors.New("owned correctness cgroup still exists")
		}
		closed, err := bootNow()
		if err != nil {
			return receiptBinding, err
		}
		if err = writeJSON(filepath.Join(dir, "closed.json"), correctnessClosed{unit, native.Cgroup, true, closed}); err != nil {
			return receiptBinding, err
		}
		required := []string{":server:precommit", ":server:forbiddenPatterns"}
		if step.Row.ExpectedFailure {
			required = []string{":server:forbiddenPatterns"}
		}
		if err = retainArmDiagnostic(prep.Root, step.Row.ID, step.Row.Arm, required); err != nil {
			return receiptBinding, err
		}
		complete, err := sealCorrectnessRow(runtime, step)
		if err != nil {
			return receiptBinding, err
		}
		receipt.Rows = append(receipt.Rows, complete)
		if err = rememberCorrectnessOrigin(runtime, step); err != nil {
			return receiptBinding, err
		}
		fmt.Fprintf(os.Stderr, "%s verified (%s)\n", step.Row.ID, step.TaskOutcome)
	}
	return receiptBinding, nil
}

func correctnessChild(args []string) int {
	if len(args) != 5 {
		return 64
	}
	path, pin, unit := args[0], args[1], args[2]
	if checkBinding(fileBinding{path, pin}) != nil {
		return 65
	}
	f, err := loadCorrectnessFreeze(fileBinding{args[3], args[4]})
	if err != nil {
		return 65
	}
	var r nativeRequest
	if readJSON(path, &r) != nil {
		return 65
	}
	var step correctnessStep
	for _, s := range f.Steps {
		if s.Row.ID == r.Row.ID {
			step = s
		}
	}
	expected, err := correctnessRequestFor(f.Runtime, step)
	if err != nil {
		return 65
	}
	self, err := ownExecutable()
	if err != nil || self != f.Runtime.Runner.Path {
		return 65
	}
	if r.Schema != expected.Schema || !reflect.DeepEqual(expected.Arguments, r.Arguments) || !reflect.DeepEqual(expected.Environment, r.Environment) || !reflect.DeepEqual(expected.Row, r.Row) || expected.Directory != r.Directory || expected.Root != r.Root || !reflect.DeepEqual(r.Runtime, f.JDK) || r.Runner != f.Runtime.Runner || !reflect.DeepEqual(r.Diagnostic, expected.Diagnostic) || r.TimeoutSeconds != 600 || checkBinding(*r.Diagnostic) != nil {
		return 65
	}
	boot, err := bootID()
	if err != nil || boot != r.BootID || boot != f.Runtime.BootID {
		return 65
	}
	now, err := bootNow()
	if err != nil || now < r.ReservedNS || now+int64(600*time.Second) > f.Runtime.DeadlineNS {
		return 65
	}
	group, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || !strings.HasSuffix(strings.TrimSpace(string(group)), "/"+unit) {
		return 65
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	checkInputs := func() bool {
		tree, e := verifyCorrectnessSource(ctx, f.Runtime, step, step.InputAfter)
		if e != nil || tree != r.SourceTree {
			return false
		}
		actual, e := inventory(filepath.Join(r.Root, "runtime/jdk-21"), []string{"bin", "conf", "lib", "release"})
		return e == nil && reflect.DeepEqual(actual, f.JDK)
	}
	if !checkInputs() {
		return 65
	}
	env, err := correctnessPrivateEnvironment(f.Runtime, r.Environment)
	if err != nil {
		return 65
	}
	syscall.Umask(022)
	dir := filepath.Dir(path)
	ownership := map[string]string{"unit": unit, "cgroup": strings.TrimSpace(string(group)), "invocationId": os.Getenv("INVOCATION_ID")}
	if writeJSON(filepath.Join(dir, "ownership.json"), ownership) != nil {
		return 65
	}
	selectors := []string{sourceInputs().TaskPath, sourceInputs().OwnerInputPath, "gradlew"}
	before, err := inventory(r.Directory, selectors)
	if err != nil || writeJSON(filepath.Join(dir, "source-before.json"), before) != nil {
		return 65
	}
	p, err := runConfiguredProcess(ctx, dir, r.Arguments[0], r.Arguments[1:], r.Directory, env, 600*time.Second)
	result := nativeResult{Schema: "buildopt.eic/native-result/v1", RowID: r.Row.ID, Process: p, Cgroup: ownership["cgroup"], InvocationID: ownership["invocationId"], InputState: "UNVERIFIED"}
	if checkInputs() {
		result.InputState = "VERIFIED"
	}
	after, e := inventory(r.Directory, selectors)
	if e != nil || !reflect.DeepEqual(before, after) || writeJSON(filepath.Join(dir, "source-after.json"), after) != nil {
		return 65
	}
	if writeJSON(filepath.Join(dir, "native-result.json"), result) != nil {
		return 65
	}
	if err != nil || result.InputState != "VERIFIED" {
		return 1
	}
	// Native failures are retained normally and judged by the independent row
	// checker. A capture helper success does not reclassify the native result.
	return 0
}

func checkCorrectness(binding fileBinding) ([]correctnessRowCheck, error) {
	if err := checkBinding(binding); err != nil {
		return nil, err
	}
	var receipt correctnessReceipt
	if err := readJSON(binding.Path, &receipt); err != nil {
		return nil, err
	}
	f, err := loadCorrectnessFreeze(receipt.Freeze)
	if err != nil {
		return nil, err
	}
	if receipt.Schema != "buildopt.eic/correctness-receipt/v1" || binding.Path != filepath.Join(f.Runtime.Root, "correctness-receipt.json") || receipt.Stopped != "" || len(receipt.Rows) != 12 || len(receipt.Transitions) != 24 {
		return nil, errors.New("correctness capture is incomplete or stopped")
	}
	f.Runtime.metadata, err = newMetadataSession(f.Runtime.Metadata)
	if err != nil {
		return nil, err
	}
	f.Runtime.provenance, err = newProvenanceSession(f.Runtime.Provenance, f.Runtime.Root)
	if err != nil {
		return nil, err
	}
	if err = checkBinding(receipt.Observations); err != nil {
		return nil, err
	}
	for path, pin := range receipt.Transitions {
		if !filepath.IsLocal(path) || filepath.Clean(path) != path {
			return nil, errors.New("unsafe transition binding")
		}
		if err := checkBinding(fileBinding{filepath.Join(f.Runtime.Root, path), pin}); err != nil {
			return nil, err
		}
	}
	results := []correctnessRowCheck{}
	for i, row := range receipt.Rows {
		if err = validateCorrectnessSequence(receipt.Rows[:i], row.RowID); err != nil {
			return nil, err
		}
		dir := filepath.Join(f.Runtime.Root, "attempts", row.RowID)
		if row.Complete.Path != filepath.Join(dir, "row-complete.json") {
			return nil, errors.New("capture path drift")
		}
		if err = checkBinding(row.Complete); err != nil {
			return nil, err
		}
		var files map[string]string
		if err = readJSON(row.Complete.Path, &files); err != nil {
			return nil, err
		}
		for name, pin := range files {
			if filepath.Base(name) != name {
				return nil, errors.New("unsafe row artifact")
			}
			if err = checkBinding(fileBinding{filepath.Join(dir, name), pin}); err != nil {
				return nil, err
			}
		}
		result, err := inspectCorrectnessRow(f.Runtime, f.Steps[i])
		if err != nil {
			return nil, err
		}
		var captured correctnessRowCheck
		if err = readJSONLimit(filepath.Join(dir, "row-check.json"), &captured, 16<<20); err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(result, captured) {
			return nil, errors.New("row summary differs from independent reconstruction")
		}
		if err = rememberCorrectnessOrigin(f.Runtime, f.Steps[i]); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	attempts, err := os.ReadDir(filepath.Join(f.Runtime.Root, "attempts"))
	if err != nil || len(attempts) != 12 {
		return nil, errors.New("missing or extra correctness starts")
	}
	return results, nil
}
