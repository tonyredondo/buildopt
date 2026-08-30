# Sticky wrapper learning POC v1

Status: superseded architecture contract; retained for diagnostic history.

## Purpose

This contract preregisters the successor experiment after
`STOP_ADAPTIVE_FRAGMENT_POC`. It tests whether a repository-committed wrapper,
central Gradle HTTP cache and separate typed decision store can provide one-
command onboarding and positive cumulative build-time value without recurring
native-retention cost.

The exact machine contract is
[`poc-sticky-wrapper-learning-v1.json`](./poc-sticky-wrapper-learning-v1.json).
The historical implementation and documentation obligations are in the
[`Sticky Wrapper Learning POC Tracker`](../docs/plans/sticky-wrapper-learning-poc-tracker.md).
The immediate successor was
[`poc-fresh-generic-optimization-v1`](./poc-fresh-generic-optimization-v1.md),
which is now closed and retained as historical evidence.

## Customer surface

A maintainer generates and commits exactly four portable files:

```text
buildoptw
buildoptw.bat
.buildopt/wrapper.properties
.buildopt/config.toml
```

The repeated developer and CI invocation is:

```text
./buildoptw <gradle args...>
```

No BuildOpt binary, credential, checkout path or hand-authored optimization
profile is committed. The wrapper downloads one checksum-pinned BuildOpt
distribution and invokes the repository's existing Gradle Wrapper.

## Authority

The Gradle cache data plane and BuildOpt state control plane are independent.
A Gradle object, key, digest or hit never grants action authority. An action
requires a canonical decision bound to repository scope, workflow, Gradle,
Wrapper, arguments, output contract, action generation, expiry and revocation.

The central service is optional for correctness. Missing service and absent,
expired, corrupt or incompatible local state select native Gradle. Credentials
come only from a private runtime source and never enter committed files or the
Gradle process.

## Lifecycle

The allowed lifecycle is:

```text
UNSEEN -> OBSERVE -> SHADOW -> TRIAL -> QUALIFIED -> ACTIVE
                                                \-> SUSPENDED -> RETIRED
```

The execution decision is one of `NATIVE_NOOP`, `OBSERVE`, `SHADOW`, `TRIAL`,
`ACTIVE_RUNTIME_PROFILE`, `ACTIVE_DURABLE_PATCH`, `SUSPENDED` or `RETIRED`.
Only measured active executions receive runtime saving attribution. A durable
patch is credited only through paired native Gradle evidence after review and
acceptance; subsequent execution is ordinary patched Gradle.

## Measurement

The control is optimized native Gradle with the same local and remote Gradle
cache opportunity. Every chronological row includes bootstrap, wrapper,
decision, state, observation, trial, cache, fallback and action costs.
Percentages from different mechanisms or repositories are never added or
averaged into a product claim.

The terminal campaign freezes five substantial public Gradle families and at
least 20 requested builds per family before timing. A row needs at least 15
valid comparable builds. Required outputs use exact bytes unless a separately
reviewed bounded equivalence contract applies.

## Decision

The complete scorecard in the machine contract is immutable after the first
timed observation. Every criterion must pass for
`CONTINUE_STICKY_WRAPPER_LEARNING_POC`. A failed correctness, breadth,
confidence, cumulative-value, payback or overhead criterion yields
`STOP_STICKY_WRAPPER_LEARNING_POC`. Incomplete evidence yields `INCOMPLETE` and
does not authorize a threshold change.

Historical generic/adaptive profile evidence provides design context only. It
does not count as a current observation, activation, saving or passing family.

## POC boundary

This contract does not require production SLOs, high availability,
multi-tenancy, billing, enterprise identity, an eight-hour soak or an external
design partner. It grants no automatic merge or production promotion authority
and owns no Test Optimization behavior.
