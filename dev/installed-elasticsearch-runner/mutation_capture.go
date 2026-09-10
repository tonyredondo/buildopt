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

const mutationClass = "LOCAL_REAL_GRADLE_MUTATION_QUALIFICATION"

var mutationInputSelectors = []string{".eic-phase", "settings.gradle", "build.gradle", "buildSrc/build.gradle", "buildSrc/src", "src"}

type mutationFreeze struct {
	Schema          string       `json:"schemaVersion"`
	Class           string       `json:"environmentClass"`
	Root            string       `json:"root"`
	Source          string       `json:"sourceWorktree"`
	Revision        string       `json:"revision"`
	Runner          fileBinding  `json:"runner"`
	Task            fileBinding  `json:"taskSource"`
	Problems        fileBinding  `json:"problemsSource"`
	Diagnostic      fileBinding  `json:"diagnostic"`
	JDK             string       `json:"jdk"`
	Gradle          string       `json:"gradle"`
	JDKInventory    []entry      `json:"jdkInventory"`
	GradleInventory []entry      `json:"gradleInventory"`
	Rows            []row        `json:"rows"`
	BootID          string       `json:"bootId"`
	StartNS         int64        `json:"startBootNanoseconds"`
	DeadlineNS      int64        `json:"deadlineBootNanoseconds"`
	MaxStarts       int          `json:"maximumStarts"`
	MaxBytes        int64        `json:"maximumBytes"`
	Correctness     *fileBinding `json:"correctnessReceipt,omitempty"`
}

type mutationRequest struct {
	Root           string      `json:"root"`
	Freeze         fileBinding `json:"freeze"`
	Runner         fileBinding `json:"runner"`
	Row            row         `json:"row"`
	Directory      string      `json:"directory"`
	Arguments      []string    `json:"arguments"`
	Environment    []string    `json:"environment"`
	TimeoutSeconds int         `json:"timeoutSeconds"`
	Inputs         []entry     `json:"inputs"`
}

func mutationRequestFor(f mutationFreeze, r row, pin string) (mutationRequest, error) {
	matched := false
	for _, expected := range mutationRowsForClass(f.Class) {
		if reflect.DeepEqual(r, expected) {
			matched = true
		}
	}
	if !matched {
		return mutationRequest{}, errors.New("unallocated mutation row")
	}
	kind, phase, _ := strings.Cut(r.State, ":")
	attempt := filepath.Join(f.Root, "attempts", r.ID)
	state := filepath.Join(f.Root, "state", r.Arm)
	args := []string{"/usr/bin/taskset", "--cpu-list", "0-3", filepath.Join(f.Gradle, "bin", "gradle"), "--offline", "--no-daemon", "--no-configuration-cache", "--build-cache", "--console=plain", "--max-workers=2", "--stacktrace", "--init-script", f.Diagnostic.Path, "-Dorg.gradle.internal.operations.trace=" + filepath.Join(attempt, "operations"), "-Pfixture=" + kind, "-Pphase=" + phase, "forbiddenPatterns"}
	if f.Class == correctnessMutationClass {
		args[2] = "0-7"
		adjusted := []string{}
		for _, arg := range args {
			if arg == "--no-configuration-cache" {
				continue
			}
			if arg == "--max-workers=2" {
				arg = "--max-workers=8"
			}
			adjusted = append(adjusted, arg)
		}
		args = adjusted
	}
	env := []string{"PATH=" + filepath.Join(f.JDK, "bin") + ":/usr/bin:/bin", "JAVA_HOME=" + f.JDK, "GRADLE_USER_HOME=" + filepath.Join(state, "gradle"), "TMPDIR=" + filepath.Join(state, "tmp"), "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TZ=UTC", "JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8 -Duser.home=" + filepath.Join(state, "user"), "BUILDOPT_TASK_GRAPH_OUTPUT=" + filepath.Join(attempt, "task-graph.jsonl")}
	return mutationRequest{Root: f.Root, Freeze: fileBinding{filepath.Join(f.Root, "freeze.json"), pin}, Runner: f.Runner, Row: r, Directory: filepath.Join(f.Root, "work", kind, r.Arm), Arguments: args, Environment: env, TimeoutSeconds: 120}, nil
}

