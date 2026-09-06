# Complete Native Correction v1 — Phase A contract

Status: `STATIC_CONTRACT_ONLY`. CNC-002 defines a directed, local non-CI
GraphQL Java study. It is not an executable campaign, tested correction, or
permission to start Gradle. The owner approved **60 starts maximum, two elapsed
hours, and a progress review at 30 minutes**. Whichever limit is reached first
stops execution. CNC-003 supplies a locally checked native harness and
[runbook](../docs/reference/complete-native-correction-capture.md); real execution
inputs and committed package identity remain CNC-004 prerequisites.

The [machine contract](./poc-complete-native-correction-v1.json) owns exact
arguments, ordered slots, resource limits and gates. The
[subject manifest](./poc-complete-native-correction-v1.subjects.json) owns source,
span, runtime and output bindings. The [plan](../docs/plans/complete-native-correction-poc.md)
owns the later, separately authorized installed-product and replication phases.

## Scope, identity and refusal

Use GraphQL revision `f2d8c9126f898c084b176631b7346bc6fbec296a`, its unchanged
Gradle 9.6.1 Wrapper, Bnd 7.1.0, and the two exact Corretto archives in the
manifest. These archive requirements do not assert installation or runtime
verification. Verify the Git archive and every file/span before execution;
resolve and retain the actual plugin binary digest before real fixtures. Any
missing or changed binding refuses execution. No `latest`, plugin upgrade,
dependency change or locally convenient JDK substitution is allowed.

The native value request is root `assemble`, with `CI` and `RELEASE_VERSION`
absent in both arms. Reject an invocation presenting either variable; do not
strip CI variables from a real CI job. Begin detached at the frozen revision,
preserving the existing HEAD-SNAPSHOT policy. CI/release/no-Git preservation
belongs in fixtures, not in the public value claim. Never freeze the public
clock or inject a release version to manufacture exact artifacts.

All five source owners and their consumers in the
[feasibility ledger](../benchmarks/results/complete-native-correction-v1/selection/feasibility.md)
must be covered. The Bnd change may configure its existing public properties
API; it cannot remove its action or alter third-party source. Lifecycle APIs
remain a hypothesis until their real fixture obligations pass. Newly found
unsupported blockers stop the recipe; names are labels, never admission rules.

## Resource and evidence ownership

The 7,200-second monotonic window starts before execution setup or downloads.
It includes preparation, quiescence, fixtures, failures, validation and waits.
The 1,800-second checkpoint records progress, remaining proof and remaining
time, and stops early if no useful progress is possible. Neither a checkpoint,
restart, compaction, human wait nor infrastructure failure resets the deadline.
Every child timeout is at most 1,200 seconds and never exceeds the remaining
window. Stop the entire owned process subtree, including detached daemons, at
the deadline and retain an
`INCOMPLETE_EXPERIMENT_BUDGET_EXHAUSTED` result, not a partial success.

Charge all real Gradle subprocesses, including nested fixture builds, probes,
prefetch, owner tests, failed starts and replacements. A test assertion is not
an extra start, but hiding Gradle behind a test framework does not make it free.
Static checks and non-Gradle children still consume elapsed time during execution.
The static preparation performed before this execution window is research
effort, not zero-cost optimization; its earlier unmetered portion stays unknown.

Use one start at a time, four workers, CPUs 0-3, mains power, and the frozen
performance/EPP envelope. Additional concurrent disk is capped at 20 GiB;
require at least 10 GiB free before each new child. Do not stop unrelated
processes, change host power policy, or delete existing worktrees implicitly.
Hosted CI checks static contracts/correctness only; no numerical wall-time gate.

Atomically reserve an attempt before spawn. Retain actual argv, environment
deltas, input/package digests, roots, monotonic timestamps, exit/signal,
stdout/stderr, outputs and terminal classification for every started child.
An ambiguous reservation after interruption must be reconciled, not rerun.
Reserve R01/R02 can replace only independently classified infrastructure loss,
once per failed attempt, with the exact original profile and both attempts
retained. Product failures, unfavorable timing and missing proof are not retries.

