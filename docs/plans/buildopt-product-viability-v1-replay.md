# BuildOpt Product Viability v1 Historical Replay Contract

Date: 2026-09-08. State: paired/adaptive executable contract remains unqualified;
the native-only engineering prefix is observed, with no candidate validation replay.
Parent: [plan](./buildopt-product-viability-v1.md).
Execution and evidence: [tracker](./buildopt-product-viability-v1-tracker.md).

The [approved Checkstyle admission](../../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md)
changes the next H1 subject, not this protocol's thresholds or held-out data.
The previous ForbiddenPatterns outcome stays negative. Keep its qualified
native cacheability delta in both future arms wherever its source binding holds;
the raw Checkstyle engine cache did not qualify as a compatible native setting.

This contract tests native correction H1, or a separately admitted H2 variant,
against repository evolution. It reuses the sound parts of the
[earlier longitudinal protocol](../../specs/poc-current-longitudinal-campaign-v1.md)
and changes the mechanism, horizon, economic questions, and product boundary.
It does not amend that campaign or the closed installed Elasticsearch study.

Sections 1-9 define the initial fixed-correction mechanism experiment. Section
10 adds the required adaptive product experiment: N/F/A comparison and causal
learning throughout validation. An unchanged correction is a control strategy,
not a restriction on the final product's learned state or selected action.

## 1. What is reproduced

The experimental unit is a repository revision, an exact requested workflow,
a declared state mode, and the state accumulated from earlier requests in that
same arm. A pair compares control and candidate at the same revision. Outputs
are compared across arms at that revision, never required to remain identical
between different commits.

Git records committed source evolution, not unsaved edits, actual developer
build frequency, branch-switch habits, historical machine load, or all former
artifact availability. Running historical code today is a backtest under a
declared environment, not a reconstruction of measured historical CI latency.
Dependency downloads, cache age and expiration also occur on today's clock;
an accelerated replay does not simulate a year of storage retention.

Choose one ordinary owner workflow per subject before seeing value. Use its
versioned CI/developer documentation as evidence. One invocation per transition
is the primary workload; synthetic edit/repeat bursts belong to a separately
labelled sensitivity experiment and cannot inflate primary request frequency.

## 2. Frozen history and development separation

The seed endpoint is Elasticsearch
`16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7` (the previously identified B0).
At BV-001, obtain its complete 101-commit first-parent ancestry: one anchor and
100 successive transitions. Do not assume existing local repositories are
unshallow or have every object.

Read-only enumeration after history availability is verified:

```bash
git -C "$buildopt_subject_git" rev-list --first-parent --max-count=101 --reverse 16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7
```