func mutationServiceArguments(unit, runner, request, pin string) []string {
	return []string{"--user", "--quiet", "--wait", "--pipe", "--collect", "--service-type=exec", "--unit=" + unit, "--property=KillMode=control-group", "--property=RuntimeMaxSec=130s", "--property=TimeoutStopSec=2s", "--property=Restart=no", "--", runner, "mutation-child", request, pin, unit}
}

func checkMutationInputs(r row, entries []entry) error {
	files := mutationBaseFiles(nil, nil)
	const sourcePrefix = "buildSrc/src/main/java/org/elasticsearch/gradle/internal/"
	taskPath, problemsPath := sourcePrefix+"precommit/ForbiddenPatternsTask.java", sourcePrefix+"conventions/problems/ElasticsearchBuildProblems.java"
	expected := map[string]string{}
	for p, b := range files {
		expected[p] = digest(b)
	}
	expected[taskPath] = sourceInputs().TaskPreimageSHA256
	if r.Arm == "N1" {
		expected[taskPath] = sourceInputs().TaskPostimageSHA256
	}
	expected[problemsPath] = problemsSourceSHA256
	kind, phase, _ := strings.Cut(r.State, ":")
	if phase == "after" {
		expected[".eic-phase"] = digest([]byte("after"))
		switch kind {
		case "relative-rename":
			expected["src/renamed.txt"] = expected["src/input.txt"]
			delete(expected, "src/input.txt")
		case "all-source-removal":
			delete(expected, "src/input.txt")
			delete(expected, "src/excluded.txt")
		case "malformed-utf8":
			expected["src/input.txt"] = digest([]byte{0xc3, 0x28})
		}
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.Type == "directory" {
			if e.Mode != 0700 {
				return errors.New("fixture directory mode drift")
			}
			continue
		}
		if e.Type != "file" || e.Mode != 0600 || expected[e.Path] == "" || expected[e.Path] != e.SHA256 || seen[e.Path] {
			return fmt.Errorf("fixture input drift: %s", e.Path)
		}
		seen[e.Path] = true
	}
	if len(seen) != len(expected) {
		return errors.New("missing fixture inputs")
	}
	return nil
}

func freezeMutations(ctx context.Context, root, source, gradle, jdk string) (fileBinding, error) {
	return freezeMutationAllocation(ctx, root, source, gradle, jdk, nil)
}

