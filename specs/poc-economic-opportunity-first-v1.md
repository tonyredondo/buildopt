# Economic Opportunity First POC v1

Status: active at `EOF-002`; `EOF-001` freezes planning authority only.

## Hypothesis

BuildOpt may become economically viable by rejecting unlikely-to-repay
optimizations before it spends customer wall time on discovery, calibration or
materialization. A source-only, no-lookahead preflight must identify recurring
compatible opportunities cheaply enough that the complete installed path is
net positive against optimized native Gradle in at least three of five public
repository families.

This route tests opportunity selection, not another optimization mechanism.
It may reuse existing generic BuildOpt mechanisms only after they satisfy the
same exact-output and fallback contracts. Historical experiments motivate the
features and budgets below, but no prior report, timing pair, profile or output
state can satisfy an EOF evidence row.

## Frozen cohort and evidence separation

The adjacent subjects file freezes five public repository families and exact
anchor revisions. `EOF-002` must derive a chronological first-parent window for
each family using commits strictly older than its evaluated commit. Every row
records the source revision, parent, changed-path digest, workflow digest,
feature digest and source SHA-256s used by the decision.

Repository, organization, task and workflow names are labels only. They may be
emitted for review but must not enter classification, thresholds or scoring.
The five families are evaluated independently; percentages and timings are
never added across mechanisms or repositories.

## Source-only opportunity model

Before Gradle starts, the preflight may use only:

- canonical requested entrypoints and flags;
- first-parent Git metadata and changed paths available locally;
- source-owned Gradle files and a previously verified graph whose source,
  Wrapper, toolchain, workflow and build-logic bindings still match; and
- prior EOF observations whose event time precedes the evaluated commit.

It must not run Gradle, inspect build outputs, read a future commit, use an EOF
candidate result as an input to its own prediction, or consume predecessor JSON
as evidence. Missing, stale or ambiguous facts produce a typed rejection.

The source screen emits one of `ADMIT_NATIVE_CEILING_PROBE`,
`REJECT_INSUFFICIENT_RECURRENCE`, `REJECT_INSUFFICIENT_VALUE_CEILING`,
`REJECT_EXCESSIVE_PAYBACK`, `REJECT_BINDING_DRIFT`,
`REJECT_INCOMPLETE_OR_AMBIGUOUS`, or `NO_ACTION`.

## Economics

All calculations use signed milliseconds. For an opportunity:

`planningGrossMs = compatibleMatchesLowerBound * potentialSavedMsPerMatch`

`planningNetMs = planningGrossMs - qualificationBudgetMs -
materializationBudgetMs - nativeRetentionBudgetMs - selectionBudgetMs`

`planningPaybackMatches = ceil(totalIncrementalCostMs / potentialSavedMsPerMatch)`

The lower bounds must be computed without future observations. Unknown value
is unavailable, never zero. Admission requires all of:

1. complete, non-drifted source bindings;
2. at least five compatible matches in the frozen prior-history horizon;
3. a fresh optimized-native ordinary observation exposes enough avoidable
   critical-path wall time to define a positive planning potential;
4. positive planning net value;
5. planning payback within five compatible matches; and
6. a preflight duration no greater than 500 ms.

Planning potential is an authorization budget, not a saving claim. The first
fresh candidate/control proof must replace it with a measured positive lower
bound. If it cannot, the opportunity returns to native Gradle and cannot enter
installed value measurement.

## Ordered blocks

- `EOF-001` freezes this contract, exact cohort, formulas, budgets, authority,
  documentation ledger and executable drift checker.
- `EOF-002` reconstructs a fresh chronological source-only recurrence ledger.
  All five families must be conclusive and at least three must expose an
  `ADMIT_NATIVE_CEILING_PROBE` row. It runs no Gradle build.
- `EOF-003` implements the versioned preflight, proves deterministic replay,
  no-lookahead behavior, name invariance, source drift rejection and a 500-ms
  local decision budget, then collects one fresh optimized-native ordinary
  observation per admitted family. At least three planning opportunities must
  have positive five-match planning net value before a candidate is authorized.
- `EOF-004` runs one minimal candidate/control proof per admitted family. It
  requires exact outputs, zero product failures, successful native fallback and
  a positive lower saving bound. No eight-pair calibration is allowed here.
- `EOF-005` runs the installed chronological campaign using only requested
  ordinary builds. At least twenty later first-parent commits per admitted
  family are retained with sign, including every rejection and native fallback.
- `EOF-006` issues the terminal scorecard and installed explanation. Product
  viability requires at least three of five net-positive families, a positive
  signed portfolio, finite payback, exact outputs and zero product failures.

A failed prerequisite closes dependent blocks as `NOT_AUTHORIZED`. Hosted CI
owns deterministic contracts and correctness, never wall-time gates.

## Budgets and stop conditions

- source-only preflight: maximum 500 ms per decision and 1,000 ms p95;
- minimal proof: one balanced candidate/control pair per admitted family;
- payback horizon: at most five compatible matches;
- installed campaign: at least twenty later commits per admitted family;
- product failure budget: zero;
- exact required outputs: mandatory;
- viability breadth: at least three of five net-positive families; and
- signed aggregate net value: greater than zero.

Stop if source completeness is below 5/5, recurrence breadth is below 3/5, any
product failure occurs, minimal proof lacks a positive lower bound in three
families, or installed value misses either breadth or signed portfolio value.
Thresholds do not move after evidence is observed.

## Non-goals

- repository-specific rules, manual profile curation or retrospective cohort
  replacement;
- another task detector, public-source patch or cacheability correction;
- production rollout, soak, design partners, SLOs or automatic merge;
- Test Optimization; and
- claiming that isolated historical wins prove current product viability.

## Verification

```bash
./dev/check-economic-opportunity-first
```
