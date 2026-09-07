# Complete Native Correction POC

**Experiment:** `COMPLETE_NATIVE_CORRECTION_V1` (`CNC`)<br>
**State:** `STOP_NATIVE_ADMISSION`; fourth CNC-004 window completed six native captures; warmed configuration is 362 ms and two native JAR manifests change without a candidate<br>
**Planning baseline:** BuildOpt `ca5fb5d8c10ac581de7478cc1dec52269da67e24`<br>
**Execution and evidence status:** [step tracker](./complete-native-correction-poc-tracker.md)<br>
**First subject:** GraphQL Java, explicitly local non-CI `assemble`; no owner-CI qualification

## Decision and authority

Test whether a complete, bounded native correction can remove all relevant
Configuration Cache blockers in one workflow, preserve behavior, save wall
time, and retain that value through the installed wrapper and ordinary changes.
Only then test replication on previously unused families.

The user requested these Markdown planning artifacts. This request does not
start the experiment, create a candidate, modify public source, or authorize a
new runtime, resource purchase, external model call, or upstream action. The
subsequent user request authorized CNC-001 source analysis, and the owner then
accepted the local non-CI scope and CNC-002 static contract work. The
[scope and budget review](../../benchmarks/results/complete-native-correction-v1/contract/local-scope-and-budget-review.md)
records the initial decisions and budget gap. The owner subsequently approved
60 starts maximum within two hours, with a review at 30 minutes. The
[Phase A contract](../../specs/poc-complete-native-correction-v1.md) and static
checker now own those exact limits; this does not launch execution.
Git publication follows the applicable user authority and
shared Git procedure; static preparation does not publish these files.

An authorized phase may proceed through its verified prerequisites without
asking again for routine work. Stop at a failed gate, an exhausted budget, or
a material scope decision. Later phases need their own frozen, authorized
budget; completing an earlier phase does not grant it.

WCNCP and SBIC remain terminal. Their missing candidate and timing phases are
not reopened. This is a directed mechanism study, not an attempt to make either
old breadth result pass. GraphQL Java is selected with knowledge of earlier
diagnostics; it can never count as an unseen prospective success.

## Product hypothesis

The intended customer path remains:

```text
buildoptw -> native Gradle -> bounded observations -> shared coordination
         -> isolated complete correction -> measured proposal -> owner review
         -> owner-applied native patch -> continued native Gradle via buildoptw
```

Gradle owns execution, outputs, and exit behavior. BuildOpt owns optional
observation, evidence, coordination, and reviewable correction delivery. The
native patch must also work without BuildOpt. Cache objects and control records
may share storage infrastructure, but not namespaces or decision authority.

The new unit of work is the smallest complete correction of a workflow's
blockers, rather than one source annotation or one external-process call.
Multiple edits are acceptable only when each is necessary for the same
behavior-preserving correction. This is not general build-script modernization.

## Evidence motivating the plan

| Evidence | What it establishes | What it does not establish |
| --- | --- | --- |
| [Reviewed-native portfolio](../../benchmarks/results/reviewed-native-patch-portfolio-v1/README.md) | Selected Micronaut and Spring corrections pass bounded correctness and value gates. | Discovery prevalence, full-repository speedup, or customer acquisition. |
| [Elasticsearch v2](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/README.md) | A third selected correction saves 7,171.75 ms in its measured workflow; owner-reported review takes 20 seconds. | Willingness to pay or recurrence in other workflows. |
| [WCNCP terminal result](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e009-final.json) | Ten conclusive families, only one actionable material family. | Absence of every possible native correction. |
| [Controlled GraphQL opportunity](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e009b/result.json) | Two native traces expose approximately six seconds of configuration work. | Six seconds of achievable saving or a correct patch. |
| [SBIC terminal result](../../benchmarks/results/source-bound-configuration-input-corrections-v1/sbic-e002-strict-diagnostics/result.json) | Only one of three families binds its expected diagnostic; capture and observability limits remain visible. | A measured failure of the proposed correction mechanism. |
| [Chronological value](../../benchmarks/results/three-class-chronological-value-v1/README.md) | Selected target benefits need not survive ordinary descendant builds. | That all durable native patches have the same lifetime limitation. |

Read the authoritative terminal records, not retracted zero-row WCNCP records
or older recommendation paragraphs. Reuse history to choose questions and
fixtures; every CNC correctness, materiality, value, and lifetime row is fresh.
Do not average percentages across workflows or hide rejected-family costs.