func freezeMutationAllocation(ctx context.Context, root, source, gradle, jdk string, correctness *fileBinding) (fileBinding, error) {
	var binding fileBinding
	parent, err := filepath.EvalSymlinks(filepath.Dir(root))
	if err != nil || !filepath.IsAbs(root) || filepath.Clean(root) != root || parent != filepath.Dir(root) {
		return binding, errors.New("new canonical mutation root required")
	}
	for _, p := range []string{source, gradle, jdk} {
		resolved, e := filepath.EvalSymlinks(p)
		if e != nil || !filepath.IsAbs(p) || resolved != p {
			return binding, errors.New("canonical source and runtimes required")
		}
	}
	base, err := nativeGit(ctx, source, "rev-parse", "HEAD")
	if err != nil || base != makeProtocol().Revisions[0] {
		return binding, errors.New("source B0 drift")
	}
	status, err := nativeGit(ctx, source, "status", "--porcelain=v1", "--untracked-files=no")
	if err != nil || status != "" {
		return binding, errors.New("source is dirty")
	}
	runner, err := ownExecutable()
	if err != nil {
		return binding, err
	}
	f := mutationFreeze{Schema: "buildopt.eic/mutation-freeze/v1", Class: mutationClass, Root: root, Source: source, Revision: base, Runner: fileBinding{Path: runner}, Task: fileBinding{filepath.Join(source, sourceInputs().TaskPath), sourceInputs().TaskPreimageSHA256}, Problems: fileBinding{filepath.Join(source, problemsSourcePath), problemsSourceSHA256}, JDK: jdk, Gradle: gradle, Rows: mutationRows(), MaxStarts: 24, MaxBytes: 5 << 30}
	if correctness != nil {
		if _, err := checkCorrectness(*correctness); err != nil {
			return binding, err
		}
		if root != filepath.Join(filepath.Dir(correctness.Path), "mutations") || source != filepath.Join(filepath.Dir(correctness.Path), "arms/N0") {
			return binding, errors.New("public mutations must belong to the verified C campaign")
		}
		f.Class = correctnessMutationClass
		f.Correctness = correctness
		f.Rows = mutationRowsForClass(f.Class)
	}
	if err = checkBinding(f.Task); err != nil {
		return binding, err
	}
	if err = checkBinding(f.Problems); err != nil {
		return binding, err
	}
	f.Runner.SHA256, err = hashFile(runner)
	if err != nil {
		return binding, err
	}
	f.JDKInventory, err = inventory(jdk, []string{"bin", "conf", "lib", "release"})
	if err != nil {
		return binding, err
	}
	f.GradleInventory, err = inventory(gradle, []string{"bin", "lib"})
	if err != nil {
		return binding, err
	}
	release, err := os.ReadFile(filepath.Join(jdk, "release"))
	if err != nil || !strings.Contains(string(release), "JAVA_RUNTIME_VERSION=\"21.0.12+8-LTS\"") {
		return binding, errors.New("wrong locked JDK")
	}
	if _, err = os.Stat(filepath.Join(gradle, "lib", "gradle-core-9.7.1.jar")); err != nil {
		return binding, err
	}
	f.BootID, err = bootID()
	if err != nil {
		return binding, err
	}
	f.StartNS, err = bootNow()
	if err != nil {
		return binding, err
	}
	f.DeadlineNS = f.StartNS + int64(2700*time.Second)
	if err = os.Mkdir(root, 0700); err != nil {
		return binding, err
	}
	for _, dir := range []string{"attempts", "work", "state/N0/gradle", "state/N0/tmp", "state/N0/user", "state/N1/gradle", "state/N1/tmp", "state/N1/user"} {
		if err = os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			return binding, err
		}
	}
	f.Diagnostic = fileBinding{filepath.Join(root, "native-diagnostic.init.gradle"), digest([]byte(nativeDiagnosticScript))}
	if err = writeNew(f.Diagnostic.Path, []byte(nativeDiagnosticScript)); err != nil {
		return binding, err
	}
	binding.Path = filepath.Join(root, "freeze.json")
	if err = writeJSON(binding.Path, f); err != nil {
		return binding, err
	}
	binding.SHA256, err = hashFile(binding.Path)
	return binding, err
}

func loadMutationFreeze(binding fileBinding, live bool) (mutationFreeze, error) {
	var f mutationFreeze
	if err := checkBinding(binding); err != nil {
		return f, err
	}
	if err := readJSON(binding.Path, &f); err != nil {
		return f, err
	}
	if f.Schema != "buildopt.eic/mutation-freeze/v1" || (f.Class != mutationClass && f.Class != correctnessMutationClass) || filepath.Join(f.Root, "freeze.json") != binding.Path || f.Revision != makeProtocol().Revisions[0] || !reflect.DeepEqual(f.Rows, mutationRowsForClass(f.Class)) || f.MaxStarts != 24 || f.MaxBytes != 5<<30 || f.StartNS <= 0 || f.DeadlineNS-f.StartNS != int64(2700*time.Second) || f.Task.SHA256 != sourceInputs().TaskPreimageSHA256 || f.Problems.SHA256 != problemsSourceSHA256 || f.Diagnostic.SHA256 != digest([]byte(nativeDiagnosticScript)) {
		return f, errors.New("mutation freeze contract drift")
	}
	if (f.Class == correctnessMutationClass) != (f.Correctness != nil) {
		return f, errors.New("public mutation parent binding missing or unexpected")
	}
	if f.Correctness != nil {
		if f.Root != filepath.Join(filepath.Dir(f.Correctness.Path), "mutations") || f.Source != filepath.Join(filepath.Dir(f.Correctness.Path), "arms/N0") {
			return f, errors.New("public mutation parent path drift")
		}
		if err := checkBinding(*f.Correctness); err != nil {
			return f, err
		}
	}
	for _, b := range []fileBinding{f.Runner, f.Task, f.Problems, f.Diagnostic} {
		if err := checkBinding(b); err != nil {
			return f, err
		}
	}
	if live {
		boot, err := bootID()
		if err != nil || boot != f.BootID {
			return f, errors.New("boot identity drift")
		}
		now, err := bootNow()
		if err != nil || now < f.StartNS || now >= f.DeadlineNS {
			return f, errors.New("mutation allocation expired")
		}
		jdk, err := inventory(f.JDK, []string{"bin", "conf", "lib", "release"})
		if err != nil || !reflect.DeepEqual(jdk, f.JDKInventory) {
			return f, errors.New("JDK bytes drift")
		}
		gradle, err := inventory(f.Gradle, []string{"bin", "lib"})
		if err != nil || !reflect.DeepEqual(gradle, f.GradleInventory) {
			return f, errors.New("Gradle bytes drift")
		}
	}
	return f, nil
}

