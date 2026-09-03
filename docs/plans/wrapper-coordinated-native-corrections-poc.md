# Wrapper-Coordinated Native Corrections POC

## Status

**Overall:** `CONTRACT_FROZEN`<br>
**Current block:** `WCNCP-004` is next (`WCNCP-001` through `WCNCP-003` are `DONE`).<br>
**Reference baseline inspected while writing this plan:**
`b8a195dc9fd55a52c15202b77f0bb979f8c97fa3`.<br>
**Execution authority:** `WCNCP-000` freezes contracts only. No public
repository patch, prospective build, candidate, new timing, owner contact,
proposal application, merge, or product-value claim exists.

## Decision

BuildOpt will test a new product unit:

> A repository-committed `buildoptw` remains the ordinary Gradle entrypoint,
> executes the repository's native Gradle Wrapper, records bounded evidence,
> coordinates multiple runners through an owner-operated backend, detects a
> material native-build defect, produces and validates a small reversible
> correction, and presents the evidence-bound proposal for owner review.
> Once accepted and applied by the owner, the repository continues using
> native Gradle through the same wrapper. BuildOpt is not inserted into the
> corrected task's steady-state execution path.

The wrapper is the durable onboarding and observation boundary. The backend is
the shared memory and coordination boundary. Gradle remains the build engine.
The reviewed source correction, not a dynamic runtime accelerator, is the unit
of customer value.

## Why this experiment exists

The current repository has established four relevant facts:

1. the committed wrapper can bootstrap BuildOpt, preserve Gradle arguments and
   process behavior, observe builds with bounded overhead, and fall back to the
   repository's Gradle Wrapper;
2. the existing central service can share Gradle cache objects and typed
   BuildOpt state while keeping their authority and metadata separate;
3. generic automatic runtime profiles have not demonstrated sufficient public
   breadth or positive longitudinal value; and
4. three reviewed native corrections have independently demonstrated material
   value in Micronaut, Spring, and Elasticsearch under exact-output gates.

This POC does not rerun the failed automatic-profile hypothesis. It asks a
different question: can the wrapper and central state turn reviewed native
corrections into a coherent, repeatable, multi-runner product experience with
acceptable customer economics?

## Primary hypothesis

For ordinary Gradle use in a prospectively frozen public cohort, BuildOpt can:

- collect sufficient source-independent runtime facts through `buildoptw`
  without materially slowing native builds;
- combine compatible observations from independent runners without accepting
  duplicates, stale bindings, untrusted forks, or conflicting writes;
- identify a material native correction in at least three repository families;
- validate at least two corrections with exact required outputs, zero
  additional product failures, repeatable wall-time improvement, and finite
  customer payback;
- deliver each qualifying correction as an understandable, digest-bound,
  exactly reversible proposal; and
- retain native Gradle for every incomplete, unsafe, non-material, regressive,
  stale, unavailable, or rejected case.

Passing the hypothesis supports an owner-operated POC. It does not establish
production readiness, unattended source mutation, automatic merge, hosted
multi-tenancy, or a universal Gradle optimization claim.

## Product promise under test

The normal command is:

```bash
./buildoptw <gradle arguments...>
```

The customer-visible behavior is:

1. run the exact requested workflow through the repository's native Gradle
   Wrapper;
2. never make successful Gradle execution depend on BuildOpt service health;
3. record a small, private, typed observation after the build;
4. explain whether BuildOpt is observing, has found an opportunity, is
   validating a proposal, or has a proposal ready for review;
5. perform expensive validation only on an authorized validator and within a
   fixed budget; and
6. leave source application and merge to the repository owner.

The wrapper may control invocation, observation, and coordination. It does not
own the repository's source, acceptance decision, or merge authority.

## Success and stop decisions

The terminal result must be exactly one of:

- `QUALIFY_WRAPPER_COORDINATED_NATIVE_CORRECTIONS_POC`;
- `STOP_INSUFFICIENT_PROSPECTIVE_OPPORTUNITY_BREADTH`;
- `STOP_INSUFFICIENT_VALIDATED_VALUE_BREADTH`;
- `STOP_WRAPPER_OR_BACKEND_COST_EXCEEDS_VALUE`;
- `STOP_REVIEW_OR_DELIVERY_NOT_USABLE`;
- `STOP_PRODUCT_CORRECTNESS_FAILURE`;
- `INCOMPLETE_EXPERIMENT_INPUT`; or
- `INCOMPLETE_PERFORMANCE_ENVIRONMENT`; or
- `INCOMPLETE_EXPERIMENT_BUDGET_EXHAUSTED`.

Thresholds are frozen before prospective capture. A near miss does not move a
threshold. Missing evidence is `INCOMPLETE`, never zero and never a negative
product result. A product-attributable correctness failure is retained and
forces `STOP_PRODUCT_CORRECTNESS_FAILURE`.

## Existing evidence and contamination boundary

### Diagnostic system lane

The existing Micronaut, Spring, and Elasticsearch qualified PatchBundle
recipes may be replayed only to prove the new wrapper/backend state machine,
authorization, lease, validation publication, review presentation, rejection,
and exact-revert plumbing. Their old source rows and timings remain historical
diagnostics. They cannot count toward prospective opportunity breadth,
qualified-family breadth, customer economics, or the terminal value decision.

The diagnostic lane must use copies or isolated worktrees and must never open
or update upstream proposals. A replay result is labeled
`HISTORICAL_RECIPE_SYSTEM_FIXTURE`.

### Prospective value lane

The viability decision uses only observations and results created after:

1. this plan and its machine contract are committed;
2. the BuildOpt implementation/package digest is frozen;
3. the public subject manifest is frozen; and
4. all experiment-local BuildOpt, Gradle-cache, dependency-cache, checkout, and
   backend namespaces are created empty.

No historical DNO, NAC, RNPP, EGNP, SWL, AF, Build Impact, remote-cache, or
other BuildOpt report may be copied, transformed, averaged, used as a prior,
or consumed as a detector input. Public repository Git history and Gradle's
own diagnostics are source material, not BuildOpt evidence, but every
observation derived from them must be freshly captured by the frozen package.

## Non-goals

This experiment does not authorize or attempt:

- replacing Gradle or its Wrapper;
- naming a generated script `gradlew` or silently intercepting direct
  `./gradlew` invocations;
- automatic application, commit, push, pull request, merge, or upstream
  communication;
- arbitrary AI-generated patches without a versioned deterministic recipe;
- repository-name, organization-name, task-name, or known-source special
  cases in product classification;
- Test Optimization, test selection, skipped required outputs, semantic-output
  relaxation, or weakened correctness checks;
- a new cache implementation, a third storage plane, or encoding BuildOpt state
  as Gradle cache objects;
- production HA, billing, hosted SaaS, general RBAC administration, KMS/HSM,
  autoscaling, soak, design partners, or production SLOs;
- claiming that a cache hit, avoided task, proposal count, or accepted review
  is a wall-time improvement; or
- using hosted-CI wall time as a performance gate.

