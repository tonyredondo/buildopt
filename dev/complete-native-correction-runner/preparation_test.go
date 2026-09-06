package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDependencySeedAndFreshHomes(t *testing.T) {
	root, _, _ := setup(t)
	source := filepath.Join(root, "homes/prefetch-b")
	for name, body := range map[string]string{"caches/modules-2/files-2.1/dependency.jar": "jar", "caches/modules-2/metadata.bin": "metadata", "caches/modules-2/metadata.lock": "lock", "wrapper/dists/gradle/bin/gradle": "wrapper", "caches/build-cache-1/value": "forbidden", "caches/configuration-cache/value": "forbidden"} {
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := seedDependencies(root); err != nil {
		t.Fatal(err)
	}
	var inventory []fileBinding
	if err := readJSON(filepath.Join(root, "dependency-seed/inventory.json"), &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory) != 3 {
		t.Fatalf("unexpected seed: %+v", inventory)
	}
	a := filepath.Join(root, "homes/a")
	b := filepath.Join(root, "homes/b")
	if err := prepareHome(root, a, false, false); err != nil {
		t.Fatal(err)
	}
	if err := prepareHome(root, b, false, false); err != nil {
		t.Fatal(err)
	}
	for _, file := range inventory {
		first, _ := fileDigest(filepath.Join(a, file.Path))
		second, _ := fileDigest(filepath.Join(b, file.Path))
		if first != second || first != file.SHA256 {
			t.Fatal("asymmetric seed")
		}
	}
	if err := prepareHome(root, a, false, false); err == nil {
		t.Fatal("occupied home reused silently")
	}
	if err := os.WriteFile(filepath.Join(root, "dependency-seed", inventory[0].Path), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareHome(root, filepath.Join(root, "homes/c"), false, false); err == nil {
		t.Fatal("seed drift accepted")
	}
}

func TestEnvironmentAndReviewGates(t *testing.T) {
	root, state, _ := setup(t)
	good := environmentEvidence{Boot: state.Boot, Started: 100, Completed: 221, QuiescenceSeconds: 120, Samples: []float64{10, 10, 10, 10, 10, 10, 10}}
	if err := checkEnvironment(state, good); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.Samples = []float64{10, 10, 10, 10, 10, 10, 12}
	if err := checkEnvironment(state, bad); err == nil {
		t.Fatal("unstable environment accepted")
	}
	bad = good
	bad.Completed = 219
	if err := checkEnvironment(state, bad); err == nil {
		t.Fatal("quiescence skipped")
	}
	due := clockReading{state.Boot, state.ReviewAt}
	if err := reviewDue(root, state, due); err == nil {
		t.Fatal("missing review accepted")
	}
	if err := writeNewJSON(filepath.Join(root, "review.json"), reviewEvidence{state.ReviewAt, "continue", "progress and remaining proof checked"}); err != nil {
		t.Fatal(err)
	}
	if err := reviewDue(root, state, due); err != nil {
		t.Fatal(err)
	}
	if state.Deadline != 7300 {
		t.Fatal("review reset deadline")
	}
}

func TestDeadlineBeforeSpawnAndResourceFailure(t *testing.T) {
	root, state, clock := setup(t)
	r := childRequest(t, root, "P01", "success")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := capture(ctx, root, state, r, clock, nil)
	if err != nil || got.Started || got.Outcome != "PRE_START_FAILURE" {
		t.Fatalf("cancelled before spawn: %+v %v", got, err)
	}
	root, state, clock = setup(t)
	r = childRequest(t, root, "P01", "wait")
	got, err = capture(context.Background(), root, state, r, clock, nil, func() error { return errors.New("fixture disk budget exhausted") })
	if err != nil || got.Outcome != "HARNESS_FAILURE" || !got.Started {
		t.Fatalf("resource guard: %+v %v", got, err)
	}
}

func TestFractionalBootStateRoundTrip(t *testing.T) {
	for _, started := range []float64{248.01, 12345.67, 3467826.29} {
		t.Run(fmt.Sprint(started), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "state")
			now := clockReading{"fractional-boot", started}
			if err := initialize(root, strings.Repeat("a", 64), now); err != nil {
				t.Fatal(err)
			}
			var state campaign
			if err := readJSON(filepath.Join(root, "state.json"), &state); err != nil {
				t.Fatal(err)
			}
			if _, err := remaining(state, now); err != nil {
				t.Fatalf("freshly initialized state refused: %+v: %v", state, err)
			}
			for _, direction := range []float64{math.Inf(-1), math.Inf(1)} {
				changed := state
				changed.Deadline = math.Nextafter(state.Deadline, direction)
				if err := validateState(changed); err == nil {
					t.Fatal("one-ULP deadline drift accepted")
				}
				changed = state
				changed.ReviewAt = math.Nextafter(state.ReviewAt, direction)
				if err := validateState(changed); err == nil {
					t.Fatal("one-ULP review drift accepted")
				}
			}
			if _, err := remaining(state, clockReading{state.Boot, state.Deadline}); err == nil {
				t.Fatal("expired state accepted")
			}
		})
	}
}

