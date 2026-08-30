# Verified request hit POC tracker

## Status

**Overall:** `ACTIVE`

**Current block:** `VRH-003 — shadow replay`

**Completed:** `VRH-001 — eligibility and capture-diagnostic audit`; `VRH-002 —
complete fail-closed safety contract`

**Next decision:** predict only identities admitted by the complete safety
contract, run native Gradle unchanged and compare the prediction with the real
outcome and repository effects.

## POC objective

Show that a generic BuildOpt wrapper can reduce cumulative customer Gradle wall
time beyond optimized native Gradle by combining two evidence-driven actions:

- `VERIFIED_REQUEST_HIT` for an exact recurrent request whose complete inputs
  and execution facts are unchanged; and
- the already observed exact partial graph for relevant changes.

Everything else runs the exact native command. The POC succeeds only through
measured customer wall-time value with byte-equivalent required outputs; action
counts alone are not value.

## Why this route exists

The previous route found exact partial-graph actions in four families but
stopped before timing because only three families supplied its preregistered
complete input. Its 113 transitions also expose a different opportunity:
69 commits do not intersect the exact request inputs. Gradle's native cache can
reuse task outputs in those cases, but Gradle still starts and configures the
build. A safe whole-request hit could avoid that fixed cost.

This is not a rename of `NATIVE_NOOP` and not a revival of Hot State:

- `NATIVE_NOOP` delegates to Gradle;
- Hot State still executed Gradle and previously regressed total wall time; and
- `VERIFIED_REQUEST_HIT` must materialize verified outputs and complete without
  starting Gradle.

## Frozen eligibility result

[`verified-request-hit-eligibility-audit-v1.json`](../../benchmarks/results/verified-request-hit-eligibility-audit-v1.json)
reclassifies no historical row. It groups the independently reconstructed
statuses by the action each future contract would need:

| Family | Partial-graph rows | Potential request-hit rows | Native fallback | Request-hit input |
| --- | ---: | ---: | ---: | --- |
| Apache Groovy | 5 | 0 | 3 | insufficient |
| Apache Kafka | 5 | 17 | 2 | sufficient |
| Micronaut Core | 2 | 14 | 11 | sufficient |
| OpenTelemetry Java Instrumentation | 0 | 22 | 5 | sufficient |
| Spring Framework | 5 | 16 | 6 | sufficient |
| **Total** | **17** | **69** | **27** | **4/5 families** |

The threshold was frozen at at least five potential rows in at least three
families. It passes at 4/5. The 86/113 combined rows are only a 76.1%
structural ceiling: none is yet safe, selected, executed or timed.

## Ordered blocks

| Order | Block | Deliverable | Done when | State |
| ---: | --- | --- | --- | --- |
| 1 | `VRH-001` Eligibility and diagnostic audit | Independent action-potential counts; preserve historical provenance; bounded forward diagnostics | 113 rows partition exactly, 4/5 potential-hit threshold passes, 8 unavailable rows remain typed, no authority invented | `DONE` |
| 2 | `VRH-002` Safety contract | Versioned request-hit record, verifier and negative matrix | Every required fact is represented; all 37 missing/drifted/unsafe fixtures retain native; zero Gradle/action/timing | `DONE` |
| 3 | `VRH-003` Shadow replay | Predict hits but execute native Gradle and compare result/output states | Repeated exact matches across Gradle 8/9, Kotlin/Groovy and eligible public rows; mismatch quarantine works | `TODO` |
| 4 | `VRH-004` Gradle-free execution | Verified output materialization and successful return without Gradle | Process-level proof that Gradle never starts; exact outputs/outcome; fail-open native fallback | `WAITING` |
| 5 | `VRH-005` Installed paired value | Controlled balanced candidate/native measurements | Positive signed mean and lower bound, non-regressive p95, exact outputs and full cost accounting in at least three families | `WAITING` |
| 6 | `VRH-006` Chronological combined value | Request hit + partial graph + native policy over all five histories | Signed cumulative savings, hit/action/fallback counts, payback and native-retention overhead; no averaged repository percentages | `WAITING` |
| 7 | `VRH-007` Terminal decision | Digest-bound continue or stop scorecard | Correctness, installed value, cumulative value, breadth, overhead and failure gates evaluated without threshold movement | `WAITING` |

## Autonomous contracts

### VRH-001 — Eligibility and diagnostic audit

Inputs are limited to the independently checked breadth result, its exact
transition ledger and the closed terminal decision. The runner must not consume
old timing, claim safety from `IRRELEVANT_TO_REQUEST`, or turn an unavailable
capture into an irrelevant row. It partitions every transition exactly once:

- `RELEVANT_COMPLETE` → existing partial-graph opportunity;
- `IRRELEVANT_TO_REQUEST` → potential `VERIFIED_REQUEST_HIT` input;
- `GLOBAL_OR_AMBIGUOUS` or `INPUT_UNAVAILABLE` → native fallback.

The eight unavailable rows remain unavailable. The original campaign init
script is frozen under its registered SHA-256 so current diagnostics can evolve
without rewriting historical provenance. Future task-input failures may expose
only sorted task paths, exception class names and truncation; messages and
absolute paths are forbidden. Diagnostics never change status or authority.

### VRH-002 — Complete safety contract

Create a versioned, canonical record and verifier. The record must bind:

1. exact framed argv and repository-relative working directory;
2. repository identity, Wrapper, Gradle, JDK and safe environment;
3. finalized requested graph and task implementation/build logic;
4. complete transitive repository inputs plus external-input identities;
5. complete present and absent output states;
6. local state, destroyables, untracked writes and side-effect eligibility;
7. task cacheability/tracking facts and source outcome; and
8. immutable content-addressed materialization references.

Required negatives include argument/cwd drift, Wrapper/Gradle/JDK/environment
drift, graph/build-logic drift, changed repository input, missing or changed
external input, untracked output, local state/destroyable declaration,
side-effectful/always-run/untracked task, missing output, altered output,
previous failure/cancellation and expired/revoked evidence. Every negative must
produce typed native retention. The block starts no extra Gradle build, selects
nothing and measures no time.

This block is complete. The closed Draft 2020-12 record and Go verifier bind
all eight required fact families. Argument identity uses an exact uint64
length-framed byte sequence; repository paths remain portable; present outputs
carry content-addressed references while expected absence is explicit. A
present matching workspace output is acceptable, and a missing workspace
output is acceptable only when its exact immutable materialization object is
available. An altered workspace output, missing or corrupt object, or
unexpected formerly absent output retains native Gradle before any write.

The committed matrix contains one complete fixture and 37 independent negative
mutations. All 37 return `RETAIN_NATIVE_GRADLE` with their expected typed
reason. The evaluator starts zero Gradle invocations, selects zero actions and
collects zero timing samples. `SAFETY_CONTRACT_COMPLETE` is classification
evidence for shadow replay only; it is not a `VERIFIED_REQUEST_HIT` action.

### VRH-003 — Shadow replay

Use only identities admitted by VRH-002. BuildOpt predicts whether the request
would hit but always runs the exact native command. Compare exit code, present
and absent outputs, expected repository effects and verifier decision. A single
mismatch quarantines the identity and requires new evidence; it may not be
averaged away. Hosted CI may exercise correctness fixtures but owns no wall-time
threshold.

### VRH-004 — Gradle-free execution

After the preregistered repeated-shadow threshold passes, implement the real
action. Restore all required outputs atomically, verify them, return the bound
successful result and prove at process level that no Gradle launcher, daemon or
child JVM starts. Any restore/verify failure removes partial materialization and
runs native Gradle. The user-facing action is `VERIFIED_REQUEST_HIT`; never call
it `NATIVE_NOOP` or a native cache hit.

### VRH-005 — Installed paired value

Install the same package a user receives. On the controlled runner, execute at
least eight balanced alternating pairs per admitted family against optimized
native Gradle with identical dependency/cache opportunities. Time outside both
arms, include lookup, restore, verification and fallback, and validate exact
outputs. Hosted CI reruns correctness only. The threshold must be frozen before
measurements and may not be weakened after results appear.

### VRH-006 — Chronological combined value

Replay the five public first-parent histories with persistent but arm-isolated
state. Apply one generic policy: potential irrelevant requests may become
verified hits; relevant rows may use a separately qualified exact partial
graph; global, ambiguous, unavailable or unsafe rows run native. Record every
negative transition and report signed cumulative milliseconds per family and
overall. Do not add or average percentages from different workloads.

### VRH-007 — Terminal decision

Continue only if there are zero correctness/product failures, the installed
action beats optimized native Gradle with confidence in at least three families,
chronological cumulative savings are positive, payback is finite and native-
retention overhead stays within its frozen budget. Otherwise stop this
hypothesis and preserve all negative evidence. A terminal stop authorizes no
automatic successor.

## Documentation ledger

Every block updates this tracker, the machine contract, benchmark index,
validation/developer reference, implementation tracker, generalization audit,
performance findings and POC one-pager. Runtime changes also update architecture,
configuration, onboarding and troubleshooting documentation. No document may
present structural opportunity as safe eligibility or measured speedup.

## Explicit non-goals

- production soak, HA, tenancy, RBAC or design-partner qualification;
- Test Optimization;
- repository-specific profiles, task allowlists or CI-file policy inference;
- reopening Runtime Tuning, Hot State, Copy or cache parameter searches; and
- Patch Autopilot before this route reaches its terminal decision.