Normal hosted CI may execute functional and correctness portions of the
experiment. Its duration, queue delay, CPU steal, throttling, shared disk,
network variability, cache eviction, or runner replacement may not be compared
against controlled-runner thresholds or converted into a product-value stop.

## Invariants

1. **Native execution:** every ordinary request ultimately invokes the
   repository's native Gradle Wrapper with the original ordered arguments.
2. **Build authority:** after the child starts, its exit code or signal remains
   authoritative. Observation or upload failure cannot replace it.
3. **Fail open for builds, fail closed for actions:** backend failure runs
   native Gradle; incomplete or unverified state cannot create or advance a
   proposal.
4. **Two logical planes:** Gradle cache data and BuildOpt control state may
   share physical CAS bytes but never metadata, lookup keys, authorization,
   retention, or decision authority.
5. **Immutable evidence:** observations, opportunities, proposals, validation
   results, review decisions, and artifacts are immutable and content-bound.
   Only typed heads, leases, and derived status projections are mutable.
6. **Exact bindings:** every action binds repository scope, revision, source
   preimage, Wrapper, Gradle, JDK, BuildOpt package, detector, workflow,
   environment class, output contract, and evidence digests.
7. **Owner authority:** only an owner decision may mark a proposal
   `OWNER_ACCEPTED`; acceptance does not imply `SOURCE_APPLIED` or merged.
8. **Exact reversion:** every deliverable patch must be idempotent to apply and
   revert and must reject source drift before mutation.
9. **No silent evidence loss:** valid negative, failed, cancelled, slow, or
   regressive observations remain in the ledger.
10. **Fresh value:** diagnostic historical recipes prove machinery only. Only
    the prospective lane decides this experiment.
11. **No name rules:** repository and task names are labels for reports, never
    classification inputs.
12. **No hidden cost:** wrapper, upload, diagnostics, validation, delivery,
    review, fallback, and failed-candidate costs are included in the relevant
    economics ledger.

## Architecture

```text
Developer workstation / CI runner
        |
        | ./buildoptw <Gradle args>
        v
+---------------- BuildOpt invocation boundary ----------------+
| verified bootstrap                                            |
| native fast-path decision                                     |
| repository Gradle Wrapper child                               |
| bounded observation recorder                                  |
| local durable outbox                                          |
+---------------------------------------------------------------+
        |                                  |
        | Gradle cache protocol            | typed HTTPS control API
        v                                  v
+---------------- owner-operated buildopt-server ----------------+
| Gradle data plane                 BuildOpt control plane        |
| /cache/<key>                      /api/v1/repositories/...      |
| cache.sqlite                      control/state metadata        |
|                                                              |
| opaque cache objects              immutable typed documents    |
| native cache retention            generation CAS + references  |
+---------------------------------------------------------------+
        |                                  |
        +----------- shared physical CAS --+
                                           |
                         opportunity aggregation / validation work
                                           |
                               authorized validator runner
                                           |
                              isolated native/candidate roots
                                           |
                                    owner review artifact
```

The product has no always-on BuildOpt component inside task execution. The
ordinary wrapper path observes and delegates. Aggregation and candidate
validation occur after or outside the customer's requested build.

## Storage model

### Gradle data plane

Gradle continues to own opaque cache objects addressed by Gradle cache keys.
Eviction is an ordinary cache miss. Cache presence never proves that a source
correction is safe, valuable, current, or accepted.

### BuildOpt control plane

Extend the current typed state model with these versioned kinds:

| Kind | Purpose | Mutation model | Suggested retention |
|---|---|---|---|
| `OBSERVATION` | One completed wrapper invocation and its bounded facts | Immutable append, idempotent identity | Raw 30 days; compacted facts 90 days for POC |
| `OPPORTUNITY` | Detector output derived from named observations | Immutable versions plus current head | Current plus 30 days after supersession |
| `PROPOSAL` | Exact patch recipe, preimage, rationale, and estimated economics | Immutable versions plus current head | While active, then 90 days |
| `VALIDATION` | Correctness, paired value, environment, outputs, and cost rows | Immutable and referenced | While proposal retained, then 90 days |
| `DECISION` | Owner accept/reject/defer decision over an exact proposal digest | Immutable append | Durable for experiment |
| `CHECKPOINT` | Resumable capture or validation work | Immutable generations | 24 hours |
| `PROJECTION` | Reconstructable current status for fast reads | Generation CAS | Rebuildable |

These are logical kinds, not Gradle cache entries. Identical payloads may
deduplicate in the physical CAS only after repository/kind authorization is
proved independently.

### Immutable publication

Every control-plane publication follows the existing order:

1. validate schema and exact repository/kind binding;
2. stream artifacts into the content-addressed blob store and verify SHA-256;
3. publish a canonical immutable manifest referencing complete artifacts;
4. verify referenced evidence and parent generations;
5. advance the exact kind head using compare-and-swap; and
6. record the idempotency result in the same metadata transaction.

Partial publication is invisible. An exact retry returns the original result.
Changed content under a reused idempotency key is a conflict. Competing writers
from the same generation yield one winner and one stale-precondition response.

## Data contracts

Machine schemas are required before implementation. The following fields are
minimum contracts; schema work may add fields only when required by a stated
invariant or checker.

### Observation envelope

Each `OBSERVATION` records:

- schema version, observation ID, repository scope, trust domain, and runner
  pseudonymous ID;
- idempotency key and monotonically scoped invocation ordinal;
- repository revision/tree, dirty-state classification, and source-snapshot
  digest without uploading unrelated source;
- Wrapper files/distribution, Gradle version, JDK identity, BuildOpt package,
  operating system, architecture, and declared resource class;
- original ordered Gradle arguments with a privacy-safe representation;
- workflow identity, requested tasks, finalized reachable graph digest, and
  task implementation identities where available;
- monotonic total duration and bounded per-task/critical-path facts;
- Configuration Cache requested/store/reuse/problem outcome;
- Build Cache mode and hit/miss facts, never cache payloads in the control
  document;
- required-output manifest digest when the protocol calls for outputs;
- child exit/signal, BuildOpt error class, upload status, and completeness;
- observation start/end timestamps for ordering, not performance measurement;
  and
- canonical document digest and producer signature/authentication context.

Unknown values are typed `UNAVAILABLE` with a reason. They are never empty
strings or numeric zero substitutes.

### Opportunity

Each `OPPORTUNITY` records:

- detector ID/version and frozen implementation digest;
- exact input observation IDs and digests;
- source facts and declaration binding/span when source is relevant;
- affected requested workflows and material critical-path estimate;
- typed correction class and typed rejection/no-action alternatives;
- required additional diagnostics;
- confidence/completeness classification;
- estimated validation cost and preliminary compatible-build payback; and
- one decision: `ACTIONABLE`, `NO_ACTION`, `INCOMPLETE`, `UNSAFE`,
  `NON_MATERIAL`, or `STALE`.

### Proposal

Each `PROPOSAL` records:

- proposal ID/version, opportunity digest, and exact detector recipe ID;
- source path, declaration span, source preimage SHA-256, and repository base;
- canonical patch transaction and exact inverse transaction;
- human rationale, expected behavior change, risk, and verification protocol;
- expected critical-path and wall-time benefit without claiming qualification;
- required output contract and owner test commands;
- source/license/privacy classification;
- `automaticApplyAuthorized=false`, `automaticMergeAuthorized=false`, and
  `productionAuthorized=false`; and