func mutationDiskGuard(f mutationFreeze) error {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(f.Root, &stats); err != nil {
		return err
	}
	if stats.Bavail*uint64(stats.Bsize) < 20<<30 {
		return errors.New("mutation qualification requires 20 GiB free")
	}
	var total int64
	return filepath.Walk(f.Root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		total += info.Size()
		if total > f.MaxBytes {
			return errors.New("mutation root exceeds frozen disk cap")
		}
		return nil
	})
}

func retainSelected(source, target string, selectors []string) ([]entry, error) {
	before, err := inventory(source, selectors)
	if err != nil {
		return nil, err
	}
	if err = os.Mkdir(target, 0700); err != nil {
		return nil, err
	}
	for _, s := range selectors {
		src, dst := filepath.Join(source, s), filepath.Join(target, s)
		if _, err = os.Lstat(src); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return nil, err
		}
		if err = copyTree(src, dst); err != nil {
			return nil, err
		}
	}
	after, err := inventory(target, selectors)
	if err != nil || !reflect.DeepEqual(before, after) {
		return nil, errors.New("retention changed declared bytes or modes")
	}
	return after, nil
}

func mutationChild(args []string) int {
	if len(args) != 3 || checkBinding(fileBinding{args[0], args[1]}) != nil {
		return 65
	}
	var r mutationRequest
	if readJSON(args[0], &r) != nil {
		return 65
	}
	f, err := loadMutationFreeze(r.Freeze, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 65
	}
	expected, err := mutationRequestFor(f, r.Row, r.Freeze.SHA256)
	if err != nil {
		return 65
	}
	expected.Inputs = r.Inputs
	self, err := ownExecutable()
	if err != nil || self != f.Runner.Path || !reflect.DeepEqual(r, expected) || args[0] != filepath.Join(f.Root, "attempts", r.Row.ID, "request.json") || args[2] != "buildopt-eic-qm-"+args[1][:20]+".service" {
		return 65
	}
	cgroup, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || !strings.HasSuffix(strings.TrimSpace(string(cgroup)), "/"+args[2]) {
		return 65
	}
	inputs, err := inventory(r.Directory, mutationInputSelectors)
	if err != nil || !reflect.DeepEqual(inputs, r.Inputs) || checkMutationInputs(r.Row, inputs) != nil {
		return 65
	}
	if mutationDiskGuard(f) != nil {
		return 65
	}
	dir := filepath.Dir(args[0])
	invocation := os.Getenv("INVOCATION_ID")
	if err = writeJSON(filepath.Join(dir, "ownership.json"), map[string]string{"unit": args[2], "cgroup": strings.TrimSpace(string(cgroup)), "invocationId": invocation}); err != nil {
		return 65
	}
	remaining := time.Duration(f.DeadlineNS)
	now, err := bootNow()
	if err != nil {
		return 65
	}
	remaining -= time.Duration(now)
	ctx, cancel := context.WithTimeout(context.Background(), remaining)
	defer cancel()
	syscall.Umask(022)
	p, runErr := runConfiguredProcess(ctx, dir, r.Arguments[0], r.Arguments[1:], r.Directory, r.Environment, 120*time.Second)
	result := nativeResult{Schema: "buildopt.eic/mutation-result/v1", RowID: r.Row.ID, Process: p, Cgroup: strings.TrimSpace(string(cgroup)), InvocationID: invocation, InputState: "UNVERIFIED"}
	inputs, err = inventory(r.Directory, mutationInputSelectors)
	if err == nil && reflect.DeepEqual(inputs, r.Inputs) {
		if _, err = loadMutationFreeze(r.Freeze, true); err == nil {
			result.InputState = "VERIFIED"
		}
	}
	if writeJSON(filepath.Join(dir, "result.json"), result) != nil {
		return 65
	}
	// Preserve expected Gradle failure in process.json. Service success means
	// capture completed; the parent checks the actual task failure and cause.
	if runErr != nil || result.InputState != "VERIFIED" || (p.Outcome != "SUCCESS" && !(r.Row.ExpectedFailure && p.Outcome == "FAILURE")) {
		return 1
	}
	return 0
}