## Subject seed and unresolved source questions

The seed comes from the [WCNCP subject manifest](../../specs/poc-wrapper-coordinated-native-corrections-v1.subjects.json):

| Binding | Historical value to verify in CNC-001 |
| --- | --- |
| Repository | `https://github.com/graphql-java/graphql-java.git` |
| Revision | `f2d8c9126f898c084b176631b7346bc6fbec296a` |
| Git archive SHA-256 | `83e80bbaa2b3e3308dd35e0b7120399f02149507674d958b0e2e4cc81718d051` |
| Build script | `build.gradle` |
| Build-script SHA-256 | `27cfbb5dc699619d3398eb651b9e6cb4dda1d6fd47f87591849d01284841bbbc` |
| Owner workflow | `.github/workflows/pull_request.yml`, arguments `assemble` |
| Historical output declaration | `build/libs/*.jar`, to be resolved into a complete exact file inventory |
| JDK distinction | Owner workflow declares Corretto 25; earlier controlled capture uses locked Temurin 25. Record and resolve this distinction before freezing CNC; do not call the environments identical. |

The [first retained strict log](../../benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e009/diagnostics/graphql-java/configuration-cache-strict-1/gradle.log.gz)
reports five problems:

| Problem | Source location in the retained log | Analysis required before a recipe |
| --- | --- | --- |
| External process | `build.gradle:144` | Worktree detection, command, working directory, streams, exit handling, and every consumer. |
| External process | `build.gradle:169` | Branch identity, detached HEAD, branch changes, error behavior, and every consumer. |
| Build-finished listener | `build.gradle:186` | Success/failure/cancellation effects, ordering, cleanup, and a supported native replacement. |
| Build-finished listener | `build.gradle:938` | Same lifecycle analysis; do not assume both listeners have the same responsibility. |
| Execution-time project access | `:jar` | CNC-001 identifies Bnd 7.1.0 `BundleTaskExtension.java:407`, not a subject closure. Analyze its supported explicit-properties configuration and prove identical bundle outputs. |

The old source span at lines 141-180 does not cover the entire correction.
CNC-001 also distinguishes the local branch-based version from CI's timestamped
version and the single `assemble` command from the owner's multi-command CI.
The owner has now explicitly selected the local non-CI scope. CI's timestamped
version semantics stay unchanged and outside the value claim; preservation of
those paths still belongs in behavioral fixtures.
Inventory all affected declarations and consumers and hash every preimage.
Newly discovered blockers are retained. If they leave the bounded recipe scope,
stop before expanding it. A source-local blocker is not automatically safe to
rewrite, and a successful Configuration Cache store is not complete correctness.

No automatic fallback subject is selected. LSSS remains a later question: its
retained log also contains a `Copy` serialization problem, and its
[BuildProperties source](../../benchmarks/results/source-bound-configuration-input-corrections-v1/sbic-e001-source-detector/sources/lsss/BuildProperties.kt)
reads the clock as well as Git HEAD. Neither that behavior nor Suwayomi's missing
report may be waved away to supply a second success.

## Scope and non-goals

Permitted future implementation, after phase authorization:

- A versioned, owner-reviewable composite recipe using supported APIs of the
  frozen Gradle version, with exact preimages, postimages, and inverse.
- Narrow capture, checker, and installed-path changes necessary to deliver and
  prove the correction using existing BuildOpt seams.
- Isolated subject worktrees, private Gradle homes, bounded local backend
  namespaces, and retained evidence within the frozen budget.

Not included:

- Changing the public workflow, Wrapper, dependency or plugin versions, required
  outputs, or semantics to make the experiment pass.
- Suppressing problems, removing required tasks or listeners without an
  equivalent replacement, or inferring safe normalization from names.
- Reopening retired runtime/profile mechanisms, adding a cache engine, or
  expanding to Android, Kotlin Multiplatform, or Test Optimization.
- Automatic source application in the owner's active checkout, upstream PRs,
  merges, deployments, paid services, or external data/model uploads.
- Soak, design partners, production hardening, or commercial validation as
  prerequisites for the technical POC.

An agent may assist source analysis inside the existing session. Its hypothesis
is not proof. Record intervention type and active time. No additional agents,
model provider, usage setting, or external service is authorized by this plan.

## Existing seams and required implementation

