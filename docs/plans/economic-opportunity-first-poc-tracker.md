# Economic Opportunity First POC Tracker

**Status:** closed — `STOP_ECONOMIC_OPPORTUNITY_FIRST_POC`<br>
**Current block:** none; all six blocks are resolved<br>
**Terminal outcomes:** `CONTINUE_ECONOMIC_OPPORTUNITY_FIRST_POC` or
`STOP_ECONOMIC_OPPORTUNITY_FIRST_POC`

## Objective

Test whether BuildOpt can avoid the economic failure shared by earlier generic
routes: spend almost nothing on opportunities that will not recur, then apply
the smallest exact-output proof only where a conservative five-match payback is
already plausible. Optimized native Gradle remains the control and fallback.

## Frozen decisions

- Previous reports are design inputs only and cannot satisfy an EOF row.
- The exact five-family anchor cohort is frozen before source reconstruction.
- Decisions use chronological source facts only; future observations and names
  cannot affect classification.
- Every incremental selection, qualification, materialization and native-
  retention cost enters signed net value.
- Unknown value is unavailable, not zero. Thresholds never move after results.
- A failed prerequisite closes dependent blocks as `NOT_AUTHORIZED`.

## Blocks

| Block | Deliverable | Entry gate | Exit gate | State |
| --- | --- | --- | --- | --- |
| `EOF-001` | Human/machine contract, exact cohort, formulas, budgets, authority, checker and documentation ledger | User-authorized successor analysis after closed NAC v2 | Contract and indexes pass without candidate builds or new timing | `DONE` |
| `EOF-002` | Fresh chronological source-only recurrence ledger | EOF-001 | 5/5 conclusive families and at least 3/5 `ADMIT_NATIVE_CEILING_PROBE`; predecessor reports supply no row and no Gradle build runs | `DONE_STOP` — 5/5 conclusive, 1/5 probe families |
| `EOF-003` | Versioned deterministic economic preflight plus fresh optimized-native ceiling observation | EOF-002 | Replay, no-lookahead, name invariance and source-drift negatives pass within budget; at least 3/5 planning opportunities are positive before a candidate | `NOT_AUTHORIZED` |
| `EOF-004` | One minimal exact-output pair per admitted family | EOF-003 | Positive lower bound, exact outputs, native fallback and zero product failures in at least 3/5 families | `NOT_AUTHORIZED` |
| `EOF-005` | Installed ordinary-build chronological campaign | EOF-004 | At least twenty later commits per admitted family; 3/5 net positive, signed portfolio positive and finite payback | `NOT_AUTHORIZED` |
| `EOF-006` | Installed explanation and terminal scorecard | EOF-005 or first failed prerequisite | Truthful terminal decision with unavailable fields preserved | `DONE` — terminal scorecard only |

## Evidence schema required next

Every `EOF-002` row records family label, repository URL, evaluated revision and
parent, changed-path digest, workflow digest, feature digest, source SHA-256s,
prior-history boundary, compatible-match lower bound, typed source decision and
reason. An independent checker reconstructs rows and family counts
instead of trusting aggregates.

The ledger must be generated from source and Git history. Reading a predecessor
benchmark JSON while constructing a row is a contract violation. Prior evidence
may be cited separately as motivation but never copied into the fresh ledger.

## Measurement and authority

`EOF-001..003` run no candidate Gradle build and create no speedup claim.
`EOF-003` may record one fresh optimized-native ordinary observation per source-
admitted family solely to bound planning potential.
`EOF-004` is the first block that may run a candidate, and only after the 3/5
source admission gate. Hosted CI validates contracts and correctness; controlled
local execution owns timing. No block patches public source or authorizes
production, automatic merge, soak, design partners or Test Optimization.

## Stop conditions

- fewer than 5/5 conclusive source families or 3/5 ceiling probes;
- lookahead, name-dependent classification, ambiguous binding or source drift;
- preflight above 500 ms for one decision or 1,000 ms p95;
- any required-output mismatch or product failure;
- fewer than three positive minimal proofs; or
- fewer than three net-positive installed families or non-positive signed
  portfolio value.

## EOF-002 result

The checked ledger contains 320 fresh rows: 64 chronological commits for each
frozen family. All source bindings, parents, changed-path digests, workflow
digests and feature digests are present; Gradle starts, candidate builds,
timing samples, public-source writes and predecessor evidence inputs are zero.

| Family | Owner-only matches | Decision |
| --- | ---: | --- |
| Apache Kafka | 11 | `ADMIT_NATIVE_CEILING_PROBE` |
| Spring Framework | 3 | `REJECT_INSUFFICIENT_RECURRENCE` |
| Apache Groovy | 0 | `REJECT_INSUFFICIENT_RECURRENCE` |
| Micronaut Core | 0 | `REJECT_INSUFFICIENT_RECURRENCE` |
| OpenTelemetry Java Instrumentation | 0 | `REJECT_INSUFFICIENT_RECURRENCE` |

The frozen gate requires three probe families. At 1/5, `EOF-003..005` are
closed as `NOT_AUTHORIZED`; `EOF-006` records the terminal stop.

## Terminal decision and recommendation

The route closes as `STOP_ECONOMIC_OPPORTUNITY_FIRST_POC`. Source completeness
passes; recurrence breadth fails; every later economic, correctness and value
field remains `NOT_MEASURED_NOT_AUTHORIZED`. No successor is authorized by the
scorecard.

The strongest remaining product-level hypothesis is remote-cache locality:
compare optimized native Gradle reading the same remote cache directly with
native Gradle reading the same objects through a prewarmed verifying BuildOpt
Edge/L1. This removes per-change profile recurrence from the value path and
tests a capability that already has bounded positive mechanism evidence. A
future `REMOTE_CACHE_LOCALITY_VALUE_V2` must freeze equal cache opportunity,
network conditions, five-family breadth, exact outputs, outage/corruption
fallback and signed installed overhead before any timing. This recommendation
is not successor authority.

## Documentation ledger

Every block updates this tracker, the human and machine contract, specification
and benchmark indexes, validation reference, implementation tracker/evidence
ledger, generalization audit, performance findings and POC one-pager. Runtime
or installed-command changes additionally update architecture, onboarding,
configuration and troubleshooting documentation.