type mutationRowEvidence struct {
	Row         row                 `json:"row"`
	Observation mutationObservation `json:"observation"`
	Seed        taskSnapshot        `json:"seedSnapshot"`
	Outputs     []entry             `json:"outputs"`
}

func checkMutationRow(f mutationFreeze, r row, freezePin string) (mutationRowEvidence, error) {
	out := mutationRowEvidence{Row: r}
	dir := filepath.Join(f.Root, "attempts", r.ID)
	var request mutationRequest
	if err := readJSON(filepath.Join(dir, "request.json"), &request); err != nil {
		return out, err
	}
	expected, err := mutationRequestFor(f, r, freezePin)
	if err != nil {
		return out, err
	}
	inputs, err := inventory(filepath.Join(dir, "inputs"), mutationInputSelectors)
	if err != nil {
		return out, err
	}
	expected.Inputs = inputs
	if !reflect.DeepEqual(request, expected) {
		return out, errors.New("retained request or inputs drift")
	}
	if err = checkMutationInputs(r, inputs); err != nil {
		return out, err
	}
	if err = rejectExtraRetainedOutputs(filepath.Join(dir, "inputs"), inputs); err != nil {
		return out, err
	}
	var result nativeResult
	if err = readJSON(filepath.Join(dir, "result.json"), &result); err != nil {
		return out, err
	}
	var p processRecord
	if err = readJSON(filepath.Join(dir, "process.json"), &p); err != nil {
		return out, err
	}
	if result.Schema != "buildopt.eic/mutation-result/v1" || result.RowID != r.ID || result.InputState != "VERIFIED" || !reflect.DeepEqual(result.Process, p) || result.InvocationID == "" {
		return out, errors.New("mutation process record drift")
	}
	if err = checkProcess(dir, p, f.StartNS, f.DeadlineNS); err != nil {
		return out, err
	}
	if p.EndNS-p.StartNS > int64(121*time.Second) {
		return out, errors.New("mutation process exceeded deadline")
	}
	requestPin, err := hashFile(filepath.Join(dir, "request.json"))
	if err != nil {
		return out, err
	}
	unit := "buildopt-eic-qm-" + requestPin[:20] + ".service"
	var serviceArgs []string
	if err = readJSON(filepath.Join(dir, "service-request.json"), &serviceArgs); err != nil || !reflect.DeepEqual(serviceArgs, mutationServiceArguments(unit, f.Runner.Path, filepath.Join(dir, "request.json"), requestPin)) {
		return out, errors.New("service ownership or bounds drift")
	}
	var ownership map[string]string
	if err = readJSON(filepath.Join(dir, "ownership.json"), &ownership); err != nil {
		return out, err
	}
	if len(ownership) != 3 || ownership["unit"] != unit || ownership["cgroup"] != result.Cgroup || ownership["invocationId"] != result.InvocationID || !strings.HasPrefix(result.Cgroup, "0::/") || !strings.HasSuffix(result.Cgroup, "/"+unit) {
		return out, errors.New("mutation cgroup identity drift")
	}
	var closed struct {
		Unit   string `json:"unit"`
		Cgroup string `json:"cgroup"`
		Absent bool   `json:"absent"`
		EndNS  int64  `json:"endBootNanoseconds"`
	}
	if err = readJSON(filepath.Join(dir, "closed.json"), &closed); err != nil || closed.Unit != unit || closed.Cgroup != result.Cgroup || !closed.Absent || closed.EndNS < p.EndNS || closed.EndNS > f.DeadlineNS {
		return out, errors.New("owned service not closed")
	}
	report, err := gradlecriticalpath.Analyze(filepath.Join(dir, "operations-log.txt"), filepath.Join(dir, "task-graph.jsonl"), map[string]string{"N0": "control", "N1": "candidate"}[r.Arm])
	if err != nil {
		return out, err
	}
	targetCount, seedCount := 0, 0
	for _, t := range report.Tasks {
		if t.Identity == ":forbiddenPatterns" {
			targetCount++
			out.Observation.Outcome = t.Outcome
			// ExecuteTask reports the original class; the graph and input
			// snapshot report Gradle's generated decorated class separately.
			if t.TaskClass != "org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsTask" {
				return out, errors.New("wrong owner implementation")
			}
		}
		if t.Identity == ":seedForbiddenPatterns" {
			seedCount++
			if t.Outcome != "EXECUTED" && t.Outcome != "FROM-CACHE" {
				return out, errors.New("seed did not populate or restore cache")
			}
		}
		if t.Outcome == "FAILED" && !(r.ExpectedFailure && t.Identity == ":forbiddenPatterns") {
			return out, errors.New("unexpected task failure")
		}
	}
	kind, phase, _ := strings.Cut(r.State, ":")
	if targetCount != 1 || (phase == "before" && seedCount != 1) || (phase == "after" && seedCount != 0) {
		return out, errors.New("owner/seed graph drift")
	}
	out.Observation.Process = p
	stderr, err := os.ReadFile(filepath.Join(dir, "stderr.log"))
	if err != nil {
		return out, err
	}
	out.Observation.UTF8Failure = strings.Contains(string(stderr), "Failed to read ") && strings.Contains(string(stderr), "as UTF_8")
	snapshots, err := mutationSnapshots(filepath.Join(dir, "operations-log.txt"))
	if err != nil {
		return out, err
	}
	out.Observation.Cache, out.Seed = snapshots[":forbiddenPatterns"], snapshots[":seedForbiddenPatterns"]
	want := "EXECUTED"
	if phase == "before" && r.Arm == "N1" {
		want = "FROM-CACHE"
	}
	if phase == "after" && kind == "all-source-removal" {
		want = "NO-SOURCE"
	}
	if r.ExpectedFailure {
		want = "FAILED"
	}
	if out.Observation.Outcome != want || (p.Outcome == "FAILURE") != r.ExpectedFailure || (r.ExpectedFailure && !out.Observation.UTF8Failure) {
		return out, fmt.Errorf("%s: observed %s/%s, required %s (UTF-8 cause %t)", r.ID, out.Observation.Outcome, p.Outcome, want, out.Observation.UTF8Failure)
	}
	if phase == "before" && r.Arm == "N1" && (out.Observation.Cache.Hash == "" || out.Observation.Cache.Hash != out.Seed.Hash) {
		return out, errors.New("target did not restore its observed seed key")
	}
	selectors, err := graphOutputSelectors(filepath.Join(dir, "task-graph.jsonl"), []string{":forbiddenPatterns"})
	if err != nil {
		return out, err
	}
	wantSelectors := []string{"build/markers/forbiddenPatterns"}
	if phase == "before" {
		wantSelectors = append(wantSelectors, "build/markers/seedForbiddenPatterns")
	}
	if !reflect.DeepEqual(selectors, wantSelectors) {
		return out, errors.New("unexpected owner outputs")
	}
	out.Outputs, err = inventory(filepath.Join(dir, "outputs"), selectors)
	if err != nil {
		return out, err
	}
	if err = rejectExtraRetainedOutputs(filepath.Join(dir, "outputs"), out.Outputs); err != nil {
		return out, err
	}
	for _, e := range out.Outputs {
		if (want == "FAILED" || want == "NO-SOURCE") && e.Type == "absent" {
			// A skipped or failed action may retain the previous marker.
			// The required contract is exact N0/N1 bytes and absence parity.
			continue
		}
		if e.Type != "file" || e.Size != 4 || e.SHA256 != digest([]byte("done")) || e.Mode != 0644 {
			return out, errors.New("owner marker bytes/type/mode drift")
		}
	}
	return out, nil
}