| Existing owner | Reuse | Gap that must be proved before use |
| --- | --- | --- |
| `internal/strictdiagnostic`, `cmd/strict-diagnostic-report-v2` | Root-log-owned report parsing and identical-URI deduplication. | Complete report publication and new phase bindings, including interrupted starts. |
| `dev/capture-strict-diagnostic-reliability` | Native invocation and capture behavior as precedent. | It requires a `.git` directory and creates a clone. Do not invoke it unchanged for CNC: new execution must support shared-Git worktrees and `.git` files. Preserve historical defaults through a separately tested versioned path. |
| `internal/configurationinput`, `internal/configurationinputsource`, `internal/configurationinputbinding` | Diagnostic/source facts. | Individual-process binding does not prove closure of all five blockers or all value consumers. |
| `internal/wcncpdetect`, `internal/wcncpmateriality` | Typed ownership, materiality, and refusal semantics. | A complete blocker set, not an executable or repository name, must justify admission. |
| `internal/wcncpvalidate`, `internal/wcncpreview` | Leases, isolated transactions, digest-bound review, exact revert. | Existing fixture proof is not fresh public candidate proof. |
| `cmd/buildoptw`, `internal/wcncpobserve`, `internal/sharedcache/wcncp_*` | Native passthrough, outbox, control plane, and coordination. | Ordinary onboarding must produce bindings without hand-authored `WCNCP_*` digests; controlled overhead remains unmeasured. |

Extend the nearest owner rather than building a parallel service. Freeze the
recipe only after source analysis; do not prescribe a listener replacement
before understanding its effects. Use source-first generation for generated
schemas or clients. Any API/dependency expansion needs a material decision.

## Phases, budgets, and advancement

### Phase A: directed native mechanism, CNC-001 through CNC-008

The original 40-start allocation was incomplete because nine minimal lifecycle
starts already exceeded its six fixture slots. The owner approved the complete
replacement below. The [contract](../../specs/poc-complete-native-correction-v1.md)
maps every slot, input, command, state dependency and proof obligation.

Approved ceiling: **60 total real Gradle starts, one candidate transaction,
four workers, one start at a time**. Every real Gradle invocation counts,
including fixture builds, preparation, version
probes, owner tests, failed starts, stabilization, and retries. Static analysis,
Go-only tests, and fake-child fixtures use no Gradle-start slot but consume time.

| Allocation | Maximum starts |
| --- | ---: |
| Real-Gradle recipe and runner fixtures | 24 |
| Equivalent native preparation/prefetch | 2 |
| Fresh strict diagnostics | 2 |
| Controlled native materiality | 2 |
| Public candidate correctness, owner tests, and behavior probes | 10 |
| Native paired value: two excluded stabilizations plus eight pairs | 18 |
| Classified infrastructure replacement reserve | 2 |
| **Total** | **60** |

CNC-002 maps every required proof to these slots. Several assertions may share
a start only when its raw evidence proves
each assertion. If the complete matrix cannot fit, stop for a revised budget
before execution; do not drop tests or hide auxiliary builds. The reserve is
not for retrying a slow, regressive, or incorrect candidate. At most two
infrastructure replacements are allowed and each retains its original row.

The owner-approved time limit is **two elapsed hours**, replacing the earlier
eight-hour proposal. Its monotonic execution window starts before the first
execution-phase preparation command and includes downloads, setup, waits,
fixtures, builds, and validation, not only accepted timings. Review progress
at 30 minutes against remaining prerequisites and the remaining window; stop
early if no useful progress is possible. This checkpoint does not reset or
extend the deadline. Human waiting is recorded separately and cannot silently
reset it. All runs have a 20-minute timeout shortened to the remaining phase
budget. At the deadline, stop execution, retain partial evidence, and mark
unfinished proof incomplete; never drop required tests to fit the limit.

### Phase B: installed product and persistence, CNC-009 through CNC-011

Do not squeeze this into Phase A or call Phase A a complete product result.
At CNC-009, propose and authorize a separate budget after inspecting the seed
result. A sizing estimate is **160 real Gradle starts / 12 elapsed execution
hours**, not an execution grant:

- 100 overhead observations: five modes, twenty alternating observations each;
- 27 installed-value starts: three stabilizations and eight three-arm blocks;
- 20 chronological starts: five revisions, two requests, two compared arms;
- eight integration/behavior fixture starts; and
- five preparation slots, of which at most two may instead be classified
  infrastructure replacements.