## Commands and state

Each machine `commands` array is a Gradle argument profile, not an existing
BuildOpt command. CNC-003 must freeze its literal launcher and fixture files.
Expand it only with `--gradle-user-home` and the assigned private absolute path,
the two verified runtime paths through
`-Dorg.gradle.java.installations.paths=...`,
`-Dorg.gradle.java.installations.auto-detect=false`, and
`-Dorg.gradle.java.installations.auto-download=false`.
Set `JAVA_HOME` to the pinned daemon runtime and set `TZ=UTC`, `LC_ALL=C.UTF-8`,
and UTF-8 JVM encoding identically. Reject undeclared Gradle/JVM option
injection. After P02 append `--offline` to both arms and to fixtures. No
candidate-only daemon, worker, task, cache, or output-elision flag is allowed.

Implement no-owner-home reuse with a private `HOME`, JVM `user.home` and empty
JVM `maven.repo.local` under each assigned Gradle home. Only the runner-owned
`JAVA_TOOL_OPTIONS` supplies these bindings and UTF-8; do not inherit ambient
options, Maven settings or local artifacts. Reject changed private inputs.
This is symmetric input isolation, not a change to `mavenLocal()` declarations
or the subject's version semantics. Campaign cgroup ownership may preserve
native daemons between successful requests; it must terminate them at expiry,
cancellation or capture-owner loss without touching unrelated processes.

Create registered shared-Git detached worktrees `native-a`, `native-b`, and
`candidate`, with initially empty private Gradle homes/build directories.
These are logical names, not machine paths or classification rules. P01/P02
run symmetric `assemble testClasses` only to prepare native dependencies and
the owner's test classpath; these extra preparation tasks never enter timing.
Retain an immutable dependency-only seed. Give every arm the same verified
dependency bytes, excluding Configuration Cache, build cache, execution
history, output and daemon state. No source copy, clone, owner-home reuse, or
unverified cache snapshot is allowed. A missing offline dependency stops rather
than enabling asymmetric network access or adding an uncounted warmup.

Execution order is P01/P02, D01/D02, M01/M02, F01-F24, C01-C10, V01-V18.
R01/R02 are conditional replacements, not extra successful rows. D01/D02 use
separate empty Configuration Cache namespaces. Their expected strict failure
is diagnostic evidence; the successful native comparator has Configuration
Cache disabled. Parse complete root-log-owned reports and all problem owners,
not just the expected five or a conveniently found HTML file.

M01/M02 use native profiles with operation/DAG capture owned by the frozen
CNC-003 instrumentation. After prefetch, apply 120 seconds of quiescence and
seven stability samples (max/min at most 1.15). Reconstruct at least 500 ms and
2% material configuration work before compiling the recipe. Instrumentation
is absent from value rows. Fresh materiality is not achievable saving.

## Real fixture protocol: F01-F24

The recipe and runner must be package-frozen before F01. CNC-003 proves generic
harness behavior with fake children; CNC-005 owns these real recipe fixtures,
after fresh native admission. This avoids requiring an admitted recipe before
its native prerequisite. The fixture uses the actual pinned Bnd extension,
source-bound original version/listener logic and the actual candidate recipe,
not a hand-written simplified substitute. Reuse owner-resolved test dependencies.

Both fixture profiles request exactly `fixtureProof`; it depends on both Bnd
JAR variants and multiple real Test tasks. Force task execution in correctness
fixtures, including candidate Configuration Cache reuse. Capture actual test
events independently and reconstruct each run's summaries: totals, failures,
skips, >500-ms inclusion, descending duration order, top 20 classes/top 50 tests,
exactly-once printing, and build-summary-before-test-summary order. Compare
format/content to that run's events, not equal clock-dependent durations across
two different runs. Exercise zero-test and nonempty-test reporting, cancellation
and fresh service state; fake counters cannot replace real listener evidence.

