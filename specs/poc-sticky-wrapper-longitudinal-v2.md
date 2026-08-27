# Sticky wrapper longitudinal campaign v2

Status: preregistered POC measurement contract; blocked by `SWL-014A..D`.

## Why v2 exists

The retained v1 diagnostic proved that the committed wrapper could execute five
substantial public builds and preserve their required outputs. It did not test
the sticky-learning hypothesis: the runner added `--build-cache` only to the
control, configured no server or project identity, set the trial budget to zero
and therefore exercised the candidate's no-op/light-observation route.

That evidence remains immutable and useful for compatibility. It is
`DIAGNOSTIC_ONLY` and cannot contribute to this protocol or the terminal
`SWL-016` decision.

## Preconditions

Timing cannot start until all four gates pass:

1. `SWL-014A` proves arm symmetry and a checker that rejects timing-only
   readiness.
2. `SWL-014B` connects observation, proposal, trial, signed decision, active
   execution, suspension and economics to the real `./buildoptw` path.
3. `SWL-014C` finds independently testable generic actions in at least three of
   the five frozen public families.
4. `SWL-014D` proves positive conservative value for those actions through the
   installed wrapper with exact outputs and complete cost attribution.

Failure of the breadth or installed-value pre-gate stops the experiment before
the longitudinal compute budget is spent. It does not itself emit the terminal
POC outcome: it records immutable stop evidence, marks the later measurement
blocks as skipped and opens `SWL-016`, which remains the only terminal
authority.

## Autonomous execution boundary

This protocol and the detailed tracker are the implementation contract. A
later agent must not rename the frozen files, add a detector, choose a different
statistics method, share a writable cache namespace between arms, move a
threshold or bypass a failed pre-gate. The machine contract contains the exact
file manifest and focused validation commands for every remaining block.

`SWL-014A` is partly complete. The v2 contract, v1 diagnostic classification,
v1 false-readiness rejection, documentation alignment and the decisions in
this section are complete. The remaining work is exactly the v2 runner with a
`--preflight-only` mode, its lifecycle-aware checker, one valid zero-pair
fixture, four named negative fixtures and the checked preflight result. No
other `SWL-014A` work is implied.

All remaining evidence uses RFC 8785 JCS, rejects unknown fields, writes
canonical UTC RFC 3339 nanosecond timestamps, lowercase hexadecimal SHA-256
digests, signed 64-bit nanosecond durations and repository-relative paths with
forward slashes. An unavailable value is typed explicitly and is never filled
with zero.

## Fair arms

For a given repository revision, control and candidate receive the same frozen
Gradle argument vector. Cache enablement is part of that vector and is never
injected into only one arm. Both arms receive equivalent cache opportunity but
never share writable cache state. Each arm gets a separate empty remote
namespace before the first measured pair. The two
namespaces use the same service, TLS route, resource limits and read/write
policy; writes made by one arm are never visible to the other arm.

Each family/arm has a separate checkout, Gradle user home, daemon registry,
Configuration Cache and local Build Cache. Those roots persist chronologically
within that arm while the checkout advances through the frozen commits.
Workspace outputs also persist unless the frozen customer workflow explicitly
cleans them. The control receives an empty inert BuildOpt state root; only the
candidate's private decision/evidence root persists across commits.

Dependency and Wrapper distribution preparation is untimed and produces an
identical read-only dependency seed for each arm. It must not copy task outputs,
Configuration Cache entries, post-start local Build Cache entries or BuildOpt
state. Both local Build Caches also start empty. The preflight result records
both empty-seed digests and both remote namespace identities so the checker can
reject cross-arm contamination before timing.

Wall time uses an external Go monotonic clock from immediately before direct
command start until command wait and required process-tree cleanup complete.
Wrapper, decision, network/state, action, Gradle and required output
verification before the next arm all count. Checkout advance,
dependency/distribution preparation, empty namespace creation and final result
serialization do not. The runner records OS, architecture, logical CPUs, CPU
quota, memory limit, Java and Gradle. The POC does not force four CPUs, but a
fingerprint change within a pair excludes it as
`EXCLUDED_RUNNER_RESOURCE_DRIFT`.

The arms are:

```text
control:   ./gradlew  <identical frozen arguments>
candidate: ./buildoptw <identical frozen arguments>
```

Order alternates by comparable pair. Candidate state is bound to the exact
repository, workflow, Wrapper, toolchain and action generations.

## Customer-path composition

`SWL-014B` adds one composition root at
`internal/launcher/sticky_learning.go`; `internal/launcher/run.go` only calls
that root. It adapts, rather than duplicates, the existing owners:

