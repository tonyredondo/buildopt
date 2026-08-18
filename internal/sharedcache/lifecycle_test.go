package sharedcache

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

const (
	testDecisionKeyID   = "commit-key-test"
	testRevocationEpoch = int64(7)
)

var lifecycleTestNow = time.Date(
	2026,
	time.July,
	30,
	12,
	0,
	0,
	0,
	time.UTC,
)

func TestPendingCommitVisibilityReplayAndExactAuthority(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	request := lifecycleAttemptRequest(
		"attempt-happy",
		"start-happy",
		"owner-one",
		12,
	)
	status, created, err := storage.StartAttempt(ctx, request)
	if err != nil || !created ||
		status.State != AttemptPending ||
		status.StateVersion != 1 {
		t.Fatalf("start = %+v/%t/%v", status, created, err)
	}
	replay, created, err := storage.StartAttempt(ctx, request)
	if err != nil || created || replay != status {
		t.Fatalf("start replay = %+v/%t/%v", replay, created, err)
	}
	conflict := request
	conflict.OwnerID = "changed-owner"
	if _, _, err := storage.StartAttempt(
		ctx,
		conflict,
	); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed start replay = %v", err)
	}

	contentByKey := map[string][]byte{
		"aaaaaaaa": []byte("first opaque Gradle object"),
		"bbbbbbbb": []byte("second opaque Gradle object"),
	}
	var objects []CommitObject
	for _, key := range []string{"aaaaaaaa", "bbbbbbbb"} {
		result, err := storage.PutPending(
			ctx,
			request.AttemptID,
			key,
			bytes.NewReader(contentByKey[key]),
		)
		if err != nil || !result.ObjectAdded {
			t.Fatalf("put %s = %+v/%v", key, result, err)
		}
		objects = append(objects, result.Object)
	}
	if file, _, err := storage.OpenCommitted(
		ctx,
		request.Repository.Tenant,
		request.NamespaceGeneration,
		"aaaaaaaa",
	); !errors.Is(err, ErrCacheMiss) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("pending read = %v", err)
	}
	status, err = storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil ||
		status.PendingObjectCount != 2 ||
		status.StateVersion != 3 {
		t.Fatalf("pending status = %+v/%v", status, err)
	}
	pendingObjects, err := storage.PendingAttemptObjects(ctx, request.AttemptID)
	if err != nil || len(pendingObjects) != len(objects) {
		t.Fatalf("pending owner inventory = %+v/%v", pendingObjects, err)
	}
	for index := range objects {
		if pendingObjects[index] != objects[index] {
			t.Fatalf("pending owner object %d = %+v, want %+v", index, pendingObjects[index], objects[index])
		}
	}

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	canonical := signLifecycleDecision(
		t,
		privateKey,
		request,
		"decision-happy",
		objects,
		testRevocationEpoch,
		lifecycleTestNow,
	)
	verified, err := VerifyCommitDecision(
		ctx,
		canonical,
		map[string]ed25519.PublicKey{
			testDecisionKeyID: publicKey,
		},
		testRevocationEpoch,
		lifecycleTestNow,
	)
	if err != nil {
		t.Fatalf("verify decision: %v", err)
	}
	commit, err := storage.CommitAttempt(
		ctx,
		3,
		testRevocationEpoch,
		verified,
	)
	if err != nil ||
		commit.Outcome != "COMMITTED" ||
		commit.ObjectCount != 2 ||
		commit.StateVersion != 4 ||
		!commit.AuditIndexed ||
		commit.RequiresReconcile {
		t.Fatalf("commit = %+v/%v", commit, err)
	}
	for key, want := range contentByKey {
		file, object, err := storage.OpenCommitted(
			ctx,
			request.Repository.Tenant,
			request.NamespaceGeneration,
			key,
		)
		if err != nil {
			t.Fatalf("open %s: %v", key, err)
		}
		actual, readErr := os.ReadFile(file.Name())
		closeErr := file.Close()
		if readErr != nil ||
			closeErr != nil ||
			!bytes.Equal(actual, want) ||
			object.DecisionDigest != commit.DecisionDigest {
			t.Fatalf(
				"hit %s = %q/%+v read=%v close=%v",
				key,
				actual,
				object,
				readErr,
				closeErr,
			)
		}
	}
	replayed, err := storage.CommitAttempt(
		ctx,
		3,
		testRevocationEpoch,
		verified,
	)
	if err != nil ||
		replayed.Outcome != "IDEMPOTENT_REPLAY" ||
		replayed.CommittedAt != commit.CommittedAt {
		t.Fatalf("commit replay = %+v/%v", replayed, err)
	}

	changedCanonical := signLifecycleDecision(
		t,
		privateKey,
		request,
		"decision-changed",
		objects,
		testRevocationEpoch,
		lifecycleTestNow,
	)
	changed, err := VerifyCommitDecision(
		ctx,
		changedCanonical,
		map[string]ed25519.PublicKey{
			testDecisionKeyID: publicKey,
		},
		testRevocationEpoch,
		lifecycleTestNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CommitAttempt(
		ctx,
		3,
		testRevocationEpoch,
		changed,
	); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed decision replay = %v", err)
	}

	assertRowCount(t, storage.cache.database, "commit_decisions", 1)
	assertRowCount(t, storage.cache.database, "committed_objects", 2)
	assertRowCount(t, storage.cache.database, "pending_objects", 0)
	assertRowCount(t, storage.control.database, "decision_audit_index", 1)
}

