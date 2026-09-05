# Source-Bound Configuration-Input Corrections POC

**Overall:** `STOP_INSUFFICIENT_DIAGNOSTIC_BINDING`<br>
**Current block:** `SBIC-006` terminal documentation is complete<br>
**Next action:** none; a materially different successor needs a new contract

## Objective and decision boundary

SBIC tests the strongest remaining reviewed-native product hypothesis: source
facts identify a configuration-time external read before BuildOpt spends a
strict diagnostic, and a small native Gradle correction makes the owner's
workflow Configuration Cache compatible and materially faster.

This is a mechanism study, not a prevalence study. A pass means BuildOpt can
detect, propose, validate, and economically qualify this correction class on at
least two public families. A fail is still useful: it will identify whether the
limiting gate is diagnostic binding, materiality, semantic correction, exact
outputs, or repeated-build value.

The authoritative contracts are the [human specification](../../specs/poc-source-bound-configuration-input-corrections-v1.md),
[machine contract](../../specs/poc-source-bound-configuration-input-corrections-v1.json),
and [subject manifest](../../specs/poc-source-bound-configuration-input-corrections-v1.subjects.json).

## Ordered execution

### SBIC-000 — contract and exact source freeze (`DONE`)

- Freeze Suwayomi Server, QuickCarpet, and LSSS at exact revisions.
- Bind archive, wrapper, workflow, source, call-site, spans, JDK, arguments, and
  required outputs before the first SBIC Gradle start.
- Record BlueMap as a prospective typed exclusion because its version path can
  mutate Git index metadata.
- Start all public diagnostics, mutations, candidates, timings, claims, and
  product failures at zero.

Proof: the independent contract checker reconstructs all structural invariants;
an exact checkout mode additionally recomputes archives and every frozen file
digest.

### SBIC-001 — source detector (`DONE`)

- Implement a versioned, name-invariant source scanner for ProcessBuilder,
  Runtime.exec, Groovy execute, supported Gradle providers, task-action
  deferral, side effects, secrets/interaction, ambiguity, and source drift.
- Emit source path/SHA, declaration and operation spans, call sites, explicit
  behavior facts, recipe, typed decision, and reason.
- Add negatives for side effects, task-only execution, supported providers,
  ambiguity, drift, secret/interactive use, and forbidden name rules.
- Reconstruct all three selected source rows and the BlueMap exclusion without
  trusting manifest decisions.

Gate: 3/3 selected families must reconstruct as pure configuration reads and
the exclusion must remain side-effecting. Otherwise stop before a public build.

Observed: the versioned scanner and independent CLI reconstruct five pure rows
across all three selected families and one side-effect row for BlueMap. Fixture
tests cover tracked providers, task-only execution, secrets/interaction,
ambiguity, source drift, multiple operations, Groovy syntax, false-positive
redirect syntax, and label invariance. No public Gradle start occurred.

### SBIC-002 — fresh strict diagnostics (`DONE_STOP`)

- Materialize clean Git-preserving checkouts and empty Gradle homes.
- Run one exact owner workflow per family with `--configuration-cache`, strict
  problems, no scan, four workers, and the frozen toolchain/environment.
- Use only the SDCR versioned runner's child-log-owned report selection and
  atomic evidence publication.
- Reparse each selected raw report and bind problems to the SBIC-001 source row.

Gate: 3/3 conclusive, zero lost starts, and at least 2/3 diagnostic-bound
families. Cold incompatibility is conclusive but not eligible.

Preparation: the versioned SBIC adapter maps only the frozen nested subject
fields into the already-proven SDCR runner contract. A fixture reconstructs the
exact mapping and proves that the canonical manifest cannot override the SDCR
runner. No public Gradle start was used for this preparation.