func TestActualCLIInitializationAndReadOnlyCheck(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	manifest := filepath.Join(temporary, "package.json")
	state := filepath.Join(temporary, "state")
	if err := run([]string{"freeze", "--repo", repo, "--package", manifest}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"init", "--repo", repo, "--package", manifest, "--state", state}); err == nil {
		t.Fatal("initialization without acknowledgement")
	}
	if err := run([]string{"init", "--repo", repo, "--package", manifest, "--state", state, "--acknowledge-phase-a"}); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, state)
	if err := run([]string{"check", "--repo", repo, "--package", manifest, "--state", state}); err != nil {
		t.Fatal(err)
	}
	if before != snapshot(t, state) {
		t.Fatal("check mutated state")
	}
	if err := run([]string{"unknown", "--repo", repo}); err == nil {
		t.Fatal("unknown phase accepted")
	}
	source, home := expectedPaths(state, "P01")
	if err := run([]string{"capture", "--repo", repo, "--package", manifest, "--state", state, "--slot", "P01", "--source", source, "--gradle-home", home, "--acknowledge-phase-a"}); err == nil {
		t.Fatal("execution without committed package/guardian/runtime prerequisites accepted")
	}
	if before != snapshot(t, state) {
		t.Fatal("failed public prerequisite wrote an attempt")
	}
	if err := run([]string{"init", "--repo", repo, "--package", manifest, "--state", state, "--acknowledge-phase-a"}); err == nil {
		t.Fatal("occupied state overwritten")
	}
}

func TestTerminalPublicationFailureRetainsAttempt(t *testing.T) {
	root, state, clock := setup(t)
	r := childRequest(t, root, "P01", "success")
	_, err := capture(context.Background(), root, state, r, clock, func() error {
		path := filepath.Join(root, "attempts/P01/result.json")
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		return os.Mkdir(path, 0700)
	})
	if err == nil {
		t.Fatal("terminal publication failure hidden")
	}
	for _, name := range []string{"reservation.json", "request.json", "started.json", "process.json", "child.log"} {
		if _, err := fileDigest(filepath.Join(root, "attempts/P01", name)); err != nil {
			t.Fatalf("lost raw evidence %s: %v", name, err)
		}
	}
	if _, err := inspect(root, state); err == nil {
		t.Fatal("incomplete publication accepted")
	}
	if _, err := capture(context.Background(), root, state, r, clock, nil); err == nil {
		t.Fatal("incomplete attempt replayed")
	}
}

func TestSummaryAndRequestTampering(t *testing.T) {
	for _, which := range []string{"summary", "request"} {
		t.Run(which, func(t *testing.T) {
			root, state, clock := setup(t)
			r := childRequest(t, root, "P01", "failure")
			if _, err := capture(context.Background(), root, state, r, clock, nil); err != nil {
				t.Fatal(err)
			}
			if which == "summary" {
				path := filepath.Join(root, "attempts/P01/result.json")
				var terminal result
				if err := readJSON(path, &terminal); err != nil {
					t.Fatal(err)
				}
				terminal.Outcome = "CHILD_SUCCESS"
				data, _ := json.Marshal(terminal)
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				path := filepath.Join(root, "attempts/P01/request.json")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				data = []byte(strings.Replace(string(data), "-test.run=^TestChild$", "changed-request", 1))
				if err = os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := inspect(root, state); err == nil {
				t.Fatal("tampered evidence accepted")
			}
		})
	}
}
