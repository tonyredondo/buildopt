package betabenchmark

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const benchmarkDecisionKeyID = "beta-benchmark-v1"

func benchmarkSigningKey(manifestDigest string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte(
		"buildopt-beta-benchmark-smoke/v1\x00" + manifestDigest,
	))
	return ed25519.NewKeyFromSeed(seed[:])
}

func verifyBenchmarkDecision(
	ctx context.Context,
	privateKey ed25519.PrivateKey,
	request sharedcache.StartAttemptRequest,
	status sharedcache.AttemptStatus,
	objects []sharedcache.CommitObject,
	now time.Time,
) (sharedcache.VerifiedCommitDecision, error) {
	return verifyBenchmarkDecisionWithAuthority(
		ctx,
		privateKey,
		request,
		status,
		objects,
		1,
		sharedcache.TestOptimizationGrant{
			State:  "NOT_REQUIRED",
			Reason: "NO_TEST_OUTPUTS",
		},
		now,
	)
}

func verifyBenchmarkDecisionWithAuthority(
	ctx context.Context,
	privateKey ed25519.PrivateKey,
	request sharedcache.StartAttemptRequest,
	status sharedcache.AttemptStatus,
	objects []sharedcache.CommitObject,
	revocationEpoch int64,
	grant sharedcache.TestOptimizationGrant,
	now time.Time,
) (sharedcache.VerifiedCommitDecision, error) {
	decision := sharedcache.CommitDecision{
		SchemaVersion:             "1.0",
		RecordType:                "COMMIT_DECISION",
		ContractVersion:           "buildopt-cache-commit/v1",
		DecisionID:                "decision-" + request.AttemptID,
		AttemptID:                 request.AttemptID,
		Repository:                request.Repository,
		SourceRevision:            request.SourceRevision,
		SourceStateDigest:         request.SourceStateDigest,
		Objects:                   objects,
		PolicyDigest:              request.PolicyDigest,
		ConfigurationPolicyDigest: request.ConfigurationPolicyDigest,
		CacheContractDigest:       request.CacheContractDigest,
		TestOptimizationGrant:     grant,
		RevocationEpoch:           revocationEpoch,
		Validation: sharedcache.CommitValidation{
			Status: "NOT_REQUIRED",
			Reason: "ALLOWLISTED_DIRECT_ACTION",
		},
		IssuedAt: now.UTC().Format(time.RFC3339Nano),
		ExpiresAt: minTime(
			now.Add(30*time.Minute),
			status.LeaseExpiresAt,
		).UTC().Format(time.RFC3339Nano),
		Authentication: sharedcache.CommitAuthentication{
			Algorithm: "Ed25519",
			KeyID:     benchmarkDecisionKeyID,
		},
	}
	provisional, err := canonicalize(decision)
	if err != nil {
		return sharedcache.VerifiedCommitDecision{}, err
	}
	digest, err := benchmarkDecisionDigest(provisional)
	if err != nil {
		return sharedcache.VerifiedCommitDecision{}, err
	}
	decision.DecisionDigest = digest
	decision.Authentication.Signature = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(
			privateKey,
			[]byte(
				"buildopt-cache-commit/v1\x00"+
					benchmarkDecisionKeyID+"\x00"+digest,
			),
		),
	)
	canonical, err := canonicalize(decision)
	if err != nil {
		return sharedcache.VerifiedCommitDecision{}, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return sharedcache.VerifyCommitDecision(
		ctx,
		canonical,
		map[string]ed25519.PublicKey{benchmarkDecisionKeyID: publicKey},
		revocationEpoch,
		now.UTC(),
	)
}

func benchmarkDecisionDigest(canonical []byte) (string, error) {
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return "", err
	}
	delete(document, "decisionDigest")
	authentication := document["authentication"].(map[string]any)
	delete(authentication, "signature")
	payload, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	payload, err = contractcrypto.CanonicalizeJCS(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func canonicalize(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return contractcrypto.CanonicalizeJCS(encoded)
}

func minTime(first time.Time, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}
