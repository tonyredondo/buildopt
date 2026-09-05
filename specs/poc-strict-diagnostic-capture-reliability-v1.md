# Strict Diagnostic Capture Reliability v1

`STRICT_DIAGNOSTIC_CAPTURE_RELIABILITY_V1` (`SDCR`) is an enabling experiment
for BuildOpt's reviewed-native correction path. CINC stopped before candidate
work because its diagnostic harness could not complete its cohort within the
frozen start budget. SDCR tests the harness failures themselves; it does not
reopen CINC, classify correction opportunity, or establish performance value.

The product seam remains the ordinary Gradle wrapper. A diagnostic attempt must
leave the public source untouched, run in a disposable Git checkout, retain a
typed result even when Gradle cannot produce a report, and select only the
Configuration Cache report owned by the root invocation.

## Frozen reliability invariants

1. Source is materialized as a Git checkout with a checked `.git` directory.
   The runner verifies clean status, exact `HEAD`, and the frozen `git archive`
   SHA-256 before Gradle starts.
2. Compatibility and diagnostic execution use the same source materialization,
   toolchain, wrapper, working directory, owner arguments, owner environment,
   empty Gradle home, worker limit, scan policy, and timeout. A warm admission
   run may not qualify a cold diagnostic.
3. Owner project-directory arguments are authoritative. The runner adds `-p`
   only when the owner arguments contain no `-p`, `--project-dir`, or
   `--project-dir=...` form.
4. Report ownership comes from the unique report URI emitted by the root Gradle
   process. The canonical target must be a regular, non-symlinked file below the
   expected root `build/reports/configuration-cache` directory. Nested build and
   TestKit reports are inventory only and cannot create ambiguity.
5. Every started Gradle child atomically publishes its log, source/Git facts,
   report inventory, selected report when present, exit code, and exactly one
   typed outcome. Failure before Gradle starts also publishes a harness result
   without incrementing the public-start count.

The allowed outcomes are `ROOT_REPORT_CAPTURED`,
`NO_CONFIGURATION_CACHE_REPORT`, `ROOT_REPORT_REFERENCE_MISSING`,
`ROOT_REPORT_REFERENCE_AMBIGUOUS`, `ROOT_REPORT_OUTSIDE_EXPECTED_ROOT`,
`SOURCE_GIT_METADATA_INVALID`, `SOURCE_REVISION_DRIFT`,
`SOURCE_ARCHIVE_DRIFT`, `OWNER_WORKFLOW_COLD_INCOMPATIBLE`, `TIME_LIMIT`, and
`HARNESS_FAILURE`. Names of repositories, tasks, plugins, or files may label a
row but may never select an outcome.

## Blocks and gates

| Block | Work | Gate |
|---|---|---|
| `SDCR-000` | Freeze this human/machine contract, three exact public probes, budgets, tracker, and independent checker | Planning only; zero public starts |
| `SDCR-001` | Implement capture runner v2 plus independent fixtures for Git metadata, cold-state identity, project-dir deduplication, root URI ownership, ambiguity, traversal, symlink, and atomic failure evidence | All fixture negatives fail closed; zero public starts |
| `SDCR-002` | Run one fresh probe for OpenAPI Generator, Licensee, and Gradle Profiler from separate Git checkouts and empty Gradle homes | Three artifact-backed starts or a typed pre-start harness stop; no report borrowed from CINC |
| `SDCR-003` | Independently reconstruct every binding, raw log/report hash, root ownership decision, and family count | 3/3 conclusive harness outcomes and zero silently lost starts |
| `SDCR-004` | Decide whether a fresh configuration-input opportunity successor may freeze a new cohort | Opens only after `SDCR-003`; it cannot revive CINC rows or budgets |

OpenAPI Generator tests cold-state compatibility, Licensee tests unique root
report ownership in the presence of nested reports, and Gradle Profiler tests a
previously unstarted workflow. These labels explain probe selection only; the
runner contains no family-specific branch.

## Budgets and stopping point

- at most three public Gradle starts, one per frozen probe;
- 1,200 seconds per start, sequential execution, four Gradle workers;
- at most 12 GiB additional disk, at least 10 GiB free before a start, and stop
  before the next start below 8 GiB;
- local functional evidence only; durations have no materiality or value
  authority;
- no public source mutation, candidate build, paired timing, speedup claim,
  automatic patch, pull request, or productization;
- predecessor reports and summaries may motivate fixtures but cannot supply an
  SDCR outcome, count, hash, or gate.

Passing SDCR means only that BuildOpt can retain and reconstruct these three
diagnostic failure modes. It authorizes planning a distinct fresh opportunity
experiment; it does not authorize CINC-004, a source patch, or timing.
