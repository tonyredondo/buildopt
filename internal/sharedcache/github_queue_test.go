package sharedcache

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/githubqueue"
)

func TestGitHubQueuePersistsExactAndUnavailableObservations(t *testing.T) {
	storage, err := Open(context.Background(), filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	created := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	started := created.Add(45 * time.Second)
	completed := started.Add(5 * time.Minute)
	event := githubqueue.Event{DeliveryID: "delivery-1", BodyDigest: "sha256:" + string(make([]byte, 64)), Action: "completed", RepositoryID: 7, RepositoryName: "owner/repo", JobID: 101, RunID: 202, RunAttempt: 1, HeadSHA: "0123456789abcdef0123456789abcdef01234567", Name: "build", Status: "completed", Conclusion: "success", CreatedAt: created, StartedAt: &started, CompletedAt: &completed, RunnerID: 11, RunnerName: "runner-11", RunnerGroupID: 22, RunnerGroupName: "linux-builders", Labels: []string{"x64", "linux", "linux"}}
	event.BodyDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if result, err := storage.PutGitHubWorkflowJob(context.Background(), event); err != nil || result != githubqueue.PutAccepted {
		t.Fatalf("put = %v/%v", result, err)
	}
	if result, err := storage.PutGitHubWorkflowJob(context.Background(), event); err != nil || result != githubqueue.PutDuplicate {
		t.Fatalf("duplicate = %v/%v", result, err)
	}
	observation, err := storage.GitHubQueueObservation(context.Background(), 101)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Availability != "EXACT" || observation.QueueMilliseconds != 45_000 || observation.RunnerGroupID != 22 || len(observation.Labels) != 2 {
		t.Fatalf("observation = %+v", observation)
	}
	conflict := event
	conflict.BodyDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := storage.PutGitHubWorkflowJob(context.Background(), conflict); !errors.Is(err, githubqueue.ErrConflict) {
		t.Fatalf("conflict = %v", err)
	}

	cancelled := event
	cancelled.DeliveryID, cancelled.JobID, cancelled.BodyDigest = "delivery-2", 102, "sha256:abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	cancelled.StartedAt, cancelled.CompletedAt, cancelled.RunnerID, cancelled.RunnerGroupID = nil, &completed, 0, 0
	cancelled.Conclusion = "cancelled"
	if _, err := storage.PutGitHubWorkflowJob(context.Background(), cancelled); err != nil {
		t.Fatal(err)
	}
	unavailable, err := storage.GitHubQueueObservation(context.Background(), 102)
	if err != nil || unavailable.Availability != "UNAVAILABLE" || unavailable.UnavailableReason != "CI_RUNNER_NOT_STARTED" {
		t.Fatalf("unavailable = %+v/%v", unavailable, err)
	}
}