The overhead modes are observation off, local enqueue, acknowledged upload,
queued-offline upload, and status lookup. Freeze their direct-native comparators
and event boundaries; a status command is not itself a Gradle start. If the
existing overhead protocol requires more observations, or ordinary workflows
and the short-workflow probe need separate rows, revise the budget before the
phase starts. Do not weaken the protocol to fit this estimate.

### Phase C: unseen replication, CNC-012 through CNC-013

Select three previously unused families from pre-outcome technical eligibility,
not measured savings. Freeze all revisions, workflows, correction capabilities,
selection/exclusion records, output contracts, and exact phase budgets before
the first new build. The Phase A/B counts do not authorize three more campaigns.
There is no automatic reserve substitution after observing a bad result.

Proposed advancement criterion: at least two independent families with complete
correctness, installed net value, and positive measured chronological value,
plus a positive total operational ledger including unsuccessful families.
This is a small replication gate, not market prevalence. The exact horizon and
complete per-family start allocation must be approved in CNC-012; no Phase C
execution is possible while these inputs are missing.

### Shared resource and failure rules

- At most 20 GiB additional concurrent disk and at least 10 GiB free before
  every new start, including preparation. Charge checkouts, caches, outputs,
  logs, and retained artifact copies; compressed size alone is not disk spend.
- Use this host, mains power, the established four-worker resource envelope,
  and no concurrent agent-launched build or benchmark. Do not stop unrelated
  processes or silently change power settings.
- Preserve existing worktrees. New subject isolation uses shared-Git
  worktrees, not clones or plain source copies. Do not mutate shared refs or
  existing owner worktrees during branch/commit behavior probes.
- Preserve raw evidence before requesting exact-path cleanup. Worktree removal
  requires the applicable cleanup approval; never use forced cleanup.
- Count failed corrections across commits and compaction under the user
  operating contract. An intentional negative fixture is not a failed fix.
  A product correctness failure in the frozen public candidate rejects it and
  stops downstream timing; it is not relabeled as infrastructure.

## Proof contracts

### Freshness and complete diagnostic closure

Freeze source archives, exact toolchains, executable/package digests, ordered
arguments, environment deltas, cache policy, output contract, protocol versions,
and baseline definition. Record actual tested bytes as well as commit SHAs.
Each started child gets an immutable attempt ID and retained terminal status,
including timeout, cancellation, publication failure, and no-report success.

Expected strict-native Configuration Cache failure is diagnostic evidence, not
a product-attributable failure. The no-patch executable control uses the best
compatible native settings; do not compare a successful candidate to a failing
strict diagnostic. Both value arms get the same native build-cache opportunity.

Reconstruct all reported blockers and inspect undeclared external state.
Require source ownership, exact declaration/call-site bindings, affected
consumers, error behavior, and a proposed correction for the entire set.
Report silence does not prove safety. In particular, version, branch, and clock
inputs may invalidate every build or change outputs and must not be suppressed.

### Correctness and reversibility

Before timing, prove:

1. A successful no-patch native control with the exact requested outputs.
2. Candidate Configuration Cache store and subsequent reuse, without new or
   remaining blockers and without skipping required work.
3. Byte-exact equality of the complete required output inventories; if even
   native/native equality is unstable, classify it and stop this exact-output
   plan rather than introducing a normalization exception.
4. Relevant source, branch/HEAD, environment, and other consumed-input changes
   invalidate correctly; an unrelated change does not invent equivalence.
5. Equivalent external-process exit/stream behavior, listener effects and
   ordering, task execution, and cleanup on relevant failure/cancellation paths.
6. Exact apply/reapply/revert/rerevert with source drift and tampering refused.
7. The frozen focused owner tests. Cross-root proof is mandatory whenever the
   proposal claims relocation or cache portability.

The CNC-002 proof matrix must name the fixture or real invocation covering each
case. A plain successful `assemble` is not sufficient. No shared success summary
may conceal an untested prerequisite.

### Controlled materiality and native value

The existing 500-ms and 2% materiality floor is an admission screen, not a
speedup estimate. Recompute it from fresh unpatched native operation/DAG data.
After prefetch, wait the existing fixed 120-second quiescence interval and
apply the seven-sample maximum/minimum ratio limit of 1.15. Also retain actual
build durations and environment drift: a short CPU probe does not certify
stability of the entire Java workload.