- status derived from immutable events, never a mutable self-assertion.

### Validation result

Each `VALIDATION` records:

- proposal and preimage digests;
- control/candidate checkout, Wrapper, Gradle, JDK, package, workflow, resource,
  and cache-namespace bindings;
- stabilization, correctness, invalidation, cross-root, and paired-value rows;
- monotonic outside-launcher durations in original order;
- complete required-output manifests and comparisons;
- candidate execution/restoration facts and source exact-revert proof;
- product, repository, environment, and infrastructure failure classes;
- mean, median, p95, paired count, positive count, conservative paired
  interval, absolute/relative saving, and all calculation inputs;
- machine time, wrapper overhead, storage/network bytes, failed work, and
  projected customer payback; and
- one typed gate decision with every failed prerequisite named.

### Owner decision

Each `DECISION` binds:

- exact proposal and validation digests;
- owner principal or experiment-owner identity;
- `ACCEPT`, `REJECT`, or `DEFER`;
- measured active review duration and optional structured concerns;
- decision timestamp and signature/authentication context; and
- an explicit statement that acceptance does not authorize automatic source
  mutation, commit, push, pull request, or merge.

## Privacy and trust

Observation is allowlist-based. The wrapper must not upload raw environment
variables, credentials, complete logs, arbitrary system properties, arbitrary
command-line values, local absolute paths, usernames, home directories, source
archives, or cache payloads.

- Known Gradle task names and safe structural flags may be stored literally.
- Unknown argument values are locally redacted or hashed with a repository-
  scoped representation; secret-looking tokens are never uploaded.
- Paths are repository-relative where required, otherwise classified and
  omitted.
- Detailed logs and source snippets remain local unless a proposal protocol
  explicitly requires a bounded artifact and owner credentials authorize it.
- Fork and pull-request runners are read-only by default and cannot publish
  authoritative observations, validations, or decisions.
- Tokens remain hash-only server-side, never enter Gradle, and never appear in
  checked evidence or logs.

The POC may reuse `CACHE_READ`, `CACHE_WRITE`, `STATE_READ`, and `STATE_WRITE`
transport capabilities, but the product-state contract must refine write
authority by actor:

| Actor | Minimum authority |
|---|---|
| Developer or untrusted CI | cache read; local observation only by default |
| Trusted observation runner | observation write for one repository scope |
| Validator runner | observation read, proposal claim, validation write |
| Repository owner | proposal read and owner-decision write |
| Administrator | token/revocation management; no implicit owner decision |

An implementation that gives every state writer implicit acceptance authority
fails the experiment.

## Local outbox and availability

The wrapper writes observations to a private local durable outbox using atomic
replacement. Upload happens only after the Gradle child completes and under a
strict post-child deadline. An unavailable backend leaves the item queued for
a later invocation and preserves the Gradle result.

The outbox is bounded by item count, bytes, and age. Eviction is oldest-first
and emits a local diagnostic; it never fabricates a successful upload.
Duplicate retry uses the original idempotency key. A verified local state
snapshot may support status display during outage but cannot advance a
proposal or validation state.

## Multi-runner coordination

### Observation aggregation

Observations may be combined only when their compatibility identity matches at
least repository scope, source/build-logic binding, Wrapper, Gradle, JDK major,
BuildOpt package/detector version, workflow, relevant options, output contract,
and declared resource class. Incompatible observations remain separate facts.

Deduplication is by canonical observation identity, not wall-clock proximity.
The aggregator never averages across workflows, revisions with relevant build
logic drift, warm/cold states, cache policies, or resource classes.

### Validation work leases

An admitted proposal produces at most one active validation lease:

- lease identity includes repository, proposal digest, protocol version, and
  environment class;
- claim uses compare-and-swap and records holder, attempt, start, and expiry;
- heartbeats extend only the same lease and never the experiment budget;
- loss or expiry makes the attempt visible and requeueable;
- publication requires the still-current lease and exact proposal digest;
- retries are capped at two infrastructure retries; correctness or value
  failures are not retried as infrastructure; and
- lease failure cannot delay or fail ordinary wrapper builds.

The checker must deterministically exercise simultaneous claims, stale holder,
late result, duplicate result, expiry/reclaim, and conflicting result cases.

## Proposal lifecycle

```text
OBSERVING
   |
   v
OPPORTUNITY_DETECTED
   | completeness, materiality, source and budget gates
   v
VALIDATION_QUEUED
   |
   v
VALIDATING --------------------> VALIDATION_INCOMPLETE
   |
   +---- correctness/value fail -> REJECTED_BY_EVIDENCE
   |
   v
REVIEW_READY
   |                |
   |                +-----------> OWNER_REJECTED / OWNER_DEFERRED
   v
OWNER_ACCEPTED
   |
   | later source observation only
   v
SOURCE_APPLIED
   |
   +---- source/build-logic drift -> STALE
```

`OWNER_ACCEPTED` is not `SOURCE_APPLIED`. `SOURCE_APPLIED` requires a later
fresh observation whose exact source postimage matches the accepted recipe;
it does not assert commit, push, PR, merge, or upstream acceptance. Every state
is reconstructable from immutable events and manifests.

## Detector scope for the POC

### Primary new detector

The prospective experiment starts with
`CONFIGURATION_CACHE_READINESS_PATCH_V1`. It may propose only a small,
repository-owned, versioned native Gradle correction for a reproducible
Configuration Cache blocker when:

- the requested workflow is material and repeatable;
- Gradle emits a stable machine-bindable problem attributable to owner source;
- the relevant source declaration and runtime binding are unambiguous;
- the recipe changes no dependency and has an exact inverse;
- source and behavior gates justify the intended semantics; and
- projected validation/customer payback is inside the frozen limit.

It must not silence a problem, add broad compatibility opt-outs, disable
Configuration Cache, ignore failures, or infer semantics from repository/task
names. Enabling Configuration Cache is not itself the success metric; only
end-to-end wall time with exact outputs is value.

### Existing reviewed-native detector catalog

Existing normalization-aware cacheability and durable task-contract recipes
may participate only when their versioned current implementations independently
classify fresh prospective evidence. Historical candidate lists do not seed the
cohort. Explicit build-logic opt-out and graph-breadth routes remain closed
unless a separate new hypothesis and contract reopens them.

### Detector result contract

Every repository/detector row must be one of:

- `ACTIONABLE_MATERIAL_CORRECTION`;
- `NO_REPRODUCIBLE_BLOCKER`;
- `NON_MATERIAL_BLOCKER`;
- `OWNER_SEMANTICS_REQUIRED`;
- `UNSAFE_OR_NON_REVERSIBLE`;
- `UNSUPPORTED_PROBLEM_CLASS`;
- `INCOMPLETE_OBSERVATION`;
- `SOURCE_OR_BINDING_AMBIGUOUS`; or
- `SOURCE_DRIFTED`.

