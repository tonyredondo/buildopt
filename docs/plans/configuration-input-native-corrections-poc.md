# Configuration-Input Native Corrections POC

**Overall:** `STOP_STRICT_DIAGNOSTIC_BUDGET_CANNOT_COMPLETE_COHORT`<br>
**Current block:** `CINC-003` is complete and terminal<br>
**Dependent blocks:** `CINC-004..007` are `NOT_AUTHORIZED`

## Decision and objective

`CONFIGURATION_INPUT_NATIVE_CORRECTIONS_V1` (`CINC`) is the next bounded
BuildOpt experiment after WCNCP stopped at one actionable material family. It
tests whether strict Gradle Configuration Cache diagnostics expose a recurring,
repository-owned correction class: configuration-time external state is read
through an API Gradle cannot track, and a small native source change replaces
that read with `ProviderFactory.exec`, `ProviderFactory.fileContents`, a typed
Gradle provider, or a `ValueSource`.

This is not a continuation of the stopped WCNCP cohort. Every CINC observation,
source row, diagnostic, candidate, output comparison, and timing sample starts
at zero after its own committed contract and subject freeze. WCNCP, DNO, NAC,
RNPP, EGNP, CPBLC, and all other reports may motivate this hypothesis but are
forbidden evidence inputs.

The product boundary remains:

`Gradle through wrapper -> observation -> reviewed proposal -> validated native correction`

The accepted repository continues to use native Gradle through the wrapper;
BuildOpt does not replace task execution or remain inside the corrected task.

## Why this is materially different

The retired CPBLC route searched explicit task cache/state opt-outs and found
zero material proposal families. CINC does not reconsider those opt-outs and
does not add `@CacheableTask`. It targets configuration-model inputs and
Configuration Cache reuse. Gradle's supported replacement for a simple
configuration-time process read is `providers.exec`; complex reads belong in a
typed `ValueSource`. The value unit is skipped configuration work on a cache
hit, not task-output restoration.

The WCNCP GraphQL result is design motivation only. It contributes no CINC row,
duration, candidate, family, or threshold result, and GraphQL is excluded from
the first CINC cohort.

## Frozen detector v1

`CONFIGURATION_INPUT_NATIVE_CORRECTION_V1` consumes only a strict Gradle
Configuration Cache problem record plus source facts bound to its reported
location. Repository, task, plugin, and file names are labels and cannot change
classification.

The first version recognizes these source operations during configuration:

- Java/Kotlin `ProcessBuilder` or `Runtime.exec`;
- Groovy `execute()` on a command value;
- Gradle `Project.exec` or `ExecOperations.exec` when the result is read while
  constructing the configuration model;
- direct file-content reads or environment/system-property reads only when the
  strict report identifies them as an untracked or unsupported configuration
  input and Gradle documents a typed provider replacement.

One source binding receives exactly one typed decision:

- `SIMPLE_PROVIDER_EXEC_ELIGIBLE`: stdout/stderr status is the only consumed
  result; command, arguments, working directory, environment delta, charset,
  exit handling, and normalization are statically complete;
- `VALUE_SOURCE_REVIEW_REQUIRED`: the external read is bounded but requires a
  typed `ValueSource` and explicit reviewer confirmation of semantics;
- `TYPED_PROVIDER_ELIGIBLE`: a reported file/property/environment read has a
  direct Gradle provider equivalent with no new semantic choice;
- `EXTERNAL_OR_GENERATED_OWNER`: the correction is outside repository-owned
  source;
- `SIDE_EFFECTING_OR_SECRET_BEARING`: the operation writes, launches a service,
  performs network access, accepts stdin, or may capture credentials;
- `AMBIGUOUS_CONTROL_FLOW`: ownership, phase, consumed result, or exact source
  span is not uniquely provable;
- `SOURCE_DRIFTED`: the bound source SHA-256 no longer matches;
- `ALREADY_SUPPORTED`: the source already uses a supported tracked API;
- `NO_ACTION`: no supported v1 problem is present.

Eligibility never follows from a command name. The same source with renamed
repository, task, path, variables, or executable must classify identically.
V1 never suppresses a problem, increases an allowed-problem count, disables
Configuration Cache, changes dependencies/plugins, or treats external plugin
source as repository-owned.

## Required row facts

Every detector row records:

- family label, repository URL, exact revision, source-tree digest, wrapper and
  owner-workflow digests;
- Gradle/JDK/package/environment bindings;
- strict problem identifier, message digest, trace owner, phase and recurrence;
- source path, source SHA-256, language, declaration and operation spans;
- operation kind and explicit command/input/output/control-flow facts;
- ownership, generated/vendor, side-effect, secret, ambiguity and drift facts;
- proposed API and recipe version when eligible; and
- one typed decision and reason.

The evidence checker reparses raw strict reports, rehashes frozen source,
reconstructs bindings and decisions, and recomputes family counts. It never
trusts report summaries.

## Cohort policy

CINC-002 freezes exactly ten public Gradle repository families and two ordered
reserves per family before the first strict diagnostic. The primary revision is
the newest commit at or before the committed cutoff that passes only these
source-neutral admissibility facts:

1. checked Gradle Wrapper and a supported JDK are present;
2. an owner-documented build or check workflow exists;
3. required outputs are declarable without inventing a workflow;
4. optimized native preflight completes within 20 minutes; and
5. the declared 20-GiB experiment budget leaves at least 10 GiB free.

The ten primaries must exclude every WCNCP primary family and the historical
GraphQL, Micronaut, Spring, and Elasticsearch recipe families. Selection cannot
inspect Configuration Cache results, search for detector syntax, or use any
historical BuildOpt opportunity or timing. Replacement is allowed only for
unavailable repository, incompatible wrapper/toolchain, unavailable owner
workflow, failed optimized-native preflight, time limit, or disk limit, and
must consume the next frozen reserve.

