package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type nativePaths struct{ Repo, Root, Source, Home, JDK25, JDK21 string }

type nativeInputEvidence struct {
	Source   sourceBinding    `json:"source"`
	Runtimes []runtimeBinding `json:"runtimes"`
	Paths    *nativePaths     `json:"paths"`
}

func recordNativeInputs(path string, subjects subjectsDocument, paths *nativePaths) error {
	return writeNewJSON(path, nativeInputEvidence{subjects.Subjects[0], subjects.Runtimes, paths})
}

func checkNativeInputs(directory string, subjects subjectsDocument, paths *nativePaths, observed processEvidence) error {
	expected := nativeInputEvidence{subjects.Subjects[0], subjects.Runtimes, paths}
	for _, name := range []string{"inputs-before.json", "inputs-after.json"} {
		var actual nativeInputEvidence
		err := readJSON(filepath.Join(directory, name), &actual)
		required := observed.Started && (name == "inputs-before.json" || observed.Outcome == "CHILD_SUCCESS" || observed.Outcome == "CHILD_FAILURE")
		if os.IsNotExist(err) && !required {
			continue
		}
		if err != nil || !reflect.DeepEqual(actual, expected) {
			return fmt.Errorf("native verified-input binding missing or changed: %s", name)
		}
	}
	return nil
}

func nativeRequest(paths nativePaths, slot string, contract protocol, outputs outputPolicy, owner ownership) (request, error) {
	for _, path := range []string{paths.Repo, paths.Root, paths.Source, paths.Home, paths.JDK25, paths.JDK21} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, "\x00\r\n\"'\\") {
			return request{}, errors.New("unsupported native input path")
		}
	}
	profile := ""
	for _, row := range contract.Rows {
		if row.ID == slot {
			profile = row.Command
		}
	}
	if !validSlot(slot) || !strings.Contains("PDM", slot[:1]) || profile == "" {
		return request{}, errors.New("unknown native request")
	}
	args := append([]string{}, contract.Commands[profile]...)
	args = append(args, "--gradle-user-home", paths.Home, "-Dorg.gradle.java.installations.paths="+paths.JDK25+","+paths.JDK21, "-Dorg.gradle.java.installations.auto-detect=false", "-Dorg.gradle.java.installations.auto-download=false")
	if slot[0] != 'P' {
		args = append(args, "--offline")
	}
	user := filepath.Join(paths.Home, "user-home")
	environment := []string{"PATH=" + filepath.Join(paths.JDK25, "bin") + ":/usr/bin:/bin", "JAVA_HOME=" + paths.JDK25,
		"HOME=" + user, "TZ=UTC", "LC_ALL=C.UTF-8", "GRADLE_USER_HOME=" + paths.Home,
		"JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8 \"-Duser.home=" + user + "\" \"-Dmaven.repo.local=" + filepath.Join(user, ".m2/repository") + "\""}
	attempt := filepath.Join(paths.Root, "attempts", slot)
	required := []string{}
	var inventory *outputPolicy
	if slot[0] == 'M' {
		args = append(args, "--init-script", filepath.Join(paths.Repo, "dev/complete-native-correction.init.gradle"), "-Dorg.gradle.internal.operations.trace="+filepath.Join(attempt, "operations"))
		environment = append(environment, "BUILDOPT_TASK_GRAPH_OUTPUT="+filepath.Join(attempt, "task-graph.jsonl"))
		required = []string{"operations-log.txt", "task-graph.jsonl", "output-inventory.json"}
		inventory = &outputs
	}
	command := append([]string{"/usr/bin/taskset", "--cpu-list", "0-3", filepath.Join(paths.Source, "gradlew")}, args...)
	return request{Slot: slot, Directory: paths.Source, Arguments: command, Environment: environment, ReportRoot: filepath.Join(paths.Source, "build/reports/configuration-cache"), RequiredArtifacts: required, TimeoutSeconds: 1200, Ownership: &owner, Outputs: inventory, Native: &paths}, nil
}

