package wcncpvalidate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

func TestAdmissionRequiresActionableCompleteAndBudget(t *testing.T) {
	t.Parallel()
	admitted := ProposalAdmission{Decision: "ACTIONABLE_MATERIAL_CORRECTION", BindingsComplete: true, BudgetRemaining: true, PreimageSHA256: fmt.Sprintf("%064d", 1), SourcePath: "build.gradle.kts"}
	if err := Admit(admitted); err != nil {
		t.Fatal(err)
	}
	for _, rejected := range []ProposalAdmission{
		{Decision: "NON_MATERIAL_BLOCKER", BindingsComplete: true, BudgetRemaining: true, PreimageSHA256: fmt.Sprintf("%064d", 1), SourcePath: "a"},
		{Decision: "ACTIONABLE_MATERIAL_CORRECTION", BindingsComplete: false, BudgetRemaining: true, PreimageSHA256: fmt.Sprintf("%064d", 1), SourcePath: "a"},
		{Decision: "ACTIONABLE_MATERIAL_CORRECTION", BindingsComplete: true, BudgetRemaining: false, PreimageSHA256: fmt.Sprintf("%064d", 1), SourcePath: "a"},
	} {
		if err := Admit(rejected); err == nil {
			t.Fatalf("admitted %+v", rejected)
		}
	}
}

func TestBudgetStopsBeforeLimitWithoutSideEffects(t *testing.T) {
	t.Parallel()
	budget := &Budget{MaxControlledMs: 1000, MaxDiskBytes: 1000}
	if err := budget.Charge(600, 600, false); err != nil {
		t.Fatal(err)
	}
	if err := budget.Charge(500, 0, false); err == nil {
		t.Fatal("over-budget charge accepted")
	}
	retries := &Budget{MaxControlledMs: 1 << 40, MaxDiskBytes: 1 << 40}
	for i := 0; i < MaximumInfrastructureRetries; i++ {
		if err := retries.Charge(1, 1, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := retries.Charge(1, 1, true); err == nil {
		t.Fatal("third infra retry accepted")
	}
}

func TestIsolatedRootsNeverTouchOutside(t *testing.T) {
	t.Parallel()
	experiment := t.TempDir()
	root, err := IsolatedRoot(experiment, "candidate-1")
	if err != nil || root == "" {
		t.Fatal(err)
	}
	if err := DiscardRoot(experiment, root); err != nil {
		t.Fatal(err)
	}
	if err := DiscardRoot(experiment, experiment); err == nil {
		t.Fatal("experiment root discard accepted")
	}
	if err := DiscardRoot(experiment, "/tmp"); err == nil {
		t.Fatal("broad path discard accepted")
	}
}

func TestApplyTransactionRejectsDriftAndRoundTrips(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := "build.gradle.kts"
	original := []byte("tasks {\n  slowTask {}\n}\n")
	if err := os.WriteFile(filepath.Join(root, path), original, 0o600); err != nil {
		t.Fatal(err)
	}
	segment := []byte("slowTask {}")
	digest := sha256.Sum256(segment)
	preimage := hex.EncodeToString(digest[:])
	start := int64(len("tasks {\n  "))
	end := start + int64(len(segment))
	previous, err := ApplyTransaction(root, path, start, end, preimage, []byte("fastTask {}"))
	if err != nil || string(previous) != string(original) {
		t.Fatalf("apply = %v", err)
	}
	// Reapply is idempotent only against the new preimage; stale preimage fails.
	if _, err := ApplyTransaction(root, path, start, end, preimage, []byte("fastTask {}")); err == nil {
		t.Fatal("stale preimage accepted")
	}
	after, _ := os.ReadFile(filepath.Join(root, path))
	forward := sha256.Sum256([]byte("fastTask {}"))
	inversePreimage := hex.EncodeToString(forward[:])
	if _, err := ApplyTransaction(root, path, start, start+int64(len("fastTask {}")), inversePreimage, segment); err != nil {
		t.Fatalf("revert = %v", err)
	}
	restored, _ := os.ReadFile(filepath.Join(root, path))
	if string(restored) != string(original) {
		t.Fatal("exact revert mismatch")
	}
	_ = after
}

func TestApplyTransactionRejectsPostimageMismatchAndPreservesMode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "build.gradle.kts")
	original := []byte("slow")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}
	preimage := sha256.Sum256(original)
	wrongPostimage := sha256.Sum256([]byte("other"))
	if _, err := ApplyTransactionChecked(root, "build.gradle.kts", 0, int64(len(original)), hex.EncodeToString(preimage[:]), hex.EncodeToString(wrongPostimage[:]), []byte("fast")); err == nil {
		t.Fatal("postimage mismatch accepted")
	}
	postimage := sha256.Sum256([]byte("fast"))
	if _, err := ApplyTransactionChecked(root, "build.gradle.kts", 0, int64(len(original)), hex.EncodeToString(preimage[:]), hex.EncodeToString(postimage[:]), []byte("fast")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
}

func TestApplyTransactionRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unprivileged Windows symlink creation is not portable")
	}
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.gradle.kts")
	if err := os.WriteFile(outside, []byte("slow"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "build.gradle.kts")); err != nil {
		t.Fatal(err)
	}
	preimage := sha256.Sum256([]byte("slow"))
	if _, err := ApplyTransaction(root, "build.gradle.kts", 0, 4, hex.EncodeToString(preimage[:]), []byte("fast")); err == nil {
		t.Fatal("symlink escape accepted")
	}
	raw, err := os.ReadFile(outside)
	if err != nil || string(raw) != "slow" {
		t.Fatalf("outside file changed: %q/%v", raw, err)
	}
}

func TestFixtureCorrectnessComparesExactOutputs(t *testing.T) {
	t.Parallel()
	build := func(root, input string) (map[string][]byte, error) {
		return map[string][]byte{"output.txt": []byte("result:" + input)}, nil
	}
	qualified := RunFixtureCorrectness(build, "baseline", "changed")
	if qualified.Decision != "QUALIFIED" || !qualified.ExactOutputs || qualified.Starts < MinimumCorrectnessStarts {
		t.Fatalf("qualified = %+v", qualified)
	}
	regressive := RunFixtureCorrectness(func(root, input string) (map[string][]byte, error) {
		if input == "changed" {
			return map[string][]byte{"output.txt": []byte("result:baseline")}, nil
		}
		return map[string][]byte{"output.txt": []byte("result:" + input)}, nil
	}, "baseline", "changed")
	if regressive.Decision != "REJECTED_CORRECTNESS" {
		t.Fatalf("missing invalidation accepted = %+v", regressive)
	}
}

func TestFixtureCorrectnessReportsActualStartsAndProductFailure(t *testing.T) {
	t.Parallel()
	starts := 0
	result := RunFixtureCorrectness(func(root, input string) (map[string][]byte, error) {
		starts++
		if starts == 2 {
			return nil, errors.New("candidate failed")
		}
		return map[string][]byte{"output.txt": []byte(input)}, nil
	}, "baseline", "changed")
	if result.Starts != 2 || result.ProductFailures != 1 || result.Decision != "REJECTED_CORRECTNESS" {
		t.Fatalf("candidate failure = %+v", result)
	}

	controlFailure := RunFixtureCorrectness(func(root, input string) (map[string][]byte, error) {
		return nil, errors.New("environment unavailable")
	}, "baseline", "changed")
	if controlFailure.Starts != 1 || controlFailure.ProductFailures != 0 {
		t.Fatalf("control failure = %+v", controlFailure)
	}
}

func TestLeaseClaimExpiryAndPublicationGate(t *testing.T) {
	ctx := t.Context()
	storage, err := sharedcache.Open(ctx, t.TempDir()+"/shared")
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC().Truncate(time.Millisecond)
	repo := fmt.Sprintf("%064d", 9)
	proposal := fmt.Sprintf("%064d", 7)
	first, err := storage.ClaimWCNCPLease(ctx, repo, proposal, ProtocolVersion, "LOCAL_FUNCTIONAL", "validator-a", time.Minute, now)
	if err != nil {
		t.Fatal(err)
	}
	// Two validators race for one proposal; exactly one acquires the lease.
	var group sync.WaitGroup
	results := make(chan error, 2)
	for _, holder := range []string{"validator-b", "validator-c"} {
		group.Add(1)
		go func(holder string) {
			defer group.Done()
			_, err := storage.ClaimWCNCPLease(ctx, repo, proposal, ProtocolVersion, "LOCAL_FUNCTIONAL", holder, time.Minute, now)
			results <- err
		}(holder)
	}
	group.Wait()
	close(results)
	held := 0
	for err := range results {
		if err != nil {
			held++
		}
	}
	if held != 2 {
		t.Fatalf("racing claims held = %d, want 2", held)
	}
	// Heartbeats extend only the same lease holder.
	if err := storage.HeartbeatWCNCPLease(ctx, first.LeaseID, "validator-a", now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := storage.HeartbeatWCNCPLease(ctx, first.LeaseID, "validator-b", now.Add(30*time.Second)); err == nil {
		t.Fatal("stale holder heartbeat accepted")
	}
	// Publication requires the still-current lease and exact digest.
	if err := storage.RequireWCNCPLease(ctx, first.LeaseID, "validator-a", proposal, now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := storage.RequireWCNCPLease(ctx, first.LeaseID, "validator-b", proposal, now.Add(30*time.Second)); err == nil {
		t.Fatal("stale holder publication accepted")
	}
	// Late result after expiry fails closed; expiry is visible and requeueable.
	expired, err := storage.ExpireWCNCPLeases(ctx, now.Add(2*time.Minute))
	if err != nil || expired != 1 {
		t.Fatalf("expiry = %d/%v", expired, err)
	}
	if err := storage.RequireWCNCPLease(ctx, first.LeaseID, "validator-a", proposal, now.Add(2*time.Minute)); err == nil {
		t.Fatal("late result accepted")
	}
	reclaimed, err := storage.ClaimWCNCPLease(ctx, repo, proposal, ProtocolVersion, "LOCAL_FUNCTIONAL", "validator-b", time.Minute, now.Add(2*time.Minute))
	if err != nil || reclaimed.Holder != "validator-b" {
		t.Fatalf("reclaim = %+v/%v", reclaimed, err)
	}
}