| Responsibility | Existing owner |
| --- | --- |
| ordinary evidence | `internal/stickyobservation` |
| isolated paired trial | `internal/stickytrial` |
| typed records, signed decisions and ledger | `internal/stickydecision` |
| active execution, counterfactual and suspension | `internal/stickyactive` |
| durable proposal | `internal/durablecatalog` |
| customer report | `internal/stickywrapper/status.go` |
| shared qualification statistics | new `internal/stickyvalue` |

`internal/stickyvalue` exposes the single frozen entry point
`Evaluate(pairs []Pair, costs Costs) (Evaluation, error)`. The exact pair, cost
and evaluation fields are listed in the machine contract. Arithmetic is
checked signed 64-bit integer arithmetic; an unknown or missing cost produces
`UNAVAILABLE` and cannot qualify an action. A later agent may choose only
unexported helper signatures, local error wrapping and fixture identifier
values.

Every customer-triggered native, candidate and counterfactual process remains
owned by `internal/launcher.executeChildWithReserved`; WS-002 process groups,
signal forwarding and exit codes cannot regress. `SWL-014B` adds
`stickyactive.Runner.RunWithExecutor` and
`stickytrial.RunPairedWithExecutor`, while the existing `Run` and `RunPaired`
methods delegate to their current direct executors for compatibility. The
composed customer path may not call `os/exec` directly.

An ordinary invocation loads the exact binding, selects a verified local
decision without blocking on the network, executes the exact active runtime
action or native Gradle, appends its bounded observation and writes a ledger
entry only when counterfactual value exists. Refresh remains asynchronous.

Trials are never inferred from generic `CI=true`. They require all of: committed
mode `auto`, a non-zero committed trial budget,
`BUILDOPT_STICKY_LEARNING=1`, a token carrying `STATE_WRITE`, and the existing
`CACHE_READ`/`STATE_READ` capabilities. The learning variable and token are
scrubbed before Gradle. Trusted learning loads a generic proposal, records the
shadow transition, reserves the exact committed budget up to five percent, runs
isolated alternating pairs, evaluates value, then publishes a signed qualified or suspended decision
by generation CAS. Mode `observe` never trials or activates; mode `off` and
`BUILDOPT_BYPASS=1` remain native before state access.

Durable actions remain review-only. BuildOpt emits an exact apply/revert
transaction outside the long-lived checkout; an owner applies the reviewed
change. A later wrapper invocation may recognize the bound postimage, but it
runs plain native Gradle. BuildOpt never automatically merges or modifies the
customer branch.

## Frozen detector catalog

`SWL-014C` may use only these existing generic detectors, in this order:

1. `TASK_CONTRACT_JAVA_V1`, implemented by
   `internal/durablecatalog.TaskProposal`, with recipe
   `CUSTOM_TASK_CONTRACT_JAVA_V1`;
2. `DECLARED_GRAPH_SCOPE_V1`, implemented by
   `internal/durablecatalog.DetectGraphBreadthOpportunity`, with recipe
   `DECLARED_GRAPH_SCOPE_V1`.

The task-contract detector currently has no generic public input producer; its
public result is therefore
`INPUT_UNAVAILABLE_NO_GENERIC_SOURCE_PRODUCER`. This route cannot add a source
parser or turn the synthetic fixture into public evidence. The graph detector
uses only the installed `profile outputs`, isolated `profile input --confirm`,
`profile propose` and `gradlecriticalpath.Analyze` pipeline with the frozen
cohort inputs. Unsupported or incomplete proposals remain visible unavailable
results.

No repository-name, task-name, path-extension or manually authored evaluated
profile rule is allowed. Each proposal requires three exact recurring
observations. Task-contract projection uses the minimum eligible task duration
among the latest three exact recurrences. Graph-scope projection uses the
minimum omitted critical-path contribution among its latest three exact
recurrences. Non-positive projections reject. Estimated compatible builds to
repay are:

```text
ceil((trial cost + validation cost + publication cost) / projected saving)
```

and must be no more than 30.

For each graph candidate, the screen runs full and proposed commands with
operation/task-graph traces, requires the frozen outputs, and uses the minimum
omitted critical-path contribution across three recurrences. The exact action
identity is the contract's SHA-256 over detector, repository scope, workflow
arguments and candidate plan. No fallback detector is selected after results
are known.

## Frozen statistics

The installed active-path gate uses eight balanced alternating pairs per
action. A pair effect is `native wall ns - candidate wall ns`. Its confidence
interval is the existing deterministic 4,096-replicate 32-bit LCG paired
bootstrap: replicate `r` starts at
`2654435761 * (r + 1) mod 2^32`, each draw advances with
`1664525 * state + 1013904223 mod 2^32`, and sorted indices 102 and 3993 are the
two-sided 95-percent bounds. Passing requires a positive mean, positive lower
bound, a strict majority of positive pairs, exact/reviewed outputs, zero
product failures and nearest-rank candidate p95 no greater than native p95.