Only the first row opens validation. `OWNER_SEMANTICS_REQUIRED` may produce a
question/report but cannot compile or time a candidate.

## Prospective cohort

Before source inspection, freeze ten substantial public Gradle repository
families. At least five must not have supplied a qualified reviewed-native
recipe in any prior BuildOpt experiment. Selection uses only public, neutral
facts fixed in the subject contract:

- active build-owning source in Kotlin or Groovy DSL or JVM task plugins;
- a checked Gradle Wrapper compatible with the controlled JDK/toolchain;
- a reproducible owner workflow with declared required outputs;
- successful optimized-native preflight within the per-build limit; and
- repository size suitable for the disk/time budget.

Freeze one primary revision and two chronological reserves per family before
running the frozen detector. A reserve replaces a primary only for a typed
repository/toolchain/environment failure defined in advance; never because the
primary has no opportunity or poor performance.

Repository and workflow labels may appear in the subject manifest and report,
but classifiers receive only source/runtime facts. The independent checker
must prove classification invariance under renamed repository, task, project,
and checkout-root fixtures.

## Controlled execution environment

Before capture, record:

- exact host, operating system/kernel, CPU allocation, memory allocation, disk
  class/free bytes, power mode, JDK, Gradle distributions, BuildOpt package,
  and clock source;
- whether the runner is dedicated and what background-load exclusion applies;
- dependency-network policy and all prefetch/setup steps;
- separate checkout, Gradle User Home, local cache, remote cache namespace, and
  BuildOpt state roots per family/arm; and
- monotonic outside-launcher measurement command and raw-row destination.

Performance gates run only on the declared controlled runner. Hosted CI checks
contracts and correctness but never accepts or rejects wall-time value.

### Execution classes and applicable gates

Every run declares exactly one environment class before execution:

| Environment class | Intended use | Hard acceptance authority |
|---|---|---|
| `CONTROLLED_PERFORMANCE` | Materiality, wrapper overhead, paired value, tails, and payback | All deterministic and performance gates |
| `STANDARD_HOSTED_CI` | Compilation, schemas, unit/integration tests, exact outputs, lifecycle, security, portability, and evidence reconstruction | Deterministic gates only; timings are diagnostic |
| `LOCAL_FUNCTIONAL` | Focused developer reproduction and debugging | Deterministic focused proof only; no terminal breadth or value authority |

A run cannot change class after its result is known. Environment metadata must
show the declared class in every raw row and summary.

### Normal hosted-CI acceptance policy

Normal CI is deliberately tolerant of performance noise without becoming
tolerant of incorrect behavior:

- compilation, schema validity, canonical hashes, namespace isolation,
  authorization, idempotency, state transitions, exact output bytes, expected
  invalidation, source drift rejection, and exact revert remain hard gates;
- elapsed time, pair sign, p95, confidence interval, cache hit rate, upload
  latency, and payback are recorded as `DIAGNOSTIC_ONLY` and are excluded from
  acceptance arithmetic;
- queue delay, runner provisioning, CPU steal, throttling, shared-disk
  contention, dependency-network outage, provider cache eviction, spot-runner
  loss, job cancellation, and hosted timeout are typed
  `CI_ENVIRONMENT_FAILURE` when supported by direct evidence;
- a CI-environment failure is retained in the ledger but is neither a favorable
  sample, an unfavorable sample, nor a product failure;
- one replacement attempt is allowed only after the original row is preserved
  and the environmental classification is established. It does not erase the
  first attempt or extend the controlled experiment budget;
- a deterministic output mismatch, wrong state transition, authorization
  bypass, source mutation, secret disclosure, or reproducible BuildOpt crash is
  still a product failure on any runner; and
- a generic CI timeout is not called a BuildOpt regression unless a focused
  reproduction attributes it to BuildOpt. Until then it is incomplete
  infrastructure evidence.

CI may use generous job timeouts and reduced fixture sizes to prove termination
and control flow. Those accommodations are fixture parameters, not relaxed
product-value thresholds, and the checked contract must prevent their results
from entering the prospective performance ledger.

### Absence or instability of the controlled runner

Functional blocks may continue on normal CI while controlled performance
capacity is unavailable. Candidate timing and any terminal value claim remain
closed. If the controlled runner cannot be established, loses its declared
resource envelope, or fails a preregistered noise/stability preflight, the
experiment reports `INCOMPLETE_PERFORMANCE_ENVIRONMENT` with completed
correctness evidence preserved.

It must not substitute CI timings, loosen 8/8 repeatability, widen the paired
interval, waive p95, or increase the payback limit. The correct response to an
unstable measurement environment is an incomplete value result, not a failed
optimization and not a weaker success criterion.

## Fixed budgets

`WCNCP-000` must encode concrete numbers in the machine contract before any
prospective capture. Unless that block changes them with an explicit owner
review before capture, use these defaults:

| Resource | Hard limit |
|---|---:|
| Public families | 10 primary plus 20 frozen reserve revisions |
| Ordinary observation builds | 3 per primary family, 30 total |
| Additional native diagnostics | 2 per family, 20 total |
| Compiled proposals | 1 per family, 10 total |
| Concurrent validations | 1 on the controlled machine |
| Infrastructure retries | 2 per proposal |
| Correctness starts | 5 per proposal before timing |
| Paired value | 8 alternating pairs per correctness-qualified proposal |
| Wall-clock experiment budget | 12 controlled-machine hours, excluding dependency prefetch |
| Concurrent on-disk experiment footprint | 30 GiB |
| Minimum free disk before a new candidate | 10 GiB above its declared maximum footprint |
| Wrapper post-child upload deadline | 100 ms per ordinary invocation |
| Local observation outbox | 100 MiB or 1,000 items or 30 days, whichever comes first |

Crossing a hard limit stops new work and produces
`INCOMPLETE_EXPERIMENT_BUDGET_EXHAUSTED`; it does not authorize deleting
unrelated files, killing unrelated processes, reducing pair counts, discarding
rows, or moving success thresholds.

## Measurement and economics

### Wrapper overhead

Measure direct `./gradlew help` and `./buildoptw help` with observation off,
local enqueue, successful upload, queued-offline upload, and status lookup.
Use at least 20 alternating observations per mode after unmeasured setup.

The numerical overhead gate applies only to `CONTROLLED_PERFORMANCE`. Normal CI
runs the same modes as functional smoke tests, proves bounded control flow and
preserved native results, and labels all durations `DIAGNOSTIC_ONLY`.

The ordinary local-enqueue path must add no more than 50 ms p95 before/after
the child and no more than 0.5% mean wall time on workflows lasting at least 10
seconds. Upload completion is bounded separately and may be deferred.

### Materiality admission

A candidate enters correctness only if the reproducible blocker or affected
task contributes at least both:

- 500 ms to the requested workflow's measured critical path; and
- 2% of optimized-native workflow wall time.

The detector may report smaller opportunities, but they are
`NON_MATERIAL_BLOCKER` and receive no candidate build.

Materiality is computed only from controlled observations. A hosted-CI trace
may locate a suspected task or Configuration Cache blocker, but it can only
produce `MATERIALITY_REQUIRES_CONTROLLED_MEASUREMENT`, never admit or reject a
candidate on timing.

