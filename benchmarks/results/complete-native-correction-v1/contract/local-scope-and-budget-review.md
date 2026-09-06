# CNC-002: accepted local scope and proof-budget review

Date: 2026-09-05. Historical preparation status: `PARTIAL_BUDGET_DECISION_REQUIRED`.
The owner subsequently approved the two-hour limit and the 60-start increase.
The [CNC-002 contract](../../../../specs/poc-complete-native-correction-v1.md)
supersedes this preliminary matrix and is checked statically; it is not an
executable runbook. No CNC Gradle start, public patch, candidate or timing row exists.

## Accepted scope

The owner accepted the recommendation following CNC-001: first study the
explicitly local non-CI `assemble` command on GraphQL Java revision
`f2d8c9126f898c084b176631b7346bc6fbec296a`. The earlier
[source-feasibility refusal](../selection/feasibility.md) remains historical;
its local-versus-CI decision is now resolved, not silently ignored.

The intended input contract is:

- Root working directory, unchanged Gradle 9.6.1 Wrapper and owner source.
- Local Git checkout; initially detached at the exact revision. Preserve the
  existing HEAD-based snapshot version. Branch behavior belongs in fixtures;
  do not change shared subject refs to manufacture a version during a pair.
- `CI` absent and `RELEASE_VERSION` absent in both public arms. Reject a host
  invocation presenting CI/release mode rather than quietly rewriting it.
- Native `assemble` is the value command. The owner's subsequent CI `check`
  command, other matrix jobs and CI qualification are not claimed.
- Correctness may add the owner's `verifyShadedClassAnnotations` and
  `compileShadedJarConsumer` as explicit, identically applied auxiliary proof
  tasks. These additional arguments never enter the value comparison.
- Both arms use the same pinned JDKs and equivalent native cache opportunities.
  No dependency, plugin, Wrapper, version policy or required output is changed.
- Preserve CI/release/no-Git behavior outside the measured local mode. A local
  measurement boundary does not authorize a source patch that breaks those
  paths. Fixtures must test them without presenting them as public CI evidence.

The existing source identifies the main final-JAR pipeline, the jcstress JAR,
and the included `performance-results-page` Java subproject. The output
contract must include root `build/libs` and the subproject's `build/libs`,
bind artifacts to their producers, and reject missing or unowned outputs.
The old two-file root inventory is not a complete new three-producer freeze.
Actual native inventories, exact hashes, intermediate-artifact preservation,
and generated producer paths still require fresh capture before a candidate.
This static note does not claim the final inventory was observed.

## Selected runtime metadata

The intended Linux x64 runtime packages are selected from the official release
metadata, read during CNC-002. These are package requirements, not installed
or runtime-verified binaries:

