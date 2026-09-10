package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
)

type nativeRequest struct {
	Schema         string       `json:"schemaVersion"`
	Root           string       `json:"root"`
	Runner         fileBinding  `json:"runner"`
	Row            row          `json:"row"`
	Directory      string       `json:"directory"`
	Arguments      []string     `json:"arguments"`
	Environment    []string     `json:"environment"`
	TimeoutSeconds int          `json:"timeoutSeconds"`
	BootID         string       `json:"bootId"`
	ReservedNS     int64        `json:"reservedBootNanoseconds"`
	SourceTree     string       `json:"sourceTree"`
	Runtime        []entry      `json:"runtime"`
	Diagnostic     *fileBinding `json:"diagnostic,omitempty"`
	Seed           *fileBinding `json:"seed,omitempty"`
}

type nativeResult struct {
	Schema       string        `json:"schemaVersion"`
	RowID        string        `json:"rowId"`
	Process      processRecord `json:"process"`
	Cgroup       string        `json:"cgroup"`
	InvocationID string        `json:"invocationId"`
	InputState   string        `json:"inputState"`
}

// Native preparation is a prerequisite, not performance evidence. Subsequent
// diagnostics and candidate rows are admitted only by their additional gates.
func makeNativeRequest(root, runner string, r row) (nativeRequest, error) {
	if !filepath.IsAbs(root) || !filepath.IsAbs(runner) || strings.ContainsAny(root, "\n\r\"'\\") {
		return nativeRequest{}, errors.New("canonical absolute native paths required")
	}
	if r.Block != "P" || r.State != "unpatched-preparation" {
		return nativeRequest{}, errors.New("only native preparation is admitted by this entrypoint")
	}
	matched := false
	for _, expected := range makeProtocol().Rows[:3] {
		if reflect.DeepEqual(r, expected) {
			matched = true
		}
	}
	if !matched {
		return nativeRequest{}, errors.New("native preparation row drift")
	}
	directory := filepath.Join(root, "arms", r.Arm)
	state := filepath.Join(root, "state", r.Arm)
	jdk := filepath.Join(root, "runtime", "jdk-21")
	env := []string{
		"PATH=" + filepath.Join(jdk, "bin") + ":/usr/bin:/bin", "JAVA_HOME=" + jdk, "RUNTIME_JAVA_HOME=" + jdk,
		"GRADLE_USER_HOME=" + filepath.Join(state, "gradle"), "TMPDIR=" + filepath.Join(state, "tmp"),
		"XDG_CACHE_HOME=" + filepath.Join(state, "cache"), "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TZ=UTC",
		"JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8 -Duser.home=" + filepath.Join(state, "user") + " -Dmaven.repo.local=" + filepath.Join(state, "maven"),
	}
	return nativeRequest{Schema: "buildopt.eic/native-request/v1", Root: root, Runner: fileBinding{Path: runner}, Row: r, Directory: directory, Arguments: append([]string{"/usr/bin/taskset", "--cpu-list", "0-7"}, r.Arguments...), Environment: env, TimeoutSeconds: 600}, nil
}

func nativeServiceArguments(unit, runner, request, pin string) []string {
	return []string{"--user", "--quiet", "--wait", "--pipe", "--collect", "--service-type=exec", "--unit=" + unit,
		"--property=KillMode=control-group", "--property=RuntimeMaxSec=610s", "--property=TimeoutStopSec=2s", "--property=Restart=no",
		"--", runner, "native-child", request, pin, unit}
}

func validateNativeSequence(root, slot string) error {
	rows := makeProtocol().Rows
	position := -1
	for i := 0; i < 5; i++ {
		if rows[i].ID == slot {
			position = i
		}
	}
	if position < 0 {
		return errors.New("unknown native preparation/diagnostic slot")
	}
	entries, err := os.ReadDir(filepath.Join(root, "attempts"))
	if err != nil {
		return err
	}
	if len(entries) != position {
		return errors.New("native starts missing, extra or already reserved; no replay")
	}
	for i := 0; i < position; i++ {
		var r nativeResult
		if err = readJSON(filepath.Join(root, "attempts", rows[i].ID, "native-result.json"), &r); err != nil {
			return err
		}
		if r.RowID != rows[i].ID || !r.Process.Started || r.Process.Outcome != "SUCCESS" || r.Process.ExitCode != 0 || r.InputState != "VERIFIED" {
			return errors.New("previous native preparation failed or is incomplete")
		}
		if rows[i].Block == "D" {
			if err := verifyNativeDiagnosticComplete(filepath.Join(root, "attempts", rows[i].ID)); err != nil {
				return errors.New("previous diagnostic capture is incomplete")
			}
		}
	}
	return nil
}

func nativeGit(ctx context.Context, source string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/git", append([]string{"-C", source}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	raw, err := cmd.Output()
	return strings.TrimSpace(string(raw)), err
}