type mutationSummary struct {
	Class         string                `json:"environmentClass"`
	Rows          []mutationRowEvidence `json:"rows"`
	Starts        int                   `json:"starts"`
	Passed        bool                  `json:"passed"`
	ValueAdmitted bool                  `json:"valueAdmitted"`
}
type mutationReceipt struct {
	Schema    string      `json:"schemaVersion"`
	Freeze    fileBinding `json:"freeze"`
	Artifacts []entry     `json:"artifacts"`
}

var mutationEvidenceSelectors = []string{"freeze.json", "native-diagnostic.init.gradle", "run-start.json", "run-result.json", "attempts"}

func mutationEvidenceSummary(f mutationFreeze, pin string) (mutationSummary, error) {
	summary := mutationSummary{Class: f.Class, Rows: []mutationRowEvidence{}}
	dirs, err := os.ReadDir(filepath.Join(f.Root, "attempts"))
	if err != nil || len(dirs) != 24 {
		return summary, errors.New("incomplete mutation allocation")
	}
	last := f.StartNS
	pairs := map[string]mutationRowEvidence{}
	for _, r := range f.Rows {
		e, err := checkMutationRow(f, r, pin)
		if err != nil {
			return summary, err
		}
		if e.Observation.Process.StartNS < last {
			return summary, errors.New("overlapping mutation starts")
		}
		last = e.Observation.Process.EndNS
		summary.Rows = append(summary.Rows, e)
		summary.Starts++
		kind, phase, _ := strings.Cut(r.State, ":")
		if r.Arm == "N1" {
			control, exists := pairs[r.State]
			if !exists || !reflect.DeepEqual(control.Outputs, e.Outputs) {
				return summary, errors.New("native/candidate output mismatch")
			}
			if phase == "after" {
				if err := checkMutationTransition(kind, pairs[kind+":candidate-before"].Observation, e.Observation); err != nil {
					return summary, err
				}
			} else {
				pairs[kind+":candidate-before"] = e
			}
		} else {
			pairs[r.State] = e
		}
	}
	summary.Passed = true
	return summary, nil
}