Run eight balanced AB/BA native pairs with independent symmetric control and
candidate state and one excluded stabilization per arm. Bind the whole command,
not only the formerly blocked task. Measure with an external monotonic envelope.
Keep raw rows, order, cache outcomes, exact outputs, and all costs.

Native value requires eight favorable pairs, mean saving of at least 500 ms
and 2%, a positive paired 95% interval, non-regressive candidate p95, and zero
additional product failures. Reuse the repository's verified statistical
definition in the new checker; freeze its implementation before data capture.
No repeated parameter search or favorable-subset selection is allowed.

Hosted CI owns deterministic contracts and correctness, never the numerical
performance gate. Unstable measurement is `INCOMPLETE_PERFORMANCE_ENVIRONMENT`,
not proof that the product is slow, and not permission to loosen thresholds.

### Installed value and chronological persistence

Define the three arms before CNC-011:

- `N0`: optimized native Gradle without the correction;
- `N1`: the same native workflow with the frozen correction;
- `W1`: the same corrected workflow through the actual packaged wrapper and
  its normal observation/backend path.

Fresh three-arm blocks distinguish native correction value (`N0-N1`), wrapper
cost (`W1-N1`), and complete installed value (`N0-W1`). Never subtract averages
from unrelated experiments. Rotate order using the frozen assignment schedule
and give every arm equivalent preparation and cache access.

Preserve the existing local-enqueue overhead requirements: at most 50 ms p95
incremental pre/post-child time and at most 0.5% mean overhead on workflows
lasting at least ten seconds. Other modes must prove their separately frozen
bounded behavior. Product qualification requires the installed arm to pass the
native-value thresholds, not merely that the patch itself is faster.

Freeze five consecutive eligible descendant revisions by chronology before
candidate outcomes, with two declared requests per revision: the first after
the change and one repeat. Do not skip incompatible or losing descendants.
Compare `N0` and `W1` from symmetric initial state, retaining each arm's evolving
native caches. Realized signed savings must remain positive; report changing
inputs, actual reuse, rejection/fallback, and cost of every row. If the required
history is unavailable, record the limitation before testing; synthetic
mutations cannot be presented as actual chronological use.

Apply the same frozen patch where its preimages remain valid; otherwise retain
native and count the overhead. Do not hand-repair the patch for each descendant.
Intentional semantic mutation fixtures remain separate from the chronological
sample. Projected payback must not be called observed repayment.

### Economics and owner review

Keep separate ledgers for research, operational compute/storage/network,
customer-visible latency, and human analysis/review. Record costs for failures,
rejected proposals, diagnostics, stabilization, and native fallbacks. Attribute
customer costs from the first CNC observation, not only from acceptance.

Report machine payback in compatible builds, retaining the 300-build screening
ceiling for this proposed route. Human seconds are reported separately; monetary
payback needs explicit labor, runner, backend, and optional model-cost rates.
Do not add unlike resources and label the total commercial ROI. A pooled
portfolio saving is not the payback of one customer's repository.

Only a value-qualified installed proposal enters first-exposure owner review.
Present the exact diff, behavior obligations, outputs, uncertainty, complete
cost ledger, and revert instructions. Record comprehension, concerns,
acceptance/rejection, and actual active review time without inventing it.
Acceptance alone never proves that the patch was applied or that anyone will
pay for the product. The existing 15-minute active-review screen is retained;
missing review evidence leaves product completion partial.

## Artifact and command interface

The human/machine contract, subject manifest and `contract` checker mode exist.
CNC-003 now has a versioned native runner, generic and explicit host fixtures,
independent attempt/output checking and a [literal runbook](../reference/complete-native-correction-capture.md).
Detached ownership and private input/output consuming paths are proved with
fake children, not public Gradle. CNC-004 recovered the exact real inputs, then
refused before Gradle on a runtime-directory verifier defect. Its repair is
proved against both actual archives; the [preflight record](../../benchmarks/results/complete-native-correction-v1/cnc004-preflight/README.md)
preserves the first package/clock. The separately approved second window reached
[successful native preparation followed by a private-home refusal](../../benchmarks/results/complete-native-correction-v1/cnc004-private-home/README.md).
The generated Java/Kotlin state repair is proved against the retained real home
and consuming fixtures; the failed row stays unchanged. No D/M/value proof exists.

