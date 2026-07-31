package runtimeoptimizer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSchedulerPersistsIsolatedIdempotentLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "runtime")
	scheduler, err := OpenScheduler(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	request := testRequest("request-1")
	entry, created, err := scheduler.Schedule(request)
	if err != nil || !created {
		t.Fatalf("schedule = %+v/%v/%v", entry, created, err)
	}
	if err := entry.ValidateIsolation(); err != nil {
		t.Fatal(err)
	}
	if entry.Plan.Candidate.ReadNamespace != entry.Plan.Candidate.WriteNamespace || entry.Plan.Candidate.WriteNamespace == entry.Plan.Stable.WriteNamespace || entry.Plan.Control.WriteNamespace != "" || !entry.Plan.Control.Authoritative {
		t.Fatalf("plan = %+v", entry.Plan)
	}
	for _, path := range []string{entry.Plan.Candidate.Workspace, entry.Plan.Candidate.Outputs, entry.Plan.Candidate.GradleUserHome, entry.Plan.Candidate.CredentialPath, entry.Plan.Control.Workspace, entry.Plan.Control.Outputs, entry.Plan.Control.GradleUserHome, entry.Plan.Control.CredentialPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0o700)
		if info.Mode().IsRegular() {
			want = 0o600
		}
		if info.Mode().Perm() != want {
			t.Fatalf("mode %s = %o", path, info.Mode().Perm())
		}
	}
	repeat, created, err := scheduler.Schedule(request)
	if err != nil || created || repeat.Plan.AttemptID != entry.Plan.AttemptID {
		t.Fatalf("repeat = %+v/%v/%v", repeat, created, err)
	}
	conflict := request
	conflict.ActionID = "other-action"
	if _, _, err := scheduler.Schedule(conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict = %v", err)
	}
	leased, err := scheduler.Lease(request.RequestID, "worker-1", 10*time.Minute)
	if err != nil || leased.State != "LEASED" {
		t.Fatalf("lease = %+v/%v", leased, err)
	}
	finished, err := scheduler.Finish(request.RequestID, "worker-1", true)
	if err != nil || finished.State != "COMPLETED" {
		t.Fatalf("finish = %+v/%v", finished, err)
	}
	reopened, err := OpenScheduler(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	persisted, created, err := reopened.Schedule(request)
	if err != nil || created || persisted.State != "COMPLETED" {
		t.Fatalf("persisted = %+v/%v/%v", persisted, created, err)
	}
}

func TestSchedulerSerializesRepositoryAndRecoversExpiredLease(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	scheduler, err := OpenScheduler(filepath.Join(t.TempDir(), "runtime"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	first := testRequest("request-1")
	second := testRequest("request-2")
	if _, _, err := scheduler.Schedule(first); err != nil {
		t.Fatal(err)
	}
	if _, _, err := scheduler.Schedule(second); err != nil {
		t.Fatal(err)
	}
	if _, err := scheduler.Lease(first.RequestID, "worker-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := scheduler.Lease(second.RequestID, "worker-2", time.Minute); !errors.Is(err, ErrRepositoryBusy) {
		t.Fatalf("busy = %v", err)
	}
	now = now.Add(2 * time.Minute)
	leased, err := scheduler.Lease(second.RequestID, "worker-2", time.Minute)
	if err != nil || leased.LeaseOwner != "worker-2" {
		t.Fatalf("recovered lease = %+v/%v", leased, err)
	}
}

func TestSchedulerRejectsUnsafeExistingCredential(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "runtime")
	scheduler, err := OpenScheduler(root, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	request := testRequest("unsafe-request")
	credentialPath := scheduler.plan(request).Candidate.CredentialPath
	if err := os.MkdirAll(filepath.Dir(credentialPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside-credential"), credentialPath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := scheduler.Schedule(request); err == nil {
		t.Fatal("unsafe credential symlink was accepted")
	}
}

func testRequest(id string) Request {
	return Request{RequestID: id, RepositoryID: "repository-1", ActionID: "action-1", PolicyDigest: "sha256:" + repeat("1", 64), BaselineDigest: "sha256:" + repeat("2", 64), WorkUnitsFingerprint: "sha256:" + repeat("3", 64), Platform: "linux-amd64", CacheCompatibilityClass: "gradle-9", ValidatesCache: true}
}
func repeat(value string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += value
	}
	return result
}