func checkMutations(binding fileBinding) (mutationSummary, error) {
	var receipt mutationReceipt
	if err := checkBinding(binding); err != nil {
		return mutationSummary{}, err
	}
	if err := readJSONLimit(binding.Path, &receipt, 16<<20); err != nil {
		return mutationSummary{}, err
	}
	f, err := loadMutationFreeze(receipt.Freeze, false)
	if err != nil {
		return mutationSummary{}, err
	}
	if f.Correctness != nil {
		if _, err := checkCorrectness(*f.Correctness); err != nil {
			return mutationSummary{}, err
		}
	}
	if receipt.Schema != "buildopt.eic/mutation-receipt/v1" || binding.Path != filepath.Join(f.Root, "receipt.json") {
		return mutationSummary{}, errors.New("mutation receipt identity drift")
	}
	actual, err := inventory(f.Root, mutationEvidenceSelectors)
	if err != nil || !reflect.DeepEqual(actual, receipt.Artifacts) {
		return mutationSummary{}, errors.New("mutation evidence bytes/type/mode drift")
	}
	summary, err := mutationEvidenceSummary(f, receipt.Freeze.SHA256)
	if err != nil {
		return summary, err
	}
	var recorded mutationSummary
	if err = readJSON(filepath.Join(f.Root, "run-result.json"), &recorded); err != nil || !reflect.DeepEqual(summary, recorded) {
		return summary, errors.New("mutation summary does not match raw evidence")
	}
	return summary, nil
}

