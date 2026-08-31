# Economic Opportunity First POC Tracker

**Status:** active<br>
**Current block:** `EOF-002` — fresh chronological source ledger<br>
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
| `EOF-002` | Fresh chronological source-only recurrence ledger | EOF-001 | 5/5 conclusive families and at least 3/5 `ADMIT_NATIVE_CEILING_PROBE`; predecessor reports supply no row and no Gradle build runs | `TODO` |
| `EOF-003` | Versioned deterministic economic preflight plus fresh optimized-native ceiling observation | EOF-002 | Replay, no-lookahead, name invariance and source-drift negatives pass within budget; at least 3/5 planning opportunities are positive before a candidate | `WAITING` |
| `EOF-004` | One minimal exact-output pair per admitted family | EOF-003 | Positive lower bound, exact outputs, native fallback and zero product failures in at least 3/5 families | `WAITING` |
| `EOF-005` | Installed ordinary-build chronological campaign | EOF-004 | At least twenty later commits per admitted family; 3/5 net positive, signed portfolio positive and finite payback | `WAITING` |
| `EOF-006` | Installed explanation and terminal scorecard | EOF-005 or first failed prerequisite | Truthful terminal decision with unavailable fields preserved | `WAITING` |

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

## Documentation ledger

Every block updates this tracker, the human and machine contract, specification
and benchmark indexes, validation reference, implementation tracker/evidence
ledger, generalization audit, performance findings and POC one-pager. Runtime
or installed-command changes additionally update architecture, onboarding,
configuration and troubleshooting documentation.