### Correctness protocol

For each admitted proposal, run at least:

1. control execution from the restored baseline;
2. candidate execution from an equivalent fresh state;
3. same-root reuse or Configuration Cache reuse as applicable;
4. relevant input/source change and expected invalidation;
5. restored baseline input and exact behavior restoration; and
6. cross-root execution/reuse when the correction claims portable caching.

Several requirements may share a start only when the raw trace proves both.
The total remains at least five starts. Compare every declared required output
byte-for-byte. Also run the repository's focused owner tests and validate exact
source apply/reapply/revert/rerevert behavior. Any mismatch, unexpected reuse,
missing invalidation, source drift, or product failure rejects the candidate
before timing.

### Paired value protocol

For every correctness-qualified proposal:

- create equivalent isolated control and candidate roots;
- give both arms equal optimized-native local/remote cache opportunity;
- exclude only preregistered stabilization starts and retain their costs;
- run eight balanced alternating pairs, reversing first-arm order by pair;
- measure outside both launchers with a monotonic clock;
- preserve every valid row and classify invalid rows before replacement;
- compare exact required outputs for every pair; and
- record task/Configuration Cache state without using it as the value metric.

This protocol runs only as `CONTROLLED_PERFORMANCE`. A normal-CI
control/candidate matrix may verify orchestration, outputs, failure retention,
and report reconstruction with small fixtures, but every timing-derived field
and decision remains `NOT_EVALUATED_STANDARD_CI`.

A correction qualifies only with:

- 8/8 favorable paired wall-time deltas;
- positive mean absolute and relative saving;
- a strictly positive conservative paired 95% interval;
- no candidate p95 regression;
- at least 500 ms and 2% mean saving;
- zero additional product failures; and
- projected customer payback at or below 300 compatible builds.

### Cost ledgers

Report two ledgers without mixing them:

1. **POC research cost:** implementation, fixture, experiment setup, debugging,
   invalid protocol, and documentation work. It is disclosed but is not a
   recurring customer cost.
2. **Customer operational cost:** wrapper overhead on all observed builds,
   observation upload, diagnostics, validation builds, failed/rejected
   candidates, proposal delivery, active owner review, and native fallbacks.

Per-proposal payback is:

```text
ceil(all attributable customer operational cost / qualified mean saving)
```

The portfolio report also includes cumulative signed value across every
prospective family; rejected candidates contribute their full cost and zero
future saving. A projection is labeled as such and never reported as realized
saving.

## Review usability protocol

Only a value-qualified proposal reaches review. Use a first-exposure artifact
containing:

- one-sentence problem and expected improvement;
- exact source diff and declaration location;
- why the transformation is semantically safe;
- correctness and invalidation evidence;
- control/candidate wall-time table, interval, p95, and payback;
- known limitations and affected workflows;
- exact apply and revert commands/transactions; and
- explicit statement that no automatic mutation or merge occurred.

Capture active review time, decision, clarification count, and structured
concerns. Do not count waiting/idle time. The usability gate requires every
qualifying proposal to be correctly understood, no critical unanswered safety
question, and median active review at or below five minutes. Owner rejection
is retained and does not become a technical failure unless caused by an
incorrect or misleading artifact.

## Ordered execution blocks

| Order | Block | Deliverable | Gate to continue |
|---:|---|---|---|
| 0 | `WCNCP-000` Contract freeze | Human plan, machine contract, subject-selection rules, schemas, budget, threat model, and independent plan checker | `DONE`: five schemas compile, ten slots remain unselected, and experiment evidence is zero |
| 1 | `WCNCP-001` Typed control state | New state kinds, canonical schemas, CAS persistence, retention, migrations, corruption and concurrency negatives | `TODO`: restart-safe immutable publication and namespace isolation must pass |
| 2 | `WCNCP-002` Remote API and authority | HTTPS routes, scoped actor capabilities, idempotency, local verified snapshots, and audit events | Wrong scope/fork/stale/partial/conflicting writes fail closed; builds remain independent |
| 3 | `WCNCP-003` Wrapper observation path | Privacy-safe recorder, local outbox, bounded uploader, status/explain projection, exact native passthrough | Cross-platform contracts pass; controlled overhead passes when that environment is available, otherwise it remains explicitly pending |
| 4 | `WCNCP-004` Aggregation and detector | Compatibility grouping, Configuration Cache detector v1, current reviewed-native catalog adapters, materiality gate, and name-invariance negatives | Synthetic Kotlin/Groovy positives and all typed negatives reconstruct independently |
| 5 | `WCNCP-005` Validation coordinator | Proposal admission, leases, isolated roots, budget accounting, correctness/value runner, immutable result publication | Concurrent claims, expiry, retry, cancellation, failure, exact outputs, and exact revert pass |
| 6 | `WCNCP-006` Review delivery | Digest-bound PatchBundle artifact, human/JSON report, accept/reject/defer events, status lifecycle | Historical recipes traverse the system lane; no public mutation or historical value reuse |
| 7 | `WCNCP-007` Multi-runner system proof | One server, at least three isolated runners, duplicate/offline/conflict/outage scenarios, and one validator | Shared observations converge once, one validation owns the lease, outage preserves native results |
| 8 | `WCNCP-008` Prospective cohort capture | Frozen ten-family manifest and 30 fresh ordinary wrapper observations from empty state | 10/10 families have conclusive observation completeness or experiment is `INCOMPLETE` |
| 9 | `WCNCP-009` Opportunity breadth gate | Independent reconstruction of observations, detector rows, critical-path materiality, and family counts | At least 3/10 families have actionable material corrections; otherwise stop before candidates |
| 10 | `WCNCP-010` Candidate correctness | At most one candidate per admitted family; source, behavior, invalidation, outputs, owner tests, and exact revert | At least two families pass correctness; any product correctness failure forces terminal stop |
| 11 | `WCNCP-011` Paired value and economics | Eight-pair controlled results, wrapper/backend costs, rejected-candidate costs, operational payback, and cumulative ledger | At least two families qualify value and portfolio signed value is positive; standard CI cannot decide this gate |
| 12 | `WCNCP-012` First-exposure owner review | Review artifacts and measured owner decisions only for value-qualified proposals | All artifacts understood; median active review <=5 minutes; no critical unresolved safety issue |
| 13 | `WCNCP-013` Terminal decision | Independent recomputation, documentation closure, and exact statement of proven boundary | One fixed terminal result with no moved threshold or missing row |

No block may borrow authority from a later block. In particular, source
classification cannot compile a candidate; opportunity breadth cannot start
timing; value qualification cannot apply source; owner acceptance cannot commit
or merge a public change.

## Block details and proof

### WCNCP-000 — Contract freeze

Create:

- `specs/poc-wrapper-coordinated-native-corrections-v1.md`;
- `specs/poc-wrapper-coordinated-native-corrections-v1.json`;
- `specs/poc-wrapper-coordinated-native-corrections-v1.subjects.json`;
- JSON Schemas and language-neutral vectors for every new document;
- `dev/check-wrapper-coordinated-native-corrections-plan`; and
- an evidence directory README with every expected artifact and state.