| Slots | Input and obligation |
| --- | --- |
| F01-F03 | Native successful reference, candidate store, then candidate reuse. Real test events, Bnd JAR bytes/manifests, both summaries and local detached version agree with their independent oracles. |
| F04-F05 | Change only the fixture Git branch result to `feature/cnc`; keep argv, project directory, script bytes, environment, Git launcher and candidate cache entry from F03. Native agrees; candidate must invalidate and produce the sanitized branch version. |
| F06-F07 | Change only fixture HEAD commit, leaving branch result `feature/cnc` unchanged. No source/build-script change; candidate retains F05's entry and reuses it because the local version does not consume the hash. |
| F08-F09 | Change only `RELEASE_VERSION` from absent to `cnc-release`; keep F07's other inputs/cache. Candidate invalidates, preserves the exact override and avoids unnecessary Git calls. |
| F10-F11 | Independent fresh namespace: `CI=true`, release absent, real short-hash path. Timestamp belongs to each run's own interval; preserve empty-hash exception and stderr. No same-input reuse or isolated CI-invalidation claim comes from this fresh pair. |
| F12-F13 | Independent fresh namespace with no repository. Preserve timestamped no-Git fallback and streams. Do not represent these non-Git fixtures as subject worktrees. |
| F14-F15 | Independent fresh namespace with an unavailable Git executable. Preserve actual launch failure; never silently return a successful version. |
| F16-F17 | Independently observe real process launch/wait/stdout/stderr for exit zero/nonzero, empty/malformed/whitespace output, branch separators and CI empty-hash exception. The original and candidate source-bound implementations must each exercise every case, catching exceptions only at the fixture observation boundary. |
| F18-F19 | Actual task failure after both summary registrations, identical declared test failure, equivalent exit and reporting. |
| F20-F21 | Actual configuration failure after both registrations; no successful task may stand in for build-work-result coverage. Capture exactly-once ordered final reporting. |
| F22-F23 | Cancel native/candidate after an observed child-start handshake. Capture bounded exit and finalization; do not use a race-prone sleep or assert graceful handling of SIGKILL. |
| F24 | Successful candidate in the same root after cancellation, resetting only cancelled entry/output state as declared. No stale service counters, failures or lifecycle actions. |

F01-F09 share one immutable configuration/argv identity per arm. Their Git
result driver is an actual executable subprocess; only its externally supplied
result changes. Repeat worktree detection against both real Git-directory and
Git-file repositories and different working directories inside each matrix
start. These checks must not start nested Gradle. F16/F17 may share assertions
only when the raw per-case trace proves execution of each source-bound path;
an early exception with unexecuted cases fails proof and stops the recipe.
Source extraction and any fixture-only context adapter must be hash-bound and
reviewable. No adapter may replace the behavior or input tracking under test.

Only F03/F05/F07/F09 claim specific cache reuse/invalidation. Preserve their
preceding entries without scenario flags or changed scripts masking the tested
dependency. Other fixture pairs prove behavior, not single-input cache tracking.
CI/no-Git clock-dependent paths must preserve current-time semantics; they must
not be advertised as reusable local hits. Record their mode separately.

## Public correctness: C01-C10

Compare complete path/size/SHA-256 inventories and bytes for every required
JAR under both root and Java 21 subproject `build/libs`, plus plain/shadow
intermediate JARs. Bind every output to its actual producer. Reject absent,
unowned, symlinked or additional unexplained artifacts. No timestamp removal,
manifest rewriting, ZIP normalization, output omission or digest substitution.
Capture two native inventories before any candidate; retain realized producer
and output identities in the immutable attempt. Static selectors are not an
already observed inventory. Unexpected producer closure stops for analysis.

C01 uses native-a and C02 native-b. All correctness rows symmetrically disable
the build cache and run the existing `verifyShadedClassAnnotations` and
`compileShadedJarConsumer` owner tasks. Remove only the captured producer
outputs before C01-C04 to force the actual JAR actions, retaining candidate
Configuration Cache between C03 and C04. Never use `--rerun-tasks` in public
correctness: it would hide whether a source mutation invalidates task state.
Retain outputs and execution history through C05-C09. C03/C04 apply the single
recipe in candidate and prove store/reuse with the same tasks and exact outputs.
Stop immediately on native/native instability
or candidate mismatch; do not proceed to a favorable timing subset.

