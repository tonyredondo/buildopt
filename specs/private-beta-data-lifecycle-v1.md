# Private-beta managed data lifecycle v1

This specification materializes A1-004 and closes A1-G05 for one
PRIVATE_BETA_ISOLATED deployment. It implements the retention, redaction,
diagnostic opt-in, and coordinated-deletion decisions established by
[data-lifecycle-v1.md](./data-lifecycle-v1.md); it does not create a
multi-tenant deletion boundary.

## Export and diagnostic authorization

SUMMARY is the only implicit export profile. It persists one keyed task-set
token. TASKS and EVIDENCE require --authorize-expanded-export and persist
keyed task identifiers only. DIAGNOSTIC additionally requires a future UTC
--diagnostic-until no more than seven days away. Opening a JSONL stream with
an authorization narrower than a previously persisted event fails closed.

Before JSON or JSONL persistence, the exporter replaces repository, trust
domain, and task identities using domain-separated HMAC-SHA-256. Each export
directory owns a generated 32-byte mode-0600 key and persists only its public
rotation identifier in the BUILD_SESSION document. Existing unkeyed
directories are rejected instead of being silently treated as redacted.

## Coordinated deletion

buildopt-server data delete accepts only an absolute marked deployment data
root and a private mode-0600 32-byte tokenization key. The complete marked
root is the deletion unit. Unknown root entries fail closed.

Deletion takes the lifecycle lock exclusively and refuses active Shared or
managed-L1 leases before writing state. It then:

1. writes and syncs a tokenized LOGICALLY_REVOKED tombstone;
2. stages and removes shared, l1, exports, evidence, and spool;
3. records PHYSICAL_COMPLETE with namespace and L1 generation floors.

Exact retries return the original tombstone. Customer-controlled downstream
destinations remain outside the physical boundary and receive only a tokenized
tombstone obligation. A retention hold is accepted only when consent, reason,
and expiry are all explicit; deletion never invents a silent hold.

Shared, token provisioning, managed L1, and BUILD_SESSION export hold the
shared side of the lifecycle lock while open. They reject logical revocation;
after physical completion, Shared authorities/attempts/tokens and managed L1
must meet the persisted generation floors before recreating state.

## Verification

Run:

    ./dev/check-private-beta-data-lifecycle

The checker validates the exact machine-readable contract and exercises
redaction, bounded profile authorization, logical-before-physical deletion,
active-lease refusal, exact replay, secret-free tombstones, and Shared/L1
generation rotation. This closes A1-004 and A1-G05 without claiming the
eight-hour soak, a design-partner deployment, or complete OPS-001/A1.