| Planned owner | Purpose |
| --- | --- |
| `specs/poc-complete-native-correction-v1.md` and `.json` | Human/machine Phase A contract and invariant/budget definitions. |
| `specs/poc-complete-native-correction-v1.subjects.json` | Exact source, workflow, toolchain, output, and proof-case bindings. |
| `dev/run-complete-native-correction` | A bounded entrypoint extending existing capture/transaction owners. |
| `dev/check-complete-native-correction` | Static `contract`, generic `fixtures`, explicit local `host-fixtures`, and read-only attempt `capture` modes; no public Gradle. |
| `benchmarks/results/complete-native-correction-v1/` | Separate CNC evidence index, immutable attempts, artifacts, and terminal decisions. |

Run `./dev/check-complete-native-correction contract` for the static freeze.
Implemented native runner modes are `freeze`, `package-check`, `init`, `guard`,
`preflight`, `seed`, `stabilize`, `review`, `capture` and `check`, with exact flags
in the runbook. Only `capture` may start a native Gradle child, after all gates.
`correctness`, `native-value`, `installed-value` and `chronology` remain future
interfaces, not executable commands. The checker accepts `contract`, generic
`fixtures`, explicit local `host-fixtures` and read-only attempt `capture`.
Planned aggregate modes are `phase-a`, `phase-b`, `replication`, and `terminal`. The checker must
not launch public builds.

Freeze actual CLI syntax and a copy-ready, checked runbook in CNC-003. Test
unknown phases, missing flags, occupied destinations, plan/source drift,
unregistered worktrees, budget exhaustion, and resumption. Do not present the
names in this section as tools that already exist.

The evidence layout separates `selection/`, `contract/`, `fixtures/`,
`attempts/`, `native-value/`, `installed/`, `chronology/`, `replication/`,
`review/`, and `decisions/`. Every stage records raw inputs and hashes; later
results reference them rather than silently replacing them. A root manifest
records authority, package/recipe identity, actual spend, and the next step.

## Terminal decisions and handoff

| Decision | Meaning and next boundary |
| --- | --- |
| `QUALIFY_DIRECTED_NATIVE_MECHANISM` | Phase A passed for the selected workflow only; installed product and replication remain unproved. |
| `QUALIFY_INSTALLED_NATIVE_CORRECTION` | Installed correctness, overhead, value, chronology, and review passed for the selected case; no unseen-family claim. |
| `QUALIFY_BOUNDED_NATIVE_CORRECTION_POC` | The frozen independent replication gate and total operational ledger pass; no commercial or universal claim. |
| `STOP_INCOMPLETE_SAFE_CORRECTION` | Complete behavior-preserving blocker closure is unavailable in scope. |
| `STOP_NATIVE_ADMISSION` | Fresh native inputs fail the existing materiality screen or expose unstable required outputs before a recipe/candidate exists; no candidate-value conclusion. |
| `STOP_PRODUCT_CORRECTNESS_FAILURE` | The candidate or installed path breaks a required behavior/output. |
| `STOP_NO_MATERIAL_NATIVE_VALUE` | A correct patch fails controlled native value. |
| `STOP_INSTALLED_COST_OR_PERSISTENCE` | Wrapper cost or ordinary changes eliminate qualified value. |
| `STOP_INSUFFICIENT_REPLICATION` | The prospective phase misses its independently frozen breadth/economic gate. |
| `INCOMPLETE_EXPERIMENT_INPUT`, `INCOMPLETE_PERFORMANCE_ENVIRONMENT`, `INCOMPLETE_EXPERIMENT_BUDGET_EXHAUSTED` | State the exact missing input or exhausted limit; no value inference and no automatic restart. |

Every exit produces a scoped decision with executed/unexecuted steps and costs.
Passing Phase A is a milestone, not completion of this full roadmap. Failure
does not authorize a substitute subject, altered recipe scope, or another run.

At implementation milestones update the evidence README, this plan only when a
decision changes, the [tracker](./complete-native-correction-poc-tracker.md),
documentation/specification indexes, validation reference, implementation
tracker/evidence ledger, generalization audit, performance findings, and handoff
one-pager. Register only implemented contracts/checkers in layout and Base CI.
English repository text and portable shared configuration remain mandatory.

After technical qualification, commercial discovery is a separate owner
decision: investigate actual demand, willingness to pay, service effort, and
whether improvements recur enough for a subscription. No sales contact, hosted
deployment, design-partner requirement, or billing work is part of this plan.
