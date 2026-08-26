# Current longitudinal campaign v1

Status: accepted POC measurement contract for `AF-014C`.

## Purpose

This protocol measures the current installed BuildOpt package over the five
public first-parent cohorts frozen by `AF-014B`. It compares the exact requested
workflow against optimized native Gradle while preserving chronological
candidate learning and every negative observation. It does not authorize a
production rollout or decide whether the adaptive-fragment POC should continue.

The authoritative machine contract is
[`poc-current-longitudinal-campaign-v1.json`](./poc-current-longitudinal-campaign-v1.json).

## Execution model

Repositories run sequentially. Each family has persistent but isolated control
and candidate checkouts, Gradle homes, build caches and daemon registries. Only
the candidate owns BuildOpt state. Observation `N` may consume only candidate
state committed by observations `1..N-1`; there is no future-state replay and
no untimed candidate warmup.

Dependency preparation runs against the anchor and then against every attempted
revision before either timed arm. It may use the network, is never included in
pair wall time and copies only Gradle dependency modules, dependency-verification
keyrings and Wrapper distributions into the two isolated Gradle homes. It never copies task outputs,
native build-cache entries, project state, daemon state or BuildOpt state. This
keeps both arms offline and equivalent when a chronological commit changes a
dependency or plugin version without treating that expected evolution as an
exclusion. Preparation attempts, outcomes and total unmeasured wall time remain
visible in raw evidence.

The per-revision preparation workspace is chronological and incremental. Gradle
must resolve changed dependency, plugin and classpath inputs, but the runner
must not force `--rerun-tasks`: rebuilding unaffected tasks would add no
dependency coverage and would turn preparation into an unmeasured duplicate of
the requested build.

After a repository closes, the runner retains its immutable `subject.json` and
removes only that repository's transient checkout, dependency and Gradle-cache
state. This bounds disk use without changing later observations or the final
aggregate.

The first pair is control-first and order alternates thereafter. Both arms run
the frozen workflow on the same public revision. The runner records independent
monotonic wall time, the candidate's non-overlapping internal phases, cache,
daemon and state fingerprints, required-output hashes and the complete installed
decision. The raw record preserves calibration cost and sample count, profile
identity, selection matching cost and bindings, discovered graph reduction and
the native-retention phase. Learning and calibration work remains inside
candidate wall time so `AF-014D` can calculate qualification cost and payback
without reconstructing facts from transient logs.

## Exclusions and reserves

A primary commit may be excluded only for native build failure, unavailable
dependencies after preparation, runner-environment failure or native-output
nondeterminism. The next frozen reserve is consumed in its recorded order. A
slow candidate, native retention, a negative delta or an unhelpful change shape
is never excluded. All exclusions remain in the raw evidence.

Git revisions are validated as hexadecimal Git object IDs (SHA-1 or SHA-256),
while executable, contract, cohort, output and state digests remain strict
SHA-256 values. A repository with zero comparable observations is still valid
raw evidence only when all 30 frozen primary/reserve attempts are present as
declared exclusions.

## Fragment boundary

The installed `buildopt optimize` command currently selects whole structural
profiles. The storage-neutral adaptive-fragment planner and Build Impact
activator are not yet wired into that public command. Therefore the campaign
must report `NO_FRAGMENT_RUNTIME` and an empty activated-fragment set unless an
installed result carries real fragment identities. A selected whole profile
must never be relabelled as a fragment. `AF-014D` owns the consequence for
mechanism attribution.

## Acceptance

Each repository must provide at least 15 comparable pairs or be explicitly
`INSUFFICIENT_COHORT`. Required outputs must be exact, all adverse deltas and
exclusions must remain visible, and measurement-only executions may not update
fragment authority. A positive or negative row requires both the corresponding
cumulative sign and at least 60% of comparable pairs in that direction;
otherwise the row is inconclusive. The deterministic checker recomputes signed
and cumulative net value from immutable observations. These are current POC measurements, not
a production, soak, design-partner or Test Optimization claim.
