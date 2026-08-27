// Command sticky-trial-benchmark runs the small, explicit paired trial used
// by SWL-010. It is intended for trusted CI fixtures, never for a developer's
// ordinary build. Each pair receives two fresh project copies and two direct
// command invocations; the resulting report keeps the raw timings and budget
// accounting for later evaluation.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickytrial"
)

const usage = "usage: sticky-trial-benchmark --candidate PATH --template PATH --gradle PATH --output PATH [--pairs N] [--natural-runner-seconds N] [--estimate-seconds N]"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("sticky-trial-benchmark", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	candidatePath := flags.String("candidate", "", "installed BuildOpt executable")
	templatePath := flags.String("template", "", "Gradle project template")
	gradlePath := flags.String("gradle", "", "repository-local Gradle executable")
	gradleVersion := flags.String("gradle-version", "UNAVAILABLE", "Gradle version label")
	outputPath := flags.String("output", "", "report JSON path")
	pairs := flags.Int("pairs", 4, "number of alternating pairs")
	naturalSeconds := flags.Int64("natural-runner-seconds", 600, "declared eligible natural runner window")
	estimateSeconds := flags.Int64("estimate-seconds", 10, "reserved upper bound for each pair")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, usage)
		return 64
	}
	if *candidatePath == "" || *templatePath == "" || *gradlePath == "" || *outputPath == "" || *pairs < 1 || *naturalSeconds < 1 || *estimateSeconds < 1 {
		fmt.Fprintln(os.Stderr, usage)
		return 64
	}
	if os.Getenv("CI") != "true" {
		fmt.Fprintln(os.Stderr, "sticky-trial-benchmark: trusted CI is required")
		return 1
	}
	if !isRegularExecutable(*candidatePath) || !isRegular(*gradlePath) || !isDirectory(*templatePath) {
		fmt.Fprintln(os.Stderr, "sticky-trial-benchmark: candidate, Gradle or template is unavailable")
		return 1
	}

	scheduler, err := stickytrial.NewScheduler(stickytrial.Budget{
		NaturalRunnerNs:  *naturalSeconds * int64(time.Second),
		MaxExtraPermille: 50,
		MaxConcurrent:    1,
		TrustedCI:        true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: create scheduler: %v\n", err)
		return 1
	}
	runRoot, err := os.MkdirTemp("", "buildopt-sticky-trial-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: create run root: %v\n", err)
		return 1
	}
	defer os.RemoveAll(runRoot)

	trials := make([]stickytrial.PairedTrial, 0, *pairs)
	for pair := 0; pair < *pairs; pair++ {
		assignment, assignErr := scheduler.Assign(
			fmt.Sprintf("sticky-trial-%02d", pair+1),
			*estimateSeconds*int64(time.Second),
		)
		if assignErr != nil {
			fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: assign pair %d: %v\n", pair+1, assignErr)
			return 1
		}
		pairRoot := filepath.Join(runRoot, fmt.Sprintf("pair-%02d", pair+1))
		isolation := makeIsolation(pairRoot)
		if err := prepareIsolation(*templatePath, isolation); err != nil {
			_, _ = scheduler.Cancel(assignment, 0)
			fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: prepare pair %d: %v\n", pair+1, err)
			return 1
		}
		candidate := stickytrial.Command{
			Program: *candidatePath,
			Args:    []string{"gradle", "--no-daemon", "--offline", "--build-cache", "clean", "compileJava"},
			Dir:     isolation.CandidateDir,
			Env:     explicitEnvironment(isolation.CandidateGradleHome, isolation.CandidateCache, isolation.CandidateState, true),
		}
		native := stickytrial.Command{
			Program: *gradlePath,
			Args:    []string{"--no-daemon", "--offline", "--build-cache", "clean", "compileJava"},
			Dir:     isolation.NativeDir,
			Env:     explicitEnvironment(isolation.NativeGradleHome, isolation.NativeCache, isolation.NativeState, false),
		}
		trial, runErr := stickytrial.RunPaired(context.Background(), assignment, isolation, candidate, native, []string{"build/classes/java/main"})
		if runErr != nil {
			if trial.Cancellation {
				fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: pair %d cancelled: %v\n", pair+1, runErr)
			} else {
				fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: run pair %d: %v\n", pair+1, runErr)
			}
			return 1
		}
		if trial.Result == stickytrial.ResultInconclusive {
			fmt.Fprintf(os.Stderr, "pair %d inconclusive: candidate=%s (%s), native=%s (%s)\n", trial.Pair, trial.Candidate.Outcome, trial.Candidate.Diagnostic(), trial.Native.Outcome, trial.Native.Diagnostic())
			return 1
		}
		trials = append(trials, trial)
		fmt.Fprintf(os.Stderr, "pair %d %s candidate=%.3fs native=%.3fs result=%s\n", trial.Pair, trial.Order, float64(trial.Candidate.DurationNs)/1e9, float64(trial.Native.DurationNs)/1e9, trial.Result)
	}
	report, err := stickytrial.Aggregate(scheduler.Snapshot(), trials, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: aggregate: %v\n", err)
		return 1
	}
	report.GradleVersion = *gradleVersion
	data, err := stickytrial.MarshalReport(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: encode report: %v\n", err)
		return 1
	}
	if err := writePrivate(*outputPath, data); err != nil {
		fmt.Fprintf(os.Stderr, "sticky-trial-benchmark: write report: %v\n", err)
		return 1
	}
	fmt.Printf("%s\n", string(data))
	return 0
}