Longitudinal family confidence uses signed net saving after every BuildOpt
cost, not gross task saving. With at least 15 chronological values, it performs
10,000 deterministic circular moving-block bootstrap replicates. Block length
is `ceil(sqrt(n))`; block starts are sampled with xorshift64 steps `(13, 7, 17)`
and each block preserves chronological order with circular wrap. The seed is
the first eight big-endian bytes of
`SHA-256("buildopt-sticky-wrapper-longitudinal-v2\\0" + familyKey)`; a zero seed
is replaced by `0x9e3779b97f4a7c15`. Sorted index 499 is the one-sided fifth
percentile lower bound and must be positive.

## Exact remaining artifacts

The machine contract freezes every created file. The primary outputs are:

- `SWL-014A`: `dev/run-sticky-wrapper-longitudinal-v2`,
  `dev/check-sticky-wrapper-longitudinal-v2`, its fixture test and
  `benchmarks/results/sticky-wrapper-longitudinal-v2-preflight.json`;
- `SWL-014B`: the launcher composition, `internal/stickyvalue`, lifecycle
  checker/spec and `sticky-wrapper-learning-lifecycle-v1.json`;
- `SWL-014C`: the public detector screen, checker/spec and
  `sticky-wrapper-opportunity-gate-v1.json`;
- `SWL-014D`: the installed active-value runner, checker/spec and
  `sticky-wrapper-active-value-v1.json`;
- `SWL-015`: `benchmarks/results/poc-sticky-wrapper-longitudinal-v2/`; and
- `SWL-016`: the independent terminal checker/spec and
  `sticky-wrapper-terminal-decision-v1.json`.

Each block uses the focused commands listed in the machine contract, then the
repository tracker, documentation, layout, lint and static-CI checks. A
completed block is committed and pushed independently; if remote access is
unavailable it is reported blocked with the validated local SHA rather than
silently treated as complete.

The machine contract also freezes every evidence shape so an implementation
agent does not invent one while coding. `SWL-014A` emits a zero-pair preflight
with two arm records, explicit writable-root/namespace comparisons, learning
authority and `VALIDATED_NOT_READY`. `SWL-014B` emits ordered lifecycle
transitions, ordinary invocations, negative native-retention cases and a
recomputed ledger. `SWL-014C` emits detector results and cost/payback-complete
actions per family. `SWL-014D` emits the eight raw pairs and independently
recomputable interval, p95 and cost fields per action. `SWL-015` has exactly
the ten named result files and the frozen row/report keys. `SWL-016` accepts
only longitudinal-ready, opportunity-stop or installed-value-stop evidence.
The exact keys, schema versions and outcome strings live under each
`implementationManifests[].resultContract`; renaming or extending them is a
contract revision, not an implementation choice.

## Required candidate evidence

Every candidate row records:

- selected lifecycle state and execution decision;
- decision generation, action identifier and binding digest;
- whether no-op, observation, shadow, trial, runtime action or reviewed durable
  action actually executed;
- complete wrapper, decision, observation, trial, cache, fallback, action,
  verification and Gradle phase costs;
- gross saving, total cost and signed net value recomputable from immutable
  records;
- output equivalence and product-attributable outcome; and
- native counterfactual or the exact reason it was not due.

Inactive, shadow, rejected or suspended actions receive zero saving. A cache hit
is an observed mechanism fact and never action authority.

## Campaign and terminal boundary

The five families, current package, chronological primary/reserve commits,
workflows, outputs, toolchains and confidence method are frozen before timing.
The target is 20 comparable requested builds per family and the minimum is 15.
Every positive, negative, no-action and excluded row remains visible.

Sample count alone cannot produce `READY_FOR_SWL_016`. Readiness additionally
requires completed preconditions, symmetric arms, exact/reviewed outputs, zero
product-attributable failures, complete lifecycle evidence, deterministic
checkpoints and a signed economic ledger.

If fewer than three families pass `SWL-014C`, that block closes with stop
evidence, `SWL-014D` and `SWL-015` become
`SKIPPED_BY_SWL_014C`, and `SWL-016` opens. If fewer than three pass
`SWL-014D`, `SWL-015` becomes `SKIPPED_BY_SWL_014D` and `SWL-016` opens. Only
`SWL-016` may emit `STOP_STICKY_WRAPPER_LEARNING_POC`.

The immutable terminal scorecard remains the one in the
[Sticky Wrapper Learning POC Tracker](../docs/plans/sticky-wrapper-learning-poc-tracker.md).
No threshold is changed by this correction. Soak testing, design-partner work,
production authority, automatic patch merge and Test Optimization remain out
of scope.