The machine contract contains all thresholds, budgets, decision enums,
freshness cutoff, forbidden evidence patterns, required output fields, and
ordered dependencies. The checker compares this tracker, the machine contract,
subject manifest, evidence index, implementation tracker, and findings docs.

Stop on dirty/unexpected baseline, ambiguous toolchain, missing source license,
or inability to freeze an empty evidence/state boundary.

### WCNCP-001 — Typed control state

Extend the narrow existing central-state seam instead of building a parallel
database. Add migrations with exact checksums and rollback/forward-open tests.
Prove repository/kind isolation even when bytes are identical. Verify restart,
partial object publication, missing/corrupt artifacts, stale/skipped
generation, concurrent CAS, exact idempotent replay, changed-request conflict,
retention, referenced-object preservation, and projection reconstruction.

No server route or wrapper integration is required to pass this block.

### WCNCP-002 — Remote API and authority

Expose the typed store through existing TLS/authentication infrastructure.
Define separate route families for immutable objects/manifests, heads, outbox
batch publication, proposal claims, lease heartbeat/release, validation
publication, and owner decisions. Prefer bulk observation upload with a fixed
item/byte maximum to reduce wrapper overhead.

Prove live token revocation, wrong repository, wrong actor, fork context,
expired token, payload digest mismatch, replay, stale head, partial batch, and
server restart. Logs contain request IDs and digests but no credentials,
source content, or raw unsafe arguments.

### WCNCP-003 — Wrapper observation path

Integrate after the existing native fast path without changing Gradle task
selection. The recorder starts only when configured and must avoid pre-child
network. It collects bounded finalized-build facts, writes one private atomic
outbox item after the child exits, and attempts a bounded batch upload. Status
and explain derive from verified local/remote projection and distinguish
`UNAVAILABLE`, `QUEUED`, `OBSERVING`, and proposal states.

Fixtures cover POSIX and Windows-shaped paths/arguments, signals, non-zero
Gradle exit, cancellation, no Wrapper, bypass, backend outage, slow backend,
corrupt response, full outbox, duplicate retry, redaction, and direct native
equivalence. Real-binary tests verify stdout/stderr/stdin/cwd/argv/exit/signal
behavior.

All deterministic behavior is required on normal CI. The fixed numerical
overhead threshold is evaluated only on the controlled runner. Lack of that
runner leaves the performance portion pending and does not block later
functional backend/detector implementation, but `WCNCP-011` and a qualifying
terminal decision remain closed.

### WCNCP-004 — Aggregation and detector

Implement compatibility grouping as a pure deterministic function over typed
facts. Add independent reconstruction from raw observations. The detector must
identify Configuration Cache problem type, owning source binding, requested
workflow reachability, repetition, critical-path materiality, safe recipe
class, and missing facts without task/repository-name branches.

Positive fixtures cover Kotlin and Groovy build logic where practical.
Negatives cover renamed labels, missing/incomplete Gradle problem data,
ambiguous source ownership, external plugin ownership, generated/vendor source,
absolute-path dependence, non-reversible edits, semantics requiring owner
input, non-material blockers, source drift, disabled Configuration Cache, and
suppression-style patches.

### WCNCP-005 — Validation coordinator

The coordinator consumes only `ACTIONABLE_MATERIAL_CORRECTION` with complete
bindings and remaining budget. It materializes control/candidate in explicit
isolated roots, never the owner's active checkout. Apply and revert reject
preimage/postimage drift. Every process, worktree, Gradle User Home, cache
namespace, output manifest, raw timing row, and cleanup target is recorded.

Cleanup is limited to experiment-owned, exact resolved paths after process and
artifact checks. It must never use destructive broad paths or touch unrelated
Gradle caches. Interrupted work resumes only from digest-compatible
checkpoints. The coordinator stops scheduling before exceeding time or disk.

### WCNCP-006 — Review delivery

Reuse the signed reviewed-native PatchBundle delivery seam. Add proposal and
validation references rather than duplicating patch semantics. Prove draft
creation, readable human report, machine report, owner accept/reject/defer,
replay, stale proposal rejection, tamper rejection, exact revert, and no-action
status. Acceptance changes only control-plane state.

Replay known recipes in this block solely as system fixtures and label every
row diagnostic-only.

### WCNCP-007 — Multi-runner system proof

Use one restarted HTTPS service and at least three isolated runner roots:

- runner A publishes a trusted observation;
- runner B publishes a compatible distinct observation;
- runner C retries one duplicate and holds one offline outbox item;
- two validators race for one proposal; exactly one acquires the lease;
- server loss during an ordinary build preserves native output/result;
- restart accepts the queued exact retry once;
- an incompatible runner observation remains isolated; and
- wrong-scope, fork, stale, late, and tampered publications fail closed.

No synthetic timing from this block counts as public value.

### WCNCP-008 — Prospective capture

Verify the freeze gate immediately before the first build. Capture exactly
three requested wrapper builds per primary family from empty BuildOpt state.
The requests must be normal owner workflows, not BuildOpt-only probes.
Additional diagnostics remain closed until `WCNCP-009` identifies missing
facts that fit the two-per-family budget.

Each family emits raw immutable observations, output bindings, outbox/upload
records, backend manifests, compatibility grouping, exclusions, and SHA-256
sums. The independent checker reconstructs every row from primary artifacts
and rejects timestamps/digests that predate the freeze or reference historical
BuildOpt evidence.

### WCNCP-009 — Opportunity breadth

Run the frozen detector catalog from fresh observations. Use at most two
additional native diagnostics per family and zero candidate builds. Require
conclusive detector rows for all ten families. Then independently reconstruct
source binding, recurrence, requested-workflow reachability, critical-path
materiality, preliminary validation cost, and family counts.

Fewer than three actionable material families stops candidate work with
`STOP_INSUFFICIENT_PROSPECTIVE_OPPORTUNITY_BREADTH`. An incomplete family yields
`INCOMPLETE_EXPERIMENT_INPUT`, not a breadth failure.

### WCNCP-010 — Candidate correctness

Compile at most one deterministic proposal per admitted family. Perform the
correctness protocol sequentially inside the resource budget. Publish all
results, including rejections. If fewer than two families pass without a
BuildOpt correctness failure, stop as insufficient validated breadth. Any
BuildOpt-caused wrong output, wrong invalidation, active-checkout mutation, or
misreported success forces `STOP_PRODUCT_CORRECTNESS_FAILURE`.

### WCNCP-011 — Paired value and economics

Run paired timing only for correctness-qualified proposals. Freeze raw rows
before aggregation. An independent checker recomputes order, durations,
outputs, positive counts, statistics, intervals, p95, costs, and payback without
trusting report summaries. Publish per-family and portfolio signed values.

Before the first pair, verify `CONTROLLED_PERFORMANCE` classification and its
noise/stability preflight. If unavailable, publish
`INCOMPLETE_PERFORMANCE_ENVIRONMENT` and preserve completed correctness work.
Never fill the missing value rows from hosted-CI diagnostics.

