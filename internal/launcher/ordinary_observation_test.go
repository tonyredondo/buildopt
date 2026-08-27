package launcher

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/stickyobservation"
)

func TestOrdinaryObservationRecordsFailedLaunchWithUnavailablePhases(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "observations", "builds.jsonl")
	t.Setenv(stickyObservationOutputEnvironment, output)
	state := newOrdinaryObservationState(root, []string{filepath.Join(root, "gradlew"), "help"})
	if state == nil {
		t.Fatal("ordinary observation was not initialized")
	}
	state.markConnection(state.startedAt, false)
	state.finishGradle(childExecution{err: errObservationTestLaunch}, []string{"help"})
	completed := state.startedAt.Add(10 * time.Millisecond)
	if err := state.finish(127, completed); err != nil {
		t.Fatal(err)
	}
	records, err := stickyobservation.Load(output)
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %d/%v", len(records), err)
	}
	if records[0].Outcome != "INFRA_FAILURE" || records[0].Timing.Gradle.Evidence != "UNAVAILABLE" {
		t.Fatalf("failed launch record = %+v", records[0])
	}
}

var errObservationTestLaunch = testLaunchError("synthetic launch failure")

type testLaunchError string

func (err testLaunchError) Error() string { return string(err) }
