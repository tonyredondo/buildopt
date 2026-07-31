package runtimeoptimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"sync"
	"time"
)

const maximumPrefetchedPolicyBytes = 1 << 20

// PolicyPrefetchKey binds a prefetched policy to one future invocation class.
type PolicyPrefetchKey struct {
	RepositoryID       string
	SourceRevision     string
	PipelineClass      string
	CompatibilityClass string
}

// PrefetchedPolicy is untrusted until the caller-supplied verifier accepts it.
type PrefetchedPolicy struct {
	Key             PolicyPrefetchKey
	Payload         []byte
	PayloadDigest   string
	PolicyDigest    string
	AuthorityDigest string
	RevocationEpoch int64
	ExpiresAt       time.Time
}

// PolicyPrefetchFetcher retrieves one candidate without granting authority.
type PolicyPrefetchFetcher func(context.Context, PolicyPrefetchKey) (PrefetchedPolicy, error)

// PolicyPrefetchVerifier authenticates the complete candidate before caching.
type PolicyPrefetchVerifier func(PrefetchedPolicy) error

type policyPrefetchCall struct {
	done chan struct{}
	err  error
}

// PolicyPrefetchHandle allows bounded waiting without coupling callers to fetch work.
type PolicyPrefetchHandle struct {
	call *policyPrefetchCall
}

// PolicyPrefetchCache is an optional in-memory latency cache with single-flight fetches.
type PolicyPrefetchCache struct {
	mutex    sync.Mutex
	calls    map[string]*policyPrefetchCall
	policies map[string]PrefetchedPolicy
}

// NewPolicyPrefetchCache creates an empty non-authoritative cache.
func NewPolicyPrefetchCache() *PolicyPrefetchCache {
	return &PolicyPrefetchCache{calls: map[string]*policyPrefetchCall{}, policies: map[string]PrefetchedPolicy{}}
}

// Start launches or joins one exact-key prefetch operation.
func (cache *PolicyPrefetchCache) Start(ctx context.Context, key PolicyPrefetchKey, now time.Time, minimumRevocationEpoch int64, fetch PolicyPrefetchFetcher, verify PolicyPrefetchVerifier) (*PolicyPrefetchHandle, error) {
	if cache == nil || ctx == nil || fetch == nil || verify == nil || !validPolicyPrefetchKey(key) || now.IsZero() || minimumRevocationEpoch < 0 {
		return nil, errors.New("start policy prefetch: invalid request")
	}
	cacheKey := policyPrefetchKeyString(key)
	cache.mutex.Lock()
	if policy, ok := cache.policies[cacheKey]; ok && policy.ExpiresAt.After(now) && policy.RevocationEpoch >= minimumRevocationEpoch {
		call := &policyPrefetchCall{done: make(chan struct{})}
		close(call.done)
		cache.mutex.Unlock()
		return &PolicyPrefetchHandle{call: call}, nil
	}
	delete(cache.policies, cacheKey)
	if call, ok := cache.calls[cacheKey]; ok {
		cache.mutex.Unlock()
		return &PolicyPrefetchHandle{call: call}, nil
	}
	call := &policyPrefetchCall{done: make(chan struct{})}
	cache.calls[cacheKey] = call
	cache.mutex.Unlock()

	go cache.fetch(ctx, key, cacheKey, now, fetch, verify, call)
	return &PolicyPrefetchHandle{call: call}, nil
}

// Wait waits for the joined prefetch or the caller's own deadline.
func (handle *PolicyPrefetchHandle) Wait(ctx context.Context) error {
	if handle == nil || handle.call == nil || ctx == nil {
		return errors.New("wait policy prefetch: invalid handle")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-handle.call.done:
		return handle.call.err
	}
}

// Use returns a defensive copy only for a live, non-revoked exact binding.
func (cache *PolicyPrefetchCache) Use(key PolicyPrefetchKey, now time.Time, minimumRevocationEpoch int64) (PrefetchedPolicy, bool) {
	if cache == nil || !validPolicyPrefetchKey(key) || now.IsZero() || minimumRevocationEpoch < 0 {
		return PrefetchedPolicy{}, false
	}
	cacheKey := policyPrefetchKeyString(key)
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	policy, ok := cache.policies[cacheKey]
	if !ok || !policy.ExpiresAt.After(now) || policy.RevocationEpoch < minimumRevocationEpoch {
		delete(cache.policies, cacheKey)
		return PrefetchedPolicy{}, false
	}
	return clonePrefetchedPolicy(policy), true
}

func (cache *PolicyPrefetchCache) fetch(ctx context.Context, key PolicyPrefetchKey, cacheKey string, now time.Time, fetch PolicyPrefetchFetcher, verify PolicyPrefetchVerifier, call *policyPrefetchCall) {
	policy, err := fetch(ctx, key)
	if err == nil {
		err = validatePrefetchedPolicy(policy, key, now)
	}
	if err == nil {
		err = verify(clonePrefetchedPolicy(policy))
	}
	cache.mutex.Lock()
	if err == nil {
		cache.policies[cacheKey] = clonePrefetchedPolicy(policy)
	}
	call.err = err
	delete(cache.calls, cacheKey)
	close(call.done)
	cache.mutex.Unlock()
}

func validatePrefetchedPolicy(policy PrefetchedPolicy, key PolicyPrefetchKey, now time.Time) error {
	if policy.Key != key || len(policy.Payload) == 0 || len(policy.Payload) > maximumPrefetchedPolicyBytes ||
		!validDigest(policy.PayloadDigest) || !validDigest(policy.PolicyDigest) || !validDigest(policy.AuthorityDigest) ||
		policy.RevocationEpoch < 0 || !policy.ExpiresAt.After(now) {
		return errors.New("policy prefetch: invalid or stale candidate")
	}
	digest := sha256.Sum256(policy.Payload)
	if policy.PayloadDigest != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("policy prefetch: payload digest mismatch")
	}
	return nil
}

func validPolicyPrefetchKey(key PolicyPrefetchKey) bool {
	return identifierPattern.MatchString(key.RepositoryID) && validDigest(key.SourceRevision) &&
		identifierPattern.MatchString(key.PipelineClass) && identifierPattern.MatchString(key.CompatibilityClass)
}

func policyPrefetchKeyString(key PolicyPrefetchKey) string {
	return key.RepositoryID + "\x00" + key.SourceRevision + "\x00" + key.PipelineClass + "\x00" + key.CompatibilityClass
}

func clonePrefetchedPolicy(policy PrefetchedPolicy) PrefetchedPolicy {
	policy.Payload = slices.Clone(policy.Payload)
	return policy
}
