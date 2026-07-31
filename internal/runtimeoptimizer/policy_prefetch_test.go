package runtimeoptimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPolicyPrefetchIsSingleFlightAndReturnsDefensiveCopy(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	key := testPolicyPrefetchKey()
	cache := NewPolicyPrefetchCache()
	release := make(chan struct{})
	var fetches atomic.Int32
	fetch := func(context.Context, PolicyPrefetchKey) (PrefetchedPolicy, error) {
		fetches.Add(1)
		<-release
		return testPrefetchedPolicy(key, now), nil
	}
	verify := func(policy PrefetchedPolicy) error {
		if string(policy.Payload) != "signed-policy" {
			return errors.New("invalid signature")
		}
		return nil
	}
	const callers = 8
	handles := make([]*PolicyPrefetchHandle, callers)
	var wait sync.WaitGroup
	wait.Add(callers)
	for index := range handles {
		go func(index int) {
			defer wait.Done()
			handle, err := cache.Start(context.Background(), key, now, 4, fetch, verify)
			if err != nil {
				t.Errorf("start: %v", err)
				return
			}
			handles[index] = handle
		}(index)
	}
	wait.Wait()
	close(release)
	for _, handle := range handles {
		if err := handle.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if fetches.Load() != 1 {
		t.Fatalf("fetches = %d", fetches.Load())
	}
	policy, ok := cache.Use(key, now, 4)
	if !ok {
		t.Fatal("verified policy was not cached")
	}
	policy.Payload[0] = 'X'
	reused, ok := cache.Use(key, now, 4)
	if !ok || string(reused.Payload) != "signed-policy" {
		t.Fatal("cached payload was mutable")
	}
}

func TestPolicyPrefetchRejectsTamperBindingExpiryAndVerification(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	key := testPolicyPrefetchKey()
	tests := []struct {
		name   string
		mutate func(*PrefetchedPolicy)
		verify PolicyPrefetchVerifier
	}{
		{name: "tamper", mutate: func(policy *PrefetchedPolicy) { policy.Payload = []byte("tampered") }},
		{name: "binding", mutate: func(policy *PrefetchedPolicy) { policy.Key.PipelineClass = "main" }},
		{name: "expired", mutate: func(policy *PrefetchedPolicy) { policy.ExpiresAt = now }},
		{name: "verification", verify: func(PrefetchedPolicy) error { return errors.New("signature rejected") }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cache := NewPolicyPrefetchCache()
			policy := testPrefetchedPolicy(key, now)
			if testCase.mutate != nil {
				testCase.mutate(&policy)
			}
			verify := testCase.verify
			if verify == nil {
				verify = func(PrefetchedPolicy) error { return nil }
			}
			handle, err := cache.Start(context.Background(), key, now, 4, func(context.Context, PolicyPrefetchKey) (PrefetchedPolicy, error) { return policy, nil }, verify)
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.Wait(context.Background()); err == nil {
				t.Fatal("invalid prefetch was accepted")
			}
			if _, ok := cache.Use(key, now, 4); ok {
				t.Fatal("invalid prefetch became usable")
			}
		})
	}
}

func TestPolicyPrefetchExpiresAndHonorsRevocation(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	key := testPolicyPrefetchKey()
	cache := NewPolicyPrefetchCache()
	policy := testPrefetchedPolicy(key, now)
	handle, err := cache.Start(context.Background(), key, now, 4, func(context.Context, PolicyPrefetchKey) (PrefetchedPolicy, error) { return policy, nil }, func(PrefetchedPolicy) error { return nil })
	if err != nil || handle.Wait(context.Background()) != nil {
		t.Fatalf("prefetch = %v/%v", handle, err)
	}
	if _, ok := cache.Use(key, now, 6); ok {
		t.Fatal("revoked prefetched policy was used")
	}
	var refetched atomic.Int32
	handle, err = cache.Start(context.Background(), key, now, 6, func(context.Context, PolicyPrefetchKey) (PrefetchedPolicy, error) {
		refetched.Add(1)
		updated := testPrefetchedPolicy(key, now)
		updated.RevocationEpoch = 6
		return updated, nil
	}, func(PrefetchedPolicy) error { return nil })
	if err != nil || handle.Wait(context.Background()) != nil || refetched.Load() != 1 {
		t.Fatalf("refetch = %v/%v/%d", handle, err, refetched.Load())
	}
	if _, ok := cache.Use(key, policy.ExpiresAt, 6); ok {
		t.Fatal("expired prefetched policy was used")
	}
}

func TestPolicyPrefetchWaitHasIndependentDeadline(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	key := testPolicyPrefetchKey()
	cache := NewPolicyPrefetchCache()
	release := make(chan struct{})
	handle, err := cache.Start(context.Background(), key, now, 0, func(context.Context, PolicyPrefetchKey) (PrefetchedPolicy, error) {
		<-release
		return testPrefetchedPolicy(key, now), nil
	}, func(PrefetchedPolicy) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := handle.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("wait = %v", err)
	}
	close(release)
	if err := handle.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func testPolicyPrefetchKey() PolicyPrefetchKey {
	return PolicyPrefetchKey{RepositoryID: "repository-1", SourceRevision: "sha256:" + repeat("1", 64), PipelineClass: "pull-request", CompatibilityClass: "gradle-9"}
}

func testPrefetchedPolicy(key PolicyPrefetchKey, now time.Time) PrefetchedPolicy {
	payload := []byte("signed-policy")
	digest := sha256.Sum256(payload)
	return PrefetchedPolicy{
		Key: key, Payload: payload, PayloadDigest: "sha256:" + hex.EncodeToString(digest[:]),
		PolicyDigest: "sha256:" + repeat("2", 64), AuthorityDigest: "sha256:" + repeat("3", 64),
		RevocationEpoch: 5, ExpiresAt: now.Add(time.Hour),
	}
}