At least two families must pass every value gate, at least one must be a family
without a prior qualified reviewed-native recipe, and cumulative prospective
customer operational value must be positive. Otherwise stop with the precise
failed gate.

### WCNCP-012 — Owner review

Generate the first-exposure artifact only after value qualification. Present
it through the intended local/web-neutral review surface and provide a direct
link or exact path. Start active timing when the owner begins reading and stop
when they decide or ask a substantive clarification. Preserve the exact
artifact digest and response.

No source application, commit, push, PR, or upstream contact follows the
decision in this experiment.

### WCNCP-013 — Terminal decision

Recompute from immutable primary artifacts:

- wrapper correctness and overhead;
- backend availability, isolation, idempotency, and concurrency;
- 10/10 prospective completeness;
- actionable and qualified family breadth;
- exact outputs, invalidation, cross-root portability, and failures;
- paired statistics and p95;
- customer operational and POC research costs;
- per-proposal and portfolio payback;
- review comprehension/time; and
- every stopped, rejected, incomplete, and unmeasured row.

The qualifying decision requires all of:

| Criterion | Required result |
|---|---:|
| Native wrapper behavior | All platform/real-binary contracts pass |
| Wrapper overhead | <=50 ms p95 local enqueue and <=0.5% mean on >=10 s builds |
| Multi-runner convergence | 3 runners, exactly-once logical observations, one validator lease |
| Prospective completeness | 10/10 families conclusive |
| Opportunity breadth | >=3/10 families actionable and material |
| Correctness breadth | >=2 families pass all correctness gates |
| Value breadth | >=2 families qualify, including >=1 new qualified family |
| Pair repeatability | 8/8 favorable for every qualified proposal |
| Paired interval and tail | Positive conservative 95%; no p95 regression |
| Material value | >=500 ms and >=2% mean saving per qualified proposal |
| Customer payback | <=300 compatible builds per qualified proposal |
| Portfolio economics | Positive cumulative signed prospective operational value |
| Product failures | 0 |
| Review usability | All understood; median active review <=5 minutes; no critical open concern |

The table is evaluated only when all performance-bearing rows come from the
frozen `CONTROLLED_PERFORMANCE` environment. On `STANDARD_HOSTED_CI`, the
corresponding cells are `NOT_EVALUATED_STANDARD_CI`, not pass or fail. The
terminal checker rejects any report that treats them as zero, averages them
with controlled rows, or uses them to decide qualification.

Publish both human and machine terminal reports. A conclusion names the exact
boundary: detector classes, repository families, workflows, environment, and
unverified production concerns. Historical recipes remain outside the score.

## Expected repository scope during execution

The implementation should begin at existing seams and add no parallel product:

- `cmd/buildopt` and `internal/launcher` for wrapper observation/outbox/status;
- existing wrapper packages for generated POSIX/Windows scripts;
- `cmd/buildopt-server` and current central HTTP/auth packages for remote API;
- the current content-addressed store and central typed-state packages for
  persistence;
- `internal/normalizationaware`, reviewed-native PatchBundle owners, and a new
  narrowly owned Configuration Cache detector package for detector logic;
- one validator coordinator package and CLI entrypoint only if no existing
  command can own the lifecycle coherently;
- `contracts/jsonschema` and `contracts/test-vectors` for portable documents;
- `specs`, `docs/plans`, `docs/findings`, and `benchmarks/results` for contract,
  route, findings, and evidence; and
- `dev/check-*` / `dev/run-*` entrypoints consistent with existing repository
  conventions.

Before each block, inspect the exact existing owner and tests. Do not create a
new server, database, launcher, patch language, canonical JSON implementation,
or statistics library when the current seam satisfies the contract.

## Required evidence layout

Use:

```text
benchmarks/results/wrapper-coordinated-native-corrections-v1/
  README.md
  contract.json
  subjects.json
  environment.json
  package-manifest.json
  observations/
  opportunities/
  proposals/
  validations/
  reviews/
  costs/
  exclusions.json
  report.json
  terminal-decision.json
  sha256sums
```

Generated raw evidence is immutable after its block closes. Corrections create
new versioned artifacts; they do not overwrite rows. `sha256sums` covers every
primary artifact in deterministic path order. README clearly distinguishes
diagnostic system fixtures from prospective decision evidence.

## Independent checkers

No checker may validate a report only by reading its summary fields. At
minimum, independent checkers must:

- regenerate canonical IDs and digests;
- reconstruct state heads and lifecycle from immutable events;
- reconstruct observation compatibility and deduplication;
- recompute family completeness, opportunity, correctness, and value counts;
- verify every referenced source preimage/postimage and patch inverse;
- recompute output manifests and equality;
- recompute paired statistics, confidence interval, p95, and payback;
- sum rejected/failed candidate and wrapper/backend costs;
- reject forbidden historical evidence digests and pre-freeze timestamps;
- rename labels and prove classifier invariance;
- verify no Gradle cache namespace authorizes control state;
- verify owner acceptance cannot create source-applied or merge state; and
- reject omitted, duplicated, reordered, or summary-only rows.

They must also verify environment-class immutability, exclude every
`STANDARD_HOSTED_CI` duration from materiality/value arithmetic, retain typed
CI-environment failures, and reject a report that converts infrastructure noise
into a product failure or a favorable replacement sample.

## Validation ladder

Use the smallest affected proof after each edit. The final implementation
state requires, as applicable:

1. focused Go unit/race tests for touched packages;
2. schema compilation and language-neutral test vectors;
3. central storage, HTTPS/authentication, state-sync, and corruption checks;
4. wrapper generator, bootstrap, passthrough, lifecycle, native-fallback,
   outbox, privacy, and overhead checks;
5. detector Kotlin/Groovy fixtures and name-invariance negatives;
6. validation coordinator, lease, cancellation, budget, exact apply/revert, and
   result-reconstruction checks;
7. diagnostic PatchBundle system-lane replay;
8. three-runner isolated system proof;
9. prospective capture and each gated evidence checker;
10. `go vet ./...` or the repository-supported equivalent;
11. ShellCheck for every changed shell script;
12. `git diff --check`;
13. layout, documentation, tracker-consistency, and Base CI static integration
    contracts; and
14. hosted Base CI and Native Platform CI after each pushed milestone that
    changes shared runtime behavior.

Do not rerun successful expensive checks unless their inputs changed. Record
the exact command, SHA, result, duration, and whether later changes invalidated
it. Hosted CI confirms portability/correctness only, not timing gates.
Its success means that deterministic checks completed on that provider; it
does not mean that wrapper overhead, candidate speedup, tails, confidence, or
payback passed. Conversely, a classified provider-capacity failure does not
reject performance until reproduced and attributed under the controlled
protocol.

## Documentation closure

Every completed block updates the smallest relevant set. The terminal block
must align:

- this plan and the machine contract;
- the evidence directory README;
- `README.md` product direction and onboarding;
- `docs/getting-started/product-onboarding.md`;
- `docs/plans/centralized-cache-and-state-roadmap.md`;
- `docs/findings/buildopt-poc-handoff.md`;
- `docs/findings/buildopt-generalization-audit.md`;
- `docs/findings/build-optimization-performance.md`;
- architecture and validation indexes;
- `implementation-tracker.md` and its evidence ledger; and
- CLI/server/package documentation affected by the delivered behavior.