func runMutations(ctx context.Context, binding fileBinding) (receiptBinding fileBinding, retErr error) {
	f, err := loadMutationFreeze(binding, true)
	if err != nil {
		return receiptBinding, err
	}
	self, err := ownExecutable()
	if err != nil || self != f.Runner.Path {
		return receiptBinding, errors.New("runner differs from freeze")
	}
	if err = writeJSON(filepath.Join(f.Root, "run-start.json"), binding); err != nil {
		return receiptBinding, err
	}
	defer func() {
		if retErr != nil {
			_ = writeJSON(filepath.Join(f.Root, "run-result.json"), map[string]string{"status": "FAILED_QUALIFICATION", "error": retErr.Error()})
		}
		artifacts, e := inventory(f.Root, mutationEvidenceSelectors)
		if e != nil {
			retErr = errors.Join(retErr, e)
			return
		}
		receiptBinding.Path = filepath.Join(f.Root, "receipt.json")
		if e = writeJSON(receiptBinding.Path, mutationReceipt{"buildopt.eic/mutation-receipt/v1", binding, artifacts}); e != nil {
			retErr = errors.Join(retErr, e)
			return
		}
		receiptBinding.SHA256, e = hashFile(receiptBinding.Path)
		retErr = errors.Join(retErr, e)
	}()
	task, err := os.ReadFile(f.Task.Path)
	if err != nil {
		return receiptBinding, err
	}
	problems, err := os.ReadFile(f.Problems.Path)
	if err != nil {
		return receiptBinding, err
	}
	for _, r := range f.Rows {
		if ctx.Err() != nil {
			return receiptBinding, ctx.Err()
		}
		if _, err = loadMutationFreeze(binding, true); err != nil {
			return receiptBinding, err
		}
		if err = mutationDiskGuard(f); err != nil {
			return receiptBinding, err
		}
		dir := filepath.Join(f.Root, "attempts", r.ID)
		if err = os.Mkdir(dir, 0700); err != nil {
			return receiptBinding, err
		}
		project, err := prepareMutationProject(f.Root, r, task, problems)
		if err != nil {
			return receiptBinding, err
		}
		request, err := mutationRequestFor(f, r, binding.SHA256)
		if err != nil {
			return receiptBinding, err
		}
		request.Inputs, err = retainSelected(project, filepath.Join(dir, "inputs"), mutationInputSelectors)
		if err != nil {
			return receiptBinding, err
		}
		if err = checkMutationInputs(r, request.Inputs); err != nil {
			return receiptBinding, err
		}
		path := filepath.Join(dir, "request.json")
		if err = writeJSON(path, request); err != nil {
			return receiptBinding, err
		}
		pin, err := hashFile(path)
		if err != nil {
			return receiptBinding, err
		}
		unit := "buildopt-eic-qm-" + pin[:20] + ".service"
		args := mutationServiceArguments(unit, self, path, pin)
		if err = writeJSON(filepath.Join(dir, "service-request.json"), args); err != nil {
			return receiptBinding, err
		}
		serviceCtx, cancel := context.WithTimeout(ctx, 140*time.Second)
		raw, runErr := runNativeService(serviceCtx, args, unit)
		cancel()
		if err = writeNew(filepath.Join(dir, "service.log"), raw); err != nil {
			return receiptBinding, err
		}
		var result nativeResult
		if err = readJSON(filepath.Join(dir, "result.json"), &result); err != nil {
			return receiptBinding, errors.Join(runErr, err)
		}
		if !strings.HasPrefix(result.Cgroup, "0::/") || !strings.HasSuffix(result.Cgroup, "/"+unit) {
			return receiptBinding, errors.New("invalid owned cgroup")
		}
		cgroup := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(result.Cgroup, "0::"))
		if _, err = os.Stat(cgroup); !os.IsNotExist(err) {
			return receiptBinding, errors.New("owned mutation cgroup still exists")
		}
		now, err := bootNow()
		if err != nil {
			return receiptBinding, err
		}
		if err = writeJSON(filepath.Join(dir, "closed.json"), struct {
			Unit   string `json:"unit"`
			Cgroup string `json:"cgroup"`
			Absent bool   `json:"absent"`
			EndNS  int64  `json:"endBootNanoseconds"`
		}{unit, result.Cgroup, true, now}); err != nil {
			return receiptBinding, err
		}
		if runErr != nil {
			return receiptBinding, fmt.Errorf("%s service: %w", r.ID, runErr)
		}
		selectors, err := graphOutputSelectors(filepath.Join(dir, "task-graph.jsonl"), []string{":forbiddenPatterns"})
		if err != nil {
			return receiptBinding, err
		}
		if _, err = retainSelected(project, filepath.Join(dir, "outputs"), selectors); err != nil {
			return receiptBinding, err
		}
		observation, err := checkMutationRow(f, r, binding.SHA256)
		if err != nil {
			return receiptBinding, err
		}
		if err = writeJSON(filepath.Join(dir, "checked.json"), observation); err != nil {
			return receiptBinding, err
		}
	}
	summary, err := mutationEvidenceSummary(f, binding.SHA256)
	if err != nil {
		return receiptBinding, err
	}
	if err = mutationDiskGuard(f); err != nil {
		return receiptBinding, err
	}
	return receiptBinding, writeJSON(filepath.Join(f.Root, "run-result.json"), summary)
}
