# Strict Diagnostic Capture Reliability POC

**Overall:** `CAPTURE_RELIABILITY_PROVEN`<br>
**Current block:** `SDCR-004` is complete<br>
**Stop point:** reached; a distinct opportunity experiment may now be planned,
but CINC rows, candidates, and budgets remain closed.

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

### SDCR-001 — runner and fixture proof (`DONE`)

- Add a versioned capture runner rather than modifying CINC's immutable v1
  runner.
- Split pure report-reference parsing and containment checks from process
  execution so hostile paths, duplicate references, missing references,
  symlinks, nested inventories, and name-invariance can be tested cheaply.
- Prove project-directory deduplication for all three Gradle spellings.
- Prove every post-start exit publishes a complete temporary bundle and then
  renames it atomically; incomplete temporary data never becomes evidence.
- Record cold-state identity in the capture rather than trusting a summary.

Proof: the versioned runner, pure Go selector, CLI, and independent checker
reconstruct sixteen positive/negative cases, including atomic evidence after a
forced post-start selector failure. No public Gradle start occurred.

### SDCR-002 — fresh public probes (`DONE`)

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

Observed: all three starts were retained. OpenAPI Generator produced the cold
Gradle 8.1.1/JDK 21 incompatibility with no report; Licensee completed while
inventorying 237 unreferenced reports; Gradle Profiler completed while
inventorying one unreferenced report. The latter two are
`NO_CONFIGURATION_CACHE_REPORT`; the runner selected no arbitrary file.

### SDCR-003 — independent reconstruction (`DONE`)

- Rehash the contract, subjects, runner, each raw log, and selected report.
- Recompute the source archive from a supplied exact checkout or a retained Git
  bundle; never trust the capture's archive field alone.
- Reparse root report references from raw logs and independently enforce
  canonical containment, regular-file, and no-symlink rules.
- Recompute starts, conclusive subjects, silently lost starts, report inventory
  counts, and outcome counts.

Pass requires 3/3 conclusive subjects and zero silently lost starts. A terminal
failure remains useful evidence and must name the exact failed invariant.

Observed: the checker reconstructs 3/3 conclusive starts, 238 inventory files,
zero child-log report references, zero selected reports, and zero lost starts.
With the supplied exact checkouts it also recomputes all three archive and
wrapper hashes.

### SDCR-004 — successor decision (`DONE`)

If SDCR passes, freeze a new cohort and a new start budget for a distinct
configuration-input opportunity experiment. It may reuse the versioned runner,
but none of the CINC or SDCR diagnostic rows may count as opportunity evidence.
If SDCR fails, repair only the failed harness invariant under a versioned
contract; do not loosen the gate or begin candidates/timing.

Decision: `AUTHORIZE_FRESH_CONFIGURATION_INPUT_SUCCESSOR_PLANNING`. The next
contract must use this runner, count cold incompatibility as an admissibility
decision rather than a lost source-gate start, and keep opportunity evidence at
zero until its own cohort is frozen.

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
