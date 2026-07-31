# Full relevant validation gate v1

This POC contract closes C4-006 and C4-G02 by composing the completed C4-005
candidate/control result with the generated Test Optimization client. A local
result other than PASSED blocks before any remote contact. Policy must provide
a digest-bound REQUIRED or NOT_REQUIRED applicability decision; NOT_REQUIRED
passes locally with no client, artifact, or network access.

REQUIRED submits FULL_RELEVANT_VALIDATION with the exact repository, revision,
source-state, action, policy, deadline, and candidate artifact set. The
content-addressed set must match the local candidate result. Artifact retrieval
is limited to opaque customer-channel identifiers or HTTPS locators; caller
filesystem paths are rejected before transport. The idempotency key is exactly
actionId:artifactSetDigest.

The gate adapts the generated single-attempt client and owns one exact retry
with unchanged body, idempotency key, and original deadline. A delayed result
is polled at most 128 times with deterministic jitter, a five-second delay cap,
and a deadline no more than 24 hours from submission. Every response must echo
the request ID and a pending operation cannot change identity.

Final results use strict JSON, JCS, a pinned Ed25519 key, and the domain
`test-optimization/v1\nTEST_VALIDATION_RESULT\n<payloadDigest>\n<keyId>`.
Identity, context, validity window, policy, and every validated artifact are
checked exactly. Only a trusted, current PASSED result allows the action;
FAILED, INCONCLUSIVE, transport failure, timeout, incompatible data, or any
binding failure blocks it.

Run `./dev/check-full-relevant-validation`.

This contract refines the RFC Test Optimization validation requirement and the
Patch Autopilot correctness gate. The checker composes the Test Optimization
contract/client suite with C4-005 and the production Java gate. Focused cases cover delayed polling, exact retry,
signed pass/fail/inconclusive results, untrusted and expired results, artifact
rebinding, response and operation identity drift, deadline exhaustion, caller
path rejection, local failure without contact, and explicit NOT_REQUIRED
without contact. It performs no customer build or remote mutation.