Repository text, schemas, logs, commit messages, and evidence remain English.
Documentation must say what was measured, what was inferred, what was not run,
and which prior hypothesis remains stopped.

## Commit and resumability protocol

Each block should end at an independently checkable commit before the next
block mutates shared contracts. A block commit includes implementation, tests,
contracts, evidence produced by that block, and aligned English documentation.
Do not combine unrelated cleanup. Do not amend, rebase, force-push, or rewrite
history.

Before commit, inspect status and the complete relevant diff, stage only
intended files, and rerun the smallest proof invalidated by final edits. Before
push, verify branch, upstream, outgoing commits, and fast-forward status. After
push, verify local `HEAD`, `origin/main`, and remote SHA. Where requested for
the execution phase, wait for Base CI and Native Platform CI to complete and
record their run IDs and conclusions.

After context compaction or handoff, resume from this document plus the checked
machine state. Reconstruct:

- current block and dependency gate;
- current branch/HEAD/origin and working-tree diff;
- frozen package/subject/environment digests;
- spent builds, time, retries, and disk budget;
- completed evidence/checkers and whether later edits invalidated them;
- live validation lease or watcher stable identity;
- pending owner authority; and
- exact next action and stopping point.

Never recreate a lease, watcher, validation, or public-source mutation from a
summary alone.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Developers bypass `buildoptw` | Make CI use the committed wrapper; report observation coverage honestly; retain explicit `BUILDOPT_BYPASS=1` |
| Wrapper becomes a new bottleneck | Native fast path, lazy recorder, local outbox, bounded upload, fixed overhead gate |
| Backend outage breaks builds | No pre-child network dependency; preserve child result; queue locally; native fallback |
| Cache objects accidentally authorize actions | Separate routes, metadata, capabilities, schemas, retention, and negative cross-plane tests |
| Multiple runners duplicate or corrupt evidence | Canonical IDs, idempotency keys, generation CAS, immutable manifests, independent reconstruction |
| Validators duplicate expensive work | Exact proposal leases, heartbeat/expiry, one active claim, bounded retries |
| Private data leaks in observations | Allowlist collection, relative paths, local redaction, bounded artifacts, secret-negative fixtures |
| A proposal changes semantics | Typed deterministic recipes, source binding, owner semantics gate, focused tests, invalidation and exact-output proof |
| Configuration Cache appears faster but outputs differ | End-to-end exact outputs and repository tests remain prerequisites |
| Known successful repositories bias the result | Historical recipes only in diagnostic lane; prospective cohort frozen before detector inspection |
| Campaign cost overwhelms saving | Materiality screen, one candidate/family, sequential validation, hard budget, complete operational ledger |
| Owner acceptance is mistaken for deployment | Separate immutable states for review, acceptance, observed source application, and staleness |
| Disk exhaustion causes unsafe cleanup | Per-candidate footprint declaration, minimum free-space gate, exact experiment-owned paths only |
| A near miss causes threshold movement | Machine-frozen gates and independent terminal checker |
| Normal CI timing noise creates a false win or rejection | Immutable environment classes; CI timing is diagnostic-only; controlled rows alone decide performance |

## Rollback

The POC is additive. At any point:

- `BUILDOPT_BYPASS=1 ./buildoptw ...` runs the repository Gradle Wrapper;
- disabling observation stops new outbox entries without changing Gradle;
- server outage or revoked state capability retains native execution;
- abandoned partial state remains invisible and expires under retention;
- a rejected/stale proposal cannot be claimed or applied;
- candidate roots can be discarded only after exact ownership/process checks;
  and
- an applied experiment recipe has a digest-bound exact inverse, but reverting
  public source requires separate owner authorization.

Removing the product feature later must leave the existing Gradle Wrapper and
repository build usable without BuildOpt state.

## Evidence ledger

| Evidence | Block | Required artifact | State |
|---|---|---|---|
| `WCNCP-E000` | WCNCP-000 | Frozen plan, machine contract, subjects, schemas, and plan checker | `DONE` — [contract](../../specs/poc-wrapper-coordinated-native-corrections-v1.md), [subject policy](../../specs/poc-wrapper-coordinated-native-corrections-v1.subjects.json), and [evidence index](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/README.md) |
| `WCNCP-E001` | WCNCP-001 | Typed-state persistence and lifecycle proof | `DONE` — [typed-state proof](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e001-typed-state.json) with five immutable kinds, CAS persistence, retention, namespace isolation, and corruption/concurrency negatives; zero prospective evidence |
| `WCNCP-E002` | WCNCP-002 | HTTPS authority/idempotency proof | `DONE` — [authority proof](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e002-authority.json) with actor-refined routes, bulk upload bounds, verified snapshots, and audit events; zero prospective evidence |
| `WCNCP-E003` | WCNCP-003 | Wrapper observation, privacy, fallback, and overhead proof | `DONE` — [observation proof](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e003-observation.json) with exact passthrough, redaction, bounded outbox/upload, and status surfaces; controlled overhead pending without a controlled runner; zero prospective evidence |
| `WCNCP-E004` | WCNCP-004 | Aggregator/detector fixtures and independent reconstruction | `TODO` |
| `WCNCP-E003` | WCNCP-003 | Wrapper observation, privacy, fallback, and overhead proof | `WAITING` |
| `WCNCP-E004` | WCNCP-004 | Aggregator/detector fixtures and independent reconstruction | `WAITING` |
| `WCNCP-E005` | WCNCP-005 | Lease and validator lifecycle proof | `WAITING` |
| `WCNCP-E006` | WCNCP-006 | Diagnostic-only reviewed PatchBundle system replay | `WAITING` |
| `WCNCP-E007` | WCNCP-007 | Three-runner backend convergence/outage proof | `WAITING` |
| `WCNCP-E008` | WCNCP-008 | Fresh ten-family observation capture | `WAITING` |
| `WCNCP-E009` | WCNCP-009 | Prospective opportunity breadth report | `WAITING` |
| `WCNCP-E010` | WCNCP-010 | Candidate correctness results | `WAITING` |
| `WCNCP-E011` | WCNCP-011 | Paired value and complete economics ledger | `WAITING` |
| `WCNCP-E012` | WCNCP-012 | First-exposure owner review artifacts/results | `WAITING` |
| `WCNCP-E013` | WCNCP-013 | Independently checked terminal decision | `WAITING` |

## Immediate next action

`WCNCP-001` through `WCNCP-003` are `DONE`: typed control state, scoped HTTPS
authority, and the privacy-safe wrapper observation path with exact native
passthrough are proved with zero prospective evidence. Controlled overhead
remains explicitly pending without a `CONTROLLED_PERFORMANCE` runner.

Implement only `WCNCP-004`: compatibility grouping as a pure deterministic
function, Configuration Cache detector v1, current reviewed-native catalog
adapters, materiality handling, and name-invariance negatives. Do not compile
candidates, run prospective builds, mutate public source, collect timing, or
claim value in that block.
