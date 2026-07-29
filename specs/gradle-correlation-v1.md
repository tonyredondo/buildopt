# Gradle correlation v1 spike

**Owner:** `SPK-001` / `GRADLE-CORR-001`<br>
**Status:** closed as `UNAVAILABLE` for selective task publication<br>
**Decision:** use the tested all-attempt fallback

## Scope and pins

This spike answers whether a remote Build Cache `PUT` can be associated
unambiguously with one Gradle task execution and its terminal outcome. It runs
on Linux AMD64 with JDK 21 and Kotlin DSL against:

| Gradle | Source | SHA-256 |
|---|---|---|
| 9.6.1 | repository Wrapper | `9c0f7faeeb306cb14e4279a3e084ca6b596894089a0638e68a07c945a32c9e14` |
| 8.14.3 | `https://services.gradle.org/distributions/gradle-8.14.3-bin.zip` | `bd71102213493060956ec229d946beee57158dbd89d0e62b91bca0fa2c5f3531` |

Gradle 8.14.3 is provisioned only under the ignored `.tools/gradle-compat/`
directory after verifying the official distribution bytes. It does not replace
the 9.6.1 golden-lane Wrapper.

## Correlation adapter

The fixture enables Gradle's version-pinned internal build-operation trace and
consumes three structural facts:

1. `ExecuteTaskBuildOperationDetails` supplies a Gradle task identity and a
   unique build-operation ID. The spike names that operation ID
   `taskExecutionId`.
2. `BuildCacheRemoteStoreBuildOperationType.Details` supplies the native
   `cacheKey`, and the completed operation supplies `stored`.
3. Every operation has a structural `parentId`. A remote store is exact only
   when walking that ancestry reaches exactly one completed task execution.

The analyzer never derives ownership from timestamps, temporal order, thread
names, task paths, or cache-key equality. Task paths are retained only as
human-readable fixture labels. The loopback HTTP server independently records
completed requests, and the analyzer requires the HTTP `PUT` multiset to equal
the completed remote-store operation multiset.

These APIs and the trace property are Gradle internals. The adapter is therefore
limited to the two tested versions and must return `UNAVAILABLE` if a required
type, field, parent edge, task outcome, or HTTP observation is absent.

## Fail-closed decision

For every observed remote `PUT`:

```text
one task ancestor + completed task outcome + matching HTTP PUT
  -> EXACT(taskExecutionId, cacheKey, outcome)

zero or multiple task ancestors, missing outcome, or observation mismatch
  -> UNATTRIBUTED
  -> capability = UNAVAILABLE
  -> attemptAborted = true
```

The analyzer has synthetic checks for both missing and multiple task ancestors.
Unexpected analyzer failures are distinct from the supported `UNATTRIBUTED`
fallback and fail the executable check.

## Matrix and result

The successful matrix contains two projects and four cacheable task shapes:
direct task actions, Worker API no isolation, Worker API process isolation, and
a real child JVM. The direct pair uses a barrier to prove overlap; every pair
has equivalent declared inputs and byte-identical outputs.

Each version runs from a cold, isolated `GRADLE_USER_HOME`:

| Scenario | 9.6.1 | 8.14.3 | Evidence |
|---|---|---|---|
| Cold remote-cache path | `UNAVAILABLE` | `UNAVAILABLE` | Task-owned stores correlate exactly, but Kotlin DSL/build accessors also emit remote stores with no task ancestor |
| Repeated clean build | `EXACT` for observed events | `EXACT` for observed events | Configuration Cache reused; all eight fixture tasks were `FROM-CACHE`; no `PUT` occurred |
| Ordinary task failure | `EXACT` for observed events | `EXACT` for observed events | One terminal task, build failed, no task-owned `PUT` |
| Build cancellation | `EXACT` for observed events | `EXACT` for observed events | One cancelled terminal path, build stopped, no task-owned `PUT` |
| Missing/ambiguous ancestry | `UNAVAILABLE` | `UNAVAILABLE` | Analyzer self-test emitted `UNATTRIBUTED` and selected whole-attempt abort |

On the cold 9.6.1 run, ten remote stores belonged to Kotlin DSL compilation or
accessor-generation work rather than task execution. The cold 8.14.3 run had
nine such stores. Warming global caches can hide those operations and make a
later run appear exact, so the acceptance harness deliberately isolates the
Gradle user home.

The gateway observes these non-task HTTP requests just like task-owned
requests. Ignoring them would violate the rule that every `PUT` must have one
task execution and outcome. `GRADLE-CORR-001` therefore closes through the
specified safe degradation: selective task publication is unavailable for
these combinations, and any `UNATTRIBUTED` event aborts the complete pending
attempt.

The ephemeral cache used by this spike exists only to exercise Gradle and is
deleted with the attempt harness. Durable pending/commit cleanup remains owned
by the later gateway atomicity work; this spike fixes the decision that drives
that cleanup.

## Executable evidence

Run the golden version only:

```bash
./dev/run -- ./dev/check-gradle-correlation-spike --gradle-9-only
```

Run the complete pinned compatibility matrix:

```bash
./dev/run -- ./dev/check-gradle-correlation-spike --full
```

The first command is also part of `dev/golden-lane-build`. The second command
downloads 8.14.3 only when its verified archive is absent.

## Consequence for `F0-019`

The local task-event channel must encode exact task-owned observations plus the
fail-closed attempt decision. At minimum it needs explicit
`taskExecutionId`, native `cacheKey`, task outcome, `PUT` outcome,
`UNATTRIBUTED`, capability state, and whole-attempt abort reason. It must not
promote the task-only subset to an exact capability when the same attempt
contains a non-task `PUT`.