func makeIsolation(root string) stickytrial.Isolation {
	return stickytrial.Isolation{
		CandidateDir:        filepath.Join(root, "candidate"),
		NativeDir:           filepath.Join(root, "native"),
		CandidateGradleHome: filepath.Join(root, "candidate-gradle-home"),
		NativeGradleHome:    filepath.Join(root, "native-gradle-home"),
		CandidateCache:      filepath.Join(root, "candidate-cache"),
		NativeCache:         filepath.Join(root, "native-cache"),
		CandidateState:      filepath.Join(root, "candidate-state"),
		NativeState:         filepath.Join(root, "native-state"),
	}
}

func prepareIsolation(template string, isolation stickytrial.Isolation) error {
	if err := copyTree(template, isolation.CandidateDir); err != nil {
		return err
	}
	if err := copyTree(template, isolation.NativeDir); err != nil {
		return err
	}
	for _, path := range []string{isolation.CandidateGradleHome, isolation.NativeGradleHome, isolation.CandidateCache, isolation.NativeCache, isolation.CandidateState, isolation.NativeState} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func explicitEnvironment(gradleHome, cacheRoot, stateRoot string, candidate bool) []string {
	environment := append([]string(nil), os.Environ()...)
	set := func(name, value string) {
		for index, entry := range environment {
			if strings.HasPrefix(entry, name+"=") {
				environment[index] = name + "=" + value
				return
			}
		}
		environment = append(environment, name+"="+value)
	}
	set("GRADLE_USER_HOME", gradleHome)
	set("XDG_CACHE_HOME", cacheRoot)
	set("BUILDOPT_STICKY_OBSERVATION", "0")
	if candidate {
		set("BUILDOPT_L1_STATE_ROOT", stateRoot)
		set("BUILDOPT_L1_TENANT_ID", "sticky-trial")
		set("BUILDOPT_L1_REPOSITORY_ID", "trial-repository")
		set("BUILDOPT_L1_TRUST_DOMAIN", "trusted-ci")
		set("BUILDOPT_L1_COMPATIBILITY_CLASS", "gradle-java-v1")
		set("BUILDOPT_L1_SECURITY_GENERATION", "1")
		set("BUILDOPT_L1_L2_WRITE_AUTHORIZED", "0")
		set("BUILDOPT_SAFE_CACHE", "1")
	} else {
		set("BUILDOPT_SAFE_CACHE", "0")
	}
	return environment
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := destination
		if relative != "." {
			target = filepath.Join(destination, relative)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("template contains a symlink")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o600
		}
		return os.WriteFile(target, data, fs.FileMode(mode))
	})
}

func writePrivate(path string, data []byte) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("output must be one clean absolute path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".sticky-trial-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func isRegular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func isRegularExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
