# Strict Diagnostic Capture Reliability POC

**Overall:** `ACTIVE`<br>
**Current block:** `SDCR-001`<br>
**Stop point:** a checked terminal `SDCR-003` reliability decision; `SDCR-004`
may authorize only the planning of a distinct opportunity experiment.

## Objective

Make BuildOpt's wrapper-coordinated native-correction diagnostics reliable
enough that cold incompatibility, missing Git metadata, and nested Gradle builds
produce reconstructable typed evidence instead of consuming an experiment
budget silently. The authoritative contracts are
[`poc-strict-diagnostic-capture-reliability-v1.md`](../../specs/poc-strict-diagnostic-capture-reliability-v1.md)
and its [machine form](../../specs/poc-strict-diagnostic-capture-reliability-v1.json).

## Execution route

### SDCR-000 — contract freeze (`DONE`)

- Freeze three exact public revisions and one start per subject.
- Freeze Git-preserving materialization, empty per-probe Gradle homes, exact
  toolchain/workflow bindings, log-owned report selection, atomic evidence, and
  typed failure outcomes.
- Keep predecessor artifacts as motivation only and all performance/candidate
  counters at zero.

Proof: the independent contract checker reconstructs the machine contract and
subject bindings from the checked files.

### SDCR-001 — runner and fixture proof (`TODO`)

- Add a versioned capture runner rather than modifying CINC's immutable v1
  runner.
- Split pure report-reference parsing and containment checks from process
  execution so hostile paths, duplicate references, missing references,
  symlinks, nested inventories, and name-invariance can be tested cheaply.
- Prove project-directory deduplication for all three Gradle spellings.
- Prove every post-start exit publishes a complete temporary bundle and then
  renames it atomically; incomplete temporary data never becomes evidence.
- Record cold-state identity in the capture rather than trusting a summary.

Proof: focused unit/fixture checker, ShellCheck, `git diff --check`, and Base CI
static integration. No public Gradle start is allowed in this block.

### SDCR-002 — fresh public probes (`WAITING`)

- Clone each frozen repository at its exact revision. Do not unpack source-only
  archives for execution.
- Run OpenAPI Generator, Licensee, then Gradle Profiler sequentially with a new
  source clone and empty Gradle home for each.
- Retain the raw child log for every start. Retain the complete report inventory
  and only the root-log-selected report when one exists.
- Stop before the next start on timeout, disk reserve, source drift, or evidence
  publication failure. Never replace a subject after observing its outcome.

Expected outcomes are deliberately not frozen: the evidence may show success,
an owner incompatibility, no report, or a typed harness failure. Conclusiveness,
not a favorable Gradle result, is the gate.

### SDCR-003 — independent reconstruction (`WAITING`)

- Rehash the contract, subjects, runner, each raw log, and selected report.
- Recompute the source archive from a supplied exact checkout or a retained Git
  bundle; never trust the capture's archive field alone.
- Reparse root report references from raw logs and independently enforce
  canonical containment, regular-file, and no-symlink rules.
- Recompute starts, conclusive subjects, silently lost starts, report inventory
  counts, and outcome counts.

Pass requires 3/3 conclusive subjects and zero silently lost starts. A terminal
failure remains useful evidence and must name the exact failed invariant.

### SDCR-004 — successor decision (`WAITING`)

If SDCR passes, freeze a new cohort and a new start budget for a distinct
configuration-input opportunity experiment. It may reuse the versioned runner,
but none of the CINC or SDCR diagnostic rows may count as opportunity evidence.
If SDCR fails, repair only the failed harness invariant under a versioned
contract; do not loosen the gate or begin candidates/timing.

## Resource controls

Before the three public probes, record available disk and cap the campaign at
12 GiB. Starts are sequential, limited to 1,200 seconds and four workers. The
runner uses normal host power settings because this is functional evidence;
hosted CI may validate fixtures and contracts but never wall time. Temporary
checkouts and Gradle homes are deleted only after their retained evidence has
passed reconstruction.

## Completion boundary

SDCR can prove diagnostic reliability only. It cannot claim that a correction
exists, that a correction is material, that BuildOpt beats native Gradle, or
that automatic mutation/merge is safe. Those remain gates of a later fresh
experiment.
