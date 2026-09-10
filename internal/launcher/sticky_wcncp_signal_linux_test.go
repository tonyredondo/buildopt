//go:build linux

package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickywrapper"
	"github.com/tonyredondo/buildopt/internal/wcncpobserve"
)

// Reuse the WS-002 child/descendant helper with application-owned environment
// names: BUILDOPT_ and WCNCP_ must remain private in the installed route.
func TestStickyWCNCPInstalledSignals(t *testing.T) {
	archive := os.Getenv("EIC_TEST_PACKAGE")
	if archive == "" {
		t.Skip("run dev/check-installed-native-correction for real package signal proof")
	}
	sourceRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	extracted := t.TempDir()
	runEICCommand(t, extracted, nil, "tar", "-xzf", archive, "-C", extracted)
	binary := filepath.Join(extracted, "buildopt-0.0.1-linux-amd64", "bin", "buildopt")
	source, err := os.ReadFile(filepath.Join(sourceRoot, "cmd/buildopt/testdata/signal-helper/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	helperDir := t.TempDir()
	helperSource := filepath.Join(helperDir, "main.go")
	if err := os.WriteFile(helperSource, bytes.ReplaceAll(source, []byte("BUILDOPT_TEST_"), []byte("NATIVE_TEST_")), 0600); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(helperDir, "signal-helper")
	runEICCommand(t, sourceRoot, nil, filepath.Join(sourceRoot, "dev/run"), "--toolchain", "go", "--", "go", "build", "-o", helper, helperSource)
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			root := t.TempDir()
			if _, err := (stickywrapper.Generator{Root: root, Resolver: wcncpFixtureResolver{}}).Init(context.Background(), stickywrapper.Config{Mode: "observe", ServerURL: "https://127.0.0.1:1", ProjectScope: "example/signals", CredentialEnv: "BUILDOPT_TOKEN", TrialBudgetPercent: 5}); err != nil {
				t.Fatal(err)
			}
			gradle := filepath.Join(root, "gradlew")
			if err := os.WriteFile(gradle, []byte("#!/bin/sh\nexec \"$NATIVE_HELPER\" \"$@\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("BUILDOPT_STICKY_WRAPPER_ROOT", root)
			t.Setenv("BUILDOPT_STICKY_OBSERVATION", "wcncp")
			t.Setenv("BUILDOPT_TOKEN", "")
			outbox := t.TempDir()
			t.Setenv("WCNCP_OUTBOX_DIR", outbox)
			t.Setenv("NATIVE_HELPER", helper)
			for _, name := range []string{"LEADER_READY", "DESCENDANT_READY", "LEADER_SIGNAL", "DESCENDANT_SIGNAL", "CLEANUP_COMPLETE"} {
				t.Setenv("NATIVE_TEST_"+name, filepath.Join(root, name))
			}
			t.Setenv("NATIVE_TEST_LEADER_EXIT_CODE", "42")
			t.Setenv("NATIVE_TEST_CLEANUP_DELAY", "0.01s")
			command := exec.Command(binary, "run", "--", gradle, "tree")
			command.Dir = root
			var output bytes.Buffer
			command.Stdout = &output
			command.Stderr = &output
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			waited := false
			defer func() {
				if !waited {
					_ = command.Process.Kill()
					_ = command.Wait()
				}
			}()
			for _, name := range []string{"LEADER_READY", "DESCENDANT_READY"} {
				deadline := time.Now().Add(5 * time.Second)
				for {
					if _, err := os.Stat(filepath.Join(root, name)); err == nil {
						break
					}
					if time.Now().After(deadline) {
						t.Fatalf("child readiness timeout for %s", name)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			if err := command.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- command.Wait() }()
			select {
			case <-done:
				waited = true
			case <-time.After(10 * time.Second):
				// The isolated helper reports its own process group. Kill only
				// this test's child group if cooperative cleanup fails.
				var process struct {
					PGID int `json:"pgid"`
				}
				raw, _ := os.ReadFile(filepath.Join(root, "LEADER_READY"))
				_ = json.Unmarshal(raw, &process)
				if process.PGID > 1 {
					_ = syscall.Kill(-process.PGID, syscall.SIGKILL)
				}
				_ = command.Process.Kill()
				<-done
				waited = true
				t.Fatal("signal forwarding timed out")
			}
			if command.ProcessState.ExitCode() != 42 {
				t.Fatalf("native signal handler exit=%d output=%q", command.ProcessState.ExitCode(), output.String())
			}
			for _, name := range []string{"LEADER_SIGNAL", "DESCENDANT_SIGNAL"} {
				raw, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || strings.TrimSpace(string(raw)) != strconv.Itoa(int(sig)) {
					t.Fatalf("%s did not receive signal: %q %v", name, raw, err)
				}
			}
			if raw, err := os.ReadFile(filepath.Join(root, "CLEANUP_COMPLETE")); err != nil || strings.TrimSpace(string(raw)) != "complete" {
				t.Fatal("descendant cleanup did not complete")
			}
			files, _ := filepath.Glob(filepath.Join(outbox, "*", "obs-*.json"))
			if len(files) != 1 {
				t.Fatalf("post-cleanup observation count=%d", len(files))
			}
			raw, _ := os.ReadFile(files[0])
			var facts wcncpobserve.ObservationFacts
			if err := json.Unmarshal(raw, &facts); err != nil || facts.Child.ExitCode == nil || *facts.Child.ExitCode != 42 {
				t.Fatalf("native handled signal outcome was replaced: %v", err)
			}
		})
	}
}
