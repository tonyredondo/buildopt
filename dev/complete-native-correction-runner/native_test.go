package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestNativeConsumer(t *testing.T) {
	separator := -1
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		return
	}
	args := os.Args[separator+1:]
	if len(args) == 0 || args[0] != "assemble" {
		t.Fatal("not the native request")
	}
	if !strings.Contains(os.Getenv("JAVA_TOOL_OPTIONS"), "-Dmaven.repo.local="+filepath.Join(os.Getenv("HOME"), ".m2/repository")) {
		t.Fatal("private Maven input lost in child")
	}
	// Real Groovy/JVM preferences and Kotlin daemon discovery create state under
	// the fresh private home. Exercise its post-child validation in every slot.
	for relative, body := range map[string]string{
		".java/.userPrefs/org/codehaus/groovy/prefs.xml": "<preferences><root type=\"user\"><map/></root></preferences>\n",
		".kotlin/daemon/kotlin-daemon.fixture.run":       "",
	} {
		path := filepath.Join(os.Getenv("HOME"), relative)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Kotlin also leaves an empty project-local sessions directory, which Git
	// cannot track and which need not be covered by the owner's ignore rules.
	if err := os.MkdirAll(".kotlin/sessions", 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(data))
	if contains(args, "--configuration-cache") {
		report, err := filepath.Abs("build/reports/configuration-cache/fixture/configuration-cache-report.html")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.MkdirAll(filepath.Dir(report), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(report, []byte("<html>fake-child strict evidence</html>"), 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Println("See the complete report at file://" + report)
		os.Exit(1)
	}
	graphPath := os.Getenv("BUILDOPT_TASK_GRAPH_OUTPUT")
	if graphPath == "" {
		return
	}
	tasks := []map[string]any{}
	for producer, path := range map[string]string{":buildFinalJar": "build/libs/final.jar", ":jcstressJar": "build/libs/stress.jar", ":performance-results-page:jar": "performance-results-page/build/libs/page.jar", ":jar": "build/intermediates/plain-jar/plain.jar", ":shadowJar": "build/intermediates/shadow-jar/shadow.jar"} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("native "+producer), 0600); err != nil {
			t.Fatal(err)
		}
		tasks = append(tasks, map[string]any{"identity": producer, "path": producer, "taskClass": "FixtureTask", "dependencies": []string{}, "outputs": []string{path}})
	}
	graph, err := json.Marshal(map[string]any{"schemaVersion": "buildopt.diagnostics/gradle-task-graph/v1", "buildPath": ":", "tasks": tasks})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(graphPath, append(graph, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(graphPath), "operations-log.txt"), []byte("fake-child operations\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func testNativeConsumerMatrix(t *testing.T, repo, root string, state campaign, owner ownership) {
	t.Helper()
	subjects, contract, err := loadInputs(repo)
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"native-a", "native-b"} {
		source := filepath.Join(root, "worktrees", role)
		if err := os.MkdirAll(source, 0700); err != nil {
			t.Fatal(err)
		}
		fixtureGit(t, source, "init", "--quiet")
		if err := os.WriteFile(filepath.Join(source, ".gitignore"), []byte("/build/\n/.gradle/\n/performance-results-page/build/\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(source, "gradlew"), []byte("#!/bin/sh\nexec "+strconv.Quote(binary)+" -test.run=^TestNativeConsumer$ -- \"$@\"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, slot := range contract.ExecutionOrder[:6] {
		source, home := expectedPaths(root, slot)
		if err := os.MkdirAll(home, 0700); err != nil {
			t.Fatal(err)
		}
		if err := prepareUserHome(home, slot == "M02"); err != nil {
			t.Fatal(err)
		}
		r, err := nativeRequest(nativePaths{repo, root, source, home, filepath.Join(root, "jdk25"), filepath.Join(root, "jdk21")}, slot, contract, subjects.Subjects[0].Outputs, owner)
		if err != nil {
			t.Fatal(err)
		}
		prepared := false
		got, err := capture(context.Background(), root, state, r, bootClock, func() error {
			name := "inputs-after.json"
			if !prepared {
				if err := prepareNativeSource(context.Background(), root, source, slot, subjects.Subjects[0].Outputs); err != nil {
					return err
				}
				prepared = true
				name = "inputs-before.json"
			}
			if err := prepareUserHome(home, true); err != nil {
				return err
			}
			// Synthetic bindings exercise reconstruction, not public runtime proof.
			return recordNativeInputs(filepath.Join(root, "attempts", slot, name), subjects, r.Native)
		})
		want := "CHILD_SUCCESS"
		if slot[0] == 'D' {
			want = "ROOT_REPORT_CAPTURED"
		}
		if err != nil || got.Outcome != want || !got.Started {
			t.Fatalf("%s: %+v %v", slot, got, err)
		}
		rows, err := inspect(root, state)
		if err != nil {
			t.Fatal(err)
		}
		if err := inspectNativeRequests(repo, root, rows); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativeCommandProfilesAndPrivateInputs(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	subjects, contract, err := loadInputs(repo)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, slot := range []string{"P01", "P02", "D01", "D02", "M01", "M02"} {
		source, home := expectedPaths(root, slot)
		r, err := nativeRequest(nativePaths{repo, root, source, home, filepath.Join(root, "jdk25"), filepath.Join(root, "jdk21")}, slot, contract, subjects.Subjects[0].Outputs, ownership{})
		if err != nil {
			t.Fatal(err)
		}
		profile := map[byte]string{'P': "prefetch", 'D': "strict", 'M': "native"}[slot[0]]
		if !reflect.DeepEqual(r.Arguments[4:4+len(contract.Commands[profile])], contract.Commands[profile]) {
			t.Fatal("changed native profile")
		}
		args := strings.Join(r.Arguments, "\n")
		env := strings.Join(r.Environment, "\n")
		if strings.Contains(args, "--offline") != (slot[0] != 'P') || strings.Contains(args, "--init-script") != (slot[0] == 'M') {
			t.Fatal("asymmetric/off-profile instrumentation or network")
		}
		if !strings.Contains(env, "HOME="+filepath.Join(home, "user-home")) || !strings.Contains(env, "-Duser.home="+filepath.Join(home, "user-home")) || !strings.Contains(env, "-Dmaven.repo.local="+filepath.Join(home, "user-home/.m2/repository")) {
			t.Fatal("ambient Maven/home input")
		}
		for _, injected := range []string{"RELEASE_VERSION=", "CI=", "M2_HOME=", "GRADLE_OPTS=", "JAVA_OPTS="} {
			if strings.Contains(env, injected) {
				t.Fatal("undeclared environment injection")
			}
		}
	}
	if err := os.Mkdir(filepath.Join(root, "private"), 0700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "private")
	if err := prepareUserHome(home, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "user-home/.m2/settings.xml"), []byte("ambient"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareUserHome(home, true); err == nil {
		t.Fatal("Maven settings drift accepted")
	}
}

func TestPrivateHomeRuntimeState(t *testing.T) {
	for _, change := range []string{"generated", "maven-settings", "maven-artifact", "gradle-settings", "extra-java", "extra-kotlin", "prefix-lookalike", "root-file", "generated-root-file", "ancestor-link", "leaf-link", "special-member"} {
		t.Run(change, func(t *testing.T) {
			home := t.TempDir()
			if err := prepareUserHome(home, false); err != nil {
				t.Fatal(err)
			}
			user := filepath.Join(home, "user-home")
			write := func(relative string) {
				t.Helper()
				path := filepath.Join(user, relative)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("fixture state"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "generated":
				write(".java/.userPrefs/org/codehaus/groovy/prefs.xml")
				write(".java/.userPrefs/.user.lock.fixture")
				write(".kotlin/daemon/kotlin-daemon.fixture.run")
			case "maven-settings":
				write(".m2/settings.xml")
			case "maven-artifact":
				write(".m2/repository/injected.jar")
			case "gradle-settings":
				write(".gradle/gradle.properties")
			case "extra-java":
				write(".java/other/settings")
			case "extra-kotlin":
				write(".kotlin/other/settings")
			case "prefix-lookalike":
				write(".java/.userPrefs-other/settings")
			case "root-file":
				write(".kotlin")
			case "generated-root-file":
				write(".java/.userPrefs")
			case "special-member":
				path := filepath.Join(user, ".kotlin/daemon/pipe")
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "ancestor-link", "leaf-link":
				path := filepath.Join(user, ".java")
				if change == "leaf-link" {
					path = filepath.Join(user, ".java/.userPrefs/link")
					if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(t.TempDir(), path); err != nil {
					t.Fatal(err)
				}
			}
			if err := prepareUserHome(home, true); (err == nil) != (change == "generated") {
				t.Fatalf("private runtime state: %v", err)
			}
			if err := prepareUserHome(home, false); err == nil {
				t.Fatal("existing private home silently reused as fresh")
			}
		})
	}
}

func TestRetainedPrivateRuntimeHome(t *testing.T) {
	home := os.Getenv("CNC_PRIVATE_RUNTIME_HOME")
	if home == "" {
		t.Skip("requires explicit retained native Gradle home; ordinary fixtures are not real-home proof")
	}
	if err := prepareUserHome(home, true); err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("CNC_RETAINED_CAMPAIGN_ROOT")
	if root == "" {
		return
	}
	if _, expectedHome := expectedPaths(root, "P01"); home != expectedHome {
		t.Fatal("retained private home does not belong to the original P01")
	}
	var state campaign
	if err := readJSON(filepath.Join(root, "state.json"), &state); err != nil {
		t.Fatal(err)
	}
	rows, err := inspect(root, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Slot != "P01" || !rows[0].Started || rows[0].ExitCode != 0 || rows[0].Outcome != "HARNESS_FAILURE" {
		t.Fatal("the correction must not upgrade the retained failed attempt")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := inspectNativeRequests(repo, root, rows); err != nil {
		t.Fatal(err)
	}
}

func TestNativeVerifiedInputReconstruction(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	subjects, _, err := loadInputs(repo)
	if err != nil {
		t.Fatal(err)
	}
	paths := &nativePaths{Repo: repo, Root: t.TempDir()}
	for _, mode := range []string{"valid", "missing-before", "missing-after", "changed-source", "changed-runtime", "changed-path", "timeout-without-after"} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			observed := processEvidence{Started: true, Outcome: "CHILD_SUCCESS"}
			for _, name := range []string{"inputs-before.json", "inputs-after.json"} {
				if mode == "missing-before" && name == "inputs-before.json" || (mode == "missing-after" || mode == "timeout-without-after") && name == "inputs-after.json" {
					continue
				}
				value := nativeInputEvidence{subjects.Subjects[0], append([]runtimeBinding{}, subjects.Runtimes...), paths}
				switch mode {
				case "changed-source":
					value.Source.Revision = "changed"
				case "changed-runtime":
					value.Runtimes[0].SHA256 = "changed"
				case "changed-path":
					value.Paths = &nativePaths{Repo: repo, Root: "changed"}
				}
				if err := writeNewJSON(filepath.Join(directory, name), value); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "timeout-without-after" {
				observed.Outcome = "TIME_LIMIT"
			}
			err := checkNativeInputs(directory, subjects, paths, observed)
			valid := mode == "valid" || mode == "timeout-without-after"
			if (err == nil) != valid {
				t.Fatalf("unexpected reconstruction result: %v", err)
			}
		})
	}
}

func TestNativeEmptyUnignoredGeneratedState(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(filepath.Join(source, ".kotlin/sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, source, "init", "--quiet")
	policy := outputPolicy{Selectors: []string{"build/libs/*.jar"}}
	if err := prepareNativeSource(context.Background(), root, source, "P01", policy); err == nil {
		t.Fatal("fresh preparation admitted preexisting generated directories")
	}
	if err := prepareNativeSource(context.Background(), root, source, "D01", policy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(source, ".kotlin")); !os.IsNotExist(err) {
		t.Fatal("empty generated state was not archived")
	}
	if info, err := os.Stat(filepath.Join(root, "attempts/D01/prior-state/.kotlin/sessions")); err != nil || !info.IsDir() {
		t.Fatal("empty directory structure was not preserved")
	}
}

func TestGeneratedSourceDirectoryBoundaries(t *testing.T) {
	for _, mode := range []string{"empty", "nested-empty", "ignored-file", "unignored-file", "zero-byte-file", "nested-file", "tracked-file", "symlink", "root-symlink", "fifo", "root-file"} {
		t.Run(mode, func(t *testing.T) {
			source := t.TempDir()
			fixtureGit(t, source, "init", "--quiet")
			directory := filepath.Join(source, ".kotlin")
			if mode == "root-file" {
				if err := os.WriteFile(directory, nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else if mode == "root-symlink" {
				if err := os.Symlink(t.TempDir(), directory); err != nil {
					t.Fatal(err)
				}
			} else if err := os.MkdirAll(filepath.Join(directory, "sessions"), 0700); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(directory, "owner.txt")
			switch mode {
			case "nested-empty":
				if err := os.MkdirAll(filepath.Join(directory, "sessions/empty/deeper"), 0700); err != nil {
					t.Fatal(err)
				}
			case "ignored-file", "unignored-file", "zero-byte-file", "nested-file", "tracked-file":
				body := []byte("preserve these bytes")
				if mode == "zero-byte-file" {
					body = nil
				}
				if mode == "nested-file" {
					file = filepath.Join(directory, "sessions/.hidden")
				}
				if err := os.WriteFile(file, body, 0600); err != nil {
					t.Fatal(err)
				}
				if mode == "ignored-file" {
					if err := os.WriteFile(filepath.Join(source, ".gitignore"), []byte("/.kotlin/\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "tracked-file" {
					fixtureGit(t, source, "add", ".kotlin/owner.txt")
				}
			case "symlink":
				if err := os.Symlink(t.TempDir(), filepath.Join(directory, "sessions/link")); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := syscall.Mkfifo(filepath.Join(directory, "sessions/pipe"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := fixtureGit(t, source, "status", "--porcelain=v1", "--ignored")
			err := verifyGeneratedSourceDirectory(context.Background(), source, ".kotlin")
			wantSuccess := mode == "empty" || mode == "nested-empty" || mode == "ignored-file"
			if (err == nil) != wantSuccess {
				t.Fatalf("validation: %v", err)
			}
			if after := fixtureGit(t, source, "status", "--porcelain=v1", "--ignored"); after != before {
				t.Fatal("read-only validation changed source state")
			}
			if _, err := os.Lstat(directory); err != nil {
				t.Fatal("validation removed the generated root")
			}
		})
	}
}

func TestRetainedEmptyGeneratedSourceState(t *testing.T) {
	root := os.Getenv("CNC_EMPTY_GENERATED_CAMPAIGN_ROOT")
	if root == "" {
		t.Skip("requires the retained real D01 pre-start refusal; fixtures are not real-source proof")
	}
	source, _ := expectedPaths(root, "P01")
	if err := verifyGeneratedSourceDirectory(context.Background(), source, ".kotlin"); err != nil {
		t.Fatal(err)
	}
	var state campaign
	if err := readJSON(filepath.Join(root, "state.json"), &state); err != nil {
		t.Fatal(err)
	}
	rows, err := inspect(root, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Slot != "P01" || rows[0].Outcome != "CHILD_SUCCESS" || rows[1].Slot != "P02" || rows[1].Outcome != "CHILD_SUCCESS" || rows[2].Slot != "D01" || rows[2].Started || rows[2].Outcome != "PRE_START_FAILURE" {
		t.Fatal("the repair must not upgrade or replace the retained three-row history")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := inspectNativeRequests(repo, root, rows); err != nil {
		t.Fatal(err)
	}
}

func TestNativeStateArchivePreservesSourceAndReports(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, source, "init", "--quiet")
	if err := os.WriteFile(filepath.Join(source, ".gitignore"), []byte("/build/\n/.gradle/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "source.txt"), []byte("owner source"), 0600); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, source, "add", "source.txt", ".gitignore")
	policy := outputPolicy{Selectors: []string{"build/libs/*.jar"}}
	if err := prepareNativeSource(context.Background(), root, source, "P01", policy); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"build/libs/native.jar", ".gradle/history.bin", "build/reports/configuration-cache/key/configuration-cache-report.html"} {
		target := filepath.Join(source, path)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(path), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := prepareNativeSource(context.Background(), root, source, "P01", policy); err == nil {
		t.Fatal("warm initial worktree accepted")
	}
	if err := prepareNativeSource(context.Background(), root, source, "M01", policy); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"build/libs/native.jar", ".gradle/history.bin"} {
		if _, err := os.Stat(filepath.Join(source, path)); !os.IsNotExist(err) {
			t.Fatal("stale output/history survived")
		}
		if _, err := fileDigest(filepath.Join(root, "attempts/M01/prior-state", path)); err != nil {
			t.Fatal("generated evidence lost")
		}
	}
	if _, err := fileDigest(filepath.Join(source, "build/reports/configuration-cache/key/configuration-cache-report.html")); err != nil {
		t.Fatal("strict report relocated")
	}
	if value, err := os.ReadFile(filepath.Join(source, "source.txt")); err != nil || string(value) != "owner source" {
		t.Fatal("owner source changed")
	}
	if diff := fixtureGit(t, source, "diff", "--name-only"); diff != "" {
		t.Fatal("tracked source changed")
	}
}