`buildopt_subject_git` must identify the exact verified subject Git directory,
not the BuildOpt checkout. Require 101 unique commit IDs and every adjacent
first-parent edge. Record tree IDs, parent IDs, UTC commit timestamps, changed
path digests, and actual span. Follow ancestry order even if timestamps are
non-monotonic. Do not select commits by changed files or known speedups.
[Git first-parent semantics](https://git-scm.com/docs/git-rev-list).

| Ordinals | Role | Permitted use |
|---|---|---|
| 0 | Cold anchor | Establish initial source/toolchain/cache state and account for startup |
| 1-10 | Initial engineering prefix | Qualify consecutive source advancement, capture and state preservation |
| 11-20 | Remaining engineering prefix | Diagnose, implement and validate the mechanism; estimate cost and freeze policy |
| 21-100 | Locked validation | Evaluate the frozen implementation/policy; no researcher tuning from these timings. The adaptive arm in section 10 may learn from completed observations under that policy |

The 100-transition window may span days rather than months. The report must
state the actual span and counts of task-input, dependency, build-logic and
toolchain changes. Missing event categories restrict claims; do not force a
historical event into existence or cherry-pick its replacement.

Elasticsearch is a known development subject. Its source has already been
inspected; do not call it an unseen repository or claim perfect research
blindness. The protected boundary is unconsumed validation outcomes and no
future-derived runtime state or repairs.

Before the seed's confirmation, freeze an ordered replication inventory of six
additional independent owners: Apache Groovy, Apache Kafka, Spring Framework,
OpenTelemetry Java, Micronaut Core, and Hibernate ORM. Resolve each repository,
declared default branch, immutable endpoint and 101-commit ancestry at the
inventory freeze, without candidate timings. Prior exposure to these families
must be disclosed; they are replication subjects, not previously unseen names.
Inspect only their engineering prefixes for selection. Retain all six outcomes.
Choose the first two with admitted causal opportunities in that fixed order;
if fewer than two qualify, G4 fails its breadth prerequisite. Do not add a
seventh family after inspecting results. A later unseen cohort requires a new
manifest and an explicit new claim.

## 3. Arms and modes

| Arm | Definition |
|---|---|
| N | Optimized native Gradle, including all compatible baseline settings/corrections frozen before timing |
| I | The same native baseline plus the exact incremental correction and any customer-required product processing |

For Elasticsearch, first qualify the retained cacheability correction as a
baseline improvement on the selected prefix. Record its exact source delta,
compatibility, setup cost, and owner tests. H1 must beat that baseline instead
of rediscovering its cache-restoration benefit. A stock-upstream diagnostic can
explain the baseline, but is not a third mandatory confirmation arm.

Primary mode P is a persistent workspace with private project state, outputs,
Gradle user home, native task cache, and daemon registry. Each arm keeps its
own state across commits. Daemon and configuration-cache policies are chosen
from supported native behavior during the prefix and then frozen symmetrically.
Record launcher, daemon, compiler, and tool runtime identities separately.

Mode E models ephemeral CI: create a fresh task-owned workspace for each job,
restore exactly the dependency/native-cache layers declared in its frozen CI
contract, and do not retain project output history unless that CI actually
does so. Run and report it separately after a positive P result. P success
authorizes only P claims; an E failure can narrow the product segment without
rewriting the P result. E uses the same two-arm, two-replication 100-transition
contract if evaluated, with a separate preflight allocation.

Neither arm borrows the other's writable state, output artifacts, native-cache
entries, or running daemons. Dependency artifacts may be prepared identically
from a separate acquisition area; the allowed copy list excludes task outputs,
native task-cache entries, project state, and BuildOpt learning. Preparation
cost and failures are retained even when excluded from paired execution time.

## 4. Source advancement and patch lifetime

For each arm, advance from its previous source tree to the next exact historical
tree while retaining only the declared generated state. Verify the full source
inventory and expected patch delta before every invocation. Generated files
tracked in Git remain source, even when located under an output directory.

In a task-owned candidate worktree, verify and reverse only the exact owned
patch before changing the historical source revision; then apply the frozen
correction if its declared preconditions still match. Never reset, clean, or
overwrite user work. A tracked/untracked collision, unexpected source drift,
or ambiguous worktree identity is an explicit refusal, not a reason for a broad
cleanup. Use the repository Git procedure for the authorized worktree actions.

A source or toolchain change outside frozen applicability produces
`NATIVE_RETAINED_UNSUPPORTED_CHANGE`, with the native workflow and the measured
refusal cost. No ad hoc human repair is inserted into locked validation. Record
the changed predicate, last applicable ordinal, and requalification requirement.

The candidate may use only state produced by its own earlier ordinals. Do not
seed a pre-trained final profile, future cache entry, or later patch. Freeze
detector rules, patch compiler, qualification policy, and any model version,
prompt or stochastic seed used by the candidate. Researcher changes to those
rules invalidate affected proof and start a separately identified experiment
version. Policy-authorized learning and generated corrections in the adaptive
arm are recorded state transitions, not researcher changes to the policy.

## 5. Correctness and native failures

Freeze the required-output and diagnostic contract before candidate execution.
Bind artifacts to their producer tasks and include the complete owner workflow,
not only a marker from the changed task. Preserve raw bytes, modes, paths,
symlink targets and relevant presence/absence. Structured failure comparison
includes exception category, rule, file, line, problem IDs and native exit.

Qualify any nondeterministic field against independent native controls and its
generation source before it becomes an allowed comparison rule. A newly seen
candidate mismatch cannot create a new normalization. Unexplained differences
stop the candidate. Existing EIC comparison rules require an applicability
audit; their old approval does not automatically cover H1 or older Gradle/JDKs.

Required H1 scenarios are enumerated in BV-004. Observe full rescans after
semantic rule changes or missing incremental history, and changed-file work
after a successful ordinary baseline. A previous failed scan must not let an
unchanged violation disappear on the next attempt.

Classify every planned ordinal as one of:

- `COMPARABLE`: native and candidate complete; required results verified.
- `NATIVE_RETAINED`: candidate safely uses native behavior; this remains a
  comparable result and its overhead counts.
- `NATIVE_BUILD_FAILURE`: native workflow itself fails; preserve diagnostics.
- `DEPENDENCY_UNAVAILABLE`: declared preparation cannot obtain required inputs.
- `NATIVE_NONDETERMINISM`: native controls cannot establish the required contract.
- `HARNESS_INVALID`: capture, source, state, timing or ownership proof is broken.
- `CANDIDATE_FAILURE`: candidate-only error or semantic mismatch; stop for safety.
- `NOT_RUN_DEPENDENCY` or `NOT_RUN_LIMIT`: retain each unexecuted ordinal and reason.

No slower result, missing optimization, or inconvenient change shape is an
exclusion. There are no reserve commits to replace unsuccessful ordinals.
When a native failure permits advancing safely, retain its natural resulting
state and proceed to the next original ordinal; otherwise mark the remainder
unrun. An infrastructure retry is limited to the manifest's declared reserve,
requires an equal pre-attempt checkpoint for both arms, and never erases its
failed attempt or charge. Candidate failure is not an infrastructure retry.

## 6. Run order, replication and measurement

After engineering, freeze the candidate and recreate fresh independent N/I
state for confirmation. Run two complete replications, each from ordinal 0
through 100. Earlier engineering state is not a confirmation checkpoint.
The cold anchor and 20 prefix transitions are rerun and charged in each
replication; only ordinals 21-100 supply confirmation savings.

Execute arms sequentially on fixed resources. Replication 1 alternates N/I
and I/N by ordinal, starting N/I; replication 2 reverses that order. Rotate
assigned physical roots between replications where qualified. Keep competing
builds out of the measurement lane without terminating unrelated processes.
Record affinity, CPU/memory constraints, filesystem, clock identity, host load,
temperature/throttling when available, and daemon/cache fingerprints.

Measure external monotonic time for the complete customer request envelope,
including required product pre-work and finalization. Record native child time
separately. Deep task traces are for attribution; qualify their overhead and
use identical instrumentation in both arms. Retain lean primary timing and
selected diagnostic captures separately if full tracing is intrusive. Never
add parallel task durations and present the sum as workflow savings.

Output capture and the independent experiment checker run outside the timed
request unless the actual product requires them on every request. Their cost
belongs in the research ledger, or the customer ledger when the product really
incurs it. No customer-required cost is removed merely by running it in a
background process; wait for completion and charge its resource usage.

Scheduled confirmation starts per subject are `2 replications * 101 revisions
* 2 arms = 404` outer workflow starts. Add dependency preparation, diagnostics,
correctness, nested TestKit calls and retry reserves to size the actual total.
Count nested starts separately without adding their elapsed time a second time
to the parent's duration.

## 7. Metric formulas and progression

All durations use integer nanoseconds in raw records and seconds in reports.
For one replication, let `N_i` and `I_i` be complete comparable request-envelope
durations. A positive `delta_i = N_i - I_i` means the candidate is faster.

Define `C_adopt` as customer machine work for discovery, verification and adoption
outside those envelopes, plus `max(0, sum(I_i - N_i, i=0..20))`. This conservatively
charges excess startup/prefix cost while giving no confirmation credit for
engineering-prefix speedups. Define `M_i` as customer maintenance work outside
the request envelope at validation ordinal i. Every charged phase has a unique
ID so it cannot be counted in both `I_i` and `C_adopt` or `M_i`.

Freeze the cost classification before confirmation. Treat ambiguous required
per-customer work conservatively as customer cost until the actual delivery
path resolves it; do not relabel qualification as research simply to pass G3.

For comparable validation ordinals through j:

```text
net_saved(j) = sum(delta_i, i=21..j) - C_adopt - sum(M_i, i=21..j)
net_fraction = net_saved(100) / sum(N_i, i=21..100)
net_per_scheduled_transition = net_saved(100) / 80
```

Non-comparable ordinals earn no saving. Deduct their identifiable extra product
work; where an attempted candidate has no valid comparison, conservatively
charge its entire envelope and disclose that treatment. Report both scheduled
and comparable denominators. A valid complete outcome requires every ordinal
classified, at least 76/80 comparable validation rows per replication, and no
unexplained correctness or harness gap. A partial horizon cannot pass G3.

Operational payback is the first validation ordinal after which `net_saved`
stays nonnegative through ordinal 100. Report temporary crossings separately.
Unrepaid adoption cost is a negative result, not an extrapolated observed payback.
Report active human time separately and incorporate it into the later monetary
customer model. Include a fully loaded research-cost sensitivity without
pretending research is free or charging it as recurring customer work.

G3 requires, in **each replication**:

- At least 5% net reduction and 1 s net per scheduled validation transition.
- Operational payback reached and retained before the fixed horizon ends.
- A strictly positive lower 95% bound for adjusted mean validation saving,
  using the dependence-aware method below.
- Candidate request p95 no greater than native p95 plus
  `max(1 s, 0.05 * native p95)`. Retain all individual regressions and the worst
  case; passing p95 does not hide a correctness or corruption event.
- Zero candidate-attributable correctness failures and valid coverage as above.

For uncertainty, resample consecutive non-overlapping blocks of five ordinal
slots, 10,000 times with seed `20260908`, within each replication. Include
missing-slot conservative charges and maintenance costs in their original
slots. Subtract fixed adoption cost once per bootstrap replicate. Report
2.5th/97.5th percentiles. Repeat the predeclared sensitivity with blocks of ten;
if the lower-bound sign changes, classify the result as inconclusive. The
method is a robustness check for this history; commits are correlated, and
two replays do not create two independent repository populations.

Compute p50/p95 by the nearest-rank definition. Report per-repository results
before portfolio totals. Do not average percentages across different workflow
sizes or let a large seed win hide losing replication subjects. Report action
rate, no-action overhead, changed-input rate, skipped/full-scan counts, bytes
processed, correction applicability, state-loss events and source-change classes
as explanatory metrics.

Checkpoint every 20 transitions for correctness, completeness, allocation and
resource health. A negative interim cumulative value is not a stopping rule.
Finish the frozen horizon unless safety, validity, or resource limits require
an explicit partial result. Statistical checking must not trigger opportunistic
extra repetitions when a result almost passes.

## 8. Record and recovery contract

BV-005 must implement a versioned schema and independent checker for these
artifacts. These are required future artifacts, not files that exist today.

| Artifact | Required contents |
|---|---|
| `manifest.json` | Protocol version/hash, hypothesis, mechanism identity, full working-tree/package digests, all subject/parent/tree/workflow/toolchain bindings, mode, predicates, thresholds, order, output contract, allocation and freeze time |
| `subjects.json` | Ordered six-family replication inventory, selection inputs from prefixes, conclusive/unavailable/no-opportunity outcomes and selected two subjects |
| `attempts.jsonl` | Immutable attempt ID, subject/replication/ordinal/arm, before/after state digests, exact command/environment allowlist, process ownership, timings, exit/signal, task outcomes, output receipts, classification, raw artifact digests and charged phase IDs |
| `costs.jsonl` | Phase ID, customer-machine/customer-human/research class, start/end, duration/resource usage, monetary inputs when observed, failed/nested start counters and attribution to envelopes |
| `checkpoint.json` | Last fully verified pair, both state digests, remaining frozen ordinals, immutable manifest/package pins, last attempt, cumulative charge, process/watcher ownership and next action |
| `result.json` | Recomputed counts, coverage, per-replication metrics, uncertainty, payback, guardrails, decision and all required-but-unrun work |
| `report.md` | Full command/scope, actual dates and transitions, curves/tables or linked standalone figures, negative results, limitations, economic interpretation and next decision |

The checker must reconstruct parent edges, ordering, artifact hashes, attempts,
classifications, metrics and costs from raw records, then compare with the
claimed summary. It must reject missing/duplicate/reordered rows, forged
duration/decision summaries, future state, shared mutable roots, missing nested
starts, changed source/package/contract bytes and unqualified normalization.

Recovery verifies repository and shared Git identity, all bound inputs, both
arm states, child/service ownership, last complete pair and spent limits. A
half-completed pair is partial; resume only through the declared equal-state
recovery rule, never by treating the completed arm as a fresh timing partner.
If the original state cannot be reconstructed, close that replication as
incomplete and retain its cost. Old roots are never overwritten for a clean log.

## 9. Maintenance and extension boundaries

The fixed replay measures correction survival. The required H3 trial below
measures automatic detection, search, validation, replacement and net value.
Use naturally observed events for recurrence claims;
fault injection can prove detection/suspension/recovery but not event frequency.
Preserve the original frozen-patch result even if a later maintained version
does better. Prevent alerts on environment-only changes from becoming alleged
source regressions through native counterfactuals and source bindings.

An extension is justified by missing calendar/event coverage after positive
seed and replication results, not by a desire to wait until a losing average
turns positive. Freeze its endpoint/count, version, workload, state, cost and
claim separately before execution. Customer pilots must additionally measure
actual command frequency and accepted review/maintenance effort.

## 10. Adaptive product evaluation

BV-011 owns this required experiment. It does not promote the mechanism-only
BV-007/BV-008 results into an adaptive-product result. Register a separate
`experimentKind: ADAPTIVE_NFA` manifest, candidate package/controller policy,
cohort and complete allocation before running it.

### 10.1 Cohort and the 20/80 boundary

Use the seed and both G4-selected repositories. Each contributes one anchor
and 100 consecutive first-parent transitions, with 20 engineering transitions
and 80 validation transitions. Freeze this cohort before developing or tuning
H3, without consuming its validation outcomes. Do not reuse the already-read
BV-007/BV-008 validation rows as an unseen controller test.

Selection uses Git metadata alone: prefer the next complete 100-transition
block after that subject's mechanism endpoint if all objects already exist;
otherwise take the closest earlier complete block with no overlap in measured
transitions with any consumed experiment. An anchor may be shared, but no
learned state is shared. Retain the selected exact SHAs and all inspected
candidate ranges. If neither block is available, mark the cohort incomplete;
do not select by observed savings or manufacture new source commits.

Earlier source exposure remains disclosed. This is a retrospective test under
a declared product version, not a claim that the version existed at the time
of those commits. Natural change-event coverage is reported after the replay;
missing events cannot be inferred from injected tests.

Use the 20 engineering transitions to fix the controller implementation,
policy parameters, permitted action classes, model/prompt version if applicable,
budget and decision rules. During the 80 validation transitions the product
continues to observe, update its state, search, validate and replace corrections
under those rules. Researchers may not change its policy, hand-pick a successor
or repair a failed case from later outcomes. A decision at ordinal i can use
its own available prefix and charged probes at i; the completed request outcome
can update decisions for subsequent requests. Every evidence receipt records
when it became available to the controller.

### 10.2 Arms and isolation

| Arm | Behavior |
|---|---|
| N | The frozen compatible optimized native baseline, with no adaptive product |
| F | That baseline plus the one initial qualified correction, kept only while its source/semantic contract remains valid; no replacement search |
| A | That baseline plus the adaptive product, allowed to maintain, suspend, retire, repair or replace corrections under the frozen policy |

F corresponds to the fixed I strategy in the initial experiment, qualified
anew at this cohort's prefix. Freeze the same initial correction and make it
available to F and A at the same ordinal after charging their own acquisition
cost. If it is inapplicable at the new anchor/prefix, report that prerequisite
failure; do not silently choose a more favorable initial correction. A must
identify and qualify later alternatives itself. A may retain a valid correction
while exploring an alternative; replacement is not forced merely by a slow run.

All three arms have independent mutable roots, caches, daemon registries and
state. N/F timing, outputs and future traces belong to the evaluator and are
not free training data for A. An A-initiated native counterfactual must use an
owned comparable checkpoint at the same revision/workflow/state mode; charge
its complete acquisition/execution cost and any contention or waiting. No
comparison of today's changed workload against yesterday's total duration can
by itself establish lost optimization value.

Run two fresh replications. Replication 1 cycles through `NFA`, `FAN`, `ANF`,
`AFN`, `FNA`, `NAF` by ordinal. Replication 2 reverses the selected order at each
matching ordinal. Runs remain sequential; no arm gets warmed by another arm's
writable state. The schedule has `2 * 101 * 3 = 606` outer workflow starts per
subject before dependency preparation, controller probes, qualification,
fallback reruns and nested TestKit starts. Size all of those before launch.

### 10.3 Controller policy and state transitions

The executable policy must give concrete values and units for every field in
this table before validation. Their derivation uses prefix measurements and
the explicit planning constraints, not validation outcomes. Unset, inconsistent
or post-hoc policy values fail admission; no new numerical threshold is claimed
as experimentally qualified by this document.

| Policy field | Required definition and proof |
|---|---|
| Applicability predicate | Source/input/API/toolchain contracts that require immediate suspension when violated; distinguish legitimate full work after state loss |
| Comparison key | Exact workflow, state mode, toolchain/resource conditions and valid native counterfactual; invalidate incomparable windows |
| Evidence window | Minimum comparable observations, maximum evidence age and repeated-window count; insufficient evidence is not zero saving |
| Activation and retention bounds | Net-benefit/confidence rule with activation stricter than suspension; calibrate noise and repeated decisions on native-only prefix/fixtures |
| Probe policy | When native probes are permitted, maximum frequency and cost, evidence freshness and state reconstruction |
| Search admission | Proven incompatibility, sustained economic decline or a new measured material opportunity; allowed H1/H2 transformation classes only |
| Search budget | At most two candidates per episode; bounded elapsed time, native starts and total compute; no concurrent searches for the same workflow |
| Retry/backoff | Cooldown in observed requests/time, maximum episodes over the campaign and the new evidence that permits reopening a failed search |
| Qualification and switching | Required correctness/economics checks, expected payback horizon, atomic application/inverse and cancellation/recovery behavior |
| Human boundary | Preauthorized automatic actions versus review-required proposals; record all interventions, wait time and automatic completion rate |

`OBSERVE` keeps a supported profitable correction with bounded overhead.
`SUSPECT` collects comparable evidence of economic deterioration; a single
slow build or machine-load spike cannot authorize replacement. Confirmed loss
or a new admitted opportunity enters `SEARCH`, then `VALIDATE`. Successful
qualification enters `ACTIVE`; rejection or episode exhaustion enters
`COOLDOWN`. An invalid applicability or correctness condition enters
`SUSPENDED` immediately, bypassing economic windows. Resume the native workflow
through the verified inverse/fallback path; do not force an inverse onto
unexpected user source edits.

Search and validation may run outside the foreground build within the owned
resource policy, but all their time/resource cost, interference, failures and
retries remain charged. Keep a still-useful correction or native Gradle during
search. If no alternative qualifies, do not claim a saving and do not keep
searching every commit. Cooldown expiry requires the policy's evidence checks.

The controller cannot broaden H2 admission, reenable retired mechanisms, weaken
input/output proof or expand permissions. Autonomous adaptation means executing
the authorized policy; arbitrary new source rewrites remain review-required.
Manual per-commit repairs disqualify an automatic-completion claim even when
the resulting patch is correct.

### 10.4 Required correctness scenarios

BV-011 must qualify each state and transition before public replay, including:

- Stable useful correction: low observation overhead, no unnecessary search.
- Normal changed input/cache miss/lost history: required native work without
  a false semantic-invalidity decision.
- Genuine source/API incompatibility: immediate suspension and safe native
  continuation, with no stale correction execution.
- Noisy slow build or environmental shift: no unqualified switch or search storm.
- Sustained decline: correct detection after the configured evidence window.
- Newly material opportunity while the current correction still works:
  bounded exploration with continued useful execution.
- Valid alternative: candidate discovered, independently qualified and switched;
  exact inverse and full owner-output/failure semantics preserved.
- Wrong-output alternative or failed qualification: rejected before activation,
  costs retained and fallback preserved.
- No profitable alternative or exhausted episode: native/useful prior correction
  retained, backoff observed, and no immediate retry loop.
- Oscillating near-threshold evidence: hysteresis/cooldown prevent repeated switches.
- Restart/cancellation during search, verification or switching: recover exact
  state/charge identity, cancel owned children and avoid duplicate activation.
- Undeclared N/F evidence, future input, budget overrun or researcher repair:
  independent checker rejects the adaptive proof.

An unsafe candidate rejected inside qualification is a recorded search failure,
not a successful optimization. An unsafe candidate activated into the customer
path is a correctness failure that stops the adaptive experiment.

### 10.5 Metrics and G5 acceptance

Apply section 7 separately to F and A versus N, with their own `C_adopt`,
request durations and outside-envelope costs. For A those costs include all
observation, native probes, search, verification, switching, failed candidates
and retries. Never charge a phase twice or omit it because it is asynchronous.
Maintain the same scheduled/comparable denominators and complete-horizon rules.

```text
adaptive_incremental_value = net_saved_A_vs_N(100) - net_saved_F_vs_N(100)
```

Recompute its uncertainty by resampling the same ordinal blocks jointly across
N/F/A, using the section 7 seed, block sizes and fixed-cost treatment. Report
each replication/subject before aggregates. Include detection delay, time
without a useful correction, false alarms, switch count, automatic episode
completion, human interventions, search cost per accepted correction and costs
of no-opportunity episodes. Ground-truth event labels are independently derived
from source/contracts/native evidence; they are not supplied free to A.

Full G5 requires qualified controller/delivery correctness and complete outcomes
for all three scheduled subjects. A must pass G3 versus N in at least two
subjects. For histories with natural adaptation events, require positive net
incremental value versus F and a positive lower 95% bound in both replications;
demonstrate at least one successful autonomous natural-event recovery or
replacement in each of the two subjects supporting the adaptive claim. Retain
stable/no-event subjects and enforce their declared observation/search budget
and section 7 p95 guardrail; do not require a needless switch to manufacture
an event. Total adaptive management cost must remain below the preregistered
20% of gross A-versus-N workflow saving over the same interval. Human effort
and monetary support cost remain separate inputs to G6.

With zero natural events, controller functionality and stable overhead can be
verified, but adaptation benefit and recurrence are `not measured`. With no
profitable alternatives, record `NO_PROFITABLE_REPLACEMENT`; with manual help,
record assisted rather than autonomous recovery. A negative or inconclusive
adaptive result does not erase a positive fixed-mechanism result, and cannot
pass the self-managing product gate. An explicitly experimental customer pilot
may gather missing event evidence after functional proof; full G5 still needs
the equivalent costed chronological evidence before the final product claim.

### 10.6 Adaptive evidence and recovery

Extend the section 8 artifacts with `controller-policy.json`,
`controller-events.jsonl` and `adaptive-result.json`. Bind policy hashes,
state before/after, evidence-availability time, applicability predicates,
window/probe inputs, candidate IDs, accepted/rejected qualification receipts,
switch/inverse receipts, cooldown state, manual interventions and unique charge
IDs. The manifest distinguishes mechanism `FIXED_NI` from product `ADAPTIVE_NFA`.

Checkpoint all three arms and A's pending jobs, consumed episode budget,
evidence window, current correction, cooldown and last verified transition.
The checker reconstructs decisions under the frozen policy and rejects future
evidence, N/F oracle access, uncharged probes, hidden interventions and missing
states or failures. Resume only a fully bound state; a partial three-arm round
follows the equal-state recovery rule and never receives fresh-row credit.