func verifyNativeSource(ctx context.Context, r nativeRequest) (string, error) {
	root, err := nativeGit(ctx, r.Directory, "rev-parse", "--show-toplevel")
	if err != nil || root != r.Directory {
		return "", errors.New("native source root mismatch")
	}
	common, err := nativeGit(ctx, r.Directory, "rev-parse", "--path-format=absolute", "--git-common-dir")
	expectedCommon, resolveErr := filepath.EvalSymlinks(filepath.Join(r.Root, "repos", "elasticsearch.git"))
	if err != nil || resolveErr != nil || common != expectedCommon {
		return "", errors.New("native shared Git identity mismatch")
	}
	head, err := nativeGit(ctx, r.Directory, "rev-parse", "HEAD")
	if err != nil || head != r.Row.Revision {
		return "", errors.New("native source revision drift")
	}
	status, err := nativeGit(ctx, r.Directory, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil || status != "" {
		return "", fmt.Errorf("native source changed: %s", status)
	}
	inputs := sourceInputs()
	if err = checkBinding(fileBinding{Path: filepath.Join(r.Directory, inputs.TaskPath), SHA256: inputs.TaskPreimageSHA256}); err != nil {
		return "", err
	}
	if err = checkBinding(fileBinding{Path: filepath.Join(r.Directory, "gradlew"), SHA256: "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"}); err != nil {
		return "", err
	}
	return nativeGit(ctx, r.Directory, "rev-parse", "HEAD^{tree}")
}

func nativePrepare(ctx context.Context, root, slot string) (nativeResult, error) {
	return nativePrepareWithSeed(ctx, root, slot, "")
}

func nativePrepareWithSeed(ctx context.Context, root, slot, seedPin string) (nativeResult, error) {
	var result nativeResult
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil || root != canonical {
		return result, errors.New("native campaign root must be canonical")
	}
	rows := makeProtocol().Rows
	if strings.HasPrefix(slot, "CD") {
		rows = cleanDiagnosticRows()
		err = validateCleanSequence(root, slot)
	} else {
		err = validateNativeSequence(root, slot)
	}
	if err != nil {
		return result, err
	}
	if err = nativeDiskGuard(root); err != nil {
		return result, err
	}
	runner, err := ownExecutable()
	if err != nil {
		return result, err
	}
	var row row
	for _, r := range rows {
		if r.ID == slot {
			row = r
			break
		}
	}
	r, err := resolvedNativeRequest(root, runner, row)
	if err != nil {
		return result, err
	}
	if r.Seed != nil {
		r.Seed.SHA256 = seedPin
		if err = verifyCleanSeed(root, *r.Seed); err != nil {
			return result, err
		}
	} else if seedPin != "" {
		return result, errors.New("seed pin requires clean scenario")
	}
	if r.Diagnostic != nil {
		if _, err = os.Lstat(r.Diagnostic.Path); os.IsNotExist(err) {
			err = writeNew(r.Diagnostic.Path, []byte(nativeDiagnosticScript))
		}
		if err != nil {
			return result, err
		}
		if err = checkBinding(*r.Diagnostic); err != nil {
			return result, err
		}
	}
	r.Runner.SHA256, err = hashFile(runner)
	if err != nil {
		return result, err
	}
	r.SourceTree, err = verifyNativeSource(ctx, r)
	if err != nil {
		return result, err
	}
	r.Runtime, err = inventory(filepath.Join(root, "runtime", "jdk-21"), []string{"bin", "conf", "lib", "release"})
	if err != nil {
		return result, err
	}
	release, err := os.ReadFile(filepath.Join(root, "runtime", "jdk-21", "release"))
	if err != nil || !strings.Contains(string(release), "JAVA_RUNTIME_VERSION=\"21.0.12+8-LTS\"") {
		return result, errors.New("locked Temurin runtime identity mismatch")
	}
	for _, subdir := range []string{"gradle", "tmp", "cache", "user", "maven"} {
		if err = os.MkdirAll(filepath.Join(root, "state", row.Arm, subdir), 0700); err != nil {
			return result, err
		}
	}
	r.BootID, err = bootID()
	if err != nil {
		return result, err
	}
	r.ReservedNS, err = bootNow()
	if err != nil {
		return result, err
	}
	dir := filepath.Join(root, "attempts", slot)
	if err = os.Mkdir(dir, 0700); err != nil {
		return result, err
	}
	requestPath := filepath.Join(dir, "native-request.json")
	if err = writeJSON(requestPath, r); err != nil {
		return result, err
	}
	pin, err := hashFile(requestPath)
	if err != nil {
		return result, err
	}
	unit := "buildopt-eic-" + pin[:20] + ".service"
	args := nativeServiceArguments(unit, runner, requestPath, pin)
	if err = writeJSON(filepath.Join(dir, "service-request.json"), args); err != nil {
		return result, err
	}
	raw, runErr := runNativeService(ctx, args, unit)
	if err = writeNew(filepath.Join(dir, "service.log"), raw); err != nil {
		return result, err
	}
	if err = readJSON(filepath.Join(dir, "native-result.json"), &result); err != nil {
		return result, errors.Join(runErr, err)
	}
	if runErr != nil {
		return result, fmt.Errorf("native preparation %s stopped: %w (raw evidence retained)", slot, runErr)
	}
	if row.Block == "D" || row.Block == "CD" {
		if err = retainNativeDiagnostic(root, slot); err != nil {
			return result, fmt.Errorf("native diagnostic capture: %w", err)
		}
	}
	return result, nativeDiskGuard(root)
}

func resolvedNativeRequest(root, runner string, row row) (nativeRequest, error) {
	if row.Block == "CD" {
		return makeCleanDiagnosticRequest(root, runner, row)
	}
	if row.Block == "D" {
		return makeNativeDiagnosticRequest(root, runner, row)
	}
	return makeNativeRequest(root, runner, row)
}

func runNativeService(ctx context.Context, args []string, unit string) ([]byte, error) {
	serviceCtx, cancel := context.WithTimeout(ctx, 622*time.Second)
	defer cancel()
	cmd := exec.CommandContext(serviceCtx, "/usr/bin/systemd-run", args...)
	cmd.Cancel = func() error {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = exec.CommandContext(stopCtx, "/usr/bin/systemctl", "--user", "stop", "--no-block", unit).Run()
		return cmd.Process.Kill()
	}
	return cmd.CombinedOutput()
}

func nativeChild(args []string) int {
	if len(args) != 3 {
		return 64
	}
	path, pin, unit := args[0], args[1], args[2]
	actual, err := hashFile(path)
	if err != nil || actual != pin {
		return 65
	}
	var r nativeRequest
	if err = readJSON(path, &r); err != nil {
		return 65
	}
	self, err := ownExecutable()
	if err != nil || self != r.Runner.Path || checkBinding(r.Runner) != nil {
		return 65
	}
	expected, err := resolvedNativeRequest(r.Root, self, r.Row)
	if err != nil {
		return 65
	}
	if r.Schema != expected.Schema || r.Directory != expected.Directory || !reflect.DeepEqual(r.Arguments, expected.Arguments) || !reflect.DeepEqual(r.Environment, expected.Environment) || !reflect.DeepEqual(r.Diagnostic, expected.Diagnostic) || r.TimeoutSeconds != 600 {
		return 65
	}
	if r.Diagnostic != nil && checkBinding(*r.Diagnostic) != nil {
		return 65
	}
	if (r.Seed == nil) != (expected.Seed == nil) {
		return 65
	}
	if r.Seed != nil && verifyCleanSeed(r.Root, *r.Seed) != nil {
		return 65
	}
	cgroup, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || !strings.HasSuffix(strings.TrimSpace(string(cgroup)), "/"+unit) {
		return 65
	}
	boot, err := bootID()
	if err != nil || boot != r.BootID {
		return 65
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	sourceTree, err := verifyNativeSource(ctx, r)
	if err != nil || sourceTree != r.SourceTree {
		return 65
	}
	actualRuntime, err := inventory(filepath.Join(r.Root, "runtime", "jdk-21"), []string{"bin", "conf", "lib", "release"})
	if err != nil || !reflect.DeepEqual(actualRuntime, r.Runtime) {
		return 65
	}
	syscall.Umask(022)
	if err = writeJSON(filepath.Join(filepath.Dir(path), "ownership.json"), map[string]string{"unit": unit, "cgroup": strings.TrimSpace(string(cgroup)), "invocationId": os.Getenv("INVOCATION_ID")}); err != nil {
		return 65
	}
	p, err := runConfiguredProcess(ctx, filepath.Dir(path), r.Arguments[0], r.Arguments[1:], r.Directory, r.Environment, 600*time.Second)
	result := nativeResult{Schema: "buildopt.eic/native-result/v1", RowID: r.Row.ID, Process: p, Cgroup: strings.TrimSpace(string(cgroup)), InvocationID: os.Getenv("INVOCATION_ID"), InputState: "UNVERIFIED"}
	if tree, e := verifyNativeSource(ctx, r); e == nil && tree == r.SourceTree {
		if runtime, e := inventory(filepath.Join(r.Root, "runtime", "jdk-21"), []string{"bin", "conf", "lib", "release"}); e == nil && reflect.DeepEqual(runtime, r.Runtime) {
			result.InputState = "VERIFIED"
		}
	}
	if e := writeJSON(filepath.Join(filepath.Dir(path), "native-result.json"), result); e != nil {
		return 65
	}
	if err != nil || p.Outcome != "SUCCESS" || result.InputState != "VERIFIED" {
		return 1
	}
	return 0
}

func nativeDiskGuard(root string) error {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(root, &stats); err != nil {
		return err
	}
	if stats.Bavail*uint64(stats.Bsize) < 40<<30 {
		return errors.New("native campaign requires 40 GiB free")
	}
	var size int64
	return filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		if size > 30<<30 {
			return errors.New("native campaign exceeds 30 GiB")
		}
		return nil
	})
}