| Role | Version | Archive SHA-256 | Official release |
| --- | --- | --- | --- |
| Gradle daemon and root Java 25 compilation | Corretto `25.0.4.8.1` | `b838e42c8e915019ed34e4cc54c7cda2e7e00d2a2a49be44578814735fc9accc` | [Corretto 25 release](https://github.com/corretto/corretto-25/releases/tag/25.0.4.8.1) |
| Included Java 21 subproject compilation | Corretto `21.0.12.9.1` | `f79824540cef882da0cdf1369f9d1d69afc14b5a9bc3a771fd5bb795793ce2f2` | [Corretto 21 release](https://github.com/corretto/corretto-21/releases/tag/21.0.12.9.1) |

The exact archive filenames are
`amazon-corretto-25.0.4.8.1-linux-x64.tar.gz` and
`amazon-corretto-21.0.12.9.1-linux-x64.tar.gz`, under the corresponding
versioned `https://corretto.aws/downloads/resources/` locations. No `latest`
download URL may enter the frozen manifest.

The repository's current `dev/run` supports Temurin JDK identifiers, not
Corretto. Do not supply a made-up `--toolchain` value or relabel a Temurin
installation as Corretto. The eventual runner must validate the actual
archive and runtime identity and supply the two explicit toolchain paths with
automatic discovery/download disabled. Implement and test that owning path
before claiming runtime readiness; no global toolchain change is made here.

## Why the original allocation cannot be frozen

The proposed Phase A ceiling was 40 real Gradle starts, including only six
real-Gradle fixture starts and eight public-correctness starts. A minimal
paired lifecycle matrix alone uses nine fixture starts:

| Required behavior | Native starts | Candidate starts |
| --- | ---: | ---: |
| Successful work and both completion summaries | 1 | 1 store plus 1 reuse |
| Task failure and failure summary | 1 | 1 |
| Configuration failure after callback registration | 1 | 1 |
| Cancellation after a deterministic start handshake | 1 | 1 |
| **Total** | **4** | **5** |

These are nine separately started builds, not nine assertions inside one
successful build. Pure parser/Go tests cannot prove actual Gradle lifecycle
callbacks. This lower bound exceeds the six reserved fixture starts before
process-error, changing-input, or complete Bnd behavior cases are allocated.
It does not prove that every conceivable 40-start redesign is impossible;
it proves that the agreed allocation is incomplete and must not be frozen
unchanged. No required proof is dropped to make the arithmetic pass.

## Approved time limit and subsequent start-count decision

The owner approved a maximum of two elapsed execution hours, replacing the
earlier eight-hour proposal, with a progress review at 30 minutes. The
[Phase A time rule](../../../../docs/plans/complete-native-correction-poc.md#phase-a-directed-native-mechanism-cnc-001-through-cnc-008)
includes preparation, downloads and validation, forbids clock resets, and
requires an incomplete result if required proof cannot finish in time. The
checkpoint is not an extension or a reason to weaken acceptance criteria.
The initial time-only approval did not grant the 60-start increase. The owner's
subsequent approval resolves that remaining decision. Neither approval starts
Gradle in CNC-002. No execution window has started.

## Replacement allocation: 60 starts, subsequently approved

This table began as a bounded proposal and is now owner-approved:

| Category | Proposed starts |
| --- | ---: |
| Real-Gradle behavioral fixtures | 24 |
| Symmetric native preparation/prefetch | 2 |
| Fresh strict diagnostics | 2 |
| Controlled native materiality | 2 |
| Public correctness and owner proof | 10 |
| Native value: two stabilizations plus eight pairs | 18 |
| Classified infrastructure replacement reserve | 2 |
| **Total** | **60** |

Apply the approved two-hour ceiling and 30-minute review. Retain 20 GiB concurrent
additional disk, at least 10 GiB free before each start, four workers, one
start at a time, and per-start timeout of at most 20 minutes bounded by the
remaining phase window. The 60-start ceiling is not an obligation to spend
every slot and does not extend time/disk limits. Failure of a prerequisite
still stops dependent execution. Preserve the two-infrastructure-replacement
maximum; a product or value failure cannot consume it as a retry.

### Fixture allocation design

The following sequence assigns all 24 starts. CNC-002's final machine contract
must turn it into explicit command/input/output assertions; CNC-003 must prove
the implementation. The same compatible fixture includes the actual Bnd
extension, representative version logic, multiple test tasks, and both
completion-report responsibilities.

| Slots | Behavior |
| --- | --- |
| F01-F03 | Native successful reference; candidate store; candidate reuse. Compare Bnd artifacts, process-result matrix, test totals/failures/skips, slow-test cutoff/ranking and summary ordering. |
| F04-F05 | Native/candidate branch-output change, with identical request and an independently changed tracked process result. Require correct configuration invalidation, not a different task list that hides it. |
| F06-F07 | Native/candidate commit change with unchanged local branch name. Distinguish source/task invalidation from Git output that the original local version computation does not consume. |
| F08-F09 | Native/candidate release override; preserve the caller's explicit version and avoid unnecessary Git reads. |
| F10-F11 | Native/candidate CI version branch; preserve timestamp/short-hash semantics. Assert each timestamp against its own execution interval, not byte equality between naturally different times or a frozen public clock. |
| F12-F13 | Native/candidate no-Git fallback and timestamp semantics; verify relative streams, trimming and original fallback. |
| F14-F15 | Native/candidate missing executable and launch failure. No silent successful fallback. |
| F16-F17 | Native/candidate nonzero, empty and malformed process-output matrix. Multiple assertions may share these starts only if each real code path and original exception/fallback is observed independently; otherwise stop before freezing. |
| F18-F19 | Native/candidate task failure with equivalent summaries and exit behavior. |
| F20-F21 | Native/candidate configuration failure after callback registration, including exactly-once final reporting. |
| F22-F23 | Native/candidate cancellation after a deterministic child-start handshake; verify bounded exit and finalization, without asserting behavior on an uncatchable kill. |
| F24 | Successful candidate after cancellation; fresh per-build service state and no stale test counters/failures. |

Each cache-invalidation proof must retain the relevant earlier cache entry and
change one consumed input at a time. Changing fixture scenario, task arguments
or build script at the same time cannot prove that input's cache tracking.
The final protocol must specify isolated cache namespaces and scenario setup
before starts, not restore unverified mutable state to obtain a desired hit.

### Public correctness allocation design

| Slots | Behavior |
| --- | --- |
| C01-C02 | Two successful optimized-native controls with complete exact inventories and identically declared owner artifact checks; stop on native/native instability. |
| C03-C04 | Candidate store/reuse with complete exact outputs and the same owner artifact checks. Force real task execution symmetrically where the correctness assertion requires it; keep these settings out of value samples. |
| C05 | Candidate on a declared unrelated-file change, preserving configuration reuse and required outputs. No simultaneous relevant mutation may mask this assertion. |
| C06-C07 | Native/candidate on the same declared relevant source mutation; correct invalidation and byte-exact equality of the mutated outputs. |
| C08-C09 | Native/candidate on the same intentional compile-failure mutation; equivalent failure and reporting. These are intentional negative probes, not successful timing rows. |
| C10 | Candidate-root exact source/probe revert followed by native execution and original output comparison; source drift and tampering are also checked statically. |

Public branch/ref mutation is unnecessary for this detached-revision study;
the isolated fixture rows own branch/HEAD behavior. No relocation/cache-portability
claim is made. Claiming it later requires additional cross-root proof before
changing the budget or acceptance conditions.

## Subsequent CNC-002 closure

The human/machine contract, exact subject manifest and independent static
checker/negatives now exist. The final contract specifies command profiles,
state transitions, output selectors and the source/compile-failure probes.
In particular, it distinguishes task-input invalidation from Configuration
Cache invalidation; public correctness does not force every task with a flag.
The preliminary rows above are design history, not the executable protocol.
The final contract preserves the existing statistical
definition used by `dev/check-economics-gated-native-patch-v2-value`: eight
pairs, 4,096 deterministic bootstrap resamples and the same percentile-index
definition. Reuse the algorithm, never the historical result rows.

Register only actually implemented artifacts and commands in indexes/layout/CI.
The implemented `contract` checker proves only the static freeze. Corretto
launch support, fixture runner, output capture and candidate correction still
require CNC-003 and later proof. No real fixture or public build is authorized
by the static preparation itself.