func inspectNativeRequests(repo, root string, results []result) error {
	subjects, contract, err := loadInputs(repo)
	if err != nil {
		return err
	}
	for index, row := range results {
		if index >= 6 || row.Slot != contract.ExecutionOrder[index] {
			return errors.New("native row sequence drift")
		}
		var actual request
		if err := readJSON(filepath.Join(root, "attempts", row.Slot, "request.json"), &actual); err != nil {
			return err
		}
		if actual.Native == nil || actual.Ownership == nil {
			return errors.New("native request binding missing")
		}
		var owner ownership
		if err := readJSON(filepath.Join(root, "ownership.json"), &owner); err != nil || owner != *actual.Ownership {
			return errors.New("native ownership record drift")
		}
		source, home := expectedPaths(root, row.Slot)
		if actual.Native.Repo != repo || actual.Native.Root != root || actual.Native.Source != source || actual.Native.Home != home {
			return errors.New("native request roots drift")
		}
		for _, path := range []string{source, home, actual.Native.JDK25, actual.Native.JDK21} {
			if err := contained(root, path); err != nil {
				return err
			}
		}
		expected, err := nativeRequest(*actual.Native, row.Slot, contract, subjects.Subjects[0].Outputs, *actual.Ownership)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(expected, actual) {
			return errors.New("native request disagrees with frozen command profile")
		}
		directory := filepath.Join(root, "attempts", row.Slot)
		var observed processEvidence
		if err := readJSON(filepath.Join(directory, "process.json"), &observed); err != nil {
			return err
		}
		if err := checkNativeInputs(directory, subjects, actual.Native, observed); err != nil {
			return err
		}
	}
	return nil
}

func prepareUserHome(home string, reuse bool) error {
	user := filepath.Join(home, "user-home")
	if !reuse {
		if err := os.Mkdir(user, 0700); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(user, ".m2/repository"), 0700); err != nil {
			return err
		}
	}
	// Native assemble must not publish to the private Maven repository or import
	// user settings. Extra files are a changed execution input, not a cache seed.
	return filepath.WalkDir(user, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return errors.New("undeclared private user-home input")
		}
		rel, err := filepath.Rel(user, path)
		if err != nil {
			return err
		}
		if rel != "." && rel != ".m2" && rel != filepath.Join(".m2", "repository") {
			return errors.New("unexpected private user-home directory")
		}
		return nil
	})
}

// Move only generated state of these exact frozen native worktrees. Never
// reset Git, remove the worktree, or discard files not proved ignored/untracked.
// Archived reports/outputs remain available for independent earlier-row checks.
func prepareNativeSource(ctx context.Context, root, source, slot string, outputs outputPolicy) error {
	if slot == "M02" {
		return nil
	}
	directories := map[string]bool{".gradle": true, ".kotlin": true}
	for _, selector := range outputs.Selectors {
		prefix, _, ok := strings.Cut(selector, "build/")
		if !ok || (prefix != "" && !filepath.IsLocal(prefix)) {
			return errors.New("unknown output build root")
		}
		directories[filepath.Join(prefix, "build")] = true
	}
	for relative := range directories {
		path := filepath.Join(source, relative)
		if err := contained(source, path); err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("generated root is not a directory")
		}
		if slot[0] == 'P' {
			return errors.New("prefetch source has preexisting build/project-cache state")
		}
		tracked, err := gitOutput(ctx, source, "ls-files", "--", relative)
		if err != nil || tracked != "" {
			return errors.New("refusing to move tracked source state")
		}
		if _, err = gitOutput(ctx, source, "check-ignore", "--", relative); err != nil {
			return errors.New("generated state is not declared ignored")
		}
		err = filepath.WalkDir(path, func(current string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
				return errors.New("unsafe generated member")
			}
			return nil
		})
		if err != nil {
			return err
		}
		destination := filepath.Join(root, "attempts", slot, "prior-state", relative)
		if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			return err
		}
		if _, err = os.Lstat(destination); !os.IsNotExist(err) {
			return errors.New("occupied archived state")
		}
		if err = os.Rename(path, destination); err != nil {
			return fmt.Errorf("archive generated state: %w", err)
		}
		if relative == "build" {
			// Keep strict evidence at its original log-owned path. Only reports are
			// restored; build outputs and execution/configuration state stay archived.
			report := filepath.Join(destination, "reports/configuration-cache")
			if _, err = os.Stat(report); err == nil {
				original := filepath.Join(path, "reports/configuration-cache")
				if err = os.MkdirAll(filepath.Dir(original), 0700); err != nil {
					return err
				}
				if err = os.Rename(report, original); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}