The native selection preflight always passes `--no-configuration-cache` so an
Isolated Projects default cannot silently turn selection into a diagnostic.
Every subsequent start passes `--no-scan`; that rule was added after Mockito's
project default unexpectedly published one scan during preflight. The incident
is an audit fact only and neither its content nor its URL is a CINC evidence
input.

This deliberately tests a fresh, un-enriched cohort. A later enriched mechanism
study would require a different contract and could not make a breadth claim.

## Ordered blocks and authority

| Block | Work | Gate / stop |
|---|---|---|
| `CINC-000` | Freeze human/machine contract, tracker, independent contract checker, boundaries and budgets | Planning only; no public diagnostic or candidate |
| `CINC-001` | Implement parser/classifier v1 and independent fixture reconstruction | `DONE` — eleven raw-report rows cover direct providers, reviewed ValueSource, external owner, side effect, secret, ambiguity, drift, already-supported, no-action, and name invariance |
| `CINC-002` | Freeze exact ten-family cohort and reserves after native-only admissibility preflight | `DONE` — ten selected, twenty reserves, ten neutral native passes, zero strict diagnostics |
| `CINC-003` | Capture two fresh strict diagnostics per family and classify source | `DONE` terminal — 7/10 conclusive, 0/3 eligible; retained evidence alone proves 15 artifact-backed starts plus at least 6 required starts exceeds the fixed maximum of 20 |
| `CINC-004` | Run controlled materiality diagnostics for eligible families | `NOT_AUTHORIZED` — CINC-003 source gate failed |
| `CINC-005` | Compile at most one reviewed proposal per family and run correctness | `NOT_AUTHORIZED` — CINC-004 did not open |
| `CINC-006` | Run eight balanced optimized-native/candidate pairs per qualified family | `NOT_AUTHORIZED` — CINC-005 did not open |
| `CINC-007` | Present value-qualified drafts for first-exposure review and reconstruct terminal economics | `NOT_AUTHORIZED` — CINC-006 did not open |

Failure of a prerequisite marks dependent blocks `NOT_AUTHORIZED`. Missing or
unstable controlled capacity is `INCOMPLETE_PERFORMANCE_ENVIRONMENT`, never a
weaker threshold. Standard hosted CI owns deterministic contracts and
correctness only; its durations cannot qualify or reject materiality or value.

## Execution limits

- additional disk: 20 GiB maximum;
- free disk before a public Gradle start: at least 10 GiB;
- stop before the next start below 8 GiB;
- owner workflow/preflight/diagnostic: 20 minutes each;
- strict diagnostics: at most 20 primary starts plus one replacement preflight
  per consumed reserve;
- controlled materiality: at most two starts per eligible family and six total;
- candidate: at most one per family and six correctness starts per candidate;
- value: one excluded stabilization per isolated root and eight balanced pairs;
- workers: four on this owner-operated host; fixed CPU/power bindings and the
  existing 1.15 seven-sample stability gate apply before controlled timing.

Before a long run, dependencies are prefetched outside measurement and the
fixed 120-second quiescence used by WCNCP-009B is applied. CINC owns fresh empty
checkout, Gradle, Build Cache, Configuration Cache, backend, and output roots.

## Correctness and patch constraints

The candidate is a digest-bound `PatchBundle` with exact preimage, postimage,
inverse, source spans, recipe version, diagnostic bindings, and validation
budget. It is applied only in an isolated copy. The public checkout, upstream
repository, and owner default branch are never mutated.

Correctness requires the exact owner arguments and outputs, strict problem
elimination without new problems, Configuration Cache store then reuse,
changed command/file/property/environment inputs invalidating the cache as
applicable, equivalent exit/error behavior, exact source revert, and zero
additional product failures. A remaining external or unrelated blocker rejects
the proposal; it is not suppressed.

## Terminal interpretation

Passing CINC would establish a bounded reviewed-native configuration-input
correction product across at least two fresh families. It would not prove
arbitrary-repository discovery, production readiness, autonomous source
mutation, automatic merge, Test Optimization, or universal Configuration Cache
compatibility.

The CINC route is terminal at `CINC-003`. Four families produced two clean
zero-problem reports. Native Build Tools recurrently exposed an external
`org.ajoberstar.git-publish` listener. Testcontainers recurrently failed on an
out-of-scope task `onlyIf` closure after storing a zero-problem cache entry.
Mockito recurrently exposed one external already-ValueSource-backed Shipkit
read and three non-serializable task/script references. These seven families
are conclusive and supply zero eligible CINC recipe families.

OpenAPI Generator exposed a cold JDK 21/Gradle 8.1.1 script incompatibility
that the warm selection preflight had missed. Licensee completed its workflow
but produced 237 root-and-nested reports, exposing an over-broad capture rule.
Gradle Profiler was not started. Three earlier harness failures did not retain
raw logs; they remain disclosed in the attempt ledger and are not required for
the stop arithmetic. Fourteen valid captures plus the retained invalid Mockito
archive-root capture independently prove 15 primary starts. Completing the
three remaining families requires at least six more strict primary starts, so
the fixed maximum of 20 cannot yield a complete cohort even under the most
favorable interpretation.

The independent [source-gate checker](../../dev/check-configuration-input-native-corrections-source-gate)
reparses every retained report, rehashes its raw log and source snapshot,
reconstructs all ten rows and family counts, and proves `15 + 6 > 20` without
trusting the evidence summary. No public source mutation, candidate build,
timing sample, speedup claim, or additional product failure exists.
