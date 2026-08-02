# Glossary

## Action

A proposed or active optimization with an auditable lifecycle. An action is not
authorized merely because code can perform it.

## Attempt

One isolated candidate/control/baseline execution and its pending cache writes,
evidence, and terminal commit or abort decision.

## Authority

The signed, scoped, current policy and revocation state that permits an
operation. A blob, checksum, token, or successful build is not authority by
itself.

## Baseline

The original Gradle command and policy used as the correctness reference.
BuildOpt preserves this path and its exit status.

## Build Impact

The conservative capability that may choose a repository-authorized smaller
set of Gradle entrypoints from a declared graph. Unknown state falls back to
the full original graph. It does not optimize Test-owned checks.

## `BUILD_SESSION`

The normative immutable record of one build session, including identity,
timing, outcome, capability-labelled observations, and redacted context.

## Cache planes

- **L1:** Gradle's native local `DirectoryBuildCache`, private and
  generation-segmented.
- **L2:** Gradle's HTTP cache endpoint, always the local verifying gateway from
  Gradle's perspective.
- **Edge:** optional nearby bounded read-through/pending-write node.
- **Shared:** first-party backend and sole commit/collision authority in the
  current POC.

## Capability status

- **EXACT:** the supported adapter/fixture proves the stated observation.
- **APPROXIMATED:** a named bounded method estimates it.
- **UNAVAILABLE:** it cannot currently be provided and has an explicit safe
  fallback.
- **UNKNOWN:** the platform/Gradle combination is outside proven profiles and
  receives no borrowed policy.

## Commit decision

The authenticated control-plane decision that atomically makes a complete
pending candidate visible as committed, or aborts it. Blob presence cannot
substitute for it.

## Compatibility class

The bounded Gradle, JDK, DSL, plugin, OS, and architecture identity under which
evidence and cache behavior are valid. Policy is not reused across classes.

## Control plane

Policy, revocation, attempts, commit decisions, validation evidence, rollout,
and lifecycle state. It authorizes data-plane use.

## Data plane

Cache bytes and their bounded transport/storage paths. The data plane cannot
mint its own stable authority.

## Edge Cache

An optional owner-operated cache near runners. It can serve only committed
objects under current revocation and keeps offline writes visible only to the
exact pending attempt.

## Evidence

Immutable observations bound to source, policy, compatibility, method, and
time. Observation is not proof until the owning qualification gate accepts it.

## Fail closed / fail open

- **Fail closed:** reject the optimization or privileged operation when proof
  is incomplete, preserving correctness and authority.
- **Fail open to baseline:** allow the original Gradle command to run when an
  optional optimization or telemetry path is unavailable.

Fail open never means inventing successful telemetry or accepting unverified
cache content.

## Golden lane

The pinned primary development/runtime combination used for reproducible
implementation evidence: Gradle Wrapper, JDK, runner class, and toolchains.

## Local Verifying Cache Gateway

The loopback L2 endpoint owned by the launcher. It authenticates Gradle
locally, hides upstream credentials, verifies policy/checksums/revocation, and
maps cache failures to safe Gradle behavior.

## Managed L1

A launcher-owned native Gradle local cache scoped by tenant, repository, trust
domain, compatibility class, and security generation, with an invocation
lease.

## Neutral envelope

The paired measurement boundary that compares the native baseline and
optimization-off wrapper without attributing unobserved time savings.

## Patch Autopilot

The signed PR-only system for exact bounded repository transformations,
isolated application, full relevant validation, draft delivery, monitoring,
and exact revert. It does not merge or force-push.

## Pending / committed / aborted

Cache object lifecycle states. Pending is visible only to the exact authorized
attempt; committed is generally readable under current authority; aborted is
never promoted.

## POC

The owner-operated proof-of-concept profile implemented here. It demonstrates
functional composition and synthetic evidence but does not claim production
multi-tenancy, HA, external validation, or the deferred soak.

## Runner slot

A stable orchestrator-provided identity for serializing one managed gateway
invocation and its private persisted state. Concurrent builds use different
slots.

## Shared

The first-party single-node cache backend containing immutable blobs and
separate SQLite cache/control metadata. In the POC it is part of the trusted
computing base.

## Task Intelligence

The qualification lifecycle for task behavior and cache publication, from
observation through reviewed contract, quarantine validation, activation, and
suspension.

## Test Optimization

A separate product that owns test selection, sharding, retries, prioritization,
and flake policy. BuildOpt may consume explicit signed grants/contracts but
does not implement those decisions.

## Trust domain

An explicit security boundary included in repository/cache/policy scope. State
or credentials from one trust domain are not reused in another.