func TestOpenCommittedVerifiesConcurrentlyAndBlocksMetadataMutation(
	t *testing.T,
) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	request := lifecycleAttemptRequest(
		"attempt-parallel-read",
		"start-parallel-read",
		"owner-parallel-read",
		13,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"parallel-object",
		bytes.NewReader(bytes.Repeat([]byte("verified"), 1<<12)),
	)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifyLifecycleDecision(
		t,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		signLifecycleDecision(
			t,
			privateKey,
			request,
			"decision-parallel-read",
			[]CommitObject{pending.Object},
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	if _, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch,
		verified,
	); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	storage.testHooks.beforeCommittedBlobVerify = func() {
		entered <- struct{}{}
		<-release
	}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			file, _, openErr := storage.OpenCommitted(
				ctx,
				request.Repository.Tenant,
				request.NamespaceGeneration,
				pending.Object.Key,
			)
			if file != nil {
				_ = file.Close()
			}
			results <- openErr
		}()
	}
	for range 2 {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			close(release)
			t.Fatal("committed blob verification remained globally serialized")
		}
	}
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("parallel committed read = %v", err)
		}
	}

	storage.clock = func() time.Time {
		return lifecycleTestNow.Add(2 * time.Minute)
	}
	entered = make(chan struct{}, 2)
	release = make(chan struct{})
	storage.testHooks.beforeCommittedBlobVerify = func() {
		entered <- struct{}{}
		<-release
	}
	batches := make(chan int, 2)
	flushed := make(chan error, 2)
	storage.testHooks.beforeProtectedAccessBatch = func(size int) error {
		batches <- size
		return nil
	}
	storage.testHooks.afterProtectedAccessBatch = func(err error) {
		flushed <- err
	}
	results = make(chan error, 2)
	for range 2 {
		go func() {
			file, _, openErr := storage.OpenCommitted(
				ctx,
				request.Repository.Tenant,
				request.NamespaceGeneration,
				pending.Object.Key,
			)
			if file != nil {
				_ = file.Close()
			}
			results <- openErr
		}()
	}
	for range 2 {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			close(release)
			t.Fatal("stale protected reads did not verify concurrently")
		}
	}
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("batched protected read = %v", err)
		}
	}
	select {
	case size := <-batches:
		if size != 1 {
			t.Fatalf("deduplicated protected access batch size = %d", size)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("protected accesses did not start a batch")
	}
	select {
	case err := <-flushed:
		if err != nil {
			t.Fatalf("flush protected access batch: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("protected access batch did not finish")
	}
	select {
	case size := <-batches:
		t.Fatalf("protected accesses used a second batch of size %d", size)
	default:
	}
	storage.testHooks.beforeProtectedAccessBatch = nil
	storage.testHooks.afterProtectedAccessBatch = nil
	file, object, err := storage.OpenCommitted(
		ctx,
		request.Repository.Tenant,
		request.NamespaceGeneration,
		pending.Object.Key,
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	storage.clock = func() time.Time {
		return lifecycleTestNow.Add(4 * time.Minute)
	}
	flushed = make(chan error, 1)
	storage.testHooks.afterProtectedAccessBatch = func(err error) {
		flushed <- err
	}
	storage.lifecycleMutex.RLock()
	if err := storage.batchProtectedAccess(
		ctx,
		object,
		storage.now(),
	); err != nil {
		storage.lifecycleMutex.RUnlock()
		t.Fatal(err)
	}
	select {
	case err := <-flushed:
		if err != nil {
			storage.lifecycleMutex.RUnlock()
			t.Fatalf("flush beside verified reader = %v", err)
		}
	case <-time.After(2 * time.Second):
		storage.lifecycleMutex.RUnlock()
		t.Fatal("protected access flush waited for unrelated verified reader")
	}
	storage.lifecycleMutex.RUnlock()
	storage.testHooks.afterProtectedAccessBatch = nil

	entered = make(chan struct{}, 1)
	release = make(chan struct{})
	storage.testHooks.beforeCommittedBlobVerify = func() {
		entered <- struct{}{}
		<-release
	}
	result := make(chan error, 1)
	go func() {
		file, _, openErr := storage.OpenCommitted(
			ctx,
			request.Repository.Tenant,
			request.NamespaceGeneration,
			pending.Object.Key,
		)
		if file != nil {
			_ = file.Close()
		}
		result <- openErr
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("committed blob read did not reach verification")
	}
	mutated := make(chan error, 1)
	go func() {
		storage.lifecycleMutex.Lock()
		_, mutationErr := storage.cache.database.ExecContext(
			ctx,
			`DELETE FROM committed_objects
WHERE tenant_id = ? AND namespace_generation = ? AND cache_key = ?`,
			request.Repository.Tenant,
			request.NamespaceGeneration,
			pending.Object.Key,
		)
		storage.lifecycleMutex.Unlock()
		mutated <- mutationErr
	}()
	select {
	case mutationErr := <-mutated:
		close(release)
		t.Fatalf(
			"metadata mutated during verified read: %v",
			mutationErr,
		)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatalf("protected committed read = %v", err)
	}
	if err := <-mutated; err != nil {
		t.Fatal(err)
	}
	storage.testHooks.beforeCommittedBlobVerify = nil
	file, _, err = storage.OpenCommitted(
		ctx,
		request.Repository.Tenant,
		request.NamespaceGeneration,
		pending.Object.Key,
	)
	if file != nil {
		_ = file.Close()
	}
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("read after metadata removal = %v", err)
	}
}

func TestAbortIsIdempotentAndReconciliationDeletesReleasedBlob(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	request := lifecycleAttemptRequest(
		"attempt-abort",
		"start-abort",
		"owner-one",
		3,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"abort-key",
		strings.NewReader("pending bytes"),
	)
	if err != nil {
		t.Fatal(err)
	}
	abortRequest := AbortAttemptRequest{
		RequestID:            "abort-command",
		AttemptID:            request.AttemptID,
		ExpectedStateVersion: 2,
		Reason:               "BUILD_FAILURE",
	}
	aborted, err := storage.AbortAttempt(ctx, abortRequest)
	if err != nil ||
		aborted.Outcome != "ABORTED" ||
		aborted.Status.State != AttemptAborted ||
		aborted.Status.PendingObjectCount != 0 {
		t.Fatalf("abort = %+v/%v", aborted, err)
	}
	replay, err := storage.AbortAttempt(ctx, abortRequest)
	if err != nil || replay.Outcome != "ALREADY_ABORTED" {
		t.Fatalf("abort replay = %+v/%v", replay, err)
	}
	changed := abortRequest
	changed.RequestID = "changed-abort-command"
	if _, err := storage.AbortAttempt(
		ctx,
		changed,
	); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed abort replay = %v", err)
	}
	if file, _, err := storage.OpenCommitted(
		ctx,
		request.Repository.Tenant,
		request.NamespaceGeneration,
		pending.Object.Key,
	); !errors.Is(err, ErrCacheMiss) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("aborted read = %v", err)
	}
	report, err := storage.Reconcile(ctx)
	if err != nil || report.DeletedOrphanBlobs != 1 {
		t.Fatalf("reconcile = %+v/%v", report, err)
	}
	assertEmptyDirectory(t, storage.Layout().Spool)
	assertBlobAbsent(t, storage, pending.Object.Checksum)
}

