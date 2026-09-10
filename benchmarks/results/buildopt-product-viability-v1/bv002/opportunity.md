# BV-002: Native opportunity decision

Date: 2026-09-08. Status: **verified, G1 negative**.
Decision: **NO_MATERIAL_NATIVE_OPPORTUNITY** for Elasticsearch
`ForbiddenPatternsTask` in the frozen `:server:precommit` engineering prefix.
[Machine decision](./opportunity-decision.json),
[complete analysis](./opportunity-analysis.json),
[current tracker](../../../../docs/plans/buildopt-product-viability-v1-tracker.md).

The historical replay works, but this selected intervention does not meet the
plan's admission floors. Keep BV-003 and later candidate work deferred for this
subject. These are native observations and generous modeled ceilings, not a
measured optimizer speedup or a verdict on other repositories.

## Source, native baseline and proof

The anchor is `53a80bec683ad0b065ed9ebe6a57f984e6a91ed1` and ordinal 20 is
`22d6425e9a44ec9e1aedc2785f4478b8019c851a`. Every intervening first-parent tree
ran in order, preserving the same workspace, native caches and native daemon.
This prefix covers 9.068 hours of repository history. The separately frozen
80 validation transitions remain unconsumed; the complete 100-transition window
covers 45.959 hours, not a year of observed development.

[baseline.json](./baseline.json) freezes the observed native settings: pinned
JDK 21 and Gradle 9.7.1, the additional pinned compiler JDKs, eight workers/CPUs,
native parallel execution, build cache and a persistent Gradle daemon. The
retained `@CacheableTask` correction belongs to N, with exact forward/inverse
patches in [baseline](./baseline/native-baseline.patch). Its postimage SHA-256 is
`d6858f5ac43ad671496cf1e54e7af309cb98ebda4a9be53e61578df8baefa2d0`.

All **20 owner tests** ran with no failures, errors or skips
([owner proof](./owner-proof.json), [XML](./owner-tests.xml)). The native anchor
and all 20 transitions completed successfully. Every invocation verified the
complete tracked-source inventory, complete executed main-build task graph and
successful marker bytes `done`. No native invocation changed tracked source.
The native daemon retained PID 3447624/start ticks 31679984 throughout this
session; the containing transient unit is now unloaded.

Configuration Cache stored successfully, but its ordinary reuse in a new
daemon failed in `:server:spotlessJava`, property `lineEndingsPolicy`, with
`Spotless JVM-local cache is stale`. No custom observer/init script ran in
that probe. The preregistered compatibility rule therefore selected explicit
`--no-configuration-cache`; both the success and failure remain in the
[decision](./configuration-cache-decision.json) and
[failed reuse log](./configuration-cache-reuse-failure/stdout.log).
This is a limitation of the tested combination, not a general Gradle claim.
Checkstyle engine caching has not been qualified as an additional native feature.

The source advance created a new local branch for each exact historical tree,
with an exact owned patch inverse before switching and reapplication afterward.
It rewrote no existing branch and retained native generated state. This branch
transition policy is part of this diagnostic protocol, not an assertion that
all developers switch branches on every commit.

## Causal finding and frequency

The [prior diagnostic reconstruction](../bv001/prior-causal-diagnostics.json)
binds C007/C008 action IDs through their executing-task parents to the native
task and original mutation requests. Those older observations selected the
case; they contribute no rows or saving credit to this audit.

The action in `ForbiddenPatternsTask.java` iterates the entire declared files
collection with `Files.lines`. Its registration binds that collection to the
source sets and required resource producers. Native input snapshots, execution
reasons and action-operation ancestry show three real input changes; all other
17 requests were `UP-TO-DATE`. No skipped request is counted as new product value.

| Ordinal | Changed inputs | Files scanned | Scanner action s | Native build s | Scanner finishes before last precommit dependency, s |
|---|---|---|---|---|---|
| 7 | 11 | 8,984 | 5.557 | 130.750 | 97.556 |
| 19 | 1 | 8,984 | 3.801 | 74.464 | 45.473 |
| 20 | 6 | 8,985 | 5.298 | 88.801 | 56.670 |

The causal hypothesis was to process changed files after a successful native
scan, while retaining full rescans for semantic changes, missing/invalid state
and failure recovery. The strongest alternative is supported by the prefix:
most requests already avoid the task, and other compiler/check dependencies
dominate requests that change its inputs.

## Admission arithmetic

Across the 20 transitions, native `Run build` spans total **502.069 s**; complete
command elapsed time totals **516.979 s**. The shorter native span is used as
the percentage denominator to favor the opportunity. The preparation anchor
and previous diagnostic scans receive no saving credit.

| Generous counterfactual | Total gross s | Gross s per scheduled transition | Gross native workflow reduction |
|---|---|---|---|
| Remove every scanner action, including required changed-file work | 14.656 | 0.733 | 2.919% |
| Remove the entire task on every request, including native fingerprinting | 15.735 | 0.787 | 3.134% |
| Required G3 admission floors | At least 25.103 for 5% of this native span | At least 1.000 | At least 5.000% |

Both counterfactuals assign **zero** discovery, delivery, validation, maintenance
and recurring product cost. Even the complete-task deletion falls below both
floors. The model that holds other task durations and explicit dependency edges
fixed predicts zero critical-path reduction from deleting the action. It does
not measure shared CPU, I/O or GC effects; no amplification mechanism was
established. No candidate was implemented or timed, and no confidence interval
or paired-value claim is manufactured from these native-only rows.