C05 adds only `cnc-unrelated-note.txt` containing `CNC_UNRELATED\n`; retain C04's
configuration entry and require reuse and identical outputs. Remove that
unrelated probe before the relevant-source pair; its absence is already proved
irrelevant and cannot be used as an input-tracking assertion.

C06/C07 insert `private static final int CNC_PROOF_SENTINEL = 1729;` immediately
inside `public class Assert {` in the exact hash-bound
`src/main/java/graphql/Assert.java`, identically in native-a and candidate.
Prove recompilation and changed class bytes with exact inter-arm artifacts.
This is task-input invalidation; do not require a configuration miss for a
source file Gradle correctly tracks only at task execution. C08/C09 replace
only that initializer with `"CNC_INTENTIONAL_TYPE_ERROR"`, and require the same
intentional compile failure and correct summaries. Keep all negative logs;
these rows are not timing observations or unexpected product failures.

C10 restores exact preimages for every recipe and probe file in candidate,
then executes native and compares the original required outputs. Static tests
also prove apply/reapply/revert/rerevert and refuse drift/tampering. Never use
Git reset or broad cleanup as an inverse. The probe is not a proposed product
change and never enters value rows. No public branch/ref mutation or relocation
claim is authorized by these fixtures.

## Native value and cost: V01-V18

Begin from verified original native source and the exact admitted candidate.
Discard only task-owned correctness outputs, Configuration Cache, build-cache
and execution history symmetrically; retain the dependency-only seed. Record
both initial inventories. V01/V02 are excluded native/candidate stabilizations.
V03-V18 form eight pairs: AB, BA, AB, BA, AB, BA, AB, BA. Before each row remove
the same declared producer outputs in both arms, preserving their evolving
native caches. Do not delete source or undeclared paths. Candidate must reuse
configuration in every accepted pair and both arms retain equal native build
cache opportunity. Use root `assemble` only, no auxiliary correctness task,
instrumentation, synthetic mutation or `--rerun-tasks` in value.

Measure the whole child invocation with an external monotonic envelope. Require
8/8 positive pairs, mean saving >=500 ms and >=2%, positive paired 95% lower
bound, non-regressive nearest-rank p95 and zero additional product failures.
For eight observations p95 is the maximum. Bootstrap 4,096 samples of eight
paired deltas with replacement: initial state is
`2654435761 * (sample + 1) mod 2^32`; each draw uses
`(1664525 * state + 1013904223) mod 2^32`, index
`floor(state / 536870912)`. Sort sample means and select indices 102 and 3993.
Freeze that algorithm, never historical rows or a hard-coded winning result.

Machine payback is ceil(total observed operational machine milliseconds /
mean saved milliseconds), screened at 300 compatible builds. Include failed
and excluded work; keep research, human review, storage/network and user-visible
latency ledgers distinct. Unknown earlier research time cannot become zero or
an observed repayment claim. Phase A can qualify only the directed native
mechanism; it cannot qualify wrapper overhead, persistence, breadth or commerce.

## Static checker and next boundary

Run `./dev/check-complete-native-correction contract`. It executes only Go
contract tests and reconstruction; it does not run Gradle or read old reports.
Optional `--contract FILE --subjects FILE` selects explicit documents.
Optional `--sources-root ROOT` checks the exact subject files/spans locally;
it does not verify the Git archive, runtime binaries, recipe or runtime behavior.

The checker rejects duplicate/unknown JSON keys, altered limits or authority,
missing/reordered/changed slots, inconsistent counts, source/runtime drift,
missing outputs, weaker gates and name rules. Relabeling the subject still
passes. Raw-byte/source-span tests preserve original newline bytes and refuse
unsafe paths or symlinks. None of these static tests substitutes for F01-F24.

CNC-003 must implement the portable launcher, budget/attempt state, artifact
capture, fixture context and negative-tested runbook and bind actual package
digests. Publication identity and execution authority remain separate gates.
No real Gradle run, candidate patch or measurement is started by this contract.
