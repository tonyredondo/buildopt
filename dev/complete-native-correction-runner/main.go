// Command complete-native-correction-runner owns bounded native CNC capture.
// Public execution is an explicit later operation; tests use fake children.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return errors.New("CNC capture requires Linux amd64")
	}
	if len(arguments) == 0 {
		return errors.New("mode required: freeze, package-check, init, guard, preflight, seed, stabilize, review, capture, check")
	}
	mode := arguments[0]
	flags := flag.NewFlagSet(mode, flag.ContinueOnError)
	repo := flags.String("repo", "", "absolute BuildOpt root")
	root := flags.String("state", "", "absolute campaign state root")
	packagePath := flags.String("package", "", "capture package file")
	slot := flags.String("slot", "", "native slot P01/P02/D01/D02/M01/M02")
	source := flags.String("source", "", "registered source worktree")
	common := flags.String("common-git-dir", "", "known shared Git directory")
	home := flags.String("gradle-home", "", "private Gradle home")
	jdk25 := flags.String("jdk25", "", "verified Corretto 25 root")
	archive25 := flags.String("jdk25-archive", "", "pinned Corretto 25 archive")
	jdk21 := flags.String("jdk21", "", "verified Corretto 21 root")
	archive21 := flags.String("jdk21-archive", "", "pinned Corretto 21 archive")
	acknowledge := flags.Bool("acknowledge-phase-a", false, "explicit operator start; does not grant agent authority")
	decision := flags.String("decision", "", "review decision: continue or stop")
	note := flags.String("note", "", "review progress and remaining proof")
	if err := flags.Parse(arguments[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if !filepath.IsAbs(*repo) {
		return errors.New("absolute --repo required")
	}
	switch mode {
	case "freeze", "package-check", "init", "guard", "guard-child", "preflight", "seed", "stabilize", "review", "capture", "check":
	default:
		return errors.New("unknown CNC mode")
	}
	if mode == "freeze" {
		manifest, err := makePackage(*repo)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(*packagePath) {
			return errors.New("new absolute --package required")
		}
		return writeNewJSON(*packagePath, manifest)
	}
	digest, err := verifyPackage(*repo, *packagePath)
	if err != nil {
		return err
	}
	if mode == "package-check" {
		fmt.Println(digest)
		return nil
	}
	if mode == "init" {
		if !*acknowledge {
			return errors.New("explicit execution acknowledgement required")
		}
		now, err := bootClock()
		if err != nil {
			return err
		}
		return initialize(*root, digest, now)
	}
	if !filepath.IsAbs(*root) {
		return errors.New("absolute --state required")
	}
	var state campaign
	if err = readJSON(filepath.Join(*root, "state.json"), &state); err != nil {
		return err
	}
	if state.PackageSHA256 != digest {
		return errors.New("state/package binding drift")
	}
	if mode == "check" {
		results, err := inspect(*root, state)
		if err != nil {
			return err
		}
		if err := inspectNativeRequests(*repo, *root, results); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(results)
	}
	if mode == "guard-child" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		return guardian(ctx, *root, state)
	}
	locked, err := lockCampaign(*root)
	if err != nil {
		return err
	}
	defer locked.Close()
	now, err := bootClock()
	if err != nil {
		return err
	}
	left, err := remaining(state, now)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, deadlineCancel := context.WithTimeout(ctx, time.Duration(left*float64(time.Second)))
	defer deadlineCancel()
	if mode == "guard" {
		if !*acknowledge {
			return errors.New("guardian requires execution acknowledgement")
		}
		return startGuardian(ctx, *repo, *root, *packagePath, state)
	}
	subjects, contract, err := loadInputs(*repo)
	if err != nil {
		return err
	}
	if _, err = inspect(*root, state); err != nil {
		return err
	}
	if mode == "review" {
		if now.Seconds < state.ReviewAt || (*decision != "continue" && *decision != "stop") || strings.TrimSpace(*note) == "" {
			return errors.New("review requires elapsed checkpoint, decision and progress note")
		}
		return writeNewJSON(filepath.Join(*root, "review.json"), reviewEvidence{now.Seconds, *decision, *note})
	}
	if err = reviewDue(*root, state, now); err != nil {
		return err
	}
	if mode == "seed" || mode == "stabilize" {
		if !*acknowledge {
			return errors.New("execution acknowledgement required")
		}
		previous, err := inspect(*root, state)
		if err != nil {
			return err
		}
		if len(previous) < 2 || previous[0].Outcome != "CHILD_SUCCESS" || previous[1].Outcome != "CHILD_SUCCESS" {
			return errors.New("successful P01/P02 required")
		}
		if mode == "seed" {
			return seedDependencies(*root)
		}
		return stabilize(ctx, *root, state)
	}
	preflight := func() error {
		for _, path := range []string{*source, *home, *jdk25, *jdk21, *archive25, *archive21} {
			if err := contained(*root, path); err != nil {
				return err
			}
		}
		for _, key := range []string{"CI", "RELEASE_VERSION", "GRADLE_OPTS", "JAVA_OPTS", "JAVA_TOOL_OPTIONS", "_JAVA_OPTIONS", "JDK_JAVA_OPTIONS"} {
			if _, exists := os.LookupEnv(key); exists {
				return fmt.Errorf("undeclared environment input: %s", key)
			}
		}
		used, free, err := diskUsage(*root)
		if err != nil {
			return err
		}
		if used > 20<<30 || free < 10<<30 {
			return errors.New("disk budget unavailable")
		}
		if err = verifyWorktree(ctx, *source, *common, subjects.Subjects[0]); err != nil {
			return err
		}
		if err = verifyRuntime(*archive25, *jdk25, subjects.Runtimes[0]); err != nil {
			return err
		}
		if err = verifyRuntime(*archive21, *jdk21, subjects.Runtimes[1]); err != nil {
			return err
		}
		return verifyHost(ctx)
	}
	if mode == "preflight" {
		return preflight()
	}
	if !*acknowledge {
		return errors.New("capture needs explicit execution acknowledgement")
	}
	if !strings.HasPrefix(*slot, "P") && !strings.HasPrefix(*slot, "D") && !strings.HasPrefix(*slot, "M") {
		return errors.New("candidate/fixture/value modes require later recipe gates")
	}
	if !validSlot(*slot) {
		return errors.New("unknown native slot")
	}
	expectedSource, expectedHome := expectedPaths(*root, *slot)
	if *source != expectedSource || *home != expectedHome {
		return errors.New("slot worktree/home identity mismatch")
	}
	if err = verifyCommittedPackage(ctx, *repo); err != nil {
		return err
	}
	if strings.HasPrefix(*slot, "M") {
		var environment environmentEvidence
		if err = readJSON(filepath.Join(*root, "environment.json"), &environment); err != nil {
			return err
		}
		if err = checkEnvironment(state, environment); err != nil {
			return err
		}
	}
	previous, err := inspect(*root, state)
	if err != nil {
		return err
	}
	if len(previous) >= 6 || contract.ExecutionOrder[len(previous)] != *slot {
		return errors.New("slot is out of order or already spent")
	}
	for index, result := range previous {
		if result.Slot != contract.ExecutionOrder[index] {
			return errors.New("noncontiguous attempt sequence")
		}
		if result.Outcome != "CHILD_SUCCESS" && result.Outcome != "ROOT_REPORT_CAPTURED" {
			return errors.New("prior attempt did not satisfy its capture prerequisite")
		}
	}
	var owner ownership
	if err := readJSON(filepath.Join(*root, "ownership.json"), &owner); err != nil {
		return err
	}
	requested, err := nativeRequest(nativePaths{*repo, *root, *source, *home, *jdk25, *jdk21}, *slot, contract, subjects.Subjects[0].Outputs, owner)
	if err != nil {
		return err
	}
	prepared := false
	checkInputs := func() error {
		if err := preflight(); err != nil {
			return err
		}
		if !prepared {
			if err := prepareHome(*root, *home, strings.HasPrefix(*slot, "P"), *slot == "M02"); err != nil {
				return err
			}
			if err := prepareUserHome(*home, *slot == "M02"); err != nil {
				return err
			}
			if err := prepareNativeSource(ctx, *root, *source, *slot, subjects.Subjects[0].Outputs); err != nil {
				return err
			}
			if strings.HasPrefix(*slot, "D") {
				if entries, err := os.ReadDir(requested.ReportRoot); err == nil && len(entries) > 0 {
					return errors.New("strict report root is not fresh")
				}
			}
			prepared = true
			return recordNativeInputs(filepath.Join(*root, "attempts", *slot, "inputs-before.json"), subjects, requested.Native)
		}
		if err := prepareUserHome(*home, true); err != nil {
			return err
		}
		return recordNativeInputs(filepath.Join(*root, "attempts", *slot, "inputs-after.json"), subjects, requested.Native)
	}
	guard := func() error {
		used, free, err := diskUsage(*root)
		if err != nil {
			return err
		}
		if used > 20<<30 || free < 10<<30 {
			return errors.New("disk limit reached during child")
		}
		return nil
	}
	terminal, err := capture(ctx, *root, state, requested, bootClock, checkInputs, guard)
	if err != nil {
		return err
	}
	if err = json.NewEncoder(os.Stdout).Encode(terminal); err != nil {
		return err
	}
	if terminal.Outcome != "CHILD_SUCCESS" && terminal.Outcome != "ROOT_REPORT_CAPTURED" {
		return errors.New("attempt retained without advancement")
	}
	return nil
}