First observation: Suwayomi completed 24 tasks before strict Configuration
Cache failure in 3m19s. Its log reports five occurrences of one unique external
process problem and one report file, but `--stacktrace` repeats the identical
report URI. The original v1 selector outcome is retained as
`ROOT_REPORT_REFERENCE_AMBIGUOUS`, so this start is conclusive but not
source-bound. The versioned v2 selector deduplicates only identical URI text;
two distinct paths remain ambiguous, and inventory is still never authority.
The frozen one-start budget is not silently extended or replayed.

Terminal result: QuickCarpet completed 29 tasks and retained one report with 26
problems, all unrelated Fabric Loom incompatibilities; Gradle 7.4 did not report
its three frozen `Runtime.exec` source rows. LSSS completed 67 tasks and retained
one report with two problems; its `git rev-parse HEAD` problem binds exactly to
the single frozen source row. Independent reconstruction therefore records 3/3
conclusive starts, zero lost starts, two retained reports, 28 total report
problems, one external-process problem, and only 1/3 diagnostic-bound families.
The required 2/3 gate fails.

### SBIC-003 — controlled materiality (`NOT_AUTHORIZED`)

- Prefetch dependencies outside measurement and preserve raw logs.
- Apply the existing owner-host controls, wait 120 seconds, and pass seven
  stability samples at ratio <= 1.15.
- For each diagnostic-bound family, use at most two starts to measure the
  repeated-workflow opportunity attributable to configuration reuse.

Gate: at least two families independently exceed 500 ms and 2%. Hosted CI has
no wall-time authority. If the host fails stability, stop as
`INCOMPLETE_PERFORMANCE_ENVIRONMENT`; do not relax the threshold.

### SBIC-004 — isolated candidate correctness (`NOT_AUTHORIZED`)

- Produce at most one digest-bound PatchBundle per material family.
- Apply it only in an isolated copy and preserve command, working directory,
  environment, charset, streams, exit/fallback behavior, and all consumers.
- Verify exact required outputs, strict zero-new-problem behavior,
  Configuration Cache store/reuse, relevant input invalidation, equivalent
  errors, exact inverse/revert, and zero product failures.

Gate: at least two qualified families. A public checkout or upstream repository
is never mutated.

### SBIC-005 — paired value (`NOT_AUTHORIZED`)

- Use one excluded stabilization and eight balanced native/candidate pairs per
  qualified family after prefetch and quiescence.
- Reconstruct every duration from raw monotonic timestamps.
- Require 8/8 positive pairs, mean saving >= 500 ms and >= 2%, positive paired
  95% interval, non-regressive candidate p95, and combined payback <= 300
  builds.

### SBIC-006 — terminal product decision (`DONE`)

- Independently reconstruct every gate and update the evidence index, detailed
  tracker, validation reference, implementation tracker/ledger, audit,
  performance findings, and one-page handoff.
- If at least two families qualify, publish review-only PatchBundles and a
  bounded mechanism conclusion. Otherwise stop with the exact failed gate.
- Never authorize automatic application, merge, upstream PRs, production, Test
  Optimization, or arbitrary-repository prevalence.

Decision: stop this exact route at the diagnostic-binding gate. No controlled
materiality run, public-source mutation, candidate, timing sample, PatchBundle,
speedup claim, or product failure was created. Source selection improved spend
targeting, but source evidence alone cannot guarantee that the Gradle version
will observe the operation or that unrelated plugin incompatibilities will not
dominate the strict report.

## Budgets and stop rules

- 20 GiB maximum additional disk; at least 10 GiB before a public start; stop
  before the next start below 8 GiB.
- 1,200 seconds per start, four workers, one start at a time.
- Three strict starts, at most six materiality starts, at most six correctness
  starts per candidate, and eight balanced pairs per qualified family.
- Any evidence-publication failure stops before another start.
- Temporary checkouts and homes remain until retained evidence is independently
  reconstructed, then are removed by exact path.

## Changelog

- 2026-09-05: closed SBIC at 1/3 diagnostic-bound families after three
  conclusive starts; dependent materiality, candidate, and value blocks remain
  unexecuted and unauthorized.
