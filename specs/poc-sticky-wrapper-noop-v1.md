# Sticky wrapper native no-op v1

Status: accepted POC implementation contract for `SWL-008`.

This contract defines the first consumer of the signed sticky-wrapper decision
store. It is deliberately a retention path, not an optimization engine:
Gradle remains the authority and no runtime profile, patch or trial may be
executed by this block.

## Selection rules

Before a Gradle process starts, the selector may read one repository-scoped
local snapshot from the user cache. The snapshot is usable only when all of
the following checks pass:

1. the root, records directory, cache directory and lock are private regular
   filesystem entries;
2. the head and immutable record are canonical, content-addressed and bound
   to the requested repository scope;
3. the record is a signed `STICKY_DECISION`, and the named owner Ed25519 key is
   present in the local trust registry;
4. the decision is unexpired, not revoked and has an exact match for every
   `Binding` field; and
5. the execution decision is `NATIVE_NOOP` or a fully qualified `ACTIVE`
   decision.

`NATIVE_NOOP` is accepted as the safe native result. An exact
`ACTIVE_RUNTIME_PROFILE` or `ACTIVE_DURABLE_PATCH` is reported as compatible
but remains deferred until `SWL-011`, where action execution and suspension
are implemented. `OBSERVE`, `SHADOW`, `TRIAL`, `SUSPENDED` and `RETIRED` never
execute an action here.

Any missing, expired, revoked, corrupt, busy, cross-scope, wrong-key or
otherwise incompatible state returns native Gradle. The selector never
contacts the central service, creates a missing directory, changes a file or
uses a cache object as authority.

An optional refresh callback is coalesced and scheduled asynchronously after a
native fallback. Its failure is ignored by the build path; a service outage is
therefore a native decision, not a build failure or a blocking network call.

## Compatibility and timing budget

The benchmark measures the local lookup before Gradle with synthetic signed
state and a cold/missing snapshot. It records p50, p95 and maximum nanoseconds
for 200 or more selections per case. The frozen POC limits are:

| Case | Requirement |
| --- | ---: |
| Verified local snapshot p50 | ≤ 100 ms |
| Verified local snapshot p95 | ≤ 250 ms |
| Missing or service-unavailable fallback p95 | ≤ 500 ms |

These are retention budgets, not speedup claims. A passing result proves that
the decision machinery is cheap enough to continue the learning experiment;
it does not prove that an optimization is valuable.

## Evidence

Run the deterministic check with:

```bash
./dev/check-sticky-wrapper-noop
```

The versioned measurement is
[`benchmarks/results/sticky-wrapper-noop-v1.json`](../benchmarks/results/sticky-wrapper-noop-v1.json).
The implementation is [`internal/stickydecision/selector.go`](../internal/stickydecision/selector.go)
and its focused tests. `SWL-009` consumes this selector to record ordinary
build evidence; `SWL-011` is the first block allowed to execute a qualified
action.