func TestFirstWriterCASAbortsTheCompleteLosingAttempt(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{testDecisionKeyID: publicKey}

	first := lifecycleAttemptRequest(
		"attempt-cas-first",
		"start-cas-first",
		"owner-one",
		9,
	)
	second := lifecycleAttemptRequest(
		"attempt-cas-second",
		"start-cas-second",
		"owner-two",
		9,
	)
	for _, request := range []StartAttemptRequest{first, second} {
		if _, _, err := storage.StartAttempt(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
	firstObject, err := storage.PutPending(
		ctx,
		first.AttemptID,
		"same-key",
		strings.NewReader("first writer"),
	)
	if err != nil {
		t.Fatal(err)
	}
	secondObject, err := storage.PutPending(
		ctx,
		second.AttemptID,
		"same-key",
		strings.NewReader("second writer"),
	)
	if err != nil {
		t.Fatal(err)
	}
	firstDecision := verifyLifecycleDecision(
		t,
		keys,
		signLifecycleDecision(
			t,
			privateKey,
			first,
			"decision-cas-first",
			[]CommitObject{firstObject.Object},
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	if _, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch,
		firstDecision,
	); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	secondDecision := verifyLifecycleDecision(
		t,
		keys,
		signLifecycleDecision(
			t,
			privateKey,
			second,
			"decision-cas-second",
			[]CommitObject{secondObject.Object},
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	if _, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch,
		secondDecision,
	); !errors.Is(err, ErrCASLost) {
		t.Fatalf("second CAS = %v", err)
	}
	if _, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch,
		secondDecision,
	); !errors.Is(err, ErrCASLost) {
		t.Fatalf("second CAS replay = %v", err)
	}
	secondStatus, err := storage.AttemptStatus(ctx, second.AttemptID)
	if err != nil ||
		secondStatus.State != AttemptAborted ||
		secondStatus.PendingObjectCount != 0 ||
		secondStatus.AbortReason != "CAS_LOST" {
		t.Fatalf("CAS loser status = %+v/%v", secondStatus, err)
	}
	file, _, err := storage.OpenCommitted(
		ctx,
		first.Repository.Tenant,
		first.NamespaceGeneration,
		"same-key",
	)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(file.Name())
	_ = file.Close()
	if err != nil || string(actual) != "first writer" {
		t.Fatalf("CAS winner bytes = %q/%v", actual, err)
	}
	report, err := storage.Reconcile(ctx)
	if err != nil || report.DeletedOrphanBlobs != 1 {
		t.Fatalf("CAS reconcile = %+v/%v", report, err)
	}
	assertBlobAbsent(t, storage, secondObject.Object.Checksum)
}

func TestConcurrentCommitAttemptsPublishExactlyOneCompleteWinner(
	t *testing.T,
) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{testDecisionKeyID: publicKey}
	type commitCandidate struct {
		name      string
		request   StartAttemptRequest
		objects   []CommitObject
		content   map[string]string
		decision  VerifiedCommitDecision
		commit    CommitResult
		commitErr error
	}
	candidates := []*commitCandidate{
		{
			name: "first",
			request: lifecycleAttemptRequest(
				"attempt-concurrent-first",
				"start-concurrent-first",
				"owner-concurrent-first",
				10,
			),
		},
		{
			name: "second",
			request: lifecycleAttemptRequest(
				"attempt-concurrent-second",
				"start-concurrent-second",
				"owner-concurrent-second",
				10,
			),
		},
	}
	for _, candidate := range candidates {
		if _, _, err := storage.StartAttempt(
			ctx,
			candidate.request,
		); err != nil {
			t.Fatal(err)
		}
		candidate.content = map[string]string{
			"shared-key":             candidate.name + " shared bytes",
			"only-" + candidate.name: candidate.name + " exclusive bytes",
		}
		for _, key := range []string{
			"only-" + candidate.name,
			"shared-key",
		} {
			pending, err := storage.PutPending(
				ctx,
				candidate.request.AttemptID,
				key,
				strings.NewReader(candidate.content[key]),
			)
			if err != nil {
				t.Fatal(err)
			}
			candidate.objects = append(candidate.objects, pending.Object)
		}
		candidate.decision = verifyLifecycleDecision(
			t,
			keys,
			signLifecycleDecision(
				t,
				privateKey,
				candidate.request,
				"decision-concurrent-"+candidate.name,
				candidate.objects,
				testRevocationEpoch,
				lifecycleTestNow,
			),
		)
	}

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for _, candidate := range candidates {
		waitGroup.Add(1)
		go func(candidate *commitCandidate) {
			defer waitGroup.Done()
			<-start
			candidate.commit, candidate.commitErr = storage.CommitAttempt(
				ctx,
				3,
				testRevocationEpoch,
				candidate.decision,
			)
		}(candidate)
	}
	close(start)
	waitGroup.Wait()

	var winner, loser *commitCandidate
	for _, candidate := range candidates {
		switch {
		case candidate.commitErr == nil &&
			candidate.commit.Outcome == "COMMITTED" &&
			candidate.commit.ObjectCount == 2:
			if winner != nil {
				t.Fatal("both concurrent attempts committed")
			}
			winner = candidate
		case errors.Is(candidate.commitErr, ErrCASLost):
			if loser != nil {
				t.Fatal("both concurrent attempts lost CAS")
			}
			loser = candidate
		default:
			t.Fatalf(
				"concurrent %s result = %+v/%v",
				candidate.name,
				candidate.commit,
				candidate.commitErr,
			)
		}
	}
	if winner == nil || loser == nil {
		t.Fatalf("concurrent winner/loser = %+v/%+v", winner, loser)
	}
	assertRowCount(t, storage.cache.database, "commit_decisions", 1)
	assertRowCount(t, storage.cache.database, "committed_objects", 2)
	assertRowCount(t, storage.cache.database, "pending_objects", 0)
	assertRowCount(t, storage.control.database, "decision_audit_index", 1)

	for key, want := range winner.content {
		file, _, err := storage.OpenCommitted(
			ctx,
			winner.request.Repository.Tenant,
			winner.request.NamespaceGeneration,
			key,
		)
		if err != nil {
			t.Fatalf("open winning %s: %v", key, err)
		}
		actual, readErr := os.ReadFile(file.Name())
		closeErr := file.Close()
		if readErr != nil ||
			closeErr != nil ||
			string(actual) != want {
			t.Fatalf(
				"winning %s = %q read=%v close=%v",
				key,
				actual,
				readErr,
				closeErr,
			)
		}
	}
	loserStatus, err := storage.AttemptStatus(
		ctx,
		loser.request.AttemptID,
	)
	if err != nil ||
		loserStatus.State != AttemptAborted ||
		loserStatus.AbortReason != "CAS_LOST" ||
		loserStatus.PendingObjectCount != 0 {
		t.Fatalf("concurrent loser status = %+v/%v", loserStatus, err)
	}
	if file, _, err := storage.OpenCommitted(
		ctx,
		loser.request.Repository.Tenant,
		loser.request.NamespaceGeneration,
		"only-"+loser.name,
	); !errors.Is(err, ErrCacheMiss) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("loser exclusive key became visible: %v", err)
	}
	report, err := storage.Reconcile(ctx)
	if err != nil || report.DeletedOrphanBlobs != len(loser.objects) {
		t.Fatalf("concurrent CAS reconcile = %+v/%v", report, err)
	}
	for _, object := range loser.objects {
		assertBlobAbsent(t, storage, object.Checksum)
	}
}

func TestDecisionRejectionAbortsIncompleteCoverageAndRejectsAuthority(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	request := lifecycleAttemptRequest(
		"attempt-incomplete",
		"start-incomplete",
		"owner-one",
		2,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	first, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"first-key",
		strings.NewReader("first"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"second-key",
		strings.NewReader("second"),
	); err != nil {
		t.Fatal(err)
	}
	canonical := signLifecycleDecision(
		t,
		privateKey,
		request,
		"decision-incomplete",
		[]CommitObject{first.Object},
		testRevocationEpoch,
		lifecycleTestNow,
	)
	verified := verifyLifecycleDecision(
		t,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		canonical,
	)
	if _, err := storage.CommitAttempt(
		ctx,
		3,
		testRevocationEpoch,
		verified,
	); !errors.Is(err, ErrCommitRejected) {
		t.Fatalf("incomplete commit = %v", err)
	}
	if _, err := storage.CommitAttempt(
		ctx,
		3,
		testRevocationEpoch,
		verified,
	); !errors.Is(err, ErrCommitRejected) {
		t.Fatalf("incomplete commit replay = %v", err)
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil ||
		status.State != AttemptAborted ||
		status.PendingObjectCount != 0 {
		t.Fatalf("incomplete status = %+v/%v", status, err)
	}
	assertRowCount(t, storage.cache.database, "commit_decisions", 0)
	assertRowCount(t, storage.cache.database, "committed_objects", 0)

	t.Run("invalid signature", func(t *testing.T) {
		var mutatedDecision CommitDecision
		if err := json.Unmarshal(canonical, &mutatedDecision); err != nil {
			t.Fatal(err)
		}
		signature := []byte(mutatedDecision.Authentication.Signature)
		if signature[0] == 'A' {
			signature[0] = 'B'
		} else {
			signature[0] = 'A'
		}
		mutatedDecision.Authentication.Signature = string(signature)
		mutated := canonicalizeTestValue(t, mutatedDecision)
		if _, err := VerifyCommitDecision(
			ctx,
			mutated,
			map[string]ed25519.PublicKey{
				testDecisionKeyID: publicKey,
			},
			testRevocationEpoch,
			lifecycleTestNow,
		); !errors.Is(err, ErrCommitRejected) {
			t.Fatalf("invalid signature = %v", err)
		}
	})
	t.Run("stale epoch", func(t *testing.T) {
		if _, err := VerifyCommitDecision(
			ctx,
			canonical,
			map[string]ed25519.PublicKey{
				testDecisionKeyID: publicKey,
			},
			testRevocationEpoch+1,
			lifecycleTestNow,
		); !errors.Is(err, ErrCommitRejected) {
			t.Fatalf("stale epoch = %v", err)
		}
	})
	t.Run("noncanonical", func(t *testing.T) {
		noncanonical := append([]byte(" "), canonical...)
		if _, err := VerifyCommitDecision(
			ctx,
			noncanonical,
			map[string]ed25519.PublicKey{
				testDecisionKeyID: publicKey,
			},
			testRevocationEpoch,
			lifecycleTestNow,
		); !errors.Is(err, ErrCommitRejected) {
			t.Fatalf("noncanonical decision = %v", err)
		}
	})
}

func TestCommitRechecksCurrentRevocationEpochAtTransaction(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	request := lifecycleAttemptRequest(
		"attempt-epoch-race",
		"start-epoch-race",
		"owner-one",
		13,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"epoch-key",
		strings.NewReader("epoch bytes"),
	)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifyLifecycleDecision(
		t,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		signLifecycleDecision(
			t,
			privateKey,
			request,
			"decision-epoch-race",
			[]CommitObject{pending.Object},
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	if _, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch+1,
		verified,
	); !errors.Is(err, ErrCommitRejected) {
		t.Fatalf("advanced epoch commit = %v", err)
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil ||
		status.State != AttemptAborted ||
		status.AbortReason != "POLICY_CHANGED" {
		t.Fatalf("advanced epoch status = %+v/%v", status, err)
	}
	if file, _, err := storage.OpenCommitted(
		ctx,
		request.Repository.Tenant,
		request.NamespaceGeneration,
		pending.Object.Key,
	); !errors.Is(err, ErrCacheMiss) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("advanced epoch visibility = %v", err)
	}
}

func TestTransactionRollbackAndControlAuditRepair(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	request := lifecycleAttemptRequest(
		"attempt-fault",
		"start-fault",
		"owner-one",
		4,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	var objects []CommitObject
	for _, key := range []string{"fault-a", "fault-b", "fault-c"} {
		pending, err := storage.PutPending(
			ctx,
			request.AttemptID,
			key,
			strings.NewReader(key+" object"),
		)
		if err != nil {
			t.Fatal(err)
		}
		objects = append(objects, pending.Object)
	}
	verified := verifyLifecycleDecision(
		t,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		signLifecycleDecision(
			t,
			privateKey,
			request,
			"decision-fault",
			objects,
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	fault := errors.New("injected before cache commit")
	storage.testHooks.beforeCacheCommit = func() error { return fault }
	if _, err := storage.CommitAttempt(
		ctx,
		4,
		testRevocationEpoch,
		verified,
	); !errors.Is(err, fault) {
		t.Fatalf("transaction fault = %v", err)
	}
	storage.testHooks.beforeCacheCommit = nil
	assertRowCount(t, storage.cache.database, "commit_decisions", 0)
	assertRowCount(t, storage.cache.database, "committed_objects", 0)
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil ||
		status.State != AttemptPending ||
		status.PendingObjectCount != 3 {
		t.Fatalf("rolled-back status = %+v/%v", status, err)
	}

	controlFault := errors.New("injected control index failure")
	storage.testHooks.beforeControlIndex = func() error { return controlFault }
	commit, err := storage.CommitAttempt(
		ctx,
		4,
		testRevocationEpoch,
		verified,
	)
	if err != nil ||
		commit.Outcome != "COMMITTED" ||
		commit.ObjectCount != 3 ||
		commit.AuditIndexed ||
		!commit.RequiresReconcile {
		t.Fatalf("cache commit/control fault = %+v/%v", commit, err)
	}
	storage.testHooks.beforeControlIndex = nil
	assertRowCount(t, storage.cache.database, "committed_objects", 3)
	assertRowCount(t, storage.control.database, "decision_audit_index", 0)
	report, err := storage.Reconcile(ctx)
	if err != nil || report.RepairedAuditRows != 1 {
		t.Fatalf("audit repair = %+v/%v", report, err)
	}
	assertRowCount(t, storage.control.database, "decision_audit_index", 1)
}

func TestReconcileInvalidatesWholeDecisionOnCorruptionOrMissingBlob(
	t *testing.T,
) {
	for _, testCase := range []struct {
		name   string
		mutate func(*testing.T, string)
		reason string
	}{
		{
			name: "corrupt",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(
					path,
					[]byte("corrupt"),
					0o600,
				); err != nil {
					t.Fatal(err)
				}
			},
			reason: "CORRUPT",
		},
		{
			name: "missing",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			},
			reason: "MISSING",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			storage := openLifecycleTestStorage(t)
			publicKey, privateKey, err := ed25519.GenerateKey(nil)
			if err != nil {
				t.Fatal(err)
			}
			request := lifecycleAttemptRequest(
				"attempt-"+testCase.name,
				"start-"+testCase.name,
				"owner-one",
				5,
			)
			if _, _, err := storage.StartAttempt(ctx, request); err != nil {
				t.Fatal(err)
			}
			var objects []CommitObject
			for _, key := range []string{"first", "second"} {
				pending, err := storage.PutPending(
					ctx,
					request.AttemptID,
					key,
					strings.NewReader(key+" bytes"),
				)
				if err != nil {
					t.Fatal(err)
				}
				objects = append(objects, pending.Object)
			}
			verified := verifyLifecycleDecision(
				t,
				map[string]ed25519.PublicKey{
					testDecisionKeyID: publicKey,
				},
				signLifecycleDecision(
					t,
					privateKey,
					request,
					"decision-"+testCase.name,
					objects,
					testRevocationEpoch,
					lifecycleTestNow,
				),
			)
			if _, err := storage.CommitAttempt(
				ctx,
				3,
				testRevocationEpoch,
				verified,
			); err != nil {
				t.Fatal(err)
			}
			path, err := storage.blobs.pathForDigest(
				objects[0].Checksum,
				false,
			)
			if err != nil {
				t.Fatal(err)
			}
			testCase.mutate(t, path)
			report, err := storage.Reconcile(ctx)
			if err != nil ||
				report.InvalidatedDecisions != 1 ||
				report.QuarantinedBlobs != 1 ||
				report.DeletedOrphanBlobs != 1 {
				t.Fatalf("reconcile = %+v/%v", report, err)
			}
			for _, object := range objects {
				file, _, err := storage.OpenCommitted(
					ctx,
					request.Repository.Tenant,
					request.NamespaceGeneration,
					object.Key,
				)
				if file != nil {
					_ = file.Close()
				}
				if !errors.Is(err, ErrCacheMiss) {
					t.Fatalf("%s remained visible: %v", object.Key, err)
				}
			}
			assertRowCount(t, storage.cache.database, "commit_decisions", 0)
			assertRowCount(t, storage.cache.database, "committed_objects", 0)
			var reason string
			if err := storage.cache.database.QueryRow(
				"SELECT reason FROM quarantine_records",
			).Scan(&reason); err != nil || reason != testCase.reason {
				t.Fatalf("quarantine reason = %q/%v", reason, err)
			}
			entries, err := os.ReadDir(storage.Layout().Quarantine)
			if err != nil {
				t.Fatal(err)
			}
			wantFiles := 0
			if testCase.reason == "CORRUPT" {
				wantFiles = 1
			}
			if len(entries) != wantFiles {
				t.Fatalf("quarantine files = %v, want %d", entries, wantFiles)
			}
		})
	}
}

func TestExpiredAttemptIsAbortedBeforeOrphanCollection(t *testing.T) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	request := lifecycleAttemptRequest(
		"attempt-expired",
		"start-expired",
		"owner-one",
		6,
	)
	request.LeaseExpiresAt = lifecycleTestNow.Add(5 * time.Minute)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"expiry-key",
		strings.NewReader("expires"),
	)
	if err != nil {
		t.Fatal(err)
	}
	storage.clock = func() time.Time {
		return lifecycleTestNow.Add(6 * time.Minute)
	}
	report, err := storage.Reconcile(ctx)
	if err != nil ||
		report.ExpiredAttempts != 1 ||
		report.DeletedOrphanBlobs != 1 {
		t.Fatalf("expiry reconcile = %+v/%v", report, err)
	}
	status, err := storage.AttemptStatus(ctx, request.AttemptID)
	if err != nil ||
		status.State != AttemptAborted ||
		status.AbortReason != "LEASE_EXPIRED" {
		t.Fatalf("expired status = %+v/%v", status, err)
	}
	assertBlobAbsent(t, storage, pending.Object.Checksum)
}

func TestReconcileCannotCollectBlobBetweenPendingPublishAndMetadata(
	t *testing.T,
) {
	ctx := context.Background()
	storage := openLifecycleTestStorage(t)
	request := lifecycleAttemptRequest(
		"attempt-publish-race",
		"start-publish-race",
		"owner-one",
		7,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	published := make(chan struct{})
	release := make(chan struct{})
	storage.testHooks.afterPendingBlob = func() {
		close(published)
		<-release
	}
	type putResult struct {
		result PendingObjectResult
		err    error
	}
	putDone := make(chan putResult, 1)
	go func() {
		result, err := storage.PutPending(
			ctx,
			request.AttemptID,
			"race-key",
			strings.NewReader("race bytes"),
		)
		putDone <- putResult{result: result, err: err}
	}()
	<-published
	type reconcileResult struct {
		report ReconciliationReport
		err    error
	}
	reconcileDone := make(chan reconcileResult, 1)
	go func() {
		report, err := storage.Reconcile(ctx)
		reconcileDone <- reconcileResult{report: report, err: err}
	}()
	select {
	case result := <-reconcileDone:
		t.Fatalf("reconcile crossed pending publication: %+v", result)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	storage.testHooks.afterPendingBlob = nil
	result := <-putDone
	if result.err != nil || !result.result.ObjectAdded {
		t.Fatalf("pending publish = %+v/%v", result.result, result.err)
	}
	reconciled := <-reconcileDone
	if reconciled.err != nil ||
		reconciled.report.DeletedOrphanBlobs != 0 {
		t.Fatalf(
			"post-publish reconcile = %+v/%v",
			reconciled.report,
			reconciled.err,
		)
	}
	file, err := storage.Blobs().OpenVerified(
		ctx,
		Blob{
			Digest: result.result.Object.Checksum,
			Size:   result.result.Object.SizeBytes,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
}

func TestStartupReconciliationRepairsAuditAndDeletesOrphans(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage, err := Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	storage.clock = func() time.Time { return lifecycleTestNow }
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	request := lifecycleAttemptRequest(
		"attempt-startup-repair",
		"start-startup-repair",
		"owner-one",
		8,
	)
	if _, _, err := storage.StartAttempt(ctx, request); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.PutPending(
		ctx,
		request.AttemptID,
		"committed-key",
		strings.NewReader("committed"),
	)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifyLifecycleDecision(
		t,
		map[string]ed25519.PublicKey{testDecisionKeyID: publicKey},
		signLifecycleDecision(
			t,
			privateKey,
			request,
			"decision-startup-repair",
			[]CommitObject{pending.Object},
			testRevocationEpoch,
			lifecycleTestNow,
		),
	)
	storage.testHooks.beforeControlIndex = func() error {
		return errors.New("control unavailable")
	}
	commit, err := storage.CommitAttempt(
		ctx,
		2,
		testRevocationEpoch,
		verified,
	)
	if err != nil || !commit.RequiresReconcile {
		t.Fatalf("commit needing repair = %+v/%v", commit, err)
	}
	storage.testHooks.beforeControlIndex = nil
	orphan, _, err := storage.Blobs().Put(
		ctx,
		strings.NewReader("orphan"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(ctx, root)
	if err != nil {
		t.Fatalf("startup reconcile: %v", err)
	}
	defer reopened.Close()
	assertRowCount(
		t,
		reopened.control.database,
		"decision_audit_index",
		1,
	)
	assertBlobAbsent(t, reopened, orphan.Digest)
	var (
		repaired int
		deleted  int
	)
	if err := reopened.control.database.QueryRow(
		`SELECT repaired_audit_rows, deleted_orphan_blobs
FROM reconciliation_runs
ORDER BY run_id DESC
LIMIT 1`,
	).Scan(&repaired, &deleted); err != nil ||
		repaired != 1 ||
		deleted != 1 {
		t.Fatalf(
			"startup reconciliation evidence = repaired=%d deleted=%d err=%v",
			repaired,
			deleted,
			err,
		)
	}
	file, _, err := reopened.OpenCommitted(
		ctx,
		request.Repository.Tenant,
		request.NamespaceGeneration,
		pending.Object.Key,
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
}

func TestSchemaVersionOneUpgradesTransactionallyToCurrent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(root, "cache.sqlite")
	controlPath := filepath.Join(root, "control.sqlite")
	createVersionOneDatabase(
		t,
		cachePath,
		cacheMetadataDefinition(cachePath),
	)
	createVersionOneDatabase(
		t,
		controlPath,
		controlMetadataDefinition(controlPath),
	)
	storage, err := Open(context.Background(), root)
	if err != nil {
		t.Fatalf("upgrade storage: %v", err)
	}
	defer storage.Close()
	for name, metadata := range map[string]*sqliteMetadata{
		"cache":   storage.cache,
		"control": storage.control,
	} {
		var version int
		if err := metadata.database.QueryRow(
			"PRAGMA user_version",
		).Scan(&version); err != nil || version != SchemaVersion {
			t.Fatalf("%s version = %d/%v", name, version, err)
		}
		assertRowCount(
			t,
			metadata.database,
			"schema_migrations",
			SchemaVersion,
		)
	}
	assertSchemaObjects(
		t,
		storage.cache.database,
		cacheMetadataDefinition(cachePath).objects,
	)
	assertSchemaObjects(
		t,
		storage.control.database,
		controlMetadataDefinition(controlPath).objects,
	)
}

func openLifecycleTestStorage(t *testing.T) *Storage {
	t.Helper()
	storage, err := Open(
		context.Background(),
		filepath.Join(t.TempDir(), "shared"),
	)
	if err != nil {
		t.Fatal(err)
	}
	storage.clock = func() time.Time { return lifecycleTestNow }
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Errorf("close storage: %v", err)
		}
	})
	return storage
}

func TestSharedRejectsGenerationsPredatingManagedDeletion(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deployment-data")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create managed root: %v", err)
	}
	marker, err := json.Marshal(map[string]string{
		"deploymentRoot": filepath.Join(filepath.Dir(root), "deployment"),
		"schemaVersion":  "buildopt.dev/deployment-data/v1",
	})
	if err != nil {
		t.Fatalf("encode managed root marker: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".buildopt-deployment-data.json"),
		append(marker, '\n'),
		0o600,
	); err != nil {
		t.Fatalf("write managed root marker: %v", err)
	}
	key := make([]byte, datalifecycle.RedactionKeyBytes)
	for index := range key {
		key[index] = byte(index + 1)
	}
	if _, err := datalifecycle.DeleteManagedData(
		context.Background(),
		datalifecycle.DeletionRequest{
			DataRoot:                 root,
			DeletionID:               "shared-generation-floor",
			Tenant:                   "tenant-test",
			Repository:               "repository-test",
			TrustDomain:              "trust-test",
			NextNamespaceGeneration:  8,
			NextL1SecurityGeneration: 12,
			TokenKey:                 key,
			TokenKeyVersion:          "shared-floor-v1",
			RequestedAt:              lifecycleTestNow,
		},
	); err != nil {
		t.Fatalf("establish managed deletion boundary: %v", err)
	}
	storage, err := Open(context.Background(), filepath.Join(root, "shared"))
	if err != nil {
		t.Fatalf("open Shared after completed deletion: %v", err)
	}
	t.Cleanup(func() {
		_ = storage.Close()
	})
	storage.clock = func() time.Time { return lifecycleTestNow.Add(time.Hour) }
	if storage.minimumNamespaceGeneration != 8 {
		t.Fatalf(
			"minimum namespace generation = %d",
			storage.minimumNamespaceGeneration,
		)
	}
	request := lifecycleAttemptRequest(
		"attempt-before-deletion",
		"request-before-deletion",
		"owner-before-deletion",
		7,
	)
	request.LeaseExpiresAt = storage.now().Add(time.Hour)
	if _, _, err := storage.StartAttempt(
		context.Background(),
		request,
	); err == nil || !strings.Contains(err.Error(), "predates managed deletion") {
		t.Fatalf("stale attempt error = %v", err)
	}
	if _, err := storage.IssueBetaToken(
		context.Background(),
		BetaTokenIssueRequest{
			Scope: BetaTokenScope{
				Tenant:              "tenant-test",
				Repository:          "repository-test",
				TrustDomain:         "trust-test",
				Namespace:           "stable",
				NamespaceGeneration: 7,
				Plane:               BetaTokenPlaneStable,
			},
			Access:    BetaTokenRead,
			ExpiresAt: storage.now().Add(time.Hour),
		},
		storage.now(),
	); err == nil || !strings.Contains(err.Error(), "predates managed deletion") {
		t.Fatalf("stale token error = %v", err)
	}
}

func lifecycleAttemptRequest(
	attemptID string,
	requestID string,
	ownerID string,
	generation int64,
) StartAttemptRequest {
	return StartAttemptRequest{
		RequestID: requestID,
		AttemptID: attemptID,
		Repository: RepositoryIdentity{
			Tenant:      "tenant-test",
			Repository:  "repository-test",
			TrustDomain: "trust-test",
		},
		NamespaceGeneration:       generation,
		SourceRevision:            "abcdef0123456789",
		SourceStateDigest:         "hmac-sha256:" + strings.Repeat("1", 64),
		PolicyDigest:              "sha256:" + strings.Repeat("2", 64),
		ConfigurationPolicyDigest: "sha256:" + strings.Repeat("3", 64),
		CacheContractDigest:       "sha256:" + strings.Repeat("4", 64),
		OwnerID:                   ownerID,
		LeaseID:                   "lease-" + attemptID,
		LeaseExpiresAt:            lifecycleTestNow.Add(time.Hour),
	}
}

func signLifecycleDecision(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	request StartAttemptRequest,
	decisionID string,
	objects []CommitObject,
	revocationEpoch int64,
	now time.Time,
) []byte {
	t.Helper()
	decision := CommitDecision{
		SchemaVersion:             "1.0",
		RecordType:                "COMMIT_DECISION",
		ContractVersion:           "buildopt-cache-commit/v1",
		DecisionID:                decisionID,
		AttemptID:                 request.AttemptID,
		Repository:                request.Repository,
		SourceRevision:            request.SourceRevision,
		SourceStateDigest:         request.SourceStateDigest,
		Objects:                   objects,
		PolicyDigest:              request.PolicyDigest,
		ConfigurationPolicyDigest: request.ConfigurationPolicyDigest,
		CacheContractDigest:       request.CacheContractDigest,
		TestOptimizationGrant: TestOptimizationGrant{
			State:  "NOT_REQUIRED",
			Reason: "NO_TEST_OUTPUTS",
		},
		RevocationEpoch: revocationEpoch,
		Validation: CommitValidation{
			Status: "NOT_REQUIRED",
			Reason: "ALLOWLISTED_DIRECT_ACTION",
		},
		IssuedAt:  now.Format(time.RFC3339Nano),
		ExpiresAt: now.Add(30 * time.Minute).Format(time.RFC3339Nano),
		Authentication: CommitAuthentication{
			Algorithm: "Ed25519",
			KeyID:     testDecisionKeyID,
		},
	}
	provisional := canonicalizeTestValue(t, decision)
	digest, err := commitDecisionDigest(provisional)
	if err != nil {
		t.Fatal(err)
	}
	decision.DecisionDigest = digest
	decision.Authentication.Signature = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(
			privateKey,
			commitDecisionSignaturePayload(testDecisionKeyID, digest),
		),
	)
	return canonicalizeTestValue(t, decision)
}

func canonicalizeTestValue(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := contractcrypto.CanonicalizeJCS(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func verifyLifecycleDecision(
	t *testing.T,
	keys map[string]ed25519.PublicKey,
	canonical []byte,
) VerifiedCommitDecision {
	t.Helper()
	verified, err := VerifyCommitDecision(
		context.Background(),
		canonical,
		keys,
		testRevocationEpoch,
		lifecycleTestNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	return verified
}

func assertRowCount(
	t *testing.T,
	database *sql.DB,
	table string,
	want int,
) {
	t.Helper()
	var actual int
	if err := database.QueryRow(
		"SELECT count(*) FROM " + table,
	).Scan(&actual); err != nil || actual != want {
		t.Fatalf("%s row count = %d/%v, want %d", table, actual, err, want)
	}
}

func assertBlobAbsent(t *testing.T, storage *Storage, digest string) {
	t.Helper()
	path, err := storage.blobs.pathForDigest(digest, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("blob %s remains: %v", digest, err)
	}
}

func createVersionOneDatabase(
	t *testing.T,
	path string,
	definition metadataDefinition,
) {
	t.Helper()
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", sqliteDataSource(path))
	if err != nil {
		t.Fatal(err)
	}
	migration := definition.migrations[0]
	transaction, err := database.Begin()
	if err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	for _, statement := range migration.statements {
		if _, err := transaction.Exec(statement); err != nil {
			_ = transaction.Rollback()
			_ = database.Close()
			t.Fatal(err)
		}
	}
	if _, err := transaction.Exec(
		`INSERT INTO schema_migrations
    (version, name, checksum, applied_at_unix_ms)
VALUES (?, ?, ?, ?)`,
		migration.version,
		migration.name,
		migrationChecksum(definition.role, migration),
		lifecycleTestNow.UnixMilli(),
	); err != nil {
		_ = transaction.Rollback()
		_ = database.Close()
		t.Fatal(err)
	}
	if _, err := transaction.Exec("PRAGMA user_version = 1"); err != nil {
		_ = transaction.Rollback()
		_ = database.Close()
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
}