## Every engineering transition

Each row links by ordinal to the complete native receipt/source snapshots/log
under `native-requests/NNN/`. Raw Gradle traces and full diagnostics remain in
the owned state root and are hash-bound by [evidence manifest](./evidence-manifest.json).

| Ordinal | Revision | Changed task inputs | Native task outcome | Native build s | Scanner action s |
|---|---|---|---|---|---|
| 1 | 61470113b4cc | 0 | UP-TO-DATE | 7.304 | 0.000 |
| 2 | 8de19265331e | 0 | UP-TO-DATE | 5.889 | 0.000 |
| 3 | 620db26a7418 | 0 | UP-TO-DATE | 19.723 | 0.000 |
| 4 | 7bbd92d9d98a | 0 | UP-TO-DATE | 16.383 | 0.000 |
| 5 | f35bad5aaed0 | 0 | UP-TO-DATE | 4.598 | 0.000 |
| 6 | b653442570bc | 0 | UP-TO-DATE | 4.653 | 0.000 |
| 7 | 3a00a9167b54 | 11 | EXECUTED | 130.750 | 5.557 |
| 8 | a48778e0c3fc | 0 | UP-TO-DATE | 7.345 | 0.000 |
| 9 | 3b0e446a54e5 | 0 | UP-TO-DATE | 5.394 | 0.000 |
| 10 | 10176f3a95ef | 0 | UP-TO-DATE | 16.325 | 0.000 |
| 11 | 368a07ceed50 | 0 | UP-TO-DATE | 5.621 | 0.000 |
| 12 | 3acb36664e57 | 0 | UP-TO-DATE | 27.997 | 0.000 |
| 13 | 9aec7d25f20e | 0 | UP-TO-DATE | 18.320 | 0.000 |
| 14 | 48a69d226907 | 0 | UP-TO-DATE | 15.941 | 0.000 |
| 15 | d3c85a62f63a | 0 | UP-TO-DATE | 4.271 | 0.000 |
| 16 | 54f83a8180be | 0 | UP-TO-DATE | 16.519 | 0.000 |
| 17 | 1e1b40d2f3dd | 0 | UP-TO-DATE | 5.427 | 0.000 |
| 18 | d47f43f5ff06 | 0 | UP-TO-DATE | 26.344 | 0.000 |
| 19 | 72aa4d4120d0 | 1 | EXECUTED | 74.464 | 3.801 |
| 20 | 22d6425e9a44 | 6 | EXECUTED | 88.801 | 5.298 |

## Conditional H2 and the next observation

The bounded [H2 admission decision](../bv009/h2-admission.md) admits no repair.
The module-path provider already uses `@CompileClasspath`; the actual private
constant/static-initializer change at ordinal 19 leaves downstream module
compilers up-to-date. Checkstyle already uses an empty compiled classpath and
runs on genuine source changes. None of this proves every compiler or linter
operation minimal; it fails to establish either inspected input-repair cause.

`checkstyleMain` accounts for 184.178 s of task spans, and is on the last
precommit dependency chain at ordinals 19 and 20. Main/test/internal-cluster
Checkstyle spans total 316.072 s, with overlap. These task sums are **not**
recoverable workflow seconds. They motivate the separately
[unstarted next investigation](../next-investigation.md), beginning with native
Checkstyle capabilities and output semantics. Do not relabel source checking
as an H2 invalidation defect or reopen generic cache/annotation searches.

## Costs, retained failures and recovery

The phase charged **31 reservations / 30 actual Gradle invocations / zero
nested starts**. One initial reservation ended before Gradle. A prospective
continuation added one start to the initial 30-start ceiling under the owner's
standing budget authorization; all earlier charges remain intact. The failed
owner dependency-copy correction, successful complete dependency-cache copy,
and native Configuration Cache failure remain visible.

The complete native-prefix unit used 2,205.309 CPU seconds over 735.712 elapsed
seconds, including source inventories and trace processing; this is not an
isolated JVM CPU measurement. The 21 native commands used 538.748 elapsed
seconds. Source/trace bookkeeping is research cost, not an installed-product
overhead measurement. Systemd reported 28.6G peak memory and 26.6M peak swap.
See the [resource ledger](./resource-ledger.json), [journal](./native-prefix-unit-journal.log)
and [allocation checkpoint](./allocation-at-close.json). The footprint and free
space remain within the 120 GiB / 40 GiB guards. Active engineer time was not
measured. BV-002 dependency preparation used local dependency-only caches;
no scans, code or evidence were uploaded.

Recovery key: `BUILDOPT-VIABILITY-V1` in
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`.
The subject is retained at
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/worktrees/elasticsearch-native`,
branch `bv1-elasticsearch-native-prefix-020`, ordinal-20 SHA above. Shared Git is
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/eic-native-v1/repos/elasticsearch.git`.
Its only tracked delta is the two-line retained baseline correction. The
BuildOpt checkout stays on its original dirty `main`; no stage, commit or push
was performed. Old experiment worktrees remain untouched. The files under
`tools/` are immutable diagnostic-tool snapshots; this was not BV-005 runner
qualification and the old cohort must not be blindly rerun.
